// Package errors provides the structured [TinkError] type for tink-client-go.
//
// Every client method returns *TinkError on failure. Use [errors.As] from
// the standard library to inspect the error:
//
//	var te *tinkErrors.TinkError
//	if errors.As(err, &te) {
//	    fmt.Println(te.StatusCode, te.Retryable())
//	}
package errors

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/iamkanishka/tink-client-go/types"
)

// TinkError is the structured error returned by all client methods.
//
// Field order is chosen for minimal struct padding:
// map (8 B) → interface/error (16 B) → strings (16 B each) → ErrorType string-alias (16 B) → int (8 B).
type TinkError struct {
	// ErrorDetails holds the raw API error body, parsed as a JSON object.
	ErrorDetails map[string]any
	// Cause is the underlying error (net.Error, json.SyntaxError, etc.).
	Cause error
	// Message is a human-readable description extracted from the API response.
	Message string
	// ErrorCode is the application-level code from the response body (e.g. "TOKEN_INVALID").
	ErrorCode string
	// RequestID is the Tink request ID for support escalation.
	RequestID string
	// Type classifies the error (authentication, rate_limit, network, etc.).
	Type types.ErrorType
	// StatusCode is the HTTP response status, or 0 for transport-level errors.
	StatusCode int
}

// Error implements the [error] interface. Delegates to [Format].
func (e *TinkError) Error() string { return e.Format() }

// Unwrap returns the underlying cause, enabling [errors.Is] and [errors.As] chain traversal.
func (e *TinkError) Unwrap() error { return e.Cause }

// Retryable reports whether this error is safe to retry.
//
// Returns true for:
//   - ErrorTypeNetwork and ErrorTypeTimeout (transport failures)
//   - HTTP 408 (Request Timeout), 429 (Too Many Requests)
//   - HTTP 500, 502, 503, 504 (server errors)
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

// Format returns a human-readable error description.
//
// Examples:
//   - "[401] Unauthorized (TOKEN_INVALID)"
//   - "[500] Internal server error"
//   - "connection refused"
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

// FromResponse constructs a TinkError from an HTTP status code and raw response body.
func FromResponse(statusCode int, body []byte) *TinkError {
	var details map[string]any
	_ = json.Unmarshal(body, &details)
	return &TinkError{
		ErrorDetails: details,
		Type:         typeFromStatus(statusCode),
		Message:      extractMessage(body),
		StatusCode:   statusCode,
		ErrorCode:    extractField(body, "errorCode", "error"),
		RequestID:    extractField(body, "requestId"),
	}
}

// FromNetworkError wraps a transport-level failure.
// Errors whose message contains "timeout" or "deadline exceeded" are classified
// as [types.ErrorTypeTimeout] rather than [types.ErrorTypeNetwork].
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

// FromDecodeError wraps a JSON parse failure.
func FromDecodeError(cause error) *TinkError {
	msg := "failed to decode response"
	if cause != nil {
		msg = cause.Error()
	}
	return &TinkError{Type: types.ErrorTypeDecode, Message: msg, Cause: cause}
}

// Validation creates a validation_error for missing or invalid configuration.
func Validation(msg string) *TinkError {
	return &TinkError{Type: types.ErrorTypeValidation, Message: msg}
}

// New creates a TinkError with the given type and message.
func New(t types.ErrorType, msg string) *TinkError {
	return &TinkError{Type: t, Message: msg}
}

// ── private helpers ────────────────────────────────────────────────────────

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
	var m map[string]any
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
	var m map[string]any
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
