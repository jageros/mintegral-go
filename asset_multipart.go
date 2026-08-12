package mintegral

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
)

func multipartFileBody(source UploadSource) (bodyFactory, string, error) {
	if source.Filename() == "" || source.Size() < 0 {
		return nil, "", fmt.Errorf("%w: invalid upload source", ErrInvalidRequest)
	}
	boundaryWriter := multipart.NewWriter(io.Discard)
	boundary := boundaryWriter.Boundary()
	contentType := boundaryWriter.FormDataContentType()
	return func() (io.ReadCloser, int64, error) {
		input, err := source.Open()
		if err != nil {
			return nil, 0, err
		}
		reader, writer := io.Pipe()
		done := make(chan struct{})
		go func() {
			defer close(done)
			writeMultipartFile(writer, input, boundary, source.Filename())
		}()
		return &multipartReadCloser{PipeReader: reader, input: input, done: done}, -1, nil
	}, contentType, nil
}

type multipartReadCloser struct {
	*io.PipeReader
	input io.Closer
	done  <-chan struct{}
}

func (reader *multipartReadCloser) Close() error {
	err := errors.Join(reader.PipeReader.Close(), reader.input.Close())
	<-reader.done
	return err
}

func writeMultipartFile(output *io.PipeWriter, input io.ReadCloser, boundary, filename string) {
	writer := multipart.NewWriter(output)
	if err := writer.SetBoundary(boundary); err != nil {
		closeErr := input.Close()
		_ = output.CloseWithError(errors.Join(err, closeErr))
		return
	}
	part, err := writer.CreateFormFile("file", filename)
	if err == nil {
		_, err = io.Copy(part, input)
	}
	if closeErr := input.Close(); err == nil {
		err = closeErr
	}
	if closeErr := writer.Close(); err == nil {
		err = closeErr
	}
	_ = output.CloseWithError(err)
}
