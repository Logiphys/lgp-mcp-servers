package itglue

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Logiphys/lgp-mcp/pkg/mcputil"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerConfigurationTools(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	srv.AddTool(
		mcp.NewTool("itglue_search_configurations",
			mcp.WithDescription("Search IT Glue configurations with optional filters. Returns a paginated list of configuration items."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(true),
			mcp.WithString("organization_id",
				mcp.Description("Filter by organization ID."),
			),
			mcp.WithString("name",
				mcp.Description("Filter by configuration name (contains match)."),
			),
			mcp.WithString("configuration_type_id",
				mcp.Description("Filter by configuration type ID."),
			),
			mcp.WithString("configuration_status_id",
				mcp.Description("Filter by configuration status ID."),
			),
			mcp.WithString("serial_number",
				mcp.Description("Filter by serial number."),
			),
			mcp.WithString("rmm_id",
				mcp.Description("Filter by RMM integration ID."),
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
				filters["filter[organization_id]"] = v
			}
			if v := req.GetString("name", ""); v != "" {
				filters["filter[name]"] = v
			}
			if v := req.GetString("configuration_type_id", ""); v != "" {
				filters["filter[configuration_type_id]"] = v
			}
			if v := req.GetString("configuration_status_id", ""); v != "" {
				filters["filter[configuration_status_id]"] = v
			}
			if v := req.GetString("serial_number", ""); v != "" {
				filters["filter[serial_number]"] = v
			}
			if v := req.GetString("rmm_id", ""); v != "" {
				filters["filter[rmm_id]"] = v
			}
			if v := req.GetString("psa_id", ""); v != "" {
				filters["filter[psa_id]"] = v
			}
			page := req.GetInt("page_number", 1)
			pageSize := req.GetInt("page_size", 50)

			items, meta, err := client.List(ctx, "/configurations", filters, page, pageSize)
			if err != nil {
				logger.Error("itglue_search_configurations failed", "error", err)
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
		mcp.NewTool("itglue_get_configuration",
			mcp.WithDescription("Get a single IT Glue configuration item by ID."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithNumber("id",
				mcp.Description("The configuration ID."),
				mcp.Required(),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			id := req.GetInt("id", 0)
			if id == 0 {
				return mcputil.ErrorResult(fmt.Errorf("id is required")), nil
			}
			item, err := client.Get(ctx, fmt.Sprintf("/configurations/%d", id))
			if err != nil {
				logger.Error("itglue_get_configuration failed", "id", id, "error", err)
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.JSONResult(item), nil
		},
	)
}
