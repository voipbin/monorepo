# bin-tag-manager Operations

## Common Failure Modes

| Symptom | Likely Cause | Resolution |
|---------|-------------|-----------|
| Tag create returns conflict | Duplicate tag name for same customer | Use a different name; names must be unique per customer |
| Cascading delete incomplete | `subscribehandler` missed `customer_deleted` event | Check RabbitMQ dead-letter queue; re-publish event |
| Stale tag data after update | Redis cache not invalidated | Restart pod; cache has TTL and will expire |
| Tag list returns 0 items | Soft-delete sentinel query not filtering correctly | Check `tm_delete` column value in DB |
| RabbitMQ connection refused | `RABBITMQ_ADDRESS` misconfigured | Verify secret and network policy |

## Debugging Guide

```bash
# Pod logs
kubectl logs -n voipbin -l app=tag-manager --tail=100

# Admin CLI — list tags for a customer
./bin/tag-control tag list --customer_id <uuid> --limit 50

# Get a specific tag
./bin/tag-control tag get --id <uuid>

# Create a tag directly
./bin/tag-control tag create --customer_id <uuid> --name "my-tag"

# Delete a tag
./bin/tag-control tag delete --id <uuid>

# Build
cd bin-tag-manager && go build -o ./bin/ ./cmd/...

# Run tests
go test ./...

# Run tests with coverage
go test -coverprofile cp.out -v $(go list ./...)
go tool cover -html=cp.out -o cp.html

# Lint
golangci-lint run -v --timeout 5m

# Generate mocks
go generate ./...
```

## Configuration

| Flag / Env | Description | Default |
|-----------|-------------|---------|
| `DATABASE_DSN` | MySQL connection string | `testid:testpassword@tcp(127.0.0.1:3306)/test` |
| `RABBITMQ_ADDRESS` | RabbitMQ server | `amqp://guest:guest@localhost:5672` |
| `REDIS_ADDRESS` | Redis server | `127.0.0.1:6379` |
| `REDIS_PASSWORD` | Redis auth | empty |
| `REDIS_DATABASE` | Redis DB index | `1` |
| `PROMETHEUS_ENDPOINT` | Metrics path | `/metrics` |
| `PROMETHEUS_LISTEN_ADDRESS` | Metrics listen address | `:2112` |

## Prometheus Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `tag_manager_receive_request_process_time` | Histogram | `type`, `method` | RPC request processing duration |
| `tag_manager_subscribe_event_process_time` | Histogram | `publisher`, `type` | Event processing duration |
