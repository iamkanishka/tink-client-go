package webhooks_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/iamkanishka/tink-client-go/types"
	"github.com/iamkanishka/tink-client-go/webhooks"
)

const testSecret = "test_webhook_secret_xyz_123"

func sign(t *testing.T, secret string, payload []byte) string {
	t.Helper()
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

func makeBody(t *testing.T, eventType string, data map[string]interface{}) []byte {
	t.Helper()
	payload := map[string]interface{}{
		"type":      eventType,
		"data":      data,
		"timestamp": "2024-01-15T12:00:00Z",
	}
	b, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	return b
}

// =============================================================================
// Verifier
// =============================================================================

func TestVerifier_Verify_Valid(t *testing.T) {
	v := webhooks.NewVerifier(testSecret)
	payload := []byte(`{"type":"credentials.updated","data":{"userId":"u1"}}`)
	sig := sign(t, testSecret, payload)
	if err := v.Verify(payload, sig); err != nil {
		t.Errorf("expected nil error for valid signature, got: %v", err)
	}
}

func TestVerifier_Verify_MissingSignature(t *testing.T) {
	v := webhooks.NewVerifier(testSecret)
	payload := []byte(`{"type":"test","data":{}}`)
	err := v.Verify(payload, "")
	if err == nil {
		t.Fatal("expected error for missing signature")
	}
	var ve *webhooks.VerificationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected *VerificationError, got %T", err)
	}
	if ve.Code != "missing_signature" {
		t.Errorf("expected code missing_signature, got %q", ve.Code)
	}
}

func TestVerifier_Verify_WrongSecret(t *testing.T) {
	v := webhooks.NewVerifier(testSecret)
	payload := []byte(`{"type":"test","data":{}}`)
	sig := sign(t, "wrong-secret", payload)
	err := v.Verify(payload, sig)
	if err == nil {
		t.Fatal("expected error for wrong secret")
	}
	var ve *webhooks.VerificationError
	if errors.As(err, &ve) && ve.Code != "invalid_signature" {
		t.Errorf("expected invalid_signature, got %q", ve.Code)
	}
}

func TestVerifier_Verify_TamperedBody(t *testing.T) {
	v := webhooks.NewVerifier(testSecret)
	original := []byte(`{"type":"test","data":{}}`)
	tampered := []byte(`{"type":"test","data":{"injected":true}}`)
	sig := sign(t, testSecret, original)
	if err := v.Verify(tampered, sig); err == nil {
		t.Fatal("expected error for tampered body")
	}
}

func TestVerifier_Verify_InvalidHexSignature(t *testing.T) {
	v := webhooks.NewVerifier(testSecret)
	payload := []byte(`{"type":"test","data":{}}`)
	if err := v.Verify(payload, "not-valid-hex!!!"); err == nil {
		t.Fatal("expected error for invalid hex signature")
	}
}

func TestVerifier_GenerateSignatureHex_Deterministic(t *testing.T) {
	v := webhooks.NewVerifier(testSecret)
	payload := []byte("hello world")
	sig1 := v.GenerateSignatureHex(payload)
	sig2 := v.GenerateSignatureHex(payload)
	if sig1 != sig2 {
		t.Error("signatures should be deterministic")
	}
}

func TestVerifier_GenerateSignatureHex_Format(t *testing.T) {
	v := webhooks.NewVerifier(testSecret)
	sig := v.GenerateSignatureHex([]byte("payload"))
	if len(sig) != 64 {
		t.Errorf("expected 64 hex chars, got %d", len(sig))
	}
	for _, c := range sig {
		if !('0' <= c && c <= '9') && !('a' <= c && c <= 'f') {
			t.Errorf("non-hex character %q in signature", c)
		}
	}
}

func TestVerifier_RoundTrip(t *testing.T) {
	v := webhooks.NewVerifier(testSecret)
	payload := []byte("round trip test payload 12345")
	sig := v.GenerateSignatureHex(payload)
	if err := v.Verify(payload, sig); err != nil {
		t.Errorf("round trip failed: %v", err)
	}
}

// =============================================================================
// Handler — HandleRequest
// =============================================================================

func TestHandler_HandleRequest_ValidEvent(t *testing.T) {
	h := webhooks.NewHandler(testSecret)
	body := makeBody(t, "credentials.updated", map[string]interface{}{"userId": "u1"})
	sig := sign(t, testSecret, body)

	var received *types.WebhookEvent
	h.On(types.WebhookEventCredentialsUpdated, func(_ context.Context, e *types.WebhookEvent) error {
		received = e
		return nil
	})

	if err := h.HandleRequest(context.Background(), body, sig); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if received == nil {
		t.Fatal("handler was not called")
	}
	if received.Type != "credentials.updated" {
		t.Errorf("expected type credentials.updated, got %q", received.Type)
	}
	if received.Data["userId"] != "u1" {
		t.Errorf("expected userId u1, got %v", received.Data["userId"])
	}
	if received.Timestamp != "2024-01-15T12:00:00Z" {
		t.Errorf("unexpected timestamp: %q", received.Timestamp)
	}
}

func TestHandler_HandleRequest_TestWebhookReturnNil(t *testing.T) {
	h := webhooks.NewHandler(testSecret)
	body := makeBody(t, "test", map[string]interface{}{})
	sig := sign(t, testSecret, body)

	var called bool
	h.OnAll(func(_ context.Context, e *types.WebhookEvent) error {
		called = true
		return nil
	})

	if err := h.HandleRequest(context.Background(), body, sig); err != nil {
		t.Errorf("expected nil for test webhook, got: %v", err)
	}
	if called {
		t.Error("handlers should NOT be called for test webhooks")
	}
}

func TestHandler_HandleRequest_InvalidSignature(t *testing.T) {
	h := webhooks.NewHandler(testSecret)
	body := makeBody(t, "credentials.updated", nil)
	err := h.HandleRequest(context.Background(), body, "invalidsig")
	if err == nil {
		t.Fatal("expected error for invalid signature")
	}
}

func TestHandler_HandleRequest_MissingSignature(t *testing.T) {
	h := webhooks.NewHandler(testSecret)
	body := makeBody(t, "credentials.updated", nil)
	err := h.HandleRequest(context.Background(), body, "")
	if err == nil {
		t.Fatal("expected error for missing signature")
	}
	var ve *webhooks.VerificationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected *VerificationError, got %T", err)
	}
}

func TestHandler_HandleRequest_InvalidJSON(t *testing.T) {
	h := webhooks.NewHandler(testSecret)
	body := []byte("not json")
	sig := sign(t, testSecret, body)
	err := h.HandleRequest(context.Background(), body, sig)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestHandler_HandleRequest_MissingType(t *testing.T) {
	h := webhooks.NewHandler(testSecret)
	body, _ := json.Marshal(map[string]interface{}{"data": map[string]interface{}{}})
	sig := sign(t, testSecret, body)
	var ve *webhooks.VerificationError
	err := h.HandleRequest(context.Background(), body, sig)
	if !errors.As(err, &ve) || ve.Code != "missing_type" {
		t.Errorf("expected missing_type VerificationError, got: %v", err)
	}
}

func TestHandler_HandleRequest_MissingData(t *testing.T) {
	h := webhooks.NewHandler(testSecret)
	body, _ := json.Marshal(map[string]interface{}{"type": "credentials.updated"})
	sig := sign(t, testSecret, body)
	var ve *webhooks.VerificationError
	err := h.HandleRequest(context.Background(), body, sig)
	if !errors.As(err, &ve) || ve.Code != "missing_data" {
		t.Errorf("expected missing_data VerificationError, got: %v", err)
	}
}

// =============================================================================
// Handler — On / Off / OnAll
// =============================================================================

func TestHandler_On_MultipleHandlersSameType(t *testing.T) {
	h := webhooks.NewHandler(testSecret)
	var count int32
	h.On(types.WebhookEventCredentialsUpdated, func(_ context.Context, _ *types.WebhookEvent) error {
		atomic.AddInt32(&count, 1)
		return nil
	})
	h.On(types.WebhookEventCredentialsUpdated, func(_ context.Context, _ *types.WebhookEvent) error {
		atomic.AddInt32(&count, 1)
		return nil
	})
	body := makeBody(t, "credentials.updated", map[string]interface{}{})
	sig := sign(t, testSecret, body)
	if err := h.HandleRequest(context.Background(), body, sig); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if atomic.LoadInt32(&count) != 2 {
		t.Errorf("expected 2 handler calls, got %d", count)
	}
}

func TestHandler_OnAll_ReceivesAllEventTypes(t *testing.T) {
	h := webhooks.NewHandler(testSecret)
	var count int32
	h.OnAll(func(_ context.Context, _ *types.WebhookEvent) error {
		atomic.AddInt32(&count, 1)
		return nil
	})

	for _, eventType := range []string{"credentials.updated", "credentials.refresh.failed", "provider_consents.created"} {
		body := makeBody(t, eventType, map[string]interface{}{})
		sig := sign(t, testSecret, body)
		if err := h.HandleRequest(context.Background(), body, sig); err != nil {
			t.Fatalf("unexpected error for %s: %v", eventType, err)
		}
	}
	if atomic.LoadInt32(&count) != 3 {
		t.Errorf("expected 3 wildcard calls, got %d", count)
	}
}

func TestHandler_SpecificAndWildcardBothFire(t *testing.T) {
	h := webhooks.NewHandler(testSecret)
	var specific, wildcard int32
	h.On(types.WebhookEventCredentialsUpdated, func(_ context.Context, _ *types.WebhookEvent) error {
		atomic.AddInt32(&specific, 1)
		return nil
	})
	h.OnAll(func(_ context.Context, _ *types.WebhookEvent) error {
		atomic.AddInt32(&wildcard, 1)
		return nil
	})
	body := makeBody(t, "credentials.updated", map[string]interface{}{})
	sig := sign(t, testSecret, body)
	if err := h.HandleRequest(context.Background(), body, sig); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if atomic.LoadInt32(&specific) != 1 {
		t.Errorf("specific handler: expected 1 call, got %d", specific)
	}
	if atomic.LoadInt32(&wildcard) != 1 {
		t.Errorf("wildcard handler: expected 1 call, got %d", wildcard)
	}
}

func TestHandler_UnmatchedTypeDoesNotFire(t *testing.T) {
	h := webhooks.NewHandler(testSecret)
	var called bool
	h.On(types.WebhookEventProviderConsentsCreated, func(_ context.Context, _ *types.WebhookEvent) error {
		called = true
		return nil
	})
	body := makeBody(t, "credentials.updated", map[string]interface{}{})
	sig := sign(t, testSecret, body)
	if err := h.HandleRequest(context.Background(), body, sig); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if called {
		t.Error("mismatched handler should not have been called")
	}
}

func TestHandler_Off_StopsDispatching(t *testing.T) {
	h := webhooks.NewHandler(testSecret)
	var called bool
	h.On(types.WebhookEventCredentialsUpdated, func(_ context.Context, _ *types.WebhookEvent) error {
		called = true
		return nil
	})
	h.Off(types.WebhookEventCredentialsUpdated)

	body := makeBody(t, "credentials.updated", map[string]interface{}{})
	sig := sign(t, testSecret, body)
	if err := h.HandleRequest(context.Background(), body, sig); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if called {
		t.Error("handler should not fire after Off()")
	}
}

func TestHandler_Chaining(t *testing.T) {
	h := webhooks.NewHandler(testSecret)
	result := h.
		On(types.WebhookEventCredentialsUpdated, func(_ context.Context, _ *types.WebhookEvent) error { return nil }).
		OnAll(func(_ context.Context, _ *types.WebhookEvent) error { return nil })
	if result != h {
		t.Error("On and OnAll should return h for chaining")
	}
}

func TestHandler_HandlerError_AllHandlersStillRun(t *testing.T) {
	h := webhooks.NewHandler(testSecret)
	var secondCalled bool
	h.On(types.WebhookEventCredentialsUpdated, func(_ context.Context, _ *types.WebhookEvent) error {
		return errors.New("first handler error")
	})
	h.On(types.WebhookEventCredentialsUpdated, func(_ context.Context, _ *types.WebhookEvent) error {
		secondCalled = true
		return nil
	})
	body := makeBody(t, "credentials.updated", map[string]interface{}{})
	sig := sign(t, testSecret, body)
	err := h.HandleRequest(context.Background(), body, sig)
	if err == nil {
		t.Error("expected error from failing handler")
	}
	if !secondCalled {
		t.Error("second handler should still run despite first error")
	}
}

func TestHandler_Handlers_ReturnsSnapshot(t *testing.T) {
	h := webhooks.NewHandler(testSecret)
	h.On(types.WebhookEventCredentialsUpdated, func(_ context.Context, _ *types.WebhookEvent) error { return nil })
	h.On(types.WebhookEventCredentialsUpdated, func(_ context.Context, _ *types.WebhookEvent) error { return nil })
	h.OnAll(func(_ context.Context, _ *types.WebhookEvent) error { return nil })

	snap := h.Handlers()
	if snap["credentials.updated"] != 2 {
		t.Errorf("expected 2 handlers for credentials.updated, got %d", snap["credentials.updated"])
	}
	if snap["*"] != 1 {
		t.Errorf("expected 1 wildcard handler, got %d", snap["*"])
	}
}

// =============================================================================
// VerificationError
// =============================================================================

func TestVerificationError_Error(t *testing.T) {
	e := &webhooks.VerificationError{Code: "invalid_signature", Message: "mismatch"}
	got := e.Error()
	if got == "" {
		t.Error("Error() should return non-empty string")
	}
}

func TestVerificationError_AsTarget(t *testing.T) {
	v := webhooks.NewVerifier(testSecret)
	err := v.Verify([]byte("payload"), "")
	var ve *webhooks.VerificationError
	if !errors.As(err, &ve) {
		t.Errorf("expected *VerificationError, got %T", err)
	}
}
