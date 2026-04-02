package dattouc

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/Logiphys/lgp-mcp-servers/pkg/apihelper"
	"github.com/Logiphys/lgp-mcp-servers/pkg/resilience"
)

// Config holds Datto Unified Continuity API credentials and settings.
type Config struct {
	PublicKey string
	SecretKey string
	BaseURL   string // optional override
}

// PageInfo holds pagination metadata from a list response.
type PageInfo struct {
	TotalCount int
	Page       int
	PerPage    int
}

// Client is the Datto Unified Continuity REST API client.
type Client struct {
	http       *apihelper.Client
	middleware *resilience.Middleware
	logger     *slog.Logger
}

// NewClient creates a new Datto Unified Continuity API client.
func NewClient(cfg Config, logger *slog.Logger) *Client {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://api.datto.com/v1"
	}

	credentials := base64.StdEncoding.EncodeToString([]byte(cfg.PublicKey + ":" + cfg.SecretKey))

	httpClient := apihelper.NewClient(apihelper.ClientConfig{
		BaseURL:    baseURL,
		Timeout:    30 * time.Second,
		MaxRetries: 3,
		UserAgent:  "lgp-mcp-servers/datto-uc",
		Headers: map[string]string{
			"Authorization": fmt.Sprintf("Basic %s", credentials),
			"Content-Type":  "application/json",
		},
	})

	// Rate limit: 600 req/hr (conservative); circuit breaker with 5 failure threshold.
	mw := resilience.NewMiddleware(resilience.Config{
		RateLimit:        600,
		FailureThreshold: 5,
		Cooldown:         60 * time.Second,
		SuccessThreshold: 2,
	})

	return &Client{
		http:       httpClient,
		middleware: mw,
		logger:     logger,
	}
}

// Get performs a GET request and returns the parsed JSON response as a map.
func (c *Client) Get(ctx context.Context, path string, params map[string]string) (map[string]any, error) {
	raw, err := c.doGet(ctx, path, params)
	if err != nil {
		return nil, err
	}

	var result map[string]any
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("dattouc: failed to parse response: %w", err)
	}
	return result, nil
}

// GetList performs a GET request and extracts the data array plus pagination info.
// The Datto API returns {"items": [...], "pagination": {...}} or a bare array.
func (c *Client) GetList(ctx context.Context, path string, params map[string]string) ([]any, *PageInfo, error) {
	raw, err := c.doGet(ctx, path, params)
	if err != nil {
		return nil, nil, err
	}

	// Try bare array first.
	var arr []any
	if json.Unmarshal(raw, &arr) == nil {
		return arr, nil, nil
	}

	// Try object with "items" field (Datto BCDR convention).
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil, nil, fmt.Errorf("dattouc: failed to parse list response: %w", err)
	}

	// Try "items" first, then "data".
	data, ok := obj["items"]
	if !ok {
		data, ok = obj["data"]
	}
	if !ok {
		// Single object wrapped — return as single-element list with no pagination.
		return []any{obj}, nil, nil
	}

	items, ok := data.([]any)
	if !ok {
		return nil, nil, fmt.Errorf("dattouc: unexpected data field type in list response")
	}

	pi := extractPageInfo(obj, params)
	return items, pi, nil
}

// TestConnection verifies connectivity by calling GET /bcdr/device with perPage=1.
func (c *Client) TestConnection(ctx context.Context) error {
	_, err := c.Get(ctx, "/bcdr/device", map[string]string{"perPage": "1"})
	if err != nil {
		return fmt.Errorf("dattouc: connection test failed: %w", err)
	}
	return nil
}

// doGet performs the actual GET call through the middleware.
func (c *Client) doGet(ctx context.Context, path string, params map[string]string) ([]byte, error) {
	result, err := c.middleware.Execute(ctx, func() (any, error) {
		raw, err := c.http.Get(ctx, path, params)
		if err != nil {
			return nil, fmt.Errorf("dattouc: GET %s: %w", path, err)
		}
		return raw, nil
	})
	if err != nil {
		return nil, err
	}
	return result.([]byte), nil
}

// extractPageInfo parses pagination metadata from a response object.
func extractPageInfo(obj map[string]any, params map[string]string) *PageInfo {
	totalCount := 0

	// Check for pagination sub-object (Datto BCDR style).
	if pag, ok := obj["pagination"].(map[string]any); ok {
		if v, ok := pag["count"]; ok {
			switch n := v.(type) {
			case float64:
				totalCount = int(n)
			case int:
				totalCount = n
			}
		}
		if v, ok := pag["totalCount"]; ok {
			switch n := v.(type) {
			case float64:
				totalCount = int(n)
			case int:
				totalCount = n
			}
		}
	}

	// Fallback: top-level totalCount.
	if totalCount == 0 {
		if v, ok := obj["totalCount"]; ok {
			switch n := v.(type) {
			case float64:
				totalCount = int(n)
			case int:
				totalCount = n
			}
		}
	}

	page := 1
	if p, ok := params["page"]; ok {
		fmt.Sscanf(p, "%d", &page) //nolint:errcheck
	}

	perPage := 0
	if pp, ok := params["perPage"]; ok {
		fmt.Sscanf(pp, "%d", &perPage) //nolint:errcheck
	}

	return &PageInfo{
		TotalCount: totalCount,
		Page:       page,
		PerPage:    perPage,
	}
}
