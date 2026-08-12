package mintegral

import (
	"bytes"
	"context"
	"crypto/md5" //nolint:gosec // Mintegral 人群上传协议要求校验 MD5 内容摘要。
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type audiencePresignData struct {
	S3        *audiencePresignStorage `json:"s3"`
	OSS       *audiencePresignStorage `json:"oss"`
	FileName  string                  `json:"file_name"`
	FileMD5   ContentMD5              `json:"file_md5"`
	Method    string                  `json:"method"`
	AccessID  string                  `json:"accessid"`
	Host      string                  `json:"host"`
	Expire    string                  `json:"expire"`
	Signature string                  `json:"signature"`
	Policy    string                  `json:"policy"`
	Directory string                  `json:"dir"`
	DataPath  string                  `json:"data_path"`
	TTL       int64                   `json:"ttl"`
	AreaType  int                     `json:"area_type"`
}

type audiencePresignStorage struct {
	Method    string `json:"method"`
	URL       string `json:"url"`
	AccessID  string `json:"accessid"`
	Host      string `json:"host"`
	Expire    string `json:"expire"`
	Signature string `json:"signature"`
	Policy    string `json:"policy"`
	Directory string `json:"dir"`
	DataPath  string `json:"data_path"`
}

// PresignUpload 获取与指定文件元数据绑定的限时上传计划。
func (s *AudienceService) PresignUpload(ctx context.Context, request AudiencePresignRequest, options ...RequestOption) (AudienceUploadPlan, error) {
	if err := validatePresignRequest(request); err != nil {
		return AudienceUploadPlan{}, err
	}
	spec := requestSpec{
		operation: "audience.presign_upload", method: http.MethodGet,
		path: audiencePath + "/presigned-upload-data", authenticated: true, retryable: true,
		query: func() (url.Values, error) {
			values := make(url.Values)
			values.Set("area_type", strconv.Itoa(request.AreaType))
			values.Set("file_name", request.FileName)
			values.Set("file_md5", request.FileMD5.String())
			return values, nil
		},
	}
	data, err := doJSON[audiencePresignData](ctx, s.client, spec, options)
	if err != nil {
		return AudienceUploadPlan{}, err
	}
	return s.parseUploadPlan(request, &data)
}

// Upload 按预签名计划上传文件；options 为保持统一签名而接收，但不会解析或用于认证。
// 为保证重放字节完全一致，发送前会流式创建与源文件等大的本地临时快照。
func (s *AudienceService) Upload(ctx context.Context, plan AudienceUploadPlan, source UploadSource, _ ...RequestOption) (result AudienceUploadResult, err error) { //nolint:gocritic // 公共契约按值接收不可变上传计划。
	if !s.client.clock.Now().Before(plan.ExpiresAt) {
		return AudienceUploadResult{}, ErrUploadExpired
	}
	if source.Filename() != plan.FileName || source.Size() != plan.FileSize || source.ContentMD5() != plan.FileMD5 {
		return AudienceUploadResult{}, fmt.Errorf("%w: upload source does not match plan", ErrInvalidRequest)
	}
	snapshot, cleanup, err := newAudienceUploadSnapshot(source)
	if err != nil {
		return AudienceUploadResult{}, err
	}
	defer func() { err = errors.Join(err, cleanup()) }()
	if !s.client.clock.Now().Before(plan.ExpiresAt) {
		return AudienceUploadResult{}, ErrUploadExpired
	}
	spec, dataPath, err := audienceUploadSpec(&plan, snapshot)
	if err != nil {
		return AudienceUploadResult{}, err
	}
	requestURL, err := s.client.buildRequestURL(spec)
	if err != nil {
		return AudienceUploadResult{}, err
	}
	for attempt := 1; attempt <= 2; attempt++ {
		response, sendErr := s.client.send(ctx, spec, requestURL, Credentials{})
		if sendErr == nil && response != nil && response.StatusCode >= 200 && response.StatusCode < 300 {
			if closeErr := response.Body.Close(); closeErr != nil {
				return AudienceUploadResult{}, newTransportError(spec, closeErr)
			}
			return AudienceUploadResult{DataPath: dataPath}, nil
		}
		if response != nil {
			status := response.StatusCode
			if closeErr := response.Body.Close(); closeErr != nil {
				return AudienceUploadResult{}, newTransportError(spec, closeErr)
			}
			if !shouldRetryStatus(status) || attempt == 2 {
				return AudienceUploadResult{}, &APIError{Operation: spec.operation, HTTPStatus: status}
			}
		} else if attempt == 2 || !shouldRetryError(sendErr) {
			return AudienceUploadResult{}, newTransportError(spec, sendErr)
		}
		if !s.client.clock.Now().Before(plan.ExpiresAt) {
			return AudienceUploadResult{}, ErrUploadExpired
		}
	}
	return AudienceUploadResult{}, ErrTransport
}

// UploadFile 获取预签名计划并上传文件，不创建人群包。
func (s *AudienceService) UploadFile(ctx context.Context, request AudiencePresignRequest, source UploadSource, options ...RequestOption) (AudienceUploadResult, error) {
	if source.Filename() != request.FileName || source.Size() != request.FileSize || source.ContentMD5() != request.FileMD5 {
		return AudienceUploadResult{}, fmt.Errorf("%w: upload source does not match presign request", ErrInvalidRequest)
	}
	plan, err := s.PresignUpload(ctx, request, options...)
	if err != nil {
		return AudienceUploadResult{}, err
	}
	return s.Upload(ctx, plan, source, options...)
}

func validatePresignRequest(request AudiencePresignRequest) error {
	name := strings.TrimSpace(request.FileName)
	lower := strings.ToLower(name)
	validExtension := strings.HasSuffix(lower, ".txt") || strings.HasSuffix(lower, ".csv") || strings.HasSuffix(lower, ".txt.gz") || strings.HasSuffix(lower, ".csv.gz")
	if request.AreaType < 1 || request.AreaType > 2 || name == "" || filepath.Base(name) != name || request.FileSize < 1 || request.FileSize > 5<<30 || !validExtension {
		return fmt.Errorf("%w: invalid audience presign request", ErrInvalidRequest)
	}
	if _, err := ParseContentMD5(request.FileMD5.String()); err != nil {
		return err
	}
	return nil
}

func (s *AudienceService) parseUploadPlan(request AudiencePresignRequest, data *audiencePresignData) (AudienceUploadPlan, error) {
	if (data.AreaType != 0 && data.AreaType != request.AreaType) || (data.FileName != "" && data.FileName != request.FileName) || (data.FileMD5 != "" && data.FileMD5 != request.FileMD5) || data.TTL <= 0 {
		return AudienceUploadPlan{}, fmt.Errorf("%w: presign response does not match request", ErrUnexpectedResponse)
	}
	plan := AudienceUploadPlan{AreaType: request.AreaType, FileName: request.FileName, FileMD5: request.FileMD5, FileSize: request.FileSize, ExpiresAt: s.client.clock.Now().Add(time.Duration(data.TTL) * time.Second)}
	if request.AreaType == 1 && data.S3 != nil {
		plan.S3 = &AudienceS3Upload{Method: data.S3.Method, URL: data.S3.URL, DataPath: data.S3.DataPath}
	}
	storage := data.OSS
	if request.AreaType == 2 && storage == nil {
		storage = data.S3
	}
	if request.AreaType == 2 && storage == nil && data.Host != "" {
		storage = &audiencePresignStorage{Method: data.Method, AccessID: data.AccessID, Host: data.Host, Expire: data.Expire, Signature: data.Signature, Policy: data.Policy, Directory: data.Directory, DataPath: data.DataPath}
	}
	if storage != nil {
		plan.OSS = &AudienceOSSUpload{Method: storage.Method, AccessID: storage.AccessID, Host: storage.Host, Expire: storage.Expire, Signature: storage.Signature, Policy: storage.Policy, Directory: storage.Directory, DataPath: storage.DataPath}
		if expire, err := strconv.ParseInt(storage.Expire, 10, 64); err == nil {
			ossExpiry := time.Unix(expire, 0)
			if ossExpiry.After(s.client.clock.Now()) && ossExpiry.Before(plan.ExpiresAt) {
				plan.ExpiresAt = ossExpiry
			}
		}
	}
	if (plan.AreaType == 1 && (plan.S3 == nil || !strings.EqualFold(plan.S3.Method, http.MethodPut))) || (plan.AreaType == 2 && (plan.OSS == nil || !strings.EqualFold(plan.OSS.Method, http.MethodPost))) {
		return AudienceUploadPlan{}, fmt.Errorf("%w: invalid presign upload method", ErrUnexpectedResponse)
	}
	return plan, nil
}

func audienceUploadSpec(plan *AudienceUploadPlan, source UploadSource) (requestSpec, string, error) {
	if plan.AreaType == 1 && plan.S3 != nil {
		if err := validateStorageURL(plan.S3.URL); err != nil {
			return requestSpec{}, "", err
		}
		return requestSpec{operation: "audience.upload_s3", method: http.MethodPut, target: absoluteTarget, absoluteURL: plan.S3.URL, body: uploadSourceBody(source), outcomeRisk: true}, plan.S3.DataPath, nil
	}
	if plan.AreaType == 2 && plan.OSS != nil {
		if err := validateStorageURL(plan.OSS.Host); err != nil {
			return requestSpec{}, "", err
		}
		body, contentType, err := ossMultipartBody(plan.OSS, source)
		if err != nil {
			return requestSpec{}, "", err
		}
		return requestSpec{operation: "audience.upload_oss", method: http.MethodPost, target: absoluteTarget, absoluteURL: plan.OSS.Host, body: body, contentType: contentType, outcomeRisk: true}, plan.OSS.DataPath, nil
	}
	return requestSpec{}, "", fmt.Errorf("%w: upload plan has no storage target", ErrInvalidRequest)
}

func ossMultipartBody(plan *AudienceOSSUpload, source UploadSource) (bodyFactory, string, error) {
	var prefix bytes.Buffer
	writer := multipart.NewWriter(&prefix)
	fields := []struct{ name, value string }{{"key", plan.Directory + source.Filename()}, {"OSSAccessKeyId", plan.AccessID}, {"policy", plan.Policy}, {"signature", plan.Signature}, {"success_action_status", "200"}}
	for _, field := range fields {
		if err := writer.WriteField(field.name, field.value); err != nil {
			return nil, "", fmt.Errorf("%w: build OSS upload", ErrInvalidRequest)
		}
	}
	if _, err := writer.CreateFormFile("file", source.Filename()); err != nil {
		return nil, "", fmt.Errorf("%w: build OSS file part", ErrInvalidRequest)
	}
	boundary := writer.Boundary()
	suffix := []byte("\r\n--" + boundary + "--\r\n")
	return func() (io.ReadCloser, int64, error) {
		file, err := source.Open()
		if err != nil {
			return nil, 0, err
		}
		verified := &verifiedUploadReader{reader: file, hash: md5.New(), size: source.Size(), md5: source.ContentMD5()} //nolint:gosec // Provider 契约要求 MD5。
		reader := io.MultiReader(bytes.NewReader(prefix.Bytes()), verified, bytes.NewReader(suffix))
		return &multipartUploadBody{Reader: reader, file: file}, int64(prefix.Len()) + source.Size() + int64(len(suffix)), nil
	}, writer.FormDataContentType(), nil
}

type multipartUploadBody struct {
	io.Reader
	file io.ReadCloser
}

func (body *multipartUploadBody) Close() error { return body.file.Close() }

func uploadSourceBody(source UploadSource) bodyFactory {
	return func() (io.ReadCloser, int64, error) {
		reader, err := source.Open()
		if err != nil {
			return nil, 0, err
		}
		return &verifiedUploadBody{verifiedUploadReader: verifiedUploadReader{reader: reader, hash: md5.New(), size: source.Size(), md5: source.ContentMD5()}, closer: reader}, source.Size(), nil //nolint:gosec // Provider 契约要求 MD5。
	}
}

func validateStorageURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed == nil {
		return fmt.Errorf("%w: invalid storage URL", ErrInvalidRequest)
	}
	secureScheme := parsed.Scheme == "https" || parsed.Scheme == "http" && isLoopbackHost(parsed.Hostname())
	if parsed.Host == "" || parsed.User != nil || parsed.Fragment != "" || !secureScheme {
		return fmt.Errorf("%w: invalid storage URL", ErrInvalidRequest)
	}
	return nil
}
