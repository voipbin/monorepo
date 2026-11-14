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

## Platform keys

- The runner reads the vendor keys from the environment variables of its own container: `OPENAI_API_KEY`, `DEEPGRAM_API_KEY`, `CARTESIA_API_KEY`, `XAI_API_KEY`, `ELEVENLABS_API_KEY`, `GOOGLE_API_KEY`, `OPENROUTER_API_KEY`, `LOCAL_LLM_ENDPOINTS` (compose value `[[BIN_MANAGER__<name>]]`, runner services only). The names are fixed (shared with ai-manager and tts-manager). The Go containers get none of them, read no key and send no key in the request; `cmd/pipecat-manager/komodo_compose_test.go` pins which service has which variable.
- `run.py` `_platform_key(platform_keys, name, vendor, env_name)`: the key of the request when it has one (kept so that a new runner works next to an older Go service), else the variable (trimmed; empty or a value that starts with `[[` is not set), else `<vendor> is not configured` before the vendor SDK object is built, because the SDK clients read their own variable when they get `None` or an empty key. OpenAI, Grok and Gemini prefer the key of the customer and then do not look up the platform variable; platform OpenRouter and local never use a customer key. The local endpoints are read for each session (request value, else `LOCAL_LLM_ENDPOINTS`); no module registry, nothing read at import. `log_env_state()` logs set / not set / unresolved placeholder per variable once, after `load_dotenv(override=True)`, never a value. A clean checkout has no `.env` file in `scripts/pipecat` (git-ignored); a local build context that contains one would override the container environment (`load_dotenv(override=True)`).
- The stored settings (`pipecat_configs`, `internal/configkeys`, `pipecat-control config`, the admin API `/service_configs/pipecat`) and `pipecatconfig.NewKeyResolver` / `LogSources` stay until a later removal and are not called by the services. `config set` and `config clear` print a stderr note with the variable that is read. Do not add `Env` constants to `configkeys`.
- The map type of the request path is `map[string]amai.RedactedString` (plain only in the JSON body); Python declares `platform_keys` with `repr=False`. Do not change either to a plain string. Never log the request body.
- The master key list `PIPECAT_CONFIG_ENCRYPTION_KEYS` is mapped to the Go containers (`pipecat-manager-1`, `-2`) only; the runner has no database access and no master key.
- See [docs/operations.md](docs/operations.md) (Platform keys).

## Testing pattern

gomock (go.uber.org/mock) + table-driven tests. Both Go and Python components must be running for integration tests.
