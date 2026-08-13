#!/usr/bin/env bats
# Tests for .circleci/scripts/ssh-deploy.sh

load 'test_helper'

setup() {
    setup_test_env
}

teardown() {
    teardown_test_env
}

run_script() {
    bash "$SCRIPTS_DIR/ssh-deploy.sh" "$@"
}

# A fake but well-formed OpenSSH-shaped private key, base64-encoded (as the
# real CC_DEPLOY_SSH_KEY must be) - matches what the script now requires:
# valid base64 that decodes to content containing a "PRIVATE KEY" marker.
FAKE_KEY_CONTENT="-----BEGIN OPENSSH PRIVATE KEY-----
ZmFrZS1rZXktbWF0ZXJpYWwtZm9yLXRlc3Rpbmctb25seQ==
-----END OPENSSH PRIVATE KEY-----"

valid_env() {
    export CC_DEPLOY_SSH_KEY="$(printf '%s' "$FAKE_KEY_CONTENT" | base64 -w0)"
    export CIRCLE_SHA1="cccccccccccccccccccccccccccccccccccccccc"
}

@test "-h/--help prints usage and exits 0 without requiring any env" {
    run run_script --help
    [[ "$status" -eq 0 ]]
    [[ "$output" == *"deploy one already-pushed image to bm-nyc-01"* ]]
}

@test "refuses with wrong argument count" {
    run run_script "voipbin/bin-call-manager"
    [[ "$status" -ne 0 ]]
    [[ "$output" == *"Usage:"* || "$output" == *"deploy one already-pushed image"* ]]
}

@test "refuses an image-repo not shaped like voipbin/<name>" {
    valid_env
    run run_script "not-voipbin/bin-call-manager" "bin-call-manager"
    [[ "$status" -ne 0 ]]
    [[ "$output" == *"image-repo must look like"* ]]
}

@test "refuses a compose-service-name with unexpected characters" {
    valid_env
    run run_script "voipbin/bin-call-manager" "bin_call_manager; rm -rf /"
    [[ "$status" -ne 0 ]]
    [[ "$output" == *"compose-service-name must be"* ]]
}

@test "refuses to run when CC_DEPLOY_SSH_KEY is not set" {
    export CIRCLE_SHA1="cccccccccccccccccccccccccccccccccccccccc"
    unset CC_DEPLOY_SSH_KEY
    run run_script "voipbin/bin-call-manager" "bin-call-manager"
    [[ "$status" -ne 0 ]]
    [[ "$output" == *"CC_DEPLOY_SSH_KEY is not set"* ]]
}

@test "refuses to run when CIRCLE_SHA1 is not set" {
    export CC_DEPLOY_SSH_KEY="fake-key-material"
    unset CIRCLE_SHA1
    run run_script "voipbin/bin-call-manager" "bin-call-manager"
    [[ "$status" -ne 0 ]]
    [[ "$output" == *"CIRCLE_SHA1 is not set"* ]]
}

@test "refuses a CIRCLE_SHA1 that isn't a full 40-char SHA" {
    valid_env
    export CIRCLE_SHA1="abc123"
    run run_script "voipbin/bin-call-manager" "bin-call-manager"
    [[ "$status" -ne 0 ]]
    [[ "$output" == *"40-char git SHA"* ]]
}

@test "happy path: succeeds and reports success when ssh exits 0" {
    valid_env
    install_fake_ssh 0
    run run_script "voipbin/bin-call-manager" "bin-call-manager"

    [[ "$status" -eq 0 ]]
    [[ "$output" == *"Deploy succeeded"* ]]
    [[ "$output" == *"bin-call-manager"* ]]
    # exactly one ssh invocation
    [[ "$(ssh_call_count)" -eq 1 ]]
}

@test "ssh is invoked against the correct user@host with host-key pinning (not runtime ssh-keyscan)" {
    valid_env
    install_fake_ssh 0
    run run_script "voipbin/bin-call-manager" "bin-call-manager"

    local args
    args="$(ssh_call_args 1)"
    [[ "$args" == *"deploy@104.243.38.39"* ]]
    [[ "$args" == *"StrictHostKeyChecking=yes"* ]]
    [[ "$args" == *"UserKnownHostsFile="* ]]
    # No ssh-keyscan invocation anywhere - host keys are pinned, not
    # trust-on-first-use at CI runtime.
    [[ "$output" != *"ssh-keyscan"* ]]
}

@test "the remote command targets the LIVE versions.lock/docker-compose.yml, never the .dist templates" {
    valid_env
    install_fake_ssh 0
    run run_script "voipbin/bin-call-manager" "bin-call-manager"

    local remote_cmd
    remote_cmd="$(ssh_call_remote_command 1)"
    [[ "$remote_cmd" == *"LOCK_FILE='/opt/voipbin/install/versions.lock'"* ]]
    [[ "$remote_cmd" == *"COMPOSE_FILE='/opt/voipbin/install/docker-compose.yml'"* ]]
    [[ "$remote_cmd" != *"versions.lock.dist"* ]]
    [[ "$remote_cmd" != *"docker-compose.yml.dist"* ]]
}

@test "the remote command bumps the correct image-repo, tag, and compose service" {
    valid_env
    install_fake_ssh 0
    run run_script "voipbin/bin-call-manager" "bin-call-manager"

    local remote_cmd
    remote_cmd="$(ssh_call_remote_command 1)"
    [[ "$remote_cmd" == *"bump-image-digest.sh 'voipbin/bin-call-manager' 'voipbin/bin-call-manager:cccccccccccccccccccccccccccccccccccccccc' 'cccccccccccccccccccccccccccccccccccccccc'"* ]]
    [[ "$remote_cmd" == *"docker compose pull 'bin-call-manager'"* ]]
    [[ "$remote_cmd" == *"docker compose up -d 'bin-call-manager'"* ]]
}

@test "a different service's arguments propagate correctly (not hardcoded to bin-call-manager)" {
    valid_env
    install_fake_ssh 0
    run run_script "voipbin/bin-agent-manager" "bin-agent-manager"

    local remote_cmd
    remote_cmd="$(ssh_call_remote_command 1)"
    [[ "$remote_cmd" == *"bump-image-digest.sh 'voipbin/bin-agent-manager' 'voipbin/bin-agent-manager:cccccccccccccccccccccccccccccccccccccccc'"* ]]
    [[ "$remote_cmd" == *"docker compose up -d 'bin-agent-manager'"* ]]
}

@test "failure path: reports failure and manual-recovery guidance when ssh/remote exits non-zero" {
    valid_env
    install_fake_ssh 1
    run run_script "voipbin/bin-call-manager" "bin-call-manager"

    [[ "$status" -ne 0 ]]
    [[ "$output" == *"Deploy failed"* ]]
    [[ "$output" == *"does not auto-rollback"* ]]
    # Manual recovery guidance references the actual image-repo and service
    [[ "$output" == *"voipbin/bin-call-manager"* ]]
    [[ "$output" == *"docker compose up -d bin-call-manager"* ]]
}

secret_key_env() {
    # The embedded marker (not just the base64 blob itself) is what these
    # tests check never leaks - covers both "the raw CC_DEPLOY_SSH_KEY
    # value" and "the decoded key content" leaking.
    local content="-----BEGIN OPENSSH PRIVATE KEY-----
super-secret-deploy-key-xyz789
-----END OPENSSH PRIVATE KEY-----"
    export CC_DEPLOY_SSH_KEY="$(printf '%s' "$content" | base64 -w0)"
    export CIRCLE_SHA1="cccccccccccccccccccccccccccccccccccccccc"
}

@test "SSH key value never appears in stdout/stderr output" {
    secret_key_env
    install_fake_ssh 0
    run run_script "voipbin/bin-call-manager" "bin-call-manager"

    [[ "$status" -eq 0 ]]
    [[ "$output" != *"super-secret-deploy-key-xyz789"* ]]
    [[ "$output" != *"$CC_DEPLOY_SSH_KEY"* ]]
}

@test "SSH key value never appears anywhere in the ssh invocation's argv (only the key FILE path is passed)" {
    secret_key_env
    install_fake_ssh 0
    run run_script "voipbin/bin-call-manager" "bin-call-manager"

    [[ "$status" -eq 0 ]]
    local args
    args="$(ssh_call_args 1)"
    [[ "$args" != *"super-secret-deploy-key-xyz789"* ]]
    [[ "$args" != *"$CC_DEPLOY_SSH_KEY"* ]]
}

@test "refuses when CC_DEPLOY_SSH_KEY is not valid base64" {
    export CC_DEPLOY_SSH_KEY="not-valid-base64!!!"
    export CIRCLE_SHA1="cccccccccccccccccccccccccccccccccccccccc"
    run run_script "voipbin/bin-call-manager" "bin-call-manager"

    [[ "$status" -ne 0 ]]
    [[ "$output" == *"not valid base64"* ]]
}

@test "refuses when CC_DEPLOY_SSH_KEY decodes to something without a PRIVATE KEY marker" {
    export CC_DEPLOY_SSH_KEY="$(printf 'just some unrelated text' | base64 -w0)"
    export CIRCLE_SHA1="cccccccccccccccccccccccccccccccccccccccc"
    run run_script "voipbin/bin-call-manager" "bin-call-manager"

    [[ "$status" -ne 0 ]]
    [[ "$output" == *"does not look like a private key"* ]]
}

@test "tolerates a valid base64 payload wrapped in stray characters a copy-paste can introduce" {
    # Regression test: a real CircleCI run failed on a value that decoded
    # incorrectly after being pasted into the context-variable web UI, even
    # though the underlying base64 payload itself was correct - surrounding
    # whitespace/quotes/CR are exactly the kind of stray characters a
    # copy-paste can pick up. This must still succeed, not just fail with a
    # clearer error.
    valid_env
    local clean="$CC_DEPLOY_SSH_KEY"
    export CC_DEPLOY_SSH_KEY=$'  "'"$clean"$'"\r\n\t '
    install_fake_ssh 0
    run run_script "voipbin/bin-call-manager" "bin-call-manager"

    [[ "$status" -eq 0 ]]
    [[ "$output" == *"Deploy succeeded"* ]]
}

@test "still rejects a base64 payload corrupted by more than stray wrapper characters" {
    # Companion to the tolerance test above: stripping only non-base64
    # characters must not mask a genuinely truncated/reordered payload -
    # confirm corruption of the actual content is still caught by one of
    # the two downstream checks (an invalid-length truncation like this one
    # typically fails base64 decoding itself; a same-length reordering
    # would instead fail the PRIVATE KEY marker check - either is an
    # acceptable backstop, hence the OR below).
    valid_env
    # Drop the middle of the payload rather than just wrap it - this is
    # corruption of the actual content, not incidental wrapper noise.
    export CC_DEPLOY_SSH_KEY="${CC_DEPLOY_SSH_KEY:0:10}"
    run run_script "voipbin/bin-call-manager" "bin-call-manager"

    [[ "$status" -ne 0 ]]
    [[ "$output" == *"not valid base64"* || "$output" == *"does not look like a private key"* ]]
}

@test "the SSH key temp file is removed after the script exits (success path)" {
    valid_env
    install_fake_ssh 0
    # Capture the key-file path via a wrapper: run in a subshell so we can
    # inspect $MOCK_BIN_DIR's sibling temp files before/after - simpler:
    # assert no stray files matching the mktemp pattern remain in TMPDIR
    # under our controlled TEST_TEMP_DIR-adjacent area is fragile, so
    # instead assert indirectly via lsof-free approach: grep the ssh args
    # for the -i path, then check it's gone post-run.
    run run_script "voipbin/bin-call-manager" "bin-call-manager"
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
    run run_script "voipbin/bin-call-manager" "bin-call-manager"
    [[ "$status" -ne 0 ]]

    local args key_file
    args="$(ssh_call_args 1)"
    key_file="$(echo "$args" | grep -A1 '^-i$' | tail -1)"
    [[ -n "$key_file" ]]
    [[ ! -f "$key_file" ]]
}
