package dattobcdr

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/Logiphys/lgp-mcp/pkg/mcputil"
)

// RegisterTools registers all Datto BCDR MCP tools on the given server.
func RegisterTools(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	registerTestConnection(srv, client, logger)
	registerListDevices(srv, client, logger)
	registerGetDevice(srv, client, logger)
	registerListDeviceAssets(srv, client, logger)
	registerListDeviceAgents(srv, client, logger)
	registerListDeviceShares(srv, client, logger)
	registerListDeviceAlerts(srv, client, logger)
	registerListDeviceVMRestores(srv, client, logger)
	registerListAgents(srv, client, logger)
	registerGetActivityLog(srv, client, logger)
}

// --- helpers ----------------------------------------------------------------

func addPaginationParams(params map[string]string, req mcp.CallToolRequest) {
	if v := req.GetInt("page", 0); v > 0 {
		params["page"] = strconv.Itoa(v)
	}
	if v := req.GetInt("perPage", 0); v > 0 {
		params["perPage"] = strconv.Itoa(v)
	}
}

// addUnderscorePaginationParams adds _page/_perPage for endpoints that use underscore-prefixed params.
func addUnderscorePaginationParams(params map[string]string, req mcp.CallToolRequest) {
	if v := req.GetInt("page", 0); v > 0 {
		params["_page"] = strconv.Itoa(v)
	}
	if v := req.GetInt("perPage", 0); v > 0 {
		params["_perPage"] = strconv.Itoa(v)
	}
}

func buildListResult(items []any, pageInfo *PageInfo) *mcp.CallToolResult {
	result := map[string]any{"data": items}
	if pageInfo != nil {
		result["pagination"] = map[string]any{
			"total_count": pageInfo.TotalCount,
			"page":        pageInfo.Page,
			"per_page":    pageInfo.PerPage,
		}
	}
	return mcputil.JSONResult(result)
}

// --- tool registrations -----------------------------------------------------

func registerTestConnection(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("datto_bcdr_test_connection",
		mcp.WithDescription("Test connectivity to the Datto BCDR API. Returns success or an error message."),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if err := client.TestConnection(ctx); err != nil {
			logger.ErrorContext(ctx, "test connection failed", "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return mcputil.TextResult("Datto BCDR API connection successful"), nil
	})
}

func registerListDevices(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("datto_bcdr_list_devices",
		mcp.WithDescription("List all Datto BCDR devices (SIRIS, ALTO, NAS). Supports pagination."),
		mcp.WithNumber("page", mcp.Description("Page number for pagination"), mcp.Min(1)),
		mcp.WithNumber("perPage", mcp.Description("Number of results per page (default 100)"), mcp.Min(1), mcp.Max(1000)),
		mcp.WithBoolean("showHiddenDevices", mcp.Description("Include hidden devices in results")),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		params := make(map[string]string)
		addPaginationParams(params, req)
		if v := req.GetBool("showHiddenDevices", false); v {
			params["showHiddenDevices"] = "true"
		}

		items, pageInfo, err := client.GetList(ctx, "/bcdr/device", params)
		if err != nil {
			logger.ErrorContext(ctx, "list devices failed", "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return buildListResult(items, pageInfo), nil
	})
}

func registerGetDevice(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("datto_bcdr_get_device",
		mcp.WithDescription("Get details of a specific Datto BCDR device by serial number."),
		mcp.WithString("serialNumber", mcp.Description("The device serial number"), mcp.Required()),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		sn := req.GetString("serialNumber", "")
		if sn == "" {
			return mcputil.ErrorResult(fmt.Errorf("serialNumber is required")), nil
		}

		path := fmt.Sprintf("/bcdr/device/%s", sn)
		result, err := client.Get(ctx, path, nil)
		if err != nil {
			logger.ErrorContext(ctx, "get device failed", "serialNumber", sn, "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return mcputil.JSONResult(result), nil
	})
}

func registerListDeviceAssets(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("datto_bcdr_list_device_assets",
		mcp.WithDescription("List all assets (agents and shares) for a specific Datto BCDR device."),
		mcp.WithString("serialNumber", mcp.Description("The device serial number"), mcp.Required()),
		mcp.WithNumber("page", mcp.Description("Page number for pagination"), mcp.Min(1)),
		mcp.WithNumber("perPage", mcp.Description("Number of results per page (default 100)"), mcp.Min(1), mcp.Max(1000)),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		sn := req.GetString("serialNumber", "")
		if sn == "" {
			return mcputil.ErrorResult(fmt.Errorf("serialNumber is required")), nil
		}

		params := make(map[string]string)
		addPaginationParams(params, req)

		path := fmt.Sprintf("/bcdr/device/%s/asset", sn)
		items, pageInfo, err := client.GetList(ctx, path, params)
		if err != nil {
			logger.ErrorContext(ctx, "list device assets failed", "serialNumber", sn, "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return buildListResult(items, pageInfo), nil
	})
}

func registerListDeviceAgents(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("datto_bcdr_list_device_agents",
		mcp.WithDescription("List agents for a specific Datto BCDR device."),
		mcp.WithString("serialNumber", mcp.Description("The device serial number"), mcp.Required()),
		mcp.WithNumber("page", mcp.Description("Page number for pagination"), mcp.Min(1)),
		mcp.WithNumber("perPage", mcp.Description("Number of results per page (default 100)"), mcp.Min(1), mcp.Max(1000)),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		sn := req.GetString("serialNumber", "")
		if sn == "" {
			return mcputil.ErrorResult(fmt.Errorf("serialNumber is required")), nil
		}

		params := make(map[string]string)
		addPaginationParams(params, req)

		path := fmt.Sprintf("/bcdr/device/%s/asset/agent", sn)
		items, pageInfo, err := client.GetList(ctx, path, params)
		if err != nil {
			logger.ErrorContext(ctx, "list device agents failed", "serialNumber", sn, "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return buildListResult(items, pageInfo), nil
	})
}

func registerListDeviceShares(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("datto_bcdr_list_device_shares",
		mcp.WithDescription("List shares (NAS backup targets) for a specific Datto BCDR device."),
		mcp.WithString("serialNumber", mcp.Description("The device serial number"), mcp.Required()),
		mcp.WithNumber("page", mcp.Description("Page number for pagination"), mcp.Min(1)),
		mcp.WithNumber("perPage", mcp.Description("Number of results per page (default 100)"), mcp.Min(1), mcp.Max(1000)),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		sn := req.GetString("serialNumber", "")
		if sn == "" {
			return mcputil.ErrorResult(fmt.Errorf("serialNumber is required")), nil
		}

		params := make(map[string]string)
		addPaginationParams(params, req)

		path := fmt.Sprintf("/bcdr/device/%s/asset/share", sn)
		items, pageInfo, err := client.GetList(ctx, path, params)
		if err != nil {
			logger.ErrorContext(ctx, "list device shares failed", "serialNumber", sn, "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return buildListResult(items, pageInfo), nil
	})
}

func registerListDeviceAlerts(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("datto_bcdr_list_device_alerts",
		mcp.WithDescription("List alerts for a specific Datto BCDR device."),
		mcp.WithString("serialNumber", mcp.Description("The device serial number"), mcp.Required()),
		mcp.WithNumber("page", mcp.Description("Page number for pagination"), mcp.Min(1)),
		mcp.WithNumber("perPage", mcp.Description("Number of results per page (default 100)"), mcp.Min(1), mcp.Max(1000)),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		sn := req.GetString("serialNumber", "")
		if sn == "" {
			return mcputil.ErrorResult(fmt.Errorf("serialNumber is required")), nil
		}

		params := make(map[string]string)
		addPaginationParams(params, req)

		path := fmt.Sprintf("/bcdr/device/%s/alert", sn)
		items, pageInfo, err := client.GetList(ctx, path, params)
		if err != nil {
			logger.ErrorContext(ctx, "list device alerts failed", "serialNumber", sn, "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return buildListResult(items, pageInfo), nil
	})
}

func registerListDeviceVMRestores(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("datto_bcdr_list_device_vm_restores",
		mcp.WithDescription("List VM restores for a specific Datto BCDR device."),
		mcp.WithString("serialNumber", mcp.Description("The device serial number"), mcp.Required()),
		mcp.WithNumber("page", mcp.Description("Page number for pagination"), mcp.Min(1)),
		mcp.WithNumber("perPage", mcp.Description("Number of results per page (default 100)"), mcp.Min(1), mcp.Max(1000)),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		sn := req.GetString("serialNumber", "")
		if sn == "" {
			return mcputil.ErrorResult(fmt.Errorf("serialNumber is required")), nil
		}

		params := make(map[string]string)
		addPaginationParams(params, req)

		path := fmt.Sprintf("/bcdr/device/%s/vm-restores", sn)
		items, pageInfo, err := client.GetList(ctx, path, params)
		if err != nil {
			logger.ErrorContext(ctx, "list device vm restores failed", "serialNumber", sn, "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return buildListResult(items, pageInfo), nil
	})
}

func registerListAgents(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("datto_bcdr_list_agents",
		mcp.WithDescription("List all Datto BCDR agents (Endpoint Backup for PCs)."),
		mcp.WithNumber("page", mcp.Description("Page number for pagination"), mcp.Min(1)),
		mcp.WithNumber("perPage", mcp.Description("Number of results per page (default 100)"), mcp.Min(1), mcp.Max(1000)),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		params := make(map[string]string)
		addUnderscorePaginationParams(params, req)

		items, pageInfo, err := client.GetList(ctx, "/bcdr/agent", params)
		if err != nil {
			logger.ErrorContext(ctx, "list agents failed", "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return buildListResult(items, pageInfo), nil
	})
}

func registerGetActivityLog(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("datto_bcdr_get_activity_log",
		mcp.WithDescription("Get the Datto BCDR activity log. Supports filtering by date and user."),
		mcp.WithNumber("page", mcp.Description("Page number for pagination"), mcp.Min(1)),
		mcp.WithNumber("perPage", mcp.Description("Number of results per page (default 100)"), mcp.Min(1), mcp.Max(1000)),
		mcp.WithString("since", mcp.Description("Filter entries since this date (ISO 8601 format)")),
		mcp.WithString("user", mcp.Description("Filter by user email")),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		params := make(map[string]string)
		addUnderscorePaginationParams(params, req)
		if v := req.GetString("since", ""); v != "" {
			params["since"] = v
		}
		if v := req.GetString("user", ""); v != "" {
			params["user"] = v
		}

		items, pageInfo, err := client.GetList(ctx, "/report/activity-log", params)
		if err != nil {
			logger.ErrorContext(ctx, "get activity log failed", "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return buildListResult(items, pageInfo), nil
	})
}
