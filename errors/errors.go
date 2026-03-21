// Package errors provides the structured TinkError type for tink-client-go.
package errors

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/iamkanishka/tink-client-go/types"
)

// TinkError is the structured error returned by all client methods.
type TinkError struct {
	Type         types.ErrorType
	Message      string
	StatusCode   int
	ErrorCode    string
	RequestID    string
	ErrorDetails map[string]interface{}
	Cause        error
}

func (e *TinkError) Error() string { return e.Format() }
func (e *TinkError) Unwrap() error { return e.Cause }

// Retryable returns true when the error is safe to retry.
// Network errors, timeouts, and 5xx/408/429 are retryable.
func (e *TinkError) Retryable() bool {
	if e.Type == types.ErrorTypeNetwork || e.Type == types.ErrorTypeTimeout {
		return true
	}
	switch e.StatusCode {
	case http.StatusRequestTimeout,
		http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	}
	return false
}

// Format returns a human-readable description: "[401] Unauthorized (TOKEN_INVALID)"
func (e *TinkError) Format() string {
	var sb strings.Builder
	if e.StatusCode != 0 {
		fmt.Fprintf(&sb, "[%d] ", e.StatusCode)
	}
	sb.WriteString(e.Message)
	if e.ErrorCode != "" {
		fmt.Fprintf(&sb, " (%s)", e.ErrorCode)
	}
	return sb.String()
}

// FromResponse creates a TinkError from an HTTP status code and raw body bytes.
func FromResponse(statusCode int, body []byte) *TinkError {
	var details map[string]interface{}
	_ = json.Unmarshal(body, &details)
	return &TinkError{
		Type:         typeFromStatus(statusCode),
		Message:      extractMessage(body),
		StatusCode:   statusCode,
		ErrorCode:    extractField(body, "errorCode", "error"),
		RequestID:    extractField(body, "requestId"),
		ErrorDetails: details,
	}
}

// FromNetworkError wraps a transport-level failure.
func FromNetworkError(cause error) *TinkError {
	if cause == nil {
		return nil
	}
	msg := cause.Error()
	t := types.ErrorTypeNetwork
	lower := strings.ToLower(msg)
	if strings.Contains(lower, "timeout") ||
		strings.Contains(lower, "deadline exceeded") ||
		strings.Contains(lower, "context deadline") {
		t = types.ErrorTypeTimeout
	}
	return &TinkError{Type: t, Message: msg, Cause: cause}
}

// FromDecodeError wraps a JSON decode failure.
func FromDecodeError(cause error) *TinkError {
	msg := "failed to decode response"
	if cause != nil {
		msg = cause.Error()
	}
	return &TinkError{Type: types.ErrorTypeDecode, Message: msg, Cause: cause}
}

// Validation creates a validation_error for missing/invalid config.
func Validation(msg string) *TinkError {
	return &TinkError{Type: types.ErrorTypeValidation, Message: msg}
}

// New creates a TinkError with the given type and message.
func New(t types.ErrorType, msg string) *TinkError {
	return &TinkError{Type: t, Message: msg}
}

func typeFromStatus(s int) types.ErrorType {
	switch {
	case s == 401:
		return types.ErrorTypeAuthentication
	case s == 429:
		return types.ErrorTypeRateLimit
	case s == 400:
		return types.ErrorTypeValidation
	case s >= 400:
		return types.ErrorTypeAPI
	default:
		return types.ErrorTypeUnknown
	}
}

func extractMessage(body []byte) string {
	var m map[string]interface{}
	if err := json.Unmarshal(body, &m); err != nil {
		if s := strings.TrimSpace(string(body)); s != "" {
			return s
		}
		return "HTTP error"
	}
	for _, k := range []string{"errorMessage", "error_description", "message", "error"} {
		if v, ok := m[k]; ok {
			if s, ok := v.(string); ok && s != "" {
				return s
			}
		}
	}
	return "unknown error"
}

func extractField(body []byte, keys ...string) string {
	var m map[string]interface{}
	if json.Unmarshal(body, &m) != nil {
		return ""
	}
	for _, k := range keys {
		if v, ok := m[k]; ok {
			if s, ok := v.(string); ok {
				return s
			}
		}
	}
	return ""
}
