package mintegral

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestRetryStatus_allowsOnlyTransientStatuses(t *testing.T) {
	// Given
	transient := []int{http.StatusRequestTimeout, http.StatusTooManyRequests, http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout}
	permanent := []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound, http.StatusConflict, http.StatusTeapot}

	for _, status := range transient {
		if !shouldRetryStatus(status) {
			t.Errorf("shouldRetryStatus(%d) = false, want true", status)
		}
	}
	for _, status := range permanent {
		if shouldRetryStatus(status) {
			t.Errorf("shouldRetryStatus(%d) = true, want false", status)
		}
	}
}

func TestClient_executeRejectsNilResponseWithoutError(t *testing.T) {
	// Given
	client, err := NewClient(WithHTTPClient(&http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, nil
	})}))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	// When
	response, executeErr := client.execute(context.Background(), requestSpec{
		operation: "test.nil_response",
		method:    http.MethodGet,
		path:      "/test",
	}, nil)

	// Then
	if response != nil {
		if response.Body != nil {
			if closeErr := response.Body.Close(); closeErr != nil {
				t.Errorf("close unexpected response body: %v", closeErr)
			}
		}
		t.Fatalf("execute() response = %v, want nil", response)
	}
	if !errors.Is(executeErr, ErrTransport) {
		t.Fatalf("execute() error = %v, want ErrTransport", executeErr)
	}
}

func TestClient_buildRequestURLRejectsUnknownOrUnavailableTarget(t *testing.T) {
	// Given
	client, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	// When
	_, unknownErr := client.buildRequestURL(requestSpec{target: baseTarget(99)})
	client.apiBaseURL = nil
	_, unavailableErr := client.buildRequestURL(requestSpec{target: apiTarget})

	// Then
	if !errors.Is(unknownErr, ErrInvalidRequest) {
		t.Fatalf("unknown target error = %v, want ErrInvalidRequest", unknownErr)
	}
	if !errors.Is(unavailableErr, ErrInvalidRequest) {
		t.Fatalf("unavailable target error = %v, want ErrInvalidRequest", unavailableErr)
	}
}

func TestClient_executeDoesNotRetryPermanentNetworkErrors(t *testing.T) {
	// Given
	cases := []struct {
		name      string
		err       error
		preserved bool
	}{
		{name: "context cancellation", err: context.Canceled, preserved: true},
		{name: "context deadline", err: context.DeadlineExceeded, preserved: true},
		{name: "DNS", err: &net.DNSError{Name: "api.example.test", IsTemporary: false}},
		{name: "URL", err: &url.Error{Op: "Get", URL: "https://api.example.test", Err: errors.New("unsupported protocol")}},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			var calls atomic.Int64
			client, err := NewClient(
				WithRetryPolicy(RetryPolicy{MaxAttempts: 3, InitialBackoff: 0, MaxBackoff: 0}),
				WithHTTPClient(&http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
					calls.Add(1)
					return nil, testCase.err
				})}),
			)
			if err != nil {
				t.Fatalf("NewClient() error = %v", err)
			}

			// When
			response, executeErr := client.execute(context.Background(), requestSpec{
				operation: "test.permanent_network",
				method:    http.MethodGet,
				path:      "/test",
				retryable: true,
			}, nil)
			if response != nil {
				closeTestResponse(t, response)
			}

			// Then
			if testCase.preserved && !errors.Is(executeErr, testCase.err) {
				t.Fatalf("execute() error = %v, want preserved %v", executeErr, testCase.err)
			}
			if !testCase.preserved && !errors.Is(executeErr, ErrTransport) {
				t.Fatalf("execute() error = %v, want ErrTransport", executeErr)
			}
			if got := calls.Load(); got != 1 {
				t.Fatalf("transport calls = %d, want 1", got)
			}
		})
	}
}

func TestTransportError_redactsURLAndPreservesOnlyContextCauses(t *testing.T) {
	// Given
	const secret = "signed-query-secret"
	transportErr := newTransportError(requestSpec{operation: "test.redact_transport", outcomeRisk: true}, &url.Error{
		Op:  "Get",
		URL: "https://storage.example.test/file?token=" + secret,
		Err: errors.New("connection reset"),
	})
	canceledErr := newTransportError(requestSpec{operation: "test.cancel_transport"}, context.Canceled)

	// When
	message := transportErr.Error()
	var urlError *url.Error

	// Then
	if strings.Contains(message, secret) || strings.Contains(message, "https://") {
		t.Fatalf("transport error leaked sensitive URL: %q", message)
	}
	if errors.As(transportErr, &urlError) {
		t.Fatalf("transport error exposes URL error: %v", urlError)
	}
	if !errors.Is(transportErr, ErrTransport) || !errors.Is(transportErr, ErrOutcomeUnknown) {
		t.Fatalf("transport error lost category: %v", transportErr)
	}
	if !errors.Is(canceledErr, context.Canceled) {
		t.Fatalf("canceled transport error = %v, want context.Canceled", canceledErr)
	}
}

func TestRetryError_retriesOnlyKnownTransientNetworkErrors(t *testing.T) {
	// Given
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{name: "EOF", err: io.EOF, want: true},
		{name: "unexpected EOF", err: io.ErrUnexpectedEOF, want: true},
		{name: "network operation", err: &net.OpError{Op: "dial", Err: errors.New("connection refused")}, want: true},
		{name: "permanent URL", err: &url.Error{Op: "Get", URL: "https://api.example.test", Err: errors.New("unsupported protocol")}, want: false},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			// When
			got := shouldRetryError(testCase.err)

			// Then
			if got != testCase.want {
				t.Fatalf("shouldRetryError(%v) = %t, want %t", testCase.err, got, testCase.want)
			}
		})
	}
}

func TestClient_executeRetriesTransientTimeoutWithFreshTimestamp(t *testing.T) {
	// Given
	clock := incrementingClock{value: time.Unix(1_700_000_000, 0)}
	var calls atomic.Int64
	timestamps := make(chan string, 2)
	client, err := NewClient(
		WithDefaultCredentials(mustCredentials(t, "access", "api")),
		WithClock(&clock),
		WithRetryPolicy(RetryPolicy{MaxAttempts: 2, InitialBackoff: 0, MaxBackoff: 0}),
		WithHTTPClient(&http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			timestamps <- request.Header.Get("timestamp")
			if calls.Add(1) == 1 {
				return nil, timeoutNetworkError{}
			}
			return jsonResponse(http.StatusOK, `{"code":200,"data":{}}`), nil
		})}),
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	// When
	_, err = doJSON[struct{}](context.Background(), client, requestSpec{
		operation:     "test.retry_timestamp",
		method:        http.MethodGet,
		path:          "/test",
		authenticated: true,
		retryable:     true,
	}, nil)
	// Then
	if err != nil {
		t.Fatalf("doJSON() error = %v", err)
	}
	if got := calls.Load(); got != 2 {
		t.Fatalf("transport calls = %d, want 2", got)
	}
	first, second := <-timestamps, <-timestamps
	if first == second {
		t.Fatalf("retry timestamps = %q and %q, want fresh timestamp per attempt", first, second)
	}
}

type incrementingClock struct {
	next  atomic.Int64
	value time.Time
}

func (c *incrementingClock) Now() time.Time {
	return c.value.Add(time.Duration(c.next.Add(1)-1) * time.Second)
}

type timeoutNetworkError struct{}

func (timeoutNetworkError) Error() string   { return "network timeout" }
func (timeoutNetworkError) Timeout() bool   { return true }
func (timeoutNetworkError) Temporary() bool { return true }

var _ net.Error = timeoutNetworkError{}
