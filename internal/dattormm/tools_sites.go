package dattormm

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Logiphys/lgp-mcp/pkg/mcputil"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerSiteTools(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	// datto_get_site
	srv.AddTool(
		mcp.NewTool("datto_get_site",
			mcp.WithDescription("Get details for a specific Datto RMM site."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithString("siteUid",
				mcp.Description("The UID of the site."),
				mcp.Required(),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			siteUid := req.GetString("siteUid", "")
			result, err := client.Get(ctx, fmt.Sprintf("/site/%s", siteUid), nil)
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.JSONResult(result), nil
		},
	)

	// datto_list_site_devices
	srv.AddTool(
		mcp.NewTool("datto_list_site_devices",
			mcp.WithDescription("List devices belonging to a specific Datto RMM site."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithString("siteUid",
				mcp.Description("The UID of the site."),
				mcp.Required(),
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
			siteUid := req.GetString("siteUid", "")
			params := paginationParams(req)
			if v := req.GetString("filterId", ""); v != "" {
				params["filterId"] = v
			}
			items, pageInfo, err := client.GetList(ctx, fmt.Sprintf("/site/%s/devices", siteUid), params)
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return listResult(items, pageInfo), nil
		},
	)

	// datto_list_site_open_alerts
	srv.AddTool(
		mcp.NewTool("datto_list_site_open_alerts",
			mcp.WithDescription("List open alerts for a specific Datto RMM site."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithString("siteUid",
				mcp.Description("The UID of the site."),
				mcp.Required(),
			),
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
			siteUid := req.GetString("siteUid", "")
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
			items, pageInfo, err := client.GetList(ctx, fmt.Sprintf("/site/%s/alerts/open", siteUid), params)
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return listResult(items, pageInfo), nil
		},
	)

	// datto_list_site_resolved_alerts
	srv.AddTool(
		mcp.NewTool("datto_list_site_resolved_alerts",
			mcp.WithDescription("List resolved alerts for a specific Datto RMM site."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithString("siteUid",
				mcp.Description("The UID of the site."),
				mcp.Required(),
			),
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
			siteUid := req.GetString("siteUid", "")
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
			items, pageInfo, err := client.GetList(ctx, fmt.Sprintf("/site/%s/alerts/resolved", siteUid), params)
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return listResult(items, pageInfo), nil
		},
	)

	// datto_list_site_variables
	srv.AddTool(
		mcp.NewTool("datto_list_site_variables",
			mcp.WithDescription("List variables configured for a specific Datto RMM site."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithString("siteUid",
				mcp.Description("The UID of the site."),
				mcp.Required(),
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
			siteUid := req.GetString("siteUid", "")
			params := paginationParams(req)
			items, pageInfo, err := client.GetList(ctx, fmt.Sprintf("/site/%s/variables", siteUid), params)
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return listResult(items, pageInfo), nil
		},
	)

	// datto_get_site_settings
	srv.AddTool(
		mcp.NewTool("datto_get_site_settings",
			mcp.WithDescription("Get settings for a specific Datto RMM site."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithString("siteUid",
				mcp.Description("The UID of the site."),
				mcp.Required(),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			siteUid := req.GetString("siteUid", "")
			result, err := client.Get(ctx, fmt.Sprintf("/site/%s/settings", siteUid), nil)
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.JSONResult(result), nil
		},
	)

	// datto_list_site_filters
	srv.AddTool(
		mcp.NewTool("datto_list_site_filters",
			mcp.WithDescription("List filters configured for a specific Datto RMM site."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithString("siteUid",
				mcp.Description("The UID of the site."),
				mcp.Required(),
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
			siteUid := req.GetString("siteUid", "")
			params := paginationParams(req)
			items, pageInfo, err := client.GetList(ctx, fmt.Sprintf("/site/%s/filters", siteUid), params)
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return listResult(items, pageInfo), nil
		},
	)

	// datto_get_site_network_interfaces
	srv.AddTool(
		mcp.NewTool("datto_get_site_network_interfaces",
			mcp.WithDescription("Fetch shortened device records with network interface info (IPs, MACs, subnet) for a specific Datto RMM site."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithString("siteUid",
				mcp.Description("The UID of the site."),
				mcp.Required(),
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
			siteUid := req.GetString("siteUid", "")
			params := paginationParams(req)
			items, pageInfo, err := client.GetList(ctx, fmt.Sprintf("/site/%s/devices/network-interface", siteUid), params)
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return listResult(items, pageInfo), nil
		},
	)

	// datto_create_site
	srv.AddTool(
		mcp.NewTool("datto_create_site",
			mcp.WithDescription("Create a new site in Datto RMM."),
			mcp.WithString("name",
				mcp.Description("Name of the new site."),
				mcp.Required(),
			),
			mcp.WithString("description",
				mcp.Description("Description of the site."),
			),
			mcp.WithString("notes",
				mcp.Description("Notes for the site."),
			),
			mcp.WithBoolean("onDemand",
				mcp.Description("Whether the site is an on-demand site."),
			),
			mcp.WithBoolean("splashtopAutoInstall",
				mcp.Description("Whether to automatically install Splashtop on devices in this site."),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			body := map[string]any{
				"name": req.GetString("name", ""),
			}
			args := req.GetArguments()
			if v := req.GetString("description", ""); v != "" {
				body["description"] = v
			}
			if v := req.GetString("notes", ""); v != "" {
				body["notes"] = v
			}
			if v, ok := args["onDemand"]; ok {
				body["onDemand"] = v
			}
			if v, ok := args["splashtopAutoInstall"]; ok {
				body["splashtopAutoInstall"] = v
			}
			if err := client.Put(ctx, "/site", body); err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.TextResult("Site created successfully."), nil
		},
	)

	// datto_update_site
	srv.AddTool(
		mcp.NewTool("datto_update_site",
			mcp.WithDescription("Update an existing site in Datto RMM."),
			mcp.WithString("siteUid",
				mcp.Description("The UID of the site to update."),
				mcp.Required(),
			),
			mcp.WithString("name",
				mcp.Description("New name of the site."),
			),
			mcp.WithString("description",
				mcp.Description("New description of the site."),
			),
			mcp.WithString("notes",
				mcp.Description("New notes for the site."),
			),
			mcp.WithBoolean("onDemand",
				mcp.Description("Whether the site is an on-demand site."),
			),
			mcp.WithBoolean("splashtopAutoInstall",
				mcp.Description("Whether to automatically install Splashtop on devices in this site."),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			siteUid := req.GetString("siteUid", "")
			body := map[string]any{}
			args := req.GetArguments()
			if v := req.GetString("name", ""); v != "" {
				body["name"] = v
			}
			if v := req.GetString("description", ""); v != "" {
				body["description"] = v
			}
			if v := req.GetString("notes", ""); v != "" {
				body["notes"] = v
			}
			if v, ok := args["onDemand"]; ok {
				body["onDemand"] = v
			}
			if v, ok := args["splashtopAutoInstall"]; ok {
				body["splashtopAutoInstall"] = v
			}
			result, err := client.Post(ctx, fmt.Sprintf("/site/%s", siteUid), body)
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.JSONResult(result), nil
		},
	)

	_ = logger
}
