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
| Priority field | Meaningless without queue/routing integration; agents can use `bin-tag-manager` tags as an informal priority signal in the interim (e.g. an `urgent` tag) until real routing ships | Queue-based case routing is designed |
| SLA timers / CSAT surveys | Separate feature surface, non-trivial | Explicit product requirement |
| Case assignment history table beyond event stream | `case_updated` event stream is a de-facto audit log | Ops needs to query without scanning event archives |
| Same-channel case merge (two accidentally-separate open cases for the same peer, e.g. from the timeout-race edge case in §4.2) | `ClosedReason='merged'` is reserved in the schema (§3.1) but the merge action itself is undesigned | A real duplicate-case incident is observed in practice |
| Cross-channel case history surfacing (e.g. "this contact also has a recent call case" shown on an SMS case) | Frontend/UX concern; the backend already exposes everything needed via the Interaction timeline and `Case.contact_id`, once resolved | square-admin case detail view design (§9) |
| Case-level custom metadata (arbitrary key/value fields per customer, e.g. "order number") | No schema-flexible field is designed yet; needs its own shape decision (JSON blob vs. structured) rather than being bolted on here | A concrete early-adopter customization request |
| Case-level reporting/analytics rollup (handle time, cases-per-agent, resolution rate) | `opened_at`/`closed_at`/`closed_reason`/webhook events already carry everything needed for a customer to build this themselves; a first-party rollup endpoint is a separate, non-trivial feature | Customers ask for a built-in dashboard rather than building on the webhook stream |
| `CaseNote` mentions/notifications beyond the `case_note_created` event (§3.5) | The event is a cheap, generic hook; a first-class mention data model + delivery guarantee is a separate design | Agent UI needs guaranteed mention delivery, not best-effort client-side scanning |
| `CaseNote` attachments and rich text | Plain text is sufficient for v1 internal notes | Explicit agent workflow need for embedding files/formatting in notes |

## 3. Data model

### 3.1 `Case` (new, owned by `bin-contact-manager`)

```go
type Case struct {
    ID         uuid.UUID
    CustomerID uuid.UUID

    PeerType     commonaddress.Type // reuse commonaddress.Type (tel, email, whatsapp, ...)
    PeerTarget   string             // normalized via commonaddress.NormalizeTarget — bit-identical to
                                     // contact_addresses.target and interaction.peer_target
    ReferenceType string            // reuses contact_interactions.reference_type's EXISTING stored
                                     // vocabulary as written today by contacthandler/interaction.go
                                     // ("call", "conversation_message", ...) — NOT conversation-manager's
                                     // message.ReferenceType (a different, unrelated enum with different
                                     // values). Case.ReferenceType must match Interaction.ReferenceType
                                     // exactly for the §4 join to work; this is the single correct source.

    ContactID *uuid.UUID // nullable; denormalized cache, see §3.3

    commonidentity.Owner // OwnerType + OwnerID — reused as-is from the conversation assignment precedent

    Status       string     // open | closed
    OpenedAt     *time.Time // always set at INSERT time (no code path leaves it NULL); kept as a
                             // pointer only to match the project's general nullable-timestamp
                             // convention, not because it can actually be absent
    ClosedAt     *time.Time
    ClosedReason string     // agent_closed | timeout | merged (merged reserved, unused until phase 2)
    ClosedByType string     // agent | system
    ClosedByID   *uuid.UUID

    PreviousCaseID *uuid.UUID // re-contact chain; nil for the first case with a given peer

    TMCreate *time.Time
    TMUpdate *time.Time
}
```

Constraint — **MySQL/MariaDB does not support partial/filtered unique indexes** (this platform runs MySQL exclusively via `bin-dbscheme-manager`, per its own prior precedent in `ac5d4e18060c_contact_crm_create_tables.py` for `contact_addresses.primary_contact_uk`). The identical generated-column technique is reused here instead of the Postgres-style `WHERE`-clause index from the first draft of this doc:

```sql
-- open_peer_uk carries a value only when status='open'; closed rows compute NULL,
-- which MySQL treats as distinct under UNIQUE, so any number of closed rows for the
-- same peer/reference_type may coexist, while at most one open row may exist.
open_peer_uk BINARY(?) GENERATED ALWAYS AS (
    IF(status = 'open',
       CONCAT(customer_id, '|', peer_type, '|', peer_target, '|', reference_type),
       NULL)
) STORED,

UNIQUE INDEX uq_case_open_peer (open_peer_uk);
```

(Concrete column sizing/hashing approach to be finalized at implementation time — e.g. a `SHA2(...)` digest if the concatenated key exceeds a practical index-key length — but the generated-column + distinct-NULL mechanism is the load-bearing part carried over from the existing precedent.)

At most one **open** Case per (customer, peer, reference_type). Closed cases are unconstrained; many can accumulate for the same peer over time, chained by `previous_case_id`.

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
    InteractionID *uuid.UUID // CHANGED from non-nullable uuid.UUID to nullable — see migration note below
    CaseID        *uuid.UUID // nullable — new case-level attribution path
}
```

**Breaking correction from the first draft:** `Resolution.InteractionID` is `uuid.UUID` (non-pointer) and `NOT NULL` in the schema shipped today (`models/resolution/resolution.go:18`, migration `ac5d4e18060c...py:132`). The case-level attribution mode (`case_id` set, `interaction_id` absent) is impossible to insert unless `interaction_id` is altered to nullable first. The migration in §8 explicitly includes this `ALTER COLUMN` as its own numbered step — it is not implied by adding `case_id`.

```sql
ALTER TABLE contact_resolutions MODIFY COLUMN interaction_id BINARY(16) DEFAULT NULL;
ALTER TABLE contact_resolutions ADD CONSTRAINT chk_resolution_case_or_interaction
  CHECK (interaction_id IS NOT NULL OR case_id IS NOT NULL);

-- Same MySQL generated-column technique as §3.1 (no native partial unique index):
case_positive_uk BINARY(16) GENERATED ALWAYS AS (
    IF(resolution_type = 'positive' AND interaction_id IS NULL AND tm_delete IS NULL,
       case_id, NULL)
) STORED,
UNIQUE INDEX uq_resolution_case_positive (case_positive_uk);
```

A Resolution row now has two independent modes:
- **`case_id` set** (primary path): "this whole case belongs to this contact." All Interactions carrying this `case_id` inherit the attribution.
- **`interaction_id` set** (exception path, existing behavior): fine-grained override of a single Interaction — used to add/remove one message from an otherwise-correct case attribution.

The generated-column unique index guarantees at most one active case-level positive Resolution per case, which is required for `Case.contact_id` (§3.4) to be derivable without ambiguity.

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

A `case_note_created` webhook event is emitted alongside `CaseNote` creation (same convention as `case_created`/`case_updated`), primarily so an agent UI can implement @-mention notifications by scanning `text` for mentions client-side or in a thin subscriber — this is deliberately the only notification mechanism at launch; a first-class mentions/notify data model is parked (§2). Rich formatting and attachments on `CaseNote` are also parked (§2), not silently dropped.

## 4. Case get-or-create

Triggered from the same projection points that create Interactions today (`EventCallCreated`, `EventConversationMessageCreated`, and channel-specific equivalents), before the Interaction insert.

```
BEGIN TRANSACTION

1. IF the triggering event carries an explicit case_id hint (§4.1):
     SELECT * FROM contact_cases WHERE id=? AND customer_id=? AND status='open' FOR UPDATE
     IF found: case_id = hint value  -- validated: correct tenant, still open
     ELSE: fall through to the peer/reference_type path below as if no hint were given
           (stale/invalid/closed hint never silently attaches an Interaction to the
           wrong case or a closed one — see §4.1 for why this validation is mandatory)

   ELSE (or hint invalid, falling through):
     SELECT * FROM contact_cases
     WHERE customer_id=? AND peer_type=? AND peer_target=? AND reference_type=? AND status='open'
     FOR UPDATE

     IF found AND (now - tm_update) < CASE_TIMEOUT_HOURS:
         case_id = found.id   -- reuse

     ELSE IF found (timed out):
         UPDATE contact_cases SET status='closed', closed_at=now, closed_reason='timeout'
         WHERE id = found.id
         TRY:
           INSERT INTO contact_cases (..., status='open', opened_at=now, previous_case_id=found.id)
           RETURNING id INTO case_id
         ON DUPLICATE KEY (uq_case_open_peer):
           -- another transaction's insert for the same peer won the race between our
           -- UPDATE...closed and our INSERT (see §4.2); re-select the now-open row
           -- instead of erroring
           SELECT id FROM contact_cases WHERE customer_id=? AND peer_type=? AND peer_target=?
             AND reference_type=? AND status='open' INTO case_id

     ELSE (not found):
         last_closed = SELECT * FROM contact_cases
                       WHERE customer_id=? AND peer_type=? AND peer_target=? AND reference_type=?
                       ORDER BY tm_create DESC LIMIT 1
         TRY:
           INSERT INTO contact_cases (..., status='open', opened_at=now,
                  previous_case_id = last_closed.id if exists else NULL)
           RETURNING id INTO case_id
         ON DUPLICATE KEY (uq_case_open_peer):
           -- a concurrent first-contact insert for the same peer won; reuse it
           -- rather than erroring (see §4.2)
           SELECT id FROM contact_cases WHERE customer_id=? AND peer_type=? AND peer_target=?
             AND reference_type=? AND status='open' INTO case_id

2. Attempt contact auto-match via existing address-set lookup (same mechanism as
   Interaction's automatic contact matching); if matched, set contact_id on the new Case row.
   (Skipped when case_id came from the §4.1 hint path — the case's contact_id is already
   established from whenever that case was originally opened.)

3. INSERT INTO contact_interactions (..., case_id=case_id)

4. UPDATE contact_cases SET tm_update=now WHERE id=case_id

COMMIT
```

### 4.2 Concurrency correctness

`FOR UPDATE` on an existing open row correctly serializes concurrent transactions targeting that row — any process issuing the same `SELECT ... FOR UPDATE` blocks on the DB-level row lock regardless of pod/instance boundary, so the "reuse an existing open case" branch is race-free as stated.

The **insert** branches (first contact, and timeout-triggered re-open) are a distinct race class: two transactions can simultaneously conclude "no open row exists" (nothing to lock when the row doesn't exist yet, or — for the timeout branch — a lock granted after a concurrent commit re-evaluates the `WHERE status='open'` predicate to zero rows and falls through to the same "not found" path). Both then attempt `INSERT`, and the unique index (`uq_case_open_peer`, §3.1) correctly rejects the second one as a duplicate key — but only if the pseudocode handles that rejection. The `ON DUPLICATE KEY` branches above are the fix: on a unique-constraint violation, re-select the row the other transaction just inserted and reuse it, rather than surfacing the DB error to the caller. This makes the full get-or-create race-free **with** a retry-on-conflict step, not "race-free without a retry loop" as an earlier draft of this section claimed — the corrected claim is: no distributed lock or advisory lock is needed, but the insert path must handle its own unique-constraint collision.

### 4.3 Explicit case_id override

**Problem it solves:** an agent mid-call sends an SMS to the same customer. Naive peer-based matching would key the SMS event by `reference_type=message`, landing it in a *different* Case than the active `reference_type=call` Case — even though both are part of the same engagement the agent is actively working.

**Resolution:** actions taken from an already-known case context (e.g. the agent's UI already displays `call_id` → `case_id` for the active call) must propagate that `case_id` explicitly through to the outbound action, rather than relying on the receiving side to re-infer it from peer address. This follows the "reuse the value the caller already holds, don't re-derive" convention used elsewhere in the codebase (e.g. `DeriveEndpoints`).

Priority order is strict:

```
1. Explicit case_id hint present  -> validate (customer_id match, status='open', see §4 step 1)
                                      -> valid: use it, skip peer/reference_type matching entirely
                                      -> invalid/stale/closed: fall through to step 2
2. No hint (or hint fell through) -> (customer_id, peer_type, peer_target, reference_type) get-or-create
```

**Validation is mandatory, not optional.** An unvalidated hint is both a cross-tenant leak vector (a stale or wrong cached `case_id` could attribute an Interaction to a different customer's case) and a direct violation of §5.3's closed-case-immutability invariant (a hint referencing a case that closed between when the agent's UI cached it and when the SMS was actually sent would otherwise insert into a closed case). §4 step 1's `SELECT ... WHERE id=? AND customer_id=? AND status='open'` is the single point where this is enforced — a hint that fails this check is treated exactly as if no hint were supplied, never as an error that blocks the underlying action (the SMS still gets sent and projected; it just lands in whatever case the normal peer-based path resolves to).

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

The `WHERE status='open'` guard makes this an idempotent, race-tolerant optimistic update for the **double-close** case: two identical close requests racing each other resolve safely (the second is a harmless 0-row no-op) without a distributed lock.

**This guard alone is not sufficient for the close-vs-timeout race**, which is a materially different scenario: an inbound event's lazy timeout evaluation (§5.2, `closed_reason='timeout'`, `closed_by_type='system'`) and an agent's explicit `POST /close` (`closed_reason='agent_closed'`) can race for the same row. Whichever commits first wins the row; the loser's `UPDATE ... WHERE status='open'` matches zero rows. Silently treating that 0-row result as success would let an agent believe they closed the case with `agent_closed` while the persisted `closed_reason` is actually `timeout` — a real divergence between what the actor believes happened and what is recorded, not merely a "which case does the borderline message land in" question.

**Resolution:** the close handler MUST distinguish "0 rows because already closed by someone/something else" from "0 rows because the ID doesn't exist," and MUST return the **actual persisted** `closed_reason`/`closed_by` in the response rather than assuming the caller's own action won:

```
UPDATE contact_cases SET status='closed', closed_at=now, closed_reason='agent_closed',
       closed_by_type=?, closed_by_id=? WHERE id=? AND status='open'

IF rows_affected == 1:
    return 200 { closed_reason: 'agent_closed', closed_by_type: 'agent', closed_by_id: ... }
ELSE:
    existing = SELECT closed_reason, closed_by_type, closed_by_id FROM contact_cases WHERE id=?
    IF existing.status != 'closed':  -- id genuinely doesn't exist / not yet closed by anything
        return 404
    ELSE:
        -- already closed by someone/something else; surface the TRUTH, not the caller's intent
        return 200 { closed_reason: existing.closed_reason, closed_by_type: existing.closed_by_type,
                     closed_by_id: existing.closed_by_id, already_closed: true }
```

This keeps §5.3's audit invariant genuinely unambiguous (the response always reflects what's actually persisted) instead of merely treating the race as a no-op that papers over a possible attribution mismatch.

### 5.2 Timeout

Evaluated **lazily** at the next inbound event for that peer (§4), not via a scheduled sweep. A scheduled job would produce `timeout`-closed events with no corresponding user action to explain them and adds an operational component for no real benefit at current scale.

`CASE_TIMEOUT_HOURS` — platform-wide env var, default `24`. No per-customer override (see §2 parked items). Follows the existing `bin-ai-manager` precedent (`AIcallConversationIdleTimeoutHours`) for both the config shape and the `SetXXXForTest` test-override helper convention.

### 5.3 No reopen

A closed Case is immutable. Re-contact from the same peer always creates a new Case with `previous_case_id` pointing at the prior one — regardless of how soon after closing the re-contact happens. This keeps "who closed this and when" permanently unambiguous, which matters for reporting on case handling time and for the closing-agent attribution the platform treats as a hard invariant. The cost is a `reopen`-style UX is not offered; agents needing continuity rely on the `previous_case_id` chain to see history.

**Accidental-close recovery.** The chain above only forms on the *next inbound event* from that peer. If an agent closes a case by mistake and the customer never re-contacts, there is no natural trigger to create a follow-up case — the agent has no path back into a "working" state for that engagement. To close this gap without compromising the immutability invariant above (the mistakenly-closed case's `closed_by`/`closed_at` are never altered), an agent-initiated manual continuation is included in the API surface:

```
POST /v1/cases/{id}/continue
```

Creates a new, empty `open` Case with `previous_case_id = id`, using the same `(peer_type, peer_target, reference_type, contact_id)` as the source case. Requires the source case to be `status='closed'`; requires the caller to be the case's owning agent or admin/manager (same permission shape as case assignment, §7). This is a distinct action from "reopening" the closed case itself — the old case stays exactly as it was, permanently attributed to whoever/whatever closed it; only a *new* case is created.

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

1. Create `contact_cases` table (with the generated-column-based unique index, §3.1 — not a native partial/filtered index, per the MySQL constraint noted there).
2. Add `case_id` (nullable) to `contact_interactions`.
3. Add `case_id` (nullable) to `contact_resolutions`; alter `interaction_id` to nullable (§3.3 — a real schema change from today's `NOT NULL`, not additive-only); add the CHECK constraint and the generated-column-based unique index.
4. Create `contact_case_notes` table.
5. Register `case` as a resource type in `bin-tag-manager`.
6. **No backfill.** Historical Interactions predate the Case concept; retroactively grouping them by peer+time-gap heuristics would invent case boundaries that never existed operationally and could misattribute historical activity. The feature's value is forward-looking case management, not historical reconstruction.

Step 3's `interaction_id` nullability change is the one non-purely-additive change in this migration; it relaxes a constraint (loosens `NOT NULL` to nullable) rather than tightening one, so it cannot reject previously-valid data — every existing row already has a non-null `interaction_id` and continues to satisfy the new, looser constraint unchanged.

## 9. Service/API surface impact

- **bin-contact-manager**: `CaseHandler` (get-or-create, close with truthful-persisted-state response per §5.1, continue, assign/unassign, list, list-unresolved), `CaseNoteHandler` (create/list/delete, emits `case_note_created`), Resolution handler extended with the case-level derivation hook, `case-control` CLI (including `reconcile-contact`).
- **bin-tag-manager**: register `case` as a taggable resource type (no schema change expected — generic tagging already supports arbitrary resource types).
- **bin-api-manager**: `/v1.0/cases`, `/v1.0/cases/{id}`, `/v1.0/cases/{id}/close`, `/v1.0/cases/{id}/continue`, `/v1.0/cases/{id}/notes`; extend `/v1.0/interactions` responses with `case_id`.
- **bin-openapi-manager**: spec additions per `voipbin-openapi-spec-handler-parity` convention.
- **square-admin**: case list / unresolved queue / close button / notes panel — separate frontend design, not covered here.

## 10. Test scope (high level; detailed table-driven cases in the implementation plan)

- Case get-or-create: reuse on open match, timeout-triggered chain with duplicate-key retry (§4.2), first-contact concurrent-insert retry (§4.2), explicit-hint bypass with valid/stale/wrong-tenant/closed hint (§4.3).
- Case close: idempotent double-close, close-vs-timeout race returns the actually-persisted `closed_reason`/`closed_by` rather than the caller's assumed outcome (§5.1), `continue` on a closed case creates a correctly-chained new case without mutating the source.
- Resolution → Case.contact_id derivation: create, replace (soft-delete + insert), delete, reconcile CLI idempotency, insert rejected when neither `case_id` nor `interaction_id` is set (CHECK constraint).
- Unresolved queue correctness under all three closing outcomes (§6).
- CaseNote: never appears in customer-facing Interaction list or webhook payload (explicit negative test); `case_note_created` event fires on creation.
- Cross-channel explicit-hint scenario: call-active SMS lands in the same case_id as the call; a stale/closed hint falls back to normal peer-based matching rather than erroring or attaching to the wrong case.

## 11. Rollback

No destructive migration. All new columns are nullable and all new tables are additive. Rollback is a straightforward revert-and-redeploy: existing Interaction/Resolution/Conversation behavior is untouched by this feature, since Case sits alongside them rather than replacing any existing write path.
