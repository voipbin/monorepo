#!/usr/bin/env bash
# scripts/check-service-docs.sh — warn when service source changes are staged without
# corresponding docs/ updates.
#
# Invocation points:
#   - .git/hooks/pre-commit  (automatic, soft-warn — does NOT block commit)
#   - manual: bash scripts/check-service-docs.sh
#
# Exit codes: always 0 (warn only — never blocks a commit).
#
# Trigger rules:
#   Source change                                  → Docs file to update
#   ─────────────────────────────────────────────────────────────────────
#   pkg/listenhandler/main.go                      → docs/architecture.md  (routing table)
#   cmd/*/main.go or pkg/subscribehandler/main.go  → docs/architecture.md  (events)
#   internal/config/*.go or cmd/*/init.go          → docs/operations.md    (config table)
#   models/**/*.go                                 → docs/domain.md        (domain entities)
#   go.mod (replace directives)                    → docs/dependencies.md  (local deps)
#
# Re-extraction command (outbound deps, routing, config, events):
#   bash docs/reference/extractor.sh <service-dir>

# ── Collect staged files ────────────────────────────────────────────────────
STAGED=$(git diff --cached --name-only --diff-filter=ACMR 2>/dev/null || true)

if [ -z "$STAGED" ]; then
    exit 0
fi

# ── Per-service warning accumulator ─────────────────────────────────────────
# WARN_LINES is a newline-separated list of "svc|doc_file|reason" triples.
WARN_LINES=""

_add_warn() {
    local svc="$1" doc="$2" reason="$3"
    # Suppress if the matching docs file is also staged
    if echo "$STAGED" | grep -qE "^${svc}/${doc}$"; then
        return
    fi
    WARN_LINES="${WARN_LINES}${svc}|${doc}|${reason}"$'\n'
}

while IFS= read -r f; do
    # Only care about bin-* and voip-* service paths
    svc=$(echo "$f" | grep -oE '^(bin|voip)-[^/]+' || true)
    [ -z "$svc" ] && continue

    if echo "$f" | grep -qE "^${svc}/pkg/listenhandler/main\.go$"; then
        _add_warn "$svc" "docs/architecture.md" "listenhandler (routing table)"
    fi

    if echo "$f" | grep -qE "^${svc}/(cmd/[^/]+/main|pkg/subscribehandler/main)\.go$"; then
        _add_warn "$svc" "docs/architecture.md" "subscribeTargets (events section)"
    fi

    if echo "$f" | grep -qE "^${svc}/(internal/config/|cmd/[^/]+/init\.go)"; then
        _add_warn "$svc" "docs/operations.md" "config flags"
    fi

    if echo "$f" | grep -qE "^${svc}/models/"; then
        _add_warn "$svc" "docs/domain.md" "models/"
    fi

    if echo "$f" | grep -qE "^${svc}/go\.mod$"; then
        if git diff --cached "$f" 2>/dev/null | grep -q "^[+-].*replace "; then
            _add_warn "$svc" "docs/dependencies.md" "go.mod replace directives"
        fi
    fi
done <<< "$STAGED"

# Deduplicate (same svc+doc may fire from multiple triggers)
WARN_LINES=$(echo "$WARN_LINES" | sort -u | grep . || true)

if [ -z "$WARN_LINES" ]; then
    exit 0
fi

# ── Print warnings ───────────────────────────────────────────────────────────
echo ""
echo "========================================================================"
echo "  NOTICE: Service source changed — consider updating docs/"
echo "========================================================================"
echo ""
echo "The following services have source changes that may require doc updates."
echo "This is a reminder, not a blocker — commit proceeds either way."
echo ""

# Group by service
PREV_SVC=""
while IFS='|' read -r svc doc reason; do
    [ -z "$svc" ] && continue
    if [ "$svc" != "$PREV_SVC" ]; then
        [ -n "$PREV_SVC" ] && echo ""
        echo "  ${svc}:"
        PREV_SVC="$svc"
    fi
    echo "    → ${doc}   (changed: ${reason})"
done <<< "$WARN_LINES"
echo ""

echo "Re-extraction (routing / events / config / deps):"
echo "$WARN_LINES" | cut -d'|' -f1 | sort -u | while read -r svc; do
    [ -n "$svc" ] && echo "  bash docs/reference/extractor.sh ${svc}"
done
echo ""
echo "To suppress: stage the relevant docs/*.md file(s) alongside the source change."
echo "========================================================================"
echo ""

exit 0
