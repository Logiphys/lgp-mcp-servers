package autotask

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestFormatSearchResult_Compact(t *testing.T) {
	items := []map[string]any{
		{"id": 1, "ticketNumber": "T001", "title": "Bug", "extra": "hidden", "status": 1},
		{"id": 2, "ticketNumber": "T002", "title": "Feature", "extra": "hidden", "status": 2},
	}
	result := FormatSearchResult("autotask_search_tickets", items, 1, 25)
	if strings.Contains(result, "hidden") {
		t.Error("compact format should not include non-essential fields")
	}
	if !strings.Contains(result, "T001") {
		t.Error("should contain ticketNumber")
	}
	if !strings.Contains(result, `"returned": 2`) {
		t.Error("should contain returned count")
	}
}

func TestFormatSearchResult_NonCompact(t *testing.T) {
	items := []map[string]any{
		{"id": 1, "name": "test", "extra": "included"},
	}
	result := FormatSearchResult("autotask_search_unknown", items, 1, 25)
	if !strings.Contains(result, "included") {
		t.Error("non-compact format should include all fields")
	}
}

func TestFormatSearchResult_HasMore(t *testing.T) {
	items := make([]map[string]any, 25)
	for i := range items {
		items[i] = map[string]any{"id": i}
	}
	result := FormatSearchResult("autotask_search_tickets", items, 1, 25)
	if !strings.Contains(result, "hint") {
		t.Error("should include pagination hint when hasMore is true")
	}
	if !strings.Contains(result, "page:2") {
		t.Error("hint should suggest next page")
	}
}

func TestFormatGetResult(t *testing.T) {
	item := map[string]any{"id": 42, "name": "test"}
	result := FormatGetResult(item)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}
	if parsed["id"] != float64(42) {
		t.Error("should contain id")
	}
}

func TestFormatCreateResult(t *testing.T) {
	result := FormatCreateResult("Ticket", 123)
	if !strings.Contains(result, "123") || !strings.Contains(result, "Ticket") {
		t.Error("should contain entity type and ID")
	}
}

func TestFormatNotFound_WithCriteria(t *testing.T) {
	result := FormatNotFound("Tickets", map[string]any{"companyID": 42, "status": 1})
	if !strings.Contains(result, "companyID=42") {
		t.Error("should include search criteria")
	}
}

func TestFormatNotFound_NoCriteria(t *testing.T) {
	result := FormatNotFound("Tickets", map[string]any{})
	if !strings.Contains(result, "No Tickets found") {
		t.Error("should show generic not found message")
	}
}

func TestFormatPicklistValues(t *testing.T) {
	values := []PicklistValue{
		{Value: "1", Label: "New", IsActive: true},
		{Value: "5", Label: "Complete", IsActive: true},
	}
	result := FormatPicklistValues("Ticket Statuses", values)
	if !strings.Contains(result, "New") || !strings.Contains(result, "Complete") {
		t.Error("should contain picklist labels")
	}
	if !strings.Contains(result, `"count": 2`) {
		t.Error("should contain count")
	}
}
