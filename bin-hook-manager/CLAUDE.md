# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

The hook-manager is a webhook gateway service that receives external webhook messages and forwards them to internal VoIPBIN services via RabbitMQ. It acts as a public-facing HTTPS endpoint that routes incoming webhooks to appropriate internal microservices.

## Architecture

### Service Communication Pattern

The hook-manager follows a gateway pattern:
1. Receives external HTTPS webhook requests at public endpoints
2. Validates and processes the incoming data
3. Publishes messages to RabbitMQ queues for consumption by internal services
4. Uses `bin-common-handler/pkg/requesthandler` for RabbitMQ message publishing

The service operates as a thin proxy - it does NOT directly handle business logic but delegates to downstream services (email-manager, message-manager, conversation-manager) via message queues.

### Key Components

- **API Layer** (`api/`): HTTP endpoint handlers organized by version and resource
  - `api/v1.0/emails/` - Email webhook endpoint handler
  - `api/v1.0/messages/` - Message webhook endpoint handler
  - `api/v1.0/conversation/` - Conversation webhook endpoint handler

- **Service Handler** (`pkg/servicehandler/`): Abstraction layer that publishes webhook data to RabbitMQ
  - Interface-based design to enable mocking for tests
  - Uses `requesthandler.RequestHandler` from bin-common-handler to publish messages
  - Each handler method corresponds to a webhook type (Email, Message, Conversation)

- **Models** (`models/hook/`): Simple data structures for webhook payloads

### Configuration Management

The service uses **Viper + pflag** for configuration (see `cmd/hook-manager/init.go`):
- Environment variables take precedence over flags
- Flag format: `--database_dsn`, `--rabbitmq_address`, etc.
- Environment variable format: `DATABASE_DSN`, `RABBITMQ_ADDRESS`, etc.
- All configuration is initialized in `initVariable()` before main() runs

### Monorepo Dependencies

This service imports several sibling modules via `replace` directives in go.mod:
- `monorepo/bin-common-handler` - Shared utilities for database, RabbitMQ, and request handling
- Other bin-* services are indirect dependencies

When working with shared code, remember that changes to bin-common-handler affect multiple services.

## Development Commands

### Building
```bash
# Build the service
go build -o hook-manager ./cmd/hook-manager

# Docker build (runs from monorepo root)
# See Dockerfile - it expects to be run from parent directory
```

### Testing
```bash
# Run all tests
go test ./...

# Run specific package tests
go test ./api/v1.0/emails
go test ./pkg/servicehandler

# Run with verbose output
go test -v ./...

# Run specific test
go test -run TestEmailsPOST ./api/v1.0/emails
```

### Generating Mocks
```bash
# Regenerate mocks (requires go-mock)
go generate ./...

# Specific mock generation
cd pkg/servicehandler && go generate
```

## hook-control CLI Tool

A command-line tool for testing webhook functionality. **All output is JSON format** (stdout), logs go to stderr.

```bash
# Send a test email webhook
./bin/hook-control send-email --customer_id <uuid> --email_id <uuid>

# Send a test message webhook
./bin/hook-control send-message --customer_id <uuid> --message_id <uuid>

# Send a test conversation webhook
./bin/hook-control send-conversation --customer_id <uuid> --conversation_id <uuid>
```

Uses same environment variables as hook-manager (`DATABASE_DSN`, `RABBITMQ_ADDRESS`, `REDIS_ADDRESS`, etc.).

## Running the Service

### Required Configuration

- **DATABASE_DSN**: MySQL connection string (currently connected in main but not actively used)
- **RABBITMQ_ADDRESS**: RabbitMQ connection URL (e.g., `amqp://guest:guest@localhost:5672`)
- **SSL_CERT_BASE64**: Base64-encoded SSL certificate
- **SSL_PRIVKEY_BASE64**: Base64-encoded SSL private key

The service runs on both HTTP (:80) and HTTPS (:443) simultaneously. SSL certificates are decoded from base64 and written to `/tmp/` at startup.

### Webhook Endpoints

- `POST /v1.0/emails` → forwards to email-manager
- `POST /v1.0/messages` → forwards to message-manager
- `POST /v1.0/conversation` → forwards to conversation-manager
- `GET /ping` → health check endpoint

## Testing Patterns

Tests use gomock-generated mocks of ServiceHandler interface:
- Mock is generated from `pkg/servicehandler/main.go`
- Tests verify that incoming HTTP requests correctly call service handler methods
- Example pattern in `api/v1.0/emails/emails_test.go`

## Important Notes

- The service initializes SSL certificates by decoding base64 strings to files in `/tmp/` - ensure proper permissions in containerized environments
- Prometheus metrics are exposed on port 2112 at `/metrics` by default
- All endpoints use CORS middleware allowing all origins (`AllowOrigins: ["*"]`)
- The service uses structured logging with logrus + joonix formatter for Stackdriver compatibility
