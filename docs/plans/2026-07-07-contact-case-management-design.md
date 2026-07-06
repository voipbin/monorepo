# Contact Case Management — Design

**Date:** 2026-07-07
**Status:** Design — pending implementation plan
**Author:** brainstormed with Claude Code (CPO); owner Sungtae Kim
**Related:** `docs/plans/2026-06-26-add-contact-crm-interaction-timeline-design.md` (Interaction/Resolution foundation), `docs/plans/2026-04-30-assignable-conversation-design.md` (Owner pattern precedent)

---

## 1. Motivation

Today, `bin-contact-manager` projects every Voice/SMS/Email/Chat touch into a flat `Interaction` timeline (VOIP-1208/1209). This is correct for "what happened with this contact, ever" but gives agents no way to work a **bounded unit of engagement** — there is no concept of "this issue is now open" and "this issue is now closed."

This design introduces **Case**: a thin, per-channel session header that groups related Interactions into a start/end unit agents can pick up, work, and close — without touching the existing Interaction projection pipeline.

## 2. Scope

### In scope

- `Case` entity: get-or-create keyed by `(customer_id, peer_type, peer_target, reference_type)`, with explicit-context override for same-session cross-channel actions (e.g. agent sends SMS mid-call).
- Case lifecycle: open → closed (`agent_closed` / `timeout` / `merged`-reserved). No reopen; re-contact creates a new Case linked via `previous_case_id`.
- `Interaction.case_id` (nullable FK) — Interaction stays the event log; Case is the new grouping layer.
- `Resolution.case_id` (nullable) — case-level contact attribution, promoted to first-class alongside the existing interaction-level override.
- `Case.contact_id` — denormalized cache, single source of truth is Resolution, single derivation function.
- Case ownership (`owner_type`/`owner_id`) reusing the existing `commonidentity.Owner` pattern (see `assignable-conversation-design.md`).
- Case tagging via the existing `bin-tag-manager` (Case registered as a taggable resource type).
- `CaseNote` — new, physically separate table for agent-internal notes. Never customer-facing.
- Explicit case_id propagation for actions taken inside an active case context (e.g. outbound SMS sent while a call is active).

### Out of scope (parked)

| Item | Why parked | Re-engagement signal |
|---|---|---|
| Omni-channel case merge (unify call + SMS + email cases for the same contact) | Requires stable contact resolution first; contact resolution is async/best-effort today | Contact resolution pipeline matures; real multi-channel-per-issue demand observed |
| Per-customer configurable case timeout | No real demand yet; platform-wide env default is simpler | A customer explicitly requests a different SLA window |
| Priority field | Meaningless without queue/routing integration | Queue-based case routing is designed |
| SLA timers / CSAT surveys | Separate feature surface, non-trivial | Explicit product requirement |
| Case assignment history table beyond event stream | `case_updated` event stream is a de-facto audit log | Ops needs to query without scanning event archives |

## 3. Data model

### 3.1 `Case` (new, owned by `bin-contact-manager`)

```go
type Case struct {
    ID         uuid.UUID
    CustomerID uuid.UUID

    PeerType     commonaddress.Type // reuse commonaddress.Type (tel, email, whatsapp, ...)
    PeerTarget   string             // normalized via commonaddress.NormalizeTarget — bit-identical to
                                     // contact_addresses.target and interaction.peer_target
    ReferenceType string            // reuses conversation-manager's message.ReferenceType vocabulary
                                     // (call | message | ...) — NOT a new vocabulary

    ContactID *uuid.UUID // nullable; denormalized cache, see §3.3

    commonidentity.Owner // OwnerType + OwnerID — reused as-is from the conversation assignment precedent

    Status         string     // open | closed
    OpenedAt       *time.Time
    ClosedAt       *time.Time
    ClosedReason   string     // agent_closed | timeout | merged (merged reserved, unused until phase 2)
    ClosedByType   string     // agent | system
    ClosedByID     *uuid.UUID

    PreviousCaseID *uuid.UUID // re-contact chain; nil for the first case with a given peer

    TMCreate *time.Time
    TMUpdate *time.Time
}
```

Constraint:

```sql
CREATE UNIQUE INDEX uq_case_open_peer
  ON contact_cases (customer_id, peer_type, peer_target, reference_type)
  WHERE status = 'open';
```

Partial unique index — at most one **open** Case per (customer, peer, reference_type). Closed cases are unconstrained; many can accumulate for the same peer over time, chained by `previous_case_id`.

### 3.2 `Interaction` (existing, one new column)

```go
type Interaction struct {
    // ... existing fields unchanged (VOIP-1208 design) ...
    CaseID *uuid.UUID // nullable FK -> Case.id; nil for pre-Case historical rows, always set going forward
}
```

No backfill of historical Interactions into synthetic Cases — see §8.

### 3.3 `Resolution` (existing, extended)

```go
type Resolution struct {
    // ... existing fields unchanged ...
    CaseID *uuid.UUID // nullable — new case-level attribution path
    // InteractionID remains nullable — existing per-interaction override path
}
```

```sql
ALTER TABLE contact_resolutions ADD CHECK (interaction_id IS NOT NULL OR case_id IS NOT NULL);

CREATE UNIQUE INDEX uq_resolution_case_positive
  ON contact_resolutions (case_id)
  WHERE resolution_type = 'positive' AND interaction_id IS NULL AND tm_delete IS NULL;
```

A Resolution row now has two independent modes:
- **`case_id` set** (primary path): "this whole case belongs to this contact." All Interactions carrying this `case_id` inherit the attribution.
- **`interaction_id` set** (exception path, existing behavior): fine-grained override of a single Interaction — used to add/remove one message from an otherwise-correct case attribution.

The partial unique index guarantees at most one active case-level positive Resolution per case, which is required for `Case.contact_id` (§3.4) to be derivable without ambiguity.

### 3.4 `Case.contact_id` — derivation, not a second source of truth

```
deriveCaseContactID(case_id):
  SELECT contact_id FROM contact_resolutions
  WHERE case_id = ?
    AND resolution_type = 'positive'
    AND interaction_id IS NULL
    AND tm_delete IS NULL
  -> exists: that contact_id
  -> none:   NULL
```

This single function is the **only** place `Case.contact_id` is computed. Two call sites:

1. **Write path** — invoked inside the same transaction whenever a case-level Resolution is created, soft-deleted, or replaced. `Case.contact_id` is written directly from the function's result; no re-derivation elsewhere.
2. **Recovery path** — `case-control` CLI command `case reconcile-contact <case_id | --all>` re-runs the same function and overwrites `Case.contact_id`. Idempotent. Used only if drift is discovered (e.g. a bulk import wrote Resolution rows without going through the handler). No scheduled reconciliation job at this stage — added only after a real drift incident, per platform incident-response convention.

### 3.5 `CaseNote` (new, owned by `bin-contact-manager`)

```go
type CaseNote struct {
    ID         uuid.UUID
    CustomerID uuid.UUID
    CaseID     uuid.UUID

    AuthorType string // agent | system
    AuthorID   *uuid.UUID

    Text string

    TMCreate *time.Time
    TMUpdate *time.Time
    TMDelete *time.Time
}
```

**Physically separate from `Interaction`.** CaseNote is never surfaced in any customer-facing webhook or API response. Mixing internal notes into the Interaction timeline (e.g. via `ReferenceType: "note"`) is explicitly rejected — it would require every consumer of the customer-facing timeline to correctly filter internal rows, and a single missed filter becomes a customer-visible data leak. A separate table makes that class of bug structurally impossible.

## 4. Case get-or-create

Triggered from the same projection points that create Interactions today (`EventCallCreated`, `EventConversationMessageCreated`, and channel-specific equivalents), before the Interaction insert.

```
BEGIN TRANSACTION

1. IF the triggering event carries an explicit case_id hint (§4.1):
     case_id = hint value
     -- skip steps 2-4 entirely; peer/reference_type matching is bypassed

   ELSE:
     SELECT * FROM contact_cases
     WHERE customer_id=? AND peer_type=? AND peer_target=? AND reference_type=? AND status='open'
     FOR UPDATE

     IF found AND (now - tm_update) < CASE_TIMEOUT_HOURS:
         case_id = found.id   -- reuse

     ELSE IF found (timed out):
         UPDATE contact_cases SET status='closed', closed_at=now, closed_reason='timeout'
         WHERE id = found.id
         INSERT INTO contact_cases (..., status='open', opened_at=now, previous_case_id=found.id)
         RETURNING id INTO case_id

     ELSE (not found):
         last_closed = SELECT * FROM contact_cases
                       WHERE customer_id=? AND peer_type=? AND peer_target=? AND reference_type=?
                       ORDER BY tm_create DESC LIMIT 1
         INSERT INTO contact_cases (..., status='open', opened_at=now,
                previous_case_id = last_closed.id if exists else NULL)
         RETURNING id INTO case_id

2. Attempt contact auto-match via existing address-set lookup (same mechanism as
   Interaction's automatic contact matching); if matched, set contact_id on the new Case row.

3. INSERT INTO contact_interactions (..., case_id=case_id)

4. UPDATE contact_cases SET tm_update=now WHERE id=case_id

COMMIT
```

`FOR UPDATE` serializes concurrent get-or-create attempts for the same peer within one transaction; combined with the partial unique index, this makes the operation race-free without a retry loop. Acceptable at current traffic; revisit only if lock contention is observed.

### 4.1 Explicit case_id override

**Problem it solves:** an agent mid-call sends an SMS to the same customer. Naive peer-based matching would key the SMS event by `reference_type=message`, landing it in a *different* Case than the active `reference_type=call` Case — even though both are part of the same engagement the agent is actively working.

**Resolution:** actions taken from an already-known case context (e.g. the agent's UI already displays `call_id` → `case_id` for the active call) must propagate that `case_id` explicitly through to the outbound action, rather than relying on the receiving side to re-infer it from peer address. This follows the "reuse the value the caller already holds, don't re-derive" convention used elsewhere in the codebase (e.g. `DeriveEndpoints`).

Priority order is strict:

```
1. Explicit case_id hint present  -> use it, skip peer/reference_type matching entirely
2. No hint                        -> (customer_id, peer_type, peer_target, reference_type) get-or-create
```

Only the "agent sends SMS from an active call" action is in scope for wiring this hint at launch. Additional cross-channel actions (e.g. email attachment sent from a chat case) can adopt the same mechanism later, once a real need is confirmed — no speculative wiring now.

## 5. Case lifecycle

### 5.1 Closing

```
POST /v1/cases/{id}/close
{ "closed_by_type": "agent", "closed_by_id": "<agent-uuid>" }
```

```sql
UPDATE contact_cases
SET status='closed', closed_at=now, closed_reason='agent_closed',
    closed_by_type=?, closed_by_id=?
WHERE id=? AND status='open'
```

The `WHERE status='open'` guard makes this an idempotent, race-tolerant optimistic update: a double-close request or a request racing against a concurrent inbound event both resolve safely without a distributed lock. If an inbound event's transaction commits first, the next get-or-create sees `open` and reuses the case; if this close commits first, the inbound event's get-or-create sees `closed` and opens a new chained case. Both orderings are correct; the only externally visible difference is which Case the borderline message lands in — an acceptable, non-corrupting edge case per the platform's "don't engineer for un-observed incidents" convention.

### 5.2 Timeout

Evaluated **lazily** at the next inbound event for that peer (§4), not via a scheduled sweep. A scheduled job would produce `timeout`-closed events with no corresponding user action to explain them and adds an operational component for no real benefit at current scale.

`CASE_TIMEOUT_HOURS` — platform-wide env var, default `24`. No per-customer override (see §2 parked items). Follows the existing `bin-ai-manager` precedent (`AIcallConversationIdleTimeoutHours`) for both the config shape and the `SetXXXForTest` test-override helper convention.

### 5.3 No reopen

A closed Case is immutable. Re-contact from the same peer always creates a new Case with `previous_case_id` pointing at the prior one — regardless of how soon after closing the re-contact happens. This keeps "who closed this and when" permanently unambiguous, which matters for reporting on case handling time and for the closing-agent attribution the platform treats as a hard invariant. The cost is a `reopen`-style UX is not offered; agents needing continuity rely on the `previous_case_id` chain to see history.

## 6. Contact attribution and the unresolved queue

`Case.contact_id` is nullable and this is the normal state for a new peer, not an error condition. Three outcomes:

- **Agent links to an existing contact** — creates a case-level positive Resolution; `Case.contact_id` derives immediately (§3.4).
- **Agent creates a new contact** — `ContactCreate`, then link as above.
- **Agent closes without ever resolving** — legitimate (spam, misdial, one-off). No Resolution is created; the case simply closes with `contact_id` still NULL.

```
CaseListUnresolved:
  WHERE customer_id=? AND contact_id IS NULL AND status='open'
```

This is the agent's live work queue. Closing a case removes it from the queue regardless of resolution state — closing *is* the "no further action needed" signal; no separate "permanently unresolved" marker is introduced.

Resolution can be attached retroactively to a closed Case at any time (e.g. the contact is identified later), which updates `Case.contact_id` via the same derivation function.

## 7. Case-adjacent agent workflow (this design's addition over the base Interaction/Resolution foundation)

Based on common patterns across ticketing/case platforms (assignment, internal notes, tagging, audit trail), cross-checked against what VoIPBin already has:

| Capability | Approach | Why |
|---|---|---|
| Ownership / assignment | Reuse `commonidentity.Owner` (`owner_type`/`owner_id`) exactly as conversation-manager already does it | Existing, proven pattern (`assignable-conversation-design.md`); same permission model (admin/manager assign, owning agent self-unassigns) applies unchanged |
| Tags | Register `Case` as a taggable resource in the existing `bin-tag-manager` | Purpose-built generic tagging service already exists; no new tag storage needed |
| Audit trail | Existing `case_created` / `case_updated` / `case_closed` webhook events | Same convention already used for Interaction/Resolution/Conversation; no new audit table |
| Internal notes | New `CaseNote` table (§3.5) | Genuinely missing capability; must be physically isolated from customer-facing data |
| Priority, SLA, CSAT | Explicitly out of scope (§2) | Meaningless without routing/queue integration or a separate survey feature; adding now risks unused decoration |

## 8. Migration

1. Create `contact_cases` table (with the partial unique index).
2. Add `case_id` (nullable) to `contact_interactions`.
3. Add `case_id` (nullable) to `contact_resolutions`, plus the CHECK constraint and the partial unique index.
4. Create `contact_case_notes` table.
5. Register `case` as a resource type in `bin-tag-manager`.
6. **No backfill.** Historical Interactions predate the Case concept; retroactively grouping them by peer+time-gap heuristics would invent case boundaries that never existed operationally and could misattribute historical activity. The feature's value is forward-looking case management, not historical reconstruction.

## 9. Service/API surface impact

- **bin-contact-manager**: `CaseHandler` (get-or-create, close, assign/unassign, list, list-unresolved), `CaseNoteHandler` (create/list/delete), Resolution handler extended with the case-level derivation hook, `case-control` CLI (including `reconcile-contact`).
- **bin-tag-manager**: register `case` as a taggable resource type (no schema change expected — generic tagging already supports arbitrary resource types).
- **bin-api-manager**: `/v1.0/cases`, `/v1.0/cases/{id}`, `/v1.0/cases/{id}/close`, `/v1.0/cases/{id}/notes`; extend `/v1.0/interactions` responses with `case_id`.
- **bin-openapi-manager**: spec additions per `voipbin-openapi-spec-handler-parity` convention.
- **square-admin**: case list / unresolved queue / close button / notes panel — separate frontend design, not covered here.

## 10. Test scope (high level; detailed table-driven cases in the implementation plan)

- Case get-or-create: reuse on open match, timeout-triggered chain, explicit-hint bypass, concurrent creation (race via FOR UPDATE).
- Case close: idempotent double-close, close-vs-inbound-event race (both orderings produce a consistent, non-corrupting result).
- Resolution → Case.contact_id derivation: create, replace (soft-delete + insert), delete, reconcile CLI idempotency.
- Unresolved queue correctness under all three closing outcomes (§6).
- CaseNote: never appears in customer-facing Interaction list or webhook payload (explicit negative test).
- Cross-channel explicit-hint scenario: call-active SMS lands in the same case_id as the call.

## 11. Rollback

No destructive migration. All new columns are nullable and all new tables are additive. Rollback is a straightforward revert-and-redeploy: existing Interaction/Resolution/Conversation behavior is untouched by this feature, since Case sits alongside them rather than replacing any existing write path.
