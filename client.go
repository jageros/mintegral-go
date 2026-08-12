package mintegral

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	defaultAPIBaseURL     = "https://ss-api.mintegral.com"
	defaultStorageBaseURL = "https://ss-storage-api.mintegral.com"
)

// Client 是可并发复用的 Mintegral API 客户端。
type Client struct {
	httpClient         *http.Client
	apiBaseURL         *url.URL
	storageBaseURL     *url.URL
	clock              Clock
	retryPolicy        RetryPolicy
	defaultCredentials *Credentials
}

type clientConfig Client

type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now() }

// NewClient 创建 Client。默认凭据完全可选，缺少凭据只会在鉴权请求执行时返回错误。
func NewClient(options ...ClientOption) (*Client, error) {
	apiBaseURL, err := parseBaseURL(defaultAPIBaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse default API base URL: %w", err)
	}
	storageBaseURL, err := parseBaseURL(defaultStorageBaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse default storage base URL: %w", err)
	}
	config := clientConfig{
		httpClient:     defaultHTTPClient(),
		apiBaseURL:     apiBaseURL,
		storageBaseURL: storageBaseURL,
		clock:          systemClock{},
		retryPolicy: RetryPolicy{
			MaxAttempts:    3,
			InitialBackoff: 250 * time.Millisecond,
			MaxBackoff:     2 * time.Second,
		},
	}
	for _, option := range options {
		if option == nil {
			return nil, fmt.Errorf("%w: nil client option", ErrInvalidRequest)
		}
		if err := option.applyClient(&config); err != nil {
			return nil, err
		}
	}
	config.httpClient = cloneHTTPClient(config.httpClient)
	client := Client(config)
	return &client, nil
}

func parseBaseURL(rawURL string) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, fmt.Errorf("%w: invalid base URL", ErrInvalidRequest)
	}
	if parsed.Scheme != "https" && (parsed.Scheme != "http" || !isLoopbackHost(parsed.Hostname())) {
		return nil, fmt.Errorf("%w: base URL must use HTTPS or loopback HTTP", ErrInvalidRequest)
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	return parsed, nil
}

func isLoopbackHost(host string) bool {
	return host == "localhost" || net.ParseIP(host).IsLoopback()
}

// Accounts 返回与 Client 共享传输和凭据配置的账户服务。
func (c *Client) Accounts() *AccountService { return &AccountService{client: c} }

// Campaigns 返回与 Client 共享传输和凭据配置的广告活动服务。
func (c *Client) Campaigns() *CampaignService { return &CampaignService{client: c} }

// Apps 返回与 Client 共享传输和凭据配置的应用名称服务。
func (c *Client) Apps() *AppService { return &AppService{client: c} }
