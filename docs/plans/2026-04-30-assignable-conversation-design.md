# Assignable Conversation — Design

**Date:** 2026-04-30
**Status:** Design — pending implementation plan
**Author:** brainstormed with Claude Code; owner Sungtae Kim
**Related:** none yet (parked: flow-decided assignment, push routing — see §9)

---

## 1. Motivation

Today, an inbound message on a `conversation` always triggers the registered flow (an `activeflow` is created in `flow-manager`). For some operational situations a human agent needs to take over a conversation directly instead of letting the flow run — for example, when a VIP customer should be handled manually rather than by the automated flow.

This design adds a **manual takeover** mechanism so that an admin or manager can assign a conversation to a specific agent. Once assigned, new inbound messages on that conversation no longer trigger an `activeflow`; the agent handles them directly via the existing message-send API.

## 2. Scope

### In scope (this design)

- **Manual assignment** — a user with admin or manager permission assigns a conversation to a specific agent under the same customer.
- **Single-owner cardinality** — one agent owns a conversation at a time; an agent can own many conversations.
- **Manual unassignment** — the assigning user (admin/manager) can unassign or reassign. The owning agent can self-unassign (set back to "no agent").
- **Inbound flow-trigger skip** — once assigned, the conversation-manager does **not** create new `activeflow` instances for new inbound messages. Already-running activeflows are not interrupted.
- **List filter** — agents can list conversations they own via the existing `owner_id` filter.

### Out of scope (parked, see §9)

- Flow-decided assignment (a flow action that assigns mid-flow, e.g. on VIP detection).
- Push routing / queue-based distribution.
- Multi-agent collaboration / supervisor mode.
- Assignment history / audit trail beyond the existing event stream.
- Auto-release on idle timeout.
- Skill-based / tag-based routing rules.
- Peer-to-peer hand-off between agents without admin involvement.

## 3. Data model

**No schema changes.** The infrastructure already exists.

`Conversation` (`bin-conversation-manager/models/conversation/conversation.go`) already embeds `commonidentity.Owner`:

```go
type Conversation struct {
    commonidentity.Identity
    commonidentity.Owner          // ← OwnerType + OwnerID, already present
    AccountID uuid.UUID
    // ...
}
```

`commonidentity.Owner` defines:

```go
type Owner struct {
    OwnerType OwnerType
    OwnerID   uuid.UUID
}

const (
    OwnerTypeNone  OwnerType = ""
    OwnerTypeAgent OwnerType = "agent"
)
```

The DB columns `owner_type` and `owner_id` already exist on `conversation_conversations`. They are written today as `("", uuid.Nil)` for every conversation. The pattern of "agent owns this resource" is established elsewhere in the repo (`bin-call-manager`, `bin-storage-manager`).

### State table

| State | OwnerType | OwnerID | Inbound message behavior |
|---|---|---|---|
| Unassigned (default, today's behavior) | `""` | `uuid.Nil` | Trigger activeflow per `account.MessageFlowID` (LINE) or `number.MessageFlowID` (SMS). |
| Assigned to agent | `"agent"` | `<agent-uuid>` | Store the message and publish `message_created`; **do not** create a new activeflow. |

Already-running activeflows are not affected by transitions in either direction. The flow trigger is a per-message decision.

## 4. Inbound dispatch

`bin-conversation-manager/pkg/conversationhandler/` currently triggers activeflows from two entry points:

- `hookLine()` (LINE inbound webhook) — uses `account.MessageFlowID`.
- `MessageEventReceived()` (SMS inbound via subscribed event from message-manager) — uses `number.MessageFlowID`.

Both call `MessageExecuteActiveflow(cv, m, flowID)` directly today.

### New shape

Replace the direct call site with a dispatch on **execute mode**:

```go
type ExecuteMode string

const (
    ExecuteModeNone  ExecuteMode = ""
    ExecuteModeAgent ExecuteMode = "agent"
    ExecuteModeFlow  ExecuteMode = "flow"
)

func (h *conversationHandler) getExecuteMode(cv *conversation.Conversation) ExecuteMode {
    if cv.OwnerType == commonidentity.OwnerTypeAgent && cv.OwnerID != uuid.Nil {
        return ExecuteModeAgent
    }
    return ExecuteModeFlow
}
```

The caller (the inbound entry points) drops to a single dispatcher:

```go
mode := h.getExecuteMode(cv)
switch mode {
case ExecuteModeNone:
    // reserved for future modes; safe no-op
    return nil
case ExecuteModeAgent:
    return h.runExecuteModeAgent(ctx, cv, m)
case ExecuteModeFlow:
    return h.runExecuteModeFlow(ctx, cv, m)
default:
    return fmt.Errorf("unknown execute mode: %s", mode)
}
```

### Mode handlers

- **`runExecuteModeAgent(ctx, cv, m) error`** — no-op. The agent UI learns of the new message via the existing `message_created` event (filtered by `cv.owner_id == self.id`). Logs only.

- **`runExecuteModeFlow(ctx, cv, m) error`** — switches on `cv.Type` and delegates to a per-type runner. The runner fetches its type-specific source (account for LINE, number for SMS), reads the flow ID from it, and calls a shared `executeActiveflow`:

```go
func (h *conversationHandler) runExecuteModeFlow(ctx context.Context, cv *conversation.Conversation, m *message.Message) error {
    switch cv.Type {
    case conversation.TypeLine:
        return h.runExecuteModeFlowLine(ctx, cv, m)
    case conversation.TypeMessage:
        return h.runExecuteModeFlowMessage(ctx, cv, m)
    default:
        return nil
    }
}

func (h *conversationHandler) runExecuteModeFlowLine(ctx context.Context, cv *conversation.Conversation, m *message.Message) error {
    if cv.AccountID == uuid.Nil {
        return nil
    }
    ac, errGet := h.accountHandler.Get(ctx, cv.AccountID)
    if errGet != nil {
        return errors.Wrapf(errGet, "could not get account. account_id: %s", cv.AccountID)
    }
    if errExecute := h.executeActiveflow(ctx, cv, m, ac.MessageFlowID); errExecute != nil {
        return errors.Wrapf(errExecute, "could not execute activeflow")
    }
    return nil
}

// runExecuteModeFlowMessage follows the same shape, fetching by cv.Self.Target from number-manager.

// shared, returns error only — see §4.1
func (h *conversationHandler) executeActiveflow(ctx context.Context, cv *conversation.Conversation, m *message.Message, flowID uuid.UUID) error {
    if flowID == uuid.Nil {
        return nil // no flow configured; non-error skip
    }
    // FlowV1ActiveflowCreate + setVariables + FlowV1ActiveflowExecute
    // (existing MessageExecuteActiveflow body, error-only signature)
    ...
}
```

### 4.1 Why `error` only (not `(*Activeflow, error)`)

`MessageExecuteActiveflow` returns `(*activeflow.Activeflow, error)` today, but no caller actually uses the `*activeflow.Activeflow` for anything except logging `af.ID`. Returning `(nil, nil)` to mean "skipped — no flow configured" is ambiguous and easy to misuse. Dropping the return value:

- Removes the `(nil, nil)` ambiguity at its source.
- Matches the established pattern in `hookLine()` and `MessageEventReceived()`, which already return `error` only.
- The `af.ID` log line moves inside `executeActiveflow`. No information loss.

If a future caller genuinely needs the `*Activeflow`, the return value can be reintroduced with a clean contract at that point.

### 4.2 ExecuteModeNone — reserved

`getExecuteMode(cv)` only returns `Agent` or `Flow` today. `ExecuteModeNone` is reserved as a safe-no-op slot for future owner types or explicit "ignored" states. The `default:` arm catches unknown values defensively.

## 5. API surface

### conversation-manager RPC — already in place

`PUT /v1/conversations/<id>` is a partial-update endpoint that already accepts `owner_type` and `owner_id` in its allowlist (`bin-conversation-manager/pkg/listenhandler/v1_conversations.go::processV1ConversationsIDPut`). The map-based decode (`GetFilteredItems` + `ConvertStringMapToFieldMap`) already preserves the "field absent vs field present with empty/zero value" distinction needed for partial updates.

The only behavior change in conversation-manager is **owner_type derivation** (§5.3) and **validation when assigning** (§5.4). The existing PUT route, allowlist, and decode pipeline stay as-is.

### 5.1 api-manager surface

`PUT /v1.0/conversations/<id>` accepts a partial JSON body. Clients send only the fields they want to change:

```json
PUT /v1.0/conversations/<id>
{"owner_id": "<agent-uuid>"}                              // assign or reassign
{"owner_id": "00000000-0000-0000-0000-000000000000"}      // unassign
{"name": "VIP customer"}                                  // unrelated update
{"name": ""}                                              // explicitly clear name
```

api-manager is a thin gateway. It MUST decode the JSON body as `map[string]any` (not a typed struct that drops zero values) so that `{"name": ""}` survives forwarding intact. It MUST NOT translate or derive `owner_type` — that is conversation-manager's concern.

### 5.2 Permission gate (api-manager)

Per-field check; if **any** field in the payload is denied, the entire request is rejected with **403** (no silent field-stripping):

| Field | admin / manager | owning agent | any other agent |
|:---|:---:|:---:|:---:|
| `owner_id` (assign / reassign — non-nil UUID) | ✓ | ✗ | ✗ |
| `owner_id` (unassign — nil UUID) | ✓ | ✓ (self only) | ✗ |
| `name` / `detail` / `account_id` | ✓ | ✗ | ✗ |

Customer scope is enforced as today: api-manager rejects requests for conversations outside the caller's customer.

### 5.3 owner_type derivation (conversation-manager)

The client never sends `owner_type`. Conversation-manager derives it from the resolved `owner_id` value before applying the update:

- `owner_id == uuid.Nil` → `owner_type = ""`
- `owner_id != uuid.Nil` → `owner_type = "agent"` (the only valid owner type for conversations today)

If a future feature introduces additional owner types for conversations (`team`, `ai`, etc.), the derivation rule extends — and `owner_type` may be re-exposed at the api-manager surface at that point.

### 5.4 Validation (conversation-manager)

When the resolved `owner_type == "agent"` (i.e. setting a non-nil `owner_id`):

- Validate the `OwnerID` exists as an agent (RPC `AgentV1AgentGet` to agent-manager).
- Validate the agent's `CustomerID` matches the conversation's `CustomerID`.
- Reject the update on either failure.

Unassignment (`owner_id = uuid.Nil`) skips agent validation.

### 5.5 List filter — "my conversations"

The existing `GET /v1/conversations` filter in conversation-manager already accepts `owner_id` (registered in `FieldOwnerID` and converted by `ConvertStringMapToFieldMap`). The agent UI calls:

```
GET /v1.0/conversations?owner_id=<self-agent-id>
```

No code change required at the conversation-manager layer. api-manager exposes the filter, scoped by customer per the existing rules.

## 6. Events

**No new event types.** The existing `conversation_updated` event already fires whenever a conversation is updated, including changes to `Owner`. The event payload uses `WebhookMessage`, which embeds `commonidentity.Owner`, so `owner_type` and `owner_id` are already in the JSON.

Subscribers (agent UI, webhook customers) learn about assignment changes via `conversation_updated` and filter on `owner_id`.

The implementation must verify that the existing PUT path actually publishes `conversation_updated`. If for some reason it does not, that is a pre-existing bug to fix as part of this work.

## 7. WebhookMessage / customer-facing payload

**No `WebhookMessage` change.** `bin-conversation-manager/models/conversation/webhook.go` already embeds `commonidentity.Owner` in `WebhookMessage`, and `ConvertWebhookMessage()` already copies it through unconditionally. Customers receive `owner_type` and `owner_id` in every webhook today; they just always read `""` and `00000000-0000-0000-0000-000000000000`.

When this feature ships, customers will start seeing real values (`"agent"` and a UUID) on conversations that get assigned. **This is additive on existing fields, not a breaking change**, but warrants a one-line note in the RST changelog so customers are not surprised.

## 8. RST documentation updates

Per the root [CLAUDE.md](../../CLAUDE.md) RST docs sync rule, any user-visible behavior change must update the RST docs at `bin-api-manager/docsdev/source/` and force-add the rebuilt HTML.

| File | Change |
|---|---|
| `conversation_struct_conversation.rst` | Expand the descriptions of `owner_type` and `owner_id` (already documented as fields) to explain that when populated, the conversation is currently assigned to that agent and inbound messages skip the registered flow trigger. |
| `conversation_overview.rst` | New section: **"Assigning a Conversation to an Agent"**. Cover the `PUT /v1.0/conversations/<id>` partial-update with `owner_id`, unassign via nil UUID, the permission semantics from §5.2, the "no new activeflow while assigned" rule, and the `GET /v1.0/conversations?owner_id=<id>` list filter. |
| `conversation_tutorial.rst` | Walkthrough: admin assigns a conversation, agent receives the webhook update, agent replies via the existing message-send API, agent self-unassigns, the registered flow resumes for subsequent inbound messages. |
| (changelog, top-of-overview) | One-line additive-change note about `owner_type` / `owner_id` in webhook payloads beginning to carry real values when this feature ships. |

Build and commit per the standard workflow:

```bash
cd bin-api-manager/docsdev && rm -rf build && python3 -m sphinx -M html source build
git add -f bin-api-manager/docsdev/build/
```

The rebuilt HTML is force-added because the root `.gitignore` excludes `build/`.

## 9. Test scope

### conversation-manager unit tests (gomock + table-driven)

`pkg/conversationhandler/`:

| Function | Test cases |
|---|---|
| `getExecuteMode` | OwnerType=None → `ExecuteModeFlow`; OwnerType=agent + non-nil OwnerID → `ExecuteModeAgent`; OwnerType=agent + nil OwnerID → `ExecuteModeFlow` (defensive); unknown OwnerType → `ExecuteModeFlow`. |
| `runExecuteModeFlow` | LINE cv routes to `runExecuteModeFlowLine`; Message cv routes to `runExecuteModeFlowMessage`; unsupported `cv.Type` returns nil. |
| `runExecuteModeFlowLine` | account fetched, flowID extracted, `executeActiveflow` called; `cv.AccountID == uuid.Nil` short-circuits; account fetch error wraps and propagates. |
| `runExecuteModeFlowMessage` | number fetched by `cv.Self.Target`, flowID extracted, `executeActiveflow` called; number fetch error wraps and propagates. |
| `runExecuteModeAgent` | no-op — single test asserting no flow RPCs are invoked. |
| `executeActiveflow` | `flowID == uuid.Nil` short-circuits with nil error; non-nil flowID creates + sets variables + executes; each downstream error wraps and propagates. |
| Update path validation | `OwnerType=agent` with valid agent (matching customer) succeeds; agent does not exist → reject; agent's customer mismatch → reject; unassign (`OwnerID=uuid.Nil`) skips validation. |
| `hookLine` (existing) | Existing tests unchanged. Add: when conversation has agent owner, no `MessageExecuteActiveflow` is invoked. |
| `MessageEventReceived` (existing) | Same: when conversation has agent owner, no `MessageExecuteActiveflow` is invoked. |

### conversation-manager listenhandler tests

Add table cases to the existing PUT route tests:
- partial update with only `owner_id` (non-nil) → server derives `owner_type=agent`.
- partial update with `owner_id` = nil UUID → server derives `owner_type=""`.
- partial update with `name=""` → empty string preserved through map decode.

### api-manager tests

- `PUT /v1.0/conversations/<id>` with `{"owner_id": "<uuid>"}` and admin/manager auth → forwarded to conversation-manager.
- Same payload with non-owning agent auth → 403.
- Same payload with owning agent auth, non-nil `owner_id` → 403.
- Same payload with owning agent auth, nil-UUID `owner_id` (self-unassign) → forwarded.
- Combined payload `{"owner_id": "<uuid>", "name": "..."}` with owning-agent auth → 403 (any-field rejection).
- `GET /v1.0/conversations?owner_id=<self>` returns only that agent's conversations.
- `{"name": ""}` is forwarded as-is (empty string preserved through map decode).

### api-validator integration tests

Per the api-validator workflow, add a read+mutate flow:

- Admin assigns a conversation to an agent → list-by-`owner_id` shows it under that agent → conversation webhook reflects the new owner_id → admin unassigns → list-by-`owner_id` no longer shows it.
- No cost-sensitive operations involved (no calls, SMS, email-send, or number purchase).

### Out of test scope

- That activeflows in flow-manager keep running unaffected — flow-manager's existing contract; not re-tested here.
- Webhook delivery semantics — already tested elsewhere.
- `Owner` field plumbing through `WebhookMessage` — already in place; covered indirectly.

### Coverage target

80%+ on new code per repo convention. The new surface is small (`getExecuteMode`, three dispatch runners, validation in the update handler) and trivially testable.

## 10. Out of scope (parked)

| Item | Why parked | Re-engagement signal |
|---|---|---|
| **Option A — flow-decided assignment** (a flow action like `assign_agent` that sets the owner mid-flow). | The data model and dispatch shape are already compatible — adding a flow action that calls the existing PUT works without code changes here. | When flows need to make routing decisions automatically (VIP tag, keyword match, business hours). |
| **Push routing / queue-based distribution** | queue-manager's `queuecall` model is built for ephemeral routing of voice calls, not long-running ownership of message threads. Different problem. | Volume of conversations exceeds manual triage; want round-robin / longest-idle / skill-based distribution. Likely a new "conversation queue" concept. |
| **Multi-agent collaboration / supervisor mode** | One agent per conversation is the cleanest mental model. `Owner` is single-owner by design. | Real collaboration use case (agent + supervisor; transfer with overlap). Would need a `participants` model alongside `Owner`, not a replacement. |
| **Assignment history / audit trail** | The existing `conversation_updated` event stream is a de-facto audit log for subscribers that persist it. | Operations need to query "who handled which conversation when" without scanning event archives. Adds a small `conversation_assignments` history table. |
| **Auto-release on idle timeout** | Manual unassign suffices; no abandoned-assignment problem observed yet. | When abandoned assignments become a real operational pain. Periodic sweep job. |
| **Skill-based / tag-based routing rules** | Implies push routing first. | After push routing lands. |
| **Agent UI changes beyond list filter** (notifications, unread counts, drag-and-drop reassignment) | UI scope; the API surface in §5 already provides everything the UI needs. | When the talk.voipbin.net team picks it up. |
| **Peer-to-peer hand-off between agents** without admin involvement | Out of scope per the chosen permission model (admin/manager assigns; owning agent only self-unassigns). | If the operations model needs peer-to-peer transfer. Trivial extension of the §5.2 permission gate. |

Each parked item is feasible later **without rework** — the chosen design (reuse `Owner`, dispatch-mode at inbound, no schema changes) does not paint us into a corner on any of them.
