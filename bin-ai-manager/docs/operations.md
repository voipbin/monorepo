# bin-ai-manager Operations

## Deployment

bin-ai-manager deploys via Komodo (VOIP-1348 Tier 2 rollout, following the
VOIP-1342/bin-call-manager pilot and VOIP-1347/Tier 1 pattern) instead of
the older SSH + `versions.lock` (`ssh-deploy.sh`) path.

- **Stack definition:** `bin-ai-manager/komodo/docker-compose.yml` (git is
  the source of truth for structure; Komodo only executes it on
  request).
- **CI path:** `.circleci/scripts/render-image-tag.sh` substitutes
  the built image tag, then `.circleci/scripts/komodo-api-deploy.sh`
  pushes the file's content to Komodo and triggers a deploy, gated
  by the `bin-ai-manager-deploy` job's poll/running checks.
- **Full design and cutover procedure:**
  [docs/plans/2026-08-18-bin-manager-komodo-rollout-tier2-design.md](../../docs/plans/2026-08-18-bin-manager-komodo-rollout-tier2-design.md)
  (in the monorepo root, not this service's own `docs/`).

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
| `aicall_insight_session_idle_minutes` | `AICALL_INSIGHT_SESSION_IDLE_MINUTES` | Idle minutes after which reopening an Insight Case panel starts a NEW assistant session: the customer prompt is refreshed from the AI's current prompt and only rows from the new session are replayed to the model. `0` or less disables the refresh. Default `30` | no |
| `aicall_listen_evaluate_interval_seconds` | `AICALL_LISTEN_EVALUATE_INTERVAL_SECONDS` | Debounce window: one listen evaluation turn per AIcall per this many seconds, regardless of how much was said. This is what decouples LLM cost from speech volume. Default `20` | no |
| `aicall_listen_window_size` | `AICALL_LISTEN_WINDOW_SIZE` | Rolling transcript lines kept for continuity across turns. Default `40` | no |
| `aicall_listen_qa_context_size` | `AICALL_LISTEN_QA_CONTEXT_SIZE` | Q&A message rows replayed into a listen turn's context. Default `10` | no |
| `aicall_listen_max_turns_per_aicall` | `AICALL_LISTEN_MAX_TURNS_PER_AICALL` | Hard per-AIcall turn cap; reaching it stops listening cleanly. The backstop against a pathologically long call. Default `60` | no |
| `aicall_listen_buffer_ttl_hours` | `AICALL_LISTEN_BUFFER_TTL_HOURS` | TTL on the pending/window/debounce-lock/turn-count Redis keys. Default `6` | no |
| `aicall_listen_turn_pipecatcall_id_ttl_seconds` | `AICALL_LISTEN_TURN_PIPECATCALL_ID_TTL_SECONDS` | TTL on registered listen-turn pipecatcall id set entries; only needs to outlive one turn. Default `180` | no |
| `aicall_listen_default_language` | `AICALL_LISTEN_DEFAULT_LANGUAGE` | STT language used when the AIcall carries no `stt_language`. Default `en-US` | no |
| `aicall_listen_confbridge_ready_poll_interval_seconds` | `AICALL_LISTEN_CONFBRIDGE_READY_POLL_INTERVAL_SECONDS` | Poll interval for the bounded confbridge-readiness retry. Default `2` | no |
| `aicall_listen_confbridge_ready_max_wait_seconds` | `AICALL_LISTEN_CONFBRIDGE_READY_MAX_WAIT_SECONDS` | Total wait budget before giving up with `skipped_confbridge_not_ready`. Default `30` | no |
| `aicall_listen_ensure_goroutine_timeout_seconds` | `AICALL_LISTEN_ENSURE_GOROUTINE_TIMEOUT_SECONDS` | `runListenStart`'s own detached-goroutine timeout. Default `45` | no |
| `aicall_listen_start_lock_ttl_seconds` | `AICALL_LISTEN_START_LOCK_TTL_SECONDS` | TTL on `ai:listen:startlock:<aicall_id>`, the per-AIcall lock serializing concurrent create-or-reuse sequences. Default `60` | no |
| `aicall_listen_start_lock_release_timeout_seconds` | `AICALL_LISTEN_START_LOCK_RELEASE_TIMEOUT_SECONDS` | Bound on the **detached** context the lock's release runs under, so a stuck Redis call during cleanup cannot hang the releasing goroutine. Independent of, and far below, the TTL above. Default `3` | no |
| `aicall_listen_conversation_max_message_chars` | `AICALL_LISTEN_CONVERSATION_MAX_MESSAGE_CHARS` | Per-field cap (subject, text and the joined media tokens are each capped, so one message contributes at most about three times this many characters) before a conversation line is buffered (suffix ` [truncated]`). Default `2000` | no |
| `aicall_listen_conversation_flush_jitter_ms` | `AICALL_LISTEN_CONVERSATION_FLUSH_JITTER_MS` | Upper bound of the random jitter added to the deferred flush delay (`aicall_listen_evaluate_interval_seconds` + jitter). Default `1000` | no |
| `mcp_tool_call_timeout_seconds` | `MCP_TOOL_CALL_TIMEOUT_SECONDS` | Bounds one whole MCP `tools/list` or `tools/call` (handshake, method, one re-initialisation). Also bounds the whole MCP tool dispatch path. Session-start discovery across all of an AI's servers is separately capped at 2s aggregate (`mcpSessionStartDiscoveryBudget`, a package constant). Default `10` | no |
| `mcp_tool_exposure_enabled` | `MCP_TOOL_EXPOSURE_ENABLED` | Global rollback switch for advertising MCP tools to the LLM. `false` makes the `GET /v1/aicalls/<uuid>/tools/mcp` RPC return an empty list for every AIcall, exactly as if no AI had MCP tools; built-in tools are unaffected. Default `true` | no |
| `ai_builder_model` | `AI_BUILDER_MODEL` | Model for the Builder, served through the analysis engine's base URL and key (so the key is `GOOGLE_API_KEY` when `analysis_engine_base_url` is Gemini, `ENGINE_KEY_CHATGPT` otherwise). Default `gemini-3.8-flash`. Not measured. At startup, and only when the key exists, a warning is logged when this name's prefix (`gemini` or not) does not agree with `analysis_engine_base_url` (a base URL containing `generativelanguage` is taken as Gemini); a Gemini-compatible proxy on another URL triggers it too, and it is only a warning | no |
| `ai_builder_reasoning_effort` | `AI_BUILDER_REASONING_EFFORT` | `reasoning_effort` sent to the provider; `none` turns Gemini thinking off, empty omits the field. Default `none`. Not measured | no |
| `ai_builder_max_output_tokens` | `AI_BUILDER_MAX_OUTPUT_TOKENS` | Output token cap of one Builder reply. Default `4096`. Not measured | no |
| `ai_builder_daily_limit` | `AI_BUILDER_DAILY_LIMIT` | Builder turns one customer may use in a 24 hour window that starts at the first turn (a fixed window, not a calendar day). The count is taken when the call starts running, so a call that then fails (a provider error, a timeout, or an unusable answer) still counts; a call refused earlier (invalid request, no key, counter down, busy) does not. The counter uses the shared Redis client with its default timeouts, so a Redis that is slow rather than down can hold a turn for about 12 to 20 seconds before it is refused as unavailable, and that turn holds one of the concurrent slots meanwhile. Default `200` | no |
| `ai_builder_max_concurrent` | `AI_BUILDER_MAX_CONCURRENT` | Builder calls running at once **per process**; with two replicas the platform-wide cap is twice this. A full process refuses at once with `BUILDER_BUSY`, and a refused call is not counted. Default `3`. Not measured | no |
| `ai_builder_llm_timeout_seconds` | `AI_BUILDER_LLM_TIMEOUT_SECONDS` | Deadline of one Builder model call. Must be above 0 and **at most 50**: the RPC wait api-manager uses is 55 seconds and 5 are kept for queueing and parsing; startup fails otherwise (checked on every start). Default `40`. Not measured | no |

**Two ordering invariants hold across the listen timing flags, and both are pinned as standing test assertions (`Test_ListenConfigDefaults`), not one-time default checks:**

```
aicall_listen_confbridge_ready_max_wait_seconds
    <  aicall_listen_ensure_goroutine_timeout_seconds
    <  aicall_listen_start_lock_ttl_seconds
```

1. The goroutine encloses the confbridge retry loop and needs headroom for the RPC calls each poll makes.
2. No call inside the start lock can outlive the `ctx` it runs under, so a TTL above the outer goroutine timeout can never expire out from under a goroutine that is still legitimately working. The TTL is **not** derived by summing the RPC timeouts inside the lock — that derivation was tried and withdrawn; do not reintroduce it if these values are ever retuned.

**Raising the max-wait value therefore cascades:** raise the other two to preserve the ordering.
| `prometheus_endpoint` | `PROMETHEUS_ENDPOINT` | Metrics path | `/metrics` |
| `prometheus_listen_address` | `PROMETHEUS_LISTEN_ADDRESS` | Metrics listen address | `:2112` |

Engine-specific API keys (Dialogflow service account, Grok, Anthropic, etc.) follow the same env-var pattern.

**Note:** Gemini audit evaluation uses `GOOGLE_API_KEY` (a `AIza...` Google API key), not `ENGINE_KEY_CHATGPT`. The audit model is `gemini-2.5-flash`.

## Assistant Builder: before you rely on it

There is no on/off setting, and deploying with the key IS the release: nothing else gates it. The Builder is available as soon as ai-manager has the key of the analysis engine (`GOOGLE_API_KEY` for a Gemini base URL, `ENGINE_KEY_CHATGPT` otherwise); without it the status route reports `available=false` and chat answers `BUILDER_UNAVAILABLE`. Every call costs the platform money (the per-customer cap of 200 calls a day and the 3 concurrent calls per process are the only limits; there is no platform-wide cap). It is not released and **no evaluation run has been judged by a human** (see `pkg/builderhandler/eval/README.md`, "Runs side by side"). Settle these first:
- the notice and terms for sending a customer's text to an external model (design section 11), and whether the hosted service may use it at all;
- a platform-wide daily cap (only a per-customer cap exists), and a per-minute limit (none exists; the per-customer limit of about 16.7 requests per second with a burst of 33, and the per-IP limit of 200 requests per second, apply to every route);
- load balancer and ingress timeouts of at least 65 seconds, and that the log pipeline collects the ai-manager and api-manager logs (neither was confirmed from this repository);
- how a Builder call holding an RPC worker for up to the LLM deadline affects aicall and tool RPC latency, and the circuit breaker;
- latency, `max_tokens` and the concurrency value, which are initial values that were never measured.
With no key for the analysis engine's provider, `GET /v1/ai_builder/status` quietly answers `available=false` and chat answers `BUILDER_UNAVAILABLE`; the only trace is one info line at startup. The Builder is an optional feature, so a deploy that does not use it can ignore that line. Removing the key and restarting ai-manager is also the only way to switch the Builder off. The key is read once at startup, so until a process restarts its chat keeps running and costing money, and in a rolling restart the replicas that have not restarted yet keep serving. A restarted process answers chat with `BUILDER_UNAVAILABLE` at once; the status route (and the card in the admin UI) follows up to 30 seconds later because api-manager caches it, while api-manager's chat does not consult that cache. Set or remove the key on every replica in the same deploy: while a rolling restart is half done, replicas with and without the key share one queue, so some chat calls answer `BUILDER_UNAVAILABLE` and the status route can flip between `available=true` and `available=false`. A deploy without the key also logs an unrelated error line, `GOOGLE_API_KEY is not configured; all Gemini audit requests will fail with evaluator_unavailable`, which belongs to the audit feature, not to the Builder. A Builder call that is running when ai-manager is stopped (a restart or a rolling deploy) gets no answer and the customer sees `BUILDER_TIMEOUT` after up to 55 seconds; the call still counts against the daily limit.

## Prometheus Metrics

Exposed at `PROMETHEUS_LISTEN_ADDRESS/PROMETHEUS_ENDPOINT` (default `:2112/metrics`).

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `aicall_create_total` | Counter | `reference_type` | AIcalls created |
| `aicall_end_total` | Counter | `reference_type` | AIcalls ended |
| `aicall_duration_seconds` | Histogram | `reference_type` | AIcall duration |
| `aicall_tool_execute_total` | Counter | `tool_name` | Tool executions. For a built-in tool the label is its name; every MCP tool (`mcp_` prefix) is labeled the constant `mcp`, since the remote tool name is customer-controlled and unbounded |
| `mcp_tool_advertised_total` | Counter | — | `ResolveMcpTools` resolutions that returned at least one MCP tool to pipecat |
| `mcp_tool_call_outcome_total` | Counter | `outcome` | MCP tool dispatch outcomes: `success`, `error` (the remote server returned `isError: true`), `failed` (transport failure or a fail-closed gate refused the call) |
| `aicall_backstop_reply_total` | Counter | — | Backstop/fallback replies |
| `aicall_idle_expired_total` | Counter | — | Sessions terminated due to idle timeout |
| `aicall_insight_session_refresh_total` | Counter | `result` | Insight Case panel reopens evaluated for a session refresh, by outcome: `kept` (the denominator: still live, disabled, not an Insight AI, or not an AI assistance), `refreshed` (a new session started), `failed` (the previous session was kept because the prompt could not be resolved or the write failed) |
| `aicall_interrupt_attempted_total` | Counter | — | Barge-in interruption attempts |
| `aicall_stale_response_dropped_total` | Counter | — | Stale LLM responses discarded |
| `aicall_listen_start_total` | Counter | `kind`, `result` | Listen-start attempts by kind and outcome. `kind` values: `call`, `conversation`, `unknown` (gates that run before the Case's reference type is known). `result` values: `started`, `reused`, `skipped_not_listenable`, `skipped_confbridge_not_ready`, `skipped_confbridge_error`, `skipped_start_locked`, `failed` |
| `aicall_listen_segment_total` | Counter | `result` | Transcript segments seen by listen intake. `dropped_unknown` dominates **by design** — this handler sees every final STT result platform-wide |
| `aicall_listen_turn_total` | Counter | `kind`, `result` | Listen evaluation turns by kind and outcome. `kind` values: `call`, `conversation`, `unknown`. `result` values: `ran`, `skipped_locked`, `skipped_empty`, `skipped_cap`, `skipped_case_closed` (conversation kind's stop signal), `skipped_invalid`, `skipped_register_failed`, `initial_register_failed` (the initial contact_case turn could not be registered as a listen turn, so `notify_agent` is rejected on that first turn only; fired with `kind=unknown` since listen pointers are not yet set), `failed` |
| `aicall_listen_conversation_segment_total` | Counter | `result` | Conversation messages seen by listen intake, by outcome: `buffered`, `dropped_deleted`, `dropped_empty`, `dropped_unknown` (no listener resolved, or the resolver errored), `dropped_stale` (a resolved AIcall is already over, or its pointer names another conversation), `dropped_tenant_mismatch`, `failed`. `dropped_unknown` dominates **by design** — this handler sees every conversation message platform-wide; `dropped_tenant_mismatch` must stay at zero |
| `aicall_listen_conversation_flush_total` | Counter | `result` | Deferred flush timers for conversation listening, by outcome: `ran` (won the lock and invoked a turn; read against `aicall_listen_turn_total` `skipped_empty`), `skipped_locked`, `skipped_scheduled` (a timer was already armed for this AIcall on this replica) |
| `aicall_listen_notify_total` | Counter | `kind` | Proactive notifications actually delivered to an agent's Insight panel, by listen kind |
| `aicall_listen_stop_failed_total` | Counter | — | Listen transcribe-stop RPCs that failed and fell back to the call-hangup backstop |
| `aicall_listen_membership_check_failed_total` | Counter | — | Listen-turn membership checks that errored and degraded to treating the tool call as a real Q&A turn |
| `aicall_foreign_pipecatcall_dropped_total` | Counter | `handler` | Pipecat message events dropped because they came from a pipecatcall the AIcall no longer considers its conversational turn. Defined in `pkg/messagehandler/metrics_foreign.go`, not with the six above |
| `message_create_total` | Counter | `role` | Messages created |
| `message_delivery_status_update_failed_total` | Counter | — | Delivery status update failures |
| `summary_start_total` | Counter | — | Summary jobs started |
| `summary_done_total` | Counter | — | Summary jobs completed |
| `ai_manager_builder_chat_total` | Counter | `result` | Builder turns by result: `ok`, `invalid_argument`, `unavailable` (no key, or the counter is down), `busy`, `daily_limit`, `invalid_response` (truncated or unparseable answer), `llm_error`, `internal` (a panic that the listen handler recovered; the daily count may already have been taken). The label is a fixed set; no conversation text or customer id is ever a label. `llm_error` includes a model call that exceeded `ai_builder_llm_timeout_seconds`, so it does not match api-manager's `api_manager_builder_timeout_total`, which is measured against the 55 second RPC wait |
| `ai_manager_builder_chat_duration_seconds` | Histogram | - | Builder turn latency, from entering `Chat` to returning |
| `ai_manager_builder_tokens_total` | Counter | `kind` | Model tokens the platform paid for, by `kind` (`prompt`, `completion`). Recorded even when the answer was unusable |
| `ai_manager_flow_builder_chat_total` | Counter | `result` | Flow Builder turns by result, with the same fixed labels as `ai_manager_builder_chat_total` (`internal` is a recovered panic on the flow route). Separate series: the Assistant Builder's are not touched by a Flow turn |
| `ai_manager_flow_builder_chat_duration_seconds` | Histogram | - | Flow Builder turn latency |
| `ai_manager_flow_builder_tokens_total` | Counter | `kind` | Model tokens the Flow Builder used, by `kind` (`prompt`, `completion`) |
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
- `aicall_insight_session_refresh_total{result}`: Insight Case panel reopens evaluated for a session refresh. `kept` is the denominator; a rising `failed` rate means panels are opening with a stale session (the prompt could not be resolved, or the rows/metadata could not be written), which never breaks the panel but does mean the boundary is not advancing

MCP tool input schemas are rewritten into a provider-neutral subset before they are advertised (package `pkg/mcpschema`, design `docs/plans/2026-09-29-mcp-tool-exposure-pr-b2-design.md` §15). There is no metric for it; watch these log lines from `resolveMcpOnly`:
- WARN `Dropped an mcp tool whose input schema cannot be made provider-safe. mcp_server_id: ..., tool_name: ..., reason: ..., path: ...`: the tool is not advertised and is not in the session's tool map. Repeated lines for one `mcp_server_id` mean that customer's MCP server exposes schemas VoIPBin cannot advertise (for example a required parameter with no `type`, an array without `items`, or a schema over the 64 KiB output cap or the per-tool resolution work cap, `reason: too_large`). The rest of the session's tools, built-ins included, are unaffected.
- WARN `Removed optional parameters an mcp tool's input schema cannot express provider-safely. ... dropped_properties: N, first: ...`: the tool is still advertised without those optional parameters.
- The existing WARN `Skipped mcp tools whose input schema could not be used. ... over_shared_budget: N` also counts tools skipped because the normalized schemas of one resolution exceeded their 256 KiB output budget.
- DEBUG `Normalized mcp tool input schemas. advertised: ..., dropped_keys: ..., rewrites: ...`: routine key stripping (`x-*`, `title`, `$schema`, ...) and conversions (`oneOf`, `$ref`, list `type`), totalled per resolution.

### Insight AI session history

`aicall_insight_session_idle_minutes` / `AICALL_INSIGHT_SESSION_IDLE_MINUTES`, default `30`. One AIcall lives per Case, so an agent reopening a Case days later lands in the same assistant thread. When the thread has been idle for longer than this window (measured from the newest agent question, the current session boundary, or the AIcall's creation, never from `tm_update`), reopening the panel starts a NEW session: the system rows are rewritten from the AI's current prompt, `insight_session_start` is stamped on the AIcall's metadata, and every later history rebuild replays only rows at or after that boundary. Set it to `0` or less to disable the refresh entirely and keep the previous "replay everything" behaviour.

A refresh never fails a panel open: on any failure the AIcall is returned unchanged and the previous session stays intact (`result="failed"` on the metric above).

### Insight AI live call listening

Listening is always on: it starts when an Insight panel opens on a call or conversation Case and stops with the Case, the call, or the AIcall.

`aicall_listen_conversation_max_message_chars` / `AICALL_LISTEN_CONVERSATION_MAX_MESSAGE_CHARS`, default `2000`. Per-field cap (subject, text and the joined media tokens are each capped, so one message contributes at most about three times this many characters) before a conversation line is buffered (suffix ` [truncated]`).
`aicall_listen_conversation_flush_jitter_ms` / `AICALL_LISTEN_CONVERSATION_FLUSH_JITTER_MS`, default `1000`. Upper bound of the random jitter added to the deferred flush delay (`aicall_listen_evaluate_interval_seconds` + jitter).

**How listening starts.** Explicitly, by `POST /service_agents/aicalls/<aicall-id>/listen`
(routed internally to ai-manager's `POST /v1/aicalls/<aicall-id>/listen`). It is
**not** a side effect of creating or reusing the Q&A AIcall — creating an AIcall
never starts listening. The panels make the two calls in sequence when the Case
panel opens, and the second is fire-and-forget: its response carries no
listening-status field, so "did listening actually start?" is answered by the
metrics below, not by the API.

**What shipped alongside listening.** Two changes shipped with this feature are
general fixes, always active: the two-fetch LLM context assembly (which
guarantees an AIcall's system prompt is never evicted), and the
foreign-pipecatcall guard on `contact_case` bot-LLM messages (which also drops
genuinely stale replies that used to be persisted silently). Expect
`aicall_foreign_pipecatcall_dropped_total` to become non-zero and Insight
answer *shape* to change slightly the moment the code deploys.

**What to watch:**

| Signal | Reading |
|---|---|
| `aicall_listen_turn_total{result="skipped_locked"}` vs `{result="ran"}` | How much LLM spend the debounce is saving. Near-zero `skipped_locked` means the interval is too short for the traffic |
| `aicall_listen_turn_total{result="skipped_cap"}` | Calls hitting the hard turn cap. A rising rate means the cap or the interval needs revisiting |
| `aicall_listen_notify_total` | Proactive notes actually delivered. Zero with non-zero `ran` means prompts are not triggering — a prompt problem, not a system one |
| `aicall_listen_membership_check_failed_total` | Should be ~0. Sustained non-zero means Redis is unhealthy, not that anything listen-specific is wrong |
| `aicall_listen_stop_failed_total` | Stop RPCs that missed their pod. Tolerated — the audio transport ends with the call regardless — but a high rate suggests transcribe-manager instability |
| `aicall_listen_start_total{result="skipped_confbridge_not_ready"}` | The confbridge never settled to a live 2-party bridge within the wait budget. Note this **cannot** distinguish a slow ring from a genuinely non-2-party topology, and repeated panel re-opens on one still-ringing call inflate it. A sustained rate means `aicall_listen_confbridge_ready_max_wait_seconds` is likely too short for real ring times |
| `aicall_listen_start_total{result="skipped_start_locked"}` | A second concurrent start attempt for the *same* AIcall found the lock held and stood down. Expected in small numbers (an agent re-opening a panel during a long ring); a sustained high rate means heavy concurrent re-open pressure, not a fault |

**Redis dependency.** Listening degrades to today's reactive-only behaviour if
Redis is unavailable; Insight Q&A keeps working. A Redis flush silently stops
listening for in-flight calls until the panel is reopened, which repopulates the
state. This is deliberate: there is no DB fallback on a resolver miss, because
that would put a query on a platform-wide hot path.

**The `ai:listen:startlock:<aicall_id>` key.** Held only for the duration of one
listen-start sequence, released by the goroutine that took it via a
token-checked compare-and-delete. A goroutine that genuinely crashes (pod loss)
leaves it to expire on its own `aicall_listen_start_lock_ttl_seconds` — for that
one AIcall, further start attempts stand down as `skipped_start_locked` until
then, and the next panel open after expiry works normally. Do not delete this key
by hand to "unstick" a call: if a live goroutine still holds it, doing so
reintroduces exactly the double-writer race the lock exists to prevent.

**Residual: a terminate racing the transcribe-start RPC.** The listen-start
sequence re-reads the AIcall under the start lock, immediately before its
speculative state write, and stands down as `skipped_not_listenable` if the
AIcall has been terminated or deleted while the confbridge-readiness wait was
running. Teardown deliberately does not take that lock, so one narrow window
remains: a terminate landing between that re-read and
`TranscribeV1TranscribeStart` can clear the resolver membership and
`listen_call_id` a moment before the transcribe is created, leaving a live STT
session with no listener registered. It is sub-RPC in width and self-limiting
(the session's audio transport ends when the call itself ends), and it is
accepted rather than closed by widening the lock over teardown. It surfaces, if
at all, as a `started` outcome on an AIcall that never receives a segment.

**Startup validation of the listen timing and sizing flags.** The process
refuses to start if any listen timing or sizing value is non-positive, or if
`aicall_listen_confbridge_ready_max_wait_seconds` <
`aicall_listen_ensure_goroutine_timeout_seconds` <
`aicall_listen_start_lock_ttl_seconds` does not hold. The error names the
offending values. It is not clamped: these are deploy-time typos, and a refused
start is easier to diagnose than a process quietly disagreeing with its own
configuration. The check runs at startup on both entrypoints
(`cmd/ai-manager` and `cmd/ai-control`), so a broken value cannot lie dormant
and the invariant is the config package's, not one binary's.

The sizing flags are validated for concrete reasons, not for symmetry:
`aicall_listen_window_size` of `0` makes the `LTRIM` inside the window-push Lua
script a no-op, so the rolling window grows unbounded on the transcript-intake
hot path for the whole buffer TTL (a negative value trims from the wrong end,
keeping the oldest lines), and `aicall_listen_max_turns_per_aicall` of `0` makes
the very first turn exceed the cap, silently disabling listening turns.

## Flow Builder (VOIP-1573)

`POST /v1/flow_builder/chat` is the conversational Flow Builder. It reuses the Assistant Builder's configuration (`ai_builder_*`: key, model, reasoning effort, LLM timeout, daily limit, concurrency) and adds no setting. Differences to know:

- The concurrency cap is **one pool** shared with the Assistant Builder (`ai_builder_max_concurrent`). A slow Flow turn can make an Assistant turn answer `BUILDER_BUSY` and the other way round.
- The daily counter is **separate**: Redis key `ai:flow_builder:chat:count:<customer_id>` (Assistant: `ai:builder:chat:count:<customer_id>`), with the same limit value and the same 24 hour fixed window. A deploy does not reset the Assistant counter.
- The output budget is a code constant, `8192` tokens (an Assistant turn uses `ai_builder_max_output_tokens`, default `4096`); both are not measured. Raise `ai_builder_max_output_tokens` above `8192` and the Flow Builder uses that value.
- The circuit breaker in api-manager is per RPC queue (`bin-manager.ai-manager.request`), so Flow turns share it with every other ai-manager RPC. A Flow turn that times out counts toward the same five failures. Watch `ai_manager_flow_builder_chat_total{result="llm_error"}` against the total before relying on it.
- Logs never carry the conversation, the draft or an option value; a draft option can hold anything the user typed.

### Reading a Flow Builder model call (VOIP-1576)

Every Flow turn that reaches the model writes one Info line, `The flow builder model call finished.`, in addition to the existing `The flow builder turn did not succeed.` line on a failure (the two share the same `customer_id`). A request that was refused before the model (no key, invalid request, no usable type, busy, counter failure, daily limit) writes no such line. Metrics are unchanged.

The line carries counts, sizes, durations, flags, configured constants and fixed classes only, never any conversation text, answer, draft content, prompt, schema or provider error text:

| Field | Meaning |
|---|---|
| `outcome` | `ok`; `timeout` (the handler's own `ai_builder_llm_timeout_seconds` deadline); `llm_timeout` (the caller's context deadline, a different thing); `truncated` (finish reason length); `invalid_response`; `llm_<code>` (`canceled`, `auth`, `rate_limit`, `provider_5xx`, `provider_4xx`, `other`); `error` (defensive) |
| `invalid_kind` | Only for `invalid_response`: `nil_response`, `no_choices` or `unparsable` |
| `elapsed_ms` | The provider call alone |
| `build_ms` | Building the prompt, schema and messages |
| `pre_call_ms` | Request validation, the non-blocking concurrency check and the Redis counter, before the model call |
| `chat_ms` | The whole turn up to this line |
| `finish_reason` | `stop`, `length`, `content_filter`, `tool_calls`, `function_call`, `other`, or `none` when the finish reason is empty (no answer arrived, or the provider sent none) |
| `prompt_tokens`, `completion_tokens` | Usage, 0 when the provider did not answer |
| `response_chars`, `system_chars`, `request_chars`, `schema_bytes` | Sizes (runes, bytes for the schema). `request_chars` includes the system prompt, the history and the current draft block |
| `allowed_types`, `user_turns`, `history_messages`, `current_draft_present` | Shape of the request |
| `has_draft`, `draft_discarded`, `empty_draft` | Only for `ok`: a draft was returned; the draft could not be decoded; a draft was decoded but no node (or no start node) survived the type filter |
| `model`, `reasoning_effort`, `max_tokens`, `llm_timeout_ms`, `json_mode` | The settings in force |

How to read the timeout case: `outcome=timeout` with `completion_tokens=0` means no answer arrived before the deadline, which cannot tell a provider that is not answering from a long generation. A `truncated` line with `completion_tokens` near `max_tokens` points to a runaway completion; `llm_rate_limit` and `llm_provider_5xx` point to the provider. If the timeout case stays ambiguous, raise `AI_BUILDER_LLM_TIMEOUT_SECONDS` (at most 50) as a temporary experiment and read `completion_tokens` of the calls that then finish. The setting is not in `komodo/docker-compose.yml`, so it takes a compose change and a deploy (a non-Swarm Compose deploy has no rolling update, so both replicas are recreated together; see `docs/workflows/manager-replica-scaling.md`).
