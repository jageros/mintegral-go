package mintegral

import (
	"errors"
	"net/http"
	"time"
)

func defaultHTTPClient() *http.Client {
	transport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return &http.Client{Transport: http.DefaultTransport}
	}
	clone := transport.Clone()
	clone.MaxIdleConns = 64
	clone.MaxIdleConnsPerHost = 16
	clone.IdleConnTimeout = 90 * time.Second
	clone.ResponseHeaderTimeout = 30 * time.Second
	return &http.Client{Transport: clone}
}

func cloneHTTPClient(source *http.Client) *http.Client {
	clone := *source
	originalRedirect := source.CheckRedirect
	clone.CheckRedirect = func(request *http.Request, via []*http.Request) error {
		if len(via) > 0 {
			initial := via[0].URL
			if request.URL.Host != initial.Host || (initial.Scheme == "https" && request.URL.Scheme != "https") {
				return http.ErrUseLastResponse
			}
		}
		if originalRedirect != nil {
			return originalRedirect(request, via)
		}
		if len(via) >= 10 {
			return errors.New("mintegral: stopped after 10 redirects")
		}
		return nil
	}
	return &clone
}
