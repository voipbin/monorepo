# bin-conversation-manager Operations

## Configuration

All flags support equivalent `UPPER_SNAKE_CASE` environment variables.

| Flag | Env | Description | Required |
|------|-----|-------------|----------|
| `rabbitmq_address` | `RABBITMQ_ADDRESS` | RabbitMQ connection URL | yes |
| `database_dsn` | `DATABASE_DSN` | MySQL DSN | yes |
| `redis_address` | `REDIS_ADDRESS` | Redis host:port | yes |
| `redis_password` | `REDIS_PASSWORD` | Redis auth | no |
| `redis_database` | `REDIS_DATABASE` | Redis DB index | no |
| `prometheus_endpoint` | `PROMETHEUS_ENDPOINT` | Metrics path | `/metrics` |
| `prometheus_listen_address` | `PROMETHEUS_LISTEN_ADDRESS` | Metrics listen address | `:2112` |

## Prometheus Metrics

Exposed at `PROMETHEUS_LISTEN_ADDRESS/PROMETHEUS_ENDPOINT` (default `:2112/metrics`).

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `account_create_total` | Counter | — | Messaging platform accounts created |
| `receive_request_process_time` | Histogram | `type`, `method` | RPC request latency |
| `receive_subscribe_event_process_time` | Histogram | `publisher`, `type` | Event processing latency |

## CLI Tool: conversation-control

`cmd/conversation-control` — direct DB/cache management (bypasses RabbitMQ). All output is JSON on stdout; logs go to stderr.

```bash
# Conversation commands
./bin/conversation-control conversation get  --id <uuid>
./bin/conversation-control conversation list --customer_id <uuid> [--limit 100] [--token] [--type]

# Account commands
./bin/conversation-control account create --customer_id <uuid> --type <line|sms> --secret <secret> --token <token> [--name] [--detail]
./bin/conversation-control account get    --id <uuid>
./bin/conversation-control account list   --customer_id <uuid> [--limit 100] [--token] [--type]
./bin/conversation-control account update --id <uuid> [--name] [--detail] [--secret] [--token]
./bin/conversation-control account delete --id <uuid>

# Message commands
./bin/conversation-control message get  --id <uuid>
./bin/conversation-control message list --customer_id <uuid> [--limit 100] [--token] [--conversation_id] [--direction] [--status]
```

## Common Commands

```bash
# Build
go build -o bin/conversation-manager ./cmd/conversation-manager

# Test with coverage
go test -coverprofile cp.out -v $(go list ./...)
go tool cover -html=cp.out -o cp.html

# Regenerate mocks
go generate ./...

# Full verification (mandatory before every commit)
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

# Docker build (from monorepo root)
docker build -f bin-conversation-manager/Dockerfile -t conversation-manager:latest .
```

## Database Setup (local development)

Test schemas in `scripts/database_scripts_test/`:

```bash
mysql -u root -p voipbin < scripts/database_scripts_test/table_conversation_accounts.sql
mysql -u root -p voipbin < scripts/database_scripts_test/table_conversation_conversations.sql
mysql -u root -p voipbin < scripts/database_scripts_test/table_conversation_messages.sql
mysql -u root -p voipbin < scripts/database_scripts_test/table_conversation_medias.sql
```
