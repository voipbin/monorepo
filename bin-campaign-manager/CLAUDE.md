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

## Database migrations

The schema of `campaign_campaigns`, `campaign_campaigncalls` and `campaign_outplans` is owned by this service and applied at startup (golang-migrate, embedded with `go:embed`). It does not use Alembic. Tested engines: MariaDB and MySQL 8.0, so do not use engine specific syntax.

- Files: `migrations/NNNNNN_<table>_<verb>_<type>_<items>.up.sql` and `.down.sql` (six digits, from 000001 without gaps). `000001_baseline` is the schema at the time of the conversion. Version table: `campaign_schema_migrations`.
- `cmd/campaign-manager` runs `migrationhandler.Run` in `createDBHandler`, right after the database connection and before the cache connection, RabbitMQ and listening. A failure makes the process exit with a non-zero code.
- The rules for new migrations (one DDL statement per file, `-- DESTRUCTIVE:` and `-- DOWN-LOSSY:` markers, immutable merged files, expand/contract), the recovery from a dirty database and the verification scripts are the same for every service that owns its schema: see [docs/workflows/service-owned-migrations.md](../docs/workflows/service-owned-migrations.md) and the migration section of [bin-tag-manager/CLAUDE.md](../bin-tag-manager/CLAUDE.md).
- Structure checks: `go test ./migrations/`. Git relative check: `scripts/check-service-migrations.sh bin-campaign-manager` after `git fetch origin main`. Dynamic check on scratch containers: `ALEMBIC_VENV=<venv> scripts/verify-service-migrations.sh bin-campaign-manager` (repo root).
- `campaign-control migrate status|up|down|force <version>` talks to the database only (no Redis or RabbitMQ). The DSN comes from the `DATABASE_DSN` environment variable (the `--database_dsn` flag belongs to the root command only and is not accepted after `migrate`). Run `up|down|force` only against a local throwaway database; `down` is never run automatically and the baseline down is a no-op.
- Update `scripts/database_scripts/*.sql` (the sqlite test schema) by hand when the schema changes.
- Limit of the baseline: `CREATE TABLE IF NOT EXISTS` does not compare an existing table with the baseline, so a database whose tables differ from the Alembic head starts as version 1 without any warning. Compare the normalized dump of an existing database with the head before the first deployment (`REFERENCE_DUMP` in `scripts/verify-service-migrations.sh`).
- New database order: Alembic first, then this service. The reverse fails in the Alembic stream (`Table 'campaign_campaigns' already exists`) and is unsupported while the Alembic stream still creates these tables.

## CRITICAL Rules

### Execute Loop Is Self-Scheduling, Not Externally Triggered

Campaign execution (`POST /v1/campaigns/{id}/execute`) is not called by any external scheduler or cron (verified VOIP-1282: no external scheduler or cron exists anywhere in the deployment surface, and none is needed). Instead it is a self-perpetuating chain of delayed RabbitMQ RPC calls: setting a campaign's status to `run` (`campaignRun` in `pkg/campaignhandler/status_run.go`) fires the first delayed `execute` call, and every subsequent `Execute()` (`pkg/campaignhandler/execute.go`) schedules its own successor (500ms after a successful dial, 5s after a throttle/no-target/retry-wait condition) via `CampaignV1CampaignExecute(ctx, id, delay)`. A customer only needs one API call — `PUT /v1/campaigns/{id}/status` with `{"status": "run"}` — to start a campaign; from there it is fully automatic. There is no `time.Ticker`/cron goroutine, so "no internal timer" is technically accurate, but the operationally relevant fact is that the "scheduler" this depends on is campaign-manager's own RabbitMQ consumer, not a separate external service. If campaign-manager's consumer goes down, campaigns in `run` status stop dialing until it recovers (the self-scheduling chain resumes from the pod's next successful `Execute()` — no data is lost, but progress pauses).

### Service Level Requires Queue

The `service_level` throttle only applies when `queue_id` is set. Without a queue, all available slots are dialed immediately. Ensure campaigns intended to be throttled have a valid `queue_id`.

### Stopping State Requires Cleanup

The `stopping` transition waits for in-progress calls to complete. Do not force-stop a campaign in `stopping` state unless you are certain all associated calls have been cleaned up; orphaned calls may not be tracked properly.
