package mintegral_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	mintegral "github.com/jageros/mintegral-go"
)

type exampleRoundTripper func(*http.Request) (*http.Response, error)

func (transport exampleRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	return transport(request)
}

func exampleBalanceResponse(request *http.Request) (*http.Response, error) {
	if request.Header.Get("access-key") == "" || request.Header.Get("token") == "" || request.Header.Get("timestamp") == "" {
		return nil, fmt.Errorf("missing Mintegral authentication headers")
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body: io.NopCloser(strings.NewReader(
			`{"code":200,"msg":"success","data":{"total":1,"list":[{"user_id":7,"username":"demo","currency":"USD","balance":"12.3400"}]}}`,
		)),
		Request: request,
	}, nil
}

func ExampleNewClient() {
	credentials, err := mintegral.NewCredentials("example-access", "example-api")
	if err != nil {
		panic(err)
	}
	client, err := mintegral.NewClient(
		mintegral.WithDefaultCredentials(credentials),
		mintegral.WithHTTPClient(&http.Client{Transport: exampleRoundTripper(exampleBalanceResponse)}),
	)
	if err != nil {
		panic(err)
	}
	balance, err := client.Accounts().Balance(context.Background())
	if err != nil {
		panic(err)
	}
	fmt.Println(balance.List[0].Currency, balance.List[0].Balance)
	// Output: USD 12.3400
}

func ExampleNewClient_requestCredentials() {
	client, err := mintegral.NewClient(
		mintegral.WithHTTPClient(&http.Client{Transport: exampleRoundTripper(exampleBalanceResponse)}),
	)
	if err != nil {
		panic(err)
	}
	credentials, err := mintegral.NewCredentials("tenant-access", "tenant-api")
	if err != nil {
		panic(err)
	}
	balance, err := client.Accounts().Balance(
		context.Background(),
		mintegral.WithRequestCredentials(credentials),
	)
	if err != nil {
		panic(err)
	}
	fmt.Println(balance.List[0].Username)
	// Output: demo
}
