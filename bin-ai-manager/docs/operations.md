# bin-ai-manager Operations

## Configuration

All flags support equivalent `UPPER_SNAKE_CASE` environment variables.

| Flag | Env | Description | Required |
|------|-----|-------------|----------|
| `rabbitmq_address` | `RABBITMQ_ADDRESS` | RabbitMQ connection URL | yes |
| `database_dsn` | `DATABASE_DSN` | MySQL DSN | yes |
| `redis_address` | `REDIS_ADDRESS` | Redis host:port | yes |
| `redis_password` | `REDIS_PASSWORD` | Redis auth | no |
| `redis_database` | `REDIS_DATABASE` | Redis DB index | no |
| `engine_key_chatgpt` | `ENGINE_KEY_CHATGPT` | OpenAI API key | yes |
| `google_api_key` | `GOOGLE_API_KEY` | Google API key for Gemini audit evaluation | yes |
| `aicall_conversation_idle_timeout_hours` | `AICALL_CONVERSATION_IDLE_TIMEOUT_HOURS` | Hours before idle AIcall expires | no |
| `prometheus_endpoint` | `PROMETHEUS_ENDPOINT` | Metrics path | `/metrics` |
| `prometheus_listen_address` | `PROMETHEUS_LISTEN_ADDRESS` | Metrics listen address | `:2112` |

Engine-specific API keys (Dialogflow service account, Grok, Gemini, Anthropic, etc.) follow the same env-var pattern.

## Prometheus Metrics

Exposed at `PROMETHEUS_LISTEN_ADDRESS/PROMETHEUS_ENDPOINT` (default `:2112/metrics`).

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `aicall_create_total` | Counter | `reference_type` | AIcalls created |
| `aicall_end_total` | Counter | `reference_type` | AIcalls ended |
| `aicall_duration_seconds` | Histogram | `reference_type` | AIcall duration |
| `aicall_tool_execute_total` | Counter | `tool_name` | Tool executions |
| `aicall_backstop_reply_total` | Counter | — | Backstop/fallback replies |
| `aicall_idle_expired_total` | Counter | — | Sessions terminated due to idle timeout |
| `aicall_interrupt_attempted_total` | Counter | — | Barge-in interruption attempts |
| `aicall_stale_response_dropped_total` | Counter | — | Stale LLM responses discarded |
| `message_create_total` | Counter | `role` | Messages created |
| `message_delivery_status_update_failed_total` | Counter | — | Delivery status update failures |
| `summary_start_total` | Counter | — | Summary jobs started |
| `summary_done_total` | Counter | — | Summary jobs completed |
| `receive_request_process_time` | Histogram | `type`, `method` | RPC request latency |
| `subscribe_event_process_time` | Histogram | `publisher`, `type` | Event processing latency |
| `connect` | Gauge | — | Active connections |
| `conversation_reply_send_total` | Counter | — | Conversation replies sent |
| `message_send` | Counter | — | Messages dispatched |

## CLI Tool: ai-control

`cmd/ai-control` — direct DB/cache management (bypasses RabbitMQ). All output is JSON on stdout; logs go to stderr.

```bash
# Uses: DATABASE_DSN, RABBITMQ_ADDRESS, REDIS_ADDRESS

./bin/ai-control ai create --customer_id <uuid> --name <name> --engine_type <type> --engine_model <model> [--parameter '<json>'] [--init_prompt '<text>']
./bin/ai-control ai get    --id <uuid>
./bin/ai-control ai list   --customer_id <uuid> [--limit 100] [--token]
./bin/ai-control ai update --id <uuid> [--name] [--engine_type] [--engine_model] [--parameter] [--init_prompt]
./bin/ai-control ai delete --id <uuid>
```

## Common Commands

```bash
# Build
go build -o ./bin/ai-manager ./cmd/ai-manager/

# Test with coverage
go test -coverprofile cp.out -v $(go list ./...)
go tool cover -html=cp.out -o cp.html

# Regenerate mocks
go generate ./pkg/aihandler/...
go generate ./pkg/aicallhandler/...

# Full verification (run before every commit)
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

## Alerting Guidance

Key signals to alert on:
- `aicall_idle_expired_total` — high rate indicates sessions not being explicitly terminated
- `aicall_stale_response_dropped_total` — high rate may indicate LLM latency spikes
- `aicall_interrupt_attempted_total` vs `aicall_duration_seconds` — barge-in health
- `subscribe_event_process_time` p99 — event processing backlog
