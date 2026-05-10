# bin-outdial-manager

Outbound dialing campaign management for VoIPbin. Stores outdial containers, individual call targets (up to 5 destinations per target with independent retry counters), and per-attempt call records. Primary consumer: `bin-campaign-manager`.

> Cross-cutting rules (verification workflow, branch/commit format, worktree usage, Alembic, RST sync) live in the root [CLAUDE.md](../CLAUDE.md).

## Documentation Index

| Doc | Contents |
|-----|----------|
| [docs/architecture.md](docs/architecture.md) | Component overview, layer responsibilities, request routing |
| [docs/domain.md](docs/domain.md) | Domain entities (Outdial, OutdialTarget, OutdialTargetCall), business rules |
| [docs/dependencies.md](docs/dependencies.md) | Infrastructure, upstream/downstream services, events |
| [docs/operations.md](docs/operations.md) | Failure modes, debugging, configuration, Prometheus metrics |

## Quick Reference

**Build & run:**
```bash
go build -o ./bin/ ./cmd/...
./bin/outdial-manager
```

**Verify (required before commit):**
```bash
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**CLI tool (bypasses RabbitMQ):**
```bash
./bin/outdial-control outdial list --customer_id <uuid>
./bin/outdial-control outdial get --id <outdial-uuid>
```

**Mock regeneration:**
```bash
go generate ./pkg/dbhandler/...
go generate ./pkg/cachehandler/...
go generate ./pkg/outdialhandler/...
go generate ./pkg/outdialtargethandler/...
```

## Key Facts

- **Queue (listen):** `bin-manager.outdial-manager.request`
- **No event subscriptions** (no subscribehandler)
- **Events published:** `outdial_created`, `outdial_updated`, `outdial_deleted`
- **Target statuses:** `idle` → `processing` → `done`
- **Multi-destination:** up to 5 destinations (`destination_0`–`destination_4`) with per-destination try counts
- **Available query:** filters targets by per-destination try-count thresholds for campaign retry logic
- **Soft deletes:** `tm_delete = '9999-01-01 00:00:00.000000'` for active records
- **Metrics port:** `:2112/metrics`
