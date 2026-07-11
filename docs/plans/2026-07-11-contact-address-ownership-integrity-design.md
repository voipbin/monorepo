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
- **No new API.** All four existing endpoints
  (`POST /v1/contacts/{id}/addresses`, `POST /v1/contact_addresses`,
  `PUT /v1/contact_addresses/{id}`, `DELETE /v1/contact_addresses/{id}`,
  `POST /v1/contact_addresses/{id}/claim`) keep their exact current signatures,
  request/response shapes, and status codes. This design is a pure internal
  rewire behind those endpoints. (Confirmed with pchero: no `transfer` endpoint.)
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

All four ownership-period-affecting operations are additions alongside the
existing `contact_addresses` write, inside the **same transaction** the existing
write already runs in (see §5 for the transaction restructuring this requires).

**Round-5 review found the "three separate lookups" framing (round-3/round-4's
fix) was still broken in two ways it could not both fix simultaneously — see
§8 for the specifics. This section replaces that framing entirely with a
single mechanism that resolves both.**

**The decision is made from ONE locking read, not several separate lookups.**
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
accumulated), not just the single `open_period_uk` row the earlier framing
locked. Because this is a real locking read under `FOR UPDATE` — not the
plain, un-locked `SELECT`s the round-3/4 "lookup (a)/(c)" framing used — it
reads the true latest committed state under InnoDB's locking-read semantics,
not a stale `REPEATABLE READ` snapshot: this is what closes round-5's
concurrency finding (§8). The write path then decides **in application code,
from this one already-fetched, already-locked row set** — no further queries
needed:

1. **No rows at all** (for *any* contact_id, open or closed) → true first-ever
   registration. INSERT: `valid_from=NULL`, `valid_to=NULL`.
2. **A row for a *different* contact_id exists (open or closed) and this
   contact_id has no row of its own** → reassignment (A→B, or first-time
   `ClaimAddress` of a target someone else previously held). This is the case
   round-5 review found the three-lookup framing misclassified as case 1
   above (§8) — explicitly checking "does *any* contact_id have a row here,
   not just this one" is what fixes it. INSERT: `contact_id`=this contact,
   `valid_from=NOW()` (not NULL — this contact never owned the target before
   now), `valid_to=NULL`.
3. **This contact_id has its own closed row(s)** — take the one with the
   latest `valid_to` (`ORDER BY valid_to DESC` over the in-memory set already
   fetched, no extra query). Then check the already-fetched row set for any
   *other* contact_id's row with `valid_from` strictly later than that
   closed row's `valid_to`:
   - **None found** → this contact held the target continuously up to close,
     nobody intervened. Re-open: `UPDATE ... SET valid_to=NULL WHERE id=<that
     row>` (no new row — one continuous history).
   - **Found** → someone else provably held the target in the interim (the
     A→B→A case round-4 found broken). Do **not** reopen the stale row.
     INSERT a new row instead: `contact_id`=this contact, `valid_from=NOW()`,
     `valid_to=NULL`.
4. **An open row for this exact contact_id already exists.** Round-6 review
   found this is NOT unreachable via `ClaimAddress` as originally claimed: its
   idempotency pre-check
   (`existing, err := h.AddressGet(...); if existing.ContactID == contactID {
   return nil }`, `pkg/dbhandler/address.go:182-190`) runs as a plain,
   non-locking read **before** the `FOR UPDATE` acquisition this design adds
   — so two concurrent, duplicate `ClaimAddress` calls for the same contact
   (an ordinary at-least-once-delivery retry) can both pass that pre-check
   (both see `ContactID == uuid.Nil` before either commits), then serialize
   on the `FOR UPDATE` lock, and the second one now genuinely observes an
   open row for its own contact_id post-lock. **Fix (moves the pre-check
   inside the lock, not a new branch in this table):** `AddressClaim`'s
   idempotency check is relocated to run *after* the §4 locking read, inside
   the same transaction — if the already-locked row set shows an open period
   for this exact contact_id, treat it as the pre-check always intended
   (return success, no-op — the claim already holds), rather than falling
   through into cases 1–3. With the check moved inside the lock, this case is
   genuinely unreachable for `AddressCreate` (still blocked by
   `contact_addresses`' unique index, unchanged) and is a well-defined no-op
   for `ClaimAddress`, not an unhandled branch.

| Existing operation | Ownership-period effect |
|---|---|
| `AddressCreate` / `ClaimAddress` | One of cases 1–3 above, decided from the single locked read |
| `AddressDelete` | Close the open period for this contact_id+type+target: `UPDATE ... SET valid_to=NOW() WHERE open_period_uk = <hash>`. The period row is never deleted. |
| `AddressUpdate` (target field changed) | Two locking reads required (§5.2 specifies the required order between them) — one for the old target, one for the new: close the old-target period (`valid_to=NOW()`), then apply cases 1–3 above to the new target |
| `AddressUpdate` (target field NOT changed — e.g. only `name`/`detail`/`is_primary`) | **No ownership-period effect.** The write transaction (§5.1) still performs the locking read above for this target as part of its fixed lock-ordering rule, but performs no INSERT/UPDATE against `contact_address_ownership_periods` — the lock exists purely to keep this operation inside the same serialization order as every other address write on that target, not because there is a period decision to make. |
| Contact soft-delete (`ContactDelete`) | Close **every** open period owned by that contact_id (prevents an orphaned open period after the owning Contact itself is gone) |

**A9-b guard placement (round-3 finding: this was only in §8's summary table,
never specified in this section — fixed here).** All four write handlers this
table governs — `AddAddress`, `UpdateAddress`, `RemoveAddress`, `ClaimAddress`
(`pkg/contacthandler/contact.go`) — gain an explicit `c.TMDelete != nil` check
immediately after their existing `ContactGet` call and before any
`contact_addresses`/ownership-period write, mirroring the check
`interactionListByContact` already performs
(`pkg/contacthandler/interaction_read.go:117-125`). On a soft-deleted Contact,
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
between those two calls is accepted, not solved (§7) — but case 2 above
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
the target, not just one) before deciding which of §4's cases applies,
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
across every period the Contact has ever held (mirrors the existing OR-expansion
shape in `dbhandler/interaction.go:131-139`, no new query pattern). STEP0, 3–6 of
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

## 9. Migration plan

1. New Alembic revision in `bin-dbscheme-manager/bin-manager/main/versions/`:
   `CREATE TABLE contact_address_ownership_periods` (§3.1), generated via
   `alembic revision` (never a hand-picked revision id, per
   `voipbin-dbscheme-migration` skill).
2. Backfill step in the **same migration's `upgrade()`**, guarded by an
   `INSERT ... SELECT` from `contact_addresses` (only rows with `contact_id IS
   NOT NULL`), per §8's backfill rule (`valid_from = tm_create`, `valid_to =
   NULL`).
3. Round-trip verify (MariaDB build → mysqldump → MySQL 8.0 import) per the
   `voipbin-dbscheme-migration` skill, including the generated-column UNIQUE
   behavior probes (open-period coexistence, real-duplicate rejection, distinct-
   key non-collision — the skill's three-probe recipe for reviewing any
   generated-column-UNIQUE fix).
4. No `downgrade()` data-loss concern: `DROP TABLE IF EXISTS
   contact_address_ownership_periods` is fully reversible (no other table's data
   depends on it).

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
  `ClaimAddress`/`CreateUnresolvedAddress` against their current implementations,
  to confirm no existing test fixture assumption (e.g. mock call ordering) breaks.
- (Recorded, not blocking implementation — §6.2/§8) If a customer's ownership-period
  count for a single Contact ever grows large enough to make the OR-expanded
  STEP2 query in §6.2 a real performance concern, the fix is switching
  `InteractionList`'s calling convention from an OR-expanded parameter slice to
  a real `JOIN` against `contact_address_ownership_periods` — deferred, not
  designed here, until real evidence of the problem exists.
