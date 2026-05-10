# bin-direct-manager

Direct hash routing for VoIPbin. Maps unique regeneratable hashes to customer resources (extensions, conferences, AI agents, AI teams, human agents), enabling direct SIP URI dialing without phone numbers. Hash lookups are on the critical SIP ingress path.

> Cross-cutting rules (verification workflow, branch/commit format, worktree usage, Alembic, RST sync) live in the root [CLAUDE.md](../CLAUDE.md).

## Documentation Index

| Doc | Contents |
|-----|----------|
| [docs/architecture.md](docs/architecture.md) | Component overview, layer responsibilities, request routing |
| [docs/domain.md](docs/domain.md) | Domain entity (Direct), resource types, business rules |
| [docs/dependencies.md](docs/dependencies.md) | Infrastructure, upstream/downstream services, events |
| [docs/operations.md](docs/operations.md) | Failure modes, debugging, configuration, Prometheus metrics |

## Quick Reference

**Build & run:**
```bash
go build -o bin/ ./cmd/...
./bin/direct-manager
```

**Verify (required before commit):**
```bash
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**CLI tool (bypasses RabbitMQ):**
```bash
./bin/direct-control direct list --customer-id <uuid>
./bin/direct-control direct get --id <uuid>
./bin/direct-control direct regenerate --id <uuid>
```

**Mock regeneration:**
```bash
go generate ./...
```

## Key Facts

- **Queue (listen):** `bin-manager.direct-manager.request`
- **Queue (subscribe):** `bin-manager.customer-manager.event` → cascade delete on `customer_deleted`
- **No events published**
- **Hash lookup** (`GET /v1/directs/by-hash/<hash>`) is the hot path for SIP ingress — hits Redis first
- **Hash regeneration**: `POST /v1/directs/{id}/regenerate` — atomically replaces hash, invalidates old cache entry
- **Resource types:** `extension`, `conference`, `ai`, `ai_team`, `agent`
- **Soft deletes:** `tm_delete = '9999-01-01 00:00:00.000000'` for active records
- **Metrics port:** `:2112/metrics`
