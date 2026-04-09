#!/usr/bin/env bash
# Check if an RST source file was edited without rebuilt HTML.
# Used by Claude Code PostToolUse hook on Write|Edit.
#
# Reads the tool input JSON from stdin to extract the file path.
# Exits 0 with a reminder when RST source is edited.

set -euo pipefail

# Read tool input from stdin
INPUT=$(cat)
FILE_PATH=$(echo "$INPUT" | jq -r '.tool_input.file_path // .tool_input.file // empty' 2>/dev/null || true)

# Only check .rst files under bin-api-manager/docsdev/source/
if [[ -z "$FILE_PATH" ]] || [[ "$FILE_PATH" != *.rst ]]; then
    exit 0
fi

if [[ "$FILE_PATH" != *"bin-api-manager/docsdev/source/"* ]]; then
    exit 0
fi

echo ""
echo "[Hook] RST source file modified: $FILE_PATH"
echo "[Hook] REMINDER: Before committing, you MUST rebuild the HTML docs:"
echo "  cd bin-api-manager/docsdev && rm -rf build && python3 -m sphinx -M html source build"
echo "  git add -f bin-api-manager/docsdev/build/"
echo ""

exit 0
