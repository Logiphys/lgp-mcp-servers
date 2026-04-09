package itglue

import (
	"context"
	"log/slog"

	"github.com/Logiphys/lgp-mcp-servers/pkg/mcputil"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerHealthTools(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	srv.AddTool(
		mcp.NewTool("health_check",
			mcp.WithDescription("Test the connection to the IT Glue API. Returns a success message or an error if the connection fails."),
			mcp.WithReadOnlyHintAnnotation(true),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			if err := client.TestConnection(ctx); err != nil {
				logger.Error("itglue_health_check failed", "error", err)
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.TextResult("IT Glue API connection successful."), nil
		},
	)
}
