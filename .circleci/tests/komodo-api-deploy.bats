#!/usr/bin/env bats
# Tests for .circleci/scripts/komodo-api-deploy.sh (direct-HTTPS version,
# VOIP-1342). See komodo-api-deploy-ssh-fallback.bats for the retained
# SSH-tunnel script's own coverage.

load 'test_helper'

setup() {
    setup_test_env
    install_fake_curl
    export CC_KOMODO_API_KEY="test-key"
    export CC_KOMODO_API_SECRET="test-secret"
    COMPOSE_FILE="$TEST_TEMP_DIR/docker-compose.yml"
    cat > "$COMPOSE_FILE" <<'EOF'
services:
  call-manager:
    image: voipbin/bin-call-manager:abc123
networks:
  default:
    name: install_default
    external: true
EOF
}

teardown() {
    teardown_test_env
}

run_deploy() {
    bash "$SCRIPTS_DIR/komodo-api-deploy.sh" "$@"
}

@test "usage: prints help and exits 0 with -h" {
    run run_deploy -h
    [ "$status" -eq 0 ]
    [[ "$output" == *"Usage:"* ]]
}

@test "usage: exits 1 with wrong arg count" {
    run run_deploy bin-call-manager
    [ "$status" -eq 1 ]
}

@test "validation: rejects a stack name with shell metacharacters" {
    run run_deploy 'bin-call-manager; rm -rf /' "$COMPOSE_FILE"
    [ "$status" -eq 1 ]
    [[ "$output" == *"stack-name must match"* ]]
}

@test "validation: rejects a nonexistent compose file" {
    run run_deploy bin-call-manager /nonexistent/path.yml
    [ "$status" -eq 1 ]
    [[ "$output" == *"does not exist"* ]]
}

@test "validation: requires CC_KOMODO_API_KEY" {
    unset CC_KOMODO_API_KEY
    run run_deploy bin-call-manager "$COMPOSE_FILE"
    [ "$status" -eq 1 ]
    [[ "$output" == *"CC_KOMODO_API_KEY is not set"* ]]
}

@test "validation: requires CC_KOMODO_API_SECRET" {
    unset CC_KOMODO_API_SECRET
    run run_deploy bin-call-manager "$COMPOSE_FILE"
    [ "$status" -eq 1 ]
    [[ "$output" == *"CC_KOMODO_API_SECRET is not set"* ]]
}

@test "validation: rejects a newline embedded in CC_KOMODO_API_KEY" {
    export CC_KOMODO_API_KEY=$'legit\nheader = "X-Evil: injected"'
    run run_deploy bin-call-manager "$COMPOSE_FILE"
    [ "$status" -eq 1 ]
    [[ "$output" == *"CC_KOMODO_API_KEY must not contain a newline"* ]]
    [ "$(curl_call_count)" -eq 0 ]
}

@test "validation: rejects a newline embedded in CC_KOMODO_API_SECRET" {
    export CC_KOMODO_API_SECRET=$'legit\nheader = "X-Evil: injected"'
    run run_deploy bin-call-manager "$COMPOSE_FILE"
    [ "$status" -eq 1 ]
    [[ "$output" == *"CC_KOMODO_API_SECRET must not contain a newline"* ]]
    [ "$(curl_call_count)" -eq 0 ]
}

@test "validation: rejects a double-quote in CC_KOMODO_API_KEY" {
    export CC_KOMODO_API_KEY='bad"key'
    run run_deploy bin-call-manager "$COMPOSE_FILE"
    [ "$status" -eq 1 ]
    [[ "$output" == *"must not contain a double-quote"* ]]
}

@test "validation: rejects a leading-zero CC_KOMODO_POLL_ATTEMPTS" {
    export CC_KOMODO_POLL_ATTEMPTS="030"
    run run_deploy bin-call-manager "$COMPOSE_FILE"
    [ "$status" -eq 1 ]
    [[ "$output" == *"CC_KOMODO_POLL_ATTEMPTS must be a positive integer"* ]]
}

@test "placeholder guard: refuses to deploy with an unfilled __PLACEHOLDER__ token" {
    cat > "$COMPOSE_FILE" <<'EOF'
services:
  call-manager:
    image: voipbin/bin-call-manager:__IMAGE_TAG__
networks:
  default:
    name: __NETWORK_NAME__
    external: true
EOF
    run run_deploy bin-call-manager "$COMPOSE_FILE"
    [ "$status" -eq 1 ]
    [[ "$output" == *"unfilled __PLACEHOLDER__ token"* ]]
    [[ "$output" == *"__IMAGE_TAG__"* ]]
    [[ "$output" == *"__NETWORK_NAME__"* ]]
    # Must fail before ever calling the API - no curl invocation logged.
    [ "$(curl_call_count)" -eq 0 ]
}

@test "ensure-exists: GetStack 'not found' (HTTP 500, Komodo's actual shape) triggers an idempotent CreateStack" {
    export CC_KOMODO_POLL_INTERVAL_S=1 CC_KOMODO_RUNNING_CHECK_INTERVAL_S=1
    queue_curl_response 500 '{"error":"Did not find any Stack matching bin-call-manager","trace":[]}'  # GetStack
    queue_curl_response 200 '{"name":"bin-call-manager"}'   # CreateStack
    queue_curl_response 200 '{}'                             # UpdateStack
    queue_curl_response 200 '{"_id":{"$oid":"update-123"}}'  # DeployStack
    queue_curl_response 200 '{"status":"Complete","success":true}'
    queue_running_samples 3
    run run_deploy bin-call-manager "$COMPOSE_FILE"
    [ "$status" -eq 0 ]
    [[ "$output" == *"does not exist yet - creating it"* ]]
    body="$(curl_call_body 2)"  # CreateStack is the 2nd call
    [[ "$body" == *'"CreateStack"'* ]]
    [[ "$body" == *'"bm-nyc-01"'* ]]
}

@test "ensure-exists: uses CC_KOMODO_SERVER_NAME when set, not the bm-nyc-01 default" {
    export CC_KOMODO_POLL_INTERVAL_S=1 CC_KOMODO_RUNNING_CHECK_INTERVAL_S=1
    export CC_KOMODO_SERVER_NAME=some-other-host
    queue_curl_response 500 '{"error":"Did not find any Stack matching bin-call-manager","trace":[]}'
    queue_curl_response 200 '{"name":"bin-call-manager"}'
    queue_curl_response 200 '{}'
    queue_curl_response 200 '{"_id":{"$oid":"update-123"}}'
    queue_curl_response 200 '{"status":"Complete","success":true}'
    queue_running_samples 3
    run run_deploy bin-call-manager "$COMPOSE_FILE"
    [ "$status" -eq 0 ]
    body="$(curl_call_body 2)"
    [[ "$body" == *'"some-other-host"'* ]]
}

@test "ensure-exists: if CreateStack itself fails, the script aborts rather than proceeding to UpdateStack/DeployStack" {
    queue_curl_response 500 '{"error":"Did not find any Stack matching bin-call-manager","trace":[]}'  # GetStack
    queue_curl_response 500 '{"error":"insufficient permissions"}'                                       # CreateStack fails
    run run_deploy bin-call-manager "$COMPOSE_FILE"
    [ "$status" -eq 1 ]
    [[ "$output" == *"does not exist yet - creating it"* ]]
    # Only GetStack + the failed CreateStack - never got to UpdateStack/DeployStack.
    [ "$(curl_call_count)" -eq 2 ]
}

@test "ensure-exists: a GetStack failure for any OTHER reason does not attempt to create a Stack" {
    queue_curl_response 500 '{"error":"internal server error, unrelated to stack existence"}'
    run run_deploy bin-call-manager "$COMPOSE_FILE"
    [ "$status" -eq 1 ]
    [[ "$output" == *"not attempting to create one"* ]]
    # Only the one GetStack call - no CreateStack attempted.
    [ "$(curl_call_count)" -eq 1 ]
}

@test "404 special-casing: points at the Caddy-route recovery step, not a generic message" {
    queue_curl_response 404 '{"error":"not found"}'
    run run_deploy bin-call-manager "$COMPOSE_FILE"
    [ "$status" -eq 1 ]
    [[ "$output" == *"Caddy-route-missing"* ]]
    [[ "$output" == *"setup-host.sh"* ]]
}

@test "502 special-casing: points at the docker network connect recovery step, not a generic message" {
    queue_curl_response 502 'Bad Gateway'
    run run_deploy bin-call-manager "$COMPOSE_FILE"
    [ "$status" -eq 1 ]
    [[ "$output" == *"network-attachment-dropped"* ]]
    [[ "$output" == *"docker network connect komodo_default voipbin-web-proxy"* ]]
}

# All queue_running_samples calls below queue N ListStackServices
# responses, each reporting a single service in state "running" - the gate
# (a) part-2 check (design doc §5) that runs after Komodo itself reports
# the compose update Complete+success.
queue_running_samples() {
    local n="$1"
    for ((i = 0; i < n; i++)); do
        queue_curl_response 200 '[{"service":"call-manager","container":{"state":"running"}}]'
    done
}

@test "happy path: GetStack -> UpdateStack -> DeployStack -> GetUpdate(Complete,success) -> 3x running exits 0" {
    export CC_KOMODO_POLL_INTERVAL_S=1 CC_KOMODO_RUNNING_CHECK_INTERVAL_S=1
    queue_curl_response 200 '{"id":"bin-call-manager"}'              # GetStack
    queue_curl_response 200 '{}'                                      # UpdateStack
    queue_curl_response 200 '{"_id":{"$oid":"update-123"}}'           # DeployStack
    queue_curl_response 200 '{"status":"Complete","success":true}'    # GetUpdate (poll 1)
    queue_running_samples 3                                           # ListStackServices x3
    run run_deploy bin-call-manager "$COMPOSE_FILE"
    [ "$status" -eq 0 ]
    [[ "$output" == *"DEPLOY_OK"* ]]
    [[ "$output" == *"update_id=update-123"* ]]
    [[ "$output" == *"3 consecutive running samples confirmed"* ]]
}

@test "running-gate: a non-running sample resets the consecutive count instead of failing immediately" {
    export CC_KOMODO_POLL_INTERVAL_S=1 CC_KOMODO_RUNNING_CHECK_INTERVAL_S=1
    queue_curl_response 200 '{"id":"bin-call-manager"}'
    queue_curl_response 200 '{}'
    queue_curl_response 200 '{"_id":{"$oid":"update-123"}}'
    queue_curl_response 200 '{"status":"Complete","success":true}'
    queue_running_samples 2
    queue_curl_response 200 '[{"service":"call-manager","container":{"state":"restarting"}}]'
    queue_running_samples 3
    run run_deploy bin-call-manager "$COMPOSE_FILE"
    [ "$status" -eq 0 ]
    [[ "$output" == *"DEPLOY_OK"* ]]
}

@test "running-gate: never reaching 3 consecutive running samples exits 1 (deploy 'succeeded' but stack unhealthy)" {
    export CC_KOMODO_POLL_INTERVAL_S=1 CC_KOMODO_RUNNING_CHECK_INTERVAL_S=1
    queue_curl_response 200 '{"id":"bin-call-manager"}'
    queue_curl_response 200 '{}'
    queue_curl_response 200 '{"_id":{"$oid":"update-123"}}'
    queue_curl_response 200 '{"status":"Complete","success":true}'
    for i in 1 2 3 4 5 6 7 8; do
        queue_curl_response 200 '[{"service":"call-manager","container":{"state":"restarting"}}]'
    done
    run run_deploy bin-call-manager "$COMPOSE_FILE"
    [ "$status" -eq 1 ]
    [[ "$output" == *"DEPLOY_FAILED"* ]]
    [[ "$output" == *"never reached 3 consecutive"* ]]
}

@test "happy path: sends secrets via -K header config, not curl argv" {
    export CC_KOMODO_POLL_INTERVAL_S=1 CC_KOMODO_RUNNING_CHECK_INTERVAL_S=1
    queue_curl_response 200 '{"id":"bin-call-manager"}'
    queue_curl_response 200 '{}'
    queue_curl_response 200 '{"_id":{"$oid":"update-123"}}'
    queue_curl_response 200 '{"status":"Complete","success":true}'
    queue_running_samples 3
    run run_deploy bin-call-manager "$COMPOSE_FILE"
    [ "$status" -eq 0 ]
    header_config="$(curl_call_header_config 1)"
    [[ "$header_config" == *"X-Api-Key: test-key"* ]]
    [[ "$header_config" == *"X-Api-Secret: test-secret"* ]]
    # Not present as an argv-visible -H header carrying the secret.
    run bash -c "grep -vE '^[[:space:]]*#' '$SCRIPTS_DIR/komodo-api-deploy.sh' | grep -E '\-H \"X-Api-(Key|Secret): \\\$'"
    [ "$status" -ne 0 ]
}

@test "UpdateStack request body carries the compose content, un-rendered placeholders excluded by prior guard" {
    export CC_KOMODO_POLL_INTERVAL_S=1 CC_KOMODO_RUNNING_CHECK_INTERVAL_S=1
    queue_curl_response 200 '{"id":"bin-call-manager"}'
    queue_curl_response 200 '{}'
    queue_curl_response 200 '{"_id":{"$oid":"update-123"}}'
    queue_curl_response 200 '{"status":"Complete","success":true}'
    queue_running_samples 3
    run run_deploy bin-call-manager "$COMPOSE_FILE"
    [ "$status" -eq 0 ]
    body="$(curl_call_body 2)"  # UpdateStack is the 2nd call
    [[ "$body" == *"voipbin/bin-call-manager:abc123"* ]]
    [[ "$body" == *'"UpdateStack"'* ]]
}

@test "deploy failure (Complete, success=false): exits 1 without printing raw Komodo log content" {
    export CC_KOMODO_POLL_INTERVAL_S=1
    queue_curl_response 200 '{"id":"bin-call-manager"}'
    queue_curl_response 200 '{}'
    queue_curl_response 200 '{"_id":{"$oid":"update-123"}}'
    queue_curl_response 200 '{"status":"Complete","success":false,"logs":[{"stderr":"amqp://realuser:realsecret@rabbitmq:5672"}]}'
    run run_deploy bin-call-manager "$COMPOSE_FILE"
    [ "$status" -eq 1 ]
    [[ "$output" == *"DEPLOY_FAILED"* ]]
    # The log-leakage guard: the secret-shaped content from Komodo's own
    # response must never appear in this script's own output.
    [[ "$output" != *"realsecret"* ]]
    [[ "$output" == *"Komodo UI/API directly"* ]]
}

@test "deploy timeout: exits 1 after exhausting CC_KOMODO_POLL_ATTEMPTS" {
    export CC_KOMODO_POLL_ATTEMPTS=2
    export CC_KOMODO_POLL_INTERVAL_S=1
    queue_curl_response 200 '{"id":"bin-call-manager"}'
    queue_curl_response 200 '{}'
    queue_curl_response 200 '{"_id":{"$oid":"update-123"}}'
    queue_curl_response 200 '{"status":"InProgress"}'
    queue_curl_response 200 '{"status":"InProgress"}'
    run run_deploy bin-call-manager "$COMPOSE_FILE"
    [ "$status" -eq 1 ]
    [[ "$output" == *"DEPLOY_TIMEOUT"* ]]
    [[ "$output" == *"may still land after this script gives up"* ]]
}
