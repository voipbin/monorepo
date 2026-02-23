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

		responseBilling         *billing.Billing
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
		ReferenceType: billing.ReferenceTypeSpeaking,
	}, nil)

	// Should return error
	err := h.EventTTSSpeakingStopped(ctx, s)
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
}
