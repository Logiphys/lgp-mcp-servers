package itglue

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Logiphys/lgp-mcp-servers/pkg/mcputil"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerFlexibleAssetTools(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	srv.AddTool(
		mcp.NewTool("itglue_list_flexible_asset_types",
			mcp.WithDescription("List all flexible asset types defined in IT Glue. Optionally filter by organization_id."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(true),
			mcp.WithString("organization_id",
				mcp.Description("Optional: filter flexible asset types by organization ID."),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			filters := make(map[string]string)
			if v := req.GetString("organization_id", ""); v != "" {
				filters["organization_id"] = v
			}

			items, meta, err := client.List(ctx, "/flexible_asset_types", filters, 1, 100)
			if err != nil {
				logger.Error("itglue_list_flexible_asset_types failed", "error", err)
				return mcputil.ErrorResult(err), nil
			}
			if len(items) == 0 {
				return mcputil.TextResult("No flexible asset types found."), nil
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
		mcp.NewTool("itglue_search_flexible_assets",
			mcp.WithDescription("Search IT Glue flexible assets. flexible_asset_type_id is required. Optionally filter by organization_id and name."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(true),
			mcp.WithString("flexible_asset_type_id",
				mcp.Description("The flexible asset type ID to filter by (required)."),
				mcp.Required(),
			),
			mcp.WithString("organization_id",
				mcp.Description("Optional: filter by organization ID."),
			),
			mcp.WithString("name",
				mcp.Description("Optional: filter by asset name (exact match)."),
			),
			mcp.WithNumber("page_number",
				mcp.Description("Page number to retrieve (default: 1)."),
			),
			mcp.WithNumber("page_size",
				mcp.Description("Number of results per page (default: 50)."),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			typeID := req.GetString("flexible_asset_type_id", "")
			if typeID == "" {
				return mcputil.ErrorResult(fmt.Errorf("flexible_asset_type_id is required")), nil
			}

			filters := map[string]string{
				"flexible_asset_type_id": typeID,
			}
			if v := req.GetString("organization_id", ""); v != "" {
				filters["organization_id"] = v
			}
			if v := req.GetString("name", ""); v != "" {
				filters["name"] = v
			}
			page := req.GetInt("page_number", 1)
			pageSize := req.GetInt("page_size", 50)

			items, meta, err := client.List(ctx, "/flexible_assets", filters, page, pageSize)
			if err != nil {
				logger.Error("itglue_search_flexible_assets failed", "flexible_asset_type_id", typeID, "error", err)
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
