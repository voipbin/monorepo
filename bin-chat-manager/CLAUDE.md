# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

bin-chat-manager is a microservice within a larger VoIP monorepo that manages chat functionality. It handles:
- **Chats**: 1:1 and group chat sessions
- **Chatrooms**: Individual user views of chats with participant-specific data
- **Messages**: Chat and chatroom message management

The service operates as a RabbitMQ-based RPC server, listening for requests and processing chat-related operations.

## Build and Run Commands

### Build
```bash
# Build the service
go build -o ./bin/chat-manager ./cmd/chat-manager

# Build using Docker (from monorepo root)
docker build -f bin-chat-manager/Dockerfile -t chat-manager-image .
```

### Test
```bash
# Run all tests
go test ./...

# Run tests for a specific package
go test ./pkg/chathandler/...
go test ./pkg/dbhandler/...

# Run a specific test
go test ./pkg/chathandler -run TestChatCreate

# Run tests with verbose output
go test -v ./...

# Generate mocks (required before running tests)
go generate ./...
```

### Development
```bash
# Install dependencies
go mod download
go mod vendor

# Update dependencies
go get -u ./...
go mod vendor

# Generate mock files for testing
go generate ./...
```

## chat-control CLI Tool

A command-line tool for managing chats directly via database/cache (bypasses RabbitMQ RPC). **All output is JSON format** (stdout), logs go to stderr.

```bash
# Create chat - returns created chat JSON
./bin/chat-control chat create --customer_id <uuid> [--type normal] [--name <name>] [--detail]

# Get chat - returns chat JSON
./bin/chat-control chat get --id <uuid>

# List chats - returns JSON array
./bin/chat-control chat list --customer_id <uuid> [--limit 100] [--token]

# Update chat basic info - returns updated chat JSON
./bin/chat-control chat update-basic-info --id <uuid> [--name <name>] [--detail]

# Update chat room owner - returns updated chat JSON
./bin/chat-control chat update-room-owner --id <uuid> --room_owner_id <uuid>

# Add participant - returns updated chat JSON
./bin/chat-control chat add-participant --id <uuid> --participant_id <uuid>

# Remove participant - returns updated chat JSON
./bin/chat-control chat remove-participant --id <uuid> --participant_id <uuid>

# Delete chat - returns deleted chat JSON
./bin/chat-control chat delete --id <uuid>
```

Uses same environment variables as chat-manager (`DATABASE_DSN`, `RABBITMQ_ADDRESS`, `REDIS_ADDRESS`, etc.).

## Architecture

### Service Communication Pattern

This service uses a **RabbitMQ RPC pattern** for all external communication:
- Listens on queue: `QueueNameChatRequest`
- Publishes events to: `QueueNameChatEvent`
- Uses regex-based URI routing (e.g., `/v1/chats`, `/v1/chatrooms`)
- Request/response via `sock.Request` and `sock.Response` types from `bin-common-handler`

### Core Data Model

The chat system has a hierarchical structure:

1. **Chat** (`models/chat/chat.go`): The shared chat session
   - Contains: `customer_id`, `type` (normal/group), `room_owner_id`, `participant_ids`
   - Types: `TypeNormal` (1:1) and `TypeGroup` (multi-participant)

2. **Chatroom** (`models/chatroom/chatroom.go`): User-specific view of a chat
   - Each participant gets their own chatroom record
   - Links to parent chat via `chat_id`
   - Contains participant-specific metadata via `owner_type` and `owner_id`

3. **Messages**: Two types
   - `Messagechat`: Messages at the chat level
   - `Messagechatroom`: Messages at the chatroom level

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
      ├─ chatroomhandler (pkg/chatroomhandler/)
      │   └─ Business logic for chatrooms
      │
      ├─ messagechathandler (pkg/messagechathandler/)
      │   └─ Business logic for chat messages
      │
      └─ messagechatroomhandler (pkg/messagechatroomhandler/)
          └─ Business logic for chatroom messages
              │
              └─ dbhandler (pkg/dbhandler/)
                  ├─ Database operations (MySQL)
                  └─ Cache operations (Redis via cachehandler)
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
- `pkg/databasehandler`: Database connection utilities

### Database Schema

Tables are defined in `scripts/database_scripts/`:
- `chat_chats`: Main chat records
- `chat_chatrooms`: Per-user chatroom views
- `chat_messagechats`: Chat-level messages
- `chat_messagechatrooms`: Chatroom-level messages

All tables use:
- Binary UUID (16 bytes) for IDs
- JSON columns for `participant_ids` arrays
- Timestamp fields: `tm_create`, `tm_update`, `tm_delete`

### Configuration

Runtime configuration via environment variables (see `k8s/deployment.yml`):
- `DATABASE_DSN`: MySQL connection string
- `RABBITMQ_ADDRESS`: RabbitMQ server address
- `REDIS_ADDRESS`: Redis cache server address
- `REDIS_PASSWORD`: Redis authentication
- `REDIS_DATABASE`: Redis database number (default: 1)
- `PROMETHEUS_ENDPOINT`: Metrics endpoint path
- `PROMETHEUS_LISTEN_ADDRESS`: Metrics server address

## Monorepo Context

This is one service in a larger VoIP platform monorepo. Related services include:
- `bin-agent-manager`: Agent/user management
- `bin-message-manager`: Generic messaging
- `bin-customer-manager`: Customer/tenant management
- `bin-common-handler`: Shared utilities and models

When modifying chat-manager, be aware:
- Changes to `bin-common-handler` affect all services
- Use `replace` directives in `go.mod` for local development
- Cross-service communication happens via RabbitMQ RPC

## Testing Approach

Tests use:
- `go.uber.org/mock` for mocking dependencies
- Table-driven tests (common pattern in `*_test.go` files)
- Mock generation via `go:generate` comments at package level

Before running tests, always run `go generate ./...` to ensure mocks are up-to-date.

## Common Patterns

### Creating a new endpoint

1. Add regex pattern to `pkg/listenhandler/main.go` (e.g., `regV1ChatsID`)
2. Add case to `processRequest()` switch statement
3. Implement handler method (e.g., `v1ChatsIDGet()`) in corresponding file
4. Use domain handlers (chathandler, chatroomhandler) for business logic
5. Return `sock.Response` with appropriate status code and marshaled data

### Adding a new database operation

1. Define method in `pkg/dbhandler/main.go` interface
2. Implement in corresponding file (e.g., `chat.go`, `chatroom.go`)
3. Update mock: `cd pkg/dbhandler && go generate`
4. Add tests in `*_test.go`
