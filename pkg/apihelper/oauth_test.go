package apihelper

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestTokenManager_FetchesToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != "key" || pass != "secret" {
			w.WriteHeader(401)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "tok123", "token_type": "Bearer", "expires_in": 3600})
	}))
	defer srv.Close()
	tm := NewTokenManager(OAuth2Config{TokenURL: srv.URL + "/token", ClientID: "key", ClientSecret: "secret"})
	tok, err := tm.Token(context.Background())
	if err != nil {
		t.Fatalf("Token() failed: %v", err)
	}
	if tok != "tok123" {
		t.Errorf("token = %q, want tok123", tok)
	}
}

func TestTokenManager_CachesToken(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "tok", "expires_in": 3600})
	}))
	defer srv.Close()
	tm := NewTokenManager(OAuth2Config{TokenURL: srv.URL, ClientID: "k", ClientSecret: "s"})
	_, _ = tm.Token(context.Background())
	_, _ = tm.Token(context.Background())
	if calls.Load() != 1 {
		t.Errorf("token endpoint called %d times, want 1", calls.Load())
	}
}

func TestTokenManager_RefreshesExpired(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "tok", "expires_in": 1})
	}))
	defer srv.Close()
	tm := NewTokenManager(OAuth2Config{TokenURL: srv.URL, ClientID: "k", ClientSecret: "s"})
	_, _ = tm.Token(context.Background())
	tm.mu.Lock()
	tm.expiry = time.Now().Add(-1 * time.Minute)
	tm.mu.Unlock()
	_, _ = tm.Token(context.Background())
	if calls.Load() != 2 {
		t.Errorf("calls = %d, want 2", calls.Load())
	}
}

func TestTokenManager_DeduplicatesConcurrent(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		time.Sleep(50 * time.Millisecond)
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "tok", "expires_in": 3600})
	}))
	defer srv.Close()
	tm := NewTokenManager(OAuth2Config{TokenURL: srv.URL, ClientID: "k", ClientSecret: "s"})
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = tm.Token(context.Background())
		}()
	}
	wg.Wait()
	if calls.Load() != 1 {
		t.Errorf("concurrent calls = %d, want 1", calls.Load())
	}
}
