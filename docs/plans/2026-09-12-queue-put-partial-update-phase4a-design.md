# Phase 4a: PUT /queues/{id} partial-update fix

Status: DRAFT (Round 0)
Author: CPO, per roadmap `docs/plans/2026-09-12-put-partial-update-migration-roadmap.md`
Phase: 4 of 8 (§4, roadmap row #7, rated **MEDIUM** risk) — queue half of the
two-service Phase 4 split (queues + providers land as two separate PRs per
the roadmap's one-PR-per-repo-boundary rule; this doc covers `bin-queue-manager`
only, see the sibling doc `2026-09-12-provider-put-partial-update-phase4b-design.md`
for `bin-route-manager`'s `providers`).
Precedent: PR #1291 (`mcpservers`), PR #1293 (Phase 1, `extensions`/`trunks`),
PR #1295 (Phase 2, `customer`/`customers`), PR #1298 (Phase 3, `routes`) —
this doc applies the same established pattern (roadmap §3), it does not
re-derive it.

## 1. Problem statement

`PUT /queues/{id}` requires all 7 body fields (`name`, `detail`,
`routing_method`, `tag_ids`, `wait_flow_id`, `wait_timeout`,
`service_timeout`). A client that omits any field does not get a
validation error — every field is a plain (non-pointer)
`string`/`RoutingMethod`/`[]uuid.UUID`/`uuid.UUID`/`int` value from the
OpenAPI-generated JSON body all the way down to
`bin-queue-manager/pkg/queuehandler/db.go`'s `UpdateBasicInfo`, which
**unconditionally** builds the full 7-key SQL field map (`db.go:94-102`)
regardless of what the caller actually sent.

The highest-risk fields per roadmap §2 row #7 are `wait_flow_id` and
`tag_ids`: an omitted `wait_flow_id` silently rewrites the queue's wait
flow to `uuid.Nil`, breaking the caller wait experience (per
`bin-queue-manager/CLAUDE.md`, `wait_flow_id` is executed while a caller
is on hold — a broken/missing flow means the caller hears silence or an
error instead of hold music/IVR). An omitted `tag_ids` silently wipes the
queue's agent-routing filter to an empty array, which — per this
service's own CRITICAL rule ("Tag Matching is Exact Overlap"; an agent
with no tags is never routable, and a queue with no `tag_ids` therefore
matches **zero** agents) — means every subsequent call into that queue
goes permanently unrouted (abandoned on timeout) until an operator
notices and re-sets `tag_ids`. Both are silent, `200 OK` failures with
customer-facing impact, same shape as Phase 1-3's already-fixed bugs.

The dbhandler layer is already safe — `QueueUpdate`
(`dbhandler/queue.go:236-239`) already has the `len(fields) == 0`
short-circuit and `squirrel.SetMap`-based partial-update mechanism (same
shape as Phase 2/3's dbhandler layers). The caller, `queueHandler.UpdateBasicInfo`,
is the layer that behaves differently: it never leaves `fields` empty or
partial today. This phase's fix site is `queueHandler.UpdateBasicInfo`,
matching Phase 2/3's shape (fix site one layer above dbhandler, not at
the listenhandler).

## 2. Full call chain

```
PUT /queues/{id}
  bin-api-manager/server/queues.go: PutQueuesId
        |
  bin-api-manager/pkg/servicehandler/queue.go: QueueUpdate
        |
  bin-common-handler/pkg/requesthandler/queue_queue.go: QueueV1QueueUpdate
        |
  bin-queue-manager/pkg/listenhandler/
    models/request/queues.go: V1DataQueuesIDPut (verify exact name at impl time)
    v1_queues.go: v1QueuesIDPut (line 219 call site confirmed)
        |
  bin-queue-manager/pkg/queuehandler/db.go: UpdateBasicInfo   <-- ACTUAL FIX SITE
    (builds fields map, currently unconditional, db.go:75-117)
        |
  bin-queue-manager/pkg/dbhandler/queue.go: QueueUpdate   <-- already safe,
                                                                no change needed
    (len(fields)==0 guard + squirrel SetMap, confirmed at queue.go:236-239)
```

Confirmed via direct code read: `queueHandler.UpdateBasicInfo`'s signature
is `UpdateBasicInfo(ctx, id, name string, detail string, routingMethod
queue.RoutingMethod, tagIDs []uuid.UUID, waitFlowID uuid.UUID, waitTimeout
int, serviceTimeout int)` (`db.go:75-85`) and unconditionally builds
`fields := map[queue.Field]any{queue.FieldName: name, ...}`
(`db.go:94-102`) before calling `h.db.QueueUpdate`.

**Note on parameter naming vs. OpenAPI field naming**: the OpenAPI schema
and this doc use `wait_timeout`/`service_timeout`; the Go function
parameter is named `waitTimeout` (not `timeoutWait`) at the
`queuehandler`/dbhandler layers but IS named `timeoutWait` at the
`bin-api-manager/pkg/servicehandler/queue.go` layer (`QueueUpdate`'s
6th param, confirmed at `queue.go:218`) — this pre-existing naming
inconsistency across layers is NOT introduced or fixed by this phase;
preserve each layer's existing parameter name when converting to a
pointer (do not rename while also changing type, to keep the diff
reviewable and avoid conflating two unrelated changes).

## 3. In scope

Per roadmap §3's established pattern, applied to this chain:

1. **OpenAPI schema** — `bin-openapi-manager/openapi/paths/queues/id.yaml`:
   remove the `required:` block (all 7 fields become optional); update
   each field's `description` to state the omitted-means-unchanged
   contract, plus a top-level schema `description`. `tag_ids`'s
   description should note that an explicit empty array (`[]`) clears all
   tags (a real, meaningful, if operationally dangerous per §1's CRITICAL
   rule, value) distinct from omitting the field; `wait_timeout`/
   `service_timeout` should note that `0` is a real, meaningful value
   (confirmed: `queuecallhandler`'s `Create` only schedules the
   self-timeout RPC `if tt.responseQueuecall.TimeoutWait > 0`
   (`db_test.go:332`), meaning `0`/negative effectively means "no
   auto-timeout" — a legitimate, if unusual, operational choice) distinct
   from omission, mirroring Phase 3's `priority: 0` precedent, now
   confirmed rather than assumed for this service's domain.
2. **`bin-api-manager/server/queues.go`** (`PutQueuesId`): stop
   unconditionally dereferencing OpenAPI-generated fields to plain values;
   pass the generated pointers straight through once the schema is
   optional. `tag_ids` is `[]string` today (array of ID strings) — becomes
   `*[]string`, requiring parse-each-element-to-uuid.UUID logic
   conditional on the pointer being non-nil, and (mirroring Phase 3 §3.1's
   `provider_id` malformed-input discipline) a malformed tag ID string
   inside a present `tag_ids` array must be rejected with 400, not
   silently dropped or zero-valued. `wait_flow_id` similarly needs the
   3-state (omitted/valid/malformed) treatment Phase 3 established for
   `provider_id`.
3. **`bin-api-manager/pkg/servicehandler/queue.go`**: `QueueUpdate`
   signature changes to accept pointers for all 7 mutable fields.
4. **`bin-api-manager/pkg/servicehandler/main.go`**: `ServiceHandler`
   interface declaration updated; `mock_main.go` regenerated.
5. **`bin-common-handler/pkg/requesthandler/queue_queue.go`**:
   `QueueV1QueueUpdate` signature changes to pointers; marshals into the
   wire DTO as pointers too.
6. **`bin-common-handler/pkg/requesthandler/main.go`**: interface
   declaration updated; `mock_main.go` regenerated.
7. **`bin-queue-manager/pkg/listenhandler/models/request/queues.go`**:
   the PUT wire DTO's fields become pointers with `omitempty` JSON tags.
8. **`bin-queue-manager/pkg/listenhandler/v1_queues.go`**
   (`v1QueuesIDPut`, confirmed call site at line 219): pass pointers
   through to `queueHandler.UpdateBasicInfo` unchanged in shape.
9. **`bin-queue-manager/pkg/queuehandler/db.go`** (`UpdateBasicInfo`) —
   **THE ACTUAL FIX SITE**: signature changes from 7 plain values to 7
   pointers; the `fields := map[queue.Field]any{...}` literal
   (`db.go:94-102`, currently unconditional) becomes built conditionally.
   `len(fields) == 0` short-circuits to `h.Get(ctx, id)` (confirmed at
   `db.go:31-49`: `queueHandler` already has its own `Get` wrapper that
   translates `dbhandler.ErrNotFound` to `cerrors.NotFound` — reuse it
   directly, matching Phase 1-3's discipline of not duplicating error
   translation).

   **`routing_method` validation asymmetry (confirmed, not deferred)**:
   `Create` validates `routing_method` (`create.go:69-79`: rejects
   anything other than `queue.RoutingMethodRandom` with
   `INVALID_ROUTING_METHOD`), but `UpdateBasicInfo` performs no equivalent
   check today and this phase does not add one — per roadmap §3 step 7,
   "any field-specific validation... becomes conditional on non-nil," and
   there is no existing validation on this field at this layer to make
   conditional. This is a pre-existing gap (Update accepts any
   `RoutingMethod` value without validation, unlike Create), not
   introduced or worsened by this fix, and is explicitly out of scope
   here (§4) — noted so a reviewer doesn't mistake the absence of a
   `routing_method` malformed-value 400 test (contrast with `wait_flow_id`/
   `tag_ids` elements, which DO get such tests per §5) for an oversight.
10. **`bin-queue-manager/pkg/queuehandler/main.go`** (interface file,
    exact name TBD, verify at implementation time): `QueueHandler`
    interface's `UpdateBasicInfo` declaration updated; `mock_main.go`
    regenerated.
11. **`bin-queue-manager/cmd/queue-control/main.go`** (`runQueueUpdate`,
    exact function name TBD verify at implementation time — CLI binary
    confirmed to exist at `cmd/queue-control/main.go`, NOT absent as an
    earlier draft of this doc incorrectly assumed): calls
    `handler.UpdateBasicInfo` directly with plain
    `viper.GetString`/`viper.GetInt` values today (confirmed call site:
    `name := viper.GetString("name")`, ...,
    `res, err := handler.UpdateBasicInfo(context.Background(), queueID,
    name, viper.GetString("detail"), queue.RoutingMethod(viper.GetString("routing-method")),
    tagIDs, waitFlowID, viper.GetInt("wait-timeout"),
    viper.GetInt("service-timeout"))`), after its own required-flag check
    (`name == ""` rejected). This call site must be updated to pass
    pointers to keep compiling. Mirrors Phase 2/3's CLI treatment
    exactly: wrap every value in a pointer unconditionally, preserving
    the CLI's exact current all-fields-sent behavior. No CLI UX change
    (no flag-changed detection) in this phase — out of scope, same
    rationale as Phase 2/3 (오버엔지니어링 지양, no speculative CLI UX
    work without a concrete request). Note also that `tagIDs` (parsed via
    a local `parseUUIDs` helper) and `waitFlowID` (parsed via
    `uuid.FromStringOrNil`, which silently produces `uuid.Nil` on a
    malformed flag value with no error) become pointers too; unlike the
    HTTP layer (§3 step 2), this phase does NOT add malformed-input
    rejection to the CLI's existing `uuid.FromStringOrNil` usage — that
    is a pre-existing CLI behavior gap, not introduced by this fix, and
    is out of scope here (mirrors Phase 3's CLI-vs-HTTP-layer scoping
    distinction — the CLI keeps its current error-handling shape, only
    the pointer-wrapping changes).
12. **RST docs** (`bin-api-manager/docsdev/source/queue_struct_queue.rst`
    or equivalent — verify exact filename at implementation time): add an
    Implementation Hint note stating the new contract and specifically
    calling out `wait_flow_id`/`tag_ids`'s customer-facing risk (mirrors
    the risk callout already in roadmap row #7); clean Sphinx rebuild
    committed in the same commit.
13. **Tests** at every layer per roadmap §3 step 11 — see §5 below.

## 4. Out of scope

- **`PUT /queues/{id}/routing_method`** and **`PUT /queues/{id}/tag_ids`**
  — single-field PUTs, already exempt per roadmap §3.2 (rows #33/#34). Not
  touched by this phase.
- **`PUT /providers/{id}`** — separate endpoint, separate service
  (`bin-route-manager`), lands as its own PR per the roadmap's
  one-PR-per-repo-boundary rule — see sibling design doc
  `2026-09-12-provider-put-partial-update-phase4b-design.md`.
- **`square-admin` frontend** — not verified in this repo (separate repo),
  same caveat as every prior phase: flagged, low-risk residual.
- **`POST /queues`** (create) — roadmap out of scope generally, same
  reasoning as every other phase's Create exclusion.
- **Queue/conference/tag domain validation beyond presence** (e.g. does
  `wait_flow_id` reference an existing flow, do `tag_ids` reference
  existing tags) — if such validation exists today in
  `queueHandler.UpdateBasicInfo`, this phase preserves it unchanged, only
  gating it on the relevant pointer being non-nil.

## 5. Testing strategy

Mirrors Phase 1-3, adapted to this call chain's fix site
(`queueHandler.UpdateBasicInfo`, matching Phase 2/3's shape) and its
`[]uuid.UUID` slice field (a new type shape not previously exercised —
Phase 1's `AuthTypes` was `*[]sipauth.AuthType`, the closest precedent):

- **`bin-queue-manager/pkg/queuehandler/db_test.go`**: table-driven cases
  for `UpdateBasicInfo` — one field set / rest nil (assert exact `fields`
  map contents per field); all-omitted no-op case (assert
  `dbhandler.QueueUpdate` is NOT called, `Times(0)`); - a `wait_timeout: intPtr(0)` / `service_timeout: intPtr(0)` case distinct
  from `nil`, confirming `0` (a legitimate "no auto-timeout" value, per
  §3 step 1's `TimeoutWait > 0` finding) is treated as a real, set value
  rather than omission;
  `tag_ids: &[]uuid.UUID{}` (pointer to an explicit empty slice) case
  distinct from `tag_ids: nil`, confirming the explicit-clear-vs-omit
  distinction holds for a slice field, not just scalar/enum fields
  (this is the single most operationally dangerous case per §1 — an
  explicit empty `tag_ids` genuinely means "no agents can serve this
  queue," and the design's job is to make sure that's a deliberate act,
  not an accident of omission).
- **`bin-queue-manager/pkg/listenhandler/v1_queues_test.go`**:
  extend/add a table test confirming `v1QueuesIDPut` passes pointers
  through to `UpdateBasicInfo` unchanged (thin pass-through).
- **`bin-api-manager/pkg/servicehandler/queue_test.go`**: update existing
  `QueueUpdate` mock call sites for the new pointer signature; add at
  minimum one new case with a partial body.
- **`bin-common-handler/pkg/requesthandler/queue_queue_test.go`**:
  mechanical signature update.
- **`bin-api-manager/server/queues_test.go`** (HTTP layer): add a
  name-only PUT test; add a dedicated malformed-`wait_flow_id` 400 test
  and a malformed-element-inside-`tag_ids` 400 test (mirrors Phase 3
  §3.1's `provider_id` treatment, extended to a slice field).
- **CLI test file** (if one exists for the queue-control-equivalent
  binary — verify at implementation time): mechanical signature update
  only, no new cases, mirroring Phase 2/3.
- **Revert-and-rerun requirement**: for at least the `tag_ids: nil` case
  (the single highest-risk field per §1 — a silently-emptied `tag_ids`
  has no error signal anywhere in the stack today, making it the purest
  instance of the bug class this migration exists to fix), confirm the
  new test genuinely FAILS against pre-fix `db.go` (git stash the fix,
  rerun, confirm FAIL, restore) before considering the phase complete.
- Full verification workflow per phase exit criteria (roadmap §5):
  `go mod tidy && go mod vendor && go generate ./... && go test ./... &&
  golangci-lint run` in every touched service (`bin-queue-manager`,
  `bin-common-handler`, `bin-api-manager`).

## 6. Non-goals

- No retroactive remediation for queue rows whose `wait_flow_id`/`tag_ids`
  may have already been silently wiped in production by the pre-fix
  behavior (mirrors PR #1291 §8 and the roadmap §6).
- No `square-admin` frontend changes bundled into this PR (roadmap §6).
- No new PATCH method / HTTP verb redesign (roadmap §6).
- No changes to `PUT /providers/{id}` (separate PR, sibling doc).
- No new domain-level validation beyond what already exists in
  `queueHandler.UpdateBasicInfo`/`Create` (§4 above).

## 7. Round 1 review disposition

Independent review verdict: REQUEST CHANGES. Findings and fixes:

| # | Finding | Severity | Fix location |
|---|---|---|---|
| 1 | §3 step 9's no-op short-circuit instructed calling `h.db.QueueGet` directly instead of the handler's own `Get` wrapper, risking a dropped `ErrNotFound`→`cerrors.NotFound` translation | IMPORTANT | §3 step 9 corrected to `h.Get(ctx, id)`, confirmed via `db.go:31-49` that this wrapper exists and does the translation |
| 2 | The doc flagged `tag_ids`/`wait_flow_id` for 3-state malformed-input handling but never addressed why `routing_method` doesn't get equivalent treatment, despite `Create` validating it and `Update` not | IMPORTANT | §3 step 9 extended with an explicit "routing_method validation asymmetry" note: confirmed `Create` validates (`create.go:69-79`), `Update` does not and this phase does not add it (pre-existing gap, not in scope) — documented so it isn't mistaken for an oversight |
| 3 | `wait_timeout`/`service_timeout` zero-value meaningfulness was deferred to "implementation time" without settling it now | MODERATE | §3 step 1 and §5 updated with a confirmed finding: `queuecallhandler`'s self-timeout scheduling only fires `if TimeoutWait > 0` (`db_test.go:332`), so `0`/negative is a legitimate "no auto-timeout" value, mirroring Phase 3's `priority: 0` |
| 4 | §3 step 11 (CLI call site) was left as an open TBD despite the reviewer's own investigation showing it was independently confirmable — and the reviewer's own "no CLI exists" claim in their write-up was itself wrong (a subsequent direct check found `cmd/queue-control/main.go` DOES exist and DOES call `UpdateBasicInfo`) | IMPORTANT (doc gap + reviewer error caught by re-verification) | §3 step 11 rewritten with the confirmed real call site (`cmd/queue-control/main.go`'s `handler.UpdateBasicInfo(...)`), the exact current signature, and the required pointer-wrapping fix, mirroring Phase 2/3's CLI treatment |
| 5, 6 | Minor: CLI's local `name`-required gate vs. API-level optionality wanted a clarifying note; a call-chain filename citation was flagged as possibly wrong | LOW | Re-verified §2 and §3's file citations (`bin-api-manager/server/queues.go` vs `bin-queue-manager/pkg/listenhandler/v1_queues.go`) against actual paths — both correct as written, no change needed beyond what findings 1-4 already fixed |

Not changed (reviewer confirmed correct as originally written): §1's core
problem statement (`wait_flow_id`/`tag_ids` silent-wipe risk,
CRITICAL-rule-grounded blast-radius reasoning); §2's call chain structure;
dbhandler-already-safe conclusion; §4/§6's scope boundaries.

**Note on finding #4's own review process**: the Round 1 reviewer's
initial claim that `bin-queue-manager` has no CLI binary was itself
incorrect and was caught by direct re-verification
(`search_files(target='files')` on `bin-queue-manager/cmd/` found
`cmd/queue-control/main.go`) before trusting it into this doc — a
reminder that even review findings must be independently re-checked
against the actual filesystem, not accepted at face value, consistent
with this monorepo's standing review-loop discipline.

## 8. Round 3 review disposition

Independent review verdict: APPROVE.

Re-verified from source, independently of Round 1/2's own narration
(including the standing discipline that a prior round's "confirmed via
X" claim must be re-checked, not trusted):

- `bin-queue-manager/pkg/queuehandler/db.go:75-117` — `UpdateBasicInfo`'s
  current 7-plain-value signature and unconditional `fields` map literal
  match §1/§2/§3 step 9 exactly, byte for byte against current HEAD.
- `bin-queue-manager/pkg/queuehandler/db.go:32-49` — `Get` wrapper's
  `dbhandler.ErrNotFound` → `cerrors.NotFound` translation confirmed
  present, matching §3 step 9's no-op short-circuit instruction (Round 1
  finding #1's fix holds).
- `bin-queue-manager/pkg/dbhandler/queue.go:236-249` — `QueueUpdate`'s
  `len(fields)==0` guard and `SetMap`-based partial update confirmed
  unchanged and safe, matching §1's "dbhandler already safe" claim.
- `bin-queue-manager/pkg/listenhandler/v1_queues.go:216-227` and
  `models/request/v1_queues.go`'s `V1DataQueuesIDPut` — confirmed current
  plain-value pass-through shape matches §3 steps 7-8's before-state
  description (note: the doc's §2 cites the request DTO file as
  `models/request/queues.go`; the actual file is
  `models/request/v1_queues.go` — a filename mismatch, but harmless: §2
  itself already hedges with "(verify exact name at impl time)", and no
  other section depends on the exact filename for correctness, only on
  the struct name `V1DataQueuesIDPut`, which is correct. Flagged as a
  cosmetic NIT, not a blocking finding, since the doc's own caveat already
  covers it.)
- `bin-api-manager/server/queues.go:173-217` (`PutQueuesId`) — confirmed
  current unconditional dereference of `req.Name`/`req.TagIds`/
  `req.WaitFlowId`/etc. into plain values, matching §3 step 2's
  before-state.
- `bin-api-manager/pkg/servicehandler/queue.go:209-220` — `QueueUpdate`'s
  parameter named `timeoutWait` (not `waitTimeout`) reconfirmed, matching
  §2's naming-inconsistency note precisely.
- `bin-queue-manager/pkg/queuehandler/create.go:69-79` — `routing_method`
  validation confirmed present in `Create`, confirmed absent from
  `UpdateBasicInfo`, matching §3 step 9's asymmetry note (Round 1 finding
  #2's fix holds).
- `bin-queue-manager/pkg/queuecallhandler/db_test.go:332-333` —
  `TimeoutWait > 0` gate on `QueueV1QueuecallTimeoutWait` reconfirmed,
  matching §3 step 1/§5's zero-value reasoning (Round 1 finding #3's fix
  holds).
- `bin-queue-manager/cmd/queue-control/main.go:303-357` (`runUpdate`) —
  confirmed real call site, calls `handler.UpdateBasicInfo` with plain
  `viper.GetString`/`viper.GetInt` values as §3 step 11 describes,
  reconfirming Round 1 finding #4's correction (and reconfirming, per bug
  class 14's mitigation, that the CLI binary genuinely exists — did not
  trust the doc's own prior "confirmed" narration, re-ran the filesystem
  check directly).
- `bin-openapi-manager/openapi/paths/queues/id.yaml` — file exists,
  `put:` block present, matching §3 step 1's target path.
- Roadmap cross-references: row #7 (`docs/plans/2026-09-12-put-partial-update-migration-roadmap.md:64`)
  confirms MEDIUM risk tier and the same 7-field list; rows #33/#34
  confirm the single-field `routing_method`/`tag_ids` PUT exemptions
  cited in §4. Both match the doc's claims exactly.
- Precedent PRs #1291, #1293, #1295, #1298 confirmed MERGED via `gh pr
  view`, supporting the doc's "established pattern, not re-derived"
  framing.
- Internal consistency: read §1 through §7 end-to-end against each other
  (not just against source) — Goals/problem-statement, in-scope,
  out-of-scope, testing, and non-goals sections agree with each other and
  with the Round 1/Round 2 disposition tables; no contradiction found
  (bug class 4/8 pattern checked and not present).

No new findings. The one NIT identified (request DTO filename citation in
§2) is non-blocking, already self-hedged by the doc, and does not affect
implementation correctness since the struct name is correct and is what
implementers actually need.
