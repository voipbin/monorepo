# bin-registrar-manager

SIP registration management for VoIPbin. Provisions and manages Asterisk PJSIP extensions (user endpoints) and trunks (carrier connections) across two databases. Serves contact registration lookups on the SIP ingress path.

> Cross-cutting rules (verification workflow, branch/commit format, worktree usage, Alembic, RST sync) live in the root [CLAUDE.md](../CLAUDE.md).

## Documentation Index

| Doc | Contents |
|-----|----------|
| [docs/architecture.md](docs/architecture.md) | Component overview, dual-DB architecture, layer responsibilities, request routing |
| [docs/domain.md](docs/domain.md) | Domain entities (Extension, Trunk, Contact), Asterisk table mapping, business rules |
| [docs/dependencies.md](docs/dependencies.md) | Infrastructure (two MySQL DBs), upstream/downstream services, events |
| [docs/operations.md](docs/operations.md) | Failure modes, debugging Asterisk orphans, configuration, Prometheus metrics |

## Quick Reference

**Build & run:**
```bash
go build -o bin/registrar-manager ./cmd/registrar-manager
./bin/registrar-manager \
  --database_dsn_asterisk "user:pass@tcp(host:3306)/asterisk" \
  --database_dsn_bin "user:pass@tcp(host:3306)/bin_manager" \
  --rabbitmq_address "amqp://guest:guest@localhost:5672" \
  --redis_address "localhost:6379" \
  --domain_name_extension "ext.voipbin.net" \
  --domain_name_trunk "trunk.voipbin.net"
```

**Verify (required before commit):**
```bash
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**CLI tool (bypasses RabbitMQ):**
```bash
./bin/registrar-control extension list --customer_id <uuid>
./bin/registrar-control trunk list --customer_id <uuid>
```

**Mock regeneration:**
```bash
go generate ./...
```

## Key Facts

- **Queue (listen):** `bin-manager.registrar-manager.request`
- **Queue (subscribe):** `bin-manager.customer-manager.event` → cascade delete on `customer_deleted`
- **No events published**
- **Two databases**: `DATABASE_DSN_BIN` (extensions/trunks) + `DATABASE_DSN_ASTERISK` (`ps_*` tables)
- **Asterisk tables managed:** `ps_endpoints`, `ps_aors`, `ps_auths` (create/delete); `ps_contacts` (read-only)
- **Critical**: Extension create/delete must touch all three Asterisk tables atomically
- **Domain config required**: `domain_name_extension` and `domain_name_trunk` must be set at startup
- **Soft deletes:** bin-manager tables use `tm_delete`; Asterisk tables are hard-deleted
- **Metrics port:** `:2112/metrics`
