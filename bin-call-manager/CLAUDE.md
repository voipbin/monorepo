# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

`bin-call-manager` is the core telephony service in the VoIPbin monorepo. It manages call resources, handles Asterisk ARI (Asterisk REST Interface) events, executes atomic call actions, and orchestrates call lifecycle operations including conferences, recordings, external media streams, and group calls.

**Key Concepts:**
- **Call**: Individual call session with status tracking (ring/up/down/hangup), confbridge membership, recording state
- **Confbridge**: Conference bridge joining multiple calls together
- **Channel**: Asterisk channel representing a single media stream
- **Bridge**: Asterisk bridge connecting channels
- **Recording**: Call recording session with storage tracking
- **External Media**: WebRTC or external media streams integrated into calls/conferences
- **Group Call**: Multi-party call group managing multiple simultaneous calls

## Architecture

### Service Communication Pattern

This service uses **RabbitMQ for event-driven RPC communication**:
- **ListenHandler** (`pkg/listenhandler/`): Consumes RPC requests from queue `bin-manager.call-manager.request`, processes them, and returns responses
- **SubscribeHandler** (`pkg/subscribehandler/`): Subscribes to events from other services (asterisk-proxy, customer-manager, flow-manager, sentinel-manager)
- **NotifyHandler**: Publishes call lifecycle events to `bin-manager.call-manager.event` exchange

### Core Components

```
cmd/call-manager/main.go
    ├── Database (MySQL)
    ├── Cache (Redis via pkg/cachehandler)
    └── run()
        ├── pkg/dbhandler (MySQL operations)
        ├── Domain Handlers:
        │   ├── pkg/channelhandler (Channel operations)
        │   ├── pkg/bridgehandler (Bridge operations)
        │   ├── pkg/callhandler (Call business logic)
        │   ├── pkg/confbridgehandler (Conference management)
        │   ├── pkg/recordinghandler (Recording operations)
        │   ├── pkg/externalmediahandler (External media streams)
        │   └── pkg/groupcallhandler (Group call coordination)
        ├── pkg/arieventhandler (Asterisk ARI event processing)
        ├── runSubscribe() -> pkg/subscribehandler
        └── runRequestListen() -> pkg/listenhandler
```

**Layer Responsibilities:**
- `models/`: Data structures (call, confbridge, channel, bridge, recording, groupcall, ari events)
- `pkg/*handler/`: Domain-specific business logic handlers
- `pkg/dbhandler/`: Database operations (no query builder, uses direct SQL)
- `pkg/cachehandler/`: Redis caching for call/channel/bridge lookups
- `pkg/listenhandler/`: RabbitMQ RPC request routing with REST-like URI patterns
- `pkg/subscribehandler/`: Event consumption and processing
- `pkg/arieventhandler/`: Asterisk ARI event processing (channel/bridge state changes)

### Request Routing

ListenHandler routes requests using regex patterns matching REST-like URIs:

**Calls API (`/v1/calls/*`)**:
- `POST /v1/calls` - Create call
- `GET /v1/calls?<filters>` - List calls (pagination)
- `GET /v1/calls/<uuid>` - Get call details
- `POST /v1/calls/<uuid>` - Update call
- `DELETE /v1/calls/<uuid>` - Delete call
- `POST /v1/calls/<uuid>/hangup` - Hangup call
- `POST /v1/calls/<uuid>/hold` / `DELETE` - Hold/Unhold
- `POST /v1/calls/<uuid>/mute` / `DELETE` - Mute/Unmute
- `POST /v1/calls/<uuid>/moh` / `DELETE` - Music on hold on/off
- `POST /v1/calls/<uuid>/silence` / `DELETE` - Silence on/off
- `POST /v1/calls/<uuid>/recording_start` - Start recording
- `POST /v1/calls/<uuid>/recording_stop` - Stop recording
- `POST /v1/calls/<uuid>/play` - Play media
- `POST /v1/calls/<uuid>/talk` - TTS playback
- `POST /v1/calls/<uuid>/external-media` - Add external media
- `POST /v1/calls/<uuid>/health-check` - Health check

**Confbridges API (`/v1/confbridges/*`)**:
- `POST /v1/confbridges` - Create conference
- `GET /v1/confbridges/<uuid>` - Get conference details
- `DELETE /v1/confbridges/<uuid>` - Delete conference
- `POST /v1/confbridges/<uuid>/calls/<call-uuid>` - Add call to conference
- `DELETE /v1/confbridges/<uuid>/calls/<call-uuid>` - Remove call from conference
- `POST /v1/confbridges/<uuid>/recording_start` - Start conference recording
- `POST /v1/confbridges/<uuid>/recording_stop` - Stop conference recording
- `POST /v1/confbridges/<uuid>/external-media` - Add external media to conference
- `POST /v1/confbridges/<uuid>/flags` - Update conference flags
- `POST /v1/confbridges/<uuid>/terminate` - Terminate conference

**Channels API (`/v1/channels/*`)**:
- `POST /v1/channels/<channel-id>/health-check` - Channel health check

**Recordings API (`/v1/recordings/*`)**:
- `POST /v1/recordings` - Create recording
- `GET /v1/recordings?<filters>` - List recordings
- `GET /v1/recordings/<uuid>` - Get recording details
- `DELETE /v1/recordings/<uuid>` - Delete recording
- `POST /v1/recordings/<uuid>/stop` - Stop recording

**External Media API (`/v1/external-medias/*`)**:
- `POST /v1/external-medias` - Create external media stream
- `GET /v1/external-medias?<filters>` - List external media streams
- `GET /v1/external-medias/<uuid>` - Get external media details
- `DELETE /v1/external-medias/<uuid>` - Delete external media

**Group Calls API (`/v1/groupcalls/*`)**:
- `POST /v1/groupcalls` - Create group call
- `GET /v1/groupcalls?<filters>` - List group calls
- `GET /v1/groupcalls/<uuid>` - Get group call details
- `DELETE /v1/groupcalls/<uuid>` - Delete group call
- `POST /v1/groupcalls/<uuid>/hangup` - Hangup all calls in group
- `POST /v1/groupcalls/<uuid>/hangup_groupcall` - Hangup group call
- `POST /v1/groupcalls/<uuid>/hangup_call` - Hangup specific call in group

**Recovery API**:
- `POST /v1/recovery` - Recover call state from Homer SIP capture

### Event Subscriptions

SubscribeHandler subscribes to these RabbitMQ queues:
- **asterisk.all.event**: All Asterisk ARI events (channel/bridge state changes, DTMF, playback, recording)
- **bin-manager.customer-manager.event**: Customer lifecycle events
- **bin-manager.flow-manager.event**: Flow execution events
- **bin-manager.sentinel-manager.event**: Pod lifecycle events

Processes events including:
- **Channel Events**: StasisStart, StasisEnd, ChannelStateChange, ChannelDestroyed, ChannelDtmfReceived, ChannelHangupRequest
- **Bridge Events**: BridgeCreated, BridgeDestroyed, ChannelEnteredBridge, ChannelLeftBridge
- **Recording Events**: RecordingStarted, RecordingFinished, RecordingFailed
- **Playback Events**: PlaybackStarted, PlaybackFinished
- **Contact Events**: ContactStatusChange (SIP registration changes)

### Configuration

Uses **Viper + pflag** pattern (see `cmd/call-manager/init.go:47-180`):
- Command-line flags and environment variables (e.g., `--rabbitmq_address` or `RABBITMQ_ADDRESS`)
- Configuration parameters:
  - `database_dsn`: MySQL connection string
  - `rabbitmq_address`: RabbitMQ server address
  - `redis_address`, `redis_password`, `redis_database`: Redis connection
  - `prometheus_endpoint`, `prometheus_listen_address`: Metrics endpoint
  - `homer_api_address`, `homer_auth_token`, `homer_whitelist`: Homer SIP capture integration (optional)

## Common Commands

### Build
```bash
# Build from service directory
go build -o bin/call-manager ./cmd/call-manager

# Build with vendor (requires GOPRIVATE and git config for private repos)
export GOPRIVATE="gitlab.com/voipbin"
go mod vendor
go build ./cmd/...

# Build using Docker (from monorepo root)
docker build -t call-manager:latest -f bin-call-manager/Dockerfile .
```

### Test
```bash
# Run all tests with coverage
go test -coverprofile cp.out -v $(go list ./...)
go tool cover -html=cp.out -o cp.html
go tool cover -func=cp.out

# Run tests for specific package
go test -v ./pkg/callhandler/...

# Run single test
go test -v ./pkg/callhandler -run Test_Delete
```

### Generate Mocks
```bash
# Generate all mocks (uses go:generate directives)
go generate ./...

# Mocks are created via mockgen for interfaces in:
# - pkg/arieventhandler/main.go -> mock_main.go
# - pkg/callhandler/... (various files)
# - pkg/cachehandler/main.go -> mock_main.go
```

### Lint
```bash
# Run vet
go vet $(go list ./...)

# Run golangci-lint (if available)
golangci-lint run -v --timeout 5m
```

### Run Locally
```bash
# With environment variables
DATABASE_DSN="user:pass@tcp(127.0.0.1:3306)/bin_manager" \
RABBITMQ_ADDRESS="amqp://guest:guest@localhost:5672" \
REDIS_ADDRESS="127.0.0.1:6379" \
REDIS_DATABASE=1 \
PROMETHEUS_ENDPOINT="/metrics" \
PROMETHEUS_LISTEN_ADDRESS=":2112" \
./bin/call-manager

# Or with flags
./bin/call-manager \
  --database_dsn "user:pass@tcp(127.0.0.1:3306)/bin_manager" \
  --rabbitmq_address "amqp://guest:guest@localhost:5672" \
  --redis_address "127.0.0.1:6379" \
  --redis_database 1
```

## Monorepo Context

This service depends on local monorepo packages (see `go.mod` replace directives):
- `monorepo/bin-common-handler`: Shared handlers (sockhandler, requesthandler, notifyhandler, databasehandler, utilhandler)
- `monorepo/bin-customer-manager`: Customer event models
- `monorepo/bin-flow-manager`: Flow event models
- `monorepo/bin-sentinel-manager`: Pod event models

**Important**: The monorepo uses a complex dependency structure where services depend on each other. Always run `go mod vendor` after changing dependencies.

## Testing Patterns

Tests use **gomock** (go.uber.org/mock):
- Mock interfaces generated in same package as interface definition
- Table-driven tests with struct slices defining test cases
- Context passed to all handler methods
- Tests co-located with implementation: `<package>/<feature>_test.go`

Example test structure:
```go
tests := []struct {
    name string
    id   uuid.UUID
    responseCall *call.Call
    expectRes    *call.Call
}{
    {"normal", uuid.FromStringOrNil("..."), ...},
}
for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        mc := gomock.NewController(t)
        defer mc.Finish()
        // Setup mocks and test
    })
}
```

## Key Implementation Details

### Call Status Flow
Calls progress through statuses: `ring` → `up` → `down` → `hangup`

### Confbridge Architecture
Conferences use Asterisk ConfBridge application. Each confbridge tracks:
- Calls (participants) via junction table
- Recording state and recording_id
- External media connections
- Conference flags (mute, hold, etc.)

### ARI Event Processing
The `arieventhandler` processes all Asterisk events:
1. Receives ARI event from RabbitMQ queue `asterisk.all.event`
2. Unmarshals event type and routes to appropriate handler
3. Updates database state (calls, channels, bridges)
4. Publishes notification events via NotifyHandler
5. Triggers callbacks to flow-manager or other services

### Channel/Bridge Lifecycle
- **Channels** represent individual media streams (SIP legs, WebRTC connections)
- **Bridges** connect channels together (mixing bridges for conferences)
- Call-manager tracks both Asterisk objects and business logic objects (Call, Confbridge)

### Database Operations
Unlike other services, this service uses direct SQL queries (not Squirrel):
- Database handler provides CRUD operations
- Soft deletes use `tm_delete` timestamp (`"9999-01-01 00:00:00.000000"` for active records)

### Cache Strategy
Redis cache stores:
- Call details by ID
- Channel details by channel ID
- Bridge details by bridge ID
- Conference details by confbridge ID

Cache is updated on state changes and provides fast lookups for ARI event processing.

### Recovery Handler
The `pkg/callhandler/recovery.go` implements call state recovery from Homer SIP capture system, allowing reconstruction of call state from SIP messages when Asterisk state is lost.

## Asterisk Integration

### ARI (Asterisk REST Interface)
- Service consumes ARI events via RabbitMQ
- Does NOT directly call Asterisk ARI HTTP endpoints (delegated to asterisk-proxy service)
- Tracks Asterisk objects: channels, bridges, playbacks, recordings

### Supported Call Actions
From README.md:
- **answer**: Answer incoming call
- **play**: Play music file (URL)
- **echo**: Echo back sound only
- **stream_echo**: Echo back sound/video/DTMF

## Prometheus Metrics

Service exposes metrics on configured endpoint (default `:2112/metrics`):
- `call_manager_receive_request_process_time` - Histogram of RPC request processing time (labels: type, method)
- `call_manager_subscribe_event_process_time` - Histogram of event processing time (labels: publisher, type)
- `call_manager_ari_event_listen_total` - Counter of ARI events received (labels: type, asterisk_id)
- `call_manager_ari_event_listen_process_time` - Histogram of ARI event processing time (labels: asterisk_id, type)

## Handler Dependency Chain

Understanding the handler initialization chain is critical (see `cmd/call-manager/main.go:96-107`):

```
dbhandler
  ├── channelhandler (depends on: reqHandler, notifyHandler, db)
  ├── bridgehandler (depends on: reqHandler, notifyHandler, db)
  ├── externalMediaHandler (depends on: reqHandler, notifyHandler, db, channelHandler, bridgeHandler)
  ├── recordingHandler (depends on: reqHandler, notifyHandler, db, channelHandler, bridgeHandler)
  ├── confbridgeHandler (depends on: reqHandler, notifyHandler, db, cache, channelHandler, bridgeHandler, recordingHandler, externalMediaHandler)
  ├── groupcallHandler (depends on: reqHandler, notifyHandler, db)
  ├── recoveryHandler (depends on: reqHandler, homerAPI config)
  ├── callHandler (depends on: reqHandler, notifyHandler, db, confbridgeHandler, channelHandler, bridgeHandler, recordingHandler, externalMediaHandler, groupcallHandler, recoveryHandler)
  └── ariEventHandler (depends on: sockHandler, db, cache, reqHandler, notifyHandler, callHandler, confbridgeHandler, channelHandler, bridgeHandler, recordingHandler, externalMediaHandler)
```

When modifying handlers, respect this dependency order to avoid circular dependencies.
