package itglue

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Logiphys/lgp-mcp/pkg/mcputil"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerDocumentTools(srv *server.MCPServer, client *Client, logger *slog.Logger) {
	// ── Documents ──────────────────────────────────────────────────────────────

	srv.AddTool(
		mcp.NewTool("itglue_search_documents",
			mcp.WithDescription("Search IT Glue documents within an organization. organization_id is required."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(true),
			mcp.WithNumber("organization_id",
				mcp.Description("The organization ID to list documents for."),
				mcp.Required(),
			),
			mcp.WithString("name",
				mcp.Description("Filter by document name (exact match)."),
			),
			mcp.WithNumber("page_number",
				mcp.Description("Page number to retrieve (default: 1)."),
			),
			mcp.WithNumber("page_size",
				mcp.Description("Number of results per page (default: 50)."),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			orgID := req.GetInt("organization_id", 0)
			if orgID == 0 {
				return mcputil.ErrorResult(fmt.Errorf("organization_id is required")), nil
			}

			filters := make(map[string]string)
			if v := req.GetString("name", ""); v != "" {
				filters["name"] = v
			}
			page := req.GetInt("page_number", 1)
			pageSize := req.GetInt("page_size", 50)

			path := fmt.Sprintf("/organizations/%d/relationships/documents", orgID)
			items, meta, err := client.List(ctx, path, filters, page, pageSize)
			if err != nil {
				logger.Error("itglue_search_documents failed", "organization_id", orgID, "error", err)
				return mcputil.ErrorResult(err), nil
			}
			if len(items) == 0 {
				return mcputil.TextResult("No results found."), nil
			}
			result := map[string]any{
				"items": items,
				"pagination": map[string]any{
					"current_page": meta.CurrentPage,
					"total_pages":  meta.TotalPages,
					"total_count":  meta.TotalCount,
				},
			}
			return mcputil.JSONResult(result), nil
		},
	)

	srv.AddTool(
		mcp.NewTool("itglue_get_document",
			mcp.WithDescription("Get a single IT Glue document by ID within an organization. Both organization_id and id are required."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithNumber("organization_id",
				mcp.Description("The organization ID that owns the document."),
				mcp.Required(),
			),
			mcp.WithNumber("id",
				mcp.Description("The document ID."),
				mcp.Required(),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			orgID := req.GetInt("organization_id", 0)
			if orgID == 0 {
				return mcputil.ErrorResult(fmt.Errorf("organization_id is required")), nil
			}
			id := req.GetInt("id", 0)
			if id == 0 {
				return mcputil.ErrorResult(fmt.Errorf("id is required")), nil
			}

			path := fmt.Sprintf("/organizations/%d/relationships/documents/%d", orgID, id)
			item, err := client.Get(ctx, path)
			if err != nil {
				logger.Error("itglue_get_document failed", "organization_id", orgID, "id", id, "error", err)
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.JSONResult(item), nil
		},
	)

	srv.AddTool(
		mcp.NewTool("itglue_create_document",
			mcp.WithDescription("Create a new IT Glue document within an organization."),
			mcp.WithNumber("organization_id",
				mcp.Description("The organization ID to create the document in."),
				mcp.Required(),
			),
			mcp.WithString("name",
				mcp.Description("The document name."),
				mcp.Required(),
			),
			mcp.WithString("content",
				mcp.Description("The document content (HTML). Optional."),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			orgID := req.GetInt("organization_id", 0)
			if orgID == 0 {
				return mcputil.ErrorResult(fmt.Errorf("organization_id is required")), nil
			}
			name := req.GetString("name", "")
			if name == "" {
				return mcputil.ErrorResult(fmt.Errorf("name is required")), nil
			}

			attrs := map[string]any{
				"name": name,
			}
			if v := req.GetString("content", ""); v != "" {
				attrs["content"] = v
			}

			path := fmt.Sprintf("/organizations/%d/relationships/documents", orgID)
			item, err := client.Create(ctx, path, "documents", attrs)
			if err != nil {
				logger.Error("itglue_create_document failed", "organization_id", orgID, "error", err)
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.JSONResult(item), nil
		},
	)

	srv.AddTool(
		mcp.NewTool("itglue_update_document",
			mcp.WithDescription("Update an existing IT Glue document by ID."),
			mcp.WithNumber("id",
				mcp.Description("The document ID to update."),
				mcp.Required(),
			),
			mcp.WithString("name",
				mcp.Description("The new document name."),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			id := req.GetInt("id", 0)
			if id == 0 {
				return mcputil.ErrorResult(fmt.Errorf("id is required")), nil
			}

			attrs := map[string]any{}
			if v := req.GetString("name", ""); v != "" {
				attrs["name"] = v
			}
			if len(attrs) == 0 {
				return mcputil.ErrorResult(fmt.Errorf("at least one attribute must be provided for update")), nil
			}

			path := fmt.Sprintf("/documents/%d", id)
			item, err := client.Update(ctx, path, "documents", fmt.Sprintf("%d", id), attrs)
			if err != nil {
				logger.Error("itglue_update_document failed", "id", id, "error", err)
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.JSONResult(item), nil
		},
	)

	srv.AddTool(
		mcp.NewTool("itglue_delete_document",
			mcp.WithDescription("Delete an IT Glue document by ID. This action is irreversible."),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithNumber("id",
				mcp.Description("The document ID to delete."),
				mcp.Required(),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			id := req.GetInt("id", 0)
			if id == 0 {
				return mcputil.ErrorResult(fmt.Errorf("id is required")), nil
			}

			path := fmt.Sprintf("/documents/%d", id)
			if err := client.Delete(ctx, path); err != nil {
				logger.Error("itglue_delete_document failed", "id", id, "error", err)
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.TextResult(fmt.Sprintf("Document %d deleted successfully.", id)), nil
		},
	)

	// ── Document Sections ──────────────────────────────────────────────────────

	srv.AddTool(
		mcp.NewTool("itglue_list_document_sections",
			mcp.WithDescription("List all sections within an IT Glue document."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithNumber("document_id",
				mcp.Description("The document ID to list sections for."),
				mcp.Required(),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			docID := req.GetInt("document_id", 0)
			if docID == 0 {
				return mcputil.ErrorResult(fmt.Errorf("document_id is required")), nil
			}

			path := fmt.Sprintf("/documents/%d/relationships/document_sections", docID)
			items, meta, err := client.List(ctx, path, nil, 1, 100)
			if err != nil {
				logger.Error("itglue_list_document_sections failed", "document_id", docID, "error", err)
				return mcputil.ErrorResult(err), nil
			}
			if len(items) == 0 {
				return mcputil.TextResult("No sections found."), nil
			}
			result := map[string]any{
				"items": items,
				"pagination": map[string]any{
					"current_page": meta.CurrentPage,
					"total_pages":  meta.TotalPages,
					"total_count":  meta.TotalCount,
				},
			}
			return mcputil.JSONResult(result), nil
		},
	)

	srv.AddTool(
		mcp.NewTool("itglue_get_document_section",
			mcp.WithDescription("Get a single IT Glue document section by section ID. HTML content is stripped to plain text."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithNumber("section_id",
				mcp.Description("The document section ID."),
				mcp.Required(),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			sectionID := req.GetInt("section_id", 0)
			if sectionID == 0 {
				return mcputil.ErrorResult(fmt.Errorf("section_id is required")), nil
			}

			path := fmt.Sprintf("/document_sections/%d", sectionID)
			item, err := client.Get(ctx, path)
			if err != nil {
				logger.Error("itglue_get_document_section failed", "section_id", sectionID, "error", err)
				return mcputil.ErrorResult(err), nil
			}

			// Strip HTML from content if present.
			if attrs, ok := item["attributes"].(map[string]any); ok {
				if content, ok := attrs["content"].(string); ok {
					attrs["content"] = mcputil.StripHTML(content)
				}
			}

			return mcputil.JSONResult(item), nil
		},
	)

	srv.AddTool(
		mcp.NewTool("itglue_create_document_section",
			mcp.WithDescription("Create a new section within an IT Glue document. section_type must be 'heading' or 'text'."),
			mcp.WithNumber("document_id",
				mcp.Description("The document ID to add the section to."),
				mcp.Required(),
			),
			mcp.WithString("section_type",
				mcp.Description("The section type: 'heading' or 'text'."),
				mcp.Required(),
			),
			mcp.WithString("content",
				mcp.Description("The section content."),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			docID := req.GetInt("document_id", 0)
			if docID == 0 {
				return mcputil.ErrorResult(fmt.Errorf("document_id is required")), nil
			}

			sectionType := req.GetString("section_type", "")
			var mappedType string
			switch sectionType {
			case "heading":
				mappedType = "Document::Heading"
			case "text":
				mappedType = "Document::Text"
			default:
				return mcputil.ErrorResult(fmt.Errorf("section_type must be 'heading' or 'text', got %q", sectionType)), nil
			}

			attrs := map[string]any{
				"section-type": mappedType,
			}
			if v := req.GetString("content", ""); v != "" {
				attrs["content"] = v
			}

			path := fmt.Sprintf("/documents/%d/relationships/document_sections", docID)
			item, err := client.Create(ctx, path, "document-sections", attrs)
			if err != nil {
				logger.Error("itglue_create_document_section failed", "document_id", docID, "error", err)
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.JSONResult(item), nil
		},
	)

	srv.AddTool(
		mcp.NewTool("itglue_update_document_section",
			mcp.WithDescription("Update the content of an IT Glue document section."),
			mcp.WithNumber("section_id",
				mcp.Description("The document section ID to update."),
				mcp.Required(),
			),
			mcp.WithString("content",
				mcp.Description("The new section content."),
				mcp.Required(),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			sectionID := req.GetInt("section_id", 0)
			if sectionID == 0 {
				return mcputil.ErrorResult(fmt.Errorf("section_id is required")), nil
			}

			attrs := map[string]any{
				"content": req.GetString("content", ""),
			}

			path := fmt.Sprintf("/document_sections/%d", sectionID)
			item, err := client.Update(ctx, path, "document-sections", fmt.Sprintf("%d", sectionID), attrs)
			if err != nil {
				logger.Error("itglue_update_document_section failed", "section_id", sectionID, "error", err)
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.JSONResult(item), nil
		},
	)

	srv.AddTool(
		mcp.NewTool("itglue_delete_document_section",
			mcp.WithDescription("Delete an IT Glue document section by section ID. This action is irreversible."),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithNumber("section_id",
				mcp.Description("The document section ID to delete."),
				mcp.Required(),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			sectionID := req.GetInt("section_id", 0)
			if sectionID == 0 {
				return mcputil.ErrorResult(fmt.Errorf("section_id is required")), nil
			}

			path := fmt.Sprintf("/document_sections/%d", sectionID)
			if err := client.Delete(ctx, path); err != nil {
				logger.Error("itglue_delete_document_section failed", "section_id", sectionID, "error", err)
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.TextResult(fmt.Sprintf("Document section %d deleted successfully.", sectionID)), nil
		},
	)

	srv.AddTool(
		mcp.NewTool("itglue_publish_document",
			mcp.WithDescription("Publish an IT Glue document, making it visible to all users with access."),
			mcp.WithNumber("document_id",
				mcp.Description("The document ID to publish."),
				mcp.Required(),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			docID := req.GetInt("document_id", 0)
			if docID == 0 {
				return mcputil.ErrorResult(fmt.Errorf("document_id is required")), nil
			}

			attrs := map[string]any{
				"published": true,
			}
			path := fmt.Sprintf("/documents/%d", docID)
			item, err := client.Update(ctx, path, "documents", fmt.Sprintf("%d", docID), attrs)
			if err != nil {
				logger.Error("itglue_publish_document failed", "document_id", docID, "error", err)
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.JSONResult(item), nil
		},
	)
}
