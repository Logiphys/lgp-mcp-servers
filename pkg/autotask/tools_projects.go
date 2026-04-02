package autotask

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Logiphys/lgp-mcp-servers/pkg/mcputil"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerProjectTools(srv *server.MCPServer, client *Client, _ *slog.Logger, tier int) {
	// === PROJECTS ===

	// autotask_search_projects
	addTool(srv,
		mcp.NewTool("autotask_search_projects",
			mcp.WithDescription("Search for projects in Autotask"),
			mcp.WithString("searchTerm", mcp.Description("Search by project name")),
			mcp.WithNumber("companyID", mcp.Description("Filter by company ID")),
			mcp.WithNumber("status", mcp.Description("Filter by status ID")),
			mcp.WithNumber("projectLeadResourceID", mcp.Description("Filter by project lead resource ID")),
			mcp.WithNumber("page", mcp.Description("Page number"), mcp.Min(1)),
			mcp.WithNumber("pageSize", mcp.Description("Results per page"), mcp.Min(1), mcp.Max(100)),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(true),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.GetArguments()
			var filters []Filter
			if term := req.GetString("searchTerm", ""); term != "" {
				filters = append(filters, Filter{Op: "contains", Field: "projectName", Value: term})
			}
			if v, ok := args["companyID"]; ok {
				filters = append(filters, Filter{Op: "eq", Field: "companyID", Value: v})
			}
			if v, ok := args["status"]; ok {
				filters = append(filters, Filter{Op: "eq", Field: "status", Value: v})
			}
			if v, ok := args["projectLeadResourceID"]; ok {
				filters = append(filters, Filter{Op: "eq", Field: "projectLeadResourceID", Value: v})
			}
			items, err := client.Query(ctx, "Projects", filters, QueryOpts{
				Page:     req.GetInt("page", 1),
				PageSize: req.GetInt("pageSize", 25),
				MaxSize:  100,
			})
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			if len(items) == 0 {
				return mcputil.TextResult(FormatNotFound("projects", map[string]any{"searchTerm": req.GetString("searchTerm", "")})), nil
			}
			items = client.EnhanceWithNames(ctx, items)
			return mcputil.TextResult(FormatSearchResult("autotask_search_projects", items, req.GetInt("page", 1), req.GetInt("pageSize", 25))), nil
		},
	)

	// Tier 3 — Write
	if tier >= 3 {
	// autotask_create_project
	addTool(srv,
		mcp.NewTool("autotask_create_project",
			mcp.WithDescription("Create a new project in Autotask"),
			mcp.WithNumber("companyID", mcp.Description("Company ID"), mcp.Required()),
			mcp.WithString("projectName", mcp.Description("Project name"), mcp.Required()),
			mcp.WithNumber("status", mcp.Description("Status ID"), mcp.Required()),
			mcp.WithNumber("projectType", mcp.Description("Project type ID"), mcp.Required()),
			mcp.WithString("description", mcp.Description("Project description")),
			mcp.WithString("startDate", mcp.Description("Start date (ISO 8601, will be converted to startDateTime)")),
			mcp.WithString("endDate", mcp.Description("End date (ISO 8601, will be converted to endDateTime)")),
			mcp.WithNumber("projectLeadResourceID", mcp.Description("Project lead resource ID")),
			mcp.WithNumber("estimatedHours", mcp.Description("Estimated hours")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			data := map[string]any{
				"companyID":   req.GetInt("companyID", 0),
				"projectName": req.GetString("projectName", ""),
				"status":      req.GetInt("status", 0),
				"projectType": req.GetInt("projectType", 0),
			}
			args := req.GetArguments()
			if v, ok := args["description"]; ok {
				data["description"] = v
			}
			if v, ok := args["startDate"]; ok {
				data["startDateTime"] = v
			}
			if v, ok := args["endDate"]; ok {
				data["endDateTime"] = v
			}
			if v, ok := args["projectLeadResourceID"]; ok {
				data["projectLeadResourceID"] = v
			}
			if v, ok := args["estimatedHours"]; ok {
				data["estimatedHours"] = v
			}
			id, err := client.Create(ctx, "Projects", data)
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.TextResult(FormatCreateResult("Project", id)), nil
		},
	)

	} // end tier >= 3

	// === TASKS ===

	// autotask_search_tasks
	addTool(srv,
		mcp.NewTool("autotask_search_tasks",
			mcp.WithDescription("Search for tasks in Autotask"),
			mcp.WithString("searchTerm", mcp.Description("Search by task title")),
			mcp.WithNumber("projectID", mcp.Description("Filter by project ID")),
			mcp.WithNumber("status", mcp.Description("Filter by status ID")),
			mcp.WithNumber("assignedResourceID", mcp.Description("Filter by assigned resource ID")),
			mcp.WithNumber("page", mcp.Description("Page number"), mcp.Min(1)),
			mcp.WithNumber("pageSize", mcp.Description("Results per page"), mcp.Min(1), mcp.Max(100)),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(true),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.GetArguments()
			var filters []Filter
			if term := req.GetString("searchTerm", ""); term != "" {
				filters = append(filters, Filter{Op: "contains", Field: "title", Value: term})
			}
			if v, ok := args["projectID"]; ok {
				filters = append(filters, Filter{Op: "eq", Field: "projectID", Value: v})
			}
			if v, ok := args["status"]; ok {
				filters = append(filters, Filter{Op: "eq", Field: "status", Value: v})
			}
			if v, ok := args["assignedResourceID"]; ok {
				filters = append(filters, Filter{Op: "eq", Field: "assignedResourceID", Value: v})
			}
			items, err := client.Query(ctx, "Tasks", filters, QueryOpts{
				Page:     req.GetInt("page", 1),
				PageSize: req.GetInt("pageSize", 25),
				MaxSize:  100,
			})
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			if len(items) == 0 {
				return mcputil.TextResult(FormatNotFound("tasks", map[string]any{"searchTerm": req.GetString("searchTerm", "")})), nil
			}
			items = client.EnhanceWithNames(ctx, items)
			return mcputil.TextResult(FormatSearchResult("autotask_search_tasks", items, req.GetInt("page", 1), req.GetInt("pageSize", 25))), nil
		},
	)

	// Tier 3 — Write
	if tier >= 3 {
	// autotask_create_task
	addTool(srv,
		mcp.NewTool("autotask_create_task",
			mcp.WithDescription("Create a new task in Autotask"),
			mcp.WithNumber("projectID", mcp.Description("Project ID"), mcp.Required()),
			mcp.WithString("title", mcp.Description("Task title"), mcp.Required()),
			mcp.WithNumber("status", mcp.Description("Status ID"), mcp.Required()),
			mcp.WithString("description", mcp.Description("Task description")),
			mcp.WithNumber("assignedResourceID", mcp.Description("Assigned resource ID")),
			mcp.WithNumber("estimatedHours", mcp.Description("Estimated hours")),
			mcp.WithNumber("taskType", mcp.Description("Task type (default: 1)")),
			mcp.WithString("startDateTime", mcp.Description("Start date/time (ISO 8601)")),
			mcp.WithString("endDateTime", mcp.Description("End date/time (ISO 8601)")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			data := map[string]any{
				"projectID": req.GetInt("projectID", 0),
				"title":     req.GetString("title", ""),
				"status":    req.GetInt("status", 0),
			}
			args := req.GetArguments()
			if v, ok := args["description"]; ok {
				data["description"] = v
			}
			if v, ok := args["assignedResourceID"]; ok {
				data["assignedResourceID"] = v
			}
			if v, ok := args["estimatedHours"]; ok {
				data["estimatedHours"] = v
			}
			if _, ok := args["taskType"]; ok {
				data["taskType"] = req.GetInt("taskType", 1)
			} else {
				data["taskType"] = 1
			}
			if v, ok := args["startDateTime"]; ok {
				data["startDateTime"] = v
			}
			if v, ok := args["endDateTime"]; ok {
				data["endDateTime"] = v
			}
			id, err := client.Create(ctx, "Tasks", data)
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.TextResult(FormatCreateResult("Task", id)), nil
		},
	)

	} // end tier >= 3

	// === PHASES ===

	// autotask_list_phases
	addTool(srv,
		mcp.NewTool("autotask_list_phases",
			mcp.WithDescription("List phases for a specific project"),
			mcp.WithNumber("projectID", mcp.Description("Project ID"), mcp.Required()),
			mcp.WithNumber("pageSize", mcp.Description("Results per page"), mcp.Min(1), mcp.Max(100)),
			mcp.WithReadOnlyHintAnnotation(true),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			projectID := req.GetInt("projectID", 0)
			if projectID == 0 {
				return mcputil.ErrorResult(fmt.Errorf("projectID is required")), nil
			}
			filters := []Filter{
				{Op: "eq", Field: "projectID", Value: projectID},
			}
			pageSize := req.GetInt("pageSize", 25)
			items, err := client.Query(ctx, "Phases", filters, QueryOpts{PageSize: pageSize, MaxSize: 100})
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			if len(items) == 0 {
				return mcputil.TextResult(FormatNotFound("phases", map[string]any{"projectID": projectID})), nil
			}
			return mcputil.TextResult(FormatSearchResult("autotask_list_phases", items, 1, pageSize)), nil
		},
	)

	// Tier 3 — Write
	if tier >= 3 {
	// autotask_create_phase
	addTool(srv,
		mcp.NewTool("autotask_create_phase",
			mcp.WithDescription("Create a new phase for a project"),
			mcp.WithNumber("projectID", mcp.Description("Project ID"), mcp.Required()),
			mcp.WithString("title", mcp.Description("Phase title"), mcp.Required()),
			mcp.WithString("description", mcp.Description("Phase description")),
			mcp.WithString("startDate", mcp.Description("Start date (ISO 8601)")),
			mcp.WithString("dueDate", mcp.Description("Due date (ISO 8601)")),
			mcp.WithNumber("estimatedHours", mcp.Description("Estimated hours")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			parentID := req.GetInt("projectID", 0)
			if parentID == 0 {
				return mcputil.ErrorResult(fmt.Errorf("projectID is required")), nil
			}
			data := map[string]any{
				"projectID": parentID,
				"title":     req.GetString("title", ""),
			}
			args := req.GetArguments()
			if v, ok := args["description"]; ok {
				data["description"] = v
			}
			if v, ok := args["startDate"]; ok {
				data["startDate"] = v
			}
			if v, ok := args["dueDate"]; ok {
				data["dueDate"] = v
			}
			if v, ok := args["estimatedHours"]; ok {
				data["estimatedHours"] = v
			}
			id, err := client.CreateChild(ctx, "Projects", parentID, "Phases", data)
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.TextResult(FormatCreateResult("Phase", id)), nil
		},
	)

	} // end tier >= 3

	// === PROJECT NOTES ===

	// autotask_get_project_note
	addTool(srv,
		mcp.NewTool("autotask_get_project_note",
			mcp.WithDescription("Get a specific project note by project ID and note ID"),
			mcp.WithNumber("projectId", mcp.Description("Project ID"), mcp.Required()),
			mcp.WithNumber("noteId", mcp.Description("Note ID"), mcp.Required()),
			mcp.WithReadOnlyHintAnnotation(true),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			projectID := req.GetInt("projectId", 0)
			noteID := req.GetInt("noteId", 0)
			if projectID == 0 || noteID == 0 {
				return mcputil.ErrorResult(fmt.Errorf("projectId and noteId are required")), nil
			}
			filters := []Filter{
				{Op: "eq", Field: "projectID", Value: projectID},
				{Op: "eq", Field: "id", Value: noteID},
			}
			items, err := client.Query(ctx, "ProjectNotes", filters, QueryOpts{PageSize: 1, MaxSize: 1})
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			if len(items) == 0 {
				return mcputil.TextResult(FormatNotFound("project note", map[string]any{"projectId": projectID, "noteId": noteID})), nil
			}
			return mcputil.TextResult(FormatGetResult(items[0])), nil
		},
	)

	// autotask_search_project_notes
	addTool(srv,
		mcp.NewTool("autotask_search_project_notes",
			mcp.WithDescription("Search notes for a specific project"),
			mcp.WithNumber("projectId", mcp.Description("Project ID"), mcp.Required()),
			mcp.WithNumber("pageSize", mcp.Description("Results per page"), mcp.Min(1), mcp.Max(100)),
			mcp.WithReadOnlyHintAnnotation(true),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			projectID := req.GetInt("projectId", 0)
			if projectID == 0 {
				return mcputil.ErrorResult(fmt.Errorf("projectId is required")), nil
			}
			filters := []Filter{
				{Op: "eq", Field: "projectID", Value: projectID},
			}
			pageSize := req.GetInt("pageSize", 25)
			items, err := client.Query(ctx, "ProjectNotes", filters, QueryOpts{PageSize: pageSize, MaxSize: 100})
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			if len(items) == 0 {
				return mcputil.TextResult(FormatNotFound("project notes", map[string]any{"projectId": projectID})), nil
			}
			return mcputil.TextResult(FormatSearchResult("autotask_search_project_notes", items, 1, pageSize)), nil
		},
	)

	// Tier 3 — Write
	if tier >= 3 {
	// autotask_create_project_note
	addTool(srv,
		mcp.NewTool("autotask_create_project_note",
			mcp.WithDescription("Create a new note for a project"),
			mcp.WithNumber("projectId", mcp.Description("Project ID"), mcp.Required()),
			mcp.WithString("description", mcp.Description("Note description"), mcp.Required()),
			mcp.WithString("title", mcp.Description("Note title")),
			mcp.WithNumber("noteType", mcp.Description("Note type")),
			mcp.WithNumber("publish", mcp.Description("1=Internal Only, 2=All Users")),
			mcp.WithBoolean("isAnnouncement", mcp.Description("Is announcement (default: false)")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			parentID := req.GetInt("projectId", 0)
			if parentID == 0 {
				return mcputil.ErrorResult(fmt.Errorf("projectId is required")), nil
			}
			data := map[string]any{
				"projectID":   parentID,
				"description": req.GetString("description", ""),
			}
			args := req.GetArguments()
			if v, ok := args["title"]; ok {
				data["title"] = v
			}
			if v, ok := args["noteType"]; ok {
				data["noteType"] = v
			}
			if _, ok := args["publish"]; ok {
				data["publish"] = req.GetInt("publish", 1)
			} else {
				data["publish"] = 1
			}
			if v, ok := args["isAnnouncement"]; ok {
				data["isAnnouncement"] = v
			} else {
				data["isAnnouncement"] = false
			}
			id, err := client.CreateChild(ctx, "Projects", parentID, "Notes", data)
			if err != nil {
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.TextResult(FormatCreateResult("ProjectNote", id)), nil
		},
	)

	} // end tier >= 3

	_ = server.ToolHandlerFunc(nil)
}
