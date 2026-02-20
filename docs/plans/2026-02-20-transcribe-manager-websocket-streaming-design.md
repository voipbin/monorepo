# Transcribe-Manager WebSocket Streaming

Migrate the transcribe-manager's real-time streaming external media from AudioSocket/TCP
to WebSocket transport, matching the pattern already established by the tts-manager.

## Problem

The transcribe-manager currently uses AudioSocket over TCP for receiving live audio from
Asterisk. This requires a TCP listener on each pod (`POD_IP:8080`), the AudioSocket
binary protocol (CyCoreSystems/audiosocket library), and a custom keep-alive mechanism.
The tts-manager has already migrated to WebSocket transport (`chan_websocket`), which
eliminates the TCP listener, uses standard WebSocket framing, and handles keep-alive
natively via ping/pong.

## Approach

Follow the tts-manager's WebSocket pattern, adapted for reading audio (STT) instead of
writing audio (TTS).

### Connection Model Change

| Aspect | Current (AudioSocket/TCP) | Target (WebSocket) |
|---|---|---|
| Encapsulation | `audiosocket` | `none` |
| Transport | `tcp` | `websocket` |
| Connection type | `client` (Asterisk dials us) | `server` (we dial Asterisk) |
| Format | `slin` (16-bit PCM, 8kHz) | `slin` (unchanged) |
| External host | `POD_IP:PORT` | `INCOMING` |
| Session ID | AudioSocket KindID message | Not needed (we know the ID when dialing) |
| Keep-alive | Custom 0x10 byte packets | WebSocket ping/pong (automatic) |
| Audio read | `audiosocket.NextMessage()` | `conn.ReadMessage()` binary frames |

The audio format stays as `slin` (LINEAR16) because both GCP and AWS STT expect 16-bit
PCM at 8kHz. The tts-manager uses `ulaw` because that's what it generates for playback.

### Control Flow Change

Current flow:
1. `Start()` creates streaming record, calls `ExternalMediaStart` with listen address
2. Asterisk connects to our TCP listener
3. `Run()` accepts connection, `runStart()` extracts streaming ID from AudioSocket
4. `runStart()` selects STT provider (GCP/AWS) based on priority and calls handler
5. Handler blocks on `<-ctx.Done()` until audio connection closes

New flow:
1. `Start()` creates streaming record, calls `ExternalMediaStart` with `INCOMING` host
2. `Start()` dials WebSocket using returned `MediaURI`
3. `Start()` stores `*websocket.Conn` on streaming record
4. `Start()` spawns `runWebSocketRead()` goroutine for ping/pong/close handling
5. `Start()` spawns STT processing goroutine (provider selection + handler)
6. `Start()` returns immediately; STT handler runs until WebSocket closes

The key difference: STT processing is triggered from `Start()` instead of from the TCP
listener. The provider selection logic moves from `runStart()` into a helper called by
`Start()`.

### WebSocket Protocol

Matching the tts-manager's implementation:

- **Subprotocol**: `"media"` (chan_websocket)
- **Handshake**: Asterisk sends `MEDIA_START` text message after connection
- **Audio data**: Binary frames containing raw slin PCM bytes
- **Close detection**: `runWebSocketRead()` goroutine reads messages to handle ping/pong;
  when connection closes, it closes a `ConnAstDone` channel
- **Cleanup**: `Stop()` calls `ExternalMediaStop`, then `conn.Close()`

### Error Handling

If `websocketConnect()` fails after `ExternalMediaStart` succeeds, clean up the orphaned
external media by calling `ExternalMediaStop`. This matches the tts-manager pattern in
`start.go:116-121`.

### Streaming Model Changes

Add to `models/streaming/streaming.go`:
- `ConnAst *websocket.Conn` — WebSocket connection to Asterisk
- `ConnAstDone chan struct{}` — closed when WebSocket disconnects

The GCP/AWS media processing loops will read audio from `st.ConnAst` via
`conn.ReadMessage()`. When the connection closes, `ReadMessage()` returns an error and
the loop exits, same as the current behavior with AudioSocket read errors.

### GCP/AWS Handler Changes

Change `gcpRun` and `awsRun` signatures from:
```go
func (h *streamingHandler) gcpRun(st *streaming.Streaming, conn net.Conn) error
```
to:
```go
func (h *streamingHandler) gcpRun(st *streaming.Streaming) error
```

The connection is accessed via `st.ConnAst`. The media processing functions replace
`audiosocketGetNextMedia(conn)` with `st.ConnAst.ReadMessage()`.

### What Gets Removed

- `audiosocket.go` — AudioSocket protocol helpers (entire file)
- `runKeepAlive()`, `retryWithBackoff()` — AudioSocket keep-alive mechanism
- `runStart()` — TCP connection handler (logic moves to Start)
- TCP listener in `Run()` — replaced by no-op
- `listenAddress` field on handler struct
- `POD_IP` validation and `listenAddress` construction in `main.go`
- `github.com/CyCoreSystems/audiosocket` dependency

### What Gets Added

- `websocket.go` — `websocketConnect()` and `runWebSocketRead()` (ported from tts-manager)
- `github.com/gorilla/websocket` dependency

### Config Impact

- `POD_IP` — no longer required for streaming (only used for listen address)
- `STREAMING_LISTEN_PORT` — no longer used
- Both config values can remain in the config struct for backwards compatibility but
  the `PodIP == ""` validation in `main.go` should be removed

### Test Impact

- `start_test.go` — rewrite: `Start()` now also dials WebSocket and spawns STT. Test
  scope changes to verify ExternalMediaStart call with new parameters (INCOMING host,
  websocket transport, none encapsulation)
- `run_test.go` — delete `runKeepAlive` test and `MockConn`, `Run()` is now a no-op
- `gcp_test.go` — `gcpProcessResult` tests unaffected (they test STT result processing).
  `gcpProcessMedia` tests need WebSocket mock
- `aws_test.go` — same as gcp_test.go
- `main_test.go` — update `NewStreamingHandler` tests (remove listenAddress param)

## Files Changed

| File | Change |
|---|---|
| `models/streaming/streaming.go` | Add ConnAst, ConnAstDone fields |
| `pkg/streaminghandler/main.go` | Update constants, remove listenAddress, add websocket import |
| `pkg/streaminghandler/websocket.go` | New: websocketConnect, runWebSocketRead |
| `pkg/streaminghandler/start.go` | Rewrite: dial WebSocket, spawn STT processing |
| `pkg/streaminghandler/run.go` | Run() becomes no-op, remove runStart/keepalive |
| `pkg/streaminghandler/gcp.go` | Read from st.ConnAst instead of net.Conn param |
| `pkg/streaminghandler/aws.go` | Read from st.ConnAst instead of net.Conn param |
| `pkg/streaminghandler/stop.go` | Add conn.Close() |
| `pkg/streaminghandler/streaming.go` | No changes |
| `pkg/streaminghandler/audiosocket.go` | Delete |
| `cmd/transcribe-manager/main.go` | Remove POD_IP validation, listenAddress, update constructor |
| `internal/config/main.go` | No changes (keep fields for backwards compat) |
| Tests | Rewrite start_test, delete run_test keepalive, update gcp/aws media tests |
