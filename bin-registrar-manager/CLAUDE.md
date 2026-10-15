# bin-registrar-manager

SIP registration management for VoIPbin. Provisions and manages Asterisk PJSIP extensions (user endpoints) and trunks (carrier connections) across two databases. Serves contact registration lookups on the SIP ingress path.

> Cross-cutting rules (verification workflow, branch/commit format, worktree usage, Alembic, RST sync) live in the root [CLAUDE.md](../CLAUDE.md).

## Documentation Index

| Doc | Contents |
|-----|----------|
| [docs/architecture.md](docs/architecture.md) | Component overview, dual-DB architecture, layer responsibilities, request routing |
| [docs/domain.md](docs/domain.md) | Domain entities (Extension, Trunk, Contact), Asterisk table mapping, business rules |
| [docs/dependencies.md](docs/dependencies.md) | Infrastructure (two MySQL DBs), upstream/downstream services, events |
| [docs/operations.md](docs/operations.md) | Failure modes, debugging Asterisk orphans, configuration, Prometheus metrics |

## Quick Reference

**Build & run:**
```bash
go build -o bin/registrar-manager ./cmd/registrar-manager
./bin/registrar-manager \
  --database_dsn_asterisk "user:pass@tcp(host:3306)/asterisk" \
  --database_dsn_bin "user:pass@tcp(host:3306)/bin_manager" \
  --rabbitmq_address "amqp://guest:guest@localhost:5672" \
  --redis_address "localhost:6379" \
  --domain_name_extension "ext.voipbin.net" \
  --domain_name_trunk "trunk.voipbin.net"
```

**Verify (required before commit):**
```bash
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**CLI tool (bypasses RabbitMQ):**
```bash
./bin/registrar-control extension list --customer_id <uuid>
./bin/registrar-control trunk list --customer_id <uuid>
```

**Migration status (bin database only, no asterisk database, Redis or RabbitMQ):**
```bash
./bin/registrar-control migrate status
```

**Mock regeneration:**
```bash
go generate ./...
```

## Database migrations

The schema of `registrar_customer_domains`, `registrar_extensions`, `registrar_sip_auths` and `registrar_trunks` (bin database) is owned by this service and applied at startup (golang-migrate, embedded with `go:embed`). It does not use Alembic. The asterisk database (`ps_*` tables) stays in the Alembic `asterisk_config` stream and is not touched. Tested engines: MariaDB and MySQL 8.0, so do not use engine specific syntax.

- Files: `migrations/NNNNNN_<description>.up.sql` and `.down.sql` (six digits, from 000001 without gaps). `000001_baseline` is the schema at the time of the conversion (`CREATE TABLE IF NOT EXISTS`, no table options, down is the no-op `SELECT 1`). Version table: `registrar_schema_migrations`.
- `cmd/registrar-manager` runs `migrationhandler.Run` with `DATABASE_DSN_BIN` in `runDaemon`, right after the bin database connection and before the asterisk connection, the cache and RabbitMQ. Failure stops the start (non-zero exit).
- `registrar-control migrate status|up|down|force <version>` talks to the bin database only (it still needs the same `DOMAIN_NAME_EXTENSION` and `DOMAIN_NAME_TRUNK` settings as every registrar-control command, because the root command loads the whole configuration). Run `up`, `down` and `force` only against a local throwaway database.
- The migration code is the shared `bin-common-handler/pkg/migrationhandler`. Rules for new files (one DDL statement per file, `-- DESTRUCTIVE:` and `-- DOWN-LOSSY:` markers, immutable merged files, expand/contract), the checks and the recovery steps are in [docs/workflows/service-owned-migrations.md](../docs/workflows/service-owned-migrations.md).
- A new database is created with `CHARACTER SET utf8 COLLATE utf8_general_ci` (like the install scripts do); other defaults make the table definitions differ from the Alembic head.
- New database order: Alembic first, then this service. The reverse fails in the Alembic stream (`Table 'registrar_extensions' already exists`) and is unsupported while the Alembic stream still creates the registrar tables.
- Structure checks: `go test ./migrations/`. Git relative check: `scripts/check-service-migrations.sh bin-registrar-manager` after `git fetch origin main`. Dynamic check on scratch containers: `ALEMBIC_VENV=<venv> scripts/verify-service-migrations.sh bin-registrar-manager` with `DSN_VAR=DATABASE_DSN_BIN` and a `DAEMON_ENV` that also sets `DATABASE_DSN_ASTERISK` (repo root).

## Key Facts

- **Queue (listen):** `bin-manager.registrar-manager.request`
- **Queue (subscribe):** topic-pattern binding on the global topic exchange `bin-manager.event` (fanout removed as of VOIP-1407) → cascade delete on `customer_deleted`
- **Events published**: trunk_created/updated/deleted, extension_created/updated/deleted — published solely to the global topic exchange `bin-manager.event` as of VOIP-1407 (see docs/architecture.md)
- **Two databases**: `DATABASE_DSN_BIN` (extensions/trunks) + `DATABASE_DSN_ASTERISK` (`ps_*` tables)
- **Asterisk tables managed:** `ps_endpoints`, `ps_aors`, `ps_auths` (create/delete); `ps_contacts` (read-only)
- **Critical**: Extension create/delete must touch all three Asterisk tables atomically
- **Domain config required**: `domain_name_extension` and `domain_name_trunk` must be set at startup
- **Soft deletes:** bin-manager tables use `tm_delete`; Asterisk tables are hard-deleted
- **Metrics port:** `:2112/metrics`
