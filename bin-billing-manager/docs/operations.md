# bin-billing-manager — Operations

## Common Failure Modes

| Failure | Symptom | Likely Cause |
|---------|---------|-------------|
| Balance check rejected | Services fail to create resources | Account balance at zero or account not found |
| Paddle webhook rejected | Subscriptions not activated | The webhook signature is not valid (the billing service answers 401, which the hook service reports to Paddle as a client error), or the stored webhook secret differs from the one of the hook service. Check `billing-control config list` (source of `paddle.webhook_secret_key`) and the hook service value. See "Paddle credentials" below |
| Paddle webhook answered 500 | Paddle retries; subscriptions not activated | The webhook secret could not be read (database error, or a stored value that can not be decrypted). `billing-control config list` shows `db_undecryptable`. See "Recovery" below |
| Portal session request fails | The customer portal link is not created | The Paddle API key is missing or wrong, or could not be read (a read error never falls back to the environment variable). Check `billing-control config list` |
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
| (no flag) | `PADDLE_API_KEY` | optional | Fallback value of the stored setting `paddle.api_key` (used for the Paddle API call of the portal session) while no value is stored. Read with `os.LookupEnv`, at the time of use. It is not a flag any more |
| (no flag) | `PADDLE_WEBHOOK_SECRET_KEY` | optional | Fallback value of the stored setting `paddle.webhook_secret_key` (the secret that signs the Paddle webhooks) while no value is stored. The billing service did not read it before; it is read only for this setting |

## Paddle credentials

The Paddle credentials are two stored settings of this service (table `billing_configs`, encrypted with the master key list `BILLING_CONFIG_ENCRYPTION_KEYS`):

| Stored key | Used by | Environment fallback |
|------------|---------|----------------------|
| `paddle.api_key` | The Paddle API call that creates a customer portal session | `PADDLE_API_KEY` |
| `paddle.webhook_secret_key` | The signature check of the Paddle webhooks (`/v1/hooks/paddle`) | `PADDLE_WEBHOOK_SECRET_KEY` |

Resolution, for every use (nothing is cached, so a change is used by the next call of every replica, with no restart):

1. A stored value wins. The environment variable is then shadowed (`env_shadowed` in `config list`).
2. With no stored value, the environment variable of the key is the value (the service reads it with `os.LookupEnv`).
3. A database read error, a read that runs out of time (2 seconds) and a stored value that can not be decrypted are errors. They never fall back to the environment variable and are never turned into "not configured".

Enter and inspect the values with `billing-control config` or the admin API (`/service_configs/billing`, served as `/v1/configs` by the shared `configrpc`). The history records the source `api` for the admin API and `cli` for the command line. The request body of a secret is never logged, and a request that looks like a config request is not routed to any other handler.

```bash
# the value is read from stdin only (one line; a key is at most 512 bytes) and is never printed
printf '%s\n' "$NEW_VALUE" | ./billing-control config set --key paddle.api_key --actor-id <uuid>
printf '%s\n' "$NEW_VALUE" | ./billing-control config set --key paddle.webhook_secret_key --actor-id <uuid>
./billing-control config list
./billing-control config clear --key paddle.webhook_secret_key --actor-id <uuid>
```

`config list` shows, for each key, the source (`db`, `env`, `none`, `db_undecryptable`, `db_invalid`), whether a stored value shadows the variable, and never a value. `billing-control` reads the environment of its own process, so the source `env` or `none` that it prints can differ from what the daemon container holds: the daemon decides at the time of use. The start of the service logs one line per key with its source (key name and source only). A webhook secret that is not configured is a warning (the signature is not checked), and an API key that is not configured is an info line.

The request body limit of the admin API is 4 KB, counted in bytes of the JSON text. A secret that grows beyond it through JSON escaping (quotes, backslashes, non-ASCII) is rejected with `INVALID_JSON_BODY` (400).

### Webhook signature check

For `/v1/hooks/paddle` the billing service checks the `Paddle-Signature` header again (the hook service checks it first, and forwards the header value). The algorithm and the time window are the same as in the hook service (`ts=<unix>;h1=<hex>`, HMAC-SHA256 over `ts:body`, 300 seconds in both directions).

| State of `paddle.webhook_secret_key` | Result |
|---|---|
| Not configured (no stored value, variable empty or unset) | The check is skipped, the event is processed (a warning without request data is logged) |
| Configured, signature valid | The event is processed |
| Configured, signature wrong, expired, malformed or missing | The billing service answers 401 and processes nothing. The hook service maps this answer to a client error (400) and Paddle may not retry the event (whether Paddle retries a 400 is not confirmed) |
| Configured, but the value could not be read or decrypted | 500, nothing is processed (fail closed). The hook service answers 500, so Paddle retries later |

The signature and the body are not written to the new log lines of this check. Note that the existing parse-failure error line of the handler carries the whole request, which includes the signature, and the debug line of the request loop logs the whole request as before.

#### Risk and prerequisites

- The hook service keeps checking its own `PADDLE_WEBHOOK_SECRET_KEY` before it forwards a webhook. A request that passed the hook service and is rejected by the billing service therefore means that the stored `paddle.webhook_secret_key` differs from the value of the hook service. The hook service reports the rejection as a client error (400), and it is not confirmed that Paddle retries a 400, so a genuine event can be lost. **Store `paddle.webhook_secret_key` in the billing database only with the same value that the hook service has** (compare before you store), and only after **both the hook service and every replica of the billing service run the image of this feature**. An older billing replica ignores the stored value and an older hook service does not forward the signature (the new billing service then rejects every webhook with 401 while a secret is stored).
- During a rolling deploy do not store the webhook secret. Until it is stored, the new billing service skips the check and the behavior is the same as before.
- The deployment definition of the billing service must pass `BILLING_CONFIG_ENCRYPTION_KEYS` (Komodo: `BIN_MANAGER__BILLING_MANAGER_CONFIG_ENCRYPTION_KEYS`; Kubernetes: the optional key of the `voipbin` secret). Without it a secret can not be stored.
- The Komodo stack of the billing service does not pass `PADDLE_API_KEY` or `PADDLE_WEBHOOK_SECRET_KEY`. Where the billing container receives its Paddle API key today is not defined by this repository. Run `billing-control config list` against the deployment and confirm that `paddle.api_key` has the source you expect (`db` or `env`) before you rely on it. The portal session needs it.
- The flag `--paddle_api_key` was removed. The API key is read through the stored setting, with `PADDLE_API_KEY` as the fallback.

#### Master key rotation

Master keys have versions: the list is `<version>:<base64 key>[,...]`, the highest version encrypts new values and every listed version decrypts. To rotate: (1) add the new version to `BILLING_CONFIG_ENCRYPTION_KEYS` and deploy, keeping the old version in the list; (2) re-enter every stored value (`config set` again for each key that has a stored value; a stored value keeps the version it was written with until it is written again); (3) remove the old version from the list only after `config list` shows that every stored key has the new `key_version`.

#### Recovery

A stored value that can not be decrypted (the master key list is missing, mistyped or changed) is not replaced by the environment variable. Webhooks answer 500 while `paddle.webhook_secret_key` is such a value, and portal session requests fail while `paddle.api_key` is. `config list` shows `db_undecryptable` and the start logs an error line. Restore the master key list, or run `config clear` for the key and store it again (the environment variable is the fallback again after the clear).

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
