package mintegral

import (
	"errors"
	"fmt"
	"io"
	"os"
)

func newAudienceUploadSnapshot(source UploadSource) (snapshot UploadSource, cleanup func() error, err error) {
	input, err := source.Open()
	if err != nil {
		return UploadSource{}, nil, err
	}
	file, err := os.CreateTemp("", "mintegral-audience-upload-")
	if err != nil {
		if closeErr := input.Close(); closeErr != nil {
			return UploadSource{}, nil, fmt.Errorf("%w: prepare upload snapshot", ErrTransport)
		}
		return UploadSource{}, nil, fmt.Errorf("%w: prepare upload snapshot", ErrTransport)
	}
	path := file.Name()
	failed := true
	defer func() {
		if failed {
			err = errors.Join(err, cleanupUploadSnapshot(file, path))
		}
	}()
	_, copyErr := io.Copy(file, input)
	closeErr := input.Close()
	if copyErr != nil || closeErr != nil {
		if errors.Is(copyErr, ErrInvalidRequest) {
			return UploadSource{}, nil, fmt.Errorf("%w: upload source content changed", ErrInvalidRequest)
		}
		return UploadSource{}, nil, fmt.Errorf("%w: snapshot upload source", ErrTransport)
	}
	snapshot, err = NewUploadSource(source.Filename(), source.Size(), source.ContentMD5(), func() (io.ReadCloser, error) {
		return io.NopCloser(io.NewSectionReader(file, 0, source.Size())), nil
	})
	if err != nil {
		return UploadSource{}, nil, err
	}
	failed = false
	cleanup = func() error { return cleanupUploadSnapshot(file, path) }
	return snapshot, cleanup, nil
}

func cleanupUploadSnapshot(file *os.File, path string) error {
	closeErr := file.Close()
	removeErr := os.Remove(path)
	if errors.Is(removeErr, os.ErrNotExist) {
		removeErr = nil
	}
	if closeErr != nil || removeErr != nil {
		return fmt.Errorf("%w: cleanup upload snapshot", ErrTransport)
	}
	return nil
}
