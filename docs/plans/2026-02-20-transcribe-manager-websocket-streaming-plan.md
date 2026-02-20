# Transcribe-Manager WebSocket Streaming — Implementation Plan

Reference: `docs/plans/2026-02-20-transcribe-manager-websocket-streaming-design.md`

## Step 1: Update streaming model

Add WebSocket connection fields to `models/streaming/streaming.go`:
- `ConnAst *websocket.Conn` (json:"-")
- `ConnAstDone chan struct{}` (json:"-")

Add `github.com/gorilla/websocket` import.

Verify: `go build ./models/...`

## Step 2: Add websocket.go

Create `pkg/streaminghandler/websocket.go` with two functions ported from
`bin-tts-manager/pkg/streaminghandler/websocket.go`:

- `websocketConnect(ctx, mediaURI)` — dials WebSocket with `"media"` subprotocol, waits
  for MEDIA_START text message, returns `*websocket.Conn`
- `runWebSocketRead(conn, doneCh)` — read loop for ping/pong/close handling, closes
  `doneCh` when connection drops

Only port the connect and read functions. Do NOT port `websocketWrite` (not needed for
STT). Adapt imports for transcribe-manager package paths.

Verify: `go build ./pkg/streaminghandler/...`

## Step 3: Update external media constants in main.go

In `pkg/streaminghandler/main.go`:

- Change `defaultEncapsulation` from `EncapsulationAudioSocket` to `EncapsulationNone`
- Change `defaultTransport` from `TransportTCP` to `TransportWebsocket`
- Change `defaultConnectionType` from `"client"` to `"server"`
- Keep `defaultFormat` as `"slin"`
- Remove `listenAddress` field from `streamingHandler` struct
- Remove `listenAddress` parameter from `NewStreamingHandler` constructor
- Remove `audiosocket` and `net` imports, add `gorilla/websocket` if needed
- Remove keep-alive constants (`defaultKeepAliveInterval`, `defaultMaxRetryAttempts`,
  `defaultInitialBackoff`)

Verify: compilation will fail until later steps update callers — that's expected.

## Step 4: Rewrite start.go

Rewrite `pkg/streaminghandler/start.go` to:

1. Create streaming record (unchanged)
2. Call `CallV1ExternalMediaStart` with:
   - `externalHost`: `"INCOMING"` (instead of `h.listenAddress`)
   - Updated constants (none/websocket/server)
   - `directionListen` and `directionSpeak` stay the same
3. Dial WebSocket using `em.MediaURI` via `websocketConnect(ctx, em.MediaURI)`
4. On WebSocket connect failure: call `ExternalMediaStop` to clean up orphaned channel
5. Initialize `st.ConnAstDone = make(chan struct{})`
6. Store WebSocket connection on streaming record: `st.ConnAst = conn`
7. Spawn `go runWebSocketRead(conn, st.ConnAstDone)`
8. Spawn `go h.runSTT(st)` — new helper that runs STT provider selection and execution

Add `runSTT(st)` helper method that contains the provider selection logic currently in
`runStart()` (from run.go lines 70-94):
- Build handlers list based on `h.providerPriority`
- Try each handler in order, continue on failure
- Handlers no longer take `net.Conn` parameter

Verify: `go build ./pkg/streaminghandler/...`

## Step 5: Rewrite run.go

In `pkg/streaminghandler/run.go`:

- `Run()` becomes a no-op: `return nil`
- Delete `runStart()` (logic moved to start.go's `runSTT`)
- Delete `runKeepAlive()`
- Delete `retryWithBackoff()`
- Remove `net`, `time` imports

Verify: `go build ./pkg/streaminghandler/...`

## Step 6: Update gcp.go

In `pkg/streaminghandler/gcp.go`:

- Change `gcpRun` signature from `(st *streaming.Streaming, conn net.Conn) error` to
  `(st *streaming.Streaming) error`
- Change `gcpProcessMedia` signature: remove `conn net.Conn` parameter
- In `gcpProcessMedia`, replace `h.audiosocketGetNextMedia(conn)` with WebSocket read:
  ```go
  msgType, data, err := st.ConnAst.ReadMessage()
  if err != nil { return }
  if msgType != websocket.BinaryMessage { continue }
  ```
- Send `data` (raw bytes) to GCP instead of `m.Payload()`
- Remove `net` import, add `gorilla/websocket` import

Verify: `go build ./pkg/streaminghandler/...`

## Step 7: Update aws.go

In `pkg/streaminghandler/aws.go`:

- Change `awsRun` signature from `(st *streaming.Streaming, conn net.Conn) error` to
  `(st *streaming.Streaming) error`
- Change `awsProcessMedia` signature: remove `conn net.Conn` parameter
- In `awsProcessMedia`, replace `h.audiosocketGetNextMedia(conn)` with WebSocket read
  (same pattern as gcp.go)
- Send `data` (raw bytes) to AWS instead of `m.Payload()`
- Remove `net` import, add `gorilla/websocket` import

Verify: `go build ./pkg/streaminghandler/...`

## Step 8: Update stop.go

In `pkg/streaminghandler/stop.go`:

After `CallV1ExternalMediaStop`, add WebSocket close:
```go
if st.ConnAst != nil {
    _ = st.ConnAst.Close()
}
```

This matches the tts-manager pattern. Closing the WebSocket unblocks `runWebSocketRead`
which closes `ConnAstDone`.

Verify: `go build ./pkg/streaminghandler/...`

## Step 9: Delete audiosocket.go

Delete `pkg/streaminghandler/audiosocket.go` entirely. The `audiosocketGetStreamingID`
and `audiosocketGetNextMedia` functions are no longer used.

Verify: `go build ./pkg/streaminghandler/...`

## Step 10: Update cmd/transcribe-manager/main.go

- Remove the `POD_IP` validation block (lines 138-140)
- Remove `listenAddress` construction (line 141)
- Update `NewStreamingHandler` call to remove `listenAddress` parameter
- Remove the `listenAddress` log line

Verify: `go build ./cmd/...`

## Step 11: Update tests

### start_test.go
- Remove `listenAddress` from test cases and handler construction
- Update `CallV1ExternalMediaStart` expectations:
  - externalHost: `"INCOMING"` instead of `tt.listenAddress`
  - defaultEncapsulation: `"none"` (was `"audiosocket"`)
  - defaultTransport: `"websocket"` (was `"tcp"`)
  - defaultConnectionType: `"server"` (was `"client"`)
- Test now verifies only the ExternalMediaStart call since WebSocket connect
  will fail in unit tests (no real server). Adjust test expectations accordingly.

### run_test.go
- Delete `MockConn` struct and `Test_runKeepAlive` test entirely
- If file becomes empty, delete the file

### gcp_test.go
- `gcpProcessResult` tests: no changes needed (they don't touch the connection)
- If `gcpProcessMedia` tests exist, update to use WebSocket mock

### aws_test.go
- `awsProcessEvents` tests: no changes needed
- If `awsProcessMedia` tests exist, update to use WebSocket mock

### main_test.go
- Update `NewStreamingHandler` calls to remove `listenAddress` parameter

Verify: `go test ./...`

## Step 12: Update dependencies

```bash
cd bin-transcribe-manager
go mod tidy && go mod vendor
```

This should:
- Add `github.com/gorilla/websocket` (already in monorepo via tts-manager)
- Remove `github.com/CyCoreSystems/audiosocket` if no other imports reference it

Verify: `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`

## Step 13: Update CLAUDE.md

Update `bin-transcribe-manager/CLAUDE.md`:
- Change AudioSocket references to WebSocket
- Update config documentation: `POD_IP` and `STREAMING_LISTEN_PORT` no longer required
  for streaming
- Update streaming handler description
