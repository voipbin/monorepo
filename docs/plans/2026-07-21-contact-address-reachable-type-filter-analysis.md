# Analysis: Filter Contact.Addresses to "reachable" types only

Status: Draft, Iteration 2 (Phase 0.8 analysis, pre-design)

## 1. Problem

`bin-contact-manager`'s `contact_addresses` table currently only allows two
address types at write time (`isValidContactAddressType` in
`pkg/contacthandler/contact.go:39-46`: `tel`, `email`). A separate product
discussion (not yet implemented, not yet scoped) considered widening this
to also register webchat-session-derived addresses
(`commonaddress.TypeWebchat`) against a Contact, to solve read-time
auto-matching for webchat interactions whose `peer_target` is a session
UUID that is never known in advance.

If/when that widening happens, a single Contact could accumulate a
`contact_addresses` row per webchat session (each session's `peer_target`
is a fresh UUID with no re-use across visits), causing the row count for a
single `contact_id` to grow roughly proportional to visit count instead of
staying near-constant like tel/email.

This analysis addresses a **read-side prerequisite**: `GET
/v1/contacts/{id}` (and any other read path that populates
`Contact.Addresses`) must be able to return only "reachable" addresses
(tel/email today; any future outbound-capable channel later) without
returning O(session-count) rows of a non-reachable type, regardless of
whether/when the webchat-address-write widening ships.

**Scope note:** this analysis covers ONLY the read-side filtering
mechanism. It does NOT implement webchat address writes (still blocked by
`isValidContactAddressType`) — that remains a separate, not-yet-scoped
follow-up.

**Iteration 2 addition (see §3.1):** since `contact_addresses.type` can
today ONLY ever be `tel` or `email` (the write path enforces this and the
one historical backfill also only produced these two values — verified in
§2), the proposed read-side filter is a **behavioral no-op today.** §3.1
below discusses whether to do this work now anyway, as pre-emptive
insurance, or defer it.

## 2. Current state (verified against code + real migrations)

- `pkg/dbhandler/address.go:359-388` `AddressListByContactID(ctx, contactID)`:
  `SELECT ... FROM contact_addresses WHERE contact_id=? ORDER BY is_primary
  DESC, tm_create ASC` — no `type` filter, no `LIMIT`. This is the sole
  function backing the public `Contact.Addresses` API field (via
  `ContactGet`/`ContactList`/`contactUpdateToCache`).
- Real production migration `bin-dbscheme-manager/bin-manager/main/versions/
  ac5d4e18060c_contact_crm_create_tables.py:76`:
  `INDEX idx_contact_addresses_contact (contact_id)` — single-column index.
  (Corrects an earlier verbal reference to `idx_contact_addresses_contact_id`
  — the real name is `idx_contact_addresses_contact`.)
- Same file line 77: `UNIQUE INDEX idx_contact_addresses_identifier
  (customer_id, type, target)` — this is the real name of what was verbally
  called `idx_contact_addresses_lookup` (that name only exists in the
  throwaway SQLite test schema `scripts/database_scripts_test/contacts.sql`,
  a pre-existing test/prod naming drift unrelated to this change).
- `contact_addresses` is hard-delete (no `tm_delete` column) — confirmed in
  the same migration file's table DDL and doc comments across
  `bin-contact-manager/docs/domain.md` and `CLAUDE.md`.
- **Corrected migration audit** (iteration 1 cited the wrong 5 revisions —
  those were webchat_widgets/contact_cases changes that never touched
  `contact_addresses` at all). The migrations that actually DO touch
  `contact_addresses`, checked directly:
  - `7e981e3aa8e9_contact_addresses_add_name_detail.py` — adds `name`,
    `detail` columns via `ALTER TABLE`. No index change.
  - `bbcf80d332eb_contact_crm_m1_migrate_addresses.py` — one-time backfill
    `INSERT INTO contact_addresses ... SELECT ... FROM
    contact_phone_numbers/contact_emails` (line 59). This is the ONLY
    historical data-population path, and it only ever wrote `type='tel'`
    or `type='email'` (sourced from the two legacy single-type tables it
    was merging). No index change.
  - `2d8f0ea90565_contact_address_ownership_periods_.py` — adds the
    separate, additive `contact_address_ownership_periods` table; also
    hard-deletes a small number of A9-b-corrupted `contact_addresses` rows
    (line 195-202) as cleanup, unrelated to indexes.
  - `1d0f4d07ff58_contact_drop_legacy_phone_numbers_and_.py` — drops the
    now-redundant `contact_phone_numbers`/`contact_emails` tables (their
    data was already migrated into `contact_addresses` by `bbcf80d332eb`).
    No index change.
  - Conclusion (now verified against the correct file set): **no migration
    since `ac5d4e18060c` has altered `contact_addresses`'s indexes, and no
    migration has ever written any `type` value other than `tel`/`email`
    into the table.** This directly supports §3.1's "no-op today" finding.
- `pkg/contacthandler/interaction_read.go:133-137`'s "design §6.1" comment
  is about a DIFFERENT constraint than this analysis initially implied: it
  states `AddressListByContactID` must NOT be reused by the *ownership-period
  interaction-matching* feature (a different read path,
  `OwnershipPeriodsListByContactID`), because widening ITS semantics would
  leak closed/historical addresses into the public API. It does NOT mean
  "this function's own SQL query must never be modified." This analysis's
  proposed change (adding a `type IN (...)` filter to
  `AddressListByContactID`'s own query, §4) does not touch or widen the
  ownership-period read path at all, so there is no actual conflict with
  design §6.1's constraint. (Iteration 1 introduced a self-contradiction by
  conflating these two meanings — corrected here.)

## 3. Root cause

The read path has no notion of "reachable vs. historical/session" address
types. Any type stored in `contact_addresses` is returned unconditionally
and unbounded. Today this is harmless in practice — `type` can only be
`tel`/`email` (§2) — but it is a **latent landmine** for any future address
type whose cardinality scales with interaction count rather than with
"distinct ways to reach a person."

### 3.1 Is this worth doing now, given it is a no-op today?

Two considerations pull in opposite directions:

- **For doing it now:** the change is small, isolated, and additive (no
  schema risk beyond an index swap). Landing it before any future
  address-type widening means that widening PR only has to worry about the
  write path (whitelist) and retention, not the read path — reducing that
  future PR's blast radius and review surface.
- **Against doing it now:** the real query/cardinality needs of a future
  webchat-address (or other type) feature are unknown until that feature is
  actually scoped. Locking in `ReachableAddressTypes = {tel, email}` and an
  index shape now is a bet that the eventual feature's actual read pattern
  matches this guess. If that feature instead needs, say, a per-type page
  size or a different sort order, this index may need to be redone anyway.

**Recommendation:** proceed now, but keep the change minimal and exactly
scoped to "filter to today's known-reachable set" — do not speculatively
build any parameterization (e.g. no `?address_types=` query param) for a
feature that isn't scoped yet. If the future feature's needs differ from
this guess, redoing one index migration later is cheap; NOT doing this now
and instead having `AddressListByContactID` silently balloon in an
already-shipped webchat-write feature is a worse failure mode (an incident,
not a planned migration). This is called out as an explicit judgment call
for the design reviewer to confirm or override, not a settled fact.

## 4. Recommended approach

1. Add an application-level constant (NOT a new DB column) enumerating
   "reachable" types, e.g. `var ReachableAddressTypes =
   []commonaddress.Type{commonaddress.TypeTel, commonaddress.TypeEmail}`
   in `bin-contact-manager/pkg/contacthandler` (see §8.2 for package
   placement rationale — corrected from iteration 1's wrong claim). Starts
   identical to `isValidContactAddressType`'s write whitelist; the two
   lists are allowed to diverge later (write whitelist gates what CAN be
   created, reachable list gates what displays as "contactable") but today
   they are the same two types.
2. Change `AddressListByContactID`'s query to add `WHERE contact_id=? AND
   type IN (?)` using the constant from (1). Preserve existing
   `ORDER BY is_primary DESC, tm_create ASC`. This does NOT touch or widen
   `OwnershipPeriodsListByContactID` / the interaction-matching read path
   (see §2's correction — no conflict with design §6.1).
3. Change the index — see §8.1 for the empirically-settled decision
   (2-column `(contact_id, type)` index, verified via a real `EXPLAIN` run
   against a seeded MySQL 8.0 instance).
4. Explicitly OUT OF SCOPE: implementing webchat address writes,
   retention/TTL cleanup of any future session-derived address rows, and
   any UI change.

## 5. Independent verification history

- **Round 1 (2 parallel subagents, completed 2026-07-21 ~04:00):**
  - Subagent A (fact-check of the whole conversation's technical claims):
    5 of 6 claims CONFIRMED with exact file:line citations; 1 claim
    (hypothetical unbounded row growth from a not-yet-implemented feature)
    judged PARTIALLY CORRECT (logically sound, describes a not-yet-built
    scenario).
  - Subagent B (critique of the index-based recommendation): confirmed
    direction is sound and matches project convention, but flagged: (a)
    the composite index is a REPLACEMENT of the existing single-column
    index, not a parallel addition; (b) the proposed index does not
    necessarily cover the existing `ORDER BY`, risking filesort; (c)
    row-count growth is a retention/TTL problem that a read-filter does not
    solve — must stay explicitly out-of-scope, not silently implied as
    solved.
- **Round 2 review (independent design-review subagent, 2026-07-21
  ~04:10), verdict CHANGES_REQUESTED, 7 items — all addressed in this
  iteration:**
  1. Self-contradiction between old §2/§4/§5 on whether
     `AddressListByContactID` may be modified — **resolved**: §2 now
     clarifies the actual scope of design §6.1's constraint (does not
     forbid this change); §4 point 2 states explicitly there is no
     conflict.
  2. Factual error claiming `ReachableAddressTypes`'s recommended package
     placement "mirrors" `isValidContactAddressType`'s location —
     **corrected** in §8.2: the function actually lives in
     `pkg/contacthandler`, not `models/contact`; recommendation updated to
     match (package `pkg/contacthandler`, not `models/contact`).
  3. Migration audit cited the wrong 5 revisions (none of which touch
     `contact_addresses`) — **corrected** in §2: replaced with the 4
     migrations that actually touch the table
     (`7e981e3aa8e9`, `bbcf80d332eb`, `2d8f0ea90565`, `1d0f4d07ff58`), all
     independently re-verified to make no index changes.
  4. Unverified/optimistic claim that a 4-column composite index "covers
     WHERE + ORDER BY fully" — **corrected** in §8.1: now flagged as NOT
     settled; `type IN (tel, email)` is a multi-value predicate and MySQL's
     optimizer may not use a single range scan to satisfy the trailing
     `ORDER BY` across multiple IN-list values; an `EXPLAIN`-based
     verification step against a populated table is now a required
     pre-design-approval step, not an assumption.
  5. No discussion of whether doing this now (a behavioral no-op today) is
     justified vs. deferring until the webchat-write feature is scoped —
     **added** as new §3.1, with an explicit recommendation and rationale,
     flagged for reviewer override.
  6. "Replacement not parallel addition" finding was buried in §5 but never
     promoted to a Goal/Open Question like the others — **promoted**: now
     explicit in §6 Goal 2 and §8.1.
  7. No maintenance-risk note for `ReachableAddressTypes` silently drifting
     out of sync with new `commonaddress.Type` values — **added** as §7
     non-goal caveat and §9 risk note.

## 6. Goals for the design doc

1. `GET /v1/contacts/{id}` and `GET /v1/contacts` (list, if it also
   populates Addresses) return only tel/email addresses, sourced from an
   application-level constant, not a DB column.
2. The index change is a **replacement** of
   `idx_contact_addresses_contact (contact_id)`, not a parallel addition
   (its leftmost-prefix already covers every existing bare
   `WHERE contact_id=?` query). The exact replacement column list must be
   settled via `EXPLAIN` verification (§8.1) before design approval, not
   assumed.
3. Explicitly document that this does NOT solve row-count growth for any
   future non-tel/email address type; that remains a separate, deliberately
   deferred, future decision (§7).
4. No schema change beyond the index (no new columns, no new tables).

## 7. Non-goals

- Implementing webchat (or any other new type) address writes.
- Retention/TTL/cleanup of any address rows. **Explicit caveat:** this
  change filters what the READ path returns; it does nothing to slow or
  stop row INSERTs. If a future feature starts writing high-cardinality
  address rows, this change alone does not bound table growth — that is a
  separate, not-yet-scoped follow-up.
- Any UI/API contract change beyond the returned Addresses being filtered
  (the JSON shape of each Address element is unchanged).
- Any query parameterization (e.g. `?address_types=`) for hypothetical
  future consumers who might want the full unfiltered list — not needed by
  any known consumer today; add only if/when a real consumer asks.

## 8. Open questions for design review

### 8.1 Index shape — SETTLED via empirical `EXPLAIN` verification (2026-07-21)

**Test setup:** MySQL 8.0 (throwaway Docker container, discarded after the
test), schema identical to `ac5d4e18060c`'s `contact_addresses` DDL,
seeded with ~10,660 rows: ~200 customers x 20 contacts each with 1-3
tel/email rows (representative normal case), plus 5 "power" contacts each
carrying 500 `webchat`-type rows (the exact stress scenario §3/§9 worry
about — simulates the not-yet-scoped future feature).

**Baseline (current single-column index `idx_contact_addresses_contact
(contact_id)`)**, queried against the power contact (502 total rows: 500
webchat + 2 tel/email):

```
EXPLAIN SELECT ... FROM contact_addresses WHERE contact_id=? AND type IN
('tel','email') ORDER BY is_primary DESC, tm_create ASC;

key: idx_contact_addresses_contact
rows: 502
Extra: Using index condition; Using where; Using filesort
```

Confirms the doc's core concern empirically: today's index forces MySQL to
read **all 502 rows** (including the 500 non-reachable `webchat` rows) for
this contact, filtering the type down to 2 rows only AFTER the read.

**Candidate A — `(contact_id, type)`:**

```
key: idx_cid_type
rows: 2
Extra: Using index condition; Using filesort
```

**Candidate B — `(contact_id, type, is_primary, tm_create)`:**

```
key: idx_cid_type_primary_tmcreate
rows: 2
Extra: Using index condition; Using filesort
```

**Finding:** both candidates cut scanned rows from 502 to 2 (250x
reduction for this contact) — this is the change that matters. Neither
candidate eliminates `Using filesort`, confirming §8.1's original
optimizer-limitation caveat: MySQL does not use a single range scan to
satisfy a trailing `ORDER BY` across a multi-value `type IN (...)`
predicate, regardless of how many extra columns are added to the index.
Extending to 4 columns buys nothing over 2 columns in this measured case
(identical `rows` and identical `Extra`).

**Decision: use the 2-column `(contact_id, type)` index**, not the
4-column variant. Rationale: the 4-column index adds index size and
maintenance cost (every INSERT/UPDATE writes 2 extra indexed columns) for
zero measured benefit; the filesort that remains operates over the
already-tiny **filtered** row count (2, not 502) — its cost is
categorically different from a filesort over hundreds of unfiltered rows,
and is not worth the index-size trade-off to chase away entirely, so the
design doc need not require its removal.

**Note on baseline `Extra` value:** a subsequent review pass questioned
whether `Using index condition` should appear in the baseline (single-
column `contact_id` index) `Extra` output, on the theoretical grounds that
Index Condition Pushdown normally requires the pushed predicate's column
to be part of the index, and `type` is not part of
`idx_contact_addresses_contact`. This was independently re-verified by
re-running the exact same `EXPLAIN` a second time against a freshly
reseeded MySQL 8.0 instance (fresh container, fresh data, same schema/
query): the output reproduced identically —
`Extra: Using index condition; Using where; Using filesort`. This
twice-reproduced empirical result is reported as-is; the exact underlying
mechanism is NOT asserted as settled here (a more mundane, standard
explanation is that ICP partially pushes the `contact_id` equality — which
IS part of the index — down to storage, while `Using where` separately
applies the non-indexed `type` filter afterward; this is a plausible
reading but is not confirmed against MySQL's own documentation and should
not be treated as verified fact). What IS settled is that the reported
`rows`/`key` values are real MySQL 8.0 output, not a transcription error,
and the `rows: 502 → rows: 2` comparison across index shapes (the actual
decision-relevant number) is unaffected either way.

**Naming note:** the verification commands below use a scratch name
(`idx_cid_type`) for the throwaway test index; the actual production
migration (see below) names the replacement index
`idx_contact_addresses_contact_type`. These are deliberately different —
the scratch name was only used inside the disposable Docker test
container.

**Concrete verification commands (for the future design-doc author /
implementer to re-run against the actual dev/staging database before
this ships):**

```sql
ALTER TABLE contact_addresses ADD INDEX idx_cid_type (contact_id, type);
EXPLAIN SELECT id, customer_id, contact_id, type, target, target_name,
    is_primary, tm_create, tm_update
FROM contact_addresses
WHERE contact_id = ? AND type IN ('tel','email')
ORDER BY is_primary DESC, tm_create ASC;
-- Confirm: key = idx_cid_type, rows small (not full per-contact row count)
```

The index change is a **replacement** of the current single-column
`idx_contact_addresses_contact (contact_id)` (per Goal 2) — its
leftmost-prefix already covers all existing bare `contact_id`-only
callers, so no other query loses index coverage. Migration:
`ALTER TABLE contact_addresses DROP INDEX idx_contact_addresses_contact,
ADD INDEX idx_contact_addresses_contact_type (contact_id, type);` (single
combined ALTER, InnoDB `ALGORITHM=INPLACE, LOCK=NONE` on MySQL 8.0 /
MariaDB 10.2+, safe as an online operation regardless of table size).

### 8.2 Package placement for `ReachableAddressTypes`

Corrected from iteration 1: `isValidContactAddressType` actually lives in
`pkg/contacthandler/contact.go` (package `contacthandler`), NOT in
`models/contact` (that package only holds the unrelated
`AddressTypeTel`/`AddressTypeEmail` string constants — verified at
`models/contact/address.go:32-35`).

**Recommendation:** place `ReachableAddressTypes` in
`pkg/contacthandler` alongside `isValidContactAddressType`, since both are
contact-manager-domain read/write gating concerns over the same two
values, not a platform-wide `bin-common-handler` concept.

## 9. Risks

- **Maintenance drift risk:** `ReachableAddressTypes` must be manually kept
  in sync with any future addition to `commonaddress.Type` that should be
  considered "reachable" (e.g. a future WhatsApp integration). If someone
  adds a new outbound-capable address type to the write whitelist
  (`isValidContactAddressType`) without also adding it to
  `ReachableAddressTypes`, the API will silently under-return valid
  contactable addresses with no test failure to catch it. Mitigation to
  consider at implementation time: a unit test asserting
  `ReachableAddressTypes` is a subset of (or equal to) the write whitelist,
  so any future divergence is caught explicitly rather than silently.
- **Index-shape risk:** RESOLVED via empirical `EXPLAIN` verification —
  see §8.1. `(contact_id, type)` cuts scanned rows 250x (502→2) in the
  measured stress scenario; the residual filesort operates over the small
  filtered set, not the full unfiltered row count, and is accepted as
  negligible.
- **No-op-today risk:** see §3.1 — this work has no observable effect until
  a future, not-yet-scoped feature ships. Accepted as a deliberate
  insurance trade-off, subject to reviewer override.
