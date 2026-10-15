# bin-outdial-manager

Outbound dialing campaign management for VoIPbin. Stores outdial containers, individual call targets (up to 5 destinations per target with independent retry counters), and per-attempt call records. Primary consumer: `bin-campaign-manager`.

> Cross-cutting rules (verification workflow, branch/commit format, worktree usage, Alembic, RST sync) live in the root [CLAUDE.md](../CLAUDE.md).

## Documentation Index

| Doc | Contents |
|-----|----------|
| [docs/architecture.md](docs/architecture.md) | Component overview, layer responsibilities, request routing |
| [docs/domain.md](docs/domain.md) | Domain entities (Outdial, OutdialTarget, OutdialTargetCall), business rules |
| [docs/dependencies.md](docs/dependencies.md) | Infrastructure, upstream/downstream services, events |
| [docs/operations.md](docs/operations.md) | Failure modes, debugging, configuration, Prometheus metrics |

## Quick Reference

**Build & run:**
```bash
go build -o ./bin/ ./cmd/...
./bin/outdial-manager
```

**Verify (required before commit):**
```bash
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**CLI tool (bypasses RabbitMQ):**
```bash
./bin/outdial-control outdial list --customer_id <uuid>
./bin/outdial-control outdial get --id <outdial-uuid>
```

**Migration status (database only, no Redis or RabbitMQ):**
```bash
./bin/outdial-control migrate status
```

**Mock regeneration:**
```bash
go generate ./pkg/dbhandler/...
go generate ./pkg/cachehandler/...
go generate ./pkg/outdialhandler/...
go generate ./pkg/outdialtargethandler/...
```

## Database migrations

The schema of `outdial_outdials`, `outdial_outdialtargetcalls` and `outdial_outdialtargets` is owned by this service and applied at startup (golang-migrate, embedded with `go:embed`). It does not use Alembic. Tested engines: MariaDB and MySQL 8.0, so do not use engine specific syntax.

- Files: `migrations/NNNNNN_<description>.up.sql` and `.down.sql` (six digits, from 000001 without gaps). `000001_baseline` is the schema at the time of the conversion (`CREATE TABLE IF NOT EXISTS`, no table options, down is the no-op `SELECT 1`). Version table: `outdial_schema_migrations`.
- `cmd/outdial-manager` runs `migrationhandler.Run` in `createDBHandler`, right after the database connection and before the cache and RabbitMQ. Failure stops the start (non-zero exit).
- `outdial-control migrate status|up|down|force <version>` talks to the database only. Run `up`, `down` and `force` only against a local throwaway database.
- The migration code is the shared `bin-common-handler/pkg/migrationhandler`. Rules for new files (one DDL statement per file, `-- DESTRUCTIVE:` and `-- DOWN-LOSSY:` markers, immutable merged files, expand/contract), the checks and the recovery steps are in [docs/workflows/service-owned-migrations.md](../docs/workflows/service-owned-migrations.md).
- A new database is created with `CHARACTER SET utf8 COLLATE utf8_general_ci` (like the install scripts do); other defaults make the table definitions differ from the Alembic head.
- New database order: Alembic first, then this service. The reverse fails in the Alembic stream (`Table 'outdial_outdials' already exists`) and is unsupported while the Alembic stream still creates the outdial tables.
- Update `scripts/database_scripts/table_*.sql` (the sqlite test schema) by hand when the schema changes.
- Structure checks: `go test ./migrations/`. Git relative check: `scripts/check-service-migrations.sh bin-outdial-manager` after `git fetch origin main`. Dynamic check on scratch containers: `ALEMBIC_VENV=<venv> scripts/verify-service-migrations.sh bin-outdial-manager` (repo root).

## Key Facts

- **Queue (listen):** `bin-manager.outdial-manager.request`
- **No event subscriptions** (no subscribehandler)
- **Events published:** `outdial_created`, `outdial_updated`, `outdial_deleted`
- **Target statuses:** `idle` → `processing` → `done`
- **Multi-destination:** up to 5 destinations (`destination_0`–`destination_4`) with per-destination try counts
- **Available query:** filters targets by per-destination try-count thresholds for campaign retry logic
- **Soft deletes:** `tm_delete = '9999-01-01 00:00:00.000000'` for active records
- **Metrics port:** `:2112/metrics`
