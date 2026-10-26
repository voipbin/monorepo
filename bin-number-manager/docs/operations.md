# bin-number-manager — Operations

## Common Failure Modes

| Failure | Symptom | Likely Cause |
|---------|---------|-------------|
| Number purchase fails | 402 or provider error | Insufficient billing balance or provider API credentials invalid |
| Telnyx/Twilio API error | Number not created | Provider credentials (`telnyx_token`, `twilio_sid`/`twilio_token`) wrong or expired |
| `could not purchase: no usable provider` | Every purchase fails | No enabled provider has a secret; see `credential_source` and `usable` of `GET /v1/number_providers` and the `provider_unusable_total` metric |
| `PURCHASE_UNCONFIRMED` in the log | Purchase fails, no fallback | The order result is unknown; check the provider for an order of that number before retrying |
| `ROLLBACK_RELEASE_FAILED` in the log | A number is bought but not recorded | Release it at the provider by hand |
| `credential_source: db_undecryptable` | The provider is unusable, releases fail | The key version of the stored secret is not in `NUMBER_SECRET_ENCRYPTION_KEYS`; restore the key or delete the DB secret |
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

On a new database run the Alembic stream first and start the service afterwards. The reverse order is not supported: if the service creates `number_numbers` first, `alembic upgrade head` fails with `already exists` (checked by `scripts/verify-service-migrations.sh`). The service account also needs permission to create tables (for the version table). A new database is created with `CHARACTER SET utf8 COLLATE utf8_general_ci` (like the install scripts do).

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

**Run service locally:**
```bash
DATABASE_DSN="user:pass@tcp(127.0.0.1:3306)/bin_manager" \
RABBITMQ_ADDRESS="amqp://guest:guest@localhost:5672" \
REDIS_ADDRESS="127.0.0.1:6379" \
TELNYX_TOKEN="<token>" \
TELNYX_CONNECTION_ID="<connection-id>" \
TELNYX_PROFILE_ID="<profile-id>" \
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
| `telnyx_token` | `TELNYX_TOKEN` | required if Telnyx used and not stored in the DB | Telnyx API authentication token |
| `telnyx_connection_id` | `TELNYX_CONNECTION_ID` | required if Telnyx used | Telnyx connection ID for call routing |
| `telnyx_profile_id` | `TELNYX_PROFILE_ID` | required if Telnyx used | Telnyx messaging profile ID |
| `twilio_sid` | `TWILIO_SID` | required for Twilio release | Twilio account SID. The environment credential can release numbers but never purchase (see Provider Credentials) |
| `twilio_token` | `TWILIO_TOKEN` | required for Twilio release | Twilio auth token. The surrounding whitespace is removed |
| `number_secret_encryption_keys` | `NUMBER_SECRET_ENCRYPTION_KEYS` | `""` | Keys that encrypt the provider secrets stored in the DB: `<version>:<base64 of 32 bytes>` entries separated by commas. The highest version encrypts, the version of the row decrypts. Empty means no key: secrets cannot be stored and the environment credentials are used |

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

A provider uses one credential source at a time: with a stored secret in `number_providers` the secret and the `config` of the row are used and the environment is ignored; without one, the `TELNYX_*` / `TWILIO_*` variables are used. The admin view (`credential_source`) tells which one is active: `db`, `env`, `none` or `db_undecryptable`.

- **A malformed key list** does not stop the service (other services depend on it). The error is logged without the key material and the service starts as if no key were configured: stored secrets cannot be decrypted (the provider is `db_undecryptable`, no fallback to the environment) and new secrets are refused. `number-control` refuses to start instead. Run `number-control migrate up` before the other `number-control` commands on a database the service has not migrated yet.
- **Key rotation**: (1) add the new version to `NUMBER_SECRET_ENCRYPTION_KEYS`, keep the old one, deploy; (2) enter the secret of each provider again (it is re-encrypted with the newest version; check `key_version` in the provider view); (3) when every provider shows the newest `key_version`, remove the old version. Removing a version first makes the rows of that version undecryptable. There is no re-encryption batch: the rows are few and the secrets are external values the admin holds.
- **Recovery** of a provider that is `db_undecryptable`: restore the missing key version, or delete the stored secret (`DELETE /v1/number_providers/{name}/secret`), which returns the provider to the environment credentials.
- **Do not change the account** of a provider that still has numbers (a different token or, for Twilio, a different `sid` for the same provider, by replacing or deleting the stored secret): releasing a number needs the account it was bought on. Moving the same account's token into the DB is fine.
- **Twilio** (disabled by default): the admin enters the config first (`sid` = the account SID, `trunk_sid` = the Elastic SIP Trunk that receives the calls of the numbers), then the secret (the account auth token); the secret is refused until both config keys exist. Twilio purchases use the stored credential only (`credential_source: db`); the environment `TWILIO_SID`/`TWILIO_TOKEN` can release numbers but never purchase, and then the provider shows `usable: false`. An enabled Twilio on the environment credential makes the `required_config` counter rise at every resolution of the provider (a purchase, a release, an admin change). API key SIDs (`SK...`) are not supported. A malformed `sid` makes release and search fail, and makes purchases skip Twilio (`provider_unusable_total{reason="factory"}` rises at a purchase while Twilio is enabled); a malformed `trunk_sid` only refuses purchases (`PROVIDER_CONFIG_INVALID`). A Twilio number is delivered to the trunk only when the trunk is set, so do not enable Twilio before the trunk routes to this platform.
- **Twilio failures to watch**: `PROVIDER_AUTH_REJECTED` (HTTP 401, 403 or 404 on a purchase: the account setting is wrong, and the purchase falls back to the next provider so it would otherwise stay hidden), `PURCHASE_UNCONFIRMED` (an unclear answer: check the Twilio console for a number that was bought without a record, since Twilio bills it monthly and a retry does not find it), `ROLLBACK_RELEASE_FAILED`. A 429 (concurrency limit) stops the purchase without a fallback.
- **Before an image rollback** of this service, disable Twilio and check that no Twilio number is held: an older image cannot release Twilio numbers, so their records and the monthly charge would stay.
- Concurrent changes of the same provider's `config` by two admins are last write wins.
- The `tm_update` of a row changes with every change of the row, including a secret change; `secret_updated_by` and `tm_secret_update` tell the secret changes.
- A change of the `config` and a change of the secret of the same provider made at the same time can leave a DB-secret row without a required config key (for Telnyx `connection_id`). The provider then shows `credential_source: db` and `usable: false` and is left out of the purchases. Set the config key again, or delete the stored secret.
- The `config` of a row is used only while the provider has a stored secret. With the environment credentials the `config` shown in the provider view is ignored; `credential_source` tells which one applies.
