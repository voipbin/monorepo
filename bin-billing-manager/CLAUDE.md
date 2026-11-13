# bin-billing-manager

Manages billing accounts, balance tracking, billing record creation, and Paddle payment integration for VoIPbin.

> Cross-cutting rules (verification workflow, branch/commit format, worktrees, Alembic, RST sync) live in the root [CLAUDE.md](../CLAUDE.md).

## Docs

- [docs/architecture.md](docs/architecture.md) — components, layers, request routing
- [docs/domain.md](docs/domain.md) — Account, Billing, FailedEvent entities; billing event flows; Paddle integration rules
- [docs/dependencies.md](docs/dependencies.md) — subscribed events, monorepo deps, external services
- [docs/operations.md](docs/operations.md) — failure modes, debugging, config reference, metrics

## CRITICAL: Access Control

**Billing and Billing Account resources require `CustomerAdmin` permission ONLY.** Authorization is enforced by `bin-api-manager`, not by this service.

## CRITICAL: Unlimited-Plan Bypass

Accounts with `plan_type = unlimited` always pass balance checks — do not add metering logic that bypasses this check.

## Key Commands

```bash
# Build
go build -o ./bin/billing-manager ./cmd/billing-manager/
go build -o ./bin/billing-control ./cmd/billing-control/

# Test
go test -coverprofile cp.out -v $(go list ./...)

# Verification (mandatory before commit)
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

# billing-control — direct DB/cache access, bypasses RabbitMQ
billing-control account get --id <uuid>
billing-control account add-balance --id <uuid> --amount 10.00
billing-control account subtract-balance --id <uuid> --amount 5.00
billing-control billing list --customer-id <uuid> --limit 20

# Run locally
DATABASE_DSN="user:pass@tcp(127.0.0.1:3306)/bin_manager" \
RABBITMQ_ADDRESS="amqp://guest:guest@localhost:5672" \
REDIS_ADDRESS="127.0.0.1:6379" \
./bin/billing-manager
```

## Database Migrations

The schema of `billing_accounts`, `billing_billings`, `billing_failed_events`, `billing_configs` and `billing_config_histories` is owned by this service and applied at startup (golang-migrate, embedded with `go:embed`, shared code in `bin-common-handler/pkg/migrationhandler`). It does not use Alembic. Tested engines: MariaDB and MySQL 8.0, so do not use engine specific syntax. The full rules (file naming, one DDL statement per file, `-- DESTRUCTIVE:` and `-- DOWN-LOSSY:` markers, immutable merged files, expand/contract, baseline text rules) are in [docs/workflows/service-owned-migrations.md](../docs/workflows/service-owned-migrations.md).

- Files: `migrations/NNNNNN_<table>_<verb>_<type>_<items>.up.sql` and `.down.sql`. `000001_baseline` is the schema at the time of the conversion (`CREATE TABLE IF NOT EXISTS`, no table options; its down is the no-op `SELECT 1`). Version table: `billing_schema_migrations`.
- `cmd/billing-manager` runs `migrationhandler.Run` right after the database connection, before the cache and RabbitMQ. Failure stops the start (non-zero exit).
- `billing-control migrate status|up|down|force <version>` talks to the database only (no Redis or RabbitMQ). Run `up`, `down` and `force` only against a local throwaway database or as the recovery step in [docs/operations.md](docs/operations.md).
- A new database is created with `CHARACTER SET utf8 COLLATE utf8_general_ci` (like the install scripts do).
- Update `scripts/database_scripts_test/table_billing*.sql` in this service (the sqlite test schema) by hand when the schema changes.
- Verification: `go test ./migrations/` (structure) and, from the repo root, `scripts/verify-service-migrations.sh bin-billing-manager` (both engines).

## Paddle credentials are read per request

The Paddle API key and the Paddle webhook secret are two stored settings (`paddle.api_key`, `paddle.webhook_secret_key`, table `billing_configs`, history in `billing_config_histories`). They are read through `internal/billingconfig` for every use, from the database only: the keys declare no environment variable, and `billingconfig.New` passes a nil environment lookup to the shared `confighandler`. Never read `PADDLE_API_KEY` or `PADDLE_WEBHOOK_SECRET_KEY` directly, never add an environment fallback, and never cache the values. A database read error or a decrypt error is an error (`paddlehandler.ErrKeyUnavailable`) and is never turned into "not configured". A key without a stored value is not configured: the supplier returns an empty value without an error, and the caller answers with `paddlehandler.ErrCredentialNotConfigured`.

- The API key is read by `paddlehandler.CreatePortalSession` before the HTTP request is built; a read error or a key that is not stored means no request leaves.
- `processV1HooksPaddlePost` calls `paddleHandler.VerifyWebhook` after the hook envelope is parsed. This is the only signature check (the hook service does not verify it). No stored secret is `ErrCredentialNotConfigured` and a 500 (fail closed), a rejected signature is 401, an unreadable secret is 500. Never log the signature or the body in this path: use a new `WithField("func", ...)` entry, not the entry that holds `"request": m`.
- They can be written with `billing-control config` or the admin API (`/service_configs/billing`, served as `/v1/configs` by the shared `configrpc`). That route is handled at the top of `processRequest`, before the request log entry exists, and must never log the request, because the body of a secret write is the secret. The master key list is `BILLING_CONFIG_ENCRYPTION_KEYS` (environment only, no flag).
- The `PaddleAPIKey` config field and the `--paddle_api_key` flag were removed.
- The webhook secret and the API key must be stored before the version that rejects webhooks without a stored secret is deployed. See docs/operations.md, "Paddle credentials", for the deployment order and the rollback rule.

## Architecture Summary

- **ListenHandler**: consumes `bin-manager.billing-manager.request`; routes to accounthandler / billinghandler
- **SubscribeHandler**: consumes events from call/message/email/number/customer/tts managers; creates billing records
- **FailedEventHandler**: persists and retries billing ops that fail; hard-deletes on success
- **Cache**: Redis fronts MySQL for account lookups; invalidated on mutations

## Testing

Uses `go.uber.org/mock` (gomock). Table-driven tests. Mocks co-located with interfaces (`mock_main.go`).

```bash
go test -v ./pkg/accounthandler -run Test_IsValidBalance
```

## Database Tables

- `billing_accounts` — billing accounts with balance (soft-delete with `tm_delete`)
- `billing_billings` — billing records per event (soft-delete with `tm_delete`)
- `billing_failed_events` — retry queue (hard-deleted on success)
- `billing_configs`: stored platform settings (the Paddle credentials, secrets encrypted)
- `billing_config_histories`: change history of the stored settings (no secret material)
