# bin-schedule-manager

Platform-internal cron scheduler for VoIPbin. Stores schedules in MySQL, scans them on a tick loop, and dispatches due jobs as RabbitMQ RPC requests to other services — replacing external cron surfaces (Kubernetes CronJobs, per-service tickers) with one engine, one audit trail, and one set of metrics. Runs with two replicas; Redis locks plus a DB compare-and-swap claim guarantee at-most-once dispatch per slot.

## Key Concepts

- **Schedule**: A named, platform-owned (nil-customer in Phase 1) cron entry: 5-field UTC cron expression plus an RPC target (`target_queue`/`target_uri`/`target_method`/payload), timeout, retry budget, and enabled flag.
- **Execution**: The audit row recorded for every run (cron or manual): status `running → success | failed | abandoned`, attempt count, duration, RPC result. Pruned by the seeded `execution-retention` schedule.
- **Claim mechanics**: Every replica scans due schedules each tick. To fire a slot, a replica takes a try-once Redis lock (`schedule:lock:<id>`), checks that no run of the schedule is in flight (Forbid overlap), then atomically CAS-advances `tm_next_run` and inserts the execution row in one DB transaction. Losing the lock or the CAS means another replica owns the slot — skip, never double-fire. Runs that outlive their deadline (e.g. kill -9 mid-run) are reaped to `abandoned`, not retried.

## Public RPC Entrypoints

| Pattern | Operations |
|---------|-----------|
| `POST /v1/schedules` | Create schedule (validates cron, method, target queue allowlist, name uniqueness) |
| `GET /v1/schedules` | List schedules (filters, pagination) |
| `GET /v1/schedules/<id>` | Get schedule |
| `PUT /v1/schedules/<id>` | Update schedule (cron change resets `tm_next_run`) |
| `DELETE /v1/schedules/<id>` | Soft-delete schedule |
| `POST /v1/schedules/<id>/execute` | Manual fire-now (never consumes the cron slot) |
| `GET /v1/executions` | Execution audit trail |
| `POST /v1/executions/prune` | Retention pruning (internal; fired by the `execution-retention` schedule) |
| `POST /v1/backups` | Database backup via mysqldump (internal; fired by the `database-backup` schedule) |

## Dependencies

- **MySQL** — `schedule_schedules`, `schedule_executions` (soft-delete via `tm_delete`); also the backup target for mysqldump
- **Redis** — per-schedule claim locks (redsync)
- **RabbitMQ** — listen queue `bin-manager.schedule-manager.request`; subscribes to `bin-manager.customer-manager.event`; publishes internal events on `bin-manager.schedule-manager.event`

## Local Development

```bash
# Build
cd bin-schedule-manager
go build -o ./bin/ ./cmd/...

# Run all tests
go test ./...

# Verify before commit (mandatory)
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

# CLI tool (direct DB/cache, bypasses RabbitMQ)
./bin/schedule-control schedule list
./bin/schedule-control schedule get number-renew
./bin/schedule-control schedule disable number-renew
./bin/schedule-control execution list --schedule-id <uuid>
```

## Further Reading

- [docs/architecture.md](docs/architecture.md)
- [docs/domain.md](docs/domain.md)
- [docs/dependencies.md](docs/dependencies.md)
- [docs/operations.md](docs/operations.md)

# Deploy

`bin-schedule-manager-build` pushes the image, and a single `build-approval` gate covers
the whole pipeline (test -> build -> deploy) through to production.
`bin-schedule-manager-deploy` runs after `bin-schedule-manager-build`, bumping this service's pin on
bm-nyc-01 and recreating the container. See
`.circleci/scripts/ssh-deploy.sh` (this pattern was piloted with
bin-call-manager). The previous GKE deploy path for this service has been
removed.
