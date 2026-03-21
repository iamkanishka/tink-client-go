// Package ratelimit provides an in-process sliding-window rate limiter.
package ratelimit

import (
	"sync"
	"time"
)

// Bucket tracks requests within a single window for one key.
type Bucket struct {
	mu          sync.Mutex
	count       int
	windowStart time.Time
}

// Registry manages per-key rate-limit buckets.
type Registry struct {
	mu      sync.Mutex
	buckets map[string]*Bucket
	enabled bool
}

// New creates a Registry with rate limiting enabled.
func New() *Registry {
	return &Registry{buckets: make(map[string]*Bucket), enabled: true}
}

// SetEnabled enables or disables rate limiting globally.
func (r *Registry) SetEnabled(v bool) {
	r.mu.Lock()
	r.enabled = v
	r.mu.Unlock()
}

// Allow returns true if the request is within the limit for the window.
func (r *Registry) Allow(key string, limit int, period time.Duration) bool {
	r.mu.Lock()
	enabled := r.enabled
	r.mu.Unlock()
	if !enabled {
		return true
	}
	b := r.getOrCreate(key)
	b.mu.Lock()
	defer b.mu.Unlock()
	if time.Since(b.windowStart) >= period {
		b.count = 0
		b.windowStart = time.Now()
	}
	if b.count >= limit {
		return false
	}
	b.count++
	return true
}

// Remaining returns requests remaining in the current window.
func (r *Registry) Remaining(key string, limit int, period time.Duration) int {
	r.mu.Lock()
	enabled := r.enabled
	r.mu.Unlock()
	if !enabled {
		return limit
	}
	b := r.getOrCreate(key)
	b.mu.Lock()
	defer b.mu.Unlock()
	if time.Since(b.windowStart) >= period {
		return limit
	}
	if rem := limit - b.count; rem > 0 {
		return rem
	}
	return 0
}

// Reset clears the rate-limit state for a key.
func (r *Registry) Reset(key string) {
	r.mu.Lock()
	delete(r.buckets, key)
	r.mu.Unlock()
}

func (r *Registry) getOrCreate(key string) *Bucket {
	r.mu.Lock()
	b, ok := r.buckets[key]
	if !ok {
		b = &Bucket{windowStart: time.Now()}
		r.buckets[key] = b
	}
	r.mu.Unlock()
	return b
}
