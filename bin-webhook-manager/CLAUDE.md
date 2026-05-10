# bin-webhook-manager

Webhook subscription management and outbound delivery service. Manages customer webhook configurations and dispatches HTTP notifications to customer-configured endpoints when events occur.

> Cross-cutting rules (verification workflow, branch/commit format, worktree usage, Alembic, RST sync) live in the root [CLAUDE.md](../CLAUDE.md).

## Docs index

- [docs/architecture.md](docs/architecture.md) — component layout, request routing table, event subscriptions
- [docs/domain.md](docs/domain.md) — Webhook/destination models, delivery modes, cache invalidation
- [docs/dependencies.md](docs/dependencies.md) — local monorepo deps, external services, queue names
- [docs/operations.md](docs/operations.md) — config flags, Prometheus metrics, CLI tool, common commands

## Key concepts

- **Webhook** — outbound HTTP notification to customer-configured URI when a VoIPbin event occurs
- **Two delivery modes**: `SendWebhookToCustomer` (uses customer's saved URI/method from `bin-customer-manager`) vs `SendWebhookToURI` (caller-specified override)
- **Cache invalidation** — `pkg/accounthandler` caches customer webhook configs in Redis; `subscribehandler` invalidates on `customer_updated`/`customer_deleted` events
- **Relationship to bin-hook-manager** — `bin-hook-manager` receives inbound external webhooks; `bin-webhook-manager` dispatches outbound webhooks to customers

## Common commands

```bash
# Full verification (mandatory before every commit)
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

# Build
go build -o ./bin/webhook-manager ./cmd/webhook-manager

# Test with coverage
go test -coverprofile cp.out -v $(go list ./...)

# Regenerate mocks
go generate ./...
```

## Testing pattern

gomock (go.uber.org/mock) + table-driven tests. Mock files co-located with handler packages (`mock_*.go`).
