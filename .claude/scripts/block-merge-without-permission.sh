#!/usr/bin/env bash
# Block `gh pr merge` unless the user explicitly authorized it in this turn.
# PreToolUse hook on Bash tool — exits 2 to block the command.
#
# The ONLY safe time to call gh pr merge is when the user has just said
# "merge" (or equivalent) in their message for that specific PR.
# Review loops, automation workflows, and CI helpers MUST stop at approval
# and report back — they must NOT merge on the AI's own initiative.

set -euo pipefail

INPUT=$(cat)
COMMAND=$(printf '%s\n' "$INPUT" | jq -r '.tool_input.command // empty' 2>/dev/null || true)

if [[ -z "$COMMAND" ]]; then
    exit 0
fi

# Use printf+grep -z so the whole command string is treated as one record,
# making ^ anchor to the very start of the string rather than to each line.
# This prevents a heredoc body line that starts with the gh command text from
# triggering a false positive.

# ALLOW when MERGE_AUTHORIZED=1 immediately precedes gh pr merge in command-position
# on the SAME LINE ([ \t]+ — no newlines).  Using \s+ instead would let an AI bypass
# the guard by injecting a newline between the prefix and the command (PCRE \s matches \n).
# Note: $(gh pr merge ...) is intentionally blocked — the subshell executes the command.
if printf '%s\0' "$COMMAND" | grep -zqP '(?:^|[;&|(])[ \t]*(?:\w+=\S+[ \t]+)*MERGE_AUTHORIZED=1[ \t]+(?:\w+=\S+[ \t]+)*gh[ \t]+pr[ \t]+merge\b'; then
    exit 0
fi

if printf '%s\0' "$COMMAND" | grep -zqP '(?:^|[;&|(`)]\s*(?:\w+=\S+\s+)*gh\s+pr\s+merge\b'; then
    echo ""
    echo "========================================================================"
    echo "  BLOCKED: gh pr merge requires explicit user permission"
    echo "========================================================================"
    echo ""
    echo "NEVER merge a PR without the user explicitly saying 'merge' for this PR"
    echo "in this turn. This applies even after a review loop approves the PR."
    echo ""
    echo "A review loop ends at approval — report the approval and STOP."
    echo "Do not merge. Wait for the user to authorize the merge."
    echo ""
    echo "When the user does authorize a merge, prefix the command with"
    echo "MERGE_AUTHORIZED=1, e.g.:"
    echo "  MERGE_AUTHORIZED=1 gh pr merge <pr> --squash --delete-branch"
    echo ""
    echo "========================================================================"
    echo ""
    exit 2
fi

exit 0
