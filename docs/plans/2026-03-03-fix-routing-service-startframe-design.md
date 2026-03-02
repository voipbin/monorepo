# Fix Routing Service StartFrame Initialization

**Date:** 2026-03-03
**Status:** Approved

## Problem

The routing services (`RoutingSTTService`, `RoutingTTSService`, `RoutingLLMService`) delegate
`process_frame()` directly to child services that are standalone objects, not part of the pipecat
pipeline. These child services never receive `StartFrame` or `setup()` through pipecat's normal
lifecycle, so their internal `TaskManager` is never initialized.

This causes `"GoogleSTTService#1 TaskManager is still not initialized"` errors at pipeline startup,
followed by a flood of `"StartFrame not received yet"` errors for every audio frame.

### Error Chain

1. Pipeline starts, `StartFrame` propagates through the pipeline
2. `WebsocketClientInputTransport` starts receiving audio from the Go WebSocket
3. Audio frames reach `RoutingSTTService`, which delegates to `GoogleSTTService`
4. `GoogleSTTService` has no `TaskManager` (never received `setup()`) and hasn't been started
   (never received `StartFrame`), so it fails
5. All subsequent audio frames also fail with "StartFrame not received yet"

## Approach: Lazy StartFrame Initialization (Approach C)

Initialize child services lazily — only the active service receives `StartFrame` at pipeline start.
When switching members, forward `StartFrame` to the new service before switching. This avoids idle
STT/TTS streaming connections that could time out on the provider side.

### Changes

#### 1. Override `setup()` to propagate to all children

Child services need `setup()` called to receive the `TaskManager` reference. Override `setup()` in
all three routing services to propagate the setup to all child services:

```python
async def setup(self, setup):
    await super().setup(setup)
    for svc in self._services.values():
        await svc.setup(setup)
```

This gives all children a valid `TaskManager` without starting them (no `StartFrame` yet).

#### 2. Handle StartFrame: forward to active child only

When `process_frame` receives a `StartFrame`:
- Store the frame and direction for later replay
- Forward to the active child service via `process_frame()`
- Track which children have been started (`_started_ids` set)
- Do NOT push `StartFrame` downstream explicitly — the child's monkey-patched `push_frame` handles it

#### 3. Handle EndFrame: forward to active child only

Forward `EndFrame` to the active child only. Inactive services clean up naturally when the
process ends. This avoids N duplicate `EndFrame` pushes downstream from the monkey-patched `push_frame`.

#### 4. Lazy start on `set_active_member()`

When switching to a new member:
- If the new member's service hasn't received `StartFrame` yet (not in `_started_ids`)
  AND we have a stored `StartFrame`, forward it to the new service before switching
- Mark the new service as started

#### 5. Regular frames: no change

Audio, transcription, and other frames delegate to the active child service as before.

### Files to Modify

1. `bin-pipecat-manager/scripts/pipecat/routing_stt.py`
2. `bin-pipecat-manager/scripts/pipecat/routing_tts.py`
3. `bin-pipecat-manager/scripts/pipecat/routing_llm.py`

### Imports Required

```python
from pipecat.frames.frames import Frame, StartFrame, EndFrame
```

### Risk Assessment

- **Low risk**: Changes are contained to the three routing service files
- **Latency on member switch**: ~tens of ms while new STT/TTS initializes — acceptable for live calls
- **Monkey-patched push_frame**: Relies on child services pushing system frames downstream, which is
  standard pipecat behavior
- **setup() availability**: Requires pipecat FrameProcessor to have a public `setup()` method —
  verified with pipecat-ai >= 0.0.101
