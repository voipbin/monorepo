# Phase 7a Design: `PUT /flows/{id}` partial-update migration (hybrid)

**Status:** design review
**Roadmap:** `docs/plans/2026-09-12-put-partial-update-migration-roadmap.md` §2 row 14, §3.1, §4 Phase 7
**Precedent:** PR #1291 (`PUT /mcpservers/:id`, commit `81a9a6b98`); Phases 1-6 (merged).
**Scope:** ONE PR, ONE repo boundary (bin-flow-manager + shared bin-api-manager / bin-common-handler / bin-openapi-manager codegen).

## 1. Goal

Migrate `PUT /flows/{id}` from strict-ish full-replace to partial-update
("omit to leave unchanged") semantics for the fields that are plausible to
omit, **while deliberately keeping `actions` required** (the one field-level
exemption in roadmap §3.1).

This is a **hybrid** migration:

| Field | Today | After this PR |
|---|---|---|
| `name` | required | **optional** (nil-means-unchanged) |
| `detail` | required | **optional** (nil-means-unchanged) |
| `actions` | required | **stays required** (§3.1 — see rationale below) |
| `on_complete_flow_id` | OpenAPI-optional, but silently cleared to `uuid.Nil` at runtime | **optional AND nil-safe** — requires threading `*uuid.UUID` through 4 lower layers (wire DTO, requesthandler, servicehandler, flowHandler.Update) plus a conditional field-map entry |

### Why `actions` stays required (roadmap §3.1, verbatim rationale)

A flow's action list is edited as a whole document by the flow editor
(add/remove/reorder is a client-side operation on the full array, then the
full array is saved). There is no natural "omit `actions` to leave it
unchanged" use case for a PUT that also changes `name`/`detail`: the caller
is either renaming a flow (in which case sending the current `actions` array
back is not onerous — it's already loaded in the editor) or is editing
actions (in which case there is nothing to omit). Converting `actions` to
nil-means-unchanged would only invite an accidental clear if a client's array
serialization ever produced `null`/empty. **Decision: `actions` remains
required; a PUT with `actions` absent/null is a 400, not a no-op.**

## 2. Call chain (verified against source at branch base 5b3821994)

```
PUT /flows/{id}
  bin-api-manager/server/flows.go            PutFlowsId (line 133)
    → today passes req.Name/req.Detail directly as plain `string`, and
      collapses an absent on_complete_flow_id pointer to uuid.Nil  ← BUG SITE #1
  bin-api-manager/pkg/servicehandler/flow.go FlowUpdate (line 204)
    → takes name,detail string; actions []Action; onCompleteID uuid.UUID
  bin-common-handler/pkg/requesthandler/flow_flow.go FlowV1FlowUpdate (line 108)
    → marshals into V1DataFlowsIDPut (wire DTO)
  bin-flow-manager/pkg/listenhandler/v1_flows.go v1FlowsIDPut (line 57)
    → unmarshals request.V1DataFlowsIDPut (line 72), calls flowHandler.Update
  bin-flow-manager/pkg/flowhandler/db.go     flowHandler.Update (line 198)  ← FIX SITE
    → unconditionally builds fields{Name,Detail,Actions,OnCompleteFlowID}
      and calls db.FlowUpdate  ← BUG SITE #2 (silent full-overwrite)
```

**Current field types (Round 1 correction):** at branch base, the generated
`PutFlowsIdJSONBody.Name`/`.Detail` are **plain `string`** (they are still in
the OpenAPI `required` block, so oapi-codegen does NOT emit pointers today);
`OnCompleteFlowID` already generates as `*string` (it is not required). The
wire DTO `V1DataFlowsIDPut.OnCompleteFlowID`, `requesthandler.FlowV1FlowUpdate`,
`servicehandler.FlowUpdate`, and `flowHandler.Update` all currently take a
non-pointer `uuid.UUID`. So today there is **no** `if req.X != nil { x = *req.X }`
collapse for name/detail to remove — they are simply passed as plain strings;
the pointer threading is introduced BY this migration (after the required-block
removal makes `Name`/`Detail` generate as `*string`).


`db.FlowUpdate(ctx, id, fields map[flow.Field]any)` already accepts a partial
field map, so the fix at the flowHandler layer is to build that map
conditionally — the same shape used in every prior phase.

### Current silent-wipe bug (confirmed)

`flowHandler.Update` (db.go:217-222) builds:

```go
fields := map[flow.Field]any{
    flow.FieldName:             name,
    flow.FieldDetail:           detail,
    flow.FieldActions:          tmpActions,
    flow.FieldOnCompleteFlowID: onCompleteFlowID,
}
```

Because `server/flows.go` passes an omitted `name`/`detail` as the empty
string `""` (they are plain-string fields today, not pointers) and collapses an
absent `on_complete_flow_id` pointer to `uuid.Nil`, a client that PUTs only
`{"name": "...", "actions": [...]}` today **silently overwrites `detail` with
`""`** and clears `on_complete_flow_id`. This is the exact PR #1291 bug class.
The fix makes name/detail nil-distinguishable (pointers, post-codegen) and
threads `on_complete_flow_id` as `*uuid.UUID` end to end.

## 3. Fix (per-layer)

Apply the canonical pattern (roadmap §3), hybrid variant:

1. **OpenAPI** (`bin-openapi-manager/openapi/paths/flows/id.yaml`): in the
   PUT `required:` block, remove `name` and `detail`; **keep `actions`**.
   (`on_complete_flow_id` is already not in `required`.) Update `name`/`detail`
   descriptions to "Omit to leave unchanged." Leave `actions` description as-is
   (still required). After `go generate`, `PutFlowsIdJSONBody.Actions` stays a
   plain required slice while `Name`/`Detail` **newly** become `*string`
   (they are plain `string` today because they are still required — the pointer
   emission is a RESULT of this required-block removal, not a pre-existing
   state). Verify the regenerated types after `go generate`.
2. **`bin-api-manager/server/flows.go` `PutFlowsId`**: pass the now-`*string`
   `req.Name` / `req.Detail` through unchanged (there is no existing
   `if req.X != nil { x = *req.X }` collapse to remove — today they are plain
   strings passed directly; post-codegen they are pointers passed directly).
   **MANDATORY guard for `actions`** (linchpin of the hybrid design): the
   generated binder does NOT reject a missing/null `actions` (the field carries
   no `binding:"required"` tag, and `ServerInterfaceWrapper.PutFlowsId`
   validates only the path param), so `req.Actions` would silently arrive as a
   nil/empty slice and wipe the stored action list. Add an explicit
   `if len(req.Actions) == 0 { abort 400 INVALID_ACTIONS }` guard before calling
   the servicehandler. This guard is REQUIRED, not conditional — the entire
   "always-set actions, no `len(fields)==0` short-circuit" safety argument in
   step 7 depends on it. `on_complete_flow_id` threads through as `*uuid.UUID`
   (stop the collapse to `uuid.Nil`).
3. **`bin-api-manager/pkg/servicehandler/flow.go` `FlowUpdate`**: accept
   `name *string`, `detail *string`, keep `actions []Action` (required),
   `onCompleteFlowID *uuid.UUID`.
4. **`bin-common-handler/pkg/requesthandler/flow_flow.go` `FlowV1FlowUpdate`**:
   accept pointers for name/detail/onCompleteFlowID, marshal into the wire DTO
   as pointers (mandatory — §4.4 of the mcpservers design: skipping this
   reintroduces the bug one RabbitMQ hop later).
5. **`bin-flow-manager/pkg/listenhandler/models/request/flows.go`
   `V1DataFlowsIDPut`**: `Name`/`Detail` become `*string` with `omitempty`;
   `OnCompleteFlowID` becomes `*uuid.UUID` with `omitempty` (it is a non-pointer
   `uuid.UUID` today — this IS a required change, not "if not already");
   `Actions` stays a non-pointer required slice.
6. **`bin-flow-manager/pkg/listenhandler/v1_flows.go` `v1FlowsIDPut`**: pass
   the pointers through unchanged.
7. **`bin-flow-manager/pkg/flowhandler/db.go` `flowHandler.Update`** (fix site):
   build the fields map conditionally —
   - `if name != nil { fields[FieldName] = *name }`
   - `if detail != nil { fields[FieldDetail] = *detail }`
   - `if onCompleteFlowID != nil { fields[FieldOnCompleteFlowID] = *onCompleteFlowID }`
   - **`actions` is always set** (it is required; the handler always receives a
     non-nil slice — a nil/empty actions slice from a malformed request was
     already rejected upstream at the server layer). Call
     `GenerateFlowActions(actions)` and set `fields[FieldActions] = tmpActions`
     unconditionally, as today.

   No `len(fields)==0` short-circuit is needed here (unlike teams), because
   `actions` is always present, so `fields` is never empty — every valid
   `PUT /flows/{id}` always writes at least `actions`. (This is the key
   structural difference from the fully-optional phases and must be stated so a
   reviewer does not flag the missing short-circuit as an omission.)

### Non-goals (explicit)

- **`actions` optionality**: NOT migrated. `PUT /flows/{id}` with `actions`
  absent remains a 400. This is the §3.1 field-level exemption.
- **`PUT /flows/{id}/actions`** (the dedicated single-field actions endpoint,
  `FlowUpdateActions` / `flowHandler.UpdateActions`): untouched. It is a
  separate single-field PUT (roadmap §3.2 territory conceptually) and out of
  scope.
- **No new `db.FlowUpdate` guard**: `db.FlowUpdate` already applies only the
  keys present in the map; no dbhandler-level change.

## 4. Tests

Fix site (`bin-flow-manager/pkg/flowhandler`):
1. `name` only + `actions` → asserts `db.FlowUpdate` called with a fields map
   containing `FieldName` and `FieldActions` but **NOT** `FieldDetail` /
   `FieldOnCompleteFlowID`.
2. `detail` only + `actions` → symmetric.
3. all optional fields set + `actions` → all four keys present (regression of
   today's behavior).
4. `on_complete_flow_id` only + `actions` → `FieldOnCompleteFlowID` present,
   name/detail absent.
5. **No `actions` is NOT tested at the flowHandler layer** because the
   required-field 400 is enforced at the server layer; instead add a server-layer
   test (below) for that.

Server layer (`bin-api-manager/server/flows_test.go`):
6. `Test_flowsIDPut` rewritten with pointer helpers; assert a partial body
   (name only, with actions) reaches `servicehandler.FlowUpdate` with a nil
   detail pointer.
7. **New (mandatory)**: a PUT body missing/empty `actions` → 400
   `INVALID_ACTIONS` (asserts `servicehandler.FlowUpdate` is never reached),
   pinning the §3.1 exemption as an intentional contract. The generated binder
   does NOT auto-reject a missing `actions` (no `binding:"required"` tag), so
   the explicit `if len(req.Actions) == 0` guard from §3 step 2 is REQUIRED and
   this test lands together with that guard.

Every other touched layer (servicehandler, requesthandler, listenhandler) gets
its existing `Test_*Update` updated to the pointer signatures.

### Revert-and-rerun proof (mandatory)

Temporarily restore the unconditional field-map construction in
`flowHandler.Update` and confirm test #1/#2 FAIL (detail/on_complete wiped),
then revert and confirm PASS.

## 5. Verification sequence (per touched service)

bin-flow-manager, bin-api-manager, bin-common-handler, bin-openapi-manager:
`go generate ./...` (openapi first) → `go build ./...` → mockgen regenerate
(`FlowUpdate`, `FlowV1FlowUpdate`, flowHandler.`Update`) → `go vet ./...` →
`go test ./...` → revert-and-rerun proof → `golangci-lint run --timeout 5m`
(0 issues) → `go mod tidy && go mod vendor` → rebuild+retest.

RST: `bin-api-manager/docsdev/source/flow_struct_flow.rst` (or equivalent)
gains an Implementation Hint documenting the hybrid contract: name/detail/
on_complete_flow_id are omit-to-leave-unchanged, **actions is always required
and always fully replaces the stored action list** (no partial action merge).
Clean Sphinx rebuild + `git add -f build/`.

## 6. Review disposition

### Round 1 (`deleg_9f261e02` task 1): REQUEST CHANGES

| # | Severity | Finding | Fix applied |
|---|---|---|---|
| 1 | HIGH | Call-chain narrative wrong: name/detail are plain `string` today (still required), not pointers; server does no deref/collapse for them. | §2 diagram + "Current field types" note + §1 table + §3 step 1/2 rewritten: name/detail become `*string` only AS A RESULT of the required-block removal; no existing collapse to remove. |
| 2 | HIGH | "Missing actions → 400" does not hold today (no `binding:"required"` tag; binder validates only path param) and was treated as conditional; the whole no-short-circuit safety argument depends on it. | §3 step 2 now mandates an explicit `if len(req.Actions) == 0 { 400 INVALID_ACTIONS }` server guard; §4 test #7 lands with that guard. |
| 3 | MEDIUM | `on_complete_flow_id` threading under-specified ("if needed"); actually requires `*uuid.UUID` at 4 lower layers today (all non-pointer, server collapses nil→uuid.Nil = real silent clear). | §1 table + §3 step 5 now state the 4-layer `*uuid.UUID` conversion as a definite requirement. |
| 4 | LOW | `v1FlowsIDPut` starts at line 57, not 72. | §2 diagram corrected (57 = func start, 72 = unmarshal). |

Not changed (reviewer confirmed correct): fix-site location + unconditional
field-map at db.go:217-222; `db.FlowUpdate` partial-map capability; the
actions-always-set / no-short-circuit decision (sound GIVEN finding #2's guard,
now mandatory); test placement at fix-site vs server layer.

