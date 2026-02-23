# TTS Billing Design

## Problem Statement

TTS (Text-to-Speech) Speaking sessions are not billed. We need to charge customers for TTS usage at 3 tokens/min and $0.03/min, using the token-first cost mode (tokens deducted first, overflow to credits).

## Approach

Follow the existing call billing pattern (event-driven, duration-based):

1. `bin-tts-manager` publishes `speaking_started` and `speaking_stopped` events
2. `bin-billing-manager` subscribes to TTS events and creates/finalizes billing records
3. Duration calculated as ceiling-rounded minutes (same as calls)
4. Same rate for all TTS providers (GCP, AWS, ElevenLabs)

## Changes

### bin-tts-manager

**New file: `models/speaking/event.go`**
- `EventTypeSpeakingStarted = "speaking_started"`
- `EventTypeSpeakingStopped = "speaking_stopped"`

**Modified: `pkg/speakinghandler/speaking.go`**
- In `Create()`: publish `speaking_started` after status becomes `active`
- In `Stop()`: publish `speaking_stopped` after status becomes `stopped`

### bin-billing-manager

**Modified: `models/billing/billing.go`**
- Add `ReferenceTypeSpeaking ReferenceType = "speaking"`

**Modified: `models/billing/cost_type.go`**
- Add `CostTypeTTS CostType = "tts"`
- Add `DefaultTokenPerUnitTTS int64 = 3` (3 tokens/min)
- Add `DefaultCreditPerUnitTTS int64 = 30000` ($0.03/min in micros)
- Add case in `GetCostInfo()`: `CostTypeTTS` returns `CostInfo{CostModeTokenFirst, 3, 30000}`

**Modified: `pkg/billinghandler/billing.go`**
- Add `ReferenceTypeSpeaking` to duration-based switch in `BillingStart`
- Add `ReferenceTypeSpeaking` to duration calculation in `BillingEnd`

**New file: `pkg/billinghandler/event_tts.go`**
- `EventTTSSpeakingStarted(ctx, *speaking.Speaking)` — calls `BillingStart()`
- `EventTTSSpeakingStopped(ctx, *speaking.Speaking)` — looks up billing, calls `BillingEnd()`

**New file: `pkg/subscribehandler/tts.go`**
- `processEventTTSSpeakingStarted()` — unmarshal Speaking, delegate to billingHandler
- `processEventTTSSpeakingStopped()` — unmarshal Speaking, delegate to billingHandler

**Modified: `pkg/subscribehandler/main.go`**
- Add `speaking_started` and `speaking_stopped` event cases
- Import tts-manager Speaking model

**Modified: `cmd/billing-manager/main.go`**
- Add `QueueNameTTSEvent` to subscribe targets

## Data Flow

```
Speaking.Create() → status=active → publish speaking_started
    → billing-manager subscribes
    → BillingStart(speaking) → billing record (status=progressing)

Speaking.Stop() → status=stopped → publish speaking_stopped
    → billing-manager subscribes
    → BillingEnd(speaking) → calculate duration, deduct tokens/credits (status=end)
```

## Billing Calculation

- Duration: `ceil(seconds / 60)` billable minutes
- Token cost: 3 tokens × billable minutes
- Credit cost: $0.03 × billable minutes (30000 micros × billable minutes)
- Example: 61-second session = 2 billable units = 6 tokens or $0.06

## Services Affected

- `bin-tts-manager` — add event publications
- `bin-billing-manager` — add subscription, handlers, constants
