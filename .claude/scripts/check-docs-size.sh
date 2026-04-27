#!/usr/bin/env bash
# Run scripts/check-docs.sh after edits to root CLAUDE.md or docs/*.
# Used by Claude Code PostToolUse hook on Write|Edit.
#
# Reads the tool input JSON from stdin to extract the file path.
# Exits 0 on pass; on fail, prints the script's error and exits 0
# (warning rather than blocking — the cap is advisory at the hook layer
# and CI will be the eventual enforcement point).

set -euo pipefail

# Read tool input from stdin
INPUT=$(cat)
FILE_PATH=$(echo "$INPUT" | jq -r '.tool_input.file_path // .tool_input.file // empty' 2>/dev/null || true)

if [[ -z "$FILE_PATH" ]]; then
    exit 0
fi

# Only run when ROOT CLAUDE.md or root docs/* is edited. Service-level
# CLAUDE.md (e.g., bin-pipecat-manager/CLAUDE.md) and service-level docs are
# explicitly out of scope — the cap applies to root CLAUDE.md only, and the
# README check only reads root docs/<category>/README.md. Match both
# absolute paths (Write tool) and repo-relative paths.
case "$FILE_PATH" in
    # Reject service-level paths first
    bin-*/CLAUDE.md|*/bin-*/CLAUDE.md|bin-*/docs/*|*/bin-*/docs/*) exit 0 ;;
    # Accept root-level paths
    CLAUDE.md|*/CLAUDE.md|docs/*|*/docs/*) ;;
    *) exit 0 ;;
esac

# Find the monorepo root (where scripts/check-docs.sh lives)
SCRIPT_DIR=$(cd "$(dirname "$0")/../.." && pwd)
if [[ ! -x "$SCRIPT_DIR/scripts/check-docs.sh" ]]; then
    exit 0
fi

# Run the cap check
OUTPUT=$("$SCRIPT_DIR/scripts/check-docs.sh" 2>&1) || {
    echo ""
    echo "[Hook] check-docs.sh failed for $FILE_PATH:"
    echo "$OUTPUT" | sed 's/^/[Hook]   /'
    echo "[Hook] Move detail to docs/<category>/<topic>.md, or run \`make lint-docs\` to re-verify."
    echo ""
    exit 0
}

exit 0
