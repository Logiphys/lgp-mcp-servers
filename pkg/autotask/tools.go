package autotask

import (
	"log/slog"

	"github.com/mark3labs/mcp-go/server"
)

// RegisterTools registers all Autotask MCP tools on the given server.
func RegisterTools(srv *server.MCPServer, client *Client, picklist *PicklistCache, logger *slog.Logger) {
	registerUtilityTools(srv, client, picklist, logger)
	registerCompanyTools(srv, client, logger)
	registerContactTools(srv, client, logger)
	registerTicketTools(srv, client, picklist, logger)
	registerProjectTools(srv, client, logger)
	registerFinancialTools(srv, client, logger)
	registerTimeBillingTools(srv, client, logger)
	registerServiceCallTools(srv, client, logger)
	registerProductTools(srv, client, logger)
	registerMetaTools(srv, client, logger)
}
