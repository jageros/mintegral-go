package mintegral

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAppServiceNames_sendsAuthenticatedPostJSON(t *testing.T) {
	// Given
	credentials := mustCredentials(t, "access", "api")
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost || request.URL.Path != "/api/open/v1/target-apps/app-name" {
			t.Errorf("request = %s %s, want POST /api/open/v1/target-apps/app-name", request.Method, request.URL.Path)
		}
		assertAuthHeaders(t, request, credentials, testTime)
		if body := decodeBody(t, request); body["package_name"] != "com.example,id123" {
			t.Errorf("body = %#v, want package_name", body)
		}
		writeJSONResponse(t, writer, `{"code":200,"data":[{"package_name":"com.example","app_name":"Example"}]}`)
	}))
	defer server.Close()
	client := newServiceClient(t, server.URL, credentials)

	// When
	apps, err := client.Apps().Names(context.Background(), AppNameRequest{PackageName: "com.example,id123"})

	// Then
	if err != nil || len(apps) != 1 || apps[0].AppName != "Example" {
		t.Fatalf("Names() = %#v, %v; want decoded app", apps, err)
	}
}
