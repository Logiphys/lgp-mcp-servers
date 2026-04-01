package dattormm

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// newTestLogger returns a discard logger for tests.
func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// tokenServer returns an httptest.Server that handles OAuth2 password-grant
// token requests, recording each call so tests can assert on call counts.
func tokenServer(t *testing.T, calls *atomic.Int32) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/auth/oauth/token" {
			http.NotFound(w, r)
			return
		}
		calls.Add(1)

		// Verify Basic auth header.
		user, pass, ok := r.BasicAuth()
		if !ok || user != "testkey" || pass != "testsecret" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		// Verify form body.
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		if r.FormValue("grant_type") != "password" ||
			r.FormValue("username") != "testkey" ||
			r.FormValue("password") != "testsecret" {
			http.Error(w, "invalid grant params", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "test-bearer-token",
			"token_type":   "Bearer",
			"expires_in":   3600,
		})
	}))
}

// newTestClient creates a Client pointing at the provided base URL.
func newTestClient(t *testing.T, baseURL string) *Client {
	t.Helper()
	cfg := Config{
		APIKey:    "testkey",
		APISecret: "testsecret",
		BaseURL:   baseURL,
	}
	return NewClient(cfg, newTestLogger())
}

// TestClient_TokenFetch verifies the password-grant request format:
// Basic auth header + form body with grant_type=password.
func TestClient_TokenFetch(t *testing.T) {
	var calls atomic.Int32
	ts := tokenServer(t, &calls)
	defer ts.Close()

	mgr := newTokenManager("testkey", "testsecret", ts.URL)
	tok, err := mgr.getToken(context.Background())
	if err != nil {
		t.Fatalf("getToken: %v", err)
	}
	if tok != "test-bearer-token" {
		t.Errorf("expected 'test-bearer-token', got %q", tok)
	}
	if calls.Load() != 1 {
		t.Errorf("expected 1 token fetch, got %d", calls.Load())
	}
}

// TestClient_TokenCaching verifies that a cached token is reused and the
// token endpoint is not called again before expiry.
func TestClient_TokenCaching(t *testing.T) {
	var calls atomic.Int32
	ts := tokenServer(t, &calls)
	defer ts.Close()

	mgr := newTokenManager("testkey", "testsecret", ts.URL)

	tok1, err := mgr.getToken(context.Background())
	if err != nil {
		t.Fatalf("first getToken: %v", err)
	}
	tok2, err := mgr.getToken(context.Background())
	if err != nil {
		t.Fatalf("second getToken: %v", err)
	}
	if tok1 != tok2 {
		t.Errorf("expected same token, got %q vs %q", tok1, tok2)
	}
	if calls.Load() != 1 {
		t.Errorf("expected 1 token fetch (cached), got %d", calls.Load())
	}
}

// TestClient_TokenExpiry verifies that an expired token triggers a refresh.
func TestClient_TokenExpiry(t *testing.T) {
	var calls atomic.Int32
	ts := tokenServer(t, &calls)
	defer ts.Close()

	mgr := newTokenManager("testkey", "testsecret", ts.URL)
	// Pre-seed an expired token.
	mgr.mu.Lock()
	mgr.token = "old-token"
	mgr.expiry = time.Now().Add(-1 * time.Minute) // already expired
	mgr.mu.Unlock()

	tok, err := mgr.getToken(context.Background())
	if err != nil {
		t.Fatalf("getToken: %v", err)
	}
	if tok != "test-bearer-token" {
		t.Errorf("expected refreshed token, got %q", tok)
	}
	if calls.Load() != 1 {
		t.Errorf("expected 1 token fetch, got %d", calls.Load())
	}
}

// apiServer builds a combined test server handling both token and API paths.
func apiServer(t *testing.T, tokenCalls *atomic.Int32, apiHandler http.HandlerFunc) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/auth/oauth/token", func(w http.ResponseWriter, r *http.Request) {
		tokenCalls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "test-bearer-token",
			"token_type":   "Bearer",
			"expires_in":   3600,
		})
	})
	mux.HandleFunc("/", apiHandler)
	return httptest.NewServer(mux)
}

// TestClient_Get verifies that Get sends a Bearer token and parses the response.
func TestClient_Get(t *testing.T) {
	var tokenCalls atomic.Int32
	ts := apiServer(t, &tokenCalls, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != apiBasePath+"/account" {
			http.NotFound(w, r)
			return
		}
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-bearer-token" {
			http.Error(w, "bad auth: "+auth, http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"uid":  "acc-123",
			"name": "Test Account",
		})
	})
	defer ts.Close()

	client := newTestClient(t, ts.URL)
	resp, err := client.Get(context.Background(), apiBasePath+"/account", nil)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if resp["uid"] != "acc-123" {
		t.Errorf("expected uid='acc-123', got %v", resp["uid"])
	}
}

// TestClient_GetWithParams verifies that query parameters are appended correctly.
func TestClient_GetWithParams(t *testing.T) {
	var tokenCalls atomic.Int32
	ts := apiServer(t, &tokenCalls, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("page") != "2" || r.URL.Query().Get("max") != "50" {
			http.Error(w, "missing params", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	defer ts.Close()

	client := newTestClient(t, ts.URL)
	resp, err := client.Get(context.Background(), apiBasePath+"/devices", map[string]string{
		"page": "2",
		"max":  "50",
	})
	if err != nil {
		t.Fatalf("Get with params: %v", err)
	}
	if resp["ok"] != true {
		t.Errorf("expected ok=true, got %v", resp["ok"])
	}
}

// TestClient_GetRaw verifies that GetRaw returns raw bytes unchanged.
func TestClient_GetRaw(t *testing.T) {
	var tokenCalls atomic.Int32
	rawPayload := `{"stdout":"hello world"}`
	ts := apiServer(t, &tokenCalls, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(rawPayload))
	})
	defer ts.Close()

	client := newTestClient(t, ts.URL)
	data, err := client.GetRaw(context.Background(), apiBasePath+"/job/123/stdout", nil)
	if err != nil {
		t.Fatalf("GetRaw: %v", err)
	}
	if string(data) != rawPayload {
		t.Errorf("expected %q, got %q", rawPayload, string(data))
	}
}

// TestClient_GetList verifies list parsing including pageDetails extraction.
func TestClient_GetList(t *testing.T) {
	tests := []struct {
		name           string
		responseBody   map[string]any
		wantCount      int
		wantPage       int
		wantTotalPages int
		wantItems      int
	}{
		{
			name: "devices list with pageDetails",
			responseBody: map[string]any{
				"devices": []any{
					map[string]any{"uid": "dev-1"},
					map[string]any{"uid": "dev-2"},
					map[string]any{"uid": "dev-3"},
				},
				"pageDetails": map[string]any{
					"page":       float64(1),
					"totalPages": float64(3),
					"count":      float64(130),
				},
			},
			wantCount:      130,
			wantPage:       1,
			wantTotalPages: 3,
			wantItems:      3,
		},
		{
			name: "sites list single page",
			responseBody: map[string]any{
				"sites": []any{
					map[string]any{"uid": "site-1"},
				},
				"pageDetails": map[string]any{
					"page":       float64(1),
					"totalPages": float64(1),
					"count":      float64(1),
				},
			},
			wantCount:      1,
			wantPage:       1,
			wantTotalPages: 1,
			wantItems:      1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var tokenCalls atomic.Int32
			ts := apiServer(t, &tokenCalls, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(tt.responseBody)
			})
			defer ts.Close()

			client := newTestClient(t, ts.URL)
			items, pi, err := client.GetList(context.Background(), apiBasePath+"/devices", nil)
			if err != nil {
				t.Fatalf("GetList: %v", err)
			}
			if len(items) != tt.wantItems {
				t.Errorf("expected %d items, got %d", tt.wantItems, len(items))
			}
			if pi.Count != tt.wantCount {
				t.Errorf("expected count=%d, got %d", tt.wantCount, pi.Count)
			}
			if pi.Page != tt.wantPage {
				t.Errorf("expected page=%d, got %d", tt.wantPage, pi.Page)
			}
			if pi.TotalPages != tt.wantTotalPages {
				t.Errorf("expected totalPages=%d, got %d", tt.wantTotalPages, pi.TotalPages)
			}
		})
	}
}

// TestClient_Post verifies POST requests send a JSON body and parse the response.
func TestClient_Post(t *testing.T) {
	var tokenCalls atomic.Int32
	ts := apiServer(t, &tokenCalls, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "wrong method", http.StatusMethodNotAllowed)
			return
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "bad body", http.StatusBadRequest)
			return
		}
		if body["jobType"] != "quickJob" {
			http.Error(w, "unexpected body", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"jobUid": "job-456"})
	})
	defer ts.Close()

	client := newTestClient(t, ts.URL)
	resp, err := client.Post(context.Background(), apiBasePath+"/job/quick", map[string]any{
		"jobType": "quickJob",
	})
	if err != nil {
		t.Fatalf("Post: %v", err)
	}
	if resp["jobUid"] != "job-456" {
		t.Errorf("expected jobUid='job-456', got %v", resp["jobUid"])
	}
}

// TestClient_Patch verifies PATCH requests send a JSON body and parse the response.
func TestClient_Patch(t *testing.T) {
	var tokenCalls atomic.Int32
	ts := apiServer(t, &tokenCalls, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			http.Error(w, "wrong method", http.StatusMethodNotAllowed)
			return
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "bad body", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"updated": true})
	})
	defer ts.Close()

	client := newTestClient(t, ts.URL)
	resp, err := client.Patch(context.Background(), apiBasePath+"/site/site-1", map[string]any{
		"name": "Updated Site",
	})
	if err != nil {
		t.Fatalf("Patch: %v", err)
	}
	if resp["updated"] != true {
		t.Errorf("expected updated=true, got %v", resp["updated"])
	}
}

// TestClient_Put verifies PUT requests are sent with a JSON body.
func TestClient_Put(t *testing.T) {
	var tokenCalls atomic.Int32
	var gotMethod string
	ts := apiServer(t, &tokenCalls, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		if r.Method != http.MethodPut {
			http.Error(w, "wrong method", http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	defer ts.Close()

	client := newTestClient(t, ts.URL)
	err := client.Put(context.Background(), apiBasePath+"/device/dev-1/site", map[string]any{
		"siteUid": "site-2",
	})
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	if gotMethod != http.MethodPut {
		t.Errorf("expected PUT method, got %q", gotMethod)
	}
}

// TestClient_Delete verifies DELETE requests are sent correctly.
func TestClient_Delete(t *testing.T) {
	var tokenCalls atomic.Int32
	var gotMethod string
	ts := apiServer(t, &tokenCalls, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		if r.Method != http.MethodDelete {
			http.Error(w, "wrong method", http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	defer ts.Close()

	client := newTestClient(t, ts.URL)
	err := client.Delete(context.Background(), apiBasePath+"/variable/var-1")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("expected DELETE method, got %q", gotMethod)
	}
}

// TestClient_PlatformURLs verifies all 6 platform names resolve to their base URLs.
func TestClient_PlatformURLs(t *testing.T) {
	tests := []struct {
		platform string
		wantHost string
	}{
		{"pinotage", "pinotage-api.centrastage.net"},
		{"merlot", "merlot-api.centrastage.net"},
		{"concord", "concord-api.centrastage.net"},
		{"vidal", "vidal-api.centrastage.net"},
		{"zinfandel", "zinfandel-api.centrastage.net"},
		{"syrah", "syrah-api.centrastage.net"},
	}

	for _, tt := range tests {
		t.Run(tt.platform, func(t *testing.T) {
			cfg := Config{
				APIKey:    "k",
				APISecret: "s",
				Platform:  tt.platform,
			}
			c := NewClient(cfg, newTestLogger())
			if !strings.Contains(c.baseURL, tt.wantHost) {
				t.Errorf("platform %q: expected baseURL to contain %q, got %q",
					tt.platform, tt.wantHost, c.baseURL)
			}
		})
	}
}

// TestClient_DefaultPlatform verifies that an empty Platform defaults to "merlot".
func TestClient_DefaultPlatform(t *testing.T) {
	c := NewClient(Config{APIKey: "k", APISecret: "s"}, newTestLogger())
	if !strings.Contains(c.baseURL, "merlot-api.centrastage.net") {
		t.Errorf("expected default merlot URL, got %q", c.baseURL)
	}
}

// TestClient_BaseURLOverride verifies that BaseURL overrides the platform lookup.
func TestClient_BaseURLOverride(t *testing.T) {
	customURL := "https://custom.example.com"
	c := NewClient(Config{
		APIKey:    "k",
		APISecret: "s",
		Platform:  "merlot",
		BaseURL:   customURL,
	}, newTestLogger())
	if c.baseURL != customURL {
		t.Errorf("expected baseURL=%q, got %q", customURL, c.baseURL)
	}
}

// TestClient_TestConnection verifies TestConnection calls /api/v2/account.
func TestClient_TestConnection(t *testing.T) {
	var tokenCalls atomic.Int32
	var gotPath string
	ts := apiServer(t, &tokenCalls, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"uid": "acc-1"})
	})
	defer ts.Close()

	client := newTestClient(t, ts.URL)
	if err := client.TestConnection(context.Background()); err != nil {
		t.Fatalf("TestConnection: %v", err)
	}
	if gotPath != apiBasePath+"/account" {
		t.Errorf("expected path %q, got %q", apiBasePath+"/account", gotPath)
	}
}

// TestClient_TestConnection_Failure verifies TestConnection returns an error on failure.
func TestClient_TestConnection_Failure(t *testing.T) {
	var tokenCalls atomic.Int32
	ts := apiServer(t, &tokenCalls, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	})
	defer ts.Close()

	// Use maxRetries=0 so test is fast.
	client := newTestClient(t, ts.URL)
	client.maxRetries = 0
	if err := client.TestConnection(context.Background()); err == nil {
		t.Fatal("expected error, got nil")
	}
}

// TestClient_Retry verifies that transient 5xx errors are retried.
func TestClient_Retry(t *testing.T) {
	var tokenCalls atomic.Int32
	var requestCount atomic.Int32
	ts := apiServer(t, &tokenCalls, func(w http.ResponseWriter, r *http.Request) {
		n := requestCount.Add(1)
		if n < 3 {
			http.Error(w, "server error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	defer ts.Close()

	client := newTestClient(t, ts.URL)
	client.maxRetries = 3
	resp, err := client.Get(context.Background(), apiBasePath+"/account", nil)
	if err != nil {
		t.Fatalf("Get with retry: %v", err)
	}
	if resp["ok"] != true {
		t.Errorf("expected ok=true after retry, got %v", resp["ok"])
	}
	if requestCount.Load() < 3 {
		t.Errorf("expected at least 3 attempts, got %d", requestCount.Load())
	}
}

// TestClient_ContextCancellation verifies the client respects context cancellation.
func TestClient_ContextCancellation(t *testing.T) {
	var tokenCalls atomic.Int32
	ts := apiServer(t, &tokenCalls, func(w http.ResponseWriter, r *http.Request) {
		// Delay longer than the context deadline.
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	})
	defer ts.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	client := newTestClient(t, ts.URL)
	client.maxRetries = 0
	_, err := client.Get(ctx, apiBasePath+"/account", nil)
	if err == nil {
		t.Fatal("expected error due to context cancellation, got nil")
	}
}

// TestClient_BearerTokenHeader verifies the Authorization header is set correctly.
func TestClient_BearerTokenHeader(t *testing.T) {
	var tokenCalls atomic.Int32
	var gotAuth string
	ts := apiServer(t, &tokenCalls, func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{})
	})
	defer ts.Close()

	client := newTestClient(t, ts.URL)
	_, err := client.Get(context.Background(), apiBasePath+"/account", nil)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if gotAuth != "Bearer test-bearer-token" {
		t.Errorf("expected Authorization header 'Bearer test-bearer-token', got %q", gotAuth)
	}
}
