# Contact-Address Ownership Integrity — Design Doc

- Status: DRAFT (round 3 review pending)
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
with partially-committed state. If the compensating `RemoveAddress` calls
themselves fail (a further concurrency edge case), that failure is logged
but does not mask the original error returned to the caller — the caller
still sees why the address it cared about failed, and an orphaned partial
address set becomes an operational cleanup concern (visible via existing
`GET /v1/contacts/{id}` inspection) rather than a silent data-integrity bug.

**Round-13 finding: reusing `contacthandler.RemoveAddress` verbatim for this
compensating cleanup silently leaks a wrong event.** `RemoveAddress`
unconditionally publishes `EventTypeContactUpdated` on success
(`contact.go:393-417`). But when `ContactCreate` fails partway through its
address loop, `ContactCreated` was never published for this Contact (the
handler returns an error before ever reaching that point) — so a downstream
consumer would receive an `updated` event for a Contact it never saw
`created` for, a phantom-update with no corresponding prior state. **Fix:**
the compensating cleanup calls a **non-event-publishing** internal path —
specifically, the same `dbhandler.AddressDelete` (plus the ownership-period
Step-based closure) `contacthandler.RemoveAddress` itself calls internally,
but invoked directly rather than through the `contacthandler.RemoveAddress`
entry point, so no `EventTypeContactUpdated` is published. This is not a new
code path: it factors out exactly the portion of `RemoveAddress`'s existing
logic that both call sites need (the dbhandler-level delete + ownership-
period closure) from the portion only the public `RemoveAddress` RPC needs
(cache refresh + event publish).

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
propagates 409 (live owner), which is the correct outcome. Row repair by
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
contact, `valid_from=NOW()` (not NULL — this contact never owned the target
before now), `valid_to=NULL`.

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
is precisely where that state surfaces. All repairs increment the same
skew-orphan Prometheus counter.

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
at the same bound instead of retrying indefinitely.

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

-- becomes:
NOT EXISTS (
    SELECT 1 FROM contact_address_ownership_periods p
    WHERE p.customer_id = i.customer_id
      AND p.type = i.peer_type
      AND p.target = i.peer_target
      AND COALESCE(i.tm_interaction, i.tm_create) >= COALESCE(p.valid_from, i.tm_interaction, i.tm_create)
      AND COALESCE(i.tm_interaction, i.tm_create) <  COALESCE(p.valid_to, i.tm_interaction + INTERVAL 1 SECOND, i.tm_create + INTERVAL 1 SECOND)
)
```

This is a single correlated subquery keyed entirely on the outer row's own
`contact_interactions` columns (`i.customer_id`, `i.peer_type`, `i.peer_target`,
`i.tm_interaction`) — no Go-side loop, no bind parameters beyond the ones the
surrounding query already has. `idx_ownership_periods_lookup` (§6.4) covers
this subquery's `WHERE` clause the same way `idx_contact_addresses_lookup`
already covers the original.

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
| Round-29 finding: the round-28 covering predicate ("skip if a covering closed period exists") was undefined — the implementer-discretion ambiguity class — and the Step-1 clause vs repair (a) specified conflicting period-write orders that could leave two overlapping closed periods for one era | **Fixed** (§4 round-29: covering predicate defined — a closed period for the same tombstoned contact_id with `valid_to >= tm_delete`, evaluable from the §4 locked fetch; canonical late-cleanup order — close orphan open period first (which then satisfies the predicate), fabricate only for the period-less variant, row repair last — one era can never yield two overlapping closed periods) |
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

## 10. Open questions for round-4+ review

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
- (Recorded, not blocking implementation — §6.2/§8) If a customer's ownership-period
  count for a single Contact ever grows large enough to make the OR-expanded
  STEP2 query in §6.2 a real performance concern, the fix is switching
  `InteractionList`'s calling convention from an OR-expanded parameter slice to
  a real `JOIN` against `contact_address_ownership_periods` — deferred, not
  designed here, until real evidence of the problem exists.
