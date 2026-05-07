package logiphysci

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/Logiphys/lgp-mcp-servers/pkg/mcputil"
)

// briefSchema mirrors the BriefData dataclass in build_brief.py.
const briefSchema = `{
  "type": "object",
  "required": ["empfaenger_zeilen", "betreff", "body_paragraphs"],
  "additionalProperties": false,
  "properties": {
    "empfaenger_zeilen": {
      "type": "array",
      "items": {"type": "string"},
      "description": "Empfänger-Adressblock; erste Zeile wird automatisch fett (Klavika Medium). DIN 5008 Form A."
    },
    "betreff": {"type": "string", "description": "Betreffzeile"},
    "body_paragraphs": {
      "type": "array",
      "items": {"type": "string"},
      "description": "Brieftext, ein Absatz pro Eintrag"
    },
    "datum": {
      "type": "string",
      "description": "z.B. '06. Mai 2026' oder '06.05.2026' — optional, default heute"
    },
    "ansprechpartner": {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "name": {"type": "string"},
        "telefon": {"type": "string"},
        "email": {"type": "string"}
      }
    }
  }
}`

func registerBriefTool(srv *server.MCPServer, runner BuildRunner, logger *slog.Logger) {
	srv.AddTool(
		mcp.NewToolWithRawSchema(
			"build_brief",
			"Erzeugt einen Logiphys-Geschäftsbrief als DOCX (CI-konform, DIN-5008-orientiert).",
			json.RawMessage(briefSchema),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			payload := req.GetArguments()
			if err := requireStringArray(payload, "empfaenger_zeilen"); err != nil {
				return mcputil.ErrorResult(err), nil
			}
			if err := requireString(payload, "betreff"); err != nil {
				return mcputil.ErrorResult(err), nil
			}
			if err := requireStringArray(payload, "body_paragraphs"); err != nil {
				return mcputil.ErrorResult(err), nil
			}

			result, err := runner.RunBuilder(ctx, "build_brief.py", payload, "docx")
			if err != nil {
				logger.Error("build_brief failed", "err", err)
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.JSONResult(result), nil
		},
	)
}
