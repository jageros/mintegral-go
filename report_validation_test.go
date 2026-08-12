package mintegral

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"testing"
)

func TestReportQuery_acceptsSevenInclusiveDays(t *testing.T) {
	// Given
	start := mustReportDate(t, "2026-08-01")
	end := mustReportDate(t, "2026-08-07")
	query := ReportQuery{StartDate: start, EndDate: end, Dimensions: []Dimension{DimensionOffer}}

	// When
	normalized, err := normalizeReportQuery(&query)

	// Then
	if err != nil || normalized.Timezone != "+8" || normalized.Granularity != GranularityDaily {
		t.Fatalf("normalizeReportQuery() = %+v, %v", normalized, err)
	}
}

func TestReportQuery_rejectsForbiddenDimensionCombinations(t *testing.T) {
	tests := []struct {
		dimensions  []Dimension
		granularity Granularity
	}{
		{[]Dimension{DimensionCreative, DimensionSub}, GranularityDaily},
		{[]Dimension{DimensionCreative, DimensionPackage}, GranularityDaily},
		{[]Dimension{DimensionCreative}, GranularityHourly},
		{[]Dimension{DimensionEndcard, DimensionSub}, GranularityDaily},
		{[]Dimension{DimensionEndcard, DimensionPackage}, GranularityDaily},
		{[]Dimension{DimensionEndcard}, GranularityHourly},
	}
	for _, test := range tests {
		// Given
		query := reportTestQuery(t)
		query.Dimensions = test.dimensions
		query.Granularity = test.granularity

		// When
		_, err := normalizeReportQuery(&query)

		// Then
		if !errors.Is(err, ErrInvalidRequest) {
			t.Fatalf("normalizeReportQuery(%v, %s) error = %v", test.dimensions, test.granularity, err)
		}
	}
}

func TestReportQuery_rejectsEightInclusiveDays(t *testing.T) {
	// Given
	start := mustReportDate(t, "2026-08-01")
	end := mustReportDate(t, "2026-08-08")
	query := ReportQuery{StartDate: start, EndDate: end, Dimensions: []Dimension{DimensionOffer}}

	// When
	_, err := normalizeReportQuery(&query)

	// Then
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("normalizeReportQuery() error = %v", err)
	}
}

func TestReportStatus_acceptsScalarHours(t *testing.T) {
	// Given
	client := reportTestClient(t, func(*http.Request) (*http.Response, error) {
		return jsonResponse(200, `{"code":200,"data":{"hours":24,"is_complete":true}}`), nil
	})

	// When
	status, err := client.Reports().Status(context.Background(), reportTestQuery(t), WithRequestCredentials(mustCredentials(t, "a", "b")))

	// Then
	if err != nil || !reflect.DeepEqual(status.Hours, []int{24}) {
		t.Fatalf("Status() = %+v, %v", status, err)
	}
}

func TestReportStatus_returnsZeroValue_whenAcceptedResponseHasExplicitNullData(t *testing.T) {
	// Given
	client := reportTestClient(t, func(*http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, `{"code":202,"data": null}`), nil
	})

	// When
	status, err := client.Reports().Status(context.Background(), reportTestQuery(t), WithRequestCredentials(mustCredentials(t, "a", "b")))

	// Then
	if err != nil || !reflect.DeepEqual(status, ReportStatus{}) {
		t.Fatalf("Status() = %#v, %v; want zero status and nil error", status, err)
	}
}

func TestReportStatus_rejectsMissingDataAndClassifiesErrorsBeforeNullData(t *testing.T) {
	cases := []struct {
		name       string
		httpStatus int
		body       string
		want       error
	}{
		{name: "missing data", httpStatus: http.StatusOK, body: `{"code":200}`, want: ErrUnexpectedResponse},
		{name: "business error with null", httpStatus: http.StatusOK, body: `{"code":10000,"data":null}`},
		{name: "HTTP error with null", httpStatus: http.StatusBadRequest, body: `{"code":200,"data":null}`},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			// Given
			client := reportTestClient(t, func(*http.Request) (*http.Response, error) {
				return jsonResponse(testCase.httpStatus, testCase.body), nil
			})

			// When
			_, err := client.Reports().Status(context.Background(), reportTestQuery(t), WithRequestCredentials(mustCredentials(t, "a", "b")))

			// Then
			if testCase.want != nil {
				if !errors.Is(err, testCase.want) {
					t.Fatalf("Status() error = %v, want %v", err, testCase.want)
				}
				return
			}
			var apiError *APIError
			if !errors.As(err, &apiError) {
				t.Fatalf("Status() error = %v, want APIError", err)
			}
		})
	}
}

func TestReportHours_UnmarshalJSON_clearsPrepopulatedValue_whenJSONNull(t *testing.T) {
	// Given
	hours := reportHours{1, 2}

	// When
	err := hours.UnmarshalJSON([]byte(" \n null \t"))

	// Then
	if err != nil || hours != nil {
		t.Fatalf("reportHours.UnmarshalJSON() = %#v, %v; want nil hours and nil error", hours, err)
	}
}

func TestReportHours_UnmarshalJSON_rejectsNilReceiver(t *testing.T) {
	// Given
	var hours *reportHours

	// When
	err := hours.UnmarshalJSON([]byte(`[1]`))

	// Then
	if !errors.Is(err, ErrUnexpectedResponse) {
		t.Fatalf("reportHours.UnmarshalJSON() error = %v, want ErrUnexpectedResponse", err)
	}
}
