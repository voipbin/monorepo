#!/bin/bash
# VoIPBin - deploy one already-pushed image to bm-nyc-01 over SSH
#
# CI-internal tooling, NOT part of any product surface. Bumps the pin for
# ONE image in bm-nyc-01's LIVE (untracked, operator-owned) versions.lock/
# docker-compose.yml and reconciles the running container - never touches
# voipbin/voipbin's versions.lock.dist/docker-compose.yml.dist (those are a
# completely separate, deploy-unrelated update path - see
# voipbin/voipbin/install/CLAUDE.md's "versions.lock.dist vs versions.lock"
# section). No PR, no git commit anywhere - this only mutates files on the
# remote host.
#
# Preconditions (caller's responsibility):
#   - The image at <image-repo>:<circle-sha1> has already been built and
#     pushed to Docker Hub (this script does not build or push anything).
#   - bm-nyc-01 already has a voipbin/voipbin install/ checkout at
#     /opt/voipbin/install with the versions.lock.dist/docker-compose.yml.dist
#     split already applied (bm-nyc-01 was migrated through that split before
#     this script was written).
#
# Usage:
#   CC_BM_NYC_01_DEPLOY_SSH_KEY=<private-key> ./.circleci/scripts/ssh-deploy-bm.sh <image-repo> <compose-service-name>
#
#   <image-repo>            e.g. voipbin/bin-call-manager - must already have
#                           an entry in bm-nyc-01's live versions.lock (this
#                           script only bumps an existing pin, same guard as
#                           voipbin/voipbin's bump-image-digest.sh, which it
#                           calls on the remote side).
#   <compose-service-name>  e.g. bin-call-manager - the docker-compose
#                           service name to pull/recreate after the pin bump.
#
# Environment:
#   CC_BM_NYC_01_DEPLOY_SSH_KEY   Private key for the dedicated 'deploy'
#                                 account on bm-nyc-01 (required). NEVER
#                                 echoed; written to a 600 temp file that is
#                                 removed on exit via a trap, regardless of
#                                 how the script exits.
#   CIRCLE_SHA1                   CircleCI's built-in full-length commit SHA
#                                 (required) - this is both the docker tag
#                                 already pushed and the source-commit
#                                 recorded in versions.lock.
#
# Target host and its SSH host keys are fixed below (not resolved via a
# runtime `ssh-keyscan`, which would trust whatever answers on first
# connect - these were captured directly from bm-nyc-01 out of band).
#
# Safety: only ever touches ONE image's pin in the LIVE versions.lock/
# docker-compose.yml and restarts ONE compose service. Does not attempt any
# automatic rollback on failure - deploying a bad image is a human decision
# to fix, same philosophy as voipbin/voipbin's open-versions-lock-pr.sh. On
# failure this prints the exact command to manually redeploy the previous
# digest.

set -e

BM_HOST="104.243.38.39"
BM_USER="deploy"
BM_KNOWN_HOSTS_LINE_RSA="104.243.38.39 ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQCtenMCo++fOqJ2NZiPDGlsteCEHTY5kFR7TCBpZsC/LyYaQYVeZaW+HRrhs1KS3FQhfDFSjYs6htnXoK6U8t+qiWb4c7iExQEeFgagxozIAkjmRERy5wnlH4RJ76loeKIcC/cDmW+eyASkGinCeAyT3DbYK5BVo0VnpXazNzgHzW7cK8G9w86+8gSt/G7f2e4iHk4qeXd2zMtuSXCF5L2gO0h3cfAHXecRUKMnWLPsczXJv/HlLSM3Xm7IjOVFHIZksJO/iD0kw2RN+fSehofokuc03Qq7462eZqjgsF53p4pEmNnWGDsQmdbE/e18NVsHh82I+1APaeORj2za+GCiEOUY+74oeLr1Omg5KiGEK6GagvVe8Ca/zJ19e0T+VkiIAGjZgseBOON3UyEHzEyUusfTsmBYWLubv1Fag4SP280yd+MeydCqr4IJg8jbGtQ+KWixxKDzXnIfuIk0lsyMfojVKeRkxKyxxwdEzWgX+nhtBPgHD2WKx+heCh5ri4E="
BM_KNOWN_HOSTS_LINE_ED25519="104.243.38.39 ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIGU/p87VedpGVqLoASzbDGJJjoFtjmKHREQzA+9gaRzq"
REMOTE_INSTALL_DIR="/opt/voipbin/install"

log_info()  { echo "[INFO] $*"; }
log_error() { echo "[ERROR] $*" >&2; }

usage() {
    sed -n '2,52p' "${BASH_SOURCE[0]}" | sed 's/^# \{0,1\}//'
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
    usage
    exit 0
fi

if [[ $# -ne 2 ]]; then
    usage
    exit 1
fi

IMAGE_REPO="$1"
COMPOSE_SERVICE="$2"

if [[ ! "$IMAGE_REPO" =~ ^voipbin/[a-z0-9]([a-z0-9-]*[a-z0-9])?$ ]]; then
    log_error "image-repo must look like 'voipbin/<name>' (got: $IMAGE_REPO)"
    exit 1
fi

if [[ ! "$COMPOSE_SERVICE" =~ ^[a-z0-9]([a-z0-9-]*[a-z0-9])?$ ]]; then
    log_error "compose-service-name must be a plain compose service name (got: $COMPOSE_SERVICE)"
    exit 1
fi

if [[ -z "${CC_BM_NYC_01_DEPLOY_SSH_KEY:-}" ]]; then
    log_error "CC_BM_NYC_01_DEPLOY_SSH_KEY is not set"
    exit 1
fi

if [[ -z "${CIRCLE_SHA1:-}" ]]; then
    log_error "CIRCLE_SHA1 is not set"
    exit 1
fi

if ! [[ "$CIRCLE_SHA1" =~ ^[0-9a-f]{40}$ ]]; then
    log_error "CIRCLE_SHA1 must be a full 40-char git SHA (got: $CIRCLE_SHA1)"
    exit 1
fi

SSH_KEY_FILE="$(mktemp)"
KNOWN_HOSTS_FILE="$(mktemp)"
cleanup() {
    rm -f "$SSH_KEY_FILE" "$KNOWN_HOSTS_FILE"
}
trap cleanup EXIT

# Written from a variable, never logged: the key material itself never
# appears in this script's own output.
printf '%s\n' "$CC_BM_NYC_01_DEPLOY_SSH_KEY" > "$SSH_KEY_FILE"
chmod 600 "$SSH_KEY_FILE"

printf '%s\n%s\n' "$BM_KNOWN_HOSTS_LINE_RSA" "$BM_KNOWN_HOSTS_LINE_ED25519" > "$KNOWN_HOSTS_FILE"

SSH_OPTS=(
    -i "$SSH_KEY_FILE"
    -o "UserKnownHostsFile=$KNOWN_HOSTS_FILE"
    -o GlobalKnownHostsFile=/dev/null
    -o StrictHostKeyChecking=yes
    -o ConnectTimeout=10
    -o BatchMode=yes
)

log_info "Deploying $IMAGE_REPO:$CIRCLE_SHA1 -> bm-nyc-01 (compose service: $COMPOSE_SERVICE)"

# ---- Remote command, single SSH session: bump the pin, then pull+recreate
# the one container. LOCK_FILE/COMPOSE_FILE explicitly point at the LIVE
# (untracked) files, never the .dist templates - deploy stays fully
# decoupled from voipbin/voipbin's git history. bump-image-digest.sh already
# resolves the digest for a plain <repo>:<tag> ref (registry lookup), so no
# digest needs to be pre-resolved here. ----
REMOTE_CMD="set -e
cd '$REMOTE_INSTALL_DIR'
LOCK_FILE='$REMOTE_INSTALL_DIR/versions.lock' COMPOSE_FILE='$REMOTE_INSTALL_DIR/docker-compose.yml' \
    ./scripts/bump-image-digest.sh '$IMAGE_REPO' '$IMAGE_REPO:$CIRCLE_SHA1' '$CIRCLE_SHA1'
docker compose pull '$COMPOSE_SERVICE'
docker compose up -d '$COMPOSE_SERVICE'
consecutive_running=0
for i in \$(seq 1 8); do
    sleep 2
    state=\$(docker compose ps --format '{{.State}}' '$COMPOSE_SERVICE' 2>/dev/null || true)
    if [[ \"\$state\" == \"running\" ]]; then
        consecutive_running=\$((consecutive_running + 1))
    else
        consecutive_running=0
    fi
    # Require 3 consecutive 'running' samples (~6s apart) before declaring
    # success - a single sample could catch a crash-looping container
    # mid-cycle and falsely report success.
    if [[ \"\$consecutive_running\" -ge 3 ]]; then
        echo \"DEPLOY_OK state=\$state\"
        exit 0
    fi
done
echo \"DEPLOY_FAILED last_state=\$state\" >&2
docker compose logs --tail 30 '$COMPOSE_SERVICE' >&2 || true
exit 1"

if ssh "${SSH_OPTS[@]}" "${BM_USER}@${BM_HOST}" "$REMOTE_CMD"; then
    log_info "Deploy succeeded: $COMPOSE_SERVICE is running $IMAGE_REPO:$CIRCLE_SHA1"
    exit 0
else
    rc=$?
    log_error "Deploy failed (exit $rc). $COMPOSE_SERVICE on bm-nyc-01 may be running a bad image."
    log_error "This script does not auto-rollback - a human must decide the next step."
    log_error "To manually redeploy a known-good digest, re-run bump-image-digest.sh on bm-nyc-01"
    log_error "with that digest, e.g.:"
    log_error "  ssh ${BM_USER}@${BM_HOST} \"cd $REMOTE_INSTALL_DIR && LOCK_FILE=./versions.lock COMPOSE_FILE=./docker-compose.yml ./scripts/bump-image-digest.sh $IMAGE_REPO <known-good-ref> <known-good-commit> && docker compose up -d $COMPOSE_SERVICE\""
    exit "$rc"
fi
