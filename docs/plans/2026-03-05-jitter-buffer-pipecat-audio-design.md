# Go-Side Jitter Buffer for Pipecat Audio Forwarding

## Date
2026-03-05

## Problem
Test call `eed6cce7-e4a1-495f-ade1-254ec3463c6c` exhibited choppy AI voice (~500ms cutting pattern). Root cause:
- Python pipecat's team pipeline runs 9 gRPC connections on a single asyncio event loop
- Event loop contention causes irregular audio frame delivery timing
- The Go audio forwarder (`runnerWebsocketHandleAudio`) writes directly to Asterisk with zero buffering
- Asterisk conference bridge reads at fixed 20ms intervals — missing data = silence mixed in

## Solution (v1 — simple jitter buffer)
Initial attempt: small jitter buffer draining at 20ms cadence. Did NOT fix the issue.

### Why v1 failed
Comparison with `bin-tts-manager` (which has good audio quality) revealed:
- TTS-manager: ElevenLabs delivers audio **faster than real-time** → `websocketWrite()` always has data → no gaps possible
- Pipecat-manager: audio arrives from Python at roughly real-time rate (or slower due to asyncio contention) → jitter buffer drains at real-time rate → buffer runs dry when production dips below real-time → Asterisk gets silence → choppy

A jitter buffer that drains at real-time rate cannot help when the audio **production rate** is the bottleneck, not timing irregularities.

## Solution (v2 — pre-fill jitter buffer)
Adopt tts-manager's approach: **pre-fill the buffer before starting to drain**, creating a head-start that absorbs production-rate dips.

### Design
- `AudioJitterBuffer`: mutex-protected byte buffer with max capacity of 1s (32000 bytes at 16kHz mono 16-bit)
- **Pre-fill threshold**: 200ms (6400 bytes / 10 chunks) must accumulate before drain starts
- Write path: `runnerWebsocketHandleAudio` writes audio into the jitter buffer
- Drain path: `runJitterBufferDrain` goroutine:
  1. Waits for Asterisk connection
  2. Waits for pre-fill threshold (polls every 5ms)
  3. Writes first chunk **immediately** (no tick wait, matching tts-manager pattern)
  4. Then drains at fixed 20ms cadence
- Overflow: if buffer exceeds max, oldest bytes are discarded
- Non-call reference types fall back to the original direct-write path

### Files Changed
| File | Action |
|------|--------|
| `bin-pipecat-manager/models/pipecatcall/jitterbuffer.go` | **Created** — AudioJitterBuffer with pre-fill support |
| `bin-pipecat-manager/models/pipecatcall/jitterbuffer_test.go` | **Created** — unit tests including pre-fill tests |
| `bin-pipecat-manager/models/pipecatcall/session.go` | **Edited** — added JitterBuffer field to Session |
| `bin-pipecat-manager/pkg/pipecatcallhandler/runner.go` | **Edited** — audio handler writes to buffer; added runJitterBufferDrain with pre-fill |
| `bin-pipecat-manager/pkg/pipecatcallhandler/start.go` | **Edited** — init buffer + start drain goroutine in both call paths |
| `bin-pipecat-manager/pkg/pipecatcallhandler/session.go` | **Edited** — log + reset buffer in SessionStop cleanup |

### Verification
- All tests pass (`go test ./...`)
- No new lint issues (`golangci-lint run -v --timeout 5m`)
- Production verification: deploy and test call to confirm voice quality improvement
