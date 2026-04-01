package dattormm

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Logiphys/lgp-mcp/pkg/mcputil"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerAlertTools(srv *server.MCPServer, c *Client, _ *slog.Logger) {
	srv.AddTool(
		mcp.NewTool("datto_get_alert",
			mcp.WithDescription("Retrieve a single Datto RMM alert by its UID."),
			mcp.WithString("alertUid",
				mcp.Description("The UID of the alert to retrieve."),
				mcp.Required(),
			),
			mcp.WithReadOnlyHintAnnotation(true),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			alertUid := req.GetString("alertUid", "")
			if alertUid == "" {
				return mcputil.ErrorResult(fmt.Errorf("alertUid is required")), nil
			}
			result, err := c.Get(ctx, fmt.Sprintf("/alert/%s", alertUid), nil)
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.JSONResult(result), nil
		},
	)

	srv.AddTool(
		mcp.NewTool("datto_resolve_alert",
			mcp.WithDescription("Resolve a Datto RMM alert by its UID."),
			mcp.WithString("alertUid",
				mcp.Description("The UID of the alert to resolve."),
				mcp.Required(),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			alertUid := req.GetString("alertUid", "")
			if alertUid == "" {
				return mcputil.ErrorResult(fmt.Errorf("alertUid is required")), nil
			}
			result, err := c.Post(ctx, fmt.Sprintf("/alert/%s/resolve", alertUid), nil)
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.JSONResult(result), nil
		},
	)
}
