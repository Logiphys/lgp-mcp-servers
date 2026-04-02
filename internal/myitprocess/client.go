package myitprocess

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/Logiphys/lgp-mcp-servers/pkg/apihelper"
	"github.com/Logiphys/lgp-mcp-servers/pkg/resilience"
)

// Config holds MyITProcess API credentials and settings.
type Config struct {
	APIKey string
}

// PageInfo holds pagination metadata from a list response.
type PageInfo struct {
	TotalCount int
	Page       int
	PageSize   int
}

// Client is the MyITProcess REST API client.
type Client struct {
	http       *apihelper.Client
	middleware *resilience.Middleware
	logger     *slog.Logger
}

// NewClient creates a new MyITProcess API client.
func NewClient(cfg Config, logger *slog.Logger) *Client {
	httpClient := apihelper.NewClient(apihelper.ClientConfig{
		BaseURL:    "https://reporting.live.myitprocess.com/public-api/v1",
		Timeout:    30 * time.Second,
		MaxRetries: 3,
		UserAgent:  "lgp-mcp-servers/myitprocess",
		Headers: map[string]string{
			"mitp-api-key": cfg.APIKey,
			"Content-Type": "application/json",
		},
	})

	// Rate limit: 50 req/min; circuit breaker with 5 failure threshold.
	mw := resilience.NewMiddleware(resilience.Config{
		RateLimit:        50,
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
		return nil, fmt.Errorf("myitprocess: failed to parse response: %w", err)
	}
	return result, nil
}

// GetList performs a GET request and extracts the items array plus pagination info.
// The response format is {"page": 1, "pageSize": 100, "totalCount": N, "items": [...]}.
func (c *Client) GetList(ctx context.Context, path string, params map[string]string) ([]any, *PageInfo, error) {
	raw, err := c.doGet(ctx, path, params)
	if err != nil {
		return nil, nil, err
	}

	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil, nil, fmt.Errorf("myitprocess: failed to parse list response: %w", err)
	}

	items, ok := obj["items"]
	if !ok {
		// No items field — return the whole object as a single-element list.
		return []any{obj}, nil, nil
	}

	arr, ok := items.([]any)
	if !ok {
		return nil, nil, fmt.Errorf("myitprocess: unexpected items field type in list response")
	}

	pi := extractPageInfo(obj)
	return arr, pi, nil
}

// TestConnection verifies connectivity by calling GET /clients with pageSize=1.
func (c *Client) TestConnection(ctx context.Context) error {
	_, _, err := c.GetList(ctx, "/clients", map[string]string{"pageSize": "1"})
	if err != nil {
		return fmt.Errorf("myitprocess: connection test failed: %w", err)
	}
	return nil
}

// doGet performs the actual GET call through the middleware.
func (c *Client) doGet(ctx context.Context, path string, params map[string]string) ([]byte, error) {
	result, err := c.middleware.Execute(ctx, func() (any, error) {
		raw, err := c.http.Get(ctx, path, params)
		if err != nil {
			return nil, fmt.Errorf("myitprocess: GET %s: %w", path, err)
		}
		return raw, nil
	})
	if err != nil {
		return nil, err
	}
	return result.([]byte), nil
}

// extractPageInfo parses pagination metadata from a response object.
func extractPageInfo(obj map[string]any) *PageInfo {
	pi := &PageInfo{}

	if v, ok := obj["totalCount"]; ok {
		switch n := v.(type) {
		case float64:
			pi.TotalCount = int(n)
		case int:
			pi.TotalCount = n
		}
	}

	if v, ok := obj["page"]; ok {
		switch n := v.(type) {
		case float64:
			pi.Page = int(n)
		case int:
			pi.Page = n
		}
	}

	if v, ok := obj["pageSize"]; ok {
		switch n := v.(type) {
		case float64:
			pi.PageSize = int(n)
		case int:
			pi.PageSize = n
		}
	}

	return pi
}
