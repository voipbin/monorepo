# Phase 7b Design: `PUT /service_agents/me` partial-update migration

**Status:** design review
**Roadmap:** `docs/plans/2026-09-12-put-partial-update-migration-roadmap.md` §2 row 20, §4 Phase 7
**Precedent:** PR #1291 (`PUT /mcpservers/:id`); Phases 1-6 (merged).
**Scope:** ONE PR, ONE repo boundary (bin-agent-manager + shared bin-api-manager / bin-common-handler / bin-openapi-manager codegen).

## 1. Goal

Migrate `PUT /service_agents/me` from strict full-replace to partial-update
("omit to leave unchanged") semantics for `name`, `detail`, `ring_method`.

| Field | Today | After this PR |
|---|---|---|
| `name` | required | **optional** (nil-means-unchanged) |
| `detail` | required | **optional** (nil-means-unchanged) |
| `ring_method` | required | **optional** (nil-means-unchanged) |

`ring_method` is the highest-risk field: an agent who PUTs only a name change
would today silently reset their ring method to the zero value, changing how
their calls ring with no error (roadmap §2 row 20, MEDIUM).

## 2. CRITICAL scope finding: the fix site is SHARED with admin `PUT /agents/{id}`

This is the load-bearing design decision for this phase and must be reviewed
carefully. Verified against source at branch base 5b3821994:

Both endpoints converge on the **same** private helper, RPC method, wire DTO,
and fix site:

```
PUT /service_agents/me                    PUT /agents/{id}  (admin surface)
  server/service_agents_me.go               server/agents.go
    PutServiceAgentsMe                         PutAgentsId
    → passes req.Name/Detail/RingMethod        → collapses *req.Name → "" etc.  ← BUG (admin)
  servicehandler ServiceAgentMeUpdate       servicehandler AgentUpdate
    (agent.go, IsAgent check)                  (agent.go:215, admin/manager perm check)
         \                                    /
          → h.agentUpdate(ctx,id,name,detail,ringMethod)   ← SHARED private helper (agent.go:248)
             → reqHandler.AgentV1AgentUpdate(...)          ← SHARED RPC (flow_flow → agent_agents.go:285)
                → wire DTO V1DataAgentsIDPut               ← SHARED (agents.go request DTO)
                   → bin-agent-manager listenhandler v1_agents.go
                      → agentHandler.UpdateBasicInfo (agent.go:273)   ← SHARED FIX SITE
                         → dbUpdateInfo → db.AgentSetBasicInfo
                            → builds fields{Name,Detail,RingMethod} UNCONDITIONALLY  ← silent-wipe bug
                            → db.AgentUpdate(ctx,id,fields map)   ← already partial-capable
```

### Consequence

**It is impossible to fix `/service_agents/me` in isolation.** The moment we
change the shared `agentUpdate` helper, `AgentV1AgentUpdate` RPC signature, the
`V1DataAgentsIDPut` wire DTO, and the `agentHandler.UpdateBasicInfo`/
`dbUpdateInfo` fix site to pointer semantics, the admin `PUT /agents/{id}`
call path is necessarily carried along. Both `server` entrypoints
(`PutServiceAgentsMe` and `PutAgentsId`) must be updated in the same PR to
thread pointers instead of collapsing them to zero values.

### Is that in scope? Yes — and it's a bug fix, not scope creep

- The roadmap classified `PUT /agents/{id}` (admin) as "already partial"
  because its OpenAPI PUT body has **no `required:` block** (confirmed:
  `agents/id.yaml` PUT lists name/detail/ring_method with no `required`).
  It was therefore NOT given its own §2 row or phase.
- **But the roadmap's classification looked only at the OpenAPI layer.** At
  runtime, `server/agents.go PutAgentsId` still does
  `name := ""; if req.Name != nil { name = *req.Name }` and the shared fix site
  still writes all three fields unconditionally — so `PUT /agents/{id}` has the
  **exact same silent-wipe bug** as the strict endpoints, despite its
  "already partial" OpenAPI shape. Its optional OpenAPI fields are a false
  comfort: optional on the wire, mandatory-overwrite in the handler.
- Fixing the shared chain fixes both surfaces at once. This is the correct,
  minimal outcome: one shared code path, one fix, both entrypoints corrected.
  Splitting them would be impossible without duplicating the entire agent-update
  chain, which violates DRY and the shared-helper design.

**Decision: this PR migrates the shared agent-basic-info update chain, fixing
BOTH `PUT /service_agents/me` and `PUT /agents/{id}` (admin) to pointer/
nil-means-unchanged semantics. Both server entrypoints are updated. This is
reported explicitly to 대표님 as a scope observation (the roadmap undercounted
`/agents/{id}` as already-safe when it is not).**

This does NOT expand to a new PR per the one-PR-per-repo rule: `/agents/{id}`
and `/service_agents/me` both live in bin-agent-manager and share one code
path — this is one logical change to one repo boundary.

## 3. Call chain fix (per-layer)

`db.AgentUpdate(ctx, id, fields map[agent.Field]any)` already exists
(dbhandler/agent.go:506) and applies only the keys present, so the fix site is
`agentHandler.UpdateBasicInfo` / `dbUpdateInfo` building that map conditionally.

1. **OpenAPI**
   - `paths/service_agents/me.yaml`: remove the `required: [name, detail,
     ring_method]` block; add "Omit to leave unchanged." to each description.
   - `paths/agents/id.yaml`: already has no `required` block — no schema
     change needed there, but confirm `PutAgentsIdJSONBody` generates all three
     as pointers (it already does).
   - `ring_method` is a `$ref` to `AgentManagerAgentRingMethod`; confirm it
     generates as `*AgentManagerAgentRingMethod` when not required.
2. **`server/service_agents_me.go PutServiceAgentsMe`**: pass `req.Name`,
   `req.Detail`, `req.RingMethod` through as pointers (currently passes them but
   wraps `RingMethod(req.RingMethod)` — must handle the nil `*RingMethod` case).
3. **`server/agents.go PutAgentsId`**: stop collapsing `*req.Name → ""`,
   `*req.RingMethod → RingMethodRingAll`; pass pointers straight through.
4. **`servicehandler` `AgentUpdate` + `ServiceAgentMeUpdate`**: accept
   `name *string`, `detail *string`, `ringMethod *amagent.RingMethod`; the
   shared private helper `agentUpdate` accepts pointers and passes them to the
   RPC.
5. **`servicehandler/main.go` interface**: update both `AgentUpdate` and
   `ServiceAgentMeUpdate` signatures; regenerate mock.
6. **`bin-common-handler/pkg/requesthandler/agent_agents.go` `AgentV1AgentUpdate`**:
   accept pointers, marshal into `V1DataAgentsIDPut` as pointers.
7. **`bin-agent-manager/pkg/listenhandler/models/request/agents.go`
   `V1DataAgentsIDPut`**: `Name`/`Detail` → `*string`, `RingMethod` →
   `*string` (or `*RingMethod`), all `omitempty`.
8. **`bin-agent-manager/pkg/listenhandler/v1_agents.go`**: pass pointers through.
9. **`bin-agent-manager/pkg/agenthandler` fix site** (`UpdateBasicInfo` →
   `dbUpdateInfo`): accept pointers; build the `fields` map conditionally
   (`if name != nil { fields[FieldName] = *name }`, etc.); call
   `db.AgentUpdate(ctx, id, fields)` directly instead of
   `db.AgentSetBasicInfo` (which is the unconditional 3-field wrapper).
   - Add a `len(fields)==0` short-circuit → return `db.AgentGet(ctx, id)`
     (fresh read, no write, no event publish) for a true no-op PUT, matching
     the teams/campaigns/mcpservers precedent.
   - `db.AgentSetBasicInfo` becomes unused by this path; leave it in place
     (still unit-tested, may have other callers — grep confirms only this path
     + tests) — do NOT delete (avoid dead-code churn; matches Phase 6a's
     treatment of the bypassed `dbhandler` wrappers).

## 4. Tests

Fix site (`bin-agent-manager/pkg/agenthandler`):
1. `name` only → fields map has `FieldName` only; no `FieldDetail`/`FieldRingMethod`.
2. `ring_method` only → `FieldRingMethod` only.
3. all set → all three (regression of today).
4. true no-op (all nil) → `db.AgentUpdate` `.Times(0)`, `db.AgentGet` called once,
   no event published.

servicehandler:
5. `Test_AgentUpdate` (admin) and a `ServiceAgentMeUpdate` test updated to
   pointer signatures; assert partial pass-through.

server:
6. `Test_agentsIDPut` (admin) and `Test_mePUT` updated with pointer helpers;
   partial body reaches servicehandler with correct nil pointers.

requesthandler + listenhandler: existing `Test_*` updated to pointer signatures.

### Revert-and-rerun proof (mandatory)

Restore unconditional field-map construction in the fix site; confirm test
#1/#2 FAIL (detail/ring_method wiped on a partial update), then revert → PASS.

## 5. Verification sequence

bin-agent-manager, bin-api-manager, bin-common-handler, bin-openapi-manager:
`go generate ./...` (openapi first) → `go build ./...` → mockgen regenerate
(agentHandler, servicehandler, requesthandler) → `go vet` → `go test ./...` →
revert-and-rerun proof → `golangci-lint run --timeout 5m` (0 issues) →
`go mod tidy && go mod vendor` → rebuild+retest.

RST: `bin-api-manager/docsdev/source/` — the agent / service-agent struct doc
gains an Implementation Hint on the omit-to-leave-unchanged contract for
name/detail/ring_method (covering BOTH the admin and self-service surfaces,
since they share behavior). Clean Sphinx rebuild + `git add -f build/`.

## 6. Review disposition

(Filled in during the design review loop.)
