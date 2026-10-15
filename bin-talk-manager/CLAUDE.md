# bin-talk-manager

Manages chat sessions, messages, participants, and emoji reactions for VoIPbin. RabbitMQ RPC service backed by MySQL + Redis.

> Cross-cutting rules (verification, branch/commit format, worktrees, Alembic, RST sync) live in the root [CLAUDE.md](../CLAUDE.md).

## Docs

- [docs/architecture.md](docs/architecture.md) — component overview, layer responsibilities, request routing
- [docs/domain.md](docs/domain.md) — Chat, Participant, Message entities; business rules
- [docs/dependencies.md](docs/dependencies.md) — upstream services and infrastructure
- [docs/operations.md](docs/operations.md) — failure modes, debugging, configuration, metrics

## Common Commands

```bash
# Build
go build -o ./bin/talk-manager ./cmd/talk-manager
go build -o ./bin/talk-control ./cmd/talk-control

# Test
go test ./...
go generate ./...   # regenerate mocks before testing

# Verification (mandatory before commit)
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

## Database migrations

The schema of `talk_chats`, `talk_messages` and `talk_participants` is owned by this service and applied at startup (golang-migrate, embedded with `go:embed`). It does not use Alembic. Tested engines: MariaDB 12.3 and MySQL 8.0, so do not use engine specific syntax. The general rules (one DDL statement per file, `-- DESTRUCTIVE:` and `-- DOWN-LOSSY:` markers, immutable merged files, expand/contract, baseline rules) are in [docs/workflows/service-owned-migrations.md](../docs/workflows/service-owned-migrations.md); `bin-tag-manager/CLAUDE.md` is the reference.

- Files: `migrations/NNNNNN_<table>_<verb>_<type>_<items>.up.sql` and `.down.sql`. `000001_baseline` is the schema at the time of the conversion (three `CREATE TABLE IF NOT EXISTS` statements, `DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci` written out, foreign keys named `<table>_ibfk_N`, `json` columns). The baseline down is a no-op.
- `cmd/talk-manager` runs the migrations right after the database connection, before the message broker and the Redis client. Failure stops the start. Version table: `talk_schema_migrations`.
- `talk-control migrate status|up|down|force <version>` talks to the database only. Run `up|down|force` only against a local throwaway database.
- Recovery from a dirty database: stop the crash-looping container, check whether the failed file's statement was applied, then `talk-control migrate force <N-1 or N>` (`talk-control migrate force -- -1` resets to unversioned). For the baseline either force target is safe (all three statements are `CREATE TABLE IF NOT EXISTS`). A termination signal does not interrupt a running migration.
- A new database is created with `CHARACTER SET utf8 COLLATE utf8_general_ci` (like the install scripts do); other defaults make the table definitions differ from the Alembic head.
- New database order: Alembic first, then this service, while the Alembic stream still creates the tables. The reverse is unsupported: Alembic then fails at revision `3ffdc562b708` (it drops every existing index, and the talk foreign keys need `idx_chat_id`).
- The sqlite test schema (`scripts/database_scripts/table_*.sql`) is a hand maintained copy; update it when the schema changes.
- Structure checks: `go test ./migrations/`. Git relative check: `scripts/check-service-migrations.sh bin-talk-manager` (repo root, after `git fetch origin main`). Dynamic check on scratch containers: `ALEMBIC_VENV=<venv> scripts/verify-service-migrations.sh bin-talk-manager` (repo root).

## talk-control CLI

```bash
./bin/talk-control chat list --customer-id <uuid>
./bin/talk-control message list --chat-id <uuid>
./bin/talk-control participant list --customer-id <uuid> --chat-id <uuid>
./bin/talk-control reaction add --message-id <uuid> --emoji <emoji> --owner-type <type> --owner-id <uuid>
./bin/talk-control migrate status
```

## Critical Implementation Notes

**Threading validation**: When `parent_id` is set, the parent must exist and belong to the same `chat_id`. Soft-deleted parents are intentionally allowed — preserves thread structure; UI renders them as placeholders.

**Atomic reactions**: Use `JSON_ARRAY_APPEND` / `JSON_REMOVE` in single `UPDATE` statements. Never read-modify-write reactions at application level.

**Participant re-join**: Uses `ON DUPLICATE KEY UPDATE` (MySQL) to UPSERT. Tests use SQLite `ON CONFLICT DO UPDATE SET` — both achieve the same result.

**Timestamps**: Always use `utilHandler.TimeGetCurTime()`. Never use `time.Now().UTC().Format()` — produces ISO 8601 format that MySQL rejects.

**Logging**: Every handler function must create `log := logrus.WithFields(logrus.Fields{"func": "...", "request": m})` at start. Error format: `"Could not <action>. err: %v"`.

## Testing Patterns

- Handler tests: gomock + table-driven with `reflect.DeepEqual` assertions.
- DB tests: SQLite in-memory via `scripts/database_scripts/`.
- Use `uuid.FromStringOrNil()` with fixed UUID strings — never random generation in tests.
- Error test functions named `Test_<Func>_error` separate from normal `Test_<Func>`.
