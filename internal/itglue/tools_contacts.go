package itglue

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Logiphys/lgp-mcp/pkg/mcputil"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerContactTools(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	srv.AddTool(
		mcp.NewTool("itglue_search_contacts",
			mcp.WithDescription("Search IT Glue contacts with optional filters. Returns a paginated list of contacts."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(true),
			mcp.WithString("organization_id",
				mcp.Description("Filter by organization ID."),
			),
			mcp.WithString("first_name",
				mcp.Description("Filter by first name."),
			),
			mcp.WithString("last_name",
				mcp.Description("Filter by last name."),
			),
			mcp.WithString("title",
				mcp.Description("Filter by title."),
			),
			mcp.WithString("contact_type_id",
				mcp.Description("Filter by contact type ID."),
			),
			mcp.WithString("psa_id",
				mcp.Description("Filter by PSA integration ID."),
			),
			mcp.WithNumber("page_number",
				mcp.Description("Page number to retrieve (default: 1)."),
			),
			mcp.WithNumber("page_size",
				mcp.Description("Number of results per page (default: 50)."),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			filters := make(map[string]string)
			if v := req.GetString("organization_id", ""); v != "" {
				filters["organization_id"] = v
			}
			if v := req.GetString("first_name", ""); v != "" {
				filters["first_name"] = v
			}
			if v := req.GetString("last_name", ""); v != "" {
				filters["last_name"] = v
			}
			if v := req.GetString("title", ""); v != "" {
				filters["title"] = v
			}
			if v := req.GetString("contact_type_id", ""); v != "" {
				filters["contact_type_id"] = v
			}
			if v := req.GetString("psa_id", ""); v != "" {
				filters["psa_id"] = v
			}
			page := req.GetInt("page_number", 1)
			pageSize := req.GetInt("page_size", 50)

			items, meta, err := client.List(ctx, "/contacts", filters, page, pageSize)
			if err != nil {
				logger.Error("itglue_search_contacts failed", "error", err)
				return mcputil.ErrorResult(err), nil
			}
			if len(items) == 0 {
				return mcputil.TextResult("No results found."), nil
			}
			result := map[string]any{
				"items": items,
				"pagination": map[string]any{
					"current_page": meta.CurrentPage,
					"total_pages":  meta.TotalPages,
					"total_count":  meta.TotalCount,
				},
			}
			return mcputil.JSONResult(result), nil
		},
	)

	srv.AddTool(
		mcp.NewTool("itglue_get_contact",
			mcp.WithDescription("Get a single IT Glue contact by ID."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithNumber("id",
				mcp.Description("The contact ID."),
				mcp.Required(),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			id := req.GetInt("id", 0)
			if id == 0 {
				return mcputil.ErrorResult(fmt.Errorf("id is required")), nil
			}
			item, err := client.Get(ctx, fmt.Sprintf("/contacts/%d", id))
			if err != nil {
				logger.Error("itglue_get_contact failed", "id", id, "error", err)
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.JSONResult(item), nil
		},
	)
}
