#!/usr/bin/env bats
#
# Pins the voip-kamailio-proxy Komodo stack definition and its CI wiring
# (VOIP-1486). See voip-kamailio-proxy/docs/plans/2026-09-07-kamailio-proxy-
# komodo-stack-design.md.
#
# CONSTRAINT: the shell-tests job installs only bats and mawk
# (config_work.yml:2176-2183), so there is no PyYAML and no yq here. Every
# assertion below is grep/awk over the raw text. Assertions anchor on YAML key
# shapes rather than bare substrings, because both files carry comments that
# mention the very names some assertions require to be ABSENT.
#
# KNOWN TRIGGER GAP: shell-tests is armed only by .circleci/scripts/.*,
# .circleci/tests/.*, docs/reference/extractor.sh and docs/reference/tests/.*
# (config.yml:59-62). Neither voip-kamailio-proxy/.* nor .circleci/config_work.yml
# is on that list, so editing the compose file or the deploy wiring does not
# re-run this suite; only editing a file under .circleci/tests/ does.
#
# Partial mitigation, and only for the compose file: the single highest-value
# property, the __IMAGE_TAG__ placeholder, is independently enforced at deploy
# time by render-image-tag.sh, which exits 1 when it is missing. The two
# CI-wiring assertions at the end have no such backstop - they are a guard for
# a reviewer to run, not a gate CI will apply on their behalf.

load test_helper

REPO_ROOT="$(dirname "$CIRCLECI_DIR")"
COMPOSE="voip-kamailio-proxy/komodo/docker-compose.yml"
CONFIG_WORK=".circleci/config_work.yml"

setup() {
    cd "$REPO_ROOT"
}

# bats aborts a test body on the first failed assertion, so an inline
# teardown_test_env would leak the mktemp -d whenever a test actually fails.
teardown() {
    teardown_test_env
}

@test "compose: file exists and declares exactly one service, kamailio-proxy" {
    [ -f "$COMPOSE" ]
    [ "$(grep -c '^  kamailio-proxy:$' "$COMPOSE")" -eq 1 ]
    # Any second service key under services: would break the single-target
    # assumption the Prometheus scrape job and the replica count rest on.
    run bash -c "awk '/^services:/{f=1;next} /^networks:/{f=0} f' '$COMPOSE' | grep -cE '^  [a-z][a-z0-9-]*:\$'"
    [ "$output" -eq 1 ]
}

@test "compose: image carries the __IMAGE_TAG__ placeholder, not a literal tag" {
    [ "$(grep -cF 'image: voipbin/voip-kamailio-proxy:__IMAGE_TAG__' "$COMPOSE")" -eq 1 ]
    run grep -cE 'image: voipbin/voip-kamailio-proxy:[0-9a-f]{7,}' "$COMPOSE"
    [ "$output" -eq 0 ]
}

@test "compose: render-image-tag.sh substitutes the placeholder" {
    setup_test_env
    local copy="$TEST_TEMP_DIR/docker-compose.yml"
    cp "$COMPOSE" "$copy"

    run "$SCRIPTS_DIR/render-image-tag.sh" "$copy" deadbeefcafe1234
    [ "$status" -eq 0 ]
    [ "$(grep -c '__IMAGE_TAG__' "$copy")" -eq 0 ]
    [ "$(grep -c ':deadbeefcafe1234' "$copy")" -eq 1 ]
}

@test "compose: restart always, two replicas, no container_name" {
    # restart: always (not on-failure) because a failed getKamailioID exits 0.
    [ "$(grep -c '^    restart: always$' "$COMPOSE")" -eq 1 ]
    [ "$(grep -c '^      replicas: 2$' "$COMPOSE")" -eq 1 ]
    # Compose rejects container_name together with replicas > 1.
    run grep -c '^[[:space:]]*container_name:' "$COMPOSE"
    [ "$output" -eq 0 ]
}

@test "compose: environment sets exactly the four intended variables" {
    # Count first, so a fifth variable cannot slip in alongside the four
    # asserted below.
    run bash -c "awk '/^    environment:/{f=1;next} /^    [a-z]/{f=0} f' '$COMPOSE' | grep -cE '^      - [A-Z_]+='"
    [ "$output" -eq 4 ]

    [ "$(grep -cE '^[[:space:]]*-[[:space:]]*RABBITMQ_ADDRESS=\[\[BIN_MANAGER__RABBITMQ_ADDRESS\]\]$' "$COMPOSE")" -eq 1 ]
    [ "$(grep -cE '^[[:space:]]*-[[:space:]]*PROMETHEUS_LISTEN_ADDRESS=:2112$' "$COMPOSE")" -eq 1 ]
    [ "$(grep -cE '^[[:space:]]*-[[:space:]]*PROMETHEUS_ENDPOINT=/metrics$' "$COMPOSE")" -eq 1 ]
    [ "$(grep -cE '^[[:space:]]*-[[:space:]]*SIP_TIMEOUT=' "$COMPOSE")" -eq 1 ]

    # Deliberately omitted: their defaults are already correct, and writing
    # them is the only way to get them wrong. Pinned in Go by
    # cmd/kamailio-proxy/main_test.go Test_defaults. The env-entry anchor
    # matters: both names appear in this file's comments.
    run grep -cE '^[[:space:]]*-[[:space:]]*RABBITMQ_QUEUE_LISTEN=' "$COMPOSE"
    [ "$output" -eq 0 ]
    run grep -cE '^[[:space:]]*-[[:space:]]*INTERFACE_NAME=' "$COMPOSE"
    [ "$output" -eq 0 ]
}

@test "compose: no cap_add, depends_on, or resource limits" {
    # The SIP OPTIONS probe uses net.DialUDP, not a raw socket, so the
    # Ansible definition's NET_RAW is not needed.
    for key in cap_add depends_on mem_limit cpus; do
        run grep -cE "^[[:space:]]*${key}:" "$COMPOSE"
        [ "$output" -eq 0 ]
    done
}

@test "compose: attaches to the external production network" {
    [ "$(grep -c '^    name: production$' "$COMPOSE")" -eq 1 ]
    [ "$(grep -c '^    external: true$' "$COMPOSE")" -eq 1 ]
}

@test "ci: deploy job exists and calls both deployment scripts on this compose file" {
    [ "$(grep -c '^  voip-kamailio-proxy-deploy:$' "$CONFIG_WORK")" -eq 1 ]
    [ "$(grep -cF "render-image-tag.sh voip-kamailio-proxy/komodo/docker-compose.yml" "$CONFIG_WORK")" -eq 1 ]
    [ "$(grep -cF "komodo-api-deploy.sh voip-kamailio-proxy voip-kamailio-proxy/komodo/docker-compose.yml" "$CONFIG_WORK")" -eq 1 ]
}

@test "ci: deploy workflow entry has the production context and requires the build job" {
    # Range ends at the next workflow-entry bullet OR at any dedent. Ending on
    # the bullet alone is not enough: this entry is the last one in its
    # workflow, so the range would run on into the following workflow block.
    local entry
    entry="$(awk '/^      - voip-kamailio-proxy-deploy:$/{f=1;next} /^      - /||/^  [^ ]/{f=0} f' "$CONFIG_WORK")"

    # Without the production context the Komodo credentials are not injected.
    [ "$(printf '%s\n' "$entry" | grep -c '<<: \*context_production')" -eq 1 ]

    # Positive assertion, deliberately. Asserting "no mismatching line" would
    # pass silently against the flow-sequence form
    # (requires: [voip-kamailio-proxy-build]), because then the standalone
    # requires: line disappears and there is nothing left to mismatch.
    [ "$(printf '%s\n' "$entry" | grep -A1 '^[[:space:]]*requires:$' | grep -cE '^[[:space:]]*- voip-kamailio-proxy-build$')" -eq 1 ]
}
