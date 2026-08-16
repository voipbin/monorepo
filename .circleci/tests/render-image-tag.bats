#!/usr/bin/env bats
# Tests for .circleci/scripts/render-image-tag.sh

load 'test_helper'

setup() {
    setup_test_env
    COMPOSE_FILE="$TEST_TEMP_DIR/docker-compose.yml"
}

teardown() {
    teardown_test_env
}

run_render() {
    bash "$SCRIPTS_DIR/render-image-tag.sh" "$@"
}

@test "usage: prints help and exits 0 with -h" {
    run run_render -h
    [ "$status" -eq 0 ]
    [[ "$output" == *"Usage:"* ]]
}

@test "usage: exits 1 with wrong arg count" {
    run run_render "$COMPOSE_FILE"
    [ "$status" -eq 1 ]
}

@test "validation: rejects a nonexistent compose file" {
    run run_render /nonexistent/path.yml abc123
    [ "$status" -eq 1 ]
    [[ "$output" == *"does not exist"* ]]
}

@test "validation: rejects an empty tag" {
    echo 'image: voipbin/bin-call-manager:__IMAGE_TAG__' > "$COMPOSE_FILE"
    run run_render "$COMPOSE_FILE" ""
    [ "$status" -eq 1 ]
    [[ "$output" == *"must be a non-empty string"* ]]
}

@test "validation: rejects a tag containing shell metacharacters" {
    echo 'image: voipbin/bin-call-manager:__IMAGE_TAG__' > "$COMPOSE_FILE"
    run run_render "$COMPOSE_FILE" 'abc; rm -rf /'
    [ "$status" -eq 1 ]
    [[ "$output" == *"must be a non-empty string"* ]]
}

@test "placeholder guard: fails loudly, unmodified file, when __IMAGE_TAG__ is missing" {
    echo 'image: voipbin/bin-call-manager:already-pinned' > "$COMPOSE_FILE"
    cp "$COMPOSE_FILE" "$COMPOSE_FILE.orig"
    run run_render "$COMPOSE_FILE" abc123
    [ "$status" -eq 1 ]
    [[ "$output" == *"placeholder not found"* ]]
    diff "$COMPOSE_FILE" "$COMPOSE_FILE.orig"
}

@test "substitution: replaces __IMAGE_TAG__ with the given tag, in place" {
    printf 'services:\n  call-manager:\n    image: voipbin/bin-call-manager:__IMAGE_TAG__\n' > "$COMPOSE_FILE"
    run run_render "$COMPOSE_FILE" "$(printf 'a%.0s' {1..40})"
    [ "$status" -eq 0 ]
    run cat "$COMPOSE_FILE"
    [[ "$output" == *"voipbin/bin-call-manager:$(printf 'a%.0s' {1..40})"* ]]
    [[ "$output" != *"__IMAGE_TAG__"* ]]
}

@test "substitution: touches nothing else in the file" {
    printf 'services:\n  call-manager:\n    image: voipbin/bin-call-manager:__IMAGE_TAG__\nnetworks:\n  default:\n    name: install_default\n    external: true\n' > "$COMPOSE_FILE"
    run run_render "$COMPOSE_FILE" deadbeef
    [ "$status" -eq 0 ]
    run grep -c "install_default" "$COMPOSE_FILE"
    [ "$output" -eq 1 ]
    run grep -c "external: true" "$COMPOSE_FILE"
    [ "$output" -eq 1 ]
}
