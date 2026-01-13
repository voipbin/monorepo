package accounthandler

import (
	"context"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-storage-manager/models/account"
	"monorepo/bin-storage-manager/pkg/dbhandler"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_Create(t *testing.T) {

	tests := []struct {
		name string

		customerID uuid.UUID

		responseUUID    uuid.UUID
		responseAccount *account.Account

		expectFilters map[account.Field]any
		expectAccount *account.Account
	}{
		{
			name: "normal",

			customerID: uuid.FromStringOrNil("955fa98a-1530-11ef-94b7-cfc6e6161c56"),

			responseUUID: uuid.FromStringOrNil("9e444942-199b-11ef-ad96-27cf40005448"),
			responseAccount: &account.Account{
				ID: uuid.FromStringOrNil("95ebf6c4-1530-11ef-932d-037065591eab"),
			},

			expectFilters: map[account.Field]any{
				account.FieldDeleted:    false,
				account.FieldCustomerID: uuid.FromStringOrNil("955fa98a-1530-11ef-94b7-cfc6e6161c56"),
			},
			expectAccount: &account.Account{
				ID:         uuid.FromStringOrNil("9e444942-199b-11ef-ad96-27cf40005448"),
				CustomerID: uuid.FromStringOrNil("955fa98a-1530-11ef-94b7-cfc6e6161c56"),
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
			h := &accountHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)

			mockDB.EXPECT().AccountGets(ctx, "", uint64(1), tt.expectFilters).Return([]*account.Account{}, nil)
			mockDB.EXPECT().AccountCreate(ctx, tt.expectAccount).Return(nil)
			mockDB.EXPECT().AccountGet(ctx, tt.responseUUID).Return(tt.responseAccount, nil)
			mockNotify.EXPECT().PublishEvent(ctx, account.EventTypeAccountCreated, tt.responseAccount)

			res, err := h.Create(ctx, tt.customerID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseAccount) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseAccount, res)
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

			id: uuid.FromStringOrNil("26c0a430-199e-11ef-a630-d3d542a183ea"),

			responseAccount: &account.Account{
				ID: uuid.FromStringOrNil("26c0a430-199e-11ef-a630-d3d542a183ea"),
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
			h := &accountHandler{
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

			if !reflect.DeepEqual(res, tt.responseAccount) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseAccount, res)
			}
		})
	}
}

func Test_Gets(t *testing.T) {

	tests := []struct {
		name string

		token   string
		size    uint64
		filters map[account.Field]any

		responseAccounts []*account.Account
	}{
		{
			name: "normal",

			token: "2024-05-16 03:22:17.995000",
			size:  10,
			filters: map[account.Field]any{
				account.FieldCustomerID: uuid.FromStringOrNil("5d2c486c-199e-11ef-9826-5b645642ab65"),
			},

			responseAccounts: []*account.Account{
				{
					ID: uuid.FromStringOrNil("5d081046-199e-11ef-9e54-6ff7cd134471"),
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
			h := &accountHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().AccountGets(ctx, tt.token, tt.size, tt.filters).Return(tt.responseAccounts, nil)

			res, err := h.Gets(ctx, tt.token, tt.size, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseAccounts) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseAccounts, res)
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

			id: uuid.FromStringOrNil("7973d152-199e-11ef-b2b5-db084d3be49c"),

			responseAccount: &account.Account{
				ID: uuid.FromStringOrNil("7973d152-199e-11ef-b2b5-db084d3be49c"),
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
			h := &accountHandler{
				utilHandler:   mockUtil,
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

func Test_IncreaseFileInfo(t *testing.T) {

	tests := []struct {
		name string

		id        uuid.UUID
		filecount int64
		filesize  int64

		responseAccount *account.Account
	}{
		{
			name: "normal",

			id:        uuid.FromStringOrNil("7973d152-199e-11ef-b2b5-db084d3be49c"),
			filecount: 10,
			filesize:  10240,

			responseAccount: &account.Account{
				ID: uuid.FromStringOrNil("7973d152-199e-11ef-b2b5-db084d3be49c"),
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
			h := &accountHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().AccountIncreaseFileInfo(ctx, tt.id, tt.filecount, tt.filesize).Return(nil)
			mockDB.EXPECT().AccountGet(ctx, tt.id).Return(tt.responseAccount, nil)
			mockNotify.EXPECT().PublishEvent(ctx, account.EventTypeAccountUpdated, tt.responseAccount)

			res, err := h.IncreaseFileInfo(ctx, tt.id, tt.filecount, tt.filesize)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseAccount) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseAccount, res)
			}
		})
	}
}

func Test_DecreaseFileInfo(t *testing.T) {

	tests := []struct {
		name string

		id        uuid.UUID
		filecount int64
		filesize  int64

		responseAccount *account.Account
	}{
		{
			name: "normal",

			id:        uuid.FromStringOrNil("be5dffbc-19a4-11ef-b246-0b07e0c1d561"),
			filecount: 10,
			filesize:  10240,

			responseAccount: &account.Account{
				ID: uuid.FromStringOrNil("be5dffbc-19a4-11ef-b246-0b07e0c1d561"),
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
			h := &accountHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().AccountDecreaseFileInfo(ctx, tt.id, tt.filecount, tt.filesize).Return(nil)
			mockDB.EXPECT().AccountGet(ctx, tt.id).Return(tt.responseAccount, nil)
			mockNotify.EXPECT().PublishEvent(ctx, account.EventTypeAccountUpdated, tt.responseAccount)

			res, err := h.DecreaseFileInfo(ctx, tt.id, tt.filecount, tt.filesize)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseAccount) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseAccount, res)
			}
		})
	}
}

func Test_ValidateFileInfoByCustomerID(t *testing.T) {

	tests := []struct {
		name string

		customerID uuid.UUID
		filecount  int64
		filesize   int64

		responseAccounts []*account.Account
		expectFilters    map[account.Field]any
	}{
		{
			name: "normal",

			customerID: uuid.FromStringOrNil("1f156e1c-19aa-11ef-84fa-1facf3d5fbf1"),
			filecount:  10,
			filesize:   10240,

			responseAccounts: []*account.Account{
				{
					ID: uuid.FromStringOrNil("22fb7260-19aa-11ef-970a-8fd8ab8e92c0"),
				},
			},
			expectFilters: map[account.Field]any{
				account.FieldDeleted:    false,
				account.FieldCustomerID: uuid.FromStringOrNil("1f156e1c-19aa-11ef-84fa-1facf3d5fbf1"),
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
			h := &accountHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().AccountGets(ctx, "", uint64(1), tt.expectFilters).Return(tt.responseAccounts, nil)

			res, err := h.ValidateFileInfoByCustomerID(ctx, tt.customerID, tt.filecount, tt.filesize)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseAccounts[0]) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseAccounts[0], res)
			}
		})
	}
}
