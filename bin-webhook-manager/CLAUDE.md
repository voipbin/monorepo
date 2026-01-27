# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

bin-webhook-manager is a Go microservice for managing webhook event notifications in a VoIP system. It receives webhook requests from other services and publishes webhook events for delivery to customer-defined endpoints.

## Build and Test Commands

```bash
# Build the webhook-manager daemon
go build -o ./bin/webhook-manager ./cmd/webhook-manager

# Build the webhook-control CLI tool
go build -o ./bin/webhook-control ./cmd/webhook-control

# Run the daemon (requires configuration via flags or env vars)
./bin/webhook-manager

# Run all tests
go test ./...

# Run tests with coverage
go test -coverprofile cp.out -v $(go list ./...)
go tool cover -html=cp.out -o cp.html

# Run a single test
go test -v -run TestName ./pkg/packagename/...

# Generate all mocks (uses go.uber.org/mock via //go:generate directives)
go generate ./pkg/listenhandler/...
go generate ./pkg/webhookhandler/...
go generate ./pkg/dbhandler/...
go generate ./pkg/cachehandler/...
go generate ./pkg/accounthandler/...
go generate ./pkg/subscribehandler/...
```

### webhook-control CLI Tool

A command-line tool for triggering webhook operations. **All output is JSON format** (stdout), logs go to stderr.

```bash
# Send webhook to customer using their configured webhook settings
./bin/webhook-control webhook send-to-customer \
  --customer_id <uuid> \
  --data '{"type":"event_type","data":{"key":"value"}}' \
  [--data_type application/json]

# Send webhook to a specific URI with specified method
./bin/webhook-control webhook send-to-uri \
  --customer_id <uuid> \
  --uri https://example.com/webhook \
  --data '{"type":"event_type","data":{"key":"value"}}' \
  [--method POST] \
  [--data_type application/json]
```

Uses same environment variables as webhook-manager (`DATABASE_DSN`, `RABBITMQ_ADDRESS`, `REDIS_ADDRESS`, etc.).

## Architecture

### Service Layer Structure

The service follows a layered architecture with handler separation:

1. **cmd/webhook-manager/** - Main daemon entry point with configuration via Cobra/Viper (flags and env vars)
2. **pkg/listenhandler/** - RabbitMQ RPC request handler with regex-based URI routing for REST-like webhook API operations
3. **pkg/webhookhandler/** - Core business logic for webhook processing and event publishing
4. **pkg/subscribehandler/** - Event subscriber handling customer-related events from other services
5. **pkg/accounthandler/** - Account information retrieval and webhook configuration management
6. **pkg/dbhandler/** - Database operations with MySQL and Redis cache coordination
7. **pkg/cachehandler/** - Redis cache operations for account lookups
8. **models/** - Data structures (webhook, account, event)

### Inter-Service Communication

- Uses RabbitMQ for message passing between microservices
- Listens on `QueueNameWebhookRequest` for webhook delivery requests
- Publishes events to `QueueNameWebhookEvent` when webhooks are published
- Subscribes to customer-manager events for customer updates
- **Monorepo structure**: All sibling services are referenced via `replace` directives in go.mod pointing to `../bin-*-manager` directories. When modifying shared dependencies, changes affect all services immediately.

### Key Patterns

- Handler interfaces with mock generation using `go.uber.org/mock` (`//go:generate mockgen`)
- Table-driven tests with `testing` package
- Prometheus metrics exposed at configurable endpoint (process time histograms)
- Context propagation through all handler methods
- UUID-based entity identification using `github.com/gofrs/uuid`

### Request Flow

```
RabbitMQ Request → listenhandler (regex routing) → webhookhandler → accounthandler
                                                                  ↓
                                                    notifyhandler (publish event)
                                                                  ↓
                                                              dbhandler → MySQL/Redis
```

### Webhook Delivery Methods

The service supports two webhook delivery patterns:

1. **SendWebhookToCustomer** - Retrieves webhook configuration from customer account settings (URI and method) and publishes webhook event
2. **SendWebhookToURI** - Directly sends webhook to specified URI with provided method and data type

Both methods publish `webhook_published` events to the event queue for asynchronous processing.

### Configuration

Environment variables / flags (all have defaults):
- `DATABASE_DSN` - MySQL connection string (default: `testid:testpassword@tcp(127.0.0.1:3306)/test`)
- `RABBITMQ_ADDRESS` - RabbitMQ connection (default: `amqp://guest:guest@localhost:5672`)
- `REDIS_ADDRESS`, `REDIS_PASSWORD`, `REDIS_DATABASE` - Redis cache configuration
- `PROMETHEUS_ENDPOINT`, `PROMETHEUS_LISTEN_ADDRESS` - Metrics endpoint (default: `/metrics` on `:2112`)

Configuration is handled via `spf13/viper` and `spf13/pflag`, supporting both command-line flags and environment variables with automatic env binding.
