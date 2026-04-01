package mcputil

import (
	"encoding/json"
	"fmt"
	"strings"
)

type FieldSet map[string][]string

var DefaultFields = FieldSet{
	"Ticket":      {"id", "ticketNumber", "title", "status", "priority", "companyID", "assignedResourceID", "createDate", "dueDateTime"},
	"Company":     {"id", "companyName", "isActive", "phone", "city", "state"},
	"Contact":     {"id", "firstName", "lastName", "emailAddress", "companyID"},
	"Project":     {"id", "projectName", "status", "companyID", "projectLeadResourceID", "startDate", "endDate"},
	"Task":        {"id", "title", "status", "projectID", "assignedResourceID", "percentComplete"},
	"Resource":    {"id", "firstName", "lastName", "email", "isActive"},
	"TimeEntry":   {"id", "resourceID", "ticketID", "dateWorked", "hoursWorked", "summaryNotes"},
	"BillingItem": {"id", "itemName", "companyID", "ticketID", "postedDate", "totalAmount", "invoiceID"},
}

var nameMapping = map[string]string{
	"companyID":             "companyName",
	"assignedResourceID":    "assignedResourceName",
	"resourceID":            "resourceName",
	"projectLeadResourceID": "projectLeadName",
}

func FormatCompact(entityType string, data []map[string]any, fields FieldSet) string {
	essentialFields, ok := fields[entityType]
	if !ok {
		essentialFields, ok = DefaultFields[entityType]
	}
	var sb strings.Builder
	for i, item := range data {
		if i > 0 {
			sb.WriteString("\n---\n")
		}
		if ok {
			for _, f := range essentialFields {
				if v, exists := item[f]; exists && v != nil {
					fmt.Fprintf(&sb, "%s: %v\n", f, v)
				}
			}
			for _, nameField := range nameMapping {
				if v, exists := item[nameField]; exists && v != nil {
					fmt.Fprintf(&sb, "%s: %v\n", nameField, v)
				}
			}
		} else {
			b, _ := json.MarshalIndent(item, "", "  ")
			sb.Write(b)
			sb.WriteByte('\n')
		}
	}
	return strings.TrimSpace(sb.String())
}

func FormatFull(data map[string]any) string {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Sprintf("error formatting response: %v", err)
	}
	return string(b)
}

func WithPagination(text string, current, total, count int) string {
	return fmt.Sprintf("%s\n\n--- Page %d/%d | %d total results ---", text, current, total, count)
}

func WithNames(data map[string]any, names map[string]string) map[string]any {
	result := make(map[string]any, len(data)+len(names))
	for k, v := range data {
		result[k] = v
	}
	for idField, resolvedName := range names {
		if nameField, ok := nameMapping[idField]; ok {
			result[nameField] = resolvedName
		}
	}
	return result
}
