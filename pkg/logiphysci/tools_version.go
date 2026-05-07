package logiphysci

import (
	"context"
	"log/slog"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/Logiphys/lgp-mcp-servers/pkg/mcputil"
)

// Build-time variables wired through ldflags.
//
// Example Make recipe:
//
//	-X 'github.com/Logiphys/lgp-mcp-servers/pkg/logiphysci.SkillSHA=$(SKILL_SHA)' \
//	-X 'github.com/Logiphys/lgp-mcp-servers/pkg/logiphysci.SkillTag=$(SKILL_TAG)' \
//	-X 'github.com/Logiphys/lgp-mcp-servers/pkg/logiphysci.LoadedAt=$(LOADED_AT)'
var (
	SkillSHA = "unknown"
	SkillTag = "none"
	LoadedAt = "unknown"
)

func registerVersionTool(srv *server.MCPServer, _ *slog.Logger) {
	srv.AddTool(
		mcp.NewTool("version",
			mcp.WithDescription("Returns the loaded skill marketplace SHA, tag, and load timestamp — used for audit and compliance trails."),
			mcp.WithReadOnlyHintAnnotation(true),
		),
		func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return mcputil.JSONResult(map[string]any{
				"server":                "logiphys-ci-mcp",
				"skill_marketplace_sha": SkillSHA,
				"skill_marketplace_tag": SkillTag,
				"loaded_at":             LoadedAt,
			}), nil
		},
	)
}
