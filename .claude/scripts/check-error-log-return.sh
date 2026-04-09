#!/usr/bin/env bash
# Check a single .go file for the log-then-return anti-pattern.
# Detects: log.Errorf(...) immediately followed by return ... errors.Wrap
# Used by Claude Code PostToolUse hook on Write|Edit.
#
# Reads the tool input JSON from stdin to extract the file path.
# Exits 0 always (warning only, does not block).

set -euo pipefail

# Read tool input from stdin
INPUT=$(cat)
FILE_PATH=$(echo "$INPUT" | jq -r '.tool_input.file_path // .tool_input.file // empty' 2>/dev/null || true)

# Only check .go files
if [[ -z "$FILE_PATH" ]] || [[ "$FILE_PATH" != *.go ]]; then
    exit 0
fi

# Skip vendor, generated, and mock files
if [[ "$FILE_PATH" == *"/vendor/"* ]] || [[ "$FILE_PATH" == *"/gens/"* ]] || [[ "$FILE_PATH" == *"mock_"* ]]; then
    exit 0
fi

# Skip test files
if [[ "$FILE_PATH" == *"_test.go" ]]; then
    exit 0
fi

# Detect log.Errorf followed by return with errors.Wrap on the next non-empty line
VIOLATIONS=0
PREV_LINE=""
PREV_NUM=0
LINE_NUM=0
while IFS= read -r line; do
    LINE_NUM=$((LINE_NUM + 1))
    TRIMMED=$(echo "$line" | sed 's/^[[:space:]]*//')

    # Skip empty lines
    if [[ -z "$TRIMMED" ]]; then
        continue
    fi

    # Check if previous line was log.Errorf and current line is return with errors.Wrap
    if [[ "$PREV_LINE" =~ log\.(Errorf|Error)\( ]] && [[ "$TRIMMED" =~ ^return.*errors\.(Wrap|Wrapf)\( ]]; then
        echo "[Hook] Warning: log-then-return pattern at $FILE_PATH:$PREV_NUM-$LINE_NUM"
        echo "[Hook] Consider: wrap and propagate the error instead of logging at this level."
        echo "[Hook] See docs/coding-conventions.md §4.4 Error Propagation Pattern."
        VIOLATIONS=$((VIOLATIONS + 1))
    fi

    PREV_LINE="$TRIMMED"
    PREV_NUM=$LINE_NUM
done < "$FILE_PATH"

if [ "$VIOLATIONS" -gt 0 ]; then
    echo ""
    echo "[Hook] Found $VIOLATIONS log-then-return pattern(s). This is a warning — not blocked."
    echo "[Hook] Convention: inner functions should wrap+return errors; log at a reasonable level."
fi

# Always exit 0 — this is advisory only
exit 0
