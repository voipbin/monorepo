# Timestamp Migration Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Migrate all timestamp fields from `string` to `*time.Time` across 23 services and 93 model files.

**Architecture:** Direct type change in Go structs. Database already uses `DATETIME(6)`, so sql driver handles conversion automatically. No schema migrations needed.

**Tech Stack:** Go 1.21+, MySQL DATETIME(6), standard library `time` package, `database/sql` driver.

---

## Task 1: Add New Time Utilities to bin-common-handler

**Files:**
- Modify: `bin-common-handler/pkg/utilhandler/time.go`
- Modify: `bin-common-handler/pkg/utilhandler/time_test.go`
- Modify: `bin-common-handler/pkg/databasehandler/main.go`

**Step 1: Write the failing tests for new time utilities**

Add to `bin-common-handler/pkg/utilhandler/time_test.go`:

```go
func Test_TimeNow(t *testing.T) {
	result := TimeNow()

	if result == nil {
		t.Error("Expected non-nil pointer, got nil")
	}

	// Verify the time is recent (within last minute)
	now := time.Now().UTC()
	diff := now.Sub(*result)
	if diff < 0 || diff > time.Minute {
		t.Errorf("TimeNow is not recent. Now: %v, Result: %v, Diff: %v", now, *result, diff)
	}
}

func Test_TimeNowAdd(t *testing.T) {
	before := time.Now().UTC()
	result := TimeNowAdd(time.Hour)
	after := time.Now().UTC().Add(time.Hour)

	if result == nil {
		t.Error("Expected non-nil pointer, got nil")
	}

	if result.Before(before.Add(time.Hour)) || result.After(after) {
		t.Errorf("TimeNowAdd result out of expected range")
	}
}

func Test_IsSentinel(t *testing.T) {
	tests := []struct {
		name   string
		input  *time.Time
		expect bool
	}{
		{"nil", nil, false},
		{"sentinel", NewSentinel(), true},
		{"non-sentinel", TimeNow(), false},
		{"zero time", func() *time.Time { t := time.Time{}; return &t }(), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsSentinel(tt.input)
			if result != tt.expect {
				t.Errorf("IsSentinel(%v) = %v, want %v", tt.input, result, tt.expect)
			}
		})
	}
}

func Test_NewSentinel(t *testing.T) {
	result := NewSentinel()

	if result == nil {
		t.Error("Expected non-nil pointer, got nil")
	}

	expected := time.Date(9999, 1, 1, 0, 0, 0, 0, time.UTC)
	if !result.Equal(expected) {
		t.Errorf("NewSentinel() = %v, want %v", *result, expected)
	}
}

func Test_SentinelDeleteTime(t *testing.T) {
	expected := time.Date(9999, 1, 1, 0, 0, 0, 0, time.UTC)
	if !SentinelDeleteTime.Equal(expected) {
		t.Errorf("SentinelDeleteTime = %v, want %v", SentinelDeleteTime, expected)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd bin-common-handler && go test -v ./pkg/utilhandler/... -run "Test_TimeNow|Test_IsSentinel|Test_NewSentinel|Test_SentinelDeleteTime|Test_TimeNowAdd"`
Expected: FAIL with undefined functions

**Step 3: Write minimal implementation**

Add to `bin-common-handler/pkg/utilhandler/time.go`:

```go
// SentinelDeleteTime is the sentinel value used for soft deletes (9999-01-01 00:00:00 UTC)
var SentinelDeleteTime = time.Date(9999, 1, 1, 0, 0, 0, 0, time.UTC)

// TimeNow returns a pointer to the current UTC time
func TimeNow() *time.Time {
	now := time.Now().UTC()
	return &now
}

// TimeNowAdd returns a pointer to the current UTC time plus the given duration
func TimeNowAdd(d time.Duration) *time.Time {
	t := time.Now().UTC().Add(d)
	return &t
}

// NewSentinel returns a pointer to the sentinel time value
func NewSentinel() *time.Time {
	t := SentinelDeleteTime
	return &t
}

// IsSentinel returns true if the given time pointer points to the sentinel value
func IsSentinel(t *time.Time) bool {
	if t == nil {
		return false
	}
	return t.Equal(SentinelDeleteTime)
}

// Method versions for utilHandler interface
func (h *utilHandler) TimeNow() *time.Time {
	return TimeNow()
}

func (h *utilHandler) TimeNowAdd(d time.Duration) *time.Time {
	return TimeNowAdd(d)
}

func (h *utilHandler) IsSentinel(t *time.Time) bool {
	return IsSentinel(t)
}

func (h *utilHandler) NewSentinel() *time.Time {
	return NewSentinel()
}
```

**Step 4: Update UtilHandler interface**

Add to interface in `bin-common-handler/pkg/utilhandler/main.go`:

```go
// Add these methods to the UtilHandler interface
TimeNow() *time.Time
TimeNowAdd(d time.Duration) *time.Time
IsSentinel(t *time.Time) bool
NewSentinel() *time.Time
```

**Step 5: Run test to verify it passes**

Run: `cd bin-common-handler && go test -v ./pkg/utilhandler/... -run "Test_TimeNow|Test_IsSentinel|Test_NewSentinel|Test_SentinelDeleteTime|Test_TimeNowAdd"`
Expected: PASS

**Step 6: Update databasehandler DefaultTimeStamp**

In `bin-common-handler/pkg/databasehandler/main.go`, add:

```go
import "time"

// DefaultSentinelTime is the time.Time version of the sentinel value
var DefaultSentinelTime = time.Date(9999, 1, 1, 0, 0, 0, 0, time.UTC)

// NewDefaultSentinel returns a pointer to the default sentinel time
func NewDefaultSentinel() *time.Time {
	t := DefaultSentinelTime
	return &t
}

// IsSentinelTime returns true if the given time equals the sentinel value
func IsSentinelTime(t *time.Time) bool {
	if t == nil {
		return false
	}
	return t.Equal(DefaultSentinelTime)
}
```

**Step 7: Run full verification**

Run: `cd bin-common-handler && go mod tidy && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`
Expected: All pass

**Step 8: Commit**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-Timestamp-string-to-time-migration
git add bin-common-handler/
git commit -m "NOJIRA-Timestamp-string-to-time-migration

- bin-common-handler: Add TimeNow, TimeNowAdd, NewSentinel, IsSentinel utilities
- bin-common-handler: Add SentinelDeleteTime constant for *time.Time usage
- bin-common-handler: Add DefaultSentinelTime and helpers in databasehandler"
```

---

## Task 2: Migrate bin-agent-manager Models

**Files:**
- Modify: `bin-agent-manager/models/agent/agent.go`
- Modify: `bin-agent-manager/models/agent/webhook.go`
- Modify: `bin-agent-manager/pkg/dbhandler/agent.go`
- Modify: `bin-agent-manager/pkg/dbhandler/main.go`
- Modify: Test files in `bin-agent-manager/`

**Step 1: Update the Agent struct**

In `bin-agent-manager/models/agent/agent.go`, change:

```go
// Before
TMCreate string `json:"tm_create,omitempty" db:"tm_create"`
TMUpdate string `json:"tm_update,omitempty" db:"tm_update"`
TMDelete string `json:"tm_delete,omitempty" db:"tm_delete"`

// After
TMCreate *time.Time `json:"tm_create,omitempty" db:"tm_create"`
TMUpdate *time.Time `json:"tm_update,omitempty" db:"tm_update"`
TMDelete *time.Time `json:"tm_delete,omitempty" db:"tm_delete"`
```

Add import: `"time"`

**Step 2: Update the WebhookMessage struct**

In `bin-agent-manager/models/agent/webhook.go`, make same changes to timestamp fields.

**Step 3: Update dbhandler**

In `bin-agent-manager/pkg/dbhandler/agent.go`, change:

```go
// Before
a.TMUpdate = commondatabasehandler.DefaultTimeStamp
a.TMDelete = commondatabasehandler.DefaultTimeStamp

// After
a.TMUpdate = commondatabasehandler.NewDefaultSentinel()
a.TMDelete = commondatabasehandler.NewDefaultSentinel()
```

And:
```go
// Before
a.TMCreate = now

// After (assuming now is from utilHandler.TimeNow())
a.TMCreate = h.utilHandler.TimeNow()
```

**Step 4: Update dbhandler queries**

In queries using sentinel comparison:

```go
// Before
Where(squirrel.GtOrEq{string(agent.FieldTMDelete): commondatabasehandler.DefaultTimeStamp})

// After
Where(squirrel.GtOrEq{string(agent.FieldTMDelete): commondatabasehandler.DefaultSentinelTime})
```

**Step 5: Update DefaultTimeStamp constant usage**

In `bin-agent-manager/pkg/dbhandler/main.go`, remove the local `DefaultTimeStamp` constant if it exists, use `commondatabasehandler.DefaultSentinelTime` instead.

**Step 6: Update test files**

Update all test files to use `*time.Time` instead of string for timestamp fields. Use `commondatabasehandler.NewDefaultSentinel()` for sentinel values.

**Step 7: Run verification**

Run: `cd bin-agent-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`
Expected: All pass

**Step 8: Commit**

```bash
git add bin-agent-manager/
git commit -m "NOJIRA-Timestamp-string-to-time-migration

- bin-agent-manager: Migrate timestamp fields from string to *time.Time
- bin-agent-manager: Update dbhandler to use TimeNow() and NewDefaultSentinel()
- bin-agent-manager: Update tests for new timestamp types"
```

---

## Task 3: Migrate bin-call-manager Models

**Files:**
- Modify: `bin-call-manager/models/call/call.go`
- Modify: `bin-call-manager/models/call/webhook.go`
- Modify: `bin-call-manager/models/groupcall/main.go`
- Modify: `bin-call-manager/models/channel/main.go`
- Modify: `bin-call-manager/models/recording/recording.go` (if exists)
- Modify: `bin-call-manager/models/confbridge/confbridge.go` (if exists)
- Modify: `bin-call-manager/pkg/dbhandler/*.go`
- Modify: Test files

**Step 1: Update the Call struct**

In `bin-call-manager/models/call/call.go`:

```go
// Before
TMRinging     string `json:"tm_ringing,omitempty" db:"tm_ringing"`
TMProgressing string `json:"tm_progressing,omitempty" db:"tm_progressing"`
TMHangup      string `json:"tm_hangup,omitempty" db:"tm_hangup"`
TMCreate      string `json:"tm_create,omitempty" db:"tm_create"`
TMUpdate      string `json:"tm_update,omitempty" db:"tm_update"`
TMDelete      string `json:"tm_delete,omitempty" db:"tm_delete"`

// After
TMRinging     *time.Time `json:"tm_ringing,omitempty" db:"tm_ringing"`
TMProgressing *time.Time `json:"tm_progressing,omitempty" db:"tm_progressing"`
TMHangup      *time.Time `json:"tm_hangup,omitempty" db:"tm_hangup"`
TMCreate      *time.Time `json:"tm_create,omitempty" db:"tm_create"`
TMUpdate      *time.Time `json:"tm_update,omitempty" db:"tm_update"`
TMDelete      *time.Time `json:"tm_delete,omitempty" db:"tm_delete"`
```

**Step 2: Update all other model structs in bin-call-manager**

Apply same pattern to:
- `models/groupcall/main.go`
- `models/channel/main.go`
- `models/recording/recording.go`
- `models/confbridge/confbridge.go`
- Any webhook.go files

**Step 3: Update dbhandler**

In `bin-call-manager/pkg/dbhandler/call.go`:

```go
// Before
now := h.utilHandler.TimeGetCurTime()
c.TMCreate = now
c.TMUpdate = commondatabasehandler.DefaultTimeStamp
c.TMDelete = commondatabasehandler.DefaultTimeStamp
c.TMRinging = commondatabasehandler.DefaultTimeStamp
c.TMProgressing = commondatabasehandler.DefaultTimeStamp
c.TMHangup = commondatabasehandler.DefaultTimeStamp

// After
c.TMCreate = h.utilHandler.TimeNow()
c.TMUpdate = commondatabasehandler.NewDefaultSentinel()
c.TMDelete = commondatabasehandler.NewDefaultSentinel()
c.TMRinging = commondatabasehandler.NewDefaultSentinel()
c.TMProgressing = commondatabasehandler.NewDefaultSentinel()
c.TMHangup = commondatabasehandler.NewDefaultSentinel()
```

**Step 4: Update test files**

Update all `_test.go` files to use `*time.Time`.

**Step 5: Run verification**

Run: `cd bin-call-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`
Expected: All pass

**Step 6: Commit**

```bash
git add bin-call-manager/
git commit -m "NOJIRA-Timestamp-string-to-time-migration

- bin-call-manager: Migrate timestamp fields from string to *time.Time
- bin-call-manager: Update dbhandler to use TimeNow() and NewDefaultSentinel()
- bin-call-manager: Update tests for new timestamp types"
```

---

## Task 4-23: Migrate Remaining Services

**Repeat the pattern from Tasks 2-3 for each remaining service:**

| Task | Service | Model Files |
|------|---------|-------------|
| 4 | bin-ai-manager | 8 files |
| 5 | bin-billing-manager | 4 files |
| 6 | bin-campaign-manager | 6 files |
| 7 | bin-conference-manager | 4 files |
| 8 | bin-contact-manager | 4 files |
| 9 | bin-conversation-manager | 7 files |
| 10 | bin-customer-manager | 4 files |
| 11 | bin-email-manager | 2 files |
| 12 | bin-flow-manager | 4 files |
| 13 | bin-message-manager | 2 files |
| 14 | bin-number-manager | 3 files |
| 15 | bin-outdial-manager | 5 files |
| 16 | bin-pipecat-manager | 1 file |
| 17 | bin-queue-manager | 4 files |
| 18 | bin-registrar-manager | 5 files |
| 19 | bin-route-manager | 4 files |
| 20 | bin-storage-manager | 4 files |
| 21 | bin-tag-manager | 2 files |
| 22 | bin-talk-manager | 3 files |
| 23 | bin-transcribe-manager | 4 files |
| 24 | bin-transfer-manager | 2 files |

**For each service, follow this pattern:**

1. Find all model files with timestamp fields: `grep -rn "TMCreate\s*string" bin-<service>/ --include="*.go" | grep -v vendor`
2. Update struct definitions: `string` → `*time.Time`
3. Add `import "time"` where needed
4. Update dbhandler code:
   - `TimeGetCurTime()` → `TimeNow()`
   - `commondatabasehandler.DefaultTimeStamp` → `commondatabasehandler.NewDefaultSentinel()`
   - String comparisons → `IsSentinelTime()` or nil checks
5. Update test files
6. Run verification: `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`
7. Commit with service name prefix

---

## Task 25: Full Monorepo Verification

**Step 1: Run verification for ALL services**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-Timestamp-string-to-time-migration

for dir in bin-*/; do
  if [ -f "$dir/go.mod" ]; then
    echo "=== $dir ===" && \
    (cd "$dir" && \
      go mod tidy && \
      go mod vendor && \
      go generate ./... && \
      go test ./... && \
      golangci-lint run -v --timeout 5m) || echo "FAILED: $dir"
  fi
done
```

Expected: All services pass

**Step 2: Fix any failures**

Address any compilation errors or test failures discovered.

**Step 3: Final commit**

```bash
git add .
git commit -m "NOJIRA-Timestamp-string-to-time-migration

- all: Final verification pass for timestamp migration
- all: Fix any remaining issues from full monorepo verification"
```

---

## Task 26: Cleanup Deprecated Functions (Optional)

After confirming migration is complete:

**Step 1: Search for remaining usages of old functions**

```bash
grep -rn "TimeGetCurTime()" bin-*/ --include="*.go" | grep -v vendor
```

**Step 2: If no usages found, deprecate old functions**

In `bin-common-handler/pkg/utilhandler/time.go`:

```go
// Deprecated: Use TimeNow() instead
func TimeGetCurTime() string {
	return time.Now().UTC().Format(ISO8601Layout)
}
```

**Step 3: Commit**

```bash
git add bin-common-handler/
git commit -m "NOJIRA-Timestamp-string-to-time-migration

- bin-common-handler: Deprecate TimeGetCurTime in favor of TimeNow"
```

---

## Summary of Code Change Patterns

### Pattern 1: Struct Field Change
```go
// Before
TMCreate string `json:"tm_create,omitempty" db:"tm_create"`

// After
TMCreate *time.Time `json:"tm_create,omitempty" db:"tm_create"`
```

### Pattern 2: Setting Current Time
```go
// Before
model.TMCreate = h.utilHandler.TimeGetCurTime()

// After
model.TMCreate = h.utilHandler.TimeNow()
```

### Pattern 3: Setting Sentinel Value
```go
// Before
model.TMDelete = commondatabasehandler.DefaultTimeStamp

// After
model.TMDelete = commondatabasehandler.NewDefaultSentinel()
```

### Pattern 4: Checking for Sentinel
```go
// Before
if model.TMDelete != commondatabasehandler.DefaultTimeStamp { ... }

// After
if !commondatabasehandler.IsSentinelTime(model.TMDelete) { ... }
```

### Pattern 5: Checking for Empty/Unset
```go
// Before
if model.TMCreate == "" { ... }

// After
if model.TMCreate == nil { ... }
```

### Pattern 6: SQL Query with Sentinel
```go
// Before
Where(squirrel.GtOrEq{"tm_delete": commondatabasehandler.DefaultTimeStamp})

// After
Where(squirrel.GtOrEq{"tm_delete": commondatabasehandler.DefaultSentinelTime})
```

### Pattern 7: Test Data
```go
// Before
TMCreate: "2026-01-01T00:00:00.000000Z",
TMDelete: "9999-01-01T00:00:00.000000Z",

// After
TMCreate: utilhandler.TimeNow(),
TMDelete: commondatabasehandler.NewDefaultSentinel(),
```
