# bin-email-manager — Operations

## Common Failure Modes

| Failure | Symptom | Likely Cause |
|---------|---------|-------------|
| Email not delivered | Status stays `initiated` | Both Mailgun and SendGrid APIs failing; check credentials |
| Mailgun auth failure | Primary provider rejected; SendGrid is tried | `MAILGUN_API_KEY` invalid, expired, not set or an unresolved placeholder (the start log shows the state of the variable) |
| SendGrid auth failure | Fallback also fails; email stuck | `SENDGRID_API_KEY` invalid, expired, not set or an unresolved placeholder (the start log shows the state of the variable) |
| Attachment failure | Email stays `initiated`; no provider sends it | `bin-storage-manager` RPC failed, recording or file not found or deleted after the email was created, or download failed |
| Attachment refused | `POST /v1/emails` answers 400, 404 or 503 and no email is created | 400: more than 5 attachments, more than 18 MiB in total, duplicate, unknown `reference_type`, or a file of type `rag`; 404: the recording or file does not exist, is deleted or belongs to another customer; 503: storage did not answer within 2 seconds (the request can be retried) |
| Webhook status not updated | Email stays at `initiated` or `processed` | `bin-hook-manager` not forwarding; or provider webhook URL misconfigured; or (Mailgun) `MAILGUN_WEBHOOK_SIGNING_KEY` not set or different from the key in the Mailgun account (look for `Rejected the mailgun webhook` warnings, which name the reason only) |
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

The credentials are read from the environment at the time of use (see the table of variables below); a database outage does not affect a send. A variable that is not set, is empty, or holds an unresolved placeholder (a value that starts with `[[`, the text of a deployment variable that does not exist) means the key is not configured. The placeholder is never sent to a provider. The start logs one line per variable: set (info), not set (warning) and unresolved placeholder (warning); the variable name and its state only, never a value. In the compose definition the value is `[[BIN_MANAGER__<variable name>]]` with the same name as the variable; a missing deployment variable leaves the placeholder. In Kubernetes `SENDGRID_API_KEY` and `MAILGUN_API_KEY` are optional keys of the `voipbin` secret (a missing key means not configured); `MAILGUN_WEBHOOK_SIGNING_KEY` is passed by the compose definition only.

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
| (no flag) | `SENDGRID_API_KEY` | not configured | SendGrid API key (environment only, never a flag, never logged) |
| (no flag) | `MAILGUN_API_KEY` | not configured | Mailgun API key (environment only, never a flag, never logged) |
| (no flag) | `MAILGUN_WEBHOOK_SIGNING_KEY` | not configured | Key that verifies the Mailgun webhook signature (environment only, never a flag, never logged). Not set: every Mailgun webhook is rejected |
| `email_config_encryption_keys` | `EMAIL_CONFIG_ENCRYPTION_KEYS` | `""` (environment only, no flag) | Not read by the code. The deployment definition still passes it; a later change removes the line |

**Stored settings:** nothing is stored in the database. The tables `email_configs` and `email_config_histories` are no longer read or written by the code.

**Rollout and rollback:** every variable above must exist in the deployment. Rollback is the previous image together with the previous deployment definition; the previous image reads the same variables.

**Mailgun webhook signature:** a Mailgun webhook is processed only when its `signature` (hex HMAC-SHA256 of `timestamp` + `token` with the signing key) matches and its `timestamp` is within 24 hours in the past and 5 minutes in the future. A request that fails the check changes no email status and publishes no `email_updated` event. If the signing key is not configured, every Mailgun webhook is rejected and the service logs a warning at start.

**Key rotation:** only one signing key is supported. After the key is changed on the Mailgun side, change `MAILGUN_WEBHOOK_SIGNING_KEY` in the deployment and restart the replicas. Webhooks signed with the old key are rejected until then.

## Prometheus Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `receive_request_process_time` | Histogram | `type`, `method` | RPC request processing duration |

**Alert guidance:**
- `receive_request_process_time` p99 rising → emailhandler slow; check provider API latency and storage-manager RPC.
- No email events published in billing-manager → email billing broken; check `bin-manager.email-manager.event` exchange.
