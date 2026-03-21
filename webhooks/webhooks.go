// Package webhooks provides Tink webhook signature verification and typed event dispatch.
//
// Tink signs every webhook payload with your webhook secret using HMAC-SHA256.
// The signature is sent in the X-Tink-Signature header as a hex string.
//
// This package verifies signatures using crypto/subtle.ConstantTimeCompare
// to prevent timing attacks, then dispatches parsed events to your handlers.
//
// Example (net/http handler):
//
//	wh, _ := webhooks.NewHandler(os.Getenv("TINK_WEBHOOK_SECRET"))
//	wh.On(types.WebhookEventCredentialsUpdated, func(ctx context.Context, e *types.WebhookEvent) error {
//	    log.Printf("credentials updated for user %s", e.Data["userId"])
//	    return nil
//	})
//
//	http.HandleFunc("/webhooks/tink", func(w http.ResponseWriter, r *http.Request) {
//	    body, _ := io.ReadAll(r.Body)
//	    sig := r.Header.Get("X-Tink-Signature")
//	    if err := wh.HandleRequest(r.Context(), body, sig); err != nil {
//	        http.Error(w, err.Error(), http.StatusBadRequest)
//	        return
//	    }
//	    w.WriteHeader(http.StatusOK)
//	})
package webhooks

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/iamkanishka/tink-client-go/types"
)

// VerificationError is returned when webhook signature verification fails.
type VerificationError struct {
	// Code is a machine-readable failure code.
	Code    string
	Message string
}

func (e *VerificationError) Error() string {
	return fmt.Sprintf("webhook verification failed [%s]: %s", e.Code, e.Message)
}

// Verifier verifies Tink HMAC-SHA256 webhook signatures.
// It is safe for concurrent use.
type Verifier struct {
	secret []byte
}

// NewVerifier creates a Verifier using the given webhook signing secret.
func NewVerifier(secret string) *Verifier {
	return &Verifier{secret: []byte(secret)}
}

// Verify verifies the HMAC-SHA256 signature of a raw webhook payload.
//
// - payload: raw request body bytes (must be the exact bytes received, before any parsing)
// - signature: value of the X-Tink-Signature header (hex-encoded)
//
// Returns nil if the signature is valid. Returns *VerificationError otherwise.
func (v *Verifier) Verify(payload []byte, signature string) error {
	if signature == "" {
		return &VerificationError{
			Code:    "missing_signature",
			Message: "X-Tink-Signature header is absent — the request may not be from Tink",
		}
	}
	sigBytes, err := hex.DecodeString(signature)
	if err != nil {
		return &VerificationError{
			Code:    "invalid_signature_format",
			Message: "X-Tink-Signature is not valid hex",
		}
	}
	expected := v.GenerateSignature(payload)
	// crypto/hmac.Equal uses constant-time comparison internally.
	if !hmac.Equal(sigBytes, expected) {
		return &VerificationError{
			Code:    "invalid_signature",
			Message: "webhook signature mismatch — payload may have been tampered with",
		}
	}
	return nil
}

// GenerateSignature returns the HMAC-SHA256 signature for a payload.
// Useful for testing or for sending test webhooks to yourself.
func (v *Verifier) GenerateSignature(payload []byte) []byte {
	mac := hmac.New(sha256.New, v.secret)
	mac.Write(payload)
	return mac.Sum(nil)
}

// GenerateSignatureHex returns the hex-encoded HMAC-SHA256 signature.
func (v *Verifier) GenerateSignatureHex(payload []byte) string {
	return hex.EncodeToString(v.GenerateSignature(payload))
}

// HandlerFunc is a typed webhook event handler.
type HandlerFunc func(ctx context.Context, event *types.WebhookEvent) error

// Handler combines signature verification with typed event dispatch.
// It is safe for concurrent use.
type Handler struct {
	verifier *Verifier
	mu       sync.RWMutex
	handlers map[string][]HandlerFunc
}

// NewHandler creates a Handler for the given webhook secret.
func NewHandler(secret string) *Handler {
	return &Handler{
		verifier: NewVerifier(secret),
		handlers: make(map[string][]HandlerFunc),
	}
}

// On registers a handler for a specific event type.
// Use "*" as eventType to receive all events (wildcard handler).
// Multiple handlers per event type are supported; all are called in order.
// Returns h for method chaining.
func (h *Handler) On(eventType types.WebhookEventType, fn HandlerFunc) *Handler {
	h.mu.Lock()
	defer h.mu.Unlock()
	key := string(eventType)
	h.handlers[key] = append(h.handlers[key], fn)
	return h
}

// OnAll registers a wildcard handler that receives every event type.
func (h *Handler) OnAll(fn HandlerFunc) *Handler {
	return h.On("*", fn)
}

// Off removes all handlers for the given event type.
func (h *Handler) Off(eventType types.WebhookEventType) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.handlers, string(eventType))
}

// HandleRequest verifies the signature, parses the payload, and dispatches to handlers.
//
// Returns nil for test webhooks (they are silently acknowledged).
// Returns *VerificationError if the signature is invalid.
// Returns the first non-nil error from any handler, wrapped with the event type.
func (h *Handler) HandleRequest(ctx context.Context, body []byte, signature string) error {
	// 1. Verify HMAC-SHA256 signature
	if err := h.verifier.Verify(body, signature); err != nil {
		return err
	}

	// 2. Parse JSON payload
	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return &VerificationError{Code: "invalid_json", Message: "webhook body is not valid JSON"}
	}

	// 3. Validate required fields
	if _, ok := raw["type"]; !ok {
		return &VerificationError{Code: "missing_type", Message: "webhook payload missing 'type' field"}
	}
	if _, ok := raw["data"]; !ok {
		return &VerificationError{Code: "missing_data", Message: "webhook payload missing 'data' field"}
	}

	// 4. Silently acknowledge test webhooks without dispatching
	if raw["type"] == "test" {
		return nil
	}

	// 5. Build typed event
	data, _ := raw["data"].(map[string]interface{})
	if data == nil {
		data = make(map[string]interface{})
	}
	ts, _ := raw["timestamp"].(string)
	event := &types.WebhookEvent{
		Type:      fmt.Sprint(raw["type"]),
		Data:      data,
		Timestamp: ts,
		Raw:       raw,
	}

	// 6. Dispatch — collect all handler errors
	return h.dispatch(ctx, event)
}

// dispatch calls all matching handlers for the event type plus wildcard handlers.
// Calls are sequential. Returns the first error encountered.
func (h *Handler) dispatch(ctx context.Context, event *types.WebhookEvent) error {
	h.mu.RLock()
	specific := h.handlers[event.Type]
	wildcards := h.handlers["*"]
	h.mu.RUnlock()

	all := make([]HandlerFunc, 0, len(specific)+len(wildcards))
	all = append(all, specific...)
	all = append(all, wildcards...)

	var errs []error
	for _, fn := range all {
		if err := fn(ctx, event); err != nil {
			errs = append(errs, fmt.Errorf("handler for %q: %w", event.Type, err))
		}
	}
	return errors.Join(errs...)
}

// Handlers returns a snapshot of all registered handlers keyed by event type.
// The returned map is a copy — modifying it does not affect the Handler.
func (h *Handler) Handlers() map[string]int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	out := make(map[string]int, len(h.handlers))
	for k, v := range h.handlers {
		out[k] = len(v)
	}
	return out
}
