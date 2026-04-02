package autotask

import (
	"log/slog"

	"github.com/mark3labs/mcp-go/server"
)

// RegisterTools registers all Autotask MCP tools on the given server.
func RegisterTools(srv *server.MCPServer, client *Client, picklist *PicklistCache, logger *slog.Logger, tier int) {
	registerUtilityTools(srv, client, picklist, logger)
	registerCompanyTools(srv, client, logger, tier)
	registerContactTools(srv, client, logger, tier)
	registerTicketTools(srv, client, picklist, logger, tier)
	registerProjectTools(srv, client, logger, tier)
	registerFinancialTools(srv, client, logger, tier)
	registerTimeBillingTools(srv, client, logger, tier)
	registerServiceCallTools(srv, client, logger, tier)
	registerProductTools(srv, client, logger)
	registerMetaTools(srv, client, logger)
}
