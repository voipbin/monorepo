# Timestamp Migration Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Migrate all timestamp fields from `string` to `*time.Time` across 23 services and 93 model files, using `nil` for unset values.

**Architecture:** Direct type change in Go structs. Database already uses `DATETIME(6)`. Alembic migration converts sentinel values (`9999-01-01`) to `NULL`. Go code uses `nil` to represent "not set".

**Tech Stack:** Go 1.21+, MySQL DATETIME(6), standard library `time` package, `database/sql` driver, Alembic migrations.

**Semantics:**
- `TMCreate`: Always set when record is created
- `TMUpdate`: `nil` = never updated, `*time.Time` = last update time
- `TMDelete`: `nil` = not deleted (active), `*time.Time` = deleted at this time

---

## Task 1: Add New Time Utilities to bin-common-handler

**Files:**
- Modify: `bin-common-handler/pkg/utilhandler/time.go`
- Modify: `bin-common-handler/pkg/utilhandler/time_test.go`

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

func Test_IsDeleted(t *testing.T) {
	tests := []struct {
		name   string
		input  *time.Time
		expect bool
	}{
		{"nil means not deleted", nil, false},
		{"non-nil means deleted", TimeNow(), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsDeleted(tt.input)
			if result != tt.expect {
				t.Errorf("IsDeleted(%v) = %v, want %v", tt.input, result, tt.expect)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd bin-common-handler && go test -v ./pkg/utilhandler/... -run "Test_TimeNow|Test_TimeNowAdd|Test_IsDeleted"`
Expected: FAIL with undefined functions

**Step 3: Write minimal implementation**

Add to `bin-common-handler/pkg/utilhandler/time.go`:

```go
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

// IsDeleted returns true if the timestamp is not nil (record has been deleted)
// nil means not deleted, non-nil means deleted at that time
func IsDeleted(t *time.Time) bool {
	return t != nil
}

// Method versions for utilHandler interface
func (h *utilHandler) TimeNow() *time.Time {
	return TimeNow()
}

func (h *utilHandler) TimeNowAdd(d time.Duration) *time.Time {
	return TimeNowAdd(d)
}

func (h *utilHandler) IsDeleted(t *time.Time) bool {
	return IsDeleted(t)
}
```

**Step 4: Update UtilHandler interface**

Add to interface in `bin-common-handler/pkg/utilhandler/main.go`:

```go
// Add these methods to the UtilHandler interface
TimeNow() *time.Time
TimeNowAdd(d time.Duration) *time.Time
IsDeleted(t *time.Time) bool
```

**Step 5: Run test to verify it passes**

Run: `cd bin-common-handler && go test -v ./pkg/utilhandler/... -run "Test_TimeNow|Test_TimeNowAdd|Test_IsDeleted"`
Expected: PASS

**Step 6: Run full verification**

Run: `cd bin-common-handler && go mod tidy && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`
Expected: All pass

**Step 7: Commit**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-Timestamp-string-to-time-migration
git add bin-common-handler/
git commit -m "NOJIRA-Timestamp-string-to-time-migration

- bin-common-handler: Add TimeNow, TimeNowAdd utilities for *time.Time
- bin-common-handler: Add IsDeleted helper (nil = not deleted, non-nil = deleted)"
```

---

## Task 2: Create Alembic Migration to Convert Sentinel to NULL

**Files:**
- Create: `bin-dbscheme-manager/bin-manager/main/versions/XXXXXX_timestamp_sentinel_to_null.py`

**Step 1: Create the migration file**

```bash
cd bin-dbscheme-manager/bin-manager/main
alembic -c alembic.ini revision -m "timestamp_sentinel_to_null"
```

**Step 2: Edit the migration file**

```python
"""timestamp_sentinel_to_null

Revision ID: <generated>
Revises: <previous>
Create Date: <generated>

"""
from alembic import op

# revision identifiers
revision = '<generated>'
down_revision = '<previous>'
branch_labels = None
depends_on = None

# List of all tables with tm_update and tm_delete columns
TABLES_WITH_TIMESTAMPS = [
    'agent_agents',
    'ai_ais',
    'ai_aicalls',
    'ai_messages',
    'ai_summaries',
    'billing_accounts',
    'billing_billings',
    'call_calls',
    'call_channels',
    'call_confbridges',
    'call_groupcalls',
    'call_recordings',
    'campaign_campaigns',
    'campaign_campaigncalls',
    'campaign_outplans',
    'conference_conferences',
    'conference_conferencecalls',
    'contact_contacts',
    'contact_contact_numbers',
    'contact_contact_emails',
    'contact_contact_addresses',
    'contact_contact_activities',
    'contact_contact_groups',
    'conversation_accounts',
    'conversation_conversations',
    'conversation_medias',
    'customer_customers',
    'customer_accesskeys',
    'email_emails',
    'flow_flows',
    'flow_activeflows',
    'message_messages',
    'number_numbers',
    'outdial_outdials',
    'outdial_outdialtargets',
    'pipecat_pipecatcalls',
    'queue_queues',
    'queue_queuecalls',
    'registrar_sip_auths',
    'registrar_trunks',
    'route_routes',
    'storage_accounts',
    'storage_files',
    'tag_tags',
    'talk_chats',
    'talk_chatmembers',
    'talk_messages',
    'transcribe_transcribes',
    'transfer_transfers',
]

# Sentinel value patterns (both formats for safety)
SENTINEL_VALUES = [
    '9999-01-01 00:00:00.000000',
    '9999-01-01T00:00:00.000000Z',
]


def upgrade():
    """Convert sentinel timestamp values to NULL."""
    for table in TABLES_WITH_TIMESTAMPS:
        for sentinel in SENTINEL_VALUES:
            # Update tm_update
            op.execute(f"""
                UPDATE `{table}`
                SET tm_update = NULL
                WHERE tm_update = '{sentinel}'
            """)
            # Update tm_delete
            op.execute(f"""
                UPDATE `{table}`
                SET tm_delete = NULL
                WHERE tm_delete = '{sentinel}'
            """)


def downgrade():
    """Convert NULL back to sentinel timestamp values."""
    sentinel = '9999-01-01T00:00:00.000000Z'
    for table in TABLES_WITH_TIMESTAMPS:
        # Revert tm_update
        op.execute(f"""
            UPDATE `{table}`
            SET tm_update = '{sentinel}'
            WHERE tm_update IS NULL
        """)
        # Revert tm_delete
        op.execute(f"""
            UPDATE `{table}`
            SET tm_delete = '{sentinel}'
            WHERE tm_delete IS NULL
        """)
```

**Step 3: Verify migration file syntax**

Run: `cd bin-dbscheme-manager/bin-manager/main && python -m py_compile versions/<migration_file>.py`
Expected: No syntax errors

**Step 4: Commit (DO NOT run alembic upgrade)**

```bash
git add bin-dbscheme-manager/
git commit -m "NOJIRA-Timestamp-string-to-time-migration

- bin-dbscheme-manager: Add migration to convert sentinel timestamps to NULL
- bin-dbscheme-manager: Affects tm_update and tm_delete in all tables"
```

**IMPORTANT:** Do NOT run `alembic upgrade`. The migration will be applied manually by a human with proper authorization and database access.

---

## Task 3: Migrate bin-agent-manager Models

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
now := h.utilHandler.TimeGetCurTime()
a.TMCreate = now
a.TMUpdate = commondatabasehandler.DefaultTimeStamp
a.TMDelete = commondatabasehandler.DefaultTimeStamp

// After - REMOVE the TMUpdate/TMDelete lines entirely, they default to nil
a.TMCreate = h.utilHandler.TimeNow()
// DO NOT set TMUpdate or TMDelete - nil is the correct default
```

**IMPORTANT:** Do NOT replace sentinel assignments with `= nil`. Simply **delete** those lines. The `*time.Time` fields default to `nil` automatically.

**Step 4: Update dbhandler queries**

In queries filtering for non-deleted records:

```go
// Before
Where(squirrel.GtOrEq{string(agent.FieldTMDelete): commondatabasehandler.DefaultTimeStamp})

// After (NULL means not deleted)
Where(squirrel.Eq{string(agent.FieldTMDelete): nil})
```

**Step 5: Remove DefaultTimeStamp constant**

In `bin-agent-manager/pkg/dbhandler/main.go`, remove the local `DefaultTimeStamp` constant - it's no longer needed.

**Step 6: Update test files**

Update all test files to use `*time.Time` instead of string for timestamp fields:

```go
// Before
TMCreate: "2026-01-01T00:00:00.000000Z",
TMUpdate: DefaultTimeStamp,
TMDelete: DefaultTimeStamp,

// After
TMCreate: utilhandler.TimeNow(),
TMUpdate: nil,
TMDelete: nil,
```

**Step 7: Run verification**

Run: `cd bin-agent-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`
Expected: All pass

**Step 8: Commit**

```bash
git add bin-agent-manager/
git commit -m "NOJIRA-Timestamp-string-to-time-migration

- bin-agent-manager: Migrate timestamp fields from string to *time.Time
- bin-agent-manager: Use nil for unset TMUpdate/TMDelete
- bin-agent-manager: Update queries to check for NULL instead of sentinel"
```

---

## Task 4: Migrate bin-call-manager Models

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

// After - REMOVE all sentinel assignments, only set TMCreate
c.TMCreate = h.utilHandler.TimeNow()
// DO NOT set TMUpdate, TMDelete, TMRinging, TMProgressing, TMHangup
// They default to nil and will be set when the event actually occurs
```

**IMPORTANT:** Delete the sentinel assignment lines. Do NOT replace with `= nil`.

**Step 4: Update test files**

Update all `_test.go` files to use `*time.Time` and `nil` for unset values.

**Step 5: Run verification**

Run: `cd bin-call-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`
Expected: All pass

**Step 6: Commit**

```bash
git add bin-call-manager/
git commit -m "NOJIRA-Timestamp-string-to-time-migration

- bin-call-manager: Migrate timestamp fields from string to *time.Time
- bin-call-manager: Use nil for unset event timestamps (TMRinging, TMProgressing, TMHangup)
- bin-call-manager: Use nil for TMUpdate/TMDelete"
```

---

## Tasks 5-25: Migrate Remaining Services

**Repeat the pattern from Tasks 3-4 for each remaining service:**

| Task | Service | Model Files |
|------|---------|-------------|
| 5 | bin-ai-manager | 8 files |
| 6 | bin-billing-manager | 4 files |
| 7 | bin-campaign-manager | 6 files |
| 8 | bin-conference-manager | 4 files |
| 9 | bin-contact-manager | 4 files |
| 10 | bin-conversation-manager | 7 files |
| 11 | bin-customer-manager | 4 files |
| 12 | bin-email-manager | 2 files |
| 13 | bin-flow-manager | 4 files |
| 14 | bin-message-manager | 2 files |
| 15 | bin-number-manager | 3 files |
| 16 | bin-outdial-manager | 5 files |
| 17 | bin-pipecat-manager | 1 file |
| 18 | bin-queue-manager | 4 files |
| 19 | bin-registrar-manager | 5 files |
| 20 | bin-route-manager | 4 files |
| 21 | bin-storage-manager | 4 files |
| 22 | bin-tag-manager | 2 files |
| 23 | bin-talk-manager | 3 files |
| 24 | bin-transcribe-manager | 4 files |
| 25 | bin-transfer-manager | 2 files |

**For each service, follow this pattern:**

1. Find all model files with timestamp fields: `grep -rn "TMCreate\s*string" bin-<service>/ --include="*.go" | grep -v vendor`
2. Update struct definitions: `string` → `*time.Time`
3. Add `import "time"` where needed
4. Update dbhandler Create functions:
   - `TimeGetCurTime()` → `TimeNow()` for TMCreate
   - **DELETE** all `= commondatabasehandler.DefaultTimeStamp` lines (don't replace with nil)
5. Update dbhandler Update functions:
   - When updating, set `TMUpdate = h.utilHandler.TimeNow()`
6. Update dbhandler Delete functions:
   - When soft-deleting, set `TMDelete = h.utilHandler.TimeNow()`
7. Update SQL queries: `WHERE tm_delete >= '9999...'` → `WHERE tm_delete IS NULL`
8. Update sentinel comparisons → `nil` checks or `IsDeleted()`
9. Update test files (remove sentinel values, use nil for unset)
10. Run verification: `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`
11. Commit with service name prefix

---

## Task 26: Update OpenAPI Schema

**Files:**
- Modify: `bin-openapi-manager/openapi/openapi.yaml`

**Step 1: Update tm_create fields (non-nullable)**

Add `format: date-time` to all `tm_create` fields:

```yaml
# Before
tm_create:
  type: string
  description: Created timestamp.

# After
tm_create:
  type: string
  format: date-time
  description: Created timestamp.
```

**Step 2: Update tm_update, tm_delete fields (nullable)**

Add `format: date-time` and `nullable: true` to all nullable timestamp fields:

```yaml
# Before
tm_update:
  type: string
  description: Updated timestamp.
tm_delete:
  type: string
  description: Deleted timestamp.

# After
tm_update:
  type: string
  format: date-time
  nullable: true
  description: Updated timestamp. Null if never updated.
tm_delete:
  type: string
  format: date-time
  nullable: true
  description: Deleted timestamp. Null if not deleted.
```

**Step 3: Update event timestamps (nullable)**

Apply same pattern to event timestamps like `tm_ringing`, `tm_progressing`, `tm_hangup`:

```yaml
# After
tm_ringing:
  type: string
  format: date-time
  nullable: true
  description: Timestamp when call started ringing. Null if not yet ringing.
tm_progressing:
  type: string
  format: date-time
  nullable: true
  description: Timestamp when call was answered. Null if not yet answered.
tm_hangup:
  type: string
  format: date-time
  nullable: true
  description: Timestamp when call ended. Null if still active.
```

**Step 4: Regenerate OpenAPI code**

Run: `cd bin-openapi-manager && go generate ./...`

**Step 5: Run verification**

Run: `cd bin-openapi-manager && go mod tidy && go test ./... && golangci-lint run -v --timeout 5m`
Expected: All pass

**Step 6: Commit**

```bash
git add bin-openapi-manager/
git commit -m "NOJIRA-Timestamp-string-to-time-migration

- bin-openapi-manager: Add format: date-time to all timestamp fields
- bin-openapi-manager: Add nullable: true to tm_update, tm_delete, and event timestamps
- bin-openapi-manager: Update descriptions to clarify null semantics"
```

---

## Task 27: Full Monorepo Verification

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

## Task 28: Cleanup Deprecated Functions (Optional)

After confirming migration is complete:

**Step 1: Search for remaining usages of old functions**

```bash
grep -rn "TimeGetCurTime()" bin-*/ --include="*.go" | grep -v vendor
grep -rn "DefaultTimeStamp" bin-*/ --include="*.go" | grep -v vendor
```

**Step 2: If no usages found, deprecate old functions**

In `bin-common-handler/pkg/utilhandler/time.go`:

```go
// Deprecated: Use TimeNow() instead
func TimeGetCurTime() string {
	return time.Now().UTC().Format(ISO8601Layout)
}
```

In `bin-common-handler/pkg/databasehandler/main.go`:

```go
// Deprecated: Use nil instead. This constant is only kept for backward compatibility.
const DefaultTimeStamp = "9999-01-01T00:00:00.000000Z"
```

**Step 3: Commit**

```bash
git add bin-common-handler/
git commit -m "NOJIRA-Timestamp-string-to-time-migration

- bin-common-handler: Deprecate TimeGetCurTime in favor of TimeNow
- bin-common-handler: Deprecate DefaultTimeStamp (use nil instead)"
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

### Pattern 3: Remove Sentinel Assignments (TMUpdate, TMDelete, event timestamps)
```go
// Before - explicitly setting sentinel
model.TMUpdate = commondatabasehandler.DefaultTimeStamp
model.TMDelete = commondatabasehandler.DefaultTimeStamp
model.TMRinging = commondatabasehandler.DefaultTimeStamp

// After - DELETE these lines entirely
// *time.Time fields default to nil, no assignment needed
```

**IMPORTANT:** Do NOT replace with `= nil`. Simply remove the lines.

### Pattern 4: Checking if Deleted
```go
// Before
if model.TMDelete != commondatabasehandler.DefaultTimeStamp { ... }

// After
if utilhandler.IsDeleted(model.TMDelete) { ... }
// Or simply:
if model.TMDelete != nil { ... }
```

### Pattern 5: Checking for Empty/Unset
```go
// Before
if model.TMCreate == "" { ... }

// After
if model.TMCreate == nil { ... }
```

### Pattern 6: SQL Query for Non-Deleted Records
```go
// Before
Where(squirrel.GtOrEq{"tm_delete": commondatabasehandler.DefaultTimeStamp})

// After
Where(squirrel.Eq{"tm_delete": nil})
```

### Pattern 7: Test Data
```go
// Before
agent := &agent.Agent{
    TMCreate: "2026-01-01T00:00:00.000000Z",
    TMUpdate: "9999-01-01T00:00:00.000000Z",
    TMDelete: "9999-01-01T00:00:00.000000Z",
}

// After - omit TMUpdate/TMDelete entirely (nil by default)
agent := &agent.Agent{
    TMCreate: utilhandler.TimeNow(),
    // TMUpdate and TMDelete omitted - they're nil by default
}

// Or if you need to be explicit in tests:
agent := &agent.Agent{
    TMCreate: utilhandler.TimeNow(),
    TMUpdate: nil,
    TMDelete: nil,
}
```

### Pattern 8: Setting Delete Time (Soft Delete)
```go
// Before
model.TMDelete = h.utilHandler.TimeGetCurTime()

// After
model.TMDelete = h.utilHandler.TimeNow()
```
