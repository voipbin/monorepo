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
COMMAND=$(echo "$INPUT" | jq -r '.tool_input.command // empty' 2>/dev/null || true)

if [[ -z "$COMMAND" ]]; then
    exit 0
fi

# Allow when MERGE_AUTHORIZED=1 immediately precedes gh pr merge in command-position.
# The combined PCRE matches the authorized form in one pass, preventing the newline
# bypass that two separate greps would allow.
if echo "$COMMAND" | grep -qP '(?:^|[;&|(])\s*(?:\w+=\S+\s+)*MERGE_AUTHORIZED=1\s+(?:\w+=\S+\s+)*gh\s+pr\s+merge\b'; then
    exit 0
fi

if echo "$COMMAND" | grep -qP '(?:^|[;&|(])\s*(?:\w+=\S+\s+)*gh\s+pr\s+merge\b'; then
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
