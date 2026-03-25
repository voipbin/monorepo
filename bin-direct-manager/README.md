# bin-direct-manager

Manages direct hash lookups for SIP URI routing in the VoIPbin platform. Each direct record maps a unique hash to a customer resource (extension, conference, AI, AI team, or agent), enabling direct SIP URI dialing without phone numbers.

## Architecture

### Service Communication

- **Listen Handler** (`pkg/listenhandler/`): Consumes RPC requests from RabbitMQ queue `bin-manager.direct-manager.request`
- **Subscribe Handler** (`pkg/subscribehandler/`): Subscribes to events from other services (e.g., customer deletions) on queue `bin-manager.direct-manager.subscribe`
- **Notify Handler**: Publishes direct lifecycle events to exchange `bin-manager.direct-manager.event`

### Core Components

```
cmd/direct-manager/main.go
    ├── Database (MySQL)
    ├── Cache (Redis via pkg/cachehandler)
    └── run()
        ├── pkg/dbhandler (MySQL operations via Squirrel)
        ├── pkg/directhandler (Direct hash business logic)
        ├── runListen() -> pkg/listenhandler
        └── runSubscribe() -> pkg/subscribehandler
```

### Request Routing

ListenHandler routes requests using regex patterns matching REST-like URIs:

- `POST /v1/directs` - Create a new direct hash
- `GET /v1/directs?<filters>` - List directs (pagination via page_size/page_token)
- `GET /v1/directs/<uuid>` - Get direct by ID
- `GET /v1/directs/by-hash/<hash>` - Get direct by hash value
- `POST /v1/directs/<uuid>/regenerate` - Regenerate a direct hash
- `DELETE /v1/directs/<uuid>` - Delete a direct

### Event Subscriptions

SubscribeHandler subscribes to:
- `bin-manager.customer-manager.event`: Handles customer deletion events to clean up associated direct records

## Configuration

Uses **Cobra + Viper** for configuration management. Configuration is loaded via command-line flags or environment variables.

| Flag | Environment Variable | Default | Description |
|------|---------------------|---------|-------------|
| `--database_dsn` | `DATABASE_DSN` | `testid:testpassword@tcp(127.0.0.1:3306)/test` | MySQL connection DSN |
| `--rabbitmq_address` | `RABBITMQ_ADDRESS` | `amqp://guest:guest@localhost:5672` | RabbitMQ server address |
| `--redis_address` | `REDIS_ADDRESS` | `127.0.0.1:6379` | Redis server address |
| `--redis_password` | `REDIS_PASSWORD` | `""` | Redis password |
| `--redis_database` | `REDIS_DATABASE` | `1` | Redis database index |
| `--prometheus_endpoint` | `PROMETHEUS_ENDPOINT` | `/metrics` | Prometheus metrics endpoint path |
| `--prometheus_listen_address` | `PROMETHEUS_LISTEN_ADDRESS` | `:2112` | Prometheus listen address |

## Building

```bash
# Build binaries
go build -o bin/ ./cmd/...

# Docker build (from monorepo root)
docker build -f bin-direct-manager/Dockerfile -t direct-manager .
```

## Testing

```bash
# Run all tests
go test ./...

# Run specific package tests
go test -v ./pkg/directhandler/...

# Run single test
go test -v -run TestCreate ./pkg/directhandler/
```

## Running Locally

```bash
./bin/direct-manager \
  --database_dsn "user:pass@tcp(127.0.0.1:3306)/voipbin" \
  --rabbitmq_address "amqp://guest:guest@localhost:5672" \
  --redis_address "127.0.0.1:6379" \
  --redis_database 1
```

Or with environment variables:

```bash
DATABASE_DSN="user:pass@tcp(127.0.0.1:3306)/voipbin" \
RABBITMQ_ADDRESS="amqp://guest:guest@localhost:5672" \
REDIS_ADDRESS="127.0.0.1:6379" \
REDIS_DATABASE=1 \
./bin/direct-manager
```

## direct-control CLI

A command-line tool for managing direct records via database/cache (bypasses RabbitMQ RPC). All output is JSON format (stdout), logs go to stderr.

### Build

```bash
go build -o bin/direct-control ./cmd/direct-control
```

### Commands

```bash
# Create a direct hash
direct-control direct create \
  --customer-id <uuid> \
  --resource-type <type> \
  --resource-id <uuid>

# Get a direct by ID
direct-control direct get --id <uuid>

# List directs for a customer
direct-control direct list \
  --customer-id <uuid> \
  [--limit 100] \
  [--token <pagination-token>]

# Delete a direct
direct-control direct delete --id <uuid>

# Regenerate a direct hash
direct-control direct regenerate --id <uuid>
```

The CLI uses the same configuration flags and environment variables as `direct-manager`.

## Prometheus Metrics

- `direct_manager_receive_request_process_time` - Histogram of RPC request processing time (labels: type, method)
