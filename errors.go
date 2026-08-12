package mintegral

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	// ErrCredentialsRequired 表示当前鉴权请求没有可用凭据。
	ErrCredentialsRequired = errors.New("mintegral: credentials required")
	// ErrInvalidCredentials 表示显式提供的凭据不完整。
	ErrInvalidCredentials = errors.New("mintegral: invalid credentials")
	// ErrInvalidRequest 表示请求参数不符合接口契约。
	ErrInvalidRequest = errors.New("mintegral: invalid request")
	// ErrTransport 表示 HTTP 传输失败，错误文本不会暴露请求地址或凭据。
	ErrTransport = errors.New("mintegral: transport failure")
	// ErrAPI 表示 Mintegral 返回了业务错误。
	ErrAPI = errors.New("mintegral: API failure")
	// ErrPermissionDenied 表示账号没有调用接口所需的高级权限。
	ErrPermissionDenied = errors.New("mintegral: permission denied")
	// ErrRateLimited 表示请求受到 HTTP 限流。
	ErrRateLimited = errors.New("mintegral: rate limited")
	// ErrUnexpectedResponse 表示响应不符合已记录的 JSON 或 HTTP 契约。
	ErrUnexpectedResponse = errors.New("mintegral: unexpected response")
	// ErrInvalidReport 表示下载的报表表头或数据行无效。
	ErrInvalidReport = errors.New("mintegral: invalid report")
	// ErrReportTimeout 表示报表在最长等待时间内未就绪。
	ErrReportTimeout = errors.New("mintegral: report timeout")
	// ErrUploadExpired 表示人群包预签名上传计划已经过期。
	ErrUploadExpired = errors.New("mintegral: upload plan expired")
	// ErrOutcomeUnknown 表示写请求已经发出，但客户端无法确认服务端是否已应用。
	ErrOutcomeUnknown = errors.New("mintegral: outcome unknown")
	// ErrPartialDelivery 表示报表已有部分批次交付成功，后续读取或处理失败。
	ErrPartialDelivery = errors.New("mintegral: partial delivery")
)

// APIError 描述 Mintegral 返回的 HTTP 或业务错误。
type APIError struct {
	// Operation 是失败的 SDK 操作名，不包含请求地址。
	Operation string
	// Message 是经过脱敏和长度限制的服务端错误说明。
	Message string
	// HTTPStatus 是服务端返回的 HTTP 状态码。
	HTTPStatus int
	// Code 是 Mintegral 业务状态码。
	Code int
	// RetryAfter 是服务端 Retry-After 建议的等待时长；未提供或无效时为零。
	RetryAfter       time.Duration
	permissionDenied bool
}

// Error 返回不包含凭据和完整请求地址的错误摘要。
func (e *APIError) Error() string {
	if e == nil {
		return "mintegral: API failure"
	}
	message := redactAPIErrorMessage(e.Message)
	if len(message) > 512 {
		message = message[:512]
	}
	return fmt.Sprintf("mintegral: %s failed (http=%d code=%d): %s", e.Operation, e.HTTPStatus, e.Code, message)
}

func redactAPIErrorMessage(message string, sensitiveValues ...string) string {
	message = strings.TrimSpace(message)
	for _, sensitive := range sensitiveValues {
		if sensitive != "" && strings.Contains(message, sensitive) {
			return "upstream API error"
		}
	}
	lower := strings.ToLower(message)
	for _, marker := range []string{
		"http://", "https://", "?", "access", "api", "token", "policy", "signature",
	} {
		if strings.Contains(lower, marker) {
			return "upstream API error"
		}
	}
	return message
}

// Is 支持通过 errors.Is 按错误类别判断 API 错误。
func (e *APIError) Is(target error) bool {
	if target == ErrAPI {
		return true
	}
	if target == ErrRateLimited {
		return e != nil && e.HTTPStatus == httpStatusTooManyRequests
	}
	if target == ErrPermissionDenied {
		return e != nil && (e.HTTPStatus == httpStatusForbidden || e.permissionDenied || containsPermissionDenied(e.Message))
	}
	return false
}

type transportError struct {
	operation      string
	cause          error
	outcomeUnknown bool
}

func (e *transportError) Error() string {
	return "mintegral: " + e.operation + ": transport failure"
}

func (e *transportError) Unwrap() error { return e.cause }

func (e *transportError) Is(target error) bool {
	return target == ErrTransport || (target == ErrOutcomeUnknown && e.outcomeUnknown)
}

type partialDeliveryError struct {
	cause error
}

func (e *partialDeliveryError) Error() string {
	return "mintegral: report delivery failed after partial success"
}
func (e *partialDeliveryError) Unwrap() error { return e.cause }
func (e *partialDeliveryError) Is(target error) bool {
	return target == ErrPartialDelivery
}

func containsPermissionDenied(message string) bool {
	lower := strings.ToLower(message)
	return strings.Contains(lower, "permission denied") || strings.Contains(lower, "no permission")
}

const (
	httpStatusForbidden       = 403
	httpStatusTooManyRequests = 429
)
