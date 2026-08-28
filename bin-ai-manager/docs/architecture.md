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
| Transport | `pkg/subscribehandler` | Consumes call/pipecat events via pattern bindings on the global topic exchange `bin-manager.event` (VOIP-1406); fanout legs retained as rollback surface until VOIP-1407 |
| Transport | `notifyhandler` (via bin-common-handler) | Publishes events to the fanout exchange `bin-manager.ai-manager.event` and, since VOIP-1405, dual-publishes the same payload to the global topic exchange `bin-manager.event` |
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
| `POST /v1/ais/<uuid>/activate_insight` | Make this Insight AI the customer's active one |
| `POST /v1/ais/<uuid>/direct-hash-regenerate` | Regenerate AI secret hash |
| `GET /v1/aicalls?` | List AI call sessions (paginated) |
| `GET /v1/aicalls/<uuid>` | Get AI call session |
| `POST /v1/aicalls` | Start AI call session |
| `POST /v1/aicalls/<uuid>/terminate` | Terminate AI call |
| `POST /v1/aicalls/<uuid>/tool_execute` | Execute LLM tool (called by pipecat-manager) |
| `GET /v1/aicalls/<uuid>/participants(\?|$)` | List participants of an AI call (paginated) |
| `GET /v1/ais/<uuid>/participants(\?|$)` | List AI calls an AI agent participated in (paginated) |
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
| `GET /v1/ais/<uuid>/prompt_histories` | List prompt history for an AI |
| `GET /v1/ais/<uuid>/prompt_histories/<uuid>` | Get single AI prompt history entry |
| `GET /v1/aiaudits?` | List AI audit records (paginated) |
| `POST /v1/aiaudits` | Create AI audit records for an AIcall |
| `GET /v1/aiaudits/<uuid>` | Get a single AI audit record |
| `DELETE /v1/aiaudits/<uuid>` | Soft-delete an AI audit record |
| `GET /v1/aipromptproposals?` | List AI prompt improvement proposals (paginated) |
| `POST /v1/aipromptproposals` | Create an AI prompt improvement proposal from a set of audits |
| `GET /v1/aipromptproposals/<uuid>` | Get a single AI prompt improvement proposal |
| `POST /v1/aipromptproposals/<uuid>/accept` | Accept a proposal and apply it to the AI's `init_prompt` |
| `POST /v1/aipromptproposals/<uuid>/reject` | Reject a proposal without applying it |
| `POST /v1/aipromptproposals/expire` | Sweep endpoint: expire completed proposals whose basis prompt has drifted (invoked by schedule-manager cron, replaces the old in-process ticker) |
| `DELETE /v1/aipromptproposals/<uuid>` | Soft-delete an AI prompt improvement proposal |

## Event Subscriptions

SubscribeHandler (`pkg/subscribehandler/`) consumes from the queue `bin-manager.ai-manager.subscribe`. Since VOIP-1406 the queue is bound to the **global topic exchange `bin-manager.event`** with one pattern per dispatched (publisher, event-type) pair — 10 patterns total, pinned byte-for-byte by the binding golden test (`pkg/subscribehandler/binding_golden_test.go`):

| Pattern | Purpose |
|---------|---------|
| `call-manager.confbridge.*.joined` / `call-manager.confbridge.*.leaved` | Confbridge join/leave — drives AIcall state transitions |
| `call-manager.call.*.hangup` | Call hangup — terminates the AIcall |
| `call-manager.dtmf.*.received` | DTMF input during an AIcall |
| `pipecat-manager.message.*.user_transcription` / `.bot_llm` / `.bot_llm_intermediate` | Realtime conversation messages |
| `pipecat-manager.pipecatcall.*.initialized` / `.terminated` | Pipecat session lifecycle |
| `pipecat-manager.team.*.member_switched` | AI team member switch |

The `conference-manager.conference.*.updated` pair is deliberately NOT bound: its dispatch case is unreachable today and stays that way (VOIP-1406 design §4; follow-up VOIP-1422 decides activate-or-delete).

The old per-service **fanout subscriptions are retained in code as the rollback surface until VOIP-1407** (`QueueSubscribe` to `bin-manager.call-manager.event`, `bin-manager.transcribe-manager.event`, `bin-manager.tts-manager.event`, `bin-manager.pipecat-manager.event`); on each boot Run() re-subscribes them, then unbinds all four again after the topic binds succeed. The transcribe and tts legs were dead binds (zero dispatch cases) and are dropped the same way.

## Events Published

Exchange: `bin-manager.ai-manager.event` (fanout, system of record) and — since VOIP-1405 — the global topic exchange `bin-manager.event`.

`cmd/ai-manager` and both NotifyHandler instances inside `cmd/ai-control` construct their NotifyHandler with `notifyhandler.WithGlobalTopicPublish()`, so every event is published twice: once to the per-service fanout exchange (unchanged) and once to the global topic exchange with the routing key `ai-manager.<resource>.<subscription-id>.<action>`. All three construction sites must stay in lockstep on this option — enabling it in only some would leave consumers with gaps depending on which process published. A topic publish failure never propagates to the caller and never affects the fanout publish. See [docs/domain.md](domain.md) for the per-event routing keys and the monorepo `docs/plans/2026-08-27-voip-1404-global-topic-exchange-design.md` for the schema.

| Event type | Trigger |
|-----------|---------|
| `ai.EventTypeCreated` | AI configuration created |
| `ai.EventTypeUpdated` | AI configuration updated |
| `ai.EventTypeDeleted` | AI configuration deleted |
| `aicall.EventTypeStatusInitializing` | AIcall session created |
| `aicall.EventTypeStatusProgressing` | AIcall started processing |
| `aicall.EventTypeStatusPausing` | AIcall paused |
| `aicall.EventTypeStatusResuming` | AIcall resumed |
| `aicall.EventTypeStatusTerminating` | AIcall termination started |
| `aicall.EventTypeStatusTerminated` | AIcall terminated |
| `message.EventTypeMessageCreated` | New message added to conversation |
| `message.EventTypeMessageIntermediate` | Streaming/intermediate message fragment |
| `summary.EventTypeCreated` | Summary created |
| `summary.EventTypeUpdated` | Summary updated |
| `summary.EventTypeDeleted` | Summary deleted |
| `team.EventTypeCreated` | AI team created |
| `team.EventTypeUpdated` | AI team updated |
| `team.EventTypeDeleted` | AI team deleted |

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
