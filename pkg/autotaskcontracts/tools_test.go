package autotaskcontracts

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Logiphys/lgp-mcp-servers/pkg/autotask"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func callMonth(t *testing.T, handler http.HandlerFunc, args string) (text string, failed bool) {
	t.Helper()
	api := httptest.NewServer(handler)
	t.Cleanup(api.Close)
	c := autotask.NewClient(autotask.Config{Username: "test", BaseURL: api.URL}, slog.Default())
	srv := server.NewMCPServer("contracts", "test")
	RegisterTools(srv, c)
	if len(srv.ListTools()) != 1 || srv.GetTool("get_service_month") == nil {
		t.Fatal("unexpected business tools")
	}
	req := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"get_service_month","arguments":` + args + `}}`
	resp := srv.HandleMessage(t.Context(), []byte(req))
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatal(err)
	}
	var envelope struct {
		Result *mcp.CallToolResult `json:"result"`
		Error  *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Error != nil {
		return envelope.Error.Message, true
	}
	if envelope.Result == nil || len(envelope.Result.Content) != 1 {
		t.Fatalf("invalid MCP result %s", data)
	}
	return envelope.Result.Content[0].(mcp.TextContent).Text, envelope.Result.IsError
}

const aprilArgs = `{"contractID":7,"month":"2026-04"}`

func units(line, item, count int, start, end string, bundle bool) map[string]any {
	row := map[string]any{"contractID": 7, "units": count, "startDate": start, "endDate": end}
	if bundle {
		row["contractServiceBundleID"] = line
		row["serviceBundleID"] = item
	} else {
		row["contractServiceID"] = line
		row["serviceID"] = item
	}
	return row
}

func page(w http.ResponseWriter, items []map[string]any, next any) {
	if items == nil {
		items = []map[string]any{}
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"items": items, "pageDetails": map[string]any{"nextPageUrl": next}})
}

func TestMonthThroughMCP(t *testing.T) {
	periodFilters := []autotask.Filter{
		{Op: "eq", Field: "contractID", Value: float64(7)},
		{Op: "gte", Field: "endDate", Value: "2026-04-01T00:00:00Z"},
		{Op: "lt", Field: "startDate", Value: "2026-05-01T00:00:00Z"},
	}
	raw, failed := callMonth(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("ImpersonationResourceId") != "" || r.Header.Get("UserName") != "test" {
			t.Error("read did not use the configured API user")
		}
		if r.URL.Path == "/Contracts/7" && r.Method == "GET" {
			fmt.Fprint(w, `{"item":{"id":7,"description":"private-contract-field"}}`)
			return
		}
		if r.Method != "POST" {
			t.Errorf("unexpected %s %s", r.Method, r.URL)
			http.NotFound(w, r)
			return
		}
		var body struct {
			Filter     []autotask.Filter
			MaxRecords int
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body.Filter) != 1 || body.Filter[0].Op != "and" || body.MaxRecords != 500 {
			t.Error("bad query")
			w.WriteHeader(400)
			return
		}
		entity := strings.Split(strings.TrimPrefix(r.URL.Path, "/"), "/")[0]
		want := periodFilters
		switch entity {
		case "ServiceBundleServices":
			want = []autotask.Filter{{Op: "in", Field: "serviceBundleID", Value: []any{float64(20)}}}
		case "ServiceBundles":
			want = []autotask.Filter{{Op: "in", Field: "id", Value: []any{float64(20)}}}
		case "Services":
			want = []autotask.Filter{{Op: "in", Field: "id", Value: []any{float64(8), float64(9)}}}
		}
		if !reflect.DeepEqual(body.Filter[0].Items, want) {
			t.Errorf("wrong %s filters %+v", entity, body.Filter)
		}
		next := "http://" + r.Host + "/" + entity + "/query/next?page=2"
		first := units(11, 8, 2, "2026-03-25", "2026-04-10", false)
		switch r.URL.RequestURI() {
		case "/ContractServiceUnits/query":
			page(w, []map[string]any{units(12, 8, 1, "2026-04-25", "2026-05-15", false), first}, next)
		case "/ContractServiceUnits/query/next?page=2":
			page(w, []map[string]any{units(11, 8, 4, "2026-04-10T02:00:00+02:00", "2026-05-01", false), first}, nil)
		case "/ContractServiceBundleUnits/query":
			page(w, []map[string]any{units(31, 20, 5, "2026-04-15", "2026-04-20", true), units(31, 20, 3, "2026-04-01", "2026-04-14", true)}, nil)
		case "/ServiceBundleServices/query":
			page(w, []map[string]any{{"serviceBundleID": 20, "serviceID": 9}}, next)
		case "/ServiceBundleServices/query/next?page=2":
			page(w, []map[string]any{{"serviceBundleID": 20, "serviceID": 8}}, nil)
		case "/Services/query":
			page(w, []map[string]any{{"id": 9, "name": "C", "isActive": false}, {"id": 8, "name": "A", "unitCost": "private-cost"}}, nil)
		case "/ServiceBundles/query":
			page(w, []map[string]any{{"id": 20, "name": "Copilot"}}, nil)
		default:
			t.Errorf("unexpected URL %s", r.URL)
			http.NotFound(w, r)
		}
	}, aprilArgs)
	if failed || strings.Contains(raw, "private-") {
		t.Fatalf("unexpected result: %s", raw)
	}
	var result monthReport
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		t.Fatal(err)
	}
	if result.ContractID != 7 || result.Month != "2026-04" || len(result.Periods) != 5 || result.CatalogNote == "" {
		t.Fatalf("wrong monthly report %s", raw)
	}
	if _, err := time.Parse(time.RFC3339, result.CatalogReadAt); err != nil {
		t.Fatal(err)
	}
	for i, want := range []struct {
		start, end string
		direct     int64
		bundle     int
	}{
		{"01", "09", 2, 3}, {"10", "14", 4, 3}, {"15", "20", 4, 5}, {"21", "24", 4, 0}, {"25", "30", 5, 0},
	} {
		p := result.Periods[i]
		if p.StartDate != "2026-04-"+want.start || p.EndDate != "2026-04-"+want.end || len(p.Services) != 1 {
			t.Fatalf("interval %d: %+v", i, p)
		}
		s := p.Services[0]
		if s.ServiceID != 8 || s.Name != "A" || s.DirectUnits != want.direct || s.DirectLines[0].ContractServiceID != 11 {
			t.Fatalf("direct service: %+v", s)
		}
		if want.bundle != 0 {
			if p.ServiceTotalsComplete || s.TotalUnits != nil || !reflect.DeepEqual(p.Bundles, []bundleQuantity{{31, 20, "Copilot", want.bundle}}) ||
				!reflect.DeepEqual(p.CurrentCatalogEstimates, []catalogEstimate{{31, 8, "A", want.bundle}, {31, 9, "C", want.bundle}}) {
				t.Fatalf("historical and current quantities: %+v", p)
			}
		} else if !p.ServiceTotalsComplete || s.TotalUnits == nil || *s.TotalUnits != want.direct || len(p.Bundles)+len(p.CurrentCatalogEstimates) != 0 {
			t.Fatalf("direct total: %+v", p)
		}
	}
	if got := result.Periods[4].Services[0].DirectLines; !reflect.DeepEqual(got, []directLine{{11, 4}, {12, 1}}) {
		t.Fatalf("separate positions: %v", got)
	}
}

func TestMonthCalendarAndConflicts(t *testing.T) {
	for _, tc := range []struct {
		month, start, end string
		count             int
	}{
		{"2024-02", "2024-01-15", "2024-03-31", 29}, {"2026-12", "2026-12-31", "2027-01-01", 1},
		{"2026-04", "2026-03-01", "2026-03-31", 0}, {"2026-04", "2026-05-01", "2026-05-15", 0},
	} {
		t.Run(tc.month+tc.start, func(t *testing.T) {
			month, _ := time.Parse("2006-01", tc.month)
			p, err := parsePeriod(units(1, 2, 1, tc.start, tc.end, false), 7, "contractServiceID", "serviceID")
			if err != nil {
				t.Fatal(err)
			}
			days, err := expandPeriods(month, []unitPeriod{p})
			if err != nil {
				t.Fatal(err)
			}
			n := 0
			for _, day := range days {
				n += len(day)
			}
			if n != tc.count {
				t.Fatalf("got %d active days, want %d", n, tc.count)
			}
		})
	}
	month, _ := time.Parse("2006-01", "2026-04")
	one := unitPeriod{booking: booking{1, 2, 1}, start: month, end: month.AddDate(0, 0, 5)}
	for _, other := range []unitPeriod{
		{booking: booking{1, 2, 2}, start: one.start, end: one.end},
		{booking: one.booking, start: month.AddDate(0, 0, 1), end: one.end},
	} {
		if _, err := expandPeriods(month, []unitPeriod{other, one}); err == nil {
			t.Fatal("accepted ambiguous periods")
		}
	}
}

func TestMonthInvalidInputAndEmptyContract(t *testing.T) {
	for _, args := range []string{`{}`, `{"contractID":0,"month":"2026-04"}`, `{"contractID":1.5,"month":"2026-04"}`, `{"contractID":7,"month":"2026-13"}`, `{"contractID":7,"month":"2026-04","companyID":1}`} {
		_, failed := callMonth(t, func(http.ResponseWriter, *http.Request) { t.Error("invalid input reached the provider") }, args)
		if !failed {
			t.Errorf("accepted %s", args)
		}
	}
	raw, failed := callMonth(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/Contracts/7":
			fmt.Fprint(w, `{"item":{"id":7}}`)
		case "/ContractServiceUnits/query", "/ContractServiceBundleUnits/query":
			page(w, nil, nil)
		default:
			t.Errorf("unexpected catalog read %s", r.URL.Path)
		}
	}, aprilArgs)
	var report monthReport
	if failed || json.Unmarshal([]byte(raw), &report) != nil || report.Periods == nil || len(report.Periods) > 0 || report.CatalogReadAt != "" {
		t.Fatalf("empty month: %s", raw)
	}
}

func TestMonthBundleWithoutCurrentMembers(t *testing.T) {
	raw, failed := callMonth(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/Contracts/7":
			fmt.Fprint(w, `{"item":{"id":7}}`)
		case "/ContractServiceUnits/query":
			page(w, []map[string]any{units(11, 8, 2, "2026-04-01", "2026-04-30", false)}, nil)
		case "/ContractServiceBundleUnits/query":
			page(w, []map[string]any{units(31, 20, 3, "2026-04-01", "2026-04-30", true)}, nil)
		case "/ServiceBundleServices/query":
			page(w, nil, nil)
		case "/ServiceBundles/query":
			page(w, []map[string]any{{"id": 20, "name": "Bundle"}}, nil)
		case "/Services/query":
			page(w, []map[string]any{{"id": 8, "name": "A"}}, nil)
		default:
			t.Errorf("unexpected request %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}, aprilArgs)
	var report monthReport
	if failed || json.Unmarshal([]byte(raw), &report) != nil || report.CatalogReadAt == "" || len(report.Periods) != 1 {
		t.Fatalf("expected a dated report: %s", raw)
	}
	p := report.Periods[0]
	if p.ServiceTotalsComplete || !reflect.DeepEqual(p.CurrentCatalogEstimates, []catalogEstimate{}) ||
		len(p.Bundles) != 1 || p.Bundles[0].Units != 3 || len(p.Services) != 1 || p.Services[0].DirectUnits != 2 || p.Services[0].TotalUnits != nil {
		t.Fatalf("historical quantities with an empty current catalog: %s", raw)
	}
}

func TestMonthUpstreamErrors(t *testing.T) {
	for _, tc := range []struct{ name, failedEntity string }{
		{"unknown contract", "Contracts"}, {"outside contract", "ContractServiceUnits"},
		{"units absent", "ContractServiceUnits"}, {"denied second page", "ContractServiceUnits"},
		{"unreadable members", "ServiceBundleServices"}, {"missing member name", "Services"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw, failed := callMonth(t, func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/Contracts/7":
					if tc.name == "unknown contract" {
						fmt.Fprint(w, `{"item":null}`)
						return
					}
					fmt.Fprint(w, `{"item":{"id":7}}`)
				case "/ContractServiceUnits/query":
					row := units(1, 8, 2, "2026-04-01", "2026-04-30", false)
					switch tc.name {
					case "outside contract":
						row["contractID"] = 99
					case "units absent":
						delete(row, "units")
					case "denied second page":
						page(w, []map[string]any{row}, "http://"+r.Host+r.URL.Path+"/next?page=2")
						return
					}
					page(w, []map[string]any{row}, nil)
				case "/ContractServiceUnits/query/next":
					w.WriteHeader(http.StatusForbidden)
				case "/ContractServiceBundleUnits/query":
					page(w, []map[string]any{units(12, 20, 1, "2026-04-15", "2026-04-30", true)}, nil)
				case "/ServiceBundleServices/query":
					if tc.name == "unreadable members" {
						w.WriteHeader(http.StatusForbidden)
						return
					}
					page(w, []map[string]any{{"serviceID": 42, "serviceBundleID": 20}}, nil)
				case "/ServiceBundles/query":
					page(w, []map[string]any{{"id": 20, "name": "P"}}, nil)
				case "/Services/query":
					page(w, []map[string]any{{"id": 8, "name": "A"}}, nil)
				default:
					t.Errorf("unexpected request %s", r.URL.Path)
					http.NotFound(w, r)
				}
			}, aprilArgs)
			if !failed || !strings.Contains(raw, tc.failedEntity) || strings.Contains(raw, `"periods"`) {
				t.Errorf("expected %s error, got %s", tc.failedEntity, raw)
			}
		})
	}
}
