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
| `GOOGLE_API_KEY` | Gemini LLM only (passed explicitly to `GoogleLLMService`) |
| `ANTHROPIC_API_KEY` | Not used today. The runner's LLM dispatch (`run.py`) handles only OpenAI, Grok and Gemini, and no deployment provisions this key. |
| `DEEPGRAM_API_KEY` | Deepgram STT |
| `CARTESIA_API_KEY` | Cartesia TTS |
| `ELEVENLABS_API_KEY` | ElevenLabs TTS |
| `GOOGLE_APPLICATION_CREDENTIALS` | Google Cloud TTS/STT. `GoogleTTSService`/`GoogleSTTService` take no api_key and fall back to Application Default Credentials, so they need a service account key file, not `GOOGLE_API_KEY`. On GKE this came implicitly from the node metadata server; on bare metal the Komodo stack mounts the shared `bin-manager` service account at `/run/secrets/google_service_account.json` (VOIP-1482). |

## Prometheus Metrics

Exposed at `PROMETHEUS_LISTEN_ADDRESS/PROMETHEUS_ENDPOINT` (default `:2112/metrics`).

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `pipecat_manager_llm_flush_exit_total` | Counter | — | LLM flush operations that exited cleanly |
| `pipecat_manager_llm_flush_finalize_outcome_total` | Counter | `outcome` | LLM flush finalization outcomes |
| `pipecat_manager_llm_idle_watchdog_fired_total` | Counter | — | Idle watchdog triggers |
| `pipecat_manager_mcp_tool_list_fallback_total` | Counter | — | runnerStartScript could not fetch the AIcall's MCP tools from ai-manager (`AIV1AIcallToolList` failed) and started the session with built-in tools only (fail-open for the MCP half; the built-in half is unaffected). A sustained non-zero rate means customers' MCP tools are silently unavailable to the LLM |
| `pipecat_manager_pipeline_error_total` | Counter | `category`, `fatal` | RTVI `error` frames from the runner, by classified category (authentication, rate_limited, timeout, function_call, internal, unknown). Counted whether or not a customer notice was published (VOIP-1542) |
| `pipecat_manager_rtvi_error_response_total` | Counter | — | RTVI `error-response` frames: the runner rejected a request pipecat-manager sent (e.g. `send-text`). A platform contract failure; any sustained rate is a bug (VOIP-1542) |
| `pipecat_manager_tool_resolve_fallback_total` | Counter | — | runnerStartScript failed CLOSED to an empty tool list after an AI lookup failure. Fail-closed by design (docs/plans/2026-07-30-case-insight-assistant-tool-expansion-design.md §2.4; this reverses the prior fail-open VOIP-1234 §6 v4 decision) since tool-access control must favor least-privilege over availability. A sustained non-zero rate should still be investigated and alerted on, since it means sessions are running with NO tools instead of the AI's configured whitelist |
| `receive_request_process_time` | Histogram | `type`, `method` | RPC request latency |

Circuit-breaker metrics from `bin-common-handler/pkg/requesthandler` are also registered under the `pipecat_manager_*` namespace. See [docs/patterns/circuit-breaker.md](../../docs/patterns/circuit-breaker.md).

**Gotcha:** do not add metric names already registered by `bin-common-handler/pkg/requesthandler/main.go#initPrometheus()` — duplicate names cause `prometheus.MustRegister` to panic at startup.

## Troubleshooting

### Gemini session with fewer tools than configured

Before a Gemini session starts, the runner validates the tool list with the installed pipecat Gemini adapter and google-genai `GenerateContentConfig` (`scripts/pipecat/gemini_tool_filter.py`, called from `run.py create_llm_service`). One tool the client-side validator rejects would otherwise fail every turn of the session, built-ins included. Only the rejected tools are dropped; OpenAI and Grok sessions are not filtered. Runner log lines (loguru, each ending in `pipeline id=<id>`):

- WARN `Dropped tool '<name>' rejected by the Gemini schema validator: <n> error(s), first: <loc>: <msg>`: one line per dropped tool. The schema itself is never logged. A dropped built-in tool is a regression signal.
- INFO `Gemini tool validation dropped <k> of <n> tools`: summary when anything was dropped.
- WARN `Gemini tool validation skipped ...`, `... failed for the filtered tool set ...` or `... failed unexpectedly ...`: the validator itself could not run or the filtered set still failed. The filter fails open and keeps every tool, so the session behaves as it would without the filter.

The dropped tool's handler is still registered by name but is inert, since the LLM never sees the tool. Server-side (HTTP 400) provider rejections are not caught by this filter. MCP tool schemas are normalized to a provider-neutral subset by bin-ai-manager before they reach the runner, so a drop here usually means a schema construct that normalization does not cover. No metric; the log lines are the signal.

### WARN `Pipeline error. fatal: ...`

The Go side handles every RTVI `error` message from the runner (pipecat's pipeline `ErrorFrame`, for example a provider request rejected before it is sent, such as a tool schema the provider refuses) in `runnerHandlePipelineError`. It classifies the text, counts it in `pipecat_manager_pipeline_error_total{category, fatal}`, and logs the first frame of each category per session at WARN with `pipecatcall_id` and `category` (repeats at DEBUG). The error text is capped at 2000 bytes; the runner's own ERROR record holds the full message. The text can include fragments of customer tool schemas (internal logs only; the customer-facing `pipeline_error` event carries only the category). To join with bin-ai-manager logs, use `pipecatcall_id`. See docs/domain.md "Pipeline errors" for the notice policy.

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

- Both Go (port 8080) and Python (port 8000) components must be running in the same network namespace — the Go side drives the Python runner at `http://localhost:8000/run`.
- The Dockerfile builds one image carrying both the Go binary and the Python pipeline (deps preinstalled); each deployment runs it twice — once as the Go service, once as the Python runner. On GKE these were two containers in one pod (`k8s/deployment.yml`); on Komodo/Compose they are the `pipecat-manager-1`/`-2` and `pipecat-script-runner-1`/`-2` services, each runner joined to its own manager via `network_mode: "service:pipecat-manager-N"`.
- Per-pod queues are declared **volatile** — they auto-delete when the pod terminates, preventing dead-letter buildup.

## Deployment (Komodo)

Komodo-managed (VOIP-1350), same mechanism as the other `bin-*-manager` services
(see bin-call-manager for the original pattern). Deployed via
`.circleci/scripts/render-image-tag.sh` + `.circleci/scripts/komodo-api-deploy.sh`
from `komodo/docker-compose.yml`, with `komodo/environment.env` passed as the
deploy script's optional third argument and PATCHed into the Stack's own
`environment` field.

That environment file carries `GCP_SA_JSON=[[BIN_MANAGER__GOOGLE_APPLICATION_CREDENTIALS_JSON]]`,
the Komodo Variable holding the shared `bin-manager` Google service account key.
Compose materializes it as the `gcp_sa_json` secret at
`/run/secrets/google_service_account.json` in both runners (VOIP-1482).

**If that Variable is missing, the deploy fails and takes the runners down.**
Compose refuses to start a service whose secret has no source, so `docker compose up`
exits non-zero and the deploy goes red, but the runner containers have already been
recreated and are left in `Created`. `restart: always` never applies to a container
that never reached Running, so both runners stay down and every `ai_talk` fails,
which is worse than the Google-only breakage this change fixes. Treat a failed
deploy here as a live outage: set the Variable and redeploy.

Three deviations from the Tier 1/2 template, all intentional:
- **Non-distroless runtime** (`python:3.12-slim`, needed to run the Python
  Pipecat pipeline) — the healthcheck uses `python3 -c "import urllib..."`
  instead of the fleet-standard `wget` CMD, since `python:3.12-slim` has
  neither `wget` nor `curl`.
- **`POD_IP` is not a Komodo Variable.** It was originally meant to be set via
  the K8s Downward API (`status.podIP`) for per-pod queue routing, but that
  wiring was never carried over to Docker Compose — `install/`'s own
  `docker-compose.yml.dist` already fell back to the literal string
  `pipecat-manager`. The Komodo compose file keeps that same literal
  (`POD_IP=pipecat-manager`), matching current production behavior exactly.
  A real per-container unique `HostID` (needed if this service is ever
  scaled to multiple replicas) is a separate follow-up, not part of this
  cutover.
- **`pipecat-script-runner` sidecar** (added 2026-08-22,
  NOJIRA-Fix-pipecat-runner-sidecar): the Python Pipecat pipeline runs as
  a second service from the same image (`python /app/scripts/pipecat/main.py`,
  uvicorn on 0.0.0.0:8000), sharing the Go container's network namespace
  via `network_mode: "service:pipecat-manager-N"` so localhost:8000 works
  exactly as it did in the GKE pod. It was dropped in the original Komodo
  cutover, which made every `ai_talk` action fail with connection-refused
  on localhost:8000 and tear the call down. It receives the
  STT/LLM/TTS API-key env vars the GKE runner container had, plus the
  Google service account key GKE used to supply implicitly through the
  node metadata server (mounted as a Compose secret, VOIP-1482).
