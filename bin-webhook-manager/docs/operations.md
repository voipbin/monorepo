# bin-webhook-manager Operations

## Configuration

All flags support equivalent `UPPER_SNAKE_CASE` environment variables.

| Flag | Env | Description | Default |
|------|-----|-------------|---------|
| `rabbitmq_address` | `RABBITMQ_ADDRESS` | RabbitMQ connection URL | `amqp://guest:guest@localhost:5672` |
| `database_dsn` | `DATABASE_DSN` | MySQL DSN | `testid:testpassword@tcp(127.0.0.1:3306)/test` |
| `redis_address` | `REDIS_ADDRESS` | Redis host:port | required |
| `redis_password` | `REDIS_PASSWORD` | Redis auth | optional |
| `redis_database` | `REDIS_DATABASE` | Redis DB index | optional |
| `prometheus_endpoint` | `PROMETHEUS_ENDPOINT` | Metrics path | `/metrics` |
| `prometheus_listen_address` | `PROMETHEUS_LISTEN_ADDRESS` | Metrics listen address | `:2112` |

## Prometheus Metrics

Exposed at `PROMETHEUS_LISTEN_ADDRESS/PROMETHEUS_ENDPOINT` (default `:2112/metrics`).

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `receive_request_process_time` | Histogram | `type`, `method` | RPC request latency |
| `receive_subscribe_event_process_time` | Histogram | `publisher`, `type` | Event processing latency |

## CLI Tool: webhook-control

`cmd/webhook-control` — triggers webhook deliveries directly for testing.

```bash
# Send webhook using customer's saved config
./bin/webhook-control webhook send-to-customer \
  --customer_id <uuid> \
  --data '{"type":"event_type","data":{"key":"value"}}' \
  [--data_type application/json]

# Send webhook to a specific URI
./bin/webhook-control webhook send-to-uri \
  --customer_id <uuid> \
  --uri https://example.com/webhook \
  --data '{"type":"event_type","data":{"key":"value"}}' \
  [--method POST] \
  [--data_type application/json]
```

## Common Commands

```bash
# Build daemon
go build -o ./bin/webhook-manager ./cmd/webhook-manager

# Build CLI
go build -o ./bin/webhook-control ./cmd/webhook-control

# Test with coverage
go test -coverprofile cp.out -v $(go list ./...)
go tool cover -html=cp.out -o cp.html

# Regenerate mocks
go generate ./pkg/listenhandler/...
go generate ./pkg/webhookhandler/...
go generate ./pkg/dbhandler/...
go generate ./pkg/cachehandler/...
go generate ./pkg/accounthandler/...
go generate ./pkg/subscribehandler/...

# Full verification (mandatory before every commit)
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```
