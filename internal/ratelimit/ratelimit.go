// Package ratelimit provides a thread-safe in-process sliding-window rate limiter.
//
// The limiter stores a default limit and period at construction time so callers
// only need to supply the key on each request:
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

// Info holds the current rate-limit state for a key.
type Info struct {
	Count     int
	Limit     int
	Remaining int
	ResetsIn  time.Duration
}

// bucket tracks the sliding-window count for one key.
// Fields are ordered to minimise struct padding.
type bucket struct {
	windowStart time.Time // 24 bytes
	count       int       // 8 bytes
}

// Limiter is a fixed-limit, fixed-period sliding-window rate limiter.
// It is safe for concurrent use.
type Limiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket
	limit   int
	period  time.Duration
	enabled bool
}

// New creates a Limiter with the given limit and window period.
// All calls to Allow/Remaining/Inspect use these defaults.
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

// SetEnabled enables or disables the limiter globally. When disabled, Allow
// always returns true. Useful in tests.
func (l *Limiter) SetEnabled(v bool) {
	l.mu.Lock()
	l.enabled = v
	l.mu.Unlock()
}

// Allow reports whether a request for key is within the rate limit.
// Increments the counter on "ok". Returns true immediately when disabled.
func (l *Limiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.enabled {
		return true
	}
	b := l.bucket(key)
	if b.count >= l.limit {
		return false
	}
	b.count++
	return true
}

// Remaining returns the number of requests remaining in the current window.
// Does not increment the counter.
func (l *Limiter) Remaining(key string) int {
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.enabled {
		return l.limit
	}
	b := l.bucket(key)
	if rem := l.limit - b.count; rem > 0 {
		return rem
	}
	return 0
}

// Reset clears the counter for the given key.
func (l *Limiter) Reset(key string) {
	l.mu.Lock()
	delete(l.buckets, key)
	l.mu.Unlock()
}

// ResetAll clears all counters.
func (l *Limiter) ResetAll() {
	l.mu.Lock()
	l.buckets = make(map[string]*bucket)
	l.mu.Unlock()
}

// Inspect returns a non-mutating snapshot of the rate-limit state for key.
func (l *Limiter) Inspect(key string) Info {
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.enabled {
		return Info{Count: 0, Limit: l.limit, Remaining: l.limit}
	}
	b := l.bucket(key)
	rem := l.limit - b.count
	if rem < 0 {
		rem = 0
	}
	resetsIn := l.period - time.Since(b.windowStart)
	if resetsIn < 0 {
		resetsIn = 0
	}
	return Info{Count: b.count, Limit: l.limit, Remaining: rem, ResetsIn: resetsIn}
}

// bucket returns the current-window bucket for key, resetting if the window
// has expired. Caller must hold l.mu.
func (l *Limiter) bucket(key string) *bucket {
	b, ok := l.buckets[key]
	if !ok || time.Since(b.windowStart) >= l.period {
		b = &bucket{windowStart: time.Now()}
		l.buckets[key] = b
	}
	return b
}
