package billinghandler

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	cmcall "monorepo/bin-call-manager/models/call"
	cmrecording "monorepo/bin-call-manager/models/recording"
	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	tmspeaking "monorepo/bin-tts-manager/models/speaking"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"
	gomock "go.uber.org/mock/gomock"

	cerrors "monorepo/bin-common-handler/models/errors"

	"monorepo/bin-billing-manager/models/billing"
	"monorepo/bin-billing-manager/pkg/accounthandler"
	"monorepo/bin-billing-manager/pkg/dbhandler"
)

var errInjected = errors.New("injected failure")

func newTestHandler(t *testing.T) (*billingHandler, *dbhandler.MockDBHandler, *notifyhandler.MockNotifyHandler, *accounthandler.MockAccountHandler) {
	t.Helper()

	mc := gomock.NewController(t)
	t.Cleanup(mc.Finish)

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockAccount := accounthandler.NewMockAccountHandler(mc)

	h := &billingHandler{
		utilHandler:    utilhandler.NewMockUtilHandler(mc),
		db:             mockDB,
		notifyHandler:  mockNotify,
		accountHandler: mockAccount,
	}
	return h, mockDB, mockNotify, mockAccount
}

// durationSamples returns the number and the sum of the observations recorded for the reference type.
func durationSamples(t *testing.T, referenceType billing.ReferenceType) (uint64, float64) {
	t.Helper()

	reg := prometheus.NewPedanticRegistry()
	if err := reg.Register(promBillingDurationSeconds); err != nil {
		t.Fatalf("could not register the collector: %v", err)
	}
	families, err := reg.Gather()
	if err != nil {
		t.Fatalf("could not gather: %v", err)
	}

	for _, f := range families {
		for _, m := range f.GetMetric() {
			for _, l := range m.GetLabel() {
				if l.GetValue() == string(referenceType) {
					return m.GetHistogram().GetSampleCount(), m.GetHistogram().GetSampleSum()
				}
			}
		}
	}
	return 0, 0
}

// Test_event_start_failures verifies that a failing BillingStart is reported by every start event
// with the wrapper message and the cause.
func Test_event_start_failures(t *testing.T) {
	customerID := uuid.FromStringOrNil("d1000000-0000-0000-0000-000000000001")
	refID := uuid.FromStringOrNil("d1000000-0000-0000-0000-000000000002")
	start := time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)

	tests := []struct {
		name          string
		referenceType billing.ReferenceType
		call          func(h *billingHandler) error
	}{
		{
			name:          "call progressing",
			referenceType: billing.ReferenceTypeCall,
			call: func(h *billingHandler) error {
				return h.EventCMCallProgressing(context.Background(), &cmcall.Call{
					Identity:      commonidentity.Identity{ID: refID, CustomerID: customerID},
					Direction:     cmcall.DirectionOutgoing,
					Destination:   commonaddress.Address{Type: commonaddress.TypeTel},
					TMProgressing: &start,
				})
			},
		},
		{
			name:          "recording started",
			referenceType: billing.ReferenceTypeRecording,
			call: func(h *billingHandler) error {
				return h.EventCMRecordingStarted(context.Background(), &cmrecording.Recording{
					Identity: commonidentity.Identity{ID: refID, CustomerID: customerID},
					TMStart:  &start,
				})
			},
		},
		{
			name:          "tts speaking started",
			referenceType: billing.ReferenceTypeSpeaking,
			call: func(h *billingHandler) error {
				return h.EventTTSSpeakingStarted(context.Background(), &tmspeaking.Speaking{
					Identity: commonidentity.Identity{ID: refID, CustomerID: customerID},
					TMCreate: &start,
				})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mockDB, _, _ := newTestHandler(t)

			// a real database failure of the idempotency lookup stops the start before any account
			// lookup or creation (no further expectation is registered).
			mockDB.EXPECT().BillingGetByReferenceTypeAndID(gomock.Any(), tt.referenceType, refID).Return(nil, errInjected)

			err := tt.call(h)
			if !errors.Is(err, errInjected) {
				t.Fatalf("expected the injected cause, got: %v", err)
			}
			if !strings.Contains(err.Error(), "could not start a billing") {
				t.Errorf("expected the start wrapper message, got: %v", err)
			}
		})
	}
}

// Test_event_end_failures verifies that a failing BillingEnd is reported by every end event. The
// reference lookup succeeds with a billable call type, the end time is set, and the consume step
// fails.
func Test_event_end_failures(t *testing.T) {
	refID := uuid.FromStringOrNil("d2000000-0000-0000-0000-000000000001")
	start := time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)
	end := start.Add(30 * time.Second)

	tests := []struct {
		name string
		call func(h *billingHandler) error
	}{
		{
			name: "call hangup",
			call: func(h *billingHandler) error {
				return h.EventCMCallHangup(context.Background(), &cmcall.Call{
					Identity: commonidentity.Identity{ID: refID},
					TMHangup: &end,
				})
			},
		},
		{
			name: "recording finished",
			call: func(h *billingHandler) error {
				return h.EventCMRecordingFinished(context.Background(), &cmrecording.Recording{
					Identity: commonidentity.Identity{ID: refID},
					TMEnd:    &end,
				})
			},
		},
		{
			name: "tts speaking stopped",
			call: func(h *billingHandler) error {
				return h.EventTTSSpeakingStopped(context.Background(), &tmspeaking.Speaking{
					Identity: commonidentity.Identity{ID: refID},
					TMUpdate: &end,
				})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mockDB, _, _ := newTestHandler(t)

			bill := &billing.Billing{
				Identity:       commonidentity.Identity{ID: uuid.FromStringOrNil("d2000000-0000-0000-0000-000000000002")},
				AccountID:      uuid.FromStringOrNil("d2000000-0000-0000-0000-000000000003"),
				ReferenceType:  billing.ReferenceTypeCall,
				CostType:       billing.CostTypeCallPSTNOutgoing,
				TMBillingStart: &start,
			}
			mockDB.EXPECT().BillingGetByReferenceID(gomock.Any(), refID).Return(bill, nil)
			mockDB.EXPECT().BillingConsumeAndRecord(gomock.Any(), bill, bill.AccountID, billing.CalculateBillableUnits(30), 30, billing.GetCostInfo(bill.CostType), &end).Return(nil, errInjected)

			err := tt.call(h)
			if !errors.Is(err, errInjected) {
				t.Fatalf("expected the injected cause, got: %v", err)
			}
			if !strings.Contains(err.Error(), "could not end the billing") {
				t.Errorf("expected the end wrapper message, got: %v", err)
			}
		})
	}
}

// Test_BillingStart_retry_and_end_failures covers the failure of the end phase when an unfinished
// immediate-end billing is retried.
func Test_BillingStart_retry_and_end_failures(t *testing.T) {
	customerID := uuid.FromStringOrNil("d3000000-0000-0000-0000-000000000001")
	refID := uuid.FromStringOrNil("d3000000-0000-0000-0000-000000000002")
	accountID := uuid.FromStringOrNil("d3000000-0000-0000-0000-000000000003")
	billID := uuid.FromStringOrNil("d3000000-0000-0000-0000-000000000004")
	start := time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)
	costInfo := billing.GetCostInfo(billing.CostTypeSMS)

	t.Run("retry of an unfinished billing fails", func(t *testing.T) {
		h, mockDB, _, _ := newTestHandler(t)

		existing := &billing.Billing{
			Identity:      commonidentity.Identity{ID: billID},
			AccountID:     accountID,
			Status:        billing.StatusProgressing,
			ReferenceType: billing.ReferenceTypeSMS,
			CostType:      billing.CostTypeSMS,
		}
		mockDB.EXPECT().BillingGetByReferenceTypeAndID(gomock.Any(), billing.ReferenceTypeSMS, refID).Return(existing, nil)
		mockDB.EXPECT().BillingConsumeAndRecord(gomock.Any(), existing, accountID, 1, 0, costInfo, &start).Return(nil, errInjected)

		err := h.BillingStart(context.Background(), customerID, billing.ReferenceTypeSMS, refID, billing.CostTypeSMS, &start, &commonaddress.Address{}, &commonaddress.Address{})
		if !errors.Is(err, errInjected) || !strings.Contains(err.Error(), "could not complete billing on retry") {
			t.Errorf("expected the retry failure, got: %v", err)
		}
	})
}

// Test_BillingEnd_free_call covers the free call end: the failing status write, the failing
// re-read, and the success path including the recorded duration sample and published payload.
func Test_BillingEnd_free_call(t *testing.T) {
	billID := uuid.FromStringOrNil("d4000000-0000-0000-0000-000000000001")
	start := time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)
	end := start.Add(42 * time.Second)

	bill := &billing.Billing{
		Identity:       commonidentity.Identity{ID: billID},
		ReferenceType:  billing.ReferenceTypeCallExtension,
		CostType:       billing.CostTypeCallExtension,
		TMBillingStart: &start,
	}

	t.Run("status write fails", func(t *testing.T) {
		h, mockDB, _, _ := newTestHandler(t)

		mockDB.EXPECT().BillingSetStatusEnd(gomock.Any(), billID, 0, 0, int64(0), int64(0), int64(0), int64(0), &end).Return(errInjected)

		err := h.BillingEnd(context.Background(), bill, &end, &commonaddress.Address{}, &commonaddress.Address{})
		if !errors.Is(err, errInjected) || !strings.Contains(err.Error(), "could not end free billing") {
			t.Errorf("expected the status write failure, got: %v", err)
		}
	})

	t.Run("re-read fails and is returned as is", func(t *testing.T) {
		h, mockDB, _, _ := newTestHandler(t)

		mockDB.EXPECT().BillingSetStatusEnd(gomock.Any(), billID, 0, 0, int64(0), int64(0), int64(0), int64(0), &end).Return(nil)
		mockDB.EXPECT().BillingGet(gomock.Any(), billID).Return(nil, errInjected)

		err := h.BillingEnd(context.Background(), bill, &end, &commonaddress.Address{}, &commonaddress.Address{})
		if err != errInjected {
			t.Errorf("expected the unwrapped injected error, got: %v", err)
		}
	})

	t.Run("success records the duration and publishes the update", func(t *testing.T) {
		h, mockDB, mockNotify, _ := newTestHandler(t)

		res := &billing.Billing{
			Identity:       commonidentity.Identity{ID: billID},
			ReferenceType:  billing.ReferenceTypeCallExtension,
			CostType:       billing.CostTypeCallExtension,
			Status:         billing.StatusEnd,
			TMBillingStart: &start,
			TMBillingEnd:   &end,
		}
		mockDB.EXPECT().BillingSetStatusEnd(gomock.Any(), billID, 0, 0, int64(0), int64(0), int64(0), int64(0), &end).Return(nil)
		mockDB.EXPECT().BillingGet(gomock.Any(), billID).Return(res, nil)
		mockNotify.EXPECT().PublishEvent(gomock.Any(), billing.EventTypeBillingUpdated, res)

		beforeCount, beforeSum := durationSamples(t, billing.ReferenceTypeCallExtension)
		if err := h.BillingEnd(context.Background(), bill, &end, &commonaddress.Address{}, &commonaddress.Address{}); err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		afterCount, afterSum := durationSamples(t, billing.ReferenceTypeCallExtension)
		if afterCount != beforeCount+1 {
			t.Errorf("expected one more duration sample, before: %d, after: %d", beforeCount, afterCount)
		}
		if afterSum-beforeSum != 42 {
			t.Errorf("expected the observed duration to be 42 seconds, got: %v", afterSum-beforeSum)
		}
	})
}

// Test_Get_not_found_mapping verifies the typed not found error and the plain wrap of other errors.
func Test_Get_not_found_mapping(t *testing.T) {
	id := uuid.FromStringOrNil("d5000000-0000-0000-0000-000000000001")

	t.Run("not found is typed", func(t *testing.T) {
		h, mockDB, _, _ := newTestHandler(t)
		mockDB.EXPECT().BillingGet(gomock.Any(), id).Return(nil, dbhandler.ErrNotFound)

		res, err := h.Get(context.Background(), id)
		if res != nil {
			t.Errorf("expected no billing, got: %v", res)
		}
		var ve *cerrors.VoipbinError
		if !errors.As(err, &ve) || ve.Status != cerrors.StatusNotFound || ve.Reason != "BILLING_NOT_FOUND" {
			t.Fatalf("expected a typed not found error, got: %v", err)
		}
		if !errors.Is(err, dbhandler.ErrNotFound) {
			t.Errorf("expected the cause to stay reachable, got: %v", err)
		}
	})

	t.Run("other errors are wrapped", func(t *testing.T) {
		h, mockDB, _, _ := newTestHandler(t)
		mockDB.EXPECT().BillingGet(gomock.Any(), id).Return(nil, errInjected)

		_, err := h.Get(context.Background(), id)
		var ve *cerrors.VoipbinError
		if errors.As(err, &ve) {
			t.Errorf("did not expect a typed error, got: %v", err)
		}
		if !errors.Is(err, errInjected) || !strings.Contains(err.Error(), "could not get billing") {
			t.Errorf("expected the wrapped cause, got: %v", err)
		}
	})
}
