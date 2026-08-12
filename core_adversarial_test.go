package mintegral

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
)

func TestCredentials_FormattingAndJSON_redactSecrets(t *testing.T) {
	// Given
	credentials := mustCredentials(t, "access-secret", "api-secret")

	// When
	plain := credentials.String()
	goSyntax := fmt.Sprintf("%#v", credentials)
	encoded, err := json.Marshal(credentials)
	// Then
	if err != nil {
		t.Fatalf("json.Marshal(Credentials) error = %v", err)
	}
	for _, output := range []string{plain, goSyntax, string(encoded)} {
		if strings.Contains(output, "access-secret") || strings.Contains(output, "api-secret") {
			t.Fatalf("credential output leaked secret: %q", output)
		}
	}
	var redacted string
	if err := json.Unmarshal(encoded, &redacted); err != nil {
		t.Fatalf("json.Unmarshal(redacted credentials) error = %v", err)
	}
	if redacted != "<redacted>" {
		t.Fatalf("decoded credential JSON = %q, want redacted value", redacted)
	}
}

func TestApplyAuthentication_matchesFixedProtocolVector(t *testing.T) {
	// Given
	credentials := mustCredentials(t, "access-key", "api-key")
	header := make(http.Header)

	// When
	applyAuthentication(header, credentials, 1_471_256_697)

	// Then
	if got := header.Get("access-key"); got != "access-key" {
		t.Fatalf("access-key = %q, want fixed vector access key", got)
	}
	if got := header.Get("timestamp"); got != "1471256697" {
		t.Fatalf("timestamp = %q, want fixed vector timestamp", got)
	}
	if got := header.Get("token"); got != "cc1dc54a234cd82f8c05951323a668a3" {
		t.Fatalf("token = %q, want fixed protocol vector", got)
	}
}

func TestClient_executeDoesNotMarshalBody_whenCredentialsAreInvalid(t *testing.T) {
	// Given
	var calls atomic.Int64
	client, err := NewClient(
		WithDefaultCredentials(mustCredentials(t, "default-access", "default-api")),
		WithHTTPClient(&http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			calls.Add(1)
			return nil, errors.New("transport must not be called")
		})}),
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	// When
	_, err = doJSON[struct{}](context.Background(), client, requestSpec{
		operation:     "test.lazy_json",
		method:        http.MethodPost,
		path:          "/test",
		body:          jsonBody(make(chan int)),
		authenticated: true,
	}, []RequestOption{WithRequestCredentials(Credentials{})})

	// Then
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("doJSON() error = %v, want ErrInvalidCredentials", err)
	}
	if got := calls.Load(); got != 0 {
		t.Fatalf("transport calls = %d, want 0", got)
	}
}

func TestClient_executeReturnsJSONEncodingError_afterCredentialsResolve(t *testing.T) {
	// Given
	var calls atomic.Int64
	client, err := NewClient(
		WithDefaultCredentials(mustCredentials(t, "access", "api")),
		WithHTTPClient(&http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			calls.Add(1)
			return nil, errors.New("transport must not be called")
		})}),
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	// When
	_, err = doJSON[struct{}](context.Background(), client, requestSpec{
		operation:     "test.lazy_json_error",
		method:        http.MethodPost,
		path:          "/test",
		body:          jsonBody(make(chan int)),
		authenticated: true,
		outcomeRisk:   true,
	}, nil)

	// Then
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("doJSON() error = %v, want ErrInvalidRequest", err)
	}
	if errors.Is(err, ErrTransport) || errors.Is(err, ErrOutcomeUnknown) {
		t.Fatalf("doJSON() error = %v, must not be transport or outcome-unknown before send", err)
	}
	if got := calls.Load(); got != 0 {
		t.Fatalf("transport calls = %d, want 0", got)
	}
}
