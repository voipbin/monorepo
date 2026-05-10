# bin-conversation-manager

Multi-channel conversation management service. Handles SMS/MMS and LINE messaging threads, inbound/outbound message delivery, and platform account credentials.

> Cross-cutting rules (verification workflow, branch/commit format, worktree usage, Alembic, RST sync) live in the root [CLAUDE.md](../CLAUDE.md).

## Docs index

- [docs/architecture.md](docs/architecture.md) — component layout, request routing table, event subscriptions
- [docs/domain.md](docs/domain.md) — Account/Conversation/Message models, message flows, flow variables
- [docs/dependencies.md](docs/dependencies.md) — local monorepo deps, external services, queue names
- [docs/operations.md](docs/operations.md) — config flags, Prometheus metrics, CLI tool, common commands

## Key concepts

- **Account** — platform credentials (LINE channel secret/token, SMS provider); type: `sms` | `line`
- **Conversation** — thread between two parties identified by `(account_id, dialog_id)`; type: `message` | `line`
- **Message** — individual message; direction `inbound`/`outbound`, status `progressing`/`done`/`failed`
- **DialogID** — external platform conversation identifier (LINE chatroom ID, SMS thread)
- Inbound LINE messages arrive via `POST /v1/hooks`; inbound SMS/MMS arrive via `message-manager` event subscription

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
