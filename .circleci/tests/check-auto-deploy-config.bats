#!/usr/bin/env bats
# Tests for .circleci/scripts/check-auto-deploy-config.rb (VOIP-1586). The
# checker parses the real config; every negative case takes the real
# config_work.yml (or the real allowlist), applies ONE exact-text mutation, and
# expects the checker to fail with the matching message. bats 1.2.1 compatible.

load 'test_helper'

CHECKER="$SCRIPTS_DIR/check-auto-deploy-config.rb"
REAL_CONFIG="$CIRCLECI_DIR/config_work.yml"
REAL_LIST="$CIRCLECI_DIR/auto-deploy-services.txt"

setup() {
    setup_test_env
    CFG="$TEST_TEMP_DIR/config_work.yml"
    LST="$TEST_TEMP_DIR/auto-deploy-services.txt"
    cp "$REAL_CONFIG" "$CFG"
    cp "$REAL_LIST" "$LST"
}

teardown() {
    teardown_test_env
}

# mutate <file> <old> <new> : replace the FIRST occurrence of old; fails the
# test if old is absent (a mutation that does not apply proves nothing).
mutate() {
    OLD="$2" NEW="$3" ruby -e '
      f = ARGV[0]; s = File.read(f); o = ENV["OLD"]; n = ENV["NEW"]
      i = s.index(o) or abort("mutation target not found: #{o.inspect}")
      s[i, o.length] = n
      File.write(f, s)' "$1"
}

run_checker() {
    run ruby "$CHECKER" "$CFG" "$LST"
}

expect_fail() {
    [ "$status" -eq 1 ]
    [[ "$output" == *"$1"* ]]
}

@test "real config and real allowlist pass" {
    run ruby "$CHECKER"
    [ "$status" -eq 0 ]
    [[ "$output" == *"OK"* ]]
}

@test "a comment mentioning -test-auto and non_main_filters is ignored (positive)" {
    printf '\n# bin-fake-manager-test-auto uses non_main_filters\n' >> "$CFG"
    run_checker
    [ "$status" -eq 0 ]
}

# ---- (a) shape of an allowlisted workflow ----

@test "build-approval without a filter fails" {
    mutate "$CFG" $'type: approval\n          <<: *non_main_filters' 'type: approval'
    run_checker
    expect_fail "build-approval filters must be exactly ignore: main"
}

@test "build-approval with only: main instead of ignore fails" {
    mutate "$CFG" $'type: approval\n          <<: *non_main_filters' $'type: approval\n          filters:\n            branches:\n              only: main'
    run_checker
    expect_fail "build-approval filters must be exactly ignore: main"
}

@test "the gated test job without ignore main fails" {
    mutate "$CFG" $'bin-agent-manager-test:\n          <<: *non_main_filters\n' $'bin-agent-manager-test:\n'
    run_checker
    expect_fail "bin-agent-manager-test filters must be exactly ignore: main"
}

@test "the gated test job losing its build-approval requires fails" {
    mutate "$CFG" $'<<: *non_main_filters\n          requires:\n            - build-approval\n      - bin-agent-manager-test:\n          name: bin-agent-manager-test-auto' $'<<: *non_main_filters\n      - bin-agent-manager-test:\n          name: bin-agent-manager-test-auto'
    run_checker
    expect_fail "bin-agent-manager-test must require exactly build-approval"
}

@test "test-auto with a requires fails" {
    mutate "$CFG" $'name: bin-agent-manager-test-auto\n          <<: *deploy_filters\n' $'name: bin-agent-manager-test-auto\n          <<: *deploy_filters\n          requires:\n            - build-approval\n'
    run_checker
    expect_fail "bin-agent-manager-test-auto must not have requires"
}

@test "test-auto that is not only main fails" {
    mutate "$CFG" $'name: bin-agent-manager-test-auto\n          <<: *deploy_filters' $'name: bin-agent-manager-test-auto\n          <<: *non_main_filters'
    run_checker
    expect_fail "bin-agent-manager-test-auto filters must be exactly only: main"
}

@test "build not requiring test-auto fails" {
    mutate "$CFG" $'            - bin-agent-manager-test\n            - bin-agent-manager-test-auto\n' $'            - bin-agent-manager-test\n'
    run_checker
    expect_fail "bin-agent-manager-build must require exactly bin-agent-manager-test and bin-agent-manager-test-auto"
}

@test "build with an extra requires fails" {
    mutate "$CFG" $'            - bin-agent-manager-test\n            - bin-agent-manager-test-auto\n' $'            - bin-agent-manager-test\n            - bin-agent-manager-test-auto\n            - build-approval\n'
    run_checker
    expect_fail "bin-agent-manager-build must require exactly"
}

@test "deploy without serial-group fails" {
    mutate "$CFG" $'          serial-group: << pipeline.project.slug >>/deploy-bin-agent-manager\n' ''
    run_checker
    expect_fail "bin-agent-manager-deploy serial-group must be"
}

@test "deploy with another service's serial-group fails" {
    mutate "$CFG" 'deploy-bin-agent-manager' 'deploy-bin-contact-manager'
    run_checker
    expect_fail "bin-agent-manager-deploy serial-group must be"
}

@test "deploy with its own filters overriding the anchor fails" {
    mutate "$CFG" $'<<: [*context_production, *deploy_filters]\n          serial-group: << pipeline.project.slug >>/deploy-bin-agent-manager' $'<<: [*context_production, *deploy_filters]\n          filters:\n            branches:\n              only: /.*/\n          serial-group: << pipeline.project.slug >>/deploy-bin-agent-manager'
    run_checker
    expect_fail "bin-agent-manager-deploy filters must be exactly only: main"
}

@test "deploy requiring something other than build fails" {
    mutate "$CFG" $'serial-group: << pipeline.project.slug >>/deploy-bin-agent-manager\n          requires:\n            - bin-agent-manager-build' $'serial-group: << pipeline.project.slug >>/deploy-bin-agent-manager\n          requires:\n            - bin-agent-manager-test-auto'
    run_checker
    expect_fail "bin-agent-manager-deploy must require exactly bin-agent-manager-build"
}

@test "an extra job in an auto workflow fails" {
    mutate "$CFG" $'      - bin-agent-manager-deploy:\n          <<: [*context_production, *deploy_filters]\n          serial-group' $'      - bin-extra-job\n      - bin-agent-manager-deploy:\n          <<: [*context_production, *deploy_filters]\n          serial-group'
    run_checker
    expect_fail "jobs must be exactly"
}

@test "deploy command without the guard env fails" {
    mutate "$CFG" $'            DEPLOY_REQUIRE_LATEST_FOR_PATHS="bin-agent-manager" \\\n' ''
    run_checker
    expect_fail 'must set DEPLOY_REQUIRE_LATEST_FOR_PATHS="bin-agent-manager"'
}

@test "guard env with another service's path fails" {
    mutate "$CFG" 'DEPLOY_REQUIRE_LATEST_FOR_PATHS="bin-agent-manager"' 'DEPLOY_REQUIRE_LATEST_FOR_PATHS="bin-contact-manager"'
    run_checker
    expect_fail 'must set DEPLOY_REQUIRE_LATEST_FOR_PATHS="bin-agent-manager"'
}

@test "guard env set on a separate line (not exported to the deploy command) fails" {
    mutate "$CFG" $'DEPLOY_REQUIRE_LATEST_FOR_PATHS="bin-agent-manager" \\\n' $'DEPLOY_REQUIRE_LATEST_FOR_PATHS="bin-agent-manager"\n'
    run_checker
    expect_fail 'must set DEPLOY_REQUIRE_LATEST_FOR_PATHS="bin-agent-manager"'
}

@test "komodo-api-deploy.sh called with another service's stack name fails" {
    mutate "$CFG" 'komodo-api-deploy.sh bin-agent-manager bin-agent-manager/komodo' 'komodo-api-deploy.sh bin-contact-manager bin-agent-manager/komodo'
    run_checker
    expect_fail "first argument must be bin-agent-manager"
}

@test "deploy without the production context fails" {
    mutate "$CFG" $'<<: [*context_production, *deploy_filters]\n          serial-group: << pipeline.project.slug >>/deploy-bin-agent-manager' $'<<: *deploy_filters\n          serial-group: << pipeline.project.slug >>/deploy-bin-agent-manager'
    run_checker
    expect_fail "bin-agent-manager-deploy must use the production context"
}

@test "a second komodo-api-deploy.sh command in the deploy job fails" {
    mutate "$CFG" $'              .circleci/scripts/komodo-api-deploy.sh bin-agent-manager bin-agent-manager/komodo/docker-compose.yml\n' $'              .circleci/scripts/komodo-api-deploy.sh bin-agent-manager bin-agent-manager/komodo/docker-compose.yml\n      - run: .circleci/scripts/komodo-api-deploy.sh bin-agent-manager other.yml\n'
    run_checker
    expect_fail "must run komodo-api-deploy.sh exactly once"
}

@test "komodo-api-deploy.sh called with a stack name that only starts with the service name fails" {
    mutate "$CFG" 'komodo-api-deploy.sh bin-agent-manager bin-agent-manager/komodo' 'komodo-api-deploy.sh bin-agent-manager-x bin-agent-manager/komodo'
    run_checker
    expect_fail "first argument must be bin-agent-manager"
}

@test "the test-auto job built from a different job definition fails" {
    mutate "$CFG" $'bin-agent-manager-test:\n          name: bin-agent-manager-test-auto' $'bin-agent-manager-other:\n          name: bin-agent-manager-test-auto'
    run_checker
    expect_fail "bin-agent-manager-test-auto key must be bin-agent-manager-test"
}

@test "an allowlisted workflow without a jobs list fails" {
    mutate "$CFG" $'  bin-storage-manager:\n    when: << pipeline.parameters.run-bin-storage-manager >>\n    jobs:' $'  bin-storage-manager:\n    when: << pipeline.parameters.run-bin-storage-manager >>\n    nojobs:'
    run_checker
    expect_fail "bin-storage-manager is in the allowlist but has no jobs list"
}

@test "guard env overridden to empty later on the same command fails" {
    mutate "$CFG" $'DEPLOY_REQUIRE_LATEST_FOR_PATHS="bin-agent-manager" \\\n' $'DEPLOY_REQUIRE_LATEST_FOR_PATHS="bin-agent-manager" DEPLOY_REQUIRE_LATEST_FOR_PATHS="" \\\n'
    run_checker
    expect_fail "exactly once on the same command"
}

@test "guard env stripped by an env -u prefix fails" {
    mutate "$CFG" $'DEPLOY_REQUIRE_LATEST_FOR_PATHS="bin-agent-manager" \\\n            CC_KOMODO_POLL_ATTEMPTS=60 CC_KOMODO_POLL_INTERVAL_S=3 \\\n              .circleci/scripts/komodo-api-deploy.sh bin-agent-manager' $'DEPLOY_REQUIRE_LATEST_FOR_PATHS="bin-agent-manager" \\\n            CC_KOMODO_POLL_ATTEMPTS=60 CC_KOMODO_POLL_INTERVAL_S=3 \\\n              env -u DEPLOY_REQUIRE_LATEST_FOR_PATHS .circleci/scripts/komodo-api-deploy.sh bin-agent-manager'
    run_checker
    expect_fail "must be VAR=value assignments followed by komodo-api-deploy.sh"
}

@test "guard env followed by a true; no-op that detaches the assignment fails" {
    mutate "$CFG" $'DEPLOY_REQUIRE_LATEST_FOR_PATHS="bin-agent-manager" \\\n            CC_KOMODO_POLL_ATTEMPTS=60 CC_KOMODO_POLL_INTERVAL_S=3 \\\n              .circleci/scripts/komodo-api-deploy.sh bin-agent-manager' $'DEPLOY_REQUIRE_LATEST_FOR_PATHS="bin-agent-manager" true; \\\n            CC_KOMODO_POLL_ATTEMPTS=60 CC_KOMODO_POLL_INTERVAL_S=3 \\\n              .circleci/scripts/komodo-api-deploy.sh bin-agent-manager'
    run_checker
    expect_fail "must be VAR=value assignments followed by komodo-api-deploy.sh"
}

@test "the deploy command commented out fails" {
    mutate "$CFG" $'              .circleci/scripts/komodo-api-deploy.sh bin-agent-manager bin-agent-manager/komodo' $'              # .circleci/scripts/komodo-api-deploy.sh bin-agent-manager bin-agent-manager/komodo'
    run_checker
    [ "$status" -eq 1 ]
}

@test "an approval-kept deploy renamed so it no longer ends in -deploy is still checked" {
    mutate "$CFG" $'      - bin-call-manager-deploy:\n          <<: [*context_production, *deploy_filters]' $'      - bin-call-manager-deploy:\n          name: bin-call-manager-ship\n          <<: [*context_production, *deploy_filters]'
    mutate "$CFG" $'      - bin-call-manager-build:\n          <<: *context_production\n          requires:\n            - bin-call-manager-test' $'      - bin-call-manager-build:\n          <<: *context_production'
    run_checker
    expect_fail "bin-call-manager-ship is not gated by build-approval"
}

@test "guard env hidden inside another variable's value fails" {
    mutate "$CFG" $'DEPLOY_REQUIRE_LATEST_FOR_PATHS="bin-agent-manager" \\\n' $'FOO=a-DEPLOY_REQUIRE_LATEST_FOR_PATHS=bin-agent-manager \\\n'
    run_checker
    expect_fail "exactly once on the same command"
}

@test "an auto deploy job whose workflow key names another service's job fails" {
    mutate "$CFG" $'      - bin-agent-manager-deploy:\n          <<: [*context_production, *deploy_filters]\n          serial-group' $'      - bin-call-manager-deploy:\n          name: bin-agent-manager-deploy\n          <<: [*context_production, *deploy_filters]\n          serial-group'
    run_checker
    expect_fail "bin-agent-manager-deploy key must be bin-agent-manager-deploy"
}

# ---- (c) anchors ----

@test "deploy_filters anchor widened beyond main fails" {
    mutate "$CFG" $'      only: main\n' $'      only: [main, /.*/]\n'
    run_checker
    expect_fail "deploy_filters anchor is not exactly only: main"
}

@test "non_main_filters anchor tampered fails" {
    mutate "$CFG" $'      ignore: main\n' $'      ignore: [main, dev]\n'
    run_checker
    expect_fail "non_main_filters anchor is not exactly ignore: main"
}

# ---- (d) allowlist / config consistency (both directions) ----

@test "duplicate allowlist entry fails" {
    echo "bin-agent-manager" >> "$LST"
    run_checker
    expect_fail "allowlist has a duplicate entry: bin-agent-manager"
}

@test "converted workflow missing from the allowlist fails" {
    mutate "$LST" $'bin-agent-manager\n' ''
    run_checker
    expect_fail "workflow bin-agent-manager has a -test-auto job but is not in the allowlist"
}

@test "allowlist entry without a workflow in config fails" {
    echo "bin-nope-manager" >> "$LST"
    run_checker
    expect_fail "allowlist entry has no workflow in config: bin-nope-manager"
}

@test "allowlist entry whose workflow is not converted fails" {
    echo "bin-call-manager" >> "$LST"
    run_checker
    expect_fail "workflow bin-call-manager: jobs must be exactly"
}

# ---- (f) deny list ----

@test "every deny-listed service is rejected from the allowlist" {
    for svc in bin-timeline-manager bin-rag-manager bin-trigger-sender voip-asterisk-proxy voip-rtpengine-proxy voip-kamailio-proxy; do
        cp "$REAL_LIST" "$LST"
        echo "$svc" >> "$LST"
        run_checker
        expect_fail "allowlist entry is not allowed to be auto-deployed: $svc"
    done
}

# ---- (b)/(e) approval-kept workflows stay approval-kept ----

@test "a -test-auto job added to a workflow outside the allowlist fails" {
    mutate "$CFG" $'      - bin-call-manager-test:\n          requires:' $'      - bin-call-manager-test:\n          name: bin-call-manager-test-auto\n          requires:'
    run_checker
    expect_fail "workflow bin-call-manager has a -test-auto job but is not in the allowlist"
}

@test "an ignore-main filter added to an approval-kept workflow fails" {
    mutate "$CFG" $'  bin-call-manager:\n    when: << pipeline.parameters.run-bin-call-manager >>\n    jobs:\n      - build-approval:\n          type: approval\n' $'  bin-call-manager:\n    when: << pipeline.parameters.run-bin-call-manager >>\n    jobs:\n      - build-approval:\n          type: approval\n          <<: *non_main_filters\n'
    run_checker
    expect_fail "job build-approval has an ignore branch filter but bin-call-manager is not auto-deploy"
}

@test "build-approval removed from an approval-kept workflow fails" {
    mutate "$CFG" $'  voip-kamailio-proxy:\n    when:\n      or: [<< pipeline.parameters.run-voip-kamailio-proxy >>]\n    jobs:\n      - build-approval:\n          type: approval\n      - voip-kamailio-proxy-test:\n          requires:\n            - build-approval\n' $'  voip-kamailio-proxy:\n    when:\n      or: [<< pipeline.parameters.run-voip-kamailio-proxy >>]\n    jobs:\n      - voip-kamailio-proxy-test\n'
    run_checker
    expect_fail "workflow voip-kamailio-proxy: approval-kept workflow needs an unfiltered build-approval"
}

@test "an approval-kept deploy cut off from build-approval fails" {
    mutate "$CFG" $'      - bin-call-manager-build:\n          <<: *context_production\n          requires:\n            - bin-call-manager-test' $'      - bin-call-manager-build:\n          <<: *context_production'
    run_checker
    expect_fail "bin-call-manager-deploy is not gated by build-approval"
}

@test "an unreadable config fails closed" {
    printf 'workflows: [unclosed\n' > "$CFG"
    run_checker
    [ "$status" -eq 1 ]
    [[ "$output" == *"cannot check"* ]]
}
