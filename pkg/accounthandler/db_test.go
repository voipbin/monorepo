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

		customerID  uuid.UUID
		accountName string
		detail      string

		responseUUID    uuid.UUID
		responseAccount *account.Account

		expectAccount *account.Account
	}

	tests := []test{
		{
			name: "normal",

			customerID:  uuid.FromStringOrNil("06011a02-08f3-11ee-b4c1-73257fafcdb3"),
			accountName: "test name",
			detail:      "test detail",

			responseUUID: uuid.FromStringOrNil("3972c20e-08f4-11ee-893b-032e633ef73a"),
			responseAccount: &account.Account{
				ID: uuid.FromStringOrNil("3972c20e-08f4-11ee-893b-032e633ef73a"),
			},

			expectAccount: &account.Account{
				ID:            uuid.FromStringOrNil("3972c20e-08f4-11ee-893b-032e633ef73a"),
				CustomerID:    uuid.FromStringOrNil("06011a02-08f3-11ee-b4c1-73257fafcdb3"),
				Name:          "test name",
				Detail:        "test detail",
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

			res, err := h.Create(ctx, tt.customerID, tt.accountName, tt.detail)
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

func Test_GetsByCustomerID(t *testing.T) {

	type test struct {
		name string

		customerID uuid.UUID
		size       uint64
		token      string

		responseAccounts []*account.Account
	}

	tests := []test{
		{
			name: "normal",

			customerID: uuid.FromStringOrNil("84fcc4f2-0e5a-11ee-8075-fb1ea789b671"),
			size:       10,
			token:      "2023-06-07 03:22:17.995000",

			responseAccounts: []*account.Account{
				{
					ID: uuid.FromStringOrNil("ba0e7152-0b96-11ee-9bbb-ef5ba49d06fb"),
				},
				{
					ID: uuid.FromStringOrNil("ba3c016c-0b96-11ee-9850-53f4edbab1be"),
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

			mockDB.EXPECT().AccountGetsByCustomerID(ctx, tt.customerID, tt.size, tt.token).Return(tt.responseAccounts, nil)

			res, err := h.Gets(ctx, tt.customerID, tt.size, tt.token)
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
				ID: uuid.FromStringOrNil("920b8062-0e68-11ee-b337-7315804888d0"),
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

func Test_DeletesByCustomerID(t *testing.T) {

	type test struct {
		name string

		customerID uuid.UUID

		responseAccounts []*account.Account

		expectRes []*account.Account
	}

	tests := []test{
		{
			name: "normal",

			customerID: uuid.FromStringOrNil("92435eb0-0e68-11ee-b841-3fe2d10a7ab9"),

			responseAccounts: []*account.Account{
				{
					ID:       uuid.FromStringOrNil("d7c03c9c-0e68-11ee-a061-7ff3a502f79b"),
					TMDelete: dbhandler.DefaultTimeStamp,
				},
				{
					ID:       uuid.FromStringOrNil("d7e88e72-0e68-11ee-8ecd-fffa5f8b006c"),
					TMDelete: dbhandler.DefaultTimeStamp,
				},
			},
			expectRes: []*account.Account{
				{
					ID:       uuid.FromStringOrNil("d7c03c9c-0e68-11ee-a061-7ff3a502f79b"),
					TMDelete: dbhandler.DefaultTimeStamp,
				},
				{
					ID:       uuid.FromStringOrNil("d7e88e72-0e68-11ee-8ecd-fffa5f8b006c"),
					TMDelete: dbhandler.DefaultTimeStamp,
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

			mockDB.EXPECT().AccountGetsByCustomerID(ctx, tt.customerID, uint64(100), "").Return(tt.responseAccounts, nil)

			for _, a := range tt.responseAccounts {
				mockDB.EXPECT().AccountDelete(ctx, a.ID).Return(nil)
				mockDB.EXPECT().AccountGet(ctx, a.ID).Return(a, nil)
			}

			res, err := h.DeletesByCustomerID(ctx, tt.customerID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
