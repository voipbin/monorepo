package requesthandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	bmaccount "gitlab.com/voipbin/bin-manager/billing-manager.git/models/account"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

func Test_BillingV1AccountGets(t *testing.T) {

	tests := []struct {
		name string

		size  uint64
		token string

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectRes     []bmaccount.Account
		response      *rabbitmqhandler.Response
	}{
		{
			name: "normal",

			size:         10,
			token:        "2023-06-08 03:22:17.995000",
			expectTarget: "bin-manager.billing-manager.request",
			expectRequest: &rabbitmqhandler.Request{
				URI:    "/v1/accounts?page_token=2023-06-08+03%3A22%3A17.995000&page_size=10",
				Method: rabbitmqhandler.RequestMethodGet,
			},
			expectRes: []bmaccount.Account{
				{
					ID: uuid.FromStringOrNil("022bfc94-0b9b-11ee-8ea1-f3e4fbd66309"),
				},
				{
					ID: uuid.FromStringOrNil("025e6814-0b9b-11ee-8e8d-93e70b8939a0"),
				},
			},

			response: &rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"022bfc94-0b9b-11ee-8ea1-f3e4fbd66309"},{"id":"025e6814-0b9b-11ee-8e8d-93e70b8939a0"}]`),
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

			res, err := reqHandler.BillingV1AccountGets(ctx, tt.token, tt.size)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}

		})
	}
}

func Test_BillingV1AccountGet(t *testing.T) {

	tests := []struct {
		name string

		accountID uuid.UUID

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectRes     *bmaccount.Account
		response      *rabbitmqhandler.Response
	}{
		{
			name: "normal",

			accountID: uuid.FromStringOrNil("9000392c-0b9b-11ee-aa1d-8b84b3626bc7"),

			expectTarget: "bin-manager.billing-manager.request",
			expectRequest: &rabbitmqhandler.Request{
				URI:    "/v1/accounts/9000392c-0b9b-11ee-aa1d-8b84b3626bc7",
				Method: rabbitmqhandler.RequestMethodGet,
			},
			expectRes: &bmaccount.Account{
				ID: uuid.FromStringOrNil("9000392c-0b9b-11ee-aa1d-8b84b3626bc7"),
			},

			response: &rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"9000392c-0b9b-11ee-aa1d-8b84b3626bc7"}`),
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

			res, err := reqHandler.BillingV1AccountGet(ctx, tt.accountID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}

		})
	}
}

func Test_BillingV1AccountIsValidBalanceByCustomerID(t *testing.T) {

	tests := []struct {
		name string

		customerID uuid.UUID

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectRes     bool
		response      *rabbitmqhandler.Response
	}{
		{
			name: "normal",

			customerID: uuid.FromStringOrNil("1fa789f8-0b93-11ee-95d8-2356e5d027fa"),

			expectTarget: "bin-manager.billing-manager.request",
			expectRequest: &rabbitmqhandler.Request{
				URI:    "/v1/accounts/customer_id/1fa789f8-0b93-11ee-95d8-2356e5d027fa/is_valid_balance",
				Method: rabbitmqhandler.RequestMethodPost,
			},
			expectRes: true,

			response: &rabbitmqhandler.Response{
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
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.BillingV1AccountIsValidBalanceByCustomerID(ctx, tt.customerID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}

		})
	}
}

func Test_BillingV1AccountGetByCustomerID(t *testing.T) {

	tests := []struct {
		name string

		customerID uuid.UUID

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectRes     *bmaccount.Account

		response *rabbitmqhandler.Response
	}{
		{
			name: "normal",

			customerID: uuid.FromStringOrNil("2553921e-0b95-11ee-9ca9-5baa145aa2d0"),

			expectTarget: "bin-manager.billing-manager.request",
			expectRequest: &rabbitmqhandler.Request{
				URI:    "/v1/accounts/customer_id/2553921e-0b95-11ee-9ca9-5baa145aa2d0",
				Method: rabbitmqhandler.RequestMethodGet,
			},
			expectRes: &bmaccount.Account{
				ID: uuid.FromStringOrNil("2553921e-0b95-11ee-9ca9-5baa145aa2d0"),
			},

			response: &rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"2553921e-0b95-11ee-9ca9-5baa145aa2d0"}`),
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

			res, err := reqHandler.BillingV1AccountGetByCustomerID(ctx, tt.customerID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}

		})
	}
}

func Test_BillingV1AccountAddBalance(t *testing.T) {

	tests := []struct {
		name string

		accountID uuid.UUID
		balance   float32

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectRes     *bmaccount.Account

		response *rabbitmqhandler.Response
	}{
		{
			name: "normal",

			accountID: uuid.FromStringOrNil("79403360-0dbf-11ee-b1ad-c3eebc4a6196"),
			balance:   20,

			expectTarget: "bin-manager.billing-manager.request",
			expectRequest: &rabbitmqhandler.Request{
				URI:      "/v1/accounts/79403360-0dbf-11ee-b1ad-c3eebc4a6196/balance_add",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"balance":20}`),
			},
			expectRes: &bmaccount.Account{
				ID: uuid.FromStringOrNil("79403360-0dbf-11ee-b1ad-c3eebc4a6196"),
			},

			response: &rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"79403360-0dbf-11ee-b1ad-c3eebc4a6196"}`),
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

			res, err := reqHandler.BillingV1AccountAddBalance(ctx, tt.accountID, tt.balance)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_BillingV1AccountSubtractBalance(t *testing.T) {

	tests := []struct {
		name string

		accountID uuid.UUID
		balance   float32

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectRes     *bmaccount.Account

		response *rabbitmqhandler.Response
	}{
		{
			name: "normal",

			accountID: uuid.FromStringOrNil("c7b00aa2-0dbf-11ee-ab39-b7ac15120be3"),
			balance:   20,

			expectTarget: "bin-manager.billing-manager.request",
			expectRequest: &rabbitmqhandler.Request{
				URI:      "/v1/accounts/c7b00aa2-0dbf-11ee-ab39-b7ac15120be3/balance_subtract",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"balance":20}`),
			},
			expectRes: &bmaccount.Account{
				ID: uuid.FromStringOrNil("c7b00aa2-0dbf-11ee-ab39-b7ac15120be3"),
			},

			response: &rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"c7b00aa2-0dbf-11ee-ab39-b7ac15120be3"}`),
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

			res, err := reqHandler.BillingV1AccountSubtractBalance(ctx, tt.accountID, tt.balance)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}
