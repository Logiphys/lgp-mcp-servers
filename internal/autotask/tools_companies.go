package autotask

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Logiphys/lgp-mcp/pkg/mcputil"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerCompanyTools(srv *server.MCPServer, client *Client, _ *slog.Logger) {
	// autotask_search_companies
	addTool(srv,
		mcp.NewTool("autotask_search_companies",
			mcp.WithDescription("Search for companies in Autotask (25 results per page default)"),
			mcp.WithString("searchTerm", mcp.Description("Search term for company name")),
			mcp.WithBoolean("isActive", mcp.Description("Filter by active status")),
			mcp.WithNumber("page", mcp.Description("Page number"), mcp.Min(1)),
			mcp.WithNumber("pageSize", mcp.Description("Results per page"), mcp.Min(1), mcp.Max(200)),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(true),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.GetArguments()
			var filters []Filter
			if term := req.GetString("searchTerm", ""); term != "" {
				filters = append(filters, Filter{Op: "contains", Field: "companyName", Value: term})
			}
			if v, ok := args["isActive"]; ok {
				filters = append(filters, Filter{Op: "eq", Field: "isActive", Value: v})
			}
			items, err := client.Query(ctx, "Companies", filters, QueryOpts{
				Page:     req.GetInt("page", 1),
				PageSize: req.GetInt("pageSize", 25),
				MaxSize:  200,
			})
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			if len(items) == 0 {
				return mcputil.TextResult(FormatNotFound("companies", map[string]any{"searchTerm": req.GetString("searchTerm", "")})), nil
			}
			items = client.EnhanceWithNames(ctx, items)
			return mcputil.TextResult(FormatSearchResult("autotask_search_companies", items, req.GetInt("page", 1), req.GetInt("pageSize", 25))), nil
		},
	)

	// autotask_create_company
	addTool(srv,
		mcp.NewTool("autotask_create_company",
			mcp.WithDescription("Create a new company in Autotask"),
			mcp.WithString("companyName", mcp.Description("Company name"), mcp.Required()),
			mcp.WithNumber("companyType", mcp.Description("Company type ID"), mcp.Required()),
			mcp.WithString("phone", mcp.Description("Phone number")),
			mcp.WithString("address1", mcp.Description("Street address")),
			mcp.WithString("city", mcp.Description("City")),
			mcp.WithString("state", mcp.Description("State/Province")),
			mcp.WithString("postalCode", mcp.Description("Postal/ZIP code")),
			mcp.WithNumber("ownerResourceID", mcp.Description("Owner resource ID")),
			mcp.WithBoolean("isActive", mcp.Description("Active status")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			data := map[string]any{
				"companyName": req.GetString("companyName", ""),
				"companyType": req.GetInt("companyType", 0),
			}
			args := req.GetArguments()
			if v, ok := args["phone"]; ok {
				data["phone"] = v
			}
			if v, ok := args["address1"]; ok {
				data["address1"] = v
			}
			if v, ok := args["city"]; ok {
				data["city"] = v
			}
			if v, ok := args["state"]; ok {
				data["state"] = v
			}
			if v, ok := args["postalCode"]; ok {
				data["postalCode"] = v
			}
			if v, ok := args["ownerResourceID"]; ok {
				data["ownerResourceID"] = v
			}
			if v, ok := args["isActive"]; ok {
				data["isActive"] = v
			}
			id, err := client.Create(ctx, "Companies", data)
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.TextResult(FormatCreateResult("Company", id)), nil
		},
	)

	// autotask_update_company
	addTool(srv,
		mcp.NewTool("autotask_update_company",
			mcp.WithDescription("Update an existing company in Autotask"),
			mcp.WithNumber("id", mcp.Description("Company ID"), mcp.Required()),
			mcp.WithString("companyName", mcp.Description("Company name")),
			mcp.WithString("phone", mcp.Description("Phone number")),
			mcp.WithString("address1", mcp.Description("Street address")),
			mcp.WithString("city", mcp.Description("City")),
			mcp.WithString("state", mcp.Description("State/Province")),
			mcp.WithString("postalCode", mcp.Description("Postal/ZIP code")),
			mcp.WithBoolean("isActive", mcp.Description("Active status")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			id := req.GetInt("id", 0)
			if id == 0 {
				return mcputil.ErrorResult(fmt.Errorf("id is required")), nil
			}
			data := map[string]any{}
			args := req.GetArguments()
			if v, ok := args["companyName"]; ok {
				data["companyName"] = v
			}
			if v, ok := args["phone"]; ok {
				data["phone"] = v
			}
			if v, ok := args["address1"]; ok {
				data["address1"] = v
			}
			if v, ok := args["city"]; ok {
				data["city"] = v
			}
			if v, ok := args["state"]; ok {
				data["state"] = v
			}
			if v, ok := args["postalCode"]; ok {
				data["postalCode"] = v
			}
			if v, ok := args["isActive"]; ok {
				data["isActive"] = v
			}
			if err := client.Update(ctx, "Companies", id, data); err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.TextResult(FormatUpdateResult("Company", id)), nil
		},
	)

	// Company Notes
	// autotask_get_company_note
	addTool(srv,
		mcp.NewTool("autotask_get_company_note",
			mcp.WithDescription("Get a specific company note by company ID and note ID"),
			mcp.WithNumber("companyId", mcp.Description("Company ID"), mcp.Required()),
			mcp.WithNumber("noteId", mcp.Description("Note ID"), mcp.Required()),
			mcp.WithReadOnlyHintAnnotation(true),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			companyID := req.GetInt("companyId", 0)
			noteID := req.GetInt("noteId", 0)
			if companyID == 0 || noteID == 0 {
				return mcputil.ErrorResult(fmt.Errorf("companyId and noteId are required")), nil
			}
			filters := []Filter{
				{Op: "eq", Field: "accountId", Value: companyID},
				{Op: "eq", Field: "id", Value: noteID},
			}
			items, err := client.Query(ctx, "CompanyNotes", filters, QueryOpts{PageSize: 1, MaxSize: 1})
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			if len(items) == 0 {
				return mcputil.TextResult(FormatNotFound("company note", map[string]any{"companyId": companyID, "noteId": noteID})), nil
			}
			return mcputil.TextResult(FormatGetResult(items[0])), nil
		},
	)

	// autotask_search_company_notes
	addTool(srv,
		mcp.NewTool("autotask_search_company_notes",
			mcp.WithDescription("Search notes for a specific company"),
			mcp.WithNumber("companyId", mcp.Description("Company ID"), mcp.Required()),
			mcp.WithNumber("pageSize", mcp.Description("Results per page"), mcp.Min(1), mcp.Max(100)),
			mcp.WithReadOnlyHintAnnotation(true),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			companyID := req.GetInt("companyId", 0)
			if companyID == 0 {
				return mcputil.ErrorResult(fmt.Errorf("companyId is required")), nil
			}
			filters := []Filter{
				{Op: "eq", Field: "accountId", Value: companyID},
			}
			pageSize := req.GetInt("pageSize", 25)
			items, err := client.Query(ctx, "CompanyNotes", filters, QueryOpts{PageSize: pageSize, MaxSize: 100})
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			if len(items) == 0 {
				return mcputil.TextResult(FormatNotFound("company notes", map[string]any{"companyId": companyID})), nil
			}
			return mcputil.TextResult(FormatSearchResult("autotask_search_company_notes", items, 1, pageSize)), nil
		},
	)

	// autotask_create_company_note
	addTool(srv,
		mcp.NewTool("autotask_create_company_note",
			mcp.WithDescription("Create a new note for a company"),
			mcp.WithNumber("companyId", mcp.Description("Company ID"), mcp.Required()),
			mcp.WithString("description", mcp.Description("Note description"), mcp.Required()),
			mcp.WithString("title", mcp.Description("Note title")),
			mcp.WithNumber("actionType", mcp.Description("Action type")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			parentID := req.GetInt("companyId", 0)
			if parentID == 0 {
				return mcputil.ErrorResult(fmt.Errorf("companyId is required")), nil
			}
			data := map[string]any{
				"accountId":   parentID,
				"description": req.GetString("description", ""),
			}
			args := req.GetArguments()
			if v, ok := args["title"]; ok {
				data["title"] = v
			}
			if v, ok := args["actionType"]; ok {
				data["actionType"] = v
			}
			id, err := client.CreateChild(ctx, "Companies", parentID, "Notes", data)
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.TextResult(FormatCreateResult("CompanyNote", id)), nil
		},
	)

	_ = server.ToolHandlerFunc(nil)
}
