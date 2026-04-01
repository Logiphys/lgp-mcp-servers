package dattormm

import (
	"context"
	"log/slog"

	"github.com/Logiphys/lgp-mcp/pkg/mcputil"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerSystemTools(srv *server.MCPServer, c *Client, _ *slog.Logger) {
	srv.AddTool(
		mcp.NewTool("datto_get_system_status",
			mcp.WithDescription("Retrieve the current system status of the Datto RMM platform."),
			mcp.WithReadOnlyHintAnnotation(true),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			result, err := c.Get(ctx, "/system/status", nil)
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.JSONResult(result), nil
		},
	)

	srv.AddTool(
		mcp.NewTool("datto_get_rate_limit",
			mcp.WithDescription("Retrieve the current API rate limit information for Datto RMM."),
			mcp.WithReadOnlyHintAnnotation(true),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			result, err := c.Get(ctx, "/rate-limit", nil)
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.JSONResult(result), nil
		},
	)

	srv.AddTool(
		mcp.NewTool("datto_get_pagination_config",
			mcp.WithDescription("Retrieve the pagination configuration for the Datto RMM API."),
			mcp.WithReadOnlyHintAnnotation(true),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			result, err := c.Get(ctx, "/pagination", nil)
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.JSONResult(result), nil
		},
	)
}
