package retry_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/iamkanishka/tink-client-go/internal/retry"
)

type retryableErr struct{ msg string }

func (e *retryableErr) Error() string   { return e.msg }
func (e *retryableErr) Retryable() bool { return true }

type permanentErr struct{ msg string }

func (e *permanentErr) Error() string   { return e.msg }
func (e *permanentErr) Retryable() bool { return false }

func TestDo_SuccessOnFirstAttempt(t *testing.T) {
	calls := 0
	err := retry.Do(context.Background(), retry.DefaultPolicy(), func() error {
		calls++
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 1 {
		t.Errorf("expected 1 call, got %d", calls)
	}
}

func TestDo_RetriesOnRetryableError(t *testing.T) {
	calls := 0
	p := retry.Policy{MaxAttempts: 3, BaseDelay: time.Millisecond, MaxDelay: time.Millisecond, JitterFactor: 0}
	err := retry.Do(context.Background(), p, func() error {
		calls++
		if calls < 3 {
			return &retryableErr{"temp"}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 3 {
		t.Errorf("expected 3 calls, got %d", calls)
	}
}

func TestDo_NoRetryOnPermanentError(t *testing.T) {
	calls := 0
	p := retry.Policy{MaxAttempts: 5, BaseDelay: time.Millisecond, MaxDelay: time.Millisecond}
	err := retry.Do(context.Background(), p, func() error {
		calls++
		return &permanentErr{"permanent"}
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if calls != 1 {
		t.Errorf("expected 1 call (no retry on permanent), got %d", calls)
	}
}

func TestDo_ExhaustsMaxAttempts(t *testing.T) {
	calls := 0
	p := retry.Policy{MaxAttempts: 4, BaseDelay: time.Millisecond, MaxDelay: time.Millisecond, JitterFactor: 0}
	err := retry.Do(context.Background(), p, func() error {
		calls++
		return &retryableErr{"always fails"}
	})
	if err == nil {
		t.Fatal("expected error after exhausting retries")
	}
	if calls != 4 {
		t.Errorf("expected 4 calls, got %d", calls)
	}
}

func TestDo_RespectsContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	p := retry.Policy{MaxAttempts: 5, BaseDelay: time.Millisecond, MaxDelay: time.Millisecond}
	err := retry.Do(ctx, p, func() error {
		return &retryableErr{"fail"}
	})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got: %v", err)
	}
}

func TestDo_ContextCancelledDuringDelay(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	calls := 0
	p := retry.Policy{MaxAttempts: 10, BaseDelay: 100 * time.Millisecond, MaxDelay: time.Second, JitterFactor: 0}
	retry.Do(ctx, p, func() error { //nolint
		calls++
		return &retryableErr{"fail"}
	})
	// Should have been cut short by the context timeout
	if calls >= 10 {
		t.Error("should not have completed all 10 attempts before context timeout")
	}
}

func TestDo_CustomShouldRetry_AlwaysRetry(t *testing.T) {
	calls := 0
	p := retry.Policy{
		MaxAttempts: 3,
		BaseDelay:   time.Millisecond,
		MaxDelay:    time.Millisecond,
		ShouldRetry: func(err error) bool { return true },
	}
	retry.Do(context.Background(), p, func() error { //nolint
		calls++
		return errors.New("plain error")
	})
	if calls != 3 {
		t.Errorf("custom ShouldRetry=always: expected 3 calls, got %d", calls)
	}
}

func TestDo_CustomShouldRetry_NeverRetry(t *testing.T) {
	calls := 0
	p := retry.Policy{
		MaxAttempts: 3,
		BaseDelay:   time.Millisecond,
		MaxDelay:    time.Millisecond,
		ShouldRetry: func(err error) bool { return false },
	}
	retry.Do(context.Background(), p, func() error { //nolint
		calls++
		return &retryableErr{"retryable but overridden"}
	})
	if calls != 1 {
		t.Errorf("custom ShouldRetry=never: expected 1 call, got %d", calls)
	}
}

func TestDo_MaxAttemptsDefault(t *testing.T) {
	calls := 0
	p := retry.Policy{BaseDelay: time.Millisecond, MaxDelay: time.Millisecond}
	retry.Do(context.Background(), p, func() error { //nolint
		calls++
		return &retryableErr{"fail"}
	})
	if calls != 3 {
		t.Errorf("default MaxAttempts should be 3, got %d calls", calls)
	}
}

func TestCalculateDelay_ExponentialGrowth(t *testing.T) {
	base := 100 * time.Millisecond
	d1 := retry.CalculateDelay(1, base, time.Minute, 0)
	d2 := retry.CalculateDelay(2, base, time.Minute, 0)
	d3 := retry.CalculateDelay(3, base, time.Minute, 0)
	if d1 != base {
		t.Errorf("attempt 1: want %v, got %v", base, d1)
	}
	if d2 != 2*base {
		t.Errorf("attempt 2: want %v, got %v", 2*base, d2)
	}
	if d3 != 4*base {
		t.Errorf("attempt 3: want %v, got %v", 4*base, d3)
	}
}

func TestCalculateDelay_RespectsMaxDelay(t *testing.T) {
	d := retry.CalculateDelay(20, time.Second, 5*time.Second, 0)
	if d > 5*time.Second {
		t.Errorf("delay %v exceeds maxDelay 5s", d)
	}
}

func TestCalculateDelay_NeverNegative(t *testing.T) {
	for i := 0; i < 200; i++ {
		d := retry.CalculateDelay(1, 0, 0, 1.0)
		if d < 0 {
			t.Fatalf("delay should never be negative, got %v", d)
		}
	}
}

func TestCalculateDelay_JitterApplied(t *testing.T) {
	base := 1000 * time.Millisecond
	seen := make(map[time.Duration]bool)
	for i := 0; i < 50; i++ {
		d := retry.CalculateDelay(1, base, time.Minute, 0.1)
		if d < 900*time.Millisecond || d > 1100*time.Millisecond {
			t.Errorf("delay with 10%% jitter out of range: %v", d)
		}
		seen[d] = true
	}
	// With jitter, not all 50 delays should be identical
	if len(seen) < 2 {
		t.Error("jitter should produce varied delays")
	}
}
