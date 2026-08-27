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
| `host_id` | UUID | Random UUID generated at process startup (`uuid.NewV4()`), not `POD_IP` — used for per-pod queue routing; changes on every process restart |
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

Streaming sessions live in memory on the pod that created them (`mapStreaming`, mutex-protected via `muStreaming`). The session's `host_id` — a random UUID generated once when the process starts, not `POD_IP` — is persisted to MySQL so follow-up RPCs (`stop`, `health-check`) can be routed to the correct process. Because `host_id` is regenerated on every restart, a restart invalidates routing to any of that process's prior sessions, independent of whether the pod's IP changed.

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

Every event below is published to the per-service fanout exchange `bin-manager.transcribe-manager.event`, and — since VOIP-1404 — also to the global topic exchange `bin-manager.event` with the routing key `transcribe-manager.<resource>.<transcribe-id>.<action>`. The third key segment is the *subscription address*: it is always the transcribe-id, across all three resource namespaces, so one transcription session is followed with `transcribe-manager.transcribe.<id>.#`, `transcribe-manager.transcript.<id>.#`, and `transcribe-manager.streaming.<id>.#`.

| Event | Data | Trigger | Topic routing key |
|-------|------|---------|-------------------|
| `transcribe.EventTypeTranscribeCreated` | `*transcribe.Transcribe` | Session successfully created | `transcribe-manager.transcribe.<transcribe-id>.created` |
| `transcribe.EventTypeTranscribeProgressing` | `*transcribe.Transcribe` | Transcription is actively processing audio | `transcribe-manager.transcribe.<transcribe-id>.progressing` |
| `transcribe.EventTypeTranscribeDone` | `*transcribe.Transcribe` | Session finalized/stopped | `transcribe-manager.transcribe.<transcribe-id>.done` |
| `transcribe.EventTypeTranscribeDeleted` | `*transcribe.Transcribe` | Session deleted | `transcribe-manager.transcribe.<transcribe-id>.deleted` |
| `streaming.EventTypeSpeechStarted` | `*streaming.Speech` | Voice activity started on a streaming leg | `transcribe-manager.transcribe.<transcribe-id>.speech_started` |
| `streaming.EventTypeSpeechInterim` | `*streaming.Speech` | Interim (non-final) STT result | `transcribe-manager.transcribe.<transcribe-id>.speech_interim` |
| `streaming.EventTypeSpeechEnded` | `*streaming.Speech` | Voice activity ended on a streaming leg | `transcribe-manager.transcribe.<transcribe-id>.speech_ended` |
| `streaming.EventTypeStreamingStarted` | `*streaming.Streaming` | Streaming leg registered in the session map | `transcribe-manager.streaming.<transcribe-id>.started` |
| `streaming.EventTypeStreamingStopped` | `*streaming.Streaming` | Streaming leg removed from the session map | `transcribe-manager.streaming.<transcribe-id>.stopped` |
| `transcript.EventTypeTranscriptCreated` | `*transcript.Transcript` | Final STT result persisted | `transcribe-manager.transcript.<transcribe-id>.created` |

`Speech`, `Streaming`, and `Transcript` all implement `eventtopic.SubscriptionIdentifier` (pointer receiver) returning `TranscribeID`, because their own ids are either regenerated per event (`Speech`, `Transcript`) or address the wrong resource (`Streaming`). `Transcribe` needs no override — its own id already is the subscription address.

`transcript.EventTypeTranscriptDeleted` is defined but never published: `pkg/transcripthandler/db.go:33` publishes `transcript_created` on the delete path. This is a known bug tracked separately; the golden routing-key table in `models/transcribe/routingkey_golden_test.go` pins the current behavior and must be updated together with the fix.
