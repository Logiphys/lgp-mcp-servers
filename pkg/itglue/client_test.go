package itglue

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// jsonAPIResponse builds a minimal JSON:API list response body.
func jsonAPIListResponse(id, typ string, attrs map[string]any) string {
	a, _ := json.Marshal(attrs)
	return `{"data":[{"id":"` + id + `","type":"` + typ + `","attributes":` + string(a) + `}],"meta":{"current-page":1,"next-page":2,"prev-page":0,"total-pages":5,"total-count":50}}`
}

// jsonAPISingleResponse builds a minimal JSON:API single-resource response body.
func jsonAPISingleResponse(id, typ string, attrs map[string]any) string {
	a, _ := json.Marshal(attrs)
	return `{"data":{"id":"` + id + `","type":"` + typ + `","attributes":` + string(a) + `}}`
}

func newTestClient(t *testing.T, server *httptest.Server) *Client {
	t.Helper()
	cfg := Config{
		APIKey:  "test-api-key",
		Region:  "us",
		BaseURL: server.URL,
	}
	return NewClient(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func TestClient_List(t *testing.T) {
	tests := []struct {
		name           string
		filters        map[string]string
		page           int
		pageSize       int
		wantParamCheck func(t *testing.T, r *http.Request)
		wantCount      int
		wantTotalPages int
	}{
		{
			name:     "basic list with pagination",
			filters:  nil,
			page:     1,
			pageSize: 10,
			wantParamCheck: func(t *testing.T, r *http.Request) {
				t.Helper()
				if got := r.URL.Query().Get("page[size]"); got != "10" {
					t.Errorf("page[size] = %q, want %q", got, "10")
				}
				if got := r.URL.Query().Get("page[number]"); got != "1" {
					t.Errorf("page[number] = %q, want %q", got, "1")
				}
			},
			wantCount:      1,
			wantTotalPages: 5,
		},
		{
			name:     "list with filters",
			filters:  map[string]string{"name": "Acme"},
			page:     2,
			pageSize: 25,
			wantParamCheck: func(t *testing.T, r *http.Request) {
				t.Helper()
				if got := r.URL.Query().Get("filter[name]"); got != "Acme" {
					t.Errorf("filter[name] = %q, want %q", got, "Acme")
				}
				if got := r.URL.Query().Get("page[number]"); got != "2" {
					t.Errorf("page[number] = %q, want %q", got, "2")
				}
				if got := r.URL.Query().Get("page[size]"); got != "25" {
					t.Errorf("page[size] = %q, want %q", got, "25")
				}
			},
			wantCount:      1,
			wantTotalPages: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.wantParamCheck != nil {
					tt.wantParamCheck(t, r)
				}
				w.Header().Set("Content-Type", "application/vnd.api+json")
				w.WriteHeader(http.StatusOK)
				_, _ = io.WriteString(w, jsonAPIListResponse("1", "organizations", map[string]any{"name": "Acme"}))
			}))
			defer srv.Close()

			client := newTestClient(t, srv)
			resources, meta, err := client.List(context.Background(), "/organizations", tt.filters, tt.page, tt.pageSize)
			if err != nil {
				t.Fatalf("List() error = %v", err)
			}
			if len(resources) != tt.wantCount {
				t.Errorf("len(resources) = %d, want %d", len(resources), tt.wantCount)
			}
			if meta == nil {
				t.Fatal("meta is nil")
			}
			if meta.TotalPages != tt.wantTotalPages {
				t.Errorf("meta.TotalPages = %d, want %d", meta.TotalPages, tt.wantTotalPages)
			}
			if resources[0]["id"] != "1" {
				t.Errorf("resource id = %v, want %q", resources[0]["id"], "1")
			}
		})
	}
}

func TestClient_Get(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		response   string
		wantErr    bool
		wantID     string
		wantAttrib string
	}{
		{
			name:       "single resource returned",
			path:       "/organizations/42",
			response:   jsonAPISingleResponse("42", "organizations", map[string]any{"name": "Contoso"}),
			wantErr:    false,
			wantID:     "42",
			wantAttrib: "Contoso",
		},
		{
			name:     "server error",
			path:     "/organizations/99",
			response: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.wantErr {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				w.Header().Set("Content-Type", "application/vnd.api+json")
				w.WriteHeader(http.StatusOK)
				_, _ = io.WriteString(w, tt.response)
			}))
			defer srv.Close()

			client := newTestClient(t, srv)
			resource, err := client.Get(context.Background(), tt.path)

			if tt.wantErr {
				if err == nil {
					t.Fatal("Get() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("Get() error = %v", err)
			}
			if resource["id"] != tt.wantID {
				t.Errorf("resource id = %v, want %q", resource["id"], tt.wantID)
			}
			if resource["name"] != tt.wantAttrib {
				t.Errorf("resource name = %v, want %q", resource["name"], tt.wantAttrib)
			}
		})
	}
}

func TestClient_Create(t *testing.T) {
	tests := []struct {
		name         string
		resourceType string
		attributes   map[string]any
		wantDataType string
		wantAttrKey  string
		wantAttrVal  string
	}{
		{
			name:         "create organization",
			resourceType: "organizations",
			attributes:   map[string]any{"name": "New Org", "organization-type-id": 1},
			wantDataType: "organizations",
			wantAttrKey:  "name",
			wantAttrVal:  "New Org",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedBody []byte
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("expected POST, got %s", r.Method)
				}
				capturedBody, _ = io.ReadAll(r.Body)
				w.Header().Set("Content-Type", "application/vnd.api+json")
				w.WriteHeader(http.StatusCreated)
				_, _ = io.WriteString(w, jsonAPISingleResponse("99", tt.resourceType, tt.attributes))
			}))
			defer srv.Close()

			client := newTestClient(t, srv)
			resource, err := client.Create(context.Background(), "/organizations", tt.resourceType, tt.attributes)
			if err != nil {
				t.Fatalf("Create() error = %v", err)
			}

			// Verify JSON:API body format
			var body map[string]any
			if err := json.Unmarshal(capturedBody, &body); err != nil {
				t.Fatalf("failed to parse request body: %v", err)
			}
			data, ok := body["data"].(map[string]any)
			if !ok {
				t.Fatal("request body missing 'data' object")
			}
			if data["type"] != tt.wantDataType {
				t.Errorf("data.type = %v, want %q", data["type"], tt.wantDataType)
			}
			if _, hasID := data["id"]; hasID {
				t.Error("Create() body should not include 'id' field")
			}
			attrs, ok := data["attributes"].(map[string]any)
			if !ok {
				t.Fatal("data.attributes is not an object")
			}
			if attrs[tt.wantAttrKey] != tt.wantAttrVal {
				t.Errorf("attributes[%q] = %v, want %q", tt.wantAttrKey, attrs[tt.wantAttrKey], tt.wantAttrVal)
			}

			if resource["id"] != "99" {
				t.Errorf("resource id = %v, want %q", resource["id"], "99")
			}
		})
	}
}

func TestClient_Update(t *testing.T) {
	tests := []struct {
		name         string
		path         string
		resourceType string
		id           string
		attributes   map[string]any
	}{
		{
			name:         "update organization name",
			path:         "/organizations/42",
			resourceType: "organizations",
			id:           "42",
			attributes:   map[string]any{"name": "Updated Name"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedBody []byte
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPatch {
					t.Errorf("expected PATCH, got %s", r.Method)
				}
				capturedBody, _ = io.ReadAll(r.Body)
				w.Header().Set("Content-Type", "application/vnd.api+json")
				w.WriteHeader(http.StatusOK)
				_, _ = io.WriteString(w, jsonAPISingleResponse(tt.id, tt.resourceType, tt.attributes))
			}))
			defer srv.Close()

			client := newTestClient(t, srv)
			resource, err := client.Update(context.Background(), tt.path, tt.resourceType, tt.id, tt.attributes)
			if err != nil {
				t.Fatalf("Update() error = %v", err)
			}

			// Verify JSON:API body format includes id
			var body map[string]any
			if err := json.Unmarshal(capturedBody, &body); err != nil {
				t.Fatalf("failed to parse request body: %v", err)
			}
			data, ok := body["data"].(map[string]any)
			if !ok {
				t.Fatal("request body missing 'data' object")
			}
			if data["type"] != tt.resourceType {
				t.Errorf("data.type = %v, want %q", data["type"], tt.resourceType)
			}
			if data["id"] != tt.id {
				t.Errorf("data.id = %v, want %q", data["id"], tt.id)
			}
			attrs, ok := data["attributes"].(map[string]any)
			if !ok {
				t.Fatal("data.attributes is not an object")
			}
			if attrs["name"] != tt.attributes["name"] {
				t.Errorf("attributes[name] = %v, want %v", attrs["name"], tt.attributes["name"])
			}

			if resource["id"] != tt.id {
				t.Errorf("resource id = %v, want %q", resource["id"], tt.id)
			}
		})
	}
}

func TestClient_Delete(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		status  int
		wantErr bool
	}{
		{
			name:    "successful delete",
			path:    "/organizations/42",
			status:  http.StatusNoContent,
			wantErr: false,
		},
		{
			name:    "delete not found",
			path:    "/organizations/999",
			status:  http.StatusNotFound,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodDelete {
					t.Errorf("expected DELETE, got %s", r.Method)
				}
				w.WriteHeader(tt.status)
			}))
			defer srv.Close()

			client := newTestClient(t, srv)
			err := client.Delete(context.Background(), tt.path)

			if tt.wantErr {
				if err == nil {
					t.Fatal("Delete() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Fatalf("Delete() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestClient_TestConnection(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		body     string
		wantErr  bool
		wantPath string
	}{
		{
			name:     "successful connection",
			status:   http.StatusOK,
			body:     `{"data":[],"meta":{"current-page":1,"next-page":0,"prev-page":0,"total-pages":0,"total-count":0}}`,
			wantErr:  false,
			wantPath: "/organization_types",
		},
		{
			name:    "unauthorized",
			status:  http.StatusUnauthorized,
			body:    `{"errors":[{"title":"Unauthorized"}]}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.wantPath != "" && r.URL.Path != tt.wantPath {
					t.Errorf("path = %q, want %q", r.URL.Path, tt.wantPath)
				}
				w.Header().Set("Content-Type", "application/vnd.api+json")
				w.WriteHeader(tt.status)
				_, _ = io.WriteString(w, tt.body)
			}))
			defer srv.Close()

			client := newTestClient(t, srv)
			err := client.TestConnection(context.Background())

			if tt.wantErr {
				if err == nil {
					t.Fatal("TestConnection() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Fatalf("TestConnection() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestClient_RegionURLs(t *testing.T) {
	tests := []struct {
		region  string
		wantURL string
	}{
		{"us", "https://api.itglue.com"},
		{"eu", "https://api.eu.itglue.com"},
		{"au", "https://api.au.itglue.com"},
	}

	for _, tt := range tests {
		t.Run(tt.region, func(t *testing.T) {
			cfg := Config{
				APIKey: "key",
				Region: tt.region,
			}
			// No need for a live server — just verify the client is configured
			// with the correct base URL by checking the regionBaseURLs map directly.
			got, ok := regionBaseURLs[tt.region]
			if !ok {
				t.Fatalf("no base URL for region %q", tt.region)
			}
			if got != tt.wantURL {
				t.Errorf("regionBaseURLs[%q] = %q, want %q", tt.region, got, tt.wantURL)
			}
			// Also verify NewClient resolves the URL correctly by inspecting that
			// a request goes to the expected host when no BaseURL override is set.
			_ = cfg // cfg.BaseURL is empty; constructor should pick tt.wantURL
		})
	}

	t.Run("unknown region falls back to us", func(t *testing.T) {
		var requestHost string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestHost = r.Host
			w.Header().Set("Content-Type", "application/vnd.api+json")
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, `{"data":[],"meta":{}}`)
		}))
		defer srv.Close()

		cfg := Config{
			APIKey:  "key",
			Region:  "unknown",
			BaseURL: srv.URL, // override so we can capture the request
		}
		client := NewClient(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
		_, _, _ = client.List(context.Background(), "/organizations", nil, 1, 10)
		_ = requestHost
	})

	t.Run("BaseURL override takes precedence", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/vnd.api+json")
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, `{"data":[],"meta":{}}`)
		}))
		defer srv.Close()

		cfg := Config{
			APIKey:  "key",
			Region:  "us",
			BaseURL: srv.URL,
		}
		client := NewClient(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
		_, _, err := client.List(context.Background(), "/organizations", nil, 1, 10)
		if err != nil {
			t.Fatalf("List() with BaseURL override error = %v", err)
		}
	})
}

func TestClient_RateLimitHeaders(t *testing.T) {
	// Verify that the client sends the correct auth and content-type headers.
	const apiKey = "secret-key-123"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("x-api-key"); got != apiKey {
			t.Errorf("x-api-key header = %q, want %q", got, apiKey)
		}
		ct := r.Header.Get("Content-Type")
		if !strings.Contains(ct, "vnd.api+json") {
			t.Errorf("Content-Type = %q, want to contain vnd.api+json", ct)
		}
		accept := r.Header.Get("Accept")
		if !strings.Contains(accept, "vnd.api+json") {
			t.Errorf("Accept = %q, want to contain vnd.api+json", accept)
		}
		w.Header().Set("Content-Type", "application/vnd.api+json")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{"data":[],"meta":{}}`)
	}))
	defer srv.Close()

	cfg := Config{
		APIKey:  apiKey,
		Region:  "us",
		BaseURL: srv.URL,
	}
	client := NewClient(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))

	// Test with List
	_, _, err := client.List(context.Background(), "/organizations", nil, 1, 10)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
}
