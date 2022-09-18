package servicehandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	cspermission "gitlab.com/voipbin/bin-manager/customer-manager.git/models/permission"

	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/dbhandler"
)

func TestCustomerCreate(t *testing.T) {

	type test struct {
		name string

		customer      *cscustomer.Customer
		username      string
		password      string
		customerName  string
		detail        string
		webhookMethod cscustomer.WebhookMethod
		webhookURI    string
		lineSecret    string
		lineToken     string
		permissionIDs []uuid.UUID

		responseCustomer *cscustomer.Customer
		expectRes        *cscustomer.WebhookMessage
	}

	tests := []test{
		{
			"normal",

			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("3e6fe9c8-837e-11ec-84ef-b762e8a7a8fc"),
				PermissionIDs: []uuid.UUID{
					cspermission.PermissionAdmin.ID,
				},
			},
			"test",
			"testpassword",
			"test",
			"test detail",
			cscustomer.WebhookMethodPost,
			"test.com",
			"c5ea6344-ed44-11ec-b5bf-6726d8b17878",
			"c6511440-ed44-11ec-a3d7-3708bbaa641e",
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

			mockReq.EXPECT().CustomerV1CustomerCreate(ctx, 30000, tt.username, tt.password, tt.customerName, tt.detail, tt.webhookMethod, tt.webhookURI, tt.lineSecret, tt.lineToken, tt.permissionIDs).Return(tt.responseCustomer, nil)

			res, err := h.CustomerCreate(ctx, tt.customer, tt.username, tt.password, tt.customerName, tt.detail, tt.webhookMethod, tt.webhookURI, tt.lineSecret, tt.lineToken, tt.permissionIDs)
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

		customer *cscustomer.Customer
		id       uuid.UUID

		responseCustomer *cscustomer.Customer
		expectRes        *cscustomer.WebhookMessage
	}

	tests := []test{
		{
			"normal",

			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("3e6fe9c8-837e-11ec-84ef-b762e8a7a8fc"),
				PermissionIDs: []uuid.UUID{
					cspermission.PermissionAdmin.ID,
				},
			},
			uuid.FromStringOrNil("a0f4b592-837e-11ec-9f5f-2f2051d4adac"),

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

func TestCustomerGets(t *testing.T) {

	type test struct {
		name string

		customer *cscustomer.Customer
		size     uint64
		token    string

		responseCustomers []cscustomer.Customer
		expectRes         []*cscustomer.WebhookMessage
	}

	tests := []test{
		{
			"normal",

			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("3e6fe9c8-837e-11ec-84ef-b762e8a7a8fc"),
				PermissionIDs: []uuid.UUID{
					cspermission.PermissionAdmin.ID,
				},
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

			res, err := h.CustomerGets(ctx, tt.customer, tt.size, tt.token)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func TestCustomerUpdate(t *testing.T) {

	type test struct {
		name string

		customer      *cscustomer.Customer
		id            uuid.UUID
		customerName  string
		detail        string
		webhookMethod cscustomer.WebhookMethod
		webhookURI    string

		responseCustomers *cscustomer.Customer
		expectRes         *cscustomer.WebhookMessage
	}

	tests := []test{
		{
			"normal",

			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("3e6fe9c8-837e-11ec-84ef-b762e8a7a8fc"),
				PermissionIDs: []uuid.UUID{
					cspermission.PermissionAdmin.ID,
				},
			},
			uuid.FromStringOrNil("d83b9e02-837f-11ec-af3d-b75e44476e6b"),
			"name new",
			"detail new",
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
			mockReq.EXPECT().CustomerV1CustomerUpdate(ctx, tt.id, tt.customerName, tt.detail, tt.webhookMethod, tt.webhookURI).Return(tt.responseCustomers, nil)

			res, err := h.CustomerUpdate(ctx, tt.customer, tt.id, tt.customerName, tt.detail, tt.webhookMethod, tt.webhookURI)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func TestCustomerDelete(t *testing.T) {

	type test struct {
		name string

		customer *cscustomer.Customer
		id       uuid.UUID

		responseCustomers *cscustomer.Customer
		expectRes         *cscustomer.WebhookMessage
	}

	tests := []test{
		{
			"normal",

			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("3e6fe9c8-837e-11ec-84ef-b762e8a7a8fc"),
				PermissionIDs: []uuid.UUID{
					cspermission.PermissionAdmin.ID,
				},
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

			res, err := h.CustomerDelete(ctx, tt.customer, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func TestCustomerUpdatePassword(t *testing.T) {

	type test struct {
		name string

		customer *cscustomer.Customer
		id       uuid.UUID
		password string

		responseCustomers *cscustomer.Customer
		expectRes         *cscustomer.WebhookMessage
	}

	tests := []test{
		{
			"normal",

			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("3e6fe9c8-837e-11ec-84ef-b762e8a7a8fc"),
				PermissionIDs: []uuid.UUID{
					cspermission.PermissionAdmin.ID,
				},
			},
			uuid.FromStringOrNil("d83b9e02-837f-11ec-af3d-b75e44476e6b"),
			"new password",

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
			mockReq.EXPECT().CustomerV1CustomerUpdatePassword(ctx, 30000, tt.id, tt.password).Return(tt.responseCustomers, nil)

			res, err := h.CustomerUpdatePassword(ctx, tt.customer, tt.id, tt.password)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func TestCustomerUpdatePermissionIDs(t *testing.T) {

	type test struct {
		name string

		customer      *cscustomer.Customer
		id            uuid.UUID
		permissionIDs []uuid.UUID

		responseCustomer *cscustomer.Customer
		expectRes        *cscustomer.WebhookMessage
	}

	tests := []test{
		{
			"normal",

			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("3e6fe9c8-837e-11ec-84ef-b762e8a7a8fc"),
				PermissionIDs: []uuid.UUID{
					cspermission.PermissionAdmin.ID,
				},
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

			res, err := h.CustomerUpdatePermissionIDs(ctx, tt.customer, tt.id, tt.permissionIDs)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
