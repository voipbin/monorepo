# Add transport_data to External Media — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Thread a new `transport_data` string field through the entire external media stack, from API callers down to the Asterisk ARI `externalMedia` endpoint.

**Architecture:** Add an optional string field at every layer: model struct, request models, handler interfaces, RPC functions, and ARI call. All existing callers pass `""`. Redis-only storage (no Alembic migration).

**Tech Stack:** Go, RabbitMQ RPC, Redis, Asterisk ARI, OpenAPI

**Worktree:** `/home/pchero/gitvoipbin/monorepo-worktrees/NOJIRA-add-transport-data-external-media`

**Verification command for each service:**
```bash
cd <service-dir> && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

---

### Task 1: Model and Field Constants

Add the `TransportData` field to the ExternalMedia model struct and field constants.

**Files:**
- Modify: `bin-call-manager/models/externalmedia/main.go:24-31`
- Modify: `bin-call-manager/models/externalmedia/field.go:22-28`

**Step 1: Add `TransportData` field to ExternalMedia struct**

In `bin-call-manager/models/externalmedia/main.go`, add the field after `Transport`:

```go
	ExternalHost    string        `json:"external_host"`
	Encapsulation   Encapsulation `json:"encapsulation"` // Payload encapsulation protocol
	Transport       Transport     `json:"transport"`
	TransportData   string        `json:"transport_data,omitempty"` // transport-specific data
	ConnectionType  string        `json:"connection_type"`
	Format          string        `json:"format"`
```

**Step 2: Add `FieldTransportData` constant to field.go**

In `bin-call-manager/models/externalmedia/field.go`, add after `FieldTransport`:

```go
	FieldTransport       Field = "transport"        // transport
	FieldTransportData   Field = "transport_data"   // transport-specific data
	FieldConnectionType  Field = "connection_type"  // connection_type
```

**Step 3: Commit**

```bash
cd /home/pchero/gitvoipbin/monorepo-worktrees/NOJIRA-add-transport-data-external-media
git add bin-call-manager/models/externalmedia/main.go bin-call-manager/models/externalmedia/field.go
git commit -m "NOJIRA-add-transport-data-external-media

- bin-call-manager: Add TransportData field to ExternalMedia model and field constants"
```

---

### Task 2: Request Models

Add `TransportData` to all three request structs.

**Files:**
- Modify: `bin-call-manager/pkg/listenhandler/models/request/externalmedias.go:12-24`
- Modify: `bin-call-manager/pkg/listenhandler/models/request/calls.go:77-86`
- Modify: `bin-call-manager/pkg/listenhandler/models/request/confbridge.go:24-31`

**Step 1: Add to `V1DataExternalMediasPost`**

In `externalmedias.go`, add after `Transport`:

```go
	Transport       string                      `json:"transport,omitempty"`
	TransportData   string                      `json:"transport_data,omitempty"`
	ConnectionType  string                      `json:"connection_type,omitempty"`
```

**Step 2: Add to `V1DataCallsIDExternalMediaPost`**

In `calls.go`, add after `Transport`:

```go
	Transport       string                  `json:"transport,omitempty"`
	TransportData   string                  `json:"transport_data,omitempty"`
	ConnectionType  string                  `json:"connection_type,omitempty"`
```

**Step 3: Add to `V1DataConfbridgesIDExternalMediaPost`**

In `confbridge.go`, add after `Transport`:

```go
	Transport       string    `json:"transport,omitempty"`
	TransportData   string    `json:"transport_data,omitempty"`
	ConnectionType  string    `json:"connection_type,omitempty"`
```

**Step 4: Commit**

```bash
git add bin-call-manager/pkg/listenhandler/models/request/externalmedias.go \
        bin-call-manager/pkg/listenhandler/models/request/calls.go \
        bin-call-manager/pkg/listenhandler/models/request/confbridge.go
git commit -m "NOJIRA-add-transport-data-external-media

- bin-call-manager: Add TransportData to external media request models"
```

---

### Task 3: Flow Action Model and OpenAPI Spec

Add `TransportData` to the flow action option and update the OpenAPI schema.

**Files:**
- Modify: `bin-flow-manager/models/action/option.go:211-220`
- Modify: `bin-openapi-manager/openapi/openapi.yaml:3425-3429`

**Step 1: Add to `OptionExternalMediaStart`**

In `bin-flow-manager/models/action/option.go`, add after `Transport`:

```go
	Transport       string `json:"transport,omitempty"`        // transport. default: udp
	TransportData   string `json:"transport_data,omitempty"`   // transport-specific data
	ConnectionType  string `json:"connection_type,omitempty"`  // connection type. default: client
```

**Step 2: Update OpenAPI spec**

In `bin-openapi-manager/openapi/openapi.yaml`, add `transport_data` field to `FlowManagerActionOptionExternalMediaStart` after the `data` property (before the blank line at 3429):

```yaml
        data:
          type: string
          description: Optional data to pass to the external media endpoint.
          example: ""
        transport_data:
          type: string
          description: Transport-specific data. For websocket, this is appended to the dialstring.
          example: ""
```

**Step 3: Regenerate OpenAPI types and API server code**

```bash
cd /home/pchero/gitvoipbin/monorepo-worktrees/NOJIRA-add-transport-data-external-media/bin-openapi-manager && go generate ./...
cd /home/pchero/gitvoipbin/monorepo-worktrees/NOJIRA-add-transport-data-external-media/bin-api-manager && go generate ./...
```

**Step 4: Commit**

```bash
cd /home/pchero/gitvoipbin/monorepo-worktrees/NOJIRA-add-transport-data-external-media
git add bin-flow-manager/models/action/option.go \
        bin-openapi-manager/openapi/openapi.yaml \
        bin-openapi-manager/gens/ \
        bin-api-manager/gens/
git commit -m "NOJIRA-add-transport-data-external-media

- bin-flow-manager: Add TransportData to OptionExternalMediaStart
- bin-openapi-manager: Add transport_data field to FlowManagerActionOptionExternalMediaStart schema
- bin-api-manager: Regenerate server code from updated OpenAPI spec"
```

---

### Task 4: bin-common-handler — ARI and RPC Interfaces + Implementations

Update all 4 interface methods and their implementations in bin-common-handler.

**Files:**
- Modify: `bin-common-handler/pkg/requesthandler/main.go` (4 interface signatures)
- Modify: `bin-common-handler/pkg/requesthandler/ast_channel.go:393-438`
- Modify: `bin-common-handler/pkg/requesthandler/call_externalmedias.go:63-107`
- Modify: `bin-common-handler/pkg/requesthandler/call_calls.go:323-362`
- Modify: `bin-common-handler/pkg/requesthandler/call_confbridge.go:122-155`

**Step 1: Update `AstChannelExternalMedia` in `main.go` interface**

Find the line:
```go
AstChannelExternalMedia(ctx context.Context, asteriskID string, channelID string, externalHost string, encapsulation string, transport string, connectionType string, format string, direction string, data string, variables map[string]string) (*cmchannel.Channel, error)
```

Add `transportData string` after `transport string`:
```go
AstChannelExternalMedia(ctx context.Context, asteriskID string, channelID string, externalHost string, encapsulation string, transport string, transportData string, connectionType string, format string, direction string, data string, variables map[string]string) (*cmchannel.Channel, error)
```

**Step 2: Update `AstChannelExternalMedia` implementation in `ast_channel.go`**

Update the function signature to add `transportData string` after `transport string`.

Add `TransportData` to the Data struct:
```go
type Data struct {
    ChannelID      string            `json:"channel_id"`
    App            string            `json:"app"`
    ExternalHost   string            `json:"external_host"`
    Encapsulation  string            `json:"encapsulation,omitempty"`
    Transport      string            `json:"transport,omitempty"`
    TransportData  string            `json:"transport_data,omitempty"`
    ConnectionType string            `json:"connection_type,omitempty"`
    Format         string            `json:"format"`
    Direction      string            `json:"direction,omitempty"`
    Data           string            `json:"data,omitempty"`
    Variables      map[string]string `json:"variables,omitempty"`
}
```

Add `TransportData: transportData,` to the marshal call.

**Step 3: Update `CallV1ExternalMediaStart` in `main.go` interface**

Find the interface method (around line 522-533) and add `transportData string` after `transport string`:
```go
CallV1ExternalMediaStart(
    ctx context.Context,
    externalMediaID uuid.UUID,
    referenceType cmexternalmedia.ReferenceType,
    referenceID uuid.UUID,
    externalHost string,
    encapsulation string,
    transport string,
    transportData string,
    connectionType string,
    format string,
    directionListen cmexternalmedia.Direction,
    directionSpeak cmexternalmedia.Direction,
) (*cmexternalmedia.ExternalMedia, error)
```

**Step 4: Update `CallV1ExternalMediaStart` implementation in `call_externalmedias.go`**

Update function signature to add `transportData string` after `transport string`.

Add `TransportData` to the request struct marshaling:
```go
reqData := &cmrequest.V1DataExternalMediasPost{
    ID:              externalMediaID,
    ReferenceType:   referenceType,
    ReferenceID:     referenceID,
    ExternalHost:    externalHost,
    Encapsulation:   encapsulation,
    Transport:       transport,
    TransportData:   transportData,
    ConnectionType:  connectionType,
    Format:          format,
    DirectionListen: directionListen,
    DirectionSpeak:  directionSpeak,
}
```

**Step 5: Update `CallV1CallExternalMediaStart` in `main.go` interface**

Find the interface method and add `transportData string` after `transport string`.

**Step 6: Update `CallV1CallExternalMediaStart` implementation in `call_calls.go`**

Update function signature. Add `TransportData: transportData,` to the marshaled struct.

**Step 7: Update `CallV1ConfbridgeExternalMediaStart` in `main.go` interface**

Find the interface method (around line 507-516) and add `transportData string` after `transport string`.

**Step 8: Update `CallV1ConfbridgeExternalMediaStart` implementation in `call_confbridge.go`**

Update function signature. Add `TransportData: transportData,` to the marshaled struct.

**Step 9: Regenerate mocks**

```bash
cd /home/pchero/gitvoipbin/monorepo-worktrees/NOJIRA-add-transport-data-external-media/bin-common-handler && go generate ./...
```

**Step 10: Update tests**

Update test files to include `transportData` parameter in all call sites:
- `bin-common-handler/pkg/requesthandler/ast_channel_test.go`
- `bin-common-handler/pkg/requesthandler/call_externalmedias_test.go`
- `bin-common-handler/pkg/requesthandler/call_calls_test.go`
- `bin-common-handler/pkg/requesthandler/call_confbridge_test.go`

In each test, add `""` (empty string) for `transportData` at the matching position in function calls and expected request structs.

**Step 11: Verify bin-common-handler**

```bash
cd /home/pchero/gitvoipbin/monorepo-worktrees/NOJIRA-add-transport-data-external-media/bin-common-handler && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 12: Commit**

```bash
cd /home/pchero/gitvoipbin/monorepo-worktrees/NOJIRA-add-transport-data-external-media
git add bin-common-handler/
git commit -m "NOJIRA-add-transport-data-external-media

- bin-common-handler: Add transportData param to AstChannelExternalMedia
- bin-common-handler: Add transportData param to CallV1ExternalMediaStart
- bin-common-handler: Add transportData param to CallV1CallExternalMediaStart
- bin-common-handler: Add transportData param to CallV1ConfbridgeExternalMediaStart
- bin-common-handler: Regenerate mocks and update tests"
```

---

### Task 5: bin-call-manager — Internal Handler Chain

Thread `transportData` through all internal handlers in bin-call-manager.

**Files:**
- Modify: `bin-call-manager/pkg/externalmediahandler/main.go:30-42` (interface)
- Modify: `bin-call-manager/pkg/externalmediahandler/start.go` (4 functions)
- Modify: `bin-call-manager/pkg/externalmediahandler/db.go:13-31` (Create function)
- Modify: `bin-call-manager/pkg/callhandler/main.go` (ExternalMediaStart interface)
- Modify: `bin-call-manager/pkg/callhandler/external_media.go:23-34` (ExternalMediaStart impl)
- Modify: `bin-call-manager/pkg/callhandler/action.go:663-689` (actionExecuteExternalMediaStart)
- Modify: `bin-call-manager/pkg/confbridgehandler/main.go` (ExternalMediaStart interface)
- Modify: `bin-call-manager/pkg/confbridgehandler/external_media.go:17-26` (ExternalMediaStart impl)
- Modify: `bin-call-manager/pkg/channelhandler/main.go:73` (StartExternalMedia interface)
- Modify: `bin-call-manager/pkg/channelhandler/start.go:45-67` (StartExternalMedia impl)
- Modify: `bin-call-manager/pkg/listenhandler/v1_external_medias.go:85-97`
- Modify: `bin-call-manager/pkg/listenhandler/v1_calls.go:483-494`
- Modify: `bin-call-manager/pkg/listenhandler/v1_confbridges.go:225-234`

**Step 1: Update `ExternalMediaHandler.Start()` interface in `externalmediahandler/main.go`**

Add `transportData string` after `transport externalmedia.Transport`:
```go
Start(
    ctx context.Context,
    id uuid.UUID,
    referenceType externalmedia.ReferenceType,
    referenceID uuid.UUID,
    externalHost string,
    encapsulation externalmedia.Encapsulation,
    transport externalmedia.Transport,
    transportData string,
    connectionType string,
    format string,
    directionListen externalmedia.Direction,
    directionSpeak externalmedia.Direction,
) (*externalmedia.ExternalMedia, error)
```

**Step 2: Update `Start()` implementation and sub-functions in `externalmediahandler/start.go`**

Add `transportData string` param to `Start()`, `startReferenceTypeCall()`, `startReferenceTypeConfbridge()`, and `startExternalMedia()`.

Thread it through every call chain:

In `Start()`:
```go
case externalmedia.ReferenceTypeCall:
    return h.startReferenceTypeCall(ctx, id, referenceID, externalHost, encapsulation, transport, transportData, format, directionListen, directionSpeak)

case externalmedia.ReferenceTypeConfbridge:
    return h.startReferenceTypeConfbridge(ctx, id, referenceID, externalHost, encapsulation, transport, transportData, format)
```

In `startReferenceTypeCall()` — pass to `startExternalMedia()`:
```go
res, err := h.startExternalMedia(
    ctx,
    id,
    ch.AsteriskID,
    br.ID,
    playbackID,
    externalmedia.ReferenceTypeCall,
    c.ID,
    externalHost,
    encapsulation,
    transport,
    transportData,
    format,
    directionListen,
    directionSpeak,
)
```

Same pattern in `startReferenceTypeConfbridge()`.

In `startExternalMedia()` — pass to `Create()` and `StartExternalMedia()`:
```go
em, err := h.Create(
    ctx,
    id,
    asteriskID,
    extChannelID,
    bridgeID,
    playbackID,
    referenceType,
    referenceID,
    "",
    0,
    externalHost,
    encapsulation,
    transport,
    transportData,
    defaultConnectionType,
    format,
    directionListen,
    directionSpeak,
)
```

```go
extCh, err := h.channelHandler.StartExternalMedia(
    ctx,
    asteriskID,
    extChannelID,
    externalHost,
    string(encapsulation),
    string(transport),
    transportData,
    defaultConnectionType,
    format,
    defaultDirection,
    chData,
    nil,
)
```

**Step 3: Update `Create()` in `externalmediahandler/db.go`**

Add `transportData string` param after `transport`. Set on struct:
```go
extMedia := &externalmedia.ExternalMedia{
    ...
    Transport:       transport,
    TransportData:   transportData,
    ConnectionType:  connectionType,
    ...
}
```

**Step 4: Update `ChannelHandler.StartExternalMedia()` interface in `channelhandler/main.go`**

Add `transportData string` after `transport string`:
```go
StartExternalMedia(ctx context.Context, asteriskID string, id string, externalHost string, encapsulation string, transport string, transportData string, connectionType string, format string, direction string, data string, variables map[string]string) (*channel.Channel, error)
```

**Step 5: Update `StartExternalMedia()` implementation in `channelhandler/start.go`**

Add `transportData string` param. Pass to `AstChannelExternalMedia`:
```go
res, err := h.reqHandler.AstChannelExternalMedia(ctx, asteriskID, id, externalHost, encapsulation, transport, transportData, connectionType, format, direction, data, variables)
```

**Step 6: Update `CallHandler.ExternalMediaStart()` interface in `callhandler/main.go`**

Add `transportData string` after `transport externalmedia.Transport`.

**Step 7: Update `ExternalMediaStart()` impl in `callhandler/external_media.go`**

Add param, thread to `externalMediaHandler.Start()`:
```go
tmp, err := h.externalMediaHandler.Start(
    ctx,
    externalMediaID,
    externalmedia.ReferenceTypeCall,
    c.ID,
    externalHost,
    encapsulation,
    transport,
    transportData,
    connectionType,
    format,
    directionListen,
    directionSpeak,
)
```

**Step 8: Update `actionExecuteExternalMediaStart()` in `callhandler/action.go`**

Pass `option.TransportData`:
```go
cc, err := h.ExternalMediaStart(
    ctx,
    c.ID,
    uuid.Nil,
    option.ExternalHost,
    externalmedia.Encapsulation(option.Encapsulation),
    externalmedia.Transport(option.Transport),
    option.TransportData,
    option.ConnectionType,
    option.Format,
    externalmedia.Direction(option.DirectionListen),
    externalmedia.Direction(option.DirectionSpeak),
)
```

**Step 9: Update `ConfbridgeHandler.ExternalMediaStart()` interface in `confbridgehandler/main.go`**

Add `transportData string` after `transport externalmedia.Transport`.

**Step 10: Update `ExternalMediaStart()` impl in `confbridgehandler/external_media.go`**

Add param, thread to `externalMediaHandler.Start()`:
```go
tmp, err := h.externalMediaHandler.Start(
    ctx,
    externalMediaID,
    externalmedia.ReferenceTypeConfbridge,
    c.ID,
    externalHost,
    encapsulation,
    transport,
    transportData,
    connectionType,
    format,
    externalmedia.DirectionBoth,
    externalmedia.DirectionBoth,
)
```

**Step 11: Update listenhandler call sites**

In `listenhandler/v1_external_medias.go` `processV1ExternalMediasPost()`:
```go
tmp, err := h.externalMediaHandler.Start(
    ctx,
    req.ID,
    req.ReferenceType,
    req.ReferenceID,
    req.ExternalHost,
    externalmedia.Encapsulation(req.Encapsulation),
    externalmedia.Transport(req.Transport),
    req.TransportData,
    req.ConnectionType,
    req.Format,
    req.DirectionListen,
    req.DirectionSpeak,
)
```

In `listenhandler/v1_calls.go` `processV1CallsIDExternalMediaPost()`:
```go
tmp, err := h.callHandler.ExternalMediaStart(
    ctx,
    id,
    req.ExternalMediaID,
    req.ExternalHost,
    externalmedia.Encapsulation(req.Encapsulation),
    externalmedia.Transport(req.Transport),
    req.TransportData,
    req.ConnectionType,
    req.Format,
    req.DirectionListen,
    req.DirectionSpeak,
)
```

In `listenhandler/v1_confbridges.go` `processV1ConfbridgesIDExternalMediaPost()`:
```go
tmp, err := h.confbridgeHandler.ExternalMediaStart(
    ctx,
    id,
    req.ExternalMediaID,
    req.ExternalHost,
    externalmedia.Encapsulation(req.Encapsulation),
    externalmedia.Transport(req.Transport),
    req.TransportData,
    req.ConnectionType,
    req.Format,
)
```

**Step 12: Regenerate mocks**

```bash
cd /home/pchero/gitvoipbin/monorepo-worktrees/NOJIRA-add-transport-data-external-media/bin-call-manager && go generate ./...
```

**Step 13: Update all tests**

Add `""` or appropriate `transportData` parameter to all mock expectations and function calls in:
- `bin-call-manager/pkg/externalmediahandler/db_test.go`
- `bin-call-manager/pkg/externalmediahandler/start_test.go` (if exists)
- `bin-call-manager/pkg/channelhandler/start_test.go`
- `bin-call-manager/pkg/listenhandler/v1_external_medias_test.go`
- `bin-call-manager/pkg/listenhandler/v1_calls_test.go`
- `bin-call-manager/pkg/listenhandler/v1_confbridge_test.go`
- `bin-call-manager/pkg/callhandler/external_media_test.go` (if exists)
- `bin-call-manager/pkg/confbridgehandler/external_media_test.go` (if exists)

For each test, search for existing calls to the modified functions and add the `transportData` parameter (usually `""`) in the correct position (after `transport`).

**Step 14: Verify bin-call-manager**

```bash
cd /home/pchero/gitvoipbin/monorepo-worktrees/NOJIRA-add-transport-data-external-media/bin-call-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 15: Commit**

```bash
cd /home/pchero/gitvoipbin/monorepo-worktrees/NOJIRA-add-transport-data-external-media
git add bin-call-manager/
git commit -m "NOJIRA-add-transport-data-external-media

- bin-call-manager: Thread transportData through all external media handler chain
- bin-call-manager: Update ExternalMediaHandler, CallHandler, ConfbridgeHandler, ChannelHandler interfaces
- bin-call-manager: Update listenhandler to pass transportData from request models
- bin-call-manager: Regenerate mocks and update all tests"
```

---

### Task 6: External Caller Services

Update the 5 external services that call `CallV1ExternalMediaStart` to pass `""` for `transportData`.

**Files:**
- Modify: `bin-api-manager/pkg/streamhandler/start.go` (1 call site)
- Modify: `bin-tts-manager/pkg/streaminghandler/start.go` (1 call site)
- Modify: `bin-transcribe-manager/pkg/streaminghandler/start.go` (1 call site)
- Modify: `bin-pipecat-manager/pkg/pipecatcallhandler/start.go` (2 call sites)

**Step 1: Update each caller**

In every `CallV1ExternalMediaStart(` call, add `""` after the `transport` parameter (e.g., after `defaultTransport,`):

```go
em, err := h.reqHandler.CallV1ExternalMediaStart(
    ctx,
    ...,
    defaultTransport,
    "", // transportData
    defaultConnectionType,
    ...
)
```

Do this for all 5 call sites across the 4 services.

**Step 2: Update tests in each service**

Add `""` to mock expectations of `CallV1ExternalMediaStart` in:
- `bin-api-manager/pkg/streamhandler/start_test.go`
- `bin-tts-manager/pkg/streaminghandler/start_test.go`
- `bin-transcribe-manager/pkg/streaminghandler/start_test.go`
- `bin-pipecat-manager/pkg/pipecatcallhandler/start_test.go` (if exists)

**Step 3: Verify each service** (run in parallel or sequentially)

```bash
cd /home/pchero/gitvoipbin/monorepo-worktrees/NOJIRA-add-transport-data-external-media/bin-api-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

```bash
cd /home/pchero/gitvoipbin/monorepo-worktrees/NOJIRA-add-transport-data-external-media/bin-tts-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

```bash
cd /home/pchero/gitvoipbin/monorepo-worktrees/NOJIRA-add-transport-data-external-media/bin-transcribe-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

```bash
cd /home/pchero/gitvoipbin/monorepo-worktrees/NOJIRA-add-transport-data-external-media/bin-pipecat-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 4: Verify remaining affected services** (flow-manager, openapi-manager)

```bash
cd /home/pchero/gitvoipbin/monorepo-worktrees/NOJIRA-add-transport-data-external-media/bin-flow-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

```bash
cd /home/pchero/gitvoipbin/monorepo-worktrees/NOJIRA-add-transport-data-external-media/bin-openapi-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 5: Commit**

```bash
cd /home/pchero/gitvoipbin/monorepo-worktrees/NOJIRA-add-transport-data-external-media
git add bin-api-manager/ bin-tts-manager/ bin-transcribe-manager/ bin-pipecat-manager/ bin-flow-manager/ bin-openapi-manager/
git commit -m "NOJIRA-add-transport-data-external-media

- bin-api-manager: Pass empty transportData to CallV1ExternalMediaStart
- bin-tts-manager: Pass empty transportData to CallV1ExternalMediaStart
- bin-transcribe-manager: Pass empty transportData to CallV1ExternalMediaStart
- bin-pipecat-manager: Pass empty transportData to CallV1ExternalMediaStart
- bin-flow-manager: Vendor update for TransportData field
- bin-openapi-manager: Vendor update for TransportData field"
```

---

### Task 7: Final Verification and PR

Run full verification on all 8 affected services and create PR.

**Step 1: Final check — fetch latest main and check for conflicts**

```bash
cd /home/pchero/gitvoipbin/monorepo-worktrees/NOJIRA-add-transport-data-external-media
git fetch origin main
git merge-tree $(git merge-base HEAD origin/main) HEAD origin/main | grep -E "^(CONFLICT|changed in both)"
git log --oneline HEAD..origin/main
```

**Step 2: If conflicts exist, rebase and re-verify**

```bash
git rebase origin/main
# Resolve conflicts if any, then re-run verification for affected services
```

**Step 3: Push and create PR**

```bash
git push -u origin NOJIRA-add-transport-data-external-media
```

```bash
gh pr create --title "NOJIRA-add-transport-data-external-media" --body "$(cat <<'EOF'
Add transport_data field support to the external media stack, threading it from
API callers through inter-service RPC down to the Asterisk ARI externalMedia endpoint.
This enables upcoming websocket transport support (chan_websocket integration).

- bin-call-manager: Add TransportData field to ExternalMedia model
- bin-call-manager: Thread transportData through all handler interfaces and implementations
- bin-call-manager: Update request models for all external media endpoints
- bin-common-handler: Add transportData param to AstChannelExternalMedia ARI call
- bin-common-handler: Add transportData param to CallV1ExternalMediaStart RPC
- bin-common-handler: Add transportData param to CallV1CallExternalMediaStart RPC
- bin-common-handler: Add transportData param to CallV1ConfbridgeExternalMediaStart RPC
- bin-api-manager: Pass empty transportData to CallV1ExternalMediaStart
- bin-tts-manager: Pass empty transportData to CallV1ExternalMediaStart
- bin-transcribe-manager: Pass empty transportData to CallV1ExternalMediaStart
- bin-pipecat-manager: Pass empty transportData to CallV1ExternalMediaStart
- bin-flow-manager: Add TransportData to OptionExternalMediaStart
- bin-openapi-manager: Add transport_data to FlowManagerActionOptionExternalMediaStart schema
EOF
)"
```
