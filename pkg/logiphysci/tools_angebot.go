package logiphysci

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/Logiphys/lgp-mcp-servers/pkg/mcputil"
)

// angebotSchema mirrors the AngebotData dataclass in build_angebot.py.
const angebotSchema = `{
  "type": "object",
  "required": ["empfaenger_zeilen", "angebotsnummer", "gueltig_bis", "einleitung", "positionen"],
  "additionalProperties": false,
  "properties": {
    "empfaenger_zeilen": {
      "type": "array",
      "items": {"type": "string"},
      "description": "Empfänger-Adressblock (DIN 5008)"
    },
    "angebotsnummer": {"type": "string", "description": "z.B. 'AN-2026-001'"},
    "gueltig_bis": {"type": "string", "description": "Gültigkeitsdatum, z.B. '31.05.2026'"},
    "einleitung": {"type": "string", "description": "Einleitungstext über der Positionstabelle"},
    "positionen": {
      "type": "array",
      "minItems": 1,
      "items": {
        "type": "object",
        "required": ["pos", "menge", "beschreibung", "ep", "gp"],
        "additionalProperties": false,
        "properties": {
          "pos": {"type": "integer", "description": "Positionsnummer"},
          "menge": {"type": "string", "description": "Menge inkl. Einheit, z.B. '2 Stk' oder '5 h'"},
          "beschreibung": {"type": "string"},
          "ep": {"type": "number", "description": "Einzelpreis netto"},
          "gp": {"type": "number", "description": "Gesamtpreis netto"}
        }
      }
    },
    "mwst_satz": {"type": "number", "description": "MwSt-Satz in Prozent, default 19.0"},
    "zahlungsbedingungen": {"type": "string"},
    "hinweise": {"type": "array", "items": {"type": "string"}},
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

func registerAngebotTool(srv *server.MCPServer, runner BuildRunner, logger *slog.Logger) {
	srv.AddTool(
		mcp.NewToolWithRawSchema(
			"build_angebot",
			"Erzeugt ein Logiphys-Angebot als PDF (CI-konform, DIN-5008-orientiert).",
			json.RawMessage(angebotSchema),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			payload := req.GetArguments()
			for _, k := range []string{"angebotsnummer", "gueltig_bis", "einleitung"} {
				if err := requireString(payload, k); err != nil {
					return mcputil.ErrorResult(err), nil
				}
			}
			if err := requireStringArray(payload, "empfaenger_zeilen"); err != nil {
				return mcputil.ErrorResult(err), nil
			}
			if err := requireNonEmptyArray(payload, "positionen"); err != nil {
				return mcputil.ErrorResult(err), nil
			}

			result, err := runner.RunBuilder(ctx, "build_angebot.py", payload, "pdf")
			if err != nil {
				logger.Error("build_angebot failed", "err", err)
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.JSONResult(result), nil
		},
	)
}
