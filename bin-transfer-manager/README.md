# bin-transfer-manager

Call transfer orchestration service for VoIPbin. Handles attended and blind transfer operations, coordinating confbridge state and groupcall lifecycles with `bin-call-manager`.

## Key Concepts

- **Attended transfer**: Transferer consults transferee before completing handoff; supports rollback if consultation fails
- **Blind transfer**: Immediate handoff without consultation; confbridge uses `NoAutoLeave` flag to survive the transferer hangup
- **Transfer state**: Entirely event-driven — advances only when `subscribehandler` receives call-manager events (`groupcall_progressing`, `groupcall_hangup`, `call_hangup`)

## Public RPC Entrypoints

| Pattern | Operations |
|---------|-----------|
| `POST /v1/transfers` | Initiate transfer |
| `GET /v1/transfers` | List transfers |
| `GET /v1/transfers/<id>` | Get transfer |
| `DELETE /v1/transfers/<id>` | Cancel transfer |
| `POST /v1/transfers/<id>/complete` | Complete attended transfer |

## Dependencies

- **MySQL** — transfer records (soft-delete via `tm_delete`)
- **Redis** — transfer cache
- **RabbitMQ** — listen queue `bin-manager.transfer-manager.request`; subscribes to `bin-manager.call-manager.event`
- **bin-call-manager** — confbridge mute/unmute/flag manipulation, groupcall management

## Local Development

```bash
# Build
cd bin-transfer-manager
go build -o ./bin/ ./cmd/...

# Run all tests
go test ./...

# Verify before commit (mandatory)
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

# CLI tool (bypasses RabbitMQ)
./bin/transfer-control transfer get-by-call --call_id <uuid>
./bin/transfer-control transfer get-by-groupcall --groupcall_id <uuid>
```

## Further Reading

- [docs/architecture.md](docs/architecture.md)
- [docs/domain.md](docs/domain.md)
- [docs/dependencies.md](docs/dependencies.md)
- [docs/operations.md](docs/operations.md)
