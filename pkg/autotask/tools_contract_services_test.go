package autotask

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func TestContractServicesAtDate(t *testing.T) {
	const prefix = "/ATServicesRest/V1.0/"
	rows := []map[string]any{
		{"id": 11, "contractID": 7, "serviceID": 8, "unitPrice": 9.99, "internalCurrencyUnitPrice": 8.8, "unitCost": 3.0},
		{"id": 12, "contractID": 7, "serviceID": 8, "unitPrice": 4.39},
		{"id": 13, "contractID": 7, "serviceID": 9, "unitPrice": 0},
	}
	period := func(id, line int, start, end string, units int, price float64) map[string]any {
		return map[string]any{"id": id, "contractServiceID": line, "contractID": 7, "serviceID": 8, "startDate": start, "endDate": end, "units": units, "price": price, "internalCurrencyPrice": price - 0.5}
	}
	var base string
	requests := 0
	_, api := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Method != "POST" || r.Header.Get(ImpersonationHeader) != "" {
			t.Errorf("unexpected method or read impersonation")
		}
		var payload struct {
			Filter     []Filter `json:"filter"`
			MaxRecords int
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Error(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if len(payload.Filter) != 1 || payload.Filter[0].Op != "and" {
			t.Errorf("bad query %+v", payload)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		filters := payload.Filter[0].Items
		var items []map[string]any
		var next any
		switch r.URL.Path {
		case prefix + "ContractServices/query":
			if payload.MaxRecords != 3 || !reflect.DeepEqual(filters, []Filter{{Op: "eq", Field: "contractID", Value: float64(7)}}) {
				t.Errorf("line query %+v", payload)
			}
			items = rows
			next = base + prefix + "ContractServices/query/next?paging=lines"
		case prefix + "ContractServices/query/next":
			if r.URL.Query().Get("paging") != "lines" {
				t.Error("wrong line cursor")
			}
			items = []map[string]any{}
		case prefix + "ContractServiceUnits/query", prefix + "ContractServiceUnits/query/next":
			want := []Filter{{Op: "in", Field: "contractServiceID", Value: []any{float64(11), float64(12), float64(13)}}, {Op: "lt", Field: "startDate", Value: "2026-04-16T00:00:00Z"}, {Op: "gte", Field: "endDate", Value: "2026-04-15T00:00:00Z"}}
			if !reflect.DeepEqual(filters, want) || payload.MaxRecords != 500 {
				t.Errorf("period query %+v", payload)
			}
			if strings.HasSuffix(r.URL.Path, "/next") {
				if r.URL.Query().Get("paging") != "units" {
					t.Error("wrong period cursor")
				}
				items = []map[string]any{period(32, 11, "2026-04-15T00:00:00", "2026-04-30", 4, 4.04)}
			} else {
				items = []map[string]any{period(31, 11, "2026-04-01", "2026-04-15", 2, 4.39), period(33, 12, "2026-04-01", "2026-04-30", 1, -2)}
				next = base + prefix + "ContractServiceUnits/query/next?paging=units"
			}
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			w.WriteHeader(404)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"items": items, "pageDetails": map[string]any{"nextPageUrl": next}})
	})
	base = api.URL
	c := NewClient(Config{BaseURL: base + strings.TrimSuffix(prefix, "/")}, testLogger())
	off := server.NewMCPServer("off", "1")
	RegisterTools(off, c, nil, testLogger())
	if off.GetTool(contractServiceTool) != nil {
		t.Fatal("opt-in tool enabled by default")
	}
	c.enableContractServices = true
	on := server.NewMCPServer("on", "1")
	RegisterTools(on, c, nil, testLogger())
	for _, srv := range []*server.MCPServer{off, on} {
		got := callTool(t, srv, "list_category_tools", map[string]any{"category": "financial"}, nil)
		if strings.Contains(got.Content[0].(mcp.TextContent).Text, contractServiceTool) {
			t.Fatal("unfiltered category catalog advertised the opt-in tool")
		}
	}
	baseArgs := map[string]any{"contractID": 7, "pageSize": 3, "asOf": "2026-04-15", "toolName": contractServiceTool}
	if res := callTool(t, off, "execute_tool", baseArgs, nil); !res.IsError || requests != 0 {
		t.Fatal("disabled meta-dispatch reached a backend")
	}
	res := callTool(t, on, "execute_tool", baseArgs, map[string]any{"logiphys.de/actor": map[string]any{"autotask_resource_id": 42}})
	if res.IsError {
		t.Fatal(res.Content)
	}
	var result struct {
		Items   []map[string]any
		AsOf    string
		Summary struct {
			NextPageURL string
			HasMore     bool
		}
	}
	if err := json.Unmarshal([]byte(res.Content[0].(mcp.TextContent).Text), &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Items) != 3 || result.AsOf != "2026-04-15" || !result.Summary.HasMore || requests != 3 {
		t.Fatalf("unexpected result %+v", result)
	}
	first, second, missing := result.Items[0], result.Items[1], result.Items[2]
	if first["quantity"] != float64(4) || first["unitPrice"] != 9.99 || first["period"].(map[string]any)["price"] != 4.04 || second["quantity"] != float64(1) || second["period"].(map[string]any)["price"] != float64(-2) || missing["period"] != nil || missing["quantity"] != nil {
		t.Fatalf("incorrect dated join %+v", result.Items)
	}
	if _, leak := first["unitCost"]; leak {
		t.Fatal("unexpected cost field")
	}
	for _, args := range []map[string]any{
		{"contractID": 7, "asOf": "2026-04-15", "pageSize": 3, "page": 2},
		{"contractID": 7, "asOf": "2026-04-15", "pageSize": 3, "nextPageUrl": result.Summary.NextPageURL},
	} {
		if got := callTool(t, on, contractServiceTool, args, nil); got.IsError || !strings.Contains(got.Content[0].(mcp.TextContent).Text, `"items": []`) {
			t.Fatalf("final page: %+v", got)
		}
	}
}

func TestContractServiceFiltersAndFailures(t *testing.T) {
	c, _ := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/ContractServices/query") {
			var p struct {
				Filter []Filter `json:"filter"`
			}
			_ = json.NewDecoder(r.Body).Decode(&p)
			want := []Filter{{Op: "in", Field: "contractID", Value: []any{float64(1), float64(2)}}, {Op: "eq", Field: "id", Value: float64(3)}, {Op: "in", Field: "serviceID", Value: []any{float64(4)}}}
			if len(p.Filter) != 1 || !reflect.DeepEqual(p.Filter[0].Items, want) {
				t.Errorf("wrong filters %v", p)
			}
			_, _ = fmt.Fprint(w, `{"items":[{"id":3,"contractID":1,"serviceID":4}],"pageDetails":{"nextPageUrl":null}}`)
		} else {
			w.WriteHeader(http.StatusForbidden)
		}
	})
	c.enableContractServices = true
	srv := server.NewMCPServer("test", "1")
	RegisterTools(srv, c, nil, testLogger())
	for _, args := range []map[string]any{
		{}, {"contractIDs": []int{}}, {"contractID": 0}, {"contractID": 1, "contractIDs": []int{1}}, {"contractID": 1, "page": -1}, {"contractID": 1, "pageSize": 501}, {"contractID": 1, "serviceIDs": []any{1.2}}, {"contractID": 1, "asOf": "2026-02-30"}, {"contractID": 1, "nextPageUrl": 17},
		{"contractIDs": []int{1, 2}, "id": 3, "serviceIDs": []int{4}},
	} {
		if got := callTool(t, srv, contractServiceTool, args, nil); !got.IsError {
			t.Fatalf("expected failure for %+v", args)
		}
	}
}

func TestContractServicePageIntegrity(t *testing.T) {
	const first = "2026-04-01"
	const later = "2026-04-15"
	for _, tc := range []struct {
		name       string
		contractID int
		starts     []string
		wantError  bool
	}{
		{"older ties superseded", 7, []string{first, first, later}, false},
		{"latest tie", 7, []string{later, first, later}, true},
		{"different contract on resumed page", 8, nil, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, api := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
				var rows []map[string]any
				if strings.HasSuffix(r.URL.Path, "/ContractServices/query/next") {
					rows = []map[string]any{{"id": 11, "contractID": tc.contractID, "serviceID": 8}}
				} else {
					if tc.contractID != 7 {
						t.Error("queried units outside the requested contract")
					}
					for i, start := range tc.starts {
						rows = append(rows, map[string]any{"id": i + 1, "contractServiceID": 11, "contractID": 7, "serviceID": 8, "startDate": start, "endDate": "2026-04-30", "units": i + 1})
					}
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"items": rows, "pageDetails": map[string]any{"nextPageUrl": nil}})
			})
			req := mcp.CallToolRequest{}
			req.Params.Arguments = map[string]any{"asOf": later, "contractID": 7, "nextPageUrl": api.URL + "/ContractServices/query/next?cursor=x"}
			got, err := c.searchContractServices(t.Context(), req)
			if err != nil || got.IsError != tc.wantError || (!got.IsError && !strings.Contains(got.Content[0].(mcp.TextContent).Text, `"quantity": 3`)) {
				t.Fatalf("unexpected result %v, err %v", got.Content, err)
			}
		})
	}
}
