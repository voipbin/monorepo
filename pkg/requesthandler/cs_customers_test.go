package requesthandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

func Test_CSV1CustomerGets(t *testing.T) {

	tests := []struct {
		name string

		pageToken string
		pageSize  uint64

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response

		expectRes []cscustomer.Customer
	}{
		{
			"normal",

			"2021-03-02 03:23:20.995000",
			10,

			"bin-manager.customer-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/customers?page_token=2021-03-02+03%3A23%3A20.995000&page_size=10",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: ContentTypeJSON,
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"30071608-7e43-11ec-b04a-bb4270e3e223","username":"test1","name":"test user 1","detail":"test user 1 detail","permission_ids":[]},{"id":"5ca81a9a-7e43-11ec-b271-5b65823bfdd3","username":"test2","name":"test user 2","detail":"test user 2 detail","permission_ids":[]}]`),
			},
			[]cscustomer.Customer{
				{
					ID:            uuid.FromStringOrNil("30071608-7e43-11ec-b04a-bb4270e3e223"),
					Username:      "test1",
					Name:          "test user 1",
					Detail:        "test user 1 detail",
					PermissionIDs: []uuid.UUID{},
				},
				{
					ID:            uuid.FromStringOrNil("5ca81a9a-7e43-11ec-b271-5b65823bfdd3"),
					Username:      "test2",
					PasswordHash:  "",
					Name:          "test user 2",
					Detail:        "test user 2 detail",
					PermissionIDs: []uuid.UUID{},
				},
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

			res, err := reqHandler.CSV1CustomerGets(ctx, tt.pageToken, tt.pageSize)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_CSV1CustomerGet(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response

		expectRes *cscustomer.Customer
	}{
		{
			"normal",

			uuid.FromStringOrNil("951a4038-7e43-11ec-bc59-4f1dc0de20b0"),

			"bin-manager.customer-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/customers/951a4038-7e43-11ec-bc59-4f1dc0de20b0",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: ContentTypeJSON,
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"951a4038-7e43-11ec-bc59-4f1dc0de20b0","username":"test1","name":"test user 1","detail":"test user 1 detail","permission_ids":[]}`),
			},
			&cscustomer.Customer{
				ID:            uuid.FromStringOrNil("951a4038-7e43-11ec-bc59-4f1dc0de20b0"),
				Username:      "test1",
				Name:          "test user 1",
				Detail:        "test user 1 detail",
				PermissionIDs: []uuid.UUID{},
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

			res, err := reqHandler.CSV1CustomerGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_CSV1CustomerDelete(t *testing.T) {

	tests := []struct {
		name string

		customerID uuid.UUID

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response

		expectRes *cscustomer.Customer
	}{
		{
			"normal",

			uuid.FromStringOrNil("d6afec8c-7e43-11ec-ab03-ff394ae04b39"),

			"bin-manager.customer-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/customers/d6afec8c-7e43-11ec-ab03-ff394ae04b39",
				Method:   rabbitmqhandler.RequestMethodDelete,
				DataType: ContentTypeJSON,
			},
			&rabbitmqhandler.Response{
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

			res, err := reqHandler.CSV1CustomerDelete(ctx, tt.customerID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_CSV1CustomerCreate(t *testing.T) {

	tests := []struct {
		name string

		username      string
		password      string
		userName      string
		detail        string
		webhookMethod cscustomer.WebhookMethod
		webhookURI    string
		lineSecret    string
		lineToken     string
		permissionIDs []uuid.UUID

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response

		expectRes *cscustomer.Customer
	}{
		{
			"normal",

			"test1",
			"testpassword",
			"test1",
			"detail1",
			cscustomer.WebhookMethodPost,
			"test.com",
			"f5cb47ec-ed42-11ec-b4a5-cfb1a63e302d",
			"f6003650-ed42-11ec-8821-93d06ca99641",
			[]uuid.UUID{
				uuid.FromStringOrNil("db0e2c52-7e44-11ec-811d-ab2fbb79302a"),
			},

			"bin-manager.customer-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/customers",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"username":"test1","password":"testpassword","name":"test1","detail":"detail1","webhook_method":"POST","webhook_uri":"test.com","line_secret":"f5cb47ec-ed42-11ec-b4a5-cfb1a63e302d","line_token":"f6003650-ed42-11ec-8821-93d06ca99641","permission_ids":["db0e2c52-7e44-11ec-811d-ab2fbb79302a"]}`),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"e46cbfd4-7e44-11ec-b7de-a7cfacf1121f","username":"test1","name":"test1","detail":"detail1","webhook_method":"POST","webhook_uri":"test.com","line_secret":"f5cb47ec-ed42-11ec-b4a5-cfb1a63e302d","line_token":"f6003650-ed42-11ec-8821-93d06ca99641","permission_ids":["db0e2c52-7e44-11ec-811d-ab2fbb79302a"]}`),
			},
			&cscustomer.Customer{
				ID:            uuid.FromStringOrNil("e46cbfd4-7e44-11ec-b7de-a7cfacf1121f"),
				Username:      "test1",
				Name:          "test1",
				Detail:        "detail1",
				WebhookMethod: cscustomer.WebhookMethodPost,
				WebhookURI:    "test.com",
				LineSecret:    "f5cb47ec-ed42-11ec-b4a5-cfb1a63e302d",
				LineToken:     "f6003650-ed42-11ec-8821-93d06ca99641",
				PermissionIDs: []uuid.UUID{
					uuid.FromStringOrNil("db0e2c52-7e44-11ec-811d-ab2fbb79302a"),
				},
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

			res, err := reqHandler.CSV1CustomerCreate(
				ctx,
				requestTimeoutDefault,
				tt.username,
				tt.password,
				tt.userName,
				tt.detail,
				tt.webhookMethod,
				tt.webhookURI,
				tt.lineSecret,
				tt.lineToken,
				tt.permissionIDs,
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

func Test_CSV1CustomerUpdateBasicInfo(t *testing.T) {

	tests := []struct {
		name string

		id            uuid.UUID
		userName      string
		detail        string
		webhookMethod cscustomer.WebhookMethod
		webhookURI    string

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response
		expectRes     *cscustomer.Customer
	}{
		{
			"normal",

			uuid.FromStringOrNil("eed8e316-7e45-11ec-bcac-97541487f2c1"),
			"test1",
			"detail1",
			cscustomer.WebhookMethodPost,
			"test.com",

			"bin-manager.customer-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/customers/eed8e316-7e45-11ec-bcac-97541487f2c1",
				Method:   rabbitmqhandler.RequestMethodPut,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"name":"test1","detail":"detail1","webhook_method":"POST","webhook_uri":"test.com"}`),
			},
			&rabbitmqhandler.Response{
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

			res, err := reqHandler.CSV1CustomerUpdate(ctx, tt.id, tt.userName, tt.detail, tt.webhookMethod, tt.webhookURI)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_CSV1CustomerUpdatePassword(t *testing.T) {

	tests := []struct {
		name string

		id       uuid.UUID
		password string

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response
		expectRes     *cscustomer.Customer
	}{
		{
			"normal",

			uuid.FromStringOrNil("3f54d1ec-7e46-11ec-bc5a-8f233b20baaf"),
			"password1",

			"bin-manager.customer-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/customers/3f54d1ec-7e46-11ec-bc5a-8f233b20baaf/password",
				Method:   rabbitmqhandler.RequestMethodPut,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"password":"password1"}`),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"3f54d1ec-7e46-11ec-bc5a-8f233b20baaf"}`),
			},
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("3f54d1ec-7e46-11ec-bc5a-8f233b20baaf"),
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

			res, err := reqHandler.CSV1CustomerUpdatePassword(ctx, requestTimeoutDefault, tt.id, tt.password)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_CSV1CustomerUpdatePermission(t *testing.T) {

	tests := []struct {
		name string

		id            uuid.UUID
		permissionIDs []uuid.UUID

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response
		expectRes     *cscustomer.Customer
	}{
		{
			"normal",

			uuid.FromStringOrNil("af7c2c04-7e46-11ec-845e-ebf1194dd77a"),
			[]uuid.UUID{
				uuid.FromStringOrNil("be48c440-7e46-11ec-82c9-2fac536f93e0"),
			},

			"bin-manager.customer-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/customers/af7c2c04-7e46-11ec-845e-ebf1194dd77a/permission_ids",
				Method:   rabbitmqhandler.RequestMethodPut,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"permission_ids":["be48c440-7e46-11ec-82c9-2fac536f93e0"]}`),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"af7c2c04-7e46-11ec-845e-ebf1194dd77a"}`),
			},
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("af7c2c04-7e46-11ec-845e-ebf1194dd77a"),
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

			res, err := reqHandler.CSV1CustomerUpdatePermissionIDs(ctx, tt.id, tt.permissionIDs)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}

		})
	}
}
