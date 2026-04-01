package resilience

import (
	"testing"
	"time"
)

func TestRateLimiter_NewDefaults(t *testing.T) {
	rl := NewRateLimiter(5000)
	if got := rl.Available(); got != 5000 {
		t.Errorf("initial tokens = %f, want 5000", got)
	}
}

func TestRateLimiter_Allow(t *testing.T) {
	rl := NewRateLimiter(100)
	if !rl.Allow(1) {
		t.Error("Allow(1) = false, want true")
	}
	if got := rl.Available(); got >= 100 {
		t.Errorf("Available() = %f, should be less than 100", got)
	}
}

func TestRateLimiter_AllowExhausted(t *testing.T) {
	rl := NewRateLimiter(10)
	for i := 0; i < 10; i++ {
		if !rl.Allow(1) {
			t.Fatalf("Allow(1) = false at iteration %d", i)
		}
	}
	if rl.Allow(1) {
		t.Error("Allow(1) = true when exhausted")
	}
}

func TestRateLimiter_AllowMultiple(t *testing.T) {
	rl := NewRateLimiter(10)
	if !rl.Allow(5) {
		t.Error("Allow(5) = false")
	}
	if !rl.Allow(5) {
		t.Error("Allow(5) = false (should have exactly 5 left)")
	}
	if rl.Allow(1) {
		t.Error("Allow(1) = true when exhausted")
	}
}

func TestRateLimiter_WaitTime(t *testing.T) {
	rl := NewRateLimiter(3600) // 1 token per second
	rl.Allow(3600)
	wt := rl.WaitTime()
	if wt <= 0 {
		t.Errorf("WaitTime() = %v, want > 0", wt)
	}
	if wt > 2*time.Second {
		t.Errorf("WaitTime() = %v, want <= 2s", wt)
	}
}

func TestRateLimiter_Reset(t *testing.T) {
	rl := NewRateLimiter(100)
	rl.Allow(100)
	rl.Reset()
	if got := rl.Available(); got != 100 {
		t.Errorf("Available after Reset = %f, want 100", got)
	}
}

func TestRateLimiter_Refill(t *testing.T) {
	rl := NewRateLimiter(3600) // 1 token/sec
	rl.Allow(3600)
	rl.mu.Lock()
	rl.lastRefill = rl.lastRefill.Add(-2 * time.Second)
	rl.mu.Unlock()
	avail := rl.Available()
	if avail < 1.5 || avail > 2.5 {
		t.Errorf("Available after 2s = %f, want ~2.0", avail)
	}
}

func TestRateLimiter_CapsAtMax(t *testing.T) {
	rl := NewRateLimiter(100)
	rl.mu.Lock()
	rl.lastRefill = rl.lastRefill.Add(-24 * time.Hour)
	rl.mu.Unlock()
	if got := rl.Available(); got != 100 {
		t.Errorf("Available capped = %f, want 100", got)
	}
}

func TestRateLimiter_Concurrent(t *testing.T) {
	rl := NewRateLimiter(1000)
	done := make(chan bool, 100)
	for i := 0; i < 100; i++ {
		go func() {
			rl.Allow(1)
			_ = rl.Available()
			_ = rl.WaitTime()
			done <- true
		}()
	}
	for i := 0; i < 100; i++ {
		<-done
	}
}
