# Migrate tts-manager Streaming from AudioSocket to WebSocket (chan_websocket)

## Problem Statement

The tts-manager streaming handler currently uses Asterisk's AudioSocket protocol over TCP for real-time TTS audio delivery. This requires:
- A TCP listener on each pod (`POD_IP:8080`)
- Asterisk connecting as a TCP client to tts-manager
- AudioSocket binary framing (format + length + PCM data)
- Continuous silence feed every 20ms to prevent channel teardown
- SLIN 16-bit encoding (320 bytes/frame), requiring LINEAR16 from GCP TTS

Asterisk now supports `chan_websocket`, a WebSocket-based external media channel driver with an INCOMING mode where the application connects TO Asterisk. This eliminates the need for a listener, simplifies the protocol (raw binary frames, no wrapping), and solves K8s pod routing issues.

## Approach

Replace AudioSocket entirely in tts-manager with WebSocket using `chan_websocket` INCOMING mode. The tts-manager becomes a WebSocket client that connects to Asterisk's HTTP server. Control plane still goes through call-manager (architecture preserved). Focus on GCP TTS provider first with MULAW encoding.

Also implement a real `SayFlush` operation that immediately stops audio playback and discards buffered audio without disconnecting the session.

## Key Decisions

- **INCOMING mode**: tts-manager connects TO Asterisk (not reverse). Eliminates TCP listener and K8s pod routing issues. Same pod that creates the session initiates the WebSocket connection.
- **Replace AudioSocket entirely**: No fallback. All vendor handlers (GCP, ElevenLabs, AWS) updated.
- **MULAW encoding**: 8-bit, 160 bytes/20ms. GCP StreamingSynthesize supports MULAW natively. Half the bandwidth of LINEAR16.
- **No silence feed**: chan_websocket handles idle internally, unlike AudioSocket which tears down the channel without continuous data.
- **Per-stream sub-context for Flush**: Enables immediate audio cutoff without disconnecting the WebSocket session.
- **`ASTERISK_WS_PORT` config on call-manager**: Configurable port (default 8088) for constructing WebSocket media URI. All Asterisk pods share the same HTTP config.

## Architecture

### Session Setup Flow

```
tts-manager                    call-manager                     Asterisk
    |                               |                              |
    |-- CallV1ExternalMediaStart -->|                              |
    |   (external_host=INCOMING     |                              |
    |    transport=websocket        |-- ARI POST externalMedia --->|
    |    encapsulation=none         |   (via asterisk-proxy/MQ)    |
    |    connection_type=server     |                              |
    |    format=ulaw)               |<-- channel response ---------|
    |                               |   (MEDIA_WEBSOCKET_          |
    |                               |    CONNECTION_ID in vars)    |
    |                               |                              |
    |                               | cache.AsteriskAddressInternalGet()
    |                               | → Redis: asterisk.<mac>.address-internal
    |                               | construct ws://ip:port/media/{id}
    |                               | store MediaURI on ExternalMedia
    |                               |                              |
    |<-- ExternalMedia (MediaURI) --|                              |
    |                                                              |
    |-- WebSocket dial to ws://asterisk-ip:8088/media/{id} ------>|
    |<-- MEDIA_START JSON text message ----------------------------|
    |   (connection ready)                                         |
    |                                                              |
    | [spawn read goroutine for ping/pong/close handling]          |
    |                                                              |
```

### Audio Streaming Flow (after SayInit + SayAdd)

```
GCP TTS                   tts-manager                    Asterisk
    |                          |                             |
    |<-- StreamingSynthesize --|                             |
    |   (MULAW, 8kHz)         |                             |
    |                          |                             |
    |-- audio chunk ---------> |                             |
    |                          |-- binary WS frame --------> |
    |                          |   (160 bytes raw MULAW)     |
    |                          |   [20ms pacing]             |
    |                          |                             |
    |-- audio chunk ---------> |                             |
    |                          |-- binary WS frame --------> |
    |                          |   ...                       |
```

### SayFlush Flow

```
SayFlush()
  |
  |-- StreamCancel()              cancel per-stream sub-context
  |     |-- websocketWrite()      sees ctx.Err() → stops sending frames
  |     |-- runProcess()          Recv() loop exits (stream closed)
  |
  |-- cf.Client.Close()           kills gRPC connection immediately
  |                               discards any GCP-side buffered audio
  |-- cf.Stream = nil
  |-- cf.Client = nil
  |
  |   WebSocket to Asterisk stays open → Asterisk hears silence
  |
  ... later ...
  |
SayAdd(text)
  |-- cf.Stream == nil → reconnect
  |     |-- h.connect(voiceID, langCode)    new GCP client + stream
  |     |-- new StreamCtx/StreamCancel
  |     |-- go runProcess()                 restart audio delivery
  |-- stream.Send(text)                     audio resumes
```

### Say Operation Lifecycle

```
SayInit(id, messageID)
  → VendorConfig == nil? → runStreamer() → gcpHandler.Init() + Run()
  → Sets messageID on streaming

SayAdd(id, messageID, text)  [can be called multiple times]
  → Queued on GCP gRPC stream, synthesized in FIFO order
  → Single runProcess() goroutine delivers audio sequentially
  → If Stream == nil (after flush): reconnects, restarts runProcess()

SayFlush(id)
  → Cancels StreamCtx, closes GCP client/stream
  → Immediately stops audio delivery, discards buffered audio
  → WebSocket stays alive, session continues

SayFinish(id, messageID)
  → CloseSend() on GCP stream, drains remaining audio normally

SayStop(id)
  → Cancels session context, tears down everything including WebSocket
```

### Session Teardown Flow

```
Stop(streamingID)
  → Cancel session context
  → gcpHandler terminates (Run() unblocks, terminate() cleans up)
  → WebSocket read goroutine exits, closes WebSocket
  → Asterisk detects disconnect → destroys channel
  → Asterisk sends StasisEnd → call-manager cleans up ExternalMedia
```

## Files to Change

### bin-call-manager

#### models/externalmedia/main.go

Add constants and field:
- `TransportWebsocket Transport = "websocket"`
- `EncapsulationNone Encapsulation = "none"`
- `MediaURI string` field on ExternalMedia struct (`json:"media_uri,omitempty"`)

#### internal/config/main.go

Add `AsteriskWSPort int` to Config struct and bind `ASTERISK_WS_PORT` env var (default 8088).

#### cmd/call-manager/main.go

Pass `cache` and `cfg.AsteriskWSPort` to `NewExternalMediaHandler()`.

#### pkg/externalmediahandler/main.go

- Add `cache cachehandler.CacheHandler` and `asteriskWSPort int` to `externalMediaHandler` struct
- Update `NewExternalMediaHandler()` to accept these parameters
- Add constant: `ChannelVariableWebSocketConnectionID = "MEDIA_WEBSOCKET_CONNECTION_ID"`

#### pkg/externalmediahandler/start.go

Fix pre-existing bug: thread `connectionType` parameter through instead of hardcoded `defaultConnectionType` at lines 289 and 308.

After ARI response in `startExternalMedia()`, add WebSocket branch:
```go
if transport == externalmedia.TransportWebsocket {
    connectionID := extCh.Data[ChannelVariableWebSocketConnectionID].(string)
    asteriskIP, _ := h.cache.AsteriskAddressInternalGet(ctx, asteriskID)
    mediaURI := fmt.Sprintf("ws://%s:%d/media/%s", asteriskIP, h.asteriskWSPort, connectionID)
    // update ExternalMedia with MediaURI
}
```

#### pkg/externalmediahandler/db.go

Set `MediaURI` on ExternalMedia struct in `Create()`.

### bin-tts-manager

#### models/streaming/streaming.go

Change `ConnAst net.Conn` to `ConnAst *websocket.Conn` (gorilla/websocket).

#### pkg/streaminghandler/main.go

- Change defaults: `encapsulation=none`, `transport=websocket`, `connectionType=server`, `format=ulaw`
- Remove `listenAddress` field from struct
- Remove `defaultSilenceFeedInterval` constant
- Change `Run()` to return nil immediately (no-op) or remove from interface

#### pkg/streaminghandler/audiosocket.go

Delete entirely. Replaced by websocket.go.

#### pkg/streaminghandler/websocket.go (new)

- `websocketConnect(ctx, mediaURI) (*websocket.Conn, error)` — dials WebSocket with `media` subprotocol, reads MEDIA_START text message, returns conn
- `websocketWrite(ctx, conn, data) error` — fragments into 160-byte chunks, sends as binary WebSocket frames with 20ms pacing
- `runWebSocketRead(ctx, cancel, conn)` — read loop for ping/pong/close handling, cancels context on error

#### pkg/streaminghandler/run.go

Remove:
- `Run()` TCP listener (or make no-op)
- `runStart()` AudioSocket connection handler
- `runSilenceFeed()` silence frame sender
- `runKeepConsume()` incoming data drain

Keep:
- `runStreamer()` (unchanged)
- `getStreamerByProvider()` (unchanged)

#### pkg/streaminghandler/start.go

Change `startExternalMedia()`:
- Parameters: `external_host="INCOMING"`, `encapsulation="none"`, `transport="websocket"`, `connection_type="server"`, `format="ulaw"`
- After `CallV1ExternalMediaStart()` returns, connect WebSocket to `em.MediaURI`
- Store `*websocket.Conn` on `streaming.ConnAst`
- Spawn WebSocket read goroutine for connection lifecycle

#### pkg/streaminghandler/gcp.go

GCPConfig changes:
- `ConnAst net.Conn` → `ConnAst *websocket.Conn`
- Add `StreamCtx context.Context` and `StreamCancel context.CancelFunc` (per-stream sub-context)
- Add `VoiceID string` and `LangCode string` (stored at Init for reconnect after flush)

Init():
- Create both session context (`Ctx`) and stream sub-context (`StreamCtx`)
- Store `VoiceID` and `LangCode` on config

connect():
- `AudioEncoding_LINEAR16` → `AudioEncoding_MULAW`

runProcess():
- Use `StreamCtx` for `websocketWrite()` context
- On exit, do NOT cancel session context (only publish PlayFinished event)

SayFlush():
- Cancel `StreamCtx` (stops writes and runProcess immediately)
- Close GCP client and stream
- Set `Stream = nil`, `Client = nil`

SayAdd():
- If `Stream == nil`: reconnect (new client + stream + StreamCtx), restart `runProcess()`
- Then send text

Run():
- Block on session `Ctx.Done()` (not stream context)
- `terminate()` cleans up whatever stream/client is current

#### pkg/streaminghandler/elevenlabs.go

- `ConnAst net.Conn` → `ConnAst *websocket.Conn` in ElevenlabsConfig
- Replace `audiosocketWrite()` call with `websocketWrite()`

#### pkg/streaminghandler/aws.go

- `ConnAst net.Conn` → `ConnAst *websocket.Conn` in AWSConfig
- Replace `audiosocketWrite()` call with `websocketWrite()`

#### cmd/tts-manager/main.go

- Remove `listenAddress` construction (line 127)
- Remove `listenAddress` from `NewStreamingHandler()` call
- Remove `go runStreaming(streamingHandler)` call (or keep if Run() is no-op)

### bin-openapi-manager

#### openapi/openapi.yaml

- Add `websocket` to Transport enum
- Add `none` to Encapsulation enum
- Add `media_uri` field to ExternalMedia schema

### Code Generation

Regenerate mocks for:
- `bin-call-manager`: externalmediahandler, cachehandler
- `bin-tts-manager`: streaminghandler
- `bin-openapi-manager`: OpenAPI types
- `bin-api-manager`: server code from OpenAPI

### Tests

Update mock expectations in all affected test files to match:
- New `NewExternalMediaHandler()` signature (added cache, asteriskWSPort)
- Fixed `connectionType` threading in startExternalMedia
- New `*websocket.Conn` type on ConnAst
- New `websocketWrite()` calls replacing `audiosocketWrite()`
- Real `SayFlush` behavior

## Pre-existing Issue (Fixed Here)

`connectionType` parameter in `externalmediahandler.Start()` is accepted but silently dropped. `startExternalMedia()` always uses `defaultConnectionType` ("client") at lines 289 and 308. This is fixed as part of this change since INCOMING mode requires `connection_type=server`.

## Backwards Compatibility

- Existing AudioSocket callers (bin-transcribe-manager, bin-pipecat-manager, bin-api-manager) pass `transport=udp/tcp`, `encapsulation=rtp/audiosocket`. No change to their behavior.
- `connectionType` bug fix: all current callers pass `""` or `"client"`, so no behavioral change.
- New `MediaURI` field is `omitempty` — absent from wire format when empty (all non-WebSocket cases).
- New `TransportWebsocket`/`EncapsulationNone` constants are additive.
- Redis-stored ExternalMedia structs missing `MediaURI` deserialize with zero value (`""`).

## Services Requiring Verification

`bin-call-manager`, `bin-tts-manager`, `bin-openapi-manager`, `bin-api-manager`
