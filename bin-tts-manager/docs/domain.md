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

## Key Business Rules Summary

| Rule | Details |
|------|---------|
| Provider fallback | GCP → AWS Polly; `speech_fallback_total` tracks fallbacks |
| Per-pod routing | Use `HOSTNAME` as `host_id`; route streaming RPCs to per-pod queue |
| Session isolation | Each streaming session owns a goroutine + ElevenLabs WebSocket + AudioSocket connection |
| Keep-alive | 30-second pings; failure cleans up session |
| No event subscriptions | TTS is invoked synchronously via RPC; no SubscribeHandler |
