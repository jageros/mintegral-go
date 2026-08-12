package mintegral

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"sync/atomic"
	"testing"
	"time"
)

func TestReportOpen_pollsThenParsesTSV(t *testing.T) {
	// Given
	var calls atomic.Int64
	client := reportTestClient(t, func(request *http.Request) (*http.Response, error) {
		call := calls.Add(1)
		assertReportQuery(t, request.URL.Query())
		if call == 1 {
			return jsonResponse(http.StatusOK, `{"code":201,"data":{"hours":[1],"is_complete":false}}`), nil
		}
		if call == 2 {
			return jsonResponse(http.StatusOK, `{"code":200,"data":{"hours":[1,2],"is_complete":true}}`), nil
		}
		return reportResponse("\ufeff" + reportHeader + "\tFuture\r\n" + reportRow("7", "1.2300") + "\t\"a\tb\nc\"\r\n"), nil
	})

	// When
	stream, err := client.Reports().Open(context.Background(), ReportOpenRequest{
		Query: reportTestQuery(t), PollInterval: time.Nanosecond,
	}, WithRequestCredentials(mustCredentials(t, "access", "api")))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer closeReportStream(t, stream)
	row, err := stream.Next()
	// Then
	if err != nil {
		t.Fatalf("Next() error = %v", err)
	}
	if row.Date.String() != "2026-08-12" || row.Impressions != 7 || row.Spend.String() != "1.2300" {
		t.Fatalf("row = %+v", row)
	}
	if value, ok := row.Extras.Get("Future"); !ok || value != "a\tb\nc" {
		t.Fatalf("extra = %q, %v", value, ok)
	}
	if _, err = stream.Next(); !errors.Is(err, io.EOF) {
		t.Fatalf("second Next() error = %v, want EOF", err)
	}
	if err = stream.Close(); err != nil {
		t.Fatalf("second Close() error = %v", err)
	}
}

func TestReportOpen_returnsToStatusWhenDownloadNotReady(t *testing.T) {
	// Given
	var types []string
	client := reportTestClient(t, func(request *http.Request) (*http.Response, error) {
		types = append(types, request.URL.Query().Get("type"))
		switch len(types) {
		case 1:
			return jsonResponse(200, `{"code":200,"data":{"is_complete":true}}`), nil
		case 2:
			return jsonResponse(200, `{"code":203,"msg":"building"}`), nil
		case 3:
			return jsonResponse(200, `{"code":200,"data":{"is_complete":true}}`), nil
		case 4:
			return jsonResponse(200, `{"code":204,"msg":"building"}`), nil
		case 5:
			return jsonResponse(200, `{"code":200,"data":{"is_complete":true}}`), nil
		case 6:
			return jsonResponse(200, `{"code":205,"msg":"building"}`), nil
		case 7:
			return jsonResponse(200, `{"code":200,"data":{"is_complete":true}}`), nil
		default:
			return reportResponse(reportHeader + "\n" + reportRow("1", "0") + "\n"), nil
		}
	})

	// When
	stream, err := client.Reports().Open(context.Background(), ReportOpenRequest{Query: reportTestQuery(t), PollInterval: time.Nanosecond}, WithRequestCredentials(mustCredentials(t, "a", "b")))
	// Then
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer closeReportStream(t, stream)
	if !reflect.DeepEqual(types, []string{"1", "2", "1", "2", "1", "2", "1", "2"}) {
		t.Fatalf("request types = %v", types)
	}
}

func TestReportOpen_downloadsIncompleteReportWhenExplicitlyAllowed(t *testing.T) {
	// Given
	var types []string
	client := reportTestClient(t, func(request *http.Request) (*http.Response, error) {
		types = append(types, request.URL.Query().Get("type"))
		if len(types) == 1 {
			return jsonResponse(200, `{"code":200,"data":{"is_complete":false}}`), nil
		}
		return reportResponse(reportHeader + "\n" + reportRow("1", "0") + "\n"), nil
	})

	// When
	stream, err := client.Reports().Open(context.Background(), ReportOpenRequest{
		Query: reportTestQuery(t), AllowIncomplete: true,
	}, WithRequestCredentials(mustCredentials(t, "a", "b")))
	// Then
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer closeReportStream(t, stream)
	if !reflect.DeepEqual(types, []string{"1", "2"}) {
		t.Fatalf("request types = %v", types)
	}
}

func TestReportOpen_timeoutAndCancellation(t *testing.T) {
	for _, test := range []struct {
		name string
		ctx  func() (context.Context, context.CancelFunc)
		wait time.Duration
		want error
	}{
		{"timeout", func() (context.Context, context.CancelFunc) { return context.WithCancel(context.Background()) }, time.Nanosecond, ErrReportTimeout},
		{"cancel", func() (context.Context, context.CancelFunc) { return context.WithCancel(context.Background()) }, time.Hour, context.Canceled},
	} {
		t.Run(test.name, func(t *testing.T) {
			// Given
			client := reportTestClient(t, func(*http.Request) (*http.Response, error) {
				return jsonResponse(200, `{"code":202,"data":{}}`), nil
			})
			ctx, cancel := test.ctx()
			if errors.Is(test.want, context.Canceled) {
				cancel()
			} else {
				defer cancel()
			}

			// When
			_, err := client.Reports().Open(ctx, ReportOpenRequest{Query: reportTestQuery(t), PollInterval: time.Nanosecond, MaxWait: test.wait}, WithRequestCredentials(mustCredentials(t, "a", "b")))

			// Then
			if !errors.Is(err, test.want) {
				t.Fatalf("Open() error = %v, want %v", err, test.want)
			}
		})
	}
}

func reportTestClient(t *testing.T, roundTrip roundTripFunc) *Client {
	t.Helper()
	client, err := NewClient(WithAPIBaseURL("https://api.example.test"), WithHTTPClient(&http.Client{Transport: roundTrip}), WithRetryPolicy(RetryPolicy{MaxAttempts: 1}))
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func reportTestQuery(t *testing.T) ReportQuery {
	t.Helper()
	start := mustReportDate(t, "2026-08-12")
	return ReportQuery{Timezone: "+8", StartDate: start, EndDate: start, Dimensions: []Dimension{DimensionOffer}, Granularity: GranularityDaily}
}

func assertReportQuery(t *testing.T, query url.Values) {
	t.Helper()
	if query.Get("start_time") != "2026-08-12" || query.Get("dimension_option") != "Offer" {
		t.Fatalf("query = %v", query)
	}
}

func reportResponse(body string) *http.Response {
	response := jsonResponse(http.StatusOK, body)
	response.Header.Set("Content-Type", "text/tab-separated-values")
	return response
}

func TestReportConsume_acknowledgesOnlySuccessfulBatches(t *testing.T) {
	// Given
	client := reportTestClient(t, func(request *http.Request) (*http.Response, error) {
		if request.URL.Query().Get("type") == "1" {
			return jsonResponse(200, `{"code":200,"data":{"is_complete":true}}`), nil
		}
		return reportResponse(reportHeader + "\n" + reportRow("1", "0") + "\n" + reportRow("2", "0") + "\n" + reportRow("3", "0") + "\n"), nil
	})
	stop := errors.New("store unavailable")
	var batches [][]int64

	// When
	delivery, err := client.Reports().Consume(context.Background(), ReportConsumeRequest{Open: ReportOpenRequest{Query: reportTestQuery(t)}, BatchSize: 2}, func(_ context.Context, rows []ReportRow) error {
		values := make([]int64, len(rows))
		for index := range rows {
			values[index] = rows[index].Impressions
		}
		batches = append(batches, values)
		if len(batches) == 2 {
			return stop
		}
		return nil
	}, WithRequestCredentials(mustCredentials(t, "a", "b")))

	// Then
	if !errors.Is(err, ErrPartialDelivery) || !errors.Is(err, stop) {
		t.Fatalf("Consume() error = %v", err)
	}
	if delivery.AcknowledgedRows != 2 || !reflect.DeepEqual(batches, [][]int64{{1, 2}, {3}}) {
		t.Fatalf("delivery=%+v batches=%v", delivery, batches)
	}
	if delivery.ParsedRows != 3 || !delivery.Status.IsComplete {
		t.Fatalf("delivery=%+v, want three parsed rows and complete status", delivery)
	}
}

func TestReportConsume_returnsPartialDeliveryWhenCloseFailsAfterAcknowledgement(t *testing.T) {
	// Given
	closeFailure := errors.New("close failed")
	client := reportTestClient(t, func(request *http.Request) (*http.Response, error) {
		if request.URL.Query().Get("type") == "1" {
			return jsonResponse(200, `{"code":200,"data":{"hours":[24],"is_complete":true}}`), nil
		}
		response := reportResponse(reportHeader + "\n" + reportRow("1", "0") + "\n")
		response.Body = &reportCloseErrorBody{Reader: response.Body, closeErr: closeFailure}
		return response, nil
	})

	// When
	delivery, err := client.Reports().Consume(context.Background(), ReportConsumeRequest{
		Open: ReportOpenRequest{Query: reportTestQuery(t)}, BatchSize: 1,
	}, func(context.Context, []ReportRow) error { return nil }, WithRequestCredentials(mustCredentials(t, "a", "b")))

	// Then
	if !errors.Is(err, ErrPartialDelivery) || !errors.Is(err, closeFailure) {
		t.Fatalf("Consume() error = %v, want partial close failure", err)
	}
	if delivery.ParsedRows != 1 || delivery.AcknowledgedRows != 1 || !delivery.Status.IsComplete {
		t.Fatalf("delivery=%+v", delivery)
	}
}

type reportCloseErrorBody struct {
	io.Reader
	closeErr error
}

func (b *reportCloseErrorBody) Close() error { return b.closeErr }

const reportHeader = "Date\tCurrency\tImpression\tClick\tConversion\tEcpm\tCpc\tCtr\tCvr\tIvr\tSpend"

func reportRow(impressions, spend string) string {
	return "20260812\tUSD\t" + impressions + "\t0\t0\t0\t0\t0\t0\t0\t" + spend
}

func mustReportDate(t *testing.T, value string) Date {
	t.Helper()
	date, err := ParseDate(value)
	if err != nil {
		t.Fatalf("ParseDate(%q) error = %v", value, err)
	}
	return date
}

func closeReportStream(t *testing.T, stream *ReportStream) {
	t.Helper()
	if err := stream.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}
