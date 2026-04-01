package rocketcyber

import (
	"context"
	"log/slog"
	"strconv"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/Logiphys/lgp-mcp/pkg/mcputil"
)

// RegisterTools registers all RocketCyber MCP tools on the given server.
func RegisterTools(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	registerTestConnection(srv, client, logger)
	registerGetAccount(srv, client, logger)
	registerListAgents(srv, client, logger)
	registerListIncidents(srv, client, logger)
	registerListEvents(srv, client, logger)
	registerGetEventSummary(srv, client, logger)
	registerListFirewalls(srv, client, logger)
	registerListApps(srv, client, logger)
	registerGetDefender(srv, client, logger)
	registerGetOffice(srv, client, logger)
}

// --- helpers ----------------------------------------------------------------

func addPaginationParams(params map[string]string, req mcp.CallToolRequest) {
	if v := req.GetInt("page", 0); v > 0 {
		params["page"] = strconv.Itoa(v)
	}
	if v := req.GetInt("pageSize", 0); v > 0 {
		params["pageSize"] = strconv.Itoa(v)
	}
}

func addDateRangeParams(params map[string]string, req mcp.CallToolRequest) {
	if v := req.GetString("startDate", ""); v != "" {
		params["startDate"] = v
	}
	if v := req.GetString("endDate", ""); v != "" {
		params["endDate"] = v
	}
}

func buildListResult(items []any, pageInfo *PageInfo) *mcp.CallToolResult {
	result := map[string]any{"data": items}
	if pageInfo != nil {
		result["pagination"] = map[string]any{
			"total_count": pageInfo.TotalCount,
			"page":        pageInfo.Page,
			"page_size":   pageInfo.PageSize,
		}
	}
	return mcputil.JSONResult(result)
}

// --- tool registrations -----------------------------------------------------

func registerTestConnection(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("rocketcyber_test_connection",
		mcp.WithDescription("Test connectivity to the RocketCyber API. Returns success or an error message."),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if err := client.TestConnection(ctx); err != nil {
			logger.ErrorContext(ctx, "test connection failed", "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return mcputil.TextResult("RocketCyber API connection successful"), nil
	})
}

func registerGetAccount(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("rocketcyber_get_account",
		mcp.WithDescription("Get RocketCyber account details. If accountId is provided, returns that specific account; otherwise returns the current account."),
		mcp.WithNumber("accountId", mcp.Description("Optional account ID to retrieve a specific account")),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		path := "/account"
		params := make(map[string]string)
		if id := req.GetInt("accountId", 0); id > 0 {
			params["accountId"] = strconv.Itoa(id)
		}

		result, err := client.Get(ctx, path, params)
		if err != nil {
			logger.ErrorContext(ctx, "get account failed", "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return mcputil.JSONResult(result), nil
	})
}

func registerListAgents(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("rocketcyber_list_agents",
		mcp.WithDescription("List RocketCyber agents with optional filters. Supports pagination and date range filtering."),
		mcp.WithNumber("accountId", mcp.Description("Filter by account ID")),
		mcp.WithString("status", mcp.Description("Filter by agent status")),
		mcp.WithString("hostname", mcp.Description("Filter by hostname")),
		mcp.WithString("platform", mcp.Description("Filter by platform (e.g. windows, mac, linux)")),
		mcp.WithNumber("page", mcp.Description("Page number for pagination"), mcp.Min(1)),
		mcp.WithNumber("pageSize", mcp.Description("Number of results per page"), mcp.Min(1), mcp.Max(1000)),
		mcp.WithString("startDate", mcp.Description("Start date filter (ISO 8601 format)")),
		mcp.WithString("endDate", mcp.Description("End date filter (ISO 8601 format)")),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		params := make(map[string]string)
		addPaginationParams(params, req)
		addDateRangeParams(params, req)
		if id := req.GetInt("accountId", 0); id > 0 {
			params["accountId"] = strconv.Itoa(id)
		}
		if v := req.GetString("status", ""); v != "" {
			params["status"] = v
		}
		if v := req.GetString("hostname", ""); v != "" {
			params["hostname"] = v
		}
		if v := req.GetString("platform", ""); v != "" {
			params["platform"] = v
		}

		items, pageInfo, err := client.GetList(ctx, "/agents", params)
		if err != nil {
			logger.ErrorContext(ctx, "list agents failed", "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return buildListResult(items, pageInfo), nil
	})
}

func registerListIncidents(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("rocketcyber_list_incidents",
		mcp.WithDescription("List RocketCyber incidents with optional filters. Supports pagination and date range filtering."),
		mcp.WithString("status", mcp.Description("Filter by status: active, resolved, or all")),
		mcp.WithString("severity", mcp.Description("Filter by severity level")),
		mcp.WithString("title", mcp.Description("Filter by incident title (partial match)")),
		mcp.WithNumber("page", mcp.Description("Page number for pagination"), mcp.Min(1)),
		mcp.WithNumber("pageSize", mcp.Description("Number of results per page"), mcp.Min(1), mcp.Max(1000)),
		mcp.WithString("startDate", mcp.Description("Start date filter (ISO 8601 format)")),
		mcp.WithString("endDate", mcp.Description("End date filter (ISO 8601 format)")),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		params := make(map[string]string)
		addPaginationParams(params, req)
		addDateRangeParams(params, req)
		if v := req.GetString("status", ""); v != "" {
			params["status"] = v
		}
		if v := req.GetString("severity", ""); v != "" {
			params["severity"] = v
		}
		if v := req.GetString("title", ""); v != "" {
			params["title"] = v
		}

		items, pageInfo, err := client.GetList(ctx, "/incidents", params)
		if err != nil {
			logger.ErrorContext(ctx, "list incidents failed", "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return buildListResult(items, pageInfo), nil
	})
}

func registerListEvents(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("rocketcyber_list_events",
		mcp.WithDescription("List RocketCyber security events with optional filters. Supports pagination and date range filtering."),
		mcp.WithString("eventType", mcp.Description("Filter by event type")),
		mcp.WithString("severity", mcp.Description("Filter by severity level")),
		mcp.WithString("hostname", mcp.Description("Filter by hostname")),
		mcp.WithNumber("page", mcp.Description("Page number for pagination"), mcp.Min(1)),
		mcp.WithNumber("pageSize", mcp.Description("Number of results per page"), mcp.Min(1), mcp.Max(1000)),
		mcp.WithString("startDate", mcp.Description("Start date filter (ISO 8601 format)")),
		mcp.WithString("endDate", mcp.Description("End date filter (ISO 8601 format)")),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		params := make(map[string]string)
		addPaginationParams(params, req)
		addDateRangeParams(params, req)
		if v := req.GetString("eventType", ""); v != "" {
			params["eventType"] = v
		}
		if v := req.GetString("severity", ""); v != "" {
			params["severity"] = v
		}
		if v := req.GetString("hostname", ""); v != "" {
			params["hostname"] = v
		}

		items, pageInfo, err := client.GetList(ctx, "/events", params)
		if err != nil {
			logger.ErrorContext(ctx, "list events failed", "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return buildListResult(items, pageInfo), nil
	})
}

func registerGetEventSummary(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("rocketcyber_get_event_summary",
		mcp.WithDescription("Get a summary of RocketCyber security events, optionally filtered by account and date range."),
		mcp.WithNumber("accountId", mcp.Description("Filter by account ID")),
		mcp.WithString("startDate", mcp.Description("Start date filter (ISO 8601 format)")),
		mcp.WithString("endDate", mcp.Description("End date filter (ISO 8601 format)")),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		params := make(map[string]string)
		addDateRangeParams(params, req)
		if id := req.GetInt("accountId", 0); id > 0 {
			params["accountId"] = strconv.Itoa(id)
		}

		result, err := client.Get(ctx, "/events/summary", params)
		if err != nil {
			logger.ErrorContext(ctx, "get event summary failed", "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return mcputil.JSONResult(result), nil
	})
}

func registerListFirewalls(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("rocketcyber_list_firewalls",
		mcp.WithDescription("List RocketCyber-monitored firewalls with optional filters. Supports pagination."),
		mcp.WithString("connectivity", mcp.Description("Filter by connectivity status")),
		mcp.WithString("hostname", mcp.Description("Filter by hostname")),
		mcp.WithString("vendor", mcp.Description("Filter by firewall vendor")),
		mcp.WithNumber("page", mcp.Description("Page number for pagination"), mcp.Min(1)),
		mcp.WithNumber("pageSize", mcp.Description("Number of results per page"), mcp.Min(1), mcp.Max(1000)),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		params := make(map[string]string)
		addPaginationParams(params, req)
		if v := req.GetString("connectivity", ""); v != "" {
			params["connectivity"] = v
		}
		if v := req.GetString("hostname", ""); v != "" {
			params["hostname"] = v
		}
		if v := req.GetString("vendor", ""); v != "" {
			params["vendor"] = v
		}

		items, pageInfo, err := client.GetList(ctx, "/firewalls", params)
		if err != nil {
			logger.ErrorContext(ctx, "list firewalls failed", "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return buildListResult(items, pageInfo), nil
	})
}

func registerListApps(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("rocketcyber_list_apps",
		mcp.WithDescription("List RocketCyber security apps with optional filters."),
		mcp.WithString("status", mcp.Description("Filter by app status")),
		mcp.WithString("name", mcp.Description("Filter by app name")),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		params := make(map[string]string)
		if v := req.GetString("status", ""); v != "" {
			params["status"] = v
		}
		if v := req.GetString("name", ""); v != "" {
			params["name"] = v
		}

		items, pageInfo, err := client.GetList(ctx, "/apps", params)
		if err != nil {
			logger.ErrorContext(ctx, "list apps failed", "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return buildListResult(items, pageInfo), nil
	})
}

func registerGetDefender(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("rocketcyber_get_defender",
		mcp.WithDescription("Get Windows Defender status and details from RocketCyber. Optionally filter by account ID."),
		mcp.WithNumber("accountId", mcp.Description("Optional account ID to filter results")),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		params := make(map[string]string)
		if id := req.GetInt("accountId", 0); id > 0 {
			params["accountId"] = strconv.Itoa(id)
		}

		result, err := client.Get(ctx, "/defender", params)
		if err != nil {
			logger.ErrorContext(ctx, "get defender failed", "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return mcputil.JSONResult(result), nil
	})
}

func registerGetOffice(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("rocketcyber_get_office",
		mcp.WithDescription("Get Microsoft 365 / Office 365 status and details from RocketCyber. Optionally filter by account ID."),
		mcp.WithNumber("accountId", mcp.Description("Optional account ID to filter results")),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		params := make(map[string]string)
		if id := req.GetInt("accountId", 0); id > 0 {
			params["accountId"] = strconv.Itoa(id)
		}

		result, err := client.Get(ctx, "/office365", params)
		if err != nil {
			logger.ErrorContext(ctx, "get office365 failed", "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return mcputil.JSONResult(result), nil
	})
}
