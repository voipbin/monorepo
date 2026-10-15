# bin-number-manager

PSTN phone number lifecycle management — purchases, manages, and releases DID numbers via Telnyx and Twilio providers, and routes inbound calls/messages to flows.

> Cross-cutting rules (verification workflow, branch/commit format, worktrees, Alembic, RST sync) live in the root [CLAUDE.md](../CLAUDE.md).

## Docs

- [docs/architecture.md](docs/architecture.md) — components, layers, request routing
- [docs/domain.md](docs/domain.md) — Number, AvailableNumber, ProviderNumber entities; lifecycle rules; provider strategy
- [docs/dependencies.md](docs/dependencies.md) — subscribed events, outbound RPCs, monorepo deps
- [docs/operations.md](docs/operations.md) — failure modes, debugging, config reference, metrics

## CRITICAL: Balance Check Before Purchase

Every number purchase calls `bin-billing-manager` for balance validation **before** contacting the provider. Do not bypass this check.

## CRITICAL: Provider Credentials

Both Telnyx and Twilio credentials must be valid for number operations to succeed. Missing or expired credentials cause silent purchase failures at the provider level.

## Key Commands

```bash
# Build
go build -o ./bin/ ./cmd/...

# Test
go test -coverprofile cp.out -v $(go list ./...)

# Verification (mandatory before commit)
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

# number-control — direct DB/cache access, bypasses RabbitMQ
./bin/number-control migrate status   # schema migrations, database only (no Redis or RabbitMQ)
./bin/number-control number get --id <uuid>
./bin/number-control number list --customer-id <uuid>
./bin/number-control number get-available --country-code US
./bin/number-control number update --id <uuid> --call-flow-id <flow-uuid>
./bin/number-control number delete --id <uuid>

# Run locally
DATABASE_DSN="user:pass@tcp(127.0.0.1:3306)/bin_manager" \
RABBITMQ_ADDRESS="amqp://guest:guest@localhost:5672" \
REDIS_ADDRESS="127.0.0.1:6379" \
TELNYX_TOKEN="<token>" \
./bin/number-manager
```

## Architecture Summary

- **ListenHandler**: consumes `bin-manager.number-manager.request`; routes to numberhandler
- **SubscribeHandler**: `customer_deleted` → release all customer numbers; `flow_deleted` → clear flow references
- **Provider dispatch**: numberhandler delegates to `numberhandlertelnyx` or `numberhandlertwilio`
- **ProviderNumber record**: created alongside Number; holds provider's own reference ID for release operations

## Testing

Uses `go.uber.org/mock` (gomock). Table-driven tests. Mocks co-located with handlers.

```bash
go test -v -run TestName ./pkg/numberhandler/...
```

## Database Tables

- `number_numbers` — phone numbers (soft-delete with `tm_delete`). The only table owned by this service; `ProviderNumber` is an in-memory model, not a table.

## Database migrations

The schema of `number_numbers` is owned by this service and applied at startup (golang-migrate, embedded with `go:embed`). It does not use Alembic. Tested engines: MariaDB and MySQL 8.0, so do not use engine specific syntax.

- Files: `migrations/NNNNNN_<table>_<verb>_<type>_<items>.up.sql` and `.down.sql` (six digits, from 000001 without gaps). `000001_baseline` is the schema at the time of the conversion.
- `cmd/number-manager` runs `Up()` right after the database connection, before the cache and RabbitMQ. Failure stops the start (non-zero exit). Version table: `number_schema_migrations`.
- Rules for every new migration:
  - One DDL statement per file (DDL is not transactional; a failure in a later statement would leave the first applied with the database marked dirty). Only the baseline may hold several `CREATE TABLE IF NOT EXISTS` statements.
  - Every up has a down. A down that cannot restore data starts with `-- DOWN-LOSSY: <reason>`. An up with a destructive operation (DROP, TRUNCATE, RENAME, MODIFY/CHANGE COLUMN, UPDATE/DELETE/INSERT SELECT, adding UNIQUE/FK/CHECK/NOT NULL, SET/DROP DEFAULT, ENGINE/CHARSET/COLLATE/ROW_FORMAT, ON DUPLICATE KEY UPDATE) starts with `-- DESTRUCTIVE: <reason>` and its PR body starts with `[DESTRUCTIVE]`.
  - Merged files are immutable and numbers are never reused: fix mistakes with a new number, and a revert PR changes the code only, never `migrations/`. A number lower than the highest merged one is silently skipped on databases that are already ahead.
  - Expand/contract: the old code must work on the new schema. Add columns as NULL or with a DEFAULT and list columns explicitly in queries; add indexes with `ALGORITHM=INPLACE, LOCK=NONE`; drop a column in a later PR than the one that stops using it; replace instead of rename or retype (add, dual write, switch reads, drop); backfill before NOT NULL/UNIQUE/FK/CHECK. Large data changes do not belong in a migration file.
  - Update `scripts/database_scripts_test/table_numbers.sql` (in this service directory; the sqlite test schema) by hand when the schema changes.
- Safety behavior: the migration session uses `lock_wait_timeout=10`; two instances serialize on a MySQL named lock (an instance that cannot get it within 10 seconds fails while opening the migrator and is restarted); a database version higher than the image's newest file (not dirty) is skipped with a warning so an older image can still start; a dirty database refuses to start. The version check before `Up()` runs outside the lock, so an image rollback racing a migration can fail once and is absorbed by the restart.
- A new database is created with `CHARACTER SET utf8 COLLATE utf8_general_ci` (like the install scripts do); other defaults make the table definition differ from the Alembic head.
- New database order: Alembic first, then this service. The reverse fails in the Alembic stream (`Table 'number_numbers' already exists`) and is unsupported while the Alembic stream still creates `number_numbers`.
- The baseline is `CREATE TABLE IF NOT EXISTS`: it does not compare the shape of an existing table, so a database that is behind the Alembic head is not detected. Compare an existing database with the head before the first deployment (`REFERENCE_DUMP` of `scripts/verify-service-migrations.sh`).
- The migration code is the shared `bin-common-handler/pkg/migrationhandler` (`Run`, `Migrator`, `NewMigrateCommand`); this service holds only the SQL files, `migrations/embed.go`, the call in `cmd/number-manager` and the `migrate` command registration in `cmd/number-control`. See [docs/workflows/service-owned-migrations.md](../docs/workflows/service-owned-migrations.md).
- Structure checks: `go test ./migrations/` runs `migrationcheck.Check` (pairs, contiguous numbers, one statement, baseline rules, marker format) and the stray non-.sql file test. The `-- DESTRUCTIVE:` marker and the file description naming are not checked automatically yet; they are checked by hand in review. Git relative check (merged files unchanged, numbers continue origin/main): `scripts/check-service-migrations.sh bin-number-manager` after `git fetch origin main`. Dynamic check on scratch containers: `ALEMBIC_VENV=<venv> EXISTING_ROWS_SQL=<file> scripts/verify-service-migrations.sh bin-number-manager` (repo root; `EXISTING_ROWS_SQL` holds an INSERT statement for an existing row).
- Recovery from a dirty database: stop the crash-looping container, check whether the failed file's statement was applied, then `number-control migrate force <N-1 or N>` (`number-control migrate force -- -1` resets to unversioned). A file that failed on every environment may be fixed in place only when it is the highest number, never succeeded anywhere, and the PR body says `MIGRATION-FIX: N` with the failure log.
- Run `number-control migrate up|down|force` only against a local throwaway database. `status` still creates the version table when it is missing.
- A termination signal does not interrupt a running migration (the process is killed after the stop grace period), so a long DDL migration can leave the database dirty; see the recovery steps above.
- `number-control migrate status|up|down|force <version>` talks to the database only (no Redis or RabbitMQ). `down` is for development and emergencies and is never run automatically; the baseline down is a no-op.
