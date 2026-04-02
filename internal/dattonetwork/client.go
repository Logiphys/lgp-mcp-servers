package dattonetwork

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

// Config holds Datto Networking (DNA) API credentials and settings.
type Config struct {
	PublicKey string
	SecretKey string
	BaseURL   string // optional override
}

// Client is the Datto Networking REST API client.
type Client struct {
	http       *apihelper.Client
	middleware *resilience.Middleware
	logger     *slog.Logger
}

// NewClient creates a new Datto Networking API client.
func NewClient(cfg Config, logger *slog.Logger) *Client {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://api.dna.datto.com/dna-api/v1"
	}

	// Datto APIs use Basic Auth with publicKey:secretKey.
	credentials := base64.StdEncoding.EncodeToString([]byte(cfg.PublicKey + ":" + cfg.SecretKey))

	httpClient := apihelper.NewClient(apihelper.ClientConfig{
		BaseURL:    baseURL,
		Timeout:    30 * time.Second,
		MaxRetries: 3,
		UserAgent:  "lgp-mcp/datto-network",
		Headers: map[string]string{
			"Authorization": fmt.Sprintf("Basic %s", credentials),
			"Content-Type":  "application/json",
		},
	})

	// Rate limit: 1000 req/hr; circuit breaker with 5 failure threshold.
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
		return nil, fmt.Errorf("dattonetwork: failed to parse response: %w", err)
	}
	return result, nil
}

// GetList performs a GET request and returns a bare JSON array.
// The DNA API returns bare arrays for list endpoints.
func (c *Client) GetList(ctx context.Context, path string, params map[string]string) ([]any, error) {
	raw, err := c.doGet(ctx, path, params)
	if err != nil {
		return nil, err
	}

	// Try bare array first.
	var arr []any
	if json.Unmarshal(raw, &arr) == nil {
		return arr, nil
	}

	// Try object with "data" field as fallback.
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil, fmt.Errorf("dattonetwork: failed to parse list response: %w", err)
	}

	data, ok := obj["data"]
	if !ok {
		// Single object wrapped — return as single-element list.
		return []any{obj}, nil
	}

	items, ok := data.([]any)
	if !ok {
		return nil, fmt.Errorf("dattonetwork: unexpected data field type in list response")
	}

	return items, nil
}

// TestConnection verifies connectivity by calling GET /whoami.
func (c *Client) TestConnection(ctx context.Context) error {
	_, err := c.Get(ctx, "/whoami", nil)
	if err != nil {
		return fmt.Errorf("dattonetwork: connection test failed: %w", err)
	}
	return nil
}

// doGet performs the actual GET call through the middleware.
func (c *Client) doGet(ctx context.Context, path string, params map[string]string) ([]byte, error) {
	result, err := c.middleware.Execute(ctx, func() (any, error) {
		raw, err := c.http.Get(ctx, path, params)
		if err != nil {
			return nil, fmt.Errorf("dattonetwork: GET %s: %w", path, err)
		}
		return raw, nil
	})
	if err != nil {
		return nil, err
	}
	return result.([]byte), nil
}
