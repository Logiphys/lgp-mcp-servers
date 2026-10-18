package autotask

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/Logiphys/lgp-mcp-servers/pkg/mcputil"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// recordedRequest is one HTTP request seen by the fake Autotask API.
type recordedRequest struct {
	method        string
	path          string
	impersonation string
}

// fakeAutotask records requests and answers with minimal valid payloads.
type fakeAutotask struct {
	mu       sync.Mutex
	requests []recordedRequest
}

func (f *fakeAutotask) handler(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	f.requests = append(f.requests, recordedRequest{
		method:        r.Method,
		path:          r.URL.Path,
		impersonation: r.Header.Get(ImpersonationHeader),
	})
	f.mu.Unlock()

	switch {
	case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/query"):
		_ = json.NewEncoder(w).Encode(map[string]any{"items": []any{}})
	case r.Method == http.MethodPost:
		_ = json.NewEncoder(w).Encode(map[string]any{"itemId": 555})
	case strings.HasSuffix(r.URL.Path, "/entityInformation/fields"):
		_ = json.NewEncoder(w).Encode(map[string]any{"fields": []any{}})
	case r.Method == http.MethodGet:
		_ = json.NewEncoder(w).Encode(map[string]any{"item": map[string]any{"id": 1}})
	default:
		_ = json.NewEncoder(w).Encode(map[string]any{})
	}
}

func (f *fakeAutotask) snapshot() []recordedRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]recordedRequest(nil), f.requests...)
}

func TestClientImpersonationHeader(t *testing.T) {
	ops := []struct {
		name  string
		write bool
		call  func(ctx context.Context, c *Client) error
	}{
		{"Get", false, func(ctx context.Context, c *Client) error { _, err := c.Get(ctx, "Tickets", 1); return err }},
		{"Query", false, func(ctx context.Context, c *Client) error {
			_, err := c.Query(ctx, "Tickets", nil, QueryOpts{})
			return err
		}},
		{"GetFieldInfo", false, func(ctx context.Context, c *Client) error { _, err := c.GetFieldInfo(ctx, "Tickets"); return err }},
		{"TestConnection", false, func(ctx context.Context, c *Client) error { return c.TestConnection(ctx) }},
		{"Create", true, func(ctx context.Context, c *Client) error {
			_, err := c.Create(ctx, "Tickets", map[string]any{})
			return err
		}},
		{"CreateChild", true, func(ctx context.Context, c *Client) error {
			_, err := c.CreateChild(ctx, "Tickets", 1, "Notes", map[string]any{})
			return err
		}},
		{"Update", true, func(ctx context.Context, c *Client) error { return c.Update(ctx, "Tickets", 1, map[string]any{}) }},
		{"Delete", true, func(ctx context.Context, c *Client) error { return c.Delete(ctx, "ServiceCalls", 1) }},
		{"DeleteChild", true, func(ctx context.Context, c *Client) error { return c.DeleteChild(ctx, "Tickets", 1, "Charges", 2) }},
	}

	ctxs := []struct {
		name       string
		ctx        context.Context
		impersonID string // expected header value on writes
	}{
		{"no actor", context.Background(), ""},
		{"actor without resource id", mcputil.WithActor(context.Background(), mcputil.Actor{User: "a@example.com", Channel: "connector"}), ""},
		{"actor with resource id", mcputil.WithActor(context.Background(), mcputil.Actor{User: "a@example.com", Channel: "connector", AutotaskResourceID: 29682885}), "29682885"},
	}

	for _, op := range ops {
		for _, cc := range ctxs {
			t.Run(op.name+"/"+cc.name, func(t *testing.T) {
				fake := &fakeAutotask{}
				client, _ := setupTestServer(t, fake.handler)
				if err := op.call(cc.ctx, client); err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				reqs := fake.snapshot()
				if len(reqs) == 0 {
					t.Fatal("no request recorded")
				}
				want := ""
				if op.write {
					want = cc.impersonID
				}
				for _, r := range reqs {
					if r.impersonation != want {
						t.Errorf("%s %s: %s = %q, want %q", r.method, r.path, ImpersonationHeader, r.impersonation, want)
					}
				}
			})
		}
	}
}

// callTool sends a JSON-RPC tools/call through the MCP server, like the
// gateway does over stdio.
func callTool(t *testing.T, srv *server.MCPServer, name string, args map[string]any, meta map[string]any) *mcp.CallToolResult {
	t.Helper()
	params := map[string]any{"name": name, "arguments": args}
	if meta != nil {
		params["_meta"] = meta
	}
	msg, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": params})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	resp := srv.HandleMessage(context.Background(), msg)
	b, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}
	var envelope struct {
		Result *mcp.CallToolResult `json:"result"`
		Error  any                 `json:"error"`
	}
	if err := json.Unmarshal(b, &envelope); err != nil {
		t.Fatalf("unmarshal response: %v (%s)", err, b)
	}
	if envelope.Error != nil || envelope.Result == nil {
		t.Fatalf("tools/call %s failed: %s", name, b)
	}
	return envelope.Result
}

func resultText(t *testing.T, r *mcp.CallToolResult) string {
	t.Helper()
	if len(r.Content) == 0 {
		t.Fatal("empty result")
	}
	tc, ok := r.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("unexpected content type %T", r.Content[0])
	}
	return tc.Text
}

func TestToolsImpersonation(t *testing.T) {
	actorMeta := map[string]any{
		mcputil.ActorMetaKey: map[string]any{"user": "alice@example.com", "channel": "connector", "autotask_resource_id": 4711},
	}
	actorMetaNoID := map[string]any{
		mcputil.ActorMetaKey: map[string]any{"user": "alice@example.com", "channel": "connector"},
	}
	createTicket := map[string]any{"companyID": 1, "title": "t", "description": "d"}

	tests := []struct {
		name string
		tool string
		args map[string]any
		meta map[string]any
		// wantWrite is the method+path of the write request, empty for read tools.
		wantWrite string
		wantID    string // expected header value on the write request
	}{
		{"create without meta", "create_ticket", createTicket, nil, "POST /Tickets", ""},
		{"create with actor without id", "create_ticket", createTicket, actorMetaNoID, "POST /Tickets", ""},
		{"create with actor", "create_ticket", createTicket, actorMeta, "POST /Tickets", "4711"},
		{"update with actor", "update_ticket", map[string]any{"ticketId": 9, "status": 5}, actorMeta, "PATCH /Tickets", "4711"},
		{"child create with actor", "create_ticket_note", map[string]any{"ticketId": 9, "title": "n", "description": "d", "noteType": 1, "publish": 1}, actorMeta, "POST /Tickets/9/Notes", "4711"},
		{"delete with actor", "delete_service_call", map[string]any{"id": 3}, actorMeta, "DELETE /ServiceCalls/3", "4711"},
		{"read with actor", "search_tickets", map[string]any{"searchTerm": "x"}, actorMeta, "", ""},
		{"execute_tool create with actor", "execute_tool", map[string]any{"toolName": "create_ticket", "companyID": 1, "title": "t", "description": "d"}, actorMeta, "POST /Tickets", "4711"},
		{"execute_tool read with actor", "execute_tool", map[string]any{"toolName": "get_ticket_details", "ticketID": 9}, actorMeta, "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeAutotask{}
			client, _ := setupTestServer(t, fake.handler)
			srv := server.NewMCPServer("autotask-test", "test")
			RegisterTools(srv, client, NewPicklistCache(client, testLogger()), testLogger())

			res := callTool(t, srv, tt.tool, tt.args, tt.meta)
			if res.IsError {
				t.Fatalf("tool returned error: %s", resultText(t, res))
			}

			sawWrite := false
			for _, r := range fake.snapshot() {
				key := r.method + " " + r.path
				isWrite := r.method != http.MethodGet && !strings.HasSuffix(r.path, "/query")
				if isWrite {
					if key != tt.wantWrite {
						t.Errorf("unexpected write %s (want %q)", key, tt.wantWrite)
					}
					sawWrite = true
					if r.impersonation != tt.wantID {
						t.Errorf("%s: %s = %q, want %q", key, ImpersonationHeader, r.impersonation, tt.wantID)
					}
					continue
				}
				if r.impersonation != "" {
					t.Errorf("read %s must not impersonate, got %q", key, r.impersonation)
				}
			}
			if tt.wantWrite != "" && !sawWrite {
				t.Errorf("expected write %s, none recorded", tt.wantWrite)
			}
		})
	}
}

// TestCreateToolsReturnItemID checks that every create tool reports the new
// object's ID as itemId, so the gateway audit can record it.
func TestCreateToolsReturnItemID(t *testing.T) {
	var createTools []string
	for _, tools := range ToolCategories {
		for _, name := range tools {
			if strings.HasPrefix(name, "create_") {
				createTools = append(createTools, name)
			}
		}
	}
	if len(createTools) < 19 {
		t.Fatalf("expected at least 19 create tools, found %d", len(createTools))
	}

	// Generic arguments that satisfy the required parameters of all create tools.
	args := map[string]any{
		"companyID": 1, "companyId": 1, "ticketId": 9, "ticketID": 9, "projectId": 2, "projectID": 2,
		"quoteId": 3, "quoteID": 3, "serviceCallID": 4, "serviceCallTicketID": 5, "expenseReportID": 6,
		"resourceId": 7, "resourceID": 7, "contactID": 8, "opportunityId": 10, "opportunityID": 10,
		"title": "t", "name": "n", "description": "d", "companyName": "c", "firstName": "f", "lastName": "l",
		"emailAddress": "e@example.com", "companyType": 1, "phone": "1", "ownerResourceID": 7,
		"noteType": 1, "publish": 1, "status": 1, "startDate": "2026-01-01", "endDate": "2026-01-02",
		"startDateTime": "2026-01-01T08:00:00Z", "endDateTime": "2026-01-01T09:00:00Z",
		"dateWorked": "2026-01-01", "hoursWorked": 1, "summaryNotes": "s", "taskID": 11,
		"projectType": 1, "taskType": 1, "quantity": 1, "unitPrice": 1, "billingCodeID": 12,
		"chargeType": 1, "expenseDate": "2026-01-01", "expenseCategory": 1, "amount": 1,
		"weekEnding": "2026-01-03", "submitterID": 7, "roleID": 13, "expectedCloseDate": "2026-02-01",
		"probability": 50, "stage": 1, "rating": 1, "projectedRevenue": 1,
		"isNonBillable": false, "productID": 14, "serviceID": 15, "isOptional": false,
		"datePurchased": "2026-01-01", "unitQuantity": 1, "unitCost": 1, "chargeName": "c",
	}

	fake := &fakeAutotask{}
	client, _ := setupTestServer(t, fake.handler)
	srv := server.NewMCPServer("autotask-test", "test")
	RegisterTools(srv, client, NewPicklistCache(client, testLogger()), testLogger())

	for _, name := range createTools {
		t.Run(name, func(t *testing.T) {
			res := callTool(t, srv, name, args, nil)
			text := resultText(t, res)
			if res.IsError {
				t.Fatalf("tool returned error: %s", text)
			}
			var out map[string]any
			if err := json.Unmarshal([]byte(text), &out); err != nil {
				t.Fatalf("result is not JSON: %v (%s)", err, text)
			}
			if got := fmt.Sprint(out["itemId"]); got != "555" {
				t.Errorf("itemId = %v, want 555 (result: %s)", out["itemId"], text)
			}
		})
	}
}
