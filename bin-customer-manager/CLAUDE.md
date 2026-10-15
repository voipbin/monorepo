# bin-customer-manager

Foundational identity service managing tenant organizations (customers) and their API credentials (access keys). Almost all other services depend on customer context — this is the root of the VoIPbin tenant hierarchy.

> Cross-cutting rules (verification workflow, branch/commit format, worktrees, Alembic, RST sync) live in the root [CLAUDE.md](../CLAUDE.md).

## Docs

- [docs/architecture.md](docs/architecture.md) — components, layers, request routing
- [docs/domain.md](docs/domain.md) — Customer, AccessKey entities; lifecycle rules; cascade behavior
- [docs/dependencies.md](docs/dependencies.md) — events published, outbound RPCs, monorepo deps
- [docs/operations.md](docs/operations.md) — failure modes, debugging, config reference, metrics

## CRITICAL: No SubscribeHandler

This service does not consume events from other services. All state is driven by inbound RPC only. If you add event-driven behavior, it must go through the request queue.

## CRITICAL: Foundational Dependency

`bin-customer-manager` events (`customer_deleted`) trigger cascading cleanup in `bin-number-manager` and `bin-billing-manager`. Ensure event publishing is functional before deploying changes.

## Key Commands

```bash
# Build
go build -o bin/customer-manager ./cmd/customer-manager
go build -o bin/customer-control ./cmd/customer-control

# Test
go test -coverprofile cp.out -v $(go list ./...)

# Verification (mandatory before commit)
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

# customer-control migrate status — database only (no Redis or RabbitMQ)
./bin/customer-control migrate status

# customer-control — direct DB/cache access, bypasses RabbitMQ
./bin/customer-control customer get --id <uuid>
./bin/customer-control customer list [--limit 100] [--token]
./bin/customer-control accesskey list --customer-id <uuid>
./bin/customer-control customer create --email test@example.com --name "Test"

# Run locally
DATABASE_DSN="user:pass@tcp(127.0.0.1:3306)/voipbin" \
RABBITMQ_ADDRESS="amqp://guest:guest@localhost:5672" \
REDIS_ADDRESS="127.0.0.1:6379" \
./bin/customer-manager
```

## Architecture Summary

- **No SubscribeHandler** — purely RPC-driven
- **ListenHandler**: consumes `bin-manager.customer-manager.request`
- **Cache-first reads**: Redis → MySQL on miss; invalidated on all mutations
- **Pagination**: cursor-based (`tm_create` timestamp as page token)
- **Outbound RPC**: agent-manager (username check), billing-manager (account link validation)

## Testing

Uses `go.uber.org/mock` (gomock). Table-driven tests. Parameterized raw SQL (no Squirrel).

```bash
go test -v ./pkg/customerhandler -run Test_Delete
```

## Database migrations

The schema of `customer_customers` and `customer_accesskeys` is owned by this service and applied at startup (golang-migrate, embedded with `go:embed`). It does not use Alembic. Tested engines: MariaDB and MySQL 8.0, so do not use engine specific syntax.

- Files: `migrations/NNNNNN_<description>.up.sql` and `.down.sql` (six digits, from 000001 without gaps). `000001_baseline` is the schema at the time of the conversion (`CREATE TABLE IF NOT EXISTS`, no table options, down is the no-op `SELECT 1`). Version table: `customer_schema_migrations`.
- `cmd/customer-manager` runs `migrationhandler.Run` in `runDaemon`, right after the database connection and before the cache and RabbitMQ. Failure stops the start (non-zero exit).
- `customer-control migrate status|up|down|force <version>` talks to the database only. Run `up`, `down` and `force` only against a local throwaway database.
- The migration code is the shared `bin-common-handler/pkg/migrationhandler`. Rules for new files (one DDL statement per file, `-- DESTRUCTIVE:` and `-- DOWN-LOSSY:` markers, immutable merged files, expand/contract), the checks and the recovery steps are in [docs/workflows/service-owned-migrations.md](../docs/workflows/service-owned-migrations.md).
- A new database is created with `CHARACTER SET utf8 COLLATE utf8_general_ci` (like the install scripts do); other defaults make the table definitions differ from the Alembic head.
- New database order: Alembic first, then this service. The reverse fails in the Alembic stream (`Table 'customer_customers' already exists`) and is unsupported while the Alembic stream still creates the customer tables.
- Update `scripts/database_scripts_test/*.sql` (the sqlite test schema) by hand when the schema changes.
- Structure checks: `go test ./migrations/`. Git relative check: `scripts/check-service-migrations.sh bin-customer-manager` after `git fetch origin main`. Dynamic check on scratch containers: `ALEMBIC_VENV=<venv> scripts/verify-service-migrations.sh bin-customer-manager` (repo root).

## Database Tables

- `customer_customers` — customers (soft-delete with `tm_delete`)
- `customer_accesskeys` — access keys (soft-delete with `tm_delete`)
