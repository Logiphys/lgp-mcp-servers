# Journal — lgp-mcp-servers

## Aktueller Fokus

Issue #4 (Branch `feat/autotask-impersonation`): Gateway-Benutzer aus `_meta["logiphys.de/actor"]` übernehmen, `ImpersonationResourceId` bei schreibenden Autotask-Aufrufen. Gehört zu Logiphys/lgp-mcp-gateway#22 (Spec A4 auf Branch `feat/traceability`).

Plan:
1. `pkg/mcputil/actor.go`: `Actor`, `ActorFromRequest`, `WithActor`/`ActorFromContext`, Handler-Wrapper `WithActorContext`; `LogValue` ohne UPN.
2. `pkg/apihelper`: request-spezifische Header per Context (`WithRequestHeaders`).
3. `pkg/autotask`: `addTool` wickelt jeden Handler in `WithActorContext` (auch `execute_tool` → Kontext trägt Actor weiter); `Create/CreateChild/Update/Delete/DeleteChild` setzen `ImpersonationResourceId`, wenn Actor mit Resource-ID im Kontext; `Get/Query/GetFieldInfo` nie.
4. `FormatCreateResult` liefert zusätzlich `itemId` (Gateway-Audit `from: result, path: itemId`), `id` bleibt.
5. Tests table-driven mit `-race`; `make test`, `make lint`, `make check-docs`; PR mit `Refs #4`, nicht mergen.

Stand 2026-10-18: Schritte 1–5 umgesetzt (`15015ad`), PR offen, wartet auf Andreas' Review/Freigabe.

## Zuletzt erledigt (mit Commit-SHAs)

- `15015ad` feat(autotask): Gateway-Actor aus `_meta` + `ImpersonationResourceId` bei Writes, `itemId` in Anlege-Ergebnissen (#4) — TDD, `make test` grün
- `6300032` fix(autotask): PATCH auf Entity-Collection mit ID im Body (405-Fix) — TDD, Suite grün
- `5d78696` feat(logiphys-ci): Sync mit Marketplace lgp-ci 2.x — Submodule auf `af4301e` (lgp-docs 2.2.0), Skill-Dir-Rename `logiphys-ci`→`lgp-ci` (mit Prod-Fallback), neues Tool `build_konzept`, `build_angebot`-Schema generisch (nur noch `empfaenger_zeilen` Pflicht)
- Gateway-Deploy auf Azure-VM (lgp-mcp-gateway): Gateway-Binary `bridge-v0.1.3-3-gf938398` **plus** Backend-Binaries `bin/autotask-mcp` + `bin/logiphys-ci-mcp` (v1.3.1-7-g5d78696) **plus** Skill-Dateien `skills/logiphys-ci/` (Inhalt = lgp-ci v2.2.0). Backups mit Suffix `.bak.20260611` auf der VM.
- Smoke-Test bestanden: Datev-Tickets 202371, 209994, 210216, 211264 (braun-steine) via `autotask_update_ticket` auf Status 5 geschlossen, per `get_ticket_details` verifiziert.
- Marketplace `26a23fa`: 405-Workaround aus lgp-autotask SKILL.md zurückgebaut (Skill 1.0.1, Plugin lgp-mcp-gateway 0.2.1); `af4301e` + Tag `lgp-docs/v2.2.0`: Bericht/Konzept-Release committet.
- gateway-Repo: origin/main per Fast-Forward auf `f938398` (Submodule-Pin 5d78696).

## Nächster konkreter Schritt

PR für #4 reviewen/mergen (Andreas). Danach: Autotask-Berechtigung für Impersonation setzen und an einem Testobjekt prüfen, ob eine nicht erlaubte Impersonation fehlschlägt oder auf „API User“ zurückfällt. Submodul-Zeiger im Gateway **nicht** hier anheben — kommt mit lgp-mcp-gateway#197.

Früher notiert: Optional: Release-Tag `v1.4.0` im Monorepo setzen → GitHub-Actions-Release für Techniker-Maschinen (lokale autotask-mcp-Instanzen haben den 405-Bug sonst weiterhin).

## Offene Entscheidungen / Blocker

- Vorbestehend auf `main`: `make lint` (errcheck in `pkg/logiphysci/runner.go:70`) und `make check-docs` (README-Serverliste ohne logiphys-ci-mcp, kein pkg-Mapping) schlagen fehl — nicht Teil von #4
- Pax8-Backend am Gateway meldet 401 (vorbestehend, API-Key prüfen)
- gateway-Repo: lokales main (`1373fbe`, Bridge-Refactor "phase R") ist von origin/main divergiert — vor nächstem Push auf neues origin/main rebasen (origin/main inzwischen weiter: Dependabot-Fixes via PR #83 gemerged)
- VM `bin/` enthält ~20 alte Backup-Binaries — gelegentlich aufräumen
- Marketplace: unfertige lgp-email-tools-Arbeit (sorter.py + neue Skills) liegt uncommitted im Working Tree

## Wichtige Findings

- **Autotask-Impersonation:** Header `ImpersonationResourceId`; Doku nennt Add/Edit, Delete nur für wenige Entities. Verhalten bei verweigerter Impersonation ist undokumentiert. `execute_tool` baut einen frischen Request ohne `_meta` — der Actor läuft deshalb über den Context weiter (`mcputil.WithActorContext` in `addTool`).
- **check-docs:** Server-Namen in Backticks im README-Fließtext brechen die Tool-Zählung (Skript greift Tabellenzeile per `` `name` ``).
- **Gateway-Architektur:** Das Gateway spawnt Backend-MCP-Server als separate Binaries aus `/opt/lgp-mcp-gateway/bin/` — ein Gateway-Binary-Deploy ersetzt NICHT die Backends. Für Monorepo-Fixes immer die betroffenen `bin/<server>-mcp` mit deployen (GOOS=linux GOARCH=amd64).
- **Autotask REST:** PATCH nur auf `/{entity}` (Collection) mit `id` im Body; `PATCH /{entity}/{id}` → HTTP 405. DELETE dagegen mit ID im Pfad. Nach 405 öffnet die Gateway-Middleware den Circuit Breaker (~60 s auch Reads blockiert).
- **CI-Server auf der VM:** wird mit explizitem `LOGIPHYS_CI_SKILL_DIR` auf `skills/logiphys-ci` (alter Verzeichnisname) gestartet — Verzeichnisname bleibt, Inhalt ist lgp-ci 2.2.0. Binary-Default sucht `lgp-ci` mit Fallback `logiphys-ci`.
- **Deploy-Runbook** siehe `~/Developer/lgp-mcp-gateway/HANDOVER.md` (scp + systemctl; Service vorher stoppen, sonst "Text file busy").

---

## Historie

### 2026-10-18 — Issue #4: Gateway-Actor + Autotask-Impersonation

Spec A4 (lgp-mcp-gateway `feat/traceability`) im Backend umgesetzt. Header-Name gegen Autotask-REST-Doku geprüft. Request-spezifische Header über Context in `apihelper` statt neuer Post/Patch/Delete-Varianten — Credential-Header gewinnen immer. Alle 19 Anlege-Tools liefen schon über `FormatCreateResult`; dort `itemId` ergänzt.

### 2026-06-11 — Autotask-405-Fix + logiphys-ci-Sync

Handover `docs/handover-archiv/2026-06-11-autotask-update-405.md` vollständig abgearbeitet: Fix per TDD (Test zuerst auf Collection-PATCH umgestellt, RED verifiziert, dann `Client.Update` gefixt), alle 5 update_*-Tools über die zentrale Methode mitgefixt. CI-Server-Abgleich ergab 33 Tage Submodule-Rückstand inkl. Breaking-Rename und fehlendem Konzept-Tool — komplett nachgezogen, inkl. Integrationstest für `build_konzept` (PDF-Magic-Bytes). Learning: Backend-Binaries auf der VM sind eigenständig deploybar und wurden beim ersten Deploy-Versuch übersehen (Gateway-Tausch allein reichte nicht — 405 blieb, bis `bin/autotask-mcp` ersetzt war).
