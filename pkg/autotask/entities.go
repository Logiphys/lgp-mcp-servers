package autotask

import "strings"

// CompactFields maps entity types to their essential field names for compact responses.
var CompactFields = map[string][]string{
	"Tickets":                   {"id", "ticketNumber", "title", "status", "priority", "companyID", "assignedResourceID", "createDate", "dueDateTime"},
	"Companies":                 {"id", "companyName", "isActive", "phone", "city", "state"},
	"Contacts":                  {"id", "firstName", "lastName", "emailAddress", "companyID"},
	"Projects":                  {"id", "projectName", "status", "companyID", "projectLeadResourceID", "startDate", "endDate"},
	"Tasks":                     {"id", "title", "status", "projectID", "assignedResourceID", "percentComplete"},
	"Resources":                 {"id", "firstName", "lastName", "email", "isActive"},
	"BillingItems":              {"id", "itemName", "companyID", "ticketID", "projectID", "postedDate", "totalAmount", "invoiceID", "billingItemType"},
	"BillingItemApprovalLevels": {"id", "timeEntryID", "approvalLevel", "approvalResourceID", "approvalDateTime"},
	"TimeEntries":               {"id", "resourceID", "ticketID", "projectID", "taskID", "dateWorked", "hoursWorked", "summaryNotes"},
	"TicketCharges":             {"id", "ticketID", "name", "chargeType", "unitQuantity", "unitPrice", "datePurchased"},
}

// CompactSearchTools maps tool names to their entity type for compact formatting.
var CompactSearchTools = map[string]string{
	"search_tickets":                      "Tickets",
	"search_companies":                    "Companies",
	"search_contacts":                     "Contacts",
	"search_projects":                     "Projects",
	"search_tasks":                        "Tasks",
	"search_resources":                    "Resources",
	"search_billing_items":                "BillingItems",
	"search_billing_item_approval_levels": "BillingItemApprovalLevels",
	"search_time_entries":                 "TimeEntries",
	"search_ticket_charges":               "TicketCharges",
}

// EntityAliases normalizes entity type names for get_field_info.
var EntityAliases = map[string]string{
	"tasks":              "Tasks",
	"projecttasks":       "Tasks",
	"ProjectTasks":       "Tasks",
	"tickets":            "Tickets",
	"companies":          "Companies",
	"accounts":           "Companies",
	"contacts":           "Contacts",
	"resources":          "Resources",
	"projects":           "Projects",
	"timeentries":        "TimeEntries",
	"billingitems":       "BillingItems",
	"configurationitems": "ConfigurationItems",
	"contracts":          "Contracts",
	"invoices":           "Invoices",
	"quotes":             "Quotes",
	"quoteitems":         "QuoteItems",
	"opportunities":      "Opportunities",
	"products":           "Products",
	"services":           "Services",
	"servicebundles":     "ServiceBundles",
	"expensereports":     "ExpenseReports",
	"expenseitems":       "ExpenseItems",
	"servicecalls":       "ServiceCalls",
	"ticketcharges":      "TicketCharges",
	"phases":             "Phases",
}

// NormalizeEntityType converts user input to canonical Autotask entity type name.
func NormalizeEntityType(input string) string {
	if alias, ok := EntityAliases[input]; ok {
		return alias
	}
	// Try lowercase lookup
	lower := strings.ToLower(input)
	for k, v := range EntityAliases {
		if strings.ToLower(k) == lower {
			return v
		}
	}
	// Return as-is if no alias found (might be a valid entity type)
	return input
}
