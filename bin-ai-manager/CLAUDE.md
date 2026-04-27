# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

`bin-ai-manager` is a Go service within the VoIPbin monorepo that manages AI-powered voice conversations. It orchestrates AI calls, processes messages through various LLM engines (OpenAI, Dialogflow, etc.), handles speech-to-text/text-to-speech operations, and integrates with the broader VoIPbin telephony platform.

**Key Concepts:**
- **AI**: Per-customer AI configuration (engine type, model, init prompt, TTS/STT settings).
- **AIcall**: Active conversation session linking an AI configuration to a reference (call/conversation/task) with lifecycle status.
- **Engine**: LLM provider integration (OpenAI, Dialogflow, Grok, Gemini, etc.); messages are routed to the configured engine handler.
- **Tool**: Function-calling capability exposed to the LLM (`connect_call`, `send_email`, `set_variables`, etc.).
- **Pipecat integration**: Real-time audio (STT → LLM → TTS) is delegated to `bin-pipecat-manager`; this service owns orchestration and persistence.

This service operates as an event-driven microservice using RabbitMQ for message passing and Redis for caching.

> Cross-cutting rules (verification workflow, branch/commit format, worktree usage, Alembic, RST sync) live in the root [CLAUDE.md](../CLAUDE.md). This file documents only what is specific to `bin-ai-manager`.

## Common Commands

### Testing
```bash
# Run all tests with coverage
go test -coverprofile cp.out -v $(go list ./...)

# View coverage report
go tool cover -html=cp.out -o cp.html

# Run tests for a specific package
go test -v ./pkg/aihandler/...

# Run a single test
go test -v ./pkg/aihandler -run TestSpecificTest
```

### Building
```bash
# Build the binary
go build -o ./bin/ai-manager ./cmd/ai-manager/

# Build using Docker (from monorepo root)
docker build -t ai-manager:latest -f bin-ai-manager/Dockerfile .
```

### Code Quality
```bash
# Run golint
golint -set_exit_status $(go list ./...)

# Run golangci-lint (comprehensive linting)
golangci-lint run -v --timeout 5m

# Run go vet
go vet $(go list ./...)
```

### Dependency Management
```bash
# Download dependencies
go mod download

# Vendor dependencies (required for CI/CD)
go mod vendor

# Update specific dependency
go get <package>@<version>
go mod tidy
go mod vendor
```

### Mock Generation
Mocks are generated using mockgen. To regenerate mocks for a package:
```bash
# The go:generate directives are in each main.go file
go generate ./pkg/aihandler/...
go generate ./pkg/aicallhandler/...
# etc.
```

## ai-control CLI Tool

A command-line tool for managing AI configurations directly via database/cache (bypasses RabbitMQ RPC). **All output is JSON format** (stdout), logs go to stderr.

```bash
# Create AI configuration - returns created AI JSON
./bin/ai-control ai create --customer_id <uuid> --name <name> --engine_type <type> --engine_model <model> [--parameter '<json>'] [--init_prompt '<text>']

# Get AI configuration - returns AI JSON
./bin/ai-control ai get --id <uuid>

# List AI configurations - returns JSON array
./bin/ai-control ai list --customer_id <uuid> [--limit 100] [--token]

# Update AI configuration - returns updated AI JSON
./bin/ai-control ai update --id <uuid> [--name <name>] [--engine_type <type>] [--engine_model <model>] [--parameter '<json>'] [--init_prompt '<text>']

# Delete AI configuration - returns deleted AI JSON
./bin/ai-control ai delete --id <uuid>
```

Uses same environment variables as ai-manager (`DATABASE_DSN`, `RABBITMQ_ADDRESS`, `REDIS_ADDRESS`, etc.).

## Architecture

### Service Communication Pattern

This service uses **RabbitMQ for event-driven RPC communication**:
- **ListenHandler** (`pkg/listenhandler/`): Consumes RPC requests from queue `bin-manager.ai-manager.request` (routes: `/v1/ais`, `/v1/aicalls`, `/v1/messages`, `/v1/summaries`, `/v1/services`)
- **SubscribeHandler** (`pkg/subscribehandler/`): Subscribes to call/transcribe/tts/pipecat events
- **NotifyHandler**: Publishes AI lifecycle events to exchange `bin-manager.ai-manager.event`

### Core Components

**Main Entry Point** (`cmd/ai-manager/main.go:30-178`)
- Initializes database (MySQL), cache (Redis), and message queue (RabbitMQ) connections
- Sets up two main event processing paths: `listenhandler` and `subscribehandler`
- Runs as a long-lived service with signal handling

**ListenHandler** (`pkg/listenhandler/`)
- Processes synchronous RPC-style requests via RabbitMQ queue (`QueueNameAIRequest`)
- Implements REST-like API handlers using regex pattern matching
- Routes: `/v1/ais`, `/v1/aicalls`, `/v1/messages`, `/v1/summaries`, `/v1/services`
- Returns responses directly to caller

**SubscribeHandler** (`pkg/subscribehandler/`)
- Processes asynchronous events from other services
- Subscribes to: `QueueNameCallEvent`, `QueueNameTranscribeEvent`, `QueueNameTTSEvent`, `QueueNamePipecatEvent`
- Handles call lifecycle events (hangup, conference join/leave), transcriptions, and pipecat messages

### Domain Handlers

**AIHandler** (`pkg/aihandler/`)
- CRUD operations for AI configurations (models/ai/)
- Manages AI engine settings: type, model, init prompts, TTS/STT configuration
- Tracks engine types: OpenAI, Dialogflow, and many others (see `models/ai/main.go:41-88`)

**AIcallHandler** (`pkg/aicallhandler/`)
- Manages active AI conversation sessions (models/aicall/)
- Lifecycle: Start → ProcessStart → Progressing → Terminate
- Integrates with call-manager (telephony), pipecat-manager (real-time streaming), flow-manager
- Handles tool execution for function calling
- Reference types: call (telephony), conversation (chat), task (background processing)

**MessageHandler** (`pkg/messagehandler/`)
- Stores and processes conversation messages (models/message/)
- Routes messages to appropriate engine handler (OpenAI vs Dialogflow)
- Processes events from pipecat-manager for real-time transcription and LLM responses
- Tracks message roles: system, user, assistant, tool

**SummaryHandler** (`pkg/summaryhandler/`)
- Generates call/conversation summaries using LLMs
- Async processing pattern: create → process → update status

**Engine Handlers**
- `engine_openai_handler/`: OpenAI API integration for message processing
- `engine_dialogflow_handler/`: Google Dialogflow CX/ES integration

### Data Models

**AI** (`models/ai/main.go:8-31`)
- Configuration for an AI agent
- EngineModel format: `<target>.<model>` (e.g., `openai.gpt-4o`, `grok.grok-3`, `dialogflow.cx`)
- Supports 18+ LLM providers (Anthropic, AWS, Azure, Cerebras, DeepSeek, Grok, etc.)
- TTS types: cartesia, deepgram, elevenlabs, openai, etc.
- STT types: cartesia, deepgram, elevenlabs

**AIcall** (`models/aicall/main.go:10-38`)
- Active conversation session linking AI config to a reference (call/conversation/task)
- Tracks confbridge_id (conference bridge) and pipecatcall_id (streaming session)
- Status flow: initiating → progressing → pausing/resuming → terminating → terminated
- Gender and language settings for TTS voice selection

**Message** (`models/message/`)
- Individual message in a conversation
- Supports tool calls for function calling capabilities
- Direction: inbound/outbound, Role: system/user/assistant/tool

### AI Manager and Pipecat Manager Relationship

The AI Manager (Go) and Pipecat Manager (Python) work together to provide real-time AI voice conversations:

```
                              +-------------------+
                              |   Flow Manager    |
                              |  (ai_talk action) |
                              +--------+----------+
                                       |
                                       | Start AI session
                                       v
+-------------------+        +-------------------+        +-------------------+
|                   |        |                   |        |                   |
|    Asterisk       |<------>|   AI Manager      |<------>|  Pipecat Manager  |
|  (8kHz audio)     |  HTTP  |     (Go)          | RMQ/WS |    (Python)       |
|                   |        |                   |        |                   |
+-------------------+        +--------+----------+        +--------+----------+
       ^                              |                            |
       |                              |                            |
       | RTP audio                    | Tool                       | Real-time
       |                              | execution                  | processing
       v                              v                            v
+-------------------+        +-------------------+        +-------------------+
|       User        |        | call-manager      |        |    STT / LLM      |
|    (Phone)        |        | message-manager   |        |      / TTS        |
|                   |        | email-manager     |        |   Providers       |
+-------------------+        +-------------------+        +-------------------+
```

**Responsibilities:**

| Component | Responsibility |
|-----------|----------------|
| AI Manager (Go) | Orchestration, session management, tool execution, database persistence |
| Pipecat Manager (Python) | Real-time audio processing, STT/LLM/TTS pipeline, WebSocket streaming |

**Audio Flow with Sample Rate Conversion:**

```
User (Phone)       Asterisk        Pipecat         STT/LLM/TTS
     |                |               |                |
     | RTP 8kHz PCM   |               |                |
     +--------------->|               |                |
     |                | WebSocket     |                |
     |                | 8kHz audio    |                |
     |                +-------------->|                |
     |                |               | Resample to    |
     |                |               | 16kHz          |
     |                |               +--------------->| STT
     |                |               |                |
     |                |               |<---------------+ LLM Response
     |                |               | Resample to    |
     |                |               | 8kHz           |
     |                |<--------------+                |
     |<---------------| RTP playback  |                |
```

**Tool Execution Flow:**

When the LLM detects a function call (e.g., "transfer me to sales"):

1. Python Pipecat detects `function_call` in LLM response
2. Pipecat sends HTTP POST to Go AI Manager (`/tool/execute`)
3. AI Manager's `AIcallHandler.ToolHandle()` processes the request
4. Tool executed via appropriate service (call-manager, message-manager, etc.)
5. Result returned to Pipecat
6. LLM generates verbal response based on result
7. TTS converts response to audio

**Tool Definitions:**

Tool definitions are centralized in `pkg/toolhandler/definitions.go`. The Pipecat Manager requests these definitions when starting a session and only receives tools enabled via the AI's `tool_names` field.

Available tools:
- `connect_call`: Transfer/connect calls
- `send_email`: Send email messages
- `send_message`: Send SMS messages
- `stop_media`: Stop current media playback
- `stop_service`: End AI conversation (soft stop)
- `stop_flow`: Terminate entire flow (hard stop)
- `set_variables`: Save data to flow context
- `get_variables`: Retrieve data from flow context
- `get_aicall_messages`: Get message history

**Communication Patterns:**

| Direction | Protocol | Purpose |
|-----------|----------|---------|
| Flow → AI Manager | RabbitMQ RPC | Start/stop AI sessions |
| AI Manager → Pipecat | RabbitMQ + HTTP | Session management, tool results |
| Pipecat → AI Manager | HTTP | Tool execution requests |
| Asterisk ↔ Pipecat | WebSocket | Real-time audio streaming |
| Pipecat → STT/LLM/TTS | HTTP/WebSocket | External AI provider APIs |

**External Dependencies**
- MySQL database for persistent storage
- Redis for caching
- RabbitMQ for event-driven messaging
- Google Cloud Dialogflow API
- OpenAI API (and other LLM providers)

## Request Routing

ListenHandler routes requests using regex patterns matching REST-like URIs (see `pkg/listenhandler/`):

**AIs API (`/v1/ais/*`):**
- `POST /v1/ais` — Create AI configuration
- `GET /v1/ais?<filters>` — List AIs (pagination)
- `GET /v1/ais/<uuid>` — Get AI configuration
- `PUT /v1/ais/<uuid>` — Update AI configuration
- `DELETE /v1/ais/<uuid>` — Delete AI configuration

**AIcalls API (`/v1/aicalls/*`):**
- `POST /v1/aicalls` — Start AI call session
- `GET /v1/aicalls?<filters>` — List AI calls
- `GET /v1/aicalls/<uuid>` — Get AI call
- `POST /v1/aicalls/<uuid>/terminate` — Terminate AI call
- `POST /v1/aicalls/<uuid>/pause` / `POST /v1/aicalls/<uuid>/resume` — Pause/resume

**Messages API (`/v1/messages/*`):**
- `POST /v1/messages` — Create message
- `GET /v1/messages?<filters>` — List messages
- `GET /v1/messages/<uuid>` — Get message

**Summaries API (`/v1/summaries/*`):**
- `POST /v1/summaries` — Create summary (async LLM)
- `GET /v1/summaries/<uuid>` — Get summary
- `GET /v1/summaries?<filters>` — List summaries

**Services API (`/v1/services/*`):**
- `POST /v1/services/type/aicall` — Create AI call service (used by flow-manager)

## Event Subscriptions

SubscribeHandler subscribes to:
- **bin-manager.call-manager.event**: Call lifecycle events (hangup, conference join/leave) to drive AIcall state.
- **bin-manager.transcribe-manager.event**: Transcription events for non-realtime AI flows.
- **bin-manager.tts-manager.event**: TTS lifecycle events.
- **bin-manager.pipecat-manager.event**: Pipecat session events (initialized, message arrived) — drives realtime conversation state.

### Event Flow Examples

**Inbound AI Call Flow**
1. External service creates AI call via listenhandler (`POST /v1/aicalls`)
2. AIcallHandler.Start() creates AIcall record
3. AIcallHandler.ProcessStart() initializes pipecat session
4. SubscribeHandler receives PipecatcallInitialized event
5. MessageHandler processes user transcriptions from pipecat
6. Engine handler generates AI response
7. Response sent back through pipecat for TTS playback
8. On call end, EventCMCallHangup triggers cleanup

**Summary Generation Flow**
1. Request via listenhandler (`POST /v1/summaries`)
2. SummaryHandler creates summary record with "processing" status
3. Retrieves messages from aicall
4. Sends to OpenAI for summarization
5. Updates summary record with result

## Monorepo Context

This service depends on local monorepo packages (see `go.mod` replace directives):
- `monorepo/bin-common-handler`: Shared utilities (database, request/notify handlers, socket handling)
- `monorepo/bin-call-manager`: Telephony call management
- `monorepo/bin-pipecat-manager`: Real-time audio streaming with Pipecat framework
- `monorepo/bin-flow-manager`: Conversation flow orchestration

`bin-ai-manager` is a per-pod-routing client of `bin-pipecat-manager`: follow-up RPCs target the Pipecat pod that owns the in-memory session via the per-pod queue convention. See `bin-pipecat-manager/CLAUDE.md` and [docs/patterns/per-pod-queues.md](../docs/patterns/per-pod-queues.md).

Always run `go mod vendor` after changing dependencies.

## Testing Patterns

Tests use **gomock** (go.uber.org/mock):
- Mockgen-generated mocks for handler dependencies
- Test files co-located with implementation: `<package>/<feature>_test.go`
- Database operations tested with mock DBHandler
- Example: `pkg/aicallhandler/start_test.go` tests AIcall creation with various scenarios

```go
tests := []struct {
    name      string
    input     InputType
    mockSetup func(*MockHandler)
    expectRes ResultType
    expectErr bool
}{
    {"success case", input1, setupMock1, expected1, false},
    {"error case", input2, setupMock2, nil, true},
}
for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        mc := gomock.NewController(t)
        defer mc.Finish()
        // test implementation
    })
}
```

## Key Implementation Details

### Handler Pattern
All domain handlers follow this structure:
```go
type FooHandler interface {
    // Interface methods
}

type fooHandler struct {
    db            DBHandler
    reqHandler    RequestHandler
    notifyHandler NotifyHandler
}

func NewFooHandler(...) FooHandler {
    return &fooHandler{...}
}
```

### AIcall Lifecycle
Status flow: `initiating` → `progressing` → (`pausing`/`resuming`) → `terminating` → `terminated`. Lifecycle is driven by RPC entrypoints and external events from `bin-pipecat-manager` and `bin-call-manager`.

### Engine Routing
`MessageHandler` selects an engine handler based on the AI's `engine_type`:
- `engine_openai_handler/`: OpenAI-compatible chat completions API (also used for Grok via base URL override).
- `engine_dialogflow_handler/`: Google Dialogflow CX/ES.

### Tool Execution
Tool definitions live in `pkg/toolhandler/definitions.go`; only tools enabled in the AI's `tool_names` field are exposed to the LLM. Pipecat invokes tools via HTTP POST to `AIcallHandler.ToolHandle()`, which dispatches to the appropriate manager.

### Error Handling & Context
- Errors are logged with logrus and propagated up to the handler layer
- All public handler methods accept `context.Context` as the first parameter
- IDs are UUID v4 (`github.com/gofrs/uuid`); identity model from `bin-common-handler` provides base `ID`/`CustomerID` fields

## Configuration

Uses **Viper + pflag** pattern (see `cmd/ai-manager/init.go`):

| Flag / Env | Description | Default |
|------------|-------------|---------|
| `database_dsn` / `DATABASE_DSN` | MySQL connection string | required |
| `rabbitmq_address` / `RABBITMQ_ADDRESS` | RabbitMQ server | required |
| `redis_address` / `REDIS_ADDRESS` | Redis cache | required |
| `redis_password` / `REDIS_PASSWORD` | Redis auth | optional |
| `redis_database` / `REDIS_DATABASE` | Redis DB index | optional |
| `chatgpt_key` / `CHATGPT_KEY` | OpenAI API key | required |
| `prometheus_endpoint` / `PROMETHEUS_ENDPOINT` | Metrics path | `/metrics` |
| `prometheus_listen_address` / `PROMETHEUS_LISTEN_ADDRESS` | Metrics port | `:2112` |

Engine-specific keys (Dialogflow service account, Grok/Gemini/Anthropic keys, etc.) are configured via the same env-var pattern.

## Prometheus Metrics

Service exports metrics on the configured endpoint (default `:2112/metrics`):
- `ai_manager_aicall_create_total` — counter of AIcalls created (label `reference_type`)
- `ai_manager_aicall_end_total` — counter of AIcalls ended (label `reference_type`)
- `ai_manager_aicall_duration_seconds` — histogram of AIcall duration (label `reference_type`)
- `ai_manager_aicall_tool_execute_total` — counter of tool executions (label `tool_name`)
- `ai_manager_message_create_total` — counter of messages created (label `role`)
- `ai_manager_subscribe_event_process_time` — histogram of event processing latency (labels `publisher`, `type`)

## CI/CD Pipeline

See `.gitlab-ci.yml` for the full pipeline:
- **ensure**: Download and vendor dependencies
- **test**: Run linting (golint, golangci-lint), vet, and tests with coverage
- **build**: Build Docker image and push to registry
- **release**: Deploy to GKE using kustomize (manual trigger)
