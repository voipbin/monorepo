# bin-route-manager

Outbound call routing management for VoIPbin. Manages SIP providers (gateways) and routing rules. Key feature: `GET /v1/dialroutes` merges customer-specific routes with system-default fallbacks to determine the effective carrier list for a given destination.

> Cross-cutting rules (verification workflow, branch/commit format, worktree usage, Alembic, RST sync) live in the root [CLAUDE.md](../CLAUDE.md).

## Documentation Index

| Doc | Contents |
|-----|----------|
| [docs/architecture.md](docs/architecture.md) | Component overview, layer responsibilities, request routing |
| [docs/domain.md](docs/domain.md) | Domain entities (Provider, Route, Dialroute), dialroute merge algorithm, business rules |
| [docs/dependencies.md](docs/dependencies.md) | Infrastructure, upstream/downstream services |
| [docs/operations.md](docs/operations.md) | Failure modes, debugging, configuration, Prometheus metrics |

## Quick Reference

**Build & run:**
```bash
go build -o ./bin/ ./cmd/...
./bin/route-manager
```

**Verify (required before commit):**
```bash
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**CLI tool (bypasses RabbitMQ):**
```bash
./bin/route-control route list --customer_id <uuid>
./bin/route-control route dialroute-list --customer_id <uuid> --target 1
```

**Mock regeneration:**
```bash
go generate ./pkg/listenhandler/...
go generate ./pkg/routehandler/...
go generate ./pkg/providerhandler/...
go generate ./pkg/dbhandler/...
go generate ./pkg/cachehandler/...
```

## Key Facts

- **Queue (listen):** `bin-manager.route-manager.request`
- **No event subscriptions, no events published**
- **Dialroute merge**: customer routes + `CustomerIDBasicRoute` (`00000000-0000-0000-0000-000000000001`) defaults — customer wins on provider overlap
- **Target values:** country code string (e.g., `"1"` for US) or `"all"` for catch-all
- **Route priority:** lower integer = higher priority
- **Provider types:** `sip` (only type currently)
- **Soft deletes:** `tm_delete = '9999-01-01 00:00:00.000000'` for both providers and routes
- **Config required:** `external_sip_gateway_addresses` for provider setup
- **Metrics port:** `:2112/metrics`
