package mintegral

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const maxJSONResponseBytes = 8 << 20

type responseEnvelope struct {
	Data    json.RawMessage `json:"data"`
	Msg     string          `json:"msg"`
	Message string          `json:"message"`
	Code    int             `json:"code"`
}

func decodeEnvelope[T any](response *http.Response, operation string, allowEmptyData bool, now time.Time, sensitiveValues ...string) (T, error) {
	var zero T
	body, err := readBounded(response.Body, maxJSONResponseBytes)
	if err != nil {
		return zero, fmt.Errorf("%w: %s response body: %w", ErrUnexpectedResponse, operation, err)
	}
	var envelope responseEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		return zero, fmt.Errorf("%w: %s returned invalid JSON", ErrUnexpectedResponse, operation)
	}
	message, err := envelopeMessage(envelope.Msg, envelope.Message)
	if err != nil {
		return zero, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 || envelope.Code != 200 {
		return zero, &APIError{
			Operation:        operation,
			HTTPStatus:       response.StatusCode,
			Code:             envelope.Code,
			Message:          redactAPIErrorMessage(message, sensitiveValues...),
			RetryAfter:       retryAfterDuration(response.Header.Get("Retry-After"), now),
			permissionDenied: containsPermissionDenied(message),
		}
	}
	if len(envelope.Data) == 0 || string(envelope.Data) == "null" {
		if allowEmptyData {
			return zero, nil
		}
		return zero, fmt.Errorf("%w: %s returned empty data", ErrUnexpectedResponse, operation)
	}
	var data T
	if err := json.Unmarshal(envelope.Data, &data); err != nil {
		return zero, fmt.Errorf("%w: %s data has an invalid field type", ErrUnexpectedResponse, operation)
	}
	return data, nil
}

func retryAfterDuration(value string, now time.Time) time.Duration {
	value = strings.TrimSpace(value)
	seconds, err := strconv.ParseInt(value, 10, 64)
	if err == nil {
		if seconds > 0 && seconds <= maxRetryAfterSeconds {
			return time.Duration(seconds) * time.Second
		}
		return 0
	}
	date, err := http.ParseTime(value)
	if err != nil || !date.After(now) {
		return 0
	}
	return date.Sub(now)
}

const maxRetryAfterSeconds = int64((1<<63 - 1) / time.Second)

func envelopeMessage(msg, message string) (string, error) {
	msg = strings.TrimSpace(msg)
	message = strings.TrimSpace(message)
	if msg != "" && message != "" && msg != message {
		return "", fmt.Errorf("%w: conflicting msg and message fields", ErrUnexpectedResponse)
	}
	if msg != "" {
		return msg, nil
	}
	return message, nil
}

func readBounded(reader io.Reader, maximum int64) ([]byte, error) {
	limited := io.LimitReader(reader, maximum+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maximum {
		return nil, fmt.Errorf("body exceeds %d bytes", maximum)
	}
	return data, nil
}
