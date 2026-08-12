package mintegral

import (
	"context"
	//nolint:gosec // Mintegral 鉴权协议固定要求 MD5，不能替换为其他摘要算法。
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const maxRetryDrainBytes = 32 << 10

type preSendError struct{ cause error }

func (e *preSendError) Error() string { return e.cause.Error() }
func (e *preSendError) Unwrap() error { return e.cause }

//nolint:gocritic // requestSpec 是不可变的内部请求计划，按值传递可隔离调用方后续修改。
func (c *Client) execute(ctx context.Context, spec requestSpec, options []RequestOption) (*http.Response, error) {
	credentials := Credentials{}
	var err error
	if spec.authenticated {
		credentials, err = c.resolveCredentials(options)
		if err != nil {
			return nil, err
		}
	}
	requestURL, err := c.buildRequestURL(spec)
	if err != nil {
		return nil, err
	}
	attempts := 1
	if spec.retryable {
		attempts = c.retryPolicy.MaxAttempts
	}
	for attempt := 1; attempt <= attempts; attempt++ {
		response, sendErr := c.send(ctx, spec, requestURL, credentials)
		var beforeSend *preSendError
		if errors.As(sendErr, &beforeSend) {
			return nil, sendErr
		}
		if sendErr == nil {
			if response == nil {
				return nil, newTransportError(spec, ErrTransport)
			}
			if !shouldRetryStatus(response.StatusCode) {
				return response, nil
			}
		}
		retry := sendErr != nil && shouldRetryError(sendErr)
		if response != nil && shouldRetryStatus(response.StatusCode) {
			retry = true
		}
		if attempt == attempts || !retry {
			if sendErr != nil {
				return nil, newTransportError(spec, sendErr)
			}
			return response, nil
		}
		if response != nil {
			if _, drainErr := io.Copy(io.Discard, io.LimitReader(response.Body, maxRetryDrainBytes)); drainErr != nil {
				return nil, newTransportError(spec, drainErr)
			}
			if closeErr := response.Body.Close(); closeErr != nil {
				return nil, newTransportError(spec, closeErr)
			}
		}
		delay := retryDelay(response, c.retryPolicy.delay(attempt), c.clock.Now())
		if waitErr := waitForRetry(ctx, delay); waitErr != nil {
			return nil, waitErr
		}
	}
	return nil, fmt.Errorf("%w: retry loop exhausted", ErrTransport)
}

func retryDelay(response *http.Response, fallback time.Duration, now time.Time) time.Duration {
	if response == nil {
		return fallback
	}
	if retryAfter := retryAfterDuration(response.Header.Get("Retry-After"), now); retryAfter > 0 {
		return retryAfter
	}
	return fallback
}

//nolint:gocritic // requestSpec 是不可变的内部请求计划，按值传递可隔离调用方后续修改。
func (c *Client) send(
	ctx context.Context,
	spec requestSpec,
	requestURL *url.URL,
	credentials Credentials,
) (*http.Response, error) {
	var body io.ReadCloser
	contentLength := int64(0)
	if spec.body != nil {
		var err error
		body, contentLength, err = spec.body()
		if err != nil {
			return nil, &preSendError{cause: err}
		}
	}
	request, err := http.NewRequestWithContext(ctx, spec.method, requestURL.String(), body)
	if err != nil {
		if body != nil {
			return nil, &preSendError{cause: errorsJoin(err, body.Close())}
		}
		return nil, &preSendError{cause: err}
	}
	request.ContentLength = contentLength
	request.Header = spec.header.Clone()
	if request.Header == nil {
		request.Header = make(http.Header)
	}
	if spec.contentType != "" {
		request.Header.Set("Content-Type", spec.contentType)
	}
	request.Header.Set("Accept", "application/json")
	if spec.authenticated {
		applyAuthentication(request.Header, credentials, c.clock.Now().Unix())
	}
	response, err := c.httpClient.Do(request) //nolint:gosec // 目标仅由已校验 base URL、固定路径和编码后的查询参数构成。
	if response != nil && response.Request == nil {
		response.Request = request
	}
	return response, err
}

//nolint:gocritic // requestSpec 是不可变的内部请求计划，按值传递可隔离调用方后续修改。
func (c *Client) buildRequestURL(spec requestSpec) (*url.URL, error) {
	var base *url.URL
	switch spec.target {
	case apiTarget:
		base = c.apiBaseURL
	case storageTarget:
		base = c.storageBaseURL
	case absoluteTarget:
		parsed, err := url.Parse(spec.absoluteURL)
		if err != nil || parsed.Host == "" || parsed.User != nil {
			return nil, fmt.Errorf("%w: invalid absolute request URL", ErrInvalidRequest)
		}
		base = parsed
	default:
		return nil, fmt.Errorf("%w: unknown request target", ErrInvalidRequest)
	}
	if base == nil {
		return nil, fmt.Errorf("%w: request target is unavailable", ErrInvalidRequest)
	}
	clone := *base
	if spec.target != absoluteTarget {
		clone.Path = strings.TrimRight(base.Path, "/") + spec.path
	}
	if spec.query != nil {
		query, err := spec.query()
		if err != nil {
			return nil, err
		}
		clone.RawQuery = query.Encode()
	}
	return &clone, nil
}

func applyAuthentication(header http.Header, credentials Credentials, unixSeconds int64) {
	timestamp := strconv.FormatInt(unixSeconds, 10)
	inner := md5.Sum([]byte(timestamp))                                         //nolint:gosec // Mintegral 的签名协议固定使用 MD5。
	outer := md5.Sum([]byte(credentials.apiKey + hex.EncodeToString(inner[:]))) //nolint:gosec // Mintegral 的签名协议要求。
	header.Set("access-key", credentials.accessKey)
	header.Set("timestamp", timestamp)
	header.Set("token", hex.EncodeToString(outer[:]))
}

//nolint:gocritic // requestSpec 是不可变的内部请求计划，按值传递可隔离调用方后续修改。
func newTransportError(spec requestSpec, cause error) error {
	if errors.Is(cause, ErrInvalidRequest) {
		cause = ErrInvalidRequest
	} else if !errors.Is(cause, context.Canceled) && !errors.Is(cause, context.DeadlineExceeded) {
		cause = ErrTransport
	}
	return &transportError{operation: spec.operation, cause: cause, outcomeUnknown: spec.outcomeRisk}
}

func errorsJoin(first, second error) error {
	if second == nil {
		return first
	}
	return errors.Join(first, fmt.Errorf("close body: %w", second))
}
