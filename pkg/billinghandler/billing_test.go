package billinghandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"

	"gitlab.com/voipbin/bin-manager/billing-manager.git/models/account"
	"gitlab.com/voipbin/bin-manager/billing-manager.git/models/billing"
	"gitlab.com/voipbin/bin-manager/billing-manager.git/pkg/accounthandler"
	"gitlab.com/voipbin/bin-manager/billing-manager.git/pkg/dbhandler"
)

func Test_BillingStart(t *testing.T) {

	type test struct {
		name string

		customerID     uuid.UUID
		referenceType  billing.ReferenceType
		referenceID    uuid.UUID
		tmBillingStart string
		source         *commonaddress.Address
		destination    *commonaddress.Address

		responseAccount *account.Account
		responseUUID    uuid.UUID
		responseBilling *billing.Billing

		expectBilling *billing.Billing
	}

	tests := []test{
		{
			name: "normal",

			customerID:     uuid.FromStringOrNil("a1f18e42-09f7-11ee-8485-5b8f354924bd"),
			referenceType:  billing.ReferenceTypeCall,
			referenceID:    uuid.FromStringOrNil("a21b5088-09f7-11ee-b0f7-2ba7f59b91a8"),
			tmBillingStart: "2023-06-08 03:22:17.995000",
			source: &commonaddress.Address{
				Target: "+821100000001",
			},
			destination: &commonaddress.Address{
				Target: "+821100000002",
			},

			responseAccount: &account.Account{
				ID:         uuid.FromStringOrNil("a2519e7c-09f7-11ee-a244-cb84b8c33ef0"),
				CustomerID: uuid.FromStringOrNil("a1f18e42-09f7-11ee-8485-5b8f354924bd"),
			},
			responseUUID: uuid.FromStringOrNil("a27635f2-09f7-11ee-8003-8f7d80359ed0"),
			responseBilling: &billing.Billing{
				ID: uuid.FromStringOrNil("a27635f2-09f7-11ee-8003-8f7d80359ed0"),
			},

			expectBilling: &billing.Billing{
				ID:               uuid.FromStringOrNil("a27635f2-09f7-11ee-8003-8f7d80359ed0"),
				CustomerID:       uuid.FromStringOrNil("a1f18e42-09f7-11ee-8485-5b8f354924bd"),
				AccountID:        uuid.FromStringOrNil("a2519e7c-09f7-11ee-a244-cb84b8c33ef0"),
				Status:           billing.StatusProgressing,
				ReferenceType:    billing.ReferenceTypeCall,
				ReferenceID:      uuid.FromStringOrNil("a21b5088-09f7-11ee-b0f7-2ba7f59b91a8"),
				CostPerUnit:      defaultCostPerUnitReferenceTypeCall,
				CostTotal:        0,
				BillingUnitCount: 0,
				TMBillingStart:   "2023-06-08 03:22:17.995000",
				TMBillingEnd:     dbhandler.DefaultTimeStamp,
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

			mockAccount.EXPECT().GetByCustomerID(ctx, tt.customerID).Return(tt.responseAccount, nil)
			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockDB.EXPECT().BillingCreate(ctx, tt.expectBilling).Return(nil)
			mockDB.EXPECT().BillingGet(ctx, tt.responseUUID).Return(tt.responseBilling, nil)
			mockNotify.EXPECT().PublishEvent(ctx, billing.EventTypeBillingCreated, tt.responseBilling)

			if err := h.BillingStart(ctx, tt.customerID, tt.referenceType, tt.referenceID, tt.tmBillingStart, tt.source, tt.destination); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_BillingEnd(t *testing.T) {

	type test struct {
		name string

		customerID    uuid.UUID
		referenceType billing.ReferenceType
		referenceID   uuid.UUID
		tmBillingEnd  string
		source        *commonaddress.Address
		destination   *commonaddress.Address

		responseBilling *billing.Billing
		responseAccount *account.Account

		expectBillingUnitCount float32
	}

	tests := []test{
		{
			name: "normal",

			customerID:    uuid.FromStringOrNil("00f7f28a-09fa-11ee-8eb6-0bf81ac3875f"),
			referenceType: billing.ReferenceTypeCall,
			referenceID:   uuid.FromStringOrNil("011dc56e-09fa-11ee-8a6c-1735bc47d3ef"),
			tmBillingEnd:  "2023-06-08 03:22:17.995000",
			source: &commonaddress.Address{
				Target: "+821100000001",
			},
			destination: &commonaddress.Address{
				Target: "+821100000002",
			},

			responseBilling: &billing.Billing{
				ID:             uuid.FromStringOrNil("a27635f2-09f7-11ee-8003-8f7d80359ed0"),
				TMBillingStart: "2023-06-08 03:22:07.991000",
			},
			responseAccount: &account.Account{
				ID: uuid.FromStringOrNil("01592262-09fa-11ee-bf7d-b34a889017cc"),
			},

			expectBillingUnitCount: 10.004,
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

			mockDB.EXPECT().BillingGetByReferenceID(ctx, tt.referenceID).Return(tt.responseBilling, nil)
			mockUtil.EXPECT().TimeParse(tt.responseBilling.TMBillingStart).Return(utilhandler.TimeParse(tt.responseBilling.TMBillingStart))
			mockUtil.EXPECT().TimeParse(tt.tmBillingEnd).Return(utilhandler.TimeParse(tt.tmBillingEnd))

			mockDB.EXPECT().BillingSetStatusEnd(ctx, tt.responseBilling.ID, tt.expectBillingUnitCount, tt.tmBillingEnd).Return(nil)
			mockDB.EXPECT().BillingGet(ctx, tt.responseBilling.ID).Return(tt.responseBilling, nil)

			mockAccount.EXPECT().SubstractBalanceByCustomer(ctx, tt.responseBilling.CustomerID, tt.responseBilling.CostTotal).Return(tt.responseAccount, nil)

			if err := h.BillingEnd(ctx, tt.customerID, tt.referenceType, tt.referenceID, tt.tmBillingEnd, tt.source, tt.destination); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
