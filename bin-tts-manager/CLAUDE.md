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

**`POD_IP` is for AudioSocket advertising**: `POD_IP` (Kubernetes Downward API) is used to tell Asterisk where to dial for audio frames — it is NOT the queue host ID.

**Provider fallback order**: GCP Cloud TTS (primary) → AWS Polly (fallback). GCP uses ADC; fallback is tracked by `speech_fallback_total` metric.

**Multi-container shared volume**: Go service writes audio to `/shared-data`; Python HTTP sidecar serves them on port 80. Do not serve files directly from the Go service.

**ElevenLabs streaming sessions**: Each session runs in a dedicated goroutine with context cancellation. Keep-alive pings every 30 seconds. Failure cleans up the session automatically.

**No event subscriptions**: No SubscribeHandler — TTS is invoked synchronously via RPC only.

## Per-Pod Queue Pattern

See [docs/patterns/per-pod-queues.md](../docs/patterns/per-pod-queues.md) for the canonical pattern. Note: `bin-tts-manager` differs from `bin-transcribe-manager` in that it uses `HOSTNAME` (not `POD_IP`) as the HostID.
