package logiphysci

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/Logiphys/lgp-mcp-servers/pkg/mcputil"
)

// mahnungSchema mirrors the MahnungData dataclass in build_mahnung.py.
const mahnungSchema = `{
  "type": "object",
  "required": ["empfaenger_zeilen", "mahnstufe", "offene_posten", "frist_datum"],
  "additionalProperties": false,
  "properties": {
    "empfaenger_zeilen": {
      "type": "array",
      "items": {"type": "string"},
      "description": "Empfänger-Adressblock (DIN 5008)"
    },
    "mahnstufe": {
      "type": "integer",
      "enum": [0, 1, 2, 3],
      "description": "0=Zahlungserinnerung, 1=1. Mahnung, 2=2. Mahnung, 3=letzte Mahnung"
    },
    "offene_posten": {
      "type": "array",
      "minItems": 1,
      "items": {
        "type": "object",
        "required": ["rechnungsnr", "rechnungsdatum", "faellig_seit", "betrag"],
        "additionalProperties": false,
        "properties": {
          "rechnungsnr": {"type": "string"},
          "rechnungsdatum": {"type": "string", "description": "z.B. '15.04.2026'"},
          "faellig_seit": {"type": "string", "description": "z.B. '30.04.2026'"},
          "betrag": {"type": "number", "description": "Bruttobetrag in EUR"},
          "mahngebuehr": {"type": "number", "description": "default 0.0"},
          "verzugszinsen": {"type": "number", "description": "default 0.0"}
        }
      }
    },
    "frist_datum": {"type": "string", "description": "Zahlungsfrist, z.B. '21.05.2026'"},
    "extra_body": {"type": "array", "items": {"type": "string"}, "description": "zusätzliche Absätze über/unter dem Standardtext"},
    "absender_name": {"type": "string", "description": "Name unter der Grußformel"},
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

func registerMahnungTool(srv *server.MCPServer, runner BuildRunner, logger *slog.Logger) {
	srv.AddTool(
		mcp.NewToolWithRawSchema(
			"build_mahnung",
			"Erzeugt eine Logiphys-Mahnung als DOCX (CI-konform, mit Mahnstufe 0–3).",
			json.RawMessage(mahnungSchema),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			payload := req.GetArguments()
			if err := requireStringArray(payload, "empfaenger_zeilen"); err != nil {
				return mcputil.ErrorResult(err), nil
			}
			if err := requireInt(payload, "mahnstufe"); err != nil {
				return mcputil.ErrorResult(err), nil
			}
			if err := requireNonEmptyArray(payload, "offene_posten"); err != nil {
				return mcputil.ErrorResult(err), nil
			}
			if err := requireString(payload, "frist_datum"); err != nil {
				return mcputil.ErrorResult(err), nil
			}
			if num, ok := toInt(payload["mahnstufe"]); !ok || num < 0 || num > 3 {
				return mcputil.ErrorResult(fmt.Errorf("mahnstufe must be 0, 1, 2, or 3")), nil
			}

			result, err := runner.RunBuilder(ctx, "build_mahnung.py", payload, "docx")
			if err != nil {
				logger.Error("build_mahnung failed", "err", err)
				return mcputil.ErrorResult(err), nil
			}
			return mcputil.JSONResult(result), nil
		},
	)
}

// toInt converts JSON-decoded numeric values to int.
func toInt(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int32:
		return int(n), true
	case int64:
		return int(n), true
	case float32:
		return int(n), true
	case float64:
		return int(n), true
	default:
		return 0, false
	}
}
