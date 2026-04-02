package mcputil

import (
	"context"
	"runtime"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ServerInfo holds metadata about an MCP server binary.
type ServerInfo struct {
	Name       string // e.g. "autotask-mcp"
	Version    string // set via ldflags or "dev"
	BuildDate  string // set via ldflags, e.g. "2026-04-02T09:30:00Z"
	Prefix     string // tool name prefix, e.g. "autotask"
	AccessTier int    // 1=Safe Read-Only, 2=Read + Sensitive Data, 3=Full Access
}

// tierDescription returns a human-readable label for the given access tier.
func tierDescription(tier int) string {
	switch tier {
	case 2:
		return "Read + Sensitive Data"
	case 3:
		return "Full Access"
	default:
		return "Safe Read-Only"
	}
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
			return JSONResult(map[string]any{
				"server":           info.Name,
				"version":          info.Version,
				"build_date":       buildDate,
				"developer":        "Logiphys Datensysteme GmbH",
				"website":          "https://logiphys.de",
				"runtime":          runtime.Version(),
				"os":               runtime.GOOS,
				"arch":             runtime.GOARCH,
				"access_tier":      info.AccessTier,
				"tier_description": tierDescription(info.AccessTier),
			}), nil
		},
	)
}
