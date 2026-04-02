package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
)

func MustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic(fmt.Sprintf("required environment variable %s is not set", key))
	}
	return v
}

func OptEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

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

// AccessTier reads an access-tier env var, returning 1–3 (default 1).
func AccessTier(envKey string) int {
	v := os.Getenv(envKey)
	if v == "" {
		return 1
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 1 {
		return 1
	}
	if n > 3 {
		return 3
	}
	return n
}
