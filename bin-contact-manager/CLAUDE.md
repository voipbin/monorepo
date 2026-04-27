# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

`bin-contact-manager` is a Go microservice for managing CRM-style contact records in a VoIP system. It provides CRUD operations for contacts with support for multiple phone numbers, emails, and tag assignments. It also supports contact lookup by phone number (E.164) or email for caller ID enrichment.

**Key Concepts:**
- **Contact**: A CRM record (first/last name, display name, company, job title, source, external_id, notes) belonging to a customer.
- **PhoneNumber / Email**: 1:N child records on a contact for multi-number / multi-email support.
- **Tag assignment**: Many-to-many link between contacts and tags managed in `bin-tag-manager`.
- **Lookup**: O(1) lookup by E.164 phone or email for caller-ID enrichment in inbound flows.

> Cross-cutting rules (verification workflow, branch/commit format, worktree usage, Alembic, RST sync) live in the root [CLAUDE.md](../CLAUDE.md). This file documents only what is specific to `bin-contact-manager`.

## Common Commands

```bash
# Build the service and CLI
go build -o ./bin/ ./cmd/...

# Run the daemon (requires configuration via flags or env vars)
./bin/contact-manager

# Run the CLI tool
./bin/contact-control

# Run all tests
go test ./...

# Run tests with coverage
go test -coverprofile cp.out -v $(go list ./...)
go tool cover -html=cp.out -o cp.html

# Run a single test
go test -v -run TestName ./pkg/packagename/...

# Lint
golint -set_exit_status $(go list ./...)
golangci-lint run -v --timeout 5m

# Vet
go vet $(go list ./...)

# Generate all mocks (uses go.uber.org/mock via //go:generate directives)
go generate ./pkg/listenhandler/...
go generate ./pkg/subscribehandler/...
go generate ./pkg/contacthandler/...
go generate ./pkg/dbhandler/...
go generate ./pkg/cachehandler/...
```

## contact-control CLI Tool

A command-line tool for managing contacts directly via database/cache (bypasses RabbitMQ RPC). **All output is JSON format** (stdout), logs go to stderr.

```bash
# Create contact - returns created contact JSON
./bin/contact-control contact create --customer-id <uuid> --first-name <name> [--last-name <name>] [--display-name <name>] [--company <company>] [--job-title <title>] [--source <source>] [--external-id <id>] [--notes <notes>]

# Get contact - returns contact JSON
./bin/contact-control contact get --id <uuid>

# List contacts - returns JSON array
./bin/contact-control contact list --customer-id <uuid> [--limit 100] [--token]

# Update contact - returns updated contact JSON
./bin/contact-control contact update --id <uuid> [--first-name <name>] [--last-name <name>] [--display-name <name>] [--company <company>] [--job-title <title>] [--external-id <id>] [--notes <notes>]

# Delete contact - returns deleted contact JSON
./bin/contact-control contact delete --id <uuid>

# Lookup contact - returns contact JSON
./bin/contact-control contact lookup --customer-id <uuid> --phone-e164 <+1234567890>
./bin/contact-control contact lookup --customer-id <uuid> --email <email@example.com>
```

Uses same environment variables as contact-manager (`DATABASE_DSN`, `RABBITMQ_ADDRESS`, `REDIS_ADDRESS`, etc.).

## Architecture

### Service Layer Structure

The service follows a layered architecture with handler separation:

1. **cmd/contact-manager/** - Main daemon entry point with configuration via pflag/Viper (flags and env vars)
2. **pkg/listenhandler/** - RabbitMQ RPC request handler with regex-based URI routing for REST-like API operations
3. **pkg/subscribehandler/** - Event subscriber handling cascading deletions from customer-manager
4. **pkg/contacthandler/** - Core business logic for contact CRUD operations and event publishing
5. **pkg/dbhandler/** - Database operations with MySQL and Redis cache coordination
6. **pkg/cachehandler/** - Redis cache operations for contact lookups
7. **models/contact/** - Data structures (Contact, PhoneNumber, Email, event types, webhook)

### Inter-Service Communication

- Uses RabbitMQ for message passing between microservices
- Listens on `QueueNameContactRequest` for RPC requests
- Publishes events to `QueueNameContactEvent` when contacts change (created, updated, deleted)
- Subscribes to `QueueNameCustomerEvent` for cascading deletions when customers are removed
- **Monorepo structure**: All sibling services are referenced via `replace` directives in go.mod pointing to `../bin-*-manager` directories. When modifying shared dependencies, changes affect all services immediately.

### Key Patterns

- Handler interfaces with mock generation using `go.uber.org/mock` (`//go:generate mockgen`)
- Table-driven tests with `testing` package
- Prometheus metrics exposed at configurable endpoint (default `:2112/metrics`)
- Context propagation through all handler methods
- UUID-based entity identification using `github.com/gofrs/uuid`
- Soft delete pattern with `tm_delete` timestamp

### Request Flow

```
RabbitMQ Request → listenhandler (regex routing) → contacthandler → dbhandler → MySQL/Redis
                                                                 ↓
                                                             notifyhandler → RabbitMQ event publish

Event Flow:
RabbitMQ Event → subscribehandler → contacthandler → cleanup operations
```

## Request Routing

The service handles REST-like requests through RabbitMQ with URI pattern matching:

**Contacts API (`/v1/contacts/*`):**
- `GET /v1/contacts?<params>` — List contacts with pagination and filters
- `POST /v1/contacts` — Create new contact (with optional phone numbers, emails, tags)
- `GET /v1/contacts/{contact-id}` — Get specific contact
- `PUT /v1/contacts/{contact-id}` — Update contact basic fields
- `DELETE /v1/contacts/{contact-id}` — Soft delete contact
- `GET /v1/contacts/lookup?customer_id=<uuid>&phone_e164=<e164>` — Lookup by phone
- `GET /v1/contacts/lookup?customer_id=<uuid>&email=<email>` — Lookup by email

**Phone Numbers API:**
- `POST /v1/contacts/{contact-id}/phone-numbers` — Add phone number to contact
- `DELETE /v1/contacts/{contact-id}/phone-numbers/{phone-id}` — Remove phone number

**Emails API:**
- `POST /v1/contacts/{contact-id}/emails` — Add email to contact
- `DELETE /v1/contacts/{contact-id}/emails/{email-id}` — Remove email

**Tags API:**
- `POST /v1/contacts/{contact-id}/tags` — Add tag to contact
- `DELETE /v1/contacts/{contact-id}/tags/{tag-id}` — Remove tag from contact

**Events Published:**
- `contact_created` — when a new contact is created
- `contact_updated` — when contact information is updated
- `contact_deleted` — when a contact is deleted

## Event Subscriptions

SubscribeHandler subscribes to:
- **bin-manager.customer-manager.event**: `customer_deleted` → cascading deletion of contacts owned by the deleted customer.

## Monorepo Context

This service depends on local monorepo packages (see `go.mod` replace directives):
- `monorepo/bin-common-handler`: Shared utilities (sockhandler, requesthandler, notifyhandler)
- `monorepo/bin-customer-manager`: Customer event models (cascading deletes)
- `monorepo/bin-tag-manager`: Tag definitions used for contact tagging

Contacts can be mapped to `commonaddress.Address` for call routing — mapping is done by consuming services.

Always run `go mod vendor` after changing dependencies.

## Testing Patterns

Tests use **gomock** (go.uber.org/mock):
- Mock interfaces co-located with handlers (`mock_*.go`)
- Table-driven tests with struct slices

```go
tests := []struct {
    name      string
    input     InputType
    mockSetup func(*MockHandler)
    expectRes ResultType
    expectErr bool
}{
    {"success case", input1, setupMock1, expected1, false},
    {"error case", input2, setupMock2, nil, true},
}
for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        mc := gomock.NewController(t)
        defer mc.Finish()
        // test implementation
    })
}
```

## Key Implementation Details

### Cache Strategy
Contacts are cached in Redis for fast lookups; cache is invalidated on updates and deletions. Uses `pkg/cachehandler` for all Redis operations.

### Database Tables
- `contact_manager_contact` — Main contact records
- `contact_manager_phone_number` — Phone numbers linked to contacts
- `contact_manager_email` — Email addresses linked to contacts
- `contact_manager_tag_assignment` — Tag assignments linking contacts to tags

### Soft Deletes
Records use `tm_delete` timestamp (`"9999-01-01 00:00:00.000000"` for active records).

## Configuration

Environment variables / flags:

| Flag / Env | Description | Default |
|------------|-------------|---------|
| `database_dsn` / `DATABASE_DSN` | MySQL connection string | `testid:testpassword@tcp(127.0.0.1:3306)/test` |
| `rabbitmq_address` / `RABBITMQ_ADDRESS` | RabbitMQ server | `amqp://guest:guest@localhost:5672` |
| `redis_address` / `REDIS_ADDRESS` | Redis server | `127.0.0.1:6379` |
| `redis_password` / `REDIS_PASSWORD` | Redis password | empty |
| `redis_database` / `REDIS_DATABASE` | Redis DB index | `1` |
| `prometheus_endpoint` / `PROMETHEUS_ENDPOINT` | Metrics path | `/metrics` |
| `prometheus_listen_address` / `PROMETHEUS_LISTEN_ADDRESS` | Metrics port | `:2112` |

## Prometheus Metrics

Service exposes metrics on the configured endpoint (default `:2112/metrics`):
- `contact_manager_receive_request_process_time` — histogram of RPC request processing time (labels: type, method)
- `contact_manager_subscribe_event_process_time` — histogram of event processing time (labels: publisher, type)
