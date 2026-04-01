# Phase 1: Foundation (`pkg/*`) Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build all shared libraries (`pkg/config`, `pkg/resilience`, `pkg/mcputil`, `pkg/apihelper`) with full test coverage so all 4 MCP servers can depend on them.

**Architecture:** Four independent packages under `pkg/` with zero dependencies between them (except `pkg/mcputil` depends on `mcp-go`). Each package is built TDD-style: failing test first, then implementation. Go 1.23, `slog` for logging, `sync.Mutex`/`sync.RWMutex` for concurrency, `iter.Seq` for pagination.

**Tech Stack:** Go 1.23, `github.com/mark3labs/mcp-go`, `golang.org/x/sync` (singleflight), stdlib only otherwise.

**Design Reference:** `docs/design.md` sections 2.1-2.4 for exact API signatures.

**TypeScript References (for porting logic):**
- Rate Limiter: `/Users/zeisler/lgp-autotask-mcp/src/utils/rateLimiter.ts`
- Circuit Breaker: `/Users/zeisler/lgp-autotask-mcp/src/utils/circuitBreaker.ts`
- Response Compactor: `/Users/zeisler/lgp-autotask-mcp/src/utils/responseCompactor.ts`
- Response Formatter: `/Users/zeisler/lgp-autotask-mcp/src/utils/response.formatter.ts`

---

## Task 1: Go Module Init & Dependencies

**Files:**
- Create: `go.mod`
- Create: `go.sum` (auto-generated)

**Step 1: Initialize Go module**

```bash
cd /Users/zeisler/lgp-mcp
go mod init github.com/Logiphys/lgp-mcp
```

**Step 2: Add dependencies**

```bash
go get github.com/mark3labs/mcp-go@latest
go get golang.org/x/sync
```

**Step 3: Verify module**

```bash
go mod tidy
cat go.mod
```

Expected: module `github.com/Logiphys/lgp-mcp` with go 1.23+, require block with mcp-go and x/sync.

**Step 4: Commit**

```bash
git add go.mod go.sum
git commit -m "feat: initialize Go module with mcp-go and x/sync dependencies"
```

---

## Task 2: `pkg/config` — Environment Configuration

**Files:**
- Create: `pkg/config/config.go`
- Create: `pkg/config/config_test.go`
- Delete: `pkg/config/.gitkeep`

**Step 1: Write the failing tests**

```go
// pkg/config/config_test.go
package config

import (
	"log/slog"
	"os"
	"testing"
)

func TestMustEnv_Present(t *testing.T) {
	t.Setenv("TEST_VAR", "hello")
	got := MustEnv("TEST_VAR")
	if got != "hello" {
		t.Errorf("MustEnv = %q, want %q", got, "hello")
	}
}

func TestMustEnv_Missing(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustEnv did not panic for missing var")
		}
	}()
	MustEnv("NONEXISTENT_VAR_12345")
}

func TestMustEnv_Empty(t *testing.T) {
	t.Setenv("TEST_VAR", "")
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustEnv did not panic for empty var")
		}
	}()
	MustEnv("TEST_VAR")
}

func TestOptEnv_Present(t *testing.T) {
	t.Setenv("TEST_VAR", "value")
	got := OptEnv("TEST_VAR", "fallback")
	if got != "value" {
		t.Errorf("OptEnv = %q, want %q", got, "value")
	}
}

func TestOptEnv_Missing(t *testing.T) {
	got := OptEnv("NONEXISTENT_VAR_12345", "fallback")
	if got != "fallback" {
		t.Errorf("OptEnv = %q, want %q", got, "fallback")
	}
}

func TestOptEnv_Empty(t *testing.T) {
	t.Setenv("TEST_VAR", "")
	got := OptEnv("TEST_VAR", "fallback")
	if got != "fallback" {
		t.Errorf("OptEnv = %q, want %q", got, "fallback")
	}
}

func TestLogLevel(t *testing.T) {
	tests := []struct {
		env  string
		want slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"DEBUG", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"warning", slog.LevelWarn},
		{"error", slog.LevelError},
		{"", slog.LevelInfo},
		{"invalid", slog.LevelInfo},
	}
	for _, tt := range tests {
		t.Run(tt.env, func(t *testing.T) {
			if tt.env == "" {
				os.Unsetenv("LOG_LEVEL")
			} else {
				t.Setenv("LOG_LEVEL", tt.env)
			}
			if got := LogLevel(); got != tt.want {
				t.Errorf("LogLevel(%q) = %v, want %v", tt.env, got, tt.want)
			}
		})
	}
}
```

**Step 2: Run tests to verify they fail**

```bash
go test ./pkg/config/... -v -race -count=1
```

Expected: FAIL — functions not defined.

**Step 3: Write implementation**

```go
// pkg/config/config.go
package config

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
)

// MustEnv returns the value of the environment variable named by key.
// It panics if the variable is not set or is empty.
func MustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic(fmt.Sprintf("required environment variable %s is not set", key))
	}
	return v
}

// OptEnv returns the value of the environment variable named by key,
// or fallback if the variable is not set or is empty.
func OptEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// LogLevel parses the LOG_LEVEL environment variable and returns the
// corresponding slog.Level. Defaults to slog.LevelInfo.
func LogLevel() slog.Level {
	switch strings.ToLower(os.Getenv("LOG_LEVEL")) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
```

**Step 4: Run tests to verify they pass**

```bash
go test ./pkg/config/... -v -race -count=1
```

Expected: PASS (all 7 test cases).

**Step 5: Commit**

```bash
rm pkg/config/.gitkeep
git add pkg/config/
git commit -m "feat: add pkg/config with MustEnv, OptEnv, LogLevel"
```

---

## Task 3: `pkg/resilience/ratelimiter` — Token Bucket Rate Limiter

**Files:**
- Create: `pkg/resilience/ratelimiter.go`
- Create: `pkg/resilience/ratelimiter_test.go`

**Step 1: Write the failing tests**

Port logic from `/Users/zeisler/lgp-autotask-mcp/src/utils/rateLimiter.ts`.

```go
// pkg/resilience/ratelimiter_test.go
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
	rl := NewRateLimiter(100) // 100 tokens/hour
	// Should allow consuming tokens
	if !rl.Allow(1) {
		t.Error("Allow(1) = false, want true")
	}
	if got := rl.Available(); got >= 100 {
		t.Errorf("Available() = %f, should be less than 100 after consuming 1", got)
	}
}

func TestRateLimiter_AllowExhausted(t *testing.T) {
	rl := NewRateLimiter(10)
	// Exhaust all tokens
	for i := 0; i < 10; i++ {
		if !rl.Allow(1) {
			t.Fatalf("Allow(1) = false at iteration %d, want true", i)
		}
	}
	// Next request should fail
	if rl.Allow(1) {
		t.Error("Allow(1) = true when exhausted, want false")
	}
}

func TestRateLimiter_AllowMultiple(t *testing.T) {
	rl := NewRateLimiter(10)
	if !rl.Allow(5) {
		t.Error("Allow(5) = false, want true")
	}
	if !rl.Allow(5) {
		t.Error("Allow(5) = false, want true (should have exactly 5 left)")
	}
	if rl.Allow(1) {
		t.Error("Allow(1) = true when exhausted, want false")
	}
}

func TestRateLimiter_WaitTime(t *testing.T) {
	rl := NewRateLimiter(3600) // 1 token per second
	// Exhaust all tokens
	rl.Allow(3600)
	wt := rl.WaitTime()
	if wt <= 0 {
		t.Errorf("WaitTime() = %v, want > 0", wt)
	}
	if wt > 2*time.Second {
		t.Errorf("WaitTime() = %v, want <= 2s for 1 token at 1/sec", wt)
	}
}

func TestRateLimiter_Reset(t *testing.T) {
	rl := NewRateLimiter(100)
	rl.Allow(100) // exhaust
	rl.Reset()
	if got := rl.Available(); got != 100 {
		t.Errorf("Available after Reset = %f, want 100", got)
	}
}

func TestRateLimiter_Refill(t *testing.T) {
	rl := NewRateLimiter(3600) // 1 token/sec
	rl.Allow(3600)             // exhaust
	// Simulate time passing by adjusting lastRefill
	rl.mu.Lock()
	rl.lastRefill = rl.lastRefill.Add(-2 * time.Second)
	rl.mu.Unlock()
	// Should have ~2 tokens refilled
	avail := rl.Available()
	if avail < 1.5 || avail > 2.5 {
		t.Errorf("Available after 2s = %f, want ~2.0", avail)
	}
}

func TestRateLimiter_CapsAtMax(t *testing.T) {
	rl := NewRateLimiter(100)
	// Simulate a long time passing without consuming
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
```

**Step 2: Run tests to verify they fail**

```bash
go test ./pkg/resilience/... -v -race -count=1
```

Expected: FAIL — `NewRateLimiter` not defined.

**Step 3: Write implementation**

```go
// pkg/resilience/ratelimiter.go
package resilience

import (
	"math"
	"sync"
	"time"
)

// RateLimiter implements a token bucket algorithm for API rate limiting.
type RateLimiter struct {
	mu         sync.Mutex
	tokens     float64
	maxTokens  float64
	refillRate float64 // tokens per millisecond
	lastRefill time.Time
}

// NewRateLimiter creates a rate limiter with the given tokens-per-hour capacity.
func NewRateLimiter(tokensPerHour int) *RateLimiter {
	max := float64(tokensPerHour)
	return &RateLimiter{
		tokens:     max,
		maxTokens:  max,
		refillRate: max / (60 * 60 * 1000), // tokens per millisecond
		lastRefill: time.Now(),
	}
}

func (r *RateLimiter) refill() {
	now := time.Now()
	elapsed := float64(now.Sub(r.lastRefill).Milliseconds())
	r.tokens = math.Min(r.maxTokens, r.tokens+elapsed*r.refillRate)
	r.lastRefill = now
}

// Allow checks whether n tokens are available and consumes them if so.
// Returns false immediately if insufficient tokens (non-blocking).
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

// WaitTime returns how long the caller should wait before 1 token becomes available.
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

// Available returns the current number of available tokens.
func (r *RateLimiter) Available() float64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.refill()
	return r.tokens
}

// Reset restores the rate limiter to its initial full-capacity state.
func (r *RateLimiter) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tokens = r.maxTokens
	r.lastRefill = time.Now()
}
```

**Step 4: Run tests to verify they pass**

```bash
go test ./pkg/resilience/... -v -race -count=1
```

Expected: PASS.

**Step 5: Commit**

```bash
git add pkg/resilience/ratelimiter.go pkg/resilience/ratelimiter_test.go
git commit -m "feat: add token bucket rate limiter in pkg/resilience"
```

---

## Task 4: `pkg/resilience/circuitbreaker` — Circuit Breaker

**Files:**
- Create: `pkg/resilience/circuitbreaker.go`
- Create: `pkg/resilience/circuitbreaker_test.go`

**Step 1: Write the failing tests**

Port from `/Users/zeisler/lgp-autotask-mcp/src/utils/circuitBreaker.ts`.

```go
// pkg/resilience/circuitbreaker_test.go
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
		t.Error("CanExecute = true in OPEN state, want false")
	}
}

func TestCircuitBreaker_DefaultThresholds(t *testing.T) {
	cb := NewCircuitBreaker()
	// Default failure threshold is 5
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
	cb.RecordFailure() // -> OPEN
	if cb.State() != StateOpen {
		t.Fatal("expected OPEN")
	}
	time.Sleep(60 * time.Millisecond)
	if !cb.CanExecute() {
		t.Error("CanExecute = false after cooldown, want true (HALF_OPEN)")
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
	cb.RecordFailure() // -> OPEN
	time.Sleep(5 * time.Millisecond)
	cb.CanExecute() // -> HALF_OPEN
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
	cb.RecordFailure() // -> OPEN
	time.Sleep(5 * time.Millisecond)
	cb.CanExecute() // -> HALF_OPEN
	cb.RecordFailure()
	if cb.State() != StateOpen {
		t.Errorf("state = %v, want StateOpen after HALF_OPEN failure", cb.State())
	}
}

func TestCircuitBreaker_SuccessResetsFailures(t *testing.T) {
	cb := NewCircuitBreaker(WithFailureThreshold(3))
	cb.RecordFailure()
	cb.RecordFailure()
	cb.RecordSuccess() // resets failure count
	cb.RecordFailure()
	cb.RecordFailure()
	if cb.State() != StateClosed {
		t.Error("should be CLOSED — success reset failure count")
	}
}

func TestCircuitBreaker_Reset(t *testing.T) {
	cb := NewCircuitBreaker(WithFailureThreshold(1))
	cb.RecordFailure() // -> OPEN
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
```

**Step 2: Run tests to verify they fail**

```bash
go test ./pkg/resilience/... -v -race -count=1
```

Expected: FAIL — types not defined.

**Step 3: Write implementation**

```go
// pkg/resilience/circuitbreaker.go
package resilience

import (
	"sync"
	"time"
)

// CircuitState represents the state of the circuit breaker.
type CircuitState int

const (
	StateClosed   CircuitState = iota // normal operation
	StateOpen                         // rejecting requests
	StateHalfOpen                     // testing recovery
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

// CircuitBreaker implements the circuit breaker pattern.
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

// Option configures a CircuitBreaker.
type Option func(*CircuitBreaker)

// WithFailureThreshold sets the number of failures before opening the circuit.
func WithFailureThreshold(n int) Option {
	return func(cb *CircuitBreaker) { cb.failureThreshold = n }
}

// WithCooldown sets the duration before transitioning from OPEN to HALF_OPEN.
func WithCooldown(d time.Duration) Option {
	return func(cb *CircuitBreaker) { cb.cooldown = d }
}

// WithSuccessThreshold sets the number of successes in HALF_OPEN before closing.
func WithSuccessThreshold(n int) Option {
	return func(cb *CircuitBreaker) { cb.successThreshold = n }
}

// NewCircuitBreaker creates a circuit breaker with the given options.
// Defaults: failureThreshold=5, cooldown=30s, successThreshold=3.
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

// CanExecute returns true if a request should be allowed through.
// Transitions OPEN -> HALF_OPEN when cooldown has elapsed.
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

// RecordSuccess records a successful call.
// In HALF_OPEN: increments success count, closes circuit if threshold reached.
// In CLOSED: resets failure count.
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

// RecordFailure records a failed call.
// In CLOSED: increments failure count, opens circuit if threshold reached.
// In HALF_OPEN: immediately reopens circuit.
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

// State returns the current state of the circuit breaker.
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Reset returns the circuit breaker to its initial CLOSED state.
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.state = StateClosed
	cb.failures = 0
	cb.successes = 0
	cb.lastFailure = time.Time{}
}
```

**Step 4: Run tests to verify they pass**

```bash
go test ./pkg/resilience/... -v -race -count=1
```

Expected: PASS.

**Step 5: Commit**

```bash
git add pkg/resilience/circuitbreaker.go pkg/resilience/circuitbreaker_test.go
git commit -m "feat: add circuit breaker pattern in pkg/resilience"
```

---

## Task 5: `pkg/resilience/compactor` — Response Compaction

**Files:**
- Create: `pkg/resilience/compactor.go`
- Create: `pkg/resilience/compactor_test.go`

**Step 1: Write the failing tests**

Port from `/Users/zeisler/lgp-autotask-mcp/src/utils/responseCompactor.ts`.

```go
// pkg/resilience/compactor_test.go
package resilience

import (
	"encoding/json"
	"testing"
)

func TestCompact_Nil(t *testing.T) {
	if got := Compact(nil); got != nil {
		t.Errorf("Compact(nil) = %v, want nil", got)
	}
}

func TestCompact_Primitives(t *testing.T) {
	tests := []struct {
		name  string
		input any
		want  any
	}{
		{"zero", 0, 0},
		{"false", false, false},
		{"empty string", "", ""},
		{"string", "hello", "hello"},
		{"number", 42.5, 42.5},
		{"true", true, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Compact(tt.input)
			if got != tt.want {
				t.Errorf("Compact(%v) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestCompact_RemovesNulls(t *testing.T) {
	input := map[string]any{
		"name": "test",
		"age":  nil,
		"city": "Berlin",
	}
	got := Compact(input).(map[string]any)
	if _, ok := got["age"]; ok {
		t.Error("Compact should remove nil values")
	}
	if got["name"] != "test" || got["city"] != "Berlin" {
		t.Error("Compact should preserve non-nil values")
	}
}

func TestCompact_RemovesEmptySlices(t *testing.T) {
	input := map[string]any{
		"name":  "test",
		"items": []any{},
	}
	got := Compact(input).(map[string]any)
	if _, ok := got["items"]; ok {
		t.Error("Compact should remove empty slices")
	}
}

func TestCompact_RemovesEmptyMaps(t *testing.T) {
	input := map[string]any{
		"name":   "test",
		"nested": map[string]any{},
	}
	got := Compact(input).(map[string]any)
	if _, ok := got["nested"]; ok {
		t.Error("Compact should remove empty maps")
	}
}

func TestCompact_PreservesMeaningfulValues(t *testing.T) {
	input := map[string]any{
		"zero":  float64(0),
		"false": false,
		"empty": "",
	}
	got := Compact(input).(map[string]any)
	if len(got) != 3 {
		t.Errorf("Compact should preserve 0, false, empty string; got %d keys", len(got))
	}
}

func TestCompact_Recursive(t *testing.T) {
	input := map[string]any{
		"outer": map[string]any{
			"keep":   "yes",
			"remove": nil,
			"deep": map[string]any{
				"only_null": nil,
			},
		},
	}
	got := Compact(input).(map[string]any)
	outer := got["outer"].(map[string]any)
	if _, ok := outer["remove"]; ok {
		t.Error("should remove nested nil")
	}
	if _, ok := outer["deep"]; ok {
		t.Error("should remove map that becomes empty after compaction")
	}
	if outer["keep"] != "yes" {
		t.Error("should keep non-nil nested values")
	}
}

func TestCompact_Array(t *testing.T) {
	input := []any{
		"hello",
		nil,
		map[string]any{"a": nil},
		map[string]any{"b": "keep"},
	}
	got := Compact(input).([]any)
	if len(got) != 2 {
		t.Errorf("compacted array len = %d, want 2", len(got))
	}
}

func TestCompact_ReturnsNilForAllNullMap(t *testing.T) {
	input := map[string]any{
		"a": nil,
		"b": nil,
	}
	got := Compact(input)
	if got != nil {
		t.Errorf("Compact of all-nil map = %v, want nil", got)
	}
}

func TestEstimateSavings(t *testing.T) {
	original := map[string]any{
		"a": "hello",
		"b": nil,
		"c": []any{},
		"d": map[string]any{},
	}
	compacted := Compact(original)
	pct := EstimateSavings(original, compacted)
	if pct <= 0 {
		t.Errorf("EstimateSavings = %f, want > 0", pct)
	}
	if pct >= 100 {
		t.Errorf("EstimateSavings = %f, want < 100", pct)
	}
}
```

**Step 2: Run tests to verify they fail**

```bash
go test ./pkg/resilience/... -v -race -count=1
```

Expected: FAIL.

**Step 3: Write implementation**

```go
// pkg/resilience/compactor.go
package resilience

import "encoding/json"

// Compact recursively removes nil values, empty slices, and empty maps
// from the given data. Preserves 0, false, and empty strings.
// Operates on map[string]any and []any (JSON-unmarshalled structures).
func Compact(data any) any {
	switch v := data.(type) {
	case map[string]any:
		return compactMap(v)
	case []any:
		return compactSlice(v)
	default:
		return data
	}
}

func compactMap(m map[string]any) any {
	result := make(map[string]any, len(m))
	for k, v := range m {
		if v == nil {
			continue
		}
		compacted := Compact(v)
		if compacted == nil {
			continue
		}
		if s, ok := compacted.([]any); ok && len(s) == 0 {
			continue
		}
		if m2, ok := compacted.(map[string]any); ok && len(m2) == 0 {
			continue
		}
		result[k] = compacted
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func compactSlice(s []any) any {
	var result []any
	for _, v := range s {
		if v == nil {
			continue
		}
		compacted := Compact(v)
		if compacted == nil {
			continue
		}
		if s2, ok := compacted.([]any); ok && len(s2) == 0 {
			continue
		}
		if m, ok := compacted.(map[string]any); ok && len(m) == 0 {
			continue
		}
		result = append(result, compacted)
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

// EstimateSavings returns the percentage of bytes saved by compaction,
// estimated via JSON serialization length.
func EstimateSavings(original, compacted any) float64 {
	origBytes, err := json.Marshal(original)
	if err != nil {
		return 0
	}
	compBytes, err := json.Marshal(compacted)
	if err != nil {
		return 0
	}
	origLen := float64(len(origBytes))
	if origLen == 0 {
		return 0
	}
	return ((origLen - float64(len(compBytes))) / origLen) * 100
}
```

**Step 4: Run tests to verify they pass**

```bash
go test ./pkg/resilience/... -v -race -count=1
```

Expected: PASS.

**Step 5: Commit**

```bash
git add pkg/resilience/compactor.go pkg/resilience/compactor_test.go
git commit -m "feat: add response compactor in pkg/resilience"
```

---

## Task 6: `pkg/resilience/middleware` — Combined Resilience Middleware

**Files:**
- Create: `pkg/resilience/middleware.go`
- Create: `pkg/resilience/middleware_test.go`
- Delete: `pkg/resilience/.gitkeep`

**Step 1: Write the failing tests**

```go
// pkg/resilience/middleware_test.go
package resilience

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestMiddleware_SuccessfulExecution(t *testing.T) {
	mw := NewMiddleware(Config{
		RateLimit:        5000,
		FailureThreshold: 5,
		Cooldown:         30 * time.Second,
		SuccessThreshold: 3,
		Compact:          false,
	})
	result, err := mw.Execute(context.Background(), func() (any, error) {
		return map[string]any{"status": "ok"}, nil
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	m := result.(map[string]any)
	if m["status"] != "ok" {
		t.Errorf("result = %v, want {status: ok}", result)
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
		t.Error("compaction should remove nil values")
	}
	if _, ok := m["c"]; ok {
		t.Error("compaction should remove empty slices")
	}
}

func TestMiddleware_RateLimitExhausted(t *testing.T) {
	mw := NewMiddleware(Config{RateLimit: 1})
	// First call succeeds
	_, err := mw.Execute(context.Background(), func() (any, error) {
		return "ok", nil
	})
	if err != nil {
		t.Fatalf("first Execute failed: %v", err)
	}
	// Second call should be rate limited
	_, err = mw.Execute(context.Background(), func() (any, error) {
		return "ok", nil
	})
	if err == nil {
		t.Error("expected rate limit error")
	}
	if !errors.Is(err, ErrRateLimited) {
		t.Errorf("error = %v, want ErrRateLimited", err)
	}
}

func TestMiddleware_CircuitBreakerOpens(t *testing.T) {
	mw := NewMiddleware(Config{
		FailureThreshold: 2,
		Cooldown:         1 * time.Second,
	})
	fail := errors.New("api error")
	for i := 0; i < 2; i++ {
		mw.Execute(context.Background(), func() (any, error) {
			return nil, fail
		})
	}
	// Circuit should be open now
	_, err := mw.Execute(context.Background(), func() (any, error) {
		return "ok", nil
	})
	if !errors.Is(err, ErrCircuitOpen) {
		t.Errorf("error = %v, want ErrCircuitOpen", err)
	}
}

func TestMiddleware_RecordsSuccessAndFailure(t *testing.T) {
	mw := NewMiddleware(Config{FailureThreshold: 5})
	// Success
	mw.Execute(context.Background(), func() (any, error) {
		return "ok", nil
	})
	state, failures := mw.CircuitBreakerStatus()
	if state != StateClosed || failures != 0 {
		t.Errorf("after success: state=%v failures=%d", state, failures)
	}
	// Failure
	mw.Execute(context.Background(), func() (any, error) {
		return nil, errors.New("fail")
	})
	state, failures = mw.CircuitBreakerStatus()
	if failures != 1 {
		t.Errorf("after failure: failures=%d, want 1", failures)
	}
}

func TestMiddleware_DisabledComponents(t *testing.T) {
	// Zero values disable rate limiter and circuit breaker
	mw := NewMiddleware(Config{})
	for i := 0; i < 100; i++ {
		_, err := mw.Execute(context.Background(), func() (any, error) {
			return "ok", nil
		})
		if err != nil {
			t.Fatalf("Execute %d failed: %v", i, err)
		}
	}
}

func TestMiddleware_ContextCancellation(t *testing.T) {
	mw := NewMiddleware(Config{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := mw.Execute(ctx, func() (any, error) {
		return "ok", nil
	})
	if err == nil {
		t.Error("expected context cancellation error")
	}
}

func TestMiddleware_StatusMethods(t *testing.T) {
	mw := NewMiddleware(Config{RateLimit: 1000, FailureThreshold: 5})
	avail, wt := mw.RateLimiterStatus()
	if avail <= 0 || wt != 0 {
		t.Errorf("RateLimiterStatus = (%f, %v), want (>0, 0)", avail, wt)
	}
	if mw.IsCircuitOpen() {
		t.Error("IsCircuitOpen = true, want false")
	}
}
```

**Step 2: Run tests to verify they fail**

```bash
go test ./pkg/resilience/... -v -race -count=1
```

Expected: FAIL.

**Step 3: Write implementation**

```go
// pkg/resilience/middleware.go
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

// Config configures the resilience middleware.
type Config struct {
	RateLimit        int           // tokens per hour (0 = disabled)
	FailureThreshold int           // circuit breaker failures (0 = disabled)
	Cooldown         time.Duration // circuit breaker cooldown
	SuccessThreshold int           // half-open successes to close
	Compact          bool          // enable response compaction
}

// Middleware combines rate limiting, circuit breaking, and response compaction.
type Middleware struct {
	rateLimiter    *RateLimiter
	circuitBreaker *CircuitBreaker
	compact        bool
}

// NewMiddleware creates a resilience middleware with the given configuration.
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

// Execute runs fn through the middleware chain:
// 1. Context check
// 2. Circuit breaker (fast-fail if open)
// 3. Rate limiter (fail if exhausted)
// 4. Execute fn
// 5. Record success/failure
// 6. Compact response if enabled
func (m *Middleware) Execute(ctx context.Context, fn func() (any, error)) (any, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("context cancelled: %w", err)
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

// IsCircuitOpen returns true if the circuit breaker is in the OPEN state.
func (m *Middleware) IsCircuitOpen() bool {
	if m.circuitBreaker == nil {
		return false
	}
	return m.circuitBreaker.State() == StateOpen
}

// RateLimiterStatus returns current available tokens and wait time.
func (m *Middleware) RateLimiterStatus() (available float64, waitTime time.Duration) {
	if m.rateLimiter == nil {
		return 0, 0
	}
	return m.rateLimiter.Available(), m.rateLimiter.WaitTime()
}

// CircuitBreakerStatus returns the current state and failure count.
func (m *Middleware) CircuitBreakerStatus() (state CircuitState, failures int) {
	if m.circuitBreaker == nil {
		return StateClosed, 0
	}
	m.circuitBreaker.mu.RLock()
	defer m.circuitBreaker.mu.RUnlock()
	return m.circuitBreaker.state, m.circuitBreaker.failures
}
```

**Step 4: Run tests to verify they pass**

```bash
go test ./pkg/resilience/... -v -race -count=1
```

Expected: PASS.

**Step 5: Commit**

```bash
rm pkg/resilience/.gitkeep
git add pkg/resilience/
git commit -m "feat: add resilience middleware combining rate limiter, circuit breaker, compactor"
```

---

## Task 7: `pkg/mcputil/errors` — Standard Error Messages

**Files:**
- Create: `pkg/mcputil/errors.go`

**Step 1: Write implementation** (no tests needed for static declarations)

```go
// pkg/mcputil/errors.go
package mcputil

import "errors"

var (
	ErrNotFound     = errors.New("resource not found")
	ErrRateLimited  = errors.New("rate limit exceeded, retry after wait period")
	ErrCircuitOpen  = errors.New("service temporarily unavailable (circuit breaker open)")
	ErrValidation   = errors.New("input validation failed")
	ErrUnauthorized = errors.New("authentication failed — check credentials")
)
```

**Step 2: Commit**

```bash
git add pkg/mcputil/errors.go
git commit -m "feat: add standard MCP error messages in pkg/mcputil"
```

---

## Task 8: `pkg/mcputil/result` — Response Builders

**Files:**
- Create: `pkg/mcputil/result.go`
- Create: `pkg/mcputil/result_test.go`

**Step 1: Write the failing tests**

```go
// pkg/mcputil/result_test.go
package mcputil

import (
	"errors"
	"testing"
)

func TestTextResult(t *testing.T) {
	r := TextResult("hello")
	if r.IsError {
		t.Error("TextResult should not be an error")
	}
	if len(r.Content) != 1 {
		t.Fatalf("content len = %d, want 1", len(r.Content))
	}
}

func TestErrorResult(t *testing.T) {
	r := ErrorResult(errors.New("something broke"))
	if !r.IsError {
		t.Error("ErrorResult should be an error")
	}
}

func TestJSONResult(t *testing.T) {
	data := map[string]any{"id": 1, "name": "test"}
	r := JSONResult(data)
	if r.IsError {
		t.Error("JSONResult should not be an error")
	}
	if len(r.Content) != 1 {
		t.Fatalf("content len = %d, want 1", len(r.Content))
	}
}

func TestJSONResult_MarshalError(t *testing.T) {
	// Channels cannot be marshalled to JSON
	r := JSONResult(make(chan int))
	if !r.IsError {
		t.Error("JSONResult with unmarshalable data should be an error")
	}
}
```

**Step 2: Run tests to verify they fail**

```bash
go test ./pkg/mcputil/... -v -race -count=1
```

Expected: FAIL.

**Step 3: Write implementation**

```go
// pkg/mcputil/result.go
package mcputil

import (
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

// TextResult returns a successful text response.
func TextResult(text string) *mcp.CallToolResult {
	return mcp.NewToolResultText(text)
}

// ErrorResult returns an error response from an error.
func ErrorResult(err error) *mcp.CallToolResult {
	return mcp.NewToolResultError(err.Error())
}

// JSONResult returns a successful response with JSON-serialized data.
// If marshalling fails, returns an error result instead.
func JSONResult(data any) *mcp.CallToolResult {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal response: %v", err))
	}
	return mcp.NewToolResultText(string(b))
}
```

**Step 4: Run tests to verify they pass**

```bash
go test ./pkg/mcputil/... -v -race -count=1
```

Expected: PASS.

**Step 5: Commit**

```bash
git add pkg/mcputil/result.go pkg/mcputil/result_test.go pkg/mcputil/errors.go
git commit -m "feat: add MCP result builders and error constants in pkg/mcputil"
```

---

## Task 9: `pkg/mcputil/annotations` — Tool Annotations

**Files:**
- Create: `pkg/mcputil/annotations.go`
- Create: `pkg/mcputil/annotations_test.go`

**Step 1: Write the failing tests**

```go
// pkg/mcputil/annotations_test.go
package mcputil

import (
	"testing"
)

func TestAnnotations(t *testing.T) {
	tests := []struct {
		name string
		ann  ToolAnnotations
	}{
		{"ReadOnly", ReadOnly()},
		{"Destructive", Destructive()},
		{"Idempotent", Idempotent()},
		{"OpenWorld", OpenWorld()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.ann.Annotation == nil {
				t.Error("annotation should not be nil")
			}
		})
	}
}

func TestReadOnly_Values(t *testing.T) {
	ann := ReadOnly()
	if ann.Annotation.ReadOnly == nil || !*ann.Annotation.ReadOnly {
		t.Error("ReadOnly annotation should have readOnly=true")
	}
	if ann.Annotation.Destructive == nil || *ann.Annotation.Destructive {
		t.Error("ReadOnly annotation should have destructive=false")
	}
}

func TestDestructive_Values(t *testing.T) {
	ann := Destructive()
	if ann.Annotation.Destructive == nil || !*ann.Annotation.Destructive {
		t.Error("Destructive annotation should have destructive=true")
	}
	if ann.Annotation.ReadOnly == nil || *ann.Annotation.ReadOnly {
		t.Error("Destructive annotation should have readOnly=false")
	}
}
```

**Step 2: Run tests to verify they fail**

```bash
go test ./pkg/mcputil/... -v -race -count=1
```

Expected: FAIL.

**Step 3: Write implementation**

```go
// pkg/mcputil/annotations.go
package mcputil

import "github.com/mark3labs/mcp-go/mcp"

// ToolAnnotations wraps mcp.ToolAnnotation for convenient construction.
type ToolAnnotations struct {
	Annotation *mcp.ToolAnnotation
}

func boolPtr(b bool) *bool { return &b }

// ReadOnly returns annotations marking a tool as read-only and non-destructive.
func ReadOnly() ToolAnnotations {
	return ToolAnnotations{Annotation: &mcp.ToolAnnotation{
		ReadOnly:    boolPtr(true),
		Destructive: boolPtr(false),
	}}
}

// Destructive returns annotations marking a tool as destructive.
func Destructive() ToolAnnotations {
	return ToolAnnotations{Annotation: &mcp.ToolAnnotation{
		ReadOnly:    boolPtr(false),
		Destructive: boolPtr(true),
	}}
}

// Idempotent returns annotations marking a tool as idempotent.
func Idempotent() ToolAnnotations {
	return ToolAnnotations{Annotation: &mcp.ToolAnnotation{
		Idempotent: boolPtr(true),
	}}
}

// OpenWorld returns annotations marking a tool as depending on external state.
func OpenWorld() ToolAnnotations {
	return ToolAnnotations{Annotation: &mcp.ToolAnnotation{
		OpenWorld: boolPtr(true),
	}}
}
```

**Step 4: Run tests to verify they pass**

```bash
go test ./pkg/mcputil/... -v -race -count=1
```

Expected: PASS.

**Step 5: Commit**

```bash
git add pkg/mcputil/annotations.go pkg/mcputil/annotations_test.go
git commit -m "feat: add tool annotation helpers in pkg/mcputil"
```

---

## Task 10: `pkg/mcputil/htmlstrip` — HTML to Plaintext

**Files:**
- Create: `pkg/mcputil/htmlstrip.go`
- Create: `pkg/mcputil/htmlstrip_test.go`

**Step 1: Write the failing tests**

```go
// pkg/mcputil/htmlstrip_test.go
package mcputil

import (
	"strings"
	"testing"
)

func TestStripHTML_Basic(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"plain text", "hello world", "hello world"},
		{"simple tag", "<b>bold</b>", "bold"},
		{"paragraph", "<p>first</p><p>second</p>", "first\n\nsecond"},
		{"br tag", "line1<br>line2", "line1\nline2"},
		{"br self-closing", "line1<br/>line2", "line1\nline2"},
		{"br with space", "line1<br />line2", "line1\nline2"},
		{"list items", "<ul><li>one</li><li>two</li></ul>", "\none\ntwo"},
		{"nested", "<div><p>hello <b>world</b></p></div>", "hello world"},
		{"empty", "", ""},
		{"entities", "&amp; &lt; &gt; &quot;", "& < > \""},
		{"nbsp", "hello&nbsp;world", "hello world"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StripHTML(tt.input)
			if got != tt.want {
				t.Errorf("StripHTML(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestStripHTMLWithLimit(t *testing.T) {
	long := "<p>" + strings.Repeat("a", 100) + "</p>"
	got := StripHTMLWithLimit(long, 50)
	if len(got) > 50 {
		t.Errorf("len = %d, want <= 50", len(got))
	}
}

func TestStripHTMLWithLimit_Default(t *testing.T) {
	long := "<p>" + strings.Repeat("a", 30000) + "</p>"
	got := StripHTML(long)
	if len(got) > 25000 {
		t.Errorf("default limit exceeded: len = %d, want <= 25000", len(got))
	}
}

func TestStripHTML_WhitespaceCollapse(t *testing.T) {
	input := "<p>  lots   of    spaces  </p>"
	got := StripHTML(input)
	if strings.Contains(got, "  ") {
		t.Errorf("should collapse whitespace: %q", got)
	}
}
```

**Step 2: Run tests to verify they fail**

```bash
go test ./pkg/mcputil/... -v -race -count=1
```

Expected: FAIL.

**Step 3: Write implementation**

```go
// pkg/mcputil/htmlstrip.go
package mcputil

import (
	"regexp"
	"strings"
)

const defaultMaxChars = 25000

var (
	brRe        = regexp.MustCompile(`<br\s*/?>`)
	pCloseRe    = regexp.MustCompile(`</p>`)
	pOpenRe     = regexp.MustCompile(`<p[^>]*>`)
	liRe        = regexp.MustCompile(`<li[^>]*>`)
	tagRe       = regexp.MustCompile(`<[^>]+>`)
	spacesRe    = regexp.MustCompile(`[ \t]+`)
	blankLinesRe = regexp.MustCompile(`\n{3,}`)
)

var entities = strings.NewReplacer(
	"&amp;", "&",
	"&lt;", "<",
	"&gt;", ">",
	"&quot;", "\"",
	"&#39;", "'",
	"&apos;", "'",
	"&nbsp;", " ",
)

// StripHTML converts HTML to plaintext with a default 25,000 character limit.
func StripHTML(html string) string {
	return StripHTMLWithLimit(html, defaultMaxChars)
}

// StripHTMLWithLimit converts HTML to plaintext, truncating to maxChars.
func StripHTMLWithLimit(html string, maxChars int) string {
	if html == "" {
		return ""
	}

	s := html

	// Convert block elements to newlines
	s = brRe.ReplaceAllString(s, "\n")
	s = pCloseRe.ReplaceAllString(s, "\n\n")
	s = pOpenRe.ReplaceAllString(s, "")
	s = liRe.ReplaceAllString(s, "\n")

	// Strip remaining tags
	s = tagRe.ReplaceAllString(s, "")

	// Decode entities
	s = entities.Replace(s)

	// Collapse whitespace (not newlines)
	s = spacesRe.ReplaceAllString(s, " ")

	// Collapse excessive blank lines
	s = blankLinesRe.ReplaceAllString(s, "\n\n")

	s = strings.TrimSpace(s)

	if maxChars > 0 && len(s) > maxChars {
		s = s[:maxChars]
	}

	return s
}
```

**Step 4: Run tests to verify they pass**

```bash
go test ./pkg/mcputil/... -v -race -count=1
```

Expected: PASS.

**Step 5: Commit**

```bash
git add pkg/mcputil/htmlstrip.go pkg/mcputil/htmlstrip_test.go
git commit -m "feat: add HTML-to-plaintext converter in pkg/mcputil"
```

---

## Task 11: `pkg/mcputil/formatter` — Entity-Aware Response Formatter

**Files:**
- Create: `pkg/mcputil/formatter.go`
- Create: `pkg/mcputil/formatter_test.go`
- Delete: `pkg/mcputil/.gitkeep`

**Step 1: Write the failing tests**

```go
// pkg/mcputil/formatter_test.go
package mcputil

import (
	"strings"
	"testing"
)

func TestFormatCompact(t *testing.T) {
	data := []map[string]any{
		{"id": 1, "ticketNumber": "T001", "title": "Bug", "extra": "hidden"},
		{"id": 2, "ticketNumber": "T002", "title": "Feature", "extra": "hidden"},
	}
	fields := FieldSet{
		"Ticket": {"id", "ticketNumber", "title"},
	}
	result := FormatCompact("Ticket", data, fields)
	if !strings.Contains(result, "T001") {
		t.Error("should contain ticketNumber T001")
	}
	if !strings.Contains(result, "T002") {
		t.Error("should contain ticketNumber T002")
	}
	if strings.Contains(result, "hidden") {
		t.Error("should not contain non-essential field 'extra'")
	}
}

func TestFormatCompact_UnknownEntity(t *testing.T) {
	data := []map[string]any{
		{"id": 1, "name": "test"},
	}
	result := FormatCompact("Unknown", data, FieldSet{})
	if !strings.Contains(result, "test") {
		t.Error("unknown entity should include all fields")
	}
}

func TestFormatFull(t *testing.T) {
	data := map[string]any{"id": 1, "name": "test", "active": true}
	result := FormatFull(data)
	if !strings.Contains(result, "name") || !strings.Contains(result, "test") {
		t.Error("FormatFull should include all fields")
	}
}

func TestWithPagination(t *testing.T) {
	result := WithPagination("data here", 1, 5, 25)
	if !strings.Contains(result, "Page 1/5") {
		t.Error("should contain page info")
	}
	if !strings.Contains(result, "25") {
		t.Error("should contain total count")
	}
}

func TestWithNames(t *testing.T) {
	data := map[string]any{
		"companyID":          42,
		"assignedResourceID": 7,
	}
	names := map[string]string{
		"companyID":          "Acme Corp",
		"assignedResourceID": "John Doe",
	}
	result := WithNames(data, names)
	if result["companyName"] != "Acme Corp" {
		t.Errorf("companyName = %v, want Acme Corp", result["companyName"])
	}
	if result["assignedResourceName"] != "John Doe" {
		t.Errorf("assignedResourceName = %v, want John Doe", result["assignedResourceName"])
	}
}
```

**Step 2: Run tests to verify they fail**

```bash
go test ./pkg/mcputil/... -v -race -count=1
```

Expected: FAIL.

**Step 3: Write implementation**

```go
// pkg/mcputil/formatter.go
package mcputil

import (
	"encoding/json"
	"fmt"
	"strings"
)

// FieldSet maps entity types to their essential field names.
type FieldSet map[string][]string

// DefaultFields contains essential fields per entity type.
var DefaultFields = FieldSet{
	"Ticket":      {"id", "ticketNumber", "title", "status", "priority", "companyID", "assignedResourceID", "createDate", "dueDateTime"},
	"Company":     {"id", "companyName", "isActive", "phone", "city", "state"},
	"Contact":     {"id", "firstName", "lastName", "emailAddress", "companyID"},
	"Project":     {"id", "projectName", "status", "companyID", "projectLeadResourceID", "startDate", "endDate"},
	"Task":        {"id", "title", "status", "projectID", "assignedResourceID", "percentComplete"},
	"Resource":    {"id", "firstName", "lastName", "email", "isActive"},
	"TimeEntry":   {"id", "resourceID", "ticketID", "dateWorked", "hoursWorked", "summaryNotes"},
	"BillingItem": {"id", "itemName", "companyID", "ticketID", "postedDate", "totalAmount", "invoiceID"},
}

// nameMapping maps ID field names to their resolved name field names.
var nameMapping = map[string]string{
	"companyID":          "companyName",
	"assignedResourceID": "assignedResourceName",
	"resourceID":         "resourceName",
	"projectLeadResourceID": "projectLeadName",
}

// FormatCompact returns a compact string representation of entities,
// showing only essential fields for the given entity type.
func FormatCompact(entityType string, data []map[string]any, fields FieldSet) string {
	essentialFields, ok := fields[entityType]
	if !ok {
		essentialFields, ok = DefaultFields[entityType]
	}

	var sb strings.Builder
	for i, item := range data {
		if i > 0 {
			sb.WriteString("\n---\n")
		}
		if ok {
			for _, f := range essentialFields {
				if v, exists := item[f]; exists && v != nil {
					fmt.Fprintf(&sb, "%s: %v\n", f, v)
				}
			}
			// Also include resolved names if present
			for _, nameField := range nameMapping {
				if v, exists := item[nameField]; exists && v != nil {
					fmt.Fprintf(&sb, "%s: %v\n", nameField, v)
				}
			}
		} else {
			// Unknown entity — include all fields
			b, _ := json.MarshalIndent(item, "", "  ")
			sb.Write(b)
			sb.WriteByte('\n')
		}
	}
	return strings.TrimSpace(sb.String())
}

// FormatFull returns a full JSON representation of a single entity.
func FormatFull(data map[string]any) string {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Sprintf("error formatting response: %v", err)
	}
	return string(b)
}

// WithPagination appends pagination info to the response text.
func WithPagination(text string, current, total, count int) string {
	return fmt.Sprintf("%s\n\n--- Page %d/%d | %d total results ---", text, current, total, count)
}

// WithNames enriches a data map with resolved names from a name mapping.
func WithNames(data map[string]any, names map[string]string) map[string]any {
	result := make(map[string]any, len(data)+len(names))
	for k, v := range data {
		result[k] = v
	}
	for idField, resolvedName := range names {
		if nameField, ok := nameMapping[idField]; ok {
			result[nameField] = resolvedName
		}
	}
	return result
}
```

**Step 4: Run tests to verify they pass**

```bash
go test ./pkg/mcputil/... -v -race -count=1
```

Expected: PASS.

**Step 5: Commit**

```bash
rm pkg/mcputil/.gitkeep
git add pkg/mcputil/
git commit -m "feat: add entity-aware response formatter in pkg/mcputil"
```

---

## Task 12: `pkg/apihelper/client` — Shared HTTP Client

**Files:**
- Create: `pkg/apihelper/client.go`
- Create: `pkg/apihelper/client_test.go`

**Step 1: Write the failing tests**

```go
// pkg/apihelper/client_test.go
package apihelper

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestClient_Get(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Query().Get("filter") != "active" {
			t.Error("missing query param")
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer srv.Close()

	c := NewClient(ClientConfig{BaseURL: srv.URL, Timeout: 5 * time.Second})
	body, err := c.Get(context.Background(), "/test", map[string]string{"filter": "active"})
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	var result map[string]string
	json.Unmarshal(body, &result)
	if result["status"] != "ok" {
		t.Errorf("status = %s, want ok", result["status"])
	}
}

func TestClient_Post(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("missing content-type header")
		}
		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		if body["name"] != "test" {
			t.Error("body not decoded correctly")
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{"id": 1})
	}))
	defer srv.Close()

	c := NewClient(ClientConfig{BaseURL: srv.URL})
	body, err := c.Post(context.Background(), "/items", map[string]string{"name": "test"})
	if err != nil {
		t.Fatalf("Post failed: %v", err)
	}
	var result map[string]any
	json.Unmarshal(body, &result)
	if result["id"] != float64(1) {
		t.Errorf("id = %v, want 1", result["id"])
	}
}

func TestClient_Patch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("method = %s, want PATCH", r.Method)
		}
		json.NewEncoder(w).Encode(map[string]string{"updated": "true"})
	}))
	defer srv.Close()

	c := NewClient(ClientConfig{BaseURL: srv.URL})
	_, err := c.Patch(context.Background(), "/items/1", map[string]string{"name": "updated"})
	if err != nil {
		t.Fatalf("Patch failed: %v", err)
	}
}

func TestClient_Delete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := NewClient(ClientConfig{BaseURL: srv.URL})
	err := c.Delete(context.Background(), "/items/1")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
}

func TestClient_CustomHeaders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Api-Key") != "secret" {
			t.Error("custom header not sent")
		}
		if r.Header.Get("User-Agent") != "test-agent" {
			t.Error("user-agent not set")
		}
		w.WriteHeader(200)
	}))
	defer srv.Close()

	c := NewClient(ClientConfig{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
		Headers:   map[string]string{"X-Api-Key": "secret"},
	})
	c.Get(context.Background(), "/test", nil)
}

func TestClient_Retry429(t *testing.T) {
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if attempts.Add(1) <= 2 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"ok": "true"})
	}))
	defer srv.Close()

	c := NewClient(ClientConfig{BaseURL: srv.URL, MaxRetries: 3})
	_, err := c.Get(context.Background(), "/test", nil)
	if err != nil {
		t.Fatalf("should succeed after retries: %v", err)
	}
	if attempts.Load() != 3 {
		t.Errorf("attempts = %d, want 3", attempts.Load())
	}
}

func TestClient_Retry5xx(t *testing.T) {
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if attempts.Add(1) <= 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"ok": "true"})
	}))
	defer srv.Close()

	c := NewClient(ClientConfig{BaseURL: srv.URL, MaxRetries: 2})
	_, err := c.Get(context.Background(), "/test", nil)
	if err != nil {
		t.Fatalf("should succeed after retry: %v", err)
	}
}

func TestClient_NoRetry4xx(t *testing.T) {
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "bad request"}`))
	}))
	defer srv.Close()

	c := NewClient(ClientConfig{BaseURL: srv.URL, MaxRetries: 3})
	_, err := c.Get(context.Background(), "/test", nil)
	if err == nil {
		t.Error("should return error for 400")
	}
	if attempts.Load() != 1 {
		t.Errorf("should not retry 4xx: attempts = %d", attempts.Load())
	}
}

func TestClient_ContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
	}))
	defer srv.Close()

	c := NewClient(ClientConfig{BaseURL: srv.URL})
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err := c.Get(ctx, "/test", nil)
	if err == nil {
		t.Error("should fail with cancelled context")
	}
}
```

**Step 2: Run tests to verify they fail**

```bash
go test ./pkg/apihelper/... -v -race -count=1
```

Expected: FAIL.

**Step 3: Write implementation**

```go
// pkg/apihelper/client.go
package apihelper

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand/v2"
	"net/http"
	"time"
)

// ClientConfig configures the shared HTTP client.
type ClientConfig struct {
	BaseURL    string
	Timeout    time.Duration
	MaxRetries int
	UserAgent  string
	Headers    map[string]string
}

// Client is a shared HTTP client with retry, timeout, and custom headers.
type Client struct {
	http    *http.Client
	baseURL string
	retries int
	agent   string
	headers map[string]string
}

// NewClient creates a configured HTTP client.
func NewClient(cfg ClientConfig) *Client {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	retries := cfg.MaxRetries
	if retries == 0 {
		retries = 3
	}
	agent := cfg.UserAgent
	if agent == "" {
		agent = "lgp-mcp/dev"
	}
	return &Client{
		http:    &http.Client{Timeout: timeout},
		baseURL: cfg.BaseURL,
		retries: retries,
		agent:   agent,
		headers: cfg.Headers,
	}
}

// Get performs a GET request with optional query parameters.
func (c *Client) Get(ctx context.Context, path string, params map[string]string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	if params != nil {
		q := req.URL.Query()
		for k, v := range params {
			q.Set(k, v)
		}
		req.URL.RawQuery = q.Encode()
	}
	return c.do(req)
}

// Post performs a POST request with a JSON body.
func (c *Client) Post(ctx context.Context, path string, body any) ([]byte, error) {
	return c.doWithBody(ctx, http.MethodPost, path, body)
}

// Patch performs a PATCH request with a JSON body.
func (c *Client) Patch(ctx context.Context, path string, body any) ([]byte, error) {
	return c.doWithBody(ctx, http.MethodPatch, path, body)
}

// Delete performs a DELETE request.
func (c *Client) Delete(ctx context.Context, path string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	_, err = c.do(req)
	return err
}

func (c *Client) doWithBody(ctx context.Context, method, path string, body any) ([]byte, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshalling body: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	return c.do(req)
}

func (c *Client) do(req *http.Request) ([]byte, error) {
	req.Header.Set("User-Agent", c.agent)
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}

	var lastErr error
	for attempt := 0; attempt <= c.retries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(math.Pow(2, float64(attempt-1))) * 100 * time.Millisecond
			jitter := time.Duration(rand.IntN(int(backoff/2 + 1)))
			select {
			case <-req.Context().Done():
				return nil, req.Context().Err()
			case <-time.After(backoff + jitter):
			}
			// Clone the request for retry (body needs re-reading)
			var bodyBytes []byte
			if req.Body != nil {
				bodyBytes, _ = io.ReadAll(req.Body)
				req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			}
		}

		resp, err := c.http.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			if req.Context().Err() != nil {
				return nil, lastErr
			}
			continue
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return body, nil
		}

		// Only retry on 429 and 5xx
		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
			continue
		}

		// Non-retryable error (4xx except 429)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}
```

**Step 4: Run tests to verify they pass**

```bash
go test ./pkg/apihelper/... -v -race -count=1
```

Expected: PASS.

**Step 5: Commit**

```bash
git add pkg/apihelper/client.go pkg/apihelper/client_test.go
git commit -m "feat: add shared HTTP client with retry in pkg/apihelper"
```

---

## Task 13: `pkg/apihelper/oauth` — OAuth2 Token Manager

**Files:**
- Create: `pkg/apihelper/oauth.go`
- Create: `pkg/apihelper/oauth_test.go`

**Step 1: Write the failing tests**

```go
// pkg/apihelper/oauth_test.go
package apihelper

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestTokenManager_FetchesToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != "key" || pass != "secret" {
			w.WriteHeader(401)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{
			"access_token": "tok123",
			"token_type":   "Bearer",
			"expires_in":   3600,
		})
	}))
	defer srv.Close()

	tm := NewTokenManager(OAuth2Config{
		TokenURL:     srv.URL + "/token",
		ClientID:     "key",
		ClientSecret: "secret",
	})
	tok, err := tm.Token(context.Background())
	if err != nil {
		t.Fatalf("Token() failed: %v", err)
	}
	if tok != "tok123" {
		t.Errorf("token = %q, want tok123", tok)
	}
}

func TestTokenManager_CachesToken(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		json.NewEncoder(w).Encode(map[string]any{
			"access_token": "tok",
			"expires_in":   3600,
		})
	}))
	defer srv.Close()

	tm := NewTokenManager(OAuth2Config{TokenURL: srv.URL, ClientID: "k", ClientSecret: "s"})
	tm.Token(context.Background())
	tm.Token(context.Background())
	if calls.Load() != 1 {
		t.Errorf("token endpoint called %d times, want 1 (should be cached)", calls.Load())
	}
}

func TestTokenManager_RefreshesExpired(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		json.NewEncoder(w).Encode(map[string]any{
			"access_token": "tok",
			"expires_in":   1, // expires in 1 second
		})
	}))
	defer srv.Close()

	tm := NewTokenManager(OAuth2Config{TokenURL: srv.URL, ClientID: "k", ClientSecret: "s"})
	tm.Token(context.Background())
	// Force expiry by manipulating internal state
	tm.mu.Lock()
	tm.expiry = time.Now().Add(-1 * time.Minute)
	tm.mu.Unlock()
	tm.Token(context.Background())
	if calls.Load() != 2 {
		t.Errorf("token endpoint called %d times, want 2 (should refresh)", calls.Load())
	}
}

func TestTokenManager_DeduplicatesConcurrent(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		time.Sleep(50 * time.Millisecond)
		json.NewEncoder(w).Encode(map[string]any{
			"access_token": "tok",
			"expires_in":   3600,
		})
	}))
	defer srv.Close()

	tm := NewTokenManager(OAuth2Config{TokenURL: srv.URL, ClientID: "k", ClientSecret: "s"})
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			tm.Token(context.Background())
		}()
	}
	wg.Wait()
	if calls.Load() != 1 {
		t.Errorf("concurrent calls = %d, want 1 (singleflight)", calls.Load())
	}
}
```

**Step 2: Run tests to verify they fail**

```bash
go test ./pkg/apihelper/... -v -race -count=1
```

Expected: FAIL.

**Step 3: Write implementation**

```go
// pkg/apihelper/oauth.go
package apihelper

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

// OAuth2Config configures the OAuth2 token manager.
type OAuth2Config struct {
	TokenURL     string
	ClientID     string // API Key
	ClientSecret string // API Secret
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

// TokenManager handles OAuth2 token caching and refresh.
type TokenManager struct {
	cfg    OAuth2Config
	mu     sync.Mutex
	token  string
	expiry time.Time
	group  singleflight.Group
}

const tokenRefreshBuffer = 5 * time.Minute

// NewTokenManager creates a token manager with the given OAuth2 configuration.
func NewTokenManager(cfg OAuth2Config) *TokenManager {
	return &TokenManager{cfg: cfg}
}

// Token returns a valid access token, refreshing if necessary.
// Concurrent calls are deduplicated via singleflight.
func (t *TokenManager) Token(ctx context.Context) (string, error) {
	t.mu.Lock()
	if t.token != "" && time.Now().Before(t.expiry.Add(-tokenRefreshBuffer)) {
		tok := t.token
		t.mu.Unlock()
		return tok, nil
	}
	t.mu.Unlock()

	result, err, _ := t.group.Do("token", func() (any, error) {
		return t.fetchToken(ctx)
	})
	if err != nil {
		return "", err
	}
	return result.(string), nil
}

func (t *TokenManager) fetchToken(ctx context.Context) (string, error) {
	data := url.Values{"grant_type": {"client_credentials"}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.cfg.TokenURL,
		strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("creating token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(t.cfg.ClientID, t.cfg.ClientSecret)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token request returned HTTP %d: %s", resp.StatusCode, string(body))
	}

	var tok tokenResponse
	if err := json.Unmarshal(body, &tok); err != nil {
		return "", fmt.Errorf("parsing token response: %w", err)
	}

	t.mu.Lock()
	t.token = tok.AccessToken
	t.expiry = time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second)
	t.mu.Unlock()

	return tok.AccessToken, nil
}
```

**Step 4: Run tests to verify they pass**

```bash
go test ./pkg/apihelper/... -v -race -count=1
```

Expected: PASS.

**Step 5: Commit**

```bash
git add pkg/apihelper/oauth.go pkg/apihelper/oauth_test.go
git commit -m "feat: add OAuth2 token manager with singleflight in pkg/apihelper"
```

---

## Task 14: `pkg/apihelper/jsonapi` — JSON:API Parser

**Files:**
- Create: `pkg/apihelper/jsonapi.go`
- Create: `pkg/apihelper/jsonapi_test.go`

**Step 1: Write the failing tests**

```go
// pkg/apihelper/jsonapi_test.go
package apihelper

import (
	"encoding/json"
	"testing"
)

func TestParseJSONAPIResponse_SingleResource(t *testing.T) {
	raw := `{
		"data": {
			"id": "42",
			"type": "organizations",
			"attributes": {
				"name": "Acme",
				"short-name": "acme"
			}
		}
	}`
	resp, err := ParseJSONAPIResponse([]byte(raw))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("data len = %d, want 1", len(resp.Data))
	}
	if resp.Data[0].ID != "42" {
		t.Errorf("id = %s, want 42", resp.Data[0].ID)
	}
	if resp.Data[0].Attributes["name"] != "Acme" {
		t.Error("attributes not parsed")
	}
}

func TestParseJSONAPIResponse_Collection(t *testing.T) {
	raw := `{
		"data": [
			{"id": "1", "type": "organizations", "attributes": {"name": "A"}},
			{"id": "2", "type": "organizations", "attributes": {"name": "B"}}
		],
		"meta": {
			"current-page": 1,
			"next-page": 2,
			"total-pages": 3,
			"total-count": 25
		}
	}`
	resp, err := ParseJSONAPIResponse([]byte(raw))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(resp.Data) != 2 {
		t.Fatalf("data len = %d, want 2", len(resp.Data))
	}
	if resp.Meta.CurrentPage != 1 || resp.Meta.TotalPages != 3 {
		t.Errorf("meta = %+v", resp.Meta)
	}
}

func TestBuildFilterParams(t *testing.T) {
	params := BuildFilterParams(map[string]string{
		"organization-id": "42",
		"name":            "test",
	})
	if params.Get("filter[organization-id]") != "42" {
		t.Error("filter param not set correctly")
	}
	if params.Get("filter[name]") != "test" {
		t.Error("filter param not set correctly")
	}
}

func TestFlattenResource(t *testing.T) {
	res := JSONAPIResource{
		ID:   "5",
		Type: "configurations",
		Attributes: map[string]any{
			"name":    "Server01",
			"status":  "active",
		},
	}
	flat := FlattenResource(res)
	if flat["id"] != "5" {
		t.Error("id not set")
	}
	if flat["type"] != "configurations" {
		t.Error("type not set")
	}
	if flat["name"] != "Server01" {
		t.Error("attributes not flattened")
	}
}

func TestParseJSONAPIResponse_Empty(t *testing.T) {
	raw := `{"data": []}`
	resp, err := ParseJSONAPIResponse([]byte(raw))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(resp.Data) != 0 {
		t.Errorf("data len = %d, want 0", len(resp.Data))
	}
}

func TestParseJSONAPIResponse_Invalid(t *testing.T) {
	_, err := ParseJSONAPIResponse([]byte(`{invalid`))
	if err == nil {
		t.Error("should return error for invalid JSON")
	}
}

func TestKebabToCamel(t *testing.T) {
	tests := []struct{ in, want string }{
		{"short-name", "shortName"},
		{"organization-id", "organizationId"},
		{"name", "name"},
		{"a-b-c", "aBC"},
	}
	for _, tt := range tests {
		got := kebabToCamel(tt.in)
		if got != tt.want {
			t.Errorf("kebabToCamel(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

// Verify JSON output is valid
func TestFlattenResource_JSONSerializable(t *testing.T) {
	res := JSONAPIResource{ID: "1", Type: "test", Attributes: map[string]any{"a": "b"}}
	flat := FlattenResource(res)
	_, err := json.Marshal(flat)
	if err != nil {
		t.Fatalf("flat resource not JSON-serializable: %v", err)
	}
}
```

**Step 2: Run tests to verify they fail**

```bash
go test ./pkg/apihelper/... -v -race -count=1
```

Expected: FAIL.

**Step 3: Write implementation**

```go
// pkg/apihelper/jsonapi.go
package apihelper

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"unicode"
)

// JSONAPIResource represents a single JSON:API resource.
type JSONAPIResource struct {
	ID         string         `json:"id"`
	Type       string         `json:"type"`
	Attributes map[string]any `json:"attributes"`
}

// PaginationMeta holds pagination info from a JSON:API response.
type PaginationMeta struct {
	CurrentPage int `json:"current-page"`
	NextPage    int `json:"next-page"`
	PrevPage    int `json:"prev-page"`
	TotalPages  int `json:"total-pages"`
	TotalCount  int `json:"total-count"`
}

// JSONAPIResponse is the parsed result of a JSON:API response.
type JSONAPIResponse struct {
	Data []JSONAPIResource
	Meta PaginationMeta
}

// rawJSONAPIResponse handles the polymorphic "data" field (single or array).
type rawJSONAPIResponse struct {
	Data json.RawMessage `json:"data"`
	Meta PaginationMeta  `json:"meta"`
}

// ParseJSONAPIResponse parses a JSON:API response body.
// Handles both single-resource and collection responses.
func ParseJSONAPIResponse(body []byte) (*JSONAPIResponse, error) {
	var raw rawJSONAPIResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("parsing JSON:API response: %w", err)
	}

	resp := &JSONAPIResponse{Meta: raw.Meta}

	// Try array first
	var resources []JSONAPIResource
	if err := json.Unmarshal(raw.Data, &resources); err == nil {
		resp.Data = resources
		return resp, nil
	}

	// Try single resource
	var single JSONAPIResource
	if err := json.Unmarshal(raw.Data, &single); err == nil {
		resp.Data = []JSONAPIResource{single}
		return resp, nil
	}

	// Empty or null data
	resp.Data = []JSONAPIResource{}
	return resp, nil
}

// FlattenResource merges a JSON:API resource's id, type, and attributes into a flat map.
func FlattenResource(r JSONAPIResource) map[string]any {
	flat := make(map[string]any, len(r.Attributes)+2)
	flat["id"] = r.ID
	flat["type"] = r.Type
	for k, v := range r.Attributes {
		flat[k] = v
	}
	return flat
}

// BuildFilterParams creates URL query parameters from a filter map.
// Keys are wrapped in filter[] notation: {"name": "x"} -> filter[name]=x.
func BuildFilterParams(filters map[string]string) url.Values {
	params := url.Values{}
	for k, v := range filters {
		params.Set(fmt.Sprintf("filter[%s]", k), v)
	}
	return params
}

// kebabToCamel converts kebab-case to camelCase.
func kebabToCamel(s string) string {
	parts := strings.Split(s, "-")
	for i := 1; i < len(parts); i++ {
		if len(parts[i]) > 0 {
			runes := []rune(parts[i])
			runes[0] = unicode.ToUpper(runes[0])
			parts[i] = string(runes)
		}
	}
	return strings.Join(parts, "")
}
```

**Step 4: Run tests to verify they pass**

```bash
go test ./pkg/apihelper/... -v -race -count=1
```

Expected: PASS.

**Step 5: Commit**

```bash
git add pkg/apihelper/jsonapi.go pkg/apihelper/jsonapi_test.go
git commit -m "feat: add JSON:API parser for IT Glue in pkg/apihelper"
```

---

## Task 15: `pkg/apihelper/mapping` — Generic ID-to-Name Cache

**Files:**
- Create: `pkg/apihelper/mapping.go`
- Create: `pkg/apihelper/mapping_test.go`

**Step 1: Write the failing tests**

```go
// pkg/apihelper/mapping_test.go
package apihelper

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestMappingCache_GetAndCache(t *testing.T) {
	var calls atomic.Int32
	mc := NewMappingCache[int, string](30 * time.Minute)
	fetch := func(id int) (string, error) {
		calls.Add(1)
		return "Company A", nil
	}
	v1, err := mc.Get(context.Background(), 1, fetch)
	if err != nil || v1 != "Company A" {
		t.Fatalf("Get = (%q, %v), want (Company A, nil)", v1, err)
	}
	v2, _ := mc.Get(context.Background(), 1, fetch)
	if v2 != "Company A" {
		t.Error("cached value should be returned")
	}
	if calls.Load() != 1 {
		t.Errorf("fetch called %d times, want 1", calls.Load())
	}
}

func TestMappingCache_TTLExpiry(t *testing.T) {
	var calls atomic.Int32
	mc := NewMappingCache[int, string](50 * time.Millisecond)
	fetch := func(id int) (string, error) {
		calls.Add(1)
		return "value", nil
	}
	mc.Get(context.Background(), 1, fetch)
	time.Sleep(60 * time.Millisecond)
	mc.Get(context.Background(), 1, fetch)
	if calls.Load() != 2 {
		t.Errorf("fetch called %d times after TTL, want 2", calls.Load())
	}
}

func TestMappingCache_Warm(t *testing.T) {
	mc := NewMappingCache[int, string](30 * time.Minute)
	err := mc.Warm(context.Background(), func() (map[int]string, error) {
		return map[int]string{1: "A", 2: "B", 3: "C"}, nil
	})
	if err != nil {
		t.Fatalf("Warm failed: %v", err)
	}
	var fetchCalled bool
	v, _ := mc.Get(context.Background(), 2, func(int) (string, error) {
		fetchCalled = true
		return "", nil
	})
	if fetchCalled {
		t.Error("fetch should not be called for warmed key")
	}
	if v != "B" {
		t.Errorf("value = %q, want B", v)
	}
}

func TestMappingCache_Clear(t *testing.T) {
	mc := NewMappingCache[int, string](30 * time.Minute)
	mc.Get(context.Background(), 1, func(int) (string, error) { return "v", nil })
	mc.Clear()
	size, _ := mc.Stats()
	if size != 0 {
		t.Errorf("size after Clear = %d, want 0", size)
	}
}

func TestMappingCache_Stats(t *testing.T) {
	mc := NewMappingCache[int, string](30 * time.Minute)
	fetch := func(int) (string, error) { return "v", nil }
	mc.Get(context.Background(), 1, fetch) // miss
	mc.Get(context.Background(), 1, fetch) // hit
	mc.Get(context.Background(), 1, fetch) // hit
	size, hitRate := mc.Stats()
	if size != 1 {
		t.Errorf("size = %d, want 1", size)
	}
	// 2 hits out of 3 total = 66.6%
	if hitRate < 60 || hitRate > 70 {
		t.Errorf("hitRate = %f, want ~66.6", hitRate)
	}
}

func TestMappingCache_ConcurrentAccess(t *testing.T) {
	mc := NewMappingCache[int, string](30 * time.Minute)
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			mc.Get(context.Background(), id%5, func(k int) (string, error) {
				return "val", nil
			})
		}(i)
	}
	wg.Wait()
}

func TestMappingCache_SingleFlight(t *testing.T) {
	var calls atomic.Int32
	mc := NewMappingCache[int, string](30 * time.Minute)
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			mc.Get(context.Background(), 1, func(int) (string, error) {
				calls.Add(1)
				time.Sleep(20 * time.Millisecond)
				return "val", nil
			})
		}()
	}
	wg.Wait()
	if calls.Load() > 2 {
		t.Errorf("fetch called %d times, want <=2 (singleflight)", calls.Load())
	}
}
```

**Step 2: Run tests to verify they fail**

```bash
go test ./pkg/apihelper/... -v -race -count=1
```

Expected: FAIL.

**Step 3: Write implementation**

```go
// pkg/apihelper/mapping.go
package apihelper

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/singleflight"
)

type cacheEntry[V any] struct {
	value   V
	expires time.Time
}

// MappingCache is a generic TTL-based ID-to-name cache with singleflight deduplication.
type MappingCache[K comparable, V any] struct {
	mu      sync.RWMutex
	entries map[K]cacheEntry[V]
	ttl     time.Duration
	group   singleflight.Group
	hits    atomic.Int64
	total   atomic.Int64
}

// NewMappingCache creates a mapping cache with the given TTL.
func NewMappingCache[K comparable, V any](ttl time.Duration) *MappingCache[K, V] {
	return &MappingCache[K, V]{
		entries: make(map[K]cacheEntry[V]),
		ttl:     ttl,
	}
}

// Get returns the cached value for key, or fetches it using the provided function.
func (m *MappingCache[K, V]) Get(ctx context.Context, key K, fetch func(K) (V, error)) (V, error) {
	m.total.Add(1)

	m.mu.RLock()
	if e, ok := m.entries[key]; ok && time.Now().Before(e.expires) {
		m.mu.RUnlock()
		m.hits.Add(1)
		return e.value, nil
	}
	m.mu.RUnlock()

	cacheKey := fmt.Sprintf("%v", key)
	result, err, _ := m.group.Do(cacheKey, func() (any, error) {
		v, err := fetch(key)
		if err != nil {
			return nil, err
		}
		m.mu.Lock()
		m.entries[key] = cacheEntry[V]{value: v, expires: time.Now().Add(m.ttl)}
		m.mu.Unlock()
		return v, nil
	})
	if err != nil {
		var zero V
		return zero, err
	}
	return result.(V), nil
}

// Warm preloads the cache with all entries from fetchAll.
func (m *MappingCache[K, V]) Warm(ctx context.Context, fetchAll func() (map[K]V, error)) error {
	all, err := fetchAll()
	if err != nil {
		return fmt.Errorf("warming cache: %w", err)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	expires := time.Now().Add(m.ttl)
	for k, v := range all {
		m.entries[k] = cacheEntry[V]{value: v, expires: expires}
	}
	return nil
}

// Clear removes all entries from the cache.
func (m *MappingCache[K, V]) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries = make(map[K]cacheEntry[V])
	m.hits.Store(0)
	m.total.Store(0)
}

// Stats returns the current cache size and hit rate percentage.
func (m *MappingCache[K, V]) Stats() (size int, hitRate float64) {
	m.mu.RLock()
	size = len(m.entries)
	m.mu.RUnlock()
	total := m.total.Load()
	if total == 0 {
		return size, 0
	}
	return size, float64(m.hits.Load()) / float64(total) * 100
}
```

**Step 4: Run tests to verify they pass**

```bash
go test ./pkg/apihelper/... -v -race -count=1
```

Expected: PASS.

**Step 5: Commit**

```bash
git add pkg/apihelper/mapping.go pkg/apihelper/mapping_test.go
git commit -m "feat: add generic ID-to-name mapping cache in pkg/apihelper"
```

---

## Task 16: `pkg/apihelper/pagination` — Pagination Iterator

**Files:**
- Create: `pkg/apihelper/pagination.go`
- Create: `pkg/apihelper/pagination_test.go`
- Delete: `pkg/apihelper/.gitkeep`

**Step 1: Write the failing tests**

```go
// pkg/apihelper/pagination_test.go
package apihelper

import (
	"context"
	"errors"
	"testing"
)

func TestPaginate_SinglePage(t *testing.T) {
	fetch := func(ctx context.Context, page int) ([]string, bool, error) {
		return []string{"a", "b"}, false, nil
	}
	var items []string
	for item := range Paginate(context.Background(), fetch) {
		items = append(items, item)
	}
	if len(items) != 2 {
		t.Errorf("items = %v, want [a b]", items)
	}
}

func TestPaginate_MultiplePages(t *testing.T) {
	fetch := func(ctx context.Context, page int) ([]int, bool, error) {
		switch page {
		case 1:
			return []int{1, 2}, true, nil
		case 2:
			return []int{3, 4}, true, nil
		case 3:
			return []int{5}, false, nil
		default:
			return nil, false, nil
		}
	}
	var items []int
	for item := range Paginate(context.Background(), fetch) {
		items = append(items, item)
	}
	if len(items) != 5 {
		t.Errorf("len = %d, want 5; items = %v", len(items), items)
	}
}

func TestPaginate_EmptyResult(t *testing.T) {
	fetch := func(ctx context.Context, page int) ([]string, bool, error) {
		return nil, false, nil
	}
	var count int
	for range Paginate(context.Background(), fetch) {
		count++
	}
	if count != 0 {
		t.Errorf("count = %d, want 0", count)
	}
}

func TestPaginate_ErrorStops(t *testing.T) {
	fetch := func(ctx context.Context, page int) ([]string, bool, error) {
		if page == 2 {
			return nil, false, errors.New("api error")
		}
		return []string{"item"}, true, nil
	}
	var count int
	for range Paginate(context.Background(), fetch) {
		count++
	}
	// Should get items from page 1 only
	if count != 1 {
		t.Errorf("count = %d, want 1 (stop on error)", count)
	}
}

func TestPaginate_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	fetch := func(ctx context.Context, page int) ([]string, bool, error) {
		if page == 2 {
			cancel()
		}
		return []string{"item"}, true, nil
	}
	var count int
	for range Paginate(ctx, fetch) {
		count++
	}
	// Should stop after context cancellation
	if count > 2 {
		t.Errorf("count = %d, should stop shortly after cancellation", count)
	}
}
```

**Step 2: Run tests to verify they fail**

```bash
go test ./pkg/apihelper/... -v -race -count=1
```

Expected: FAIL.

**Step 3: Write implementation**

```go
// pkg/apihelper/pagination.go
package apihelper

import (
	"context"
	"iter"
)

// PageFetcher fetches a page of items. Returns items, hasMore, and error.
type PageFetcher[T any] func(ctx context.Context, page int) (items []T, hasMore bool, err error)

// Paginate returns a Go 1.23 iterator that lazily fetches pages.
// Stops on error, empty page, hasMore=false, or context cancellation.
func Paginate[T any](ctx context.Context, fetch PageFetcher[T]) iter.Seq[T] {
	return func(yield func(T) bool) {
		for page := 1; ; page++ {
			if ctx.Err() != nil {
				return
			}
			items, hasMore, err := fetch(ctx, page)
			if err != nil {
				return
			}
			for _, item := range items {
				if !yield(item) {
					return
				}
			}
			if !hasMore || len(items) == 0 {
				return
			}
		}
	}
}
```

**Step 4: Run tests to verify they pass**

```bash
go test ./pkg/apihelper/... -v -race -count=1
```

Expected: PASS.

**Step 5: Commit**

```bash
rm pkg/apihelper/.gitkeep
git add pkg/apihelper/
git commit -m "feat: add pagination iterator with Go 1.23 iter.Seq in pkg/apihelper"
```

---

## Task 17: Full Test Suite & Lint

**Step 1: Run all tests with race detection**

```bash
cd /Users/zeisler/lgp-mcp
make test
```

Expected: ALL PASS with -race flag.

**Step 2: Run linter (if golangci-lint installed)**

```bash
make lint 2>/dev/null || echo "golangci-lint not installed, skip"
```

**Step 3: Run go vet**

```bash
go vet ./...
```

Expected: No issues.

**Step 4: Verify build**

```bash
go build ./...
```

Expected: Compiles cleanly.

**Step 5: Commit cleanup (remove remaining .gitkeep files)**

```bash
rm -f pkg/resilience/.gitkeep pkg/mcputil/.gitkeep pkg/apihelper/.gitkeep pkg/config/.gitkeep
git add -u
# Only commit if there are changes
git diff --cached --quiet || git commit -m "chore: remove .gitkeep files from populated pkg directories"
```

---

## Summary

| Task | Package | What | Tests |
|------|---------|------|-------|
| 1 | root | go.mod init, dependencies | — |
| 2 | pkg/config | MustEnv, OptEnv, LogLevel | 7 |
| 3 | pkg/resilience | RateLimiter (token bucket) | 9 |
| 4 | pkg/resilience | CircuitBreaker (state machine) | 9 |
| 5 | pkg/resilience | Compactor (null/empty removal) | 10 |
| 6 | pkg/resilience | Middleware (combines all three) | 8 |
| 7 | pkg/mcputil | Standard error messages | — |
| 8 | pkg/mcputil | TextResult, ErrorResult, JSONResult | 4 |
| 9 | pkg/mcputil | Tool annotation helpers | 4 |
| 10 | pkg/mcputil | HTML-to-plaintext converter | 4 |
| 11 | pkg/mcputil | Entity-aware formatter | 5 |
| 12 | pkg/apihelper | HTTP client with retry | 9 |
| 13 | pkg/apihelper | OAuth2 token manager | 4 |
| 14 | pkg/apihelper | JSON:API parser | 7 |
| 15 | pkg/apihelper | ID-to-name mapping cache | 6 |
| 16 | pkg/apihelper | Pagination iterator (iter.Seq) | 5 |
| 17 | — | Full test suite, lint, vet | — |

**Total: 17 tasks, ~85 tests, 16 files of implementation code.**
