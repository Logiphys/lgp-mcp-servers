# Handover: `logiphys-ci-mcp` Server bauen

**Datum:** 2026-05-07
**Branch:** `feat/logiphys-ci-mcp`
**Vorherige Session:** Cowork (Sonnet 4.6) — Submodule eingehängt, Verzeichnisstruktur angelegt, Server-Code noch nicht geschrieben.

---

## TL;DR — Was zu tun ist

Im `lgp-mcp-servers`-Monorepo einen 10. Backend-MCP-Server `logiphys-ci-mcp` schreiben, der die Python-Helper-Scripts aus `external/logiphys-marketplace/plugins/lgp-docs/skills/logiphys-ci/scripts/` als Subprocess aufruft und CI-konforme DOCX/PDFs generiert. Code-Vorbild: `cmd/autotask-mcp/` + `pkg/autotask/`.

```bash
cd ~/Developer/lgp-mcp-servers
git checkout feat/logiphys-ci-mcp
git submodule status external/logiphys-marketplace
# Erwartet: f4937d4 oder neuer
ls cmd/logiphys-ci-mcp/        # leer — main.go fehlt
ls pkg/logiphys-ci/             # leer — runner.go + tools.go fehlen
```

---

## Verifikation am Anfang der Session

```bash
cd ~/Developer/lgp-mcp-servers
git branch --show-current      # → feat/logiphys-ci-mcp
git submodule status            # external/logiphys-marketplace muss eingehängt sein
ls cmd/                         # 10 Verzeichnisse, davon 9 mit main.go + 1 leeres logiphys-ci-mcp/
go test ./pkg/...               # alle bestehenden Tests grün

# SSH-Zugang zum Production-Gateway (für späteren Deploy)
ssh lgp-mcp-gateway 'whoami && systemctl is-active lgp-mcp-gateway'
# Erwartet: azureuser, active
```

Falls SSH-Alias nicht da: siehe Abschnitt „Server-Zugang" weiter unten.

---

## Architektur

```
Mitarbeiter / Langdock
        │ (MCP über HTTPS)
        ▼
┌─────────────────────────────┐
│ lgp-mcp-gateway (Azure VM)  │   Auth + RBAC + Audit
│ /opt/lgp-mcp-gateway/       │
└──────────┬──────────────────┘
           │ stdio (subprocess)
           ▼
┌─────────────────────────────┐
│ logiphys-ci-mcp (NEU)       │   Go-Binary, MCP-stdio
│ ./bin/logiphys-ci-mcp       │
└──────────┬──────────────────┘
           │ subprocess (exec.Command)
           ▼
┌─────────────────────────────┐
│ python3 build_brief.py …    │   Python-Helper aus Submodule
│ external/logiphys-marketplace/
│   plugins/lgp-docs/skills/
│     logiphys-ci/scripts/    │
└──────────┬──────────────────┘
           ▼
       DOCX / PDF
   (Base64 zurück über MCP)
```

**Source of Truth für die Helper-Scripts:** `github.com/Logiphys/logiphys-marketplace` — als Git-Submodule unter `external/logiphys-marketplace`. SHA-Pinning, kein Auto-Pull zur Laufzeit. Update-Workflow siehe Abschnitt „Sync-Strategie" weiter unten.

---

## Was schon vorbereitet ist

- Branch `feat/logiphys-ci-mcp` lokal angelegt
- Submodule `external/logiphys-marketplace` initial auf SHA `f4937d4` (= Marketplace `main` Stand 2026-05-07)
- Leere Verzeichnisse `cmd/logiphys-ci-mcp/` und `pkg/logiphys-ci/`

Noch nichts committet — du startest mit einem leeren working-tree-diff.

---

## Code zu schreiben

### 1. `cmd/logiphys-ci-mcp/main.go`

Vorbild: `cmd/autotask-mcp/main.go` (etwas anpassen — wir haben keinen externen API-Client, sondern einen lokalen `Runner`):

```go
package main

import (
    "log/slog"
    "os"
    "path/filepath"

    "github.com/mark3labs/mcp-go/server"

    "github.com/Logiphys/lgp-mcp-servers/pkg/config"
    logiphysci "github.com/Logiphys/lgp-mcp-servers/pkg/logiphys-ci"
    "github.com/Logiphys/lgp-mcp-servers/pkg/mcputil"
)

var (
    version   = "dev"
    buildDate = ""
)

func main() {
    logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
        Level: config.LogLevel(),
    }))

    skillDir := config.OptEnv("LOGIPHYS_CI_SKILL_DIR", defaultSkillDir())
    pythonBin := config.OptEnv("LOGIPHYS_CI_PYTHON_BIN", "python3")
    outputDir := config.OptEnv("LOGIPHYS_CI_OUTPUT_DIR", os.TempDir())

    runner := logiphysci.NewRunner(logiphysci.RunnerConfig{
        SkillDir:  skillDir,
        PythonBin: pythonBin,
        OutputDir: outputDir,
        Logger:    logger,
    })

    srv := server.NewMCPServer("logiphys-ci-mcp", version)
    logiphysci.RegisterTools(srv, runner, logger)
    mcputil.RegisterServerInfoTool(srv, mcputil.ServerInfo{
        Name:      "logiphys-ci-mcp",
        Version:   version,
        BuildDate: buildDate,
    })

    if err := server.ServeStdio(srv); err != nil {
        logger.Error("serve error", "err", err)
        os.Exit(1)
    }
}

// defaultSkillDir resolves the skill directory relative to the binary's
// runtime location: <bin>/../skills/logiphys-ci  (für Production-Deploy)
// fallback auf das Submodule für lokale Tests.
func defaultSkillDir() string {
    if exe, err := os.Executable(); err == nil {
        candidate := filepath.Join(filepath.Dir(exe), "..", "skills", "logiphys-ci")
        if _, err := os.Stat(candidate); err == nil {
            return candidate
        }
    }
    // local-dev fallback — Submodule-Pfad relativ zum Repo-Root
    return "external/logiphys-marketplace/plugins/lgp-docs/skills/logiphys-ci"
}
```

### 2. `pkg/logiphys-ci/runner.go`

Subprocess-Wrapper. JSON ins tempfile, Python aufrufen, Output-Datei lesen, Base64-encoden, Cleanup.

```go
package logiphysci

import (
    "context"
    "encoding/base64"
    "encoding/json"
    "fmt"
    "log/slog"
    "os"
    "os/exec"
    "path/filepath"
    "time"
)

type RunnerConfig struct {
    SkillDir  string         // .../logiphys-ci/
    PythonBin string         // "python3"
    OutputDir string         // /tmp oder /var/lib/lgp-mcp-gateway/ci-output
    Logger    *slog.Logger
}

type Runner struct {
    cfg RunnerConfig
}

func NewRunner(cfg RunnerConfig) *Runner { return &Runner{cfg: cfg} }

// RunBuilder ruft scripts/<scriptName> mit den gegebenen JSON-Daten auf und
// liefert {filename, mime_type, content_base64, size_bytes} zurück.
func (r *Runner) RunBuilder(ctx context.Context, scriptName string, payload any, ext string) (map[string]any, error) {
    // 1. Payload nach tempfile schreiben
    jsonFile, err := os.CreateTemp(r.cfg.OutputDir, "logiphys-ci-*.json")
    if err != nil {
        return nil, fmt.Errorf("tempfile: %w", err)
    }
    defer os.Remove(jsonFile.Name())
    if err := json.NewEncoder(jsonFile).Encode(payload); err != nil {
        jsonFile.Close()
        return nil, fmt.Errorf("encode payload: %w", err)
    }
    jsonFile.Close()

    // 2. Output-Pfad
    outFile := filepath.Join(r.cfg.OutputDir, fmt.Sprintf("logiphys-ci-%d.%s", time.Now().UnixNano(), ext))
    defer os.Remove(outFile)

    // 3. Python aufrufen
    scriptPath := filepath.Join(r.cfg.SkillDir, "scripts", scriptName)
    cmd := exec.CommandContext(ctx, r.cfg.PythonBin, scriptPath,
        "--json", jsonFile.Name(),
        "--output", outFile,
    )
    out, err := cmd.CombinedOutput()
    if err != nil {
        r.cfg.Logger.Error("python failed", "script", scriptName, "stderr", string(out), "err", err)
        return nil, fmt.Errorf("python %s failed: %w (stderr: %s)", scriptName, err, string(out))
    }

    // 4. Output lesen + Base64
    data, err := os.ReadFile(outFile)
    if err != nil {
        return nil, fmt.Errorf("read output: %w", err)
    }
    mime := mimeForExt(ext)
    return map[string]any{
        "filename":       filepath.Base(outFile),
        "mime_type":      mime,
        "content_base64": base64.StdEncoding.EncodeToString(data),
        "size_bytes":     len(data),
    }, nil
}

func mimeForExt(ext string) string {
    switch ext {
    case "docx":
        return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
    case "pdf":
        return "application/pdf"
    default:
        return "application/octet-stream"
    }
}
```

### 3. `pkg/logiphys-ci/tools.go`

Registrierung aller Tools:

```go
package logiphysci

import (
    "log/slog"

    "github.com/mark3labs/mcp-go/server"
)

func RegisterTools(srv *server.MCPServer, runner *Runner, logger *slog.Logger) {
    registerBriefTool(srv, runner, logger)
    registerAngebotTool(srv, runner, logger)
    registerBerichtTool(srv, runner, logger)
    registerLieferscheinTool(srv, runner, logger)
    registerMahnungTool(srv, runner, logger)
    registerVersionTool(srv, runner, logger)
}
```

### 4. `pkg/logiphys-ci/tools_brief.go` — Vorlage für die anderen 4

Tool-Definition mit JSON-Schema, Handler ruft `runner.RunBuilder("build_brief.py", payload, "docx")` auf.

JSON-Schema-Quelle: die `BriefData`-Dataclass in `external/logiphys-marketplace/plugins/lgp-docs/skills/logiphys-ci/scripts/build_brief.py`:

```python
@dataclass
class BriefData:
    empfaenger_zeilen: list[str]
    betreff: str
    body_paragraphs: list[str]
    datum: Optional[str] = None
    ansprechpartner: Optional[Ansprechpartner] = None
    unterschrift: bool = False
    unterschrift_after_index: Optional[int] = None
    sig_height_mm: float = 18.0
```

Tool-Schema in Go (mark3labs/mcp-go):

```go
mcp.NewToolWithRawSchema(
    "logiphys_ci_build_brief",
    "Erzeugt einen Logiphys-Geschäftsbrief als DOCX (CI-konform, DIN-5008-orientiert).",
    json.RawMessage(`{
      "type": "object",
      "required": ["empfaenger_zeilen", "betreff", "body_paragraphs"],
      "properties": {
        "empfaenger_zeilen": { "type": "array", "items": {"type": "string"}, "description": "Empfänger-Adressblock; erste Zeile wird automatisch fett (Klavika Medium)" },
        "betreff": { "type": "string" },
        "body_paragraphs": { "type": "array", "items": {"type": "string"} },
        "datum": { "type": "string", "description": "z.B. '06. Mai 2026', optional, default heute" },
        "ansprechpartner": {
          "type": "object",
          "properties": {
            "name": {"type": "string"},
            "telefon": {"type": "string"},
            "email": {"type": "string"}
          }
        },
        "unterschrift": { "type": "boolean", "default": false },
        "unterschrift_after_index": { "type": "integer" },
        "sig_height_mm": { "type": "number", "default": 18.0 }
      }
    }`),
)
```

### 5. Die anderen 4 Tool-Dateien

Schemas ableiten aus den Python-Dataclasses:

| Datei | Script | Output | Pflichtfelder (aus Python-Dataclass) |
|---|---|---|---|
| `tools_angebot.go` | `build_angebot.py` | pdf | `empfaenger_zeilen`, `angebotsnummer`, `gueltig_bis`, `einleitung`, `positionen` |
| `tools_bericht.go` | `build_bericht.py` | docx | `empfaenger_zeilen`, `berichtstitel`, `abschnitte` |
| `tools_lieferschein.go` | `build_lieferschein.py` | pdf | `lieferadresse_zeilen`, `lieferschein_nummer`, `positionen` |
| `tools_mahnung.go` | `build_mahnung.py` | docx | `empfaenger_zeilen`, `mahnstufe`, `offene_posten`, `frist_datum` |

Komplette JSON-Schemas siehe Docstrings am Anfang der jeweiligen Python-Files.

### 6. `tools_version.go` — Audit/Compliance-Tool

Liefert die SHA und Tag-Info des geladenen Submodules. Praktisch fürs Gateway-Audit-Log und für Langdock-User („Welche Version habe ich gerade benutzt?").

```go
// Schema: kein Input.
// Response:
// {
//   "server_version": "logiphys-ci-mcp/v1.0.0",
//   "skill_marketplace_sha": "f4937d4",
//   "skill_marketplace_tag": "lgp-docs/v1.1.0",
//   "loaded_at": "2026-05-07T08:35:12Z"
// }
//
// SHA + Tag werden zur Build-Zeit aus dem Submodule via `git -C external/logiphys-marketplace`
// extrahiert und als Linker-Flags reingegeben (`-X 'pkg/logiphys-ci.SkillSHA=...'`)
// Siehe Makefile-Anpassung weiter unten.
```

---

## Makefile-Anpassung

Im `Makefile` einen Target hinzufügen:

```makefile
.PHONY: build-logiphys-ci-mcp
build-logiphys-ci-mcp:
	@SKILL_SHA=$$(git -C external/logiphys-marketplace rev-parse --short HEAD) ; \
	 SKILL_TAG=$$(git -C external/logiphys-marketplace describe --tags --abbrev=0 2>/dev/null || echo none) ; \
	 LOADED_AT=$$(date -u +%Y-%m-%dT%H:%M:%SZ) ; \
	 go build -ldflags "\
	   -X 'main.version=$(VERSION)' \
	   -X 'main.buildDate=$(BUILD_DATE)' \
	   -X 'github.com/Logiphys/lgp-mcp-servers/pkg/logiphys-ci.SkillSHA=$$SKILL_SHA' \
	   -X 'github.com/Logiphys/lgp-mcp-servers/pkg/logiphys-ci.SkillTag=$$SKILL_TAG' \
	   -X 'github.com/Logiphys/lgp-mcp-servers/pkg/logiphys-ci.LoadedAt=$$LOADED_AT'" \
	   -o bin/logiphys-ci-mcp ./cmd/logiphys-ci-mcp
```

Plus `build-all`-Target um `build-logiphys-ci-mcp` erweitern.

---

## Lokaler Smoketest

```bash
cd ~/Developer/lgp-mcp-servers
make build-logiphys-ci-mcp

# Helper-Scripts brauchen python-docx + reportlab + pypdf + pillow
pip3 install --user python-docx reportlab pypdf pillow
# (otf2ttf wird nicht zur Laufzeit gebraucht — TTFs liegen schon im Marketplace-Submodule)

# MCP-Server lokal mit einer Tool-Initialize-Sequence anwerfen — siehe pkg/<server>/tools_test.go für Pattern
echo '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}' | ./bin/logiphys-ci-mcp 2>/dev/null | head -20

# Erwartet: JSON-Response mit den 5 + version Tools im "tools"-Array
```

---

## Tests

Pattern aus `pkg/autotask/tools_test.go` übernehmen:
- Table-driven mit `-race`
- Mock-Runner injizieren um Python-Aufruf zu vermeiden
- JSON-Schema-Validation testen
- Edge-Cases: fehlende Pflichtfelder, ungültige `mahnstufe`, ungültiges Datum

Mindestens ein Smoketest **mit echter Python-Ausführung** in einem `*_integration_test.go` (mit Build-Tag `integration`) — ruft das Skript wirklich auf, prüft DOCX-Magic-Bytes (`PK\x03\x04`).

---

## Sync-Strategie: Marketplace ↔ MCP-Server

**Grundprinzip:** SHA-gepinntes Submodule. Kein Auto-Pull zur Laufzeit. Updates werden explizit ausgelöst.

**Update-Loop bei Skill-Änderung (5 Befehle, in `release-ci.sh` paketiert):**

```bash
# 1. Marketplace-Edit + Tag (im logiphys-marketplace Repo)
cd ~/Developer/logiphys-marketplace
# … edit, commit …
git tag lgp-docs/v1.2.0
git push origin main lgp-docs/v1.2.0

# 2. lgp-mcp-servers: Submodule bump
cd ~/Developer/lgp-mcp-servers
git submodule update --remote external/logiphys-marketplace
git add external/logiphys-marketplace
git commit -m "deps: bump logiphys-marketplace to lgp-docs/v1.2.0"
git push

# 3. lgp-mcp-gateway: Submodule bump
cd ~/Developer/lgp-mcp-gateway
git submodule update --remote external/lgp-mcp-servers
git add external/lgp-mcp-servers
git commit -m "deps: bump lgp-mcp-servers (logiphys-ci v1.2.0)"
git push

# 4. Build + Deploy
GOOS=linux GOARCH=amd64 make build-all
scp gateway lgp-mcp-gateway:/tmp/gateway-new
scp bin/logiphys-ci-mcp lgp-mcp-gateway:/tmp/logiphys-ci-mcp-new
rsync -av --delete \
   external/lgp-mcp-servers/external/logiphys-marketplace/plugins/lgp-docs/skills/logiphys-ci/ \
   lgp-mcp-gateway:/tmp/skills-logiphys-ci-new/

# 5. Restart auf dem Server
ssh lgp-mcp-gateway "
  sudo systemctl stop lgp-mcp-gateway && \
  sudo cp /tmp/gateway-new /opt/lgp-mcp-gateway/bin/gateway && \
  sudo cp /tmp/logiphys-ci-mcp-new /opt/lgp-mcp-gateway/bin/logiphys-ci-mcp && \
  sudo rsync -av --delete /tmp/skills-logiphys-ci-new/ /opt/lgp-mcp-gateway/skills/logiphys-ci/ && \
  sudo systemctl start lgp-mcp-gateway && \
  sudo systemctl status lgp-mcp-gateway --no-pager | head -5
"
```

**Skript-Vorschlag:** `bin/release-ci.sh` im Gateway-Repo, Aufruf `release-ci.sh lgp-docs/v1.2.0` macht 2–5 automatisch.

---

## Gateway-Konfig (kommt nach Server-Code)

Im `lgp-mcp-gateway`-Repo (separates Repo, separater Commit/Push):

### `config/backends.yaml` ergänzen

```yaml
  logiphys-ci:
    type: go
    secrets: []   # CI-Server hat keine externen API-Keys
```

### `config/servers/logiphys-ci.yaml` neu

```yaml
name: logiphys-ci
type: binary
enabled: false
command: ./bin/logiphys-ci-mcp
env:
  LOGIPHYS_CI_SKILL_DIR: "/opt/lgp-mcp-gateway/skills/logiphys-ci"
  LOGIPHYS_CI_PYTHON_BIN: "python3"
  LOGIPHYS_CI_OUTPUT_DIR: "/var/lib/lgp-mcp-gateway/ci-output"
  LOGIPHYS_CI_ACCESS_TIER: "1"
tiering:
  tier2: []
  tier3: []
```

### `config/tiering.yaml` ergänzen

```yaml
logiphys-ci:
  tier2: []
  tier3: []
```

### `config/roles.yaml` — jede Rolle bekommt Tier 1

```yaml
roles:
  MCP-Technik:
    autotask: 3
    …
    logiphys-ci: 1
  MCP-Vertrieb:
    …
    logiphys-ci: 1
  MCP-Beratung:
    …
    logiphys-ci: 1
  MCP-Automation:
    …
    logiphys-ci: 1
  MCP-GL:
    …
    logiphys-ci: 1
```

### `deploy/Dockerfile` ergänzen

Python + Helper-Pakete + Skill-Verzeichnis ins Image:

```dockerfile
RUN apt-get update && apt-get install -y --no-install-recommends \
       python3 python3-pip \
    && rm -rf /var/lib/apt/lists/*
RUN pip3 install --break-system-packages --no-cache-dir \
       python-docx==1.2.0 reportlab>=4.0 pypdf>=3.0 pillow

# Skill-Files (Helper-Scripts + Klavika-Fonts + Logo + stammdaten.json)
COPY external/lgp-mcp-servers/external/logiphys-marketplace/plugins/lgp-docs/skills/logiphys-ci /opt/lgp-mcp-gateway/skills/logiphys-ci
```

---

## Server-Zugang (für späteren Deploy)

`~/.ssh/config` auf dem MacBook:

```
Host lgp-mcp-gateway
    HostName lgp-mcp-gateway.germanywestcentral.cloudapp.azure.com
    User azureuser
    IdentityFile ~/.ssh/id_ed25519
    IdentitiesOnly yes
```

Test: `ssh lgp-mcp-gateway 'whoami && systemctl is-active lgp-mcp-gateway'`

**Falls Connection timeout:** NSG der Azure-Resource-Group `LGP_AI_Tools` / NSG `lgp-mcp-gateway-nsg` checken — die externe IPv4 muss whitelisted sein. Aktuelle IP holen mit `curl -s -4 https://api.ipify.org`. Anpassung im [Azure-Portal](https://portal.azure.com/#@logiphys.de/resource/subscriptions/e0fc62ac-89f9-4041-a95a-269128cea878/resourceGroups/LGP_AI_Tools/providers/Microsoft.Network/networkSecurityGroups/lgp-mcp-gateway-nsg/overview) oder via `az network nsg rule update` (`brew install azure-cli` falls noch nicht installiert).

---

## Check-out / Verification nach Deploy

```bash
ssh lgp-mcp-gateway "sudo journalctl -u lgp-mcp-gateway -n 50 --no-pager"
# Im Log sollte stehen: "starting backend server: name=logiphys-ci command=./bin/logiphys-ci-mcp"

# Tool-Liste prüfen über Gateway-API (mit gültigem API-Key):
curl -sk https://lgp-mcp-gateway.germanywestcentral.cloudapp.azure.com:8443/api/v1/tools \
  -H "Authorization: Bearer $LGP_GATEWAY_KEY" | jq '.tools[] | select(.server == "logiphys-ci")'
# Erwartet: 6 Tools (5 build_* + version)

# Smoketest: einen Brief generieren und Base64-Antwort prüfen
# (am einfachsten via Cowork oder Langdock — der Server wird beim ersten Tool-Call gestartet)
```

---

## Offene Punkte (außerhalb dieser Session)

- **ACA-Migration:** Der Plan vom 10. April (`docs/plans/2026-04-10-azure-deployment-design.md` im Gateway-Repo) sieht Azure Container Apps mit GHCR vor. Aktuell läuft die VM. Frage an Andreas: ist der ACA-Plan verworfen oder nur verschoben? Beeinflusst die Deploy-Strategie nicht *jetzt*, aber langfristig schon.
- **`logiphys-sign` Skill:** In den Python-Helpern sind `unterschrift`-Felder vorbereitet, der Sign-Skill selbst ist nicht in dieser Session entstanden — wenn der separat kommt, muss das MCP-Tool-Schema die Felder mit unterstützen (Schema oben enthält sie schon).
- **Versionierung im Marketplace** für gemeinsamen Tag-Sync zwischen Repo, Submodule und Server-Audit-Trail — beim ersten release-ci.sh-Run klären.

---

## Datei-Inventar (Kontext für die nächste Session)

```
~/Developer/lgp-mcp-servers/
├── HANDOVER-logiphys-ci-mcp.md   ← DIESES DOKUMENT
├── cmd/
│   ├── autotask-mcp/main.go      ← Vorbild für main.go
│   ├── … (8 weitere)
│   └── logiphys-ci-mcp/          ← LEER, main.go fehlt
├── pkg/
│   ├── autotask/
│   │   ├── client.go             ← Vorbild für runner.go (anderer Use-Case, aber ähnliche Struktur)
│   │   ├── tools.go              ← Vorbild für RegisterTools
│   │   └── tools_*.go            ← Vorbild für tools_brief.go etc.
│   ├── … (8 weitere)
│   ├── config/                   ← MustEnv, OptEnv, LogLevel
│   ├── mcputil/                  ← RegisterServerInfoTool
│   └── logiphys-ci/              ← LEER
└── external/
    └── logiphys-marketplace/     ← Submodule, SHA f4937d4 (Python-Helper hier drin)
        └── plugins/lgp-docs/skills/logiphys-ci/
            ├── SKILL.md
            ├── scripts/
            │   ├── build_brief.py        ← Python-Helper (5 Stück)
            │   ├── build_angebot.py
            │   ├── build_bericht.py
            │   ├── build_lieferschein.py
            │   └── build_mahnung.py
            └── assets/
                ├── stammdaten.json       ← Logiphys-Pflichtangaben
                ├── logiphys_logo.png
                └── fonts/Klavika-*.{otf,ttf}
```

```
~/Developer/lgp-mcp-gateway/
├── HANDOVER.md                   ← April-Handover, Datto-RMM-Fixes (separates Thema)
├── docs/plans/2026-04-10-azure-deployment-*.md  ← ACA-Migration-Plan (offen)
├── config/
│   ├── backends.yaml             ← logiphys-ci hier ergänzen
│   ├── servers/                  ← logiphys-ci.yaml hier neu
│   ├── tiering.yaml              ← logiphys-ci hier ergänzen
│   └── roles.yaml                ← Tier 1 für alle Rollen
└── deploy/Dockerfile             ← Python + Helper-Pakete + Skill-Verzeichnis
```

```
~/Developer/logiphys-marketplace/
├── docs/HANDOVER-2026-05-07.md   ← Marketplace-seitiger Handover (Mitarbeiter-Onboarding etc.)
└── plugins/lgp-docs/skills/logiphys-ci/  ← Source of Truth — wird via Submodule eingebunden
```

---

## Reihenfolge-Empfehlung

1. **Server-Code schreiben** (cmd + pkg, Code-Vorbild autotask)
2. **Lokal bauen** mit `make build-logiphys-ci-mcp`
3. **Smoketest** (jsonrpc tools/list, dann ein build_brief mit minimalem JSON)
4. **Tests** schreiben (unit + integration)
5. **Branch pushen + PR**
6. **Gateway-Repo:** Submodule bump, Konfig-Files, Dockerfile
7. **Lokal Gateway bauen** mit `make build-all`
8. **Gateway lokal starten** und über Cowork den neuen Server testen
9. **Deploy auf Azure-VM** (siehe „Sync-Strategie" Schritt 4–5)
10. **`bin/release-ci.sh`** in Gateway-Repo committen für künftige Updates

Wenn alles steht, ist Langdock-User mit MCP-Auth automatisch der erste echte Konsument.
