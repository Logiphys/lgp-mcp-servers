package itglue

import (
	"log/slog"

	"github.com/mark3labs/mcp-go/server"
)

// RegisterTools registers all IT Glue tools with the MCP server.
func RegisterTools(srv *server.MCPServer, client *Client, logger *slog.Logger, tier int) {
	// Tier 1 — Safe Read-Only (always registered)
	registerOrganizationTools(srv, client, logger)
	registerConfigurationTools(srv, client, logger)
	registerDocumentReadTools(srv, client, logger)
	registerFlexibleAssetTools(srv, client, logger)
	registerHealthTools(srv, client, logger)
	registerLocationTools(srv, client, logger)
	registerMetadataTools(srv, client, logger)
	registerDomainTools(srv, client, logger)
	registerExpirationTools(srv, client, logger)
	registerConfigurationInterfaceTools(srv, client, logger)

	// Tier 2 — Sensitive Data
	if tier >= 2 {
		registerContactTools(srv, client, logger)
		registerPasswordTools(srv, client, logger)
	}

	// Tier 3 — Write Operations
	if tier >= 3 {
		registerDocumentWriteTools(srv, client, logger)
	}
}
