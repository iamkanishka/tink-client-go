package errors_test

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	tinkErrors "github.com/iamkanishka/tink-client-go/errors"
	"github.com/iamkanishka/tink-client-go/types"
)

func TestTinkError_ImplementsError(t *testing.T) {
	e := &tinkErrors.TinkError{
		Type: types.ErrorTypeAuthentication, Message: "Unauthorized",
		StatusCode: http.StatusUnauthorized, ErrorCode: "TOKEN_INVALID",
	}
	if e.Error() == "" {
		t.Error("Error() must return non-empty string")
	}
}

func TestTinkError_Format(t *testing.T) {
	cases := []struct {
		e        *tinkErrors.TinkError
		name     string
		contains string
	}{
		{
			name:     "with status",
			e:        &tinkErrors.TinkError{Message: "Unauthorized", StatusCode: 401},
			contains: "[401]",
		},
		{
			name:     "with error code",
			e:        &tinkErrors.TinkError{Message: "Bad", StatusCode: 400, ErrorCode: "INVALID_SCOPE"},
			contains: "INVALID_SCOPE",
		},
		{
			name:     "no status",
			e:        &tinkErrors.TinkError{Message: "Connection refused"},
			contains: "Connection refused",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.e.Format()
			if got == "" {
				t.Error("Format() must not be empty")
			}
			if tc.contains != "" && !strings.Contains(got, tc.contains) {
				t.Errorf("Format() = %q, want to contain %q", got, tc.contains)
			}
		})
	}
}

func TestTinkError_Unwrap(t *testing.T) {
	cause := fmt.Errorf("underlying cause")
	e := &tinkErrors.TinkError{Type: types.ErrorTypeNetwork, Message: "net error", Cause: cause}
	if !errors.Is(e, cause) {
		t.Error("errors.Is should find cause via Unwrap()")
	}
}

func TestTinkError_Retryable(t *testing.T) {
	cases := []struct {
		e        *tinkErrors.TinkError
		name     string
		expected bool
	}{
		{name: "network_error", e: &tinkErrors.TinkError{Type: types.ErrorTypeNetwork}, expected: true},
		{name: "timeout", e: &tinkErrors.TinkError{Type: types.ErrorTypeTimeout}, expected: true},
		{name: "429", e: &tinkErrors.TinkError{Type: types.ErrorTypeAPI, StatusCode: 429}, expected: true},
		{name: "500", e: &tinkErrors.TinkError{Type: types.ErrorTypeAPI, StatusCode: 500}, expected: true},
		{name: "502", e: &tinkErrors.TinkError{Type: types.ErrorTypeAPI, StatusCode: 502}, expected: true},
		{name: "503", e: &tinkErrors.TinkError{Type: types.ErrorTypeAPI, StatusCode: 503}, expected: true},
		{name: "504", e: &tinkErrors.TinkError{Type: types.ErrorTypeAPI, StatusCode: 504}, expected: true},
		{name: "408", e: &tinkErrors.TinkError{Type: types.ErrorTypeAPI, StatusCode: 408}, expected: true},
		{name: "401", e: &tinkErrors.TinkError{Type: types.ErrorTypeAuthentication, StatusCode: 401}, expected: false},
		{name: "400", e: &tinkErrors.TinkError{Type: types.ErrorTypeValidation, StatusCode: 400}, expected: false},
		{name: "404", e: &tinkErrors.TinkError{Type: types.ErrorTypeAPI, StatusCode: 404}, expected: false},
		{name: "validation_no_status", e: &tinkErrors.TinkError{Type: types.ErrorTypeValidation}, expected: false},
		{name: "decode_error", e: &tinkErrors.TinkError{Type: types.ErrorTypeDecode}, expected: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.e.Retryable(); got != tc.expected {
				t.Errorf("Retryable() = %v, want %v", got, tc.expected)
			}
		})
	}
}

func TestFromResponse_StatusMapping(t *testing.T) {
	cases := []struct {
		errType types.ErrorType
		status  int
	}{
		{errType: types.ErrorTypeAuthentication, status: http.StatusUnauthorized},
		{errType: types.ErrorTypeRateLimit, status: http.StatusTooManyRequests},
		{errType: types.ErrorTypeValidation, status: http.StatusBadRequest},
		{errType: types.ErrorTypeAPI, status: http.StatusForbidden},
		{errType: types.ErrorTypeAPI, status: http.StatusInternalServerError},
		{errType: types.ErrorTypeAPI, status: http.StatusBadGateway},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("status_%d", tc.status), func(t *testing.T) {
			e := tinkErrors.FromResponse(tc.status, []byte(`{"errorMessage":"test"}`))
			if e.Type != tc.errType {
				t.Errorf("status %d: Type = %q, want %q", tc.status, e.Type, tc.errType)
			}
			if e.StatusCode != tc.status {
				t.Errorf("StatusCode = %d, want %d", e.StatusCode, tc.status)
			}
		})
	}
}

func TestFromResponse_MessageExtraction(t *testing.T) {
	cases := []struct {
		name    string
		body    string
		wantMsg string
	}{
		{name: "errorMessage field", body: `{"errorMessage":"Token invalid"}`, wantMsg: "Token invalid"},
		{name: "message field", body: `{"message":"Rate limited"}`, wantMsg: "Rate limited"},
		{name: "error field", body: `{"error":"invalid_grant"}`, wantMsg: "invalid_grant"},
		{name: "error_description field", body: `{"error_description":"The token has expired"}`, wantMsg: "The token has expired"},
		{name: "plain text body", body: "Internal Server Error", wantMsg: "Internal Server Error"},
		{name: "empty body", body: "", wantMsg: "HTTP error"},
		{name: "empty JSON object", body: `{}`, wantMsg: "unknown error"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := tinkErrors.FromResponse(500, []byte(tc.body))
			if e.Message != tc.wantMsg {
				t.Errorf("Message = %q, want %q", e.Message, tc.wantMsg)
			}
		})
	}
}

func TestFromResponse_ErrorCodeExtraction(t *testing.T) {
	e := tinkErrors.FromResponse(400, []byte(`{"errorMessage":"Bad","errorCode":"INVALID_SCOPE"}`))
	if e.ErrorCode != "INVALID_SCOPE" {
		t.Errorf("ErrorCode = %q, want INVALID_SCOPE", e.ErrorCode)
	}
}

func TestFromResponse_RequestIDExtraction(t *testing.T) {
	e := tinkErrors.FromResponse(500, []byte(`{"errorMessage":"err","requestId":"req-abc-123"}`))
	if e.RequestID != "req-abc-123" {
		t.Errorf("RequestID = %q, want req-abc-123", e.RequestID)
	}
}

func TestFromResponse_ErrorDetailsPopulated(t *testing.T) {
	e := tinkErrors.FromResponse(500, []byte(`{"errorMessage":"err","extra":"data"}`))
	if e.ErrorDetails == nil {
		t.Error("ErrorDetails should be populated from body")
	}
	if e.ErrorDetails["extra"] != "data" {
		t.Errorf("ErrorDetails[extra] = %v, want 'data'", e.ErrorDetails["extra"])
	}
}

func TestFromNetworkError_Types(t *testing.T) {
	cases := []struct {
		msg       string
		wantType  types.ErrorType
		retryable bool
	}{
		{msg: "connection refused", wantType: types.ErrorTypeNetwork, retryable: true},
		{msg: "request timeout after 30s", wantType: types.ErrorTypeTimeout, retryable: true},
		{msg: "context deadline exceeded", wantType: types.ErrorTypeTimeout, retryable: true},
		{msg: "context Deadline Exceeded (uppercase)", wantType: types.ErrorTypeTimeout, retryable: true},
	}
	for _, tc := range cases {
		t.Run(tc.msg, func(t *testing.T) {
			e := tinkErrors.FromNetworkError(fmt.Errorf("%s", tc.msg))
			if e.Type != tc.wantType {
				t.Errorf("Type = %q, want %q", e.Type, tc.wantType)
			}
			if e.Retryable() != tc.retryable {
				t.Errorf("Retryable() = %v, want %v", e.Retryable(), tc.retryable)
			}
		})
	}
}

func TestFromNetworkError_NilReturnsNil(t *testing.T) {
	if e := tinkErrors.FromNetworkError(nil); e != nil {
		t.Errorf("expected nil for nil cause, got %v", e)
	}
}

func TestFromNetworkError_PreservesCause(t *testing.T) {
	cause := fmt.Errorf("original error")
	e := tinkErrors.FromNetworkError(cause)
	if !errors.Is(e, cause) {
		t.Error("errors.Is should find original cause")
	}
}

func TestFromDecodeError(t *testing.T) {
	e := tinkErrors.FromDecodeError(fmt.Errorf("unexpected EOF"))
	if e.Type != types.ErrorTypeDecode {
		t.Errorf("Type = %q, want decode_error", e.Type)
	}
	if e.Retryable() {
		t.Error("decode_error should not be retryable")
	}
	if e.Message == "" {
		t.Error("Message should not be empty")
	}
}

func TestFromDecodeError_NilCause(t *testing.T) {
	e := tinkErrors.FromDecodeError(nil)
	if e.Message == "" {
		t.Error("Message should have fallback for nil cause")
	}
}

func TestValidation(t *testing.T) {
	e := tinkErrors.Validation("missing clientID")
	if e.Type != types.ErrorTypeValidation {
		t.Errorf("Type = %q, want validation_error", e.Type)
	}
	if e.Message != "missing clientID" {
		t.Errorf("Message = %q, want 'missing clientID'", e.Message)
	}
	if e.Retryable() {
		t.Error("validation_error should not be retryable")
	}
}

func TestNew(t *testing.T) {
	e := tinkErrors.New(types.ErrorTypeRateLimit, "slow down")
	if e.Type != types.ErrorTypeRateLimit {
		t.Errorf("Type = %q, want rate_limit_error", e.Type)
	}
	if e.Message != "slow down" {
		t.Errorf("Message = %q, want 'slow down'", e.Message)
	}
}

func TestErrorsAs(t *testing.T) {
	e := tinkErrors.Validation("bad config")
	var te *tinkErrors.TinkError
	if !errors.As(e, &te) {
		t.Error("errors.As should work with *TinkError")
	}
	if te.Type != types.ErrorTypeValidation {
		t.Errorf("te.Type = %q, want validation_error", te.Type)
	}
}
