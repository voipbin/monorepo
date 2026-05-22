# bin-ai-manager Domain Model

## Core Concepts

### AI (Configuration)
Per-customer AI agent configuration stored in MySQL. Defines which LLM engine to use, voice settings, available tools, and initial prompts.

Key fields:
- `engine_type` — provider identifier (see engine list below)
- `engine_model` — format `<target>.<model>` e.g. `openai.gpt-4o`, `grok.grok-3`, `dialogflow.cx`
- `init_prompt` — system prompt injected at session start
- `tool_names` — list of LLM tool names enabled for this AI
- `tts_type`, `stt_type` — voice provider selection
- `voice_gender`, `voice_language` — TTS voice selection parameters

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

### Message
Individual message within an AIcall conversation. Persisted in MySQL for context replay and summaries.

- `role`: `system` | `user` | `assistant` | `tool`
- `direction`: `inbound` | `outbound`
- `active_ai_id` — UUID of the AI configuration that was active when the message was created; `uuid.Nil` if the aicall or team lookup fails at creation time, or for non-AICall reference paths
- Supports tool call payloads for function-calling workflows

### Summary
Async LLM-generated summary of an AIcall's message history.

Status: `processing` → `done` | `failed`

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
| `send_email` | Send an email via email-manager |
| `send_message` | Send SMS via message-manager |
| `stop_media` | Stop current TTS audio playback |
| `stop_service` | Soft-end the AI conversation |
| `stop_flow` | Hard-terminate the entire flow |
| `set_variables` | Write to flow context variables |
| `get_variables` | Read from flow context variables |
| `get_aicall_messages` | Retrieve conversation history |

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

**Write path:** `aihandler.Create` and `aihandler.Update` insert rows after the AI DB write succeeds. Insert is best-effort (failure is logged; AI operation succeeds).

**Empty prompt:** No row is inserted when `init_prompt == ""`.

## Soft-Delete Pattern

All entities use `tm_delete = "9999-01-01 00:00:00.000000"` for active records. Deleted records receive the actual deletion timestamp, preserving history for audit and message replay.
