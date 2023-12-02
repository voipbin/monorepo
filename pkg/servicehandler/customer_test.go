package servicehandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	bmaccount "gitlab.com/voipbin/bin-manager/billing-manager.git/models/account"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	cspermission "gitlab.com/voipbin/bin-manager/customer-manager.git/models/permission"

	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/dbhandler"
)

func Test_CustomerCreate(t *testing.T) {

	type test struct {
		name string

		agent         *amagent.Agent
		username      string
		password      string
		customerName  string
		detail        string
		email         string
		phoneNumber   string
		address       string
		webhookMethod cscustomer.WebhookMethod
		webhookURI    string
		permissionIDs []uuid.UUID

		responseCustomer *cscustomer.Customer
		expectRes        *cscustomer.WebhookMessage
	}

	tests := []test{
		{
			"normal",

			&amagent.Agent{
				ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Permission: amagent.PermissionProjectSuperAdmin,
			},
			"test",
			"testpassword",
			"test",
			"test detail",
			"test@test.com",
			"+821100000001",
			"somewhere",
			cscustomer.WebhookMethodPost,
			"test.com",
			[]uuid.UUID{
				cspermission.PermissionAdmin.ID,
			},

			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("ade4707c-837d-11ec-a600-f30a3ccf56ae"),
			},
			&cscustomer.WebhookMessage{
				ID: uuid.FromStringOrNil("ade4707c-837d-11ec-a600-f30a3ccf56ae"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}

			ctx := context.Background()

			mockReq.EXPECT().CustomerV1CustomerCreate(ctx, 30000, tt.username, tt.password, tt.customerName, tt.detail, tt.email, tt.phoneNumber, tt.address, tt.webhookMethod, tt.webhookURI, tt.permissionIDs).Return(tt.responseCustomer, nil)

			res, err := h.CustomerCreate(ctx, tt.agent, tt.username, tt.password, tt.customerName, tt.detail, tt.email, tt.phoneNumber, tt.address, tt.webhookMethod, tt.webhookURI, tt.permissionIDs)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func TestCustomerGet(t *testing.T) {

	type test struct {
		name string

		customer *amagent.Agent
		id       uuid.UUID

		responseCustomer *cscustomer.Customer
		expectRes        *cscustomer.WebhookMessage
	}

	tests := []test{
		{
			"normal",

			&amagent.Agent{
				ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
				CustomerID: uuid.FromStringOrNil("a0f4b592-837e-11ec-9f5f-2f2051d4adac"),
				Permission: amagent.PermissionCustomerAdmin,
			},
			uuid.FromStringOrNil("a0f4b592-837e-11ec-9f5f-2f2051d4adac"),

			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("a0f4b592-837e-11ec-9f5f-2f2051d4adac"),
			},
			&cscustomer.WebhookMessage{
				ID: uuid.FromStringOrNil("a0f4b592-837e-11ec-9f5f-2f2051d4adac"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}

			ctx := context.Background()

			mockReq.EXPECT().CustomerV1CustomerGet(ctx, tt.id).Return(tt.responseCustomer, nil)

			res, err := h.CustomerGet(ctx, tt.customer, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_CustomerGets(t *testing.T) {

	type test struct {
		name string

		agent *amagent.Agent
		size  uint64
		token string

		responseCustomers []cscustomer.Customer
		expectRes         []*cscustomer.WebhookMessage
	}

	tests := []test{
		{
			"normal",

			&amagent.Agent{
				ID:         uuid.FromStringOrNil("3e6fe9c8-837e-11ec-84ef-b762e8a7a8fc"),
				Permission: amagent.PermissionProjectSuperAdmin,
			},

			10,
			"2020-09-20T03:23:20.995000",

			[]cscustomer.Customer{
				{
					ID: uuid.FromStringOrNil("8ffa19a2-837f-11ec-b57e-9f3906006c0a"),
				},
			},
			[]*cscustomer.WebhookMessage{
				{
					ID: uuid.FromStringOrNil("8ffa19a2-837f-11ec-b57e-9f3906006c0a"),
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

			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}

			ctx := context.Background()

			mockReq.EXPECT().CustomerV1CustomerGets(ctx, tt.token, tt.size).Return(tt.responseCustomers, nil)

			res, err := h.CustomerGets(ctx, tt.agent, tt.size, tt.token)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_CustomerUpdate(t *testing.T) {

	type test struct {
		name  string
		agent *amagent.Agent

		id            uuid.UUID
		customerName  string
		detail        string
		email         string
		phoneNumber   string
		address       string
		webhookMethod cscustomer.WebhookMethod
		webhookURI    string

		responseCustomers *cscustomer.Customer
		expectRes         *cscustomer.WebhookMessage
	}

	tests := []test{
		{
			"normal",

			&amagent.Agent{
				ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
				CustomerID: uuid.FromStringOrNil("8ffa19a2-837f-11ec-b57e-9f3906006c0a"),
				Permission: amagent.PermissionCustomerAdmin,
			},
			uuid.FromStringOrNil("8ffa19a2-837f-11ec-b57e-9f3906006c0a"),
			"name new",
			"detail new",
			"test@test.com",
			"+821100000001",
			"somewhere",
			cscustomer.WebhookMethodPost,
			"test.com",

			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("8ffa19a2-837f-11ec-b57e-9f3906006c0a"),
			},
			&cscustomer.WebhookMessage{
				ID: uuid.FromStringOrNil("8ffa19a2-837f-11ec-b57e-9f3906006c0a"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}

			ctx := context.Background()

			mockReq.EXPECT().CustomerV1CustomerGet(ctx, tt.id).Return(tt.responseCustomers, nil)
			mockReq.EXPECT().CustomerV1CustomerUpdate(ctx, tt.id, tt.customerName, tt.detail, tt.email, tt.phoneNumber, tt.address, tt.webhookMethod, tt.webhookURI).Return(tt.responseCustomers, nil)

			res, err := h.CustomerUpdate(ctx, tt.agent, tt.id, tt.customerName, tt.detail, tt.email, tt.phoneNumber, tt.address, tt.webhookMethod, tt.webhookURI)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_CustomerDelete(t *testing.T) {

	type test struct {
		name string

		agent *amagent.Agent
		id    uuid.UUID

		responseCustomers *cscustomer.Customer
		expectRes         *cscustomer.WebhookMessage
	}

	tests := []test{
		{
			"normal",

			&amagent.Agent{
				ID:         uuid.FromStringOrNil("3e6fe9c8-837e-11ec-84ef-b762e8a7a8fc"),
				Permission: amagent.PermissionProjectSuperAdmin,
			},
			uuid.FromStringOrNil("d83b9e02-837f-11ec-af3d-b75e44476e6b"),

			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("8ffa19a2-837f-11ec-b57e-9f3906006c0a"),
			},
			&cscustomer.WebhookMessage{
				ID: uuid.FromStringOrNil("8ffa19a2-837f-11ec-b57e-9f3906006c0a"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}

			ctx := context.Background()

			mockReq.EXPECT().CustomerV1CustomerGet(ctx, tt.id).Return(tt.responseCustomers, nil)
			mockReq.EXPECT().CustomerV1CustomerDelete(ctx, tt.id).Return(tt.responseCustomers, nil)

			res, err := h.CustomerDelete(ctx, tt.agent, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_CustomerUpdatePassword(t *testing.T) {

	type test struct {
		name string

		agent    *amagent.Agent
		id       uuid.UUID
		password string

		responseCustomers *cscustomer.Customer
		expectRes         *cscustomer.WebhookMessage
	}

	tests := []test{
		{
			"normal",

			&amagent.Agent{
				ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
				CustomerID: uuid.FromStringOrNil("d83b9e02-837f-11ec-af3d-b75e44476e6b"),
				Permission: amagent.PermissionCustomerAdmin,
			},
			uuid.FromStringOrNil("d83b9e02-837f-11ec-af3d-b75e44476e6b"),
			"new password",

			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("d83b9e02-837f-11ec-af3d-b75e44476e6b"),
			},
			&cscustomer.WebhookMessage{
				ID: uuid.FromStringOrNil("d83b9e02-837f-11ec-af3d-b75e44476e6b"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}

			ctx := context.Background()

			mockReq.EXPECT().CustomerV1CustomerGet(ctx, tt.id).Return(tt.responseCustomers, nil)
			mockReq.EXPECT().CustomerV1CustomerUpdatePassword(ctx, 30000, tt.id, tt.password).Return(tt.responseCustomers, nil)

			res, err := h.CustomerUpdatePassword(ctx, tt.agent, tt.id, tt.password)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_CustomerUpdatePermissionIDs(t *testing.T) {

	type test struct {
		name string

		agent         *amagent.Agent
		id            uuid.UUID
		permissionIDs []uuid.UUID

		responseCustomer *cscustomer.Customer
		expectRes        *cscustomer.WebhookMessage
	}

	tests := []test{
		{
			"normal",

			&amagent.Agent{
				ID:         uuid.FromStringOrNil("3e6fe9c8-837e-11ec-84ef-b762e8a7a8fc"),
				Permission: amagent.PermissionProjectSuperAdmin,
			},
			uuid.FromStringOrNil("d83b9e02-837f-11ec-af3d-b75e44476e6b"),
			[]uuid.UUID{
				uuid.FromStringOrNil("fb0baf8e-8380-11ec-8083-43ca175f4211"),
			},

			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("d83b9e02-837f-11ec-af3d-b75e44476e6b"),
			},
			&cscustomer.WebhookMessage{
				ID: uuid.FromStringOrNil("d83b9e02-837f-11ec-af3d-b75e44476e6b"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}

			ctx := context.Background()

			mockReq.EXPECT().CustomerV1CustomerUpdatePermissionIDs(ctx, tt.id, tt.permissionIDs).Return(tt.responseCustomer, nil)

			res, err := h.CustomerUpdatePermissionIDs(ctx, tt.agent, tt.id, tt.permissionIDs)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_CustomerUpdateBillingAccountID(t *testing.T) {

	type test struct {
		name string

		agent *amagent.Agent

		customerID       uuid.UUID
		billingAccountID uuid.UUID

		responseCustomer       *cscustomer.Customer
		responseBillingAccount *bmaccount.Account
		expectRes              *cscustomer.WebhookMessage
	}

	tests := []test{
		{
			name: "normal",

			agent: &amagent.Agent{
				ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
				CustomerID: uuid.FromStringOrNil("965f317e-1771-11ee-ac07-77247b121f85"),
				Permission: amagent.PermissionCustomerAdmin,
			},

			customerID:       uuid.FromStringOrNil("965f317e-1771-11ee-ac07-77247b121f85"),
			billingAccountID: uuid.FromStringOrNil("96a2ce84-1771-11ee-a155-83bf9a14ae55"),

			responseCustomer: &cscustomer.Customer{
				ID: uuid.FromStringOrNil("965f317e-1771-11ee-ac07-77247b121f85"),
			},
			responseBillingAccount: &bmaccount.Account{
				ID:         uuid.FromStringOrNil("96a2ce84-1771-11ee-a155-83bf9a14ae55"),
				CustomerID: uuid.FromStringOrNil("965f317e-1771-11ee-ac07-77247b121f85"),
				TMDelete:   defaultTimestamp,
			},
			expectRes: &cscustomer.WebhookMessage{
				ID: uuid.FromStringOrNil("965f317e-1771-11ee-ac07-77247b121f85"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().CustomerV1CustomerGet(ctx, tt.customerID).Return(tt.responseCustomer, nil)
			mockReq.EXPECT().BillingV1AccountGet(ctx, tt.billingAccountID).Return(tt.responseBillingAccount, nil)
			mockReq.EXPECT().CustomerV1CustomerUpdateBillingAccountID(ctx, tt.customerID, tt.billingAccountID).Return(tt.responseCustomer, nil)

			res, err := h.CustomerUpdateBillingAccountID(ctx, tt.agent, tt.customerID, tt.billingAccountID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
