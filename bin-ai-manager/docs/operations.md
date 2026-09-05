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
| `aicall_listen_enabled` | `AICALL_LISTEN_ENABLED` | Master kill switch for Insight AI realtime call listening. Default **`false`** — the feature ships dark | no |
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
| `aicall_listen_conversation_enabled` | `AICALL_LISTEN_CONVERSATION_ENABLED` | Variant switch for realtime listening on conversation (message) Cases; evaluated after `aicall_listen_enabled` and only on the conversation branch. Off: the trigger returns `skipped_disabled`; intake returns before the resolver lookup, so no new turns are scheduled; an already-armed deferred flush or the next evaluated turn stops the session and clears its state, otherwise its Redis keys expire at their TTLs. Call listens are unaffected. Default `false` | no |
| `aicall_listen_conversation_max_message_chars` | `AICALL_LISTEN_CONVERSATION_MAX_MESSAGE_CHARS` | Per-field cap (subject, text and the joined media tokens are each capped, so one message contributes at most about three times this many characters) before a conversation line is buffered (suffix ` [truncated]`). Default `2000` | no |
| `aicall_listen_conversation_flush_jitter_ms` | `AICALL_LISTEN_CONVERSATION_FLUSH_JITTER_MS` | Upper bound of the random jitter added to the deferred flush delay (`aicall_listen_evaluate_interval_seconds` + jitter). Default `1000` | no |

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
| `aicall_listen_start_total` | Counter | `kind`, `result` | Listen-start attempts by kind and outcome. `kind` values: `call`, `conversation`, `unknown` (gates that run before the Case's reference type is known). `result` values: `started`, `reused`, `skipped_not_listenable`, `skipped_disabled`, `skipped_confbridge_not_ready`, `skipped_confbridge_error`, `skipped_start_locked`, `failed` |
| `aicall_listen_segment_total` | Counter | `result` | Transcript segments seen by listen intake. `dropped_unknown` dominates **by design** — this handler sees every final STT result platform-wide |
| `aicall_listen_turn_total` | Counter | `kind`, `result` | Listen evaluation turns by kind and outcome. `kind` values: `call`, `conversation`, `unknown`. `result` values: `ran`, `skipped_locked`, `skipped_empty`, `skipped_cap`, `skipped_disabled`, `skipped_case_closed` (conversation kind's stop signal), `skipped_invalid`, `skipped_register_failed`, `failed` |
| `aicall_listen_conversation_segment_total` | Counter | `result` | Conversation messages seen by listen intake, by outcome: `buffered`, `dropped_deleted`, `dropped_empty`, `dropped_unknown` (no listener resolved, or the resolver errored), `dropped_stale` (a resolved AIcall is already over, or its pointer names another conversation), `dropped_tenant_mismatch`, `failed`. `dropped_unknown` dominates **by design** — this handler sees every conversation message platform-wide; `dropped_tenant_mismatch` must stay at zero. Nothing is metered while the listen flags are off — intake returns before the resolver |
| `aicall_listen_conversation_flush_total` | Counter | `result` | Deferred flush timers for conversation listening, by outcome: `ran` (won the lock and invoked a turn; read against `aicall_listen_turn_total` `skipped_empty`), `skipped_locked`, `skipped_scheduled` (a timer was already armed for this AIcall on this replica) |
| `aicall_listen_notify_total` | Counter | `kind` | Proactive notifications actually delivered to an agent's Insight panel, by listen kind |
| `aicall_listen_stop_failed_total` | Counter | — | Listen transcribe-stop RPCs that failed and fell back to the call-hangup backstop |
| `aicall_listen_membership_check_failed_total` | Counter | — | Listen-turn membership checks that errored and degraded to treating the tool call as a real Q&A turn |
| `aicall_foreign_pipecatcall_dropped_total` | Counter | `handler` | Pipecat message events dropped because they came from a pipecatcall the AIcall no longer considers its conversational turn. Defined in `pkg/messagehandler/metrics_foreign.go`, not with the six above |
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

### Insight AI live call listening

Kill switch: `aicall_listen_enabled` / `AICALL_LISTEN_ENABLED`, default `false`.

`aicall_listen_conversation_enabled` / `AICALL_LISTEN_CONVERSATION_ENABLED`, default `false`. Variant switch for realtime listening on conversation (message) Cases; evaluated after `aicall_listen_enabled` and only on the conversation branch. Off: the trigger returns `skipped_disabled`; intake returns before the resolver lookup, so no new turns are scheduled; an already-armed deferred flush or the next evaluated turn stops the session and clears its state, otherwise its Redis keys expire at their TTLs. Call listens are unaffected.
`aicall_listen_conversation_max_message_chars` / `AICALL_LISTEN_CONVERSATION_MAX_MESSAGE_CHARS`, default `2000`. Per-field cap (subject, text and the joined media tokens are each capped, so one message contributes at most about three times this many characters) before a conversation line is buffered (suffix ` [truncated]`).
`aicall_listen_conversation_flush_jitter_ms` / `AICALL_LISTEN_CONVERSATION_FLUSH_JITTER_MS`, default `1000`. Upper bound of the random jitter added to the deferred flush delay (`aicall_listen_evaluate_interval_seconds` + jitter).

**How listening starts.** Explicitly, by `POST /service_agents/aicalls/<aicall-id>/listen`
(routed internally to ai-manager's `POST /v1/aicalls/<aicall-id>/listen`). It is
**not** a side effect of creating or reusing the Q&A AIcall — creating an AIcall
never starts listening. The panels make the two calls in sequence when the Case
panel opens, and the second is fire-and-forget: its response carries no
listening-status field, so "did listening actually start?" is answered by the
metrics below, not by the API.

**Turning it off mid-call.** A rollback takes effect on an in-flight session at
its next *evaluated turn*, and turns are triggered by transcript segments, not by
a timer — so for an active conversation that is typically within one
`aicall_listen_evaluate_interval_seconds` (default 20s), but a call that has gone
quiet may not stop until it ends. Call hangup is the guaranteed backstop, and it
is independent of the flag.

**What the flag does NOT gate.** Two changes shipped with this feature are
general fixes and are active regardless: the two-fetch LLM context assembly
(which guarantees an AIcall's system prompt is never evicted), and the
foreign-pipecatcall guard on `contact_case` bot-LLM messages (which also drops
genuinely stale replies that used to be persisted silently). Expect
`aicall_foreign_pipecatcall_dropped_total` to become non-zero and Insight answer
*shape* to change slightly the moment the code deploys, independent of the flag.

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
configuration. The check runs even with `AICALL_LISTEN_ENABLED=false`, so a
broken value cannot lie dormant until the flag is turned on, and it runs from
both entrypoints (`cmd/ai-manager` and `cmd/ai-control`) so the invariant is the
config package's, not one binary's.

The sizing flags are validated for concrete reasons, not for symmetry:
`aicall_listen_window_size` of `0` makes the `LTRIM` inside the window-push Lua
script a no-op, so the rolling window grows unbounded on the transcript-intake
hot path for the whole buffer TTL (a negative value trims from the wrong end,
keeping the oldest lines), and `aicall_listen_max_turns_per_aicall` of `0` makes
the very first turn exceed the cap, silently disabling listening turns.
