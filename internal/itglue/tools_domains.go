package itglue

import (
	"context"
	"log/slog"

	"github.com/Logiphys/lgp-mcp/pkg/mcputil"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerDomainTools(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	srv.AddTool(
		mcp.NewTool("itglue_search_domains",
			mcp.WithDescription("Search IT Glue domains with optional filters. Returns a paginated list of domains."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(true),
			mcp.WithString("organization_id",
				mcp.Description("Filter by organization ID."),
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
			page := req.GetInt("page_number", 1)
			pageSize := req.GetInt("page_size", 50)

			items, meta, err := client.List(ctx, "/domains", filters, page, pageSize)
			if err != nil {
				logger.Error("itglue_search_domains failed", "error", err)
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
