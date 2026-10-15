# bin-timeline-manager

Platform-wide audit log and event timeline service. Subscribes to 27 event queues, stores events in ClickHouse, and exposes a cursor-paginated read API over RabbitMQ RPC.

> Cross-cutting rules (verification, branch/commit format, worktrees, Alembic, RST sync) live in the root [CLAUDE.md](../CLAUDE.md).

## Docs

- [docs/architecture.md](docs/architecture.md) — component overview, layer responsibilities, request routing
- [docs/domain.md](docs/domain.md) — Event entity, business rules, wildcard matching, SIP analysis
- [docs/dependencies.md](docs/dependencies.md) — 27 subscribed queues, infrastructure, Homer API
- [docs/operations.md](docs/operations.md) — failure modes, debugging, configuration, metrics

## Common Commands

```bash
# Build
go build -o ./bin/timeline-manager ./cmd/timeline-manager/
go build -o ./bin/timeline-control ./cmd/timeline-control/

# Test
go test -coverprofile cp.out -v $(go list ./...)

# timeline-control operations
./bin/timeline-control migrate up
./bin/timeline-control migrate version
./bin/timeline-control health

# Verification (mandatory before commit)
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

## Critical Implementation Notes

**ClickHouse type constraint**: The ClickHouse Go driver cannot scan `String` columns into custom Go types. `models/event/` must use `string` for `Publisher` and `Type`. Convert to `ServiceName` and other rich types only in `eventhandler` at the API boundary. Violating this causes runtime panics:
```
converting String to *outline.ServiceName is unsupported
```

**Read-write separation**: The RPC listen path is read-only. The subscribe handler ingests events from 27 queues into ClickHouse. These are independent code paths — do not mix.

**Batch ingestion**: Subscribe handler batches ClickHouse writes. Monitor `subscribe_batch_insert_time` and `subscribe_batch_size` metrics for ingestion health. The intake channel drops events silently when full — watch `subscribe_event_dropped_total` (any nonzero value is permanent data loss) and `subscribe_event_channel_usage` (occupancy histogram) for backpressure.

**ClickHouse migrations**: Managed via golang-migrate. Run `timeline-control migrate up` after adding new migration files in `migrations/`. File format: `NNNNNN_description.up.sql` / `NNNNNN_description.down.sql`.

## MySQL migrations (analysis store)

The MySQL table `timeline_analyses` is owned by this service and created at startup (golang-migrate, embedded with `go:embed`). It does not use Alembic. This is separate from the ClickHouse migrations above: different directory, different version table, independent runs. Tested engines: MariaDB and MySQL 8.0, so do not use engine specific syntax.

- Files: `migrations/mysql/NNNNNN_<table>_<verb>_<type>_<items>.up.sql` and `.down.sql` (six digits, from 000001 without gaps; package `mysqlmigrations`). `000001_baseline` is the schema at the time of the conversion. The ClickHouse files stay in `migrations/`; golang-migrate's file source ignores the `mysql/` subdirectory.
- `cmd/timeline-manager` runs `Up()` right after the MySQL connection of the analysis store, before the handlers use the tables. Failure stops the start (non-zero exit). Version table: `timeline_schema_migrations` (the ClickHouse one is `schema_migrations`). The MySQL store is optional: when `DATABASE_DSN` is empty the MySQL migrations are skipped, like the analysis feature itself.
- Rules for every new migration (one DDL statement per file, down for every up, `-- DOWN-LOSSY:` and `-- DESTRUCTIVE:` markers, immutable merged files, expand/contract) are the ones in [bin-tag-manager/CLAUDE.md](../bin-tag-manager/CLAUDE.md) and [docs/workflows/service-owned-migrations.md](../docs/workflows/service-owned-migrations.md). The baseline carries the table option `DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci` for `timeline_analyses`, which pins the collation independent of the server default.
- Recovery: `timeline-control migrate-mysql status|up|down|force <version>` talks to the MySQL database only (no ClickHouse, no RabbitMQ). Run it only against a local throwaway database. `timeline-control migrate ...` remains the ClickHouse command.
- Update `scripts/database_scripts_test/table_timeline_analyses.sql` (the sqlite test schema) by hand when the schema changes.
- Structure check: `go test ./migrations/mysql/`. Dynamic check on scratch containers: `ALEMBIC_VENV=<venv> MIGRATE_CMD=migrate-mysql scripts/verify-service-migrations.sh bin-timeline-manager migrations/mysql` (repo root).

**Uses Cobra + Viper** (not pflag directly) — see `internal/config/` for configuration loading.

## Testing Patterns

- gomock (go.uber.org/mock) for handler interface mocks
- Table-driven tests with context passed to all handler methods
- Mocks co-located in same package as interface definition
