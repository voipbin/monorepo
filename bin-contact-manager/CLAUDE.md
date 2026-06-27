# bin-contact-manager

CRM-style contact record management for VoIPbin. Handles contacts, phone numbers, emails, and tag assignments. Serves O(1) phone/email lookup for inbound call caller-ID enrichment.

> Cross-cutting rules (verification workflow, branch/commit format, worktree usage, Alembic, RST sync) live in the root [CLAUDE.md](../CLAUDE.md).

## Documentation Index

| Doc | Contents |
|-----|----------|
| [docs/architecture.md](docs/architecture.md) | Component overview, layer responsibilities, request routing |
| [docs/domain.md](docs/domain.md) | Domain entities (Contact, PhoneNumber, Email, TagAssignment), business rules |
| [docs/dependencies.md](docs/dependencies.md) | Infrastructure, upstream/downstream services, events |
| [docs/operations.md](docs/operations.md) | Failure modes, debugging, configuration, Prometheus metrics |

## Quick Reference

**Build & run:**
```bash
go build -o ./bin/ ./cmd/...
./bin/contact-manager
```

**Verify (required before commit):**
```bash
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**CLI tool (bypasses RabbitMQ):**
```bash
./bin/contact-control contact list --customer-id <uuid>
./bin/contact-control contact lookup --customer-id <uuid> --phone-e164 +15551234567
```

**Mock regeneration:**
```bash
go generate ./pkg/listenhandler/...
go generate ./pkg/subscribehandler/...
go generate ./pkg/contacthandler/...
go generate ./pkg/dbhandler/...
go generate ./pkg/cachehandler/...
```

## Key Facts

- **Queue (listen):** `bin-manager.contact-manager.request`
- **Queue (subscribe):** `bin-manager.customer-manager.event` → cascade delete on `customer_deleted`
- **Events published:** `contact_created`, `contact_updated`, `contact_deleted`
- **Databases:** MySQL (`contact_contacts`, `contact_addresses`, `contact_tag_assignments`) + Redis (contact-body cache, DB index 1). `contact_addresses` is the unified identifier store (phone + email) that replaced the legacy `contact_phone_numbers` / `contact_emails` tables (VOIP-1207); rows carry `type` ('tel'|'email') and a normalized `target`.
- **Soft deletes:** active records have `tm_delete IS NULL`; deletion records the actual timestamp in `tm_delete` (no `9999-01-01` sentinel). `ContactList` and the public get/lookup paths exclude soft-deleted records by default; the by-id dbhandler primitive stays unfiltered so the delete event payload can re-read the tombstone. `contact_addresses` is hard-delete (no `tm_delete`), mirroring `agent_addresses`.
- **Caller-ID lookup:** `ContactLookupByPhone` / `ByEmail` query `contact_addresses` directly (`WHERE customer_id=? AND type=? AND target=?`) then load the contact; there is no per-identifier Redis cache (only the contact-body cache keyed by contact id).
- **Metrics port:** `:2112/metrics`
