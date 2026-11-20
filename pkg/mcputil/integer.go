package mcputil

import (
	"fmt"
	"math"
)

// IntArgument accepts integral JSON numbers without truncating invalid input.
func IntArgument(args map[string]any, name string, fallback, minimum, maximum int) (int, error) {
	raw, exists := args[name]
	if !exists {
		return fallback, nil
	}
	var value float64
	switch v := raw.(type) {
	case float64:
		value = v
	case int:
		value = float64(v)
	default:
		if minimum == 1 {
			return 0, fmt.Errorf("%s must be a positive integer", name)
		}
		return 0, fmt.Errorf("%s must be an integer", name)
	}
	if math.Trunc(value) != value || !(value >= float64(minimum) && value <= float64(maximum)) {
		return 0, fmt.Errorf("%s must be an integer between %d and %d", name, minimum, maximum)
	}
	return int(value), nil
}
