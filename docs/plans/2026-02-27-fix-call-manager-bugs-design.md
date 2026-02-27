# Fix call-manager Bugs

## Problem

Architecture review of `bin-call-manager` identified 5 bugs across the callhandler and models packages.

## Bugs

### Bug 1: `errors.Wrap(err, ...)` wraps nil in 6 methods

**Files:** `pkg/callhandler/hold.go`, `moh.go`, `silence.go`
**Methods:** HoldOn, HoldOff, MOHOn, MOHOff, SilenceOn, SilenceOff

Each method calls `h.Get(ctx, id)` into `err`, then calls a channelHandler method into `errHold`/`errMoh`/`errSilence`. On failure, they return `errors.Wrap(err, "...")` — wrapping the outer `err` (nil from successful Get) instead of the inner error. The function silently returns nil on failure.

**Fix:** Change to `errors.Wrap(errHold, ...)` / `errMoh` / `errSilence`.

### Bug 2: Goroutine logs wrong error variable

**File:** `pkg/callhandler/hangup.go:70`

A goroutine captures `err` from outer scope instead of its own `errReq`.

**Fix:** Change `err` to `errReq` in the log statement.

### Bug 3: Copy-paste in ConvertWebhookMessage()

**File:** `models/groupcall/webhook.go:67`

`AnswerGroupcallID: h.AnswerCallID` should be `AnswerGroupcallID: h.AnswerGroupcallID`.

**Fix:** Use correct source field. Add test assertion.

### Bug 4: JSON tag typo "reference_typee"

**File:** `models/externalmedia/main.go:16`

JSON tag has double-e: `json:"reference_typee"`. This is a breaking API change but the field name is clearly wrong.

**Fix:** Change to `json:"reference_type"`. Update all test fixtures.

### Bug 5: UpdateMuteDirection error swallowed

**File:** `pkg/callhandler/mute.go:38-41`

`UpdateMuteDirection` failure is logged but not returned. DB state diverges from Asterisk state.

**Fix:** Return the error.

## Verification

- `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`
