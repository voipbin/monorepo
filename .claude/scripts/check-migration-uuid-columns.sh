#!/usr/bin/env bash
# Check Alembic migration files for UUID columns declared as VARCHAR(36)
# inside the upgrade() function.
#
# Project convention: UUID columns must be BINARY(16). Go's MySQL driver
# does not consistently send uuid.UUID as 16 bytes — only the
# `commondatabasehandler.PrepareFields()` pipeline (with `db:"id,uuid"`
# tags) or explicit `.Bytes()` calls produce the 16-byte form. VARCHAR(36)
# UUID columns therefore tend to silently store inconsistent byte
# representations across code paths and break cross-table joins.
#
# Used by Claude Code PostToolUse hook on Write|Edit.
# Reads the tool input JSON from stdin to extract the file path.
# Exits 2 (block) with warning if VARCHAR(36) UUID columns are detected
# inside upgrade(); exits 0 otherwise.
#
# The check is intentionally limited to upgrade() so legitimate
# downgrade-to-VARCHAR rollbacks are not blocked.

set -euo pipefail

INPUT=$(cat)
FILE_PATH=$(echo "$INPUT" | jq -r '.tool_input.file_path // .tool_input.file // empty' 2>/dev/null || true)

if [[ -z "$FILE_PATH" ]] || [[ "$FILE_PATH" != *.py ]]; then
    exit 0
fi

if [[ "$FILE_PATH" != *"bin-dbscheme-manager"*"/versions/"* ]]; then
    exit 0
fi

# Extract only the upgrade() function body (lines from `def upgrade()` up to
# the next `def ` declaration). awk preserves original line numbers via
# NR substitution so the warning output is accurate.
UPGRADE_BODY=$(awk '
    /^def upgrade\(/   { in_up = 1; next }
    /^def [a-zA-Z_]+\(/ { in_up = 0 }
    in_up              { printf "%d:%s\n", NR, $0 }
' "$FILE_PATH" 2>/dev/null || true)

if [[ -z "$UPGRADE_BODY" ]]; then
    exit 0
fi

# Match column declarations / MODIFY statements where:
#   - column name is exactly `id`, or ends in `_id` (so `paid`, `valid`, `void`,
#     `grid`, `kid`, etc. are not flagged)
#   - followed by VARCHAR(36) (any whitespace inside the parens)
# Skip Python `#` comments and SQL `--` comments.
SUSPECT_LINES=$(echo "$UPGRADE_BODY" \
    | grep -viE '^[0-9]+:[[:space:]]*#' \
    | grep -viE '^[0-9]+:.*--' \
    | grep -iE '(^|[^a-z_])(id|[a-z_]+_id)[[:space:]]+varchar\([[:space:]]*36[[:space:]]*\)' \
    || true)

if [[ -z "$SUSPECT_LINES" ]]; then
    exit 0
fi

echo "[Hook] BLOCKED: detected UUID column(s) declared as VARCHAR(36) in upgrade() of $FILE_PATH" >&2
echo "[Hook]" >&2
echo "[Hook] UUID columns must be BINARY(16), not VARCHAR(36):" >&2
echo "[Hook]   - The 16-byte representation is only sent on the wire when call sites" >&2
echo "[Hook]     use commondatabasehandler.PrepareFields() (via 'db:\"id,uuid\"' tag)" >&2
echo "[Hook]     or explicit uuid.UUID.Bytes() at the dbhandler call site." >&2
echo "[Hook]   - Otherwise gofrs/uuid sends a 36-char string, which a VARCHAR(36)" >&2
echo "[Hook]     column accepts but a BINARY(16) column would reject — and worse," >&2
echo "[Hook]     bootstrap migrations that copy from BINARY(16) source columns into" >&2
echo "[Hook]     a VARCHAR(36) target store inconsistent bytes that never compare equal." >&2
echo "[Hook]" >&2
echo "[Hook] Suspect lines (line:content):" >&2
echo "$SUSPECT_LINES" | sed 's/^/[Hook]   /' >&2
echo "[Hook]" >&2
echo "[Hook] Use BINARY(16) instead. See docs/conventions/database.md §7.0a (UUID Column Type)." >&2

exit 2
