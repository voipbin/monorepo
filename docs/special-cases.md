# Special Cases

> **Quick Reference:** For triggers, see [CLAUDE.md](../CLAUDE.md#critical-before-committing-changes)

This document covers special scenarios that require updating multiple services across the monorepo.

## Changes to bin-common-handler

**🚨 CRITICAL: If you make ANY changes to `bin-common-handler`, you MUST update ALL 30+ services in the monorepo.**

The `bin-common-handler` is a shared library used by ALL other services. Changes to it require updating every service to maintain consistency across the entire monorepo.

### What Counts as a "Change to bin-common-handler"

Changes that affect dependent services include:
- ✅ Adding new functions or methods (e.g., `PrepareFields()`, `ScanRow()`)
- ✅ Modifying existing interfaces or function signatures
- ✅ Adding/removing fields in shared models (`models/outline`, `models/identity`)
- ✅ Changing behavior of existing handlers
- ✅ Updating dependencies in bin-common-handler's `go.mod`

### Complete Update Workflow

**Step 1: Make changes to bin-common-handler**
```bash
cd bin-common-handler

# Make your changes, then run verification
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

**Step 2: Update ALL services in monorepo**
```bash
# From monorepo root, update all 30+ services
cd /home/pchero/gitvoipbin/monorepo

find . -maxdepth 2 -name "go.mod" -execdir bash -c \
  "echo 'Updating \$(basename \$(pwd))...' && \
   go mod tidy && \
   go mod vendor && \
   go generate ./... && \
   go clean -testcache && \
   go test ./..." \;
```

**What this does for EACH service:**
1. `go mod tidy` - Syncs go.mod with bin-common-handler changes (NOT `go get -u`)
2. `go mod vendor` - Updates vendored bin-common-handler code
3. `go generate ./...` - Regenerates mocks that depend on bin-common-handler interfaces
4. **`go clean -testcache`** - **CRITICAL**: Clears test cache to force actual test execution
5. `go test ./...` - Verifies the service still works with new bin-common-handler

**⚠️ WARNING: ALWAYS use `go clean -testcache` when testing after bin-common-handler changes!**
- Go caches test results, which can hide failures introduced by your changes
- Cached tests show "PASS" even though they never actually re-ran with the new code
- This is the #1 cause of missing test failures in the monorepo

**Step 3: Verify key services compile**
```bash
# Spot check critical services
cd bin-api-manager && go build ./...
cd ../bin-call-manager && go build ./...
cd ../bin-flow-manager && go build ./...
```

**Step 4: Commit ALL changes together**
```bash
cd /home/pchero/gitvoipbin/monorepo

# Commit bin-common-handler changes
git add bin-common-handler/
git commit -m "feat(common-handler): add new database mapping utilities"

# Commit dependency updates for all services
git add */go.mod */go.sum
git commit -m "chore: update dependencies after bin-common-handler changes"
```

### Projects Affected

**All services depend on bin-common-handler** (30+ projects):
- bin-api-manager (REST API gateway)
- bin-call-manager (Call routing)
- bin-flow-manager (Flow execution)
- bin-conference-manager (Conferencing)
- bin-ai-manager (AI integration)
- bin-webhook-manager (Webhooks)
- bin-agent-manager (Agent management)
- bin-customer-manager (Customer data)
- And 22+ other services...

### Why This Is Critical

- **Compilation errors**: Stale vendored dependencies will break builds
- **Runtime failures**: Services may use outdated interfaces/models
- **Test failures**: Mock regeneration required if interfaces changed
- **Type mismatches**: Shared utility changes affect test expectations, not just interfaces
- **Test cache masking failures**: Cached tests can hide breaking changes
- **Inconsistent state**: Half-updated monorepo leads to subtle bugs

### Verification Checklist

After changing bin-common-handler, verify EACH dependent service:

- [ ] **`go mod tidy`** - Dependencies synced
- [ ] **`go mod vendor`** - Vendored code updated
- [ ] **`go generate ./...`** - Mocks regenerated
- [ ] **`go clean -testcache`** - Test cache cleared ⚠️ NEVER SKIP THIS
- [ ] **`go test ./...`** - All tests pass (actually ran, not cached)
- [ ] **Test expectations updated** - If types/values changed in shared utilities
- [ ] **`golangci-lint run`** - No linting issues
- [ ] **`go build ./...`** - Compiles successfully

**If ANY test fails:**
1. Don't just regenerate mocks and retry
2. Investigate WHAT changed in bin-common-handler
3. Update test EXPECTATIONS to match new behavior
4. Clear cache and re-run: `go clean -testcache && go test ./...`

### Common Mistakes to Avoid

❌ **DON'T run tests without clearing cache first** - `go test` uses cached results, hiding failures
❌ **DON'T just regenerate mocks and assume tests will pass** - Test EXPECTATIONS need updates for type changes
❌ **DON'T run `go get -u`** - This updates ALL dependencies, not just bin-common-handler
❌ **DON'T skip `go generate`** - Mocks will be stale if interfaces changed
❌ **DON'T skip `go test`** - Won't catch breaking changes until production
❌ **DON'T commit bin-common-handler alone** - All services must be updated together

✅ **DO run `go clean -testcache` before testing** - Force actual test execution, not cached results
✅ **DO update test expectations when shared utilities change types** - Especially for type conversions
✅ **DO run the complete workflow** - `go mod tidy && go mod vendor && go generate && go clean -testcache && go test`
✅ **DO verify tests ACTUALLY ran** (not cached) for all services before committing
✅ **DO commit bin-common-handler and dependency updates together** (can be separate commits in same PR)

### Troubleshooting

**If services fail to compile after update:**
```bash
# Clean and retry for specific service
cd bin-<service-name>
rm -rf vendor/
go clean -modcache
go mod tidy && go mod vendor && go build ./...
```

**If tests fail in multiple services after bin-common-handler changes:**

🚨 **CRITICAL: Test failures are NOT just about regenerating mocks!**

When bin-common-handler shared utilities change (especially type conversions, data transformations, or filter handling), test EXPECTATIONS must be updated to match the new behavior.

**Common scenario (like the UUID filter bug):**
1. You modify a shared utility in bin-common-handler (e.g., database field type conversion)
2. This changes the TYPES of values passed to mocked functions (e.g., string → uuid.UUID)
3. Mock expectations in tests still expect the OLD types
4. Tests fail with "expected call doesn't match" errors
5. **Solution**: Update test expectations to use the new types, NOT just regenerate mocks

**Example - What to fix when type conversions change:**
```go
// ❌ WRONG - Old test expectation (before fix)
expectFilters: map[Field]any{
    FieldCustomerID: "5e4a0680-804e-11ec-8477-2fea5968d85b",  // String
}

// ✅ CORRECT - Updated test expectation (after UUID conversion fix)
expectFilters: map[Field]any{
    FieldCustomerID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),  // UUID type
}

// Alternative: Use gomock.Any() if exact matching is problematic
mockReq.EXPECT().SomeMethod(ctx, gomock.Any()).Return(response, nil)
```

**Systematic approach to fixing test failures:**
1. **Identify the changed behavior** in bin-common-handler
2. **Find all test files** with failing expectations (`grep -r "FieldCustomerID" *_test.go`)
3. **Update test expectations** to match new types/behavior
4. **Re-run with clean cache**: `go clean -testcache && go test ./...`
5. **Verify actual execution**: Check test ran (not cached)

**Common changes requiring test expectation updates:**
- ✅ Type conversions (string → UUID, string → int, etc.)
- ✅ Filter map value types
- ✅ Field name changes or additions
- ✅ Enum value changes
- ✅ Default value changes

**DON'T just regenerate mocks and assume tests will pass!**
- Mock interfaces may be unchanged, but VALUES passed have changed types
- Test expectations must be manually updated to match new value types

**Never commit changes to bin-common-handler without updating dependent services.**

## Changes to Public-Facing Models and OpenAPI Schemas {#openapi-sync}

**CRITICAL: If you modify public-facing data structures in ANY service, you MUST verify and update the corresponding OpenAPI schemas in `bin-openapi-manager`.**

### What are public-facing models?

Services expose data to external APIs, webhooks, and events through specific structs:
- `WebhookMessage` structs (e.g., `call.WebhookMessage`, `conference.WebhookMessage`)
- API response models that are returned to external clients
- Any data structure used in RPC responses to `bin-api-manager`

### The Rule

When a service defines what data is exposed publicly (via `WebhookMessage` or similar), the corresponding OpenAPI schema in `bin-openapi-manager/openapi/openapi.yaml` MUST accurately reflect all fields.

**Example Mapping:**
```
bin-call-manager/models/call/webhook.go → WebhookMessage struct
    ↓ maps to ↓
bin-openapi-manager/openapi/openapi.yaml → CallManagerCall schema

bin-conference-manager/models/conference/webhook.go → WebhookMessage struct
    ↓ maps to ↓
bin-openapi-manager/openapi/openapi.yaml → ConferenceManagerConference schema
```

### Validation Process

1. **Before modifying a WebhookMessage or public model:**
   - Note all fields currently in the struct

2. **After making changes:**
   - Identify the corresponding OpenAPI schema in `bin-openapi-manager/openapi/openapi.yaml`
   - Compare fields in the Go struct vs. OpenAPI schema
   - Add any missing fields to the OpenAPI schema
   - Remove deprecated fields from the OpenAPI schema
   - Ensure field types match (string, array, object, etc.)
   - Ensure enums are properly defined

3. **Regenerate OpenAPI code:**
```bash
cd bin-openapi-manager
go generate ./...
```

4. **Update dependent services:**
```bash
cd bin-api-manager
go mod tidy && go mod vendor && go generate ./...
```

### Why this is critical

- `bin-openapi-manager` is the source of truth for the public REST API contract
- External API consumers rely on accurate OpenAPI documentation
- API documentation (Swagger UI) must match actual service behavior
- Type mismatches cause runtime errors and API consumer confusion

### Common scenarios requiring validation

- Adding new fields to a model
- Changing field types or nullability
- Adding new enum values
- Renaming fields (breaking change - coordinate carefully)
- Removing deprecated fields

**Never commit changes to public-facing models without verifying OpenAPI schemas are synchronized.**
