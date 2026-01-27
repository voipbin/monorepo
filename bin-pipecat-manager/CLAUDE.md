# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

bin-pipecat-manager is a hybrid Go/Python microservice that manages AI-powered voice conversations in a VoIP system. It integrates Pipecat (a Python framework for voice AI pipelines) with Go-based infrastructure, coordinating real-time audio streaming, LLM interactions, STT (Speech-to-Text), and TTS (Text-to-Speech) services.

## Build and Test Commands

```bash
# Build the Go daemon
go build -o ./bin/ ./cmd/...

# Run the daemon (requires configuration via env vars)
./bin/pipecat-manager

# Run all Go tests
go test ./...

# Run tests with coverage
go test -coverprofile cp.out -v $(go list ./...)
go tool cover -html=cp.out -o cp.html

# Run a single test
go test -v -run TestName ./pkg/packagename/...

# Generate mocks (uses go.uber.org/mock via //go:generate directives)
go generate ./pkg/pipecatcallhandler/...
go generate ./pkg/dbhandler/...
go generate ./pkg/cachehandler/...

# Build Docker image (includes both Go and Python components)
docker build -t pipecat-manager -f Dockerfile ../

# Generate protobuf (if modifying proto/frames.proto)
protoc --go_out=. --go_opt=paths=source_relative proto/frames.proto
```

## pipecat-control CLI Tool

A command-line tool for managing pipecat calls. **All output is JSON format** (stdout), logs go to stderr.

**Note:** This tool requires the soxr system library for audio processing.

```bash
# Get a pipecatcall - returns pipecatcall JSON
./bin/pipecat-control pipecatcall get --id <uuid>

# Start a new pipecatcall - returns created pipecatcall JSON
./bin/pipecat-control pipecatcall start --reference_type <type> --reference_id <uuid> [--customer_id]

# Terminate a pipecatcall - returns terminated pipecatcall JSON
./bin/pipecat-control pipecatcall terminate --id <uuid>

# Send a message to a pipecatcall
./bin/pipecat-control pipecatcall send-message --id <uuid> --message <text>
```

Uses same environment variables as pipecat-manager (`DATABASE_DSN`, `RABBITMQ_ADDRESS`, `REDIS_ADDRESS`, etc.).

### Python Component

The Python FastAPI service (`scripts/pipecat/main.py`) runs on port 8000 and must be running for the Go service to function:

```bash
cd scripts/pipecat

# Install dependencies
pip install -r requirements.txt

# Run the Python API server
python main.py

# Or with uvicorn directly
uvicorn main:app --host 0.0.0.0 --port 8000
```

## Architecture

### Hybrid Go/Python Design

This service uniquely combines Go and Python components:

1. **Go Service (Port 8080)** - Main orchestration layer
   - HTTP endpoints for WebSocket connections from Pipecat Python runners
   - RabbitMQ RPC request handler for creating/managing pipecat calls
   - Audiosocket connections to Asterisk for real-time audio streaming
   - Session lifecycle management and state coordination
   - Database and cache operations

2. **Python Service (Port 8000)** - Pipecat pipeline executor
   - FastAPI server with `/run` and `/stop` endpoints
   - Executes Pipecat AI voice pipelines (STT → LLM → TTS)
   - WebSocket client connecting back to Go service
   - Async task management for concurrent pipeline executions

### Service Layer Structure

**Go Packages:**

1. **cmd/pipecat-manager/** - Main entry point with Cobra/Viper configuration
2. **pkg/pipecatcallhandler/** - Core orchestration logic
   - `pythonrunner.go` - HTTP client communicating with Python FastAPI service
   - `audiosocket.go` - Handles Audiosocket protocol for Asterisk audio streaming
   - `websocket.go` - WebSocket server for Pipecat client connections
   - `pipecatframe.go` - Protobuf frame serialization/deserialization
   - `session.go` - Manages active pipecat call sessions
   - `run.go`, `runner.go` - Pipeline execution coordination
3. **pkg/listenhandler/** - RabbitMQ RPC handler with regex-based routing
4. **pkg/httphandler/** - HTTP endpoints for WebSocket upgrades and tool callbacks
5. **pkg/dbhandler/** - MySQL operations with Redis caching
6. **pkg/cachehandler/** - Redis cache operations
7. **models/pipecatcall/** - Data structures for pipecat sessions
8. **models/pipecatframe/** - Protobuf frame definitions (generated from proto/frames.proto)
9. **models/message/** - Message structures for pipecat interactions

**Python Scripts:**

1. **scripts/pipecat/main.py** - FastAPI server entry point
2. **scripts/pipecat/run.py** - Pipecat pipeline construction and execution
3. **scripts/pipecat/tools.py** - LLM function calling tools (connect_call, send_email, etc.)
4. **scripts/pipecat/task.py** - Async task lifecycle management
5. **scripts/pipecat/common.py** - Shared utilities

### Inter-Service Communication

- **RabbitMQ**: Message passing between microservices
  - Listens on `bin-manager.pipecat-manager.request` (shared queue)
  - Listens on `QueueNamePipecatRequest.<host_id>` (volatile, per-pod queue)
  - Publishes events to `QueueNamePipecatEvent`
- **HTTP/WebSocket**:
  - Go → Python: HTTP POST to `localhost:8000/run` and `/stop`
  - Python → Go: WebSocket client to `<POD_IP>:8080` for bidirectional frame streaming
- **Audiosocket**: TCP connection to Asterisk for 8kHz PCM audio streaming
- **Monorepo**: Sibling services referenced via `replace` directives in go.mod pointing to `../bin-*-manager`

### Request Flow: Starting a Pipecat Call

```
RabbitMQ Request (POST /v1/pipecatcalls)
  ↓
listenhandler.processV1PipecatcallsPost
  ↓
pipecatcallhandler.Start
  ├─→ Create DB record
  ├─→ pythonrunner.Start (HTTP POST to Python :8000/run)
  │    ↓
  │   Python run_pipeline creates:
  │    ├─ LLM service (OpenAI, etc.)
  │    ├─ STT service (Deepgram, Whisper)
  │    ├─ TTS service (Cartesia, ElevenLabs)
  │    └─ WebSocket client → connects to Go :8080
  │         ↓
  └─→ pipecatcallhandler.Run (goroutine)
       └─→ Accepts Audiosocket from Asterisk
            └─→ Bidirectional audio bridge:
                 Asterisk (8kHz) ←audiosocket→ Go ←websocket/protobuf→ Python/Pipecat
```

### Audio Flow Architecture

```
Asterisk PBX (8kHz SLIN)
  ↓ [Audiosocket TCP]
Go audiosocketHandler.GetNextMedia() → 8kHz PCM bytes
  ↓ [Upsample to 16kHz if needed]
Go pipecatframeHandler.CreateAudioRawFrame() → Protobuf AudioRawFrame
  ↓ [WebSocket]
Python Pipecat pipeline:
  AudioRawFrame → STT → LLMContext → LLM → TTS → AudioRawFrame
  ↓ [WebSocket]
Go pipecatframeHandler.Parse() → Extract PCM bytes
  ↓ [Downsample 16kHz → 8kHz if needed]
Go audiosocketHandler.Write() → 8kHz PCM bytes
  ↓ [Audiosocket TCP]
Asterisk PBX
```

### Key Patterns

- **Interface-based mocking**: All handlers use interfaces with `//go:generate mockgen` for testability
- **Protobuf frames**: Custom proto definitions in `proto/frames.proto` for efficient WebSocket communication
- **Session management**: `pipecatcall.Session` tracks active connections (Audiosocket, WebSocket, contexts)
- **Dual transport**: Asterisk uses Audiosocket (8kHz), Pipecat uses WebSocket+Protobuf (16kHz)
- **Sample rate conversion**: `Upsample8kTo16k` and downsample for audio quality matching
- **Context propagation**: All handler methods accept `context.Context` for cancellation
- **UUID-based IDs**: Using `github.com/gofrs/uuid` throughout

### Configuration

Environment variables / flags:
- `DATABASE_DSN` - MySQL connection string
- `RABBITMQ_ADDRESS` - RabbitMQ connection (e.g., `amqp://guest:guest@localhost:5672`)
- `REDIS_ADDRESS`, `REDIS_PASSWORD`, `REDIS_DATABASE` - Redis cache
- `POD_IP` - **Required**: IP address for Python to connect back via WebSocket
- `PROMETHEUS_ENDPOINT`, `PROMETHEUS_LISTEN_ADDRESS` - Metrics endpoint

Python environment variables (in `.env` or exported):
- LLM provider API keys: `OPENAI_API_KEY`, `ANTHROPIC_API_KEY`, etc.
- STT provider keys: `DEEPGRAM_API_KEY`
- TTS provider keys: `CARTESIA_API_KEY`, `ELEVENLABS_API_KEY`, `GOOGLE_API_KEY`

### Pipecat Integration Details

- **pipecat-ai framework**: Python library for building voice AI pipelines
- **Supported providers**:
  - LLM: OpenAI (configurable model via `llm_type` like "openai.gpt-4")
  - STT: Deepgram, Whisper
  - TTS: Cartesia, ElevenLabs, Google
- **RTVI protocol**: Used for runtime control and tool calling
- **VAD**: Silero Voice Activity Detection for turn-taking
- **Tool calling**: LLM can invoke VoIP functions (see `tools.py` for `connect_call`, `send_email`, `stop_flow`, etc.)
- **Frame types**: TextFrame, AudioRawFrame, TranscriptionFrame, MessageFrame (defined in proto)

### Important Implementation Notes

1. **Audio Sample Rates**:
   - Asterisk/Audiosocket: 8kHz SLIN (signed linear 16-bit PCM)
   - Pipecat/WebSocket: 16kHz (some services require higher quality)
   - Conversion handled in `audiosocketHandler.GetDataSamples()` and `Upsample8kTo16k()`

2. **Protobuf Frame Protocol**:
   - All WebSocket messages between Go and Python use protobuf-serialized frames
   - Generated code: `models/pipecatframe/frames.pb.go` (Go), Python uses `pipecat.serializers.protobuf`

3. **Session Lifecycle**:
   - Created in DB on `/v1/pipecatcalls POST`
   - Python runner started asynchronously
   - Audiosocket connection expected from Asterisk
   - WebSocket connection from Python Pipecat client
   - All cleaned up on `/v1/pipecatcalls/<id>/stop POST`

4. **Tool Execution Flow**:
   - Python Pipecat LLM makes function call
   - Python sends HTTP request to Go `httpHandler.RunnerToolHandle`
   - Go makes RPC request to ai-manager service
   - Response returned to Python → Pipecat → LLM context

5. **Monorepo Dependencies**:
   - Imports from `monorepo/bin-common-handler` (shared utilities, RabbitMQ, database)
   - Imports from `monorepo/bin-call-manager` (external media models)
   - Imports from `monorepo/bin-ai-manager` (AI call integration)
