# tts-manager WebSocket Streaming Migration — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace AudioSocket with WebSocket (chan_websocket INCOMING mode) in tts-manager and implement real SayFlush.

**Architecture:** tts-manager becomes a WebSocket client connecting TO Asterisk. call-manager constructs the `ws://` media URI from Redis-cached Asterisk IP + configurable port + connection ID returned by ARI. GCP TTS switches to MULAW encoding. Per-stream sub-context enables instant SayFlush without disconnecting the session.

**Tech Stack:** Go, gorilla/websocket (already in tts-manager go.mod), Asterisk chan_websocket, GCP StreamingSynthesize with MULAW

**Design document:** `docs/plans/2026-02-20-tts-manager-websocket-streaming-design.md`

---

## Task 1: bin-call-manager — ExternalMedia model + constants

Add `TransportWebsocket`, `EncapsulationNone` constants and `MediaURI` field to the ExternalMedia model.

**Files:**
- Modify: `bin-call-manager/models/externalmedia/main.go:47-59` (constants + struct)

**Step 1: Add constants and field**

In `bin-call-manager/models/externalmedia/main.go`:

Add to Encapsulation constants (after line 49):
```go
EncapsulationNone        Encapsulation = "none"
```

Add to Transport constants (after line 58):
```go
TransportWebsocket Transport = "websocket"
```

Add `MediaURI` field to ExternalMedia struct (after `TransportData` line 27):
```go
MediaURI        string        `json:"media_uri,omitempty"`
```

**Step 2: Verify it compiles**

Run: `cd bin-call-manager && go build ./...`
Expected: SUCCESS

**Step 3: Commit**

```
git add bin-call-manager/models/externalmedia/main.go
git commit -m "NOJIRA-tts-manager-websocket-streaming-design

- bin-call-manager: Add TransportWebsocket, EncapsulationNone constants and MediaURI field"
```

---

## Task 2: bin-call-manager — Config + handler constructor changes

Add `AsteriskWSPort` config and pass `cache` + `asteriskWSPort` to `NewExternalMediaHandler()`.

**Files:**
- Modify: `bin-call-manager/internal/config/main.go:21-32` (Config struct)
- Modify: `bin-call-manager/internal/config/main.go:45-84` (bindConfig + LoadGlobalConfig)
- Modify: `bin-call-manager/pkg/externalmediahandler/main.go:50-53` (constants)
- Modify: `bin-call-manager/pkg/externalmediahandler/main.go:95-124` (struct + constructor)
- Modify: `bin-call-manager/cmd/call-manager/main.go:144` (handler init call)

**Step 1: Add AsteriskWSPort to Config**

In `bin-call-manager/internal/config/main.go`:

Add to Config struct (after `HomerWhitelist` line 31):
```go
AsteriskWSPort int // AsteriskWSPort is the Asterisk HTTP server port for WebSocket media connections.
```

Add flag in `bindConfig()` (after the homer_whitelist flag, before `bindings := ...`):
```go
f.Int("asterisk_ws_port", 8088, "Asterisk WebSocket media port")
```

Add to bindings map:
```go
"asterisk_ws_port": "ASTERISK_WS_PORT",
```

Add to `LoadGlobalConfig()` in the Config struct literal:
```go
AsteriskWSPort: viper.GetInt("asterisk_ws_port"),
```

**Step 2: Add cache and port to externalmediahandler struct**

In `bin-call-manager/pkg/externalmediahandler/main.go`:

Add constant (after line 53):
```go
ChannelVariableWebSocketConnectionID = "MEDIA_WEBSOCKET_CONNECTION_ID"
```

Add fields to `externalMediaHandler` struct (after `bridgeHandler` line 102):
```go
cache          cachehandler.CacheHandler
asteriskWSPort int
```

Add import for cachehandler:
```go
"monorepo/bin-call-manager/pkg/cachehandler"
```

Update `NewExternalMediaHandler()` signature and body:
```go
func NewExternalMediaHandler(
	requestHandler requesthandler.RequestHandler,
	notifyHandler notifyhandler.NotifyHandler,
	db dbhandler.DBHandler,
	channelHandler channelhandler.ChannelHandler,
	bridgeHandler bridgehandler.BridgeHandler,
	cache cachehandler.CacheHandler,
	asteriskWSPort int,
) ExternalMediaHandler {

	h := &externalMediaHandler{
		utilHandler:    utilhandler.NewUtilHandler(),
		reqHandler:     requestHandler,
		notifyHandler:  notifyHandler,
		db:             db,
		channelHandler: channelHandler,
		bridgeHandler:  bridgeHandler,
		cache:          cache,
		asteriskWSPort: asteriskWSPort,
	}

	return h
}
```

**Step 3: Update call site in main.go**

In `bin-call-manager/cmd/call-manager/main.go` line 144, change:
```go
externalMediaHandler := externalmediahandler.NewExternalMediaHandler(reqHandler, notifyHandler, db, channelHandler, bridgeHandler)
```
to:
```go
externalMediaHandler := externalmediahandler.NewExternalMediaHandler(reqHandler, notifyHandler, db, channelHandler, bridgeHandler, cache, cfg.AsteriskWSPort)
```

**Step 4: Regenerate mocks and fix tests**

Run: `cd bin-call-manager && go generate ./...`

Then update any test files that construct `externalMediaHandler` structs directly. The tests in `start_test.go` and `db_test.go` construct the handler as `&externalMediaHandler{...}` — add `cache` and `asteriskWSPort` fields.

Search pattern: `grep -rn "externalMediaHandler{" bin-call-manager/pkg/externalmediahandler/`

For each test file that constructs the struct, add:
```go
cache:          mockCache,  // if mock exists, or nil
asteriskWSPort: 8088,
```

If tests don't use cache, pass `nil` (the field won't be called in non-WebSocket paths).

**Step 5: Verify**

Run: `cd bin-call-manager && go build ./... && go test ./...`
Expected: All pass

**Step 6: Commit**

```
git add bin-call-manager/
git commit -m "NOJIRA-tts-manager-websocket-streaming-design

- bin-call-manager: Add AsteriskWSPort config (default 8088, env ASTERISK_WS_PORT)
- bin-call-manager: Add cache and asteriskWSPort to externalmediahandler constructor"
```

---

## Task 3: bin-call-manager — Fix connectionType bug + WebSocket branch in start.go

Fix the pre-existing `connectionType` bug and add WebSocket media URI construction.

**Files:**
- Modify: `bin-call-manager/pkg/externalmediahandler/start.go:254-336`
- Modify: `bin-call-manager/pkg/externalmediahandler/db.go:36-59` (add MediaURI to Create)

**Step 1: Add MediaURI parameter to Create()**

In `bin-call-manager/pkg/externalmediahandler/db.go`, add `mediaURI string` parameter to `Create()` and set it on the struct:

Add parameter after `directionSpeak`:
```go
func (h *externalMediaHandler) Create(
	ctx context.Context,
	id uuid.UUID,
	asteriskID string,
	channelID string,
	bridgeID string,
	playbackID string,
	referenceType externalmedia.ReferenceType,
	referenceID uuid.UUID,
	localIP string,
	localPort int,
	externalHost string,
	encapsulation externalmedia.Encapsulation,
	transport externalmedia.Transport,
	transportData string,
	connectionType string,
	format string,
	directionListen externalmedia.Direction,
	directionSpeak externalmedia.Direction,
	mediaURI string,
) (*externalmedia.ExternalMedia, error) {
```

Add to the struct literal:
```go
MediaURI:        mediaURI,
```

**Step 2: Fix connectionType bug + add WebSocket branch in start.go**

In `bin-call-manager/pkg/externalmediahandler/start.go`, in `startExternalMedia()`:

Replace line 289 (`defaultConnectionType`) with `connectionType`:
```go
connectionType,
```

Replace line 308 (`defaultConnectionType`) with `connectionType`:
```go
connectionType,
```

Add `connectionType` parameter to `startExternalMedia()` signature (add after `transportData`):
```go
func (h *externalMediaHandler) startExternalMedia(
	ctx context.Context,
	id uuid.UUID,
	asteriskID string,
	bridgeID string,
	playbackID string,
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
) (*externalmedia.ExternalMedia, error) {
```

Add default for connectionType (after the format default):
```go
if connectionType == "" {
	connectionType = defaultConnectionType
}
```

Update `Create()` call to pass empty `mediaURI` initially:
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
	connectionType,
	format,
	directionListen,
	directionSpeak,
	"", // mediaURI — populated below for WebSocket transport
)
```

After the `UpdateLocalAddress` call (line 329-333), add the WebSocket branch:
```go
// For WebSocket transport, construct the media URI from Asterisk's internal address
if transport == externalmedia.TransportWebsocket {
	connectionID, ok := extCh.Data[ChannelVariableWebSocketConnectionID].(string)
	if !ok || connectionID == "" {
		return nil, fmt.Errorf("could not get WebSocket connection ID from channel variables")
	}

	asteriskIP, errCache := h.cache.AsteriskAddressInternalGet(ctx, asteriskID)
	if errCache != nil {
		return nil, errors.Wrapf(errCache, "could not get asterisk internal address. asterisk_id: %s", asteriskID)
	}

	mediaURI := fmt.Sprintf("ws://%s:%d/media/%s", asteriskIP, h.asteriskWSPort, connectionID)
	res.MediaURI = mediaURI

	// persist the updated MediaURI
	if errDB := h.db.ExternalMediaSet(ctx, res); errDB != nil {
		return nil, errors.Wrapf(errDB, "could not update external media with media URI")
	}
}
```

Thread `connectionType` through the two callers:

In `startReferenceTypeCall()` (line 128), add `connectionType` parameter to the call. But wait — this function doesn't receive `connectionType`. We need to add it.

Actually, looking at the code more carefully: `Start()` receives `connectionType` but the intermediate functions `startReferenceTypeCall()` and `startReferenceTypeConfbridge()` do NOT pass it through. This is the bug. Fix:

Add `connectionType string` parameter to `startReferenceTypeCall()` and `startReferenceTypeConfbridge()`:

In `startReferenceTypeCall()` signature, add after `transportData`:
```go
connectionType string,
```

In `startReferenceTypeConfbridge()` signature, add after `transportData`:
```go
connectionType string,
```

Thread it through to `startExternalMedia()` calls inside both functions.

Update `Start()` to pass `connectionType` to both callers (lines 49 and 52):
```go
case externalmedia.ReferenceTypeCall:
	return h.startReferenceTypeCall(ctx, id, referenceID, externalHost, encapsulation, transport, transportData, connectionType, format, directionListen, directionSpeak)

case externalmedia.ReferenceTypeConfbridge:
	return h.startReferenceTypeConfbridge(ctx, id, referenceID, externalHost, encapsulation, transport, transportData, connectionType, format)
```

**Step 3: Verify**

Run: `cd bin-call-manager && go generate ./... && go build ./... && go test ./...`
Expected: All pass

**Step 4: Commit**

```
git add bin-call-manager/
git commit -m "NOJIRA-tts-manager-websocket-streaming-design

- bin-call-manager: Fix connectionType parameter threading (was silently dropped)
- bin-call-manager: Add WebSocket media URI construction in startExternalMedia
- bin-call-manager: Add MediaURI parameter to Create()"
```

---

## Task 4: bin-call-manager — Full verification

**Step 1: Run full verification workflow**

```bash
cd bin-call-manager && \
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```
Expected: All pass

**Step 2: Commit if any generated files changed**

---

## Task 5: bin-openapi-manager — Add WebSocket transport fields to OpenAPI spec

**Files:**
- Modify: `bin-openapi-manager/openapi/openapi.yaml`

**Step 1: Read CLAUDE.md for openapi-manager**

Read `bin-openapi-manager/CLAUDE.md` for AI-Native specification rules before modifying.

**Step 2: Add media_uri to ExternalMedia schema**

Search for the `CallManagerExternalMedia` schema in the OpenAPI YAML. Add:
```yaml
media_uri:
  type: string
  description: WebSocket media URI for connecting to Asterisk. Present only when transport is websocket.
  example: "ws://10.0.1.5:8088/media/abc123"
```

The transport and encapsulation fields are already plain strings (not enums), so `"websocket"` and `"none"` are already valid values. No enum changes needed.

**Step 3: Verify**

```bash
cd bin-openapi-manager && \
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

**Step 4: Verify bin-api-manager (generated from OpenAPI)**

```bash
cd bin-api-manager && \
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

**Step 5: Commit**

```
git add bin-openapi-manager/ bin-api-manager/
git commit -m "NOJIRA-tts-manager-websocket-streaming-design

- bin-openapi-manager: Add media_uri field to ExternalMedia schema
- bin-api-manager: Regenerate server code from updated OpenAPI spec"
```

---

## Task 6: bin-tts-manager — New websocket.go + delete audiosocket.go

Create the WebSocket utility functions that replace AudioSocket. Then delete audiosocket.go.

**Files:**
- Create: `bin-tts-manager/pkg/streaminghandler/websocket.go`
- Delete: `bin-tts-manager/pkg/streaminghandler/audiosocket.go`

**Step 1: Create websocket.go**

```go
package streaminghandler

import (
	"context"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	websocketMaxFragmentSize = 160                   // 160 bytes = 20ms at 8kHz MULAW (8-bit)
	websocketWriteDelay      = 20 * time.Millisecond // 20ms pacing between frames
	websocketSubprotocol     = "media"               // chan_websocket subprotocol
)

// websocketConnect dials the Asterisk chan_websocket endpoint and waits for the
// MEDIA_START text message that signals the channel is ready.
func websocketConnect(ctx context.Context, mediaURI string) (*websocket.Conn, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "websocketConnect",
		"media_uri": mediaURI,
	})

	dialer := websocket.Dialer{
		Subprotocols: []string{websocketSubprotocol},
	}

	conn, _, err := dialer.DialContext(ctx, mediaURI, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "could not dial WebSocket. media_uri: %s", mediaURI)
	}

	// Read the MEDIA_START text message from Asterisk
	msgType, msg, err := conn.ReadMessage()
	if err != nil {
		_ = conn.Close()
		return nil, errors.Wrapf(err, "could not read MEDIA_START message")
	}
	if msgType != websocket.TextMessage {
		_ = conn.Close()
		return nil, errors.Errorf("expected text message for MEDIA_START, got type %d", msgType)
	}
	log.Debugf("Received MEDIA_START message: %s", string(msg))

	return conn, nil
}

// websocketWrite fragments and sends raw audio data over a WebSocket connection
// as binary frames with 20ms pacing. For MULAW at 8kHz, each 160-byte frame
// represents 20ms of audio.
func websocketWrite(ctx context.Context, conn *websocket.Conn, data []byte) error {
	if len(data) == 0 {
		return nil
	}

	offset := 0
	payloadLen := len(data)

	for offset < payloadLen {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		fragmentLen := min(websocketMaxFragmentSize, payloadLen-offset)
		fragment := data[offset : offset+fragmentLen]

		if err := conn.WriteMessage(websocket.BinaryMessage, fragment); err != nil {
			return errors.Wrapf(err, "failed to write WebSocket binary frame")
		}

		offset += fragmentLen

		select {
		case <-time.After(websocketWriteDelay):
			continue
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return nil
}

// runWebSocketRead reads from the WebSocket connection to handle ping/pong and
// close frames. Without a read loop, gorilla/websocket won't acknowledge pings
// and the connection will time out. Cancels the provided cancel func on error
// or connection close.
func runWebSocketRead(ctx context.Context, cancel context.CancelFunc, conn *websocket.Conn) {
	log := logrus.WithField("func", "runWebSocketRead")
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			_, _, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
					log.Debugf("WebSocket closed normally: %v", err)
				} else {
					log.Errorf("WebSocket read error: %v", err)
				}
				return
			}
		}
	}
}
```

**Step 2: Delete audiosocket.go**

```bash
git rm bin-tts-manager/pkg/streaminghandler/audiosocket.go
```

**Step 3: Verify it compiles (it won't yet — callers still reference audiosocket functions)**

This file will compile standalone, but callers in gcp.go, elevenlabs.go, aws.go, and run.go still reference `audiosocketWrite` and other functions. Those are fixed in subsequent tasks. Don't commit yet — continue to Task 7.

---

## Task 7: bin-tts-manager — Streaming model + defaults + run.go cleanup

Change the `ConnAst` type, update defaults, remove AudioSocket-specific code from run.go, and clean up main.go.

**Files:**
- Modify: `bin-tts-manager/models/streaming/streaming.go:34` (ConnAst type)
- Modify: `bin-tts-manager/pkg/streaminghandler/main.go:59-68` (defaults, listenAddress)
- Modify: `bin-tts-manager/pkg/streaminghandler/run.go` (gut AudioSocket code)
- Modify: `bin-tts-manager/pkg/streaminghandler/streaming.go:135-146` (UpdateConnAst signature)
- Modify: `bin-tts-manager/cmd/tts-manager/main.go:125-142` (remove listenAddress, Run())

**Step 1: Change ConnAst type in streaming model**

In `bin-tts-manager/models/streaming/streaming.go`, change line 34:
```go
ConnAst   *websocket.Conn `json:"-"` // WebSocket connection to Asterisk
```

Add import:
```go
"github.com/gorilla/websocket"
```

Remove the `"net"` import (no longer needed).

**Step 2: Update defaults in main.go**

In `bin-tts-manager/pkg/streaminghandler/main.go`, change the default constants (lines 59-64):
```go
const (
	defaultEncapsulation  = string(cmexternalmedia.EncapsulationNone)
	defaultTransport      = string(cmexternalmedia.TransportWebsocket)
	defaultConnectionType = "server"
	defaultFormat         = "ulaw"
)
```

Remove `defaultSilenceFeedInterval` constant (lines 66-68).

Remove `listenAddress` field from struct (line 178).

Update `NewStreamingHandler()` — remove `listenAddress` parameter:
```go
func NewStreamingHandler(
	reqHandler requesthandler.RequestHandler,
	notifyHandler notifyhandler.NotifyHandler,
	podID string,
	elevenlabsAPIKey string,
	awsAccessKey string,
	awsSecretKey string,
) StreamingHandler {
```

Remove `listenAddress: listenAddress,` from the struct literal.

Change `Run()` in the interface to just return nil or remove it entirely. Since it's part of the public interface and used in main.go, make it a no-op:

In `run.go`, replace the entire `Run()` function with:
```go
func (h *streamingHandler) Run() error {
	// No-op: WebSocket connections are initiated per-session in startExternalMedia(),
	// not via a shared TCP listener.
	return nil
}
```

**Step 3: Remove AudioSocket functions from run.go**

Delete these functions from `run.go`:
- `runStart()` (lines 38-73)
- `runKeepConsume()` (lines 75-94)
- `runSilenceFeed()` (lines 96-130)

Remove unused imports (`"net"`, `"time"`, `"fmt"`). Keep `"context"` for `runStreamer()`.

**Step 4: Update UpdateConnAst in streaming.go**

In `bin-tts-manager/pkg/streaminghandler/streaming.go`, change `UpdateConnAst` (line 135):
```go
func (h *streamingHandler) UpdateConnAst(streamingID uuid.UUID, connAst *websocket.Conn) (*streaming.Streaming, error) {
```

Add import:
```go
"github.com/gorilla/websocket"
```

Remove `"net"` import.

**Step 5: Update cmd/tts-manager/main.go**

Remove `listenAddress` construction (line 127):
```go
// DELETE: listenAddress := fmt.Sprintf("%s:8080", localAddress)
```

Update `NewStreamingHandler()` call (line 130):
```go
streamingHandler := streaminghandler.NewStreamingHandler(reqHandler, notifyHandler, podID, config.Get().ElevenlabsAPIKey, config.Get().AWSAccessKey, config.Get().AWSSecretKey)
```

The `localAddress` variable is no longer needed. Remove it (line 125) if nothing else uses it. Check: `ttsHandler` uses it on line 129. Keep `localAddress` but remove `listenAddress`.

Remove `"fmt"` import if unused after removing `listenAddress`.

Keep `go runStreaming(streamingHandler)` — `Run()` is now a no-op but the call is harmless. Or remove it and the `runStreaming` function entirely. Prefer removing for cleanliness.

**Step 6: Verify compilation**

Run: `cd bin-tts-manager && go build ./...`
Expected: Compilation errors from gcp.go, elevenlabs.go, aws.go referencing `audiosocketWrite` and `net.Conn`. Fixed in next tasks.

---

## Task 8: bin-tts-manager — Update vendor handlers (GCP, ElevenLabs, AWS)

Update all three vendor handlers to use `*websocket.Conn` and `websocketWrite()`.

**Files:**
- Modify: `bin-tts-manager/pkg/streaminghandler/gcp.go:28-41` (GCPConfig struct)
- Modify: `bin-tts-manager/pkg/streaminghandler/gcp.go:133-171` (Init)
- Modify: `bin-tts-manager/pkg/streaminghandler/gcp.go:173-217` (connect — MULAW)
- Modify: `bin-tts-manager/pkg/streaminghandler/gcp.go:236-296` (Run + runProcess)
- Modify: `bin-tts-manager/pkg/streaminghandler/gcp.go:298-341` (SayAdd + SayFlush)
- Modify: `bin-tts-manager/pkg/streaminghandler/elevenlabs.go:28-40` (ElevenlabsConfig)
- Modify: `bin-tts-manager/pkg/streaminghandler/elevenlabs.go:295` (audiosocketWrite call)
- Modify: `bin-tts-manager/pkg/streaminghandler/aws.go:28-43` (AWSConfig)
- Modify: `bin-tts-manager/pkg/streaminghandler/aws.go:236` (audiosocketWrite call)

### GCP Handler

**Step 1: Update GCPConfig struct**

In `gcp.go`, change the GCPConfig struct:
```go
type GCPConfig struct {
	Streaming *streaming.Streaming

	Ctx    context.Context
	Cancel context.CancelFunc

	StreamCtx    context.Context    // per-stream sub-context, cancelled by SayFlush
	StreamCancel context.CancelFunc

	Client  *texttospeech.Client
	Stream  texttospeechpb.TextToSpeech_StreamingSynthesizeClient
	ConnAst *websocket.Conn

	VoiceID  string // stored for reconnect after flush
	LangCode string // stored for reconnect after flush

	Message *message.Message

	muStream sync.Mutex // protects Stream, Client, StreamCtx/StreamCancel
}
```

Add import:
```go
"github.com/gorilla/websocket"
```

Remove `"net"` import.

**Step 2: Update Init() to create StreamCtx and store voice/lang**

In `Init()`, change the context creation and GCPConfig construction:
```go
func (h *gcpHandler) Init(ctx context.Context, st *streaming.Streaming) (any, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "gcpHandler.Init",
		"streaming_id": st.ID,
	})

	voiceID := h.getVoiceID(ctx, st)
	log.Debugf("Using GCP voice: %s", voiceID)

	langCode := h.extractLangCode(voiceID, st.Language)

	client, stream, err := h.connect(ctx, voiceID, langCode)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to initialize GCP StreamingSynthesize")
	}

	cfCtx, cfCancel := context.WithCancel(context.Background())
	streamCtx, streamCancel := context.WithCancel(cfCtx)

	res := &GCPConfig{
		Streaming:    st,
		Ctx:          cfCtx,
		Cancel:       cfCancel,
		StreamCtx:    streamCtx,
		StreamCancel: streamCancel,
		Client:       client,
		Stream:       stream,
		ConnAst:      st.ConnAst,
		VoiceID:      voiceID,
		LangCode:     langCode,
		Message: &message.Message{
			Identity: commonidentity.Identity{
				ID:         st.MessageID,
				CustomerID: st.CustomerID,
			},
			StreamingID: st.ID,
		},
		muStream: sync.Mutex{},
	}

	h.notifyHandler.PublishEvent(cfCtx, message.EventTypeInitiated, res.Message)

	return res, nil
}
```

**Step 3: Change connect() to use MULAW encoding**

In `connect()`, change `AudioEncoding_LINEAR16` to `AudioEncoding_MULAW`:
```go
StreamingAudioConfig: &texttospeechpb.StreamingAudioConfig{
	AudioEncoding:   texttospeechpb.AudioEncoding_MULAW,
	SampleRateHertz: defaultGCPStreamingSampleRate,
},
```

**Step 4: Update runProcess() to use StreamCtx and websocketWrite**

In `runProcess()`, change the defer to NOT cancel session context:
```go
defer func() {
	cf.StreamCancel()
	h.notifyHandler.PublishEvent(cf.Ctx, message.EventTypePlayFinished, msg)
}()
```

Change the Recv loop to check StreamCtx and use websocketWrite:
```go
for {
	select {
	case <-cf.StreamCtx.Done():
		return
	default:
	}

	cf.muStream.Lock()
	stream := cf.Stream
	cf.muStream.Unlock()

	if stream == nil {
		return
	}

	resp, err := stream.Recv()
	if err != nil {
		log.Infof("GCP stream ended: %v", err)
		return
	}

	audioData := resp.GetAudioContent()
	if len(audioData) == 0 {
		continue
	}

	if errWrite := websocketWrite(cf.StreamCtx, cf.ConnAst, audioData); errWrite != nil {
		log.Errorf("Could not write audio to asterisk: %v", errWrite)
		return
	}
}
```

**Step 5: Implement real SayFlush()**

Replace the SayFlush implementation:
```go
func (h *gcpHandler) SayFlush(vendorConfig any) error {
	cf, ok := vendorConfig.(*GCPConfig)
	if !ok || cf == nil {
		return fmt.Errorf("vendorConfig is not a *GCPConfig or is nil")
	}

	cf.muStream.Lock()
	defer cf.muStream.Unlock()

	// Cancel the stream sub-context to stop runProcess and websocketWrite immediately
	cf.StreamCancel()

	// Close gRPC stream and client to discard any GCP-side buffered audio
	if cf.Stream != nil {
		_ = cf.Stream.CloseSend()
		cf.Stream = nil
	}
	if cf.Client != nil {
		_ = cf.Client.Close()
		cf.Client = nil
	}

	return nil
}
```

**Step 6: Update SayAdd() with reconnect logic**

Replace SayAdd:
```go
func (h *gcpHandler) SayAdd(vendorConfig any, text string) error {
	cf, ok := vendorConfig.(*GCPConfig)
	if !ok || cf == nil {
		return fmt.Errorf("vendorConfig is not a *GCPConfig or is nil")
	}

	cf.muStream.Lock()
	defer cf.muStream.Unlock()

	// If stream was flushed, reconnect
	if cf.Stream == nil {
		client, stream, err := h.connect(cf.Ctx, cf.VoiceID, cf.LangCode)
		if err != nil {
			return errors.Wrapf(err, "failed to reconnect GCP stream after flush")
		}
		cf.Client = client
		cf.Stream = stream

		// Create new stream sub-context
		streamCtx, streamCancel := context.WithCancel(cf.Ctx)
		cf.StreamCtx = streamCtx
		cf.StreamCancel = streamCancel

		// Restart the audio delivery goroutine
		go h.runProcess(cf)
	}

	req := &texttospeechpb.StreamingSynthesizeRequest{
		StreamingRequest: &texttospeechpb.StreamingSynthesizeRequest_Input{
			Input: &texttospeechpb.StreamingSynthesisInput{
				InputSource: &texttospeechpb.StreamingSynthesisInput_Text{
					Text: text,
				},
			},
		},
	}

	if err := cf.Stream.Send(req); err != nil {
		return errors.Wrapf(err, "failed to send text to GCP stream")
	}

	cf.Message.TotalMessage += text
	cf.Message.TotalCount++

	return nil
}
```

### ElevenLabs Handler

**Step 7: Update ElevenlabsConfig**

In `elevenlabs.go`, change `ConnAst` type (line 35):
```go
ConnAst *websocket.Conn `json:"-"` // connector between the service and Asterisk
```

Remove `"net"` from imports. The `websocket` import is already present.

**Step 8: Replace audiosocketWrite with websocketWrite**

In `runProcess()` (line 295), change:
```go
if errWrite := audiosocketWrite(cf.Ctx, cf.ConnAst, data); errWrite != nil {
```
to:
```go
if errWrite := websocketWrite(cf.Ctx, cf.ConnAst, data); errWrite != nil {
```

Also in the `runProcess()` defer and error handling, change `net.ErrClosed` check (line 266).
Since we no longer import `"net"`, change:
```go
if errors.Cause(err) == net.ErrClosed {
```
to:
```go
if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
```

### AWS Handler

**Step 9: Update AWSConfig**

In `aws.go`, change `ConnAst` type (line 35):
```go
ConnAst *websocket.Conn
```

Add import:
```go
"github.com/gorilla/websocket"
```

Remove `"net"` from imports.

**Step 10: Replace audiosocketWrite with websocketWrite**

In `runProcess()` (line 236), change:
```go
if errWrite := audiosocketWrite(cf.Ctx, cf.ConnAst, audioData); errWrite != nil {
```
to:
```go
if errWrite := websocketWrite(cf.Ctx, cf.ConnAst, audioData); errWrite != nil {
```

**Step 11: Verify compilation**

Run: `cd bin-tts-manager && go build ./...`
Expected: SUCCESS

**Step 12: Commit**

```
git add bin-tts-manager/
git commit -m "NOJIRA-tts-manager-websocket-streaming-design

- bin-tts-manager: Add websocket.go with connect/write/read functions
- bin-tts-manager: Delete audiosocket.go (replaced by websocket.go)
- bin-tts-manager: Change ConnAst from net.Conn to *websocket.Conn
- bin-tts-manager: Update defaults to websocket/none/server/ulaw
- bin-tts-manager: Make Run() a no-op (no TCP listener needed)
- bin-tts-manager: Remove runStart, runSilenceFeed, runKeepConsume
- bin-tts-manager: GCP handler uses MULAW, per-stream context, real SayFlush
- bin-tts-manager: ElevenLabs/AWS handlers use websocketWrite"
```

---

## Task 9: bin-tts-manager — Update startExternalMedia for WebSocket

Connect to Asterisk via WebSocket after creating external media.

**Files:**
- Modify: `bin-tts-manager/pkg/streaminghandler/start.go:84-113`

**Step 1: Update startExternalMedia()**

Replace `startExternalMedia()`:
```go
func (h *streamingHandler) startExternalMedia(ctx context.Context, st *streaming.Streaming) error {
	log := logrus.WithFields(logrus.Fields{
		"func":         "startExternalMedia",
		"streaming_id": st.ID,
	})

	em, err := h.requestHandler.CallV1ExternalMediaStart(
		ctx,
		st.ID,
		externalmedia.ReferenceType(st.ReferenceType),
		st.ReferenceID,
		"INCOMING",
		defaultEncapsulation,
		defaultTransport,
		"", // transportData
		defaultConnectionType,
		defaultFormat,
		externalmedia.DirectionNone,
		externalmedia.Direction(st.Direction),
	)
	if err != nil {
		log.Errorf("Could not create external media. err: %v", err)
		promStreamingErrorTotal.WithLabelValues("unknown").Inc()
		return err
	}
	log.WithField("external_media", em).Debugf("Started external media. external_media_id: %s, media_uri: %s", em.ID, em.MediaURI)

	// Connect to Asterisk via WebSocket
	conn, err := websocketConnect(ctx, em.MediaURI)
	if err != nil {
		log.Errorf("Could not connect WebSocket to Asterisk. err: %v", err)
		return err
	}
	log.Debugf("WebSocket connected to Asterisk. media_uri: %s", em.MediaURI)

	// Store the WebSocket connection on the streaming record
	if _, errUpdate := h.UpdateConnAst(st.ID, conn); errUpdate != nil {
		_ = conn.Close()
		return errUpdate
	}

	// Spawn read goroutine for ping/pong/close handling.
	// Uses a session-scoped context that will be cancelled when Stop() is called.
	sessionCtx, sessionCancel := context.WithCancel(context.Background())
	go func() {
		runWebSocketRead(sessionCtx, sessionCancel, conn)
		_ = conn.Close()
	}()

	// Store the session cancel so Stop() can tear down the WebSocket
	st.ConnAstCancel = sessionCancel

	return nil
}
```

Wait — the Streaming struct doesn't have `ConnAstCancel`. Let me reconsider. The WebSocket read goroutine needs to be cancelled when `Stop()` is called. Currently, `Stop()` calls `CallV1ExternalMediaStop()` which tells Asterisk to tear down the channel, which closes the WebSocket from Asterisk's side, which causes `runWebSocketRead` to exit naturally.

So we don't need to store the cancel — the read goroutine exits when Asterisk closes the connection. Simplify:

```go
func (h *streamingHandler) startExternalMedia(ctx context.Context, st *streaming.Streaming) error {
	log := logrus.WithFields(logrus.Fields{
		"func":         "startExternalMedia",
		"streaming_id": st.ID,
	})

	em, err := h.requestHandler.CallV1ExternalMediaStart(
		ctx,
		st.ID,
		externalmedia.ReferenceType(st.ReferenceType),
		st.ReferenceID,
		"INCOMING",
		defaultEncapsulation,
		defaultTransport,
		"", // transportData
		defaultConnectionType,
		defaultFormat,
		externalmedia.DirectionNone,
		externalmedia.Direction(st.Direction),
	)
	if err != nil {
		log.Errorf("Could not create external media. err: %v", err)
		promStreamingErrorTotal.WithLabelValues("unknown").Inc()
		return err
	}
	log.WithField("external_media", em).Debugf("Started external media. external_media_id: %s, media_uri: %s", em.ID, em.MediaURI)

	// Connect to Asterisk via WebSocket
	conn, err := websocketConnect(ctx, em.MediaURI)
	if err != nil {
		log.Errorf("Could not connect WebSocket to Asterisk. err: %v", err)
		return err
	}
	log.Debugf("WebSocket connected to Asterisk. media_uri: %s", em.MediaURI)

	// Store the WebSocket connection on the streaming record
	if _, errUpdate := h.UpdateConnAst(st.ID, conn); errUpdate != nil {
		_ = conn.Close()
		return errUpdate
	}

	// Spawn read goroutine for WebSocket lifecycle (ping/pong/close).
	// When Asterisk closes the channel, the read goroutine exits and closes the conn.
	go func() {
		readCtx, readCancel := context.WithCancel(context.Background())
		defer readCancel()
		runWebSocketRead(readCtx, readCancel, conn)
	}()

	return nil
}
```

Remove `h.listenAddress` reference. Remove unused imports. The `externalmedia` import is already present.

**Step 2: Verify compilation**

Run: `cd bin-tts-manager && go build ./...`
Expected: SUCCESS

**Step 3: Commit**

```
git add bin-tts-manager/pkg/streaminghandler/start.go
git commit -m "NOJIRA-tts-manager-websocket-streaming-design

- bin-tts-manager: Update startExternalMedia to use INCOMING mode and WebSocket"
```

---

## Task 10: bin-tts-manager — Code generation + test fixes

**Files:**
- Modify: Generated mock files
- Modify: Test files with updated signatures

**Step 1: Regenerate mocks**

```bash
cd bin-tts-manager && go generate ./...
```

**Step 2: Fix test files**

Search for test files that reference AudioSocket functions or `net.Conn`:
```bash
grep -rn "audiosocket\|net\.Conn\|listenAddress" bin-tts-manager/ --include="*_test.go"
```

For `start_test.go`: Update `NewStreamingHandler()` calls to remove `listenAddress` parameter.

For `run_test.go`: The `MockConn` implementing `net.Conn` is used for `runSilenceFeed()` tests. Since `runSilenceFeed()` is deleted, these tests should be removed or rewritten.

For any test constructing `streamingHandler` struct directly, remove `listenAddress` field.

**Step 3: Verify**

```bash
cd bin-tts-manager && go test ./...
```

**Step 4: Commit**

```
git add bin-tts-manager/
git commit -m "NOJIRA-tts-manager-websocket-streaming-design

- bin-tts-manager: Regenerate mocks and fix tests for WebSocket migration"
```

---

## Task 11: Full verification — all affected services

**Step 1: Verify bin-call-manager**

```bash
cd bin-call-manager && \
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

**Step 2: Verify bin-tts-manager**

```bash
cd bin-tts-manager && \
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

**Step 3: Verify bin-openapi-manager**

```bash
cd bin-openapi-manager && \
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

**Step 4: Verify bin-api-manager**

```bash
cd bin-api-manager && \
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

**Step 5: Commit any final fixes**

---

## Task 12: Remove runStreaming from main.go + final cleanup

**Step 1: Clean up tts-manager main.go**

If `Run()` is a no-op, remove `go runStreaming(streamingHandler)` and the `runStreaming` function from `cmd/tts-manager/main.go`.

**Step 2: Remove unused `fmt` import if present**

After removing `listenAddress := fmt.Sprintf(...)`, check if `fmt` is still used.

**Step 3: Final verification**

```bash
cd bin-tts-manager && go build ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 4: Commit**

```
git add bin-tts-manager/cmd/tts-manager/main.go
git commit -m "NOJIRA-tts-manager-websocket-streaming-design

- bin-tts-manager: Remove unused runStreaming and listenAddress from main"
```
