package cache_test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/iamkanishka/tink-client-go/internal/cache"
)

func TestLRU_SetAndGet(t *testing.T) {
	c := cache.New(10)
	c.Set("k", "v", time.Minute)
	v, ok := c.Get("k")
	if !ok {
		t.Fatal("expected cache hit")
	}
	if v != "v" {
		t.Errorf("got %v, want 'v'", v)
	}
}

func TestLRU_MissOnUnknownKey(t *testing.T) {
	c := cache.New(10)
	_, ok := c.Get("missing")
	if ok {
		t.Error("expected cache miss for unknown key")
	}
}

func TestLRU_MissAfterExpiry(t *testing.T) {
	c := cache.New(10)
	c.Set("k", "v", 2*time.Millisecond)
	time.Sleep(10 * time.Millisecond)
	_, ok := c.Get("k")
	if ok {
		t.Error("expected expired entry to be a miss")
	}
}

func TestLRU_UpdateExistingKey(t *testing.T) {
	c := cache.New(10)
	c.Set("k", "old", time.Minute)
	c.Set("k", "new", time.Minute)
	v, ok := c.Get("k")
	if !ok {
		t.Fatal("expected cache hit after update")
	}
	if v != "new" {
		t.Errorf("got %v, want 'new'", v)
	}
}

func TestLRU_EvictsLRUWhenFull(t *testing.T) {
	c := cache.New(3)
	c.Set("a", 1, time.Minute)
	c.Set("b", 2, time.Minute)
	c.Set("c", 3, time.Minute)
	c.Get("a")                 // touch 'a' → b is now LRU
	c.Set("d", 4, time.Minute) // should evict 'b'
	if _, ok := c.Get("a"); !ok {
		t.Error("'a' should still be cached (recently used)")
	}
	if _, ok := c.Get("b"); ok {
		t.Error("'b' should have been evicted (least recently used)")
	}
	if _, ok := c.Get("c"); !ok {
		t.Error("'c' should still be cached")
	}
	if _, ok := c.Get("d"); !ok {
		t.Error("'d' should be cached (just inserted)")
	}
}

func TestLRU_Delete(t *testing.T) {
	c := cache.New(10)
	c.Set("k", "v", time.Minute)
	c.Delete("k")
	if _, ok := c.Get("k"); ok {
		t.Error("expected key to be deleted")
	}
}

func TestLRU_DeleteNonExistent(t *testing.T) {
	c := cache.New(10)
	c.Delete("does-not-exist") // should not panic
}

func TestLRU_InvalidatePrefix(t *testing.T) {
	c := cache.New(20)
	c.Set("user:u1:accounts", "a", time.Minute)
	c.Set("user:u1:transactions", "t", time.Minute)
	c.Set("user:u2:accounts", "a2", time.Minute)
	c.Set("public:providers", "p", time.Minute)
	c.InvalidatePrefix("user:u1:")
	if _, ok := c.Get("user:u1:accounts"); ok {
		t.Error("user:u1:accounts should be invalidated")
	}
	if _, ok := c.Get("user:u1:transactions"); ok {
		t.Error("user:u1:transactions should be invalidated")
	}
	if _, ok := c.Get("user:u2:accounts"); !ok {
		t.Error("user:u2:accounts should still be cached")
	}
	if _, ok := c.Get("public:providers"); !ok {
		t.Error("public:providers should still be cached")
	}
}

func TestLRU_InvalidatePrefixEmpty(t *testing.T) {
	c := cache.New(10)
	c.Set("k", "v", time.Minute)
	c.InvalidatePrefix("no-match:")
	if _, ok := c.Get("k"); !ok {
		t.Error("key should still be cached after non-matching prefix")
	}
}

func TestLRU_Flush(t *testing.T) {
	c := cache.New(10)
	for i := 0; i < 5; i++ {
		c.Set(fmt.Sprintf("k%d", i), i, time.Minute)
	}
	c.Flush()
	if c.Len() != 0 {
		t.Errorf("expected 0 after flush, got %d", c.Len())
	}
}

func TestLRU_Len(t *testing.T) {
	c := cache.New(10)
	if c.Len() != 0 {
		t.Error("empty cache should have Len 0")
	}
	c.Set("a", 1, time.Minute)
	c.Set("b", 2, time.Minute)
	if c.Len() != 2 {
		t.Errorf("expected Len 2, got %d", c.Len())
	}
}

func TestLRU_LenDoesNotCountExpired(t *testing.T) {
	c := cache.New(10)
	c.Set("expired", "v", time.Millisecond)
	c.Set("live", "v", time.Minute)
	time.Sleep(10 * time.Millisecond)
	c.Get("expired") // triggers removal
	if c.Len() != 1 {
		t.Errorf("expected Len 1 after expired removal, got %d", c.Len())
	}
}

func TestLRU_ConcurrentSafety(t *testing.T) {
	c := cache.New(100)
	var wg sync.WaitGroup
	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			key := fmt.Sprintf("key%d", n%50)
			c.Set(key, n, time.Minute)
			c.Get(key)
			if n%10 == 0 {
				c.Delete(key)
			}
		}(i)
	}
	wg.Wait()
}

func TestLRU_MaxSizeEnforced(t *testing.T) {
	c := cache.New(5)
	for i := 0; i < 20; i++ {
		c.Set(fmt.Sprintf("k%d", i), i, time.Minute)
	}
	if c.Len() > 5 {
		t.Errorf("cache exceeded maxSize: Len = %d, want ≤ 5", c.Len())
	}
}
