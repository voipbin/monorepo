# bin-message-manager — Operations

## Common Failure Modes

| Failure | Symptom | Likely Cause |
|---------|---------|-------------|
| SMS not sent | Message created but target stays `queued` | Provider API credentials invalid or provider API down |
| Balance rejection | `POST /v1/messages` returns 402 | Customer billing balance insufficient |
| Webhook not processed | Target status never updates | `bin-hook-manager` not forwarding; or provider webhook URL misconfigured |
| Inbound MMS has no `medias` | `message_updated` with attachments never arrives for an inbound `mms` | See the `inbound_media_total` results: `host` or `auth` means the media host of the provider is not the assumed one or needs credentials (change `mediaAllowedHostSuffixes` in `media_fetcher.go`), `storage` means bin-storage-manager failed or its quota is full, `busy` means more ingests than slots, `size` and `type` mean the attachment is outside the limits. The host name is in the logs for `host`, `auth` and `network`; URLs are never logged. |
| MMS target `failed` | A message of type `mms` has a target in `failed` and the others are `queued` or `sent` | The provider refused the media or the destination (the `mms_target_failed_total` counter shows the provider and the reason), or the destination is outside the US and Canada and Telnyx refused it |
| MessageBird auth failure | Targets fail with 401 | The `MESSAGEBIRD_API_KEY` variable is expired, wrong or not configured (see [Provider tokens](#provider-tokens)) |
| Telnyx auth failure | Telnyx answers 401, which counts as a refusal, so the targets go on to MessageBird and fail only when MessageBird refuses them as well | The `TELNYX_API_KEY` variable is expired, wrong or not configured (see [Provider tokens](#provider-tokens)) |
| Cache stale | Stale message data returned | Redis out of sync; restart clears cache |

## Schema Migration Failures

Migrations run at startup, before the RPC listener starts. A failure exits the process with a non-zero code.

| Symptom (log) | Meaning | Action |
|---|---|---|
| `Dirty database version N. Fix and force version.` | A previous run stopped inside migration N (failure, kill) | Stop the crash-looping container. Check whether migration N's statement was applied. `message-control migrate force <N-1>` if not (for N=1 use `force -- -1`), `force <N>` if so (`force -- -1` resets to unversioned), then start again. See the migration section of `CLAUDE.md` |
| `can't acquire lock` | Another instance is migrating for longer than 10 seconds | The restart policy retries; no action unless it persists |
| `Error 1205 ... Lock wait timeout` | Only for ALTER style migrations after the baseline (the baseline does not block on an existing table): a long transaction holds the table; the migration left the database dirty | Find the long transaction, then force `N-1` and restart |
| `CREATE command denied ... message_schema_migrations` | The database account cannot create tables | Grant CREATE to the service account |
| `db version N is ahead of this image (max M); skipping migrations` (warning) | An older image started after a newer one migrated | Expected during rollback; no action |

`message-control migrate status` shows the applied version, the image version and the dirty flag (it creates the version table if missing). `message-control` reads `DATABASE_DSN` and opens only the database. In the runtime image (distroless, no shell) run it as a one-off container with the same image and environment: `docker run --rm --entrypoint ./message-control <image> migrate status`.

## Order on a New Database

Create the database with `CHARACTER SET utf8 COLLATE utf8_general_ci` (as the install scripts do). On a new database the service creates its tables at startup (the baseline is `CREATE TABLE IF NOT EXISTS`); no other schema tool runs first. The service account also needs permission to create tables (for the version table).

## Debugging Guide

**Get a message (bypasses RabbitMQ):**
```bash
./bin/message-control message get --id <uuid>
```

**List messages for a customer:**
```bash
./bin/message-control message list --customer_id <uuid> [--limit 100] [--token]
```

**Delete a message:**
```bash
./bin/message-control message delete --id <uuid>
```

**Migration state of a local database:**
```bash
DATABASE_DSN='<user>:<password>@tcp(127.0.0.1:3306)/<database>' ./bin/message-control migrate status
```

**Check provider webhook configuration:**
- MessageBird dashboard → Channels → your number → Delivery Reports URL
- Telnyx portal → Messaging Profiles → your profile → Webhooks

**Provider webhook endpoint** (via hook-manager):
```
POST https://hook.voipbin.net/v1.0/messages/messagebird
POST https://hook.voipbin.net/v1.0/messages/telnyx
```

**Run service locally:**
```bash
DATABASE_DSN="user:pass@tcp(127.0.0.1:3306)/voipbin" \
RABBITMQ_ADDRESS="amqp://guest:guest@localhost:5672" \
REDIS_ADDRESS="127.0.0.1:6379" \
./bin/message-manager
```

## Deployment

bin-message-manager deploys via Komodo (VOIP-1348 Tier 2 rollout,
following the VOIP-1342/bin-call-manager pilot and VOIP-1347/Tier 1
pattern) instead of the older SSH + `versions.lock` (`ssh-deploy.sh`)
path.

- **Stack definition:** `bin-message-manager/komodo/docker-compose.yml`
  (git is the source of truth for structure; Komodo only executes it on
  request).
- **CI path:** `.circleci/scripts/render-image-tag.sh` substitutes
  the built image tag, then `.circleci/scripts/komodo-api-deploy.sh`
  pushes the file's content to Komodo and triggers a deploy, gated
  by the `bin-message-manager-deploy` job's poll/running checks.
- **Full design and cutover procedure:**
  [docs/plans/2026-08-18-bin-manager-komodo-rollout-tier2-design.md](../../docs/plans/2026-08-18-bin-manager-komodo-rollout-tier2-design.md)
  (in the monorepo root, not this service's own `docs/`).

## Configuration

All flags can also be set via environment variable (uppercase, underscores).

| Flag | Env Var | Default | Description |
|------|---------|---------|-------------|
| `rabbitmq_address` | `RABBITMQ_ADDRESS` | required | RabbitMQ server address |
| `database_dsn` | `DATABASE_DSN` | required | MySQL DSN |
| `redis_address` | `REDIS_ADDRESS` | required | Redis address |
| `redis_password` | `REDIS_PASSWORD` | `""` | Redis auth password |
| `redis_database` | `REDIS_DATABASE` | `0` | Redis DB index |
| `prometheus_endpoint` | `PROMETHEUS_ENDPOINT` | `/metrics` | Metrics path |
| `prometheus_listen_address` | `PROMETHEUS_LISTEN_ADDRESS` | `:2112` | Metrics listen address |
| (no flag) | `MESSAGEBIRD_API_KEY` | `""` | MessageBird access key. Environment only, never a flag; read at the time of use. Not set, empty or an unresolved `[[...]]` placeholder means not configured |
| (no flag) | `TELNYX_API_KEY` | `""` | Telnyx API key of the SMS and MMS sends. Environment only, never a flag; the same variable and value as in `bin-number-manager`. Not set, empty or an unresolved placeholder means not configured |
| (no flag) | `MESSAGE_CONFIG_ENCRYPTION_KEYS` | `""` | Not read by the code. The deployment definition still passes it; a later change removes the line |

## Provider tokens

The two provider tokens are read from environment variables, and only from there. Nothing is stored in the database (the tables `message_configs` and `message_config_histories` are no longer read or written by the code) and no admin API writes them.

| Variable | Used for |
|----------|----------|
| `MESSAGEBIRD_API_KEY` | SMS and MMS sent through MessageBird (the targets that Telnyx refused) |
| `TELNYX_API_KEY` | SMS and MMS sent through Telnyx |

- **Source:** the environment variable only. A variable that is not set, is empty or still holds an unresolved deployment placeholder (a value that starts with `[[`) means that the token is not configured: the supplier gives an empty token and no error, and the request is sent with that empty credential and refused by the provider. The placeholder is never sent as a token. At start the service logs one line per variable (name and state only, never a value): a warning for `TELNYX_API_KEY` that is not set, an info line for `MESSAGEBIRD_API_KEY` that is not set, and a warning with the words "unresolved placeholder" for a placeholder.
- **Read per send, no cache:** the variable is read for every provider request. The environment of a container changes with a new deployment only, so a rotation takes effect when the replicas restart.
- **One variable for Telnyx:** `TELNYX_API_KEY` is the variable that `bin-number-manager` reads for its Telnyx secret too, so one rotation of the deployment variable covers both services. The two services must hold the same value; this is checked in the deployment variables, not in the code.
- **Rollback:** roll back the image together with its deployment definition. The variables above stay in the deployment variable store and are unused by an older image.

### Rotation order

1. Change `MESSAGEBIRD_API_KEY` or `TELNYX_API_KEY` in the deployment variable store and redeploy the service. For Telnyx, `bin-number-manager` reads the same variable, so redeploy it as well; the two services must end with the same value. During a rolling deploy two replicas can hold different values until the deploy ends.
2. Check the result without sending a message: the start log of each replica shows the variable as set (never the value). The provider accepting the value shows only at the next real send (an authentication failure is a 401): watch `request_external_process_time` and the log of the first sends. Do not send a test message for the check: a send costs money.
3. If the value is wrong, correct the deployment variable and redeploy.

## Prometheus Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `receive_request_process_time` | Histogram | `type`, `method` | RPC request processing duration |
| `mms_target_failed_total` | Counter | `provider`, `reason` | MMS targets that no provider accepted (`rejected`, `error`, `ineligible`) |
| `mms_media_check_seconds` | Histogram | none | Duration of the media validation and signed URL refresh of one MMS request |
| `inbound_media_total` | Counter | `result` | Media of received messages by result: `stored`, `parse`, `type`, `host`, `size`, `auth`, `status`, `network`, `timeout`, `storage`, `overflow`, `busy` |
| `segment_mismatch_total` | Counter | `provider` | Targets whose segment count reported by an SMS provider differs from the platform count |
| `receipt_unmatched_total` | Counter | none | Delivery receipts that matched no message |
| `dispatch_final_write_failed_total` | Counter | none | Dispatches whose final result could not be written (the targets stay `queued`) |
| `event_dropped_total` | Counter | none | `message_updated` events dropped because the message could not be read |
| `promoted_messages_total` | Counter | `segments` | Texts promoted to a text-only MMS |
| `promoted_mms_total` | Counter | `provider`, `outcome` | MMS requests of promoted messages by outcome |
| `promoted_fallback_total` | Counter | `reason`, `segments` | Targets of promoted messages sent as SMS segments (`ineligible`, `rejected`) |
| `request_external_process_time` | Histogram | `provider`, `method` | Provider API call duration |
| `messagebird_number_send_total` | Counter | `type` | Total SMS sent via MessageBird |
| `telnyx_number_send_total` | Counter | `type` | Total SMS sent via Telnyx |

**Alert guidance:**
- `inbound_media_total{result="host"}` or `{result="auth"}` rising → the assumptions about the media host or authentication of the provider are wrong; every attachment of a received MMS is skipped.
- `inbound_media_total{result="storage"}` or `{result="busy"}` rising → storage-manager unavailable or full, or more received MMS than the process can ingest (two at a time; memory is the reason for the small limit).
- `messagebird_number_send_total` dropping to zero → MessageBird auth failure or API down; SMS will fail silently.
- `request_external_process_time` p99 > 10s → provider API slow; messages may time out.
- `promoted_fallback_total{reason="rejected"}` rising → providers refuse a text-only MMS; the promoted traffic is sent as SMS segments but charged the MMS price.
- `dispatch_final_write_failed_total` rising → database trouble during dispatch; targets stay `queued` without a result.
- `receive_request_process_time` rising → RabbitMQ queue backing up; check consumer health.
