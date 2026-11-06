# bin-email-manager

Email delivery service — sends emails via Mailgun (primary) and SendGrid (failover), tracks delivery status via provider webhooks, and supports call recording attachments via bin-storage-manager.

> Cross-cutting rules (verification workflow, branch/commit format, worktrees, Alembic, RST sync) live in the root [CLAUDE.md](../CLAUDE.md).

## Docs

- [docs/architecture.md](docs/architecture.md) — components, layers, request routing
- [docs/domain.md](docs/domain.md) — Email, Attachment entities; status lifecycle; provider failover; webhook processing
- [docs/dependencies.md](docs/dependencies.md) — events published, outbound RPCs, monorepo deps
- [docs/operations.md](docs/operations.md) — failure modes, debugging, config reference, metrics

## CRITICAL: Provider Failover Order

Mailgun is attempted first; SendGrid is the fallback. If both fail, the email stays at `initiated` status with an empty `provider_type`. Both API keys must be configured. After a successful send, `provider_type` and `provider_reference_id` are recorded together.

## CRITICAL: No SubscribeHandler

This service does not consume events. Delivery status updates come via `POST /v1/hooks` (forwarded from bin-hook-manager). Do not add a subscribehandler without a clear use case.

## Key Commands

```bash
# Build
go build -o bin/email-manager ./cmd/email-manager

# Test
go test -coverprofile cp.out -v $(go list ./...)

# Verification (mandatory before commit)
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

# email-control — direct DB/cache access, bypasses RabbitMQ
./bin/email-control email get --id <uuid>
./bin/email-control email list --customer_id <uuid> [--limit 100] [--token]
./bin/email-control email delete --id <uuid>

# Stored provider credentials (database only; the value is read from stdin, never from an argument)
./bin/email-control config list
./bin/email-control config set --key mailgun.api_key --actor-id <uuid> < value-file
./bin/email-control config clear --key mailgun.api_key --actor-id <uuid>

# Migration status (database only, no Redis or RabbitMQ)
./bin/email-control migrate status

# Run locally
DATABASE_DSN="user:pass@tcp(127.0.0.1:3306)/voipbin" \
RABBITMQ_ADDRESS="amqp://guest:guest@localhost:5672" \
REDIS_ADDRESS="127.0.0.1:6379" \
SENDGRID_API_KEY="SG.xxx" \
MAILGUN_API_KEY="xxx" \
MAILGUN_WEBHOOK_SIGNING_KEY="xxx" \
./bin/email-manager
```

## Architecture Summary

- **No SubscribeHandler** — purely RPC-driven + inbound webhooks
- **ListenHandler**: consumes `QueueNameEmailRequest`
- **Provider engines**: `engine_sendgrid.go` and `engine_mailgun.go`; tried in order (Mailgun, then SendGrid)
- **Attachments**: resolved via RPC to bin-storage-manager before send
- **Database**: Squirrel SQL builder (not raw SQL)

## Testing

Uses `go.uber.org/mock` (gomock). Table-driven tests. Mocks for emailhandler, engines, dbhandler.

```bash
go test -v ./pkg/emailhandler -run Test_Send
```

## Webhook Configuration

External provider webhooks (via bin-hook-manager):
- SendGrid: `https://hook.voipbin.net/v1.0/emails/sendgrid`
- Mailgun: `https://hook.voipbin.net/v1.0/emails/mailgun`

## Database migrations

The schema of `email_emails` is owned by this service and applied at startup (golang-migrate, embedded with `go:embed`). It does not use Alembic. Tested engines: MariaDB and MySQL 8.0, so do not use engine specific syntax.

- Files: `migrations/NNNNNN_<description>.up.sql` and `.down.sql` (six digits, from 000001 without gaps). `000001_baseline` is the schema at the time of the conversion (`CREATE TABLE IF NOT EXISTS`, no table options, down is the no-op `SELECT 1`). Version table: `email_schema_migrations`.
- `cmd/email-manager` runs `migrationhandler.Run` in `createDBHandler`, right after the database connection and before the cache and RabbitMQ. Failure stops the start (non-zero exit).
- `email-control migrate status|up|down|force <version>` talks to the database only. Run `up`, `down` and `force` only against a local throwaway database.
- The migration code is the shared `bin-common-handler/pkg/migrationhandler`. Rules for new files (one DDL statement per file, `-- DESTRUCTIVE:` and `-- DOWN-LOSSY:` markers, immutable merged files, expand/contract), the checks and the recovery steps are in [docs/workflows/service-owned-migrations.md](../docs/workflows/service-owned-migrations.md).
- A new database is created with `CHARACTER SET utf8 COLLATE utf8_general_ci` (like the install scripts do); other defaults make the table definitions differ from the Alembic head.
- New database order: Alembic first, then this service. The reverse fails in the Alembic stream (`Table 'email_emails' already exists`) and is unsupported while the Alembic stream still creates the email table.
- Update `scripts/database_scripts/table_*.sql` (the sqlite test schema) by hand when the schema changes.
- Structure checks: `go test ./migrations/`. Git relative check: `scripts/check-service-migrations.sh bin-email-manager` after `git fetch origin main`. Dynamic check on scratch containers: `ALEMBIC_VENV=<venv> scripts/verify-service-migrations.sh bin-email-manager` (repo root).

## Database Tables

- `email_emails` — email records with provider info, status, attachments (soft-delete with `tm_delete`)
- `email_configs` — platform-wide stored settings: provider credentials, encrypted with the master key. One row per key (`config_key`), optimistic lock by `revision`
- `email_config_histories` — one row per change (action, actor, key versions, time). Never holds a value, a hash or a length of a secret

## Provider credentials

The SendGrid API key, the Mailgun API key and the Mailgun webhook signing key are resolved when they are used: the stored value (`email_configs`) first, then the environment variable (`SENDGRID_API_KEY`, `MAILGUN_API_KEY`, `MAILGUN_WEBHOOK_SIGNING_KEY`), so a database without rows behaves exactly as before. A change made with `email-control config set` is used by the next call without a restart. The key declarations are in `internal/configkeys`; the shared library is `bin-common-handler/pkg/confighandler`.

- A stored secret that can not be decrypted is an error; the environment value is never used instead. A database read error is an error as well, and a Mailgun webhook is then rejected (the provider retries).
- Master key list: environment variable `EMAIL_CONFIG_ENCRYPTION_KEYS` (`<version>:<base64 of 32 bytes>[,...]`, the entry with the highest version encrypts). It is environment only, there is no flag. Without it the service starts, secrets can not be stored and a stored secret is reported as undecryptable.
- Adding a setting means a new key in `internal/configkeys` and the golden test there; a key never changes its kind.
