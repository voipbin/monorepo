# bin-number-manager — Operations

## Common Failure Modes

| Failure | Symptom | Likely Cause |
|---------|---------|-------------|
| Number purchase fails | 402 or provider error | Insufficient billing balance or provider API credentials invalid |
| Telnyx/Twilio API error | Number not created | The stored provider credentials (secret and `config` of `number_providers`) are wrong or expired |
| `could not purchase: no usable provider` | Every purchase fails | No enabled provider has a stored secret (the environment is not read); see `credential_source` and `usable` of `GET /v1/number_providers` and the `provider_unusable_total` metric |
| `PURCHASE_UNCONFIRMED` in the log | Purchase fails, no fallback | The order result is unknown; check the provider for an order of that number before retrying |
| `ROLLBACK_RELEASE_FAILED` in the log | A number is bought but not recorded | Release it at the provider by hand |
| `credential_source: db_undecryptable` | The provider is unusable, releases fail | The key version of the stored secret is not in `NUMBER_SECRET_ENCRYPTION_KEYS`; restore the key or delete the DB secret and store a new one |
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

**Provider settings and secrets (database and key list only, no Redis or RabbitMQ):**
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
- **`secret delete`** removes the stored secret and keeps the config; the `credential_source` of the output tells which credential the provider uses afterwards. A provider that has no row yet is reported as not found.
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
| `number_secret_encryption_keys` | `NUMBER_SECRET_ENCRYPTION_KEYS` | `""` | Keys that encrypt the provider secrets stored in the DB: `<version>:<base64 of 32 bytes>` entries separated by commas. The highest version encrypts, the version of the row decrypts. Empty means no key: secrets cannot be stored and no provider is usable |

## Prometheus Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `receive_request_process_time` | Histogram | `type`, `method` | RPC request processing duration |
| `receive_subscribe_event_process_time` | Histogram | `publisher`, `type` | Event processing duration |
| `request_external_process_time` | Histogram | `provider`, `method` | External provider API call duration |
| `number_create_total` | Counter | — | Total number purchase operations |
| `telnyx_number_create_total` | Counter | — | Telnyx-specific number purchases |
| `twilio_number_create_total` | Counter | — | Twilio-specific number purchases |
| `provider_unusable_total` | Counter | `provider`, `reason` | Resolutions of an enabled provider that could not be used (`no_secret`, `db_undecryptable`, `required_config`, `factory`) |
| `provider_purchase_total` | Counter | `provider`, `result` | Purchase attempts (`success`, `rejected`, `unconfirmed`) |

**Alert guidance:**
- `provider_unusable_total` increasing for an enabled provider → it cannot be used (`no_secret`, `db_undecryptable`, `required_config` or `factory`); check the provider view and the key list.
- `provider_purchase_total{result="unconfirmed"}` increasing → purchases stop without a fallback because the order result is unknown; check the provider for the orders.
- `telnyx_number_create_total` or `twilio_number_create_total` dropping while `number_create_total` stays constant → provider-specific failures; check provider API credentials.
- `request_external_process_time` p99 > 5s → provider API latency; purchases will time out.
- `receive_subscribe_event_process_time` high → cascade operations (customer/flow deletes) backing up.

## Provider Credentials and Secret Keys

The credentials of a provider come from its row in `number_providers` only: the stored secret and the `config` of the row. The service reads no provider credential from the environment (the `TELNYX_*` and `TWILIO_*` variables are not read; VOIP-1722). Without a stored secret the provider is `none` and cannot purchase, release or search. The admin view (`credential_source`) tells the state: `db`, `none` or `db_undecryptable`. At start the daemon logs a warning when no provider is usable for a purchase (the check resolves each purchasable provider once, so the `provider_unusable_total` counter of an enabled provider without a secret rises by one at every start).

**Before deploying** this version, store the Telnyx secret and `connection_id` (and the Twilio `sid`, `trunk_sid` and secret if Twilio numbers must still be released) with the Number Provider API; without them purchase, release and search fail.

- **Rollback**: an image from before VOIP-1722 reads the `TELNYX_*` / `TWILIO_*` variables again for a provider that has no stored secret (or after a secret deletion) as long as the deployment still injects them; a stored secret takes precedence in both images.
- **A malformed key list** does not stop the service (other services depend on it). The error is logged without the key material and the service starts as if no key were configured: stored secrets cannot be decrypted (the provider is `db_undecryptable`; no provider can purchase, release or search because no credential is read from the environment) and new secrets are refused. `number-control` refuses to start instead. Run `number-control migrate up` before the other `number-control` commands on a database the service has not migrated yet.
- **Key rotation**: (1) add the new version to `NUMBER_SECRET_ENCRYPTION_KEYS`, keep the old one, deploy; (2) enter the secret of each provider again (it is re-encrypted with the newest version; check `key_version` in the provider view); (3) when every provider shows the newest `key_version`, remove the old version. Removing a version first makes the rows of that version undecryptable. There is no re-encryption batch: the rows are few and the secrets are external values the admin holds.
- **Recovery** of a provider that is `db_undecryptable`: restore the missing key version, or delete the stored secret (`DELETE /v1/number_providers/{name}/secret` or `number-control provider secret delete`) and store a new one. After the deletion the provider is `none` and cannot release the numbers it sold until a new secret is stored, so store the new secret right away.
- **Upgrade note**: a secret stored by an earlier version (its ciphertext was bound to the provider name) cannot be decrypted by this version. Enter it again with `PUT /v1/number_providers/{name}/secret` once every instance runs the new image; until then the provider is `db_undecryptable`. A re-entered secret overwrites the stored one without decrypting it.
- **Do not change the account** of a provider that still has numbers (a different token or, for Twilio, a different `sid` for the same provider, by replacing or deleting the stored secret): releasing a number needs the account it was bought on. Moving the same account's token into the DB is fine.
- **Twilio** (disabled by default): the admin enters the config first (`sid` = the account SID, `trunk_sid` = the Elastic SIP Trunk that receives the calls of the numbers), then the secret (the account auth token); the secret is refused until both config keys exist. Twilio uses the stored credential only (`credential_source: db`). An enabled Twilio without a stored secret makes the `no_secret` counter rise at every resolution of the provider (a purchase, a release, an admin change). API key SIDs (`SK...`) are not supported. A malformed `sid` makes release and search fail, and makes purchases skip Twilio (`provider_unusable_total{reason="factory"}` rises at a purchase while Twilio is enabled); a malformed `trunk_sid` only refuses purchases (`PROVIDER_CONFIG_INVALID`). A Twilio number is delivered to the trunk only when the trunk is set, so do not enable Twilio before the trunk routes to this platform.
- **Twilio failures to watch**: `PROVIDER_AUTH_REJECTED` (HTTP 401, 403 or 404 on a purchase: the account setting is wrong, and the purchase falls back to the next provider so it would otherwise stay hidden), `PURCHASE_UNCONFIRMED` (an unclear answer: check the Twilio console for a number that was bought without a record, since Twilio bills it monthly and a retry does not find it), `ROLLBACK_RELEASE_FAILED`. A 429 (concurrency limit) stops the purchase without a fallback.
- **Before an image rollback** of this service, disable Twilio and check that no Twilio number is held: an older image cannot release Twilio numbers, so their records and the monthly charge would stay.
- Concurrent changes of the same provider's `config` by two admins are last write wins.
- The `tm_update` of a row changes with every change of the row, including a secret change; `secret_updated_by` and `tm_secret_update` tell the secret changes.
- A change of the `config` and a change of the secret of the same provider made at the same time can leave a DB-secret row without a required config key (for Telnyx `connection_id`). The provider then shows `credential_source: db` and `usable: false` and is left out of the purchases. Set the config key again, or delete the stored secret.
- The `config` of a row is used only while the provider has a stored secret. For a row without a secret the `config` shown in the provider view is ignored (`credential_source: none`).
