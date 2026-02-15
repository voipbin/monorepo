package accounthandler

import (
	"context"
	"fmt"
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

func Test_ListByCustomerID(t *testing.T) {

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
			token: "2023-06-07T03:22:17.995000Z",
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
		balance   int64

		responseAccount *account.Account
	}

	tests := []test{
		{
			name: "normal",

			accountID: uuid.FromStringOrNil("287f9762-09f6-11ee-a305-2f57be32e59c"),
			balance:   20100000,

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
		balance   int64

		responseAccount *account.Account
	}

	tests := []test{
		{
			name: "normal",

			accountID: uuid.FromStringOrNil("28c90ab4-09f6-11ee-97f3-6b2314b71a97"),
			balance:   20100000,

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

func Test_SubtractBalanceWithCheck_normal(t *testing.T) {

	tests := []struct {
		name string

		accountID uuid.UUID
		amount    int64

		responseAccount        *account.Account
		responseUpdatedAccount *account.Account
	}{
		{
			name: "normal account uses atomic check",

			accountID: uuid.FromStringOrNil("bb111111-0000-0000-0000-000000000001"),
			amount:    15500000,

			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("bb111111-0000-0000-0000-000000000001"),
				},
				PlanType:      account.PlanTypeFree,
				BalanceCredit: 100000000,
			},
			responseUpdatedAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("bb111111-0000-0000-0000-000000000001"),
				},
				PlanType:      account.PlanTypeFree,
				BalanceCredit: 84500000,
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

			// get account to check type
			mockDB.EXPECT().AccountGet(ctx, tt.accountID).Return(tt.responseAccount, nil)
			// atomic check-and-subtract
			mockDB.EXPECT().AccountSubtractBalanceWithCheck(ctx, tt.accountID, tt.amount).Return(nil)
			// get updated account
			mockDB.EXPECT().AccountGet(ctx, tt.accountID).Return(tt.responseUpdatedAccount, nil)

			res, err := h.SubtractBalanceWithCheck(ctx, tt.accountID, tt.amount)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.responseUpdatedAccount, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseUpdatedAccount, res)
			}
		})
	}
}

func Test_SubtractBalanceWithCheck_unlimited(t *testing.T) {

	tests := []struct {
		name string

		accountID uuid.UUID
		amount    int64

		responseAccount        *account.Account
		responseUpdatedAccount *account.Account
	}{
		{
			name: "unlimited plan account bypasses check",

			accountID: uuid.FromStringOrNil("cc111111-0000-0000-0000-000000000001"),
			amount:    50000000,

			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cc111111-0000-0000-0000-000000000001"),
				},
				PlanType:      account.PlanTypeUnlimited,
				BalanceCredit: 10000000,
			},
			responseUpdatedAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cc111111-0000-0000-0000-000000000001"),
				},
				PlanType:      account.PlanTypeUnlimited,
				BalanceCredit: -40000000,
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

			// get account to check plan type — unlimited
			mockDB.EXPECT().AccountGet(ctx, tt.accountID).Return(tt.responseAccount, nil)
			// unlimited bypasses to SubtractBalance (no check)
			mockDB.EXPECT().AccountSubtractBalance(ctx, tt.accountID, tt.amount).Return(nil)
			// get updated account
			mockDB.EXPECT().AccountGet(ctx, tt.accountID).Return(tt.responseUpdatedAccount, nil)

			res, err := h.SubtractBalanceWithCheck(ctx, tt.accountID, tt.amount)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.responseUpdatedAccount, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseUpdatedAccount, res)
			}
		})
	}
}

func Test_SubtractBalanceWithCheck_insufficient(t *testing.T) {
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

	accountID := uuid.FromStringOrNil("dd111111-0000-0000-0000-000000000001")

	mockDB.EXPECT().AccountGet(ctx, accountID).Return(&account.Account{
		Identity:      commonidentity.Identity{ID: accountID},
		PlanType:      account.PlanTypeFree,
		BalanceCredit: 5000000,
	}, nil)
	mockDB.EXPECT().AccountSubtractBalanceWithCheck(ctx, accountID, int64(50000000)).Return(fmt.Errorf("insufficient balance"))

	_, err := h.SubtractBalanceWithCheck(ctx, accountID, 50000000)
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
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

func Test_dbUpdatePaymentInfo(t *testing.T) {

	type test struct {
		name string

		id            uuid.UUID
		paymentType   account.PaymentType
		paymentMethod account.PaymentMethod

		expectFields     map[account.Field]any
		responseAccounts *account.Account
	}

	tests := []test{
		{
			name: "normal",

			id:            uuid.FromStringOrNil("6a1b2c3d-5e6f-11ee-a7be-3c29338e8297"),
			paymentType:   account.PaymentTypePrepaid,
			paymentMethod: account.PaymentMethodCreditCard,

			expectFields: map[account.Field]any{
				account.FieldPaymentType:   account.PaymentTypePrepaid,
				account.FieldPaymentMethod: account.PaymentMethodCreditCard,
			},
			responseAccounts: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("6a1b2c3d-5e6f-11ee-a7be-3c29338e8297"),
				},
			},
		},
		{
			name: "db update error",

			id:            uuid.FromStringOrNil("7b2c3d4e-6f70-11ee-b8cf-4d3a449f9308"),
			paymentType:   account.PaymentTypePrepaid,
			paymentMethod: account.PaymentMethodCreditCard,

			expectFields: map[account.Field]any{
				account.FieldPaymentType:   account.PaymentTypePrepaid,
				account.FieldPaymentMethod: account.PaymentMethodCreditCard,
			},
			responseAccounts: nil,
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

			if tt.name == "db update error" {
				mockDB.EXPECT().AccountUpdate(ctx, tt.id, tt.expectFields).Return(fmt.Errorf("update failed"))

				_, err := h.dbUpdatePaymentInfo(ctx, tt.id, tt.paymentType, tt.paymentMethod)
				if err == nil {
					t.Errorf("Wrong match. expect: error, got: nil")
				}
				return
			}

			mockDB.EXPECT().AccountUpdate(ctx, tt.id, tt.expectFields).Return(nil)
			mockDB.EXPECT().AccountGet(ctx, tt.id).Return(tt.responseAccounts, nil)

			res, err := h.dbUpdatePaymentInfo(ctx, tt.id, tt.paymentType, tt.paymentMethod)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.responseAccounts, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseAccounts, res)
			}
		})
	}
}

func Test_dbUpdatePaymentInfo_get_error(t *testing.T) {
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

	id := uuid.FromStringOrNil("8c3d4e5f-7081-11ee-c9d0-5e4b55a0a419")
	fields := map[account.Field]any{
		account.FieldPaymentType:   account.PaymentTypePrepaid,
		account.FieldPaymentMethod: account.PaymentMethodCreditCard,
	}

	mockDB.EXPECT().AccountUpdate(ctx, id, fields).Return(nil)
	mockDB.EXPECT().AccountGet(ctx, id).Return(nil, fmt.Errorf("get failed"))

	_, err := h.dbUpdatePaymentInfo(ctx, id, account.PaymentTypePrepaid, account.PaymentMethodCreditCard)
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
}

func Test_dbUpdatePlanType(t *testing.T) {

	type test struct {
		name string

		id       uuid.UUID
		planType account.PlanType

		expectFields     map[account.Field]any
		responseAccounts *account.Account
	}

	tests := []test{
		{
			name: "normal - free to basic",

			id:       uuid.FromStringOrNil("9d4e5f60-8192-11ee-dae1-6f5c66b1b520"),
			planType: account.PlanTypeBasic,

			expectFields: map[account.Field]any{
				account.FieldPlanType: account.PlanTypeBasic,
			},
			responseAccounts: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("9d4e5f60-8192-11ee-dae1-6f5c66b1b520"),
				},
			},
		},
		{
			name: "normal - basic to professional",

			id:       uuid.FromStringOrNil("ae5f6071-92a3-11ee-ebf2-7066776c631"),
			planType: account.PlanTypeProfessional,

			expectFields: map[account.Field]any{
				account.FieldPlanType: account.PlanTypeProfessional,
			},
			responseAccounts: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("ae5f6071-92a3-11ee-ebf2-7066776c631"),
				},
			},
		},
		{
			name: "db update error",

			id:       uuid.FromStringOrNil("bf607182-a3b4-11ee-fc03-8177887d742"),
			planType: account.PlanTypeUnlimited,

			expectFields: map[account.Field]any{
				account.FieldPlanType: account.PlanTypeUnlimited,
			},
			responseAccounts: nil,
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

			if tt.name == "db update error" {
				mockDB.EXPECT().AccountUpdate(ctx, tt.id, tt.expectFields).Return(fmt.Errorf("update failed"))

				_, err := h.dbUpdatePlanType(ctx, tt.id, tt.planType)
				if err == nil {
					t.Errorf("Wrong match. expect: error, got: nil")
				}
				return
			}

			mockDB.EXPECT().AccountUpdate(ctx, tt.id, tt.expectFields).Return(nil)
			mockDB.EXPECT().AccountGet(ctx, tt.id).Return(tt.responseAccounts, nil)

			res, err := h.dbUpdatePlanType(ctx, tt.id, tt.planType)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.responseAccounts, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseAccounts, res)
			}
		})
	}
}

func Test_dbUpdatePlanType_get_error(t *testing.T) {
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

	id := uuid.FromStringOrNil("c0718293-b4c5-11ee-0d14-9288998e853")
	fields := map[account.Field]any{
		account.FieldPlanType: account.PlanTypeBasic,
	}

	mockDB.EXPECT().AccountUpdate(ctx, id, fields).Return(nil)
	mockDB.EXPECT().AccountGet(ctx, id).Return(nil, fmt.Errorf("get failed"))

	_, err := h.dbUpdatePlanType(ctx, id, account.PlanTypeBasic)
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
}

func Test_Get_error(t *testing.T) {
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

	id := uuid.FromStringOrNil("d18293a4-c5d6-11ee-1e25-a399aa9f964")

	mockDB.EXPECT().AccountGet(ctx, id).Return(nil, fmt.Errorf("connection timeout"))

	_, err := h.Get(ctx, id)
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
}

func Test_GetByCustomerID_error(t *testing.T) {

	type test struct {
		name string

		customerID uuid.UUID

		responseCustomer *cmcustomer.Customer
		responseAccount  *account.Account
	}

	tests := []test{
		{
			name: "customer not found",

			customerID: uuid.FromStringOrNil("e293a4b5-d6e7-11ee-2f36-b4aabbe0a75"),

			responseCustomer: nil,
			responseAccount:  nil,
		},
		{
			name: "account not found",

			customerID: uuid.FromStringOrNil("f3a4b5c6-e7f8-11ee-3047-c5bbccf1b86"),

			responseCustomer: &cmcustomer.Customer{
				ID:               uuid.FromStringOrNil("f3a4b5c6-e7f8-11ee-3047-c5bbccf1b86"),
				BillingAccountID: uuid.FromStringOrNil("04b5c6d7-f809-11ee-4158-d6ccdde2c97"),
			},
			responseAccount: nil,
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

			if tt.name == "customer not found" {
				mockReq.EXPECT().CustomerV1CustomerGet(ctx, tt.customerID).Return(nil, fmt.Errorf("customer not found"))

				_, err := h.GetByCustomerID(ctx, tt.customerID)
				if err == nil {
					t.Errorf("Wrong match. expect: error, got: nil")
				}
				return
			}

			if tt.name == "account not found" {
				mockReq.EXPECT().CustomerV1CustomerGet(ctx, tt.customerID).Return(tt.responseCustomer, nil)
				mockDB.EXPECT().AccountGet(ctx, tt.responseCustomer.BillingAccountID).Return(nil, fmt.Errorf("account not found"))

				_, err := h.GetByCustomerID(ctx, tt.customerID)
				if err == nil {
					t.Errorf("Wrong match. expect: error, got: nil")
				}
			}
		})
	}
}

func Test_List_error(t *testing.T) {
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

	size := uint64(10)
	token := "2023-06-07T03:22:17.995000Z"
	filters := map[account.Field]any{
		account.FieldCustomerID: uuid.FromStringOrNil("15c6d7e8-091a-11ee-5269-e7ddeeef3da8"),
	}

	mockDB.EXPECT().AccountList(ctx, size, token, filters).Return(nil, fmt.Errorf("database error"))

	_, err := h.List(ctx, size, token, filters)
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
}

func Test_SubtractBalance_error(t *testing.T) {

	type test struct {
		name string

		accountID uuid.UUID
		balance   int64
	}

	tests := []test{
		{
			name: "db subtract error",

			accountID: uuid.FromStringOrNil("26d7e8f9-1a2b-11ee-637a-f8eeff04eb9"),
			balance:   10500000,
		},
		{
			name: "db get error",

			accountID: uuid.FromStringOrNil("37e8f90a-2b3c-11ee-748b-09ff0015fca"),
			balance:   20000000,
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

			if tt.name == "db subtract error" {
				mockDB.EXPECT().AccountSubtractBalance(ctx, tt.accountID, tt.balance).Return(fmt.Errorf("subtract failed"))

				_, err := h.SubtractBalance(ctx, tt.accountID, tt.balance)
				if err == nil {
					t.Errorf("Wrong match. expect: error, got: nil")
				}
				return
			}

			if tt.name == "db get error" {
				mockDB.EXPECT().AccountSubtractBalance(ctx, tt.accountID, tt.balance).Return(nil)
				mockDB.EXPECT().AccountGet(ctx, tt.accountID).Return(nil, fmt.Errorf("get failed"))

				_, err := h.SubtractBalance(ctx, tt.accountID, tt.balance)
				if err == nil {
					t.Errorf("Wrong match. expect: error, got: nil")
				}
			}
		})
	}
}

func Test_AddBalance_error(t *testing.T) {

	type test struct {
		name string

		accountID uuid.UUID
		balance   int64
	}

	tests := []test{
		{
			name: "db add error",

			accountID: uuid.FromStringOrNil("48f90a1b-3c4d-11ee-859c-1a00112600db"),
			balance:   15500000,
		},
		{
			name: "db get error",

			accountID: uuid.FromStringOrNil("590a1b2c-4d5e-11ee-96ad-2b11223711ec"),
			balance:   25000000,
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

			if tt.name == "db add error" {
				mockDB.EXPECT().AccountAddBalance(ctx, tt.accountID, tt.balance).Return(fmt.Errorf("add failed"))

				_, err := h.AddBalance(ctx, tt.accountID, tt.balance)
				if err == nil {
					t.Errorf("Wrong match. expect: error, got: nil")
				}
				return
			}

			if tt.name == "db get error" {
				mockDB.EXPECT().AccountAddBalance(ctx, tt.accountID, tt.balance).Return(nil)
				mockDB.EXPECT().AccountGet(ctx, tt.accountID).Return(nil, fmt.Errorf("get failed"))

				_, err := h.AddBalance(ctx, tt.accountID, tt.balance)
				if err == nil {
					t.Errorf("Wrong match. expect: error, got: nil")
				}
			}
		})
	}
}

func Test_Delete_error(t *testing.T) {

	type test struct {
		name string

		accountID uuid.UUID
	}

	tests := []test{
		{
			name: "db delete error",

			accountID: uuid.FromStringOrNil("6a1b2c3d-5e6f-11ee-a7be-3c22334822fd"),
		},
		{
			name: "db get error",

			accountID: uuid.FromStringOrNil("7b2c3d4e-6f70-11ee-b8cf-4d33445933ge"),
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

			if tt.name == "db delete error" {
				mockDB.EXPECT().AccountDelete(ctx, tt.accountID).Return(fmt.Errorf("delete failed"))

				_, err := h.Delete(ctx, tt.accountID)
				if err == nil {
					t.Errorf("Wrong match. expect: error, got: nil")
				}
				return
			}

			if tt.name == "db get error" {
				mockDB.EXPECT().AccountDelete(ctx, tt.accountID).Return(nil)
				mockDB.EXPECT().AccountGet(ctx, tt.accountID).Return(nil, fmt.Errorf("get failed"))

				_, err := h.Delete(ctx, tt.accountID)
				if err == nil {
					t.Errorf("Wrong match. expect: error, got: nil")
				}
			}
		})
	}
}

func Test_Create_db_error(t *testing.T) {
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

	customerID := uuid.FromStringOrNil("8c3d4e5f-7081-11ee-c9d0-5e44556a44hf")
	responseUUID := uuid.FromStringOrNil("9d4e5f60-8192-11ee-dae1-6f55667b55ig")

	mockUtil.EXPECT().UUIDCreate().Return(responseUUID)
	mockDB.EXPECT().AccountCreate(ctx, gomock.Any()).Return(nil)
	mockDB.EXPECT().AccountGet(ctx, responseUUID).Return(nil, fmt.Errorf("get failed after create"))

	_, err := h.Create(ctx, customerID, "test", "detail", account.PaymentTypePrepaid, account.PaymentMethodNone)
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
}

func Test_AddTokens(t *testing.T) {

	type test struct {
		name string

		accountID uuid.UUID
		amount    int64

		responseAccount *account.Account
	}

	tests := []test{
		{
			name: "normal",

			accountID: uuid.FromStringOrNil("a1111111-0000-0000-0000-000000000001"),
			amount:    500,

			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a1111111-0000-0000-0000-000000000001"),
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

			mockDB.EXPECT().AccountAddTokens(ctx, tt.accountID, tt.amount).Return(nil)
			mockDB.EXPECT().AccountGet(ctx, tt.accountID).Return(tt.responseAccount, nil)

			res, err := h.AddTokens(ctx, tt.accountID, tt.amount)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.responseAccount, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseAccount, res)
			}
		})
	}
}

func Test_SubtractTokens(t *testing.T) {

	type test struct {
		name string

		accountID uuid.UUID
		amount    int64

		responseAccount *account.Account
	}

	tests := []test{
		{
			name: "normal",

			accountID: uuid.FromStringOrNil("a2222222-0000-0000-0000-000000000001"),
			amount:    300,

			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a2222222-0000-0000-0000-000000000001"),
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

			mockDB.EXPECT().AccountSubtractTokens(ctx, tt.accountID, tt.amount).Return(nil)
			mockDB.EXPECT().AccountGet(ctx, tt.accountID).Return(tt.responseAccount, nil)

			res, err := h.SubtractTokens(ctx, tt.accountID, tt.amount)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.responseAccount, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseAccount, res)
			}
		})
	}
}

func Test_SubtractTokensWithCheck_normal(t *testing.T) {

	tests := []struct {
		name string

		accountID uuid.UUID
		amount    int64

		responseAccount        *account.Account
		responseUpdatedAccount *account.Account
	}{
		{
			name: "normal account uses atomic check",

			accountID: uuid.FromStringOrNil("a3333333-0000-0000-0000-000000000001"),
			amount:    500,

			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a3333333-0000-0000-0000-000000000001"),
				},
				PlanType:     account.PlanTypeFree,
				BalanceToken: 1000,
			},
			responseUpdatedAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a3333333-0000-0000-0000-000000000001"),
				},
				PlanType:     account.PlanTypeFree,
				BalanceToken: 500,
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

			// get account to check type
			mockDB.EXPECT().AccountGet(ctx, tt.accountID).Return(tt.responseAccount, nil)
			// atomic check-and-subtract
			mockDB.EXPECT().AccountSubtractTokensWithCheck(ctx, tt.accountID, tt.amount).Return(nil)
			// get updated account
			mockDB.EXPECT().AccountGet(ctx, tt.accountID).Return(tt.responseUpdatedAccount, nil)

			res, err := h.SubtractTokensWithCheck(ctx, tt.accountID, tt.amount)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.responseUpdatedAccount, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseUpdatedAccount, res)
			}
		})
	}
}

func Test_SubtractTokensWithCheck_unlimited(t *testing.T) {

	tests := []struct {
		name string

		accountID uuid.UUID
		amount    int64

		responseAccount        *account.Account
		responseUpdatedAccount *account.Account
	}{
		{
			name: "unlimited plan account bypasses check",

			accountID: uuid.FromStringOrNil("a4444444-0000-0000-0000-000000000001"),
			amount:    2000,

			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a4444444-0000-0000-0000-000000000001"),
				},
				PlanType:     account.PlanTypeUnlimited,
				BalanceToken: 100,
			},
			responseUpdatedAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a4444444-0000-0000-0000-000000000001"),
				},
				PlanType:     account.PlanTypeUnlimited,
				BalanceToken: -1900,
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

			// get account to check plan type — unlimited
			mockDB.EXPECT().AccountGet(ctx, tt.accountID).Return(tt.responseAccount, nil)
			// unlimited bypasses to SubtractTokens (no check)
			mockDB.EXPECT().AccountSubtractTokens(ctx, tt.accountID, tt.amount).Return(nil)
			// get updated account
			mockDB.EXPECT().AccountGet(ctx, tt.accountID).Return(tt.responseUpdatedAccount, nil)

			res, err := h.SubtractTokensWithCheck(ctx, tt.accountID, tt.amount)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.responseUpdatedAccount, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseUpdatedAccount, res)
			}
		})
	}
}

func Test_SubtractTokensWithCheck_insufficient(t *testing.T) {
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

	accountID := uuid.FromStringOrNil("a5555555-0000-0000-0000-000000000001")

	mockDB.EXPECT().AccountGet(ctx, accountID).Return(&account.Account{
		Identity:     commonidentity.Identity{ID: accountID},
		PlanType:     account.PlanTypeFree,
		BalanceToken: 100,
	}, nil)
	mockDB.EXPECT().AccountSubtractTokensWithCheck(ctx, accountID, int64(2000)).Return(fmt.Errorf("insufficient balance"))

	_, err := h.SubtractTokensWithCheck(ctx, accountID, 2000)
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
}

func Test_AddTokens_error(t *testing.T) {

	type test struct {
		name string

		accountID uuid.UUID
		amount    int64
	}

	tests := []test{
		{
			name: "db add error",

			accountID: uuid.FromStringOrNil("a6666666-0000-0000-0000-000000000001"),
			amount:    500,
		},
		{
			name: "db get error",

			accountID: uuid.FromStringOrNil("a7777777-0000-0000-0000-000000000001"),
			amount:    500,
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

			if tt.name == "db add error" {
				mockDB.EXPECT().AccountAddTokens(ctx, tt.accountID, tt.amount).Return(fmt.Errorf("add failed"))

				_, err := h.AddTokens(ctx, tt.accountID, tt.amount)
				if err == nil {
					t.Errorf("Wrong match. expect: error, got: nil")
				}
				return
			}

			if tt.name == "db get error" {
				mockDB.EXPECT().AccountAddTokens(ctx, tt.accountID, tt.amount).Return(nil)
				mockDB.EXPECT().AccountGet(ctx, tt.accountID).Return(nil, fmt.Errorf("get failed"))

				_, err := h.AddTokens(ctx, tt.accountID, tt.amount)
				if err == nil {
					t.Errorf("Wrong match. expect: error, got: nil")
				}
			}
		})
	}
}

func Test_SubtractTokens_error(t *testing.T) {

	type test struct {
		name string

		accountID uuid.UUID
		amount    int64
	}

	tests := []test{
		{
			name: "db subtract error",

			accountID: uuid.FromStringOrNil("a8888888-0000-0000-0000-000000000001"),
			amount:    300,
		},
		{
			name: "db get error",

			accountID: uuid.FromStringOrNil("a9999999-0000-0000-0000-000000000001"),
			amount:    300,
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

			if tt.name == "db subtract error" {
				mockDB.EXPECT().AccountSubtractTokens(ctx, tt.accountID, tt.amount).Return(fmt.Errorf("subtract failed"))

				_, err := h.SubtractTokens(ctx, tt.accountID, tt.amount)
				if err == nil {
					t.Errorf("Wrong match. expect: error, got: nil")
				}
				return
			}

			if tt.name == "db get error" {
				mockDB.EXPECT().AccountSubtractTokens(ctx, tt.accountID, tt.amount).Return(nil)
				mockDB.EXPECT().AccountGet(ctx, tt.accountID).Return(nil, fmt.Errorf("get failed"))

				_, err := h.SubtractTokens(ctx, tt.accountID, tt.amount)
				if err == nil {
					t.Errorf("Wrong match. expect: error, got: nil")
				}
			}
		})
	}
}

func Test_SubtractTokensWithCheck_get_error(t *testing.T) {
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

	accountID := uuid.FromStringOrNil("ab000000-0000-0000-0000-000000000001")

	mockDB.EXPECT().AccountGet(ctx, accountID).Return(nil, fmt.Errorf("initial get failed"))

	_, err := h.SubtractTokensWithCheck(ctx, accountID, int64(500))
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
}

func Test_SubtractBalanceWithCheck_get_error(t *testing.T) {
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

	accountID := uuid.FromStringOrNil("ae5f6071-92a3-11ee-ebf2-7066778c66jh")

	mockDB.EXPECT().AccountGet(ctx, accountID).Return(nil, fmt.Errorf("initial get failed"))

	_, err := h.SubtractBalanceWithCheck(ctx, accountID, int64(50000000))
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
}
