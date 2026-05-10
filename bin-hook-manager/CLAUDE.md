# bin-hook-manager

Public-facing HTTPS webhook gateway. Receives inbound HTTP webhook requests from external platforms (email, SMS, LINE, Paddle billing) and forwards them as RabbitMQ messages to internal services. No inbound RabbitMQ queue — this service exposes HTTP, not RPC.

> Cross-cutting rules (verification workflow, branch/commit format, worktree usage, Alembic, RST sync) live in the root [CLAUDE.md](../CLAUDE.md).

## Docs index

- [docs/architecture.md](docs/architecture.md) — component layout, execution model (HTTP→RabbitMQ bridge), routing table
- [docs/domain.md](docs/domain.md) — webhook types, Paddle verification, service handler interface
- [docs/dependencies.md](docs/dependencies.md) — local monorepo deps, external services, target queues
- [docs/operations.md](docs/operations.md) — config flags, CLI tool, common commands

## Key concepts

- **Thin proxy** — no business logic; transforms HTTP payloads into RabbitMQ messages
- **HTTP routing** — URL path determines destination: `/v1.0/emails` → email-manager, `/v1.0/messages` → message-manager, `/v1.0/conversation` → conversation-manager
- **No RabbitMQ inbound queue** — unlike all other services, this service only publishes; it does not consume from a request queue
- **SSL at application layer** — certificates passed as base64 env vars (`SSL_CERT_BASE64`, `SSL_PRIVKEY_BASE64`), written to `/tmp/` at startup

## CRITICAL: No listenhandler pattern

This service uses **Gin HTTP** (not RabbitMQ RPC). Do not add a `pkg/listenhandler`. HTTP endpoints are defined in `api/v1.0/*/`.

## Common commands

```bash
# Full verification (mandatory before every commit)
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

# Build
go build -o hook-manager ./cmd/hook-manager

# Test
go test -v ./...

# Regenerate mocks
go generate ./...
```

## Testing pattern

gomock mocks of `ServiceHandler` interface. Tests verify HTTP requests correctly invoke service handler methods. See `api/v1.0/emails/emails_test.go` for reference.
