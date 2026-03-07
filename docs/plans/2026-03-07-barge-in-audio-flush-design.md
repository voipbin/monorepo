# Barge-In Audio Flush Design

## Problem

When a user speaks during AI TTS playback (barge-in), the AI voice doesn't stop immediately despite Pipecat detecting the interruption via VAD. The root cause: `UnpacedWebsocketClientOutputTransport` delivers TTS audio faster than real-time to Asterisk's `chan_websocket`, which buffers it internally. By the time Pipecat cancels its output task, Asterisk already has seconds of audio queued for playback. There's no mechanism to flush that buffer.

## Approaches Considered and Rejected

### 1. Send silence to Asterisk on barge-in (REJECTED)

Override `process_frame()` in `UnpacedWebsocketClientOutputTransport` to send a `TextFrame("flush_audio")` to Go on `InterruptionFrame`. Go then writes silence frames to Asterisk.

**Why it failed:** Asterisk's `chan_websocket` uses a playback queue, not a fixed-size ring buffer. Sending silence just appends to the queue — it doesn't replace buffered audio. The user hears `[stale TTS] → [silence] → [new TTS]` instead of immediate cutoff.

Additional issue: a single large silence write (16000 bytes) caused Asterisk to close the WebSocket with code 1003 (unsupported data). Chunking into 640-byte frames (20ms slin16) fixed the crash but didn't solve the fundamental buffering problem.

### 2. Go-side jitter buffer with flush (REJECTED)

Buffer audio in Go and deliver to Asterisk at real-time rate. On flush, discard the Go-side buffer.

**Why rejected:** Adds significant complexity (audio queue, writer goroutine, timing logic) and ~100ms initial latency. Asterisk 23's `chan_websocket` already has built-in buffering and re-timing with a native `FLUSH_MEDIA` command, making a Go-side buffer redundant.

### 3. Pipecat standard paced transport (REJECTED)

Remove unpaced transport entirely and use Pipecat's standard `WebsocketClientTransport` with built-in `_write_audio_sleep()` pacing.

**Why rejected:** Pipecat's asyncio event loop causes timing drift in `_write_audio_sleep()` during CPU contention (LLM callbacks, STT processing, GC). This produces audible audio gaps/cutting. The problem is well-documented in Pipecat Issue #3222.

## Solution: Unpaced Transport + Asterisk FLUSH_MEDIA

Keep `UnpacedWebsocketClientOutputTransport` (no audio gaps) and use Asterisk's native `FLUSH_MEDIA` WebSocket text command for barge-in (instant audio stop).

### How it works

1. Pipecat sends TTS audio faster than real-time via unpaced transport → Go forwards immediately → Asterisk buffers and re-times playback internally
2. On barge-in: Pipecat's VAD detects speech → `InterruptionFrame` → Pipecat cancels its audio task (clears internal queue)
3. The output transport override sends a `TextFrame("FLUSH_MEDIA")` to Go on `InterruptionFrame`
4. Go receives the TextFrame, sends `FLUSH_MEDIA` as a WebSocket text message to Asterisk
5. Asterisk discards all queued audio immediately → audio stops

### Why this works

- **No audio gaps**: Unpaced delivery means Pipecat doesn't need precise asyncio timing; Asterisk handles re-timing
- **Instant barge-in**: `FLUSH_MEDIA` is a native `chan_websocket` command that discards the entire playback queue
- **Minimal complexity**: Small Python override + one Go handler for the text command

### Asterisk chan_websocket commands used

- `FLUSH_MEDIA` (text message TO Asterisk): Discards all queued but not-yet-played audio frames. Available in Asterisk 22.6.0+/23.0.0+. Bug fix for queue counter reset merged in 22.8.0+ (PR #1303).

### References

- [Asterisk chan_websocket docs](https://docs.asterisk.org/Configuration/Channel-Drivers/WebSocket/)
- [Pipecat Issue #3222 - Jitter Buffer Support](https://github.com/pipecat-ai/pipecat/issues/3222)
- [Asterisk Issue #1304 - FLUSH_MEDIA bug fix](https://github.com/asterisk/asterisk/issues/1304)

## Files Changed

1. `bin-pipecat-manager/scripts/pipecat/run.py` - Add `UnpacedWebsocketClientOutputTransport` (no-ops `_write_audio_sleep`, sends `FLUSH_MEDIA` on InterruptionFrame) and `UnpacedWebsocketClientTransport` wrapper; use unpaced transport for output direction
2. `bin-pipecat-manager/pkg/pipecatcallhandler/runner.go` - Handle `FLUSH_MEDIA` TextFrame by forwarding as text message to Asterisk WebSocket
3. `bin-pipecat-manager/CLAUDE.md` - Update audio delivery documentation
