# Agent Address Normalization (symmetric store + lookup) with backfill migration

- Follow-up to: voipbin/monorepo#1002 (shared NormalizeTarget)
- Class: hardening / refactor + data migration
- Date: 2026-06-21

## 1. Problem Statement

Issue #1002 promoted a shared `commonaddress.NormalizeTarget` and adopted it in
6 services, but DEFERRED `bin-agent-manager` because agent addresses are matched
by EXACT byte-for-byte JSON comparison, and normalizing only one side of that
match would break it.

Code-verified facts:

- `bin-agent-manager/models/agent/agent.go:27`: `Addresses []commonaddress.Address`
  persisted as a JSON column (`addresses json`).
- The agent-by-address lookup is an EXACT match:
  `bin-dbscheme-manager`-managed `agents.addresses` is queried via
  `dbhandler/agent.go:328`
  `json_contains(addresses, JSON_OBJECT('type', ?, 'target', ?))` with args
  `(address.Type, address.Target)`. Any difference in the stored vs. queried
  `target` string yields a MISS.
- The lookup key is passed RAW from call-manager:
  `bin-call-manager/pkg/callhandler/start.go:707` and
  `bin-call-manager/pkg/groupcallhandler/dial.go:267` both call
  `AgentV1AgentGetByCustomerIDAndAddress(ctx, 1000, customerID, *addr)` with the
  unnormalized address (the `addr.Type == TypeAgent` branch uses an agent UUID via
  `AgentV1AgentGet` and is NOT an address match).
- Today neither the store nor the lookup normalizes, so they are CONSISTENTLY raw
  (matching works by luck of identical formatting). Introducing normalization on
  only one side breaks live agent call routing.

Why it matters: agent addresses are a LIVE call-routing key. An inbound/outbound
call resolves its owner agent by this exact-match join; a normalization
asymmetry would silently route-miss (agent stops receiving calls).

## 2. Scope

In scope (the symmetric set — ALL must land together):

1. Normalize the agent address STORE:
   - `agenthandler.Create` (agent.go:102): normalize each `addresses[i].Target`
     before persist.
   - `agenthandler.UpdateAddresses` (agent.go:360): normalize each
     `addresses[i].Target` BEFORE the validation/dup-check loop (agent.go:377),
     so the internal `GetByCustomerIDAndAddress` dup-check (agent.go:412) also
     uses the normalized form.
2. Normalize the agent address LOOKUP KEY:
   - `agenthandler.GetByCustomerIDAndAddress` (agent.go:61): normalize the
     incoming `address.Target` before the DB call (covers the RPC entry +
     the internal dup-check caller).
   - `bin-call-manager/pkg/callhandler/start.go` `getAddressOwner` (start.go:689):
     normalize `addr.Target` before the `AgentV1AgentGetByCustomerIDAndAddress`
     call (the non-Agent-UUID branch only).
   - `bin-call-manager/pkg/groupcallhandler/dial.go` `getAddressOwner`
     (dial.go:249): same.
3. BACKFILL (Go one-shot CLI subcommand `agent-control agent normalize-addresses`):
   normalize the `target` of every existing agent's `addresses` JSON so stored
   values match the new normalized lookup key. Without this, EXISTING agents
   route-miss until their addresses are next edited. The subcommand REUSES the
   real `commonaddress.NormalizeTarget` (zero drift), scans all agents via the
   existing `AgentList` token pagination, and rewrites changed addresses via the
   existing `AgentSetAddresses` chokepoint (so cache invalidation runs).
4. Tests: agent-manager store + lookup normalization regression tests;
   call-manager getAddressOwner normalization regression tests (both handlers).

Out of scope:

- The shared `NormalizeTarget` primitive itself (delivered in #1002).
- Re-normalizing other managers (done in #1002).
- Changing the exact-match `json_contains` mechanism (kept; backfill makes it
  correct rather than replacing it with a slow app-level compare).
- An Alembic data migration / Python port of NormalizeTarget (REJECTED, see §3.3):
  any non-Go reimplementation reintroduces a drift-vs-Go risk that the CLI
  subcommand structurally eliminates.

## 3. Design

### 3.1 Normalization call (identical on all 5 code sites)

```go
// loss-proof, idempotent; error discarded (original returned on non-normalizable)
addr.Target, _ = commonaddress.NormalizeTarget(addr.Type, addr.Target)
```

- Value slices (`agent.Addresses`, the agent store paths) normalized BY INDEX.
- The lookup key in call-manager is `*addr` (pointer); normalize `addr.Target`
  before passing `*addr` to the RPC. Do NOT normalize the `addr.Type == TypeAgent`
  branch (agent UUID, opaque — NormalizeTarget would identity it anyway, but leave
  it untouched for clarity).
- `TypeAgent`/opaque types are identity-normalized, so an address whose Type is
  e.g. `tel`/`sip` is the only one whose `target` actually changes — exactly the
  routing keys we need consistent.

### 3.2 The symmetry invariant (the load-bearing correctness point)

The match is `stored.target == normalize(lookup.target)` ONLY IF
`stored.target == normalize(stored.target)`. So:

- New/updated agents: store-side normalization (scope 1) guarantees
  `stored.target` is already canonical.
- Existing agents: the backfill (scope 3) rewrites `stored.target` to canonical.
- Every lookup caller (scope 2): normalizes the key to canonical.

All three legs are required; any one missing reintroduces a route-miss class.
Because `NormalizeTarget` is idempotent, re-running the backfill or
double-normalizing a caller is safe.

### 3.3 Backfill: Go one-shot CLI subcommand (zero-drift)

The backfill is a new cobra subcommand on the EXISTING `agent-control` CLI:
`agent-control agent normalize-addresses`. The `cmd/agent-control` binary is
already built into the deployed agent-manager image (the Dockerfile runs
`go build -o /app/bin/ ./cmd/...`, confirmed), so the backfill ships with the
release and is executed once by an operator via `kubectl exec` / a one-off Job
against staging then prod. No separate build or deploy artifact.

Why CLI subcommand, not an Alembic Python port (decision):

- **Zero drift by construction.** The subcommand calls the SAME
  `commonaddress.NormalizeTarget` that the store and lookup paths call. There is
  no second implementation to keep in parity. An Alembic migration runs in a
  `python:3.11-slim` image with no Go toolchain, forcing a Python reimplementation
  of the tel/whatsapp/email/sip + identity rules (loss-proof no-digit, sip
  host-token-only lowercasing, ASCII-only digit strip). Any byte divergence
  between that port and Go produces a stored canonical that differs from the Go
  lookup key and silently defeats the backfill. The CLI subcommand eliminates that
  entire failure class.
- **Reuses existing chokepoints.** It scans all agents through the existing
  `AgentList(ctx, size, token, filters)` token pagination (`tm_create`-keyed,
  `deleted=false` filter, customer-agnostic) and writes changed agents through the
  existing `AgentSetAddresses` path, so cache invalidation and JSON marshaling
  behave exactly as the live service.
- **Per-element by the element's own `type`.** Agents can hold mixed-type
  addresses (e.g. a `tel` alongside an `extension`). NormalizeTarget is called per
  array element keyed on that element's `Type`, so opaque targets (extension,
  agent UUID) are identity-normalized and never corrupted.

Subcommand behavior:

- **DB + cache wiring.** `AgentSetAddresses` is a `dbhandler.DBHandler` method,
  NOT on the `agenthandler.AgentHandler` interface. The subcommand constructs the
  `dbhandler` directly via `dbhandler.NewHandler(sqlDB, cache)` (the same
  constructor the service uses), which requires a real `cachehandler` — the
  subcommand also builds the Redis cache via `cachehandler.NewHandler(addr, pw,
  db)` (the existing `initCache` pattern), so the id-keyed cache invalidation in
  `AgentSetAddresses` actually runs and live reads (after consumers resume) see
  the canonical value. The kubectl-exec / one-off-pod environment therefore needs
  Redis reachability (runbook note). It deliberately does NOT route writes through
  `agenthandler.UpdateAddresses`, because that path runs the extension registrar
  RPC and the cross-agent dup-check (which would REJECT a backfill write the
  moment normalization creates a collision — exactly the rows we need to write and
  then surface).
- **Full-scan completeness (count reconciliation + page-grow retry).** `AgentList`
  uses a strict `tm_create < token` cursor (microsecond resolution) with no
  tie-breaker. To guarantee no agent is skipped at a page boundary where multiple
  rows share an identical `tm_create`, the subcommand (a) SELECTs the total
  non-deleted agent COUNT before the scan, (b) counts agents actually visited
  during the scan, and (c) on a mismatch RETRIES the whole scan with a larger page
  size (so an equal-`tm_create` cluster fits inside one page), up to a bounded
  number of retries, and only then FAILS LOUDLY. This converts the count check
  from a potential dead-end into a self-healing guard. A skipped agent would
  otherwise stay raw and route-miss forever after the window closes, and a second
  dry-run using the same pagination would not detect it.
- **`--dry-run` (default true).** First pass logs every intended change
  (`agent_id`, `type`, raw -> canonical), the full collision report, and the
  count reconciliation, WITHOUT writing. The operator inspects, then re-runs with
  `--dry-run=false` to apply.
- **Collision gate (hard-fail, no override).** The apply pass (`--dry-run=false`)
  REFUSES to run if the collision count is > 0. There is no override flag: an
  override would simply recommit the HIGH#2 dual-ownership state by another path
  (two agents sharing one canonical -> nondeterministic routing). Collisions are
  resolved by the operator (editing the duplicate agents) and then the apply is
  re-run; the gate is a hard precondition, not a warning.
- **Global collision detection (not per-page).** Collisions are tracked in a
  single map keyed `(customer_id, type, canonical_target)` accumulated across the
  ENTIRE scan, populated from the POST-normalization value of EVERY agent's every
  address (not just the agents that changed). This catches collisions that span
  two different pagination pages, and collisions between an already-canonical
  agent and a freshly-normalized one.
- **Original dump (recovery path).** Because normalization is irreversible (raw
  formatting is lost) and there is no undo subcommand, the apply pass writes each
  agent's PRE-change `addresses` JSON to a log/dump (agent_id + original array)
  before mutating, giving the operator a manual reconstruction path.
- **Only writes changed agents.** An agent whose every address is already
  canonical is skipped (no AgentSetAddresses call), bounding write volume to the
  genuinely-raw rows. Combined with NormalizeTarget idempotency, this is what
  makes a re-run / partial-failure resume safe — no separate write-time CAS guard
  is needed (the §3.4 consumer stop already removes all concurrency).
- **No reverse.** A normalization backfill is not cleanly reversible. There is no
  "undo" subcommand. The operational rollback is the code revert, constrained by
  the §3.5 forward-only guard.

Why collisions must be cleared (nondeterministic routing): the by-address lookup
`AgentGetByCustomerIDAndAddress` (dbhandler/agent.go) issues a `json_contains`
query with NO `ORDER BY` and takes the first row. If two agents share a canonical
target, the routing owner returned is whatever row MySQL yields first —
nondeterministic. The same nondeterminism leaks into `UpdateAddresses`' dup-check
(`ag.ID != a.ID`), which can wrongly block a colliding agent from editing its own
addresses. Collision is therefore a real defect, not a cosmetic one; the gate
forces an operator decision before any canonical collapse is committed.

This is run by a human against staging then prod (AI prohibition on prod
mutation). Local validation uses a populated throwaway DB with `--dry-run` then
`--dry-run=false`.

### 3.4 Transition / deploy ordering (RPC consumers stopped, not "drained")

Per CEO decision a brief downtime is acceptable. v5 spends that downtime to
ELIMINATE the transient window. The earlier "deploy live, then backfill while
serving" ordering is REJECTED: it creates a live read-modify-write race vs
concurrent `UpdateAddresses` (lost-update) and a uniqueness-bypass that can leave
two agents permanently owning the same canonical address (HIGH#1, HIGH#2). Both
vanish only if NO agent-mutating and NO by-address-lookup work runs during the
backfill.

Concrete stop mechanism (VoIPBin is RabbitMQ RPC, not an HTTP LB, so "drain" is
not self-evident). The by-address RPC has exactly two production producers
(call-manager `start.go` + `dial.go` getAddressOwner, established in review), and
the agent-mutating producers are the agent-manager write RPCs. To guarantee zero
concurrency we STOP the RPC CONSUMERS, not "drain traffic":

```
1. Merge PR (5 code sites normalized + agent-control normalize-addresses subcommand).
2. Open a maintenance window. Scale the agent-manager Deployment consumers and the
   call-manager (routing) consumers to ZERO replicas (or otherwise stop their RPC
   consumption). NOTE: stopping the by-address lookup path means inbound call
   routing is OFFLINE for the window — this is full telephony downtime for agent
   routing, not a partial degrade. This is the accepted-downtime cost.
3. Deploy the normalized code (new replicas with normalization + the subcommand).
   Keep their consumers stopped until after backfill, OR run the backfill from a
   one-off pod/job that does NOT consume RPCs (the subcommand talks to the DB
   directly and consumes nothing).
4. Run the backfill against the now-quiescent DB:
   a. `agent-control agent normalize-addresses --dry-run` -> inspect the
      raw->canonical diff, the collision report, and the before/after agent COUNT
      reconciliation (§3.3 completeness check).
   b. If collisions > 0, STOP and resolve duplicates first (apply hard-fails on
      collisions; there is NO override — see §3.3). Collisions make routing
      nondeterministic, so they MUST be cleared inside the window.
   c. `--dry-run=false` to apply, then re-run `--dry-run` to confirm a clean
      (zero-change, zero-collision, count-reconciled) second pass.
5. Restore consumers (scale back up). Every agent address is canonical; store ==
   lookup key for both existing and new agents.
```

Because all three legs (store, lookup, backfill) land before consumers resume,
there is no window in which one side is normalized and the other is not. The
downtime is the stop+deploy+backfill duration, paid once.

Runbook notes (non-blocking, from convergence review):
- Before the window, re-confirm the by-address RPC has no producers beyond the two
  call-manager `getAddressOwner` sites (so the consumer-stop list stays complete
  if new callers were added since this design).
- The backfill pod/exec environment needs BOTH MySQL and Redis reachability
  (DB scan/write + the `SELECT COUNT` reconciliation + id-keyed cache invalidation).
- The pre-change original dump can be large for high agent counts; prefer a
  structured file artifact over inline logs when the changed-row count is high.

### 3.5 Rollback guard (mandatory runbook note)

Once the backfill has rewritten stored addresses to canonical form, rolling the
CODE back to a version that does NOT normalize the lookup key would make raw
lookup keys miss the canonical stored values — an unrecoverable break (raw form
is gone). Therefore:

- The lookup-key normalization is FORWARD-ONLY. The deploy runbook MUST state:
  "do not roll the agent-manager / call-manager code back below the
  agent-address-normalization release." A revert is only safe BEFORE the backfill
  runs (stored values still raw, raw keys still match).
- `NormalizeTarget` is idempotent, so re-applying the release after a revert is
  safe.

### 3.6 Affected files

| Service | File | Change |
|---------|------|--------|
| bin-agent-manager | `pkg/agenthandler/agent.go` | normalize store in Create + UpdateAddresses (by index, before the validation/dup-check loop); normalize lookup key in GetByCustomerIDAndAddress |
| bin-agent-manager | `pkg/agenthandler/agent_test.go` | store + lookup normalization regression tests (incl. extension identity) |
| bin-call-manager | `pkg/callhandler/start.go` | normalize lookup key in getAddressOwner (address-match branch only; TypeAgent UUID branch untouched) |
| bin-call-manager | `pkg/callhandler/start_test.go` | regression test |
| bin-call-manager | `pkg/groupcallhandler/dial.go` | normalize lookup key in getAddressOwner (address-match branch only) |
| bin-call-manager | `pkg/groupcallhandler/dial_test.go` | regression test |
| bin-agent-manager | `cmd/agent-control/main.go` | new `normalize-addresses` subcommand: construct `dbhandler` + Redis `cachehandler` directly, full-scan via AgentList pagination with before/after COUNT reconciliation, NormalizeTarget per element by type, global `(customer_id,type,canonical)` collision map, `--dry-run` (default true), collision gate (hard-fail, no override), pre-change original dump, write changed via AgentSetAddresses |

## 4. Test Strategy

- agent-manager: assert `Create`/`UpdateAddresses` persist canonical
  `Addresses[i].Target` (gomock matcher on the db create/set-addresses call);
  assert `GetByCustomerIDAndAddress` issues the DB query with the canonical
  target (gomock arg match). Include a `tel` punctuated case, a `sip` host-case
  case (host token lowercased, `;params`/`?headers` preserved verbatim), and an
  opaque/UUID identity case (extension/agent untouched).
- call-manager: assert `getAddressOwner` (both start.go and dial.go) calls
  `AgentV1AgentGetByCustomerIDAndAddress` with the canonical address; assert the
  `TypeAgent` UUID branch is unchanged (still `AgentV1AgentGet` by id).
- backfill: the subcommand reuses the real `NormalizeTarget`, so there is NO
  second normalizer to golden-vector. Validation is operational: run with
  `--dry-run` against a local populated throwaway DB, confirm the previewed
  raw -> canonical diffs and collision report are correct, then `--dry-run=false`
  and re-run `--dry-run` to confirm a clean (no-change) second pass (idempotency).

## 5. Verification

Per CLAUDE.md, run the full workflow in bin-agent-manager and bin-call-manager.
For the backfill subcommand: build via the standard agent-manager verification
(`go build ./cmd/...` is exercised by the workflow), then run
`agent-control agent normalize-addresses --dry-run` against a LOCAL throwaway DB
only (never staging/prod — AI prohibition), confirm the diff + collision report,
apply with `--dry-run=false`, and confirm an idempotent clean second pass. A human
runs it against staging/prod.

## 6. Sections marked N/A (hardening-class)

New domain model / REST API / webhook / flow vars / RabbitMQ action / Prometheus
/ PII-LLM: N/A. This adds no entity or endpoint; it inserts an existing
pure-function at 5 sites and backfills one JSON column.

## 7. Open Questions (resolved)

| # | Question | Decision | Owner |
|---|----------|----------|-------|
| 1 | Backfill mechanism? | Go one-shot CLI subcommand `agent-control agent normalize-addresses`, reusing the real `NormalizeTarget` (zero drift). The agent-control binary already ships in the deployed image (`go build ./cmd/...`). Alembic Python-port REJECTED: a non-Go reimplementation reintroduces a parity/drift risk the CLI structurally removes. | CEO/CTO |
| 2 | downgrade() reversibility? | No-op (irreversible); the operational rollback is the code revert, constrained by the §3.5 forward-only guard | CTO |
| 3 | Deploy ordering / downtime? | CEO accepted a brief downtime: code-first then immediate backfill (§3.4). Zero-downtime dual-key expand/contract rejected as over-engineering. | CEO/CTO |
| 4 | agent_addresses normalized table instead of JSON column? | NO. Keep the JSON column; table normalization is explicitly out of scope (CEO decision). | CEO |

## 8. Review Summary

### Round 1 (2 reviewers: code-verification + adversarial) — both CHANGES REQUESTED

Code-verification reviewer: symmetric store+lookup set verified COMPLETE against
the repo (5 sites; the two call-manager getAddressOwner methods are the ONLY
production callers of the by-address RPC; agent-manager's single
GetByCustomerIDAndAddress chokepoint covers RPC entry + internal dup-check; cache
is id-keyed not address-keyed, no other store/match site missed). Blocker: the
"Go command" backfill is infeasible (python:3.11-slim migration image, no Go
toolchain; backfill only runs in human staging/prod upgrade against a populated
DB).

Adversarial reviewer: core normalization + symmetry invariant + script-driven
backfill + idempotency all sound. Blockers: (HIGH) deploy-ordering transient
break is unavoidable under single-key EXACT match — recommended dual-key
expand/contract for zero-downtime; (HIGH) no-op downgrade + code rollback =
silent unrecoverable break. Plus MEDIUM: pin backfill to deployed normalizer +
mirror ErrNotNormalizable, confirm extension->identity, log post-normalize
collisions; LOW: address-branch-only placement, slice-by-index mutation test.

### v2 (this revision) — findings applied
- Backfill = Python port embedded in `upgrade()`, per-element by `type`, parity
  tested vs Go golden vectors (resolves code-verify HIGH + fidelity MEDIUM).
- Deploy: CEO accepted brief downtime -> code-first + immediate backfill (§3.4);
  dual-key expand/contract rejected as over-engineering (resolves adversarial
  HIGH #1 per CEO decision).
- Rollback guard §3.5: lookup-normalization is forward-only; runbook must forbid
  rolling below this release post-backfill (resolves adversarial HIGH #2).
- Collision logging in the migration (MEDIUM).
- extension identity test + lookup normalization in the address-match branch only
  / TypeAgent UUID branch untouched / store normalize by index (MEDIUM + LOW).
- agent_addresses table: explicitly out of scope (CEO).

A fresh re-review round on v2 follows (mandatory after the material change).

### v3 (this revision) — backfill mechanism changed per CEO decision

The CEO chose a Go one-shot CLI subcommand over the Alembic Python-port (the SQL
/ pure-query alternative was also weighed and rejected for the same drift reason:
reproducing sip host-token-only lowercasing and the loss-proof no-digit rule
byte-for-byte in SQL is fragile). Changes:

- §2 scope item 3 + §3.3: backfill is now `agent-control agent
  normalize-addresses`, reusing the real `NormalizeTarget` (zero drift by
  construction). Scans via existing `AgentList` pagination, writes via existing
  `AgentSetAddresses` chokepoint, `--dry-run` default-true preview, per-element
  by type, collision logging on both passes.
- Confirmed the agent-control binary already ships in the deployed image
  (Dockerfile `go build -o /app/bin/ ./cmd/...`), so no new deploy artifact;
  operator runs it via `kubectl exec`.
- §3.6 affected-files: dropped the bin-dbscheme-manager Alembic file, added the
  cmd/agent-control subcommand row.
- §4/§5: validation is operational (dry-run preview + idempotent second pass on a
  local throwaway DB), not golden-vector parity (no second normalizer exists).
- §7 Q1 updated; §2 out-of-scope now explicitly rejects the Alembic/Python port.

A fresh re-review round on v3 follows (mandatory after this material change).

### v4 (this revision) — round-1 review findings applied

Round 1 (2 fresh reviewers): code-verification APPROVE (one accuracy note);
adversarial CHANGES REQUESTED (HIGH 2 + MEDIUM 3 + LOW 3). All applied:

- **HIGH#1 (live backfill lost-update race) + HIGH#2 (uniqueness-bypass ->
  permanent dual-ownership):** root cause was "backfill while serving live
  traffic". §3.4 rewritten to run deploy + backfill inside ONE drained
  maintenance window (CEO downtime spent to remove the transient window entirely,
  not just bound it). With no agent-mutating/lookup traffic during the window,
  both HIGH findings are structurally impossible.
- **MEDIUM#3 (collision -> nondeterministic routing, log-only insufficient):**
  added a collision GATE — apply refuses with collisions present unless
  `--allow-collisions`; documented the `ORDER BY`-less first-row nondeterminism
  and the dup-check self-block in §3.3.
- **MEDIUM#4 (collision detection scope):** specified a GLOBAL
  `(customer_id,type,canonical)` map over the whole scan, populated from every
  agent's post-normalization addresses (cross-page, incl. already-canonical).
- **MEDIUM#5 (dry-run/apply TOCTOU):** subsumed by the drained window; plus a
  write-time re-read guard and a confirm-clean second dry-run pass.
- **code-verify accuracy note + LOW#6:** §3.3 now states the subcommand wires
  `dbhandler` directly (AgentSetAddresses is not on the agenthandler interface)
  and writes a pre-change original `addresses` dump as a manual recovery path.
- **LOW#7:** §4 test strategy adds a sip host-case + `;params` preservation case.

A fresh re-review round on v4 follows (mandatory after this material change).

### v5 (this revision) — round-2 review findings applied

Round 2 (2 fresh reviewers): implementability APPROVE (each code change verified
implementable against the repo: signatures, interfaces, pagination loop, no mock
regen, test feasibility); adversarial CHANGES REQUESTED (HIGH 1 + MEDIUM 1 +
LOW 3). Applied:

- **HIGH#1 (drain mechanism unspecified for RabbitMQ):** §3.4 no longer
  hand-waves "drain traffic". It now specifies STOPPING the RPC consumers
  (scale agent-manager + call-manager consumers to zero / run the backfill from a
  non-consuming one-off pod) and explicitly states that stopping the by-address
  lookup path means inbound agent-call routing is fully offline for the window
  (the accepted-downtime cost). The zero-concurrency guarantee is now enforceable.
- **MEDIUM#2 (pagination completeness — same-`tm_create` page-boundary skip):**
  added before/after non-deleted-agent COUNT reconciliation that FAILS LOUDLY on
  mismatch (a same-microsecond straddle would otherwise leave an agent raw and a
  second dry-run, using the same cursor, would not catch it).
- **LOW#3 (`--allow-collisions` footgun):** removed. The collision gate is now a
  hard-fail with no override; collisions are resolved by editing the duplicate
  agents, then the apply is re-run.
- **LOW#4 (write-time re-read guard over-engineering):** removed. With consumers
  stopped there is no concurrency to guard; resume safety rests on
  NormalizeTarget idempotency + "only write changed agents".
- **LOW#5 (Redis cachehandler wiring not mentioned):** §3.3 now states the
  subcommand builds the Redis `cachehandler` (existing `initCache` pattern) so
  `AgentSetAddresses` cache invalidation runs; runbook needs Redis reachability.
- LOW#6 (global collision map memory): confirmed non-issue at current scale.

This is the second consecutive material change, so a fresh re-review round on v5
follows (mandatory).
