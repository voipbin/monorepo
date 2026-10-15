# bin-schedule-manager

Platform-internal cron scheduler (VOIP-1283). Absorbs external cron surfaces (the number-manager Kubernetes CronJob, future ticker migrations) into one DB-driven dispatch engine with an audit trail, at-most-once claim semantics, and Prometheus visibility. Phase 1 schedules are all platform-owned (nil customer); there is no user-visible API surface yet.

> Cross-cutting rules (verification workflow, branch/commit format, worktrees, Alembic, RST sync) live in the root [CLAUDE.md](../CLAUDE.md). This file covers only what is specific to `bin-schedule-manager`.

## Key facts

- **MySQL** tables `schedule_schedules` / `schedule_executions` (audit trail), **Redis** claim locks via redsync (`schedule:lock:<id>`).
- **RabbitMQ queue**: `bin-manager.schedule-manager.request`
- **Subscribes to**: `bin-manager.customer-manager.event` (`customer_deleted` → delete the customer's schedules)
- **Publishes**: internal events only on `bin-manager.schedule-manager.event` (`schedule_created/updated/deleted`, `execution_succeeded/failed`). **No customer webhooks in Phase 1** — every schedule is nil-customer, so there is deliberately no `models/schedule/webhook.go` and no RST struct docs (Phase 3 scope).
- **replicas: 2** by design — every replica runs the same tick loop; Redis lock + DB CAS claim make concurrent replicas safe (kill -9 failover without double-fire).
- **Housekeeping dogfoods the engine**: `execution-retention` and `database-backup` are seeded schedules that target the service's own request queue (`/v1/executions/prune`, `/v1/backups`).

## Package layout

| Package | Role |
|---------|------|
| `cmd/schedule-manager` | Daemon entry point (cobra; Bootstrap/LoadGlobalConfig) |
| `cmd/schedule-control` | Admin CLI (direct DB/cache, no RabbitMQ — works when the broker path is unhealthy) |
| `internal/config` | Viper/pflag config binding |
| `models/schedule` | Schedule struct, cron helpers, filters, internal event types |
| `models/execution` | Execution struct (audit row), status machine, filters |
| `pkg/listenhandler` | RabbitMQ RPC router (regex dispatch) |
| `pkg/subscribehandler` | Event consumer (`customer_deleted`) |
| `pkg/schedulehandler` | Schedule CRUD, validation, next-run computation, name uniqueness |
| `pkg/dispatchhandler` | Tick loop, claim (lock + CAS), dispatch, record, reaper, manual execute |
| `pkg/backuphandler` | mariadb-dump subprocess + gzip + retention pruning (design §7) |
| `pkg/dbhandler` | MySQL via squirrel; sqlite test harness |
| `pkg/cachehandler` | Redis + redsync claim locks |

## Database migrations

The schema of `schedule_schedules` and `schedule_executions` is owned by this service and applied at startup (golang-migrate, embedded with `go:embed`). It does not use Alembic. Tested engines: MariaDB and MySQL 8.0, so do not use engine specific syntax. The shared code is `bin-common-handler/pkg/migrationhandler`; the workflow is in [docs/workflows/service-owned-migrations.md](../docs/workflows/service-owned-migrations.md), and the rules for every new file (one DDL statement per file, up/down pairs, `-- DESTRUCTIVE:` and `-- DOWN-LOSSY:` markers, immutable merged files, expand/contract) are in the same document.

- Files: `migrations/NNNNNN_<table>_<verb>_<type>_<items>.up.sql` and `.down.sql` (six digits, from 000001 without gaps). Version table: `schedule_schema_migrations`.
  - `000001_baseline`: `CREATE TABLE IF NOT EXISTS` of the two tables, with the table option `DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci ROW_FORMAT=DYNAMIC` that Alembic declared. The down is a no-op.
  - `000002_schedule_schedules_seed_platform_jobs`: the nine platform schedules (`number-renew`, `database-backup`, `execution-retention`, `billing-monthly-topup`, `billing-failed-event-retry`, `ai-proposal-expiry`, `customer-unverified-cleanup`, `customer-frozen-expiry`, `campaign-flow-reconcile`). A single `INSERT ... SELECT` that inserts only when `schedule_schedules` has no row at all, so a database that already has rows (renamed, disabled or soft deleted seeds included) is never touched. Ids are generated per run (`UUID()`), the name is the stable handle. The down is a no-op.
- `cmd/schedule-manager` runs `Up()` right after the database connection, before the cache and RabbitMQ. Failure stops the start (non-zero exit).
- `schedule-control migrate status|up|down|force <version>` talks to the database only (no Redis or RabbitMQ). Run `up|down|force` only against a local throwaway database. Recovery from a dirty database: stop the crash-looping container, check whether the failed file's statement was applied, then `schedule-control migrate force <N-1 or N>`.
- A new database needs the Alembic stream first and this service afterwards (see [docs/operations.md](docs/operations.md)).
- The sqlite test schema `scripts/database_scripts_test/table_schedule_*.sql` is kept by hand: update it when the schema changes.
- Structure checks: `go test ./migrations/`. Dynamic check on scratch containers: `ALEMBIC_VENV=<venv> scripts/verify-service-migrations.sh bin-schedule-manager` (repo root).

## Request routing

| Pattern | Operations |
|---------|-----------|
| `/v1/schedules$` | POST (create; validates cron, method whitelist, `target_queue` allowlist, active-name uniqueness) |
| `/v1/schedules?(.*)$` | GET (list with filters/pagination) |
| `/v1/schedules/<uuid>$` | GET, PUT (cron change ⇒ `tm_next_run=NULL`), DELETE (soft) |
| `/v1/schedules/<uuid>/execute$` | POST (manual fire-now; never consumes the cron slot) |
| `/v1/executions?(.*)$` | GET (audit trail) |
| `/v1/executions/prune$` | POST (internal; invoked by the `execution-retention` schedule) |
| `/v1/backups$` | POST (internal; invoked by the `database-backup` schedule) |

## schedule-control CLI

Direct DB/cache access, no RabbitMQ — `schedule disable` stops a misbehaving dispatch loop even when the broker is down. Name arguments resolve in the platform (nil-customer) namespace; UUIDs resolve by id.

```bash
./bin/schedule-control schedule list
./bin/schedule-control schedule get number-renew
./bin/schedule-control schedule disable number-renew
./bin/schedule-control schedule enable number-renew
./bin/schedule-control execution list --schedule-id <uuid>
./bin/schedule-control migrate status
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
| `PROMETHEUS_ENDPOINT` | Metrics path | `/metrics` |
| `PROMETHEUS_LISTEN_ADDRESS` | Metrics listen address | `:2112` |
| `SCHEDULE_TICK_INTERVAL_SEC` | Dispatch loop scan cadence (seconds) | `10` |
| `SCHEDULE_DISPATCH_CONCURRENCY` | Max in-flight dispatches per replica | `10` |
| `SCHEDULE_EXECUTION_RETENTION_DAYS` | Execution-row retention applied by `/v1/executions/prune` | `90` |
| `SCHEDULE_BACKUP_DIR` | Backup dump directory. **No default — deliberately.** A default would let a surface that forgot to mount the shared volume "succeed" into ephemeral container disk (silent backup loss). Unset ⇒ the backup job fails loudly; nothing else breaks. | (none) |
| `SCHEDULE_BACKUP_RETENTION_COUNT` | Newest backup files to keep | `7` |

## CRITICAL: Runtime image deviation (debian-slim + mariadb-client)

The fleet standard runtime image is distroless static. This service's runtime stage is **`debian:bookworm-slim` + `default-mysql-client`(→ mariadb-client) from Debian's own repo** — a sanctioned deviation (design §7, same status as bin-call-manager's raw-SQL exception):

- The scheduled DB backup executes `mariadb-dump` as a subprocess, which needs a binary-carrying base image.
- The platform DB is MariaDB 12.3 LTS (VOIP-1386). The client was originally kept at mysqldump 8.0 through the cutover with `--column-statistics=0`/`--set-gtid-purged=OFF` as MySQL-8-only compatibility shims (VOIP-1387); VOIP-1397 completed the switch to `mariadb-dump` once the cutover stabilized, dropping both flags (mariadb-dump doesn't recognize either) and installing the client straight from Debian bookworm's own repository — no external repo or GPG key needed, unlike the `mysql-community-client` package this replaced.
- `mysqldump` still works as an invocation name (Debian's `default-mysql-client` ships it as a compat symlink to the real `mariadb-dump` binary), but the code and this Dockerfile call the binary `mariadb-dump` directly for clarity.

Do NOT "fix" the Dockerfile back to distroless — the mysqldump/mariadb-dump subprocess requirement is a real, permanent constraint, not a stopgap.

## CRITICAL: DB-driven `target_queue` — sanctioned exception to two conventions

The dispatch engine reads `target_queue` as a **string from the database** and converts it to `commonoutline.QueueName` at the `SendRequest` call site. This is a validated exception to:

1. **`bin-common-handler/CLAUDE.md`'s "no free-form queue strings" rule**, and
2. **`docs/conventions/rpc.md` §9.1's "use typed request methods, never construct raw RPC requests" rule** — a generic dispatcher is definitionally untyped.

The invariant is kept equivalent by CRUD-time validation: `POST/PUT /v1/schedules` rejects any `target_queue` that does not match the `commonoutline.QueueNameRequestAll()` enumeration (turning a typo into an immediate 400 instead of a blocked dispatch slot). Do not copy this pattern into other services, and do not weaken the create/update-time allowlist check.

## CRITICAL: At-most-once / Forbid semantics

The engine guarantees **at-most-once per slot** (fire-and-record, not at-least-once):

- Claim = Redis try-once lock (contention filter) + DB CAS on `tm_next_run` + execution-row insert in one transaction. A replica that loses the CAS skips (`skipped_cas`); a lock miss skips (`skipped_lock`).
- **Forbid overlap**: a schedule never runs concurrently with itself; the slot is late, not lost. Repeated overlap skips of the same in-flight run are counted once per slot (`skipped_overlap`).
- **Catch-up is single-fire**: after downtime, one run fires and `tm_next_run` advances from now — missed slots are not replayed.
- **Manual execute never consumes the cron slot**: `/v1/schedules/<id>/execute` touches neither `tm_next_run` nor `tm_last_run`, and works on disabled schedules. Cron and manual runs mutually exclude via the same lock + overlap guard (409 while in flight).
- Executions past `tm_deadline` are reaped to `abandoned` (kill -9 recovery) — the run is recorded as lost, never silently retried.
- **Shutdown is abandon-not-drain, intentionally**: the daemon cancels the dispatch loop context and exits without joining in-flight dispatch goroutines. An in-flight run that dies with the process is exactly the kill -9 case — the reaper marks it `abandoned` and the next slot proceeds. Do not add a graceful drain that delays shutdown; at-most-once semantics make it unnecessary.

Full semantics with rationale: [docs/plans/2026-08-01-bin-schedule-manager-design.md](../docs/plans/2026-08-01-bin-schedule-manager-design.md) §5. Do not add automatic re-dispatch of failed/abandoned runs without going through a design review — destructive jobs (e.g. `number-renew`) depend on at-most-once.

## Further reading

- [docs/architecture.md](docs/architecture.md)
- [docs/domain.md](docs/domain.md)
- [docs/dependencies.md](docs/dependencies.md)
- [docs/operations.md](docs/operations.md)
- Design: [docs/plans/2026-08-01-bin-schedule-manager-design.md](../docs/plans/2026-08-01-bin-schedule-manager-design.md)
