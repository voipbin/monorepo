package customerhandler

import (
	"context"
	"testing"

	bmaccount "monorepo/bin-billing-manager/models/account"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"monorepo/bin-customer-manager/models/customer"
	"monorepo/bin-customer-manager/pkg/dbhandler"
)

func Test_Gets(t *testing.T) {

	tests := []struct {
		name    string
		size    uint64
		token   string
		filters map[string]string

		result []*customer.Customer
	}{
		{
			"normal",
			10,
			"",
			map[string]string{
				"deleted": "false",
			},

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
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().CustomerGets(gomock.Any(), tt.size, tt.token, tt.filters).Return(tt.result, nil)
			_, err := h.Gets(ctx, tt.size, tt.token, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_Create(t *testing.T) {

	tests := []struct {
		name string

		userName      string
		detail        string
		email         string
		phoneNumber   string
		address       string
		webhookMethod customer.WebhookMethod
		webhookURI    string

		responseUUID           uuid.UUID
		responseBillingAccount *bmaccount.Account
		responseHash           string

		expectCustomer *customer.Customer
	}{
		{
			name: "normal",

			userName:      "test1",
			detail:        "detail1",
			email:         "test@test.com",
			phoneNumber:   "+821100000001",
			address:       "somewhere",
			webhookMethod: customer.WebhookMethodPost,
			webhookURI:    "test.com",

			responseUUID: uuid.FromStringOrNil("4b9ff112-02ec-11ee-b037-5b5c308ec044"),
			responseBillingAccount: &bmaccount.Account{
				ID: uuid.FromStringOrNil("2d5d9a8c-0e87-11ee-aeaf-4b3b6fad0c9b"),
			},
			responseHash: "$2a$12$KEqTmfExiTmQ0HBspD6x7.XBkG1mVVAKidWG6J.zUeTtdgb0NXppq",

			expectCustomer: &customer.Customer{
				ID:               uuid.FromStringOrNil("4b9ff112-02ec-11ee-b037-5b5c308ec044"),
				Name:             "test1",
				Detail:           "detail1",
				Email:            "test@test.com",
				PhoneNumber:      "+821100000001",
				Address:          "somewhere",
				WebhookMethod:    customer.WebhookMethodPost,
				WebhookURI:       "test.com",
				BillingAccountID: uuid.Nil,
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
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockDB.EXPECT().CustomerCreate(ctx, tt.expectCustomer).Return(nil)
			mockDB.EXPECT().CustomerGet(ctx, tt.responseUUID).Return(&customer.Customer{}, nil)
			mockNotify.EXPECT().PublishEvent(ctx, customer.EventTypeCustomerCreated, gomock.Any()).Return()

			_, err := h.Create(ctx, tt.userName, tt.detail, tt.email, tt.phoneNumber, tt.address, tt.webhookMethod, tt.webhookURI)
			if err != nil {
				t.Errorf("Wrong match. expect:ok, got:%v", err)
			}
		})
	}
}

func Test_dbDelete(t *testing.T) {
	tests := []struct {
		name string
		id   uuid.UUID

		responseCustomer *customer.Customer
	}{
		{
			name: "normal1",
			id:   uuid.FromStringOrNil("4cd23368-7cb7-11ec-9466-8318ef5a7125"),

			responseCustomer: &customer.Customer{
				ID: uuid.FromStringOrNil("4cd23368-7cb7-11ec-9466-8318ef5a7125"),
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
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().CustomerDelete(gomock.Any(), tt.id).Return(nil)
			mockDB.EXPECT().CustomerGet(gomock.Any(), tt.id).Return(tt.responseCustomer, nil)
			mockNotify.EXPECT().PublishEvent(gomock.Any(), customer.EventTypeCustomerDeleted, tt.responseCustomer).Return()

			_, err := h.dbDelete(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_UpdateBasicInfo(t *testing.T) {

	tests := []struct {
		name string

		id           uuid.UUID
		customerName string
		detail       string
		email        string
		phoneNumber  string
		address      string

		webhookMethod customer.WebhookMethod
		webhookURI    string
	}{
		{
			name: "normal",

			id:            uuid.FromStringOrNil("c106fa66-7cb7-11ec-b438-1320d9493dee"),
			customerName:  "name new",
			detail:        "detail new",
			email:         "update@email",
			phoneNumber:   "+821100000001",
			address:       "update address",
			webhookMethod: customer.WebhookMethodPost,
			webhookURI:    "test.com",
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
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().CustomerSetBasicInfo(gomock.Any(), tt.id, tt.customerName, tt.detail, tt.email, tt.phoneNumber, tt.address, tt.webhookMethod, tt.webhookURI).Return(nil)
			mockDB.EXPECT().CustomerGet(gomock.Any(), gomock.Any()).Return(&customer.Customer{}, nil)
			mockNotify.EXPECT().PublishEvent(gomock.Any(), customer.EventTypeCustomerUpdated, gomock.Any()).Return()

			_, err := h.UpdateBasicInfo(ctx, tt.id, tt.customerName, tt.detail, tt.email, tt.phoneNumber, tt.address, tt.webhookMethod, tt.webhookURI)
			if err != nil {
				t.Errorf("Wrong match. expect:ok, got:%v", err)
			}
		})
	}
}

func Test_UpdateBillingAccountID(t *testing.T) {

	tests := []struct {
		name string

		id               uuid.UUID
		billingAccountID uuid.UUID
	}{
		{
			"normal",
			uuid.FromStringOrNil("f2eb3d1e-0f8f-11ee-b3bb-178ed8e3acb7"),
			uuid.FromStringOrNil("f32c6dca-0f8f-11ee-8aca-cfcc26f6900e"),
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
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().CustomerSetBillingAccountID(gomock.Any(), tt.id, tt.billingAccountID)
			mockDB.EXPECT().CustomerGet(gomock.Any(), tt.id).Return(&customer.Customer{}, nil)
			mockNotify.EXPECT().PublishEvent(gomock.Any(), customer.EventTypeCustomerUpdated, gomock.Any()).Return()

			_, err := h.UpdateBillingAccountID(ctx, tt.id, tt.billingAccountID)
			if err != nil {
				t.Errorf("Wrong match. expect:ok, got:%v", err)
			}
		})
	}
}
