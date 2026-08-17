#!/bin/bash
# VoIPBin - substitute a real image tag into a Komodo compose fragment's
# __IMAGE_TAG__ placeholder, in place.
#
# Replaces bump-image-digest.sh's role for services on the Komodo deploy
# path (VOIP-1342): no versions.lock, no digest resolution - the image
# reference is a plain tag (the same $CIRCLE_SHA1 the build step already
# pushed), substituted directly into the compose fragment that
# komodo-api-deploy.sh will push to Komodo. See
# docs/plans/2026-08-16-komodo-call-manager-cutover-design.md §3.
#
# Usage:
#   ./.circleci/scripts/render-image-tag.sh <compose-file> <tag>
#
#   <compose-file>  Path to a komodo/docker-compose.yml containing exactly
#                   one __IMAGE_TAG__ placeholder (per image line - a file
#                   with multiple services each needs their own call, or
#                   the placeholder needs to be unique per line, which it
#                   already will be as long as the tag substituted is the
#                   same for all of them, since this is a plain string
#                   replace of every occurrence).
#   <tag>           The tag to substitute in, e.g. $CIRCLE_SHA1. Must look
#                   like a plausible Docker tag (enforced below) - this is
#                   not a full Docker tag grammar validator, just a
#                   sanity check against the empty-string/whitespace/
#                   shell-metacharacter failure modes that would otherwise
#                   produce a broken-but-plausible-looking compose file.
set -euo pipefail

log_error() { echo "[ERROR] $*" >&2; }

usage() {
  sed -n '2,/^[^#]/{/^[^#]/!p}' "${BASH_SOURCE[0]}" | sed 's/^# \{0,1\}//'
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

if [[ $# -ne 2 ]]; then
  usage
  exit 1
fi

COMPOSE_FILE="$1"
TAG="$2"

if [[ ! -f "$COMPOSE_FILE" ]]; then
  log_error "compose-file does not exist: $COMPOSE_FILE"
  exit 1
fi

# Docker tags are [A-Za-z0-9_][A-Za-z0-9._-]{0,127}; this is a deliberately
# looser check (just rejects empty/whitespace/shell-metacharacter input,
# not the full grammar) since the real validation that matters is "did the
# build step actually push something at this tag", which this script has
# no way to check anyway.
if [[ -z "$TAG" || ! "$TAG" =~ ^[A-Za-z0-9][A-Za-z0-9._-]*$ ]]; then
  log_error "tag must be a non-empty string of [A-Za-z0-9._-] characters, starting alphanumeric (got: '$TAG')"
  exit 1
fi

if ! grep -q '__IMAGE_TAG__' "$COMPOSE_FILE"; then
  log_error "__IMAGE_TAG__ placeholder not found in $COMPOSE_FILE - refusing to proceed (a silent no-op substitution would push the literal string '__IMAGE_TAG__' as the image ref, which is a worse failure than this explicit error)"
  exit 1
fi

# In-place substitute. sed's `|` delimiter is safe here since TAG's
# allowed charset (checked above) cannot contain `|`.
sed -i "s|__IMAGE_TAG__|${TAG}|g" "$COMPOSE_FILE"

echo "[INFO] Rendered __IMAGE_TAG__ -> $TAG in $COMPOSE_FILE"
