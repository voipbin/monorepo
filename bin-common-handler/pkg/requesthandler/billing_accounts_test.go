package requesthandler

import (
	"context"
	"reflect"
	"testing"

	bmaccount "monorepo/bin-billing-manager/models/account"
	bmbilling "monorepo/bin-billing-manager/models/billing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
)

func Test_BillingV1AccountGets(t *testing.T) {

	tests := []struct {
		name string

		size    uint64
		token   string
		filters map[string]string

		expectURL     string
		expectTarget  string
		expectRequest *sock.Request
		expectRes     []bmaccount.Account
		response      *sock.Response
	}{
		{
			name: "normal",

			size:  10,
			token: "2023-06-08 03:22:17.995000",
			filters: map[string]string{
				"customer_id": "33a95f94-0e7c-11ee-aeb3-57a93b9f70fd",
			},

			expectURL:    "/v1/accounts?page_token=2023-06-08+03%3A22%3A17.995000&page_size=10",
			expectTarget: "bin-manager.billing-manager.request",
			expectRequest: &sock.Request{
				URI:    "/v1/accounts?page_token=2023-06-08+03%3A22%3A17.995000&page_size=10&filter_customer_id=33a95f94-0e7c-11ee-aeb3-57a93b9f70fd",
				Method: sock.RequestMethodGet,
			},
			expectRes: []bmaccount.Account{
				{
					ID: uuid.FromStringOrNil("022bfc94-0b9b-11ee-8ea1-f3e4fbd66309"),
				},
				{
					ID: uuid.FromStringOrNil("025e6814-0b9b-11ee-8e8d-93e70b8939a0"),
				},
			},

			response: &sock.Response{
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
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			reqHandler := requestHandler{
				sock:        mockSock,
				utilHandler: mockUtil,
			}
			ctx := context.Background()

			mockUtil.EXPECT().URLMergeFilters(tt.expectURL, tt.filters).Return(utilhandler.URLMergeFilters(tt.expectURL, tt.filters))
			mockSock.EXPECT().PublishRequest(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)
			res, err := reqHandler.BillingV1AccountGets(ctx, tt.token, tt.size, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}

		})
	}
}

func Test_BillingV1AccountCreate(t *testing.T) {

	tests := []struct {
		name string

		customerID    uuid.UUID
		accountName   string
		detail        string
		paymentType   bmaccount.PaymentType
		paymentMethod bmaccount.PaymentMethod

		expectTarget  string
		expectRequest *sock.Request
		expectRes     *bmaccount.Account
		response      *sock.Response
	}{
		{
			name: "normal",

			customerID:    uuid.FromStringOrNil("513712d6-0e7c-11ee-9a95-1b0696a625b6"),
			accountName:   "test name",
			detail:        "test detail",
			paymentType:   bmaccount.PaymentTypePrepaid,
			paymentMethod: bmaccount.PaymentMethodCreditCard,

			expectTarget: "bin-manager.billing-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/accounts",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"customer_id":"513712d6-0e7c-11ee-9a95-1b0696a625b6","name":"test name","detail":"test detail","payment_type":"prepaid","payment_method":"credit card"}`),
			},
			expectRes: &bmaccount.Account{
				ID: uuid.FromStringOrNil("8be2a24c-0e7c-11ee-957a-c7e813baceb9"),
			},

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"8be2a24c-0e7c-11ee-957a-c7e813baceb9"}`),
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
			mockSock.EXPECT().PublishRequest(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.BillingV1AccountCreate(ctx, tt.customerID, tt.accountName, tt.detail, tt.paymentType, tt.paymentMethod)
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
		expectRequest *sock.Request
		expectRes     *bmaccount.Account
		response      *sock.Response
	}{
		{
			name: "normal",

			accountID: uuid.FromStringOrNil("9000392c-0b9b-11ee-aa1d-8b84b3626bc7"),

			expectTarget: "bin-manager.billing-manager.request",
			expectRequest: &sock.Request{
				URI:    "/v1/accounts/9000392c-0b9b-11ee-aa1d-8b84b3626bc7",
				Method: sock.RequestMethodGet,
			},
			expectRes: &bmaccount.Account{
				ID: uuid.FromStringOrNil("9000392c-0b9b-11ee-aa1d-8b84b3626bc7"),
			},

			response: &sock.Response{
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
			mockSock.EXPECT().PublishRequest(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

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

func Test_BillingV1AccountDelete(t *testing.T) {

	tests := []struct {
		name string

		accountID uuid.UUID

		expectTarget  string
		expectRequest *sock.Request
		expectRes     *bmaccount.Account
		response      *sock.Response
	}{
		{
			name: "normal",

			accountID: uuid.FromStringOrNil("9c2bd1f6-0e80-11ee-91d4-37bdb8051fad"),

			expectTarget: "bin-manager.billing-manager.request",
			expectRequest: &sock.Request{
				URI:    "/v1/accounts/9c2bd1f6-0e80-11ee-91d4-37bdb8051fad",
				Method: sock.RequestMethodDelete,
			},
			expectRes: &bmaccount.Account{
				ID: uuid.FromStringOrNil("9c2bd1f6-0e80-11ee-91d4-37bdb8051fad"),
			},

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"9c2bd1f6-0e80-11ee-91d4-37bdb8051fad"}`),
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
			mockSock.EXPECT().PublishRequest(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.BillingV1AccountDelete(ctx, tt.accountID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}

		})
	}
}

func Test_BillingV1AccountAddBalanceForce(t *testing.T) {

	tests := []struct {
		name string

		accountID uuid.UUID
		balance   float32

		expectTarget  string
		expectRequest *sock.Request
		expectRes     *bmaccount.Account

		response *sock.Response
	}{
		{
			name: "normal",

			accountID: uuid.FromStringOrNil("79403360-0dbf-11ee-b1ad-c3eebc4a6196"),
			balance:   20,

			expectTarget: "bin-manager.billing-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/accounts/79403360-0dbf-11ee-b1ad-c3eebc4a6196/balance_add_force",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"balance":20}`),
			},
			expectRes: &bmaccount.Account{
				ID: uuid.FromStringOrNil("79403360-0dbf-11ee-b1ad-c3eebc4a6196"),
			},

			response: &sock.Response{
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
			mockSock.EXPECT().PublishRequest(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.BillingV1AccountAddBalanceForce(ctx, tt.accountID, tt.balance)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_BillingV1AccountSubtractBalanceForce(t *testing.T) {

	tests := []struct {
		name string

		accountID uuid.UUID
		balance   float32

		expectTarget  string
		expectRequest *sock.Request
		expectRes     *bmaccount.Account

		response *sock.Response
	}{
		{
			name: "normal",

			accountID: uuid.FromStringOrNil("c7b00aa2-0dbf-11ee-ab39-b7ac15120be3"),
			balance:   20,

			expectTarget: "bin-manager.billing-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/accounts/c7b00aa2-0dbf-11ee-ab39-b7ac15120be3/balance_subtract_force",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"balance":20}`),
			},
			expectRes: &bmaccount.Account{
				ID: uuid.FromStringOrNil("c7b00aa2-0dbf-11ee-ab39-b7ac15120be3"),
			},

			response: &sock.Response{
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
			mockSock.EXPECT().PublishRequest(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.BillingV1AccountSubtractBalanceForce(ctx, tt.accountID, tt.balance)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_BillingV1AccountIsValidBalance(t *testing.T) {

	tests := []struct {
		name string

		accountID   uuid.UUID
		billingType bmbilling.ReferenceType
		country     string
		Count       int

		expectTarget  string
		expectRequest *sock.Request
		expectRes     bool

		response *sock.Response
	}{
		{
			name: "normal",

			accountID:   uuid.FromStringOrNil("6ec4c6cc-134f-11ee-acb1-83e6a5d0d5cf"),
			billingType: bmbilling.ReferenceTypeCall,
			country:     "us",
			Count:       3,

			expectTarget: "bin-manager.billing-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/accounts/6ec4c6cc-134f-11ee-acb1-83e6a5d0d5cf/is_valid_balance",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"billing_type":"call","country":"us","count":3}`),
			},
			expectRes: true,

			response: &sock.Response{
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
			mockSock.EXPECT().PublishRequest(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.BillingV1AccountIsValidBalance(ctx, tt.accountID, tt.billingType, tt.country, tt.Count)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_BillingV1AccountUpdateBasicInfo(t *testing.T) {

	tests := []struct {
		name string

		accountID   uuid.UUID
		accountName string
		detail      string

		expectTarget  string
		expectRequest *sock.Request
		expectRes     *bmaccount.Account
		response      *sock.Response
	}{
		{
			name: "normal",

			accountID:   uuid.FromStringOrNil("c1085dc6-4cd5-11ee-8065-a7ccfdd78669"),
			accountName: "test name",
			detail:      "test detail",

			expectTarget: "bin-manager.billing-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/accounts/c1085dc6-4cd5-11ee-8065-a7ccfdd78669",
				Method:   sock.RequestMethodPut,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"name":"test name","detail":"test detail"}`),
			},
			expectRes: &bmaccount.Account{
				ID: uuid.FromStringOrNil("c1085dc6-4cd5-11ee-8065-a7ccfdd78669"),
			},

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"c1085dc6-4cd5-11ee-8065-a7ccfdd78669"}`),
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
			mockSock.EXPECT().PublishRequest(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.BillingV1AccountUpdateBasicInfo(ctx, tt.accountID, tt.accountName, tt.detail)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}

		})
	}
}

func Test_BillingV1AccountUpdatePaymentInfo(t *testing.T) {

	tests := []struct {
		name string

		accountID     uuid.UUID
		paymentType   bmaccount.PaymentType
		paymentMethod bmaccount.PaymentMethod

		expectTarget  string
		expectRequest *sock.Request
		expectRes     *bmaccount.Account
		response      *sock.Response
	}{
		{
			name: "normal",

			accountID:     uuid.FromStringOrNil("c149ecbe-4cd5-11ee-bf72-872e67a10683"),
			paymentType:   bmaccount.PaymentTypePrepaid,
			paymentMethod: bmaccount.PaymentMethodCreditCard,

			expectTarget: "bin-manager.billing-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/accounts/c149ecbe-4cd5-11ee-bf72-872e67a10683/payment_info",
				Method:   sock.RequestMethodPut,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"payment_type":"prepaid","payment_method":"credit card"}`),
			},
			expectRes: &bmaccount.Account{
				ID: uuid.FromStringOrNil("c149ecbe-4cd5-11ee-bf72-872e67a10683"),
			},

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"c149ecbe-4cd5-11ee-bf72-872e67a10683"}`),
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
			mockSock.EXPECT().PublishRequest(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.BillingV1AccountUpdatePaymentInfo(ctx, tt.accountID, tt.paymentType, tt.paymentMethod)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}

		})
	}
}
