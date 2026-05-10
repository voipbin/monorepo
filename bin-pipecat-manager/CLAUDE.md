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
- **HostID** = `POD_IP` (K8s Downward API); must be set for per-pod queue routing — follow-up RPCs from `bin-ai-manager` use `pipecatcall.HostID`
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

## Testing pattern

gomock (go.uber.org/mock) + table-driven tests. Both Go and Python components must be running for integration tests.
