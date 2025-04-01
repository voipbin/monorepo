package billinghandler

import (
	"context"
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

		expectBilling *billing.Billing
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

			expectBilling: &billing.Billing{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("0a4ebb9a-f548-11ee-b96f-23e8b75fea2c"),
				},
				AccountID:     uuid.FromStringOrNil("e403a1da-f547-11ee-b4ac-43fc6e27a70b"),
				Status:        billing.StatusProgressing,
				ReferenceType: billing.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("b215ed62-f548-11ee-813d-7f31c7ccb7eb"),
				CostPerUnit:   billing.DefaultCostPerUnitReferenceTypeCall,
				TMBillingEnd:  dbhandler.DefaultTimeStamp,
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

			// BillingStart
			mockAccount.EXPECT().GetByCustomerID(ctx, tt.call.CustomerID).Return(tt.responseAccount, nil)
			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockDB.EXPECT().BillingCreate(ctx, tt.expectBilling).Return(nil)
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

		responseBilling *billing.Billing
		responseAccount *account.Account

		expectBilling         *billing.Billing
		expectBillingDuration float32
	}

	tests := []test{
		{
			name: "normal",

			call: &cmcall.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("beaacf10-f549-11ee-9511-77ae64a3ef25"),
				},
				TMHangup: "2023-06-08 03:23:17.995000",
			},

			responseBilling: &billing.Billing{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("39b65350-f54a-11ee-8c56-0b22b45c70b4"),
				},
				ReferenceType:  billing.ReferenceTypeCall,
				ReferenceID:    uuid.FromStringOrNil("beaacf10-f549-11ee-9511-77ae64a3ef25"),
				TMBillingStart: "2023-06-08 03:22:17.995000",
			},
			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("d5cdedca-f54a-11ee-a551-97c7e626fb5f"),
				},
			},

			expectBilling: &billing.Billing{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("0a4ebb9a-f548-11ee-b96f-23e8b75fea2c"),
				},
				AccountID:     uuid.FromStringOrNil("d5cdedca-f54a-11ee-a551-97c7e626fb5f"),
				Status:        billing.StatusProgressing,
				ReferenceType: billing.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("b215ed62-f548-11ee-813d-7f31c7ccb7eb"),
				CostPerUnit:   billing.DefaultCostPerUnitReferenceTypeCall,
				TMBillingEnd:  dbhandler.DefaultTimeStamp,
			},
			expectBillingDuration: float32(60),
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

			// BillingEnd
			mockUtil.EXPECT().TimeParse(tt.responseBilling.TMBillingStart).Return(utilhandler.TimeParse(tt.responseBilling.TMBillingStart))
			mockUtil.EXPECT().TimeParse(tt.call.TMHangup).Return(utilhandler.TimeParse(tt.call.TMHangup))
			mockDB.EXPECT().BillingSetStatusEnd(ctx, tt.responseBilling.ID, tt.expectBillingDuration, tt.call.TMHangup).Return(nil)
			mockDB.EXPECT().BillingGet(ctx, tt.responseBilling.ID).Return(tt.responseBilling, nil)

			mockAccount.EXPECT().SubtractBalance(ctx, tt.responseBilling.AccountID, tt.responseBilling.CostTotal).Return(tt.responseAccount, nil)

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

		responseAccount *account.Account
		responseUUIDs   []uuid.UUID

		expectBillings []*billing.Billing
	}

	tests := []test{
		{
			name: "normal",

			message: &mmmessage.Message{
				ID: uuid.FromStringOrNil("2cb5bb08-f54c-11ee-a40b-0f5555eb875b"),
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

			expectBillings: []*billing.Billing{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("a28315ec-f54c-11ee-ac34-df26f5ac5453"),
					},
					AccountID:     uuid.FromStringOrNil("9435f36a-f54c-11ee-99ff-373fc575fdc9"),
					Status:        billing.StatusProgressing,
					ReferenceType: billing.ReferenceTypeSMS,
					ReferenceID:   uuid.FromStringOrNil("2cb5bb08-f54c-11ee-a40b-0f5555eb875b"),
					CostPerUnit:   billing.DefaultCostPerUnitReferenceTypeSMS,
					TMBillingEnd:  dbhandler.DefaultTimeStamp,
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("a2e403d4-f54c-11ee-8880-73605142bc5d"),
					},
					AccountID:     uuid.FromStringOrNil("9435f36a-f54c-11ee-99ff-373fc575fdc9"),
					Status:        billing.StatusProgressing,
					ReferenceType: billing.ReferenceTypeSMS,
					ReferenceID:   uuid.FromStringOrNil("2cb5bb08-f54c-11ee-a40b-0f5555eb875b"),
					CostPerUnit:   billing.DefaultCostPerUnitReferenceTypeSMS,
					TMBillingEnd:  dbhandler.DefaultTimeStamp,
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
				// BillingStart
				mockAccount.EXPECT().GetByCustomerID(ctx, tt.message.CustomerID).Return(tt.responseAccount, nil)
				mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDs[i])
				mockDB.EXPECT().BillingCreate(ctx, tt.expectBillings[i]).Return(nil)
				mockDB.EXPECT().BillingGet(ctx, tt.responseUUIDs[i]).Return(tt.expectBillings[i], nil)
				mockNotify.EXPECT().PublishEvent(ctx, billing.EventTypeBillingCreated, tt.expectBillings[i])

				// BillingEnd
				mockDB.EXPECT().BillingSetStatusEnd(ctx, tt.expectBillings[i].ID, float32(1), tt.expectBillings[i].TMBillingStart).Return(nil)
				mockDB.EXPECT().BillingGet(ctx, tt.expectBillings[i].ID).Return(tt.expectBillings[i], nil)
				mockAccount.EXPECT().SubtractBalance(ctx, tt.expectBillings[i].AccountID, tt.expectBillings[i].CostTotal).Return(tt.responseAccount, nil)
			}

			if err := h.EventMMMessageCreated(ctx, tt.message); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			time.Sleep(time.Millisecond * 100)
		})
	}
}

func Test_EventNMNumberCreated(t *testing.T) {

	type test struct {
		name string

		number *nmnumber.Number

		responseAccount *account.Account
		responseUUID    uuid.UUID

		expectBilling *billing.Billing
	}

	tests := []test{
		{
			name: "normal",

			number: &nmnumber.Number{
				ID:         uuid.FromStringOrNil("7359bada-f54e-11ee-ae36-37d1feaf6c4c"),
				CustomerID: uuid.FromStringOrNil("74483cdc-f54e-11ee-ac89-bb4150764799"),
			},

			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("74057276-f54e-11ee-b35b-cf292d0c7298"),
				},
			},
			responseUUID: uuid.FromStringOrNil("73c8040e-f54e-11ee-a59f-2ba1b61918fd"),

			expectBilling: &billing.Billing{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("73c8040e-f54e-11ee-a59f-2ba1b61918fd"),
				},
				AccountID:     uuid.FromStringOrNil("74057276-f54e-11ee-b35b-cf292d0c7298"),
				Status:        billing.StatusProgressing,
				ReferenceType: billing.ReferenceTypeNumber,
				ReferenceID:   uuid.FromStringOrNil("7359bada-f54e-11ee-ae36-37d1feaf6c4c"),
				CostPerUnit:   billing.DefaultCostPerUnitReferenceTypeNumber,
				TMBillingEnd:  dbhandler.DefaultTimeStamp,
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

			// BillingStart
			mockAccount.EXPECT().GetByCustomerID(ctx, tt.number.CustomerID).Return(tt.responseAccount, nil)
			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockDB.EXPECT().BillingCreate(ctx, tt.expectBilling).Return(nil)
			mockDB.EXPECT().BillingGet(ctx, tt.responseUUID).Return(tt.expectBilling, nil)
			mockNotify.EXPECT().PublishEvent(ctx, billing.EventTypeBillingCreated, tt.expectBilling)

			// BillingEnd
			mockDB.EXPECT().BillingSetStatusEnd(ctx, tt.expectBilling.ID, float32(1), tt.expectBilling.TMBillingStart).Return(nil)
			mockDB.EXPECT().BillingGet(ctx, tt.expectBilling.ID).Return(tt.expectBilling, nil)
			mockAccount.EXPECT().SubtractBalance(ctx, tt.expectBilling.AccountID, tt.expectBilling.CostTotal).Return(tt.responseAccount, nil)

			if err := h.EventNMNumberCreated(ctx, tt.number); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			time.Sleep(time.Millisecond * 100)

		})
	}
}

func Test_EventNMNumberRenewed(t *testing.T) {

	type test struct {
		name string

		number *nmnumber.Number

		responseAccount *account.Account
		responseUUID    uuid.UUID

		expectBilling *billing.Billing
	}

	tests := []test{
		{
			name: "normal",

			number: &nmnumber.Number{
				ID:         uuid.FromStringOrNil("e2eda0c8-f54e-11ee-9c57-c76cbcca2410"),
				CustomerID: uuid.FromStringOrNil("e3537aa6-f54e-11ee-84fb-bb29ab77496c"),
			},

			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("e38e5c34-f54e-11ee-9f4c-bf30ab98b5c1"),
				},
			},
			responseUUID: uuid.FromStringOrNil("e3c41b80-f54e-11ee-becf-33857841a543"),

			expectBilling: &billing.Billing{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("e3c41b80-f54e-11ee-becf-33857841a543"),
				},
				AccountID:     uuid.FromStringOrNil("e38e5c34-f54e-11ee-9f4c-bf30ab98b5c1"),
				Status:        billing.StatusProgressing,
				ReferenceType: billing.ReferenceTypeNumberRenew,
				ReferenceID:   uuid.FromStringOrNil("e2eda0c8-f54e-11ee-9c57-c76cbcca2410"),
				CostPerUnit:   billing.DefaultCostPerUnitReferenceTypeNumber,
				TMBillingEnd:  dbhandler.DefaultTimeStamp,
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

			// BillingStart
			mockAccount.EXPECT().GetByCustomerID(ctx, tt.number.CustomerID).Return(tt.responseAccount, nil)
			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockDB.EXPECT().BillingCreate(ctx, tt.expectBilling).Return(nil)
			mockDB.EXPECT().BillingGet(ctx, tt.responseUUID).Return(tt.expectBilling, nil)
			mockNotify.EXPECT().PublishEvent(ctx, billing.EventTypeBillingCreated, tt.expectBilling)

			// BillingEnd
			mockDB.EXPECT().BillingSetStatusEnd(ctx, tt.expectBilling.ID, float32(1), tt.expectBilling.TMBillingStart).Return(nil)
			mockDB.EXPECT().BillingGet(ctx, tt.expectBilling.ID).Return(tt.expectBilling, nil)
			mockAccount.EXPECT().SubtractBalance(ctx, tt.expectBilling.AccountID, tt.expectBilling.CostTotal).Return(tt.responseAccount, nil)

			if err := h.EventNMNumberRenewed(ctx, tt.number); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			time.Sleep(time.Millisecond * 100)
		})
	}
}
