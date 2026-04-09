package itglue

import (
	"context"
	"log/slog"

	"github.com/Logiphys/lgp-mcp-servers/pkg/mcputil"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerExpirationTools(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	srv.AddTool(
		mcp.NewTool("list_expirations",
			mcp.WithDescription("List expiring items in IT Glue with optional filters. Returns a paginated list of expirations."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(true),
			mcp.WithString("organization_id",
				mcp.Description("Filter by organization ID."),
			),
			mcp.WithString("resource_type",
				mcp.Description("Filter by resource type (e.g. configurations, passwords, domains, ssl_certificates)."),
			),
			mcp.WithString("range",
				mcp.Description("Filter by expiration range: past, this_month, next_month."),
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
			if v := req.GetString("resource_type", ""); v != "" {
				filters["resource_type"] = v
			}
			if v := req.GetString("range", ""); v != "" {
				filters["range"] = v
			}
			page := req.GetInt("page_number", 1)
			pageSize := req.GetInt("page_size", 50)

			items, meta, err := client.List(ctx, "/expirations", filters, page, pageSize)
			if err != nil {
				logger.Error("itglue_list_expirations failed", "error", err)
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
}
