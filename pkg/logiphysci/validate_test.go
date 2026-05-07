package logiphysci

import (
	"strings"
	"testing"
)

func TestRequireString(t *testing.T) {
	tests := []struct {
		name    string
		payload map[string]any
		key     string
		wantErr string
	}{
		{"present non-empty", map[string]any{"x": "ok"}, "x", ""},
		{"missing", map[string]any{}, "x", `missing required field "x"`},
		{"wrong type", map[string]any{"x": 42}, "x", `must be a string`},
		{"empty string", map[string]any{"x": ""}, "x", `must not be empty`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := requireString(tt.payload, tt.key)
			checkErr(t, err, tt.wantErr)
		})
	}
}

func TestRequireStringArray(t *testing.T) {
	tests := []struct {
		name    string
		payload map[string]any
		key     string
		wantErr string
	}{
		{"valid", map[string]any{"x": []any{"a", "b"}}, "x", ""},
		{"missing", map[string]any{}, "x", `missing required field "x"`},
		{"wrong type", map[string]any{"x": "not array"}, "x", `must be an array`},
		{"empty", map[string]any{"x": []any{}}, "x", `must not be empty`},
		{"non-string element", map[string]any{"x": []any{"a", 5}}, "x", `must be a string`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := requireStringArray(tt.payload, tt.key)
			checkErr(t, err, tt.wantErr)
		})
	}
}

func TestRequireNonEmptyArray(t *testing.T) {
	tests := []struct {
		name    string
		payload map[string]any
		key     string
		wantErr string
	}{
		{"valid mixed", map[string]any{"x": []any{map[string]any{"k": 1}, "string-ok"}}, "x", ""},
		{"missing", map[string]any{}, "x", `missing required field "x"`},
		{"wrong type", map[string]any{"x": 99}, "x", `must be an array`},
		{"empty", map[string]any{"x": []any{}}, "x", `must not be empty`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := requireNonEmptyArray(tt.payload, tt.key)
			checkErr(t, err, tt.wantErr)
		})
	}
}

func TestRequireInt(t *testing.T) {
	tests := []struct {
		name    string
		payload map[string]any
		key     string
		wantErr string
	}{
		{"int", map[string]any{"x": 1}, "x", ""},
		{"float64 (json default)", map[string]any{"x": float64(2)}, "x", ""},
		{"int64", map[string]any{"x": int64(3)}, "x", ""},
		{"missing", map[string]any{}, "x", `missing required field "x"`},
		{"string", map[string]any{"x": "5"}, "x", `must be a number`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := requireInt(tt.payload, tt.key)
			checkErr(t, err, tt.wantErr)
		})
	}
}

func TestToInt(t *testing.T) {
	cases := []struct {
		in      any
		wantOK  bool
		wantInt int
	}{
		{1, true, 1},
		{int32(2), true, 2},
		{int64(3), true, 3},
		{float32(4), true, 4},
		{float64(5), true, 5},
		{"6", false, 0},
		{nil, false, 0},
	}
	for _, c := range cases {
		got, ok := toInt(c.in)
		if ok != c.wantOK || got != c.wantInt {
			t.Errorf("toInt(%v) = (%d, %t), want (%d, %t)", c.in, got, ok, c.wantInt, c.wantOK)
		}
	}
}

func TestMimeForExt(t *testing.T) {
	cases := map[string]string{
		"docx":    "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		"pdf":     "application/pdf",
		"unknown": "application/octet-stream",
	}
	for ext, want := range cases {
		if got := MimeForExt(ext); got != want {
			t.Errorf("MimeForExt(%q) = %q, want %q", ext, got, want)
		}
	}
}

// checkErr asserts that err either is nil (when wantSubstr == "") or contains the substring.
func checkErr(t *testing.T, err error, wantSubstr string) {
	t.Helper()
	if wantSubstr == "" {
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		return
	}
	if err == nil {
		t.Fatalf("expected error containing %q, got nil", wantSubstr)
	}
	if !strings.Contains(err.Error(), wantSubstr) {
		t.Errorf("error %q does not contain %q", err.Error(), wantSubstr)
	}
}
