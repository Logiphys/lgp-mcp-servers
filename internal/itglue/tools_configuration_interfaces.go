package itglue

import (
	"context"
	"log/slog"

	"github.com/Logiphys/lgp-mcp/pkg/mcputil"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerConfigurationInterfaceTools(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	srv.AddTool(
		mcp.NewTool("itglue_search_configuration_interfaces",
			mcp.WithDescription("Search IT Glue configuration interfaces with optional filters. Returns a paginated list of configuration interfaces."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(true),
			mcp.WithString("configuration_id",
				mcp.Description("Filter by configuration ID."),
			),
			mcp.WithString("ip_address",
				mcp.Description("Filter by IP address."),
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
			if v := req.GetString("configuration_id", ""); v != "" {
				filters["configuration_id"] = v
			}
			if v := req.GetString("ip_address", ""); v != "" {
				filters["ip_address"] = v
			}
			page := req.GetInt("page_number", 1)
			pageSize := req.GetInt("page_size", 50)

			items, meta, err := client.List(ctx, "/configuration_interfaces", filters, page, pageSize)
			if err != nil {
				logger.Error("itglue_search_configuration_interfaces failed", "error", err)
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
