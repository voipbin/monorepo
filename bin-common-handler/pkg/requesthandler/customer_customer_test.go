package requesthandler

import (
	"context"
	"reflect"
	"testing"

	bmbilling "monorepo/bin-billing-manager/models/billing"
	cscustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
)

func Test_CustomerV1CustomerGets(t *testing.T) {

	tests := []struct {
		name string

		pageToken string
		pageSize  uint64
		filters   map[string]string

		expectURL     string
		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response

		expectRes []cscustomer.Customer
	}{
		{
			"normal",

			"2021-03-02 03:23:20.995000",
			10,
			map[string]string{
				"deleted": "false",
			},

			"/v1/customers?page_token=2021-03-02+03%3A23%3A20.995000&page_size=10",
			"bin-manager.customer-manager.request",
			&sock.Request{
				URI:      "/v1/customers?page_token=2021-03-02+03%3A23%3A20.995000&page_size=10&filter_deleted=false",
				Method:   sock.RequestMethodGet,
				DataType: ContentTypeJSON,
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"30071608-7e43-11ec-b04a-bb4270e3e223","username":"test1","name":"test user 1","detail":"test user 1 detail","permission_ids":[]},{"id":"5ca81a9a-7e43-11ec-b271-5b65823bfdd3","username":"test2","name":"test user 2","detail":"test user 2 detail","permission_ids":[]}]`),
			},
			[]cscustomer.Customer{
				{
					ID:     uuid.FromStringOrNil("30071608-7e43-11ec-b04a-bb4270e3e223"),
					Name:   "test user 1",
					Detail: "test user 1 detail",
				},
				{
					ID:     uuid.FromStringOrNil("5ca81a9a-7e43-11ec-b271-5b65823bfdd3"),
					Name:   "test user 2",
					Detail: "test user 2 detail",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			h := requestHandler{
				sock:        mockSock,
				utilHandler: mockUtil,
			}
			ctx := context.Background()

			mockUtil.EXPECT().URLMergeFilters(tt.expectURL, tt.filters).Return(utilhandler.URLMergeFilters(tt.expectURL, tt.filters))
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := h.CustomerV1CustomerGets(ctx, tt.pageToken, tt.pageSize, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_CustomerV1CustomerGet(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response

		expectRes *cscustomer.Customer
	}{
		{
			"normal",

			uuid.FromStringOrNil("951a4038-7e43-11ec-bc59-4f1dc0de20b0"),

			"bin-manager.customer-manager.request",
			&sock.Request{
				URI:      "/v1/customers/951a4038-7e43-11ec-bc59-4f1dc0de20b0",
				Method:   sock.RequestMethodGet,
				DataType: ContentTypeJSON,
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"951a4038-7e43-11ec-bc59-4f1dc0de20b0","username":"test1","name":"test user 1","detail":"test user 1 detail","permission_ids":[]}`),
			},
			&cscustomer.Customer{
				ID:     uuid.FromStringOrNil("951a4038-7e43-11ec-bc59-4f1dc0de20b0"),
				Name:   "test user 1",
				Detail: "test user 1 detail",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()

			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.CustomerV1CustomerGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_CustomerV1CustomerDelete(t *testing.T) {

	tests := []struct {
		name string

		customerID uuid.UUID

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response

		expectRes *cscustomer.Customer
	}{
		{
			"normal",

			uuid.FromStringOrNil("d6afec8c-7e43-11ec-ab03-ff394ae04b39"),

			"bin-manager.customer-manager.request",
			&sock.Request{
				URI:      "/v1/customers/d6afec8c-7e43-11ec-ab03-ff394ae04b39",
				Method:   sock.RequestMethodDelete,
				DataType: ContentTypeJSON,
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"d6afec8c-7e43-11ec-ab03-ff394ae04b39"}`),
			},

			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("d6afec8c-7e43-11ec-ab03-ff394ae04b39"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()

			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.CustomerV1CustomerDelete(ctx, tt.customerID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_CustomerV1CustomerCreate(t *testing.T) {

	tests := []struct {
		name string

		userName      string
		detail        string
		email         string
		phoneNumber   string
		address       string
		webhookMethod cscustomer.WebhookMethod
		webhookURI    string

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response

		expectRes *cscustomer.Customer
	}{
		{
			"normal",

			"test1",
			"detail1",
			"test@test.com",
			"+821100000001",
			"somewhere",
			cscustomer.WebhookMethodPost,
			"test.com",

			"bin-manager.customer-manager.request",
			&sock.Request{
				URI:      "/v1/customers",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"name":"test1","detail":"detail1","email":"test@test.com","phone_number":"+821100000001","address":"somewhere","webhook_method":"POST","webhook_uri":"test.com"}`),
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"47943ef0-cb2f-11ee-adbd-136bc293c7e1"}`),
			},
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("47943ef0-cb2f-11ee-adbd-136bc293c7e1"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()

			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.CustomerV1CustomerCreate(
				ctx,
				requestTimeoutDefault,
				tt.userName,
				tt.detail,
				tt.email,
				tt.phoneNumber,
				tt.address,
				tt.webhookMethod,
				tt.webhookURI,
			)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_CustomerV1CustomerUpdateBasicInfo(t *testing.T) {

	tests := []struct {
		name string

		id            uuid.UUID
		userName      string
		detail        string
		email         string
		phoneNumber   string
		address       string
		webhookMethod cscustomer.WebhookMethod
		webhookURI    string

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response
		expectRes     *cscustomer.Customer
	}{
		{
			"normal",

			uuid.FromStringOrNil("eed8e316-7e45-11ec-bcac-97541487f2c1"),
			"test1",
			"detail1",
			"test@test.com",
			"+821100000001",
			"somewhere",
			cscustomer.WebhookMethodPost,
			"test.com",

			"bin-manager.customer-manager.request",
			&sock.Request{
				URI:      "/v1/customers/eed8e316-7e45-11ec-bcac-97541487f2c1",
				Method:   sock.RequestMethodPut,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"name":"test1","detail":"detail1","email":"test@test.com","phone_number":"+821100000001","address":"somewhere","webhook_method":"POST","webhook_uri":"test.com"}`),
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"eed8e316-7e45-11ec-bcac-97541487f2c1"}`),
			},
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("eed8e316-7e45-11ec-bcac-97541487f2c1"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()

			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.CustomerV1CustomerUpdate(ctx, tt.id, tt.userName, tt.detail, tt.email, tt.phoneNumber, tt.address, tt.webhookMethod, tt.webhookURI)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_CustomerV1CustomerIsValidBalance(t *testing.T) {

	tests := []struct {
		name string

		customerID    uuid.UUID
		referenceType bmbilling.ReferenceType
		country       string
		count         int

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response
		expectRes     bool
	}{
		{
			name: "normal",

			customerID:    uuid.FromStringOrNil("57e0d56e-0f8e-11ee-a32d-4b65fba800d5"),
			referenceType: bmbilling.ReferenceTypeCall,
			country:       "us",
			count:         3,

			expectTarget: "bin-manager.customer-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/customers/57e0d56e-0f8e-11ee-a32d-4b65fba800d5/is_valid_balance",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"reference_type":"call","country":"us","count":3}`),
			},
			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"valid":true}`),
			},
			expectRes: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()

			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.CustomerV1CustomerIsValidBalance(ctx, tt.customerID, tt.referenceType, tt.country, tt.count)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}

		})
	}
}

func Test_CustomerV1CustomerUpdateBillingAccountID(t *testing.T) {

	tests := []struct {
		name string

		id               uuid.UUID
		billingAccountID uuid.UUID

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response
		expectRes     *cscustomer.Customer
	}{
		{
			"normal",

			uuid.FromStringOrNil("2935091e-0f94-11ee-a5e5-a34227ad44a6"),
			uuid.FromStringOrNil("296b4aba-0f94-11ee-99c9-ab67bb9c534a"),

			"bin-manager.customer-manager.request",
			&sock.Request{
				URI:      "/v1/customers/2935091e-0f94-11ee-a5e5-a34227ad44a6/billing_account_id",
				Method:   sock.RequestMethodPut,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"billing_account_id":"296b4aba-0f94-11ee-99c9-ab67bb9c534a"}`),
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"2935091e-0f94-11ee-a5e5-a34227ad44a6"}`),
			},
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("2935091e-0f94-11ee-a5e5-a34227ad44a6"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()

			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.CustomerV1CustomerUpdateBillingAccountID(ctx, tt.id, tt.billingAccountID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}

		})
	}
}
