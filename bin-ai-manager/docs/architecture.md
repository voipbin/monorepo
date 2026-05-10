# bin-ai-manager Architecture

## Component Overview

`bin-ai-manager` manages AI-powered voice and text conversations. It orchestrates AI call sessions, routes messages to LLM engines, and coordinates real-time audio processing with `bin-pipecat-manager`.

```
cmd/ai-manager/main.go
    ├── pkg/dbhandler          (MySQL + Redis cache)
    ├── pkg/cachehandler       (Redis operations)
    ├── pkg/listenhandler      (RabbitMQ RPC request router)
    ├── pkg/subscribehandler   (event consumer)
    ├── pkg/aihandler          (AI config CRUD)
    ├── pkg/aicallhandler      (conversation session lifecycle)
    ├── pkg/messagehandler     (message storage + engine dispatch)
    ├── pkg/summaryhandler     (async LLM summaries)
    ├── pkg/toolhandler        (LLM function-call definitions)
    ├── pkg/engine_openai_handler    (OpenAI/Grok API integration)
    └── pkg/engine_dialogflow_handler (Dialogflow CX/ES integration)
```

**Supporting binaries:**
- `cmd/ai-control/` — CLI tool for direct DB/cache operations (bypasses RabbitMQ)

## Layer Responsibilities

| Layer | Package(s) | Responsibility |
|-------|-----------|----------------|
| Transport | `pkg/listenhandler` | Receives RPC requests from `bin-manager.ai-manager.request`, routes by URI regex |
| Transport | `pkg/subscribehandler` | Consumes events from call, transcribe, tts, pipecat queues |
| Transport | `notifyhandler` (via bin-common-handler) | Publishes events to `bin-manager.ai-manager.event` |
| Domain | `pkg/aihandler` | AI configuration CRUD (engine type, model, TTS/STT settings, tool list) |
| Domain | `pkg/aicallhandler` | AIcall session lifecycle: initiating → progressing → terminating → terminated |
| Domain | `pkg/messagehandler` | Message storage, engine selection, real-time transcript processing |
| Domain | `pkg/summaryhandler` | Async summary generation via LLM |
| Domain | `pkg/toolhandler` | LLM tool definitions; dispatches tool calls to downstream managers |
| Engine | `pkg/engine_openai_handler` | OpenAI Chat Completions API (also Grok via base URL override) |
| Engine | `pkg/engine_dialogflow_handler` | Google Dialogflow CX/ES |
| Data | `pkg/dbhandler` | MySQL CRUD via Squirrel SQL builder |
| Data | `pkg/cachehandler` | Redis cache for AI/AIcall lookups |

## Request Routing

ListenHandler (`pkg/listenhandler/`) routes by regex URI pattern over the shared queue `bin-manager.ai-manager.request`.

| Pattern | Purpose |
|---------|---------|
| `GET /v1/ais?` | List AI configurations (paginated) |
| `GET/PUT/DELETE /v1/ais/<uuid>` | Get / update / delete AI config |
| `POST /v1/ais` | Create AI configuration |
| `POST /v1/ais/<uuid>/direct-hash-regenerate` | Regenerate AI secret hash |
| `GET /v1/aicalls?` | List AI call sessions (paginated) |
| `GET /v1/aicalls/<uuid>` | Get AI call session |
| `POST /v1/aicalls` | Start AI call session |
| `POST /v1/aicalls/<uuid>/terminate` | Terminate AI call |
| `POST /v1/aicalls/<uuid>/tool_execute` | Execute LLM tool (called by pipecat-manager) |
| `GET /v1/messages?` | List messages |
| `GET/POST /v1/messages/<uuid>` | Get / create message |
| `POST /v1/services/type/aicall` | Create AI call service (used by flow-manager) |
| `POST /v1/services/type/summary` | Create summary service |
| `POST /v1/services/type/task` | Create task service |
| `GET /v1/summaries?` | List summaries |
| `GET/POST /v1/summaries/<uuid>` | Get / create summary |
| `GET /v1/tools` | List available LLM tools |
| `GET /v1/teams?` | List AI teams |
| `GET/POST /v1/teams/<uuid>` | Get / create AI team |
| `POST /v1/teams/<uuid>/direct-hash-regenerate` | Regenerate team secret hash |

## Event Subscriptions

SubscribeHandler (`pkg/subscribehandler/`) consumes:

| Queue | Event types handled |
|-------|-------------------|
| `bin-manager.call-manager.event` | Call hangup, conference join/leave — drives AIcall state transitions |
| `bin-manager.transcribe-manager.event` | Transcription results for non-realtime flows |
| `bin-manager.tts-manager.event` | TTS lifecycle events |
| `bin-manager.pipecat-manager.event` | Pipecat session initialized, message arrived — drives realtime conversation state |

## Events Published

Exchange: `bin-manager.ai-manager.event`

| Event type | Trigger |
|-----------|---------|
| `ai.EventTypeCreated` | AI configuration created |
| `ai.EventTypeUpdated` | AI configuration updated |
| `ai.EventTypeDeleted` | AI configuration deleted |
| `message.EventTypeMessageCreated` | New message added to conversation |
| `message.EventTypeMessageIntermediate` | Streaming/intermediate message fragment |

## AI Manager ↔ Pipecat Manager Relationship

`bin-ai-manager` owns orchestration and persistence; `bin-pipecat-manager` owns real-time audio processing.

```
Flow Manager
    │ POST /v1/aicalls (RabbitMQ RPC)
    ▼
AI Manager (Go) ──RabbitMQ──► Pipecat Manager (Python)
    │                               │
    │ tool_execute RPC ◄─HTTP──────┘
    │
    ▼
call-manager / message-manager / email-manager (tool dispatch)
```

Follow-up RPCs to pipecat-manager (tool results, stop) target the **per-pod queue** (`bin-manager.pipecat-manager.request.<POD_IP>`) using `pipecatcall.HostID`. See [docs/patterns/per-pod-queues.md](../../docs/patterns/per-pod-queues.md).
