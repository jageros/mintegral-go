package mintegral

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestAudienceUpload_replayUsesVerifiedByteSnapshot(t *testing.T) {
	var networkCalls atomic.Int64
	var sourceOpens atomic.Int64
	client, err := NewClient(
		WithClock(fixedClock{value: time.Unix(100, 0)}),
		WithHTTPClient(&http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			if networkCalls.Add(1) == 1 {
				prefix := make([]byte, 1)
				if _, readErr := io.ReadFull(request.Body, prefix); readErr != nil || string(prefix) != "a" {
					t.Fatalf("first request prefix = %q, %v", prefix, readErr)
				}
				return nil, timeoutNetworkError{}
			}
			body, readErr := io.ReadAll(request.Body)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if string(body) != "abc" {
				t.Fatalf("replayed body = %q, want verified snapshot", body)
			}
			return jsonResponse(http.StatusOK, ""), nil
		})}),
	)
	if err != nil {
		t.Fatal(err)
	}
	contentMD5 := mustContentMD5(t, "900150983cd24fb0d6963f7d28e17f72")
	source, err := NewUploadSource("ids.csv", 3, contentMD5, func() (io.ReadCloser, error) {
		if sourceOpens.Add(1) == 1 {
			return io.NopCloser(strings.NewReader("abc")), nil
		}
		return io.NopCloser(strings.NewReader("xyz")), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	plan := AudienceUploadPlan{AreaType: 1, FileName: "ids.csv", FileSize: 3, FileMD5: contentMD5, ExpiresAt: time.Unix(200, 0), S3: &AudienceS3Upload{Method: http.MethodPut, URL: "https://storage.example.test/upload", DataPath: "s3://bucket/ids.csv"}}

	result, err := client.Audiences().Upload(context.Background(), plan, source)

	if err != nil || result.DataPath != plan.S3.DataPath {
		t.Fatalf("Upload() = %#v, %v", result, err)
	}
	if networkCalls.Load() != 2 || sourceOpens.Load() != 1 {
		t.Fatalf("network calls = %d, source opens = %d; want 2 and 1", networkCalls.Load(), sourceOpens.Load())
	}
}
