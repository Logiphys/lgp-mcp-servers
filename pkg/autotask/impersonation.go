package autotask

import (
	"context"
	"strconv"

	"github.com/Logiphys/lgp-mcp-servers/pkg/apihelper"
	"github.com/Logiphys/lgp-mcp-servers/pkg/mcputil"
)

// ImpersonationHeader is the Autotask REST header that attributes a create or
// update to the given resource instead of the API user.
const ImpersonationHeader = "ImpersonationResourceId"

// writeContext returns ctx with the impersonation header set if the gateway
// actor in ctx carries an Autotask resource ID. Only write operations use it;
// reads always run as the API user.
func writeContext(ctx context.Context) context.Context {
	a, ok := mcputil.ActorFromContext(ctx)
	if !ok || !a.HasAutotaskResourceID() {
		return ctx
	}
	return apihelper.WithRequestHeaders(ctx, map[string]string{
		ImpersonationHeader: strconv.Itoa(a.AutotaskResourceID),
	})
}
