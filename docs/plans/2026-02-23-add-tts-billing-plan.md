# TTS Billing Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Charge customers for TTS Speaking sessions at 3 tokens/min and $0.03/min using the existing duration-based billing pattern.

**Architecture:** tts-manager publishes `speaking_started`/`speaking_stopped` events via RabbitMQ. billing-manager subscribes and creates/finalizes billing records using the same BillingStart/BillingEnd pattern used for calls.

**Tech Stack:** Go, RabbitMQ (event pub/sub), gomock (testing)

**Worktree:** `~/gitvoipbin/monorepo-worktrees/NOJIRA-add-tts-billing/`

---

### Task 1: Add Speaking event type constants (bin-tts-manager)

**Files:**
- Create: `bin-tts-manager/models/speaking/event.go`
- Create: `bin-tts-manager/models/speaking/event_test.go`

**Step 1: Write the event constants file**

```go
// bin-tts-manager/models/speaking/event.go
package speaking

const (
	EventTypeSpeakingStarted string = "speaking_started"
	EventTypeSpeakingStopped string = "speaking_stopped"
)
```

**Step 2: Write the test**

Follow the pattern in `bin-tts-manager/models/streaming/event_test.go`:

```go
// bin-tts-manager/models/speaking/event_test.go
package speaking

import (
	"testing"
)

func TestEventTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{
			name:     "event_type_speaking_started",
			constant: EventTypeSpeakingStarted,
			expected: "speaking_started",
		},
		{
			name:     "event_type_speaking_stopped",
			constant: EventTypeSpeakingStopped,
			expected: "speaking_stopped",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}
```

**Step 3: Run test to verify it passes**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-tts-billing/bin-tts-manager && go test ./models/speaking/...`
Expected: PASS

**Step 4: Commit**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-tts-billing
git add bin-tts-manager/models/speaking/event.go bin-tts-manager/models/speaking/event_test.go
git commit -m "NOJIRA-add-tts-billing

- bin-tts-manager: Add speaking_started and speaking_stopped event type constants"
```

---

### Task 2: Add notifyHandler to speakingHandler and publish events (bin-tts-manager)

The `speakingHandler` currently has no `notifyHandler`. We need to add it and publish events on Create (active) and Stop (stopped).

**Files:**
- Modify: `bin-tts-manager/pkg/speakinghandler/main.go` — add notifyHandler field and constructor param
- Modify: `bin-tts-manager/pkg/speakinghandler/speaking.go` — publish events in Create() and Stop()
- Modify: `bin-tts-manager/pkg/speakinghandler/main_test.go` — update constructor test
- Modify: `bin-tts-manager/pkg/speakinghandler/speaking_test.go` — add PublishEvent mock expectations
- Modify: `bin-tts-manager/cmd/tts-manager/main.go:137` — pass notifyHandler to NewSpeakingHandler

**Step 1: Modify `main.go` — add notifyHandler to struct and constructor**

In `bin-tts-manager/pkg/speakinghandler/main.go`:

- Add import: `"monorepo/bin-common-handler/pkg/notifyhandler"`
- Add field `notifyHandler notifyhandler.NotifyHandler` to `speakingHandler` struct
- Add param `notifyHandler notifyhandler.NotifyHandler` to `NewSpeakingHandler`
- Set the field in the constructor

Result:
```go
import (
	"context"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-tts-manager/models/speaking"
	"monorepo/bin-tts-manager/models/streaming"
	"monorepo/bin-tts-manager/pkg/dbhandler"
	"monorepo/bin-tts-manager/pkg/streaminghandler"

	"github.com/gofrs/uuid"
)

type speakingHandler struct {
	db               dbhandler.DBHandler
	streamingHandler streaminghandler.StreamingHandler
	notifyHandler    notifyhandler.NotifyHandler
	podID            string
}

func NewSpeakingHandler(
	db dbhandler.DBHandler,
	streamingHandler streaminghandler.StreamingHandler,
	notifyHandler notifyhandler.NotifyHandler,
	podID string,
) SpeakingHandler {
	return &speakingHandler{
		db:               db,
		streamingHandler: streamingHandler,
		notifyHandler:    notifyHandler,
		podID:            podID,
	}
}
```

**Step 2: Modify `speaking.go` — publish events**

In `Create()`, after the successful `SpeakingGet` at line 123 (the final return path), before returning, add:
```go
h.notifyHandler.PublishEvent(ctx, speaking.EventTypeSpeakingStarted, res)
```

In `Stop()`, after the successful `SpeakingGet` at line 261 (the final return path), before returning, add:
```go
h.notifyHandler.PublishEvent(ctx, speaking.EventTypeSpeakingStopped, res)
```

**Step 3: Modify `cmd/tts-manager/main.go:137` — pass notifyHandler**

Change:
```go
speakingHandler := speakinghandler.NewSpeakingHandler(dbHandler, streamingHandler, podID)
```
To:
```go
speakingHandler := speakinghandler.NewSpeakingHandler(dbHandler, streamingHandler, notifyHandler, podID)
```

Note: `notifyHandler` is already created at line 123.

**Step 4: Update `main_test.go` — update constructor test**

Add `notifyhandler` import and mock. Update `NewSpeakingHandler` call to include `mockNotify`:
```go
mockNotify := notifyhandler.NewMockNotifyHandler(mc)
h := NewSpeakingHandler(mockDB, mockStreaming, mockNotify, "test-pod")
```
Add assertion: `if sh.notifyHandler == nil { t.Error("notifyHandler should not be nil") }`

**Step 5: Update `speaking_test.go` — add PublishEvent expectations**

In every test that creates a `speakingHandler`, add `mockNotify`:
```go
mockNotify := notifyhandler.NewMockNotifyHandler(mc)
h := &speakingHandler{
	db:               mockDB,
	streamingHandler: mockStreaming,
	notifyHandler:    mockNotify,
	podID:            "test-pod",
}
```

For tests where Create() succeeds (normal, existing stopped session allows, default provider):
Add expectation after `mockDB.EXPECT().SpeakingGet(...)`:
```go
mockNotify.EXPECT().PublishEvent(ctx, speaking.EventTypeSpeakingStarted, tt.responseGet)
```

For `Test_Stop` where stop succeeds (normal, streaming stop error non-fatal):
Add expectation after `mockDB.EXPECT().SpeakingGet(ctx, tt.id).Return(tt.responseGetAfter, ...)`:
```go
mockNotify.EXPECT().PublishEvent(ctx, speaking.EventTypeSpeakingStopped, tt.responseGetAfter)
```

Tests where Get/Say/Flush/Delete don't trigger events: just add `mockNotify` to the handler struct with no expectations.

**Step 6: Run tests**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-tts-billing/bin-tts-manager && go test ./pkg/speakinghandler/...`
Expected: PASS

**Step 7: Run full verification workflow for bin-tts-manager**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-tts-billing/bin-tts-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`
Expected: All pass

**Step 8: Commit**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-tts-billing
git add bin-tts-manager/
git commit -m "NOJIRA-add-tts-billing

- bin-tts-manager: Add notifyHandler to speakingHandler
- bin-tts-manager: Publish speaking_started event on Create
- bin-tts-manager: Publish speaking_stopped event on Stop"
```

---

### Task 3: Add TTS billing constants (bin-billing-manager)

**Files:**
- Modify: `bin-billing-manager/models/billing/billing.go` — add ReferenceTypeSpeaking
- Modify: `bin-billing-manager/models/billing/cost_type.go` — add CostTypeTTS, rates, GetCostInfo case
- Modify: `bin-billing-manager/models/billing/cost_type_test.go` — add test case for TTS

**Step 1: Add ReferenceTypeSpeaking**

In `bin-billing-manager/models/billing/billing.go`, add to the reference type constants:
```go
ReferenceTypeSpeaking         ReferenceType = "speaking"
```

**Step 2: Add cost type and rates**

In `bin-billing-manager/models/billing/cost_type.go`:

Add to CostType constants:
```go
CostTypeTTS              CostType = "tts"
```

Add to credit rates:
```go
DefaultCreditPerUnitTTS          int64 = 30000   // $0.03/min
```

Add to token rates:
```go
DefaultTokenPerUnitTTS int64 = 3
```

Add case in `GetCostInfo()`:
```go
case CostTypeTTS:
	return CostInfo{CostModeTokenFirst, DefaultTokenPerUnitTTS, DefaultCreditPerUnitTTS}
```

**Step 3: Add test case**

In `bin-billing-manager/models/billing/cost_type_test.go`, add to the tests slice:
```go
{
	name:                "tts - token first",
	costType:            CostTypeTTS,
	expectMode:          CostModeTokenFirst,
	expectTokenPerUnit:  DefaultTokenPerUnitTTS,
	expectCreditPerUnit: DefaultCreditPerUnitTTS,
},
```

**Step 4: Run test**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-tts-billing/bin-billing-manager && go test ./models/billing/...`
Expected: PASS

**Step 5: Commit**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-tts-billing
git add bin-billing-manager/models/billing/
git commit -m "NOJIRA-add-tts-billing

- bin-billing-manager: Add ReferenceTypeSpeaking for TTS billing
- bin-billing-manager: Add CostTypeTTS with 3 tokens/min and \$0.03/min rates"
```

---

### Task 4: Update BillingStart and BillingEnd to support Speaking (bin-billing-manager)

**Files:**
- Modify: `bin-billing-manager/pkg/billinghandler/billing.go` — add Speaking to switch cases

**Step 1: Update BillingStart switch**

In `BillingStart()`, in the switch that determines `flagEnd` (around line 76-86), add `billing.ReferenceTypeSpeaking` to the duration-based case:
```go
case billing.ReferenceTypeCall, billing.ReferenceTypeCallExtension, billing.ReferenceTypeSpeaking:
	flagEnd = false
```

**Step 2: Update BillingEnd switch**

In `BillingEnd()`, in the switch that calculates `usageDuration` and `billableUnits` (around line 147-159), add `billing.ReferenceTypeSpeaking` to the duration-based case:
```go
case billing.ReferenceTypeCall, billing.ReferenceTypeCallExtension, billing.ReferenceTypeSpeaking:
```

**Step 3: Run existing tests to verify no regressions**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-tts-billing/bin-billing-manager && go test ./pkg/billinghandler/...`
Expected: PASS (existing tests still pass)

**Step 4: Commit**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-tts-billing
git add bin-billing-manager/pkg/billinghandler/billing.go
git commit -m "NOJIRA-add-tts-billing

- bin-billing-manager: Add ReferenceTypeSpeaking to BillingStart/BillingEnd duration-based switches"
```

---

### Task 5: Add TTS event handlers in billingHandler (bin-billing-manager)

**Files:**
- Create: `bin-billing-manager/pkg/billinghandler/event_tts.go`
- Create: `bin-billing-manager/pkg/billinghandler/event_tts_test.go`
- Modify: `bin-billing-manager/pkg/billinghandler/main.go` — add methods to BillingHandler interface

**Step 1: Write event_tts.go**

Follow the call event handler pattern from `event.go`:

```go
package billinghandler

import (
	"context"

	commonaddress "monorepo/bin-common-handler/models/address"
	tmspeaking "monorepo/bin-tts-manager/models/speaking"

	"monorepo/bin-billing-manager/models/billing"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// EventTTSSpeakingStarted handles the tts-manager's speaking_started event
func (h *billingHandler) EventTTSSpeakingStarted(ctx context.Context, s *tmspeaking.Speaking) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "EventTTSSpeakingStarted",
		"speaking_id": s.ID,
		"customer_id": s.CustomerID,
	})
	log.Debugf("Received speaking_started event. speaking_id: %s", s.ID)

	if errBilling := h.BillingStart(
		ctx,
		s.CustomerID,
		billing.ReferenceTypeSpeaking,
		s.ID,
		billing.CostTypeTTS,
		s.TMCreate,
		&commonaddress.Address{},
		&commonaddress.Address{},
	); errBilling != nil {
		return errors.Wrap(errBilling, "could not start a billing")
	}

	return nil
}

// EventTTSSpeakingStopped handles the tts-manager's speaking_stopped event
func (h *billingHandler) EventTTSSpeakingStopped(ctx context.Context, s *tmspeaking.Speaking) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "EventTTSSpeakingStopped",
		"speaking_id": s.ID,
	})
	log.Debugf("Received speaking_stopped event. speaking_id: %s", s.ID)

	b, err := h.GetByReferenceID(ctx, s.ID)
	if err != nil {
		// could not get billing. nothing to do.
		return nil
	}

	if s.TMUpdate == nil {
		return errors.Errorf("invalid tm_update. speaking_id: %s, tm_update: nil", s.ID)
	}

	if errEnd := h.BillingEnd(ctx, b, s.TMUpdate, &commonaddress.Address{}, &commonaddress.Address{}); errEnd != nil {
		return errors.Wrapf(errEnd, "could not end the billing. billing_id: %s, speaking_id: %s", b.ID, s.ID)
	}

	return nil
}
```

**Step 2: Add methods to BillingHandler interface**

In `bin-billing-manager/pkg/billinghandler/main.go`, add to the interface:
```go
EventTTSSpeakingStarted(ctx context.Context, s *tmspeaking.Speaking) error
EventTTSSpeakingStopped(ctx context.Context, s *tmspeaking.Speaking) error
```

Add import: `tmspeaking "monorepo/bin-tts-manager/models/speaking"`

**Step 3: Regenerate mocks**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-tts-billing/bin-billing-manager && go generate ./pkg/billinghandler/...`

**Step 4: Write test file**

Create `bin-billing-manager/pkg/billinghandler/event_tts_test.go`. Follow the exact pattern from `Test_EventCMCallProgressing` and `Test_EventCMCallHangup`:

```go
package billinghandler

import (
	"context"
	"fmt"
	"testing"
	"time"

	"monorepo/bin-billing-manager/models/account"
	"monorepo/bin-billing-manager/models/billing"
	"monorepo/bin-billing-manager/pkg/accounthandler"
	"monorepo/bin-billing-manager/pkg/dbhandler"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	tmspeaking "monorepo/bin-tts-manager/models/speaking"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func Test_EventTTSSpeakingStarted(t *testing.T) {

	tmCreate := time.Date(2026, 2, 23, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name string

		speaking *tmspeaking.Speaking

		responseAccount *account.Account
		responseUUID    uuid.UUID
		responseBilling *billing.Billing
	}{
		{
			name: "normal",

			speaking: &tmspeaking.Speaking{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("aa000001-0000-0000-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("aa000002-0000-0000-0000-000000000001"),
				},
				TMCreate: &tmCreate,
			},

			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("aa000003-0000-0000-0000-000000000001"),
				},
			},
			responseUUID: uuid.FromStringOrNil("aa000004-0000-0000-0000-000000000001"),
			responseBilling: &billing.Billing{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("aa000004-0000-0000-0000-000000000001"),
				},
				AccountID:         uuid.FromStringOrNil("aa000003-0000-0000-0000-000000000001"),
				TransactionType:   billing.TransactionTypeUsage,
				Status:            billing.StatusProgressing,
				ReferenceType:     billing.ReferenceTypeSpeaking,
				ReferenceID:       uuid.FromStringOrNil("aa000001-0000-0000-0000-000000000001"),
				CostType:          billing.CostTypeTTS,
				RateTokenPerUnit:  billing.DefaultTokenPerUnitTTS,
				RateCreditPerUnit: billing.DefaultCreditPerUnitTTS,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockAccount := accounthandler.NewMockAccountHandler(mc)

			h := billingHandler{
				utilHandler:    mockUtil,
				db:             mockDB,
				notifyHandler:  mockNotify,
				accountHandler: mockAccount,
			}
			ctx := context.Background()

			// idempotency check
			mockDB.EXPECT().BillingGetByReferenceTypeAndID(ctx, billing.ReferenceTypeSpeaking, tt.speaking.ID).Return(nil, dbhandler.ErrNotFound)

			// BillingStart
			mockAccount.EXPECT().GetByCustomerID(ctx, tt.speaking.CustomerID).Return(tt.responseAccount, nil)
			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockDB.EXPECT().BillingCreate(ctx, gomock.Any()).Return(nil)
			mockDB.EXPECT().BillingGet(ctx, tt.responseUUID).Return(tt.responseBilling, nil)
			mockNotify.EXPECT().PublishEvent(ctx, billing.EventTypeBillingCreated, tt.responseBilling)

			if err := h.EventTTSSpeakingStarted(ctx, tt.speaking); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_EventTTSSpeakingStopped(t *testing.T) {

	tmBillingStart := time.Date(2026, 2, 23, 10, 0, 0, 0, time.UTC)
	tmUpdate := time.Date(2026, 2, 23, 10, 1, 0, 0, time.UTC) // 60 seconds later

	tests := []struct {
		name string

		speaking *tmspeaking.Speaking

		responseBilling        *billing.Billing
		responseConsumedBilling *billing.Billing
	}{
		{
			name: "normal",

			speaking: &tmspeaking.Speaking{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("bb000001-0000-0000-0000-000000000001"),
				},
				TMUpdate: &tmUpdate,
			},

			responseBilling: &billing.Billing{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("bb000002-0000-0000-0000-000000000001"),
				},
				AccountID:         uuid.FromStringOrNil("bb000003-0000-0000-0000-000000000001"),
				ReferenceType:     billing.ReferenceTypeSpeaking,
				ReferenceID:       uuid.FromStringOrNil("bb000001-0000-0000-0000-000000000001"),
				CostType:          billing.CostTypeTTS,
				RateTokenPerUnit:  billing.DefaultTokenPerUnitTTS,
				RateCreditPerUnit: billing.DefaultCreditPerUnitTTS,
				TMBillingStart:    &tmBillingStart,
			},
			responseConsumedBilling: &billing.Billing{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("bb000002-0000-0000-0000-000000000001"),
				},
				AccountID:         uuid.FromStringOrNil("bb000003-0000-0000-0000-000000000001"),
				ReferenceType:     billing.ReferenceTypeSpeaking,
				ReferenceID:       uuid.FromStringOrNil("bb000001-0000-0000-0000-000000000001"),
				CostType:          billing.CostTypeTTS,
				RateTokenPerUnit:  billing.DefaultTokenPerUnitTTS,
				RateCreditPerUnit: billing.DefaultCreditPerUnitTTS,
				TMBillingStart:    &tmBillingStart,
				TMBillingEnd:      &tmUpdate,
				Status:            billing.StatusEnd,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockAccount := accounthandler.NewMockAccountHandler(mc)

			h := billingHandler{
				utilHandler:    mockUtil,
				db:             mockDB,
				notifyHandler:  mockNotify,
				accountHandler: mockAccount,
			}
			ctx := context.Background()

			mockDB.EXPECT().BillingGetByReferenceID(ctx, tt.speaking.ID).Return(tt.responseBilling, nil)

			// BillingEnd - atomic consume and record
			// 60s duration -> ceil(60/60) = 1 billable unit
			mockDB.EXPECT().BillingConsumeAndRecord(
				ctx,
				tt.responseBilling,
				tt.responseBilling.AccountID,
				1,  // billableUnits
				60, // usageDuration (seconds)
				billing.GetCostInfo(tt.responseBilling.CostType),
				tt.speaking.TMUpdate,
			).Return(tt.responseConsumedBilling, nil)
			mockNotify.EXPECT().PublishEvent(ctx, billing.EventTypeBillingUpdated, tt.responseConsumedBilling)

			if err := h.EventTTSSpeakingStopped(ctx, tt.speaking); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_EventTTSSpeakingStopped_billing_not_found(t *testing.T) {

	tmUpdate := time.Date(2026, 2, 23, 10, 1, 0, 0, time.UTC)

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockAccount := accounthandler.NewMockAccountHandler(mc)

	h := billingHandler{
		utilHandler:    mockUtil,
		db:             mockDB,
		notifyHandler:  mockNotify,
		accountHandler: mockAccount,
	}
	ctx := context.Background()

	s := &tmspeaking.Speaking{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("cc000001-0000-0000-0000-000000000001"),
		},
		TMUpdate: &tmUpdate,
	}

	mockDB.EXPECT().BillingGetByReferenceID(ctx, s.ID).Return(nil, fmt.Errorf("not found"))

	// Should return nil (silently ignores)
	if err := h.EventTTSSpeakingStopped(ctx, s); err != nil {
		t.Errorf("Wrong match. expect: nil, got: %v", err)
	}
}

func Test_EventTTSSpeakingStopped_nil_tmupdate(t *testing.T) {

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockAccount := accounthandler.NewMockAccountHandler(mc)

	h := billingHandler{
		utilHandler:    mockUtil,
		db:             mockDB,
		notifyHandler:  mockNotify,
		accountHandler: mockAccount,
	}
	ctx := context.Background()

	s := &tmspeaking.Speaking{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("dd000001-0000-0000-0000-000000000001"),
		},
		TMUpdate: nil,
	}

	mockDB.EXPECT().BillingGetByReferenceID(ctx, s.ID).Return(&billing.Billing{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("dd000002-0000-0000-0000-000000000001"),
		},
	}, nil)

	// Should return error
	err := h.EventTTSSpeakingStopped(ctx, s)
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
}
```

**Step 5: Run tests**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-tts-billing/bin-billing-manager && go test ./pkg/billinghandler/...`
Expected: PASS

**Step 6: Commit**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-tts-billing
git add bin-billing-manager/pkg/billinghandler/
git commit -m "NOJIRA-add-tts-billing

- bin-billing-manager: Add EventTTSSpeakingStarted and EventTTSSpeakingStopped handlers
- bin-billing-manager: Add TTS event methods to BillingHandler interface"
```

---

### Task 6: Add TTS subscribe handler and wire up events (bin-billing-manager)

**Files:**
- Create: `bin-billing-manager/pkg/subscribehandler/tts.go`
- Create: `bin-billing-manager/pkg/subscribehandler/tts_test.go`
- Modify: `bin-billing-manager/pkg/subscribehandler/main.go` — add event cases and import
- Modify: `bin-billing-manager/cmd/billing-manager/main.go` — add QueueNameTTSEvent to subscribe targets

**Step 1: Create tts.go**

Follow the pattern from `call.go`:

```go
package subscribehandler

import (
	"context"
	"encoding/json"

	"monorepo/bin-common-handler/models/sock"
	tmspeaking "monorepo/bin-tts-manager/models/speaking"

	"github.com/pkg/errors"
)

// processEventTTSSpeakingStarted handles the tts-manager's speaking_started event
func (h *subscribeHandler) processEventTTSSpeakingStarted(ctx context.Context, m *sock.Event) error {
	var s tmspeaking.Speaking
	if err := json.Unmarshal([]byte(m.Data), &s); err != nil {
		return errors.Wrapf(err, "could not unmarshal the data. processEventTTSSpeakingStarted. err: %v", err)
	}

	if errEvent := h.billingHandler.EventTTSSpeakingStarted(ctx, &s); errEvent != nil {
		return errors.Wrapf(errEvent, "could not handle the event. processEventTTSSpeakingStarted. err: %v", errEvent)
	}

	return nil
}

// processEventTTSSpeakingStopped handles the tts-manager's speaking_stopped event
func (h *subscribeHandler) processEventTTSSpeakingStopped(ctx context.Context, m *sock.Event) error {
	var s tmspeaking.Speaking
	if err := json.Unmarshal([]byte(m.Data), &s); err != nil {
		return errors.Wrapf(err, "could not unmarshal the data. processEventTTSSpeakingStopped. err: %v", err)
	}

	if errEvent := h.billingHandler.EventTTSSpeakingStopped(ctx, &s); errEvent != nil {
		return errors.Wrapf(errEvent, "could not handle the event. processEventTTSSpeakingStopped. err: %v", errEvent)
	}

	return nil
}
```

**Step 2: Create tts_test.go**

Follow the pattern from `call_test.go`:

```go
package subscribehandler

import (
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	tmspeaking "monorepo/bin-tts-manager/models/speaking"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-billing-manager/pkg/billinghandler"
)

func Test_processEventTTSSpeakingStarted(t *testing.T) {

	tests := []struct {
		name  string
		event *sock.Event

		expectSpeaking *tmspeaking.Speaking
	}{
		{
			name: "normal",

			event: &sock.Event{
				Publisher: string(commonoutline.ServiceNameTTSManager),
				Type:      tmspeaking.EventTypeSpeakingStarted,
				DataType:  "application/json",
				Data:      []byte(`{"id":"aa111111-0000-0000-0000-000000000001"}`),
			},

			expectSpeaking: &tmspeaking.Speaking{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("aa111111-0000-0000-0000-000000000001"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockBilling := billinghandler.NewMockBillingHandler(mc)

			h := subscribeHandler{
				sockHandler:    mockSock,
				billingHandler: mockBilling,
			}

			mockBilling.EXPECT().EventTTSSpeakingStarted(gomock.Any(), tt.expectSpeaking).Return(nil)

			if err := h.processEvent(tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_processEventTTSSpeakingStopped(t *testing.T) {

	tests := []struct {
		name  string
		event *sock.Event

		expectSpeaking *tmspeaking.Speaking
	}{
		{
			name: "normal",

			event: &sock.Event{
				Publisher: string(commonoutline.ServiceNameTTSManager),
				Type:      tmspeaking.EventTypeSpeakingStopped,
				DataType:  "application/json",
				Data:      []byte(`{"id":"bb111111-0000-0000-0000-000000000001"}`),
			},

			expectSpeaking: &tmspeaking.Speaking{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("bb111111-0000-0000-0000-000000000001"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockBilling := billinghandler.NewMockBillingHandler(mc)

			h := subscribeHandler{
				sockHandler:    mockSock,
				billingHandler: mockBilling,
			}

			mockBilling.EXPECT().EventTTSSpeakingStopped(gomock.Any(), tt.expectSpeaking).Return(nil)

			if err := h.processEvent(tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
```

**Step 3: Add event cases in `main.go`**

In `bin-billing-manager/pkg/subscribehandler/main.go`:

Add import: `tmspeaking "monorepo/bin-tts-manager/models/speaking"`

Add cases in `processEvent()` switch, before the default case:
```go
//// tts-manager
// speaking
case m.Publisher == string(commonoutline.ServiceNameTTSManager) && m.Type == tmspeaking.EventTypeSpeakingStarted:
	err = h.processEventTTSSpeakingStarted(ctx, m)

case m.Publisher == string(commonoutline.ServiceNameTTSManager) && m.Type == tmspeaking.EventTypeSpeakingStopped:
	err = h.processEventTTSSpeakingStopped(ctx, m)
```

**Step 4: Add subscribe target in `cmd/billing-manager/main.go`**

In `cmd/billing-manager/main.go`, add to the `subscribeTargets` slice (around line 162-168):
```go
string(commonoutline.QueueNameTTSEvent),
```

**Step 5: Run tests**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-tts-billing/bin-billing-manager && go test ./pkg/subscribehandler/...`
Expected: PASS

**Step 6: Run full verification workflow for bin-billing-manager**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-tts-billing/bin-billing-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`
Expected: All pass

**Step 7: Commit**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-tts-billing
git add bin-billing-manager/
git commit -m "NOJIRA-add-tts-billing

- bin-billing-manager: Add TTS subscribe handler for speaking_started and speaking_stopped events
- bin-billing-manager: Subscribe to QueueNameTTSEvent in billing-manager startup"
```

---

### Task 7: Final verification and push

**Step 1: Run full verification for both services**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-tts-billing/bin-tts-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-tts-billing/bin-billing-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 2: Check for conflicts with main**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-tts-billing
git fetch origin main
git merge-tree $(git merge-base HEAD origin/main) HEAD origin/main | grep -E "^(CONFLICT|changed in both)"
```

**Step 3: Push and create PR**

```bash
git push -u origin NOJIRA-add-tts-billing
```

Create PR with title: `NOJIRA-add-tts-billing`
