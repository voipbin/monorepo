# Plan: Provider Call — Phase 1: `bin-call-manager` skip-source-validation metadata key

## Summary
Add a new typed metadata key `skip_source_validation` to `bin-call-manager`. When set to `true` on a Call's `Metadata` at creation time, `getValidatedSourceForOutgoingCall` returns the admin-supplied source address verbatim — bypassing the customer-ownership lookup and the silent fallback to `Customer.DefaultOutgoingSourceNumberID`. This is Phase 1 of the Provider Call PRD and unblocks Phase 4 (the public admin endpoint), because the api-manager handler will set this key server-side and the call-manager listen-handler's `ValidMetadataKeys` gate must recognise it or reject the request with HTTP 400.

## User Story
As a platform admin running a provider call, I want my chosen source number to reach the provider verbatim, so the provider's From / P-Asserted-Identity allowlist check succeeds instead of seeing the customer's default outgoing number.

## Problem → Solution
**Current state:** `getValidatedSourceForOutgoingCall` (`bin-call-manager/pkg/callhandler/outgoing_call.go:694`) silently swaps an admin-supplied source that isn't owned by the customer with `Customer.DefaultOutgoingSourceNumberID`. Providers commonly reject INVITEs whose `From` / PAI doesn't match a pre-allowed caller ID, so the test fails for the wrong reason (wrong source, not provider issue).

**Desired state:** The admin-only call-creation path can set `Call.Metadata["skip_source_validation"] = true` and the validator short-circuits on the tel-destination branch, returning `&source` unchanged.

## Metadata
- **Complexity**: Small
- **Source PRD**: `.claude/PRPs/prds/provider-call.prd.md`
- **PRD Phase**: Phase 1 — `bin-call-manager` source-validation bypass
- **Estimated Files**: 3 changed (`metadata.go`, `metadata_test.go`, `outgoing_call.go`), 2 test files touched (`outgoing_call_test.go`)

---

## UX Design

Internal change — no user-facing UX transformation. The PRD as a whole is an admin API; this phase is the call-manager side of the plumbing.

---

## Mandatory Reading

| Priority | File | Lines | Why |
|---|---|---|---|
| P0 | `bin-call-manager/models/call/metadata.go` | 1-32 | The pattern to follow exactly. Defines `MetadataKey` alias, existing constants `MetadataKeyRTPDebug` + `MetadataKeyRouteProviderIDs`, and the `ValidMetadataKeys` registry with "how to add a new key" comment. |
| P0 | `bin-call-manager/pkg/callhandler/outgoing_call.go` | 694-766 | `getValidatedSourceForOutgoingCall` — the function the new metadata key modifies. Shows current structure: non-tel bypass → nil-customer bypass → ownership lookup → default fallback → nil return. |
| P0 | `bin-call-manager/pkg/callhandler/outgoing_call.go` | 170-207 | `CreateCallOutgoing`'s call site for `getValidatedSourceForOutgoingCall` (line 197). Shows that `metadata map[string]interface{}` is already available at that scope (it's a parameter on `CreateCallOutgoing` at line 117, and already passed to `getDialroutes` at line 177). |
| P0 | `bin-call-manager/pkg/callhandler/outgoing_call.go` | 460-531 | `getDialroutes` — the reference pattern from PR #793 for reading a typed key out of `metadata map[string]interface{}`. Uses `raw, ok := metadata[call.MetadataKeyXxx]` then type-asserts to `[]interface{}`. For a bool key, the equivalent is `raw.(bool)`. |
| P1 | `bin-call-manager/models/call/metadata_test.go` | 1-26 | Registry-completeness test. New constant must be added to the `required` slice so the test keeps catching declared-but-unregistered constants. |
| P1 | `bin-call-manager/pkg/callhandler/outgoing_call_test.go` | 1796-2068 | `Test_getValidatedSourceForOutgoingCall` — the table-driven test suite to extend. 10 existing cases show the full gomock + t.Run + reflect.DeepEqual shape. |

## External Documentation

No external research needed — feature uses established internal patterns.

---

## Patterns to Mirror

### METADATA_KEY_DECLARATION
// SOURCE: bin-call-manager/models/call/metadata.go:12-18
```go
// MetadataKeyRouteProviderIDs lists provider UUIDs (as a []string) that the call
// must be routed through in failover order. Used by internal admin-test flows.
// Set CREATION-TIME only by server-side trusted code. When present, call-manager
// forwards the IDs to route-manager's DialrouteList, which returns synthetic
// dialroutes bypassing normal customer/default merging.
MetadataKeyRouteProviderIDs MetadataKey = "route_provider_ids"
```

### REGISTRY_ENTRY
// SOURCE: bin-call-manager/models/call/metadata.go:28-31
```go
var ValidMetadataKeys = map[MetadataKey]bool{
    MetadataKeyRTPDebug:         true,
    MetadataKeyRouteProviderIDs: true,
}
```

### METADATA_READ_PATTERN
// SOURCE: bin-call-manager/pkg/callhandler/outgoing_call.go:494-500 (abbreviated — the real check is array-typed; bool-typed analog shown for the new key)
```go
// array-typed read from the existing route_provider_ids key (reference only):
if raw, ok := metadata[call.MetadataKeyRouteProviderIDs]; ok {
    arr, ok := raw.([]interface{})
    if !ok {
        log.Errorf("route_provider_ids metadata is not a []interface{}: %T", raw)
        return nil, fmt.Errorf("route_provider_ids metadata has invalid shape: %T", raw)
    }
    ...
}
```

### EARLY_RETURN_BYPASS
// SOURCE: bin-call-manager/pkg/callhandler/outgoing_call.go:713-723
```go
// non-tel destinations don't need source validation
if destination.Type != commonaddress.TypeTel {
    return &source
}

// if customer info is not available (fail-open from customer-manager),
// skip validation and return source as-is
if cu == nil {
    log.Infof("Customer info not available. Skipping source number validation.")
    return &source
}
log = log.WithField("customer_id", cu.ID)
```
The new `skip_source_validation` check belongs in this same "bypass" section — between the non-tel check and the `cu == nil` check is fine, or immediately after the `cu == nil` check. Both placements work; the plan places it immediately after the non-tel bypass so the flag works even when customer info is present.

### LOGRUS_LOGGING
// SOURCE: bin-call-manager/pkg/callhandler/outgoing_call.go:707-711
```go
log := logrus.WithFields(logrus.Fields{
    "func":        "getValidatedSourceForOutgoingCall",
    "source":      source,
    "destination": destination,
})
```
Follow this shape for any new log lines.

### TABLE_DRIVEN_TEST
// SOURCE: bin-call-manager/pkg/callhandler/outgoing_call_test.go:1796-2068
```go
tests := []struct {
    name string
    source      commonaddress.Address
    destination commonaddress.Address
    customer    *cucustomer.Customer
    responseNumbers []nmnumber.Number
    responseNumErr  error
    responseDefaultNum    *nmnumber.Number
    responseDefaultNumErr error
    expectRes *commonaddress.Address
}{
    // cases...
}
for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        mc := gomock.NewController(t)
        defer mc.Finish()
        mockReq := requesthandler.NewMockRequestHandler(mc)
        h := &callHandler{reqHandler: mockReq}
        ctx := context.Background()
        // conditional mock setup based on tt fields
        if tt.customer != nil && ... {
            mockReq.EXPECT().NumberV1NumberList(...).Return(tt.responseNumbers, tt.responseNumErr)
        }
        res := h.getValidatedSourceForOutgoingCall(ctx, tt.source, tt.destination, tt.customer)
        if !reflect.DeepEqual(res, tt.expectRes) {
            t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
        }
    })
}
```

---

## Files to Change

| File | Action | Justification |
|---|---|---|
| `bin-call-manager/models/call/metadata.go` | UPDATE | Declare `MetadataKeySkipSourceValidation` constant; add it to `ValidMetadataKeys` registry |
| `bin-call-manager/models/call/metadata_test.go` | UPDATE | Add new constant to the `required` slice in `Test_ValidMetadataKeys_contains_all_declared_constants` |
| `bin-call-manager/pkg/callhandler/outgoing_call.go` | UPDATE | Extend `getValidatedSourceForOutgoingCall` signature with `metadata map[string]interface{}`; early-return when key is `true`; update the sole caller at `CreateCallOutgoing` to pass `metadata` |
| `bin-call-manager/pkg/callhandler/outgoing_call_test.go` | UPDATE | Extend `Test_getValidatedSourceForOutgoingCall` to cover the skip-on / skip-off / non-bool-value cases; update existing test case call sites to pass `nil` for the new `metadata` param |

## NOT Building

- No change to `Call.Metadata` model — the field already exists as a JSON column.
- No listen-handler changes — `ValidMetadataKeys` is already consulted by the existing PR #793 listen-handler guard; adding the new key to the registry is enough.
- No `bin-common-handler` changes — no new RPC signatures.
- No `bin-route-manager` changes (Phase 2 handles that).
- No `bin-api-manager` changes (Phase 4 handles the public endpoint).
- No new OpenAPI / RST docs — `skip_source_validation` is an internal key, documented only by the constant's Go doc-comment + the PRD.
- No migration — no DB schema change.
- No E.164 format check when skip is active. The admin-triggered provider-call flow is trusted to supply a source the provider's carrier will accept; enforcing E.164 here would defeat the purpose. Empty strings pass through too — downstream `setChannelVariablesCallerID` handles empty CALLERID gracefully for non-anonymous calls, and the anonymous path has its own PAI validation at `outgoing_call.go:674` that will surface a clean error.

---

## Step-by-Step Tasks

### Task 1: Declare the `MetadataKeySkipSourceValidation` constant + register it
- **ACTION**: Update `bin-call-manager/models/call/metadata.go`.
- **IMPLEMENT**: Add a new `MetadataKey` constant after `MetadataKeyRouteProviderIDs`:
  ```go
  // MetadataKeySkipSourceValidation, when set to true, instructs call-manager's
  // getValidatedSourceForOutgoingCall to return the caller-supplied source address
  // verbatim — skipping the customer-ownership lookup and the silent fallback to
  // Customer.DefaultOutgoingSourceNumberID. Used by internal admin-test flows that
  // must preserve a source number the provider's carrier has pre-authorized
  // (which is typically NOT a number owned by any voipbin customer).
  // Set CREATION-TIME only by server-side trusted code. Do not expose in any
  // customer-facing API body.
  MetadataKeySkipSourceValidation MetadataKey = "skip_source_validation"
  ```
  Add a new entry to `ValidMetadataKeys`:
  ```go
  var ValidMetadataKeys = map[MetadataKey]bool{
      MetadataKeyRTPDebug:             true,
      MetadataKeyRouteProviderIDs:     true,
      MetadataKeySkipSourceValidation: true,
  }
  ```
- **MIRROR**: `METADATA_KEY_DECLARATION` + `REGISTRY_ENTRY` above.
- **IMPORTS**: none added (file has none today).
- **GOTCHA**: The doc-comment MUST state "CREATION-TIME only" and "server-side only" — matches the existing trust-invariant wording from PR #793's design doc and the PRD. Also MUST mention that no customer-facing API body should accept this key.
- **VALIDATE**: `go build ./...` in `bin-call-manager` must succeed.

### Task 2: Update the metadata registry completeness test
- **ACTION**: Update `bin-call-manager/models/call/metadata_test.go`.
- **IMPLEMENT**: Add the new constant to the `required` slice:
  ```go
  required := []MetadataKey{
      MetadataKeyRTPDebug,
      MetadataKeyRouteProviderIDs,
      MetadataKeySkipSourceValidation,
  }
  ```
- **MIRROR**: Existing structure of the file (no change to surrounding tests).
- **IMPORTS**: none added.
- **GOTCHA**: `Test_ValidMetadataKeys_rejects_unknown` doesn't need changes — it already tests the inverse invariant.
- **VALIDATE**: `go test ./models/call/...` in `bin-call-manager` passes.

### Task 3: Extend `getValidatedSourceForOutgoingCall` signature and add the bypass branch
- **ACTION**: Update `bin-call-manager/pkg/callhandler/outgoing_call.go`.
- **IMPLEMENT**:
  1. Change the function signature from:
     ```go
     func (h *callHandler) getValidatedSourceForOutgoingCall(
         ctx context.Context,
         source commonaddress.Address,
         destination commonaddress.Address,
         cu *cucustomer.Customer,
     ) *commonaddress.Address
     ```
     to:
     ```go
     func (h *callHandler) getValidatedSourceForOutgoingCall(
         ctx context.Context,
         source commonaddress.Address,
         destination commonaddress.Address,
         cu *cucustomer.Customer,
         metadata map[string]interface{},
     ) *commonaddress.Address
     ```
  2. Update the doc-comment block (lines 694-700) to document the new parameter and the skip behavior. Suggested text:
     ```
     // getValidatedSourceForOutgoingCall validates and resolves the source address for an outgoing call.
     // For tel-type destinations, the source number must:
     // 1. Have a valid E.164 format (starts with "+")
     // 2. Belong to the customer as a normal (non-virtual) number
     // If either condition fails, it falls back to the customer's default outgoing source number.
     // If no valid source can be determined, it returns nil.
     // For non-tel destinations, the source is returned as-is.
     //
     // If metadata contains MetadataKeySkipSourceValidation == true, all of the above is
     // bypassed and the caller-supplied source is returned verbatim. This is used by
     // internal admin-test flows (e.g., the provider-call endpoint) where the admin needs
     // to preserve a source the provider's carrier has pre-authorized.
     ```
  3. Insert the bypass branch immediately after the non-tel bypass (line 716) and before the `cu == nil` check:
     ```go
     // non-tel destinations don't need source validation
     if destination.Type != commonaddress.TypeTel {
         return &source
     }

     // metadata opts out of customer-ownership validation entirely
     // (used by internal admin-test flows — see MetadataKeySkipSourceValidation)
     if skip, ok := metadata[call.MetadataKeySkipSourceValidation].(bool); ok && skip {
         log.Infof("Source validation skipped per metadata. source: %s", source.Target)
         return &source
     }

     // if customer info is not available (fail-open from customer-manager),
     // skip validation and return source as-is
     if cu == nil {
         ...
     }
     ```
- **MIRROR**: `EARLY_RETURN_BYPASS` + `METADATA_READ_PATTERN` + `LOGRUS_LOGGING` above. The `raw.(bool)` form is the simpler bool-typed analog of `getDialroutes`'s `raw.([]interface{})`; comma-ok avoids panics on wrong-typed values.
- **IMPORTS**: no new imports — `call` package is already imported as `"monorepo/bin-call-manager/models/call"` (see line 27).
- **GOTCHA**: Use the comma-ok type assertion `skip, ok := metadata[call.MetadataKeySkipSourceValidation].(bool)`. A missing key returns the zero-value `interface{}` (nil), and the assertion yields `ok=false, skip=false` — which is what you want (skip only when key is present AND `true`). Do NOT use a non-comma-ok assertion; it panics on a missing key or on a wrong-typed value.
- **GOTCHA 2**: The listen-handler's `ValidMetadataKeys` check (added by PR #793) rejects requests with unknown metadata keys at HTTP 400. Phase 4's api-manager handler will set `skip_source_validation: true` — if Phase 1 doesn't land first, those requests will fail. That's why Phase 1 blocks Phase 4. (Documented in the PRD's Parallelism Notes.)
- **GOTCHA 3**: Update the sole caller at line 197 (`s := h.getValidatedSourceForOutgoingCall(ctx, source, destination, cu)`) to pass the new `metadata` param:
  ```go
  s := h.getValidatedSourceForOutgoingCall(ctx, source, destination, cu, metadata)
  ```
  `metadata` is already a parameter of the enclosing `CreateCallOutgoing` function (line 117).
- **VALIDATE**: `go build ./...` in `bin-call-manager` must succeed. No call-site changes outside `outgoing_call.go` (the function is unexported — grep confirms only one caller).

### Task 4: Extend `Test_getValidatedSourceForOutgoingCall` with skip-source-validation cases
- **ACTION**: Update `bin-call-manager/pkg/callhandler/outgoing_call_test.go` at `Test_getValidatedSourceForOutgoingCall` (line 1796).
- **IMPLEMENT**:
  1. Add a new field `metadata map[string]interface{}` to the anonymous test-case struct at line 1798.
  2. For each of the 10 existing test cases, leave `metadata` as its zero value (`nil`). This asserts no-regression of the existing behavior.
  3. Add four new test cases at the end of the `tests` slice:
     ```go
     {
         name: "skip_source_validation=true preserves unowned source (no mocks called)",
         source: commonaddress.Address{
             Type:   commonaddress.TypeTel,
             Target: "+15559999999", // not owned by customer
         },
         destination: commonaddress.Address{
             Type:   commonaddress.TypeTel,
             Target: "+821100000002",
         },
         customer: &cucustomer.Customer{
             ID: uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000001"),
             DefaultOutgoingSourceNumberID: uuid.FromStringOrNil("b0000000-0000-0000-0000-000000000001"),
         },
         metadata: map[string]interface{}{
             call.MetadataKeySkipSourceValidation: true,
         },
         expectRes: &commonaddress.Address{
             Type:   commonaddress.TypeTel,
             Target: "+15559999999",
         },
     },
     {
         name: "skip_source_validation=true preserves non-E.164 source",
         source: commonaddress.Address{
             Type:   commonaddress.TypeTel,
             Target: "5551234", // not E.164, admin-supplied verbatim
         },
         destination: commonaddress.Address{
             Type:   commonaddress.TypeTel,
             Target: "+821100000002",
         },
         customer: &cucustomer.Customer{
             ID: uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000001"),
         },
         metadata: map[string]interface{}{
             call.MetadataKeySkipSourceValidation: true,
         },
         expectRes: &commonaddress.Address{
             Type:   commonaddress.TypeTel,
             Target: "5551234",
         },
     },
     {
         name: "skip_source_validation=false falls through to ownership validation",
         source: commonaddress.Address{
             Type:   commonaddress.TypeTel,
             Target: "+821100000001",
         },
         destination: commonaddress.Address{
             Type:   commonaddress.TypeTel,
             Target: "+821100000002",
         },
         customer: &cucustomer.Customer{
             ID: uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000001"),
         },
         metadata: map[string]interface{}{
             call.MetadataKeySkipSourceValidation: false,
         },
         responseNumbers: []nmnumber.Number{{Number: "+821100000001"}},
         expectRes: &commonaddress.Address{
             Type:   commonaddress.TypeTel,
             Target: "+821100000001",
         },
     },
     {
         name: "skip_source_validation with non-bool value falls through to ownership validation",
         source: commonaddress.Address{
             Type:   commonaddress.TypeTel,
             Target: "+821100000001",
         },
         destination: commonaddress.Address{
             Type:   commonaddress.TypeTel,
             Target: "+821100000002",
         },
         customer: &cucustomer.Customer{
             ID: uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000001"),
         },
         metadata: map[string]interface{}{
             call.MetadataKeySkipSourceValidation: "true", // wrong type — string not bool
         },
         responseNumbers: []nmnumber.Number{{Number: "+821100000001"}},
         expectRes: &commonaddress.Address{
             Type:   commonaddress.TypeTel,
             Target: "+821100000001",
         },
     },
     ```
  4. Update the call site at line 2053 to pass the new param:
     ```go
     res := h.getValidatedSourceForOutgoingCall(ctx, tt.source, tt.destination, tt.customer, tt.metadata)
     ```
  5. The skip-on cases MUST NOT configure any mocks — the whole point is that no `NumberV1NumberList` / `NumberV1NumberGet` call is made. The existing conditional mock-setup block (lines 2039-2051) already gates on `tt.customer != nil && tt.destination.Type == commonaddress.TypeTel && strings.HasPrefix(tt.source.Target, "+")` — the skip-on cases meet those conditions and would set up mocks that never fire, causing gomock to fail with "missing call". Extend the mock-setup guard to also skip when `skip_source_validation` is true in the metadata:
     ```go
     skipValidation := false
     if v, ok := tt.metadata[call.MetadataKeySkipSourceValidation].(bool); ok && v {
         skipValidation = true
     }
     if !skipValidation && tt.customer != nil && tt.destination.Type == commonaddress.TypeTel && strings.HasPrefix(tt.source.Target, "+") {
         mockReq.EXPECT().NumberV1NumberList(...)
     }
     if !skipValidation && tt.customer != nil && tt.destination.Type == commonaddress.TypeTel && (tt.responseDefaultNum != nil || tt.responseDefaultNumErr != nil) {
         mockReq.EXPECT().NumberV1NumberGet(...)
     }
     ```
- **MIRROR**: `TABLE_DRIVEN_TEST` above.
- **IMPORTS**: no new imports — `call` is already imported in the test file (seen in line 269).
- **GOTCHA**: gomock is strict about unfulfilled expectations. Any `mockReq.EXPECT()` call that doesn't fire causes the test to fail. The skip-on cases must not set up any mocks for `NumberV1NumberList` or `NumberV1NumberGet`.
- **GOTCHA 2**: The "skip with non-bool value falls through" case is important — it tests the defensive comma-ok assertion. If someone future-breaks the assertion to something unsafe, this test catches it.
- **VALIDATE**: `go test -run Test_getValidatedSourceForOutgoingCall ./pkg/callhandler/...` passes all 14 cases.

---

## Testing Strategy

### Unit Tests

| Test | Input | Expected Output | Edge Case? |
|---|---|---|---|
| `Test_ValidMetadataKeys_contains_all_declared_constants` | `MetadataKeySkipSourceValidation` in required slice | registry has it | N — completeness check |
| `skip_source_validation=true preserves unowned source` | tel destination, unowned source, customer has default-number configured, `metadata[skip]=true` | source returned verbatim; zero mock calls | Y — the critical behavior |
| `skip_source_validation=true preserves non-E.164 source` | tel destination, `Target: "5551234"`, `metadata[skip]=true` | source returned verbatim | Y — documents "no E.164 check when skip" |
| `skip_source_validation=false falls through to ownership validation` | tel destination, owned source, `metadata[skip]=false` | existing ownership-validated source | N — regression guard |
| `skip_source_validation with non-bool value falls through` | `metadata[skip]="true"` (string) | existing ownership-validated source (falls through) | Y — defensive type-assertion guard |
| 10 existing cases with `metadata: nil` | unchanged | unchanged | N — no-regression for prior behavior |

### Edge Cases Checklist
- [x] Empty/missing metadata map (existing cases cover this via `nil`)
- [x] Key present with `true`
- [x] Key present with `false`
- [x] Key present with wrong type
- [x] Key absent but metadata non-nil (implicit via existing tests if we pass `map[string]interface{}{}` somewhere — acceptable to skip since nil is equivalent for read)
- [x] Skip-on with customer present (should still bypass customer lookup)
- [x] Skip-on with customer absent (already hits non-customer bypass first; skip check never reached — acceptable, documented by the placement decision)

---

## Validation Commands

### Static Analysis + Unit Tests + Lint (full monorepo verification workflow)
```bash
cd bin-call-manager && \
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```
EXPECT: All five steps green. No stale `go.sum` entries. No new lint warnings.

### Focused test run for quick iteration
```bash
cd bin-call-manager && \
go test -v -run Test_getValidatedSourceForOutgoingCall ./pkg/callhandler/...
```
EXPECT: 14 cases (10 existing + 4 new) all pass.

### Registry completeness
```bash
cd bin-call-manager && \
go test -v -run Test_ValidMetadataKeys ./models/call/...
```
EXPECT: Both tests pass.

### Manual Validation
- [ ] Grep to confirm only one caller of `getValidatedSourceForOutgoingCall` was updated: `grep -rn "getValidatedSourceForOutgoingCall" bin-call-manager/` should show exactly one callsite in `outgoing_call.go` (plus the definition and the tests).
- [ ] Confirm `call.MetadataKeySkipSourceValidation` is referenced in exactly the three expected locations: `metadata.go`, `metadata_test.go`, `outgoing_call.go`, and `outgoing_call_test.go`.

---

## Acceptance Criteria
- [ ] All 4 tasks completed
- [ ] Full verification workflow (`go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`) green in `bin-call-manager`
- [ ] Existing 10 `Test_getValidatedSourceForOutgoingCall` cases still pass
- [ ] 4 new test cases added and passing
- [ ] `Test_ValidMetadataKeys_contains_all_declared_constants` updated and passing
- [ ] No lint errors
- [ ] PRD Phase 1 status flipped from `pending` → `in-progress` (and eventually `complete` after merge) with the plan file path referenced

## Completion Checklist
- [ ] Metadata-key doc-comment states "CREATION-TIME only" and "server-side only" (matches PR #793 pattern)
- [ ] Registry entry added in same commit as the constant declaration
- [ ] Type assertion is comma-ok form (defensive against missing key / wrong type)
- [ ] All call sites of `getValidatedSourceForOutgoingCall` updated for the new signature (only one exists)
- [ ] Test cases exercise: skip-on happy path, skip-on with non-E.164 source, skip-off no-regression, skip with non-bool value
- [ ] Mock-setup guard extended so skip-on cases don't register unfulfilled gomock expectations
- [ ] No hardcoded values — all UUIDs / numbers are consistent with existing test fixture style
- [ ] No unnecessary scope additions (no changes to `setChannelVariablesCallerID`, no listen-handler edits, no docs)
- [ ] Self-contained — no questions needed during implementation

## Risks
| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Mock-setup guard misses a skip-on case, producing "missing call" gomock error | M | Low (test failure, easy fix) | Explicit `skipValidation := ...` local boolean and early-skip both mock blocks |
| Admin passes `skip_source_validation=true` with empty source, producing a call with no caller ID | L | Low (admin error; downstream logs it) | Out of scope per PRD — admin is trusted. Documented in `NOT Building`. |
| PR #793's listen-handler rejects Phase 4's test requests before Phase 1 lands | H (if phases mis-sequenced) | Medium (Phase 4 tests fail with HTTP 400) | PRD dependency graph already makes Phase 4 depend on Phase 1. Merge Phase 1 first. |
| Future change to `getValidatedSourceForOutgoingCall` forgets to propagate `metadata` into a new code path | L | Low (skip flag silently ignored) | Comma-ok assertion is defensive; existing tests cover the no-skip path; this is a pattern concern, not a Phase-1 risk |

## Notes
- This phase changes **no public-facing API** — all effects are internal to `bin-call-manager`.
- This phase changes **no DB schema** — no migration.
- Phase 2 (route-manager `ProviderCall` entity) can run concurrently with this phase; they touch different packages.
- The placement of the skip branch (after non-tel bypass, before customer-nil bypass) means: when the customer record is unavailable, the existing fail-open `return &source` already applies and the skip flag doesn't need to be consulted. When the customer record IS available, the skip flag is the only way to bypass the ownership lookup. This is the intended behavior.
- Reasons to NOT add an E.164 format check when skip is active (despite the PRD hinting at "minimal E.164 sanity"):
  1. The provider-call flow is admin-only; admins are trusted.
  2. Provider carriers sometimes accept non-E.164 source numbers (e.g., national-only formats); enforcing E.164 here would defeat the feature.
  3. Empty sources are handled downstream by `setChannelVariablesCallerID` — not worth duplicating the check.
  The `NOT Building` section spells this out.
