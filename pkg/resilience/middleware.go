package resilience

import (
	"context"
	"errors"
	"fmt"
	"time"
)

var (
	ErrRateLimited = errors.New("rate limit exceeded, retry after wait period")
	ErrCircuitOpen = errors.New("service temporarily unavailable (circuit breaker open)")
)

type Config struct {
	RateLimit        int
	FailureThreshold int
	Cooldown         time.Duration
	SuccessThreshold int
	Compact          bool
}

type Middleware struct {
	rateLimiter    *RateLimiter
	circuitBreaker *CircuitBreaker
	compact        bool
}

func NewMiddleware(cfg Config) *Middleware {
	m := &Middleware{compact: cfg.Compact}
	if cfg.RateLimit > 0 {
		m.rateLimiter = NewRateLimiter(cfg.RateLimit)
	}
	if cfg.FailureThreshold > 0 {
		var opts []Option
		opts = append(opts, WithFailureThreshold(cfg.FailureThreshold))
		if cfg.Cooldown > 0 {
			opts = append(opts, WithCooldown(cfg.Cooldown))
		}
		if cfg.SuccessThreshold > 0 {
			opts = append(opts, WithSuccessThreshold(cfg.SuccessThreshold))
		}
		m.circuitBreaker = NewCircuitBreaker(opts...)
	}
	return m
}

func (m *Middleware) Execute(ctx context.Context, fn func() (any, error)) (any, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("context canceled: %w", err)
	}
	if m.circuitBreaker != nil && !m.circuitBreaker.CanExecute() {
		return nil, ErrCircuitOpen
	}
	if m.rateLimiter != nil && !m.rateLimiter.Allow(1) {
		return nil, ErrRateLimited
	}
	result, err := fn()
	if m.circuitBreaker != nil {
		if err != nil {
			m.circuitBreaker.RecordFailure()
		} else {
			m.circuitBreaker.RecordSuccess()
		}
	}
	if err != nil {
		return nil, err
	}
	if m.compact {
		result = Compact(result)
	}
	return result, nil
}

func (m *Middleware) IsCircuitOpen() bool {
	if m.circuitBreaker == nil {
		return false
	}
	return m.circuitBreaker.State() == StateOpen
}

func (m *Middleware) RateLimiterStatus() (available float64, waitTime time.Duration) {
	if m.rateLimiter == nil {
		return 0, 0
	}
	return m.rateLimiter.Available(), m.rateLimiter.WaitTime()
}

func (m *Middleware) CircuitBreakerStatus() (state CircuitState, failures int) {
	if m.circuitBreaker == nil {
		return StateClosed, 0
	}
	m.circuitBreaker.mu.RLock()
	defer m.circuitBreaker.mu.RUnlock()
	return m.circuitBreaker.state, m.circuitBreaker.failures
}
