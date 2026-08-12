package mintegral

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Credentials 保存 Mintegral 的 Access Key 和 API Key。
//
// 字段不可导出，格式化时始终脱敏；零值只能用于表示没有凭据。
type Credentials struct {
	accessKey string
	apiKey    string
}

// NewCredentials 校验并创建凭据。accessKey 和 apiKey 去除首尾空白后都必须非空。
func NewCredentials(accessKey, apiKey string) (Credentials, error) {
	accessKey = strings.TrimSpace(accessKey)
	apiKey = strings.TrimSpace(apiKey)
	if accessKey == "" || apiKey == "" {
		return Credentials{}, fmt.Errorf("%w: access key and API key must both be non-empty", ErrInvalidCredentials)
	}
	return Credentials{accessKey: accessKey, apiKey: apiKey}, nil
}

// String 返回固定脱敏文本。
func (Credentials) String() string { return "<redacted>" }

// GoString 返回固定脱敏文本，避免 %#v 泄漏凭据。
func (Credentials) GoString() string { return "mintegral.Credentials(<redacted>)" }

// MarshalJSON 返回固定脱敏文本，避免凭据进入 JSON 日志或载荷。
func (Credentials) MarshalJSON() ([]byte, error) { return json.Marshal("<redacted>") }

func (c Credentials) valid() bool {
	return c.accessKey != "" && c.apiKey != ""
}
