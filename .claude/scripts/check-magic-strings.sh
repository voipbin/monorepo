#!/usr/bin/env bash
# Check a single .go file for hardcoded direct resource type magic strings.
# Used by Claude Code PostToolUse hook on Write|Edit.
#
# Reads the tool input JSON from stdin to extract the file path.
# Exits 2 (block) if violations found, 0 otherwise.

set -euo pipefail

# Read tool input from stdin
INPUT=$(cat)
FILE_PATH=$(echo "$INPUT" | jq -r '.tool_input.file_path // .tool_input.file // empty' 2>/dev/null || true)

# Only check .go files
if [[ -z "$FILE_PATH" ]] || [[ "$FILE_PATH" != *.go ]]; then
    exit 0
fi

# Skip the constants definition file itself
if [[ "$FILE_PATH" == *"bin-direct-manager/models/direct/direct.go" ]]; then
    exit 0
fi

# Skip vendor, generated, and mock files
if [[ "$FILE_PATH" == *"/vendor/"* ]] || [[ "$FILE_PATH" == *"/gens/"* ]] || [[ "$FILE_PATH" == *"mock_"* ]]; then
    exit 0
fi

# Check for magic strings in resource type assignments
PATTERNS=(
    'ResourceType:\s*"(ai|ai_team|agent|queue|conference|extension)"'
    'ResourceType\s*=\s*"(ai|ai_team|agent|queue|conference|extension)"'
)

VIOLATIONS=0
for pattern in "${PATTERNS[@]}"; do
    while IFS=: read -r line content; do
        # Skip lines with nolint:magicstring override
        if echo "$content" | grep -q 'nolint:magicstring'; then
            continue
        fi
        echo "[Hook] Magic string at $FILE_PATH:$line: $content"
        echo "[Hook] Use dmdirect.ResourceType* constants from bin-direct-manager/models/direct/direct.go"
        VIOLATIONS=$((VIOLATIONS + 1))
    done < <(grep -nE "$pattern" "$FILE_PATH" 2>/dev/null || true)
done

if [ "$VIOLATIONS" -gt 0 ]; then
    echo ""
    echo "[Hook] Found $VIOLATIONS magic string violation(s). Use constants:"
    echo "  dmdirect.ResourceTypeAI, dmdirect.ResourceTypeAITeam, dmdirect.ResourceTypeAgent,"
    echo "  dmdirect.ResourceTypeQueue, dmdirect.ResourceTypeConference, dmdirect.ResourceTypeExtension"
    echo "  To suppress: add // nolint:magicstring on the line"
    exit 2
fi

exit 0
