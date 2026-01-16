package listenhandler

import (
	reflect "reflect"
	"testing"

	bmbilling "monorepo/bin-billing-manager/models/billing"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-customer-manager/models/customer"
	"monorepo/bin-customer-manager/pkg/customerhandler"
)

func Test_processV1CustomersGet(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request
		size    uint64
		token   string

		expectFilters     map[customer.Field]any
		responseCustomers []*customer.Customer
		expectRes         *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/customers?page_size=10&page_token=2021-11-23%2017:55:39.712000",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
				Data:     []byte(`{"deleted":false}`),
			},
			10,
			"2021-11-23 17:55:39.712000",

			map[customer.Field]any{
				customer.FieldDeleted: false,
			},
			[]*customer.Customer{
				{
					ID:       uuid.FromStringOrNil("31b08066-7db2-11ec-8786-c7d9cf6c9b5f"),
					TMCreate: "",
					TMUpdate: "",
					TMDelete: "",
				},
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"31b08066-7db2-11ec-8786-c7d9cf6c9b5f","billing_account_id":"00000000-0000-0000-0000-000000000000"}]`),
			},
		},
		{
			"2 customers",
			&sock.Request{
				URI:      "/v1/customers?page_size=10&page_token=2021-11-23%2017:55:39.712000",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
				Data:     []byte(`{"deleted":false}`),
			},
			10,
			"2021-11-23 17:55:39.712000",

			map[customer.Field]any{
				customer.FieldDeleted: false,
			},
			[]*customer.Customer{
				{
					ID: uuid.FromStringOrNil("9f8a7880-7db2-11ec-9602-930411a1581f"),
				},
				{
					ID: uuid.FromStringOrNil("a032c710-7db2-11ec-bfe0-83fa85a82603"),
				},
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"9f8a7880-7db2-11ec-9602-930411a1581f","billing_account_id":"00000000-0000-0000-0000-000000000000"},{"id":"a032c710-7db2-11ec-bfe0-83fa85a82603","billing_account_id":"00000000-0000-0000-0000-000000000000"}]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockCustomer := customerhandler.NewMockCustomerHandler(mc)

			h := &listenHandler{
				sockHandler:     mockSock,
				reqHandler:      mockReq,
				customerHandler: mockCustomer,
			}

			mockCustomer.EXPECT().List(gomock.Any(), tt.size, tt.token, gomock.Any()).Return(tt.responseCustomers, nil)

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
		request *sock.Request

		customerName     string
		detail           string
		email            string
		phoneNumber      string
		address          string
		webhookMethod    customer.WebhookMethod
		webhookURI       string
		billingAccountID uuid.UUID

		responseCustomer *customer.Customer
		expectRes        *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/customers",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"username": "test", "password": "password", "name": "name test", "detail": "detail test", "email": "test@test.com", "phone_number": "+821100000001", "address": "somewhere", "webhook_method": "POST", "webhook_uri": "test.com", "line_secret": "335671d0-ed3f-11ec-95d5-bb7d97d73379", "line_token": "339c9ebc-ed3f-11ec-bb15-4f3b18e06796", "permission_ids": ["03796e14-7cb4-11ec-9dba-e72023efd1c6"],"billing_account_id":"57e13956-0e84-11ee-886f-972ac028efa9"}`),
			},

			customerName:     "name test",
			detail:           "detail test",
			email:            "test@test.com",
			phoneNumber:      "+821100000001",
			address:          "somewhere",
			webhookMethod:    customer.WebhookMethodPost,
			webhookURI:       "test.com",
			billingAccountID: uuid.FromStringOrNil("57e13956-0e84-11ee-886f-972ac028efa9"),

			responseCustomer: &customer.Customer{
				ID: uuid.FromStringOrNil("2043c49e-7db4-11ec-92b7-73af5ed663c9"),

				Name:          "name test",
				Detail:        "detail test",
				Email:         "test@test.com",
				PhoneNumber:   "+821100000001",
				Address:       "somewhere",
				WebhookMethod: "POST",
				WebhookURI:    "test.com",

				BillingAccountID: uuid.FromStringOrNil("57e13956-0e84-11ee-886f-972ac028efa9"),
			},
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"2043c49e-7db4-11ec-92b7-73af5ed663c9","name":"name test","detail":"detail test","email":"test@test.com","phone_number":"+821100000001","address":"somewhere","webhook_method":"POST","webhook_uri":"test.com","billing_account_id":"57e13956-0e84-11ee-886f-972ac028efa9"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockCustomer := customerhandler.NewMockCustomerHandler(mc)

			h := &listenHandler{
				sockHandler:     mockSock,
				reqHandler:      mockReq,
				customerHandler: mockCustomer,
			}

			mockCustomer.EXPECT().Create(
				gomock.Any(),
				tt.customerName,
				tt.detail,
				tt.email,
				tt.phoneNumber,
				tt.address,
				tt.webhookMethod,
				tt.webhookURI,
			).Return(tt.responseCustomer, nil)

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
		request *sock.Request

		id uuid.UUID

		customer  *customer.Customer
		expectRes *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/customers/2cfbb148-7dc7-11ec-85df-47cf2c8492f0",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
			},

			uuid.FromStringOrNil("2cfbb148-7dc7-11ec-85df-47cf2c8492f0"),

			&customer.Customer{
				ID: uuid.FromStringOrNil("2cfbb148-7dc7-11ec-85df-47cf2c8492f0"),
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"2cfbb148-7dc7-11ec-85df-47cf2c8492f0","billing_account_id":"00000000-0000-0000-0000-000000000000"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockCustomer := customerhandler.NewMockCustomerHandler(mc)

			h := &listenHandler{
				sockHandler:     mockSock,
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

func Test_processV1CustomersIDDelete(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		customerID       uuid.UUID
		responseCustomer *customer.Customer

		expectRes *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/customers/5071b05e-7dc8-11ec-9746-5f318f662852",
				Method:   sock.RequestMethodDelete,
				DataType: "application/json",
			},

			uuid.FromStringOrNil("5071b05e-7dc8-11ec-9746-5f318f662852"),
			&customer.Customer{
				ID: uuid.FromStringOrNil("5071b05e-7dc8-11ec-9746-5f318f662852"),
			},

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"5071b05e-7dc8-11ec-9746-5f318f662852","billing_account_id":"00000000-0000-0000-0000-000000000000"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockCustomer := customerhandler.NewMockCustomerHandler(mc)

			h := &listenHandler{
				sockHandler:     mockSock,
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

func Test_processV1CustomersIDPut(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		id            uuid.UUID
		userName      string
		detail        string
		email         string
		phoneNumber   string
		address       string
		webhookMethod customer.WebhookMethod
		webhookURI    string

		responseCustomer *customer.Customer

		expectRes *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/customers/5a8fac06-7dd4-11ec-b4e7-ab52242f6b29",
				Method:   sock.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"name":"test2", "detail": "detail2", "email": "update@test.com", "phone_number": "+821100000001", "address": "update address", "webhook_method": "POST", "webhook_uri": "test.com"}`),
			},

			uuid.FromStringOrNil("5a8fac06-7dd4-11ec-b4e7-ab52242f6b29"),
			"test2",
			"detail2",
			"update@test.com",
			"+821100000001",
			"update address",
			customer.WebhookMethodPost,
			"test.com",

			&customer.Customer{
				ID: uuid.FromStringOrNil("5a8fac06-7dd4-11ec-b4e7-ab52242f6b29"),
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"5a8fac06-7dd4-11ec-b4e7-ab52242f6b29","billing_account_id":"00000000-0000-0000-0000-000000000000"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockCustomer := customerhandler.NewMockCustomerHandler(mc)

			h := &listenHandler{
				sockHandler:     mockSock,
				reqHandler:      mockReq,
				customerHandler: mockCustomer,
			}

			mockCustomer.EXPECT().UpdateBasicInfo(gomock.Any(), tt.id, tt.userName, tt.detail, tt.email, tt.phoneNumber, tt.address, tt.webhookMethod, tt.webhookURI).Return(tt.responseCustomer, nil)
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

func Test_processV1CustomersIDIsValidBalance(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		id            uuid.UUID
		referenceType bmbilling.ReferenceType
		country       string
		count         int

		responseValid bool
		expectRes     *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/customers/dd74462c-0e88-11ee-a276-dbfe542e4ab0/is_valid_balance",
				Method:   sock.RequestMethodPost,
				DataType: requesthandler.ContentTypeJSON,
				Data:     []byte(`{"reference_type":"call","country":"us","count":3}`),
			},

			id:            uuid.FromStringOrNil("dd74462c-0e88-11ee-a276-dbfe542e4ab0"),
			referenceType: bmbilling.ReferenceTypeCall,
			country:       "us",
			count:         3,

			responseValid: true,
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"valid":true}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockCustomer := customerhandler.NewMockCustomerHandler(mc)

			h := &listenHandler{
				sockHandler:     mockSock,
				reqHandler:      mockReq,
				customerHandler: mockCustomer,
			}

			mockCustomer.EXPECT().IsValidBalance(gomock.Any(), tt.id, tt.referenceType, tt.country, tt.count).Return(tt.responseValid, nil)
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

func Test_processV1CustomersIDBillingAccountIDPut(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		id               uuid.UUID
		billingAccountID uuid.UUID

		responseCustomer *customer.Customer
		expectRes        *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/customers/cc2cedde-0f90-11ee-a04e-1723e1a00731/billing_account_id",
				Method:   sock.RequestMethodPut,
				DataType: requesthandler.ContentTypeJSON,
				Data:     []byte(`{"billing_account_id":"ccdb52d4-0f90-11ee-9ba9-ab40a2fc7434"}`),
			},

			id:               uuid.FromStringOrNil("cc2cedde-0f90-11ee-a04e-1723e1a00731"),
			billingAccountID: uuid.FromStringOrNil("ccdb52d4-0f90-11ee-9ba9-ab40a2fc7434"),

			responseCustomer: &customer.Customer{
				ID: uuid.FromStringOrNil("cc2cedde-0f90-11ee-a04e-1723e1a00731"),
			},
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"cc2cedde-0f90-11ee-a04e-1723e1a00731","billing_account_id":"00000000-0000-0000-0000-000000000000"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockCustomer := customerhandler.NewMockCustomerHandler(mc)

			h := &listenHandler{
				sockHandler:     mockSock,
				reqHandler:      mockReq,
				customerHandler: mockCustomer,
			}

			mockCustomer.EXPECT().UpdateBillingAccountID(gomock.Any(), tt.id, tt.billingAccountID).Return(tt.responseCustomer, nil)
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
