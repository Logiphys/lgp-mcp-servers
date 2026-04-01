package mcputil

import (
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

func TextResult(text string) *mcp.CallToolResult {
	return mcp.NewToolResultText(text)
}

func ErrorResult(err error) *mcp.CallToolResult {
	return mcp.NewToolResultError(err.Error())
}

func JSONResult(data any) *mcp.CallToolResult {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal response: %v", err))
	}
	return mcp.NewToolResultText(string(b))
}
