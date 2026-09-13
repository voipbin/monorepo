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
| `on_complete_flow_id` | already optional | already optional (no change to its optionality; threading fixed if needed) |

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
    → dereferences req.Name/req.Detail to plain string  ← BUG SITE #1 (server)
  bin-api-manager/pkg/servicehandler/flow.go FlowUpdate (line 204)
    → takes name,detail string; actions []Action; onCompleteID uuid.UUID
  bin-common-handler/pkg/requesthandler/flow_flow.go FlowV1FlowUpdate (line 108)
    → marshals into V1DataFlowsIDPut (wire DTO)
  bin-flow-manager/pkg/listenhandler/v1_flows.go v1FlowsIDPut (line 72)
    → unmarshals request.V1DataFlowsIDPut, calls flowHandler.Update
  bin-flow-manager/pkg/flowhandler/db.go     flowHandler.Update (line 198)  ← FIX SITE
    → unconditionally builds fields{Name,Detail,Actions,OnCompleteFlowID}
      and calls db.FlowUpdate  ← BUG SITE #2 (silent full-overwrite)
```

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

Because `server/flows.go` collapses an absent `name`/`detail` pointer to `""`
(and an absent `on_complete_flow_id` to `uuid.Nil`), a client that PUTs only
`{"name": "...", "actions": [...]}` today **silently overwrites `detail` with
`""`** and clears `on_complete_flow_id`. This is the exact PR #1291 bug class.

## 3. Fix (per-layer)

Apply the canonical pattern (roadmap §3), hybrid variant:

1. **OpenAPI** (`bin-openapi-manager/openapi/paths/flows/id.yaml`): in the
   PUT `required:` block, remove `name` and `detail`; **keep `actions`**.
   (`on_complete_flow_id` is already not in `required`.) Update `name`/`detail`
   descriptions to "Omit to leave unchanged." Leave `actions` description as-is
   (still required). Regenerate: `PutFlowsIdJSONBody` will keep `Actions` as a
   required field and `Name`/`Detail` as `*string` (they already generate as
   pointers today since oapi-codegen makes non-`required` object props
   pointers; verify after `go generate`).
2. **`bin-api-manager/server/flows.go` `PutFlowsId`**: pass `req.Name` /
   `req.Detail` through as `*string` (stop the `if req.X != nil { x = *req.X }`
   collapse). `actions` stays a required slice (validate present/non-nil →
   400 `INVALID_ACTIONS` if absent, mirroring the required-field contract).
   `on_complete_flow_id` continues to thread through as a pointer.
3. **`bin-api-manager/pkg/servicehandler/flow.go` `FlowUpdate`**: accept
   `name *string`, `detail *string`, keep `actions []Action` (required),
   `onCompleteFlowID *uuid.UUID`.
4. **`bin-common-handler/pkg/requesthandler/flow_flow.go` `FlowV1FlowUpdate`**:
   accept pointers for name/detail/onCompleteFlowID, marshal into the wire DTO
   as pointers (mandatory — §4.4 of the mcpservers design: skipping this
   reintroduces the bug one RabbitMQ hop later).
5. **`bin-flow-manager/pkg/listenhandler/models/request/flows.go`
   `V1DataFlowsIDPut`**: `Name`/`Detail` become `*string` with `omitempty`;
   `OnCompleteFlowID` becomes `*uuid.UUID` (if not already); `Actions` stays a
   non-pointer required slice.
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
7. **New**: a PUT body missing `actions` → 400 (asserts `servicehandler.FlowUpdate`
   is never reached), pinning the §3.1 exemption as an intentional contract.
   (Confirm the exact error code the server returns for a missing required field;
   if the generated binder does not auto-reject, add an explicit `if req.Actions
   == nil` guard returning `INVALID_ACTIONS`.)

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

(Filled in during the design review loop.)
