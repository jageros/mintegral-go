package mintegral

import (
	"crypto/md5" //nolint:gosec // Mintegral 上传协议固定使用 MD5 内容摘要，不用于密码学安全。
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// UploadSource 描述可重复打开的上传内容。
// 每次 Open 都必须返回从内容起点读取的独立 reader。
type UploadSource struct {
	opener   func() (io.ReadCloser, error)
	filename string
	md5      ContentMD5
	size     int64
}

// NewUploadSource 从 opener 工厂创建可重复打开的上传源。
func NewUploadSource(filename string, size int64, contentMD5 ContentMD5, opener func() (io.ReadCloser, error)) (UploadSource, error) {
	filename = filepath.Base(strings.TrimSpace(filename))
	if filename == "." || filename == string(filepath.Separator) || filename == "" || size < 0 || opener == nil {
		return UploadSource{}, fmt.Errorf("%w: invalid upload source", ErrInvalidRequest)
	}
	if _, err := ParseContentMD5(contentMD5.String()); err != nil {
		return UploadSource{}, err
	}
	return UploadSource{filename: filename, size: size, md5: contentMD5, opener: opener}, nil
}

// NewFileUploadSource 校验文件并计算摘要，返回每次重新打开文件的上传源。
func NewFileUploadSource(path string) (UploadSource, error) {
	file, err := os.Open(path) //nolint:gosec // path 是调用方显式选择的本地上传文件。
	if err != nil {
		return UploadSource{}, fmt.Errorf("%w: open upload file: %w", ErrInvalidRequest, err)
	}
	info, statErr := file.Stat()
	if statErr != nil {
		return UploadSource{}, fmt.Errorf("%w: inspect upload file: %w", ErrInvalidRequest, errors.Join(statErr, file.Close()))
	}
	if !info.Mode().IsRegular() {
		if closeErr := file.Close(); closeErr != nil {
			return UploadSource{}, fmt.Errorf("%w: close non-regular upload file: %w", ErrInvalidRequest, closeErr)
		}
		return UploadSource{}, fmt.Errorf("%w: upload path must be a regular file", ErrInvalidRequest)
	}
	hash := md5.New() //nolint:gosec // Mintegral 上传契约使用 MD5 识别内容。
	_, copyErr := io.Copy(hash, file)
	closeErr := file.Close()
	if copyErr != nil || closeErr != nil {
		return UploadSource{}, fmt.Errorf("%w: read upload file: %w", ErrInvalidRequest, errors.Join(copyErr, closeErr))
	}
	md5Value := ContentMD5(hex.EncodeToString(hash.Sum(nil)))
	return NewUploadSource(filepath.Base(path), info.Size(), md5Value, func() (io.ReadCloser, error) {
		reader, openErr := os.Open(path) //nolint:gosec // 重开构造时已校验的同一本地上传文件。
		if openErr != nil {
			return nil, fmt.Errorf("%w: reopen upload file: %w", ErrInvalidRequest, openErr)
		}
		return reader, nil
	})
}

// Filename 返回 multipart 中使用的安全文件名。
func (s UploadSource) Filename() string { return s.filename }

// Size 返回内容字节数。
func (s UploadSource) Size() int64 { return s.size }

// ContentMD5 返回内容摘要。
func (s UploadSource) ContentMD5() ContentMD5 { return s.md5 }

// Open 返回一个从内容起点读取的新 reader。
func (s UploadSource) Open() (io.ReadCloser, error) {
	if s.opener == nil {
		return nil, fmt.Errorf("%w: zero upload source", ErrInvalidRequest)
	}
	reader, err := s.opener()
	if err != nil {
		return nil, fmt.Errorf("%w: open upload source", ErrInvalidRequest)
	}
	if reader == nil {
		return nil, fmt.Errorf("%w: upload source opener returned nil", ErrInvalidRequest)
	}
	return &verifiedUploadBody{verifiedUploadReader: verifiedUploadReader{reader: reader, hash: md5.New(), md5: s.md5, size: s.size}, closer: reader}, nil //nolint:gosec // Mintegral 上传契约使用 MD5 识别内容。
}
