package logiphysci

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/Logiphys/lgp-mcp-servers/pkg/mcputil"
)

// berichtSchema mirrors the BerichtData dataclass in build_bericht.py.
const berichtSchema = `{
  "type": "object",
  "required": ["empfaenger_zeilen", "berichtstitel", "abschnitte"],
  "additionalProperties": false,
  "properties": {
    "empfaenger_zeilen": {
      "type": "array",
      "items": {"type": "string"},
      "description": "Empfänger-Adressblock (DIN 5008)"
    },
    "berichtstitel": {"type": "string", "description": "z.B. 'Diagnosebericht' oder 'Wartungsbericht'"},
    "abschnitte": {
      "type": "array",
      "minItems": 1,
      "items": {
        "type": "object",
        "required": ["ueberschrift"],
        "additionalProperties": false,
        "properties": {
          "ueberschrift": {"type": "string"},
          "level": {"type": "integer", "enum": [1, 2, 3], "description": "1=H1, 2=H2, 3=H3 (default 1)"},
          "absaetze": {"type": "array", "items": {"type": "string"}},
          "tabelle": {
            "type": "object",
            "required": ["spalten", "zeilen"],
            "additionalProperties": false,
            "properties": {
              "spalten": {"type": "array", "items": {"type": "string"}, "minItems": 1},
              "zeilen": {
                "type": "array",
                "items": {"type": "array", "items": {"type": "string"}}
              }
            }
          }
        }
      }
    },
    "auftragsnummer": {"type": "string"},
    "anlage": {"type": "string"},
    "techniker": {"type": "string"},
    "datum": {"type": "string", "description": "z.B. '06.05.2026' — optional, default heute"},
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

func registerBerichtTool(srv *server.MCPServer, runner BuildRunner, logger *slog.Logger) {
	srv.AddTool(
		mcp.NewToolWithRawSchema(
			"build_bericht",
			"Erzeugt einen Logiphys-Diagnose-/Wartungsbericht als DOCX (CI-konform).",
			json.RawMessage(berichtSchema),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			payload := req.GetArguments()
			if err := requireStringArray(payload, "empfaenger_zeilen"); err != nil {
				return mcputil.ErrorResult(err), nil
			}
			if err := requireString(payload, "berichtstitel"); err != nil {
				return mcputil.ErrorResult(err), nil
			}
			if err := requireNonEmptyArray(payload, "abschnitte"); err != nil {
				return mcputil.ErrorResult(err), nil
			}

			result, err := runner.RunBuilder(ctx, "build_bericht.py", payload, "docx")
			if err != nil {
				logger.Error("build_bericht failed", "err", err)
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.JSONResult(result), nil
		},
	)
}
