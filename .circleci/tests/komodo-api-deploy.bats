#!/usr/bin/env bats
# Tests for .circleci/scripts/komodo-api-deploy.sh

load 'test_helper'

setup() {
    setup_test_env
    COMPOSE_FIXTURE="$TEST_TEMP_DIR/docker-compose.yml"
    cat > "$COMPOSE_FIXTURE" <<'EOF'
services:
  test:
    image: nginx:latest
EOF
}

teardown() {
    teardown_test_env
}

run_script() {
    bash "$SCRIPTS_DIR/komodo-api-deploy.sh" "$@"
}

# A tiny Komodo API stand-in used only by the "remote body execution"
# tests below - unlike every other test in this file (which stubs `ssh`
# and only inspects the constructed command string), these actually
# extract and EXECUTE the remote heredoc body against a real local HTTP
# server. This closes a real gap: a bug where `curl -f` silently
# discarded the HTTP response body on failure "looked right" in the
# constructed command string (the code to print the body was right
# there) but was wrong at actual execution time - only caught by running
# it. $1 selects a canned response scenario.
start_mock_komodo() {
    local scenario="$1"
    MOCK_KOMODO_PORT=19977
    MOCK_KOMODO_SCRIPT="$TEST_TEMP_DIR/mock_komodo.py"
    MOCK_KOMODO_LOG="$TEST_TEMP_DIR/mock_komodo.log"
    case "$scenario" in
    http_error)
        cat > "$MOCK_KOMODO_SCRIPT" <<'PYEOF'
import http.server, json, sys
port = int(sys.argv[1])
class H(http.server.BaseHTTPRequestHandler):
    def do_POST(self):
        n = int(self.headers.get('Content-Length', 0)); self.rfile.read(n)
        self.send_response(400)
        self.send_header('Content-Type', 'application/json')
        self.end_headers()
        self.wfile.write(json.dumps({"error": "stack not found: teststack"}).encode())
    def log_message(self, *a): pass
http.server.HTTPServer(('127.0.0.1', port), H).serve_forever()
PYEOF
        ;;
    deploy_success)
        cat > "$MOCK_KOMODO_SCRIPT" <<'PYEOF'
import http.server, json, sys
port = int(sys.argv[1])
class H(http.server.BaseHTTPRequestHandler):
    def do_POST(self):
        n = int(self.headers.get('Content-Length', 0))
        body = json.loads(self.rfile.read(n))
        self.send_response(200); self.send_header('Content-Type', 'application/json'); self.end_headers()
        t = body.get('type')
        if t == 'UpdateStack':
            resp = {"ok": True}
        elif t == 'DeployStack':
            resp = {"_id": {"$oid": "abc123"}}
        else:
            resp = {"status": "Complete", "success": True, "logs": []}
        self.wfile.write(json.dumps(resp).encode())
    def log_message(self, *a): pass
http.server.HTTPServer(('127.0.0.1', port), H).serve_forever()
PYEOF
        ;;
    deploy_failure)
        cat > "$MOCK_KOMODO_SCRIPT" <<'PYEOF'
import http.server, json, sys
port = int(sys.argv[1])
class H(http.server.BaseHTTPRequestHandler):
    def do_POST(self):
        n = int(self.headers.get('Content-Length', 0))
        body = json.loads(self.rfile.read(n))
        self.send_response(200); self.send_header('Content-Type', 'application/json'); self.end_headers()
        t = body.get('type')
        if t == 'UpdateStack':
            resp = {"ok": True}
        elif t == 'DeployStack':
            resp = {"_id": {"$oid": "def456"}}
        else:
            resp = {"status": "Complete", "success": False, "logs": [{"stderr": "Error: image pull failed"}]}
        self.wfile.write(json.dumps(resp).encode())
    def log_message(self, *a): pass
http.server.HTTPServer(('127.0.0.1', port), H).serve_forever()
PYEOF
        ;;
    esac
    python3 "$MOCK_KOMODO_SCRIPT" "$MOCK_KOMODO_PORT" > "$MOCK_KOMODO_LOG" 2>&1 &
    MOCK_KOMODO_PID=$!
    # Poll until the server actually accepts connections instead of a
    # fixed sleep - avoids flakiness under load.
    for _ in $(seq 1 50); do
        if curl -s -o /dev/null "http://127.0.0.1:$MOCK_KOMODO_PORT/" 2>/dev/null; then
            break
        fi
        sleep 0.1
    done
}

stop_mock_komodo() {
    [[ -n "${MOCK_KOMODO_PID:-}" ]] && kill "$MOCK_KOMODO_PID" 2>/dev/null
}

# Extracts the remote heredoc body from the real script, points its API
# base at our mock server's port instead of the real 127.0.0.1:9120, and
# substitutes placeholder tokens the same way the script's own sed does.
run_remote_body_against_mock() {
    local stack_name="$1" compose_b64="$2"
    extract_heredoc_body "$SCRIPTS_DIR/komodo-api-deploy.sh" "REMOTE_SCRIPT_EOF" \
        | sed "s|127.0.0.1:9120|127.0.0.1:$MOCK_KOMODO_PORT|" \
        | sed "s|@@STACK_NAME@@|${stack_name}|; s|@@COMPOSE_B64@@|${compose_b64}|; s|@@POLL_ATTEMPTS@@|3|; s|@@POLL_INTERVAL_S@@|1|" \
        > "$TEST_TEMP_DIR/extracted_remote.sh"
    printf 'fake-api-key\nfake-api-secret\n' | bash "$TEST_TEMP_DIR/extracted_remote.sh"
}

@test "[remote body] HTTP error response body is actually captured and printed, not discarded by curl -f" {
    if ! command -v python3 >/dev/null 2>&1; then skip "python3 not available"; fi
    start_mock_komodo http_error
    run run_remote_body_against_mock "teststack" "c2VydmljZXM6CiAgdGVzdDoKICAgIGltYWdlOiBuZ2lueAo="
    stop_mock_komodo

    [[ "$status" -ne 0 ]]
    [[ "$output" == *"returned HTTP 400"* ]]
    [[ "$output" == *"stack not found: teststack"* ]]
}

@test "[remote body] full write -> execute -> read poll loop succeeds against a real HTTP server" {
    if ! command -v python3 >/dev/null 2>&1; then skip "python3 not available"; fi
    start_mock_komodo deploy_success
    run run_remote_body_against_mock "teststack" "c2VydmljZXM6CiAgdGVzdDoKICAgIGltYWdlOiBuZ2lueAo="
    stop_mock_komodo

    [[ "$status" -eq 0 ]]
    [[ "$output" == *"DEPLOY_OK stack=teststack update_id=abc123"* ]]
}

@test "[remote body] a deploy that completes with success=false reports failure and the remote logs" {
    if ! command -v python3 >/dev/null 2>&1; then skip "python3 not available"; fi
    start_mock_komodo deploy_failure
    run run_remote_body_against_mock "teststack" "c2VydmljZXM6CiAgdGVzdDoKICAgIGltYWdlOiBuZ2lueAo="
    stop_mock_komodo

    [[ "$status" -ne 0 ]]
    [[ "$output" == *"DEPLOY_FAILED stack=teststack update_id=def456"* ]]
    [[ "$output" == *"Error: image pull failed"* ]]
}

FAKE_KEY_CONTENT="-----BEGIN OPENSSH PRIVATE KEY-----
ZmFrZS1rZXktbWF0ZXJpYWwtZm9yLXRlc3Rpbmctb25seQ==
-----END OPENSSH PRIVATE KEY-----"

valid_env() {
    export CC_DEPLOY_SSH_KEY="$(printf '%s' "$FAKE_KEY_CONTENT" | base64 -w0)"
    export CC_KOMODO_API_KEY="fake-komodo-api-key"
    export CC_KOMODO_API_SECRET="fake-komodo-api-secret"
}

@test "-h/--help prints usage and exits 0 without requiring any env" {
    run run_script --help
    [[ "$status" -eq 0 ]]
    [[ "$output" == *"deploy via Komodo API"* ]]
}

@test "-h prints usage and exits 0 without requiring any env" {
    run run_script -h
    [[ "$status" -eq 0 ]]
    [[ "$output" == *"deploy via Komodo API"* ]]
}

@test "refuses with wrong argument count (too few)" {
    run run_script "voipbin-test-stack"
    [[ "$status" -ne 0 ]]
    [[ "$output" == *"Usage:"* || "$output" == *"deploy via Komodo API"* ]]
}

@test "refuses with wrong argument count (too many)" {
    valid_env
    run run_script "voipbin-test-stack" "$COMPOSE_FIXTURE" "unexpected-extra-arg"
    [[ "$status" -ne 0 ]]
    [[ "$output" == *"Usage:"* || "$output" == *"deploy via Komodo API"* ]]
}

@test "refuses a stack-name with unexpected characters" {
    valid_env
    run run_script "test-stack; rm -rf /" "$COMPOSE_FIXTURE"
    [[ "$status" -ne 0 ]]
    [[ "$output" == *"stack-name must be"* || "$output" == *"stack-name must match"* ]]
}

@test "refuses when local-compose-file does not exist" {
    valid_env
    run run_script "voipbin-test-stack" "$TEST_TEMP_DIR/does-not-exist.yml"
    [[ "$status" -ne 0 ]]
    [[ "$output" == *"local-compose-file does not exist"* ]]
}

@test "refuses to run when CC_DEPLOY_SSH_KEY is not set" {
    export CC_KOMODO_API_KEY="k"
    export CC_KOMODO_API_SECRET="s"
    unset CC_DEPLOY_SSH_KEY
    run run_script "voipbin-test-stack" "$COMPOSE_FIXTURE"
    [[ "$status" -ne 0 ]]
    [[ "$output" == *"CC_DEPLOY_SSH_KEY is not set"* ]]
}

@test "refuses to run when CC_KOMODO_API_KEY is not set" {
    valid_env
    unset CC_KOMODO_API_KEY
    run run_script "voipbin-test-stack" "$COMPOSE_FIXTURE"
    [[ "$status" -ne 0 ]]
    [[ "$output" == *"CC_KOMODO_API_KEY is not set"* ]]
}

@test "refuses to run when CC_KOMODO_API_SECRET is not set" {
    valid_env
    unset CC_KOMODO_API_SECRET
    run run_script "voipbin-test-stack" "$COMPOSE_FIXTURE"
    [[ "$status" -ne 0 ]]
    [[ "$output" == *"CC_KOMODO_API_SECRET is not set"* ]]
}

@test "refuses when CC_DEPLOY_SSH_KEY is not valid base64" {
    export CC_DEPLOY_SSH_KEY="not-valid-base64!!!"
    export CC_KOMODO_API_KEY="k"
    export CC_KOMODO_API_SECRET="s"
    run run_script "voipbin-test-stack" "$COMPOSE_FIXTURE"
    [[ "$status" -ne 0 ]]
    [[ "$output" == *"not valid base64"* ]]
}

@test "refuses when CC_DEPLOY_SSH_KEY decodes to something without a PRIVATE KEY marker" {
    export CC_DEPLOY_SSH_KEY="$(printf 'just some unrelated text' | base64 -w0)"
    export CC_KOMODO_API_KEY="k"
    export CC_KOMODO_API_SECRET="s"
    run run_script "voipbin-test-stack" "$COMPOSE_FIXTURE"
    [[ "$status" -ne 0 ]]
    [[ "$output" == *"does not look like a private key"* ]]
}

@test "happy path: succeeds and reports success when ssh exits 0" {
    valid_env
    install_fake_ssh 0
    run run_script "voipbin-test-stack" "$COMPOSE_FIXTURE"

    [[ "$status" -eq 0 ]]
    [[ "$output" == *"Deploy succeeded"* ]]
    [[ "$output" == *"voipbin-test-stack"* ]]
    [[ "$(ssh_call_count)" -eq 1 ]]
}

@test "ssh is invoked against the correct user@host with host-key pinning (not runtime ssh-keyscan)" {
    valid_env
    install_fake_ssh 0
    run run_script "voipbin-test-stack" "$COMPOSE_FIXTURE"

    local args
    args="$(ssh_call_args 1)"
    [[ "$args" == *"deploy@104.243.38.39"* ]]
    [[ "$args" == *"StrictHostKeyChecking=yes"* ]]
    [[ "$args" == *"UserKnownHostsFile="* ]]
    [[ "$output" != *"ssh-keyscan"* ]]
}

@test "the remote command targets Komodo's loopback-only API, never a public address" {
    valid_env
    install_fake_ssh 0
    run run_script "voipbin-test-stack" "$COMPOSE_FIXTURE"

    local remote_cmd
    remote_cmd="$(ssh_call_remote_command 1)"
    [[ "$remote_cmd" == *"http://127.0.0.1:9120"* ]]
    # Never anything other than loopback for the API base.
    [[ "$remote_cmd" != *"0.0.0.0:9120"* ]]
}

@test "the remote command's decoded compose content matches the local fixture exactly" {
    valid_env
    install_fake_ssh 0
    run run_script "voipbin-test-stack" "$COMPOSE_FIXTURE"

    local remote_cmd embedded_b64 decoded expected
    remote_cmd="$(ssh_call_remote_command 1)"
    embedded_b64="$(printf '%s' "$remote_cmd" | grep -o "printf '%s' '[^']*' | base64 -d" | sed "s/^printf '%s' '//; s/' | base64 -d$//")"
    [[ -n "$embedded_b64" ]]
    decoded="$(printf '%s' "$embedded_b64" | base64 -d)"
    expected="$(cat "$COMPOSE_FIXTURE")"
    [[ "$decoded" == "$expected" ]]
}

@test "a compose file whose base64 encoding contains '/' still substitutes correctly (sed delimiter regression)" {
    # Three 0xFF bytes base64-encode to '////' - guarantees '/' characters
    # land in the embedded payload, which would corrupt a '/'-delimited
    # sed substitution. This is the exact bug caught and fixed before
    # shipping (see the script's own comment on the sed delimiter choice).
    valid_env
    printf '\xff\xff\xffservices:\n  test:\n    image: nginx\n' > "$COMPOSE_FIXTURE"
    install_fake_ssh 0
    run run_script "voipbin-test-stack" "$COMPOSE_FIXTURE"

    [[ "$status" -eq 0 ]]
    local remote_cmd
    remote_cmd="$(ssh_call_remote_command 1)"
    # No leftover placeholder tokens - if the sed substitution had been
    # corrupted by an embedded '/', a literal placeholder or a truncated
    # command would remain instead of a clean substitution.
    [[ "$remote_cmd" != *"@@STACK_NAME@@"* ]]
    [[ "$remote_cmd" != *"@@COMPOSE_B64@@"* ]]
    [[ "$remote_cmd" == *"STACK_NAME=\"voipbin-test-stack\""* ]]
}

@test "a stack-name equal to another placeholder token's name does not corrupt the chained sed substitution" {
    # Regression test for a real bug caught in security review: the four
    # placeholders are substituted via one chained sed (multiple s///
    # separated by ';'), which re-scans text earlier substitutions just
    # inserted. With the OLD __NAME__-style tokens (same charset
    # STACK_NAME is allowed to contain), a stack-name equal to e.g.
    # "COMPOSE_B64" could get re-matched and corrupted by a later s///
    # in the chain. The @@NAME@@ tokens use '@', which is outside
    # STACK_NAME's ^[A-Za-z0-9._-]+$ charset, structurally preventing
    # this - this test proves a stack-name that WOULD have collided
    # under the old scheme now substitutes cleanly.
    valid_env
    install_fake_ssh 0
    run run_script "COMPOSE_B64" "$COMPOSE_FIXTURE"

    [[ "$status" -eq 0 ]]
    local remote_cmd
    remote_cmd="$(ssh_call_remote_command 1)"
    [[ "$remote_cmd" == *'STACK_NAME="COMPOSE_B64"'* ]]
    # The compose content placeholder must still have been replaced by
    # the actual base64 payload, not corrupted/overwritten by the
    # stack-name substitution that ran first in the chain.
    [[ "$remote_cmd" != *"@@COMPOSE_B64@@"* ]]
}

@test "Komodo API key/secret are never present in the extracted remote script's curl invocation (no argv leak via ps)" {
    # Regression test for a real bug caught in security review: curl -H
    # puts header VALUES directly in this process's argv, visible to any
    # other local account on bm-nyc-01 via ps/proc for the call's
    # duration - undercutting the same ps-exposure guarantee this script
    # already provides for the SSH hop. Verify the source never
    # constructs a curl invocation with -H "X-Api-...: <value>" as actual
    # code (excluding comment lines, which legitimately reference this
    # exact pattern by name when explaining why it was replaced).
    run bash -c "grep -vE '^[[:space:]]*#' '$SCRIPTS_DIR/komodo-api-deploy.sh' | grep -E '-H \"X-Api-(Key|Secret): \\\$'"
    [[ "$status" -ne 0 ]]
    # And confirm the actual replacement mechanism (curl -K -, config
    # fed on curl's own stdin) is present.
    run grep -F 'curl -sS -K -' "$SCRIPTS_DIR/komodo-api-deploy.sh"
    [[ "$status" -eq 0 ]]
}

@test "the remote command's decoded compose content preserves a trailing newline byte-for-byte" {
    # Command substitution strips trailing newlines; the script guards
    # against this with a trailing sentinel char that gets stripped back
    # off (see script.sh's comment on `compose_content`). Verify the
    # decoded content on the remote side keeps the source file's trailing
    # newline instead of silently losing it.
    valid_env
    printf 'services:\n  test:\n    image: nginx\n\n\n' > "$COMPOSE_FIXTURE"
    install_fake_ssh 0
    run run_script "voipbin-test-stack" "$COMPOSE_FIXTURE"
    [[ "$status" -eq 0 ]]

    local remote_cmd embedded_b64
    remote_cmd="$(ssh_call_remote_command 1)"
    embedded_b64="$(printf '%s' "$remote_cmd" | grep -o "printf '%s' '[^']*' | base64 -d" | sed "s/^printf '%s' '//; s/' | base64 -d$//")"
    # Decode straight to a file (not through a $() string, which would
    # itself strip trailing newlines and defeat the point of this test) -
    # byte-for-byte diff against the source fixture is the real assertion.
    printf '%s' "$embedded_b64" | base64 -d > "$TEST_TEMP_DIR/decoded_output.yml"
    diff "$COMPOSE_FIXTURE" "$TEST_TEMP_DIR/decoded_output.yml"
}

@test "CC_KOMODO_POLL_ATTEMPTS and CC_KOMODO_POLL_INTERVAL_S override the default poll loop" {
    valid_env
    export CC_KOMODO_POLL_ATTEMPTS=5
    export CC_KOMODO_POLL_INTERVAL_S=1
    install_fake_ssh 0
    run run_script "voipbin-test-stack" "$COMPOSE_FIXTURE"

    [[ "$status" -eq 0 ]]
    local remote_cmd
    remote_cmd="$(ssh_call_remote_command 1)"
    [[ "$remote_cmd" == *'POLL_ATTEMPTS="5"'* ]]
    [[ "$remote_cmd" == *'POLL_INTERVAL_S="1"'* ]]
}

@test "refuses a non-numeric CC_KOMODO_POLL_ATTEMPTS" {
    valid_env
    export CC_KOMODO_POLL_ATTEMPTS="thirty"
    run run_script "voipbin-test-stack" "$COMPOSE_FIXTURE"
    [[ "$status" -ne 0 ]]
    [[ "$output" == *"CC_KOMODO_POLL_ATTEMPTS must be a positive integer"* ]]
}

@test "refuses a CC_KOMODO_POLL_ATTEMPTS with a leading zero" {
    # Regression test: bash arithmetic treats a leading-zero numeral as
    # octal ($((POLL_ATTEMPTS * POLL_INTERVAL_S)) in the remote script's
    # timeout message would hard-fail on e.g. "038" - 8/9 aren't valid
    # octal digits). Caught in review before shipping.
    valid_env
    export CC_KOMODO_POLL_ATTEMPTS="030"
    run run_script "voipbin-test-stack" "$COMPOSE_FIXTURE"
    [[ "$status" -ne 0 ]]
    [[ "$output" == *"CC_KOMODO_POLL_ATTEMPTS must be a positive integer with no leading zero"* ]]
}

@test "refuses a CC_KOMODO_POLL_INTERVAL_S with a leading zero" {
    valid_env
    export CC_KOMODO_POLL_INTERVAL_S="02"
    run run_script "voipbin-test-stack" "$COMPOSE_FIXTURE"
    [[ "$status" -ne 0 ]]
    [[ "$output" == *"CC_KOMODO_POLL_INTERVAL_S must be a positive integer with no leading zero"* ]]
}

@test "refuses a CC_KOMODO_API_KEY containing an embedded newline" {
    valid_env
    export CC_KOMODO_API_KEY=$'line-one\nline-two'
    run run_script "voipbin-test-stack" "$COMPOSE_FIXTURE"
    [[ "$status" -ne 0 ]]
    [[ "$output" == *"CC_KOMODO_API_KEY must not contain a newline"* ]]
}

@test "refuses a CC_KOMODO_API_SECRET containing an embedded newline" {
    valid_env
    export CC_KOMODO_API_SECRET=$'line-one\nline-two'
    run run_script "voipbin-test-stack" "$COMPOSE_FIXTURE"
    [[ "$status" -ne 0 ]]
    [[ "$output" == *"CC_KOMODO_API_SECRET must not contain a newline"* ]]
}

@test "refuses a CC_KOMODO_API_KEY containing a double-quote" {
    # This validation is the load-bearing control that lets call_api's
    # curl -K header_config skip implementing curl's config-file
    # quote-escaping (see script.sh's comment on the check). Without it,
    # a key/secret containing '"' could break out of the `header = "..."`
    # directive's quoting on the remote side.
    valid_env
    export CC_KOMODO_API_KEY='has"a-quote'
    run run_script "voipbin-test-stack" "$COMPOSE_FIXTURE"
    [[ "$status" -ne 0 ]]
    [[ "$output" == *"CC_KOMODO_API_KEY must not contain a double-quote or backslash character"* ]]
}

@test "refuses a CC_KOMODO_API_KEY containing a backslash" {
    valid_env
    export CC_KOMODO_API_KEY='has\a-backslash'
    run run_script "voipbin-test-stack" "$COMPOSE_FIXTURE"
    [[ "$status" -ne 0 ]]
    [[ "$output" == *"CC_KOMODO_API_KEY must not contain a double-quote or backslash character"* ]]
}

@test "refuses a CC_KOMODO_API_SECRET containing a double-quote" {
    valid_env
    export CC_KOMODO_API_SECRET='has"a-quote'
    run run_script "voipbin-test-stack" "$COMPOSE_FIXTURE"
    [[ "$status" -ne 0 ]]
    [[ "$output" == *"CC_KOMODO_API_SECRET must not contain a double-quote or backslash character"* ]]
}

@test "refuses a CC_KOMODO_API_SECRET containing a backslash" {
    valid_env
    export CC_KOMODO_API_SECRET='has\a-backslash'
    run run_script "voipbin-test-stack" "$COMPOSE_FIXTURE"
    [[ "$status" -ne 0 ]]
    [[ "$output" == *"CC_KOMODO_API_SECRET must not contain a double-quote or backslash character"* ]]
}

@test "a different stack name propagates correctly (not hardcoded)" {
    valid_env
    install_fake_ssh 0
    run run_script "some-other-stack" "$COMPOSE_FIXTURE"

    local remote_cmd
    remote_cmd="$(ssh_call_remote_command 1)"
    [[ "$remote_cmd" == *"STACK_NAME=\"some-other-stack\""* ]]
}

@test "failure path: reports failure and manual-recovery guidance when ssh/remote exits non-zero" {
    valid_env
    install_fake_ssh 1
    run run_script "voipbin-test-stack" "$COMPOSE_FIXTURE"

    [[ "$status" -ne 0 ]]
    [[ "$output" == *"Deploy failed"* ]]
    [[ "$output" == *"does not auto-rollback"* ]]
    [[ "$output" == *"voipbin-test-stack"* ]]
}

@test "the SSH key temp file is removed after the script exits (success path)" {
    valid_env
    install_fake_ssh 0
    run run_script "voipbin-test-stack" "$COMPOSE_FIXTURE"
    [[ "$status" -eq 0 ]]

    local args key_file
    args="$(ssh_call_args 1)"
    key_file="$(echo "$args" | grep -A1 '^-i$' | tail -1)"
    [[ -n "$key_file" ]]
    [[ ! -f "$key_file" ]]
}

@test "the SSH key temp file is removed after the script exits (failure path)" {
    valid_env
    install_fake_ssh 1
    run run_script "voipbin-test-stack" "$COMPOSE_FIXTURE"
    [[ "$status" -ne 0 ]]

    local args key_file
    args="$(ssh_call_args 1)"
    key_file="$(echo "$args" | grep -A1 '^-i$' | tail -1)"
    [[ -n "$key_file" ]]
    [[ ! -f "$key_file" ]]
}

secret_env() {
    local content="-----BEGIN OPENSSH PRIVATE KEY-----
super-secret-deploy-key-xyz789
-----END OPENSSH PRIVATE KEY-----"
    export CC_DEPLOY_SSH_KEY="$(printf '%s' "$content" | base64 -w0)"
    export CC_KOMODO_API_KEY="super-secret-api-key-abc123"
    export CC_KOMODO_API_SECRET="super-secret-api-secret-def456"
}

@test "SSH key value never appears in stdout/stderr output" {
    secret_env
    install_fake_ssh 0
    run run_script "voipbin-test-stack" "$COMPOSE_FIXTURE"

    [[ "$status" -eq 0 ]]
    [[ "$output" != *"super-secret-deploy-key-xyz789"* ]]
    [[ "$output" != *"$CC_DEPLOY_SSH_KEY"* ]]
}

@test "Komodo API key/secret never appear in stdout/stderr output" {
    secret_env
    install_fake_ssh 0
    run run_script "voipbin-test-stack" "$COMPOSE_FIXTURE"

    [[ "$status" -eq 0 ]]
    [[ "$output" != *"super-secret-api-key-abc123"* ]]
    [[ "$output" != *"super-secret-api-secret-def456"* ]]
}

@test "Komodo API key/secret never appear in the ssh invocation's argv" {
    secret_env
    install_fake_ssh 0
    run run_script "voipbin-test-stack" "$COMPOSE_FIXTURE"

    [[ "$status" -eq 0 ]]
    local args
    args="$(ssh_call_args 1)"
    [[ "$args" != *"super-secret-api-key-abc123"* ]]
    [[ "$args" != *"super-secret-api-secret-def456"* ]]
}

@test "Komodo API key/secret never appear in the remote command string" {
    secret_env
    install_fake_ssh 0
    run run_script "voipbin-test-stack" "$COMPOSE_FIXTURE"

    [[ "$status" -eq 0 ]]
    local remote_cmd
    remote_cmd="$(ssh_call_remote_command 1)"
    [[ "$remote_cmd" != *"super-secret-api-key-abc123"* ]]
    [[ "$remote_cmd" != *"super-secret-api-secret-def456"* ]]
}

@test "Komodo API key/secret are delivered via ssh stdin, as the first two lines, in order" {
    secret_env
    install_fake_ssh 0
    run run_script "voipbin-test-stack" "$COMPOSE_FIXTURE"

    [[ "$status" -eq 0 ]]
    local stdin_content first_line second_line
    stdin_content="$(ssh_call_stdin 1)"
    first_line="$(printf '%s' "$stdin_content" | sed -n '1p')"
    second_line="$(printf '%s' "$stdin_content" | sed -n '2p')"
    [[ "$first_line" == "super-secret-api-key-abc123" ]]
    [[ "$second_line" == "super-secret-api-secret-def456" ]]
}

@test "refuses when CC_DEPLOY_SSH_KEY decoding leaves no readable temp file after failure (no leftover key material)" {
    export CC_DEPLOY_SSH_KEY="not-valid-base64!!!"
    export CC_KOMODO_API_KEY="k"
    export CC_KOMODO_API_SECRET="s"
    local before after
    before="$(find "$TEST_TEMP_DIR" -maxdepth 1 -type f | wc -l)"
    run run_script "voipbin-test-stack" "$COMPOSE_FIXTURE"
    [[ "$status" -ne 0 ]]
    after="$(find "$TEST_TEMP_DIR" -maxdepth 1 -type f | wc -l)"
    # mktemp files land outside TEST_TEMP_DIR (system TMPDIR) - this just
    # guards against a future refactor accidentally writing key material
    # into our own fixture directory.
    [[ "$before" -eq "$after" ]]
}
