# Design: Filter Contact.Addresses to reachable types + supporting index

Status: Draft (Phase 2)

Prerequisite: analysis at
`docs/plans/2026-07-21-contact-address-reachable-type-filter-analysis.md`
(APPROVED, 2 consecutive APPROVED verdicts, 2026-07-21). This design
implements that analysis's recommended approach. Read the analysis first
for full rationale, the two independent-subagent verification rounds, and
the empirical `EXPLAIN` results this design's index choice is based on.

## 1. Problem statement

`GET /v1/contacts/{id}` and any other read path that populates
`Contact.Addresses` currently return EVERY row in `contact_addresses` for
a given `contact_id`, unconditionally and without a `LIMIT`
(`AddressListByContactID`, `bin-contact-manager/pkg/dbhandler/address.go:
359-388`). Today this is harmless because `contact_addresses.type` can
only ever be `tel` or `email` (enforced by
`isValidContactAddressType`, `pkg/contacthandler/contact.go:39-46`, and
confirmed against the one historical backfill migration
`bbcf80d332eb`). But it is a latent landmine: if any future feature widens
the write path to allow a high-cardinality address type (e.g. a
webchat-session-derived address, `commonaddress.TypeWebchat`, whose
`peer_target` is a fresh UUID per session with no re-use across visits),
a single Contact's address count could grow unbounded, and every existing
consumer of `Contact.Addresses` would silently start returning and paying
for those rows too.

This design closes that gap now, as a small, isolated, additive change,
before any such future feature exists — see analysis §3.1 for the
"why now, given it's a no-op today" judgment call (accepted).

## 2. Goals

1. `AddressListByContactID` returns only addresses whose `type` is in a
   single, named, application-level "reachable types" list — today exactly
   `{tel, email}` — instead of every row for the `contact_id`.
2. The underlying index changes from a single-column
   `idx_contact_addresses_contact (contact_id)` to a composite
   `idx_contact_addresses_contact_type (contact_id, type)`, so the new
   `type IN (...)` filter is satisfied by an index range scan instead of a
   full per-contact table scan (empirically verified in the analysis doc:
   502 rows scanned → 2 rows scanned in the measured stress case).
3. Existing sort order (`is_primary DESC, tm_create ASC`) is preserved
   exactly; observable JSON shape of each returned `Address` element is
   unchanged.
4. `ReachableAddressTypes` is defined in exactly one place and reused by
   both the query filter and a new unit test that keeps it honest against
   `isValidContactAddressType`'s write whitelist (see §5.4).

## 3. Non-goals

- Implementing any new address type write path (e.g. webchat). Still
  blocked by `isValidContactAddressType`; out of scope here.
- Retention/TTL/cleanup of any `contact_addresses` rows. This change only
  affects what a read path RETURNS; it does not slow or stop row
  `INSERT`s. If a future feature starts writing high-cardinality address
  rows, unbounded table growth remains a separate, not-yet-scoped
  follow-up (analysis §7).
- Any new query parameter (e.g. `?address_types=`) for hypothetical future
  consumers wanting the unfiltered list. No known consumer needs this
  today.
- Any change to `AddressList` (the customer-wide, non-contact-scoped
  listing function at `address.go:292-353`) or to
  `OwnershipPeriodsListByContactID` / the interaction-matching read path
  (`address_ownership_read.go`) — both are untouched and out of scope.

## 4. Affected files

| File | Change |
|---|---|
| `bin-contact-manager/models/contact/address.go` | Add `ReachableAddressTypes` package-level var next to the existing `AddressTypeTel`/`AddressTypeEmail` constants (§5.1) — this is the ONLY location for the new var; NOT `pkg/contacthandler/contact.go`, which would cause a circular import (see §5.2) |
| `bin-contact-manager/pkg/contacthandler/contact_test.go` | Add unit test asserting `ReachableAddressTypes` is a subset of the write whitelist (drift guard, analysis §9 risk mitigation) |
| `bin-contact-manager/pkg/dbhandler/address.go` | Modify `AddressListByContactID`'s SQL to add `WHERE ... AND type IN (?)`; update the function's now-stale doc comment (§5.2) |
| `bin-contact-manager/pkg/dbhandler/address_test.go` | Extend `Test_AddressListByContactID` to insert a non-reachable-type row and assert it is excluded |
| `bin-contact-manager/scripts/database_scripts_test/contacts.sql` | Replace `idx_contact_addresses_contact_id` with composite `idx_contact_addresses_contact_type (contact_id, type)` (SQLite test schema, keeps parity with the real migration below) |
| `bin-dbscheme-manager/bin-manager/main/versions/<new>_contact_addresses_replace_contact_index_with_type.py` | New Alembic migration: `DROP INDEX idx_contact_addresses_contact` + `ADD INDEX idx_contact_addresses_contact_type (contact_id, type)` |

No changes to: `models/contact/contact.go`,
`pkg/dbhandler/address_ownership.go`, `pkg/dbhandler/address_ownership_read.go`,
`pkg/contacthandler/interaction_read.go`, any OpenAPI spec (response shape
of `Address` is unchanged — only which rows are included changes), any
frontend (`square-admin`) code.

## 5. Exact changes

### 5.1 `models/contact/address.go` — new `ReachableAddressTypes`

**Note:** this supersedes an earlier draft of this section that proposed
placing `ReachableAddressTypes` in `pkg/contacthandler`. That placement
was found to cause a circular import (§5.2 below) and is corrected here
directly — this is the only version implementers should use.

Add to `bin-contact-manager/models/contact/address.go`, next to the
existing `AddressTypeTel`/`AddressTypeEmail` constants (`address.go:32-35`):

```go
// ReachableAddressTypes is the set of commonaddress.Type values considered
// "reachable" (usable to contact the person) for the public
// Contact.Addresses API field. Distinct from
// contacthandler.isValidContactAddressType (which gates what CAN be
// WRITTEN to contact_addresses): the two lists happen to be identical
// today (tel, email) but are allowed to diverge -- e.g. a future
// write-side type that is intentionally NOT surfaced as "reachable" (a
// session/history-only address) would be added to the write whitelist
// without being added here. See
// Test_ReachableAddressTypes_SubsetOfWriteWhitelist in
// pkg/contacthandler/contact_test.go, which asserts this list never
// silently grows to include a type the write path doesn't (yet) allow --
// the reverse direction (a type in the write whitelist but NOT in
// ReachableAddressTypes) is a legitimate, intentional future state and is
// NOT asserted against.
var ReachableAddressTypes = []commonaddress.Type{
	commonaddress.TypeTel,
	commonaddress.TypeEmail,
}
```

Placed in `models/contact` (not `pkg/contacthandler`, where
`isValidContactAddressType` lives) specifically to avoid the import cycle
described in §5.2: `models/contact` is already imported by both
`pkg/dbhandler` and `pkg/contacthandler`, and imports neither of them.

### 5.2 `pkg/dbhandler/address.go` — filtered query

**Import-cycle rationale (why §5.1 is NOT in `pkg/contacthandler`):**
`pkg/contacthandler/contact.go:16` already imports
`"monorepo/bin-contact-manager/pkg/dbhandler"`. If `ReachableAddressTypes`
lived in `pkg/contacthandler` and `pkg/dbhandler/address.go` tried to
import it back, that would be a `dbhandler -> contacthandler ->
dbhandler` circular import, which Go rejects at compile time. Placing the
constant in `models/contact` instead (§5.1) avoids this entirely, since
`models/contact` has no dependency on either package and is already
imported by both.

Current (`address.go:359-368`):

```go
func (h *handler) AddressListByContactID(_ context.Context, contactID uuid.UUID) ([]contact.Address, error) {
	query, args, err := sq.Select(addressRowColumns()...).
		From(addressTable).
		Where(sq.Eq{"contact_id": contactID.Bytes()}).
		OrderBy("is_primary desc", "tm_create asc").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. AddressListByContactID. err: %v", err)
	}
```

New (the only version to implement — compiles cleanly, `contact` package
is already imported in this file for the `contact.Address` return type):

```go
func (h *handler) AddressListByContactID(_ context.Context, contactID uuid.UUID) ([]contact.Address, error) {
	reachable := make([]string, 0, len(contact.ReachableAddressTypes))
	for _, t := range contact.ReachableAddressTypes {
		reachable = append(reachable, string(t))
	}

	query, args, err := sq.Select(addressRowColumns()...).
		From(addressTable).
		Where(sq.Eq{"contact_id": contactID.Bytes()}).
		Where(sq.Eq{"type": reachable}).
		OrderBy("is_primary desc", "tm_create asc").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. AddressListByContactID. err: %v", err)
	}
```

Also update the function's doc comment (currently `address.go:356-358`):

```go
// AddressListByContactID returns all addresses for a contact.
// contact_addresses has no soft-delete. NOT modified by this design
// (design §6.1): shared by ContactGet/ContactList/contactUpdateToCache
// to populate the public Contact.Addresses API field.
```

to:

```go
// AddressListByContactID returns a contact's addresses, filtered to
// contact.ReachableAddressTypes (today: tel, email). contact_addresses
// has no soft-delete. Shared by ContactGet/ContactList/
// contactUpdateToCache to populate the public Contact.Addresses API
// field. The design §6.1 constraint this comment used to reference
// concerns NOT reusing this function for the separate ownership-period
// interaction-matching read path (see
// pkg/contacthandler/interaction_read.go's own comment at that call
// site) -- it does not mean this function's own query may never change;
// see docs/plans/2026-07-21-contact-address-reachable-type-filter-design.md.
```

(The old comment's "NOT modified by this design (design §6.1)" is now
false — this design directly modifies this function's query — and must
not survive unedited, or it will misinform future readers exactly the way
it initially confused this design's own analysis-stage drafting; see the
approved analysis doc §2's correction of that same misreading.)

**Verified call sites (no signature change needed):**
`AddressListByContactID`'s signature stays `(ctx, contactID) ([]contact.Address,
error)` — unchanged. Confirmed exactly 3 call sites, all in
`pkg/dbhandler/contact.go`: line 75 (`ContactGet`), line 152
(`ContactList`'s per-row hydration path), line 233 (a third read path,
`contactUpdateToCache`'s pre-cache hydration). All three are
`res.Addresses, _ = h.AddressListByContactID(ctx, id)` /
`c.Addresses, _ = h.AddressListByContactID(ctx, c.ID)` — none pass
anything beyond `ctx`/id today, so a same-signature change requires zero
call-site edits. The function is also declared in the `dbhandler.Handler`
interface at `pkg/dbhandler/main.go:44` — no signature change there
either.

### 5.3 Alembic migration

Generate via `alembic revision` (per `bin-dbscheme-manager/CLAUDE.md` —
never hand-craft a revision id):

```bash
cd bin-dbscheme-manager/bin-manager
alembic -c alembic.ini revision -m "contact_addresses_replace_contact_index_with_type"
```

Then edit the generated file's `upgrade()`/`downgrade()`:

```python
def upgrade():
    op.execute("""
        ALTER TABLE contact_addresses
            DROP INDEX idx_contact_addresses_contact,
            ADD INDEX idx_contact_addresses_contact_type (contact_id, type);
    """)


def downgrade():
    op.execute("""
        ALTER TABLE contact_addresses
            DROP INDEX idx_contact_addresses_contact_type,
            ADD INDEX idx_contact_addresses_contact (contact_id);
    """)
```

Single combined `ALTER TABLE` (not two separate statements) so the
rename happens as one `ALGORITHM=INPLACE, LOCK=NONE` online DDL operation
on MySQL 8.0 / MariaDB 10.2+ (per analysis §8.1's migration-safety note).

### 5.4 New unit tests

**`pkg/contacthandler/contact_test.go`** (imports `models/contact` for
`contact.ReachableAddressTypes`; `isValidContactAddressType` is in the
same package as this test file, so it is referenced unqualified):

```go
func Test_ReachableAddressTypes_SubsetOfWriteWhitelist(t *testing.T) {
	for _, rt := range contact.ReachableAddressTypes {
		if !isValidContactAddressType(rt) {
			t.Errorf("ReachableAddressTypes contains %q, which isValidContactAddressType rejects -- "+
				"a type must be writable before it can be considered reachable", rt)
		}
	}
}
```

**`pkg/dbhandler/address_test.go`** — extend `Test_AddressListByContactID`
(currently `address_test.go:38-96`) to insert a third, non-reachable-type
row and assert it is excluded from the result:

```go
	// Insert a third address of a type NOT in ReachableAddressTypes
	// (simulating a hypothetical future webchat-session address) to
	// verify the read-side filter excludes it.
	addrID3 := uuid.FromStringOrNil("ab1b2c3d-0001-0001-0001-000000000005")
	insertTestAddress(t, dbTest, addrID3, customerID, contactID, "webchat", "session-abc123")

	res, err := h.AddressListByContactID(ctx, contactID)
	if err != nil {
		t.Fatalf("AddressListByContactID() error = %v", err)
	}
	if len(res) != 2 {
		t.Errorf("AddressListByContactID() len = %d, want exactly 2 (tel+email only, webchat excluded)", len(res))
	}
	for _, a := range res {
		if string(a.Type) == "webchat" {
			t.Errorf("AddressListByContactID() returned a webchat-type address; expected it filtered out")
		}
	}
```

Note: `insertTestAddress` (line 23-36) writes directly via raw SQL,
bypassing `isValidContactAddressType`'s write-path gate entirely — this
is intentional and already the existing test helper's behavior (it is how
the CURRENT test suite inserts tel/email rows too), so inserting a
`webchat` row this way to test the READ-side filter does not require
loosening any write-path validation.

### 5.5 SQLite test schema parity

`bin-contact-manager/scripts/database_scripts_test/contacts.sql:53`
currently has:

```sql
create index idx_contact_addresses_contact_id on contact_addresses(contact_id);
```

Replace with:

```sql
create index idx_contact_addresses_contact_type on contact_addresses(contact_id, type);
```

(Note: this test schema already uses the name
`idx_contact_addresses_contact_id`, distinct from the real migration's
`idx_contact_addresses_contact` — a PRE-EXISTING test/prod naming drift,
unrelated to this design, not introduced or worsened by this change.
Keeping the test schema's own existing naming convention rather than
forcing exact parity with the prod name is consistent with how this file
already diverges elsewhere, e.g. `idx_contact_addresses_lookup` vs. prod's
`idx_contact_addresses_identifier`.)

## 6. Verification plan

1. `cd bin-contact-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m` — full standard verification workflow, zero new failures.
2. Specifically confirm `Test_AddressListByContactID` (extended, §5.4) and the new `Test_ReachableAddressTypes_SubsetOfWriteWhitelist` both pass.
3. Confirm no import cycle: `go build ./...` must succeed after §5.2's package placement resolution.
4. Re-run the analysis doc's §8.1 `EXPLAIN` verification against the actual migrated schema (not just the throwaway Docker test) if a staging DB with realistic data is available before merge; otherwise the analysis doc's Docker-based verification stands as the pre-merge evidence.
5. Grep check: `grep -rn "idx_contact_addresses_contact\b" bin-contact-manager/ bin-dbscheme-manager/` after the change — confirm the old single-column index name has zero remaining references (only the new composite name and the migration's own `DROP INDEX idx_contact_addresses_contact` line, which references it by name intentionally to drop it).

## 7. Rollout / risk

- **Risk: index migration on a live table.** Analyzed and accepted in the
  analysis doc §8.1 — `ALGORITHM=INPLACE, LOCK=NONE` is safe regardless of
  table size on MySQL 8.0 / MariaDB 10.2+. No data migration, no
  downtime expected.
- **Risk: import cycle.** See §5.2 — resolved by placing
  `ReachableAddressTypes` in `models/contact`, not `pkg/contacthandler`.
  Must be confirmed by an actual `go build` at implementation time; this
  design's resolution is a plan, not yet verified against the compiler.
- **Risk: silent behavior change if a customer already has non-tel/email
  data.** Ruled out: `isValidContactAddressType` has ALWAYS enforced
  tel/email-only writes, and the only historical bulk-write migration
  (`bbcf80d332eb`) only ever wrote `tel`/`email` (analysis §2). There is no
  path by which a live customer already has a non-tel/email row that this
  change would newly hide.
- **Risk: maintenance drift.** Mitigated by the new
  `Test_ReachableAddressTypes_SubsetOfWriteWhitelist` (§5.4) — if a future
  PR adds a new writable type to `isValidContactAddressType` without
  considering `ReachableAddressTypes`, this only fails to auto-INCLUDE the
  new type (safe default: new types are excluded from "reachable" until
  someone deliberately adds them) and does NOT fail the build, so this
  test does not block landing new address types — it only catches the
  opposite (impossible-by-construction) direction.

## 8. Approval status

Draft — pending design review loop (min 3 rounds, 2 consecutive APPROVED
to close).

## 9. Open questions for design review

1. Should the SQLite test schema's index be renamed to exactly match a
   hypothetical future prod-name convention, or is preserving its
   existing (already-diverged) naming style, as done in §5.5,
   acceptable? Recommendation: acceptable, since exact prod-name parity
   was never a property of this test schema even before this change.
