# bin-message-manager

SMS messaging service — sends messages via Telnyx and MessageBird, tracks per-target delivery status, and processes provider webhooks for delivery updates.

> Cross-cutting rules (verification workflow, branch/commit format, worktrees, Alembic, RST sync) live in the root [CLAUDE.md](../CLAUDE.md).

## Docs

- [docs/architecture.md](docs/architecture.md) — components, layers, request routing
- [docs/domain.md](docs/domain.md) — Message, Target entities; send flow; provider selection; webhook processing
- [docs/dependencies.md](docs/dependencies.md) — events published, outbound RPCs, monorepo deps
- [docs/operations.md](docs/operations.md) — failure modes, debugging, config reference, metrics

## CRITICAL: Balance Check Before Send

`POST /v1/messages` calls `bin-billing-manager` for balance validation before creating the message. Do not bypass this check.

## CRITICAL: Async Provider Dispatch

After message creation, provider send runs in a **goroutine** — it is non-blocking to the RPC caller. The caller receives success immediately after the message record is created; actual delivery is asynchronous.

## Database migrations

The schema of the table `message_messages` is owned by this service and applied at startup (golang-migrate, embedded with `go:embed`). It does not use Alembic. The full rules, the baseline contract and the verification script are in [docs/workflows/service-owned-migrations.md](../docs/workflows/service-owned-migrations.md); `bin-tag-manager/CLAUDE.md` is the reference service.

- Files: `migrations/NNNNNN_<table>_<verb>_<type>_<items>.up.sql` and `.down.sql` (six digits, from 000001 without gaps). `000001_baseline` is the schema at the time of the conversion. Version table: `message_schema_migrations`.
- `cmd/message-manager` runs `migrationhandler.Run` right after the database connection, before the cache and RabbitMQ. Failure stops the start (non-zero exit).
- Every new migration: one DDL statement per file, an up and a down, merged files are never changed, old code must work on the new schema (expand/contract), `-- DESTRUCTIVE:` and `-- DOWN-LOSSY:` markers as described in the workflow document. Update `scripts/database_scripts_test/*.sql` (the sqlite test schema) by hand when the schema changes.
- Checks: `go test ./migrations/` (structure) and, from the repo root, `ALEMBIC_VENV=<venv> EXISTING_ROWS_SQL=<file> scripts/verify-service-migrations.sh bin-message-manager` (scratch containers, MariaDB and MySQL 8.0; `EXISTING_ROWS_SQL` is a file with INSERT statements for the table, without it the existing database step is skipped).
- Recovery from a dirty database: `message-control migrate force <version>` (see `docs/operations.md`). A file that failed on every environment may be fixed in place only when it is the highest number, never succeeded anywhere, and the PR body says `MIGRATION-FIX: N` with the failure log. Run `message-control migrate up|down|force` only against a local throwaway database.
- New database order: Alembic first, then this service. The reverse fails in the Alembic stream (`Table 'message_messages' already exists`).

## Key Commands

```bash
# Build
go build -o ./bin/ ./cmd/...

# Test
go test -coverprofile cp.out -v $(go list ./...)

# Verification (mandatory before commit)
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

# message-control — direct DB/cache access, bypasses RabbitMQ (the migrate command opens only the database)
./bin/message-control migrate status
./bin/message-control message get --id <uuid>
./bin/message-control message list --customer_id <uuid> [--limit 100] [--token]
./bin/message-control message delete --id <uuid>

# Run locally
DATABASE_DSN="user:pass@tcp(127.0.0.1:3306)/voipbin" \
RABBITMQ_ADDRESS="amqp://guest:guest@localhost:5672" \
REDIS_ADDRESS="127.0.0.1:6379" \
AUTHTOKEN_MESSAGEBIRD="token" \
AUTHTOKEN_TELNYX="token" \
./bin/message-manager
```

## Architecture Summary

- **No SubscribeHandler** — webhook events arrive via `POST /v1/hooks` from bin-hook-manager
- **ListenHandler**: consumes `bin-manager.message-manager.request`
- **Provider dispatch**: MessageBird primary, Telnyx fallback; goroutine-based async send
- **Database**: Squirrel SQL builder (not raw SQL); Targets tracked separately per recipient

## Testing

Uses `go.uber.org/mock` (gomock). Table-driven tests. Mocks for messagehandler, providers, dbhandler.

```bash
go test -v ./pkg/messagehandler -run Test_Send
```

## Webhook Configuration

Provider webhook endpoints (via bin-hook-manager):
- MessageBird: `https://hook.voipbin.net/v1.0/messages/messagebird`
- Telnyx: `https://hook.voipbin.net/v1.0/messages/telnyx`
