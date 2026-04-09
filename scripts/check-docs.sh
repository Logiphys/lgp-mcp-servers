#!/usr/bin/env bash
# check-docs.sh — verify documentation consistency with code
# Run in CI to catch stale docs before release.
# Compatible with both macOS and Linux grep (no -P flag).
set -euo pipefail

ERRORS=0
WARNINGS=0

error() { echo "ERROR: $1"; ERRORS=$((ERRORS + 1)); }
warn()  { echo "WARNING: $1"; WARNINGS=$((WARNINGS + 1)); }

# --- 1. Server list: cmd/ directories vs README table vs Makefile ---
echo "=== Checking server list consistency ==="

SERVERS_CMD=$(ls -d cmd/*-mcp 2>/dev/null | sed 's|cmd/||' | sort)
SERVERS_README=$(grep -oE '`[a-z]+-[a-z-]+-mcp`|`[a-z]+-mcp`' README.md | tr -d '`' | grep -- '-mcp$' | sort -u)
SERVERS_MAKEFILE=$(grep '^SERVERS' Makefile | sed 's/.*:= //' | tr ' ' '\n' | sort)

if [ "$SERVERS_CMD" != "$SERVERS_README" ]; then
    error "Server list mismatch between cmd/ and README"
    diff <(echo "$SERVERS_CMD") <(echo "$SERVERS_README") || true
fi

if [ "$SERVERS_CMD" != "$SERVERS_MAKEFILE" ]; then
    error "Server list mismatch between cmd/ and Makefile"
    diff <(echo "$SERVERS_CMD") <(echo "$SERVERS_MAKEFILE") || true
fi

echo "  cmd/ servers:      $(echo "$SERVERS_CMD" | wc -l | tr -d ' ')"
echo "  README servers:    $(echo "$SERVERS_README" | wc -l | tr -d ' ')"
echo "  Makefile servers:  $(echo "$SERVERS_MAKEFILE" | wc -l | tr -d ' ')"

# --- 2. Tool counts: code vs README ---
echo ""
echo "=== Checking tool counts ==="

# Map server name to pkg directory
server_to_pkg() {
    case "$1" in
        autotask-mcp)       echo "pkg/autotask" ;;
        datto-backup-mcp)   echo "pkg/dattobackup" ;;
        datto-edr-mcp)      echo "pkg/dattoedr" ;;
        datto-network-mcp)  echo "pkg/dattonetwork" ;;
        datto-rmm-mcp)      echo "pkg/dattormm" ;;
        datto-uc-mcp)       echo "pkg/dattouc" ;;
        itglue-mcp)         echo "pkg/itglue" ;;
        myitprocess-mcp)    echo "pkg/myitprocess" ;;
        rocketcyber-mcp)    echo "pkg/rocketcyber" ;;
        *) echo "" ;;
    esac
}

TOTAL_CODE=0
TOTAL_README=0

for server in $SERVERS_CMD; do
    pkg_dir=$(server_to_pkg "$server")
    if [ -z "$pkg_dir" ] || [ ! -d "$pkg_dir" ]; then
        error "No pkg directory found for $server"
        continue
    fi

    # Count mcp.NewTool calls + 1 for server_info
    code_count=$(grep -r 'mcp\.NewTool' "$pkg_dir" --include='*.go' | wc -l | tr -d ' ')
    code_count=$((code_count + 1))  # +1 for server_info tool

    # Extract count from README table: | `server-name` | ... | NN | ... |
    readme_count=$(grep "\`$server\`" README.md | sed 's/.*| *\([0-9][0-9]*\) *|.*/\1/' || echo "0")

    TOTAL_CODE=$((TOTAL_CODE + code_count))
    TOTAL_README=$((TOTAL_README + readme_count))

    if [ "$code_count" != "$readme_count" ]; then
        error "$server: code has $code_count tools, README says $readme_count"
    else
        echo "  $server: $code_count tools OK"
    fi
done

# Check total in README header
readme_total=$(grep -oE '[0-9]+ tools' README.md | head -1 | grep -oE '[0-9]+' || echo "0")
if [ "$TOTAL_CODE" != "$readme_total" ]; then
    error "Total tools: code has $TOTAL_CODE, README header says $readme_total"
else
    echo "  Total: $TOTAL_CODE tools OK"
fi

# --- 3. Architecture paths: README references vs actual directories ---
echo ""
echo "=== Checking architecture paths ==="

# Extract pkg/ and internal/ paths from README architecture block
readme_paths=$(grep -oE '(pkg|internal)/[a-z]+/' README.md | sort -u || true)
for path in $readme_paths; do
    if [ ! -d "$path" ]; then
        error "README references $path but directory does not exist"
    fi
done

# Check for internal/ references that should be pkg/
internal_refs=$(grep -c 'internal/' README.md || true)
if [ "$internal_refs" -gt 0 ]; then
    warn "README still references 'internal/' ($internal_refs times) — packages were moved to pkg/"
fi

# --- 4. CHANGELOG: check for current version entry ---
echo ""
echo "=== Checking CHANGELOG ==="

if [ -f CHANGELOG.md ]; then
    latest_tag=$(git describe --tags --abbrev=0 2>/dev/null || echo "none")
    if [ "$latest_tag" != "none" ]; then
        if ! grep -q "\[$latest_tag\]" CHANGELOG.md; then
            warn "CHANGELOG.md has no entry for latest tag $latest_tag"
        else
            echo "  CHANGELOG has entry for $latest_tag OK"
        fi
    fi
else
    warn "No CHANGELOG.md found"
fi

# --- 5. Stale feature references ---
echo ""
echo "=== Checking for stale references ==="

# Access tier env vars should not appear in README (feature removed in v1.2.0)
if grep -q 'ACCESS_TIER' README.md; then
    error "README still references ACCESS_TIER environment variables (removed in v1.2.0)"
fi

# Config examples should not reference access tiers
if [ -d config/ ]; then
    tier_refs=$(grep -rl 'ACCESS_TIER' config/ 2>/dev/null || true)
    if [ -n "$tier_refs" ]; then
        warn "Config examples still reference ACCESS_TIER: $tier_refs"
    fi
fi

# --- Summary ---
echo ""
echo "=== Summary ==="
echo "  Errors:   $ERRORS"
echo "  Warnings: $WARNINGS"

if [ "$ERRORS" -gt 0 ]; then
    echo "FAILED: $ERRORS error(s) found"
    exit 1
fi

if [ "$WARNINGS" -gt 0 ]; then
    echo "PASSED with $WARNINGS warning(s)"
fi

echo "All checks passed."
