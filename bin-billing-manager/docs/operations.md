# bin-billing-manager — Operations

## Common Failure Modes

| Failure | Symptom | Likely Cause |
|---------|---------|-------------|
| Balance check rejected | Services fail to create resources | Account balance at zero or account not found |
| Paddle webhook rejected | Subscriptions not activated | The webhook signature is not valid (the billing service answers 401, which the hook service reports to Paddle as a client error), or the stored webhook secret differs from the value that Paddle signs with. Check `billing-control config list` (source of `paddle.webhook_secret_key`). See "Paddle credentials" below |
| Paddle webhook answered 500 | Paddle may retry; subscriptions not activated | No webhook secret is stored (the error log says `The Paddle webhook secret is not stored`), or the secret could not be read (database error, or a stored value that can not be decrypted: `billing-control config list` shows `db_undecryptable`). See "Webhook signature check" and "Recovery" below |
| Portal session request fails | The customer portal link is not created | The Paddle API key is not stored (the call stops before any request to Paddle), is wrong, or could not be read. Check `billing-control config list` |
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
| (no flag) | `BILLING_CONFIG_ENCRYPTION_KEYS` | optional | Master key list that encrypts the stored Paddle credentials (environment only, never a flag, never logged). Without it the service starts, a stored secret is reported as undecryptable, and storing a secret is refused |

## Paddle credentials

The Paddle credentials are two stored settings of this service (table `billing_configs`, encrypted with the master key list `BILLING_CONFIG_ENCRYPTION_KEYS`):

| Stored key | Used by |
|------------|---------|
| `paddle.api_key` | The Paddle API call that creates a customer portal session |
| `paddle.webhook_secret_key` | The signature check of the Paddle webhooks (`/v1/hooks/paddle`) |

The keys are read from the database only. There is no environment variable fallback, and the keys have no default.

Resolution, for every use (nothing is cached, so a change is used by the next call of every replica, with no restart):

1. A stored value is the value.
2. With no stored value the key is not configured. The caller decides what that means: the webhook is rejected (500), the portal session call stops before any request to Paddle.
3. A database read error, a read that runs out of time (2 seconds) and a stored value that can not be decrypted are errors. They are never turned into "not configured".

Enter and inspect the values with `billing-control config` or the admin API (`/service_configs/billing`, served as `/v1/configs` by the shared `configrpc`). The history records the source `api` for the admin API and `cli` for the command line. The request body of a secret is never logged, and a request that looks like a config request is not routed to any other handler.

```bash
# the value is read from stdin only (one line; a key is at most 512 bytes) and is never printed
printf '%s\n' "$NEW_VALUE" | ./billing-control config set --key paddle.api_key --actor-id <uuid>
printf '%s\n' "$NEW_VALUE" | ./billing-control config set --key paddle.webhook_secret_key --actor-id <uuid>
./billing-control config list
./billing-control config clear --key paddle.webhook_secret_key --actor-id <uuid>
```

`config list` shows, for each key, the source (`db`, `none`, `db_undecryptable`, `db_invalid`) and never a value. The start of the service logs one line per key with its source (key name and source only). A webhook secret that is not configured is a warning (every Paddle webhook is rejected), and an API key that is not configured is an info line (a portal session can not be created).

The request body limit of the admin API is 4 KB, counted in bytes of the JSON text. A secret that grows beyond it through JSON escaping (quotes, backslashes, non-ASCII) is rejected with `INVALID_JSON_BODY` (400).

### Webhook signature check

For `/v1/hooks/paddle` the billing service is the only place that verifies the `Paddle-Signature` header. The hook service checks the body size and that the header exists, and forwards the header value unchanged. The algorithm and the time window are `ts=<unix>;h1=<hex>`, HMAC-SHA256 over `ts:body`, 300 seconds in both directions.

| State of `paddle.webhook_secret_key` | Result |
|---|---|
| Not stored | 500, nothing is processed (fail closed). The error log says that the secret is not stored. The hook service answers 500, so Paddle may retry later |
| Stored, signature valid | The event is processed |
| Stored, signature wrong, expired, malformed or missing | The billing service answers 401 and processes nothing. The hook service maps this answer to a client error (400) and Paddle may not retry the event |
| Stored, but the value could not be read or decrypted | 500, nothing is processed (fail closed). The hook service answers 500, so Paddle may retry later |

The signature and the body are not written to the log lines of this check. Note that the existing parse-failure error line of the handler carries the whole request, which includes the signature, and the debug line of the request loop logs the whole request as before.

Whether Paddle retries a rejected webhook (4xx and 5xx) and whether a retry carries a new signature timestamp is not confirmed. If webhooks were rejected while the secret was not stored, resend them from the Paddle dashboard after the secret is stored.

#### Risk and prerequisites

- A request with any `Paddle-Signature` header value reaches the billing service by RPC before it is rejected, because the hook service no longer verifies the signature. The hook service has no rate limit.
- Before the deployment check with `billing-control config list` that `paddle.api_key` and `paddle.webhook_secret_key` have the source `db` and `has_secret` true, and that the stored webhook secret is the value that Paddle signs with. A secret that is stored but differs from that value rejects every genuine webhook with 401 (reported as 400 by the hook service). One way to check the value: deploy the billing service first, while the hook service still verifies, send a test event from Paddle and confirm in the billing log that it passed the check. A rejection by the hook service in this test does not show whether the stored value is wrong.
- The deployment definition of the billing service must pass `BILLING_CONFIG_ENCRYPTION_KEYS` (Komodo: `BIN_MANAGER__BILLING_MANAGER_CONFIG_ENCRYPTION_KEYS`; Kubernetes: the optional key of the `voipbin` secret). Without it a secret can not be stored.
- The flag `--paddle_api_key` and the variables `PADDLE_API_KEY` and `PADDLE_WEBHOOK_SECRET_KEY` are not read by the billing service any more.

#### Deployment order and rollback

- Deploy the billing service first and wait until every replica runs the new image, then deploy the hook service. While the hook service still verifies, the old and the new hook behave the same for the billing service. A new hook service in front of an old billing service passes unverified requests when no secret is stored (the old billing service skips the check).
- To roll back the billing service, roll back the hook service first or together with it, and check the prerequisites above again: an old billing service skips the signature check while no secret is stored, so the old billing service behind the new hook service is open.
- Rolling back the hook service needs its old manifest, which passes `PADDLE_WEBHOOK_SECRET_KEY` (the Komodo variable `BIN_MANAGER__PADDLE_WEBHOOK_SECRET_KEY` and the cluster secret key stay in place and are removed in a separate step after the rollback window). If the stored webhook secret was rotated after this change, update that variable first: an old hook service with an outdated value rejects every genuine webhook with 400.
- `config clear` on a Paddle key is not a return to an environment variable: a cleared webhook secret rejects every webhook, and a cleared API key stops portal sessions.

#### Master key rotation

Master keys have versions: the list is `<version>:<base64 key>[,...]`, the highest version encrypts new values and every listed version decrypts. To rotate: (1) add the new version to `BILLING_CONFIG_ENCRYPTION_KEYS` and deploy, keeping the old version in the list; (2) re-enter every stored value (`config set` again for each key that has a stored value; a stored value keeps the version it was written with until it is written again); (3) remove the old version from the list only after `config list` shows that every stored key has the new `key_version`.

#### Recovery

A stored value that can not be decrypted (the master key list is missing, mistyped or changed) is not replaced by any other value. Webhooks answer 500 while `paddle.webhook_secret_key` is such a value, and portal session requests fail while `paddle.api_key` is. `config list` shows `db_undecryptable` and the start logs an error line. Restore the master key list, or run `config clear` for the key and store it again (the webhooks are rejected until the new value is stored).

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
