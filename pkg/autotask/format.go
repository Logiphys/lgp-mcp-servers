package autotask

import (
	"encoding/json"
	"fmt"
	"strings"
)

// FormatSearchResult formats search results with compact fields and pagination metadata.
func FormatSearchResult(toolName string, items []map[string]any, page, pageSize int) string {
	entityType, isCompact := CompactSearchTools[toolName]

	var formatted []map[string]any
	if isCompact {
		fields := CompactFields[entityType]
		for _, item := range items {
			compact := make(map[string]any)
			for _, f := range fields {
				if v, ok := item[f]; ok && v != nil {
					compact[f] = v
				}
			}
			// Include enhanced names if present
			for _, nameField := range []string{"_companyName", "_assignedResourceName", "_resourceName", "_projectLeadName"} {
				if v, ok := item[nameField]; ok && v != nil {
					compact[nameField] = v
				}
			}
			formatted = append(formatted, compact)
		}
	} else {
		formatted = items
	}

	hasMore := len(items) >= pageSize
	result := map[string]any{
		"summary": map[string]any{
			"returned": len(formatted),
			"hasMore":  hasMore,
			"page":     page,
			"pageSize": pageSize,
		},
		"items": formatted,
	}
	if hasMore {
		result["summary"].(map[string]any)["hint"] = fmt.Sprintf(
			"Use page:%d for more results, or use get_*_details for full data on specific items", page+1)
	}

	b, _ := json.MarshalIndent(result, "", "  ")
	return string(b)
}

// FormatGetResult formats a single entity as indented JSON.
func FormatGetResult(item map[string]any) string {
	b, err := json.MarshalIndent(item, "", "  ")
	if err != nil {
		return fmt.Sprintf(`{"error": "failed to format response: %v"}`, err)
	}
	return string(b)
}

// FormatCreateResult formats a create success message.
func FormatCreateResult(entityType string, id int) string {
	return fmt.Sprintf(`{"message": "%s created successfully", "id": %d}`, entityType, id)
}

// FormatUpdateResult formats an update success message.
func FormatUpdateResult(entityType string, id int) string {
	return fmt.Sprintf(`{"message": "%s %d updated successfully"}`, entityType, id)
}

// FormatDeleteResult formats a delete success message.
func FormatDeleteResult(entityType string, id int) string {
	return fmt.Sprintf(`{"message": "%s %d deleted successfully"}`, entityType, id)
}

// FormatNotFound formats a not-found error with search criteria for debugging.
func FormatNotFound(entityType string, criteria map[string]any) string {
	parts := make([]string, 0, len(criteria))
	for k, v := range criteria {
		if v != nil && v != "" && v != 0 && v != false {
			parts = append(parts, fmt.Sprintf("%s=%v", k, v))
		}
	}
	if len(parts) == 0 {
		return fmt.Sprintf(`{"error": "No %s found"}`, entityType)
	}
	return fmt.Sprintf(`{"error": "No %s found matching: %s"}`, entityType, strings.Join(parts, ", "))
}

// FormatPicklistValues formats picklist values as a readable list.
func FormatPicklistValues(label string, values []PicklistValue) string {
	result := map[string]any{
		"label": label,
		"count": len(values),
	}
	items := make([]map[string]any, 0, len(values))
	for _, v := range values {
		items = append(items, map[string]any{
			"value":    v.Value,
			"label":    v.Label,
			"isActive": v.IsActive,
		})
	}
	result["items"] = items
	b, _ := json.MarshalIndent(result, "", "  ")
	return string(b)
}
