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

# Generate mocks
go generate ./...

# Verification (mandatory before commit)
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

## Critical Implementation Notes

**Per-pod queue routing uses `POD_IP`**: The `HostID` for per-pod queue is `POD_IP` (Kubernetes Downward API `status.podIP`). Control RPCs targeting an active session (`stop`, `health-check`) must be routed to `bin-manager.transcribe-manager.request.<POD_IP>`. Contrast with `bin-tts-manager` which uses `HOSTNAME`.

**Calico POD_IP recycle limitation**: If a pod restarts and gets a new IP, sessions from the old pod are orphaned. See [docs/patterns/per-pod-queues.md](../docs/patterns/per-pod-queues.md).

**Session map locking**: Always lock/unlock `muStreaming` when accessing `mapStreaming`. Failure to lock causes data races under concurrent streaming session operations.

**Provider fallback**: Default order `gcp` → `aws`. At least one must be configured at startup. `provider` field in request overrides order.

**Status validation**: Use `models/transcribe/transcribe.go:IsUpdatableStatus` before any status transition. `done` sessions cannot be restarted.

**WebSocket audio transport**: Go dials out to Asterisk's `chan_websocket` endpoint (`MediaURI` from `ExternalMediaStart`). Connection type: `server`, transport: `websocket`, encapsulation: `none`. Raw 8 kHz slin binary frames.

**Language codes**: Must be valid BCP47 format (e.g., `en-US`, `ko-KR`). Validate at session creation.

**Uses Cobra + Viper** — see `internal/config/main.go`. Config is singleton, loaded in `PersistentPreRunE`.

## Adding New Transcribe Operations

1. Add URL regex to `pkg/listenhandler/main.go`
2. Implement handler in `pkg/listenhandler/v1_transcribes.go`
3. Add business logic to `pkg/transcribehandler/transcribe.go`
4. Add DB methods to `pkg/dbhandler/transcribe.go` if persistence needed
5. Emit notifications via `notifyhandler` for state changes
