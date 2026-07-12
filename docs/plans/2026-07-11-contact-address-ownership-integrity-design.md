# Contact-Address Ownership Integrity — Design Doc

- Status: DRAFT (under adversarial review loop; latest completed round recorded in §8)
- Owner: pchero (CEO/CTO), design partner: Hermes (CPO role)
- Ticket: NOJIRA (no JIRA ticket at time of writing)
- Affected service: `bin-contact-manager`
- Related prior work: VOIP-1204/1206/1207/1208/1209 (contact CRM interaction
  timeline), VOIP-1243/1245 (Case get-or-create explicit-only, Interaction.CaseID
  removal)

## 1. Problem statement

`contact_addresses` is a **hard-delete** table (no `tm_delete`). Every read path
that resolves "which interactions belong to this Contact" re-derives the answer
at query time by matching `contact_interactions.peer_type/peer_target` against
**whatever `contact_addresses` rows currently exist** for that Contact
(`interactionListByContact` STEP1: `AddressListByContactID`).

This produces three concrete, user-visible defects:

1. **Delete → history disappears.** Deleting a phone number/email from a Contact
   (`DELETE /v1/contact_addresses/{id}`, no reassignment) immediately removes that
   number's entire interaction history from the Contact's timeline, because the
   deleted row no longer contributes to the value-match set. The interaction rows
   themselves are untouched — only the *read-time re-derivation* breaks.
2. **Reassign → history leaks to the new owner.** If a deleted number is later
   registered to a *different* Contact, that Contact's timeline immediately shows
   the *previous* owner's interaction history, because value-matching has no
   concept of "which owner, during which time window."
3. **Delete → interaction reappears in the unresolved queue.** The same
   value-matching problem re-surfaces a previously-attributed interaction in
   `InteractionListUnresolved` once its matching address is deleted, even though
   nothing about the interaction itself changed.

The only existing mitigation is manual: an agent can attach a `contact_resolutions`
row (positive/negative) per interaction. This is a correct last line of defense but
is not automatic and does not scale to bulk history — it survives defect (1)/(2)
only for interactions someone has already individually resolved.

## 2. Scope

### In scope
- A new table, `contact_address_ownership_periods`, tracking **who owned a given
  (customer_id, type, target) during which time window**, as the backing store for
  automatic interaction-to-contact matching (replacing "currently exists" with
  "owned it at the time the interaction happened").
- Rewiring `interactionListByContact` (STEP1/STEP2) and `InteractionListUnresolved`
  to match against ownership periods instead of live `contact_addresses` rows.
- Two independently-confirmed existing bugs in `bin-contact-manager`, bundled into
  this same design/PR because they sit on the exact write paths this design must
  already touch (`AddAddress`/`UpdateAddress`/`RemoveAddress`/`ClaimAddress`):
  - **A9-b**: these four handlers do not check `Contact.TMDelete`, so a
    soft-deleted Contact can still have addresses added/updated/removed/claimed.
  - **B5**: `AddressUpdate`/`AddressDelete` do not check `RowsAffected`, so a
    request racing a concurrent delete silently returns 200 with no effect
    (`AddressClaim` already does this correctly; the pattern was not applied to
    its siblings).

### Explicitly out of scope
- **No new API.** All existing endpoints
  (`POST /v1/contacts` with inline addresses, `POST /v1/contacts/{id}/addresses`,
  `POST /v1/contact_addresses`,
  `PUT /v1/contact_addresses/{id}`, `DELETE /v1/contact_addresses/{id}`,
  `POST /v1/contact_addresses/{id}/claim`) keep their exact current signatures,
  request/response shapes, and status codes. This design is a pure internal
  rewire behind those endpoints. (Confirmed with pchero: no `transfer` endpoint.)
  The one behavior change, per §4's round-11 finding, is that `POST
  /v1/contacts`'s inline address registration now propagates
  `AddressCreate`/`AddressResetPrimary` errors instead of silently swallowing
  them — a correctness fix to existing broken error handling, not a new
  response shape (the error still surfaces through the same response
  envelope any other validation failure on that endpoint already uses).
- **No atomic reassignment RPC.** Reassigning a number between Contacts remains
  two independent calls (`DELETE` on the old Contact, then `POST`/`claim` on the
  new one), exactly as today. The TOCTOU gap between those two calls is a known,
  accepted limitation (§7).
- **D5 (originally flagged as "`TypeNone` missing from `crmIneligiblePeerTypes`")
  is DROPPED, not fixed.** Round-2 adversarial review (see §8) proved this is not
  a bug: `TypeNone`/zero-value peer is the deliberate "unknown direction" sentinel
  (`interaction.go:21-22`, covered by
  `Test_EventCallCreated_unknown_direction` in `interaction_test.go:106-138`).
  Adding it to the blacklist silently deletes a row the platform intentionally
  persists for diagnostics. No action taken.
- `Case.contact_id` derivation (`casehandler/contact_attribution.go`) and the
  still-unimplemented case-level auto-match (`AddressLookupContactIDByTypeTarget`
  dead code) — separate design track, not touched here.
- `case-control reconcile-contact` CLI — unaffected; continues to operate on
  `contact_resolutions`, independent of this table.
- Email/SMS/LINE/WhatsApp interaction projection — those channels do not feed
  `contact_interactions` at all today (no `SubscribeHandler`); out of scope.

## 3. Data model

### 3.1 New table: `contact_address_ownership_periods`

```sql
CREATE TABLE contact_address_ownership_periods (
    id          BINARY(16) NOT NULL,
    customer_id BINARY(16) NOT NULL,
    contact_id  BINARY(16) NOT NULL,   -- never NULL; unresolved addresses (CreateUnresolvedAddress)
                                        -- do not get a period until ClaimAddress assigns an owner

    type        VARCHAR(255) NOT NULL DEFAULT '',
    target      VARCHAR(255) NOT NULL DEFAULT '',

    valid_from  DATETIME(6) DEFAULT NULL,  -- NULL = unbounded past (see §8/§9 backfill)
    valid_to    DATETIME(6) DEFAULT NULL,  -- NULL = still open (current owner)

    -- "at most one OPEN period per (customer,type,target)" -- mirrors
    -- contact_cases.uq_case_open_peer (bin-contact-manager/pkg/dbhandler/kase.go,
    -- docs/plans/2026-07-07-contact-case-management-design.md §3.1) exactly:
    -- MySQL/MariaDB has no partial/filtered index, so a STORED generated column
    -- collapses to NULL (distinct under UNIQUE) for every closed period and to a
    -- deterministic hash for the single permitted open one.
    open_period_uk BINARY(32)
        GENERATED ALWAYS AS (
            IF(valid_to IS NULL,
               UNHEX(SHA2(CONCAT_WS('|', customer_id, type, target), 256)),
               NULL)
        ) STORED,

    tm_create   DATETIME(6),
    tm_update   DATETIME(6),

    PRIMARY KEY (id),
    UNIQUE INDEX idx_ownership_periods_open (open_period_uk),
    INDEX        idx_ownership_periods_contact (customer_id, contact_id),
    INDEX        idx_ownership_periods_lookup (customer_id, type, target, valid_from, valid_to)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci ROW_FORMAT=DYNAMIC;
```

This is a **new, additive** table. It does not alter `contact_addresses`,
`contact_interactions`, or `contact_resolutions` in any way — no column adds, no
index changes on existing tables. `contact_addresses` keeps its exact current
role (live "who owns this right now" cache) and its exact current hard-delete
semantics. Two deliberate, narrow exceptions to "nothing touches it
differently" exist, both cleanup of rows whose owner is gone: the migration
backfill hard-deletes A9-b-corrupted rows (§9, round-23) and
`ContactDelete`/`ContactDeleteByCustomerID` hard-delete the tombstoned
Contact's address rows alongside closing their periods (§4, round-25) —
in both cases producing exactly the state a timely `RemoveAddress` would
have left. No read path changes.

### 3.2 Why a separate table, not `valid_from`/`valid_to` columns on
    `contact_addresses` itself

Rejected alternative: add `valid_from`/`valid_to` directly to `contact_addresses`
and soft-delete instead of hard-delete. Rejected because:
- `contact_addresses` is read by `ContactGet`/`ContactList`/`contactUpdateToCache`
  (`pkg/dbhandler/contact.go:75,152,233`) to populate the public `Contact.Addresses`
  API field. Any change to what counts as a "row" there leaks into that API surface
  (a closed period must never appear in `GET /v1/contacts/{id}`). A separate table
  makes this structurally impossible instead of relying on every future reader
  remembering to filter.
- `contact_addresses` mirrors `agent_addresses`' hard-delete convention on
  purpose (per its own creation migration, `ac5d4e18060c`); changing that
  convention has a blast radius outside this design's scope.

## 4. Write paths

All five ownership-period-affecting operations are additions alongside the
existing `contact_addresses` write, inside the **same transaction** the existing
write already runs in (see §5 for the transaction restructuring this requires).

**Round-11 finding: `contacthandler.Create` (`ContactCreate`,
`pkg/contacthandler/contact.go:79-107`) is a fifth caller of
`dbhandler.AddressCreate`/`AddressResetPrimary` that this design had not
accounted for — its inline address-registration loop, run when a Contact is
created together with its initial addresses, calls both with the real
`contact_id = c.ID` (never `uuid.Nil`), so per the scope note above it is
squarely subject to the step procedure below, not exempt from it the way
`CreateUnresolvedAddress` is. It was missing from every reference to "the
four write handlers" in this document (§4's A9-b paragraph, §5.1's
transaction wrapping) purely because earlier rounds' review scope never
grepped for every *caller* of the dbhandler functions this design touches,
only re-examined the logic of handlers already in scope. Today this loop
also has no transaction and swallows `AddressCreate`/`AddressResetPrimary`
errors (`log.Warnf` + continue, no error propagation) — meaning a live-owner
conflict this design's Step 1 is meant to reject with `ErrConflict` would
today silently succeed instead, creating a Contact with an address it never
gets an ownership period for, while the real owner's open period stays open
untouched. **This design brings `ContactCreate`'s address loop under the
exact same discipline as `AddAddress`/`UpdateAddress`/`RemoveAddress`/
`ClaimAddress`: each address in the loop gets its own `BeginTx`-wrapped
operation running the §4 locking read and step procedure (with the same
fixed `AddressResetPrimary`-after-lock ordering §5.2 already specifies), and
errors from it (including Step 1's `ErrConflict`) propagate to the caller
instead of being logged and swallowed** — this is a correctness fix to
existing broken error-swallowing behavior, not new scope creep, since the
underlying bug (a live-owner conflict during Contact creation going
unnoticed) already exists today independent of this design.

**Round-12 finding: propagating the error, on its own, creates a new partial-
success state this design must own the disposition of.** `contacthandler.Create`
commits the base Contact row via `h.db.ContactCreate` (`contact.go:74`)
*before* the address loop runs (`contact.go:80-107`); each address in the
loop is its own separate `BeginTx` (round-11's fix). If address N in the loop
fails (e.g. Step 1's `ErrConflict`) after addresses 1..N-1 already committed
successfully, the caller now receives an error — but the Contact row and
those N-1 addresses (each with an already-open ownership period) remain
committed. This is a real gap, not accepted TOCTOU (§7) or out-of-scope
atomicity (§2): it is a direct consequence of round-11's own fix, not a
pre-existing limitation. **This design does not extend `ContactCreate`'s
outer scope to wrap the base Contact insert and every address in one
transaction** (that would be a materially larger change — moving Contact
creation itself under transactional control — outside this design's stated
goal of a same-API-surface internal rewire, §2). Instead: `contacthandler.Create`
gains explicit cleanup-on-partial-failure — if any address in the loop
returns an error, the handler stops the loop, and (in a *separate*, best-
effort operation, not nested inside the failed address's own transaction)
issues compensating `RemoveAddress` calls for every address that succeeded
earlier in this same loop, then still returns the original error to the
caller. **Round-31 correction (the compensating closure must DELETE the
period this same loop just created, not close it):** the round-13
non-event cleanup path originally reused `AddressDelete`'s ordinary
period rule (`valid_to = NOW()`, "the period row is never deleted") —
but applying that rule to a period *this failed create just INSERTed*
leaves a ghost: a Contact whose creation failed (never announced via
`ContactCreated`, likely retried) permanently holds a closed
`[NULL, NOW())` period for the target, so (i) the caller's retry with a
fresh Contact routes through Step 4 (`valid_from = NOW()`) instead of
the Step 5 (`valid_from = NULL`) a genuinely fresh attempt would get,
orphaning the target's pre-registration history as permanently
unresolved (defect #1 remade), and (ii) that pre-history surfaces on the
ghost Contact's timeline instead (defect #2 remade) — falsifying this
paragraph's own "converge back to no addresses were added" and
"fresh attempt" claims. Fix: the compensating cleanup path DELETEs the
ownership-period row(s) its own loop iteration inserted (identified
inside the same locked transaction by `contact_id = <the new Contact's
id>` and open state — Step 3's reopen branch can never have fired for a
brand-new contact_id, so the only period this contact can own for that
target is the one this loop just created), restoring the true "no rows"
state so a retry IS a fresh attempt. This is the single sanctioned
exception to "the period row is never deleted," scoped to compensating
cleanup of the same request's own inserts. The disposition for
interactions arriving in the brief window while the doomed period was
open: they re-match identically after the delete (value matching against
a `[NULL,∞)`-equivalent absent-period state via the retry's Step 5
period) or stay unresolved exactly as if the failed create had never
happened — no attribution is manufactured either way. This makes a failed `POST /v1/contacts` call converge back to "no
addresses were added" (the Contact row itself, and any of its own non-address
fields, are unaffected either way — this design does not touch that), so a
retry of the same request behaves as a fresh attempt rather than colliding
with partially-committed state. If the compensating cleanup calls
themselves fail (a further concurrency edge case), that failure is logged
but does not mask the original error returned to the caller — the caller
still sees why the address it cared about failed. **Round-32 update to
this failure disposition (the pre-round-31 text described the close-at-NOW
regime):** under the round-31 DELETE regime, a failed cleanup leaves an
OPEN period + live row under a ghost Contact whose `ContactCreated` was
never published — a retry then hits Step 1's live-owner conflict (the
ghost is not tombstoned, so ownership agreement passes) and receives 409,
not a fresh attempt, until the ghost is repaired. Recorded disposition:
this is a doubly-degraded best-effort-failure state (the cleanup itself
already failed, which the design accepts as log-and-continue), it is
self-identifying (the skew-orphan/cleanup-failure log line carries the
ghost's contact_id), and the standing repair paths apply — deleting the
ghost Contact via `DELETE /v1/contacts/{id}` (ContactDelete's round-25/26
cleanup closes the period and removes the row) or operator inspection via
`GET /v1/contacts?customer_id=...` listing (the ghost IS visible in
list reads; the log line carries the ghost's contact_id — **round-33
correction: do NOT rely on the create response body carrying the id; the
handler returns `(nil, err)` on failure per Go convention, so the log
line is the canonical identifier source**). **Round-33 disposition for
the post-recovery residue (previously undisposed):** `ContactDelete` is
the general-purpose path, so it CLOSES the ghost's period
(`valid_to=NOW()`) rather than applying round-31's DELETE exception
(that exception is scoped to the compensating cleanup's own transaction
and is not reachable from `ContactDelete`, which cannot know the
period's provenance). The post-recovery state is therefore a closed
ghost-owned `[valid_from (usually NULL), NOW())` period — the same
data shape round-31 eliminated from the SUCCESS path. Recorded as an
accepted limitation, deliberately NOT re-engineered: it is reachable
only through a double failure (the create failed AND its best-effort
cleanup failed), the subsequent retry mis-routes to Step 4 and the
target's pre-registration history stays attached to the tombstoned
ghost (suppressed from the unresolved queue by §6.3, invisible on the
ghost's NotFound timeline), and the standing manual correction path is
`contact_resolutions` (§7's correction mechanism — a case-level positive
resolution re-attributes any affected interaction), plus the heuristic
candidate query in §9.x's reconciliation appendix (round-34: the log
line is the canonical identifier — the query is a lossy fallback for
rotated logs, not an exact residue detector) that surfaces candidates
for operator review. This
is the same accepted-with-manual-escape standard §7 applies to its
other double-failure gaps.

**Round-13 finding: reusing `contacthandler.RemoveAddress` verbatim for this
compensating cleanup silently leaks a wrong event.** `RemoveAddress`
unconditionally publishes `EventTypeContactUpdated` on success
(`contact.go:393-417`). But when `ContactCreate` fails partway through its
address loop, `ContactCreated` was never published for this Contact (the
handler returns an error before ever reaching that point) — so a downstream
consumer would receive an `updated` event for a Contact it never saw
`created` for, a phantom-update with no corresponding prior state. **Fix
(as amended by round-31/32/35):** the compensating cleanup calls a
**non-event-publishing** internal path. **Round-35 naming correction:
this path is NOT `dbhandler.AddressDelete`** — that entry point owns the
close-at-NOW() period rule (§4 table) and calling it here would
reproduce the ghost-period bug round-31 fixed. The compensating path is
a **separate dbhandler entry point** (e.g.
`AddressDeleteCompensating(ctx, contactID, type, target)`), which in one
§5.1 transaction — following §5.2's canonical order: ownership-period
`FOR UPDATE` first, then the `contact_addresses` row delete, then the
period DELETE (round-36: order stated explicitly; no `contact_contacts`
read is needed since the caller is compensating its own just-failed
create) — hard-deletes the address row AND hard-DELETEs the open
period this same create-loop inserted (round-31's sanctioned
exception), publishing nothing. Round-36: this new function joins the
`DBHandler` interface in `pkg/dbhandler/main.go` and requires `go
generate` mock regeneration, exactly like `InteractionListByOwnershipPeriods`
(§6.2's round-19 rule); §10's checklist carries it. **Round-32
correction: the original round-13 text described this path as sharing
`RemoveAddress`'s "ownership-period Step-based closure" and factoring out
logic "both call sites need" — that sharing claim is FALSE after
round-31**: `RemoveAddress` closes the period (`valid_to=NOW()`, the
record of a real era) while the compensating path DELETEs the period its
own loop just inserted (round-31's sanctioned exception — closing it
would create the ghost-period bug round-31 fixed). The two paths share
only the row-delete portion; the period operation is deliberately
different per caller, and an implementer must NOT unify them behind one
helper that closes.

**Round-10 finding: `CreateUnresolvedAddress` (`contact_id = uuid.Nil`,
`pkg/contacthandler/contact.go:426`) calls the same `dbhandler.AddressCreate`
this section governs, but §3.1 already establishes that an unresolved
address must not get a period until `ClaimAddress` assigns a real owner.**
Applying the steps below with `contact_id = uuid.Nil` as "this contact" would
misfire — Step 1 in particular would treat *every* live owner anywhere as a
conflict against `Nil`, which is not the semantic this design intends and is
not the `ErrDuplicateTarget`/unique-index behavior `contact_addresses`
already provides for this path today. **This section's steps apply only when
`contact_id != uuid.Nil`.** `AddressCreate` calls with `contact_id ==
uuid.Nil` (i.e. only `CreateUnresolvedAddress`) skip the §4 locking read and
the steps below entirely — they write `contact_addresses` exactly as today,
with no ownership-period effect, and no period exists for that target until
a later `ClaimAddress` call (which always supplies a real `contact_id`) runs
the steps below for the first time.

**Round-8 review found the "1–5 cases" framing (round-6/7's fix) was not
actually mutually exclusive — several real states satisfy more than one
case's stated precondition (e.g. this contact's own row is open AND it also
has an older closed row; or this contact has a closed row AND a different
contact currently holds an open one), and the document never said which case
to check first. Implemented in list order, that ambiguity reproduces the
exact "unmapped 1062 from inserting over a live open row" bug this design
exists to prevent. This section replaces the flat case list with an ordered
decision procedure — an explicit sequence of checks against the one
already-locked row set, each one either resolving the write or falling
through to the next. Ordering makes exclusivity structural: once a step
matches, no later step is ever evaluated, so overlapping preconditions
between what were cases 3/4/5 can no longer both fire.**

The decision is made from ONE locking read, not several separate lookups.
Inside the transaction (§5.1), immediately after acquiring the row lock this
operation's target requires:

```sql
SELECT id, contact_id, valid_from, valid_to
FROM contact_address_ownership_periods
WHERE customer_id = ? AND type = ? AND target = ?
FOR UPDATE
```

This locks **every** period row for the target (there are realistically at
most a handful — one open, plus however many closed ones history has
accumulated), not the single `open_period_uk` row alone. Because this is a
real locking read under `FOR UPDATE` — not a plain, un-locked `SELECT` — it
reads the true latest committed state under InnoDB's locking-read semantics,
not a stale `REPEATABLE READ` snapshot (this is what closes round-5's
concurrency finding, §8). The write path then decides **in application code,
from this one already-fetched, already-locked row set**, by evaluating the
following steps **in this exact order** and stopping at the first match — no
further queries needed at any step:

**Step 1 — is there already an open row for a *different* contact_id?** (At
most one can exist, globally, because `open_period_uk` hashes only
`(customer_id, type, target)`.) If yes: **first verify ownership agreement
(same locked transaction): fetch the `contact_addresses` row for this
`(customer_id, type, target)` and require that it exists AND its
`contact_id` equals the blocking period's `contact_id` (round-24
correction: row existence alone is not sufficient — an unresolved row with
`contact_id = NULL`, or any other owner's row, occupying the slot does NOT
legitimize the period) AND — round-27 correction — **the owning Contact
itself is not tombstoned** (`contact_contacts.tm_delete IS NULL`, checked
in the same transaction): a matching row whose owner was soft-deleted by
an old-binary `ContactDelete` during the skew window (which knew nothing
of periods or the round-25 cleanup) is just as dead as a missing row, and
without this clause the ownership-agreement check would *pass* and
misjudge the tombstoned owner as live, permanently 409-locking the target.
If the row is gone, owned by anyone else, or owned by a tombstoned
Contact, the open period is an orphan: close it (`valid_to=NOW()`,
Prometheus counter) and continue to the next step as if it were closed.
**Round-31 correction (the previous text applied the late cleanup to all
three orphan causes unconditionally — wrong for two of them): the
round-28 late cleanup (closed-period-first + caller-specific row repair)
applies ONLY to the tombstoned-owner branch**, where a stale row and its
owner's `tm_delete` actually exist. In the row-gone branch there is no
row to repair and nothing to fabricate (the closed orphan period IS the
record) — close and continue is the whole action. In the
owned-by-anyone-else branch the occupying row may belong to a LIVE
contact (§9's accepted period-less degraded state) and must NOT be
touched — close the orphan and continue; if the caller's own
INSERT/UPDATE then collides with that row, the round-28 duplicate-key
path performs the tombstone check and either repairs (dead owner) or
propagates 409 (live owner). **Round-32 transaction-boundary
clarification for the live-owner outcome:** the 409 aborts the caller's
transaction, which rolls back the orphan close performed moments earlier
in that same transaction — no partial state escapes (correct), but it
also means a mixed-skew state with a LIVE occupant (B's orphan open
period + live C's period-less row) is never healed by other parties'
registration attempts: every such attempt ends 409-and-rollback.
Recorded disposition: the state is NOT permanently stuck — it heals
through the live occupant C's own next period-closing touch on the
target: **round-33 correction — the healing set is exactly {C's
`AddressDelete`, C's `AddressUpdate` with a target change (old-target
close)}**, both of which close by `open_period_uk` hash, which matches
B's orphan regardless of contact_id (RowsAffected==1, so round-17's
no-period counter correctly stays silent; the round-32 text also named
"C's `ClaimAddress`-based flows route through Step 2" as a healing path,
which was doubly wrong — with B's orphan open the procedure necessarily
hits Step 1, never Step 2, and `ClaimAddress` against C's live row is
rejected by its own pre-check/final-UPDATE guard with full rollback, so
ClaimAddress heals nothing; likewise C's `AddressCreate` (self-duplicate
409) and target-unchanged `AddressUpdate` (touches no period) do not
heal). Until one of the two healing operations occurs, the exposure is
B's timeline over-absorbing C's interactions — the same
bounded-by-next-touch standard as §9's other skew dispositions, with the
narrowing that "next touch" means next period-closing touch. This
hash-based healing close increments no counter (it is indistinguishable
in-band from an ordinary close), so the skew-orphan counter's coverage
claim is scoped accordingly: it counts Step-1/duplicate-key-path repairs,
not incidental hash-close healings. The
skew-orphan Prometheus counter must be incremented only AFTER commit
(post-commit hook or transactional outbox-style deferral), never inside
the transaction — otherwise every futile rolled-back close inflates the
repair count while the state remains unchanged. **Round-33 promotion:
this post-commit rule applies to ALL of this design's in-transaction
instrumentation, not just the skew-orphan counter** — the §9
round-16/17 `RowsAffected==0` counters have the same two inflation
sources (each §5.3 deadlock retry re-observes the same miss in its
fresh `BeginTx`, up to 3x duplication; and `AddressUpdate`'s old-target
close miss can be counted in a transaction later rolled back by the
new-target Step-1 409). Every counter defined by this design increments
post-commit only. **Round-34 implementation placement (previously
unspecified — the natural dbhandler-internal immediate increment is
exactly the form this rule forbids):** dbhandler functions do NOT call
the Prometheus counters directly. **The pending struct is scoped per
TRANSACTION (per `BeginTx` attempt), not per dbhandler call (round-35
correction — "call-scoped" was ambiguous for the two entry points where
one call owns multiple transactions):** a fresh `pendingMetrics{...}`
is created alongside each `BeginTx`, flushed to the package-local
counters (§9 round-17's `metricsNamespace` registration is unchanged)
immediately after THAT transaction's `tx.Commit()` returns nil, and
discarded on that transaction's rollback. Consequences: in repair (b)'s
three-transaction sequence each transaction's pending is independent —
T2's committed repair increment is flushed when T2 commits regardless
of whether T3 later succeeds (no lost count) and is never re-flushed by
T3 (no double count); in `ContactDeleteByCustomerID`'s per-contact
transactions each contact's increments flush with its own commit, so a
failure at contact k+1 neither loses contacts 1..k's counts nor counts
k+1's rolled-back attempt; in `ContactDelete`'s single N-way
transaction (round-14 re-check loop included) all loop passes
accumulate into the one pending struct and flush once at the single
commit. A §5.3 deadlock retry discards the previous attempt's pending
struct with the rolled-back transaction, eliminating both inflation
sources by construction. Commit observation and counter registration
live in the same layer (the transaction-owning dbhandler entry point,
per §5.1), so no cross-layer plumbing is needed. **Round-32
transaction-boundary rule for repair (b):** the duplicate-key repair
runs in a NEW §5.1 transaction opened after the caller's failed
INSERT/UPDATE transaction has rolled back (a 1062 poisons the
transaction it occurs in); the retry INSERT then follows in yet another
transaction, per the round-28/29 sequence. Row repair by
caller in the tombstoned-owner branch: `AddressCreate` hard-deletes the
stale row, `AddressUpdate` (new-target side) likewise hard-deletes it
(round-30: its own UPDATE then re-occupies the freed slot), `ClaimAddress`
resets it to `contact_id = NULL` so its own final UPDATE still finds the
NULL-owned row (see §4's round-28 correction below; §9's round-23/24
rule text is superseded by the round-27/28/31 refinements HERE — §4 is
canonical for the repair actions).** Otherwise this is a genuine
live-owner conflict.
Return `ErrConflict` (mapped to `cerrors.AlreadyExists`/409, exactly as
`ClaimAddress`'s existing pre-check does today — round-7's case 5) and roll
back. **Checked first, unconditionally**, so no later step can ever attempt
an INSERT that would collide with this row — this is what makes the old
case-3-vs-case-5 and case-2-vs-case-5 overlaps round-8 found structurally
impossible now, not just documented as an edge case.

**Step 2 — is there already an open row for *this* contact_id?** (Only
reachable here because step 1 already ruled out a different contact's open
row — `open_period_uk`'s global uniqueness means this and step 1 can never
both be true, so reaching step 2 tells you the open row, if any, is this
contact's own.) **Round-29 correction: Step 2 performs the SAME
ownership-agreement check as Step 1** (same locked transaction: the
`contact_addresses` row for this target must exist and carry this
contact_id) — round-24 fixed the slot-reoccupation scenario only for the
*different-owner* variant; the *same-owner* variant (B's own orphan open
period surviving an old-binary row deletion, then B itself re-registering)
routes here and previously had no check at all. Outcomes:
- **Row agrees (exists, owned by this contact):** caller-specific
  (round-30 — the previous text defined only `ClaimAddress`, leaving
  `AddressCreate`'s everyday duplicate path ambiguous). For
  `ClaimAddress`: true idempotent retry (round-6's case 4) — dbhandler
  commits (successful no-op, not an error), and
  `contacthandler.ClaimAddress`'s existing post-commit behavior (re-fetch
  via `AddressGet`, publish `EventTypeContactUpdated`) is unchanged —
  this design changes only what happens inside the transaction, not that
  response path. For `AddressCreate` and `AddressUpdate` (new-target
  side): this is NOT a skew state at all — it is today's ordinary
  "you already own this target" duplicate, and the row's presence means
  the INSERT/UPDATE would hit the unique index anyway; return
  `ErrDuplicateTarget`/409 exactly as today (§2's status-code promise),
  rolling back.
- **Row missing (self-orphan): repair-in-place.** The open period is this
  contact's own and correct — keep it (do NOT fabricate anything). For
  `AddressCreate`: proceed with the row INSERT and commit — the API
  contract (row exists after success) is restored, no new period is
  opened. For `AddressUpdate` (new-target side, round-30: this third
  step-procedure caller was previously unenumerated): proceed with its
  own UPDATE re-targeting its existing row to this target and commit —
  same repair-in-place semantics, no INSERT needed. For `ClaimAddress`
  (round-24's reoccupation, same-owner variant: an unresolved row now
  occupies the slot): claim the row as usual (its final UPDATE targets
  the NULL-owned row and succeeds), keep the existing open period,
  commit. **Round-30 disposition (gap attribution, recorded as
  accepted):** keeping the open period means interactions from the
  skew-deletion moment to this re-registration remain attributed to this
  contact — acceptable because the orphan is the contact's own (the
  deletion that failed to close it was an old-binary artifact, and the
  contact is demonstrably re-claiming the same target now); the
  resulting arrival-order asymmetry (a third party arriving first would
  close the period at NOW(), truncating that gap) is inherent to
  repair-on-next-touch designs and shares the bounded-window standard of
  §9's other skew dispositions.
- **Row exists but disagrees (NULL-owned or another owner):** for
  `ClaimAddress` claiming that very row, this is the repair-in-place case
  above; for `AddressCreate` this cannot coexist with its own INSERT
  succeeding — the unique index makes the insert fail first, routing
  through the round-28 duplicate-key path instead.
**Round-29 reachability correction:** the previous text claimed this case
"remains genuinely unreachable for `AddressCreate`" — that was true only
while the unique index always held a row; §9's round-23(c) skew state
(row hard-deleted, period left open) makes `AddressCreate` genuinely
arrive here for its own orphan, which is exactly the row-missing
repair-in-place branch above.

**Step 3 — does this contact_id have any closed row(s) of its own?** (Only
reached once steps 1–2 confirm no open row exists anywhere for this target —
so any row found here is guaranteed closed, removing the ambiguity that let
the old case 3 fire on a target that actually had an open row.) If yes, take
the one with the latest `valid_to` (`ORDER BY valid_to DESC` over the
already-fetched set, no extra query). Then check the same fetched set for
any *other* contact_id's row with `valid_from >= that closed row's valid_to`
(**round-19 finding: this comparison must be inclusive, `>=` not `>`** — a
prior version of this text said "strictly later," leaving the tie-break
operator to the implementer's judgment. If A's closed row's `valid_to`
exactly equals B's row's `valid_from` — an exact-timestamp tie, unlikely in
production but easy to hit with a mocked/frozen clock in a unit test — a
`>` reading finds "no intervening owner," reopens A's row unbounded, and
that reopened period now overlaps the timestamp B's row already legitimately
covers, reproducing defect #2 (history leaking across owners) through a
boundary condition rather than a missing branch. `>=` treats the tie as an
intervening owner, correctly routing to the INSERT branch below instead of
reopening) — and because step 1 already established there is no
currently-open row for any other contact_id, any such intervening row is
now guaranteed closed too, so this comparison can never rediscover the
live-owner conflict step 1 already exhausted:
  - **None found** → this contact held the target continuously up to close,
    nobody intervened. Re-open: `UPDATE ... SET valid_to=NULL WHERE id=<that
    row>` (no new row — one continuous history).
  - **Found** (the A→B→A case round-4 found broken) → do **not** reopen the
    stale row. INSERT a new row instead: `contact_id`=this contact,
    `valid_from=NOW()`, `valid_to=NULL`.

**Step 4 — does any row exist for this target at all (any contact_id, any
status)?** (Only reached once steps 1–3 rule out an open row for anyone and
a closed row for this contact_id — so any row found here necessarily
belongs to a *different* contact_id, and is closed.) If yes: reassignment
(A→B, or first-time `ClaimAddress` of a target someone else previously
held, where A's row was already closed by A's own `AddressDelete` per §7's
TOCTOU sequencing) — round-5's fix, the case the original "no rows at all"
framing misclassified as first-ever registration. INSERT: `contact_id`=this
contact, `valid_to=NULL`, and **`valid_from` is caller-specific
(round-37, bound corrected in round-38, scope note added round-39):** for
`AddressCreate`/`AddressUpdate`, `valid_from=NOW()` (not NULL — this
contact never owned the target before now). **For `ClaimAddress`,
`valid_from = latest closed valid_to for this target`** (the round-38
correction: the round-37 bound `GREATEST(latest valid_to, row
tm_create)` was doubly wrong — (a) it left the pre-`tm_create` gap
(previous era's end → row creation) outside every period, so those
interactions resurfaced in the unresolved queue AND dropped off the
claimer's timeline, whereas today's time-agnostic value-match attributes
them to the claimer; and (b) via round-28 repair (a), which resets a
dead owner's row to `contact_id = NULL` mid-transaction, `tm_create` is
the DEAD contact's registration time, not an unresolved-era start,
making the bound semantically meaningless on that path. The corrected
bound reproduces today's observable attribution for the SINGLE gap
immediately preceding the claim: a claim attributes the unowned span
since the immediately-prior era to the claimer — which is precisely what
today's value matching does for that gap — with no-overlap guaranteed
trivially (`valid_from = latest valid_to`, half-open disjoint), and the
same-instant tie resolving to the claimer by the half-open rule while
Step 3's `>=` still classifies the claimer as an intervening owner for
any later re-registration. `NOW()` here would orphan exactly the
interactions the claim was performed to attach — they would leave the
queue's suppression (row no longer NULL-owned) yet fall outside every
period: unattributed forever, resurfacing in the unresolved queue. The
same caller split applies to Step 3's INSERT arm (the intervening-owner
case) when reached by `ClaimAddress` — there, "latest closed valid_to
for this target" resolves to the intervening owner's `valid_to`
structurally (any closed row's `valid_to >= valid_from >=` the prior
owner's `valid_to`, so the target-wide max is always the most recent
era's end, never an earlier one of this contact's own prior eras; stated
explicitly here per round-39 to foreclose the reading "this contact's
own closed period," which would overlap the intervening owner's era).
One skew-scope note: when a Step-1 orphan close at `NOW()` has just
pushed latest `valid_to` to `NOW()` (mixed-skew, rounds 23/31), the
claim's `valid_from = NOW()` and the pre-close era stays attributed to
the orphan's owner up to the close — the round-23 "attribution up to the
repair point" standard, not a new gap (queue non-resurfacing still
holds: every instant is covered by either the closed orphan period or
the claimer's new period).

**Round-39 disposition (recorded limitation, not fixed further): the
bound covers only the gap immediately preceding the claim, not every
earlier gap in the target's full history.** With multiple prior eras
(A owned, then vacant, then B owned, then vacant, then C claims an
unresolved row), `valid_from = latest closed valid_to` starts at B's
`valid_to` — the A-to-B gap stays uncovered by any period. Today's
row-presence suppression covers that gap unconditionally (time-agnostic
existence); after the claim converts the row, only the periods do, and
the A-to-B gap resurfaces in the unresolved queue with no owner. This is
accepted, not fixed: reaching further back (e.g. to the earliest gap, or
to `NULL`) would re-admit round-30's rejected over-absorption (a claim
should not retroactively absorb eras this design has already
positively closed for OTHER owners' history), and the round-23
"attribution up to the repair point" standard already treats
un-owned inter-era gaps as legitimately surfaced, un-suppressed
territory once something disturbs the row that had been suppressing
them. §10's fixture set gains a case with two prior eras asserting
the older gap resurfaces (documented, not a regression from any
in-scope baseline) while the immediately-prior gap does not.

**Step 5 — no rows matched any of the above.** True first-ever registration.
INSERT: `valid_from=NULL` (unbounded past — see §9 backfill), `valid_to=NULL`.

**The target's `(type, target)` values, needed to know which row(s) to lock,
must themselves come from data already covered by this transaction's locking
discipline (round-7 finding, restated for the step-ordered form) — not from
a separate non-locking `AddressGet` preceding the lock.** For
`AddressCreate`, the caller supplies `(type, target)` directly as request
parameters, so there is nothing to look up. For `AddressClaim`, which
locates the target address row by `addressID` first: **round-8 review found
the earlier claim that this resolution was already protected by lock
ordering alone was wrong** — a plain, non-locking `AddressGet(addressID)`
executed before this transaction's `FOR UPDATE` can read a `target` value
that a concurrent `AddressUpdate` (target-change) then commits over before
this transaction's lock is acquired, leaving `AddressClaim` holding a lock
on — and writing an ownership period for — the *stale* target while the
`contact_addresses` row it ultimately updates has already moved to the new
one. Lock ordering serializes *who goes first* on a given target; it does
nothing to guarantee the target value read outside any lock is still
current by the time the lock is taken. **Fix: `AddressClaim` re-reads
`contact_addresses.target` for this `addressID` a second time, inside the
same transaction, immediately after acquiring the §4 locking read (using
that just-locked `(type, target)` to know what it locked) — and if this
second read's `target` differs from the first (pre-lock) read that
determined which `(type, target)` to lock, aborts the transaction and
retries the whole operation from the top with the freshly-read target.**
This is a compare-and-retry, not a new lock: the retry loop reuses §5.3's
existing `maxDeadlockRetries` machinery (a stale-target mismatch is treated
as the same class of transient, retry-eligible condition as a deadlock,
capped the same way) rather than introducing a second retry mechanism.

**Round-39 addition — the same by-addressID resolution problem, and the
same fix, apply to two more callers the rule previously enumerated only
for `AddressClaim`:**

- **`AddressUpdate` (target change)**: `contacthandler.UpdateAddress`
  reads the row's current `target` (`contact.go:354`) before the
  transaction to know the OLD `(type, target)` to close a period for.
  This is exactly the same stale-read hazard as `AddressClaim`'s — a
  concurrent `AddressUpdate`/`RemoveAddress`/`AddressClaim` on the same
  row can move or delete it between that read and this transaction's
  lock. `AddressUpdate` gets the identical compare-and-retry: re-read
  `contact_addresses.target` for this `addressID` immediately after
  the §4 locking read; on a mismatch, **abort the transaction entirely
  and retry the whole operation from the top with the freshly-read
  old-target (round-40 clarification: this is the same full-transaction
  restart `AddressClaim` performs, not a partial retry of only the
  old-target lock)** — `AddressUpdate` is a two-target transaction under
  §5.2's ascending-`(type,target)`-order rule, so a partial retry that
  kept the already-acquired new-target lock while re-acquiring a
  different old-target lock would reintroduce the exact two-target
  reverse-order deadlock class §5.2 exists to prevent; a full restart
  re-establishes the ascending order from scratch with the fresh
  old-target and is safe by the same argument as `AddressClaim`'s;
  on NotFound, abort with `dbhandler.ErrNotFound` (no
  retry — permanent, same as `AddressClaim`'s rule).
- **`AddressDelete`**: needs `(type, target)` from the row itself (the
  dbhandler entry point receives only `addressID`) to compute
  `open_period_uk` and close the right period. Same fix: locking read
  first, re-read after, compare-and-retry on mismatch (full-transaction
  restart, single-target so no ordering concern), abort on
  NotFound.

Both new retry points join `AddressClaim`'s in §5.3's
`maxDeadlockRetries` cap (round-40: §5.3's round-8 text named only
`AddressClaim` — the cap is stated once, generically, as covering every
stale-target-mismatch retry in this document, not per-caller).

Without this extension, a concurrent re-target (Tx1: A→B commits) racing a
stale-read `AddressDelete`/`AddressUpdate` (Tx2: reads old target A,
locks and closes A's period after Tx1 already moved the row to B) closes
the wrong (already-closed) period and leaves B's live open period
orphaned under a live Contact — the exact degraded state §9 designates
as skew-only, now producible by same-binary concurrency alone. Step
1/2's self-healing repairs it on the next touch, but the transaction's
own within-request correctness (§2's promise) is violated in the
interim. This is the same enumeration-gap failure mode round-30 named
for `AddressUpdate`'s omission elsewhere in this document.

**Round-40 note on B5's `RowsAffected == 0` guard:** the compare-and-retry
above does not make that guard dead code. Against another new-binary
transaction it is effectively unreachable (§5.2's `FOR UPDATE` on the
freshly-confirmed period serializes same-binary contenders), but during
a rolling deploy an old-binary pod bypasses the ownership-period locking
discipline entirely and can delete the row between this transaction's
post-lock re-read and its final UPDATE/DELETE — so the guard still
covers that mixed-version window. The retry logic's coverage
(same-binary staleness) and the guard's coverage (any-version deletion
after the re-read) are deliberately non-overlapping layers, not
redundant ones.

**Round-9 finding: the compare-and-retry above only defined behavior for
"target differs" — it left the post-lock re-read returning NotFound entirely
unhandled.** `contact_addresses` is hard-delete (§1), so a concurrent
`RemoveAddress` on the same `addressID` can delete the row between the
pre-lock read and the post-lock re-read. Unlike a target mismatch (transient
— retrying with the fresh target can still succeed), a NotFound here is
**permanent** — the address this `ClaimAddress` call was trying to claim no
longer exists, and no amount of retrying changes that. **Fix: if the
post-lock re-read returns NotFound, do not retry — abort the transaction and
return `dbhandler.ErrNotFound`** (a plain sentinel error; the `dbhandler`
package has no `cerrors` dependency, so this stays purely at the dbhandler
layer), **the same sentinel `AddressClaim`'s existing pre-lock NotFound case
already returns.** No new mapping code is needed: `contacthandler.ClaimAddress`
already converts `dbhandler.ErrNotFound → cerrors.NotFound` via its existing
`stderrors.Is(err, dbhandler.ErrNotFound)` branch (`contact.go:465-471`,
§5.4's convention) for that pre-lock case, and this post-lock case reuses the
exact same branch — round-9's original wording ("return `cerrors.NotFound`")
incorrectly implied the dbhandler layer itself performs that conversion,
which round-10 review found crosses the package boundary §5.4 itself
establishes. Feeding this into the
retry loop instead would misclassify a permanent condition as transient,
burning all `maxDeadlockRetries` attempts before incorrectly surfacing a
transient 5xx for something that was never going to succeed — the same class
of error-mapping mistake §5.4 fixed for B5.

| Existing operation | Ownership-period effect |
|---|---|
| `AddressCreate` / `ClaimAddress` | Steps 1–5 above, decided from the single locked read |
| `AddressDelete` | Close the open period for this contact_id+type+target: `UPDATE ... SET valid_to=NOW() WHERE open_period_uk = <hash>`. The period row is never deleted. |
| `AddressUpdate` (target field changed) | Two locking reads required (§5.2 specifies the required order between them) — one for the old target, one for the new: close the old-target period (`valid_to=NOW()`), then apply steps 1–5 above to the new target |
| `AddressUpdate` (target field NOT changed — e.g. only `name`/`detail`/`is_primary`) | **No ownership-period effect.** The write transaction (§5.1) still performs the locking read above for this target as part of its fixed lock-ordering rule, but performs no INSERT/UPDATE against `contact_address_ownership_periods` — the lock exists purely to keep this operation inside the same serialization order as every other address write on that target, not because there is a period decision to make. |
| Contact soft-delete (`ContactDelete`) | Close **every** open period owned by that contact_id (prevents an orphaned open period after the owning Contact itself is gone) — see round-13 addition below for the transaction/lock discipline this requires. **Round-25 addition (membership corrected in round-26): the cleanup also hard-deletes every live `contact_addresses` row owned by this contact_id.** Leaving rows live under a tombstoned Contact would recreate, on every future deletion, the exact permanent-lockout state round-23's backfill cleanup fixed for pre-migration data (the row blocks `AddressCreate` via the unique index and `ClaimAddress` via the non-Nil pre-check, while the A9-b guard blocks the only recovery path); deleting them is what a timely `RemoveAddress` would have produced, and the closed periods preserve the attribution history. **Round-26 correction #1 (membership): the row-cleanup target set is driven by `contact_addresses WHERE contact_id = ?` (i.e. `AddressListByContactID`), NOT by the open-period membership set** — a skew-created row with no period (§9's accepted degraded state) would be invisible to period-driven membership and would survive under the tombstone, reproducing the lockout in a third population; the period-closure loop keeps its own open-period membership, and each row-delete joins the per-target transaction for its target when one exists, or runs as its own §5.1-style transaction when no period exists. **Round-26 correction #2 (event payload): `contacthandler.Delete` snapshots the Contact including its address list BEFORE `dbhandler.ContactDelete` runs, and uses that snapshot for the `EventTypeContactDeleted` payload and the RPC response** — today's payload includes the deleted Contact's full `addresses` (the by-id `ContactGet` re-read is intentionally unfiltered for exactly this purpose, per the service CLAUDE.md), and without the snapshot the round-25 cleanup would silently strip `addresses` from the event/API response (an external contract change; `Test_Delete`'s payload expectations are added to §10's checklist) |
| Customer cascade delete (`ContactDeleteByCustomerID`, via `EventCustomerDeleted` — round-20 finding: this fifth deletion entry point was entirely missing from this design) | Same per-contact period closure **and round-25 address-row cleanup** as `ContactDelete`, applied to every Contact the bulk delete touches — see round-20 addition below |

**Round-13 finding: `ContactDelete`'s "close every open period" effect above
had no transaction, lock-ordering, or deadlock-retry discipline specified,
unlike every other row in this table.** Because a Contact can hold multiple
addresses, this operation locks multiple targets in one transaction — the
same shape as `AddressUpdate`'s two-target case (§5.2), just with an
arbitrary N instead of a fixed 2, and it is exposed to exactly the same
deadlock class: a concurrent `AddressUpdate` on one of this Contact's
targets could acquire locks in a conflicting order. **Fix, reusing §5.2's
existing rule without introducing a new one:** `ContactDelete`'s ownership-
period closure runs inside its own `BeginTx`, first `SELECT`s the full set of
this contact_id's currently-open periods (ordinary read, no lock needed yet
— membership, not values, is all that's needed at this step), then acquires
the §4 locking read for **each** of those targets in the same ascending
`(type, target)` order §5.2 already mandates for the two-target
`AddressUpdate` case (a straightforward generalization from N=2 to arbitrary
N, not a new ordering rule), closing each one in that order before
committing. This transaction is subject to the same `maxDeadlockRetries = 3`
retry loop (§5.3) as every other write path in this table.

**Round-14 finding: the membership `SELECT` above only locks the targets it
finds — a `ClaimAddress` for this same contact_id that commits a brand-new
open period *after* that `SELECT` but *before* this transaction commits is
invisible to it, so that period is never locked and never closed, an orphan
open period surviving under a Contact that no longer exists. This is not
covered by §7's accepted TOCTOU (which only discusses the A→B reassignment
delete/create gap) — it is a genuinely missed race on this operation's own
membership read, the same "plain SELECT is a stale snapshot" class of bug
rounds 5/8/9 already fixed elsewhere in this document, just not yet applied
here.** **Fix:** after acquiring the §4 locking read (and thus the row lock)
for every target from the initial membership `SELECT`, re-run that same
membership `SELECT` once more, still inside the same transaction. Because
every target this operation cares about is now individually locked, any
period that existed at the time of the *first* `SELECT` is guaranteed stable
by the second read; the only way the second read can show a *new* target
absent from the first is a `ClaimAddress` that committed in the gap between
them. If the second read finds no new targets, proceed to close and commit
as already described. If it finds one or more new targets, acquire the §4
locking read for each of them too (still following the same global ascending
`(type, target)` order — extending the already-sorted lock list, not
restarting it), and repeat this membership re-check until a pass finds
nothing new. In practice this converges in at most one extra pass per
concurrent `ClaimAddress` that actually raced the deletion, and the same
`maxDeadlockRetries = 3` bound (§5.3) still caps the whole transaction if
convergence itself deadlocks against something else. **Round-15 note (minor,
recorded not blocking): this re-check loop's pass count has no explicit cap.**
A sustained stream of `ClaimAddress` calls against new targets for this same
contact_id could keep the loop finding "one more new target" across many
passes, holding this transaction open longer than typical. This is a
liveness/latency concern, not a correctness one (each pass still converges
towards a fixed point), and is recorded here as an operational note rather
than fixed with a new cap, since introducing an arbitrary pass limit would
trade a rare long-transaction case for a new correctness question (what
happens to a `ContactDelete` that hits the cap without converging?) that
does not have an existing precedent in this document to reuse, unlike
`maxDeadlockRetries`.

**Round-19 finding: this re-check loop's "extending the already-sorted lock
list" claim is weaker than §5.2's "fixed total order eliminates this
deadlock class structurally" standard, and this document should say so
plainly rather than let the two claims sit side by side unreconciled.**
Concretely: `ContactDelete` (Tx1) locks its initial open-period set {B, D}
(B<D) in order; a concurrent `AddressUpdate` (Tx5, unrelated contact) then
commits a target swap that hands this same contact_id a brand-new open
period at target A (A<B); Tx1's re-check discovers A and requests it *after*
already holding B and D — out of ascending order relative to a *different*
concurrent `AddressUpdate` (Tx6) that legitimately acquires A before D in
ascending order and blocks waiting for D, which Tx1 holds. Tx1 now waits on
Tx6 for A while Tx6 waits on Tx1 for D: the exact reverse-order deadlock
class §5.2 was built to make structurally impossible, reintroduced here
because the re-check's late-arriving lock isn't actually re-sorted relative
to the ones already held before it. This is not a new locking bug to fix —
`maxDeadlockRetries` (§5.3) already covers it, the same way it covers every
other deadlock this document accepts as a retry-bounded cost rather than a
structurally-eliminated one — but claiming this re-check "follows the same
global ascending order" overstated what it actually guarantees. **This
paragraph corrects that:** the re-check loop is a `maxDeadlockRetries`-bounded
mitigation, not a fixed-total-order guarantee; §5.2's "structurally
impossible" framing applies to a single operation's initial lock acquisition,
not to locks a re-check loop discovers and adds afterward. In a target
rotated unusually frequently (the same call-center-number scenario §6.2's
OR-clause-growth caveat already names), this could plausibly raise
retry-exhaustion frequency for `ContactDelete` above other operations in this
table — recorded here as a known, accepted characteristic rather than solved,
consistent with how round-15's uncapped-pass-count note above was handled.

**Round-20 finding: a fifth deletion entry point, `ContactDeleteByCustomerID`
(`pkg/dbhandler/contact.go:437-491`), was entirely absent from this design.**
It is invoked by `contacthandler.EventCustomerDeleted` (`event.go:33-48`),
which `subscribehandler` fires when `bin-customer-manager` publishes a
`customer_deleted` event — a real production path, not an admin edge case —
and it soft-deletes **every** active Contact of that customer in one bulk
`UPDATE ... WHERE customer_id = ?`, touching neither `contact_addresses` nor
(under this design, if left unmodified) `contact_address_ownership_periods`.
Left as-is, a customer deletion would orphan the open ownership periods of
every Contact that customer had — the exact defect class round-13/14 closed
for single-contact `ContactDelete`, reproduced wholesale through an entry
point this document simply didn't know existed (the same "the declared
handler set itself was incomplete" failure mode as round-11's `ContactCreate`
discovery; round-12's caller sweep grepped `dbhandler.Address*` callers but
never swept "all paths that set `contact_contacts.tm_delete`").
**Fix:** `ContactDeleteByCustomerID` performs the same per-contact ownership-
period closure as `ContactDelete` — run the same close-every-open-period
transaction specified in the round-13/14 paragraphs above, per contact,
sequentially, each in its own `BeginTx` with the same ascending lock order,
membership re-check, and `maxDeadlockRetries` bound.

**Round-21 correction: the contact-id set for that per-contact loop must be
collected AFTER the bulk soft-delete `UPDATE`, not reused from the SELECT
that precedes it.** Round-20's original text said the function "already
knows the affected contact_id set" from its pre-delete SELECT
(`contact.go:437-491`'s id fetch for cache invalidation) — but that SELECT
and the bulk `UPDATE` are two separate non-transactional statements, and
the `UPDATE` carries no `tm_delete IS NULL` filter: a Contact created (with
addresses and open periods) in the gap between the SELECT and the UPDATE
gets soft-deleted by the UPDATE while being absent from the pre-collected id
set — the period-closure loop would never visit it, leaving a permanently
orphaned open period. Worse, "re-drive the event to recover" does not work
for this case: the pre-delete SELECT filters on `tm_delete IS NULL`, so a
re-driven event still cannot see the already-deleted Contact. This is the
same stale-membership-read defect class round-14 closed for target
membership inside `ContactDelete`, resurfacing at the contact-set level.
The fix is ordering, not locking: run the bulk `UPDATE ... SET tm_delete=ts
WHERE customer_id=?` first, then collect the loop's id set with
`SELECT id ... WHERE customer_id=? AND tm_delete = ts` (the exact timestamp
just written), and drive both the period-closure loop and the existing
cache-invalidation loop from that post-delete set. This also fixes the
pre-existing cache-invalidation variant of the same race (a Contact missed
by the stale id set kept a stale cache entry), at no extra cost.
Per-contact transactions (not one giant transaction over every Contact of
the customer) keep each transaction's lock footprint identical to the
already-analyzed single-`ContactDelete` case, avoid introducing a new
"lock N contacts × M targets in one transaction" ordering question this
document would otherwise have to re-derive, and are consistent with the
bulk path's existing per-row cache invalidation loop. A crash mid-loop
leaves some contacts processed and some not — acceptable because the
operation is idempotent (closing an already-closed period is the same
`RowsAffected == 0` no-op §9's round-16/17 instrumentation already
handles) and, with the round-21 ordering above, a re-driven event CAN now
find every affected Contact (`tm_delete IS NOT NULL` rows are re-selectable
by timestamp), making re-drive a genuine recovery path rather than the
false one round-20 originally claimed. (Round-22 note: this recovery
property depends on the bulk `UPDATE` carrying **no** `tm_delete IS NULL`
filter — the unfiltered form re-stamps already-deleted rows with the new
timestamp so the post-delete SELECT can find them; adding that seemingly
natural filter would silently break re-drive. Recorded so no future
cleanup "optimizes" it away.)

**Round-27 corrections (three, all consequences of round-25/26's cleanup
additions):**

1. **Event-payload snapshot must overlay the deletion timestamps.** The
   round-26 pre-delete snapshot, used verbatim, would publish
   `tm_delete: null` (the field is not `omitempty`) and a stale
   `tm_update` — the same unrecorded-external-contract-change class
   round-26 fixed for `addresses`. Rule: `contacthandler.Delete` takes the
   pre-delete snapshot for `addresses`, then overlays `tm_delete`/`tm_update`
   with the deletion timestamp (the same `ts` the `UPDATE` wrote) before
   publishing the event / returning the RPC response, so downstream
   consumers keep both the address list AND the deletion time. §10's
   `Test_Delete` item covers both fields.
2. **The row-cleanup membership joins the round-14 re-check loop.** The
   round-26 row membership (`AddressListByContactID`) was specified as a
   single read; a guard-passing `AddAddress` committing between that read
   and the tombstone commit would leave its row uncleaned (closed period +
   live row = the lockout's fourth population). Rule: each pass of the
   round-14 re-check loop re-reads BOTH memberships (open periods AND
   live address rows) under lock and processes anything new, converging
   exactly as round-14 argued; the loop terminates when a pass finds no
   new members in either set. The residual window round-14 already
   accepted (an address transaction committing entirely after
   `ContactDelete` commits — the A9-b guard's non-locking pre-check
   passed before the tombstone landed) remains accepted, now with a
   working escape hatch: the resulting state (live row + open period
   under a tombstoned owner) is exactly what the round-27 Step-1 clause
   above detects and repairs on the next registration attempt.
3. **Terminology fixed: there is ONE transaction, not per-target
   transactions.** Round-13 defined `ContactDelete`'s closure as a single
   `BeginTx` locking N targets in ascending order; later rounds' phrase
   "per-target transaction" muddied that. Canonical reading everywhere in
   this document: `ContactDelete` runs ONE §5.1 transaction that locks all
   discovered targets ascending (per §5.2's N-way rule), closes their
   periods, deletes all the Contact's address rows, soft-deletes the
   Contact, and commits atomically — so there is no partial-failure state
   between period closure and row cleanup, and a deadlock retry restarts
   the whole discovery+lock+mutate cycle idempotently.

**Round-27 skew corrections, repaired in round-28 (the deletion entry
points are themselves a skew surface — §9's analysis covered only
address-write skew):** an old-binary `ContactDelete`/`EventCustomerDeleted`
during the rollout window tombstones the Contact but leaves rows live and
periods OPEN (the old binary knows nothing of periods or cleanup). The
Step-1 tombstoned-owner clause above self-heals this on the next
registration attempt. **Round-28 correction to the repair actions (the
round-27 versions were broken):**

- **Late cleanup always writes the closed period FIRST, exactly like §9's
  backfill.** Round-27's "hard-delete the row" alone would have destroyed
  the only evidence of prior ownership: on the period-less variant the
  retry would then hit Step 5 (no rows at all → `valid_from=NULL`) and the
  new owner would absorb the dead Contact's entire era — defect #2
  manufactured by the cleanup itself, and inconsistent with §9's backfill
  which handles the identical data state by inserting a closed period
  BEFORE deleting the row. **Round-29 correction to the fabricated
  period's bounds:** §9's backfill may use `valid_from=NULL` only because
  at migration time the periods table is empty for every target (max one
  source row, per the round-23 uniqueness note); at post-migration
  runtime other contacts' legitimate closed periods can already exist for
  the same target, and a fabricated `valid_from=NULL` row can never
  satisfy Step 3's `valid_from >= closed.valid_to` intervening-owner test
  — it would be invisible to Step 3 and let a previous owner's period
  reopen across the dead owner's era (defect #2 via Step 3). Rule
  (round-30 refinement — the round-29 bounds over-attributed unowned gap
  time to the dead Contact): the fabricated period's `valid_from` =
  **`GREATEST(latest existing valid_to for this target, stale row's
  `tm_create`)`** — the stale `contact_addresses` row is already read
  inside every late-cleanup transaction, and its `tm_create` is the
  actual registration moment, a strictly more accurate lower bound than
  the previous owner's `valid_to` alone; since `tm_create >= latest
  valid_to` in every reachable ordering that matters, Step 3's `>=`
  visibility argument is preserved, and the gap between the previous
  owner's era and the dead Contact's real registration is no longer
  swallowed into the fabricated era (head-gap disposition: with the
  `tm_create` bound the head gap collapses to zero by construction).
  `valid_from` may be `NULL` only when the target has no periods at all
  AND the stale row's `tm_create` is itself NULL (not producible by
  today's write paths — defensive default only). `valid_to` = the
  owner's tombstone timestamp. If the computed `valid_from` would exceed
  `valid_to` — **stated generally (round-31): whenever
  `GREATEST(latest valid_to, tm_create) > tm_delete`, through EITHER arm;
  `tm_create > tm_delete` (the old-binary A9-b case) is only one cause,
  and a mixed-skew cleanup that just closed another owner's orphan at
  NOW() can push `latest valid_to` past a `tm_delete` even when
  `tm_create < tm_delete`, so implementations must test the computed
  bound, not the A9-b special case** — insert a zero-length period
  `[tm_create, tm_create)` instead (round-15's established zero-length
  disposition; anchored at `tm_create` rather than `tm_delete` so no
  instant that predates the row's actual existence is ever claimed). **Round-29 covering predicate
  (the "skip if already covered" test, previously undefined):** skip the
  INSERT iff the already-fetched period set contains a closed period for
  the SAME tombstoned contact_id whose `valid_to >= its
  contact_contacts.tm_delete` — i.e. that owner's final era is already
  recorded through its death; an older, earlier closed era for the same
  owner does NOT count as covering. This is evaluable entirely from the
  §4 locked fetch plus the tombstone read, no extra queries.
- **Round-29 ordering rule (Step-1 clause vs repair (a) previously
  disagreed), refined in round-31:** in every late-cleanup site the
  canonical order is: (1) close the orphan open period if one exists
  (`valid_to = NOW()`); (2) evaluate the covering predicate **for the
  tombstoned row-owner** and INSERT the fabricated period only if it
  fails; (3) the caller-specific row repair last. **Round-31 correction:
  the round-29 claim that closing the orphan "necessarily satisfies the
  covering predicate" (making step 2 a guaranteed skip) is FALSE in the
  mixed-skew state where the orphan period's owner (B) differs from the
  stale row's tombstoned owner (D)** — closing B's orphan records B's
  era, not D's, and the predicate (same tombstoned contact_id) correctly
  fails for D, so BOTH the close and the fabrication run in one cleanup.
  The predicate must therefore always be evaluated after the close,
  never skipped on the strength of the close having happened. The
  no-overlapping-periods property still holds: the closed orphan (B's)
  and the fabricated period (D's, whose `valid_from` = GREATEST(latest
  valid_to incl. the just-closed orphan, tm_create) starts at or after
  that close) cover disjoint ranges by construction.
- **(a) `ClaimAddress`: the stale row is NOT hard-deleted — its
  `contact_id` is reset to NULL (re-unresolved) instead.** Round-27's
  hard-delete would have destroyed the very row the claim's own final
  UPDATE targets (`UPDATE ... WHERE id=? AND contact_id IS NULL`,
  `RowsAffected==0 → ErrConflict` — the B5 guard this design preserves),
  making the claim always fail 409 and leaving the caller's addressID a
  404 on retry. Correct repair inside the claim's transaction (round-29:
  in the canonical late-cleanup order above): close any orphan open
  period first, insert the fabricated period only if the covering
  predicate fails (period-less variant), then `UPDATE contact_addresses
  SET contact_id = NULL WHERE id = ?`, then proceed — the claim's final
  UPDATE now finds exactly the NULL-owned row its race guard expects, and
  the step procedure sees a closed-period history (Step 4).
- **(b) `AddressCreate` on a duplicate-key error** checks (in a §5.1
  transaction) whether the conflicting row's owner is tombstoned — if so
  it performs the late cleanup (closed period first, then row delete —
  here the hard-delete is correct because the caller is inserting a NEW
  row) and retries the insert once; otherwise the duplicate error
  propagates as today. If the cleanup transaction finds the row already
  gone (a concurrent repair won), it skips straight to the retry — two
  concurrent repairs serialize on the row lock, one wins, the other's
  retry receives the legitimate duplicate-key error from the winner's new
  row; the single-retry cap rules out livelock.
- **Lock-order rule for the tombstone read (round-28): `contact_contacts`
  is read LAST, after the period and address locks, in every address-write
  transaction** — matching `ContactDelete`'s single transaction, which
  touches periods → address rows → `contact_contacts` UPDATE last. An
  implementer who instead read/locked `contact_contacts` first would
  create an AB-BA surface against `ContactDelete`; §5.2's ordering rule
  now names `contact_contacts` explicitly as the final table in the
  order. A plain (non-locking) read is sufficient: a tombstone can only
  appear, never disappear (no undelete path exists — verified round-21's
  entry-point sweep), so a stale "not tombstoned" read merely means this
  request behaves exactly as it would have moments earlier, and the
  states it would have repaired remain repairable by the next attempt.

Together these cover the period-less variant (an old-binary A9-b
`AddAddress` onto an already-tombstoned Contact during the window), which
never reaches Step 1 because no period exists — the unique-index path (b)
is precisely where that state surfaces. All Step-1/duplicate-key-path
repairs increment the same skew-orphan Prometheus counter (post-commit
only, per Step 1's round-32/33 rule; round-33 scope note: incidental
hash-close healings by a live occupant's own delete/re-target are
in-band ordinary closes and intentionally uncounted).

**Round-28 bundled bug fix (index-name drift — production trigger for
repair (b)):** `dbhandler`'s duplicate-key classifier
(`address.go:156-159`) matches the literal string
`idx_contact_addresses_cust_type_target`, but the production index (per
Alembic, the sole schema authority) is named
`idx_contact_addresses_identifier` — the classifier never matches on
production MySQL (only the SQLite test schema uses the old name, so tests
pass while production falls through to a generic 500 instead of
`ErrDuplicateTarget`/409). Repair (b)'s trigger — and today's documented
409 behavior — depend on that classification, so this design bundles the
fix (same standard as the A9-b/B5 bundled fixes, §2): correct the match
string to `idx_contact_addresses_identifier` (and the sibling
`cust_primary` → `idx_contact_addresses_primary`), and align the SQLite
test schema's index names with production so the drift cannot silently
recur. **Round-30 extension: the 1062 classifier exists only in
`AddressCreate`'s error path today — `AddressUpdate`'s Exec wraps all
errors generically (`address.go:382-384`), so the period-less
tombstoned-owner variant occupying an `AddressUpdate` new-target slot
would surface as a generic 500 with no repair trigger. The same
classification (errno 1062 + production index name →
`ErrDuplicateTarget`) is added to `AddressUpdate`'s target-change path,
and its duplicate-key handling gains the same
tombstoned-owner-check-then-retry-once behavior as repair (b)** — the §4
table already routes `AddressUpdate` (target changed) through steps 1–5
as a first-class caller, so its repair surface must match.

**A9-b guard placement (round-3 finding: this was only in §8's summary table,
never specified in this section — fixed here).** All four write handlers this
table governs — `AddAddress`, `UpdateAddress`, `RemoveAddress`, `ClaimAddress`
(`pkg/contacthandler/contact.go`) — gain an explicit `c.TMDelete != nil` check
immediately after their existing `ContactGet` call and before any
`contact_addresses`/ownership-period write, mirroring the check
`interactionListByContact` already performs
(`pkg/contacthandler/interaction_read.go:117-125`). (`ContactCreate` is
deliberately not a fifth handler here — it creates a new Contact, so there
is no pre-existing `TMDelete` state to check; it *is*, however, in scope for
the round-11 fifth-caller fix above, which is a separate concern from this
guard.) On a soft-deleted Contact,
each handler returns `cerrors.NotFound` (the same "treat as not found" shape
`interactionListByContact` and `ClaimAddress`'s existing cross-tenant check use
— `contact.go:449-455`), so a caller cannot distinguish "Contact never existed"
from "Contact was deleted," consistent with the platform's existing
not-found-not-forbidden convention. This check runs before the transaction in
§5.1 opens (it is a precondition, not part of the ownership-period write
itself) — so a request against a soft-deleted Contact never acquires the
`FOR UPDATE` lock at all.

**Reassignment (A→B) is explicitly the composition of `AddressDelete`(A) +
`AddressCreate`/`ClaimAddress`(B) as two separate calls — not a new operation.**
Round-1 review confirmed `ClaimAddress` unconditionally rejects (`ErrConflict`)
any address that already resolves to a live owner
(`pkg/dbhandler/address.go:182-218`), so "reassign while B's period is still open"
is not a reachable code path; A's row must be deleted first. The TOCTOU gap
between those two calls is accepted, not solved (§7) — but Step 4 above
(`valid_from=NOW()` for B, not NULL) is a hard requirement independent
of that gap, and is what actually prevents history leaking from A to B once
B's `AddressCreate` does land.

## 5. Transaction and concurrency strategy

Round-1 and round-2 review both confirmed `bin-contact-manager` is served by a
**shared, multi-worker RabbitMQ queue** (`bin-manager.contact-manager.request`,
10 workers, `pkg/listenhandler/main.go`), consumed across pods — an in-process
mutex (the pattern `casehandler/peerlock.go` uses) is **not** a valid concurrency
primitive here, since two requests for the same target can land on different
pods. All serialization must be DB-level.

### 5.1 Transaction wrapping

`AddressCreate`, `AddressUpdate`, `AddressDelete`, `AddressClaim`, and
`AddressResetPrimary` are today independent single-statement `Exec` calls with no
`BeginTx` (confirmed round-1/round-2, `pkg/dbhandler/address.go`). This design
wraps **all five** in a single `BeginTx` per outer operation, following
`casehandler.getOrCreateAttempt`'s shape (`pkg/casehandler/getorcreate.go:124-164`):
open tx, do every step that operation actually needs on that one tx,
run the §4 locking read (`SELECT ... FOR UPDATE` over *every* period row for
the target, not just one) before deciding which of §4's steps applies,
commit.

**`AddressResetPrimary` is conditional per operation, not universally invoked**
(round-3 finding, clarified here to remove the §5.1/§5.2 ambiguity that
survived round-3): today's code only calls it from `AddAddress`/`UpdateAddress`
when `is_primary` is part of the request (`pkg/contacthandler/contact.go:318-322,
372-378`); `RemoveAddress`/`ClaimAddress` never call it at all. This design does
not change that — `AddressDelete`/`AddressClaim`'s transactions never invoke
`AddressResetPrimary`, full stop. §5.2's "every code path follows this same
order" refers strictly to *lock acquisition order when a step is actually
performed*, not to forcing `AddressResetPrimary` to run unconditionally on
every path — the ownership-period `FOR UPDATE` is acquired first as a fixed
rule, and whichever of `AddressResetPrimary`/`contact_addresses` write steps
are applicable to *that specific operation* run after it, in that relative
order, never out of it.

### 5.2 Lock ordering (round-2 finding, addressed)

**Canonical table order within any single transaction (round-28
consolidation): `contact_address_ownership_periods` (FOR UPDATE, ascending
`(type, target)` when multiple) → `contact_addresses` writes (including
`AddressResetPrimary`) → `contact_contacts` (plain read for the round-27/28
tombstone checks, or the tombstoning UPDATE in `ContactDelete`) — always
last, never first.** The rules below define the first two positions; the
`contact_contacts`-last rule exists because `ContactDelete`'s single
transaction ends with the `contact_contacts` UPDATE, so any address-write
transaction that touched `contact_contacts` before its period/address locks
would create an AB-BA surface against it (§4's round-28 lock-order note).

Round-2 found that if `AddressResetPrimary` (contact-scoped: all rows for
`contact_id`) and the ownership-period `FOR UPDATE` (target-scoped: all period
rows for one specific `(customer_id, type, target)`, per §4's locking read)
run in different orders across two concurrent
requests touching the same Contact, InnoDB can deadlock (1213). Fix: **fixed lock
order within the single transaction, applied only on the operations that
actually invoke `AddressResetPrimary` (`AddAddress`/`UpdateAddress` with
`is_primary` set — see §5.1)** — always acquire the ownership-period `FOR
UPDATE` row lock *first*, then run `AddressResetPrimary` if this operation
calls for it, then the `contact_addresses` write, then commit. `AddressDelete`/
`AddressClaim` (which never call `AddressResetPrimary`) still acquire the
ownership-period `FOR UPDATE` lock first, per the same fixed rule, simply with
no `AddressResetPrimary` step to follow it — so no two transactions can ever
request the ownership-period lock and the primary-reset lock in reverse order
relative to each other, regardless of which subset of operations either one
performs.

**Second lock-ordering rule, for the two-target case (round-6 finding: this
was missing entirely).** `AddressUpdate` with a changed `target` (§4's table)
acquires the §4 locking read **twice** in the same transaction — once for the
old target, once for the new. Two concurrent `AddressUpdate` calls that swap
targets in opposite directions (Tx1: A→B, Tx2: B→A, same two targets) would
deadlock under any acquisition order that isn't fixed relative to the target
values themselves, not relative to old-vs-new. Fix: **always acquire the two
locking reads in ascending order of the `(type, target)` pair, byte-wise
(`ORDER BY type, target`), regardless of which one is "old" and which is
"new" for this particular call.** Every `AddressUpdate` transaction that
touches two targets follows this same total order, so two transactions
touching the same target pair can never request them in reverse order
relative to each other, closing the same class of deadlock §5.2's
`AddressResetPrimary` rule closes for the single-target case.

**Round-18 finding: `AddressUpdate` requests submitting a changed `target`
*and* `is_primary=true` in the same call are a real, tested code path
(`contacthandler.UpdateAddress`, `contact.go:359-378`'s independent `target`
and `is_primary` handling; exercised by `v1_contacts_update_test.go:33`'s
`{"target":..., "is_primary":true}` body) — the two lock-ordering rules
above were each specified in isolation and never composed for this fourth
combination (target-changed × is_primary-set, on top of the already-covered
target-unchanged × is_primary-set and target-changed × is_primary-unset
cases).** **Fix, composing the two existing rules rather than adding a
third:** when both apply to the same `AddressUpdate` call, acquire the two
`(type, target)` locking reads first, in the same ascending order the
second rule already mandates, then run `AddressResetPrimary` per the first
rule — i.e. the first rule's "ownership-period lock(s) before
`AddressResetPrimary`" ordering is unaffected by there being one lock or two;
it is simply "acquire every ownership-period lock this operation needs
(one, in ascending target order if there happen to be two) before
`AddressResetPrimary`, if this operation calls for it, before the
`contact_addresses` write, before commit." No transaction ever requests
`AddressResetPrimary` before an ownership-period lock, and no transaction
ever requests two ownership-period locks out of ascending order, regardless
of which combination of target-change and `is_primary` a given call carries
— so this fourth case introduces no lock-ordering rule that isn't already a
direct composition of the two existing ones, and no new deadlock class.

### 5.3 Deadlock retry

Even with fixed lock order, MySQL can still report deadlock 1213 under
contention (two *different* targets whose lock footprints overlap via a shared
Contact's `AddressResetPrimary` scan). Reuse `casehandler`'s exact pattern
verbatim: `maxDeadlockRetries = 3`, fresh `BeginTx` per attempt, surface
`ErrDeadlockExhausted` as a transient 5xx on exhaustion (`getorcreate.go:26-51,
86-117`). This is a straight lift, not a new mechanism.

**Round-8 addition: the same retry loop also covers `AddressClaim`'s
stale-target compare-and-retry (§4).** A pre-lock `target` read that no
longer matches the post-lock re-read is treated as the same class of
transient condition as deadlock 1213 — both mean "the world moved under us,
retry the whole operation" — and both count against the same
`maxDeadlockRetries = 3` cap rather than a separate counter, so a target that
keeps changing under a retrying `AddressClaim` fails closed (transient 5xx)
at the same bound instead of retrying indefinitely. **Round-40
clarification: this rule is caller-generic, not `AddressClaim`-specific**
— it covers every stale-target-mismatch retry §4 defines, which as of
round-39 also includes `AddressUpdate` (full-transaction restart) and
`AddressDelete` (single-target restart); all three share one
`maxDeadlockRetries` budget per outer operation, not one budget each.

### 5.4 `RowsAffected` + error mapping (B5 fix)

`AddressUpdate`/`AddressDelete` gain the same `RowsAffected == 0 → ErrConflict`
check `AddressClaim` already has (`address.go:194-210`). Critically — this is the
part round-2 found the original B5 fix proposal missed — **the dbhandler fix
alone is not sufficient**: `contacthandler.UpdateAddress`/`RemoveAddress`
(`pkg/contacthandler/contact.go:380-382, 405-407`) currently wrap every dbhandler
error in a bare `fmt.Errorf(...: %w, err)` with no `errors.Is` branch, so
`ErrConflict`/`ErrNotFound` would still fall through to
`listenhandler`'s generic 500 path. This design adds the same
`stderrors.Is(err, dbhandler.ErrConflict) → cerrors.AlreadyExists` /
`stderrors.Is(err, dbhandler.ErrNotFound) → cerrors.NotFound` branches
`ClaimAddress` already has (`contact.go:457-471`) to both handlers.

**Round-35 addition — the mapping list was incomplete for the two
handlers round-11 pulled into the step procedure.** Step 1 makes
`dbhandler.AddressCreate` return `ErrConflict` on a cross-owner live
conflict, but (a) `contacthandler.AddAddress` (`contact.go:324-332`)
today has only an `ErrDuplicateTarget` branch — the new sentinel would
fall through to a generic 500, even though the SAME user action ("POST
an address another contact already owns") returns 409
`ADDRESS_ALREADY_EXISTS` today via the unique index; and (b)
`contacthandler.Create` (round-11: Step-1 errors propagate out of the
address loop) has no mapping branches at all. Rule: **`AddAddress` and
`Create`'s address-loop error path both gain the same
`stderrors.Is(err, dbhandler.ErrConflict) → cerrors.AlreadyExists`
branch** (the alternative — reusing `ErrDuplicateTarget` as Step 1's
sentinel — is rejected because §4's repair logic must distinguish
"period conflict" from "unique-index collision": they trigger different
recovery paths). **Round-36 reason-code correction: "the same branch"
refers to the sentinel-matching mechanics only, NOT the reason-code
payload.** `ClaimAddress`'s existing branch emits
`ADDRESS_ALREADY_CLAIMED` — but the user action reaching `AddAddress`'s
new branch ("POST an address another contact already owns") returns
`ADDRESS_ALREADY_EXISTS` today via the unique-index path, and the
2026-07-02 duplicate-address design deliberately separated those two
reason codes as an external contract. Rule: `AddAddress` and `Create`'s
address-loop branches map Step-1 `ErrConflict` to
`cerrors.AlreadyExists` with reason code **`ADDRESS_ALREADY_EXISTS`**
(and its existing message), preserving today's externally observable
payload byte-for-byte; `ClaimAddress` keeps `ADDRESS_ALREADY_CLAIMED`
for its own path. The full §5.4 mapping roster is therefore:
`UpdateAddress`, `RemoveAddress`, `AddAddress`, `Create` (address
loop), plus `ClaimAddress` which already has the branches.

## 6. Read paths

### 6.1 New dbhandler function, existing function untouched

`AddressListByContactID` (`pkg/dbhandler/address.go:320`) is **not modified**.
Round-1 confirmed it is shared by `ContactGet`/`ContactList`/
`contactUpdateToCache` (`pkg/dbhandler/contact.go:75,152,233`) to populate the
public `Contact.Addresses` API field; changing its semantics would leak closed/
historical addresses into `GET /v1/contacts/{id}`.

A new function, `OwnershipPeriodsListByContactID(ctx, contactID) []OwnershipPeriod`,
is added instead, used **only** by the two interaction-read paths below.

### 6.2 `interactionListByContact` (STEP1/STEP2)

STEP1 changes from `AddressListByContactID` (live rows only) to
`OwnershipPeriodsListByContactID` (all periods, open and closed). STEP2's value
match gains a time-range condition:

```sql
(peer_type = ? AND peer_target = ?
 AND COALESCE(tm_interaction, tm_create) >= COALESCE(?, tm_interaction, tm_create)
 AND COALESCE(tm_interaction, tm_create) <  COALESCE(?, tm_interaction + INTERVAL 1 SECOND, tm_create + INTERVAL 1 SECOND))
```

**Round-41 BLOCKER, fixed here: STEP1's period-only source silently
dropped the missing-period skew population §6.3's round-40 fix already
covers on the queue side.** Because `OwnershipPeriodsListByContactID`
queries `contact_address_ownership_periods` exclusively, a target this
Contact currently, live-owns but that has zero period rows (§9
round-16/17's accepted degraded state — an old-binary pod's
`AddressCreate` skipped the period write) contributes NO bound to
STEP2's OR-list at all: not an unbounded one, none. Every interaction
for that target vanishes from the Contact's OWN timeline — worse than
§6.3's pre-round-40 bug, because there the interactions at least
resurfaced somewhere (the unresolved queue); here they resurface
nowhere, reproducing defect #1 (the timeline-vanishing defect this
entire design exists to fix) via pure skew, with no Contact deletion
involved. **Fix: STEP1 additionally queries `contact_addresses` for
`(type, target)` pairs this `contact_id` owns (`contact_id = ?`) that
have NO matching row in `contact_address_ownership_periods` for that
`(customer_id, type, target)` — the same anti-join §6.3's third
disjunct performs, mirrored from the ownership side instead of the
unresolved side — and STEP2 gives each such pair an unconditional
(unbounded, `valid_from = valid_to = NULL`) bound, identical in shape
to a true first-registration period.** This reproduces today's
time-agnostic value-match for exactly this population until the next
touch gives the row a real period, at which point it drops out of this
extra query and the ordinary period-bound path takes over — same
transient-and-bounded character as §6.3's fix, same handoff pattern.
§10 gains an `interactionListByContact` missing-period-skew fixture
symmetric to §6.3's round-40 one.

**Round-21 BLOCKER, fixed here: the previous version of this condition
compared bare `tm_interaction`, which is a documented-nullable column**
(`models/interaction/interaction.go:36-39`: "may be nil when the origin
event omits TMCreate (e.g. call events with omitempty). Stored as NULL in
that case"; the projection at `contacthandler/interaction.go:114,168`
propagates that nil directly). Under SQL three-valued logic every
comparison against a NULL `tm_interaction` yields NULL (falsy), so a bare
comparison would exclude every NULL-`tm_interaction` interaction from
every period — including the fully-unbounded `[NULL, NULL)` period — on
Contacts that never had a single reassignment. Today's pure-equality
matching returns those rows fine, so this would have been a silent
regression reintroducing defect #1 (interactions vanishing from the
timeline) and, through §6.3's mirror of the same comparison, defect #3
(currently-owned addresses' interactions resurfacing in the unresolved
queue). The fix falls back to the interaction row's `tm_create` — which
the projection always populates with `NOW()` and is `NOT NULL` — as the
effective event time for period matching: an interaction whose origin
event carried no timestamp is attributed by when this system recorded it,
the closest available approximation, rather than being unmatchable.

i.e. `tm_interaction ∈ [valid_from or -∞, valid_to or +∞)` per period, OR'd
across every period the Contact has ever held.

**Round-16 finding: the claim above that this "mirrors the existing
OR-expansion shape... no new query pattern" does not hold up against the
actual code.** `dbhandler/interaction.go:23-28`'s `AddressPair{Type, Target
string}` and the `sq.Or{sq.Eq{...}}` loop building it (`interaction.go:131-139`)
only carry pure equality clauses — there is no field to carry a period's
`valid_from`/`valid_to`, and `sq.Eq` cannot express a range condition. STEP2
cannot be implemented by passing more `AddressPair` values into the existing
`InteractionList` call; the type and the OR-builder both need to change.
**Concrete implementation:**

```go
// dbhandler/interaction.go — new type, additive alongside AddressPair
// (AddressPair itself is unchanged; every other InteractionList caller keeps
// using it exactly as today — see call-site note below)
type OwnershipPeriodBound struct {
	Type      string
	Target    string
	ValidFrom *time.Time // nil = unbounded past
	ValidTo   *time.Time // nil = still open (unbounded future)
}
```

`InteractionList`'s OR-builder gains a second code path selected by which
slice type the caller passes (a new optional parameter, not a signature
change to the existing one — see below), building each clause as:

```go
// Round-17 finding: sq.GtOrEq{"col": sq.Expr(...)}/sq.Lt{...} is NOT valid —
// squirrel's Eq/Lt/GtOrEq only special-case driver.Valuer values in their
// toSql() implementation (vendor/github.com/Masterminds/squirrel/expr.go),
// with no branch for a Sqlizer (what sq.Expr returns) placed as a map
// value — passing one there serializes to a bind-parameter placeholder
// with the *expr struct itself* as the argument, which errors at query-exec
// time ("unsupported type") since expr doesn't implement driver.Valuer.
// sq.And/sq.Or (composing whole conditions) do check for Sqlizer; individual
// column-comparison builders like Eq/Lt/GtOrEq do not. The comparison must
// instead be built as a single raw-SQL fragment via sq.Expr, with the ±∞
// fallback expressed inside that one SQL string rather than assembled from
// separate squirrel value-position helpers:
validFromSQL, validToSQL := "COALESCE(tm_interaction, tm_create)", "COALESCE(tm_interaction, tm_create) + INTERVAL 1 SECOND"
if b.ValidFrom != nil {
	validFromSQL = "?"
}
if b.ValidTo != nil {
	validToSQL = "?"
}
clause := fmt.Sprintf("(peer_type = ? AND peer_target = ? AND COALESCE(tm_interaction, tm_create) >= %s AND COALESCE(tm_interaction, tm_create) < %s)", validFromSQL, validToSQL)
args := []any{b.Type, b.Target}
if b.ValidFrom != nil {
	args = append(args, *b.ValidFrom)
}
if b.ValidTo != nil {
	args = append(args, *b.ValidTo)
}
or = append(or, sq.Expr(clause, args...))
```

(This is the same pattern §6.3's `NOT EXISTS` correlated subquery already
uses — the ±∞ fallback lives inside one SQL string, never as a value handed
to a squirrel column-comparison builder.) **Call-site impact:** `InteractionList` is
shared by `interactionListByContact`'s STEP2 (this call site, needs the new
`[]OwnershipPeriodBound` path) and by the single-address re-fetch in
`interaction_read.go:285` (STEP5, unchanged — that call site still only
needs plain equality on one already-known address and keeps using
`[]AddressPair`, not touched by this change).

**Round-18 finding: calling this "a new optional parameter, defaulted to the
existing behavior when omitted" is not implementable as stated.** Go has no
optional-parameter syntax, and `InteractionList`'s existing signature
(`ctx, customerID, size, token, peerType, peerTarget string, addressSet
[]AddressPair, since time.Time`, confirmed against every existing call site
in `interaction_test.go`) already ends in `since time.Time` — variadic
arguments are Go's only optional-parameter-like mechanism, must be the final
parameter, and Go permits at most one per signature, so a second variadic
`[]OwnershipPeriodBound` cannot be appended after `since` and cannot be
inserted before it without breaking every existing positional call site,
directly contradicting "no existing caller... needs to change." **Fix:**
this is a **new sibling function**, not a modified existing one —
`InteractionListByOwnershipPeriods(ctx, customerID, size, token, peerType,
peerTarget string, bounds []OwnershipPeriodBound, since time.Time)` — sharing
`InteractionList`'s query-building internals (the same base query, pagination,
and `since` handling) but taking `bounds` where `InteractionList` takes
`addressSet` and building the OR-clause via the round-17 `sq.Expr` fragment
instead of `sq.Eq`. `InteractionList` itself, its signature, and every
existing caller (`interaction_read.go:285`, `mock_main.go:708`, `main.go:58`,
and all of `interaction_test.go`) are completely unchanged — only STEP2 of
`interactionListByContact` calls the new function instead. STEP0, 3–6 of
`interactionListByContact` are otherwise unchanged.

**Round-19 finding (recorded, non-blocking): this section didn't say that a
new function still needs the standard `DBHandler` interface + mock update.**
`InteractionListByOwnershipPeriods` must be added to the `DBHandler`
interface (`pkg/dbhandler/main.go`) alongside `InteractionList`, and
`go generate ./pkg/dbhandler/...` re-run to regenerate `mock_main.go` before
`contacthandler`'s tests (which construct `dbhandler.NewMockDBHandler(mc)`)
can compile against it. This isn't a new step this design invents — it's the
same `go generate ./...` the repo's `CLAUDE.md` verification workflow
already mandates for any interface change — but earlier sections of this
design (the Prometheus `init()` block, the `AddressGet` retry wiring) were
specific about codegen/registration steps like this one, so leaving this one
implicit was an inconsistency worth naming rather than a functional gap.

**Round-3 finding, addressed here: OR-clause count is no longer bounded by
"how many addresses does this Contact currently have" (typically single-digit)
but by "how many ownership periods has this Contact ever held" — which grows
monotonically for any Contact that experiences repeated reassignment (e.g. a
call-center front-desk number rotated across agents).** This design does not
attempt to cap or paginate the OR-clause count in this iteration — for the
realistic case (an end-customer Contact with a handful of numbers over its
lifetime) the count stays in the same order of magnitude as today. It is
recorded as an explicit known limitation (§10) rather than solved now, because
solving it properly (e.g. a `JOIN` against
`contact_address_ownership_periods` instead of a Go-built OR-expansion, which
would also remove the OR-count concern entirely) is a larger rewrite of
`InteractionList`'s calling convention that changes today's `AddressPair`-slice
interface (`dbhandler/interaction.go`) — out of scope for a design whose stated
goal (§2) is a same-API-surface internal rewire. If a customer's period count
is ever observed approaching a problematic OR-clause size in production, that
observation is the trigger to revisit this as a follow-up, not something to
pre-solve speculatively now.

### 6.3 `InteractionListUnresolved`

Round-3 review found the original text here ("replaced with the equivalent
`NOT EXISTS` ... with the same time-range condition as §6.2") was not
sufficient to implement: §6.2's condition is expressed as Go-side bound
parameters injected per period via an OR-loop
(`dbhandler/interaction.go:131-139`'s existing shape), which is structurally
different from a single correlated subquery, where the range condition must
reference the joined table's own columns, not external bind parameters. The
actual replacement SQL, concretely:

```sql
-- was (pkg/dbhandler/interaction.go:259-264):
NOT EXISTS (
    SELECT 1 FROM contact_addresses a
    WHERE a.customer_id = i.customer_id
      AND a.type = i.peer_type
      AND a.target = i.peer_target
)

-- becomes (round-36: the unresolved-row disjunct is REQUIRED — see below):
NOT EXISTS (
    SELECT 1 FROM contact_address_ownership_periods p
    WHERE p.customer_id = i.customer_id
      AND p.type = i.peer_type
      AND p.target = i.peer_target
      AND COALESCE(i.tm_interaction, i.tm_create) >= COALESCE(p.valid_from, i.tm_interaction, i.tm_create)
      AND COALESCE(i.tm_interaction, i.tm_create) <  COALESCE(p.valid_to, i.tm_interaction + INTERVAL 1 SECOND, i.tm_create + INTERVAL 1 SECOND)
)
AND NOT EXISTS (
    SELECT 1 FROM contact_addresses a
    WHERE a.customer_id = i.customer_id
      AND a.type = i.peer_type
      AND a.target = i.peer_target
      AND a.contact_id IS NULL
)
AND NOT EXISTS (
    -- round-40: missing-period skew guard (see the finding below)
    SELECT 1 FROM contact_addresses a
    WHERE a.customer_id = i.customer_id
      AND a.type = i.peer_type
      AND a.target = i.peer_target
      AND a.contact_id IS NOT NULL
      AND NOT EXISTS (
          SELECT 1 FROM contact_address_ownership_periods p2
          WHERE p2.customer_id = a.customer_id
            AND p2.type = a.type
            AND p2.target = a.target
      )
)
```

**Round-40 finding, fixed by the third `NOT EXISTS` above: a live-owned
row with zero period rows — the missing-period skew state §9
round-16/17 already accepts as a degraded state reachable via an
old-binary pod's `AddressCreate` — was falling through both existing
disjuncts and re-surfacing ALL of that address's interactions in the
unresolved queue, including ones for a perfectly normal, currently-owned
address that was never deleted.** The first `NOT EXISTS` (periods) is
vacuously true (no period rows to match — nothing to NOT-EXISTS
against), and the second `NOT EXISTS` (unresolved-row presence) is also
vacuously true because the row IS owned (`contact_id IS NOT NULL`, so
the `contact_id IS NULL` filter excludes it). Both guards were written
for their own populations (owned-with-periods, and unresolved-without-
periods) and neither one's author considered the fourth combination
(owned, without-periods) that only exists because of §9's accepted skew
window — the same "population enumerated for the write path, not
re-checked against every read-path query" gap round-36 found for
unresolved rows, recurring one population over. The third disjunct
closes it: any address row that is owned but has no period row at all
suppresses its interactions the same way an unresolved row does (by
presence, time-agnostic) until the next touch gives it a period (Step 4
or 5, whichever applies) — at which point the third disjunct stops
matching and the first (periods) disjunct takes over, exactly the same
handoff pattern the second disjunct already uses for claims. This
degraded state is by definition transient (bounded by the rolling
deploy window per §9), so the suppression window is bounded too. §10
gains a missing-period-skew fixture for `InteractionListUnresolved`.

**Round-36 finding, fixed by the second `NOT EXISTS` above: replacing
the subquery wholesale would have silently removed the unresolved-row
suppression effect — an unrecorded external behavior change.** Today's
`NOT EXISTS` over `contact_addresses` carries no `contact_id` filter, so
a `CreateUnresolvedAddress` row (`contact_id = NULL`) suppresses its
matching interactions from the unresolved queue — that is the endpoint's
only queue-side observable effect, and per §3.1/round-10 unresolved rows
never get a period, so a periods-only subquery would (a) re-surface every
such interaction at cutover and (b) permanently strip the endpoint's
effect. The second disjunct preserves today's suppression exactly:
unresolved rows suppress by presence (time-agnostic, as today), owned
addresses suppress by period (the design's time-aware rule). **Round-37
scoping correction, bound updated in round-38, further scoped in
round-39 (the earlier "matching today's observable sequence" claim was
true only for the Step-5 claim path):** after a claim converts the row
(`contact_id` no longer NULL), the second `NOT EXISTS` stops suppressing
and the first takes over — continuity of suppression for the pre-claim
era therefore depends on the claim-created period's `valid_from` reaching
back over that era. Step 5 claims (`valid_from = NULL`) always do. Step
4 / Step 3-INSERT claims would NOT have under the old `NOW()` rule —
which is why §4's caller-specific rule sets `ClaimAddress`'s `valid_from
= latest closed valid_to`: every interaction since the IMMEDIATELY
PRIOR owner's era attaches to the claimer and does not resurface in the
unresolved queue — matching today's observable behavior for that single
gap. **Round-39 correction: "matching today's observable behavior" does
NOT extend to every earlier gap in a multi-era history** — §4's round-39
disposition records that an older, non-adjacent gap (e.g. an A-to-B gap
preceding a B-to-claimer gap) is NOT covered by the claim-created period
and DOES resurface in the queue once the claim removes the row-presence
suppression that had covered it unconditionally. This is accepted (see
§4), not a defect, but the earlier phrasing "nothing resurfaces... the
one exception is the mixed-skew clamp" overstated the guarantee to
exactly one named exception; there are two: the mixed-skew clamp AND
non-adjacent historical gaps. Claimed
rows are covered by the period subquery from the claim onward. §10's
test checklist covers `InteractionListUnresolved` with an unresolved-row
fixture on both sides of the cutover, **plus (round-37/38/39) a
claim-of-unresolved fixture on a target with prior closed periods,
asserting BOTH the unresolved-era and the immediately-prior gap
interactions attach to the claimer and do NOT resurface in the queue,
AND a two-prior-era fixture asserting the older, non-adjacent gap DOES
resurface (documented, accepted).**

These are three correlated subqueries keyed entirely on the outer row's own
`contact_interactions` columns (`i.customer_id`, `i.peer_type`, `i.peer_target`,
`i.tm_interaction`) — no Go-side loop, no bind parameters beyond the ones the
surrounding query already has. `idx_ownership_periods_lookup` (§6.4) covers
the period subquery's `WHERE` clause (used by both the first disjunct and
the round-40 third disjunct's inner `NOT EXISTS`), and the existing
`idx_contact_addresses_identifier` prefix drives the unresolved-row and
missing-period-skew subqueries' equality lookups (round-37 precision: not
fully covering — `contact_id`/`IS NOT NULL` are not in that index, so one
row lookup per matched target, at most one row by uniqueness; negligible),
the same way `idx_contact_addresses_lookup` already covers the original.

### 6.4 Index coverage (round-1 flagged as unresolved; addressed here)

`idx_ownership_periods_lookup (customer_id, type, target, valid_from, valid_to)`
is added so the OR-expanded, time-bounded STEP2 query and the `NOT EXISTS`
in §6.3 are driven by the `(customer_id, type, target)` equality prefix and
**covered** by the index (no row lookups). Round-22 correction: the two
appended range columns serve the covering role only, not range pruning —
they appear inside `COALESCE(...)` with outer-row columns mixed in, which
MySQL cannot use for index range access. This is fine in practice: a target
carries single-digit period rows, so evaluating the range condition as a
residual filter over the covered rows is negligible. The index mirrors
`idx_contact_interactions_peer`'s existing shape
(`customer_id, peer_type, peer_target`) with the two period columns appended
for coverage.

## 7. Known, accepted limitation: the reassignment TOCTOU gap

`AddressDelete`(A) and `AddressCreate`/`ClaimAddress`(B) are two independent
RabbitMQ RPCs, each its own transaction (§5.1). Between A's commit and B's
commit, no ownership period is open for that `(customer_id, type, target)`. Any
interaction landing in that window is correctly-but-unhelpfully classified
unresolved (it belongs to neither A's now-closed period nor B's not-yet-opened
one). This is: (a) the same class of gap that already exists in the *current*
production system for this exact operation sequence — not a regression this
design introduces; (b) narrow in practice (the realistic window is the time
between two API calls an operator/integration makes back-to-back, typically
sub-second); (c) explicitly not addressed by adding a new atomic API, per
pchero's direction (§2). If reassignment volume ever justifies closing this gap,
the fix is a new atomic `transfer` endpoint — deliberately not built now.

## 8. Bugs found during scenario/review passes, and their disposition

Full scenario matrix (A1–A10, B1–B5, C1–C3, D1–D6) and round-1/round-2 adversarial
review transcripts are preserved in the originating Hermes session (not
duplicated here). Disposition of each finding that survived to this design:

| ID | Finding | Disposition |
|---|---|---|
| A9-b | `AddAddress`/`UpdateAddress`/`RemoveAddress`/`ClaimAddress` don't check `Contact.TMDelete` | **Fixed** (§4's dedicated "A9-b guard placement" paragraph — an explicit `TMDelete != nil` guard added to all four handlers, mirroring `interactionListByContact`'s existing check at `interaction_read.go:117-125`. Round-4 review found this row previously mis-cited §4's unrelated "Contact soft-delete (`ContactDelete`)" table row — that row closes open ownership periods when a Contact is deleted, a different mechanism from this guard; corrected to cite the actual guard paragraph.) |
| B5 | `AddressUpdate`/`AddressDelete` missing `RowsAffected` check + error-type mapping | **Fixed** (§5.4) |
| D5 | `TypeNone` missing from `crmIneligiblePeerTypes` | **Dropped — not a bug.** Round-2 proved `TypeNone` is the deliberate "unknown direction, persist a diagnostic zero-value row" sentinel; an existing test (`interaction_test.go:106-138`) locks this behavior in. Adding it to the blacklist would silently break that intentional behavior. |
| D6 | `case-control reconcile-contact` CLI not in original scenario table | Documented as an existing, independent mechanism (§2 out-of-scope); no interaction with this design's tables. |
| Round-1 BLOCKER: `ClaimAddress` unconditional-reject makes the originally-proposed "reassign inside `ClaimAddress`" branch unreachable | **Resolved by redefinition**, not by changing `ClaimAddress`: reassignment is delete-then-create/claim (§4, §7), not a `ClaimAddress`-internal branch. |
| Round-1 BLOCKER: no DB-level uniqueness on "one open period" | **Fixed** (`open_period_uk`, §3.1, direct lift of `contact_cases.uq_case_open_peer`'s pattern) |
| Round-1 BLOCKER: `[NULL,NULL]` backfill for all existing addresses is wrong (re-introduces mis-attribution for addresses with an unknown-to-us prior reassignment history) | **Superseded by round-22.** Round-1's answer (backfill `valid_from = tm_create`) traded that risk for a worse, certain one round-22 caught: erasing currently-visible pre-registration history at cutover (see the round-22 row below). Final rule: `valid_from = NULL` after all — round-1's concern is real but is exactly the *status quo* misattribution this system already exhibits today (NULL reproduces today's time-agnostic matching, no better and no worse at cutover), whereas `tm_create` was an active regression. Round-1's underlying concern is preserved honestly in §9's round-22 accepted-limitation paragraph: pre-migration reassignment history is unrecoverable-by-design, and defect #2 is only prevented for ownership changes made after migration. |
| Round-1 finding: `AddressListByContactID` reuse would leak into `ContactGet`/`ContactList` | **Fixed** (§6.1, new function instead) |
| Round-2 finding: lock-ordering deadlock risk from omitting `AddressResetPrimary` | **Fixed** (§5.2) |
| Round-2 finding: B5's dbhandler-only fix doesn't propagate to contacthandler error mapping | **Fixed** (§5.4) |
| Round-3 BLOCKER: `AddressCreate`'s "no prior period" rule didn't distinguish true-first-registration from reassignment-to-a-different-contact, re-introducing the exact history-leak defect (§1, defect #2) this design exists to fix | **Fixed** (§4, third table row + two-lookup rule) |
| Round-3 finding: §6.3 (`InteractionListUnresolved`) described the replacement only in prose, without concrete SQL, and the described "same condition as §6.2" elided a real structural difference (Go-loop bind params vs. single correlated subquery) | **Fixed** (§6.3, concrete SQL added) |
| Round-3 finding: §6.2's OR-clause count is unbounded by ownership-period accumulation, not addressed in the original text | **Acknowledged as an explicit known limitation** (§6.2, §10) — not solved this iteration; revisit only if observed in practice, per pchero's "don't pre-solve theoretical risk" standing preference |
| Round-3 finding: §8's A9-b row referenced a guard never specified in §4's body (table led the design instead of summarizing it) | **Fixed** (§4, explicit guard-placement paragraph added) |
| Round-3 finding: §5.1 ("`AddressResetPrimary` when applicable") and §5.2 ("every code path follows this same order") read as contradictory — could be misread as forcing `AddressResetPrimary` on Delete/Claim | **Fixed** (§5.1/§5.2 reworded to make the conditionality explicit in both places) |
| Round-3 finding: §4 had no row for `AddressUpdate` calls that change neither `target` nor `is_primary` | **Fixed** (§4, explicit no-op row added) |
| Round-4 BLOCKER: §4's reopen rule (rule 2) didn't detect an intervening different-contact_id owner between a contact's old closed period and its reclaim, silently stretching the reopened row's `valid_from` back over the intervening owner's interval (A→B→A case) — this re-introduced the same history-leak defect (§1, defect #2) in the reopen path, mirroring the round-3 defect in the create path | **Fixed** (§4, third lookup (c) added + new "closed period + intervening owner" table row; also resolves round-3's secondary "which of several closed periods" ambiguity via the same lookup, ordered `valid_to DESC`) |
| Round-4 finding: §8's A9-b disposition cited §4's "Contact soft-delete" table row, which is an unrelated mechanism (closing periods on Contact deletion, not blocking writes to an already-deleted Contact) | **Fixed** (this row, corrected citation) |
| Round-5 BLOCKER: the round-3/4 "three separate lookups" framing could not classify true first-registration vs. reassignment-to-a-different-contact (both looked identical: no row for *this* contact_id, no *open* row anywhere) — a B claiming a target A had already released would be misclassified as first-ever registration, re-leaking A's history to B via `valid_from=NULL` | **Fixed** (§4 replaced entirely: a single `FOR UPDATE` read over *all* period rows for the target, decided in application code from the full fetched set — case 2 explicitly checks "does *any* contact_id have a row here", closing the classification gap) |
| Round-5 finding: the round-3/4 lookups were plain (non-locking) `SELECT`s under `REPEATABLE READ`, so a transaction's reopen-vs-new-row decision could read a stale snapshot that missed a just-committed intervening owner, re-opening a case round-4 believed fixed under concurrency | **Fixed** (same §4 replacement — the single read is a real `SELECT ... FOR UPDATE`, not a plain `SELECT`, so it observes true latest-committed state, not a snapshot) |
| Round-5 finding: `valid_from`'s "see §6 backfill" schema comment was stale (backfill logic lives in §8/§9, not §6) | **Fixed** (§3.1 comment corrected) |
| Round-6 finding: §4's case 4 ("open row for this contact_id already exists") was claimed unreachable, but `AddressClaim`'s idempotency pre-check runs as a plain non-locking read *before* the `FOR UPDATE` acquisition, so a duplicate-delivery race (two concurrent `ClaimAddress` calls for the same contact) can genuinely reach it | **Fixed** (§4 case 4 rewritten: the idempotency check moves inside the lock instead of preceding it, making the case a well-defined in-lock no-op rather than an unreachable branch) |
| Round-6 finding: §4's two-target `AddressUpdate` (target changed) acquires two locking reads in the same transaction, but §5.2's fixed lock-ordering rule only covered the single-target case — two concurrent target-swap `AddressUpdate` calls in opposite directions could deadlock with no ordering rule to prevent it | **Fixed** (§5.2, second lock-ordering rule added: always acquire the two target locks in ascending `(type, target)` order, independent of which is "old" vs "new") |
| Round-6 finding (recorded, not actioned): the document's per-round meta-commentary ("round-3 finding", "round-4 BLOCKER" inline in §4/§5's rule text) makes it harder for an implementer to separate "the current rule" from "why it changed" | Acknowledged as a real readability cost; not restructured in this pass to avoid re-opening settled sections mid-review-loop. If this design proceeds to implementation, a follow-up pass should move all "round-N found..." prose out of §4/§5's rule bodies into §8 and leave only terse rule statements plus a `(history: §8)` pointer. |
| Round-7 finding: case 4's fix (idempotency pre-check moved inside the lock) left the response's transaction handling (commit vs. rollback on no-op) and the still-non-locking target-address lookup that precedes the lock unaddressed | **Fixed** (§4 case 4: explicit "commit, unchanged post-commit response path" note added; §4 new paragraph after case 5: the `(type, target)` values feeding the lock request are shown to already be covered by this design's own lock-ordering discipline, not a separate unprotected read) |
| Round-7 BLOCKER: case 2's "(open or closed)" wording for the other contact_id's row conflated two situations with opposite required handling — a closed row (genuine reassignment, safe to INSERT) and an open row (live-owner conflict, must be rejected, not inserted — the insert would hit `idx_ownership_periods_open`'s unique constraint as an unmapped raw MySQL 1062) | **Fixed** (case 2 narrowed to "closed" only; new case 5 added for the open-row situation, mapping it to the same `ErrConflict`/409 `AddressClaim` already returns today, moved inside the lock for the same reason as case 4) |
| Round-8 BLOCKER: the round-6/7 "1–5 cases" list was never actually mutually exclusive — several real states (own row open AND own closed history; own closed row AND a different contact's currently-open row) satisfied more than one case's precondition, and no evaluation order was specified. Implemented in list order (the natural reading), this reproduced the same "unmapped 1062 from inserting over a live open row" bug round-5/7 believed already fixed | **Fixed** (§4 restructured from a flat case list into an ordered 5-step decision procedure — step 1 checks for a different contact's open row unconditionally before anything else, making every later overlap structurally impossible rather than merely documented) |
| Round-8 finding: the round-7 claim that `AddressClaim`'s pre-lock `addressID → target` resolution was "already protected by lock ordering alone" was incorrect — lock ordering serializes acquisition order, not the freshness of a value read before any lock was taken; a concurrent `AddressUpdate` target-change could commit between that read and this transaction's lock acquisition, leaving `AddressClaim` writing an ownership period for a target `contact_addresses` no longer has | **Fixed** (§4: `AddressClaim` re-reads `contact_addresses.target` inside the transaction immediately after acquiring the lock, and retries from the top via §5.3's existing deadlock-retry loop if the pre-lock and post-lock reads disagree — not a new retry mechanism) |
| Round-9 verification: re-derived the full 8-state truth table (self row absent/open/closed × other row absent/open/closed, with self-open and other-open mutually exclusive via `open_period_uk`) against the round-8 5-step procedure | **Confirmed correct** — every reachable state maps to exactly one step, no overlap or gap. This is the first round where §4's core decision logic itself produced zero findings. |
| Round-9 finding: round-8's stale-target compare-and-retry (§4/§5.3) only specified behavior for "target differs" — a concurrent `RemoveAddress` deleting the same `addressID` between the pre-lock and post-lock reads (`contact_addresses` is hard-delete) would return NotFound instead, a permanent condition the retry loop would have misclassified as transient, burning all retries before an incorrect 5xx | **Fixed** (§4: post-lock NotFound aborts immediately with `cerrors.NotFound`, no retry — the same mapping `AddressClaim`'s existing pre-lock NotFound case already uses) |
| Round-9 finding: two places in §4/§5.1 still referred to the pre-round-8 "case" terminology ("case 2 above", "which of §4's cases applies") after §4 was restructured into numbered steps, risking an implementer misreading a stale cross-reference | **Fixed** (both corrected to "Step 4" / "steps" respectively) |
| Round-10 finding: round-9's NotFound fix said "return `cerrors.NotFound`" from inside the transaction/dbhandler-layer procedure §4 describes, but `dbhandler` has no `cerrors` dependency (verified: zero imports) — the actual conversion happens only in `contacthandler.ClaimAddress`'s existing error-mapping branch, exactly as §5.4 itself establishes for B5 | **Fixed** (§4 reworded: dbhandler returns `dbhandler.ErrNotFound`, the same sentinel the existing pre-lock case already returns; `contacthandler.ClaimAddress`'s existing mapping branch handles the conversion, no new mapping code needed) |
| Round-10 BLOCKER: `CreateUnresolvedAddress` (`contact_id = uuid.Nil`) calls the same `dbhandler.AddressCreate` §4's step procedure governs, but §3.1 already establishes unresolved addresses must not get a period until `ClaimAddress` assigns a real owner — applying Step 1 with `contact_id = Nil` as "this contact" would misfire, treating every live owner anywhere as a conflict against `Nil` | **Fixed** (§4: explicit new paragraph scoping the entire step procedure to `contact_id != uuid.Nil`; `CreateUnresolvedAddress`'s `AddressCreate` calls skip the locking read and steps entirely, unchanged from today, until a later `ClaimAddress` call with a real contact_id runs the steps for the first time) |
| Round-10 verification: re-confirmed no remaining "case N" terminology and re-verified §3.1's schema supports every rule accumulated across all 9 prior rounds | **Confirmed correct**, no further findings on either point. |
| Round-11 BLOCKER: `contacthandler.Create` (`ContactCreate`) is a fifth caller of `dbhandler.AddressCreate`/`AddressResetPrimary`, missed by every prior round because review scope re-examined the logic of handlers already named in the document rather than grepping for every actual caller of the dbhandler functions this design touches. It calls both with a real (non-Nil) `contact_id`, so it is squarely subject to the step procedure, not exempt like `CreateUnresolvedAddress` — and today it also has no transaction and swallows `AddressCreate`/`AddressResetPrimary` errors entirely, meaning a live-owner conflict would silently succeed instead of surfacing `ErrConflict` | **Fixed** (§4: `ContactCreate`'s address loop now runs the same `BeginTx`-wrapped locking read and step procedure per address, with the same fixed lock ordering, and propagates errors instead of swallowing them; §2 scope note added noting the one behavior change — error propagation instead of silent swallowing — is a correctness fix to existing broken behavior, not new response-shape scope creep) |
| Round-12 verification: grepped every real call site of the 5 dbhandler Address* functions across all of `bin-contact-manager` (cmd/ + pkg/, excluding tests/mocks) | **Confirmed complete** — exactly 9 call sites across 6 logical paths (all in `pkg/contacthandler/contact.go`), all already covered by this document; `contact-control` CLI has zero address-function references; `dbhandler.ContactUpdate` only touches `contact_contacts`, never addresses; no batch/import path exists. No further callers remain. |
| Round-12 finding: round-11's fix (propagate the address loop's error instead of swallowing it) introduced a new partial-success state it didn't address — `ContactCreate` commits the base Contact row before the address loop runs, and each address is its own separate transaction, so an error on address N leaves the Contact and addresses 1..N-1 committed while the caller receives an error, with no disposition specified for that gap | **Fixed** (§4: explicit compensating-cleanup behavior added — on any address-loop failure, `ContactCreate` issues best-effort `RemoveAddress` calls for every address that succeeded earlier in the same loop before returning the original error, converging a failed request back to "no addresses were added" rather than full cross-address atomicity, which this design explicitly declines to add as out of scope) |
| Round-13 finding: `ContactDelete`'s "close every open period" effect had no transaction/lock-ordering/deadlock-retry discipline specified, unlike every other row in §4's table — it locks an arbitrary number of targets in one transaction, the same shape as `AddressUpdate`'s two-target case but with N instead of 2, exposed to the same deadlock class | **Fixed, with a round-19 correction** (§4: generalizes §5.2's existing ascending `(type, target)` lock-ordering rule from N=2 to arbitrary N, same `maxDeadlockRetries` retry loop — not a new mechanism. Round-19 later found this only holds for the *initial* lock acquisition; the round-14 re-check loop's late-added locks fall outside the fixed-total-order guarantee — see the round-19 row below) |
| Round-13 finding: reusing `contacthandler.RemoveAddress` verbatim for round-12's compensating cleanup would leak a spurious `EventTypeContactUpdated` for a Contact whose `ContactCreated` was never published (the create failed before reaching that point) | **Fixed** (§4: compensating cleanup calls the non-event-publishing dbhandler-level delete + ownership-period closure directly, not the public `RemoveAddress` entry point) |
| Round-13 finding: §10's transaction-boundary-diff checklist named five handlers but omitted `ContactCreate`, despite round-11/12 changing it — an existing test, `Test_Create_WithAddressTagErrors`, explicitly asserts today's error-swallowing behavior and will need rewriting, not just re-verification | **Fixed** (§10: `ContactCreate` added to the checklist, with the specific test named) |
| Round-13 finding, investigated and rejected: a reviewer flagged §9's backfill filter (`contact_id IS NOT NULL`) as potentially wrong on the theory that unresolved addresses are marked with the `uuid.Nil` sentinel rather than SQL `NULL`, which would make the filter a no-op | **Not a defect** — verified against `dbhandler.AddressCreate`'s actual code and its own comment (`address.go:112-117`): `uuid.Nil.Bytes()` is 16 zero bytes, "NOT SQL NULL," so the write path explicitly passes Go `nil` (→ SQL `NULL`) whenever `ContactID == uuid.Nil`. `contact_id IS NOT NULL` correctly excludes unresolved rows at the database level; the reviewer's theory conflated the Go-level zero-value sentinel with the DB-level column semantics, which the codebase itself takes explicit care to keep distinct. |
| Round-14 BLOCKER: `ContactDelete`'s ownership-period closure had no transaction/lock-ordering/deadlock-retry discipline specified, and its own membership-snapshot read only locks the targets it finds — a concurrent `ClaimAddress` committing a new open period after that read but before commit is invisible to it, leaving an orphan open period under a Contact that no longer exists | **Fixed, with a round-19 correction** (§4: generalized §5.2's ascending lock-ordering rule to arbitrary N targets, reused §5.3's retry loop; added a post-lock membership re-check that repeats until a pass finds no new targets. Round-19 later found the re-check's late-added locks are NOT actually a fixed-total-order guarantee — see the round-19 row below; the re-check is a retry-bounded mitigation, not a structural elimination) |
| Round-14 BLOCKER: §9's backfill rule didn't account for pre-existing data corrupted by the A9-b bug — a live `contact_addresses` row under an already-soft-deleted Contact would backfill as a permanently-open period with no future `ContactDelete` left to close it | **Fixed** (§9: backfill joins `contact_contacts` and closes the period at the Contact's `tm_delete` instead of leaving it open, for rows under an already-deleted Contact) |
| Round-15 finding: round-14's `tm_delete`-closing backfill branch didn't guard against `contact_addresses.tm_create > contact_contacts.tm_delete` (the true A9-b-corrupted case — an address added *after* its Contact was already deleted), which would silently insert an inverted `valid_from > valid_to` period that no interaction-matching query could ever match, permanently and silently losing that address's entire history | **Fixed** (§9: the `tm_delete`-closing branch now requires `tm_create <= tm_delete`; rows that fail this check backfill as a zero-length already-closed period `valid_from = valid_to = tm_delete` instead of an inverted range) |
| Round-15 BLOCKER: this plan never addressed the rolling-deploy version-skew window — an old-binary pod processing `AddressCreate` during a migrate-then-roll-pods deploy would create no ownership period, letting later `AddressDelete`/`ClaimAddress` calls silently no-op / misclassify as first-ever registration, reproducing defect #2 via a deploy-timing gap | **Fixed** (§9: added an explicit deployment-ordering constraint — this migration must ship as schema-and-binary-together, not the ordinary "migrate then roll pods old→new whenever" pattern, since the old binary is incompatible with the new schema being present) |
| Round-15 note (recorded, not blocking): round-14's `ContactDelete` membership re-check loop has no explicit pass-count cap, so a sustained stream of concurrent `ClaimAddress` calls against new targets could keep it re-checking across many passes | Acknowledged as a liveness/latency concern, not a correctness one (each pass still converges toward a fixed point); not given an arbitrary cap because that would trade a rare long-transaction case for a new, precedent-less correctness question (what disposition applies if the cap is hit without converging). |
| Round-16 finding: §6.2's claim that STEP2's time-range condition "mirrors the existing OR-expansion shape... no new query pattern" doesn't hold up — `AddressPair`/`sq.Eq` only carry pure equality, no field for `valid_from`/`valid_to`, and `sq.Eq` cannot express a range condition; STEP2 cannot be implemented by passing more `AddressPair` values into the existing `InteractionList` call | **Fixed** (§6.2: concrete `OwnershipPeriodBound` type and OR-builder added, additive alongside `AddressPair` — only STEP2's call site changes, `interaction_read.go:285`'s single-address re-fetch and all existing tests/mocks are unaffected) |
| Round-16 BLOCKER: round-15's "schema-and-binary-together, drain-then-replace" deployment-ordering fix for the rolling-deploy version-skew window is not enforceable — `bin-dbscheme-manager`/`bin-contact-manager` are independent CI/CD workflows with no `requires` dependency, schema migrations are applied manually outside CI/CD, and the k8s Deployment has no explicit `strategy` (inherits Kubernetes' default surge-first `RollingUpdate`, the opposite of what the fix assumed) | **Fixed** (§9: replaced the unenforceable deploy-ordering instruction with a code-level defense — the ownership-period-closing `UPDATE` gets the same `RowsAffected` check B5 established, now incrementing a Prometheus counter on a miss instead of silently no-op'ing; the residual gap (Step 5 can't distinguish "never owned" from "owned pre-migration with no period") is accepted as a bounded, monitored blast radius limited to the rollout window, not solved) |
| Round-17 finding: §6.2's `sq.GtOrEq{"col": sq.Expr(...)}` construction is not valid squirrel usage — verified against the vendored source that `Eq`/`Lt`/`GtOrEq` only special-case `driver.Valuer` values in their `toSql()`, with no branch for a `Sqlizer` in value position; this compiles but fails at query-exec time with an "unsupported type" error | **Fixed** (§6.2: rewritten to build the whole comparison as one `sq.Expr(...)` raw-SQL fragment with the ±∞ fallback embedded in the SQL string itself, the same pattern §6.3's `NOT EXISTS` subquery already uses, rather than assembling it from squirrel value-position helpers) |
| Round-17 finding: §9's Prometheus-counter fix for the version-skew defense (a) only named `AddressDelete`'s closing `UPDATE`, leaving the identical race in `AddressUpdate`'s old-target close and `ContactDelete`'s per-target close silently un-instrumented, and (b) claimed to be "consistent with an existing pattern" while `pkg/dbhandler` has zero Prometheus integration today (verified: no `prometheus` import in the package; existing metrics live only in `casehandler`/`listenhandler`/`subscribehandler`) | **Fixed** (§9: extended the `RowsAffected == 0` handling to all three closing-`UPDATE` sites, not just `AddressDelete`'s; specified `dbhandler` gaining its own `metricsNamespace` + `init()` block mirroring `casehandler`'s existing shape, package-local with no new upward dependency into the handler layer) |
| Round-18 finding: §6.2's claim that the new `InteractionList` parameter was "optional, defaulted when omitted, no existing caller/test changes" is not implementable — Go has no optional-parameter syntax, `since time.Time` is already the final parameter, and Go allows only one variadic parameter, which must be last | **Fixed** (§6.2: replaced with a new sibling function `InteractionListByOwnershipPeriods` sharing `InteractionList`'s query internals; `InteractionList` itself and every existing caller/test are genuinely unchanged) |
| Round-18 finding: §5.2's two lock-ordering rules (single-target-plus-`AddressResetPrimary`, and two-target-ascending-order) were each specified in isolation and never composed for the real, tested fourth combination — an `AddressUpdate` changing `target` and setting `is_primary=true` in the same call (`v1_contacts_update_test.go:33`) | **Fixed** (§5.2: added the composition rule — acquire every ownership-period lock the operation needs, in ascending target order if there are two, before `AddressResetPrimary`, before the `contact_addresses` write; not a third ordering rule, a direct composition of the existing two) |
| Round-19 BLOCKER: §4 Step 3's intervening-owner check said "strictly later than," leaving the tie-break operator (`>` vs `>=`) to the implementer — an exact-timestamp tie between a closed row's `valid_to` and another contact's row's `valid_from` (rare in production, easy to hit with a frozen test clock) would, under `>`, reopen the stale row unbounded and overlap the interim owner's legitimately-closed period, reproducing defect #2 through a boundary condition | **Fixed** (§4 Step 3: the comparison is now explicitly `>=`, treating an exact tie as an intervening owner and routing to the INSERT branch instead of reopening) |
| Round-19 finding: round-14's `ContactDelete` membership re-check loop claimed to "follow the same global ascending order" as §5.2's fixed-order rule, but a late-discovered target added to an already-partially-locked list is not actually re-sorted relative to locks already held — a concrete cross-transaction scenario reproduces the exact reverse-order deadlock class §5.2 exists to eliminate structurally | **Acknowledged, not solved** (§4: corrected the overstated claim — this re-check loop is a `maxDeadlockRetries`-bounded mitigation, not a structural elimination, the same standard already applied to every other deadlock this document accepts as a retry-bounded cost) |
| Round-19 finding (recorded, non-blocking): §6.2 didn't mention that `InteractionListByOwnershipPeriods` needs adding to the `DBHandler` interface and a `go generate` mock regeneration before it compiles, despite earlier sections being specific about comparable codegen/registration steps | **Fixed** (§6.2: added a paragraph naming the interface + mock-regeneration step explicitly, noting it's the existing repo-wide verification workflow, not a new one this design invents) |
| Round-20 BLOCKER: a fifth deletion entry point, `ContactDeleteByCustomerID` (invoked by `EventCustomerDeleted` when `bin-customer-manager` publishes `customer_deleted` — a real production path), bulk-soft-deletes every Contact of a customer without touching ownership periods, orphaning every open period those Contacts held — the same defect class round-13/14 closed for single-contact `ContactDelete`, through an entry point the document never knew existed. Round-12's caller sweep grepped `dbhandler.Address*` callers but never swept all paths that set `contact_contacts.tm_delete` | **Fixed** (§4: new table row + dedicated paragraph — the bulk path runs the same per-contact close-every-open-period transaction as `ContactDelete`, per contact sequentially, each in its own `BeginTx` with the same lock order/re-check/retry bound; mid-loop crash is acceptable because closure is idempotent and the event can be re-driven) |
| Round-20 finding: §8's round-13/14 `ContactDelete` rows still said plain "**Fixed**" with wording implying the same structural-elimination guarantee as §5.2, contradicting the round-19 correction in §4's body — a reader skimming only this table would over-trust the re-check loop's deadlock guarantee | **Fixed** (both rows now read "Fixed, with a round-19 correction" and point to the round-19 row) |
| Round-21 BLOCKER: §6.2/§6.3's time-range conditions compared bare `tm_interaction`, a documented-nullable column (origin events may omit TMCreate; the projection stores NULL) — under SQL three-valued logic every NULL-`tm_interaction` interaction would match no period at all, including the fully-unbounded one, silently vanishing from timelines (defect #1) and resurfacing in the unresolved queue (defect #3), a regression against today's pure-equality matching which returns those rows fine | **Fixed** (§6.2 SQL + Go fragment and §6.3 SQL all now compare `COALESCE(tm_interaction, tm_create)` — `tm_create` is projection-populated `NOW()`, `NOT NULL` — so a timestamp-less interaction is attributed by when this system recorded it instead of being unmatchable) |
| Round-21 finding: round-20's `ContactDeleteByCustomerID` fix reused the function's pre-delete SELECT id set for the period-closure loop, but that SELECT and the bulk `UPDATE` are separate non-transactional statements and the `UPDATE` has no `tm_delete IS NULL` filter — a Contact created in the gap gets soft-deleted while absent from the id set, its open periods permanently orphaned, and event re-drive cannot recover it (the SELECT's `tm_delete IS NULL` filter can't see it) — the round-14 stale-membership defect class resurfacing at the contact-set level | **Fixed** (§4 round-21 correction: bulk `UPDATE` runs first, then the loop's id set is collected post-delete via `WHERE customer_id=? AND tm_delete = ts`; also fixes the pre-existing cache-invalidation variant of the same race and makes event re-drive a genuine recovery path) |
| Round-21 verification: exhaustive grep of every path setting `contact_contacts.tm_delete` (only `ContactDelete` + `ContactDeleteByCustomerID`; `ContactUpdate`'s only caller whitelists fields and never passes `tm_delete`) and every `contact_addresses` write (only the five `dbhandler/address.go` functions; `ContactDeleteByCustomerID` never touches the table) | **Confirmed complete** — no third deletion path, no additional cascade-style address writer; §4's entry-point coverage is now exhaustive at both the contact and address levels |
| Round-22 BLOCKER: the `valid_from = tm_create` backfill (round-1's rule) would have erased currently-visible history at cutover — interactions arriving *before* Contact registration (the most common CRM flow: unknown caller first, registered afterwards) fail `>= valid_from` under the backfilled period, vanishing from timelines (defect #1) and re-entering the unresolved queue (defect #3), and the same registration made post-migration would get `valid_from=NULL` via Step 5, making attribution depend on which side of the migration it fell | **Fixed** (§9: backfill rule changed to `valid_from = NULL` — inert by construction, reproducing today's time-agnostic matching exactly at cutover; round-1's disposition row updated as superseded; round-15's inverted-range workaround retired as structurally impossible under NULL; §10's backfill-inertness open question resolved by construction) |
| Round-22 BLOCKER: §9's residual-gap paragraph claimed a "bounded blast radius (only the rollout window)" while a strictly larger unbounded gap sat unacknowledged — targets hard-deleted *before* the migration leave no row for the backfill, so their post-migration re-registration hits Step 5 and inherits the prior owner's entire pre-migration history (defect #2 itself), firing indefinitely, unfixable because hard-delete (defect #2's own root cause) destroyed the needed data | **Acknowledged, not solvable** (§9: added the honest scope statement — defect #2 is prevented for ownership changes made after migration; misattribution rooted in pre-migration deletions is unrecoverable-by-design, with `contact_resolutions` as the manual correction path; the "bounded" claim now explicitly covers only the skew-window gap) |
| Round-22 finding: round-21's `COALESCE(tm_interaction, tm_create)` fallback substitutes projection time for event time on timestamp-less interactions — under RabbitMQ at-least-once delivery with unbounded projection lag, an ownership boundary falling inside the lag misattributes the interaction to the projection-time owner (a bounded-population, unbounded-window defect-#2 variant) — and no disposition was recorded, violating this document's own every-gap-gets-a-disposition convention | **Acknowledged, accepted** (§9: recorded as an accepted trade — the alternative, leaving NULL-`tm_interaction` rows unmatchable, is a certain defect-#1 regression for the same population, strictly worse than conditional misattribution; `contact_resolutions` is the correction path) |
| Round-22 minor: §6.4's claim that the lookup index's "two range columns" prevent per-row filter scans is overstated — `valid_from`/`valid_to` appear inside COALESCE with outer-row columns mixed in, so they serve covering-only, not range pruning (real performance impact negligible: single-digit periods per target) | **Fixed** (wording corrected where §6.4 describes the index's role) |
| Round-23 finding: §9's "inert by construction" claim for the `valid_from=NULL` backfill contradicted the round-14 `tm_delete`-closing branch two paragraphs later — a closed `[NULL, tm_delete)` period stops suppressing post-deletion interactions today's time-agnostic `NOT EXISTS` wrongly hides, so those newly appear in the unresolved queue at cutover, an observable change | **Fixed** (§9: inertness claim explicitly scoped to the live-Contact branch; the deleted-Contact branch is recorded as intentionally not inert — the observable change IS the A9-b cleanup working) |
| Round-23 BLOCKER: the design would have permanently locked A9-b-corrupted targets out of every API path — the stale live row blocks `AddressCreate` (unique index) and `ClaimAddress` (unconditional `ErrConflict` on non-Nil owner), while the design's own new `TMDelete` guard removes the only existing recovery path (`RemoveAddress` on the deleted Contact), leaving manual DB surgery as the sole fix | **Fixed** (§9: the backfill hard-deletes each A9-b `contact_addresses` row after writing its closed period — attribution history lives in the period row; the target returns to a cleanly re-registrable state, exactly what a timely `RemoveAddress` would have produced) |
| Round-23 BLOCKER: the version-skew analysis covered only the missing-period direction; the reverse (new-binary opens a period, old-binary hard-deletes the address without closing it) leaves an orphaned OPEN period — indefinite Step-1 409 lockout on a target nobody owns, unbounded defect-#2 absorption into the stale owner's timeline, and a falsified "unreachable for `AddressCreate`" claim on Step 2 | **Fixed** (§4 Step 1 + §9 round-23 rule: Step 1 now verifies the blocking owner's `contact_addresses` row still exists inside the same locked transaction; if gone, the period is a skew orphan — closed on the spot with a Prometheus counter, converting permanent lockout into one-time self-healing; Step 2's unreachability claim is restored post-repair) |
| Round-23 note (recorded, non-blocking): the backfill's no-overlap/uniqueness safety silently depended on `idx_contact_addresses_identifier` (UNIQUE on a hard-delete table → max one source row per target) | **Fixed** (§9: dependency named explicitly so a future relaxation of that index is recognized as invalidating the backfill's assumptions) |
| Round-24 BLOCKER: round-23's self-healing rule specified existence-only checking ("row still exists"), but an orphaned slot can be re-occupied by `CreateUnresolvedAddress` (`contact_id=NULL`), which per round-10's exemption skips the step procedure and never triggers the check itself — a later `ClaimAddress` would see "a row exists," misjudge the orphan period as a genuine live owner, and return 409, silently reinstating the exact permanent lockout (plus unbounded defect-#2 absorption) round-23 claimed to have fixed; both reviewers independently converged on this defect, and §4's "blocking owner's row" prose vs §9's operational "row exists" spec was itself the implementer-discretion ambiguity class round-19 established as a BLOCKER | **Fixed** (§4 Step 1 + §9: the check is now ownership agreement — the row must exist AND its `contact_id` must equal the blocking period's `contact_id`; gone, NULL-owned, or differently-owned all mean orphan → close and continue; §4 and §9 wordings unified) |
| Round-24 verification: (a) Step 1's new `contact_addresses` FOR UPDATE follows the period-lock→address-read direction and no existing code path locks `contact_addresses` first (today's readers are all plain SELECTs; the five writers move inside the period-lock transaction under this design) — no reverse-order deadlock cycle; (b) false-positive orphan detection is structurally impossible because `AddressDelete` closes the period and deletes the row in one §5.1 transaction, so Step 1's locking read serializes against it and observes either pre-commit (period open + row present = genuine conflict) or post-commit (period closed = no Step-1 trigger), never the middle; (c) A9-b hard-delete cleanup leaves no `is_primary` or cache residue (uniqueness is a generated column on the row itself; tombstoned Contacts aren't cached, 24h TTL bounds worst-case) | **Confirmed** (recorded as verified-safe; the minor observation that Step 1's cross-owner row lock slightly widens the `AddressResetPrimary` AB-BA deadlock surface falls under §5.3's existing retry-bounded acceptance) |
| Round-25 BLOCKER: the round-23 backfill cleanup fixed the permanent-lockout state only for the pre-migration population — every post-migration `ContactDelete`/`ContactDeleteByCustomerID` recreates it fresh: the design closed the periods but left the tombstoned Contact's live `contact_addresses` rows in place, where they block `AddressCreate` (unique index) and `ClaimAddress` (non-Nil pre-check, no `tm_delete` inspection) while the new A9-b guard blocks the only recovery path (`RemoveAddress` on the deleted Contact), and Step 1's self-healing never fires because the periods are correctly CLOSED — every deleted Contact's phone numbers/emails would become permanently unregistrable via API; both reviewers independently converged on this defect (the round-14→21 / round-23→24 "fixed for one population, reproduced in another" failure mode again) | **Fixed** (§4: `ContactDelete`'s per-target transaction now hard-deletes the Contact's `contact_addresses` row alongside closing its period, and `ContactDeleteByCustomerID` inherits the same cleanup per contact; §3.1's "nothing touches contact_addresses differently" claim corrected to name its two deliberate cleanup exceptions) |
| Round-25 verification: round-24's ownership-agreement check introduces no false positives — (1) no committed state can show `period.contact_id ≠ address.contact_id` under the new binary (`AddressUpdate` is field-whitelisted and cannot write `contact_id`; the three ownership-writing paths all pair row and period writes in one §5.1 transaction; Step 1 reads under FOR UPDATE so transaction-internal intermediate states are invisible), (2) normal `ClaimAddress` on an unresolved row never reaches the check (no blocking open period exists), (3) a genuine live owner still yields exactly 409 | **Confirmed** (recorded as verified-safe) |
| Round-26 BLOCKER: round-25's row cleanup would silently strip `addresses` from the `contact_deleted` event payload and the `DELETE /v1/contacts/{id}` RPC response — today `contacthandler.Delete` re-reads the tombstone via the intentionally-unfiltered by-id `ContactGet` (per the service CLAUDE.md, exactly so the delete payload carries the full address list), and hard-deleting the rows before that re-read empties it (`omitempty` drops the field entirely), an unrecorded external contract change contradicting §3.1's "no read path changes" and §2's response-shape promise | **Fixed** (§4: `contacthandler.Delete` snapshots the Contact including addresses BEFORE `dbhandler.ContactDelete`, and the event payload/RPC response use the snapshot; `Test_Delete` added to §10's checklist) |
| Round-26 finding: round-25's row cleanup derived its target set from open-period membership, so a skew-created row with no period (§9's own accepted degraded state) would survive under the tombstone — permanent lockout reproduced in a third population, with Step-1 self-healing unable to fire (no period exists; `AddressCreate` dies at the unique index first): the "fixed for one population, reproduced in another" failure mode's third recurrence | **Fixed** (§4: row-cleanup membership changed to `contact_addresses WHERE contact_id = ?` — all live rows owned by the tombstoned Contact are cleaned regardless of period existence; period closure keeps its own open-period membership) |
| Round-26 verification: (2) period-based history reads are unaffected by row hard-deletes (closed periods carry the attribution; `interactionListByContact` already NotFounds tombstoned Contacts; `LookupByPhone/Email` already NotFounds tombstone resolutions today, so external lookup behavior is unchanged); (3) the row DELETE inside the per-target transaction follows §5.2's period-lock-first order (round-24's verified direction) and the N-target loop's larger lock footprint stays inside round-19's retry-bounded acceptance | **Confirmed** (recorded as verified-safe; `ContactDeleteByCustomerID` publishes no per-contact events, so the payload concern applies to single `ContactDelete` only) |
| Round-27 BLOCKER: round-26's pre-delete snapshot, used verbatim, would publish `tm_delete: null` (not `omitempty`) and stale `tm_update` in the `contact_deleted` event/RPC response — the same unrecorded-external-contract-change class round-26 fixed for `addresses`, this time losing the deletion timestamp | **Fixed** (§4 round-27 correction #1: snapshot carries `addresses`, then `tm_delete`/`tm_update` are overlaid with the deletion timestamp before publish/return; `Test_Delete` checklist item extended to both fields) |
| Round-27 finding: round-26's row-cleanup membership was a single read outside the round-14 re-check loop — a guard-passing `AddAddress` committing between that read and the tombstone would leave a live row under the tombstone (closed period + live row = the lockout's fourth population) | **Fixed** (§4 round-27 correction #2: each re-check pass re-reads BOTH memberships — open periods and live address rows — under lock until a pass finds nothing new; round-14's accepted residual window now has a working escape hatch via the round-27 Step-1 clause) |
| Round-27 finding: "per-target transaction" wording (rounds 25/26) contradicted round-13's single-`BeginTx` definition — a transaction-boundary ambiguity of the implementer-discretion class round-19 established as blocking | **Fixed** (§4 round-27 correction #3: canonical reading stated — ONE §5.1 transaction locks all targets ascending, closes periods, deletes rows, tombstones the Contact, commits atomically; deadlock retry restarts the whole cycle idempotently) |
| Round-27 BLOCKER: §9's skew analysis covered only address-write skew; the deletion entry points became a skew surface themselves once round-25/26 gave them new-binary-only behavior — an old-binary `ContactDelete` during the rollout window leaves live rows + OPEN periods under a tombstone, and round-24's ownership-agreement check *passes* on that state (row exists, contact_id matches), misjudging the dead owner as live: permanent 409 lockout in a fourth population, plus a period-less variant (old-binary A9-b `AddAddress` onto a tombstone) that never reaches Step 1 at all | **Fixed, repair actions corrected in round-28** (§4: Step 1's ownership agreement now also requires the owning Contact to not be tombstoned; tombstoned-owner states are repaired via the round-28 late-cleanup rules; `ClaimAddress`'s pre-check gains the same clause; `AddressCreate` repairs on duplicate-key errors; all repairs share the skew-orphan Prometheus counter) |
| Round-28 BLOCKER: round-27's repair (a) told `ClaimAddress` to hard-delete the stale row — destroying the very row the claim's own final UPDATE targets (`WHERE id=? AND contact_id IS NULL`, RowsAffected==0 → ErrConflict, the preserved B5 guard), so the "self-healing" claim would always fail 409 and leave the caller's addressID a 404 on retry | **Fixed** (§4 round-28: `ClaimAddress` resets the stale row to `contact_id = NULL` instead of deleting it — its final UPDATE then finds exactly the NULL-owned row its race guard expects; Step 1's clause text updated to the caller-specific repair) |
| Round-28 BLOCKER: round-27's repair (b) on the period-less variant deleted the row without writing any period — the retry then hit Step 5 (`valid_from=NULL`) and the new owner absorbed the dead Contact's entire era, defect #2 manufactured by the cleanup itself, and inconsistent with §9's backfill which handles the identical data state by writing a closed `[NULL, tm_delete)` period BEFORE deleting the row | **Fixed** (§4 round-28: every late-cleanup site writes the closed period FIRST (idempotent under retries), then repairs the row — the retry routes through Step 4, not Step 5) |
| Round-28 finding: the three tombstone-check reads added `contact_contacts` to address-write transactions with no lock-order rule — an implementer locking it first would create an AB-BA surface against `ContactDelete` (which touches periods → rows → `contact_contacts` last), the implementer-discretion class round-19 established as blocking | **Fixed** (§4 round-28: `contact_contacts` is read LAST in every address-write transaction, plain read (tombstones only ever appear, never disappear — no undelete path per round-21's sweep), and §5.2's order now names it as the final table) |
| Round-28 BLOCKER (bundled bug, production-only): `dbhandler`'s duplicate-key classifier matches the literal `idx_contact_addresses_cust_type_target`, but the production index (Alembic) is `idx_contact_addresses_identifier` — the classifier never fires on production MySQL (only the SQLite test schema uses the old name: tests green, production returns a generic 500 instead of 409), and round-27/28 repair (b)'s trigger depends on that classification | **Fixed, bundled** (§4 round-28: match strings corrected to the production names (`identifier`, and sibling `idx_contact_addresses_primary`), SQLite test schema aligned with production so the drift cannot silently recur — same bundling standard as A9-b/B5) |
| Round-29 BLOCKER: round-28's fabricated `[NULL, tm_delete)` late-cleanup period imported §9's backfill bounds into a context where their precondition (empty periods table per target) no longer holds — a `valid_from=NULL` row can never satisfy Step 3's `valid_from >= closed.valid_to` intervening-owner test, so it is invisible to Step 3 and lets a previous owner's period reopen across the dead owner's era (defect #2 via Step 3, manufactured by the cleanup) | **Fixed** (§4 round-29: fabricated period bounds are `[latest existing valid_to for the target (NULL only if no periods exist), tm_delete)`, with round-15's zero-length disposition when the tombstone predates the last era — Step 3's `>=` test now sees the fabricated era) |
| Round-29 finding: the round-28 covering predicate ("skip if a covering closed period exists") was undefined — the implementer-discretion ambiguity class — and the Step-1 clause vs repair (a) specified conflicting period-write orders that could leave two overlapping closed periods for one era | **Fixed, with a round-31 correction** (§4 round-29: covering predicate defined — a closed period for the same tombstoned contact_id with `valid_to >= tm_delete`, evaluable from the §4 locked fetch; canonical late-cleanup order — close orphan open period first, row repair last. Round-31 falsified two of this round's parenthetical claims: the close does NOT necessarily satisfy the predicate, and fabrication is NOT limited to the period-less variant — in mixed skew (orphan owner ≠ tombstoned row owner) both the close and the fabrication run; the predicate is always evaluated after the close. See the round-31 rows below and §4's refined ordering rule) |
| Round-29 BLOCKER: §4 Step 2's "genuinely unreachable for `AddressCreate`" claim contradicted §9's own round-23(c) analysis (skew deletes the row but leaves the period open — the unique index no longer blocks), and the same-owner variant of round-24's slot-reoccupation had no check at all: Step 1's self-healing only fires for a *different* contact's period, so B re-registering its own orphaned target routed to Step 2, which had no defined `AddressCreate` behavior (a literal reading returns 200 without inserting the row) and an ambiguous `ClaimAddress` no-op | **Fixed** (§4 round-29: Step 2 performs the same ownership-agreement check as Step 1 — row agrees = true idempotent no-op; row missing = repair-in-place (AddressCreate inserts the row and keeps the existing open period; ClaimAddress claims the reoccupied NULL-owned row and keeps the period); the reachability claim is corrected) |
| Round-29 verification: (1) ClaimAddress's `contact_id=NULL` reset is transaction-internal (never observable, fully rolled back on failure, no event/cache impact — post-commit paths fire only on success); (3) a third party slipping between the cleanup commit and the retry INSERT yields the same 409 the caller would have received had the third party arrived first — consistent with today's semantics | **Confirmed** (recorded as verified-safe) |
| Round-30 BLOCKER: Step 2's round-29 "row agrees" branch defined only `ClaimAddress`, and a literal first-match-stops reading would turn `AddressCreate`'s everyday "you already own this target" duplicate from 409 into a 200 no-op — an API contract change contradicting §2's status-code promise, and self-contradictory with the "row disagrees" branch's unique-index reasoning | **Fixed** (§4 Step 2: row-agrees is now caller-specific — `ClaimAddress` keeps the idempotent no-op; `AddressCreate`/`AddressUpdate` return `ErrDuplicateTarget`/409 exactly as today) |
| Round-30 BLOCKER: `AddressUpdate` (target changed) is a first-class step-procedure caller per §4's own table, but every caller-specific repair enumeration (Step 1's row repair, Step 2's outcomes, round-28's (a)/(b)) named only `AddressCreate`/`ClaimAddress` — the round-19/24/27 implementer-discretion class again, with a wrong-choice failure mode (NULL-reset would leave the slot occupied and 1062-fail `AddressUpdate`'s own re-targeting UPDATE); additionally the 1062 classifier exists only in `AddressCreate`'s path, so the period-less variant on an `AddressUpdate` target surfaces as a generic 500 with no repair trigger at all | **Fixed** (§4: `AddressUpdate` (new-target side) enumerated everywhere — Step-1 repair = hard-delete (like `AddressCreate`), Step-2 outcomes defined; round-28 bundled fix extended: the 1062+index-name classification and the tombstoned-owner-check-then-retry-once behavior are added to `AddressUpdate`'s target-change path) |
| Round-30 finding: the round-29 fabricated-period bounds (`valid_from` = latest existing `valid_to`) over-attribute the unowned gap between the previous owner's era and the dead Contact's actual registration — in the worst case (no periods + post-tombstone A9-b row) the entire fabricated era `[NULL, tm_delete)` covered time the dead Contact never owned; the stale row's `tm_create` is a strictly better lower bound available for free in the same transaction | **Fixed** (§4 round-30: `valid_from = GREATEST(latest valid_to, stale row's tm_create)` — head gap collapses to zero by construction, Step 3 visibility preserved; zero-length fallback re-anchored at `tm_create`; NULL only in a defensive default not producible by today's write paths) |
| Round-30 finding: Step 2's repair-in-place keeps the contact's own orphan open period, permanently attributing the skew-deletion→re-registration gap to that contact, with an arrival-order asymmetry (a third party arriving first truncates the gap at NOW()) — undisposed | **Fixed** (§4 Step 2: recorded as an accepted disposition — the orphan is the contact's own and the asymmetry is inherent to repair-on-next-touch, same bounded-window standard as §9's other skew dispositions) |
| Round-30 verification: Step 2's ownership-agreement read follows §5.2's canonical order (periods FOR UPDATE → addresses read, and Step 2 needs no `contact_contacts` read at all — the owner under check is the caller itself, whose liveness the A9-b guard already established) | **Confirmed** (recorded as verified-safe) |
| Round-31 BLOCKER: §4 Step 1's prose applied the round-28 late cleanup to all three orphan causes unconditionally — in the row-gone branch the cleanup's operations are undefined (no row, no tm_delete, no tm_create), and in the owned-by-anyone-else branch a LIVE contact's period-less skew row (§9's own accepted degraded state) would have been hard-deleted and its target stolen by the caller — live-data destruction worse than the defects the design fixes; §9's round-23/24 rule text (close-and-continue only) also still contradicted §4's round-27/28 refinements | **Fixed** (§4 Step 1: late cleanup scoped to the tombstoned-owner branch only; row-gone = close and continue is the whole action; owned-by-anyone-else = close and continue, letting the round-28 duplicate-key path arbitrate the occupying row (repair if dead owner, 409 if live); §4 declared canonical over §9's older rule text) |
| Round-31 finding: the fabricated-period inversion fallback's parenthetical characterized the trigger as `tm_create > tm_delete` (the A9-b case) only — but in mixed skew (closing another owner's orphan at NOW() pushes latest `valid_to` past `tm_delete`) the inversion arrives through the GREATEST's other arm with `tm_create < tm_delete`; coding the parenthetical as the condition would insert the inverted range round-15 prohibited | **Fixed** (§4: the trigger is stated generally — test the computed bound `GREATEST(...) > tm_delete`, either arm; the A9-b case is labeled as only one cause) |
| Round-31 finding: round-29's "closing the orphan necessarily satisfies the covering predicate, so fabrication is skipped" is false in mixed skew where the orphan's owner (B) differs from the stale row's tombstoned owner (D) — closing B's orphan records B's era, not D's, and both the close AND the fabrication must run | **Fixed** (§4 round-29 ordering rule refined: the covering predicate is always evaluated after the close (for the tombstoned row-owner), never skipped on the strength of the close; disjointness of the two periods holds by construction of the GREATEST bound) |
| Round-31 BLOCKER: the round-12/13 compensating cleanup reused `AddressDelete`'s close-at-NOW() rule on periods the failed `ContactCreate` loop itself had just inserted — leaving a ghost Contact holding a closed `[NULL, NOW())` period, so the caller's retry routed through Step 4 instead of Step 5, permanently orphaning the target's pre-registration history (defect #1 remade) while displaying it on the ghost's timeline (defect #2 remade), directly falsifying the paragraph's own "converge back to no addresses were added"/"fresh attempt" claims — unverified combination of two rules defined 18 rounds apart | **Fixed** (§4: compensating cleanup DELETEs the period rows its own loop inserted (identifiable in-transaction: brand-new contact_id can only own the period it just created — Step 3 reopen unreachable), restoring the true no-rows state; recorded as the single sanctioned exception to "the period row is never deleted," with the brief-window interaction disposition stated) |
| Round-31 verification: (1) same-transaction DELETE-then-UPDATE slot re-occupation is InnoDB-safe (own delete-marks don't collide in the unique check), and Step 2 row-missing re-targeting composes correctly with the §4 two-target rule (old-target period already closed by table order, round-17's RowsAffected counter covers the skew-missing case) | **Confirmed** (recorded as verified-safe) |
| Round-32 finding: Step 1's other-owner branch described the orphan close as a completed repair, but with a LIVE occupant every reachable caller path ends 409-and-rollback in the same transaction — the close never commits, so mixed-skew with a live occupant is never healed by other parties' attempts, and the non-transactional Prometheus counter would count each futile rolled-back close as a repair; repair (b)'s transaction placement relative to the poisoned 1062 transaction was also unspecified | **Fixed, with a round-33 correction** (§4: transaction boundary clarified — the rollback is correct (no partial state escapes); recorded disposition: the state heals via the live occupant's own next period-closing touch (round-33 corrected the healing set to exactly {`AddressDelete`, target-changing `AddressUpdate`} — the "hash-based close / Step 2 ClaimAddress" wording here was partly false, see the round-33 row), bounded-by-next-touch standard; the skew-orphan counter increments only post-commit; repair (b) runs in a NEW §5.1 transaction after the poisoned one rolls back, retry in a third) |
| Round-32 finding: the round-12/13 compensating-cleanup prose still carried three pre-round-31 artifacts — the round-13 paragraph still specified "ownership-period Step-based closure" and a shared-factoring claim ("both call sites need") that round-31 falsified (an implementer unifying both paths behind one closing helper reproduces the ghost-period bug), the failure-disposition text still described the close-at-NOW regime, and a "compensating RemoveAddress" naming remnant survived | **Fixed** (§4: round-13 paragraph rewritten — paths share only the row-delete portion, the period operation is deliberately different per caller (close for `RemoveAddress`, DELETE for compensation) and must not be unified; failure disposition updated for the DELETE regime — a failed cleanup leaves an open period + live row under a ghost Contact, retry meets 409 until the ghost is deleted via the standing `ContactDelete` repair path, with the ghost's id available from the log line — round-34 correction: the earlier "create response/log line" wording here overstated it, the response body carries no id per Go's `(nil, err)` convention (round-33's §4 correction), the log line is the sole canonical source) |
| Round-32 finding: §8's round-29 ordering-rule row still asserted "(which then satisfies the predicate)" and "fabricate only for the period-less variant" — both falsified by round-31 — without a correction pointer, violating the document's own table-correction standard (the round-13/14/20 precedent) | **Fixed** (§8: round-29 row now carries "Fixed, with a round-31 correction" and states both falsified claims explicitly) |
| Round-33 finding: the round-32 mixed-skew healing disposition named `ClaimAddress` flows as a healing path "through Step 2" — doubly false: with the other contact's orphan open, the procedure necessarily hits Step 1 (never Step 2), and `ClaimAddress` against the live occupant's row is rejected by its own pre-check/final-UPDATE guard with full rollback; the true healing set is exactly {live occupant's `AddressDelete`, `AddressUpdate` with target change} | **Fixed** (§4: healing set enumerated exactly, all non-healing touches named as such; hash-close healings noted as intentionally uncounted, and the "all repairs increment the counter" claim rescoped to Step-1/duplicate-key-path repairs) |
| Round-33 finding: the round-32 post-commit counter rule was applied only to the skew-orphan counter — the §9 round-16/17 `RowsAffected==0` counters have the identical inflation sources (up to 3x deadlock-retry duplication; rolled-back `AddressUpdate` old-target close misses) but were left in-transaction, the same defect class handled inconsistently | **Fixed** (§4: post-commit rule promoted to ALL of this design's in-transaction instrumentation) |
| Round-33 finding: round-32's ghost-recovery path (`ContactDelete`) CLOSES the ghost's period, terminating in exactly the data shape round-31 declared defective on the success path (closed ghost-owned `[NULL, NOW())` period → retry mis-routes to Step 4, pre-registration history permanently hidden), with no disposition for the post-recovery residue — the "two rules defined rounds apart, combination unverified" failure mode again | **Fixed, with a round-34 correction** (§4: disposition recorded as an accepted limitation — reachable only via double failure (create failed AND cleanup failed), deliberately not re-engineered; manual correction via `contact_resolutions` (§7); §9.x gains a reconciliation query — round-34 demoted it from "exact residue detector" to heuristic candidate list (see the round-34 rows); also corrected: the create response body does NOT carry the ghost id (`(nil, err)` convention) — the log line is the canonical identifier source) |
| Round-34 BLOCKER: the round-33 §9.x reconciliation query was doubly wrong — its predicate is in-band indistinguishable from a NORMAL `ContactDelete`'s data shape (false-positive flood: essentially every deleted-and-not-reregistered Contact), and its `succ.id IS NULL` condition dropped a ghost residue at exactly the moment its damage materializes (the retry's Step-4 period IS a successor — false negative on the harmful state); the "surfaces exactly these residues" claim was self-contradictory with the round-33 scenario it was written for | **Fixed** (§9.x rewritten: exactness claim withdrawn — no complete in-band ghost discriminator exists (the defining fact lives in the event stream/logs); the log line is canonical, the query is demoted to a heuristic candidate list (short Contact lifetime filter, no successor condition so the harmful state stays in the report, `succ.id != p.id` pitfall documented); §4's cross-reference updated to match) |
| Round-34 finding: the round-33 post-commit instrumentation rule had no implementation placement — §9 round-17 fixes the counters as dbhandler package-local, but the §5.1 transaction owner is the layer that observes commit, and the natural dbhandler-internal immediate increment is exactly the forbidden form (implementer-discretion ambiguity with a wrong default) | **Fixed** (§4: placement specified — the transaction-owning dbhandler entry point accumulates pending increments in a call-scoped struct and flushes to the package-local counters only after `tx.Commit()` returns nil; deadlock retries discard the pending struct with the rolled-back attempt; no cross-layer plumbing needed since the dbhandler entry point owns the transaction per §5.1) |
| Round-34 finding: §8's round-32 row still said the ghost id is "available from the create response/log line" — the create-response half was falsified by round-33's own §4 correction, without a pointer | **Fixed** (the row now carries the round-34 correction: response body carries no id per `(nil, err)`, log line is the sole canonical source) |
| Round-35 BLOCKER: Step 1's `ErrConflict` sentinel had no contacthandler mapping for `AddAddress` (only an `ErrDuplicateTarget` branch exists today) or `contacthandler.Create` (no branches at all) — a cross-owner POST that returns 409 today via the unique index would return 500 under the design, violating §2's status-code promise; the exact B5/round-2 failure class ("dbhandler sentinel fixed, handler mapping missed") reproduced by the design's own new path, at the unverified intersection of rules from rounds 7-8, 2, and 11 | **Fixed** (§5.4: `AddAddress` and `Create`'s address-loop error path gain the `ErrConflict → cerrors.AlreadyExists` branch; reusing `ErrDuplicateTarget` as the Step-1 sentinel was considered and rejected — the repair logic must distinguish period conflicts from unique-index collisions; full mapping roster enumerated) |
| Round-35 finding: the round-34 pending-metrics rule said "scoped to that call" — ambiguous for the two entry points where one dbhandler call owns multiple transactions: repair (b)'s T1/T2/T3 sequence (T2's committed repair increment could be lost if T3 fails, or double-flushed) and `ContactDeleteByCustomerID`'s per-contact transactions | **Fixed** (§4: pending struct scoped per TRANSACTION (per `BeginTx` attempt) — created with each BeginTx, flushed immediately after that transaction's commit, discarded on its rollback; per-case consequences spelled out for repair (b), `ContactDeleteByCustomerID`, and `ContactDelete`'s single N-way transaction) |
| Round-35 finding: the round-13 paragraph (as rewritten by round-32) still named `dbhandler.AddressDelete` as the compensating-cleanup entry point while simultaneously forbidding its close-at-NOW() period semantics — a function-name vs semantics contradiction that would reproduce the round-31 ghost-period bug if followed literally | **Fixed** (§4: the compensating path is a separate dbhandler entry point (`AddressDeleteCompensating`) that hard-deletes the row AND hard-DELETEs the loop's own period in one §5.1 transaction, publishing nothing; `AddressDelete` is explicitly NOT the callee) |
| Round-35 finding (recorded, non-blocking): §9.x's 60-minute lifetime filter models tm_create→tm_delete as the failed request's duration, but for recovered ghosts tm_delete is the OPERATOR's recovery time — a recovery later than the window drops the harmful residue from the report; and un-recovered ghosts (no tombstone, open period) match neither predicate, so the query covers only recovered ghosts | Recorded in §9.x as an honest scope note: the query is a lossy heuristic over recovered ghosts only; the log line remains the sole canonical identifier for both states, and the operator instruction is to tune/widen the interval when reviewing after delayed recoveries |
| Round-36 BLOCKER: §6.3's periods-only `NOT EXISTS` silently removed the unresolved-row suppression effect — today's subquery has no `contact_id` filter, so `CreateUnresolvedAddress` rows (`contact_id = NULL`, never given a period per §3.1/round-10) suppress matching interactions from the unresolved queue; the wholesale replacement would re-surface all of them at cutover and permanently strip the endpoint's only queue-side observable effect — an unrecorded external behavior change in a never-enumerated population (the round-13 backfill-filter verification checked the write side only, never this read-path consequence) | **Fixed** (§6.3: a second `NOT EXISTS` over `contact_addresses ... contact_id IS NULL` preserves today's suppression exactly — unresolved rows suppress by presence (time-agnostic), owned addresses by period (time-aware); index coverage stated for both subqueries; §10's checklist covers the cutover fixture) |
| Round-36 finding: §5.4's round-35 "the same branch" wording steered implementers to copy `ClaimAddress`'s `ADDRESS_ALREADY_CLAIMED` reason code — but the user action reaching `AddAddress`'s new branch returns `ADDRESS_ALREADY_EXISTS` today, and the 2026-07-02 design deliberately separated those codes as an external contract: an unrecorded reason-code change behind an identical 409 | **Fixed** (§5.4: `AddAddress`/`Create` map Step-1 `ErrConflict` with reason code `ADDRESS_ALREADY_EXISTS` and its existing message — byte-for-byte payload preservation; `ClaimAddress` keeps `ADDRESS_ALREADY_CLAIMED`) |
| Round-36 finding: the round-35 `AddressDeleteCompensating` entry point was absent from the DBHandler-interface/mockgen list and §10's checklist, and its internal lock order was only implicit — violating the document's own round-19 codegen-explicitness standard | **Fixed** (§4: canonical order stated explicitly (period FOR UPDATE → row delete → period DELETE, no `contact_contacts` read needed), interface membership + mock regeneration named, §10 checklist updated) |
| Round-37 BLOCKER: claiming an unresolved row on a target with prior ownership history permanently orphaned the very interactions the claim was performed to attach — Step 4's `valid_from=NOW()` (written for `AddressCreate` reassignment) combined with round-10's no-period exemption and round-36's presence-based suppression meant the claim simultaneously removed the queue suppression (row no longer NULL-owned) and created a period starting only at claim time, leaving the unresolved-era interactions outside every period AND back in the unresolved queue; the same user action silently diverged by target history (Step 5 claims attributed everything, Step 4 claims attributed nothing), and §6.3's "matching today's observable sequence" claim was false for exactly these paths — three rules defined rounds apart (10, 5, 36), combination unverified | **Fixed, with a round-38 correction** (§4 Step 4 and Step 3-INSERT: `valid_from` is now caller-specific — `AddressCreate`/`AddressUpdate` keep `NOW()`; `ClaimAddress`'s bound was set here to `GREATEST(latest closed valid_to, claimed row's tm_create)` but round-38 falsified that bound (pre-`tm_create` gap exposed, repair-(a) `tm_create` meaningless) and corrected it to `latest closed valid_to` — see the round-38 row; §10 gains the claim-of-unresolved fixture) |
| Round-37 finding (recorded, non-blocking): §6.3's "covers" wording for the unresolved-row subquery overstated — `contact_id` is not in `idx_contact_addresses_identifier`, so the lookup is index-driven plus one row fetch (at most one row by uniqueness), not covering; corrected per the document's own §6.4 round-22 precision standard | **Fixed** (§6.3 wording corrected) |
| Round-38 BLOCKER: the round-37 `GREATEST(latest valid_to, row tm_create)` bound was doubly wrong — (a) the pre-`tm_create` gap (previous era's end → row creation) stayed outside every period, so its interactions resurfaced in the unresolved queue AND dropped off the claimer's timeline (today's time-agnostic value-match attributes them to the claimer), while §6.3 falsely labeled this "not a behavior change"; (b) via round-28 repair (a) (dead owner's row reset to NULL mid-transaction), `tm_create` is the DEAD contact's registration time — the claimer would silently absorb the dead owner's post-tombstone vacancy, contradicting round-30's own head-gap standard | **Fixed** (§4: `ClaimAddress`'s bound corrected to `valid_from = latest closed valid_to` — covers the entire unowned span since the previous era, reproducing today's attribution exactly on both queue and timeline surfaces; trivially non-overlapping; the repair-(a) path loses its dependence on `tm_create` entirely; mixed-skew clamp recorded under the round-23 attribution-up-to-repair standard; §6.3 rescoped, §10 fixture extended to the pre-row-creation gap) |
| Round-38 verification: (1) the corrected bound is consistent with Step 3's `>=` tie-break and the no-overlap invariant (half-open disjoint, same-instant tie resolves to the claimer, later re-registration still sees the claimer as intervening owner); (3) `ClaimAddress` can reach Step 3-reopen (own closed period, no intervener) and the reopened `[old valid_from, ∞)` period covers the pre-claim era — suppression continuity holds on that branch too | **Confirmed** (recorded as verified-safe) |
| Round-39 BLOCKER: multi-era gap resurfacing — with two or more prior eras, `valid_from = latest closed valid_to` covers only the gap immediately preceding the claim; an older, non-adjacent gap (e.g. A-to-B, before a B-to-claimer gap) stays uncovered and resurfaces in the unresolved queue once the claim removes today's unconditional row-presence suppression — falsifying §4/§6.3/§8's prior "exactly / nothing resurfaces / one exception" claims, which had enumerated only the mixed-skew clamp | **Fixed** (§4: recorded as an accepted, not-fixed-further limitation — reaching further back would re-admit round-30's rejected over-absorption of other owners' history; §6.3 rescoped to name two exceptions instead of one; §10 gains a two-prior-era fixture asserting the older gap resurfaces) |
| Round-39 finding (non-blocking, recorded): Step 3-INSERT's "latest closed valid_to for this target" could be misread by an implementer as "this contact's own closed period's valid_to" rather than the target-wide max — both readings happen to be safe by construction here (the target-wide max structurally equals the intervening owner's valid_to), but the ambiguity itself matches the document's own round-19 implementer-discretion standard | **Fixed** (§4 Step 3-INSERT: the target-wide-max reading stated explicitly, with the structural proof that it equals the intervening owner's valid_to and cannot regress to an earlier era) |
| Round-39 BLOCKER: the pre-lock target-resolution rule (round-7/8/9) was enumerated only for `AddressClaim` — `AddressUpdate` (target change) and `AddressDelete` also resolve `(type, target)` from a pre-lock row read (`contact.go:354` for `UpdateAddress`; `AddressDelete` needs the row's own target to compute `open_period_uk`) and are equally exposed to the same stale-read hazard: a concurrent re-target racing a stale-read close/update leaves a live open period orphaned under a live Contact, same-binary, no skew window required — the round-30 enumeration-gap failure mode's 6th recurrence | **Fixed** (§4: the round-7/8/9 compare-and-retry (re-read after lock, retry on mismatch, abort with `ErrNotFound` on NotFound, reusing §5.3's retry cap) extended explicitly to `AddressUpdate` and `AddressDelete`; §10 gains mismatch-retry/NotFound-abort fixtures for both, mirroring `AddressClaim`'s existing coverage) |
| Round-39 finding (recorded, non-blocking): `AddressResetPrimary` and `AddressCreate`/`AddressUpdate` are separate `DBHandler` interface calls without a shared transaction parameter, yet §5.1 requires one `BeginTx` per outer operation — the mechanical integration approach is unspecified | Recorded as a §10 open question — plumbing detail with no attribution-correctness implication, tracked rather than blocking |
| Round-40 finding: `AddressUpdate`'s round-39 compare-and-retry said only "abort and retry with the fresh old-target," not specifying full-transaction restart vs. a partial retry that keeps the already-acquired new-target lock — the latter reading would reintroduce the exact two-target reverse-order deadlock class §5.2 exists to prevent; and §5.3's round-8 `maxDeadlockRetries` text named only `AddressClaim`, never updated to cover round-39's two new retry points — the same "fixed in one place, not propagated to sibling paths" class the document named for itself at round-17 | **Fixed** (§4: `AddressUpdate`'s retry stated explicitly as a full-transaction restart, with the two-target deadlock argument spelled out; §5.3: the cap generalized to cover every stale-target-mismatch retry §4 defines, one shared budget) |
| Round-40 finding: whether the round-39 `AddressUpdate`/`AddressDelete` retry makes B5's `RowsAffected == 0` guard unreachable (dead code) was left unstated | **Fixed** (§4: recorded as two deliberately non-overlapping layers — the retry covers same-binary staleness (serialized away by the periods lock once the retry succeeds), the guard covers an old-binary pod deleting the row in the mixed-version window between the post-lock re-read and the final write, which the new-binary retry logic cannot see) |
| Round-40 BLOCKER: §6.3's two `NOT EXISTS` disjuncts (periods, unresolved-row presence) both vacuously match a live-owned `contact_addresses` row that has zero period rows — the missing-period skew state §9 round-16/17 already accepts as reachable via an old-binary pod's `AddressCreate` — re-surfacing ALL of a normal, currently-owned, never-deleted address's interactions in the unresolved queue; each guard was written for its own population and the owned-without-periods combination (which exists only because of §9's accepted skew window) was never checked against this query, the same population-vs-query-enumeration gap round-36 found one population over | **Fixed** (§6.3: a third `NOT EXISTS` suppresses by presence any owned row with no period rows at all, handing off to the periods disjunct once the next touch gives the row a period — same pattern as the second disjunct's claim handoff; index coverage restated for three subqueries; §10 gains the missing-period-skew fixture) |
| Round-41 BLOCKER: STEP1's round-40-adjacent gap — `interactionListByContact`'s STEP1 (`OwnershipPeriodsListByContactID`) is period-only, so the same owned-but-period-less skew population §6.3's round-40 fix covers on the queue side contributes NO bound at all to STEP2's OR-list, vanishing every one of its interactions from the OWNING Contact's own timeline: worse than the pre-round-40 §6.3 bug (there interactions at least resurfaced in the unresolved queue; here they resurface nowhere), reproducing defect #1 via pure skew with no deletion involved — round-40's fix, scoped to §6.3 only, left its exact mirror-image unfixed in §6.2, the same "fixed one population, not swept to the sibling read path" class this document has now hit at rounds 19, 24, 27, 30, 36, and 40→41 | **Fixed** (§6.2 STEP1: an additional anti-join query, symmetric to §6.3's third disjunct, finds `(type, target)` pairs this Contact owns with no matching period row and gives each an unconditional unbounded bound in STEP2 — same transient, next-touch-bounded handoff; §10 gains the symmetric fixture) |

## 9. Migration plan

1. New Alembic revision in `bin-dbscheme-manager/bin-manager/main/versions/`:
   `CREATE TABLE contact_address_ownership_periods` (§3.1), generated via
   `alembic revision` (never a hand-picked revision id, per
   `voipbin-dbscheme-migration` skill).
2. Backfill step in the **same migration's `upgrade()`**, guarded by an
   `INSERT ... SELECT` from `contact_addresses` (only rows with `contact_id IS
   NOT NULL`).

   **Round-22 BLOCKER, fixed by changing the backfill rule: `valid_from` must
   backfill as `NULL` (unbounded past), not `tm_create`.** The previous rule
   (`valid_from = tm_create`) would have erased currently-visible history on
   migration day: the most common CRM flow — a call arrives from an unknown
   number first, the Contact is registered afterwards — produces interactions
   whose event time *precedes* the address row's `tm_create`. Today's
   pure-equality matching attributes those interactions to the Contact;
   under a `valid_from = tm_create` period they would fail
   `>= valid_from` and simultaneously vanish from the timeline (defect #1's
   symptom) and resurface in the unresolved queue (defect #3's symptom).
   It would also have been internally inconsistent: the same registration
   made *after* migration goes through §4 Step 5 and gets `valid_from=NULL`,
   so attribution would have depended on which side of the migration the
   registration happened to fall. `valid_from = NULL` is the backfill that
   is genuinely inert **for live addresses under live Contacts (the
   `tm_delete IS NULL` branch below)** — it reproduces today's time-agnostic
   value matching exactly for those rows, changing nothing observable at
   cutover; this design's period boundaries then take effect only for
   ownership changes made after migration. **Round-23 correction: this
   inertness claim is explicitly scoped to that branch.** The round-14
   `tm_delete`-closing branch below is deliberately NOT inert: a closed
   `[NULL, tm_delete)` period stops suppressing post-deletion interactions
   that today's time-agnostic `NOT EXISTS` wrongly hides, so on migration
   day those interactions newly appear in the unresolved queue. That is an
   intended, desirable behavior change (it is precisely the A9-b cleanup
   working), recorded here as such per this document's
   every-gap-gets-a-disposition convention. (§10's backfill-inertness open
   question is resolved for the live-branch by construction; the
   deleted-branch resolves as "intentionally not inert.")

   **Round-23 finding, fixed here: without a cleanup step, this design
   would permanently lock A9-b-corrupted targets out of every API path.**
   The A9-b row (a live `contact_addresses` row under a soft-deleted
   Contact) blocks re-registration by anyone else (`ErrDuplicateTarget`
   from `idx_contact_addresses_identifier`; `ClaimAddress` returns
   `ErrConflict` whenever `ContactID != uuid.Nil` without checking the
   owner's `tm_delete`, `address.go:187-192`), and this design's own new
   A9-b `TMDelete` guard simultaneously blocks the previously-available
   recovery path (`RemoveAddress`/`UpdateAddress` on the deleted Contact →
   now `cerrors.NotFound`) — the design would have removed the only API
   escape hatch while leaving the blocking row in place, a regression
   requiring manual DB surgery. **Fix: the migration's backfill step, after
   inserting the closed `[NULL, tm_delete)` period for each A9-b row,
   hard-deletes that `contact_addresses` row in the same `upgrade()`** —
   the period row preserves the attribution history (that is its entire
   job), and removing the orphaned live row restores the target to a
   cleanly re-registrable state, which is exactly what a `RemoveAddress`
   call would have produced had it been made when the Contact was deleted.

   **Round-23 note (recorded, non-blocking): the backfill's no-overlap and
   open-period-uniqueness safety depends on `idx_contact_addresses_identifier`
   (UNIQUE `(customer_id, type, target)` on a hard-delete table)** — the
   source table can hold at most one row per target, so the backfill cannot
   produce two periods (let alone two open periods) for one target. Named
   explicitly so a future relaxation of that unique index is recognized as
   invalidating this backfill's assumptions.

   **Round-14 finding: the backfill must not leave permanently-open periods
   under already-deleted Contacts (A9-b-corrupted data).** Before this
   design's `TMDelete` guard (§4's A9-b paragraph) existed,
   `AddAddress`/`UpdateAddress`/`RemoveAddress`/`ClaimAddress` never
   checked `Contact.TMDelete`, so a live `contact_addresses` row can already
   exist today under a Contact that was soft-deleted *before* this migration
   runs. Backfilling such a row as `valid_to = NULL` creates a
   permanently-open period nothing will ever close (`ContactDelete`'s
   closure only fires on future calls), silently swallowing interactions in
   `InteractionListUnresolved` (§6.3). **Fix:** the backfill
   `INSERT ... SELECT` joins `contact_addresses` to
   `contact_contacts ON contact_addresses.contact_id = contact_contacts.id`
   and branches on `contact_contacts.tm_delete`: rows where it `IS NULL`
   backfill as `valid_to = NULL`; rows where it `IS NOT NULL` backfill as
   `valid_to = contact_contacts.tm_delete` — an already-closed period,
   consistent with what a `ContactDelete` call made at that Contact's actual
   deletion time would have produced had this design already existed then.
   (Round-15 had found the earlier `valid_from = tm_create` rule could
   produce an inverted `valid_from > valid_to` range for addresses added
   *after* their Contact's deletion, requiring a zero-length-period
   workaround; round-22's `valid_from = NULL` rule makes inversion
   structurally impossible — `NULL` is negative infinity, always before any
   `valid_to` — so that workaround is no longer needed and is retired.)
3. Round-trip verify (MariaDB build → mysqldump → MySQL 8.0 import) per the
   `voipbin-dbscheme-migration` skill, including the generated-column UNIQUE
   behavior probes (open-period coexistence, real-duplicate rejection, distinct-
   key non-collision — the skill's three-probe recipe for reviewing any
   generated-column-UNIQUE fix).
4. No `downgrade()` data-loss concern: `DROP TABLE IF EXISTS
   contact_address_ownership_periods` is fully reversible (no other table's data
   depends on it).
5. **Round-15 finding: this plan never addressed the rolling-deploy version-
   skew window, despite §5 itself establishing that `bin-contact-manager`
   runs as multiple pods consuming a shared RabbitMQ queue.** If the schema
   migration (step 1) lands before every pod is running code that knows
   about `contact_address_ownership_periods`, an old-binary pod can process
   `AddressCreate` during the overlap and write only to `contact_addresses`
   — no ownership period is created for that target. A new-binary pod later
   processing `AddressDelete` for the same address then finds no open period
   to close (a silent no-op, since this closing `UPDATE` has no `RowsAffected`
   check the way B5's fix added one for `AddressUpdate`/`AddressDelete`'s
   `contact_addresses`-side write — see §5.4). If that target is later
   `ClaimAddress`-ed by a different contact, no period exists at all for it,
   so §4's Step 5 (true first-ever registration) misclassifies it and backs
   in `valid_from=NULL`, reproducing defect #2 (history leaking to the new
   owner) via a deploy-timing gap this design otherwise closes everywhere
   else.

   **Round-16 finding: round-15's proposed fix (a "schema-and-binary-together,
   drain-then-replace" deployment ordering constraint) is not enforceable and
   does not match this codebase's actual deploy shape.** Verified against the
   real infrastructure: `bin-dbscheme-manager` and `bin-contact-manager` are
   separate CI/CD workflows with independent path-filter triggers and no
   `requires` dependency between them (`.circleci/config_work.yml`); schema
   migrations are applied by a human running `alembic upgrade` manually
   against a VPN connection, entirely outside CI/CD
   (`bin-dbscheme-manager/docs/operations.md`); and `bin-contact-manager`'s
   `k8s/deployment.yml` has no explicit `strategy`, so it inherits Kubernetes'
   default surge-first `RollingUpdate` (new pods start before old ones
   terminate) — the opposite of the drain-first sequencing round-15's fix
   assumed. A prose deployment-ordering instruction with no CI gate and no
   k8s manifest change to enforce it is not a real constraint; nothing stops
   the schema migration and a surge-first rolling deploy from overlapping
   exactly as round-15 described. **Fix: defend against the skew in code,
   not in deploy-ordering discipline.** Two changes, both cheap and already
   consistent with patterns this design uses elsewhere:
   - **Round-17 finding: the original text here only named `AddressDelete`'s
     closing `UPDATE`, but §4 has three closing-`UPDATE` sites exposed to the
     identical skew race** — `AddressDelete`, `AddressUpdate` (target
     changed)'s old-target close, and `ContactDelete`'s per-target close
     (round-13/14's N-target generalization). Applying the `RowsAffected`
     check to only one of the three would leave the other two silently
     no-op'ing exactly as before, contradicting this fix's own "make the
     blast radius visible" goal for those paths. **All three closing-`UPDATE`
     sites get the same `RowsAffected == 0` handling**, not just
     `AddressDelete`'s.
   - **Round-17 finding: `pkg/dbhandler` has zero Prometheus integration
     today** (verified: no `prometheus` import anywhere in the package) —
     existing metrics live only in `pkg/casehandler`/`pkg/listenhandler`/
     `pkg/subscribehandler`, each with its own `metricsNamespace` var and an
     `init()` registering it via `prometheus.MustRegister`. Claiming this is
     "consistent with an existing pattern" was wrong; `dbhandler` has no
     such pattern to reuse and introducing one there for the first time is
     itself a small architectural decision this design needs to make
     explicit rather than wave at. **Fix:** `dbhandler` gains its own
     `metricsNamespace` + `init()` block, following the exact same shape as
     `casehandler`'s (same `prometheus.MustRegister` pattern, package-local
     — not a dependency on `casehandler`'s or `listenhandler`'s existing
     counters, and not a new upward dependency from `dbhandler` into any
     handler-layer package). All three closing-`UPDATE` sites above call
     this one `dbhandler`-local counter on a `RowsAffected == 0` miss: this
     is not an error (the address itself, or Contact, still updates/deletes
     normally — an old-binary-created address with no period is a
     legitimate, if degraded, state) but it now increments a Prometheus
     counter (surfaced through this service's existing `:2112/metrics`
     endpoint, per its `CLAUDE.md`) so the skew window's actual blast radius
     becomes visible in monitoring rather than being purely silent.
   - Step 5 of §4 (true first-ever registration → `valid_from=NULL`) already
     cannot distinguish "genuinely never owned by anyone" from "owned by an
     old-binary pod that never wrote a period" — those two states are
     indistinguishable from the ownership-period table alone, by
     construction, so no code change can fully close this gap without also
     inspecting `contact_addresses.tm_create` for a plausible pre-migration
     timestamp, which is out of scope here. This residual gap is accepted
     (§7) rather than solved: it degrades gracefully into "this design's
     history-leak protection does not apply to periods created during the
     migration's version-skew window," a bounded blast radius (only the
     rollout window, not indefinitely), which is a materially different
     (and much smaller) claim than round-15's "closed via deployment
     ordering," which round-16 found this codebase cannot actually enforce.
   - **Round-22 correction to the "bounded" claim above, and a second,
     larger accepted limitation it obscured: targets hard-deleted BEFORE the
     migration leave no `contact_addresses` row for the backfill to see, so
     they get no period at all — and a post-migration re-registration of
     such a target hits §4 Step 5 ("no rows at all") and opens
     `valid_from=NULL`, inheriting the previous owner's entire pre-migration
     interaction history.** This is defect #2 itself, unmitigated for the
     entire population of pre-migration deletions — and unlike the skew-window
     gap above it is NOT bounded by the rollout window: it fires whenever any
     pre-migration-deleted target is re-registered, indefinitely. It is also
     unfixable from data this system retains: `contact_addresses` is
     hard-deleted (§1's defect #2's own root cause), so no record exists of
     which targets were ever previously owned. The honest statement of what
     this design delivers, replacing the over-broad "bounded blast radius"
     claim above where it implied full coverage: **defect #2 is prevented
     for ownership changes made after this migration; reassignment
     misattribution rooted in deletions that predate the migration is
     detectable only via `contact_resolutions` manual correction (§2's
     existing mechanism) and is accepted as unrecoverable-by-design.** The
     skew-window paragraph's "bounded" claim stands only for its own
     specific gap (periods missed during rollout), not as a statement about
     this design's total residual exposure.
   - **Round-22 note (recorded, non-blocking): the `COALESCE(tm_interaction,
     tm_create)` fallback (§6.2/§6.3, round-21) trades defect #1 for a
     narrow, accepted misattribution window.** For interactions whose origin
     event carried no timestamp, `tm_create` is the projection time, not the
     event time; RabbitMQ at-least-once delivery with retries/backlog means
     projection can lag the event by an unbounded interval, and if an
     ownership boundary (delete, claim) falls inside that lag, the
     interaction is attributed to the owner at projection time rather than
     at event time — a bounded-population (NULL-`tm_interaction` rows only),
     unbounded-window variant of defect #2. Accepted rather than solved:
     the alternative (leaving those rows unmatchable) is a certain
     regression of defect #1 for the same population, strictly worse than a
     conditional misattribution; `contact_resolutions` remains the manual
     correction path, consistent with §7's other accepted gaps.
   - **Round-23 finding: the skew analysis above covered only one direction
     (old-binary write creates no period); the reverse direction — a
     new-binary pod opens a period, then an old-binary pod deletes the
     address — leaves an orphaned OPEN period, which is qualitatively worse
     than a missing one.** Sequence: new-binary handles B's
     `AddressCreate`/`ClaimAddress` on target T (open period written);
     old-binary handles B's `AddressDelete` (or target-changing
     `AddressUpdate`) — the `contact_addresses` row is hard-deleted but the
     period stays open forever (the old binary doesn't know the table
     exists). Consequences: (a) **indefinite Step-1 lockout** — any later
     registration of T finds B's orphaned open row and returns
     `ErrConflict`/409 even though nobody owns T in `contact_addresses`,
     persisting long after the rollout window closes, with no self-healing
     path short of B's own `ContactDelete` or manual surgery; (b) B's
     timeline keeps absorbing T's future interactions (defect #2,
     unbounded); (c) it also falsifies §4 Step 2's "unreachable for
     `AddressCreate`" claim in this corrupted state — with the address row
     gone, the unique index no longer blocks B re-registering T, and
     `AddressCreate` really can arrive at its own open period.
     **Disposition — detected and self-healing, not prevented:** prevention
     is impossible (the old binary cannot be patched retroactively), so the
     new binary gets one additional rule making Step 1 self-healing against
     exactly this state: **when Step 1 finds a blocking open period, it
     fetches the `contact_addresses` row for the same
     `(customer_id, type, target)` (same locked transaction,
     `SELECT ... FOR UPDATE` on the unique identifier) and requires
     ownership agreement — the row must exist AND its `contact_id` must
     equal the blocking period's `contact_id`. If the row is gone, is
     unresolved (`contact_id` NULL), or belongs to any other contact, the
     open period is an orphan — close it (`valid_to = NOW()`, incrementing
     a second dbhandler-local Prometheus counter for visibility) and
     continue the step procedure as if it were closed.**

     **Round-24 correction (why ownership agreement, not mere existence):
     the round-23 text originally specified "row still exists" — but an
     orphaned slot can be re-occupied by `CreateUnresolvedAddress`
     (`contact_id = NULL`), which per round-10's exemption skips the step
     procedure entirely and therefore never triggers this self-healing
     check itself. Under existence-only checking, a later `ClaimAddress` on
     that unresolved row would find the orphan period, see "a row exists,"
     misjudge the orphan as a genuine live owner, and return 409 — silently
     reinstating the exact permanent lockout this rule was created to
     eliminate (the unresolved row also blocks re-registration via the
     unique index, so no API path remains). Ownership agreement closes this
     hole: a NULL-owned or differently-owned row proves the period's owner
     no longer holds the address, which is precisely the orphan condition.**
     This converts the permanent 409 lockout into a one-time,
     self-repairing detour on the next registration attempt; the closed
     period retains B's attribution up to the repair point (the best
     available approximation of the unrecorded deletion time, same standard
     as the `tm_create` fallback's disposition above); and Step 2's
     unreachability claim is restored after repair because the orphan that
     made `AddressCreate` reach it no longer survives the check. During the
     skew window itself B's timeline over-absorbs until repair — accepted,
     same bounded-window standard as the missing-period direction. §4 Step
     1's text gains a cross-reference to this rule.

### 9.x Reconciliation appendix (round-33, rewritten in round-34)

**Round-34 correction — the round-33 query was doubly wrong and its
"surfaces exactly these residues" claim is withdrawn.** The original
predicate (tombstoned owner + closed period + zero address rows + no
successor period) is indistinguishable in-band from a NORMAL
`ContactDelete` (whose round-25/26 cleanup produces the identical data
shape), so it returned essentially every deleted-and-not-reregistered
Contact (false-positive flood); and its `succ.id IS NULL` requirement
made the report DROP a ghost residue at exactly the moment its damage
materializes (the retry's Step-4 period IS a successor — false negative
on the harmful state). There is NO complete in-band discriminator for
ghosts: the defining fact ("`ContactCreated` was never published") lives
outside the DB, so **the compensating-cleanup failure log line remains
the canonical ghost identifier (§4's round-33 correction), and this
query is demoted to a heuristic candidate list** for the case where logs
have rotated away:

```sql
-- HEURISTIC ghost-residue candidates (not exact — see note above):
-- closed periods owned by tombstoned Contacts with no address rows,
-- where the Contact lived only briefly (ghosts die within one failed
-- create request; tune the interval to the deployment's RPC timeout).
SELECT p.customer_id, p.type, p.target, p.contact_id,
       p.valid_from, p.valid_to, c.tm_create, c.tm_delete
FROM contact_address_ownership_periods p
JOIN contact_contacts c
  ON c.id = p.contact_id
 AND c.tm_delete IS NOT NULL
 AND TIMESTAMPDIFF(MINUTE, c.tm_create, c.tm_delete) < 60
LEFT JOIN contact_addresses a
  ON a.contact_id = p.contact_id
WHERE p.valid_to IS NOT NULL
  AND a.id IS NULL;
```

Notes: (i) the short-lifetime filter is a heuristic — a legitimately
created-then-quickly-deleted Contact also matches; rows returned are
candidates, not verdicts. **(round-35 scope note: for RECOVERED ghosts
`tm_delete` is the operator's recovery time, not the failed request's
duration — a recovery performed later than the interval drops that
residue from the report, so widen the interval when reviewing after
delayed recoveries; and UN-recovered ghosts (no tombstone, open period,
live row) match neither predicate — this query covers recovered ghosts
only, which is acceptable because un-recovered ghosts are still
actionable through their 409 symptom and the log line, the sole
canonical identifier for both states.)** (ii) There is deliberately no
successor-period condition: the harmful state (retry already re-routed
through Step 4) must stay IN the report, and a `succ`-style join would
also need a `succ.id != p.id` guard to avoid round-15 zero-length
periods matching themselves. (iii) The query is read-only, replica-safe,
and intentionally NOT wired into any automated mutation — it surfaces,
humans decide; the correction path is `contact_resolutions` (§7).

## 10. Open questions for future rounds' review

- (Round-39, recorded not blocking) `AddressResetPrimary` and
  `AddressCreate`/`AddressUpdate` are currently separate `DBHandler`
  interface calls without a shared transaction parameter
  (`contact.go:374,380`), yet §5.1 requires them under one `BeginTx` per
  outer operation. The document has not specified the mechanical
  approach (a transaction-carrying context, a combined method, or
  something else) — needed before implementation starts, tracked here
  rather than blocking this round's approval since it is a plumbing
  detail with no attribution-correctness implication (unlike this
  round's `AddressUpdate`/`AddressDelete` compare-and-retry fix, which
  changes committed data).
- Does the fixed lock order in §5.2 actually eliminate the deadlock class round-2
  found, or only narrow it? (Needs the same live-concurrency spike verification
  the `voipbin-dbscheme-migration` skill's "owner preferences" section mandates
  for any new get-or-create-shaped concurrency pattern — reasoning by analogy to
  `casehandler` is not sufficient proof per pchero's standing rule.)
- ~~Is `valid_from = tm_create` backfill actually inert for every existing
  Contact...~~ **Resolved by round-22**: the backfill rule changed to
  `valid_from = NULL`, which is inert by construction (reproduces today's
  time-agnostic value matching exactly); the dataset-audit question is moot.
- Full transaction-boundary diff for `AddAddress`/`UpdateAddress`/`RemoveAddress`/
  `ClaimAddress`/`CreateUnresolvedAddress`/`ContactCreate` against their current
  implementations, to confirm no existing test fixture assumption (e.g. mock
  call ordering) breaks. **Round-13 finding: `ContactCreate` was missing from
  this list despite round-11/12 changing it** — confirmed at least one
  existing test, `Test_Create_WithAddressTagErrors`
  (`contact_test.go:1693-1751`), explicitly asserts today's error-swallowing
  behavior (two `AddressCreate` failures, `Create()` still returns `nil`,
  `TagAssignmentCreate`/`ContactGet`/`PublishEvent` all still called) and will
  need to be rewritten, not just re-verified, to assert the new
  error-propagation-plus-compensating-cleanup behavior instead.
  **Round-26 addition: `ContactDelete` joins the same list** — `Test_Delete`
  (`contact_test.go:260-`) asserts today's delete-event payload shape, and
  the round-26 pre-delete snapshot rule (§4) exists precisely to keep that
  payload's `addresses` field intact; the test must be re-verified against
  the snapshot-based implementation, and a new case must cover "tombstoned
  Contact's address rows are hard-deleted, including period-less ones."
  **Round-36 additions:** (a) `AddressDeleteCompensating` is a new
  `DBHandler` interface method — mockgen regeneration required, plus a
  test asserting it DELETEs (not closes) the loop's own period and
  publishes nothing; (b) `InteractionListUnresolved` needs an
  unresolved-row fixture on both sides of the cutover asserting the
  suppression effect is preserved (§6.3's second `NOT EXISTS`).
  **Round-39 additions:** (a) `AddressUpdate`/`AddressDelete` gain the
  same by-addressID compare-and-retry `AddressClaim` already has —
  mismatch-retry and NotFound-abort test cases needed for both, mirroring
  `AddressClaim`'s existing rounds 8/9 fixtures; (b) claim-of-unresolved
  fixtures per §6.3's round-39 update (immediately-prior gap attaches;
  older non-adjacent gap resurfaces, documented).
  **Round-40 addition:** `InteractionListUnresolved` needs a
  missing-period-skew fixture (an owned `contact_addresses` row with
  zero period rows, per §9's accepted degraded state) asserting its
  interactions stay suppressed from the queue until the next touch gives
  the row a period.
  **Round-41 addition:** `interactionListByContact` needs the symmetric
  missing-period-skew fixture — asserting an owned, period-less row's
  interactions still appear on the OWNING Contact's timeline (not just
  correctly absent from the unresolved queue).
- (Recorded, not blocking implementation — §6.2/§8) If a customer's ownership-period
  count for a single Contact ever grows large enough to make the OR-expanded
  STEP2 query in §6.2 a real performance concern, the fix is switching
  `InteractionList`'s calling convention from an OR-expanded parameter slice to
  a real `JOIN` against `contact_address_ownership_periods` — deferred, not
  designed here, until real evidence of the problem exists.
