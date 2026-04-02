package dattormm

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Logiphys/lgp-mcp-servers/pkg/mcputil"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerJobTools(srv *server.MCPServer, c *Client, _ *slog.Logger, tier int) {
	if tier < 2 {
		return
	}
	srv.AddTool(
		mcp.NewTool("datto_get_job",
			mcp.WithDescription("Retrieve a single Datto RMM job by its UID."),
			mcp.WithString("jobUid",
				mcp.Description("The UID of the job to retrieve."),
				mcp.Required(),
			),
			mcp.WithReadOnlyHintAnnotation(true),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			jobUid := req.GetString("jobUid", "")
			if jobUid == "" {
				return mcputil.ErrorResult(fmt.Errorf("jobUid is required")), nil
			}
			result, err := c.Get(ctx, fmt.Sprintf("/job/%s", jobUid), nil)
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.JSONResult(result), nil
		},
	)

	srv.AddTool(
		mcp.NewTool("datto_get_job_components",
			mcp.WithDescription("List the components of a Datto RMM job, with pagination."),
			mcp.WithString("jobUid",
				mcp.Description("The UID of the job."),
				mcp.Required(),
			),
			mcp.WithReadOnlyHintAnnotation(true),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			jobUid := req.GetString("jobUid", "")
			if jobUid == "" {
				return mcputil.ErrorResult(fmt.Errorf("jobUid is required")), nil
			}
			params := paginationParams(req)
			items, pageInfo, err := c.GetList(ctx, fmt.Sprintf("/job/%s/components", jobUid), params)
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return listResult(items, pageInfo), nil
		},
	)

	srv.AddTool(
		mcp.NewTool("datto_get_job_results",
			mcp.WithDescription("Retrieve the results of a Datto RMM job for a specific device."),
			mcp.WithString("jobUid",
				mcp.Description("The UID of the job."),
				mcp.Required(),
			),
			mcp.WithString("deviceUid",
				mcp.Description("The UID of the device."),
				mcp.Required(),
			),
			mcp.WithReadOnlyHintAnnotation(true),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			jobUid := req.GetString("jobUid", "")
			deviceUid := req.GetString("deviceUid", "")
			if jobUid == "" {
				return mcputil.ErrorResult(fmt.Errorf("jobUid is required")), nil
			}
			if deviceUid == "" {
				return mcputil.ErrorResult(fmt.Errorf("deviceUid is required")), nil
			}
			result, err := c.Get(ctx, fmt.Sprintf("/job/%s/device/%s", jobUid, deviceUid), nil)
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.JSONResult(result), nil
		},
	)

	srv.AddTool(
		mcp.NewTool("datto_get_job_stdout",
			mcp.WithDescription("Retrieve the standard output of a Datto RMM job for a specific device."),
			mcp.WithString("jobUid",
				mcp.Description("The UID of the job."),
				mcp.Required(),
			),
			mcp.WithString("deviceUid",
				mcp.Description("The UID of the device."),
				mcp.Required(),
			),
			mcp.WithReadOnlyHintAnnotation(true),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			jobUid := req.GetString("jobUid", "")
			deviceUid := req.GetString("deviceUid", "")
			if jobUid == "" {
				return mcputil.ErrorResult(fmt.Errorf("jobUid is required")), nil
			}
			if deviceUid == "" {
				return mcputil.ErrorResult(fmt.Errorf("deviceUid is required")), nil
			}
			raw, err := c.GetRaw(ctx, fmt.Sprintf("/job/%s/device/%s/stdout", jobUid, deviceUid), nil)
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.TextResult(string(raw)), nil
		},
	)

	srv.AddTool(
		mcp.NewTool("datto_get_job_stderr",
			mcp.WithDescription("Retrieve the standard error output of a Datto RMM job for a specific device."),
			mcp.WithString("jobUid",
				mcp.Description("The UID of the job."),
				mcp.Required(),
			),
			mcp.WithString("deviceUid",
				mcp.Description("The UID of the device."),
				mcp.Required(),
			),
			mcp.WithReadOnlyHintAnnotation(true),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			jobUid := req.GetString("jobUid", "")
			deviceUid := req.GetString("deviceUid", "")
			if jobUid == "" {
				return mcputil.ErrorResult(fmt.Errorf("jobUid is required")), nil
			}
			if deviceUid == "" {
				return mcputil.ErrorResult(fmt.Errorf("deviceUid is required")), nil
			}
			raw, err := c.GetRaw(ctx, fmt.Sprintf("/job/%s/device/%s/stderr", jobUid, deviceUid), nil)
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.TextResult(string(raw)), nil
		},
	)
}
