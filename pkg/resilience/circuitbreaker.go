package resilience

import (
	"sync"
	"time"
)

type CircuitState int

const (
	StateClosed   CircuitState = iota
	StateOpen
	StateHalfOpen
)

func (s CircuitState) String() string {
	switch s {
	case StateClosed:
		return "CLOSED"
	case StateOpen:
		return "OPEN"
	case StateHalfOpen:
		return "HALF_OPEN"
	default:
		return "UNKNOWN"
	}
}

type CircuitBreaker struct {
	mu               sync.RWMutex
	state            CircuitState
	failures         int
	successes        int
	lastFailure      time.Time
	failureThreshold int
	cooldown         time.Duration
	successThreshold int
}

type Option func(*CircuitBreaker)

func WithFailureThreshold(n int) Option {
	return func(cb *CircuitBreaker) { cb.failureThreshold = n }
}

func WithCooldown(d time.Duration) Option {
	return func(cb *CircuitBreaker) { cb.cooldown = d }
}

func WithSuccessThreshold(n int) Option {
	return func(cb *CircuitBreaker) { cb.successThreshold = n }
}

func NewCircuitBreaker(opts ...Option) *CircuitBreaker {
	cb := &CircuitBreaker{
		state:            StateClosed,
		failureThreshold: 5,
		cooldown:         30 * time.Second,
		successThreshold: 3,
	}
	for _, o := range opts {
		o(cb)
	}
	return cb
}

func (cb *CircuitBreaker) CanExecute() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		if time.Since(cb.lastFailure) >= cb.cooldown {
			cb.state = StateHalfOpen
			cb.successes = 0
			return true
		}
		return false
	case StateHalfOpen:
		return true
	default:
		return false
	}
}

func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failures = 0
	if cb.state == StateHalfOpen {
		cb.successes++
		if cb.successes >= cb.successThreshold {
			cb.state = StateClosed
		}
	}
}

func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failures++
	cb.lastFailure = time.Now()
	switch cb.state {
	case StateClosed:
		if cb.failures >= cb.failureThreshold {
			cb.state = StateOpen
		}
	case StateHalfOpen:
		cb.state = StateOpen
	}
}

func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.state = StateClosed
	cb.failures = 0
	cb.successes = 0
	cb.lastFailure = time.Time{}
}
