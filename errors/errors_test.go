package errors_test

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	tinkErrors "github.com/iamkanishka/tink-client-go/errors"
	"github.com/iamkanishka/tink-client-go/types"
)

func TestTinkError_ImplementsError(t *testing.T) {
	var e error = &tinkErrors.TinkError{Type: types.ErrorTypeAPI, Message: "test"}
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
		{"with status", &tinkErrors.TinkError{Message: "Unauthorized", StatusCode: 401}, "[401]"},
		{"with error code", &tinkErrors.TinkError{Message: "Bad", StatusCode: 400, ErrorCode: "INVALID_SCOPE"}, "INVALID_SCOPE"},
		{"no status", &tinkErrors.TinkError{Message: "Connection refused"}, "Connection refused"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.e.Format()
			if got == "" {
				t.Error("Format() must not be empty")
			}
			if tc.contains != "" && !containsStr(got, tc.contains) {
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
		{"network_error", &tinkErrors.TinkError{Type: types.ErrorTypeNetwork}, true},
		{"timeout", &tinkErrors.TinkError{Type: types.ErrorTypeTimeout}, true},
		{"429", &tinkErrors.TinkError{Type: types.ErrorTypeAPI, StatusCode: 429}, true},
		{"500", &tinkErrors.TinkError{Type: types.ErrorTypeAPI, StatusCode: 500}, true},
		{"502", &tinkErrors.TinkError{Type: types.ErrorTypeAPI, StatusCode: 502}, true},
		{"503", &tinkErrors.TinkError{Type: types.ErrorTypeAPI, StatusCode: 503}, true},
		{"504", &tinkErrors.TinkError{Type: types.ErrorTypeAPI, StatusCode: 504}, true},
		{"408", &tinkErrors.TinkError{Type: types.ErrorTypeAPI, StatusCode: 408}, true},
		{"401", &tinkErrors.TinkError{Type: types.ErrorTypeAuthentication, StatusCode: 401}, false},
		{"400", &tinkErrors.TinkError{Type: types.ErrorTypeValidation, StatusCode: 400}, false},
		{"404", &tinkErrors.TinkError{Type: types.ErrorTypeAPI, StatusCode: 404}, false},
		{"validation_no_status", &tinkErrors.TinkError{Type: types.ErrorTypeValidation}, false},
		{"decode_error", &tinkErrors.TinkError{Type: types.ErrorTypeDecode}, false},
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
		status  int
		errType types.ErrorType
	}{
		{http.StatusUnauthorized, types.ErrorTypeAuthentication},
		{http.StatusTooManyRequests, types.ErrorTypeRateLimit},
		{http.StatusBadRequest, types.ErrorTypeValidation},
		{http.StatusForbidden, types.ErrorTypeAPI},
		{http.StatusInternalServerError, types.ErrorTypeAPI},
		{http.StatusBadGateway, types.ErrorTypeAPI},
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
		{"errorMessage field", `{"errorMessage":"Token invalid"}`, "Token invalid"},
		{"message field", `{"message":"Rate limited"}`, "Rate limited"},
		{"error field", `{"error":"invalid_grant"}`, "invalid_grant"},
		{"error_description field", `{"error_description":"The token has expired"}`, "The token has expired"},
		{"plain text body", "Internal Server Error", "Internal Server Error"},
		{"empty body", "", "HTTP error"},
		{"empty JSON object", `{}`, "unknown error"},
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
		{"connection refused", types.ErrorTypeNetwork, true},
		{"request timeout after 30s", types.ErrorTypeTimeout, true},
		{"context deadline exceeded", types.ErrorTypeTimeout, true},
		{"context Deadline Exceeded (uppercase)", types.ErrorTypeTimeout, true},
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

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || sub == "" ||
		func() bool {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		}())
}
