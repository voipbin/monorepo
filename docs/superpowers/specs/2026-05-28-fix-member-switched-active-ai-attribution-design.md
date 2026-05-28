# Design: Fix member_switched Notification active_ai_id Attribution

**Date:** 2026-05-28
**Branch:** NOJIRA-Fix-member-switched-active-ai-attribution
**Status:** Draft

---

## 1. Problem Statement

When a team AI call transitions from one member to another, `EventPMTeamMemberSwitched` creates a `member_switched` notification message. This message's `active_ai_id` is currently set to the **TO** (incoming) member's AI ID.

The AI audit system filters messages by `active_ai_id = aiID` when building each AI's transcript for performance evaluation. This causes two defects:

1. **FROM AI's transcript is incomplete.** The transition notification does not appear in the outgoing AI's transcript. Its transcript ends abruptly, making it look as though the AI stopped without completing a proper handoff.
2. **TO AI's transcript is misleading.** The transition notification appears as the first message in the incoming AI's transcript, as if the new AI had initiated the switch itself.

---

## 2. Root Cause

In `bin-ai-manager/pkg/messagehandler/event.go`, `EventPMTeamMemberSwitched` resolves a **single** `activeAIID` from `evt.ToMember.ID` and uses it for **two distinct purposes**:

```go
// current code (simplified)
activeAIID := h.resolveTeamMemberAIID(ctx, evt.PipecatcallReferenceID, evt.ToMember.ID)

h.Create(..., WithActiveAIID(activeAIID))        // notification message — wrong AI
h.participantHandler.Create(ctx, ..., activeAIID) // participant record — correct AI
```

The two uses have opposite requirements:
- **Notification message `active_ai_id`** should be the FROM AI — the transition is the outgoing AI's final action (it called the transition function that triggered the switch).
- **Participant recording** should be the TO AI — the incoming AI is the new participant joining.

---

## 3. Proposed Solution

Resolve two separate AI IDs and use each for its correct purpose:

```go
fromAIID := h.resolveTeamMemberAIID(ctx, evt.PipecatcallReferenceID, evt.FromMember.ID)
toAIID   := h.resolveTeamMemberAIID(ctx, evt.PipecatcallReferenceID, evt.ToMember.ID)

h.Create(..., WithActiveAIID(fromAIID))   // notification attributed to FROM AI

if h.participantHandler != nil {
    if toAIID == uuid.Nil {               // guard uses toAIID, not fromAIID
        log.Warnf(...)
    } else if err := h.participantHandler.Create(ctx, ..., toAIID); err != nil {
        log.Warnf(...)
    }
}
```

The nil-guard on participant recording uses `toAIID` — not `fromAIID`. If `toAIID` is `uuid.Nil` the participant write is skipped (same semantics as today); if `fromAIID` is `uuid.Nil` the notification is still created with `uuid.Nil` and the participant block continues unaffected.

### Why `fromAIID` is correct for the notification

The `member_switched` event fires because the FROM member's LLM invoked a transition function. The notification is a direct record of that action. Attributing it to the FROM AI makes the outgoing AI's transcript complete and causal: its conversation turns end with the notification that it handed off.

### Timing safety

`resolveTeamMemberAIID` looks up the team and walks `Members` to find the AI ID for a given member ID. It does **not** rely on `CurrentMemberID` (which has not been updated yet when this handler runs). Passing `evt.FromMember.ID` is safe for the same reason passing `evt.ToMember.ID` was safe — the member ID comes directly from the event.

### Nil-safety

If `fromAIID` resolves to `uuid.Nil` (team data unavailable or race condition), the existing `uuid.Nil` guard and warning log in the current code already handles this case gracefully — the notification message is created with `active_ai_id = uuid.Nil` and execution continues. The participant recording for `toAIID` proceeds independently and is unaffected.

---

## 4. Impact on Audit Transcripts

After the fix, each AI's audit transcript will contain:

| Message type | `active_ai_id` | Appears in |
|---|---|---|
| FROM AI conversation turns (user + assistant) | `fromAIID` | FROM AI audit |
| `member_switched` notification | `fromAIID` | FROM AI audit |
| TO AI conversation turns | `toAIID` | TO AI audit |

The TO AI's transcript starts with its own first user turn — no misleading notification at the top. The FROM AI's transcript ends with the transition notification — a complete and correct record of its participation.

---

## 5. Scope

### In scope
- `bin-ai-manager/pkg/messagehandler/event.go` — `EventPMTeamMemberSwitched`: split one `resolveTeamMemberAIID` call into two and route each result correctly.
- `bin-ai-manager/pkg/messagehandler/event_test.go` — update/add tests to assert `active_ai_id = fromAIID` on the notification and `toAIID` on the participant record.

### Out of scope
- No DB migration. The `active_ai_id` column exists; only the value stored changes.
- No audit query changes. The existing `active_ai_id = aiID` filter already produces the correct result once attribution is fixed at creation time.
- No pipecat-manager changes. The `MemberSwitchedEvent` struct already carries both `FromMember` and `ToMember`.
- No other service changes.

---

## 6. Performance

The fix adds one extra `resolveTeamMemberAIID` call per member switch. Each call issues two RPCs internally: one to fetch the AIcall and one to fetch the team. Since both calls resolve from the same AIcall and the same team, the additional cost is two RPCs per switch event — negligible given member switches are rare within a call. A future optimization could fetch the team once and pass it to both lookups, but this is not required for correctness.

---

## 7. Testing Plan

| Test case | Expected |
|---|---|
| `EventPMTeamMemberSwitched` — notification created with `fromAIID` | `active_ai_id` = FROM member's AI ID |
| `EventPMTeamMemberSwitched` — participant recorded with `toAIID` | participant AI ID = TO member's AI ID |
| `EventPMTeamMemberSwitched` — `resolveTeamMemberAIID` returns `uuid.Nil` for from-member (team data unavailable or race condition) | notification created with `active_ai_id = uuid.Nil`, participant record still attempted with `toAIID`; existing `uuid.Nil` warning log fires |
| Existing tests for single-AI calls | No regression |

---

## 8. Acceptance Criteria

- [ ] `member_switched` notification message has `active_ai_id` equal to the FROM member's AI ID.
- [ ] Participant record is created with the TO member's AI ID (unchanged from current behavior).
- [ ] All existing tests pass.
- [ ] New tests cover the corrected attribution.
- [ ] Full verification workflow passes (`go mod tidy`, `go mod vendor`, `go generate`, `go test`, `golangci-lint`).
