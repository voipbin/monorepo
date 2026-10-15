# Operations: bin-flow-manager

## Common Failure Modes

| Symptom | Likely Cause | Resolution |
|---------|--------------|------------|
| Activeflow stuck in `executing` with no progress | Action dispatch RPC call to downstream service (e.g., call-manager) timed out or returned an error | Check logs for the specific action type; verify the target service (call-manager, tts-manager, etc.) is healthy; check RabbitMQ queue depth |
| `next` endpoint returns error after action completes | The activeflow's `current_action_id` has no successor (last action in the stack) and no stack frame to pop | This is expected at end-of-flow; verify the flow has a terminal action (`hangup`) to avoid callers being left connected |
| Variable substitution returning raw `{{var}}` tokens | Variable was never set (action that should set it failed, or `gather` timed out without input) | Check activeflow variable state; verify gather action received input; check for upstream action errors |
| Activeflow not created when call arrives | Call-manager failed to invoke flow-manager; flow ID not found or customer quota exceeded | Check call-manager logs for RPC errors to flow-manager; verify flow_id exists and belongs to the customer |
| `push_actions` not advancing execution | Pushed stack frame has zero actions, or the activeflow was stopped concurrently | Verify the action list in the push request; check for concurrent `stop` calls; check `activeflow_running` metric |
| High latency on `/execute` or `/next` requests | MySQL slow queries or Redis cache miss on activeflow lookup | Check `receive_request_process_time` histogram; inspect slow query logs; verify Redis connectivity |
| Flow CRUD operations slow | Large action arrays on flows; MySQL table scan on `flows` | Check `flow_crud_total` metric; ensure proper indexing on `customer_id`; consider breaking large flows into sub-flows |

## Schema Migration Failures

Migrations run at startup, before the service connects the cache and the message broker. A failure exits the process with a non-zero code and the container is restarted. Version table: `flow_schema_migrations`.

| Symptom | Likely Cause | Resolution |
|---------|-------------|------------|
| `Dirty database version N. Fix and force version.` | A previous run stopped inside migration N (failure, kill) | Stop the crash-looping container. Check whether migration N's statement was applied. Run `flow-control migrate force <N-1>` if not (for N=1 use `force -- -1`), `force <N>` if so, then start again. The image has no shell, so run the command in a one-off container of the same image: `docker run --rm --entrypoint ./flow-control -e DATABASE_DSN=<dsn> <image> migrate force <N>` (the container must reach the database; quote a DSN that holds special characters) |
| `can't acquire lock` | Another instance is migrating for longer than 10 seconds | The restart policy retries; no action unless it persists |
| `Error 1205 ... Lock wait timeout` | A long transaction holds a metadata lock on a flow table (even `CREATE TABLE IF NOT EXISTS` takes it briefly) | Find the long transaction, then restart. If the database is marked dirty, follow the dirty row |
| `CREATE command denied ... flow_schema_migrations` | The database account cannot create tables | Grant CREATE to the service account |
| `db version N is ahead of this image (max M); skipping migrations` (warning) | An older image started after a newer one migrated | Expected during a rollback; no action |

`flow-control migrate status` shows the applied version, the image version and the dirty flag (it creates the version table when it is missing).

On a new database run the Alembic stream first and start the service afterwards. The reverse order is not supported: if the service creates the tables first, the Alembic migration that renames `activeflows` to `flow_activeflows` (and `flows` to `flow_flows`) fails with `Table 'flow_activeflows' already exists` (error 1050). The service account also needs permission to create tables (for the version table). The database must be created with `CHARACTER SET utf8 COLLATE utf8_general_ci`, like the install scripts do.

## Debugging Guide

### Key Log Patterns

```bash
# Trace all operations for a specific activeflow UUID
kubectl logs -n voipbin deploy/bin-flow-manager | grep <activeflow-uuid>

# Find action dispatch errors
kubectl logs -n voipbin deploy/bin-flow-manager | grep "actionhandler" | grep -i "error\|fail"

# Find execution loop iterations (each action advance)
kubectl logs -n voipbin deploy/bin-flow-manager | grep "activeflowhandler" | grep -E "next|execute|continue"

# Find variable substitution issues
kubectl logs -n voipbin deploy/bin-flow-manager | grep "variablehandler" | grep -i "error\|missing\|not found"
```

### Tracing an Activeflow

1. **Get the activeflow state** by checking logs or querying via API:
   ```bash
   # Via API (requires auth)
   curl -H "Authorization: Bearer <token>" https://api.voipbin.net/v1/activeflows/<uuid>
   ```

2. **Check the current action** — `current_action_id` in the activeflow shows where execution is paused.

3. **Inspect the stack map** — nested stacks appear in `stack_map`; a non-empty stack indicates a sub-flow is executing.

4. **Check for stuck executions** using the `activeflow_running` gauge — if this is unusually high and `activeflow_ended_total` has stopped increasing, flows may be stuck.

5. **Verify downstream services** — use `action_error_total` metric to identify which action types are failing most often.

### Metrics Checklist

```bash
# Rate of new activeflows created
kubectl exec -n voipbin deploy/bin-flow-manager -- curl -s localhost:9090/metrics | grep activeflow_created_total

# Count of currently running activeflows
kubectl exec -n voipbin deploy/bin-flow-manager -- curl -s localhost:9090/metrics | grep activeflow_running

# Action dispatch errors
kubectl exec -n voipbin deploy/bin-flow-manager -- curl -s localhost:9090/metrics | grep action_error_total
```

## Deployment

bin-flow-manager deploys via Komodo (VOIP-1347 Tier 1 rollout, following the
VOIP-1342/bin-call-manager pilot pattern) instead of the older SSH +
`versions.lock` (`ssh-deploy.sh`) path.

- **Stack definition:** `bin-flow-manager/komodo/docker-compose.yml` (git is
  the source of truth for structure; Komodo only executes it on
  request).
- **CI path:** `.circleci/scripts/render-image-tag.sh` substitutes
  the built image tag, then `.circleci/scripts/komodo-api-deploy.sh`
  pushes the file's content to Komodo and triggers a deploy, gated
  by the `bin-flow-manager-deploy` job's poll/running checks.
- **Full design and cutover procedure:**
  [docs/plans/2026-08-18-bin-manager-komodo-rollout-tier1-design.md](../../docs/plans/2026-08-18-bin-manager-komodo-rollout-tier1-design.md)
  (in the monorepo root, not this service's own `docs/`).

## Configuration

| Flag | Env Var | Default | Description |
|------|---------|---------|-------------|
| `rabbitmq_address` | `RABBITMQ_ADDRESS` | _(required)_ | RabbitMQ server address (amqp URL) |
| `prometheus_endpoint` | `PROMETHEUS_ENDPOINT` | _(empty)_ | HTTP path for Prometheus metrics scrape |
| `prometheus_listen_address` | `PROMETHEUS_LISTEN_ADDRESS` | _(empty)_ | Listen address for metrics HTTP server |
| `database_dsn` | `DATABASE_DSN` | _(required)_ | MySQL DSN (`user:pass@tcp(host:port)/db`) |
| `redis_address` | `REDIS_ADDRESS` | _(required)_ | Redis server address (`host:port`) |
| `redis_password` | `REDIS_PASSWORD` | _(empty)_ | Redis password (optional) |
| `redis_database` | `REDIS_DATABASE` | `0` | Redis logical database index |

## Prometheus Metrics

All metric names are prefixed with `flow_manager_` at runtime.

| Metric Name | Type | Description |
|-------------|------|-------------|
| `action_dispatch_total` | Counter | Total action dispatch calls (labels: `type`) |
| `action_error_total` | Counter | Total action dispatch errors (labels: `type`) |
| `action_executed_total` | Counter | Total actions successfully executed (labels: `type`) |
| `action_exeucte_duration` | Histogram | Action execution duration in seconds (labels: `type`) |
| `activeflow_created_total` | Counter | Total activeflows created |
| `activeflow_duration_seconds` | Histogram | Duration of completed activeflows in seconds |
| `activeflow_ended_total` | Counter | Total activeflows that reached terminal state |
| `activeflow_execute_iterations` | Counter | Total execution step iterations across all activeflows |
| `activeflow_running` | Gauge | Number of activeflows currently in `executing` state |
| `flow_crud_total` | Counter | Total flow CRUD operations (labels: `method`) |
| `receive_request_process_time` | Histogram | RPC request processing time (labels: `type`, `method`) |
| `receive_subscribe_event_process_time` | Histogram | Event subscription processing time (labels: `publisher`, `type`) |
