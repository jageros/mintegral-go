package mintegral

import (
	//nolint:gosec // Mintegral 人群上传协议要求使用 MD5 内容摘要。
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

// CampaignID 是广告活动的唯一标识。
type CampaignID int64

// OfferID 是推广商品的唯一标识。
type OfferID int64

// CreativeSetID 是创意组的唯一标识。
type CreativeSetID int64

// CreativeID 是广告创意的唯一标识。
type CreativeID int64

// AudienceID 是人群包的唯一标识。
type AudienceID int64

// AdID 是广告的唯一标识。
type AdID int64

// UserID 是 Mintegral 用户的唯一标识。
type UserID int64

// UnixSeconds 是 Unix 时间戳的秒数。
type UnixSeconds int64

// Date 是 UTC 日历日期，格式固定为 YYYY-MM-DD。
type Date string

// ParseDate 校验并解析 YYYY-MM-DD 格式的 UTC 日历日期。
func ParseDate(value string) (Date, error) {
	parsed, err := time.Parse(time.DateOnly, value)
	if err != nil || parsed.Format(time.DateOnly) != value {
		return "", fmt.Errorf("%w: date must use YYYY-MM-DD", ErrInvalidRequest)
	}
	return Date(value), nil
}

// DateFromTime 将时刻转换为对应的 UTC 日历日期。
func DateFromTime(value time.Time) Date {
	return Date(value.UTC().Format(time.DateOnly))
}

// String 返回 YYYY-MM-DD 格式的日期文本。
func (value Date) String() string { return string(value) }

// MarshalJSON 将日期编码为 YYYY-MM-DD 格式的 JSON 字符串。
func (value Date) MarshalJSON() ([]byte, error) {
	date, err := ParseDate(string(value))
	if err != nil {
		return nil, err
	}
	return json.Marshal(string(date))
}

// UnmarshalJSON 从 JSON 字符串解析 YYYY-MM-DD 格式的日期。
func (value *Date) UnmarshalJSON(data []byte) error {
	if value == nil {
		return fmt.Errorf("%w: date destination is nil", ErrUnexpectedResponse)
	}
	var text string
	if err := json.Unmarshal(data, &text); err != nil {
		return fmt.Errorf("%w: date must be a JSON string", ErrUnexpectedResponse)
	}
	date, err := ParseDate(text)
	if err != nil {
		return fmt.Errorf("%w: date must use YYYY-MM-DD", ErrUnexpectedResponse)
	}
	*value = date
	return nil
}

// DecimalText 是精确保留文本表示的十进制数。
type DecimalText string

// ParseDecimalText 校验 JSON 十进制数，并保留其原始文本表示。
func ParseDecimalText(value string) (DecimalText, error) {
	if value == "" || (value[0] != '-' && (value[0] < '0' || value[0] > '9')) {
		return "", fmt.Errorf("%w: decimal must be a JSON number", ErrInvalidRequest)
	}
	var number json.Number
	if err := json.Unmarshal([]byte(value), &number); err != nil {
		return "", fmt.Errorf("%w: decimal must be a JSON number", ErrInvalidRequest)
	}
	if number.String() != value {
		return "", fmt.Errorf("%w: decimal must be a JSON number", ErrInvalidRequest)
	}
	return DecimalText(number.String()), nil
}

// String 返回精确保留的十进制文本。
func (value DecimalText) String() string { return string(value) }

// MarshalJSON 将十进制数编码为 JSON number，而不是字符串。
func (value DecimalText) MarshalJSON() ([]byte, error) {
	decimal, err := ParseDecimalText(string(value))
	if err != nil {
		return nil, err
	}
	return []byte(decimal), nil
}

// UnmarshalJSON 从 JSON number 或字符串解析精确十进制数。
func (value *DecimalText) UnmarshalJSON(data []byte) error {
	if value == nil {
		return fmt.Errorf("%w: decimal destination is nil", ErrUnexpectedResponse)
	}
	decimal, err := parseDecimalJSON(data)
	if err != nil {
		return err
	}
	*value = decimal
	return nil
}

func parseDecimalJSON(data []byte) (DecimalText, error) {
	decimal, err := ParseDecimalText(string(data))
	if err == nil {
		return decimal, nil
	}
	var text string
	if json.Unmarshal(data, &text) == nil {
		decimal, err = ParseDecimalText(text)
		if err == nil {
			return decimal, nil
		}
	}
	return "", fmt.Errorf("%w: decimal must be a JSON number or decimal string", ErrUnexpectedResponse)
}

// ContentMD5 是 32 位小写十六进制 MD5 内容摘要。
type ContentMD5 string

// ParseContentMD5 校验并解析 32 位小写十六进制 MD5 内容摘要。
func ParseContentMD5(value string) (ContentMD5, error) {
	if len(value) != md5.Size*2 {
		return "", fmt.Errorf("%w: content MD5 must contain 32 lowercase hexadecimal characters", ErrInvalidRequest)
	}
	decoded, err := hex.DecodeString(value)
	if err != nil || hex.EncodeToString(decoded) != value {
		return "", fmt.Errorf("%w: content MD5 must contain 32 lowercase hexadecimal characters", ErrInvalidRequest)
	}
	return ContentMD5(value), nil
}

// ContentMD5FromBytes 计算内容的 32 位小写十六进制 MD5 摘要。
func ContentMD5FromBytes(value []byte) ContentMD5 {
	sum := md5.Sum(value) //nolint:gosec // Mintegral 人群上传协议要求 MD5 内容摘要。
	return ContentMD5(hex.EncodeToString(sum[:]))
}

// String 返回 32 位小写十六进制 MD5 内容摘要。
func (value ContentMD5) String() string { return string(value) }

// CountryCode 是 ISO 3166-1 alpha-2 国家或地区代码。
type CountryCode string

// String 返回国家或地区代码文本。
func (value CountryCode) String() string { return string(value) }
