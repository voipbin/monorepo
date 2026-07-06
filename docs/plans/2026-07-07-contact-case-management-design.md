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

- `Case` entity: get-or-create keyed by `(customer_id, peer_type, peer_target, reference_type)`, with a single unified `Conversation.Metadata.ContactCaseID` field (§4.3/§4.4) written from either an agent's explicit send via `POST /v1.0/cases/{id}/messages` (§4.5) or a proactive Case-open write for cross-channel continuity, and read uniformly by both inbound and outbound message events with no priority/ranking logic needed.
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
| Inbound-call-during-an-open-message-case linking (the reverse of §4.4: a customer calling in while an SMS case is already open) | §4.4 only wires the `call`-opens→links-message-Conversation direction; the symmetric direction has the identical class of gap (an inbound call is a raw carrier-webhook trigger with no more inherent context than an inbound SMS) but is not yet a reported pain point | A real operational complaint about a disconnected call case while an SMS case is open for the same customer |
| `case_link_source` provenance field (surfacing to agents *how* a Case's `case_id` was resolved — explicit §4.3 hint, §4.4 conversation-metadata hint, or plain peer match) | §4.4's linking is currently silent/invisible in the API surface (§9); adding provenance is an API + frontend design decision, not a backend mechanism this doc otherwise covers | square-admin design work surfacing Case detail, at which point provenance display can be scoped together with it |
| Per-`reference_type` keying (or an explicit precedence rule) for `Conversation.Metadata.ContactCaseID` when more than one non-`message` `reference_type` writes to it | At launch only `call` writes this field, so the single-scalar last-writer-wins limitation flagged in §4.4 is not an active defect | Before a second non-`message` `reference_type` (e.g. a hypothetical future video/chat channel) adopts the §4.4 write pattern |
| Agent override / opt-out from automatic §4.3/§4.4 hint-based case linking | Both hint sources are additive conveniences over the base peer-matching behavior, not required for correctness; an unwanted link is recoverable (the agent can still work the case, or split later once merge is designed) | A concrete agent complaint about an unwanted auto-link, at which point an explicit "don't link" mechanism can be scoped |

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

### 4.3 Case-linked messaging: a single `Conversation.Metadata` field, no priority logic

**Round-11 simplification.** Earlier drafts of this design treated "an agent explicitly sending a message from an open case" (outbound) and "a customer's inbound message picking up an open case" (inbound) as **two separate hint sources** that contact-manager would have to read and rank by priority. That framing was rejected during review: a two-source, prioritized-consumption design does not scale — every additional hint source added another priority tier and another branch in contact-manager's consumption logic, purely to solve a conflict (outbound hint vs. stale inbound hint) that does not actually need to exist if the write side is unified instead.

**The unified mechanism: whoever knows the case_id first writes it to the same field; everyone else just reads that one field.** There is exactly one place case-linking information lives: `Conversation.Metadata.ContactCaseID` (§4.4 below). There are two ways it gets **written**, but only one way it gets **read**:

- **Written proactively, before any message exists on the peer:** when contact-manager opens a non-`message` Case (today: `call`), it looks up (get-only, never creates) the sibling message `Conversation` for that peer and sets `Metadata.ContactCaseID` on it. This is unchanged from §4.4's mechanism.
- **Written explicitly, at the moment an agent sends a message from a known case (§4.5, `POST /v1.0/cases/{case_id}/messages`):** before the message is actually sent, the handler sets `Metadata.ContactCaseID` on the target `Conversation` to the case the agent is sending from — using the **exact same** `ConversationUpdateMetadata` RPC §4.4 already defines, not a second mechanism. This closes the field, not adds a rival to it.
- **Read exactly once, uniformly, regardless of direction:** every message event (inbound or outbound, from `MessageEventReceived` or `MessageEventSent`) reads `Conversation.Metadata.ContactCaseID` off the `Conversation` row it already has in hand and includes it as the (single) `case_id` hint on the event contact-manager subscribes to. There is no second hint field, no "which one wins" branch, and no priority list to extend.

**Why this cannot go stale in the way a two-source design would worry about.** In the two-source framing, the concern was: "an agent explicitly picks case B, but the Conversation's metadata still says case A from an earlier call — which one does contact-manager trust?" That scenario cannot arise here, because the *only* way an agent sends a case-linked message is through §4.5, and §4.5 **writes** the field to case B immediately before sending — there is no second value floating around to conflict with. The field always reflects "the most recent thing that claimed this Conversation," which is exactly the semantics an agent picking case B expects: their action is authoritative because it's the last write, not because a priority table says outbound beats inbound.

**Consumption, simplified accordingly.** Contact-manager's Case get-or-create (§4 step 1) validates the single `case_id` hint on the event (customer_id match, `status='open'`) exactly as before — this part is unchanged. What's removed is the "is this the §4.3 hint or the §4.4 hint, and which wins" question, because there is only ever one hint value on the wire at read time.

```
1. case_id hint present on the triggering event (regardless of direction or which write path set it)
     -> validate (customer_id match, status='open', see §4 step 1)
     -> valid: use it, skip peer/reference_type matching entirely
     -> invalid/stale/closed: fall through to step 2
2. No hint (or hint fell through) -> (customer_id, peer_type, peer_target, reference_type) get-or-create
```

**Validation is mandatory, not optional.** An unvalidated hint is both a cross-tenant leak vector (a stale or wrong cached `case_id` could attribute an Interaction to a different customer's case) and a direct violation of §5.3's closed-case-immutability invariant (a hint referencing a case that closed between when it was written and when the message was actually sent would otherwise insert into a closed case). §4 step 1's `SELECT ... WHERE id=? AND customer_id=? AND status='open'` is the single point where this is enforced — a hint that fails this check is treated exactly as if no hint were supplied, never as an error that blocks the underlying action (the message still gets sent and projected; it just lands in whatever case the normal peer-based path resolves to).

### 4.5 `POST /v1.0/cases/{case_id}/messages` — sending a message from a known case

This is the concrete API surface that makes §4.3's write path real; without it, "an agent sends a message from an open case" has no endpoint to attach to, and the unified single-field mechanism above has nothing to trigger its write side.

**Request body — source and destination are explicit, not inferred:**

```json
POST /v1.0/cases/{case_id}/messages
{
  "source": "+15551234567",       // the business's own number to send from
  "destination": "+15559876543",  // the customer's number
  "text": "..."
}
```

Requiring `source`/`destination` explicitly (rather than having the backend infer a `self` address) sidesteps a real gap found during this design's review: `Case` stores `peer_type`/`peer_target` but no `self` address, and unlike calls (which have an outbound-config "default source number" concept), SMS has no equivalent default-source convention in the codebase today. Rather than inventing one, the agent — who already sees the case detail screen and knows which of the business's numbers this conversation is on — supplies it directly. The backend's job narrows to **validating** the supplied values are safe to use, not guessing them.

**Validation, in order:**

1. **Case validation.** `case_id` belongs to the calling customer and is `status='open'` (the same check §4 step 1 already performs; reused, not reimplemented). A closed case is rejected — `POST /v1.0/cases/{case_id}/continue` (§5.3) must be called first, consistent with §5.3's closed-case-immutability invariant; silently accepting a send against a closed case would contradict that invariant instead of going through the one sanctioned reopening path.
2. **Destination-to-case binding (the check this section exists to add).** `destination` must be attributable to this specific case, checked in this order:
   - If `case.contact_id` is set (the case has a matched Contact): `destination` must appear in that Contact's registered addresses (`bin-contact-manager`'s existing `AddressListByContact`, reused as-is — no new address-matching logic). This is the strict check: an agent cannot use this endpoint to message a number unrelated to the Contact this case represents, even if they know the case_id.
   - Else (the case has no matched Contact yet — an unresolved case, §6): `destination` must equal `case.peer_target` exactly (the only address this case is actually about, absent a broader Contact record to check against).
   - Otherwise: reject with a 4xx (e.g. "destination is not associated with this case"). This is the concrete safety mechanism that prevents case_id from being usable as a bare capability token to message an arbitrary number — a caller cannot launder an unrelated send through someone else's case just by knowing its ID.
3. **Source validation.** Delegated entirely to the existing outbound-send path (`bin-api-manager`'s `ConversationMessageSend` / the underlying `sendLine`/`sendSMS` handlers already validate that `source` is a number this customer owns) — not reimplemented here.

**Execution, reusing §4.4's write mechanism exactly:**

1. `ConversationGetBySelfAndPeer(customer_id, self=source, peer=destination)` — get-only, per §4.4's round-7 correction; if the Conversation doesn't exist yet, the subsequent send (step 3 below) creates it the normal way via the existing `GetOrCreateBySelfAndPeer` inside the send path, and this endpoint's metadata-write step is simply skipped for this call (first-ever message to this destination has no prior Conversation to annotate — same accepted narrow degradation §4.4 already documents for the inbound direction, now symmetric for outbound).
2. If found: set `Metadata.ContactCaseID = case_id` via `ConversationUpdateMetadata` (§4.4's existing RPC, not a new one).
3. Send the message through the existing outbound path (`ConversationMessageSend` → `sendLine`/`sendSMS`), unchanged.

Steps 1–2 happening **before** step 3 is what makes the resulting `MessageEventSent` correctly carry the just-written `case_id` hint when contact-manager reads it per §4.3's unified read path — the write must land before the read that consumes it.

### 4.4 Why the field is never stale for the write side that matters: proactive linking from a non-message Case

**The gap this originally solved.** Before §4.3's unification, this section addressed a narrower problem: when a customer sends an inbound SMS mid-call (or shortly after), the inbound SMS event is generated by `message-manager` from a raw provider webhook, with no way for the customer's phone to carry a VoIPBin-internal `case_id`. Left unaddressed, that inbound SMS would fall through to §4's normal peer/reference_type matching, find no open `reference_type='message'` case, and spin up a **new, disconnected** SMS case. §4.3 now frames this as one of two write paths into the same field; this section specifies that particular write path in full.

**The fix — annotate an existing Conversation, never manufacture one.** `bin-conversation-manager` already maintains a stable `Conversation` row per `(customer_id, self, peer)`, created lazily by `GetOrCreateBySelfAndPeer` (`pkg/conversationhandler/db.go:43-86`) the first time any actual message crosses that peer, and it already loads that `Conversation` on **every** inbound message (`MessageEventReceived`, `pkg/conversationhandler/message.go:109-142`) to decide whether to trigger an activeflow or route to an assigned agent (`assignable-conversation-design.md`). This existing load is the near-zero-cost place to carry a case-matching hint through to the inbound path — reading one more field off a struct already in hand costs nothing extra on the read side (per §4.3's unified read path, this same load also applies on the outbound side via `MessageEventSent`).

**Round-7 correction: this must be a lookup, not a get-or-create.** An earlier draft of this section had contact-manager call `GetOrCreateBySelfAndPeer` (proactively creating the message Conversation from the *call*-opening path, before any message has ever occurred). That is unsafe: `Create()` (`bin-conversation-manager/pkg/conversationhandler/db.go:105-160`) unconditionally calls `h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, conversation.EventTypeConversationCreated, res)` (line 157) with no guard other than `customerID == uuid.Nil`, which never fires for a real tenant. Proactively creating a message-type Conversation purely as case-linking plumbing — with no message ever having been sent — would fire a genuine, real, customer-facing `conversation_created` webhook for a thread that doesn't actually exist yet from the customer's perspective. A webhook consumer reasonably treating `conversation_created` as "a new message thread started" would be actively misled, and the resulting empty Conversation row would be a permanent, orphaned artifact with no messages ever landing in it if the customer never actually texts.

**The corrected mechanism is a plain lookup, conditionally followed by an update:** when contact-manager's Case get-or-create (§4) opens a *new* Case for a `reference_type` other than `message` (today: a `call` case), it calls a new **get-only** RPC (`ConversationGetBySelfAndPeer` — exposing the existing internal `GetBySelfAndPeer` lookup, not `GetOrCreateBySelfAndPeer`, cross-service). **Note on scoping:** the underlying lookup is keyed by `(self, peer)` address pair only — it takes no `customer_id` parameter and the DB query has no `customer_id` filter, relying on the existing platform invariant that a `self` address (the business's own number/channel) is never shared across customers. This is not a new risk introduced by this section: even in the hypothetical case that invariant were ever violated, a resulting cross-tenant `Metadata.ContactCaseID` value would still fail the downstream `customer_id`+`status='open'` check at consumption time (the same tenant-isolation choke point described below), so no new validation is required here — it is called out for documentation completeness, not because it changes the design. Two outcomes:

- **Found** (the customer has an existing message history with this business, e.g. a prior SMS thread): set `Metadata.ContactCaseID` on it via the `ConversationUpdateMetadata` RPC (this same RPC is reused by §4.5's outbound write path). The optimization applies.
- **Not found** (no message-type Conversation has ever been created for this peer): do nothing. No Conversation is created, no webhook fires, no orphaned row is left behind. The first actual inbound SMS from this peer — if one ever arrives — creates its own Conversation the normal way (via `GetOrCreateBySelfAndPeer` inside `message.go`, which fires the same `conversation_created` webhook it always has, now truthfully, because a real message triggered it) and simply has no hint available on its first message; it falls through to §4's normal peer/reference_type matching for that one message, same as if this mechanism didn't exist for that specific case-pairing. This is an accepted, narrow degradation (a fresh-thread inbound SMS with zero prior message history won't retroactively link to a concurrent call on its very first message), not a correctness problem — subsequent messages on that now-created Conversation are unaffected since nothing in this flow prevents them. §4.5 documents the symmetric case for the outbound direction.

This also **narrows the new wire-level surface** from a get-or-create to a plain get: `GetBySelfAndPeer` (the read half of `db.go:43-86`) is, like its get-or-create sibling, in-process only today — `bin-common-handler/pkg/requesthandler/conversation_conversations.go` exposes only `Get/List/Create/Update` by ID, none of them keyed by `(self, peer)`. Contact-manager calling it cross-service still requires one new RPC (route + listenhandler + client method), but it is a read-only lookup with no side effects on miss, which is what makes the webhook problem disappear entirely rather than merely being mitigated.

Concretely, this requires a **new, generic `Metadata` field on `Conversation`** (not a dedicated `contact_case_id` column, and explicitly not tied to `commonidentity.Owner` — see below for why both were rejected):

```go
// bin-conversation-manager/models/conversation/metadata.go (new file)
// Follows the exact precedent of bin-customer-manager/models/customer/metadata.go:
// a typed struct stored in a single nullable JSON column, with its own dedicated
// update path — not folded into the general partial-update allowlist.
type Metadata struct {
    ContactCaseID *uuid.UUID `json:"contact_case_id,omitempty"` // set by contact-manager, from
                                                                  // either write path in §4.3, to
                                                                  // claim this Conversation for a
                                                                  // Case; read-only from
                                                                  // conversation-manager's perspective
}
```

```go
type Conversation struct {
    // ... existing fields unchanged ...
    Metadata Metadata `json:"metadata,omitempty" db:"metadata,json"` // new nullable JSON column
}
```

**Why a generic `Metadata` field, not a dedicated `contact_case_id` column:** this mirrors the `bin-customer-manager` precedent exactly (`customer.Metadata{ RTPDebug bool }`, a single typed struct in one JSON column with a dedicated `UpdateMetadata` RPC/handler exposed at its own route rather than folded into the generic partial-update allowlist — confirmed against `bin-conversation-manager/pkg/listenhandler/v1_conversations.go:184-190`, whose existing PUT allowlist is `FieldOwnerType, FieldOwnerID, FieldName, FieldDetail, FieldAccountID` and does not include `Metadata`). It keeps the door open for future cross-service annotations on `Conversation` without a new migration each time, and — more importantly for this design — it keeps `contact_case_id` visibly namespaced as "opaque metadata a related service asked us to carry," rather than looking like a first-class `Conversation` concept that `conversation-manager` itself reasons about.

**Why NOT `commonidentity.Owner` (the mechanism the CPO originally proposed and pchero explicitly rejected):** `Owner` already has a load-bearing, unrelated meaning on `Conversation` — setting it skips the registered activeflow trigger and routes inbound messages directly to the owning agent (§7 of `assignable-conversation-design.md`). Piggybacking Case-matching on `Owner` would silently couple "which Case an inbound message's Interaction attaches to" with "does this channel bypass the customer's configured flow," which are two independent product decisions. A customer-service Case being open for matching purposes must never accidentally suppress a customer's automated flow, and vice versa. The `Metadata.ContactCaseID` field is deliberately inert from `conversation-manager`'s own dispatch logic — it is never read by `getExecuteMode` or any flow/agent-routing decision, only echoed back onto message events for contact-manager's benefit. (Explicit negative test in §10 asserts this invariant holds.)

**Write path (this section's direction) — up to two RPCs, with an explicit failure-handling rule.** When contact-manager's Case get-or-create (§4) opens a *new* Case (not on cache-hit reuse of an already-linked one — see cost note below) for a `reference_type` other than `message` (today: a `call` case), it makes **up to two sequential RPCs** to conversation-manager:

1. `ConversationGetBySelfAndPeer` (get-only, per the round-7 correction above) for that same `(customer_id, peer)`. If not found, stop here — no further RPC, nothing created (see "not found" outcome above).
2. If found: set `Metadata.ContactCaseID` on the returned Conversation via the `ConversationUpdateMetadata` RPC (whole-struct replace semantics, matching `bin-customer-manager`'s `UpdateMetadata` shape).

Both RPCs happen **after** the Case DB transaction (§4) commits, never inside it — the Case's `FOR UPDATE` locks (§4.2) must not be held across a cross-service network round trip. **Failure handling:** if either RPC fails, the failure is logged and **does not roll back or fail the Case-open operation** — the Case still opens successfully; only the linking optimization is lost for this Case (the next message event on that peer falls through to normal peer/reference_type matching, §4, exactly as if this mechanism didn't exist). No retry loop is introduced for these RPCs: a missed link is a degraded-but-correct outcome (worst case: a message spins up its own case, recoverable via the existing `previous_case_id` chain semantics once an agent notices), not a correctness violation worth the complexity of a retry protocol. This is a deliberate scope decision, not an oversight.

**Cost note — the write happens once per Case *open*, not once per event.** The trigger condition is specifically "a **new** Case was just opened" (the `INSERT` branches of §4's get-or-create), not "any event resolved to this Case" — an inbound event that reuses an already-open, already-linked Case does not re-fire these RPCs. This keeps the cost bounded by Case open frequency, not match/message frequency.

**Read path.** `MessageEventReceived` already loads the `Conversation` row for every inbound SMS to evaluate `getExecuteMode`; per §4.3, `MessageEventSent` does the same for outbound. That existing load is extended to also read `conversation.Metadata.ContactCaseID` and include it as the single `case_id` hint field on the `EventTypeConversationMessageCreated` payload contact-manager subscribes to, regardless of direction. No additional DB read, no additional RPC — the value rides on data conversation-manager already has in hand for an unrelated purpose.

**Tenant-isolation invariant (referenced by §4.3's consumption step).** **Conversation-manager performs zero validation of the value it echoes** — it is treated as opaque, untrusted data on the wire. The single validating choke point is contact-manager's §4-step-1 query, and its `customer_id` predicate is always sourced from the triggering event's own independently-known customer context, **never** derived from or through the hint value itself. This is the invariant that makes the design safe against both ordinary staleness (a closed-case reference) and a hypothetical cross-tenant pollution bug (a wrong `case_id` belonging to a different customer somehow ending up in a Conversation's metadata) — both fail the same `id=? AND customer_id=?` check and fall through harmlessly, by construction, not by accident.

**Staleness and cleanup.** `Metadata.ContactCaseID` is not actively cleared when the referenced Case closes — it is left to go stale, exactly like §4.3's fail-open validation already handles staleness by design (a closed-case reference fails the `status='open'` check and falls through harmlessly). This avoids adding a reverse notification path (Case close → conversation-manager cleanup) for a field whose only failure mode, if stale, is "falls back to the behavior that would have happened anyway." If a future write (from either write path in §4.3) targets the same peer later, it overwrites `Metadata.ContactCaseID` with the new value at that point — this overwrite-on-next-write behavior is exactly what makes the single-field design in §4.3 correct without needing priority logic.

**Known structural limitation — single-slot field, no arbitration across concurrently-open sibling reference_types.** `Metadata.ContactCaseID` is a single scalar, not keyed by `reference_type`. §3.1's `uq_case_open_peer` constraint is scoped **per reference_type**, so two *different* non-`message` reference_types can legitimately both be `open` simultaneously for the same peer (that is the entire premise this mechanism exists to bridge). At launch, only `call` triggers this section's proactive write, so this is not an active launch-day defect. But if a future non-message `reference_type` adopts this same write pattern (as the "Scope at launch" paragraph below anticipates), two concurrently-open sibling Cases would race to overwrite the same field with different values — true last-writer-wins, no versioning, no per-type keying, no tie-break policy. This is an explicitly acknowledged, deferred design gap: **before wiring a second `reference_type` into this pattern, this section must be revisited** to either key `Metadata` by `reference_type` or define an explicit precedence rule. Not solved here because it is not needed here; flagged so it is not silently rediscovered as a bug later.

**No agent override / opt-out.** A valid hint (from either write path in §4.3) is consumed automatically and silently — there is no mechanism for an agent to say "I know there's an open case for this peer, but I want a genuinely separate one" for a specific message. Concretely: an agent working an open call about issue A, and the same customer texts about unrelated issue B, has no way to prevent the SMS from linking to the call's case via this mechanism. This is an accepted v1 limitation, not solved here — the mismatch is recoverable after the fact (an agent can still work the linked case and manually track that it covers two topics, or split later once Case merge is designed, §2's parked items), but it is a real, named limitation rather than a silently perfect outcome.

**No provenance signal exposed to agents.** Nothing in §9's API surface distinguishes "this Interaction's `case_id` was set via the proactive Case-open write path (this section)," "via the explicit §4.5 send-time write," or "via normal peer matching." An agent looking at a Case or Interaction cannot currently tell that automatic mid-call linking occurred versus a coincidental same-`reference_type` match. Exposing a `case_link_source` (or similarly named) field is deferred — added to §2's parked-items table — rather than solved here, since it requires an API/frontend design decision (where and how to surface it) beyond this section's backend-mechanism scope.

**Scope at launch — corrected.** Only the `call`-case-opens-and-links-the-message-Conversation direction is wired (this section) plus the explicit send-time write (§4.5). The reverse case (an open message Case linking a concurrently active *inbound* call) needed no equivalent mechanism was an earlier, incorrect claim in this document, on the theory that "calls are always explicitly initiated/answered through call-manager's own APIs which already carry full context." **That justification is incorrect for inbound calls specifically** — an inbound call is triggered by `EventCallCreated` from a raw carrier/SIP-trunk webhook, structurally identical in kind to the inbound-SMS-from-webhook scenario this section exists to fix (confirmed: §4's get-or-create pseudocode shows `EventCallCreated` resolving through the exact same generic peer/reference_type path as everything else, with no inherent extra context). The honest scope statement is: **the inbound-call-during-an-open-message-case direction has the same class of gap as the one this section fixes, and it is knowingly deferred, not solved** — added to §2's parked items with the concrete trigger "a real operational complaint about a disconnected call case while an SMS case is open for the same customer." The rationale that *does* correctly hold (and is the reason `call` was chosen as the only proactive-write direction wired at launch) is asymmetric demand: an agent proactively texting mid-call (§4.5, explicit write) and a customer texting mid-call (this section, proactive write) are both observed patterns worth solving now, while a customer calling in mid-SMS-conversation is not yet a reported pain point. If a future channel combination needs the same treatment, it follows this section's pattern, revisiting the single-slot-field limitation above first if the combination involves two simultaneously-open non-message reference_types.

### 4.5 `POST /v1.0/cases/{case_id}/messages` — sending a message from a known case

This is the concrete API surface that makes §4.3's explicit write path real; without it, "an agent sends a message from an open case" has no endpoint to attach to, and the unified single-field mechanism in §4.3/§4.4 has nothing to trigger its explicit-write side.

**Request body — source and destination are explicit, not inferred:**

```json
POST /v1.0/cases/{case_id}/messages
{
  "source": "+15551234567",       // the business's own number to send from
  "destination": "+15559876543",  // the customer's number
  "text": "..."
}
```

Requiring `source`/`destination` explicitly (rather than having the backend infer a `self` address) sidesteps a real gap found during this design's review: `Case` stores `peer_type`/`peer_target` but no `self` address, and unlike calls (which have an outbound-config "default source number" concept), SMS has no equivalent default-source convention in the codebase today. Rather than inventing one, the agent — who already sees the case detail screen and knows which of the business's numbers this conversation is on — supplies it directly. The backend's job narrows to **validating** the supplied values are safe to use, not guessing them.

**Validation, in order:**

1. **Case validation.** `case_id` belongs to the calling customer and is `status='open'` (the same check §4 step 1 already performs; reused, not reimplemented). A closed case is rejected — `POST /v1.0/cases/{case_id}/continue` (§5.3) must be called first, consistent with §5.3's closed-case-immutability invariant; silently accepting a send against a closed case would contradict that invariant instead of going through the one sanctioned reopening path.
2. **Destination-to-case binding (the check this section exists to add).** `destination` must be attributable to this specific case, checked in this order:
   - If `case.contact_id` is set (the case has a matched Contact): `destination` must appear in that Contact's registered addresses (`bin-contact-manager`'s existing `AddressListByContact`, reused as-is — no new address-matching logic). This is the strict check: an agent cannot use this endpoint to message a number unrelated to the Contact this case represents, even if they know the case_id.
   - Else (the case has no matched Contact yet — an unresolved case, §6): `destination` must equal `case.peer_target` exactly (the only address this case is actually about, absent a broader Contact record to check against).
   - Otherwise: reject with a 4xx (e.g. "destination is not associated with this case"). This is the concrete safety mechanism that prevents case_id from being usable as a bare capability token to message an arbitrary number — a caller cannot launder an unrelated send through someone else's case just by knowing its ID.
3. **Source validation.** Delegated entirely to the existing outbound-send path (`bin-api-manager`'s `ConversationMessageSend` / the underlying `sendLine`/`sendSMS` handlers already validate that `source` is a number this customer owns) — not reimplemented here.

**Execution, reusing §4.4's write mechanism exactly:**

1. `ConversationGetBySelfAndPeer(customer_id, self=source, peer=destination)` — get-only, per §4.4's round-7 correction; if the Conversation doesn't exist yet, the subsequent send (step 3 below) creates it the normal way via the existing `GetOrCreateBySelfAndPeer` inside the send path, and this endpoint's metadata-write step is simply skipped for this call (first-ever message to this destination has no prior Conversation to annotate — same accepted narrow degradation §4.4 already documents for the inbound direction, now symmetric for outbound).
2. If found: set `Metadata.ContactCaseID = case_id` via `ConversationUpdateMetadata` (§4.4's existing RPC, not a new one).
3. Send the message through the existing outbound path (`ConversationMessageSend` → `sendLine`/`sendSMS`), unchanged.

Steps 1–2 happening **before** step 3 is what makes the resulting `MessageEventSent` correctly carry the just-written `case_id` hint when contact-manager reads it per §4.3's unified read path — the write must land before the read that consumes it.

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
- **bin-conversation-manager**: new `Metadata` field on `Conversation` (nullable JSON column, mirroring `bin-customer-manager`'s `Metadata` pattern); a **new get-only RPC** (`ConversationGetBySelfAndPeer` — route + listenhandler + `bin-common-handler` client method, exposing the existing internal `GetBySelfAndPeer` lookup) — deliberately get-only, not get-or-create, so a miss creates nothing and fires no webhook (§4.4 round-7 correction); a new `ConversationUpdateMetadata`-shaped RPC for contact-manager to call; and `MessageEventReceived` extended to echo `Metadata.ContactCaseID` onto the `EventTypeConversationMessageCreated` payload it already publishes (§4.4). No change to `getExecuteMode` or any flow/agent-routing dispatch logic — the new field is inert with respect to conversation-manager's own decisions.
- **bin-tag-manager**: register `case` as a taggable resource type (no schema change expected — generic tagging already supports arbitrary resource types).
- **bin-api-manager**: `/v1.0/cases`, `/v1.0/cases/{id}`, `/v1.0/cases/{id}/close`, `/v1.0/cases/{id}/continue`, `/v1.0/cases/{id}/notes`, **`POST /v1.0/cases/{id}/messages`** (§4.5 — send a case-linked message with explicit `source`/`destination`, validated against the case's matched Contact or bare `peer_target`, then routed through the existing `ConversationMessageSend` path with a `Conversation.Metadata.ContactCaseID` write beforehand); extend `/v1.0/interactions` responses with `case_id`. `GET /v1.0/cases` supports filter query params `status`, `owner_type`/`owner_id`, and an unresolved filter (`contact_id IS NULL` combined with `status=open`) — mapping directly to the `idx_case_owner`/`idx_case_unresolved` indexes (§3.1) and the `list`/`list-unresolved` handler capabilities (§9's `bin-contact-manager` bullet above) that already exist to serve exactly these two day-one operator views ("my open cases," "unresolved queue"). Inherits the existing `PermissionProjectSuperAdmin` cross-customer bypass automatically (§7) — no new authorization code path.
- **bin-openapi-manager**: spec additions per `voipbin-openapi-spec-handler-parity` convention.
- **square-admin**: case list / unresolved queue / close button / notes panel — separate frontend design, not covered here.

## 10. Test scope (high level; detailed table-driven cases in the implementation plan)

- Case get-or-create: reuse on open match; timeout-triggered chain with locked, bounded-retry duplicate-key handling (§4.2); first-contact concurrent-insert with the same locked retry; explicit-hint bypass with valid/stale/wrong-tenant/closed hint (§4.3); the specific TOCTOU scenario from round-2 review — a case closes between a losing transaction's duplicate-key retry-select and its subsequent Interaction insert — must NOT succeed in attaching an Interaction to a since-closed case.
- `POST /v1.0/cases/{id}/messages` (§4.5): rejects a send on a closed case (must `/continue` first); destination-to-case binding accepts a `destination` in the matched Contact's address list, accepts a bare `peer_target` match when the case is unresolved, and rejects any other destination with a 4xx (explicit test that case_id cannot be used as a bare capability token to message an unrelated number); the `Conversation.Metadata.ContactCaseID` write happens before the send and is correctly reflected on the resulting `MessageEventSent`'s hint field, verified end-to-end against §4.3's unified consumption path; first-ever-destination case (no prior Conversation) sends successfully with the metadata-write step skipped, not failed.
- Case close: idempotent double-close; close-vs-timeout race returns the actually-persisted `closed_reason`/`closed_by` rather than the caller's assumed outcome (§5.1); `continue` on a closed case creates a correctly-chained new case without mutating the source; two concurrent `/continue` calls on the same case resolve via the shared get-or-create retry logic (one creates, the other reuses) rather than raising a raw DB error; `/continue` racing a genuine inbound event on the same peer resolves the same way.
- Resolution → Case.contact_id derivation: create; replace (soft-delete + insert, single transaction — verify no transient NULL is externally observable); delete; reconcile CLI idempotency; insert rejected when neither `case_id` nor `interaction_id` is set (CHECK constraint); concurrent case-level Resolution creation for the same case resolves via the same duplicate-key reuse pattern as Case inserts, not a raw error.
- Unresolved queue correctness under all three closing outcomes (§6).
- CaseNote: never appears in customer-facing Interaction list or webhook payload (explicit negative test); `case_note_created` fires via plain `PublishEvent` on creation AND is explicitly verified to never be routed through `PublishWebhookEvent` / never reach the customer webhook delivery mechanism (explicit negative test — this closes the exact gap the first two design drafts had).
- Cross-channel explicit-hint scenario (outbound, §4.3): call-active SMS lands in the same case_id as the call; a stale/closed hint falls back to normal peer-based matching rather than erroring or attaching to the wrong case.
- Cross-channel inbound-hint scenario (§4.4): a customer's inbound SMS during an active call lands in the same case_id as the call, via `Conversation.Metadata.ContactCaseID`, when a message Conversation already exists for that peer; a stale `ContactCaseID` (pointing at an already-closed case) falls back to normal peer-based matching, exactly like a stale §4.3 hint; the "not found" outcome (no prior message Conversation exists for the peer) creates nothing and fires no `conversation_created` webhook (explicit negative test — this is the exact defect a round-7 review caught in an earlier draft that proactively get-or-created the Conversation); either of the two write-path RPCs failing does not roll back or fail the Case-open operation (explicit test asserting the Case still opens successfully with only the linking optimization lost); verify `Metadata.ContactCaseID` never influences `getExecuteMode`'s flow-vs-agent dispatch decision (explicit negative test — this is the invariant that keeps the field decoupled from `commonidentity.Owner`'s unrelated semantics).
- `/continue` authorization: verify a case closed via normal agent close retains its `owner_type`/`owner_id` and that the owning agent (and only the owning agent, or admin/manager) can call `/continue` on it.

## 11. Rollback

No destructive migration. All new columns are nullable and all new tables are additive; the one relaxed constraint (`interaction_id` NOT NULL → nullable) cannot reject previously-valid data. Rollback is a straightforward revert-and-redeploy: existing Interaction/Resolution/Conversation behavior is untouched by this feature, since Case sits alongside them rather than replacing any existing write path. Reverting the Go-side `dbhandler`/`contacthandler` changes from §3.3 alongside the schema revert restores the original non-nullable `InteractionID` contract cleanly, since no external caller depends on the nullable form once this feature is rolled back. The `Conversation.Metadata` addition (§4.4) is likewise purely additive and inert to conversation-manager's own dispatch logic; reverting it drops the inbound-hint optimization and inbound SMS during an active call reverts to spinning up a disconnected Case, exactly the pre-existing (pre-§4.4) behavior. Because §4.4's write path is get-only (never creates a Conversation, per the round-7 correction), there are no orphaned Conversation rows or falsely-fired `conversation_created` webhooks to clean up on rollback either — the only state `Metadata.ContactCaseID` ever touches is a `Conversation` row that would already have existed independently of this feature, and only that one field on it is written.
