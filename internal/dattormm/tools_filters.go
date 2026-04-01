package dattormm

import (
	"context"
	"log/slog"

	"github.com/Logiphys/lgp-mcp/pkg/mcputil"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerFilterTools(srv *server.MCPServer, c *Client, _ *slog.Logger) {
	srv.AddTool(
		mcp.NewTool("datto_list_default_filters",
			mcp.WithDescription("List the default device filters defined in the Datto RMM account, with pagination."),
			mcp.WithReadOnlyHintAnnotation(true),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			params := paginationParams(req)
			items, pageInfo, err := c.GetList(ctx, "/account/filters/default", params)
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return listResult(items, pageInfo), nil
		},
	)

	srv.AddTool(
		mcp.NewTool("datto_list_custom_filters",
			mcp.WithDescription("List the custom device filters defined in the Datto RMM account, with pagination."),
			mcp.WithReadOnlyHintAnnotation(true),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			params := paginationParams(req)
			items, pageInfo, err := c.GetList(ctx, "/account/filters/custom", params)
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return listResult(items, pageInfo), nil
		},
	)
}
