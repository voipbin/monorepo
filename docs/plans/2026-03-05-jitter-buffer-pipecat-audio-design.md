# Fix Choppy AI Voice in Pipecat Audio Forwarding

## Date
2026-03-05

## Problem
Test call `eed6cce7-e4a1-495f-ade1-254ec3463c6c` exhibited choppy AI voice (~500ms cutting pattern). Root cause:
- Python pipecat's team pipeline runs 9 gRPC connections on a single asyncio event loop
- Event loop contention causes irregular audio frame delivery timing
- The Go audio forwarder (`runnerWebsocketHandleAudio`) writes directly to Asterisk with zero buffering
- Asterisk conference bridge reads at fixed 20ms intervals — missing data = silence mixed in

## Approach History

### v1 — Go-side jitter buffer (failed)
Added a simple jitter buffer draining at 20ms cadence. Did NOT fix the issue.

Comparison with `bin-tts-manager` (which has good audio quality) revealed:
- TTS-manager: ElevenLabs delivers audio **faster than real-time** → always has data → no gaps
- Pipecat-manager: audio arrives from Python at roughly real-time rate → buffer runs dry → choppy

A jitter buffer that drains at real-time rate cannot help when the audio **production rate** is the bottleneck.

### v2 — Go-side pre-fill jitter buffer (failed)
Added pre-fill threshold (200ms / 6400 bytes) before starting to drain, creating a head-start.

Diagnostic logs from test call `ab3baf37-5710-4ce0-a96b-0594f65b9e82` showed:
- Pre-fill reached 6400 bytes, but within 2s the buffer dropped to 1920 bytes
- During active TTS speech: 10-20% underrun rate (buffer at 640-1920 bytes)
- Pre-fill margin evaporated because Python delivers audio at real-time rate

Root cause: pipecat's `WebsocketClientOutputTransport._write_audio_sleep()` paces audio
at real-time rate — simulating speaker playback. This prevents the buffer from building up.

### v3 — Disable Python audio pacing + remove Go jitter buffer (final)
The true fix: disable `_write_audio_sleep()` so TTS audio is forwarded as fast as the TTS
generates it (faster than real-time), matching tts-manager's ElevenLabs pattern. Asterisk's
`chan_websocket` buffers incoming audio internally — proven by tts-manager which has no
jitter buffer and writes directly to Asterisk.

With faster-than-real-time delivery, no Go-side jitter buffer is needed. The entire jitter
buffer was removed, restoring the simple direct-write path.

## Final Solution

### Python: Subclass output transport to skip audio pacing
- `UnpacedWebsocketClientOutputTransport`: subclass that overrides `_write_audio_sleep` to no-op
- `UnpacedWebsocketClientTransport`: subclass that overrides `output()` to return the unpaced variant
- Used only for output direction (input still needs normal behavior for VAD)

### Go: Direct write to Asterisk (no jitter buffer)
- Audio from Python arrives faster than real-time
- Go writes each chunk directly to Asterisk via WebSocket (same as tts-manager pattern)
- Asterisk's chan_websocket handles buffering and timing internally

### Additional fix: Non-blocking PublishEvent
- Event publishing (transcriptions, LLM text) wrapped in goroutines to prevent RabbitMQ
  publish latency from stalling the audio frame ingestion loop

## Files Changed
| File | Action |
|------|--------|
| `bin-pipecat-manager/scripts/pipecat/run.py` | **Edited** — subclass output transport to skip audio pacing |
| `bin-pipecat-manager/pkg/pipecatcallhandler/runner.go` | **Edited** — async PublishEvent, removed jitter buffer code |
| `bin-pipecat-manager/pkg/pipecatcallhandler/runner_test.go` | **Edited** — tests for async PublishEvent |
| `bin-pipecat-manager/models/pipecatcall/jitterbuffer.go` | **Deleted** |
| `bin-pipecat-manager/models/pipecatcall/jitterbuffer_test.go` | **Deleted** |
| `bin-pipecat-manager/models/pipecatcall/session.go` | **Reverted** — removed JitterBuffer field |
| `bin-pipecat-manager/pkg/pipecatcallhandler/start.go` | **Reverted** — removed jitter buffer init and drain goroutine |
| `bin-pipecat-manager/pkg/pipecatcallhandler/session.go` | **Reverted** — removed jitter buffer cleanup |

## Verification
- All Go tests pass (`go test ./...`)
- No new lint issues (`golangci-lint run -v --timeout 5m`)
- Production verification: deploy and test call to confirm voice quality improvement
