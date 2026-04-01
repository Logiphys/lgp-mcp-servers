package autotask

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/Logiphys/lgp-mcp/pkg/apihelper"
	"github.com/Logiphys/lgp-mcp/pkg/resilience"
)

// Config holds Autotask API credentials and settings.
type Config struct {
	Username        string
	Secret          string
	IntegrationCode string
	BaseURL         string
}

// Filter represents an Autotask query filter.
type Filter struct {
	Op    string   `json:"op"`
	Field string   `json:"field,omitempty"`
	Value any      `json:"value,omitempty"`
	Items []Filter `json:"items,omitempty"`
}

// QueryOpts configures a query request.
type QueryOpts struct {
	Page     int
	PageSize int
	MaxSize  int // entity-specific maximum page size
}

// Client is the Autotask REST API client.
type Client struct {
	http       *apihelper.Client
	baseURL    string
	middleware *resilience.Middleware
	logger     *slog.Logger
	companies  *apihelper.MappingCache[int, string]
	resources  *apihelper.MappingCache[int, string]
}

// NewClient creates a new Autotask API client.
func NewClient(cfg Config, logger *slog.Logger) *Client {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://webservices24.autotask.net/ATServicesRest"
	}

	httpClient := apihelper.NewClient(apihelper.ClientConfig{
		BaseURL:    baseURL,
		Timeout:    30 * time.Second,
		MaxRetries: 3,
		UserAgent:  "lgp-mcp/autotask",
		Headers: map[string]string{
			"UserName":           cfg.Username,
			"Secret":             cfg.Secret,
			"ApiIntegrationcode": cfg.IntegrationCode,
			"Content-Type":       "application/json",
		},
	})

	mw := resilience.NewMiddleware(resilience.Config{
		RateLimit:        5000,
		FailureThreshold: 5,
		Cooldown:         30 * time.Second,
		SuccessThreshold: 3,
		Compact:          true,
	})

	return &Client{
		http:       httpClient,
		baseURL:    baseURL,
		middleware: mw,
		logger:     logger,
		companies:  apihelper.NewMappingCache[int, string](30 * time.Minute),
		resources:  apihelper.NewMappingCache[int, string](30 * time.Minute),
	}
}

// Get retrieves a single entity by ID.
// GET /{entity}/{id} -> {"item": {...}}
func (c *Client) Get(ctx context.Context, entity string, id int) (map[string]any, error) {
	result, err := c.middleware.Execute(ctx, func() (any, error) {
		path := fmt.Sprintf("/%s/%d", entity, id)
		body, err := c.http.Get(ctx, path, nil)
		if err != nil {
			return nil, err
		}
		var resp struct {
			Item map[string]any `json:"item"`
		}
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, fmt.Errorf("parsing response: %w", err)
		}
		return resp.Item, nil
	})
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}
	return result.(map[string]any), nil
}

// Query searches entities with filters.
// POST /{entity}/query with {"filter": [...], "MaxRecords": N}
func (c *Client) Query(ctx context.Context, entity string, filters []Filter, opts QueryOpts) ([]map[string]any, error) {
	result, err := c.middleware.Execute(ctx, func() (any, error) {
		pageSize := opts.PageSize
		if pageSize == 0 {
			pageSize = 500
		}
		if opts.MaxSize > 0 && pageSize > opts.MaxSize {
			pageSize = opts.MaxSize
		}

		queryBody := map[string]any{
			"filter":     filters,
			"MaxRecords": pageSize,
		}

		body, err := c.http.Post(ctx, "/"+entity+"/query", queryBody)
		if err != nil {
			return nil, err
		}

		var resp struct {
			Items []map[string]any `json:"items"`
		}
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, fmt.Errorf("parsing response: %w", err)
		}
		return resp.Items, nil
	})
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}
	return result.([]map[string]any), nil
}

// Create creates a new entity.
// POST /{entity} with body -> extracts ID from response.
func (c *Client) Create(ctx context.Context, entity string, data map[string]any) (int, error) {
	result, err := c.middleware.Execute(ctx, func() (any, error) {
		body, err := c.http.Post(ctx, "/"+entity, data)
		if err != nil {
			return nil, err
		}

		var resp map[string]any
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, fmt.Errorf("parsing response: %w", err)
		}

		id, ok := extractID(resp)
		if !ok {
			return nil, fmt.Errorf("could not extract ID from response: %s", string(body))
		}
		return id, nil
	})
	if err != nil {
		return 0, err
	}
	return result.(int), nil
}

// CreateChild creates a child entity under a parent.
// POST /{parent}/{parentID}/{child} with body.
func (c *Client) CreateChild(ctx context.Context, parent string, parentID int, child string, data map[string]any) (int, error) {
	result, err := c.middleware.Execute(ctx, func() (any, error) {
		path := fmt.Sprintf("/%s/%d/%s", parent, parentID, child)
		body, err := c.http.Post(ctx, path, data)
		if err != nil {
			return nil, err
		}

		var resp map[string]any
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, fmt.Errorf("parsing response: %w", err)
		}

		id, ok := extractID(resp)
		if !ok {
			return nil, fmt.Errorf("could not extract ID from response: %s", string(body))
		}
		return id, nil
	})
	if err != nil {
		return 0, err
	}
	return result.(int), nil
}

// Update updates an entity via PATCH.
// PATCH /{entity}/{id} with body.
func (c *Client) Update(ctx context.Context, entity string, id int, data map[string]any) error {
	_, err := c.middleware.Execute(ctx, func() (any, error) {
		path := fmt.Sprintf("/%s/%d", entity, id)
		_, err := c.http.Patch(ctx, path, data)
		return nil, err
	})
	return err
}

// Delete deletes an entity.
// DELETE /{entity}/{id}.
func (c *Client) Delete(ctx context.Context, entity string, id int) error {
	_, err := c.middleware.Execute(ctx, func() (any, error) {
		path := fmt.Sprintf("/%s/%d", entity, id)
		return nil, c.http.Delete(ctx, path)
	})
	return err
}

// DeleteChild deletes a child entity.
// DELETE /{parent}/{parentID}/{child}/{childID}.
func (c *Client) DeleteChild(ctx context.Context, parent string, parentID int, child string, childID int) error {
	_, err := c.middleware.Execute(ctx, func() (any, error) {
		path := fmt.Sprintf("/%s/%d/%s/%d", parent, parentID, child, childID)
		return nil, c.http.Delete(ctx, path)
	})
	return err
}

// GetFieldInfo retrieves field definitions for an entity type.
// GET /{entity}/entityInformation/fields.
func (c *Client) GetFieldInfo(ctx context.Context, entity string) ([]map[string]any, error) {
	result, err := c.middleware.Execute(ctx, func() (any, error) {
		path := fmt.Sprintf("/%s/entityInformation/fields", entity)
		body, err := c.http.Get(ctx, path, nil)
		if err != nil {
			return nil, err
		}

		// Try {"fields": [...]} envelope first
		var envelope struct {
			Fields []map[string]any `json:"fields"`
		}
		if err := json.Unmarshal(body, &envelope); err == nil && envelope.Fields != nil {
			return envelope.Fields, nil
		}

		// Fall back to bare array
		var fields []map[string]any
		if err := json.Unmarshal(body, &fields); err != nil {
			return nil, fmt.Errorf("parsing field info response: %w", err)
		}
		return fields, nil
	})
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}
	return result.([]map[string]any), nil
}

// TestConnection tests API connectivity by getting company 0.
func (c *Client) TestConnection(ctx context.Context) error {
	_, err := c.middleware.Execute(ctx, func() (any, error) {
		body, err := c.http.Get(ctx, "/Companies/0", nil)
		if err != nil {
			return nil, fmt.Errorf("connection test failed: %w", err)
		}
		return body, nil
	})
	return err
}

// EnhanceWithNames enriches items with company and resource names from caches.
func (c *Client) EnhanceWithNames(ctx context.Context, items []map[string]any) []map[string]any {
	for _, item := range items {
		c.enhanceCompanyName(ctx, item)
		c.enhanceResourceName(ctx, item, "assignedResourceID", "_assignedResourceName")
		c.enhanceResourceName(ctx, item, "resourceID", "_resourceName")
		c.enhanceResourceName(ctx, item, "projectLeadResourceID", "_projectLeadResourceName")
	}
	return items
}

func (c *Client) enhanceCompanyName(ctx context.Context, item map[string]any) {
	companyID, ok := toInt(item["companyID"])
	if !ok || companyID == 0 {
		return
	}
	name, err := c.companies.Get(ctx, companyID, func(id int) (string, error) {
		entity, err := c.Get(ctx, "Companies", id)
		if err != nil {
			return "", err
		}
		if n, ok := entity["companyName"].(string); ok {
			return n, nil
		}
		return "", fmt.Errorf("companyName not found for ID %d", id)
	})
	if err != nil {
		c.logger.Debug("failed to resolve company name", "companyID", companyID, "error", err)
		return
	}
	item["_companyName"] = name
}

func (c *Client) enhanceResourceName(ctx context.Context, item map[string]any, field string, targetField string) {
	resourceID, ok := toInt(item[field])
	if !ok || resourceID == 0 {
		return
	}
	name, err := c.resources.Get(ctx, resourceID, func(id int) (string, error) {
		entity, err := c.Get(ctx, "Resources", id)
		if err != nil {
			return "", err
		}
		first, _ := entity["firstName"].(string)
		last, _ := entity["lastName"].(string)
		if first == "" && last == "" {
			return "", fmt.Errorf("resource name not found for ID %d", id)
		}
		return first + " " + last, nil
	})
	if err != nil {
		c.logger.Debug("failed to resolve resource name", "field", field, "resourceID", resourceID, "error", err)
		return
	}
	item[targetField] = name
}

// extractID attempts to extract a numeric ID from an Autotask create response.
// It tries: response["itemId"], response["item"]["id"], response["id"].
func extractID(resp map[string]any) (int, bool) {
	// Pattern 1: {"itemId": N}
	if v, ok := resp["itemId"]; ok {
		if id, ok := toInt(v); ok {
			return id, true
		}
	}
	// Pattern 2: {"item": {"id": N}}
	if item, ok := resp["item"].(map[string]any); ok {
		if v, ok := item["id"]; ok {
			if id, ok := toInt(v); ok {
				return id, true
			}
		}
	}
	// Pattern 3: {"id": N}
	if v, ok := resp["id"]; ok {
		if id, ok := toInt(v); ok {
			return id, true
		}
	}
	return 0, false
}

// toInt converts a JSON number (float64) or int to int.
func toInt(v any) (int, bool) {
	switch n := v.(type) {
	case float64:
		return int(n), true
	case int:
		return n, true
	case int64:
		return int(n), true
	case json.Number:
		i, err := n.Int64()
		if err != nil {
			return 0, false
		}
		return int(i), true
	}
	return 0, false
}
