package mintegral

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ReportService 提供高级报表 v2 操作。
type ReportService struct{ client *Client }

// Reports 返回高级报表服务。
func (c *Client) Reports() *ReportService { return &ReportService{client: c} }

// Status 查询或触发报表生成。
func (s *ReportService) Status(ctx context.Context, query ReportQuery, options ...RequestOption) (ReportStatus, error) { //nolint:gocritic // 公开 API 按值接收请求，避免调用方共享可变状态。
	normalized, err := normalizeReportQuery(&query)
	if err != nil {
		return ReportStatus{}, err
	}
	return s.status(ctx, &normalized, options)
}

func (s *ReportService) status(ctx context.Context, query *ReportQuery, options []RequestOption) (status ReportStatus, err error) {
	spec := reportSpec(query, 1)
	response, err := s.client.execute(ctx, spec, options)
	if err != nil {
		return status, err
	}
	if response == nil {
		return status, fmt.Errorf("%w: report status returned a nil response", ErrUnexpectedResponse)
	}
	defer func() { err = errors.Join(err, response.Body.Close()) }()
	body, err := readBounded(response.Body, maxJSONResponseBytes)
	if err != nil {
		return status, fmt.Errorf("%w: report status body", ErrUnexpectedResponse)
	}
	var envelope responseEnvelope
	if json.Unmarshal(body, &envelope) != nil {
		return status, fmt.Errorf("%w: report status JSON", ErrUnexpectedResponse)
	}
	message, messageErr := envelopeMessage(envelope.Msg, envelope.Message)
	if messageErr != nil {
		return status, messageErr
	}
	permissionDenied := containsPermissionDenied(message)
	message = redactAPIErrorMessage(message, s.client.requestSensitiveValues(&spec, options, response)...)
	var data struct {
		Hours      reportHours `json:"hours"`
		IsComplete bool        `json:"is_complete"`
	}
	if len(envelope.Data) > 0 && json.Unmarshal(envelope.Data, &data) != nil {
		return status, fmt.Errorf("%w: report status data", ErrUnexpectedResponse)
	}
	status = ReportStatus{Code: envelope.Code, Hours: []int(data.Hours), IsComplete: data.IsComplete}
	if response.StatusCode != http.StatusOK || (envelope.Code != 200 && envelope.Code != 201 && envelope.Code != 202) {
		return status, &APIError{Operation: "reports.status", HTTPStatus: response.StatusCode, Code: envelope.Code, Message: message, permissionDenied: permissionDenied}
	}
	return status, nil
}

type reportHours []int

func (h *reportHours) UnmarshalJSON(data []byte) error {
	var values []int
	if err := json.Unmarshal(data, &values); err == nil {
		*h = values
		return nil
	}
	var value int
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	*h = []int{value}
	return nil
}

// Open 等待报表可用并打开 TSV 流。默认只接受完整报表。
func (s *ReportService) Open(ctx context.Context, request ReportOpenRequest, options ...RequestOption) (*ReportStream, error) { //nolint:gocritic // 公开 API 按值接收请求，避免调用方共享可变状态。
	query, err := normalizeReportQuery(&request.Query)
	if err != nil {
		return nil, err
	}
	interval, maximum := request.PollInterval, request.MaxWait
	if interval == 0 {
		interval = 5 * time.Second
	}
	if maximum == 0 {
		maximum = 30 * time.Minute
	}
	if interval < 0 || maximum < 0 {
		return nil, fmt.Errorf("%w: invalid polling duration", ErrInvalidRequest)
	}
	pollCtx, cancel := context.WithTimeoutCause(ctx, maximum, ErrReportTimeout)
	for {
		status, statusErr := s.status(pollCtx, &query, options)
		if statusErr != nil {
			cancel()
			return nil, statusErr
		}
		if status.Code == 200 && (request.AllowIncomplete || status.IsComplete) {
			stream, retry, openErr := s.download(ctx, &query, options)
			if openErr != nil {
				cancel()
				return nil, openErr
			}
			if !retry {
				cancel()
				stream.status = status
				return stream, nil
			}
		}
		if waitErr := waitForRetry(pollCtx, interval); waitErr != nil {
			cause := context.Cause(pollCtx)
			cancel()
			return nil, cause
		}
	}
}

func (s *ReportService) download(ctx context.Context, query *ReportQuery, options []RequestOption) (*ReportStream, bool, error) {
	spec := reportSpec(query, 2)
	response, err := s.client.execute(ctx, spec, options)
	if err != nil {
		return nil, false, err
	}
	if response == nil {
		return nil, false, fmt.Errorf("%w: report download returned a nil response", ErrUnexpectedResponse)
	}
	reader := bufio.NewReader(response.Body)
	media, _, mediaErr := mime.ParseMediaType(response.Header.Get("Content-Type"))
	prefix, peekErr := reader.Peek(64)
	if peekErr != nil && !errors.Is(peekErr, io.EOF) {
		return nil, false, closeReportResponse(response, fmt.Errorf("%w: inspect report download: %w", ErrUnexpectedResponse, peekErr))
	}
	trimmed := bytes.TrimSpace(prefix)
	containsJSON := mediaErr == nil && (media == "application/json" || strings.HasSuffix(media, "+json"))
	if response.StatusCode < 200 || response.StatusCode >= 300 || containsJSON || (len(trimmed) > 0 && trimmed[0] == '{') {
		body, readErr := readBounded(reader, maxJSONResponseBytes)
		if readErr != nil {
			return nil, false, closeReportResponse(response, readErr)
		}
		var envelope responseEnvelope
		if json.Unmarshal(body, &envelope) != nil {
			return nil, false, closeReportResponse(response, fmt.Errorf("%w: report download JSON", ErrUnexpectedResponse))
		}
		message, messageErr := envelopeMessage(envelope.Msg, envelope.Message)
		if messageErr != nil {
			return nil, false, closeReportResponse(response, messageErr)
		}
		permissionDenied := containsPermissionDenied(message)
		message = redactAPIErrorMessage(message, s.client.requestSensitiveValues(&spec, options, response)...)
		if envelope.Code >= 203 && envelope.Code <= 205 {
			return nil, true, closeReportResponse(response, nil)
		}
		return nil, false, closeReportResponse(response, &APIError{Operation: "reports.open", HTTPStatus: response.StatusCode, Code: envelope.Code, Message: message, permissionDenied: permissionDenied})
	}
	return newReportStream(response.Body, reader), false, nil
}

func closeReportResponse(response *http.Response, cause error) error {
	return errors.Join(cause, response.Body.Close())
}

func reportSpec(query *ReportQuery, reportType int) requestSpec {
	return requestSpec{operation: "reports.data", method: http.MethodGet, path: reportPath, authenticated: true, retryable: true, query: func() (url.Values, error) { return reportValues(query, reportType), nil }}
}
