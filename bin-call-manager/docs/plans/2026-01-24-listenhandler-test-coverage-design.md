# Listenhandler Test Coverage Improvement

**Date:** 2026-01-24
**Goal:** Increase listenhandler test coverage from 67% to 80%

## Current State

- **67% statement coverage** in `pkg/listenhandler/`
- **61 test functions** covering most endpoints
- **main.go untested** - No test file exists
- **2 endpoints missing tests**

## Gap Analysis

The 33% coverage gap comes from:

1. **main.go functions (0% coverage)**
   - `NewListenHandler()` - constructor
   - `simpleResponse()` - utility function
   - `Run()` - queue setup (harder to test, involves RabbitMQ)
   - `processRequest()` - routing switch statement

2. **Missing endpoint tests**
   - `PUT /calls/<id>/confbridge_id` (`processV1CallsIDConfbridgeIDPut`)
   - `GET /confbridges/<id>` (`processV1ConfbridgesIDGet`)

3. **Error paths in existing handlers**
   - Malformed JSON parsing
   - Invalid UUID formats
   - Wrong URI segment counts

## Test Plan

### Priority 1: main_test.go (biggest coverage gain)

Create `pkg/listenhandler/main_test.go` with:

| Function | Test Cases |
|----------|------------|
| `NewListenHandler()` | Returns valid handler with all dependencies injected |
| `simpleResponse()` | Returns correct status codes (200, 400, 404, 500) |
| `processRequest()` | Routes calls endpoints correctly |
| `processRequest()` | Routes confbridge endpoints correctly |
| `processRequest()` | Routes external-media endpoints correctly |
| `processRequest()` | Routes groupcall endpoints correctly |
| `processRequest()` | Routes recording endpoints correctly |
| `processRequest()` | Routes recovery endpoint correctly |
| `processRequest()` | Returns 404 for unknown URIs |
| `processRequest()` | Returns 400 when handler returns error |

### Priority 2: Missing endpoint tests

**v1_calls_test.go** - Add:
```go
func Test_processV1CallsIDConfbridgeIDPut(t *testing.T)
```
Test cases:
- Normal update with valid confbridge_id
- Invalid call UUID format
- Malformed JSON body

**v1_confbridge_test.go** - Add:
```go
func Test_processV1ConfbridgesIDGet(t *testing.T)
```
Test cases:
- Normal get returns confbridge
- Invalid confbridge UUID format

### Priority 3: Error path coverage (if needed)

Add error cases to existing tests:
- JSON unmarshal errors
- URI parsing errors with insufficient segments

## Implementation

### Files to Create
- `pkg/listenhandler/main_test.go`

### Files to Modify
- `pkg/listenhandler/v1_calls_test.go`
- `pkg/listenhandler/v1_confbridge_test.go`

### Test Patterns

Follow existing codebase patterns:
- Table-driven tests with `[]struct{}`
- gomock for interface mocking
- `gomock.NewController(t)` with `defer mc.Finish()`
- Test function naming: `Test_<functionName>` or `Test<FunctionName>`

### Verification

```bash
cd bin-call-manager
go test -cover ./pkg/listenhandler/...
```

Target: 80%+ coverage

## Estimated Impact

| Component | Before | After |
|-----------|--------|-------|
| main.go | 0% | ~80% |
| v1_calls.go | ~95% | ~98% |
| v1_confbridge.go | ~90% | ~95% |
| **Overall** | **67%** | **~80%** |

## Out of Scope

- `Run()` function (requires RabbitMQ integration testing)
- `models/request/` and `models/response/` packages (simple DTOs, low value)
- Startup validation (separate improvement)
