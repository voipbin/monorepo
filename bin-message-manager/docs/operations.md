# bin-message-manager — Operations

## Common Failure Modes

| Failure | Symptom | Likely Cause |
|---------|---------|-------------|
| SMS not sent | Message created but target stays `queued` | Provider API credentials invalid or provider API down |
| Balance rejection | `POST /v1/messages` returns 402 | Customer billing balance insufficient |
| Webhook not processed | Target status never updates | `bin-hook-manager` not forwarding; or provider webhook URL misconfigured |
| Inbound MMS has no `medias` | `message_updated` with attachments never arrives for an inbound `mms` | See the `inbound_media_total` results: `host` or `auth` means the media host of the provider is not the assumed one or needs credentials (change `mediaAllowedHostSuffixes` in `media_fetcher.go`), `storage` means bin-storage-manager failed or its quota is full, `busy` means more ingests than slots, `size` and `type` mean the attachment is outside the limits. The host name is in the logs for `host`, `auth` and `network`; URLs are never logged. |
| MMS target `failed` | A message of type `mms` has a target in `failed` and the others are `queued` or `sent` | The provider refused the media or the destination (the `mms_target_failed_total` counter shows the provider and the reason), or the destination is outside the US and Canada and Telnyx refused it |
| MessageBird auth failure | Targets fail with 401 | The stored MessageBird token (`provider.messagebird.auth_token`) is expired, wrong or not configured |
| Telnyx auth failure | Telnyx answers 401, which counts as a refusal, so the targets go on to MessageBird and fail only when MessageBird refuses them as well | The stored Telnyx token (`provider.telnyx.auth_token`) is expired, wrong or not configured |
| Targets `failed` with no provider answer | The log has `the provider token could not be read, so the request was not sent` | The stored token could not be read (database error, or `db_undecryptable`). See [Provider tokens](#provider-tokens) |
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
| (no flag) | `MESSAGE_CONFIG_ENCRYPTION_KEYS` | `""` | Master key list of the stored tokens: comma-separated `<version>:<base64 32 bytes>`. Environment only, never a flag. A malformed or empty value does not stop the start: the service logs the problem (never the value), refuses to store a secret and reports the stored tokens as undecryptable |

## Provider tokens

The two provider tokens are stored settings, read from the database only (the shared `bin-common-handler/pkg/confighandler`, tables `message_configs` and `message_config_histories`).

| Key | Used for |
|-----|----------|
| `provider.messagebird.auth_token` | SMS and MMS sent through MessageBird (the targets that Telnyx refused) |
| `provider.telnyx.auth_token` | SMS and MMS sent through Telnyx |

- **Source:** the stored value only. The environment variables `AUTHTOKEN_MESSAGEBIRD` and `AUTHTOKEN_TELNYX` are not read by this service any more (VOIP-1721): a key without a stored row is an empty token, and the start logs a warning for the Telnyx token and an info line for the MessageBird token when they are not configured (key name only). Enter both tokens with `message-control config set` or the admin API before the deploy of this version. The keys are independent.
- **Read per send, no cache:** the token is read from the database for every provider request (SMS and MMS, MessageBird and Telnyx), each read bounded to 2 seconds. A token that was changed is used by the next request on every replica, with no restart. There is no cache, so a change needs nothing else.
- **A token that can not be read is an error:** a database error, a read that ran out of time, or a stored value that can not be decrypted (`db_undecryptable`) fails the send of that target (the target is marked `failed`, no request goes to the provider and the other provider is not tried). It is never replaced by an empty token. A key whose source is `none` is an empty token.
- **Telnyx has two settings:** `bin-number-manager` reads its own Telnyx secret (the stored number provider secret only) and this service reads `provider.telnyx.auth_token`. They come from different places and one does not follow the other. When the Telnyx account key is rotated, change both: the number provider secret through the number provider API and the message token with `message-control config set`. Changing only one leaves the other service on the old key.

### Commands

```bash
# the keys with their source (never a value): db, none, db_undecryptable, db_invalid
message-control config list

# store a value: the value comes from stdin only (never an argument); --actor-id is required
printf '%s' "$NEW_TOKEN" | message-control config set --key provider.telnyx.auth_token --actor-id <uuid>

# remove the stored value (the key is then not configured)
message-control config clear --key provider.telnyx.auth_token --actor-id <uuid>
```

`message-control config` needs only the database and the master key list. The same keys can be managed through the admin API (`/service_configs/message`, history source `api`).

### Rotation order

1. Check that every replica runs an image that has this feature. During a rolling deploy two replicas run different images: **do not store a value until every replica runs the new image**, because an old replica ignores the stored value and would keep sending with the token it has.
2. Store the new value (`config set`). The next request uses it.
3. Check the result without sending a message: `config list` must show the key with source `db` and `has_secret` true. This shows that the value is stored and readable, not that the provider accepts it; a wrong value shows up as an authentication failure (401) at the next real send, so watch `request_external_process_time` and the log of the first sends. Do not send a test message for the check: a send costs money.
4. If the value is wrong, store the right one (`config clear` leaves the key not configured). An image from before VOIP-1721 reads the two environment variables again when a key has no stored row, and a token rotated in the database only is not rotated for that image; keep the variables of the deployment in step until the rollback window has ended.
5. For Telnyx, repeat the change in the number service (see above).

### Master key list

Losing or mistyping `MESSAGE_CONFIG_ENCRYPTION_KEYS` does not stop the service. A stored token then reports `db_undecryptable` and its sends fail (see above), while a key with no row is not configured. Recover by restoring the list (the stored values are readable again), or by `config clear` of the affected keys and `config set` again: the stored value is lost. `config set` is refused while the list is missing or invalid. A retired master key version must stay in the list until no row uses it (`key_version` in `config list`). The deployment definition must pass the variable to the container, and the variable must exist in the deployment system before the deploy. A missing variable does not stop the container: Komodo passes the unreplaced `[[...]]` text and the service behaves as for a malformed value.

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
