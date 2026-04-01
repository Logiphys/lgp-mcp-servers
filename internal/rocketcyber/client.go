package rocketcyber

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/Logiphys/lgp-mcp/pkg/apihelper"
	"github.com/Logiphys/lgp-mcp/pkg/resilience"
)

// Config holds RocketCyber API credentials and settings.
type Config struct {
	APIKey  string
	Region  string // default "us"
	BaseURL string // optional override
}

// PageInfo holds pagination metadata from a list response.
type PageInfo struct {
	TotalCount int
	Page       int
	PageSize   int
}

// Client is the RocketCyber REST API client.
type Client struct {
	http       *apihelper.Client
	middleware *resilience.Middleware
	logger     *slog.Logger
}

// NewClient creates a new RocketCyber API client.
func NewClient(cfg Config, logger *slog.Logger) *Client {
	region := cfg.Region
	if region == "" {
		region = "us"
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = fmt.Sprintf("https://api-%s.rocketcyber.com/v3", region)
	}

	httpClient := apihelper.NewClient(apihelper.ClientConfig{
		BaseURL:    baseURL,
		Timeout:    30 * time.Second,
		MaxRetries: 3,
		UserAgent:  "lgp-mcp/rocketcyber",
		Headers: map[string]string{
			"Authorization": fmt.Sprintf("Bearer %s", cfg.APIKey),
			"Content-Type":  "application/json",
		},
	})

	// Rate limit: 1000 req/hr ≈ 16/min; circuit breaker with 5 failure threshold.
	mw := resilience.NewMiddleware(resilience.Config{
		RateLimit:        1000,
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
		return nil, fmt.Errorf("rocketcyber: failed to parse response: %w", err)
	}
	return result, nil
}

// GetList performs a GET request and extracts the data array plus pagination info.
// The response format is {"data": [...], "totalCount": N} or a bare array.
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

	// Try object with "data" field.
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil, nil, fmt.Errorf("rocketcyber: failed to parse list response: %w", err)
	}

	data, ok := obj["data"]
	if !ok {
		// Single object wrapped — return as single-element list with no pagination.
		return []any{obj}, nil, nil
	}

	items, ok := data.([]any)
	if !ok {
		return nil, nil, fmt.Errorf("rocketcyber: unexpected data field type in list response")
	}

	pi := extractPageInfo(obj, params)
	return items, pi, nil
}

// Post performs a POST request and returns the parsed JSON response as a map.
func (c *Client) Post(ctx context.Context, path string, body any) (map[string]any, error) {
	result, err := c.middleware.Execute(ctx, func() (any, error) {
		raw, err := c.http.Post(ctx, path, body)
		if err != nil {
			return nil, fmt.Errorf("rocketcyber: POST %s: %w", path, err)
		}
		return raw, nil
	})
	if err != nil {
		return nil, err
	}

	raw := result.([]byte)
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("rocketcyber: failed to parse POST response: %w", err)
	}
	return out, nil
}

// Delete performs a DELETE request.
func (c *Client) Delete(ctx context.Context, path string) error {
	_, err := c.middleware.Execute(ctx, func() (any, error) {
		if err := c.http.Delete(ctx, path); err != nil {
			return nil, fmt.Errorf("rocketcyber: DELETE %s: %w", path, err)
		}
		return nil, nil
	})
	return err
}

// TestConnection verifies connectivity by calling GET /account.
func (c *Client) TestConnection(ctx context.Context) error {
	_, err := c.Get(ctx, "/account", nil)
	if err != nil {
		return fmt.Errorf("rocketcyber: connection test failed: %w", err)
	}
	return nil
}

// doGet performs the actual GET call through the middleware.
func (c *Client) doGet(ctx context.Context, path string, params map[string]string) ([]byte, error) {
	result, err := c.middleware.Execute(ctx, func() (any, error) {
		raw, err := c.http.Get(ctx, path, params)
		if err != nil {
			return nil, fmt.Errorf("rocketcyber: GET %s: %w", path, err)
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
	if v, ok := obj["totalCount"]; ok {
		switch n := v.(type) {
		case float64:
			totalCount = int(n)
		case int:
			totalCount = n
		}
	}

	page := 1
	if p, ok := params["page"]; ok {
		fmt.Sscanf(p, "%d", &page) //nolint:errcheck
	}

	pageSize := 0
	if ps, ok := params["pageSize"]; ok {
		fmt.Sscanf(ps, "%d", &pageSize) //nolint:errcheck
	}

	return &PageInfo{
		TotalCount: totalCount,
		Page:       page,
		PageSize:   pageSize,
	}
}
