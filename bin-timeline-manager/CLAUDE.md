# bin-timeline-manager

Platform-wide audit log and event timeline service. Subscribes to 27 event queues, stores events in ClickHouse, and exposes a cursor-paginated read API over RabbitMQ RPC.

> Cross-cutting rules (verification, branch/commit format, worktrees, Alembic, RST sync) live in the root [CLAUDE.md](../CLAUDE.md).

## Docs

- [docs/architecture.md](docs/architecture.md) — component overview, layer responsibilities, request routing
- [docs/domain.md](docs/domain.md) — Event entity, business rules, wildcard matching, SIP analysis
- [docs/dependencies.md](docs/dependencies.md) — 27 subscribed queues, infrastructure, Homer API
- [docs/operations.md](docs/operations.md) — failure modes, debugging, configuration, metrics

## Common Commands

```bash
# Build
go build -o ./bin/timeline-manager ./cmd/timeline-manager/
go build -o ./bin/timeline-control ./cmd/timeline-control/

# Test
go test -coverprofile cp.out -v $(go list ./...)

# timeline-control operations
./bin/timeline-control migrate up
./bin/timeline-control migrate version
./bin/timeline-control health

# Verification (mandatory before commit)
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

## Critical Implementation Notes

**ClickHouse type constraint**: The ClickHouse Go driver cannot scan `String` columns into custom Go types. `models/event/` must use `string` for `Publisher` and `Type`. Convert to `ServiceName` and other rich types only in `eventhandler` at the API boundary. Violating this causes runtime panics:
```
converting String to *outline.ServiceName is unsupported
```

**Read-write separation**: The RPC listen path is read-only. The subscribe handler ingests events from 27 queues into ClickHouse. These are independent code paths — do not mix.

**Batch ingestion**: Subscribe handler batches ClickHouse writes. Monitor `subscribe_batch_insert_time` and `subscribe_batch_size` metrics for ingestion health.

**ClickHouse migrations**: Managed via golang-migrate. Run `timeline-control migrate up` after adding new migration files in `migrations/`. File format: `NNNNNN_description.up.sql` / `NNNNNN_description.down.sql`.

**Uses Cobra + Viper** (not pflag directly) — see `internal/config/` for configuration loading.

## Testing Patterns

- gomock (go.uber.org/mock) for handler interface mocks
- Table-driven tests with context passed to all handler methods
- Mocks co-located in same package as interface definition
