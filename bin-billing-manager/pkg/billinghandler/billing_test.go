package billinghandler

import (
	"context"
	"fmt"
	"testing"
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-billing-manager/models/account"
	"monorepo/bin-billing-manager/models/billing"
	"monorepo/bin-billing-manager/pkg/accounthandler"
	"monorepo/bin-billing-manager/pkg/dbhandler"
)

func Test_BillingStart(t *testing.T) {

	type test struct {
		name string

		customerID     uuid.UUID
		referenceType  billing.ReferenceType
		referenceID    uuid.UUID
		tmBillingStart *time.Time
		source         *commonaddress.Address
		destination    *commonaddress.Address

		responseAccount *account.Account
		responseUUID    uuid.UUID
		responseBilling *billing.Billing

		expectBilling *billing.Billing
	}

	tmBillingStart := time.Date(2023, 6, 8, 3, 22, 17, 995000000, time.UTC)

	tests := []test{
		{
			name: "normal",

			customerID:     uuid.FromStringOrNil("a1f18e42-09f7-11ee-8485-5b8f354924bd"),
			referenceType:  billing.ReferenceTypeCall,
			referenceID:    uuid.FromStringOrNil("a21b5088-09f7-11ee-b0f7-2ba7f59b91a8"),
			tmBillingStart: &tmBillingStart,
			source: &commonaddress.Address{
				Target: "+821100000001",
			},
			destination: &commonaddress.Address{
				Target: "+821100000002",
			},

			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a2519e7c-09f7-11ee-a244-cb84b8c33ef0"),
					CustomerID: uuid.FromStringOrNil("a1f18e42-09f7-11ee-8485-5b8f354924bd"),
				},
			},
			responseUUID: uuid.FromStringOrNil("a27635f2-09f7-11ee-8003-8f7d80359ed0"),
			responseBilling: &billing.Billing{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a27635f2-09f7-11ee-8003-8f7d80359ed0"),
				},
			},

			expectBilling: &billing.Billing{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a27635f2-09f7-11ee-8003-8f7d80359ed0"),
					CustomerID: uuid.FromStringOrNil("a1f18e42-09f7-11ee-8485-5b8f354924bd"),
				},
				AccountID:        uuid.FromStringOrNil("a2519e7c-09f7-11ee-a244-cb84b8c33ef0"),
				Status:           billing.StatusProgressing,
				ReferenceType:    billing.ReferenceTypeCall,
				ReferenceID:      uuid.FromStringOrNil("a21b5088-09f7-11ee-b0f7-2ba7f59b91a8"),
				CostPerUnit:      billing.DefaultCostPerUnitReferenceTypeCall,
				CostTotal:        0,
				BillingUnitCount: 0,
				TMBillingStart:   &tmBillingStart,
				TMBillingEnd:     nil,
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

			// idempotency check - no existing billing
			mockDB.EXPECT().BillingGetByReferenceTypeAndID(ctx, tt.referenceType, tt.referenceID).Return(nil, fmt.Errorf("not found"))

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

func Test_BillingStart_number_sms(t *testing.T) {

	type test struct {
		name string

		customerID     uuid.UUID
		referenceType  billing.ReferenceType
		referenceID    uuid.UUID
		tmBillingStart *time.Time
		source         *commonaddress.Address
		destination    *commonaddress.Address

		responseAccount *account.Account
		responseUUID    uuid.UUID
		responseBilling *billing.Billing

		expectBilling *billing.Billing
	}

	tmBillingStart := time.Date(2023, 6, 8, 3, 22, 17, 995000000, time.UTC)

	tests := []test{
		{
			name: "type sms",

			customerID:     uuid.FromStringOrNil("c29c8386-16a9-11ee-ae77-c336432e00f9"),
			referenceType:  billing.ReferenceTypeSMS,
			referenceID:    uuid.FromStringOrNil("c3183f6c-16a9-11ee-b2b9-677692eb71ef"),
			tmBillingStart: &tmBillingStart,
			source: &commonaddress.Address{
				Target: "+821100000001",
			},
			destination: &commonaddress.Address{
				Target: "+821100000002",
			},

			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c3462102-16a9-11ee-97c1-077bf7697388"),
					CustomerID: uuid.FromStringOrNil("c29c8386-16a9-11ee-ae77-c336432e00f9"),
				},
			},
			responseUUID: uuid.FromStringOrNil("c378602c-16a9-11ee-a774-0f294c1a0b21"),
			responseBilling: &billing.Billing{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c378602c-16a9-11ee-a774-0f294c1a0b21"),
					CustomerID: uuid.FromStringOrNil("c29c8386-16a9-11ee-ae77-c336432e00f9"),
				},
				AccountID:     uuid.FromStringOrNil("c3462102-16a9-11ee-97c1-077bf7697388"),
				Status:        billing.StatusProgressing,
				ReferenceType: billing.ReferenceTypeSMS,
				ReferenceID:   uuid.FromStringOrNil("c3183f6c-16a9-11ee-b2b9-677692eb71ef"),
				CostPerUnit:   billing.DefaultCostPerUnitReferenceTypeSMS,
			},

			expectBilling: &billing.Billing{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c378602c-16a9-11ee-a774-0f294c1a0b21"),
					CustomerID: uuid.FromStringOrNil("c29c8386-16a9-11ee-ae77-c336432e00f9"),
				},
				AccountID:        uuid.FromStringOrNil("c3462102-16a9-11ee-97c1-077bf7697388"),
				Status:           billing.StatusProgressing,
				ReferenceType:    billing.ReferenceTypeSMS,
				ReferenceID:      uuid.FromStringOrNil("c3183f6c-16a9-11ee-b2b9-677692eb71ef"),
				CostPerUnit:      billing.DefaultCostPerUnitReferenceTypeSMS,
				CostTotal:        0,
				BillingUnitCount: 0,
				TMBillingStart:   &tmBillingStart,
				TMBillingEnd:     nil,
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

			// idempotency check - no existing billing
			mockDB.EXPECT().BillingGetByReferenceTypeAndID(ctx, tt.referenceType, tt.referenceID).Return(nil, fmt.Errorf("not found"))

			mockAccount.EXPECT().GetByCustomerID(ctx, tt.customerID).Return(tt.responseAccount, nil)
			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockDB.EXPECT().BillingCreate(ctx, tt.expectBilling).Return(nil)
			mockDB.EXPECT().BillingGet(ctx, tt.responseUUID).Return(tt.responseBilling, nil)
			mockNotify.EXPECT().PublishEvent(ctx, billing.EventTypeBillingCreated, tt.responseBilling)

			// billing end
			mockDB.EXPECT().BillingSetStatusEnd(ctx, tt.responseBilling.ID, float32(1), gomock.Any()).Return(nil)
			mockDB.EXPECT().BillingGet(ctx, tt.responseBilling.ID).Return(tt.responseBilling, nil)

			mockAccount.EXPECT().SubtractBalanceWithCheck(ctx, tt.responseBilling.AccountID, tt.responseBilling.CostTotal).Return(tt.responseAccount, nil)

			if err := h.BillingStart(ctx, tt.customerID, tt.referenceType, tt.referenceID, tt.tmBillingStart, tt.source, tt.destination); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_BillingEnd(t *testing.T) {

	type test struct {
		name string

		billing      *billing.Billing
		tmBillingEnd *time.Time
		source       *commonaddress.Address
		destination  *commonaddress.Address

		responseBilling *billing.Billing
		responseAccount *account.Account

		expectBillingUnitCount float32
	}

	tmBillingStart := time.Date(2023, 6, 8, 3, 22, 7, 991000000, time.UTC)
	tmBillingEnd := time.Date(2023, 6, 8, 3, 22, 17, 995000000, time.UTC)

	tests := []test{
		{
			name: "reference type call",

			billing: &billing.Billing{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("2349c024-16ad-11ee-aeac-f305605c1bff"),
					CustomerID: uuid.FromStringOrNil("2381e58a-16ad-11ee-94c5-4b595e64a13d"),
				},
				AccountID:      uuid.FromStringOrNil("23d43574-16ad-11ee-9c99-3b8e376bb5a3"),
				ReferenceType:  billing.ReferenceTypeCall,
				TMBillingStart: &tmBillingStart,
			},
			tmBillingEnd: &tmBillingEnd,
			source: &commonaddress.Address{
				Target: "+821100000001",
			},
			destination: &commonaddress.Address{
				Target: "+821100000002",
			},

			responseBilling: &billing.Billing{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("2349c024-16ad-11ee-aeac-f305605c1bff"),
					CustomerID: uuid.FromStringOrNil("2381e58a-16ad-11ee-94c5-4b595e64a13d"),
				},
				AccountID:      uuid.FromStringOrNil("23d43574-16ad-11ee-9c99-3b8e376bb5a3"),
				ReferenceType:  billing.ReferenceTypeCall,
				TMBillingStart: &tmBillingStart,
			},
			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("23d43574-16ad-11ee-9c99-3b8e376bb5a3"),
				},
			},

			expectBillingUnitCount: 10.004,
		},
		{
			name: "reference type sms",

			billing: &billing.Billing{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("9dc58620-16ae-11ee-8e68-639b889e0538"),
					CustomerID: uuid.FromStringOrNil("9de95a50-16ae-11ee-a50f-dfdbce035244"),
				},
				AccountID:      uuid.FromStringOrNil("9e128c36-16ae-11ee-9655-2f9b21f8f7ba"),
				ReferenceType:  billing.ReferenceTypeSMS,
				TMBillingStart: &tmBillingStart,
			},
			tmBillingEnd: &tmBillingEnd,
			source:       &commonaddress.Address{},
			destination:  &commonaddress.Address{},

			responseBilling: &billing.Billing{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("9dc58620-16ae-11ee-8e68-639b889e0538"),
					CustomerID: uuid.FromStringOrNil("9de95a50-16ae-11ee-a50f-dfdbce035244"),
				},
				AccountID:      uuid.FromStringOrNil("9e128c36-16ae-11ee-9655-2f9b21f8f7ba"),
				ReferenceType:  billing.ReferenceTypeCall,
				TMBillingStart: &tmBillingStart,
			},
			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("9e128c36-16ae-11ee-9655-2f9b21f8f7ba"),
				},
			},

			expectBillingUnitCount: 1,
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

			mockDB.EXPECT().BillingSetStatusEnd(ctx, tt.responseBilling.ID, tt.expectBillingUnitCount, tt.tmBillingEnd).Return(nil)
			mockDB.EXPECT().BillingGet(ctx, tt.responseBilling.ID).Return(tt.responseBilling, nil)

			mockAccount.EXPECT().SubtractBalanceWithCheck(ctx, tt.responseBilling.AccountID, tt.responseBilling.CostTotal).Return(tt.responseAccount, nil)

			if err := h.BillingEnd(ctx, tt.billing, tt.tmBillingEnd, tt.source, tt.destination); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_BillingStart_idempotent(t *testing.T) {

	tmBillingStart := time.Date(2023, 6, 8, 3, 22, 17, 995000000, time.UTC)

	tests := []struct {
		name string

		customerID     uuid.UUID
		referenceType  billing.ReferenceType
		referenceID    uuid.UUID
		tmBillingStart *time.Time
		source         *commonaddress.Address
		destination    *commonaddress.Address

		existingBilling *billing.Billing
	}{
		{
			name: "existing billing found - should return early",

			customerID:     uuid.FromStringOrNil("d1111111-0000-0000-0000-000000000001"),
			referenceType:  billing.ReferenceTypeCall,
			referenceID:    uuid.FromStringOrNil("d2222222-0000-0000-0000-000000000001"),
			tmBillingStart: &tmBillingStart,
			source:         &commonaddress.Address{Target: "+1234"},
			destination:    &commonaddress.Address{Target: "+5678"},

			existingBilling: &billing.Billing{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("d3333333-0000-0000-0000-000000000001"),
				},
				ReferenceType: billing.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("d2222222-0000-0000-0000-000000000001"),
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

			// idempotency check - existing billing found
			mockDB.EXPECT().BillingGetByReferenceTypeAndID(ctx, tt.referenceType, tt.referenceID).Return(tt.existingBilling, nil)

			// NO further calls expected - should return early

			if err := h.BillingStart(ctx, tt.customerID, tt.referenceType, tt.referenceID, tt.tmBillingStart, tt.source, tt.destination); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_BillingEnd_subtract_error(t *testing.T) {

	tmBillingStart := time.Date(2023, 6, 8, 3, 22, 7, 991000000, time.UTC)
	tmBillingEnd := time.Date(2023, 6, 8, 3, 22, 17, 995000000, time.UTC)

	tests := []struct {
		name string

		billing      *billing.Billing
		tmBillingEnd *time.Time
		source       *commonaddress.Address
		destination  *commonaddress.Address

		responseBilling *billing.Billing
	}{
		{
			name: "subtract error propagates",

			billing: &billing.Billing{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("e1111111-0000-0000-0000-000000000001"),
				},
				AccountID:      uuid.FromStringOrNil("e2222222-0000-0000-0000-000000000001"),
				ReferenceType:  billing.ReferenceTypeCall,
				TMBillingStart: &tmBillingStart,
			},
			tmBillingEnd: &tmBillingEnd,
			source:       &commonaddress.Address{},
			destination:  &commonaddress.Address{},

			responseBilling: &billing.Billing{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("e1111111-0000-0000-0000-000000000001"),
				},
				AccountID:      uuid.FromStringOrNil("e2222222-0000-0000-0000-000000000001"),
				ReferenceType:  billing.ReferenceTypeCall,
				TMBillingStart: &tmBillingStart,
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

			mockDB.EXPECT().BillingSetStatusEnd(ctx, tt.responseBilling.ID, gomock.Any(), tt.tmBillingEnd).Return(nil)
			mockDB.EXPECT().BillingGet(ctx, tt.responseBilling.ID).Return(tt.responseBilling, nil)
			mockAccount.EXPECT().SubtractBalanceWithCheck(ctx, tt.responseBilling.AccountID, tt.responseBilling.CostTotal).Return(nil, fmt.Errorf("insufficient balance"))

			err := h.BillingEnd(ctx, tt.billing, tt.tmBillingEnd, tt.source, tt.destination)
			if err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			}
		})
	}
}
