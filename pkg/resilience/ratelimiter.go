package resilience

import (
	"math"
	"sync"
	"time"
)

type RateLimiter struct {
	mu         sync.Mutex
	tokens     float64
	maxTokens  float64
	refillRate float64 // tokens per millisecond
	lastRefill time.Time
}

func NewRateLimiter(tokensPerHour int) *RateLimiter {
	max := float64(tokensPerHour)
	return &RateLimiter{
		tokens:     max,
		maxTokens:  max,
		refillRate: max / (60 * 60 * 1000),
		lastRefill: time.Now(),
	}
}

func (r *RateLimiter) refill() {
	now := time.Now()
	elapsed := float64(now.Sub(r.lastRefill).Milliseconds())
	r.tokens = math.Min(r.maxTokens, r.tokens+elapsed*r.refillRate)
	r.lastRefill = now
}

func (r *RateLimiter) Allow(n int) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.refill()
	needed := float64(n)
	if r.tokens < needed {
		return false
	}
	r.tokens -= needed
	return true
}

func (r *RateLimiter) WaitTime() time.Duration {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.refill()
	if r.tokens >= 1 {
		return 0
	}
	needed := 1 - r.tokens
	ms := math.Ceil(needed / r.refillRate)
	return time.Duration(ms) * time.Millisecond
}

func (r *RateLimiter) Available() float64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.refill()
	return r.tokens
}

func (r *RateLimiter) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tokens = r.maxTokens
	r.lastRefill = time.Now()
}
