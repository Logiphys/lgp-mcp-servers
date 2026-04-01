package dattormm

import (
	"log/slog"

	"github.com/mark3labs/mcp-go/server"
)

func RegisterTools(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	registerAccountTools(srv, client, logger)
	registerSiteTools(srv, client, logger)
	registerDeviceTools(srv, client, logger)
	registerAlertTools(srv, client, logger)
	registerJobTools(srv, client, logger)
	registerAuditTools(srv, client, logger)
	registerFilterTools(srv, client, logger)
	registerVariableTools(srv, client, logger)
	registerSystemTools(srv, client, logger)
	registerActivityTools(srv, client, logger)
}
