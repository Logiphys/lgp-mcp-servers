package mcputil

import (
	"context"
	"runtime"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ServerInfo holds metadata about an MCP server binary.
type ServerInfo struct {
	Name      string // e.g. "autotask-mcp"
	Version   string // set via ldflags or "dev"
	BuildDate string // set via ldflags, e.g. "2026-04-02T09:30:00Z"
	Prefix    string // tool name prefix, e.g. "autotask"
}

// RegisterServerInfoTool adds a <prefix>_server_info tool to the MCP server.
func RegisterServerInfoTool(srv *server.MCPServer, info ServerInfo) {
	toolName := info.Prefix + "_server_info"

	srv.AddTool(
		mcp.NewTool(toolName,
			mcp.WithDescription("Returns version, build, and developer information for this MCP server"),
			mcp.WithReadOnlyHintAnnotation(true),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			buildDate := info.BuildDate
			if buildDate == "" {
				buildDate = "dev"
			}
			return JSONResult(map[string]string{
				"server":     info.Name,
				"version":    info.Version,
				"build_date": buildDate,
				"developer":  "Logiphys Datensysteme GmbH",
				"website":    "https://logiphys.de",
				"runtime":    runtime.Version(),
				"os":         runtime.GOOS,
				"arch":       runtime.GOARCH,
			}), nil
		},
	)
}
