package apihelper

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

type OAuth2Config struct {
	TokenURL     string
	ClientID     string
	ClientSecret string
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

type TokenManager struct {
	cfg    OAuth2Config
	mu     sync.Mutex
	token  string
	expiry time.Time
	group  singleflight.Group
}

const tokenRefreshBuffer = 5 * time.Minute

func NewTokenManager(cfg OAuth2Config) *TokenManager {
	return &TokenManager{cfg: cfg}
}

func (t *TokenManager) Token(ctx context.Context) (string, error) {
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

func (t *TokenManager) fetchToken(ctx context.Context) (string, error) {
	data := url.Values{"grant_type": {"client_credentials"}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.cfg.TokenURL,
		strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("creating token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(t.cfg.ClientID, t.cfg.ClientSecret)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	body, _ := io.ReadAll(resp.Body) //nolint:errcheck
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token request returned HTTP %d: %s", resp.StatusCode, string(body))
	}

	var tok tokenResponse
	if err := json.Unmarshal(body, &tok); err != nil {
		return "", fmt.Errorf("parsing token response: %w", err)
	}

	t.mu.Lock()
	t.token = tok.AccessToken
	t.expiry = time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second)
	t.mu.Unlock()

	return tok.AccessToken, nil
}
