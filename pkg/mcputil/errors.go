package mcputil

import "errors"

var (
	ErrNotFound     = errors.New("resource not found")
	ErrRateLimited  = errors.New("rate limit exceeded, retry after wait period")
	ErrCircuitOpen  = errors.New("service temporarily unavailable (circuit breaker open)")
	ErrValidation   = errors.New("input validation failed")
	ErrUnauthorized = errors.New("authentication failed — check credentials")
)
