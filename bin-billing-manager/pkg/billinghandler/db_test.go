package billinghandler

import (
	"context"
	"fmt"
	reflect "reflect"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-billing-manager/models/billing"
	"monorepo/bin-billing-manager/pkg/accounthandler"
	"monorepo/bin-billing-manager/pkg/allowancehandler"
	"monorepo/bin-billing-manager/pkg/dbhandler"
)

func Test_Create(t *testing.T) {

	type test struct {
		name string

		customerID     uuid.UUID
		accountID      uuid.UUID
		referenceType  billing.ReferenceType
		referenceID    uuid.UUID
		costType       billing.CostType
		tmBillingStart *time.Time

		responseUUID    uuid.UUID
		responseBilling *billing.Billing

		expectBilling *billing.Billing
	}

	tmBillingStart := time.Date(2023, 6, 8, 3, 22, 17, 995000000, time.UTC)

	tests := []test{
		{
			name: "normal",

			customerID:     uuid.FromStringOrNil("9727c0a0-08fb-11ee-b990-6ba2967f21c4"),
			accountID:      uuid.FromStringOrNil("975d1a8e-08fb-11ee-abea-539ff7bc4054"),
			referenceType:  billing.ReferenceTypeCall,
			referenceID:    uuid.FromStringOrNil("978512dc-08fb-11ee-953a-cb7160fb1372"),
			costType:       billing.CostTypeCallPSTNOutgoing,
			tmBillingStart: &tmBillingStart,

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
				AccountID:         uuid.FromStringOrNil("975d1a8e-08fb-11ee-abea-539ff7bc4054"),
				Status:            billing.StatusProgressing,
				ReferenceType:     billing.ReferenceTypeCall,
				ReferenceID:       uuid.FromStringOrNil("978512dc-08fb-11ee-953a-cb7160fb1372"),
				CostType:          billing.CostTypeCallPSTNOutgoing,
				CostCreditPerUnit: billing.DefaultCreditPerUnitCallPSTNOutgoing,
				CostCreditTotal:   0,
				CostTokenPerUnit:  0,
				CostTokenTotal:    0,
				CostUnitCount:     0,
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

			res, err := h.Create(ctx, tt.customerID, tt.accountID, tt.referenceType, tt.referenceID, tt.costType, tt.tmBillingStart)
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

func Test_List(t *testing.T) {

	type test struct {
		name string

		size    uint64
		token   string
		filters map[billing.Field]any

		responseBillings []*billing.Billing
	}

	tests := []test{
		{
			name: "normal",

			size:  10,
			token: "2023-06-08T03:22:17.995000Z",
			filters: map[billing.Field]any{
				billing.FieldCustomerID: uuid.FromStringOrNil("bd5b3ae6-08ff-11ee-8101-1396e6f3622a"),
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

			mockDB.EXPECT().BillingList(ctx, tt.size, tt.token, tt.filters).Return(tt.responseBillings, nil)

			res, err := h.List(ctx, tt.size, tt.token, tt.filters)
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
		costUnitCount   float32
		costTokenTotal  int
		costCreditTotal float32
		tmBillingEnd    *time.Time

		responseBilling *billing.Billing
	}

	tmBillingEnd := time.Date(2023, 6, 9, 3, 22, 17, 995000000, time.UTC)

	tests := []test{
		{
			name: "normal",

			id:              uuid.FromStringOrNil("a33c2c6e-0900-11ee-b83b-5f7796e6df8a"),
			costUnitCount:   10.32,
			costTokenTotal:  0,
			costCreditTotal: 0.06192,
			tmBillingEnd:    &tmBillingEnd,

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
			mockAllowance := allowancehandler.NewMockAllowanceHandler(mc)

			h := billingHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,

				accountHandler:   mockAccount,
				allowanceHandler: mockAllowance,
			}
			ctx := context.Background()

			mockDB.EXPECT().BillingSetStatusEndWithCosts(ctx, tt.id, tt.costUnitCount, tt.costTokenTotal, tt.costCreditTotal, tt.tmBillingEnd).Return(nil)
			mockDB.EXPECT().BillingGet(ctx, tt.id).Return(tt.responseBilling, nil)

			res, err := h.UpdateStatusEnd(ctx, tt.id, tt.costUnitCount, tt.costTokenTotal, tt.costCreditTotal, tt.tmBillingEnd)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.responseBilling, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseBilling, res)
			}
		})
	}
}

func Test_Create_db_create_error(t *testing.T) {

	type test struct {
		name string

		customerID     uuid.UUID
		accountID      uuid.UUID
		referenceType  billing.ReferenceType
		referenceID    uuid.UUID
		costType       billing.CostType
		tmBillingStart *time.Time

		responseUUID uuid.UUID

		expectBilling *billing.Billing
	}

	tmBillingStart := time.Date(2023, 6, 8, 3, 22, 17, 995000000, time.UTC)

	tests := []test{
		{
			name: "db create error",

			customerID:     uuid.FromStringOrNil("11111111-0000-0000-0000-000000000001"),
			accountID:      uuid.FromStringOrNil("22222222-0000-0000-0000-000000000001"),
			referenceType:  billing.ReferenceTypeCall,
			referenceID:    uuid.FromStringOrNil("33333333-0000-0000-0000-000000000001"),
			costType:       billing.CostTypeCallPSTNOutgoing,
			tmBillingStart: &tmBillingStart,

			responseUUID: uuid.FromStringOrNil("44444444-0000-0000-0000-000000000001"),

			expectBilling: &billing.Billing{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("44444444-0000-0000-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("11111111-0000-0000-0000-000000000001"),
				},
				AccountID:         uuid.FromStringOrNil("22222222-0000-0000-0000-000000000001"),
				Status:            billing.StatusProgressing,
				ReferenceType:     billing.ReferenceTypeCall,
				ReferenceID:       uuid.FromStringOrNil("33333333-0000-0000-0000-000000000001"),
				CostType:          billing.CostTypeCallPSTNOutgoing,
				CostCreditPerUnit: billing.DefaultCreditPerUnitCallPSTNOutgoing,
				CostCreditTotal:   0,
				CostTokenPerUnit:  0,
				CostTokenTotal:    0,
				CostUnitCount:     0,
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

			h := billingHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockDB.EXPECT().BillingCreate(ctx, tt.expectBilling).Return(fmt.Errorf("db connection lost"))

			_, err := h.Create(ctx, tt.customerID, tt.accountID, tt.referenceType, tt.referenceID, tt.costType, tt.tmBillingStart)
			if err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			}
		})
	}
}

func Test_Create_db_get_error(t *testing.T) {

	type test struct {
		name string

		customerID     uuid.UUID
		accountID      uuid.UUID
		referenceType  billing.ReferenceType
		referenceID    uuid.UUID
		costType       billing.CostType
		tmBillingStart *time.Time

		responseUUID uuid.UUID

		expectBilling *billing.Billing
	}

	tmBillingStart := time.Date(2023, 6, 8, 3, 22, 17, 995000000, time.UTC)

	tests := []test{
		{
			name: "db get error after create",

			customerID:     uuid.FromStringOrNil("55555555-0000-0000-0000-000000000001"),
			accountID:      uuid.FromStringOrNil("66666666-0000-0000-0000-000000000001"),
			referenceType:  billing.ReferenceTypeCall,
			referenceID:    uuid.FromStringOrNil("77777777-0000-0000-0000-000000000001"),
			costType:       billing.CostTypeCallPSTNOutgoing,
			tmBillingStart: &tmBillingStart,

			responseUUID: uuid.FromStringOrNil("88888888-0000-0000-0000-000000000001"),

			expectBilling: &billing.Billing{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("88888888-0000-0000-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("55555555-0000-0000-0000-000000000001"),
				},
				AccountID:         uuid.FromStringOrNil("66666666-0000-0000-0000-000000000001"),
				Status:            billing.StatusProgressing,
				ReferenceType:     billing.ReferenceTypeCall,
				ReferenceID:       uuid.FromStringOrNil("77777777-0000-0000-0000-000000000001"),
				CostType:          billing.CostTypeCallPSTNOutgoing,
				CostCreditPerUnit: billing.DefaultCreditPerUnitCallPSTNOutgoing,
				CostCreditTotal:   0,
				CostTokenPerUnit:  0,
				CostTokenTotal:    0,
				CostUnitCount:     0,
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

			h := billingHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockDB.EXPECT().BillingCreate(ctx, tt.expectBilling).Return(nil)
			mockDB.EXPECT().BillingGet(ctx, tt.responseUUID).Return(nil, fmt.Errorf("connection timeout"))

			_, err := h.Create(ctx, tt.customerID, tt.accountID, tt.referenceType, tt.referenceID, tt.costType, tt.tmBillingStart)
			if err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			}
		})
	}
}

func Test_Get_error(t *testing.T) {

	type test struct {
		name string

		id uuid.UUID
	}

	tests := []test{
		{
			name: "db error",

			id: uuid.FromStringOrNil("99999999-0000-0000-0000-000000000001"),
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

			mockDB.EXPECT().BillingGet(ctx, tt.id).Return(nil, fmt.Errorf("connection timeout"))

			_, err := h.Get(ctx, tt.id)
			if err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			}
		})
	}
}

func Test_GetByReferenceID_error(t *testing.T) {

	type test struct {
		name string

		referenceID uuid.UUID

		responseBilling *billing.Billing
	}

	tests := []test{
		{
			name: "db error",

			referenceID: uuid.FromStringOrNil("aaaaaaaa-0000-0000-0000-000000000001"),
		},
		{
			name: "wrong reference type",

			referenceID: uuid.FromStringOrNil("bbbbbbbb-0000-0000-0000-000000000001"),

			responseBilling: &billing.Billing{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cccccccc-0000-0000-0000-000000000001"),
				},
				ReferenceType: billing.ReferenceTypeSMS,
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

			if tt.responseBilling == nil {
				mockDB.EXPECT().BillingGetByReferenceID(ctx, tt.referenceID).Return(nil, fmt.Errorf("connection timeout"))
			} else {
				mockDB.EXPECT().BillingGetByReferenceID(ctx, tt.referenceID).Return(tt.responseBilling, nil)
			}

			_, err := h.GetByReferenceID(ctx, tt.referenceID)
			if err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			}
		})
	}
}

func Test_List_error(t *testing.T) {

	type test struct {
		name string

		size    uint64
		token   string
		filters map[billing.Field]any
	}

	tests := []test{
		{
			name: "db error",

			size:  10,
			token: "2023-06-08T03:22:17.995000Z",
			filters: map[billing.Field]any{
				billing.FieldCustomerID: uuid.FromStringOrNil("dddddddd-0000-0000-0000-000000000001"),
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

			mockDB.EXPECT().BillingList(ctx, tt.size, tt.token, tt.filters).Return(nil, fmt.Errorf("connection timeout"))

			_, err := h.List(ctx, tt.size, tt.token, tt.filters)
			if err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			}
		})
	}
}

func Test_UpdateStatusEnd_set_error(t *testing.T) {

	type test struct {
		name string

		id              uuid.UUID
		costUnitCount   float32
		costTokenTotal  int
		costCreditTotal float32
		tmBillingEnd    *time.Time
	}

	tmBillingEnd := time.Date(2023, 6, 9, 3, 22, 17, 995000000, time.UTC)

	tests := []test{
		{
			name: "set status error",

			id:              uuid.FromStringOrNil("eeeeeeee-0000-0000-0000-000000000001"),
			costUnitCount:   10.32,
			costTokenTotal:  0,
			costCreditTotal: 0.06192,
			tmBillingEnd:    &tmBillingEnd,
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
			mockAllowance := allowancehandler.NewMockAllowanceHandler(mc)

			h := billingHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,

				accountHandler:   mockAccount,
				allowanceHandler: mockAllowance,
			}
			ctx := context.Background()

			mockDB.EXPECT().BillingSetStatusEndWithCosts(ctx, tt.id, tt.costUnitCount, tt.costTokenTotal, tt.costCreditTotal, tt.tmBillingEnd).Return(fmt.Errorf("connection timeout"))

			_, err := h.UpdateStatusEnd(ctx, tt.id, tt.costUnitCount, tt.costTokenTotal, tt.costCreditTotal, tt.tmBillingEnd)
			if err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			}
		})
	}
}

func Test_UpdateStatusEnd_get_error(t *testing.T) {

	type test struct {
		name string

		id              uuid.UUID
		costUnitCount   float32
		costTokenTotal  int
		costCreditTotal float32
		tmBillingEnd    *time.Time
	}

	tmBillingEnd := time.Date(2023, 6, 9, 3, 22, 17, 995000000, time.UTC)

	tests := []test{
		{
			name: "get error after set",

			id:              uuid.FromStringOrNil("ffffffff-0000-0000-0000-000000000001"),
			costUnitCount:   10.32,
			costTokenTotal:  0,
			costCreditTotal: 0.06192,
			tmBillingEnd:    &tmBillingEnd,
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
			mockAllowance := allowancehandler.NewMockAllowanceHandler(mc)

			h := billingHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,

				accountHandler:   mockAccount,
				allowanceHandler: mockAllowance,
			}
			ctx := context.Background()

			mockDB.EXPECT().BillingSetStatusEndWithCosts(ctx, tt.id, tt.costUnitCount, tt.costTokenTotal, tt.costCreditTotal, tt.tmBillingEnd).Return(nil)
			mockDB.EXPECT().BillingGet(ctx, tt.id).Return(nil, fmt.Errorf("connection timeout"))

			_, err := h.UpdateStatusEnd(ctx, tt.id, tt.costUnitCount, tt.costTokenTotal, tt.costCreditTotal, tt.tmBillingEnd)
			if err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			}
		})
	}
}
