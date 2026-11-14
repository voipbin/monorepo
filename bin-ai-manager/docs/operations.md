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
| `local_llm_models` | `LOCAL_LLM_MODELS` | JSON list of operator-registered local (self-hosted OpenAI-compatible) models: `[{"endpoint","id","label","capabilities","default"}]`. No URL or key. Empty means none. `ai-manager` and `ai-control` both read it (`ai-control` needs it to accept `local.` models). Invalid entries are skipped and logged as an error (counts only); the service still starts. The connection details are a stored setting of bin-pipecat-manager (`local_llm.endpoints`, see bin-pipecat-manager), not a variable of any service. | no |
| `engine_base_key_encryption_keys` | `ENGINE_BASE_KEY_ENCRYPTION_KEYS` | Comma-separated `<version>:<base64 32 bytes>` pairs (same format as `MCP_SECRET_ENCRYPTION_KEYS`, a separate key set) for the keys customers store with `customendpoint.` models. The highest version encrypts, every listed version decrypts. A malformed value stops the start; empty starts with a warning and storing an endpoint key then fails with a server error. Generate with `openssl rand -base64 32`, prefix `1:`. Losing the key set makes stored endpoint keys unusable (fail closed; customers send the key again). Rotation: add the new version to every replica first, in a quiet window; keep retired versions until no row uses them. The Komodo variable `BIN_MANAGER__AI_MANAGER_ENGINE_BASE_KEY_ENCRYPTION_KEYS` must exist before the merge (a missing variable stops the container), and the separate operator install repository must carry the same variable. The container env name `ENGINE_BASE_KEY_ENCRYPTION_KEYS` has no service prefix, while the Komodo variable carries the `AI_MANAGER_` prefix because only ai-manager uses it. | no |
| (no flag) | `AI_CONFIG_ENCRYPTION_KEYS` | Master key list of the stored platform keys: comma-separated `<version>:<base64 32 bytes>`. Environment only, never a flag. Unlike `ENGINE_BASE_KEY_ENCRYPTION_KEYS` and `MCP_SECRET_ENCRYPTION_KEYS`, a malformed or empty value does not stop the start: the service logs the problem (never the value), refuses to store a secret and reports the stored keys as undecryptable | no |
| (no flag) | `MCP_OAUTH_GITHUB_CLIENT_ID`, `MCP_OAUTH_GITHUB_CLIENT_SECRET`, `MCP_OAUTH_LINEAR_CLIENT_ID`, `MCP_OAUTH_LINEAR_CLIENT_SECRET`, `MCP_OAUTH_ASANA_CLIENT_ID`, `MCP_OAUTH_ASANA_CLIENT_SECRET`, `MCP_OAUTH_HUBSPOT_CLIENT_ID`, `MCP_OAUTH_HUBSPOT_CLIENT_SECRET` | OAuth app (client_id and client_secret) of each MCP vendor. Environment only, no flag: they are read with `os.LookupEnv` when a request needs them, and only while the vendor has no stored client_secret. A stored app (database) wins; see "MCP OAuth app credentials" below. Empty keeps the service startable; the connection of that vendor then fails at the vendor. A `client_id` value is limited to 256 bytes | no |
| `aicall_conversation_idle_timeout_hours` | `AICALL_CONVERSATION_IDLE_TIMEOUT_HOURS` | Hours before idle AIcall expires | no |
| `aicall_draft_decision_expiry_seconds` | `AICALL_DRAFT_DECISION_EXPIRY_SECONDS` | Seconds after its creation during which an SMS, email or call draft card can still be approved or rejected. Default `600`; a value of zero or less uses `600`. Evaluated on each decision, there is no timer | no |
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
| `mcp_tool_call_timeout_seconds` | `MCP_TOOL_CALL_TIMEOUT_SECONDS` | Bounds one whole MCP `tools/list` or `tools/call` (handshake, method, one re-initialisation). Also bounds the whole MCP tool dispatch path. The tool_execute RPC from `bin-pipecat-manager` waits 15 s and the Python HTTP call 20 s, so keep this value at 10 or below (the 5 s left over covers the message writes around the call, which this timeout does not bound); raise all three together otherwise (see `bin-pipecat-manager/docs/domain.md`). What the LLM sees from an MCP call is also bounded: a response over 1 MiB fails with `MCP tool result is too large`, and a successful result is cut to at most 32 KiB of text (ending in `[truncated]`) so the stored message fits `ai_messages.content`. Session-start discovery across all of an AI's servers is separately capped at 2s aggregate (`mcpSessionStartDiscoveryBudget`, a package constant). Default `10` | no |
| `mcp_tool_exposure_enabled` | `MCP_TOOL_EXPOSURE_ENABLED` | Global rollback switch for advertising MCP tools to the LLM. `false` makes the `GET /v1/aicalls/<uuid>/tools/mcp` RPC return an empty list for every AIcall, exactly as if no AI had MCP tools; built-in tools are unaffected. Default `true` | no |
| `ai_builder_model` | `AI_BUILDER_MODEL` | Model for the Builder, served through the analysis engine's base URL and key (so the key is the stored Gemini key (`llm.gemini.api_key`) when `analysis_engine_base_url` is Gemini, the stored OpenAI compatible key (`llm.openai_compat.api_key`) otherwise). Default `gemini-3.8-flash`. Not measured. At startup, and only when the key exists, a warning is logged when this name's prefix (`gemini` or not) does not agree with `analysis_engine_base_url` (a base URL containing `generativelanguage` is taken as Gemini); a Gemini-compatible proxy on another URL triggers it too, and it is only a warning | no |
| `ai_builder_reasoning_effort` | `AI_BUILDER_REASONING_EFFORT` | `reasoning_effort` sent to the provider; `none` turns Gemini thinking off, empty omits the field. Default `none`. Not measured | no |
| `ai_builder_max_output_tokens` | `AI_BUILDER_MAX_OUTPUT_TOKENS` | Output token cap of one Builder reply. Default `4096`. Not measured | no |
| `ai_builder_daily_limit` | `AI_BUILDER_DAILY_LIMIT` | Builder turns one customer may use in a 24 hour window that starts at the first turn (a fixed window, not a calendar day). The count is taken when the call starts running, so a call that then fails (a provider error, a timeout, or an unusable answer) still counts; a call refused earlier (invalid request, no key, a key that can not be read at the check, counter down, busy) does not. A key that can not be read inside the model call (after the check passed) is a failure of that call and is counted, with no refund. The counter uses the shared Redis client with its default timeouts, so a Redis that is slow rather than down can hold a turn for about 12 to 20 seconds before it is refused as unavailable, and that turn holds one of the concurrent slots meanwhile. Default `200` | no |
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
| `log_level` | `LOG_LEVEL` | Log level: `debug`, `info`, `warn` or `error` (case-insensitive). Debug lines are written only when this is `debug`. Debug lines carry ids and lengths, never conversation text, prompts or model answers (VOIP-1633). Applies to the Go service and its control CLI only | `info` |

Engine-specific API keys (Dialogflow service account, Grok, Anthropic, etc.) follow the same env-var pattern.

**Note:** Gemini audit evaluation uses the stored Gemini key (`llm.gemini.api_key`; an `AIza...` Google API key), not the OpenAI compatible key. The audit model is `gemini-2.5-flash`.

## Assistant Builder: before you rely on it

There is no on/off setting, and setting the key (`ai-control config set`, or the admin API) IS the release: nothing else gates it. The Builder is available as soon as ai-manager has the key of the analysis engine (the Gemini key for a Gemini base URL, the OpenAI compatible key otherwise); without it the status route reports `available=false` and chat answers `BUILDER_UNAVAILABLE`. Every call costs the platform money (the per-customer cap of 200 calls a day and the 3 concurrent calls per process are the only limits; there is no platform-wide cap). It is not released and **no evaluation run has been judged by a human** (see `pkg/builderhandler/eval/README.md`, "Runs side by side"). Settle these first:
- the notice and terms for sending a customer's text to an external model (design section 11), and whether the hosted service may use it at all;
- a platform-wide daily cap (only a per-customer cap exists), and a per-minute limit (none exists; the per-customer limit of about 16.7 requests per second with a burst of 33, and the per-IP limit of 200 requests per second, apply to every route);
- load balancer and ingress timeouts of at least 65 seconds, and that the log pipeline collects the ai-manager and api-manager logs (neither was confirmed from this repository);
- how a Builder call holding an RPC worker for up to the LLM deadline affects aicall and tool RPC latency, and the circuit breaker;
- latency, `max_tokens` and the concurrency value, which are initial values that were never measured.
With no key for the analysis engine's provider, `GET /v1/ai_builder/status` quietly answers `available=false` and chat answers `BUILDER_UNAVAILABLE`; the only traces are two startup lines (an info line about the Builder, and a warning that the key is not configured; key name and source, never a value). The Builder is an optional feature, so a deploy that does not use it can ignore that line. The key is read for every status and chat call, so a key that was set or cleared with `ai-control config set|clear` is used by the next call with no restart, and clearing the stored key is how the Builder is switched off. The status route (and the card in the admin UI) follows up to 30 seconds later because api-manager caches it, while api-manager's chat does not consult that cache. A key that can not be read (the database is down, or a stored value can not be decrypted) is reported as `available=false` for the status route, and chat answers `BUILDER_UNAVAILABLE` with a warning in the log. The check at the start of a chat call is bounded to 2 seconds, and the key read inside the model call is bounded by the LLM deadline, so the two together stay below the 55 seconds that api-manager waits; the Redis counter latency noted above is not part of that bound. A key that is cleared between the check and the model call sends one request without a key, which the provider refuses (one `auth` error line). A deploy without the Gemini key logs a warning that the key is not configured; it belongs to the audit feature as well as to a Gemini based Builder. A Builder call that is running when ai-manager is stopped (a restart or a rolling deploy) gets no answer and the customer sees `BUILDER_TIMEOUT` after up to 55 seconds; the call still counts against the daily limit.

## Platform LLM keys

The two platform LLM keys are stored settings. They are read from the database only: a key without a stored row is not configured (an empty key), and the environment variables `ENGINE_KEY_CHATGPT` and `GOOGLE_API_KEY` are not read by this service any more. Enter each key with `ai-control config set` or the admin API; `config clear` removes the key, and the key is then not configured.

| Stored key | Used by |
|------------|---------|
| `llm.openai_compat.api_key` | The summary, analysis and Builder calls when their base URL is not the Gemini endpoint |
| `llm.gemini.api_key` | Audit and prompt proposal, and the summary, analysis and Builder calls when their base URL is the Gemini endpoint (a base URL that contains `generativelanguage`) |

These are platform keys. They are not the `engine_key` of a customer AI, which belongs to the customer and is stored encrypted on the AI.

The key is read for each call (nothing is cached), so a change is used by the next call of every replica, with no restart. The price is one database read per call (two for a Builder chat: the check and the model call), so while the database can not be read the calls that use a platform key fail. A request that is already running keeps the key it started with; the summary and analysis calls retry for up to one minute with the key they read at the start, so a rotation can take up to a minute to reach a retrying call. An AI conversation turn (an aicall reply) does not read these stored keys: the pipecat service answers it, with the key of the AI or with its own stored platform key, so changing a stored platform key does not change the key an AI conversation uses.

```bash
# the value is read from stdin only (one line; a key is at most 512 bytes) and is never printed
printf '%s\n' "$NEW_KEY" | ./ai-control config set --key llm.gemini.api_key --actor-id <uuid>
./ai-control config list
./ai-control config clear --key llm.gemini.api_key --actor-id <uuid>
```

The same two keys can also be stored, deleted and read through the admin API of api-manager (`/service_configs/ai`, project super admin only), which reaches this service as `/v1/configs` requests. A change made that way is recorded in the history with the source `api` (the command line records `cli`) and is used by the next call like a change of the command line. The request body of a secret is never logged, and a request that looks like a config request is not routed to any other handler.

The master key list is `AI_CONFIG_ENCRYPTION_KEYS` (environment only). The deployment definition must pass it to the container before `config set` can store a value; without it the commands list and clear but refuse to set, and a stored key is reported as undecryptable. The `config` commands need the database and the master key list only; they do not validate the rest of the configuration.

Rotation:
1. Create the new key at the provider. Keep the old key valid.
2. Make sure every replica runs the image of this feature. Do not enter a value while a rolling deploy is half done, with the command line or the admin API: an older replica ignores the stored value (and does not serve the admin API), so the replicas would use different keys.
3. Run `config set` with the new key. The next call uses it.
4. Check it at once with a call that needs the key (a Builder status and one chat turn, or one audit). If the provider answers an authentication error, set the right value again (`config clear` would leave the key not configured).
5. Revoke the old key at the provider after the check. An image from before the removal of the environment fallback (VOIP-1721) reads `ENGINE_KEY_CHATGPT` and `GOOGLE_API_KEY` from the environment of the deployment again when the key has no stored row; a key that is rotated in the database only is not rotated for such an image. Keep the old key valid, and keep the environment variable of the deployment in step, until the rollback window has ended.

The pipecat service keeps its own stored `google.api_key` (this service does not read it, and neither service reads `GOOGLE_API_KEY`). After a Gemini key rotation, replace the stored key here and the stored `google.api_key` of the pipecat service with `pipecat-control config set`; the stored value here does not reach pipecat.

Recovery: a stored value that can not be decrypted (the master key is missing, mistyped or changed) is not replaced by an empty key. The calls that need that key fail with an unavailable key, `config list` shows `db_undecryptable`, and the startup logs an error line. Restore the master key list, or run `config clear --key <key>` and set the key again. A deployment that has no `AI_CONFIG_ENCRYPTION_KEYS` starts normally; `config set` is refused and a stored key is reported as undecryptable.

## MCP OAuth app credentials

The OAuth app of each MCP vendor (GitHub, Linear, Asana, HubSpot) is a pair of settings, a `client_id` and a `client_secret`, that is read each time a request needs it (connect start, connect completion, and the token refresh). Nothing is cached, so a change is used by the next request of every replica, with no restart.

| Stored key | Kind | Environment name used when nothing is stored |
|------------|------|----------------------------------------------|
| `mcp_oauth.<vendor>.client_secret` | secret (encrypted), the anchor of the pair | `MCP_OAUTH_<VENDOR>_CLIENT_SECRET` |
| `mcp_oauth.<vendor>.client_id` | parameter (plain text, at most 256 bytes) | `MCP_OAUTH_<VENDOR>_CLIENT_ID` |

`<vendor>` is `github`, `linear`, `asana` or `hubspot`.

Resolution order is `database`, then `environment`, and it is decided for the pair as a whole by the `client_secret` row:

- No `client_secret` row: both values come from the environment (as before this feature). A vendor with neither value is "not configured": the connect start still builds the authorize URL with an empty `client_id` and the vendor refuses it, as before.
- A `client_secret` row: both values come from the database only. The environment values are not used, not even for a missing `client_id`. A stored `client_secret` without a stored `client_id` is an error for that vendor (connect start and completion fail with a server error, `MCP_OAUTH_CREDENTIALS_UNAVAILABLE`, and a refresh returns the error), never an environment fallback.
- A database read error or a stored value that can not be decrypted is also an error, never an environment fallback. A failed resolution before the token call spends no refresh token and sets no backoff, so the next request tries again.

Operating rules:

1. **Write order: `client_id` first, `client_secret` last.** The `client_secret` row switches the pair to the database, so it must be the last write; with the `client_id` written first the pair is never half stored. The `client_id` is a plain parameter (`KindParam`): it can only be written with `ai-control config set`, not with the admin API (`/service_configs/ai`), and its value is readable in the history. The `client_secret` can be written with either.
2. **Removal: clear both keys.** Clear the `client_secret` and the `client_id`. If only the `client_secret` is cleared, the pair goes back to the environment values while a stored `client_id` row is left behind and ignored.
3. **Use the same app.** The refresh token of an already connected MCP server is bound to the app that issued it. Store the `client_id` and `client_secret` of the same vendor app that the environment holds now; a different app breaks every connection of that vendor until the customer reconnects.
4. **Do not store the pair while a rolling deploy is half done.** An older replica reads the environment only, so replicas would use different apps.
5. **Check the length of the environment `client_id` first.** The 256 byte limit applies to a `client_id` that comes from the environment too. A longer value is an error for that vendor and does not fall back to anything.
6. The deployment keeps passing the eight environment variables (they stay valid as the fallback until the pair is stored).

```bash
# client_id first (plain text, still read from stdin), then the client_secret
printf '%s\n' "$CLIENT_ID" | ./ai-control config set --key mcp_oauth.github.client_id --actor-id <uuid>
printf '%s\n' "$CLIENT_SECRET" | ./ai-control config set --key mcp_oauth.github.client_secret --actor-id <uuid>
./ai-control config list
```

`ai-control config list` reads the environment of the process that runs the command, not the one of the daemon. The daemon reads the vendor values from its own environment, so a vendor that the daemon serves from its environment can show as `none` in a command that was started without those variables. The service logs the source of every key at start (key name and source, never a value), and a vendor that is not configured is a warning there.

The token refresh runs detached from the request that triggered it. It resolves the pair once, at the start of the exchange (bounded to 2 seconds), and all callers that share the exchange share that result; a rotation of the app during an exchange applies from the next exchange.

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
| `ai_manager_carrier_lookup_total` | Counter | `carrier`, `outcome` | Carrier tracking lookups by outcome: `success`, `cache_hit`, an error code (`credential_not_configured`, `credential_malformed`, `credential_invalid`, `rate_limited`, `carrier_unavailable`, `service_busy`, `invalid_format`, `not_found`, `tool_unknown`) or `internal_error`. A call to a reserved-host server with no adapter behind it is not counted here; it is counted as `error` in `mcp_tool_call_outcome_total`. No customer, server, number or key is ever a label |
| `ai_manager_carrier_lookup_duration_seconds` | Histogram | `carrier` | Duration of lookups that reached the adapter |
| `ai_manager_carrier_inflight` | Gauge | `carrier` | Lookups running now |
| `ai_manager_carrier_token_exchange_total` | Counter | `carrier`, `outcome` | Carrier access token exchanges by outcome |
| `aicall_backstop_reply_total` | Counter | `result` | `pipecatcall_terminated` handling: conversation backstop replies (`sent`, `failed`, `send_failed`, `skipped_seen`, `skipped_terminated`), `skipped_voice` (neither a conversation nor a contact_case), `skipped_not_aicall`, and for contact_case turns that ended without an answer `timeout_notice_created`, `timeout_notice_failed`, `timeout_notice_skipped` (VOIP-1633; contact_case was counted under `skipped_voice` before) |
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
| `ai_manager_builder_chat_total` | Counter | `result` | Builder turns by result: `ok`, `invalid_argument`, `unavailable` (no key, or the counter is down), `key_error` (the key could not be read, at the check or inside the model call), `busy`, `daily_limit`, `invalid_response` (truncated or unparseable answer), `llm_error`, `internal` (a panic that the listen handler recovered; the daily count may already have been taken). The label is a fixed set; no conversation text or customer id is ever a label. `llm_error` includes a model call that exceeded `ai_builder_llm_timeout_seconds`, so it does not match api-manager's `api_manager_builder_timeout_total`, which is measured against the 55 second RPC wait |
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

### Carrier tracking adapters (VOIP-1677)

An MCP server whose url is on the reserved host `carrier.invalid` (for example `https://carrier.invalid/<carrier>`) is a built-in carrier adapter, not a remote server. `mcptoolhandler` answers its `tools/list` and `tools/call` in the same process and never opens a connection to it; the guarded dialer also refuses every `.invalid` host as a second line. The carriers that are open are the lines of `mcpserver.DefaultCarrierTable()` together with the adapters registered in `pkg/carriertrackhandler`; a carrier is open only when both are there, and removing either closes it. For a reserved url that is not in the table, registration is refused (`INVALID_MCP_SERVER_URL`) and an existing record answers a call with a fixed "integration is not available" text.

Every limit is per process, not per platform. None of them is measured yet; they are starting values.

| Limit | Value | Behaviour |
|-------|-------|-----------|
| Concurrent lookups per carrier | 2 | The next call is refused at once with `service_busy`; it does not wait |
| Key cooldown | 10 minutes for a rejected key, 1 minute for `rate_limited` or an adapter's short block | The same (customer, carrier, key) gets the same error without a carrier call; a re-entered key is tried at once. At most 256 entries per carrier |
| Time budget | the tool call timeout (or what is left of the caller's context), minus a 2 second reserve | Under 1 second left: no carrier call, `carrier_unavailable`. A carrier answer that arrives after the budget is discarded and nothing is stored |
| Result cache | 120 seconds, 256 entries, 16 KiB per entry | Keyed by customer, carrier, key and number; the 256 entries are shared by all customers of the process. Only results, `not_found` and `invalid_format` are cached |
| Token cache | `expires_in` minus 60 seconds, at most 1 hour, 128 entries, 4 KiB per token | A token over the size, or without a usable lifetime, is not cached |
| Response body | 1 MiB unless the adapter sets less; 64 KiB for a token response | A longer body fails the lookup instead of being cut |

**ups.** The saved key is one line `CLIENT_ID:CLIENT_SECRET` (the key pair of the UPS application), stored as the server's `bearer` secret. The tool is `track_ups`; the number must be `1Z` and 16 letters or digits. A lookup gets an access token with the client credentials grant (HTTP Basic, cached for the `expires_in` the carrier returns, which is a quoted number) and then reads the package once. The hosts are constants of the adapter and a redirect is refused. A rejected key (`credential_invalid`) comes only from the token exchange; a lookup answer 401 drops the cached token, so a key that keeps failing there costs a token exchange per lookup. Answers 403 and 429 of a lookup block the key for one minute. The result carries the carrier's state only when it is known with confidence: `delivered` needs both the status type and a delivered date, `out_for_delivery` is not produced (read `status_text`), event times are UTC only when the local time, the gmt time and the offset agree, and a delivery window is not forwarded (only the expected date).

Not adopted, with the signal that would change that: a per (customer, carrier) concurrency limit (one customer's calls refusing another customer's with `service_busy`), and blocking all token exchanges of a carrier after a 429 (repeated 429 answers in `ai_manager_carrier_token_exchange_total`).

What reaches the model is either the allow-list result (no recipient, sender, address, signature or payment field exists in it) or one of the fixed texts of `carriertrackhandler.Message`. Carrier answers, keys and tracking numbers are not written to logs or error texts.

#### FedEx

`https://carrier.invalid/fedex` is served by the FedEx adapter (tool `track_fedex`). The saved key is `CLIENT_ID:CLIENT_SECRET` of a FedEx developer project, stored as the server's `bearer` secret.

- The only host the adapter calls is `apis.fedex.com` on port 443: the OAuth token (`/oauth/token`, client credentials) and the lookup (`/track/v1/trackingnumbers`). The host is a constant of the code. The key goes in the token request body; the lookup carries only the access token in a header, and the tracking number is in the request body, never in a url.
- An access token is cached in memory until a minute before it expires (at most an hour), per customer and key. FedEx limits token requests per source address, so repeated `service_busy` outcomes in `ai_manager_carrier_token_exchange_total` mean the token cache is not enough.
- The status values the adapter returns are `label_created`, `in_transit`, `out_for_delivery`, `delivered` and `unknown`. FedEx documents few status codes; any other code gives `unknown`, with the carrier's status text and the scan events, and the original code in `carrier_status_code`. A delivery date from FedEx makes the status `delivered`.
- A shipment that FedEx does not track through this API (Authenticated Delivery) gets an `unknown` result with a fixed sentence and no events. That result is cached like any other.
- Scan events are sorted newest first by the adapter, because the FedEx schema does not define an order; an event whose time has no offset is listed after the others and has `time_local` instead of `time`.
- Answer mapping: `400` is `invalid_format` (a `400` that says the request had no tracking number is `carrier_unavailable`), `401` drops the cached token and is `carrier_unavailable`, `403` is `carrier_unavailable` and blocks the key for a minute, a `200` answer whose result (or, when the result is empty or missing, whose alerts) carries the "no data" code is treated like that `404`, and the secured delivery code gives the secured delivery result in every case; `404` is `not_found` only when the body says there is no data for the number and `carrier_unavailable` otherwise, `429` is `rate_limited` and blocks the key for a minute, every other status is `carrier_unavailable`. A sudden rise of `invalid_format` or `not_found` outcomes for this carrier can mean the request format is wrong rather than customers entering wrong numbers.

#### DHL

`https://carrier.invalid/dhl` is served by the DHL adapter (tool `track_dhl`). The saved key is the DHL API key, one value, stored as the server's `bearer` secret; it is sent in the `DHL-API-Key` request header only, never in a url. There is no token exchange. The adapter calls the one host fixed in the code, with the tracking number as the only query value, and does not follow redirects. A tracking number is 8 to 40 letters and digits (spaces and hyphens are ignored; DHL publishes no stricter rule).

| DHL answer | Result for the model | Cooldown of the key |
|------------|----------------------|---------------------|
| 200 with a shipment | the allow-list result | |
| 200 with no shipment, 404 | `not_found` (cached) | |
| 400 | `invalid_format` (cached) | |
| 401, 403 | `credential_invalid` | 10 minutes |
| 429 | `rate_limited` | the `Retry-After` seconds, between 1 second and 10 minutes; 1 minute when the header is missing, zero or negative, or not a whole number of seconds |
| any other status, a body that is not valid, a body over the limit, a transport error, a redirect | `carrier_unavailable`; only the status number is logged | |

The DHL documentation names 200, 404 and 429 only; reading 400 as `invalid_format` and 401 and 403 as a rejected key is a provisional choice that has not been seen against a real key. When the answer lists several shipments, the one whose id equals the requested number is used, and the first one when none does. A shipment DHL marks as returned is reported as `returned` whatever its status code says; the code itself stays in `carrier_status_code`. A time without an offset is the local time of the event: it is given as `time_local`, `time_zone_known` is false, and `delivered_at` stays empty. DHL requires its notice to be shown wherever the data is presented, so every DHL result carries `attribution` (also appended to the card description) and the tool description asks the model to show it.

## Database Migrations

The service applies the migrations of its thirteen tables itself (`migrations/`, embedded) on every start, after the database connection and before the cache and RabbitMQ; version table `ai_schema_migrations`. A failure exits the process with a non-zero code and the container restarts.

| Symptom | Likely Cause | Resolution |
|---------|-------------|------------|
| `Dirty database version N. Fix and force version.` | A previous run stopped inside migration N (failure, kill) | Stop the crash-looping container. Check whether migration N's statement was applied. Run `ai-control migrate force <N-1>` if not (for N=1 use `force -- -1`), `force <N>` if so, then start again. The image has no shell, so run the command in a one-off container of the same image: `docker run --rm --entrypoint ./ai-control -e DATABASE_DSN=<dsn> <image> migrate force <N>` (quote a DSN that holds special characters) |
| `can't acquire lock` | Another instance is migrating for longer than 10 seconds | The restart policy retries; no action unless it persists |
| `Error 1205 ... Lock wait timeout` | A long transaction holds a metadata lock on a table (even `CREATE TABLE IF NOT EXISTS` takes it briefly) | Find the long transaction, then restart. If the database is marked dirty, follow the dirty row |
| `CREATE command denied ... ai_schema_migrations` | The database account cannot create tables | Grant CREATE to the service account |
| `db version N is ahead of this image (max M); skipping migrations` (warning) | An older image started after a newer one migrated | Expected during a rollback; no action |

`ai-control migrate status` shows the applied version, the image version and the dirty flag (database only, no Redis or RabbitMQ). The new database must be created with `CHARACTER SET utf8 COLLATE utf8_general_ci`. On a new database the service creates its tables at startup (the baseline is `CREATE TABLE IF NOT EXISTS`); no other schema tool runs first. File rules and checks: [docs/workflows/service-owned-migrations.md](../../docs/workflows/service-owned-migrations.md).

## CLI Tool: ai-control

`cmd/ai-control` — direct DB/cache management (bypasses RabbitMQ). All output is JSON on stdout; logs go to stderr.

```bash
# Uses: DATABASE_DSN, RABBITMQ_ADDRESS, REDIS_ADDRESS

./bin/ai-control ai create --customer_id <uuid> --name <name> --engine_type <type> --engine_model <model> [--parameter '<json>'] [--init_prompt '<text>']
./bin/ai-control ai get    --id <uuid>
./bin/ai-control ai list   --customer_id <uuid> [--limit 100] [--token]
./bin/ai-control ai update --id <uuid> [--name] [--engine_type] [--engine_model] [--parameter] [--init_prompt]
./bin/ai-control ai delete --id <uuid>

# Platform LLM keys (database and master key list only; the value is read from stdin)
./bin/ai-control config list
./bin/ai-control config set   --key <key> --actor-id <uuid>
./bin/ai-control config clear --key <key> --actor-id <uuid>

# Schema migrations (database only)
./bin/ai-control migrate status
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
| `allowed_types`, `user_turns`, `history_messages`, `current_draft_present` | Shape of the request. `current_draft_present` is true whenever the request carried a current draft, even an empty one |
| `has_draft`, `draft_discarded`, `empty_draft` | Only for `ok`: a draft was returned; the draft was dropped (it was not an object, had no usable node, or had more nodes than the limit); a draft was decoded but no node (or no start node) survived the type filter |
| `model`, `reasoning_effort`, `max_tokens`, `llm_timeout_ms`, `json_mode` | The settings in force (see the note below the table) |

The Flow Builder always sends `json_object` (VOIP-1577), so `json_mode=object` and `schema_bytes=0` are the expected values. `json_mode=schema` there would mean the override in `FlowConfig` is gone: with `json_schema` the provider returned no draft and often generated until the output cap. Because `json_object` does not force the answer shape, the assistant turns of the history are sent to the model as `{"message": ...}` (VOIP-1578); with plain text turns the model answered in plain text on follow-up question turns and the parser rejected it (`outcome=invalid_response`, `invalid_kind=unparsable`). `request_chars` counts this wrapping (about 14 characters plus escapes per assistant turn).

How to read the timeout case: `outcome=timeout` with `completion_tokens=0` means no answer arrived before the deadline, which cannot tell a provider that is not answering from a long generation. A `truncated` line with `completion_tokens` near `max_tokens` points to a runaway completion; `llm_rate_limit` and `llm_provider_5xx` point to the provider. If the timeout case stays ambiguous, raise `AI_BUILDER_LLM_TIMEOUT_SECONDS` (at most 50) as a temporary experiment and read `completion_tokens` of the calls that then finish. The setting is not in `bin-ai-manager/komodo/docker-compose.yml`, so it takes a compose change and a deploy (a non-Swarm Compose deploy has no rolling update, so both replicas are recreated together; see the repository's `docs/workflows/manager-replica-scaling.md`).

Enable order (operator): store the endpoints in the pipecat service (`local_llm.endpoints`), create or update the Komodo variable `BIN_MANAGER__LOCAL_LLM_MODELS`, redeploy the pipecat stack (pipecat-manager and the runners together, same image), then redeploy ai-manager last, so a model is never listed before every pipecat container can run it. Do not merge or redeploy ai-manager between the endpoint and variable change and the pipecat redeploy. Removal is the reverse: move the AIs that use the model first (find them with a read-only query on `ai_ais.engine_model`; `ai-control ai list` filters by customer only), redeploy ai-manager first, the pipecat stack afterwards. After each deploy check the startup log line `Local llm models loaded.` (counts only; an error line means entries were skipped or the value was malformed). Compose value format: compact JSON, `$` written as `$$`. Create the Komodo variable `BIN_MANAGER__LOCAL_LLM_MODELS=[]` before the merge that introduces it: a missing variable does not stop the containers (Komodo passes the unreplaced `[[...]]` text, the service starts, logs an error instead of `Local llm models loaded.`, and offers no local model).
