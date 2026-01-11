# Database Handler Consolidation Design

## Problem Statement

The flow-manager's `pkg/dbutil` package solves manual ORM pain through struct tags and reflection. This solution should benefit all services in the monorepo, not just flow-manager.

Current situation:
1. **Isolated utility** - dbutil exists only in flow-manager
2. **Duplication** - bin-common-handler has `PrepareUpdateFields` that duplicates UUID/JSON conversion logic
3. **Missed opportunity** - Other services (call-manager, agent-manager) still use manual field mapping

## Solution: Consolidate Database Mapping in Common Handler

Move dbutil to bin-common-handler and merge with existing database utilities. Create a unified, reflection-based API that all services can adopt.

## Design

### Core API

Three functions handle all database mapping needs:

```go
// PrepareFields converts struct or map to database-ready values
// - Struct input: reads db tags, skips db:"-", converts UUID/JSON
// - Map input: converts UUID/JSON without tag filtering
func PrepareFields(data any) (map[string]any, error)

// ScanRow scans sql.Rows into struct using db tags
// Handles NULL values, UUID bytes→UUID, JSON string→struct
func ScanRow(row *sql.Rows, dest any) error

// ValidateSchema validates struct tags match database columns
// Catches schema drift at startup
func ValidateSchema(db *sql.DB, tableName string, model any) error
```

### PrepareFields Implementation

**Accepts two input types:**

**1. Struct (tag-aware)** - For INSERT operations
```go
fields, err := PrepareFields(&flow.Flow{...})
// Returns: {"id": []byte{...}, "name": "foo", "actions": "[]", ...}
// Skips fields tagged db:"-"
```

**2. Map (tag-agnostic)** - For UPDATE operations
```go
fields, err := PrepareFields(map[string]any{"name": "new", "priority": 5})
// Returns: {"name": "new", "priority": 5}
// No tag filtering, just type conversions
```

**Conversion logic (shared by both paths):**
- `uuid.UUID` → `[]byte` via `.Bytes()`
- Slices/maps/structs → `[]byte` via `json.Marshal()`
- Primitives → pass through unchanged

**Implementation flow:**
```go
func PrepareFields(data any) (map[string]any, error) {
    val := reflect.ValueOf(data)
    if val.Kind() == reflect.Ptr {
        val = val.Elem()
    }

    switch val.Kind() {
    case reflect.Struct:
        return prepareFieldsFromStruct(val)  // Tag-aware
    case reflect.Map:
        return prepareFieldsFromMap(data)    // Tag-agnostic
    default:
        return nil, fmt.Errorf("expected struct or map, got %T", data)
    }
}
```

### ScanRow (Unchanged)

Moves as-is from flow-manager. Already handles:
- NULL → zero value conversions
- UUID bytes → `uuid.UUID`
- JSON strings → slices/maps/structs
- Embedded struct fields

### ValidateSchema (New)

Validates struct tags match database schema:
```go
// At startup
if err := ValidateSchema(db, "flow_flows", &flow.Flow{}); err != nil {
    log.Fatalf("Schema mismatch: %v", err)
}
```

Detects:
- Missing database columns
- Extra database columns
- Type mismatches (UUID field with VARCHAR column)

### Usage Changes

**INSERT operations:**

Before:
```go
fields := dbutil.GetDBFields(f)
values, _ := dbutil.PrepareValues(f)
squirrel.Insert(table).Columns(fields...).Values(values...)
```

After:
```go
fields, _ := databasehandler.PrepareFields(f)
squirrel.Insert(table).SetMap(fields)
```

Benefits: Eliminates `GetDBFields()`, uses Squirrel's `SetMap()` API, columns/values always synchronized.

**UPDATE operations:**

Before:
```go
tmpFields := commondatabasehandler.PrepareUpdateFields(fields)
squirrel.Update(table).SetMap(tmpFields)
```

After:
```go
tmpFields, _ := databasehandler.PrepareFields(fields)
squirrel.Update(table).SetMap(tmpFields)
```

Benefits: Same function for both INSERT and UPDATE, consistent conversion logic.

**SELECT operations:**

Before:
```go
fields := dbutil.GetDBFields(&flow.Flow{})
squirrel.Select(fields...).From(table)

// Later, in scan:
if err := dbutil.ScanRow(row, res); err != nil { ... }
```

After:
```go
// Manual field list still needed for SELECT
squirrel.Select("id", "name", "actions", ...).From(table)

// Or use GetDBFields helper (stays available)
fields := databasehandler.GetDBFields(&flow.Flow{})
squirrel.Select(fields...).From(table)

// Scan unchanged
if err := databasehandler.ScanRow(row, res); err != nil { ... }
```

Note: `GetDBFields` stays available as a helper, but `PrepareFields` is the main API.

## Migration Strategy

### Phase 1: Add to bin-common-handler

**File structure:**
```
bin-common-handler/pkg/databasehandler/
├── main.go              # Existing (PrepareUpdateFields, ApplyFields, Connect)
├── mapping.go           # NEW: PrepareFields, ScanRow, ValidateSchema, GetDBFields
├── mapping_internal.go  # NEW: prepareFieldsFromStruct, prepareFieldsFromMap, convertValue
└── mapping_test.go      # NEW: 30+ test cases from flow-manager + new tests
```

**Changes:**
1. Copy flow-manager's `pkg/dbutil/*` → `mapping*.go` files
2. Implement `PrepareFields` as designed (struct + map support)
3. Keep `GetDBFields` as public helper (useful for SELECT queries)
4. Add `ValidateSchema` function
5. Mark `PrepareUpdateFields` as deprecated (internal implementation calls `PrepareFields`)
6. Comprehensive test suite (all existing dbutil tests + new PrepareFields tests)

**Backward compatibility:**
- Old `PrepareUpdateFields` stays functional
- Internal implementation delegates to `PrepareFields(map)`
- No breaking changes to existing services

### Phase 2: Migrate flow-manager

**Import changes:**
```go
// Before
import "monorepo/bin-flow-manager/pkg/dbutil"

// After
import "monorepo/bin-common-handler/pkg/databasehandler"
```

**Code changes:**
1. Replace `dbutil.GetDBFields + dbutil.PrepareValues` with `databasehandler.PrepareFields`
2. Change INSERT to use `.SetMap()` instead of `.Columns().Values()`
3. Update `dbutil.ScanRow` → `databasehandler.ScanRow` (same signature)
4. Update SELECT queries to use `databasehandler.GetDBFields` (optional, for consistency)
5. Delete `pkg/dbutil/` directory

**Files to update:**
- `pkg/dbhandler/flows.go` (FlowCreate, flowGetFromDB, FlowGets)
- `pkg/dbhandler/activeflow.go` (ActiveflowCreate, activeflowGetFromDB, ActiveflowGets)
- All tests importing dbutil

**Verification:**
- Run full test suite: `go test ./...`
- Run linter: `golangci-lint run -v --timeout 5m`
- Verify coverage remains 85%+

### Phase 3: Enable other services

**Documentation:**
- Add usage guide to `bin-common-handler/README.md`
- Document struct tag format and conversion rules
- Show migration examples (manual field lists → struct tags)

**Adoption path:**
Other services adopt incrementally:
1. Add `db` tags to model structs
2. Replace manual INSERT/UPDATE field lists with `PrepareFields`
3. Replace manual Scan() with `ScanRow`
4. Remove field list constants

**High-value targets:**
- bin-call-manager (call.Call model has 20+ fields)
- bin-agent-manager (agent.Agent model has 15+ fields)
- bin-conference-manager (conference.Conference model)

## Error Handling

**PrepareFields errors:**
```go
"PrepareFields: expected struct or map, got []string"
"PrepareFields: field 'Actions' JSON marshal failed: ..."
"PrepareFields: cannot process nil pointer"
```

Returns errors for:
- Invalid input types (not struct or map)
- JSON marshal failures (with field context)
- Nil pointer dereference

**ScanRow errors:**
```go
"ScanRow: dest must be pointer to struct"
"ScanRow: scanning field 'ID': sql: Scan error..."
"ScanRow: cannot unmarshal JSON for field 'Actions': ..."
```

Returns errors for:
- Non-pointer destination
- Non-struct destination
- SQL scan failures (with field context)
- JSON unmarshal failures (with field context)

## Edge Cases

### NULL Handling

**Reading (ScanRow):**
- NULL string → `""`
- NULL int → `0`
- NULL UUID → `uuid.Nil`
- NULL JSON → empty slice/struct

**Writing (PrepareFields):**
- Go zero values → actual values (not NULL)
- Explicit `nil` in map → SQL NULL

Example:
```go
// Zero value writes actual value
PrepareFields(&Flow{Name: ""})  // → {"name": ""}

// Explicit nil writes NULL
PrepareFields(map[string]any{"name": nil})  // → {"name": NULL}
```

### UUID Handling

- `uuid.Nil` → nil UUID bytes (00000000-0000-0000-0000-000000000000)
- Stored in BINARY(16) columns, never SQL NULL
- For nullable UUIDs: use `*uuid.UUID` (nil pointer → SQL NULL)

### JSON Handling

- Empty slices → `"[]"` (not NULL)
- Empty maps → `"{}"` (not NULL)
- Nil slices/maps in structs → `"null"` or empty based on zero value

### Embedded Structs

Recursively processed:
```go
type Flow struct {
    commonidentity.Identity  // ID, CustomerID fields
    Name string `db:"name"`
}

// PrepareFields result includes embedded fields at top level:
// {"id": bytes, "customer_id": bytes, "name": "foo"}
```

### Type Aliases

Custom types handled by underlying type:
```go
type Status string  // Treated as string
type Priority int   // Treated as int
```

## Testing Strategy

### Unit Tests (mapping_test.go)

**PrepareFields struct input:**
- Basic types (string, int, bool, float64)
- UUID conversion (uuid.UUID → bytes)
- JSON conversion (slices, maps, structs)
- Skip fields (db:"-")
- Embedded structs
- Empty values (empty slices, uuid.Nil)
- Error cases (invalid input, JSON marshal failures)

**PrepareFields map input:**
- UUID detection and conversion
- JSON type detection
- Primitive passthrough
- Nil value preservation

**ScanRow tests:**
- Basic types
- NULL handling (all types)
- UUID conversion (bytes → uuid.UUID)
- JSON unmarshaling
- Embedded structs
- Error cases (non-pointer, non-struct, scan failures)

**ValidateSchema tests:**
- Missing columns detected
- Extra columns detected
- Type mismatches detected
- Successful validation

**Target coverage:** 85%+ (match current dbutil coverage)

### Integration Tests

Real database operations:
```go
// Round-trip test
flow := &flow.Flow{Name: "test", Actions: []action.Action{...}}

// INSERT with PrepareFields
fields, _ := PrepareFields(flow)
Insert(table).SetMap(fields).Exec()

// SELECT with ScanRow
row := Select("*").From(table).QueryRow()
result := &flow.Flow{}
ScanRow(row, result)

// Verify result matches original
assert.Equal(t, flow.Name, result.Name)
assert.Equal(t, flow.Actions, result.Actions)
```

Test with:
- Flow model (9 fields)
- Activeflow model (15 fields)
- All field types (UUID, JSON, primitives)

### Backward Compatibility Tests

Verify old `PrepareUpdateFields` still works:
```go
// Old API
oldResult := PrepareUpdateFields(map[string]any{"id": uuid.New(), "name": "test"})

// New API
newResult, _ := PrepareFields(map[string]any{"id": uuid.New(), "name": "test"})

// Should produce identical output
assert.Equal(t, oldResult, newResult)
```

### Performance Tests

Benchmark INSERT before/after:
```go
// Ensure no regression from reflection overhead
BenchmarkInsertOld  // GetDBFields + PrepareValues
BenchmarkInsertNew  // PrepareFields
```

Reflection overhead is negligible (database I/O dominates).

## Benefits

### Immediate (flow-manager)

1. **Simpler INSERT** - One function (`PrepareFields`) replaces two (`GetDBFields + PrepareValues`)
2. **Safer INSERT** - `SetMap()` ensures columns/values always match
3. **Unified API** - Same function for INSERT and UPDATE

### Long-term (all services)

1. **Shared utility** - All services can adopt struct tag approach
2. **Less duplication** - Single conversion logic for UUID/JSON across monorepo
3. **Better organization** - Related utilities co-located in common-handler
4. **Easier onboarding** - New services start with struct tags, not manual field lists

### Maintenance

1. **Single source of truth** - One place to fix UUID/JSON conversion bugs
2. **Validation tooling** - `ValidateSchema` catches drift at startup
3. **Consistent patterns** - All services use same database mapping approach

## Risks & Mitigations

**Risk:** Breaking changes to bin-common-handler affect all services

**Mitigation:**
- Keep `PrepareUpdateFields` as deprecated wrapper (no breaking change)
- Comprehensive test suite (30+ test cases)
- Migrate flow-manager first, validate before rolling out

**Risk:** Reflection overhead impacts performance

**Mitigation:**
- Database I/O dominates (reflection cost negligible)
- Benchmark to verify no regression
- Can optimize hot paths later if needed

**Risk:** Complex migration for services with many models

**Mitigation:**
- Incremental adoption (one model at a time)
- Clear migration guide with examples
- flow-manager serves as reference implementation

**Risk:** Struct tags become single point of failure

**Mitigation:**
- `ValidateSchema` catches tag/schema mismatches at startup
- Tests verify tag parsing works correctly
- Tags are simpler than manual field lists (less error-prone)

## Implementation Checklist

### Phase 1: Add to bin-common-handler
- [ ] Create `mapping.go` with PrepareFields, ScanRow, ValidateSchema, GetDBFields
- [ ] Create `mapping_internal.go` with internal helpers
- [ ] Create `mapping_test.go` with comprehensive tests
- [ ] Mark `PrepareUpdateFields` as deprecated in docstring
- [ ] Update bin-common-handler README with usage examples
- [ ] Run tests: `go test ./pkg/databasehandler/...`
- [ ] Commit: "feat: add PrepareFields unified database mapping API"

### Phase 2: Migrate flow-manager
- [ ] Update imports: `dbutil` → `databasehandler`
- [ ] Update FlowCreate: use PrepareFields + SetMap
- [ ] Update flowGetFromDB: use GetDBFields helper
- [ ] Update FlowGets: use GetDBFields helper
- [ ] Update ActiveflowCreate: use PrepareFields + SetMap
- [ ] Update activeflowGetFromDB: use GetDBFields helper
- [ ] Update ActiveflowGets: use GetDBFields helper
- [ ] Update all tests importing dbutil
- [ ] Delete `pkg/dbutil/` directory
- [ ] Run tests: `go test ./...`
- [ ] Run linter: `golangci-lint run -v --timeout 5m`
- [ ] Verify coverage: `go test -coverprofile cp.out ./...`
- [ ] Commit: "refactor: migrate to common-handler PrepareFields API"

### Phase 3: Update all services
- [ ] Run dependency update workflow from monorepo root
- [ ] Verify other services still compile
- [ ] Commit: "chore: update dependencies after databasehandler changes"

### Phase 4: Documentation
- [ ] Update bin-common-handler README
- [ ] Add migration guide for other services
- [ ] Document struct tag format
- [ ] Add usage examples
- [ ] Commit: "docs: add PrepareFields migration guide"

## Success Metrics

**Adoption:**
- bin-common-handler has PrepareFields available ✅
- flow-manager migrated successfully ✅
- At least 2 other services adopt within 3 months

**Quality:**
- Test coverage remains 85%+
- Zero production bugs from migration
- Performance unchanged (±5% acceptable)

**Simplification:**
- flow-manager: -100 lines of boilerplate (GetDBFields calls eliminated)
- Per-service adoption: -50 to -200 lines (eliminates manual field lists)
- Monorepo-wide: Single UUID/JSON conversion implementation
