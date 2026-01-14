# Design: Refactoring Query-Based to Body-Based Filters

**Date:** 2026-01-14
**Status:** Approved
**Scope:** bin-common-handler/pkg/requesthandler and all callers across monorepo

## Problem Statement

Currently, 14 requesthandler methods send filter criteria as query parameters instead of in the request body. This violates the established pattern and creates inconsistency across the API.

Example of incorrect pattern:
```go
func CampaignV1CampaignGetsByCustomerID(ctx context.Context, customerID uuid.UUID, pageToken string, pageSize uint64) ([]Campaign, error) {
    uri := fmt.Sprintf("/v1/campaigns?page_token=%s&page_size=%d&customer_id=%s", ...)
    // Sends nil body
}
```

## Solution

Refactor all 14 methods to follow the correct pattern established by `BillingV1BillingGets`:
- Only pagination parameters (`page_token`, `page_size`) remain in query string
- All filter criteria moved to request body as JSON-marshaled `map[Field]any`
- Method signatures updated to accept filters map instead of individual parameters

Correct pattern:
```go
func CampaignV1CampaignGets(ctx context.Context, pageToken string, pageSize uint64, filters map[cacampaign.Field]any) ([]Campaign, error) {
    uri := fmt.Sprintf("/v1/campaigns?page_token=%s&page_size=%d", ...)
    m, err := json.Marshal(filters)
    if err != nil {
        return nil, errors.Wrapf(err, "could not marshal filters")
    }
    tmp, err := r.sendRequestCampaign(ctx, uri, sock.RequestMethodGet, "campaign/campaigns", requestTimeoutDefault, 0, ContentTypeJSON, m)
    // ...
}
```

## Impact Scope

- **Phase 1:** Update 14 methods in `bin-common-handler/pkg/requesthandler/` (12 files)
- **Phase 2:** Update all callers across the monorepo (20-30+ services)
- **Phase 3:** Run full test suite and verification workflow

## Methods Requiring Refactoring

Total: 14 methods across 12 files

### Group 1: Standard GetsByCustomerID Pattern (7 methods)

1. **campaign_campaigns.go:74** - `CampaignV1CampaignGetsByCustomerID`
2. **campaign_outplans.go:72** - `CampaignV1OutplanGetsByCustomerID`
3. **outdial_outdials.go:57** - `OutdialV1OutdialGetsByCustomerID`
4. **message_message.go:51** - `MessageV1MessageGets`
5. **tag_tag.go:114** - `TagV1TagGets`
6. **route_routes.go:138** - `RouteV1RouteGetsByCustomerID`
7. **campaign_campaigncalls.go:18** - `CampaignV1CampaigncallGets`

All have single `customer_id` filter in query string that moves to filters map.

### Group 2: GetsByCampaignID Pattern (1 method)

8. **campaign_campaigncalls.go:37** - `CampaignV1CampaigncallGetsByCampaignID`

Note: This merges with the existing `CampaignV1CampaigncallGets` from Group 1 into one generic method.

### Group 3: Multiple Filter Parameters (3 methods)

9. **nunmber_available_numbers.go:16** - `NumberV1AvailableNumberGets`
   - Params: `customer_id`, `country_code`, `page_size`

10. **route_dialroutes.go:18** - `RouteV1DialrouteGets`
    - Params: `customer_id`, `target`
    - No pagination

11. **registrar_contact.go:17** - `RegistrarV1ContactGets`
    - Params: `customer_id`, `extension`
    - No pagination

### Group 4: Non-GET Methods (1 method)

12. **registrar_contact.go:35** - `RegistrarV1ContactRefresh`
    - Method: PUT (not GET)
    - Params: `customer_id`, `extension`

### Group 5: Nested Resource Patterns (2 methods)

13. **queue_queue.go:196** - `QueueV1QueueGetAgents`
    - Path: `/v1/queues/{id}/agents?status={status}`
    - Path parameter `{id}` stays in URL, `status` moves to body

14. **registrar_extensions.go:144** - `RegistrarV1ExtensionGetByExtension`
    - Path: `/v1/extensions/{extension}?customer_id={customerID}`
    - Path parameter `{extension}` stays in URL, `customer_id` moves to body

## Key Design Decisions

1. **Breaking Change Strategy:** Update all signatures immediately, not gradual deprecation
2. **Backend Compatibility:** Backend services already support body-based filters
3. **Field Types:** All models already have Field type definitions
4. **Path Parameters:** IDs in URL path remain in URL, only query parameters move to body
5. **Error Handling:** Add marshaling error case, otherwise identical to current pattern

## Benefits

- Consistency with existing correct implementations
- More flexible filtering (callers can filter by any field, not just hardcoded parameters)
- Backend services already support this pattern
- Cleaner API design (filters belong in body, not URL)

## Error Handling

New error case for filter marshaling:
```go
m, err := json.Marshal(filters)
if err != nil {
    return nil, errors.Wrapf(err, "could not marshal filters")
}
```

All other error handling remains unchanged.

## Testing Strategy

### Phase 1: Unit Tests for requesthandler methods
- Update mock expectations in *_test.go files
- Verify filters are correctly marshaled to JSON body
- Test with various filter combinations

### Phase 2: Integration Tests
- Run full test suite: `go test ./...` in bin-common-handler
- Ensure no regressions

### Phase 3: Caller Updates & Testing
- Find all callers across monorepo
- Update caller code to pass filters map
- Run each service's test suite

### Phase 4: Full Monorepo Verification
- Run complete workflow for ALL services
- Verify compilation and tests pass

## Implementation Steps

### Step 1: Update bin-common-handler requesthandler methods
Modify 12 files, refactor 14 methods to use body-based filters.

### Step 2: Update requesthandler tests
Update corresponding *_test.go files with new mock expectations.

### Step 3: Verify bin-common-handler
```bash
cd bin-common-handler
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

### Step 4: Find all callers across monorepo
Search for each old method name to identify all callers.

### Step 5: Update all callers
Convert caller code pattern:
```go
// Before
campaigns, err := reqHandler.CampaignV1CampaignGetsByCustomerID(ctx, customerID, pageToken, pageSize)

// After
filters := map[cacampaign.Field]any{
    cacampaign.FieldCustomerID: customerID,
}
campaigns, err := reqHandler.CampaignV1CampaignGets(ctx, pageToken, pageSize, filters)
```

### Step 6: Update all affected services
Run verification workflow for each modified service.

### Step 7: Commit strategy
Option A: Two commits
- Commit 1: Update bin-common-handler
- Commit 2: Update all callers

Option B: Single atomic commit with both changes

## Risks & Mitigations

**Risk:** Missing callers during updates
**Mitigation:** Use comprehensive grep search across monorepo

**Risk:** Test failures in dependent services
**Mitigation:** Run full test suite before committing

**Risk:** Breaking changes cause compilation errors
**Mitigation:** This is expected and intended; fix all compilation errors before committing

## Success Criteria

- All 14 methods refactored to use body-based filters
- All callers updated to new signatures
- All tests passing across monorepo
- No compilation errors
- Consistent pattern across all requesthandler list methods
