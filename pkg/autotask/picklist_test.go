package autotask

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
)

func newTestPicklistServer(t *testing.T, callCount *atomic.Int32) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if callCount != nil {
			callCount.Add(1)
		}
		// Serve field info for /Tickets/entityInformation/fields
		fields := []map[string]any{
			{
				"name":        "status",
				"dataType":    "integer",
				"isRequired":  true,
				"isPickList":  true,
				"isQueryable": true,
				"isReadOnly":  false,
				"isReference": false,
				"picklistValues": []map[string]any{
					{"value": "1", "label": "New", "isActive": true, "sortOrder": 1},
					{"value": "5", "label": "Complete", "isActive": true, "sortOrder": 2},
					{"value": "99", "label": "Deprecated", "isActive": false, "sortOrder": 99},
				},
			},
			{
				"name":        "queueID",
				"dataType":    "integer",
				"isPickList":  true,
				"isQueryable": true,
				"isRequired":  false,
				"isReadOnly":  false,
				"isReference": false,
				"picklistValues": []map[string]any{
					{"value": "1", "label": "Helpdesk", "isActive": true},
					{"value": "2", "label": "Level II", "isActive": true},
				},
			},
			{
				"name":        "priority",
				"dataType":    "integer",
				"isPickList":  true,
				"isQueryable": true,
				"isRequired":  true,
				"isReadOnly":  false,
				"isReference": false,
				"picklistValues": []map[string]any{
					{"value": "1", "label": "Critical", "isActive": true},
					{"value": "4", "label": "Low", "isActive": true},
				},
			},
			{
				"name":        "title",
				"dataType":    "string",
				"length":      255,
				"isPickList":  false,
				"isQueryable": true,
				"isRequired":  true,
				"isReadOnly":  false,
				"isReference": false,
			},
		}
		_ = json.NewEncoder(w).Encode(fields)
	}))
}

func TestPicklistCache_GetFields(t *testing.T) {
	var calls atomic.Int32
	srv := newTestPicklistServer(t, &calls)
	defer srv.Close()

	logger := slog.Default()
	client := NewClient(Config{BaseURL: srv.URL, Username: "u", Secret: "s", IntegrationCode: "i"}, logger)
	cache := NewPicklistCache(client, logger)

	fields, err := cache.GetFields(context.Background(), "Tickets")
	if err != nil {
		t.Fatalf("GetFields failed: %v", err)
	}
	if len(fields) != 4 {
		t.Errorf("fields count = %d, want 4", len(fields))
	}
	if fields[0].Name != "status" {
		t.Errorf("first field = %s, want status", fields[0].Name)
	}
}

func TestPicklistCache_Caching(t *testing.T) {
	var calls atomic.Int32
	srv := newTestPicklistServer(t, &calls)
	defer srv.Close()

	logger := slog.Default()
	client := NewClient(Config{BaseURL: srv.URL, Username: "u", Secret: "s", IntegrationCode: "i"}, logger)
	cache := NewPicklistCache(client, logger)

	_, _ = cache.GetFields(context.Background(), "Tickets")
	_, _ = cache.GetFields(context.Background(), "Tickets")
	if calls.Load() != 1 {
		t.Errorf("API called %d times, want 1 (should be cached)", calls.Load())
	}
}

func TestPicklistCache_GetPicklistValues(t *testing.T) {
	srv := newTestPicklistServer(t, nil)
	defer srv.Close()

	logger := slog.Default()
	client := NewClient(Config{BaseURL: srv.URL, Username: "u", Secret: "s", IntegrationCode: "i"}, logger)
	cache := NewPicklistCache(client, logger)

	values, err := cache.GetPicklistValues(context.Background(), "Tickets", "status")
	if err != nil {
		t.Fatalf("GetPicklistValues failed: %v", err)
	}
	// Should only return active values (2 out of 3)
	if len(values) != 2 {
		t.Errorf("active values = %d, want 2", len(values))
	}
}

func TestPicklistCache_GetQueues(t *testing.T) {
	srv := newTestPicklistServer(t, nil)
	defer srv.Close()

	logger := slog.Default()
	client := NewClient(Config{BaseURL: srv.URL, Username: "u", Secret: "s", IntegrationCode: "i"}, logger)
	cache := NewPicklistCache(client, logger)

	queues, err := cache.GetQueues(context.Background())
	if err != nil {
		t.Fatalf("GetQueues failed: %v", err)
	}
	if len(queues) != 2 {
		t.Errorf("queues = %d, want 2", len(queues))
	}
}

func TestPicklistCache_ConcurrentAccess(t *testing.T) {
	var calls atomic.Int32
	srv := newTestPicklistServer(t, &calls)
	defer srv.Close()

	logger := slog.Default()
	client := NewClient(Config{BaseURL: srv.URL, Username: "u", Secret: "s", IntegrationCode: "i"}, logger)
	cache := NewPicklistCache(client, logger)

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = cache.GetFields(context.Background(), "Tickets")
		}()
	}
	wg.Wait()
	// Should only call API once despite concurrent access
	if calls.Load() > 2 {
		t.Errorf("API called %d times, want <=2 for concurrent access", calls.Load())
	}
}

func TestPicklistCache_ClearCache(t *testing.T) {
	var calls atomic.Int32
	srv := newTestPicklistServer(t, &calls)
	defer srv.Close()

	logger := slog.Default()
	client := NewClient(Config{BaseURL: srv.URL, Username: "u", Secret: "s", IntegrationCode: "i"}, logger)
	cache := NewPicklistCache(client, logger)

	_, _ = cache.GetFields(context.Background(), "Tickets")
	cache.ClearCache("Tickets")
	_, _ = cache.GetFields(context.Background(), "Tickets")
	if calls.Load() != 2 {
		t.Errorf("API called %d times after clear, want 2", calls.Load())
	}
}

func TestNormalizeEntityType(t *testing.T) {
	tests := []struct{ input, want string }{
		{"tasks", "Tasks"},
		{"ProjectTasks", "Tasks"},
		{"tickets", "Tickets"},
		{"SomethingNew", "SomethingNew"},
	}
	for _, tt := range tests {
		got := NormalizeEntityType(tt.input)
		if got != tt.want {
			t.Errorf("NormalizeEntityType(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
