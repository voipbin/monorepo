# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

This is the `bin-email-manager` service, part of the VoIPBin monorepo. It manages email delivery within the VoIPBin platform, integrating with various email providers (SendGrid, Mailgun) to send emails and processing webhook events for delivery status updates.

**Key Concepts:**
- **Email**: Outgoing email with status tracking, provider info, attachments, and delivery events
- **Provider Fallback**: Automatic failover between email providers (SendGrid → Mailgun) if one fails
- **Webhook Integration**: Processes delivery status callbacks from providers to update email states
- **Attachment Support**: Links to storage-manager for including recordings/files in emails

## Architecture

### Service Communication Pattern

This service uses **RabbitMQ for RPC-style communication**:
- **ListenHandler** (`pkg/listenhandler/`): Consumes RPC requests from queue `QueueNameEmailRequest`, processes them, and returns responses
- **EmailHandler** (`pkg/emailhandler/`): Core business logic for creating emails, sending via providers, and handling webhooks
- **NotifyHandler**: Publishes webhook events when email status changes

### Core Components

```
cmd/email-manager/main.go
    ├── createDBHandler()
        ├── pkg/dbhandler (MySQL via Squirrel query builder)
        └── pkg/cachehandler (Redis)
    └── run()
        ├── sockhandler (RabbitMQ connection)
        ├── requesthandler (RPC requests to other services)
        ├── notifyhandler (Webhook event publishing)
        ├── pkg/emailhandler (Business logic layer)
        └── runListen() -> pkg/listenhandler
```

**Layer Responsibilities:**
- `models/email/`: Core data structures (Email, Status, ProviderType, Attachment, WebhookMessage)
- `models/sendgrid/`: SendGrid-specific webhook event models
- `pkg/emailhandler/`: Business logic (Create/Send emails, webhook processing, provider engines)
- `pkg/dbhandler/`: Database operations using Squirrel SQL builder
- `pkg/cachehandler/`: Redis caching
- `pkg/listenhandler/`: RabbitMQ RPC request routing (REST-like paths: `/v1/emails`, `/v1/hooks`)

### Request Routing

ListenHandler routes requests using regex patterns matching REST-like URIs:
- `POST /v1/emails` - Create and send email
- `GET /v1/emails?<filters>` - List emails (pagination via page_size/page_token)
- `GET /v1/emails/<uuid>` - Get email details
- `DELETE /v1/emails/<uuid>` - Delete email
- `POST /v1/hooks` - Process webhook events from providers (query param: `?uri=/v1/hooks/sendgrid` or `?uri=/v1/hooks/mailgun`)

### Provider Integration

**Multi-Provider Strategy:**
- Emails attempt delivery via SendGrid first, then Mailgun if SendGrid fails
- See `pkg/emailhandler/send.go` - iterates through providers until one succeeds
- Each provider engine (`engine_sendgrid.go`, `engine_mailgun.go`) implements the `Send()` method
- Provider reference ID stored in database for tracking

**Webhook Processing:**
- Providers send delivery status events to: `https://hook.voipbin.net/v1.0/emails/sendgrid` or `/mailgun`
- Events update email status: `initiated` → `processed` → `delivered` → `open` → `click`
- Also handles: `bounce`, `dropped`, `deferred`, `unsubscribe`, `spamreport`
- See `pkg/emailhandler/hook.go` for webhook routing logic

### Email Attachments

Attachments reference files via `AttachmentReferenceType`:
- `recording`: Links to call recordings via `storage-manager` service
- Uses `requesthandler` to fetch file URLs from storage service
- Attachments fetched and attached before sending via provider

### Configuration

Uses **pflag + Viper** pattern (see `cmd/email-manager/init.go`):
- Command-line flags and environment variables (e.g., `--sendgrid_api_key` or `SENDGRID_API_KEY`)
- Required: `database_dsn`, `rabbitmq_address`, `redis_address`, `prometheus_endpoint`
- Provider-specific: `sendgrid_api_key`, `mailgun_api_key`

## Common Commands

### Build
```bash
# From monorepo root (expects parent directory context for replacements)
cd /path/to/monorepo/bin-email-manager
go build -o bin/email-manager ./cmd/email-manager
```

### Test
```bash
# Run all tests with coverage
go test -coverprofile cp.out -v $(go list ./...)
go tool cover -html=cp.out -o cp.html
go tool cover -func=cp.out

# Run tests for specific package
go test -v ./pkg/emailhandler/...

# Run single test
go test -v ./pkg/emailhandler -run Test_Send
```

### Generate Mocks
```bash
# Generate all mocks (uses go:generate directives)
go generate ./...

# Mocks are created via mockgen for interfaces in:
# - pkg/emailhandler/main.go -> mock_main.go
# - pkg/emailhandler/engine_sendgrid.go -> mock_engine_sendgrid.go
# - pkg/emailhandler/engine_mailgun.go -> mock_engine_mailgun.go
# - pkg/dbhandler/main.go -> mock_main.go
# - pkg/listenhandler/main.go -> mock_main.go
# - pkg/cachehandler/main.go -> mock_main.go
```

### Lint
```bash
# Run golangci-lint
golangci-lint run -v --timeout 5m

# Run vet
go vet $(go list ./...)
```

## email-control CLI Tool

A command-line tool for managing emails directly via database/cache (bypasses RabbitMQ RPC). **All output is JSON format** (stdout), logs go to stderr.

```bash
# Get email - returns email JSON
./bin/email-control email get --id <uuid>

# List emails - returns JSON array
./bin/email-control email list --customer_id <uuid> [--limit 100] [--token]

# Delete email - returns deleted email JSON
./bin/email-control email delete --id <uuid>
```

Uses same environment variables as email-manager (`DATABASE_DSN`, `RABBITMQ_ADDRESS`, `REDIS_ADDRESS`, etc.).

### Run Locally
```bash
# With environment variables
DATABASE_DSN="user:pass@tcp(127.0.0.1:3306)/voipbin" \
RABBITMQ_ADDRESS="amqp://guest:guest@localhost:5672" \
REDIS_ADDRESS="127.0.0.1:6379" \
REDIS_DATABASE=1 \
SENDGRID_API_KEY="SG.xxx" \
MAILGUN_API_KEY="xxx" \
PROMETHEUS_ENDPOINT="/metrics" \
PROMETHEUS_LISTEN_ADDRESS=":2112" \
./bin/email-manager

# Or with flags
./bin/email-manager \
  --database_dsn "user:pass@tcp(127.0.0.1:3306)/voipbin" \
  --rabbitmq_address "amqp://guest:guest@localhost:5672" \
  --redis_address "127.0.0.1:6379" \
  --sendgrid_api_key "SG.xxx" \
  --mailgun_api_key "xxx"
```

### Docker
```bash
# Build (expects monorepo root context)
docker build -f Dockerfile -t email-manager:latest ../..
```

## Monorepo Context

This service depends on local monorepo packages (see `go.mod` replace directives):
- `monorepo/bin-common-handler`: Shared handlers (sockhandler, requesthandler, notifyhandler, databasehandler)
- `monorepo/bin-storage-manager`: Models for bucketfile attachments

**Important**: Builds and Docker images assume parent monorepo directory context is available.

## Testing Patterns

Tests use **gomock** (go.uber.org/mock):
- Mock interfaces are generated in same package as interface definition (e.g., `pkg/dbhandler/mock_main.go`)
- Table-driven tests with struct slices defining test cases
- Context passed to all handler methods

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

### Email Status Flow
```
initiated -> processed -> delivered -> open/click
                      \-> bounce/dropped/deferred/unsubscribe/spamreport
```

### Provider Failover Logic
Located in `pkg/emailhandler/send.go`:
```go
// Try SendGrid first, fallback to Mailgun
providers := []struct {
    name    string
    handler func(context.Context, *email.Email) (string, error)
}{
    {"sendgrid", h.engineSendgrid.Send},
    {"mailgun", h.engineMailgun.Send},
}
```

### Database Queries
Use Squirrel SQL builder (not raw SQL):
```go
import sq "github.com/Masterminds/squirrel"

sq.Select("*").From("email_emails").
    Where(sq.Eq{"id": id}).
    Where(sq.Eq{"tm_delete": DefaultTimeStamp})
```

### Soft Deletes
Records use `tm_delete` timestamp. Default value `"9999-01-01 00:00:00.000000"` indicates active records.

### Webhook Event Publishing
After status updates, service publishes webhook events for external integrations:
```go
webhookEvent, err := email.CreateWebhookEvent()
notifyHandler.Publish(webhookEvent)
```

## Webhook Configuration

External services should configure webhooks to:
- **SendGrid**: `https://hook.voipbin.net/v1.0/emails/sendgrid`
- **Mailgun**: `https://hook.voipbin.net/v1.0/emails/mailgun`

These endpoints are handled by the hook service, which forwards to this manager via RabbitMQ.

## Database Schema

See `scripts/database_scripts/table_email_emails.sql` for schema definition.

Key tables:
- `email_emails`: Stores email records with provider info, status, attachments

## Prometheus Metrics

Service exposes metrics on configured endpoint (default `:2112/metrics`):
- `receive_request_process_time` - Histogram of RPC request processing time (labels: type, method)
