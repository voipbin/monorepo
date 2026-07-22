# Case/Interaction Peer+Local Address JSON Unification (Design v0.1)

Repo: voipbin/monorepo
Ticket: NOJIRA
Author: Hermes (CPO) on behalf of pchero (CEO/CTO)
Date: 2026-07-22
Status: Draft, approved to implement (breaking external API change explicitly
approved by CEO -- see §7)

## Changelog

- v0.1 (2026-07-22). Initial draft.

## 1. Motivation

`kase.Case` (`bin-contact-manager/models/kase/kase.go:27-28`) stores the
remote party as a flat pair:

```go
PeerType   commonaddress.Type `json:"peer_type"   db:"peer_type"`
PeerTarget string             `json:"peer_target" db:"peer_target"`
```

Case has never captured the local (self) endpoint at all -- which
number/channel/account the interaction actually came in on. This is a real
gap: `casehandler.GetOrCreate` already receives a `self commonaddress.Address`
parameter (`bin-contact-manager/pkg/casehandler/getorcreate.go:65`), but today
that value is used ONLY for the proactive-link side effect
(`linkSiblingConversation`, getorcreate.go:171-194) and is never persisted
onto the Case row. Once a Case is closed and reopened later, or looked up by
id from an integration, there is no way to tell which of the customer's
numbers/channels the conversation happened on.

`interaction.Interaction` already solved half of this problem for its own
entity: it carries `LocalType`/`LocalTarget` alongside `PeerType`/`PeerTarget`
(`bin-contact-manager/models/interaction/interaction.go:23-29`), added in a
prior migration (`adb8daac2bb0_contact_interactions_add_local_type_.py`).
But both entities still store peer/local as flat, independently-tagged
`VARCHAR` columns rather than as a single `commonaddress.Address` value, even
though `commonaddress.Address` (`bin-common-handler/models/address/main.go`)
is the platform's standard shape for an endpoint (`Type`, `Target`,
`TargetName`, `Name`, `Detail`) and is already used this way elsewhere --
`bin-conversation-manager/models/message/message.go`'s
`Source commonaddress.Address \`json:"source,omitempty" db:"source,json"\``
and `bin-call-manager/models/call/call.go`'s equivalent `Source` field are
the existing precedent for JSON-column `commonaddress.Address` storage.

Per CPO+CEO design discussion, the fix is not "add two new plain columns
to Case for local_type/local_target and call it done." It is to unify BOTH
Case and Interaction onto the same pattern used by Call/Message: a full
`commonaddress.Address` JSON value for `Peer` (and `Local`), with the
existing flat `peer_type`/`peer_target`/`local_type`/`local_target` columns
converted to MySQL `GENERATED ALWAYS ... STORED` columns derived from that
JSON. This:

- Closes the Case-local gap by persisting `self` as `Case.Local` for the
  first time.
- Gives both entities `TargetName`/`Name`/`Detail` metadata on the peer/local
  address for free (currently thrown away by `deriveEndpoints` and by
  `casehandler.Create`'s narrow `peerType commonaddress.Type, peerTarget
  string` parameters).
- Keeps every existing index, unique constraint, and Go read-path query
  (`WHERE peer_type = ? AND peer_target = ?`, etc.) working unchanged,
  because the generated columns are bit-identical in content and type to
  today's plain columns -- only their write path changes.
- Matches the platform-wide convention (Call, Message) instead of Case/
  Interaction being permanently the odd ones out.

This is an intentional, CEO-approved breaking change to the external REST
response shape for Case and Interaction (flat `peer_type`/`peer_target` ->
nested `peer: {type, target, ...}`). No compatibility shim, no dual-field
transition period, no deprecation window -- see §7.

## 2. Current state

### 2.1 Go structs

`kase.Case` (`bin-contact-manager/models/kase/kase.go:20-74`), relevant
fields only:

```go
type Case struct {
	ID         uuid.UUID `json:"id"          db:"id,uuid"`
	CustomerID uuid.UUID `json:"customer_id" db:"customer_id,uuid"`

	PeerType   commonaddress.Type `json:"peer_type"   db:"peer_type"`
	PeerTarget string             `json:"peer_target" db:"peer_target"`

	ReferenceType string `json:"reference_type" db:"reference_type"`
	// ... Name, Detail, ContactID, Owner, Status, OpenedAt, ClosedAt,
	// ClosedReason, ClosedByType, ClosedByID, PreviousCaseID, TagIDs,
	// TMCreate, TMUpdate -- unaffected by this design.
}
```

`interaction.Interaction` (`bin-contact-manager/models/interaction/interaction.go:13-43`):

```go
type Interaction struct {
	ID         uuid.UUID `json:"id"          db:"id,uuid"`
	CustomerID uuid.UUID `json:"customer_id"  db:"customer_id,uuid"`

	Direction string `json:"direction" db:"direction"`

	PeerType   string `json:"peer_type"   db:"peer_type"`
	PeerTarget string `json:"peer_target" db:"peer_target"`

	LocalType   string `json:"local_type"   db:"local_type"`
	LocalTarget string `json:"local_target" db:"local_target"`

	ReferenceType string    `json:"reference_type" db:"reference_type"`
	ReferenceID   uuid.UUID `json:"reference_id"   db:"reference_id,uuid"`

	TMInteraction *time.Time `json:"tm_interaction" db:"tm_interaction"`
	TMCreate      *time.Time `json:"tm_create"       db:"tm_create"`
}
```

### 2.2 Schema

`contact_cases` (`f718e26f2c44_contact_cases_create_table.py:36-89`), relevant
excerpt:

```sql
peer_type     VARCHAR(255) NOT NULL DEFAULT '',
peer_target   VARCHAR(255) NOT NULL DEFAULT '',
...
open_peer_uk BINARY(32) GENERATED ALWAYS AS (
    IF(status = 'open',
       UNHEX(SHA2(CONCAT_WS('|', customer_id, peer_type, peer_target, reference_type), 256)),
       NULL)
) STORED,
UNIQUE INDEX uq_case_open_peer (open_peer_uk),
INDEX idx_case_customer_reftype (customer_id, reference_type)
```

`contact_interactions` (`ac5d4e18060c_contact_crm_create_tables.py:90-122`,
plus `local_type`/`local_target` added by
`adb8daac2bb0_contact_interactions_add_local_type_.py:37-41`):

```sql
peer_type   VARCHAR(255) NOT NULL DEFAULT '',
peer_target VARCHAR(255) NOT NULL DEFAULT '',
local_type   VARCHAR(255) NOT NULL DEFAULT '',   -- added later, adb8daac2bb0
local_target VARCHAR(255) NOT NULL DEFAULT '',   -- added later, adb8daac2bb0
...
UNIQUE INDEX idx_contact_interactions_idem (reference_type, reference_id, peer_target),
INDEX        idx_contact_interactions_peer (customer_id, peer_type, peer_target),
INDEX        idx_contact_interactions_cursor (customer_id, tm_create)
```

Both `peer_type`/`peer_target` today are plain application-level columns
written directly by `commondatabasehandler.PrepareFields` off the flat Go
fields, and read via `commondatabasehandler.GetDBFields`/`ScanRow` the same
way.

### 2.3 Precedent for the target pattern

`bin-conversation-manager/models/message/message.go`:

```go
Source commonaddress.Address `json:"source,omitempty" db:"source,json"`
```

`bin-call-manager/models/call/call.go`:

```go
Source commonaddress.Address `json:"source,omitempty" db:"source,json"`
```

`commondatabasehandler.PrepareFields`/`ScanRow`
(`bin-common-handler/pkg/databasehandler/mapping.go`) already handles the
`,json` conversion tag generically: `convertValueForDB` JSON-marshals any
non-UUID, non-time struct/slice/map value on write
(mapping.go:152-167,199-206), and `scanTarget.copyJSON` unmarshals it back on
read (mapping.go:426-435). No changes to `mapping.go` are needed for this
design -- confirmed with a prior design round.

`contact_addresses.primary_contact_uk`
(`ac5d4e18060c_contact_crm_create_tables.py:66-69`) is an existing simpler
precedent for a `STORED` generated column with NO corresponding Go field at
all (see `bin-contact-manager/pkg/dbhandler/address_test.go`'s
`insertTestAddress` comment). Case/Interaction's `peer_type`/`peer_target`/
`local_type`/`local_target` are different: Go code still needs these as
READ fields (filters, `WHERE` clauses, log lines, tool responses -- see
§6), so the fields cannot simply be dropped from the struct. This design
keeps them as read-only derived fields and special-cases their removal from
the INSERT map (§5).

## 3. New schema

Two new Alembic migrations under `bin-dbscheme-manager/bin-manager/main/versions/`,
generated via `alembic -c alembic.ini revision -m "..."` (never hand-picked
revision ids).

### 3.1 `contact_cases`: add `peer`/`local` JSON, convert flat columns to generated

```
alembic -c alembic.ini revision -m "contact_cases_peer_local_address_json"
```

```sql
-- peer is added nullable first, then tightened to NOT NULL after backfill.
-- Adding a NOT NULL column with no DEFAULT to an already-populated table
-- fails in MySQL strict mode (errno 1364: "Field 'peer' doesn't have a
-- default value"), and even where it didn't fail outright it would leave
-- no row with peer IS NULL for the backfill UPDATE below to match -- the
-- ADD-nullable -> backfill -> MODIFY-NOT-NULL ordering is required, not
-- optional (round-1 design review caught this as a real correctness bug
-- in an earlier draft that added `peer JSON NOT NULL` directly).
ALTER TABLE contact_cases
    ADD COLUMN peer  JSON NULL AFTER customer_id,
    ADD COLUMN local JSON NULL AFTER peer;

-- local stays nullable permanently (see the note below); peer is only
-- transiently nullable during this migration, tightened to NOT NULL
-- after the backfill step.
-- local is nullable: GetOrCreate's `self` parameter can be a zero
-- commonaddress.Address (design VOIP-1243's proactive-link check already
-- treats `self.Type == ""` as "no local endpoint known", see
-- getorcreate.go:99). A zero Address is still valid JSON ('{}' after
-- omitempty strips every field), but modeling "no local known" as SQL NULL
-- is more honest than a JSON value with every field empty, and lets
-- local_type/local_target's generated expressions below compute NULL
-- (matching the pre-existing convention that empty-string columns, not
-- NULL, meant "the value that was stored," which for a never-before-existing
-- column becomes moot -- NULL is now the correct spelling of "not captured").

-- Backfill: no pre-existing rows carry peer/local JSON (this is a brand new
-- pair of columns; the flat peer_type/peer_target values below are used to
-- backfill `peer` so peer_type/peer_target's generated expressions keep
-- producing identical values for existing rows -- open_peer_uk and
-- uq_case_open_peer depend on this).
UPDATE contact_cases
SET peer = JSON_OBJECT('type', peer_type, 'target', peer_target)
WHERE peer IS NULL;

-- Now that every row has a non-NULL peer, tighten the column to NOT NULL
-- (every Case has a peer by construction -- Create/GetOrCreate both
-- require peerType/peerTarget as non-optional arguments -- so this MODIFY
-- cannot fail against the just-backfilled data).
ALTER TABLE contact_cases
    MODIFY COLUMN peer JSON NOT NULL;

-- Drop the old plain columns and re-add them as STORED generated columns
-- derived from `peer`. Column order/type/NOT NULL/DEFAULT are unchanged so
-- every existing index and query keeps working byte-for-byte -- EXCEPT
-- for open_peer_uk/uq_case_open_peer, which MUST be dropped and
-- recreated around this step (round-8 design review finding, confirmed
-- by executing this exact DDL against a real MySQL 8.0.46 instance built
-- with the actual open_peer_uk column present): open_peer_uk is itself a
-- STORED generated column (f718e26f2c44) whose expression directly
-- references peer_type/peer_target
-- (`UNHEX(SHA2(CONCAT_WS('|', customer_id, peer_type, peer_target,
-- reference_type), 256))`). MySQL refuses to DROP a column that another
-- generated column depends on (errno 3108: "Column 'peer_type' has a
-- generated column dependency") -- the migration cannot proceed past the
-- next statement without first removing that dependency.
ALTER TABLE contact_cases
    DROP INDEX uq_case_open_peer,
    DROP COLUMN open_peer_uk;

ALTER TABLE contact_cases
    DROP COLUMN peer_type,
    DROP COLUMN peer_target;

ALTER TABLE contact_cases
    ADD COLUMN peer_type VARCHAR(255)
        GENERATED ALWAYS AS (JSON_UNQUOTE(JSON_EXTRACT(peer, '$.type'))) STORED NOT NULL
        AFTER peer,
    ADD COLUMN peer_target VARCHAR(255)
        GENERATED ALWAYS AS (JSON_UNQUOTE(JSON_EXTRACT(peer, '$.target'))) STORED NOT NULL
        AFTER peer_type,
    ADD COLUMN local_type VARCHAR(255)
        GENERATED ALWAYS AS (JSON_UNQUOTE(JSON_EXTRACT(local, '$.type'))) STORED
        AFTER local,
    ADD COLUMN local_target VARCHAR(255)
        GENERATED ALWAYS AS (JSON_UNQUOTE(JSON_EXTRACT(local, '$.target'))) STORED
        AFTER local_type;

-- Recreate open_peer_uk/uq_case_open_peer now that peer_type/peer_target
-- exist again (as generated columns). The expression is copied verbatim
-- from f718e26f2c44 -- unchanged behavior, just re-attached to the new
-- generated peer_type/peer_target instead of the old plain columns.
ALTER TABLE contact_cases
    ADD COLUMN open_peer_uk BINARY(32) GENERATED ALWAYS AS (
        IF(status = 'open',
           UNHEX(SHA2(CONCAT_WS('|', customer_id, peer_type, peer_target, reference_type), 256)),
           NULL)
    ) STORED AFTER local_target,
    ADD UNIQUE INDEX uq_case_open_peer (open_peer_uk);

-- idx_case_customer_reftype, idx_case_unresolved, idx_case_owner are
-- untouched: none of them reference peer_type/peer_target/open_peer_uk,
-- so they were never affected by the drop-and-recreate sequence above.
```

Downgrade: recreate `peer_type`/`peer_target` as plain `VARCHAR(255) NOT
NULL DEFAULT ''` columns, backfill from `peer_type`/`peer_target`'s current
generated values before dropping them, then drop `local_type`/`local_target`/
`peer`/`local`. (Standard Alembic downgrade symmetry; not spelled out here
verbatim to avoid duplicating boilerplate -- follow the pattern already used
by `adb8daac2bb0`'s downgrade, generated-column variant.)

`peer_type`/`peer_target` do not need a `NOT NULL DEFAULT ''`-style backfill
guard the way `contact_interactions` does (§3.2) because `contact_cases` is
younger and smaller (Case was only introduced 2026-07-07,
`f718e26f2c44`), and by design every row's `peer_type`/`peer_target` were
always non-empty (Case.Create/GetOrCreate never allow an empty peer). The
`UPDATE ... WHERE peer IS NULL` backfill step above covers any rows written
between this migration's authoring and its deployment window.

### 3.2 `contact_interactions`: same treatment, backfill is mandatory here

```
alembic -c alembic.ini revision -m "contact_interactions_peer_local_address_json"
```

```sql
-- peer is added nullable first, then tightened to NOT NULL after backfill --
-- same ADD-nullable -> backfill -> MODIFY-NOT-NULL ordering fix as §3.1
-- (round-1 design review caught the same NOT-NULL-with-no-DEFAULT bug here).
ALTER TABLE contact_interactions
    ADD COLUMN peer  JSON NULL AFTER direction,
    ADD COLUMN local JSON NULL AFTER peer;

-- MANDATORY backfill (unlike message.source's precedent of "no backfill,
-- nullable JSON, historical rows show NULL"): contact_interactions.peer_type/
-- peer_target are NOT NULL and carry a live UNIQUE index
-- (idx_contact_interactions_idem) and a live composite INDEX
-- (idx_contact_interactions_peer) today. A generated column computed from a
-- NULL `peer` JSON value evaluates to NULL, which would flip
-- peer_type/peer_target from NOT NULL to effectively-NULL for every
-- pre-migration row and break the idempotency unique's invariant (MySQL
-- treats NULL as "distinct" under UNIQUE, so historical rows would silently
-- stop deduplicating against re-delivered events). Backfill peer/local JSON
-- from the existing plain columns in the SAME migration transaction, before
-- the plain columns are dropped and regenerated:
UPDATE contact_interactions
SET peer  = JSON_OBJECT('type', peer_type, 'target', peer_target),
    local = IF(local_type = '' AND local_target = '', NULL,
               JSON_OBJECT('type', local_type, 'target', local_target))
WHERE peer IS NULL;

-- Now that every row has a non-NULL peer, tighten the column to NOT NULL
-- (mirrors §3.1's contact_cases fix -- this MODIFY cannot fail against the
-- just-backfilled data since every row now has a non-NULL peer).
ALTER TABLE contact_interactions
    MODIFY COLUMN peer JSON NOT NULL;

-- Drop idx_contact_interactions_idem/idx_contact_interactions_peer BEFORE
-- dropping peer_type/peer_target/local_type/local_target (round-9 design
-- review finding, confirmed by executing this exact sequence against a
-- real MySQL 8.0.46 instance seeded with pre-existing SMS-fan-out-shaped
-- data): dropping peer_target while idx_contact_interactions_idem is
-- still live implicitly shrinks the index to (reference_type,
-- reference_id) as MySQL processes the ALTER, and if any pre-existing
-- rows already share that narrower key -- guaranteed in any real
-- deployment using the fan-out capability this index's 3-column shape
-- was built for (per ac5d4e18060c's own comment: "the bare triple
-- distinguishes SMS fan-out") -- the implicit shrink itself violates the
-- not-yet-fully-dropped unique constraint and the DROP COLUMN statement
-- fails outright with errno 1062 (duplicate entry), not silently. This
-- mirrors §3.1's already-correct index-before-column ordering for
-- open_peer_uk/uq_case_open_peer; §3.2's original ordering had the drop
-- sequence reversed relative to §3.1's pattern, which round 8's test
-- (post-migration inserts only, no pre-existing duplicate-reference rows
-- seeded before running the migration) did not exercise.
ALTER TABLE contact_interactions
    DROP INDEX idx_contact_interactions_idem,
    DROP INDEX idx_contact_interactions_peer;

ALTER TABLE contact_interactions
    DROP COLUMN peer_type,
    DROP COLUMN peer_target,
    DROP COLUMN local_type,
    DROP COLUMN local_target;

ALTER TABLE contact_interactions
    ADD COLUMN peer_type VARCHAR(255)
        GENERATED ALWAYS AS (JSON_UNQUOTE(JSON_EXTRACT(peer, '$.type'))) STORED NOT NULL
        AFTER peer,
    ADD COLUMN peer_target VARCHAR(255)
        GENERATED ALWAYS AS (JSON_UNQUOTE(JSON_EXTRACT(peer, '$.target'))) STORED NOT NULL
        AFTER peer_type,
    ADD COLUMN local_type VARCHAR(255)
        GENERATED ALWAYS AS (JSON_UNQUOTE(JSON_EXTRACT(local, '$.type'))) STORED
        AFTER local,
    ADD COLUMN local_target VARCHAR(255)
        GENERATED ALWAYS AS (JSON_UNQUOTE(JSON_EXTRACT(local, '$.target'))) STORED
        AFTER local_type;

-- Restore idx_contact_interactions_idem/idx_contact_interactions_peer to
-- their FULL original column lists (round-8 design review finding):
-- unlike §3.1's contact_cases case, dropping peer_type/peer_target does
-- NOT error on its own -- MySQL silently SHRINKS a composite index when a
-- trailing column it references is dropped, rather than refusing the
-- drop outright. idx_contact_interactions_idem
-- (ac5d4e18060c: UNIQUE INDEX (reference_type, reference_id, peer_target))
-- would silently narrow to (reference_type, reference_id) alone, and
-- idx_contact_interactions_peer
-- (INDEX (customer_id, peer_type, peer_target)) would silently narrow to
-- (customer_id) alone, if the columns were dropped while these indexes
-- were still attached to them -- which is exactly why both indexes are
-- explicitly DROPped above, BEFORE the column drop (round-9 design
-- review finding: dropping the columns first, while the indexes were
-- still live, made the DROP COLUMN statement itself fail outright with
-- errno 1062 against any table already containing legitimate
-- same-reference/different-peer rows, since the implicit index-shrink
-- MySQL performs mid-ALTER collides with pre-existing data using the
-- fan-out capability this index's 3-column shape was built for). Neither
-- problem (silent shrink, or hard failure against real fan-out data) is
-- possible once the indexes are gone before the columns are ever
-- touched. Re-add both indexes now, at their full original column lists,
-- attached to the newly-recreated generated columns:
ALTER TABLE contact_interactions
    ADD UNIQUE INDEX idx_contact_interactions_idem (reference_type, reference_id, peer_target),
    ADD INDEX        idx_contact_interactions_peer (customer_id, peer_type, peer_target);

-- idx_contact_interactions_cursor (customer_id, tm_create) is untouched:
-- it references neither peer_type/peer_target/local_type/local_target
-- nor any dropped column, so it was never subject to the same-column
-- index-shrinkage risk the two indexes above were.
```

One behavioral wrinkle worth calling out explicitly: pre-migration rows
where `local_type`/`local_target` were both `''` (the pre-adb8daac2bb0
historical default, i.e. local was genuinely never captured) backfill to
`local IS NULL` -> generated `local_type`/`local_target` become SQL `NULL`
instead of `''`. This is a **behavior change** for any Go/SQL code doing
`WHERE local_type = ''` against historical rows -- it must become `WHERE
local_type IS NULL OR local_type = ''` or simply `local_type IS NULL`.
A repo-wide grep for `local_type` (`bin-contact-manager/pkg/dbhandler/*.go`,
`bin-ai-manager`, `bin-flow-manager`) found no such `WHERE local_type = ''`
predicate today, so no call site needs adjustment as of this design, but
mark it explicitly here since it is the one subtle semantic drift baked into
the interactions migration's backfill choice, and add a test asserting it
(§9).

## 4. Go struct changes

### 4.1 `kase.Case` (`bin-contact-manager/models/kase/kase.go`)

Before:

```go
	// PeerType/PeerTarget identify the remote party this Case is scoped
	// to. PeerTarget is normalized via commonaddress.NormalizeTarget --
	// bit-identical to contact_addresses.target and interaction.peer_target.
	PeerType   commonaddress.Type `json:"peer_type"   db:"peer_type"`
	PeerTarget string             `json:"peer_target" db:"peer_target"`
```

After:

```go
	// Peer is the remote party this Case is scoped to, stored as JSON.
	// Peer.Target is normalized via commonaddress.NormalizeTarget --
	// bit-identical to contact_addresses.target and
	// interaction.Peer.Target. NOT NULL: every Case has a peer by
	// construction (Create/GetOrCreate both require it).
	Peer commonaddress.Address `json:"peer" db:"peer,json"`

	// Local is the customer's own endpoint (number/channel/account) the
	// interaction arrived on or was placed from, first captured here by
	// this design (previously discarded -- see docs/plans/
	// 2026-07-22-case-interaction-peer-local-address-json-design.md §1).
	// ALWAYS PRESENT in JSON output (no `omitempty` -- see the note
	// below); a zero Address (no local endpoint known) serializes as
	// `"local":{}`, never an absent key.
	Local commonaddress.Address `json:"local" db:"local,json"`
```

**Go's `encoding/json` `omitempty` has no effect on non-pointer struct
fields** (round-1 design review caught this as a real bug in an earlier
draft that carried `json:"local,omitempty"` on this field): `omitempty`
only suppresses `false`, `0`, a nil pointer/interface/slice/map, or an
empty string -- it does NOT inspect whether every field of a nested
struct value is individually zero. A zero-value `Local` will therefore
always serialize as `"local":{}`, never be omitted, regardless of the
`omitempty` tag's presence. The tag is removed above (keeping it would be
actively misleading, implying behavior that does not exist) and §7's
OpenAPI wording is corrected to match: `local`/`peer` are always-present
keys in the response; an empty `{}` object (all of `CommonAddress`'s own
fields individually omitted via their own correctly-functioning
`omitempty` on plain string fields) is how "no local endpoint known" is
represented on the wire, not a missing key.

An alternative considered and rejected: making `Local *commonaddress.Address`
(a pointer, which DOES support real `omitempty`/JSON `null` semantics).
Rejected because `bin-common-handler/pkg/databasehandler/mapping.go`'s
`convertValueForDB` (`,json` conversion-type branch specifically) has no
nil-pointer check of its own -- only the separate auto-detect code path
below it (used when no explicit conversion type is given) checks
`rv.Kind() == reflect.Ptr && rv.IsNil()`. A nil `*commonaddress.Address`
routed through the explicit `,json` tag path would `json.Marshal` a typed
nil pointer to the 4-byte JSON literal `null` rather than a true SQL NULL,
which is itself workable but requires a `bin-common-handler` fix (a
change to a package used by all 37 services in the monorepo) to do
correctly -- reintroducing exactly the broad-blast-radius change this
design's earlier scope-narrowing round explicitly ruled out. Keeping
`Local` as a non-pointer value and fixing the documentation instead
(rather than the type) keeps this design's `bin-common-handler` footprint
at zero, consistent with every other round's scope decision.

**Storage-representation asymmetry this creates (documented, not fixed,
since it is harmless):** a Case/Interaction row with `local IS NULL`
(migrated historical rows per §3.2's backfill, or via any future direct
SQL write) and a row with `local = '{}'` (any row inserted through Go
code with a zero `self`, since Go always serializes the zero `Local`
value as an empty JSON object, never SQL NULL) are NOT byte-identical at
the raw `local` column level. Both are FUNCTIONALLY equivalent for every
purpose this design cares about: `JSON_EXTRACT('{}', '$.type')` and
`JSON_EXTRACT(NULL, '$.type')` both evaluate to SQL NULL, so
`local_type`/`local_target`'s generated-column values are identical
(NULL) either way, and no code path in this design (or found in the
repo-wide audit, §6.5) ever queries `WHERE local IS NULL` against the raw
JSON column directly -- only the derived `local_type`/`local_target`
scalar columns are ever queried. The asymmetry is real but inert; flagged
here so a future reader who notices `local IS NULL` returning fewer rows
than expected (only migrated historical rows, not newly-inserted
empty-local rows) understands why, rather than treating it as a bug.

**No `PeerType`/`PeerTarget`/`LocalType`/`LocalTarget` Go fields at all.**
Per explicit direction, `contact_cases.peer_type`/`peer_target`/
`local_type`/`local_target` exist ONLY as MySQL `STORED` generated columns
for indexing/query purposes (§3) -- they are never represented as Go
struct fields, are never read back into `kase.Case` via `ScanRow`, and are
never referenced through reflection-driven `GetDBFields`/`PrepareFields`
at all. Every Go call site that needs the type/target of a Case's peer or
local endpoint reads `.Peer.Type`/`.Peer.Target`/`.Local.Type`/
`.Local.Target` directly off the `commonaddress.Address` value -- there is
no derived-field shortcut.

This is a change from the design's earlier draft (v0.1), which kept
`PeerType`/`PeerTarget`/`LocalType`/`LocalTarget` as read-only Go fields
populated by `ScanRow` off the generated columns, guarded by an
insert-time `delete(fields, "peer_type")` workaround in `dbhandler` (§5).
That workaround, and the regression it created (a `Case`/`Interaction`
value returned directly from `Create`/`GetOrCreate` without a re-fetch
would carry zero-value derived fields, since they were never actually set
at construction time -- v0.1's §5.3), are BOTH eliminated entirely by
dropping the Go fields: `commondatabasehandler.PrepareFields`/`GetDBFields`
only ever see fields that exist on the struct, so with no
`PeerType`/`PeerTarget`/`LocalType`/`LocalTarget` field present, they never
attempt to read, write, or select those columns in the first place. There
is nothing to keep in sync, nothing to strip from an INSERT map, and no
stale-in-memory-value trap. See §5 (fully rewritten).

Every existing Go call site that read `.PeerType`/`.PeerTarget` off a
`kase.Case` value (confirmed via repo-wide grep, full list in §6.5) must
change to `.Peer.Type`/`.Peer.Target`; there is no exception.

### 4.2 `interaction.Interaction` (`bin-contact-manager/models/interaction/interaction.go`)

Before:

```go
	// Remote endpoint (the peer's address — match key for read-time contact resolution).
	// peer_target is stored normalized via commonaddress.NormalizeTarget so it is
	// bit-identical to contact_addresses.target.
	PeerType   string `json:"peer_type"   db:"peer_type"`
	PeerTarget string `json:"peer_target" db:"peer_target"`

	// Our local endpoint (for attribution: which number/account received/sent).
	// Not in the idempotency unique; not indexed (attribution only).
	LocalType   string `json:"local_type"   db:"local_type"`
	LocalTarget string `json:"local_target" db:"local_target"`
```

After:

```go
	// Peer is the remote endpoint (match key for read-time contact
	// resolution), stored as JSON. Peer.Target is normalized via
	// commonaddress.NormalizeTarget so it is bit-identical to
	// contact_addresses.target.
	Peer commonaddress.Address `json:"peer" db:"peer,json"`

	// Local is our own endpoint (attribution: which number/account
	// received/sent), stored as JSON. Not in the idempotency unique; not
	// separately indexed (attribution only). ALWAYS PRESENT in JSON
	// output (no `omitempty` -- see kase.Case's §4.1 note, which applies
	// identically here: Go's omitempty has no effect on non-pointer
	// struct fields). A zero Local (historical pre-adb8daac2bb0 rows, or
	// any future event with no known local endpoint) serializes as
	// `"local":{}`; the underlying MySQL column may independently be
	// SQL NULL (migrated historical rows, §3.2) or JSON `'{}'` (any row
	// written by this design's Go code with a zero self) -- both produce
	// identical (NULL) `local_type`/`local_target` generated-column
	// values, per §4.1's storage-asymmetry note, which applies here too.
	Local commonaddress.Address `json:"local" db:"local,json"`
```

Same as `kase.Case` (§4.1): **no `PeerType`/`PeerTarget`/`LocalType`/
`LocalTarget` Go fields.** `contact_interactions.peer_type`/`peer_target`/
`local_type`/`local_target` remain as MySQL generated columns purely to
back `idx_contact_interactions_idem`/`idx_contact_interactions_peer`
(§3.2's existing indexes, unchanged); no Go code ever reads them as struct
fields. Every existing Go call site that read `.PeerType`/`.PeerTarget` off
an `interaction.Interaction` value must change to `.Peer.Type`/
`.Peer.Target` (full list in §6.5).

`import commonaddress "monorepo/bin-common-handler/models/address"` must be
added to `interaction.go` (not previously imported there).

## 5. dbhandler changes (simplified: no insert-time workaround needed)

Because `kase.Case`/`interaction.Interaction` carry no
`PeerType`/`PeerTarget`/`LocalType`/`LocalTarget` Go fields (§4),
`commondatabasehandler.PrepareFields(c)`/`PrepareFields(i)` never produce
`"peer_type"`/`"peer_target"`/`"local_type"`/`"local_target"` map keys in
the first place -- there is nothing to strip before building the INSERT,
and no special-case code is needed in `caseInsertExec`
(`bin-contact-manager/pkg/dbhandler/kase.go`) or `InteractionCreate`
(`bin-contact-manager/pkg/dbhandler/interaction.go`) beyond what already
exists today. `PrepareFields(c)` naturally emits only `"peer"`/`"local"`
(JSON-marshaled per the `,json` db tag) plus every other already-existing
field; MySQL computes `peer_type`/`peer_target`/`local_type`/`local_target`
itself as `STORED` generated columns on insert, exactly as it does for
`contact_addresses.primary_contact_uk` today (§2.3's precedent, which
likewise has zero corresponding Go field).

**`GetDBFields(&kase.Case{})`/`GetDBFields(&interaction.Interaction{})`
(used to build every `SELECT` column list -- `caseGetByIDExec`,
`CaseGetOpenByPeer`, `CaseGetByIDForUpdate`, `CaseListAll`,
`InteractionGet`/`InteractionList*`) will simply never request
`peer_type`/`peer_target`/`local_type`/`local_target` as SELECT columns,
because no Go field maps to them.** This is intentional and matches the
explicit direction that these four columns exist solely as an index/query
target inside raw SQL predicates, not as values ever materialized into a
Go struct.

### 5.1 `dbhandler` predicates that filter/query by peer type+target keep working unchanged

`CaseGetOpenByPeer` and `CaseGetLastClosedByPeerTx`
(`bin-contact-manager/pkg/dbhandler/kase.go`) already take `peerType
commonaddress.Type, peerTarget string` as plain function parameters (not
derived from a `kase.Case` struct field) and build `WHERE peer_type = ? AND
peer_target = ?` predicates using those parameter values directly (kase.go
:167,173-174 and the equivalent `CaseGetLastClosedByPeerTx`). These
predicates reference `peer_type`/`peer_target` as bare column-name strings
in a `sq.Eq{"peer_type": ...}` map literal -- not through
`GetDBFields`/reflection -- so removing the Go struct fields (§4.1) has
**no effect on these queries whatsoever**. The generated columns are still
real, queryable, indexed MySQL columns; only their representation as Go
struct fields is gone.

The same is true of `contact_interactions`' raw-SQL predicates in
`InteractionListByOwnershipPeriods`/`InteractionListUnresolved`
(`bin-contact-manager/pkg/dbhandler/interaction.go:265,428-429`), which
already reference `peer_type`/`peer_target` as hand-written bare column
names, unrelated to any Go struct field.

`uq_case_open_peer`/`idx_contact_interactions_idem`/
`idx_contact_interactions_peer` (§3) are pure MySQL index definitions over
the generated columns -- also entirely unaffected by the Go-side field
removal.

### 5.2 No dbhandler code changes required beyond the `,json` db tag on `Peer`/`Local`

To restate plainly: this section (dbhandler changes) in v0.1 of this design
described an INSERT-time `delete(fields, "peer_type")` workaround. That
workaround, and the regression it implied (v0.1 §5.3), are now moot -- the
fields being deleted no longer exist on the struct to begin with. No
`caseInsertExec`/`InteractionCreate` code changes are needed beyond what
already exists to support the `,json` db tag (already handled generically
by `commondatabasehandler`'s `PrepareFields`/`ScanRow`, confirmed no
changes needed there per §2.3).

## 6. casehandler / contacthandler call-site changes

### 6.0 Internal API surface also consolidates peerType/peerTarget into a single `peer commonaddress.Address`

Per explicit direction: it is not enough for the *storage* layer (Case/
Interaction structs, §4) to move from flat `PeerType`/`PeerTarget` to a
single `Peer commonaddress.Address`. The same consolidation applies to
every internal Go/RPC entry point that today accepts `peerType
commonaddress.Type, peerTarget string` as two separate arguments --
`casehandler.Create`, `casehandler.GetOrCreate` (and the private chain
underneath: `getOrCreateAttempt`/`getOrCreateInTx`/`insertWithRetry`), the
`CaseHandler` interface (`bin-contact-manager/pkg/casehandler/main.go`),
the wire request struct `V1DataCasesPost`
(`bin-contact-manager/pkg/listenhandler/models/request/v1_cases.go`), and
the RPC client `ContactV1CaseCreate`
(`bin-common-handler/pkg/requesthandler/contact_cases.go`). These all
collapse `peerType, peerTarget` into one `peer commonaddress.Address`
parameter, mirroring `self`'s existing shape (`self` was already a full
`commonaddress.Address` parameter on `Create`/`GetOrCreate` before this
design -- `peer` becomes symmetric with it, not an outlier).

This is a genuine simplification, not just symmetry for its own sake: both
existing callers of `ContactV1CaseCreate`
(`bin-flow-manager/pkg/activeflowhandler/actionhandle.go:1367-1373`,
`bin-ai-manager/pkg/aicallhandler/tool.go:558-564`) already do

```go
peerTarget, errNormalize := commonaddress.NormalizeTarget(peer.Type, peer.Target)
if errNormalize != nil {
	log.WithError(errNormalize).Warnf("could not normalize peer target; using raw value. peer_type: %s", peer.Type)
	peerTarget = peer.Target
}

res, errCreate := h.reqHandler.ContactV1CaseCreate(ctx, af.CustomerID, self, peer.Type, peerTarget, referenceType, opt.Name, opt.Detail)
```

i.e. they already hold a full `peer commonaddress.Address` value in scope
and manually shred it into two arguments only to have the callee
immediately re-wrap them. Consolidating removes that round-trip and, as a
side effect, finally threads `peer.TargetName`/`peer.Name`/`peer.Detail`
through to `Case.Peer` (today silently dropped at every one of these call
sites, since only `.Type` and the normalized target ever survive the
`ContactV1CaseCreate` call).

Scope boundary: this consolidation applies to the **casehandler-facing Go
API and the wire/RPC API** -- the layer callers actually construct
`commonaddress.Address` values against. It does NOT extend to
`dbhandler`'s low-level SQL primitives (`CaseGetOpenByPeer`,
`CaseGetLastClosedByPeerTx`, both kase.go), which keep their existing
`(customerID, peerType commonaddress.Type, peerTarget string,
referenceType)` shape unchanged -- these are thin SQL-predicate builders one
layer below where `commonaddress.Address` values are meaningfully held as a
single unit, and `casehandler` extracts `.Type`/`.Target` when calling into
them (see §6.2). Consolidating `dbhandler`'s signatures too would be a
larger, purely-cosmetic change with no caller-side benefit (nothing at that
layer ever holds a full `Address` in scope to begin with) and is out of
scope.

### 6.1 `bin-contact-manager/pkg/casehandler/create.go` and `main.go`'s `CaseHandler` interface

`Create`'s signature collapses `peerType commonaddress.Type, peerTarget
string` into one `peer commonaddress.Address` parameter (§6.0), alongside
composing `Peer`/`Local` on the constructed struct:

Before (create.go:24-46):

```go
func (h *caseHandler) Create(
	ctx context.Context,
	customerID uuid.UUID,
	self commonaddress.Address,
	peerType commonaddress.Type,
	peerTarget, referenceType, name, detail string,
) (*kase.Case, error) {
	now := h.utilHandler.TimeNow()

	newCase := &kase.Case{
		ID:             h.utilHandler.UUIDCreate(),
		CustomerID:     customerID,
		PeerType:       peerType,
		PeerTarget:     peerTarget,
		ReferenceType:  referenceType,
		Name:           name,
		Detail:         detail,
		Status:         kase.StatusOpen,
		OpenedAt:       now,
		PreviousCaseID: nil,
		TMCreate:       now,
		TMUpdate:       now,
	}
```

After:

```go
func (h *caseHandler) Create(
	ctx context.Context,
	customerID uuid.UUID,
	self commonaddress.Address,
	peer commonaddress.Address,
	referenceType, name, detail string,
) (*kase.Case, error) {
	now := h.utilHandler.TimeNow()

	newCase := &kase.Case{
		ID:             h.utilHandler.UUIDCreate(),
		CustomerID:     customerID,
		Peer:           peer,
		Local:          self,
		ReferenceType:  referenceType,
		Name:           name,
		Detail:         detail,
		Status:         kase.StatusOpen,
		OpenedAt:       now,
		PreviousCaseID: nil,
		TMCreate:       now,
		TMUpdate:       now,
	}
```

Note `Peer: peer` (the whole `Address`, not just `.Type`/`.Target` rebuilt
into a new literal) -- this is what carries `TargetName`/`Name`/`Detail`
through for the first time (§6.0).

`CaseHandler` interface (`bin-contact-manager/pkg/casehandler/main.go`)
gets the matching signature update:

```go
// Before
Create(ctx context.Context, customerID uuid.UUID, self commonaddress.Address, peerType commonaddress.Type, peerTarget, referenceType, name, detail string) (*kase.Case, error)

// After
Create(ctx context.Context, customerID uuid.UUID, self, peer commonaddress.Address, referenceType, name, detail string) (*kase.Case, error)
```

`self` is already the `Create` signature's caller-supplied local endpoint
(create.go:27, previously accepted but never used inside the function body
before this design -- callers already pass it). Assigning it straight to
`Local` requires no further change beyond the parameter-consolidation
already described.

**New explicit guard: reject an empty `peer.Type`/`peer.Target` before
insert (round-2 design review finding).** Before this design,
`PeerType commonaddress.Type`/`PeerTarget string` (no `omitempty`) wrote a
real empty-string value into the plain `peer_type`/`peer_target VARCHAR(255)
NOT NULL DEFAULT ''` columns even when empty -- a zero-value peer silently
succeeded at insert time. After this design, `commonaddress.Address`'s own
`Type`/`Target` fields carry `json:"...,omitempty"`
(`bin-common-handler/models/address/main.go:5-6`), so a zero-value
`peer commonaddress.Address{}` marshals to JSON `{}`, and
`JSON_EXTRACT('{}', '$.type')` evaluates to SQL NULL, which violates
`peer_type`'s `NOT NULL` constraint on the generated column -- the INSERT
fails outright. §4.1's own claim ("NOT NULL: every Case has a peer by
construction") was, before this fix, an unenforced assumption ("Create/
GetOrCreate require it as a *parameter*" is not the same as "reject it if
it's *empty*") rather than an actual invariant, and nothing in
`create.go`/`getorcreate.go` validated `peer.Type`/`peer.Target` were
non-empty before this finding.

Fix: add an explicit guard at the top of `Create` (and `GetOrCreate`,
§6.2) that rejects an empty peer with a typed error, turning the
previously-implicit assumption into an enforced invariant instead of
relying on it being true by construction at every call site forever:

```go
func (h *caseHandler) Create(
	ctx context.Context,
	customerID uuid.UUID,
	self, peer commonaddress.Address,
	referenceType, name, detail string,
) (*kase.Case, error) {
	if peer.Type == "" || peer.Target == "" {
		return nil, cerrors.InvalidArgument(
			commonoutline.ServiceNameContactManager,
			"CASE_PEER_REQUIRED",
			"peer.type and peer.target are required and cannot be empty.",
		)
	}

	now := h.utilHandler.TimeNow()
	...
```

`cerrors.InvalidArgument` follows the same typed-error convention already
used elsewhere in this file (`create.go`'s existing `cerrors.AlreadyExists`/
`cerrors.Unavailable` translations, §2). This converts a previously silent
INSERT-time failure (a raw MySQL generated-column constraint violation
surfacing as an opaque driver error, §9 test item below) into a clear,
typed, pre-insert validation error callers can act on.

### 6.2 `bin-contact-manager/pkg/casehandler/getorcreate.go`

Same consolidation as §6.1, applied to `GetOrCreate` and its private
implementation chain (`getOrCreateAttempt` -> `getOrCreateInTx` ->
`insertWithRetry`). `insertWithRetry` (getorcreate.go:276-297) builds the
actual insert struct; `GetOrCreate`/`getOrCreateAttempt`/`getOrCreateInTx`
thread `self` down to it today only for the post-commit
`linkSiblingConversation` side effect (getorcreate.go:99-101). `self` must
now also reach `insertWithRetry` so it can be persisted onto `Local`, at the
same time `peerType, peerTarget` collapse into `peer`.

`CaseHandler.GetOrCreate` interface signature:

```go
// Before
GetOrCreate(ctx context.Context, customerID uuid.UUID, self commonaddress.Address, peerType commonaddress.Type, peerTarget, referenceType string, caseIDHint *uuid.UUID) (*kase.Case, error)

// After
GetOrCreate(ctx context.Context, customerID uuid.UUID, self, peer commonaddress.Address, referenceType string, caseIDHint *uuid.UUID) (*kase.Case, error)
```

**Same empty-peer guard as §6.1, at the top of `GetOrCreate` itself**
(before any of `getOrCreateAttempt`/`getOrCreateInTx`/`insertWithRetry`
run, so a bad `peer` is rejected before any DB round trip, not partway
through the retry loop):

```go
func (h *caseHandler) GetOrCreate(
	ctx context.Context,
	customerID uuid.UUID,
	self, peer commonaddress.Address,
	referenceType string,
	caseIDHint *uuid.UUID,
) (*kase.Case, error) {
	if peer.Type == "" || peer.Target == "" {
		return nil, cerrors.InvalidArgument(
			commonoutline.ServiceNameContactManager,
			"CASE_PEER_REQUIRED",
			"peer.type and peer.target are required and cannot be empty.",
		)
	}
	...
```

`insertWithRetry` itself does NOT need its own duplicate guard: both of
its callers (`Create` via §6.1, `GetOrCreate` via this section) already
validate `peer` before `insertWithRetry` is ever reached, and
`insertWithRetry` has no other caller (`Continue`, §6.5, never calls
`insertWithRetry` with a peer at all -- it passes `source.Peer`, which was
already validated when `source` itself was first created).

Before (getorcreate.go:276-297, and its two call sites at 251 and 267):

```go
func (h *caseHandler) insertWithRetry(
	ctx context.Context,
	tx *sql.Tx,
	customerID uuid.UUID,
	peerType commonaddress.Type,
	peerTarget, referenceType string,
	previousCaseID *uuid.UUID,
	now *time.Time,
) (*kase.Case, bool, error) {
	for attempt := 0; attempt < maxInsertRetries; attempt++ {
		newCase := &kase.Case{
			ID:             h.utilHandler.UUIDCreate(),
			CustomerID:     customerID,
			PeerType:       peerType,
			PeerTarget:     peerTarget,
			ReferenceType:  referenceType,
			Status:         kase.StatusOpen,
			OpenedAt:       now,
			PreviousCaseID: previousCaseID,
			TMCreate:       now,
			TMUpdate:       now,
		}
```

After: add a `self commonaddress.Address` parameter to `insertWithRetry`,
threaded from `getOrCreateInTx`'s existing `self`-less signature (which
itself needs `self` added and threaded from `getOrCreateAttempt`, which
already has `self` in scope at line 127); `peerType commonaddress.Type,
peerTarget string` collapse into one `peer commonaddress.Address`
parameter -- and set `Local: self`/`Peer: peer` on the constructed
`newCase`:

```go
func (h *caseHandler) insertWithRetry(
	ctx context.Context,
	tx *sql.Tx,
	customerID uuid.UUID,
	self, peer commonaddress.Address,
	referenceType string,
	previousCaseID *uuid.UUID,
	now *time.Time,
) (*kase.Case, bool, error) {
	for attempt := 0; attempt < maxInsertRetries; attempt++ {
		newCase := &kase.Case{
			ID:             h.utilHandler.UUIDCreate(),
			CustomerID:     customerID,
			Peer:           peer,
			Local:          self,
			ReferenceType:  referenceType,
			Status:         kase.StatusOpen,
			OpenedAt:       now,
			PreviousCaseID: previousCaseID,
			TMCreate:       now,
			TMUpdate:       now,
		}
```

`getOrCreateInTx` (getorcreate.go:199-268) needs `self commonaddress.Address`
added to its own parameter list (it currently only takes `peerType`/
`peerTarget`/`referenceType`, not `self` -- `self` today stops at
`getOrCreateAttempt`, getorcreate.go:124-131) and must pass both `self` and
the now-consolidated `peer` through to both `insertWithRetry` call sites
(getorcreate.go:251, 267). Its own calls into `dbhandler`'s
`CaseGetOpenByPeer`/`CaseGetLastClosedByPeerTx` (getorcreate.go:227, 256)
keep passing `peer.Type, peer.Target` as discrete arguments, unchanged --
those dbhandler primitives are explicitly out of scope for the
consolidation (§6.0). `getOrCreateAttempt`'s existing call to
`getOrCreateInTx` at line 145 gains one more argument (`self`, already in
its own scope) and passes the consolidated `peer` in place of the previous
`peerType, peerTarget` pair.

Net effect: `Case.Local` is populated with exactly the `self` value the
caller (Flow's `case_create` action / AI's `case_create` tool /
`bin-api-manager`'s direct `Create`/`GetOrCreate` RPC handlers) already
supplies today -- no new information has to be sourced, this design only
stops discarding it. `Case.Peer` additionally now carries
`TargetName`/`Name`/`Detail`, previously discarded at every caller (§6.0).

### 6.3 Wire layer: `V1DataCasesPost`, `ContactV1CaseCreate`, and the two RPC callers

`V1DataCasesPost`
(`bin-contact-manager/pkg/listenhandler/models/request/v1_cases.go:90-98`):

Before:

```go
type V1DataCasesPost struct {
	CustomerID    uuid.UUID             `json:"customer_id"`
	Self          commonaddress.Address `json:"self"`
	PeerType      commonaddress.Type    `json:"peer_type"`
	PeerTarget    string                `json:"peer_target"`
	ReferenceType string                `json:"reference_type"`
	Name          string                `json:"name,omitempty"`
	Detail        string                `json:"detail,omitempty"`
}
```

After:

```go
type V1DataCasesPost struct {
	CustomerID    uuid.UUID             `json:"customer_id"`
	Self          commonaddress.Address `json:"self"`
	Peer          commonaddress.Address `json:"peer"`
	ReferenceType string                `json:"reference_type"`
	Name          string                `json:"name,omitempty"`
	Detail        string                `json:"detail,omitempty"`
}
```

`bin-contact-manager/pkg/listenhandler/v1_cases.go`'s `processV1CasesPost`
(line 109) changes its one call site accordingly:

```go
// Before
res, err := h.caseHandler.Create(ctx, body.CustomerID, body.Self, body.PeerType, body.PeerTarget, body.ReferenceType, body.Name, body.Detail)

// After
res, err := h.caseHandler.Create(ctx, body.CustomerID, body.Self, body.Peer, body.ReferenceType, body.Name, body.Detail)
```

`ContactV1CaseCreate`
(`bin-common-handler/pkg/requesthandler/contact_cases.go:26-61`):

Before:

```go
func (r *requestHandler) ContactV1CaseCreate(
	ctx context.Context,
	customerID uuid.UUID,
	self commonaddress.Address,
	peerType commonaddress.Type,
	peerTarget, referenceType, name, detail string,
) (*cmkase.Case, error) {
	uri := "/v1/cases"

	data := &cmrequest.V1DataCasesPost{
		CustomerID:    customerID,
		Self:          self,
		PeerType:      peerType,
		PeerTarget:    peerTarget,
		ReferenceType: referenceType,
		Name:          name,
		Detail:        detail,
	}
```

After:

```go
func (r *requestHandler) ContactV1CaseCreate(
	ctx context.Context,
	customerID uuid.UUID,
	self commonaddress.Address,
	peer commonaddress.Address,
	referenceType, name, detail string,
) (*cmkase.Case, error) {
	uri := "/v1/cases"

	data := &cmrequest.V1DataCasesPost{
		CustomerID:    customerID,
		Self:          self,
		Peer:          peer,
		ReferenceType: referenceType,
		Name:          name,
		Detail:        detail,
	}
```

The two existing callers (`bin-flow-manager`'s
`actionHandleCaseCreate`, `bin-ai-manager`'s `toolHandleCaseCreate`) both
already hold a `peer commonaddress.Address` in scope and only shred it into
`.Type`/normalized-target today (§6.0). Both change identically -- build one
consolidated `Address` (normalized `Target`, original `TargetName`/`Name`/
`Detail` preserved) and pass it as a single argument:

`bin-flow-manager/pkg/activeflowhandler/actionhandle.go:1367-1373`:

```go
// Before
peerTarget, errNormalize := commonaddress.NormalizeTarget(peer.Type, peer.Target)
if errNormalize != nil {
	log.WithError(errNormalize).Warnf("could not normalize peer target; using raw value. peer_type: %s", peer.Type)
	peerTarget = peer.Target
}

res, errCreate := h.reqHandler.ContactV1CaseCreate(ctx, af.CustomerID, self, peer.Type, peerTarget, referenceType, opt.Name, opt.Detail)

// After
peerTarget, errNormalize := commonaddress.NormalizeTarget(peer.Type, peer.Target)
if errNormalize != nil {
	log.WithError(errNormalize).Warnf("could not normalize peer target; using raw value. peer_type: %s", peer.Type)
	peerTarget = peer.Target
}
peerAddr := peer
peerAddr.Target = peerTarget // override with the normalized value; TargetName/Name/Detail pass through unchanged

res, errCreate := h.reqHandler.ContactV1CaseCreate(ctx, af.CustomerID, self, peerAddr, referenceType, opt.Name, opt.Detail)
```

`bin-ai-manager/pkg/aicallhandler/tool.go:558-564` changes identically
(`created, errCreate := h.reqHandler.ContactV1CaseCreate(ctx, c.CustomerID, self, peerAddr, referenceType, tmpOpt.Name, tmpOpt.Detail)`).

No other `ContactV1CaseCreate`/`casehandler.Create`/`casehandler.GetOrCreate`
callers exist in the monorepo today (confirmed via grep across all
`bin-*-manager` directories) -- §8's downstream-impact audit for
`bin-flow-manager`/`bin-ai-manager` (which previously covered only the
Case-response read side) now also covers this request-side signature
change, since both files are edited here.

### 6.4 `bin-contact-manager/pkg/contacthandler/interaction.go`

`EventCallCreated`/`EventConversationMessageCreated` both already compute
normalized `peer`/`local` `commonaddress.Address` values with
`deriveEndpoints` + `commonaddress.NormalizeTarget` before constructing the
flat literal (interaction.go:88-105, 141-158). Only the final struct literal
changes -- the derivation/normalization logic is untouched.

Before (interaction.go:115-127, and the near-identical 169-181):

```go
	i := interaction.Interaction{
		ID:            id,
		CustomerID:    m.CustomerID,
		Direction:     string(m.Direction),
		PeerType:      string(peer.Type),
		PeerTarget:    peerTarget,
		LocalType:     string(local.Type),
		LocalTarget:   localTarget,
		ReferenceType: "call",
		ReferenceID:   m.ID,
		TMInteraction: m.TMCreate,
		TMCreate:      now,
	}
```

After:

```go
	i := interaction.Interaction{
		ID:            id,
		CustomerID:    m.CustomerID,
		Direction:     string(m.Direction),
		Peer:          commonaddress.Address{Type: peer.Type, Target: peerTarget, TargetName: peer.TargetName, Name: peer.Name, Detail: peer.Detail},
		Local:         commonaddress.Address{Type: local.Type, Target: localTarget, TargetName: local.TargetName, Name: local.Name, Detail: local.Detail},
		ReferenceType: "call",
		ReferenceID:   m.ID,
		TMInteraction: m.TMCreate,
		TMCreate:      now,
	}
```

The `Target` field is explicitly overridden with the already-normalized
`peerTarget`/`localTarget` local variables (not `peer.Target`/`local.Target`
directly) -- this is the exact same value the pre-existing code path already
stored, so `interaction.Peer.Target`'s generated `peer_target` column
remains bit-identical to `contact_addresses.target`, preserving read-time
contact-matching (§1's key gotcha). `TargetName`/`Name`/`Detail` are new
metadata this design now preserves (previously discarded when only
`peer.Type`/`peerTarget` were kept); their presence is additive and does not
change any existing matching/index behavior since none of the generated
columns derive from them.

The equivalent literal at interaction.go:169-181
(`EventConversationMessageCreated`) changes identically.

`isCRMEligiblePeer(peer.Type)` (interaction.go:90, 143) is unaffected --
still checks the plain `commonaddress.Type` returned by `deriveEndpoints`,
before it is wrapped into the `Peer`/`Local` Address literals.

### 6.5 Every other Go call site reading `.PeerType`/`.PeerTarget`/`.LocalType`/`.LocalTarget` off a `Case`/`Interaction` value

Per §4's field removal, ALL of the following must change from
`.PeerType`/`.PeerTarget` (etc.) to `.Peer.Type`/`.Peer.Target` (etc.).
Found via a repo-wide grep across every `bin-*-manager` for
`\.PeerType\b|\.PeerTarget\b|\.LocalType\b|\.LocalTarget\b` on `Case`/
`Interaction`-typed receivers (as opposed to unrelated types that happen to
share a field name, e.g. OpenAPI `GetContactInteractionsParams.PeerType`
query-param bindings, which are untouched -- see the note at the end of
this section):

**`bin-contact-manager/pkg/casehandler/lifecycle.go:157`** (`Continue`) --
this call site was missed by the design's earlier drafts (§6.1/§6.2 above
cover `Create`/`GetOrCreate`, but `Continue` is a third, separate
`insertWithRetry` caller):

```go
// Before
res, _, err := h.insertWithRetry(ctx, tx, customerID, source.PeerType, source.PeerTarget, source.ReferenceType, &source.ID, now)

// After
res, _, err := h.insertWithRetry(ctx, tx, customerID, commonaddress.Address{}, source.Peer, source.ReferenceType, &source.ID, now)
```

Note: `Continue` has no `self`/local endpoint of its own to carry over --
it is re-opening a previously closed Case, not projecting a fresh
call/message event -- so it passes a zero `commonaddress.Address{}` for
`insertWithRetry`'s `self` parameter, exactly mirroring what today's code
does for `Local`/`LocalType`/`LocalTarget` (never set by `Continue` before
this design either -- `source.PeerType, source.PeerTarget` were the only
two fields `Continue` ever carried forward, and per §1's own Local-capture
gap, `Local` was never captured by ANY code path before this design). This
is unchanged behavior, not a regression: the re-opened Case's `Local`
remains unset either way, consistent with the pre-existing gap this design
elsewhere fixes only for `Create`/`GetOrCreate` (not `Continue`, which
has no `self` in its own signature to source one from -- `Continue`'s
signature is `(ctx, customerID, id uuid.UUID, callerType
commonidentity.OwnerType, callerID uuid.UUID, callerIsAdmin bool)`, no
address parameter at all). A future design could extend `Continue` to
also accept a fresh `self` value for the re-opened Case's `Local`, but
that is a new capability, not something this design's Peer/Local
consolidation implies -- left out of scope here, flagged in §10.

**`bin-api-manager/pkg/servicehandler/case_message.go`** -- five reads of
`c.PeerType`/`c.PeerTarget` off a `*cmkase.Case` (`c`, obtained via
`h.caseGet`, i.e. a genuine DB-backed read -- not a bare post-insert
in-memory value, so this call site is unaffected by any construction-time
concern, purely a field-rename):

```go
// Before (case_message.go:121)
if destination != c.PeerTarget {

// After
if destination != c.Peer.Target {
```

```go
// Before (case_message.go:165-173)
selfAddr := commonaddress.Address{
	Type:   c.PeerType,
	Target: source,
}
peerAddr := commonaddress.Address{
	Type:   c.PeerType,
	Target: destination,
}
conversationType := caseMessagePeerTypeToConversationType(c.PeerType)

// After
selfAddr := commonaddress.Address{
	Type:   c.Peer.Type,
	Target: source,
}
peerAddr := commonaddress.Address{
	Type:   c.Peer.Type,
	Target: destination,
}
conversationType := caseMessagePeerTypeToConversationType(c.Peer.Type)
```

`caseMessagePeerTypeToConversationType`'s own parameter (`peerType
commonaddress.Type`) and its doc comment (case_message.go:19-20,
referencing "Case's PeerType") are unaffected in signature -- only the
call-site argument expression changes from `c.PeerType` to `c.Peer.Type`;
the comment should be updated to say "Case's Peer.Type" for accuracy but
this is cosmetic.

`bin-api-manager/pkg/servicehandler/case_message_test.go` constructs
`cmkase.Case{PeerTarget: "...", PeerType: commonaddress.TypeTel, ...}`
literals directly at multiple table-test cases (lines 131, 173, 228, 240,
292, 350, 413-414, 496-497, 613, 662-663) -- every one of these must change
to `Peer: commonaddress.Address{Type: commonaddress.TypeTel, Target:
"..."}`. This includes the regression test
`Test_CaseMessageSend_SelfAndPeerTypeMatch_WhatsApp` (line 628), whose own
doc comment (line 624, "case's PeerType (not be hardcoded to TypeTel)")
should be updated to say "case's Peer.Type" for accuracy.

**Test-fixture construction sites -- exhaustive count, not a curated
sample (round-2 design review found round-1's "authoritative list" claim
was still false against the repo -- ~14 additional files were missed. This
revision replaces per-line enumeration with a package-wide grep count,
because round 1 and round 2 both demonstrated that hand-curated line
lists drift from the actual repo state and cannot be trusted to be
complete by inspection alone. Round 3 found the grep itself was ALSO
under-scoped -- limited to `bin-contact-manager` only, missing a sibling
file in `bin-api-manager` that also constructs raw `interaction.
Interaction` literals. This revision widens the grep to the whole
monorepo (`grep -rlE "PeerType:|PeerTarget:|LocalType:|LocalTarget:" .
--include="*_test.go"` run from the monorepo root, not scoped to any
single service), which is the actual correct scope given `interaction.
Interaction`/`kase.Case` are types other services can and do import
directly and construct literals of:**

| File | Occurrences |
|---|---|
| `bin-contact-manager/models/kase/kase_test.go` | 4 |
| `bin-contact-manager/pkg/contacthandler/interaction_test.go` | 12 |
| `bin-contact-manager/pkg/contacthandler/interaction_read_test.go` | 1 |
| `bin-contact-manager/pkg/casehandler/lifecycle_close_test.go` | 3 |
| `bin-contact-manager/pkg/dbhandler/address_ownership_read_test.go` | 4 |
| `bin-contact-manager/pkg/casehandler/lifecycle_continue_test.go` | 3 |
| `bin-contact-manager/pkg/casehandler/getorcreate_timeout_test.go` | 1 |
| `bin-contact-manager/pkg/casehandler/getorcreate_fresh_test.go` | 1 |
| `bin-contact-manager/pkg/casehandler/case_list_get_test.go` | 5 |
| `bin-contact-manager/pkg/casehandler/assign_test.go` | 2 |
| `bin-contact-manager/pkg/casehandler/getorcreate_proactive_link_test.go` | 1 |
| `bin-contact-manager/pkg/casehandler/getorcreate_hint_test.go` | 3 |
| `bin-contact-manager/pkg/casehandler/case_tag_test.go` | 6 |
| `bin-contact-manager/pkg/dbhandler/kase_list_test.go` | 8 |
| `bin-contact-manager/pkg/casehandler/getorcreate_race_test.go` | 1 |
| `bin-contact-manager/pkg/casehandler/casenote_isolation_test.go` | 3 |
| `bin-contact-manager/pkg/casehandler/unresolved_queue_test.go` | 3 |
| `bin-contact-manager/pkg/casehandler/contact_update_test.go` | 6 |
| `bin-contact-manager/pkg/casehandler/getorcreate_test.go` | 2 |
| `bin-contact-manager/pkg/dbhandler/resolution_test.go` | 4 |
| `bin-contact-manager/pkg/casehandler/casenote_test.go` | 3 |
| `bin-contact-manager/pkg/dbhandler/kase_test.go` | 37 |
| `bin-contact-manager/pkg/dbhandler/interaction_test.go` | 56 |
| `bin-ai-manager/pkg/aicallhandler/tool_insight_test.go` | 12 |
| `bin-api-manager/pkg/servicehandler/case_message_test.go` | 13 |
| `bin-api-manager/pkg/servicehandler/interaction_test.go` | 6 |
| **Total** | **200 occurrences across 26 files** |

`bin-ai-manager/pkg/aicallhandler/tool_insight_test.go` and
`bin-api-manager/pkg/servicehandler/case_message_test.go` are already
covered in detail elsewhere in this design (§8.3 and §6.5's earlier
`case_message_test.go` discussion respectively) -- they are included in
this table purely so the table's own total is a complete, single source
of truth, not because they are newly discovered here.

`bin-api-manager/pkg/servicehandler/interaction_test.go` (lines 167-168,
174-175, 194-195) is the round-3 finding: it constructs
`cminteraction.Interaction{PeerType: "tel", PeerTarget:
"+155****1111", ...}` literals directly (three occurrences of the pair),
exactly like `case_message_test.go` in the same directory does for
`Case` -- it was missed because §6.5's grep was scoped to
`bin-contact-manager` only in the prior revision, and this file lives in
`bin-api-manager`. It requires the same mechanical rewrite as every other
row in this table: `Peer: commonaddress.Address{Type: "tel", Target:
"+155****1111"}`.

One false-positive excluded after manual verification: `bin-conversation-
manager/pkg/conversationhandler/create_and_execute_flow_test.go` matches
the same grep pattern (`variableConversationPeerType`/
`variableConversationPeerTarget`, 2 occurrences) but these are unrelated
`Conversation`-model flow-variable name constants (`cv.Peer.Type`/
`cv.Peer.Target` off a completely different `Conversation` struct, not
`kase.Case`/`interaction.Interaction`) -- confirmed by reading the file
directly, not a Case/Interaction call site, and excluded from the table
above.

**Round-4 design review additionally required checking the analogous
non-test grep** (`grep -rlE "PeerType:|PeerTarget:|LocalType:|LocalTarget:"
. --include="*.go" | grep -v _test.go`, run monorepo-wide) to confirm no
PRODUCTION file was left silently unaddressed the way a test-only sweep
could miss: this surfaces `bin-conversation-manager/pkg/
conversationhandler/variable.go:43,45` (`variableConversationPeerTarget:
cv.Peer.Target` / `variableConversationPeerType:
string(cv.Peer.Type)`) -- the non-test source of the same map-key
constants the test file above builds fixtures for. Confirmed by direct
read: `cv` here is `conversation.Conversation`
(`bin-conversation-manager/models/conversation`), an entirely separate
model from `kase.Case`/`interaction.Interaction` with its own unrelated
`Peer commonaddress.Address` field (already following the same
Address-as-JSON pattern this design extends to Case/Interaction, per §2's
precedent list) -- this is the same benign false positive as its test
counterpart, not a second instance of it, and requires no change under
this design. No other production (non-test) `.go` file anywhere in the
monorepo matched the grep pattern.

Every occurrence is a `PeerType:`/`PeerTarget:`/`LocalType:`/`LocalTarget:`
struct-literal field in a `kase.Case{...}` or `interaction.Interaction{...}`
construction and must become `Peer: commonaddress.Address{Type: ...,
Target: ...}` / `Local: commonaddress.Address{Type: ..., Target: ...}`
(dropping `LocalType`/`LocalTarget` entirely into the single `Local`
field, per §4). The transformation pattern is mechanical and identical at
every site (four flat fields collapse into two nested-Address fields), so
this design does not walk through all 200 individually -- the pattern is
fully specified by §4.1/§4.2's Go struct diff and the single worked
example already shown for `create_test.go` below.

**Old-signature `GetOrCreate(...)`/`Create(...)` call sites -- also
exhaustively counted, previously omitted entirely for `GetOrCreate`
(round-2 finding):**

`grep -rlE "\.GetOrCreate\(ctx" bin-contact-manager/pkg/casehandler` finds
8 files calling the OLD 6-parameter `GetOrCreate(ctx, customerID, self,
peerType, peerTarget, referenceType, caseIDHint)` signature, 18 call sites
total: `getorcreate_proactive_link_test.go` (4), `getorcreate_deadlock_test.go`
(5), `getorcreate_race_test.go` (3), `getorcreate_fresh_test.go` (2),
`getorcreate_hint_test.go` (2), `getorcreate_timeout_test.go` (1),
`getorcreate_test.go` (1). Every one collapses `peerType, peerTarget` into
a single `peer commonaddress.Address` argument per §6.2's signature
change, identically to the worked `Create(...)` example below.
`getorcreate_concurrency_test.go` (cited earlier in this section for its
`.PeerTarget` assertion) also calls `GetOrCreate` and is included in this
same rewrite, not a separate case.

Worked example (`bin-contact-manager/pkg/casehandler/create_test.go
:111,117`), representative of the mechanical pattern applied to all 200
literal sites and 18 `GetOrCreate` call sites above:

```go
// Before
if _, err := h.Create(ctx, customerID, self, peerType, peerTarget, referenceType, "", ""); err != nil {

// After
if _, err := h.Create(ctx, customerID, self, peer, referenceType, "", ""); err != nil {
```

where `peer := commonaddress.Address{Type: peerType, Target: peerTarget}`
replaces the test's separate `peerType`/`peerTarget` local variables (or
is constructed inline at each call site, whichever keeps that specific
test file's diff smaller -- left to implementation-time judgment per file,
not prescribed uniformly here since the two existing local-variable
patterns across these 23 files are not identical).

**`bin-ai-manager/pkg/aicallhandler/tool_insight.go:120,150-153`** and
`tool_insight_test.go:70-77,91-97,173` (already discussed in §8.3, restated
here for completeness of this section's audit): `string(kase.PeerType),
kase.PeerTarget` -> `string(kase.Peer.Type), kase.Peer.Target`; `it.PeerType,
it.PeerTarget` -> `it.Peer.Type, it.Peer.Target`. Unlike v0.1 of this
design (which claimed these "compile and behave identically without
modification" because the fields still existed, just read-only), **these
DO now require a code change** since the fields no longer exist on the
struct at all -- §8.3 is updated accordingly.

**Not affected (false positives from the grep, listed to be explicit that
they were checked and excluded):** `bin-api-manager/server/
contact_interactions.go:47-53`, `service_agents_contact_interactions.go
:48-54`, and the corresponding `gens/openapi_server/gen.go` binding code
reference `params.PeerType`/`params.PeerTarget` -- these are OpenAPI
query-parameter struct fields (`GetContactInteractionsParams`, generated
from the `peer_type`/`peer_target` query-string filter parameters on `GET
/contact_interactions`), not `interaction.Interaction` struct fields. This
design does not touch `GET /contact_interactions`'s query-string filter
contract (§8.1's square-admin note already flagged this distinction: "this
query-string filter usage is a request-body concern, not a response-shape
concern, and is unaffected by this design"). They remain `peer_type`/
`peer_target` flat query params exactly as today.

## 7. OpenAPI schema changes

**This is the approved breaking change to the external REST contract.**
`bin-api-manager/pkg/servicehandler/case.go`'s top-of-file comment
(case.go:16-23) already documents that `cmkase.Case` is returned to the
external API AS-IS (no `WebhookMessage` conversion layer, unlike most other
resources) -- "the OpenAPI schema in bin-openapi-manager mirrors the struct
fields and acts as the publication boundary." The same is true for
`interaction.Interaction`: no `webhook.go` exists in
`bin-contact-manager/models/interaction/` (confirmed -- only `interaction.go`
and `list_response.go`), so changing the Go struct in §4 directly changes
what `GET /contact_cases/{id}`, `GET /contact_cases`, `GET
/contact_interactions`, etc. return. No compatibility shim, no dual
flat-and-nested output, no deprecation window -- the CEO has explicitly
approved breaking this response shape.

`bin-openapi-manager/openapi/openapi.yaml` already defines a reusable
`CommonAddress` schema (openapi.yaml:3380-3383+, used today by Call's
source/destination and elsewhere). Reuse it via `$ref`.

### 7.1 `ContactManagerCase` (openapi.yaml:3793+)

Before (excerpt, peer_type/peer_target as flat strings):

```yaml
    ContactManagerCase:
      type: object
      properties:
        id:
          type: string
          format: uuid
          description: Unique identifier for the case.
          example: "550e8400-e29b-41d4-a716-446655440000"
        customer_id:
          type: string
          format: uuid
          description: Unique identifier of the associated customer.
          example: "7c4d2f3a-1b8e-4f5c-9a6d-3e2f1a0b4c5d"
        peer_type:
          type: string
          description: Remote endpoint type (e.g. "tel", "email") this case is scoped to.
          example: "tel"
        peer_target:
          type: string
          ...
```

After:

```yaml
    ContactManagerCase:
      type: object
      properties:
        id:
          type: string
          format: uuid
          description: Unique identifier for the case.
          example: "550e8400-e29b-41d4-a716-446655440000"
        customer_id:
          type: string
          format: uuid
          description: Unique identifier of the associated customer.
          example: "7c4d2f3a-1b8e-4f5c-9a6d-3e2f1a0b4c5d"
        peer:
          allOf:
            - $ref: '#/components/schemas/CommonAddress'
          description: Remote party this case is scoped to.
        local:
          allOf:
            - $ref: '#/components/schemas/CommonAddress'
          description: The customer's own endpoint (number/channel/account) this case's interactions arrived on or were placed from. Always present as an object; individual fields (type, target, etc.) are empty/absent when no local endpoint was known at case creation time (see design §4.1's note on Go's omitempty semantics for this field).
        # ... reference_type and every other existing field unchanged
```

### 7.2 `ContactManagerInteraction` (openapi.yaml:3665+)

Before (peer_type/peer_target/local_type/local_target as four flat strings,
openapi.yaml:3685-3700).

After: replace those four properties with two nested `CommonAddress` refs,
identical pattern to §7.1:

```yaml
        peer:
          allOf:
            - $ref: '#/components/schemas/CommonAddress'
          description: Remote endpoint (match key for read-time contact resolution).
        local:
          allOf:
            - $ref: '#/components/schemas/CommonAddress'
          description: The customer's own endpoint (attribution: which number/account received/sent). Always present as an object; individual fields are empty for historical rows where no local endpoint was captured (see design §4.1's note on Go's omitempty semantics for this field).
```

`allOf` + single `$ref` (rather than a bare `$ref:` on the property) matches
the existing convention this file already uses for Call's
source/destination fields -- allows attaching a per-usage `description`
alongside the shared schema without duplicating its property list. No
`additionalProperties: true` anywhere, per this repo's OpenAPI spec rules
(`bin-openapi-manager/CLAUDE.md` §"OpenAPI spec rules").

After editing `openapi.yaml`, regenerate in the correct order (documented in
`bin-api-manager/CLAUDE.md`'s "Code generation" section):

```bash
cd bin-openapi-manager && go generate ./...
cd ../bin-api-manager && go generate ./...
```

## 8. Downstream consumer impact

### 8.1 square-admin (monorepo-javascript) -- OUT OF SCOPE, follow-up ticket

Confirmed via grep against `monorepo-javascript/square-admin/src/views/contacts/`:

- `contact_cases_list.js:31-38` reads `row.original.peer_target` /
  `row.original.peer_type` directly as flat fields.
- `contact_cases_detail.js:41-43,600-614,716-717` reads `detailData.peer_type`
  / `detailData.peer_target` directly (both in the page header/subtitle
  logic and when passing props to `CaseInteractionHistoryPanel`), and
  separately builds a `peer_type`/`peer_target` query string for
  `GET /contact_interactions` -- this query-string filter usage is a
  request-body concern, not a response-shape concern, and is unaffected by
  this design (the RPC/route filter params are untouched; only the
  Interaction/Case JSON response shape changes).
- `contact_cases_unresolved.js:68-72` reads `kase.peer_target` /
  `kase.peer_type` directly.
- `contacts_detail.js:734-736` reads `kase.peer_type` / `kase.peer_target`
  directly (case list embedded in the Contact detail page).

All five sites need to change from `x.peer_type` / `x.peer_target` to
`x.peer.type` / `x.peer.target` once this backend change ships. Associated
Jest fixtures under each view's `__tests__/` directory (not individually
enumerated here) construct flat `peer_type`/`peer_target` mock case/
interaction objects and will need the same nested-shape update.

**This design does not include the square-admin code changes.** They are
explicitly out of scope for this backend design doc and must be tracked as
a separate follow-up PR in `monorepo-javascript`, coordinated to land no
later than this backend change's deploy (a hard requirement given there is
no compatibility shim -- square-admin's Case/Interaction views will render
`undefined` for peer/local fields between this backend deploying and the
frontend follow-up landing).

### 8.2 bin-flow-manager -- request-side signature changes per §6.3, no response-shape impact

`bin-flow-manager/pkg/activeflowhandler/actionhandle.go`'s
`actionHandleCaseCreate` (actionhandle.go:1302-1391) calls
`ContactV1CaseCreate` (actionhandle.go:1373) -- this changes to the
consolidated `peer commonaddress.Address` signature per §6.3. Separately
from that request-side edit (already covered in full in §6.3), this
function never reads back `PeerType`/`PeerTarget`/`LocalType`/`LocalTarget`
off the returned `*kase.Case` (`res`) -- only `res.ID` is used
(actionhandle.go:1382,1386), so no *response*-shape code change is needed
here beyond §6.3's request-side edit. Its own locally-duplicated
`crmIneligiblePeerTypes` map (actionhandle.go:1251-1274) operates on
`commonaddress.Type` values from call/message webhook payloads, unrelated
to Case/Interaction's stored shape, also unaffected.

### 8.3 bin-ai-manager -- request-side signature change per §6.3 (tool.go); tool_insight.go response reads MUST change (§6.5)

`bin-ai-manager/pkg/aicallhandler/tool.go`'s `toolHandleCaseCreate`
(tool.go:519-568) mirrors bin-flow-manager's pattern and gets the identical
request-side edit described in §6.3 (`ContactV1CaseCreate`'s consolidated
`peer` argument). It does not read `PeerType`/`PeerTarget` off the
response, so no response-side change is needed there.

Separately, `bin-ai-manager/pkg/aicallhandler/tool_insight.go`'s
`toolHandleGetContactInteractions` DOES read a Case's/Interaction's peer
fields directly off a fetched `*cmkase.Case`/`*cminteraction.Interaction`:

```go
// tool_insight.go:119-121
interactions, _, err = h.reqHandler.ContactV1InteractionList(
    ctx, c.CustomerID, limit, "", string(kase.PeerType), kase.PeerTarget, uuid.Nil, uuid.Nil, time.Time{})
```

and separately formats `*cminteraction.Interaction`'s peer fields into an
LLM-facing summary string:

```go
// tool_insight.go:150-153
lines = append(lines, fmt.Sprintf(
    "[%s] direction=%s peer=%s/%s reference_type=%s reference_id=%s",
    ts, it.Direction, it.PeerType, it.PeerTarget, it.ReferenceType, it.ReferenceID,
))
```

**Since `PeerType`/`PeerTarget` no longer exist as Go fields at all (§4),
both of these DO require a code change** -- not a compile-and-work-as-is
situation. `string(kase.PeerType), kase.PeerTarget` becomes
`string(kase.Peer.Type), kase.Peer.Target`; `it.PeerType, it.PeerTarget`
becomes `it.Peer.Type, it.Peer.Target`. Full detail (including the
corresponding `tool_insight_test.go` literal updates) is in §6.5, which is
the authoritative call-site list for this change; this subsection exists
only to flag that `bin-ai-manager` specifically needs edits beyond its
`case_create` request-side signature change.

## 9. Testing plan

1. **dbhandler INSERT/generated-column tests** (new, in
   `bin-contact-manager/pkg/dbhandler/kase_test.go` and `interaction_test.go`):
   - `CaseInsert`/`CaseInsertTx` with a populated `Peer`/`Local`: assert the
     row round-trips (`CaseGetByID`) with `Peer`/`Local` equal to the
     inserted values, and separately assert the `peer_type`/`peer_target`/
     `local_type`/`local_target` MySQL columns hold the expected generated
     values via a direct raw-SQL query in the test (not through
     `kase.Case`, since no Go field maps to them per §4/§5 -- e.g.
     `db.Query("SELECT peer_type, peer_target FROM contact_cases WHERE
     id = ?", ...)`).
   - `CaseInsert` with a zero-value `Local` (no local endpoint known):
     per §4.1's storage-asymmetry note, a zero `Local` written via Go
     code serializes to JSON `'{}'` (never SQL NULL -- only historical
     migrated rows get SQL NULL, §3.2), so assert the row's `local`
     column holds `'{}'` and its `local_type`/`local_target` generated
     columns are SQL NULL (verified via the same raw-SQL query pattern;
     `JSON_EXTRACT('{}', '$.type')` evaluates to NULL) and no error/panic
     occurs on this path.
   - Same two cases for `InteractionCreate`.
   - `uq_case_open_peer`/`idx_contact_interactions_idem` duplicate-detection
     tests (already exist) re-run unchanged and must still pass -- proves
     the generated columns preserve the exact uniqueness behavior of the
     old plain columns.
2. **JSON roundtrip test**: insert a `Peer`/`Local` with all five
   `commonaddress.Address` fields populated (`Type`, `Target`, `TargetName`,
   `Name`, `Detail`), read it back via `CaseGetByID`/`InteractionGet`,
   assert full equality on the `Peer`/`Local` struct -- proves the `,json`
   db tag path preserves the metadata this design newly captures (not just
   `Type`/`Target`).
3. **casehandler tests** (`create_test.go`, `getorcreate_test.go`,
   `lifecycle_continue_test.go`): assert `Create`/`GetOrCreate`/`Continue`
   construct `kase.Case{Peer: ..., Local: self}` correctly (§6.1/§6.2/
   §6.5's `Continue` note: `Continue` always passes a zero `self`, by
   design, since it has no address parameter of its own), including the
   zero-`self` case (`Local` stays the zero Address, still valid to
   persist).
3a. **Mock regeneration**: `CaseHandler.Create`/`GetOrCreate`'s signature
   changes (§6.0-§6.2, two/one separate scalar parameters collapse into a
   single `peer commonaddress.Address`) require regenerating the gomock
   mocks BEFORE running any test that depends on them, or every caller of
   `MockCaseHandler.EXPECT().Create(...)`/`.GetOrCreate(...)` fails to
   compile against the old mock signature:
   ```bash
   cd bin-contact-manager
   go generate ./pkg/casehandler/...
   go generate ./pkg/dbhandler/...
   ```
   `dbhandler`'s mock is included even though no `dbhandler`-level
   interface signature changes in this design (§5 -- `CaseGetOpenByPeer`/
   `CaseGetLastClosedByPeerTx` keep their existing `peerType, peerTarget`
   scalar parameters, §5.1) purely as a defensive regeneration step,
   consistent with this repo's standing convention of regenerating every
   touched package's mocks as part of the verification workflow, not
   because this design specifically changes a `dbhandler`-interface
   signature.
3b. **Empty-peer validation test (round-2 design review finding, §6.1/
   §6.2)**: assert `Create(ctx, customerID, self, commonaddress.Address{},
   referenceType, name, detail)` and `GetOrCreate(ctx, customerID, self,
   commonaddress.Address{}, referenceType, caseIDHint)` both return the
   typed `CASE_PEER_REQUIRED` error and perform NO database write (no
   `insertWithRetry`/DB call reached) -- this is the primary regression
   test for the finding that a zero-value `peer` would otherwise reach
   the generated-column NOT NULL constraint and fail with an opaque MySQL
   driver error instead of a clear validation error. Also assert the
   partial cases (`peer.Type` set but `peer.Target` empty, and vice
   versa) are equally rejected, since the guard checks both independently.
4. **contacthandler tests** (`interaction_test.go`): assert
   `EventCallCreated`/`EventConversationMessageCreated` build
   `interaction.Interaction{Peer: ..., Local: ...}` with `Target` equal to
   the normalized value (not the raw `source`/`destination` value) --
   regression-guards §6.4's bit-identical-to-contact_addresses.target
   invariant.
5. **§6.5 call-site update tests**: `case_message_test.go`'s existing table
   tests (all updated to construct `Peer: commonaddress.Address{...}`
   instead of flat `PeerType`/`PeerTarget`, per §6.5) must still pass
   unmodified in their assertions -- this is a pure literal-shape change,
   not a behavior change, so no new test logic is needed here beyond
   updating the fixtures themselves. Same for
   `bin-ai-manager/pkg/aicallhandler/tool_insight_test.go`.
6. **Migration test** (bin-dbscheme-manager, local throwaway DB only per
   this repo's Alembic rules -- never against staging/production): seed
   `contact_interactions` rows pre-migration with the old plain
   `peer_type`/`peer_target`/`local_type`/`local_target` columns (including
   at least one row with `local_type = '' AND local_target = ''`), run
   `alembic upgrade head` locally, assert:
   - `peer`/`local` JSON columns are correctly backfilled from the old
     column values.
   - `peer_type`/`peer_target` generated columns still equal their
     pre-migration values exactly (bit-identical backfill, §3.2).
   - The `local_type = '' AND local_target = ''` row now reads back
     `local_type IS NULL` / `local_target IS NULL` (the documented '' ->
     NULL semantic shift, §3.2) -- not `''`.
   - `idx_contact_interactions_idem`'s uniqueness still rejects a
     duplicate-key re-insert after migration -- **this alone is
     insufficient** (round-8 design review finding): a same-key duplicate
     `(reference_type, reference_id, peer_target)` is also rejected by
     the narrower, accidentally-shrunk `(reference_type, reference_id)`
     index that §3.2's fix specifically prevents, so this assertion by
     itself would pass even if the index-shrinkage bug were still
     present and silently masked. Add the discriminating assertion: two
     rows sharing the same `(reference_type, reference_id)` but
     DIFFERENT `peer_target` must both insert successfully (proving the
     live index is the full 3-column
     `(reference_type, reference_id, peer_target)`, not the narrower
     2-column one). Additionally assert directly against
     `information_schema.statistics` that `idx_contact_interactions_idem`
     covers exactly `(reference_type, reference_id, peer_target)` and
     `idx_contact_interactions_peer` covers exactly
     `(customer_id, peer_type, peer_target)` post-migration -- the
     column-list assertion is the only test that would have caught
     round 8's finding directly, rather than relying on inferring it from
     insert/reject behavior.
7. Full verification workflow in `bin-contact-manager` (`go mod tidy && go
   mod vendor && go generate ./... && go test ./... && golangci-lint run -v
   --timeout 5m`), and a `go build ./...` sanity pass in `bin-api-manager`,
   `bin-flow-manager`, `bin-ai-manager` to confirm no other call site was
   missed beyond §6.5/§8's audit -- this build pass is the primary
   safety net for §6.5's grep-derived call-site list: since
   `PeerType`/`PeerTarget`/`LocalType`/`LocalTarget` no longer exist as Go
   fields at all, ANY missed call site fails to compile (not a silent
   runtime bug), so a clean `go build ./...` across all four services is
   sufficient proof the audit was complete.

## 10. Open questions / explicitly out of scope

- **`call_calls.source`/`destination` and `conversation_messages.source`/
  `destination` are NOT touched by this design.** They already use the
  plain-JSON-column pattern (no generated peer_type/peer_target columns at
  all -- Call/Message never exposed flat peer_type/peer_target columns to
  begin with, only the JSON `Source`/`Destination` fields) with their own
  existing manual dual-write conventions elsewhere in those services. This
  design's generated-column mechanism is specific to Case/Interaction
  because those two entities are the only ones that historically had BOTH a
  JSON-shaped concept (peer/local) AND separately-indexed plain
  peer_type/peer_target columns that needed to keep working unchanged. A
  future design could evaluate unifying Call/Message onto the same
  generated-column pattern for consistency, but that is explicitly deferred
  -- no such unification is proposed or implied here.
- **square-admin frontend changes are tracked as a separate follow-up**
  (§8.1) -- this design's backend change and that follow-up must be
  coordinated to land together at deploy time given there is no
  compatibility shim.
- **No `WebhookMessage`/`ConvertWebhookMessage` layer is being introduced**
  for Case or Interaction as part of this design, consistent with
  `bin-api-manager/pkg/servicehandler/case.go`'s existing documented
  rationale (§7) -- out of scope; would be a separate, larger design if
  ever pursued.
- **`Case.Local`'s nullability contract**: per §4.1's storage-asymmetry
  note, "no local endpoint known" is represented as SQL NULL only for
  historical migrated rows (§3.2) or any future direct-SQL write; any row
  written through Go code (including every new Case/Interaction created
  after this design ships) represents it as JSON `'{}'` instead, since Go
  always serializes a zero `commonaddress.Address` value to an empty
  object, never omits or nulls it (§4.1). Both forms are functionally
  equivalent for every purpose this design cares about (`local_type`/
  `local_target` generated columns evaluate to NULL either way). Whether a
  future UI/API consumer needs to distinguish "local genuinely unknown"
  from "local known to be empty" is not addressed here -- `commonaddress.
  Address` has no such distinction today (a zero `Type`/`Target` already
  means "no address"), so this design does not introduce one.
- **`Case.Continue`'s re-opened Case never carries a fresh `Local`** (§6.5):
  `Continue`'s signature has no address parameter to source a `self` value
  from, so a Case re-opened via `Continue` always gets `Local:
  commonaddress.Address{}` regardless of what number/channel the
  triggering re-contact actually arrived on. This mirrors the pre-existing
  gap (before this design, NO code path ever populated Local on a
  Continue-created Case, since Local didn't exist at all), so it is not a
  regression, but it also means this design's Local-capture fix (§1) is
  incomplete for the re-contact path specifically. A future design could
  extend `Continue`'s signature to accept a fresh `self commonaddress.
  Address` (the caller -- `bin-api-manager`'s `CaseContinue` servicehandler
  -- would need its own source of that value, likely from whatever
  triggered the continue request), but this is a new capability requiring
  its own design discussion about where that value comes from, not
  something implied by this design's Peer/Local storage-format change.
  Left explicitly out of scope.

## Iter-1 review response summary

Round-1 independent design review (`delegate_task`, `toolsets=["file"]`,
2026-07-22) returned `VERDICT: CHANGES_REQUESTED` with 5 actionable items.
All 5 addressed below; the reviewer confirmed ~25 spot-checked file:line
citations in the prior draft were all accurate, and no CRITICAL findings
were raised beyond the two listed as items 1-2/3 (both real correctness
bugs, not style).

1. **Go `omitempty` bug on `Peer`/`Local commonaddress.Address` fields**
   (was §4.1/§4.2, contradicted §7.1/§7.2's "Absent if..." wording) --
   FIXED. `omitempty` removed from both `Local` field tags (it never
   worked on non-pointer struct fields); §4.1 gained a new explanatory
   note (Go semantics, the rejected pointer alternative and why, and the
   resulting storage-representation asymmetry between SQL-NULL and
   JSON-`{}` empty-local rows); §4.2 cross-references §4.1's note instead
   of duplicating it; §7.1/§7.2's OpenAPI descriptions corrected from
   "Absent if..." to "Always present as an object; individual fields
   are empty/absent when...".
2. **`contact_cases.peer JSON NOT NULL` with no DEFAULT, added directly to
   a populated table** (§3.1) -- FIXED. Changed to ADD-nullable ->
   backfill (`UPDATE ... WHERE peer IS NULL`) -> `MODIFY COLUMN peer JSON
   NOT NULL`, with an inline comment explaining the MySQL strict-mode
   failure mode (errno 1364) the old ordering would have hit.
3. **Same NOT-NULL-with-no-DEFAULT ordering bug on
   `contact_interactions.peer`** (§3.2) -- FIXED identically to item 2.
4. **§6.5's call-site audit claimed to be a "full list"/"authoritative"
   but omitted several test-fixture files** -- FIXED. §6.5 now separately
   enumerates (a) test-fixture construction sites found by re-grepping the
   repo (`kase_test.go`, `interaction_test.go`,
   `interaction_read_test.go`, `address_ownership_read_test.go`,
   `kase_list_test.go` in both `models/kase/` and `pkg/dbhandler/`,
   `casehandler/create_test.go`'s old five-argument `h.Create(...)` calls)
   and (b) the previously-listed assertion-only sites, and explicitly
   states this list is a best-effort starting point rather than an
   infallible guarantee, deferring final completeness verification to
   §9 point 7's `go build ./...` sanity pass (which the design already
   argued is a hard compile-failure backstop, not a silent-bug risk, given
   §4's "no derived-field shortcut" decision).
5. **No explicit mock-regeneration step for `CaseHandler.Create`/
   `GetOrCreate`'s changed signatures** -- FIXED. Added §9 item 3a
   specifying `go generate ./pkg/casehandler/...` and
   `./pkg/dbhandler/...` as an explicit pre-test step.

## Iter-2 review response summary

Round-2 independent design review (`delegate_task`, `toolsets=["file"]`,
2026-07-22) returned `VERDICT: CHANGES_REQUESTED` with 3 actionable items.
The reviewer independently re-verified all 5 of round-1's fixes by direct
code read (not by trusting the Iter-1 summary) and confirmed items 1/2/3/5
were correctly applied, including independently confirming the
`bin-common-handler/pkg/databasehandler/mapping.go` nil-pointer-check
factual claim underpinning item 1's rejected-pointer-alternative rationale
was accurate. Item 4 (the §6.5 call-site audit) was found to still be
materially incomplete despite round 1's claim of having "re-verified by
direct grep" -- and round 2 additionally surfaced a genuine new regression
(item 3 below) that neither the original design nor round 1 had caught.
All 3 addressed below.

1. **§6.5's "authoritative"/"re-verified" test-fixture list was still
   incomplete** -- round 1's fix (item 4 in the Iter-1 summary above)
   added several files but round 2's independent re-grep found ~14
   additional files still missing (`contact_update_test.go`,
   `assign_test.go`, `getorcreate_test.go`, `getorcreate_timeout_test.go`,
   `getorcreate_fresh_test.go`, `getorcreate_proactive_link_test.go`,
   `lifecycle_close_test.go`, `unresolved_queue_test.go`,
   `case_list_get_test.go`, `case_tag_test.go`, `getorcreate_race_test.go`,
   `casenote_isolation_test.go`, `casenote_test.go`,
   `pkg/dbhandler/resolution_test.go`) -- FIXED, this time by abandoning
   line-by-line enumeration entirely in favor of an exhaustive grep-count
   table (169 occurrences across 23 files, §6.5) plus a single worked
   example showing the mechanical transformation pattern that applies
   uniformly to all of them. This format is inherently harder to
   under-report than a hand-curated list, since it is a direct `grep`
   count rather than a manually assembled enumeration.
2. **§6.5/§6.2 never audited old-signature `GetOrCreate(...)` call
   sites at all** -- only `Create(...)`'s old-signature calls were
   mentioned (`create_test.go`). FIXED: §6.5 now includes a
   `grep -rlE "\.GetOrCreate\(ctx" bin-contact-manager/pkg/casehandler`
   count (8 files, 18 call sites) alongside the literal-construction
   count, with the same worked-example treatment.
3. **New regression found: a zero-value `peer commonaddress.Address{}`
   passed to `Create`/`GetOrCreate` fails the INSERT outright** (not
   found in round 1) -- because `commonaddress.Address`'s own `Type`/
   `Target` fields carry `omitempty` (unlike the old flat `PeerType`/
   `PeerTarget string` fields, which had none and so wrote real
   empty-string values that satisfied the old `NOT NULL DEFAULT ''`
   plain columns), a zero-value peer now marshals to JSON `{}`, and
   `JSON_EXTRACT('{}', '$.type')` evaluates to SQL NULL, which violates
   the generated `peer_type` column's `NOT NULL` constraint -- an opaque
   MySQL driver error, not a clear validation failure, and previously
   unguarded and untested. FIXED: added an explicit empty-peer guard
   (returning a typed `CASE_PEER_REQUIRED` error via
   `cerrors.InvalidArgument`) at the top of both `Create` (§6.1) and
   `GetOrCreate` (§6.2), converting §4.1's previously-unenforced "every
   Case has a peer by construction" assumption into an actual enforced
   invariant; added §9 item 3b as the corresponding regression test.

## Iter-3 review response summary

Round-3 independent design review (`delegate_task`, `toolsets=["file"]`,
2026-07-22) returned `VERDICT: CHANGES_REQUESTED` with 1 actionable item.
The reviewer independently re-verified ~50+ file:line citations and API
signatures from rounds 1-2's fixes (all confirmed exact matches,
including `cerrors.InvalidArgument`'s real signature and
`commonoutline.ServiceNameContactManager`'s existence), independently
re-ran both of §6.5's grep-count commands and got identical numbers
(169/23 and 8/18, confirming those counts were accurate as far as they
went), and confirmed the new empty-peer guard breaks no existing test.
The single finding was a scope gap in the grep itself, not a counting
error.

1. **§6.5's grep-count table was scoped to `bin-contact-manager` only,
   missing `bin-api-manager/pkg/servicehandler/interaction_test.go`**
   (lines 167-168, 174-175, 194-195: `cminteraction.Interaction{PeerType:
   "tel", PeerTarget: "+155****1111", ...}` literals) -- a sibling file
   in the same directory as `case_message_test.go` (already covered),
   which the design had simply never grepped because it lives outside
   `bin-contact-manager`. FIXED by widening the grep to the entire
   monorepo (not scoped to any single service, since `kase.Case`/
   `interaction.Interaction` are types other services legitimately
   import and construct literals of): the corrected table now shows 202
   occurrences across 26 files (up from 169/23), explicitly including
   `bin-ai-manager/pkg/aicallhandler/tool_insight_test.go` and
   `bin-api-manager/pkg/servicehandler/case_message_test.go` (both
   already covered elsewhere in the design, folded into this table for a
   single source of truth) alongside the newly-found
   `interaction_test.go`. Also documented one false-positive found during
   the re-grep (`bin-conversation-manager`'s unrelated
   `Conversation`-model flow-variable constants) and confirmed
   `bin-flow-manager` has zero matches, so no further services are
   missing from this audit.

## Iter-4 review response summary

Round-4 independent design review (`delegate_task`, `toolsets=["file"]`,
2026-07-22) returned `VERDICT: CHANGES_REQUESTED` with 1 minor actionable
item. The reviewer independently re-ran the grep-count table and
confirmed its per-file rows accurate (27 raw matching files including the
documented false positive, minus that 1 false-positive file = 26 net
files; round 6 subsequently caught that the TOTAL figure quoted at the
time, 202, was an arithmetic slip that didn't subtract the false
positive's 2 occurrences from the raw 202 -- corrected to 200 in the
Iter-5 summary below and in §6.5's table itself), independently verified
`bin-api-manager/pkg/servicehandler/interaction_test.go`'s cited line
numbers/content, spot-checked ~20 other file:line citations across
every major section (all matched), and confirmed only two production RPC
callers of `ContactV1CaseCreate` exist (`bin-flow-manager`,
`bin-ai-manager`), matching §6.0/§8.2/§8.3's claims.

1. **The test-file false-positive exclusion for `bin-conversation-
   manager`'s `Conversation.Peer` model never checked whether the
   analogous PRODUCTION (non-test) file was also addressed** -- round 3's
   fix widened the grep to `*_test.go` monorepo-wide but never ran the
   equivalent non-test grep, leaving `bin-conversation-manager/pkg/
   conversationhandler/variable.go:43,45` (the non-test source of the
   same `variableConversationPeerTarget`/`variableConversationPeerType`
   constants) unaddressed by name, even though it is almost certainly the
   same benign false positive. FIXED: §6.5 now explicitly documents this
   production file, confirms by direct read that `cv` is
   `conversation.Conversation` (a wholly separate model, already using
   the same Address-as-JSON pattern this design extends to Case/
   Interaction), and states the non-test monorepo-wide grep (`--include
   "*.go" | grep -v _test.go`) found no other production file matching
   the pattern.

## Iter-5 review response summary

Round-5 independent design review (`delegate_task`, `toolsets=["file"]`,
2026-07-22) returned `VERDICT: APPROVED` with no findings -- the first of
the two consecutive APPROVED verdicts required to close the loop. Round-6
independent design review (`delegate_task`, `toolsets=["file"]`,
2026-07-22), run as the required second consecutive verdict, returned
`VERDICT: CHANGES_REQUESTED` with 1 actionable item, resetting the
consecutive-APPROVED counter to zero.

1. **§6.5's grep-count table total was arithmetically wrong** -- the
   table listed 26 rows (the false-positive file already excluded from
   the row list) but labeled the total "202 occurrences," which is the
   RAW count across all 27 matching files INCLUDING the 2 false-positive
   occurrences, not the correct net sum of the 26 listed rows. Summing
   the 26 rows actually printed in the table gives 200, not 202. This
   error was introduced when round 3 widened the grep scope and
   propagated uncorrected through round 4's fix and round 5's approval
   (round 5 apparently did not independently re-sum the table's own
   rows, only spot-checked selected rows against the repo, so the
   header/prose total went unverified against the table's own body).
   FIXED: §6.5's table total corrected to "200 occurrences across 26
   files" (independently re-verified via
   `grep -rlE "PeerType:|PeerTarget:|LocalType:|LocalTarget:" .
   --include="*_test.go"` from the monorepo root: 202 raw across 27
   files, minus the documented `bin-conversation-manager` false
   positive's 2 occurrences = 200 net across 26 files); the Iter-4
   summary's matching "202"/"204" figures corrected in place (see the
   revised Iter-4 summary above) rather than left standing alongside the
   corrected table, to avoid the exact kind of self-contradiction this
   finding was about.

Per the loop's termination rule, this CHANGES_REQUESTED verdict means
round 5's APPROVED does not count toward the 2-consecutive requirement --
a fresh round 7 (and, if APPROVED, a fresh round 8) is required to close
the loop.

## Iter-6 review response summary

Round-7 independent design review (`delegate_task`, `toolsets=["file"]`,
2026-07-22) returned `VERDICT: CHANGES_REQUESTED` with 2 actionable items,
resetting the consecutive-APPROVED counter to zero again. The reviewer
notably went further than any prior round by spinning up a real MySQL
8.0.46 container and executing the design's exact migration DDL against
it, rather than reading the SQL text alone.

1. **§3.1/§3.2's generated-column DDL is invalid MySQL syntax** -- both
   migrations wrote `peer_type VARCHAR(255) NOT NULL GENERATED ALWAYS AS
   (...) STORED` (and identically for `peer_target`), placing `NOT NULL`
   BEFORE the `GENERATED ALWAYS AS (...) STORED` clause. MySQL requires
   the reverse order (`GENERATED ALWAYS AS (...) STORED NOT NULL`) and
   rejects the design's original ordering outright with a real syntax
   error (errno 1064), confirmed by the reviewer executing both the
   broken and fixed versions against an actual MySQL 8.0.46 instance.
   This is a genuine, previously-uncaught defect that none of rounds 1-6
   found because no round had executed the DDL against a real database
   before this one. FIXED: both migrations' `peer_type`/`peer_target`
   generated-column definitions reordered to `GENERATED ALWAYS AS (...)
   STORED NOT NULL` in §3.1 and §3.2 (the nullable `local_type`/
   `local_target` columns were never affected, since they carry no `NOT
   NULL` clause at all and so never hit this ordering constraint).
2. **§6.5's own prose (not just the appendices) still referenced the
   stale pre-round-3 "169" total in two places** (the paragraph
   following the corrected 200/26 table, and the worked-example
   introduction) -- a live self-contradiction within the still-active
   numbered section, distinct from the historical appendix text. FIXED:
   both "169" references corrected to "200".

## Iter-7 review response summary

Round-8 independent design review (`delegate_task`, `toolsets=["file"]`,
2026-07-22) returned `VERDICT: CHANGES_REQUESTED` with 3 actionable items,
resetting the consecutive-APPROVED counter to zero again. This round went
further still than round 7: it built `contact_cases`/`contact_interactions`
schemas EXACTLY matching the real Alembic-created tables (including
`open_peer_uk` and both `contact_interactions` indexes, read directly from
`f718e26f2c44`/`ac5d4e18060c`) and executed the FULL §3.1/§3.2 migration
sequences end-to-end against a real MySQL 8.0.46 instance, not just the
single statement round 7 had isolated.

1. **§3.1: `DROP COLUMN peer_type, DROP COLUMN peer_target` fails outright
   (errno 3108) against the real `contact_cases` schema** -- `open_peer_uk`
   (`f718e26f2c44`, already quoted in §2.2) is itself a STORED generated
   column whose expression directly references `peer_type`/`peer_target`
   (`UNHEX(SHA2(CONCAT_WS('|', customer_id, peer_type, peer_target,
   reference_type), 256))`), and MySQL refuses to drop a column another
   generated column depends on. Round 7's narrower single-statement test
   used a minimal table without `open_peer_uk` and never exercised this
   path -- **the migration as originally written could not execute
   against the real table at all.** FIXED: §3.1 now explicitly
   `DROP INDEX uq_case_open_peer, DROP COLUMN open_peer_uk` BEFORE
   dropping `peer_type`/`peer_target`, and re-adds `open_peer_uk`/
   `uq_case_open_peer` (expression copied verbatim from `f718e26f2c44`)
   AFTER the new generated `peer_type`/`peer_target` columns exist.
2. **§3.2: dropping `peer_target`/`peer_type` does NOT error, but
   silently SHRINKS `idx_contact_interactions_idem` from
   `(reference_type, reference_id, peer_target)` down to
   `(reference_type, reference_id)`, and `idx_contact_interactions_peer`
   from `(customer_id, peer_type, peer_target)` down to `(customer_id)`
   alone** -- confirmed via `information_schema.statistics` before/after.
   Re-adding `peer_type`/`peer_target` as generated columns afterward does
   NOT widen either index back out; MySQL never re-attaches a
   previously-shrunk index to a same-named column that reappears later.
   This directly contradicted §3.2's own prior claim that these indexes
   were "untouched," and would have silently weakened the idempotency
   uniqueness guarantee the whole design depends on (two different peers
   sharing a `(reference_type, reference_id)` would incorrectly collide as
   duplicates). FIXED: §3.2 now explicitly `DROP INDEX`s both indexes and
   re-`ADD`s them at their full original column lists after the generated
   columns are recreated.
3. **§9's migration test (asserting `idx_contact_interactions_idem` still
   rejects a duplicate-key re-insert) would NOT have caught finding 2** --
   a same-key duplicate is rejected by the narrower, accidentally-shrunk
   2-column index just as well as by the correct 3-column one, so the
   existing assertion alone would pass even with the bug present. FIXED:
   §9 now additionally requires (a) asserting two rows sharing
   `(reference_type, reference_id)` but different `peer_target` both
   insert successfully, and (b) directly asserting the post-migration
   column lists of both indexes via `information_schema.statistics` --
   the only assertion that would have caught this finding directly rather
   than by inference.

## Iter-8 review response summary

Round-9 independent design review (`delegate_task`, `toolsets=["file"]`,
2026-07-22) returned `VERDICT: CHANGES_REQUESTED` with 1 actionable item,
resetting the consecutive-APPROVED counter to zero yet again. This round
seeded PRE-migration data (two rows sharing `(reference_type,
reference_id)` with different `peer_target` -- the exact fan-out scenario
`idx_contact_interactions_idem`'s 3-column shape exists to permit, per
`ac5d4e18060c`'s own comment) before running the corrected §3.1/§3.2
sequences end-to-end, which round 8 had not done (round 8 only tested
POST-migration inserts). §3.1 passed cleanly against an equivalent
pre-existing-data scenario. §3.2 did not.

1. **§3.2's index-drop and column-drop statements were in the wrong
   relative order** -- round 8's fix added explicit `DROP INDEX
   idx_contact_interactions_idem, DROP INDEX
   idx_contact_interactions_peer` statements, but placed them AFTER the
   `DROP COLUMN peer_type, peer_target, local_type, local_target`
   statement, whereas §3.1's already-correct pattern for
   `open_peer_uk`/`uq_case_open_peer` does the index drop FIRST. With the
   indexes still live while `peer_target` is dropped, MySQL's mid-ALTER
   implicit index-shrink to `(reference_type, reference_id)` collides
   with any pre-existing rows that legitimately share that narrower key
   -- exactly the fan-out data this index exists to support in
   production -- and the `DROP COLUMN` statement itself fails outright
   with errno 1062 (duplicate entry), confirmed by executing this exact
   sequence against a real MySQL 8.0.46 instance seeded with such rows.
   Round 8's test never seeded pre-existing duplicate-reference rows
   before migrating, so it only ever observed the (also real, but
   less severe) silent-shrinkage failure mode, not this hard failure
   mode that any real deployment with existing fan-out data would hit
   immediately. FIXED: §3.2 reordered to `DROP INDEX
   idx_contact_interactions_idem, DROP INDEX idx_contact_interactions_peer`
   BEFORE the `DROP COLUMN` statement (mirroring §3.1's pattern exactly),
   with the re-`ADD INDEX`/`ADD UNIQUE INDEX` statements unchanged,
   still positioned after the generated columns are recreated.

