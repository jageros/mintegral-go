package mintegral

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestAuthenticatedAPIError_redactsDirectCredentialEcho(t *testing.T) {
	credentials := mustCredentials(t, "bare-access-secret", "bare-api-secret")
	tests := []struct {
		name   string
		secret func(*http.Request) string
	}{
		{name: "access key", secret: func(*http.Request) string { return "bare-access-secret" }},
		{name: "API key", secret: func(*http.Request) string { return "bare-api-secret" }},
		{name: "token", secret: func(request *http.Request) string { return request.Header.Get("token") }},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			var echoed string
			client, err := NewClient(
				WithDefaultCredentials(credentials),
				WithHTTPClient(&http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
					echoed = testCase.secret(request)
					body, marshalErr := json.Marshal(map[string]any{"code": 10000, "message": echoed})
					if marshalErr != nil {
						t.Fatal(marshalErr)
					}
					return jsonResponse(http.StatusOK, string(body)), nil
				})}),
			)
			if err != nil {
				t.Fatal(err)
			}

			_, requestErr := client.Accounts().Balance(context.Background())

			var apiError *APIError
			if !errors.As(requestErr, &apiError) {
				t.Fatalf("Balance() error = %v, want APIError", requestErr)
			}
			if echoed == "" {
				t.Fatal("test did not capture a sensitive value")
			}
			if strings.Contains(apiError.Message, echoed) || strings.Contains(apiError.Error(), echoed) {
				t.Fatalf("API error leaked a direct credential echo: %q", echoed)
			}
		})
	}
}

func TestAuthenticatedAPIError_preservesPermissionClassificationAfterRedaction(t *testing.T) {
	credentials := mustCredentials(t, "bare-access-secret", "bare-api-secret")
	client, err := NewClient(
		WithDefaultCredentials(credentials),
		WithHTTPClient(&http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return jsonResponse(http.StatusOK, `{"code":10000,"message":"permission denied: bare-api-secret"}`), nil
		})}),
	)
	if err != nil {
		t.Fatal(err)
	}

	_, requestErr := client.Accounts().Balance(context.Background())

	if !errors.Is(requestErr, ErrPermissionDenied) {
		t.Fatalf("Balance() error = %v, want ErrPermissionDenied", requestErr)
	}
	if strings.Contains(requestErr.Error(), "bare-api-secret") {
		t.Fatalf("Balance() error leaked credentials: %v", requestErr)
	}
}
