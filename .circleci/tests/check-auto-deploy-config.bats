#!/usr/bin/env bats
# Tests for .circleci/scripts/check-auto-deploy-config.rb (VOIP-1586). The
# checker parses the real config; every negative case takes the real
# config_work.yml (or the real allowlist), applies ONE exact-text mutation, and
# expects the checker to fail with the matching message. bats 1.2.1 compatible.

load 'test_helper'

CHECKER="$SCRIPTS_DIR/check-auto-deploy-config.rb"
REAL_CONFIG="$CIRCLECI_DIR/config_work.yml"
REAL_LIST="$CIRCLECI_DIR/auto-deploy-services.txt"
REAL_RUN_LIST="$CIRCLECI_DIR/auto-run-services.txt"

setup() {
    setup_test_env
    CFG="$TEST_TEMP_DIR/config_work.yml"
    LST="$TEST_TEMP_DIR/auto-deploy-services.txt"
    cp "$REAL_CONFIG" "$CFG"
    cp "$REAL_LIST" "$LST"
    RLST="$TEST_TEMP_DIR/auto-run-services.txt"
    cp "$REAL_RUN_LIST" "$RLST"
    # The checker also reads the setup config (path-filtering mapping) and checks
    # that the directories named there exist: give it a copy and an empty tree.
    SETUP="$TEST_TEMP_DIR/setup-config.yml"
    ROOT="$TEST_TEMP_DIR/root"
    cp "$CIRCLECI_DIR/config.yml" "$SETUP"
    mkdir -p "$ROOT"
    for d in $(awk '$2 ~ /^run-bin-/ {sub(/\/\.\*$/, "", $1); print $1}' "$SETUP"); do
        mkdir -p "$ROOT/$d"
    done
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
    run ruby "$CHECKER" "$CFG" "$LST" "$SETUP" "$ROOT" "$RLST"
}

# add_keep_fixture : insert an approval-kept service workflow (and its deploy job
# definition) named bin-fixture-manager, taken from tests/fixtures. No real
# service stays approval-kept in the standard shape, so every approval-kept rule
# is exercised on this fixture. bin-fixture-manager is not in the checker's deny
# list, so a case is judged on the rule under test alone.
add_keep_fixture() {
    ruby -e '
      f = ARGV[0]; s = File.read(f)
      wf = File.read(ARGV[1]); jb = File.read(ARGV[2])
      s.sub!("\nworkflows:\n", "\nworkflows:\n" + wf) or abort("workflows key not found")
      s.sub!("\njobs:\n", "\njobs:\n" + jb) or abort("jobs key not found")
      File.write(f, s)' "$CFG" "$TESTS_DIR/fixtures/keep-workflow.yml" "$TESTS_DIR/fixtures/keep-deploy-job.yml"
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

@test "the gated test job without ignore main fails" {
    mutate "$CFG" $'bin-agent-manager-test:\n          <<: *non_main_filters\n' $'bin-agent-manager-test:\n'
    run_checker
    expect_fail "bin-agent-manager-test filters must be exactly ignore: main"
}

@test "a build-approval job restored in an auto-deploy workflow fails" {
    mutate "$CFG" $'  bin-agent-manager:\n    when: << pipeline.parameters.run-bin-agent-manager >>\n    jobs:\n' $'  bin-agent-manager:\n    when: << pipeline.parameters.run-bin-agent-manager >>\n    jobs:\n      - build-approval:\n          type: approval\n          <<: *non_main_filters\n'
    run_checker
    expect_fail "workflow bin-agent-manager: build-approval must not exist in an auto-deploy workflow"
}

@test "an approval job under another name in an auto-deploy workflow fails" {
    mutate "$CFG" $'  bin-agent-manager:\n    when: << pipeline.parameters.run-bin-agent-manager >>\n    jobs:\n' $'  bin-agent-manager:\n    when: << pipeline.parameters.run-bin-agent-manager >>\n    jobs:\n      - hold-gate:\n          type: approval\n          <<: *non_main_filters\n'
    run_checker
    expect_fail "workflow bin-agent-manager: jobs must be exactly"
    [[ "$output" == *"job hold-gate must not have a type"* ]]
}

@test "type: approval attached to the auto-deploy test job fails" {
    mutate "$CFG" $'      - bin-agent-manager-test:\n          <<: *non_main_filters\n' $'      - bin-agent-manager-test:\n          type: approval\n          <<: *non_main_filters\n'
    run_checker
    expect_fail "workflow bin-agent-manager: job bin-agent-manager-test must not have a type"
}

@test "a requires restored on the auto-deploy test job fails" {
    mutate "$CFG" $'      - bin-agent-manager-test:\n          <<: *non_main_filters\n' $'      - bin-agent-manager-test:\n          <<: *non_main_filters\n          requires:\n            - bin-agent-manager-deploy\n'
    run_checker
    expect_fail "bin-agent-manager-test must not have requires"
}

@test "an empty requires on the auto-deploy test job fails" {
    mutate "$CFG" $'      - bin-agent-manager-test:\n          <<: *non_main_filters\n' $'      - bin-agent-manager-test:\n          <<: *non_main_filters\n          requires: []\n'
    run_checker
    expect_fail "bin-agent-manager-test must not have requires"
}

@test "test-auto with a requires fails" {
    mutate "$CFG" $'name: bin-agent-manager-test-auto\n          <<: *deploy_filters\n' $'name: bin-agent-manager-test-auto\n          <<: *deploy_filters\n          requires:\n            - bin-agent-manager-test\n'
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
    mutate "$CFG" $'            - bin-agent-manager-test\n            - bin-agent-manager-test-auto\n' $'            - bin-agent-manager-test\n            - bin-agent-manager-test-auto\n            - bin-agent-manager-deploy\n'
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
    add_keep_fixture
    mutate "$CFG" $'      - bin-fixture-manager-deploy:\n          <<: [*context_production, *deploy_filters]' $'      - bin-fixture-manager-deploy:\n          name: bin-fixture-manager-ship\n          <<: [*context_production, *deploy_filters]'
    mutate "$CFG" $'      - bin-fixture-manager-build:\n          <<: *context_production\n          requires:\n            - bin-fixture-manager-test' $'      - bin-fixture-manager-build:\n          <<: *context_production'
    run_checker
    expect_fail "bin-fixture-manager-ship is not gated by build-approval"
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

@test "an only-filter on the test job of an approval-kept workflow fails" {
    add_keep_fixture
    mutate "$CFG" $'      - bin-fixture-manager-test:\n          requires:' $'      - bin-fixture-manager-test:\n          filters:\n            branches:\n              only: /^feature.*/\n          requires:'
    run_checker
    expect_fail "job bin-fixture-manager-test has a branch filter but bin-fixture-manager is not auto-deploy"
}

@test "an approval-kept deploy with a widened filter fails" {
    add_keep_fixture
    mutate "$CFG" $'      - bin-fixture-manager-deploy:\n          <<: [*context_production, *deploy_filters]' $'      - bin-fixture-manager-deploy:\n          <<: [*context_production, *deploy_filters]\n          filters:\n            branches:\n              only: /.*/'
    run_checker
    expect_fail "deploy job bin-fixture-manager-deploy has a branch filter other than only: main"
}

@test "an approval-kept deploy whose key and name no longer end in -deploy is found by its job definition" {
    add_keep_fixture
    mutate "$CFG" $'      - bin-fixture-manager-deploy:\n          <<: [*context_production, *deploy_filters]' $'      - bin-fixture-manager-ship:\n          <<: [*context_production, *deploy_filters]'
    mutate "$CFG" $'  bin-fixture-manager-deploy:\n    docker:' $'  bin-fixture-manager-ship:\n    docker:'
    mutate "$CFG" $'      - bin-fixture-manager-build:\n          <<: *context_production\n          requires:\n            - bin-fixture-manager-test' $'      - bin-fixture-manager-build:\n          <<: *context_production'
    run_checker
    expect_fail "bin-fixture-manager-ship is not gated by build-approval"
}

@test "a comment line that mentions komodo-api-deploy.sh inside the deploy command is ignored (positive)" {
    mutate "$CFG" $'            DEPLOY_REQUIRE_LATEST_FOR_PATHS="bin-agent-manager" \\\n' $'            # see komodo-api-deploy.sh for the guard\n            DEPLOY_REQUIRE_LATEST_FOR_PATHS="bin-agent-manager" \\\n'
    run_checker
    [ "$status" -eq 0 ]
}

# ---- (g) guard paths derived from the workflow's `when` (VOIP-1592) ----

@test "bin-api-manager guard covers bin-openapi-manager (positive)" {
    grep -q 'DEPLOY_REQUIRE_LATEST_FOR_PATHS="bin-api-manager bin-openapi-manager"' "$CFG"
    run_checker
    [ "$status" -eq 0 ]
}

@test "bin-api-manager guard missing the openapi path fails" {
    mutate "$CFG" 'DEPLOY_REQUIRE_LATEST_FOR_PATHS="bin-api-manager bin-openapi-manager"' 'DEPLOY_REQUIRE_LATEST_FOR_PATHS="bin-api-manager"'
    run_checker
    expect_fail 'must set DEPLOY_REQUIRE_LATEST_FOR_PATHS="bin-api-manager bin-openapi-manager"'
}

@test "extra guard path on a service whose when has no extra parameter fails" {
    mutate "$CFG" 'DEPLOY_REQUIRE_LATEST_FOR_PATHS="bin-agent-manager"' 'DEPLOY_REQUIRE_LATEST_FOR_PATHS="bin-agent-manager bin-openapi-manager"'
    run_checker
    expect_fail 'must set DEPLOY_REQUIRE_LATEST_FOR_PATHS="bin-agent-manager" exactly once'
}

@test "guard paths in another order fail" {
    mutate "$CFG" 'DEPLOY_REQUIRE_LATEST_FOR_PATHS="bin-api-manager bin-openapi-manager"' 'DEPLOY_REQUIRE_LATEST_FOR_PATHS="bin-openapi-manager bin-api-manager"'
    run_checker
    expect_fail 'must set DEPLOY_REQUIRE_LATEST_FOR_PATHS="bin-api-manager bin-openapi-manager"'
}

@test "a when parameter that is not a mapped service directory fails" {
    mutate "$CFG" '[<< pipeline.parameters.run-bin-api-manager >>, << pipeline.parameters.run-bin-openapi-manager >>]' '[<< pipeline.parameters.run-bin-api-manager >>, << pipeline.parameters.run-bin-nope-manager >>]'
    run_checker
    expect_fail 'when references run-bin-nope-manager that is not a mapped service directory'
}

@test "a when parameter whose mapping line points at another parameter fails" {
    mutate "$SETUP" 'bin-openapi-manager/.*      run-bin-openapi-manager true' 'bin-openapi-manager/.*      run-bin-api-manager true'
    run_checker
    expect_fail 'when references run-bin-openapi-manager that is not a mapped service directory'
}

@test "a when parameter whose directory does not exist fails" {
    rmdir "$ROOT/bin-openapi-manager"
    run_checker
    expect_fail 'when references run-bin-openapi-manager that is not a mapped service directory'
}

@test "a when without the service's own parameter fails" {
    mutate "$CFG" $'  bin-agent-manager:\n    when: << pipeline.parameters.run-bin-agent-manager >>' $'  bin-agent-manager:\n    when: << pipeline.parameters.run-bin-contact-manager >>'
    run_checker
    expect_fail 'workflow bin-agent-manager: when must contain its own parameter run-bin-agent-manager'
}

@test "an unsupported when form fails" {
    mutate "$CFG" $'  bin-agent-manager:\n    when: << pipeline.parameters.run-bin-agent-manager >>' $'  bin-agent-manager:\n    when:\n      and: [<< pipeline.parameters.run-bin-agent-manager >>, true]'
    run_checker
    expect_fail 'workflow bin-agent-manager: unsupported when form'
}

@test "a when parameter that is not a run-bin parameter fails even if mapped and present" {
    grep -q 'voip-asterisk-proxy/.\* *run-voip-asterisk-proxy true' "$SETUP"
    mkdir -p "$ROOT/voip-asterisk-proxy"
    mutate "$CFG" 'run-bin-openapi-manager >>]' 'run-voip-asterisk-proxy >>]'
    run_checker
    expect_fail "when references run-voip-asterisk-proxy that is not a mapped service directory"
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
    add_keep_fixture
    echo "bin-fixture-manager" >> "$LST"
    run_checker
    expect_fail "workflow bin-fixture-manager: jobs must be exactly"
    [[ "$output" != *"not allowed to be auto-deployed"* ]]
}

# ---- (f) deny list ----

@test "every deny-listed service is rejected from the allowlist" {
    for svc in bin-trigger-sender voip-asterisk-proxy voip-rtpengine-proxy voip-kamailio-proxy; do
        cp "$REAL_LIST" "$LST"
        echo "$svc" >> "$LST"
        run_checker
        expect_fail "allowlist entry is not allowed to be auto-deployed: $svc"
    done
}

@test "the migrating services are auto-deploy and a missing allowlist entry for either fails (VOIP-1595)" {
    for svc in bin-timeline-manager bin-rag-manager; do
        grep -qx "$svc" "$REAL_LIST"
        grep -vx "$svc" "$REAL_LIST" > "$LST"
        run_checker
        expect_fail "workflow $svc has a -test-auto job but is not in the allowlist"
    done
}

@test "the guard line removed from the timeline or rag deploy command fails (VOIP-1595)" {
    mutate "$CFG" $'            DEPLOY_REQUIRE_LATEST_FOR_PATHS="bin-timeline-manager" \\\n' ''
    run_checker
    [ "$status" -eq 1 ]
    cp "$REAL_CONFIG" "$CFG"
    mutate "$CFG" $'            DEPLOY_REQUIRE_LATEST_FOR_PATHS="bin-rag-manager" \\\n' ''
    run_checker
    [ "$status" -eq 1 ]
}

# ---- (g) auto-run workflows: verify-only, run on main without approval (VOIP-1597) ----

@test "an auto-run workflow dropped from the auto-run list fails" {
    grep -vx "bin-common-handler" "$REAL_RUN_LIST" > "$RLST"
    run_checker
    expect_fail "workflow bin-common-handler has a -test-auto job but is not in the allowlist"
    cp "$REAL_RUN_LIST" "$RLST"
    grep -vx "bin-openapi-manager" "$REAL_RUN_LIST" > "$RLST"
    run_checker
    expect_fail "workflow bin-openapi-manager has a -auto job but is not in the allowlist"
}

@test "a duplicate auto-run entry fails" {
    echo "bin-common-handler" >> "$RLST"
    run_checker
    expect_fail "auto-run list has a duplicate entry: bin-common-handler"
}

@test "an auto-run entry that is also an auto-deploy entry fails" {
    echo "bin-common-handler" >> "$LST"
    run_checker
    expect_fail "auto-run list entry is also in the auto-deploy allowlist: bin-common-handler"
}

@test "a deny-listed service in the auto-run list fails" {
    echo "voip-kamailio-proxy" >> "$RLST"
    run_checker
    expect_fail "auto-run list entry is not allowed to be auto-run: voip-kamailio-proxy"
}

@test "an auto-run entry without a workflow fails" {
    echo "bin-nope-manager" >> "$RLST"
    run_checker
    expect_fail "auto-run list entry has no workflow in config: bin-nope-manager"
}

@test "a missing auto-run list file fails closed" {
    rm "$RLST"
    run_checker
    [ "$status" -eq 1 ]
}

@test "an auto-run auto job without the only-main filter fails" {
    mutate "$CFG" $'          name: bin-common-handler-test-auto\n          <<: *deploy_filters' $'          name: bin-common-handler-test-auto'
    run_checker
    expect_fail "bin-common-handler-test-auto filters must be exactly only: main"
}

@test "a build-approval job restored in an auto-run workflow fails" {
    mutate "$CFG" $'  bin-openapi-manager:\n    when: << pipeline.parameters.run-bin-openapi-manager >>\n    jobs:\n' $'  bin-openapi-manager:\n    when: << pipeline.parameters.run-bin-openapi-manager >>\n    jobs:\n      - build-approval:\n          type: approval\n          <<: *non_main_filters\n'
    run_checker
    expect_fail "workflow bin-openapi-manager: build-approval must not exist in an auto-run workflow"
}

@test "type: approval attached to the auto-run verify job fails" {
    mutate "$CFG" $'      - bin-openapi-manager-validate:\n          <<: *non_main_filters\n' $'      - bin-openapi-manager-validate:\n          type: approval\n          <<: *non_main_filters\n'
    run_checker
    expect_fail "workflow bin-openapi-manager: job bin-openapi-manager-validate must not have a type"
}

@test "a requires restored on the auto-run verify job fails" {
    mutate "$CFG" $'      - bin-openapi-manager-validate:\n          <<: *non_main_filters\n' $'      - bin-openapi-manager-validate:\n          <<: *non_main_filters\n          requires:\n            - bin-openapi-manager-validate-auto\n'
    run_checker
    expect_fail "bin-openapi-manager-validate must not have requires"
}

@test "an empty requires on the auto-run verify job fails" {
    mutate "$CFG" $'      - bin-openapi-manager-validate:\n          <<: *non_main_filters\n' $'      - bin-openapi-manager-validate:\n          <<: *non_main_filters\n          requires: []\n'
    run_checker
    expect_fail "bin-openapi-manager-validate must not have requires"
}

@test "an auto-run workflow with a duplicated verify job fails" {
    mutate "$CFG" $'      - bin-openapi-manager-validate:\n          <<: *non_main_filters\n' $'      - bin-openapi-manager-validate:\n          <<: *non_main_filters\n      - bin-openapi-manager-validate:\n          <<: *non_main_filters\n'
    run_checker
    expect_fail "workflow bin-openapi-manager: jobs must be exactly one verify job and its -auto copy"
}

@test "an auto-run auto job with requires fails" {
    mutate "$CFG" $'          name: bin-openapi-manager-validate-auto\n          <<: *deploy_filters' $'          name: bin-openapi-manager-validate-auto\n          <<: *deploy_filters\n          requires:\n            - bin-openapi-manager-validate'
    run_checker
    expect_fail "bin-openapi-manager-validate-auto must not have requires"
}

@test "an auto-run workflow with an extra job fails" {
    mutate "$CFG" $'          name: bin-common-handler-test-auto\n          <<: *deploy_filters' $'          name: bin-common-handler-test-auto\n          <<: *deploy_filters\n      - bin-common-handler-build:\n          <<: *context_production\n          requires:\n            - bin-common-handler-test-auto'
    run_checker
    expect_fail "workflow bin-common-handler: jobs must be exactly one verify job and its -auto copy"
}

@test "an auto-run verify job renamed to something else fails" {
    mutate "$CFG" $'      - bin-common-handler-test:\n          <<: *non_main_filters' $'      - bin-common-handler-test:\n          name: bin-common-handler-check\n          <<: *non_main_filters'
    run_checker
    [ "$status" -eq 1 ]
}

@test "a production context on an auto-run job fails" {
    mutate "$CFG" $'          name: bin-common-handler-test-auto\n          <<: *deploy_filters' $'          name: bin-common-handler-test-auto\n          <<: [*deploy_filters, *context_production]'
    run_checker
    expect_fail "job bin-common-handler-test-auto must not use a context"
}

@test "a docker push in an auto-run job definition fails" {
    mutate "$CFG" $'            cd bin-openapi-manager\n            go mod download' $'            cd bin-openapi-manager\n            docker push example/image:latest\n            go mod download'
    run_checker
    expect_fail "job definition bin-openapi-manager-validate must not run a deploy, a docker push, login or build, or gcloud"
}

@test "a docker-build command step in an auto-run job fails" {
    mutate "$CFG" $'            cd bin-openapi-manager\n            go mod download' $'            cd bin-openapi-manager\n            go mod download\n      - docker-build:\n          source-directory: x'
    run_checker
    expect_fail "uses a step that is not allowed in a verify-only job: docker-build"
}

@test "a setup_remote_docker step in an auto-run job fails" {
    mutate "$CFG" $'            cd bin-openapi-manager\n            go mod download' $'            cd bin-openapi-manager\n            go mod download\n      - setup_remote_docker'
    run_checker
    expect_fail "uses a step that is not allowed in a verify-only job: setup_remote_docker"
}

@test "kubectl or docker image push in an auto-run job fails" {
    mutate "$CFG" $'            cd bin-openapi-manager\n            go mod download' $'            cd bin-openapi-manager\n            kubectl apply -f x\n            go mod download'
    run_checker
    expect_fail "must not run a deploy, a docker push"
}

@test "a docker push with options in an auto-run job fails" {
    mutate "$CFG" $'            cd bin-openapi-manager\n            go mod download' $'            cd bin-openapi-manager\n            docker --debug push x\n            go mod download'
    run_checker
    expect_fail "must not run a deploy, a docker push"
}

@test "a push inside the go-test command fails for an auto-run workflow" {
    mutate "$CFG" $'                  GCL_VERSION=2.14.0' $'                  docker push x\n                  GCL_VERSION=2.14.0'
    run_checker
    expect_fail "command go-test is missing or runs a deploy"
}

@test "an auto-run verify job with an unexpected name fails" {
    mutate "$CFG" $'      - bin-common-handler-test:\n          <<: *non_main_filters' $'      - bin-common-handler-test:\n          <<: *non_main_filters'
    mutate "$CFG" $'name: bin-common-handler-test-auto' $'name: bin-common-handler-other-auto'
    run_checker
    [ "$status" -eq 1 ]
}

@test "a command step inside the go-test command fails for an auto-run workflow" {
    mutate "$CFG" $'      - run: echo working on << parameters.source-directory >>' $'      - run: echo working on << parameters.source-directory >>\n      - setup_remote_docker'
    run_checker
    expect_fail "command go-test is missing or runs a deploy"
}

@test "a komodo deploy call in an auto-run job definition fails" {
    mutate "$CFG" $'            cd bin-openapi-manager\n            go mod download' $'            cd bin-openapi-manager\n            .circleci/scripts/komodo-api-deploy.sh bin-openapi-manager x\n            go mod download'
    run_checker
    expect_fail "job definition bin-openapi-manager-validate must not run a deploy, a docker push, login or build, or gcloud"
}

@test "an auto-run workflow that gains a deploy job fails" {
    mutate "$CFG" $'          name: bin-openapi-manager-validate-auto\n          <<: *deploy_filters' $'          name: bin-openapi-manager-validate-auto\n          <<: *deploy_filters\n      - bin-openapi-manager-deploy:\n          requires:\n            - bin-openapi-manager-validate-auto'
    run_checker
    [ "$status" -eq 1 ]
}

# ---- (b)/(e) approval-kept workflows stay approval-kept ----

@test "a -test-auto job added to a workflow outside the allowlist fails" {
    add_keep_fixture
    mutate "$CFG" $'      - bin-fixture-manager-test:\n          requires:' $'      - bin-fixture-manager-test:\n          name: bin-fixture-manager-test-auto\n          requires:'
    run_checker
    expect_fail "workflow bin-fixture-manager has a -test-auto job but is not in the allowlist"
}

@test "an ignore-main filter added to an approval-kept workflow fails" {
    add_keep_fixture
    mutate "$CFG" $'  bin-fixture-manager:\n    when: << pipeline.parameters.run-bin-fixture-manager >>\n    jobs:\n      - build-approval:\n          type: approval\n' $'  bin-fixture-manager:\n    when: << pipeline.parameters.run-bin-fixture-manager >>\n    jobs:\n      - build-approval:\n          type: approval\n          <<: *non_main_filters\n'
    run_checker
    expect_fail "job build-approval has an ignore branch filter but bin-fixture-manager is not auto-deploy"
}

@test "build-approval removed from an approval-kept workflow fails" {
    mutate "$CFG" $'  voip-kamailio-proxy:\n    when:\n      or: [<< pipeline.parameters.run-voip-kamailio-proxy >>]\n    jobs:\n      - build-approval:\n          type: approval\n      - voip-kamailio-proxy-test:\n          requires:\n            - build-approval\n' $'  voip-kamailio-proxy:\n    when:\n      or: [<< pipeline.parameters.run-voip-kamailio-proxy >>]\n    jobs:\n      - voip-kamailio-proxy-test\n'
    run_checker
    expect_fail "workflow voip-kamailio-proxy: approval-kept workflow needs an unfiltered build-approval"
}

@test "an approval-kept deploy cut off from build-approval fails" {
    add_keep_fixture
    mutate "$CFG" $'      - bin-fixture-manager-build:\n          <<: *context_production\n          requires:\n            - bin-fixture-manager-test' $'      - bin-fixture-manager-build:\n          <<: *context_production'
    run_checker
    expect_fail "bin-fixture-manager-deploy is not gated by build-approval"
}

@test "an unreadable config fails closed" {
    printf 'workflows: [unclosed\n' > "$CFG"
    run_checker
    [ "$status" -eq 1 ]
    [[ "$output" == *"cannot check"* ]]
}
