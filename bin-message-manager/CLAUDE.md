# bin-message-manager

SMS and MMS messaging service — sends messages via Telnyx and MessageBird (both SMS and MMS go to Telnyx first, and to MessageBird only for the targets Telnyx refused), counts the segments of a text itself (3GPP TS 23.038), sends a text of more than one segment as a text-only MMS, tracks per-target delivery status from the Telnyx receipts, and processes inbound provider webhooks.

> Cross-cutting rules (verification workflow, branch/commit format, worktrees, Alembic, RST sync) live in the root [CLAUDE.md](../CLAUDE.md).

## Docs

- [docs/architecture.md](docs/architecture.md) — components, layers, request routing
- [docs/domain.md](docs/domain.md) — Message, Target entities; send flow; provider selection; webhook processing
- [docs/dependencies.md](docs/dependencies.md) — events published, outbound RPCs, monorepo deps
- [docs/operations.md](docs/operations.md) — failure modes, debugging, config reference, metrics

## CRITICAL: Balance Check Before Send

`POST /v1/messages` calls `bin-billing-manager` for balance validation before creating the message. Do not bypass this check.

## CRITICAL: Async Provider Dispatch

After message creation, provider send runs in a **goroutine** — it is non-blocking to the RPC caller. The caller receives success immediately after the message record is created; actual delivery is asynchronous.

## Database migrations

The schema of the tables `message_messages`, `message_configs` and `message_config_histories` is owned by this service and applied at startup (golang-migrate, embedded with `go:embed`). It does not use Alembic. The full rules, the baseline text rules and the verification script are in [docs/workflows/service-owned-migrations.md](../docs/workflows/service-owned-migrations.md); `bin-tag-manager/CLAUDE.md` is the reference service.

- Files: `migrations/NNNNNN_<table>_<verb>_<type>_<items>.up.sql` and `.down.sql` (six digits, from 000001 without gaps). `000001_baseline` is the schema at the time of the conversion. Version table: `message_schema_migrations`.
- `cmd/message-manager` runs `migrationhandler.Run` right after the database connection, before the cache and RabbitMQ. Failure stops the start (non-zero exit).
- Every new migration: one DDL statement per file, an up and a down, merged files are never changed, old code must work on the new schema (expand/contract), `-- DESTRUCTIVE:` and `-- DOWN-LOSSY:` markers as described in the workflow document. Update `scripts/database_scripts_test/*.sql` (the sqlite test schema) by hand when the schema changes.
- Checks: `go test ./migrations/` (structure) and, from the repo root, `scripts/verify-service-migrations.sh bin-message-manager` (scratch containers, MariaDB and MySQL 8.0).
- Recovery from a dirty database: `message-control migrate force <version>` (see `docs/operations.md`). A file that failed on every environment may be fixed in place only when it is the highest number, never succeeded anywhere, and the PR body says `MIGRATION-FIX: N` with the failure log. Run `message-control migrate up|down|force` only against a local throwaway database.
- New database: the service creates its tables at startup (the baseline is `CREATE TABLE IF NOT EXISTS`); no other schema tool runs first.

## Key Commands

```bash
# Build
go build -o ./bin/ ./cmd/...

# Test
go test -coverprofile cp.out -v $(go list ./...)

# Verification (mandatory before commit)
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

# message-control config — the stored provider tokens (database and master key only; the value is read from stdin)
./bin/message-control config list
printf '%s' "$NEW_TOKEN" | ./bin/message-control config set --key provider.telnyx.auth_token --actor-id <uuid>
./bin/message-control config clear --key provider.telnyx.auth_token --actor-id <uuid>

# message-control — direct DB/cache access, bypasses RabbitMQ (the migrate command opens only the database)
./bin/message-control migrate status
./bin/message-control message get --id <uuid>
./bin/message-control message list --customer_id <uuid> [--limit 100] [--token]
./bin/message-control message delete --id <uuid>

# Run locally
DATABASE_DSN="user:pass@tcp(127.0.0.1:3306)/voipbin" \
RABBITMQ_ADDRESS="amqp://guest:guest@localhost:5672" \
REDIS_ADDRESS="127.0.0.1:6379" \
AUTHTOKEN_MESSAGEBIRD="token" \
AUTHTOKEN_TELNYX="token" \
./bin/message-manager
```

## CRITICAL: Provider tokens are read per request

The MessageBird and Telnyx tokens are stored settings (`message_configs`, order: stored value, then `AUTHTOKEN_MESSAGEBIRD` / `AUTHTOKEN_TELNYX`, then empty). `requestexternal` receives a supplier function (`messageconfig.MessagebirdToken`, `TelnyxToken`) and calls it for every provider request. Never read the token from `config.Get()` in a send path and never cache it: a changed token must be used by the next request. A token that can not be read is an error (never replaced by the environment value). The Telnyx token here is not the one of `bin-number-manager`; a Telnyx key rotation needs both (see docs/operations.md, "Provider tokens"). The admin API route `/v1/configs` is handled before the request log entry exists and logs nothing of the request (the body of a secret write is the secret).

## Architecture Summary

- **No SubscribeHandler** — webhook events arrive via `POST /v1/hooks` from bin-hook-manager
- **ListenHandler**: consumes `bin-manager.message-manager.request`
- **Provider dispatch**: an ordered chain, Telnyx first and MessageBird for the targets Telnyx definitively refused (see `pkg/messagehandler/dispatch.go`); a text of more than one segment is promoted to an MMS (`pkg/segmenthandler` counts the segments); goroutine-based async send
- **Database**: Squirrel SQL builder (not raw SQL); Targets tracked separately per recipient

## Testing

Uses `go.uber.org/mock` (gomock). Table-driven tests. Mocks for messagehandler, providers, dbhandler.

```bash
go test -v ./pkg/messagehandler -run Test_Send
```

## Webhook Configuration

Provider webhook endpoints (via bin-hook-manager):
- MessageBird: `https://hook.voipbin.net/v1.0/messages/messagebird`
- Telnyx: `https://hook.voipbin.net/v1.0/messages/telnyx`
