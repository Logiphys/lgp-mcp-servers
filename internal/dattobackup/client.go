package dattobackup

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"math/rand/v2"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"

	"github.com/Logiphys/lgp-mcp-servers/pkg/resilience"
)

const (
	defaultBaseURL     = "https://public-api.backup.net"
	tokenURL           = "https://login.backup.net/connect/token"
	tokenRefreshBuffer = 5 * time.Minute
	defaultTimeout     = 30 * time.Second
	defaultMaxRetries  = 3
	defaultRateLimit   = 600 // conservative
)

// Config holds Datto Backup (Unitrends) API credentials.
type Config struct {
	ClientID     string
	ClientSecret string
	BaseURL      string // optional override, default https://public-api.backup.net
}

// PageInfo holds pagination metadata from a list response.
type PageInfo struct {
	TotalRecords int
	TotalPages   int
	Page         int
	PageSize     int
}

// tokenManager handles OAuth2 client_credentials token lifecycle.
type tokenManager struct {
	clientID     string
	clientSecret string
	mu           sync.Mutex
	token        string
	expiry       time.Time
	group        singleflight.Group
}

func newTokenManager(clientID, clientSecret string) *tokenManager {
	return &tokenManager{
		clientID:     clientID,
		clientSecret: clientSecret,
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

// fetchToken performs the OAuth2 client_credentials token request.
func (t *tokenManager) fetchToken(ctx context.Context) (string, error) {
	data := url.Values{
		"grant_type": {"client_credentials"},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL,
		bytes.NewBufferString(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("creating token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// Basic Auth: base64(client_id:client_secret)
	creds := base64.StdEncoding.EncodeToString([]byte(t.clientID + ":" + t.clientSecret))
	req.Header.Set("Authorization", "Basic "+creds)

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

// Client is the Datto Backup (Unitrends) REST API client.
type Client struct {
	baseURL    string
	httpClient *http.Client
	tokenMgr   *tokenManager
	middleware *resilience.Middleware
	logger     *slog.Logger
	maxRetries int
}

// NewClient creates a new Datto Backup API client.
func NewClient(cfg Config, logger *slog.Logger) *Client {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	mw := resilience.NewMiddleware(resilience.Config{
		RateLimit:        defaultRateLimit,
		FailureThreshold: 5,
		Cooldown:         30 * time.Second,
		SuccessThreshold: 3,
	})

	return &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: defaultTimeout},
		tokenMgr:   newTokenManager(cfg.ClientID, cfg.ClientSecret),
		middleware: mw,
		logger:     logger,
		maxRetries: defaultMaxRetries,
	}
}

// doRequest executes an authenticated GET request with retry logic.
func (c *Client) doRequest(ctx context.Context, path string, params map[string]string) ([]byte, http.Header, error) {
	fullURL := c.baseURL + path
	if len(params) > 0 {
		q := url.Values{}
		for k, v := range params {
			q.Set(k, v)
		}
		fullURL += "?" + q.Encode()
	}

	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(math.Pow(2, float64(attempt-1))) * 100 * time.Millisecond
			jitter := time.Duration(rand.IntN(int(backoff/2 + 1)))
			select {
			case <-ctx.Done():
				return nil, nil, ctx.Err()
			case <-time.After(backoff + jitter):
			}
		}

		tok, err := c.tokenMgr.getToken(ctx)
		if err != nil {
			return nil, nil, fmt.Errorf("obtaining access token: %w", err)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
		if err != nil {
			return nil, nil, fmt.Errorf("creating request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+tok)
		req.Header.Set("User-Agent", "lgp-mcp-servers/dattobackup")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			if ctx.Err() != nil {
				return nil, nil, lastErr
			}
			continue
		}

		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return respBody, resp.Header, nil
		}

		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
			continue
		}

		return nil, nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	return nil, nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

// Get performs a GET request and returns the parsed JSON response.
func (c *Client) Get(ctx context.Context, path string, params map[string]string) (map[string]any, error) {
	result, err := c.middleware.Execute(ctx, func() (any, error) {
		body, _, err := c.doRequest(ctx, path, params)
		if err != nil {
			return nil, fmt.Errorf("dattobackup: GET %s: %w", path, err)
		}
		var resp map[string]any
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, fmt.Errorf("dattobackup: parsing response: %w", err)
		}
		return resp, nil
	})
	if err != nil {
		return nil, err
	}
	return result.(map[string]any), nil
}

// GetList performs a GET request and returns items array plus pagination from headers.
// Unitrends API returns pagination in headers: Paging-Total-Records, Paging-Total-Pages, etc.
func (c *Client) GetList(ctx context.Context, path string, params map[string]string) ([]any, *PageInfo, error) {
	result, err := c.middleware.Execute(ctx, func() (any, error) {
		body, headers, err := c.doRequest(ctx, path, params)
		if err != nil {
			return nil, fmt.Errorf("dattobackup: GET %s: %w", path, err)
		}

		var items []any
		if err := json.Unmarshal(body, &items); err != nil {
			return nil, fmt.Errorf("dattobackup: parsing list response: %w", err)
		}

		pi := &PageInfo{
			TotalRecords: headerInt(headers, "Paging-Total-Records"),
			TotalPages:   headerInt(headers, "Paging-Total-Pages"),
			Page:         headerInt(headers, "Paging-Page-Number"),
			PageSize:     headerInt(headers, "Paging-Page-Size"),
		}

		return &getListResult{items: items, pageInfo: pi}, nil
	})
	if err != nil {
		return nil, nil, err
	}
	lr := result.(*getListResult)
	return lr.items, lr.pageInfo, nil
}

// TestConnection verifies connectivity by fetching customers with page_size=1.
func (c *Client) TestConnection(ctx context.Context) error {
	_, _, err := c.doRequest(ctx, "/v1/customers", map[string]string{"page_size": "1"})
	if err != nil {
		return fmt.Errorf("dattobackup: connection test failed: %w", err)
	}
	return nil
}

type getListResult struct {
	items    []any
	pageInfo *PageInfo
}

func headerInt(h http.Header, key string) int {
	v := h.Get(key)
	if v == "" {
		return 0
	}
	n, _ := strconv.Atoi(v)
	return n
}
