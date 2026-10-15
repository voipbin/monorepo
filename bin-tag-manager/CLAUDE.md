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

## Database migrations

The schema of `tag_tags` is owned by this service and applied at startup (golang-migrate, embedded with `go:embed`). It does not use Alembic. Tested engines: MariaDB and MySQL 8.0, so do not use engine specific syntax.

- Files: `migrations/NNNNNN_<table>_<verb>_<type>_<items>.up.sql` and `.down.sql` (six digits, from 000001 without gaps). `000001_baseline` is the schema at the time of the conversion.
- `cmd/tag-manager` runs `Up()` right after the database connection, before the cache and RabbitMQ. Failure stops the start (non-zero exit). Version table: `tag_schema_migrations`.
- Rules for every new migration:
  - One DDL statement per file (DDL is not transactional; a failure in a later statement would leave the first applied with the database marked dirty). Only the baseline may hold several `CREATE TABLE IF NOT EXISTS` statements.
  - Every up has a down. A down that cannot restore data starts with `-- DOWN-LOSSY: <reason>`. An up with a destructive operation (DROP, TRUNCATE, RENAME, MODIFY/CHANGE COLUMN, UPDATE/DELETE/INSERT SELECT, adding UNIQUE/FK/CHECK/NOT NULL, SET/DROP DEFAULT, ENGINE/CHARSET/COLLATE/ROW_FORMAT, ON DUPLICATE KEY UPDATE) starts with `-- DESTRUCTIVE: <reason>` and its PR body starts with `[DESTRUCTIVE]`.
  - Merged files are immutable and numbers are never reused: fix mistakes with a new number, and a revert PR changes the code only, never `migrations/`. A number lower than the highest merged one is silently skipped on databases that are already ahead.
  - Expand/contract: the old code must work on the new schema. Add columns as NULL or with a DEFAULT and list columns explicitly in queries; add indexes with `ALGORITHM=INPLACE, LOCK=NONE`; drop a column in a later PR than the one that stops using it; replace instead of rename or retype (add, dual write, switch reads, drop); backfill before NOT NULL/UNIQUE/FK/CHECK. Large data changes do not belong in a migration file.
  - Update `scripts/database_scripts_test/tags.sql` (the sqlite test schema) by hand when the schema changes.
- Safety behavior: the migration session uses `lock_wait_timeout=10`; two instances serialize on a MySQL named lock (an instance that cannot get it within 10 seconds fails while opening the migrator and is restarted); a database version higher than the image's newest file (not dirty) is skipped with a warning so an older image can still start; a dirty database refuses to start. The version check before `Up()` runs outside the lock, so an image rollback racing a migration can fail once and is absorbed by the restart.
- A new database is created with `CHARACTER SET utf8 COLLATE utf8_general_ci` (like the install scripts do); other defaults make the table definitions differ from the Alembic head.
- New database order: Alembic first, then this service. The reverse fails in the Alembic stream (`Table 'tag_tags' already exists`) and is unsupported while the Alembic stream still creates `tag_tags`.
- The migration code is the shared `bin-common-handler/pkg/migrationhandler` (`Run`, `Migrator`, `NewMigrateCommand`); this service holds only the SQL files, `migrations/embed.go`, the call in `cmd/tag-manager` and the `migrate` command registration in `cmd/tag-control`. See [docs/workflows/service-owned-migrations.md](../docs/workflows/service-owned-migrations.md).
- Structure checks: `go test ./migrations/` runs `migrationcheck.Check` (pairs, contiguous numbers, one statement, baseline rules, marker format) and the no stray non-.sql file test. The `-- DESTRUCTIVE:` marker and the file description naming are not checked automatically yet; they are checked by hand in review. Git relative check (merged files unchanged, numbers continue origin/main): `scripts/check-service-migrations.sh bin-tag-manager` after `git fetch origin main`. Dynamic check on scratch containers: `ALEMBIC_VENV=<venv> scripts/verify-service-migrations.sh bin-tag-manager` (repo root).
- Recovery from a dirty database: stop the crash-looping container, check whether the failed file's statement was applied, then `tag-control migrate force <N-1 or N>` (`tag-control migrate force -- -1` resets to unversioned). A file that failed on every environment may be fixed in place only when it is the highest number, never succeeded anywhere, and the PR body says `MIGRATION-FIX: N` with the failure log.
- Run `tag-control migrate up|down|force` only against a local throwaway database. `status` still creates the version table when it is missing.
- A termination signal does not interrupt a running migration (the process is killed after the stop grace period), so a long DDL migration can leave the database dirty; see the recovery steps above.
- `tag-control migrate status|up|down|force <version>` talks to the database only (no Redis or RabbitMQ). `down` is for development and emergencies and is never run automatically; the baseline down is a no-op.

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
./bin/tag-control migrate status
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
