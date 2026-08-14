#!/bin/bash
# Minimal, self-contained bats helper for .circleci/scripts/ tests.
# Modeled on voipbin/voipbin's .circleci/tests/test_helper.bash - deliberately
# independent of any other test harness in this monorepo.

TESTS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CIRCLECI_DIR="$(dirname "$TESTS_DIR")"
SCRIPTS_DIR="$CIRCLECI_DIR/scripts"

TEST_TEMP_DIR=""
ORIGINAL_PATH="$PATH"

setup_test_env() {
    TEST_TEMP_DIR="$(mktemp -d)"
    export MOCK_BIN_DIR="$TEST_TEMP_DIR/mock_bin"
    mkdir -p "$MOCK_BIN_DIR"
    export PATH="$MOCK_BIN_DIR:$ORIGINAL_PATH"
}

teardown_test_env() {
    if [[ -n "$TEST_TEMP_DIR" && -d "$TEST_TEMP_DIR" ]]; then
        rm -rf "$TEST_TEMP_DIR"
    fi
    export PATH="$ORIGINAL_PATH"
}

# A fake `ssh` that logs every invocation's full argv to $SSH_LOG, then
# exits with the given code without ever actually connecting anywhere.
# Args within one invocation are separated by 0x1e (record separator);
# invocations are separated by 0x1d (group separator) - NOT by a plain
# newline, because the remote command argument itself contains embedded
# newlines (it's a multi-line shell script), which would otherwise be
# misread as invocation boundaries. Lets tests assert on exactly what
# remote command was constructed, without a real network call.
#
# Also captures whatever is piped into ssh's stdin to $SSH_STDIN_LOG,
# one record per invocation separated by 0x1d - used by scripts (like
# komodo-api-deploy.sh) that feed secrets to the remote side over stdin
# rather than via argv or SendEnv. Scripts that don't pipe anything into
# ssh (like ssh-deploy.sh) just produce empty records here; harmless.
install_fake_ssh() {
    SSH_LOG="$TEST_TEMP_DIR/ssh_calls.log"
    SSH_STDIN_LOG="$TEST_TEMP_DIR/ssh_stdin.log"
    : > "$SSH_LOG"
    : > "$SSH_STDIN_LOG"
    local exit_code="${1:-0}"
    cat > "$MOCK_BIN_DIR/ssh" <<EOF
#!/bin/bash
{
    for a in "\$@"; do
        printf '%s\x1e' "\$a"
    done
    printf '\x1d'
} >> "$SSH_LOG"
{
    cat
    printf '\x1d'
} >> "$SSH_STDIN_LOG"
exit $exit_code
EOF
    chmod +x "$MOCK_BIN_DIR/ssh"
}

# Returns the Nth (1-indexed) ssh invocation's argv, one arg per line.
# Uses awk with RS set to the 0x1d record separator (NOT a plain `tr
# '\x1d' '\n' | sed -n Np` split) because the remote-command argument
# itself contains real embedded newlines (it's a multi-line shell script) -
# splitting on ANY newline, including those, would truncate the record at
# the first embedded newline instead of at the actual call boundary. awk's
# RS, unlike a line-oriented tool, does not additionally split records on
# embedded '\n'. (Separately: GNU `tr`'s SET arguments do not support \xHH
# hex escapes at all - '\x1d' there is read as the literal characters
# \,x,1,d - so any `tr`-based approach here would need octal \035/\036
# instead, but is abandoned in favor of awk regardless, for the reason
# above.)
ssh_call_args() {
    local n="$1"
    awk -v RS='\035' -v n="$n" 'NR==n{print; exit}' "$SSH_LOG" | tr '\036' '\n'
}

# The last real argument passed to a given ssh invocation - for this script
# that is always the remote command string. ($NF is dropped: each record
# ends with a trailing \036 from install_fake_ssh's printf, producing one
# empty trailing field.)
ssh_call_remote_command() {
    local n="$1"
    awk -v RS='\035' -v FS='\036' -v n="$n" 'NR==n{print $(NF-1); exit}' "$SSH_LOG"
}

# How many ssh invocations were logged.
ssh_call_count() {
    awk -v RS='\035' 'END{print NR}' "$SSH_LOG"
}

# The Nth (1-indexed) ssh invocation's full stdin content, exactly as
# piped in (no field splitting - callers grep/compare this as one blob).
ssh_call_stdin() {
    local n="$1"
    awk -v RS='\035' -v n="$n" 'NR==n{print; exit}' "$SSH_STDIN_LOG"
}

# Extracts the body of a `VAR=$(cat <<'DELIM' ... DELIM)` quoted heredoc
# from a script file, i.e. the remote-side script komodo-api-deploy.sh
# builds up and ships over ssh as a single command string. Runs it as
# real bash (with placeholders substituted) so tests can catch bugs a
# pure string-inspection assertion on the constructed command cannot -
# e.g. the round-2-review-caught bug where `curl -f` silently discarded
# the HTTP response body on failure, which "looks right" in the
# constructed command string but is wrong at actual execution time.
# Args: script_file heredoc_delimiter
extract_heredoc_body() {
    local script_file="$1" delim="$2"
    # Only treat a line as the heredoc OPENER if the start token is its
    # actual trailing content (ignoring trailing whitespace), not merely
    # present anywhere on the line - a prose comment elsewhere in the
    # file may reference the same "<<'DELIM'" text without being the
    # real heredoc, which would otherwise start extraction from the
    # wrong place entirely.
    awk -v start="<<'${delim}'" -v delim="$delim" '
        { line = $0; sub(/[ \t]+$/, "", line) }
        !flag && line ~ (start "$") { flag=1; next }
        flag && $0 == delim { flag=0 }
        flag { print }
    ' "$script_file"
}

