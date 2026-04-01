package dattormm

import (
	"context"
	"log/slog"

	"github.com/Logiphys/lgp-mcp/pkg/mcputil"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerAccountTools(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	// datto_get_account
	srv.AddTool(
		mcp.NewTool("datto_get_account",
			mcp.WithDescription("Get Datto RMM account information."),
			mcp.WithReadOnlyHintAnnotation(true),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			result, err := client.Get(ctx, "/account", nil)
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.JSONResult(result), nil
		},
	)

	// datto_list_sites
	srv.AddTool(
		mcp.NewTool("datto_list_sites",
			mcp.WithDescription("List all sites in the Datto RMM account."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithString("siteName",
				mcp.Description("Filter sites by name (partial match)."),
			),
			mcp.WithNumber("page",
				mcp.Description("Page number for pagination."),
				mcp.Min(1),
			),
			mcp.WithNumber("max",
				mcp.Description("Maximum number of results per page."),
				mcp.Min(1),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			params := paginationParams(req)
			if v := req.GetString("siteName", ""); v != "" {
				params["siteName"] = v
			}
			items, pageInfo, err := client.GetList(ctx, "/account/sites", params)
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return listResult(items, pageInfo), nil
		},
	)

	// datto_list_devices
	srv.AddTool(
		mcp.NewTool("datto_list_devices",
			mcp.WithDescription("List all devices in the Datto RMM account with optional filters."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithString("hostname",
				mcp.Description("Filter devices by hostname."),
			),
			mcp.WithString("siteName",
				mcp.Description("Filter devices by site name."),
			),
			mcp.WithString("deviceType",
				mcp.Description("Filter devices by device type."),
			),
			mcp.WithString("operatingSystem",
				mcp.Description("Filter devices by operating system."),
			),
			mcp.WithString("filterId",
				mcp.Description("Filter devices by filter ID."),
			),
			mcp.WithNumber("page",
				mcp.Description("Page number for pagination."),
				mcp.Min(1),
			),
			mcp.WithNumber("max",
				mcp.Description("Maximum number of results per page."),
				mcp.Min(1),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			params := paginationParams(req)
			if v := req.GetString("hostname", ""); v != "" {
				params["hostname"] = v
			}
			if v := req.GetString("siteName", ""); v != "" {
				params["siteName"] = v
			}
			if v := req.GetString("deviceType", ""); v != "" {
				params["deviceType"] = v
			}
			if v := req.GetString("operatingSystem", ""); v != "" {
				params["operatingSystem"] = v
			}
			if v := req.GetString("filterId", ""); v != "" {
				params["filterId"] = v
			}
			items, pageInfo, err := client.GetList(ctx, "/account/devices", params)
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return listResult(items, pageInfo), nil
		},
	)

	// datto_list_users
	srv.AddTool(
		mcp.NewTool("datto_list_users",
			mcp.WithDescription("List all users in the Datto RMM account."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithNumber("page",
				mcp.Description("Page number for pagination."),
				mcp.Min(1),
			),
			mcp.WithNumber("max",
				mcp.Description("Maximum number of results per page."),
				mcp.Min(1),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			params := paginationParams(req)
			items, pageInfo, err := client.GetList(ctx, "/account/users", params)
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return listResult(items, pageInfo), nil
		},
	)

	// datto_list_account_variables
	srv.AddTool(
		mcp.NewTool("datto_list_account_variables",
			mcp.WithDescription("List all account-level variables in Datto RMM."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithNumber("page",
				mcp.Description("Page number for pagination."),
				mcp.Min(1),
			),
			mcp.WithNumber("max",
				mcp.Description("Maximum number of results per page."),
				mcp.Min(1),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			params := paginationParams(req)
			items, pageInfo, err := client.GetList(ctx, "/account/variables", params)
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return listResult(items, pageInfo), nil
		},
	)

	// datto_list_components
	srv.AddTool(
		mcp.NewTool("datto_list_components",
			mcp.WithDescription("List all components available in the Datto RMM account."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithNumber("page",
				mcp.Description("Page number for pagination."),
				mcp.Min(1),
			),
			mcp.WithNumber("max",
				mcp.Description("Maximum number of results per page."),
				mcp.Min(1),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			params := paginationParams(req)
			items, pageInfo, err := client.GetList(ctx, "/account/components", params)
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return listResult(items, pageInfo), nil
		},
	)

	// datto_list_open_alerts
	srv.AddTool(
		mcp.NewTool("datto_list_open_alerts",
			mcp.WithDescription("List all open alerts across the Datto RMM account."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithBoolean("muted",
				mcp.Description("Filter alerts by muted status."),
			),
			mcp.WithNumber("page",
				mcp.Description("Page number for pagination."),
				mcp.Min(1),
			),
			mcp.WithNumber("max",
				mcp.Description("Maximum number of results per page."),
				mcp.Min(1),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			params := paginationParams(req)
			args := req.GetArguments()
			if v, ok := args["muted"]; ok {
				if b, ok := v.(bool); ok {
					if b {
						params["muted"] = "true"
					} else {
						params["muted"] = "false"
					}
				}
			}
			items, pageInfo, err := client.GetList(ctx, "/account/alerts/open", params)
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return listResult(items, pageInfo), nil
		},
	)

	// datto_list_resolved_alerts
	srv.AddTool(
		mcp.NewTool("datto_list_resolved_alerts",
			mcp.WithDescription("List all resolved alerts across the Datto RMM account."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithBoolean("muted",
				mcp.Description("Filter alerts by muted status."),
			),
			mcp.WithNumber("page",
				mcp.Description("Page number for pagination."),
				mcp.Min(1),
			),
			mcp.WithNumber("max",
				mcp.Description("Maximum number of results per page."),
				mcp.Min(1),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			params := paginationParams(req)
			args := req.GetArguments()
			if v, ok := args["muted"]; ok {
				if b, ok := v.(bool); ok {
					if b {
						params["muted"] = "true"
					} else {
						params["muted"] = "false"
					}
				}
			}
			items, pageInfo, err := client.GetList(ctx, "/account/alerts/resolved", params)
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return listResult(items, pageInfo), nil
		},
	)

	_ = logger
}
