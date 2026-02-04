# Timestamp Migration: string to *time.Time

## Overview

Migrate all timestamp fields from `string` to `*time.Time` across the monorepo for type safety, computation convenience, API consistency, and database efficiency.

## Decisions

| Aspect | Decision |
|--------|----------|
| Strategy | Big bang - all services at once |
| API compatibility | No layer needed - RFC 3339 output identical |
| Database | No changes - already DATETIME(6) |
| Soft delete sentinel | Keep as time.Time (9999-01-01) |
| NULL handling | Use `*time.Time` (nil = NULL) |
| Field types | All timestamps use `*time.Time` uniformly |

## Scope

**23 services with model changes:**
- bin-agent-manager (2 files)
- bin-ai-manager (8 files)
- bin-billing-manager (4 files)
- bin-call-manager (9 files)
- bin-campaign-manager (6 files)
- bin-conference-manager (4 files)
- bin-contact-manager (4 files)
- bin-conversation-manager (7 files)
- bin-customer-manager (4 files)
- bin-email-manager (2 files)
- bin-flow-manager (4 files)
- bin-message-manager (2 files)
- bin-number-manager (3 files)
- bin-outdial-manager (5 files)
- bin-pipecat-manager (1 file)
- bin-queue-manager (4 files)
- bin-registrar-manager (5 files)
- bin-route-manager (4 files)
- bin-storage-manager (4 files)
- bin-tag-manager (2 files)
- bin-talk-manager (3 files)
- bin-transcribe-manager (4 files)
- bin-transfer-manager (2 files)

**Total: 93 model files** plus additional handler/service files for code pattern updates.

## Struct Field Changes

### Before

```go
type Call struct {
    ID            uuid.UUID `json:"id" db:"id,uuid"`
    // ... other fields ...

    TMRinging     string `json:"tm_ringing,omitempty" db:"tm_ringing"`
    TMProgressing string `json:"tm_progressing,omitempty" db:"tm_progressing"`
    TMHangup      string `json:"tm_hangup,omitempty" db:"tm_hangup"`
    TMCreate      string `json:"tm_create,omitempty" db:"tm_create"`
    TMUpdate      string `json:"tm_update,omitempty" db:"tm_update"`
    TMDelete      string `json:"tm_delete,omitempty" db:"tm_delete"`
}
```

### After

```go
type Call struct {
    ID            uuid.UUID  `json:"id" db:"id,uuid"`
    // ... other fields ...

    TMRinging     *time.Time `json:"tm_ringing,omitempty" db:"tm_ringing"`
    TMProgressing *time.Time `json:"tm_progressing,omitempty" db:"tm_progressing"`
    TMHangup      *time.Time `json:"tm_hangup,omitempty" db:"tm_hangup"`
    TMCreate      *time.Time `json:"tm_create,omitempty" db:"tm_create"`
    TMUpdate      *time.Time `json:"tm_update,omitempty" db:"tm_update"`
    TMDelete      *time.Time `json:"tm_delete,omitempty" db:"tm_delete"`
}
```

### Key Points

- All timestamp fields become `*time.Time` (pointer for NULL handling)
- JSON tags stay the same (field names unchanged)
- `omitempty` behavior: `nil` pointers are omitted, non-nil are serialized
- DB tags stay the same (sql driver handles DATETIME to time.Time)

## Utility Function Changes (bin-common-handler)

### Current Functions (string-based)

```go
func TimeGetCurTime() string                          // Returns current time as string
func TimeParseWithError(t string) (time.Time, error)  // Parses string to time.Time
```

### New/Updated Functions

```go
// Keep existing for backward compatibility during migration
func TimeGetCurTime() string

// Add new pointer-based helpers
func TimeNow() *time.Time                       // Returns pointer to current UTC time
func TimeParse(s string) *time.Time             // Parses string, returns nil on error
func TimeParseWithError(s string) (*time.Time, error)  // Parses string with error

// Sentinel helpers
var SentinelDeleteTime = time.Date(9999, 1, 1, 0, 0, 0, 0, time.UTC)

func IsSentinel(t *time.Time) bool              // Returns true if t equals sentinel
func NewSentinel() *time.Time                   // Returns pointer to sentinel time
```

### Usage Changes in Services

| Before (string) | After (*time.Time) |
|-----------------|-------------------|
| `model.TMCreate = utilhandler.TimeGetCurTime()` | `model.TMCreate = utilhandler.TimeNow()` |
| `if model.TMDelete == ""` | `if model.TMDelete == nil` |
| `if model.TMDelete == "9999-01-01..."` | `if utilhandler.IsSentinel(model.TMDelete)` |

## JSON Serialization Behavior

### Output Comparison

**Current output (string):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "tm_create": "2026-02-05T10:23:45.123456Z",
  "tm_update": "2026-02-05T10:23:45.123456Z",
  "tm_delete": "9999-01-01T00:00:00Z"
}
```

**New output (*time.Time):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "tm_create": "2026-02-05T10:23:45.123456Z",
  "tm_update": "2026-02-05T10:23:45.123456Z",
  "tm_delete": "9999-01-01T00:00:00Z"
}
```

**Result:** Identical output. Go's `time.Time` marshals to RFC 3339 by default.

### Edge Cases

| Scenario | Before (string) | After (*time.Time) |
|----------|-----------------|-------------------|
| NULL in DB | `""` (empty string) | `null` or omitted (with omitempty) |
| Zero value | `""` or `"0001-01-01..."` | `null` (nil pointer) |
| Sentinel | `"9999-01-01T00:00:00Z"` | `"9999-01-01T00:00:00Z"` (unchanged) |

## Database Scanning Behavior

Go's `database/sql` driver automatically handles `DATETIME(6)` to `*time.Time` conversion:

```go
// Before: manual string handling
var tmCreate string
row.Scan(&tmCreate)  // DB returns string representation

// After: automatic conversion
var tmCreate *time.Time
row.Scan(&tmCreate)  // DB returns time.Time directly, NULL becomes nil
```

### No Changes Needed In

- Database schema (already `DATETIME(6)`)
- SQL queries (same column names)
- Alembic migrations (no new migrations required)

### Timezone Consideration

MySQL `DATETIME(6)` stores without timezone. Ensure connection string includes:
```
?parseTime=true&loc=UTC
```

## Code Change Patterns

### 1. Setting Current Time

```go
// Before
model.TMCreate = utilhandler.TimeGetCurTime()

// After
model.TMCreate = utilhandler.TimeNow()
```

### 2. Checking for Empty/Unset

```go
// Before
if model.TMCreate == "" {
    // not set
}

// After
if model.TMCreate == nil {
    // not set
}
```

### 3. Soft Delete Sentinel Checks

```go
// Before
if model.TMDelete != "9999-01-01 00:00:00.000000" {
    // record is deleted
}

// After
if !utilhandler.IsSentinel(model.TMDelete) {
    // record is deleted
}
```

### 4. Time Comparisons

```go
// Before (string comparison - error prone!)
if model.TMCreate > model.TMUpdate {
    // ...
}

// After (proper time comparison)
if model.TMCreate != nil && model.TMUpdate != nil && model.TMCreate.After(*model.TMUpdate) {
    // ...
}
```

### 5. Duration Calculations

```go
// Before
tmCreate, _ := utilhandler.TimeParseWithError(model.TMCreate)
duration := time.Since(tmCreate)

// After
if model.TMCreate != nil {
    duration := time.Since(*model.TMCreate)
}
```

## Implementation Order

### Step 1: Update bin-common-handler First

- Add new `*time.Time` utility functions (`TimeNow`, `IsSentinel`, `NewSentinel`)
- Keep existing string functions temporarily (backward compatibility)
- Run verification workflow

### Step 2: Update Model Structs in All 23 Services

- Change `string` to `*time.Time` for all timestamp fields
- Update imports to include `"time"` package
- This will cause compilation errors (expected)

### Step 3: Fix Compilation Errors Service by Service

- Update handler code to use new patterns
- Update string comparisons to nil checks
- Update sentinel checks to use `IsSentinel()`
- Run verification workflow for each service

### Step 4: Update Tests

- Fix test assertions expecting string timestamps
- Update mock data to use `*time.Time`
- Run `go test ./...` for each service

### Step 5: Remove Deprecated String Functions

- Remove old `TimeGetCurTime()` if no longer used
- Final cleanup pass

### Step 6: Full Verification

- Run verification workflow for all 30+ services
- Ensure all tests pass

## Risks & Mitigations

### Risk 1: API Breaking Change for NULL Handling

- **Before:** NULL in DB becomes `""` (empty string) in JSON
- **After:** NULL in DB becomes `null` or omitted in JSON
- **Mitigation:** This is actually more correct behavior. Most clients handle `null` properly. Low risk since most timestamps have values.

### Risk 2: Sentinel Format Mismatch

- **Risk:** Legacy data might have `"9999-01-01 00:00:00.000000"` (space-separated) vs `"9999-01-01T00:00:00Z"` (ISO format)
- **Mitigation:** Both parse to the same `time.Time` value. The `IsSentinel()` function compares actual time values, not strings.

### Risk 3: Build Failures During Migration

- **Risk:** Big bang means many files change at once, potential for missed updates
- **Mitigation:** Compiler will catch type mismatches. Run verification workflow after each service.

### Risk 4: Test Failures

- **Risk:** Tests may assert on string values
- **Mitigation:** Update test assertions. Most will be straightforward type changes.

### Risk 5: Timezone Inconsistency

- **Risk:** Different services might handle timezones differently
- **Mitigation:** Ensure all DSN connections use `parseTime=true&loc=UTC`. All `TimeNow()` returns UTC.
