package accounthandler

import (
	"context"
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	cmcustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-billing-manager/models/account"
	"monorepo/bin-billing-manager/pkg/dbhandler"
)

func Test_Create(t *testing.T) {

	type test struct {
		name string

		customerID    uuid.UUID
		accountName   string
		detail        string
		paymentType   account.PaymentType
		paymentMethod account.PaymentMethod

		responseUUID    uuid.UUID
		responseAccount *account.Account

		expectAccount *account.Account
	}

	tests := []test{
		{
			name: "normal",

			customerID:    uuid.FromStringOrNil("06011a02-08f3-11ee-b4c1-73257fafcdb3"),
			accountName:   "test name",
			detail:        "test detail",
			paymentType:   account.PaymentTypePrepaid,
			paymentMethod: account.PaymentMethodNone,

			responseUUID: uuid.FromStringOrNil("3972c20e-08f4-11ee-893b-032e633ef73a"),
			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("3972c20e-08f4-11ee-893b-032e633ef73a"),
				},
			},

			expectAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("3972c20e-08f4-11ee-893b-032e633ef73a"),
					CustomerID: uuid.FromStringOrNil("06011a02-08f3-11ee-b4c1-73257fafcdb3"),
				},
				Name:          "test name",
				Detail:        "test detail",
				Type:          account.TypeNormal,
				Balance:       0,
				PaymentType:   account.PaymentTypePrepaid,
				PaymentMethod: account.PaymentMethodNone,
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

			h := accountHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockDB.EXPECT().AccountCreate(ctx, tt.expectAccount).Return(nil)
			mockDB.EXPECT().AccountGet(ctx, tt.responseUUID).Return(tt.responseAccount, nil)
			mockNotify.EXPECT().PublishEvent(ctx, account.EventTypeAccountCreated, tt.responseAccount)

			res, err := h.Create(ctx, tt.customerID, tt.accountName, tt.detail, tt.paymentType, tt.paymentMethod)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.responseAccount, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseAccount, res)
			}
		})
	}
}

func Test_Get(t *testing.T) {

	type test struct {
		name string

		id uuid.UUID

		responseAccount *account.Account
	}

	tests := []test{
		{
			name: "normal",

			id: uuid.FromStringOrNil("8ba5c986-08f4-11ee-a292-4f01790e9d2b"),

			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("8ba5c986-08f4-11ee-a292-4f01790e9d2b"),
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

			h := accountHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().AccountGet(ctx, tt.id).Return(tt.responseAccount, nil)

			res, err := h.Get(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.responseAccount, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseAccount, res)
			}
		})
	}
}

func Test_GetByCustomerID(t *testing.T) {

	type test struct {
		name string

		customerID uuid.UUID

		responseCustomer *cmcustomer.Customer
		responseAccount  *account.Account
	}

	tests := []test{
		{
			name: "normal",

			customerID: uuid.FromStringOrNil("b7211fa2-08f4-11ee-8f49-17448c4b2951"),

			responseCustomer: &cmcustomer.Customer{
				ID:               uuid.FromStringOrNil("b7211fa2-08f4-11ee-8f49-17448c4b2951"),
				BillingAccountID: uuid.FromStringOrNil("b74fe472-08f4-11ee-8dbb-8bec316a131c"),
			},
			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("b74fe472-08f4-11ee-8dbb-8bec316a131c"),
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
			mockReq := requesthandler.NewMockRequestHandler(mc)

			h := accountHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockReq.EXPECT().CustomerV1CustomerGet(ctx, tt.customerID).Return(tt.responseCustomer, nil)
			mockDB.EXPECT().AccountGet(ctx, tt.responseCustomer.BillingAccountID).Return(tt.responseAccount, nil)

			res, err := h.GetByCustomerID(ctx, tt.customerID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.responseAccount, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseAccount, res)
			}
		})
	}
}

func Test_GetsByCustomerID(t *testing.T) {

	type test struct {
		name string

		size    uint64
		token   string
		filters map[account.Field]any

		responseAccounts []*account.Account
	}

	tests := []test{
		{
			name: "normal",

			size:  10,
			token: "2023-06-07 03:22:17.995000",
			filters: map[account.Field]any{
				account.FieldCustomerID: uuid.FromStringOrNil("480cd15e-f3d8-11ee-8212-bfff8eb203cc"),
			},

			responseAccounts: []*account.Account{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("ba0e7152-0b96-11ee-9bbb-ef5ba49d06fb"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("ba3c016c-0b96-11ee-9850-53f4edbab1be"),
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

			h := accountHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().AccountList(ctx, tt.size, tt.token, tt.filters).Return(tt.responseAccounts, nil)

			res, err := h.List(ctx, tt.size, tt.token, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.responseAccounts, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseAccounts, res)
			}
		})
	}
}

func Test_SubtractBalance(t *testing.T) {

	type test struct {
		name string

		accountID uuid.UUID
		balance   float32

		responseAccount *account.Account
	}

	tests := []test{
		{
			name: "normal",

			accountID: uuid.FromStringOrNil("287f9762-09f6-11ee-a305-2f57be32e59c"),
			balance:   20.1,

			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("287f9762-09f6-11ee-a305-2f57be32e59c"),
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

			h := accountHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().AccountSubtractBalance(ctx, tt.accountID, tt.balance).Return(nil)
			mockDB.EXPECT().AccountGet(ctx, tt.accountID).Return(tt.responseAccount, nil)

			res, err := h.SubtractBalance(ctx, tt.accountID, tt.balance)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.responseAccount, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseAccount, res)
			}
		})
	}
}

func Test_AddBalance(t *testing.T) {

	type test struct {
		name string

		accountID uuid.UUID
		balance   float32

		responseAccount *account.Account
	}

	tests := []test{
		{
			name: "normal",

			accountID: uuid.FromStringOrNil("28c90ab4-09f6-11ee-97f3-6b2314b71a97"),
			balance:   20.1,

			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("f3f570d8-09f6-11ee-a665-fb550b87c0f6"),
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

			h := accountHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().AccountAddBalance(ctx, tt.accountID, tt.balance).Return(nil)
			mockDB.EXPECT().AccountGet(ctx, tt.accountID).Return(tt.responseAccount, nil)

			res, err := h.AddBalance(ctx, tt.accountID, tt.balance)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.responseAccount, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseAccount, res)
			}
		})
	}
}

func Test_Delete(t *testing.T) {

	type test struct {
		name string

		accountID uuid.UUID

		responseAccount *account.Account
	}

	tests := []test{
		{
			name: "normal",

			accountID: uuid.FromStringOrNil("920b8062-0e68-11ee-b337-7315804888d0"),

			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("920b8062-0e68-11ee-b337-7315804888d0"),
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

			h := accountHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().AccountDelete(ctx, tt.accountID).Return(nil)
			mockDB.EXPECT().AccountGet(ctx, tt.accountID).Return(tt.responseAccount, nil)

			res, err := h.Delete(ctx, tt.accountID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.responseAccount, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseAccount, res)
			}
		})
	}
}

func Test_dbUpdateBasicInfo(t *testing.T) {

	type test struct {
		name string

		id          uuid.UUID
		accountName string
		detail      string

		expectFields     map[account.Field]any
		responseAccounts *account.Account
	}

	tests := []test{
		{
			name: "normal",

			id:          uuid.FromStringOrNil("5fc8b882-4cce-11ee-a6be-2b19227e7197"),
			accountName: "update name",
			detail:      "update detail",

			expectFields: map[account.Field]any{
				account.FieldName:   "update name",
				account.FieldDetail: "update detail",
			},
			responseAccounts: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("5fc8b882-4cce-11ee-a6be-2b19227e7197"),
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

			h := accountHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().AccountUpdate(ctx, tt.id, tt.expectFields).Return(nil)
			mockDB.EXPECT().AccountGet(ctx, tt.id).Return(tt.responseAccounts, nil)

			res, err := h.dbUpdateBasicInfo(ctx, tt.id, tt.accountName, tt.detail)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.responseAccounts, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseAccounts, res)
			}
		})
	}
}
