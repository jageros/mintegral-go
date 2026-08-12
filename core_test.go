package mintegral

import (
	"context"
	//nolint:gosec // Mintegral 鉴权协议固定要求 MD5，测试向量必须复现该算法。
	"crypto/md5"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type fixedClock struct {
	value time.Time
}

func (c fixedClock) Now() time.Time { return c.value }

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestNewClientWithoutCredentials(t *testing.T) {
	client, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	if client == nil {
		t.Fatal("NewClient() returned nil client")
	}
}

func TestAuthenticatedRequestWithoutCredentialsDoesNotSend(t *testing.T) {
	var calls atomic.Int64
	client, err := NewClient(WithHTTPClient(&http.Client{Transport: roundTripFunc(
		func(*http.Request) (*http.Response, error) {
			calls.Add(1)
			return nil, nil
		},
	)}))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	_, err = doJSON[struct{}](context.Background(), client, requestSpec{
		operation:     "test.auth",
		method:        http.MethodGet,
		path:          "/test",
		authenticated: true,
	}, nil)
	if !errors.Is(err, ErrCredentialsRequired) {
		t.Fatalf("error = %v, want ErrCredentialsRequired", err)
	}
	if got := calls.Load(); got != 0 {
		t.Fatalf("transport calls = %d, want 0", got)
	}
}

func TestRequestCredentialsOverrideDefault(t *testing.T) {
	defaultCredentials := mustCredentials(t, "default-access", "default-api")
	requestCredentials := mustCredentials(t, "request-access", "request-api")
	clock := fixedClock{value: time.Unix(1_471_256_697, 0)}

	client, err := NewClient(
		WithDefaultCredentials(defaultCredentials),
		WithClock(clock),
		WithHTTPClient(&http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			assertAuthHeaders(t, request, requestCredentials, clock.Now())
			return jsonResponse(http.StatusOK, `{"code":200,"msg":"success","data":{}}`), nil
		})}),
		WithAPIBaseURL("https://api.example.test"),
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	_, err = doJSON[struct{}](context.Background(), client, requestSpec{
		operation:     "test.override",
		method:        http.MethodGet,
		path:          "/test",
		authenticated: true,
	}, []RequestOption{WithRequestCredentials(requestCredentials)})
	if err != nil {
		t.Fatalf("doJSON() error = %v", err)
	}
}

func TestInvalidRequestCredentialsDoNotFallBack(t *testing.T) {
	var calls atomic.Int64
	defaultCredentials := mustCredentials(t, "default-access", "default-api")
	client, err := NewClient(
		WithDefaultCredentials(defaultCredentials),
		WithHTTPClient(&http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			calls.Add(1)
			return nil, nil
		})}),
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	_, err = doJSON[struct{}](context.Background(), client, requestSpec{
		operation:     "test.invalid_override",
		method:        http.MethodGet,
		path:          "/test",
		authenticated: true,
	}, []RequestOption{WithRequestCredentials(Credentials{})})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("error = %v, want ErrInvalidCredentials", err)
	}
	if got := calls.Load(); got != 0 {
		t.Fatalf("transport calls = %d, want 0", got)
	}
}

func TestConcurrentRequestCredentialsRemainIsolated(t *testing.T) {
	clock := fixedClock{value: time.Unix(1_700_000_000, 0)}
	var calls atomic.Int64
	var failures atomic.Int64
	client, err := NewClient(
		WithClock(clock),
		WithAPIBaseURL("https://api.example.test"),
		WithHTTPClient(&http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			access := request.Header.Get("access-key")
			apiKey := strings.TrimPrefix(access, "access-")
			credentials := mustCredentials(t, access, "api-"+apiKey)
			if request.Header.Get("token") != expectedToken(credentials, clock.Now()) {
				failures.Add(1)
			}
			calls.Add(1)
			return jsonResponse(http.StatusOK, `{"code":200,"data":{}}`), nil
		})}),
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	const count = 32
	var group sync.WaitGroup
	group.Add(count)
	for index := range count {
		go func() {
			defer group.Done()
			value := string("ABCDEFGHIJKLMNOPQRSTUVWXYZ012345"[index])
			credentials := mustCredentials(t, "access-"+value, "api-"+value)
			_, callErr := doJSON[struct{}](context.Background(), client, requestSpec{
				operation:     "test.concurrent",
				method:        http.MethodGet,
				path:          "/test",
				authenticated: true,
			}, []RequestOption{WithRequestCredentials(credentials)})
			if callErr != nil {
				failures.Add(1)
			}
		}()
	}
	group.Wait()

	if got := calls.Load(); got != count {
		t.Fatalf("transport calls = %d, want %d", got, count)
	}
	if got := failures.Load(); got != 0 {
		t.Fatalf("credential isolation failures = %d", got)
	}
}

func mustCredentials(t *testing.T, accessKey, apiKey string) Credentials {
	t.Helper()
	credentials, err := NewCredentials(accessKey, apiKey)
	if err != nil {
		t.Fatalf("NewCredentials() error = %v", err)
	}
	return credentials
}

func assertAuthHeaders(t *testing.T, request *http.Request, credentials Credentials, now time.Time) {
	t.Helper()
	if got := request.Header.Get("access-key"); got != credentials.accessKey {
		t.Errorf("access-key = %q, want %q", got, credentials.accessKey)
	}
	if got := request.Header.Get("timestamp"); got != "1471256697" {
		t.Errorf("timestamp = %q, want 1471256697", got)
	}
	if got := request.Header.Get("token"); got != expectedToken(credentials, now) {
		t.Errorf("token = %q, want independently calculated token", got)
	}
}

func expectedToken(credentials Credentials, now time.Time) string {
	timestamp := []byte(strconv.FormatInt(now.Unix(), 10))
	inner := md5.Sum(timestamp)                                                 //nolint:gosec // Mintegral 的协议固定使用 MD5。
	outer := md5.Sum([]byte(credentials.apiKey + hex.EncodeToString(inner[:]))) //nolint:gosec // Mintegral 协议要求。
	return hex.EncodeToString(outer[:])
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
