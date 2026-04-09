package dattoedr

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/Logiphys/lgp-mcp-servers/pkg/apihelper"
	"github.com/Logiphys/lgp-mcp-servers/pkg/resilience"
)

// Config holds Datto EDR API credentials and settings.
type Config struct {
	APIKey  string
	BaseURL string // required, instance-specific (e.g. https://yourorg.infocyte.com)
}

// Client is the Datto EDR (Infocyte) REST API client.
type Client struct {
	http       *apihelper.Client
	middleware *resilience.Middleware
	logger     *slog.Logger
	apiKey     string // passed as access_token query param
}

// NewClient creates a new Datto EDR API client.
func NewClient(cfg Config, logger *slog.Logger) *Client {
	httpClient := apihelper.NewClient(apihelper.ClientConfig{
		BaseURL:    cfg.BaseURL,
		Timeout:    30 * time.Second,
		MaxRetries: 3,
		UserAgent:  "lgp-mcp-servers/datto-edr",
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	})

	// Rate limit: 600 req/min; circuit breaker with 5 failure threshold.
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
		apiKey:     cfg.APIKey,
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
		return nil, fmt.Errorf("dattoedr: failed to parse response: %w", err)
	}
	return result, nil
}

// GetList performs a GET request and returns a bare JSON array (LoopBack-style).
func (c *Client) GetList(ctx context.Context, path string, params map[string]string) ([]any, error) {
	raw, err := c.doGet(ctx, path, params)
	if err != nil {
		return nil, err
	}

	// LoopBack APIs return bare arrays.
	var arr []any
	if json.Unmarshal(raw, &arr) == nil {
		return arr, nil
	}

	// Fallback: try object with data field.
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil, fmt.Errorf("dattoedr: failed to parse list response: %w", err)
	}

	if data, ok := obj["data"]; ok {
		if items, ok := data.([]any); ok {
			return items, nil
		}
	}

	// Single object — return as single-element list.
	return []any{obj}, nil
}

// Post performs a POST request and returns the parsed JSON response as a map.
func (c *Client) Post(ctx context.Context, path string, body any) (map[string]any, error) {
	authPath := path + "?access_token=" + c.apiKey
	result, err := c.middleware.Execute(ctx, func() (any, error) {
		raw, err := c.http.Post(ctx, authPath, body)
		if err != nil {
			return nil, fmt.Errorf("dattoedr: POST %s: %w", path, err)
		}
		return raw, nil
	})
	if err != nil {
		return nil, err
	}

	raw := result.([]byte)
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("dattoedr: failed to parse POST response: %w", err)
	}
	return out, nil
}

// TestConnection verifies connectivity by calling GET /api/Agents with limit=1.
func (c *Client) TestConnection(ctx context.Context) error {
	params := map[string]string{
		"filter[limit]": "1",
	}
	_, err := c.doGet(ctx, "/api/Agents", params)
	if err != nil {
		return fmt.Errorf("dattoedr: connection test failed: %w", err)
	}
	return nil
}

// injectToken adds the access_token query parameter for authentication.
func (c *Client) injectToken(params map[string]string) map[string]string {
	if params == nil {
		params = make(map[string]string)
	}
	params["access_token"] = c.apiKey
	return params
}

// doGet performs the actual GET call through the middleware.
func (c *Client) doGet(ctx context.Context, path string, params map[string]string) ([]byte, error) {
	params = c.injectToken(params)
	result, err := c.middleware.Execute(ctx, func() (any, error) {
		raw, err := c.http.Get(ctx, path, params)
		if err != nil {
			return nil, fmt.Errorf("dattoedr: GET %s: %w", path, err)
		}
		return raw, nil
	})
	if err != nil {
		return nil, err
	}
	return result.([]byte), nil
}

// LoopBack filter helpers

// AddWhereFilter adds a LoopBack where-clause filter parameter.
func AddWhereFilter(params map[string]string, field, value string) {
	if value != "" {
		params[fmt.Sprintf("filter[where][%s]", field)] = value
	}
}

// AddLimitFilter adds a LoopBack limit filter parameter.
func AddLimitFilter(params map[string]string, limit int) {
	if limit > 0 {
		params["filter[limit]"] = fmt.Sprintf("%d", limit)
	}
}

// AddSkipFilter adds a LoopBack skip (offset) filter parameter.
func AddSkipFilter(params map[string]string, skip int) {
	if skip > 0 {
		params["filter[skip]"] = fmt.Sprintf("%d", skip)
	}
}

// AddOrderFilter adds a LoopBack order filter parameter.
func AddOrderFilter(params map[string]string, order string) {
	if order != "" {
		params["filter[order]"] = order
	}
}

// BuildJSONFilter builds a LoopBack filter as a single JSON string parameter.
// Some endpoints don't support bracket notation and require filter={"limit":N,...}.
func BuildJSONFilter(limit, skip int, order string) map[string]string {
	params := make(map[string]string)
	filter := make(map[string]any)
	if limit > 0 {
		filter["limit"] = limit
	}
	if skip > 0 {
		filter["skip"] = skip
	}
	if order != "" {
		filter["order"] = order
	}
	if len(filter) > 0 {
		b, _ := json.Marshal(filter) //nolint:errcheck
		params["filter"] = string(b)
	}
	return params
}
