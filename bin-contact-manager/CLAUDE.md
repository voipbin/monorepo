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

**Migration status (database only, no Redis or RabbitMQ):**
```bash
./bin/contact-control migrate status
```

**Mock regeneration:**
```bash
go generate ./pkg/listenhandler/...
go generate ./pkg/subscribehandler/...
go generate ./pkg/contacthandler/...
go generate ./pkg/dbhandler/...
go generate ./pkg/cachehandler/...
```

## Database migrations

The schema of the nine tables `contact_activities`, `contact_address_ownership_periods`, `contact_addresses`, `contact_case_notes`, `contact_cases`, `contact_contacts`, `contact_list_members`, `contact_lists` and `contact_tag_assignments` is owned by this service and applied at startup (golang-migrate, embedded with `go:embed`). It does not use Alembic. Tested engines: MariaDB and MySQL 8.0, so do not use engine specific syntax.

- Files: `migrations/NNNNNN_<description>.up.sql` and `.down.sql` (six digits, from 000001 without gaps). `000001_baseline` is the schema at the time of the conversion (`CREATE TABLE IF NOT EXISTS`, the table options Alembic declared, down is the no-op `SELECT 1`). Version table: `contact_schema_migrations`.
- `cmd/contact-manager` runs `migrationhandler.Run` right after the database connection and before the cache and RabbitMQ. Failure stops the start (non-zero exit).
- `contact-control migrate status|up|down|force <version>` talks to the database only. Run `up`, `down` and `force` only against a local throwaway database. `case-control` has no database access and no `migrate` command.
- The migration code is the shared `bin-common-handler/pkg/migrationhandler`. Rules for new files (one DDL statement per file, `-- DESTRUCTIVE:` and `-- DOWN-LOSSY:` markers, immutable merged files, expand/contract), the checks and the recovery steps are in [docs/workflows/service-owned-migrations.md](../docs/workflows/service-owned-migrations.md).
- A new database is created with `CHARACTER SET utf8 COLLATE utf8_general_ci` (like the install scripts do); other defaults make the table definitions differ from the Alembic head.
- Limit of the baseline: `CREATE TABLE IF NOT EXISTS` does not compare an existing table with the baseline, so a database whose tables differ from the Alembic head starts as version 1 without any warning. Compare the normalized dump of an existing database with the head before the first deployment (`REFERENCE_DUMP` in `scripts/verify-service-migrations.sh`).
- New database order: Alembic first, then this service. The reverse fails in the Alembic stream (`Table 'contact_contacts' already exists`) and is unsupported while the Alembic stream still creates the contact tables.
- Update `scripts/database_scripts_test/contacts.sql` (the sqlite test schema) by hand when the schema changes.
- Structure checks: `go test ./migrations/` (file pairs, numbering, statement rules and the table options per table; it does not compare columns or indexes, the dump comparison of the verify script does). Git relative check: `scripts/check-service-migrations.sh bin-contact-manager` after `git fetch origin main`. Dynamic check on scratch containers: `ALEMBIC_VENV=<venv> CONTROL_BIN=contact-control scripts/verify-service-migrations.sh bin-contact-manager` (repo root).

## Key Facts

- **Queue (listen):** `bin-manager.contact-manager.request`
- **Queue (subscribe):** `bin-manager.customer-manager.event` → cascade delete on `customer_deleted`
- **Events published:** `contact_created`, `contact_updated`, `contact_deleted`
- **Databases:** MySQL (`contact_contacts`, `contact_addresses`, `contact_tag_assignments`) + Redis (contact-body cache, DB index 1). `contact_addresses` is the unified identifier store (phone + email) that replaced the legacy `contact_phone_numbers` / `contact_emails` tables (VOIP-1207); rows carry `type` ('tel'|'email') and a normalized `target`.
- **Soft deletes:** active records have `tm_delete IS NULL`; deletion records the actual timestamp in `tm_delete` (no `9999-01-01` sentinel). `ContactList` and the public get/lookup paths exclude soft-deleted records by default; the by-id dbhandler primitive stays unfiltered so the delete event payload can re-read the tombstone. `contact_addresses` is hard-delete (no `tm_delete`), mirroring `agent_addresses`.
- **Caller-ID lookup:** `ContactLookupByPhone` / `ByEmail` query `contact_addresses` directly (`WHERE customer_id=? AND type=? AND target=?`) then load the contact; there is no per-identifier Redis cache (only the contact-body cache keyed by contact id).
- **Metrics port:** `:2112/metrics`
