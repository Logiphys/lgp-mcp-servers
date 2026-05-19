package autotask

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

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func setupTestServer(t *testing.T, handler http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	cfg := Config{
		Username:        "testuser",
		Secret:          "testsecret",
		IntegrationCode: "TESTCODE",
		BaseURL:         srv.URL,
	}
	client := NewClient(cfg, testLogger())
	return client, srv
}

func TestAuthHeaders(t *testing.T) {
	client, _ := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("UserName") != "testuser" {
			t.Errorf("expected UserName header 'testuser', got %q", r.Header.Get("UserName"))
		}
		if r.Header.Get("Secret") != "testsecret" {
			t.Errorf("expected Secret header 'testsecret', got %q", r.Header.Get("Secret"))
		}
		if r.Header.Get("ApiIntegrationcode") != "TESTCODE" {
			t.Errorf("expected ApiIntegrationcode header 'TESTCODE', got %q", r.Header.Get("ApiIntegrationcode"))
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{"item": map[string]any{"id": 1}})
	})

	_, err := client.Get(context.Background(), "Tickets", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGet(t *testing.T) {
	client, _ := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/Tickets/42" {
			t.Errorf("expected path /Tickets/42, got %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"item": map[string]any{
				"id":    float64(42),
				"title": "Test ticket",
			},
		})
	})

	item, err := client.Get(context.Background(), "Tickets", 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item == nil {
		t.Fatal("expected item, got nil")
	}
	if item["title"] != "Test ticket" {
		t.Errorf("expected title 'Test ticket', got %v", item["title"])
	}
}

func TestQuery(t *testing.T) {
	client, _ := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/Tickets/query" {
			t.Errorf("expected path /Tickets/query, got %s", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]any
		_ = json.Unmarshal(body, &reqBody)

		if reqBody["MaxRecords"] == nil {
			t.Error("expected MaxRecords in request body")
		}
		filters, ok := reqBody["filter"].([]any)
		if !ok || len(filters) == 0 {
			t.Error("expected non-empty filter array in request body")
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"items": []map[string]any{
				{"id": float64(1), "title": "Ticket 1"},
				{"id": float64(2), "title": "Ticket 2"},
			},
		})
	})

	filters := []Filter{
		{Op: "eq", Field: "status", Value: 1},
	}
	items, err := client.Query(context.Background(), "Tickets", filters, QueryOpts{PageSize: 100})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
}

func TestCreateItemIdPattern(t *testing.T) {
	client, _ := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"itemId": float64(999)})
	})

	id, err := client.Create(context.Background(), "Tickets", map[string]any{"title": "New"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 999 {
		t.Errorf("expected ID 999, got %d", id)
	}
}

func TestCreateItemNestedIdPattern(t *testing.T) {
	client, _ := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"item": map[string]any{"id": float64(888)},
		})
	})

	id, err := client.Create(context.Background(), "Tickets", map[string]any{"title": "New"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 888 {
		t.Errorf("expected ID 888, got %d", id)
	}
}

func TestCreateTopLevelIdPattern(t *testing.T) {
	client, _ := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"id": float64(777)})
	})

	id, err := client.Create(context.Background(), "Tickets", map[string]any{"title": "New"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 777 {
		t.Errorf("expected ID 777, got %d", id)
	}
}

func TestCreateChild(t *testing.T) {
	client, _ := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/Tickets/123/Notes" {
			t.Errorf("expected path /Tickets/123/Notes, got %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"itemId": float64(456)})
	})

	id, err := client.CreateChild(context.Background(), "Tickets", 123, "Notes", map[string]any{"text": "note"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 456 {
		t.Errorf("expected ID 456, got %d", id)
	}
}

func TestDeleteChild(t *testing.T) {
	client, _ := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/Tickets/123/Notes/456" {
			t.Errorf("expected path /Tickets/123/Notes/456, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	})

	err := client.DeleteChild(context.Background(), "Tickets", 123, "Notes", 456)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdate(t *testing.T) {
	client, _ := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if r.URL.Path != "/Tickets/42" {
			t.Errorf("expected path /Tickets/42, got %s", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]any
		_ = json.Unmarshal(body, &reqBody)
		if reqBody["status"] != float64(5) {
			t.Errorf("expected status 5 in body, got %v", reqBody["status"])
		}

		w.WriteHeader(http.StatusOK)
	})

	err := client.Update(context.Background(), "Tickets", 42, map[string]any{"status": 5})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDelete(t *testing.T) {
	client, _ := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/Tickets/42" {
			t.Errorf("expected path /Tickets/42, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	})

	err := client.Delete(context.Background(), "Tickets", 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetFieldInfoWithEnvelope(t *testing.T) {
	client, _ := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/Tickets/entityInformation/fields" {
			t.Errorf("expected path /Tickets/entityInformation/fields, got %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"fields": []map[string]any{
				{"name": "id", "type": "integer"},
				{"name": "title", "type": "string"},
			},
		})
	})

	fields, err := client.GetFieldInfo(context.Background(), "Tickets")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(fields))
	}
	if fields[0]["name"] != "id" {
		t.Errorf("expected first field name 'id', got %v", fields[0]["name"])
	}
}

func TestGetFieldInfoBareArray(t *testing.T) {
	client, _ := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{"name": "id", "type": "integer"},
		})
	})

	fields, err := client.GetFieldInfo(context.Background(), "Tickets")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fields) != 1 {
		t.Fatalf("expected 1 field, got %d", len(fields))
	}
}

func TestEnhanceWithNames(t *testing.T) {
	callCount := 0
	client, _ := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		callCount++
		switch {
		case strings.HasPrefix(r.URL.Path, "/Companies/"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"item": map[string]any{
					"id":          float64(10),
					"companyName": "Acme Corp",
				},
			})
		case strings.HasPrefix(r.URL.Path, "/Resources/"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"item": map[string]any{
					"id":        float64(20),
					"firstName": "Jane",
					"lastName":  "Doe",
				},
			})
		}
	})

	items := []map[string]any{
		{
			"id":                 float64(1),
			"companyID":          float64(10),
			"assignedResourceID": float64(20),
		},
	}

	result := client.EnhanceWithNames(context.Background(), items)
	if result[0]["_companyName"] != "Acme Corp" {
		t.Errorf("expected _companyName 'Acme Corp', got %v", result[0]["_companyName"])
	}
	if result[0]["_assignedResourceName"] != "Jane Doe" {
		t.Errorf("expected _assignedResourceName 'Jane Doe', got %v", result[0]["_assignedResourceName"])
	}
}

func TestHTTPError(t *testing.T) {
	client, _ := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error": "not found"}`))
	})

	_, err := client.Get(context.Background(), "Tickets", 999)
	if err == nil {
		t.Fatal("expected error for HTTP 404, got nil")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("expected error to contain '404', got: %s", err.Error())
	}
}

func TestNewClientNoBaseURL(t *testing.T) {
	cfg := Config{
		Username:        "user",
		Secret:          "secret",
		IntegrationCode: "code",
	}
	client := NewClient(cfg, testLogger())
	if client.baseURL != "" {
		t.Errorf("expected empty base URL when not provided, got %s", client.baseURL)
	}
}

func TestNewClientCustomBaseURL(t *testing.T) {
	cfg := Config{
		Username:        "user",
		Secret:          "secret",
		IntegrationCode: "code",
		BaseURL:         "https://custom.example.com/api",
	}
	client := NewClient(cfg, testLogger())
	if client.baseURL != "https://custom.example.com/api" {
		t.Errorf("expected custom base URL, got %s", client.baseURL)
	}
}

func TestExtractID(t *testing.T) {
	tests := []struct {
		name   string
		resp   map[string]any
		wantID int
		wantOK bool
	}{
		{
			name:   "itemId pattern",
			resp:   map[string]any{"itemId": float64(100)},
			wantID: 100,
			wantOK: true,
		},
		{
			name:   "item.id pattern",
			resp:   map[string]any{"item": map[string]any{"id": float64(200)}},
			wantID: 200,
			wantOK: true,
		},
		{
			name:   "id pattern",
			resp:   map[string]any{"id": float64(300)},
			wantID: 300,
			wantOK: true,
		},
		{
			name:   "no id",
			resp:   map[string]any{"message": "ok"},
			wantID: 0,
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, ok := extractID(tt.resp)
			if ok != tt.wantOK {
				t.Errorf("extractID() ok = %v, want %v", ok, tt.wantOK)
			}
			if id != tt.wantID {
				t.Errorf("extractID() id = %d, want %d", id, tt.wantID)
			}
		})
	}
}

func TestTestConnection(t *testing.T) {
	client, _ := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/Companies/0" {
			t.Errorf("expected path /Companies/0, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	})

	err := client.TestConnection(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestQueryDefaultPageSize(t *testing.T) {
	client, _ := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]any
		_ = json.Unmarshal(body, &reqBody)

		maxRecords := reqBody["MaxRecords"].(float64)
		if maxRecords != 500 {
			t.Errorf("expected default MaxRecords 500, got %v", maxRecords)
		}

		_ = json.NewEncoder(w).Encode(map[string]any{"items": []map[string]any{}})
	})

	_, err := client.Query(context.Background(), "Tickets", nil, QueryOpts{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestQueryMaxSizeCap(t *testing.T) {
	client, _ := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]any
		_ = json.Unmarshal(body, &reqBody)

		maxRecords := reqBody["MaxRecords"].(float64)
		if maxRecords != 50 {
			t.Errorf("expected MaxRecords capped to 50, got %v", maxRecords)
		}

		_ = json.NewEncoder(w).Encode(map[string]any{"items": []map[string]any{}})
	})

	_, err := client.Query(context.Background(), "Tickets", nil, QueryOpts{PageSize: 100, MaxSize: 50})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
