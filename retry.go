package mintegral

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"time"
)

func validateRetryPolicy(policy RetryPolicy) error {
	if policy.MaxAttempts < 1 || policy.MaxAttempts > 10 {
		return fmt.Errorf("%w: retry max attempts must be between 1 and 10", ErrInvalidRequest)
	}
	if policy.InitialBackoff < 0 || policy.MaxBackoff < policy.InitialBackoff {
		return fmt.Errorf("%w: invalid retry backoff", ErrInvalidRequest)
	}
	return nil
}

func (p RetryPolicy) delay(attempt int) time.Duration {
	delay := p.InitialBackoff
	for range attempt - 1 {
		if delay >= p.MaxBackoff/2 {
			return p.MaxBackoff
		}
		delay *= 2
	}
	if delay > p.MaxBackoff {
		return p.MaxBackoff
	}
	return delay
}

func shouldRetryStatus(status int) bool {
	switch status {
	case 408, 429, 500, 502, 503, 504:
		return true
	default:
		return false
	}
}

func shouldRetryError(err error) bool {
	if err == nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	var dnsError *net.DNSError
	if errors.As(err, &dnsError) {
		return false
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}
	var networkError net.Error
	if errors.As(err, &networkError) && networkError.Timeout() {
		return true
	}
	var operationError *net.OpError
	return errors.As(err, &operationError)
}

func waitForRetry(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		select {
		case <-ctx.Done():
			return context.Cause(ctx)
		default:
			return nil
		}
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return context.Cause(ctx)
	case <-timer.C:
		return nil
	}
}
