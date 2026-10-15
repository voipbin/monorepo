# bin-conversation-manager

Multi-channel conversation management service. Handles SMS/MMS, LINE, and WhatsApp messaging threads, incoming/outgoing message delivery, and platform account credentials.

> Cross-cutting rules (verification workflow, branch/commit format, worktree usage, Alembic, RST sync) live in the root [CLAUDE.md](../CLAUDE.md).

## Docs index

- [docs/architecture.md](docs/architecture.md) — component layout, request routing table, event subscriptions
- [docs/domain.md](docs/domain.md) — Account/Conversation/Message models, message flows, flow variables
- [docs/dependencies.md](docs/dependencies.md) — local monorepo deps, external services, queue names
- [docs/operations.md](docs/operations.md) — config flags, Prometheus metrics, CLI tool, common commands

## Key concepts

- **Account** — platform credentials (LINE channel secret/token, SMS provider, WhatsApp Meta credentials); type: `sms` | `line` | `whatsapp`
- **Conversation** — thread between two parties identified by `(account_id, dialog_id)`; type: `message` | `line` | `whatsapp`
- **Message** — individual message; direction `incoming`/`outgoing`, status `progressing`/`done`/`failed`
- **DialogID** — external platform conversation identifier (LINE chatroom ID, WhatsApp recipient phone, SMS thread)
- Incoming LINE/WhatsApp messages arrive via `POST /v1/hooks`; incoming SMS/MMS arrive via `message-manager` event subscription

## Database migrations

The schema of `conversation_accounts`, `conversation_conversations`, `conversation_medias` and `conversation_messages` is owned by this service and applied at startup (golang-migrate, embedded with `go:embed`). It does not use Alembic. Tested engines: MariaDB and MySQL 8.0, so do not use engine specific syntax.

- Files: `migrations/NNNNNN_<table>_<verb>_<type>_<items>.up.sql` and `.down.sql` (six digits, from 000001 without gaps). `000001_baseline` is the schema at the time of the conversion. Version table: `conversation_schema_migrations`.
- `cmd/conversation-manager` runs `migrationhandler.Run` right after the database connection and before the cache connection, RabbitMQ and listening. A failure makes the process exit with a non-zero code.
- The rules for new migrations (one DDL statement per file, `-- DESTRUCTIVE:` and `-- DOWN-LOSSY:` markers, immutable merged files, expand/contract), the recovery from a dirty database and the verification scripts are the same for every service that owns its schema: see [docs/workflows/service-owned-migrations.md](../docs/workflows/service-owned-migrations.md) and the migration section of [bin-tag-manager/CLAUDE.md](../bin-tag-manager/CLAUDE.md).
- Structure checks: `go test ./migrations/`. Git relative check: `scripts/check-service-migrations.sh bin-conversation-manager` after `git fetch origin main`. Dynamic check on scratch containers: `ALEMBIC_VENV=<venv> scripts/verify-service-migrations.sh bin-conversation-manager` (repo root).
- `conversation-control migrate status|up|down|force <version>` talks to the database only (no Redis or RabbitMQ). The DSN comes from the `DATABASE_DSN` environment variable (the `--database_dsn` flag belongs to the root command only and is not accepted after `migrate`). Run `up|down|force` only against a local throwaway database; `down` is never run automatically and the baseline down is a no-op.
- Update `scripts/database_scripts_test/*.sql` (the sqlite test schema) by hand when the schema changes.
- Limit of the baseline: `CREATE TABLE IF NOT EXISTS` does not compare an existing table with the baseline, so a database whose tables differ from the Alembic head starts as version 1 without any warning. Compare the normalized dump of an existing database with the head before the first deployment (`REFERENCE_DUMP` in `scripts/verify-service-migrations.sh`).
- New database order: Alembic first, then this service. The reverse fails in the Alembic stream (`Table 'conversation_accounts' already exists`) and is unsupported while the Alembic stream still creates these tables.

## Common commands

```bash
# Full verification (mandatory before every commit)
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

# Build
go build -o bin/conversation-manager ./cmd/conversation-manager

# Test with coverage
go test -coverprofile cp.out -v $(go list ./...)

# Regenerate mocks
go generate ./...
```

## Testing pattern

gomock (go.uber.org/mock) + table-driven tests. Database test schemas in `scripts/database_scripts_test/`. Mock files co-located with handler packages.
