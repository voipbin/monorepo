# bin-message-manager

SMS messaging service — sends messages via Telnyx and MessageBird, tracks per-target delivery status, and processes provider webhooks for delivery updates.

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

## Key Commands

```bash
# Build
go build -o bin/message-manager ./cmd/message-manager

# Test
go test -coverprofile cp.out -v $(go list ./...)

# Verification (mandatory before commit)
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

# message-control — direct DB/cache access, bypasses RabbitMQ
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

## Architecture Summary

- **No SubscribeHandler** — webhook events arrive via `POST /v1/hooks` from bin-hook-manager
- **ListenHandler**: consumes `bin-manager.message-manager.request`
- **Provider dispatch**: MessageBird primary, Telnyx fallback; goroutine-based async send
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
