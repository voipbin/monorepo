# bin-scheduler-manager

Platform-internal cron scheduler (VOIP-1283). Absorbs external cron surfaces (the number-manager Kubernetes CronJob, future ticker migrations) into one DB-driven dispatch engine with an audit trail, at-most-once claim semantics, and Prometheus visibility. Phase 1 schedules are all platform-owned (nil customer); there is no user-visible API surface yet.

> Cross-cutting rules (verification workflow, branch/commit format, worktrees, Alembic, RST sync) live in the root [CLAUDE.md](../CLAUDE.md). This file covers only what is specific to `bin-scheduler-manager`.

## Key facts

- **MySQL** tables `scheduler_schedules` / `scheduler_executions` (audit trail), **Redis** claim locks via redsync (`scheduler:lock:<id>`).
- **RabbitMQ queue**: `bin-manager.scheduler-manager.request`
- **Subscribes to**: `bin-manager.customer-manager.event` (`customer_deleted` → delete the customer's schedules)
- **Publishes**: internal events only on `bin-manager.scheduler-manager.event` (`schedule_created/updated/deleted`, `execution_succeeded/failed`). **No customer webhooks in Phase 1** — every schedule is nil-customer, so there is deliberately no `models/schedule/webhook.go` and no RST struct docs (Phase 3 scope).
- **replicas: 2** by design — every replica runs the same tick loop; Redis lock + DB CAS claim make concurrent replicas safe (kill -9 failover without double-fire).
- **Housekeeping dogfoods the engine**: `execution-retention` and `database-backup` are seeded schedules that target the service's own request queue (`/v1/executions/prune`, `/v1/backups`).

## Package layout

| Package | Role |
|---------|------|
| `cmd/scheduler-manager` | Daemon entry point (cobra; Bootstrap/LoadGlobalConfig) |
| `cmd/scheduler-control` | Admin CLI (direct DB/cache, no RabbitMQ — works when the broker path is unhealthy) |
| `internal/config` | Viper/pflag config binding |
| `models/schedule` | Schedule struct, cron helpers, filters, internal event types |
| `models/execution` | Execution struct (audit row), status machine, filters |
| `pkg/listenhandler` | RabbitMQ RPC router (regex dispatch) |
| `pkg/subscribehandler` | Event consumer (`customer_deleted`) |
| `pkg/schedulehandler` | Schedule CRUD, validation, next-run computation, name uniqueness |
| `pkg/dispatchhandler` | Tick loop, claim (lock + CAS), dispatch, record, reaper, manual execute |
| `pkg/backuphandler` | mysqldump subprocess + gzip + retention pruning (design §7) |
| `pkg/dbhandler` | MySQL via squirrel; sqlite test harness |
| `pkg/cachehandler` | Redis + redsync claim locks |

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

## scheduler-control CLI

Direct DB/cache access, no RabbitMQ — `schedule disable` stops a misbehaving dispatch loop even when the broker is down. Name arguments resolve in the platform (nil-customer) namespace; UUIDs resolve by id.

```bash
./bin/scheduler-control schedule list
./bin/scheduler-control schedule get number-renew
./bin/scheduler-control schedule disable number-renew
./bin/scheduler-control schedule enable number-renew
./bin/scheduler-control execution list --schedule-id <uuid>
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
| `SCHEDULER_TICK_INTERVAL_SEC` | Dispatch loop scan cadence (seconds) | `10` |
| `SCHEDULER_DISPATCH_CONCURRENCY` | Max in-flight dispatches per replica | `10` |
| `SCHEDULER_EXECUTION_RETENTION_DAYS` | Execution-row retention applied by `/v1/executions/prune` | `90` |
| `SCHEDULER_BACKUP_DIR` | Backup dump directory. **No default — deliberately.** A default would let a surface that forgot to mount the shared volume "succeed" into ephemeral container disk (silent backup loss). Unset ⇒ the backup job fails loudly; nothing else breaks. | (none) |
| `SCHEDULER_BACKUP_RETENTION_COUNT` | Newest backup files to keep | `7` |

## CRITICAL: Runtime image deviation (debian-slim + mysql-community-client)

The fleet standard runtime image is distroless static. This service's runtime stage is **`debian:bookworm-slim` + `mysql-community-client` 8.0 from MySQL's official APT repo** — a sanctioned deviation (design §7, same status as bin-call-manager's raw-SQL exception):

- The scheduled DB backup executes `mysqldump` as a subprocess, which needs a binary-carrying base image.
- The client must be **genuine MySQL 8.0**, not MariaDB: alpine's `mysql-client` and debian's `default-mysql-client` are MariaDB aliases, an unsupported pairing against the MySQL 8.0 platform DB (`caching_sha2_password` auth, incompatible dump output).
- Client pin follows the platform DB server version. When the platform migrates to MySQL 8.4/9.x, bumping this package is part of that migration's checklist.

Do NOT "fix" the Dockerfile back to distroless or swap in a MariaDB client.

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

Full semantics with rationale: [docs/plans/2026-08-01-bin-scheduler-manager-design.md](../docs/plans/2026-08-01-bin-scheduler-manager-design.md) §5. Do not add automatic re-dispatch of failed/abandoned runs without going through a design review — destructive jobs (e.g. `number-renew`) depend on at-most-once.

## Further reading

- [docs/architecture.md](docs/architecture.md)
- [docs/domain.md](docs/domain.md)
- [docs/dependencies.md](docs/dependencies.md)
- [docs/operations.md](docs/operations.md)
- Design: [docs/plans/2026-08-01-bin-scheduler-manager-design.md](../docs/plans/2026-08-01-bin-scheduler-manager-design.md)
