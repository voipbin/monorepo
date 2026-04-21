# Call Metadata `route_provider_ids` Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a general-purpose, internal-only metadata pass-through on `Call` creation, starting with the `route_provider_ids` key that lets internal processes override route-manager's dialroute selection with a specific ordered list of provider IDs.

**Architecture:** Extend the internal `CallV1CallsCreate` RPC with an optional `metadata map[string]interface{}` parameter. Call-manager persists it on `Call.Metadata`, then extracts `route_provider_ids` when requesting dialroutes. Route-manager's `DialrouteList` gains a new `targetProviderIDs []uuid.UUID` parameter; when non-empty, it returns synthetic routes in array order, bypassing normal route merging.

**Tech Stack:** Go 1.22+, RabbitMQ RPC (sock.Request/Response), gomock, MySQL (existing `metadata` JSON column), Redis cache.

**Design Document:** `docs/plans/2026-04-21-call-metadata-route-provider-ids-design.md`

**Scope:** This plan implements the metadata mechanism and `route_provider_ids` wiring through call-manager + route-manager + RPC layer. The admin-facing `POST /v1/providers/{id}/calls` endpoint and `ProviderCall` persistence belong to the separate `provider-test-call.prd.md` — out of scope here.

**Affected services (7):** `bin-common-handler`, `bin-route-manager`, `bin-call-manager`, `bin-flow-manager`, `bin-queue-manager`, `bin-api-manager`, `bin-campaign-manager` (all caller-signature updates only; no new customer-facing endpoint in this PR).

**Worktree:** `~/gitvoipbin/monorepo/.worktrees/NOJIRA-call-metadata-route-provider-ids`

---

## Conventions for every task

- **Verification workflow** after every commit touching a given service:
  ```bash
  cd bin-<service> && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
  ```
- Do **not** `git add vendor/` — it's gitignored. Dockerfiles regenerate it at build time.
- Commit messages follow the monorepo format (title = branch name, bullet list with project prefixes).
- Follow the `rtp_debug` precedent (`models/call/metadata.go`, `pkg/callhandler/start.go:656-668`) for all call-manager patterns.

---

## Phase 1 — bin-common-handler RPC layer

### Task 1: Extend `RouteV1DialrouteList` with `targetProviderIDs`

**Files:**
- Modify: `bin-common-handler/pkg/requesthandler/route_dialroutes.go`
- Test: `bin-common-handler/pkg/requesthandler/route_dialroutes_test.go`

**Step 1: Update the signature + write the failing test**

Add a new parameter `targetProviderIDs []uuid.UUID` to `RouteV1DialrouteList`. The parameter is serialized into the existing request body (not as filter). Add a new test case in `route_dialroutes_test.go` that asserts a request with `targetProviderIDs` set includes `{"target_provider_ids":["..."]}` in the request data.

```go
// route_dialroutes.go
func (r *requestHandler) RouteV1DialrouteList(
    ctx context.Context,
    filters map[rmroute.Field]any,
    targetProviderIDs []uuid.UUID,
) ([]rmroute.Route, error) {
    uri := "/v1/dialroutes"

    m, err := json.Marshal(rmrequest.V1DataDialroutesGet{
        Filters:           filters,
        TargetProviderIDs: targetProviderIDs,
    })
    // ... rest unchanged
}
```

You'll need to introduce `V1DataDialroutesGet` in `bin-route-manager/pkg/listenhandler/models/request/dialroute.go` (see Task 5). For now, stub the struct in this package so the test compiles, then replace the import in Task 5.

**Step 2: Run test to verify it fails**

```bash
cd bin-common-handler
go test ./pkg/requesthandler -run TestRouteV1DialrouteList -v
# Expected: FAIL (new assertion or compile error on missing field)
```

**Step 3: Implement the signature change**

Marshal `targetProviderIDs` into the request body when non-empty; omit when nil/empty for backward-compatible wire format.

**Step 4: Run test to verify it passes**

```bash
go test ./pkg/requesthandler -run TestRouteV1DialrouteList -v
# Expected: PASS
```

**Step 5: Commit (do NOT run full verification yet — Phase 1 finishes together)**

```bash
git add bin-common-handler/pkg/requesthandler/route_dialroutes.go bin-common-handler/pkg/requesthandler/route_dialroutes_test.go
git commit -m "NOJIRA-call-metadata-route-provider-ids

- bin-common-handler: Add targetProviderIDs param to RouteV1DialrouteList"
```

---

### Task 2: Extend `CallV1CallsCreate` **and** `CallV1CallCreateWithID` with `metadata`

**Files:**
- Modify: `bin-common-handler/pkg/requesthandler/call_calls.go:110-148` (`CallV1CallsCreate`)
- Modify: `bin-common-handler/pkg/requesthandler/call_calls.go:153-189` (`CallV1CallCreateWithID`)
- Modify: `bin-common-handler/pkg/requesthandler/main.go` (both interface signatures)
- Test: `bin-common-handler/pkg/requesthandler/call_calls_test.go:291`
- Modify: `bin-call-manager/pkg/listenhandler/models/request/calls.go` (add `Metadata` field to both `V1DataCallsPost` and `V1DataCallsIDPost`)

**Parity requirement:** Both RPCs must accept `metadata` so groupcall fan-out (`bin-call-manager/pkg/groupcallhandler`) and campaign calls (`bin-campaign-manager`) can carry metadata identically. v1 callers all pass `nil`.

**Step 1: Update request struct first**

```go
// bin-call-manager/pkg/listenhandler/models/request/calls.go
type V1DataCallsPost struct {
    FlowID         uuid.UUID               `json:"flow_id,omitempty"`
    CustomerID     uuid.UUID               `json:"customer_id,omitempty"`
    MasterCallID   uuid.UUID               `json:"master_call_id,omitempty"`
    Source         commonaddress.Address   `json:"source,omitempty"`
    Destinations   []commonaddress.Address `json:"destinations,omitempty"`
    EarlyExecution bool                    `json:"early_execution,omitempty"`
    Connect        bool                    `json:"connect,omitempty"`
    Anonymous      string                  `json:"anonymous,omitempty"`
    Metadata       map[string]interface{}  `json:"metadata,omitempty"` // NEW: internal-only metadata passthrough
}
```

**Step 2: Write the failing test**

Add a test case to `call_calls_test.go` (around line 291) that passes a non-nil `metadata` map and asserts the marshaled request body includes `"metadata":{"route_provider_ids":["..."]}`.

**Step 3: Run test to verify it fails**

```bash
cd bin-common-handler
go test ./pkg/requesthandler -run TestCallV1CallsCreate -v
# Expected: FAIL
```

**Step 4: Implement the signature changes for both RPCs**

```go
// call_calls.go — CallV1CallsCreate
func (r *requestHandler) CallV1CallsCreate(
    ctx context.Context,
    customerID uuid.UUID,
    flowID uuid.UUID,
    masterCallID uuid.UUID,
    source *commonaddress.Address,
    destinations []commonaddress.Address,
    earlyExecution bool,
    connect bool,
    anonymous string,
    metadata map[string]interface{}, // NEW
) ([]*cmcall.Call, []*cmgroupcall.Groupcall, error) {
    // marshal V1DataCallsPost with Metadata: metadata
    // ... rest unchanged
}

// call_calls.go — CallV1CallCreateWithID
func (r *requestHandler) CallV1CallCreateWithID(
    ctx context.Context,
    id uuid.UUID,
    customerID uuid.UUID,
    flowID uuid.UUID,
    activeflowID uuid.UUID,
    masterCallID uuid.UUID,
    source *commonaddress.Address,
    destination *commonaddress.Address,
    groupcallID uuid.UUID,
    earlyExecution bool,
    connect bool,
    anonymous string,
    metadata map[string]interface{}, // NEW
) (*cmcall.Call, error) {
    // marshal V1DataCallsIDPost with Metadata: metadata
    // ... rest unchanged
}
```

**Step 5: Run test to verify it passes**

```bash
go test ./pkg/requesthandler -run TestCallV1CallsCreate -v
# Expected: PASS
```

**Step 6: Update existing test call sites in this file**

The existing test at line 291 will now fail to compile because it's missing the 9th parameter. Pass `nil` for backward compatibility.

```go
// call_calls_test.go:291
resCalls, resGroupcalls, err := reqHandler.CallV1CallsCreate(ctx, tt.customerID, tt.flowID, tt.masterCallID, tt.source, tt.destinations, tt.ealryExecution, tt.connect, "", nil)
```

**Step 7: Commit**

```bash
git add bin-common-handler/ bin-call-manager/pkg/listenhandler/models/request/calls.go
git commit -m "NOJIRA-call-metadata-route-provider-ids

- bin-common-handler: Add metadata param to CallV1CallsCreate
- bin-call-manager: Add Metadata field to V1DataCallsPost request"
```

---

### Task 3: Regenerate mocks + verify bin-common-handler

**Step 1: Regenerate mocks**

```bash
cd bin-common-handler
go generate ./...
```

**Step 2: Run verification**

```bash
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

Expected: all tests pass, lint clean.

**Step 3: Commit**

```bash
git add bin-common-handler/
git commit -m "NOJIRA-call-metadata-route-provider-ids

- bin-common-handler: Regenerate mocks for extended signatures"
```

---

## Phase 2 — bin-route-manager: synthetic dialroute support

### Task 4: Add `TargetProviderIDs` to dialroute request model

**Files:**
- Modify: `bin-route-manager/pkg/listenhandler/models/request/dialroute.go`

**Step 1: Check existing request shape**

```bash
cat bin-route-manager/pkg/listenhandler/models/request/dialroute.go
```

The existing dialroute endpoint probably accepts `customer_id` and `target` as query params, not a request body. Verify with:

```bash
grep -n "dialroutes" bin-route-manager/pkg/listenhandler/*.go
```

**Step 2: Decide transport**

If it's a GET with query params, change to accept a JSON body OR add `target_provider_ids` as a repeated query param. Preferred: **change to POST-style body** so arrays are clean. Keep the HTTP method as-is on the RPC bus (method strings in sock.Request don't carry REST semantics).

**Step 3: Add request struct**

```go
// bin-route-manager/pkg/listenhandler/models/request/dialroute.go
package request

import (
    "github.com/gofrs/uuid"
    rmroute "monorepo/bin-route-manager/models/route"
)

type V1DataDialroutesGet struct {
    Filters           map[rmroute.Field]any `json:"filters,omitempty"`
    TargetProviderIDs []uuid.UUID           `json:"target_provider_ids,omitempty"`
}
```

**Step 4: Commit**

```bash
git add bin-route-manager/pkg/listenhandler/models/request/dialroute.go
git commit -m "NOJIRA-call-metadata-route-provider-ids

- bin-route-manager: Add V1DataDialroutesGet request struct"
```

---

### Task 5: Extend `DialrouteList` handler with target provider override

**Files:**
- Modify: `bin-route-manager/pkg/routehandler/dialroute.go:14`
- Modify: `bin-route-manager/pkg/routehandler/main.go` (interface definition)
- Test: `bin-route-manager/pkg/routehandler/dialroute_test.go`

**Step 1: Write the failing tests**

Add test cases to `dialroute_test.go` covering:

1. **Single provider override** — one ID in the array, returns one synthetic route with `Route.ID == ProviderID`.
2. **Three providers in order** — asserts three synthetic routes, IDs match provider IDs, priorities 0/1/2, and each `Route.ID` is unique.
3. **Synthetic route IDs are unique and non-Nil** — critical for call-manager failover (C1).
4. **Unknown provider returns error** — mock `providerHandler.Get` to return an error; assert `DialrouteList` propagates the error without falling back to normal merge (S2).
5. **Empty `targetProviderIDs` preserves existing behavior** — the signature change alone must not regress the normal merge path. Update existing tests to pass `nil` for the new param.

```go
func Test_DialrouteList_WithTargetProviderIDs(t *testing.T) {
    tests := []struct {
        name              string
        customerID        uuid.UUID
        target            string
        targetProviderIDs []uuid.UUID

        providerGetError  error // for unknown-provider case

        expectRes []*route.Route
        expectErr bool
    }{
        {
            name:              "single provider override returns synthetic route with ID=ProviderID",
            customerID:        uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
            target:            "+1",
            targetProviderIDs: []uuid.UUID{uuid.FromStringOrNil("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")},
            expectRes: []*route.Route{
                {
                    ID:         uuid.FromStringOrNil("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"), // = ProviderID
                    CustomerID: uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
                    ProviderID: uuid.FromStringOrNil("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
                    Target:     "+1",
                    Priority:   0,
                    Name:       "synthetic-test-route",
                    Detail:     "Synthetic route generated for route_provider_ids override. Not persisted.",
                },
            },
        },
        // ... more cases per list above
    }
    // ...
}
```

**Step 2: Run test to verify it fails**

```bash
cd bin-route-manager
go test ./pkg/routehandler -run Test_DialrouteList -v
# Expected: FAIL (signature mismatch or missing override branch)
```

**Step 3: Implement the override branch**

```go
// bin-route-manager/pkg/routehandler/dialroute.go:14
func (h *routeHandler) DialrouteList(
    ctx context.Context,
    customerID uuid.UUID,
    target string,
    targetProviderIDs []uuid.UUID, // NEW
) ([]*route.Route, error) {
    log := logrus.WithFields(logrus.Fields{
        "func":                "DialrouteList",
        "customer_id":         customerID,
        "target":              target,
        "target_provider_ids": targetProviderIDs,
    })

    // Override: when targetProviderIDs is set, return synthetic routes in order.
    if len(targetProviderIDs) > 0 {
        // Validate provider existence before constructing synthetic routes (S2).
        // Fail fast so the admin gets a clear error instead of a silent mid-dial hangup.
        for _, pid := range targetProviderIDs {
            if _, err := h.providerHandler.Get(ctx, pid); err != nil {
                log.Errorf("Could not get provider for synthetic dialroute. provider_id: %s, err: %v", pid, err)
                return nil, errors.Wrapf(err, "provider not found: %s", pid)
            }
        }

        // C1 fix: use ProviderID as the synthetic Route.ID so call-manager's failover
        // tracking (outgoing_call.go:306, 571) can uniquely identify each route.
        // Avoids generating throwaway UUIDs. Duplicate provider IDs make the duplicate
        // unreachable by the failover tracker (pathological input, accepted as-is).
        res := make([]*route.Route, 0, len(targetProviderIDs))
        for i, pid := range targetProviderIDs {
            res = append(res, &route.Route{
                ID:         pid, // synthetic ID = provider ID
                CustomerID: customerID,
                ProviderID: pid,
                Target:     target,
                Priority:   i,
                Name:       "synthetic-test-route",
                Detail:     "Synthetic route generated for route_provider_ids override. Not persisted.",
            })
        }
        log.WithField("synthetic_routes", res).Info("Returning synthetic dialroutes for provider override")
        return res, nil
    }

    // existing normal-merge path unchanged below
    customerRoutes, err := h.ListByTarget(ctx, customerID, target)
    // ...
}
```

Notes:
- **Info-level log** on override activation — it's a significant state change worth tracing in production (internal admin feature, not a customer event).
- **`providerHandler.Get`** requires the route handler to depend on the provider handler. Check the existing handler dependency graph in `cmd/route-manager/main.go` — if not already wired, add the dependency in a separate earlier step. If it would create a cycle, use `h.db.ProviderGet` directly (already used by `providerHandler.Get` internally).
- **Synthetic route field population** (S3): `Name` and `Detail` are populated with human-readable strings so log/event consumers see something meaningful instead of empty strings. `TMCreate`/`TMUpdate`/`TMDelete` remain nil (acceptable: the route is never persisted).

**Step 4: Update the interface in `main.go`**

```bash
grep -n "DialrouteList" bin-route-manager/pkg/routehandler/main.go
```

Update the interface signature to match.

**Step 5: Regenerate mocks**

```bash
go generate ./pkg/routehandler/...
```

**Step 6: Run test to verify it passes**

```bash
go test ./pkg/routehandler -run Test_DialrouteList -v
# Expected: PASS
```

**Step 7: Commit**

```bash
git add bin-route-manager/pkg/routehandler/
git commit -m "NOJIRA-call-metadata-route-provider-ids

- bin-route-manager: Extend DialrouteList with targetProviderIDs override
- bin-route-manager: Return synthetic routes in array order when override set
- bin-route-manager: Regenerate routehandler mocks"
```

---

### Task 6: Wire the override through route-manager's listenhandler

**Files:**
- Modify: `bin-route-manager/pkg/listenhandler/v1_dialroutes.go` (or equivalent)

**Step 1: Find the listen handler for dialroutes**

```bash
grep -rn "dialroutes\|DialrouteList" bin-route-manager/pkg/listenhandler/ --include="*.go"
```

**Step 2: Parse `target_provider_ids` from the request body**

Adjust the handler to:
1. Unmarshal `V1DataDialroutesGet` from `m.Data` (instead of or in addition to existing query-param parsing).
2. Pass `req.TargetProviderIDs` into `h.routeHandler.DialrouteList(...)`.
3. Backward compat: if body is empty (existing callers), default to `nil` slice — existing tests still pass.

**Step 3: Add a listenhandler test for the new field**

**Step 4: Run tests**

```bash
go test ./pkg/listenhandler -v
```

**Step 5: Commit**

```bash
git add bin-route-manager/pkg/listenhandler/
git commit -m "NOJIRA-call-metadata-route-provider-ids

- bin-route-manager: Parse target_provider_ids from dialroute request body"
```

---

### Task 7: Verify bin-route-manager

```bash
cd bin-route-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

Expected: green. Commit any `go.mod`/`go.sum` drift.

```bash
git add bin-route-manager/go.mod bin-route-manager/go.sum 2>/dev/null
git commit -m "NOJIRA-call-metadata-route-provider-ids

- bin-route-manager: Sync go.mod after signature changes" || true
```

---

## Phase 3 — bin-call-manager: metadata on call + wire to dialroute

### Task 8: Add `MetadataKeyRouteProviderIDs` constant

**Files:**
- Modify: `bin-call-manager/models/call/metadata.go`

**Step 1: Add the constant**

```go
// bin-call-manager/models/call/metadata.go
package call

type MetadataKey = string

const (
    // MetadataKeyRTPDebug indicates RTP debug capture was enabled for this call.
    MetadataKeyRTPDebug MetadataKey = "rtp_debug"

    // MetadataKeyRouteProviderIDs lists provider UUIDs (as strings) that the call
    // must be routed through in order. Used by the admin provider-test-call flow.
    // When set, call-manager passes these IDs to route-manager's DialrouteList,
    // which returns synthetic dialroutes bypassing normal customer/default merging.
    MetadataKeyRouteProviderIDs MetadataKey = "route_provider_ids"
)
```

**Step 2: Commit**

```bash
git add bin-call-manager/models/call/metadata.go
git commit -m "NOJIRA-call-metadata-route-provider-ids

- bin-call-manager: Add MetadataKeyRouteProviderIDs constant"
```

---

### Task 9: Extend `CreateCallsOutgoing` / `CreateCallOutgoing` signatures

**Files:**
- Modify: `bin-call-manager/pkg/callhandler/main.go:61-72` (interface)
- Modify: `bin-call-manager/pkg/callhandler/outgoing_call.go:42, 102`

**Step 1: Update interface**

```go
// bin-call-manager/pkg/callhandler/main.go
CreateCallsOutgoing(
    ctx context.Context,
    customerID uuid.UUID,
    flowID uuid.UUID,
    masterCallID uuid.UUID,
    source *commonaddress.Address,
    destinations []commonaddress.Address,
    earlyExecution bool,
    connect bool,
    anonymous string,
    metadata map[string]interface{}, // NEW
) ([]*call.Call, []*groupcall.Groupcall, error)

CreateCallOutgoing(
    ctx context.Context,
    id uuid.UUID,
    customerID uuid.UUID,
    flowID uuid.UUID,
    activeflowID uuid.UUID,
    masterCallID uuid.UUID,
    groupcallID uuid.UUID,
    source *commonaddress.Address,
    destination *commonaddress.Address,
    earlyExecution bool,
    connect bool,
    anonymous string,
    metadata map[string]interface{}, // NEW
) (*call.Call, error)
```

**Step 2: Propagate metadata into Call record**

In `outgoing_call.go`, wherever the Call struct is constructed (probably via `h.db.CallCreate(...)` or a helper in `db.go`), merge the passed-in `metadata` with any already-set keys (like `rtp_debug`).

```go
// Inside CreateCallOutgoing, before DB insert:
if metadata == nil {
    metadata = map[string]interface{}{}
}
// Merge is "caller wins" for now — acceptable since rtp_debug is set later (post-creation),
// so there's no collision at creation time.
newCall := &call.Call{
    // ... existing fields
    Metadata: metadata,
}
```

Verify that `start.go:656-668`'s RTP-debug path still works: it reads `c.Metadata`, adds `rtp_debug`, updates the row. If `Metadata` is now pre-populated from the caller, the RTP-debug path must not overwrite — it already uses `c.Metadata[key] = true`, so keys coexist. Test this.

**Step 3: Write the failing test**

In `outgoing_call_test.go`, add a case where `CreateCallOutgoing` is called with `metadata = {"route_provider_ids": ["abc"]}` and assert the DB `CallCreate` receives a Call with `Metadata["route_provider_ids"] = ["abc"]`.

**Step 4: Run test**

```bash
cd bin-call-manager
go test ./pkg/callhandler -run Test_CreateCallOutgoing -v
# Expected: FAIL → PASS after implementing
```

**Step 5: Fix test call sites inside callhandler**

`CreateCallsOutgoing` internally calls `CreateCallOutgoing` at line 70. Pass `metadata` through.

**Step 6: Commit**

```bash
git add bin-call-manager/pkg/callhandler/
git commit -m "NOJIRA-call-metadata-route-provider-ids

- bin-call-manager: Add metadata param to CreateCallsOutgoing/CreateCallOutgoing
- bin-call-manager: Persist caller-supplied metadata on Call record"
```

---

### Task 10: Update `processV1CallsPost` listen handler

**Files:**
- Modify: `bin-call-manager/pkg/listenhandler/v1_calls.go:114-155`
- Modify: `bin-call-manager/pkg/listenhandler/v1_calls.go:159-200` (processV1CallsIDPost — also needs `Metadata` plumbing)
- Modify: `bin-call-manager/pkg/listenhandler/models/request/calls.go` (add `Metadata` to `V1DataCallsIDPost` too)
- Test: `bin-call-manager/pkg/listenhandler/v1_calls_test.go`

**Step 1: Write failing test**

Add a `TestProcessV1CallsPost_WithMetadata` test asserting that a request body containing `"metadata": {"route_provider_ids": ["abc"]}` results in `h.callHandler.CreateCallsOutgoing` being called with that metadata argument.

**Step 2: Run to fail**

**Step 3: Plumb `req.Metadata` into the handler call**

```go
// v1_calls.go:131
calls, groupcalls, err := h.callHandler.CreateCallsOutgoing(
    ctx, req.CustomerID, req.FlowID, req.MasterCallID,
    req.Source, req.Destinations,
    req.EarlyExecution, req.Connect, req.Anonymous,
    req.Metadata, // NEW
)
```

Do the same for `processV1CallsIDPost`.

**Step 4: Commit**

```bash
git add bin-call-manager/pkg/listenhandler/
git commit -m "NOJIRA-call-metadata-route-provider-ids

- bin-call-manager: Plumb metadata through processV1CallsPost and CallsIDPost
- bin-call-manager: Add Metadata field to V1DataCallsIDPost"
```

---

### Task 11: Read `route_provider_ids` in `getDialroutes`

**Files:**
- Modify: `bin-call-manager/pkg/callhandler/outgoing_call.go:454-486` (`getDialroutes`)
- Test: `bin-call-manager/pkg/callhandler/outgoing_call_test.go:382, 1257`

**Step 1: Change `getDialroutes` signature to accept a Call (not just customerID + destination)**

The current signature doesn't see metadata. The cleanest fix is:

```go
func (h *callHandler) getDialroutes(ctx context.Context, c *call.Call) ([]rmroute.Route, error) {
    customerID := c.CustomerID
    destination := &c.Destination
    // existing body
    // ...

    // Extract targetProviderIDs from metadata
    var targetProviderIDs []uuid.UUID
    if raw, ok := c.Metadata[call.MetadataKeyRouteProviderIDs]; ok {
        if arr, ok := raw.([]interface{}); ok {
            for _, v := range arr {
                if s, ok := v.(string); ok {
                    if id, err := uuid.FromString(s); err == nil {
                        targetProviderIDs = append(targetProviderIDs, id)
                    }
                }
            }
        }
    }

    res, err := h.reqHandler.RouteV1DialrouteList(ctx, filters, targetProviderIDs)
    // ...
}
```

Update the caller at line 173 to pass the full Call.

**Step 2: Write the failing test**

Add a test case to `outgoing_call_test.go` Test_getDialroutes that constructs a Call with `Metadata["route_provider_ids"] = []interface{}{"uuid-a", "uuid-b"}`, mocks `RouteV1DialrouteList` expecting the parsed UUID slice, and asserts it's called.

**Step 3: Run, fail, implement, pass**

```bash
go test ./pkg/callhandler -run Test_getDialroutes -v
```

**Step 4: Update existing mock expectations**

Lines 382, 1257 of `outgoing_call_test.go` use `gomock.Any()` for the filter param — they'll still pass for existing cases but now need `gomock.Any()` for the new `targetProviderIDs` param too:

```go
mockReq.EXPECT().RouteV1DialrouteList(ctx, gomock.Any(), gomock.Any()).Return(...)
```

**Step 5: Commit**

```bash
git add bin-call-manager/pkg/callhandler/
git commit -m "NOJIRA-call-metadata-route-provider-ids

- bin-call-manager: Extract route_provider_ids from call metadata
- bin-call-manager: Forward targetProviderIDs to RouteV1DialrouteList
- bin-call-manager: Update outgoing_call tests for new RPC signature"
```

---

### Task 12: Verify bin-call-manager

```bash
cd bin-call-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

Commit any drift. Address any failures with minimal fixes (likely more test call sites using `gomock.Any()` for the new param).

---

## Phase 4 — Update other CallV1CallsCreate callers

### Task 13: Update bin-flow-manager callers

**Files:**
- Modify: `bin-flow-manager/pkg/activeflowhandler/actionhandle.go:539, 933`
- Modify: `bin-flow-manager/pkg/activeflowhandler/actionhandle_test.go:970, 4509`

**Step 1: Add `nil` metadata to both call sites**

```go
// actionhandle.go:539
resCalls, resGroupcalls, err := h.reqHandler.CallV1CallsCreate(
    ctx, f.CustomerID, f.ID, af.ReferenceID,
    &opt.Source, opt.Destinations,
    earlyExecution, executeNext, opt.Anonymous,
    nil, // metadata: flow-manager does not set metadata
)
```

Do the same at line 933.

**Step 2: Update test mock expectations**

```go
// actionhandle_test.go:970
mockReq.EXPECT().CallV1CallsCreate(
    ctx, tt.responseFlow.CustomerID, tt.responseFlow.ID, tt.af.ReferenceID,
    tt.expectCallSource, tt.expectCallDestinations,
    tt.expectEarlyExecution, tt.expectExecuteNextMasterOnHangup, tt.expectAnonymous,
    nil, // NEW: metadata
).Return(tt.responseCalls, tt.responseGroupcalls, nil)
```

Same for line 4509.

**Step 3: Verify**

```bash
cd bin-flow-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 4: Commit**

```bash
git add bin-flow-manager/
git commit -m "NOJIRA-call-metadata-route-provider-ids

- bin-flow-manager: Pass nil metadata to CallV1CallsCreate (no metadata needed)"
```

---

### Task 14: Update bin-queue-manager callers

**Files:**
- Modify: `bin-queue-manager/pkg/queuecallhandler/execute.go:48`
- Modify: `bin-queue-manager/pkg/queuecallhandler/execute_test.go:110`

Same pattern as Task 13 — pass `nil` for metadata.

```bash
cd bin-queue-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
git add bin-queue-manager/
git commit -m "NOJIRA-call-metadata-route-provider-ids

- bin-queue-manager: Pass nil metadata to CallV1CallsCreate"
```

---

### Task 14b: Update bin-call-manager `groupcallhandler` callers of `CallV1CallCreateWithID`

**Files:**
- Modify: `bin-call-manager/pkg/groupcallhandler/start.go:151, 227, 351`
- Modify: `bin-call-manager/pkg/groupcallhandler/dial.go:119`
- Modify: corresponding `_test.go` files that set mock expectations on `CallV1CallCreateWithID`

**Step 1: Pass `nil` metadata at every call site**

```go
// Example — start.go:151
tmp, err := h.reqHandler.CallV1CallCreateWithID(
    ctx, callID, customerID, flowID, uuid.Nil, masterCallID,
    source, destination, id, false, false, anonymous,
    nil, // metadata: groupcall fan-out carries no metadata in v1
)
```

**Step 2: Update test expectations**

`gomock.Any()` or explicit `nil` on the new param for every `mockReq.EXPECT().CallV1CallCreateWithID(...)` call.

**Step 3: Verify (already part of bin-call-manager verification in Task 12)**

This task's changes are inside `bin-call-manager`, so the Task 12 verification run covers them. Commit separately for reviewability:

```bash
git add bin-call-manager/pkg/groupcallhandler/
git commit -m "NOJIRA-call-metadata-route-provider-ids

- bin-call-manager: Pass nil metadata from groupcallhandler CallV1CallCreateWithID callers"
```

---

### Task 14c: Update bin-campaign-manager caller

**Files:**
- Modify: `bin-campaign-manager/pkg/campaignhandler/execute.go:252`
- Modify: corresponding `_test.go` file

**Step 1: Pass `nil` metadata**

```go
newCall, err := h.reqHandler.CallV1CallCreateWithID(
    ctx, /* existing args */,
    nil, // metadata
)
```

**Step 2: Update test expectations**

Add the new param to any `mockReq.EXPECT().CallV1CallCreateWithID(...)` call.

**Step 3: Verify**

```bash
cd bin-campaign-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 4: Commit**

```bash
git add bin-campaign-manager/
git commit -m "NOJIRA-call-metadata-route-provider-ids

- bin-campaign-manager: Pass nil metadata to CallV1CallCreateWithID"
```

---

### Task 15: Update bin-api-manager existing caller

**Files:**
- Modify: `bin-api-manager/pkg/servicehandler/call.go:111`
- Modify: `bin-api-manager/pkg/servicehandler/call_test.go:388`

Pass `nil` for metadata here too — the customer-facing `POST /v1/calls` path does **not** forward any client-supplied metadata. Only the future admin endpoint (separate PRD) will pass a non-nil metadata.

```go
// call.go:111
tmpCalls, tmpGroupcalls, err := h.reqHandler.CallV1CallsCreate(
    ctx, a.CustomerID, targetFlowID, uuid.Nil,
    source, destinations, false, false, anonymous,
    nil, // metadata: customer-facing path never sets metadata
)
```

Verify:

```bash
cd bin-api-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
git add bin-api-manager/
git commit -m "NOJIRA-call-metadata-route-provider-ids

- bin-api-manager: Pass nil metadata from customer-facing call create (internal-only feature)"
```

---

## Phase 5 — Documentation

### Task 16: Update RST docs

**Files:**
- Modify: `bin-api-manager/docsdev/source/call_struct_metadata.rst` (create) or append to `call_overview.rst`

**Step 1: Document the metadata field as internal-only**

Add a short note under the call struct docs:

```rst
Call Metadata
-------------

The ``metadata`` field on a Call is an internal-only map populated by VoIPbin
services. It is not accepted as input on the public ``POST /v1/calls`` endpoint.
Internal processes (e.g. the admin provider-test-call flow) may set keys that
alter call behavior:

- ``route_provider_ids`` (array of provider UUIDs) — when present, the call is
  routed through these providers in order, bypassing normal route selection.
  Set by internal admin-test flows only.
- ``rtp_debug`` (bool) — when true, RTPEngine PCAP capture is enabled for the
  call. Inherited from customer and number metadata.

Customers see ``metadata`` in call webhook events and ``GET /v1/calls/{id}``
responses for the calls they own.
```

**Step 2: Rebuild HTML**

```bash
cd bin-api-manager/docsdev
rm -rf build && python3 -m sphinx -M html source build
```

**Step 3: Force-add build output**

```bash
git add -f bin-api-manager/docsdev/build/
git add bin-api-manager/docsdev/source/
git commit -m "NOJIRA-call-metadata-route-provider-ids

- bin-api-manager: Document internal-only call metadata and route_provider_ids key"
```

---

## Phase 6 — Final full verification

### Task 17: Cross-service smoke check

Run verification on every affected service in sequence. Stop at the first failure.

```bash
for svc in bin-common-handler bin-route-manager bin-call-manager bin-flow-manager bin-queue-manager bin-api-manager bin-campaign-manager; do
  echo "=== $svc ==="
  cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-call-metadata-route-provider-ids/$svc
  go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m || { echo "FAILED: $svc"; exit 1; }
done
```

Expected: all green.

### Task 18: Fetch main + check conflicts (pre-PR)

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-call-metadata-route-provider-ids
git fetch origin main
git merge-tree $(git merge-base HEAD origin/main) HEAD origin/main | grep -E "^(CONFLICT|changed in both)" || echo "No conflicts"
git log --oneline HEAD..origin/main
```

If conflicts: rebase, resolve, re-run Task 17.

### Task 19: Push + open PR

```bash
git push -u origin NOJIRA-call-metadata-route-provider-ids

gh pr create --title "NOJIRA-call-metadata-route-provider-ids" --body "$(cat <<'EOF'
Add internal-only metadata pass-through on Call creation and wire route_provider_ids
through call-manager and route-manager.

- bin-common-handler: Add metadata param to CallV1CallsCreate; add targetProviderIDs to RouteV1DialrouteList; regenerate mocks
- bin-route-manager: Return synthetic dialroutes in array order when targetProviderIDs is set; parse target_provider_ids from request body
- bin-call-manager: Add MetadataKeyRouteProviderIDs constant; persist caller-supplied metadata on Call; extract and forward to route-manager from getDialroutes
- bin-flow-manager: Pass nil metadata to CallV1CallsCreate (unchanged behavior)
- bin-queue-manager: Pass nil metadata to CallV1CallsCreate (unchanged behavior)
- bin-api-manager: Pass nil metadata from customer-facing call create; document internal-only metadata in RST docs
- bin-campaign-manager: Pass nil metadata to CallV1CallCreateWithID
EOF
)"
```

---

## Out of Scope (follow-up PRs)

- **Admin endpoint `POST /v1/providers/{id}/calls`** — belongs to `provider-test-call.prd.md`. Will consume this metadata mechanism.
- **`ProviderCall` persistence** — provider-test-call PRD territory.
- **Metadata mutation API** — future: `POST /v1/calls/{id}/metadata` for runtime updates.
- **Customer-facing metadata whitelist** — not needed today; metadata is strictly internal.

## Risk Register

| Risk | Likelihood | Mitigation |
|---|---|---|
| `Call.Metadata` collision between caller-supplied and `rtp_debug` | Low | `rtp_debug` is set post-creation via `Metadata[key] = true`; collision-free. Documented merge rule: post-creation keys overwrite same-key caller values |
| Signature change breaks a service missed in the audit | Medium | Full verification runs on all 7 services; `go build` fails fast on missed sites |
| Synthetic routes have duplicate `uuid.Nil` IDs breaking failover | **Resolved by C1 fix** | Synthetic `Route.ID` = `ProviderID`; test asserts all IDs unique and non-Nil |
| Customer-derived input forwarded into `metadata` param | Medium | Design invariant forbids it; reviewer must check every new `CallV1CallsCreate`/`CallV1CallCreateWithID` caller for unsafe forwarding |
| Metadata leaks sensitive info via webhook | Low | `route_provider_ids` is UUIDs only; admin owns the call; acceptable for v1 |
| `uuid.FromString` typo silently skips IDs | Medium | Log at Info level when any ID fails to parse; fail fast at api-manager in the future admin endpoint |

---

## Execution Handoff

**Plan complete and saved to `docs/plans/2026-04-21-call-metadata-route-provider-ids-plan.md`.**

Two execution options:

1. **Subagent-Driven (this session)** — I dispatch a fresh subagent per task, review between tasks, fast iteration. Best for catching cross-service issues in real time.
2. **Parallel Session (separate)** — Open a new session in this worktree with `superpowers:executing-plans`, batch execution with checkpoints. Best if you want to focus attention elsewhere.

**Which approach?**
