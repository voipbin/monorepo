# Conversation Title Auto-Generation Design

**Date:** 2026-05-02
**Status:** Approved

## Problem

All auto-generated conversation titles appear as `"conversation with "` (with trailing space) because the title is built by string concatenation with `peer.TargetName`, which is empty for most SMS/MMS peers. The code has two additional issues:

1. `GetOrCreateBySelfAndPeer` hardcodes `conversation.TypeMessage` instead of passing the `conversationType` parameter — the LINE channel would always produce an SMS label if routed through this path.
2. The LINE follow-event path (`hookEventTypeFollow`) duplicates hardcoded title logic independently.

## Goals

- Titles are informative and distinct across all conversations.
- Channel type is visible in the title (`"SMS · "`, `"LINE · "`).
- Peer identity is shown with best available info: display name, raw identifier, or `"Unknown"`.
- For opaque platform IDs (LINE user IDs, UUIDs), raw identifiers are suppressed from the title.
- Single canonical title-building function, no duplication.
- No public API contract change (`Create()` signature and behavior unchanged).
- No database migration (existing rows left as-is).

## Non-Goals

- Retroactive migration of existing `"conversation with "` titles.
- Contact lookup / enrichment at creation time.
- User-facing rename UI (already possible via existing `PUT /v1/conversations/:id`).

## Format

### Name field

| Condition | Result |
|-----------|--------|
| `TargetName` and `Target` both known, target is human-readable | `"SMS · Alice (+14155551234)"` |
| `TargetName` known, target is opaque or empty | `"LINE · Alice"` |
| Only `Target` known | `"SMS · +14155551234"` |
| Both empty | `"SMS · Unknown"` |

Channel labels: `TypeMessage → "SMS"`, `TypeLine → "LINE"`, unknown → `string(type)`.
Separator: ` · ` (U+00B7 MIDDLE DOT).

### Detail field

`"{Label} conversation"` — e.g., `"SMS conversation"`, `"LINE conversation"`.

The `self` and `peer` address fields on the `Conversation` struct already carry structured directional information; repeating it as prose in `detail` is redundant.

### Human-readable vs opaque targets

The raw `Target` field is shown in parentheses only when the address type signals a human-readable identifier:

| Address type | Target example | Human-readable? |
|---|---|---|
| `TypeTel` | `+14155551234` | ✅ Yes |
| `TypeEmail` | `alice@example.com` | ✅ Yes |
| `TypeSIP` | `alice@sip.example.com` | ✅ Yes |
| `TypeExtension` | `1001` | ✅ Yes |
| `TypeLine` | `Uxxxxxxxxxxxxxxxxx` | ❌ No (opaque user ID) |
| `TypeAgent` | UUID | ❌ No |
| `TypeAI` | UUID | ❌ No |
| `TypeConference` | UUID | ❌ No |
| Unknown | — | ❌ No (conservative default) |

## Implementation

### New package: `internal/convtitle/build.go`

```go
package convtitle

import (
    commonaddress "monorepo/bin-common-handler/models/address"
    "monorepo/bin-conversation-manager/models/conversation"
)

const titleSep = " · " // U+00B7 MIDDLE DOT

// Build returns the auto-generated name and detail for a new conversation.
func Build(convType conversation.Type, peer commonaddress.Address) (name, detail string) {
    label := channelLabel(convType)
    name = label + titleSep + peerName(peer)
    detail = label + " conversation"
    return
}

// channelLabel returns the human-readable channel name for a conversation type.
// When adding a new conversation.Type, add a case here — do not rely on the fallback.
func channelLabel(t conversation.Type) string {
    switch t {
    case conversation.TypeLine:
        return "LINE"
    case conversation.TypeMessage:
        return "SMS"
    default:
        return string(t)
    }
}

// peerName returns the best available display name for a peer address.
// For human-readable address types (tel, email, sip, extension), the raw
// Target is appended in parentheses when a TargetName is also present.
// For opaque types (line user IDs, UUIDs), the raw Target is suppressed.
func peerName(peer commonaddress.Address) string {
    if peer.TargetName != "" {
        if humanReadableTarget(peer.Type) && peer.Target != "" {
            return peer.TargetName + " (" + peer.Target + ")"
        }
        return peer.TargetName
    }
    if peer.Target != "" {
        return peer.Target
    }
    return "Unknown"
}

// humanReadableTarget returns true when the address Target field contains
// a human-readable identifier (phone number, email, SIP URI, extension).
// New address types with human-readable targets must be added here explicitly.
// Unknown types default to false (opaque) for safety.
func humanReadableTarget(t commonaddress.Type) bool {
    switch t {
    case commonaddress.TypeTel, commonaddress.TypeEmail,
        commonaddress.TypeSIP, commonaddress.TypeExtension:
        return true
    default:
        return false
    }
}
```

### Files changed

| File | Change |
|------|--------|
| **NEW** `internal/convtitle/build.go` | Title-building logic |
| **NEW** `internal/convtitle/build_test.go` | Unit tests: all peerName branches × channel types; both-empty → `"Unknown"`; opaque type with non-empty Target |
| **MODIFY** `pkg/conversationhandler/db.go` | `GetOrCreateBySelfAndPeer`: replace hardcoded string with `convtitle.Build(conversationType, peer)`; fix pre-existing bug — replace hardcoded `conversation.TypeMessage` with `conversationType` parameter |
| **MODIFY** `pkg/linehandler/hook.go` | `hookEventTypeFollow`: call `convtitle.Build(conversation.TypeLine, *peer)`; pass result to `ConversationV1ConversationCreate` RPC |
| **MODIFY** `pkg/conversationhandler/db_test.go` | Update `Test_GetOrCreateBySelfAndPeer_Create` mock expectation: `Name: "SMS · Peer Name (+0987654321)"`, `Detail: "SMS conversation"` |
| **MODIFY** `bin-api-manager/docsdev/source/conversation_overview.rst` | Add note: auto-generated title format; rebuild HTML and commit both |

### `Create()` — unchanged

No modification to `Create()`. Auto-generation happens only at the two internal call sites.

### Data flow after the change

```
Inbound SMS event
  → subscribehandler
  → conversationhandler.MessageEventReceived
  → conversationhandler.GetOrCreateBySelfAndPeer
  → convtitle.Build(conversationType, peer) → ("SMS · Alice (+1415…)", "SMS conversation")
  → conversationhandler.Create(…, name, detail, conversationType, …)

LINE follow event
  → linehandler.hookEventTypeFollow
  → convtitle.Build(conversation.TypeLine, peer) → ("LINE · Alice", "LINE conversation")
  → reqHandler.ConversationV1ConversationCreate(…, name, detail, TypeLine, …)
  → [RabbitMQ RPC] → listenhandler → conversationhandler.Create(…)
```

## Examples

| Scenario | name | detail |
|---|---|---|
| SMS from Alice (+14155551234) to +14155557777 | `SMS · Alice (+14155551234)` | `SMS conversation` |
| SMS from unknown number | `SMS · +14155551234` | `SMS conversation` |
| SMS, both fields empty | `SMS · Unknown` | `SMS conversation` |
| LINE follow from Alice | `LINE · Alice` | `LINE conversation` |
| LINE follow, no display name | `LINE · Unknown` | `LINE conversation` |

## Verification

Run from `bin-conversation-manager/`:
```bash
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

Rebuild RST docs from `bin-api-manager/docsdev/`:
```bash
rm -rf build && python3 -m sphinx -M html source build
```
