#!/usr/bin/env bash
# Check Alembic migration files for table names that don't exist in the codebase.
# Used by Claude Code PostToolUse hook on Write|Edit.
#
# Reads the tool input JSON from stdin to extract the file path.
# Exits 2 (block) with warning if suspicious table names found, 0 otherwise.

set -euo pipefail

# Read tool input from stdin
INPUT=$(cat)
FILE_PATH=$(echo "$INPUT" | jq -r '.tool_input.file_path // .tool_input.file // empty' 2>/dev/null || true)

# Only check .py files in dbscheme-manager migration directories
if [[ -z "$FILE_PATH" ]] || [[ "$FILE_PATH" != *.py ]]; then
    exit 0
fi

if [[ "$FILE_PATH" != *"bin-dbscheme-manager"*"/versions/"* ]]; then
    exit 0
fi

# Extract table names from ALTER TABLE / CREATE TABLE / DROP TABLE statements
TABLES=$(grep -oiE '(ALTER|CREATE|DROP)\s+TABLE\s+(IF\s+(NOT\s+)?EXISTS\s+)?`?([a-zA-Z_][a-zA-Z0-9_]*)`?' "$FILE_PATH" 2>/dev/null \
    | sed -E 's/(ALTER|CREATE|DROP)\s+TABLE\s+(IF\s+(NOT\s+)?EXISTS\s+)?`?//i' \
    | sed 's/`//g' \
    | sort -u || true)

if [[ -z "$TABLES" ]]; then
    exit 0
fi

# Find the monorepo root (walk up from file path)
REPO_ROOT="$FILE_PATH"
while [[ "$REPO_ROOT" != "/" ]]; do
    REPO_ROOT=$(dirname "$REPO_ROOT")
    if [[ -f "$REPO_ROOT/CLAUDE.md" ]] && [[ -d "$REPO_ROOT/bin-call-manager" ]]; then
        break
    fi
done

if [[ "$REPO_ROOT" == "/" ]]; then
    exit 0
fi

WARNINGS=0
for table in $TABLES; do
    # Search for this table name as a constant in dbhandler files across all services
    FOUND=$(grep -rls "\"$table\"" "$REPO_ROOT"/bin-*/pkg/dbhandler/*.go 2>/dev/null | head -1 || true)

    if [[ -z "$FOUND" ]]; then
        echo "[Hook] WARNING: Table '$table' in migration not found in any dbhandler."
        echo "[Hook] Verify the table name by running: grep -r 'Table = \"' bin-*/pkg/dbhandler/ | grep -i '$(echo "$table" | sed "s/_/.*_/g")'"
        WARNINGS=$((WARNINGS + 1))
    fi
done

if [[ "$WARNINGS" -gt 0 ]]; then
    echo ""
    echo "[Hook] Found $WARNINGS unverified table name(s) in migration."
    echo "[Hook] Table naming convention: <domain>_<entities> (e.g., call_groupcalls, flow_flows)"
    echo "[Hook] NOT <service>_manager_<entities>"
    exit 2
fi

exit 0
