package itglue

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/Logiphys/lgp-mcp-servers/pkg/apihelper"
	"github.com/Logiphys/lgp-mcp-servers/pkg/resilience"
)

// regionBaseURLs maps IT Glue region codes to their API base URLs.
var regionBaseURLs = map[string]string{
	"us": "https://api.itglue.com",
	"eu": "https://api.eu.itglue.com",
	"au": "https://api.au.itglue.com",
}

// Config holds IT Glue API credentials and region settings.
type Config struct {
	APIKey  string
	Region  string // "us", "eu", "au"
	BaseURL string // optional override
}

// Client is the IT Glue REST API client.
type Client struct {
	http       *apihelper.Client
	middleware *resilience.Middleware
	logger     *slog.Logger
}

// NewClient creates a new IT Glue API client.
func NewClient(cfg Config, logger *slog.Logger) *Client {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		if url, ok := regionBaseURLs[cfg.Region]; ok {
			baseURL = url
		} else {
			baseURL = regionBaseURLs["us"]
		}
	}

	httpClient := apihelper.NewClient(apihelper.ClientConfig{
		BaseURL:    baseURL,
		Timeout:    30 * time.Second,
		MaxRetries: 3,
		UserAgent:  "lgp-mcp-servers/itglue",
		Headers: map[string]string{
			"x-api-key":    cfg.APIKey,
			"Content-Type": "application/vnd.api+json",
			"Accept":       "application/vnd.api+json",
		},
	})

	mw := resilience.NewMiddleware(resilience.Config{
		RateLimit:        10000,
		FailureThreshold: 5,
		Cooldown:         30 * time.Second,
		SuccessThreshold: 3,
	})

	return &Client{
		http:       httpClient,
		middleware: mw,
		logger:     logger,
	}
}

// List retrieves a paginated, filtered list of resources at the given path.
// Returns flattened resource maps and pagination metadata.
func (c *Client) List(ctx context.Context, path string, filters map[string]string, page, pageSize int) ([]map[string]any, *apihelper.PaginationMeta, error) {
	result, err := c.middleware.Execute(ctx, func() (any, error) {
		params := make(map[string]string)

		// Build filter params
		if len(filters) > 0 {
			for k, v := range apihelper.BuildFilterParams(filters) {
				if len(v) > 0 {
					params[k] = v[0]
				}
			}
		}

		// Add pagination params
		if pageSize > 0 {
			params["page[size]"] = strconv.Itoa(pageSize)
		}
		if page > 0 {
			params["page[number]"] = strconv.Itoa(page)
		}

		c.logger.DebugContext(ctx, "itglue list", slog.String("path", path), slog.Int("page", page), slog.Int("pageSize", pageSize))

		body, err := c.http.Get(ctx, path, params)
		if err != nil {
			return nil, fmt.Errorf("GET %s: %w", path, err)
		}

		parsed, err := apihelper.ParseJSONAPIResponse(body)
		if err != nil {
			return nil, fmt.Errorf("parsing response from %s: %w", path, err)
		}

		resources := make([]map[string]any, 0, len(parsed.Data))
		for _, r := range parsed.Data {
			resources = append(resources, apihelper.FlattenResource(r))
		}

		return &listResult{resources: resources, meta: parsed.Meta}, nil
	})
	if err != nil {
		return nil, nil, err
	}

	lr := result.(*listResult)
	return lr.resources, &lr.meta, nil
}

// listResult is an internal holder for List results passed through middleware.
type listResult struct {
	resources []map[string]any
	meta      apihelper.PaginationMeta
}

// Get retrieves a single resource by its full path (e.g. "/organizations/123").
func (c *Client) Get(ctx context.Context, path string) (map[string]any, error) {
	result, err := c.middleware.Execute(ctx, func() (any, error) {
		c.logger.DebugContext(ctx, "itglue get", slog.String("path", path))

		body, err := c.http.Get(ctx, path, nil)
		if err != nil {
			return nil, fmt.Errorf("GET %s: %w", path, err)
		}

		parsed, err := apihelper.ParseJSONAPIResponse(body)
		if err != nil {
			return nil, fmt.Errorf("parsing response from %s: %w", path, err)
		}

		if len(parsed.Data) == 0 {
			return nil, fmt.Errorf("no resource returned from %s", path)
		}

		return apihelper.FlattenResource(parsed.Data[0]), nil
	})
	if err != nil {
		return nil, err
	}

	return result.(map[string]any), nil
}

// Create posts a new JSON:API resource to the given path.
func (c *Client) Create(ctx context.Context, path string, resourceType string, attributes map[string]any) (map[string]any, error) {
	requestBody := map[string]any{
		"data": map[string]any{
			"type":       resourceType,
			"attributes": attributes,
		},
	}

	result, err := c.middleware.Execute(ctx, func() (any, error) {
		c.logger.DebugContext(ctx, "itglue create", slog.String("path", path), slog.String("type", resourceType))

		body, err := c.http.Post(ctx, path, requestBody)
		if err != nil {
			return nil, fmt.Errorf("POST %s: %w", path, err)
		}

		parsed, err := apihelper.ParseJSONAPIResponse(body)
		if err != nil {
			return nil, fmt.Errorf("parsing create response from %s: %w", path, err)
		}

		if len(parsed.Data) == 0 {
			return nil, fmt.Errorf("no resource returned from POST %s", path)
		}

		return apihelper.FlattenResource(parsed.Data[0]), nil
	})
	if err != nil {
		return nil, err
	}

	return result.(map[string]any), nil
}

// Update patches an existing JSON:API resource at the given path.
func (c *Client) Update(ctx context.Context, path string, resourceType string, id string, attributes map[string]any) (map[string]any, error) {
	requestBody := map[string]any{
		"data": map[string]any{
			"type":       resourceType,
			"id":         id,
			"attributes": attributes,
		},
	}

	result, err := c.middleware.Execute(ctx, func() (any, error) {
		c.logger.DebugContext(ctx, "itglue update", slog.String("path", path), slog.String("type", resourceType), slog.String("id", id))

		body, err := c.http.Patch(ctx, path, requestBody)
		if err != nil {
			return nil, fmt.Errorf("PATCH %s: %w", path, err)
		}

		parsed, err := apihelper.ParseJSONAPIResponse(body)
		if err != nil {
			return nil, fmt.Errorf("parsing update response from %s: %w", path, err)
		}

		if len(parsed.Data) == 0 {
			return nil, fmt.Errorf("no resource returned from PATCH %s", path)
		}

		return apihelper.FlattenResource(parsed.Data[0]), nil
	})
	if err != nil {
		return nil, err
	}

	return result.(map[string]any), nil
}

// Delete removes a resource at the given path.
func (c *Client) Delete(ctx context.Context, path string) error {
	_, err := c.middleware.Execute(ctx, func() (any, error) {
		c.logger.DebugContext(ctx, "itglue delete", slog.String("path", path))

		if err := c.http.Delete(ctx, path); err != nil {
			return nil, fmt.Errorf("DELETE %s: %w", path, err)
		}
		return nil, nil
	})
	return err
}

// TestConnection verifies API connectivity by fetching organization types.
func (c *Client) TestConnection(ctx context.Context) error {
	_, err := c.middleware.Execute(ctx, func() (any, error) {
		c.logger.DebugContext(ctx, "itglue test connection")

		body, err := c.http.Get(ctx, "/organization_types", map[string]string{
			"page[size]": "1",
		})
		if err != nil {
			return nil, fmt.Errorf("GET /organization_types: %w", err)
		}

		if _, err := apihelper.ParseJSONAPIResponse(body); err != nil {
			return nil, fmt.Errorf("parsing test connection response: %w", err)
		}

		return nil, nil
	})
	if err != nil {
		return fmt.Errorf("itglue test connection: %w", err)
	}
	return nil
}
