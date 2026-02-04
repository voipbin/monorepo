# ISO 8601 Timestamp Migration Design

## Overview

Migrate all timestamp handling across the VoIPbin monorepo from the custom format to ISO 8601 with microsecond precision.

**Current format:** `2024-01-15 10:30:45.123456`
**Target format:** `2024-01-15T10:30:45.123456Z`

## Background

### Current State

The monorepo uses a custom timestamp format defined in `bin-common-handler/pkg/utilhandler/time.go`:

```go
layout := "2006-01-02 15:04:05.000000"  // Custom format with microseconds
```

**Usage across services:**
- 30+ services use `TimeGetCurTime()` for timestamp generation
- All timestamp fields are stored as `string` type in Go models
- Database uses MySQL `DATETIME(6)` columns (microsecond precision)
- Soft-delete pattern uses sentinel value `"9999-01-01 00:00:00.000000"`

**Exception:** `bin-timeline-manager` already uses ISO 8601 for pagination tokens (`"2006-01-02T15:04:05.000Z"` with milliseconds).

### Why ISO 8601?

- Industry standard format
- Easier integration for API consumers
- Unambiguous timezone handling (Z suffix = UTC)
- Better tooling support (parsing libraries, debugging)

## Target Format

```
2024-01-15T10:30:45.123456Z
│          │               │
│          │               └── Z = UTC timezone
│          └── T = date/time separator (ISO 8601)
└── Microsecond precision (6 decimal places)
```

## Scope of Changes

### Components to Modify

| Component | Change Required |
|-----------|-----------------|
| `bin-common-handler/pkg/utilhandler/time.go` | Update format layout in `TimeGetCurTime()`, `TimeGetCurTimeAdd()` |
| `bin-common-handler/pkg/utilhandler/time.go` | Update `TimeParse()`, `TimeParseWithError()` to accept both formats |
| `bin-common-handler/pkg/utilhandler/time.go` | Update `DefaultTimeStamp` constant |
| `bin-common-handler/pkg/utilhandler/time_test.go` | Add/update tests for new format |
| `bin-dbscheme-manager` | Alembic migration to convert existing data |
| All 30+ services | Add timestamp format validation tests |
| `bin-timeline-manager` | Align pagination tokens to microseconds |

### No Changes Required

- Model struct definitions (remain `string` type)
- JSON tags on models
- Database schema (`DATETIME(6)` unchanged)
- Individual service business logic (they call utility functions)

## Implementation Plan

### Phase 1: bin-common-handler Changes

**File:** `bin-common-handler/pkg/utilhandler/time.go`

1. **Update `TimeGetCurTime()`**
   ```go
   // Before
   layout := "2006-01-02 15:04:05.000000"

   // After
   layout := "2006-01-02T15:04:05.000000Z"
   ```

2. **Update `TimeGetCurTimeAdd()`**
   - Same layout change as above

3. **Update `TimeParse()` and `TimeParseWithError()`**
   - Accept both old and new formats for backward compatibility
   ```go
   func TimeParseWithError(timeString string) (time.Time, error) {
       // Try new format first
       layouts := []string{
           "2006-01-02T15:04:05.000000Z",  // New ISO 8601
           "2006-01-02 15:04:05.000000",   // Old format (backward compat)
       }
       for _, layout := range layouts {
           if t, err := time.Parse(layout, timeString); err == nil {
               return t, nil
           }
       }
       return time.Time{}, fmt.Errorf("unable to parse time: %s", timeString)
   }
   ```

4. **Update `DefaultTimeStamp` constant**
   ```go
   // Before
   const DefaultTimeStamp = "9999-01-01 00:00:00.000000"

   // After
   const DefaultTimeStamp = "9999-01-01T00:00:00.000000Z"
   ```

5. **Add unit tests**
   - Test new format output
   - Test parsing both formats
   - Test sentinel value

### Phase 2: Database Migration (Alembic)

**Location:** `bin-dbscheme-manager/bin-manager/main/versions/`

Create new migration file to convert all existing timestamp strings.

**Tables and columns to update:**

| Table | Columns |
|-------|---------|
| calls | tm_create, tm_update, tm_delete, tm_ringing, tm_progressing, tm_hangup |
| channels | tm_create, tm_update, tm_delete |
| contacts | tm_create, tm_update, tm_delete |
| billing_accounts | tm_create, tm_update, tm_delete |
| billing_billings | tm_create, tm_update, tm_delete |
| flows | tm_create, tm_update, tm_delete |
| activeflows | tm_create, tm_update, tm_delete |
| conferences | tm_create, tm_update, tm_delete |
| conferencecalls | tm_create, tm_update, tm_delete |
| agents | tm_create, tm_update, tm_delete |
| campaigns | tm_create, tm_update, tm_delete |
| campaigncalls | tm_create, tm_update, tm_delete |
| conversations | tm_create, tm_update, tm_delete |
| customers | tm_create, tm_update, tm_delete |
| emails | tm_create, tm_update, tm_delete |
| messages | tm_create, tm_update, tm_delete |
| numbers | tm_create, tm_update, tm_delete |
| outdials | tm_create, tm_update, tm_delete |
| outdialtargets | tm_create, tm_update, tm_delete |
| outplans | tm_create, tm_update, tm_delete |
| queues | tm_create, tm_update, tm_delete |
| queuecalls | tm_create, tm_update, tm_delete, tm_end |
| recordings | tm_create, tm_update, tm_delete |
| routes | tm_create, tm_update, tm_delete |
| storage_files | tm_create, tm_update, tm_delete |
| storage_accounts | tm_create, tm_update, tm_delete |
| tags | tm_create, tm_update, tm_delete |
| transcribes | tm_create, tm_update, tm_delete |
| users | tm_create, tm_update, tm_delete |
| extensions | tm_create, tm_update, tm_delete |
| trunks | tm_create, tm_update, tm_delete |
| registrar_sip_auths | tm_create, tm_update, tm_delete |
| ai_ais | tm_create, tm_update, tm_delete |
| ai_aicalls | tm_create, tm_update, tm_delete, tm_end |
| ai_messages | tm_create, tm_delete |
| ai_summaries | tm_create, tm_update, tm_delete |
| chatbots | tm_create, tm_update, tm_delete |
| chatbotcalls | tm_create, tm_update, tm_delete |
| chatbot_messages | tm_create, tm_delete |
| pipecat_pipecatcalls | tm_create, tm_update, tm_delete |
| talk_chats | tm_create |
| talk_messages | tm_create |
| conversation_accounts | tm_create, tm_update, tm_delete |
| conversation_conversations | tm_create, tm_update, tm_delete |
| conversation_medias | tm_create, tm_update, tm_delete |

**Migration SQL pattern:**

```sql
-- Upgrade: Convert to ISO 8601
UPDATE table_name SET
    tm_create = CONCAT(REPLACE(SUBSTRING(tm_create, 1, 10), ' ', ''), 'T', SUBSTRING(tm_create, 12), 'Z'),
    tm_update = CONCAT(REPLACE(SUBSTRING(tm_update, 1, 10), ' ', ''), 'T', SUBSTRING(tm_update, 12), 'Z'),
    tm_delete = CONCAT(REPLACE(SUBSTRING(tm_delete, 1, 10), ' ', ''), 'T', SUBSTRING(tm_delete, 12), 'Z')
WHERE tm_create IS NOT NULL;

-- Downgrade: Convert back to custom format
UPDATE table_name SET
    tm_create = CONCAT(SUBSTRING(tm_create, 1, 10), ' ', SUBSTRING(tm_create, 12, 15)),
    tm_update = CONCAT(SUBSTRING(tm_update, 1, 10), ' ', SUBSTRING(tm_update, 12, 15)),
    tm_delete = CONCAT(SUBSTRING(tm_delete, 1, 10), ' ', SUBSTRING(tm_delete, 12, 15))
WHERE tm_create IS NOT NULL;
```

### Phase 3: Service Verification & Testing

For each of the 30+ services:

1. **Update dependencies**
   ```bash
   go mod tidy && go mod vendor
   ```

2. **Add timestamp format tests**

   Example test file additions:

   ```go
   // bin-contact-manager/pkg/dbhandler/timestamp_test.go

   func TestTimestamp_Format_IsISO8601(t *testing.T) {
       iso8601Regex := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{6}Z$`)

       timestamp := utilHandler.TimeGetCurTime()
       assert.True(t, iso8601Regex.MatchString(timestamp),
           "Expected ISO 8601 format, got: %s", timestamp)
   }

   func TestTimestamp_Parse_OldFormat(t *testing.T) {
       oldFormat := "2024-01-15 10:30:45.123456"
       parsed, err := utilHandler.TimeParseWithError(oldFormat)
       assert.NoError(t, err)
       assert.False(t, parsed.IsZero())
   }

   func TestTimestamp_Parse_NewFormat(t *testing.T) {
       newFormat := "2024-01-15T10:30:45.123456Z"
       parsed, err := utilHandler.TimeParseWithError(newFormat)
       assert.NoError(t, err)
       assert.False(t, parsed.IsZero())
   }

   func TestTimestamp_Sentinel_IsISO8601(t *testing.T) {
       expected := "9999-01-01T00:00:00.000000Z"
       assert.Equal(t, expected, utilhandler.DefaultTimeStamp)
   }

   func TestTimestamp_DBRoundTrip(t *testing.T) {
       // Create record
       contact := &contact.Contact{
           ID:       uuid.New(),
           Name:     "Test Contact",
           TMCreate: utilHandler.TimeGetCurTime(),
           TMUpdate: utilhandler.DefaultTimeStamp,
           TMDelete: utilhandler.DefaultTimeStamp,
       }

       err := dbHandler.ContactCreate(ctx, contact)
       require.NoError(t, err)

       // Read back
       retrieved, err := dbHandler.ContactGet(ctx, contact.ID)
       require.NoError(t, err)

       // Verify format preserved
       iso8601Regex := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{6}Z$`)
       assert.True(t, iso8601Regex.MatchString(retrieved.TMCreate))
       assert.True(t, iso8601Regex.MatchString(retrieved.TMUpdate))
       assert.True(t, iso8601Regex.MatchString(retrieved.TMDelete))
   }
   ```

3. **Run full verification**
   ```bash
   go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
   ```

**Services requiring timestamp tests:**

- bin-ai-manager
- bin-agent-manager
- bin-billing-manager
- bin-call-manager
- bin-campaign-manager
- bin-chatbot-manager
- bin-conference-manager
- bin-contact-manager
- bin-conversation-manager
- bin-customer-manager
- bin-email-manager
- bin-flow-manager
- bin-message-manager
- bin-number-manager
- bin-outdial-manager
- bin-pipecat-manager
- bin-queue-manager
- bin-registrar-manager
- bin-route-manager
- bin-storage-manager
- bin-tag-manager
- bin-talk-manager
- bin-transcribe-manager
- bin-transfer-manager
- bin-webhook-manager

### Phase 4: Timeline-Manager Alignment

Update pagination token format from milliseconds to microseconds for consistency:

```go
// Before (bin-timeline-manager/pkg/eventhandler/event.go)
res.NextPageToken = events[pageSize-1].Timestamp.Format("2006-01-02T15:04:05.000Z")

// After
res.NextPageToken = events[pageSize-1].Timestamp.Format("2006-01-02T15:04:05.000000Z")
```

## Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| Parsing failures during transition | `TimeParse()` accepts both old and new formats |
| Missed timestamp columns in migration | Audit all Alembic migrations to enumerate every timestamp column |
| Hardcoded sentinel comparisons | Search codebase: `grep -r "9999-01-01" --include="*.go"` and update to use constant |
| ClickHouse timeline data mismatch | Uses `time.Time` internally - pagination token format change is transparent |
| Migration script errors | Test migration on staging database first |

## Hardcoded Sentinel Audit

Before implementation, run:

```bash
grep -r "9999-01-01" --include="*.go" bin-*/
```

All matches must use `utilhandler.DefaultTimeStamp` constant instead of hardcoded strings.

## Rollback Plan

If issues arise post-deployment:

1. Revert `bin-common-handler` changes
2. Run Alembic downgrade migration to convert timestamps back to custom format
3. Redeploy affected services

## Success Criteria

- [ ] All services generate timestamps in ISO 8601 format
- [ ] All services can parse both old and new formats
- [ ] Database contains only ISO 8601 formatted timestamps
- [ ] Sentinel value updated to ISO 8601 format
- [ ] All timestamp tests pass across all services
- [ ] Timeline-manager pagination uses microsecond precision
- [ ] No hardcoded sentinel strings remain in codebase

## Timeline Estimate

| Phase | Effort |
|-------|--------|
| Phase 1: bin-common-handler | Small |
| Phase 2: Database migration | Medium (many tables) |
| Phase 3: Service verification | Large (30+ services) |
| Phase 4: Timeline-manager | Small |

## References

- [ISO 8601 Wikipedia](https://en.wikipedia.org/wiki/ISO_8601)
- [Go time package layout](https://pkg.go.dev/time#pkg-constants)
- `bin-common-handler/pkg/utilhandler/time.go` - Current implementation
- `bin-timeline-manager/pkg/eventhandler/event.go` - ISO 8601 example
