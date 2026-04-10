// Package retry implements exponential back-off with jitter for Go 1.25+.
//
// Usage:
//
//	err := retry.Do(ctx, retry.DefaultPolicy(), func() error {
//	    return callExternalAPI()
//	})
package retry

import (
	"context"
	"math"
	"math/rand/v2"
	"time"
)

// Policy defines retry behaviour.
//
// ShouldRetry is placed first so the func pointer (8 B) occupies the first
// word; the four value-type fields follow contiguously. This is the tightest
// possible layout — the struct cannot be made smaller.
//
//nolint:govet // fieldalignment: func field must lead; layout is already optimal
type Policy struct {
	ShouldRetry  func(err error) bool
	BaseDelay    time.Duration
	MaxDelay     time.Duration
	JitterFactor float64
	MaxAttempts  int
}

// DefaultPolicy returns sensible production defaults:
// 3 attempts, 1 s base delay, 30 s cap, 10 % jitter.
func DefaultPolicy() Policy {
	return Policy{
		MaxAttempts:  3,
		BaseDelay:    time.Second,
		MaxDelay:     30 * time.Second,
		JitterFactor: 0.1,
	}
}

// Do executes fn up to p.MaxAttempts times, retrying on retryable errors.
// It stops immediately if ctx is cancelled.
func Do(ctx context.Context, p Policy, fn func() error) error {
	if p.MaxAttempts <= 0 {
		p.MaxAttempts = 3
	}
	if p.BaseDelay == 0 {
		p.BaseDelay = time.Second
	}
	if p.MaxDelay == 0 {
		p.MaxDelay = 30 * time.Second
	}

	var lastErr error
	for attempt := range p.MaxAttempts { // Go 1.25+ range-over-integer
		if err := ctx.Err(); err != nil {
			return err
		}
		lastErr = fn()
		if lastErr == nil {
			return nil
		}
		if !canRetry(p, lastErr) || attempt == p.MaxAttempts-1 {
			return lastErr
		}
		delay := CalculateDelay(attempt+1, p.BaseDelay, p.MaxDelay, p.JitterFactor)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}
	return lastErr
}

// CalculateDelay returns the back-off duration for attempt n (1-indexed).
//
// Formula: min(base × 2^(n−1), maxDelay) ± jitter.
// Uses math/rand/v2 which is automatically seeded (Go 1.25+).
func CalculateDelay(attempt int, base, maxDelay time.Duration, jitterFactor float64) time.Duration {
	exp := float64(base) * math.Pow(2, float64(attempt-1))
	if maxDelay > 0 && time.Duration(exp) > maxDelay {
		exp = float64(maxDelay)
	}
	// rand.Float64 from math/rand/v2 is non-cryptographic; that is intentional here.
	jitter := exp * jitterFactor * (rand.Float64()*2 - 1) //nolint:gosec // non-cryptographic jitter for back-off
	if d := time.Duration(exp + jitter); d > 0 {
		return d
	}
	return 0
}

// canRetry reports whether the error should trigger a retry.
// It checks p.ShouldRetry first, then falls back to the Retryable() interface.
func canRetry(p Policy, err error) bool {
	if p.ShouldRetry != nil {
		return p.ShouldRetry(err)
	}
	type retryableErr interface{ Retryable() bool }
	if r, ok := err.(retryableErr); ok {
		return r.Retryable()
	}
	return false
}
