//go:build pilot

package pilot_test

import (
	"context"
	"encoding/json"
	"maps"
	"net"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
)

// TestPilot reads only explicitly supplied test fixtures. Keep product data and
// credentials out of test output, including when the upstream rejects a call.
func TestPilot(t *testing.T) {
	if binary := os.Getenv("LGP_PILOT_GATEWAY_BIN"); binary != "" {
		startLocalFixture(t, binary)
	}
	var f struct {
		DeviceUID, DeniedDeviceUID, PatchID, InstallStatus, AsOf string
		PatchPage                                                *int
		ContractID, LineID                                       int
		Quantity, UnitPrice, PeriodPrice                         *float64
	}
	data, err := os.ReadFile(os.Getenv("LGP_PILOT_FIXTURE"))
	if err != nil || json.Unmarshal(data, &f) != nil {
		t.Fatal("could not read LGP_PILOT_FIXTURE")
	}
	if f.DeviceUID == "" || f.DeniedDeviceUID == "" || f.DeviceUID == f.DeniedDeviceUID || f.PatchID == "" || f.InstallStatus == "" {
		t.Fatal("provide the allowed device, denied device and expected patch status")
	}
	if f.ContractID <= 0 || f.LineID <= 0 || f.AsOf == "" || f.Quantity == nil || f.UnitPrice == nil || f.PeriodPrice == nil {
		t.Fatal("provide a contract line, date and known portal prices and units")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 90*time.Second)
	defer cancel()
	allowed := pilotClient(t, ctx, "LGP_PILOT_ALLOWED_KEY")
	basic := pilotClient(t, ctx, "LGP_PILOT_BASIC_KEY")
	const patches = "datto_rmm_list_patches"
	const lines = "autotask_search_contract_services"
	const dispatch = "autotask_execute_tool"
	for _, role := range []struct {
		c       *client.Client
		visible bool
	}{{allowed, true}, {basic, false}} {
		list, err := role.c.ListTools(ctx, mcp.ListToolsRequest{})
		if err != nil {
			t.Fatal("could not list gateway tools")
		}
		for _, name := range []string{patches, lines} {
			visible := slices.ContainsFunc(list.Tools, func(v mcp.Tool) bool { return v.Name == name })
			if visible != role.visible {
				t.Fatalf("unexpected role visibility for %s", name)
			}
		}
	}
	patchArgs := map[string]any{"deviceUid": f.DeviceUID, "installStatus": f.InstallStatus}
	if f.PatchPage != nil {
		patchArgs["page"] = *f.PatchPage
	}
	lineArgs := map[string]any{"contractID": f.ContractID, "id": f.LineID, "asOf": f.AsOf}
	var patchPage struct {
		Items []struct{ PatchID, InstallStatus string }
	}
	res, err := call(ctx, allowed, patches, patchArgs)
	readResult(t, patches, res, err, &patchPage)
	if !slices.ContainsFunc(patchPage.Items, func(p struct{ PatchID, InstallStatus string }) bool {
		return p.PatchID == f.PatchID && p.InstallStatus == f.InstallStatus
	}) {
		t.Fatal("known patch/status differs from the portal fixture")
	}
	type contractPage struct {
		AsOf  string
		Items []struct {
			ID, ContractID      int
			Quantity, UnitPrice *float64
			Period              *struct{ Price *float64 }
		}
	}
	var contracts contractPage
	res, err = call(ctx, allowed, lines, lineArgs)
	readResult(t, lines, res, err, &contracts)
	if contracts.AsOf != f.AsOf || len(contracts.Items) != 1 {
		t.Fatal("expected exactly one line at the reference date")
	}
	row := contracts.Items[0]
	if row.ID != f.LineID || row.ContractID != f.ContractID || row.Quantity == nil || *row.Quantity != *f.Quantity {
		t.Fatal("contract quantity differs from the portal fixture")
	}
	if row.UnitPrice == nil || *row.UnitPrice != *f.UnitPrice || row.Period == nil || row.Period.Price == nil || *row.Period.Price != *f.PeriodPrice {
		t.Fatal("contract values differ from the portal fixture")
	}
	meta := maps.Clone(lineArgs)
	meta["toolName"] = "search_contract_services"
	res, err = call(ctx, allowed, dispatch, meta)
	var dispatched contractPage
	readResult(t, dispatch, res, err, &dispatched)
	if !reflect.DeepEqual(dispatched, contracts) {
		t.Fatal("direct and dispatched reads differ")
	}
	denied := func(c *client.Client, name string, args map[string]any, reasons ...string) {
		t.Helper()
		result, callErr := call(ctx, c, name, args)
		var message string
		switch {
		case callErr != nil:
			message = callErr.Error()
		case result != nil && result.IsError && len(result.Content) > 0:
			if body, ok := result.Content[0].(mcp.TextContent); ok {
				message = body.Text
			}
		}
		if !slices.ContainsFunc(reasons, func(s string) bool { return strings.Contains(strings.ToLower(message), s) }) {
			t.Fatalf("missing expected denial for %s", name)
		}
	}
	denied(basic, patches, patchArgs, "not found", "unknown tool", "not available", "not allowed", "access denied")
	denied(basic, lines, lineArgs, "not found", "unknown tool", "not available", "not allowed", "access denied")
	denied(basic, dispatch, meta, "nicht freigegeben", "not allowed", "access denied")
	denied(allowed, patches, map[string]any{"deviceUid": f.DeniedDeviceUID}, "http 403", "http 404")
}

func pilotClient(t *testing.T, ctx context.Context, keyName string) *client.Client {
	t.Helper()
	endpoint := os.Getenv("LGP_PILOT_GATEWAY_URL")
	u, err := url.Parse(endpoint)
	if err != nil || u.Host == "" || u.User != nil {
		t.Fatal("invalid test gateway URL")
	}
	localHTTP := u.Scheme == "http" && (u.Hostname() == "localhost" || net.ParseIP(u.Hostname()).IsLoopback())
	if u.Scheme != "https" && !localHTTP {
		t.Fatal("expected HTTPS test gateway (or loopback HTTP)")
	}
	key := os.Getenv(keyName)
	if key == "" {
		t.Fatalf("%s missing", keyName)
	}
	c, err := client.NewStreamableHttpClient(endpoint, transport.WithHTTPHeaders(map[string]string{"Authorization": "Bearer " + key}),
		transport.WithHTTPBasicClient(&http.Client{CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}))
	if err != nil {
		t.Fatal("could not create test client")
	}
	if err = c.Start(ctx); err != nil {
		t.Fatal("could not start test client")
	}
	t.Cleanup(func() { _ = c.Close() })
	init := mcp.InitializeRequest{}
	init.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	init.Params.ClientInfo = mcp.Implementation{Name: "lgp-pilot-test", Version: "1"}
	if _, err = c.Initialize(ctx, init); err != nil {
		t.Fatalf("test gateway rejected initialization for %s", keyName)
	}
	return c
}

func call(ctx context.Context, c *client.Client, name string, args map[string]any) (*mcp.CallToolResult, error) {
	req := mcp.CallToolRequest{}
	req.Params.Name = name
	req.Params.Arguments = args
	return c.CallTool(ctx, req)
}

func readResult(t *testing.T, name string, result *mcp.CallToolResult, err error, dest any) {
	t.Helper()
	if err != nil || result == nil || result.IsError || len(result.Content) == 0 {
		t.Fatalf("permitted call to %s failed", name)
	}
	body, ok := result.Content[0].(mcp.TextContent)
	if !ok || json.Unmarshal([]byte(body.Text), dest) != nil {
		t.Fatalf("unexpected %s result format", name)
	}
}
