#!/bin/bash
# VoIPBin - deploy via Komodo API (kustomize-style: git is the source of
# truth, this script pushes the compose file's CURRENT content to Komodo
# and triggers a deploy). Reusable across services; not yet wired into any
# individual service's CircleCI job (VOIP-1341 scope - see that ticket).
#
# Mirrors ssh-deploy.sh's SSH/security pattern (key handling, host key
# pinning, no runtime ssh-keyscan) - same target host, same 'deploy'
# account/CC_DEPLOY_SSH_KEY context variable. See that script's header for
# the full rationale on those choices; this header only documents what's
# different here.
#
# What this does:
#   1. SSHes into bm-nyc-01 (never exposes Komodo's API outside that
#      session - Komodo core stays bound to 127.0.0.1, no public IP/domain,
#      per the explicit decision this ticket documents).
#   2. From inside that SSH session, calls Komodo's REST API against
#      http://127.0.0.1:9120:
#        a. POST /write   {"type":"UpdateStack", ...file_contents...}
#        b. POST /execute {"type":"DeployStack", ...}
#        c. POST /read    {"type":"GetUpdate", ...} - polled until terminal
#   3. Exits 0 only if the Update's terminal status is Complete+success.
#
# What this does NOT do:
#   - Does NOT decompose the existing monolithic docker-compose.yml (32
#     services in one file) into per-service files - that's separate,
#     deferred work (see VOIP-1341's "스코프 제외" section). This script is
#     ready to be adopted per-service once that decomposition happens; it
#     is not wired into any service's deploy job yet.
#   - Does NOT expose Komodo's API publicly. Every call happens inside an
#     SSH session to bm-nyc-01, targeting 127.0.0.1:9120 - deliberate
#     decision (see VOIP-1341) given Komodo Periphery's host-root-
#     equivalent docker.sock access.
#   - Does NOT enable Komodo's own polling/webhook auto-deploy for the
#     target stack. CI remains the only trigger, matching the explicit,
#     deliberate-only-on-CI-action model ssh-deploy.sh already uses.
#   - Does NOT create the target Stack. It must already exist (Write
#     permission for the 'circleci' Komodo Service User granted on it) -
#     same "only bumps an existing entry" precondition ssh-deploy.sh has
#     for versions.lock.
#
# Usage:
#   CC_DEPLOY_SSH_KEY=<base64-private-key> \
#   CC_KOMODO_API_KEY=<key> CC_KOMODO_API_SECRET=<secret> \
#     ./.circleci/scripts/komodo-api-deploy.sh <stack-name> <local-compose-file>
#
#   <stack-name>          The Komodo Stack resource's name (or id). Must
#                         already exist - see Preconditions below. Must
#                         match ^[A-Za-z0-9._-]+$ (enforced below) - this
#                         keeps it safe to splice into the remote script
#                         template without any risk of breaking out of its
#                         quoting context.
#   <local-compose-file>  Path to the docker-compose.yml IN THE CI CHECKOUT
#                         (never touches bm-nyc-01's filesystem - no rsync,
#                         no scp, nothing written to disk on the server).
#                         This script reads the file locally, base64-
#                         encodes it, and embeds that content directly in
#                         the Komodo API request body (file_contents) sent
#                         over the SSH session. The CURRENT content at the
#                         checked-out commit becomes the Stack's new
#                         file_contents - this is what makes the model
#                         "kustomize-style": git is the only source of
#                         truth for deployed content, Komodo just executes
#                         it on request.
#
# Environment:
#   CC_DEPLOY_SSH_KEY      Same key ssh-deploy.sh uses (base64-encoded, ONE
#                          line). Required. Never echoed; written to a 600
#                          temp file removed on exit.
#   CC_KOMODO_API_KEY      Komodo API key for the 'circleci' Service User.
#   CC_KOMODO_API_SECRET   Matching API secret. Required alongside the key.
#                          Crosses the SSH boundary over stdin (read by a
#                          `read -r` in the very first two lines of the
#                          remote script), never as a command-line argument
#                          and never via SSH's SendEnv (bm-nyc-01's sshd
#                          AcceptEnv allowlist does not include these names
#                          - checked directly against the live sshd_config
#                          before choosing this approach; SendEnv fails
#                          silently, not loudly, when the server doesn't
#                          accept a variable, which would have produced a
#                          confusing empty-header failure downstream
#                          instead of a clear one here). Command arguments
#                          are visible to any local user via `ps` for the
#                          process's lifetime; stdin is not - this keeps
#                          the secret out of that exposure even though
#                          bm-nyc-01 currently only has root+deploy
#                          accounts.
#   CC_KOMODO_POLL_ATTEMPTS    Optional. How many times to poll Komodo's
#                              GetUpdate before giving up. Default: 30.
#   CC_KOMODO_POLL_INTERVAL_S  Optional. Seconds to sleep between polls.
#                              Default: 2 (so 30x2s = 60s total by
#                              default). Raise either for a stack whose
#                              `docker compose pull && up` genuinely takes
#                              longer than 60s.
#
# Preconditions (caller's responsibility):
#   - The target Stack's underlying image has already been built and
#     pushed (this script does not build or push anything - same
#     precondition as ssh-deploy.sh).
#   - The Stack already exists in Komodo, with the 'circleci' Service User
#     granted Write permission on it (Write implies Execute - verified
#     empirically against a live Komodo instance before this script was
#     written). Creating a Stack/granting permission is a manual, one-time
#     bootstrap step (Komodo UI or API), not something this script does.
#   - bm-nyc-01 has jq installed (used to build/parse the API JSON
#     payloads on the remote side).
#
# Implementation note: the remote script is captured via a QUOTED heredoc
# (<<'REMOTE_SCRIPT_EOF') so the local shell performs zero expansion on it
# - no $VAR substitution, no backslash processing, no command
# substitution. The two dynamic values (stack name, base64 compose
# content) are spliced in afterward via a plain `sed` replace of two
# unique placeholder tokens. This was deliberately chosen over interpolating
# variables directly into an unquoted heredoc (the first version of this
# script did that and had a real, caught-in-testing quoting bug from the
# nested backslash-escaping needed for jq's JSON filters inside an
# SSH-command-inside-a-heredoc-inside-a-bash-script) - the placeholder+sed
# approach has exactly one substitution mechanism to reason about instead
# of three nested ones.
set -euo pipefail

DEPLOY_HOST="104.243.38.39"
DEPLOY_USER="deploy"
KNOWN_HOSTS_LINE_RSA="104.243.38.39 ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQCtenMCo++fOqJ2NZiPDGlsteCEHTY5kFR7TCBpZsC/LyYaQYVeZaW+HRrhs1KS3FQhfDFSjYs6htnXoK6U8t+qiWb4c7iExQEeFgagxozIAkjmRERy5wnlH4RJ76loeKIcC/cDmW+eyASkGinCeAyT3DbYK5BVo0VnpXazNzgHzW7cK8G9w86+8gSt/G7f2e4iHk4qeXd2zMtuSXCF5L2gO0h3cfAHXecRUKMnWLPsczXJv/HlLSM3Xm7IjOVFHIZksJO/iD0kw2RN+fSehofokuc03Qq7462eZqjgsF53p4pEmNnWGDsQmdbE/e18NVsHh82I+1APaeORj2za+GCiEOUY+74oeLr1Omg5KiGEK6GagvVe8Ca/zJ19e0T+VkiIAGjZgseBOON3UyEHzEyUusfTsmBYWLubv1Fag4SP280yd+MeydCqr4IJg8jbGtQ+KWixxKDzXnIfuIk0lsyMfojVKeRkxKyxxwdEzWgX+nhtBPgHD2WKx+heCh5ri4E="
KNOWN_HOSTS_LINE_ED25519="104.243.38.39 ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIGU/p87VedpGVqLoASzbDGJJjoFtjmKHREQzA+9gaRzq"

log_info()  { echo "[INFO] $*"; }
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

STACK_NAME="$1"
LOCAL_COMPOSE_FILE="$2"

if [[ ! "$STACK_NAME" =~ ^[A-Za-z0-9._-]+$ ]]; then
  log_error "stack-name must match ^[A-Za-z0-9._-]+\$ (got: $STACK_NAME)"
  exit 1
fi

if [[ ! -f "$LOCAL_COMPOSE_FILE" ]]; then
  log_error "local-compose-file does not exist: $LOCAL_COMPOSE_FILE"
  exit 1
fi

if [[ -z "${CC_DEPLOY_SSH_KEY:-}" ]]; then
  log_error "CC_DEPLOY_SSH_KEY is not set"
  exit 1
fi
if [[ -z "${CC_KOMODO_API_KEY:-}" ]]; then
  log_error "CC_KOMODO_API_KEY is not set"
  exit 1
fi
if [[ -z "${CC_KOMODO_API_SECRET:-}" ]]; then
  log_error "CC_KOMODO_API_SECRET is not set"
  exit 1
fi

# The remote side reads these as two `read -r` lines off this ssh
# session's stdin, in order (see the ssh invocation at the bottom of this
# file). A literal newline embedded in either value would desync that
# two-line protocol - CC_KOMODO_API_KEY would be silently truncated at the
# embedded newline and CC_KOMODO_API_SECRET would pick up the remainder,
# producing a confusing auth failure instead of a clear error here.
if [[ "$CC_KOMODO_API_KEY" == *$'\n'* ]]; then
  log_error "CC_KOMODO_API_KEY must not contain a newline"
  exit 1
fi
if [[ "$CC_KOMODO_API_SECRET" == *$'\n'* ]]; then
  log_error "CC_KOMODO_API_SECRET must not contain a newline"
  exit 1
fi

# The remote side embeds these into a curl -K config file's
# `header = "X-Api-Key: ..."` directive (see call_api in the remote
# script template below) rather than curl argv, specifically to keep
# them out of `ps`/`/proc/<pid>/cmdline` visibility on bm-nyc-01, the
# same property already enforced for the SSH hop itself. curl config
# files support backslash-escaping inside a quoted directive value, but
# rather than replicate that escaping logic here (or on the remote side)
# just to handle a `"` or `\` that a Komodo API token has no legitimate
# reason to contain, reject it up front with a clear error - simpler and
# safer than a bespoke escaper that only gets exercised on the rare
# input it needs to handle correctly.
if [[ "$CC_KOMODO_API_KEY" == *'"'* || "$CC_KOMODO_API_KEY" == *'\'* ]]; then
  log_error "CC_KOMODO_API_KEY must not contain a double-quote or backslash character"
  exit 1
fi
if [[ "$CC_KOMODO_API_SECRET" == *'"'* || "$CC_KOMODO_API_SECRET" == *'\'* ]]; then
  log_error "CC_KOMODO_API_SECRET must not contain a double-quote or backslash character"
  exit 1
fi

# How long to wait for Komodo to finish applying the deploy before giving
# up (POLL_ATTEMPTS x POLL_INTERVAL_S seconds total, 60s by default).
# Override for stacks whose `docker compose pull && up` genuinely takes
# longer than that.
POLL_ATTEMPTS="${CC_KOMODO_POLL_ATTEMPTS:-30}"
POLL_INTERVAL_S="${CC_KOMODO_POLL_INTERVAL_S:-2}"
# No leading zero: the remote script does arithmetic on these
# ($((POLL_ATTEMPTS * POLL_INTERVAL_S)) in its timeout message) and bash
# arithmetic treats a leading-zero numeral as octal - "038" would be a
# runtime arithmetic error (8/9 aren't valid octal digits), not just a
# cosmetic issue. ^[1-9][0-9]*$ also rejects "0" itself, which wouldn't
# make sense operationally (zero attempts/zero-second interval).
if [[ ! "$POLL_ATTEMPTS" =~ ^[1-9][0-9]*$ ]]; then
  log_error "CC_KOMODO_POLL_ATTEMPTS must be a positive integer with no leading zero (got: $POLL_ATTEMPTS)"
  exit 1
fi
if [[ ! "$POLL_INTERVAL_S" =~ ^[1-9][0-9]*$ ]]; then
  log_error "CC_KOMODO_POLL_INTERVAL_S must be a positive integer with no leading zero (got: $POLL_INTERVAL_S)"
  exit 1
fi

SSH_KEY_FILE="$(mktemp)"
KNOWN_HOSTS_FILE="$(mktemp)"
cleanup() {
  rm -f "$SSH_KEY_FILE" "$KNOWN_HOSTS_FILE"
}
trap cleanup EXIT
chmod 600 "$SSH_KEY_FILE"

# See ssh-deploy.sh's identical block for why base64 (not raw multi-line
# key) and why the permissive character strip before decoding.
CLEANED_KEY="$(printf '%s' "$CC_DEPLOY_SSH_KEY" | tr -dc 'A-Za-z0-9+/=\n')"
if ! printf '%s' "$CLEANED_KEY" | base64 -d > "$SSH_KEY_FILE" 2>/dev/null; then
  log_error "CC_DEPLOY_SSH_KEY is not valid base64 (must be 'base64 -w0 <private-key-file>', not the raw key)"
  exit 1
fi
if ! grep -q "PRIVATE KEY" "$SSH_KEY_FILE"; then
  log_error "CC_DEPLOY_SSH_KEY decoded but does not look like a private key (no 'PRIVATE KEY' marker) - check the value registered in the CircleCI context"
  exit 1
fi

printf '%s\n%s\n' "$KNOWN_HOSTS_LINE_RSA" "$KNOWN_HOSTS_LINE_ED25519" > "$KNOWN_HOSTS_FILE"

SSH_OPTS=(
  -i "$SSH_KEY_FILE"
  -o "UserKnownHostsFile=$KNOWN_HOSTS_FILE"
  -o GlobalKnownHostsFile=/dev/null
  -o StrictHostKeyChecking=yes
  -o ConnectTimeout=10
  -o BatchMode=yes
)

# Base64-encode the compose content so it can be embedded as a single,
# shell-metacharacter-free token in the remote script template below -
# avoids any quoting/escaping hazard from the file's own content (which
# can contain quotes, `$`, backticks, etc.). The remote side decodes it
# and hands it to `jq --arg` for JSON-safe embedding into the API request
# body - jq handles the JSON string escaping, this script never
# hand-builds JSON via string concatenation.
COMPOSE_B64="$(base64 -w0 < "$LOCAL_COMPOSE_FILE")"

log_info "Deploying Stack '$STACK_NAME' from local file $LOCAL_COMPOSE_FILE (content pushed via Komodo API, nothing written to bm-nyc-01's disk)..."

# ---- Remote script template. Fully quoted heredoc - no expansion at
# capture time. @@STACK_NAME@@/@@COMPOSE_B64@@/@@POLL_ATTEMPTS@@/
# @@POLL_INTERVAL_S@@ are literal placeholder tokens, replaced below via a
# single chained `sed` before this is sent as the SSH command. The `@`
# delimiter is deliberate, not cosmetic: all four substituted values
# (STACK_NAME's ^[A-Za-z0-9._-]+$ charset, COMPOSE_B64's base64 alphabet,
# and the two digit-only POLL_* values) are structurally incapable of
# containing `@`, so none of them can ever equal - or get corrupted into
# resembling - one of these tokens partway through the chain. An earlier
# draft used `__NAME__`-style tokens built from the SAME charset
# STACK_NAME is allowed to contain; a caller passing
# `stack-name=__COMPOSE_B64__` (a legal value per the regex) would have
# had their STACK_NAME silently overwritten by a LATER s/// in the same
# chain re-matching text an EARLIER s/// had just inserted - caught in
# review, verified by reproducing the exact corruption locally, before
# this shipped. All API calls target 127.0.0.1:9120 - Komodo core is
# never reachable outside this SSH session. CC_KOMODO_API_KEY/SECRET are
# read from this script's own stdin (first two lines) - see the ssh
# invocation at the bottom of this file for what feeds them in. ----
REMOTE_SCRIPT_TEMPLATE=$(cat <<'REMOTE_SCRIPT_EOF'
set -euo pipefail
IFS= read -r CC_KOMODO_API_KEY
IFS= read -r CC_KOMODO_API_SECRET
API="http://127.0.0.1:9120"
STACK_NAME="@@STACK_NAME@@"
POLL_ATTEMPTS="@@POLL_ATTEMPTS@@"
POLL_INTERVAL_S="@@POLL_INTERVAL_S@@"
# Command substitution strips trailing newlines; a trailing 'X' sentinel
# (stripped right after) preserves any trailing newline(s) the source
# compose file actually had, so file_contents stays byte-for-byte
# identical to what's checked into git - the script's own stated goal.
compose_content="$(printf '%s' '@@COMPOSE_B64@@' | base64 -d; printf X)"
compose_content="${compose_content%X}"

# Shared helper so all three API calls (write/execute/read) get identical
# error handling: on a non-2xx or a curl-level (connection) failure, print
# the request type and the actual response body to stderr before
# propagating the failure - losing this on 2 of 3 calls (as an earlier
# draft did) leaves an operator with only a generic "deploy failed", no
# idea why.
#
# Deliberately does NOT use `curl -f` ("fail on HTTP error"): -f discards
# the response body entirely on a non-2xx response (that's its documented
# behavior - the body is only preserved with --fail-with-body, which needs
# curl >=7.76 and isn't guaranteed available on bm-nyc-01), which would
# silently defeat the whole point of this helper - caught empirically
# during review by running this exact heredoc body against a stand-in
# HTTP server and observing "Response: " print blank on both an HTTP 400
# and a refused connection. Instead: request the HTTP status code via
# `-w '\n%{http_code}'` appended after the body, split it off the
# captured output ourselves, and treat any non-2xx as failure while still
# having the real body in hand to print. `-sS` (not bare `-s`) keeps
# curl's own connection-level error text (refused/timeout/DNS) flowing to
# this script's stderr - visible in the CI job's ssh output - instead of
# being suppressed.
#
# The two Komodo secrets are passed to curl via `-K -` (a config file fed
# on curl's own stdin, `header = "..."` directives), NOT `-H` on curl's
# argv. `-H "X-Api-Secret: $CC_KOMODO_API_SECRET"` would put the secret
# in plaintext in this process's argv for its whole lifetime - visible to
# any other local account on bm-nyc-01 via `ps`/`/proc/<pid>/cmdline` -
# caught in review as undercutting the exact same `ps`-exposure guarantee
# this script already goes out of its way to provide for the SSH hop
# (see CC_KOMODO_API_KEY/SECRET in this file's header). The local-side
# checks rejecting `"`/`\` in either secret (see the main script body)
# exist so this config-file value never needs its own escaping logic.
call_api() {
  local endpoint="$1" body="$2" resp http_code http_body header_config
  header_config="$(printf 'header = "X-Api-Key: %s"\nheader = "X-Api-Secret: %s"\n' \
    "$CC_KOMODO_API_KEY" "$CC_KOMODO_API_SECRET")"
  # --connect-timeout/--max-time bound how long a hung/unresponsive
  # Komodo core (TCP connects fine but never replies) can stall this
  # script - without them, a hang here is bounded only by the CI job's
  # own overall timeout, not by anything under this script's control.
  if ! resp="$(printf '%s' "$header_config" | curl -sS -K - -w '\n%{http_code}' --connect-timeout 10 --max-time 30 -X POST "$API/$endpoint" \
      -H "Content-Type: application/json" \
      -d "$body")"; then
    echo "Komodo API call to /$endpoint failed to execute (network/connection error - see curl's error output above). Request body: $body" >&2
    return 1
  fi
  http_code="${resp##*$'\n'}"
  http_body="${resp%$'\n'*}"
  if [[ ! "$http_code" =~ ^2[0-9][0-9]$ ]]; then
    echo "Komodo API call to /$endpoint returned HTTP $http_code. Request body: $body" >&2
    echo "Response body: $http_body" >&2
    return 1
  fi
  printf '%s' "$http_body"
}

update_body="$(jq -n --arg id "$STACK_NAME" --arg contents "$compose_content" \
  '{type:"UpdateStack",params:{id:$id,config:{file_contents:$contents}}}')"
if ! call_api write "$update_body" >/dev/null; then
  echo "UpdateStack failed - Stack '$STACK_NAME' may not exist yet (this script does not create Stacks, see its header)." >&2
  exit 1
fi

deploy_body="$(jq -n --arg stack "$STACK_NAME" \
  '{type:"DeployStack",params:{stack:$stack,services:[],stop_time:null}}')"
deploy_res="$(call_api execute "$deploy_body")" || exit 1
update_id="$(printf '%s' "$deploy_res" | jq -r '._id."$oid"')"
if [[ -z "$update_id" || "$update_id" == "null" ]]; then
  echo "DeployStack did not return an update id. Response: $deploy_res" >&2
  exit 1
fi

for i in $(seq 1 "$POLL_ATTEMPTS"); do
  sleep "$POLL_INTERVAL_S"
  status_body="$(jq -n --arg id "$update_id" '{type:"GetUpdate",params:{id:$id}}')"
  status_res="$(call_api read "$status_body")" || exit 1
  status="$(printf '%s' "$status_res" | jq -r '.status')"
  if [[ "$status" == "Complete" ]]; then
    success="$(printf '%s' "$status_res" | jq -r '.success')"
    if [[ "$success" == "true" ]]; then
      echo "DEPLOY_OK stack=$STACK_NAME update_id=$update_id"
      exit 0
    else
      echo "DEPLOY_FAILED stack=$STACK_NAME update_id=$update_id" >&2
      printf '%s' "$status_res" | jq -r '.logs[]?.stderr // empty' >&2
      exit 1
    fi
  fi
done
echo "DEPLOY_TIMEOUT stack=$STACK_NAME update_id=$update_id (still InProgress after $((POLL_ATTEMPTS * POLL_INTERVAL_S))s)" >&2
exit 1
REMOTE_SCRIPT_EOF
)

# Delimiter is '|', not '/': COMPOSE_B64 is base64 (alphabet A-Za-z0-9+/=)
# and CAN legitimately contain '/', which would terminate a '/'-delimited
# sed s/// pattern early and corrupt the substitution. '|' and '&' are not
# in the base64 alphabet, so this is safe without further escaping (sed's
# replacement text also treats '&' as "whole match" - irrelevant here
# since neither substituted value contains '&').
REMOTE_CMD="$(printf '%s' "$REMOTE_SCRIPT_TEMPLATE" | sed "s|@@STACK_NAME@@|${STACK_NAME}|g; s|@@COMPOSE_B64@@|${COMPOSE_B64}|g; s|@@POLL_ATTEMPTS@@|${POLL_ATTEMPTS}|g; s|@@POLL_INTERVAL_S@@|${POLL_INTERVAL_S}|g")"

if printf '%s\n%s\n' "$CC_KOMODO_API_KEY" "$CC_KOMODO_API_SECRET" | \
    ssh "${SSH_OPTS[@]}" "${DEPLOY_USER}@${DEPLOY_HOST}" "$REMOTE_CMD"; then
  log_info "Deploy succeeded: $STACK_NAME is up on bm-nyc-01 (via Komodo)"
  exit 0
else
  rc=$?
  log_error "Deploy failed (exit $rc). $STACK_NAME on bm-nyc-01 may be running a bad image."
  log_error "This script does not auto-rollback - a human must decide the next step:"
  log_error "  ssh ${DEPLOY_USER}@${DEPLOY_HOST} \"curl -s http://127.0.0.1:9120/read -H 'X-Api-Key: <key>' -H 'X-Api-Secret: <secret>' -d '{\\\"type\\\":\\\"GetStack\\\",\\\"params\\\":{\\\"stack\\\":\\\"$STACK_NAME\\\"}}'\""
  exit "$rc"
fi
