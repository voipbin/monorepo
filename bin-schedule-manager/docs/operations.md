# bin-schedule-manager Operations

## Common Failure Modes

| Symptom | Likely Cause | Resolution |
|---------|-------------|-----------|
| Schedule not firing | Disabled (`enabled=0`), soft-deleted, `tm_next_run` stuck NULL, or cron expression never matches | `schedule-control schedule get <name>` — check `enabled`, `tm_delete`, `tm_next_run`, `cron`; `schedule_lag_seconds` gauge shows overdue schedules; re-enable or fix cron via API/CLI |
| Suspected double-fire | Almost always a false alarm: both replicas scanned the slot and one skipped | Check `dispatch_total{result="skipped_cas"}` / `skipped_lock` — skips there mean the claim machinery worked. A real double-fire would need two execution rows with the same `(schedule_id, trigger_type, tm_scheduled)`, which the unique key rejects; query `schedule_executions` for the slot to confirm |
| Backup job failing | `SCHEDULE_BACKUP_DIR` unset (it has NO default — deliberate), or the directory is not a mounted shared volume / not writable | Set `SCHEDULE_BACKUP_DIR` and mount a **ReadWriteMany** volume shared by all replicas (any replica may run the job). An RWO PVC is a misconfiguration: the second replica sits Pending. Check the execution row's `error` and `backup_last_bytes` (0 or stale = investigate) |
| `execution_rows` gauge growing without bound | `execution-retention` schedule disabled/deleted, or `/v1/executions/prune` failing | `schedule-control schedule get execution-retention` — confirm it exists, is enabled, and its last executions succeeded (`execution list --schedule-id <id>`); re-enable or fire manually via `/v1/schedules/<id>/execute` |
| High `skipped_lock` rate / lock contention | Normal with 2 replicas scanning the same due set; abnormal spikes suggest Redis latency or a very short tick interval | Expected in steady state (one replica wins, one skips). If both replicas report failures acquiring locks (errors, not skips), check Redis health and `REDIS_ADDRESS` |
| No schedules firing at all (no skips, no claims) | Redis unavailable — a non-busy lock error fails the claim closed, by design: at-most-once must not degrade to lockless firing | **Redis is a hard availability dependency of dispatch.** Restore Redis; firing resumes on the next tick with no catch-up backfill (single-fire catch-up per design §5.4) |
| Runs ending `abandoned` | Replica killed mid-run (kill -9 / OOM / node loss), or job genuinely exceeded its deadline | Expected recovery path: the reaper records the loss and the next cron slot fires normally. Repeated abandonment of the same schedule ⇒ raise `timeout_ms` or investigate the target service |

## Debugging Guide

```bash
# Pod logs (dispatch decisions log the schedule name and execution UUID at
# dispatch start/end — that UUID is the correlation handle to downstream logs)
kubectl logs -n bin-manager -l app=schedule-manager --tail=200

# Grep patterns
kubectl logs -n bin-manager -l app=schedule-manager | grep -i "claimAndDispatch"
kubectl logs -n bin-manager -l app=schedule-manager | grep -i "reapAbandoned"

# Admin CLI (direct DB/cache — works even when RabbitMQ is down)
./bin/schedule-control schedule list
./bin/schedule-control schedule get number-renew
./bin/schedule-control schedule disable number-renew   # emergency stop for one schedule
./bin/schedule-control execution list --schedule-id <uuid>

# Emergency stop for ALL firing: scale to zero (nothing else degrades)
kubectl scale deployment/schedule-manager -n bin-manager --replicas=0

# Build / test / lint
cd bin-schedule-manager && go build -o ./bin/ ./cmd/...
go test ./...
golangci-lint run -v --timeout 5m
go generate ./...
```

Tracing a run: find the execution row (`execution list` or `GET /v1/executions?filters=schedule_id:<id>`) → `status`, `status_code`, `error`, `attempt_count`, `duration_ms` tell the story; the execution UUID appears in the scheduler's dispatch start/end log lines (RequestID is not propagated on this path — the UUID log correlation is the design).

## Configuration

| Flag | Env Var | Default | Description |
|------|---------|---------|-------------|
| `rabbitmq_address` | `RABBITMQ_ADDRESS` | (required) | RabbitMQ server address |
| `prometheus_endpoint` | `PROMETHEUS_ENDPOINT` | `/metrics` | Prometheus metrics endpoint path |
| `prometheus_listen_address` | `PROMETHEUS_LISTEN_ADDRESS` | `:2112` | Prometheus metrics listen address |
| `database_dsn` | `DATABASE_DSN` | (required) | MySQL connection DSN (also the mariadb-dump target) |
| `redis_address` | `REDIS_ADDRESS` | (required) | Redis server address |
| `redis_password` | `REDIS_PASSWORD` | empty | Redis password |
| `redis_database` | `REDIS_DATABASE` | `1` | Redis logical database index |
| `schedule_tick_interval_sec` | `SCHEDULE_TICK_INTERVAL_SEC` | `10` | Dispatch loop scan cadence in seconds |
| `schedule_dispatch_concurrency` | `SCHEDULE_DISPATCH_CONCURRENCY` | `10` | Max in-flight dispatches per replica |
| `schedule_execution_retention_days` | `SCHEDULE_EXECUTION_RETENTION_DAYS` | `90` | Age in days after which `/v1/executions/prune` deletes execution rows |
| `schedule_backup_dir` | `SCHEDULE_BACKUP_DIR` | (none — must be set explicitly) | Backup dump directory. No default by design: a default would let a surface that forgot to mount the shared volume "succeed" into ephemeral disk. Unset ⇒ backup fails loudly with an actionable error. Must be one ReadWriteMany volume shared by all replicas |
| `schedule_backup_retention_count` | `SCHEDULE_BACKUP_RETENTION_COUNT` | `7` | Number of newest backup files to keep |

## Prometheus Metrics

Namespace: `schedule_manager` (design §5.7).

| Metric Name | Type | Labels | Description |
|-------------|------|--------|-------------|
| `schedule_manager_dispatch_total` | Counter | `schedule_name`, `result` | Dispatch outcomes: `success`, `failed`, `abandoned` (counted from the bulk reap with an empty `schedule_name` label), `skipped_lock`, `skipped_cas`, `skipped_overlap` (incremented once per skipped slot, not per tick) |
| `schedule_manager_dispatch_duration_ms` | Histogram | `schedule_name` | Dispatch wall time in milliseconds |
| `schedule_manager_schedule_last_success_timestamp` | Gauge | `schedule_name` | Unix timestamp of the last successful dispatch (the doctor/alerting hook) |
| `schedule_manager_schedule_lag_seconds` | Gauge | `schedule_name` | Seconds overdue (now − `tm_next_run`) at scan time |
| `schedule_manager_execution_rows` | Gauge | — | Total `schedule_executions` row count, refreshed once per tick (never per scrape); alarms if `execution-retention` stops working |
| `schedule_manager_backup_last_bytes` | Gauge | — | Size in bytes of the most recent successful backup dump; a doctor check flags 0/missing |
| `schedule_manager_receive_request_process_time` | Histogram | `type`, `method` | Standard listenhandler RPC processing time |
| `schedule_manager_receive_subscribe_event_process_time` | Histogram | `publisher`, `type` | Subscribe event processing time |

**Per-replica gauge semantics — aggregate with `max by (schedule_name)`.** Each replica reports its own gauge values; the replica that did not perform a dispatch reports stale or absent values. Alert rules on `schedule_last_success_timestamp` and `schedule_lag_seconds` MUST aggregate with `max by (schedule_name)` across replicas — never `min` or a single series — or a healthy schedule will look stale from the losing replica's series. (Recorded here so the first VOIP-1280 alert rule is not written wrong.)

Label-cardinality note: `schedule_name` labels are acceptable while all schedules are platform-owned (single-digit count). Phase 3 customer schedules will NOT be per-schedule labeled.

## Backup Runbook — dump-and-restore smoke test

Verifies that a dump produced by the backup job actually restores. Run against a scratch MariaDB 12.3 (never production) — matches the platform DB engine since VOIP-1386, and the client since VOIP-1397. Automated as part of VOIP-1281's sandbox onboarding; manual procedure:

```bash
# 0. Fire the backup now (or take the newest existing dump)
#    via RPC: POST /v1/backups on bin-manager.schedule-manager.request,
#    or manual-execute the database-backup schedule:
#    POST /v1/schedules/<database-backup-id>/execute
DUMP=$(ls -t "$SCHEDULE_BACKUP_DIR"/voipbin-*.sql.gz | head -1)
echo "testing: $DUMP"

# 1. Scratch MariaDB 12.3
docker run -d --name backup-smoke -e MARIADB_ROOT_PASSWORD=smoke \
  -p 33061:3306 mariadb:12.3
until docker exec backup-smoke mariadb-admin ping -uroot -psmoke --silent; do sleep 2; done

# 2. Restore
gunzip -c "$DUMP" | docker exec -i backup-smoke mariadb -uroot -psmoke

# 3. Diff table counts against the source (spot-check the schedule tables
#    plus a few high-traffic tables)
docker exec backup-smoke mariadb -uroot -psmoke -e \
  "SELECT table_schema, COUNT(*) AS tables FROM information_schema.tables
   WHERE table_schema NOT IN ('mysql','sys','information_schema','performance_schema')
   GROUP BY table_schema;"
docker exec backup-smoke mariadb -uroot -psmoke -e \
  "SELECT COUNT(*) FROM voipbin.schedule_schedules; SELECT COUNT(*) FROM voipbin.schedule_executions;"
# Compare with the same queries on the source DB — counts must match the dump time.

# 4. Cleanup
docker rm -f backup-smoke
```

Pass criteria: restore completes without error; table sets and row counts match the source at dump time; the execution row for the backup run recorded `{"path", "bytes"}` in `result` and `backup_last_bytes` equals the file size.

Credential hygiene reminder: the backup passes the DB password via a 0600 `--defaults-extra-file` temp file (never argv, never `MYSQL_PWD`); error logs are redacted. Do not "simplify" this when touching `pkg/backuphandler`.

## Deployment (Komodo)

Komodo-managed (VOIP-1350), same mechanism as the other `bin-*-manager` services
(see bin-call-manager for the original pattern). Deployed via
`.circleci/scripts/render-image-tag.sh` + `.circleci/scripts/komodo-api-deploy.sh`
from `komodo/docker-compose.yml`.

Three deviations from the Tier 1/2 template, all intentional:
- **Non-distroless runtime** (`debian:bookworm-slim` + `default-mysql-client`
  (→ mariadb-client, VOIP-1397), see the CRITICAL note in
  [../CLAUDE.md](../CLAUDE.md)) — but `debian:bookworm-slim` ships with
  neither `wget` nor `curl` by default, so the fleet-standard `wget`
  healthcheck silently fails on this image (confirmed live: the pre-Komodo
  `install/` container had been reporting Docker-unhealthy for 5 days with
  `exec: "wget": executable file not found`). The Dockerfile keeps `curl`
  in the runtime stage instead of purging it, and the Komodo compose
  healthcheck uses `curl -fsS` against `/metrics` instead of `wget`.
- **Bind-mounted backup volume**: `SCHEDULE_BACKUP_DIR=/backups` is mounted from
  the host path `/opt/voipbin/install/backups/scheduled-db` on bm-nyc-01
  (confirmed via `docker inspect` against the pre-cutover container), so
  existing backup history is preserved across the cutover rather than starting
  a new directory under Komodo's stack working dir.
- **`deploy: replicas: 2`, no `container_name`** (NOJIRA-Schedule-manager-scale-poc):
  this service is the fleet's pilot for the bin-*-manager 1→2 replica
  rollout. Compose rejects a fixed `container_name` together with
  `deploy.replicas > 1` ("container name must be unique"), so
  `container_name: voipbin-schedule-manager` was removed; Komodo is
  expected to name the containers from the compose project/service (e.g.
  `bin-schedule-manager-schedule-manager-1`/`-2`) — **not yet confirmed
  against a live deploy on bm-nyc-01** (`komodo-api-deploy.sh` itself notes
  its container-state read is unverified against the live instance). Update
  this once the PoC actually deploys.
  - **Backup path is the one genuinely new surface** (dispatch-loop replica
    safety already ran on GKE; the bind-mounted backup path did not). It
    remains safe: the mount is a host bind mount on a single bare-metal
    node, which is exactly the ReadWriteMany semantics this doc already
    requires ("any replica may run the job"), and double-execution is
    prevented at two layers — `bin-manager.schedule-manager.request` is a
    shared competing-consumer queue (only one replica receives the RPC),
    and the design §5.3 claim (Redis lock + DB CAS + unique execution row)
    prevents double-dispatch. `pkg/backuphandler` itself has no file-level
    lock (writes straight to `outPath`, prunes by directory listing) — this
    is fine only as long as the claim machinery above keeps two replicas
    from running the backup job concurrently; do not weaken that guarantee
    without revisiting this note.
  - **CI deploy gate has a known blind spot with replicas > 1**:
    `.circleci/scripts/komodo-api-deploy.sh`'s `check_stack_running` reads
    `.container.state` per compose *service*, not per container. With 2
    replicas it can report healthy on a 1-of-2 state — a green deploy does
    not by itself prove both replicas are up. **Manual verification is
    required after every deploy of this service until the gate is fixed
    fleet-wide:**
    ```bash
    # On bm-nyc-01: expect exactly 2 running containers
    docker ps --filter name=schedule-manager --format '{{.Names}}\t{{.Status}}'
    ```
    ```promql
    # In Prometheus: expect 2 up series
    count(up{job="voipbin-managers", service="schedule-manager"})
    ```
  - **Alerting blind spot**: `InstanceDown` (`up == 0`) cannot catch a lost
    replica under DNS SD (the target goes stale, not to 0), and
    `ManagerServiceGone` only fires at zero surviving replicas — a 1-of-2
    degraded state is silently invisible today. No `count by (service)(up{...}) < 2`-style
    alert exists yet; tracked as a follow-up for the fleet rollout, not
    blocking this PoC.
  - **Runbook naming drift**: alert runbooks in `monorepo-etc`'s
    `alert-rules.yml` assume containers are named `voipbin-<service>` (used
    for `docker ps --filter name=...` substring matching). This service no
    longer has that name. The substring filter still happens to match
    (`schedule-manager` is contained in the new Komodo-generated names), so
    nothing breaks operationally, but this is an undocumented exception
    that will multiply as more services join the rollout — flag it in the
    fleet rollout design doc rather than fixing per-service.
