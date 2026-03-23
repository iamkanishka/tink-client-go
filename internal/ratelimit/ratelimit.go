// Package ratelimit provides a thread-safe in-process sliding-window rate limiter.
//
// Usage:
//
//	l := ratelimit.New(100, time.Hour)
//	if !l.Allow("user:u1") {
//	    return ErrTooManyRequests
//	}
package ratelimit

import (
	"sync"
	"time"
)

// Info holds a non-mutating snapshot of rate-limit state for one key.
// time.Duration (8 B) leads so all fields pack without padding.
type Info struct {
	ResetsIn  time.Duration
	Count     int
	Limit     int
	Remaining int
}

// bucket tracks the sliding-window count for one key.
type bucket struct {
	windowStart time.Time // 24 B — largest field first
	count       int       //  8 B
}

// Limiter is a fixed-limit, fixed-period sliding-window rate limiter.
// Safe for concurrent use by multiple goroutines.
//
// Field order: map (8 B), period (8 B), mu (variable), limit (8 B), enabled (1 B).
// enabled is placed after limit so the compiler can pack it in the same word
// as the bool's trailing padding without inserting a gap before the mutex.
type Limiter struct {
	buckets map[string]*bucket
	period  time.Duration
	mu      sync.Mutex
	limit   int
	enabled bool
}

// New creates a Limiter with the given limit and window period.
func New(limit int, period time.Duration) *Limiter {
	if limit <= 0 {
		limit = 100
	}
	if period <= 0 {
		period = time.Hour
	}
	return &Limiter{
		buckets: make(map[string]*bucket),
		limit:   limit,
		period:  period,
		enabled: true,
	}
}

// SetEnabled enables or disables rate limiting globally.
// Useful in tests to bypass all limits.
func (l *Limiter) SetEnabled(v bool) {
	l.mu.Lock()
	l.enabled = v
	l.mu.Unlock()
}

// Allow reports whether a request for key is within the configured limit.
// Increments the window counter on success.
// Always returns true when the limiter is disabled.
func (l *Limiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.enabled {
		return true
	}
	b := l.getBucket(key)
	if b.count >= l.limit {
		return false
	}
	b.count++
	return true
}

// Remaining returns the number of requests remaining in the current window
// without incrementing the counter.
func (l *Limiter) Remaining(key string) int {
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.enabled {
		return l.limit
	}
	b := l.getBucket(key)
	if rem := l.limit - b.count; rem > 0 {
		return rem
	}
	return 0
}

// Reset clears the rate-limit state for key.
func (l *Limiter) Reset(key string) {
	l.mu.Lock()
	delete(l.buckets, key)
	l.mu.Unlock()
}

// ResetAll clears state for all keys.
func (l *Limiter) ResetAll() {
	l.mu.Lock()
	l.buckets = make(map[string]*bucket)
	l.mu.Unlock()
}

// Inspect returns a non-mutating [Info] snapshot for key.
func (l *Limiter) Inspect(key string) Info {
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.enabled {
		return Info{Limit: l.limit, Remaining: l.limit}
	}
	b := l.getBucket(key)
	rem := max(0, l.limit-b.count) // Go 1.21 builtin max
	resetsIn := max(0, int64(l.period-time.Since(b.windowStart)))
	return Info{
		ResetsIn:  time.Duration(resetsIn),
		Count:     b.count,
		Limit:     l.limit,
		Remaining: rem,
	}
}

// getBucket returns (or creates) the current-window bucket for key.
// Resets the window if it has expired.
// Caller must hold l.mu.
func (l *Limiter) getBucket(key string) *bucket {
	b, ok := l.buckets[key]
	if !ok || time.Since(b.windowStart) >= l.period {
		b = &bucket{windowStart: time.Now()}
		l.buckets[key] = b
	}
	return b
}
