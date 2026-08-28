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

## Events Published

Every event below is published to the per-service fanout exchange `bin-manager.pipecat-manager.event`, and — since VOIP-1405 — also to the global topic exchange `bin-manager.event` with the routing key `pipecat-manager.<resource>.<pipecatcall-id>.<action>`. The third key segment is the *subscription address*: it is always the pipecatcall-id, across all three resource namespaces, so one AI voice session is followed with `pipecat-manager.pipecatcall.<id>.#`, `pipecat-manager.message.<id>.#`, and `pipecat-manager.team.<id>.#`.

| Event | Data | Trigger | Topic routing key |
|-------|------|---------|-------------------|
| `pipecatcall.EventTypeCreated` | `*pipecatcall.Pipecatcall` | Pipecatcall record created | `pipecat-manager.pipecatcall.<pipecatcall-id>.created` |
| `pipecatcall.EventTypeInitialized` | `*pipecatcall.Pipecatcall` | Pipecat runner WebSocket connected — session ready for audio | `pipecat-manager.pipecatcall.<pipecatcall-id>.initialized` |
| `pipecatcall.EventTypePipecatcallTerminated` | `*pipecatcall.Pipecatcall` | Session torn down (published exactly once per pipecatcall) | `pipecat-manager.pipecatcall.<pipecatcall-id>.terminated` |
| `pipecatcall.EventTypeDeleted` | `*pipecatcall.Pipecatcall` | Pipecatcall record deleted | `pipecat-manager.pipecatcall.<pipecatcall-id>.deleted` |
| `message.EventTypeBotTranscription` | `*message.Message` | Bot speech transcribed | `pipecat-manager.message.<pipecatcall-id>.bot_transcription` |
| `message.EventTypeUserTranscription` | `*message.Message` | Final user STT result | `pipecat-manager.message.<pipecatcall-id>.user_transcription` |
| `message.EventTypeUserLLM` | `*message.Message` | User text delivered to the LLM | `pipecat-manager.message.<pipecatcall-id>.user_llm` |
| `message.EventTypeBotLLMIntermediate` | `*message.Message` | Per-tick delta of an in-flight LLM generation | `pipecat-manager.message.<pipecatcall-id>.bot_llm_intermediate` |
| `message.EventTypeBotLLM` | `*message.Message` | Final LLM reply for one generation | `pipecat-manager.message.<pipecatcall-id>.bot_llm` |
| `message.EventTypeTeamMemberSwitched` | `*message.MemberSwitchedEvent` | Team member transition during an AI call | `pipecat-manager.team.<pipecatcall-id>.member_switched` |

`Message` and `MemberSwitchedEvent` both implement `eventtopic.SubscriptionIdentifier` (pointer receiver) returning `PipecatcallID`. `Message` needs the override because its own id is not an address: the transcription and user-llm events mint a fresh uuid per event, while the bot-llm events reuse a per-generation id that no subscriber can know in advance. `MemberSwitchedEvent` needs it because it carries no top-level `id` at all — without the override the key would degrade to the `-` placeholder. `Pipecatcall` needs no override: its own id already is the subscription address.

The routing keys are pinned by `models/pipecatcall/routingkey_golden_test.go`; the override behavior (including the "address is never the own id" property) is pinned in `models/message`.

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
