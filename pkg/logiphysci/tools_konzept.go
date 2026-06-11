package logiphysci

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/Logiphys/lgp-mcp-servers/pkg/mcputil"
)

// konzeptSchema mirrors the KonzeptData dataclass in build_konzept.py.
const konzeptSchema = `{
  "type": "object",
  "required": ["titel"],
  "additionalProperties": false,
  "properties": {
    "titel": {"type": "string", "description": "Dokumenttitel auf dem Deckblatt, z.B. 'Diagnose- und Lösungsbericht'"},
    "untertitel": {"type": "string", "description": "Untertitel auf dem Deckblatt"},
    "auftraggeber": {"type": "string", "description": "Auftraggeber/Kunde, erscheint auf dem Deckblatt"},
    "bezug": {"type": "string", "description": "Bezug, z.B. Ticket-/Vorgangs-/Berichtsnummer"},
    "datum": {"type": "string", "description": "z.B. '11.06.2026' — optional, default heute"},
    "version": {"type": "string", "description": "Dokumentversion, z.B. '1.0'"},
    "dokumenttitel_kopfzeile": {"type": "string", "description": "Kurztitel für die Kopfzeile der Inhaltsseiten (default: titel)"},
    "ersteller_name": {"type": "string", "description": "Verantwortliche Person auf dem Deckblatt"},
    "ersteller_rolle": {"type": "string"},
    "ersteller_email": {"type": "string"},
    "ersteller_telefon": {"type": "string"},
    "vorspann_blocks": {
      "type": "array",
      "description": "Inhaltsblöcke der Inhaltsseiten. String = Absatz; Objekt: {type:'paragraph', text, bold?} | {type:'heading', text, level:1-3} | {type:'bullets'|'numbered', items:[...]} | {type:'spacer', mm} | {type:'table', rows:[[Kopf...],[...]], col_widths?: relative Anteile, z.B. [0.22, 0.78]}",
      "items": {"anyOf": [{"type": "string"}, {"type": "object"}]}
    }
  }
}`

func registerKonzeptTool(srv *server.MCPServer, runner BuildRunner, logger *slog.Logger) {
	srv.AddTool(
		mcp.NewToolWithRawSchema(
			"build_konzept",
			"Erzeugt ein Logiphys-Konzept-/Bericht-PDF mit Deckblatt und Inhaltsseiten (CI-konform). Standard-Weg für ausführliche Diagnose-/Lösungsberichte, Konzepte und Whitepaper.",
			json.RawMessage(konzeptSchema),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			payload := req.GetArguments()
			if err := requireString(payload, "titel"); err != nil {
				return mcputil.ErrorResult(err), nil
			}

			result, err := runner.RunBuilder(ctx, "build_konzept.py", payload, "pdf")
			if err != nil {
				logger.Error("build_konzept failed", "err", err)
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.JSONResult(result), nil
		},
	)
}
