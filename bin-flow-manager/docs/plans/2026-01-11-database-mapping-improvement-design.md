# Database Mapping Improvement Design

## Problem Statement

Manual ORM work in the current database handler creates multiple pain points:

1. **Maintenance burden** - Each new field requires updates in 5+ places (field list, Scan(), Insert values, Update map, etc.)
2. **Scattered type conversions** - UUID.Bytes(), JSON marshal/unmarshal repeat throughout the codebase
3. **Poor readability** - Field mappings confuse developers, especially newcomers
4. **Runtime errors** - Field count mismatches and type errors escape compile-time detection

## Solution: Struct Tags + Reflection Helpers

Struct tags define database mappings. Generic helpers eliminate manual conversions.

## Tag Format

Add `db` tags to model structs that describe how each field maps to the database:

```
`db:"column_name[,conversion_type]"`
```

**Examples:**
```go
type Flow struct {
    ID         uuid.UUID `db:"id,uuid"`           // UUID needs byte conversion
    CustomerID uuid.UUID `db:"customer_id,uuid"`
    Name       string    `db:"name"`              // Direct mapping
    Actions    []Action  `db:"actions,json"`      // JSON marshal/unmarshal
    TMCreate   string    `db:"tm_create"`
    Persist    bool      `db:"-"`                 // Skip this field (not in DB)
}
```

**Supported conversion types:**
- `uuid` - Handles uuid.UUID ↔ []byte conversion
- `json` - Handles struct/slice ↔ JSON string conversion
- (empty) - Direct type mapping (string, int, etc.)
- `-` - Skip field entirely

## Helper Functions

Create helper functions in `pkg/dbutil` (or extend `pkg/dbhandler`):

### 1. GetDBFields
```go
// GetDBFields returns ordered column names from struct tags
func GetDBFields(model interface{}) []string
```

Reads struct tags to generate column names. Replaces manually maintained field lists like `flowsFields`.

### 2. ScanRow
```go
// ScanRow scans a sql.Row/sql.Rows into a struct using db tags
// Handles UUID byte conversion, JSON unmarshaling automatically
func ScanRow(row *sql.Rows, dest interface{}) error
```

Eliminates manual Scan() calls with 10+ parameters. Applies conversions based on tags automatically.

### 3. PrepareValues
```go
// PrepareValues converts struct fields to database values for INSERT/UPDATE
// Handles UUID to bytes, JSON marshaling, etc.
func PrepareValues(model interface{}) ([]interface{}, error)
```

Converts struct values to database-ready values for INSERT/UPDATE operations.

## Integration Approach

Integrate incrementally without breaking existing functionality:

**Step 1: Add db tags to models**
- `models/flow/flow.go`
- `models/activeflow/activeflow.go`
- `models/variable/variable.go`

**Step 2: Replace manual operations**

Example transformation in `pkg/dbhandler/flows.go`:

**Before:**
```go
var flowsFields = []string{"id", "customer_id", ...}  // Manual maintenance

func (h *handler) FlowCreate(ctx context.Context, f *flow.Flow) error {
    tmpActions, err := json.Marshal(f.Actions)  // Manual conversion
    if err != nil {
        return err
    }

    sb := squirrel.Insert(flowsTable).
        Columns(flowsFields...).
        Values(f.ID.Bytes(), f.CustomerID.Bytes(), ..., tmpActions, ...)  // Manual values
}
```

**After:**
```go
func (h *handler) FlowCreate(ctx context.Context, f *flow.Flow) error {
    fields := dbutil.GetDBFields(f)
    values, err := dbutil.PrepareValues(f)
    if err != nil {
        return err
    }

    sb := squirrel.Insert(flowsTable).
        Columns(fields...).
        Values(values...)
}
```

**Step 3: Simplify row scanning**

**Before:**
```go
func (h *handler) flowGetFromRow(row *sql.Rows) (*flow.Flow, error) {
    var actions string
    res := &flow.Flow{}
    if err := row.Scan(&res.ID, &res.CustomerID, &res.Type, ...); err != nil {
        return nil, err
    }
    if err := json.Unmarshal([]byte(actions), &res.Actions); err != nil {
        return nil, err
    }
    return res, nil
}
```

**After:**
```go
func (h *handler) flowGetFromRow(row *sql.Rows) (*flow.Flow, error) {
    res := &flow.Flow{}
    if err := dbutil.ScanRow(row, res); err != nil {
        return nil, err
    }
    return res, nil
}
```

## Error Handling Strategy

1. **Tag parsing errors** - Panic at startup for malformed tags (fail fast)
2. **Type mismatch errors** - Return clear error messages: `"field Actions: cannot convert []Action to database value"`
3. **Scan errors** - Wrap SQL errors with field context: `"scanning field ID: sql: Scan error..."`

## Edge Cases

### UUID Handling
- `uuid.Nil` stores as `00000000-0000-0000-0000-000000000000` (the nil UUID bytes)
- Conversion to SQL NULL does not occur
- `uuid.UUID` fields use NOT NULL columns
- For nullable UUIDs in the future, use `*uuid.UUID` (pointer converts nil → SQL NULL)

### NULL Handling
Handle SQL NULL values from manual inserts or legacy data:

**Conversion rules:**
- NULL string → `""` (empty string)
- NULL int → `0`
- NULL UUID bytes → `uuid.Nil`
- NULL JSON → Empty slice `[]` or empty struct `{}` (depending on field type)
- NULL timestamp → `""` (empty string)

**Implementation approach:**

Use `sql.Null*` types as intermediate scan targets:

```go
func ScanRow(row *sql.Rows, dest interface{}) error {
    // For each field with db tag:
    switch conversionType {
    case "uuid":
        var nullBytes sql.NullString
        scan(&nullBytes)
        if nullBytes.Valid {
            field.Set(uuid.FromBytes(nullBytes.String))
        } else {
            field.Set(uuid.Nil)  // NULL → uuid.Nil
        }
    case "json":
        var nullJSON sql.NullString
        scan(&nullJSON)
        if nullJSON.Valid && nullJSON.String != "" {
            json.Unmarshal(nullJSON.String, &field)
        } else {
            // Leave as zero value (empty slice/struct)
        }
    }
}
```

### Other Edge Cases
- **Empty JSON fields** - Empty slices `[]Action{}` store as `"[]"`, not NULL
- **Skipped fields** - Fields tagged with `db:"-"` (like `Persist bool`) are ignored
- **Pointer fields** - Support `*string`, `*int` for nullable columns
- **Time fields** - Keep current `string` type for timestamps (no automatic time.Time conversion to avoid timezone issues)

## Validation

Create validation helper to catch missing tags at startup:

```go
// ValidateModel ensures all exported fields have db tags (except those marked with "-")
func ValidateModel(model interface{}) error
```

Call in `main.go` or handler initialization to catch configuration errors early.

## Testing Strategy

1. **Unit tests** - Test each helper function with various field types
2. **Table-driven tests** - Test conversion edge cases (NULL values, empty strings, uuid.Nil, etc.)
3. **Integration tests** - Use actual database for flows/activeflows/variables

## Benefits

1. **Type-safe** - Struct changes cause compiler errors when tags are missing
2. **Eliminates field list maintenance** - No more manual `flowsFields` arrays
3. **Centralizes conversion logic** - UUID, JSON conversions in one place
4. **Minimal dependencies** - Uses only Go stdlib reflection
5. **Works with existing code** - Compatible with current squirrel queries
6. **Better readability** - Mapping is self-documenting in struct definition

## Migration Path

1. Implement `pkg/dbutil` helpers with comprehensive tests
2. Add db tags to `models/flow/flow.go`
3. Update `pkg/dbhandler/flows.go` to use new helpers
4. Verify all flow operations work correctly
5. Repeat for `activeflow` and `variable` models
6. Remove old manual field lists and conversion code
7. Run full test suite and integration tests

## Risks and Mitigations

**Risk:** Reflection overhead impacts performance
**Mitigation:** Benchmark critical paths. Database I/O dwarfs reflection cost.

**Risk:** Complex edge cases in scanning logic
**Mitigation:** Comprehensive test coverage for NULL handling, type conversions, and error cases.

**Risk:** Breaking existing functionality during migration
**Mitigation:** Incremental rollout per model type, thorough testing at each step.
