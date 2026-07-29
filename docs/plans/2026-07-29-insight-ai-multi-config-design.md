# Insight AI: allow multiple configs per customer, single active one

**Date:** 2026-07-29
**Ticket:** NOJIRA
**Service:** `bin-ai-manager` (backend), `bin-api-manager`, `bin-openapi-manager`, `square-admin` (frontend)
**Revision:** r4 — addresses round-1, round-2, and round-3 design review (see "Review history" at bottom)

## Problem

SQUARE-23 (merged 2026-07-29, same day) added a DB-level constraint that
allows at most one non-deleted `type=insight` AI per customer
(`ai_ais.active_insight_key` generated column + `uq_ai_active_insight_key`
unique index; see `2026-07-29-insight-ai-one-per-customer-design.md`). It
has **not been applied to any non-local environment yet** (pending its own
audit/cleanup step) — confirmed by that design's "Open items" section.

In practice this blocks a legitimate workflow: an operator wants to prepare
or experiment with a second Insight AI configuration (different prompt,
model, tools) before switching production traffic to it. Today they cannot
even save a second draft — `POST /ais` returns `409
AI_INSIGHT_ALREADY_EXISTS` unless they delete the existing one first.

## Goal

- Allow a customer to have **multiple** `type=insight` AI records.
- Exactly **one** of them may be "active" — the one the Case Insight
  Assistant panel auto-attaches to a case (`ServiceAgentAIcallCreate`
  resolution logic).
- **AICall Test** (square-admin's "Test Agent" button, `TestAgentSheet.js`)
  keeps working against any Insight AI regardless of active status — it
  already targets a specific `assistance_id` directly, not through the
  active-resolution path. No backend change needed; covered by a
  regression test only.

## Non-goals

- No change to `type=normal` AI behavior.
- No bulk/multi-activate.
- No re-run of SQUARE-23's duplicate-customer audit/cleanup — moot, since
  SQUARE-23's migration is being replaced before it's ever applied (see
  §1).

## Design

### 1. Schema change — chain a new migration on top of SQUARE-23

r2 proposed amending the already-merged `f4c4c2407cee` (SQUARE-23) in
place to avoid a double table rewrite. Round-2 review rejected this:
Alembic keys on revision ID, not content — any database that has already
run `f4c4c2407cee` (plausible for local dev/CI DBs, since SQUARE-23 merged
to main the same day) would silently skip the amended `upgrade()` on
`alembic upgrade head`, permanently missing `is_insight_active` with no
error. Amending a migration file already merged to main also rewrites the
meaning of a shipped commit, which is its own review hazard independent of
the stamping bug.

**Decision: chain a new migration instead**, accepting the one-time cost
of a second `ai_ais` table rewrite. This is correct by construction
regardless of what has or hasn't been applied where — no environment can
be silently left in a partial state. Whether to batch this rewrite into
the same maintenance window as SQUARE-23's own (still-pending) rollout is
a deployment-sequencing call for 대표님/ops, not something fixed by this
design; flagged again in "Open items" below.

New migration (generated via `alembic revision` against the
then-current head — confirmed today to be `f4c4c2407cee`; re-verify at
implementation time in case another migration has landed in the
interim):

```sql
ALTER TABLE ai_ais
  ADD COLUMN is_insight_active BOOLEAN NOT NULL DEFAULT FALSE;

DROP INDEX uq_ai_active_insight_key ON ai_ais;
ALTER TABLE ai_ais DROP COLUMN active_insight_key;

ALTER TABLE ai_ais
ADD COLUMN active_insight_key BINARY(16) GENERATED ALWAYS AS (
  IF(type = 'insight' AND tm_delete IS NULL AND is_insight_active = TRUE,
     customer_id, NULL)
) STORED;

CREATE UNIQUE INDEX uq_ai_active_insight_key ON ai_ais(active_insight_key);
```

(`NOT NULL DEFAULT FALSE`, not nullable — a tri-state NULL/false split for
"only meaningful on insight rows" is unenforceable in practice: this
codebase's `sql.NullBool` mapping already collapses NULL to `false` on
scan, and `AICreate` writes `false` for every row regardless of type. The
generated column's own `type='insight'` guard is what actually scopes the
semantics, not the column's nullability.)

`upgrade()`/`downgrade()` use the same `_column_exists`/`_index_exists`
idempotency guards as `f4c4c2407cee`. `downgrade()` restores SQUARE-23's
original `active_insight_key` definition (customer_id-keyed, no
`is_insight_active` condition) and drops the new column — this is a
one-way door in practice once a customer has 2+ insight AIs (restoring
"one per customer" would then fail with 1062 on `CREATE UNIQUE INDEX`);
document this explicitly in the downgrade's code comment so a future
maintainer doesn't attempt it blind against a populated table.

The SQLite test-schema stand-in (`bin-ai-manager/scripts/database_scripts_test/table_ai_ais.sql`)
must be updated in the same change — SQUARE-23's design doc already
established this file needs a `CASE WHEN`-based equivalent since MySQL
`IF(...)` isn't valid SQLite; this revision's version needs the
`is_insight_active` column added to that same stand-in.

### 2. Model / API surface

- `models/ai/main.go`: add `IsInsightActive bool` field (`db:"is_insight_active"`).
- `models/ai/field.go`: add `FieldIsInsightActive` constant.
- `models/ai/filters.go`: add `IsInsightActive bool` to `FieldStruct` with
  `filter:"is_insight_active"`. **Required**, not optional plumbing: this
  service's `pkg/listenhandler/v1_ais.go` re-parses incoming filters via
  `utilhandler.ConvertFilters[ai.FieldStruct, ai.Field]`, which treats
  `FieldStruct` as an allowlist and *silently drops* any filter key not
  declared there (`bin-common-handler/pkg/utilhandler/filters.go`) — no
  error, no warning. Without this addition, §4's `is_insight_active=true`
  query below would silently degrade to "most recently created," making
  the whole feature inert with no test failure to catch it unless the
  filter's presence is explicitly asserted (see §6). This is a separate
  allowlist from `bin-api-manager`'s `convertAIFilters`/
  `ConvertMapToTypedMap` (which only governs the api-manager→ai-manager
  RPC hop) — both layers need the field, not just one.
- `models/ai/webhook.go`: add `IsInsightActive bool` (no `omitempty`) to
  `WebhookMessage` (root CLAUDE.md: the wire contract is driven by
  `WebhookMessage`, not the internal struct — RST docs must reflect this
  field too). `omitempty` would drop the field from the wire entirely when
  `false`, making "inactive" indistinguishable from "field absent" for
  any client parsing the JSON.

New endpoint, `pkg/dbhandler/ai.go`:

```go
func (h *ai) AIActivateInsight(ctx context.Context, id uuid.UUID) (*ai.AI, error)
```

Following the transactional pattern already used by `AIAcceptProposal`
(`pkg/dbhandler/aipromptproposal.go:221-356`: `h.db.BeginTx` → `SELECT ...
FOR UPDATE` scoped to the customer's currently-active insight row → clear
its `is_insight_active` → set target row's `is_insight_active = TRUE` →
`Commit` → deferred cache refresh for **both** rows via `aiUpdateToCache`
using `context.Background()`, matching `aipromptproposal.go:231-243`).
Cache refresh for both the deactivated and newly-activated row is
mandatory — `AIGet` (`pkg/dbhandler/ai.go`) is cache-first per this
service's cache-invariant rule (`bin-ai-manager/CLAUDE.md`); skipping
either refresh leaves the Case panel resolving a stale AI indefinitely.

Handler method `aihandler.ActivateInsight(ctx, id)`:
- 400 if the target AI's `type != insight`.
- Otherwise delegates to `dbhandler.AIActivateInsight`.
- A concurrent-activation race (two calls racing for the same customer)
  surfaces as the unique-index collision; translate via
  `IsErrDuplicate` to a **new** error reason distinct from SQUARE-23's
  `AI_INSIGHT_ALREADY_EXISTS` (whose message — "delete it before creating
  another" — is actively wrong advice under this design). New reason:
  `AI_INSIGHT_ACTIVATION_CONFLICT`, message "Another activation is in
  progress for this customer's Insight AI; retry." SQUARE-23's original
  `AI_INSIGHT_ALREADY_EXISTS` translation exists in **two** places —
  `aihandler.Create` and `dbUpdate` (`db.go:211-216`) — and both become
  effectively unreachable once creates always default inactive (§3): a
  plain create or update can no longer collide with an active row, since
  nothing but `AIActivateInsight` ever sets `is_insight_active = TRUE`.
  Remove both translations rather than leaving stale, now-misleading copy
  ("delete it before creating another") as dead code; the unique index
  itself still protects the invariant, it just can no longer be hit from
  `Create`/`Update`.

Full plumbing required (not just `bin-ai-manager`), per this monorepo's
layering rules:
- `bin-ai-manager/pkg/listenhandler`: new route, unwrapped domain args in,
  domain `*ai.AI` out (style A, per root CLAUDE.md's transport-DTO rule).
- `bin-openapi-manager/openapi/paths/ais/`: new
  `POST /ais/{id}/activate_insight` path spec.
- `bin-api-manager`: route + handler passthrough.
- `bin-common-handler/pkg/requesthandler`: new RPC client method.
- `bin-ai-manager/docs/architecture.md`: routing-table sync (root CLAUDE.md
  service-docs-sync rule).
- `bin-api-manager/docsdev/source/`: RST docs for the new endpoint (root
  CLAUDE.md RST-sync rule) — rebuild HTML, force-add build output.

`GET /ais?type=insight` (existing list) is unaffected; response now
includes `is_insight_active` via the `WebhookMessage` change above.

### 3. Create/delete/type-change side effects on `is_insight_active`

Dropped from r1: "auto-activate on first create." Round-1 review found it
underspecified (doesn't distinguish "first ever" from "no currently
active") and racy (two concurrent first-creates both see zero-active and
both attempt to become active, one loses to the unique index and gets a
confusing 409 on a plain create). Simpler and race-free: **every newly
created Insight AI defaults to inactive, no exceptions.** The Case panel's
existing "no active found → fall back to most-recently-created" logic
(§4) already covers the zero-active-AI case correctly without needing a
special-cased create path.

`AIDelete` and `AIUpdate` (`pkg/dbhandler/ai.go`) do blind field-map
writes today with no current-row read (`AIUpdate` maps only the fields the
caller passed; `AIDelete` sets `tm_update`/`tm_delete`). Making the
`is_insight_active` clear **conditional** on "was this row active" or
"is type changing away from insight" would require adding a read before
the write, which races against a concurrent `AIActivateInsight` (§2) —
the activation could commit between that read and this write, and the
clear would silently undo a just-won activation.

Avoid the race entirely by making both clears **unconditional**, not
conditional on current state:
- `AIDelete`: always include `is_insight_active = false` in its
  `SetMap`/`UPDATE`, regardless of the row's current type or active
  status. Harmless no-op on rows where it's already false; no read
  needed.
- `AIUpdate`/`buildUpdateFields`: whenever the resolved `aiType` for the
  update is anything other than `insight` (i.e. the row is `normal`, or
  is changing to `normal`), always include `is_insight_active = false` in
  the same `UPDATE` statement — again unconditionally, not "if it was
  previously true."

This also closes a gap the generated column alone doesn't cover: after
§1's `tm_delete IS NULL` guard frees a deleted row's unique-index slot,
the row's own `is_insight_active` field would otherwise still read `TRUE`
if left uncleared — and per §4's fix below, the Case-panel resolution
query must filter `deleted=false` in addition to `is_insight_active=true`,
precisely because a stale `TRUE` on a soft-deleted row is otherwise
indistinguishable from a live one to any caller that reads the field
directly instead of going through the generated-column-backed list
query.

### 4. Case panel resolution logic change

`ServiceAgentAIcallCreate`'s Insight AI lookup
(`bin-api-manager/pkg/servicehandler/serviceagent_aicall.go`,
`resolveInsightAIID`) currently calls `AIV1AIList(ctx, "", 100, filters)`
sorted `tm_create desc` and takes the first result — under SQUARE-23's "at
most one insight row" constraint that 100-row page was always sufficient.
This design removes that cap entirely, so a customer with more than 100
Insight AI configs could have their active one fall outside that page,
silently selecting the wrong AI with no error — exactly the failure class
this whole feature exists to prevent.

Fix: split into two queries, not one list-then-filter:
1. `AIV1AIList(ctx, "", 1, filters{type: insight, is_insight_active: true,
   deleted: false})` — size 1, since at most one row can ever match.
   `deleted: false` is required per §3's note above — a stale `TRUE` on a
   soft-deleted row must not be selected. Requires the `filters.go`
   addition in §2 on the `bin-ai-manager` side, not just the api-manager
   conversion layer — see that section for why both are needed.
2. If that returns zero rows (data anomaly, or a brand-new customer whose
   only Insight AI hasn't been activated yet — see §5), fall back to the
   existing `AIV1AIList(ctx, "", 100, filters{type: insight, deleted:
   false})` sorted by `tm_create desc`, taking the first result — same
   defense-in-depth fallback SQUARE-23's design already established.

Add a test for the >100-Insight-AIs case explicitly (§6). Update
`resolveInsightAIID`'s existing doc comment, which still describes the
SQUARE-23-era "single type=insight AI, most-recently-created if 2+ exist"
behavior, to match this new two-query resolution.

### 5. Frontend (square-admin)

- Insight AI list/cards (`ais_detail.js` area): "Active" badge, "Activate"
  button on inactive cards → calls the new endpoint, refreshes list
  (optimistic update rolled back on failure).
- **Zero-active state must be visible, not silent.** Since creates always
  default inactive (§3), a customer's first (and possibly only) Insight
  AI shows no "Active" badge even though the Case panel is already using
  it via the §4 fallback — otherwise this reads as a bug ("I made an
  assistant, why isn't it active?"). Determine this state via the same
  dedicated `is_insight_active=true` query as §4 (not by inspecting only
  the currently-rendered page of the list, which could paginate past 100
  and miss it) and render an explicit "Currently used (no assistant
  activated yet)" affordance on the fallback-selected AI.
- `TestAgentSheet.js`: no change — regression-tested only.
- English-only UI copy (admin default locale is English — already flagged
  this session as a separate bug elsewhere in the Case panel).

### 6. Testing

- SQLite stand-in (`table_ai_ais.sql`) updated first — every test below
  depends on it (§1).
- `dbhandler`:
  - Two `type=insight` creates for the same customer both succeed, both
    default inactive.
  - `AIActivateInsight`: activating AI-2 while AI-1 is active clears AI-1,
    sets AI-2, both cache entries refreshed (assert via a subsequent
    `AIGet` that doesn't hit a stale cache).
  - Activating an already-active AI is a no-op success (idempotent), not
    an error.
  - Activating a soft-deleted AI returns an error (target row doesn't
    exist per normal `AIGet` semantics — no special-casing needed).
  - `type=normal` creation/activation attempt unaffected /
    rejected-with-400 respectively.
  - Delete of an active Insight AI clears `is_insight_active` (§3).
  - Type-change of an active Insight AI to `normal` clears
    `is_insight_active` (§3).
  - **Concurrency:** the `FOR UPDATE` race path is skip-gated on SQLite,
    same as the existing precedent (`aipromptproposal_test.go`'s
    `skipReasonNoForUpdate`) — do not attempt to simulate the race on
    SQLite; note in the test file that MySQL-backed race coverage is a
    follow-up if this becomes a real incident, consistent with how
    `AIAcceptProposal`'s own race path is currently handled in this repo.
- `aihandler`: `ActivateInsight` unit test — 400 on `type=normal` target;
  success path with mocked dbhandler.
- `bin-api-manager`: `ServiceAgentAIcallCreate` resolution test — active
  AI wins over more-recently-created-but-inactive one; zero-active falls
  back to most-recent; active AI outside the top-100-by-recency page is
  still found via the dedicated `is_insight_active=true` query (§4).
- `is_insight_active` present in `WebhookMessage` output and the RST
  struct doc for `/ais` (root CLAUDE.md: RST struct docs track
  `WebhookMessage`, not the internal model).
- **Filter-hop regression test:** an integration-level test (not just a
  `servicehandler`-layer unit test) asserting a `GET /ais` call with
  `is_insight_active=true` actually reaches `bin-ai-manager`'s dbhandler
  query filtered on that field — i.e. that the filter survives the
  `pkg/listenhandler/v1_ais.go` → `utilhandler.ConvertFilters` allowlist
  hop, not just the `bin-api-manager` conversion layer. This is the
  specific gap round-3 review caught (§2) and a plain unit test against
  `servicehandler` alone would not have caught it.
- `AIActivateInsight` targeting an AI belonging to a different customer
  than the caller's session is rejected — authz lives in `bin-api-manager`
  today, but add one `dbhandler`-level guard test since the `FOR UPDATE`
  scope is derived from the target row's own `customer_id`, so a
  regression in the api-manager check would otherwise let a cross-customer
  activation silently clear the wrong customer's active AI.
- square-admin: manual/E2E — create 2 Insight AIs (both succeed, no 409),
  activate the second, verify Case panel switches to it, verify AICall
  Test succeeds against the still-inactive first one.

## Open items

- Deployment sequencing (when to actually run the chained migration
  relative to SQUARE-23's own still-pending rollout — same maintenance
  window or separately) is an ops decision for 대표님/ops, not fixed by
  this design.

## Known minor limitation (accepted, not fixed)

`AIActivateInsight`'s `FOR UPDATE` locks the customer's *currently-active*
row (§2), not the target row being activated. A concurrent "change target
row's `type` to `normal`" update racing an in-flight activation of that
same row could theoretically leave `is_insight_active=true` on a
`type=normal` row. This is benign: the generated column's `type='insight'`
guard means such a row never holds the unique-index slot, and §4's
resolution query filters `type=insight`, so it's never selected. Left as
author's discretion (round-2 review's M3) rather than widening the lock,
since the extra lock scope isn't needed for correctness, only for a
field-level invariant with no observable effect.

## Review history

- r1 → round-1 architect review: REQUEST CHANGES (C1–C6, H1–H4, M1–M4).
  r2 addressed: wrong transaction-helper citation (C1, fixed via
  `AIAcceptProposal` pattern), missing cache invalidation (C2, fixed via
  mandatory dual-row cache refresh), unexecutable SQLite race test (C3,
  fixed via skip-gating like precedent), missing SQLite stand-in scope
  (C4, fixed), migration-chaining cost (C6, initially "fixed" by amending
  SQUARE-23's migration in place — reverted in r3, see below),
  racy/underspecified auto-activate (H1, fixed — dropped), silent
  deactivation on delete/type-change (H3, fixed), stale error message
  (H4, fixed), missing model/webhook/plumbing (M1/M2, fixed in §2),
  unenforceable NULL tri-state (M3, fixed), missing test cases (M4,
  fixed).
- r2 → round-2 architect review: REQUEST CHANGES (B1–B3, M1–M5). This
  revision (r3) addresses: unsound "amend in place" migration approach
  (B1 — Alembic revision-ID stamping means any DB that already ran
  SQUARE-23's migration would silently skip the amended version; reverted
  to a normal chained migration, accepting the one-time double-rewrite
  cost as the correct-by-construction choice), Case-panel resolution
  silently missing the active AI beyond the 100-row page (B2 — fixed with
  a dedicated `is_insight_active=true` size-1 query before the
  most-recent-created fallback), racy conditional read-modify-write on
  delete/type-change (B3 — fixed by making both `is_insight_active=false`
  clears unconditional, no read needed), zero-active UX regression (M1,
  fixed in §5), asymmetric dead-error-path cleanup (M2, fixed — both
  `Create` and `Update` translations removed), missing test cases (M4,
  fixed in §6). M3 (widen `FOR UPDATE` scope) and M5 (no over-engineering
  found) left as author's discretion per reviewer's own framing.
- r3 → round-3 architect review: REQUEST CHANGES (one blocking: B2's
  fix was directionally right but the `is_insight_active` filter would be
  silently dropped by `bin-ai-manager`'s `pkg/listenhandler/v1_ais.go`
  allowlist conversion, `utilhandler.ConvertFilters[ai.FieldStruct,
  ai.Field]`, degrading the query to "most recent" with no error —
  distinct from and in addition to the api-manager-side conversion this
  design already covered). This revision (r4) adds the missing
  `models/ai/filters.go` entry (§2), corrects the incorrect "available
  for free" claim, adds a filter-hop regression test (§6), removes
  `omitempty` from the webhook field (N5), pins the migration to "current
  head at implementation time" rather than a hardcoded revision (N4),
  fixes the zero-active UX detection to use the dedicated query instead
  of page-local inspection (N2), and documents the benign FOR-UPDATE
  scope limitation explicitly (N3, "Known minor limitation" section).
  B1 and B3 were independently re-verified correct with no changes
  needed.
