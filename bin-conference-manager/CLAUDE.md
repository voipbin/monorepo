# bin-conference-manager

## Overview

`bin-conference-manager` manages multi-party audio conference sessions in VoIPbin. It owns the Conference and Conferencecall (participant) entities, coordinates with call-manager's confbridge for the actual audio mixing, and handles conference lifecycle operations including recording and transcription. It is a Class A standard Go RPC manager.

> Cross-cutting rules (verification workflow, branch/commit format, worktree usage, Alembic, RST sync) live in the root [CLAUDE.md](../CLAUDE.md). This file documents only what is specific to `bin-conference-manager`.

## Key Concepts

- **Conference**: Top-level session entity; types are `conference` (manual stop), `connect` (auto-stops when 1 participant left), `queue` (managed by queue-manager)
- **Conferencecall**: A participant in a conference; tracks the `reference_id` (call UUID) and join/leave status
- **Confbridge**: The actual audio-mixing bridge in call-manager; linked via `confbridge_id` on the conference — this service does NOT own confbridge
- **Recording**: Recording sessions are managed by call-manager; this service initiates/stops them and stores the `recording_id`
- **Transcription**: Live transcription started/stopped via bin-transcribe-manager; `transcribe_id` stored on the conference
- **Pre/post flows**: Optional flow IDs executed before participants speak and after the conference ends

## Common Commands

| Command | Purpose |
|---------|---------|
| `cd bin-conference-manager && go build ./...` | Compile |
| `go test ./...` | Run all tests |
| `go test -v ./pkg/conferencehandler/...` | Test a specific package |
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

## Database migrations

The schema of `conference_conferencecalls` and `conference_conferences` is owned by this service and applied at startup (golang-migrate, embedded with `go:embed`). It does not use Alembic. Tested engines: MariaDB and MySQL 8.0, so do not use engine specific syntax.

- Files: `migrations/NNNNNN_<table>_<verb>_<type>_<items>.up.sql` and `.down.sql` (six digits, from 000001 without gaps). `000001_baseline` is the schema at the time of the conversion. Version table: `conference_schema_migrations`.
- `cmd/conference-manager` runs `migrationhandler.Run` in `runMain`, right after the database connection and before the cache connection, RabbitMQ and listening. A failure makes the process exit with a non-zero code.
- The rules for new migrations (one DDL statement per file, `-- DESTRUCTIVE:` and `-- DOWN-LOSSY:` markers, immutable merged files, expand/contract), the recovery from a dirty database and the verification scripts are the same for every service that owns its schema: see [docs/workflows/service-owned-migrations.md](../docs/workflows/service-owned-migrations.md) and the migration section of [bin-tag-manager/CLAUDE.md](../bin-tag-manager/CLAUDE.md).
- Structure checks: `go test ./migrations/`. Git relative check: `scripts/check-service-migrations.sh bin-conference-manager` after `git fetch origin main`. Dynamic check on scratch containers: `ALEMBIC_VENV=<venv> scripts/verify-service-migrations.sh bin-conference-manager` (repo root).
- `conference-control migrate status|up|down|force <version>` talks to the database only (no Redis or RabbitMQ). The DSN comes from the `DATABASE_DSN` environment variable (the `--database_dsn` flag belongs to the root command only and is not accepted after `migrate`). Run `up|down|force` only against a local throwaway database; `down` is never run automatically and the baseline down is a no-op.
- Update `scripts/database_scripts_test/*.sql` (the sqlite test schema) by hand when the schema changes.
- Limit of the baseline: `CREATE TABLE IF NOT EXISTS` does not compare an existing table with the baseline, so a database whose tables differ from the Alembic head starts as version 1 without any warning. Compare the normalized dump of an existing database with the head before the first deployment (`REFERENCE_DUMP` in `scripts/verify-service-migrations.sh`).
- New database order: Alembic first, then this service. The reverse fails in the Alembic stream (`Table 'conference_conferences' already exists`) and is unsupported while the Alembic stream still creates these tables.

## CRITICAL Rules

### Confbridge Ownership

This service does NOT own confbridges — call-manager does. When creating a conference, the confbridge must be created in call-manager first; the returned `confbridge_id` is stored on the conference. Do not attempt to manage confbridges directly.

### Event-Driven Participant State

Conferencecall status is updated by consuming call-manager events (confbridge join/leave) via `subscribehandler`. Do not poll call-manager for participant state. If participant events are lost, conferencecalls may remain in `initiating` state indefinitely.

### Type-Specific Stop Logic

Only send `stop` to a conference when explicitly requested. The `connect` type auto-terminates via call-manager confbridge logic — do not add duplicate stop logic for this type in this service.
