# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

`bin-webhook-manager` is a Go microservice for managing webhook event notifications in a VoIP system. It receives webhook requests from other services and publishes webhook events for delivery to customer-defined endpoints.

**Key Concepts:**
- **Webhook**: An outbound HTTP notification dispatched to a customer-configured URI when an event of interest occurs.
- **Customer webhook config**: Each customer has a `webhook_method` and `webhook_uri` set in `bin-customer-manager` — this service reads that config to know where to send.
- **Two delivery modes**: `SendWebhookToCustomer` uses the customer's saved config; `SendWebhookToURI` lets callers override the destination.

> Cross-cutting rules (verification workflow, branch/commit format, worktree usage, Alembic, RST sync) live in the root [CLAUDE.md](../CLAUDE.md). This file documents only what is specific to `bin-webhook-manager`.

## Common Commands

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

## Request Routing

ListenHandler routes RPC requests using regex patterns matching REST-like URIs:

**Webhooks API (`/v1/webhooks/*`):**
- `POST /v1/webhooks/send-to-customer` — Look up the customer's saved webhook config and dispatch
- `POST /v1/webhooks/send-to-uri` — Dispatch to a caller-specified URI/method

## Event Subscriptions

SubscribeHandler subscribes to:
- **bin-manager.customer-manager.event**: `customer_updated`, `customer_deleted` — invalidates the local accounthandler cache so subsequent webhook dispatches see the new URI/method.

## Monorepo Context

This service depends on local monorepo packages (see `go.mod` replace directives):
- `monorepo/bin-common-handler`: Shared handlers (sockhandler, requesthandler, notifyhandler, databasehandler)
- `monorepo/bin-customer-manager`: Customer event/account models for resolving webhook destinations

Always run `go mod vendor` after changing dependencies.

## Testing Patterns

Tests use **gomock** (go.uber.org/mock):
- Mock interfaces co-located with the handler (`mock_*.go`)
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

### Account Cache Invalidation
Customer webhook configs are cached in Redis via `pkg/accounthandler/`. The cache must be invalidated when `bin-customer-manager` publishes `customer_updated` or `customer_deleted` events, otherwise webhook dispatches will continue using the old URI/method.

### Event Publishing
Both delivery paths emit a `webhook_published` event to `bin-manager.webhook-manager.event` after queuing the dispatch. Downstream consumers can correlate these to the originating customer/event for analytics or retry coordination.

## Configuration

Configuration is handled via `spf13/viper` and `spf13/pflag`, supporting both command-line flags and environment variables with automatic env binding.

| Flag / Env | Description | Default |
|------------|-------------|---------|
| `database_dsn` / `DATABASE_DSN` | MySQL connection string | `testid:testpassword@tcp(127.0.0.1:3306)/test` |
| `rabbitmq_address` / `RABBITMQ_ADDRESS` | RabbitMQ server | `amqp://guest:guest@localhost:5672` |
| `redis_address` / `REDIS_ADDRESS` | Redis cache | required |
| `redis_password` / `REDIS_PASSWORD` | Redis auth | optional |
| `redis_database` / `REDIS_DATABASE` | Redis DB index | optional |
| `prometheus_endpoint` / `PROMETHEUS_ENDPOINT` | Metrics path | `/metrics` |
| `prometheus_listen_address` / `PROMETHEUS_LISTEN_ADDRESS` | Metrics port | `:2112` |

## Prometheus Metrics

Service exposes metrics on the configured endpoint (default `:2112/metrics`):
- `webhook_manager_receive_request_process_time` — histogram of RPC request processing time (labels: type, method)
- `webhook_manager_subscribe_event_process_time` — histogram of event processing time (labels: publisher, type)
