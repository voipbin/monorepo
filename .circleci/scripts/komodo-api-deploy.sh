#!/bin/bash
# VoIPBin - deploy via Komodo API, direct HTTPS (kustomize-style: git is
# the source of truth, this script pushes the compose file's CURRENT
# content to Komodo and triggers a deploy).
#
# VOIP-1342 rewrite of the VOIP-1341 script: that version talked to Komodo
# over an SSH tunnel to bm-nyc-01 (Komodo core was loopback-only at the
# time). Komodo is now reachable directly over HTTPS
# (https://komodo.voipbin.net, via bm-nyc-01's own Caddy - see
# monorepo-etc/infra-komodo/README.md's "Public access" section and
# monorepo-etc's infra-secret-sync-komodo.sh, which already uses this same
# transport for the same endpoint), so this script no longer needs SSH at
# all. The SSH-tunnel version is kept (functionally unchanged, one
# security hardening fix - see its own header) as
# komodo-api-deploy-ssh-fallback.sh - the break-glass path for an incident
# where this HTTPS endpoint itself is unreachable (see this repo's
# docs/plans/2026-08-16-komodo-call-manager-cutover-design.md §2/§3).
#
# What this does:
#   1. GET (via /read GetStack) to confirm the target Stack exists.
#      Idempotent ensure-exists (대표님's explicit instruction, 2026-08-16,
#      reversing VOIP-1341's original "never creates a Stack" precondition
#      - manual per-service bootstrap doesn't scale to 32 services): if
#      Komodo reports the Stack doesn't exist, CreateStack it (empty
#      file_contents, poll_for_updates/auto_update/webhook_enabled all
#      false, server = $KOMODO_SERVER_NAME) before continuing. Any other
#      GetStack failure (a real API/auth/network problem) still aborts -
#      only the specific "not found" case triggers a create.
#   2. Guards against any unfilled `__PLACEHOLDER__`-shaped token in the
#      rendered compose content before sending it anywhere - catches not
#      just a missed image-tag substitution (render-image-tag.sh already
#      guards that one specifically) but any other placeholder a compose
#      fragment might still contain (e.g. a network name meant to be
#      filled in by hand once - see the cutover design doc §1).
#   3. POST /write   {"type":"UpdateStack", ...file_contents...}
#   4. POST /execute {"type":"DeployStack", ...}
#   5. POST /read    {"type":"GetUpdate", ...} - polled until terminal.
#   6. Exits 0 only if the Update's terminal status is Complete+success.
#
# What this does NOT do:
#   - Does NOT touch an EXISTING Stack's config beyond `file_contents` (and,
#     when a 3rd argument is given, `environment` - see Usage below) -
#     UpdateStack (step 3 below) is PATCH-style - the ensure-exists check
#     in step 1 only ever creates a Stack that doesn't exist yet; it never
#     modifies one that's already there.
#   - Does NOT enable Komodo's own polling/webhook auto-deploy for the
#     target stack - CI remains the only trigger (set at Stack bootstrap
#     time: poll_for_updates/auto_update/webhook_enabled all false).
#   - Does NOT print Komodo's raw deploy-failure log content to CI's own
#     output. Komodo's GetUpdate response, on a compose parse error, can
#     echo real interpolated secret values (DSN/AMQP credentials) - this
#     script's own request bodies stay safe (un-interpolated `[[VAR]]`
#     placeholders, never real values), but the *response* on failure is
#     post-interpolation. On failure this script prints only the update ID
#     and a pointer to check it via the Komodo UI/API directly (an
#     authenticated channel), not CI's broadly-readable job log.
#
# Usage:
#   CC_KOMODO_API_KEY=<key> CC_KOMODO_API_SECRET=<secret> \
#     ./.circleci/scripts/komodo-api-deploy.sh <stack-name> <local-compose-file> [local-environment-file]
#
#   <stack-name>          The Komodo Stack resource's name (or id). Must
#                         already exist - see Preconditions below. Must
#                         match ^[A-Za-z0-9._-]+$ (enforced below).
#   <local-compose-file>  Path to the docker-compose.yml IN THE CI CHECKOUT.
#                         This script reads the file locally and sends its
#                         CURRENT content as the Stack's new file_contents
#                         - this is what makes the model "kustomize-style":
#                         git is the only source of truth for deployed
#                         content, Komodo just executes it on request.
#   [local-environment-file]  Optional. Path to a plain KEY=VALUE env file
#                         IN THE CI CHECKOUT (e.g. komodo/environment.env).
#                         Its content is PATCHed into the Stack's own
#                         `environment` config field (distinct from
#                         file_contents' service-level `environment:`
#                         blocks) - Komodo writes this into the Stack's
#                         run-directory .env file before `docker compose up`
#                         runs, which Compose can then read via `${VAR}`
#                         substitution or (VOIP-1351) an
#                         environment-sourced `secrets:` block. Supports the
#                         same `[[VARIABLE]]` interpolation as
#                         file_contents. Omit this argument entirely for a
#                         service with no Stack-level environment need -
#                         this preserves the exact previous behavior/request
#                         shape for every existing call site.
#
# Environment:
#   CC_KOMODO_API_KEY      Komodo API key for the 'circleci' Service User.
#   CC_KOMODO_API_SECRET   Matching API secret. Required alongside the key.
#                          Sent to curl via `-K -` (a config file fed on
#                          curl's own stdin, `header = "..."` directives),
#                          NOT `-H` on curl's argv - keeps secrets out of
#                          this process's argv/`ps` visibility for its
#                          whole lifetime, same property ssh-deploy.sh and
#                          the prior SSH-tunnel version of this script
#                          already maintain for their own secrets.
#   CC_KOMODO_POLL_ATTEMPTS    Optional. How many times to poll Komodo's
#                              GetUpdate before giving up. Default: 30.
#   CC_KOMODO_POLL_INTERVAL_S  Optional. Seconds to sleep between polls.
#                              Default: 2 (so 30x2s = 60s total by
#                              default). Raise either for a stack whose
#                              `docker compose pull && up` genuinely takes
#                              longer than 60s - set explicitly per job,
#                              don't rely on this default, for any service
#                              with a large image (see the cutover design
#                              doc §4).
#   CC_KOMODO_RUNNING_CHECK_INTERVAL_S  Optional. Seconds between the up-to-8
#                              samples of the post-deploy 3-consecutive-
#                              running check (design doc §5 gate a).
#                              Default: 2.
#   CC_KOMODO_SERVER_NAME      Optional. The Komodo Server to create the
#                              Stack on if it doesn't exist yet (step 1
#                              above). Default: bm-nyc-01.
#
# Preconditions (caller's responsibility):
#   - The target Stack's underlying image has already been built and
#     pushed (this script does not build or push anything).
#   - The 'circleci' Komodo Service User has Write permission on the
#     target Stack if it already exists, or Admin (to create new
#     resources) if it doesn't yet - both already true for this account.
#   - The rendered compose file has no unfilled `__PLACEHOLDER__` tokens
#     left in it (checked in step 2 above, but the caller's render step -
#     e.g. render-image-tag.sh - should already have handled the ones it
#     owns).
set -euo pipefail

API_BASE="https://komodo.voipbin.net"

log_info()  { echo "[INFO] $*"; }
log_error() { echo "[ERROR] $*" >&2; }

usage() {
  sed -n '2,/^[^#]/{/^[^#]/!p}' "${BASH_SOURCE[0]}" | sed 's/^# \{0,1\}//'
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

if [[ $# -ne 2 && $# -ne 3 ]]; then
  usage
  exit 1
fi

STACK_NAME="$1"
LOCAL_COMPOSE_FILE="$2"
LOCAL_ENV_FILE="${3:-}"

if [[ ! "$STACK_NAME" =~ ^[A-Za-z0-9._-]+$ ]]; then
  log_error "stack-name must match ^[A-Za-z0-9._-]+\$ (got: $STACK_NAME)"
  exit 1
fi

if [[ ! -f "$LOCAL_COMPOSE_FILE" ]]; then
  log_error "local-compose-file does not exist: $LOCAL_COMPOSE_FILE"
  exit 1
fi

if [[ -n "$LOCAL_ENV_FILE" && ! -f "$LOCAL_ENV_FILE" ]]; then
  log_error "local-environment-file does not exist: $LOCAL_ENV_FILE"
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
# Both secrets are interpolated into the curl -K config file's
# `header = "..."` directives (see call_api below) via a plain printf, one
# directive per line. An embedded newline in either value would let it
# inject an entirely new config-file directive (not just corrupt one
# header's value) - curl config files support more than headers (e.g.
# proxy), so this is checked before the quote/backslash check below, not
# folded into it. (Dropped by mistake in this script's rewrite from the
# SSH-tunnel version - see komodo-api-deploy-ssh-fallback.sh, which still
# has this check - and restored here.)
if [[ "$CC_KOMODO_API_KEY" == *$'\n'* ]]; then
  log_error "CC_KOMODO_API_KEY must not contain a newline"
  exit 1
fi
if [[ "$CC_KOMODO_API_SECRET" == *$'\n'* ]]; then
  log_error "CC_KOMODO_API_SECRET must not contain a newline"
  exit 1
fi
# The header-config-file value never needs its own escaping logic as long
# as these can't contain a `"` or `\` - a Komodo API token has no
# legitimate reason to contain either, so reject up front rather than
# implement a bespoke escaper only exercised on input it shouldn't get.
if [[ "$CC_KOMODO_API_KEY" == *'"'* || "$CC_KOMODO_API_KEY" == *'\'* ]]; then
  log_error "CC_KOMODO_API_KEY must not contain a double-quote or backslash character"
  exit 1
fi
if [[ "$CC_KOMODO_API_SECRET" == *'"'* || "$CC_KOMODO_API_SECRET" == *'\'* ]]; then
  log_error "CC_KOMODO_API_SECRET must not contain a double-quote or backslash character"
  exit 1
fi

POLL_ATTEMPTS="${CC_KOMODO_POLL_ATTEMPTS:-30}"
POLL_INTERVAL_S="${CC_KOMODO_POLL_INTERVAL_S:-2}"
RUNNING_CHECK_INTERVAL_S="${CC_KOMODO_RUNNING_CHECK_INTERVAL_S:-2}"
# Target Server for a Stack this script ends up creating (see the
# ensure-exists block below) - Komodo's own CreateStack validation
# resolves a Server by name as well as by id ("in case it comes in as
# name", per its own source), so the plain host name is safe to pass
# as-is. Override per-job if a service ever deploys to a different host.
KOMODO_SERVER_NAME="${CC_KOMODO_SERVER_NAME:-bm-nyc-01}"
if [[ ! "$POLL_ATTEMPTS" =~ ^[1-9][0-9]*$ ]]; then
  log_error "CC_KOMODO_POLL_ATTEMPTS must be a positive integer with no leading zero (got: $POLL_ATTEMPTS)"
  exit 1
fi
if [[ ! "$POLL_INTERVAL_S" =~ ^[1-9][0-9]*$ ]]; then
  log_error "CC_KOMODO_POLL_INTERVAL_S must be a positive integer with no leading zero (got: $POLL_INTERVAL_S)"
  exit 1
fi
if [[ ! "$RUNNING_CHECK_INTERVAL_S" =~ ^[1-9][0-9]*$ ]]; then
  log_error "CC_KOMODO_RUNNING_CHECK_INTERVAL_S must be a positive integer with no leading zero (got: $RUNNING_CHECK_INTERVAL_S)"
  exit 1
fi

# Placeholder guard (design doc §2 item 2): catches any remaining
# __SOMETHING__-shaped token in the compose content that should have been
# filled in before this script ever ran - a missed image-tag substitution
# (render-image-tag.sh already checks that one specifically, but this is a
# second, independent line of defense) or a network-name placeholder that
# only gets filled in by hand, once, into the git-committed file.
if grep -Eq '__[A-Z_]+__' "$LOCAL_COMPOSE_FILE"; then
  log_error "compose file still contains an unfilled __PLACEHOLDER__ token - refusing to deploy:"
  grep -En '__[A-Z_]+__' "$LOCAL_COMPOSE_FILE" >&2
  exit 1
fi
if [[ -n "$LOCAL_ENV_FILE" ]] && grep -Eq '__[A-Z_]+__' "$LOCAL_ENV_FILE"; then
  log_error "environment file still contains an unfilled __PLACEHOLDER__ token - refusing to deploy:"
  grep -En '__[A-Z_]+__' "$LOCAL_ENV_FILE" >&2
  exit 1
fi

COMPOSE_CONTENT="$(cat "$LOCAL_COMPOSE_FILE")"
ENV_CONTENT=""
if [[ -n "$LOCAL_ENV_FILE" ]]; then
  ENV_CONTENT="$(cat "$LOCAL_ENV_FILE")"
fi

# Shared helper so all API calls get identical error handling: on a
# non-2xx or a curl-level (connection) failure, print the request type and
# the actual response body to stderr before propagating the failure.
#
# Deliberately does NOT use `curl -f` - see the prior SSH-tunnel version
# of this script for why (discards the response body on non-2xx, which
# would defeat this helper's whole point). Instead requests the HTTP
# status via `-w` and splits it off manually.
#
# 404 and 502 are special-cased (design doc §2 item 6, §"Current state"):
# both are Komodo/Caddy fragility modes this endpoint hits *routinely*
# from unrelated operations (a `setup-host.sh` rerun deletes the Caddy
# route -> 404; the main stack's own CircleCI recreating voipbin-web-proxy
# drops the `docker network connect` -> 502) - a generic "non-2xx, check
# the logs" message would leave an operator without the one piece of
# information that actually resolves the failure.
# LAST_ERROR_BODY is set on the generic (non-404/502) failure branch below
# so callers that need to distinguish *why* a call failed (e.g. GetStack's
# ensure-exists check just below) can inspect the actual response body,
# not just the fact that call_api returned non-zero.
LAST_ERROR_BODY=""
call_api() {
  local endpoint="$1" body="$2" resp http_code http_body header_config
  LAST_ERROR_BODY=""
  header_config="$(printf 'header = "X-Api-Key: %s"\nheader = "X-Api-Secret: %s"\n' \
    "$CC_KOMODO_API_KEY" "$CC_KOMODO_API_SECRET")"
  if ! resp="$(printf '%s' "$header_config" | curl -sS -K - -w '\n%{http_code}' --connect-timeout 10 --max-time 30 -X POST "$API_BASE/$endpoint" \
      -H "Content-Type: application/json" \
      -d "$body")"; then
    log_error "Komodo API call to /$endpoint failed to execute (network/connection error - see curl's error output above)."
    return 1
  fi
  http_code="${resp##*$'\n'}"
  http_body="${resp%$'\n'*}"
  case "$http_code" in
    404)
      log_error "Komodo API call to /$endpoint returned HTTP 404."
      log_error "This is the Caddy-route-missing failure mode: any 'setup-host.sh' rerun on bm-nyc-01 deletes the komodo.voipbin.net route."
      log_error "Recovery: re-apply the Caddy block per monorepo-etc/infra-komodo/README.md's 'bm-nyc-01: Caddy route' section."
      return 1
      ;;
    502)
      log_error "Komodo API call to /$endpoint returned HTTP 502."
      log_error "This is the network-attachment-dropped failure mode: voipbin-web-proxy being recreated (including routinely by the main stack's own CircleCI pipeline) undoes 'docker network connect komodo_default voipbin-web-proxy'."
      log_error "Recovery: on bm-nyc-01, run: docker network connect komodo_default voipbin-web-proxy"
      return 1
      ;;
  esac
  if [[ ! "$http_code" =~ ^2[0-9][0-9]$ ]]; then
    log_error "Komodo API call to /$endpoint returned HTTP $http_code."
    log_error "Response body: $http_body"
    LAST_ERROR_BODY="$http_body"
    return 1
  fi
  printf '%s' "$http_body"
}

# Ensure-exists (idempotent): GetStack, and if Komodo reports the Stack
# doesn't exist, CreateStack (대표님's explicit instruction, 2026-08-16 -
# reverses this script's original "never creates a Stack" precondition,
# since manual per-service bootstrap doesn't scale to 32 services). Note
# Komodo returns this as an HTTP 500 with a specific error message, NOT a
# 404 - confirmed empirically against the live instance, not assumed - so
# detection is by message match, not status code.
log_info "Checking Komodo Stack '$STACK_NAME' exists..."
get_stack_body="$(jq -n --arg stack "$STACK_NAME" '{type:"GetStack",params:{stack:$stack}}')"
if ! call_api read "$get_stack_body" >/dev/null; then
  if [[ "$LAST_ERROR_BODY" == *"Did not find any Stack matching"* ]]; then
    log_info "Stack '$STACK_NAME' does not exist yet - creating it (idempotent bootstrap, server=$KOMODO_SERVER_NAME)."
    create_body="$(jq -n --arg name "$STACK_NAME" --arg server "$KOMODO_SERVER_NAME" --arg env "$ENV_CONTENT" \
      '{type:"CreateStack",params:{name:$name,config:{server_id:$server,file_contents:"",environment:$env,poll_for_updates:false,auto_update:false,webhook_enabled:false}}}')"
    call_api write "$create_body" >/dev/null
    log_info "Stack '$STACK_NAME' created."
  else
    log_error "GetStack failed for a reason other than 'Stack not found' - not attempting to create one. See the error above."
    exit 1
  fi
fi

log_info "Deploying Stack '$STACK_NAME' from local file $LOCAL_COMPOSE_FILE (content pushed via Komodo API, direct HTTPS)..."

# environment is only included in the PATCH when a 3rd argument was given -
# an existing call site that never passes one gets the exact same request
# shape as before this option existed (see this script's "What this does
# NOT do" header note).
if [[ -n "$LOCAL_ENV_FILE" ]]; then
  update_body="$(jq -n --arg id "$STACK_NAME" --arg contents "$COMPOSE_CONTENT" --arg env "$ENV_CONTENT" \
    '{type:"UpdateStack",params:{id:$id,config:{file_contents:$contents,environment:$env}}}')"
else
  update_body="$(jq -n --arg id "$STACK_NAME" --arg contents "$COMPOSE_CONTENT" \
    '{type:"UpdateStack",params:{id:$id,config:{file_contents:$contents}}}')"
fi
call_api write "$update_body" >/dev/null

deploy_body="$(jq -n --arg stack "$STACK_NAME" \
  '{type:"DeployStack",params:{stack:$stack,services:[],stop_time:null}}')"
deploy_res="$(call_api execute "$deploy_body")"
update_id="$(printf '%s' "$deploy_res" | jq -r '._id."$oid"')"
if [[ -z "$update_id" || "$update_id" == "null" ]]; then
  log_error "DeployStack did not return an update id. Response: $deploy_res"
  exit 1
fi

# Gate (a) part 2 (design doc §5): Komodo's own Complete+success (checked
# below) only means `docker compose up` returned 0 - started, not
# necessarily staying up. This second, independent check polls
# ListStackServices (confirmed via Komodo's own API docs/source to return
# each service's live container state, e.g. .container.state == "running")
# for 3 CONSECUTIVE running samples ~6s apart, same numeric gate
# ssh-deploy.sh already uses for the same reason (a single sample could
# catch a crash-looping container mid-cycle and falsely report success).
# Requires every service in the Stack to be running, not just one specific
# name - keeps this generic across stacks with more than one service.
# NOT YET verified against the live Komodo instance (only against Komodo's
# published API docs/source) - design doc §5's own open item. If
# `.container.state` turns out not to be the real field/shape on
# bm-nyc-01's Komodo version, this poll will consistently report
# not-running and every deploy will fail this gate - confirm this against
# a real deploy before trusting it unattended in CI.
check_stack_running() {
  local services_body services_res states
  services_body="$(jq -n --arg stack "$STACK_NAME" '{type:"ListStackServices",params:{stack:$stack}}')"
  services_res="$(call_api read "$services_body")" || return 1
  states="$(printf '%s' "$services_res" | jq -r '.[].container.state')"
  [[ -n "$states" ]] && ! grep -qv '^running$' <<< "$states"
}

for _ in $(seq 1 "$POLL_ATTEMPTS"); do
  sleep "$POLL_INTERVAL_S"
  status_body="$(jq -n --arg id "$update_id" '{type:"GetUpdate",params:{id:$id}}')"
  status_res="$(call_api read "$status_body")"
  status="$(printf '%s' "$status_res" | jq -r '.status')"
  if [[ "$status" == "Complete" ]]; then
    success="$(printf '%s' "$status_res" | jq -r '.success')"
    if [[ "$success" == "true" ]]; then
      consecutive_running=0
      for _ in 1 2 3 4 5 6 7 8; do
        sleep "$RUNNING_CHECK_INTERVAL_S"
        if check_stack_running; then
          consecutive_running=$((consecutive_running + 1))
        else
          consecutive_running=0
        fi
        if [[ "$consecutive_running" -ge 3 ]]; then
          log_info "DEPLOY_OK stack=$STACK_NAME update_id=$update_id (3 consecutive running samples confirmed)"
          exit 0
        fi
      done
      log_error "DEPLOY_FAILED stack=$STACK_NAME update_id=$update_id (Komodo reported the compose update succeeded, but the stack's services never reached 3 consecutive 'running' samples afterward)"
      exit 1
    else
      # Deliberately does NOT print status_res's raw log content - see
      # this script's header ("What this does NOT do") for why: Komodo's
      # GetUpdate response is post-interpolation and could contain real
      # secret values on a compose parse error.
      log_error "DEPLOY_FAILED stack=$STACK_NAME update_id=$update_id"
      log_error "Check the failure details via the Komodo UI/API directly (an authenticated channel) - not reproduced here to avoid leaking post-interpolation secret values into this CI log."
      exit 1
    fi
  fi
done
log_error "DEPLOY_TIMEOUT stack=$STACK_NAME update_id=$update_id (still InProgress after $((POLL_ATTEMPTS * POLL_INTERVAL_S))s)"
log_error "The deploy may still land after this script gives up - check Komodo directly before assuming failure and retrying."
exit 1
