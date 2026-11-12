# bin-email-manager — Operations

## Common Failure Modes

| Failure | Symptom | Likely Cause |
|---------|---------|-------------|
| Email not delivered | Status stays `initiated` | Both Mailgun and SendGrid APIs failing; check credentials |
| Mailgun auth failure | Primary provider rejected; SendGrid is tried | stored `mailgun.api_key` invalid, expired or not configured (check `email-control config list`: source `db`, `has_secret` true) |
| SendGrid auth failure | Fallback also fails; email stuck | stored `sendgrid.api_key` invalid, expired or not configured (check `email-control config list`: source `db`, `has_secret` true) |
| Attachment failure | Email stays `initiated`; no provider sends it | `bin-storage-manager` RPC failed, recording not found or download failed |
| Webhook status not updated | Email stays at `initiated` or `processed` | `bin-hook-manager` not forwarding; or provider webhook URL misconfigured; or (Mailgun) the stored `mailgun.webhook_signing_key` empty or different from the key in the Mailgun account (look for `Rejected the mailgun webhook` warnings, which name the reason only) |
| Cache stale | Old email data returned | Redis out of sync; restart clears cache |

## Debugging Guide

**Get an email (bypasses RabbitMQ):**
```bash
./bin/email-control email get --id <uuid>
```

**List emails for a customer:**
```bash
./bin/email-control email list --customer_id <uuid> [--limit 100] [--token]
```

**Stored provider credentials (database only):**
```bash
./bin/email-control config list     # key, source (db, none, ...), env_shadowed (always false); never a value
printf '%s' "$VALUE" | ./bin/email-control config set --key mailgun.api_key --actor-id <uuid>
./bin/email-control config clear --key mailgun.api_key --actor-id <uuid>
```
The value is read from stdin only. The tables are created by the daemon's migration at its start (on a local database run `./bin/email-control migrate up` first). `EMAIL_CONFIG_ENCRYPTION_KEYS` must be set for `set`.

The credentials are read from the database at the time of use. Consequence: a database outage makes a send (or a Mailgun webhook verification) fail with an error; the credential is never replaced by an empty key. A key without a stored row is not configured: the environment variables `SENDGRID_API_KEY`, `MAILGUN_API_KEY` and `MAILGUN_WEBHOOK_SIGNING_KEY` are not read by this service any more (VOIP-1721), so enter every credential with `email-control config set` or the admin API before the deploy of this version, and check them with `config list` (`source` must be `db`, `has_secret` true). `config clear` leaves the key not configured. The start logs a warning for each key that is not configured (key name only). The tables are created by the daemon at its start, so `email-control config` needs the daemon to have been deployed first.

**Delete an email:**
```bash
./bin/email-control email delete --id <uuid>
```

**Check provider webhook configuration:**
- SendGrid: Settings → Mail Settings → Event Webhook → URL must be `https://hook.voipbin.net/v1.0/emails/sendgrid`
- Mailgun: Sending → Webhooks → URL must be `https://hook.voipbin.net/v1.0/emails/mailgun` for the accepted, delivered, temporary failure, permanent failure, opened, clicked, unsubscribed and complained events. Opened and clicked events also need tracking enabled on the domain. Emails sent before the `voipbin_message_id` variable was added carry no email id, so their events are ignored.

**Run service locally:**
```bash
DATABASE_DSN="user:pass@tcp(127.0.0.1:3306)/voipbin" \
RABBITMQ_ADDRESS="amqp://guest:guest@localhost:5672" \
REDIS_ADDRESS="127.0.0.1:6379" \
./bin/email-manager
```

**Check email delivery via SendGrid Activity Feed:**
- SendGrid dashboard → Activity → search by recipient email or `provider_reference_id`

## Database Migrations

The service applies the migration of its table itself (`migrations/`, embedded) on every start, after the database connection and before the cache and RabbitMQ; version table `email_schema_migrations`.

| Situation | Behavior | Action |
|-----------|----------|--------|
| No pending migration | log `no pending migrations`, normal start | none |
| Pending migration | applied under a MySQL named lock, then normal start; two instances serialize | none |
| Database version higher than the image | skipped with a warning, normal start | none |
| Database marked dirty (a migration failed half way) | the start is refused, the container restarts in a loop | stop the container, check whether the failed file was applied, then `email-control migrate force <version>` against that database |
| Lock not obtained within 10 seconds | the start fails and the container restarts | none, a restart retries |

Check the state with `./bin/email-control migrate status` (database only). The new database must be created with `CHARACTER SET utf8 COLLATE utf8_general_ci`. File rules and checks: [docs/workflows/service-owned-migrations.md](../../docs/workflows/service-owned-migrations.md).

## Deployment

bin-email-manager deploys via Komodo (VOIP-1348 Tier 2 rollout, following
the VOIP-1342/bin-call-manager pilot and VOIP-1347/Tier 1 pattern)
instead of the older SSH + `versions.lock` (`ssh-deploy.sh`) path.

- **Stack definition:** `bin-email-manager/komodo/docker-compose.yml`
  (git is the source of truth for structure; Komodo only executes it on
  request).
- **CI path:** `.circleci/scripts/render-image-tag.sh` substitutes
  the built image tag, then `.circleci/scripts/komodo-api-deploy.sh`
  pushes the file's content to Komodo and triggers a deploy, gated
  by the `bin-email-manager-deploy` job's poll/running checks.
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
| `email_config_encryption_keys` | `EMAIL_CONFIG_ENCRYPTION_KEYS` | `""` (environment only, no flag) | Master key list for the stored secrets, `<version>:<base64 of 32 bytes>[,...]`, the entry with the highest version encrypts. Without it secrets can not be stored |

**Stored values:** the SendGrid API key, the Mailgun API key and the Mailgun webhook signing key are stored in `email_configs` (`email-control config set` or the admin API) and nowhere else. A change is used by the next call without a restart. `email-control config list` shows the source (`db` or `none`); `env_shadowed` is kept in the output for compatibility and is always false. A stored secret that can not be decrypted (wrong or missing master key) is an error. Rollback note: an image from before VOIP-1721 reads the three environment variables again when the key has no stored row, and a key rotated in the database only is not rotated for that image; keep the variables of the deployment in step until the rollback window has ended.

**Upgrade note:** a stored secret written by an earlier version (its ciphertext was bound to its row) cannot be decrypted by this version. Enter it again with `email-control config set` from the new image once every instance runs it.

**Mailgun webhook signature:** a Mailgun webhook is processed only when its `signature` (hex HMAC-SHA256 of `timestamp` + `token` with the signing key) matches and its `timestamp` is within 24 hours in the past and 5 minutes in the future. A request that fails the check changes no email status and publishes no `email_updated` event. If the signing key is empty, every Mailgun webhook is rejected and the service logs a warning at start (source `none`). If the key can not be read (database error) or can not be decrypted, the webhook is rejected as well.

**Key rotation:** only one signing key is supported. After the key is changed on the Mailgun side, store the new value with `email-control config set --key mailgun.webhook_signing_key`; the next webhook uses it, with no redeploy. Webhooks signed with the old key are rejected until the value is changed.

## Prometheus Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `receive_request_process_time` | Histogram | `type`, `method` | RPC request processing duration |

**Alert guidance:**
- `receive_request_process_time` p99 rising → emailhandler slow; check provider API latency and storage-manager RPC.
- No email events published in billing-manager → email billing broken; check `bin-manager.email-manager.event` exchange.
