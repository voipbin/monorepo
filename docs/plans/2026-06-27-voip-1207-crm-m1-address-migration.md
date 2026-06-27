# VOIP-1207 CRM v1 M1: address migration (phone/email -> contact_addresses)

- Ticket: VOIP-1207 (implementation step 2 of VOIP-1204)
- Depends on: VOIP-1206 (dbscheme, `contact_addresses` shipped, merged `a89cc3ca4`)
- Parent design: `docs/plans/2026-06-26-add-contact-crm-interaction-timeline-design.md` §7.1
- Services touched: `bin-contact-manager` (core), `bin-common-handler` (shared RPC
  client signature), `bin-api-manager` (REST->RPC), `bin-openapi-manager` +
  `bin-api-manager` gens (OpenAPI spec + regen). Big-bang cutover.

## 1. Goal

Retire `contact_phone_numbers` + `contact_emails` as the source of truth and make
`contact_addresses` the single store for a contact's identifiers. The inbound
caller-ID lookup hot path must read from `contact_addresses`. The `phone_numbers[]`
/ `emails[]` shape on the contact object is preserved via reverse-projection, with
ONE intentional breaking change: the `number_e164` field is removed everywhere
(§3.1, §7), since VoIPBin is currently the only API consumer.

This is the highest operational-risk step in the CRM build: caller-ID lookup is
on the inbound-call hot path. Sequencing is **data migration -> verify -> code
deploy** in one window.

## 2. Current state (verified against code, 2026-06-27)

### 2.1 Storage today

- `contact_phone_numbers`: `id, customer_id, contact_id, number, number_e164, type, is_primary, tm_create`
- `contact_emails`: `id, customer_id, contact_id, address, type, is_primary, tm_create`
- Both are already-normalized child tables (NOT JSON embeds). No JSON work.
- Scale (production `bin_manager`, per §7.1): phone 781, email 1022, distinct
  e164 86, orphan rows 0. Under 2k total -> single transactional `INSERT` from a
  unified CTE (see §4),
  no chunking, no downtime.

### 2.2 Target table (shipped by VOIP-1206)

`contact_addresses`:
```
id BINARY(16) PK, customer_id BINARY(16), contact_id BINARY(16) NULL,
type VARCHAR(255), target VARCHAR(255), target_name VARCHAR(255),
is_primary TINYINT(1),
primary_contact_uk BINARY(16) GENERATED (IF(is_primary=1, contact_id, NULL)) STORED,
tm_create DATETIME(6), tm_update DATETIME(6)
UNIQUE (customer_id, type, target)
UNIQUE (customer_id, primary_contact_uk)
CHECK  (is_primary=0 OR contact_id IS NOT NULL)
```
Hard-delete (no `tm_delete`), mirrors `agent_addresses`.

### 2.3 Caller-ID hot path (code reality vs design/CLAUDE.md drift)

The dbhandler methods that must move:

- `ContactLookupByPhone(customerID, phoneE164)` -> `SELECT contact_id FROM contact_phone_numbers WHERE customer_id=? AND number_e164=?`, then `ContactGet`.
- `ContactLookupByEmail(customerID, email)` -> same against `contact_emails.address`.
- `PhoneNumberListByContactID` / `EmailListByContactID` -> reverse-projection feeders. Called from THREE sites: `ContactGet` (contact.go:152-153), `contactUpdateToCache` (74-75), and `ContactList` (234-235). All populate `PhoneNumbers[]` / `Emails[]`.
- CRUD: `PhoneNumber{Create,Update,Delete,ResetPrimary}`, `Email{...}` are called from contacthandler. `PhoneNumberGet` / `EmailGet` are in the interface (dbhandler/main.go:30-44) but currently called only from dbhandler tests, NOT from contacthandler; they still move to the unified table since the interface is kept.

**DRIFT FOUND (must correct, do NOT implement as written):** §7.1 and
`bin-contact-manager/CLAUDE.md` both say the caller-ID lookup uses a Redis cache
keyed `(customer_id, phone_e164)` / `(customer_id, email)` that must be rewritten
to `(customer_id, type, target)`. **No such cache exists in code.**
`ContactLookupByPhone/ByEmail` query the DB directly every call; `cachehandler`
only has `ContactGet`/`ContactSet` (the contact body cache keyed by contact id).
There is no per-identifier lookup cache to rewrite. So:
- The lookup rewrite is FROM/WHERE only (point the query at `contact_addresses`).
- The contact-body cache (`ContactGet`/`ContactSet`) is unaffected; it caches the
  assembled contact, which keeps working once the feeders read `contact_addresses`.
- `bin-contact-manager/CLAUDE.md` "Lookup cache" bullet is corrected to match.

## 3. The two data-model collisions (DECISIONS NEEDED / recorded)

`contact_addresses` is narrower than the legacy tables. Two legacy fields have no
home. Both are customer-visible via REST, so this is a behavior change, recorded
explicitly rather than silently dropped.

### 3.1 Raw `number` is dropped; `Number` field is repurposed to hold the E.164 value

`PhoneNumber` today carries both `Number` (raw, as typed: `(555) 123-4567`) and
`NumberE164` (normalized). `contact_addresses` stores only `target` (normalized).
Decision (pchero, confirmed): **keep the normalized value, drop the raw value, and
keep the single field name `number` carrying the E.164 string.** Concretely:

- The raw-input semantics of `Number` are dropped (the original `(555) 123-4567`
  form is not preserved anywhere).
- The `NumberE164` field is removed from the model and the REST payload.
- The surviving field is named `number` (db tag `number`, JSON `number`) and its
  VALUE is the normalized E.164 string. So normalization is preserved; only the
  column/field naming collapses from two fields to one.

- REST impact (read side): `phone_numbers[]` objects lose the `number_e164` key
  entirely; `phone_numbers[].number` now returns the E.164 string (e.g.
  `+155****4567`). Any client reading `number_e164` must read `number` instead.
- REST impact (write side): create/update no longer accept a `number_e164` input
  field. Clients send `number`, which is normalized to E.164 server-side. A client
  that previously supplied only `number_e164` will have that input dropped.
- Webhook impact: `contact_created/updated/deleted` payloads embed the same
  `PhoneNumber` struct (`models/contact/webhook.go:27`), so `number_e164`
  disappears from event payloads too.
- These are breaking shape changes (read, write, webhook), accepted per pchero
  decision (VoIPBin is the only API consumer today). Full propagation in §7.
- Email has no raw/normalized split (`Address` only), so email is unaffected.
- Reverse-projection (§7.1) maps `contact_addresses.target` -> `PhoneNumber.Number`.

### 3.2 Sub-type (`mobile`/`work`/`home` ...) has no column

`PhoneNumber.Type` (`mobile|work|home|fax|other`) and `Email.Type`
(`work|personal|other`) are organizational sub-types. `contact_addresses.type`
is the CHANNEL type (`tel`|`email`, from `commonaddress.Type`), a different axis.
`target_name` is a label ("John's mobile"), not a type. §7.1's mapping omits the
sub-type entirely.

- Decision (pchero, confirmed): **drop the sub-type.** It was never a real field
  (no code branches on `PhoneNumber.Type`/`Email.Type`; verified). The CRM design
  treats an address as `(channel type, target)`. Reverse-projection returns
  `type = ""`. No `sub_type` column is added to `contact_addresses`.

**DATA-LOSS / IDENTITY interaction with §4 dedup (verified against legacy schema).**
`contact_emails` has NO unique constraint (only a non-unique index on
`(customer_id, address)`; migration `a1b2c3d4e5f6` line 81-82), but the new
`contact_addresses` enforces `UNIQUE (customer_id, type, target)`. This is the
intended CRM model ("identifier dedup, matches agent_addresses", parent design §3.1
line 65): within one customer an address resolves to exactly ONE contact. The
migration collapses legacy duplicates by keeping the LOWEST `id` row and deleting
the rest (pchero decision: pure `id` order, deterministic, no manual gate). Two
collision classes exist, both handled identically (delete losers, enforce UNIQUE):

- **(a) same-contact fold (benign).** One contact holds the same address twice under
  different sub-types (`john@x.com` as `work` AND `personal`), or case variants
  (`John@x.com` vs `john@x.com`). Both fold to `(customer_id,'email','john@x.com')`.
  Lowest-`id` row survives. The rows carry the identical identifier and the only
  thing distinguishing them was the sub-type we are dropping anyway. Survivor keeps
  the same `contact_id`, so nothing re-attributes.

- **(b) cross-contact collision (re-attribution, accepted).** TWO DIFFERENT contacts
  under one customer hold the same email (shared `info@`, family address, or a
  duplicate-contact data-quality issue). Legacy allowed this; the new UNIQUE forbids
  it. The migration keeps the LOWEST `id` row and DELETES the other (pchero
  decision: just delete and enforce the unique, no manual contact-merge gate). The
  losing contact permanently loses that email identifier, and `ContactLookupByEmail`
  (the inbound hot path) thereafter resolves that address to the surviving contact
  only. This is a deliberate change of "who this address belongs to," and it is the
  intended model (an address = one identity per customer). There is no semantic
  "right" winner across contacts; lowest `id` is an arbitrary-but-deterministic
  choice. Probe §6 A still reports the cross-contact count for visibility in deploy
  notes, but it is informational only and does not block the migration.

`contact_phone_numbers` has neither class of problem: it carries
`UNIQUE (customer_id, number_e164)` (migration `a1b2c3d4e5f6` line 65), so the same
e164 cannot exist twice under one customer (neither same- nor cross-contact). The
collision risk is email-only.

## 4. Migration mapping (active rows only)

`contact_contacts` is soft-deleted (`tm_delete IS NULL` = active, VOIP-1205).
Only addresses whose parent contact is active are migrated.

Source-to-target field mapping:
```
contact_phone_numbers p  -> type='tel',   target=p.number_e164,    target_name=''
contact_emails        e  -> type='email', target=LOWER(e.address), target_name=''
(both)                   -> contact_id, customer_id, tm_create preserved
```

**CRITICAL: this is ONE INSERT from a unified CTE, NOT two independent
`INSERT...SELECT` statements.** Two independent inserts are structurally broken
here: a contact that has both a primary phone AND a primary email maps both rows to
`primary_contact_uk = contact_id`, and MySQL/MariaDB check `UNIQUE` immediately
(not deferred), so the second insert fails with ERROR 1062 before any demote can
run. The migration MUST therefore UNION both legacy tables first, then dedup and
compute the final `is_primary` across the union. Ordering is pure `id` per pchero
decision (lowest legacy `id` wins, fully deterministic). It is TWO-STAGE on
purpose: dedup picks survivors by `id` FIRST, then primary is computed by `id` over
ONLY the survivors. (If primary were computed before dedup, a non-primary lower-id
duplicate could win dedup and delete the primary row, leaving a contact with no
primary. Staging avoids that.) Skeleton:

```sql
INSERT INTO contact_addresses
  (id, customer_id, contact_id, type, target, target_name, is_primary, tm_create)
WITH unioned AS (
  SELECT p.customer_id, p.contact_id, 'tel'   AS type, p.number_e164  AS target,
         p.is_primary, p.tm_create, p.id AS id_src
  FROM contact_phone_numbers p
  JOIN contact_contacts c ON c.id=p.contact_id AND c.tm_delete IS NULL
  UNION ALL
  SELECT e.customer_id, e.contact_id, 'email', LOWER(e.address),
         e.is_primary, e.tm_create, e.id
  FROM contact_emails e
  JOIN contact_contacts c ON c.id=e.contact_id AND c.tm_delete IS NULL
),
deduped AS (   -- STAGE 1: one row per (customer,type,target), lowest id wins
  SELECT *,
    ROW_NUMBER() OVER (PARTITION BY customer_id, type, target
                       ORDER BY id_src ASC) AS dup_rank
  FROM unioned
),
survivors AS (
  SELECT * FROM deduped WHERE dup_rank = 1
),
ranked AS (    -- STAGE 2: one primary per contact, lowest id wins, over survivors
  SELECT *,
    ROW_NUMBER() OVER (PARTITION BY contact_id, is_primary
                       ORDER BY id_src ASC) AS primary_rank
  FROM survivors
)
SELECT
  UNHEX(REPLACE(UUID(),'-','')),
  customer_id, contact_id, type, target, '',
  CASE WHEN is_primary=1 AND primary_rank=1 THEN 1 ELSE 0 END,
  tm_create
FROM ranked;
```

Notes on the skeleton:
- STAGE 1 `deduped`/`survivors`: `dup_rank=1` keeps exactly one row per
  `(customer_id, type, target)` (lowest `id`), satisfying
  `UNIQUE (customer_id, type, target)`. Folded email duplicates are dropped (a real
  row deletion, see §3.2). tel is collision-free at source, so `dup_rank` is a
  no-op (always 1) for tel.
- EDGE CASE (verified, accepted): if a same-contact email fold has its lower-`id`
  row marked NON-primary and its higher-`id` row marked primary, STAGE 1 keeps the
  lower-`id` (non-primary) row and the primary row is deleted. The contact then ends
  with that email NOT primary, i.e. it can lose a primary flag it previously had.
  This is a deterministic consequence of pure lowest-`id` ordering on both stages
  and is accepted (is_primary is a preferred-address hint, not a required field; no
  invariant demands every contact have a primary).
- STAGE 2 `ranked`: among a contact's survivor rows, `primary_rank` partitions by
  `(contact_id, is_primary)` and orders by `id_src`. Final `is_primary` is 1 only
  for the single `is_primary=1 AND primary_rank=1` row per contact (lowest `id`
  among its primaries); all other ex-primaries demote to 0. This satisfies
  `UNIQUE (customer_id, primary_contact_uk)` in one insert, and because it runs on
  survivors, dedup can never delete the chosen primary.
- DO NOT remove `is_primary` from the STAGE 2 partition key. The `is_primary=0`
  group also gets a `primary_rank=1` row, but the final CASE's `is_primary=1` guard
  discards it, so non-primary rows never become primary. Partitioning by
  `(contact_id, is_primary)` (not just `contact_id`) is what keeps the primary
  ranking from being polluted by non-primary rows. A "simplification" that drops
  `is_primary` from the partition would silently break this.
- Zero-primary contacts are fine: if a contact has no `is_primary=1` survivor, all
  its rows compute `primary_contact_uk=NULL`, which is distinct under MySQL UNIQUE,
  so no violation.
- All ordering is `id_src ASC` (legacy PK, unique) so there are NO ties: the output
  is fully deterministic and the migration is repeatable.
- MariaDB 10.2+/MySQL 8.0 both support window functions; the production build
  pipeline (MariaDB -> mysqldump -> MySQL 8.0) runs `upgrade()` on MariaDB, so
  this is available. Verified at implementation on a local throwaway DB.

Constraint interactions (why the skeleton is correct):
- `id` is regenerated per row (`UNHEX(REPLACE(UUID(),'-',''))`); legacy ids are not
  preserved (they were never external handles), but `id_src` is used as the final
  deterministic tiebreaker inside the CTE.
- `UNIQUE (customer_id, type, target)`:
  - **tel: cannot collide** at source — `contact_phone_numbers` enforces
    `UNIQUE (customer_id, number_e164)` (legacy migration line 65). `dup_rank` is a
    harmless no-op for tel (always 1). (The §7.1 "distinct e164 86 vs 781 rows"
    figure counts across customers/contacts and is NOT evidence of per-customer
    duplication.)
  - **email: can collide** — no legacy unique + `LOWER()` folds case/sub-type
    variants. `dup_rank` resolves it deterministically (lowest `id` survives).
    Pre-migration probe §6 A counts how many rows this deletes.
- `CHECK (is_primary=0 OR contact_id IS NOT NULL)`: every migrated row has a
  non-null `contact_id` (we only take rows with a parent), so the check always
  holds. Unresolved (contact_id NULL) addresses do not arise in M1.
- `UNIQUE (customer_id, primary_contact_uk)`: legacy enforced this for NEITHER
  table reliably (phone unique is on `(customer_id, number_e164)`, independent of
  `is_primary`; email has no unique). So legacy may hold, per contact: (a) a primary
  phone AND a primary email (cross-type), and (b) two primary phones / two primary
  emails (same-type). The STAGE 2 `primary_rank` window collapses ALL of these to
  one primary per contact, deterministically by lowest `id_src` (over the deduped
  survivors).

## 5. Execution model

### 5.1 Who runs the data migration

AI must not run `alembic upgrade` against staging/production
(`bin-dbscheme-manager/CLAUDE.md` hard rule). Therefore:

- The data move is authored as an **Alembic data-migration file** in
  `bin-dbscheme-manager/bin-manager/main/versions/`, chained after `ac5d4e18060c`,
  generated via `alembic revision` (never hand-picked revision id).
- `upgrade()` runs the single unified-CTE `INSERT` from §4 inside the implicit
  transaction.
- `downgrade()` is `TRUNCATE contact_addresses`. **This is safe ONLY before code
  cutover, when `contact_addresses` holds exactly this migration's output and has
  no other writer.** After cutover the new contact-manager binary writes live user
  CRUD into `contact_addresses`, and at that point this `downgrade()` would destroy
  production data. The downgrade body therefore carries an explicit guard comment:
  "DO NOT downgrade past this revision after the M1 code is deployed; the table is
  then the live source of truth. Roll back via binary redeploy + forward-fix
  migration, not schema downgrade." The real operational rollback path is the
  binary redeploy in §5.2, never an `alembic downgrade` in production.
- pchero executes `upgrade()` in the deploy window. The migration file is the
  deploy unit.

Note: per VOIP-1206's cross-engine note, `contact_addresses` has a STORED
generated column. This migration is DATA-only (no schema build step in the Docker
pipeline re-dumps populated data), and the `INSERT...SELECT` never lists
`primary_contact_uk`, so the ERROR 3105 class does not arise here.

### 5.2 Deploy ordering (one window)

1. Run data migration (`INSERT...SELECT`). `contact_addresses` populated.
2. Verify: row counts match probe expectations; spot-check caller-ID lookups
   against both old and new tables return the same contact.
3. Deploy contact-manager code reading `contact_addresses`.
4. Legacy `contact_phone_numbers` / `contact_emails` become dormant (not dropped
   in this ticket — table DROP deferred so rollback = redeploy old binary).

Rollback trigger: caller-ID match-failure rate spike -> redeploy previous binary
(reads legacy tables, still intact). Table DROP is a separate later ticket once
M1 is proven in production.

**Split-brain on rollback (accepted, with a forward-fix-only policy).** The new
binary writes address CRUD ONLY to `contact_addresses`; it does NOT dual-write the
legacy tables. So any address change made between cutover and a rollback (new
number added, primary changed, address deleted) lands in `contact_addresses` but
NOT in the now-frozen legacy snapshot. A naive "redeploy old binary" would then
silently lose or revert those edits, because the old binary reads the stale legacy
tables. This is a real divergence, accepted rather than engineered around, on these
grounds:
- The rollback window is expected to be minutes (caller-ID failure spike is
  detected fast), and address edits in that window are rare (this is a read-heavy
  lookup path, not a write-heavy one).
- Building dual-write or reverse-sync to make rollback lossless is exactly the kind
  of "theoretical-but-unobserved" robustness cost we do not pay before an incident
  proves it necessary.

Policy instead of dual-write: **forward-fix only.** If rollback happens, treat the
legacy tables as authoritative again, and any `contact_addresses` edits made during
the window are re-applied manually from logs if a specific customer reports a lost
edit. The deploy notes record the cutover timestamp so the affected window is
bounded and auditable. If production later shows this window is not negligible, a
follow-up ticket adds reverse-sync; not before.

## 6. Pre-migration probes (run read-only before authoring final SQL)

```sql
-- A. identifier-uniqueness collisions (would break UNIQUE customer_id,type,target)
--    tel arm is expected EMPTY (legacy UNIQUE(customer_id,number_e164) prevents it);
--    email arm is the real one (no legacy unique + LOWER() folding).
--    NOTE: customer_id MUST be table-qualified — it exists on both the address
--    table and contact_contacts, so an unqualified ref is ERROR 1052 ambiguous.
--    The extra COUNT(DISTINCT contact_id) splits the two collision classes of §3.2:
--      distinct_contacts = 1  -> (a) same-contact fold (benign, sub-type/case)
--      distinct_contacts > 1  -> (b) cross-contact collision (hot-path remap, §3.2 b)
SELECT p.customer_id, 'tel' t, p.number_e164 tgt,
       COUNT(*) n, COUNT(DISTINCT p.contact_id) distinct_contacts
FROM contact_phone_numbers p JOIN contact_contacts c ON c.id=p.contact_id AND c.tm_delete IS NULL
GROUP BY p.customer_id, p.number_e164 HAVING n>1
UNION ALL
SELECT e.customer_id, 'email', LOWER(e.address),
       COUNT(*) n, COUNT(DISTINCT e.contact_id) distinct_contacts
FROM contact_emails e JOIN contact_contacts c ON c.id=e.contact_id AND c.tm_delete IS NULL
GROUP BY e.customer_id, LOWER(e.address) HAVING n>1;

-- B. multiple-primary collisions (would break UNIQUE primary_contact_uk).
--    Catches BOTH cross-type (primary phone + primary email) AND same-type
--    (two primary phones / two primary emails) — legacy forbade neither.
SELECT contact_id, COUNT(*) n FROM (
  SELECT p.contact_id FROM contact_phone_numbers p
    JOIN contact_contacts c ON c.id=p.contact_id AND c.tm_delete IS NULL WHERE p.is_primary=1
  UNION ALL
  SELECT e.contact_id FROM contact_emails e
    JOIN contact_contacts c ON c.id=e.contact_id AND c.tm_delete IS NULL WHERE e.is_primary=1
) x GROUP BY contact_id HAVING n>1;
```

The §4 unified-CTE INSERT already self-heals all of A and B in a single pass
(no separate resolution branches are added). These probes are run BEFORE deploy to
(1) quantify how many rows the migration deletes (A, especially the cross-contact
class b for the deploy-notes review list) and (2) confirm the data shape matches
expectations. If A's `distinct_contacts > 1` count is non-zero, those specific
addresses are exported for deploy-notes visibility per §3.2 (b) and §10 item 5.
Per pchero decision the migration does not block on this; it is informational.

## 7. Code changes

`number_e164` is removed everywhere (full breaking deprecation, pchero decision:
VoIPBin itself is the only API consumer today, so break now rather than carry a
compat shim). The field is wired through FOUR layers plus the OpenAPI spec; all
must change together or the build breaks / the contract drifts.

### 7.1 bin-contact-manager

1. dbhandler: repoint `ContactLookupByPhone` -> `contact_addresses WHERE type='tel' AND target=?`;
   `ContactLookupByEmail` -> `type='email'`. Keep the `Limit(1)` + `ContactGet` shape.
2. model `models/contact/phonenumber.go`: remove the `NumberE164` field. The
   surviving `Number` field (db `number`, JSON `number`) carries the normalized
   E.164 value. `models/contact/field.go:42`: remove the
   `PhoneNumberFieldNumberE164 = "number_e164"` enum.
3. dbhandler: rewrite `PhoneNumberListByContactID` / `EmailListByContactID` as
   reverse-projections over `contact_addresses` filtered by `type`, mapped back
   into `contact.PhoneNumber` / `contact.Email` (`Number`=target, sub-type="").
4. dbhandler: rewrite `PhoneNumber{Create,Update,Delete,Get,ResetPrimary}` and
   `Email{...}` to operate on `contact_addresses` (single store). `ResetPrimary`
   now spans the contact (one primary per contact across ALL types, not per type),
   matching the `UNIQUE (customer_id, primary_contact_uk)` invariant.
5. contacthandler `contact.go:71,289`: `p.NumberE164 = normalizeE164(p.NumberE164,
   p.Number)` collapses to `p.Number = normalizeE164("", p.Number)` (normalize the
   single field in place; raw input is gone). Update path `contact.go:334-337`:
   `fields["number_e164"] = normalizeE164(...)` becomes `fields["number"] =
   normalizeE164(...)`.
6. listenhandler `v1_contacts.go:112,368` + request model
   `listenhandler/models/request/contacts.go:38`: remove `NumberE164` field and its
   `json:"number_e164"` tag. Create/update requests no longer accept `number_e164`;
   `number` is the only phone input and is normalized server-side.
7. Drop `phoneNumberTable` / `emailTable` constants; introduce `addressTable =
   "contact_addresses"`.
8. cachehandler: unchanged (body cache only; no per-identifier cache exists).
9. Correct `bin-contact-manager/CLAUDE.md` "Lookup cache" + "Databases" bullets;
   update `docs/domain.md`, `operations.md` table refs.
10. Regenerate mocks (`go generate ./...`).

Note: the `contact-control` CLI `--phone-e164` lookup FLAG (`cmd/contact-control/
main.go:361,378`) is a query parameter value, not the removed field, and STAYS.

### 7.2 bin-common-handler (shared RPC client)

11. `pkg/requesthandler/contact_phonenumbers.go:17-29`:
    `ContactV1PhoneNumberCreate(..., numberE164 string, ...)` drops the
    `numberE164` parameter and the `NumberE164:` assignment into the request struct.
    This is a shared-library signature change; all callers (7.3) must update in the
    same PR. Regenerate the requesthandler mock.

### 7.3 bin-api-manager (REST -> RPC)

12. `pkg/servicehandler/contact.go:317,342` and
    `serviceagent_contact.go:270,295`: drop the `numberE164` parameter and stop
    passing it to `ContactV1PhoneNumberCreate`. Update the corresponding REST
    handler input binding so `number_e164` is no longer read from the request body.
    Regenerate servicehandler mocks.

### 7.4 OpenAPI spec (external contract)

13. `bin-openapi-manager/openapi/openapi.yaml:3447-3449`: remove the `number_e164`
    property from the phone-number schema; change the `number` property description
    from "Phone number as entered." to "Phone number in E.164 format."
14. Regenerate: `bin-openapi-manager` `go generate` (updates `gens/models/gen.go`),
    then `bin-api-manager` `go generate` (updates `gens/openapi_server/gen.go` and
    the redoc `openapi.json` / `api.html`). Never hand-edit generated files.

### 7.5 Breaking-change surface (external-visible)

- REST response: `phone_numbers[].number_e164` key disappears; `number` returns the
  E.164 string.
- REST request: create/update no longer accept `number_e164`; send `number`.
- Webhook payload: `contact_created/updated/deleted` events embed `PhoneNumber`
  (`models/contact/webhook.go:27`), so the `number_e164` key disappears from event
  payloads too.

This is a deliberate breaking change, accepted because VoIPBin is currently the
only API consumer (pchero, confirmed). No deprecation window, no dual-key compat.

## 8. Testing

### 8.1 Data-migration SQL (the high-risk artifact)

Round-trip the §4 unified-CTE INSERT on a local throwaway MySQL 8.0 with a seeded
fixture, and assert each edge case as a regression (NOT a one-off manual check).
The fixture MUST cover, and the test MUST assert:
- **cross-type double primary**: a contact with a primary phone AND a primary email
  -> exactly one survives as primary, lowest `id` wins (no `UNIQUE
  primary_contact_uk` violation on insert).
- **same-type double primary**: a contact with two primary phones -> lowest `id`
  stays primary, the other demotes to 0.
- **email same-contact fold**: one contact with `John@x.com` (work) + `john@x.com`
  (personal) -> one row, deletion count reported.
- **email cross-contact collision**: two DIFFERENT contacts under one customer with
  the same email -> one row survives (lowest `id`), the losing contact loses the
  address; assert the deterministic survivor and that the lookup now resolves to the
  winner (this is §3.2 (b) behavior, must be locked by a test).
- **primary-flag-loss trap (the reason STAGE 1 precedes STAGE 2)**: one contact with
  the same email twice where the LOWER-`id` row is NON-primary and the HIGHER-`id`
  row IS primary -> STAGE 1 keeps the lower-`id` (non-primary) row and deletes the
  primary one, so the contact ends with that email `is_primary=0`. Assert exactly 1
  row and `is_primary=0`. This locks the two-stage ordering; without it a future
  refactor could invert the stages and resurrect the "dedup deletes the primary"
  bug (§4).
- **clean baseline**: a contact with one primary phone + one non-primary email ->
  both migrate unchanged, phone stays primary. (Guards against over-aggressive
  dedup/demote on well-formed data.)
- **soft-deleted parent excluded**: an address whose contact has `tm_delete` set ->
  absent from `contact_addresses`.
- **global invariant**: every contact ends with `SUM(is_primary) <= 1`; all CHECK
  and both UNIQUE constraints hold.

(The MariaDB build -> mysqldump -> MySQL 8.0 round trip for the schema itself is
already proven in 1206; this step proves the DATA insert + generated column on top.)

### 8.2 dbhandler unit tests

- Rewrite the phone/email test fixtures to seed `contact_addresses`; assert lookup
  (caller-ID by tel/email), list (reverse-projection back into PhoneNumber/Email),
  CRUD, and the cross-type single-primary rule enforced by `ResetPrimary`.

### 8.3 Verification workflow

- Run the full verification workflow (go mod tidy/vendor/generate/test,
  golangci-lint) in EACH touched Go service: `bin-contact-manager`,
  `bin-common-handler`, `bin-api-manager`. The `numberE164` signature removal in
  bin-common-handler will not compile until bin-api-manager callers are updated, so
  build all three together.
- After the OpenAPI spec edit, run `go generate` in `bin-openapi-manager` then
  `bin-api-manager`, and confirm the generated `number_e164` references are gone
  (grep the gens dirs).

## 9. Out of scope

- Dropping `contact_phone_numbers` / `contact_emails` tables (later ticket, after
  M1 proven; keeps rollback cheap).
- Interaction projection (VOIP-1208) and read API (VOIP-1209).
- Interaction backfill (§7.2, explicitly not done).

## 10. Decisions (pchero, confirmed)

All five were reviewed and decided; recorded here for the implementer.

1. **Sub-type drop (§3.2)** — CONFIRMED drop `mobile/work/home` and `work/personal`.
   They were never real fields (no code reads them). No `target_name` mapping, no
   `sub_type` column.
2. **Field collapse (§3.1)** — CONFIRMED. Drop the `NumberE164` field; keep a single
   field named `number` whose VALUE is the normalized E.164 string. Normalization is
   preserved; the raw input form is dropped; `phone_numbers[].number_e164` disappears
   from the REST payload and `phone_numbers[].number` returns the E.164 value.
3. **Primary-collision rule (§4)** — CONFIRMED: resolve by lowest `id`. When a contact
   has multiple primaries (cross-type primary phone + primary email, or same-type two
   primaries), the lowest-`id` survivor among its deduped rows stays primary; the rest
   demote to `is_primary=0`.
4. **Email same-contact dedup (§3.2 a / §4)** — CONFIRMED. When one contact holds the
   same address under two sub-types or case variants, fold to one row (lowest `id`
   survives) and DELETE the rest. Same identifier, same contact_id, no re-attribution.
5. **Cross-contact email collision (§3.2 b)** — CONFIRMED: just delete and enforce the
   UNIQUE, no manual contact-merge gate. When two different contacts share an email,
   the lowest-`id` row survives, the other contact loses the address, and
   `ContactLookupByEmail` resolves it to the survivor. Probe §6 A's
   `distinct_contacts > 1` count is recorded in deploy notes for visibility only; it
   does not block the migration.
