package dattormm

import (
	"strconv"

	"github.com/Logiphys/lgp-mcp/pkg/mcputil"
	"github.com/mark3labs/mcp-go/mcp"
)

func paginationParams(req mcp.CallToolRequest) map[string]string {
	params := make(map[string]string)
	if v := req.GetInt("page", 0); v > 0 {
		params["page"] = strconv.Itoa(v)
	}
	if v := req.GetInt("max", 0); v > 0 {
		params["max"] = strconv.Itoa(v)
	}
	return params
}

func listResult(items []any, pageInfo *PageInfo) *mcp.CallToolResult {
	result := map[string]any{"items": items}
	if pageInfo != nil {
		result["pagination"] = map[string]any{
			"page":        pageInfo.Page,
			"total_pages": pageInfo.TotalPages,
			"count":       pageInfo.Count,
		}
	}
	return mcputil.JSONResult(result)
}
