# Recording Billing Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Charge for recordings at 3 tokens/min ($0.03/min overflow), with balance check before recording starts.

**Architecture:** Two-phase billing (start/end) triggered by recording events from call-manager, handled by billing-manager. Balance check in call-manager's recordinghandler before creating Asterisk resources.

**Tech Stack:** Go, RabbitMQ events, gomock tests, OpenAPI YAML, RST docs

**Design doc:** `docs/plans/2026-02-26-recording-billing-design.md`

**Worktree:** `~/gitvoipbin/monorepo-worktrees/NOJIRA-add-recording-billing`

---

### Task 1: Add billing model constants for recording

**Files:**
- Modify: `bin-billing-manager/models/billing/billing.go:80` (add ReferenceTypeRecording after ReferenceTypeSpeaking)
- Modify: `bin-billing-manager/models/billing/cost_type.go:34` (add CostTypeRecording), `:45-46` (add default rates), `:51-52` (add token rate), `:71-72` (add GetCostInfo case)

**Step 1: Add ReferenceTypeRecording**

In `bin-billing-manager/models/billing/billing.go`, after line 80 (`ReferenceTypeSpeaking`), add:

```go
	ReferenceTypeRecording         ReferenceType = "recording"
```

**Step 2: Add CostTypeRecording and default rates**

In `bin-billing-manager/models/billing/cost_type.go`:

After line 34 (`CostTypeTTS`), add:
```go
	CostTypeRecording        CostType = "recording"
```

After line 45 (`DefaultCreditPerUnitTTS`), add:
```go
	DefaultCreditPerUnitRecording int64 = 30000   // $0.03/min
```

After line 51 (`DefaultTokenPerUnitTTS`), add:
```go
	DefaultTokenPerUnitRecording int64 = 3
```

**Step 3: Add GetCostInfo case**

In `bin-billing-manager/models/billing/cost_type.go`, after the `CostTypeTTS` case (line 72), add:

```go
	case CostTypeRecording:
		return CostInfo{CostModeTokenFirst, DefaultTokenPerUnitRecording, DefaultCreditPerUnitRecording}
```

**Step 4: Verify it compiles**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-recording-billing/bin-billing-manager && go build ./...`
Expected: success

**Step 5: Commit**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-recording-billing
git add bin-billing-manager/models/billing/billing.go bin-billing-manager/models/billing/cost_type.go
git commit -m "NOJIRA-add-recording-billing

- bin-billing-manager: Add ReferenceTypeRecording and CostTypeRecording billing model constants
- bin-billing-manager: Add recording rates (3 tokens/min, \$0.03/min) and GetCostInfo case"
```

---

### Task 2: Add balance check for recording in billing-manager

**Files:**
- Modify: `bin-billing-manager/pkg/accounthandler/balance.go:108-118` (add case before default)
- Modify: `bin-billing-manager/pkg/accounthandler/balance_test.go` (add test cases)

**Step 1: Write the failing test**

In `bin-billing-manager/pkg/accounthandler/balance_test.go`, add these test cases to the `tests` slice in `Test_IsValidBalance` (before the closing `}` of the slice at the `unsupported billing type` test case):

```go
		{
			name: "recording with enough tokens returns true",

			accountID:   uuid.FromStringOrNil("c1c1c1c1-1111-11ee-86c6-111111111111"),
			billingType: billing.ReferenceTypeRecording,
			count:       1,

			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("c1c1c1c1-1111-11ee-86c6-111111111111"),
				},
				BalanceToken:  10,
				BalanceCredit: 0,
				TMDelete:      nil,
			},
			expectRes: true,
		},
		{
			name: "recording with no tokens but enough credit returns true",

			accountID:   uuid.FromStringOrNil("c2c2c2c2-2222-11ee-86c6-222222222222"),
			billingType: billing.ReferenceTypeRecording,
			count:       1,

			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("c2c2c2c2-2222-11ee-86c6-222222222222"),
				},
				BalanceToken:  0,
				BalanceCredit: 100000,
				TMDelete:      nil,
			},
			expectRes: true,
		},
		{
			name: "recording with no tokens and insufficient credit returns false",

			accountID:   uuid.FromStringOrNil("c3c3c3c3-3333-11ee-86c6-333333333333"),
			billingType: billing.ReferenceTypeRecording,
			count:       1,

			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("c3c3c3c3-3333-11ee-86c6-333333333333"),
				},
				BalanceToken:  0,
				BalanceCredit: 1,
				TMDelete:      nil,
			},
			expectRes: false,
		},
```

**Step 2: Run test to verify it fails**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-recording-billing/bin-billing-manager && go test -v ./pkg/accounthandler/ -run Test_IsValidBalance`
Expected: FAIL — `recording` hits the `default` case returning `unsupported billing type` error.

**Step 3: Write the implementation**

In `bin-billing-manager/pkg/accounthandler/balance.go`, add this case before the `default:` case (before line 116):

```go
	case billing.ReferenceTypeRecording:
		if a.BalanceToken > 0 {
			promAccountBalanceCheckTotal.WithLabelValues("valid").Inc()
			return true, nil
		}
		costInfo := billing.GetCostInfo(billing.CostTypeRecording)
		expectCost := costInfo.CreditPerUnit * int64(count)
		if a.BalanceCredit >= expectCost {
			promAccountBalanceCheckTotal.WithLabelValues("valid").Inc()
			return true, nil
		}
```

**Step 4: Run test to verify it passes**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-recording-billing/bin-billing-manager && go test -v ./pkg/accounthandler/ -run Test_IsValidBalance`
Expected: PASS

**Step 5: Commit**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-recording-billing
git add bin-billing-manager/pkg/accounthandler/balance.go bin-billing-manager/pkg/accounthandler/balance_test.go
git commit -m "NOJIRA-add-recording-billing

- bin-billing-manager: Add ReferenceTypeRecording balance check (token-first, credit overflow)
- bin-billing-manager: Add balance_test.go cases for recording with tokens, credit, insufficient"
```

---

### Task 3: Add recording billing event handlers

**Files:**
- Create: `bin-billing-manager/pkg/billinghandler/event_recording.go`
- Create: `bin-billing-manager/pkg/billinghandler/event_recording_test.go`
- Modify: `bin-billing-manager/pkg/billinghandler/main.go:50` (add interface methods)

**Step 1: Write the test file**

Create `bin-billing-manager/pkg/billinghandler/event_recording_test.go`:

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
	cmrecording "monorepo/bin-call-manager/models/recording"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func Test_EventCMRecordingStarted(t *testing.T) {

	tmCreate := time.Date(2026, 2, 26, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name string

		recording *cmrecording.Recording

		responseAccount *account.Account
		responseUUID    uuid.UUID
		responseBilling *billing.Billing
	}{
		{
			name: "normal",

			recording: &cmrecording.Recording{
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
				ReferenceType:     billing.ReferenceTypeRecording,
				ReferenceID:       uuid.FromStringOrNil("aa000001-0000-0000-0000-000000000001"),
				CostType:          billing.CostTypeRecording,
				RateTokenPerUnit:  billing.DefaultTokenPerUnitRecording,
				RateCreditPerUnit: billing.DefaultCreditPerUnitRecording,
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
			mockDB.EXPECT().BillingGetByReferenceTypeAndID(ctx, billing.ReferenceTypeRecording, tt.recording.ID).Return(nil, dbhandler.ErrNotFound)

			// BillingStart
			mockAccount.EXPECT().GetByCustomerID(ctx, tt.recording.CustomerID).Return(tt.responseAccount, nil)
			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockDB.EXPECT().BillingCreate(ctx, gomock.Any()).Return(nil)
			mockDB.EXPECT().BillingGet(ctx, tt.responseUUID).Return(tt.responseBilling, nil)
			mockNotify.EXPECT().PublishEvent(ctx, billing.EventTypeBillingCreated, tt.responseBilling)

			if err := h.EventCMRecordingStarted(ctx, tt.recording); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_EventCMRecordingFinished(t *testing.T) {

	tmBillingStart := time.Date(2026, 2, 26, 10, 0, 0, 0, time.UTC)
	tmUpdate := time.Date(2026, 2, 26, 10, 1, 0, 0, time.UTC) // 60 seconds later

	tests := []struct {
		name string

		recording *cmrecording.Recording

		responseBilling         *billing.Billing
		responseConsumedBilling *billing.Billing
	}{
		{
			name: "normal",

			recording: &cmrecording.Recording{
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
				ReferenceType:     billing.ReferenceTypeRecording,
				ReferenceID:       uuid.FromStringOrNil("bb000001-0000-0000-0000-000000000001"),
				CostType:          billing.CostTypeRecording,
				RateTokenPerUnit:  billing.DefaultTokenPerUnitRecording,
				RateCreditPerUnit: billing.DefaultCreditPerUnitRecording,
				TMBillingStart:    &tmBillingStart,
			},
			responseConsumedBilling: &billing.Billing{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("bb000002-0000-0000-0000-000000000001"),
				},
				AccountID:         uuid.FromStringOrNil("bb000003-0000-0000-0000-000000000001"),
				ReferenceType:     billing.ReferenceTypeRecording,
				ReferenceID:       uuid.FromStringOrNil("bb000001-0000-0000-0000-000000000001"),
				CostType:          billing.CostTypeRecording,
				RateTokenPerUnit:  billing.DefaultTokenPerUnitRecording,
				RateCreditPerUnit: billing.DefaultCreditPerUnitRecording,
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

			mockDB.EXPECT().BillingGetByReferenceID(ctx, tt.recording.ID).Return(tt.responseBilling, nil)

			// BillingEnd - atomic consume and record
			// 60s duration -> ceil(60/60) = 1 billable unit
			mockDB.EXPECT().BillingConsumeAndRecord(
				ctx,
				tt.responseBilling,
				tt.responseBilling.AccountID,
				1,  // billableUnits
				60, // usageDuration (seconds)
				billing.GetCostInfo(tt.responseBilling.CostType),
				tt.recording.TMUpdate,
			).Return(tt.responseConsumedBilling, nil)
			mockNotify.EXPECT().PublishEvent(ctx, billing.EventTypeBillingUpdated, tt.responseConsumedBilling)

			if err := h.EventCMRecordingFinished(ctx, tt.recording); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_EventCMRecordingFinished_billing_not_found(t *testing.T) {

	tmUpdate := time.Date(2026, 2, 26, 10, 1, 0, 0, time.UTC)

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

	r := &cmrecording.Recording{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("cc000001-0000-0000-0000-000000000001"),
		},
		TMUpdate: &tmUpdate,
	}

	mockDB.EXPECT().BillingGetByReferenceID(ctx, r.ID).Return(nil, fmt.Errorf("not found"))

	// Should return nil (silently ignores)
	if err := h.EventCMRecordingFinished(ctx, r); err != nil {
		t.Errorf("Wrong match. expect: nil, got: %v", err)
	}
}

func Test_EventCMRecordingFinished_nil_tmupdate(t *testing.T) {

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

	r := &cmrecording.Recording{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("dd000001-0000-0000-0000-000000000001"),
		},
		TMUpdate: nil,
	}

	mockDB.EXPECT().BillingGetByReferenceID(ctx, r.ID).Return(&billing.Billing{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("dd000002-0000-0000-0000-000000000001"),
		},
		ReferenceType: billing.ReferenceTypeRecording,
	}, nil)

	// Should return error
	err := h.EventCMRecordingFinished(ctx, r)
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-recording-billing/bin-billing-manager && go test -v ./pkg/billinghandler/ -run Test_EventCMRecording`
Expected: FAIL — `EventCMRecordingStarted` and `EventCMRecordingFinished` don't exist yet.

**Step 3: Write the implementation**

Create `bin-billing-manager/pkg/billinghandler/event_recording.go`:

```go
package billinghandler

import (
	"context"

	commonaddress "monorepo/bin-common-handler/models/address"
	cmrecording "monorepo/bin-call-manager/models/recording"

	"monorepo/bin-billing-manager/models/billing"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// EventCMRecordingStarted handles the call-manager's recording_started event
func (h *billingHandler) EventCMRecordingStarted(ctx context.Context, r *cmrecording.Recording) error {
	log := logrus.WithFields(logrus.Fields{
		"func":         "EventCMRecordingStarted",
		"recording_id": r.ID,
		"customer_id":  r.CustomerID,
	})
	log.Debugf("Received recording_started event. recording_id: %s", r.ID)

	if errBilling := h.BillingStart(
		ctx,
		r.CustomerID,
		billing.ReferenceTypeRecording,
		r.ID,
		billing.CostTypeRecording,
		r.TMCreate,
		&commonaddress.Address{},
		&commonaddress.Address{},
	); errBilling != nil {
		return errors.Wrap(errBilling, "could not start a billing")
	}

	return nil
}

// EventCMRecordingFinished handles the call-manager's recording_finished event
func (h *billingHandler) EventCMRecordingFinished(ctx context.Context, r *cmrecording.Recording) error {
	log := logrus.WithFields(logrus.Fields{
		"func":         "EventCMRecordingFinished",
		"recording_id": r.ID,
	})
	log.Debugf("Received recording_finished event. recording_id: %s", r.ID)

	b, err := h.GetByReferenceID(ctx, r.ID)
	if err != nil {
		// could not get billing. nothing to do.
		return nil
	}

	if r.TMUpdate == nil {
		return errors.Errorf("invalid tm_update. recording_id: %s, tm_update: nil", r.ID)
	}

	if errEnd := h.BillingEnd(ctx, b, r.TMUpdate, &commonaddress.Address{}, &commonaddress.Address{}); errEnd != nil {
		return errors.Wrapf(errEnd, "could not end the billing. billing_id: %s, recording_id: %s", b.ID, r.ID)
	}

	return nil
}
```

**Step 4: Add interface methods**

In `bin-billing-manager/pkg/billinghandler/main.go`, add the import for recording model and two interface methods. After line 9 (`cmcall "monorepo/bin-call-manager/models/call"`), add:

```go
	cmrecording "monorepo/bin-call-manager/models/recording"
```

After line 50 (`EventTTSSpeakingStopped`), add:

```go
	EventCMRecordingStarted(ctx context.Context, r *cmrecording.Recording) error
	EventCMRecordingFinished(ctx context.Context, r *cmrecording.Recording) error
```

**Step 5: Regenerate mocks and run tests**

Run:
```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-recording-billing/bin-billing-manager && \
go generate ./pkg/billinghandler/... && \
go test -v ./pkg/billinghandler/ -run Test_EventCMRecording
```
Expected: PASS (all 4 test functions)

**Step 6: Commit**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-recording-billing
git add bin-billing-manager/pkg/billinghandler/event_recording.go \
        bin-billing-manager/pkg/billinghandler/event_recording_test.go \
        bin-billing-manager/pkg/billinghandler/main.go \
        bin-billing-manager/pkg/billinghandler/mock_main.go
git commit -m "NOJIRA-add-recording-billing

- bin-billing-manager: Add EventCMRecordingStarted and EventCMRecordingFinished billing handlers
- bin-billing-manager: Add BillingHandler interface methods for recording events
- bin-billing-manager: Add tests for recording billing start, finish, not-found, nil-tmupdate"
```

---

### Task 4: Wire recording events into subscribe handler

**Files:**
- Create: `bin-billing-manager/pkg/subscribehandler/recording.go`
- Create: `bin-billing-manager/pkg/subscribehandler/recording_test.go`
- Modify: `bin-billing-manager/pkg/subscribehandler/main.go:10,198` (add import and switch cases)

**Step 1: Write the test file**

Create `bin-billing-manager/pkg/subscribehandler/recording_test.go`:

```go
package subscribehandler

import (
	"testing"

	cmrecording "monorepo/bin-call-manager/models/recording"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-billing-manager/pkg/billinghandler"
)

func Test_processEventCMRecordingStarted(t *testing.T) {

	tests := []struct {
		name  string
		event *sock.Event

		expectRecording *cmrecording.Recording
	}{
		{
			name: "normal",

			event: &sock.Event{
				Publisher: string(commonoutline.ServiceNameCallManager),
				Type:      cmrecording.EventTypeRecordingStarted,
				DataType:  "application/json",
				Data:      []byte(`{"id":"aa111111-0000-0000-0000-000000000001"}`),
			},

			expectRecording: &cmrecording.Recording{
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

			mockBilling.EXPECT().EventCMRecordingStarted(gomock.Any(), tt.expectRecording).Return(nil)

			if err := h.processEvent(tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_processEventCMRecordingFinished(t *testing.T) {

	tests := []struct {
		name  string
		event *sock.Event

		expectRecording *cmrecording.Recording
	}{
		{
			name: "normal",

			event: &sock.Event{
				Publisher: string(commonoutline.ServiceNameCallManager),
				Type:      cmrecording.EventTypeRecordingFinished,
				DataType:  "application/json",
				Data:      []byte(`{"id":"bb111111-0000-0000-0000-000000000001"}`),
			},

			expectRecording: &cmrecording.Recording{
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

			mockBilling.EXPECT().EventCMRecordingFinished(gomock.Any(), tt.expectRecording).Return(nil)

			if err := h.processEvent(tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-recording-billing/bin-billing-manager && go test -v ./pkg/subscribehandler/ -run Test_processEventCMRecording`
Expected: FAIL — the `processEvent` switch doesn't handle recording events yet (falls through to default/no-op).

**Step 3: Write the subscribe handler**

Create `bin-billing-manager/pkg/subscribehandler/recording.go`:

```go
package subscribehandler

import (
	"context"
	"encoding/json"

	cmrecording "monorepo/bin-call-manager/models/recording"
	"monorepo/bin-common-handler/models/sock"

	"github.com/pkg/errors"
)

// processEventCMRecordingStarted handles the call-manager's recording_started event
func (h *subscribeHandler) processEventCMRecordingStarted(ctx context.Context, m *sock.Event) error {
	var r cmrecording.Recording
	if err := json.Unmarshal([]byte(m.Data), &r); err != nil {
		return errors.Wrapf(err, "could not unmarshal the data. processEventCMRecordingStarted. err: %v", err)
	}

	if errEvent := h.billingHandler.EventCMRecordingStarted(ctx, &r); errEvent != nil {
		return errors.Wrapf(errEvent, "could not handle the event. processEventCMRecordingStarted. err: %v", errEvent)
	}

	return nil
}

// processEventCMRecordingFinished handles the call-manager's recording_finished event
func (h *subscribeHandler) processEventCMRecordingFinished(ctx context.Context, m *sock.Event) error {
	var r cmrecording.Recording
	if err := json.Unmarshal([]byte(m.Data), &r); err != nil {
		return errors.Wrapf(err, "could not unmarshal the data. processEventCMRecordingFinished. err: %v", err)
	}

	if errEvent := h.billingHandler.EventCMRecordingFinished(ctx, &r); errEvent != nil {
		return errors.Wrapf(errEvent, "could not handle the event. processEventCMRecordingFinished. err: %v", errEvent)
	}

	return nil
}
```

**Step 4: Wire into event router**

In `bin-billing-manager/pkg/subscribehandler/main.go`:

Add import after line 10 (`cmcall "monorepo/bin-call-manager/models/call"`):

```go
	cmrecording "monorepo/bin-call-manager/models/recording"
```

Add switch cases after the TTS section (after line 198, before the default section). Insert:

```go

	//// call-manager
	// recording
	case m.Publisher == string(commonoutline.ServiceNameCallManager) && m.Type == cmrecording.EventTypeRecordingStarted:
		err = h.processEventCMRecordingStarted(ctx, m)

	case m.Publisher == string(commonoutline.ServiceNameCallManager) && m.Type == cmrecording.EventTypeRecordingFinished:
		err = h.processEventCMRecordingFinished(ctx, m)
```

**Step 5: Regenerate mocks and run tests**

Run:
```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-recording-billing/bin-billing-manager && \
go generate ./pkg/subscribehandler/... && \
go test -v ./pkg/subscribehandler/ -run Test_processEventCMRecording
```
Expected: PASS

**Step 6: Commit**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-recording-billing
git add bin-billing-manager/pkg/subscribehandler/recording.go \
        bin-billing-manager/pkg/subscribehandler/recording_test.go \
        bin-billing-manager/pkg/subscribehandler/main.go \
        bin-billing-manager/pkg/subscribehandler/mock_main.go
git commit -m "NOJIRA-add-recording-billing

- bin-billing-manager: Add subscribe handler for recording_started and recording_finished events
- bin-billing-manager: Wire recording events into processEvent switch from call-manager publisher
- bin-billing-manager: Add tests for recording event subscribe processing"
```

---

### Task 5: Run full verification for bin-billing-manager

**Step 1: Run full verification workflow**

Run:
```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-recording-billing/bin-billing-manager && \
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```
Expected: All pass. If lint or test issues, fix them before proceeding.

**Step 2: Commit any generated/vendor changes**

If `go mod tidy`, `go mod vendor`, or `go generate` produced changes:

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-recording-billing
git add bin-billing-manager/
git commit -m "NOJIRA-add-recording-billing

- bin-billing-manager: Run full verification (mod tidy, vendor, generate, test, lint)"
```

---

### Task 6: Add balance check in call-manager recording handler

**Files:**
- Modify: `bin-call-manager/pkg/recordinghandler/recording.go:38-41,141-143` (add balance checks)
- Modify: `bin-call-manager/pkg/recordinghandler/recording_test.go` (add balance check mock expectations)

**Step 1: Update existing test to expect balance check**

In `bin-call-manager/pkg/recordinghandler/recording_test.go`:

For `Test_recordingReferenceTypeCall`, add the import `"monorepo/bin-billing-manager/models/billing"` to the imports. Then add the balance check mock expectation after `mockChannel.EXPECT().Get(...)` (after the line for `mockChannel.EXPECT().Get(ctx, tt.responseCall.ChannelID)...`):

```go
			mockReq.EXPECT().BillingV1AccountIsValidBalanceByCustomerID(ctx, tt.responseCall.CustomerID, billing.ReferenceTypeRecording, "", 1).Return(true, nil)
```

For `Test_recordingReferenceTypeConfbridge`, add the balance check mock expectation after `mockBridge.EXPECT().Get(...)`:

```go
			mockReq.EXPECT().BillingV1AccountIsValidBalanceByCustomerID(ctx, tt.responseConfbridge.CustomerID, billing.ReferenceTypeRecording, "", 1).Return(true, nil)
```

**Step 2: Run test to verify it fails**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-recording-billing/bin-call-manager && go test -v ./pkg/recordinghandler/ -run "Test_recordingReferenceTypeCall|Test_recordingReferenceTypeConfbridge"`
Expected: FAIL — tests expect a `BillingV1AccountIsValidBalanceByCustomerID` call that doesn't happen yet.

**Step 3: Add balance check to recordingReferenceTypeCall**

In `bin-call-manager/pkg/recordinghandler/recording.go`, add import for billing:

```go
	"monorepo/bin-billing-manager/models/billing"
```

In `recordingReferenceTypeCall`, after getting the channel (after line 46, the `}` closing the channel error check) and before the activeflowID nil check (line 48), add:

```go

	// check customer balance before starting recording
	validBalance, errBalance := h.reqHandler.BillingV1AccountIsValidBalanceByCustomerID(ctx, c.CustomerID, billing.ReferenceTypeRecording, "", 1)
	if errBalance != nil {
		return nil, errors.Wrap(errBalance, "could not check balance for recording")
	}
	if !validBalance {
		return nil, fmt.Errorf("insufficient balance for recording. customer_id: %s", c.CustomerID)
	}
```

**Step 4: Add balance check to recordingReferenceTypeConfbridge**

In `recordingReferenceTypeConfbridge`, after getting the confbridge (after line 143, `}` closing confbridge error check), before the activeflowID nil check (line 146), add:

```go

	// check customer balance before starting recording
	validBalance, errBalance := h.reqHandler.BillingV1AccountIsValidBalanceByCustomerID(ctx, cb.CustomerID, billing.ReferenceTypeRecording, "", 1)
	if errBalance != nil {
		return nil, errors.Wrap(errBalance, "could not check balance for recording")
	}
	if !validBalance {
		return nil, fmt.Errorf("insufficient balance for recording. customer_id: %s", cb.CustomerID)
	}
```

**Step 5: Run test to verify it passes**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-recording-billing/bin-call-manager && go test -v ./pkg/recordinghandler/ -run "Test_recordingReferenceTypeCall|Test_recordingReferenceTypeConfbridge"`
Expected: PASS

**Step 6: Commit**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-recording-billing
git add bin-call-manager/pkg/recordinghandler/recording.go \
        bin-call-manager/pkg/recordinghandler/recording_test.go
git commit -m "NOJIRA-add-recording-billing

- bin-call-manager: Add balance check before recording start for both call and confbridge
- bin-call-manager: Reject recording if customer has insufficient tokens and credits
- bin-call-manager: Update recording tests to expect balance check call"
```

---

### Task 7: Run full verification for bin-call-manager

**Step 1: Run full verification workflow**

Run:
```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-recording-billing/bin-call-manager && \
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```
Expected: All pass.

**Step 2: Commit any generated/vendor changes**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-recording-billing
git add bin-call-manager/
git commit -m "NOJIRA-add-recording-billing

- bin-call-manager: Run full verification (mod tidy, vendor, generate, test, lint)"
```

---

### Task 8: Update OpenAPI enums and regenerate

**Files:**
- Modify: `bin-openapi-manager/openapi/openapi.yaml:485,498,514,526` (add recording to enums)

**Step 1: Add recording to BillingManagerBillingreferenceType enum**

In `bin-openapi-manager/openapi/openapi.yaml`, after line 485 (`- speaking`), add:

```yaml
        - recording
```

After line 498 (`- BillingManagerBillingreferenceTypeSpeaking`), add:

```yaml
        - BillingManagerBillingreferenceTypeRecording
```

**Step 2: Add recording to BillingManagerBillingCostType enum**

After line 514 (`- tts`), add:

```yaml
        - recording
```

After line 526 (`- BillingManagerBillingCostTypeTTS`), add:

```yaml
        - BillingManagerBillingCostTypeRecording
```

**Step 3: Regenerate OpenAPI types**

Run:
```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-recording-billing/bin-openapi-manager && \
go generate ./...
```

**Step 4: Regenerate API server code**

Run:
```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-recording-billing/bin-api-manager && \
go generate ./...
```

**Step 5: Run verification for both services**

Run:
```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-recording-billing/bin-openapi-manager && \
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-recording-billing/bin-api-manager && \
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```
Expected: All pass.

**Step 6: Commit**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-recording-billing
git add bin-openapi-manager/ bin-api-manager/
git commit -m "NOJIRA-add-recording-billing

- bin-openapi-manager: Add recording to BillingManagerBillingreferenceType and BillingManagerBillingCostType enums
- bin-openapi-manager: Regenerate OpenAPI types
- bin-api-manager: Regenerate API server code from updated OpenAPI spec"
```

---

### Task 9: Update RST billing documentation

**Files:**
- Modify: `bin-api-manager/docsdev/source/billing_account_overview.rst`

**Step 1: Update all RST sections**

Update the following sections in `billing_account_overview.rst`:

1. **Line 28** — Change:
   `Each plan tier includes a monthly allocation of tokens that cover certain service types (virtual number calls and TTS).`
   To:
   `Each plan tier includes a monthly allocation of tokens that cover certain service types (virtual number calls, TTS, and recording).`

2. **Lines 56-67** — Update ASCII diagram to add Recording box. Replace the service boxes section with:
   ```
       +----------+  +----------+  +----------+  +----------+  +----------+  +----------+  +----------+
       | VN Calls |  |   TTS    |  |Recording |  |PSTN Calls|  |   SMS    |  |  Email   |  | Numbers  |
       | 1 tok/min|  | 3 tok/min|  | 3 tok/min|  | per min  |  | per msg  |  | per msg  |  | per num  |
       +----------+  +----------+  +----------+  +----------+  +----------+  +----------+  +----------+
            |             |             |             |             |              |             |
            | token first | token first | token first |             |              |             |
            | then credit | then credit | then credit | credit only | credit only  | credit only | credit only
            v             v             v             v             v              v             v
       +---------+   +---------+   +---------+   +---------+   +---------+   +---------+   +---------+
       | $0.001  |   |  $0.03  |   |  $0.03  |   |  $0.01  |   |  $0.01  |   |  $0.01  |   |  $5.00  |
       | /minute |   | /minute |   | /minute |   | /minute |   | /message|   | /message|   | /number |
       +---------+   +---------+   +---------+   +---------+   +---------+   +---------+   +---------+
   ```

3. **Line 74** — Change:
   `- **Token-Eligible Services**: VN calls (1 token/minute) and TTS (3 tokens/minute) consume tokens first, then overflow to credits.`
   To:
   `- **Token-Eligible Services**: VN calls (1 token/minute), TTS (3 tokens/minute), and Recording (3 tokens/minute) consume tokens first, then overflow to credits.`

4. **Lines 139-140** — Add row to Token Rates table. After the TTS row, add:
   ```
   +----------------------+------------------+----------------------------------------+
   | Recording            | 3 tokens         | Per minute (ceiling-rounded)           |
   +----------------------+------------------+----------------------------------------+
   ```

5. **Lines 163-164** — Add row to Credit Rates table. After TTS (overflow) row, add:
   ```
   +----------------------+------------------+------------------+-------------------------+
   | Recording (overflow) | $0.03            | 30,000           | Per minute              |
   +----------------------+------------------+------------------+-------------------------+
   ```

6. **Line 170** — Change:
   `When a token-eligible service is used (VN call or TTS):`
   To:
   `When a token-eligible service is used (VN call, TTS, or Recording):`

7. **After line 223** (after the TTS billing example), add a recording example:
   ```

       Recording (3 minutes 20 seconds) with tokens available:
       +--------------------------------------------+
       | Duration: 3 min 20 sec -> 4 minutes        |
       | (ceiling-rounded to next whole minute)      |
       | Token cost: 4 x 3 = 12 tokens              |
       | Credit cost: 0 micros (covered by tokens)   |
       | Ledger entry:                               |
       |   amount_token: -12                         |
       |   amount_credit: 0                          |
       +--------------------------------------------+
   ```

8. **Lines 306-307** — Change:
   ```
            | VN calls,
            | TTS
   ```
   To:
   ```
            | VN calls,
            | TTS, Recording
   ```

9. **Lines 312-313** — Change:
   ```
       | - 3 tokens call   |
       | - 6 tokens TTS    |
   ```
   To:
   ```
       | - 3 tokens call   |
       | - 6 tokens TTS    |
       | - 12 tokens rec   |
   ```

10. **Line 360** — Change:
    `| - 2 TTS sessions (avg 5 min) = 30 tokens   |`
    To:
    `| - 2 TTS sessions (avg 5 min) = 30 tokens   |`
    Add after it:
    `| - 1 Recording (10 min) = 30 tokens          |`
    Then update `balance_token: 40` to `balance_token: 10` accordingly.

11. **Line 432** — Change:
    `- Choose plan tier based on expected VN call and TTS volume`
    To:
    `- Choose plan tier based on expected VN call, TTS, and recording volume`

12. **Lines 464-465** — Change:
    `| Unexpected overflow       | Check ``balance_token`` on account; VN calls    |`
    `|                           | and TTS consume tokens first                   |`
    To:
    `| Unexpected overflow       | Check ``balance_token`` on account; VN calls,   |`
    `|                           | TTS, and recording consume tokens first         |`

**Step 2: Rebuild HTML**

Run:
```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-recording-billing/bin-api-manager/docsdev && \
rm -rf build && \
python3 -m sphinx -M html source build
```
Expected: Build succeeds.

**Step 3: Commit**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-recording-billing
git add bin-api-manager/docsdev/source/billing_account_overview.rst
git add -f bin-api-manager/docsdev/build/
git commit -m "NOJIRA-add-recording-billing

- bin-api-manager: Add recording to billing rate tables, diagrams, and examples in RST docs
- bin-api-manager: Rebuild HTML docs"
```

---

### Task 10: Final full verification and PR preparation

**Step 1: Run full verification for all changed services**

Run each in sequence:

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-recording-billing/bin-billing-manager && \
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-recording-billing/bin-call-manager && \
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-recording-billing/bin-openapi-manager && \
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-recording-billing/bin-api-manager && \
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

Expected: All pass.

**Step 2: Check for conflicts with main**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-recording-billing
git fetch origin main
git merge-tree $(git merge-base HEAD origin/main) HEAD origin/main | grep -E "^(CONFLICT|changed in both)"
git log --oneline HEAD..origin/main
```
Expected: No conflicts.

**Step 3: Push and create PR**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-recording-billing
git push -u origin NOJIRA-add-recording-billing
```

Create PR with title `NOJIRA-add-recording-billing` and appropriate body per CLAUDE.md conventions.
