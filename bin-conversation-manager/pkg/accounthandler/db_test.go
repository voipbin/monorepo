package accounthandler

import (
	"context"
	reflect "reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-conversation-manager/models/account"
	"monorepo/bin-conversation-manager/pkg/dbhandler"
	"monorepo/bin-conversation-manager/pkg/linehandler"
)

func Test_Create(t *testing.T) {

	tests := []struct {
		name string

		customerID  uuid.UUID
		accountType account.Type
		accountName string
		detail      string
		secret      string
		token       string

		responseUUID    uuid.UUID
		responseAccount *account.Account

		expectAccount *account.Account
	}{
		{
			name: "normal",

			customerID:  uuid.FromStringOrNil("3b24255a-e60b-11ec-9815-5f679b51ac4d"),
			accountType: account.TypeLine,
			accountName: "test name",
			detail:      "test detail",
			secret:      "test secret",
			token:       "test token",

			responseUUID: uuid.FromStringOrNil("4f187bba-fdf7-11ed-87b2-d7cc82900fb6"),
			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("4f187bba-fdf7-11ed-87b2-d7cc82900fb6"),
				},
			},

			expectAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("4f187bba-fdf7-11ed-87b2-d7cc82900fb6"),
					CustomerID: uuid.FromStringOrNil("3b24255a-e60b-11ec-9815-5f679b51ac4d"),
				},
				Type:   account.TypeLine,
				Name:   "test name",
				Detail: "test detail",
				Secret: "test secret",
				Token:  "test token",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockLine := linehandler.NewMockLineHandler(mc)

			h := accountHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				lineHandler:   mockLine,
			}
			ctx := context.Background()

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockLine.EXPECT().Setup(ctx, tt.expectAccount).Return(nil)
			mockDB.EXPECT().AccountCreate(ctx, tt.expectAccount).Return(nil)
			mockDB.EXPECT().AccountGet(ctx, tt.responseUUID).Return(tt.responseAccount, nil)
			mockNotify.EXPECT().PublishEvent(ctx, account.EventTypeAccountCreated, tt.responseAccount)

			res, err := h.Create(ctx, tt.customerID, tt.accountType, tt.accountName, tt.detail, tt.secret, tt.token)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseAccount) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.responseAccount, res)
			}
		})
	}
}

func Test_Get(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseAccount *account.Account
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("4f33e7be-fdf8-11ed-9db9-77abc56c6166"),

			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("4f33e7be-fdf8-11ed-9db9-77abc56c6166"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)

			h := accountHandler{
				db:         mockDB,
				reqHandler: mockReq,
			}
			ctx := context.Background()

			mockDB.EXPECT().AccountGet(ctx, tt.id).Return(tt.responseAccount, nil)

			res, err := h.Get(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseAccount) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.responseAccount, res)
			}

		})
	}
}

func Test_List(t *testing.T) {

	tests := []struct {
		name string

		size    uint64
		token   string
		filters map[account.Field]any

		responseAccounts []*account.Account
	}{
		{
			name: "normal",

			size:  10,
			token: "2020-05-03%2021:35:02.809",
			filters: map[account.Field]any{
				account.FieldCustomerID: "99a9734a-3e16-11ef-94d4-9b7a8c5e0f6c",
			},

			responseAccounts: []*account.Account{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("1bcb19d2-fe49-11ed-be94-038ef49c197b"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("1bef6e18-fe49-11ed-a76f-dbb98d479fc1"),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)

			h := accountHandler{
				db:         mockDB,
				reqHandler: mockReq,
			}
			ctx := context.Background()

			mockDB.EXPECT().AccountList(ctx, tt.size, tt.token, tt.filters).Return(tt.responseAccounts, nil)

			res, err := h.List(ctx, tt.token, tt.size, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseAccounts) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.responseAccounts, res)
			}

		})
	}
}

func Test_Update(t *testing.T) {

	tests := []struct {
		name string

		id     uuid.UUID
		fields map[account.Field]any

		responseAccount *account.Account
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("b283ec96-fdf9-11ed-9d19-27d9e432deb5"),
			fields: map[account.Field]any{
				account.FieldName: "update name",
			},

			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("b283ec96-fdf9-11ed-9d19-27d9e432deb5"),
				},
				Type: account.TypeLine,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			mockLine := linehandler.NewMockLineHandler(mc)

			h := accountHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				lineHandler:   mockLine,
			}
			ctx := context.Background()

			mockDB.EXPECT().AccountUpdate(ctx, tt.id, tt.fields).Return(nil)
			mockDB.EXPECT().AccountGet(ctx, tt.id).Return(tt.responseAccount, nil)
			mockLine.EXPECT().Setup(ctx, tt.responseAccount).Return(nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseAccount.CustomerID, account.EventTypeAccountUpdated, tt.responseAccount)

			res, err := h.Update(ctx, tt.id, tt.fields)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseAccount) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.responseAccount, res)
			}

		})
	}
}

func Test_Delete(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseAccount *account.Account
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("74a879e6-fe49-11ed-98e7-576bc17c7b79"),

			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("74a879e6-fe49-11ed-98e7-576bc17c7b79"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &accountHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().AccountDelete(ctx, tt.id).Return(nil)
			mockDB.EXPECT().AccountGet(ctx, tt.id).Return(tt.responseAccount, nil)
			mockNotify.EXPECT().PublishEvent(ctx, account.EventTypeAccountDeleted, tt.responseAccount)

			res, err := h.Delete(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseAccount) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseAccount, res)
			}

		})
	}
}
