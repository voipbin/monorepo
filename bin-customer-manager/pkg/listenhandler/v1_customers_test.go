package listenhandler

import (
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	bmbilling "gitlab.com/voipbin/bin-manager/billing-manager.git/models/billing"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"

	"gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	"gitlab.com/voipbin/bin-manager/customer-manager.git/pkg/customerhandler"
)

func Test_processV1CustomersGet(t *testing.T) {

	tests := []struct {
		name    string
		request *rabbitmqhandler.Request
		size    uint64
		token   string

		responseFilters   map[string]string
		responseCustomers []*customer.Customer
		expectRes         *rabbitmqhandler.Response
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

			map[string]string{
				"deleted": "false",
			},
			[]*customer.Customer{
				{
					ID:       uuid.FromStringOrNil("31b08066-7db2-11ec-8786-c7d9cf6c9b5f"),
					TMCreate: "",
					TMUpdate: "",
					TMDelete: "",
				},
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"31b08066-7db2-11ec-8786-c7d9cf6c9b5f","billing_account_id":"00000000-0000-0000-0000-000000000000"}]`),
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

			map[string]string{
				"deleted": "false",
			},
			[]*customer.Customer{
				{
					ID: uuid.FromStringOrNil("9f8a7880-7db2-11ec-9602-930411a1581f"),
				},
				{
					ID: uuid.FromStringOrNil("a032c710-7db2-11ec-bfe0-83fa85a82603"),
				},
			},
			&rabbitmqhandler.Response{
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

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockCustomer := customerhandler.NewMockCustomerHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)

			h := &listenHandler{
				rabbitSock:      mockSock,
				reqHandler:      mockReq,
				utilHandler:     mockUtil,
				customerHandler: mockCustomer,
			}

			mockUtil.EXPECT().URLParseFilters(gomock.Any()).Return(tt.responseFilters)
			mockCustomer.EXPECT().Gets(gomock.Any(), tt.size, tt.token, tt.responseFilters).Return(tt.responseCustomers, nil)

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

		customerName     string
		detail           string
		email            string
		phoneNumber      string
		address          string
		webhookMethod    customer.WebhookMethod
		webhookURI       string
		billingAccountID uuid.UUID

		responseCustomer *customer.Customer
		expectRes        *rabbitmqhandler.Response
	}{
		{
			name: "normal",
			request: &rabbitmqhandler.Request{
				URI:      "/v1/customers",
				Method:   rabbitmqhandler.RequestMethodPost,
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
			expectRes: &rabbitmqhandler.Response{
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

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockCustomer := customerhandler.NewMockCustomerHandler(mc)

			h := &listenHandler{
				rabbitSock:      mockSock,
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
				ID: uuid.FromStringOrNil("2cfbb148-7dc7-11ec-85df-47cf2c8492f0"),
			},
			&rabbitmqhandler.Response{
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
				Data:       []byte(`{"id":"5071b05e-7dc8-11ec-9746-5f318f662852","billing_account_id":"00000000-0000-0000-0000-000000000000"}`),
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
		email         string
		phoneNumber   string
		address       string
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
			&rabbitmqhandler.Response{
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

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockCustomer := customerhandler.NewMockCustomerHandler(mc)

			h := &listenHandler{
				rabbitSock:      mockSock,
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
		request *rabbitmqhandler.Request

		id            uuid.UUID
		referenceType bmbilling.ReferenceType
		country       string
		count         int

		responseValid bool
		expectRes     *rabbitmqhandler.Response
	}{
		{
			name: "normal",
			request: &rabbitmqhandler.Request{
				URI:      "/v1/customers/dd74462c-0e88-11ee-a276-dbfe542e4ab0/is_valid_balance",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: requesthandler.ContentTypeJSON,
				Data:     []byte(`{"reference_type":"call","country":"us","count":3}`),
			},

			id:            uuid.FromStringOrNil("dd74462c-0e88-11ee-a276-dbfe542e4ab0"),
			referenceType: bmbilling.ReferenceTypeCall,
			country:       "us",
			count:         3,

			responseValid: true,
			expectRes: &rabbitmqhandler.Response{
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

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockCustomer := customerhandler.NewMockCustomerHandler(mc)

			h := &listenHandler{
				rabbitSock:      mockSock,
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
		request *rabbitmqhandler.Request

		id               uuid.UUID
		billingAccountID uuid.UUID

		responseCustomer *customer.Customer
		expectRes        *rabbitmqhandler.Response
	}{
		{
			name: "normal",
			request: &rabbitmqhandler.Request{
				URI:      "/v1/customers/cc2cedde-0f90-11ee-a04e-1723e1a00731/billing_account_id",
				Method:   rabbitmqhandler.RequestMethodPut,
				DataType: requesthandler.ContentTypeJSON,
				Data:     []byte(`{"billing_account_id":"ccdb52d4-0f90-11ee-9ba9-ab40a2fc7434"}`),
			},

			id:               uuid.FromStringOrNil("cc2cedde-0f90-11ee-a04e-1723e1a00731"),
			billingAccountID: uuid.FromStringOrNil("ccdb52d4-0f90-11ee-9ba9-ab40a2fc7434"),

			responseCustomer: &customer.Customer{
				ID: uuid.FromStringOrNil("cc2cedde-0f90-11ee-a04e-1723e1a00731"),
			},
			expectRes: &rabbitmqhandler.Response{
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

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockCustomer := customerhandler.NewMockCustomerHandler(mc)

			h := &listenHandler{
				rabbitSock:      mockSock,
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
