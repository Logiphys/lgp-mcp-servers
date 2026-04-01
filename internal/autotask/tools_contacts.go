package autotask

import (
	"context"
	"log/slog"

	"github.com/Logiphys/lgp-mcp/pkg/mcputil"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerContactTools(srv *server.MCPServer, client *Client, _ *slog.Logger) {
	// autotask_search_contacts
	addTool(srv,
		mcp.NewTool("autotask_search_contacts",
			mcp.WithDescription("Search for contacts in Autotask (25 results per page default)"),
			mcp.WithString("searchTerm", mcp.Description("Search term for first name, last name, or email")),
			mcp.WithNumber("companyID", mcp.Description("Filter by company ID")),
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
				filters = append(filters, Filter{
					Op: "or",
					Items: []Filter{
						{Op: "contains", Field: "firstName", Value: term},
						{Op: "contains", Field: "lastName", Value: term},
						{Op: "contains", Field: "emailAddress", Value: term},
					},
				})
			}
			if v, ok := args["companyID"]; ok {
				filters = append(filters, Filter{Op: "eq", Field: "companyID", Value: v})
			}
			if v, ok := args["isActive"]; ok {
				filters = append(filters, Filter{Op: "eq", Field: "isActive", Value: v})
			}
			items, err := client.Query(ctx, "Contacts", filters, QueryOpts{
				Page:     req.GetInt("page", 1),
				PageSize: req.GetInt("pageSize", 25),
				MaxSize:  200,
			})
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			if len(items) == 0 {
				return mcputil.TextResult(FormatNotFound("contacts", map[string]any{"searchTerm": req.GetString("searchTerm", "")})), nil
			}
			items = client.EnhanceWithNames(ctx, items)
			return mcputil.TextResult(FormatSearchResult("autotask_search_contacts", items, req.GetInt("page", 1), req.GetInt("pageSize", 25))), nil
		},
	)

	// autotask_create_contact
	addTool(srv,
		mcp.NewTool("autotask_create_contact",
			mcp.WithDescription("Create a new contact in Autotask"),
			mcp.WithNumber("companyID", mcp.Description("Company ID"), mcp.Required()),
			mcp.WithString("firstName", mcp.Description("First name"), mcp.Required()),
			mcp.WithString("lastName", mcp.Description("Last name"), mcp.Required()),
			mcp.WithString("emailAddress", mcp.Description("Email address")),
			mcp.WithString("phone", mcp.Description("Phone number")),
			mcp.WithString("title", mcp.Description("Job title")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			data := map[string]any{
				"companyID": req.GetInt("companyID", 0),
				"firstName": req.GetString("firstName", ""),
				"lastName":  req.GetString("lastName", ""),
			}
			args := req.GetArguments()
			if v, ok := args["emailAddress"]; ok {
				data["emailAddress"] = v
			}
			if v, ok := args["phone"]; ok {
				data["phone"] = v
			}
			if v, ok := args["title"]; ok {
				data["title"] = v
			}
			id, err := client.Create(ctx, "Contacts", data)
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.TextResult(FormatCreateResult("Contact", id)), nil
		},
	)

	_ = server.ToolHandlerFunc(nil)
}
