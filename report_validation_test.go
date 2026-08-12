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
