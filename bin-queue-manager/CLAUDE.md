# bin-queue-manager

## Overview

`bin-queue-manager` manages inbound call queues in VoIPbin. It holds callers in a waiting state, routes them to available agents using tag-based matching, and coordinates the conference bridge used to connect agent and caller. It is a Class A standard Go RPC manager.

> Cross-cutting rules (verification workflow, branch/commit format, worktree usage, Alembic, RST sync) live in the root [CLAUDE.md](../CLAUDE.md). This file documents only what is specific to `bin-queue-manager`.

## Key Concepts

- **Queue**: Configuration entity — routing method, `tag_ids` (agent filter), `wait_timeout`, `service_timeout`
- **Queuecall**: A single call waiting in a queue; status: `initiating` → `waiting` → `connecting` → `service` → `done`/`abandoned`
- **Routing method**: `random` — picks a random available agent whose tags overlap with the queue's required tags
- **Tag matching**: Queue's `tag_ids` must overlap with agent's `tag_ids` for the agent to be eligible
- **Conference connection**: When an agent is matched, a conference (in conference-manager) bridges the caller and agent
- **Wait timeout**: Maximum time a caller waits before being abandoned; self-scheduled (no external scheduler — see "Timeouts Are Self-Scheduled, Not Externally Triggered" below)
- **Service timeout**: Maximum agent service duration; self-scheduled, triggers forced disconnect via `timeout_service`

## Common Commands

| Command | Purpose |
|---------|---------|
| `cd bin-queue-manager && go build ./...` | Compile |
| `go test ./...` | Run all tests |
| `go test -v ./pkg/queuecallhandler/...` | Test a specific package |
| `golangci-lint run -v --timeout 5m` | Lint |
| `go generate ./...` | Regenerate mocks |
| `./bin/queue-control migrate status` | Migration status (database only, no Redis or RabbitMQ) |

## Database migrations

The schema of `queue_queues`, `queue_queuecalls` and `queue_queuecallreferences` is owned by this service and applied at startup (golang-migrate, embedded with `go:embed`). It does not use Alembic. Tested engines: MariaDB and MySQL 8.0, so do not use engine specific syntax.

- Files: `migrations/NNNNNN_<description>.up.sql` and `.down.sql` (six digits, from 000001 without gaps). `000001_baseline` is the schema at the time of the conversion (`CREATE TABLE IF NOT EXISTS`, no table options, down is the no-op `SELECT 1`). Version table: `queue_schema_migrations`.
- `cmd/queue-manager` runs `migrationhandler.Run` in `runDaemon`, right after the database connection and before the cache and RabbitMQ. Failure stops the start (non-zero exit).
- `queue-control migrate status|up|down|force <version>` talks to the database only. Run `up`, `down` and `force` only against a local throwaway database.
- The migration code is the shared `bin-common-handler/pkg/migrationhandler`. Rules for new files (one DDL statement per file, `-- DESTRUCTIVE:` and `-- DOWN-LOSSY:` markers, immutable merged files, expand/contract), the checks and the recovery steps are in [docs/workflows/service-owned-migrations.md](../docs/workflows/service-owned-migrations.md).
- A new database is created with `CHARACTER SET utf8 COLLATE utf8_general_ci` (like the install scripts do); other defaults make the table definitions differ from the Alembic head.
- New database order: Alembic first, then this service. The reverse fails in the Alembic stream (`Table 'queue_queues' already exists`) and is unsupported while the Alembic stream still creates the queue tables.
- Update `scripts/database_scripts_test/*.sql` in this service directory (the sqlite test schema) by hand when the schema changes.
- Structure checks: `go test ./migrations/`. Git relative check: `scripts/check-service-migrations.sh bin-queue-manager` after `git fetch origin main`. Dynamic check on scratch containers: `ALEMBIC_VENV=<venv> scripts/verify-service-migrations.sh bin-queue-manager` (repo root).

## Architecture
→ [docs/architecture.md](docs/architecture.md)

## Domain / Business Logic
→ [docs/domain.md](docs/domain.md)

## Dependencies
→ [docs/dependencies.md](docs/dependencies.md)

## Operations
→ [docs/operations.md](docs/operations.md)

## CRITICAL Rules

### Tag Matching is Exact Overlap

Queue routing only considers agents whose `tag_ids` intersect with the queue's `tag_ids`. An agent with no tags is never routable. Verify tag IDs are consistent across queue and agent configurations when debugging routing failures.

### Conference is the Bridge

Agent-caller connection uses conference-manager. If conference-manager is unavailable or at capacity, all queue routing will fail. Monitor conference-manager health alongside queue-manager.

### Timeouts Are Self-Scheduled, Not Externally Triggered

Wait and service timeouts are not enforced by a `time.Ticker`/cron goroutine, but they are not dormant either (verified VOIP-1282: no external scheduler or cron exists anywhere in the deployment surface, and none is needed). Instead, `bin-queue-manager` schedules its own timeout via a delayed RabbitMQ RPC to itself the instant the relevant state begins: `Create` (`pkg/queuecallhandler/db.go`) fires a delayed `QueueV1QueuecallTimeoutWait` the moment a queuecall is created (delay = `queue.WaitTimeout`), and `UpdateStatusService` fires a delayed `QueueV1QueuecallTimeoutService` the moment a call moves to `service` (delay = `queue.ServiceTimeout`). Both ride the same shared `x-delayed-message` RabbitMQ exchange every service uses for self-timers. This is fully automatic and built into the `queue_join` flow action itself (`bin-flow-manager`'s `actionHandleQueueJoin` → `QueueV1ServiceTypeQueuecallStart` → `ServiceStart` → `Create`) — a flow author does not need to add a separate wait/timer action. `TimeoutWait`/`TimeoutService` re-check the queuecall's status before acting, so a call that already left waiting/service before the delay fires is a no-op.
