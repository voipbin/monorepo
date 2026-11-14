# bin-number-manager — Operations

## Common Failure Modes

| Failure | Symptom | Likely Cause |
|---------|---------|-------------|
| Number purchase fails | 402 or provider error | Insufficient billing balance or provider API credentials invalid |
| Telnyx/Twilio API error | Number not created | The provider credentials (the secret variable, and the config of the row or its variable) are wrong or expired |
| `could not purchase: no usable provider` | Every purchase fails | No enabled provider has its secret variable (`TELNYX_API_KEY`, `TWILIO_AUTH_TOKEN`) set to a value; see the start log of the variables, `credential_source` and `usable` of `GET /v1/number_providers` and the `provider_unusable_total` metric |
| `PURCHASE_UNCONFIRMED` in the log | Purchase fails, no fallback | The order result is unknown; check the provider for an order of that number before retrying |
| `ROLLBACK_RELEASE_FAILED` in the log | A number is bought but not recorded | Release it at the provider by hand |
| `credential_source: none` for a provider that should work | The provider is unusable, releases fail | Its secret variable is unset, empty or an unresolved `[[...]]` placeholder (the start log names the variable and its state); set the deployment variable and restart |
| Cascade not triggered | Numbers persist after customer delete | subscribehandler not receiving `customer_deleted` event; check RabbitMQ queue |
| Flow reference stale | Inbound calls fail after flow deleted | subscribehandler not receiving `flow_deleted` event; flow IDs not cleared |
| Available number search empty | No numbers returned | Country code not supported by provider, or provider API rate limited |
| Cache stale | Old number data served | Redis out of sync; restart flushes and forces DB reads |

## Schema Migration Failures

Migrations run at startup, before the service listens. A failure exits the process with a non-zero code.

| Symptom (log) | Meaning | Action |
|---|---|---|
| `Dirty database version N. Fix and force version.` | A previous run stopped inside migration N (failure, kill) | Stop the crash-looping container. Check whether migration N's statement was applied. `number-control migrate force <N-1>` if not (for N=1 use `force -- -1`), `force <N>` if so (`force -- -1` resets to unversioned), then start again. See the migration section of `CLAUDE.md` |
| `can't acquire lock` | Another instance is migrating for longer than 10 seconds | Restart policy retries; no action unless it persists |
| `Error 1205 ... Lock wait timeout` | Only for ALTER style migrations after the baseline (the baseline does not block on an existing table): a long transaction holds the table; the migration left the database dirty | Find the long transaction, then force `N-1` and restart |
| `CREATE command denied ... number_schema_migrations` | The database account cannot create tables | Grant CREATE to the service account |
| `db version N is ahead of this image (max M); skipping migrations` (warning) | An older image started after a newer one migrated | Expected during rollback; no action |

A failed first migration shows as version 1 dirty; only a failed down of the first migration is shown as version 0 not dirty (the library hides the marker for an unversioned database), in which case `up` still refuses with `Dirty database version` and the table above applies. `number-control migrate status` shows the applied version, the image version and the dirty flag (it creates the version table if missing).

## Order on a New Database

On a new database the service creates its tables at startup (the baseline is `CREATE TABLE IF NOT EXISTS`); no other schema tool runs first. The service account also needs permission to create tables (for the version table). A new database is created with `CHARACTER SET utf8 COLLATE utf8_general_ci` (like the install scripts do).

## Debugging Guide

**Get a number:**
```bash
./bin/number-control number get --id <uuid>
```

**List numbers for a customer:**
```bash
./bin/number-control number list --customer-id <uuid> [--limit 100] [--token]
```

**Schema migration status (database only, no Redis or RabbitMQ):**
```bash
./bin/number-control migrate status
```

**Search available numbers:**
```bash
./bin/number-control number get-available --country-code US [--limit 10]
```

**Update call flow on a number:**
```bash
./bin/number-control number update --id <uuid> --call-flow-id <flow-uuid>
```

**Delete (release) a number:**
```bash
./bin/number-control number delete --id <uuid>
```

**Provider settings and stored secrets (database and key list only, no Redis or RabbitMQ). The service reads the secret from the environment variable; the stored secret is not used:**
```bash
./bin/number-control provider list
./bin/number-control provider update --name telnyx --actor-id <uuid> [--enabled=true|false] [--priority <n>] [--config connection_id=<id>] [--config profile_id=<id>]
read -rs TOKEN; printf '%s' "$TOKEN" | ./bin/number-control provider secret set --name telnyx --actor-id <uuid>; unset TOKEN
./bin/number-control provider secret delete --name telnyx --actor-id <uuid>
```
`name` is `telnyx` or `twilio`. The commands call the same provider handler as the admin API (`/v1/number_providers`) and print the provider view as JSON: `credential_source`, `has_secret`, `key_version`, `usable` and the non-secret `config`, never a secret, a ciphertext or a length.

- **Order**: set the required config first (Telnyx `connection_id`; Twilio `sid` and `trunk_sid`), then the secret. The secret is refused until the config exists, and a config is applied only after a secret is stored. `--config key=` (empty value) removes a key. Write a bool flag with an equal sign (`--enabled=false`); a value after a bool flag is a positional argument and the command refuses positional arguments.
- **The secret is read from stdin only** and is never an argument or a flag. Use a pipe (`printf` is a shell builtin, so the value is not in the process list) or a file only you can read (`< file`). `read -rs` is bash; do not `export` the variable. A terminal or `/dev/null` is refused; a pipe that you type into by hand (for example `exec -T` without a pipe) still shows what you type. One trailing newline is removed, surrounding spaces are removed by the handler, a value with more than one line is refused, and the longest secret is 1024 bytes.
- **In a container** keep stdin attached without a terminal: `docker compose exec -T <service> /app/bin/number-control provider ...` or `kubectl exec -i ... -- /app/bin/number-control provider ...`. The image is distroless (no shell) and holds `/app/bin/number-control`. The container needs `DATABASE_DSN` and `NUMBER_SECRET_ENCRYPTION_KEYS` in its environment; do not pass `--database_dsn` or `--number_secret_encryption_keys` as arguments.
- **`--actor-id`** is checked for the UUID format only, so `updated_by` and `secret_updated_by` record what the operator declares.
- **No restart** is needed: the service resolves a provider at every call and keeps no cache of it.
- **`secret set` and `secret delete` act on the stored row only.** The service reads the secret from the environment variable (`TELNYX_API_KEY`, `TWILIO_AUTH_TOKEN`), so the commands print one line to stderr saying that the stored value is not used. Rotate a secret in the deployment variable. `has_secret` and `credential_source` in the output follow the variable, so `has_secret` can stay true after `secret delete` while the variable is set. A provider that has no row yet is reported as not found.
- **Do not change the account** of a provider that still has numbers (see Provider Credentials and Secret Keys).
- **A malformed key list** makes every `provider` command fail (including `list` and `secret delete`); fix `NUMBER_SECRET_ENCRYPTION_KEYS` and run it again.

**Run service locally:**
```bash
DATABASE_DSN="user:pass@tcp(127.0.0.1:3306)/bin_manager" \
RABBITMQ_ADDRESS="amqp://guest:guest@localhost:5672" \
REDIS_ADDRESS="127.0.0.1:6379" \
NUMBER_SECRET_ENCRYPTION_KEYS="1:<base64 of 32 random bytes>" \
./bin/number-manager
```

## Deployment

bin-number-manager deploys via Komodo (VOIP-1348 Tier 2 rollout,
following the VOIP-1342/bin-call-manager pilot and VOIP-1347/Tier 1
pattern) instead of the older SSH + `versions.lock` (`ssh-deploy.sh`)
path.

- **Stack definition:** `bin-number-manager/komodo/docker-compose.yml`
  (git is the source of truth for structure; Komodo only executes it on
  request).
- **CI path:** `.circleci/scripts/render-image-tag.sh` substitutes
  the built image tag, then `.circleci/scripts/komodo-api-deploy.sh`
  pushes the file's content to Komodo and triggers a deploy, gated
  by the `bin-number-manager-deploy` job's poll/running checks.
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
| `prometheus_endpoint` | `PROMETHEUS_ENDPOINT` | `""` | Metrics path (e.g. `/metrics`) |
| `prometheus_listen_address` | `PROMETHEUS_LISTEN_ADDRESS` | `""` | Metrics listen address (e.g. `:2112`) |
| `number_secret_encryption_keys` | `NUMBER_SECRET_ENCRYPTION_KEYS` | `""` | Keys that encrypt the provider secrets stored in the DB by the admin API: `<version>:<base64 of 32 bytes>` entries separated by commas. The stored secrets are not used by the service; empty means no key and a secret cannot be stored |

## Prometheus Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `receive_request_process_time` | Histogram | `type`, `method` | RPC request processing duration |
| `receive_subscribe_event_process_time` | Histogram | `publisher`, `type` | Event processing duration |
| `request_external_process_time` | Histogram | `provider`, `method` | External provider API call duration |
| `number_create_total` | Counter | — | Total number purchase operations |
| `telnyx_number_create_total` | Counter | — | Telnyx-specific number purchases |
| `twilio_number_create_total` | Counter | — | Twilio-specific number purchases |
| `provider_unusable_total` | Counter | `provider`, `reason` | Resolutions of an enabled provider that could not be used (`no_secret`, `required_config`, `factory`) |
| `provider_purchase_total` | Counter | `provider`, `result` | Purchase attempts (`success`, `rejected`, `unconfirmed`) |

**Alert guidance:**
- `provider_unusable_total` increasing for an enabled provider → it cannot be used (`no_secret`, `required_config` or `factory`); check the provider view and the credential variables.
- `provider_purchase_total{result="unconfirmed"}` increasing → purchases stop without a fallback because the order result is unknown; check the provider for the orders.
- `telnyx_number_create_total` or `twilio_number_create_total` dropping while `number_create_total` stays constant → provider-specific failures; check provider API credentials.
- `request_external_process_time` p99 > 5s → provider API latency; purchases will time out.
- `receive_subscribe_event_process_time` high → cascade operations (customer/flow deletes) backing up.

## Provider Credentials and Secret Keys

The secret of a provider comes from an environment variable only; the stored ciphertext of `number_providers` is not read or decrypted.

| Provider | Secret | Config filled from a variable when the row has no value |
|---|---|---|
| telnyx | `TELNYX_API_KEY` | `connection_id` from `TELNYX_CONNECTION_ID`, `profile_id` from `TELNYX_PROFILE_ID` |
| twilio | `TWILIO_AUTH_TOKEN` | `sid` from `TWILIO_ACCOUNT_SID` (`trunk_sid` has no variable, set it in the row) |

- **Deployment variables**: the compose and k8s definitions pass these five variables (k8s as optional keys of the secret `voipbin`, so a missing key does not stop the pod). The same deployment variable names are used by the other services that share a vendor key (`TELNYX_API_KEY` is also read by bin-message-manager); the values must be equal.
- **Not configured**: an unset, empty or unresolved placeholder (`[[...]]`, the text of a deployment variable that was not resolved) secret variable gives `credential_source: none`; the provider cannot purchase, release or search. A placeholder config variable is treated as not set, so a required config that is then missing makes the provider unusable for a purchase only. At start the service logs once the state of each variable (`set`, `unset` or `placeholder`, never a value) and a warning when no provider is usable for a purchase (the check resolves each purchasable provider once, so the `provider_unusable_total` counter of an enabled provider without a secret rises by one at every start).
- **Row and variable**: the explicit `config` of a row wins over the variable; a key that the row does not have, or has empty, is filled from the variable. A deployment without any row works with the variables alone. The provider view shows the `config` of the row only, never a variable value, so a provider that is usable only through the variables shows an empty `config`; the value in use is the row value or, when the row has none, the variable. A value left in the row from before hides a changed variable, so check the row `config` when a variable change has no effect.
- **`credential_source` and `has_secret`** follow the effective secret: `env` with `has_secret: true` when the secret variable holds a value, else `none` with `false`. The stored ciphertext does not count. `db` and `db_undecryptable` are not produced any more. `key_version` and `secret_updated_by` describe the stored row (legacy, not used).
- **Stored secret**: `PUT` and `DELETE` of `/v1/number_providers/{name}/secret` and `number-control provider secret set|delete` still change the stored row only; they do not change the credential in use. Rotate a secret in the deployment variable and restart; a stored value left behind goes stale and is what an image from before the variables were read again would use.
- **Rolling update and rollback**: an image that reads the stored secret and one that reads the variables can run side by side, so the variable and the stored secret must hold the same value until the update ends. Roll back with the previous image together with its previous deployment definition; the new variables then stay unused.
- **A malformed key list** does not stop the service (other services depend on it). The error is logged without the key material and the service starts as if no key were configured: a new secret cannot be stored. Provider credentials are not affected. `number-control` refuses to start instead. Run `number-control migrate up` before the other `number-control` commands on a database the service has not migrated yet.
- **Key rotation** of `NUMBER_SECRET_ENCRYPTION_KEYS` only concerns the stored secrets, which are not used: add the new version and keep the old one while stored secrets exist.
- **Do not change the account** of a provider that still has numbers (a different token or, for Twilio, a different `sid`, in the variables or the row): releasing a number needs the account it was bought on.
- **Twilio** (disabled by default): the account SID is `sid` (row config or `TWILIO_ACCOUNT_SID`), `trunk_sid` = the Elastic SIP Trunk that receives the calls of the numbers (row config only), the secret is the account auth token (`TWILIO_AUTH_TOKEN`). The admin API refuses to store a secret until both row config keys exist. Twilio takes its secret from `TWILIO_AUTH_TOKEN` (`credential_source: env`). An enabled Twilio without that variable makes the `no_secret` counter rise at every resolution of the provider (a purchase, a release, an admin change). API key SIDs (`SK...`) are not supported. A malformed `sid` makes release and search fail, and makes purchases skip Twilio (`provider_unusable_total{reason="factory"}` rises at a purchase while Twilio is enabled); a malformed `trunk_sid` only refuses purchases (`PROVIDER_CONFIG_INVALID`). A Twilio number is delivered to the trunk only when the trunk is set, so do not enable Twilio before the trunk routes to this platform.
- **Twilio failures to watch**: `PROVIDER_AUTH_REJECTED` (HTTP 401, 403 or 404 on a purchase: the account setting is wrong, and the purchase falls back to the next provider so it would otherwise stay hidden), `PURCHASE_UNCONFIRMED` (an unclear answer: check the Twilio console for a number that was bought without a record, since Twilio bills it monthly and a retry does not find it), `ROLLBACK_RELEASE_FAILED`. A 429 (concurrency limit) stops the purchase without a fallback.
- **Before an image rollback** of this service, disable Twilio and check that no Twilio number is held: an older image cannot release Twilio numbers, so their records and the monthly charge would stay.
- Concurrent changes of the same provider's `config` by two admins are last write wins.
- The `tm_update` of a row changes with every change of the row, including a secret change; `secret_updated_by` and `tm_secret_update` tell the secret changes.
- A change of the `config` and a change of the secret of the same provider made at the same time can leave a row that holds a stored secret without a required config key (for Telnyx `connection_id`); the write checks keep the required keys of such a row. A provider whose variables are set and whose required config is missing shows `credential_source: env` and `usable: false` and is left out of the purchases. Set the config key in the row or the variable.
- The `config` of a row is used only while the provider has its secret variable. Without it the provider is `none` and the row config is ignored.
