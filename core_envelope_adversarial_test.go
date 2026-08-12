package mintegral

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestClient_executeDoesNotRetry_whenContextIsCanceledDuringBackoff(t *testing.T) {
	// Given
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	var calls atomic.Int64
	client, err := NewClient(
		WithRetryPolicy(RetryPolicy{MaxAttempts: 3, InitialBackoff: time.Hour, MaxBackoff: time.Hour}),
		WithHTTPClient(&http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			calls.Add(1)
			cancel()
			response := jsonResponse(http.StatusServiceUnavailable, `{"code":503}`)
			response.Header.Set("Retry-After", "3600")
			return response, nil
		})}),
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	// When
	response, err := client.execute(ctx, requestSpec{
		operation: "test.cancel_retry",
		method:    http.MethodGet,
		path:      "/test",
		retryable: true,
	}, nil)
	if response != nil {
		closeTestResponse(t, response)
	}

	// Then
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("execute() error = %v, want context.Canceled", err)
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("transport calls = %d, want 1", got)
	}
}

func TestDecodeEnvelope_rejectsConflictingMessagesAndOversizedOrMistypedBodies(t *testing.T) {
	// Given
	cases := []struct {
		name string
		body string
	}{
		{name: "conflicting message fields", body: `{"code":200,"msg":"one","message":"two","data":{}}`},
		{name: "oversized body", body: strings.Repeat("x", maxJSONResponseBytes+1)},
		{name: "invalid data type", body: `{"code":200,"data":{"value":"not-a-number"}}`},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			// When
			response := jsonResponse(http.StatusOK, testCase.body)
			defer func() {
				if closeErr := response.Body.Close(); closeErr != nil {
					t.Errorf("response body close error = %v", closeErr)
				}
			}()
			_, err := decodeEnvelope[struct {
				Value int `json:"value"`
			}](response, "test.envelope", false, time.Time{})

			// Then
			if !errors.Is(err, ErrUnexpectedResponse) {
				t.Fatalf("decodeEnvelope() error = %v, want ErrUnexpectedResponse", err)
			}
		})
	}
}

func TestClient_rejectsCrossHostRedirectBeforeCredentialedSecondRequest(t *testing.T) {
	// Given
	var targetCalls atomic.Int64
	target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		targetCalls.Add(1)
	}))
	t.Cleanup(target.Close)
	redirector := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		http.Redirect(writer, request, target.URL, http.StatusFound)
	}))
	t.Cleanup(redirector.Close)
	client, err := NewClient(
		WithDefaultCredentials(mustCredentials(t, "access", "api")),
		WithAPIBaseURL(redirector.URL),
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	// When
	response, err := client.execute(context.Background(), requestSpec{
		operation:     "test.cross_host_redirect",
		method:        http.MethodGet,
		path:          "/redirect",
		authenticated: true,
	}, nil)
	// Then
	if err != nil {
		t.Fatalf("execute() error = %v", err)
	}
	if response == nil {
		t.Fatal("execute() returned a nil response")
	}
	t.Cleanup(func() { closeTestResponse(t, response) })
	if got := response.StatusCode; got != http.StatusFound {
		t.Fatalf("response status = %d, want %d", got, http.StatusFound)
	}
	if got := targetCalls.Load(); got != 0 {
		t.Fatalf("redirect target calls = %d, want 0", got)
	}
}

func TestClient_rejectsHTTPSRedirectDowngrade(t *testing.T) {
	// Given
	client := cloneHTTPClient(&http.Client{})
	initial, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://api.example.test/start", http.NoBody)
	if err != nil {
		t.Fatalf("http.NewRequest() error = %v", err)
	}
	downgrade, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://api.example.test/next", http.NoBody)
	if err != nil {
		t.Fatalf("http.NewRequest() error = %v", err)
	}

	// When
	err = client.CheckRedirect(downgrade, []*http.Request{initial})

	// Then
	if !errors.Is(err, http.ErrUseLastResponse) {
		t.Fatalf("CheckRedirect() error = %v, want http.ErrUseLastResponse", err)
	}
}

func TestAPIError_Error_redactsSensitiveProviderDetails(t *testing.T) {
	// Given
	const secret = "should-not-leak"
	errorValue := &APIError{
		Operation:  "test.redaction",
		HTTPStatus: http.StatusUnauthorized,
		Code:       10000,
		Message:    "https://api.example.test/path?access_key=" + secret + "&api_key=" + secret + "&token=" + secret + "&policy=" + secret + "&signature=" + secret,
	}

	// When
	message := errorValue.Error()

	// Then
	for _, forbidden := range []string{secret, "https://", "access_key", "api_key", "token=", "policy=", "signature="} {
		if strings.Contains(message, forbidden) {
			t.Fatalf("APIError.Error() leaked %q in %q", forbidden, message)
		}
	}
}

func TestDecodeEnvelope_APIErrorDoesNotRetainSensitiveProviderDetails(t *testing.T) {
	// Given
	const secret = "should-not-retain"
	response := jsonResponse(http.StatusUnauthorized, `{"code":10000,"message":"https://api.example.test/path?token=`+secret+`"}`)
	defer func() {
		if closeErr := response.Body.Close(); closeErr != nil {
			t.Errorf("response body close error = %v", closeErr)
		}
	}()

	// When
	_, err := decodeEnvelope[struct{}](response, "test.redacted_api_error", false, time.Time{})

	// Then
	var apiError *APIError
	if !errors.As(err, &apiError) {
		t.Fatalf("decodeEnvelope() error = %v, want APIError", err)
	}
	if strings.Contains(apiError.Message, secret) || strings.Contains(apiError.Message, "https://") {
		t.Fatalf("APIError.Message retained sensitive provider detail: %q", apiError.Message)
	}
}

func TestRetryAfterDuration_acceptsSecondsAndHTTPDate(t *testing.T) {
	// Given
	now := time.Date(2026, time.August, 12, 8, 0, 0, 0, time.UTC)
	cases := []struct {
		name  string
		value string
		want  time.Duration
	}{
		{name: "seconds", value: "15", want: 15 * time.Second},
		{name: "HTTP date", value: now.Add(20 * time.Second).Format(http.TimeFormat), want: 20 * time.Second},
		{name: "invalid", value: "later", want: 0},
		{name: "past date", value: now.Add(-time.Second).Format(http.TimeFormat), want: 0},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			// When
			got := retryAfterDuration(testCase.value, now)

			// Then
			if got != testCase.want {
				t.Fatalf("retryAfterDuration(%q) = %s, want %s", testCase.value, got, testCase.want)
			}
		})
	}
}

func TestDecodeEnvelope_APIErrorStoresRetryAfter(t *testing.T) {
	// Given
	response := jsonResponse(http.StatusTooManyRequests, `{"code":10000,"message":"limited"}`)
	response.Header.Set("Retry-After", "9")

	// When
	_, err := decodeEnvelope[struct{}](response, "test.retry_after", false, time.Time{})

	// Then
	closeTestResponse(t, response)
	var apiError *APIError
	if !errors.As(err, &apiError) {
		t.Fatalf("decodeEnvelope() error = %v, want APIError", err)
	}
	if apiError.RetryAfter != 9*time.Second {
		t.Fatalf("APIError.RetryAfter = %s, want 9s", apiError.RetryAfter)
	}
}

func TestDoJSON_APIErrorUsesInjectedClockForRetryAfterDate(t *testing.T) {
	// Given
	now := time.Date(2026, time.August, 12, 8, 0, 0, 0, time.UTC)
	client, err := NewClient(
		WithClock(fixedClock{value: now}),
		WithHTTPClient(&http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			response := jsonResponse(http.StatusTooManyRequests, `{"code":10000,"message":"limited"}`)
			response.Header.Set("Retry-After", now.Add(11*time.Second).Format(http.TimeFormat))
			return response, nil
		})}),
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	// When
	_, err = doJSON[struct{}](context.Background(), client, requestSpec{operation: "test.fixed_clock", method: http.MethodGet, path: "/test"}, nil)

	// Then
	var apiError *APIError
	if !errors.As(err, &apiError) {
		t.Fatalf("doJSON() error = %v, want APIError", err)
	}
	if apiError.RetryAfter != 11*time.Second {
		t.Fatalf("APIError.RetryAfter = %s, want 11s", apiError.RetryAfter)
	}
}

func closeTestResponse(t *testing.T, response *http.Response) {
	t.Helper()
	if err := response.Body.Close(); err != nil {
		t.Errorf("response body close error = %v", err)
	}
}
