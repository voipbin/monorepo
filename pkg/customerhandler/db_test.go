package customerhandler

import (
	"context"
	"fmt"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	bmaccount "gitlab.com/voipbin/bin-manager/billing-manager.git/models/account"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"

	"gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	"gitlab.com/voipbin/bin-manager/customer-manager.git/models/permission"
	"gitlab.com/voipbin/bin-manager/customer-manager.git/pkg/dbhandler"
)

func Test_Gets(t *testing.T) {

	tests := []struct {
		name   string
		size   uint64
		token  string
		result []*customer.Customer
	}{
		{
			"normal",
			10,
			"",
			[]*customer.Customer{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &customerHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyhandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().CustomerGets(gomock.Any(), tt.size, tt.token).Return(tt.result, nil)
			_, err := h.Gets(ctx, tt.size, tt.token)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_Create(t *testing.T) {

	tests := []struct {
		name string

		username      string
		password      string
		userName      string
		detail        string
		webhookMethod customer.WebhookMethod
		webhookURI    string
		permissionIDs []uuid.UUID

		responseUUID           uuid.UUID
		responseBillingAccount *bmaccount.Account
	}{
		{
			name: "normal",

			username:      "test username",
			password:      "test userpass",
			userName:      "test1",
			detail:        "detail1",
			webhookMethod: customer.WebhookMethodPost,
			webhookURI:    "test.com",
			permissionIDs: []uuid.UUID{
				permission.PermissionAdmin.ID,
			},

			responseUUID: uuid.FromStringOrNil("4b9ff112-02ec-11ee-b037-5b5c308ec044"),
			responseBillingAccount: &bmaccount.Account{
				ID: uuid.FromStringOrNil("2d5d9a8c-0e87-11ee-aeaf-4b3b6fad0c9b"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &customerHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				db:            mockDB,
				notifyhandler: mockNotify,
			}
			ctx := context.Background()

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockReq.EXPECT().BillingV1AccountCreate(ctx, tt.responseUUID, gomock.Any(), gomock.Any()).Return(tt.responseBillingAccount, nil)
			mockDB.EXPECT().CustomerGetByUsername(ctx, tt.username).Return(nil, fmt.Errorf("not found"))
			mockDB.EXPECT().CustomerCreate(ctx, gomock.Any()).Return(nil)
			mockDB.EXPECT().CustomerGet(ctx, tt.responseUUID).Return(&customer.Customer{}, nil)
			mockNotify.EXPECT().PublishEvent(ctx, customer.EventTypeCustomerCreated, gomock.Any()).Return()

			_, err := h.Create(ctx, tt.username, tt.password, tt.userName, tt.detail, tt.webhookMethod, tt.webhookURI, tt.permissionIDs)
			if err != nil {
				t.Errorf("Wrong match. expect:ok, got:%v", err)
			}
		})
	}
}

func Test_Delete(t *testing.T) {
	tests := []struct {
		name                    string
		id                      uuid.UUID
		responseBillingAccounts []bmaccount.Account
	}{
		{
			name: "normal1",
			id:   uuid.FromStringOrNil("4cd23368-7cb7-11ec-9466-8318ef5a7125"),

			responseBillingAccounts: []bmaccount.Account{
				{
					ID:       uuid.FromStringOrNil("9f795cf8-0e89-11ee-91c9-4b1ab8ec02e8"),
					TMDelete: dbhandler.DefaultTimeStamp,
				},
				{
					ID:       uuid.FromStringOrNil("9fc63fa0-0e89-11ee-ab57-37a53b33df1c"),
					TMDelete: dbhandler.DefaultTimeStamp,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &customerHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyhandler: mockNotify,
			}
			ctx := context.Background()

			mockReq.EXPECT().BillingV1AccountGets(ctx, tt.id, "", uint64(100)).Return(tt.responseBillingAccounts, nil)
			for _, a := range tt.responseBillingAccounts {
				mockReq.EXPECT().BillingV1AccountDelete(ctx, a.ID).Return(&a, nil)
			}

			mockDB.EXPECT().CustomerDelete(gomock.Any(), tt.id).Return(nil)
			mockDB.EXPECT().CustomerGet(gomock.Any(), gomock.Any()).Return(&customer.Customer{}, nil)
			mockNotify.EXPECT().PublishEvent(gomock.Any(), customer.EventTypeCustomerDeleted, gomock.Any()).Return()

			_, err := h.Delete(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestUpdateBasicInfo(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	h := &customerHandler{
		reqHandler:    mockReq,
		db:            mockDB,
		notifyhandler: mockNotify,
	}

	tests := []struct {
		name string

		id            uuid.UUID
		userName      string
		detail        string
		webhookMethod customer.WebhookMethod
		webhookURI    string
	}{
		{
			"normal",
			uuid.FromStringOrNil("c106fa66-7cb7-11ec-b438-1320d9493dee"),
			"name new",
			"detail new",
			customer.WebhookMethodPost,
			"test.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockDB.EXPECT().CustomerSetBasicInfo(gomock.Any(), tt.id, tt.userName, tt.detail, tt.webhookMethod, tt.webhookURI).Return(nil)
			mockDB.EXPECT().CustomerGet(gomock.Any(), gomock.Any()).Return(&customer.Customer{}, nil)
			mockNotify.EXPECT().PublishEvent(gomock.Any(), customer.EventTypeCustomerUpdated, gomock.Any()).Return()

			_, err := h.UpdateBasicInfo(ctx, tt.id, tt.userName, tt.detail, tt.webhookMethod, tt.webhookURI)
			if err != nil {
				t.Errorf("Wrong match. expect:ok, got:%v", err)
			}
		})
	}
}

func TestUpdatePassword(t *testing.T) {

	tests := []struct {
		name string

		id       uuid.UUID
		password string
	}{
		{
			"normal",
			uuid.FromStringOrNil("9d96af3a-7cb8-11ec-bada-e76b739ab5b9"),
			"password new",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &customerHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyhandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().CustomerSetPasswordHash(gomock.Any(), tt.id, gomock.Any()).Return(nil)
			mockDB.EXPECT().CustomerGet(gomock.Any(), tt.id).Return(&customer.Customer{}, nil)
			mockNotify.EXPECT().PublishEvent(gomock.Any(), customer.EventTypeCustomerUpdated, gomock.Any()).Return()

			_, err := h.UpdatePassword(ctx, tt.id, tt.password)
			if err != nil {
				t.Errorf("Wrong match. expect:ok, got:%v", err)
			}
		})
	}
}

func TestUpdatePermissionIDs(t *testing.T) {

	tests := []struct {
		name string

		id            uuid.UUID
		permissionIDs []uuid.UUID
	}{
		{
			"normal",
			uuid.FromStringOrNil("a872209e-7cca-11ec-ae6b-c706f32a1000"),
			[]uuid.UUID{
				uuid.FromStringOrNil("d1a7d68e-7cca-11ec-8236-6b6d097114c1"),
			},
		},
		{
			"2 items",
			uuid.FromStringOrNil("c4505628-7cca-11ec-8ece-8347f1fd0064"),
			[]uuid.UUID{
				uuid.FromStringOrNil("c478b2d0-7cca-11ec-9c9e-ebf437c734cc"),
				uuid.FromStringOrNil("c4d08456-7cca-11ec-bdca-7391a485b4fb"),
			},
		},
		{
			"0 item",
			uuid.FromStringOrNil("dc5dfb12-7cca-11ec-8202-7318fc97013a"),
			[]uuid.UUID{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &customerHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyhandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().CustomerSetPermissionIDs(gomock.Any(), tt.id, tt.permissionIDs)
			mockDB.EXPECT().CustomerGet(gomock.Any(), gomock.Any()).Return(&customer.Customer{}, nil)
			mockNotify.EXPECT().PublishEvent(gomock.Any(), customer.EventTypeCustomerUpdated, gomock.Any()).Return()

			_, err := h.UpdatePermissionIDs(ctx, tt.id, tt.permissionIDs)
			if err != nil {
				t.Errorf("Wrong match. expect:ok, got:%v", err)
			}
		})
	}
}

func TestLogin(t *testing.T) {

	tests := []struct {
		name string

		username string
		password string

		responseGet *customer.Customer
	}{
		{
			"normal",
			"a13c6c24-7ccc-11ec-86c7-133d05b8ea4e",
			"password1",

			&customer.Customer{
				ID:           uuid.FromStringOrNil("a13c6c24-7ccc-11ec-86c7-133d05b8ea4e"),
				Username:     "a13c6c24-7ccc-11ec-86c7-133d05b8ea4e",
				PasswordHash: "$2a$12$z6fM.TRL7XdYJc7Ea.GGHOCIDe46vWl.h485o5hiid454ASroCOga",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &customerHandler{
				reqHandler: mockReq,
				db:         mockDB,
			}
			ctx := context.Background()

			mockDB.EXPECT().CustomerGetByUsername(gomock.Any(), tt.username).Return(tt.responseGet, nil)

			_, err := h.Login(ctx, tt.username, tt.password)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
