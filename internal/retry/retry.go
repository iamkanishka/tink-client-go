// Package retry implements exponential back-off with jitter.
package retry

import (
	"context"
	"math"
	"math/rand/v2"
	"time"
)

// Policy defines retry behaviour.
// ShouldRetry is placed first so the func pointer (8B) leads, followed by
// value types. The struct cannot be made smaller than 40 bytes because it
// contains a func + two time.Duration + float64 + int.
//
//nolint:govet // fieldalignment: all fields are necessary; struct cannot be smaller
type Policy struct {
	ShouldRetry  func(err error) bool
	BaseDelay    time.Duration
	MaxDelay     time.Duration
	JitterFactor float64
	MaxAttempts  int
}

// DefaultPolicy returns sensible production defaults.
func DefaultPolicy() Policy {
	return Policy{
		MaxAttempts:  3,
		BaseDelay:    time.Second,
		MaxDelay:     30 * time.Second,
		JitterFactor: 0.1,
	}
}

// Do executes fn up to p.MaxAttempts times, retrying on retryable errors.
// Respects context cancellation between attempts.
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
	for attempt := 1; ; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		lastErr = fn()
		if lastErr == nil {
			return nil
		}
		if !canRetry(p, lastErr) || attempt >= p.MaxAttempts {
			return lastErr
		}
		delay := CalculateDelay(attempt, p.BaseDelay, p.MaxDelay, p.JitterFactor)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}
}

// CalculateDelay returns the back-off duration for attempt n (1-indexed).
// Formula: min(base * 2^(n-1), maxDelay) ± jitter.
func CalculateDelay(attempt int, base, maxDelay time.Duration, jitterFactor float64) time.Duration {
	exp := float64(base) * math.Pow(2, float64(attempt-1))
	if maxDelay > 0 && time.Duration(exp) > maxDelay {
		exp = float64(maxDelay)
	}
	// math/rand/v2 is automatically seeded; safe for non-cryptographic use.
	jitter := exp * jitterFactor * (rand.Float64()*2 - 1) //nolint:gosec
	if d := time.Duration(exp + jitter); d > 0 {
		return d
	}
	return 0
}

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
