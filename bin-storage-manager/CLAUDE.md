# bin-storage-manager

File and media storage service for the VoIPbin platform. Manages customer storage accounts with 10 GB quota enforcement, stores files and call recordings in Google Cloud Storage, generates signed download URLs, and handles cascading deletes when customers are removed.

> Cross-cutting rules (verification workflow, branch/commit format, worktrees, Alembic, RST sync) live in the root [CLAUDE.md](../CLAUDE.md). This file covers only what is specific to `bin-storage-manager`.

## Key facts

- **MySQL + Redis** for records and cache.
- **Two GCS buckets**: `gcp_bucket_name_media` (persistent) and `gcp_bucket_name_tmp` (transient zips).
- **Signed URLs** default to 24-hour expiry; refresh via `POST /v1/files/<uuid>/download_uri_refresh`.
- **RabbitMQ queue**: `bin-manager.storage-manager.request`
- **Subscribes to**: `bin-manager.customer-manager.event` (`customer_deleted` → cascading delete)

## Package layout

| Package | Role |
|---------|------|
| `cmd/storage-manager` | Daemon entry point |
| `cmd/storage-control` | Admin CLI (JSON output, bypasses RabbitMQ); `migrate` subcommand for the schema migrations |
| `migrations` | Embedded golang-migrate SQL files (service-owned schema of `storage_accounts`, `storage_files`) |
| `pkg/listenhandler` | RabbitMQ RPC router (regex dispatch) |
| `pkg/subscribehandler` | Event consumer (customer_deleted) |
| `pkg/storagehandler` | Core business logic |
| `pkg/filehandler` | GCS operations and signed URL generation (signing needs a `GOOGLE_APPLICATION_CREDENTIALS` service account key file; no in-cluster metadata-server fallback — see [Signing degradation](#signing-degradation)) |
| `pkg/accounthandler` | 10 GB quota enforcement |
| `pkg/dbhandler` | MySQL reads/writes |
| `pkg/cachehandler` | Redis cache |

## Database migrations

The schema of `storage_accounts` and `storage_files` is owned by this service and applied at startup (golang-migrate, embedded with `go:embed`). It does not use Alembic. Tested engines: MariaDB and MySQL 8.0, so do not use engine specific syntax.

- Files: `migrations/NNNNNN_<table>_<verb>_<type>_<items>.up.sql` and `.down.sql` (six digits, from 000001 without gaps). `000001_baseline` is the schema at the time of the conversion.
- `cmd/storage-manager` runs `Up()` right after the database connection, before the cache and RabbitMQ. Failure stops the start (non-zero exit). Version table: `storage_schema_migrations`.
- Rules for every new migration:
  - One DDL statement per file (DDL is not transactional; a failure in a later statement would leave the first applied with the database marked dirty). Only the baseline may hold several `CREATE TABLE IF NOT EXISTS` statements.
  - Every up has a down. A down that cannot restore data starts with `-- DOWN-LOSSY: <reason>`. An up with a destructive operation (DROP, TRUNCATE, RENAME, MODIFY/CHANGE COLUMN, UPDATE/DELETE/INSERT SELECT, adding UNIQUE/FK/CHECK/NOT NULL, SET/DROP DEFAULT, ENGINE/CHARSET/COLLATE/ROW_FORMAT, ON DUPLICATE KEY UPDATE) starts with `-- DESTRUCTIVE: <reason>` and its PR body starts with `[DESTRUCTIVE]`.
  - Merged files are immutable and numbers are never reused: fix mistakes with a new number, and a revert PR changes the code only, never `migrations/`. A number lower than the highest merged one is silently skipped on databases that are already ahead.
  - Expand/contract: the old code must work on the new schema. Add columns as NULL or with a DEFAULT and list columns explicitly in queries; add indexes with `ALGORITHM=INPLACE, LOCK=NONE`; drop a column in a later PR than the one that stops using it; replace instead of rename or retype (add, dual write, switch reads, drop); backfill before NOT NULL/UNIQUE/FK/CHECK. Large data changes do not belong in a migration file.
  - Update `scripts/database_scripts_test/table_storage_accounts.sql` and `table_storage_files.sql` (in this service directory; the sqlite test schema) by hand when the schema changes.
- Safety behavior: the migration session uses `lock_wait_timeout=10`; two instances serialize on a MySQL named lock (an instance that cannot get it within 10 seconds fails while opening the migrator and is restarted); a database version higher than the image's newest file (not dirty) is skipped with a warning so an older image can still start; a dirty database refuses to start. The version check before `Up()` runs outside the lock, so an image rollback racing a migration can fail once and is absorbed by the restart.
- A new database is created with `CHARACTER SET utf8 COLLATE utf8_general_ci` (like the install scripts do); other defaults make the table definitions differ from the Alembic head.
- New database order: Alembic first, then this service. The reverse fails in the Alembic stream (`already exists` on one of the two tables) and is unsupported while the Alembic stream still creates these tables.
- The migration code is the shared `bin-common-handler/pkg/migrationhandler` (`Run`, `Migrator`, `NewMigrateCommand`); this service holds only the SQL files, `migrations/embed.go`, the call in `cmd/storage-manager` and the `migrate` command registration in `cmd/storage-control`. See [docs/workflows/service-owned-migrations.md](../docs/workflows/service-owned-migrations.md).
- Structure checks: `go test ./migrations/` runs `migrationcheck.Check` (pairs, contiguous numbers, one statement, baseline rules, marker format) and the stray non-.sql file test. The `-- DESTRUCTIVE:` marker and the file description naming are not checked automatically yet; they are checked by hand in review. Git relative check (merged files unchanged, numbers continue origin/main): `scripts/check-service-migrations.sh bin-storage-manager` after `git fetch origin main`. Dynamic check on scratch containers: `ALEMBIC_VENV=<venv> EXISTING_ROWS_SQL=<file> scripts/verify-service-migrations.sh bin-storage-manager` (repo root; `EXISTING_ROWS_SQL` holds INSERT statements for existing rows of both tables).
- Recovery from a dirty database: stop the crash-looping container, check whether the failed file's statement was applied, then `storage-control migrate force <N-1 or N>` (`storage-control migrate force -- -1` resets to unversioned). A file that failed on every environment may be fixed in place only when it is the highest number, never succeeded anywhere, and the PR body says `MIGRATION-FIX: N` with the failure log.
- Run `storage-control migrate up|down|force` only against a local throwaway database. `status` still creates the version table when it is missing.
- A termination signal does not interrupt a running migration (the process is killed after the stop grace period), so a long DDL migration can leave the database dirty; see the recovery steps above.
- `storage-control migrate status|up|down|force <version>` talks to the database only (no Redis or RabbitMQ). `down` is for development and emergencies and is never run automatically; the baseline down is a no-op.

## Request routing

| Pattern | Operations |
|---------|-----------|
| `/v1/accounts?` | GET (list) |
| `/v1/accounts$` | POST (create) |
| `/v1/accounts/<uuid>$` | GET, DELETE |
| `/v1/files?` | GET (list) |
| `/v1/files$` | POST (create) |
| `/v1/files/<uuid>$` | GET, DELETE |
| `/v1/files/<uuid>/download_uri_refresh$` | POST (refresh URL) |
| `/v1/compressfiles$` | POST (create zip) |
| `/v1/recordings/(.*)` | GET, DELETE (by reference_id) |

## storage-control CLI

```bash
./bin/storage-control file list --customer_id <uuid> --limit 50
./bin/storage-control account get --id <uuid>
./bin/storage-control recording list --customer_id <uuid>
```

## Common commands

```bash
go build -o ./bin/storage-manager ./cmd/storage-manager
go test ./...
go generate ./pkg/filehandler/...
golangci-lint run -v --timeout 5m
```

## Configuration

| Env | Description | Default |
|-----|-------------|---------|
| `DATABASE_DSN` | MySQL DSN | required |
| `RABBITMQ_ADDRESS` | RabbitMQ server | required |
| `REDIS_ADDRESS` | Redis server | required |
| `GCP_PROJECT_ID` | GCP project | required |
| `GCP_BUCKET_NAME_MEDIA` | Persistent bucket | required |
| `GCP_BUCKET_NAME_TMP` | Temporary bucket | required |
| `GOOGLE_APPLICATION_CREDENTIALS` | Service account JSON key path, used to sign download URLs | optional (see below) |
| `PROMETHEUS_LISTEN_ADDRESS` | Metrics port | `:2112` |

## Signing degradation

`GOOGLE_APPLICATION_CREDENTIALS` is optional — the service boots without it instead of
crash-looping.

- **Unset**: `NewFileHandler` logs a warning and returns a working handler with no
  private key. Every signing-dependent read path (`DownloadURIGet`,
  `DownloadURIRefresh`, and their `storagehandler` consumers `CompressfileCreate` and
  `RecordingGet`) returns a structured `*cerrors.VoipbinError` with status
  `UNAVAILABLE` and reason `SIGNING_NOT_CONFIGURED`, which `listenhandler.errorResponse`
  translates into a typed 503 for API callers instead of an opaque 500.
- **Unset AND no other ADC source**: the variable is also the storage client's primary
  application-default-credentials source, so on a deployment with no credential at all
  (no key file, no GKE workload-identity metadata server) the authenticated client
  cannot be built either. `newStorageClient` falls back to an unauthenticated client so
  the process still starts; GCS calls then fail per request rather than at boot. Only
  the non-GCS capabilities (file get/list from MySQL, account bookkeeping, the
  `customer_deleted` cascade) are fully functional in that mode.
- **Set but unreadable/unparsable**: still fatal. Absence of the variable is an
  intentional keyless deployment; a broken path is misconfiguration and must be loud.
- **File `Create`**: tolerates ANY download-URI failure — the missing-key error above
  and a sign-time rejection of a present-but-invalid key. The GCS object has already
  been moved to its final location by that point, so failing would break the primary
  write path AND orphan the object. Instead the record is persisted with an empty
  `uri_download` and a NULL `tm_download_expire`, the error is logged, and `Create`
  succeeds. `POST /v1/files/<uuid>/download_uri_refresh` populates the URI later, once
  a usable signing key exists.

Unaffected without a key: file get/list/delete, account bookkeeping, and the
`customer_deleted` cascade.

## Further reading

- [docs/architecture.md](docs/architecture.md)
- [docs/domain.md](docs/domain.md)
- [docs/dependencies.md](docs/dependencies.md)
- [docs/operations.md](docs/operations.md)
