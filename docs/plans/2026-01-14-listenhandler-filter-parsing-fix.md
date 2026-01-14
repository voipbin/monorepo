# ListenHandler Filter Parsing Fix - Design Document

**Date:** 2026-01-14
**Status:** Approved
**Author:** Claude Code

## Problem Statement

The requesthandler in `bin-common-handler` was recently changed to send filters in the **request body** (as JSON) instead of URL query parameters. However, all listenhandler implementations across 13+ services are still parsing filters from URL query parameters with `filter_` prefix, resulting in empty filters being passed to database queries.

### Current Broken Flow

1. **Sender (requesthandler)**: Constructs URI with pagination only: `/v1/ais?page_token=X&page_size=Y`
2. **Sender (requesthandler)**: Marshals filters to JSON and puts in `m.Data` (request body)
3. **Receiver (listenhandler)**: ✅ Parses pagination from URL query params correctly
4. **Receiver (listenhandler)**: ❌ Tries to parse filters from URL query params → Gets empty filters!

### Example

**requesthandler sends:**
```go
uri := fmt.Sprintf("/v1/ais?page_token=%s&page_size=%d", pageToken, pageSize)
m, _ := json.Marshal(filters) // filters = map[ai.Field]any{"customer_id": uuid, "deleted": false}
r.sendRequestAI(ctx, uri, sock.RequestMethodGet, "ai/ais", 30000, 0, ContentTypeJSON, m)
```

**listenhandler receives:**
```go
// Pagination: OK
pageSize := uint64(tmpSize)  // ✅ Correctly parsed from URL
pageToken := u.Query().Get(PageToken)  // ✅ Correctly parsed from URL

// Filters: BROKEN
filters := getFilters(u)  // ❌ Returns empty map - no filter_* params in URL!
```

## Affected Services

13+ services have list endpoints that need updating:

1. **bin-ai-manager** - `/v1/ais`, `/v1/aicalls`, `/v1/messages`, `/v1/summaries`
2. **bin-call-manager** - `/v1/calls`, `/v1/recordings`, `/v1/external-medias`, `/v1/groupcalls`
3. **bin-conference-manager** - `/v1/conferences`, `/v1/conferencecalls`
4. **bin-billing-manager** - `/v1/accounts`, `/v1/billings`
5. **bin-message-manager** - `/v1/messages`
6. **bin-number-manager** - `/v1/numbers`, `/v1/available-numbers`
7. **bin-registrar-manager** - `/v1/contacts`, `/v1/extensions`
8. **bin-customer-manager** - `/v1/accesskeys`, `/v1/customers`
9. **bin-transcribe-manager** - `/v1/transcribes`, `/v1/transcripts`
10. **bin-queue-manager** - `/v1/queues`, `/v1/queuecalls`
11. **bin-agent-manager** - `/v1/agents`
12. **bin-tag-manager** - `/v1/tags`
13. **bin-conversation-manager** - `/v1/accounts`, `/v1/conversations`, `/v1/messages`

## Solution Design

### Overview

Introduce a struct-based generic filter parser that:
1. Parses filters from request body JSON
2. Validates filters against an explicit FieldStruct definition
3. Converts filter values to correct types based on struct field types
4. Returns typed `map[Field]any` for database queries

### Architecture

```
Request Body (JSON)
    ↓
ParseFiltersFromRequestBody() → map[string]any
    ↓
ConvertFilters[FieldStruct, Field]() → map[Field]any
    ↓
Database Handler Gets()
```

### Component 1: ParseFiltersFromRequestBody

**Location:** `bin-common-handler/pkg/utilhandler/filters.go` (new file)

**Purpose:** Unmarshal JSON filters from request body to generic map

**Signature:**
```go
func ParseFiltersFromRequestBody(data []byte) (map[string]any, error)
```

**Implementation:**
```go
func ParseFiltersFromRequestBody(data []byte) (map[string]any, error) {
    if len(data) == 0 {
        return map[string]any{}, nil
    }

    var filters map[string]any
    if err := json.Unmarshal(data, &filters); err != nil {
        return nil, errors.Wrap(err, "could not unmarshal filters from request body")
    }

    return filters, nil
}
```

### Component 2: ConvertFilters (Generic)

**Location:** `bin-common-handler/pkg/utilhandler/filters.go` (new file)

**Purpose:** Convert and validate filters using FieldStruct definition

**Signature:**
```go
func ConvertFilters[FS any, F ~string](fieldStruct FS, filters map[string]any) (map[F]any, error)
```

**Parameters:**
- `FS`: FieldStruct type that defines allowed filters (e.g., `ai.FieldStruct`)
- `F`: Field type for keys (e.g., `ai.Field`)
- `fieldStruct`: Instance of FieldStruct (can be zero value)
- `filters`: Raw filters from request body

**Implementation:**
```go
func ConvertFilters[FS any, F ~string](fieldStruct FS, filters map[string]any) (map[F]any, error) {
    result := make(map[F]any)

    // Get FieldStruct type
    typ := reflect.TypeOf(fieldStruct)
    if typ.Kind() == reflect.Ptr {
        typ = typ.Elem()
    }

    // Loop through all fields in FieldStruct
    for i := 0; i < typ.NumField(); i++ {
        field := typ.Field(i)

        // Get filter key from tag
        filterKey := field.Tag.Get("filter")
        if filterKey == "" || filterKey == "-" {
            continue
        }

        // Check if this filter exists in received data
        filterValue, exists := filters[filterKey]
        if !exists {
            continue
        }

        // Convert value based on FieldStruct field type
        converted, err := convertValueToType(filterValue, field.Type)
        if err != nil {
            return nil, errors.Wrapf(err, "could not convert filter %s", filterKey)
        }

        // Add to result with Field key
        result[F(filterKey)] = converted
    }

    return result, nil
}

func convertValueToType(value any, targetType reflect.Type) (any, error) {
    if value == nil {
        return nil, nil
    }

    // UUID conversion
    if targetType.String() == "uuid.UUID" {
        if str, ok := value.(string); ok {
            return uuid.FromString(str)
        }
    }

    // Bool
    if targetType.Kind() == reflect.Bool {
        if b, ok := value.(bool); ok {
            return b, nil
        }
    }

    // String
    if targetType.Kind() == reflect.String {
        if str, ok := value.(string); ok {
            return str, nil
        }
    }

    // Numbers (JSON unmarshals as float64)
    if targetType.Kind() >= reflect.Int && targetType.Kind() <= reflect.Uint64 {
        if f, ok := value.(float64); ok {
            return int64(f), nil
        }
    }

    return value, nil
}
```

### Component 3: FieldStruct Definitions

**Location:** Each service's `models/<resource>/filters.go` (new file)

**Purpose:** Explicitly define allowed filters and their types

**Example - AI Manager:**
```go
// bin-ai-manager/models/ai/filters.go
package ai

import "github.com/gofrs/uuid"

type FieldStruct struct {
    CustomerID uuid.UUID `filter:"customer_id"`
    Name       string    `filter:"name"`
    Deleted    bool      `filter:"deleted"`
    EngineType string    `filter:"engine_type"`
    EngineModel string   `filter:"engine_model"`
}
```

**Example - Call Manager:**
```go
// bin-call-manager/models/call/filters.go
package call

import "github.com/gofrs/uuid"

type FieldStruct struct {
    CustomerID   uuid.UUID `filter:"customer_id"`
    Status       string    `filter:"status"`
    Deleted      bool      `filter:"deleted"`
    ConfbridgeID uuid.UUID `filter:"confbridge_id"`
    Direction    string    `filter:"direction"`
}
```

### Component 4: Updated ListenHandler Pattern

**Location:** Each service's `pkg/listenhandler/v1_<resource>.go`

**Before (broken):**
```go
func (h *listenHandler) processV1AIsGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
    u, _ := url.Parse(m.URI)

    // Pagination from URL
    pageSize := uint64(tmpSize)
    pageToken := u.Query().Get(PageToken)

    // Filters from URL - BROKEN!
    filters := getFilters(u)  // Empty map!
    typedFilters := convertToAIFilters(filters)

    tmp, err := h.aiHandler.Gets(ctx, pageSize, pageToken, typedFilters)
    // ...
}
```

**After (fixed):**
```go
func (h *listenHandler) processV1AIsGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
    u, _ := url.Parse(m.URI)

    // Pagination from URL (unchanged)
    pageSize := uint64(tmpSize)
    pageToken := u.Query().Get(PageToken)

    // Filters from request body - FIXED!
    tmpFilters, err := h.utilHandler.ParseFiltersFromRequestBody(m.Data)
    if err != nil {
        log.Errorf("Could not parse filters. err: %v", err)
        return simpleResponse(400), nil
    }

    // Convert using FieldStruct
    typedFilters, err := h.utilHandler.ConvertFilters[ai.FieldStruct, ai.Field](
        ai.FieldStruct{},
        tmpFilters,
    )
    if err != nil {
        log.Errorf("Could not convert filters. err: %v", err)
        return simpleResponse(400), nil
    }

    tmp, err := h.aiHandler.Gets(ctx, pageSize, pageToken, typedFilters)
    // ...
}
```

## Benefits

### Security
- **Explicit whitelist**: Only filters defined in FieldStruct are accepted
- **Type safety**: Invalid types are rejected at conversion time
- **No injection**: Validates against struct definition, not arbitrary input

### Maintainability
- **Single implementation**: Generic `ConvertFilters` works for all services
- **Clear contract**: FieldStruct documents allowed filters
- **Type-driven**: Uses Go's type system for conversion

### Developer Experience
- **Two places to update**: Field constants + FieldStruct
- **Compile-time safety**: Generics ensure type correctness
- **Familiar pattern**: Uses struct tags like `json:`, `db:`

## Implementation Plan

### Phase 1: Core Infrastructure (bin-common-handler)
1. Create `pkg/utilhandler/filters.go`
2. Implement `ParseFiltersFromRequestBody()`
3. Implement `ConvertFilters[FS, F]()`
4. Add methods to `UtilHandler` interface
5. Update mock generation
6. Write unit tests
7. Run verification workflow: `go mod tidy && go mod vendor && go generate ./... && go test ./...`

### Phase 2: Update All Services (parallel)
For each of 13+ services:
1. Create `models/<resource>/filters.go` with FieldStruct
2. Update `pkg/listenhandler/v1_<resources>.go` list handlers
3. Remove old `getFilters()` or `convertTo*Filters()` functions
4. Update unit tests
5. Run verification workflow

### Phase 3: Dependency Updates
After bin-common-handler changes:
1. Update all services: `go mod tidy && go mod vendor`
2. Verify compilation: `go build ./...`
3. Run tests: `go test ./...`

### Phase 4: Testing & Validation
1. Unit tests for each service's list endpoint
2. Integration tests: requesthandler → listenhandler
3. Manual testing on dev environment
4. Verify database queries receive correct filters

## Testing Strategy

### Unit Tests

**bin-common-handler tests:**
```go
func Test_ParseFiltersFromRequestBody(t *testing.T) {
    tests := []struct {
        name      string
        data      []byte
        expectRes map[string]any
        expectErr bool
    }{
        {"empty", []byte{}, map[string]any{}, false},
        {"valid json", []byte(`{"customer_id":"uuid","deleted":false}`), map[string]any{...}, false},
        {"invalid json", []byte(`{invalid}`), nil, true},
    }
    // ...
}

func Test_ConvertFilters(t *testing.T) {
    type TestFieldStruct struct {
        CustomerID uuid.UUID `filter:"customer_id"`
        Deleted    bool      `filter:"deleted"`
    }

    tests := []struct {
        name      string
        filters   map[string]any
        expectRes map[string]any
        expectErr bool
    }{
        {"uuid conversion", map[string]any{"customer_id": "valid-uuid"}, map[string]any{...}, false},
        {"bool passthrough", map[string]any{"deleted": true}, map[string]any{"deleted": true}, false},
        {"unknown filter ignored", map[string]any{"unknown": "value"}, map[string]any{}, false},
    }
    // ...
}
```

**Service-level tests:**
Update existing `Test_processV1<Resource>sGet` tests to verify filters are parsed from body.

### Integration Tests

Create roundtrip test:
```go
func Test_RequestHandlerToListenHandler_Filters(t *testing.T) {
    // 1. requesthandler sends filters in body
    // 2. listenhandler receives and parses
    // 3. Verify filters match expected values
}
```

## Migration Path

### No Backward Compatibility Needed
- Internal RPC communication only (not external API)
- Requesthandler already sends filters in body
- Listenhandlers just need to parse from correct location
- Can deploy all services together

### Deployment Order
1. Deploy bin-common-handler changes
2. Update all services (go mod tidy && go mod vendor)
3. Deploy all services (can be done together since requesthandler already changed)

## Rollback Plan

If issues arise:
1. Revert listenhandler changes in affected services
2. Restore old URL-based filter parsing temporarily
3. Revert requesthandler to send filters in URL (previous behavior)

## Success Criteria

- [ ] All 13+ services successfully parse filters from request body
- [ ] Database queries receive correct filter values
- [ ] All unit tests pass
- [ ] Integration tests verify end-to-end flow
- [ ] No breaking changes to external APIs
- [ ] List endpoints return filtered results correctly

## Open Questions

None - design approved.

## References

- VOIP-1190: Related database handler refactoring
- bin-common-handler/pkg/requesthandler: Request sender implementation
- bin-common-handler/pkg/utilhandler: Utility handler location
