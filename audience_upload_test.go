package mintegral

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestAudiencePresignUpload_authenticatesAndBuildsExpiringS3Plan(t *testing.T) {
	// Given
	now := time.Unix(1_700_000_000, 0)
	client, err := NewClient(
		WithAPIBaseURL("https://api.example.test"),
		WithClock(fixedClock{value: now}),
		WithHTTPClient(&http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			if request.Header.Get("access-key") == "" {
				t.Error("presign request lacks authentication")
			}
			if got := request.URL.RawQuery; got != "area_type=1&file_md5=900150983cd24fb0d6963f7d28e17f72&file_name=ids.csv" {
				t.Errorf("query = %q", got)
			}
			return jsonResponse(http.StatusOK, `{"code":200,"message":"success","data":{"area_type":1,"file_name":"ids.csv","file_md5":"900150983cd24fb0d6963f7d28e17f72","ttl":300,"s3":{"method":"PUT","url":"https://storage.example.test/upload?signature=x","data_path":"s3://bucket/ids.csv"}}}`), nil
		})}),
	)
	if err != nil {
		t.Fatal(err)
	}
	md5Value := mustContentMD5(t, "900150983cd24fb0d6963f7d28e17f72")

	// When
	plan, err := client.Audiences().PresignUpload(context.Background(), AudiencePresignRequest{AreaType: 1, FileName: "ids.csv", FileMD5: md5Value, FileSize: 3}, WithRequestCredentials(mustCredentials(t, "a", "b")))
	// Then
	if err != nil {
		t.Fatalf("PresignUpload() error = %v", err)
	}
	if plan.S3 == nil || plan.S3.DataPath != "s3://bucket/ids.csv" || !plan.ExpiresAt.Equal(now.Add(300*time.Second)) || plan.FileSize != 3 {
		t.Fatalf("plan = %#v", plan)
	}
}

func TestAudiencePresignUpload_usesEarlierOSSExpiryAndRedactsPlan(t *testing.T) {
	// Given
	now := time.Unix(1_700_000_000, 0)
	client, err := NewClient(
		WithAPIBaseURL("https://api.example.test"),
		WithClock(fixedClock{value: now}),
		WithHTTPClient(&http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return jsonResponse(http.StatusOK, `{"code":200,"msg":"success","data":{"ttl":300,"oss":{"method":"POST","accessid":"secret-access","host":"https://oss.example.test","expire":"1700000120","signature":"secret-signature","policy":"secret-policy","dir":"dir/","data_path":"oss://bucket/dir/ids.csv"}}}`), nil
		})}),
	)
	if err != nil {
		t.Fatal(err)
	}
	contentMD5, err := ParseContentMD5("900150983cd24fb0d6963f7d28e17f72")
	if err != nil {
		t.Fatal(err)
	}

	// When
	plan, err := client.Audiences().PresignUpload(context.Background(), AudiencePresignRequest{AreaType: 2, FileName: "ids.csv", FileMD5: contentMD5, FileSize: 3}, WithRequestCredentials(mustCredentials(t, "a", "b")))
	encoded, marshalErr := json.Marshal(plan)

	// Then
	if err != nil || marshalErr != nil {
		t.Fatalf("PresignUpload() error = %v, MarshalJSON() error = %v", err, marshalErr)
	}
	if !plan.ExpiresAt.Equal(time.Unix(1_700_000_120, 0)) {
		t.Fatalf("ExpiresAt = %v, want earlier OSS expiry", plan.ExpiresAt)
	}
	redacted := string(encoded) + plan.String() + plan.GoString() + plan.OSS.String() + plan.OSS.GoString()
	for _, secret := range []string{"secret-access", "secret-signature", "secret-policy", "oss.example.test"} {
		if strings.Contains(redacted, secret) {
			t.Fatalf("redacted representations contain %q", secret)
		}
	}
}

func TestAudienceUpload_sendsRawS3WithoutAuthenticationAndReplaysOnce(t *testing.T) {
	// Given
	var calls atomic.Int64
	client, err := NewClient(WithClock(fixedClock{value: time.Unix(100, 0)}), WithHTTPClient(&http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.Header.Get("access-key") != "" || request.Header.Get("token") != "" || request.Header.Get("timestamp") != "" {
			t.Error("storage request contains Mintegral authentication")
		}
		body, readErr := io.ReadAll(request.Body)
		if readErr != nil {
			t.Fatal(readErr)
		}
		if request.Method != http.MethodPut || string(body) != "abc" || request.Header.Get("Content-Type") != "" {
			t.Errorf("storage request = %s content-type=%q body=%q", request.Method, request.Header.Get("Content-Type"), body)
		}
		if calls.Add(1) == 1 {
			return jsonResponse(http.StatusServiceUnavailable, "retry"), nil
		}
		return jsonResponse(http.StatusOK, ""), nil
	})}))
	if err != nil {
		t.Fatal(err)
	}
	md5Value := mustContentMD5(t, "900150983cd24fb0d6963f7d28e17f72")
	source, err := NewUploadSource("ids.csv", 3, md5Value, func() (io.ReadCloser, error) { return io.NopCloser(strings.NewReader("abc")), nil })
	if err != nil {
		t.Fatal(err)
	}
	plan := AudienceUploadPlan{AreaType: 1, FileName: "ids.csv", FileSize: 3, FileMD5: md5Value, ExpiresAt: time.Unix(200, 0), S3: &AudienceS3Upload{Method: "PUT", URL: "https://storage.example.test/upload", DataPath: "s3://bucket/ids.csv"}}

	// When
	result, err := client.Audiences().Upload(context.Background(), plan, source, nil)

	// Then
	if err != nil || result.DataPath != "s3://bucket/ids.csv" || calls.Load() != 2 {
		t.Fatalf("Upload() = %#v, %v; calls=%d", result, err, calls.Load())
	}
}

func TestAudienceUpload_sendsOSSMultipartWithoutAuthentication(t *testing.T) {
	// Given
	client, err := NewClient(WithClock(fixedClock{value: time.Unix(100, 0)}), WithHTTPClient(&http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.Header.Get("access-key") != "" || request.Method != http.MethodPost {
			t.Error("OSS storage request contract mismatch")
		}
		reader, parseErr := request.MultipartReader()
		if parseErr != nil {
			t.Fatal(parseErr)
		}
		fields := readMultipart(t, reader)
		for key, want := range map[string]string{"key": "dir/ids.csv", "OSSAccessKeyId": "access", "policy": "policy", "signature": "signature", "success_action_status": "200", "file": "abc"} {
			if fields[key] != want {
				t.Errorf("multipart %s = %q, want %q", key, fields[key], want)
			}
		}
		return jsonResponse(http.StatusOK, ""), nil
	})}))
	if err != nil {
		t.Fatal(err)
	}
	md5Value := mustContentMD5(t, "900150983cd24fb0d6963f7d28e17f72")
	source := mustUploadSource(t, "ids.csv", md5Value, "abc")
	plan := AudienceUploadPlan{AreaType: 2, FileName: "ids.csv", FileSize: 3, FileMD5: md5Value, ExpiresAt: time.Unix(200, 0), OSS: &AudienceOSSUpload{Method: "POST", AccessID: "access", Host: "https://oss.example.test", Signature: "signature", Policy: "policy", Directory: "dir/", DataPath: "oss://bucket/dir/ids.csv"}}

	// When
	result, err := client.Audiences().Upload(context.Background(), plan, source)

	// Then
	if err != nil || result.DataPath != "oss://bucket/dir/ids.csv" {
		t.Fatalf("Upload() = %#v, %v", result, err)
	}
}

func TestAudienceUpload_rejectsExpiredOrChangedSourceBeforeSending(t *testing.T) {
	// Given
	var calls atomic.Int64
	client := mustAudienceUploadClient(t, time.Unix(200, 0), func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, errors.New("unexpected") })
	md5Value := mustContentMD5(t, "900150983cd24fb0d6963f7d28e17f72")
	otherMD5 := mustContentMD5(t, "d16fb36f0911f878998c136191af705e")
	source := mustUploadSource(t, "changed.csv", otherMD5, "xyz")
	plan := AudienceUploadPlan{AreaType: 1, FileName: "ids.csv", FileSize: 3, FileMD5: md5Value, ExpiresAt: time.Unix(200, 0), S3: &AudienceS3Upload{Method: "PUT", URL: "https://storage.example.test", DataPath: "s3://bucket/ids.csv"}}

	// When
	_, err := client.Audiences().Upload(context.Background(), plan, source)

	// Then
	if !errors.Is(err, ErrUploadExpired) || calls.Load() != 0 {
		t.Fatalf("Upload() error = %v, calls=%d", err, calls.Load())
	}
}

func TestAudienceUpload_rejectsChangedSourceBeforeSending(t *testing.T) {
	// Given
	var calls atomic.Int64
	client := mustAudienceUploadClient(t, time.Unix(100, 0), func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, errors.New("unexpected") })
	md5Value := mustContentMD5(t, "900150983cd24fb0d6963f7d28e17f72")
	otherMD5 := mustContentMD5(t, "d16fb36f0911f878998c136191af705e")
	source := mustUploadSource(t, "changed.csv", otherMD5, "xyz")
	plan := AudienceUploadPlan{AreaType: 1, FileName: "ids.csv", FileSize: 3, FileMD5: md5Value, ExpiresAt: time.Unix(200, 0), S3: &AudienceS3Upload{Method: "PUT", URL: "https://storage.example.test", DataPath: "s3://bucket/ids.csv"}}

	// When
	_, err := client.Audiences().Upload(context.Background(), plan, source)

	// Then
	if !errors.Is(err, ErrInvalidRequest) || calls.Load() != 0 {
		t.Fatalf("Upload() error = %v, calls=%d", err, calls.Load())
	}
}

func TestAudienceUpload_rejectsChangedContentWithoutReplay(t *testing.T) {
	// Given
	var calls atomic.Int64
	client := mustAudienceUploadClient(t, time.Unix(100, 0), func(request *http.Request) (*http.Response, error) {
		calls.Add(1)
		_, err := io.ReadAll(request.Body)
		if err != nil {
			return nil, err
		}
		return jsonResponse(http.StatusOK, ""), nil
	})
	declaredMD5 := mustContentMD5(t, "900150983cd24fb0d6963f7d28e17f72")
	source := mustUploadSource(t, "ids.csv", declaredMD5, "xyz")
	plan := AudienceUploadPlan{AreaType: 1, FileName: "ids.csv", FileSize: 3, FileMD5: declaredMD5, ExpiresAt: time.Unix(200, 0), S3: &AudienceS3Upload{Method: "PUT", URL: "https://storage.example.test", DataPath: "s3://bucket/ids.csv"}}

	// When
	_, err := client.Audiences().Upload(context.Background(), plan, source)

	// Then
	if !errors.Is(err, ErrInvalidRequest) || calls.Load() != 0 {
		t.Fatalf("Upload() error = %v, calls=%d", err, calls.Load())
	}
}

func readMultipart(t *testing.T, reader *multipart.Reader) map[string]string {
	t.Helper()
	fields := make(map[string]string)
	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			return fields
		}
		if err != nil {
			t.Fatal(err)
		}
		value, readErr := io.ReadAll(part)
		if readErr != nil {
			t.Fatal(readErr)
		}
		fields[part.FormName()] = string(value)
	}
}

func mustContentMD5(t *testing.T, raw string) ContentMD5 {
	t.Helper()
	value, err := ParseContentMD5(raw)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func mustUploadSource(t *testing.T, name string, contentMD5 ContentMD5, content string) UploadSource {
	t.Helper()
	source, err := NewUploadSource(name, int64(len(content)), contentMD5, func() (io.ReadCloser, error) { return io.NopCloser(strings.NewReader(content)), nil })
	if err != nil {
		t.Fatal(err)
	}
	return source
}

func mustAudienceUploadClient(t *testing.T, now time.Time, transport roundTripFunc) *Client {
	t.Helper()
	client, err := NewClient(WithClock(fixedClock{value: now}), WithHTTPClient(&http.Client{Transport: transport}))
	if err != nil {
		t.Fatal(err)
	}
	return client
}
