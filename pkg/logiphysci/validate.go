package logiphysci

import "fmt"

// requireString returns an error if key is missing or not a non-empty string.
func requireString(payload map[string]any, key string) error {
	v, ok := payload[key]
	if !ok {
		return fmt.Errorf("missing required field %q", key)
	}
	s, ok := v.(string)
	if !ok {
		return fmt.Errorf("field %q must be a string, got %T", key, v)
	}
	if s == "" {
		return fmt.Errorf("field %q must not be empty", key)
	}
	return nil
}

// requireStringArray returns an error if key is missing, not an array, empty,
// or contains non-string entries.
func requireStringArray(payload map[string]any, key string) error {
	v, ok := payload[key]
	if !ok {
		return fmt.Errorf("missing required field %q", key)
	}
	arr, ok := v.([]any)
	if !ok {
		return fmt.Errorf("field %q must be an array, got %T", key, v)
	}
	if len(arr) == 0 {
		return fmt.Errorf("field %q must not be empty", key)
	}
	for i, e := range arr {
		if _, ok := e.(string); !ok {
			return fmt.Errorf("field %q[%d] must be a string, got %T", key, i, e)
		}
	}
	return nil
}

// requireNonEmptyArray returns an error if key is missing, not an array, or empty.
// Element types are not checked here — the Python helper validates the inner shape.
func requireNonEmptyArray(payload map[string]any, key string) error {
	v, ok := payload[key]
	if !ok {
		return fmt.Errorf("missing required field %q", key)
	}
	arr, ok := v.([]any)
	if !ok {
		return fmt.Errorf("field %q must be an array, got %T", key, v)
	}
	if len(arr) == 0 {
		return fmt.Errorf("field %q must not be empty", key)
	}
	return nil
}

// requireInt returns an error if key is missing or not a JSON number.
// JSON numbers decode to float64 in encoding/json; the Python helper casts to int.
func requireInt(payload map[string]any, key string) error {
	v, ok := payload[key]
	if !ok {
		return fmt.Errorf("missing required field %q", key)
	}
	switch v.(type) {
	case int, int32, int64, float32, float64:
		return nil
	default:
		return fmt.Errorf("field %q must be a number, got %T", key, v)
	}
}
