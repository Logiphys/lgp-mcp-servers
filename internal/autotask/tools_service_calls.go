package autotask

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Logiphys/lgp-mcp/pkg/mcputil"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerServiceCallTools(srv *server.MCPServer, client *Client, _ *slog.Logger) {
	// === SERVICE CALLS ===

	// autotask_search_service_calls
	addTool(srv,
		mcp.NewTool("autotask_search_service_calls",
			mcp.WithDescription("Search for service calls in Autotask"),
			mcp.WithNumber("companyID", mcp.Description("Filter by company ID")),
			mcp.WithNumber("status", mcp.Description("Filter by status")),
			mcp.WithString("startDateTime", mcp.Description("Filter by start date/time (ISO 8601)")),
			mcp.WithString("endDateTime", mcp.Description("Filter by end date/time (ISO 8601)")),
			mcp.WithNumber("page", mcp.Description("Page number"), mcp.Min(1)),
			mcp.WithNumber("pageSize", mcp.Description("Results per page"), mcp.Min(1), mcp.Max(100)),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(true),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.GetArguments()
			var filters []Filter
			if v, ok := args["companyID"]; ok {
				filters = append(filters, Filter{Op: "eq", Field: "companyID", Value: v})
			}
			if v, ok := args["status"]; ok {
				filters = append(filters, Filter{Op: "eq", Field: "status", Value: v})
			}
			if v, ok := args["startDateTime"]; ok {
				filters = append(filters, Filter{Op: "gte", Field: "startDateTime", Value: v})
			}
			if v, ok := args["endDateTime"]; ok {
				filters = append(filters, Filter{Op: "lte", Field: "endDateTime", Value: v})
			}
			items, err := client.Query(ctx, "ServiceCalls", filters, QueryOpts{
				Page:     req.GetInt("page", 1),
				PageSize: req.GetInt("pageSize", 25),
				MaxSize:  100,
			})
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			if len(items) == 0 {
				return mcputil.TextResult(FormatNotFound("service calls", map[string]any{})), nil
			}
			items = client.EnhanceWithNames(ctx, items)
			return mcputil.TextResult(FormatSearchResult("autotask_search_service_calls", items, req.GetInt("page", 1), req.GetInt("pageSize", 25))), nil
		},
	)

	// autotask_get_service_call
	addTool(srv,
		mcp.NewTool("autotask_get_service_call",
			mcp.WithDescription("Get a specific service call by ID"),
			mcp.WithNumber("id", mcp.Description("Service call ID"), mcp.Required()),
			mcp.WithReadOnlyHintAnnotation(true),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			id := req.GetInt("id", 0)
			if id == 0 {
				return mcputil.ErrorResult(fmt.Errorf("id is required")), nil
			}
			item, err := client.Get(ctx, "ServiceCalls", id)
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			if item == nil {
				return mcputil.TextResult(FormatNotFound("service call", map[string]any{"id": id})), nil
			}
			enhanced := client.EnhanceWithNames(ctx, []map[string]any{item})
			return mcputil.TextResult(FormatGetResult(enhanced[0])), nil
		},
	)

	// autotask_create_service_call
	addTool(srv,
		mcp.NewTool("autotask_create_service_call",
			mcp.WithDescription("Create a new service call in Autotask"),
			mcp.WithNumber("companyID", mcp.Description("Company ID"), mcp.Required()),
			mcp.WithNumber("status", mcp.Description("Status"), mcp.Required()),
			mcp.WithString("startDateTime", mcp.Description("Start date/time (ISO 8601)"), mcp.Required()),
			mcp.WithString("description", mcp.Description("Description")),
			mcp.WithString("endDateTime", mcp.Description("End date/time (ISO 8601)")),
			mcp.WithNumber("duration", mcp.Description("Duration")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			data := map[string]any{
				"companyID":     req.GetInt("companyID", 0),
				"status":        req.GetInt("status", 0),
				"startDateTime": req.GetString("startDateTime", ""),
			}
			args := req.GetArguments()
			if v, ok := args["description"]; ok {
				data["description"] = v
			}
			if v, ok := args["endDateTime"]; ok {
				data["endDateTime"] = v
			}
			if v, ok := args["duration"]; ok {
				data["duration"] = v
			}
			id, err := client.Create(ctx, "ServiceCalls", data)
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.TextResult(FormatCreateResult("ServiceCall", id)), nil
		},
	)

	// autotask_update_service_call
	addTool(srv,
		mcp.NewTool("autotask_update_service_call",
			mcp.WithDescription("Update an existing service call"),
			mcp.WithNumber("id", mcp.Description("Service call ID"), mcp.Required()),
			mcp.WithNumber("companyID", mcp.Description("Company ID")),
			mcp.WithNumber("status", mcp.Description("Status")),
			mcp.WithString("description", mcp.Description("Description")),
			mcp.WithString("startDateTime", mcp.Description("Start date/time (ISO 8601)")),
			mcp.WithString("endDateTime", mcp.Description("End date/time (ISO 8601)")),
			mcp.WithNumber("duration", mcp.Description("Duration")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			id := req.GetInt("id", 0)
			if id == 0 {
				return mcputil.ErrorResult(fmt.Errorf("id is required")), nil
			}
			data := map[string]any{}
			args := req.GetArguments()
			if v, ok := args["companyID"]; ok {
				data["companyID"] = v
			}
			if v, ok := args["status"]; ok {
				data["status"] = v
			}
			if v, ok := args["description"]; ok {
				data["description"] = v
			}
			if v, ok := args["startDateTime"]; ok {
				data["startDateTime"] = v
			}
			if v, ok := args["endDateTime"]; ok {
				data["endDateTime"] = v
			}
			if v, ok := args["duration"]; ok {
				data["duration"] = v
			}
			if err := client.Update(ctx, "ServiceCalls", id, data); err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.TextResult(FormatUpdateResult("ServiceCall", id)), nil
		},
	)

	// autotask_delete_service_call
	addTool(srv,
		mcp.NewTool("autotask_delete_service_call",
			mcp.WithDescription("Delete a service call by ID"),
			mcp.WithNumber("id", mcp.Description("Service call ID"), mcp.Required()),
			mcp.WithDestructiveHintAnnotation(true),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			id := req.GetInt("id", 0)
			if id == 0 {
				return mcputil.ErrorResult(fmt.Errorf("id is required")), nil
			}
			if err := client.Delete(ctx, "ServiceCalls", id); err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.TextResult(FormatDeleteResult("ServiceCall", id)), nil
		},
	)

	// === SERVICE CALL TICKETS ===

	// autotask_search_service_call_tickets
	addTool(srv,
		mcp.NewTool("autotask_search_service_call_tickets",
			mcp.WithDescription("Search for tickets associated with a service call"),
			mcp.WithNumber("serviceCallID", mcp.Description("Service call ID"), mcp.Required()),
			mcp.WithNumber("pageSize", mcp.Description("Results per page"), mcp.Min(1), mcp.Max(100)),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(true),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			serviceCallID := req.GetInt("serviceCallID", 0)
			if serviceCallID == 0 {
				return mcputil.ErrorResult(fmt.Errorf("serviceCallID is required")), nil
			}
			filters := []Filter{
				{Op: "eq", Field: "serviceCallID", Value: serviceCallID},
			}
			pageSize := req.GetInt("pageSize", 25)
			items, err := client.Query(ctx, "ServiceCallTickets", filters, QueryOpts{PageSize: pageSize, MaxSize: 100})
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			if len(items) == 0 {
				return mcputil.TextResult(FormatNotFound("service call tickets", map[string]any{"serviceCallID": serviceCallID})), nil
			}
			return mcputil.TextResult(FormatSearchResult("autotask_search_service_call_tickets", items, 1, pageSize)), nil
		},
	)

	// autotask_create_service_call_ticket
	addTool(srv,
		mcp.NewTool("autotask_create_service_call_ticket",
			mcp.WithDescription("Associate a ticket with a service call"),
			mcp.WithNumber("serviceCallID", mcp.Description("Service call ID"), mcp.Required()),
			mcp.WithNumber("ticketID", mcp.Description("Ticket ID to associate"), mcp.Required()),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			parentID := req.GetInt("serviceCallID", 0)
			ticketID := req.GetInt("ticketID", 0)
			if parentID == 0 || ticketID == 0 {
				return mcputil.ErrorResult(fmt.Errorf("serviceCallID and ticketID are required")), nil
			}
			data := map[string]any{
				"serviceCallID": parentID,
				"ticketID":      ticketID,
			}
			id, err := client.CreateChild(ctx, "ServiceCalls", parentID, "Tickets", data)
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.TextResult(FormatCreateResult("ServiceCallTicket", id)), nil
		},
	)

	// autotask_delete_service_call_ticket
	addTool(srv,
		mcp.NewTool("autotask_delete_service_call_ticket",
			mcp.WithDescription("Remove a ticket association from a service call"),
			mcp.WithNumber("serviceCallID", mcp.Description("Service call ID"), mcp.Required()),
			mcp.WithNumber("ticketId", mcp.Description("Service call ticket association ID to delete"), mcp.Required()),
			mcp.WithDestructiveHintAnnotation(true),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			serviceCallID := req.GetInt("serviceCallID", 0)
			ticketID := req.GetInt("ticketId", 0)
			if serviceCallID == 0 || ticketID == 0 {
				return mcputil.ErrorResult(fmt.Errorf("serviceCallID and ticketId are required")), nil
			}
			if err := client.DeleteChild(ctx, "ServiceCalls", serviceCallID, "Tickets", ticketID); err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.TextResult(FormatDeleteResult("ServiceCallTicket", ticketID)), nil
		},
	)

	// === SERVICE CALL TICKET RESOURCES ===

	// autotask_search_service_call_ticket_resources
	addTool(srv,
		mcp.NewTool("autotask_search_service_call_ticket_resources",
			mcp.WithDescription("Search for resources associated with a service call ticket"),
			mcp.WithNumber("serviceCallTicketID", mcp.Description("Service call ticket association ID"), mcp.Required()),
			mcp.WithNumber("pageSize", mcp.Description("Results per page"), mcp.Min(1), mcp.Max(100)),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(true),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			scTicketID := req.GetInt("serviceCallTicketID", 0)
			if scTicketID == 0 {
				return mcputil.ErrorResult(fmt.Errorf("serviceCallTicketID is required")), nil
			}
			filters := []Filter{
				{Op: "eq", Field: "serviceCallTicketID", Value: scTicketID},
			}
			pageSize := req.GetInt("pageSize", 25)
			items, err := client.Query(ctx, "ServiceCallTicketResources", filters, QueryOpts{PageSize: pageSize, MaxSize: 100})
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			if len(items) == 0 {
				return mcputil.TextResult(FormatNotFound("service call ticket resources", map[string]any{"serviceCallTicketID": scTicketID})), nil
			}
			return mcputil.TextResult(FormatSearchResult("autotask_search_service_call_ticket_resources", items, 1, pageSize)), nil
		},
	)

	// autotask_create_service_call_ticket_resource
	addTool(srv,
		mcp.NewTool("autotask_create_service_call_ticket_resource",
			mcp.WithDescription("Associate a resource with a service call ticket"),
			mcp.WithNumber("serviceCallTicketID", mcp.Description("Service call ticket association ID"), mcp.Required()),
			mcp.WithNumber("resourceID", mcp.Description("Resource ID to associate"), mcp.Required()),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			parentID := req.GetInt("serviceCallTicketID", 0)
			resourceID := req.GetInt("resourceID", 0)
			if parentID == 0 || resourceID == 0 {
				return mcputil.ErrorResult(fmt.Errorf("serviceCallTicketID and resourceID are required")), nil
			}
			data := map[string]any{
				"serviceCallTicketID": parentID,
				"resourceID":          resourceID,
			}
			id, err := client.CreateChild(ctx, "ServiceCallTickets", parentID, "Resources", data)
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.TextResult(FormatCreateResult("ServiceCallTicketResource", id)), nil
		},
	)

	// autotask_delete_service_call_ticket_resource
	addTool(srv,
		mcp.NewTool("autotask_delete_service_call_ticket_resource",
			mcp.WithDescription("Remove a resource association from a service call ticket"),
			mcp.WithNumber("serviceCallTicketID", mcp.Description("Service call ticket association ID"), mcp.Required()),
			mcp.WithNumber("resourceId", mcp.Description("Resource association ID to delete"), mcp.Required()),
			mcp.WithDestructiveHintAnnotation(true),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			scTicketID := req.GetInt("serviceCallTicketID", 0)
			resourceID := req.GetInt("resourceId", 0)
			if scTicketID == 0 || resourceID == 0 {
				return mcputil.ErrorResult(fmt.Errorf("serviceCallTicketID and resourceId are required")), nil
			}
			if err := client.DeleteChild(ctx, "ServiceCallTickets", scTicketID, "Resources", resourceID); err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.TextResult(FormatDeleteResult("ServiceCallTicketResource", resourceID)), nil
		},
	)

	_ = server.ToolHandlerFunc(nil)
}
