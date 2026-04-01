package dattormm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"math/rand/v2"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"

	"github.com/Logiphys/lgp-mcp/pkg/resilience"
)

// platformURLs maps Datto RMM platform names to their API base URLs.
var platformURLs = map[string]string{
	"pinotage":  "https://pinotage-api.centrastage.net",
	"merlot":    "https://merlot-api.centrastage.net",
	"concord":   "https://concord-api.centrastage.net",
	"vidal":     "https://vidal-api.centrastage.net",
	"zinfandel": "https://zinfandel-api.centrastage.net",
	"syrah":     "https://syrah-api.centrastage.net",
}

const (
	apiBasePath         = "/api/v2"
	tokenRefreshBuffer  = 5 * time.Minute
	defaultPlatform     = "merlot"
	defaultTimeout      = 30 * time.Second
	defaultMaxRetries   = 3
	defaultRateLimit    = 1000 // requests per hour, conservative
)

// Config holds Datto RMM API credentials and settings.
type Config struct {
	APIKey    string
	APISecret string
	Platform  string // default "merlot"
	BaseURL   string // optional override
}

// PageInfo holds pagination metadata from a Datto RMM list response.
type PageInfo struct {
	Page       int
	TotalPages int
	Count      int
}

// tokenManager handles OAuth2 password-grant token lifecycle for Datto RMM.
type tokenManager struct {
	apiKey    string
	apiSecret string
	tokenURL  string
	mu        sync.Mutex
	token     string
	expiry    time.Time
	group     singleflight.Group
}

func newTokenManager(apiKey, apiSecret, baseURL string) *tokenManager {
	return &tokenManager{
		apiKey:    apiKey,
		apiSecret: apiSecret,
		tokenURL:  baseURL + "/auth/oauth/token",
	}
}

// getToken returns a cached token or fetches a fresh one.
func (t *tokenManager) getToken(ctx context.Context) (string, error) {
	t.mu.Lock()
	if t.token != "" && time.Now().Before(t.expiry.Add(-tokenRefreshBuffer)) {
		tok := t.token
		t.mu.Unlock()
		return tok, nil
	}
	t.mu.Unlock()

	result, err, _ := t.group.Do("token", func() (any, error) {
		return t.fetchToken(ctx)
	})
	if err != nil {
		return "", err
	}
	return result.(string), nil
}

// fetchToken performs the Datto RMM password-grant OAuth2 token request.
// The request uses Basic auth (apiKey:apiSecret) and sends the credentials
// again as form body fields per the Datto RMM API specification.
func (t *tokenManager) fetchToken(ctx context.Context) (string, error) {
	data := url.Values{
		"grant_type": {"password"},
		"username":   {t.apiKey},
		"password":   {t.apiSecret},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.tokenURL,
		strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("creating token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(t.apiKey, t.apiSecret)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token request returned HTTP %d: %s", resp.StatusCode, string(body))
	}

	var tok struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &tok); err != nil {
		return "", fmt.Errorf("parsing token response: %w", err)
	}

	t.mu.Lock()
	t.token = tok.AccessToken
	t.expiry = time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second)
	t.mu.Unlock()

	return tok.AccessToken, nil
}

// Client is the Datto RMM REST API client.
type Client struct {
	baseURL    string
	httpClient *http.Client
	tokenMgr   *tokenManager
	middleware *resilience.Middleware
	logger     *slog.Logger
	maxRetries int
}

// NewClient creates a new Datto RMM API client.
func NewClient(cfg Config, logger *slog.Logger) *Client {
	platform := cfg.Platform
	if platform == "" {
		platform = defaultPlatform
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		if u, ok := platformURLs[platform]; ok {
			baseURL = u
		} else {
			baseURL = platformURLs[defaultPlatform]
		}
	}

	mw := resilience.NewMiddleware(resilience.Config{
		RateLimit:        defaultRateLimit,
		FailureThreshold: 5,
		Cooldown:         30 * time.Second,
		SuccessThreshold: 3,
		Compact:          false,
	})

	return &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: defaultTimeout},
		tokenMgr:   newTokenManager(cfg.APIKey, cfg.APISecret, baseURL),
		middleware: mw,
		logger:     logger,
		maxRetries: defaultMaxRetries,
	}
}

// doRequest executes an authenticated HTTP request with retry logic for
// transient errors (429, 5xx). Bearer token is injected per-request.
func (c *Client) doRequest(ctx context.Context, method, path string, body any) ([]byte, error) {
	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshalling request body: %w", err)
		}
	}

	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(math.Pow(2, float64(attempt-1))) * 100 * time.Millisecond
			jitter := time.Duration(rand.IntN(int(backoff/2 + 1)))
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff + jitter):
			}
		}

		tok, err := c.tokenMgr.getToken(ctx)
		if err != nil {
			return nil, fmt.Errorf("obtaining access token: %w", err)
		}

		var reqBody io.Reader
		if bodyBytes != nil {
			reqBody = bytes.NewReader(bodyBytes)
		}
		req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
		if err != nil {
			return nil, fmt.Errorf("creating request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+tok)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "lgp-mcp/dattormm")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			if ctx.Err() != nil {
				return nil, lastErr
			}
			continue
		}

		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return respBody, nil
		}

		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
			continue
		}

		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

// Get performs a GET request and returns the parsed JSON response.
func (c *Client) Get(ctx context.Context, path string, params map[string]string) (map[string]any, error) {
	result, err := c.middleware.Execute(ctx, func() (any, error) {
		fullPath := buildPath(path, params)
		body, err := c.doRequest(ctx, http.MethodGet, fullPath, nil)
		if err != nil {
			return nil, err
		}
		var resp map[string]any
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, fmt.Errorf("parsing response: %w", err)
		}
		return resp, nil
	})
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}
	return result.(map[string]any), nil
}

// GetRaw performs a GET request and returns the raw response bytes.
func (c *Client) GetRaw(ctx context.Context, path string, params map[string]string) ([]byte, error) {
	result, err := c.middleware.Execute(ctx, func() (any, error) {
		fullPath := buildPath(path, params)
		return c.doRequest(ctx, http.MethodGet, fullPath, nil)
	})
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}
	return result.([]byte), nil
}

// Post performs a POST request with a JSON body and returns the parsed response.
func (c *Client) Post(ctx context.Context, path string, body any) (map[string]any, error) {
	result, err := c.middleware.Execute(ctx, func() (any, error) {
		respBody, err := c.doRequest(ctx, http.MethodPost, path, body)
		if err != nil {
			return nil, err
		}
		var resp map[string]any
		if err := json.Unmarshal(respBody, &resp); err != nil {
			return nil, fmt.Errorf("parsing response: %w", err)
		}
		return resp, nil
	})
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}
	return result.(map[string]any), nil
}

// Patch performs a PATCH request with a JSON body and returns the parsed response.
func (c *Client) Patch(ctx context.Context, path string, body any) (map[string]any, error) {
	result, err := c.middleware.Execute(ctx, func() (any, error) {
		respBody, err := c.doRequest(ctx, http.MethodPatch, path, body)
		if err != nil {
			return nil, err
		}
		var resp map[string]any
		if err := json.Unmarshal(respBody, &resp); err != nil {
			return nil, fmt.Errorf("parsing response: %w", err)
		}
		return resp, nil
	})
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}
	return result.(map[string]any), nil
}

// Put performs a PUT request with a JSON body. Used for operations like move-device.
func (c *Client) Put(ctx context.Context, path string, body any) error {
	_, err := c.middleware.Execute(ctx, func() (any, error) {
		_, err := c.doRequest(ctx, http.MethodPut, path, body)
		return nil, err
	})
	return err
}

// Delete performs a DELETE request.
func (c *Client) Delete(ctx context.Context, path string) error {
	_, err := c.middleware.Execute(ctx, func() (any, error) {
		_, err := c.doRequest(ctx, http.MethodDelete, path, nil)
		return nil, err
	})
	return err
}

// GetList performs a paginated GET and returns all items along with page metadata.
// Datto RMM list responses contain a top-level array key (varies per endpoint)
// and a "pageDetails" object. This method extracts the first array value found.
func (c *Client) GetList(ctx context.Context, path string, params map[string]string) ([]any, *PageInfo, error) {
	result, err := c.middleware.Execute(ctx, func() (any, error) {
		fullPath := buildPath(path, params)
		respBody, err := c.doRequest(ctx, http.MethodGet, fullPath, nil)
		if err != nil {
			return nil, err
		}

		var raw map[string]any
		if err := json.Unmarshal(respBody, &raw); err != nil {
			return nil, fmt.Errorf("parsing list response: %w", err)
		}

		pi := extractPageInfo(raw)
		items := extractItems(raw)

		return &getListResult{items: items, pageInfo: pi}, nil
	})
	if err != nil {
		return nil, nil, err
	}
	if result == nil {
		return nil, nil, nil
	}
	lr := result.(*getListResult)
	return lr.items, lr.pageInfo, nil
}

// TestConnection verifies connectivity by fetching the account endpoint.
func (c *Client) TestConnection(ctx context.Context) error {
	_, err := c.middleware.Execute(ctx, func() (any, error) {
		body, err := c.doRequest(ctx, http.MethodGet, apiBasePath+"/account", nil)
		if err != nil {
			return nil, fmt.Errorf("connection test failed: %w", err)
		}
		return body, nil
	})
	return err
}

// getListResult is an internal carrier for GetList results through middleware.Execute.
type getListResult struct {
	items    []any
	pageInfo *PageInfo
}

// buildPath appends query parameters to a path string.
func buildPath(path string, params map[string]string) string {
	if len(params) == 0 {
		return path
	}
	q := url.Values{}
	for k, v := range params {
		q.Set(k, v)
	}
	return path + "?" + q.Encode()
}

// extractPageInfo reads the "pageDetails" object from a Datto RMM response map.
func extractPageInfo(raw map[string]any) *PageInfo {
	pd, ok := raw["pageDetails"].(map[string]any)
	if !ok {
		return &PageInfo{}
	}
	return &PageInfo{
		Page:       toInt(pd["page"]),
		TotalPages: toInt(pd["totalPages"]),
		Count:      toInt(pd["count"]),
	}
}

// extractItems finds the first array value in a Datto RMM response map,
// skipping the "pageDetails" key which is always an object.
func extractItems(raw map[string]any) []any {
	for k, v := range raw {
		if k == "pageDetails" {
			continue
		}
		if arr, ok := v.([]any); ok {
			return arr
		}
	}
	return nil
}

// toInt converts a JSON number (float64) to int.
func toInt(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	}
	return 0
}
