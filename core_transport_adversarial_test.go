package mintegral

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"sync/atomic"
	"testing"
	"time"
)

func TestClient_executeDrainsOnlyBoundedRetryResponse(t *testing.T) {
	// Given
	body := &countingReadCloser{Reader: bytes.NewReader(make([]byte, maxRetryDrainBytes+1))}
	var calls atomic.Int64
	client, err := NewClient(
		WithRetryPolicy(RetryPolicy{MaxAttempts: 2, InitialBackoff: 0, MaxBackoff: 0}),
		WithHTTPClient(&http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			if calls.Add(1) == 1 {
				return &http.Response{StatusCode: http.StatusServiceUnavailable, Body: body, Header: make(http.Header)}, nil
			}
			return jsonResponse(http.StatusOK, `{"code":200,"data":{}}`), nil
		})}),
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	// When
	response, err := client.execute(context.Background(), requestSpec{
		operation: "test.retry_drain",
		method:    http.MethodGet,
		path:      "/test",
		retryable: true,
	}, nil)
	if response != nil {
		closeTestResponse(t, response)
	}

	// Then
	if err != nil {
		t.Fatalf("execute() error = %v", err)
	}
	if got := body.read.Load(); got != maxRetryDrainBytes {
		t.Fatalf("retry response bytes read = %d, want bounded %d", got, maxRetryDrainBytes)
	}
	if !body.closed.Load() {
		t.Fatal("retry response body was not closed")
	}
}

func TestRetryDelay_prefersRetryAfterUsingInjectedClock(t *testing.T) {
	// Given
	now := time.Date(2026, time.August, 12, 8, 0, 0, 0, time.UTC)
	response := jsonResponse(http.StatusServiceUnavailable, `{"code":503}`)
	response.Header.Set("Retry-After", now.Add(12*time.Second).Format(http.TimeFormat))

	// When
	delay := retryDelay(response, time.Second, now)

	// Then
	closeTestResponse(t, response)
	if delay != 12*time.Second {
		t.Fatalf("retryDelay() = %s, want Retry-After 12s", delay)
	}
}

type countingReadCloser struct {
	io.Reader
	closed atomic.Bool
	read   atomic.Int64
}

func (r *countingReadCloser) Read(data []byte) (int, error) {
	count, err := r.Reader.Read(data)
	r.read.Add(int64(count))
	return count, err
}

func (r *countingReadCloser) Close() error {
	r.closed.Store(true)
	return nil
}
