# bin-transfer-manager

Handles call transfer operations (attended and blind) in the VoIPbin platform. Coordinates confbridge state and groupcall lifecycles with `bin-call-manager`, driven by call events.

> Cross-cutting rules (verification, branch/commit format, worktrees, Alembic, RST sync) live in the root [CLAUDE.md](../CLAUDE.md).

## Docs

- [docs/architecture.md](docs/architecture.md) — component overview, layer responsibilities, request routing
- [docs/domain.md](docs/domain.md) — Transfer entity, attended/blind state machines, event-driven transitions
- [docs/dependencies.md](docs/dependencies.md) — upstream services, subscribed queues, infrastructure
- [docs/operations.md](docs/operations.md) — failure modes, debugging, configuration, metrics

## Common Commands

```bash
# Build
go build -o ./bin/transfer-manager ./cmd/transfer-manager
go build -o ./bin/transfer-control ./cmd/transfer-control

# Test
go test ./...
go test -coverprofile cp.out -v $(go list ./...)

# transfer-control operations
./bin/transfer-control transfer service-start --transfer-type attended --transferer-call-id <uuid> --transferee-addresses '<json>'
./bin/transfer-control transfer get-by-call --call_id <uuid>
./bin/transfer-control transfer get-by-groupcall --groupcall_id <uuid>

# Verification (mandatory before commit)
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

## Critical Implementation Notes

**Attended transfer rollback**: `attendedUnblock` removes MOH and mute from original bridge participants. If this is not called (e.g., due to a bug), parties remain stuck on hold. Always ensure rollback is invoked on `groupcall_hangup` or transfer cancel.

**Blind transfer confbridge flag**: `blindBlock` sets `FlagNoAutoLeave` so the confbridge survives the transferer hangup. `blindUnblock` MUST clear this flag on failure — otherwise the confbridge never auto-destroys.

**State transitions are event-driven**: Transfer state advances only when `subscribehandler` receives call-manager events (`groupcall_progressing`, `groupcall_hangup`, `call_hangup`). Do not poll or use timers.

**Soft deletes**: Active records use `tm_delete = "9999-01-01 00:00:00.000000"`.

## Testing Patterns

- gomock (go.uber.org/mock) for all handler interfaces
- Table-driven tests covering attended/blind workflows, rollback paths, and error handling
- Mock files: `mock_*.go` co-located with handler interfaces
