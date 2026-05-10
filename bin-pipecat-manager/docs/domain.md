# bin-pipecat-manager Domain Model

## Core Concepts

### Pipecatcall
A single AI voice session. Has a MySQL record (persistent) plus an in-memory session (bound to one pod).

Key fields:
- `HostID` — pod IP (`POD_IP` from K8s Downward API); used for per-pod RabbitMQ routing
- `ReferenceType` / `ReferenceID` — the resource being served (call, conversation, task)
- `Status` — session lifecycle state

**Per-pod ownership:** every pipecatcall is anchored to exactly one pod. Follow-up operations (stop, message-send, ping) must be routed to `bin-manager.pipecat-manager.request.<HostID>`.

### Session (in-memory)
Runtime state held in `pkg/pipecatcallhandler/session.go`:
- `ConnAst` — Asterisk WebSocket connection
- `ConnAstDone` — channel closed on Asterisk disconnect; drives cleanup
- Python pipeline handle

### Protobuf Frames
All WebSocket messages between Go and Python use protobuf frames (`proto/frames.proto`):

| Frame type | Direction | Purpose |
|-----------|-----------|---------|
| `AudioRawFrame` | bidirectional | Raw PCM audio samples (16 kHz slin16) |
| `TextFrame` | Go → Asterisk | Control messages (e.g., `FLUSH_MEDIA` for barge-in) |
| `TranscriptionFrame` | Python → Go | STT transcript results |
| `MessageFrame` | bidirectional | Structured message payloads |

## Pipecat Pipeline

Python `run.py` constructs the pipeline:

```
Asterisk audio (via Go WebSocket)
    → VAD (Silero Voice Activity Detection)
    → STT (Deepgram / Whisper)
    → LLM (OpenAI / Grok / Gemini)
    → TTS (Cartesia / ElevenLabs / Google)
    → audio back to Asterisk
```

LLM providers supported (configured by bin-ai-manager at session start):
- `openai.gpt-4o`, `grok.grok-3`, `grok.grok-3-mini`
- `gemini.gemini-2.5-flash`, `gemini.gemini-1.5-pro`
- Others via RTVI protocol

STT providers: Deepgram, Whisper
TTS providers: Cartesia, ElevenLabs, Google

## Tool Execution

When the LLM emits a function call:
1. Python sends HTTP request to Go `httpHandler.RunnerToolHandle`
2. Go sends RPC request to `bin-ai-manager` (`POST /v1/aicalls/<uuid>/tool_execute`)
3. AI Manager executes the tool and returns the result
4. Go returns result to Python → injected into LLM context

## Audio Sample Rate (CRITICAL)

- Target sample rate: **16 kHz** end-to-end
- Pipecat default is 24 kHz — `audio_out_sample_rate=16000` in `PipelineParams` is **mandatory**
- Without this setting, per-chunk resampling creates audible boundary artifacts (robotic audio)
- See [docs/plans/2026-01-22-audio-resampling-design.md](plans/2026-01-22-audio-resampling-design.md) for background

## ConnAstDone Pattern

`runAsteriskReceivedMediaHandle` goroutine closes the `ConnAstDone` channel on Asterisk WebSocket disconnect. The lifecycle monitor goroutine waits on `ConnAstDone` and triggers full cleanup (Python pipeline stop, DB record update). This ensures sessions are torn down on actual hangup even if the stop RPC is never received.
