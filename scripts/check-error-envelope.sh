#!/usr/bin/env bash
# scripts/check-error-envelope.sh — guards against the internal
# Domain field re-leaking into external HTTP error responses.
#
# The bin-api-manager error envelope is built exclusively by
# bin-api-manager/lib/apierror/EnvelopeFor. Any "domain": literal
# inside an open-coded gin.H{} envelope under server/, lib/middleware/,
# or lib/service/ is a regression. Test files are exempt because they
# may legitimately mention "domain" in absence-of-domain assertions.
#
# Invocation: from the monorepo root.
#   make lint-error-envelope
#
# Exits non-zero on any match.

set -euo pipefail

SCOPES=(
  bin-api-manager/server
  bin-api-manager/lib/middleware
  bin-api-manager/lib/service
)

MATCHES=0

for scope in "${SCOPES[@]}"; do
  if [[ ! -d "$scope" ]]; then
    continue
  fi
  while IFS= read -r line; do
    MATCHES=$((MATCHES + 1))
    echo "$line" >&2
  done < <(grep -rEn '"domain"\s*:' "$scope" --include="*.go" --exclude="*_test.go" || true)
done

if [[ $MATCHES -gt 0 ]]; then
  echo "" >&2
  echo "FAIL: found $MATCHES open-coded \"domain\": literal(s) in api-manager non-test files." >&2
  echo "The external HTTP error envelope MUST NOT include \"domain\". Use bin-api-manager/lib/apierror.EnvelopeFor." >&2
  exit 1
fi

echo "OK: no open-coded \"domain\": literals in api-manager error sites."
