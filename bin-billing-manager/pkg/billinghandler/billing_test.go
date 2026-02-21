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
		costType       billing.CostType
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
			costType:       billing.CostTypeCallPSTNOutgoing,
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
				AccountID:         uuid.FromStringOrNil("a2519e7c-09f7-11ee-a244-cb84b8c33ef0"),
				TransactionType:   billing.TransactionTypeUsage,
				Status:            billing.StatusProgressing,
				ReferenceType:     billing.ReferenceTypeCall,
				ReferenceID:       uuid.FromStringOrNil("a21b5088-09f7-11ee-b0f7-2ba7f59b91a8"),
				CostType:          billing.CostTypeCallPSTNOutgoing,
				RateCreditPerUnit: billing.DefaultCreditPerUnitCallPSTNOutgoing,
				RateTokenPerUnit:  0,
				TMBillingStart:    &tmBillingStart,
				TMBillingEnd:      nil,
			},
		},
		{
			name: "call_extension",

			customerID:     uuid.FromStringOrNil("f1111111-0000-0000-0000-000000000001"),
			referenceType:  billing.ReferenceTypeCallExtension,
			referenceID:    uuid.FromStringOrNil("f2222222-0000-0000-0000-000000000001"),
			costType:       billing.CostTypeCallExtension,
			tmBillingStart: &tmBillingStart,
			source: &commonaddress.Address{
				Type:   commonaddress.TypeExtension,
				Target: "1001",
			},
			destination: &commonaddress.Address{
				Type:   commonaddress.TypeExtension,
				Target: "1002",
			},

			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("f3333333-0000-0000-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("f1111111-0000-0000-0000-000000000001"),
				},
			},
			responseUUID: uuid.FromStringOrNil("f4444444-0000-0000-0000-000000000001"),
			responseBilling: &billing.Billing{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("f4444444-0000-0000-0000-000000000001"),
				},
			},

			expectBilling: &billing.Billing{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("f4444444-0000-0000-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("f1111111-0000-0000-0000-000000000001"),
				},
				AccountID:       uuid.FromStringOrNil("f3333333-0000-0000-0000-000000000001"),
				TransactionType: billing.TransactionTypeUsage,
				Status:          billing.StatusProgressing,
				ReferenceType:   billing.ReferenceTypeCallExtension,
				ReferenceID:     uuid.FromStringOrNil("f2222222-0000-0000-0000-000000000001"),
				CostType:        billing.CostTypeCallExtension,
				TMBillingStart:  &tmBillingStart,
				TMBillingEnd:    nil,
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
			mockDB.EXPECT().BillingGetByReferenceTypeAndID(ctx, tt.referenceType, tt.referenceID).Return(nil, dbhandler.ErrNotFound)

			mockAccount.EXPECT().GetByCustomerID(ctx, tt.customerID).Return(tt.responseAccount, nil)
			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockDB.EXPECT().BillingCreate(ctx, tt.expectBilling).Return(nil)
			mockDB.EXPECT().BillingGet(ctx, tt.responseUUID).Return(tt.responseBilling, nil)
			mockNotify.EXPECT().PublishEvent(ctx, billing.EventTypeBillingCreated, tt.responseBilling)

			if err := h.BillingStart(ctx, tt.customerID, tt.referenceType, tt.referenceID, tt.costType, tt.tmBillingStart, tt.source, tt.destination); err != nil {
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
		costType       billing.CostType
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
			costType:       billing.CostTypeSMS,
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
				AccountID:         uuid.FromStringOrNil("c3462102-16a9-11ee-97c1-077bf7697388"),
				Status:            billing.StatusProgressing,
				ReferenceType:     billing.ReferenceTypeSMS,
				ReferenceID:       uuid.FromStringOrNil("c3183f6c-16a9-11ee-b2b9-677692eb71ef"),
				CostType:          billing.CostTypeSMS,
				RateTokenPerUnit:  0,
				RateCreditPerUnit: billing.DefaultCreditPerUnitSMS,
			},

			expectBilling: &billing.Billing{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c378602c-16a9-11ee-a774-0f294c1a0b21"),
					CustomerID: uuid.FromStringOrNil("c29c8386-16a9-11ee-ae77-c336432e00f9"),
				},
				AccountID:         uuid.FromStringOrNil("c3462102-16a9-11ee-97c1-077bf7697388"),
				TransactionType:   billing.TransactionTypeUsage,
				Status:            billing.StatusProgressing,
				ReferenceType:     billing.ReferenceTypeSMS,
				ReferenceID:       uuid.FromStringOrNil("c3183f6c-16a9-11ee-b2b9-677692eb71ef"),
				CostType:          billing.CostTypeSMS,
				RateCreditPerUnit: billing.DefaultCreditPerUnitSMS,
				RateTokenPerUnit:  0,
				TMBillingStart:    &tmBillingStart,
				TMBillingEnd:      nil,
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
			mockDB.EXPECT().BillingGetByReferenceTypeAndID(ctx, tt.referenceType, tt.referenceID).Return(nil, dbhandler.ErrNotFound)

			mockAccount.EXPECT().GetByCustomerID(ctx, tt.customerID).Return(tt.responseAccount, nil)
			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockDB.EXPECT().BillingCreate(ctx, tt.expectBilling).Return(nil)
			mockDB.EXPECT().BillingGet(ctx, tt.responseUUID).Return(tt.responseBilling, nil)
			mockNotify.EXPECT().PublishEvent(ctx, billing.EventTypeBillingCreated, tt.responseBilling)

			// billing end - SMS uses BillingConsumeAndRecord atomically
			mockDB.EXPECT().BillingConsumeAndRecord(ctx, tt.responseBilling, tt.responseBilling.AccountID, 1, 0, int64(0), billing.DefaultCreditPerUnitSMS, tt.tmBillingStart).Return(tt.responseBilling, nil)
			mockNotify.EXPECT().PublishEvent(ctx, billing.EventTypeBillingUpdated, tt.responseBilling)

			if err := h.BillingStart(ctx, tt.customerID, tt.referenceType, tt.referenceID, tt.costType, tt.tmBillingStart, tt.source, tt.destination); err != nil {
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
	}

	tmBillingStart := time.Date(2023, 6, 8, 3, 22, 7, 991000000, time.UTC)
	tmBillingEnd := time.Date(2023, 6, 8, 3, 22, 17, 995000000, time.UTC)

	tests := []test{
		{
			name: "reference type call - credit only (PSTN outgoing)",

			billing: &billing.Billing{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("2349c024-16ad-11ee-aeac-f305605c1bff"),
					CustomerID: uuid.FromStringOrNil("2381e58a-16ad-11ee-94c5-4b595e64a13d"),
				},
				AccountID:         uuid.FromStringOrNil("23d43574-16ad-11ee-9c99-3b8e376bb5a3"),
				ReferenceType:     billing.ReferenceTypeCall,
				CostType:          billing.CostTypeCallPSTNOutgoing,
				RateCreditPerUnit: billing.DefaultCreditPerUnitCallPSTNOutgoing,
				TMBillingStart:    &tmBillingStart,
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
				AccountID:         uuid.FromStringOrNil("23d43574-16ad-11ee-9c99-3b8e376bb5a3"),
				ReferenceType:     billing.ReferenceTypeCall,
				CostType:          billing.CostTypeCallPSTNOutgoing,
				RateCreditPerUnit: billing.DefaultCreditPerUnitCallPSTNOutgoing,
				TMBillingStart:    &tmBillingStart,
			},
		},
		{
			name: "reference type sms - token + credit overflow",

			billing: &billing.Billing{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("9dc58620-16ae-11ee-8e68-639b889e0538"),
					CustomerID: uuid.FromStringOrNil("9de95a50-16ae-11ee-a50f-dfdbce035244"),
				},
				AccountID:         uuid.FromStringOrNil("9e128c36-16ae-11ee-9655-2f9b21f8f7ba"),
				ReferenceType:     billing.ReferenceTypeSMS,
				CostType:          billing.CostTypeSMS,
				RateTokenPerUnit:  0,
				RateCreditPerUnit: billing.DefaultCreditPerUnitSMS,
				TMBillingStart:    &tmBillingStart,
			},
			tmBillingEnd: &tmBillingEnd,
			source:       &commonaddress.Address{},
			destination:  &commonaddress.Address{},

			responseBilling: &billing.Billing{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("9dc58620-16ae-11ee-8e68-639b889e0538"),
					CustomerID: uuid.FromStringOrNil("9de95a50-16ae-11ee-a50f-dfdbce035244"),
				},
				AccountID:         uuid.FromStringOrNil("9e128c36-16ae-11ee-9655-2f9b21f8f7ba"),
				ReferenceType:     billing.ReferenceTypeSMS,
				CostType:          billing.CostTypeSMS,
				RateTokenPerUnit:  0,
				RateCreditPerUnit: billing.DefaultCreditPerUnitSMS,
				TMBillingStart:    &tmBillingStart,
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

			// BillingEnd now uses BillingConsumeAndRecord atomically
			var expectBillableUnits int
			var expectUsageDuration int
			switch tt.billing.ReferenceType {
			case billing.ReferenceTypeCall:
				expectUsageDuration = int(tt.tmBillingEnd.Sub(*tt.billing.TMBillingStart).Seconds())
				expectBillableUnits = billing.CalculateBillableUnits(expectUsageDuration)
			default:
				expectUsageDuration = 0
				expectBillableUnits = 1
			}

			mockDB.EXPECT().BillingConsumeAndRecord(ctx, tt.billing, tt.billing.AccountID, expectBillableUnits, expectUsageDuration, tt.billing.RateTokenPerUnit, tt.billing.RateCreditPerUnit, tt.tmBillingEnd).Return(tt.responseBilling, nil)
			mockNotify.EXPECT().PublishEvent(ctx, billing.EventTypeBillingUpdated, tt.responseBilling)

			if err := h.BillingEnd(ctx, tt.billing, tt.tmBillingEnd, tt.source, tt.destination); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_BillingEnd_call_extension(t *testing.T) {

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
			name: "call_extension skips balance deduction",

			billing: &billing.Billing{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("fa111111-0000-0000-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("fa222222-0000-0000-0000-000000000001"),
				},
				AccountID:      uuid.FromStringOrNil("fa333333-0000-0000-0000-000000000001"),
				ReferenceType:  billing.ReferenceTypeCallExtension,
				CostType:       billing.CostTypeCallExtension,
				TMBillingStart: &tmBillingStart,
			},
			tmBillingEnd: &tmBillingEnd,
			source: &commonaddress.Address{
				Type:   commonaddress.TypeExtension,
				Target: "1001",
			},
			destination: &commonaddress.Address{
				Type:   commonaddress.TypeExtension,
				Target: "1002",
			},

			responseBilling: &billing.Billing{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("fa111111-0000-0000-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("fa222222-0000-0000-0000-000000000001"),
				},
				AccountID:      uuid.FromStringOrNil("fa333333-0000-0000-0000-000000000001"),
				ReferenceType:  billing.ReferenceTypeCallExtension,
				CostType:       billing.CostTypeCallExtension,
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

			// CostTypeCallExtension -> BillingSetStatusEnd with zero costs
			mockDB.EXPECT().BillingSetStatusEnd(ctx, tt.responseBilling.ID, 0, 0, int64(0), int64(0), int64(0), int64(0), tt.tmBillingEnd).Return(nil)
			mockDB.EXPECT().BillingGet(ctx, tt.responseBilling.ID).Return(tt.responseBilling, nil)
			mockNotify.EXPECT().PublishEvent(ctx, billing.EventTypeBillingUpdated, tt.responseBilling)

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
		costType       billing.CostType
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
			costType:       billing.CostTypeCallPSTNOutgoing,
			tmBillingStart: &tmBillingStart,
			source:         &commonaddress.Address{Target: "+1234"},
			destination:    &commonaddress.Address{Target: "+5678"},

			existingBilling: &billing.Billing{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("d3333333-0000-0000-0000-000000000001"),
				},
				Status:        billing.StatusEnd,
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

			if err := h.BillingStart(ctx, tt.customerID, tt.referenceType, tt.referenceID, tt.costType, tt.tmBillingStart, tt.source, tt.destination); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_BillingEnd_consume_error(t *testing.T) {

	tmBillingStart := time.Date(2023, 6, 8, 3, 22, 7, 991000000, time.UTC)
	tmBillingEnd := time.Date(2023, 6, 8, 3, 22, 17, 995000000, time.UTC)

	tests := []struct {
		name string

		billing      *billing.Billing
		tmBillingEnd *time.Time
		source       *commonaddress.Address
		destination  *commonaddress.Address
	}{
		{
			name: "consume and record error propagates",

			billing: &billing.Billing{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("e1111111-0000-0000-0000-000000000001"),
				},
				AccountID:         uuid.FromStringOrNil("e2222222-0000-0000-0000-000000000001"),
				ReferenceType:     billing.ReferenceTypeCall,
				CostType:          billing.CostTypeCallPSTNOutgoing,
				RateCreditPerUnit: billing.DefaultCreditPerUnitCallPSTNOutgoing,
				TMBillingStart:    &tmBillingStart,
			},
			tmBillingEnd: &tmBillingEnd,
			source:       &commonaddress.Address{},
			destination:  &commonaddress.Address{},
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

			// BillingConsumeAndRecord returns error
			usageDuration := int(tt.tmBillingEnd.Sub(*tt.billing.TMBillingStart).Seconds())
			billableUnits := billing.CalculateBillableUnits(usageDuration)
			mockDB.EXPECT().BillingConsumeAndRecord(ctx, tt.billing, tt.billing.AccountID, billableUnits, usageDuration, tt.billing.RateTokenPerUnit, tt.billing.RateCreditPerUnit, tt.tmBillingEnd).Return(nil, fmt.Errorf("insufficient balance"))

			err := h.BillingEnd(ctx, tt.billing, tt.tmBillingEnd, tt.source, tt.destination)
			if err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			}
		})
	}
}

func Test_BillingStart_db_error_on_idempotency_check(t *testing.T) {

	tmBillingStart := time.Date(2023, 6, 8, 3, 22, 17, 995000000, time.UTC)

	tests := []struct {
		name string

		customerID     uuid.UUID
		referenceType  billing.ReferenceType
		referenceID    uuid.UUID
		costType       billing.CostType
		tmBillingStart *time.Time
		source         *commonaddress.Address
		destination    *commonaddress.Address
	}{
		{
			name: "db error on idempotency check",

			customerID:     uuid.FromStringOrNil("10000001-0000-0000-0000-000000000001"),
			referenceType:  billing.ReferenceTypeCall,
			referenceID:    uuid.FromStringOrNil("10000002-0000-0000-0000-000000000001"),
			costType:       billing.CostTypeCallPSTNOutgoing,
			tmBillingStart: &tmBillingStart,
			source:         &commonaddress.Address{Target: "+1234"},
			destination:    &commonaddress.Address{Target: "+5678"},
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

			// idempotency check returns non-ErrNotFound error
			mockDB.EXPECT().BillingGetByReferenceTypeAndID(ctx, tt.referenceType, tt.referenceID).Return(nil, fmt.Errorf("connection refused"))

			// NO further calls expected — should return error immediately

			err := h.BillingStart(ctx, tt.customerID, tt.referenceType, tt.referenceID, tt.costType, tt.tmBillingStart, tt.source, tt.destination)
			if err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			}
		})
	}
}

func Test_BillingStart_idempotent_retry_sms(t *testing.T) {

	tmBillingStart := time.Date(2023, 6, 8, 3, 22, 17, 995000000, time.UTC)

	tests := []struct {
		name string

		customerID     uuid.UUID
		referenceType  billing.ReferenceType
		referenceID    uuid.UUID
		costType       billing.CostType
		tmBillingStart *time.Time
		source         *commonaddress.Address
		destination    *commonaddress.Address

		existingBilling *billing.Billing
		responseBilling *billing.Billing
	}{
		{
			name: "existing progressing SMS billing - should retry end",

			customerID:     uuid.FromStringOrNil("10000003-0000-0000-0000-000000000001"),
			referenceType:  billing.ReferenceTypeSMS,
			referenceID:    uuid.FromStringOrNil("10000004-0000-0000-0000-000000000001"),
			costType:       billing.CostTypeSMS,
			tmBillingStart: &tmBillingStart,
			source:         &commonaddress.Address{Target: "+1234"},
			destination:    &commonaddress.Address{Target: "+5678"},

			existingBilling: &billing.Billing{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("10000005-0000-0000-0000-000000000001"),
				},
				AccountID:         uuid.FromStringOrNil("10000006-0000-0000-0000-000000000001"),
				Status:            billing.StatusProgressing,
				ReferenceType:     billing.ReferenceTypeSMS,
				ReferenceID:       uuid.FromStringOrNil("10000004-0000-0000-0000-000000000001"),
				CostType:          billing.CostTypeSMS,
				RateTokenPerUnit:  0,
				RateCreditPerUnit: billing.DefaultCreditPerUnitSMS,
				TMBillingStart:    &tmBillingStart,
			},
			responseBilling: &billing.Billing{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("10000005-0000-0000-0000-000000000001"),
				},
				AccountID:         uuid.FromStringOrNil("10000006-0000-0000-0000-000000000001"),
				Status:            billing.StatusEnd,
				ReferenceType:     billing.ReferenceTypeSMS,
				ReferenceID:       uuid.FromStringOrNil("10000004-0000-0000-0000-000000000001"),
				CostType:          billing.CostTypeSMS,
				RateTokenPerUnit:  0,
				RateCreditPerUnit: billing.DefaultCreditPerUnitSMS,
				TMBillingStart:    &tmBillingStart,
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

			// idempotency check - existing billing found with progressing status
			mockDB.EXPECT().BillingGetByReferenceTypeAndID(ctx, tt.referenceType, tt.referenceID).Return(tt.existingBilling, nil)

			// BillingEnd should be called - SMS uses BillingConsumeAndRecord atomically
			mockDB.EXPECT().BillingConsumeAndRecord(ctx, tt.existingBilling, tt.existingBilling.AccountID, 1, 0, int64(0), billing.DefaultCreditPerUnitSMS, tt.tmBillingStart).Return(tt.responseBilling, nil)
			mockNotify.EXPECT().PublishEvent(ctx, billing.EventTypeBillingUpdated, tt.responseBilling)

			err := h.BillingStart(ctx, tt.customerID, tt.referenceType, tt.referenceID, tt.costType, tt.tmBillingStart, tt.source, tt.destination)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_BillingStart_idempotent_progressing_call(t *testing.T) {

	tmBillingStart := time.Date(2023, 6, 8, 3, 22, 17, 995000000, time.UTC)

	tests := []struct {
		name string

		customerID     uuid.UUID
		referenceType  billing.ReferenceType
		referenceID    uuid.UUID
		costType       billing.CostType
		tmBillingStart *time.Time
		source         *commonaddress.Address
		destination    *commonaddress.Address

		existingBilling *billing.Billing
	}{
		{
			name: "existing progressing Call billing - should return early",

			customerID:     uuid.FromStringOrNil("10000007-0000-0000-0000-000000000001"),
			referenceType:  billing.ReferenceTypeCall,
			referenceID:    uuid.FromStringOrNil("10000008-0000-0000-0000-000000000001"),
			costType:       billing.CostTypeCallPSTNOutgoing,
			tmBillingStart: &tmBillingStart,
			source:         &commonaddress.Address{Target: "+1234"},
			destination:    &commonaddress.Address{Target: "+5678"},

			existingBilling: &billing.Billing{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("10000009-0000-0000-0000-000000000001"),
				},
				Status:        billing.StatusProgressing,
				ReferenceType: billing.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("10000008-0000-0000-0000-000000000001"),
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

			// idempotency check - existing billing found with progressing status
			mockDB.EXPECT().BillingGetByReferenceTypeAndID(ctx, tt.referenceType, tt.referenceID).Return(tt.existingBilling, nil)

			// NO further calls expected - calls just return early without ending

			err := h.BillingStart(ctx, tt.customerID, tt.referenceType, tt.referenceID, tt.costType, tt.tmBillingStart, tt.source, tt.destination)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_BillingStart_unsupported_reference_type(t *testing.T) {

	tmBillingStart := time.Date(2023, 6, 8, 3, 22, 17, 995000000, time.UTC)

	tests := []struct {
		name string

		customerID     uuid.UUID
		referenceType  billing.ReferenceType
		referenceID    uuid.UUID
		costType       billing.CostType
		tmBillingStart *time.Time
		source         *commonaddress.Address
		destination    *commonaddress.Address

		responseAccount *account.Account
	}{
		{
			name: "unsupported reference type",

			customerID:     uuid.FromStringOrNil("1000000a-0000-0000-0000-000000000001"),
			referenceType:  billing.ReferenceType("unsupported"),
			referenceID:    uuid.FromStringOrNil("1000000b-0000-0000-0000-000000000001"),
			costType:       billing.CostType("unsupported"),
			tmBillingStart: &tmBillingStart,
			source:         &commonaddress.Address{Target: "+1234"},
			destination:    &commonaddress.Address{Target: "+5678"},

			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("1000000c-0000-0000-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("1000000a-0000-0000-0000-000000000001"),
				},
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
			mockDB.EXPECT().BillingGetByReferenceTypeAndID(ctx, tt.referenceType, tt.referenceID).Return(nil, dbhandler.ErrNotFound)

			// GetByCustomerID succeeds
			mockAccount.EXPECT().GetByCustomerID(ctx, tt.customerID).Return(tt.responseAccount, nil)

			// Should return error "unsupported reference type" before attempting to create

			err := h.BillingStart(ctx, tt.customerID, tt.referenceType, tt.referenceID, tt.costType, tt.tmBillingStart, tt.source, tt.destination)
			if err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			}
		})
	}
}

func Test_BillingStart_account_not_found(t *testing.T) {

	tmBillingStart := time.Date(2023, 6, 8, 3, 22, 17, 995000000, time.UTC)

	tests := []struct {
		name string

		customerID     uuid.UUID
		referenceType  billing.ReferenceType
		referenceID    uuid.UUID
		costType       billing.CostType
		tmBillingStart *time.Time
		source         *commonaddress.Address
		destination    *commonaddress.Address
	}{
		{
			name: "account not found",

			customerID:     uuid.FromStringOrNil("1000000d-0000-0000-0000-000000000001"),
			referenceType:  billing.ReferenceTypeCall,
			referenceID:    uuid.FromStringOrNil("1000000e-0000-0000-0000-000000000001"),
			costType:       billing.CostTypeCallPSTNOutgoing,
			tmBillingStart: &tmBillingStart,
			source:         &commonaddress.Address{Target: "+1234"},
			destination:    &commonaddress.Address{Target: "+5678"},
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
			mockDB.EXPECT().BillingGetByReferenceTypeAndID(ctx, tt.referenceType, tt.referenceID).Return(nil, dbhandler.ErrNotFound)

			// GetByCustomerID returns error
			mockAccount.EXPECT().GetByCustomerID(ctx, tt.customerID).Return(nil, fmt.Errorf("account not found"))

			err := h.BillingStart(ctx, tt.customerID, tt.referenceType, tt.referenceID, tt.costType, tt.tmBillingStart, tt.source, tt.destination)
			if err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			}
		})
	}
}

func Test_BillingStart_create_error(t *testing.T) {

	tmBillingStart := time.Date(2023, 6, 8, 3, 22, 17, 995000000, time.UTC)

	tests := []struct {
		name string

		customerID     uuid.UUID
		referenceType  billing.ReferenceType
		referenceID    uuid.UUID
		costType       billing.CostType
		tmBillingStart *time.Time
		source         *commonaddress.Address
		destination    *commonaddress.Address

		responseAccount *account.Account
		responseUUID    uuid.UUID
	}{
		{
			name: "create error",

			customerID:     uuid.FromStringOrNil("1000000f-0000-0000-0000-000000000001"),
			referenceType:  billing.ReferenceTypeCall,
			referenceID:    uuid.FromStringOrNil("10000010-0000-0000-0000-000000000001"),
			costType:       billing.CostTypeCallPSTNOutgoing,
			tmBillingStart: &tmBillingStart,
			source:         &commonaddress.Address{Target: "+1234"},
			destination:    &commonaddress.Address{Target: "+5678"},

			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("10000011-0000-0000-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("1000000f-0000-0000-0000-000000000001"),
				},
			},
			responseUUID: uuid.FromStringOrNil("10000012-0000-0000-0000-000000000001"),
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
			mockDB.EXPECT().BillingGetByReferenceTypeAndID(ctx, tt.referenceType, tt.referenceID).Return(nil, dbhandler.ErrNotFound)

			// GetByCustomerID succeeds
			mockAccount.EXPECT().GetByCustomerID(ctx, tt.customerID).Return(tt.responseAccount, nil)

			// Create fails
			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockDB.EXPECT().BillingCreate(ctx, gomock.Any()).Return(fmt.Errorf("db connection lost"))

			err := h.BillingStart(ctx, tt.customerID, tt.referenceType, tt.referenceID, tt.costType, tt.tmBillingStart, tt.source, tt.destination)
			if err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			}
		})
	}
}

func Test_BillingEnd_nil_timestamps(t *testing.T) {

	tests := []struct {
		name string

		billing      *billing.Billing
		tmBillingEnd *time.Time
		source       *commonaddress.Address
		destination  *commonaddress.Address

		responseBilling *billing.Billing
	}{
		{
			name: "nil timestamps - should pass zero duration",

			billing: &billing.Billing{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("10000013-0000-0000-0000-000000000001"),
				},
				AccountID:         uuid.FromStringOrNil("10000014-0000-0000-0000-000000000001"),
				ReferenceType:     billing.ReferenceTypeCall,
				CostType:          billing.CostTypeCallPSTNOutgoing,
				RateCreditPerUnit: billing.DefaultCreditPerUnitCallPSTNOutgoing,
				TMBillingStart:    nil,
			},
			tmBillingEnd: nil,
			source:       &commonaddress.Address{},
			destination:  &commonaddress.Address{},

			responseBilling: &billing.Billing{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("10000013-0000-0000-0000-000000000001"),
				},
				AccountID:         uuid.FromStringOrNil("10000014-0000-0000-0000-000000000001"),
				ReferenceType:     billing.ReferenceTypeCall,
				CostType:          billing.CostTypeCallPSTNOutgoing,
				RateCreditPerUnit: billing.DefaultCreditPerUnitCallPSTNOutgoing,
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

			// nil timestamps: usageDuration=0, billableUnits=0
			mockDB.EXPECT().BillingConsumeAndRecord(ctx, tt.billing, tt.billing.AccountID, 0, 0, tt.billing.RateTokenPerUnit, tt.billing.RateCreditPerUnit, (*time.Time)(nil)).Return(tt.responseBilling, nil)
			mockNotify.EXPECT().PublishEvent(ctx, billing.EventTypeBillingUpdated, tt.responseBilling)

			err := h.BillingEnd(ctx, tt.billing, tt.tmBillingEnd, tt.source, tt.destination)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_BillingEnd_unsupported_reference_type(t *testing.T) {

	tmBillingEnd := time.Date(2023, 6, 8, 3, 22, 17, 995000000, time.UTC)

	tests := []struct {
		name string

		billing      *billing.Billing
		tmBillingEnd *time.Time
		source       *commonaddress.Address
		destination  *commonaddress.Address
	}{
		{
			name: "unsupported reference type",

			billing: &billing.Billing{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("10000015-0000-0000-0000-000000000001"),
				},
				ReferenceType: billing.ReferenceType("unknown"),
			},
			tmBillingEnd: &tmBillingEnd,
			source:       &commonaddress.Address{},
			destination:  &commonaddress.Address{},
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

			// NO DB calls expected - should return error immediately

			err := h.BillingEnd(ctx, tt.billing, tt.tmBillingEnd, tt.source, tt.destination)
			if err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			}
		})
	}
}

func Test_BillingEnd_set_status_end_error(t *testing.T) {

	tmBillingStart := time.Date(2023, 6, 8, 3, 22, 7, 991000000, time.UTC)
	tmBillingEnd := time.Date(2023, 6, 8, 3, 22, 17, 995000000, time.UTC)

	tests := []struct {
		name string

		billing      *billing.Billing
		tmBillingEnd *time.Time
		source       *commonaddress.Address
		destination  *commonaddress.Address
	}{
		{
			name: "consume and record error",

			billing: &billing.Billing{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("10000016-0000-0000-0000-000000000001"),
				},
				AccountID:         uuid.FromStringOrNil("10000016-0000-0000-0000-000000000002"),
				ReferenceType:     billing.ReferenceTypeCall,
				CostType:          billing.CostTypeCallPSTNOutgoing,
				RateCreditPerUnit: billing.DefaultCreditPerUnitCallPSTNOutgoing,
				TMBillingStart:    &tmBillingStart,
			},
			tmBillingEnd: &tmBillingEnd,
			source:       &commonaddress.Address{},
			destination:  &commonaddress.Address{},
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

			usageDuration := int(tt.tmBillingEnd.Sub(*tt.billing.TMBillingStart).Seconds())
			billableUnits := billing.CalculateBillableUnits(usageDuration)
			mockDB.EXPECT().BillingConsumeAndRecord(ctx, tt.billing, tt.billing.AccountID, billableUnits, usageDuration, tt.billing.RateTokenPerUnit, tt.billing.RateCreditPerUnit, tt.tmBillingEnd).Return(nil, fmt.Errorf("connection timeout"))

			err := h.BillingEnd(ctx, tt.billing, tt.tmBillingEnd, tt.source, tt.destination)
			if err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			}
		})
	}
}
