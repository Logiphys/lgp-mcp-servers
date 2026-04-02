package itglue

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Logiphys/lgp-mcp/pkg/mcputil"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerConfigurationInterfaceTools(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	srv.AddTool(
		mcp.NewTool("itglue_search_configuration_interfaces",
			mcp.WithDescription("Search IT Glue configuration interfaces (network adapters, IPs, MACs) for a specific configuration. Requires configuration_id."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(true),
			mcp.WithString("configuration_id",
				mcp.Description("The configuration ID to list interfaces for (required)."),
				mcp.Required(),
			),
			mcp.WithNumber("page_number",
				mcp.Description("Page number to retrieve (default: 1)."),
			),
			mcp.WithNumber("page_size",
				mcp.Description("Number of results per page (default: 50)."),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			configID := req.GetString("configuration_id", "")
			if configID == "" {
				return mcputil.ErrorResult(fmt.Errorf("configuration_id is required")), nil
			}

			filters := make(map[string]string)
			page := req.GetInt("page_number", 1)
			pageSize := req.GetInt("page_size", 50)

			path := fmt.Sprintf("/configurations/%s/relationships/configuration_interfaces", configID)
			items, meta, err := client.List(ctx, path, filters, page, pageSize)
			if err != nil {
				logger.Error("itglue_search_configuration_interfaces failed", "error", err)
				return mcputil.ErrorResult(err), nil
			}
			if len(items) == 0 {
				return mcputil.TextResult("No configuration interfaces found."), nil
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
