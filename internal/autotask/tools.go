package autotask

import (
	"context"
	"log/slog"

	"github.com/Logiphys/lgp-mcp/pkg/mcputil"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// RegisterTools registers all Autotask MCP tools on the given server.
func RegisterTools(srv *server.MCPServer, client *Client, picklist *PicklistCache, logger *slog.Logger) {
	// === UTILITY TOOLS ===

	srv.AddTool(
		mcp.NewTool("autotask_test_connection",
			mcp.WithDescription("Test the connection to Autotask API"),
			mcp.WithReadOnlyHintAnnotation(true),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			if err := client.TestConnection(ctx); err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.TextResult(`{"message": "Connection to Autotask API successful"}`), nil
		},
	)

	// Tools will be added in batches (Tasks 4-8)
}
