#!/usr/bin/env bats
# Tests for .circleci/scripts/check-service-migration-default-collation.sh
#
# LIST_ONLY=1 stops after the target resolution and validation, so no database, mysql client or go
# toolchain is needed here. The database steps (create, collation precondition, build, up, down,
# up, summary, exit code) are verified against a scratch MariaDB in the PR evidence.

load 'test_helper'

setup() {
    setup_test_env
    export GIT_AUTHOR_NAME=test GIT_AUTHOR_EMAIL=test@example.com
    export GIT_COMMITTER_NAME=test GIT_COMMITTER_EMAIL=test@example.com
    unset CIRCLE_BRANCH TARGETS LIST_ONLY
    SCRIPT="$SCRIPTS_DIR/check-service-migration-default-collation.sh"
    ORIGIN="$TEST_TEMP_DIR/origin.git"
    REPO="$TEST_TEMP_DIR/work"
}

teardown() {
    teardown_test_env
}

# mk_svc <dir> <up-file-count> [<subdir>]: a service with the files the script validates.
mk_svc() {
    local dir="$1" n="$2" sub="${3:-migrations}" i name
    name="${dir#bin-}"; name="${name%-manager}"
    mkdir -p "$dir/$sub" "$dir/cmd/$name-control"
    echo "package migrations" >"$dir/$sub/embed.go"
    echo "package main" >"$dir/cmd/$name-control/main.go"
    for ((i = 1; i <= n; i++)); do
        printf -v num '%06d' "$i"
        echo "SELECT 1;" >"$dir/$sub/${num}_t$i.up.sql"
        echo "SELECT 1;" >"$dir/$sub/${num}_t$i.down.sql"
    done
}

# fixture: the layout of the monorepo in small, pushed to a bare origin, cloned to $REPO.
make_fixture() {
    local seed="$TEST_TEMP_DIR/seed"
    git init -q -b main "$seed"
    (
        cd "$seed" || exit 1
        mk_svc bin-tag-manager 1
        mk_svc bin-call-manager 2
        mk_svc bin-email-manager 3
        mk_svc bin-timeline-manager 1 migrations/mysql
        mkdir -p bin-timeline-manager/cmd/timeline-control
        echo "CREATE TABLE x (a Int8) ENGINE = MergeTree ORDER BY a;" >bin-timeline-manager/migrations/000001_events.up.sql
        mkdir -p bin-rag-manager/migrations && echo "SELECT 1;" >bin-rag-manager/migrations/000001_rag.up.sql
        echo x >README.md
        git add -A && git commit -q -m seed
    )
    git init -q --bare -b main "$ORIGIN"
    git -C "$seed" push -q "file://$ORIGIN" main
    git clone -q "file://$ORIGIN" "$REPO"
    git -C "$REPO" checkout -q -b pr
}

commit_all() { git -C "$REPO" add -A && git -C "$REPO" commit -q -m "$1"; }

run_list() { (cd "$REPO" && LIST_ONLY=1 "$@" bash "$SCRIPT"); }

@test "a changed service migration file selects that service only" {
    make_fixture
    echo "SELECT 2;" >"$REPO/bin-call-manager/migrations/000002_t2.up.sql"
    commit_all change
    run run_list env
    [ "$status" -eq 0 ]
    [[ "$output" == *"target=bin-call-manager n=2 destructive=0 validate=ok"* ]]
    [[ "$output" != *"bin-tag-manager"* ]]
}

@test "two changed services select both" {
    make_fixture
    echo "-- a" >>"$REPO/bin-call-manager/migrations/000001_t1.up.sql"
    echo "-- a" >>"$REPO/bin-email-manager/migrations/000003_t3.up.sql"
    commit_all change
    run run_list env
    [ "$status" -eq 0 ]
    [[ "$output" == *"target=bin-call-manager"* ]]
    [[ "$output" == *"target=bin-email-manager n=3"* ]]
}

@test "the timeline mysql set is its own target" {
    make_fixture
    echo "-- a" >>"$REPO/bin-timeline-manager/migrations/mysql/000001_t1.up.sql"
    commit_all change
    run run_list env
    [ "$status" -eq 0 ]
    [[ "$output" == *"target=bin-timeline-manager@mysql n=1 destructive=0 validate=ok"* ]]
}

@test "a timeline path outside mysql/ is ClickHouse and only the smoke target runs" {
    make_fixture
    echo "-- a" >>"$REPO/bin-timeline-manager/migrations/000001_events.up.sql"
    mkdir -p "$REPO/bin-timeline-manager/migrations/mongo" && echo x >"$REPO/bin-timeline-manager/migrations/mongo/a.sql"
    commit_all change
    run run_list env
    [ "$status" -eq 0 ]
    [[ "$output" == *"skipped: bin-timeline-manager migrations/ (outside mysql/) is ClickHouse"* ]]
    [[ "$output" == *"smoke target bin-tag-manager"* ]]
    [[ "$output" == *"target=bin-tag-manager n=1"* ]]
    [[ "$output" != *"target=bin-timeline-manager"* ]]
}

@test "bin-rag-manager is skipped as PostgreSQL" {
    make_fixture
    echo "-- a" >>"$REPO/bin-rag-manager/migrations/000001_rag.up.sql"
    commit_all change
    run run_list env
    [ "$status" -eq 0 ]
    [[ "$output" == *"skipped: bin-rag-manager is PostgreSQL"* ]]
    [[ "$output" == *"smoke target bin-tag-manager"* ]]
}

@test "a change outside migrations (the script itself) runs the smoke target" {
    make_fixture
    echo "# a" >>"$REPO/README.md"
    commit_all change
    run run_list env
    [ "$status" -eq 0 ]
    [[ "$output" == *"smoke target bin-tag-manager"* ]]
}

@test "a removed service is skipped" {
    make_fixture
    git -C "$REPO" rm -rq bin-call-manager
    git -C "$REPO" commit -q -m remove
    run run_list env
    [ "$status" -eq 0 ]
    [[ "$output" == *"skipped: bin-call-manager was removed"* ]]
}

@test "deleting only a down file still selects the service" {
    make_fixture
    git -C "$REPO" rm -q bin-email-manager/migrations/000003_t3.down.sql
    git -C "$REPO" commit -q -m deldown
    run run_list env
    [ "$status" -eq 0 ]
    [[ "$output" == *"target=bin-email-manager n=3"* ]]
}

@test "a file renamed out of migrations still selects the service of the old path" {
    make_fixture
    mkdir -p "$REPO/docs"
    git -C "$REPO" mv bin-email-manager/migrations/000003_t3.down.sql docs/moved.sql
    git -C "$REPO" commit -q -m move
    run run_list env
    [ "$status" -eq 0 ]
    [[ "$output" == *"target=bin-email-manager n=3"* ]]
}

@test "a non-ASCII path is not lost to path quoting" {
    make_fixture
    echo x >"$REPO/bin-call-manager/migrations/한글.txt"
    commit_all change
    run run_list env
    [ "$status" -eq 0 ]
    [[ "$output" == *"target=bin-call-manager n=2"* ]]
}

@test "validation: a missing embed.go fails" {
    make_fixture
    rm "$REPO/bin-call-manager/migrations/embed.go"
    commit_all change
    run run_list env
    [ "$status" -eq 1 ]
    [[ "$output" == *"validate=FAIL"*"embed.go is missing"* ]]
}

@test "validation: the control binary directory is derived from the service name" {
    make_fixture
    rm -r "$REPO/bin-call-manager/cmd/call-control"
    echo "-- a" >>"$REPO/bin-call-manager/migrations/000001_t1.up.sql"
    commit_all change
    run run_list env
    [ "$status" -eq 1 ]
    [[ "$output" == *"bin-call-manager/cmd/call-control is missing"* ]]
}

@test "validation: no up file fails" {
    make_fixture
    rm "$REPO"/bin-call-manager/migrations/*.up.sql
    commit_all change
    run run_list env
    [ "$status" -eq 1 ]
    [[ "$output" == *"no *.up.sql"* ]]
}

@test "validation: a gap in the numbers fails" {
    make_fixture
    mv "$REPO/bin-email-manager/migrations/000002_t2.up.sql" "$REPO/bin-email-manager/migrations/000004_t2.up.sql"
    commit_all change
    run run_list env
    [ "$status" -eq 1 ]
    [[ "$output" == *"without gaps"* ]]
}

@test "destructive: only a marker in an *.up.sql counts" {
    make_fixture
    echo "-- DESTRUCTIVE: drops a column" >>"$REPO/bin-email-manager/migrations/000002_t2.up.sql"
    echo "-- DESTRUCTIVE: in a test file" >>"$REPO/bin-call-manager/migrations/migrations_test.go"
    echo "-- DESTRUCTIVE: in a down file" >>"$REPO/bin-call-manager/migrations/000001_t1.down.sql"
    commit_all change
    run run_list env
    [ "$status" -eq 0 ]
    [[ "$output" == *"target=bin-email-manager n=3 destructive=1"* ]]
    [[ "$output" == *"target=bin-call-manager n=2 destructive=0"* ]]
}

@test "TARGETS replaces the scan, applies the exclusions and has no smoke fallback" {
    make_fixture
    run run_list env TARGETS="bin-rag-manager bin-timeline-manager bin-gone-manager"
    [ "$status" -eq 0 ]
    [[ "$output" == *"skipped: bin-rag-manager is PostgreSQL"* ]]
    [[ "$output" == *"skipped: bin-gone-manager was removed"* ]]
    [[ "$output" != *"smoke target"* ]]
    [[ "$output" != *"target=bin-tag-manager"* ]]
}

@test "TARGETS accepts the timeline mysql id" {
    make_fixture
    run run_list env TARGETS="bin-timeline-manager@mysql"
    [ "$status" -eq 0 ]
    [[ "$output" == *"target=bin-timeline-manager@mysql n=1"* ]]
}

@test "main branch: the HEAD~1 range is used" {
    make_fixture
    git -C "$REPO" checkout -q main
    echo "-- a" >>"$REPO/bin-call-manager/migrations/000001_t1.up.sql"
    commit_all change
    run run_list env CIRCLE_BRANCH=main
    [ "$status" -eq 0 ]
    [[ "$output" == *"target=bin-call-manager"* ]]
    [[ "$output" != *"target=bin-email-manager"* ]]
}

@test "main branch: a file renamed out of migrations still selects the service of the old path" {
    make_fixture
    git -C "$REPO" checkout -q main
    mkdir -p "$REPO/docs"
    git -C "$REPO" mv bin-email-manager/migrations/000003_t3.down.sql docs/moved.sql
    git -C "$REPO" commit -q -m move
    run run_list env CIRCLE_BRANCH=main
    [ "$status" -eq 0 ]
    [[ "$output" == *"target=bin-email-manager n=3"* ]]
}

@test "main branch: a depth 1 clone is deepened to reach HEAD~1" {
    make_fixture
    git -C "$REPO" checkout -q main
    echo "-- a" >>"$REPO/bin-call-manager/migrations/000001_t1.up.sql"
    commit_all change
    git -C "$REPO" push -q origin main
    rm -rf "$REPO"
    git clone -q --depth=1 "file://$ORIGIN" "$REPO"
    run run_list env CIRCLE_BRANCH=main
    [ "$status" -eq 0 ]
    [[ "$output" == *"target=bin-call-manager"* ]]
    [[ "$output" != *"every target"* ]]
}

@test "PR branch: a depth 1 clone is deepened until the merge base resolves" {
    make_fixture
    echo "-- a" >>"$REPO/bin-call-manager/migrations/000001_t1.up.sql"
    commit_all change
    git -C "$REPO" push -q origin pr
    rm -rf "$REPO"
    git clone -q --depth=1 --branch pr "file://$ORIGIN" "$REPO"
    run run_list env
    [ "$status" -eq 0 ]
    [[ "$output" == *"target=bin-call-manager"* ]]
}

@test "unknown changed files: every target runs (fail safe) and the count follows the fixture" {
    make_fixture
    git -C "$REPO" remote remove origin
    run run_list env
    [ "$status" -eq 0 ]
    [[ "$output" == *"changed files unknown; checking every target"* ]]
    local expected
    expected="$(cd "$REPO" && ls -d bin-*/migrations | grep -v -e bin-rag-manager -e bin-timeline-manager | wc -l)"
    expected=$((expected + 1))   # the timeline mysql set
    [ "$(grep -c '^target=' <<<"$output")" -eq "$expected" ]
    [[ "$output" == *"target=bin-timeline-manager@mysql"* ]]
}

@test "main branch without HEAD~1 and no remote: every target runs" {
    make_fixture
    git -C "$REPO" remote remove origin
    rm -rf "$REPO/.git" && git -C "$REPO" init -q -b main && git -C "$REPO" add -A && git -C "$REPO" commit -q -m one
    run run_list env CIRCLE_BRANCH=main
    [ "$status" -eq 0 ]
    [[ "$output" == *"changed files unknown; checking every target"* ]]
}
