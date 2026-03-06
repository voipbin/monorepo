# Barge-In Audio Flush Design

## Problem

When a user speaks during AI TTS playback (barge-in), the AI voice doesn't stop immediately despite Pipecat detecting the interruption via VAD. The root cause: `UnpacedWebsocketClientOutputTransport` delivers TTS audio faster than real-time to Asterisk's `chan_websocket`, which buffers it internally. By the time Pipecat cancels its output task, Asterisk already has seconds of audio queued for playback. There's no mechanism to flush that buffer.

## Approaches Considered and Rejected

### 1. Send silence to Asterisk on barge-in (REJECTED)

Override `process_frame()` in `UnpacedWebsocketClientOutputTransport` to send a `TextFrame("flush_audio")` to Go on `InterruptionFrame`. Go then writes silence frames to Asterisk.

**Why it failed:** Asterisk's `chan_websocket` uses a playback queue, not a fixed-size ring buffer. Sending silence just appends to the queue — it doesn't replace buffered audio. The user hears `[stale TTS] → [silence] → [new TTS]` instead of immediate cutoff.

Additional issue: a single large silence write (16000 bytes) caused Asterisk to close the WebSocket with code 1003 (unsupported data). Chunking into 640-byte frames (20ms slin16) fixed the crash but didn't solve the fundamental buffering problem.

### 2. Go-side audio pacing with flush support (REJECTED)

Buffer audio in Go and deliver to Asterisk at real-time rate. On flush, discard the Go-side buffer.

**Why rejected:** Adds significant complexity (audio queue, writer goroutine, timing logic) when Pipecat already has built-in audio pacing that handles this correctly.

## Solution: Use Pipecat's Standard Paced Transport

Remove `UnpacedWebsocketClientOutputTransport` and `UnpacedWebsocketClientTransport` entirely. Use Pipecat's standard `WebsocketClientTransport` for both input and output directions.

With paced transport:
- Pipecat's `_write_audio_sleep()` delivers audio at real-time rate
- Asterisk has at most ~20ms of audio buffered at any time
- On barge-in, Pipecat's native `InterruptionFrame` handling cancels the output task
- Audio stops almost instantly — no flush mechanism needed

### Trade-off

The unpaced transport was originally introduced to avoid audio gaps caused by asyncio contention in Python. With `audio_out_sample_rate=16000` (matching Asterisk's slin16), each frame is smaller, reducing contention pressure. If audio gaps reappear, the fix should be Go-side pacing (approach #2 above), not disabling pacing entirely.

## Files Changed

1. `bin-pipecat-manager/scripts/pipecat/run.py` - Remove `UnpacedWebsocketClientOutputTransport`, `UnpacedWebsocketClientTransport`, and unpaced transport selection in `create_websocket_transport`
