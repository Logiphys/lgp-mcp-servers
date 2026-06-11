# Journal — lgp-mcp-servers

## Aktueller Fokus

Kein offener Task. Autotask-405-Fix und logiphys-ci-Sync sind abgeschlossen und deployt (11.06.2026).

## Zuletzt erledigt (mit Commit-SHAs)

- `6300032` fix(autotask): PATCH auf Entity-Collection mit ID im Body (405-Fix) — TDD, Suite grün
- `5d78696` feat(logiphys-ci): Sync mit Marketplace lgp-ci 2.x — Submodule auf `af4301e` (lgp-docs 2.2.0), Skill-Dir-Rename `logiphys-ci`→`lgp-ci` (mit Prod-Fallback), neues Tool `build_konzept`, `build_angebot`-Schema generisch (nur noch `empfaenger_zeilen` Pflicht)
- Gateway-Deploy auf Azure-VM (lgp-mcp-gateway): Gateway-Binary `bridge-v0.1.3-3-gf938398` **plus** Backend-Binaries `bin/autotask-mcp` + `bin/logiphys-ci-mcp` (v1.3.1-7-g5d78696) **plus** Skill-Dateien `skills/logiphys-ci/` (Inhalt = lgp-ci v2.2.0). Backups mit Suffix `.bak.20260611` auf der VM.
- Smoke-Test bestanden: Datev-Tickets 202371, 209994, 210216, 211264 (braun-steine) via `autotask_update_ticket` auf Status 5 geschlossen, per `get_ticket_details` verifiziert.
- Marketplace `26a23fa`: 405-Workaround aus lgp-autotask SKILL.md zurückgebaut (Skill 1.0.1, Plugin lgp-mcp-gateway 0.2.1); `af4301e` + Tag `lgp-docs/v2.2.0`: Bericht/Konzept-Release committet.
- gateway-Repo: origin/main per Fast-Forward auf `f938398` (Submodule-Pin 5d78696).

## Nächster konkreter Schritt

Optional: Release-Tag `v1.4.0` im Monorepo setzen → GitHub-Actions-Release für Techniker-Maschinen (lokale autotask-mcp-Instanzen haben den 405-Bug sonst weiterhin).

## Offene Entscheidungen / Blocker

- Pax8-Backend am Gateway meldet 401 (vorbestehend, API-Key prüfen)
- gateway-Repo: lokales main (`1373fbe`, Bridge-Refactor "phase R") ist von origin/main divergiert — vor nächstem Push auf neues origin/main rebasen (origin/main inzwischen weiter: Dependabot-Fixes via PR #83 gemerged)
- VM `bin/` enthält ~20 alte Backup-Binaries — gelegentlich aufräumen
- Marketplace: unfertige lgp-email-tools-Arbeit (sorter.py + neue Skills) liegt uncommitted im Working Tree

## Wichtige Findings

- **Gateway-Architektur:** Das Gateway spawnt Backend-MCP-Server als separate Binaries aus `/opt/lgp-mcp-gateway/bin/` — ein Gateway-Binary-Deploy ersetzt NICHT die Backends. Für Monorepo-Fixes immer die betroffenen `bin/<server>-mcp` mit deployen (GOOS=linux GOARCH=amd64).
- **Autotask REST:** PATCH nur auf `/{entity}` (Collection) mit `id` im Body; `PATCH /{entity}/{id}` → HTTP 405. DELETE dagegen mit ID im Pfad. Nach 405 öffnet die Gateway-Middleware den Circuit Breaker (~60 s auch Reads blockiert).
- **CI-Server auf der VM:** wird mit explizitem `LOGIPHYS_CI_SKILL_DIR` auf `skills/logiphys-ci` (alter Verzeichnisname) gestartet — Verzeichnisname bleibt, Inhalt ist lgp-ci 2.2.0. Binary-Default sucht `lgp-ci` mit Fallback `logiphys-ci`.
- **Deploy-Runbook** siehe `~/Developer/lgp-mcp-gateway/HANDOVER.md` (scp + systemctl; Service vorher stoppen, sonst "Text file busy").

---

## Historie

### 2026-06-11 — Autotask-405-Fix + logiphys-ci-Sync

Handover `docs/handover-archiv/2026-06-11-autotask-update-405.md` vollständig abgearbeitet: Fix per TDD (Test zuerst auf Collection-PATCH umgestellt, RED verifiziert, dann `Client.Update` gefixt), alle 5 update_*-Tools über die zentrale Methode mitgefixt. CI-Server-Abgleich ergab 33 Tage Submodule-Rückstand inkl. Breaking-Rename und fehlendem Konzept-Tool — komplett nachgezogen, inkl. Integrationstest für `build_konzept` (PDF-Magic-Bytes). Learning: Backend-Binaries auf der VM sind eigenständig deploybar und wurden beim ersten Deploy-Versuch übersehen (Gateway-Tausch allein reichte nicht — 405 blieb, bis `bin/autotask-mcp` ersetzt war).
