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
