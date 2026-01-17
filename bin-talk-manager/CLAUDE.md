# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

bin-talk-manager is a microservice within a larger VoIP monorepo that manages talk functionality with threading and reactions. It handles:
- **Talks**: Chat sessions (1:1 and group)
- **Messages**: Text messages with threading support (replies to messages)
- **Participants**: Talk membership management with re-join support
- **Reactions**: Emoji reactions on messages with atomic operations

The service operates as a RabbitMQ-based RPC server, listening for requests and processing talk-related operations.

## Build and Run Commands

### Build
```bash
# Build the service
go build -o ./bin/talk-manager ./cmd/talk-manager

# Build using Docker (from monorepo root)
docker build -f bin-talk-manager/Dockerfile -t talk-manager-image .
```

### Test
```bash
# Run all tests
go test ./...

# Run tests for a specific package
go test ./pkg/talkhandler/...
go test ./pkg/messagehandler/...
go test ./pkg/dbhandler/...

# Run a specific test
go test ./pkg/talkhandler -run Test_TalkCreate

# Run tests with verbose output
go test -v ./...

# Run tests with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Generate mocks (required before running tests)
go generate ./...
```

### Development
```bash
# Install dependencies
go mod download
go mod tidy
go mod vendor

# Update dependencies
go get -u ./...
go mod vendor

# Generate mock files for testing
go generate ./...
```

## Architecture

### Service Communication Pattern

This service uses a **RabbitMQ RPC pattern** for all external communication:
- Listens on queue: `QueueNameTalkRequest`
- Publishes events to: `QueueNameTalkEvent`
- Uses regex-based URI routing (e.g., `/v1/talks`, `/v1/messages`)
- Request/response via `sock.Request` and `sock.Response` types from `bin-common-handler`

### Core Data Model

The talk system has a three-table structure:

1. **Talk** (`models/talk/talk.go`): The shared talk session
   - Contains: `customer_id`, `type` (normal/group)
   - Types: `TypeNormal` (1:1) and `TypeGroup` (multi-participant)

2. **Participant** (`models/participant/participant.go`): Talk membership
   - Links users to talks via `chat_id`, `owner_type`, `owner_id`
   - UNIQUE constraint allows UPSERT for re-joining
   - Hard delete (no `tm_delete` field)

3. **Message** (`models/message/message.go`): Messages with threading and reactions
   - Threading: Optional `parent_id` for replies
   - Reactions: JSON metadata array with atomic operations
   - Soft delete with `tm_delete` timestamp

### Handler Architecture

The codebase follows a layered architecture with clear separation of concerns:

```
listenhandler (pkg/listenhandler/)
  ├─ Receives RabbitMQ RPC requests
  ├─ Routes based on URI/method regex matching
  └─ Delegates to domain handlers
      │
      ├─ talkhandler (pkg/talkhandler/)
      │   └─ Business logic for talks
      │
      ├─ messagehandler (pkg/messagehandler/)
      │   └─ Business logic for messages with threading validation
      │
      ├─ participanthandler (pkg/participanthandler/)
      │   └─ Business logic for participants
      │
      └─ reactionhandler (pkg/reactionhandler/)
          └─ Business logic for reactions with atomic operations
              │
              └─ dbhandler (pkg/dbhandler/)
                  ├─ Database operations (MySQL)
                  └─ Atomic JSON operations for reactions

```

Each handler layer:
- Defines an interface (in `main.go`)
- Provides a concrete implementation
- Has a corresponding mock generated via `go:generate mockgen`

### Key Dependencies on bin-common-handler

This service heavily relies on `monorepo/bin-common-handler`:
- `models/identity`: Common identity/owner fields
- `models/sock`: RabbitMQ request/response types
- `models/outline`: Service names and queue constants
- `pkg/sockhandler`: RabbitMQ connection management
- `pkg/requesthandler`: Outgoing RPC request utilities
- `pkg/notifyhandler`: Event notification utilities
- `pkg/databasehandler`: Database connection and field utilities

### Database Schema

Tables are defined in `scripts/database_scripts/`:
- `talk_chats`: Main talk records
- `talk_participants`: Talk membership (with UNIQUE constraint for UPSERT)
- `talk_messages`: Messages with optional parent_id and metadata JSON

All tables use:
- Binary UUID (16 bytes) for IDs
- JSON columns for `metadata` (reactions) and `medias` arrays
- Timestamp fields: `tm_create`, `tm_update`, `tm_delete`

### Configuration

Runtime configuration via environment variables (see `k8s/deployment.yml`):
- `DATABASE_DSN`: MySQL connection string
- `RABBITMQ_ADDRESS`: RabbitMQ server address
- `REDIS_ADDRESS`: Redis cache server address (optional)
- `REDIS_PASSWORD`: Redis authentication (optional)
- `REDIS_DATABASE`: Redis database number (default: 1)
- `PROMETHEUS_ENDPOINT`: Metrics endpoint path
- `PROMETHEUS_LISTEN_ADDRESS`: Metrics server address

## Critical Features

### Threading Validation Policy

The message handler implements a critical threading validation policy:

1. **Parent must exist in database** - `MessageGet` verifies parent message exists
2. **Parent must be in same talk** - Prevents cross-talk threading attacks
3. **INTENTIONALLY ALLOWS soft-deleted parents** - Preserves thread structure even when parent messages are deleted

```go
// From messagehandler/message.go
// CRITICAL: Validate parent is in same talk (prevents cross-talk threading)
if parent.ChatID != req.ChatID {
    return nil, errors.New("parent message must be in the same talk")
}

// INTENTIONALLY ALLOWED: Parent can be soft-deleted (TMDelete != "")
// Reason: Preserve thread structure even when parent messages are deleted
// UI should display deleted parent as placeholder (e.g., "Message deleted")
```

This policy is thoroughly tested in `pkg/messagehandler/message_test.go`.

### Atomic Reaction Operations

The reaction handler uses atomic MySQL JSON operations to prevent race conditions:

**MessageAddReactionAtomic** uses `JSON_ARRAY_APPEND`:
```sql
UPDATE talk_messages
SET metadata = JSON_SET(
    metadata,
    '$.reactions',
    JSON_ARRAY_APPEND(
        JSON_EXTRACT(metadata, '$.reactions'),
        '$',
        CAST(? AS JSON)
    )
),
tm_update = ?
WHERE id = ?
```

This prevents lost updates when multiple users add reactions simultaneously.

### Participant Re-join Support (UPSERT)

The participant handler uses MySQL's `INSERT ... ON DUPLICATE KEY UPDATE` to allow users to leave and re-join talks:

```sql
INSERT INTO talk_participants
(id, customer_id, chat_id, owner_type, owner_id, tm_joined)
VALUES (?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
tm_joined = VALUES(tm_joined)
```

The UNIQUE constraint on `(chat_id, owner_type, owner_id)` triggers the UPSERT behavior.

## Testing Approach

### Test File Organization

Tests use the standard Go pattern:
- Test files live alongside source code: `handler.go` → `handler_test.go`
- Each handler package has tests: `talkhandler/talk_test.go`, `messagehandler/message_test.go`, etc.
- Database tests use SQLite in-memory: `dbhandler/*_test.go` with `main_test.go` for setup
- Listenhandler tests: `listenhandler/*_test.go` testing all endpoint handlers
- Total test coverage: ~6,500 lines across 12 test files

### Test Coverage Requirements (REQUIRED)

**CRITICAL: All functions MUST have their own tests unless they are clearly duplicated.**

**The Rule:**
- Every exported function in every package MUST have at least one test function
- Every endpoint handler function MUST have comprehensive test coverage
- Every business logic function MUST have tests for normal cases and error cases
- Helper functions that are clearly duplicates MAY share test coverage with their primary implementation

**Examples:**

✅ **CORRECT - All functions tested:**
```
talkhandler/talk.go:
  - TalkCreate()    → talk_test.go: Test_TalkCreate()
  - TalkGet()       → talk_test.go: Test_TalkGet()
  - TalkList()      → talk_test.go: Test_TalkList()
  - TalkDelete()    → talk_test.go: Test_TalkDelete()

listenhandler/v1_talks.go:
  - processV1Talks()       → v1_talks_test.go: Test_processV1Talks()
  - v1TalksPost()          → v1_talks_test.go: Test_processV1TalksPost()
  - v1TalksGet()           → v1_talks_test.go: Test_processV1TalksGet()
  - processV1TalksID()     → v1_talks_test.go: Test_processV1TalksIDGet()
```

❌ **WRONG - Functions without tests:**
```
listenhandler/v1_talks.go has 5 functions but no test file
messagehandler/message.go has 4 functions but only Test_MessageCreate() exists
```

**Exceptions (functions that don't need separate tests):**
- Private helper functions used only within tested functions
- Trivial getters/setters (e.g., `func (t *Talk) GetID() uuid.UUID { return t.ID }`)
- Functions that are exact duplicates (same logic, different types)

**Verification:**
When adding new code, verify every exported function has corresponding tests:
```bash
# List all exported functions
grep -n "^func.*{$" handler.go

# Verify each has a test function
grep -n "^func Test_" handler_test.go
```

**This rule applies to ALL packages:**
- ✅ pkg/talkhandler - All CRUD functions tested
- ✅ pkg/messagehandler - All CRUD functions tested
- ✅ pkg/participanthandler - All CRUD functions tested
- ✅ pkg/reactionhandler - All CRUD functions tested
- ✅ pkg/dbhandler - All database operations tested
- ✅ pkg/listenhandler - All endpoint handlers tested

### Table-Driven Test Pattern (REQUIRED)

**All tests MUST use table-driven structure with subtests:**

```go
func Test_TalkCreate(t *testing.T) {
    tests := []struct {
        name string

        // Input fields
        customerID uuid.UUID
        talkType   talk.Type

        // Mock response fields
        responseTalk *talk.Talk

        // Expected result fields
        expectRes   *talk.Talk
        expectError bool
    }{
        {
            name: "normal",
            customerID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
            talkType: talk.TypeNormal,
            responseTalk: &talk.Talk{
                Identity: commonidentity.Identity{
                    ID: uuid.FromStringOrNil("6ebc6880-31da-11ed-8e95-a3bc92af9795"),
                },
            },
            expectRes: &talk.Talk{...},
        },
        {
            name: "group talk",
            // ... test case
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

### Mock Setup Pattern (REQUIRED)

**All handler tests MUST use gomock for dependency injection:**

```go
func Test_TalkCreate(t *testing.T) {
    tests := []struct { /* ... */ }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Step 1: Create gomock controller
            mc := gomock.NewController(t)
            defer mc.Finish()

            // Step 2: Create all required mocks
            mockDB := dbhandler.NewMockDBHandler(mc)
            mockNotify := notifyhandler.NewMockNotifyHandler(mc)

            // Step 3: Create handler with mocks injected
            h := &talkHandler{
                dbHandler:     mockDB,
                notifyHandler: mockNotify,
            }

            // Step 4: Set up expectations in call order
            ctx := context.Background()
            mockDB.EXPECT().TalkCreate(ctx, gomock.Any()).Return(nil)
            mockDB.EXPECT().TalkGet(ctx, tt.expectRes.ID).Return(tt.responseTalk, nil)
            mockNotify.EXPECT().PublishWebhookEvent(ctx, gomock.Any(), talk.EventTypeTalkCreated, gomock.Any())

            // Step 5: Call the function
            res, err := h.TalkCreate(ctx, tt.customerID, tt.talkType)

            // Step 6: Assert results
            if err != nil {
                t.Errorf("Wrong match. expect: ok, got: %v", err)
            }
            if !reflect.DeepEqual(res, tt.expectRes) {
                t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
            }
        })
    }
}
```

### Assertion Patterns (REQUIRED)

**Standard error assertion:**
```go
if err != nil {
    t.Errorf("Wrong match. expect: ok, got: %v", err)
}
```

**Result comparison using reflect.DeepEqual:**
```go
if !reflect.DeepEqual(res, tt.expectRes) {
    t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
}
```

**Error expectation:**
```go
if err == nil {
    t.Errorf("Wrong match. expect: error, got: nil")
}
```

### Database Testing Pattern

**Database tests use SQLite in-memory database (NOT mocks):**

```go
// main_test.go - Required setup file
package dbhandler

import (
    "database/sql"
    "os"
    "path/filepath"
    "testing"

    _ "github.com/mattn/go-sqlite3"
    log "github.com/sirupsen/logrus"
    "github.com/smotes/purse"
)

var dbTest *sql.DB = nil

func TestMain(m *testing.M) {
    // Create in-memory SQLite database
    db, err := sql.Open("sqlite3", `file::memory:?cache=shared`)
    if err != nil {
        log.Fatalf("Failed to open database: %v", err)
    }
    db.SetMaxOpenConns(1)

    // Load SQL schema files from scripts/database_scripts/
    ps, err := purse.New(filepath.Join("../../scripts/database_scripts"))
    if err != nil {
        log.Fatalf("Failed to load SQL files: %v", err)
    }

    for _, file := range ps.Files() {
        contents, ok := ps.Get(file)
        if !ok {
            log.Fatalf("SQL file not loaded: %s", file)
        }
        _, err := db.Exec(contents)
        if err != nil {
            log.Fatalf("Failed to execute SQL: %v", err)
        }
    }

    dbTest = db
    defer db.Close()

    os.Exit(m.Run())
}
```

**Database operation tests:**
```go
func Test_TalkCreate(t *testing.T) {
    tests := []struct {
        name string
        data *talk.Talk
    }{
        {
            "normal",
            &talk.Talk{
                Identity: commonidentity.Identity{
                    ID: uuid.FromStringOrNil("..."),
                    CustomerID: uuid.FromStringOrNil("..."),
                },
                Type: talk.TypeNormal,
            },
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            h := &dbHandler{
                db: dbTest,  // Use in-memory test database
            }
            ctx := context.Background()

            // Test create
            if err := h.TalkCreate(ctx, tt.data); err != nil {
                t.Errorf("Wrong match. expect: ok, got: %v", err)
            }

            // Test retrieval to verify
            res, err := h.TalkGet(ctx, tt.data.ID)
            if err != nil {
                t.Errorf("Wrong match. expect: ok, got: %v", err)
            }

            // Clear timestamps for comparison (auto-generated)
            res.TMCreate = ""
            res.TMUpdate = ""
            tt.data.TMCreate = ""
            tt.data.TMUpdate = ""

            if !reflect.DeepEqual(tt.data, res) {
                t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.data, res)
            }
        })
    }
}
```

### Test Function Naming (REQUIRED)

**Pattern:** `Test_<FunctionName>` or `Test_<FunctionName>_error`

Examples:
- `Test_TalkCreate` - Normal cases
- `Test_TalkCreate_error` - Error cases (separate function)
- `Test_TalkGet` - Normal retrieval
- `Test_TalkGet_error` - Error retrieval
- `Test_TalkList` - List operations

**Test case names** (within table):
- Use lowercase, descriptive names: `"normal"`, `"not found"`, `"empty list"`, `"validation error"`

### Error Case Testing (REQUIRED)

**Create separate test functions for error cases:**

```go
func Test_TalkCreate(t *testing.T) {
    tests := []struct {
        name string
        // normal cases only
    }{
        {"normal", /* ... */},
        {"group talk", /* ... */},
    }
    // test implementation
}

func Test_TalkCreate_error(t *testing.T) {
    tests := []struct {
        name string
        // error cases only
    }{
        {"nil customer id", /* ... */},
        {"invalid type", /* ... */},
        {"database failure", /* ... */},
    }
    // test implementation with error expectations
}
```

### Critical Test Requirements

1. **Use concrete UUIDs** - NEVER use random UUID generation, always use `uuid.FromStringOrNil()` with real UUID strings
2. **Mock all dependencies** - Handler tests mock dbHandler, notifyHandler, etc.
3. **Test actual SQL** - Database tests use SQLite in-memory, not mocks
4. **Clear timestamps** - Before comparison, clear auto-generated timestamp fields
5. **Test validation** - Include tests for all validation rules (nil checks, empty strings, type checks)
6. **Test error paths** - Separate error test functions with proper error assertions
7. **Test critical features** - Threading validation, atomic operations, UPSERT behavior

### Test Coverage Expectations

For bin-talk-manager test coverage:
- **talkhandler**: 100% of implementation code
- **messagehandler**: 66% (complex threading logic)
- **participanthandler**: ~80%
- **reactionhandler**: ~75%
- **dbhandler**: Actual SQL operations tested with SQLite

All critical code paths (validation, threading, atomic operations) are covered.

## Common Patterns

### Creating a new endpoint

1. Add regex pattern to `pkg/listenhandler/main.go`
2. Add case to `processRequest()` switch statement
3. Implement handler method in corresponding file
4. Use domain handlers (talkhandler, messagehandler) for business logic
5. Return `sock.Response` with appropriate status code and marshaled data

### Adding a new database operation

1. Define method in `pkg/dbhandler/main.go` interface
2. Implement in corresponding file (e.g., `talk.go`, `message.go`)
3. Update mock: `cd pkg/dbhandler && go generate`
4. Add tests in `*_test.go`

### Adding a new business logic method

1. Define method in handler interface (e.g., `pkg/talkhandler/main.go`)
2. Implement in handler file (e.g., `pkg/talkhandler/talk.go`)
3. Add validation, database calls, webhook publishing
4. Update mock: `cd pkg/talkhandler && go generate`
5. Add table-driven tests in `*_test.go`

## Monorepo Context

This is one service in a larger VoIP platform monorepo. Related services include:
- `bin-agent-manager`: Agent/user management
- `bin-message-manager`: Generic messaging (deprecated, replaced by bin-talk-manager)
- `bin-chat-manager`: Web chat (legacy, functionality merged into bin-talk-manager)
- `bin-customer-manager`: Customer/tenant management
- `bin-common-handler`: Shared utilities and models

When modifying talk-manager, be aware:
- Changes to `bin-common-handler` affect all services
- Use `replace` directives in `go.mod` for local development
- Cross-service communication happens via RabbitMQ RPC
