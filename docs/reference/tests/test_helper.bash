#!/bin/bash
# Minimal, self-contained bats helper for docs/reference/extractor.sh tests.
# Modeled on .circleci/tests/test_helper.bash - deliberately independent of
# any other test harness in this monorepo.
#
# extractor.sh resolves two paths RELATIVE TO THE CURRENT WORKING DIRECTORY
# (not relative to the script itself, and not relative to the service dir
# it's asked to scan):
#   docs/reference/service-taxonomy.md              (service name -> class)
#   bin-common-handler/models/outline/queuename.go  (QueueName const -> string)
# Its usage comment says "Run from monorepo root." for exactly this reason.
# So every test builds a self-contained fake monorepo root under
# TEST_TEMP_DIR with those two files plus a fixture service directory, cds
# into it, and invokes the real extractor.sh (by its fixed absolute path,
# captured below before any cd) against that fixture.

TESTS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REFERENCE_DIR="$(dirname "$TESTS_DIR")"
EXTRACTOR_SH="$REFERENCE_DIR/extractor.sh"

TEST_TEMP_DIR=""
ORIGINAL_PATH="$PATH"
ORIGINAL_PWD=""

setup_extractor_test_env() {
    TEST_TEMP_DIR="$(mktemp -d)"
    ORIGINAL_PWD="$PWD"

    mkdir -p "$TEST_TEMP_DIR/docs/reference"
    mkdir -p "$TEST_TEMP_DIR/bin-common-handler/models/outline"

    cat > "$TEST_TEMP_DIR/docs/reference/service-taxonomy.md" <<'EOF'
# Service Taxonomy (test fixture - not the real one)
| Service | Class | Notes |
|---------|-------|-------|
EOF

    cat > "$TEST_TEMP_DIR/bin-common-handler/models/outline/queuename.go" <<'EOF'
package outline

// QueueName type define (test fixture - not the real one)
type QueueName string

const (
EOF

    export MOCK_BIN_DIR="$TEST_TEMP_DIR/mock_bin"
    mkdir -p "$MOCK_BIN_DIR"
    export PATH="$MOCK_BIN_DIR:$ORIGINAL_PATH"
}

teardown_extractor_test_env() {
    cd "$ORIGINAL_PWD" 2>/dev/null || true
    if [[ -n "$TEST_TEMP_DIR" && -d "$TEST_TEMP_DIR" ]]; then
        rm -rf "$TEST_TEMP_DIR"
    fi
    export PATH="$ORIGINAL_PATH"
}

# Registers a service in the fake taxonomy table extractor.sh looks up the
# service's class from. Args: service_name class [notes]
add_service_class() {
    local svc="$1" class="$2" notes="${3:-test fixture}"
    printf '| %s | %s | %s |\n' "$svc" "$class" "$notes" \
        >> "$TEST_TEMP_DIR/docs/reference/service-taxonomy.md"
}

# Registers a QueueName constant -> string value mapping, in the shape
# resolve_const_names()'s `grep -E "^\s+${const_name}\s+"` expects.
# Args: const_name value
add_queue_name() {
    local const_name="$1" value="$2"
    printf '\t%s QueueName = "%s"\n' "$const_name" "$value" \
        >> "$TEST_TEMP_DIR/bin-common-handler/models/outline/queuename.go"
}

# Writes stdin (use as `write_fixture_file "path" <<'EOF' ... EOF`) to a file
# at $TEST_TEMP_DIR/<rel_path>, creating parent directories as needed.
write_fixture_file() {
    local rel_path="$1" full_path
    full_path="$TEST_TEMP_DIR/$rel_path"
    mkdir -p "$(dirname "$full_path")"
    cat > "$full_path"
}

# Runs extractor.sh against a fixture service directory (given as a path
# relative to TEST_TEMP_DIR, e.g. "fake-svc") from within the fake monorepo
# root, capturing $status/$output the way bats' `run` normally would (this
# must be called as `run run_extractor <svc>` from the test, same as any
# other `run`-wrapped command, so bats' assertions work unchanged).
run_extractor() {
    local svc_dir="$1"
    (cd "$TEST_TEMP_DIR" && bash "$EXTRACTOR_SH" "$svc_dir")
}

# The generated extracted.json for a fixture service, as a jq-queryable path.
extracted_json_path() {
    local svc="$1"
    echo "$TEST_TEMP_DIR/docs/.docs-gen/${svc}.extracted.json"
}

# jq's compact output for a field of a fixture service's extracted.json.
extracted_json_field() {
    local svc="$1" query="$2"
    jq -c "$query" "$(extracted_json_path "$svc")"
}

# Makes `awk` on PATH resolve to mawk (a non-gawk implementation) instead of
# whatever it normally would (gawk, in this dev environment) so a test can
# assert a fix behaves identically under both. Skips the calling test (via
# bats' `skip`) if mawk isn't installed on this machine, rather than failing.
use_mawk() {
    local mawk_path
    mawk_path="$(command -v mawk)" || skip "mawk not installed on this machine"
    ln -sf "$mawk_path" "$MOCK_BIN_DIR/awk"
}
