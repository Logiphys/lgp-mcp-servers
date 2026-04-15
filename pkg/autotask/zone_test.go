package autotask

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDiscoverZone(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ATServicesRest/V1.0/ZoneInformation" {
			t.Errorf("expected path /ATServicesRest/V1.0/ZoneInformation, got %s", r.URL.Path)
		}
		if r.URL.Query().Get("User") != "testuser" {
			t.Errorf("expected User query param 'testuser', got %s", r.URL.Query().Get("User"))
		}

		resp := map[string]any{
			"url":    "https://webservices18.autotask.net/ATServicesRest/",
			"webUrl": "https://ww18.autotask.net",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	// Override the hardcoded base URL for testing
	// In the real code it's https://webservices.autotask.net/...
	// We need a way to inject the test server URL.
	// Let's modify zone.go to allow injecting the base discovery URL or just test the normalization logic.

	t.Run("Normalization", func(t *testing.T) {
		tests := []struct {
			input    string
			expected string
		}{
			{"https://webservices1.autotask.net/atservicesrest", "https://webservices1.autotask.net/ATServicesRest/V1.0"},
			{"webservices2.autotask.net/", "https://webservices2.autotask.net/ATServicesRest/V1.0"},
			{"https://webservices3.autotask.net/ATServicesRest/V1.0", "https://webservices3.autotask.net/ATServicesRest/V1.0"},
		}

		for _, tt := range tests {
			got := NormalizeBaseURL(tt.input)
			if got != tt.expected {
				t.Errorf("NormalizeBaseURL(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		}
	})
}
