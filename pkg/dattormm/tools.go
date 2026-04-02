package dattormm

import (
	"log/slog"

	"github.com/mark3labs/mcp-go/server"
)

func RegisterTools(srv *server.MCPServer, client *Client, logger *slog.Logger, tier int) {
	registerAccountTools(srv, client, logger, tier)
	registerSiteTools(srv, client, logger, tier)
	registerDeviceTools(srv, client, logger, tier)
	registerAlertTools(srv, client, logger, tier)
	registerJobTools(srv, client, logger, tier)
	registerAuditTools(srv, client, logger, tier)
	registerFilterTools(srv, client, logger)
	registerVariableTools(srv, client, logger, tier)
	registerSystemTools(srv, client, logger)
	registerActivityTools(srv, client, logger, tier)
}
