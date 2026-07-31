# bin-scheduler-manager Operations

## Common Failure Modes

| Symptom | Likely Cause | Resolution |
|---------|-------------|-----------|
| Schedule not firing | Disabled (`enabled=0`), soft-deleted, `tm_next_run` stuck NULL, or cron expression never matches | `scheduler-control schedule get <name>` — check `enabled`, `tm_delete`, `tm_next_run`, `cron`; `schedule_lag_seconds` gauge shows overdue schedules; re-enable or fix cron via API/CLI |
| Suspected double-fire | Almost always a false alarm: both replicas scanned the slot and one skipped | Check `dispatch_total{result="skipped_cas"}` / `skipped_lock` — skips there mean the claim machinery worked. A real double-fire would need two execution rows with the same `(schedule_id, trigger_type, tm_scheduled)`, which the unique key rejects; query `scheduler_executions` for the slot to confirm |
| Backup job failing | `SCHEDULER_BACKUP_DIR` unset (it has NO default — deliberate), or the directory is not a mounted shared volume / not writable | Set `SCHEDULER_BACKUP_DIR` and mount a **ReadWriteMany** volume shared by all replicas (any replica may run the job). An RWO PVC is a misconfiguration: the second replica sits Pending. Check the execution row's `error` and `backup_last_bytes` (0 or stale = investigate) |
| `execution_rows` gauge growing without bound | `execution-retention` schedule disabled/deleted, or `/v1/executions/prune` failing | `scheduler-control schedule get execution-retention` — confirm it exists, is enabled, and its last executions succeeded (`execution list --schedule-id <id>`); re-enable or fire manually via `/v1/schedules/<id>/execute` |
| High `skipped_lock` rate / lock contention | Normal with 2 replicas scanning the same due set; abnormal spikes suggest Redis latency or a very short tick interval | Expected in steady state (one replica wins, one skips). If both replicas report failures acquiring locks (errors, not skips), check Redis health and `REDIS_ADDRESS` |
| Runs ending `abandoned` | Replica killed mid-run (kill -9 / OOM / node loss), or job genuinely exceeded its deadline | Expected recovery path: the reaper records the loss and the next cron slot fires normally. Repeated abandonment of the same schedule ⇒ raise `timeout_ms` or investigate the target service |

## Debugging Guide

```bash
# Pod logs (dispatch decisions log the schedule name and execution UUID at
# dispatch start/end — that UUID is the correlation handle to downstream logs)
kubectl logs -n bin-manager -l app=scheduler-manager --tail=200

# Grep patterns
kubectl logs -n bin-manager -l app=scheduler-manager | grep -i "claimAndDispatch"
kubectl logs -n bin-manager -l app=scheduler-manager | grep -i "reapAbandoned"

# Admin CLI (direct DB/cache — works even when RabbitMQ is down)
./bin/scheduler-control schedule list
./bin/scheduler-control schedule get number-renew
./bin/scheduler-control schedule disable number-renew   # emergency stop for one schedule
./bin/scheduler-control execution list --schedule-id <uuid>

# Emergency stop for ALL firing: scale to zero (nothing else degrades)
kubectl scale deployment/scheduler-manager -n bin-manager --replicas=0

# Build / test / lint
cd bin-scheduler-manager && go build -o ./bin/ ./cmd/...
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
| `database_dsn` | `DATABASE_DSN` | (required) | MySQL connection DSN (also the mysqldump target) |
| `redis_address` | `REDIS_ADDRESS` | (required) | Redis server address |
| `redis_password` | `REDIS_PASSWORD` | empty | Redis password |
| `redis_database` | `REDIS_DATABASE` | `1` | Redis logical database index |
| `scheduler_tick_interval_sec` | `SCHEDULER_TICK_INTERVAL_SEC` | `10` | Dispatch loop scan cadence in seconds |
| `scheduler_dispatch_concurrency` | `SCHEDULER_DISPATCH_CONCURRENCY` | `10` | Max in-flight dispatches per replica |
| `scheduler_execution_retention_days` | `SCHEDULER_EXECUTION_RETENTION_DAYS` | `90` | Age in days after which `/v1/executions/prune` deletes execution rows |
| `scheduler_backup_dir` | `SCHEDULER_BACKUP_DIR` | (none — must be set explicitly) | Backup dump directory. No default by design: a default would let a surface that forgot to mount the shared volume "succeed" into ephemeral disk. Unset ⇒ backup fails loudly with an actionable error. Must be one ReadWriteMany volume shared by all replicas |
| `scheduler_backup_retention_count` | `SCHEDULER_BACKUP_RETENTION_COUNT` | `7` | Number of newest backup files to keep |

## Prometheus Metrics

Namespace: `scheduler_manager` (design §5.7).

| Metric Name | Type | Labels | Description |
|-------------|------|--------|-------------|
| `scheduler_manager_dispatch_total` | Counter | `schedule_name`, `result` | Dispatch outcomes: `success`, `failed`, `abandoned` (counted from the bulk reap with an empty `schedule_name` label), `skipped_lock`, `skipped_cas`, `skipped_overlap` (incremented once per skipped slot, not per tick) |
| `scheduler_manager_dispatch_duration_ms` | Histogram | `schedule_name` | Dispatch wall time in milliseconds |
| `scheduler_manager_schedule_last_success_timestamp` | Gauge | `schedule_name` | Unix timestamp of the last successful dispatch (the doctor/alerting hook) |
| `scheduler_manager_schedule_lag_seconds` | Gauge | `schedule_name` | Seconds overdue (now − `tm_next_run`) at scan time |
| `scheduler_manager_execution_rows` | Gauge | — | Total `scheduler_executions` row count, refreshed once per tick (never per scrape); alarms if `execution-retention` stops working |
| `scheduler_manager_backup_last_bytes` | Gauge | — | Size in bytes of the most recent successful backup dump; a doctor check flags 0/missing |
| `scheduler_manager_receive_request_process_time` | Histogram | `type`, `method` | Standard listenhandler RPC processing time |
| `scheduler_manager_receive_subscribe_event_process_time` | Histogram | `publisher`, `type` | Subscribe event processing time |

**Per-replica gauge semantics — aggregate with `max by (schedule_name)`.** Each replica reports its own gauge values; the replica that did not perform a dispatch reports stale or absent values. Alert rules on `schedule_last_success_timestamp` and `schedule_lag_seconds` MUST aggregate with `max by (schedule_name)` across replicas — never `min` or a single series — or a healthy schedule will look stale from the losing replica's series. (Recorded here so the first VOIP-1280 alert rule is not written wrong.)

Label-cardinality note: `schedule_name` labels are acceptable while all schedules are platform-owned (single-digit count). Phase 3 customer schedules will NOT be per-schedule labeled.

## Backup Runbook — dump-and-restore smoke test

Verifies that a dump produced by the backup job actually restores. Run against a scratch MySQL 8.0 (never production). Automated as part of VOIP-1281's sandbox onboarding; manual procedure:

```bash
# 0. Fire the backup now (or take the newest existing dump)
#    via RPC: POST /v1/backups on bin-manager.scheduler-manager.request,
#    or manual-execute the database-backup schedule:
#    POST /v1/schedules/<database-backup-id>/execute
DUMP=$(ls -t "$SCHEDULER_BACKUP_DIR"/voipbin-*.sql.gz | head -1)
echo "testing: $DUMP"

# 1. Scratch MySQL 8.0
docker run -d --name backup-smoke -e MYSQL_ROOT_PASSWORD=smoke \
  -p 33061:3306 mysql:8.0
until docker exec backup-smoke mysqladmin ping -uroot -psmoke --silent; do sleep 2; done

# 2. Restore
gunzip -c "$DUMP" | docker exec -i backup-smoke mysql -uroot -psmoke

# 3. Diff table counts against the source (spot-check the scheduler tables
#    plus a few high-traffic tables)
docker exec backup-smoke mysql -uroot -psmoke -e \
  "SELECT table_schema, COUNT(*) AS tables FROM information_schema.tables
   WHERE table_schema NOT IN ('mysql','sys','information_schema','performance_schema')
   GROUP BY table_schema;"
docker exec backup-smoke mysql -uroot -psmoke -e \
  "SELECT COUNT(*) FROM voipbin.scheduler_schedules; SELECT COUNT(*) FROM voipbin.scheduler_executions;"
# Compare with the same queries on the source DB — counts must match the dump time.

# 4. Cleanup
docker rm -f backup-smoke
```

Pass criteria: restore completes without error; table sets and row counts match the source at dump time; the execution row for the backup run recorded `{"path", "bytes"}` in `result` and `backup_last_bytes` equals the file size.

Credential hygiene reminder: the backup passes the DB password via a 0600 `--defaults-extra-file` temp file (never argv, never `MYSQL_PWD`); error logs are redacted. Do not "simplify" this when touching `pkg/backuphandler`.
