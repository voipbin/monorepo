# Insight AI: enforce one per customer (SQUARE-23)

**Date:** 2026-07-29
**Ticket:** [SQUARE-23](https://voipbin.atlassian.net/browse/SQUARE-23)
**Service:** `bin-ai-manager`

## Problem

`type=insight` AI records are currently unbounded per customer. square-admin's
Case detail Insight Assistant panel assumes "one Insight AI per customer" and
auto-selects a single AI to drive the panel; today it falls back to "most
recently created" when it finds more than one. That fallback masks operator
mistakes (accidentally creating a second Insight AI) instead of preventing
them.

This change adds a database-level constraint so a customer can never have
more than one non-deleted `type=insight` AI, and returns a clear API error
when a create or update would violate it.

## Non-goals

- Enforcing any cap on `type=normal` AIs (unaffected by this change).
- Removing square-admin's existing "most recent wins" fallback — it remains
  as a defensive read-path safeguard even after this constraint lands.
- Automating cleanup of pre-existing duplicate customers as part of the
  schema migration (see "Existing duplicates" below — handled out of band,
  before the migration runs).

## Design

### 1. Schema: generated column + unique index

MySQL has no partial unique index. Follow the existing precedent in this
same table family — `ai_aicalls.active_reference_key`
(`a5a40c93d3e6_ai_aicalls_add_active_reference_key_.py`, VOIP-1234) — which
solves an equivalent "at most one active X per customer" constraint with a
`STORED` generated column plus a `UNIQUE INDEX`.

```sql
ALTER TABLE ai_ais
ADD COLUMN active_insight_key BINARY(16) GENERATED ALWAYS AS (
  IF(type = 'insight' AND tm_delete IS NULL, customer_id, NULL)
) STORED;

CREATE UNIQUE INDEX uq_ai_active_insight_key ON ai_ais(active_insight_key);
```

Behavior:
- `type != 'insight'` → column computes `NULL` → any number of rows may
  share the same `NULL` (MySQL treats `NULL` as distinct in a `UNIQUE`
  index), so `normal` AIs are entirely unaffected.
- `type = 'insight'` and `tm_delete IS NULL` (not soft-deleted) → column
  computes `customer_id` → at most one such row per customer.
- `type = 'insight'` and soft-deleted (`tm_delete IS NOT NULL`) → `NULL` →
  a customer may freely create a new Insight AI after deleting the old one.

Migration file: created via `alembic revision` in `bin-dbscheme-manager`
(never hand-authored — see that service's `CLAUDE.md`), following the
`_column_exists` / `_index_exists` idempotency-guard pattern already used in
`a5a40c93d3e6_...` so a partially-applied migration can be safely re-run.

**Operational cost (inherited from the precedent, not new):** the cited
`a5a40c93d3e6_ai_aicalls_add_active_reference_key_.py` migration's own
history documents that `ADD COLUMN ... STORED` on a populated table forces
MySQL to rewrite the entire table in place, holding a metadata/table lock
for the duration; that migration is the exact incident this design mirrors,
including partial-failure retries against a stale generated-column
definition (which is why the `_column_exists`/`_index_exists` idempotency
guards exist at all). This design inherits the same risk class on `ai_ais`.
Before the migration is applied to any non-local environment:
- Confirm `ai_ais`'s current row count and expected rewrite duration are
  acceptable for an in-place `ALTER`; if the table is large enough that the
  lock duration would be customer-visible, use the same mitigation the
  precedent's follow-up used (a maintenance window) or an online schema
  tool (`pt-online-schema-change` / `gh-ost`) instead of a bare `ALTER
  TABLE`.
- This decision is an ops call at apply time, not something this design
  file fixes in advance; it is called out here so it isn't missed the way
  it was on the precedent.

**Downgrade:** mirrors the precedent 1:1 — `DROP INDEX
uq_ai_active_insight_key ON ai_ais` followed by `ALTER TABLE ai_ais DROP
COLUMN active_insight_key`, guarded by the same `_index_exists` /
`_column_exists` checks used on the way up so the downgrade is also safe to
re-run against a partially-applied state.

### 2. API-level error handling

Both `aihandler.Create` and `aihandler.Update` (`bin-ai-manager/pkg/aihandler`)
must translate a duplicate-key failure on `active_insight_key` into a clear
error, not a raw SQL error. `Update` is in scope too: a `normal → insight`
type change on an existing AI hits the same constraint and needs the same
translation.

Reuse the existing `dbhandler.IsErrDuplicate(err)` helper
(`bin-ai-manager/pkg/dbhandler/main.go`), already used for the same purpose
in `aicallhandler/start.go` (`startReferenceTypeContactCase`):

```go
res, err := h.dbCreate(ctx, ...)
if err != nil {
    if dbhandler.IsErrDuplicate(err) {
        return nil, cerrors.AlreadyExists(
            commonoutline.ServiceNameAIManager,
            "AI_INSIGHT_ALREADY_EXISTS",
            "This customer already has an Insight AI. Delete the existing one before creating another.",
        ).Wrap(err)
    }
    return nil, errors.Wrapf(err, "could not create ai")
}
```

`cerrors.StatusAlreadyExists` maps to HTTP 409 via the existing
`HTTPStatusFor` table (`bin-common-handler/models/errors/rpc.go`) — no new
status code or mapping needed.

**Limitation (inherited, not new):** `dbhandler.IsErrDuplicate`
(`bin-ai-manager/pkg/dbhandler/main.go`) does a blunt substring match on
`"Duplicate entry"` with no index-name scoping. Once `uq_ai_active_insight_key`
exists on `ai_ais`, any duplicate-key error on that table — e.g. an
extremely unlikely `id`/PK collision — would also get mapped to
`AI_INSIGHT_ALREADY_EXISTS`, which would be a misleading message for that
case. This weakness pre-dates this design (it already applies to the sole
prior caller, `aicallhandler/start.go`'s internal race-detection check),
but it now backs a customer-facing error message rather than an internal
check, so the risk surface is different. Not fixed as part of this change;
noted here so a future index-scoped refinement (e.g. matching the index
name in the error text) can be tracked if it becomes a real issue.

`Update` resolves `aiType` once, before the prompt-state branching
(`pkg/aihandler/chatbot.go`, the `aiType == ai.TypeNone` fallback block),
and passes that same resolved `aiType` into `buildUpdateFields` in all
three branches of the `switch`. So a caller can clear the prompt **and**
change `type` from `normal` to `insight` in the same request — the
`promptCleared` branch's `h.db.AIUpdate` call is exactly as
duplicate-key-reachable as the other two, not "orthogonal to type" as the
branch name might suggest. All three branches need the translation:

1. `promptChanged` — inline `h.db.AIUpdate` call in `chatbot.go`.
2. `promptCleared` — inline `h.db.AIUpdate` call in `chatbot.go` (the
   type-change-while-clearing-prompt case above).
3. `default` ("prompt unchanged") — `chatbot.go` does not call
   `h.db.AIUpdate` directly here; it returns `h.dbUpdate(ctx, ...)`, and
   `dbUpdate` (`pkg/aihandler/db.go`) is the one that calls `h.db.AIUpdate`.
   `dbUpdate` does not return that error unwrapped, though: it wraps it via
   `errors.Wrapf(err, "could not update ai")` (`pkg/errors`, i.e. github.com
   /pkg/errors) before returning. That wrapping does not defeat
   `IsErrDuplicate` — it does a substring match on `err.Error()`, and
   `pkg/errors` wrapping preserves the original message text in `Error()`
   — but it does mean the sentinel/type is not preserved as-is, only the
   message text is. The `IsErrDuplicate` translation for this branch must
   therefore live inside `dbUpdate` in `db.go`, not in `chatbot.go`. An
   implementation that only edits `chatbot.go` will leave this branch's
   duplicate-key errors as raw wrapped SQL errors, missing the AC's
   rejection guarantee for the most common update path (type unchanged,
   prompt unchanged).

Concretely: add the `IsErrDuplicate` check around the `h.db.AIUpdate` call
in each of `chatbot.go`'s `promptChanged` and `promptCleared` branches, and
around the `h.db.AIUpdate` call inside `dbUpdate` in `db.go`.

### 3. Existing duplicates (AC: migration/cleanup review)

`CREATE UNIQUE INDEX` fails with a 1062 if any customer already has 2+
non-deleted Insight AIs at migration time — the exact failure mode already
hit once in production by the `ai_aicalls` migration this design mirrors.
Handled as a pre-migration, out-of-band step (not part of the migration
file, and not raw SQL against production — `bin-dbscheme-manager/CLAUDE.md`
prohibits AI-run schema/data changes against non-local databases):

1. **Audit (read-only):** run a `SELECT customer_id, COUNT(*) FROM ai_ais
   WHERE type = 'insight' AND tm_delete IS NULL GROUP BY customer_id HAVING
   COUNT(*) > 1` against a read replica or via an ops-approved read path,
   and report the affected customer list.
2. **Resolution (product decision, human-approved):** default proposal is
   "keep the most recently created Insight AI per customer" — matching
   square-admin's existing fallback exactly, so no customer's active
   assistant changes. Extras are removed via the existing `DELETE /ais/{id}`
   endpoint (soft delete, goes through `aiHandler.Delete`, cache
   invalidation included) — not a raw SQL `UPDATE`/`DELETE`.
3. Only after step 2 confirms zero customers with 2+ active Insight AIs does
   the migration's `CREATE UNIQUE INDEX` get applied.

### 4. Testing

**Prerequisite — SQLite test-schema stand-in.** `bin-ai-manager`'s
`dbhandler` tests (`pkg/dbhandler/main_test.go`) run against an in-memory
SQLite database (`go-sqlite3`), loading table DDL from
`scripts/database_scripts_test/*.sql` — they do **not** exercise the
Alembic/MySQL migration at all. The precedent this design cites needed its
own SQLite-compatible stand-in for exactly this reason:
`scripts/database_scripts_test/table_ai_aicalls.sql` defines
`active_reference_key` using `CASE WHEN` + string concatenation instead of
MySQL's `IF(...)`, with an inline comment explaining it is a SQLite
substitute for the real `STORED` generated column. MySQL's `IF(...)` is not
valid SQLite syntax, so a literal copy of this design's migration DDL into
a test script would not apply.

Before the dbhandler tests below can pass, `scripts/database_scripts_test/table_ai_ais.sql`
must be updated with an equivalent SQLite stand-in, e.g.:

```sql
active_insight_key varchar(255) GENERATED ALWAYS AS (
  CASE WHEN type = 'insight' AND tm_delete IS NULL
    THEN customer_id
    ELSE NULL
  END
) STORED,
```

```sql
create unique index uq_ai_active_insight_key on ai_ais(active_insight_key);
```

added alongside the existing `create table ai_ais(...)` block, mirroring
`table_ai_aicalls.sql`'s pattern (`customer_id` is already `binary(16)`
here, so — unlike the precedent's hash-based key — no `X'...'` hex-literal
handling is needed). This file update is in scope for this change, not a
follow-up; without it the test plan below is unexecutable as described.

- `dbhandler`: table-driven test creating two `type=insight` AIs for the
  same `customer_id` — the first `AICreate` succeeds, the second must
  return an error satisfying `IsErrDuplicate` (no second row is ever
  persisted, since the unique index rejects it at insert time). A second
  `AICreate` attempt after soft-deleting the first must then succeed,
  proving the generated column frees the slot once the original row's
  `tm_delete` is set. A second `type=normal` AI for the same customer must
  always succeed (regression guard for the non-goal).
- `aihandler`: `Create` and `Update` unit tests asserting the duplicate-key
  path returns a `cerrors.VoipbinError` with `StatusAlreadyExists` and
  reason `AI_INSIGHT_ALREADY_EXISTS`, using a mock `dbhandler` that returns
  an error matching `IsErrDuplicate`. `Update` needs one case per branch
  that reaches `h.db.AIUpdate`, since each has a separate code path for
  the translation (see §2):
  - `promptChanged` with a `type` change to `insight` (duplicate exists).
  - `promptCleared` with a `type` change to `insight` (duplicate exists)
    — regression test for the case where clearing the prompt and changing
    `type` happen in the same request; this is the scenario the original
    "`promptCleared` cannot change `type`" description would have caused
    an implementer to skip.
  - `default` ("prompt unchanged") with a `type` change to `insight`
    (duplicate exists) — exercises the translation inside `dbUpdate`
    (`db.go`), not `chatbot.go`.

## Open items for the ticket's AC

Ticket AC line 2 ("검토") is satisfied by §3 above — audit query defined,
resolution approach proposed (human-approved before execution, no schema
change until clear). Actual execution of the audit and any cleanup is
tracked as a follow-up step before the migration merges, not part of this
design.
