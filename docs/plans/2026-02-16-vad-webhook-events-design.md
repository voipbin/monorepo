# VAD Webhook Events Design

## Problem Statement

The transcribe-manager processes real-time STT via GCP and AWS providers but currently discards interim/partial results. There is no way for customers to know when speech starts and stops on a transcription stream. We want to publish VAD (Voice Activity Detection) webhook events derived from the STT provider responses.

## Approach

Derive VAD signals from the existing STT streaming pipeline. Both GCP and AWS providers return interim/partial results before delivering final transcriptions. These interim results naturally indicate speech activity:

- First interim result = speech started
- Subsequent interim results = speech ongoing (with partial text)
- Final result = speech ended (utterance complete)

No new infrastructure, no separate VAD engine. Fire-and-forget webhook events only (no DB persistence).

## Event Types

Three new webhook event types in `models/streaming/event.go`:

| Event Type | Trigger | Payload |
|---|---|---|
| `transcribe_speech_started` | First interim/partial result received | streaming_id, transcribe_id, direction, tm_event |
| `transcribe_speech_interim` | Subsequent interim/partial result | streaming_id, transcribe_id, direction, message (partial text), tm_event |
| `transcribe_speech_ended` | Final result received | streaming_id, transcribe_id, direction, tm_event |

## Webhook Payload

New `models/streaming/webhook.go`:

```go
type WebhookMessage struct {
    commonidentity.Identity                              // ID = streaming_id, CustomerID
    TranscribeID uuid.UUID         `json:"transcribe_id"`
    Direction    transcript.Direction `json:"direction"`
    Message      string              `json:"message,omitempty"` // partial text for speech_interim
    TMEvent      *time.Time          `json:"tm_event"`
}
```

Implements `notifyhandler.WebhookMessage` interface via `CreateWebhookEvent()`.

## State Machine

Per-stream `speaking` boolean (local variable in the result-processing goroutine):

```
State: not speaking
  → interim result received → set speaking=true, emit speech_started, emit speech_interim
State: speaking
  → interim result received → emit speech_interim (with partial text)
  → final result received → set speaking=false, emit speech_ended
```

## Implementation Changes

### models/streaming/event.go
Add three new event type constants.

### models/streaming/webhook.go (new file)
WebhookMessage struct, ConvertWebhookMessage helper, CreateWebhookEvent method.

### pkg/streaminghandler/gcp.go — gcpProcessResult
- Add `speaking` boolean state variable
- Process interim results (currently skipped): extract partial text, publish VAD events
- Remove `time.Sleep(100ms)` on interim results
- On final result: emit `speech_ended` if was speaking, then existing transcript creation

### pkg/streaminghandler/aws.go — awsProcessResult
- Add `speaking` boolean state variable
- Process partial results (currently skipped): extract partial text, publish VAD events
- On non-partial result: emit `speech_ended` if was speaking, then existing transcript creation

## Volume Consideration

Interim results can be frequent (every 100-300ms per utterance). Each generates a `speech_interim` webhook. This is by design — the customer chose the verbose event set. No throttling.

## Files Changed

| File | Change |
|---|---|
| `models/streaming/event.go` | Add 3 event type constants |
| `models/streaming/webhook.go` | New: WebhookMessage + interface methods |
| `pkg/streaminghandler/gcp.go` | Track speaking state, publish VAD events |
| `pkg/streaminghandler/aws.go` | Track speaking state, publish VAD events |

## Out of Scope

- No database persistence for VAD events
- No OpenAPI schema changes (no new API endpoints)
- No changes to other services
- No throttling or rate-limiting of interim events
