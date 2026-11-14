# bin-pipecat-manager

Hybrid Go/Python service for real-time AI voice pipeline execution. Go owns transport, session lifecycle, and DB persistence. Python runs the Pipecat pipeline (STT → LLM → TTS).

> Cross-cutting rules (verification workflow, branch/commit format, worktree usage, Alembic, RST sync) live in the root [CLAUDE.md](../CLAUDE.md).

## Docs index

- [docs/architecture.md](docs/architecture.md) — component layout, request routing (shared + per-pod queues), session lifecycle
- [docs/domain.md](docs/domain.md) — pipecatcall model, protobuf frames, audio architecture, tool execution
- [docs/dependencies.md](docs/dependencies.md) — local monorepo deps, external services, queue names, Python deps
- [docs/operations.md](docs/operations.md) — config flags, Prometheus metrics, CLI tool, common commands
- [docs/plans/](docs/plans/) — design documents for past non-trivial changes

## Key concepts

- **Pipecatcall** — one AI voice session; one MySQL record + one in-memory session anchored to a single pod
- **HostID** = `POD_IP` (set per container in `komodo/docker-compose.yml`: `pipecat-manager-1` / `pipecat-manager-2`); must be set for per-pod queue routing — follow-up RPCs from `bin-ai-manager` use `pipecatcall.HostID`
- **Per-pod queue** — `bin-manager.pipecat-manager.request.<POD_IP>` (volatile); see [docs/patterns/per-pod-queues.md](../docs/patterns/per-pod-queues.md)
- **16 kHz end-to-end** — `audio_out_sample_rate=16000` in Python `PipelineParams` is mandatory; Pipecat defaults to 24 kHz and will cause robotic audio without this setting

## CRITICAL: audio sample rate

Always keep `audio_out_sample_rate=16000` in `PipelineParams` in `scripts/pipecat/run.py`. Removing it causes resampling artifacts. See [docs/domain.md#audio-sample-rate-critical](docs/domain.md#audio-sample-rate-critical) and [docs/plans/2026-01-22-audio-resampling-design.md](docs/plans/2026-01-22-audio-resampling-design.md).

## CRITICAL: Prometheus metric names

Before adding metrics, check `bin-common-handler/pkg/requesthandler/main.go#initPrometheus()` for existing names. Duplicate registration causes panic at startup.

## Common commands

```bash
# Go: full verification (mandatory before every commit)
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

# Go: build
go build -o ./bin/ ./cmd/...

# Protobuf: regenerate (only when proto/frames.proto changes)
protoc --go_out=. --go_opt=paths=source_relative proto/frames.proto

# Python: run local
cd scripts/pipecat && pip install -r requirements.txt && uvicorn main:app --host 0.0.0.0 --port 8000
```

## Database migrations

The schema of `pipecat_pipecatcalls`, `pipecat_configs` and `pipecat_config_histories` is owned by this service and applied at startup (golang-migrate, embedded with `go:embed`). It does not use Alembic. Tested engines: MariaDB and MySQL 8.0, so do not use engine specific syntax.

- Files: `migrations/NNNNNN_<description>.up.sql` and `.down.sql` (six digits, from 000001 without gaps). `000001_baseline` is the schema at the time of the conversion (`CREATE TABLE IF NOT EXISTS`, no table options, down is the no-op `SELECT 1`). Version table: `pipecat_schema_migrations`.
- `cmd/pipecat-manager` runs `migrationhandler.Run` in `createDBHandler`, right after the database connection and before the cache and RabbitMQ. Failure stops the start (non-zero exit). The Python script-runner containers do not touch the database.
- `pipecat-control migrate status|up|down|force <version>` talks to the database only. Run `up`, `down` and `force` only against a local throwaway database.
- The migration code is the shared `bin-common-handler/pkg/migrationhandler`. Rules for new files (one DDL statement per file, `-- DESTRUCTIVE:` and `-- DOWN-LOSSY:` markers, immutable merged files, expand/contract), the checks and the recovery steps are in [docs/workflows/service-owned-migrations.md](../docs/workflows/service-owned-migrations.md).
- A new database is created with `CHARACTER SET utf8 COLLATE utf8_general_ci` (like the install scripts do).
- New database: the service creates its tables at startup (the baseline is `CREATE TABLE IF NOT EXISTS`); no other schema tool runs first.
- `000002` and `000003` add the two config tables (CHECK constraints are `chk_pipecat_configs_*` and `chk_pipecat_config_histories_*`).
- Update `scripts/database_scripts_test/table_*.sql` (the sqlite test schema) by hand when the schema changes.
- Structure checks: `go test ./migrations/`. Git relative check: `scripts/check-service-migrations.sh bin-pipecat-manager` after `git fetch origin main`. Dynamic check on scratch containers: `scripts/verify-service-migrations.sh bin-pipecat-manager` (repo root).

## Stored platform keys

- Eight vendor keys can be stored encrypted in `pipecat_configs` (`internal/configkeys`): `openai.api_key`, `deepgram.api_key`, `cartesia.api_key`, `xai.api_key`, `elevenlabs.api_key`, `google.api_key`, `openrouter.api_key`, `local_llm.endpoints`. The key names are also the keys of `platform_keys` in the runner request, so they never change.
- Source: the database only. The handler gets a nil environment lookup (`pipecatconfig.New`) and no key declares an environment variable; a key without a stored row is not configured, and a variable of the old name (for example `OPENAI_API_KEY`) in either container has no effect. The runner uses the keys of its request only: `_platform_key` raises `<vendor> is not configured` before the vendor SDK object is built, because the SDK clients read the process environment when they get `None` or an empty key. OpenAI, Grok and Gemini prefer the key of the customer; platform OpenRouter and local never use a customer key. The local endpoints are parsed from the request value for each session (no module registry, nothing read at import).
- `runnerStartScript` calls `sessionPlatformKeys` once, for a single AI and a team alike, and passes the map to `PythonRunner.Start`. Only the vendors that the session uses are resolved. Reads are bounded to 2 seconds in total; a key that can not be read is left out with one warning (key name, pipecatcall id, reason; never a value).
- The map type is `map[string]amai.RedactedString` (plain only in the JSON body); Python declares `platform_keys` with `repr=False`. Do not change either to a plain string. Never log the request body.
- The master key list `PIPECAT_CONFIG_ENCRYPTION_KEYS` is mapped to the Go containers (`pipecat-manager-1`, `-2`) only. The runner has no database access and no master key; `cmd/pipecat-manager/komodo_compose_test.go` pins this.
- Manage with `pipecat-control config list|set|clear` or the admin API (`/service_configs/pipecat`). See [docs/operations.md](docs/operations.md) (Stored platform keys).

## Testing pattern

gomock (go.uber.org/mock) + table-driven tests. Both Go and Python components must be running for integration tests.
