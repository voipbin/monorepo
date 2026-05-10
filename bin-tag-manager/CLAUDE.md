# bin-tag-manager

Lightweight CRUD service for managing customer-scoped tags. Tags are labels that other services (contacts, queues, campaigns) attach to resources for categorization. Handles event-driven cascading deletes when customers are removed.

> Cross-cutting rules (verification workflow, branch/commit format, worktrees, Alembic, RST sync) live in the root [CLAUDE.md](../CLAUDE.md). This file covers only what is specific to `bin-tag-manager`.

## Key facts

- **MySQL + Redis** for records and cache. Soft-deletes via `tm_delete` sentinel.
- **RabbitMQ queue**: `bin-manager.tag-manager.request`
- **Subscribes to**: `bin-manager.customer-manager.event` (`customer_deleted` → bulk tag delete)
- **Publishes**: `tag_created`, `tag_updated`, `tag_deleted` on `bin-manager.tag-manager.event`

## Package layout

| Package | Role |
|---------|------|
| `cmd/tag-manager` | Daemon entry point |
| `cmd/tag-control` | Admin CLI (JSON output, bypasses RabbitMQ) |
| `pkg/listenhandler` | RabbitMQ RPC router (regex dispatch) |
| `pkg/subscribehandler` | Event consumer (customer_deleted) |
| `pkg/taghandler` | Core business logic and event publishing |
| `pkg/dbhandler` | MySQL reads/writes with soft-delete |
| `pkg/cachehandler` | Redis cache |
| `models/tag` | Tag struct, events, WebhookMessage |

## Request routing

| Pattern | Operations |
|---------|-----------|
| `/v1/tags$` | POST (create) |
| `/v1/tags?(.*)$` | GET (list) |
| `/v1/tags/<uuid>$` | GET, PUT, DELETE |

## tag-control CLI

```bash
./bin/tag-control tag create --customer_id <uuid> --name "my-tag"
./bin/tag-control tag list --customer_id <uuid>
./bin/tag-control tag get --id <uuid>
./bin/tag-control tag delete --id <uuid>
```

## Common commands

```bash
go build -o ./bin/ ./cmd/...
go test ./...
go generate ./...
golangci-lint run -v --timeout 5m
```

## Configuration

| Env | Description | Default |
|-----|-------------|---------|
| `DATABASE_DSN` | MySQL DSN | required |
| `RABBITMQ_ADDRESS` | RabbitMQ server | required |
| `REDIS_ADDRESS` | Redis server | required |
| `REDIS_PASSWORD` | Redis auth | empty |
| `REDIS_DATABASE` | Redis DB index | `1` |
| `PROMETHEUS_LISTEN_ADDRESS` | Metrics port | `:2112` |

## Further reading

- [docs/architecture.md](docs/architecture.md)
- [docs/domain.md](docs/domain.md)
- [docs/dependencies.md](docs/dependencies.md)
- [docs/operations.md](docs/operations.md)
