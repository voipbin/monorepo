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

## talk-control CLI

```bash
./bin/talk-control chat list --customer-id <uuid>
./bin/talk-control message list --chat-id <uuid>
./bin/talk-control participant list --customer-id <uuid> --chat-id <uuid>
./bin/talk-control reaction add --message-id <uuid> --emoji <emoji> --owner-type <type> --owner-id <uuid>
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
