package logiphysci

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/Logiphys/lgp-mcp-servers/pkg/mcputil"
)

// lieferscheinSchema mirrors the LieferscheinData dataclass in build_lieferschein.py.
const lieferscheinSchema = `{
  "type": "object",
  "required": ["lieferadresse_zeilen", "lieferschein_nummer", "positionen"],
  "additionalProperties": false,
  "properties": {
    "lieferadresse_zeilen": {
      "type": "array",
      "items": {"type": "string"},
      "description": "Lieferadresse (DIN 5008)"
    },
    "lieferschein_nummer": {"type": "string", "description": "z.B. 'LS-2026-001'"},
    "positionen": {
      "type": "array",
      "minItems": 1,
      "items": {
        "type": "object",
        "required": ["pos", "menge", "beschreibung"],
        "additionalProperties": false,
        "properties": {
          "pos": {"type": "integer"},
          "menge": {"type": "string", "description": "z.B. '2 Stk'"},
          "beschreibung": {"type": "string"},
          "seriennr": {"type": "string", "description": "Seriennummer, optional"}
        }
      }
    },
    "rechnungsadresse_zeilen": {
      "type": "array",
      "items": {"type": "string"},
      "description": "Abweichende Rechnungsadresse, optional"
    },
    "bestellnummer": {"type": "string"},
    "lieferdatum": {"type": "string", "description": "z.B. '06.05.2026'"},
    "hinweise": {"type": "array", "items": {"type": "string"}},
    "datum": {"type": "string", "description": "Belegdatum — optional, default heute"},
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

func registerLieferscheinTool(srv *server.MCPServer, runner BuildRunner, logger *slog.Logger) {
	srv.AddTool(
		mcp.NewToolWithRawSchema(
			"build_lieferschein",
			"Erzeugt einen Logiphys-Lieferschein als PDF (CI-konform, DIN-5008-orientiert).",
			json.RawMessage(lieferscheinSchema),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			payload := req.GetArguments()
			if err := requireStringArray(payload, "lieferadresse_zeilen"); err != nil {
				return mcputil.ErrorResult(err), nil
			}
			if err := requireString(payload, "lieferschein_nummer"); err != nil {
				return mcputil.ErrorResult(err), nil
			}
			if err := requireNonEmptyArray(payload, "positionen"); err != nil {
				return mcputil.ErrorResult(err), nil
			}

			result, err := runner.RunBuilder(ctx, "build_lieferschein.py", payload, "pdf")
			if err != nil {
				logger.Error("build_lieferschein failed", "err", err)
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.JSONResult(result), nil
		},
	)
}
