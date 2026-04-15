package autotask

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ZoneInfo contains the tenant-specific REST and web URLs discovered via the
// Autotask ZoneInformation endpoint.
type ZoneInfo struct {
	BaseURL string // normalized REST base (…/ATServicesRest/)
	WebURL  string // web UI base (e.g., https://ww18.autotask.net/)
}

// DiscoverZone performs zone discovery and returns both REST base and web UI base URLs.
func DiscoverZone(ctx context.Context, username string) (ZoneInfo, error) {
	user := strings.TrimSpace(username)
	if user == "" {
		return ZoneInfo{}, fmt.Errorf("zone discovery requires username")
	}
	base := "https://webservices.autotask.net/ATServicesRest/V1.0/ZoneInformation"
	u, err := url.Parse(base)
	if err != nil {
		return ZoneInfo{}, fmt.Errorf("parsing base discovery URL: %w", err)
	}
	q := u.Query()
	// The API expects "User" as the parameter name (case-sensitive per docs).
	q.Set("User", user)
	u.RawQuery = q.Encode()

	hc := &http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return ZoneInfo{}, fmt.Errorf("creating zone discovery request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := hc.Do(req)
	if err != nil {
		return ZoneInfo{}, fmt.Errorf("zone discovery request failed: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return ZoneInfo{}, fmt.Errorf("zone discovery failed with status %d", resp.StatusCode)
	}

	var m map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		return ZoneInfo{}, fmt.Errorf("parsing zone discovery response: %w", err)
	}

	raw, _ := m["url"].(string)
	if strings.TrimSpace(raw) == "" {
		return ZoneInfo{}, fmt.Errorf("zone discovery response missing 'url'")
	}

	web := ""
	if s, ok := m["webUrl"].(string); ok {
		web = strings.TrimSpace(s)
		if web != "" && !strings.HasSuffix(web, "/") {
			web += "/"
		}
	}

	return ZoneInfo{
		BaseURL: NormalizeBaseURL(raw),
		WebURL:  web,
	}, nil
}

// NormalizeBaseURL ensures the upstream base URL is properly formed and uses the
// case-sensitive path segment "/ATServicesRest/V1.0" expected by many tenants.
func NormalizeBaseURL(base string) string {
	s := strings.TrimSpace(base)
	if s == "" {
		return s
	}

	if !strings.HasPrefix(s, "http://") && !strings.HasPrefix(s, "https://") {
		s = "https://" + s
	}

	u, err := url.Parse(s)
	if err != nil {
		// Fallback: just enforce path casing heuristically.
		return forceATServicesRestPath(s)
	}
	// Normalize path to exactly "/ATServicesRest/V1.0" (trailing slash ensured).
	u.Path = "/ATServicesRest/V1.0"

	// Remove query/fragment if any were present.
	u.RawQuery = ""
	u.Fragment = ""
	return u.String()
}

func forceATServicesRestPath(s string) string {
	// Replace any case-insensitive /atservicesrest segment with /ATServicesRest/V1.0
	lower := strings.ToLower(s)
	idx := strings.Index(lower, "/atservicesrest")
	if idx >= 0 {
		prefix := s[:idx]
		return strings.TrimSuffix(prefix, "/") + "/ATServicesRest/V1.0"
	}
	// No segment found; append it (keeping scheme/host as-is if present)
	return strings.TrimSuffix(s, "/") + "/ATServicesRest/V1.0"
}
