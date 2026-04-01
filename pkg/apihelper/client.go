package apihelper

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand/v2"
	"net/http"
	"time"
)

type ClientConfig struct {
	BaseURL    string
	Timeout    time.Duration
	MaxRetries int
	UserAgent  string
	Headers    map[string]string
}

type Client struct {
	http    *http.Client
	baseURL string
	retries int
	agent   string
	headers map[string]string
}

func NewClient(cfg ClientConfig) *Client {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	retries := cfg.MaxRetries
	if retries == 0 {
		retries = 3
	}
	agent := cfg.UserAgent
	if agent == "" {
		agent = "lgp-mcp/dev"
	}
	return &Client{
		http:    &http.Client{Timeout: timeout},
		baseURL: cfg.BaseURL,
		retries: retries,
		agent:   agent,
		headers: cfg.Headers,
	}
}

func (c *Client) Get(ctx context.Context, path string, params map[string]string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	if params != nil {
		q := req.URL.Query()
		for k, v := range params {
			q.Set(k, v)
		}
		req.URL.RawQuery = q.Encode()
	}
	return c.do(req)
}

func (c *Client) Post(ctx context.Context, path string, body any) ([]byte, error) {
	return c.doWithBody(ctx, http.MethodPost, path, body)
}

func (c *Client) Patch(ctx context.Context, path string, body any) ([]byte, error) {
	return c.doWithBody(ctx, http.MethodPatch, path, body)
}

func (c *Client) Delete(ctx context.Context, path string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	_, err = c.do(req)
	return err
}

func (c *Client) doWithBody(ctx context.Context, method, path string, body any) ([]byte, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshalling body: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	return c.do(req)
}

func (c *Client) do(req *http.Request) ([]byte, error) {
	req.Header.Set("User-Agent", c.agent)
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}

	// Save body for retries
	var bodyBytes []byte
	if req.Body != nil {
		bodyBytes, _ = io.ReadAll(req.Body)
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}

	var lastErr error
	for attempt := 0; attempt <= c.retries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(math.Pow(2, float64(attempt-1))) * 100 * time.Millisecond
			jitter := time.Duration(rand.IntN(int(backoff/2 + 1)))
			select {
			case <-req.Context().Done():
				return nil, req.Context().Err()
			case <-time.After(backoff + jitter):
			}
			// Reset body for retry
			if bodyBytes != nil {
				req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			}
		}

		resp, err := c.http.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			if req.Context().Err() != nil {
				return nil, lastErr
			}
			continue
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return body, nil
		}

		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
			continue
		}

		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}
