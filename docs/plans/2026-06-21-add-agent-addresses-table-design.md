# Normalize agent addresses into a dedicated agent_addresses table

- Issue: voipbin/monorepo#1005
- Follow-up to: #1002 / PR #1004 (shared NormalizeTarget + symmetric store/lookup + backfill)
- Class: schema refactor + data migration (encapsulated in bin-agent-manager)
- Date: 2026-06-21

## 1. Problem Statement

Agent endpoint addresses are stored as a JSON column `agent_agents.addresses`
(`[]commonaddress.Address`), and the owner lookup is an exact match:

```sql
json_contains(addresses, JSON_OBJECT('type', ?, 'target', ?))
```
(`bin-agent-manager/pkg/dbhandler/agent.go:328`, `AgentGetByCustomerIDAndAddress`).

Code-verified structural drawbacks:

1. **No index**: `json_contains` cannot use an index, so every by-address owner
   lookup is a full `agent_agents` scan. This runs on a live inbound
   call-routing path (call-manager `start.go` + `dial.go` getAddressOwner are the
   only two production callers, established in #1004 review).
2. **Nondeterministic owner**: the lookup query has no `ORDER BY` and takes the
   first row (`agentGetFromRow` on the first `rows.Next()`), so two agents sharing
   a canonical target route to whichever row MySQL yields first.
3. **No DB-level uniqueness**: there is no constraint on
   `(customer_id, type, target)`. Duplicate ownership can only be surfaced by
   application logging (the #1004 backfill logs it) but never prevented.
4. **Maintenance friction**: backfill/iteration over the JSON column needs the
   pagination + JSON-rewrite gymnastics #1004 had to build.

Why it matters: agent addresses are a live call-routing key. The current design
is correct only by luck of formatting (which #1004 made canonical) and pays a
full-scan + nondeterminism + no-uniqueness cost on every routing decision.

## 2. Scope

In scope (all inside bin-agent-manager; the RPC contract and the
`agent.Addresses` domain model stay UNCHANGED):

1. New child table `agent_addresses` (Alembic schema migration in
   `bin-dbscheme-manager`), with a UNIQUE constraint that makes duplicate
   ownership a DB-enforced invariant.
2. dbhandler rewrite of the address touch points to read/write the child
   table instead of the JSON column:
   - READ (hydrate `agent.Addresses` on Get/List/GetByUsername).
   - CREATE (`AgentCreate`): insert agent row + address rows in one transaction.
   - SET (`AgentSetAddresses`): replace the agent's address rows in one
     transaction.
   - DELETE (`AgentDelete`): when an agent is soft-deleted, HARD-DELETE its
     `agent_addresses` rows in the same operation, so the agent stops owning its
     targets (frees the UNIQUE slots) and the by-address lookup can no longer
     resolve to a deleted agent (see §3.3 delete path + §3.4).
   - LOOKUP (`AgentGetByCustomerIDAndAddress`): indexed equality on the child
     table instead of `json_contains`.
3. Data migration: populate `agent_addresses` from the existing
   `agent_agents.addresses` JSON for every agent, then (in a later, separate
   step once verified) drop the JSON column. The drop is deliberately NOT in the
   same migration (see §3.5, expand/contract).
4. Tests: dbhandler tests for the new read/write/lookup paths against the
   sqlite/MySQL test harness; agenthandler tests unchanged in contract.

Out of scope:

- Address normalization rules (delivered in #1002; the new table stores the
  already-canonical values).
- Any RPC contract / domain model / webhook change visible to other services
  (`agent.Addresses` stays the wire/domain shape).
- queue-manager (routes by `tag_ids`, not address; confirmed not affected).
- call-manager / api-manager (consume `agent.Addresses` via RPC / domain model;
  not exposed to the schema change).

## 3. Design

### 3.1 Schema

New table (Alembic, mirrors the existing `agent_agents` column conventions:
`binary(16)` ids, `datetime(6)` timestamps):

```sql
-- step 1: table WITHOUT the unique constraint (so the data backfill cannot fail)
CREATE TABLE agent_addresses (
  id           binary(16),    -- surrogate PK (uuid)
  agent_id     binary(16),    -- FK-by-convention to agent_agents.id
  customer_id  binary(16),    -- denormalized for the by-(customer,addr) lookup + UNIQUE
  type         varchar(255),
  target       varchar(255),
  target_name  varchar(255),  -- carries commonaddress.Address.TargetName
  name         varchar(255),  -- carries commonaddress.Address.Name
  detail       text,          -- carries commonaddress.Address.Detail
  idx          int,           -- preserves address ORDER within an agent (ring order)

  tm_create    datetime(6),
  tm_update    datetime(6),

  PRIMARY KEY (id)
);
CREATE INDEX idx_agent_addresses_agent_id ON agent_addresses(agent_id);
-- non-unique lookup index, present from the start (used by the hot path)
CREATE INDEX idx_agent_addresses_owner ON agent_addresses(customer_id, type, target);

-- step 2 (AFTER the data backfill AND after duplicates are resolved, see §3.5):
-- promote the lookup index to a UNIQUE constraint
CREATE UNIQUE INDEX uniq_agent_addresses_owner
  ON agent_addresses(customer_id, type, target);
DROP INDEX idx_agent_addresses_owner ON agent_addresses;
```

Design points:
- **HARD DELETE on the child table (no `tm_delete`).** Resolved Q1. A child
  address row has no independent identity, history, or external reference (it is
  always owned by exactly one agent and only ever read as part of that agent).
  `AgentSetAddresses` therefore DELETEs the agent's rows and re-inserts. This lets
  a plain `UNIQUE(customer_id, type, target)` enforce single live ownership
  cleanly. Including `tm_delete` in the UNIQUE (the v1 mistake) would NOT work:
  MySQL treats multiple NULLs as distinct, so two live rows (`tm_delete IS NULL`)
  would bypass the constraint and defeat the whole point. MySQL has no partial
  `UNIQUE ... WHERE` index, and the codebase uses `tm_delete IS NULL` for live
  (no `9999` sentinel convention), so hard-delete is the clean fit.
- **`idx` column preserves order.** `commonaddress.Address` order is semantically
  meaningful (linear ring method tries addresses in sequence). On read the rows
  are `ORDER BY idx`; on `AgentSetAddresses` the rows are reassigned `idx = 0..n-1`
  in slice order inside the transaction.
- **UNIQUE added in a SEPARATE step, after duplicate resolution** (§3.5), because
  it is a NEW invariant that did not exist under the JSON column.
- **`customer_id` denormalized** onto the child row so the by-address lookup is a
  single-table indexed equality (`customer_id=? AND type=? AND target=?`)
  returning `agent_id`, with no join needed for the hot path.

### 3.2 dbhandler read path (hydrate agent.Addresses)

`agent_agents` no longer carries `addresses`. Every code path that returns an
`*agent.Agent` (`AgentGet` via cache/DB, `AgentList`, `AgentGetByUsername`,
`agentGetFromRow`) must hydrate `Addresses` from `agent_addresses`:

- Single-agent reads: after scanning the agent row, `SELECT ... FROM
  agent_addresses WHERE agent_id=? AND tm_delete IS NULL ORDER BY idx`.
- `AgentList` (N agents): avoid N+1 by one `SELECT ... WHERE agent_id IN (...)
  AND tm_delete IS NULL ORDER BY agent_id, idx` and group in Go.
- The Redis cache stores the full `agent.Agent` (already includes `Addresses`),
  so cache hits need NO extra query; only DB reads hydrate. (Confirmed: cache is
  id-keyed full-agent JSON.)

### 3.3 dbhandler write paths (transactional)

The current `AgentCreate` is a single `INSERT`; `AgentSetAddresses` is a thin
wrapper that delegates to the shared `AgentUpdate`/`agentUpdate` (PrepareFields +
SetMap + single Exec, also used by SetTagIDs/SetStatus/etc.). Both become
multi-statement against two tables and MUST be transactional so the agent row and
its address rows never diverge. `AgentSetAddresses` can no longer reuse the shared
`agentUpdate` path (that path writes only `agent_agents`); it gets its own tx
implementation.

- **AgentCreate**: `BEGIN; INSERT agent_agents (no addresses col once contract
  lands); INSERT agent_addresses (one row per address, idx = 0..n-1); COMMIT`.
- **AgentSetAddresses**: `BEGIN; DELETE FROM agent_addresses WHERE agent_id=?;
  INSERT new rows (idx = 0..n-1 in slice order); COMMIT`. Hard-delete + re-insert
  (per §3.1) makes the replace atomic and reassigns idx deterministically, so a
  concurrent reader never sees a partial or mis-ordered set.
- **AgentDelete**: today this is a soft-delete of the agent row
  (`agent_agents.tm_delete`). The child table has NO `tm_delete` (hard-delete
  model, §3.1), so AgentDelete MUST also `DELETE FROM agent_addresses WHERE
  agent_id=?` IN THE SAME `*sql.Tx` as the agent soft-delete UPDATE (`BEGIN;
  UPDATE agent_agents SET tm_delete=...; DELETE FROM agent_addresses WHERE
  agent_id=?; COMMIT`). A non-transactional "agent soft-deleted but child rows
  survive" partial failure would reproduce exactly HIGH#N1 (orphan rows occupy the
  UNIQUE slot + lookup resolves to a deleted agent), so the same-tx guarantee is
  load-bearing, not cosmetic. Two reasons this delete is required (adversarial
  HIGH#N1): (1) a soft-deleted agent must stop owning its
  `(customer_id,type,target)` slots, otherwise the UNIQUE permanently blocks any
  other agent from ever taking that target; (2) the new lookup (§3.4) filters only
  on `(customer_id,type,target)` and no longer joins the agent's `tm_delete`, so
  leaving child rows behind would let the by-address lookup resolve to a DELETED
  agent. After commit, INVALIDATE the agent's cache entry (same guard as the other
  write paths below), so a cache-first by-id read does not serve a deleted agent's
  stale Addresses. NOTE: all agent-deletion paths converge here (the
  `EventCustomerDeleted` bulk delete also routes through `deleteForce ->
  AgentDelete`), and the codebase has no agent restore/undelete path, so this one
  change covers every deletion and there is no "addresses lost on restore" case.
- Requires a `*sql.Tx` path in dbhandler (none exists today; `main.go` holds only
  `db *sql.DB`). Add a minimal tx helper used by these write methods.

**UNIQUE violation -> domain error (resolves adversarial H2).** The UNIQUE
constraint is a NEW invariant. Two write entry points reach it differently:
- `UpdateAddresses` (agenthandler) ALREADY runs a cross-agent dup-check
  (`GetByCustomerIDAndAddress` then `ag.ID != a.ID`), so it mostly rejects
  cross-agent duplicates at the app layer today. But that check + insert is NOT
  atomic, so the DB UNIQUE is still the required backstop against a race.
- `AgentCreate` (agenthandler) does NOT check addresses against other agents at
  all (it only checks username existence). A brand-new agent registering a target
  already owned by another agent succeeds today and will now hit the UNIQUE. This
  is the real new rejection surface.
In both cases the write paths MUST translate a MySQL duplicate-key error (1062)
into a typed domain error (e.g. `cerrors.AlreadyExists` /
`ADDRESS_ALREADY_ASSIGNED`), the same class the UpdateAddresses dup-check already
returns, so callers get a clean 409-style result rather than a raw DB error. The
RPC envelope is unchanged; only an additional rejection case (mainly on Create) is
added. Documented here and in §6.

**Cache divergence guard (resolves adversarial M3).** Today `agentUpdateToCache`
runs after the write and its error is discarded (`_ = ...`), outside any tx. With
the split, a committed DB change followed by a failed cache refresh would leave a
stale cached agent (old Addresses) while the by-address lookup reads the new DB
rows -> the routing lookup (lookup -> id -> cache-first AgentGet) could return an
agent whose cached Addresses disagree with ownership. Mitigation: on a cache
refresh failure after commit, INVALIDATE (delete) the agent's cache entry instead
of leaving it stale, so the next read falls through to the DB and re-hydrates.
This keeps cache-vs-DB convergence without requiring the cache write to be in the
DB transaction.

### 3.4 dbhandler lookup path (the win)

`AgentGetByCustomerIDAndAddress` changes from a full-scan `json_contains` to:

```sql
SELECT agent_id FROM agent_addresses
WHERE customer_id=? AND type=? AND target=?
LIMIT 1;            -- then load the agent by id (cache-first)
```

Indexed by `idx_agent_addresses_owner` (later promoted to `uniq_...`).
Deterministic owner because, once the UNIQUE constraint is in place, there is at
most one owner of a canonical target (the nondeterminism in §1.2 is eliminated
by construction, not by ORDER BY). The agent is then loaded by id through the
existing cache-first `AgentGet`.

**Deleted-agent exclusion (adversarial HIGH#N1).** The old `json_contains` query
joined `WHERE tm_delete IS NULL` to skip soft-deleted agents. The new lookup does
NOT filter on the agent's `tm_delete`; instead, a deleted agent has NO
`agent_addresses` rows (AgentDelete hard-deletes them, §3.3), so it simply cannot
match. This preserves the old behavior (deleted agents never win a by-address
lookup) by data presence rather than by a join predicate.

### 3.5 Migration strategy (expand/contract + duplicate gate, two PRs)

A schema split of a live routing key is done expand/contract to stay reversible.
The ordering below makes the UNIQUE constraint the LAST step, after a duplicate
gate, so the new invariant cannot fail the data migration (resolves adversarial
H1).

```
PR A (this PR) — EXPAND, run inside a maintenance window (consumers stopped):
  1. Alembic: CREATE agent_addresses with the NON-unique lookup index only
     (no UNIQUE yet). The JSON column stays.
  2. Stop agent-mutating + by-address consumers (scale agent-manager +
     call-manager consumers to 0), per the #1004 §3.4 precedent, so no concurrent
     write races the migration. (Accepted brief downtime — CEO decision class.)
  3. Data backfill: populate agent_addresses from the JSON column for every agent
     (idx = array position). Reuse the agent-control one-off pattern from #1004
     (Go, zero drift, ships in the image), NOT an Alembic data step.
  4. DUPLICATE GATE: run a detection query
     `SELECT customer_id,type,target,COUNT(*) FROM agent_addresses
      GROUP BY customer_id,type,target HAVING COUNT(*)>1`.
     - If any duplicates exist, STOP: a human resolves them (decide the true
       owner, drop the loser's row) BEFORE the UNIQUE is added. The #1004
       collision log is the input. The backfill tool prints this report.
     - NOTE: the #1004 prod dry-run already reported "collisions: none" for the
       1293 live agents, so in practice this gate is expected to pass cleanly;
       it exists so the migration is correct even if that ever changes.
  5. Only after the gate passes: promote the index to UNIQUE
     (`CREATE UNIQUE INDEX ...; DROP INDEX idx_...`).
  6. Deploy the new agent-manager code (reads + writes use agent_addresses;
     UNIQUE violations mapped to a domain error per §3.3). The JSON column is left
     in place, untouched, purely as a rollback safety net for this PR.
  7. Resume consumers. Verify by-address routing.

PR B (separate, AFTER PR A is verified in prod) — CONTRACT:
  8. Drop the agent_agents.addresses JSON column and the `addresses,json` db tag
     from the Agent struct.
```

Why this ordering:
- The UNIQUE is added only after the duplicate gate, so the migration can never
  abort on a 1062 mid-backfill (H1).
- The irreversible step (dropping the JSON column) is isolated in PR B, after
  prod verification, so PR A is revertible. During PR A the JSON column is read-
  frozen (new code reads the table) but left intact; a revert to pre-PR code reads
  the JSON column, which is still the last-known-good snapshot from before the
  window. (Because writes were stopped during the window, the JSON column is not
  stale relative to the cutover.)
- Stop-the-world (not dual-write) is chosen for simplicity, consistent with #1004;
  dual-write was considered and rejected as unnecessary complexity given the
  accepted brief-downtime decision. The trade-off: a revert AFTER the window has
  resumed writes would lose writes made to the table-only path; the runbook must
  state that a revert is only clean before consumers resume (mirrors the #1004
  forward-only guard).

### 3.6 Affected files

| Service | File | Change |
|---------|------|--------|
| bin-dbscheme-manager | `bin-manager/main/versions/<rev>_create_agent_addresses.py` | CREATE TABLE + indexes; data migration populate from JSON |
| bin-agent-manager | `pkg/dbhandler/agent.go` | tx-based AgentCreate + AgentSetAddresses; AgentDelete also hard-deletes child rows; hydrate Addresses on Get/List/GetByUsername; rewrite AgentGetByCustomerIDAndAddress to indexed child-table lookup; map 1062 -> ADDRESS_ALREADY_ASSIGNED; cache-invalidate on post-commit refresh failure |
| bin-agent-manager | `pkg/dbhandler/agent_address.go` (new) | child-table CRUD helpers (insert rows, list by agent_id(s), replace, lookup owner) |
| bin-agent-manager | `pkg/dbhandler/agent_test.go` / new test file | read/write/lookup regression tests incl. order preservation + UNIQUE collision |
| bin-agent-manager | `scripts/database_scripts_test/agent_addresses.sql` (new) | test-harness schema |
| bin-agent-manager | `models/agent/*` | drop the `addresses,json` db tag from the Agent struct (column no longer on agent_agents) once contract PR lands; during expand it stays for dual-write |

## 4. Test Strategy

- dbhandler: create an agent with N ordered addresses, read back and assert order
  (idx) is preserved; SetAddresses replaces the set atomically AND reassigns idx
  0..n-1 (assert order after a replace that reorders/removes elements); lookup
  returns the right agent by (customer,type,target).
- UNIQUE collision: two agents, same canonical target -> the second write returns
  the mapped domain error (`ADDRESS_ALREADY_ASSIGNED`), NOT a raw DB error. This
  test MUST run against the real MySQL engine, not sqlite: NULL-UNIQUE and
  duplicate-key (1062) semantics differ, so a green sqlite run does not prove the
  constraint (adversarial L3 / MEDIUM#2). See §5 for the MySQL harness requirement.
- delete: AgentDelete (soft-deletes the agent) hard-deletes the agent's
  agent_addresses rows; assert (1) a by-address lookup for that target no longer
  resolves (deleted agent excluded), and (2) another agent can now register the
  freed target without a UNIQUE collision (slot released) (adversarial HIGH#N1).
- N+1 guard: AgentList with several agents issues ONE address query, not N. If the
  id list can exceed a safe placeholder count, the query is chunked (adversarial
  M5); AgentGetByUsername goes through AgentList(size=1) so single reads are fine.
- cache: a cache hit returns Addresses without touching agent_addresses; a cache
  refresh failure after a committed write INVALIDATES the entry (assert the next
  read re-hydrates from DB) (adversarial M3).
- migration/backfill: against a local populated throwaway DB, assert row counts
  match (sum of JSON array lengths == agent_addresses rows), a spot-check of order,
  AND that the duplicate-gate detection query returns the expected rows on a
  seeded-duplicate fixture (so the gate is proven to catch H1, not just assumed).
- Full verification workflow (go mod tidy/vendor/generate/test, golangci-lint) in
  bin-agent-manager; `alembic upgrade head` on a local throwaway DB only.

## 5. Verification

Per CLAUDE.md: full workflow in bin-agent-manager. Migration applied to
staging/prod by a human (AI prohibition on prod mutation). The by-address lookup
is exercised by the existing call-manager getAddressOwner regression tests
(contract unchanged) plus new dbhandler tests.

**MySQL harness requirement (adversarial MEDIUM#2 / L3).** The dbhandler test
harness today uses in-memory sqlite (`dbhandler/main_test.go`), which does NOT
reproduce MySQL's UNIQUE / 1062 / multi-NULL semantics. The UNIQUE-collision and
1062->domain-error tests therefore CANNOT be proven on sqlite. The design requires
one of: (a) a dedicated MySQL-backed integration test target run in CI for these
specific tests, or (b) if that is infeasible in CI, an explicit documented manual
verification step against a local MySQL before the migration is applied. A green
sqlite suite alone does not certify the constraint behavior; this MUST be called
out in the PR so the reviewer does not mistake sqlite-green for proof.

## 6. Sections marked N/A

New REST API / webhook / flow vars / RabbitMQ action / Prometheus / PII-LLM:
N/A. No new endpoint or entity; the domain model and RPC contract are unchanged.
This is an internal storage refactor of one existing service.

## 7. Open Questions (resolved in v2)

| # | Question | Decision | Owner |
|---|----------|----------|-------|
| 1 | UNIQUE + soft-delete (NULL defeats MySQL UNIQUE for live rows) | HARD-DELETE child rows on replace; plain `UNIQUE(customer_id,type,target)`. Child rows have no independent identity/history/reference. (§3.1) | CTO |
| 2 | dbhandler *sql.Tx helper? | None exists; add a minimal tx helper for the two write paths. (§3.3) | CTO |
| 3 | Data migration mechanism | Reuse the agent-control one-off (Go, zero drift, ships in image); Alembic only does CREATE TABLE. (§3.5) | CEO/CTO |
| 4 | Cutover: dual-write vs stop-the-world | Stop-the-world in a maintenance window (consumer scale-to-zero), per #1004; forward-only after resume. (§3.5) | CEO |
| 5 | Drop the JSON column in this PR or follow-up | Follow-up PR (contract), after prod verification of PR A. (§3.5) | CEO/CTO |
| 6 | Pre-existing cross-agent duplicates failing the UNIQUE migration | Duplicate GATE before adding UNIQUE: detect, human-resolve, then add UNIQUE. #1004 prod dry-run already shows 0 collisions, so expected clean. (§3.5) | CEO/CTO |
| 7 | UNIQUE makes a previously-succeeding cross-agent duplicate now fail | Map MySQL 1062 to a typed domain error (`ADDRESS_ALREADY_ASSIGNED`); the real new surface is Create (which has no app dup-check); UpdateAddresses already app-checks. (§3.3, §6) | CTO |
| 8 | Soft-deleted agent vs hard-delete child rows | AgentDelete hard-deletes the agent's agent_addresses rows in the same op, so deleted agents release UNIQUE slots and cannot win a by-address lookup (replaces the old `tm_delete IS NULL` join). (§3.3, §3.4) | CTO |

## 8. Review Summary

### Round 1 (2 reviewers) — code-verification APPROVE / adversarial CHANGES REQUESTED

Code-verification: all factual claims accurate (json_contains lookup, 4 touch
points, single-statement create, id-keyed full-agent cache, commonaddress fields,
queue-manager tag-based, call/api consume via RPC). Corrections: AgentSetAddresses
delegates to the shared agentUpdate (cannot be reused for the tx path); hydration
must happen at Get/List level, not inside ScanRow. Encapsulation premise holds.

Adversarial: CHANGES REQUESTED. HIGH#1 pre-existing duplicates would fail the
UNIQUE data migration (no gate); HIGH#2 UNIQUE silently changes API behavior
(previously-succeeding cross-agent duplicate now errors); HIGH#3 §3.1 body schema
`UNIQUE(...,tm_delete)` does not enforce the invariant (MySQL NULL semantics).
MEDIUM: hard-delete safety, idx reassign on replace, cache divergence on
post-commit refresh failure, cutover ordering, IN-list size. LOW: rollback/JSON
drop coupling, db-tag timing, sqlite-vs-MySQL UNIQUE.

### v2 (this revision) — findings applied

- HIGH#3 + Q1: schema body fixed to plain `UNIQUE(customer_id,type,target)` with
  HARD-DELETE child rows (no tm_delete); lookup SQL drops the tm_delete clause.
- HIGH#1 + Q6: §3.5 adds a duplicate GATE (detect -> human-resolve -> THEN add
  UNIQUE), with UNIQUE as the last migration step so the backfill can't abort.
- HIGH#2 + Q7: §3.3 maps MySQL 1062 to `ADDRESS_ALREADY_ASSIGNED`; behavior change
  documented in §3.3/§6.
- MEDIUM: idx reassign 0..n-1 inside the replace tx (§3.1/§3.3); cache-invalidate
  on post-commit refresh failure (§3.3); ordered cutover sequence with consumer
  stop + forward-only guard (§3.5); IN-list chunking + MySQL-only UNIQUE test +
  duplicate-gate test (§4).
- The AgentSetAddresses-delegation and Get/List-hydration corrections folded into
  §3.2/§3.3.

A fresh re-review round on v2 follows (mandatory after the material change).

### Round 2 (adversarial convergence) — CHANGES REQUESTED

Confirmed HIGH#1/#2/#3 genuinely resolved. Found a NEW HIGH#N1: hard-delete on
the child table interacts badly with AgentDelete's soft-delete. A soft-deleted
agent would leave orphan child rows that (a) permanently occupy the UNIQUE slot
and (b) let the new lookup (which dropped the `tm_delete IS NULL` join) resolve to
a DELETED agent. Plus MEDIUM#1 (the §3.3 dup-check causal wording was backwards:
UpdateAddresses already cross-agent-checks; Create is the real new surface) and
MEDIUM#2 (sqlite harness cannot prove the MySQL UNIQUE/1062 behavior).

### v3 (this revision) — Round-2 findings applied

- HIGH#N1 + Q8: AgentDelete now hard-deletes the agent's agent_addresses rows in
  the same op (§3.3 delete path), so deleted agents release UNIQUE slots; §3.4
  documents deleted-agent exclusion by data presence (no child rows) instead of
  the old `tm_delete IS NULL` join. Added as the 5th touch point in §2 and §3.6.
- MEDIUM#1 + Q7: §3.3 corrected — UpdateAddresses already runs a cross-agent
  dup-check (non-atomic, so UNIQUE is still the backstop); AgentCreate is the real
  new rejection surface (no app check). 1062 maps to ADDRESS_ALREADY_ASSIGNED.
- MEDIUM#2: §5 adds the MySQL-harness requirement (sqlite cannot certify the
  constraint; CI MySQL target or documented manual MySQL verification); §4 delete
  test added.

A fresh re-review round on v3 follows (mandatory after this material change).

### Round 3 (final convergence) — APPROVE

Confirmed HIGH#N1 + MEDIUM#1/#2 genuinely resolved against the real code; no new
HIGH/MEDIUM. Verified: AgentDelete is the single convergence point for all agent
deletions (incl. the EventCustomerDeleted bulk path via deleteForce), there is no
agent restore/undelete path, and the maintenance-window consumer stop makes any
mid-flight lookup moot. Two non-blocking nits folded in: (1) §3.3 AgentDelete now
explicitly states the child hard-delete runs in the SAME `*sql.Tx` as the agent
soft-delete (partial failure would re-create HIGH#N1); (2) §3.3 adds cache
invalidation after AgentDelete so a cache-first by-id read does not serve a
deleted agent's stale Addresses. Design is approved for implementation.
