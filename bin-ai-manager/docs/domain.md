# bin-ai-manager Domain Model

## Core Concepts

### AI (Configuration)
Per-customer AI agent configuration stored in MySQL. Defines which LLM engine to use, voice settings, available tools, and initial prompts.

Key fields:
- `type` — `normal` (default) or `insight`. An Insight AI uses a dedicated system prompt and is restricted to the Insight tool set.
- `is_insight_active` — boolean; marks the single `type=insight` AI that the Case Insight Assistant panel auto-attaches to. A customer may hold any number of Insight AIs, but at most one may be active — enforced by the `ai_ais.active_insight_key` generated column and its unique index (see `bin-dbscheme-manager` migration `27a91e200854`). Creates always default to `false`; only `POST /v1/ais/<uuid>/activate_insight` (`dbhandler.AIActivateInsight`) ever sets it `true`, and it is cleared unconditionally on delete and on any update whose resolved type is not `insight`. When a customer has no active Insight AI, resolution falls back to the most recently created one.
- `engine_type` — provider identifier (see engine list below)
- `engine_model` — format `<target>.<model>` e.g. `openai.gpt-5`, `grok.grok-3`, `anthropic.claude-haiku-4.5`. Validated against the model catalog on create, and on update only when the value changes (see Engine Model Catalog below)
- `init_prompt` — system prompt injected at session start
- `current_prompt_history_id` — UUID pointing to the `ai_ai_prompt_histories` row that reflects the init_prompt at this moment; `uuid.Nil` when no history has been recorded yet. Updated atomically with every prompt change/clear. Exposed in webhook events.
- `tool_names` — list of LLM tool names enabled for this AI
- `tts_type`, `stt_type` — voice provider selection
- `voice_gender`, `voice_language` — TTS voice selection parameters
- `smart_turn_enabled` — boolean; enables smart turn detection during AI call sessions
- `auto_aicall_audit_enabled` — boolean; when true, any finished AICall involving this AI is automatically audited (triggers `AIAudit` creation on AICall termination)

### AIcall (Session)
Active conversation session linking an AI configuration to a reference resource.

Reference types:
- `call` — telephony call (via call-manager)
- `conversation` — chat thread (via conversation-manager)
- `task` — background processing task
- `contact_case` — CRM Case panel session (the Insight Assistant)

Status lifecycle:
```
initiating → progressing → (pausing ↔ resuming) → terminating → terminated
```

Key fields:
- `confbridge_id` — conference bridge hosting the call audio
- `pipecatcall_id` — pipecat session ID for real-time audio
- `host_id` — IP of the pipecat pod owning the session (for per-pod routing)
- `listen_call_id` — the live call this `contact_case` Insight AIcall is currently listening to, or `uuid.Nil` when it is not listening. A **real, indexed column** rather than a metadata key for exactly one reason: on call hangup, `EventCMCallHangup` must run `WHERE listen_call_id = ?` to find every AIcall watching that call (plural — two Cases can share one call), and JSON metadata is not usefully indexable. Deliberately **not** exposed on the webhook — internal plumbing, same treatment as `Message.PipecatcallID`.
- `metadata` — JSON map written at call-start. Carries `prompt_snapshots` (a `[]PromptSnapshot` capturing the AI/team prompt versions active when the call began), plus, while the Insight Assistant is listening to a live call, two listen keys. Future keys may be added without schema migration.

Listen metadata keys (present only while a `contact_case` AIcall is listening):

| Key | Type | Notes |
|---|---|---|
| `listen_transcribe_id` | string (UUID) | The transcribe session this AIcall reads while listening. Read by the listen-start idempotency check and by every stop path, always with the AIcall row already in hand — hence metadata rather than a column |
| `listen_owns_transcribe` | bool | Whether THIS AIcall started that session, as opposed to reusing one another AIcall already had running on the same call. **Only the owner may stop it**; a non-owner must never touch a session another listener still depends on |
| `listen_conversation_id` | string (UUID) | The conversation this AIcall is listening to (conversation Cases only). Metadata rather than a column: no event-driven sweep queries by it. An AIcall carries at most one of `listen_transcribe_id` / `listen_conversation_id` |

#### PromptSnapshot

Embedded in `AIcall.Metadata["prompt_snapshots"]`. One entry per AI agent at call-start.

| Field | Type | Notes |
|---|---|---|
| `ai_id` | UUID | AI configuration that supplied the prompt |
| `prompt_history_id` | UUID | `current_prompt_history_id` of that AI at call-start; `uuid.Nil` if no history |
| `prompt` | string | Resolved (variable-substituted) init_prompt value |
| `member_id` | UUID | Team member ID; `uuid.Nil` for single-AI calls |

### Message
Individual message within an AIcall conversation. Persisted in MySQL for context replay and summaries.

- `role`: `system` | `user` | `assistant` | `tool`
- `direction`: `inbound` | `outbound`
- `origin`: how the message came to exist, orthogonally to `role`. Three values:

| Value | User-visible? | Notes |
|---|---|---|
| `""` (empty) | — | The default: every ordinary message, one that answers or asks something |
| `proactive` | **Yes** | An AI-initiated notification sent via `notify_agent` while monitoring a live call. Real conversational content: stored as `role=assistant`, replayed into future LLM context so the AI remembers what it told the agent, and rendered with a distinct treatment by the frontends. Reaches tenant webhook payloads |
| `listen_internal` | No — internal bookkeeping | The mechanical tool-call / tool-result rows a listen evaluation turn writes. **Never** replayed into any future LLM context — `getPipecatcallMessages` excludes them at the SQL layer, so they cannot evict the AIcall's own system prompt or the agent's Q&A history from the capped replay window. Still reaches webhook payloads, so it is documented rather than hidden |
- `active_ai_id` — UUID of the AI configuration that was active when the message was created; `uuid.Nil` if the aicall or team lookup fails at creation time, or for non-AICall reference paths
- Supports tool call payloads for function-calling workflows

### Summary
Async LLM-generated summary of an AIcall's message history.

Status: `processing` → `done` | `failed`

### Participant
A join row recording which AI agent participated in which AIcall. Stored in `ai_aicall_participants` (created by PR #934). Composite primary key `(ai_id, aicall_id)` — no separate `id` or `customer_id` column.

Key fields:
- `ai_id` — UUID of the AI configuration that participated
- `aicall_id` — UUID of the AIcall session
- `tm_create` — timestamp of first participation

### Team
AI team configuration grouping multiple AI agents for routing or escalation scenarios.

## Topic Exchange Subscription Addresses

Since VOIP-1405 every event is also published to the global topic exchange `bin-manager.event` with the routing key `ai-manager.<resource>.<subscription-id>.<action>` (see [docs/architecture.md](architecture.md) for the wiring). The third key segment is the *subscription address* — the id by which a consumer addresses the stream, which is not always the payload's own id.

| Event | Data | Subscription address | Topic routing key |
|-------|------|----------------------|-------------------|
| `ai.EventTypeCreated` / `Updated` / `Deleted` | `*ai.AI` | own id | `ai-manager.ai.<ai-id>.created` / `.updated` / `.deleted` |
| `aicall.EventTypeStatus*` (6 statuses) | `*aicall.AIcall` | own id | `ai-manager.aicall.<aicall-id>.status_<state>` |
| `message.EventTypeMessageCreated` | `*message.Message` | `AIcallID` | `ai-manager.aimessage.<aicall-id>.created` |
| `message.EventTypeMessageIntermediate` | `*message.IntermediateWebhookMessage` | `AIcallID` | `ai-manager.aimessage.<aicall-id>.intermediate` |
| `summary.EventTypeCreated` / `Updated` / `Deleted` | `*summary.Summary` | own id | `ai-manager.summary.<summary-id>.created` / `.updated` / `.deleted` |
| `team.EventTypeCreated` / `Updated` / `Deleted` | `*team.Team` | own id | `ai-manager.team.<team-id>.created` / `.updated` / `.deleted` |

`Message` and `IntermediateWebhookMessage` implement `eventtopic.SubscriptionIdentifier` (pointer receiver) returning `AIcallID`. `Message` has a stable persisted id, but that id first reaches a subscriber inside the `aimessage_created` event itself, so it cannot be pre-bound; `IntermediateWebhookMessage` is a non-persisted streaming fragment whose id changes per delta (ordered by `sequence`), so its own id is not an address at all. Both therefore address the parent AIcall, which means one conversation is followed with `ai-manager.aicall.<aicall-id>.#` plus `ai-manager.aimessage.<aicall-id>.#`. Single-message retrieval stays available over RPC.

`AI`, `AIcall`, `Summary`, and `Team` need no override — their own ids already are the subscription addresses.

The routing keys above are pinned by the golden table in `models/ai/routingkey_golden_test.go`, which must be updated in the same change as any event-type or address change.

## Engine Model Catalog

`models/ai/catalog.go` holds `catalog`, the curated allow-list of selectable engine models. Each `ModelEntry` has `ID` (the `EngineModel`), `Label`, `Vendor`, `Route`, `UpstreamSlug`, `Recommended`, `Tags`, `Description` and `CustomPrefix`.

- `Route` is `direct` (the customer supplies the provider key; runner type is the model id itself) `openrouter` (platform-managed, no customer key needed) or `custom_openrouter` (the single `custom.openrouter` entry: the customer types a model ID after the `openrouter.` prefix and must supply their own OpenRouter key). `Route`, `UpstreamSlug` and `CustomPrefix` are internal; only the prefix is exposed, as `model_id_prefix`.
- Direct entries: Gemini (2.5 Flash, 2.5 Pro, 2.0 Flash, Pro latest), OpenAI (GPT-5.2, 5.1, 5, 5 mini, 5 nano), xAI (Grok 3, Grok 3 mini).
- OpenRouter entries (`anthropic.claude-haiku-4.5`, `anthropic.claude-sonnet-4.5`, `meta.llama-3.3-70b-instruct`, `meta.llama-4-maverick`, `deepseek.deepseek-v3.2`, `qwen.qwen3-235b-a22b-2507`, `qwen.qwen3-30b-a3b-instruct-2507`, `mistral.mistral-medium-3.1`, `mistral.mistral-small-3.2-24b-instruct`) map to an upstream slug such as `anthropic/claude-haiku-4.5`.
- `CatalogView()` returns the customer-facing `ModelInfo` list served by `GET /v1/ai_models` (`GET /ai_models` on the public API): `id`, `label`, `vendor`, `recommended`, `tags` (never null), `description`, `platform_managed` (true for `openrouter` routes), `key_mode` and `model_id_prefix` (custom entry only). `key_mode` is derived from the route: `direct` gives `own_or_default`, `openrouter` gives `platform`, `custom_openrouter` gives `own_required`. Tags currently include `low-cost`.

### ResolveEngine

`ResolveEngine(EngineModel)` (`models/ai/resolve.go`) is fail-closed and returns a `Resolved` (`Entry`, `RunnerType`, `BlankKey`, `RequireKey`) plus an `Outcome`:

| Outcome | When | Result |
|---------|------|--------|
| `OutcomeCatalog` | The model id exactly matches a catalog entry | Direct route: `RunnerType` is the model id. OpenRouter route: `RunnerType` is `platform_openrouter.<upstream slug>` and `BlankKey` is true, so the customer `engine_key` is never forwarded |
| `OutcomeDirectPassthrough` | Not in the catalog, but the id is `<prefix>.<non-empty rest>` with prefix `openai`, `gemini` or `grok` | `RunnerType` is the model id, `Entry` is nil, the customer key is used. Keeps existing AIs with older direct model names working |
| `OutcomeCustomOpenRouter` | The id starts with `openrouter.` (case sensitive) and the rest passes `ValidateCustomModelID` (`<author>/<slug>`, exactly one slash, letters, digits, `.`, `_`, `-`, author is not `openrouter`, total length within 255) | `RunnerType` is the model id, `Entry` is nil, `RequireKey` is true and `BlankKey` is false: the customer key is mandatory and the platform key is never used. The platform `platform_openrouter.` prefix is never produced for this outcome |
| `OutcomeRejected` | Anything else (unknown prefix, empty rest, no dot, e.g. `dialogflow.cx`) | Zero `Resolved`; create/update fails with an invalid engine model error and a session start fails |

`IsValidEngineModel` returns true for every outcome except `OutcomeRejected`. Create always validates; update validates only when `engine_model` changed, so an unchanged legacy value keeps saving. `ValidateEngine(model, key, modelChanged)` checks the final state: a changed model that resolves to `OutcomeRejected` returns `ErrInvalidEngineModel`, and any `openrouter.` model with an empty (or whitespace) key returns `ErrEngineKeyRequired`, on create and on every update. `aihandler` maps these to `INVALID_ENGINE_MODEL` and `ENGINE_KEY_REQUIRED` without echoing the customer-typed model ID. bin-pipecat-manager calls `ResolveEngine` at session start (`resolveSessionLLM`), so a rejected model cannot bypass validation.

## LLM Engine Providers

`bin-ai-manager` supports 18+ providers via `engine_type`:

| engine_type | Provider | Integration |
|------------|---------|-------------|
| `openai` | OpenAI | OpenAI Chat Completions API |
| `grok` | xAI Grok | OpenAI-compatible API (base URL override) |
| `gemini` | Google Gemini | OpenAI-compatible endpoint |
| `anthropic` | Anthropic Claude | OpenAI-compatible or native |
| `dialogflow` | Google Dialogflow | Dialogflow CX/ES SDK |
| `azure` | Azure OpenAI | OpenAI-compatible with Azure endpoint |
| `aws` | Amazon Bedrock | AWS SDK |
| `cerebras` | Cerebras | OpenAI-compatible |
| `deepseek` | DeepSeek | OpenAI-compatible |
| ... | (others) | Various |

Engine selection at message dispatch time: `MessageHandler` reads the AIcall's AI config `engine_type` and routes to the appropriate engine handler package.

## LLM Tools (Function Calling)

Tool definitions live in `pkg/toolhandler/definitions.go`. Only tools listed in the AI's `tool_names` field are exposed to the LLM.

| Tool name | Action |
|-----------|--------|
| `connect_call` | Transfer or bridge a call |
| `create_call` | Place a new, independent outbound call (not bridged; current AI session continues) |
| `send_email` | Send an email via email-manager |
| `send_message` | Send SMS via message-manager |
| `stop_media` | Stop current TTS audio playback |
| `stop_service` | Soft-end the AI conversation |
| `stop_flow` | Hard-terminate the entire flow |
| `set_variables` | Write to flow context variables |
| `get_variables` | Read from flow context variables |
| `get_aicall_messages` | Retrieve conversation history |
| `search_knowledge` | Query the AI's knowledge base (RAG) |
| `get_correlation` | Retrieve the correlation graph (related resource ids) for an activeflow |
| `get_resource` | Retrieve a curated summary of a single resource by type+id (call, groupcall, recording, transcribe incl. transcripts, summary, aicall incl. conversation history, conferencecall, queuecall); customer-ownership enforced. For `aicall`, an opt-in `include_config` boolean additionally renders the customer-authored session prompt snapshots in an escaped, capped config block (never the platform base prompt) |
| `get_contact_profile` | Insight-only, read-only: returns the profile (name/company/job title + up to 5 reachable addresses, primary first) of the contact linked to the current Case. No arguments; always scoped to the current Case |
| `get_call_transcript` | Insight-only, read-only: returns the merged, chronological transcript of a call's live in-call transcription (transcribe_start) sessions, given a `call_id`. Access is tenant-only (not scoped to the current Case's contact/peer) |
| `list_queues` | Lists the customer's Queues (id, name, detail); read-only, used to discover which Queue to route to before calling `join_queue` |
| `join_queue` | Places the current call into a Queue for human-agent routing by adding a `queue_join` action to the activeflow, then terminates the AIcall; `ReferenceTypeCall` only, mirrors `connect_call`'s add-action-then-terminate shape |

**MCP tools.** In addition to the built-ins above, a Normal-type AI with whitelisted MCP servers (`mcp_server_ids`) gets those servers' tools, namespaced as `mcp_<8 hex of server id>_<remote tool name>`. At session start pipecat-manager keeps resolving built-ins itself and calls `GET /v1/aicalls/<uuid>/tools/mcp` for the MCP half only, appending the result; if that RPC fails the session continues with built-ins only. Insight AIs and team AIcalls never get MCP tools. The name-to-server mapping is stored in the AIcall metadata key `mcp_tool_map` for dispatch and is never published on the webhook; the webhook carries a counts-only `mcp_tool_status` (`servers`, `tools`) instead. A remote result with `isError: true` is returned to the LLM as a failed tool call. Each tool's input schema is normalized to a provider-neutral subset before advertisement (`pkg/mcpschema`: allowlisted keywords only, `$ref`/`oneOf`/list `type` rewritten, optional parameters that cannot be expressed removed); a tool whose schema cannot be normalized is not advertised and is not in `mcp_tool_map`; call arguments are forwarded to the MCP server unchanged.

Tool execution flow:
1. LLM in Pipecat emits a function call
2. Pipecat sends `POST /v1/aicalls/<uuid>/tool_execute` to AI Manager via RabbitMQ RPC
3. `AIcallHandler.ToolHandle()` dispatches to the appropriate manager service
4. Result returned to Pipecat → LLM context

## Real-Time Audio Architecture

```
User phone → Asterisk (8kHz RTP)
                │
                ▼
          Pipecat Manager (Go WebSocket) ─► Python Pipecat pipeline
                                                │
                                           STT (Deepgram / Whisper)
                                                │
                                           LLM (OpenAI / Grok / Gemini)
                                                │
                                           TTS (Cartesia / ElevenLabs / Google)
                                                │
                                    ◄──── 16kHz audio back to Asterisk
```

`bin-ai-manager` does **not** handle audio directly. It owns session state and tool dispatch; `bin-pipecat-manager` owns the audio pipeline.

### AIPromptHistory

Immutable record of a single `init_prompt` value for an AI at a point in time.

**Table:** `ai_ai_prompt_histories`

| Field       | Type   | Notes                                           |
|-------------|--------|-------------------------------------------------|
| id          | UUID   | PK                                              |
| customer_id | UUID   | Copied from parent AI at insert time            |
| ai_id       | UUID   | FK → ai_ais.id                                  |
| prompt      | string | The init_prompt value at this point in time     |
| tm_create   | time   | Set by dbhandler; immutable after creation      |

No `tm_update` or `tm_delete` — rows are append-only.

**Write path:** `aihandler.Create` and `aihandler.Update` insert rows after the AI DB write succeeds. The history row `id` is pre-generated before the AI write so that `ai_ais.current_prompt_history_id` and the new `ai_ai_prompt_histories` row share the same UUID atomically. Insert is best-effort (failure is logged; AI operation succeeds).

**Prompt cleared:** When `init_prompt` is set to `""`, `current_prompt_history_id` is reset to `uuid.Nil` and no history row is created.

**Empty prompt:** No row is inserted when `init_prompt == ""`.

### AIAudit

On-demand quality audit of a single AI agent's performance in one AIcall. Created by `POST /v1/aiaudits`; the handler spawns a background Gemini evaluation goroutine and returns immediately with status `progressing`.

**Table:** `ai_ai_audits`

| Field | Type | Notes |
|---|---|---|
| id | UUID | PK |
| customer_id | UUID | Copied from the AIcall at creation time |
| aicall_id | UUID | FK → ai_aicalls.id |
| ai_id | UUID | FK → ai_ais.id |
| prompt_history_id | UUID | Snapshot of the AI's prompt history at call-start; `uuid.Nil` if unavailable |
| status | string | `progressing` → `completed` \| `failed` |
| overall_score | *int | 0–100 composite score; `null` while progressing or on failure |
| evaluation | JSON | Per-dimension breakdown from Gemini; `null` while progressing |
| message_ids | JSON | Ordered array of message IDs (newest-first) included in the Gemini transcript; `null` while progressing, on failure, or for historical records |
| language | string | BCP 47 tag (e.g. `en-US`) used for the evaluation prompt |
| error | string | Canonicalized error code on failure (see `aiaudit.Error` constants) |
| tm_create / tm_update / tm_delete | time | Standard audit timestamps |

**Error codes:** `invalid_call_metadata`, `prompt_snapshot_not_found`, `prompt_snapshot_has_no_history_id`, `invalid_evaluator_response`, `evaluator_unavailable`, `cancelled`

**Concurrency limits:** global cap of 100 in-flight evaluations; per-customer cap of 10.

**Stale sweep:** On service startup, any `progressing` audits older than 5 minutes are marked `failed` to recover from crashed goroutines.

### AIPromptProposal

A user-initiated request to improve an AI's `init_prompt` based on a set of completed audits. Created by `POST /v1/aipromptproposals`; the handler spawns a background Gemini 2.5 Pro generation goroutine and returns immediately with status `progressing`.

**Table:** `ai_ai_prompt_proposals`

| Field | Type | Notes |
|---|---|---|
| id | UUID | PK |
| customer_id | UUID | Copied from the AI at creation time |
| ai_id | UUID | FK → ai_ais.id |
| audit_ids | JSON | Ordered array of 1..20 audit UUIDs the proposal was generated from |
| language | string | BCP 47 tag (e.g. `en-US`) used for the generation prompt |
| basis_prompt_history_id | UUID | The AI's `CurrentPromptHistoryID` snapshotted when the proposal was created |
| original_prompt | string | The basis prompt text (snapshot at create time) |
| proposed_prompt | string | The improved prompt generated by Gemini; empty while `progressing` |
| rationale | string | Human-readable explanation of why the proposed prompt is an improvement; empty while `progressing` |
| status | string | `progressing` → `completed` → `accepted` \| `rejected` \| `expired`; or `progressing` → `failed` |
| applied_prompt_history_id | UUID | The new `ai_ai_prompt_histories` row id written on accept; `uuid.Nil` until accepted |
| error | string | Canonicalized error code on failure |
| tm_create / tm_update / tm_delete | time | Standard audit timestamps |

**Lifecycle:**

- `progressing` — Gemini generation is running asynchronously.
- `completed` — generation succeeded; `proposed_prompt` and `rationale` are populated. Awaits user action.
- `accepted` — the user accepted the proposal; `applied_prompt_history_id` is populated and the AI's `init_prompt` has been updated.
- `rejected` — the user dismissed the proposal without applying it.
- `expired` — the AI's `CurrentPromptHistoryID` advanced beyond `basis_prompt_history_id` between create and accept, so accept was refused.
- `failed` — generation failed (see `error`).

**On accept:** transactionally writes a new `AIPromptHistory` row referencing this proposal, updates `AI.InitPrompt` and `AI.CurrentPromptHistoryID`, and sets `applied_prompt_history_id` on the proposal. The new history row is the canonical record that the AI's current prompt was applied from this proposal.

**Drift handling:**

- If `AI.CurrentPromptHistoryID` advanced beyond `basis_prompt_history_id` between create and accept, the accept is rejected with `409 prompt version drifted` and the proposal is marked `expired`.
- If any source audit was deleted between create and accept, accept returns `409 audit set invalidated`. The proposal is left as-is so the user can decide.

**Validation at create time:** all selected audits MUST be for the target AI AND for the AI's current prompt version. Mismatches cause `POST /v1/aipromptproposals` to return `400 audit prompt version mismatch` with the offending audit ids in the error message.

**Rate-limiting:** at most 3 `progressing` proposals per customer (exceeding returns `429`); global semaphore caps 30 concurrent generation goroutines.

## Soft-Delete Pattern

All entities use `tm_delete = "9999-01-01 00:00:00.000000"` for active records. Deleted records receive the actual deletion timestamp, preserving history for audit and message replay.
