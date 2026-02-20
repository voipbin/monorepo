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
4. `Start()` spawns `runSTT()` goroutine (provider selection + handler)
5. `Start()` returns immediately; STT handler runs until WebSocket closes

No separate read goroutine is needed. The STT media processor's `ReadMessage()` loop
is the sole reader on the connection — gorilla/websocket handles ping/pong/close frames
internally within any `ReadMessage()` call, unlike the tts-manager which needs a dedicated
`runWebSocketRead()` because its main loop writes to the WebSocket.

The key difference from the old flow: STT processing is triggered from `Start()` instead
of from the TCP listener. The provider selection logic moves from `runStart()` into
`runSTT()`.

### WebSocket Protocol

Matching the tts-manager's implementation:

- **Subprotocol**: `"media"` (chan_websocket)
- **Handshake**: Asterisk sends `MEDIA_START` text message after connection
- **Audio data**: Binary frames containing raw slin PCM bytes
- **Close detection**: The STT media processor's `ReadMessage()` loop is the sole reader;
  gorilla/websocket handles ping/pong/close frames internally within `ReadMessage()`.
  When the connection closes, `ReadMessage()` returns an error and the loop exits.
- **Cleanup**: `Stop()` calls `ExternalMediaStop`, then `conn.Close()` to release the
  file descriptor and unblock the media processor's `ReadMessage()` loop

### Error Handling

If `websocketConnect()` fails after `ExternalMediaStart` succeeds, clean up both the
orphaned external media channel (via `ExternalMediaStop`) and the in-memory streaming
record (via `h.Delete()`). This prevents stale "started" entries from remaining in the
map.

### Streaming Model Changes

Add to `models/streaming/streaming.go`:
- `ConnAst *websocket.Conn` — WebSocket connection to Asterisk

No done channel is needed. Unlike the tts-manager (which uses `ConnAstDone` to signal its
write loop), the transcribe-manager has no write loop — the STT media processor's
`ReadMessage()` naturally unblocks with an error when the connection closes.

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

- `websocket.go` — `websocketConnect()` (adapted from tts-manager; `runWebSocketRead()` omitted
  because the STT media processor is the sole reader)
- `github.com/gorilla/websocket` dependency

### Config Impact

- `POD_IP` — no longer required for streaming (only used for listen address)
- `STREAMING_LISTEN_PORT` — no longer used
- Both config values can remain in the config struct for backwards compatibility but
  the `PodIP == ""` validation in `main.go` should be removed

### Test Impact

- `start_test.go` — rewrite: tests the error path where `websocketConnect()` fails.
  Verifies ExternalMediaStart is called with new parameters (`INCOMING` host, `websocket`
  transport, `none` encapsulation), and verifies cleanup of both the orphaned external
  media (`ExternalMediaStop`) and the streaming record (`Delete`) on failure
- `run_test.go` — delete `runKeepAlive` test and `MockConn`, replace with `Run()` no-op test
- `gcp_test.go` — no changes needed (existing tests cover `gcpProcessResult` only)
- `aws_test.go` — no changes needed (existing tests cover `awsProcessResult` only)
- `main_test.go` — update `NewStreamingHandler` tests (remove listenAddress param)

## Files Changed

| File | Change |
|---|---|
| `models/streaming/streaming.go` | Add ConnAst field |
| `pkg/streaminghandler/main.go` | Update constants, remove listenAddress, remove keep-alive constants |
| `pkg/streaminghandler/websocket.go` | New: websocketConnect |
| `pkg/streaminghandler/start.go` | Rewrite: dial WebSocket, spawn runSTT, cleanup on failure |
| `pkg/streaminghandler/run.go` | Run() becomes no-op, remove runStart/keepalive |
| `pkg/streaminghandler/gcp.go` | Read from st.ConnAst instead of net.Conn param |
| `pkg/streaminghandler/aws.go` | Read from st.ConnAst instead of net.Conn param |
| `pkg/streaminghandler/stop.go` | Add conn.Close() to release fd and unblock ReadMessage |
| `pkg/streaminghandler/audiosocket.go` | Delete |
| `cmd/transcribe-manager/main.go` | Remove POD_IP validation, listenAddress, update constructor |
| `cmd/transcribe-control/main.go` | Remove listenAddress placeholder, update constructor |
| `internal/config/main.go` | No changes (keep fields for backwards compat) |
| Tests | Rewrite start_test (error path + cleanup), delete run_test keepalive, update main_test |
