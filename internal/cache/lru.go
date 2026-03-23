// Package cache provides a concurrency-safe, fixed-size LRU cache with per-entry TTLs.
//
// The cache uses a doubly-linked list to track access order and a map for O(1) lookups.
// All operations acquire a single Mutex; the LRU is designed for low-contention
// workloads where cache hits are far more frequent than evictions.
package cache

import (
	"container/list"
	"sync"
	"time"
)

// entry holds metadata for one cached item.
// expiresAt (24 B) leads so the pointer and interface fields pack without padding.
type entry struct {
	expiresAt time.Time
	elem      *list.Element
	value     any
	key       string
}

// LRU is a concurrency-safe, size-bounded LRU cache with per-entry TTL expiry.
type LRU struct {
	items   map[string]*entry
	order   *list.List
	mu      sync.Mutex
	maxSize int
}

// New creates an LRU cache capped at maxSize entries.
// If maxSize ≤ 0, the default of 512 is used.
func New(maxSize int) *LRU {
	if maxSize <= 0 {
		maxSize = 512
	}
	return &LRU{
		maxSize: maxSize,
		items:   make(map[string]*entry, maxSize),
		order:   list.New(),
	}
}

// Get retrieves a value by key.
// Returns (value, true) on a cache hit; (nil, false) on a miss or expired entry.
// A hit promotes the entry to the front of the LRU list.
func (c *LRU) Get(key string) (any, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.items[key]
	if !ok {
		return nil, false
	}
	if time.Now().After(e.expiresAt) {
		c.remove(e)
		return nil, false
	}
	c.order.MoveToFront(e.elem)
	return e.value, true
}

// Set inserts or updates a key with the given TTL.
// If the cache is at capacity, the least-recently-used entry is evicted first.
func (c *LRU) Set(key string, value any, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e, ok := c.items[key]; ok {
		e.value = value
		e.expiresAt = time.Now().Add(ttl)
		c.order.MoveToFront(e.elem)
		return
	}
	if c.order.Len() >= c.maxSize {
		if back := c.order.Back(); back != nil {
			c.remove(back.Value.(*entry)) //nolint:errcheck // *entry assertion is always valid
		}
	}
	ne := &entry{key: key, value: value, expiresAt: time.Now().Add(ttl)}
	ne.elem = c.order.PushFront(ne)
	c.items[key] = ne
}

// Delete removes a single key. No-op if the key is absent.
func (c *LRU) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e, ok := c.items[key]; ok {
		c.remove(e)
	}
}

// InvalidatePrefix removes all entries whose key starts with prefix.
// Used to invalidate all cached data for a specific user or path.
func (c *LRU) InvalidatePrefix(prefix string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for k, e := range c.items {
		if len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			c.remove(e)
		}
	}
}

// Flush removes all entries from the cache.
func (c *LRU) Flush() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[string]*entry, c.maxSize)
	c.order.Init()
}

// Len returns the number of entries currently in the cache.
// This may include entries that are expired but not yet evicted.
func (c *LRU) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.items)
}

// remove unlinks entry e from the LRU list and deletes it from the map.
// Caller must hold c.mu.
func (c *LRU) remove(e *entry) {
	c.order.Remove(e.elem)
	delete(c.items, e.key)
}
