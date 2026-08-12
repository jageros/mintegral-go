package mintegral

import (
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestDate_ParseDate_normalizesOnlyCalendarDates(t *testing.T) {
	// Given
	valid := "2026-08-12"
	invalid := "2026-8-12"

	// When
	date, err := ParseDate(valid)
	_, invalidErr := ParseDate(invalid)

	// Then
	if err != nil {
		t.Fatalf("ParseDate(%q) error = %v", valid, err)
	}
	if got := string(date); got != valid {
		t.Fatalf("ParseDate(%q) = %q, want %q", valid, got, valid)
	}
	if !errors.Is(invalidErr, ErrInvalidRequest) {
		t.Fatalf("ParseDate(%q) error = %v, want ErrInvalidRequest", invalid, invalidErr)
	}
}

func TestDate_DateFromTime_usesUTCDate(t *testing.T) {
	// Given
	instant := time.Date(2026, time.August, 11, 23, 30, 0, 0, time.FixedZone("UTC-1", -3600))

	// When
	date := DateFromTime(instant)

	// Then
	if got := string(date); got != "2026-08-12" {
		t.Fatalf("DateFromTime() = %q, want UTC date 2026-08-12", got)
	}
}

func TestDate_JSON_requiresCanonicalDate(t *testing.T) {
	// Given
	var date Date

	// When
	err := json.Unmarshal([]byte(`"2026-02-29"`), &date)

	// Then
	if !errors.Is(err, ErrUnexpectedResponse) {
		t.Fatalf("json.Unmarshal() error = %v, want ErrUnexpectedResponse", err)
	}
}

func TestDecimalText_ParseDecimalText_preservesExactRepresentation(t *testing.T) {
	// Given
	raw := "12345678901234567890.12345678901234567890"

	// When
	decimal, err := ParseDecimalText(raw)
	// Then
	if err != nil {
		t.Fatalf("ParseDecimalText(%q) error = %v", raw, err)
	}
	if got := decimal.String(); got != raw {
		t.Fatalf("ParseDecimalText(%q) = %q, want exact input", raw, got)
	}
}

func TestDecimalText_UnmarshalJSON_acceptsNumberAndString(t *testing.T) {
	// Given
	var fromNumber DecimalText
	var fromString DecimalText

	// When
	numberErr := json.Unmarshal([]byte(`1.20e+03`), &fromNumber)
	stringErr := json.Unmarshal([]byte(`"1.20e+03"`), &fromString)

	// Then
	if numberErr != nil {
		t.Fatalf("unmarshal number error = %v", numberErr)
	}
	if stringErr != nil {
		t.Fatalf("unmarshal string error = %v", stringErr)
	}
	if got := string(fromNumber); got != "1.20e+03" {
		t.Fatalf("number decimal = %q, want exact representation", got)
	}
	if got := string(fromString); got != "1.20e+03" {
		t.Fatalf("string decimal = %q, want exact representation", got)
	}
}

func TestDecimalText_MarshalJSON_encodesAsNumber(t *testing.T) {
	// Given
	decimal, err := ParseDecimalText("0.0100")
	if err != nil {
		t.Fatalf("ParseDecimalText() error = %v", err)
	}

	// When
	encoded, err := json.Marshal(decimal)
	// Then
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if got := string(encoded); got != "0.0100" {
		t.Fatalf("json.Marshal() = %s, want JSON number 0.0100", encoded)
	}
}

func TestDecimalText_ParseDecimalText_rejectsInvalidJSONNumber(t *testing.T) {
	tests := []string{"01.2", "null", `"1.2"`, "true"}
	for _, raw := range tests {
		t.Run(raw, func(t *testing.T) {
			// When
			_, err := ParseDecimalText(raw)

			// Then
			if !errors.Is(err, ErrInvalidRequest) {
				t.Fatalf("ParseDecimalText(%q) error = %v, want ErrInvalidRequest", raw, err)
			}
		})
	}
}

func TestContentMD5_ParseContentMD5_acceptsLowercaseHexDigest(t *testing.T) {
	// Given
	digest := "d41d8cd98f00b204e9800998ecf8427e"

	// When
	parsed, err := ParseContentMD5(digest)
	// Then
	if err != nil {
		t.Fatalf("ParseContentMD5() error = %v", err)
	}
	if got := parsed.String(); got != digest {
		t.Fatalf("ContentMD5.String() = %q, want %q", got, digest)
	}
}

func TestContentMD5FromBytes_returnsLowercaseHexDigest(t *testing.T) {
	// Given
	content := []byte("hello")

	// When
	digest := ContentMD5FromBytes(content)

	// Then
	if got := digest.String(); got != "5d41402abc4b2a76b9719d911017c592" {
		t.Fatalf("ContentMD5FromBytes() = %q, want MD5 hex digest", got)
	}
}
