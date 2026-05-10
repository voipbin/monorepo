# bin-customer-manager

Foundational identity service managing tenant organizations (customers) and their API credentials (access keys). Almost all other services depend on customer context — this is the root of the VoIPbin tenant hierarchy.

> Cross-cutting rules (verification workflow, branch/commit format, worktrees, Alembic, RST sync) live in the root [CLAUDE.md](../CLAUDE.md).

## Docs

- [docs/architecture.md](docs/architecture.md) — components, layers, request routing
- [docs/domain.md](docs/domain.md) — Customer, AccessKey entities; lifecycle rules; cascade behavior
- [docs/dependencies.md](docs/dependencies.md) — events published, outbound RPCs, monorepo deps
- [docs/operations.md](docs/operations.md) — failure modes, debugging, config reference, metrics

## CRITICAL: No SubscribeHandler

This service does not consume events from other services. All state is driven by inbound RPC only. If you add event-driven behavior, it must go through the request queue.

## CRITICAL: Foundational Dependency

`bin-customer-manager` events (`customer_deleted`) trigger cascading cleanup in `bin-number-manager` and `bin-billing-manager`. Ensure event publishing is functional before deploying changes.

## Key Commands

```bash
# Build
go build -o bin/customer-manager ./cmd/customer-manager
go build -o bin/customer-control ./cmd/customer-control

# Test
go test -coverprofile cp.out -v $(go list ./...)

# Verification (mandatory before commit)
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

# customer-control — direct DB/cache access, bypasses RabbitMQ
./bin/customer-control customer get --id <uuid>
./bin/customer-control customer list [--limit 100] [--token]
./bin/customer-control accesskey list --customer-id <uuid>
./bin/customer-control customer create --email test@example.com --name "Test"

# Run locally
DATABASE_DSN="user:pass@tcp(127.0.0.1:3306)/voipbin" \
RABBITMQ_ADDRESS="amqp://guest:guest@localhost:5672" \
REDIS_ADDRESS="127.0.0.1:6379" \
./bin/customer-manager
```

## Architecture Summary

- **No SubscribeHandler** — purely RPC-driven
- **ListenHandler**: consumes `bin-manager.customer-manager.request`
- **Cache-first reads**: Redis → MySQL on miss; invalidated on all mutations
- **Pagination**: cursor-based (`tm_create` timestamp as page token)
- **Outbound RPC**: agent-manager (username check), billing-manager (account link validation)

## Testing

Uses `go.uber.org/mock` (gomock). Table-driven tests. Parameterized raw SQL (no Squirrel).

```bash
go test -v ./pkg/customerhandler -run Test_Delete
```

## Database Tables

- `customer_customers` — customers (soft-delete with `tm_delete`)
- `customer_accesskeys` — access keys (soft-delete with `tm_delete`)
