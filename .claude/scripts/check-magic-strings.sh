#!/usr/bin/env bash
# Check a single .go file for hardcoded magic strings that should use typed constants.
# Used by Claude Code PostToolUse hook on Write|Edit.
#
# RULE: Never use raw string literals for domain values. Always use the typed
# constant from the package that owns the type. This applies to status values,
# type discriminators, providers, directions, reference types, and any other
# domain-specific string that has a corresponding typed constant defined.
#
# Reads the tool input JSON from stdin to extract the file path.
# Exits 2 (block) if violations found, 0 otherwise.
# Suppress per-line with: // nolint:magicstring
#
# To add new patterns: append to the RULES array below.
# Format: "regex_pattern§owning_package§constant_prefix" (§ delimiter)

set -euo pipefail

# Read tool input from stdin
INPUT=$(cat)
FILE_PATH=$(echo "$INPUT" | jq -r '.tool_input.file_path // .tool_input.file // empty' 2>/dev/null || true)

# Only check .go files
if [[ -z "$FILE_PATH" ]] || [[ "$FILE_PATH" != *.go ]]; then
    exit 0
fi

# Skip vendor, generated, and mock files
if [[ "$FILE_PATH" == *"/vendor/"* ]] || [[ "$FILE_PATH" == *"/gens/"* ]] || [[ "$FILE_PATH" == *"mock_"* ]]; then
    exit 0
fi

# Skip test files — test data often uses string literals legitimately
if [[ "$FILE_PATH" == *"_test.go" ]]; then
    exit 0
fi

# ─── Constants definition files (skip the files that DEFINE the constants) ───
CONST_DEFS=(
    "bin-direct-manager/models/direct/"
    "bin-tts-manager/models/tts/tts.go"
    "bin-tts-manager/models/streaming/"
    "bin-tts-manager/models/speaking/"
    "bin-flow-manager/models/action/"
    "bin-call-manager/models/call/"
    "bin-call-manager/models/recording/"
    "bin-call-manager/models/confbridge/"
    "bin-call-manager/models/externalmedia/"
    "bin-agent-manager/models/agent/"
    "bin-ai-manager/models/aicall/"
    "bin-ai-manager/models/ai/"
    "bin-billing-manager/models/billing/"
    "bin-common-handler/models/"
)
for def in "${CONST_DEFS[@]}"; do
    if [[ "$FILE_PATH" == *"$def"* ]]; then
        exit 0
    fi
done

# ─── Rules table ─────────────────────────────────────────────────────────────
# Each rule: "regex_pattern§owner_package§constant_examples"
#
# The regex should match string literals used in comparisons, assignments,
# or struct field values — NOT inside comments, imports, or log messages.
# Use \b word boundaries where possible to reduce false positives.
RULES=(
    # Direct resource types
    'ResourceType\s*(:=|=|:)\s*"(ai|ai_team|agent|queue|conference|extension)"§bin-direct-manager/models/direct§dmdirect.ResourceType*'

    # TTS providers (assignments, struct fields, and comparisons)
    '[Pp]rovider\s*(:=|=|:)\s*"(gcp|aws)"§bin-tts-manager/models/tts§tmtts.ProviderGCP, tmtts.ProviderAWS'
    '[Pp]rovider\s*[!=]=\s*"(gcp|aws)"§bin-tts-manager/models/tts§tmtts.ProviderGCP, tmtts.ProviderAWS'

    # Streaming vendor names
    'VendorName\s*(:=|=|:)\s*"(gcp|elevenlabs|aws|none)"§bin-tts-manager/models/streaming§streaming.VendorName*'

    # Streaming directions
    'Direction\s*(:=|=|:)\s*"(both|in|out)"§bin-tts-manager/models/streaming or externalmedia§streaming.Direction* or externalmedia.Direction*'

    # Streaming reference types
    'ReferenceType\s*(:=|=|:)\s*"(call|confbridge|conference|ai|queue|transcribe|transfer)"§owning models package§<pkg>.ReferenceType*'

    # Call statuses
    'Status\s*(:=|=|:)\s*"(dialing|ringing|progressing|terminating|canceling|hangup)"§bin-call-manager/models/call§call.Status*'

    # Call types
    'Type\s*(:=|=|:)\s*"(flow|conference|sip-service)"§bin-call-manager/models/call§call.Type*'

    # Call directions
    'Direction\s*(:=|=|:)\s*"(incoming|outgoing)"§bin-call-manager/models/call§call.Direction*'

    # Call hangup reasons
    'HangupReason\s*(:=|=|:)\s*"(normal|failed|busy|cancel|timeout|noanswer|dialout|amd)"§bin-call-manager/models/call§call.HangupReason*'

    # Recording statuses
    'Status\s*(:=|=|:)\s*"(initiating|recording|stopping|ended)"§bin-call-manager/models/recording§recording.Status*'

    # Confbridge statuses
    'Status\s*(:=|=|:)\s*"(progressing|terminating|terminated)"§bin-call-manager/models/confbridge§confbridge.Status*'

    # External media encapsulation
    'Encapsulation\s*(:=|=|:)\s*"(rtp|audiosocket|none)"§bin-call-manager/models/externalmedia§externalmedia.Encapsulation*'

    # External media transport
    'Transport\s*(:=|=|:)\s*"(udp|tcp|websocket)"§bin-call-manager/models/externalmedia§externalmedia.Transport*'

    # Agent statuses
    'Status\s*(:=|=|:)\s*"(available|away|busy|offline)"§bin-agent-manager/models/agent§agent.Status*'

    # AI message roles
    'Role\s*(:=|=|:)\s*"(system|user|assistant|function|tool)"§bin-ai-manager/models/aicall§aicall.MessageRole*'

    # Billing cost types
    'CostType\s*(:=|=|:)\s*"(call_pstn_outgoing|call_pstn_incoming|call_vn|call_extension|call_direct_ext|sms|email|number|number_renew|tts|recording)"§bin-billing-manager/models/billing§billing.CostType*'
)

VIOLATIONS=0

for rule in "${RULES[@]}"; do
    IFS='§' read -r pattern owner constants <<< "$rule"

    while IFS=: read -r line content; do
        # Skip lines with nolint:magicstring override
        if echo "$content" | grep -q 'nolint:magicstring'; then
            continue
        fi
        echo "[Hook] Magic string at $FILE_PATH:$line:$content"
        echo "[Hook]   -> Use typed constants from $owner ($constants)"
        VIOLATIONS=$((VIOLATIONS + 1))
    done < <(grep -nE "$pattern" "$FILE_PATH" 2>/dev/null || true)
done

if [ "$VIOLATIONS" -gt 0 ]; then
    echo ""
    echo "[Hook] Found $VIOLATIONS magic string violation(s)."
    echo "[Hook] RULE: Never use raw string literals for domain values."
    echo "[Hook]       Use the typed constant from the package that owns the type."
    echo "[Hook]       Suppress per-line: // nolint:magicstring"
    exit 2
fi

exit 0
