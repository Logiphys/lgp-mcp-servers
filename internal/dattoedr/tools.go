package dattoedr

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/Logiphys/lgp-mcp/pkg/mcputil"
)

// RegisterTools registers all Datto EDR MCP tools on the given server.
func RegisterTools(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	// Read-only tools
	registerTestConnection(srv, client, logger)
	registerListAgents(srv, client, logger)
	registerGetAgent(srv, client, logger)
	registerListAlerts(srv, client, logger)
	registerGetAlert(srv, client, logger)
	registerListAlertsArchive(srv, client, logger)
	registerListOrganizations(srv, client, logger)
	registerListLocations(srv, client, logger)
	registerListDeviceGroups(srv, client, logger)
	registerListPolicies(srv, client, logger)
	registerListRules(srv, client, logger)
	registerListSuppressionRules(srv, client, logger)
	registerListExtensions(srv, client, logger)
	registerListQuarantinedFiles(srv, client, logger)
	registerGetDashboard(srv, client, logger)

	registerListScans(srv, client, logger)
	registerGetScan(srv, client, logger)
	registerListScanHosts(srv, client, logger)
	registerListJobs(srv, client, logger)
	registerGetHostScanResult(srv, client, logger)
	registerGetResponseResult(srv, client, logger)
	registerGetTaskStatus(srv, client, logger)
	registerListActivityTraces(srv, client, logger)
	registerGetAlertCount(srv, client, logger)
	registerGetAgentCount(srv, client, logger)

	// Action tools
	registerScanAgent(srv, client, logger)
	registerIsolateHost(srv, client, logger)
	registerRestoreHost(srv, client, logger)
	registerRunExtension(srv, client, logger)
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

func registerListScans(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("datto_edr_list_scans",
		mcp.WithDescription("List Datto EDR scan history with status and timestamps. Supports LoopBack pagination."),
		mcp.WithNumber("limit", mcp.Description("Maximum number of results (default 100, max 1000)"), mcp.Min(1), mcp.Max(1000)),
		mcp.WithNumber("skip", mcp.Description("Number of results to skip for pagination"), mcp.Min(0)),
		mcp.WithString("order", mcp.Description("Sort order (e.g. 'createdAt DESC')")),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		limit := req.GetInt("limit", 100)
		if limit > 1000 {
			limit = 1000
		}
		params := BuildJSONFilter(limit, req.GetInt("skip", 0), req.GetString("order", ""))

		items, err := client.GetList(ctx, "/api/Scans", params)
		if err != nil {
			logger.ErrorContext(ctx, "list scans failed", "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return buildListResult(items), nil
	})
}

func registerGetScan(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("datto_edr_get_scan",
		mcp.WithDescription("Get details of a specific Datto EDR scan by ID."),
		mcp.WithString("id", mcp.Description("The scan ID"), mcp.Required()),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id := req.GetString("id", "")
		if id == "" {
			return mcputil.ErrorResult(fmt.Errorf("id is required")), nil
		}

		path := fmt.Sprintf("/api/Scans/%s", id)
		result, err := client.Get(ctx, path, nil)
		if err != nil {
			logger.ErrorContext(ctx, "get scan failed", "id", id, "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return mcputil.JSONResult(result), nil
	})
}

func registerListScanHosts(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("datto_edr_list_scan_hosts",
		mcp.WithDescription("List hosts scanned with per-host results. Filter by scanId. Supports LoopBack pagination."),
		mcp.WithString("scanId", mcp.Description("Filter by scan ID")),
		mcp.WithNumber("limit", mcp.Description("Maximum number of results (default 100, max 1000)"), mcp.Min(1), mcp.Max(1000)),
		mcp.WithNumber("skip", mcp.Description("Number of results to skip for pagination"), mcp.Min(0)),
		mcp.WithString("order", mcp.Description("Sort order (e.g. 'createdAt DESC')")),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		limit := req.GetInt("limit", 100)
		if limit > 1000 {
			limit = 1000
		}
		params := BuildJSONFilter(limit, req.GetInt("skip", 0), req.GetString("order", ""))

		scanId := req.GetString("scanId", "")
		if scanId != "" {
			// Add where clause to the JSON filter
			params["filter"] = addWhereToJSONFilter(params["filter"], "scanId", scanId)
		}

		items, err := client.GetList(ctx, "/api/ScanHosts", params)
		if err != nil {
			logger.ErrorContext(ctx, "list scan hosts failed", "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return buildListResult(items), nil
	})
}

func registerListJobs(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("datto_edr_list_jobs",
		mcp.WithDescription("List Datto EDR background jobs. Supports LoopBack pagination."),
		mcp.WithNumber("limit", mcp.Description("Maximum number of results (default 100, max 1000)"), mcp.Min(1), mcp.Max(1000)),
		mcp.WithNumber("skip", mcp.Description("Number of results to skip for pagination"), mcp.Min(0)),
		mcp.WithString("order", mcp.Description("Sort order (e.g. 'createdAt DESC')")),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		limit := req.GetInt("limit", 100)
		if limit > 1000 {
			limit = 1000
		}
		params := BuildJSONFilter(limit, req.GetInt("skip", 0), req.GetString("order", ""))

		items, err := client.GetList(ctx, "/api/Jobs", params)
		if err != nil {
			logger.ErrorContext(ctx, "list jobs failed", "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return buildListResult(items), nil
	})
}

func registerGetHostScanResult(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("datto_edr_get_host_scan_result",
		mcp.WithDescription("Get detailed scan result for a specific host by ID."),
		mcp.WithString("id", mcp.Description("The host scan result ID"), mcp.Required()),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id := req.GetString("id", "")
		if id == "" {
			return mcputil.ErrorResult(fmt.Errorf("id is required")), nil
		}

		path := fmt.Sprintf("/api/HostScanResults/%s", id)
		result, err := client.Get(ctx, path, nil)
		if err != nil {
			logger.ErrorContext(ctx, "get host scan result failed", "id", id, "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return mcputil.JSONResult(result), nil
	})
}

func registerGetResponseResult(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("datto_edr_get_response_result",
		mcp.WithDescription("Get the result of a response action by ID."),
		mcp.WithString("id", mcp.Description("The response result ID"), mcp.Required()),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id := req.GetString("id", "")
		if id == "" {
			return mcputil.ErrorResult(fmt.Errorf("id is required")), nil
		}

		path := fmt.Sprintf("/api/ResponseResults/%s", id)
		result, err := client.Get(ctx, path, nil)
		if err != nil {
			logger.ErrorContext(ctx, "get response result failed", "id", id, "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return mcputil.JSONResult(result), nil
	})
}

func registerGetTaskStatus(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("datto_edr_get_task_status",
		mcp.WithDescription("Check the status of an async task by ID."),
		mcp.WithString("id", mcp.Description("The user task ID"), mcp.Required()),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id := req.GetString("id", "")
		if id == "" {
			return mcputil.ErrorResult(fmt.Errorf("id is required")), nil
		}

		path := fmt.Sprintf("/api/UserTasks/%s", id)
		result, err := client.Get(ctx, path, nil)
		if err != nil {
			logger.ErrorContext(ctx, "get task status failed", "id", id, "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return mcputil.JSONResult(result), nil
	})
}

func registerListActivityTraces(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("datto_edr_list_activity_traces",
		mcp.WithDescription("List timeline of host activities. Filter by agentId. Supports LoopBack pagination."),
		mcp.WithString("agentId", mcp.Description("Filter by agent ID")),
		mcp.WithNumber("limit", mcp.Description("Maximum number of results (default 100, max 1000)"), mcp.Min(1), mcp.Max(1000)),
		mcp.WithNumber("skip", mcp.Description("Number of results to skip for pagination"), mcp.Min(0)),
		mcp.WithString("order", mcp.Description("Sort order (e.g. 'createdAt DESC')")),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		params := make(map[string]string)
		addLoopBackPagination(params, req)
		AddWhereFilter(params, "agentId", req.GetString("agentId", ""))

		items, err := client.GetList(ctx, "/api/ActivityTraces", params)
		if err != nil {
			logger.ErrorContext(ctx, "list activity traces failed", "err", err)
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

func registerRunExtension(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("datto_edr_run_extension",
		mcp.WithDescription("Run a Datto EDR extension on a target agent."),
		mcp.WithString("extensionId", mcp.Description("The extension ID to run"), mcp.Required()),
		mcp.WithString("agentId", mcp.Description("The agent ID to run the extension on"), mcp.Required()),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		extensionId := req.GetString("extensionId", "")
		if extensionId == "" {
			return mcputil.ErrorResult(fmt.Errorf("extensionId is required")), nil
		}
		agentId := req.GetString("agentId", "")
		if agentId == "" {
			return mcputil.ErrorResult(fmt.Errorf("agentId is required")), nil
		}

		body := map[string]any{"extensionId": extensionId, "agentId": agentId}
		result, err := client.Post(ctx, "/api/Extensions/run", body)
		if err != nil {
			logger.ErrorContext(ctx, "run extension failed", "extensionId", extensionId, "agentId", agentId, "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return mcputil.JSONResult(result), nil
	})
}

// addWhereToJSONFilter adds a where clause to an existing JSON filter string.
func addWhereToJSONFilter(filterJSON, field, value string) string {
	filter := make(map[string]any)
	if filterJSON != "" {
		_ = json.Unmarshal([]byte(filterJSON), &filter)
	}
	where, ok := filter["where"].(map[string]any)
	if !ok {
		where = make(map[string]any)
	}
	where[field] = value
	filter["where"] = where
	b, _ := json.Marshal(filter)
	return string(b)
}
