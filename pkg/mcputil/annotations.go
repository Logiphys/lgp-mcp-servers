package mcputil

import "github.com/mark3labs/mcp-go/mcp"

type ToolAnnotations struct {
	Annotation *mcp.ToolAnnotation
}

func boolPtr(b bool) *bool { return &b }

func ReadOnly() ToolAnnotations {
	return ToolAnnotations{Annotation: &mcp.ToolAnnotation{
		ReadOnlyHint:    boolPtr(true),
		DestructiveHint: boolPtr(false),
	}}
}

func Destructive() ToolAnnotations {
	return ToolAnnotations{Annotation: &mcp.ToolAnnotation{
		ReadOnlyHint:    boolPtr(false),
		DestructiveHint: boolPtr(true),
	}}
}

func Idempotent() ToolAnnotations {
	return ToolAnnotations{Annotation: &mcp.ToolAnnotation{
		IdempotentHint: boolPtr(true),
	}}
}

func OpenWorld() ToolAnnotations {
	return ToolAnnotations{Annotation: &mcp.ToolAnnotation{
		OpenWorldHint: boolPtr(true),
	}}
}
