# Speaking API Design

Date: 2026-02-14

## Problem Statement

The TTS service currently offers two modes:

1. **Batch TTS** (`/v1/calls/{id}/talk`) - generates a complete audio file, then plays it. Customers cannot interrupt or control playback mid-stream.
2. **Internal streaming TTS** - streams audio in real-time via ElevenLabs WebSocket into live calls. However, this is only available internally through flow actions and has no external API.

Customers need direct control over real-time TTS on active calls: sending text incrementally, stopping speech mid-sentence, and clearing queued messages. This requires a new external API resource.

## Design

### API: `/v1/speakings` as an Independent Resource

The `speaking` resource represents an active streaming TTS session attached to a call (or conference bridge). It is an independent, first-class entity with its own lifecycle, linked to a call via `reference_type` and `reference_id`.

#### Endpoints

| Method | Path | Purpose |
|--------|------|---------|
| `POST` | `/v1/speakings` | Create a session. Eagerly sets up AudioSocket channel on the referenced call and connects to ElevenLabs. |
| `GET` | `/v1/speakings` | List sessions (filterable by customer_id, reference_type, reference_id, status). |
| `GET` | `/v1/speakings/{id}` | Get session details and status. |
| `POST` | `/v1/speakings/{id}/say` | Add text to the speech queue. Can be called multiple times. |
| `POST` | `/v1/speakings/{id}/flush` | Cancel current speech and clear all queued messages. Session stays open for new `/say` calls. |
| `POST` | `/v1/speakings/{id}/stop` | Terminate the session. Closes AudioSocket and ElevenLabs connections. |
| `DELETE` | `/v1/speakings/{id}` | Soft-delete the record. |

#### Create Request Body

```json
{
  "reference_type": "call",
  "reference_id": "<call-uuid>",
  "language": "en-US",
  "provider": "elevenlabs",
  "voice_id": "EXAVITQu4vr4xnSDxMaL",
  "direction": "out"
}
```

- `provider`: Only `elevenlabs` supported in v1. Defaults to `elevenlabs` if empty.
- `voice_id`: Provider-specific voice ID. If empty, the system picks a default based on language using the existing `elevenlabsVoiceIDMap`.
- `direction`: Audio injection direction — `in` (only far side hears), `out` (only caller hears), `both`.

#### Say Request Body

```json
{
  "text": "Hello, how can I help you today?"
}
```

Flush and stop have no request body.

#### Response

All mutating endpoints return the speaking record:

```json
{
  "id": "<uuid>",
  "customer_id": "<uuid>",
  "reference_type": "call",
  "reference_id": "<uuid>",
  "language": "en-US",
  "provider": "elevenlabs",
  "voice_id": "EXAVITQu4vr4xnSDxMaL",
  "direction": "out",
  "status": "active",
  "tm_create": "...",
  "tm_update": "...",
  "tm_delete": "..."
}
```

### Session Lifecycle

```
POST /v1/speakings        -> status: initiating
AudioSocket connects +
ElevenLabs connects       -> status: active
POST .../say              -> text queued and streamed
POST .../say              -> more text queued
POST .../flush            -> queue cleared, current speech stopped, session stays open
POST .../say              -> new text queued (session is still active)
POST .../stop             -> status: stopped (connections closed)
DELETE /v1/speakings/{id} -> soft-deleted
```

#### Operations

- **Say**: Sends text to ElevenLabs via the persistent WebSocket. Can be called multiple times. Text is queued and streamed incrementally.
- **Flush**: Clears all queued messages and stops current playback. The session remains open — new `/say` calls work immediately after. ElevenLabs supports this natively via `{"text": "", "flush": true}`.
- **Stop**: Terminates the session entirely. Closes the ElevenLabs WebSocket and AudioSocket connections. Updates DB status to `stopped`.
- **SayFinish** (internal only): Not exposed in the external API. The internal streaming handler uses this to signal end-of-message for flow-driven TTS. For the speaking API, the session stays alive indefinitely until `/stop`.

### Data Model

#### Database Table: `tts_manager_speaking`

| Column | Type | Notes |
|--------|------|-------|
| `id` | `binary(16)` | UUID, primary key |
| `customer_id` | `binary(16)` | UUID, required |
| `reference_type` | `varchar(32)` | "call", "confbridge" |
| `reference_id` | `binary(16)` | UUID of the referenced entity |
| `language` | `varchar(16)` | e.g. "en-US" |
| `provider` | `varchar(32)` | e.g. "elevenlabs" |
| `voice_id` | `varchar(255)` | Provider-specific voice ID |
| `direction` | `varchar(8)` | "in", "out", "both" |
| `status` | `varchar(16)` | "initiating", "active", "stopped" |
| `pod_id` | `varchar(64)` | Kubernetes pod hostname for RPC routing |
| `tm_create` | `datetime(6)` | Created timestamp |
| `tm_update` | `datetime(6)` | Last updated |
| `tm_delete` | `datetime(6)` | Soft-delete timestamp |

#### Status Lifecycle

- `initiating`: Session created, AudioSocket + ElevenLabs connection in progress.
- `active`: Connections established, ready for `/say`.
- `stopped`: Session terminated (via `/stop`, call ended, or pod death).

#### Go Model: `models/speaking/speaking.go`

```go
type Speaking struct {
    commonidentity.Identity

    ReferenceType streaming.ReferenceType `json:"reference_type" db:"reference_type"`
    ReferenceID   uuid.UUID               `json:"reference_id"   db:"reference_id,uuid"`
    Language      string                  `json:"language"        db:"language"`
    Provider      string                  `json:"provider"        db:"provider"`
    VoiceID       string                  `json:"voice_id"        db:"voice_id"`
    Direction     streaming.Direction     `json:"direction"       db:"direction"`
    Status        Status                  `json:"status"          db:"status"`
    PodID         string                  `json:"pod_id"          db:"pod_id"`

    TMCreate string `json:"tm_create" db:"tm_create"`
    TMUpdate string `json:"tm_update" db:"tm_update"`
    TMDelete string `json:"tm_delete" db:"tm_delete"`
}
```

#### Relationship to Internal Streaming

The `speaking.ID` is used as the `streaming.ID` — they share the same UUID. When a speaking is created, the handler creates both the DB record and the in-memory streaming session. The DB record is the source of truth for the API; the in-memory session is the source of truth for live connections.

### Internal Architecture

#### Request Flow

```
Customer API Request
    -> bin-api-manager (HTTP)
    -> bin-tts-manager (RabbitMQ RPC)
    -> speakinghandler (new) -> streaminghandler (existing)
    -> ElevenLabs WebSocket + Asterisk AudioSocket
```

#### What Already Exists and Can Be Reused

- `streaminghandler.Start()` - creates a streaming session, requests an external media channel from call-manager, waits for AudioSocket connection.
- `streaminghandler.SayAdd()` / `SayStop()` - say operations.
- `streaming.Streaming` model - has reference_type, reference_id, language, direction.
- `elevenlabsHandler` - full WebSocket lifecycle with the `streamer` interface.
- Pod-specific queue routing (`bin-manager.tts-manager.request.{pod_id}`) - already implemented in `main.go`.

#### What Needs to Be Built

1. **OpenAPI spec** (`bin-openapi-manager`) - speaking resource and endpoint definitions.
2. **API routes** (`bin-api-manager`) - HTTP routes translating to RPC calls.
3. **RPC client methods** (`bin-common-handler`) - new `TTSV1Speaking*` methods.
4. **RPC handlers** (`bin-tts-manager/pkg/listenhandler`) - new routes for speaking operations.
5. **Speaking handler** (`bin-tts-manager/pkg/speakinghandler`) - new handler orchestrating DB records + streaming sessions.
6. **DB handler** (`bin-tts-manager/pkg/dbhandler`) - CRUD for `tts_manager_speaking` table.
7. **Speaking model** (`bin-tts-manager/models/speaking`) - Go struct with DB tags.
8. **Flush support** - new `SayFlush()` method on the `streamer` interface and ElevenLabs implementation.
9. **Database migration** (`bin-dbscheme-manager`) - Alembic migration for the new table.

### Flush Implementation

The flush operation stops current playback and clears all queued text while keeping the session alive.

#### ElevenLabs Native Support

ElevenLabs WebSocket accepts an empty text message with `flush: true` to clear its internal buffer:

```json
{"text": "", "flush": true}
```

This stops the current generation and discards pending audio. The WebSocket stays open.

#### Implementation

```go
func (h *elevenlabsHandler) SayFlush(vendorConfig any) error {
    cf := vendorConfig.(*ElevenlabsConfig)

    cf.muConnWebsock.Lock()
    defer cf.muConnWebsock.Unlock()

    if cf.ConnWebsock == nil {
        return fmt.Errorf("the ConnWebsock is nil")
    }

    msg := ElevenlabsMessage{Text: "", Flush: true}
    return cf.ConnWebsock.WriteJSON(msg)
}
```

#### Stale Audio Discarding

After a flush, ElevenLabs may still send a few buffered audio chunks from the previous generation before acknowledging the flush. The `runProcess` goroutine needs to discard these stale chunks. Approach: increment an atomic counter on flush; `runProcess` skips audio writes when the counter changes mid-stream. ElevenLabs sends `isFinal: true` at the end of each generation, which signals when the flushed generation is fully drained.

#### Updated Streamer Interface

```go
type streamer interface {
    Init(ctx context.Context, st *streaming.Streaming) (any, error)
    Run(vendorConfig any) error
    SayStop(vendorConfig any) error
    SayAdd(vendorConfig any, text string) error
    SayFinish(vendorConfig any) error
    SayFlush(vendorConfig any) error  // new
}
```

### RPC Methods

New methods in `bin-common-handler/pkg/requesthandler/tts_speakings.go`:

| RPC Method | Triggered by | Notes |
|------------|-------------|-------|
| `TTSV1SpeakingCreate` | `POST /v1/speakings` | Uses shared queue (any pod can handle initial creation) |
| `TTSV1SpeakingGet` | `GET /v1/speakings/{id}` | Uses shared queue (reads from DB) |
| `TTSV1SpeakingGets` | `GET /v1/speakings` | Uses shared queue (reads from DB) |
| `TTSV1SpeakingSay` | `POST /v1/speakings/{id}/say` | Uses pod-targeted queue (must reach the pod with the live session) |
| `TTSV1SpeakingFlush` | `POST /v1/speakings/{id}/flush` | Uses pod-targeted queue |
| `TTSV1SpeakingStop` | `POST /v1/speakings/{id}/stop` | Uses pod-targeted queue |
| `TTSV1SpeakingDelete` | `DELETE /v1/speakings/{id}` | Uses shared queue (DB operation) |

Pod-targeted routing: The API layer reads `pod_id` from the speaking DB record and routes `/say`, `/flush`, `/stop` to `bin-manager.tts-manager.request.{pod_id}`.

The existing streaming RPC methods (`TTSV1StreamingCreate`, `SayInit`, `SayAdd`, `SayStop`, `SayFinish`, `StreamingDelete`) and their listen handler routes in `v1_streamings.go` are still used by the internal flow path and must be retained.

### Edge Cases

#### Call Ends While Speaking Is Active

Asterisk closes the AudioSocket TCP connection. The `runKeepConsume` goroutine detects the read error and cancels the context, which tears down the ElevenLabs WebSocket. A cleanup callback in `runStart` updates the speaking DB status to `stopped`. Subsequent `/say` or `/flush` calls return an error.

#### ElevenLabs WebSocket Drops

The `runProcess` goroutine detects the read error and exits. For v1: mark the session as `stopped`. Customer must create a new session. (Future: auto-reconnect transparently.)

#### AudioSocket Connection Timeout on Create

`POST /v1/speakings` requests an external media channel from call-manager, but the AudioSocket connection never arrives. Timeout after ~10 seconds. Update DB status to `stopped`. Return error to the customer.

The create flow is synchronous (matching the existing `streaminghandler.Start()` pattern). Expected latency: 1-3 seconds. This is acceptable for v1.

#### Concurrent `/say` and `/flush`

The existing `muConnWebsock` mutex serializes WebSocket writes. If flush wins, the just-queued text gets flushed. If say wins, the text is queued then immediately flushed. Either way, behavior is consistent.

#### `/say` After `/stop`

The handler checks the in-memory session — it's gone. Falls back to checking DB status — it's `stopped`. Returns error: "session is no longer active."

#### One Active Session Per Reference

Only one active speaking session is allowed per `reference_type` + `reference_id`. On `POST /v1/speakings`, query for any existing session with the same reference and status `active` or `initiating`. Return error if found.

This also prevents conflicts with flow-driven streaming sessions on the same call.

#### Pod Death (Orphaned Records)

If a tts-manager pod crashes, in-memory sessions are lost but DB records remain `active`. When a `/say`, `/flush`, or `/stop` request times out trying to reach the dead pod's queue, update the DB status to `stopped`. This is lazy cleanup — no background reaper needed for v1.

### Compatibility: Gender Field

The internal `streaming.Streaming` model uses `gender` to select an ElevenLabs voice via the `elevenlabsVoiceIDMap`. The speaking API uses `provider` + `voice_id` instead.

To avoid breaking the internal streaming path (used by AI talk service):
- Keep `gender` in the `streaming.Streaming` model.
- Add `provider` and `voice_id` as new fields alongside `gender`.
- The speaking API populates `provider`/`voice_id`; the internal flow path continues using `gender`.
- ElevenLabs handler checks `voice_id` first; if empty, falls back to `gender + language` lookup.

This allows the internal AI talk flow to continue working unchanged while the external API uses the new fields.

## Implementation Plan

### Step 1: Database Migration (`bin-dbscheme-manager`)

Create Alembic migration for `tts_manager_speaking` table.

### Step 2: OpenAPI Spec (`bin-openapi-manager`)

Define speaking resource and all 7 endpoints. Speaking model, SpeakingCreate request, SpeakingSay request.

### Step 3: tts-manager Core (`bin-tts-manager`)

| Component | Changes |
|-----------|---------|
| `models/speaking/` | New model with DB tags, Status type, Field type |
| `pkg/dbhandler/` | CRUD operations for speaking table |
| `pkg/streaminghandler/` | Add `provider`/`voice_id` fields to streaming model. Add `SayFlush()` to `streamer` interface. Implement flush in `elevenlabsHandler`. Keep `gender` for backwards compatibility. |
| `pkg/speakinghandler/` | New handler: Create (DB + streaming + external media), Say, Flush, Stop, Get, Gets, Delete. One-session-per-reference check. Pod death lazy cleanup. |
| `pkg/listenhandler/` | New RPC routes for `/v1/speakings/*`. Delete old `/v1/streamings/*` routes. |

### Step 4: RPC Client Methods (`bin-common-handler`)

Create `tts_speakings.go` with `TTSV1Speaking*` methods. Delete unused `tts_streamings.go` methods and routes.

### Step 5: API Routes (`bin-api-manager`)

Add HTTP routes in api-manager. Regenerate server code from OpenAPI spec. Pod-targeted routing for say/flush/stop.

### Step 6: Tests

- Unit tests for `speakinghandler` (create, say, flush, stop, one-session-per-reference, pod death cleanup).
- Unit tests for `dbhandler` speaking CRUD.
- Unit tests for `elevenlabsHandler.SayFlush()`.
- API validator tests for read-only endpoints (GET/list).

### Dependency Order

```
Step 1 (migration)
    -> Step 2 (OpenAPI)
    -> Step 3 + Step 4 (parallel: tts-manager core + RPC client)
        -> Step 5 (API routes)
            -> Step 6 (tests, run throughout)
```

## Future Considerations (Not in v1)

- **Additional providers**: Add GCP and AWS as streaming TTS providers with fallback chain.
- **VAD / barge-in**: Detect caller speech via AudioSocket incoming audio, auto-flush on voice activity.
- **Auto-reconnect**: Transparently reconnect ElevenLabs WebSocket on drop without terminating the session.
- **Batch TTS via ElevenLabs**: Add ElevenLabs REST API as a 3rd batch provider.
- **Background reaper**: Periodically scan for orphaned `active` records on dead pods.
