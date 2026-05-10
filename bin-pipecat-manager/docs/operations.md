# bin-pipecat-manager Operations

## Configuration

All Go flags support equivalent `UPPER_SNAKE_CASE` environment variables.

| Flag | Env | Description | Required |
|------|-----|-------------|----------|
| `rabbitmq_address` | `RABBITMQ_ADDRESS` | RabbitMQ connection URL | yes |
| `database_dsn` | `DATABASE_DSN` | MySQL DSN | yes |
| `redis_address` | `REDIS_ADDRESS` | Redis host:port | yes |
| `redis_password` | `REDIS_PASSWORD` | Redis auth | no |
| `redis_database` | `REDIS_DATABASE` | Redis DB index | no |
| `prometheus_endpoint` | `PROMETHEUS_ENDPOINT` | Metrics path | `/metrics` |
| `prometheus_listen_address` | `PROMETHEUS_LISTEN_ADDRESS` | Metrics listen address | `:2112` |
| `POD_IP` | `POD_IP` | Pod IP (K8s Downward API); used as HostID for per-pod routing | yes |

Python environment variables (set in `.env` or exported):

| Env | Purpose |
|-----|---------|
| `OPENAI_API_KEY` | OpenAI LLM |
| `XAI_API_KEY` | Grok (xAI) LLM |
| `GOOGLE_API_KEY` | Gemini LLM + Google TTS |
| `ANTHROPIC_API_KEY` | Anthropic Claude |
| `DEEPGRAM_API_KEY` | Deepgram STT |
| `CARTESIA_API_KEY` | Cartesia TTS |
| `ELEVENLABS_API_KEY` | ElevenLabs TTS |

## Prometheus Metrics

Exposed at `PROMETHEUS_LISTEN_ADDRESS/PROMETHEUS_ENDPOINT` (default `:2112/metrics`).

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `pipecat_manager_llm_flush_exit_total` | Counter | — | LLM flush operations that exited cleanly |
| `pipecat_manager_llm_flush_finalize_outcome_total` | Counter | `outcome` | LLM flush finalization outcomes |
| `pipecat_manager_llm_idle_watchdog_fired_total` | Counter | — | Idle watchdog triggers |
| `receive_request_process_time` | Histogram | `type`, `method` | RPC request latency |

Circuit-breaker metrics from `bin-common-handler/pkg/requesthandler` are also registered under the `pipecat_manager_*` namespace. See [docs/patterns/circuit-breaker.md](../../docs/patterns/circuit-breaker.md).

**Gotcha:** do not add metric names already registered by `bin-common-handler/pkg/requesthandler/main.go#initPrometheus()` — duplicate names cause `prometheus.MustRegister` to panic at startup.

## CLI Tool: pipecat-control

`cmd/pipecat-control` — direct DB/cache management. All output is JSON on stdout.

```bash
./bin/pipecat-control pipecatcall get       --id <uuid>
./bin/pipecat-control pipecatcall start     --reference_type <type> --reference_id <uuid>
./bin/pipecat-control pipecatcall terminate --id <uuid>
./bin/pipecat-control pipecatcall send-message --id <uuid> --message <text>
```

Requires `soxr` system library installed.

## Common Commands

```bash
# Go: build
go build -o ./bin/ ./cmd/...

# Go: test with coverage
go test -coverprofile cp.out -v $(go list ./...)
go tool cover -html=cp.out -o cp.html

# Go: regenerate mocks
go generate ./pkg/pipecatcallhandler/...
go generate ./pkg/dbhandler/...
go generate ./pkg/cachehandler/...

# Go: full verification (mandatory before every commit)
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

# Protobuf: regenerate frames (only when modifying proto/frames.proto)
protoc --go_out=. --go_opt=paths=source_relative proto/frames.proto

# Python: install dependencies
cd scripts/pipecat && pip install -r requirements.txt

# Python: run the FastAPI service (port 8000)
cd scripts/pipecat && uvicorn main:app --host 0.0.0.0 --port 8000
```

## Deployment Notes

- Both Go (port 8080) and Python (port 8000) components must be running on the same pod.
- `POD_IP` must be set via the K8s Downward API (`status.podIP`).
- The Dockerfile builds both Go and Python components; Python service is started alongside the Go binary via process supervisor.
- Per-pod queues are declared **volatile** — they auto-delete when the pod terminates, preventing dead-letter buildup.
