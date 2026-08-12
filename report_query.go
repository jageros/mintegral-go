package mintegral

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const reportPath = "/api/v2/reports/data"

func normalizeReportQuery(source *ReportQuery) (ReportQuery, error) {
	query := *source
	start, err := time.Parse(time.DateOnly, query.StartDate.String())
	if err != nil {
		return ReportQuery{}, fmt.Errorf("%w: start date is required", ErrInvalidRequest)
	}
	end, err := time.Parse(time.DateOnly, query.EndDate.String())
	if err != nil {
		return ReportQuery{}, fmt.Errorf("%w: end date is required", ErrInvalidRequest)
	}
	if end.Before(start) || end.Sub(start) > 6*24*time.Hour {
		return ReportQuery{}, fmt.Errorf("%w: report date range must contain at most seven inclusive days", ErrInvalidRequest)
	}
	query.Timezone = strings.TrimSpace(query.Timezone)
	if query.Timezone == "" {
		query.Timezone = "+8"
	}
	if !validReportTimezone(query.Timezone) {
		return ReportQuery{}, fmt.Errorf("%w: invalid timezone", ErrInvalidRequest)
	}
	if query.Granularity == "" {
		query.Granularity = GranularityDaily
	}
	if query.Granularity != GranularityDaily && query.Granularity != GranularityHourly {
		return ReportQuery{}, fmt.Errorf("%w: invalid granularity", ErrInvalidRequest)
	}
	if len(query.Dimensions) == 0 {
		return ReportQuery{}, fmt.Errorf("%w: dimensions are required", ErrInvalidRequest)
	}
	seen := make(map[Dimension]struct{}, len(query.Dimensions))
	for _, dimension := range query.Dimensions {
		if !validDimension(dimension) {
			return ReportQuery{}, fmt.Errorf("%w: invalid dimension %q", ErrInvalidRequest, dimension)
		}
		if _, duplicate := seen[dimension]; duplicate {
			return ReportQuery{}, fmt.Errorf("%w: duplicate dimension", ErrInvalidRequest)
		}
		seen[dimension] = struct{}{}
	}
	_, creative := seen[DimensionCreative]
	_, endcard := seen[DimensionEndcard]
	_, sub := seen[DimensionSub]
	_, pkg := seen[DimensionPackage]
	if (creative || endcard) && (sub || pkg || query.Granularity == GranularityHourly) {
		return ReportQuery{}, fmt.Errorf("%w: forbidden report dimension combination", ErrInvalidRequest)
	}
	return query, nil
}

func validDimension(d Dimension) bool {
	switch d {
	case DimensionOffer, DimensionCampaign, DimensionCampaignPackage, DimensionCreative, DimensionAdType, DimensionSub, DimensionPackage, DimensionLocation, DimensionEndcard, DimensionAdOutputType, DimensionDma, DimensionState:
		return true
	default:
		return false
	}
}

func validReportTimezone(value string) bool {
	if len(value) < 2 || (value[0] != '+' && value[0] != '-') {
		return false
	}
	hour, err := strconv.Atoi(value[1:])
	return err == nil && hour <= 11
}

func reportValues(query *ReportQuery, reportType int) url.Values {
	dimensions := make([]string, len(query.Dimensions))
	for index := range query.Dimensions {
		dimensions[index] = string(query.Dimensions[index])
	}
	return url.Values{"timezone": {query.Timezone}, "start_time": {query.StartDate.String()}, "end_time": {query.EndDate.String()}, "dimension_option": {strings.Join(dimensions, ",")}, "time_granularity": {string(query.Granularity)}, "type": {strconv.Itoa(reportType)}}
}
