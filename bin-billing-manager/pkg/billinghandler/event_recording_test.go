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

	tmCreate := time.Date(2026, 2, 23, 10, 0, 0, 0, time.UTC)

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
					ID:         uuid.FromStringOrNil("ee000001-0000-0000-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("ee000002-0000-0000-0000-000000000001"),
				},
				TMCreate: &tmCreate,
			},

			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("ee000003-0000-0000-0000-000000000001"),
				},
			},
			responseUUID: uuid.FromStringOrNil("ee000004-0000-0000-0000-000000000001"),
			responseBilling: &billing.Billing{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("ee000004-0000-0000-0000-000000000001"),
				},
				AccountID:         uuid.FromStringOrNil("ee000003-0000-0000-0000-000000000001"),
				TransactionType:   billing.TransactionTypeUsage,
				Status:            billing.StatusProgressing,
				ReferenceType:     billing.ReferenceTypeRecording,
				ReferenceID:       uuid.FromStringOrNil("ee000001-0000-0000-0000-000000000001"),
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

	tmBillingStart := time.Date(2026, 2, 23, 10, 0, 0, 0, time.UTC)
	tmUpdate := time.Date(2026, 2, 23, 10, 1, 0, 0, time.UTC) // 60 seconds later

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
					ID: uuid.FromStringOrNil("ff000001-0000-0000-0000-000000000001"),
				},
				TMUpdate: &tmUpdate,
			},

			responseBilling: &billing.Billing{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("ff000002-0000-0000-0000-000000000001"),
				},
				AccountID:         uuid.FromStringOrNil("ff000003-0000-0000-0000-000000000001"),
				ReferenceType:     billing.ReferenceTypeRecording,
				ReferenceID:       uuid.FromStringOrNil("ff000001-0000-0000-0000-000000000001"),
				CostType:          billing.CostTypeRecording,
				RateTokenPerUnit:  billing.DefaultTokenPerUnitRecording,
				RateCreditPerUnit: billing.DefaultCreditPerUnitRecording,
				TMBillingStart:    &tmBillingStart,
			},
			responseConsumedBilling: &billing.Billing{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("ff000002-0000-0000-0000-000000000001"),
				},
				AccountID:         uuid.FromStringOrNil("ff000003-0000-0000-0000-000000000001"),
				ReferenceType:     billing.ReferenceTypeRecording,
				ReferenceID:       uuid.FromStringOrNil("ff000001-0000-0000-0000-000000000001"),
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

	r := &cmrecording.Recording{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("ff100001-0000-0000-0000-000000000001"),
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
			ID: uuid.FromStringOrNil("ff200001-0000-0000-0000-000000000001"),
		},
		TMUpdate: nil,
	}

	mockDB.EXPECT().BillingGetByReferenceID(ctx, r.ID).Return(&billing.Billing{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("ff200002-0000-0000-0000-000000000001"),
		},
		ReferenceType: billing.ReferenceTypeRecording,
	}, nil)

	// Should return error
	err := h.EventCMRecordingFinished(ctx, r)
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
}
