#!/usr/bin/env bash
# Warn when a service source file is written/edited without the corresponding
# docs/ file also being updated. Fires as a Claude Code PostToolUse hook on Write|Edit.
# Always exits 0 (warn only — never blocks).
#
# Trigger rules (matches scripts/check-service-docs.sh):
#   pkg/listenhandler/main.go              → docs/architecture.md  (routing table)
#   cmd/*/main.go, subscribehandler/main.go → docs/architecture.md (events)
#   internal/config/*.go, cmd/*/init.go    → docs/operations.md    (config flags)
#   models/.../*.go (any depth)            → docs/domain.md        (domain entities)
#   go.mod (replace directives only)       → docs/dependencies.md  (local deps)

set -euo pipefail

INPUT=$(cat)
FILE_PATH=$(echo "$INPUT" | jq -r '.tool_input.file_path // .tool_input.file // empty' 2>/dev/null || true)

if [[ -z "$FILE_PATH" ]]; then
    exit 0
fi

# Extract service name
SVC=$(echo "$FILE_PATH" | grep -oE '(bin|voip)-[^/]+' | head -1 || true)
if [[ -z "$SVC" ]]; then
    exit 0
fi

DOC_FILE=""
REASON=""

if echo "$FILE_PATH" | grep -qE "${SVC}/pkg/listenhandler/main\.go$"; then
    DOC_FILE="docs/architecture.md"; REASON="routing table (listenhandler)"
elif echo "$FILE_PATH" | grep -qE "${SVC}/(cmd/[^/]+/main|pkg/subscribehandler/main)\.go$"; then
    DOC_FILE="docs/architecture.md"; REASON="events section (subscribeTargets)"
elif echo "$FILE_PATH" | grep -qE "${SVC}/(internal/config/|cmd/[^/]+/init\.go)"; then
    DOC_FILE="docs/operations.md"; REASON="config flags"
elif echo "$FILE_PATH" | grep -qE "${SVC}/models/"; then
    DOC_FILE="docs/domain.md"; REASON="domain entities"
elif echo "$FILE_PATH" | grep -qE "${SVC}/go\.mod$"; then
    DOC_FILE="docs/dependencies.md"; REASON="go.mod (check if replace directives changed)"
else
    exit 0
fi

# Don't warn if the file being edited IS the docs file
if echo "$FILE_PATH" | grep -qE "${SVC}/${DOC_FILE}$"; then
    exit 0
fi

echo ""
echo "[Hook] ${SVC}: consider updating ${DOC_FILE}"
echo "[Hook] Reason: ${REASON} was modified"
echo "[Hook] Re-extract: bash docs/reference/extractor.sh ${SVC}"
echo ""

exit 0
