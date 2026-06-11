# bin-ai-manager Domain Model

## Core Concepts

### AI (Configuration)
Per-customer AI agent configuration stored in MySQL. Defines which LLM engine to use, voice settings, available tools, and initial prompts.

Key fields:
- `engine_type` — provider identifier (see engine list below)
- `engine_model` — format `<target>.<model>` e.g. `openai.gpt-4o`, `grok.grok-3`, `dialogflow.cx`
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

Status lifecycle:
```
initiating → progressing → (pausing ↔ resuming) → terminating → terminated
```

Key fields:
- `confbridge_id` — conference bridge hosting the call audio
- `pipecatcall_id` — pipecat session ID for real-time audio
- `host_id` — IP of the pipecat pod owning the session (for per-pod routing)
- `metadata` — JSON map written at call-start. Currently carries one key: `prompt_snapshots` (a `[]PromptSnapshot` capturing the AI/team prompt versions active when the call began). Future keys may be added without schema migration.

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
| `get_resource` | Retrieve a curated summary of a single resource by type+id (call, groupcall, recording, transcribe incl. transcripts, summary, aicall incl. conversation history, conferencecall, queuecall); customer-ownership enforced |

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
