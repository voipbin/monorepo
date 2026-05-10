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
- **Wait timeout**: Maximum time a caller waits before being abandoned; triggered by an external scheduler call
- **Service timeout**: Maximum agent service duration; triggers forced disconnect via `timeout_service`

## Common Commands

| Command | Purpose |
|---------|---------|
| `cd bin-queue-manager && go build ./...` | Compile |
| `go test ./...` | Run all tests |
| `go test -v ./pkg/queuecallhandler/...` | Test a specific package |
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

## CRITICAL Rules

### Tag Matching is Exact Overlap

Queue routing only considers agents whose `tag_ids` intersect with the queue's `tag_ids`. An agent with no tags is never routable. Verify tag IDs are consistent across queue and agent configurations when debugging routing failures.

### Conference is the Bridge

Agent-caller connection uses conference-manager. If conference-manager is unavailable or at capacity, all queue routing will fail. Monitor conference-manager health alongside queue-manager.

### Timeout Requires External Trigger

Wait and service timeouts are NOT enforced internally by a timer in this service. The scheduler (flow actions or an external cron) must call `timeout_wait` / `timeout_service`. If the scheduler stops, callers may wait indefinitely.
