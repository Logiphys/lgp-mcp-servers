package resilience

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestMiddleware_SuccessfulExecution(t *testing.T) {
	mw := NewMiddleware(Config{RateLimit: 5000, FailureThreshold: 5, Cooldown: 30 * time.Second, SuccessThreshold: 3, Compact: false})
	result, err := mw.Execute(context.Background(), func() (any, error) {
		return map[string]any{"status": "ok"}, nil
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result.(map[string]any)["status"] != "ok" {
		t.Errorf("result = %v", result)
	}
}

func TestMiddleware_WithCompaction(t *testing.T) {
	mw := NewMiddleware(Config{Compact: true})
	result, err := mw.Execute(context.Background(), func() (any, error) {
		return map[string]any{"a": "keep", "b": nil, "c": []any{}}, nil
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	m := result.(map[string]any)
	if _, ok := m["b"]; ok {
		t.Error("compaction should remove nil")
	}
	if _, ok := m["c"]; ok {
		t.Error("compaction should remove empty slices")
	}
}

func TestMiddleware_RateLimitExhausted(t *testing.T) {
	mw := NewMiddleware(Config{RateLimit: 1})
	_, _ = mw.Execute(context.Background(), func() (any, error) { return "ok", nil })
	_, err := mw.Execute(context.Background(), func() (any, error) { return "ok", nil })
	if !errors.Is(err, ErrRateLimited) {
		t.Errorf("error = %v, want ErrRateLimited", err)
	}
}

func TestMiddleware_CircuitBreakerOpens(t *testing.T) {
	mw := NewMiddleware(Config{FailureThreshold: 2, Cooldown: 1 * time.Second})
	fail := errors.New("api error")
	for i := 0; i < 2; i++ {
		_, _ = mw.Execute(context.Background(), func() (any, error) { return nil, fail })
	}
	_, err := mw.Execute(context.Background(), func() (any, error) { return "ok", nil })
	if !errors.Is(err, ErrCircuitOpen) {
		t.Errorf("error = %v, want ErrCircuitOpen", err)
	}
}

func TestMiddleware_RecordsSuccessAndFailure(t *testing.T) {
	mw := NewMiddleware(Config{FailureThreshold: 5})
	_, _ = mw.Execute(context.Background(), func() (any, error) { return "ok", nil })
	state, failures := mw.CircuitBreakerStatus()
	if state != StateClosed || failures != 0 {
		t.Errorf("after success: state=%v failures=%d", state, failures)
	}
	_, _ = mw.Execute(context.Background(), func() (any, error) { return nil, errors.New("fail") })
	_, failures = mw.CircuitBreakerStatus()
	if failures != 1 {
		t.Errorf("after failure: failures=%d, want 1", failures)
	}
}

func TestMiddleware_DisabledComponents(t *testing.T) {
	mw := NewMiddleware(Config{})
	for i := 0; i < 100; i++ {
		_, err := mw.Execute(context.Background(), func() (any, error) { return "ok", nil })
		if err != nil {
			t.Fatalf("Execute %d failed: %v", i, err)
		}
	}
}

func TestMiddleware_ContextCancellation(t *testing.T) {
	mw := NewMiddleware(Config{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := mw.Execute(ctx, func() (any, error) { return "ok", nil })
	if err == nil {
		t.Error("expected context cancellation error")
	}
}

func TestMiddleware_StatusMethods(t *testing.T) {
	mw := NewMiddleware(Config{RateLimit: 1000, FailureThreshold: 5})
	avail, wt := mw.RateLimiterStatus()
	if avail <= 0 || wt != 0 {
		t.Errorf("RateLimiterStatus = (%f, %v)", avail, wt)
	}
	if mw.IsCircuitOpen() {
		t.Error("IsCircuitOpen = true, want false")
	}
}
