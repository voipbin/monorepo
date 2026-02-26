# Recording Billing Design

## Problem Statement

Recordings are currently free. We need to charge for recording based on recording duration at 3 tokens/min (token-first) with $0.03/min credit overflow — the same rate structure as TTS.

## Approach

Option A: Balance check in `bin-call-manager` (pre-start), billing in `bin-billing-manager` (event-driven two-phase). This follows the established patterns for calls and TTS.

## Design

### 1. Billing Model Changes (bin-billing-manager)

**`models/billing/billing.go`** — Add:
- `ReferenceTypeRecording ReferenceType = "recording"`

**`models/billing/cost_type.go`** — Add:
- `CostTypeRecording CostType = "recording"`
- `DefaultCreditPerUnitRecording int64 = 30000` ($0.03/min)
- `DefaultTokenPerUnitRecording int64 = 3`
- `GetCostInfo` case: `CostTypeRecording` → `CostInfo{CostModeTokenFirst, 3, 30000}`

### 2. Balance Check (bin-call-manager)

**`pkg/recordinghandler/recording.go`** — In both `recordingReferenceTypeCall()` and `recordingReferenceTypeConfbridge()`, after retrieving the call/confbridge (which provides `CustomerID`), add balance check via `reqHandler.BillingV1AccountIsValidBalanceByCustomerID()` with `billing.ReferenceTypeRecording`. Reject recording if balance insufficient (before creating snoop channels or Asterisk recording requests).

**`bin-billing-manager/pkg/accounthandler/balance.go`** — Add `ReferenceTypeRecording` case in `IsValidBalance` switch. Token-first mode: accept if `BalanceToken > 0`, else check credit >= `CreditPerUnit * count`.

### 3. Billing Event Handlers (bin-billing-manager)

Two-phase billing triggered by call-manager recording events (already on subscribed `QueueNameCallEvent`):

**New file `pkg/billinghandler/event_recording.go`:**
- `EventCMRecordingStarted(r *Recording)` → `BillingStart()` with `ReferenceTypeRecording`, `CostTypeRecording`, `r.TMCreate`
- `EventCMRecordingFinished(r *Recording)` → `GetByReferenceID(r.ID)`, then `BillingEnd()` with `r.TMUpdate`

**New file `pkg/subscribehandler/recording.go`:**
- `processEventCMRecordingStarted()` — unmarshal Recording, call billingHandler
- `processEventCMRecordingFinished()` — unmarshal Recording, call billingHandler

**`pkg/subscribehandler/main.go`** — Add cases for `recording_started` and `recording_finished` from call-manager.

**`pkg/billinghandler/main.go`** — Add `EventCMRecordingStarted` and `EventCMRecordingFinished` to `BillingHandler` interface.

### 4. OpenAPI Updates (bin-openapi-manager)

**`openapi/openapi.yaml`:**
- Add `recording` to `BillingManagerBillingreferenceType` enum
- Add `recording` to `BillingManagerBillingCostType` enum
- Regenerate `bin-openapi-manager` and `bin-api-manager`

### 5. RST Documentation (bin-api-manager)

**`docsdev/source/billing_account_overview.rst`:**
- Add recording to token-eligible services list
- Add Recording box to ASCII diagram (3 tok/min)
- Add row to Token Rates table: Recording, 3 tokens, per minute (ceiling-rounded)
- Add row to Credit Rates table: Recording (overflow), $0.03, 30000, per minute
- Add recording to token consumption explanation
- Add recording billing example
- Mention recording in plan selection tips

## Files Changed

| Service | File | Change |
|---------|------|--------|
| bin-billing-manager | models/billing/billing.go | Add ReferenceTypeRecording |
| bin-billing-manager | models/billing/cost_type.go | Add CostTypeRecording, defaults, GetCostInfo case |
| bin-billing-manager | pkg/accounthandler/balance.go | Add ReferenceTypeRecording case in IsValidBalance |
| bin-billing-manager | pkg/billinghandler/main.go | Add EventCMRecordingStarted, EventCMRecordingFinished to interface |
| bin-billing-manager | pkg/billinghandler/event_recording.go | New: recording billing event handlers |
| bin-billing-manager | pkg/subscribehandler/main.go | Add recording event cases + import |
| bin-billing-manager | pkg/subscribehandler/recording.go | New: unmarshal + dispatch recording events |
| bin-call-manager | pkg/recordinghandler/recording.go | Add balance check before recording start |
| bin-openapi-manager | openapi/openapi.yaml | Add recording to reference_type and cost_type enums |
| bin-openapi-manager | (generated) | go generate ./... |
| bin-api-manager | (generated) | go generate ./... |
| bin-api-manager | docsdev/source/billing_account_overview.rst | Add recording to rate tables, diagrams, examples |

## Tests

- bin-billing-manager/pkg/billinghandler/event_recording_test.go — test started/finished handlers
- bin-billing-manager/pkg/subscribehandler/recording_test.go — test event processing
- bin-billing-manager/pkg/accounthandler/balance_test.go — add ReferenceTypeRecording test cases
- bin-call-manager/pkg/recordinghandler/recording_test.go — test balance check pass/fail paths

## Not Changed

- No DB migration needed (recording billing uses existing billing_billings table)
- No new subscribe target (call-manager events already subscribed)

## Key Decisions

- **Rates**: 3 tokens/min, $0.03/min — identical to TTS
- **Billing mode**: TokenFirst (tokens consumed first, overflow to credits)
- **Unit rounding**: Ceiling-rounded to minutes (existing CalculateBillableUnits)
- **BillingStart timestamp**: r.TMCreate (recording TMStart is nil at event time)
- **BillingEnd timestamp**: r.TMUpdate (recording TMEnd not reliably set)
- **Method naming**: EventCMRecordingFinished (matches event type recording_finished)
