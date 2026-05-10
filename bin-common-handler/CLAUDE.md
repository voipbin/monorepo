# bin-common-handler

Shared Go library used by every `bin-*-manager` service in the monorepo. Provides inter-service RPC, event publishing, RabbitMQ transport, circuit breaking, DB helpers, and shared identity types. **Not a deployable service** — no `cmd/`, no listen queue, no subscriptions.

> Cross-cutting rules (verification workflow, branch/commit format, worktrees, Alembic, RST sync) live in the root [CLAUDE.md](../CLAUDE.md). This file covers only what is specific to `bin-common-handler`.

## CRITICAL: Admission rule

**A package may only live in `bin-common-handler` if it is used by 3 or more services.**

- Single-consumer or dual-consumer packages belong in the consuming service(s).
- If a package's usage later grows to 3+ services, it can be promoted here.
- Internal plumbing packages (`rabbitmqhandler`, wrapped by `sockhandler`) are exempt.

**Why:** Every change here triggers verification across all 37 consumer services. Keep the library lean.

## Package layout

| Package | Role |
|---------|------|
| `models/identity` | Common `Identity` struct (ID, CustomerID) embedded in all resources |
| `models/sock` | Wire types: `sock.Request`, `sock.Response`, `sock.Event` |
| `models/outline` | Canonical service names and queue name constants |
| `models/address` | Address-related models |
| `models/service` | Service definition types |
| `pkg/requesthandler` | Typed inter-service RPC client; all calls go through `sendRequest()` |
| `pkg/notifyhandler` | Event publishing (`PublishEvent`) and webhook delivery (`PublishWebhook`) |
| `pkg/sockhandler` | Abstract message-broker interface (backed by RabbitMQ) |
| `pkg/rabbitmqhandler` | Low-level RabbitMQ connection/channel management (internal) |
| `pkg/circuitbreakerhandler` | Per-target CB; 5 failures → 30s open; integrated into every RPC |
| `pkg/databasehandler` | SQL helpers: `PrepareFields`, `GetDBFields`, `ScanRow` |
| `pkg/utilhandler` | UUID, hashing, timestamp, email validation, URL utilities |

## Key rules

- **`requesthandler` is the only correct RPC path.** All service-to-service calls must go through it. Bypassing it loses circuit breaker protection.
- **Do not import `rabbitmqhandler` in consumers.** Use `sockhandler` instead.
- **Do not register metrics with the same names that `requesthandler` already registers.** `prometheus.MustRegister` panics on duplicates.
- **UUID fields need `,uuid` db tag; JSON fields need `,json` db tag** on models.
- **Use `models/outline` queue name constants** — no hardcoded queue strings in consumer services.

## When changing a public API

1. Build `bin-common-handler` itself: `go build ./...`
2. Build every consumer (run CI, or manually: `go build ./...` in each `bin-*-manager`)
3. Use AST-aware tooling for bulk renames (`gopls rename`), not plain `sed`
4. Run the full verification workflow in each affected service before committing

See [docs/workflows/common-gotchas.md](../docs/workflows/common-gotchas.md) for the shared-library update gotcha.

## Common commands

```bash
go test ./...
go generate ./...
golangci-lint run -v --timeout 5m

# Single package test
go test ./pkg/utilhandler/...
```

## Further reading

- [docs/architecture.md](docs/architecture.md) — full package descriptions and metrics
- [docs/usage.md](docs/usage.md) — import patterns and common usage examples
- [docs/patterns/circuit-breaker.md](../docs/patterns/circuit-breaker.md)
- [docs/patterns/per-pod-queues.md](../docs/patterns/per-pod-queues.md)
