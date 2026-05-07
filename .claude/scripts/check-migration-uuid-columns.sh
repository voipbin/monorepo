#!/usr/bin/env bash
# Check Alembic migration files for UUID columns declared as VARCHAR(36).
# Project convention: UUID columns must be BINARY(16) (matches Go uuid.UUID
# wire format and the rest of the monorepo's tables).
# Used by Claude Code PostToolUse hook on Write|Edit.
#
# Reads the tool input JSON from stdin to extract the file path.
# Exits 2 (block) with warning if VARCHAR(36) UUID columns are detected, 0 otherwise.

set -euo pipefail

INPUT=$(cat)
FILE_PATH=$(echo "$INPUT" | jq -r '.tool_input.file_path // .tool_input.file // empty' 2>/dev/null || true)

if [[ -z "$FILE_PATH" ]] || [[ "$FILE_PATH" != *.py ]]; then
    exit 0
fi

if [[ "$FILE_PATH" != *"bin-dbscheme-manager"*"/versions/"* ]]; then
    exit 0
fi

# Look for column declarations or MODIFY statements where a column whose name
# ends in `id` (id, customer_id, agent_id, ...) is given VARCHAR(36).
# Permissive whitespace match. Case-insensitive.
SUSPECT_LINES=$(grep -niE '(^|[^a-z_])([a-z_]*id)[[:space:]]+varchar\([[:space:]]*36[[:space:]]*\)' "$FILE_PATH" 2>/dev/null \
    | grep -viE '\-\-.*varchar' \
    || true)

if [[ -z "$SUSPECT_LINES" ]]; then
    exit 0
fi

echo "[Hook] BLOCKED: detected UUID column(s) declared as VARCHAR(36) in $FILE_PATH" >&2
echo "[Hook]" >&2
echo "[Hook] UUID columns must be BINARY(16), not VARCHAR(36):" >&2
echo "[Hook]   - Go's MySQL driver sends uuid.UUID as 16 raw bytes." >&2
echo "[Hook]   - VARCHAR(36) silently stores those bytes as garbage characters," >&2
echo "[Hook]     and JOINs against other tables (BINARY(16)) never match." >&2
echo "[Hook]" >&2
echo "[Hook] Suspect lines:" >&2
echo "$SUSPECT_LINES" | sed 's/^/[Hook]   /' >&2
echo "[Hook]" >&2
echo "[Hook] Use BINARY(16) instead. See docs/conventions/database.md §7.0a (UUID Column Type)." >&2

exit 2
