# Fix Pipecat Audio Quality Issues

## Problem

Poor sound quality during AI calls traced to several bugs in pipecat-manager's
media stream handling code (Go side).

## Approach

Fix four issues in the Go audio pipeline:

1. **Fatal error on audio write terminates output handler** — A single transient
   write error to Asterisk kills all TTS audio for the remainder of the call.
   Fix: log and continue instead of returning.

2. **Silent frame dropping with `time.After` GC pressure** — `pushFrame()` uses
   `time.After` per frame (~50/sec), leaking timers until they fire. Frames are
   dropped silently with no monitoring. Fix: non-blocking fast path + reusable
   timer slow path + periodic drop logging via atomic counter on Session.

3. **No frame drop monitoring** — When audio frames are dropped due to channel
   backpressure, there is no logging or metrics. Fix: add atomic counter to
   Session, log periodically, and log total on session stop.

4. **Pong routed through audio channel** — Input receiver sends Pong via
   `SendData` which goes through the audio queue. In practice this is dead code
   (gorilla/websocket handles Ping/Pong internally via concurrent-safe
   `WriteControl`), but it is misleading and fragile. Fix: remove the `SendData`
   call and add a clarifying comment.

## Files Changed

- `models/pipecatcall/session.go` — Add `DroppedFrames atomic.Int64` field
- `pkg/pipecatcallhandler/pipecatframe.go` — Rewrite `pushFrame` with fast path
  and proper timer; add drop logging
- `pkg/pipecatcallhandler/runner.go` — Log+continue on audio error instead of
  return; clean up Pong handling
- `pkg/pipecatcallhandler/session.go` — Log total dropped frames on session stop

## Resampling Investigation

Confirmed that no resampling occurs anywhere in the pipeline under normal
operation. `audio_out_sample_rate=16000` in Python ensures all TTS providers
generate 16kHz natively. Go's `GetDataSamples` resampler is never invoked.
