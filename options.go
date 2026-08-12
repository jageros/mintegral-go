package mintegral

import (
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// Clock 提供请求签名使用的当前时间。
type Clock interface {
	Now() time.Time
}

// RetryPolicy 配置只读请求的有限重试。
type RetryPolicy struct {
	// MaxAttempts 是包含首次发送在内的最大物理请求次数。
	MaxAttempts int
	// InitialBackoff 是第一次重试前的等待时间。
	InitialBackoff time.Duration
	// MaxBackoff 是指数退避的最长等待时间。
	MaxBackoff time.Duration
}

// ClientOption 配置 Client。该接口由本包密封，只能使用 With 开头的构造函数。
type ClientOption interface {
	applyClient(*clientConfig) error
}

// RequestOption 只配置当前一次调用，不会修改 Client。
type RequestOption interface {
	applyRequest(*requestConfig) error
}

type clientOptionFunc func(*clientConfig) error

func (f clientOptionFunc) applyClient(config *clientConfig) error { return f(config) }

type requestOptionFunc func(*requestConfig) error

func (f requestOptionFunc) applyRequest(config *requestConfig) error { return f(config) }

// WithDefaultCredentials 设置请求未显式覆盖时使用的默认凭据。
func WithDefaultCredentials(credentials Credentials) ClientOption {
	return clientOptionFunc(func(config *clientConfig) error {
		if !credentials.valid() {
			return ErrInvalidCredentials
		}
		config.defaultCredentials = &credentials
		return nil
	})
}

// WithHTTPClient 注入 HTTP Client。SDK 会复制该值并禁止跨主机重定向。
func WithHTTPClient(client *http.Client) ClientOption {
	return clientOptionFunc(func(config *clientConfig) error {
		if client == nil {
			return fmt.Errorf("%w: HTTP client is nil", ErrInvalidRequest)
		}
		config.httpClient = client
		return nil
	})
}

// WithAPIBaseURL 覆盖 Mintegral 管理接口地址，主要用于本地契约测试。
func WithAPIBaseURL(rawURL string) ClientOption {
	return baseURLOption(rawURL, func(config *clientConfig, parsed *url.URL) { config.apiBaseURL = parsed })
}

// WithStorageBaseURL 覆盖 Mintegral 素材上传地址，主要用于本地契约测试。
func WithStorageBaseURL(rawURL string) ClientOption {
	return baseURLOption(rawURL, func(config *clientConfig, parsed *url.URL) { config.storageBaseURL = parsed })
}

// WithClock 注入签名时钟，适用于确定性测试。
func WithClock(clock Clock) ClientOption {
	return clientOptionFunc(func(config *clientConfig) error {
		if clock == nil {
			return fmt.Errorf("%w: clock is nil", ErrInvalidRequest)
		}
		config.clock = clock
		return nil
	})
}

// WithRetryPolicy 覆盖只读请求的最大尝试次数和退避时间。
func WithRetryPolicy(policy RetryPolicy) ClientOption {
	return clientOptionFunc(func(config *clientConfig) error {
		if err := validateRetryPolicy(policy); err != nil {
			return err
		}
		config.retryPolicy = policy
		return nil
	})
}

// WithRequestCredentials 为当前调用覆盖默认凭据。
func WithRequestCredentials(credentials Credentials) RequestOption {
	return requestOptionFunc(func(config *requestConfig) error {
		config.credentials = credentials
		config.credentialsSet = true
		return nil
	})
}

func baseURLOption(rawURL string, assign func(*clientConfig, *url.URL)) ClientOption {
	return clientOptionFunc(func(config *clientConfig) error {
		parsed, err := parseBaseURL(rawURL)
		if err != nil {
			return err
		}
		assign(config, parsed)
		return nil
	})
}
