# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

bin-talk-manager is a microservice within a larger VoIP monorepo that manages chat functionality with threading and reactions. It handles:
- **Chats**: Chat sessions (direct 1:1, group, and public talk channels)
- **Messages**: Text messages with threading support (replies to messages)
- **Participants**: Chat membership management with re-join support
- **Reactions**: Emoji reactions on messages with atomic operations

The service operates as a RabbitMQ-based RPC server, listening for requests and processing chat-related operations.

## Build and Run Commands

### Build
```bash
# Build the service
go build -o ./bin/talk-manager ./cmd/talk-manager

# Build the CLI tool
go build -o ./bin/talk-control ./cmd/talk-control

# Build using Docker (from monorepo root)
docker build -f bin-talk-manager/Dockerfile -t talk-manager-image .
```

### Test
```bash
# Run all tests
go test ./...

# Run tests for a specific package
go test ./pkg/chathandler/...
go test ./pkg/messagehandler/...
go test ./pkg/dbhandler/...

# Run a specific test
go test ./pkg/chathandler -run Test_ChatCreate

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

## talk-control CLI Tool

A command-line tool for managing chats and messages directly via database/cache (bypasses RabbitMQ RPC). **All output is JSON format** (stdout), logs go to stderr.

```bash
# Chat commands
./bin/talk-control chat create --customer-id <uuid> --type <direct|group|talk> --creator-type <type> --creator-id <uuid> [--name] [--detail]
./bin/talk-control chat get --id <uuid>
./bin/talk-control chat list --customer-id <uuid> [--size 100] [--token]
./bin/talk-control chat update --id <uuid> [--name] [--detail]
./bin/talk-control chat delete --id <uuid>

# Message commands
./bin/talk-control message create --chat-id <uuid> --owner-type <type> --owner-id <uuid> --type <normal|system> --text <text> [--parent-id <uuid>]
./bin/talk-control message get --id <uuid>
./bin/talk-control message list --chat-id <uuid> [--size 100] [--token]
./bin/talk-control message delete --id <uuid>

# Participant commands
./bin/talk-control participant add --chat-id <uuid> --owner-type <type> --owner-id <uuid>
./bin/talk-control participant list --customer-id <uuid> --chat-id <uuid>
./bin/talk-control participant remove --customer-id <uuid> --participant-id <uuid>

# Reaction commands
./bin/talk-control reaction add --message-id <uuid> --emoji <emoji> --owner-type <type> --owner-id <uuid>
./bin/talk-control reaction remove --message-id <uuid> --emoji <emoji> --owner-type <type> --owner-id <uuid>
```

Uses same environment variables as talk-manager (`DATABASE_DSN`, `RABBITMQ_ADDRESS`, `REDIS_ADDRESS`, etc.).

## Architecture

### Service Communication Pattern

This service uses a **RabbitMQ RPC pattern** for all external communication:
- Listens on queue: `QueueNameTalkRequest`
- Publishes events to: `QueueNameTalkEvent`
- Uses regex-based URI routing (e.g., `/v1/chats`, `/v1/messages`)
- Request/response via `sock.Request` and `sock.Response` types from `bin-common-handler`

### API Endpoints

| Method | URI | Description |
|--------|-----|-------------|
| POST | `/v1/chats` | Create a new chat |
| GET | `/v1/chats` | List chats |
| GET | `/v1/chats/{id}` | Get chat by ID |
| PUT | `/v1/chats/{id}` | Update chat |
| DELETE | `/v1/chats/{id}` | Delete chat |
| POST | `/v1/chats/{id}/participants` | Add participant to chat |
| GET | `/v1/chats/{id}/participants` | List chat participants |
| DELETE | `/v1/chats/{id}/participants/{pid}` | Remove participant |
| GET | `/v1/participants` | List participants (filtered) |
| POST | `/v1/messages` | Create a message |
| GET | `/v1/messages` | List messages |
| GET | `/v1/messages/{id}` | Get message by ID |
| DELETE | `/v1/messages/{id}` | Delete message |
| POST | `/v1/messages/{id}/reactions` | Add reaction to message |
| DELETE | `/v1/messages/{id}/reactions` | Remove reaction from message |

### Core Data Model

The chat system has a three-table structure:

1. **Chat** (`models/chat/chat.go`): The chat session
   - Contains: `customer_id`, `type`, `name`, `detail`, `member_count`
   - Types: `TypeDirect` (1:1 private), `TypeGroup` (multi-user private), `TypeTalk` (public channel)

2. **Participant** (`models/participant/participant.go`): Chat membership
   - Links users to chats via `chat_id`, `owner_type`, `owner_id`
   - UNIQUE constraint allows UPSERT for re-joining
   - Hard delete (no `tm_delete` field)

3. **Message** (`models/message/message.go`): Messages with threading and reactions
   - Threading: Optional `parent_id` for replies
   - Reactions: JSON metadata array with atomic operations
   - Media attachments: Supports address, agent, file, and link types
   - Soft delete with `tm_delete` timestamp

### Handler Architecture

The codebase follows a layered architecture with clear separation of concerns:

```
listenhandler (pkg/listenhandler/)
  ├─ Receives RabbitMQ RPC requests
  ├─ Routes based on URI/method regex matching
  └─ Delegates to domain handlers
      │
      ├─ chathandler (pkg/chathandler/)
      │   └─ Business logic for chats
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
- `talk_chats`: Main chat records
- `talk_participants`: Chat membership (with UNIQUE constraint for UPSERT)
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
2. **Parent must be in same chat** - Prevents cross-chat threading attacks
3. **INTENTIONALLY ALLOWS soft-deleted parents** - Preserves thread structure even when parent messages are deleted

```go
// From messagehandler/message.go
// CRITICAL: Validate parent is in same chat (prevents cross-chat threading)
if parent.ChatID != req.ChatID {
    return nil, errors.New("parent message must be in the same chat")
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

The participant handler uses UPSERT to allow users to leave and re-join chats. The UNIQUE constraint on `(chat_id, owner_type, owner_id)` triggers the UPSERT behavior.

**CRITICAL: MySQL vs SQLite UPSERT syntax difference**

**Production (MySQL)** uses `ON DUPLICATE KEY UPDATE`:
```sql
INSERT INTO talk_participants
(id, customer_id, chat_id, owner_type, owner_id, tm_joined)
VALUES (?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
id = VALUES(id),
tm_joined = VALUES(tm_joined)
```

**Tests (SQLite)** use `ON CONFLICT DO UPDATE`:
```sql
INSERT INTO talk_participants
(id, customer_id, chat_id, owner_type, owner_id, tm_joined)
VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT(chat_id, owner_type, owner_id) DO UPDATE SET
id = excluded.id,
tm_joined = excluded.tm_joined
```

**Important notes:**
- Production code in `pkg/dbhandler/participant.go` MUST use MySQL syntax
- Test schemas in `scripts/database_scripts/` use SQLite syntax
- Both achieve the same result: re-joining participants update `tm_joined` timestamp
- The UNIQUE constraint is required in both databases for UPSERT to work

### Timestamp Handling (REQUIRED)

**CRITICAL: ALWAYS use utilHandler.TimeGetCurTime() for timestamp generation. NEVER use time.Now().UTC().Format() directly.**

**The Rule:**
All timestamp generation for database operations MUST use `utilHandler.TimeGetCurTime()` from `bin-common-handler/pkg/utilhandler`.

**Why this matters:**
1. **MySQL compatibility** - Returns format `YYYY-MM-DD HH:MM:SS.microseconds` that MySQL expects
2. **Consistency** - All services use the same timestamp format
3. **Testability** - Timestamps can be mocked in tests for deterministic results
4. **No timezone issues** - Handles UTC conversion internally

**Correct pattern:**
```go
// In handler (dbhandler, reactionhandler, etc.)
type handler struct {
    utilHandler commonutil.UtilHandler
}

// When setting timestamps
func (h *handler) Create(ctx context.Context, resource *Resource) error {
    now := h.utilHandler.TimeGetCurTime()  // ✅ CORRECT
    resource.TMCreate = now
    resource.TMUpdate = now
    // ...
}
```

**Wrong pattern:**
```go
// ❌ WRONG - Direct time formatting
now := time.Now().UTC().Format("2006-01-02 15:04:05.000000")
resource.TMCreate = now

// ❌ WRONG - ISO 8601 format with T and Z
now := time.Now().UTC().Format("2006-01-02T15:04:05.000000Z")
resource.TMCreate = now  // Causes MySQL error!
```

**Testing pattern:**
```go
func Test_Create(t *testing.T) {
    mc := gomock.NewController(t)
    defer mc.Finish()

    mockDB := dbhandler.NewMockDBHandler(mc)
    mockUtil := commonutil.NewMockUtilHandler(mc)

    h := &handler{
        dbHandler:   mockDB,
        utilHandler: mockUtil,
    }

    // Mock timestamp generation
    mockUtil.EXPECT().TimeGetCurTime().Return("2024-01-17 10:30:00.000000")
    mockDB.EXPECT().Create(ctx, gomock.Any()).Return(nil)

    err := h.Create(ctx, resource)
    // ...
}
```

**Available utilHandler timestamp functions:**
- `TimeGetCurTime()` - Current UTC time (use this for database timestamps)
- `TimeGetCurTimeAdd(duration)` - Current time + duration
- `TimeGetCurTimeRFC3339()` - RFC3339 format (for APIs, not database)
- `TimeParse(timeString)` - Parse timestamp string to time.Time

**Related pattern:** See `bin-flow-manager` for reference implementation.

### Logging Pattern (REQUIRED)

**CRITICAL: ALWAYS use structured logging with function name and request context. NEVER use bare `logrus.Errorf()` directly.**

**The Rule:**
All handler functions MUST create a logger with context fields at the start of the function, then use that logger for all logging within the function.

**Correct pattern:**
```go
func (h *listenHandler) v1MessagesGet(ctx context.Context, m commonsock.Request) (*commonsock.Response, error) {
    log := logrus.WithFields(logrus.Fields{"func": "v1MessagesGet", "request": m})

    // ... function logic ...

    if err != nil {
        log.Errorf("Could not list the messages. err: %v", err)
        return simpleResponse(500), nil
    }

    // ... rest of function ...
}
```

**Wrong pattern:**
```go
// ❌ WRONG - No function context
func (h *listenHandler) v1MessagesGet(ctx context.Context, m commonsock.Request) (*commonsock.Response, error) {
    // ... function logic ...

    if err != nil {
        logrus.Errorf("Failed to list messages: %v", err)  // Missing function name and request!
        return simpleResponse(500), nil
    }
}
```

**Why this matters:**
1. **Debugging** - Function name in logs helps quickly identify which handler failed
2. **Request tracing** - Request context helps reproduce issues
3. **Consistency** - All logs from a function share the same context
4. **Searchability** - Can filter logs by function name

**Error message format:**
- Use format: `"Could not <action>. err: %v"`
- Examples:
  - `log.Errorf("Could not list the messages. err: %v", err)`
  - `log.Errorf("Could not get the chat. err: %v", err)`
  - `log.Errorf("Could not parse filters. err: %v", err)`

**Required fields:**
- `"func"`: Function name (e.g., `"v1MessagesGet"`)
- `"request"`: The request object `m` for RPC handlers

**Optional additional fields:**
```go
log := logrus.WithFields(logrus.Fields{
    "func":        "v1MessagesGet",
    "request":     m,
    "customer_id": customerID,  // Add relevant IDs when available
    "chat_id":     chatID,
})
```

## Testing Approach

### Test File Organization

Tests use the standard Go pattern:
- Test files live alongside source code: `handler.go` → `handler_test.go`
- Each handler package has tests: `chathandler/chat_test.go`, `messagehandler/message_test.go`, etc.
- Database tests use SQLite in-memory: `dbhandler/*_test.go` with `main_test.go` for setup
- Listenhandler tests: `listenhandler/*_test.go` testing all endpoint handlers

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
chathandler/chat.go:
  - ChatCreate()    → chat_test.go: Test_ChatCreate()
  - ChatGet()       → chat_test.go: Test_ChatGet()
  - ChatList()      → chat_test.go: Test_ChatList()
  - ChatDelete()    → chat_test.go: Test_ChatDelete()

listenhandler/v1_chats.go:
  - v1ChatsPost()          → v1_chats_test.go: Test_v1ChatsPost()
  - v1ChatsGet()           → v1_chats_test.go: Test_v1ChatsGet()
  - v1ChatsIDGet()         → v1_chats_test.go: Test_v1ChatsIDGet()
```

❌ **WRONG - Functions without tests:**
```
listenhandler/v1_chats.go has 5 functions but no test file
messagehandler/message.go has 4 functions but only Test_MessageCreate() exists
```

**Exceptions (functions that don't need separate tests):**
- Private helper functions used only within tested functions
- Trivial getters/setters (e.g., `func (c *Chat) GetID() uuid.UUID { return c.ID }`)
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
- ✅ pkg/chathandler - All CRUD functions tested
- ✅ pkg/messagehandler - All CRUD functions tested
- ✅ pkg/participanthandler - All CRUD functions tested
- ✅ pkg/reactionhandler - All CRUD functions tested
- ✅ pkg/dbhandler - All database operations tested
- ✅ pkg/listenhandler - All endpoint handlers tested

### Table-Driven Test Pattern (REQUIRED)

**All tests MUST use table-driven structure with subtests:**

```go
func Test_ChatCreate(t *testing.T) {
    tests := []struct {
        name string

        // Input fields
        customerID uuid.UUID
        chatType   chat.Type

        // Mock response fields
        responseChat *chat.Chat

        // Expected result fields
        expectRes   *chat.Chat
        expectError bool
    }{
        {
            name: "normal",
            customerID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
            chatType: chat.TypeDirect,
            responseChat: &chat.Chat{
                Identity: commonidentity.Identity{
                    ID: uuid.FromStringOrNil("6ebc6880-31da-11ed-8e95-a3bc92af9795"),
                },
            },
            expectRes: &chat.Chat{...},
        },
        {
            name: "group chat",
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
func Test_ChatCreate(t *testing.T) {
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
            h := &chatHandler{
                dbHandler:     mockDB,
                notifyHandler: mockNotify,
            }

            // Step 4: Set up expectations in call order
            ctx := context.Background()
            mockDB.EXPECT().ChatCreate(ctx, gomock.Any()).Return(nil)
            mockDB.EXPECT().ChatGet(ctx, tt.expectRes.ID).Return(tt.responseChat, nil)
            mockNotify.EXPECT().PublishWebhookEvent(ctx, gomock.Any(), chat.EventTypeChatCreated, gomock.Any())

            // Step 5: Call the function
            res, err := h.ChatCreate(ctx, tt.customerID, tt.chatType)

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
    "github.com/sirupsen/logrus"
    "github.com/smotes/purse"
)

var dbTest *sql.DB = nil

func TestMain(m *testing.M) {
    // Create in-memory SQLite database
    db, err := sql.Open("sqlite3", `file::memory:?cache=shared`)
    if err != nil {
        logrus.Fatalf("Failed to open database: %v", err)
    }
    db.SetMaxOpenConns(1)

    // Load SQL schema files from scripts/database_scripts/
    ps, err := purse.New(filepath.Join("../../scripts/database_scripts"))
    if err != nil {
        logrus.Fatalf("Failed to load SQL files: %v", err)
    }

    for _, file := range ps.Files() {
        contents, ok := ps.Get(file)
        if !ok {
            logrus.Fatalf("SQL file not loaded: %s", file)
        }
        _, err := db.Exec(contents)
        if err != nil {
            logrus.Fatalf("Failed to execute SQL: %v", err)
        }
    }

    dbTest = db
    defer db.Close()

    os.Exit(m.Run())
}
```

**Database operation tests:**
```go
func Test_ChatCreate(t *testing.T) {
    tests := []struct {
        name string
        data *chat.Chat
    }{
        {
            "normal",
            &chat.Chat{
                Identity: commonidentity.Identity{
                    ID: uuid.FromStringOrNil("..."),
                    CustomerID: uuid.FromStringOrNil("..."),
                },
                Type: chat.TypeDirect,
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
            if err := h.ChatCreate(ctx, tt.data); err != nil {
                t.Errorf("Wrong match. expect: ok, got: %v", err)
            }

            // Test retrieval to verify
            res, err := h.ChatGet(ctx, tt.data.ID)
            if err != nil {
                t.Errorf("Wrong match. expect: ok, got: %v", err)
            }

            // Clear timestamps for comparison (auto-generated)
            res.TMCreate = nil
            res.TMUpdate = nil
            tt.data.TMCreate = nil
            tt.data.TMUpdate = nil

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
- `Test_ChatCreate` - Normal cases
- `Test_ChatCreate_error` - Error cases (separate function)
- `Test_ChatGet` - Normal retrieval
- `Test_ChatGet_error` - Error retrieval
- `Test_ChatList` - List operations

**Test case names** (within table):
- Use lowercase, descriptive names: `"normal"`, `"not found"`, `"empty list"`, `"validation error"`

### Error Case Testing (REQUIRED)

**Create separate test functions for error cases:**

```go
func Test_ChatCreate(t *testing.T) {
    tests := []struct {
        name string
        // normal cases only
    }{
        {"normal", /* ... */},
        {"group chat", /* ... */},
    }
    // test implementation
}

func Test_ChatCreate_error(t *testing.T) {
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
- **chathandler**: 100% of implementation code
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
4. Use domain handlers (chathandler, messagehandler) for business logic
5. Return `sock.Response` with appropriate status code and marshaled data

### Adding a new database operation

1. Define method in `pkg/dbhandler/main.go` interface
2. Implement in corresponding file (e.g., `chat.go`, `message.go`)
3. Update mock: `cd pkg/dbhandler && go generate`
4. Add tests in `*_test.go`

### Adding a new business logic method

1. Define method in handler interface (e.g., `pkg/chathandler/main.go`)
2. Implement in handler file (e.g., `pkg/chathandler/chat.go`)
3. Add validation, database calls, webhook publishing
4. Update mock: `cd pkg/chathandler && go generate`
5. Add table-driven tests in `*_test.go`

## Monorepo Context

This is one service in a larger VoIP platform monorepo. Related services include:
- `bin-agent-manager`: Agent/user management
- `bin-message-manager`: Generic messaging
- `bin-customer-manager`: Customer/tenant management
- `bin-common-handler`: Shared utilities and models

When modifying talk-manager, be aware:
- Changes to `bin-common-handler` affect all services
- Use `replace` directives in `go.mod` for local development
- Cross-service communication happens via RabbitMQ RPC
