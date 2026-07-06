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

- `Case` entity: get-or-create keyed by `(customer_id, peer_type, peer_target, reference_type)`, with explicit-context override for same-session cross-channel actions (e.g. agent sends SMS mid-call), and a symmetric inbound-side hint carried via `bin-conversation-manager`'s `Conversation.Metadata` for the far more common case of a customer replying mid-call (§4.4).
- Case lifecycle: open → closed (`agent_closed` / `timeout` / `merged`-reserved). No reopen; re-contact creates a new Case linked via `previous_case_id`. Agent-initiated `/continue` for accidental-close recovery.
- `Interaction.case_id` (nullable FK) — Interaction stays the event log; Case is the new grouping layer.
- `Resolution.case_id` (nullable) — case-level contact attribution, promoted to first-class alongside the existing interaction-level override. This requires relaxing `Resolution.InteractionID` to nullable, which is a real Go-side breaking change, not just a DB `ALTER` — see §3.3.
- `Case.contact_id` — denormalized cache, single source of truth is Resolution, single derivation function.
- Case ownership (`owner_type`/`owner_id`) reusing the existing `commonidentity.Owner` pattern (see `assignable-conversation-design.md`); ownership is never cleared by closing (§7).
- Case tagging via the existing `bin-tag-manager` (Case registered as a taggable resource type).
- `CaseNote` — new, physically separate table for agent-internal notes. Never customer-facing, including at the event-delivery layer (§3.5).
- Explicit case_id propagation for actions taken inside an active case context (e.g. outbound SMS sent while a call is active), plus the symmetric inbound-side mechanism via `bin-conversation-manager`'s new `Conversation.Metadata.ContactCaseID` field (§4.4) — this closes a real gap found in review: an inbound customer message during an active call has no channel to carry a case-matching hint the way an agent-initiated outbound action does.

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
| Real-time agent-facing delivery of `case_note_created` (e.g. a live notification badge when a teammate adds a note) | No internal-only real-time channel to agents exists today independent of the customer webhook pipeline (verified against `bin-api-manager/pkg/subscribehandler` in round-3 review); building one is genuinely new infrastructure, not a reuse of something that already exists | An agent UI requirement for live note notifications, at which point the new internal delivery lane gets its own design |

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

    commonidentity.Owner // OwnerType + OwnerID — reused as-is from the conversation assignment precedent;
                          // NEVER cleared by closing a case (§7) — this is a load-bearing invariant for
                          // /continue's authorization (§5.3)

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

Constraint — **MySQL/MariaDB does not support partial/filtered unique indexes** (this platform runs MySQL exclusively via `bin-dbscheme-manager`, per its own prior precedent in `ac5d4e18060c_contact_crm_create_tables.py` for `contact_addresses.primary_contact_uk`). The identical generated-column technique is reused here, with a fixed-size hash digest rather than a raw `CONCAT` (see rationale below) to avoid the truncation risk a raw-value approach would carry:

```sql
-- open_peer_uk carries a value only when status='open'; closed rows compute NULL,
-- which MySQL treats as distinct under UNIQUE, so any number of closed rows for the
-- same peer/reference_type may coexist, while at most one open row may exist.
--
-- SHA2(..., 256) is used INSTEAD OF a raw CONCAT of the key fields. peer_type/peer_target/
-- reference_type are VARCHAR(255) under utf8mb4 (up to 4 bytes/char); a raw concatenation of
-- customer_id + all three fields can approach ~3KB in the worst case, close to or over
-- InnoDB's index key-length limits, and MySQL SILENTLY TRUNCATES an oversized value assigned
-- into a fixed-size BINARY column rather than erroring — which could create false-positive
-- uniqueness collisions between genuinely different (customer, peer, reference_type) tuples.
-- A fixed-size hash input eliminates that risk outright; this decision is made now, not
-- deferred to implementation, per round-2 design review.
open_peer_uk BINARY(32) GENERATED ALWAYS AS (
    IF(status = 'open',
       UNHEX(SHA2(CONCAT_WS('|', customer_id, peer_type, peer_target, reference_type), 256)),
       NULL)
) STORED,

UNIQUE INDEX uq_case_open_peer (open_peer_uk),

-- Supporting indexes for the hot-path queries this design specifies elsewhere:
INDEX idx_case_unresolved (customer_id, status, contact_id),   -- backs §6 CaseListUnresolved
INDEX idx_case_owner      (customer_id, owner_type, owner_id); -- backs §7 "my cases" list
```

At most one **open** Case per (customer, peer, reference_type). Closed cases are unconstrained; many can accumulate for the same peer over time, chained by `previous_case_id`. `CONCAT_WS` (not `CONCAT`) is used so a `NULL` component (none of these fields are nullable today, but this is defensive) produces a distinguishable hash input rather than collapsing the whole expression to `NULL`.

### 3.2 `Interaction` (existing, one new column)

```go
type Interaction struct {
    // ... existing fields unchanged (VOIP-1208 design) ...
    CaseID *uuid.UUID // nullable FK -> Case.id; nil for pre-Case historical rows, always set going forward
}
```

No backfill of historical Interactions into synthetic Cases — see §8.

### 3.3 `Resolution` (existing, extended) — including the Go-side breaking change

```go
type Resolution struct {
    // ... existing fields unchanged ...
    InteractionID *uuid.UUID // CHANGED from non-nullable uuid.UUID to nullable
    CaseID        *uuid.UUID // nullable — new case-level attribution path
}
```

**This is not a pure additive change — it is a breaking change to existing Go call sites, and the implementation plan must scope it as its own work item, not treat it as implied by the DB `ALTER`.** Concretely, today `models/resolution/resolution.go:18` declares `InteractionID uuid.UUID` (non-pointer), and callers rely on that non-nullability as a scoping parameter, not merely a type:

- `pkg/dbhandler/resolution.go` — `ResolutionDelete` / `ResolutionListByInteraction` take `interactionID uuid.UUID` as a **mandatory row-scoping parameter** used directly in `WHERE interaction_id = ?`. A case-level Resolution (interaction_id absent) has nothing to pass here — these functions need a `case_id`-scoped counterpart (`ResolutionListByCase`, and a `ResolutionDelete` variant that accepts either scope), not just a widened signature.
- `pkg/contacthandler/resolution.go` — propagates the same non-nullable signature upward; needs updating in lockstep.
- `pkg/contacthandler/interaction_read.go` — builds a `map[uuid.UUID]bool` keyed directly off `r.InteractionID` with no nil-check; this must be guarded once `InteractionID` can be nil.
- All existing table-driven tests in `dbhandler/resolution_test.go` and `contacthandler/resolution_test.go` need new nil-`InteractionID` cases.

The DB migration itself is additive-safe (§8), but the surrounding Go code requires a real refactor. This is called out explicitly here so the implementation plan sizes it correctly rather than discovering it mid-implementation.

```sql
ALTER TABLE contact_resolutions MODIFY COLUMN interaction_id BINARY(16) DEFAULT NULL;
ALTER TABLE contact_resolutions ADD CONSTRAINT chk_resolution_case_or_interaction
  CHECK (interaction_id IS NOT NULL OR case_id IS NOT NULL);

-- Same MySQL generated-column + SHA2 technique as §3.1 (no native partial unique index,
-- and case_id alone is a 16-byte UUID so no truncation risk here — SHA2 is used purely
-- for consistency with §3.1, a raw case_id-keyed generated column would also be safe):
case_positive_uk BINARY(16) GENERATED ALWAYS AS (
    IF(resolution_type = 'positive' AND interaction_id IS NULL AND tm_delete IS NULL,
       case_id, NULL)
) STORED,
UNIQUE INDEX uq_resolution_case_positive (case_positive_uk);
```

A Resolution row now has two independent modes:
- **`case_id` set** (primary path): "this whole case belongs to this contact." All Interactions carrying this `case_id` inherit the attribution.
- **`interaction_id` set** (exception path, existing behavior): fine-grained override of a single Interaction — used to add/remove one message from an otherwise-correct case attribution.

The generated-column unique index guarantees at most one active case-level positive Resolution per case. Creating a case-level positive Resolution when one already exists (uncommon, but not impossible under concurrent agent action) is a duplicate-key situation with exactly the same shape as the Case-insert races in §4 — the handler MUST apply the identical `ON DUPLICATE KEY` reuse pattern (re-select the existing active resolution and treat the call as a no-op/already-resolved) rather than surfacing a raw DB error.

The **soft-delete-then-insert "replace"** flow (used when correcting a case's contact attribution) MUST run both statements plus the `Case.contact_id` derivation write (§3.4) inside a **single transaction**. Splitting them across separate transactions would let the case transiently derive to `contact_id = NULL` between the delete and the insert, causing it to flicker into and out of §6's unresolved queue — not data corruption, but a visible inconsistency this design explicitly rules out by requiring single-transaction atomicity for the replace operation.

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

1. **Write path** — invoked inside the same transaction whenever a case-level Resolution is created, soft-deleted, or replaced (see §3.3's atomicity requirement for the replace case specifically). `Case.contact_id` is written directly from the function's result; no re-derivation elsewhere.
2. **Recovery path** — `case-control` CLI command `case reconcile-contact <case_id | --all>` re-runs the same function and overwrites `Case.contact_id`. Idempotent. Used only if drift is discovered (e.g. a bulk import wrote Resolution rows without going through the handler). No scheduled reconciliation job at this stage — added only after a real drift incident, per platform incident-response convention.

Concurrent closing (§5.1) and resolving (this section) of the same Case do not race destructively: both are row-level `UPDATE`s on the same `Case` row, so InnoDB's row lock serializes them regardless of which columns each statement touches. Whichever commits second still lands correctly — closing never clobbers `contact_id`, and resolving never clobbers `status`/`closed_*`. §6's explicit support for retroactively resolving a closed Case depends on, and is consistent with, this guarantee.

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

**Physically separate from `Interaction`.** CaseNote is never surfaced in any customer-facing webhook or API response. Mixing internal notes into the Interaction timeline (e.g. via `ReferenceType: "note"`) is explicitly rejected — it would require every consumer of the customer-facing timeline to correctly filter internal rows, and a single missed filter becomes a customer-visible data leak. A separate table makes that class of bug structurally impossible **at the storage layer** — but the guarantee must also hold at the event-delivery layer, which the first draft of this section got wrong (corrected below).

**`case_note_created` delivery — corrected from the first draft, and corrected again after round-3 review.** An earlier draft said this event is published "same convention as `case_created`/`case_updated`." Both of those ARE customer-facing today, delivered via `notifyHandler.PublishWebhookEvent()` → webhook-manager → the customer's configured webhook endpoint, unconditionally (the only guard is `customerID == uuid.Nil`). Publishing `case_note_created` through that same path would leak internal note text to the customer webhook endpoint — a real, self-contradictory defect the first draft had.

The next draft then claimed `case_note_created` could instead ride "the same mechanism agent-manager or conversation-manager use for agent-facing real-time updates" — this claim was checked against the actual code in round 3 and is **false**: `bin-api-manager/pkg/subscribehandler/main.go`'s `processEvent` switch, which is the only thing that feeds the agent-facing WebSocket bridge (`pkg/websockhandler`), has exactly one case (`ServiceNameWebhookManager` + `EventTypeWebhookPublished`); every other publisher/event type — including bare `PublishEvent`-only calls like `agent.EventTypeAgentUpdated` — falls through `default: return` and is silently dropped. There is no existing internal-only real-time channel to agents that bypasses the customer webhook pipeline. Even the chat/talk real-time fan-out (`createTopics` in `subscribehandler/webhookmanager.go`) is invoked *from inside* the webhook-published event handler, not a separate lane.

**Corrected, honest scope for this design:** real-time agent notification for new `CaseNote`s (the original motivation for `case_note_created`) is **out of scope for this design** (added to the parked table, §2) — building it requires genuinely new infrastructure (e.g. extending the `subscribehandler` switch with a new non-webhook-sourced case, or a new internal ZMQ/pubsub lane from `bin-contact-manager`), which is real, unscoped work that does not exist to "reuse" and should not be implied here. What this design DOES commit to: `CaseNote` creation publishes a bare `case_note_created` event via the plain `notifyHandler.PublishEvent()` primitive (RabbitMQ pub/sub only, **not** `PublishWebhookEvent`) — the same primitive `bin-agent-manager` uses for `agent_updated` — purely as an internal audit/future-consumption signal. This event is **not** currently delivered anywhere (it is dropped by `subscribehandler`'s `default` branch exactly like `agent_updated` is today), which is the safe default: it cannot leak to a customer webhook because it never enters the webhook-manager pipeline, and mentions/notifications remain fully parked (§2) rather than half-implemented against a channel that doesn't reach anyone yet. §10 includes an explicit negative test asserting `case_note_created` is never routed through `PublishWebhookEvent` / never reaches the customer webhook delivery mechanism.

Rich formatting, attachments, and real-time agent-facing delivery of `CaseNote` are all parked (§2), not silently dropped.

## 4. Case get-or-create

Triggered from the same projection points that create Interactions today (`EventCallCreated`, `EventConversationMessageCreated`, and channel-specific equivalents), before the Interaction insert.

```
BEGIN TRANSACTION

1. IF the triggering event carries an explicit case_id hint (§4.3):
     SELECT * FROM contact_cases WHERE id=? AND customer_id=? AND status='open' FOR UPDATE
     IF found: case_id = hint value  -- validated: correct tenant, still open
     ELSE: fall through to the peer/reference_type path below as if no hint were given
           (stale/invalid/closed hint never silently attaches an Interaction to the
           wrong case or a closed one — see §4.3 for why this validation is mandatory)

   ELSE (or hint invalid, falling through):
     SELECT * FROM contact_cases
     WHERE customer_id=? AND peer_type=? AND peer_target=? AND reference_type=? AND status='open'
     FOR UPDATE

     IF found AND (now - tm_update) < CASE_TIMEOUT_HOURS:
         case_id = found.id   -- reuse

     ELSE IF found (timed out):
         UPDATE contact_cases SET status='closed', closed_at=now, closed_reason='timeout'
         WHERE id = found.id
         LOOP (bounded, e.g. max 3 attempts — see §4.2 for why a loop, not a single retry):
           TRY:
             INSERT INTO contact_cases (..., status='open', opened_at=now, previous_case_id=found.id)
             RETURNING id INTO case_id
             BREAK
           ON DUPLICATE KEY (uq_case_open_peer):
             SELECT * FROM contact_cases WHERE customer_id=? AND peer_type=? AND peer_target=?
               AND reference_type=? AND status='open' FOR UPDATE
             IF found and still 'open': case_id = found.id; BREAK
             ELSE: retry the INSERT (the row we just raced against may itself have since
                   closed/timed out; loop rather than assume the first re-select is final)

     ELSE (not found):
         last_closed = SELECT * FROM contact_cases
                       WHERE customer_id=? AND peer_type=? AND peer_target=? AND reference_type=?
                       ORDER BY tm_create DESC LIMIT 1
         LOOP (bounded, same shape as above):
           TRY:
             INSERT INTO contact_cases (..., status='open', opened_at=now,
                    previous_case_id = last_closed.id if exists else NULL)
             RETURNING id INTO case_id
             BREAK
           ON DUPLICATE KEY (uq_case_open_peer):
             SELECT * FROM contact_cases WHERE customer_id=? AND peer_type=? AND peer_target=?
               AND reference_type=? AND status='open' FOR UPDATE
             IF found AND still 'open': case_id = found.id; BREAK
             ELSE: retry the INSERT

2. Attempt contact auto-match via existing address-set lookup (same mechanism as
   Interaction's automatic contact matching); if matched, set contact_id on the new Case row.
   (Skipped when case_id came from the §4.3 hint path — the case's contact_id is already
   established from whenever that case was originally opened.)

3. INSERT INTO contact_interactions (..., case_id=case_id)

4. UPDATE contact_cases SET tm_update=now WHERE id=case_id

COMMIT
```

### 4.2 Concurrency correctness

`FOR UPDATE` on an existing open row correctly serializes concurrent transactions targeting that row — any process issuing the same `SELECT ... FOR UPDATE` blocks on the DB-level row lock regardless of pod/instance boundary, so the "reuse an existing open case" branch is race-free as stated.

The **insert** branches (first contact, and timeout-triggered re-open) are a distinct race class: two transactions can simultaneously conclude "no open row exists" and both attempt `INSERT`, which the unique index (`uq_case_open_peer`, §3.1) correctly rejects for the loser as a duplicate key.

**Round-2 correction: the retry-select itself must be locked and re-checked, not treated as final.** An earlier draft of this section used a bare `SELECT` (no `FOR UPDATE`) in the `ON DUPLICATE KEY` branch, on the assumption that re-reading the winning row was sufficient. That is not safe: between that unlocked read and this transaction's later steps (3: `INSERT INTO contact_interactions`, 4: `UPDATE tm_update`), a third actor could close the winning case via `POST /close` (§5.1) with nothing to block it, resulting in an Interaction inserted into (and `tm_update` bumped on) a case that is now `status='closed'` — silently violating §5.3's closed-case-immutability invariant. The corrected pseudocode above fixes this two ways: (a) the retry-select takes `FOR UPDATE`, extending this transaction's lock to the winning row so no other transaction can close it out from under us before we commit, and (b) the retry is a **bounded loop**, not a single re-select — if the row we raced against itself transitions out of `open` before we re-select it (a second, rarer race), we fall through and retry the insert again rather than proceeding with a stale assumption.

This makes the full get-or-create race-free with a bounded retry-on-conflict loop, not a single retry — the earlier "race-free without a retry loop" framing understated the mechanism required; the corrected claim is: no distributed or advisory lock is needed, but the insert path must both retry on conflict and re-lock what it finds on retry.

**Loop exhaustion (all bounded attempts fail):** this is an extremely rare thundering-herd scenario (repeated collision across all retry attempts for the same peer). The handler surfaces a transient 5xx error to the caller rather than silently dropping the event; the triggering event's at-least-once delivery semantics (already relied upon elsewhere in the platform, e.g. `contact_interactions`' idempotency key) ensure the caller/queue redelivers and retries the whole operation. No special-cased fallback logic is introduced for this path.

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

### 4.4 Inbound hint gap and its fix: conversation metadata

**The gap.** §4.3's hint mechanism only works in the **outbound** direction — it requires a caller that already knows `case_id` to explicitly pass it along (e.g. the agent UI passing the active call's `case_id` when it triggers an SMS send). This has no symmetric counterpart for **inbound** messages: when a customer sends an SMS mid-call (or shortly after), the inbound SMS event is generated by `message-manager` from a raw provider webhook, with no way for the customer's phone to carry a VoIPBin-internal `case_id`. Left unaddressed, an inbound SMS during an active call would fall through to §4's normal peer/reference_type matching, find no open `reference_type='message'` case, and spin up a **new, disconnected** SMS case — breaking the very continuity §4.3 was designed to preserve, and doing so on the far more common direction (customers reply far more often than agents proactively text mid-call).

**The fix — reuse `bin-conversation-manager`, not a new cross-service lookup.** `bin-conversation-manager` already maintains a stable `Conversation` row per `(customer_id, self, peer)` via `GetOrCreateBySelfAndPeer` (`pkg/conversationhandler/db.go`), and it already loads that `Conversation` on **every** inbound message (`MessageEventReceived`) to decide whether to trigger an activeflow or route to an assigned agent (`assignable-conversation-design.md`). This is the natural place to carry a case-matching hint through to the inbound path at near-zero cost — no new cross-service RPC per inbound message, just a new field read off a row `bin-conversation-manager` was already loading.

Concretely, this requires a **new, generic `Metadata` field on `Conversation`** (not a dedicated `contact_case_id` column, and explicitly not tied to `commonidentity.Owner` — see below for why both were rejected):

```go
// bin-conversation-manager/models/conversation/metadata.go (new file)
// Follows the exact precedent of bin-customer-manager/models/customer/metadata.go:
// a typed struct stored in a single nullable JSON column, with its own dedicated
// update path — not folded into the general partial-update allowlist.
type Metadata struct {
    ContactCaseID *uuid.UUID `json:"contact_case_id,omitempty"` // set by contact-manager when a
                                                                  // Case wants inbound messages on
                                                                  // this Conversation to prefer
                                                                  // attaching to it; read-only from
                                                                  // conversation-manager's perspective
}
```

```go
type Conversation struct {
    // ... existing fields unchanged ...
    Metadata Metadata `json:"metadata,omitempty" db:"metadata,json"` // new nullable JSON column
}
```

**Why a generic `Metadata` field, not a dedicated `contact_case_id` column:** this mirrors the `bin-customer-manager` precedent exactly (`customer.Metadata{ RTPDebug bool }`, a single typed struct in one JSON column with a dedicated `UpdateMetadata` RPC/handler, not a proliferation of top-level columns for every cross-service annotation). It keeps the door open for future cross-service annotations on `Conversation` without a new migration each time, and — more importantly for this design — it keeps `contact_case_id` visibly namespaced as "opaque metadata a related service asked us to carry," rather than looking like a first-class `Conversation` concept that `conversation-manager` itself reasons about.

**Why NOT `commonidentity.Owner` (the mechanism the CPO originally proposed and pchero explicitly rejected):** `Owner` already has a load-bearing, unrelated meaning on `Conversation` — setting it skips the registered activeflow trigger and routes inbound messages directly to the owning agent (§7 of `assignable-conversation-design.md`). Piggybacking Case-matching on `Owner` would silently couple "which Case an inbound message's Interaction attaches to" with "does this channel bypass the customer's configured flow," which are two independent product decisions. A customer-service Case being open for matching purposes must never accidentally suppress a customer's automated flow, and vice versa. The `Metadata.ContactCaseID` field is deliberately inert from `conversation-manager`'s own dispatch logic (§4 of that design) — it is never read by `getExecuteMode` or any flow/agent-routing decision, only echoed back onto outbound events for contact-manager's benefit.

**Write path.** When contact-manager's Case get-or-create (§4) opens or reuses a Case for a `reference_type` other than `message` (today: a `call` case), it makes one RPC to conversation-manager: get-or-create the `TypeMessage` Conversation for that same `(customer_id, peer)` (reusing `GetOrCreateBySelfAndPeer` exactly as it exists today — no new conversation-manager entity), then set `Metadata.ContactCaseID` on it to the just-opened Case's ID via a new `ConversationUpdateMetadata`-style RPC (mirroring `bin-customer-manager`'s `UpdateMetadata` shape: replace-the-whole-struct semantics, not a partial-field patch). This write happens once per Case open, not per message — cheap, and bounded by Case open/close frequency rather than message volume.

**Read path.** `MessageEventReceived` (`bin-conversation-manager/pkg/conversationhandler/message.go`) already loads the `Conversation` row for every inbound SMS to evaluate `getExecuteMode`. That existing load is extended to also read `conversation.Metadata.ContactCaseID` and include it as a `case_id` hint field on the `EventTypeConversationMessageCreated` payload that contact-manager subscribes to. No additional DB read, no additional RPC — the value rides on data conversation-manager already has in hand for an unrelated purpose.

**Consumption.** Contact-manager's Case get-or-create (§4) treats this conversation-metadata-derived hint through the **exact same validation and priority order** as the §4.3 explicit hint (customer_id match, `status='open'` check, fail-open to normal peer matching on any mismatch) — it is simply a second source for the same `case_id` hint parameter, not a parallel code path. The combined priority order is:

```
1. Explicit case_id hint from the triggering action itself (§4.3, e.g. agent-initiated outbound SMS during a call)
2. Else, case_id hint carried on the inbound event via Conversation.Metadata.ContactCaseID (§4.4, this section)
3. Else (or either hint invalid/stale) -> normal (customer_id, peer_type, peer_target, reference_type) get-or-create (§4)
```

**Staleness and cleanup.** `Metadata.ContactCaseID` is not actively cleared when the referenced Case closes — it is left to go stale, exactly like the §4.3 hint's fail-open validation already handles staleness by design (a closed-case reference fails the `status='open'` check and falls through harmlessly). This avoids adding a reverse notification path (Case close → conversation-manager cleanup) for a field whose only failure mode, if stale, is "falls back to the behavior that would have happened anyway." If a future Case opens for the same peer later, its own get-or-create overwrites `Metadata.ContactCaseID` with the new value at that point.

**Scope at launch.** Only the `call`-case-opens-and-links-the-message-Conversation direction is wired. The symmetric case (an open message Case wanting to link a concurrently active call) is not needed at launch since call-manager has no equivalent inbound-hint gap — calls are always explicitly initiated/answered through call-manager's own APIs, which already carry full context. If a future channel combination needs the same treatment, it follows this section's pattern.

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

The `WHERE status='open'` guard makes this an idempotent, race-tolerant optimistic update for the **double-close** case: two identical close requests racing each other resolve safely (the second is a harmless 0-row no-op) without a distributed lock. This `UPDATE` deliberately does not touch `owner_type`/`owner_id` — ownership is never cleared by closing (§7), which is what makes `/continue`'s owning-agent authorization (§5.3) work on a case that has since closed.

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

Creates a new, empty `open` Case with `previous_case_id = id`, using the same `(peer_type, peer_target, reference_type, contact_id)` as the source case. Requires the source case to be `status='closed'`; requires the caller to be the case's owning agent or admin/manager. **This authorization check relies on the invariant stated in §3.1/§7 that closing a case never clears `owner_type`/`owner_id`** — without that invariant, `/continue` would have no owning agent to check against for any case that closed normally. A case that timed out unowned (never assigned) has no meaningful owning agent at all and correctly falls through to the admin/manager path.

`/continue`'s insert is subject to **the exact same race as §4's insert branches** — two agents calling `/continue` on the same closed case simultaneously, or `/continue` racing a genuine inbound event on the same peer, both attempt `INSERT ... status='open'` against the same `uq_case_open_peer` constraint. `/continue` MUST reuse §4's bounded retry-and-reuse loop (locked re-select, retry on conflict) rather than defining its own handling or, worse, leaving the DB unique-constraint violation to surface as a raw, unhandled error. Concretely: `/continue`'s handler calls the same internal get-or-create primitive used by §4's insert branches, parameterized with `previous_case_id = id` instead of a peer-derived value, rather than reimplementing the insert logic. This is a distinct action from "reopening" the closed case itself — the old case stays exactly as it was, permanently attributed to whoever/whatever closed it; only a *new* case is created (or, per the shared retry logic, an already-open case for that peer is reused if one was concurrently created by another path).

## 6. Contact attribution and the unresolved queue

`Case.contact_id` is nullable and this is the normal state for a new peer, not an error condition. Three outcomes:

- **Agent links to an existing contact** — creates a case-level positive Resolution; `Case.contact_id` derives immediately (§3.4).
- **Agent creates a new contact** — `ContactCreate`, then link as above.
- **Agent closes without ever resolving** — legitimate (spam, misdial, one-off). No Resolution is created; the case simply closes with `contact_id` still NULL.

```
CaseListUnresolved:
  WHERE customer_id=? AND status='open' AND contact_id IS NULL
```

Backed by `idx_case_unresolved (customer_id, status, contact_id)` (§3.1). This is the agent's live work queue. Closing a case removes it from the queue regardless of resolution state — closing *is* the "no further action needed" signal; no separate "permanently unresolved" marker is introduced.

Resolution can be attached retroactively to a closed Case at any time (e.g. the contact is identified later), which updates `Case.contact_id` via the same derivation function. See §3.4 for why this does not race destructively against a concurrent close.

## 7. Case-adjacent agent workflow (this design's addition over the base Interaction/Resolution foundation)

Based on common patterns across ticketing/case platforms (assignment, internal notes, tagging, audit trail), cross-checked against what VoIPBin already has:

| Capability | Approach | Why |
|---|---|---|
| Ownership / assignment | Reuse `commonidentity.Owner` (`owner_type`/`owner_id`) exactly as conversation-manager already does it | Existing, proven pattern (`assignable-conversation-design.md`); same permission model (admin/manager assign, owning agent self-unassigns) applies unchanged to *open* cases |
| Ownership survives closing | Closing a case (§5.1) never clears `owner_type`/`owner_id` | Load-bearing for `/continue`'s authorization (§5.3); an implementer adding a "clear owner on close" cleanup step would silently break `/continue` for agent-closed cases, so this is stated as an explicit invariant here rather than left implicit |
| Tags | Register `Case` as a taggable resource in the existing `bin-tag-manager` | Purpose-built generic tagging service already exists; no new tag storage needed |
| Audit trail | Existing `case_created` / `case_updated` / `case_closed` webhook events | Same convention already used for Interaction/Resolution/Conversation; no new audit table |
| Internal notes | New `CaseNote` table (§3.5), created via a plain `PublishEvent`-based `case_note_created` signal — not the webhook-manager customer-delivery path | Genuinely missing capability; must be physically **and** transport-isolated from customer-facing data (see §3.5's corrected event-delivery design). Real-time agent delivery of this event is explicitly parked (§2), not claimed as already solved. |
| Priority, SLA, CSAT | Explicitly out of scope (§2) | Meaningless without routing/queue integration or a separate survey feature; adding now risks unused decoration |
| Platform-operator cross-customer visibility | Case/CaseNote endpoints are exposed through `bin-api-manager`'s standard permission gate and therefore inherit the platform's existing `PermissionProjectSuperAdmin` cross-customer bypass (used today for support/investigation) automatically — no new authorization code path is introduced for this | Called out explicitly so an implementer doesn't add case-specific customer-scoping logic that inadvertently excludes the existing superadmin bypass. If a future review finds Case/CaseNote do NOT correctly inherit this bypass, that is an implementation bug against this stated intent, not a design gap. |

## 8. Migration

1. Create `contact_cases` table (with the generated-column-based unique index and supporting indexes, §3.1 — not a native partial/filtered index, per the MySQL constraint noted there).
2. Add `case_id` (nullable) to `contact_interactions`.
3. Add `case_id` (nullable) to `contact_resolutions`; alter `interaction_id` to nullable (§3.3 — a real schema change from today's `NOT NULL`, not additive-only, AND a real Go-side breaking change to `dbhandler`/`contacthandler` call sites that must be scoped as its own implementation work, not assumed free); add the CHECK constraint and the generated-column-based unique index.
4. Create `contact_case_notes` table (with an index on `case_id` to back the note-list query).
5. Register `case` as a resource type in `bin-tag-manager`.
6. **No backfill.** Historical Interactions predate the Case concept; retroactively grouping them by peer+time-gap heuristics would invent case boundaries that never existed operationally and could misattribute historical activity. The feature's value is forward-looking case management, not historical reconstruction.

Step 3's `interaction_id` nullability change is additive-safe **at the DB level** (relaxing `NOT NULL` to nullable cannot reject previously-valid data), but is a real breaking change at the **Go code level** per §3.3 — the implementation plan must include the `dbhandler`/`contacthandler` refactor and test updates as explicit, separately-estimated work, not fold it silently into "add a column."

## 9. Service/API surface impact

- **bin-contact-manager**: `CaseHandler` (get-or-create with the bounded retry loop per §4.2, close with truthful-persisted-state response per §5.1, continue reusing the same get-or-create primitive per §5.3, assign/unassign, list, list-unresolved), `CaseNoteHandler` (create/list/delete, emits `case_note_created` via a plain `PublishEvent` — never `PublishWebhookEvent` — per §3.5), Resolution handler extended with the case-level derivation hook and case-scoped list/delete operations (§3.3), `case-control` CLI (including `reconcile-contact`). New RPC call to `bin-conversation-manager` when a non-message Case opens, to set `Conversation.Metadata.ContactCaseID` on the sibling message Conversation for the same peer (§4.4).
- **bin-conversation-manager**: new `Metadata` field on `Conversation` (nullable JSON column, mirroring `bin-customer-manager`'s `Metadata` pattern), a new `ConversationUpdateMetadata`-shaped RPC for contact-manager to call, and `MessageEventReceived` extended to echo `Metadata.ContactCaseID` onto the `EventTypeConversationMessageCreated` payload it already publishes (§4.4). No change to `getExecuteMode` or any flow/agent-routing dispatch logic — the new field is inert with respect to conversation-manager's own decisions.
- **bin-tag-manager**: register `case` as a taggable resource type (no schema change expected — generic tagging already supports arbitrary resource types).
- **bin-api-manager**: `/v1.0/cases`, `/v1.0/cases/{id}`, `/v1.0/cases/{id}/close`, `/v1.0/cases/{id}/continue`, `/v1.0/cases/{id}/notes`; extend `/v1.0/interactions` responses with `case_id`. Inherits the existing `PermissionProjectSuperAdmin` cross-customer bypass automatically (§7) — no new authorization code path.
- **bin-openapi-manager**: spec additions per `voipbin-openapi-spec-handler-parity` convention.
- **square-admin**: case list / unresolved queue / close button / notes panel — separate frontend design, not covered here.

## 10. Test scope (high level; detailed table-driven cases in the implementation plan)

- Case get-or-create: reuse on open match; timeout-triggered chain with locked, bounded-retry duplicate-key handling (§4.2); first-contact concurrent-insert with the same locked retry; explicit-hint bypass with valid/stale/wrong-tenant/closed hint (§4.3); the specific TOCTOU scenario from round-2 review — a case closes between a losing transaction's duplicate-key retry-select and its subsequent Interaction insert — must NOT succeed in attaching an Interaction to a since-closed case.
- Case close: idempotent double-close; close-vs-timeout race returns the actually-persisted `closed_reason`/`closed_by` rather than the caller's assumed outcome (§5.1); `continue` on a closed case creates a correctly-chained new case without mutating the source; two concurrent `/continue` calls on the same case resolve via the shared get-or-create retry logic (one creates, the other reuses) rather than raising a raw DB error; `/continue` racing a genuine inbound event on the same peer resolves the same way.
- Resolution → Case.contact_id derivation: create; replace (soft-delete + insert, single transaction — verify no transient NULL is externally observable); delete; reconcile CLI idempotency; insert rejected when neither `case_id` nor `interaction_id` is set (CHECK constraint); concurrent case-level Resolution creation for the same case resolves via the same duplicate-key reuse pattern as Case inserts, not a raw error.
- Unresolved queue correctness under all three closing outcomes (§6).
- CaseNote: never appears in customer-facing Interaction list or webhook payload (explicit negative test); `case_note_created` fires via plain `PublishEvent` on creation AND is explicitly verified to never be routed through `PublishWebhookEvent` / never reach the customer webhook delivery mechanism (explicit negative test — this closes the exact gap the first two design drafts had).
- Cross-channel explicit-hint scenario (outbound, §4.3): call-active SMS lands in the same case_id as the call; a stale/closed hint falls back to normal peer-based matching rather than erroring or attaching to the wrong case.
- Cross-channel inbound-hint scenario (§4.4): a customer's inbound SMS during an active call lands in the same case_id as the call, via `Conversation.Metadata.ContactCaseID`; a stale `ContactCaseID` (pointing at an already-closed case) falls back to normal peer-based matching, exactly like a stale §4.3 hint; verify `Metadata.ContactCaseID` never influences `getExecuteMode`'s flow-vs-agent dispatch decision (explicit negative test — this is the invariant that keeps the field decoupled from `commonidentity.Owner`'s unrelated semantics).
- `/continue` authorization: verify a case closed via normal agent close retains its `owner_type`/`owner_id` and that the owning agent (and only the owning agent, or admin/manager) can call `/continue` on it.

## 11. Rollback

No destructive migration. All new columns are nullable and all new tables are additive; the one relaxed constraint (`interaction_id` NOT NULL → nullable) cannot reject previously-valid data. Rollback is a straightforward revert-and-redeploy: existing Interaction/Resolution/Conversation behavior is untouched by this feature, since Case sits alongside them rather than replacing any existing write path. Reverting the Go-side `dbhandler`/`contacthandler` changes from §3.3 alongside the schema revert restores the original non-nullable `InteractionID` contract cleanly, since no external caller depends on the nullable form once this feature is rolled back. The `Conversation.Metadata` addition (§4.4) is likewise purely additive and inert to conversation-manager's own dispatch logic; reverting it drops the inbound-hint optimization and inbound SMS during an active call reverts to spinning up a disconnected Case, exactly the pre-existing (pre-§4.4) behavior — no data loss, no behavior change beyond losing the optimization.
