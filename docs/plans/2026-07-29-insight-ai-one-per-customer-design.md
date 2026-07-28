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

The same translation is added at the two duplicate-key-reachable exit
points in `aihandler.Update` (the `promptChanged` and default/"prompt
unchanged" branches both call into `h.db.AIUpdate`; `promptCleared` cannot
change `type`, since clearing the prompt is orthogonal to type, but the
helper call is added generically so it also covers that branch defensively).

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

- `dbhandler`: table-driven test creating two `type=insight` AIs for the
  same `customer_id` — second `AICreate` must return an error satisfying
  `IsErrDuplicate`. A third AI after soft-deleting the first two must
  succeed. A second `type=normal` AI for the same customer must always
  succeed (regression guard for the non-goal).
- `aihandler`: `Create` and `Update` unit tests asserting the duplicate-key
  path returns a `cerrors.VoipbinError` with `StatusAlreadyExists` and
  reason `AI_INSIGHT_ALREADY_EXISTS`, using a mock `dbhandler` that returns
  an error matching `IsErrDuplicate`.

## Open items for the ticket's AC

Ticket AC line 2 ("검토") is satisfied by §3 above — audit query defined,
resolution approach proposed (human-approved before execution, no schema
change until clear). Actual execution of the audit and any cleanup is
tracked as a follow-up step before the migration merges, not part of this
design.
