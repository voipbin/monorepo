package billinghandler

import (
	"context"
	reflect "reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-billing-manager/models/billing"
	"monorepo/bin-billing-manager/pkg/accounthandler"
	"monorepo/bin-billing-manager/pkg/dbhandler"
)

func Test_Create(t *testing.T) {

	type test struct {
		name string

		customerID     uuid.UUID
		accountID      uuid.UUID
		referenceType  billing.ReferenceType
		referenceID    uuid.UUID
		costPerUnit    float32
		tmBillingStart string

		responseUUID    uuid.UUID
		responseBilling *billing.Billing

		expectBilling *billing.Billing
	}

	tests := []test{
		{
			name: "normal",

			customerID:     uuid.FromStringOrNil("9727c0a0-08fb-11ee-b990-6ba2967f21c4"),
			accountID:      uuid.FromStringOrNil("975d1a8e-08fb-11ee-abea-539ff7bc4054"),
			referenceType:  billing.ReferenceTypeCall,
			referenceID:    uuid.FromStringOrNil("978512dc-08fb-11ee-953a-cb7160fb1372"),
			costPerUnit:    billing.DefaultCostPerUnitReferenceTypeCall,
			tmBillingStart: "2023-06-08 03:22:17.995000",

			responseUUID: uuid.FromStringOrNil("97a8cf42-08fb-11ee-a352-8fbcbed34869"),
			responseBilling: &billing.Billing{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("97a8cf42-08fb-11ee-a352-8fbcbed34869"),
				},
			},

			expectBilling: &billing.Billing{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("97a8cf42-08fb-11ee-a352-8fbcbed34869"),
					CustomerID: uuid.FromStringOrNil("9727c0a0-08fb-11ee-b990-6ba2967f21c4"),
				},
				AccountID:      uuid.FromStringOrNil("975d1a8e-08fb-11ee-abea-539ff7bc4054"),
				Status:         billing.StatusProgressing,
				ReferenceType:  billing.ReferenceTypeCall,
				ReferenceID:    uuid.FromStringOrNil("978512dc-08fb-11ee-953a-cb7160fb1372"),
				CostPerUnit:    billing.DefaultCostPerUnitReferenceTypeCall,
				CostTotal:      0,
				TMBillingStart: "2023-06-08 03:22:17.995000",
				TMBillingEnd:   dbhandler.DefaultTimeStamp,
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

			h := billingHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockDB.EXPECT().BillingCreate(ctx, tt.expectBilling).Return(nil)
			mockDB.EXPECT().BillingGet(ctx, tt.responseUUID).Return(tt.responseBilling, nil)

			mockNotify.EXPECT().PublishEvent(ctx, billing.EventTypeBillingCreated, tt.responseBilling)

			res, err := h.Create(ctx, tt.customerID, tt.accountID, tt.referenceType, tt.referenceID, tt.costPerUnit, tt.tmBillingStart)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.responseBilling, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseBilling, res)
			}
		})
	}
}

func Test_Get(t *testing.T) {

	type test struct {
		name string

		id uuid.UUID

		responseBilling *billing.Billing
	}

	tests := []test{
		{
			name: "normal",

			id: uuid.FromStringOrNil("02be7194-08ff-11ee-a093-e33795e9c217"),

			responseBilling: &billing.Billing{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("02be7194-08ff-11ee-a093-e33795e9c217"),
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
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,

				accountHandler: mockAccount,
			}
			ctx := context.Background()

			mockDB.EXPECT().BillingGet(ctx, tt.id).Return(tt.responseBilling, nil)

			res, err := h.Get(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.responseBilling, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseBilling, res)
			}
		})
	}
}

func Test_GetByReferenceID(t *testing.T) {

	type test struct {
		name string

		referenceID uuid.UUID

		responseBilling *billing.Billing
	}

	tests := []test{
		{
			name: "normal",

			referenceID: uuid.FromStringOrNil("627a9144-08ff-11ee-ab4e-37938da304ad"),

			responseBilling: &billing.Billing{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("62a020a8-08ff-11ee-bbc8-47c7ce23b6bb"),
				},
				ReferenceType: billing.ReferenceTypeCall,
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
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,

				accountHandler: mockAccount,
			}
			ctx := context.Background()

			mockDB.EXPECT().BillingGetByReferenceID(ctx, tt.referenceID).Return(tt.responseBilling, nil)

			res, err := h.GetByReferenceID(ctx, tt.referenceID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.responseBilling, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseBilling, res)
			}
		})
	}
}

func Test_Gets(t *testing.T) {

	type test struct {
		name string

		size    uint64
		token   string
		filters map[string]string

		responseBillings []*billing.Billing
	}

	tests := []test{
		{
			name: "normal",

			size:  10,
			token: "2023-06-08 03:22:17.995000",
			filters: map[string]string{
				"customer_id": "bd5b3ae6-08ff-11ee-8101-1396e6f3622a",
			},

			responseBillings: []*billing.Billing{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("bd9346e8-08ff-11ee-b6dd-cbff408887a9"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("bdcb268a-08ff-11ee-a67f-d307488a3fe3"),
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
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,

				accountHandler: mockAccount,
			}
			ctx := context.Background()

			mockDB.EXPECT().BillingGets(ctx, tt.size, tt.token, tt.filters).Return(tt.responseBillings, nil)

			res, err := h.Gets(ctx, tt.size, tt.token, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.responseBillings, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseBillings, res)
			}
		})
	}
}

func Test_UpdateStatusEnd(t *testing.T) {

	type test struct {
		name string

		id              uuid.UUID
		billingDuration float32
		tmBillingEnd    string

		responseBilling *billing.Billing
	}

	tests := []test{
		{
			name: "normal",

			id:              uuid.FromStringOrNil("a33c2c6e-0900-11ee-b83b-5f7796e6df8a"),
			billingDuration: 10.32,
			tmBillingEnd:    "2023-06-09 03:22:17.995000",

			responseBilling: &billing.Billing{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a33c2c6e-0900-11ee-b83b-5f7796e6df8a"),
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
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,

				accountHandler: mockAccount,
			}
			ctx := context.Background()

			mockDB.EXPECT().BillingSetStatusEnd(ctx, tt.id, tt.billingDuration, tt.tmBillingEnd).Return(nil)
			mockDB.EXPECT().BillingGet(ctx, tt.id).Return(tt.responseBilling, nil)

			res, err := h.UpdateStatusEnd(ctx, tt.id, tt.billingDuration, tt.tmBillingEnd)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.responseBilling, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseBilling, res)
			}
		})
	}
}
