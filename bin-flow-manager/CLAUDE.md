# bin-flow-manager

## Overview

`bin-flow-manager` is the call-flow orchestration service in VoIPbin. It stores Flow templates (ordered sequences of Actions) and executes them as Activeflow instances against live calls, dispatching each Action as an RPC call to the appropriate downstream service (call-manager, tts-manager, ai-manager, queue-manager, etc.). It is a Class A standard Go RPC manager.

> Cross-cutting rules (verification workflow, branch/commit format, worktree usage, Alembic, RST sync) live in the root [CLAUDE.md](../CLAUDE.md). This file documents only what is specific to `bin-flow-manager`.

## Key Concepts

- **Flow**: Stored template — an ordered list of Actions defining call behavior (answer → play → gather → condition → hangup)
- **Activeflow**: Live execution instance of a Flow; tracks current action, execution stack, and runtime variables
- **Action**: Atomic step with a `type` (e.g., `talk`, `queue`, `ai_talk`) and `param` map; dispatched to the appropriate manager
- **Stack**: Nested execution frames enabling sub-flow and branching patterns; pushed via `push_actions`, popped on completion
- **Variable**: Runtime key-value pairs scoped to an activeflow; set by gather/API actions, substituted into later action params via `{{var_name}}`
- **Direct hash**: Per-flow hash enabling webhook-triggered execution without API auth

## Common Commands

| Command | Purpose |
|---------|---------|
| `cd bin-flow-manager && go build ./...` | Compile |
| `go test ./...` | Run all tests |
| `go test -v ./pkg/activeflowhandler/...` | Test a specific package |
| `golangci-lint run -v --timeout 5m` | Lint |
| `go generate ./...` | Regenerate mocks |

## Architecture
→ [docs/architecture.md](docs/architecture.md)

## Domain / Business Logic
→ [docs/domain.md](docs/domain.md)

## Dependencies
→ [docs/dependencies.md](docs/dependencies.md)

## Operations
→ [docs/operations.md](docs/operations.md)

## Design Plans
→ [docs/plans/](docs/plans/)

## Database migrations

The schema of `flow_flows` and `flow_activeflows` is owned by this service and applied at startup (golang-migrate, embedded with `go:embed`, shared code in `bin-common-handler/pkg/migrationhandler`). It does not use Alembic. Tested engines: MariaDB and MySQL 8.0, so do not use engine specific syntax. The full rules (file naming, one DDL statement per file, `-- DESTRUCTIVE:` and `-- DOWN-LOSSY:` markers, immutable merged files, expand/contract, baseline conversion) are in [docs/workflows/service-owned-migrations.md](../docs/workflows/service-owned-migrations.md).

- Files: `migrations/NNNNNN_<table>_<verb>_<type>_<items>.up.sql` and `.down.sql`. `000001_baseline` is the schema at the time of the conversion (`CREATE TABLE IF NOT EXISTS`, no table options, down is the no-op `SELECT 1`). Version table: `flow_schema_migrations`.
- `cmd/flow-manager` runs `migrationhandler.Run` in `runDaemon` right after the database connection, before the cache and RabbitMQ. Failure stops the start (non-zero exit).
- `flow-control migrate status|up|down|force <version>` talks to the database only (no Redis or RabbitMQ). Run `up`, `down` and `force` only against a local throwaway database or as the recovery step in [docs/operations.md](docs/operations.md).
- `scripts/database_scripts/table_flows.sql` and `table_activeflow.sql` (the sqlite test schema, inside this service) are edited by hand when the schema changes.
- Verification: `go test ./migrations/` (structure) and, from the repo root, `ALEMBIC_VENV=<venv> scripts/verify-service-migrations.sh bin-flow-manager` (both engines).

## CRITICAL Rules

### Action Dispatch Safety

When adding a new action type, register it in `pkg/actionhandler` and add a corresponding case in the dispatch switch. Unregistered action types cause activeflows to error at runtime.

### Stack Frame Integrity

`push_actions` and `add_actions` mutate the activeflow's stack in MySQL. Always reload the activeflow from the DB after a stack mutation; do not rely on in-memory state for subsequent operations.

### Cache Strategy

Activeflow reads use Redis cache. All writes to MySQL must be mirrored to Redis immediately. Never skip the cache update.

### Subscribed Events

This service subscribes to `customer-manager` events to cascade customer deletion to flows and activeflows. Do not remove this subscription without ensuring cleanup is handled elsewhere.
