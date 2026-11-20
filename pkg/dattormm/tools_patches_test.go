package dattormm

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func TestPatchRead(t *testing.T) {
	var tokens, calls atomic.Int32
	api := apiServer(t, &tokens, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method != http.MethodGet || r.Header.Get("Authorization") != "Bearer test-bearer-token" {
			t.Errorf("unexpected patch request %s", r.Method)
		}
		switch r.URL.Path {
		case "/api/v2/device/device-1/patches":
			q := r.URL.Query()
			if q.Get("page") != "0" || q.Get("max") != "2" || q.Get("installStatus") != "APPROVED_PENDING" {
				t.Errorf("wrong filters: %v", q)
			}
			_, _ = w.Write([]byte(`{"patches":[{"patchId":"p1","installStatus":"APPROVED_PENDING","manualOverride":true,"futureField":"unreviewed"}],"pageDetails":{"count":1,"totalCount":3,"nextPageUrl":"next-page"}}`))
		case "/api/v2/site/site-1/patches":
			_, _ = w.Write([]byte(`{"patches":[{"patchId":"p1","totalDevices":3}],"pageDetails":{"count":1,"totalCount":1,"nextPageUrl":null}}`))
		case "/api/v2/site/empty/patches":
			_, _ = w.Write([]byte(`{"patches":[],"pageDetails":{"count":0,"totalCount":0,"nextPageUrl":null}}`))
		case "/api/v2/device/restricted/patches":
			w.WriteHeader(http.StatusForbidden)
		case "/api/v2/device/malformed/patches":
			_, _ = w.Write([]byte(`{"message":"unexpected response"}`))
		default:
			t.Errorf("unexpected request path %s", r.URL.Path)
			w.WriteHeader(http.StatusBadRequest)
		}
	})
	defer api.Close()
	c := newTestClient(t, api.URL)
	disabled := server.NewMCPServer("test", "1")
	RegisterTools(disabled, c, newTestLogger())
	if disabled.GetTool("list_patches") != nil {
		t.Fatal("patch tool published without opt-in")
	}
	c.enablePatchReads = true
	srv := server.NewMCPServer("test", "1")
	RegisterTools(srv, c, newTestLogger())
	tool := srv.GetTool("list_patches")
	if tool == nil || tool.Tool.Annotations.ReadOnlyHint == nil || !*tool.Tool.Annotations.ReadOnlyHint {
		t.Fatal("missing read-only registration")
	}
	for _, tc := range []struct {
		name      string
		args      map[string]any
		wantError bool
		upstream  bool
	}{
		{"device", map[string]any{"deviceUid": "device-1", "page": 0, "max": 2, "installStatus": "APPROVED_PENDING"}, false, true},
		{"site", map[string]any{"siteUid": "site-1"}, false, true},
		{"empty", map[string]any{"siteUid": "empty"}, false, true},
		{"denied", map[string]any{"deviceUid": "restricted"}, true, true},
		{"malformed", map[string]any{"deviceUid": "malformed"}, true, true},
		{"no scope", map[string]any{}, true, false},
		{"two scopes", map[string]any{"deviceUid": "a", "siteUid": "b"}, true, false},
		{"multiple segments", map[string]any{"deviceUid": "a/b"}, true, false},
		{"status", map[string]any{"deviceUid": "a", "installStatus": "all"}, true, false},
		{"page", map[string]any{"deviceUid": "a", "page": 1.5}, true, false},
		{"size", map[string]any{"deviceUid": "a", "max": 0}, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := mcp.CallToolRequest{}
			req.Params.Arguments = tc.args
			before := calls.Load()
			got, err := tool.Handler(context.Background(), req)
			if err != nil || got.IsError != tc.wantError || (calls.Load() != before) != tc.upstream {
				t.Fatalf("result=%v err=%v", got, err)
			}
			if tc.wantError {
				return
			}
			var body struct {
				Items []struct {
					InstallStatus  string `json:"installStatus"`
					TotalDevices   int    `json:"totalDevices"`
					ManualOverride bool   `json:"manualOverride"`
				} `json:"items"`
				Pagination struct {
					TotalCount int     `json:"totalCount"`
					Next       *string `json:"nextPageUrl"`
				} `json:"pagination"`
			}
			if err := json.Unmarshal([]byte(got.Content[0].(mcp.TextContent).Text), &body); err != nil {
				t.Fatal(err)
			}
			if tc.name == "device" && (len(body.Items) != 1 || body.Items[0].InstallStatus != "APPROVED_PENDING" || !body.Items[0].ManualOverride || body.Pagination.TotalCount != 3 || body.Pagination.Next == nil || strings.Contains(got.Content[0].(mcp.TextContent).Text, "unreviewed")) {
				t.Fatalf("device page: %+v", body)
			}
			if tc.name == "site" && (len(body.Items) != 1 || body.Items[0].TotalDevices != 3) {
				t.Fatalf("site page: %+v", body)
			}
			if tc.name == "empty" && (body.Items == nil || len(body.Items) != 0 || body.Pagination.Next != nil) {
				t.Fatalf("empty page: %+v", body)
			}
		})
	}
}
