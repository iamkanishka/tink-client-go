package ratelimit_test

import (
	"sync"
	"testing"
	"time"

	"github.com/iamkanishka/tink-client-go/internal/ratelimit"
)

func TestAllow_UnderLimit(t *testing.T) {
	l := ratelimit.New(5, time.Minute)
	for i := 0; i < 5; i++ {
		if !l.Allow("k") {
			t.Fatalf("request %d should be allowed", i+1)
		}
	}
}

func TestAllow_AtLimit_Denied(t *testing.T) {
	l := ratelimit.New(3, time.Minute)
	for i := 0; i < 3; i++ {
		l.Allow("k")
	}
	if l.Allow("k") {
		t.Error("request beyond limit should be denied")
	}
}

func TestAllow_WindowReset(t *testing.T) {
	l := ratelimit.New(2, 5*time.Millisecond)
	l.Allow("k")
	l.Allow("k")
	if l.Allow("k") {
		t.Error("should be limited within window")
	}
	time.Sleep(15 * time.Millisecond)
	if !l.Allow("k") {
		t.Error("should be allowed after window expires")
	}
}

func TestAllow_DifferentKeysIsolated(t *testing.T) {
	l := ratelimit.New(2, time.Minute)
	l.Allow("k1")
	l.Allow("k1")
	if l.Allow("k1") {
		t.Error("k1 should be limited")
	}
	if !l.Allow("k2") {
		t.Error("k2 should be independent")
	}
}

func TestRemaining(t *testing.T) {
	l := ratelimit.New(10, time.Minute)
	l.Allow("k")
	l.Allow("k")
	if rem := l.Remaining("k"); rem != 8 {
		t.Errorf("expected 8 remaining, got %d", rem)
	}
}

func TestRemaining_NeverNegative(t *testing.T) {
	l := ratelimit.New(2, time.Minute)
	for i := 0; i < 5; i++ {
		l.Allow("k")
	}
	if rem := l.Remaining("k"); rem < 0 {
		t.Errorf("remaining should not be negative, got %d", rem)
	}
}

func TestReset(t *testing.T) {
	l := ratelimit.New(2, time.Minute)
	l.Allow("k")
	l.Allow("k")
	if l.Allow("k") {
		t.Error("should be limited before reset")
	}
	l.Reset("k")
	if !l.Allow("k") {
		t.Error("should be allowed after reset")
	}
}

func TestReset_NonExistentKey(t *testing.T) {
	l := ratelimit.New(5, time.Minute)
	l.Reset("never-set") // should not panic
}

func TestResetAll(t *testing.T) {
	l := ratelimit.New(2, time.Minute)
	l.Allow("k1")
	l.Allow("k1")
	l.Allow("k2")
	l.Allow("k2")
	l.ResetAll()
	if !l.Allow("k1") || !l.Allow("k2") {
		t.Error("all keys should be allowed after ResetAll")
	}
}

func TestSetEnabled_Disabled(t *testing.T) {
	l := ratelimit.New(1, time.Minute)
	l.SetEnabled(false)
	for i := 0; i < 100; i++ {
		if !l.Allow("k") {
			t.Fatalf("request %d should be allowed when limiter disabled", i)
		}
	}
}

func TestSetEnabled_ReEnabled(t *testing.T) {
	l := ratelimit.New(2, time.Minute)
	l.SetEnabled(false)
	l.SetEnabled(true)
	l.Allow("k")
	l.Allow("k")
	if l.Allow("k") {
		t.Error("should enforce limit after re-enabling")
	}
}

func TestInspect(t *testing.T) {
	l := ratelimit.New(10, time.Minute)
	l.Allow("k")
	l.Allow("k")
	info := l.Inspect("k")
	if info.Count != 2 {
		t.Errorf("Count = %d, want 2", info.Count)
	}
	if info.Limit != 10 {
		t.Errorf("Limit = %d, want 10", info.Limit)
	}
	if info.Remaining != 8 {
		t.Errorf("Remaining = %d, want 8", info.Remaining)
	}
	if info.ResetsIn <= 0 || info.ResetsIn > time.Minute {
		t.Errorf("ResetsIn out of range: %v", info.ResetsIn)
	}
}

func TestInspect_Disabled(t *testing.T) {
	l := ratelimit.New(10, time.Minute)
	l.SetEnabled(false)
	info := l.Inspect("k")
	if info.Count != 0 {
		t.Errorf("disabled: Count = %d, want 0", info.Count)
	}
	if info.Limit != 10 {
		t.Errorf("disabled: Limit = %d, want 10", info.Limit)
	}
	if info.Remaining != 10 {
		t.Errorf("disabled: Remaining = %d, want 10", info.Remaining)
	}
}

func TestConcurrentAccess(t *testing.T) {
	l := ratelimit.New(1000, time.Minute)
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			l.Allow("shared")
			l.Remaining("shared")
			l.Inspect("shared")
		}(i)
	}
	wg.Wait()
}
