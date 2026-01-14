# ListenHandler Filter Parsing Fix - Implementation Plan

**Date:** 2026-01-14
**Related Design:** 2026-01-14-listenhandler-filter-parsing-fix.md

## Overview

This plan details the step-by-step implementation of the filter parsing fix across the monorepo.

## Phase 1: Core Infrastructure (bin-common-handler)

### Step 1.1: Create filters.go file

**File:** `bin-common-handler/pkg/utilhandler/filters.go`

**Tasks:**
- [ ] Create new file
- [ ] Add package declaration and imports
- [ ] Implement `ParseFiltersFromRequestBody(data []byte) (map[string]any, error)`
- [ ] Implement `ConvertFilters[FS any, F ~string](fieldStruct FS, filters map[string]any) (map[F]any, error)`
- [ ] Implement `convertValueToType(value any, targetType reflect.Type) (any, error)` helper

**Dependencies:** None

**Verification:**
```bash
cd bin-common-handler
go build ./pkg/utilhandler/...
```

### Step 1.2: Update UtilHandler interface

**File:** `bin-common-handler/pkg/utilhandler/main.go`

**Tasks:**
- [ ] Add `ParseFiltersFromRequestBody(data []byte) (map[string]any, error)` to interface
- [ ] Add `ConvertFilters[FS any, F ~string](fieldStruct FS, filters map[string]any) (map[F]any, error)` to interface
- [ ] Implement methods in `utilHandler` struct (call standalone functions)

**Verification:**
```bash
cd bin-common-handler
go build ./pkg/utilhandler/...
```

### Step 1.3: Generate mocks

**Tasks:**
- [ ] Run `go generate ./pkg/utilhandler/`
- [ ] Verify `mock_main.go` updated with new methods

**Verification:**
```bash
cd bin-common-handler/pkg/utilhandler
go generate .
git diff mock_main.go
```

### Step 1.4: Write unit tests

**File:** `bin-common-handler/pkg/utilhandler/filters_test.go`

**Tasks:**
- [ ] Test `ParseFiltersFromRequestBody` with:
  - Empty data
  - Valid JSON
  - Invalid JSON
  - Complex nested structures
- [ ] Test `ConvertFilters` with:
  - UUID string → uuid.UUID conversion
  - Bool passthrough
  - String passthrough
  - Number conversions (JSON float64 → int)
  - Unknown filters (should be ignored)
  - Invalid UUID format (should error)
  - Type mismatches

**Verification:**
```bash
cd bin-common-handler
go test ./pkg/utilhandler/... -v -run Test_ParseFiltersFromRequestBody
go test ./pkg/utilhandler/... -v -run Test_ConvertFilters
```

### Step 1.5: Run full verification workflow

**Tasks:**
- [ ] `go mod tidy`
- [ ] `go mod vendor`
- [ ] `go generate ./...`
- [ ] `go test ./...`
- [ ] `golangci-lint run -v --timeout 5m`

**Verification:**
```bash
cd bin-common-handler
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

## Phase 2: Update Services (13+ services)

### Service Update Pattern

For each service, follow this pattern:

#### Step 2.X.1: Create FieldStruct definition

**File:** `bin-<service>/models/<resource>/filters.go`

**Tasks:**
- [ ] Create new file
- [ ] Define `FieldStruct` with `filter:` tags
- [ ] Include all filterable fields from existing models

**Example structure:**
```go
package <resource>

import "github.com/gofrs/uuid"

type FieldStruct struct {
    CustomerID uuid.UUID `filter:"customer_id"`
    Deleted    bool      `filter:"deleted"`
    // ... other fields
}
```

#### Step 2.X.2: Update listenhandler list methods

**File:** `bin-<service>/pkg/listenhandler/v1_<resource>s.go`

**Tasks:**
- [ ] Locate `processV1<Resource>sGet` function
- [ ] Keep pagination parsing from URL (unchanged)
- [ ] Replace filter parsing:
  - Remove: `filters := getFilters(u)` or `filters := h.utilHandler.URLParseFilters(u)`
  - Add: Parse from body and convert using FieldStruct
- [ ] Update error handling

**Pattern:**
```go
// OLD:
filters := h.utilHandler.URLParseFilters(u)
typedFilters := convertToTypedFilters(filters)

// NEW:
tmpFilters, err := h.utilHandler.ParseFiltersFromRequestBody(m.Data)
if err != nil {
    log.Errorf("Could not parse filters. err: %v", err)
    return simpleResponse(400), nil
}
typedFilters, err := h.utilHandler.ConvertFilters[<resource>.FieldStruct, <resource>.Field](
    <resource>.FieldStruct{},
    tmpFilters,
)
if err != nil {
    log.Errorf("Could not convert filters. err: %v", err)
    return simpleResponse(400), nil
}
```

#### Step 2.X.3: Remove old converter functions (optional)

**File:** `bin-<service>/pkg/listenhandler/main.go` or similar

**Tasks:**
- [ ] Remove `getFilters(u *url.URL)` if it exists (ai-manager pattern)
- [ ] Remove `convertTo<Resource>Filters(filters map[string]string)` functions
- [ ] Keep any functions that do custom business logic beyond type conversion

#### Step 2.X.4: Update unit tests

**File:** `bin-<service>/pkg/listenhandler/v1_<resource>s_test.go`

**Tasks:**
- [ ] Update `Test_processV1<Resource>sGet` to:
  - Pass filters in request body (m.Data) instead of URL
  - Verify error handling for invalid filter JSON
  - Verify type conversions work correctly

#### Step 2.X.5: Run service verification workflow

**Tasks:**
- [ ] Update dependencies: `go mod tidy && go mod vendor`
- [ ] Regenerate mocks: `go generate ./...`
- [ ] Run tests: `go test ./...`
- [ ] Run linter: `golangci-lint run -v --timeout 5m`

**Verification:**
```bash
cd bin-<service>
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

### Service-Specific Details

#### 2.1: bin-ai-manager

**Resources to update:**
- [ ] `models/ai/filters.go` (FieldStruct)
- [ ] `models/aicall/filters.go` (FieldStruct)
- [ ] `models/message/filters.go` (FieldStruct)
- [ ] `models/summary/filters.go` (FieldStruct)

**Listenhandlers to update:**
- [ ] `pkg/listenhandler/v1_ais.go` → `processV1AIsGet`
- [ ] `pkg/listenhandler/v1_aicalls.go` → `processV1AIcallsGet`
- [ ] `pkg/listenhandler/v1_messages.go` → `processV1MessagesGet`
- [ ] `pkg/listenhandler/v1_summaries.go` → `processV1SummariesGet`

**Remove:**
- [ ] `pkg/listenhandler/main.go` → `getFilters(u *url.URL)`
- [ ] `pkg/listenhandler/main.go` → `convertToAIFilters()`
- [ ] `pkg/listenhandler/main.go` → `convertToAIcallFilters()`
- [ ] `pkg/listenhandler/main.go` → `convertToMessageFilters()`
- [ ] `pkg/listenhandler/main.go` → `convertToSummaryFilters()`

#### 2.2: bin-call-manager

**Resources to update:**
- [ ] `models/call/filters.go` (FieldStruct)
- [ ] `models/recording/filters.go` (FieldStruct)
- [ ] `models/externalmedia/filters.go` (FieldStruct)
- [ ] `models/groupcall/filters.go` (FieldStruct)

**Listenhandlers to update:**
- [ ] `pkg/listenhandler/v1_calls.go` → `processV1CallsGet`
- [ ] `pkg/listenhandler/v1_recordings.go` → `processV1RecordingsGet`
- [ ] `pkg/listenhandler/v1_external_medias.go` → `processV1ExternalMediasGet`
- [ ] `pkg/listenhandler/v1_groupcalls.go` → `processV1GroupcallsGet`

#### 2.3: bin-conference-manager

**Resources to update:**
- [ ] `models/conference/filters.go` (FieldStruct)
- [ ] `models/conferencecall/filters.go` (FieldStruct)

**Listenhandlers to update:**
- [ ] `pkg/listenhandler/v1_conferences.go` → `processV1ConferencesGet`
- [ ] `pkg/listenhandler/v1_conferencecalls.go` → `processV1ConferencecallsGet`

**Note:** Conference-manager already has `ConvertStringMapToFieldMap()` - can be replaced

#### 2.4: bin-billing-manager

**Resources to update:**
- [ ] `models/account/filters.go` (FieldStruct)
- [ ] `models/billing/filters.go` (FieldStruct)

**Listenhandlers to update:**
- [ ] `pkg/listenhandler/v1_accounts.go` → `processV1AccountsGet`
- [ ] `pkg/listenhandler/v1_billings.go` → `processV1BillingsGet`

**Note:** Check for `urlFiltersToBillingFilters` function

#### 2.5: bin-message-manager

**Resources to update:**
- [ ] `models/message/filters.go` (FieldStruct)

**Listenhandlers to update:**
- [ ] `pkg/listenhandler/v1_messages.go` → `processV1MessagesGet`

#### 2.6: bin-number-manager

**Resources to update:**
- [ ] `models/number/filters.go` (FieldStruct)
- [ ] `models/availablenumber/filters.go` (FieldStruct)

**Listenhandlers to update:**
- [ ] `pkg/listenhandler/v1_numbers.go` → `processV1NumbersGet`
- [ ] `pkg/listenhandler/v1_available_numbers.go` → `processV1AvailableNumbersGet`

#### 2.7: bin-registrar-manager

**Resources to update:**
- [ ] `models/extension/filters.go` (FieldStruct)
- [ ] `models/astcontact/filters.go` (FieldStruct)

**Listenhandlers to update:**
- [ ] `pkg/listenhandler/v1_extensions.go` → `processV1ExtensionsGet`
- [ ] `pkg/listenhandler/v1_contacts.go` → `processV1ContactsGet`

#### 2.8: bin-customer-manager

**Resources to update:**
- [ ] `models/customer/filters.go` (FieldStruct)
- [ ] `models/accesskey/filters.go` (FieldStruct)

**Listenhandlers to update:**
- [ ] `pkg/listenhandler/v1_customers.go` → `processV1CustomersGet`
- [ ] `pkg/listenhandler/v1_accesskeys.go` → `processV1AccesskeysGet`

#### 2.9: bin-transcribe-manager

**Resources to update:**
- [ ] `models/transcribe/filters.go` (FieldStruct)
- [ ] `models/transcript/filters.go` (FieldStruct)

**Listenhandlers to update:**
- [ ] `pkg/listenhandler/v1_transcribes.go` → `processV1TranscribesGet`
- [ ] `pkg/listenhandler/v1_transcripts.go` → (if exists)

#### 2.10: bin-queue-manager

**Resources to update:**
- [ ] `models/queue/filters.go` (FieldStruct)
- [ ] `models/queuecall/filters.go` (FieldStruct)

**Listenhandlers to update:**
- [ ] `pkg/listenhandler/v1_queues.go` → `processV1QueuesGet`
- [ ] `pkg/listenhandler/v1_queuecalls.go` → `processV1QueuecallsGet`

#### 2.11: bin-agent-manager

**Resources to update:**
- [ ] `models/agent/filters.go` (FieldStruct)

**Listenhandlers to update:**
- [ ] `pkg/listenhandler/v1_agents.go` → `processV1AgentsGet`

#### 2.12: bin-tag-manager

**Resources to update:**
- [ ] `models/tag/filters.go` (FieldStruct)

**Listenhandlers to update:**
- [ ] `pkg/listenhandler/v1_tags.go` → `processV1TagsGet`

#### 2.13: bin-conversation-manager

**Resources to update:**
- [ ] `models/account/filters.go` (FieldStruct)
- [ ] `models/conversation/filters.go` (FieldStruct)
- [ ] `models/message/filters.go` (FieldStruct)

**Listenhandlers to update:**
- [ ] `pkg/listenhandler/v1_accounts.go` → `processV1AccountsGet`
- [ ] `pkg/listenhandler/v1_conversations.go` → `processV1ConversationsGet`
- [ ] `pkg/listenhandler/v1_messages.go` → `processV1MessagesGet`

## Phase 3: Final Verification

### Step 3.1: Update all service dependencies

**Tasks:**
- [ ] Run from monorepo root:
  ```bash
  find . -maxdepth 2 -name "go.mod" -execdir bash -c \
    "echo 'Updating $(basename $(pwd))...' && \
     go mod tidy && \
     go mod vendor" \;
  ```

### Step 3.2: Verify all services compile

**Tasks:**
- [ ] Spot check critical services:
  ```bash
  cd bin-ai-manager && go build ./...
  cd ../bin-call-manager && go build ./...
  cd ../bin-conference-manager && go build ./...
  cd ../bin-billing-manager && go build ./...
  ```

### Step 3.3: Run all tests

**Tasks:**
- [ ] Run from monorepo root:
  ```bash
  find . -name "go.mod" -execdir go test ./... \;
  ```

### Step 3.4: Integration testing

**Tasks:**
- [ ] Test requesthandler → listenhandler flow
- [ ] Verify filters are correctly parsed
- [ ] Check database queries receive expected filters
- [ ] Test with various filter types (UUID, bool, string, numbers)

## Phase 4: Git Workflow

### Step 4.1: Check current branch

**Tasks:**
- [ ] Run `git branch --show-current`
- [ ] If on `main`, create feature branch

### Step 4.2: Commit changes

**Strategy:** Separate commits for clarity

**Commit 1: bin-common-handler**
```bash
cd bin-common-handler
git add pkg/utilhandler/filters.go
git add pkg/utilhandler/filters_test.go
git add pkg/utilhandler/main.go
git add pkg/utilhandler/mock_main.go
git commit -m "$(cat <<'EOF'
VOIP-XXXX: Add generic filter parsing from request body

- bin-common-handler: Add ParseFiltersFromRequestBody() to parse JSON filters
  from request body instead of URL query parameters
- bin-common-handler: Add ConvertFilters[FS, F]() generic function for
  type-safe filter conversion using FieldStruct definitions
- bin-common-handler: Add convertValueToType() helper for UUID, bool, string,
  and number conversions
- bin-common-handler: Update UtilHandler interface with new methods
- bin-common-handler: Add comprehensive unit tests for filter parsing

Test results: All tests passing
EOF
)"
```

**Commit 2: Service updates (can be one commit or per-service)**
```bash
git add bin-ai-manager/models/*/filters.go
git add bin-ai-manager/pkg/listenhandler/v1_*.go
git add bin-call-manager/models/*/filters.go
git add bin-call-manager/pkg/listenhandler/v1_*.go
# ... repeat for all services

git commit -m "$(cat <<'EOF'
VOIP-XXXX: Migrate all services to parse filters from request body

Updated 13 services to use new generic filter parsing pattern from
bin-common-handler. Filters are now parsed from request body JSON instead
of URL query parameters, matching the requesthandler implementation.

- bin-ai-manager: Add FieldStruct for ai, aicall, message, summary models
- bin-ai-manager: Update list handlers to parse filters from request body
- bin-ai-manager: Remove old getFilters() and converter functions
- bin-call-manager: Add FieldStruct for call, recording, externalmedia,
  groupcall models
- bin-call-manager: Update list handlers to use new filter parsing
- bin-conference-manager: Add FieldStruct for conference, conferencecall models
- bin-conference-manager: Replace ConvertStringMapToFieldMap with generic version
- bin-billing-manager: Add FieldStruct and update list handlers
- bin-message-manager: Add FieldStruct and update list handlers
- bin-number-manager: Add FieldStruct and update list handlers
- bin-registrar-manager: Add FieldStruct and update list handlers
- bin-customer-manager: Add FieldStruct and update list handlers
- bin-transcribe-manager: Add FieldStruct and update list handlers
- bin-queue-manager: Add FieldStruct and update list handlers
- bin-agent-manager: Add FieldStruct and update list handlers
- bin-tag-manager: Add FieldStruct and update list handlers
- bin-conversation-manager: Add FieldStruct and update list handlers

Test results: All 13 services passing
EOF
)"
```

**Commit 3: Dependency updates**
```bash
git add */go.mod */go.sum
git commit -m "chore: update dependencies after filter parsing changes"
```

### Step 4.3: Run final verification before push

**Tasks:**
- [ ] From monorepo root, run verification on all changed services
- [ ] Ensure no lint errors
- [ ] Ensure all tests pass

### Step 4.4: Push to remote

**Tasks:**
- [ ] Push branch: `git push -u origin <branch-name>`
- [ ] Create pull request if needed

## Rollback Plan

If critical issues are discovered:

1. **Revert listenhandler changes:**
   ```bash
   git revert <commit-hash>
   ```

2. **Emergency hotfix pattern:**
   - Restore old URL-based filter parsing in affected services
   - Keep pagination from URL
   - Add temporary backward compatibility layer

3. **Full rollback:**
   - Revert all commits in reverse order
   - Re-vendor dependencies
   - Redeploy services

## Success Checklist

- [ ] bin-common-handler: New filter parsing functions implemented and tested
- [ ] All 13 services: FieldStruct definitions created
- [ ] All 13 services: List handlers updated to parse from request body
- [ ] All 13 services: Old converter functions removed
- [ ] All 13 services: Unit tests updated and passing
- [ ] All services compile successfully
- [ ] All tests pass
- [ ] Integration tests verify correct filter behavior
- [ ] Lint checks pass
- [ ] Documentation updated
- [ ] Changes committed with proper commit messages
- [ ] Changes pushed to remote

## Time Estimate

- Phase 1 (bin-common-handler): 2-3 hours
- Phase 2 (13 services): 30-45 minutes per service = 6-10 hours
- Phase 3 (verification): 1-2 hours
- Phase 4 (git workflow): 30 minutes

**Total: 10-16 hours**

## Notes

- Services can be updated in parallel by different developers
- Each service update is independent once bin-common-handler is done
- Test as you go - don't wait until the end
- Keep commits atomic and well-documented
