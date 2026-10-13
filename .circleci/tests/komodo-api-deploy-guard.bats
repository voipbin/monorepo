#!/usr/bin/env bats
# Tests for the superseded-deploy guard in .circleci/scripts/komodo-api-deploy.sh
# (VOIP-1586, env DEPLOY_REQUIRE_LATEST_FOR_PATHS). Every case uses a real
# temporary git repository with a bare origin; failure branches that a real
# repository cannot produce (git subcommand failures) use a PATH shim for git.
# Kept bats 1.2.1 compatible (the shell-tests job installs bats from apt).

load 'test_helper'

setup() {
    setup_test_env
    install_fake_curl
    export CC_KOMODO_API_KEY="test-key"
    export CC_KOMODO_API_SECRET="test-secret"
    export GIT_AUTHOR_NAME="t" GIT_AUTHOR_EMAIL="t@example.com"
    export GIT_COMMITTER_NAME="t" GIT_COMMITTER_EMAIL="t@example.com"
    export GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_SYSTEM=/dev/null
    REAL_GIT="$(PATH="$ORIGINAL_PATH" command -v git)"
    COMPOSE_FILE="$TEST_TEMP_DIR/docker-compose.yml"
    cat > "$COMPOSE_FILE" <<'EOF'
services:
  agent-manager:
    image: voipbin/bin-agent-manager:abc123
EOF
    ORIGIN="$TEST_TEMP_DIR/origin.git"
    WORK="$TEST_TEMP_DIR/work"
    CLONE="$TEST_TEMP_DIR/clone"
    git init -q --bare -b main "$ORIGIN"
    git init -q -b main "$WORK"
    git -C "$WORK" remote add origin "file://$ORIGIN"
}

teardown() {
    teardown_test_env
}

# commit_file <path> <content> -> prints the new commit sha
commit_file() {
    mkdir -p "$(dirname "$WORK/$1")"
    echo "$2" >> "$WORK/$1"
    git -C "$WORK" add -A
    git -C "$WORK" commit -q -m "change $1"
    git -C "$WORK" rev-parse HEAD
}

push_origin() {
    git -C "$WORK" push -q origin main
}

full_clone() {
    git clone -q "file://$ORIGIN" "$CLONE"
}

shallow_clone() {
    git clone -q --depth 1 "file://$ORIGIN" "$CLONE"
    [ "$(git -C "$CLONE" rev-parse --is-shallow-repository)" = "true" ]
}

# run_deploy_in <dir> : runs the deploy script from <dir>
run_deploy_in() {
    local dir="$1"
    ( cd "$dir" && bash "$SCRIPTS_DIR/komodo-api-deploy.sh" bin-agent-manager "$COMPOSE_FILE" )
}

# A git shim that fails when its args contain the given subcommand word (and,
# optionally, the given extra flag); otherwise delegates to the real git. The
# guard calls `git --literal-pathspecs log ...`, so the subcommand is not $1:
# the shim scans all args.
install_git_shim() {
    local fail_word="$1" fail_flag="${2:-}"
    cat > "$MOCK_BIN_DIR/git" <<EOF
#!/bin/bash
hit_word=0 hit_flag=0
for a in "\$@"; do
    [ "\$a" = "$fail_word" ] && hit_word=1
    if [ -n "$fail_flag" ] && [ "\$a" = "$fail_flag" ]; then hit_flag=1; fi
done
if [ \$hit_word -eq 1 ] && { [ -z "$fail_flag" ] || [ \$hit_flag -eq 1 ]; }; then
    echo "git-shim: forced failure for \$*" >&2
    exit 1
fi
exec "$REAL_GIT" "\$@"
EOF
    chmod +x "$MOCK_BIN_DIR/git"
}

# Records every git invocation to $GIT_CALL_LOG, then delegates.
install_git_logger() {
    export GIT_CALL_LOG="$TEST_TEMP_DIR/git_calls.log"
    : > "$GIT_CALL_LOG"
    cat > "$MOCK_BIN_DIR/git" <<EOF
#!/bin/bash
echo "\$*" >> "$GIT_CALL_LOG"
exec "$REAL_GIT" "\$@"
EOF
    chmod +x "$MOCK_BIN_DIR/git"
}

@test "guard off by default: git is never called and no guard message appears, even outside a repo" {
    install_git_logger
    unset DEPLOY_REQUIRE_LATEST_FOR_PATHS CIRCLE_SHA1
    run run_deploy_in /
    [[ "$output" != *"guard"* ]]
    [[ "$output" != *"DEPLOY_SUPERSEDED"* ]]
    [ ! -s "$GIT_CALL_LOG" ]
    # Off means the deploy goes on to Komodo (the fake curl has no queued
    # response, so it fails there), not that it stops early.
    [ "$(curl_call_count)" -gt 0 ]
}

@test "guard off when the variable is the empty string" {
    install_git_logger
    export DEPLOY_REQUIRE_LATEST_FOR_PATHS=""
    run run_deploy_in /
    [[ "$output" != *"guard"* ]]
    [ ! -s "$GIT_CALL_LOG" ]
    [ "$(curl_call_count)" -gt 0 ]
}

@test "guard on: the newest commit touching the path passes" {
    commit_file bin-agent-manager/a.txt one >/dev/null
    sha="$(commit_file bin-agent-manager/a.txt two)"
    push_origin; full_clone
    export DEPLOY_REQUIRE_LATEST_FOR_PATHS="bin-agent-manager" CIRCLE_SHA1="$sha"
    run run_deploy_in "$CLONE"
    [[ "$output" == *"Superseded-deploy guard passed"* ]]
    [[ "$output" != *"DEPLOY_SUPERSEDED"* ]]
}

@test "guard on: commits after it that touch only other paths do not supersede it" {
    sha="$(commit_file bin-agent-manager/a.txt one)"
    commit_file bin-other-manager/b.txt two >/dev/null
    commit_file docs/c.txt three >/dev/null
    push_origin; full_clone
    export DEPLOY_REQUIRE_LATEST_FOR_PATHS="bin-agent-manager" CIRCLE_SHA1="$sha"
    run run_deploy_in "$CLONE"
    [[ "$output" == *"Superseded-deploy guard passed"* ]]
}

@test "guard on: a detached-HEAD checkout of the old commit (as CircleCI does) is still superseded" {
    old="$(commit_file bin-agent-manager/a.txt one)"
    new="$(commit_file bin-agent-manager/a.txt two)"
    push_origin; full_clone
    # CircleCI checks out CIRCLE_SHA1 detached, so HEAD is the old commit and
    # looking at HEAD instead of origin/main would always report "latest".
    git -C "$CLONE" checkout -q --detach "$old"
    export DEPLOY_REQUIRE_LATEST_FOR_PATHS="bin-agent-manager" CIRCLE_SHA1="$old"
    run run_deploy_in "$CLONE"
    [ "$status" -eq 1 ]
    [[ "$output" == *"DEPLOY_SUPERSEDED sha=$old latest=$new"* ]]
    [ "$(curl_call_count)" -eq 0 ]
}

@test "guard on: a stale local origin/main is refreshed from the remote before comparing" {
    old="$(commit_file bin-agent-manager/a.txt one)"
    push_origin; full_clone
    new="$(commit_file bin-agent-manager/a.txt two)"
    push_origin
    # The clone's origin/main still points at the old commit until the guard fetches.
    export DEPLOY_REQUIRE_LATEST_FOR_PATHS="bin-agent-manager" CIRCLE_SHA1="$old"
    run run_deploy_in "$CLONE"
    [ "$status" -eq 1 ]
    [[ "$output" == *"DEPLOY_SUPERSEDED sha=$old latest=$new"* ]]
}

@test "guard on: an older commit is superseded by a newer commit touching the path, and Komodo is never called" {
    old="$(commit_file bin-agent-manager/a.txt one)"
    new="$(commit_file bin-agent-manager/a.txt two)"
    push_origin; full_clone
    export DEPLOY_REQUIRE_LATEST_FOR_PATHS="bin-agent-manager" CIRCLE_SHA1="$old"
    run run_deploy_in "$CLONE"
    [ "$status" -eq 1 ]
    [[ "$output" == *"DEPLOY_SUPERSEDED sha=$old latest=$new"* ]]
    [ "$(curl_call_count)" -eq 0 ]
}

@test "guard on: re-running the newest commit passes even after unrelated commits piled up" {
    sha="$(commit_file bin-agent-manager/a.txt one)"
    commit_file bin-other-manager/b.txt two >/dev/null
    push_origin; full_clone
    export DEPLOY_REQUIRE_LATEST_FOR_PATHS="bin-agent-manager" CIRCLE_SHA1="$sha"
    run run_deploy_in "$CLONE"
    [[ "$output" == *"Superseded-deploy guard passed"* ]]
}

@test "guard on: a path prefix is not confused with a sibling directory (bin-agent-manager vs bin-agent-manager-x)" {
    sha="$(commit_file bin-agent-manager/a.txt one)"
    commit_file bin-agent-manager-x/b.txt two >/dev/null
    push_origin; full_clone
    export DEPLOY_REQUIRE_LATEST_FOR_PATHS="bin-agent-manager" CIRCLE_SHA1="$sha"
    run run_deploy_in "$CLONE"
    [[ "$output" == *"Superseded-deploy guard passed"* ]]
}

@test "guard on: with several paths a newer commit on any of them supersedes" {
    sha="$(commit_file bin-agent-manager/a.txt one)"
    new="$(commit_file bin-openapi-manager/b.txt two)"
    push_origin; full_clone
    export DEPLOY_REQUIRE_LATEST_FOR_PATHS="bin-agent-manager bin-openapi-manager" CIRCLE_SHA1="$sha"
    run run_deploy_in "$CLONE"
    [ "$status" -eq 1 ]
    [[ "$output" == *"DEPLOY_SUPERSEDED sha=$sha latest=$new"* ]]
}

@test "guard on: a shallow clone is deepened and then passes" {
    commit_file bin-agent-manager/a.txt one >/dev/null
    sha="$(commit_file bin-agent-manager/a.txt two)"
    push_origin; shallow_clone
    export DEPLOY_REQUIRE_LATEST_FOR_PATHS="bin-agent-manager" CIRCLE_SHA1="$sha"
    run run_deploy_in "$CLONE"
    [[ "$output" == *"Shallow clone: deepening"* ]]
    [[ "$output" == *"Superseded-deploy guard passed"* ]]
    [ "$(git -C "$CLONE" rev-parse --is-shallow-repository)" = "false" ]
}

@test "guard on: when --unshallow fails the --deepen fallback is used and passes" {
    commit_file bin-agent-manager/a.txt one >/dev/null
    sha="$(commit_file bin-agent-manager/a.txt two)"
    push_origin; shallow_clone
    install_git_shim "--unshallow"
    export DEPLOY_REQUIRE_LATEST_FOR_PATHS="bin-agent-manager" CIRCLE_SHA1="$sha"
    run run_deploy_in "$CLONE"
    [[ "$output" == *"Superseded-deploy guard passed"* ]]
}

@test "guard on: a clone that stays shallow after deepening fails closed" {
    commit_file bin-agent-manager/a.txt one >/dev/null
    sha="$(commit_file bin-agent-manager/a.txt two)"
    push_origin; shallow_clone
    # Only the unshallow/deepen fetches fail; the plain first fetch succeeds.
    cat > "$MOCK_BIN_DIR/git" <<EOF
#!/bin/bash
for a in "\$@"; do
    if [ "\$a" = "--unshallow" ] || [ "\${a#--deepen}" != "\$a" ]; then
        echo "git-shim: deepening blocked" >&2
        exit 1
    fi
done
exec "$REAL_GIT" "\$@"
EOF
    chmod +x "$MOCK_BIN_DIR/git"
    export DEPLOY_REQUIRE_LATEST_FOR_PATHS="bin-agent-manager" CIRCLE_SHA1="$sha"
    run run_deploy_in "$CLONE"
    [ "$status" -eq 1 ]
    [[ "$output" == *"still shallow"* ]]
    [ "$(curl_call_count)" -eq 0 ]
}

@test "guard on: a clone without a fetch refspec still gets origin/main via the explicit refspec" {
    sha="$(commit_file bin-agent-manager/a.txt one)"
    push_origin; full_clone
    git -C "$CLONE" config --unset-all remote.origin.fetch
    git -C "$CLONE" update-ref -d refs/remotes/origin/main
    export DEPLOY_REQUIRE_LATEST_FOR_PATHS="bin-agent-manager" CIRCLE_SHA1="$sha"
    run run_deploy_in "$CLONE"
    [[ "$output" == *"Superseded-deploy guard passed"* ]]
}

@test "guard on: a failing fetch of origin/main fails closed" {
    sha="$(commit_file bin-agent-manager/a.txt one)"
    push_origin; full_clone
    git -C "$CLONE" remote set-url origin "file://$TEST_TEMP_DIR/does-not-exist.git"
    export DEPLOY_REQUIRE_LATEST_FOR_PATHS="bin-agent-manager" CIRCLE_SHA1="$sha"
    run run_deploy_in "$CLONE"
    [ "$status" -eq 1 ]
    [[ "$output" == *"could not fetch origin/main"* ]]
    [ "$(curl_call_count)" -eq 0 ]
}

@test "guard on: a path with no history on origin/main fails closed" {
    sha="$(commit_file bin-agent-manager/a.txt one)"
    push_origin; full_clone
    export DEPLOY_REQUIRE_LATEST_FOR_PATHS="bin-nope-manager" CIRCLE_SHA1="$sha"
    run run_deploy_in "$CLONE"
    [ "$status" -eq 1 ]
    [[ "$output" == *"no history on origin/main"* ]]
    [ "$(curl_call_count)" -eq 0 ]
}

@test "guard on: a failing git log (not an empty result) fails closed with its own message" {
    sha="$(commit_file bin-agent-manager/a.txt one)"
    push_origin; full_clone
    install_git_shim "log"
    export DEPLOY_REQUIRE_LATEST_FOR_PATHS="bin-agent-manager" CIRCLE_SHA1="$sha"
    run run_deploy_in "$CLONE"
    [ "$status" -eq 1 ]
    [[ "$output" == *"git log failed"* ]]
    [ "$(curl_call_count)" -eq 0 ]
}

@test "guard on: a missing CIRCLE_SHA1 fails closed" {
    commit_file bin-agent-manager/a.txt one >/dev/null
    push_origin; full_clone
    export DEPLOY_REQUIRE_LATEST_FOR_PATHS="bin-agent-manager"
    unset CIRCLE_SHA1
    run run_deploy_in "$CLONE"
    [ "$status" -eq 1 ]
    [[ "$output" == *"CIRCLE_SHA1 is not"* ]]
    [ "$(curl_call_count)" -eq 0 ]
}

@test "guard on: outside a git checkout fails closed" {
    export DEPLOY_REQUIRE_LATEST_FOR_PATHS="bin-agent-manager" CIRCLE_SHA1="abc"
    mkdir -p "$TEST_TEMP_DIR/not-a-repo"
    run run_deploy_in "$TEST_TEMP_DIR/not-a-repo"
    [ "$status" -eq 1 ]
    [[ "$output" == *"not one"* ]]
    [ "$(curl_call_count)" -eq 0 ]
}

@test "guard on: a value with no paths (whitespace only) fails closed" {
    export DEPLOY_REQUIRE_LATEST_FOR_PATHS="   " CIRCLE_SHA1="abc"
    run run_deploy_in /
    [ "$status" -eq 1 ]
    [[ "$output" == *"has no paths"* ]]
    [ "$(curl_call_count)" -eq 0 ]
}

@test "guard on: path entries starting with a dash or containing a colon are rejected" {
    export CIRCLE_SHA1="abc"
    export DEPLOY_REQUIRE_LATEST_FOR_PATHS="-x"
    run run_deploy_in /
    [ "$status" -eq 1 ]
    [[ "$output" == *"entry is not allowed"* ]]
    export DEPLOY_REQUIRE_LATEST_FOR_PATHS="a:b"
    run run_deploy_in /
    [ "$status" -eq 1 ]
    [[ "$output" == *"entry is not allowed"* ]]
    [ "$(curl_call_count)" -eq 0 ]
}

@test "guard on: glob characters are literal, so bin-* matches nothing and fails closed" {
    sha="$(commit_file bin-agent-manager/a.txt one)"
    push_origin; full_clone
    export DEPLOY_REQUIRE_LATEST_FOR_PATHS="bin-*" CIRCLE_SHA1="$sha"
    run run_deploy_in "$CLONE"
    [ "$status" -eq 1 ]
    [[ "$output" == *"no history on origin/main"* ]]
}

@test "guard on: a value with an embedded newline is rejected, not silently truncated" {
    export DEPLOY_REQUIRE_LATEST_FOR_PATHS=$'bin-agent-manager\nbin-other-manager' CIRCLE_SHA1="abc"
    run run_deploy_in /
    [ "$status" -eq 1 ]
    [[ "$output" == *"must be a single line"* ]]
    [ "$(curl_call_count)" -eq 0 ]
}
