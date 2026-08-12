package mintegral_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	mintegral "github.com/jageros/mintegral-go"
)

func manualQASource(t *testing.T) (mintegral.UploadSource, mintegral.ContentMD5) {
	t.Helper()
	data := []byte("abc")
	checksum := mintegral.ContentMD5FromBytes(data)
	source, err := mintegral.NewUploadSource("ids.csv", int64(len(data)), checksum, func() (io.ReadCloser, error) {
		return io.NopCloser(strings.NewReader(string(data))), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return source, checksum
}

func TestManualQAAudienceS3RawPUT(t *testing.T) {
	now := time.Unix(100, 0)
	credentials := manualQACredentials(t, "upload-access", "upload-api")
	source, checksum := manualQASource(t)
	var storageCalls atomic.Int64
	storage := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		storageCalls.Add(1)
		if request.Method != http.MethodPut || request.Header.Get("access-key") != "" || request.Header.Get("token") != "" || request.Header.Get("timestamp") != "" || request.Header.Get("Content-Type") != "" {
			t.Errorf("S3 request = method=%s access=%q token=%q timestamp=%q content-type=%q", request.Method, request.Header.Get("access-key"), request.Header.Get("token"), request.Header.Get("timestamp"), request.Header.Get("Content-Type"))
		}
		body, err := io.ReadAll(request.Body)
		if err != nil || string(body) != "abc" {
			t.Errorf("S3 body = %q, err=%v", body, err)
		}
		response.WriteHeader(http.StatusOK)
	}))
	defer storage.Close()
	api := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		manualQAAuth(t, request, "upload-api", now)
		if request.Method != http.MethodGet || request.URL.Path != "/api/open/v1/audience/presigned-upload-data" || request.URL.Query().Get("area_type") != "1" || request.URL.Query().Get("file_name") != "ids.csv" || request.URL.Query().Get("file_md5") != checksum.String() {
			t.Errorf("presign request = %s %s?%s", request.Method, request.URL.Path, request.URL.RawQuery)
		}
		manualQAWrite(t, response, `{"code":200,"data":{"area_type":1,"file_name":"ids.csv","file_md5":"900150983cd24fb0d6963f7d28e17f72","ttl":300,"s3":{"method":"PUT","url":"`+storage.URL+`","data_path":"s3://bucket/ids.csv"}}}`)
	}))
	defer api.Close()
	client, err := mintegral.NewClient(mintegral.WithAPIBaseURL(api.URL), mintegral.WithClock(manualQAFixedClock{value: now}), mintegral.WithHTTPClient(api.Client()))
	if err != nil {
		t.Fatal(err)
	}
	result, err := client.Audiences().UploadFile(context.Background(), mintegral.AudiencePresignRequest{AreaType: 1, FileName: "ids.csv", FileMD5: checksum, FileSize: 3}, source, mintegral.WithRequestCredentials(credentials))
	if err != nil || result.DataPath != "s3://bucket/ids.csv" {
		t.Fatalf("UploadFile(S3) = %#v, %v", result, err)
	}
	if storageCalls.Load() != 1 {
		t.Fatalf("S3 storage calls = %d, want 1", storageCalls.Load())
	}
}

func TestManualQAAudienceOSSMultipartPOST(t *testing.T) {
	now := time.Unix(100, 0)
	credentials := manualQACredentials(t, "upload-access", "upload-api")
	source, checksum := manualQASource(t)
	var storageCalls atomic.Int64
	storage := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		storageCalls.Add(1)
		if request.Method != http.MethodPost || !strings.HasPrefix(request.Header.Get("Content-Type"), "multipart/form-data;") || request.Header.Get("access-key") != "" || request.Header.Get("token") != "" || request.Header.Get("timestamp") != "" {
			t.Errorf("OSS request = method=%s content-type=%q access=%q token=%q timestamp=%q", request.Method, request.Header.Get("Content-Type"), request.Header.Get("access-key"), request.Header.Get("token"), request.Header.Get("timestamp"))
		}
		reader, err := request.MultipartReader()
		if err != nil {
			t.Errorf("multipart reader = %v", err)
			return
		}
		fields := make(map[string]string)
		for {
			part, nextErr := reader.NextPart()
			if nextErr == io.EOF {
				break
			}
			if nextErr != nil {
				t.Errorf("multipart part = %v", nextErr)
				return
			}
			value, readErr := io.ReadAll(part)
			if readErr != nil {
				t.Errorf("multipart value = %v", readErr)
			}
			fields[part.FormName()] = string(value)
		}
		for name, want := range map[string]string{"key": "dir/ids.csv", "OSSAccessKeyId": "access", "policy": "policy", "signature": "signature", "success_action_status": "200", "file": "abc"} {
			if fields[name] != want {
				t.Errorf("OSS field %s = %q, want %q", name, fields[name], want)
			}
		}
		response.WriteHeader(http.StatusOK)
	}))
	defer storage.Close()
	api := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		manualQAAuth(t, request, "upload-api", now)
		if request.URL.Query().Get("area_type") != "2" || request.URL.Query().Get("file_md5") != checksum.String() {
			t.Errorf("presign query = %s", request.URL.RawQuery)
		}
		manualQAWrite(t, response, `{"code":200,"data":{"area_type":2,"file_name":"ids.csv","file_md5":"900150983cd24fb0d6963f7d28e17f72","ttl":300,"oss":{"method":"POST","accessid":"access","host":"`+storage.URL+`","policy":"policy","signature":"signature","dir":"dir/","data_path":"oss://bucket/dir/ids.csv"}}}`)
	}))
	defer api.Close()
	client, err := mintegral.NewClient(mintegral.WithAPIBaseURL(api.URL), mintegral.WithClock(manualQAFixedClock{value: now}), mintegral.WithHTTPClient(api.Client()))
	if err != nil {
		t.Fatal(err)
	}
	result, err := client.Audiences().UploadFile(context.Background(), mintegral.AudiencePresignRequest{AreaType: 2, FileName: "ids.csv", FileMD5: checksum, FileSize: 3}, source, mintegral.WithRequestCredentials(credentials))
	if err != nil || result.DataPath != "oss://bucket/dir/ids.csv" {
		t.Fatalf("UploadFile(OSS) = %#v, %v", result, err)
	}
	if storageCalls.Load() != 1 {
		t.Fatalf("OSS storage calls = %d, want 1", storageCalls.Load())
	}
}
