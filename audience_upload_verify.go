package mintegral

import (
	"encoding/hex"
	"fmt"
	"hash"
	"io"
)

type verifiedUploadReader struct {
	reader io.Reader
	hash   hash.Hash
	md5    ContentMD5
	size   int64
	read   int64
	done   bool
}

func (reader *verifiedUploadReader) Read(buffer []byte) (int, error) {
	if reader.done {
		return 0, io.EOF
	}
	count, err := reader.reader.Read(buffer)
	reader.read += int64(count)
	_, _ = reader.hash.Write(buffer[:count])
	if reader.read > reader.size {
		return count, fmt.Errorf("%w: upload source size changed", ErrInvalidRequest)
	}
	if err == io.EOF && reader.read != reader.size {
		return count, fmt.Errorf("%w: upload source content changed", ErrInvalidRequest)
	}
	if reader.read == reader.size {
		var extra [1]byte
		extraCount, extraErr := reader.reader.Read(extra[:])
		if extraCount != 0 || (extraErr != nil && extraErr != io.EOF) {
			return count, fmt.Errorf("%w: upload source size changed", ErrInvalidRequest)
		}
		if hex.EncodeToString(reader.hash.Sum(nil)) != reader.md5.String() {
			return count, fmt.Errorf("%w: upload source content changed", ErrInvalidRequest)
		}
		reader.done = true
	}
	return count, err
}

type verifiedUploadBody struct {
	verifiedUploadReader
	closer io.Closer
}

func (body *verifiedUploadBody) Close() error { return body.closer.Close() }
