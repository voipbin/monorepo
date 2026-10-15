# bin-agent-manager

## Overview

`bin-agent-manager` manages call-center agents in VoIPbin: their identity, authentication, SIP contact addresses, permissions, real-time status (available/away/busy/offline), and tag-based routing membership. It is a Class A standard Go RPC manager.

> Cross-cutting rules (verification workflow, branch/commit format, worktree usage, Alembic, RST sync) live in the root [CLAUDE.md](../CLAUDE.md). This file documents only what is specific to `bin-agent-manager`.

## Key Concepts

- **Agent**: A call-center operator with status, SIP addresses, permission flags, ring method, and tag IDs
- **Status**: Real-time availability — `available`, `away`, `busy`, `offline`; driven by call-manager events
- **Ring method**: `ringall` (all addresses simultaneously) or `linear` (addresses tried in sequence)
- **Permission**: Bitfield with project-level and customer-level flags (agent/admin/manager)
- **Tag IDs**: Used by queue-manager to filter eligible agents for routing; changing tags immediately affects queue membership
- **Password reset**: Requires `password_reset_base_url` config; uses email-manager for delivery

## Common Commands

| Command | Purpose |
|---------|---------|
| `cd bin-agent-manager && go build ./...` | Compile |
| `go test ./...` | Run all tests |
| `go test -v ./pkg/agenthandler/...` | Test a specific package |
| `golangci-lint run -v --timeout 5m` | Lint |
| `go generate ./...` | Regenerate mocks |

## Database Migrations

The schema of `agent_agents` and `agent_addresses` is owned by this service and applied at startup (golang-migrate, embedded with `go:embed`, shared code in `bin-common-handler/pkg/migrationhandler`). It does not use Alembic. Tested engines: MariaDB and MySQL 8.0, so do not use engine specific syntax. The full rules (file naming, one DDL statement per file, `-- DESTRUCTIVE:` and `-- DOWN-LOSSY:` markers, immutable merged files, expand/contract, baseline conversion) are in [docs/workflows/service-owned-migrations.md](../docs/workflows/service-owned-migrations.md).

- Files: `migrations/NNNNNN_<table>_<verb>_<type>_<items>.up.sql` and `.down.sql`. `000001_baseline` is the schema at the time of the conversion (`CREATE TABLE IF NOT EXISTS`, no table options, down is the no-op `SELECT 1`). Version table: `agent_schema_migrations`.
- `cmd/agent-manager` runs `migrationhandler.Run` right after the database connection, before the cache and RabbitMQ. Failure stops the start (non-zero exit).
- `agent-control migrate status|up|down|force <version>` talks to the database only (no Redis or RabbitMQ). Run `up`, `down` and `force` only against a local throwaway database or as the recovery step in [docs/operations.md](docs/operations.md).
- A new database is created with `CHARACTER SET utf8 COLLATE utf8_general_ci` (like the install scripts do); other defaults make the table definitions differ from the Alembic head.
- New database order: Alembic first, then this service. The reverse fails in the Alembic stream (`Table 'agent_agents' already exists`) and is unsupported while the Alembic stream still creates the agent tables.
- Update `scripts/database_scripts_test/*.sql` (the sqlite test schema) by hand when the schema changes.
- Verification: `go test ./migrations/` (structure) and, from the repo root, `ALEMBIC_VENV=<venv> scripts/verify-service-migrations.sh bin-agent-manager` (both engines).

## Architecture
→ [docs/architecture.md](docs/architecture.md)

## Domain / Business Logic
→ [docs/domain.md](docs/domain.md)

## Dependencies
→ [docs/dependencies.md](docs/dependencies.md)

## Operations
→ [docs/operations.md](docs/operations.md)

## CRITICAL Rules

### Status is Event-Driven

Agent status changes to/from `busy` are driven by call-manager events received in `subscribehandler` (`groupcall_progressing` sets `busy` on answer; call-end returns to `available`). During dial before answer, the agent is excluded from re-selection by the method B reservation fields (`reserve_reference_id`), not by a status change. Do not rely on polling or timeouts for status recovery. If subscribehandler is down, agents can remain stuck in incorrect states.

### Password Reset Base URL

The `password_reset_base_url` config flag MUST be set in production for password reset emails to contain a valid link. Deploying without this flag causes silent failures in the forgot-password flow.

### Address Lookup Exactness

The `get_by_customer_id_address` endpoint does exact-match lookup on the SIP address string. Ensure SIP URI normalization is consistent when storing and querying addresses (same port, same parameters).
