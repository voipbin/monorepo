# Pipecat Manager: Replace Audiosocket with WebSocket External Media

## Problem

The pipecat-manager currently uses Audiosocket TCP (`encapsulation: "audiosocket"`, `transport: "tcp"`) to communicate with Asterisk. This requires:
- A TCP listener goroutine in the Go service
- Asterisk connecting as a client to the listener
- 8kHz SLIN audio with manual upsample/downsample to/from 16kHz for the Python Pipecat side

The tts-manager already uses WebSocket external media (`encapsulation: "none"`, `transport: "websocket"`), which is simpler and better supported. Pipecat-manager should use the same pattern.

## Approach

Full replacement of Audiosocket with WebSocket, following the tts-manager pattern.

### Current Architecture

```
Asterisk --[Audiosocket TCP, 8kHz slin]--> Go (TCP listener port 8080)
                                           Go --[internal WS, 16kHz protobuf]--> Python Pipecat
                                           Go <--[internal WS, 16kHz protobuf]-- Python Pipecat
Asterisk <--[Audiosocket TCP, 8kHz slin]-- Go
```

### Target Architecture

```
Asterisk (WS server) --[WebSocket, 16kHz slin16]--> Go (dials em.MediaURI)
                                                     Go --[internal WS, 16kHz protobuf]--> Python Pipecat
                                                     Go <--[internal WS, 16kHz protobuf]-- Python Pipecat
Asterisk (WS server) <--[WebSocket, 16kHz slin16]-- Go
```

No sample rate conversion needed. The internal Go ↔ Python WebSocket connections (input/output) remain unchanged.

## Design

### 1. Constants Update (main.go)

```go
const (
    defaultEncapsulation  = string(cmexternalmedia.EncapsulationNone)     // was "audiosocket"
    defaultTransport      = string(cmexternalmedia.TransportWebsocket)    // was "tcp"
    defaultConnectionType = "server"                                      // was "client"
    defaultFormat         = "slin16"                                      // was "slin"
    defaultFrameSize      = 640     // 16000 Hz * 2 bytes * 20ms
    websocketSubprotocol  = "media"
    websocketWriteDelay   = 20 * time.Millisecond
)
```

Remove `listenAddress` field from handler struct and configuration.

### 2. External Media Creation (start.go)

Update `CallV1ExternalMediaStart` parameters:
- `externalHost`: `h.listenAddress` → `"INCOMING"`
- `encapsulation`: `"audiosocket"` → `"none"`
- `transport`: `"tcp"` → `"websocket"`
- `connectionType`: `"client"` → `"server"`
- `format`: `"slin"` → `"slin16"`

After external media is created, immediately dial `em.MediaURI` using the WebSocket connect pattern from tts-manager:
1. Dial with `"media"` subprotocol
2. Wait for `MEDIA_START` text message (10s timeout)
3. Store `*websocket.Conn` and `ConnAstDone` channel in session

### 3. Session Model (models/pipecatcall/session.go)

Replace:
- `AsteriskConn net.Conn` → `ConnAst *websocket.Conn`
- Add `ConnAstDone chan struct{}` for lifecycle management

### 4. Remove TCP Listener (run.go)

Remove:
- `Run()` TCP listener (`net.Listen`, `listener.Accept`)
- `runStart()` Audiosocket connection handler
- Streaming ID extraction from Audiosocket first message

The connection is now established in `start.go` after external media creation. The session is created there too, since we have the WebSocket connection immediately.

### 5. Audio Reading: Asterisk → Go → Python (run.go)

Replace `runAsteriskReceivedMediaHandle()`:
- Current: `audiosocketHandler.GetNextMedia(conn)` → `Upsample8kTo16k()` → `SendAudio()`
- New: Read binary WebSocket frame (raw slin16 bytes) → `SendAudio()` directly

No sample rate conversion needed since both Asterisk and Python use 16kHz.

### 6. Audio Writing: Python → Go → Asterisk (runner.go)

Replace audio write in `runnerWebsocketHandleAudio()`:
- Current: `GetDataSamples(sampleRate, data)` → `audiosocketHandler.Write(conn, data)`
- New: If `sampleRate != 16000`, resample to 16kHz (safety net). Write binary WebSocket frames with 640-byte chunks and 20ms pacing.

Follow tts-manager's `websocketWrite()` pattern for frame pacing.

### 7. WebSocket Helper Functions

Add new functions (in existing `websocket.go` or new file):
- `websocketAsteriskConnect(ctx, mediaURI)` — dial with "media" subprotocol, wait for MEDIA_START
- `websocketAsteriskWrite(ctx, conn, data, frameSize)` — write with 20ms pacing
- `runWebSocketAsteriskRead(conn, doneCh)` — lifecycle goroutine (closes doneCh on disconnect)

### 8. Cleanup

Remove from `audiosocket.go`:
- `GetStreamingID()` — Audiosocket first message parsing
- `GetNextMedia()` — Audiosocket audio frame reading
- `Upsample8kTo16k()` — no longer needed with slin16
- `WrapDataPCM16Bit()` — Audiosocket framing
- `Write()` — Audiosocket fragmented writing

Keep:
- `GetDataSamples()` — safety net for resampling if Python sends non-16kHz audio

### 9. Test Coverage

**WebSocket connection tests:**
- Successful connection with MEDIA_START handshake
- Connection dial failure
- MEDIA_START timeout (>10s)
- Unexpected first message (not MEDIA_START)

**Audio reading tests (Asterisk → Go):**
- Read binary WebSocket frame, verify raw slin16 bytes forwarded to Python channel
- WebSocket close mid-read (clean shutdown)
- Context cancellation during read loop

**Audio writing tests (Go → Asterisk):**
- Write with correct 640-byte frame size and 20ms pacing
- Data smaller than one frame
- Data spanning multiple frames
- WebSocket close mid-write
- Context cancellation during write

**External media start tests:**
- Verify correct parameters passed (encapsulation: "none", transport: "websocket", etc.)
- External media creation failure → clean error
- WebSocket dial failure after external media created → stop external media and clean up

**Session lifecycle tests:**
- ConnAstDone channel closes when WebSocket disconnects
- Terminate properly closes WebSocket and stops external media

**Resampling safety net:**
- Audio at 16kHz passes through unchanged
- Audio at non-16kHz gets resampled to 16kHz

## Files Changed

| File | Action |
|------|--------|
| `pkg/pipecatcallhandler/main.go` | Update constants, remove listenAddress |
| `pkg/pipecatcallhandler/start.go` | New external media params + WebSocket dial + session creation |
| `pkg/pipecatcallhandler/run.go` | Remove TCP listener, update audio read to WebSocket |
| `pkg/pipecatcallhandler/runner.go` | Update audio write to use WebSocket frames |
| `pkg/pipecatcallhandler/audiosocket.go` | Remove Audiosocket-specific functions, keep GetDataSamples |
| `pkg/pipecatcallhandler/websocket.go` | Add Asterisk WebSocket connect/write/lifecycle functions |
| `models/pipecatcall/session.go` | Replace net.Conn with *websocket.Conn + done channel |

## Trade-offs

- **Removes Audiosocket fallback** — if WebSocket has issues in some environment, there's no fallback. Mitigated by tts-manager already proving WebSocket works.
- **Eliminates resampling** — better audio quality (no interpolation artifacts), less CPU usage.
- **Simpler connection model** — no TCP listener needed, Go dials Asterisk's MediaURI directly.
