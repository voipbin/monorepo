# bin-billing-manager — Operations

## Common Failure Modes

| Failure | Symptom | Likely Cause |
|---------|---------|-------------|
| Balance check rejected | Services fail to create resources | Account balance at zero or account not found |
| Paddle webhook rejected | Subscriptions not activated | The webhook signature is not valid (the billing service answers 401, which the hook service reports to Paddle as a client error), or the value of `PADDLE_WEBHOOK_SECRET` differs from the value that Paddle signs with. Check the start log line of `PADDLE_WEBHOOK_SECRET` (set, not set, unresolved placeholder). See "Paddle credentials" below |
| Paddle webhook answered 500 | Paddle may retry; subscriptions not activated | `PADDLE_WEBHOOK_SECRET` is not set or still holds an unresolved placeholder (the start log says so, and the error log says that the webhook secret is not configured). See "Webhook signature check" below |
| Portal session request fails | The customer portal link is not created | `PADDLE_API_KEY` is not set or still holds an unresolved placeholder (the call stops before any request to Paddle), or is wrong. Check the start log line of `PADDLE_API_KEY` |
| FailedEvent accumulation | `failed_event_save_total` rising | Downstream RPC failures during billing event processing |
| Event processing lag | Billing records not created | subscribehandler consumer stalled; check RabbitMQ queue depth |
| Cache stale | Old balance returned | Redis out of sync; restart flushes cache and forces DB reads |
| Missing billing record | Calls billed incorrectly | `call_hangup` event dropped before billing record finalized |

## Schema Migration Failures

Migrations run at startup, before the service connects the cache and the message broker; version table `billing_schema_migrations`. A failure exits the process with a non-zero code and the container is restarted.

| Symptom | Likely Cause | Resolution |
|---------|-------------|------------|
| `Dirty database version N. Fix and force version.` | A previous run stopped inside migration N (failure, kill) | Stop the crash-looping container. Check whether migration N's statement was applied. Run `billing-control migrate force <N-1>` if not (for N=1 use `force -- -1`), `force <N>` if so, then start again. The image has no shell, so run the command in a one-off container of the same image: `docker run --rm --entrypoint ./billing-control -e DATABASE_DSN=<dsn> <image> migrate force <N>` (the container must reach the database; quote a DSN that holds special characters) |
| `can't acquire lock` | Another instance is migrating for longer than 10 seconds | The restart policy retries; no action unless it persists |
| `Error 1205 ... Lock wait timeout` | A long transaction holds a metadata lock on a billing table (even `CREATE TABLE IF NOT EXISTS` takes it briefly) | Find the long transaction, then restart. If the database is marked dirty, follow the dirty row |
| `CREATE command denied ... billing_schema_migrations` | The database account cannot create tables | Grant CREATE to the service account |
| `db version N is ahead of this image (max M); skipping migrations` (warning) | An older image started after a newer one migrated | Expected during a rollback; no action |

`billing-control migrate status` shows the applied version, the image version and the dirty flag (it creates the version table when it is missing).

On a new database the service creates its tables at startup (the baseline is `CREATE TABLE IF NOT EXISTS`); no other schema tool runs first. The service account also needs permission to create tables (for the version table). The database must be created with `CHARACTER SET utf8 COLLATE utf8_general_ci`, like the install scripts do.

## Debugging Guide

**Check account balance:**
```bash
billing-control account get --id <account-uuid>
```

**List recent billing records for a customer:**
```bash
billing-control billing list --customer-id <uuid> --limit 20
```

**Force balance adjustment (admin only):**
```bash
# Add balance
billing-control account add-balance --id <account-uuid> --amount 10.00

# Subtract balance
billing-control account subtract-balance --id <account-uuid> --amount 5.00
```

**Check failed events (via database):**
```sql
SELECT * FROM billing_failed_events WHERE status != 'finished' ORDER BY tm_create DESC LIMIT 20;
```

**Run service locally:**
```bash
DATABASE_DSN="user:pass@tcp(127.0.0.1:3306)/bin_manager" \
RABBITMQ_ADDRESS="amqp://guest:guest@localhost:5672" \
REDIS_ADDRESS="127.0.0.1:6379" \
REDIS_DATABASE=1 \
./bin/billing-manager
```

**Check Paddle webhook delivery:**
- Paddle dashboard → Notifications → Event log
- Look for events with status `failed` and check the response body

## Deployment

bin-billing-manager deploys via Komodo (VOIP-1347 Tier 1 rollout, following the
VOIP-1342/bin-call-manager pilot pattern) instead of the older SSH +
`versions.lock` (`ssh-deploy.sh`) path.

- **Stack definition:** `bin-billing-manager/komodo/docker-compose.yml` (git is
  the source of truth for structure; Komodo only executes it on
  request).
- **CI path:** `.circleci/scripts/render-image-tag.sh` substitutes
  the built image tag, then `.circleci/scripts/komodo-api-deploy.sh`
  pushes the file's content to Komodo and triggers a deploy, gated
  by the `bin-billing-manager-deploy` job's poll/running checks.
- **Full design and cutover procedure:**
  [docs/plans/2026-08-18-bin-manager-komodo-rollout-tier1-design.md](../../docs/plans/2026-08-18-bin-manager-komodo-rollout-tier1-design.md)
  (in the monorepo root, not this service's own `docs/`).

## Configuration

All flags can also be set via environment variable (uppercase, underscores). No defaults shown means the flag is required.

| Flag | Env Var | Default | Description |
|------|---------|---------|-------------|
| `rabbitmq_address` | `RABBITMQ_ADDRESS` | required | RabbitMQ server address |
| `database_dsn` | `DATABASE_DSN` | required | MySQL DSN |
| `redis_address` | `REDIS_ADDRESS` | required | Redis address |
| `redis_password` | `REDIS_PASSWORD` | `""` | Redis auth password |
| `redis_database` | `REDIS_DATABASE` | `0` | Redis DB index |
| `prometheus_endpoint` | `PROMETHEUS_ENDPOINT` | `""` | Metrics path (e.g. `/metrics`) |
| `prometheus_listen_address` | `PROMETHEUS_LISTEN_ADDRESS` | `""` | Metrics listen address (e.g. `:2112`) |
| `paddle_price_id_basic` | `PADDLE_PRICE_ID_BASIC` | required | Paddle price ID for basic plan |
| `paddle_price_id_professional` | `PADDLE_PRICE_ID_PROFESSIONAL` | required | Paddle price ID for professional plan |
| (no flag) | `PADDLE_API_KEY` | optional | Paddle API key (customer portal session). Environment only, never a flag, never logged. Unset, empty or an unresolved placeholder (`[[...`) means not configured |
| (no flag) | `PADDLE_WEBHOOK_SECRET` | required for webhooks | Secret that signs the Paddle webhooks. Environment only, never a flag, never logged. Unset, empty or an unresolved placeholder means not configured: every Paddle webhook is rejected |
| (no flag) | `BILLING_CONFIG_ENCRYPTION_KEYS` | optional | Not read by the code. The deployment definition still passes it; a later change removes the line |

## Paddle credentials

The Paddle credentials are read from two environment variables of this service:

| Variable | Used by |
|----------|---------|
| `PADDLE_API_KEY` | The Paddle API call that creates a customer portal session |
| `PADDLE_WEBHOOK_SECRET` | The signature check of the Paddle webhooks (`/v1/hooks/paddle`) |

The variables are the only source. Nothing is stored in the database (the tables `billing_configs` and `billing_config_histories` are no longer read or written by the code). The compose value is `[[BIN_MANAGER__<variable name>]]` with the same name as the variable; in Kubernetes the variable is an optional key of the `voipbin` secret with the same name.

Resolution, for every use (nothing is cached, so a replica restart is the only way a value changes):

1. A variable with a value is the value (surrounding white space is removed).
2. A variable that is not set, is empty, or holds an unresolved placeholder (a value that starts with `[[`, the text of a deployment variable that does not exist) means the key is not configured. The caller decides what that means: the webhook is rejected (500), the portal session call stops before any request to Paddle. The placeholder is never sent anywhere.
3. A deployment variable that is missing leaves the placeholder in the compose value. The start log says so once per variable (`unresolved placeholder`, the variable name, never a value).

The start of the service logs one line per variable with its state (set, not set, placeholder; name and state only). A webhook secret that is not set is a warning (every Paddle webhook is rejected), and an API key that is not set is an info line (a portal session can not be created). An unresolved placeholder is a warning for both.

### Webhook signature check

For `/v1/hooks/paddle` the billing service is the only place that verifies the `Paddle-Signature` header. The hook service checks the body size and that the header exists, and forwards the header value unchanged. The algorithm and the time window are `ts=<unix>;h1=<hex>`, HMAC-SHA256 over `ts:body`, 300 seconds in both directions.

| State of `PADDLE_WEBHOOK_SECRET` | Result |
|---|---|
| Not set, empty or an unresolved placeholder | 500, nothing is processed (fail closed). The error log says that the secret is not configured. The hook service answers 500, so Paddle may retry later |
| Set, signature valid | The event is processed |
| Set, signature wrong, expired, malformed or missing | The billing service answers 401 and processes nothing. The hook service maps this answer to a client error (400) and Paddle may not retry the event |

The signature and the body are not written to the log lines of this check. Note that the existing parse-failure error line of the handler carries the whole request, which includes the signature, and the debug line of the request loop logs the whole request as before.

Whether Paddle retries a rejected webhook (4xx and 5xx) and whether a retry carries a new signature timestamp is not confirmed. If webhooks were rejected while the secret was not set, resend them from the Paddle dashboard after the variable is set.

#### Risk and prerequisites

- The hook service does not verify the signature. It checks the body size and that the `Paddle-Signature` header exists, and the billing service verifies the signature.
- Both variables are mandatory for a deployment that uses Paddle. Without `PADDLE_WEBHOOK_SECRET` every Paddle webhook is rejected (fail closed), which loses payment events; without `PADDLE_API_KEY` no portal session can be created. Before the deployment, check that the deployment variables `PADDLE_API_KEY` and `PADDLE_WEBHOOK_SECRET` exist and that the webhook secret is the value that Paddle signs with. A secret that differs from that value rejects every genuine webhook with 401 (reported as 400 by the hook service).
- `PADDLE_WEBHOOK_SECRET` is a new name: the hook service of an older version used `PADDLE_WEBHOOK_SECRET_KEY`. The billing service does not read that name.
- The flag `--paddle_api_key` does not exist; the variables are not flags.

#### Deployment order and rollback

- Deploy the billing service first and wait until every replica runs the new image, then the hook service (it does not verify the signature).
- Rollback is the previous image together with the previous deployment definition. The previous image reads the same variables.

## Prometheus Metrics

Metrics are exposed on the configured `prometheus_listen_address` + `prometheus_endpoint`.

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `billing_manager_receive_request_process_time` | Histogram | `type`, `method` | RPC request processing duration |
| `billing_manager_receive_subscribe_event_process_time` | Histogram | `publisher`, `type` | Event processing duration |
| `account_balance_check_total` | Counter | — | Total balance check operations |
| `account_create_total` | Counter | — | Total account creation operations |
| `billing_create_total` | Counter | — | Total billing records created |
| `billing_duration_seconds` | Histogram | — | Billing event duration in seconds |
| `billing_end_total` | Counter | — | Total billing records finalized |
| `failed_event_exhausted_total` | Counter | — | Failed events that exhausted retries |
| `failed_event_retry_total` | Counter | — | Failed event retry attempts |
| `failed_event_save_total` | Counter | — | Failed events persisted to retry queue |

**Alert guidance:**
- `failed_event_exhausted_total` increasing → billing records are being permanently lost; investigate downstream RPC failures.
- `account_balance_check_total` spike → elevated resource creation attempts; normal under load, but check for abuse.
- `billing_manager_receive_subscribe_event_process_time` p99 > 1s → subscribehandler processing too slow; check DB query performance.
