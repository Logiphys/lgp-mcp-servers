package apihelper

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestClient_Get(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Query().Get("filter") != "active" {
			t.Error("missing query param")
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer srv.Close()
	c := NewClient(ClientConfig{BaseURL: srv.URL, Timeout: 5 * time.Second})
	body, err := c.Get(context.Background(), "/test", map[string]string{"filter": "active"})
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	var result map[string]string
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if result["status"] != "ok" {
		t.Errorf("status = %s", result["status"])
	}
}

func TestClient_Post(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("missing content-type")
		}
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["name"] != "test" {
			t.Error("body not decoded correctly")
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 1})
	}))
	defer srv.Close()
	c := NewClient(ClientConfig{BaseURL: srv.URL})
	body, err := c.Post(context.Background(), "/items", map[string]string{"name": "test"})
	if err != nil {
		t.Fatalf("Post failed: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if result["id"] != float64(1) {
		t.Errorf("id = %v", result["id"])
	}
}

func TestClient_Patch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("method = %s, want PATCH", r.Method)
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"updated": "true"})
	}))
	defer srv.Close()
	c := NewClient(ClientConfig{BaseURL: srv.URL})
	_, err := c.Patch(context.Background(), "/items/1", map[string]string{"name": "updated"})
	if err != nil {
		t.Fatalf("Patch failed: %v", err)
	}
}

func TestClient_Delete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()
	c := NewClient(ClientConfig{BaseURL: srv.URL})
	if err := c.Delete(context.Background(), "/items/1"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
}

func TestClient_CustomHeaders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Api-Key") != "secret" {
			t.Error("custom header not sent")
		}
		if r.Header.Get("User-Agent") != "test-agent" {
			t.Error("user-agent not set")
		}
		w.WriteHeader(200)
	}))
	defer srv.Close()
	c := NewClient(ClientConfig{BaseURL: srv.URL, UserAgent: "test-agent", Headers: map[string]string{"X-Api-Key": "secret"}})
	_, _ = c.Get(context.Background(), "/test", nil)
}

func TestClient_Retry429(t *testing.T) {
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if attempts.Add(1) <= 2 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"ok": "true"})
	}))
	defer srv.Close()
	c := NewClient(ClientConfig{BaseURL: srv.URL, MaxRetries: 3})
	_, err := c.Get(context.Background(), "/test", nil)
	if err != nil {
		t.Fatalf("should succeed after retries: %v", err)
	}
	if attempts.Load() != 3 {
		t.Errorf("attempts = %d, want 3", attempts.Load())
	}
}

func TestClient_Retry5xx(t *testing.T) {
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if attempts.Add(1) <= 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"ok": "true"})
	}))
	defer srv.Close()
	c := NewClient(ClientConfig{BaseURL: srv.URL, MaxRetries: 2})
	_, err := c.Get(context.Background(), "/test", nil)
	if err != nil {
		t.Fatalf("should succeed after retry: %v", err)
	}
}

func TestClient_NoRetry4xx(t *testing.T) {
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error": "bad request"}`))
	}))
	defer srv.Close()
	c := NewClient(ClientConfig{BaseURL: srv.URL, MaxRetries: 3})
	_, err := c.Get(context.Background(), "/test", nil)
	if err == nil {
		t.Error("should return error for 400")
	}
	if attempts.Load() != 1 {
		t.Errorf("should not retry 4xx: attempts = %d", attempts.Load())
	}
}

func TestClient_ContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
	}))
	defer srv.Close()
	c := NewClient(ClientConfig{BaseURL: srv.URL})
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err := c.Get(ctx, "/test", nil)
	if err == nil {
		t.Error("should fail with canceled context")
	}
}

func TestRequestHeadersFromContext(t *testing.T) {
	tests := []struct {
		name    string
		ctx     context.Context
		static  map[string]string
		want    map[string]string
		wantNot []string
	}{
		{
			name:    "no request headers",
			ctx:     context.Background(),
			static:  map[string]string{"Secret": "s"},
			want:    map[string]string{"Secret": "s"},
			wantNot: []string{"X-Extra"},
		},
		{
			name:   "request header added",
			ctx:    WithRequestHeaders(context.Background(), map[string]string{"X-Extra": "1"}),
			static: map[string]string{"Secret": "s"},
			want:   map[string]string{"Secret": "s", "X-Extra": "1"},
		},
		{
			name:   "request header cannot override static auth header",
			ctx:    WithRequestHeaders(context.Background(), map[string]string{"Secret": "evil"}),
			static: map[string]string{"Secret": "s"},
			want:   map[string]string{"Secret": "s"},
		},
		{
			name: "nested calls merge headers",
			ctx: WithRequestHeaders(
				WithRequestHeaders(context.Background(), map[string]string{"X-A": "a"}),
				map[string]string{"X-B": "b"}),
			want: map[string]string{"X-A": "a", "X-B": "b"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodPatch, http.MethodDelete} {
				var got http.Header
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					got = r.Header.Clone()
					w.WriteHeader(http.StatusOK)
				}))
				c := NewClient(ClientConfig{BaseURL: srv.URL, Headers: tt.static})

				var err error
				switch method {
				case http.MethodGet:
					_, err = c.Get(tt.ctx, "/x", nil)
				case http.MethodPost:
					_, err = c.Post(tt.ctx, "/x", map[string]any{})
				case http.MethodPatch:
					_, err = c.Patch(tt.ctx, "/x", map[string]any{})
				case http.MethodDelete:
					err = c.Delete(tt.ctx, "/x")
				}
				srv.Close()
				if err != nil {
					t.Fatalf("%s: unexpected error: %v", method, err)
				}
				for k, v := range tt.want {
					if got.Get(k) != v {
						t.Errorf("%s: header %s = %q, want %q", method, k, got.Get(k), v)
					}
				}
				for _, k := range tt.wantNot {
					if got.Get(k) != "" {
						t.Errorf("%s: header %s unexpectedly set to %q", method, k, got.Get(k))
					}
				}
			}
		})
	}
}
