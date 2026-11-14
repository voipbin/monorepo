# bin-tts-manager

Text-to-speech synthesis service with two modes: batch (pre-recorded file generation) and real-time streaming (ElevenLabs WebSocket → Asterisk AudioSocket). Multi-container pod with Go service + Python HTTP sidecar.

> Cross-cutting rules (verification, branch/commit format, worktrees, Alembic, RST sync) live in the root [CLAUDE.md](../CLAUDE.md).

## Docs

- [docs/architecture.md](docs/architecture.md) — component overview, dual-mode architecture, request routing
- [docs/domain.md](docs/domain.md) — Speech and Streaming entities, provider selection, session lifecycle
- [docs/dependencies.md](docs/dependencies.md) — upstream services, external TTS providers, infrastructure
- [docs/operations.md](docs/operations.md) — failure modes, debugging, configuration, metrics

## Common Commands

```bash
# Build
go build -o ./bin/ ./cmd/...

# Test
go test ./...
go test -coverprofile cp.out -v $(go list ./...)

# Generate mocks
go generate ./pkg/ttshandler/...
go generate ./pkg/streaminghandler/...
go generate ./pkg/audiohandler/...
go generate ./pkg/buckethandler/...
go generate ./pkg/cachehandler/...

# Verification (mandatory before commit)
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

## Critical Implementation Notes

**Per-pod queue routing uses `HOSTNAME`** (not `POD_IP`): The `HostID` for the per-pod queue is `HOSTNAME` (Kubernetes pod name). Streaming control RPCs (`say_init`, `say_add`, `say_stop`, `say_finish`) must be routed to `bin-manager.tts-manager.request.<HOSTNAME>`.

**`POD_IP` is the batch-TTS media URL host**: `POD_IP` feeds `buckethandler`'s media URL (`http://<host>/<file>`, fetched by Asterisk) — it is NOT the queue host ID. On GKE it is the pod IP (Downward API), rewritten to the pod-DNS form; on Docker Compose (<prod-host>) it is the media http sidecar's Compose service name (`tts-manager-http`), used verbatim. It is the only `POD_IP` consumer in this service (`cmd/tts-manager/main.go`).

**Provider fallback order**: GCP Cloud TTS (primary) → AWS Polly (fallback). GCP uses ADC; fallback is tracked by `speech_fallback_total` metric.

**Provider keys are read from the environment** (`internal/configkeys`, `internal/ttsconfig`): `AWS_ACCESS_KEY`, `AWS_SECRET_KEY` and `ELEVENLABS_API_KEY` are read per request through `ttsconfig.AWSCredentials()` and `ttsconfig.ElevenlabsKey()` (`bin-common-handler/pkg/envsecret`: not set, empty or an unresolved `[[...]]` placeholder means not configured, an empty key and no error); nothing is cached except the Polly client of the last key pair. A pair with only one AWS variable set is refused (`ErrKeyUnavailable` wrapping `ErrAWSPairIncomplete`) and logged once at start by `ttsconfig.LogStates`. Nothing is stored in the database and no admin API writes the keys. Do not pass key strings to the handlers; pass the supplier functions. The variables are shared with `bin-transcribe-manager` (AWS) and `bin-pipecat-manager` (ElevenLabs); roll back the image together with its deployment definition. The code does not read `TTS_CONFIG_ENCRYPTION_KEYS`; the deployment definition still passes it until a later change removes the line. See [docs/operations.md](docs/operations.md#provider-keys).

**Multi-container shared volume**: Go service writes audio to `/shared-data`; Python HTTP sidecar serves them on port 80. Do not serve files directly from the Go service.

**ElevenLabs streaming sessions**: Each session runs in a dedicated goroutine with context cancellation. Keep-alive pings every 30 seconds. Failure cleans up the session automatically.

**No event subscriptions**: No SubscribeHandler — TTS is invoked synchronously via RPC only.

## Database migrations

The schema of `tts_manager_speaking`, `tts_configs` and `tts_config_histories` (the last two are no longer read or written by the code) is owned by this service and applied at startup (golang-migrate, embedded with `go:embed`, shared code in `bin-common-handler/pkg/migrationhandler`). It does not use Alembic. Tested engines: MariaDB and MySQL 8.0, so do not use engine specific syntax. The full rules (file naming, one DDL statement per file, `-- DESTRUCTIVE:` and `-- DOWN-LOSSY:` markers, immutable merged files, expand/contract, baseline text rules) are in [docs/workflows/service-owned-migrations.md](../docs/workflows/service-owned-migrations.md).

- Files: `migrations/NNNNNN_<table>_<verb>_<type>_<items>.up.sql` and `.down.sql`. `000001_baseline` is the schema at the time of the conversion; its down is a no-op. Version table: `tts_schema_migrations`.
- `cmd/tts-manager` runs `migrationhandler.Run` right after the database connection, before the cache and RabbitMQ. Failure stops the start (non-zero exit).
- The configs tables are platform wide (no customer column) and have no table options. `tts_manager_speaking` keeps the table option `DEFAULT CHARSET=utf8` (written as `DEFAULT CHARSET=utf8mb3`); the structure test `migrations/migrations_test.go` lists it.
- `scripts/database_scripts_test/` holds the sqlite test schema of the two configs tables (used by the config store tests); update it by hand when those tables change. `tts_manager_speaking` has no sqlite test schema.
- `tts-control create` reads the provider keys from the environment; it does not use the database.
- `tts-control migrate status|up|down|force <version>` talks to the database only (no Redis or RabbitMQ). Run `up`, `down` and `force` only against a local throwaway database or as the recovery step in [docs/operations.md](docs/operations.md).
- Verification: `go test ./migrations/` (structure) and, from the repo root, `scripts/verify-service-migrations.sh bin-tts-manager` (both engines).

## Per-Pod Queue Pattern

See [docs/patterns/per-pod-queues.md](../docs/patterns/per-pod-queues.md) for the canonical pattern. Note: `bin-tts-manager` uses `HOSTNAME` as the HostID, while `bin-transcribe-manager` uses a random UUID generated fresh at each process start (not `POD_IP`).
