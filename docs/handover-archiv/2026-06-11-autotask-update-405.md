# Handover: Autotask-Updates schlagen mit HTTP 405 fehl (PATCH auf falsche Route)

> **ERLEDIGT 11.06.2026** — Fix `6300032`, CI-Sync `5d78696`, Gateway-Deploy
> `bridge-v0.1.3-3-gf938398` inkl. Backend-Binaries; alle vier Datev-Tickets
> via `autotask_update_ticket` auf Status 5 geschlossen und verifiziert;
> SKILL.md-Workaround zurückgebaut (Marketplace `26a23fa`).
> Details: `docs/journal.md`. Ursprünglicher Handover-Text unverändert darunter.

**Datum:** 11.06.2026 · **Gefunden von:** Claude-Session mit AZ (Datev-Tickets schließen) · **Priorität:** Hoch — sämtliche Autotask-Update-Tools im Gateway sind unbenutzbar

## Problem

Jeder `autotask_update_ticket`-Aufruf über das MCP-Gateway scheitert deterministisch:

```
HTTP 405: {"Message":"The requested resource does not support http method 'PATCH'."}
```

Folgewirkung: Nach dem 405 öffnet die Gateway-Middleware den **Circuit Breaker** („service temporarily unavailable"), weitere Autotask-Calls werden ~60 s lang abgewiesen — auch Reads.

## Ursache (lokalisiert)

[pkg/autotask/client.go:211-219](pkg/autotask/client.go) — `Client.Update()` sendet:

```
PATCH /{entity}/{id}        ← z. B. PATCH /Tickets/202371
```

Die Autotask REST API (V1.0) unterstützt PATCH **nur auf der Entity-Collection** mit der ID im Body:

```
PATCH /{entity}             ← z. B. PATCH /Tickets
Body: {"id": 202371, "status": 5}
```

Quelle: 405 am 11.06.2026 reproduzierbar verifiziert; der dokumentierte Workaround in `logiphys-marketplace/plugins/lgp-mcp-gateway/skills/lgp-autotask/SKILL.md` (Abschnitt „autotask_update_ticket -> 405") nutzt genau dieses Collection-PATCH-Format erfolgreich.

## Fix

In `Client.Update()` die ID vom Pfad in den Body verschieben:

```go
func (c *Client) Update(ctx context.Context, entity string, id int, data map[string]any) error {
    _, err := c.middleware.Execute(ctx, func() (any, error) {
        body := make(map[string]any, len(data)+1)
        for k, v := range data {
            body[k] = v
        }
        body["id"] = id
        _, err := c.http.Patch(ctx, "/"+entity, body)
        return nil, err
    })
    return err
}
```

**Mit anpassen:**

- [pkg/autotask/client_test.go:210](pkg/autotask/client_test.go) prüft die PATCH-Methode — Erwartung auf Collection-Pfad (`/Tickets`) und `id` im Request-Body umstellen.
- Alle Tools, die `Client.Update()` nutzen, sind automatisch mitgefixt (u. a. `update_ticket`, `update_company`, `update_service_call`, `update_ticket_charge`, `update_quote_item` — siehe `tools_*.go`). Gegenprüfen: DELETE `/{entity}/{id}` ist bei Autotask korrekt — nur PATCH ist betroffen.

## Verifikation nach Fix + Deploy

1. `go test ./pkg/autotask/...`
2. Deploy aufs Gateway (Azure, lgp-mcp-gateway.germanywestcentral.cloudapp.azure.com)
3. Realer Smoke-Test steht bereit: Die vier Datev-Tickets **202371, 209994, 210216, 211264** (T20250704.0035, T20251006.0064, T20251009.0018, T20251023.0023) sollen auf **Status 5 (Abgeschlossen)** — Freigabe durch AZ liegt vom 11.06.2026 vor. Läuft das über `autotask_update_ticket` durch, ist der Fix bestätigt und die Aufgabe gleich miterledigt.

## Aufräumen danach

- Workaround-Abschnitt „autotask_update_ticket -> 405" in der lgp-autotask SKILL.md (logiphys-marketplace) entfernen bzw. auf „behoben seit <Version>" kürzen — inkl. des Python-Direct-REST-Snippets (liest Secrets aus `.env`, soll nicht der Dauerweg sein).
- Hinweis ebenda („Zeiteintrag auf abgeschlossenem Ticket … Status-Update via PATCH-Workaround") auf das reguläre Tool umstellen.
