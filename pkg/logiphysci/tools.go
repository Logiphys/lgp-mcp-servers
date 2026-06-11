package logiphysci

import (
	"log/slog"

	"github.com/mark3labs/mcp-go/server"
)

// RegisterTools registers all logiphys-ci MCP tools on the given server.
func RegisterTools(srv *server.MCPServer, runner BuildRunner, logger *slog.Logger) {
	registerBriefTool(srv, runner, logger)
	registerAngebotTool(srv, runner, logger)
	registerBerichtTool(srv, runner, logger)
	registerKonzeptTool(srv, runner, logger)
	registerLieferscheinTool(srv, runner, logger)
	registerMahnungTool(srv, runner, logger)
	registerVersionTool(srv, logger)
}
