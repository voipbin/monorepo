#!/usr/bin/env bash
#
# Enforces the test conventions declared in docs/conventions/testing.md
# that no Go linter can express. Only inspects files changed relative to
# the merge base with main, so the existing backlog does not fail the build.
#
# Fail-closed: if the merge base cannot be resolved the script exits non-zero.
# The CI job is responsible for fetching origin/main before running this
# (CircleCI's checkout only fetches the current branch's refspec).
#
set -uo pipefail

BASE_REF="origin/main"

if ! MERGE_BASE="$(git merge-base "${BASE_REF}" HEAD 2>/dev/null)" || [ -z "${MERGE_BASE}" ]; then
  echo "check-test-conventions: FAILED to resolve merge base with ${BASE_REF}." >&2
  echo "" >&2
  echo "The job must fetch the base branch before running this script:" >&2
  echo "  git fetch --no-tags origin '+refs/heads/main:refs/remotes/origin/main'" >&2
  echo "" >&2
  echo "Refusing to pass silently: a skipped gate is indistinguishable from" >&2
  echo "a passing one, which is how conventions drift in the first place." >&2
  exit 1
fi

# Resolve the changed test files. `git diff` failure must NOT be swallowed:
# a shallow clone can resolve the merge base and still fail to diff it, and an
# empty array is indistinguishable from "nothing changed". Capture the status
# explicitly instead of relying on mapfile's, which reflects the redirect and
# is always 0.
#
# core.quotePath=false keeps non-ASCII paths usable; git would otherwise emit
# them octal-escaped and quoted, and the name would not resolve on disk.
if ! DIFF_OUT="$(git -c core.quotePath=false diff --name-only --diff-filter=d "${MERGE_BASE}" HEAD -- '*_test.go')"; then
  echo "check-test-conventions: FAILED to diff ${MERGE_BASE}..HEAD." >&2
  echo "The repository may be a shallow clone missing the base commit's objects." >&2
  exit 1
fi

CHANGED=()
while IFS= read -r line; do
  [ -n "${line}" ] || continue
  case "${line}" in vendor/*|*/vendor/*) continue ;; esac
  CHANGED+=("${line}")
done <<< "${DIFF_OUT}"

if [ "${#CHANGED[@]}" -eq 0 ]; then
  echo "check-test-conventions: no changed test files."
  exit 0
fi

# Inspect ADDED LINES ONLY, not whole files.
#
# Checking whole files would make any repo-wide reformat fail this gate: a
# gofmt-only pass rewrites 151 test files, and those files carry 178
# pre-existing violations that the branch never introduced. Scoping to added
# lines keeps the gate on what the branch actually wrote, which is what
# "changed files only" was meant to express in the first place.
#
# -U0 emits no context lines, so every '+' line is genuinely new.
# vendor/ is excluded here too: the pathspec, not the CHANGED array, is what
# bounds this diff. Filtering only the file list would let a vendored test
# file's added lines reach the rules.
if ! DIFF_U0="$(git -c core.quotePath=false diff -U0 "${MERGE_BASE}" HEAD \
  -- '*_test.go' ':(exclude)vendor/**' ':(exclude)*/vendor/**')"; then
  echo "check-test-conventions: FAILED to produce a unified diff." >&2
  exit 1
fi

fail=0

report() {
  # $1 = rule label, $2 = doc anchor, $3 = matches
  echo ""
  echo "✖ ${1}"
  echo "  Convention: docs/conventions/testing.md${2}"
  echo "${3}" | sed 's/^/    /'
  fail=1
}

# Report added-line violations with file:line. `git diff -U0` hunk headers
# (@@ -a,b +c,d @@) carry the new-file line number, so walk the diff and keep
# a running counter; a bare grep over added lines would lose the location.
scan() {
  # $1 = ERE to match against added lines.
  # Note the patterns below are POSIX EREs as awk understands them: no \b, no
  # \<, no \s. awk warns about (and ignores) unknown escapes, which silently
  # disables a rule -- Rule 3 was lost this way during review.
  awk -v pat="$1" '
    /^\+\+\+ b\// { file = substr($0, 7); next }
    /^@@ / {
      # @@ -old,cnt +new,cnt @@
      split($3, a, ",")
      line = a[1]; sub(/^\+/, "", line)
      next
    }
    /^\+/ {
      body = substr($0, 2)
      if (body ~ pat) printf "%s:%d:%s\n", file, line, body
      line++
    }
  ' <<< "${DIFF_U0}"
}

# Rule 1 — Test_<MethodName>, not TestXxx_Case. See 6.4.1 for why bare
# TestXxx (no underscore) is deliberately NOT matched.
m="$(scan '^func Test[A-Z][A-Za-z0-9]*_')"
[ -n "${m}" ] && report \
  "Test function must be named Test_<MethodName> (got TestXxx_Case)." \
  " (13.6 Test Function Naming)" "${m}"

# Rule 2 — assertions use reflect.DeepEqual + t.Errorf, not testify.
m="$(scan '"github[.]com/stretchr/testify')"
[ -n "${m}" ] && report \
  "testify is not used in this repository; use reflect.DeepEqual + t.Errorf." \
  " (13.5 Assertion Pattern)" "${m}"

# Rule 3 — the gomock controller variable is named mc.
m="$(scan '(^|[^A-Za-z0-9_])ctrl[ \t]*:=[ \t]*gomock[.]NewController')"
[ -n "${m}" ] && report \
  "Name the gomock controller 'mc' (mc := gomock.NewController(t))." \
  " (13.3 Test Structure Conventions)" "${m}"

if [ "${fail}" -ne 0 ]; then
  echo ""
  echo "Test convention check failed. These rules are documented in"
  echo "docs/conventions/testing.md and are enforced only on files this"
  echo "branch changes; pre-existing violations elsewhere are untouched."
  exit 1
fi

echo "check-test-conventions: OK (${#CHANGED[@]} file(s) checked)"
