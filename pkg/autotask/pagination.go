package autotask

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// QueryPage contains one upstream page and its authoritative continuation.
type QueryPage struct {
	Items       []map[string]any
	NextPageURL *string
}

func queryPayload(filters []Filter, opts QueryOpts) map[string]any {
	pageSize := opts.PageSize
	if pageSize == 0 {
		pageSize = 500
	}
	if opts.MaxSize > 0 && pageSize > opts.MaxSize {
		pageSize = opts.MaxSize
	}

	// Autotask API rejects null filters — ensure at least one
	if filters == nil {
		filters = []Filter{{Op: "gte", Field: "id", Value: 0}}
	}

	// Autotask query endpoint requires filters wrapped in an "and" group.
	filterPayload := []Filter{{Op: "and", Items: filters}}

	return map[string]any{
		"filter":     filterPayload,
		"MaxRecords": pageSize,
	}
}

// QueryPage fetches one page. Continuations require the original filters and size.
// Unlike Query, this method requires explicit upstream pagination metadata.
func (c *Client) QueryPage(ctx context.Context, entity string, filters []Filter, opts QueryOpts, nextPageURL string) (QueryPage, error) {
	path := "/" + entity + "/query"
	if nextPageURL != "" {
		var err error
		path, err = c.continuationPath(entity, nextPageURL)
		if err != nil {
			return QueryPage{}, err
		}
	}
	result, err := c.middleware.Execute(ctx, func() (any, error) {
		body, err := c.http.Post(ctx, path, queryPayload(filters, opts))
		if err != nil {
			return nil, err
		}
		var resp struct {
			Items       []map[string]any `json:"items"`
			PageDetails struct {
				NextPageURL json.RawMessage `json:"nextPageUrl"`
			} `json:"pageDetails"`
		}
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, fmt.Errorf("parsing page: %w", err)
		}
		if resp.Items == nil || len(resp.PageDetails.NextPageURL) == 0 {
			return nil, fmt.Errorf("missing query page items or continuation metadata")
		}
		page := QueryPage{Items: resp.Items}
		if err := json.Unmarshal(resp.PageDetails.NextPageURL, &page.NextPageURL); err != nil {
			return nil, fmt.Errorf("parsing continuation: %w", err)
		}
		if page.NextPageURL != nil {
			if _, err := c.continuationPath(entity, *page.NextPageURL); err != nil {
				return nil, err
			}
			if *page.NextPageURL == nextPageURL {
				return nil, fmt.Errorf("repeated query continuation")
			}
		}
		return page, nil
	})
	if err != nil {
		return QueryPage{}, err
	}
	return result.(QueryPage), nil
}

// Only pass a validated relative path to the credentialed HTTP client.
func (c *Client) continuationPath(entity, continuation string) (string, error) {
	base, err := url.Parse(c.baseURL)
	if err != nil {
		return "", fmt.Errorf("invalid API base URL")
	}
	next, err := url.Parse(continuation)
	if err != nil || next.User != nil || next.Fragment != "" ||
		!strings.EqualFold(next.Scheme, base.Scheme) || !strings.EqualFold(next.Host, base.Host) ||
		(next.Scheme != "https" && next.Scheme != "http") ||
		!strings.EqualFold(next.EscapedPath(), strings.TrimRight(base.EscapedPath(), "/")+"/"+entity+"/query/next") || next.RawQuery == "" {
		return "", fmt.Errorf("invalid query continuation URL")
	}
	// Reconstruct only the trusted endpoint; keep the opaque query bytes intact.
	prefix := "/"
	if strings.HasSuffix(c.baseURL, "/") {
		prefix = ""
	}
	return prefix + entity + "/query/next?" + next.RawQuery, nil
}

// numberedPage replays upstream continuations for a numeric page or resumes a supplied cursor.
func (c *Client) numberedPage(ctx context.Context, entity string, filters []Filter, opts QueryOpts, page int, next string) (QueryPage, error) {
	if next != "" {
		page = 1
	}
	seen := make(map[string]bool)
	for {
		if seen[next] {
			return QueryPage{}, fmt.Errorf("repeated query continuation")
		}
		seen[next] = true
		result, err := c.QueryPage(ctx, entity, filters, opts, next)
		if err != nil || page == 1 {
			return result, err
		}
		if result.NextPageURL == nil {
			return QueryPage{Items: []map[string]any{}}, nil
		}
		next = *result.NextPageURL
		page--
	}
}
