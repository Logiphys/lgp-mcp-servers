package autotask

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Logiphys/lgp-mcp-servers/pkg/mcputil"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerTimeBillingTools(srv *server.MCPServer, client *Client, _ *slog.Logger) {
	// === TIME ENTRIES ===

	// autotask_create_time_entry
	addTool(srv,
		mcp.NewTool("create_time_entry",
			mcp.WithDescription("Create a new time entry in Autotask"),
			mcp.WithNumber("ticketID", mcp.Description("Ticket ID to log time against"), mcp.Required()),
			mcp.WithNumber("resourceID", mcp.Description("Resource (technician) ID"), mcp.Required()),
			mcp.WithString("dateWorked", mcp.Description("Date worked (YYYY-MM-DD)"), mcp.Required()),
			mcp.WithNumber("hoursWorked", mcp.Description("Number of hours worked"), mcp.Required()),
			mcp.WithString("summaryNotes", mcp.Description("Summary notes visible to customer")),
			mcp.WithString("internalNotes", mcp.Description("Internal notes not visible to customer")),
			mcp.WithNumber("roleID", mcp.Description("Role ID for this time entry")),
			mcp.WithNumber("billingCodeID", mcp.Description("Billing code ID")),
			mcp.WithString("startDateTime", mcp.Description("Start date/time (ISO 8601)")),
			mcp.WithString("endDateTime", mcp.Description("End date/time (ISO 8601)")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			data := map[string]any{
				"ticketID":    req.GetInt("ticketID", 0),
				"resourceID":  req.GetInt("resourceID", 0),
				"dateWorked":  req.GetString("dateWorked", ""),
				"hoursWorked": req.GetFloat("hoursWorked", 0),
			}
			args := req.GetArguments()
			if v, ok := args["summaryNotes"]; ok {
				data["summaryNotes"] = v
			}
			if v, ok := args["internalNotes"]; ok {
				data["internalNotes"] = v
			}
			if v, ok := args["roleID"]; ok {
				data["roleID"] = v
			}
			if v, ok := args["billingCodeID"]; ok {
				data["billingCodeID"] = v
			}
			if v, ok := args["startDateTime"]; ok {
				data["startDateTime"] = v
			}
			if v, ok := args["endDateTime"]; ok {
				data["endDateTime"] = v
			}
			id, err := client.Create(ctx, "TimeEntries", data)
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.TextResult(FormatCreateResult("TimeEntry", id)), nil
		},
	)

	// autotask_search_time_entries
	addTool(srv,
		mcp.NewTool("search_time_entries",
			mcp.WithDescription("Search for time entries in Autotask"),
			mcp.WithNumber("ticketID", mcp.Description("Filter by ticket ID")),
			mcp.WithNumber("resourceID", mcp.Description("Filter by resource ID")),
			mcp.WithString("dateWorkedStart", mcp.Description("Filter entries on or after this date (YYYY-MM-DD)")),
			mcp.WithString("dateWorkedEnd", mcp.Description("Filter entries on or before this date (YYYY-MM-DD)")),
			mcp.WithNumber("page", mcp.Description("Page number"), mcp.Min(1)),
			mcp.WithNumber("pageSize", mcp.Description("Results per page"), mcp.Min(1), mcp.Max(500)),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(true),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.GetArguments()
			var filters []Filter
			if v, ok := args["ticketID"]; ok {
				filters = append(filters, Filter{Op: "eq", Field: "ticketID", Value: v})
			}
			if v, ok := args["resourceID"]; ok {
				filters = append(filters, Filter{Op: "eq", Field: "resourceID", Value: v})
			}
			if v, ok := args["dateWorkedStart"]; ok {
				filters = append(filters, Filter{Op: "gte", Field: "dateWorked", Value: v})
			}
			if v, ok := args["dateWorkedEnd"]; ok {
				filters = append(filters, Filter{Op: "lte", Field: "dateWorked", Value: v})
			}
			items, err := client.Query(ctx, "TimeEntries", filters, QueryOpts{
				Page:     req.GetInt("page", 1),
				PageSize: req.GetInt("pageSize", 25),
				MaxSize:  500,
			})
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			if len(items) == 0 {
				return mcputil.TextResult(FormatNotFound("time entries", map[string]any{})), nil
			}
			items = client.EnhanceWithNames(ctx, items)
			return mcputil.TextResult(FormatSearchResult("search_time_entries", items, req.GetInt("page", 1), req.GetInt("pageSize", 25))), nil
		},
	)

	// === BILLING ITEMS ===

	// autotask_search_billing_items
	addTool(srv,
		mcp.NewTool("search_billing_items",
			mcp.WithDescription("Search for billing items in Autotask"),
			mcp.WithNumber("ticketID", mcp.Description("Filter by ticket ID")),
			mcp.WithNumber("invoiceID", mcp.Description("Filter by invoice ID")),
			mcp.WithNumber("companyID", mcp.Description("Filter by company ID")),
			mcp.WithString("itemDateStart", mcp.Description("Filter items on or after this date (YYYY-MM-DD)")),
			mcp.WithString("itemDateEnd", mcp.Description("Filter items on or before this date (YYYY-MM-DD)")),
			mcp.WithNumber("page", mcp.Description("Page number"), mcp.Min(1)),
			mcp.WithNumber("pageSize", mcp.Description("Results per page"), mcp.Min(1), mcp.Max(500)),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(true),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.GetArguments()
			var filters []Filter
			if v, ok := args["ticketID"]; ok {
				filters = append(filters, Filter{Op: "eq", Field: "ticketID", Value: v})
			}
			if v, ok := args["invoiceID"]; ok {
				filters = append(filters, Filter{Op: "eq", Field: "invoiceID", Value: v})
			}
			if v, ok := args["companyID"]; ok {
				filters = append(filters, Filter{Op: "eq", Field: "companyID", Value: v})
			}
			if v, ok := args["itemDateStart"]; ok {
				filters = append(filters, Filter{Op: "gte", Field: "itemDate", Value: v})
			}
			if v, ok := args["itemDateEnd"]; ok {
				filters = append(filters, Filter{Op: "lte", Field: "itemDate", Value: v})
			}
			items, err := client.Query(ctx, "BillingItems", filters, QueryOpts{
				Page:     req.GetInt("page", 1),
				PageSize: req.GetInt("pageSize", 25),
				MaxSize:  500,
			})
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			if len(items) == 0 {
				return mcputil.TextResult(FormatNotFound("billing items", map[string]any{})), nil
			}
			items = client.EnhanceWithNames(ctx, items)
			return mcputil.TextResult(FormatSearchResult("search_billing_items", items, req.GetInt("page", 1), req.GetInt("pageSize", 25))), nil
		},
	)

	// autotask_get_billing_item
	addTool(srv,
		mcp.NewTool("get_billing_item",
			mcp.WithDescription("Get a specific billing item by ID"),
			mcp.WithNumber("id", mcp.Description("Billing item ID"), mcp.Required()),
			mcp.WithReadOnlyHintAnnotation(true),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			id := req.GetInt("id", 0)
			if id == 0 {
				return mcputil.ErrorResult(fmt.Errorf("id is required")), nil
			}
			item, err := client.Get(ctx, "BillingItems", id)
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			if item == nil {
				return mcputil.TextResult(FormatNotFound("billing item", map[string]any{"id": id})), nil
			}
			enhanced := client.EnhanceWithNames(ctx, []map[string]any{item})
			return mcputil.TextResult(FormatGetResult(enhanced[0])), nil
		},
	)

	// autotask_search_billing_item_approval_levels
	addTool(srv,
		mcp.NewTool("search_billing_item_approval_levels",
			mcp.WithDescription("Search for billing item approval levels in Autotask"),
			mcp.WithNumber("billingItemID", mcp.Description("Filter by billing item ID")),
			mcp.WithNumber("timeEntryID", mcp.Description("Filter by time entry ID")),
			mcp.WithNumber("approvalLevel", mcp.Description("Filter by approval level")),
			mcp.WithString("approvalDateTimeStart", mcp.Description("Filter approvals on or after this date/time (ISO 8601)")),
			mcp.WithString("approvalDateTimeEnd", mcp.Description("Filter approvals on or before this date/time (ISO 8601)")),
			mcp.WithNumber("page", mcp.Description("Page number"), mcp.Min(1)),
			mcp.WithNumber("pageSize", mcp.Description("Results per page"), mcp.Min(1), mcp.Max(500)),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(true),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.GetArguments()
			var filters []Filter
			if v, ok := args["billingItemID"]; ok {
				filters = append(filters, Filter{Op: "eq", Field: "billingItemID", Value: v})
			}
			if v, ok := args["timeEntryID"]; ok {
				filters = append(filters, Filter{Op: "eq", Field: "timeEntryID", Value: v})
			}
			if v, ok := args["approvalLevel"]; ok {
				filters = append(filters, Filter{Op: "eq", Field: "approvalLevel", Value: v})
			}
			if v, ok := args["approvalDateTimeStart"]; ok {
				filters = append(filters, Filter{Op: "gte", Field: "approvalDateTime", Value: v})
			}
			if v, ok := args["approvalDateTimeEnd"]; ok {
				filters = append(filters, Filter{Op: "lte", Field: "approvalDateTime", Value: v})
			}
			items, err := client.Query(ctx, "BillingItemApprovalLevels", filters, QueryOpts{
				Page:     req.GetInt("page", 1),
				PageSize: req.GetInt("pageSize", 25),
				MaxSize:  500,
			})
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			if len(items) == 0 {
				return mcputil.TextResult(FormatNotFound("billing item approval levels", map[string]any{})), nil
			}
			return mcputil.TextResult(FormatSearchResult("search_billing_item_approval_levels", items, req.GetInt("page", 1), req.GetInt("pageSize", 25))), nil
		},
	)

	// === EXPENSE REPORTS ===

	// autotask_get_expense_report
	addTool(srv,
		mcp.NewTool("get_expense_report",
			mcp.WithDescription("Get a specific expense report by ID"),
			mcp.WithNumber("id", mcp.Description("Expense report ID"), mcp.Required()),
			mcp.WithReadOnlyHintAnnotation(true),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			id := req.GetInt("id", 0)
			if id == 0 {
				return mcputil.ErrorResult(fmt.Errorf("id is required")), nil
			}
			item, err := client.Get(ctx, "ExpenseReports", id)
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			if item == nil {
				return mcputil.TextResult(FormatNotFound("expense report", map[string]any{"id": id})), nil
			}
			enhanced := client.EnhanceWithNames(ctx, []map[string]any{item})
			return mcputil.TextResult(FormatGetResult(enhanced[0])), nil
		},
	)

	// autotask_search_expense_reports
	addTool(srv,
		mcp.NewTool("search_expense_reports",
			mcp.WithDescription("Search for expense reports in Autotask"),
			mcp.WithString("searchTerm", mcp.Description("Search by expense report name")),
			mcp.WithNumber("submitterID", mcp.Description("Filter by submitter resource ID")),
			mcp.WithNumber("status", mcp.Description("Filter by status")),
			mcp.WithNumber("page", mcp.Description("Page number"), mcp.Min(1)),
			mcp.WithNumber("pageSize", mcp.Description("Results per page"), mcp.Min(1), mcp.Max(500)),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(true),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.GetArguments()
			var filters []Filter
			if term := req.GetString("searchTerm", ""); term != "" {
				filters = append(filters, Filter{Op: "contains", Field: "name", Value: term})
			}
			if v, ok := args["submitterID"]; ok {
				filters = append(filters, Filter{Op: "eq", Field: "submitterID", Value: v})
			}
			if v, ok := args["status"]; ok {
				filters = append(filters, Filter{Op: "eq", Field: "status", Value: v})
			}
			items, err := client.Query(ctx, "ExpenseReports", filters, QueryOpts{
				Page:     req.GetInt("page", 1),
				PageSize: req.GetInt("pageSize", 25),
				MaxSize:  500,
			})
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			if len(items) == 0 {
				return mcputil.TextResult(FormatNotFound("expense reports", map[string]any{})), nil
			}
			items = client.EnhanceWithNames(ctx, items)
			return mcputil.TextResult(FormatSearchResult("search_expense_reports", items, req.GetInt("page", 1), req.GetInt("pageSize", 25))), nil
		},
	)

	// autotask_create_expense_report
	addTool(srv,
		mcp.NewTool("create_expense_report",
			mcp.WithDescription("Create a new expense report in Autotask"),
			mcp.WithString("name", mcp.Description("Expense report name"), mcp.Required()),
			mcp.WithNumber("submitterID", mcp.Description("Submitter resource ID"), mcp.Required()),
			mcp.WithNumber("status", mcp.Description("Status")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			data := map[string]any{
				"name":        req.GetString("name", ""),
				"submitterID": req.GetInt("submitterID", 0),
			}
			args := req.GetArguments()
			if v, ok := args["status"]; ok {
				data["status"] = v
			}
			id, err := client.Create(ctx, "ExpenseReports", data)
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.TextResult(FormatCreateResult("ExpenseReport", id)), nil
		},
	)

	// autotask_create_expense_item
	addTool(srv,
		mcp.NewTool("create_expense_item",
			mcp.WithDescription("Create a new expense item under an expense report"),
			mcp.WithNumber("expenseReportID", mcp.Description("Parent expense report ID"), mcp.Required()),
			mcp.WithString("description", mcp.Description("Description of the expense"), mcp.Required()),
			mcp.WithString("expenseDate", mcp.Description("Date of the expense (YYYY-MM-DD)"), mcp.Required()),
			mcp.WithNumber("expenseAmount", mcp.Description("Amount of the expense"), mcp.Required()),
			mcp.WithNumber("expenseCategory", mcp.Description("Expense category ID")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			parentID := req.GetInt("expenseReportID", 0)
			if parentID == 0 {
				return mcputil.ErrorResult(fmt.Errorf("expenseReportID is required")), nil
			}
			data := map[string]any{
				"expenseReportID": parentID,
				"description":     req.GetString("description", ""),
				"expenseDate":     req.GetString("expenseDate", ""),
				"expenseAmount":   req.GetFloat("expenseAmount", 0),
			}
			args := req.GetArguments()
			if v, ok := args["expenseCategory"]; ok {
				data["expenseCategory"] = v
			}
			id, err := client.CreateChild(ctx, "ExpenseReports", parentID, "Items", data)
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.TextResult(FormatCreateResult("ExpenseItem", id)), nil
		},
	)

	_ = server.ToolHandlerFunc(nil)
}
