package dattormm

import (
	"context"
	"log/slog"
	"strconv"

	"github.com/Logiphys/lgp-mcp/pkg/mcputil"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerActivityTools(srv *server.MCPServer, c *Client, _ *slog.Logger, tier int) {
	if tier < 2 {
		return
	}
	srv.AddTool(
		mcp.NewTool("datto_get_activity_logs",
			mcp.WithDescription("Retrieve activity logs from the Datto RMM account."),
			mcp.WithNumber("size",
				mcp.Description("Number of log entries to return."),
			),
			mcp.WithString("order",
				mcp.Description("Sort order for results: asc or desc."),
			),
			mcp.WithString("from",
				mcp.Description("Start of the time range as an ISO 8601 datetime (e.g. 2024-01-01T00:00:00Z)."),
			),
			mcp.WithString("until",
				mcp.Description("End of the time range as an ISO 8601 datetime (e.g. 2024-01-31T23:59:59Z)."),
			),
			mcp.WithReadOnlyHintAnnotation(true),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.GetArguments()
			params := map[string]string{}

			if v, ok := args["size"]; ok && v != nil {
				switch n := v.(type) {
				case float64:
					params["size"] = strconv.Itoa(int(n))
				case int:
					params["size"] = strconv.Itoa(n)
				}
			}
			if v, ok := args["order"].(string); ok && v != "" {
				params["order"] = v
			}
			if v, ok := args["from"].(string); ok && v != "" {
				params["from"] = v
			}
			if v, ok := args["until"].(string); ok && v != "" {
				params["until"] = v
			}

			result, err := c.Get(ctx, "/activity-logs", params)
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.JSONResult(result), nil
		},
	)
}
