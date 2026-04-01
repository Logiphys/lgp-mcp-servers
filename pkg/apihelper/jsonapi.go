package apihelper

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"unicode"
)

type JSONAPIResource struct {
	ID         string         `json:"id"`
	Type       string         `json:"type"`
	Attributes map[string]any `json:"attributes"`
}

type PaginationMeta struct {
	CurrentPage int `json:"current-page"`
	NextPage    int `json:"next-page"`
	PrevPage    int `json:"prev-page"`
	TotalPages  int `json:"total-pages"`
	TotalCount  int `json:"total-count"`
}

type JSONAPIResponse struct {
	Data []JSONAPIResource
	Meta PaginationMeta
}

type rawJSONAPIResponse struct {
	Data json.RawMessage `json:"data"`
	Meta PaginationMeta  `json:"meta"`
}

func ParseJSONAPIResponse(body []byte) (*JSONAPIResponse, error) {
	var raw rawJSONAPIResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("parsing JSON:API response: %w", err)
	}
	resp := &JSONAPIResponse{Meta: raw.Meta}
	var resources []JSONAPIResource
	if err := json.Unmarshal(raw.Data, &resources); err == nil {
		resp.Data = resources
		return resp, nil
	}
	var single JSONAPIResource
	if err := json.Unmarshal(raw.Data, &single); err == nil {
		resp.Data = []JSONAPIResource{single}
		return resp, nil
	}
	resp.Data = []JSONAPIResource{}
	return resp, nil
}

func FlattenResource(r JSONAPIResource) map[string]any {
	flat := make(map[string]any, len(r.Attributes)+2)
	flat["id"] = r.ID
	flat["type"] = r.Type
	for k, v := range r.Attributes {
		flat[k] = v
	}
	return flat
}

func BuildFilterParams(filters map[string]string) url.Values {
	params := url.Values{}
	for k, v := range filters {
		params.Set(fmt.Sprintf("filter[%s]", k), v)
	}
	return params
}

func kebabToCamel(s string) string {
	parts := strings.Split(s, "-")
	for i := 1; i < len(parts); i++ {
		if len(parts[i]) > 0 {
			runes := []rune(parts[i])
			runes[0] = unicode.ToUpper(runes[0])
			parts[i] = string(runes)
		}
	}
	return strings.Join(parts, "")
}
