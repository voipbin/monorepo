package listenhandler

import (
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	"gitlab.com/voipbin/bin-manager/customer-manager.git/models/permission"
	"gitlab.com/voipbin/bin-manager/customer-manager.git/pkg/customerhandler"
)

func Test_processV1CustomersGet(t *testing.T) {

	tests := []struct {
		name    string
		request *rabbitmqhandler.Request
		size    uint64
		token   string

		users     []*customer.Customer
		expectRes *rabbitmqhandler.Response
	}{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:      "/v1/customers?page_size=10&page_token=2021-11-23%2017:55:39.712000",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: "application/json",
			},
			10,
			"2021-11-23 17:55:39.712000",

			[]*customer.Customer{
				{
					ID:            uuid.FromStringOrNil("31b08066-7db2-11ec-8786-c7d9cf6c9b5f"),
					Username:      "test1",
					PasswordHash:  "",
					PermissionIDs: []uuid.UUID{},
					TMCreate:      "",
					TMUpdate:      "",
					TMDelete:      "",
				},
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"31b08066-7db2-11ec-8786-c7d9cf6c9b5f","username":"test1","name":"","detail":"","webhook_method":"","webhook_uri":"","permission_ids":[],"tm_create":"","tm_update":"","tm_delete":""}]`),
			},
		},
		{
			"2 customers",
			&rabbitmqhandler.Request{
				URI:      "/v1/customers?page_size=10&page_token=2021-11-23%2017:55:39.712000",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: "application/json",
			},
			10,
			"2021-11-23 17:55:39.712000",

			[]*customer.Customer{
				{
					ID:            uuid.FromStringOrNil("9f8a7880-7db2-11ec-9602-930411a1581f"),
					Username:      "test1",
					PermissionIDs: []uuid.UUID{},
				},
				{
					ID:            uuid.FromStringOrNil("a032c710-7db2-11ec-bfe0-83fa85a82603"),
					Username:      "test2",
					PermissionIDs: []uuid.UUID{},
				},
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"9f8a7880-7db2-11ec-9602-930411a1581f","username":"test1","name":"","detail":"","webhook_method":"","webhook_uri":"","permission_ids":[],"tm_create":"","tm_update":"","tm_delete":""},{"id":"a032c710-7db2-11ec-bfe0-83fa85a82603","username":"test2","name":"","detail":"","webhook_method":"","webhook_uri":"","permission_ids":[],"tm_create":"","tm_update":"","tm_delete":""}]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockCustomer := customerhandler.NewMockCustomerHandler(mc)

			h := &listenHandler{
				rabbitSock:      mockSock,
				reqHandler:      mockReq,
				customerHandler: mockCustomer,
			}

			mockCustomer.EXPECT().Gets(gomock.Any(), tt.size, tt.token).Return(tt.users, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}

func Test_processV1CustomersPost(t *testing.T) {

	tests := []struct {
		name    string
		request *rabbitmqhandler.Request

		username      string
		password      string
		userName      string
		detail        string
		webhookMethod customer.WebhookMethod
		webhookURI    string
		lineSecret    string
		lineToken     string
		permissionIDs []uuid.UUID

		customer  *customer.Customer
		expectRes *rabbitmqhandler.Response
	}{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:      "/v1/customers",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"username": "test", "password": "password", "name": "name test", "detail": "detail test", "webhook_method": "POST", "webhook_uri": "test.com", "line_secret": "335671d0-ed3f-11ec-95d5-bb7d97d73379", "line_token": "339c9ebc-ed3f-11ec-bb15-4f3b18e06796", "permission_ids": ["03796e14-7cb4-11ec-9dba-e72023efd1c6"]}`),
			},

			"test",
			"password",
			"name test",
			"detail test",
			customer.WebhookMethodPost,
			"test.com",
			"335671d0-ed3f-11ec-95d5-bb7d97d73379",
			"339c9ebc-ed3f-11ec-bb15-4f3b18e06796",
			[]uuid.UUID{
				permission.PermissionAdmin.ID,
			},

			&customer.Customer{
				ID:       uuid.FromStringOrNil("2043c49e-7db4-11ec-92b7-73af5ed663c9"),
				Username: "test",

				Name:          "name test",
				Detail:        "detail test",
				WebhookMethod: "POST",
				WebhookURI:    "test.com",
				LineSecret:    "335671d0-ed3f-11ec-95d5-bb7d97d73379",
				LineToken:     "339c9ebc-ed3f-11ec-bb15-4f3b18e06796",

				PermissionIDs: []uuid.UUID{
					permission.PermissionAdmin.ID,
				},
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"2043c49e-7db4-11ec-92b7-73af5ed663c9","username":"test","name":"name test","detail":"detail test","webhook_method":"POST","webhook_uri":"test.com","line_secret":"335671d0-ed3f-11ec-95d5-bb7d97d73379","line_token":"339c9ebc-ed3f-11ec-bb15-4f3b18e06796","permission_ids":["03796e14-7cb4-11ec-9dba-e72023efd1c6"],"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockCustomer := customerhandler.NewMockCustomerHandler(mc)

			h := &listenHandler{
				rabbitSock:      mockSock,
				reqHandler:      mockReq,
				customerHandler: mockCustomer,
			}

			mockCustomer.EXPECT().Create(gomock.Any(), tt.username, tt.password, tt.userName, tt.detail, tt.webhookMethod, tt.webhookURI, tt.lineSecret, tt.lineToken, tt.permissionIDs).Return(tt.customer, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}

func Test_processV1CustomersIDGet(t *testing.T) {

	tests := []struct {
		name    string
		request *rabbitmqhandler.Request

		id uuid.UUID

		customer  *customer.Customer
		expectRes *rabbitmqhandler.Response
	}{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:      "/v1/customers/2cfbb148-7dc7-11ec-85df-47cf2c8492f0",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: "application/json",
			},

			uuid.FromStringOrNil("2cfbb148-7dc7-11ec-85df-47cf2c8492f0"),

			&customer.Customer{
				ID:            uuid.FromStringOrNil("2cfbb148-7dc7-11ec-85df-47cf2c8492f0"),
				Username:      "test",
				PermissionIDs: []uuid.UUID{},
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"2cfbb148-7dc7-11ec-85df-47cf2c8492f0","username":"test","name":"","detail":"","webhook_method":"","webhook_uri":"","permission_ids":[],"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockCustomer := customerhandler.NewMockCustomerHandler(mc)

			h := &listenHandler{
				rabbitSock:      mockSock,
				reqHandler:      mockReq,
				customerHandler: mockCustomer,
			}

			mockCustomer.EXPECT().Get(gomock.Any(), tt.id).Return(tt.customer, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}

func Test_processV1UsersIDDelete(t *testing.T) {

	tests := []struct {
		name    string
		request *rabbitmqhandler.Request

		customerID       uuid.UUID
		responseCustomer *customer.Customer

		expectRes *rabbitmqhandler.Response
	}{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:      "/v1/customers/5071b05e-7dc8-11ec-9746-5f318f662852",
				Method:   rabbitmqhandler.RequestMethodDelete,
				DataType: "application/json",
			},

			uuid.FromStringOrNil("5071b05e-7dc8-11ec-9746-5f318f662852"),
			&customer.Customer{
				ID: uuid.FromStringOrNil("5071b05e-7dc8-11ec-9746-5f318f662852"),
			},

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"5071b05e-7dc8-11ec-9746-5f318f662852","username":"","name":"","detail":"","webhook_method":"","webhook_uri":"","permission_ids":null,"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockCustomer := customerhandler.NewMockCustomerHandler(mc)

			h := &listenHandler{
				rabbitSock:      mockSock,
				reqHandler:      mockReq,
				customerHandler: mockCustomer,
			}

			mockCustomer.EXPECT().Delete(gomock.Any(), tt.customerID).Return(tt.responseCustomer, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_processV1UsersIDPut(t *testing.T) {

	tests := []struct {
		name    string
		request *rabbitmqhandler.Request

		id            uuid.UUID
		userName      string
		detail        string
		webhookMethod customer.WebhookMethod
		webhookURI    string

		responseCustomer *customer.Customer

		expectRes *rabbitmqhandler.Response
	}{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:      "/v1/customers/5a8fac06-7dd4-11ec-b4e7-ab52242f6b29",
				Method:   rabbitmqhandler.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"name":"test2", "detail": "detail2", "webhook_method": "POST", "webhook_uri": "test.com"}`),
			},

			uuid.FromStringOrNil("5a8fac06-7dd4-11ec-b4e7-ab52242f6b29"),
			"test2",
			"detail2",
			customer.WebhookMethodPost,
			"test.com",

			&customer.Customer{
				ID: uuid.FromStringOrNil("5a8fac06-7dd4-11ec-b4e7-ab52242f6b29"),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"5a8fac06-7dd4-11ec-b4e7-ab52242f6b29","username":"","name":"","detail":"","webhook_method":"","webhook_uri":"","permission_ids":null,"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockCustomer := customerhandler.NewMockCustomerHandler(mc)

			h := &listenHandler{
				rabbitSock:      mockSock,
				reqHandler:      mockReq,
				customerHandler: mockCustomer,
			}

			mockCustomer.EXPECT().UpdateBasicInfo(gomock.Any(), tt.id, tt.userName, tt.detail, tt.webhookMethod, tt.webhookURI).Return(tt.responseCustomer, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}

func Test_processV1UsersIDPasswordPut(t *testing.T) {

	tests := []struct {
		name    string
		request *rabbitmqhandler.Request

		id       uuid.UUID
		password string

		responseCustomer *customer.Customer
		expectRes        *rabbitmqhandler.Response
	}{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:      "/v1/customers/1887f2d6-7dd5-11ec-9141-f7f46aaf294c/password",
				Method:   rabbitmqhandler.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"password":"password2"}`),
			},

			uuid.FromStringOrNil("1887f2d6-7dd5-11ec-9141-f7f46aaf294c"),
			"password2",

			&customer.Customer{
				ID: uuid.FromStringOrNil("1887f2d6-7dd5-11ec-9141-f7f46aaf294c"),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"1887f2d6-7dd5-11ec-9141-f7f46aaf294c","username":"","name":"","detail":"","webhook_method":"","webhook_uri":"","permission_ids":null,"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockCustomer := customerhandler.NewMockCustomerHandler(mc)

			h := &listenHandler{
				rabbitSock:      mockSock,
				reqHandler:      mockReq,
				customerHandler: mockCustomer,
			}

			mockCustomer.EXPECT().UpdatePassword(gomock.Any(), tt.id, tt.password).Return(tt.responseCustomer, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}

func Test_processV1CustomersIDPermissionIDsPut(t *testing.T) {

	tests := []struct {
		name    string
		request *rabbitmqhandler.Request

		id            uuid.UUID
		permissionIDs []uuid.UUID

		responseCustomer *customer.Customer
		expectRes        *rabbitmqhandler.Response
	}{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:      "/v1/customers/e00b4e98-7dd5-11ec-82c1-8b583557f04d/permission_ids",
				Method:   rabbitmqhandler.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"permission_ids":["03796e14-7cb4-11ec-9dba-e72023efd1c6"]}`),
			},

			uuid.FromStringOrNil("e00b4e98-7dd5-11ec-82c1-8b583557f04d"),
			[]uuid.UUID{
				permission.PermissionAdmin.ID,
			},

			&customer.Customer{
				ID: uuid.FromStringOrNil("e00b4e98-7dd5-11ec-82c1-8b583557f04d"),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"e00b4e98-7dd5-11ec-82c1-8b583557f04d","username":"","name":"","detail":"","webhook_method":"","webhook_uri":"","permission_ids":null,"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockCustomer := customerhandler.NewMockCustomerHandler(mc)

			h := &listenHandler{
				rabbitSock:      mockSock,
				reqHandler:      mockReq,
				customerHandler: mockCustomer,
			}

			mockCustomer.EXPECT().UpdatePermissionIDs(gomock.Any(), tt.id, tt.permissionIDs).Return(tt.responseCustomer, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_processV1CustomersIDLineInfoPut(t *testing.T) {

	tests := []struct {
		name    string
		request *rabbitmqhandler.Request

		id         uuid.UUID
		lineSecret string
		lineToken  string

		responseCustomer *customer.Customer
		expectRes        *rabbitmqhandler.Response
	}{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:      "/v1/customers/e00b4e98-7dd5-11ec-82c1-8b583557f04d/line_info",
				Method:   rabbitmqhandler.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"line_secret":"02d819cc-e61a-11ec-afad-83eb2bd7847b", "line_token":"0308b0c8-e61a-11ec-b2ff-1bf8777cc942"}`),
			},

			uuid.FromStringOrNil("e00b4e98-7dd5-11ec-82c1-8b583557f04d"),
			"02d819cc-e61a-11ec-afad-83eb2bd7847b",
			"0308b0c8-e61a-11ec-b2ff-1bf8777cc942",

			&customer.Customer{
				ID: uuid.FromStringOrNil("e00b4e98-7dd5-11ec-82c1-8b583557f04d"),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"e00b4e98-7dd5-11ec-82c1-8b583557f04d","username":"","name":"","detail":"","webhook_method":"","webhook_uri":"","permission_ids":null,"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockCustomer := customerhandler.NewMockCustomerHandler(mc)

			h := &listenHandler{
				rabbitSock:      mockSock,
				reqHandler:      mockReq,
				customerHandler: mockCustomer,
			}

			mockCustomer.EXPECT().UpdateLineInfo(gomock.Any(), tt.id, tt.lineSecret, tt.lineToken).Return(tt.responseCustomer, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
