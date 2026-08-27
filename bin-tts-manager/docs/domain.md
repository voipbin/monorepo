# bin-tts-manager Domain

## Domain Entities

### Speech (Batch TTS)

A pre-recorded audio file generated from text and stored locally. Served via Python HTTP sidecar.

| Field | Type | Description |
|-------|------|-------------|
| `id` | UUID | Primary key |
| `customer_id` | UUID | Owning tenant |
| `text` | string | Input text for synthesis |
| `language` | string | BCP47 language code |
| `voice` | string | Voice identifier (provider-specific) |
| `gender` | string | Voice gender |
| `file_path` | string | Path on `/shared-data` volume |
| `url` | string | HTTP URL served by Python sidecar |
| `tm_create` | timestamp | Creation time |

### Streaming Session

An in-memory real-time TTS session anchored to one pod. Coordinates ElevenLabs WebSocket ↔ AudioSocket frame delivery.

| Field | Type | Description |
|-------|------|-------------|
| `id` | UUID | Session identifier |
| `customer_id` | UUID | Owning tenant |
| `host_id` | string | `HOSTNAME` of the owning pod — used for per-pod queue routing |
| `pod_ip` | string | Pod IP — used to advertise AudioSocket endpoint to Asterisk |
| `status` | enum | Session status (active, stopped, etc.) |
| `language` | string | BCP47 language code |

## Key Business Rules

### Provider Selection (Batch Mode)

The `audiohandler` attempts TTS providers in sequence:
1. **Google Cloud TTS** (primary): Uses Application Default Credentials (ADC). Regional endpoint: `eu-texttospeech.googleapis.com:443`.
2. **AWS Polly** (fallback): Uses `aws_access_key` and `aws_secret_key` credentials.

If the primary provider fails, the fallback is tried automatically. The `speech_fallback_total` metric counts fallback invocations.

### Streaming Session Lifecycle

1. **Create**: Client calls `POST /v1/streamings` (on shared queue) — returns session ID and `host_id`.
2. **Control**: Client routes subsequent control RPCs (`say_init`, `say_add`, `say_stop`, `say_finish`) to the per-pod queue for the owning pod's `host_id`.
3. **Audio delivery**: Asterisk dials into `pod_ip:8080` via AudioSocket; the streaming handler routes frames to the ElevenLabs WebSocket for the session.
4. **Cleanup**: Session destroyed on `say_finish` or explicit stop; goroutine and WebSocket are closed.

### Per-Pod Session Anchoring

Streaming sessions live in a mutex-protected in-memory map on the pod that created them. Follow-up RPCs must be routed to the same pod via the per-pod queue (`bin-manager.tts-manager.request.<HOSTNAME>`). The `host_id` field on the session record tells callers which queue to use.

### Keep-Alive Management

Streaming sessions send keep-alive pings every 30 seconds via AudioSocket protocol. If the ElevenLabs WebSocket disconnects, the session is cleaned up and the error is recorded in `streaming_error_total`.

### Audio File Serving

Batch audio files are written to `/shared-data` by the Go service. The Python HTTP sidecar (port 80) serves these files to callers. The shared volume is the only coupling between the two containers.

### Events Published

Every event below is published to the per-service fanout exchange `bin-manager.tts-manager.event`, and — since VOIP-1405 — also to the global topic exchange `bin-manager.event` with the routing key `tts-manager.<resource>.<subscription-id>.<action>`. The third key segment is the *subscription address*: for the `streaming` and `message` namespaces it is always the **streaming-id**, so one TTS session is followed with `tts-manager.streaming.<streaming-id>.#` plus `tts-manager.message.<streaming-id>.#`. The `speaking` namespace is an independent persistent resource addressed by its own id.

| Event | Data | Trigger | Topic routing key |
|-------|------|---------|-------------------|
| `speaking.EventTypeSpeakingStarted` | `*speaking.Speaking` | Speaking session moved to `active` | `tts-manager.speaking.<speaking-id>.started` |
| `speaking.EventTypeSpeakingStopped` | `*speaking.Speaking` | Speaking session stopped | `tts-manager.speaking.<speaking-id>.stopped` |
| `streaming.EventTypeStreamingCreated` | `*streaming.Streaming` | Streaming session registered in the session map | `tts-manager.streaming.<streaming-id>.created` |
| `streaming.EventTypeStreamingDeleted` | `*streaming.Streaming` | Streaming session removed from the session map | `tts-manager.streaming.<streaming-id>.deleted` |
| `message.EventTypeInitiated` | `*message.Message` | Vendor streamer initialized for an utterance | `tts-manager.message.<streaming-id>.initiated` |
| `message.EventTypePlayStarted` | `*message.Message` | Synthesized audio started playing | `tts-manager.message.<streaming-id>.play_started` |
| `message.EventTypePlayFinished` | `*message.Message` | Synthesized audio finished playing | `tts-manager.message.<streaming-id>.play_finished` |

`*message.Message` implements `eventtopic.SubscriptionIdentifier` (pointer receiver) returning `StreamingID`. Its own id is captured once at streamer init and is never refreshed, so from the second utterance of a session on it diverges from `streaming.MessageID` and addresses nothing. `Streaming` and `Speaking` need no override — their own ids already are the subscription address.

`streaming.EventTypeStreamingFinished`, `streaming.EventTypeStreamingPlayStarted` and `streaming.EventTypeStreamingPlayFinished` are declared in `models/streaming/event.go` but have no publish site — they are dead constants and are deliberately absent from the table above and from the golden routing-key table in `models/streaming/routingkey_golden_test.go`. Their live twins are the `message_play_*` types (published by all three vendor backends: gcp, aws, elevenlabs).

## Key Business Rules Summary

| Rule | Details |
|------|---------|
| Provider fallback | GCP → AWS Polly; `speech_fallback_total` tracks fallbacks |
| Per-pod routing | Use `HOSTNAME` as `host_id`; route streaming RPCs to per-pod queue |
| Session isolation | Each streaming session owns a goroutine + ElevenLabs WebSocket + AudioSocket connection |
| Keep-alive | 30-second pings; failure cleans up session |
| No event subscriptions | TTS is invoked synchronously via RPC; no SubscribeHandler |
