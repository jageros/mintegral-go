package mintegral

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type baseTarget uint8

const (
	apiTarget baseTarget = iota
	storageTarget
	absoluteTarget
)

type bodyFactory func() (io.ReadCloser, int64, error)

type requestSpec struct {
	query            func() (url.Values, error)
	body             bodyFactory
	header           http.Header
	absoluteURL      string
	operation        string
	method           string
	path             string
	contentType      string
	target           baseTarget
	authenticated    bool
	allowMissingData bool
	retryable        bool
	outcomeRisk      bool
}

type requestConfig struct {
	credentials    Credentials
	credentialsSet bool
}

func (c *Client) resolveCredentials(options []RequestOption) (Credentials, error) {
	config := requestConfig{}
	for _, option := range options {
		if option == nil {
			return Credentials{}, fmt.Errorf("%w: nil request option", ErrInvalidRequest)
		}
		if err := option.applyRequest(&config); err != nil {
			return Credentials{}, err
		}
	}
	if config.credentialsSet {
		if !config.credentials.valid() {
			return Credentials{}, ErrInvalidCredentials
		}
		return config.credentials, nil
	}
	if c.defaultCredentials != nil {
		return *c.defaultCredentials, nil
	}
	return Credentials{}, ErrCredentialsRequired
}

func jsonBody[T any](value T) bodyFactory {
	return func() (io.ReadCloser, int64, error) {
		data, err := json.Marshal(value)
		if err != nil {
			return nil, 0, fmt.Errorf("%w: encode JSON request body", ErrInvalidRequest)
		}
		return io.NopCloser(bytes.NewReader(data)), int64(len(data)), nil
	}
}

//nolint:gocritic // requestSpec 是不可变的内部请求计划，按值传递可隔离调用方后续修改。
func doJSON[T any](ctx context.Context, client *Client, spec requestSpec, options []RequestOption) (T, error) {
	var zero T
	response, err := client.execute(ctx, spec, options)
	if err != nil {
		return zero, err
	}
	if response == nil {
		return zero, fmt.Errorf("%w: %s returned no response", ErrTransport, spec.operation)
	}
	sensitiveValues := client.requestSensitiveValues(&spec, options, response)
	result, decodeErr := decodeEnvelope[T](response, spec.operation, spec.allowMissingData, client.clock.Now(), sensitiveValues...)
	closeErr := response.Body.Close()
	if decodeErr != nil {
		return zero, errors.Join(decodeErr, closeErr)
	}
	if closeErr != nil {
		return zero, fmt.Errorf("%w: close %s response body: %w", ErrUnexpectedResponse, spec.operation, closeErr)
	}
	return result, nil
}

func (c *Client) requestSensitiveValues(spec *requestSpec, options []RequestOption, response *http.Response) []string {
	values := make([]string, 0, 4)
	if spec.authenticated {
		if credentials, err := c.resolveCredentials(options); err == nil {
			values = append(values, credentials.accessKey, credentials.apiKey)
		}
	}
	if response != nil && response.Request != nil {
		values = append(values,
			response.Request.Header.Get("access-key"),
			response.Request.Header.Get("token"),
		)
	}
	return values
}
