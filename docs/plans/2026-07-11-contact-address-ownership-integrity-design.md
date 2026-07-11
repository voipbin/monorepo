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
semantics; nothing reads or writes it differently as a side effect of this design.

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
caller. This makes a failed `POST /v1/contacts` call converge back to "no
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
`(customer_id, type, target)`.) If yes: this is a live-owner conflict.
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
contact's own.) If yes: `AddressClaim`'s idempotency retry — round-6's case
4. dbhandler still commits the transaction (nothing to roll back; this is a
successful no-op, not an error), and `contacthandler.ClaimAddress`'s
existing post-commit behavior (re-fetch via `AddressGet`, publish
`EventTypeContactUpdated`) is unchanged either way — this design changes
only what happens inside the transaction, not that response path. This case
remains genuinely unreachable for `AddressCreate` (still blocked upstream by
`contact_addresses`' unique index).

**Step 3 — does this contact_id have any closed row(s) of its own?** (Only
reached once steps 1–2 confirm no open row exists anywhere for this target —
so any row found here is guaranteed closed, removing the ambiguity that let
the old case 3 fire on a target that actually had an open row.) If yes, take
the one with the latest `valid_to` (`ORDER BY valid_to DESC` over the
already-fetched set, no extra query). Then check the same fetched set for
any *other* contact_id's row with `valid_from` strictly later than that
closed row's `valid_to` — and because step 1 already established there is no
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
| Contact soft-delete (`ContactDelete`) | Close **every** open period owned by that contact_id (prevents an orphaned open period after the owning Contact itself is gone) — see round-13 addition below for the transaction/lock discipline this requires |

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
(peer_type = ? AND peer_target = ? AND tm_interaction >= COALESCE(?, tm_interaction)
                                    AND tm_interaction <  COALESCE(?, tm_interaction + INTERVAL 1 SECOND))
```

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
validFromSQL, validToSQL := "tm_interaction", "tm_interaction + INTERVAL 1 SECOND"
if b.ValidFrom != nil {
	validFromSQL = "?"
}
if b.ValidTo != nil {
	validToSQL = "?"
}
clause := fmt.Sprintf("(peer_type = ? AND peer_target = ? AND tm_interaction >= %s AND tm_interaction < %s)", validFromSQL, validToSQL)
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
`[]AddressPair`, not touched by this change). The existing
`mock_main.go:708`/`main.go:58` signatures are additive (a new optional
parameter, defaulted to the existing behavior when omitted), so no existing
caller or existing test needs to change — only STEP2's specific call site
in `interactionListByContact` is rewritten to use the new path. STEP0, 3–6 of
`interactionListByContact` are unchanged.

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
      AND i.tm_interaction >= COALESCE(p.valid_from, i.tm_interaction)
      AND i.tm_interaction <  COALESCE(p.valid_to, i.tm_interaction + INTERVAL 1 SECOND)
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
is added specifically so the OR-expanded, time-bounded STEP2 query and the
`NOT EXISTS` in §6.3 are index-covered rather than falling back to a per-row
filter scan. This mirrors `idx_contact_interactions_peer`'s existing shape
(`customer_id, peer_type, peer_target`) with the two range columns appended.

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
| Round-1 BLOCKER: `[NULL,NULL]` backfill for all existing addresses is wrong (re-introduces mis-attribution for addresses with an unknown-to-us prior reassignment history) | **Backfill changed**: for every currently-live `contact_addresses` row, backfill an open period with `valid_from = that row's own tm_create` (not NULL), `valid_to = NULL`. `valid_from=NULL` (truly unbounded) is reserved for periods created *after* this migration ships, where "opened at CREATE time" already is the correct answer. The one-time backfill explicitly cannot know whether a live row's number was silently reassigned before this feature existed — using its own `tm_create` as the lower bound is the honest answer ("we don't know what happened before this address was (re-)registered"), not a claim of unbounded history. This is documented as a **known backfill limitation**, not silently glossed over. |
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
| Round-13 finding: `ContactDelete`'s "close every open period" effect had no transaction/lock-ordering/deadlock-retry discipline specified, unlike every other row in §4's table — it locks an arbitrary number of targets in one transaction, the same shape as `AddressUpdate`'s two-target case but with N instead of 2, exposed to the same deadlock class | **Fixed** (§4: generalizes §5.2's existing ascending `(type, target)` lock-ordering rule from N=2 to arbitrary N, same `maxDeadlockRetries` retry loop — not a new mechanism) |
| Round-13 finding: reusing `contacthandler.RemoveAddress` verbatim for round-12's compensating cleanup would leak a spurious `EventTypeContactUpdated` for a Contact whose `ContactCreated` was never published (the create failed before reaching that point) | **Fixed** (§4: compensating cleanup calls the non-event-publishing dbhandler-level delete + ownership-period closure directly, not the public `RemoveAddress` entry point) |
| Round-13 finding: §10's transaction-boundary-diff checklist named five handlers but omitted `ContactCreate`, despite round-11/12 changing it — an existing test, `Test_Create_WithAddressTagErrors`, explicitly asserts today's error-swallowing behavior and will need rewriting, not just re-verification | **Fixed** (§10: `ContactCreate` added to the checklist, with the specific test named) |
| Round-13 finding, investigated and rejected: a reviewer flagged §9's backfill filter (`contact_id IS NOT NULL`) as potentially wrong on the theory that unresolved addresses are marked with the `uuid.Nil` sentinel rather than SQL `NULL`, which would make the filter a no-op | **Not a defect** — verified against `dbhandler.AddressCreate`'s actual code and its own comment (`address.go:112-117`): `uuid.Nil.Bytes()` is 16 zero bytes, "NOT SQL NULL," so the write path explicitly passes Go `nil` (→ SQL `NULL`) whenever `ContactID == uuid.Nil`. `contact_id IS NOT NULL` correctly excludes unresolved rows at the database level; the reviewer's theory conflated the Go-level zero-value sentinel with the DB-level column semantics, which the codebase itself takes explicit care to keep distinct. |
| Round-14 BLOCKER: `ContactDelete`'s ownership-period closure had no transaction/lock-ordering/deadlock-retry discipline specified, and its own membership-snapshot read only locks the targets it finds — a concurrent `ClaimAddress` committing a new open period after that read but before commit is invisible to it, leaving an orphan open period under a Contact that no longer exists | **Fixed** (§4: generalized §5.2's ascending lock-ordering rule to arbitrary N targets, reused §5.3's retry loop; added a post-lock membership re-check that repeats until a pass finds no new targets) |
| Round-14 BLOCKER: §9's backfill rule didn't account for pre-existing data corrupted by the A9-b bug — a live `contact_addresses` row under an already-soft-deleted Contact would backfill as a permanently-open period with no future `ContactDelete` left to close it | **Fixed** (§9: backfill joins `contact_contacts` and closes the period at the Contact's `tm_delete` instead of leaving it open, for rows under an already-deleted Contact) |
| Round-15 finding: round-14's `tm_delete`-closing backfill branch didn't guard against `contact_addresses.tm_create > contact_contacts.tm_delete` (the true A9-b-corrupted case — an address added *after* its Contact was already deleted), which would silently insert an inverted `valid_from > valid_to` period that no interaction-matching query could ever match, permanently and silently losing that address's entire history | **Fixed** (§9: the `tm_delete`-closing branch now requires `tm_create <= tm_delete`; rows that fail this check backfill as a zero-length already-closed period `valid_from = valid_to = tm_delete` instead of an inverted range) |
| Round-15 BLOCKER: this plan never addressed the rolling-deploy version-skew window — an old-binary pod processing `AddressCreate` during a migrate-then-roll-pods deploy would create no ownership period, letting later `AddressDelete`/`ClaimAddress` calls silently no-op / misclassify as first-ever registration, reproducing defect #2 via a deploy-timing gap | **Fixed** (§9: added an explicit deployment-ordering constraint — this migration must ship as schema-and-binary-together, not the ordinary "migrate then roll pods old→new whenever" pattern, since the old binary is incompatible with the new schema being present) |
| Round-15 note (recorded, not blocking): round-14's `ContactDelete` membership re-check loop has no explicit pass-count cap, so a sustained stream of concurrent `ClaimAddress` calls against new targets could keep it re-checking across many passes | Acknowledged as a liveness/latency concern, not a correctness one (each pass still converges toward a fixed point); not given an arbitrary cap because that would trade a rare long-transaction case for a new, precedent-less correctness question (what disposition applies if the cap is hit without converging). |
| Round-16 finding: §6.2's claim that STEP2's time-range condition "mirrors the existing OR-expansion shape... no new query pattern" doesn't hold up — `AddressPair`/`sq.Eq` only carry pure equality, no field for `valid_from`/`valid_to`, and `sq.Eq` cannot express a range condition; STEP2 cannot be implemented by passing more `AddressPair` values into the existing `InteractionList` call | **Fixed** (§6.2: concrete `OwnershipPeriodBound` type and OR-builder added, additive alongside `AddressPair` — only STEP2's call site changes, `interaction_read.go:285`'s single-address re-fetch and all existing tests/mocks are unaffected) |
| Round-16 BLOCKER: round-15's "schema-and-binary-together, drain-then-replace" deployment-ordering fix for the rolling-deploy version-skew window is not enforceable — `bin-dbscheme-manager`/`bin-contact-manager` are independent CI/CD workflows with no `requires` dependency, schema migrations are applied manually outside CI/CD, and the k8s Deployment has no explicit `strategy` (inherits Kubernetes' default surge-first `RollingUpdate`, the opposite of what the fix assumed) | **Fixed** (§9: replaced the unenforceable deploy-ordering instruction with a code-level defense — the ownership-period-closing `UPDATE` gets the same `RowsAffected` check B5 established, now incrementing a Prometheus counter on a miss instead of silently no-op'ing; the residual gap (Step 5 can't distinguish "never owned" from "owned pre-migration with no period") is accepted as a bounded, monitored blast radius limited to the rollout window, not solved) |
| Round-17 finding: §6.2's `sq.GtOrEq{"col": sq.Expr(...)}` construction is not valid squirrel usage — verified against the vendored source that `Eq`/`Lt`/`GtOrEq` only special-case `driver.Valuer` values in their `toSql()`, with no branch for a `Sqlizer` in value position; this compiles but fails at query-exec time with an "unsupported type" error | **Fixed** (§6.2: rewritten to build the whole comparison as one `sq.Expr(...)` raw-SQL fragment with the ±∞ fallback embedded in the SQL string itself, the same pattern §6.3's `NOT EXISTS` subquery already uses, rather than assembling it from squirrel value-position helpers) |
| Round-17 finding: §9's Prometheus-counter fix for the version-skew defense (a) only named `AddressDelete`'s closing `UPDATE`, leaving the identical race in `AddressUpdate`'s old-target close and `ContactDelete`'s per-target close silently un-instrumented, and (b) claimed to be "consistent with an existing pattern" while `pkg/dbhandler` has zero Prometheus integration today (verified: no `prometheus` import in the package; existing metrics live only in `casehandler`/`listenhandler`/`subscribehandler`) | **Fixed** (§9: extended the `RowsAffected == 0` handling to all three closing-`UPDATE` sites, not just `AddressDelete`'s; specified `dbhandler` gaining its own `metricsNamespace` + `init()` block mirroring `casehandler`'s existing shape, package-local with no new upward dependency into the handler layer) |

## 9. Migration plan

1. New Alembic revision in `bin-dbscheme-manager/bin-manager/main/versions/`:
   `CREATE TABLE contact_address_ownership_periods` (§3.1), generated via
   `alembic revision` (never a hand-picked revision id, per
   `voipbin-dbscheme-migration` skill).
2. Backfill step in the **same migration's `upgrade()`**, guarded by an
   `INSERT ... SELECT` from `contact_addresses` (only rows with `contact_id IS
   NOT NULL`), per §8's backfill rule (`valid_from = tm_create`, `valid_to =
   NULL`).

   **Round-14 finding: this backfill rule, as originally written, does not
   account for pre-existing data corrupted by the A9-b bug this design also
   fixes.** Before this design's `TMDelete` guard (§4's A9-b paragraph)
   existed, `AddAddress`/`UpdateAddress`/`RemoveAddress`/`ClaimAddress` never
   checked `Contact.TMDelete`, so a live `contact_addresses` row can already
   exist today under a Contact that was soft-deleted *before* this migration
   runs — a state the runtime guard prevents going forward but cannot undo
   retroactively for data already written. Backfilling such a row as
   `valid_to = NULL` (the plain rule above) creates a permanently-open
   period: `ContactDelete`'s "close every open period" trigger (§4, including
   round-14's re-check fix above) only fires on *future* `ContactDelete`
   calls, and this Contact's deletion already happened in the past, so
   nothing will ever close it. That phantom open period would then silently
   swallow interactions in `InteractionListUnresolved` (§6.3), whose
   `NOT EXISTS` correlated subquery checks period existence only, never
   `contact_contacts.tm_delete` — exactly the kind of silent misattribution
   §1 exists to fix, reintroduced via this design's own migration. **Fix:**
   the backfill `INSERT ... SELECT` joins `contact_addresses` to
   `contact_contacts ON contact_addresses.contact_id = contact_contacts.id`
   and branches on `contact_contacts.tm_delete`: rows where it `IS NULL`
   backfill as `valid_to = NULL` (the plain rule, unchanged); rows where it
   `IS NOT NULL` backfill as `valid_to = contact_contacts.tm_delete` instead
   — an already-closed period, consistent with what a `ContactDelete` call
   made at that Contact's actual deletion time would have produced had this
   design already existed then.

   **Round-15 finding: this `tm_delete`-closing branch, as written, does not
   guard against `valid_from > valid_to`.** The A9-b-corrupted data this
   branch exists to handle is, by definition, a `contact_addresses` row
   whose owning Contact was already soft-deleted *before* the write-guard
   existed — meaning `contact_addresses.tm_create` (this backfill's
   `valid_from`) can be *later* than `contact_contacts.tm_delete` (this
   branch's `valid_to`): the address was added to an already-deleted Contact,
   exactly the scenario A9-b made possible. `contact_address_ownership_periods`
   has no `CHECK (valid_from < valid_to)` constraint (§3.1), and
   `open_period_uk` only activates when `valid_to IS NULL`, so an inverted
   row like this inserts silently with no error. Every interaction-matching
   path that uses this row's range (§6.2's `>= valid_from AND < valid_to`,
   §6.3's `NOT EXISTS` correlated subquery) can never match anything against
   an inverted range — the address's entire interaction history becomes
   permanently unmatchable, and because §6.3 treats "no matching period" the
   same as "period doesn't exist," those interactions don't even resurface in
   the unresolved queue; they silently vanish. This is defect #1 from §1
   (delete → history disappears) reintroduced by this design's own backfill
   for exactly the corrupted-data case it was trying to fix. **Fix:** the
   backfill's `tm_delete`-closing branch additionally requires
   `contact_addresses.tm_create <= contact_contacts.tm_delete`; rows that
   fail this check (the true A9-b-corrupted case — added after the Contact's
   own deletion) backfill as a **zero-length, already-closed period**
   instead (`valid_from = valid_to = contact_contacts.tm_delete`) rather than
   an inverted one — this correctly represents "no interaction could ever
   have occurred while this contact_id validly owned this target" (the
   Contact was already gone), without the inverted range's silent-vanishing
   failure mode, and without violating any constraint.
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

## 10. Open questions for round-4+ review

- Does the fixed lock order in §5.2 actually eliminate the deadlock class round-2
  found, or only narrow it? (Needs the same live-concurrency spike verification
  the `voipbin-dbscheme-migration` skill's "owner preferences" section mandates
  for any new get-or-create-shaped concurrency pattern — reasoning by analogy to
  `casehandler` is not sufficient proof per pchero's standing rule.)
- Is `valid_from = tm_create` backfill actually inert for every existing
  Contact, or does any existing Contact already have observable reassignment
  history that this backfill would silently misrepresent as "always owned since
  registration"? (Requires a read-only diagnostic query against a real dataset,
  not reasoning from the empty-table round-trip.)
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
- (Recorded, not blocking implementation — §6.2/§8) If a customer's ownership-period
  count for a single Contact ever grows large enough to make the OR-expanded
  STEP2 query in §6.2 a real performance concern, the fix is switching
  `InteractionList`'s calling convention from an OR-expanded parameter slice to
  a real `JOIN` against `contact_address_ownership_periods` — deferred, not
  designed here, until real evidence of the problem exists.
