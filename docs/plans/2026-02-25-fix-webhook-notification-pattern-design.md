# Fix Webhook Notification Pattern

## Problem

`PublishWebhookEvent` does two things with the same `data` parameter:

```go
func (h *notifyHandler) PublishWebhookEvent(ctx, customerID, eventType, data WebhookMessage) {
    go h.PublishEvent(ctx, eventType, data)         // json.Marshal(data) — raw object
    go h.PublishWebhook(ctx, customerID, eventType, data)  // data.CreateWebhookEvent() — filtered
}
```

- `PublishEvent` (internal event queue): calls `json.Marshal(data)` — serializes whatever is passed.
- `PublishWebhook` (customer webhook): calls `data.CreateWebhookEvent()` — converts via `ConvertWebhookMessage()` to filter internal fields.

The correct pattern (Pattern A) is to pass the **internal struct** so that:
- Internal event queue gets ALL fields (for other services, analytics, ClickHouse)
- Customer webhook gets filtered fields via `CreateWebhookEvent()`

Three call sites use the wrong pattern (Pattern B) — they pre-convert to `WebhookMessage` before calling `PublishWebhookEvent`, causing the internal event queue to lose data.

## Affected Call Sites

### 1. bin-talk-manager/pkg/messagehandler/message.go (lines 186-207)

`publishMessageCreatedEvent()` and `publishMessageDeletedEvent()` pre-convert to `*message.WebhookMessage`. The internal `Message` struct and `WebhookMessage` have identical fields, and `Message` already implements `CreateWebhookEvent()`.

**Fix:** Pass `msg` (*message.Message) directly. Remove the pre-conversion.

### 2. bin-talk-manager/pkg/reactionhandler/reaction.go (lines 137-147)

`publishReactionUpdated()` pre-converts to `*message.WebhookMessage`.

**Fix:** Pass `m` (*message.Message) directly. Remove the pre-conversion.

### 3. bin-transcribe-manager/pkg/streaminghandler/result.go (lines 52-70)

`Streaming.ConvertWebhookMessage(message, tmEvent)` takes extra parameters not stored on the `Streaming` struct. The webhook payload is constructed from multiple sources (session + speech result + timestamp). `Streaming` can't implement `CreateWebhookEvent()` because it doesn't have those fields.

**Fix:** Introduce a new `Speech` struct that combines session data with per-event data.

## Design: Speech Struct (bin-transcribe-manager)

### Why not add fields to Streaming?

`Streaming` is a live session object holding `ConnAst *websocket.Conn`. It represents the persistent session, not a per-event snapshot. Adding ephemeral per-event fields (`Message`, `TMEvent`) would be:
- Semantically wrong (session vs event)
- Unsafe (shared struct mutated per event)

### New struct: Speech

```go
// streaming/speech.go

type Speech struct {
    commonidentity.Identity

    StreamingID  uuid.UUID            `json:"streaming_id"`
    TranscribeID uuid.UUID            `json:"transcribe_id"`
    Language     string               `json:"language"`
    Direction    transcript.Direction  `json:"direction"`

    Message string     `json:"message,omitempty"`
    TMEvent *time.Time `json:"tm_event"`

    TMCreate *time.Time `json:"tm_create"`
}
```

Constructor on `Streaming`:

```go
func (h *Streaming) NewSpeech(message string, tmEvent *time.Time) *Speech {
    return &Speech{
        Identity: commonidentity.Identity{
            ID:         uuid.Must(uuid.NewV4()),
            CustomerID: h.CustomerID,
        },
        StreamingID:  h.ID,
        TranscribeID: h.TranscribeID,
        Language:     h.Language,
        Direction:    h.Direction,
        Message:      message,
        TMEvent:      tmEvent,
        TMCreate:     tmEvent,
    }
}
```

### Updated WebhookMessage

```go
// streaming/webhook.go

type WebhookMessage struct {
    commonidentity.Identity

    StreamingID  uuid.UUID            `json:"streaming_id"`
    TranscribeID uuid.UUID            `json:"transcribe_id"`
    Direction    transcript.Direction  `json:"direction"`
    Message      string               `json:"message,omitempty"`
    TMEvent      *time.Time           `json:"tm_event"`

    TMCreate *time.Time `json:"tm_create"`
}

func (h *Speech) ConvertWebhookMessage() *WebhookMessage {
    return &WebhookMessage{
        Identity:     h.Identity,
        StreamingID:  h.StreamingID,
        TranscribeID: h.TranscribeID,
        Direction:    h.Direction,
        Message:      h.Message,
        TMEvent:      h.TMEvent,
        TMCreate:     h.TMCreate,
    }
}

func (h *Speech) CreateWebhookEvent() ([]byte, error) {
    e := h.ConvertWebhookMessage()
    m, err := json.Marshal(e)
    if err != nil {
        return nil, err
    }
    return m, nil
}
```

`Language` is intentionally excluded from `WebhookMessage` — internal event queue gets it via `json.Marshal(Speech)`, customer webhook does not.

### Caller change in result.go

Before:
```go
webhookMsg := rp.st.ConvertWebhookMessage("", &now)
rp.notifyHandler.PublishWebhookEvent(ctx, rp.st.CustomerID, streaming.EventTypeSpeechStarted, webhookMsg)
```

After:
```go
evt := rp.st.NewSpeech("", &now)
rp.notifyHandler.PublishWebhookEvent(ctx, rp.st.CustomerID, streaming.EventTypeSpeechStarted, evt)
```

## Cleanup

- Remove `ConvertWebhookMessage()` from `Streaming` struct
- Remove `CreateWebhookEvent()` from `WebhookMessage` struct (move to `Speech`)
- Remove `WebhookMessage.CreateWebhookEvent()` on `*message.WebhookMessage` in bin-talk-manager (only keep it on `*message.Message`)

## Files Changed

### bin-talk-manager
- `pkg/messagehandler/message.go` — Remove pre-conversion, pass `msg` directly
- `pkg/reactionhandler/reaction.go` — Remove pre-conversion, pass `m` directly
- `models/message/message.go` — Remove `CreateWebhookEvent()` on `*WebhookMessage`
- `pkg/messagehandler/message_test.go` — Update mock expectations
- `pkg/reactionhandler/reaction_test.go` — Update mock expectations

### bin-transcribe-manager
- `models/streaming/speech.go` — New file: `Speech` struct with constructor
- `models/streaming/webhook.go` — Update `WebhookMessage`, move methods to `Speech`
- `models/streaming/streaming.go` — Add `NewSpeech()` constructor
- `pkg/streaminghandler/result.go` — Use `NewSpeech()` instead of `ConvertWebhookMessage()`
- `pkg/streaminghandler/result_test.go` — Update tests
