# Go-Side Jitter Buffer for Pipecat Audio Forwarding

## Date
2026-03-05

## Problem
Test call `eed6cce7-e4a1-495f-ade1-254ec3463c6c` exhibited choppy AI voice (~500ms cutting pattern). Root cause:
- Python pipecat's team pipeline runs 9 gRPC connections on a single asyncio event loop
- Event loop contention causes irregular audio frame delivery timing
- The Go audio forwarder (`runnerWebsocketHandleAudio`) writes directly to Asterisk with zero buffering
- Asterisk conference bridge reads at fixed 20ms intervals — missing data = silence mixed in

## Solution
Add a small jitter buffer on the Go side to absorb Python timing irregularities, draining at a fixed 20ms cadence matching Asterisk's mixing timer.

### Design
- `AudioJitterBuffer`: a mutex-protected byte buffer with max capacity of 500ms (16000 bytes at 16kHz mono 16-bit)
- Write path: `runnerWebsocketHandleAudio` writes resampled audio into the jitter buffer instead of directly to Asterisk
- Drain path: `runJitterBufferDrain` goroutine ticks every 20ms, reads 640-byte chunks (20ms at 16kHz), and writes them to the Asterisk WebSocket
- Overflow: if buffer exceeds max, oldest bytes are discarded (prevents unbounded growth)
- Non-call reference types (no Asterisk connection) fall back to the original direct-write path

### Files Changed
| File | Action |
|------|--------|
| `bin-pipecat-manager/models/pipecatcall/jitterbuffer.go` | **Created** — AudioJitterBuffer implementation |
| `bin-pipecat-manager/models/pipecatcall/jitterbuffer_test.go` | **Created** — unit tests |
| `bin-pipecat-manager/models/pipecatcall/session.go` | **Edited** — added JitterBuffer field to Session |
| `bin-pipecat-manager/pkg/pipecatcallhandler/runner.go` | **Edited** — audio handler writes to buffer; added runJitterBufferDrain |
| `bin-pipecat-manager/pkg/pipecatcallhandler/start.go` | **Edited** — init buffer + start drain goroutine in both call paths |
| `bin-pipecat-manager/pkg/pipecatcallhandler/session.go` | **Edited** — log + reset buffer in SessionStop cleanup |

### Verification
- All tests pass (`go test ./...`)
- No new lint issues (`golangci-lint run -v --timeout 5m`)
- Production verification: deploy and test call to confirm voice quality improvement
