package mcputil

import (
	"strings"
	"testing"
)

func TestFormatCompact(t *testing.T) {
	data := []map[string]any{
		{"id": 1, "ticketNumber": "T001", "title": "Bug", "extra": "hidden"},
		{"id": 2, "ticketNumber": "T002", "title": "Feature", "extra": "hidden"},
	}
	fields := FieldSet{"Ticket": {"id", "ticketNumber", "title"}}
	result := FormatCompact("Ticket", data, fields)
	if !strings.Contains(result, "T001") {
		t.Error("should contain T001")
	}
	if !strings.Contains(result, "T002") {
		t.Error("should contain T002")
	}
	if strings.Contains(result, "hidden") {
		t.Error("should not contain non-essential field")
	}
}

func TestFormatCompact_UnknownEntity(t *testing.T) {
	data := []map[string]any{{"id": 1, "name": "test"}}
	result := FormatCompact("Unknown", data, FieldSet{})
	if !strings.Contains(result, "test") {
		t.Error("unknown entity should include all fields")
	}
}

func TestFormatFull(t *testing.T) {
	data := map[string]any{"id": 1, "name": "test", "active": true}
	result := FormatFull(data)
	if !strings.Contains(result, "name") || !strings.Contains(result, "test") {
		t.Error("FormatFull should include all fields")
	}
}

func TestWithPagination(t *testing.T) {
	result := WithPagination("data here", 1, 5, 25)
	if !strings.Contains(result, "Page 1/5") {
		t.Error("should contain page info")
	}
	if !strings.Contains(result, "25") {
		t.Error("should contain total count")
	}
}

func TestWithNames(t *testing.T) {
	data := map[string]any{"companyID": 42, "assignedResourceID": 7}
	names := map[string]string{"companyID": "Acme Corp", "assignedResourceID": "John Doe"}
	result := WithNames(data, names)
	if result["companyName"] != "Acme Corp" {
		t.Errorf("companyName = %v", result["companyName"])
	}
	if result["assignedResourceName"] != "John Doe" {
		t.Errorf("assignedResourceName = %v", result["assignedResourceName"])
	}
}
