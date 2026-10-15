# bin-webchat-manager

Website chat service. Owns the widget configuration, the visitor sessions and the messages of the webchat channel, and publishes `webchat_message_created` and `webchat_session_ended` events.

> Cross-cutting rules (verification workflow, branch/commit format, worktrees, Alembic, RST sync) live in the root [CLAUDE.md](../CLAUDE.md). This file covers only what is specific to `bin-webchat-manager`.

## Key facts

- **MySQL + Redis** for records and cache. Soft-deletes via `tm_delete`.
- **RabbitMQ queue**: `bin-manager.webchat-manager.request`
- **Tables**: `webchat_widgets`, `webchat_sessions`, `webchat_messages`

## Package layout

| Package | Role |
|---------|------|
| `cmd/webchat-manager` | Daemon entry point |
| `cmd/webchat-control` | Admin CLI; only `migrate` for now (database only, bypasses Redis and RabbitMQ) |
| `pkg/listenhandler` | RabbitMQ RPC router (regex dispatch) |
| `pkg/widgethandler`, `pkg/sessionhandler`, `pkg/messagehandler` | Business logic |
| `pkg/dbhandler`, `pkg/cachehandler` | MySQL and Redis |
| `migrations` | Embedded schema migration files |

## Database migrations

The schema of the three tables is owned by this service and applied at startup (golang-migrate, embedded with `go:embed`). It does not use Alembic. The full rules, the baseline contract and the verification script are in [docs/workflows/service-owned-migrations.md](../docs/workflows/service-owned-migrations.md); `bin-tag-manager/CLAUDE.md` is the reference service.

- Files: `migrations/NNNNNN_<table>_<verb>_<type>_<items>.up.sql` and `.down.sql` (six digits, from 000001 without gaps). `000001_baseline` is the schema at the time of the conversion. Version table: `webchat_schema_migrations`.
- `cmd/webchat-manager` runs `migrationhandler.Run` right after the database connection, before the cache and RabbitMQ. Failure stops the start (non-zero exit).
- Every new migration: one DDL statement per file, an up and a down, merged files are never changed, old code must work on the new schema (expand/contract), `-- DESTRUCTIVE:` and `-- DOWN-LOSSY:` markers as described in the workflow document. Update `scripts/database_scripts_test/*.sql` (the sqlite test schema) by hand when the schema changes.
- Checks: `go test ./migrations/` (structure) and, from the repo root, `ALEMBIC_VENV=<venv> EXISTING_ROWS_SQL=<file> scripts/verify-service-migrations.sh bin-webchat-manager` (scratch containers, MariaDB and MySQL 8.0; `EXISTING_ROWS_SQL` is a file with INSERT statements for the three tables, without it the existing database step is skipped).
- Recovery from a dirty database: `webchat-control migrate force <version>` (see `docs/operations.md`). Run `webchat-control migrate up|down|force` only against a local throwaway database.
- New database order: Alembic first, then this service. The reverse fails in the Alembic stream (`Table 'webchat_widgets' already exists`).

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
| `DATABASE_DSN` | MySQL DSN | `testid:testpassword@tcp(127.0.0.1:3306)/test` |
| `RABBITMQ_ADDRESS` | RabbitMQ server | `amqp://guest:guest@localhost:5672` |
| `REDIS_ADDRESS` | Redis server | `127.0.0.1:6379` |
| `REDIS_PASSWORD` | Redis auth | empty |
| `REDIS_DATABASE` | Redis DB index | `1` |
| `PROMETHEUS_ENDPOINT` | Metrics path | `/metrics` |
| `PROMETHEUS_LISTEN_ADDRESS` | Metrics port | `:2112` |

## Further reading

- [README.md](README.md)
- [docs/operations.md](docs/operations.md)
