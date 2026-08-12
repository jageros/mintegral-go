package mintegral

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"
)

func TestDoJSON_acceptsExplicitNullAndRejectsMissingDataUnlessExplicitlyAllowed(t *testing.T) {
	cases := []struct {
		name       string
		body       string
		allow      bool
		want       error
		wantResult bool
	}{
		{name: "explicit null returns zero", body: "{\"code\":200,\"data\": \n null \t}", wantResult: true},
		{name: "missing data rejects", body: `{"code":200}`, want: ErrUnexpectedResponse},
		{name: "missing data allowed", body: `{"code":200}`, allow: true, wantResult: true},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			// Given
			client, err := NewClient(WithHTTPClient(&http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return jsonResponse(http.StatusOK, testCase.body), nil
			})}))
			if err != nil {
				t.Fatalf("NewClient() error = %v", err)
			}

			// When
			result, callErr := doJSON[struct{}](context.Background(), client, requestSpec{operation: "test.empty_data", method: http.MethodGet, path: "/test", allowMissingData: testCase.allow}, nil)

			// Then
			if testCase.want == nil && callErr != nil {
				t.Fatalf("doJSON() error = %v", callErr)
			}
			if testCase.want != nil && !errors.Is(callErr, testCase.want) {
				t.Fatalf("doJSON() error = %v, want %v", callErr, testCase.want)
			}
			if testCase.wantResult && result != (struct{}{}) {
				t.Fatalf("doJSON() result = %#v, want zero result", result)
			}
		})
	}
}

func TestDecodeEnvelope_classifiesAPIErrorBeforeExplicitNullData(t *testing.T) {
	// Given
	response := jsonResponse(http.StatusBadRequest, `{"code":10000,"data":null}`)

	// When
	_, err := decodeEnvelope[struct{}](response, "test.null_api_error", false, time.Time{})
	if closeErr := response.Body.Close(); closeErr != nil {
		t.Errorf("response body close error = %v", closeErr)
	}

	// Then
	var apiError *APIError
	if !errors.As(err, &apiError) {
		t.Fatalf("decodeEnvelope() error = %v, want APIError", err)
	}
}
