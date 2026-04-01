package autotask

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Logiphys/lgp-mcp/pkg/mcputil"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerTicketTools(srv *server.MCPServer, client *Client, _ *PicklistCache, _ *slog.Logger) {
	// === TICKETS ===

	// autotask_search_tickets
	addTool(srv,
		mcp.NewTool("autotask_search_tickets",
			mcp.WithDescription("Search for tickets in Autotask (25 results per page default)"),
			mcp.WithString("searchTerm", mcp.Description("Search by ticket number (begins with)")),
			mcp.WithNumber("companyID", mcp.Description("Filter by company ID")),
			mcp.WithNumber("status", mcp.Description("Filter by status ID")),
			mcp.WithNumber("assignedResourceID", mcp.Description("Filter by assigned resource ID")),
			mcp.WithString("createdAfter", mcp.Description("Filter tickets created after this date (ISO 8601)")),
			mcp.WithString("createdBefore", mcp.Description("Filter tickets created before this date (ISO 8601)")),
			mcp.WithString("lastActivityAfter", mcp.Description("Filter by last activity after this date (ISO 8601)")),
			mcp.WithNumber("page", mcp.Description("Page number"), mcp.Min(1)),
			mcp.WithNumber("pageSize", mcp.Description("Results per page"), mcp.Min(1), mcp.Max(500)),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(true),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.GetArguments()
			var filters []Filter
			if term := req.GetString("searchTerm", ""); term != "" {
				filters = append(filters, Filter{Op: "beginsWith", Field: "ticketNumber", Value: term})
			}
			if v, ok := args["companyID"]; ok {
				filters = append(filters, Filter{Op: "eq", Field: "companyID", Value: v})
			}
			if v, ok := args["status"]; ok {
				filters = append(filters, Filter{Op: "eq", Field: "status", Value: v})
			}
			if v, ok := args["assignedResourceID"]; ok {
				filters = append(filters, Filter{Op: "eq", Field: "assignedResourceID", Value: v})
			}
			if v, ok := args["createdAfter"]; ok {
				filters = append(filters, Filter{Op: "gte", Field: "createDate", Value: v})
			}
			if v, ok := args["createdBefore"]; ok {
				filters = append(filters, Filter{Op: "lte", Field: "createDate", Value: v})
			}
			if v, ok := args["lastActivityAfter"]; ok {
				filters = append(filters, Filter{Op: "gte", Field: "lastActivityDate", Value: v})
			}
			items, err := client.Query(ctx, "Tickets", filters, QueryOpts{
				Page:     req.GetInt("page", 1),
				PageSize: req.GetInt("pageSize", 25),
				MaxSize:  500,
			})
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			if len(items) == 0 {
				return mcputil.TextResult(FormatNotFound("tickets", map[string]any{"searchTerm": req.GetString("searchTerm", "")})), nil
			}
			items = client.EnhanceWithNames(ctx, items)
			return mcputil.TextResult(FormatSearchResult("autotask_search_tickets", items, req.GetInt("page", 1), req.GetInt("pageSize", 25))), nil
		},
	)

	// autotask_get_ticket_details
	addTool(srv,
		mcp.NewTool("autotask_get_ticket_details",
			mcp.WithDescription("Get detailed information for a specific ticket by ID"),
			mcp.WithNumber("ticketID", mcp.Description("Ticket ID"), mcp.Required()),
			mcp.WithReadOnlyHintAnnotation(true),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			id := req.GetInt("ticketID", 0)
			if id == 0 {
				return mcputil.ErrorResult(fmt.Errorf("ticketID is required")), nil
			}
			item, err := client.Get(ctx, "Tickets", id)
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			if item == nil {
				return mcputil.TextResult(FormatNotFound("ticket", map[string]any{"id": id})), nil
			}
			enhanced := client.EnhanceWithNames(ctx, []map[string]any{item})
			return mcputil.TextResult(FormatGetResult(enhanced[0])), nil
		},
	)

	// autotask_create_ticket
	addTool(srv,
		mcp.NewTool("autotask_create_ticket",
			mcp.WithDescription("Create a new ticket in Autotask"),
			mcp.WithNumber("companyID", mcp.Description("Company ID"), mcp.Required()),
			mcp.WithString("title", mcp.Description("Ticket title"), mcp.Required()),
			mcp.WithString("description", mcp.Description("Ticket description"), mcp.Required()),
			mcp.WithNumber("status", mcp.Description("Status ID")),
			mcp.WithNumber("priority", mcp.Description("Priority ID")),
			mcp.WithNumber("assignedResourceID", mcp.Description("Assigned resource ID")),
			mcp.WithNumber("assignedResourceRoleID", mcp.Description("Assigned resource role ID")),
			mcp.WithNumber("contactID", mcp.Description("Contact ID")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			data := map[string]any{
				"companyID":   req.GetInt("companyID", 0),
				"title":       req.GetString("title", ""),
				"description": req.GetString("description", ""),
			}
			args := req.GetArguments()
			if v, ok := args["status"]; ok {
				data["status"] = v
			}
			if v, ok := args["priority"]; ok {
				data["priority"] = v
			}
			if v, ok := args["assignedResourceID"]; ok {
				data["assignedResourceID"] = v
			}
			if v, ok := args["assignedResourceRoleID"]; ok {
				data["assignedResourceRoleID"] = v
			}
			if v, ok := args["contactID"]; ok {
				data["contactID"] = v
			}
			id, err := client.Create(ctx, "Tickets", data)
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.TextResult(FormatCreateResult("Ticket", id)), nil
		},
	)

	// autotask_update_ticket
	addTool(srv,
		mcp.NewTool("autotask_update_ticket",
			mcp.WithDescription("Update an existing ticket in Autotask"),
			mcp.WithNumber("ticketId", mcp.Description("Ticket ID"), mcp.Required()),
			mcp.WithString("title", mcp.Description("Ticket title")),
			mcp.WithString("description", mcp.Description("Ticket description")),
			mcp.WithNumber("status", mcp.Description("Status ID")),
			mcp.WithNumber("priority", mcp.Description("Priority ID")),
			mcp.WithNumber("assignedResourceID", mcp.Description("Assigned resource ID")),
			mcp.WithNumber("assignedResourceRoleID", mcp.Description("Assigned resource role ID")),
			mcp.WithString("dueDateTime", mcp.Description("Due date/time (ISO 8601)")),
			mcp.WithNumber("contactID", mcp.Description("Contact ID")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			id := req.GetInt("ticketId", 0)
			if id == 0 {
				return mcputil.ErrorResult(fmt.Errorf("ticketId is required")), nil
			}
			data := map[string]any{}
			args := req.GetArguments()
			if v, ok := args["title"]; ok {
				data["title"] = v
			}
			if v, ok := args["description"]; ok {
				data["description"] = v
			}
			if v, ok := args["status"]; ok {
				data["status"] = v
			}
			if v, ok := args["priority"]; ok {
				data["priority"] = v
			}
			if v, ok := args["assignedResourceID"]; ok {
				data["assignedResourceID"] = v
			}
			if v, ok := args["assignedResourceRoleID"]; ok {
				data["assignedResourceRoleID"] = v
			}
			if v, ok := args["dueDateTime"]; ok {
				data["dueDateTime"] = v
			}
			if v, ok := args["contactID"]; ok {
				data["contactID"] = v
			}
			if err := client.Update(ctx, "Tickets", id, data); err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.TextResult(FormatUpdateResult("Ticket", id)), nil
		},
	)

	// === TICKET CHARGES ===

	// autotask_get_ticket_charge
	addTool(srv,
		mcp.NewTool("autotask_get_ticket_charge",
			mcp.WithDescription("Get a specific ticket charge by ID"),
			mcp.WithNumber("chargeId", mcp.Description("Charge ID"), mcp.Required()),
			mcp.WithReadOnlyHintAnnotation(true),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			id := req.GetInt("chargeId", 0)
			if id == 0 {
				return mcputil.ErrorResult(fmt.Errorf("chargeId is required")), nil
			}
			item, err := client.Get(ctx, "TicketCharges", id)
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			if item == nil {
				return mcputil.TextResult(FormatNotFound("ticket charge", map[string]any{"id": id})), nil
			}
			return mcputil.TextResult(FormatGetResult(item)), nil
		},
	)

	// autotask_search_ticket_charges
	addTool(srv,
		mcp.NewTool("autotask_search_ticket_charges",
			mcp.WithDescription("Search charges for a ticket"),
			mcp.WithNumber("ticketId", mcp.Description("Ticket ID")),
			mcp.WithNumber("page", mcp.Description("Page number"), mcp.Min(1)),
			mcp.WithNumber("pageSize", mcp.Description("Results per page"), mcp.Min(1), mcp.Max(100)),
			mcp.WithReadOnlyHintAnnotation(true),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.GetArguments()
			var filters []Filter
			defaultPageSize := 10
			if v, ok := args["ticketId"]; ok {
				filters = append(filters, Filter{Op: "eq", Field: "ticketID", Value: v})
				defaultPageSize = 25
			}
			pageSize := req.GetInt("pageSize", defaultPageSize)
			items, err := client.Query(ctx, "TicketCharges", filters, QueryOpts{
				Page:     req.GetInt("page", 1),
				PageSize: pageSize,
				MaxSize:  100,
			})
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			if len(items) == 0 {
				return mcputil.TextResult(FormatNotFound("ticket charges", map[string]any{"ticketId": req.GetInt("ticketId", 0)})), nil
			}
			return mcputil.TextResult(FormatSearchResult("autotask_search_ticket_charges", items, req.GetInt("page", 1), pageSize)), nil
		},
	)

	// autotask_create_ticket_charge
	addTool(srv,
		mcp.NewTool("autotask_create_ticket_charge",
			mcp.WithDescription("Create a new charge for a ticket"),
			mcp.WithNumber("ticketID", mcp.Description("Ticket ID"), mcp.Required()),
			mcp.WithString("name", mcp.Description("Charge name"), mcp.Required()),
			mcp.WithNumber("chargeType", mcp.Description("Charge type ID"), mcp.Required()),
			mcp.WithString("description", mcp.Description("Charge description")),
			mcp.WithNumber("unitQuantity", mcp.Description("Unit quantity")),
			mcp.WithNumber("unitPrice", mcp.Description("Unit price")),
			mcp.WithNumber("unitCost", mcp.Description("Unit cost")),
			mcp.WithString("datePurchased", mcp.Description("Date purchased (ISO 8601)")),
			mcp.WithNumber("productID", mcp.Description("Product ID")),
			mcp.WithNumber("billingCodeID", mcp.Description("Billing code ID")),
			mcp.WithBoolean("billableToAccount", mcp.Description("Billable to account (default: true)")),
			mcp.WithNumber("status", mcp.Description("Status")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			parentID := req.GetInt("ticketID", 0)
			if parentID == 0 {
				return mcputil.ErrorResult(fmt.Errorf("ticketID is required")), nil
			}
			data := map[string]any{
				"ticketID":          parentID,
				"name":              req.GetString("name", ""),
				"chargeType":        req.GetInt("chargeType", 0),
				"billableToAccount": true,
			}
			args := req.GetArguments()
			if v, ok := args["description"]; ok {
				data["description"] = v
			}
			if v, ok := args["unitQuantity"]; ok {
				data["unitQuantity"] = v
			}
			if v, ok := args["unitPrice"]; ok {
				data["unitPrice"] = v
			}
			if v, ok := args["unitCost"]; ok {
				data["unitCost"] = v
			}
			if v, ok := args["datePurchased"]; ok {
				data["datePurchased"] = v
			}
			if v, ok := args["productID"]; ok {
				data["productID"] = v
			}
			if v, ok := args["billingCodeID"]; ok {
				data["billingCodeID"] = v
			}
			if v, ok := args["billableToAccount"]; ok {
				data["billableToAccount"] = v
			}
			if v, ok := args["status"]; ok {
				data["status"] = v
			}
			id, err := client.CreateChild(ctx, "Tickets", parentID, "Charges", data)
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.TextResult(FormatCreateResult("TicketCharge", id)), nil
		},
	)

	// autotask_update_ticket_charge
	addTool(srv,
		mcp.NewTool("autotask_update_ticket_charge",
			mcp.WithDescription("Update a ticket charge"),
			mcp.WithNumber("chargeId", mcp.Description("Charge ID"), mcp.Required()),
			mcp.WithString("name", mcp.Description("Charge name")),
			mcp.WithString("description", mcp.Description("Charge description")),
			mcp.WithNumber("unitQuantity", mcp.Description("Unit quantity")),
			mcp.WithNumber("unitPrice", mcp.Description("Unit price")),
			mcp.WithNumber("unitCost", mcp.Description("Unit cost")),
			mcp.WithBoolean("billableToAccount", mcp.Description("Billable to account")),
			mcp.WithNumber("status", mcp.Description("Status")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			id := req.GetInt("chargeId", 0)
			if id == 0 {
				return mcputil.ErrorResult(fmt.Errorf("chargeId is required")), nil
			}
			data := map[string]any{}
			args := req.GetArguments()
			if v, ok := args["name"]; ok {
				data["name"] = v
			}
			if v, ok := args["description"]; ok {
				data["description"] = v
			}
			if v, ok := args["unitQuantity"]; ok {
				data["unitQuantity"] = v
			}
			if v, ok := args["unitPrice"]; ok {
				data["unitPrice"] = v
			}
			if v, ok := args["unitCost"]; ok {
				data["unitCost"] = v
			}
			if v, ok := args["billableToAccount"]; ok {
				data["billableToAccount"] = v
			}
			if v, ok := args["status"]; ok {
				data["status"] = v
			}
			if err := client.Update(ctx, "TicketCharges", id, data); err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.TextResult(FormatUpdateResult("TicketCharge", id)), nil
		},
	)

	// autotask_delete_ticket_charge
	addTool(srv,
		mcp.NewTool("autotask_delete_ticket_charge",
			mcp.WithDescription("Delete a ticket charge by ID"),
			mcp.WithNumber("ticketId", mcp.Description("Parent ticket ID"), mcp.Required()),
			mcp.WithNumber("chargeId", mcp.Description("Charge ID to delete"), mcp.Required()),
			mcp.WithDestructiveHintAnnotation(true),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			ticketID := req.GetInt("ticketId", 0)
			chargeID := req.GetInt("chargeId", 0)
			if ticketID == 0 || chargeID == 0 {
				return mcputil.ErrorResult(fmt.Errorf("ticketId and chargeId are required")), nil
			}
			if err := client.DeleteChild(ctx, "Tickets", ticketID, "Charges", chargeID); err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.TextResult(FormatDeleteResult("TicketCharge", chargeID)), nil
		},
	)

	// === TICKET NOTES ===

	// autotask_get_ticket_note
	addTool(srv,
		mcp.NewTool("autotask_get_ticket_note",
			mcp.WithDescription("Get a specific ticket note by ticket ID and note ID"),
			mcp.WithNumber("ticketId", mcp.Description("Ticket ID"), mcp.Required()),
			mcp.WithNumber("noteId", mcp.Description("Note ID"), mcp.Required()),
			mcp.WithReadOnlyHintAnnotation(true),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			ticketID := req.GetInt("ticketId", 0)
			noteID := req.GetInt("noteId", 0)
			if ticketID == 0 || noteID == 0 {
				return mcputil.ErrorResult(fmt.Errorf("ticketId and noteId are required")), nil
			}
			filters := []Filter{
				{Op: "eq", Field: "ticketID", Value: ticketID},
				{Op: "eq", Field: "id", Value: noteID},
			}
			items, err := client.Query(ctx, "TicketNotes", filters, QueryOpts{PageSize: 1, MaxSize: 1})
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			if len(items) == 0 {
				return mcputil.TextResult(FormatNotFound("ticket note", map[string]any{"ticketId": ticketID, "noteId": noteID})), nil
			}
			return mcputil.TextResult(FormatGetResult(items[0])), nil
		},
	)

	// autotask_search_ticket_notes
	addTool(srv,
		mcp.NewTool("autotask_search_ticket_notes",
			mcp.WithDescription("Search notes for a specific ticket"),
			mcp.WithNumber("ticketId", mcp.Description("Ticket ID"), mcp.Required()),
			mcp.WithNumber("pageSize", mcp.Description("Results per page"), mcp.Min(1), mcp.Max(100)),
			mcp.WithReadOnlyHintAnnotation(true),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			ticketID := req.GetInt("ticketId", 0)
			if ticketID == 0 {
				return mcputil.ErrorResult(fmt.Errorf("ticketId is required")), nil
			}
			filters := []Filter{
				{Op: "eq", Field: "ticketID", Value: ticketID},
			}
			pageSize := req.GetInt("pageSize", 25)
			items, err := client.Query(ctx, "TicketNotes", filters, QueryOpts{PageSize: pageSize, MaxSize: 100})
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			if len(items) == 0 {
				return mcputil.TextResult(FormatNotFound("ticket notes", map[string]any{"ticketId": ticketID})), nil
			}
			return mcputil.TextResult(FormatSearchResult("autotask_search_ticket_notes", items, 1, pageSize)), nil
		},
	)

	// autotask_create_ticket_note
	addTool(srv,
		mcp.NewTool("autotask_create_ticket_note",
			mcp.WithDescription("Create a new note for a ticket"),
			mcp.WithNumber("ticketId", mcp.Description("Ticket ID"), mcp.Required()),
			mcp.WithString("description", mcp.Description("Note description"), mcp.Required()),
			mcp.WithString("title", mcp.Description("Note title")),
			mcp.WithNumber("noteType", mcp.Description("1=General, 2=Appointment")),
			mcp.WithNumber("publish", mcp.Description("1=Internal Only, 2=All Users")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			parentID := req.GetInt("ticketId", 0)
			if parentID == 0 {
				return mcputil.ErrorResult(fmt.Errorf("ticketId is required")), nil
			}
			data := map[string]any{
				"ticketID":    parentID,
				"description": req.GetString("description", ""),
			}
			args := req.GetArguments()
			if v, ok := args["title"]; ok {
				data["title"] = v
			}
			if _, ok := args["noteType"]; ok {
				data["noteType"] = req.GetInt("noteType", 1)
			} else {
				data["noteType"] = 1
			}
			if _, ok := args["publish"]; ok {
				data["publish"] = req.GetInt("publish", 1)
			} else {
				data["publish"] = 1
			}
			id, err := client.CreateChild(ctx, "Tickets", parentID, "Notes", data)
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.TextResult(FormatCreateResult("TicketNote", id)), nil
		},
	)

	// === TICKET ATTACHMENTS ===

	// autotask_get_ticket_attachment
	addTool(srv,
		mcp.NewTool("autotask_get_ticket_attachment",
			mcp.WithDescription("Get a specific ticket attachment by ticket ID and attachment ID"),
			mcp.WithNumber("ticketId", mcp.Description("Ticket ID"), mcp.Required()),
			mcp.WithNumber("attachmentId", mcp.Description("Attachment ID"), mcp.Required()),
			mcp.WithReadOnlyHintAnnotation(true),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			ticketID := req.GetInt("ticketId", 0)
			attachmentID := req.GetInt("attachmentId", 0)
			if ticketID == 0 || attachmentID == 0 {
				return mcputil.ErrorResult(fmt.Errorf("ticketId and attachmentId are required")), nil
			}
			filters := []Filter{
				{Op: "eq", Field: "parentID", Value: ticketID},
				{Op: "eq", Field: "id", Value: attachmentID},
			}
			items, err := client.Query(ctx, "TicketAttachments", filters, QueryOpts{PageSize: 1, MaxSize: 1})
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			if len(items) == 0 {
				return mcputil.TextResult(FormatNotFound("ticket attachment", map[string]any{"ticketId": ticketID, "attachmentId": attachmentID})), nil
			}
			return mcputil.TextResult(FormatGetResult(items[0])), nil
		},
	)

	// autotask_search_ticket_attachments
	addTool(srv,
		mcp.NewTool("autotask_search_ticket_attachments",
			mcp.WithDescription("Search attachments for a specific ticket"),
			mcp.WithNumber("ticketId", mcp.Description("Ticket ID"), mcp.Required()),
			mcp.WithNumber("pageSize", mcp.Description("Results per page"), mcp.Min(1), mcp.Max(50)),
			mcp.WithReadOnlyHintAnnotation(true),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			ticketID := req.GetInt("ticketId", 0)
			if ticketID == 0 {
				return mcputil.ErrorResult(fmt.Errorf("ticketId is required")), nil
			}
			filters := []Filter{
				{Op: "eq", Field: "parentID", Value: ticketID},
			}
			pageSize := req.GetInt("pageSize", 10)
			items, err := client.Query(ctx, "TicketAttachments", filters, QueryOpts{PageSize: pageSize, MaxSize: 50})
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			if len(items) == 0 {
				return mcputil.TextResult(FormatNotFound("ticket attachments", map[string]any{"ticketId": ticketID})), nil
			}
			return mcputil.TextResult(FormatSearchResult("autotask_search_ticket_attachments", items, 1, pageSize)), nil
		},
	)

	// === RESOURCES ===

	// autotask_search_resources
	addTool(srv,
		mcp.NewTool("autotask_search_resources",
			mcp.WithDescription("Search for resources (technicians/users) in Autotask"),
			mcp.WithString("searchTerm", mcp.Description("Search term for email, first name, or last name")),
			mcp.WithBoolean("isActive", mcp.Description("Filter by active status")),
			mcp.WithNumber("page", mcp.Description("Page number"), mcp.Min(1)),
			mcp.WithNumber("pageSize", mcp.Description("Results per page"), mcp.Min(1), mcp.Max(500)),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(true),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.GetArguments()
			var filters []Filter
			if term := req.GetString("searchTerm", ""); term != "" {
				filters = append(filters, Filter{
					Op: "or",
					Items: []Filter{
						{Op: "contains", Field: "email", Value: term},
						{Op: "contains", Field: "firstName", Value: term},
						{Op: "contains", Field: "lastName", Value: term},
					},
				})
			}
			if v, ok := args["isActive"]; ok {
				filters = append(filters, Filter{Op: "eq", Field: "isActive", Value: v})
			}
			items, err := client.Query(ctx, "Resources", filters, QueryOpts{
				Page:     req.GetInt("page", 1),
				PageSize: req.GetInt("pageSize", 25),
				MaxSize:  500,
			})
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			if len(items) == 0 {
				return mcputil.TextResult(FormatNotFound("resources", map[string]any{"searchTerm": req.GetString("searchTerm", "")})), nil
			}
			return mcputil.TextResult(FormatSearchResult("autotask_search_resources", items, req.GetInt("page", 1), req.GetInt("pageSize", 25))), nil
		},
	)

	// === CONFIGURATION ITEMS ===

	// autotask_search_configuration_items
	addTool(srv,
		mcp.NewTool("autotask_search_configuration_items",
			mcp.WithDescription("Search for configuration items (assets) in Autotask"),
			mcp.WithString("searchTerm", mcp.Description("Search by serial number or reference title")),
			mcp.WithNumber("companyID", mcp.Description("Filter by company ID")),
			mcp.WithBoolean("isActive", mcp.Description("Filter by active status")),
			mcp.WithNumber("productID", mcp.Description("Filter by product ID")),
			mcp.WithNumber("page", mcp.Description("Page number"), mcp.Min(1)),
			mcp.WithNumber("pageSize", mcp.Description("Results per page"), mcp.Min(1), mcp.Max(500)),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(true),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.GetArguments()
			var filters []Filter
			if term := req.GetString("searchTerm", ""); term != "" {
				filters = append(filters, Filter{
					Op: "or",
					Items: []Filter{
						{Op: "contains", Field: "serialNumber", Value: term},
						{Op: "contains", Field: "referenceTitle", Value: term},
					},
				})
			}
			if v, ok := args["companyID"]; ok {
				filters = append(filters, Filter{Op: "eq", Field: "companyID", Value: v})
			}
			if v, ok := args["isActive"]; ok {
				filters = append(filters, Filter{Op: "eq", Field: "isActive", Value: v})
			}
			if v, ok := args["productID"]; ok {
				filters = append(filters, Filter{Op: "eq", Field: "productID", Value: v})
			}
			items, err := client.Query(ctx, "ConfigurationItems", filters, QueryOpts{
				Page:     req.GetInt("page", 1),
				PageSize: req.GetInt("pageSize", 25),
				MaxSize:  500,
			})
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			if len(items) == 0 {
				return mcputil.TextResult(FormatNotFound("configuration items", map[string]any{"searchTerm": req.GetString("searchTerm", "")})), nil
			}
			items = client.EnhanceWithNames(ctx, items)
			return mcputil.TextResult(FormatSearchResult("autotask_search_configuration_items", items, req.GetInt("page", 1), req.GetInt("pageSize", 25))), nil
		},
	)

	_ = server.ToolHandlerFunc(nil)
}
