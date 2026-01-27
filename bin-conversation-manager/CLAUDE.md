# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

This is the `bin-conversation-manager` service, part of the VoIPbin monorepo. It manages multi-channel conversations (SMS/MMS, LINE messaging) and their messages, handling bidirectional communication with external messaging platforms.

**Key Concepts:**
- **Conversation**: A communication thread between two parties (self/peer) with associated account credentials and dialog context
- **Message**: Individual text/media content within a conversation, tracked with direction (incoming/outgoing) and status (progressing/done/failed)
- **Account**: Platform-specific credentials (LINE channel secret/token, SMS provider credentials) for sending messages
- **Dialog ID**: External platform conversation identifier (LINE chatroom ID, SMS thread ID)

## Architecture

### Service Communication Pattern

This service uses **RabbitMQ for RPC-style communication**:
- **ListenHandler** (`pkg/listenhandler/`): Consumes RPC requests from queue `bin-manager.conversation-manager.request`, processes them, and returns responses
- **SubscribeHandler** (`pkg/subscribehandler/`): Subscribes to events from message-manager (new SMS/MMS messages) to create/update conversations
- **NotifyHandler**: Publishes events to exchange `bin-manager.conversation-manager.event` when conversation/message state changes

### Core Components

```
cmd/conversation-manager/main.go
    ├── createDBHandler() -> pkg/dbhandler (MySQL + Redis cache)
    ├── run()
        ├── pkg/accounthandler (Manage messaging platform accounts)
        ├── pkg/conversationhandler (Business logic for conversations)
        ├── pkg/messagehandler (Business logic for messages)
        ├── pkg/linehandler (LINE Bot SDK integration)
        ├── pkg/smshandler (SMS/MMS sending via message-manager)
        ├── runListen() -> pkg/listenhandler
        └── runSubscribe() -> pkg/subscribehandler
```

**Layer Responsibilities:**
- `models/conversation/`: Conversation data structures (Type: message/line, DialogID, Self/Peer addresses)
- `models/message/`: Message data structures (Direction, Status, ReferenceType, TransactionID)
- `models/account/`: Account data structures (Type: sms/line, Secret, Token)
- `models/media/`: Media attachments for messages
- `pkg/conversationhandler/`: Business logic (Create/Get/Update conversations, handle webhooks, process events)
- `pkg/messagehandler/`: Business logic (Create/Send messages, manage status updates)
- `pkg/accounthandler/`: Manage messaging platform credentials
- `pkg/linehandler/`: LINE Bot SDK integration (send/receive messages, parse webhooks)
- `pkg/smshandler/`: SMS/MMS integration via message-manager RPC calls
- `pkg/dbhandler/`: Database operations using Squirrel SQL builder
- `pkg/cachehandler/`: Redis caching
- `pkg/listenhandler/`: RabbitMQ RPC request routing (REST-like paths)
- `pkg/subscribehandler/`: Event consumption from message-manager

### Request Routing

ListenHandler routes requests using regex patterns matching REST-like URIs:

**Accounts:**
- `POST /v1/accounts` - Create messaging platform account
- `GET /v1/accounts?<filters>` - List accounts (pagination via page_size/page_token)
- `GET /v1/accounts/<uuid>` - Get account
- `PUT /v1/accounts/<uuid>` - Update account credentials
- `DELETE /v1/accounts/<uuid>` - Delete account

**Conversations:**
- `POST /v1/conversations` - Create conversation
- `GET /v1/conversations?<filters>` - List conversations (pagination via page_size/page_token)
- `GET /v1/conversations/<uuid>` - Get conversation
- `PUT /v1/conversations/<uuid>` - Update conversation
- `POST /v1/conversations/<uuid>/messages` - Send message in conversation

**Messages:**
- `POST /v1/messages` - Create message record
- `GET /v1/messages?<filters>` - List messages (pagination via page_size/page_token)
- `GET /v1/messages/<uuid>` - Get message
- `PUT /v1/messages/<uuid>` - Update message
- `DELETE /v1/messages/<uuid>` - Delete message

**Webhooks (Hook Handler):**
- `POST /v1/hooks` - Generic webhook endpoint
- `POST /v1/setup` - Setup/configuration webhook

### Event Subscriptions

SubscribeHandler processes events from:
- **message-manager**: `message_created` - Creates conversation and message records when SMS/MMS received

### Webhook Processing

The service receives webhooks from external platforms:
- **LINE**: Incoming messages via LINE Bot SDK webhook (parsed in `pkg/linehandler/hook.go`)
  - Creates conversation if new chatroom
  - Creates incoming message record
  - Publishes conversation/message events
- Hook URI pattern: `/v1.0/conversation/accounts/<account_id>`

### Configuration

Uses **Viper + pflag** pattern (see `cmd/conversation-manager/init.go`):
- Command-line flags and environment variables (e.g., `--rabbitmq_address` or `RABBITMQ_ADDRESS`)
- Required: `database_dsn`, `rabbitmq_address`, `redis_address`, `redis_database`, `redis_password`, `prometheus_endpoint`, `prometheus_listen_address`

## Common Commands

### Build
```bash
# From monorepo root (expects parent directory context for replacements)
cd /path/to/monorepo/bin-conversation-manager
go build -o bin/conversation-manager ./cmd/conversation-manager
```

### Test
```bash
# Run all tests with coverage
go test -coverprofile cp.out -v $(go list ./...)
go tool cover -html=cp.out -o cp.html
go tool cover -func=cp.out

# Run tests for specific package
go test -v ./pkg/conversationhandler/...

# Run single test
go test -v ./pkg/conversationhandler -run Test_Create
```

### Generate Mocks
```bash
# Generate all mocks (uses go:generate directives)
go generate ./...

# Mocks are created via mockgen for interfaces in:
# - pkg/conversationhandler/main.go -> mock_conversationhandler.go
# - pkg/messagehandler/main.go -> mock_messagehandler.go
# - pkg/accounthandler/main.go -> mock_accounthandler.go
# - pkg/dbhandler/main.go -> mock_main.go
# - pkg/linehandler/main.go -> mock_linehandler.go
# - pkg/smshandler/main.go -> mock_smshandler.go
# - pkg/subscribehandler/main.go -> mock_subscribehandler.go
# - pkg/cachehandler/main.go -> mock_cachehandler.go
```

### Lint
```bash
# Run golangci-lint
golangci-lint run -v --timeout 5m

# Run vet
go vet $(go list ./...)
```

## conversation-control CLI Tool

A command-line tool for managing conversations directly via database/cache (bypasses RabbitMQ RPC). **All output is JSON format** (stdout), logs go to stderr.

```bash
# Conversation commands
./bin/conversation-control conversation get --id <uuid>
./bin/conversation-control conversation list --customer_id <uuid> [--limit 100] [--token] [--type]

# Account commands (messaging platform credentials)
./bin/conversation-control account create --customer_id <uuid> --type <line|sms> --secret <secret> --token <token> [--name] [--detail]
./bin/conversation-control account get --id <uuid>
./bin/conversation-control account list --customer_id <uuid> [--limit 100] [--token] [--type]
./bin/conversation-control account update --id <uuid> [--name] [--detail] [--secret] [--token]
./bin/conversation-control account delete --id <uuid>

# Message commands
./bin/conversation-control message get --id <uuid>
./bin/conversation-control message list --customer_id <uuid> [--limit 100] [--token] [--conversation_id] [--direction] [--status]
```

Uses same environment variables as conversation-manager (`DATABASE_DSN`, `RABBITMQ_ADDRESS`, `REDIS_ADDRESS`, etc.).

### Run Locally
```bash
# With environment variables
DATABASE_DSN="user:pass@tcp(127.0.0.1:3306)/voipbin" \
RABBITMQ_ADDRESS="amqp://guest:guest@localhost:5672" \
REDIS_ADDRESS="127.0.0.1:6379" \
REDIS_DATABASE=1 \
REDIS_PASSWORD="" \
PROMETHEUS_ENDPOINT="/metrics" \
PROMETHEUS_LISTEN_ADDRESS=":2112" \
./bin/conversation-manager

# Or with flags
./bin/conversation-manager \
  --database_dsn "user:pass@tcp(127.0.0.1:3306)/voipbin" \
  --rabbitmq_address "amqp://guest:guest@localhost:5672" \
  --redis_address "127.0.0.1:6379" \
  --redis_database 1
```

### Docker
```bash
# Build (expects monorepo root context)
docker build -f Dockerfile -t conversation-manager:latest ../..

# The Dockerfile builds from monorepo root:
# - Stage 1: go build in /app/bin-conversation-manager
# - Stage 2: Copy binary to alpine image
```

## Monorepo Context

This service depends on local monorepo packages (see `go.mod` replace directives):
- `monorepo/bin-common-handler`: Shared handlers (sockhandler, requesthandler, notifyhandler, databasehandler, utilhandler)
- `monorepo/bin-message-manager`: SMS/MMS message models and RPC calls
- `monorepo/bin-flow-manager`: Flow models (referenced in conversation variables)
- `monorepo/bin-hook-manager`: Hook models (webhook processing)
- `monorepo/bin-number-manager`: Number models (phone number management)

**Important**: Builds and Docker images assume parent monorepo directory context is available.

## Testing Patterns

Tests use **gomock** (go.uber.org/mock):
- Mock interfaces are generated in same package as interface definition (e.g., `pkg/dbhandler/mock_main.go`)
- Table-driven tests with struct slices defining test cases
- Context passed to all handler methods
- Setup test database using SQL scripts in `scripts/database_scripts_test/`

Example test structure:
```go
tests := []struct {
    name    string
    // inputs
    // expected results
}{
    {"normal", ...},
}
for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        mc := gomock.NewController(t)
        defer mc.Finish()
        // Setup mocks and test
    })
}
```

## Key Implementation Details

### Conversation Flow

1. **Incoming Message (LINE webhook)**:
   - Webhook received at hook endpoint
   - `conversationhandler.Hook()` parses account ID from URI
   - `linehandler.Hook()` processes LINE webhook payload
   - Creates/retrieves conversation by DialogID
   - Creates incoming message record
   - Publishes `conversation_created` and `message_created` events

2. **Outgoing Message (API request)**:
   - Client calls `POST /v1/conversations/<id>/messages`
   - `conversationhandler.MessageSend()` creates message record
   - `linehandler.Send()` or `smshandler.Send()` sends via platform
   - Updates message status to "done" or "failed"
   - Publishes `message_created` event

3. **Incoming SMS/MMS (event subscription)**:
   - message-manager publishes `message_created` event
   - `subscribehandler` processes event
   - `conversationhandler.Event()` creates conversation/message records
   - Publishes `conversation_created` and `message_created` events

### Database Schema

Tables (see `scripts/database_scripts_test/`):
- `conversation_accounts`: Messaging platform credentials (type, secret, token)
- `conversation_conversations`: Conversation threads (type, dialog_id, self, peer addresses)
- `conversation_messages`: Individual messages (direction, status, text, medias JSON)
- `conversation_medias`: Media attachments metadata

### Database Queries

Use Squirrel SQL builder (not raw SQL):
```go
import sq "github.com/Masterminds/squirrel"

sq.Select("*").From("conversation_conversations").
    Where(sq.Eq{"id": id}).
    Where(sq.Eq{"tm_delete": DefaultTimeStamp})
```

### Soft Deletes

Records use `tm_delete` timestamp. Default value `"9999-01-01 00:00:00.000000"` indicates active records.

### Cache Strategy

Redis cache is used for lookups by ID. Database is source of truth; cache updates on mutations.

### LINE Integration

LINE Bot SDK v7 is used (`github.com/line/line-bot-sdk-go/v7`):
- Client creation uses account secret/token
- Webhook signature verification required
- PushMessage API for outgoing messages
- Supports text messages (media support pending)

### SMS/MMS Integration

SMS/MMS handled via RPC calls to message-manager:
- `smshandler.Send()` makes RPC request to message-manager
- message-manager handles actual SMS/MMS delivery
- Webhook events from message-manager create conversation records

### Conversation Variables

Conversations can set flow variables for integration with flow-manager:
- `voipbin.conversation.self.*` - Self party info (name, detail, target, type)
- `voipbin.conversation.peer.*` - Peer party info (name, detail, target, type)
- `voipbin.conversation.id` - Conversation UUID
- `voipbin.conversation.owner_id` - Owner UUID
- `voipbin.conversation.message.text` - Message text content

## Database Setup

Test database schemas available in `scripts/database_scripts_test/`:
- `table_conversation_accounts.sql`
- `table_conversation_conversations.sql`
- `table_conversation_messages.sql`
- `table_conversation_medias.sql`

Use these for local development database setup.

## Prometheus Metrics

Service exposes metrics on configured endpoint (default `:2112/metrics`):
- `receive_request_process_time` - Histogram of RPC request processing time (labels: type, method)
- `receive_subscribe_event_process_time` - Histogram of event processing time (labels: publisher, type)
