package resilience

import "encoding/json"

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
