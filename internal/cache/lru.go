// Package cache provides a concurrency-safe, generic LRU cache with per-entry TTLs.
package cache

import (
	"container/list"
	"sync"
	"time"
)

// entry holds the metadata for one cached item.
// expiresAt (24B) leads so the pointer and interface fields pack tightly after it.
type entry struct {
	expiresAt time.Time
	elem      *list.Element
	value     interface{}
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

// Get retrieves a value. Returns (value, true) on hit; (nil, false) on miss or expiry.
func (c *LRU) Get(key string) (interface{}, bool) {
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

// Set inserts or updates a key. Evicts the LRU entry when at capacity.
func (c *LRU) Set(key string, value interface{}, ttl time.Duration) {
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
			c.remove(back.Value.(*entry)) //nolint:errcheck // *entry cast is always valid; remove never errors
		}
	}
	ne := &entry{key: key, value: value, expiresAt: time.Now().Add(ttl)}
	ne.elem = c.order.PushFront(ne)
	c.items[key] = ne
}

// Delete removes a key. No-op if absent.
func (c *LRU) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e, ok := c.items[key]; ok {
		c.remove(e)
	}
}

// InvalidatePrefix removes all entries whose key starts with prefix.
func (c *LRU) InvalidatePrefix(prefix string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for k, e := range c.items {
		if len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			c.remove(e)
		}
	}
}

// Flush removes all entries.
func (c *LRU) Flush() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[string]*entry, c.maxSize)
	c.order.Init()
}

// Len returns the current number of entries (including expired ones not yet evicted).
func (c *LRU) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.items)
}

func (c *LRU) remove(e *entry) {
	c.order.Remove(e.elem)
	delete(c.items, e.key)
}
