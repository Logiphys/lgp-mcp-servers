package resilience

import (
	"testing"
	"time"
)

func TestCircuitBreaker_InitialState(t *testing.T) {
	cb := NewCircuitBreaker()
	if cb.State() != StateClosed {
		t.Errorf("initial state = %v, want StateClosed", cb.State())
	}
	if !cb.CanExecute() {
		t.Error("CanExecute = false in CLOSED state")
	}
}

func TestCircuitBreaker_OpensAfterThreshold(t *testing.T) {
	cb := NewCircuitBreaker(WithFailureThreshold(3))
	for i := 0; i < 3; i++ {
		cb.RecordFailure()
	}
	if cb.State() != StateOpen {
		t.Errorf("state after 3 failures = %v, want StateOpen", cb.State())
	}
	if cb.CanExecute() {
		t.Error("CanExecute = true in OPEN state")
	}
}

func TestCircuitBreaker_DefaultThresholds(t *testing.T) {
	cb := NewCircuitBreaker()
	for i := 0; i < 4; i++ {
		cb.RecordFailure()
	}
	if cb.State() != StateClosed {
		t.Error("should still be CLOSED after 4 failures (threshold=5)")
	}
	cb.RecordFailure()
	if cb.State() != StateOpen {
		t.Error("should be OPEN after 5 failures")
	}
}

func TestCircuitBreaker_HalfOpenAfterCooldown(t *testing.T) {
	cb := NewCircuitBreaker(
		WithFailureThreshold(1),
		WithCooldown(50*time.Millisecond),
	)
	cb.RecordFailure()
	if cb.State() != StateOpen {
		t.Fatal("expected OPEN")
	}
	time.Sleep(60 * time.Millisecond)
	if !cb.CanExecute() {
		t.Error("CanExecute = false after cooldown")
	}
	if cb.State() != StateHalfOpen {
		t.Errorf("state = %v, want StateHalfOpen", cb.State())
	}
}

func TestCircuitBreaker_ClosesAfterSuccessThreshold(t *testing.T) {
	cb := NewCircuitBreaker(
		WithFailureThreshold(1),
		WithCooldown(1*time.Millisecond),
		WithSuccessThreshold(2),
	)
	cb.RecordFailure()
	time.Sleep(5 * time.Millisecond)
	cb.CanExecute()
	cb.RecordSuccess()
	if cb.State() != StateHalfOpen {
		t.Error("should still be HALF_OPEN after 1 success (threshold=2)")
	}
	cb.RecordSuccess()
	if cb.State() != StateClosed {
		t.Errorf("state = %v, want StateClosed after 2 successes", cb.State())
	}
}

func TestCircuitBreaker_HalfOpenFailureReopens(t *testing.T) {
	cb := NewCircuitBreaker(
		WithFailureThreshold(1),
		WithCooldown(1*time.Millisecond),
	)
	cb.RecordFailure()
	time.Sleep(5 * time.Millisecond)
	cb.CanExecute()
	cb.RecordFailure()
	if cb.State() != StateOpen {
		t.Errorf("state = %v, want StateOpen after HALF_OPEN failure", cb.State())
	}
}

func TestCircuitBreaker_SuccessResetsFailures(t *testing.T) {
	cb := NewCircuitBreaker(WithFailureThreshold(3))
	cb.RecordFailure()
	cb.RecordFailure()
	cb.RecordSuccess()
	cb.RecordFailure()
	cb.RecordFailure()
	if cb.State() != StateClosed {
		t.Error("should be CLOSED — success reset failure count")
	}
}

func TestCircuitBreaker_Reset(t *testing.T) {
	cb := NewCircuitBreaker(WithFailureThreshold(1))
	cb.RecordFailure()
	cb.Reset()
	if cb.State() != StateClosed {
		t.Errorf("state after Reset = %v, want StateClosed", cb.State())
	}
	if !cb.CanExecute() {
		t.Error("CanExecute after Reset = false")
	}
}

func TestCircuitBreaker_Concurrent(t *testing.T) {
	cb := NewCircuitBreaker()
	done := make(chan bool, 100)
	for i := 0; i < 100; i++ {
		go func(n int) {
			if n%3 == 0 {
				cb.RecordFailure()
			} else {
				cb.RecordSuccess()
			}
			_ = cb.CanExecute()
			_ = cb.State()
			done <- true
		}(i)
	}
	for i := 0; i < 100; i++ {
		<-done
	}
}
