# bin-campaign-manager

## Overview

`bin-campaign-manager` orchestrates outbound calling campaigns in VoIPbin. It manages Campaign configuration, Outplan dial settings, and individual Campaigncall attempts, coordinating with call-manager to place calls and with flow-manager to execute on-connect actions. It is a Class A standard Go RPC manager.

> Cross-cutting rules (verification workflow, branch/commit format, worktree usage, Alembic, RST sync) live in the root [CLAUDE.md](../CLAUDE.md). This file documents only what is specific to `bin-campaign-manager`.

## Key Concepts

- **Campaign**: Outbound campaign entity; statuses: `stop`, `run`, `stopping`; references an outdial (target list), outplan, and optional queue
- **Campaigncall**: Single call attempt; up to 5 destination slots with independent retry counters; references a Call or Activeflow
- **Outplan**: Dialing configuration — `source` (caller ID), `dial_timeout`, `try_interval`, `max_try_count_0..4`; shared across campaigns
- **Service level**: Percentage throttle (0-100) based on available agents in the linked queue; 0 means no dialing
- **Actions**: Flow actions executed on call connect (play message, transfer to queue, etc.)
- **Next campaign chaining**: `next_campaign_id` enables sequential campaign execution after current campaign completes

## Common Commands

| Command | Purpose |
|---------|---------|
| `cd bin-campaign-manager && go build ./...` | Compile |
| `go test ./...` | Run all tests |
| `go test -v ./pkg/campaignhandler/...` | Test a specific package |
| `golangci-lint run -v --timeout 5m` | Lint |
| `go generate ./...` | Regenerate mocks |
| `./bin/campaign-control campaign get --id <uuid>` | Inspect a campaign (bypasses RabbitMQ) |

## Architecture
→ [docs/architecture.md](docs/architecture.md)

## Domain / Business Logic
→ [docs/domain.md](docs/domain.md)

## Dependencies
→ [docs/dependencies.md](docs/dependencies.md)

## Operations
→ [docs/operations.md](docs/operations.md)

## CRITICAL Rules

### Execute Loop Requires External Trigger

Campaign execution (`POST /v1/campaigns/{id}/execute`) must be called by an external scheduler to actually place calls. The service does not run an internal timer. If the scheduler stops, campaigns in `run` status will not dial.

### Service Level Requires Queue

The `service_level` throttle only applies when `queue_id` is set. Without a queue, all available slots are dialed immediately. Ensure campaigns intended to be throttled have a valid `queue_id`.

### Stopping State Requires Cleanup

The `stopping` transition waits for in-progress calls to complete. Do not force-stop a campaign in `stopping` state unless you are certain all associated calls have been cleaned up; orphaned calls may not be tracked properly.
