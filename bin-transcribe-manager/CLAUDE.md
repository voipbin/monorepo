# bin-transcribe-manager

Real-time speech-to-text transcription service for VoIP calls and conferences. Integrates GCP Speech-to-Text and AWS Transcribe, with per-pod in-memory streaming sessions and WebSocket transport to Asterisk.

> Cross-cutting rules (verification, branch/commit format, worktrees, Alembic, RST sync) live in the root [CLAUDE.md](../CLAUDE.md).

## Docs

- [docs/architecture.md](docs/architecture.md) — component overview, layer responsibilities, request routing
- [docs/domain.md](docs/domain.md) — Transcribe and Transcript entities, provider selection, per-pod anchoring
- [docs/dependencies.md](docs/dependencies.md) — upstream services, subscribed queues, STT providers
- [docs/operations.md](docs/operations.md) — failure modes, debugging, configuration, metrics
- [docs/plans/](docs/plans/) — dated design documents (preserved — do not delete)

## Common Commands

```bash
# Build
go build -o ./bin/transcribe-manager ./cmd/transcribe-manager

# Test
go test -v ./...
go test -coverprofile cp.out -v $(go list ./...)

# transcribe-control operations
./bin/transcribe-control transcribe list --customer_id <uuid>
./bin/transcribe-control transcribe get --id <uuid>
./bin/transcribe-control transcribe stop --id <uuid>
./bin/transcribe-control migrate status

# Generate mocks
go generate ./...

# Verification (mandatory before commit)
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

## Database migrations

The schema of `transcribe_transcribes` and `transcribe_transcripts` is owned by this service and applied at startup (golang-migrate, embedded with `go:embed`). It does not use Alembic. Tested engines: MariaDB and MySQL 8.0, so do not use engine specific syntax.

- Files: `migrations/NNNNNN_<table>_<verb>_<type>_<items>.up.sql` and `.down.sql` (six digits, from 000001 without gaps). `000001_baseline` is the schema at the time of the conversion.
- `cmd/transcribe-manager` runs `Up()` right after the database connection, before the cache and RabbitMQ. Failure stops the start (non-zero exit). Version table: `transcribe_schema_migrations`.
- Rules for every new migration:
  - One DDL statement per file (DDL is not transactional; a failure in a later statement would leave the first applied with the database marked dirty). Only the baseline may hold several `CREATE TABLE IF NOT EXISTS` statements.
  - Every up has a down. A down that cannot restore data starts with `-- DOWN-LOSSY: <reason>`. An up with a destructive operation (DROP, TRUNCATE, RENAME, MODIFY/CHANGE COLUMN, UPDATE/DELETE/INSERT SELECT, adding UNIQUE/FK/CHECK/NOT NULL, SET/DROP DEFAULT, ENGINE/CHARSET/COLLATE/ROW_FORMAT, ON DUPLICATE KEY UPDATE) starts with `-- DESTRUCTIVE: <reason>` and its PR body starts with `[DESTRUCTIVE]`.
  - Merged files are immutable and numbers are never reused: fix mistakes with a new number, and a revert PR changes the code only, never `migrations/`. A number lower than the highest merged one is silently skipped on databases that are already ahead.
  - Expand/contract: the old code must work on the new schema. Add columns as NULL or with a DEFAULT and list columns explicitly in queries; add indexes with `ALGORITHM=INPLACE, LOCK=NONE`; drop a column in a later PR than the one that stops using it; replace instead of rename or retype (add, dual write, switch reads, drop); backfill before NOT NULL/UNIQUE/FK/CHECK. Large data changes do not belong in a migration file.
  - Update `scripts/database_scripts_test/table_transcribes.sql` and `scripts/database_scripts_test/table_transcripts.sql` (the sqlite test schema) by hand when the schema changes.
- Safety behavior: the migration session uses `lock_wait_timeout=10`; two instances serialize on a MySQL named lock (an instance that cannot get it within 10 seconds fails while opening the migrator and is restarted); a database version higher than the image's newest file (not dirty) is skipped with a warning so an older image can still start; a dirty database refuses to start.
- A new database is created with `CHARACTER SET utf8 COLLATE utf8_general_ci` (like the install scripts do); other defaults make the table definitions differ from the Alembic head.
- New database order: Alembic first, then this service. The reverse fails in the Alembic stream (`Table 'transcribe_transcribes' already exists`) and is unsupported while the Alembic stream still creates these tables.
- The migration code is the shared `bin-common-handler/pkg/migrationhandler` (`Run`, `Migrator`, `NewMigrateCommand`); this service holds only the SQL files, `migrations/embed.go`, the call in `cmd/transcribe-manager` and the `migrate` command registration in `cmd/transcribe-control`.
- Structure checks: `go test ./migrations/` runs `migrationcheck.Check` and the no stray non-.sql file test. Git relative check: `scripts/check-service-migrations.sh bin-transcribe-manager` after `git fetch origin main`. Dynamic check on scratch containers: `ALEMBIC_VENV=<venv> scripts/verify-service-migrations.sh bin-transcribe-manager` (repo root).
- Recovery from a dirty database: stop the crash-looping container, check whether the failed file's statement was applied, then `transcribe-control migrate force <N-1 or N>` (`transcribe-control migrate force -- -1` resets to unversioned).
- Run `transcribe-control migrate up|down|force` only against a local throwaway database. `status` still creates the version table when it is missing.
- `transcribe-control migrate status|up|down|force <version>` talks to the database only (no Redis or RabbitMQ). `down` is for development and emergencies and is never run automatically; the baseline down is a no-op.

## Critical Implementation Notes

**Per-pod queue routing uses a random `HostID` UUID**: `cmd/transcribe-manager/main.go` generates `hostID := uuid.Must(uuid.NewV4())` once at process startup — it is not `POD_IP` or any other pod identity. Control RPCs targeting an active session (`stop`, `health-check`) must be routed to `bin-manager.transcribe-manager-<host_id>.request` (note the hyphen before the UUID and the `.request` suffix at the end, not a dot-suffixed `<host_id>` after `.request`). Contrast with `bin-tts-manager` which uses `HOSTNAME`.

**HostID changes on every restart**: Because `hostID` is a fresh random UUID generated each time the process starts (not tied to pod IP or hostname), the per-pod queue name changes on every restart, not only when the underlying pod IP changes. Sessions from before a restart are orphaned — the old queue is gone and any caller still holding the old `host_id` must treat it as unreachable. See [docs/patterns/per-pod-queues.md](../docs/patterns/per-pod-queues.md) for the general per-pod queue pattern (used there with a `POD_IP`-based identity by other services; transcribe-manager instead uses a process-lifetime random UUID).

**Session map locking**: Always lock/unlock `muStreaming` when accessing `mapStreaming`. Failure to lock causes data races under concurrent streaming session operations.

**Provider fallback**: Default order `gcp` → `aws`. `provider` field in request overrides order.

**No STT provider is not fatal**: if neither GCP nor AWS initializes at startup, `streaminghandler.NewStreamingHandler` returns the disabled implementation (`NewDisabledStreamingHandler`) rather than `nil`. The service boots normally and every non-streaming capability (transcribe CRUD, transcript reads, listen/subscribe handlers) stays available; only `Start`/`Stop` on the streaming path fail, with a structured `*cerrors.VoipbinError` (`Status: Unavailable`, `Reason: "STT_NOT_CONFIGURED"`, domain `bin-transcribe-manager`) — this follows the same structured-error convention as `pkg/transcribehandler/start.go`'s `TRANSCRIBE_ALREADY_PROGRESSING`, so it round-trips correctly through `pkg/listenhandler`'s `errorResponse()` as a typed API error rather than an opaque 500. Never reintroduce a `streamingHandler == nil` check in `cmd/transcribe-manager/main.go` (or `cmd/transcribe-control/main.go` — both had the same check removed) — the constructor cannot return nil, and the disabled implementation's `Run()` must keep returning `nil` or startup aborts again.

**Status validation**: Use `models/transcribe/transcribe.go:IsUpdatableStatus` before any status transition. `done` sessions cannot be restarted.

**Transcribe ids are no longer guaranteed-unguessable server-generated v4 UUIDs**: `transcribehandler.Start` accepts an optional caller-supplied `id`. A caller can set a transcribe's `id` equal to another resource's id (e.g. a call id), which makes log/timeline correlation by "looks like a call id" unreliable going forward — always confirm via the actual `reference_id`/`reference_type` fields, not by eyeballing whether an id "looks like" a particular resource type. See `docs/plans/2026-09-03-caller-specified-transcribe-id-design.md` for the full design.

**WebSocket audio transport**: Go dials out to Asterisk's `chan_websocket` endpoint (`MediaURI` from `ExternalMediaStart`). Connection type: `server`, transport: `websocket`, encapsulation: `none`. Raw 8 kHz slin binary frames.

**Language codes**: Must be valid BCP47 format (e.g., `en-US`, `ko-KR`). Validate at session creation.

**Uses Cobra + Viper** — see `internal/config/main.go`. Config is singleton, loaded in `PersistentPreRunE`.

## Adding New Transcribe Operations

1. Add URL regex to `pkg/listenhandler/main.go`
2. Implement handler in `pkg/listenhandler/v1_transcribes.go`
3. Add business logic to `pkg/transcribehandler/transcribe.go`
4. Add DB methods to `pkg/dbhandler/transcribe.go` if persistence needed
5. Emit notifications via `notifyhandler` for state changes
