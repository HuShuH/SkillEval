// Package providers contains Phase 1 migration provider contracts and OpenAI-compatible HTTP support.
// It also defines stable provider error classes for retry and diagnostics.
package providers

import (
	"context"
	"errors"
	"fmt"
	"net"
)

// ErrorClass describes a normalized provider error category.
type ErrorClass string

const (
	ErrorClassConfig      ErrorClass = "config_error"
	ErrorClassAuth        ErrorClass = "auth_error"
	ErrorClassRateLimit   ErrorClass = "rate_limit"
	ErrorClassServer      ErrorClass = "server_error"
	ErrorClassNetwork     ErrorClass = "network_error"
	ErrorClassTimeout     ErrorClass = "timeout"
	ErrorClassCanceled    ErrorClass = "canceled"
	ErrorClassBadResponse ErrorClass = "bad_response"
)

// ProviderError is a typed provider error that keeps safe, diagnosable metadata.
type ProviderError struct {
	Class      ErrorClass
	StatusCode int
	Message    string
	Retryable  bool
	Temporary  bool
	Cause      error
}

func (e *ProviderError) Error() string {
	if e == nil {
		return ""
	}
	if e.StatusCode != 0 {
		return fmt.Sprintf("%s (status=%d): %s", e.Class, e.StatusCode, e.Message)
	}
	return fmt.Sprintf("%s: %s", e.Class, e.Message)
}

func (e *ProviderError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

// IsRetryable reports whether the given error is safe to retry.
func IsRetryable(err error) bool {
	var providerErr *ProviderError
	if errors.As(err, &providerErr) {
		return providerErr.Retryable
	}
	return false
}

// ErrorClassOf returns a normalized error class when available.
func ErrorClassOf(err error) string {
	var providerErr *ProviderError
	if errors.As(err, &providerErr) {
		return string(providerErr.Class)
	}
	switch {
	case errors.Is(err, context.Canceled):
		return string(ErrorClassCanceled)
	case errors.Is(err, context.DeadlineExceeded):
		return string(ErrorClassTimeout)
	default:
		return ""
	}
}

// StatusCodeOf returns the provider HTTP status code when available.
func StatusCodeOf(err error) int {
	var providerErr *ProviderError
	if errors.As(err, &providerErr) {
		return providerErr.StatusCode
	}
	return 0
}

func classifyTransportError(err error) *ProviderError {
	switch {
	case errors.Is(err, context.Canceled):
		return &ProviderError{Class: ErrorClassCanceled, Message: "request canceled", Retryable: false, Temporary: false, Cause: err}
	case errors.Is(err, context.DeadlineExceeded):
		return &ProviderError{Class: ErrorClassTimeout, Message: "request timed out", Retryable: true, Temporary: true, Cause: err}
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		class := ErrorClassNetwork
		if netErr.Timeout() {
			class = ErrorClassTimeout
		}
		return &ProviderError{
			Class:     class,
			Message:   "network request failed",
			Retryable: true,
			Temporary: true,
			Cause:     err,
		}
	}
	return &ProviderError{
		Class:     ErrorClassNetwork,
		Message:   "network request failed",
		Retryable: true,
		Temporary: true,
		Cause:     err,
	}
}
