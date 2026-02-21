package billinghandler

import (
	"context"
	"fmt"
	"testing"
	"time"

	cmcall "monorepo/bin-call-manager/models/call"

	"monorepo/bin-billing-manager/models/account"
	"monorepo/bin-billing-manager/models/billing"
	"monorepo/bin-billing-manager/pkg/accounthandler"
	"monorepo/bin-billing-manager/pkg/dbhandler"
	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	ememail "monorepo/bin-email-manager/models/email"
	mmmessage "monorepo/bin-message-manager/models/message"
	mmtarget "monorepo/bin-message-manager/models/target"

	nmnumber "monorepo/bin-number-manager/models/number"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func Test_EventCMCallProgressing(t *testing.T) {

	type test struct {
		name string

		call *cmcall.Call

		responseAccount *account.Account
		responseUUID    uuid.UUID

		expectBilling  *billing.Billing
		expectCostType billing.CostType
		expectRefType  billing.ReferenceType
	}

	tests := []test{
		{
			name: "normal",

			call: &cmcall.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("b215ed62-f548-11ee-813d-7f31c7ccb7eb"),
				},
			},

			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("e403a1da-f547-11ee-b4ac-43fc6e27a70b"),
				},
			},
			responseUUID: uuid.FromStringOrNil("0a4ebb9a-f548-11ee-b96f-23e8b75fea2c"),

			expectRefType:  billing.ReferenceTypeCall,
			expectCostType: billing.CostTypeCallPSTNOutgoing,
			expectBilling: &billing.Billing{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("0a4ebb9a-f548-11ee-b96f-23e8b75fea2c"),
				},
				AccountID:         uuid.FromStringOrNil("e403a1da-f547-11ee-b4ac-43fc6e27a70b"),
				TransactionType:   billing.TransactionTypeUsage,
				Status:            billing.StatusProgressing,
				ReferenceType:     billing.ReferenceTypeCall,
				ReferenceID:       uuid.FromStringOrNil("b215ed62-f548-11ee-813d-7f31c7ccb7eb"),
				CostType:          billing.CostTypeCallPSTNOutgoing,
				RateCreditPerUnit: billing.DefaultCreditPerUnitCallPSTNOutgoing,
				TMBillingEnd:      nil,
			},
		},
		{
			name: "outgoing call to extension creates call_extension billing",

			call: &cmcall.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("fb111111-0000-0000-0000-000000000001"),
				},
				Direction: cmcall.DirectionOutgoing,
				Destination: commonaddress.Address{
					Type:   commonaddress.TypeExtension,
					Target: "1002",
				},
			},

			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("fb222222-0000-0000-0000-000000000001"),
				},
			},
			responseUUID: uuid.FromStringOrNil("fb333333-0000-0000-0000-000000000001"),

			expectRefType:  billing.ReferenceTypeCallExtension,
			expectCostType: billing.CostTypeCallExtension,
			expectBilling: &billing.Billing{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("fb333333-0000-0000-0000-000000000001"),
				},
				AccountID:       uuid.FromStringOrNil("fb222222-0000-0000-0000-000000000001"),
				TransactionType: billing.TransactionTypeUsage,
				Status:          billing.StatusProgressing,
				ReferenceType:   billing.ReferenceTypeCallExtension,
				ReferenceID:     uuid.FromStringOrNil("fb111111-0000-0000-0000-000000000001"),
				CostType:        billing.CostTypeCallExtension,
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

			// idempotency check
			mockDB.EXPECT().BillingGetByReferenceTypeAndID(ctx, tt.expectRefType, tt.call.ID).Return(nil, dbhandler.ErrNotFound)

			// BillingStart
			mockAccount.EXPECT().GetByCustomerID(ctx, tt.call.CustomerID).Return(tt.responseAccount, nil)
			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockDB.EXPECT().BillingCreate(ctx, gomock.Any()).Return(nil)
			mockDB.EXPECT().BillingGet(ctx, tt.responseUUID).Return(tt.expectBilling, nil)
			mockNotify.EXPECT().PublishEvent(ctx, billing.EventTypeBillingCreated, tt.expectBilling)

			if err := h.EventCMCallProgressing(ctx, tt.call); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_EventCMCallHangup(t *testing.T) {

	type test struct {
		name string

		call *cmcall.Call

		responseBilling        *billing.Billing
		responseConsumedBilling *billing.Billing
	}

	tmBillingStart := time.Date(2023, 6, 8, 3, 22, 17, 995000000, time.UTC)
	tmHangup := time.Date(2023, 6, 8, 3, 23, 17, 995000000, time.UTC)

	tests := []test{
		{
			name: "normal",

			call: &cmcall.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("beaacf10-f549-11ee-9511-77ae64a3ef25"),
				},
				TMHangup: &tmHangup,
			},

			responseBilling: &billing.Billing{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("39b65350-f54a-11ee-8c56-0b22b45c70b4"),
				},
				AccountID:         uuid.FromStringOrNil("d5cdedca-f54a-11ee-a551-97c7e626fb5f"),
				ReferenceType:     billing.ReferenceTypeCall,
				ReferenceID:       uuid.FromStringOrNil("beaacf10-f549-11ee-9511-77ae64a3ef25"),
				CostType:          billing.CostTypeCallPSTNOutgoing,
				RateCreditPerUnit: billing.DefaultCreditPerUnitCallPSTNOutgoing,
				TMBillingStart:    &tmBillingStart,
			},
			responseConsumedBilling: &billing.Billing{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("39b65350-f54a-11ee-8c56-0b22b45c70b4"),
				},
				AccountID:         uuid.FromStringOrNil("d5cdedca-f54a-11ee-a551-97c7e626fb5f"),
				ReferenceType:     billing.ReferenceTypeCall,
				ReferenceID:       uuid.FromStringOrNil("beaacf10-f549-11ee-9511-77ae64a3ef25"),
				CostType:          billing.CostTypeCallPSTNOutgoing,
				RateCreditPerUnit: billing.DefaultCreditPerUnitCallPSTNOutgoing,
				TMBillingStart:    &tmBillingStart,
				TMBillingEnd:      &tmHangup,
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

			mockDB.EXPECT().BillingGetByReferenceID(ctx, tt.call.ID).Return(tt.responseBilling, nil)

			// BillingEnd - atomic consume and record
			// 60s duration -> ceil(60/60) = 1 billable unit
			mockDB.EXPECT().BillingConsumeAndRecord(
				ctx,
				tt.responseBilling,
				tt.responseBilling.AccountID,
				1,  // billableUnits
				60, // usageDuration (seconds)
				billing.GetCostInfo(tt.responseBilling.CostType),
				tt.call.TMHangup,
			).Return(tt.responseConsumedBilling, nil)
			mockNotify.EXPECT().PublishEvent(ctx, billing.EventTypeBillingUpdated, tt.responseConsumedBilling)

			if err := h.EventCMCallHangup(ctx, tt.call); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_EventMMMessageCreated(t *testing.T) {

	type test struct {
		name string

		message *mmmessage.Message

		responseAccount  *account.Account
		responseUUIDs    []uuid.UUID
		responseBillings []*billing.Billing
		responseConsumed []*billing.Billing
	}

	tests := []test{
		{
			name: "normal",

			message: &mmmessage.Message{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2cb5bb08-f54c-11ee-a40b-0f5555eb875b"),
				},
				Targets: []mmtarget.Target{
					{
						Destination: commonaddress.Address{},
					},
					{
						Destination: commonaddress.Address{},
					},
				},
			},

			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("9435f36a-f54c-11ee-99ff-373fc575fdc9"),
				},
			},
			responseUUIDs: []uuid.UUID{
				uuid.FromStringOrNil("a28315ec-f54c-11ee-ac34-df26f5ac5453"),
				uuid.FromStringOrNil("a2e403d4-f54c-11ee-8880-73605142bc5d"),
			},
			responseBillings: []*billing.Billing{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("a28315ec-f54c-11ee-ac34-df26f5ac5453"),
					},
					AccountID:         uuid.FromStringOrNil("9435f36a-f54c-11ee-99ff-373fc575fdc9"),
					TransactionType:   billing.TransactionTypeUsage,
					Status:            billing.StatusProgressing,
					ReferenceType:     billing.ReferenceTypeSMS,
					CostType:          billing.CostTypeSMS,
					RateTokenPerUnit:  0,
					RateCreditPerUnit: billing.DefaultCreditPerUnitSMS,
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("a2e403d4-f54c-11ee-8880-73605142bc5d"),
					},
					AccountID:         uuid.FromStringOrNil("9435f36a-f54c-11ee-99ff-373fc575fdc9"),
					TransactionType:   billing.TransactionTypeUsage,
					Status:            billing.StatusProgressing,
					ReferenceType:     billing.ReferenceTypeSMS,
					CostType:          billing.CostTypeSMS,
					RateTokenPerUnit:  0,
					RateCreditPerUnit: billing.DefaultCreditPerUnitSMS,
				},
			},
			responseConsumed: []*billing.Billing{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("a28315ec-f54c-11ee-ac34-df26f5ac5453"),
					},
					AccountID:         uuid.FromStringOrNil("9435f36a-f54c-11ee-99ff-373fc575fdc9"),
					TransactionType:   billing.TransactionTypeUsage,
					Status:            billing.StatusEnd,
					ReferenceType:     billing.ReferenceTypeSMS,
					CostType:          billing.CostTypeSMS,
					RateTokenPerUnit:  0,
					RateCreditPerUnit: billing.DefaultCreditPerUnitSMS,
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("a2e403d4-f54c-11ee-8880-73605142bc5d"),
					},
					AccountID:         uuid.FromStringOrNil("9435f36a-f54c-11ee-99ff-373fc575fdc9"),
					TransactionType:   billing.TransactionTypeUsage,
					Status:            billing.StatusEnd,
					ReferenceType:     billing.ReferenceTypeSMS,
					CostType:          billing.CostTypeSMS,
					RateTokenPerUnit:  0,
					RateCreditPerUnit: billing.DefaultCreditPerUnitSMS,
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

			for i := range tt.message.Targets {
				// per-target deterministic reference ID
				targetRefID := uuid.NewV5(tt.message.ID, fmt.Sprintf("target-%d", i))

				// idempotency check
				mockDB.EXPECT().BillingGetByReferenceTypeAndID(ctx, billing.ReferenceTypeSMS, targetRefID).Return(nil, dbhandler.ErrNotFound)

				// BillingStart -> Create
				mockAccount.EXPECT().GetByCustomerID(ctx, tt.message.CustomerID).Return(tt.responseAccount, nil)
				mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDs[i])
				mockDB.EXPECT().BillingCreate(ctx, gomock.Any()).Return(nil)
				mockDB.EXPECT().BillingGet(ctx, tt.responseUUIDs[i]).Return(tt.responseBillings[i], nil)
				mockNotify.EXPECT().PublishEvent(ctx, billing.EventTypeBillingCreated, tt.responseBillings[i])

				// BillingEnd - atomic consume and record (SMS: 1 billable unit, 0 usage duration)
				mockDB.EXPECT().BillingConsumeAndRecord(
					ctx,
					tt.responseBillings[i],
					tt.responseBillings[i].AccountID,
					1, // billableUnits
					0, // usageDuration
					billing.GetCostInfo(tt.responseBillings[i].CostType),
					gomock.Any(), // tmBillingEnd
				).Return(tt.responseConsumed[i], nil)
				mockNotify.EXPECT().PublishEvent(ctx, billing.EventTypeBillingUpdated, tt.responseConsumed[i])
			}

			if err := h.EventMMMessageCreated(ctx, tt.message); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_EventNMNumberCreated(t *testing.T) {

	type test struct {
		name string

		number *nmnumber.Number

		responseAccount  *account.Account
		responseUUID     uuid.UUID
		responseBilling  *billing.Billing
		responseConsumed *billing.Billing
	}

	tests := []test{
		{
			name: "normal",

			number: &nmnumber.Number{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("7359bada-f54e-11ee-ae36-37d1feaf6c4c"),
					CustomerID: uuid.FromStringOrNil("74483cdc-f54e-11ee-ac89-bb4150764799"),
				},
			},

			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("74057276-f54e-11ee-b35b-cf292d0c7298"),
				},
			},
			responseUUID: uuid.FromStringOrNil("73c8040e-f54e-11ee-a59f-2ba1b61918fd"),
			responseBilling: &billing.Billing{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("73c8040e-f54e-11ee-a59f-2ba1b61918fd"),
				},
				AccountID:         uuid.FromStringOrNil("74057276-f54e-11ee-b35b-cf292d0c7298"),
				TransactionType:   billing.TransactionTypeUsage,
				Status:            billing.StatusProgressing,
				ReferenceType:     billing.ReferenceTypeNumber,
				ReferenceID:       uuid.FromStringOrNil("7359bada-f54e-11ee-ae36-37d1feaf6c4c"),
				CostType:          billing.CostTypeNumber,
				RateCreditPerUnit: billing.DefaultCreditPerUnitNumber,
			},
			responseConsumed: &billing.Billing{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("73c8040e-f54e-11ee-a59f-2ba1b61918fd"),
				},
				AccountID:         uuid.FromStringOrNil("74057276-f54e-11ee-b35b-cf292d0c7298"),
				TransactionType:   billing.TransactionTypeUsage,
				Status:            billing.StatusEnd,
				ReferenceType:     billing.ReferenceTypeNumber,
				ReferenceID:       uuid.FromStringOrNil("7359bada-f54e-11ee-ae36-37d1feaf6c4c"),
				CostType:          billing.CostTypeNumber,
				RateCreditPerUnit: billing.DefaultCreditPerUnitNumber,
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
			mockDB.EXPECT().BillingGetByReferenceTypeAndID(ctx, billing.ReferenceTypeNumber, tt.number.ID).Return(nil, dbhandler.ErrNotFound)

			// BillingStart -> Create
			mockAccount.EXPECT().GetByCustomerID(ctx, tt.number.CustomerID).Return(tt.responseAccount, nil)
			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockDB.EXPECT().BillingCreate(ctx, gomock.Any()).Return(nil)
			mockDB.EXPECT().BillingGet(ctx, tt.responseUUID).Return(tt.responseBilling, nil)
			mockNotify.EXPECT().PublishEvent(ctx, billing.EventTypeBillingCreated, tt.responseBilling)

			// BillingEnd - atomic consume and record (Number: 1 billable unit, 0 usage duration)
			mockDB.EXPECT().BillingConsumeAndRecord(
				ctx,
				tt.responseBilling,
				tt.responseBilling.AccountID,
				1, // billableUnits
				0, // usageDuration
				billing.GetCostInfo(tt.responseBilling.CostType),
				gomock.Any(), // tmBillingEnd
			).Return(tt.responseConsumed, nil)
			mockNotify.EXPECT().PublishEvent(ctx, billing.EventTypeBillingUpdated, tt.responseConsumed)

			if err := h.EventNMNumberCreated(ctx, tt.number); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

		})
	}
}

func Test_EventNMNumberCreated_virtual(t *testing.T) {

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

	n := &nmnumber.Number{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("a1b2c3d4-e5f6-11ee-aaaa-000000000001"),
			CustomerID: uuid.FromStringOrNil("a1b2c3d4-e5f6-11ee-aaaa-000000000002"),
		},
		Type: nmnumber.TypeVirtual,
	}

	// no mock expectations -- billing should be skipped entirely

	if err := h.EventNMNumberCreated(ctx, n); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
}

func Test_EventNMNumberRenewed(t *testing.T) {

	type test struct {
		name string

		number *nmnumber.Number

		responseAccount   *account.Account
		responseUUID      uuid.UUID
		responseTimeNow   *time.Time
		referenceIDExpect uuid.UUID
		responseBilling   *billing.Billing
		responseConsumed  *billing.Billing
	}

	now := time.Date(2026, 2, 12, 10, 0, 0, 0, time.UTC)
	numberID := uuid.FromStringOrNil("e2eda0c8-f54e-11ee-9c57-c76cbcca2410")
	expectedRefID := uuid.NewV5(uuid.Nil, numberID.String()+":renew:2026-02")

	tests := []test{
		{
			name: "normal",

			number: &nmnumber.Number{
				Identity: commonidentity.Identity{
					ID:         numberID,
					CustomerID: uuid.FromStringOrNil("e3537aa6-f54e-11ee-84fb-bb29ab77496c"),
				},
			},

			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("e38e5c34-f54e-11ee-9f4c-bf30ab98b5c1"),
				},
			},
			responseUUID:      uuid.FromStringOrNil("e3c41b80-f54e-11ee-becf-33857841a543"),
			responseTimeNow:   &now,
			referenceIDExpect: expectedRefID,
			responseBilling: &billing.Billing{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("e3c41b80-f54e-11ee-becf-33857841a543"),
				},
				AccountID:         uuid.FromStringOrNil("e38e5c34-f54e-11ee-9f4c-bf30ab98b5c1"),
				TransactionType:   billing.TransactionTypeUsage,
				Status:            billing.StatusProgressing,
				ReferenceType:     billing.ReferenceTypeNumberRenew,
				ReferenceID:       expectedRefID,
				CostType:          billing.CostTypeNumberRenew,
				RateCreditPerUnit: billing.DefaultCreditPerUnitNumber,
			},
			responseConsumed: &billing.Billing{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("e3c41b80-f54e-11ee-becf-33857841a543"),
				},
				AccountID:         uuid.FromStringOrNil("e38e5c34-f54e-11ee-9f4c-bf30ab98b5c1"),
				TransactionType:   billing.TransactionTypeUsage,
				Status:            billing.StatusEnd,
				ReferenceType:     billing.ReferenceTypeNumberRenew,
				ReferenceID:       expectedRefID,
				CostType:          billing.CostTypeNumberRenew,
				RateCreditPerUnit: billing.DefaultCreditPerUnitNumber,
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

			// deterministic reference ID generation
			mockUtil.EXPECT().TimeNow().Return(tt.responseTimeNow)
			mockUtil.EXPECT().NewV5UUID(uuid.Nil, tt.number.ID.String()+":renew:"+tt.responseTimeNow.Format("2006-01")).Return(tt.referenceIDExpect)

			// idempotency check
			mockDB.EXPECT().BillingGetByReferenceTypeAndID(ctx, billing.ReferenceTypeNumberRenew, tt.referenceIDExpect).Return(nil, dbhandler.ErrNotFound)

			// BillingStart -> Create
			mockAccount.EXPECT().GetByCustomerID(ctx, tt.number.CustomerID).Return(tt.responseAccount, nil)
			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockDB.EXPECT().BillingCreate(ctx, gomock.Any()).Return(nil)
			mockDB.EXPECT().BillingGet(ctx, tt.responseUUID).Return(tt.responseBilling, nil)
			mockNotify.EXPECT().PublishEvent(ctx, billing.EventTypeBillingCreated, tt.responseBilling)

			// BillingEnd - atomic consume and record (Number renew: 1 billable unit, 0 usage duration)
			mockDB.EXPECT().BillingConsumeAndRecord(
				ctx,
				tt.responseBilling,
				tt.responseBilling.AccountID,
				1, // billableUnits
				0, // usageDuration
				billing.GetCostInfo(tt.responseBilling.CostType),
				gomock.Any(), // tmBillingEnd
			).Return(tt.responseConsumed, nil)
			mockNotify.EXPECT().PublishEvent(ctx, billing.EventTypeBillingUpdated, tt.responseConsumed)

			if err := h.EventNMNumberRenewed(ctx, tt.number); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_EventNMNumberRenewed_virtual(t *testing.T) {

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

	n := &nmnumber.Number{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("b2c3d4e5-f6a7-11ee-bbbb-000000000001"),
			CustomerID: uuid.FromStringOrNil("b2c3d4e5-f6a7-11ee-bbbb-000000000002"),
		},
		Type: nmnumber.TypeVirtual,
	}

	// no mock expectations -- billing should be skipped entirely

	if err := h.EventNMNumberRenewed(ctx, n); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
}

func Test_getReferenceTypeForCall(t *testing.T) {

	tests := []struct {
		name string

		call *cmcall.Call

		expectReferenceType billing.ReferenceType
	}{
		{
			name: "incoming call from PSTN",
			call: &cmcall.Call{
				Direction: cmcall.DirectionIncoming,
				Source: commonaddress.Address{
					Type: commonaddress.TypeTel,
				},
			},
			expectReferenceType: billing.ReferenceTypeCall,
		},
		{
			name: "incoming call from extension",
			call: &cmcall.Call{
				Direction: cmcall.DirectionIncoming,
				Source: commonaddress.Address{
					Type: commonaddress.TypeExtension,
				},
			},
			expectReferenceType: billing.ReferenceTypeCallExtension,
		},
		{
			name: "incoming call from agent",
			call: &cmcall.Call{
				Direction: cmcall.DirectionIncoming,
				Source: commonaddress.Address{
					Type: commonaddress.TypeAgent,
				},
			},
			expectReferenceType: billing.ReferenceTypeCallExtension,
		},
		{
			name: "incoming call from sip",
			call: &cmcall.Call{
				Direction: cmcall.DirectionIncoming,
				Source: commonaddress.Address{
					Type: commonaddress.TypeSIP,
				},
			},
			expectReferenceType: billing.ReferenceTypeCallExtension,
		},
		{
			name: "outgoing call to PSTN",
			call: &cmcall.Call{
				Direction: cmcall.DirectionOutgoing,
				Destination: commonaddress.Address{
					Type: commonaddress.TypeTel,
				},
			},
			expectReferenceType: billing.ReferenceTypeCall,
		},
		{
			name: "outgoing call to extension",
			call: &cmcall.Call{
				Direction: cmcall.DirectionOutgoing,
				Destination: commonaddress.Address{
					Type: commonaddress.TypeExtension,
				},
			},
			expectReferenceType: billing.ReferenceTypeCallExtension,
		},
		{
			name: "outgoing call to agent",
			call: &cmcall.Call{
				Direction: cmcall.DirectionOutgoing,
				Destination: commonaddress.Address{
					Type: commonaddress.TypeAgent,
				},
			},
			expectReferenceType: billing.ReferenceTypeCallExtension,
		},
		{
			name: "outgoing call to sip",
			call: &cmcall.Call{
				Direction: cmcall.DirectionOutgoing,
				Destination: commonaddress.Address{
					Type: commonaddress.TypeSIP,
				},
			},
			expectReferenceType: billing.ReferenceTypeCallExtension,
		},
		{
			name: "outgoing call to conference",
			call: &cmcall.Call{
				Direction: cmcall.DirectionOutgoing,
				Destination: commonaddress.Address{
					Type: commonaddress.TypeConference,
				},
			},
			expectReferenceType: billing.ReferenceTypeCallExtension,
		},
		{
			name: "outgoing call to line",
			call: &cmcall.Call{
				Direction: cmcall.DirectionOutgoing,
				Destination: commonaddress.Address{
					Type: commonaddress.TypeLine,
				},
			},
			expectReferenceType: billing.ReferenceTypeCallExtension,
		},
		{
			name: "outgoing call to email",
			call: &cmcall.Call{
				Direction: cmcall.DirectionOutgoing,
				Destination: commonaddress.Address{
					Type: commonaddress.TypeEmail,
				},
			},
			expectReferenceType: billing.ReferenceTypeCallExtension,
		},
		{
			name: "incoming call with no source type",
			call: &cmcall.Call{
				Direction: cmcall.DirectionIncoming,
				Source: commonaddress.Address{
					Type: commonaddress.TypeNone,
				},
			},
			expectReferenceType: billing.ReferenceTypeCallExtension,
		},
		{
			name: "unknown direction defaults to call",
			call: &cmcall.Call{
				Direction: "",
			},
			expectReferenceType: billing.ReferenceTypeCall,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := getReferenceTypeForCall(tt.call)
			if res != tt.expectReferenceType {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expectReferenceType, res)
			}
		})
	}
}

func Test_getCostTypeForCall(t *testing.T) {

	tests := []struct {
		name string

		call *cmcall.Call

		expectCostType billing.CostType
	}{
		{
			name: "incoming call to virtual number (dst=tel with +899 prefix)",
			call: &cmcall.Call{
				Direction: cmcall.DirectionIncoming,
				Source: commonaddress.Address{
					Type:   commonaddress.TypeSIP,
					Target: "sip:user@example.com",
				},
				Destination: commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: nmnumber.VirtualNumberPrefix + "1234567",
				},
			},
			expectCostType: billing.CostTypeCallVN,
		},
		{
			name: "incoming PSTN call (dst=tel, no +899 prefix)",
			call: &cmcall.Call{
				Direction: cmcall.DirectionIncoming,
				Source: commonaddress.Address{
					Type: commonaddress.TypeTel,
				},
				Destination: commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+14155551234",
				},
			},
			expectCostType: billing.CostTypeCallPSTNIncoming,
		},
		{
			name: "incoming call from SIP to tel (not +899) - free (not PSTN without src=tel)",
			call: &cmcall.Call{
				Direction: cmcall.DirectionIncoming,
				Source: commonaddress.Address{
					Type:   commonaddress.TypeSIP,
					Target: "sip:trunk@provider.com",
				},
				Destination: commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+14155551234",
				},
			},
			expectCostType: billing.CostTypeCallExtension,
		},
		{
			name: "incoming direct extension (src=sip, dst=extension)",
			call: &cmcall.Call{
				Direction: cmcall.DirectionIncoming,
				Source: commonaddress.Address{
					Type:   commonaddress.TypeSIP,
					Target: "sip:user@example.com",
				},
				Destination: commonaddress.Address{
					Type:   commonaddress.TypeExtension,
					Target: "1001",
				},
			},
			expectCostType: billing.CostTypeCallDirectExt,
		},
		{
			name: "incoming call from extension - free",
			call: &cmcall.Call{
				Direction: cmcall.DirectionIncoming,
				Source: commonaddress.Address{
					Type: commonaddress.TypeExtension,
				},
				Destination: commonaddress.Address{
					Type:   commonaddress.TypeExtension,
					Target: "1001",
				},
			},
			expectCostType: billing.CostTypeCallExtension,
		},
		{
			name: "incoming call from agent - free",
			call: &cmcall.Call{
				Direction: cmcall.DirectionIncoming,
				Source: commonaddress.Address{
					Type: commonaddress.TypeAgent,
				},
				Destination: commonaddress.Address{
					Type:   commonaddress.TypeExtension,
					Target: "1002",
				},
			},
			expectCostType: billing.CostTypeCallExtension,
		},
		{
			name: "outgoing call to PSTN",
			call: &cmcall.Call{
				Direction: cmcall.DirectionOutgoing,
				Destination: commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+14155551234",
				},
			},
			expectCostType: billing.CostTypeCallPSTNOutgoing,
		},
		{
			name: "outgoing call to extension - free",
			call: &cmcall.Call{
				Direction: cmcall.DirectionOutgoing,
				Destination: commonaddress.Address{
					Type:   commonaddress.TypeExtension,
					Target: "1003",
				},
			},
			expectCostType: billing.CostTypeCallExtension,
		},
		{
			name: "outgoing call to agent - free",
			call: &cmcall.Call{
				Direction: cmcall.DirectionOutgoing,
				Destination: commonaddress.Address{
					Type: commonaddress.TypeAgent,
				},
			},
			expectCostType: billing.CostTypeCallExtension,
		},
		{
			name: "outgoing call to SIP - free",
			call: &cmcall.Call{
				Direction: cmcall.DirectionOutgoing,
				Destination: commonaddress.Address{
					Type: commonaddress.TypeSIP,
				},
			},
			expectCostType: billing.CostTypeCallExtension,
		},
		{
			name: "outgoing call to conference - free",
			call: &cmcall.Call{
				Direction: cmcall.DirectionOutgoing,
				Destination: commonaddress.Address{
					Type: commonaddress.TypeConference,
				},
			},
			expectCostType: billing.CostTypeCallExtension,
		},
		{
			name: "unknown direction defaults to PSTN outgoing",
			call: &cmcall.Call{
				Direction: "",
			},
			expectCostType: billing.CostTypeCallPSTNOutgoing,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := getCostTypeForCall(tt.call)
			if res != tt.expectCostType {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expectCostType, res)
			}
		})
	}
}

func Test_EventCMCallHangup_nil_tmhangup(t *testing.T) {

	tests := []struct {
		name string

		call *cmcall.Call

		responseBilling *billing.Billing
	}{
		{
			name: "nil tm_hangup should error",

			call: &cmcall.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("20000001-0000-0000-0000-000000000001"),
				},
				TMHangup: nil,
			},

			responseBilling: &billing.Billing{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("20000002-0000-0000-0000-000000000001"),
				},
				ReferenceType: billing.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("20000001-0000-0000-0000-000000000001"),
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

			// BillingGetByReferenceID returns valid billing
			mockDB.EXPECT().BillingGetByReferenceID(ctx, tt.call.ID).Return(tt.responseBilling, nil)

			// NO further calls expected - should return error

			err := h.EventCMCallHangup(ctx, tt.call)
			if err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			}
		})
	}
}

func Test_EventCMCallHangup_billing_not_found(t *testing.T) {

	tmHangup := time.Date(2023, 6, 8, 3, 23, 17, 995000000, time.UTC)

	tests := []struct {
		name string

		call *cmcall.Call
	}{
		{
			name: "billing not found - should return nil",

			call: &cmcall.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("20000003-0000-0000-0000-000000000001"),
				},
				TMHangup: &tmHangup,
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

			// BillingGetByReferenceID returns error
			mockDB.EXPECT().BillingGetByReferenceID(ctx, tt.call.ID).Return(nil, fmt.Errorf("not found"))

			// Should return nil (silently ignores)

			err := h.EventCMCallHangup(ctx, tt.call)
			if err != nil {
				t.Errorf("Wrong match. expect: nil, got: %v", err)
			}
		})
	}
}

func Test_EventMMMessageCreated_empty_targets(t *testing.T) {

	tests := []struct {
		name string

		message *mmmessage.Message
	}{
		{
			name: "empty targets - should return nil",

			message: &mmmessage.Message{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("20000004-0000-0000-0000-000000000001"),
				},
				Targets: []mmtarget.Target{},
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

			// NO DB or account calls expected

			err := h.EventMMMessageCreated(ctx, tt.message)
			if err != nil {
				t.Errorf("Wrong match. expect: nil, got: %v", err)
			}
		})
	}
}

func Test_EventMMMessageCreated_error_on_first_target(t *testing.T) {

	tests := []struct {
		name string

		message *mmmessage.Message
	}{
		{
			name: "error on first target - should not process second",

			message: &mmmessage.Message{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("20000005-0000-0000-0000-000000000001"),
				},
				Targets: []mmtarget.Target{
					{
						Destination: commonaddress.Address{},
					},
					{
						Destination: commonaddress.Address{},
					},
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

			// First target: idempotency check passes, but GetByCustomerID fails
			targetRefID := uuid.NewV5(tt.message.ID, "target-0")
			mockDB.EXPECT().BillingGetByReferenceTypeAndID(ctx, billing.ReferenceTypeSMS, targetRefID).Return(nil, dbhandler.ErrNotFound)
			mockAccount.EXPECT().GetByCustomerID(ctx, tt.message.CustomerID).Return(nil, fmt.Errorf("account not found"))

			// Should return error immediately - second target NOT processed

			err := h.EventMMMessageCreated(ctx, tt.message)
			if err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			}
		})
	}
}

func Test_EventNMNumberCreated_billing_error(t *testing.T) {

	tests := []struct {
		name string

		number *nmnumber.Number
	}{
		{
			name: "non-virtual number with billing error",

			number: &nmnumber.Number{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("20000006-0000-0000-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("20000007-0000-0000-0000-000000000001"),
				},
				Type: nmnumber.TypeNormal,
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
			mockDB.EXPECT().BillingGetByReferenceTypeAndID(ctx, billing.ReferenceTypeNumber, tt.number.ID).Return(nil, dbhandler.ErrNotFound)

			// GetByCustomerID returns error
			mockAccount.EXPECT().GetByCustomerID(ctx, tt.number.CustomerID).Return(nil, fmt.Errorf("account not found"))

			err := h.EventNMNumberCreated(ctx, tt.number)
			if err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			}
		})
	}
}

func Test_EventEMEmailCreated(t *testing.T) {

	type test struct {
		name string

		email *ememail.Email

		responseAccount  *account.Account
		responseUUIDs    []uuid.UUID
		responseBillings []*billing.Billing
		responseConsumed []*billing.Billing
	}

	tests := []test{
		{
			name: "single destination",

			email: &ememail.Email{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("e0000001-0000-0000-0000-000000000001"),
				},
				Source: &commonaddress.Address{
					Type:   commonaddress.TypeEmail,
					Target: "from@example.com",
				},
				Destinations: []commonaddress.Address{
					{
						Type:   commonaddress.TypeEmail,
						Target: "to@example.com",
					},
				},
			},

			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("e0000002-0000-0000-0000-000000000001"),
				},
			},
			responseUUIDs: []uuid.UUID{
				uuid.FromStringOrNil("e0000003-0000-0000-0000-000000000001"),
			},
			responseBillings: []*billing.Billing{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("e0000003-0000-0000-0000-000000000001"),
					},
					AccountID:         uuid.FromStringOrNil("e0000002-0000-0000-0000-000000000001"),
					TransactionType:   billing.TransactionTypeUsage,
					Status:            billing.StatusProgressing,
					ReferenceType:     billing.ReferenceTypeEmail,
					CostType:          billing.CostTypeEmail,
					RateTokenPerUnit:  0,
					RateCreditPerUnit: billing.DefaultCreditPerUnitEmail,
				},
			},
			responseConsumed: []*billing.Billing{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("e0000003-0000-0000-0000-000000000001"),
					},
					AccountID:         uuid.FromStringOrNil("e0000002-0000-0000-0000-000000000001"),
					TransactionType:   billing.TransactionTypeUsage,
					Status:            billing.StatusEnd,
					ReferenceType:     billing.ReferenceTypeEmail,
					CostType:          billing.CostTypeEmail,
					RateTokenPerUnit:  0,
					RateCreditPerUnit: billing.DefaultCreditPerUnitEmail,
				},
			},
		},
		{
			name: "multiple destinations",

			email: &ememail.Email{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("e0000004-0000-0000-0000-000000000001"),
				},
				Source: &commonaddress.Address{
					Type:   commonaddress.TypeEmail,
					Target: "from@example.com",
				},
				Destinations: []commonaddress.Address{
					{
						Type:   commonaddress.TypeEmail,
						Target: "to1@example.com",
					},
					{
						Type:   commonaddress.TypeEmail,
						Target: "to2@example.com",
					},
				},
			},

			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("e0000005-0000-0000-0000-000000000001"),
				},
			},
			responseUUIDs: []uuid.UUID{
				uuid.FromStringOrNil("e0000006-0000-0000-0000-000000000001"),
				uuid.FromStringOrNil("e0000006-0000-0000-0000-000000000002"),
			},
			responseBillings: []*billing.Billing{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("e0000006-0000-0000-0000-000000000001"),
					},
					AccountID:         uuid.FromStringOrNil("e0000005-0000-0000-0000-000000000001"),
					TransactionType:   billing.TransactionTypeUsage,
					Status:            billing.StatusProgressing,
					ReferenceType:     billing.ReferenceTypeEmail,
					CostType:          billing.CostTypeEmail,
					RateTokenPerUnit:  0,
					RateCreditPerUnit: billing.DefaultCreditPerUnitEmail,
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("e0000006-0000-0000-0000-000000000002"),
					},
					AccountID:         uuid.FromStringOrNil("e0000005-0000-0000-0000-000000000001"),
					TransactionType:   billing.TransactionTypeUsage,
					Status:            billing.StatusProgressing,
					ReferenceType:     billing.ReferenceTypeEmail,
					CostType:          billing.CostTypeEmail,
					RateTokenPerUnit:  0,
					RateCreditPerUnit: billing.DefaultCreditPerUnitEmail,
				},
			},
			responseConsumed: []*billing.Billing{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("e0000006-0000-0000-0000-000000000001"),
					},
					AccountID:         uuid.FromStringOrNil("e0000005-0000-0000-0000-000000000001"),
					TransactionType:   billing.TransactionTypeUsage,
					Status:            billing.StatusEnd,
					ReferenceType:     billing.ReferenceTypeEmail,
					CostType:          billing.CostTypeEmail,
					RateTokenPerUnit:  0,
					RateCreditPerUnit: billing.DefaultCreditPerUnitEmail,
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("e0000006-0000-0000-0000-000000000002"),
					},
					AccountID:         uuid.FromStringOrNil("e0000005-0000-0000-0000-000000000001"),
					TransactionType:   billing.TransactionTypeUsage,
					Status:            billing.StatusEnd,
					ReferenceType:     billing.ReferenceTypeEmail,
					CostType:          billing.CostTypeEmail,
					RateTokenPerUnit:  0,
					RateCreditPerUnit: billing.DefaultCreditPerUnitEmail,
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

			for i := range tt.email.Destinations {
				// per-destination deterministic reference ID
				targetRefID := uuid.NewV5(tt.email.ID, fmt.Sprintf("target-%d", i))

				// idempotency check
				mockDB.EXPECT().BillingGetByReferenceTypeAndID(ctx, billing.ReferenceTypeEmail, targetRefID).Return(nil, dbhandler.ErrNotFound)

				// BillingStart -> Create
				mockAccount.EXPECT().GetByCustomerID(ctx, tt.email.CustomerID).Return(tt.responseAccount, nil)
				mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDs[i])
				mockDB.EXPECT().BillingCreate(ctx, gomock.Any()).Return(nil)
				mockDB.EXPECT().BillingGet(ctx, tt.responseUUIDs[i]).Return(tt.responseBillings[i], nil)
				mockNotify.EXPECT().PublishEvent(ctx, billing.EventTypeBillingCreated, tt.responseBillings[i])

				// BillingEnd - atomic consume and record (Email: 1 billable unit, 0 usage duration)
				mockDB.EXPECT().BillingConsumeAndRecord(
					ctx,
					tt.responseBillings[i],
					tt.responseBillings[i].AccountID,
					1, // billableUnits
					0, // usageDuration
					billing.GetCostInfo(tt.responseBillings[i].CostType),
					gomock.Any(), // tmBillingEnd
				).Return(tt.responseConsumed[i], nil)
				mockNotify.EXPECT().PublishEvent(ctx, billing.EventTypeBillingUpdated, tt.responseConsumed[i])
			}

			if err := h.EventEMEmailCreated(ctx, tt.email); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_EventEMEmailCreated_billing_error(t *testing.T) {

	tests := []struct {
		name string

		email *ememail.Email
	}{
		{
			name: "BillingStart fails - returns error",

			email: &ememail.Email{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("e0000007-0000-0000-0000-000000000001"),
				},
				Source: &commonaddress.Address{
					Type:   commonaddress.TypeEmail,
					Target: "from@example.com",
				},
				Destinations: []commonaddress.Address{
					{
						Type:   commonaddress.TypeEmail,
						Target: "to@example.com",
					},
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

			// First destination: idempotency check passes, but GetByCustomerID fails
			targetRefID := uuid.NewV5(tt.email.ID, "target-0")
			mockDB.EXPECT().BillingGetByReferenceTypeAndID(ctx, billing.ReferenceTypeEmail, targetRefID).Return(nil, dbhandler.ErrNotFound)
			mockAccount.EXPECT().GetByCustomerID(ctx, tt.email.CustomerID).Return(nil, fmt.Errorf("account not found"))

			// Should return error immediately

			err := h.EventEMEmailCreated(ctx, tt.email)
			if err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			}
		})
	}
}

func Test_EventNMNumberRenewed_billing_error(t *testing.T) {

	now := time.Date(2026, 2, 12, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name string

		number *nmnumber.Number
	}{
		{
			name: "non-virtual number with billing error on renew",

			number: &nmnumber.Number{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("20000008-0000-0000-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("20000009-0000-0000-0000-000000000001"),
				},
				Type: nmnumber.TypeNormal,
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

			// deterministic reference ID generation
			expectedRefID := uuid.NewV5(uuid.Nil, tt.number.ID.String()+":renew:"+now.Format("2006-01"))
			mockUtil.EXPECT().TimeNow().Return(&now)
			mockUtil.EXPECT().NewV5UUID(uuid.Nil, tt.number.ID.String()+":renew:"+now.Format("2006-01")).Return(expectedRefID)

			// idempotency check - no existing billing
			mockDB.EXPECT().BillingGetByReferenceTypeAndID(ctx, billing.ReferenceTypeNumberRenew, expectedRefID).Return(nil, dbhandler.ErrNotFound)

			// GetByCustomerID returns error
			mockAccount.EXPECT().GetByCustomerID(ctx, tt.number.CustomerID).Return(nil, fmt.Errorf("account not found"))

			err := h.EventNMNumberRenewed(ctx, tt.number)
			if err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			}
		})
	}
}
