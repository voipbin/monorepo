# Fix TTS AudioSocket Silence Feed

## Problem

When a speaking/say request is made, the tts-manager's AudioSocket connection to Asterisk gets torn down almost immediately because Asterisk's `audiosocket_read` receives EAGAIN (no data available) and kills the channel.

### Error Trace

```
Asterisk: WARNING: res_audiosocket.c:282 Failed to read header from AudioSocket because: Resource temporarily unavailable
Asterisk: ERROR: chan_audiosocket.c:92 Failed to receive frame from AudioSocket server
tts-manager: runKeepConsume: Error reading from connection: EOF
tts-manager: runStreamer: Handler initialization failed: context canceled
tts-manager: runProcess: Could not write processed audio data to asterisk connection: use of closed network connection
```

### Root Cause

1. Asterisk connects to tts-manager via AudioSocket TCP
2. Asterisk's bridge media loop tries to read audio frames every ~20ms
3. tts-manager hasn't sent any data yet (still initializing ElevenLabs WebSocket)
4. Asterisk gets EAGAIN on read -> kills the AudioSocket channel -> closes TCP connection
5. tts-manager's `runKeepConsume` receives EOF -> cancels context
6. ElevenLabs WebSocket init fails due to canceled context
7. `runStart` returns -> defer closes conn
8. Later `SayInit` re-initializes vendor but uses the stale, closed `ConnAst`

The existing `runKeepAlive` sends a keepalive every 10 seconds, but Asterisk's media loop requires data every ~20ms.

### Comparison with pipecat-manager

pipecat-manager uses bidirectional external media (`DirectionIn` + `DirectionOut`), so Asterisk both sends and receives audio. This keeps the AudioSocket connection active through the write path even when the read path has no data yet. tts-manager uses output-only (`DirectionNone` + `DirectionOut`), meaning Asterisk only tries to read.

## Solution

Replace `runKeepAlive` with `runSilenceFeed` that sends proper 20ms silence frames at 20ms intervals continuously.

### Changes

**File: `bin-tts-manager/pkg/streaminghandler/run.go`**

1. Add `runSilenceFeed(ctx, cancel, conn)` function:
   - Sends 320 bytes of silence (160 samples x 2 bytes/sample = 20ms at 8kHz mono)
   - Wrapped in AudioSocket format via `audiosocketWrapDataPCM16Bit`
   - Interval: 20ms (matching Asterisk's media loop timing)
   - Stops on context cancellation or write error

2. In `runStart`: replace `go h.runKeepAlive(...)` with `go h.runSilenceFeed(...)`

3. Remove `runKeepAlive` and `retryWithBackoff` functions

**No changes to `audiosocket.go` or `elevenlabs.go`.**

### Silence Frame Format

- 320 bytes of zeros = 160 samples x 2 bytes/sample = 20ms at 8kHz 16-bit mono
- Wrapped by `audiosocketWrapDataPCM16Bit`: 1 byte format (0x10) + 2 bytes length (320) + 320 bytes data = 323 bytes total
- Sent every 20ms

### Concurrent Write Safety

When `audiosocketWrite` sends real audio, both silence feed and real audio write to the same TCP connection concurrently. This is safe because:
- Go's `net.Conn.Write` is goroutine-safe
- Each write is a complete AudioSocket frame (self-contained header + payload)
- Silence frames produce no audible sound, so interleaving is harmless
- Asterisk processes frames sequentially from its read buffer
