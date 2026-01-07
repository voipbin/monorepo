# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`bin-ai-manager` is a Go service within the VoipBin monorepo that manages AI-powered voice conversations. It orchestrates AI calls, processes messages through various LLM engines (OpenAI, Dialogflow, etc.), handles speech-to-text/text-to-speech operations, and integrates with the broader VoipBin telephony platform.

This service operates as an event-driven microservice using RabbitMQ for message passing and Redis for caching.

## Development Commands

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

## Architecture

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
- EngineModel format: `<target>.<model>` (e.g., `openai.gpt-4o`, `dialogflow.cx`)
- Supports 18+ LLM providers (Anthropic, AWS, Azure, Cerebras, DeepSeek, etc.)
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

### Integration Points

**Monorepo Dependencies**
This service depends on several sibling services in the monorepo:
- `bin-common-handler`: Shared utilities (database, request/notify handlers, socket handling)
- `bin-call-manager`: Telephony call management
- `bin-pipecat-manager`: Real-time audio streaming with Pipecat framework
- `bin-flow-manager`: Conversation flow orchestration

All monorepo dependencies use `replace` directives in `go.mod` pointing to `../<service-name>`

**External Dependencies**
- MySQL database for persistent storage
- Redis for caching
- RabbitMQ for event-driven messaging
- Google Cloud Dialogflow API
- OpenAI API (and other LLM providers)

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

## Testing Patterns

- Tests use mockgen-generated mocks for dependencies
- Test files co-located with implementation: `<package>/<feature>_test.go`
- Database operations tested with mock DBHandler
- Example: `pkg/aicallhandler/start_test.go` tests AIcall creation with various scenarios

## Configuration

The service reads configuration from environment variables (though not explicitly shown in code, this is typical for the monorepo pattern):
- Database DSN (MySQL connection string)
- Redis address, password, database number
- RabbitMQ address
- Engine API keys (e.g., ChatGPT key)
- Prometheus metrics endpoint

## Prometheus Metrics

The service exports metrics on:
- `ai_manager_ai_create_total`: AI configurations created (by engine_type)
- `ai_manager_message_create_total`: Messages created (by engine_type)
- `ai_manager_message_process_time`: Message processing latency
- `ai_manager_subscribe_event_process_time`: Event processing latency (by publisher and type)

## CI/CD Pipeline

See `.gitlab-ci.yml` for the full pipeline:
- **ensure**: Download and vendor dependencies
- **test**: Run linting (golint, golangci-lint), vet, and tests with coverage
- **build**: Build Docker image and push to registry
- **release**: Deploy to GKE using kustomize (manual trigger)

## Common Patterns

**Handler Pattern**
All domain handlers follow this structure:
```go
type FooHandler interface {
    // Interface methods
}

type fooHandler struct {
    // dependencies injected
    db DBHandler
    reqHandler RequestHandler
    notifyHandler NotifyHandler
}

func NewFooHandler(...) FooHandler {
    return &fooHandler{...}
}
```

**Error Handling**
- Errors are logged with logrus at the point of occurrence
- Errors propagate up to the handler layer
- HTTP-style status codes used in request/response even though transport is RabbitMQ

**Context Usage**
- All public handler methods accept `context.Context` as first parameter
- Used for cancellation and timeout propagation
- Database and external API calls respect context

**UUID Usage**
- All IDs are UUID v4 using `github.com/gofrs/uuid` package
- Identity model from common-handler provides base ID/CustomerID fields
