# bin-transcribe-manager Domain

## Domain Entities

### Transcribe (Session)

A transcription session associating a call/conference/recording with an STT provider. Stored in MySQL.

| Field | Type | Description |
|-------|------|-------------|
| `id` | UUID | Primary key |
| `customer_id` | UUID | Owning tenant |
| `reference_type` | string | Type of resource being transcribed (call, conference, recording) |
| `reference_id` | UUID | ID of the resource being transcribed |
| `language` | string | BCP47 language code (e.g., `en-US`, `ko-KR`) |
| `direction` | enum | Audio direction: `in`, `out`, or `both` |
| `status` | enum | `progressing` or `done` |
| `host_id` | string | `POD_IP` of the owning pod — used for per-pod queue routing |
| `tm_create` | timestamp | Creation time |
| `tm_update` | timestamp | Last update time |
| `tm_delete` | timestamp | Soft-delete marker |

### Transcript (Segment)

An individual transcribed text segment from an STT provider.

| Field | Type | Description |
|-------|------|-------------|
| `id` | UUID | Primary key |
| `transcribe_id` | UUID | FK to parent session |
| `direction` | enum | `in` or `out` |
| `text` | string | Transcribed text |
| `start_time` | float | Segment start (seconds from call start) |
| `end_time` | float | Segment end (seconds) |
| `confidence` | float | Provider confidence score |
| `tm_create` | timestamp | When segment was received |

## Key Business Rules

### Provider Selection

At startup, all providers with valid credentials are initialized. At session creation time:
- Default order: `gcp` → `aws`
- Callers may pass a `provider` field (`"gcp"` or `"aws"`) to prefer a specific provider
- Fallback to the next available provider if preferred is unavailable

Both providers use 8 kHz, 16-bit mono signed linear PCM (slin) audio:
- **GCP**: `speech.Client`, LINEAR16 encoding
- **AWS**: `transcribestreaming.Client`, PCM encoding

### Per-Pod Session Anchoring

Streaming sessions live in memory on the pod that created them (`mapStreaming`, mutex-protected via `muStreaming`). The session's `host_id = POD_IP` is persisted to MySQL so follow-up RPCs (`stop`, `health-check`) can be routed to the correct pod.

Always lock/unlock the session map when accessing it. Implement proper cleanup in `Stop()` to prevent goroutine and WebSocket leaks.

### Audio Transport

The streaming handler dials out to Asterisk's `chan_websocket` endpoint (MediaURI from `ExternalMediaStart`). Asterisk pushes raw 8 kHz slin binary frames over WebSocket; these frames are forwarded to the STT provider stream.

WebSocket connection is `client` side (Go dials Asterisk), connection type `server`, transport `websocket`, encapsulation `none`.

### Status Transitions

Only valid status transitions are allowed. See `models/transcribe/transcribe.go:IsUpdatableStatus`. A `done` session cannot be restarted.

### Event-Driven Cleanup

- `call_hangup` (from call-manager): Finalizes any active transcription session associated with the call.
- `customer_deleted` (from customer-manager): Cascading deletion of all the customer's transcribe sessions.

### Events Published

| Event | Trigger |
|-------|---------|
| `transcribe.EventTypeTranscribeCreated` | Session successfully created |
| `transcribe.EventTypeTranscribeDone` | Session finalized/stopped |
| `transcribe.EventTypeTranscribeProgressing` | Transcription is actively processing audio |
