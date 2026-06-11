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
// Since lgp-ci 2.x build_angebot is a generic document builder: only
// empfaenger_zeilen is required, everything else has script-side defaults.
const angebotSchema = `{
  "type": "object",
  "required": ["empfaenger_zeilen"],
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
    "anrede": {"type": "string", "description": "Anrede zwischen Betreff und Einleitung, z.B. 'Sehr geehrter Herr Muster,' — optional"},
    "betreff": {"type": "string", "description": "Eigene Betreffzeile statt 'ANGEBOT' — macht das Dokument zum generischen Geschäftsdokument"},
    "positionen": {
      "type": "array",
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
      },
      "description": "Positionstabelle — optional; ohne Positionen entfällt die Tabelle"
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
    },
    "meta_zeilen": {
      "type": "array",
      "items": {"type": "array", "items": {"type": "string"}, "minItems": 2, "maxItems": 2},
      "description": "Eigene [Label, Wert]-Paare im Metablock; ersetzen Angebotsnummer/Gültig-bis/Zahlung"
    },
    "vorspann_blocks": {
      "type": "array",
      "description": "Inhaltsblöcke vor der Positionstabelle. String = Absatz; Objekt: {type:'paragraph', text, bold?} | {type:'heading', text, level:1-3} | {type:'bullets'|'numbered', items:[...]} | {type:'spacer', mm} | {type:'table', rows:[[Kopf...],[...]], col_widths?: relative Anteile}",
      "items": {"anyOf": [{"type": "string"}, {"type": "object"}]}
    },
    "nachspann_blocks": {
      "type": "array",
      "description": "Inhaltsblöcke nach der Positionstabelle, gleiche Struktur wie vorspann_blocks",
      "items": {"anyOf": [{"type": "string"}, {"type": "object"}]}
    },
    "unterschrift": {"type": "boolean", "description": "Absender-Unterschriftsblock einfügen (default false)"},
    "absender_grussformel": {"type": "string", "description": "default 'Mit freundlichen Grüßen'"},
    "absender_name": {"type": "string", "description": "Name unter der Absender-Unterschrift"},
    "sig_width_mm": {"type": "number", "description": "Breite des Unterschriftsbilds in mm, default 50"},
    "sig_height_mm": {"type": "number", "description": "Höhe des Unterschriftsbilds in mm, default 18"},
    "auftraggeber_signatur": {
      "anyOf": [
        {"type": "boolean"},
        {
          "type": "object",
          "additionalProperties": false,
          "properties": {
            "annahme_text": {"type": "string"},
            "heading": {"type": "string", "description": "default 'Auftragsannahme'"},
            "ort_datum_label": {"type": "string"},
            "unterschrift_label": {"type": "string"},
            "unterzeichner_name": {"type": "string"}
          }
        }
      ],
      "description": "Annahmeerklärung mit Unterschriftslinie für den Kunden; true = Defaults"
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
			if err := requireStringArray(payload, "empfaenger_zeilen"); err != nil {
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
