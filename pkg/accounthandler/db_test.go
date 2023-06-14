package accounthandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"

	"gitlab.com/voipbin/bin-manager/billing-manager.git/models/account"
	"gitlab.com/voipbin/bin-manager/billing-manager.git/pkg/dbhandler"
)

func Test_Create(t *testing.T) {

	type test struct {
		name string

		customerID uuid.UUID

		responseUUID    uuid.UUID
		responseAccount *account.Account

		expectAccount *account.Account
	}

	tests := []test{
		{
			name: "normal",

			customerID: uuid.FromStringOrNil("06011a02-08f3-11ee-b4c1-73257fafcdb3"),

			responseUUID: uuid.FromStringOrNil("3972c20e-08f4-11ee-893b-032e633ef73a"),
			responseAccount: &account.Account{
				ID: uuid.FromStringOrNil("3972c20e-08f4-11ee-893b-032e633ef73a"),
			},

			expectAccount: &account.Account{
				ID:            uuid.FromStringOrNil("3972c20e-08f4-11ee-893b-032e633ef73a"),
				CustomerID:    uuid.FromStringOrNil("06011a02-08f3-11ee-b4c1-73257fafcdb3"),
				Type:          account.TypeNormal,
				Balance:       0,
				PaymentType:   account.PaymentTypeNone,
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

			res, err := h.Create(ctx, tt.customerID)
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
				ID: uuid.FromStringOrNil("8ba5c986-08f4-11ee-a292-4f01790e9d2b"),
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

		responseAccount *account.Account
	}

	tests := []test{
		{
			name: "normal",

			customerID: uuid.FromStringOrNil("b7211fa2-08f4-11ee-8f49-17448c4b2951"),

			responseAccount: &account.Account{
				ID: uuid.FromStringOrNil("b74fe472-08f4-11ee-8dbb-8bec316a131c"),
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

			mockDB.EXPECT().AccountGetByCustomerID(ctx, tt.customerID).Return(tt.responseAccount, nil)

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

func Test_SubstractBalanceByCustomer(t *testing.T) {

	type test struct {
		name string

		customerID uuid.UUID
		balance    float32

		responseAccount *account.Account
	}

	tests := []test{
		{
			name: "normal",

			customerID: uuid.FromStringOrNil("28c90ab4-09f6-11ee-97f3-6b2314b71a97"),
			balance:    20.1,

			responseAccount: &account.Account{
				ID: uuid.FromStringOrNil("287f9762-09f6-11ee-a305-2f57be32e59c"),
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

			mockDB.EXPECT().AccountSubstractBalanceByCustomerID(ctx, tt.customerID, tt.balance).Return(nil)
			mockDB.EXPECT().AccountGetByCustomerID(ctx, tt.customerID).Return(tt.responseAccount, nil)

			res, err := h.SubstractBalanceByCustomer(ctx, tt.customerID, tt.balance)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.responseAccount, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseAccount, res)
			}
		})
	}
}

func Test_AddBalanceByCustomer(t *testing.T) {

	type test struct {
		name string

		customerID uuid.UUID
		balance    float32

		responseAccount *account.Account
	}

	tests := []test{
		{
			name: "normal",

			customerID: uuid.FromStringOrNil("28c90ab4-09f6-11ee-97f3-6b2314b71a97"),
			balance:    20.1,

			responseAccount: &account.Account{
				ID: uuid.FromStringOrNil("f3f570d8-09f6-11ee-a665-fb550b87c0f6"),
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

			mockDB.EXPECT().AccountAddBalanceByCustomerID(ctx, tt.customerID, tt.balance).Return(nil)
			mockDB.EXPECT().AccountGetByCustomerID(ctx, tt.customerID).Return(tt.responseAccount, nil)

			res, err := h.AddBalanceByCustomer(ctx, tt.customerID, tt.balance)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.responseAccount, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseAccount, res)
			}
		})
	}
}
