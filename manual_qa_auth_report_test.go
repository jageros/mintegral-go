package mintegral_test

import (
	"context"
	"crypto/md5" //nolint:gosec // Verify Mintegral's documented MD5 token vector.
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	mintegral "github.com/jageros/mintegral-go"
)

type manualQAFixedClock struct{ value time.Time }

func (clock manualQAFixedClock) Now() time.Time { return clock.value }

func manualQACredentials(t *testing.T, access, api string) mintegral.Credentials {
	t.Helper()
	credentials, err := mintegral.NewCredentials(access, api)
	if err != nil {
		t.Fatal(err)
	}
	return credentials
}

func manualQAExpectedToken(api string, now time.Time) string {
	timestamp := strconv.FormatInt(now.Unix(), 10)
	inner := md5.Sum([]byte(timestamp))                          //nolint:gosec // Verify the provider signing vector.
	outer := md5.Sum([]byte(api + hex.EncodeToString(inner[:]))) //nolint:gosec // Verify the provider signing vector.
	return hex.EncodeToString(outer[:])
}

func manualQAAuth(t *testing.T, request *http.Request, api string, now time.Time) {
	t.Helper()
	if request.Header.Get("access-key") == "" || request.Header.Get("timestamp") != strconv.FormatInt(now.Unix(), 10) || request.Header.Get("token") != manualQAExpectedToken(api, now) {
		t.Errorf("authentication headers = access=%q timestamp=%q token=%q", request.Header.Get("access-key"), request.Header.Get("timestamp"), request.Header.Get("token"))
	}
}

func manualQAWrite(t *testing.T, response http.ResponseWriter, body string) {
	t.Helper()
	if _, err := io.WriteString(response, body); err != nil {
		t.Errorf("write fake response: %v", err)
	}
}

func TestManualQANewClientWithoutDefaultCredentialsMakesNoNetworkCall(t *testing.T) {
	var calls atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		calls.Add(1)
		http.Error(response, "unexpected request", http.StatusInternalServerError)
	}))
	defer server.Close()
	client, err := mintegral.NewClient(mintegral.WithAPIBaseURL(server.URL), mintegral.WithStorageBaseURL(server.URL), mintegral.WithHTTPClient(server.Client()))
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.Accounts().Balance(context.Background())
	if !errors.Is(err, mintegral.ErrCredentialsRequired) {
		t.Fatalf("Balance() error = %v", err)
	}
	if calls.Load() != 0 {
		t.Fatalf("fake server calls = %d, want 0", calls.Load())
	}
}

func TestManualQACredentialsOverrideDefaultAndOfferStatus(t *testing.T) {
	now := time.Unix(1_471_256_697, 0)
	defaultCredentials := manualQACredentials(t, "default-access", "default-api")
	requestCredentials := manualQACredentials(t, "request-access", "request-api")
	var calls atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		calls.Add(1)
		switch request.URL.Path {
		case "/api/open/v1/account/balance":
			if request.Method != http.MethodGet {
				t.Errorf("balance method = %s", request.Method)
			}
			access := request.Header.Get("access-key")
			api := map[string]string{"default-access": "default-api", "request-access": "request-api"}[access]
			manualQAAuth(t, request, api, now)
			manualQAWrite(t, response, `{"code":200,"data":{"total":0,"list":[]}}`)
		case "/api/open/v1/offer/status":
			body, readErr := io.ReadAll(request.Body)
			if readErr != nil || string(body) != `{"offer_id":42,"status":"RUNNING"}` {
				t.Errorf("status body = %q, err=%v", body, readErr)
			}
			manualQAAuth(t, request, "request-api", now)
			manualQAWrite(t, response, `{"code":200,"data":null}`)
		default:
			http.NotFound(response, request)
		}
	}))
	defer server.Close()
	client, err := mintegral.NewClient(mintegral.WithAPIBaseURL(server.URL), mintegral.WithDefaultCredentials(defaultCredentials), mintegral.WithClock(manualQAFixedClock{value: now}), mintegral.WithHTTPClient(server.Client()))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = client.Accounts().Balance(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err = client.Accounts().Balance(context.Background(), mintegral.WithRequestCredentials(requestCredentials)); err != nil {
		t.Fatal(err)
	}
	if err = client.Offers().SetStatus(context.Background(), mintegral.SetOfferStatusRequest{OfferID: 42, Status: mintegral.OfferStatusRunning}, mintegral.WithRequestCredentials(requestCredentials)); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 3 {
		t.Fatalf("fake server calls = %d, want 3", calls.Load())
	}
}

func TestManualQAReportsStatusOpenAndTSV(t *testing.T) {
	now := time.Unix(1_471_256_697, 0)
	credentials := manualQACredentials(t, "report-access", "report-api")
	date, err := mintegral.ParseDate("2026-08-12")
	if err != nil {
		t.Fatal(err)
	}
	query := mintegral.ReportQuery{Timezone: "+8", StartDate: date, EndDate: date, Dimensions: []mintegral.Dimension{mintegral.DimensionOffer}, Granularity: mintegral.GranularityDaily}
	var calls atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		manualQAAuth(t, request, "report-api", now)
		if request.Method != http.MethodGet || request.URL.Path != "/api/v2/reports/data" || request.URL.Query().Get("start_time") != "2026-08-12" || request.URL.Query().Get("type") == "" {
			t.Errorf("report request = %s %s?%s", request.Method, request.URL.Path, request.URL.RawQuery)
		}
		typeValue := request.URL.Query().Get("type")
		switch calls.Add(1) {
		case 1, 2:
			if typeValue != "1" {
				t.Errorf("status type = %q", typeValue)
			}
			manualQAWrite(t, response, `{"code":200,"data":{"hours":[1],"is_complete":true}}`)
		default:
			if typeValue != "2" {
				t.Errorf("download type = %q", typeValue)
			}
			response.Header().Set("Content-Type", "text/tab-separated-values")
			manualQAWrite(t, response, "Date\tCurrency\tImpression\tClick\tConversion\tEcpm\tCpc\tCtr\tCvr\tIvr\tSpend\n20260812\tUSD\t7\t1\t0\t1.2300\t0.4\t0.2\t0.25\t0.05\t9.8700\n")
		}
	}))
	defer server.Close()
	client, err := mintegral.NewClient(mintegral.WithAPIBaseURL(server.URL), mintegral.WithClock(manualQAFixedClock{value: now}), mintegral.WithHTTPClient(server.Client()), mintegral.WithRetryPolicy(mintegral.RetryPolicy{MaxAttempts: 1}))
	if err != nil {
		t.Fatal(err)
	}
	status, err := client.Reports().Status(context.Background(), query, mintegral.WithRequestCredentials(credentials))
	if err != nil || status.Code != 200 || !status.IsComplete {
		t.Fatalf("Status() = %#v, %v", status, err)
	}
	stream, err := client.Reports().Open(context.Background(), mintegral.ReportOpenRequest{Query: query, PollInterval: time.Nanosecond, MaxWait: time.Second}, mintegral.WithRequestCredentials(credentials))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if closeErr := stream.Close(); closeErr != nil {
			t.Errorf("close report stream: %v", closeErr)
		}
	})
	row, err := stream.Next()
	if err != nil || row.Impressions != 7 || row.Spend.String() != "9.8700" || !strings.EqualFold(row.Currency, "USD") {
		t.Fatalf("TSV row = %#v, %v", row, err)
	}
	if _, err = stream.Next(); !errors.Is(err, io.EOF) {
		t.Fatalf("second Next() error = %v", err)
	}
	if calls.Load() != 3 {
		t.Fatalf("report calls = %d, want 3", calls.Load())
	}
}
