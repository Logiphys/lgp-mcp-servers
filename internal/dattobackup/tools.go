package dattobackup

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/Logiphys/lgp-mcp/pkg/mcputil"
)

// RegisterTools registers all Datto Backup MCP tools on the given server.
func RegisterTools(srv *server.MCPServer, client *Client, logger *slog.Logger, tier int) {
	// Tier 1 — Safe Read-Only
	registerTestConnection(srv, client, logger)
	registerListAppliances(srv, client, logger)
	registerListAssets(srv, client, logger)
	registerListBackups(srv, client, logger)
	registerListAlerts(srv, client, logger)
	registerGetAgentVersion(srv, client, logger)
	registerListSpanningDomains(srv, client, logger)
	registerListEntraDomains(srv, client, logger)

	// Tier 2 — Sensitive (customer/user data)
	if tier >= 2 {
		registerListCustomers(srv, client, logger)
		registerListEndpointAssets(srv, client, logger)
		registerListSpanningDomainUsers(srv, client, logger)
	}
}

// --- helpers ----------------------------------------------------------------

func addPaginationParams(params map[string]string, req mcp.CallToolRequest) {
	if v := req.GetInt("page", 0); v > 0 {
		params["page_number"] = strconv.Itoa(v)
	}
	if v := req.GetInt("pageSize", 0); v > 0 {
		params["page_size"] = strconv.Itoa(v)
	}
}

func buildListResult(items []any, pageInfo *PageInfo) *mcp.CallToolResult {
	result := map[string]any{"data": items}
	if pageInfo != nil {
		result["pagination"] = map[string]any{
			"total_records": pageInfo.TotalRecords,
			"total_pages":   pageInfo.TotalPages,
			"page":          pageInfo.Page,
			"page_size":     pageInfo.PageSize,
		}
	}
	return mcputil.JSONResult(result)
}

// --- tool registrations -----------------------------------------------------

func registerTestConnection(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("datto_backup_test_connection",
		mcp.WithDescription("Test connectivity to the Datto Backup (Unitrends) API. Returns success or an error message."),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if err := client.TestConnection(ctx); err != nil {
			logger.ErrorContext(ctx, "test connection failed", "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return mcputil.TextResult("Datto Backup API connection successful"), nil
	})
}

func registerListCustomers(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("datto_backup_list_customers",
		mcp.WithDescription("List Datto Backup customers (tenants/organizations)."),
		mcp.WithString("name", mcp.Description("Filter by customer name")),
		mcp.WithNumber("page", mcp.Description("Page number for pagination"), mcp.Min(1)),
		mcp.WithNumber("pageSize", mcp.Description("Number of results per page (default 50)"), mcp.Min(1)),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		params := make(map[string]string)
		addPaginationParams(params, req)
		if v := req.GetString("name", ""); v != "" {
			params["name"] = v
		}

		items, pageInfo, err := client.GetList(ctx, "/v1/customers", params)
		if err != nil {
			logger.ErrorContext(ctx, "list customers failed", "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return buildListResult(items, pageInfo), nil
	})
}

func registerListAppliances(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("datto_backup_list_appliances",
		mcp.WithDescription("List Datto Backup appliances (physical/virtual backup devices)."),
		mcp.WithString("customerId", mcp.Description("Filter by customer ID (UUID)")),
		mcp.WithString("name", mcp.Description("Filter by appliance name")),
		mcp.WithString("isOnline", mcp.Description("Filter by online status: true or false")),
		mcp.WithString("version", mcp.Description("Filter by appliance version")),
		mcp.WithString("helix_status", mcp.Description("Filter by Helix status")),
		mcp.WithString("order_by", mcp.Description("Field to order by")),
		mcp.WithString("order_direction", mcp.Description("Order direction: asc|desc")),
		mcp.WithNumber("page", mcp.Description("Page number for pagination"), mcp.Min(1)),
		mcp.WithNumber("pageSize", mcp.Description("Number of results per page (default 50)"), mcp.Min(1)),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		params := make(map[string]string)
		addPaginationParams(params, req)
		if v := req.GetString("customerId", ""); v != "" {
			params["customer_id"] = v
		}
		if v := req.GetString("name", ""); v != "" {
			params["name"] = v
		}
		if v := req.GetString("isOnline", ""); v != "" {
			params["is_online"] = v
		}
		if v := req.GetString("version", ""); v != "" {
			params["version"] = v
		}
		if v := req.GetString("helix_status", ""); v != "" {
			params["helix_status"] = v
		}
		if v := req.GetString("order_by", ""); v != "" {
			params["order_by"] = v
		}
		if v := req.GetString("order_direction", ""); v != "" {
			params["order_direction"] = v
		}

		items, pageInfo, err := client.GetList(ctx, "/v1/appliances", params)
		if err != nil {
			logger.ErrorContext(ctx, "list appliances failed", "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return buildListResult(items, pageInfo), nil
	})
}

func registerListAssets(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("datto_backup_list_assets",
		mcp.WithDescription("List protected assets (servers, VMs) on Datto Backup appliances. Use include parameters to embed last backup info and links."),
		mcp.WithString("customerId", mcp.Description("Filter by customer ID (UUID)")),
		mcp.WithString("assetTag", mcp.Description("Filter by appliance asset tag")),
		mcp.WithString("name", mcp.Description("Filter by asset name")),
		mcp.WithString("includeBackups", mcp.Description("Include last backup info: true or false (default false)")),
		mcp.WithString("type", mcp.Description("Filter by asset type")),
		mcp.WithString("ip", mcp.Description("Filter by IP address")),
		mcp.WithString("order_by", mcp.Description("Field to order by")),
		mcp.WithString("order_direction", mcp.Description("Order direction: asc|desc")),
		mcp.WithNumber("page", mcp.Description("Page number for pagination"), mcp.Min(1)),
		mcp.WithNumber("pageSize", mcp.Description("Number of results per page (default 50)"), mcp.Min(1)),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		params := make(map[string]string)
		addPaginationParams(params, req)
		if v := req.GetString("customerId", ""); v != "" {
			params["customer_id"] = v
		}
		if v := req.GetString("assetTag", ""); v != "" {
			params["asset_tag"] = v
		}
		if v := req.GetString("name", ""); v != "" {
			params["name"] = v
		}
		if v := req.GetString("includeBackups", ""); v == "true" {
			params["include[backups]"] = "last"
			params["include[links]"] = "all"
		}
		if v := req.GetString("type", ""); v != "" {
			params["type"] = v
		}
		if v := req.GetString("ip", ""); v != "" {
			params["ip"] = v
		}
		if v := req.GetString("order_by", ""); v != "" {
			params["order_by"] = v
		}
		if v := req.GetString("order_direction", ""); v != "" {
			params["order_direction"] = v
		}

		items, pageInfo, err := client.GetList(ctx, "/v1/assets", params)
		if err != nil {
			logger.ErrorContext(ctx, "list assets failed", "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return buildListResult(items, pageInfo), nil
	})
}

func registerListBackups(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("datto_backup_list_backups",
		mcp.WithDescription("List backup jobs/records. Filter by customer, status, date range, etc."),
		mcp.WithString("customerId", mcp.Description("Filter by customer ID (UUID)")),
		mcp.WithString("assetTag", mcp.Description("Filter by appliance asset tag")),
		mcp.WithString("status", mcp.Description("Filter by status: Successful, Warning, Failed, InProgress, Unknown")),
		mcp.WithString("startTimeFrom", mcp.Description("Filter backups started after this time (ISO 8601)")),
		mcp.WithString("startTimeTo", mcp.Description("Filter backups started before this time (ISO 8601)")),
		mcp.WithNumber("page", mcp.Description("Page number for pagination"), mcp.Min(1)),
		mcp.WithNumber("pageSize", mcp.Description("Number of results per page (default 50)"), mcp.Min(1)),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		params := make(map[string]string)
		addPaginationParams(params, req)
		if v := req.GetString("customerId", ""); v != "" {
			params["customer_id"] = v
		}
		if v := req.GetString("assetTag", ""); v != "" {
			params["asset_tag"] = v
		}
		if v := req.GetString("status", ""); v != "" {
			params["status"] = v
		}
		if v := req.GetString("startTimeFrom", ""); v != "" {
			params["start_time_from"] = v
		}
		if v := req.GetString("startTimeTo", ""); v != "" {
			params["start_time_to"] = v
		}

		items, pageInfo, err := client.GetList(ctx, "/v1/backups", params)
		if err != nil {
			logger.ErrorContext(ctx, "list backups failed", "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return buildListResult(items, pageInfo), nil
	})
}

func registerListAlerts(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("datto_backup_list_alerts",
		mcp.WithDescription("List BackupIQ alerts. The type parameter is required."),
		mcp.WithString("type", mcp.Description("Required: Alert type — alert, job, conditional, or helix"), mcp.Required()),
		mcp.WithString("severity", mcp.Description("Filter by severity: alarm, critical, or warning")),
		mcp.WithString("customerId", mcp.Description("Filter by customer ID (UUID)")),
		mcp.WithString("assetTag", mcp.Description("Filter by appliance asset tag")),
		mcp.WithString("isDismissed", mcp.Description("Filter by dismissed status: true or false")),
		mcp.WithString("is_muted", mcp.Description("Filter by muted status: true|false")),
		mcp.WithString("customer_name", mcp.Description("Filter by customer name")),
		mcp.WithString("appliance_name", mcp.Description("Filter by appliance name")),
		mcp.WithString("order_by", mcp.Description("Field to order by")),
		mcp.WithString("order_direction", mcp.Description("Order direction: asc|desc")),
		mcp.WithNumber("page", mcp.Description("Page number for pagination"), mcp.Min(1)),
		mcp.WithNumber("pageSize", mcp.Description("Number of results per page (default 50)"), mcp.Min(1)),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		params := make(map[string]string)
		addPaginationParams(params, req)
		if v := req.GetString("type", ""); v != "" {
			params["type"] = v
		}
		if v := req.GetString("severity", ""); v != "" {
			params["severity"] = v
		}
		if v := req.GetString("customerId", ""); v != "" {
			params["customer_id"] = v
		}
		if v := req.GetString("assetTag", ""); v != "" {
			params["asset_tag"] = v
		}
		if v := req.GetString("isDismissed", ""); v != "" {
			params["is_dismissed"] = v
		}
		if v := req.GetString("is_muted", ""); v != "" {
			params["is_muted"] = v
		}
		if v := req.GetString("customer_name", ""); v != "" {
			params["customer_name"] = v
		}
		if v := req.GetString("appliance_name", ""); v != "" {
			params["appliance_name"] = v
		}
		if v := req.GetString("order_by", ""); v != "" {
			params["order_by"] = v
		}
		if v := req.GetString("order_direction", ""); v != "" {
			params["order_direction"] = v
		}

		items, pageInfo, err := client.GetList(ctx, "/v1/backupiq/alerts", params)
		if err != nil {
			logger.ErrorContext(ctx, "list alerts failed", "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return buildListResult(items, pageInfo), nil
	})
}

func registerGetAgentVersion(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("datto_backup_get_agent_version",
		mcp.WithDescription("Get the latest Datto Backup agent version and download link."),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		result, err := client.Get(ctx, "/v1/agents/latest", nil)
		if err != nil {
			logger.ErrorContext(ctx, "get agent version failed", "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return mcputil.JSONResult(result), nil
	})
}

func registerListEndpointAssets(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("datto_backup_list_endpoint_assets",
		mcp.WithDescription("List endpoint backup assets (PCs, laptops)."),
		mcp.WithString("customerId", mcp.Description("Filter by customer ID")),
		mcp.WithString("name", mcp.Description("Filter by asset name")),
		mcp.WithNumber("page", mcp.Description("Page number for pagination"), mcp.Min(1)),
		mcp.WithNumber("pageSize", mcp.Description("Number of results per page (default 50)"), mcp.Min(1)),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		params := make(map[string]string)
		addPaginationParams(params, req)
		if v := req.GetString("customerId", ""); v != "" {
			params["customer_id"] = v
		}
		if v := req.GetString("name", ""); v != "" {
			params["name"] = v
		}

		items, pageInfo, err := client.GetList(ctx, "/api/epb/v1/assets", params)
		if err != nil {
			logger.ErrorContext(ctx, "list endpoint assets failed", "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return buildListResult(items, pageInfo), nil
	})
}

func registerListSpanningDomains(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("datto_backup_list_spanning_domains",
		mcp.WithDescription("List Spanning Backup domains (M365/Google Workspace tenants) with license and storage info."),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		items, pageInfo, err := client.GetList(ctx, "/v2/spanning/domains", nil)
		if err != nil {
			logger.ErrorContext(ctx, "list spanning domains failed", "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return buildListResult(items, pageInfo), nil
	})
}

func registerListSpanningDomainUsers(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("datto_backup_list_spanning_domain_users",
		mcp.WithDescription("List users within a Spanning Backup domain, including backup status per service."),
		mcp.WithString("domainId", mcp.Description("The Spanning domain ID"), mcp.Required()),
		mcp.WithNumber("page", mcp.Description("Page number for pagination"), mcp.Min(1)),
		mcp.WithNumber("pageSize", mcp.Description("Number of results per page (default 50)"), mcp.Min(1)),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		domainID := req.GetString("domainId", "")
		if domainID == "" {
			return mcputil.ErrorResult(fmt.Errorf("domainId is required")), nil
		}

		params := make(map[string]string)
		addPaginationParams(params, req)

		path := fmt.Sprintf("/v2/spanning/domains/%s/users", domainID)
		items, pageInfo, err := client.GetList(ctx, path, params)
		if err != nil {
			logger.ErrorContext(ctx, "list spanning domain users failed", "domainId", domainID, "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return buildListResult(items, pageInfo), nil
	})
}

func registerListEntraDomains(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("datto_backup_list_entra_domains",
		mcp.WithDescription("List Microsoft Entra ID domains with backup and license info."),
		mcp.WithNumber("page", mcp.Description("Page number for pagination"), mcp.Min(1)),
		mcp.WithNumber("pageSize", mcp.Description("Number of results per page (default 50)"), mcp.Min(1)),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		params := make(map[string]string)
		addPaginationParams(params, req)

		items, pageInfo, err := client.GetList(ctx, "/v1/entra/domains", params)
		if err != nil {
			logger.ErrorContext(ctx, "list entra domains failed", "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return buildListResult(items, pageInfo), nil
	})
}
