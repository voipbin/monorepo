#!/usr/bin/env bash
# scripts/check-docs.sh — guards against root CLAUDE.md regrowth and missing category READMEs.
# Run from the monorepo root.
#
# NOT YET WIRED INTO CI. v1 ships this as an advisory local check; the cap is enforced
# only when a contributor remembers to run the script. Wiring it into .circleci/config.yml
# (or a Makefile lint-docs target called from CI) is a follow-up — see the
# 2026-04-27-claude-md-categorization design doc, "Open questions / future work".
# Until then, run manually as part of the verification workflow.

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
