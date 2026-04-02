package myitprocess

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/Logiphys/lgp-mcp/pkg/mcputil"
)

// RegisterTools registers all MyITProcess MCP tools on the given server.
func RegisterTools(srv *server.MCPServer, client *Client, logger *slog.Logger, tier int) {
	// Tier 1 — Safe Read-Only
	registerTestConnection(srv, client, logger)
	registerListReviews(srv, client, logger)
	registerListOverdueReviews(srv, client, logger)
	registerListFindings(srv, client, logger)
	registerListRecommendations(srv, client, logger)
	registerGetRecommendationConfigurations(srv, client, logger)
	registerListInitiatives(srv, client, logger)

	// Tier 2 — Sensitive (client/user data)
	if tier >= 2 {
		registerListClients(srv, client, logger)
		registerListUsers(srv, client, logger)
		registerListMeetings(srv, client, logger)
	}
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

// addQueryFilter appends a queryFilters entry to the params map.
// MyITProcess expects: queryFilters={"field":"<field>","predicate":"<op>","condition":"<value>"}
// Multiple filters are passed as repeated queryFilters params; since we use a map,
// we serialize them as a JSON array.
type queryFilter struct {
	Field     string `json:"field"`
	Predicate string `json:"predicate"`
	Condition string `json:"condition"`
}

func buildQueryFilters(filters []queryFilter) string {
	if len(filters) == 0 {
		return ""
	}
	if len(filters) == 1 {
		b, _ := json.Marshal(filters[0])
		return string(b)
	}
	b, _ := json.Marshal(filters)
	return string(b)
}

func addSortingRule(params map[string]string, field, direction string) {
	rule := map[string]string{"field": field, "direction": direction}
	b, _ := json.Marshal(rule)
	params["sortingRules"] = string(b)
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
	tool := mcp.NewTool("myitprocess_test_connection",
		mcp.WithDescription("Test connectivity to the MyITProcess API. Returns success or an error message."),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if err := client.TestConnection(ctx); err != nil {
			logger.ErrorContext(ctx, "test connection failed", "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return mcputil.TextResult("MyITProcess API connection successful"), nil
	})
}

func registerListClients(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("myitprocess_list_clients",
		mcp.WithDescription("List MyITProcess clients with optional filters."),
		mcp.WithString("name", mcp.Description("Filter by client name (contains match)")),
		mcp.WithString("isActive", mcp.Description("Filter by active status: true or false")),
		mcp.WithNumber("page", mcp.Description("Page number for pagination"), mcp.Min(1)),
		mcp.WithNumber("pageSize", mcp.Description("Number of results per page"), mcp.Min(1), mcp.Max(100)),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		params := make(map[string]string)
		addPaginationParams(params, req)

		var filters []queryFilter
		if v := req.GetString("name", ""); v != "" {
			filters = append(filters, queryFilter{Field: "name", Predicate: "contains", Condition: v})
		}
		if v := req.GetString("isActive", ""); v != "" {
			filters = append(filters, queryFilter{Field: "isActive", Predicate: "equal", Condition: v})
		}
		if qf := buildQueryFilters(filters); qf != "" {
			params["queryFilters"] = qf
		}

		items, pageInfo, err := client.GetList(ctx, "/clients", params)
		if err != nil {
			logger.ErrorContext(ctx, "list clients failed", "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return buildListResult(items, pageInfo), nil
	})
}

func registerListUsers(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("myitprocess_list_users",
		mcp.WithDescription("List MyITProcess users with optional filters."),
		mcp.WithString("firstName", mcp.Description("Filter by first name (contains match)")),
		mcp.WithString("lastName", mcp.Description("Filter by last name (contains match)")),
		mcp.WithString("roleName", mcp.Description("Filter by role name (contains match)")),
		mcp.WithNumber("page", mcp.Description("Page number for pagination"), mcp.Min(1)),
		mcp.WithNumber("pageSize", mcp.Description("Number of results per page"), mcp.Min(1), mcp.Max(100)),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		params := make(map[string]string)
		addPaginationParams(params, req)

		var filters []queryFilter
		if v := req.GetString("firstName", ""); v != "" {
			filters = append(filters, queryFilter{Field: "firstName", Predicate: "contains", Condition: v})
		}
		if v := req.GetString("lastName", ""); v != "" {
			filters = append(filters, queryFilter{Field: "lastName", Predicate: "contains", Condition: v})
		}
		if v := req.GetString("roleName", ""); v != "" {
			filters = append(filters, queryFilter{Field: "roleName", Predicate: "contains", Condition: v})
		}
		if qf := buildQueryFilters(filters); qf != "" {
			params["queryFilters"] = qf
		}

		items, pageInfo, err := client.GetList(ctx, "/users", params)
		if err != nil {
			logger.ErrorContext(ctx, "list users failed", "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return buildListResult(items, pageInfo), nil
	})
}

func registerListReviews(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("myitprocess_list_reviews",
		mcp.WithDescription("List MyITProcess reviews with optional filters."),
		mcp.WithString("status", mcp.Description("Filter by review status")),
		mcp.WithString("clientName", mcp.Description("Filter by client name (contains match)")),
		mcp.WithNumber("page", mcp.Description("Page number for pagination"), mcp.Min(1)),
		mcp.WithNumber("pageSize", mcp.Description("Number of results per page"), mcp.Min(1), mcp.Max(100)),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		params := make(map[string]string)
		addPaginationParams(params, req)

		var filters []queryFilter
		if v := req.GetString("status", ""); v != "" {
			filters = append(filters, queryFilter{Field: "status", Predicate: "equal", Condition: v})
		}
		if v := req.GetString("clientName", ""); v != "" {
			filters = append(filters, queryFilter{Field: "clientName", Predicate: "contains", Condition: v})
		}
		if qf := buildQueryFilters(filters); qf != "" {
			params["queryFilters"] = qf
		}

		items, pageInfo, err := client.GetList(ctx, "/reviews", params)
		if err != nil {
			logger.ErrorContext(ctx, "list reviews failed", "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return buildListResult(items, pageInfo), nil
	})
}

func registerListOverdueReviews(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("myitprocess_list_overdue_reviews",
		mcp.WithDescription("List MyITProcess overdue reviews."),
		mcp.WithNumber("page", mcp.Description("Page number for pagination"), mcp.Min(1)),
		mcp.WithNumber("pageSize", mcp.Description("Number of results per page"), mcp.Min(1), mcp.Max(100)),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		params := make(map[string]string)
		addPaginationParams(params, req)

		items, pageInfo, err := client.GetList(ctx, "/reviews/categories/overdue", params)
		if err != nil {
			logger.ErrorContext(ctx, "list overdue reviews failed", "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return buildListResult(items, pageInfo), nil
	})
}

func registerListFindings(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("myitprocess_list_findings",
		mcp.WithDescription("List MyITProcess findings with optional filters."),
		mcp.WithNumber("reviewId", mcp.Description("Filter by review ID")),
		mcp.WithString("isArchived", mcp.Description("Filter by archived status: true or false")),
		mcp.WithNumber("page", mcp.Description("Page number for pagination"), mcp.Min(1)),
		mcp.WithNumber("pageSize", mcp.Description("Number of results per page"), mcp.Min(1), mcp.Max(100)),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		params := make(map[string]string)
		addPaginationParams(params, req)

		var filters []queryFilter
		if v := req.GetInt("reviewId", 0); v > 0 {
			filters = append(filters, queryFilter{Field: "reviewId", Predicate: "equal", Condition: strconv.Itoa(v)})
		}
		if v := req.GetString("isArchived", ""); v != "" {
			filters = append(filters, queryFilter{Field: "isArchived", Predicate: "equal", Condition: v})
		}
		if qf := buildQueryFilters(filters); qf != "" {
			params["queryFilters"] = qf
		}

		items, pageInfo, err := client.GetList(ctx, "/findings", params)
		if err != nil {
			logger.ErrorContext(ctx, "list findings failed", "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return buildListResult(items, pageInfo), nil
	})
}

func registerListRecommendations(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("myitprocess_list_recommendations",
		mcp.WithDescription("List MyITProcess recommendations with optional filters."),
		mcp.WithNumber("clientId", mcp.Description("Filter by client ID")),
		mcp.WithString("status", mcp.Description("Filter by recommendation status")),
		mcp.WithString("priority", mcp.Description("Filter by priority")),
		mcp.WithString("type", mcp.Description("Filter by recommendation type")),
		mcp.WithNumber("page", mcp.Description("Page number for pagination"), mcp.Min(1)),
		mcp.WithNumber("pageSize", mcp.Description("Number of results per page"), mcp.Min(1), mcp.Max(100)),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		params := make(map[string]string)
		addPaginationParams(params, req)

		var filters []queryFilter
		if v := req.GetInt("clientId", 0); v > 0 {
			filters = append(filters, queryFilter{Field: "clientId", Predicate: "equal", Condition: strconv.Itoa(v)})
		}
		if v := req.GetString("status", ""); v != "" {
			filters = append(filters, queryFilter{Field: "status", Predicate: "equal", Condition: v})
		}
		if v := req.GetString("priority", ""); v != "" {
			filters = append(filters, queryFilter{Field: "priority", Predicate: "equal", Condition: v})
		}
		if v := req.GetString("type", ""); v != "" {
			filters = append(filters, queryFilter{Field: "type", Predicate: "equal", Condition: v})
		}
		if qf := buildQueryFilters(filters); qf != "" {
			params["queryFilters"] = qf
		}

		items, pageInfo, err := client.GetList(ctx, "/recommendations", params)
		if err != nil {
			logger.ErrorContext(ctx, "list recommendations failed", "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return buildListResult(items, pageInfo), nil
	})
}

func registerGetRecommendationConfigurations(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("myitprocess_get_recommendation_configurations",
		mcp.WithDescription("Get configurations for a specific MyITProcess recommendation by ID."),
		mcp.WithNumber("id", mcp.Description("The recommendation ID"), mcp.Required()),
		mcp.WithNumber("page", mcp.Description("Page number for pagination"), mcp.Min(1)),
		mcp.WithNumber("pageSize", mcp.Description("Number of results per page"), mcp.Min(1), mcp.Max(100)),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id := req.GetInt("id", 0)
		if id == 0 {
			return mcputil.ErrorResult(fmt.Errorf("id is required")), nil
		}

		params := make(map[string]string)
		addPaginationParams(params, req)

		path := fmt.Sprintf("/recommendations/%d/configurations", id)
		items, pageInfo, err := client.GetList(ctx, path, params)
		if err != nil {
			logger.ErrorContext(ctx, "get recommendation configurations failed", "id", id, "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return buildListResult(items, pageInfo), nil
	})
}

func registerListInitiatives(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("myitprocess_list_initiatives",
		mcp.WithDescription("List MyITProcess initiatives with optional filters."),
		mcp.WithNumber("clientId", mcp.Description("Filter by client ID")),
		mcp.WithString("isArchived", mcp.Description("Filter by archived status: true or false")),
		mcp.WithNumber("page", mcp.Description("Page number for pagination"), mcp.Min(1)),
		mcp.WithNumber("pageSize", mcp.Description("Number of results per page"), mcp.Min(1), mcp.Max(100)),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		params := make(map[string]string)
		addPaginationParams(params, req)

		var filters []queryFilter
		if v := req.GetInt("clientId", 0); v > 0 {
			filters = append(filters, queryFilter{Field: "clientId", Predicate: "equal", Condition: strconv.Itoa(v)})
		}
		if v := req.GetString("isArchived", ""); v != "" {
			filters = append(filters, queryFilter{Field: "isArchived", Predicate: "equal", Condition: v})
		}
		if qf := buildQueryFilters(filters); qf != "" {
			params["queryFilters"] = qf
		}

		items, pageInfo, err := client.GetList(ctx, "/initiatives", params)
		if err != nil {
			logger.ErrorContext(ctx, "list initiatives failed", "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return buildListResult(items, pageInfo), nil
	})
}

func registerListMeetings(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	tool := mcp.NewTool("myitprocess_list_meetings",
		mcp.WithDescription("List MyITProcess meetings with optional filters."),
		mcp.WithNumber("clientId", mcp.Description("Filter by client ID")),
		mcp.WithString("status", mcp.Description("Filter by meeting status")),
		mcp.WithNumber("page", mcp.Description("Page number for pagination"), mcp.Min(1)),
		mcp.WithNumber("pageSize", mcp.Description("Number of results per page"), mcp.Min(1), mcp.Max(100)),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		params := make(map[string]string)
		addPaginationParams(params, req)

		var filters []queryFilter
		if v := req.GetInt("clientId", 0); v > 0 {
			filters = append(filters, queryFilter{Field: "clientId", Predicate: "equal", Condition: strconv.Itoa(v)})
		}
		if v := req.GetString("status", ""); v != "" {
			filters = append(filters, queryFilter{Field: "status", Predicate: "equal", Condition: v})
		}
		if qf := buildQueryFilters(filters); qf != "" {
			params["queryFilters"] = qf
		}

		items, pageInfo, err := client.GetList(ctx, "/meetings", params)
		if err != nil {
			logger.ErrorContext(ctx, "list meetings failed", "err", err)
			return mcputil.ErrorResult(err), nil
		}
		return buildListResult(items, pageInfo), nil
	})
}
