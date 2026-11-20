package dattormm

import (
	"context"
	"fmt"
	"math"
	"net/url"
	"strconv"
	"strings"

	"github.com/Logiphys/lgp-mcp-servers/pkg/mcputil"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerPatchTool(srv *server.MCPServer, c *Client) {
	srv.AddTool(mcp.NewTool("list_patches",
		mcp.WithDescription("List patches for exactly one device or site. Site results aggregate devices by patch."),
		mcp.WithString("deviceUid", mcp.Description("Device UID, mutually exclusive with siteUid.")),
		mcp.WithString("siteUid", mcp.Description("Site UID, mutually exclusive with deviceUid.")),
		mcp.WithString("installStatus", mcp.Enum("INSTALLED", "APPROVED_PENDING", "NOT_APPROVED")),
		mcp.WithNumber("page", mcp.Description("Upstream page number. Omit for the first page."), mcp.Min(0)),
		mcp.WithNumber("max", mcp.Description("Requested page size."), mcp.Min(1)),
		mcp.WithReadOnlyHintAnnotation(true),
	), c.listPatches)
}

func (c *Client) listPatches(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	_, device := args["deviceUid"]
	_, site := args["siteUid"]
	if device == site {
		return mcputil.ErrorResult(fmt.Errorf("provide exactly one of deviceUid and siteUid")), nil
	}
	scope := "device"
	if site {
		scope = "site"
	}
	uid, err := req.RequireString(scope + "Uid")
	if err != nil || strings.TrimSpace(uid) == "" || uid == "." || uid == ".." || strings.ContainsAny(uid, "/\\%") {
		return mcputil.ErrorResult(fmt.Errorf("invalid %sUid", scope)), nil
	}
	params := map[string]string{}
	if _, ok := args["installStatus"]; ok {
		status, _ := args["installStatus"].(string)
		switch status {
		case "INSTALLED", "APPROVED_PENDING", "NOT_APPROVED":
			params["installStatus"] = status
		default:
			return mcputil.ErrorResult(fmt.Errorf("unknown installStatus")), nil
		}
	}
	for name, minimum := range map[string]int{"page": 0, "max": 1} {
		if _, ok := args[name]; ok {
			value, err := mcputil.IntArgument(args, name, 0, minimum, math.MaxInt32)
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			params[name] = strconv.Itoa(value)
		}
	}
	data, err := c.Get(ctx, "/"+scope+"/"+url.PathEscape(uid)+"/patches", params)
	if err != nil {
		return mcputil.ErrorResult(err), nil
	}
	items, ok := data["patches"].([]any)
	page, hasPage := data["pageDetails"].(map[string]any)
	if !ok || !hasPage {
		return mcputil.ErrorResult(fmt.Errorf("missing patch list or pagination metadata")), nil
	}
	// Project the documented fields so future provider changes go through review.
	fields := []string{"patchId", "category", "type", "title", "description", "releaseDate", "severity", "maxSize", "rebootRequired", "requireUserInput", "kbArticleId", "installStatus"}
	if site {
		fields = append(fields, "totalDevices")
	} else {
		fields = append(fields, "manualOverride")
	}
	patches := make([]map[string]any, 0, len(items))
	for _, item := range items {
		record, ok := item.(map[string]any)
		if !ok {
			return mcputil.ErrorResult(fmt.Errorf("invalid patch record")), nil
		}
		patches = append(patches, patchFields(record, fields...))
	}
	return mcputil.JSONResult(map[string]any{
		"items": patches, "pagination": patchFields(page, "count", "totalCount", "prevPageUrl", "nextPageUrl"),
	}), nil
}

func patchFields(record map[string]any, fields ...string) map[string]any {
	selected := make(map[string]any, len(fields))
	for _, field := range fields {
		if value, found := record[field]; found {
			selected[field] = value
		}
	}
	return selected
}
