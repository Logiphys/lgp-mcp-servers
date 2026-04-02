package itglue

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Logiphys/lgp-mcp/pkg/mcputil"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerPasswordTools(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	srv.AddTool(
		mcp.NewTool("itglue_search_passwords",
			mcp.WithDescription("Search IT Glue passwords with optional filters. Returns metadata only — actual password values are not included in results."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(true),
			mcp.WithString("organization_id",
				mcp.Description("Filter by organization ID."),
			),
			mcp.WithString("name",
				mcp.Description("Filter by password name (exact match)."),
			),
			mcp.WithString("password_category_id",
				mcp.Description("Filter by password category ID."),
			),
			mcp.WithString("url",
				mcp.Description("Filter by URL associated with the password."),
			),
			mcp.WithString("username",
				mcp.Description("Filter by username."),
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
			if v := req.GetString("name", ""); v != "" {
				filters["name"] = v
			}
			if v := req.GetString("password_category_id", ""); v != "" {
				filters["password_category_id"] = v
			}
			if v := req.GetString("url", ""); v != "" {
				filters["url"] = v
			}
			if v := req.GetString("username", ""); v != "" {
				filters["username"] = v
			}
			page := req.GetInt("page_number", 1)
			pageSize := req.GetInt("page_size", 50)

			items, meta, err := client.List(ctx, "/passwords", filters, page, pageSize)
			if err != nil {
				logger.Error("itglue_search_passwords failed", "error", err)
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
		mcp.NewTool("itglue_get_password",
			mcp.WithDescription("Get a single IT Glue password entry by ID. Use show_password=true to include the actual password value in the response."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithNumber("id",
				mcp.Description("The password ID."),
				mcp.Required(),
			),
			mcp.WithBoolean("show_password",
				mcp.Description("If true, include the actual password value in the response (default: false)."),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			id := req.GetInt("id", 0)
			if id == 0 {
				return mcputil.ErrorResult(fmt.Errorf("id is required")), nil
			}

			path := fmt.Sprintf("/passwords/%d", id)

			args := req.GetArguments()
			if showPw, ok := args["show_password"]; ok {
				if v, ok := showPw.(bool); ok && v {
					path += "?show_password=true"
				}
			}

			item, err := client.Get(ctx, path)
			if err != nil {
				logger.Error("itglue_get_password failed", "id", id, "error", err)
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.JSONResult(item), nil
		},
	)
}
