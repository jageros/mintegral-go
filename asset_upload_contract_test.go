package mintegral

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestAssetsUploadMedia_whenCalled_streamsAuthenticatedMultipart(t *testing.T) {
	client := contractClient(t, func(request *http.Request) *http.Response {
		if request.URL.Path != "/api/open/v1/creatives/upload" || request.Header.Get("access-key") == "" {
			t.Fatalf("path/auth = %q %q", request.URL.Path, request.Header.Get("access-key"))
		}
		reader, err := request.MultipartReader()
		if err != nil {
			t.Fatal(err)
		}
		part, err := reader.NextPart()
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(part)
		if err != nil {
			t.Fatal(err)
		}
		if part.FormName() != "file" || part.FileName() != "ad.png" || string(data) != "payload" {
			t.Fatalf("multipart = %q %q %q", part.FormName(), part.FileName(), data)
		}
		if _, err = reader.NextPart(); !errors.Is(err, io.EOF) {
			t.Fatalf("second part = %v", err)
		}
		return jsonResponse(200, `{"code":200,"data":{"creative_md5":"321c3cf486ed509164edec1e1981fec8","creative_name":"ad.png"}}`)
	})
	source, err := NewUploadSource("ad.png", 7, ContentMD5("321c3cf486ed509164edec1e1981fec8"), func() (io.ReadCloser, error) { return io.NopCloser(strings.NewReader("payload")), nil })
	if err != nil {
		t.Fatal(err)
	}
	result, err := client.Assets().UploadMedia(context.Background(), source)
	if err != nil || result.CreativeName != "ad.png" {
		t.Fatalf("UploadMedia() = %#v, %v", result, err)
	}
}

func TestAssetsUploadPlayable_whenCalled_usesPlayableStoragePath(t *testing.T) {
	client := contractClient(t, func(request *http.Request) *http.Response {
		if request.Method != http.MethodPost || request.URL.Host != "storage.example.test" || request.URL.Path != "/api/open/v1/playable/upload" {
			t.Fatalf("request = %s %s", request.Method, request.URL.String())
		}
		reader, err := request.MultipartReader()
		if err != nil {
			t.Fatal(err)
		}
		part, err := reader.NextPart()
		if err != nil || part.FormName() != "file" {
			t.Fatalf("part = %#v, error = %v", part, err)
		}
		return jsonResponse(200, `{"code":200,"data":{"creative_md5":"900150983cd24fb0d6963f7d28e17f72","creative_name":"x.zip"}}`)
	})
	source, err := NewUploadSource("x.zip", 3, ContentMD5("900150983cd24fb0d6963f7d28e17f72"), func() (io.ReadCloser, error) { return io.NopCloser(strings.NewReader("abc")), nil })
	if err != nil {
		t.Fatal(err)
	}
	if _, err = client.Assets().UploadPlayable(context.Background(), source); err != nil {
		t.Fatalf("UploadPlayable() error = %v", err)
	}
}

func TestUploadSource_whenOpenedTwice_returnsFreshContent(t *testing.T) {
	source, err := NewUploadSource("x.zip", 3, ContentMD5("900150983cd24fb0d6963f7d28e17f72"), func() (io.ReadCloser, error) { return io.NopCloser(strings.NewReader("abc")), nil })
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		reader, openErr := source.Open()
		if openErr != nil {
			t.Fatal(openErr)
		}
		data, readErr := io.ReadAll(reader)
		if readErr != nil {
			t.Fatal(readErr)
		}
		if closeErr := reader.Close(); closeErr != nil {
			t.Fatal(closeErr)
		}
		if string(data) != "abc" {
			t.Fatalf("data = %q", data)
		}
	}
}

func TestUploadSource_whenContentDiffers_rejectsStream(t *testing.T) {
	source, err := NewUploadSource("x.zip", 3, ContentMD5("900150983cd24fb0d6963f7d28e17f72"), func() (io.ReadCloser, error) {
		return io.NopCloser(strings.NewReader("abd")), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	reader, err := source.Open()
	if err != nil {
		t.Fatal(err)
	}
	_, err = io.ReadAll(reader)
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("ReadAll() error = %v, want ErrInvalidRequest", err)
	}
	if closeErr := reader.Close(); closeErr != nil {
		t.Fatalf("Close() error = %v", closeErr)
	}
}

func TestMultipartBody_whenConsumerCloses_closesBlockedSource(t *testing.T) {
	blocked := newBlockingUploadReader()
	source, err := NewUploadSource("x.zip", 3, ContentMD5("900150983cd24fb0d6963f7d28e17f72"), func() (io.ReadCloser, error) { return blocked, nil })
	if err != nil {
		t.Fatal(err)
	}
	factory, _, err := multipartFileBody(source)
	if err != nil {
		t.Fatal(err)
	}
	body, _, err := factory()
	if err != nil {
		t.Fatal(err)
	}
	buffer := make([]byte, 1024)
	if _, err = body.Read(buffer); err != nil {
		t.Fatal(err)
	}
	if err = body.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-blocked.closed:
	case <-time.After(time.Second):
		t.Fatal("blocked source was not closed")
	}
}

type blockingUploadReader struct {
	closed chan struct{}
	once   sync.Once
}

func newBlockingUploadReader() *blockingUploadReader {
	return &blockingUploadReader{closed: make(chan struct{})}
}

func (reader *blockingUploadReader) Read([]byte) (int, error) {
	<-reader.closed
	return 0, errors.New("closed")
}

func (reader *blockingUploadReader) Close() error {
	reader.once.Do(func() { close(reader.closed) })
	return nil
}
