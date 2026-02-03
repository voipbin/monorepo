# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

bin-contact-manager is a Go microservice for managing CRM-style contact records in a VoIP system. It provides CRUD operations for contacts with support for multiple phone numbers, emails, and tag assignments. It also supports contact lookup by phone number (E.164) or email for caller ID enrichment.

## Build and Test Commands

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

### Configuration

Environment variables / flags:
- `DATABASE_DSN` - MySQL connection string (default: `testid:testpassword@tcp(127.0.0.1:3306)/test`)
- `RABBITMQ_ADDRESS` - RabbitMQ connection (default: `amqp://guest:guest@localhost:5672`)
- `REDIS_ADDRESS` - Redis server address (default: `127.0.0.1:6379`)
- `REDIS_PASSWORD` - Redis password (default: empty)
- `REDIS_DATABASE` - Redis database index (default: `1`)
- `PROMETHEUS_ENDPOINT` - Metrics endpoint path (default: `/metrics`)
- `PROMETHEUS_LISTEN_ADDRESS` - Metrics server address (default: `:2112`)

### API Endpoints (via RabbitMQ RPC)

The service handles REST-like requests through RabbitMQ with URI pattern matching:

**Contacts:**
- `GET /v1/contacts?<params>` - List contacts with pagination and filters
- `POST /v1/contacts` - Create new contact (with optional phone numbers, emails, tags)
- `GET /v1/contacts/{contact-id}` - Get specific contact
- `PUT /v1/contacts/{contact-id}` - Update contact basic fields
- `DELETE /v1/contacts/{contact-id}` - Soft delete contact
- `GET /v1/contacts/lookup?customer_id=<uuid>&phone_e164=<e164>` - Lookup by phone
- `GET /v1/contacts/lookup?customer_id=<uuid>&email=<email>` - Lookup by email

**Phone Numbers:**
- `POST /v1/contacts/{contact-id}/phone-numbers` - Add phone number to contact
- `DELETE /v1/contacts/{contact-id}/phone-numbers/{phone-id}` - Remove phone number

**Emails:**
- `POST /v1/contacts/{contact-id}/emails` - Add email to contact
- `DELETE /v1/contacts/{contact-id}/emails/{email-id}` - Remove email

**Tags:**
- `POST /v1/contacts/{contact-id}/tags` - Add tag to contact
- `DELETE /v1/contacts/{contact-id}/tags/{tag-id}` - Remove tag from contact

### Event Types Published

- `contact_created` - When a new contact is created
- `contact_updated` - When contact information is updated
- `contact_deleted` - When a contact is deleted

### Database Tables

- `contact_manager_contact` - Main contact records
- `contact_manager_phone_number` - Phone numbers linked to contacts
- `contact_manager_email` - Email addresses linked to contacts
- `contact_manager_tag_assignment` - Tag assignments linking contacts to tags

### Cache Strategy

- Contacts are cached in Redis for fast lookups
- Cache is invalidated on updates and deletions
- Uses `pkg/cachehandler` for all Redis operations

### Integration with Other Services

- **bin-tag-manager**: Provides tag definitions used for contact tagging
- **bin-customer-manager**: Provides customer context; contacts are cleaned up when customers are deleted
- **commonaddress.Address**: Contacts can be mapped to addresses for call routing (mapping done by consuming services)
