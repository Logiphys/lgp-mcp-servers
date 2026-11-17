package autotask

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/mark3labs/mcp-go/server"
)

type companyResult struct {
	Items   []map[string]any `json:"items"`
	Summary struct {
		Returned    int     `json:"returned"`
		Page        int     `json:"page"`
		PageSize    int     `json:"pageSize"`
		HasMore     bool    `json:"hasMore"`
		NextPageURL *string `json:"nextPageUrl"`
	} `json:"summary"`
}

func TestCompanyCursorTraversal(t *testing.T) {
	for _, pages := range [][][]int{
		{{1, 2}, {3, 4}, {5}}, {{1, 2}, {3, 4}}, {{1, 2}, {}}, {{}}, {{1, 2}, {}, {3}},
	} {
		t.Run(fmt.Sprint(pages), func(t *testing.T) {
			requests := 0
			var base, originalBody string
			_, srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
				index := requests
				requests++
				if index >= len(pages) {
					t.Errorf("extra request %s", r.URL)
					w.WriteHeader(400)
					return
				}
				if r.Method != http.MethodPost {
					t.Errorf("method = %s", r.Method)
				}
				path := "/ATServicesRest/V1.0/Companies/query"
				if index > 0 {
					path += "/next"
				}
				if r.URL.Path != path {
					t.Errorf("path = %s", r.URL.Path)
				}
				if index > 0 && r.URL.RawQuery != fmt.Sprintf("paging=%%7B%%22page%%22%%3A%d%%7D", index) {
					t.Errorf("query = %s", r.URL.RawQuery)
				}
				if r.Header.Get("Secret") != "testsecret" {
					t.Error("missing auth")
				}
				body, _ := io.ReadAll(r.Body)
				if index == 0 {
					originalBody = string(body)
				}
				if string(body) != originalBody {
					t.Error("filters changed during traversal")
				}
				if !strings.Contains(string(body), `"MaxRecords":2`) || !strings.Contains(string(body), `"field":"isActive","value":true`) {
					t.Errorf("body = %s", body)
				}
				items := []map[string]any{}
				for _, id := range pages[index] {
					items = append(items, map[string]any{"id": id, "companyName": fmt.Sprintf("Company %d", id)})
				}
				var next *string
				if index+1 < len(pages) {
					value := base + fmt.Sprintf("/atservicesrest/v1.0/Companies/query/next?paging=%%7B%%22page%%22%%3A%d%%7D", index+1)
					next = &value
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"items": items, "pageDetails": map[string]any{"nextPageUrl": next}})
			})
			base = srv.URL
			client := NewClient(Config{BaseURL: base + "/ATServicesRest/V1.0", Secret: "testsecret"}, testLogger())
			mcpServer := server.NewMCPServer("test", "1")
			registerCompanyTools(mcpServer, client, testLogger())
			args := map[string]any{"isActive": true, "pageSize": 2}
			var got, want []int
			for _, ids := range pages {
				want = append(want, ids...)
			}
			for page := 1; page <= len(pages); page++ {
				args["page"] = page
				result := callTool(t, mcpServer, "search_companies", args, nil)
				if result.IsError {
					t.Fatalf("tool error: %s", resultText(t, result))
				}
				var decoded companyResult
				raw := resultText(t, result)
				if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
					t.Fatal(err)
				}
				if decoded.Items == nil || decoded.Summary.Returned != len(pages[page-1]) || decoded.Summary.Page != page || decoded.Summary.PageSize != 2 {
					t.Fatalf("invalid envelope: %s", raw)
				}
				more := page < len(pages)
				if decoded.Summary.HasMore != more || (decoded.Summary.NextPageURL != nil) != more {
					t.Fatalf("invalid continuation: %s", raw)
				}
				if !strings.Contains(raw, `"nextPageUrl":`) {
					t.Fatal("missing explicit continuation")
				}
				for _, item := range decoded.Items {
					got = append(got, int(item["id"].(float64)))
				}
				if more {
					args["nextPageUrl"] = *decoded.Summary.NextPageURL
				}
			}
			if !reflect.DeepEqual(got, want) || requests != len(pages) {
				t.Fatalf("got %v want %v; requests %d", got, want, requests)
			}
		})
	}
}

func TestCompanyLegacyPages(t *testing.T) {
	for _, page := range []int{1, 2, 3} {
		t.Run(fmt.Sprint(page), func(t *testing.T) {
			requests := 0
			var base string
			client, _ := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
				requests++
				var next *string
				if requests == 1 {
					value := base + "/Companies/query/next?paging=two"
					next = &value
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"items": []map[string]any{{"id": requests, "companyName": "Test"}}, "pageDetails": map[string]any{"nextPageUrl": next}})
			})
			base = client.baseURL
			srv := server.NewMCPServer("test", "1")
			registerCompanyTools(srv, client, testLogger())
			args := map[string]any{}
			if page > 1 {
				args["page"] = page
			}
			result := callTool(t, srv, "search_companies", args, nil)
			if result.IsError {
				t.Fatal(resultText(t, result))
			}
			var decoded companyResult
			if err := json.Unmarshal([]byte(resultText(t, result)), &decoded); err != nil {
				t.Fatal(err)
			}
			if page <= 2 && (len(decoded.Items) != 1 || int(decoded.Items[0]["id"].(float64)) != page) {
				t.Fatalf("wrong page: %+v", decoded)
			}
			if page == 3 && (len(decoded.Items) != 0 || decoded.Summary.HasMore) {
				t.Fatal("expected empty end")
			}
			if requests != min(page, 2) || decoded.Summary.PageSize != 25 {
				t.Fatalf("requests=%d result=%+v", requests, decoded)
			}
		})
	}
}

func TestCompanyInvalidArguments(t *testing.T) {
	client, _ := setupTestServer(t, func(w http.ResponseWriter, _ *http.Request) { t.Error("unexpected HTTP request"); w.WriteHeader(400) })
	srv := server.NewMCPServer("test", "1")
	registerCompanyTools(srv, client, testLogger())
	for _, args := range []map[string]any{
		{"page": 0}, {"page": -1}, {"page": 1.5}, {"page": "2"}, {"page": 1e30},
		{"pageSize": 0}, {"pageSize": 201}, {"pageSize": 1.5}, {"nextPageUrl": ""}, {"nextPageUrl": 12},
	} {
		t.Run(fmt.Sprint(args), func(t *testing.T) {
			result := callTool(t, srv, "search_companies", args, nil)
			if !result.IsError {
				t.Fatal("expected invalid argument error")
			}
		})
	}
}

func TestCompanyPageFailure(t *testing.T) {
	for _, tc := range []struct {
		name, body       string
		status, attempts int
	}{
		{"4xx", "denied", 403, 1}, {"5xx", "unavailable", 500, 4},
		{"json", "{", 200, 1}, {"missing", `{"items":[]}`, 200, 1},
		{"null details", `{"items":[],"pageDetails":null}`, 200, 1},
		{"missing cursor", `{"items":[],"pageDetails":{}}`, 200, 1},
		{"numeric cursor", `{"items":[],"pageDetails":{"nextPageUrl":4}}`, 200, 1},
		{"empty cursor", `{"items":[],"pageDetails":{"nextPageUrl":""}}`, 200, 1},
		{"missing items", `{"pageDetails":{"nextPageUrl":null}}`, 200, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			first, next := 0, 0
			var base string
			client, _ := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/Companies/query" {
					first++
					_ = json.NewEncoder(w).Encode(map[string]any{"items": []map[string]any{{"id": 1}}, "pageDetails": map[string]any{"nextPageUrl": base + "/Companies/query/next?paging=two"}})
					return
				}
				next++
				w.WriteHeader(tc.status)
				_, _ = io.WriteString(w, tc.body)
			})
			base = client.baseURL
			srv := server.NewMCPServer("test", "1")
			registerCompanyTools(srv, client, testLogger())
			result := callTool(t, srv, "search_companies", map[string]any{"page": 2}, nil)
			if !result.IsError || first != 1 || next != tc.attempts {
				t.Fatalf("error=%v first=%d next=%d", result.IsError, first, next)
			}
		})
	}
}

func TestQueryPageCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	client, _ := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		cancel()
		<-r.Context().Done()
	})
	_, err := client.QueryPage(ctx, "Companies", nil, QueryOpts{}, "")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation error, got %v", err)
	}
}

func TestCompanyRepeatedContinuation(t *testing.T) {
	calls := 0
	var base string
	client, _ := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		next := "a"
		if r.URL.RawQuery == "paging=a" {
			next = "b"
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"items": []map[string]any{}, "pageDetails": map[string]any{"nextPageUrl": base + "/Companies/query/next?paging=" + next}})
	})
	base = client.baseURL
	srv := server.NewMCPServer("test", "1")
	registerCompanyTools(srv, client, testLogger())
	result := callTool(t, srv, "search_companies", map[string]any{"page": 4}, nil)
	if !result.IsError || calls != 3 {
		t.Fatalf("expected cycle rejection, calls=%d", calls)
	}
}

func TestQueryPageRejectsUnsafeContinuation(t *testing.T) {
	client, _ := setupTestServer(t, func(w http.ResponseWriter, _ *http.Request) { t.Error("unexpected request"); w.WriteHeader(400) })
	base := client.baseURL
	for _, cursor := range []string{
		"https://other.invalid/Companies/query/next?paging=a", strings.Replace(base, "http:", "https:", 1) + "/Companies/query/next?paging=a",
		base + "1/Companies/query/next?paging=a", strings.Replace(base, "http://", "http://user:pass@", 1) + "/Companies/query/next?paging=a",
		base + "/Companies/query/next?paging=a#fragment", base + "/Tickets/query/next?paging=a", base + "/evil/Companies/query/next?paging=a",
		base + "/Companies/query/next", base + "/Companies/query/%6eext?paging=a", "/Companies/query/next?paging=a",
	} {
		t.Run(cursor, func(t *testing.T) {
			_, err := client.QueryPage(context.Background(), "Companies", nil, QueryOpts{}, cursor)
			if err == nil {
				t.Fatal("expected rejected cursor")
			}
		})
	}
}

func TestAutotaskDoesNotFollowRedirect(t *testing.T) {
	var requests atomic.Int32
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { requests.Add(1) }))
	defer target.Close()
	client, _ := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusTemporaryRedirect)
	})
	_, err := client.QueryPage(context.Background(), "Companies", nil, QueryOpts{}, "")
	if err == nil || requests.Load() != 0 {
		t.Fatalf("redirect followed: requests=%d err=%v", requests.Load(), err)
	}
}

func TestCompanySchemaContinuation(t *testing.T) {
	srv := server.NewMCPServer("test", "1")
	registerCompanyTools(srv, NewClient(Config{}, testLogger()), testLogger())
	response := srv.HandleMessage(context.Background(), json.RawMessage(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`))
	data, err := json.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}
	var envelope struct {
		Result struct {
			Tools []struct {
				Name        string `json:"name"`
				InputSchema struct {
					Properties map[string]struct {
						Type string `json:"type"`
					} `json:"properties"`
					Required []string `json:"required"`
				} `json:"inputSchema"`
			} `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		t.Fatal(err)
	}
	for _, tool := range envelope.Result.Tools {
		if tool.Name != "search_companies" {
			continue
		}
		if tool.InputSchema.Properties["nextPageUrl"].Type != "string" {
			t.Fatal("missing cursor schema")
		}
		for _, required := range tool.InputSchema.Required {
			if required == "nextPageUrl" {
				t.Fatal("cursor must be optional")
			}
		}
		return
	}
	t.Fatal("missing search_companies tool")
}
