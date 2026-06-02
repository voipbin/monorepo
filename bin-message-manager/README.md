# bin-message-manager

SMS messaging service for VoIPbin. Sends messages via Telnyx and MessageBird, tracks per-target delivery status, and processes provider webhooks for delivery updates.

## Key Concepts

- **Message**: Top-level outbound SMS record; references one or more targets
- **Target**: Per-recipient delivery record with status tracking (`initiating`, `queued`, `sent`, `delivered`, `failed`)
- **Provider dispatch**: MessageBird primary, Telnyx fallback; send is asynchronous (goroutine) — caller receives success after record creation
- **Webhook**: Delivery status updates arrive from providers via `bin-hook-manager` → `POST /v1/hooks`

## Public RPC Entrypoints

| Pattern | Operations |
|---------|-----------|
| `POST /v1/messages` | Send message (requires verified account) |
| `GET /v1/messages` | List messages |
| `GET /v1/messages/<id>` | Get message |
| `DELETE /v1/messages/<id>` | Delete message |
| `GET /v1/messagetargets` | List targets |
| `GET /v1/messagetargets/<id>` | Get target |
| `POST /v1/hooks` | Provider webhook handler |

## Dependencies

- **MySQL** — message and target records
- **Redis** — message and target cache
- **RabbitMQ** — listen queue `bin-manager.message-manager.request`
- **bin-billing-manager** — balance validation before send
- **bin-hook-manager** — routes inbound provider webhooks to this service

## Local Development

```bash
# Build
cd bin-message-manager
go build -o ./bin/ ./cmd/...

# Run all tests
go test ./...

# Verify before commit (mandatory)
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

# CLI tool (bypasses RabbitMQ)
./bin/message-control message list --customer_id <uuid>
./bin/message-control message get --id <uuid>
```

## Further Reading

- [docs/architecture.md](docs/architecture.md)
- [docs/domain.md](docs/domain.md)
- [docs/dependencies.md](docs/dependencies.md)
- [docs/operations.md](docs/operations.md)
