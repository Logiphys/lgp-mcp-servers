package dattoedr

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/Logiphys/lgp-mcp-servers/pkg/mcputil"
)

// RegisterTools registers all Datto EDR MCP tools on the given server.
func RegisterTools(srv *server.MCPServer, client *Client, logger *slog.Logger, tier int) {
	// Tier 1 — Safe Read-Only
	registerTestConnection(srv, client, logger)
	registerGetDashboard(srv, client, logger)
	registerListAgents(srv, client, logger)
	registerGetAgent(srv, client, logger)
	registerGetAgentCount(srv, client, logger)
	registerListAlerts(srv, client, logger)
	registerGetAlert(srv, client, logger)
	registerGetAlertCount(srv, client, logger)
	registerListOrganizations(srv, client, logger)
	registerListLocations(srv, client, logger)
	registerListDeviceGroups(srv, client, logger)
	registerListPolicies(srv, client, logger)
	registerListRules(srv, client, logger)
	registerListSuppressionRules(srv, client, logger)
	registerListExtensions(srv, client, logger)

	// Tier 2 — Sensitive
	if tier >= 2 {
		registerListAlertsArchive(srv, client, logger)
		registerListQuarantinedFiles(srv, client, logger)
	}

	// Tier 3 — Actions
	if tier >= 3 {
		registerScanAgent(srv, client, logger)
		registerIsolateHost(srv, client, logger)
		registerRestoreHost(srv, client, logger)
	}
}

// --- helpers ----------------------------------------------------------------

func addLoopBackPagination(params map[string]string, req mcp.CallToolRequest) {
	limit := req.GetInt("limit", 100)
	if limit > 1000 {
		limit = 1000
	}
	AddLimitFilter(params, limit)
	AddSkipFilter(params, req.GetInt("skip", 0))
	AddOrderFilter(params, req.GetString("order", ""))
}

func buildListResult(items []any) *mcp.CallToolResult {
	result := map[string]any{
		"data":  items,
		"count": len(items),
	}
	return mcputil.JSONResult(result)
}

// --- read-only tool registrations -------------------------------------------

func registerTestConnection(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("datto_edr_test_connection",
		mcp.WithDescription("Test connectivity to the Datto EDR (Infocyte) API. Returns success or an error message."),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if err := client.TestConnection(ctx); err != nil {
			logger.ErrorContext(ctx, "test connection failed", "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return mcputil.TextResult("Datto EDR API connection successful"), nil
	})
}

func registerListAgents(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("datto_edr_list_agents",
		mcp.WithDescription("List Datto EDR agents with optional filters. Supports LoopBack pagination."),
		mcp.WithString("hostname", mcp.Description("Filter by hostname")),
		mcp.WithString("customerId", mcp.Description("Filter by customer/organization ID")),
		mcp.WithString("connectivity", mcp.Description("Filter by connectivity status")),
		mcp.WithNumber("limit", mcp.Description("Maximum number of results (default 100, max 1000)"), mcp.Min(1), mcp.Max(1000)),
		mcp.WithNumber("skip", mcp.Description("Number of results to skip for pagination"), mcp.Min(0)),
		mcp.WithString("order", mcp.Description("Sort order (e.g. 'createdAt DESC')")),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		params := make(map[string]string)
		addLoopBackPagination(params, req)
		AddWhereFilter(params, "hostname", req.GetString("hostname", ""))
		AddWhereFilter(params, "customerId", req.GetString("customerId", ""))
		AddWhereFilter(params, "connectivity", req.GetString("connectivity", ""))

		items, err := client.GetList(ctx, "/api/Agents", params)
		if err != nil {
			logger.ErrorContext(ctx, "list agents failed", "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return buildListResult(items), nil
	})
}

func registerGetAgent(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("datto_edr_get_agent",
		mcp.WithDescription("Get details of a specific Datto EDR agent by ID."),
		mcp.WithString("id", mcp.Description("The agent ID"), mcp.Required()),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id := req.GetString("id", "")
		if id == "" {
			return mcputil.ErrorResult(fmt.Errorf("id is required")), nil
		}

		path := fmt.Sprintf("/api/Agents/%s", id)
		result, err := client.Get(ctx, path, nil)
		if err != nil {
			logger.ErrorContext(ctx, "get agent failed", "id", id, "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return mcputil.JSONResult(result), nil
	})
}

func registerListAlerts(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("datto_edr_list_alerts",
		mcp.WithDescription("List Datto EDR alerts with optional filters. Supports LoopBack pagination."),
		mcp.WithString("severity", mcp.Description("Filter by severity level")),
		mcp.WithString("agentId", mcp.Description("Filter by agent ID")),
		mcp.WithNumber("limit", mcp.Description("Maximum number of results (default 100, max 1000)"), mcp.Min(1), mcp.Max(1000)),
		mcp.WithNumber("skip", mcp.Description("Number of results to skip for pagination"), mcp.Min(0)),
		mcp.WithString("order", mcp.Description("Sort order (e.g. 'createdAt DESC')")),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		params := make(map[string]string)
		addLoopBackPagination(params, req)
		AddWhereFilter(params, "severity", req.GetString("severity", ""))
		AddWhereFilter(params, "agentId", req.GetString("agentId", ""))

		items, err := client.GetList(ctx, "/api/Alerts", params)
		if err != nil {
			logger.ErrorContext(ctx, "list alerts failed", "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return buildListResult(items), nil
	})
}

func registerGetAlert(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("datto_edr_get_alert",
		mcp.WithDescription("Get details of a specific Datto EDR alert by ID."),
		mcp.WithString("id", mcp.Description("The alert ID"), mcp.Required()),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id := req.GetString("id", "")
		if id == "" {
			return mcputil.ErrorResult(fmt.Errorf("id is required")), nil
		}

		path := fmt.Sprintf("/api/AlertDetails/%s", id)
		result, err := client.Get(ctx, path, nil)
		if err != nil {
			logger.ErrorContext(ctx, "get alert failed", "id", id, "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return mcputil.JSONResult(result), nil
	})
}

func registerListAlertsArchive(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("datto_edr_list_alerts_archive",
		mcp.WithDescription("List archived Datto EDR alerts with optional filters. Supports LoopBack pagination."),
		mcp.WithString("severity", mcp.Description("Filter by severity level")),
		mcp.WithString("agentId", mcp.Description("Filter by agent ID")),
		mcp.WithNumber("limit", mcp.Description("Maximum number of results (default 100, max 1000)"), mcp.Min(1), mcp.Max(1000)),
		mcp.WithNumber("skip", mcp.Description("Number of results to skip for pagination"), mcp.Min(0)),
		mcp.WithString("order", mcp.Description("Sort order (e.g. 'createdAt DESC')")),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		params := make(map[string]string)
		addLoopBackPagination(params, req)
		AddWhereFilter(params, "severity", req.GetString("severity", ""))
		AddWhereFilter(params, "agentId", req.GetString("agentId", ""))

		items, err := client.GetList(ctx, "/api/AlertsArchive", params)
		if err != nil {
			logger.ErrorContext(ctx, "list alerts archive failed", "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return buildListResult(items), nil
	})
}

func registerListOrganizations(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("datto_edr_list_organizations",
		mcp.WithDescription("List Datto EDR organizations. Supports LoopBack pagination."),
		mcp.WithNumber("limit", mcp.Description("Maximum number of results (default 100, max 1000)"), mcp.Min(1), mcp.Max(1000)),
		mcp.WithNumber("skip", mcp.Description("Number of results to skip for pagination"), mcp.Min(0)),
		mcp.WithString("order", mcp.Description("Sort order (e.g. 'name ASC')")),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		params := make(map[string]string)
		addLoopBackPagination(params, req)

		items, err := client.GetList(ctx, "/api/Organizations", params)
		if err != nil {
			logger.ErrorContext(ctx, "list organizations failed", "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return buildListResult(items), nil
	})
}

func registerListLocations(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("datto_edr_list_locations",
		mcp.WithDescription("List Datto EDR locations. Supports LoopBack pagination."),
		mcp.WithNumber("limit", mcp.Description("Maximum number of results (default 100, max 1000)"), mcp.Min(1), mcp.Max(1000)),
		mcp.WithNumber("skip", mcp.Description("Number of results to skip for pagination"), mcp.Min(0)),
		mcp.WithString("order", mcp.Description("Sort order (e.g. 'name ASC')")),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		params := make(map[string]string)
		addLoopBackPagination(params, req)

		items, err := client.GetList(ctx, "/api/Locations", params)
		if err != nil {
			logger.ErrorContext(ctx, "list locations failed", "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return buildListResult(items), nil
	})
}

func registerListDeviceGroups(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("datto_edr_list_device_groups",
		mcp.WithDescription("List Datto EDR device groups. Supports LoopBack pagination."),
		mcp.WithNumber("limit", mcp.Description("Maximum number of results (default 100, max 1000)"), mcp.Min(1), mcp.Max(1000)),
		mcp.WithNumber("skip", mcp.Description("Number of results to skip for pagination"), mcp.Min(0)),
		mcp.WithString("order", mcp.Description("Sort order (e.g. 'name ASC')")),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		params := make(map[string]string)
		addLoopBackPagination(params, req)

		items, err := client.GetList(ctx, "/api/DeviceGroups", params)
		if err != nil {
			logger.ErrorContext(ctx, "list device groups failed", "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return buildListResult(items), nil
	})
}

func registerListPolicies(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("datto_edr_list_policies",
		mcp.WithDescription("List Datto EDR policies. Supports LoopBack pagination."),
		mcp.WithNumber("limit", mcp.Description("Maximum number of results (default 100, max 1000)"), mcp.Min(1), mcp.Max(1000)),
		mcp.WithNumber("skip", mcp.Description("Number of results to skip for pagination"), mcp.Min(0)),
		mcp.WithString("order", mcp.Description("Sort order (e.g. 'name ASC')")),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		params := make(map[string]string)
		addLoopBackPagination(params, req)

		items, err := client.GetList(ctx, "/api/Policies", params)
		if err != nil {
			logger.ErrorContext(ctx, "list policies failed", "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return buildListResult(items), nil
	})
}

func registerListRules(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("datto_edr_list_rules",
		mcp.WithDescription("List Datto EDR detection rules. Supports LoopBack pagination."),
		mcp.WithNumber("limit", mcp.Description("Maximum number of results (default 100, max 1000)"), mcp.Min(1), mcp.Max(1000)),
		mcp.WithNumber("skip", mcp.Description("Number of results to skip for pagination"), mcp.Min(0)),
		mcp.WithString("order", mcp.Description("Sort order (e.g. 'name ASC')")),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		limit := req.GetInt("limit", 100)
		if limit > 1000 {
			limit = 1000
		}
		params := BuildJSONFilter(limit, req.GetInt("skip", 0), req.GetString("order", ""))

		items, err := client.GetList(ctx, "/api/Rules", params)
		if err != nil {
			logger.ErrorContext(ctx, "list rules failed", "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return buildListResult(items), nil
	})
}

func registerListSuppressionRules(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("datto_edr_list_suppression_rules",
		mcp.WithDescription("List Datto EDR alert suppression rules. Supports LoopBack pagination."),
		mcp.WithNumber("limit", mcp.Description("Maximum number of results (default 100, max 1000)"), mcp.Min(1), mcp.Max(1000)),
		mcp.WithNumber("skip", mcp.Description("Number of results to skip for pagination"), mcp.Min(0)),
		mcp.WithString("order", mcp.Description("Sort order (e.g. 'name ASC')")),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		limit := req.GetInt("limit", 100)
		if limit > 1000 {
			limit = 1000
		}
		params := BuildJSONFilter(limit, req.GetInt("skip", 0), req.GetString("order", ""))

		items, err := client.GetList(ctx, "/api/SuppressionRules", params)
		if err != nil {
			logger.ErrorContext(ctx, "list suppression rules failed", "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return buildListResult(items), nil
	})
}

func registerListExtensions(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("datto_edr_list_extensions",
		mcp.WithDescription("List Datto EDR extensions (response actions, collection modules). Supports LoopBack pagination."),
		mcp.WithNumber("limit", mcp.Description("Maximum number of results (default 100, max 1000)"), mcp.Min(1), mcp.Max(1000)),
		mcp.WithNumber("skip", mcp.Description("Number of results to skip for pagination"), mcp.Min(0)),
		mcp.WithString("order", mcp.Description("Sort order (e.g. 'name ASC')")),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		limit := req.GetInt("limit", 100)
		if limit > 1000 {
			limit = 1000
		}
		params := BuildJSONFilter(limit, req.GetInt("skip", 0), req.GetString("order", ""))

		items, err := client.GetList(ctx, "/api/Extensions", params)
		if err != nil {
			logger.ErrorContext(ctx, "list extensions failed", "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return buildListResult(items), nil
	})
}

func registerListQuarantinedFiles(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("datto_edr_list_quarantined_files",
		mcp.WithDescription("List Datto EDR quarantined files. Supports LoopBack pagination."),
		mcp.WithNumber("limit", mcp.Description("Maximum number of results (default 100, max 1000)"), mcp.Min(1), mcp.Max(1000)),
		mcp.WithNumber("skip", mcp.Description("Number of results to skip for pagination"), mcp.Min(0)),
		mcp.WithString("order", mcp.Description("Sort order (e.g. 'createdAt DESC')")),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		params := make(map[string]string)
		addLoopBackPagination(params, req)

		items, err := client.GetList(ctx, "/api/QuarantinedFiles", params)
		if err != nil {
			logger.ErrorContext(ctx, "list quarantined files failed", "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return buildListResult(items), nil
	})
}

func registerGetDashboard(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("datto_edr_get_dashboard",
		mcp.WithDescription("Get Datto EDR dashboard data with summary statistics."),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		items, err := client.GetList(ctx, "/api/DashboardData", nil)
		if err != nil {
			logger.ErrorContext(ctx, "get dashboard failed", "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return buildListResult(items), nil
	})
}

func registerGetAlertCount(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("datto_edr_get_alert_count",
		mcp.WithDescription("Get the total count of Datto EDR alerts."),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		result, err := client.Get(ctx, "/api/Alerts/count", nil)
		if err != nil {
			logger.ErrorContext(ctx, "get alert count failed", "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return mcputil.JSONResult(result), nil
	})
}

func registerGetAgentCount(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("datto_edr_get_agent_count",
		mcp.WithDescription("Get the total count of Datto EDR agents."),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		result, err := client.Get(ctx, "/api/Agents/count", nil)
		if err != nil {
			logger.ErrorContext(ctx, "get agent count failed", "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return mcputil.JSONResult(result), nil
	})
}

// --- action tool registrations ----------------------------------------------

func registerScanAgent(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("datto_edr_scan_agent",
		mcp.WithDescription("Initiate a scan on a Datto EDR agent. This triggers a full endpoint scan."),
		mcp.WithString("agentId", mcp.Description("The agent ID to scan"), mcp.Required()),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		agentId := req.GetString("agentId", "")
		if agentId == "" {
			return mcputil.ErrorResult(fmt.Errorf("agentId is required")), nil
		}

		body := map[string]any{"id": agentId}
		result, err := client.Post(ctx, "/api/Agents/scan", body)
		if err != nil {
			logger.ErrorContext(ctx, "scan agent failed", "agentId", agentId, "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return mcputil.JSONResult(result), nil
	})
}

func registerIsolateHost(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("datto_edr_isolate_host",
		mcp.WithDescription("Isolate a host from the network via Datto EDR. The agent will only communicate with the EDR platform. Use datto_edr_restore_host to undo."),
		mcp.WithString("agentId", mcp.Description("The agent ID to isolate"), mcp.Required()),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		agentId := req.GetString("agentId", "")
		if agentId == "" {
			return mcputil.ErrorResult(fmt.Errorf("agentId is required")), nil
		}

		body := map[string]any{"id": agentId, "isolate": true}
		result, err := client.Post(ctx, "/api/Agents/toggleIsolation", body)
		if err != nil {
			logger.ErrorContext(ctx, "isolate host failed", "agentId", agentId, "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return mcputil.JSONResult(result), nil
	})
}

func registerRestoreHost(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("datto_edr_restore_host",
		mcp.WithDescription("Restore a previously isolated host back to normal network connectivity via Datto EDR."),
		mcp.WithString("agentId", mcp.Description("The agent ID to restore"), mcp.Required()),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		agentId := req.GetString("agentId", "")
		if agentId == "" {
			return mcputil.ErrorResult(fmt.Errorf("agentId is required")), nil
		}

		body := map[string]any{"id": agentId, "isolate": false}
		result, err := client.Post(ctx, "/api/Agents/toggleIsolation", body)
		if err != nil {
			logger.ErrorContext(ctx, "restore host failed", "agentId", agentId, "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return mcputil.JSONResult(result), nil
	})
}

