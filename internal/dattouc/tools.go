package dattouc

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/Logiphys/lgp-mcp/pkg/mcputil"
)

// RegisterTools registers all Datto Unified Continuity MCP tools on the given server.
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
	registerListSaaSDomains(srv, client, logger)
	registerGetSaaSSeats(srv, client, logger)
	registerGetSaaSApplications(srv, client, logger)
	registerGetDeviceVolumeAssets(srv, client, logger)
	// Direct-to-Cloud (DTC) tools
	registerListDTCAssets(srv, client, logger)
	registerListDTCRMMTemplates(srv, client, logger)
	registerGetDTCStoragePool(srv, client, logger)
	registerListDTCClientAssets(srv, client, logger)
	registerGetDTCAsset(srv, client, logger)
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
	tool := mcp.NewTool("datto_uc_test_connection",
		mcp.WithDescription("Test connectivity to the Datto Unified Continuity API. Returns success or an error message."),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if err := client.TestConnection(ctx); err != nil {
			logger.ErrorContext(ctx, "test connection failed", "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return mcputil.TextResult("Datto Unified Continuity API connection successful"), nil
	})
}

func registerListDevices(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("datto_uc_list_devices",
		mcp.WithDescription("List all BCDR devices (SIRIS, ALTO, NAS). Supports pagination."),
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
	tool := mcp.NewTool("datto_uc_get_device",
		mcp.WithDescription("Get details of a specific BCDR device by serial number."),
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
	tool := mcp.NewTool("datto_uc_list_device_assets",
		mcp.WithDescription("List all assets (agents and shares) for a specific BCDR device."),
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
	tool := mcp.NewTool("datto_uc_list_device_agents",
		mcp.WithDescription("List agents for a specific BCDR device."),
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
	tool := mcp.NewTool("datto_uc_list_device_shares",
		mcp.WithDescription("List shares (NAS backup targets) for a specific BCDR device."),
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
	tool := mcp.NewTool("datto_uc_list_device_alerts",
		mcp.WithDescription("List alerts for a specific BCDR device."),
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
	tool := mcp.NewTool("datto_uc_list_device_vm_restores",
		mcp.WithDescription("List VM restores for a specific BCDR device."),
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
	tool := mcp.NewTool("datto_uc_list_agents",
		mcp.WithDescription("List all BCDR agents (Endpoint Backup for PCs)."),
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
	tool := mcp.NewTool("datto_uc_get_activity_log",
		mcp.WithDescription("Get the Datto Unified Continuity activity log. Supports filtering by date and user."),
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

func registerListSaaSDomains(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("datto_uc_list_saas_domains",
		mcp.WithDescription("List all SaaS Protection domains (M365/Google Workspace tenants)."),
		mcp.WithNumber("page", mcp.Description("Page number for pagination"), mcp.Min(1)),
		mcp.WithNumber("perPage", mcp.Description("Number of results per page (default 100)"), mcp.Min(1), mcp.Max(1000)),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		params := make(map[string]string)
		addUnderscorePaginationParams(params, req)

		items, pageInfo, err := client.GetList(ctx, "/saas/domains", params)
		if err != nil {
			logger.ErrorContext(ctx, "list saas domains failed", "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return buildListResult(items, pageInfo), nil
	})
}

func registerGetSaaSSeats(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("datto_uc_get_saas_seats",
		mcp.WithDescription("List protected seats/users for a SaaS Protection customer."),
		mcp.WithString("saasCustomerId", mcp.Description("The SaaS customer ID"), mcp.Required()),
		mcp.WithNumber("page", mcp.Description("Page number for pagination"), mcp.Min(1)),
		mcp.WithNumber("perPage", mcp.Description("Number of results per page (default 100)"), mcp.Min(1), mcp.Max(1000)),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		saasID := req.GetString("saasCustomerId", "")
		if saasID == "" {
			return mcputil.ErrorResult(fmt.Errorf("saasCustomerId is required")), nil
		}

		params := make(map[string]string)
		addUnderscorePaginationParams(params, req)

		path := fmt.Sprintf("/saas/%s/seats", saasID)
		items, pageInfo, err := client.GetList(ctx, path, params)
		if err != nil {
			logger.ErrorContext(ctx, "get saas seats failed", "saasCustomerId", saasID, "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return buildListResult(items, pageInfo), nil
	})
}

func registerGetSaaSApplications(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("datto_uc_get_saas_applications",
		mcp.WithDescription("List protected applications for a SaaS Protection customer."),
		mcp.WithString("saasCustomerId", mcp.Description("The SaaS customer ID"), mcp.Required()),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		saasID := req.GetString("saasCustomerId", "")
		if saasID == "" {
			return mcputil.ErrorResult(fmt.Errorf("saasCustomerId is required")), nil
		}

		path := fmt.Sprintf("/saas/%s/applications", saasID)
		items, pageInfo, err := client.GetList(ctx, path, nil)
		if err != nil {
			logger.ErrorContext(ctx, "get saas applications failed", "saasCustomerId", saasID, "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return buildListResult(items, pageInfo), nil
	})
}

func registerGetDeviceVolumeAssets(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("datto_uc_get_device_volume_assets",
		mcp.WithDescription("Get assets for a specific volume on a BCDR device."),
		mcp.WithString("serialNumber", mcp.Description("The device serial number"), mcp.Required()),
		mcp.WithString("volumeName", mcp.Description("The volume name"), mcp.Required()),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		sn := req.GetString("serialNumber", "")
		if sn == "" {
			return mcputil.ErrorResult(fmt.Errorf("serialNumber is required")), nil
		}
		vol := req.GetString("volumeName", "")
		if vol == "" {
			return mcputil.ErrorResult(fmt.Errorf("volumeName is required")), nil
		}

		path := fmt.Sprintf("/bcdr/device/%s/asset/%s", sn, vol)
		result, err := client.Get(ctx, path, nil)
		if err != nil {
			logger.ErrorContext(ctx, "get device volume assets failed", "serialNumber", sn, "volumeName", vol, "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return mcputil.JSONResult(result), nil
	})
}

// --- Direct-to-Cloud (DTC) tools --------------------------------------------

func registerListDTCAssets(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("datto_uc_list_dtc_assets",
		mcp.WithDescription("List all Direct-to-Cloud assets. Supports pagination."),
		mcp.WithNumber("page", mcp.Description("Page number for pagination"), mcp.Min(1)),
		mcp.WithNumber("perPage", mcp.Description("Number of results per page (default 100)"), mcp.Min(1), mcp.Max(1000)),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		params := make(map[string]string)
		addUnderscorePaginationParams(params, req)

		items, pageInfo, err := client.GetList(ctx, "/dtc/assets", params)
		if err != nil {
			logger.ErrorContext(ctx, "list dtc assets failed", "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return buildListResult(items, pageInfo), nil
	})
}

func registerListDTCRMMTemplates(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("datto_uc_list_dtc_rmm_templates",
		mcp.WithDescription("List RMM templates for Direct-to-Cloud."),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		items, pageInfo, err := client.GetList(ctx, "/dtc/rmm-templates", nil)
		if err != nil {
			logger.ErrorContext(ctx, "list dtc rmm templates failed", "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return buildListResult(items, pageInfo), nil
	})
}

func registerGetDTCStoragePool(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("datto_uc_get_dtc_storage_pool",
		mcp.WithDescription("Get Direct-to-Cloud storage pool usage."),
		mcp.WithString("poolName", mcp.Description("Optional storage pool name to filter by")),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		params := make(map[string]string)
		if v := req.GetString("poolName", ""); v != "" {
			params["poolName"] = v
		}

		result, err := client.Get(ctx, "/dtc/storage-pool", params)
		if err != nil {
			logger.ErrorContext(ctx, "get dtc storage pool failed", "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return mcputil.JSONResult(result), nil
	})
}

func registerListDTCClientAssets(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("datto_uc_list_dtc_client_assets",
		mcp.WithDescription("List Direct-to-Cloud assets for a specific client."),
		mcp.WithString("clientId", mcp.Description("The client ID"), mcp.Required()),
		mcp.WithNumber("page", mcp.Description("Page number for pagination"), mcp.Min(1)),
		mcp.WithNumber("perPage", mcp.Description("Number of results per page (default 100)"), mcp.Min(1), mcp.Max(1000)),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID := req.GetString("clientId", "")
		if clientID == "" {
			return mcputil.ErrorResult(fmt.Errorf("clientId is required")), nil
		}

		params := make(map[string]string)
		addUnderscorePaginationParams(params, req)

		path := fmt.Sprintf("/dtc/%s/assets", clientID)
		items, pageInfo, err := client.GetList(ctx, path, params)
		if err != nil {
			logger.ErrorContext(ctx, "list dtc client assets failed", "clientId", clientID, "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return buildListResult(items, pageInfo), nil
	})
}

func registerGetDTCAsset(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("datto_uc_get_dtc_asset",
		mcp.WithDescription("Get details of a specific Direct-to-Cloud asset."),
		mcp.WithString("clientId", mcp.Description("The client ID"), mcp.Required()),
		mcp.WithString("assetUuid", mcp.Description("The asset UUID"), mcp.Required()),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID := req.GetString("clientId", "")
		if clientID == "" {
			return mcputil.ErrorResult(fmt.Errorf("clientId is required")), nil
		}
		assetUUID := req.GetString("assetUuid", "")
		if assetUUID == "" {
			return mcputil.ErrorResult(fmt.Errorf("assetUuid is required")), nil
		}

		path := fmt.Sprintf("/dtc/%s/assets/%s", clientID, assetUUID)
		result, err := client.Get(ctx, path, nil)
		if err != nil {
			logger.ErrorContext(ctx, "get dtc asset failed", "clientId", clientID, "assetUuid", assetUUID, "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return mcputil.JSONResult(result), nil
	})
}
