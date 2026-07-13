# VOIP-1254: Replace contact_case_tag_assignments junction table with Case.TagIDs JSON column

Status: Draft (iter 0)

## Problem statement

`contact_case_tag_assignments` (migration `5c7bc362be27`, landed 2026-07-07 as part
of the Case-Contact design's round-22 correction) is a junction table linking
`contact_cases.id` to `bin-tag-manager` tag IDs, with dedicated
dbhandler CRUD (`CaseTagAssignmentCreate/Delete/ListByCaseID`) and case-handler
methods (`CaseTagAdd/Remove/List`).

On review, pchero flagged this as likely over-engineering, and investigation
confirmed it: `bin-queue-manager` already stores `Queue.TagIDs` as a plain JSON
column (`db:"tag_ids,json"`) for a **heavier** use case (real-time agent-tag
intersection matching at call-routing time), with **no junction table and no
reverse-lookup index**. Case tag usage (a handful of tags, low-frequency
agent-driven writes, no matching/routing requirement) is strictly lighter than
Queue's.

The junction-table rationale from the original design review (referential
integrity, write concurrency) does not actually hold:
- `bin-contact-manager` does not subscribe to `bin-tag-manager`'s `tag_deleted`
  event today, so referential integrity across the FK-like relationship is not
  enforced either way, junction table or JSON column.
- Case tag writes are low-frequency UI actions (an agent tagging a case), not a
  routing hot path — the concurrency argument that justified a
  unique-index-backed junction table for other resources does not apply here.

## Goals

1. Add `Case.TagIDs []uuid.UUID` with `db:"tag_ids,json"` to
   `bin-contact-manager/models/kase/kase.go`, mirroring
   `bin-queue-manager/models/queue/queue.go`'s `TagIDs` field exactly (same Go
   type, same db-tag conversion type).
2. Add `contact_cases.tag_ids` as a JSON column via Alembic migration, and drop
   `contact_case_tag_assignments` (created in `5c7bc362be27`) via a follow-up
   migration in the same PR.
3. Remove `pkg/dbhandler/case_tag_assignment.go`
   (`CaseTagAssignmentCreate/Delete/ListByCaseID`) and its test file; replace
   with a direct `Case` column update path.
4. Update `pkg/casehandler/case_tag.go` (`CaseTagAdd/Remove/List`) to operate on
   the new column instead of the junction table, **keeping the existing public
   method signatures unchanged** (`CaseTagAdd(ctx, customerID, caseID, tagID)`,
   `CaseTagRemove(ctx, customerID, caseID, tagID)`,
   `CaseTagList(ctx, customerID, caseID) ([]uuid.UUID, error)`) so
   `pkg/listenhandler/v1_cases.go` (the REST-adjacent RPC layer, VOIP-1242's
   surface) requires zero changes.

## Non-goals

- No reverse lookup (all cases with tag X). Matches Queue's precedent — Queue
  has no such index either, and nothing in the current design needs it.
- No change to `bin-tag-manager` itself. Cases continue to reference the same
  `Tag` rows (by `tag_id`) that Contacts and Queues already do; the tag
  existence check (`TagV1TagGet`) in `CaseTagAdd` is preserved unchanged.
- No change to `contact_tag_assignments` (the separate Contact-to-tag junction
  table) — that table is unrelated and out of scope.
- This ticket does NOT build VOIP-1242's REST/OpenAPI layer. VOIP-1242 is
  updated (via ticket comment, already done 2026-07-14) to build on top of this
  simpler storage once this ticket lands; the existing internal RPC methods
  (`processV1CasesIDTagsGet/Post`, `.../IDDelete` in `v1_cases.go`) are already
  wired and unaffected by this change (see Goal 4).

## Affected files

| File | Change |
|---|---|
| `bin-contact-manager/models/kase/kase.go` | Add `TagIDs []uuid.UUID` field with `db:"tag_ids,json"` |
| `bin-contact-manager/models/kase/kase_test.go` | Extend `Test_Case_ConstructAndMarshal` to cover `TagIDs`; add a nil/empty-slice case to `Test_Case_NilOptionalFields` |
| `bin-contact-manager/pkg/dbhandler/kase.go` | Add `CaseUpdateTagIDs` (customer-scoped direct column update, mirrors `caseUpdateContactIDExec`'s shape) |
| `bin-contact-manager/pkg/dbhandler/main.go` | Add `CaseUpdateTagIDs` to the `DBHandler` interface; remove the 3 `CaseTagAssignment*` interface entries |
| `bin-contact-manager/pkg/dbhandler/case_tag_assignment.go` | **Delete** |
| `bin-contact-manager/pkg/dbhandler/case_tag_assignment_test.go` | **Delete** |
| `bin-contact-manager/pkg/dbhandler/mock_main.go` | Regenerate (`go generate ./pkg/dbhandler/...`) |
| `bin-contact-manager/pkg/casehandler/case_tag.go` | Rewrite `CaseTagAdd/Remove/List` to read-modify-write `Case.TagIDs` via `CaseGetByID` + `CaseUpdateTagIDs`; rename `verifyCaseOwnership` → `verifyCaseOwnershipAndGet` (returns `*kase.Case`); add new `containsUUID` helper; signatures unchanged |
| `bin-contact-manager/pkg/casehandler/case_tag_test.go` | Rewrite fixtures/mocks for the new dbhandler call shape |
| `bin-contact-manager/pkg/casehandler/contact_update.go` | Update sole `verifyCaseOwnership` call site for the renamed/re-signatured function (discard returned Case) |
| `bin-contact-manager/pkg/casehandler/casenote.go` | Update sole `verifyCaseOwnership` call site for the renamed/re-signatured function (discard returned Case) |
| `bin-contact-manager/pkg/casehandler/case_list_get.go` | Update `CaseGet`'s `verifyCaseOwnership` call site for the rename; use the returned Case to drop the redundant second `CaseGetByID` call (`return c, nil`) |
| `bin-dbscheme-manager/bin-manager/main/versions/<new>_contact_cases_add_column_tag_ids.py` | `ALTER TABLE contact_cases ADD COLUMN tag_ids JSON DEFAULT NULL` |
| `bin-dbscheme-manager/bin-manager/main/versions/<new>_contact_case_tag_assignments_drop_table.py` | `DROP TABLE IF EXISTS contact_case_tag_assignments` (chained after the ADD COLUMN migration) |
| `bin-dbscheme-manager/docs/schema-ownership.md` | Remove the (currently missing — see Pitfall below) `contact_case_tag_assignments` row if present; no new row needed since `tag_ids` is a column on the already-listed `contact_cases`... **actually `contact_cases` itself is not yet listed either — see Open Questions.** |
| `bin-contact-manager/scripts/database_scripts_test/*.sql` (if a case-table test fixture exists) | Add `tag_ids json` column to keep SQLite/local test schema in sync (grep during implementation) |

## Design decision: dedicated `CaseUpdateTagIDs`, not a generic field-map updater

`bin-queue-manager`'s `QueueUpdate(ctx, id, fields map[queue.Field]any)` is a
generic multi-field updater that `UpdateTagIDs` calls with a single-key map.
`bin-contact-manager/pkg/dbhandler/kase.go` has no equivalent generic updater
today — every mutation (`CaseUpdateContactID`, `CaseClearContactID`,
`CaseUpdateStatusClosed`, `CaseUpdateTMUpdate`) is a small, dedicated,
customer-scoped function built directly with squirrel. Introducing a generic
`Field`/map-based updater purely to reuse Queue's exact call shape would be a
larger footprint change than this ticket needs, and would deviate from every
other mutation already in `kase.go`.

**Decision:** add `CaseUpdateTagIDs(ctx, customerID, id uuid.UUID, tagIDs
[]uuid.UUID) error`. The customer-scoped `UPDATE ... WHERE id=? AND
customer_id=?` structure mirrors `caseUpdateContactIDExec`'s shape exactly
(single dedicated function, no transaction wrapper needed since
`CaseTagAdd/Remove` do a plain read-then-write, not a multi-statement
derivation). **The column-value conversion does NOT mirror
`caseUpdateContactIDExec` literally** — `caseUpdateContactIDExec` manually
calls `.Bytes()` on a single `uuid.UUID` for a binary-UUID column, but
`tag_ids` is a JSON column holding `[]uuid.UUID`, which needs JSON
marshaling, not `.Bytes()`. A bare `sq.Update(caseTable).Set("tag_ids",
tagIDs)` would pass the raw `[]uuid.UUID` slice straight to the SQL driver,
which has no support for that Go type (not `[]byte`, `string`, numeric,
`time.Time`, or a `driver.Valuer`) and fails at `Exec()` time with an
"unsupported type" error — this would be caught immediately by
`CaseUpdateTagIDs`'s own round-trip test in §Verification plan, but it's a
concrete implementation blocker worth resolving in the design rather than at
implementation time.

**Fix: route the `tag_ids` value through `commondatabasehandler.PrepareFields`
on a small ad-hoc map, exactly matching `bin-queue-manager`'s own
`QueueUpdate`/`UpdateTagIDs` precedent** (`bin-queue-manager/pkg/dbhandler/
queue.go`'s `QueueUpdate` calls `commondatabasehandler.PrepareFields(fields)`
where `fields` is a `map[queue.Field]any{queue.FieldTagIDs: tagIDs}` — per
`mapping.go`'s `prepareFieldsFromMap`, a map input auto-detects the
`[]uuid.UUID` slice and JSON-marshals it before it reaches squirrel, with no
`db:` tag needed since map inputs skip tag-based filtering entirely):

```go
func caseUpdateTagIDsExec(exec sqlExecutor, customerID, id uuid.UUID, tagIDs []uuid.UUID) error {
	fields, err := commondatabasehandler.PrepareFields(map[string]any{
		"tag_ids": tagIDs,
	})
	if err != nil {
		return fmt.Errorf("could not prepare fields. CaseUpdateTagIDs. err: %v", err)
	}

	query, args, err := sq.Update(caseTable).
		SetMap(fields).
		Where(sq.Eq{"id": id.Bytes()}).
		Where(sq.Eq{"customer_id": customerID.Bytes()}).
		ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. CaseUpdateTagIDs. err: %v", err)
	}

	if _, err := exec.Exec(query, args...); err != nil {
		return fmt.Errorf("could not execute. CaseUpdateTagIDs. err: %v", err)
	}

	return nil
}

func (h *handler) CaseUpdateTagIDs(ctx context.Context, customerID, id uuid.UUID, tagIDs []uuid.UUID) error {
	return caseUpdateTagIDsExec(h.db, customerID, id, tagIDs)
}
```

This keeps `kase.go` internally consistent with its own existing
customer-scoped-mutation shape (matches `caseUpdateContactIDExec`'s
structure) while sourcing the actual JSON conversion mechanics from the
already-proven `PrepareFields`-on-a-map path Queue itself uses for the exact
same `[]uuid.UUID`-into-JSON-column write — not inventing a third, untested
conversion approach (e.g. a bare `json.Marshal` call) that neither existing
precedent in the codebase actually uses.

## CaseTagAdd/Remove/List new implementation shape

```go
func (h *caseHandler) CaseTagAdd(ctx context.Context, customerID, caseID, tagID uuid.UUID) error {
	c, err := verifyCaseOwnershipAndGet(ctx, h.db, customerID, caseID) // returns *kase.Case, not just error
	if err != nil {
		return err
	}
	if _, err := h.reqHandler.TagV1TagGet(ctx, tagID); err != nil {
		return ErrTagNotFound
	}
	if containsUUID(c.TagIDs, tagID) {
		return nil // idempotent no-op, already tagged
	}
	newTagIDs := append(append([]uuid.UUID{}, c.TagIDs...), tagID)
	if err := h.db.CaseUpdateTagIDs(ctx, customerID, caseID, newTagIDs); err != nil {
		return err
	}
	h.notifyHandler.PublishEvent(ctx, "case_tag_added", map[string]uuid.UUID{
		"case_id": caseID,
		"tag_id":  tagID,
	})
	return nil
}
```

**Event publishing: `CaseTagAdd`/`CaseTagRemove` now publish `case_tag_added` /
`case_tag_removed`** via the plain `notifyHandler.PublishEvent()` primitive
(never `PublishWebhookEvent()`), matching the established convention on every
other Case-scoped mutation in this package (`contact_update.go`'s
`case_contact_attributed/detached`, `casenote.go`'s
`case_note_created/deleted`). This is a deliberate, in-scope addition, not a
carry-forward of the current (pre-this-design) `case_tag.go`'s actual
behavior — the current junction-table-based `CaseTagAdd/Remove` publish
**no** event today, which is itself an inconsistency with the rest of the
package. Since this design rewrites `CaseTagAdd/Remove` end-to-end, adding the
event call closes that gap rather than carrying it forward silently.
`CaseTagList` (read-only) publishes nothing, consistent with every other
read-path method in this package.

`CaseTagRemove` is the symmetric filter-out-and-write, **explicitly
idempotent on the no-op path, mirroring `CaseTagAdd`'s already-tagged
no-op exactly:**

```go
func (h *caseHandler) CaseTagRemove(ctx context.Context, customerID, caseID, tagID uuid.UUID) error {
	c, err := verifyCaseOwnershipAndGet(ctx, h.db, customerID, caseID)
	if err != nil {
		return err
	}
	if !containsUUID(c.TagIDs, tagID) {
		return nil // idempotent no-op, tag was never present -- no write, no event
	}
	newTagIDs := make([]uuid.UUID, 0, len(c.TagIDs)-1)
	for _, id := range c.TagIDs {
		if id != tagID {
			newTagIDs = append(newTagIDs, id)
		}
	}
	if err := h.db.CaseUpdateTagIDs(ctx, customerID, caseID, newTagIDs); err != nil {
		return err
	}
	h.notifyHandler.PublishEvent(ctx, "case_tag_removed", map[string]uuid.UUID{
		"case_id": caseID,
		"tag_id":  tagID,
	})
	return nil
}
```

Removing an absent `tag_id` is a no-op, not an error — symmetric with `Add`'s
already-tagged no-op. The no-op path skips **both** the `CaseUpdateTagIDs`
write (nothing to persist — writing an identical slice back would be a
wasted round trip) **and** the `case_tag_removed` event (firing a "removed"
event for a tag that was never present would be a semantically false audit
record). This mirrors `CaseTagAdd`'s no-op branch skipping both its write
and its event for the same reason.

`CaseTagList` becomes simply
`verifyCaseOwnershipAndGet(ctx, h.db, customerID, caseID)` followed by
`return c.TagIDs, nil` — no separate `CaseGetByID` dbhandler call at all;
`TagIDs` is already part of the `Case` row returned by the ownership check.

**`verifyCaseOwnership` must be widened to return the fetched `*kase.Case`**
(not just an error) so `CaseTagAdd/Remove` can read `TagIDs` off the same row
they already fetched for the ownership check, instead of a second
`CaseGetByID` round trip. Rename to `verifyCaseOwnershipAndGet` and update
**all 6 existing call sites** (verified by grep, not assumed) to keep the
package compiling:
- `case_tag.go`: `CaseTagAdd`, `CaseTagRemove`, `CaseTagList` (3 sites — these
  are the ones that actually use the returned Case, per the shape above).
- `contact_update.go`: 1 site (discard the returned Case; only the ownership
  check is needed there today).
- `casenote.go`: 1 site (discard the returned Case).
- `case_list_get.go`'s `CaseGet` (line 59): 1 site — **this one was missed in
  an earlier draft of this design and is a compile-breaking omission once the
  function's return signature changes.** `CaseGet` currently does
  `verifyCaseOwnership` followed by a second, redundant `h.db.CaseGetByID`
  call (lines 59-62) to fetch the same row twice; take this opportunity to use
  the newly-returned Case directly and drop the second DB round trip
  (`return c, nil` instead of `return h.db.CaseGetByID(ctx, id)`) as a small,
  in-scope efficiency win alongside the rename.

`containsUUID(slice []uuid.UUID, target uuid.UUID) bool` in the `CaseTagAdd`
sketch above is a **new helper introduced by this design**, not an existing
utility — add it (in `case_tag.go` or a small shared helpers file) rather
than searching for a pre-existing one.

## Concurrency note (explicitly scoped as acceptable, not a regression)

The read-modify-write shape above (`CaseGetByID` → mutate slice in Go →
`CaseUpdateTagIDs`) is **not** atomic against a concurrent second
`CaseTagAdd`/`CaseTagRemove` on the same case — a classic lost-update window.
The junction table this replaces did not have this problem (each tag was its
own row, insert/delete were independent). This is a real, if narrow,
regression in a race window.

**Accepted per the ticket's own framing:** case tag writes are low-frequency,
single-agent UI actions (an agent tagging the case they are actively working),
not a routing hot path with concurrent writers — the same framing that
justified dropping the junction table's concurrency guarantees in the first
place. Two agents simultaneously tagging the same open case is an edge case
already implicitly accepted by removing the per-row uniqueness the junction
table provided. If this becomes a real problem in practice (observed
lost-update reports), the fix is a `SELECT ... FOR UPDATE` scoped
read-modify-write (mirrors the existing `CaseGetByIDForUpdate` /
`BeginTx` pattern already in `kase.go` for the get-or-create path), not a
revert to the junction table. Flagging explicitly rather than silently
building it in, since it's the one place this design consciously trades
consistency for simplicity in a way Queue's design doesn't need to (Queue's
`UpdateTagIDs` fully replaces the whole list from the admin API in one PUT,
with only one caller and no add/remove semantics — Case has separate
add/remove semantics on top of the same storage, which is precisely the shape
that introduces the lost-update window Queue never has).

## Migration DDL

```sql
-- <rev1>_contact_cases_add_column_tag_ids.py (down_revision = current head)
ALTER TABLE contact_cases ADD COLUMN tag_ids JSON DEFAULT NULL AFTER previous_case_id;
-- downgrade(): ALTER TABLE contact_cases DROP COLUMN tag_ids;

-- <rev2>_contact_case_tag_assignments_drop_table.py (down_revision = <rev1>)
DROP TABLE IF EXISTS contact_case_tag_assignments;
-- downgrade(): recreates contact_case_tag_assignments verbatim (copy from 5c7bc362be27)
```

Two separate migrations (not one) so `downgrade()` can cleanly recreate
`contact_case_tag_assignments` (verbatim copy from `5c7bc362be27`) without
also having to un-populate `tag_ids` data that may have been written between
the two migrations landing — matches this skill's existing "which function
contains the DROP" convention (the DROP lives in `upgrade()`, `downgrade()`
recreates the table).

**Asymmetric downgrade data-loss risk (rev1 only, explicitly accepted):**
rev2's `downgrade()` only recreates the junction table's empty *structure* —
it does not, and cannot, repopulate it with any `tag_ids` data written after
this PR ships, because the junction table concept (one row per case-tag pair)
was already retired by rev2's own `upgrade()`. rev1's `downgrade()`
(`DROP COLUMN tag_ids`) is a hard, unrecoverable data loss of every tag
assignment written via the new column — there is no fallback path back to
the junction table's format once you've dropped the column, since a
downgrade by construction runs rev2's downgrade (recreate empty junction
table) before rev1's downgrade (drop the JSON column) can even fire, in
strict LIFO migration order. This is called out explicitly in Rollout / risk
below rather than left implicit; it is treated the same way this
skill's precedent treats any schema downgrade on a live-data column — an
accepted operational risk of running `alembic downgrade`, not a design defect
to engineer around (this repo's standing rule: AI must never run
`alembic downgrade` against anything but a local throwaway DB in the first
place, per `voipbin-dbscheme-migration`'s CLAUDE.md prohibition; downgrade is
a human-operated, deliberate action with its own operational awareness of
data loss, same as any other production schema rollback).

Both DDL statements are already-proven patterns per
`voipbin-dbscheme-migration`: plain `JSON` column ADD round-trips cleanly
(verified precedent: VOIP-1215 `conversation_messages` source/destination,
and Queue's own `tag_ids`/`wait_queue_call_ids` JSON columns already live in
production), and a plain `DROP TABLE IF EXISTS` on a table with zero
production rows expected (Case tagging shipped 2026-07-07 and has not yet
been exposed via any client-facing surface — VOIP-1242's REST layer, the
actual write path for this table, was never built) is not a new pattern.
**Verification during implementation must confirm zero rows exist in
`contact_case_tag_assignments` in any real target before treating the drop as
side-effect-free** (Goal: read-only `SELECT COUNT(*) FROM
contact_case_tag_assignments` — expected 0, but empirically confirm rather
than assume, per `voipbin-dbscheme-migration`'s repeated lesson about
diagnosing real data before writing a DROP).

## Verification plan

1. **Docker round-trip** (fast path: reuse `voipbin-test-db-1` persistent
   MySQL 8.0 container per `voipbin-dbscheme-migration`'s Fast Path section,
   since this reuses an already-proven JSON-column-ADD pattern — not a new
   constraint requiring the full MariaDB→mysqldump→MySQL round-trip):
   - `alembic upgrade head` applies both new migrations cleanly.
   - `DESCRIBE contact_cases` shows `tag_ids` as `json`/`longtext` with a
     `json_valid` CHECK.
   - `SHOW TABLES LIKE 'contact_case_tag_assignments'` returns empty.
   - `alembic downgrade -2` recreates the junction table verbatim (diff
     against `5c7bc362be27`'s `CREATE TABLE`) and drops `tag_ids`; `upgrade
     head` again to leave the container clean.
2. **Go unit tests** (`go test ./...` in `bin-contact-manager`, full package,
   not name-filtered):
   - `kase_test.go`: TagIDs marshal/unmarshal, nil-vs-empty-slice.
   - `dbhandler`: `CaseUpdateTagIDs` round-trip (set, overwrite, clear to
     empty slice) against the SQLite test harness.
   - `casehandler/case_tag_test.go`: Add/Remove/List against the new
     read-modify-write shape, including the idempotent-add-when-already-tagged
     case, the symmetric idempotent-remove-when-not-tagged case (asserting
     no `CaseUpdateTagIDs` call and no `PublishEvent` call on that path,
     mirroring the add-side no-op assertion), and the tenant-isolation case
     (wrong customer_id → `ErrNotFound`, unchanged from today).
3. **Full verification workflow** per root CLAUDE.md: `go mod tidy && go mod
   vendor && go generate ./... && go test ./... && golangci-lint run -v
   --timeout 5m` in `bin-contact-manager`.
4. Confirm `pkg/listenhandler/v1_cases_test.go`'s existing
   `CaseTagList/Add/Remove` mock expectations (lines ~353-373) still compile
   and pass unmodified — they mock `caseHandler`'s interface, which per Goal 4
   is unchanged.

## Rollout / risk

- **Risk: data loss if `contact_case_tag_assignments` has real rows.** Mitigated
  by the empirical zero-row check above before finalizing the DROP migration;
  if non-zero, the migration must add a data-migration step (read junction
  rows, group by case_id, write to the new `tag_ids` JSON column) before the
  DROP — this is a live open question pending that check (see below).
- **Risk: `alembic downgrade` after this ships loses all `tag_ids` data
  written since (asymmetric downgrade risk).** Explicitly accepted, per the
  Migration DDL section above — rev1's `downgrade()` (`DROP COLUMN tag_ids`)
  is unrecoverable once run, since the junction-table fallback format no
  longer exists after rev2's `upgrade()`. Consistent with this repo's
  standing rule that AI never runs `alembic downgrade` against anything but a
  local throwaway DB; a human operator running downgrade against a real
  target already owns this class of risk for any schema rollback.
- **Risk: lost-update race on concurrent CaseTagAdd/Remove.** Explicitly
  accepted per the Concurrency note above; not a blocking risk for this
  ticket's scope.
- **Risk: breaking VOIP-1242's future REST layer expectations.** Mitigated by
  keeping `CaseTagAdd/Remove/List`'s public signatures unchanged (Goal 4);
  VOIP-1242 already carries a ticket comment pointing at this ticket's
  landing as a prerequisite.

## Open questions

1. **Does `contact_case_tag_assignments` have any real rows in staging/production
   today?** Must be checked (read-only `SELECT COUNT(*)`) before finalizing
   the DROP migration; if non-empty, a backfill step is required in `upgrade()`
   before the drop. Given this feature has never been exposed via any
   client-facing surface (VOIP-1242's REST layer was never built), zero rows
   is the expected outcome, but this must be empirically confirmed per
   `voipbin-dbscheme-migration`'s standing lesson, not assumed.
2. `bin-dbscheme-manager/docs/schema-ownership.md` does not currently list
   `contact_cases` at all (only `contact_addresses/contact_contacts/
   contact_interactions/contact_resolutions/contact_tag_assignments` are
   listed) — this is pre-existing doc drift from the 2026-07-07 Case launch,
   not something this ticket introduced. Should this ticket also add the
   missing `contact_cases` row while touching this file, or is that a
   separate doc-hygiene ticket? Recommendation: fix it in this PR since we are
   already touching schema-ownership.md context for this table family, one
   extra line, no scope creep risk.

## Iter-1 review response summary

Round 1 review (`delegate_task`, read-only) found one real gap and one minor
clarity item, both fixed:

- **Finding 1 (real, compile-breaking):** `verifyCaseOwnership` call-site
  enumeration was incomplete — missed `case_list_get.go`'s `CaseGet` (a 4th
  call site beyond `case_tag.go`/`contact_update.go`/`casenote.go`). Fixed:
  §"CaseTagAdd/Remove/List new implementation shape" now enumerates all 6
  call sites explicitly and notes `CaseGet`'s bonus redundant-fetch removal;
  the Affected files table now lists all 3 previously-implicit files
  (`contact_update.go`, `casenote.go`, `case_list_get.go`) individually.
- **Finding 2 (minor):** `containsUUID` was used in the code sketch without
  stating it's a new helper. Fixed: explicit callout added in the same
  section.

## Iter-2 review response summary

Round 2 review re-verified all round-1 fixes as correctly applied (call-site
enumeration, DBHandler interface diff consistency, silent-breakage check on
`contact_update.go`/`casenote.go`, migration column-placement claim vs.
current `contact_cases` schema history) and found one new, genuine issue:

- **Finding (real, wording bug):** the `CaseTagList` description paragraph
  still referenced the stale `verifyCaseOwnership` name and an extra,
  contradictory `CaseGetByID` step. Fixed: reworded to state plainly that
  `CaseTagList` is `verifyCaseOwnershipAndGet(...)` + `return c.TagIDs, nil`,
  no separate dbhandler call.

## Iter-3 review response summary

Round 3 review re-verified iter-1/iter-2 fixes hold (no stray old-name
references, correct internal consistency) and found two new, genuine gaps:

- **Finding 1 (real):** rev1's `downgrade()` data-loss risk (dropping
  `tag_ids` with no fallback) was undocumented. Fixed: explicit
  `downgrade()` DDL added to §Migration DDL, plus a new paragraph explaining
  why it's an accepted, unrecoverable risk (consistent with this repo's
  never-run-downgrade-against-real-targets rule), plus a new Rollout/risk
  bullet.
- **Finding 2 (real):** the design never stated whether `CaseTagAdd/Remove`
  should publish an audit event, despite `contact_update.go`/`casenote.go`
  establishing that convention for every other Case mutation in this
  package. Fixed: decided to add `case_tag_added`/`case_tag_removed`
  `PublishEvent` calls (closing a pre-existing inconsistency in the current
  junction-table implementation, which publishes nothing today), documented
  in the code sketch and a new explanatory paragraph.

## Iter-4 review response summary

Round 4 review verified all prior fixes remain consistent (event payload
shape matches `contact_update.go`/`casenote.go` exactly, no cross-section
contradictions after 3 rounds of edits) and found one new, genuine gap:

- **Finding (real):** `CaseTagRemove`'s no-op behavior (removing an absent
  tag_id) was never specified, unlike `CaseTagAdd`'s explicit no-op branch.
  Fixed: added a full `CaseTagRemove` code sketch, explicitly idempotent
  (no-op on absent tag, skipping both the DB write and the `case_tag_removed`
  event, symmetric with `CaseTagAdd`'s no-op skipping its write and event),
  plus a matching test-case addition in §Verification plan.

## Iter-5 review response summary

Round 6 review found one real, previously-unaddressed gap (write-path JSON
serialization mechanics for `CaseUpdateTagIDs`):

- **Finding (real, implementation-blocking):** the design's "mirrors
  `caseUpdateContactIDExec`'s shape" prescription omitted how `tag_ids`
  (`[]uuid.UUID` into a JSON column) actually gets serialized on write.
  `caseUpdateContactIDExec`'s literal shape (`.Set("contact_id",
  contactID.Bytes())`) is a manual single-UUID-to-binary conversion that does
  not transfer to a JSON-array column; a bare `.Set("tag_ids", tagIDs)` would
  fail at `Exec()` time (unsupported driver type). Fixed: §"Design decision"
  now specifies routing the value through
  `commondatabasehandler.PrepareFields` on a small ad-hoc map, matching
  `bin-queue-manager`'s own `QueueUpdate`/`UpdateTagIDs` precedent exactly
  (verified via `mapping.go`'s map-input auto-JSON-detection), with a full
  code sketch for `caseUpdateTagIDsExec`/`CaseUpdateTagIDs`.

## Approval status

**APPROVED** — round 7 and round 8 both returned `VERDICT: APPROVED` (2
consecutive), closing the design review loop after 6 rounds of real fixes
(rounds 1-4, 6). Ready for Phase 4 implementation.
