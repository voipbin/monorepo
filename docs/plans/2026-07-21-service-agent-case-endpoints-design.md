# service_agents/contact_cases: Agent-facing Case endpoints (message history + owner assignment)

- Ticket: NOJIRA
- Author: Hermes (CPO), for pchero (CEO/CTO)
- Status: DESIGN APPROVED — closed after 4 review rounds (8 independent
  reviews total, 2 consecutive fully clean rounds). Ready for
  implementation.
- Context: pchero has decided square-talk will drop `Conversation` as its
  primary work unit and replace it with `Case` (contact-manager's Case
  entity). This design covers the two backend gaps identified as blocking
  that migration's first phase. It does NOT cover the square-talk UI
  itself, and does NOT cover removing/deprecating Conversation — those are
  separate, later phases.

## 1. Problem

Case (`bin-contact-manager`, `models/kase`) is a per-peer session header:
open/closed lifecycle, notes, tags, contact attribution, and
(`commonidentity.Owner`) an owner field. Today every Case endpoint in
`bin-api-manager` is gated `PermissionCustomerAdmin|PermissionCustomerManager`
only (`pkg/servicehandler/case.go`, `case_note.go`, `case_message.go`) —
confirmed by reading every `CaseXxx` function in that package. There is
**no `/service_agents/contact_cases/*` surface at all** (zero files matching
that name in `bin-api-manager/server` or `openapi/paths/service_agents/`).

square-talk is an Agent-facing app (most callers hold only
`CustomerAgent`, not Admin/Manager). Per the platform's own established
rule (`bin-openapi-manager/CLAUDE.md`'s "never widen the Admin/Manager
endpoint" section), square-talk must call a *new*
`/service_agents/contact_cases/*` surface — never a relaxed version of the
existing Admin/Manager one.

Two capabilities square-talk needs to replace Conversation as the primary
work unit do not exist in ANY form today (not even Admin/Manager-gated):

1. **Case-scoped message history.** `POST /contact_cases/{id}/messages`
   (send) exists; there is no read path. A chat UI needs to render a
   scrollback thread, not just fire-and-forget sends.
2. **Case owner assignment.** `Case.Owner` (`commonidentity.Owner`,
   embedded in `kase.Case`) is **never written by any production code
   path today** — verified by grepping every construction of `Owner{...}`
   in `bin-contact-manager`: `Create` (create.go) explicitly zero-values
   it (its own test asserts "no auto-assignment"), `insertWithRetry`
   (getorcreate.go) never sets it on the constructed `newCase`, and
   `Continue` (lifecycle.go) only *reads* `source.OwnerType`/
   `source.OwnerID` for its authorization check, never writes them. The
   only places `Owner{...}` is populated are test fixtures. The
   `owner_type`/`owner_id` DB columns exist and are read/filtered on
   (`CaseList`'s owner filter), but nothing in application code writes
   them — this is a fully dead-write field today, not merely a
   field lacking a public RPC. Confirmed: no `CaseAssign`/`CaseUnassign`
   RPC exists anywhere in `pkg/casehandler` or `pkg/listenhandler`
   (`v1_cases.go`'s complete route list has no assign/unassign handler).

## 2. Gap A: Case message history — architectural decision required

### 2.1 Why this isn't a simple "add a GET" ticket

`Case` has no stored "self" address. `CaseMessageSend`
(`bin-api-manager/pkg/servicehandler/case_message.go`) only learns the
self address (`source`) at send time, as caller-supplied input, and uses
it to resolve/create a `Conversation` via
`ConversationV1ConversationGetOrCreateBySelfAndPeer(customerID, type, "",
selfAddr, peerAddr)`. Nothing persists that mapping onto the Case itself
except a **fail-open, best-effort** write:
`ConversationV1ConversationUpdateMetadata(conv.ID,
Metadata{ContactCaseID: &caseID})` — one-directional (Conversation points
at Case; Case does not point at Conversation), and explicitly allowed to
silently fail (§4.5 step 5's fail-open comment).

Consequence: **a single Case can, in principle, span multiple
Conversations** — one per distinct `source` number the case has ever sent
from — and there is no authoritative index from `case_id` back to the set
of `conversation_id`s it touched (only the reverse, best-effort pointer).
This is the real reason `case_id` was never added to
`ContactManagerInteraction`'s read contract either (per the 2026-07-09
square-admin Case design's Gap #2 — the same underlying gap, previously
scoped out of that PR).

### 2.2 Options considered

**Option 1 — Reuse Conversation's message store via the metadata
back-pointer, list-then-merge.** Query
`ConversationV1ConversationList(filters={FieldDeleted: false})`, filter
client-side (or via a new dbhandler filter) for `Metadata.ContactCaseID
== caseID`, then fan out `ConversationV1MessageList` per matching
conversation and merge/sort by `tm_create`. Correct given today's actual
data model, but O(N) conversations scanned unless a new filter/index is
added, and depends on the fail-open write never having silently failed
(§2.1) — a case could have a "phantom" conversation with messages that
never got tagged.

**Option 2 — Make the Case→Conversation link authoritative:
`Case.ConversationIDs []uuid.UUID` (or a join table), written
transactionally (not fail-open) at send time.** Removes the phantom-link
risk and turns the metadata lookup into an O(1) indexed list. Requires a
migration on `contact_cases` (or a new `case_conversations` join table if
Case can legitimately span multiple Conversations, which §2.1 established
it can) and requires changing `CaseMessageSend`'s step 5 from fail-open
best-effort to a required, atomic write.

**Option 3 — Case gets its OWN message store, independent of
Conversation** (mirroring how Case already has its own Notes, not reusing
some other service's note concept). `POST /contact_cases/{id}/messages`
would write to a new `contact_case_messages` table instead of delegating
to conversation-manager, and Case would stop depending on
conversation-manager for message send/receive at all. This is the
cleanest long-term shape if Conversation is genuinely being dropped
platform-wide (not just from square-talk's UI) — but is by far the
largest scope: it re-implements delivery/status tracking
(`progressing`/`done`/`failed`) that conversation-manager already owns,
and requires resolving how *inbound* messages (today ingested by
conversation-manager's LINE/WhatsApp/SMS webhook handlers) get attributed
to a Case instead of/in addition to a Conversation.

### 2.3 Recommendation

Start with **Option 2** for this phase. Reasoning:

- Option 1 ships fastest but bakes in the phantom-link risk permanently
  and adds an unindexed scan — acceptable for a stopgap, not for square-
  talk's primary work-unit surface.
- Option 3 is very likely the right *end state* if Conversation is
  actually being dropped, but re-deciding conversation-manager's inbound
  message ingestion architecture is a separate, much larger design that
  shouldn't block Case adoption in square-talk today. Flagging this
  explicitly as the probable Phase 2/3 direction, not deciding it now.
- Option 2 is a bounded, correct fix: it turns today's best-effort
  metadata write into a real invariant, at the cost of one migration and
  one behavior change (send fails if the link-write fails, instead of
  silently proceeding).

**Resolved with pchero (2026-07-21):** "Conversation is being dropped"
means square-talk's UI stops surfacing the Conversation concept —
`bin-conversation-manager` continues to exist and serve as Case's
message-delivery backend indefinitely. Option 2 is therefore the durable
design for this area, not a stepping stone toward Option 3. Option 3
(Case owning its own message store, independent of conversation-manager)
is out of scope, not just deferred.

### 2.4 Scope for THIS design (Option 2, confirmed)

- `bin-contact-manager`: add `case_conversations` join table
  (`case_id`, `conversation_id`, `tm_create`, unique on the pair) — a join
  table rather than a single FK column on `Case`, since §2.1 established
  one Case can span multiple Conversations (multiple `source` numbers).
  New dbhandler methods: `CaseConversationAdd`, `CaseConversationsByCase`.
  Insert-only (no `CaseConversationDelete` in this design's scope) and
  hard-delete-shaped (no `tm_delete` column) — this matches the existing
  join-table precedent in this service, `contact_tag_assignments`
  (`tag_assignment.go`'s `TagAssignmentDelete` issues a real SQL
  `DELETE`, no soft-delete), not `Case`'s own soft-deleted
  `contact_case_notes`. This codebase does not have a single universal
  soft-delete convention for junction tables specifically; the join-table
  precedent, not the notes precedent, is the relevant one here.
  **Why a join table and not a JSON array column** (this repo's OWN
  closest precedent, `Case.TagIDs`, deliberately uses a JSON array —
  see kase.go's VOIP-1254 doc comment): Tag membership is small,
  bounded, and never needs a reverse/indexed lookup (nothing queries
  "which Cases have Tag X" as a hot path). Conversation membership is
  monotonically growing over a Case's lifetime and, per §2.4 below,
  MUST support an indexed reverse lookup and correct pagination across
  potentially many rows — properties a JSON column doesn't give
  cheaply. The two fields are structurally similar (both 1:N off
  `Case`) but have different access patterns, which is why they get
  different storage.
- `bin-api-manager`'s `CaseMessageSend` (`case_message.go`): change step 5
  from fail-open `ConversationV1ConversationUpdateMetadata` to a required
  write recording `(caseID, conv.ID)` in the new join table — if this
  write fails, the whole send fails (no more silent phantom links).
- New `ServiceHandler.CaseMessageList(ctx, a, caseID, size, token)`:
  tenant-checks the case (`caseGet`), loads its conversation ID set from
  the join table, fans out `ConversationV1MessageList` per conversation
  (or a new conversation-manager RPC taking multiple IDs, TBD at
  implementation — an internal fan-out-vs-batched-RPC choice that does
  not change the wire contract below), merges by `tm_create` DESC,
  returns a page.
- **Pagination-token contract (load-bearing, decided here, not left
  TBD):** `token` is a **composite cursor `(tm_create, message_id)`**,
  not a bare `tm_create` value — `tm_create` alone is not guaranteed
  unique across (or even within) conversations (concurrent inbound
  delivery across two conversations, or the backfill inserting
  historical messages with coarse timestamps, can produce ties). The
  merge algorithm re-queries every conversation in the case's
  `case_conversations` set with `WHERE (tm_create, id) < (token_time,
  token_id)` (lexicographic tuple comparison) on each fan-out call, then
  merges by the same `(tm_create, id)` DESC ordering — this is correct
  and resumable regardless of how many conversations the case spans or
  how many messages share a `tm_create` value, because ties are broken
  deterministically by `message_id` on both sides of the page boundary,
  so no tied message is silently skipped or duplicated across pages.
  `message_id` is a random UUIDv4 (`commonidentity.Identity.ID`, per
  `bin-conversation-manager/models/message/message.go`) — the tie-break
  ORDER it imposes is arbitrary/non-chronological, and that is fine: a
  keyset-pagination tie-breaker only needs to be stable and applied
  identically on both sides of the page boundary, not semantically
  meaningful. Implementation note (standard k-way-merge invariant, not a
  new decision): each per-conversation fan-out query must independently
  fetch at least `size` rows before the cross-conversation merge/sort
  step, or the merge can under-fill a page when one conversation has
  many eligible rows and another has few.
- **Backfill for pre-existing Cases (explicitly in scope, concrete
  execution plan):** Cases created before this migration ships have zero
  rows in the new `case_conversations` table — their only link to a
  Conversation is the old, possibly-stale `Metadata.ContactCaseID`
  back-pointer (§2.1). A one-time backfill script, run once at deploy
  time as a required step of this PR's rollout (not a follow-up):
  - **Idempotency:** the script issues `INSERT IGNORE` (MySQL upsert
    semantics against `case_conversations`' existing unique constraint on
    `(case_id, conversation_id)`) — a partial-failure retry or an
    accidental second run is a safe no-op for every row already
    inserted, never a constraint-violation abort.
  - **Data source / pagination (inherits and resolves §2.2's open
    scan-cost question, does not leave it open):** the script requires a
    new, backfill-only dbhandler query on `bin-conversation-manager`,
    `ConversationListByContactCaseIDNotNil(token, size)`, added
    specifically for this one-time job — a `WHERE
    JSON_EXTRACT(metadata, '$.contact_case_id') IS NOT NULL` scan with
    `tm_create`-cursor pagination (same cursor shape as every other list
    in this codebase), run to completion once. This is deliberately NOT
    the same code path as `CaseMessageList`'s steady-state read (which
    never scans all Conversations, only a Case's own
    `case_conversations` rows) — the O(N) scan cost is acceptable here
    specifically because it runs exactly once, not per-request.
- **Event/webhook publication (explicitly out of scope for this PR):**
  no new customer-facing webhook event is added for Assign/Unassign/new-
  Case-message in this design (see §4). Flagged so a reader doesn't
  assume it's included.
- OpenAPI: new `GET /service_agents/contact_cases/{id}/messages` — new
  file `openapi/paths/service_agents/contact_cases_id_messages.yaml`,
  reusing `ConversationManagerMessage`'s existing response schema.

## 3. Gap B: Case owner assignment (service-agent surface)

### 3.1 Scope

**Correction from review:** there is no existing "self-assign" precedent
to mirror. `bin-conversation-manager`'s Conversation has no dedicated
assign endpoint at all — ownership is set only via the generic
`ServiceAgentConversationUpdate`, which is gated
`PermissionCustomerAdmin|PermissionCustomerManager` ONLY (a plain agent
cannot self-assign a Conversation today). Only `Unassign` has an
agent-self-service precedent (`ServiceAgentConversationUnassign`: admin/
manager may unassign anyone, the owning agent may self-unassign).
Therefore the self-assign capability proposed below for Case is a **new,
more permissive authorization decision**, not a mirror of an existing
pattern — justified here on its own merits (a Case-based square-talk
needs agents to be able to pick up an unowned case themselves, the same
"claim this" motion a queue/ticketing UI needs; Conversation never
needed this because square-talk's existing assignment flow routes
through `AssignAgentDropdown`, an admin/manager-driven UI action, not an
agent self-claim).

Net-new both at the RPC layer (contact-manager has none today) and the
service-agent HTTP layer:

- `bin-contact-manager`: new `casehandler.Assign(ctx, customerID, id,
  ownerType, ownerID)` / `Unassign(ctx, customerID, id)`, backed by new
  `dbhandler.CaseUpdateOwner`/`CaseClearOwner` (customer_id-scoped UPDATE,
  same tenant-isolation shape as `CaseUpdateStatusClosed`). New listen
  routes `POST /v1/cases/{id}/assign`, `POST /v1/cases/{id}/unassign`.
- `bin-api-manager`: new `ServiceAgentCaseAssign`/`ServiceAgentCaseUnassign`
  in a new `pkg/servicehandler/serviceagent_case.go`. **Explicit
  authorization rule for Assign (spelled out, not left to analogy):**
  - A plain agent (`PermissionAll` only) may assign a case **only to
    themselves** (`ownerID` in the request must equal `a.AgentID()`;
    any other `ownerID` from a plain agent returns
    `ErrPermissionDenied`) — this is the "claim an unowned case" motion.
  - Admin/manager callers may assign a case to **any** agent of the
    customer (reassignment), matching `ServiceAgentConversationUpdate`'s
    existing admin/manager-can-set-arbitrary-owner precedent.
  - Reassigning an already-owned case: a plain agent may only assign
    when the case is currently unowned (`OwnerID == uuid.Nil`) **OR
    already owned by that same agent** (idempotent self-claim / redundant
    re-confirmation is a harmless no-op, not a denial — an agent
    retrying an assign call, e.g. after a network timeout, must not be
    rejected just because their own prior call already succeeded). A
    plain agent may never assign a case another, *different* agent
    currently owns. Admin/manager bypass this check entirely
    (consistent with their unrestricted access elsewhere in this
    package).
  - **Unassign of an already-unowned case:** denied for any non-admin/
    manager caller (there is no "owning agent" to match against, so the
    self-unassign carve-out never applies) — admin/manager may still
    call it as a no-op.
  - **Unassign** keeps the already-correct, precedented rule: admin/
    manager may unassign any case; the owning agent may self-unassign;
    any other agent gets `ErrPermissionDenied` — this part IS a direct
    mirror of `ServiceAgentConversationUnassign`.
- OpenAPI: new `openapi/paths/service_agents/contact_cases_id_assign.yaml`
  and `..._unassign.yaml`.

### 3.2 Open question

`commonidentity.Owner`'s doc comment on `kase.Case` (§ design
2026-07-07) says it's reused "exactly as conversation-manager already
does it" and explicitly states it is "NEVER cleared by closing a Case —
load-bearing invariant for /continue's authorization." Unassign must
preserve this: unassign clears current-owner (agent working it now) but
must NOT be confused with anything `/continue`'s ownership check reads —
verify at implementation time that `/continue`'s owner check
(`lifecycle.go`'s `Continue`, `isOwner := source.OwnerType ==
callerType && source.OwnerID == callerID`) reads the CLOSED case's owner
at close time, not a live-mutable field that a later unassign could
retroactively affect. (Should be fine since Continue reads the *source*,
already-closed case, and Unassign would only ever be called on an *open*
case — but this needs an explicit test, not just inference.)

## 4. Explicitly out of scope for this design

- square-talk UI itself (routing, ChatRoom/MessageList repointing to
  Case, ContactDetail Cases section, ChatHeader Case panel) — separate
  frontend design, sequenced after this backend work lands.
- Any decision to deprecate or remove `bin-conversation-manager` — see
  §2.3's open question; not decided here.
- Case-scoped **notes/tags** service-agent exposure — already gated
  Admin/Manager only server-side (`case_note.go`); whether agents need
  note read/write is a product question for the square-talk UI design,
  not this backend-endpoints design.
- Real-time (WebSocket) push for new Case messages — out of scope; square-
  talk's existing conversation WS topic pattern would need its own
  follow-up design once the UI phase is scoped.
- New customer-facing webhook/event publication for Assign/Unassign/new-
  Case-message (see §2.4) — none added in this design.

## 5. Test plan

- `bin-contact-manager`: dbhandler tests for `CaseConversationAdd`/
  `CaseConversationsByCase`, `CaseUpdateOwner`/`CaseClearOwner`
  (customer-id tenant isolation — cross-tenant UPDATE must affect 0 rows);
  casehandler tests for `Assign`/`Unassign`.
- `bin-api-manager`: servicehandler tests for `CaseMessageList` (multi-
  conversation merge/sort correctness, tenant isolation via caseGet),
  `ServiceAgentCaseAssign`/`Unassign` (plain-agent-permission identity
  accepted, not just admin/manager); `CaseMessageSend`'s changed fail-open
  → required-write behavior (assert send fails if the join-table write
  fails, where today it would have silently succeeded).
- Full verification workflow in `bin-contact-manager` and
  `bin-api-manager`; `bin-openapi-manager go generate` first.

## 6. Review history

**Round 1 (2 parallel independent reviews: technical accuracy,
implementability) — both REQUEST CHANGES.**

Technical review found: (a) §1's claim that Owner is "write-only via two
narrow, indirect paths" was inaccurate — Owner is in fact never written
by any production code path today (Create explicitly zero-values it,
insertWithRetry never sets it, Continue only reads it); corrected in §1;
(b) §3.1's claim that the proposed self-assign authorization mirrors
`ServiceAgentConversationUnassign`'s precedent was wrong for the
*assign* side — Conversation has no self-assign precedent at all
(assignment is Admin/Manager-only via the generic Update); corrected in
§3.1 with an explicit, non-precedent-claiming justification and a fully
spelled-out authorization matrix; (c) the join-table proposal in §2.4
was architecturally reasonable but didn't address the repo's own
sibling precedent (`Case.TagIDs`'s JSON-array choice) — corrected with
an explicit rationale for why the two 1:N relations get different
storage.

Implementability review found: (a) `CaseMessageList`'s pagination-token
semantics across merged conversations were undefined, a load-bearing gap
— corrected in §2.4 with an explicit cursor contract; (b) backfilling
`case_conversations` for pre-existing Cases was entirely unaddressed,
risking silently empty scrollback for every Case older than this
migration — corrected in §2.4, made an explicit, required deployment
step; (c) event/webhook publication scope was unstated — corrected, made
explicit in §2.4/§4 as out of scope for this PR; (d) Assign's
self-vs-others authorization rule was asserted by analogy rather than
spelled out — corrected in §3.1 with a full authorization matrix
(self-assign only when unowned for plain agents; admin/manager
unrestricted reassignment).

No architectural rewrite needed — all corrections were scoped,
additive fixes to specific claims and specific missing details,
consistent with both reviewers assessing the document as fundamentally
sound in its option analysis and phasing.

**Round 2 (2 parallel independent fresh reviews: technical
re-verification, implementability re-verification) — technical APPROVED,
implementability REQUEST CHANGES.**

Technical review independently re-verified every Round 1 correction
against source directly (Owner's dead-write status, the TagIDs
join-table-vs-JSON rationale, the Conversation assign/unassign asymmetry)
and confirmed all accurate; found no new fabricated or inaccurate claims
in a fresh full-document pass.

Implementability review found three remaining gaps in the Round 1
fixes, all corrected now: (a) the pagination-token contract used a bare
`tm_create` cursor with no tie-breaker — concurrent inbound messages
across two conversations (or coarse backfill timestamps) sharing the
same `tm_create` could be silently dropped at a page boundary; corrected
to a composite `(tm_create, message_id)` cursor with lexicographic tuple
comparison; (b) the backfill plan named a trigger point and data source
but left idempotency and the underlying scan-cost question (inherited
from §2.2) unresolved; corrected with an explicit `INSERT IGNORE`
idempotency guarantee and a dedicated one-time backfill-only dbhandler
query, explicitly distinguished from `CaseMessageList`'s steady-state
read path so the O(N) scan cost is scoped to a single one-time job; (c)
the Assign authorization matrix's literal if/else translation would have
denied an agent redundantly re-confirming ownership of their own
already-owned case (a plausible idempotent-retry UI flow); corrected to
explicitly allow self-reassign-to-self as a no-op, and added an explicit
rule for Unassign-of-an-unowned-case by a non-admin caller.

**Round 3 (2 parallel independent fresh reviews: implementability final
re-verification, whole-document fresh-reader pass) — both APPROVED, first
clean round.**

Implementability re-verification confirmed: the composite
`(tm_create, message_id)` cursor genuinely eliminates the skip/dup risk
(verified `message_id` is a stable random UUIDv4 on the real `Message`
model, which is sufficient for a keyset tie-breaker even though the
order it imposes is non-chronological); the backfill's `INSERT IGNORE` +
dedicated one-time scan query is concrete enough for engineer handoff;
the Assign matrix has zero remaining ambiguous input combinations when
re-derived from scratch. Two small, explicitly non-blocking
documentation nits were folded in anyway (tie-break-is-arbitrary-and-
that's-fine clarification; standard k-way-merge minimum-fetch-per-source
note) — both incorporated above in §2.4.

Whole-document fresh-reader pass found no internal contradictions
between sections, confirmed the document's stated scope (title, Context
bullet, §4 out-of-scope list) matches what §2-§3 actually describe
building, independently re-verified several §1-§3 code-fact claims
against the real repo (all confirmed accurate), and confirmed no
remaining implementer-must-invent decision exists outside the one
explicitly-flagged §3.2 test requirement.

This is the first fully clean round (both angles APPROVED with zero
REQUEST CHANGES items). Per the review-loop's 2-consecutive-clean-round
gate, one more confirming round is required before this document is
considered closed for implementation handoff.

**Round 4 (2 parallel independent fresh reviews: implementer-lens
re-verification, adversarial whole-document pass) — both APPROVED,
second consecutive clean round. Loop closed.**

Implementer-lens re-verification independently re-confirmed every
net-new claim from Round 3's fixes (the UUIDv4 tie-breaker's
"stable-but-arbitrary" property, the k-way-merge invariant, the
backfill's `INSERT IGNORE` idempotency) against the real repo one more
time, and re-confirmed §1's Owner dead-write claim by file:line citation.
Zero remaining issues found; the document is implementable as written
with zero invented decisions.

Adversarial pass checked three conventions not directly examined by any
prior round: (a) the §3.1 authorization matrix against the real
`hasPermission`/`PermissionAll` idiom used elsewhere in
`bin-api-manager` (`serviceagent_conversation.go`,
`serviceagent_tag.go`, `serviceagent_interaction.go`) — confirmed the
two-tier pattern matches exactly; (b) whether `case_conversations`
needs a `tm_delete` column given this codebase's soft-delete
conventions — confirmed the relevant precedent is the join-table
pattern (`contact_tag_assignments`, hard-delete), not the notes pattern,
and the design is correct as written; folded in an explicit citation of
that precedent into §2.4 so the choice reads as deliberate rather than
overlooked; (c) the proposed OpenAPI file names against the real
`<resource>_id_<action>.yaml` convention in
`openapi/paths/service_agents/` — confirmed exact match. No blocking
issues found in either pass.

**Loop closed. Document is APPROVED for implementation handoff.**
