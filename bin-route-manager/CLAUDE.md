# bin-route-manager

Outbound call routing management for VoIPbin. Manages SIP providers (gateways) and routing rules. Key feature: `GET /v1/dialroutes` merges customer-specific routes with system-default fallbacks to determine the effective carrier list for a given destination.

> Cross-cutting rules (verification workflow, branch/commit format, worktree usage, Alembic, RST sync) live in the root [CLAUDE.md](../CLAUDE.md).

## Documentation Index

| Doc | Contents |
|-----|----------|
| [docs/architecture.md](docs/architecture.md) | Component overview, layer responsibilities, request routing |
| [docs/domain.md](docs/domain.md) | Domain entities (Provider, Route, Dialroute), dialroute merge algorithm, business rules |
| [docs/dependencies.md](docs/dependencies.md) | Infrastructure, upstream/downstream services |
| [docs/operations.md](docs/operations.md) | Failure modes, debugging, configuration, Prometheus metrics |

## Quick Reference

**Build & run:**
```bash
go build -o ./bin/ ./cmd/...
./bin/route-manager
```

**Verify (required before commit):**
```bash
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**CLI tool (bypasses RabbitMQ):**
```bash
./bin/route-control route list --customer_id <uuid>
./bin/route-control route dialroute-list --customer_id <uuid> --target 1
```

**Migration status (database only, no Redis or RabbitMQ):**
```bash
./bin/route-control migrate status
```

**Mock regeneration:**
```bash
go generate ./pkg/listenhandler/...
go generate ./pkg/routehandler/...
go generate ./pkg/providerhandler/...
go generate ./pkg/dbhandler/...
go generate ./pkg/cachehandler/...
```

## Database migrations

The schema of `route_providercalls`, `route_providers` and `route_routes` is owned by this service and applied at startup (golang-migrate, embedded with `go:embed`). It does not use Alembic. Tested engines: MariaDB and MySQL 8.0, so do not use engine specific syntax.

- Files: `migrations/NNNNNN_<description>.up.sql` and `.down.sql` (six digits, from 000001 without gaps). `000001_baseline` is the schema at the time of the conversion (`CREATE TABLE IF NOT EXISTS`, no table options, down is the no-op `SELECT 1`). Version table: `route_schema_migrations`.
- `cmd/route-manager` runs `migrationhandler.Run` right after the database connection and before the cache and RabbitMQ. Failure stops the start (non-zero exit).
- `route-control migrate status|up|down|force <version>` talks to the database only. Run `up`, `down` and `force` only against a local throwaway database.
- The migration code is the shared `bin-common-handler/pkg/migrationhandler`. Rules for new files (one DDL statement per file, `-- DESTRUCTIVE:` and `-- DOWN-LOSSY:` markers, immutable merged files, expand/contract), the checks and the recovery steps are in [docs/workflows/service-owned-migrations.md](../docs/workflows/service-owned-migrations.md).
- A new database is created with `CHARACTER SET utf8 COLLATE utf8_general_ci` (like the install scripts do); other defaults make the table definitions differ from the Alembic head.
- New database order: Alembic first, then this service. The reverse fails in the Alembic stream (`Table 'route_providers' already exists`) and is unsupported while the Alembic stream still creates the route tables.
- Update `scripts/database_scripts/table_*.sql` (the sqlite test schema) by hand when the schema changes.
- Structure checks: `go test ./migrations/`. Git relative check: `scripts/check-service-migrations.sh bin-route-manager` after `git fetch origin main`. Dynamic check on scratch containers: `ALEMBIC_VENV=<venv> scripts/verify-service-migrations.sh bin-route-manager` (repo root).

## Key Facts

- **Queue (listen):** `bin-manager.route-manager.request`
- **No event subscriptions.** Events published: route_*, provider_*, providercall_* — published solely to the global topic exchange `bin-manager.event` as of VOIP-1407 (see docs/architecture.md)
- **Dialroute merge**: customer routes + `CustomerIDBasicRoute` (`00000000-0000-0000-0000-000000000001`) defaults — customer wins on provider overlap
- **Target values:** country code string (e.g., `"1"` for US) or `"all"` for catch-all
- **Route priority:** lower integer = higher priority
- **Provider types:** `sip` (only type currently)
- **Soft deletes:** `tm_delete = '9999-01-01 00:00:00.000000'` for both providers and routes
- **Config required:** `external_sip_gateway_addresses` for provider setup
- **Metrics port:** `:2112/metrics`
