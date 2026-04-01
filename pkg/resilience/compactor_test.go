package resilience

import (
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
	input := map[string]any{"name": "test", "age": nil, "city": "Berlin"}
	got := Compact(input).(map[string]any)
	if _, ok := got["age"]; ok {
		t.Error("should remove nil values")
	}
	if got["name"] != "test" || got["city"] != "Berlin" {
		t.Error("should preserve non-nil values")
	}
}

func TestCompact_RemovesEmptySlices(t *testing.T) {
	input := map[string]any{"name": "test", "items": []any{}}
	got := Compact(input).(map[string]any)
	if _, ok := got["items"]; ok {
		t.Error("should remove empty slices")
	}
}

func TestCompact_RemovesEmptyMaps(t *testing.T) {
	input := map[string]any{"name": "test", "nested": map[string]any{}}
	got := Compact(input).(map[string]any)
	if _, ok := got["nested"]; ok {
		t.Error("should remove empty maps")
	}
}

func TestCompact_PreservesMeaningfulValues(t *testing.T) {
	input := map[string]any{"zero": float64(0), "false": false, "empty": ""}
	got := Compact(input).(map[string]any)
	if len(got) != 3 {
		t.Errorf("should preserve 0, false, empty string; got %d keys", len(got))
	}
}

func TestCompact_Recursive(t *testing.T) {
	input := map[string]any{
		"outer": map[string]any{
			"keep": "yes", "remove": nil,
			"deep": map[string]any{"only_null": nil},
		},
	}
	got := Compact(input).(map[string]any)
	outer := got["outer"].(map[string]any)
	if _, ok := outer["remove"]; ok {
		t.Error("should remove nested nil")
	}
	if _, ok := outer["deep"]; ok {
		t.Error("should remove map that becomes empty")
	}
	if outer["keep"] != "yes" {
		t.Error("should keep non-nil nested values")
	}
}

func TestCompact_Array(t *testing.T) {
	input := []any{"hello", nil, map[string]any{"a": nil}, map[string]any{"b": "keep"}}
	got := Compact(input).([]any)
	if len(got) != 2 {
		t.Errorf("compacted array len = %d, want 2", len(got))
	}
}

func TestCompact_ReturnsNilForAllNullMap(t *testing.T) {
	input := map[string]any{"a": nil, "b": nil}
	got := Compact(input)
	if got != nil {
		t.Errorf("Compact of all-nil map = %v, want nil", got)
	}
}

func TestEstimateSavings(t *testing.T) {
	original := map[string]any{"a": "hello", "b": nil, "c": []any{}, "d": map[string]any{}}
	compacted := Compact(original)
	pct := EstimateSavings(original, compacted)
	if pct <= 0 || pct >= 100 {
		t.Errorf("EstimateSavings = %f, want between 0 and 100", pct)
	}
}
