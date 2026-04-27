#!/usr/bin/env bash
# scripts/check-docs.sh — guards against root CLAUDE.md regrowth and missing category READMEs.
# Run from the monorepo root.
#
# Invocation points:
#   - make lint-docs                                  (manual)
#   - .claude/scripts/check-docs-size.sh              (Claude Code PostToolUse hook,
#                                                      fires on Write|Edit of CLAUDE.md
#                                                      or docs/* during AI work sessions)
#   - CI integration is a follow-up enhancement.

set -euo pipefail

ROOT_CAP=350
ROOT_LINES=$(wc -l < CLAUDE.md)
if [[ $ROOT_LINES -gt $ROOT_CAP ]]; then
  echo "FAIL: root CLAUDE.md is $ROOT_LINES lines (cap $ROOT_CAP). Move detail to docs/<category>/<topic>.md." >&2
  exit 1
fi

for category in architecture conventions workflows patterns reference; do
  if [[ ! -f "docs/$category/README.md" ]]; then
    echo "FAIL: docs/$category/README.md missing." >&2
    exit 1
  fi
done

echo "OK: root CLAUDE.md = $ROOT_LINES lines, all category READMEs present."
