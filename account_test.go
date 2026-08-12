package mintegral

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAccountServiceBalance_sendsAuthenticatedGet(t *testing.T) {
	// Given
	credentials := mustCredentials(t, "access", "api")
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", request.Method)
		}
		if request.URL.Path != "/api/open/v1/account/balance" {
			t.Errorf("path = %q", request.URL.Path)
		}
		assertAuthHeaders(t, request, credentials, fixedClock{value: testTime}.Now())
		writeJSONResponse(t, writer, `{"code":200,"data":{"total":1,"list":[{"user_id":7,"username":"advertiser","currency":"USD","balance":12.5}]}}`)
	}))
	defer server.Close()
	client := newServiceClient(t, server.URL, credentials)

	// When
	balance, err := client.Accounts().Balance(context.Background())
	// Then
	if err != nil {
		t.Fatalf("Balance() error = %v", err)
	}
	if balance.Total != 1 || len(balance.List) != 1 || balance.List[0].UserID != UserID(7) || balance.List[0].Balance != DecimalText("12.5") {
		t.Fatalf("Balance() = %#v, want decoded balance", balance)
	}
}

func newServiceClient(t *testing.T, baseURL string, credentials Credentials) *Client {
	t.Helper()
	client, err := NewClient(
		WithAPIBaseURL(baseURL),
		WithDefaultCredentials(credentials),
		WithClock(fixedClock{value: testTime}),
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	return client
}

func decodeBody(t *testing.T, request *http.Request) map[string]any {
	t.Helper()
	var body map[string]any
	if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
		t.Fatalf("decode JSON body: %v", err)
	}
	return body
}

func writeJSONResponse(t *testing.T, writer http.ResponseWriter, body string) {
	t.Helper()
	if _, err := io.WriteString(writer, body); err != nil {
		t.Errorf("write JSON response: %v", err)
	}
}
