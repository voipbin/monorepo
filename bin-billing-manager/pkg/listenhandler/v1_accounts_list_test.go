package listenhandler

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-billing-manager/models/account"
	"monorepo/bin-billing-manager/models/billing"
	"monorepo/bin-billing-manager/pkg/accounthandler"
	"monorepo/bin-billing-manager/pkg/dbhandler"
)

func Test_processV1AccountsGet(t *testing.T) {

	type test struct {
		name    string
		request *sock.Request

		// when false the handler must not be called
		expectCall    bool
		responseList  []*account.Account
		responseErr   error
		expectFilters map[account.Field]any
		expectSize    uint64
		expectToken   string

		expectRes *sock.Response
	}

	tests := []test{
		{
			name: "normal with typed filters from the body",
			request: &sock.Request{
				URI:    "/v1/accounts?page_size=10&page_token=2023-06-08T03:22:17.995000Z",
				Method: sock.RequestMethodGet,
				Data:   []byte(`{"customer_id":"6a93f71e-f542-11ee-9a48-7f8011d36229","deleted":false}`),
			},

			expectCall:   true,
			responseList: []*account.Account{},
			expectFilters: map[account.Field]any{
				account.FieldCustomerID: uuid.FromStringOrNil("6a93f71e-f542-11ee-9a48-7f8011d36229"),
				account.FieldDeleted:    false,
			},
			expectSize:  uint64(10),
			expectToken: "2023-06-08T03:22:17.995000Z",

			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[]`),
			},
		},
		{
			name: "unknown filter key is dropped silently",
			request: &sock.Request{
				URI:    "/v1/accounts?page_size=5",
				Method: sock.RequestMethodGet,
				Data:   []byte(`{"no_such_field":"x","deleted":true}`),
			},

			expectCall:   true,
			responseList: []*account.Account{},
			expectFilters: map[account.Field]any{
				account.FieldDeleted: true,
			},
			expectSize:  uint64(5),
			expectToken: "",

			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[]`),
			},
		},
		{
			name: "filter in the query string only is not a filter",
			request: &sock.Request{
				URI:    "/v1/accounts?page_size=5&customer_id=6a93f71e-f542-11ee-9a48-7f8011d36229",
				Method: sock.RequestMethodGet,
			},

			expectCall:    true,
			responseList:  []*account.Account{},
			expectFilters: map[account.Field]any{},
			expectSize:    uint64(5),
			expectToken:   "",

			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[]`),
			},
		},
		{
			name: "broken filter json",
			request: &sock.Request{
				URI:    "/v1/accounts?page_size=5",
				Method: sock.RequestMethodGet,
				Data:   []byte(`{"customer_id":`),
			},

			expectRes: &sock.Response{StatusCode: 400},
		},
		{
			name: "customer_id is not a uuid",
			request: &sock.Request{
				URI:    "/v1/accounts?page_size=5",
				Method: sock.RequestMethodGet,
				Data:   []byte(`{"customer_id":"not-a-uuid"}`),
			},

			expectRes: &sock.Response{StatusCode: 400},
		},
		{
			name: "url can not be parsed",
			request: &sock.Request{
				URI:    "/%zz/v1/accounts?page_size=5",
				Method: sock.RequestMethodGet,
			},

			expectRes: &sock.Response{StatusCode: 400},
		},
		{
			name: "handler returns not found",
			request: &sock.Request{
				URI:    "/v1/accounts?page_size=5",
				Method: sock.RequestMethodGet,
			},

			expectCall:    true,
			responseErr:   dbhandler.ErrNotFound,
			expectFilters: map[account.Field]any{},
			expectSize:    uint64(5),
			expectToken:   "",

			expectRes: &sock.Response{StatusCode: 404},
		},
		{
			name: "handler returns an unclassified error",
			request: &sock.Request{
				URI:    "/v1/accounts?page_size=5",
				Method: sock.RequestMethodGet,
			},

			expectCall:    true,
			responseErr:   errors.New("backend failure"),
			expectFilters: map[account.Field]any{},
			expectSize:    uint64(5),
			expectToken:   "",

			expectRes: &sock.Response{StatusCode: 500},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockAccount := accounthandler.NewMockAccountHandler(mc)

			h := &listenHandler{
				sockHandler:    mockSock,
				accountHandler: mockAccount,
			}

			if tt.expectCall {
				mockAccount.EXPECT().List(gomock.Any(), tt.expectSize, tt.expectToken, tt.expectFilters).Return(tt.responseList, tt.responseErr)
			}

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

// the url parse failure is returned as an error by the handler itself and
// only turned into a 400 by the default error handler of processRequest.
func Test_processV1AccountsGet_urlParseError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h := &listenHandler{
		sockHandler:    sockhandler.NewMockSockHandler(mc),
		accountHandler: accounthandler.NewMockAccountHandler(mc),
	}

	res, err := h.processV1AccountsGet(context.Background(), &sock.Request{
		URI:    "/%zz/v1/accounts?page_size=5",
		Method: sock.RequestMethodGet,
	})
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil response, got: %v", res)
	}
}

func Test_processV1AccountsIsValidBalanceByCustomerIDPost(t *testing.T) {

	type test struct {
		name    string
		request *sock.Request

		expectCall        bool
		responseValid     bool
		responseErr       error
		expectCustomerID  uuid.UUID
		expectBillingType billing.ReferenceType
		expectCountry     string
		expectCount       int

		expectRes *sock.Response
	}

	tests := []test{
		{
			name: "valid",
			request: &sock.Request{
				URI:      "/v1/accounts/is_valid_balance_by_customer_id",
				Method:   sock.RequestMethodPost,
				DataType: requesthandler.ContentTypeJSON,
				Data:     []byte(`{"customer_id":"7c8e5a2e-1111-4a6b-9c0d-0123456789ab","billing_type":"call","country":"us","count":3}`),
			},

			expectCall:        true,
			responseValid:     true,
			expectCustomerID:  uuid.FromStringOrNil("7c8e5a2e-1111-4a6b-9c0d-0123456789ab"),
			expectBillingType: billing.ReferenceTypeCall,
			expectCountry:     "us",
			expectCount:       3,

			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"valid":true}`),
			},
		},
		{
			name: "not valid",
			request: &sock.Request{
				URI:      "/v1/accounts/is_valid_balance_by_customer_id",
				Method:   sock.RequestMethodPost,
				DataType: requesthandler.ContentTypeJSON,
				Data:     []byte(`{"customer_id":"7c8e5a2e-1111-4a6b-9c0d-0123456789ab","billing_type":"sms","country":"kr","count":1}`),
			},

			expectCall:        true,
			responseValid:     false,
			expectCustomerID:  uuid.FromStringOrNil("7c8e5a2e-1111-4a6b-9c0d-0123456789ab"),
			expectBillingType: billing.ReferenceTypeSMS,
			expectCountry:     "kr",
			expectCount:       1,

			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"valid":false}`),
			},
		},
		{
			name: "customer_id that is not a uuid is passed on as the nil uuid",
			request: &sock.Request{
				URI:      "/v1/accounts/is_valid_balance_by_customer_id",
				Method:   sock.RequestMethodPost,
				DataType: requesthandler.ContentTypeJSON,
				Data:     []byte(`{"customer_id":"not-a-uuid","billing_type":"call","country":"us","count":1}`),
			},

			expectCall:        true,
			responseValid:     false,
			expectCustomerID:  uuid.Nil,
			expectBillingType: billing.ReferenceTypeCall,
			expectCountry:     "us",
			expectCount:       1,

			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"valid":false}`),
			},
		},
		{
			name: "handler returns not found",
			request: &sock.Request{
				URI:      "/v1/accounts/is_valid_balance_by_customer_id",
				Method:   sock.RequestMethodPost,
				DataType: requesthandler.ContentTypeJSON,
				Data:     []byte(`{"customer_id":"7c8e5a2e-1111-4a6b-9c0d-0123456789ab","billing_type":"call","country":"us","count":1}`),
			},

			expectCall:        true,
			responseErr:       dbhandler.ErrNotFound,
			expectCustomerID:  uuid.FromStringOrNil("7c8e5a2e-1111-4a6b-9c0d-0123456789ab"),
			expectBillingType: billing.ReferenceTypeCall,
			expectCountry:     "us",
			expectCount:       1,

			expectRes: &sock.Response{StatusCode: 404},
		},
		{
			name: "handler returns an unclassified error",
			request: &sock.Request{
				URI:      "/v1/accounts/is_valid_balance_by_customer_id",
				Method:   sock.RequestMethodPost,
				DataType: requesthandler.ContentTypeJSON,
				Data:     []byte(`{"customer_id":"7c8e5a2e-1111-4a6b-9c0d-0123456789ab","billing_type":"call","country":"us","count":1}`),
			},

			expectCall:        true,
			responseErr:       errors.New("backend failure"),
			expectCustomerID:  uuid.FromStringOrNil("7c8e5a2e-1111-4a6b-9c0d-0123456789ab"),
			expectBillingType: billing.ReferenceTypeCall,
			expectCountry:     "us",
			expectCount:       1,

			expectRes: &sock.Response{StatusCode: 500},
		},
		{
			name: "broken body is a 400 through processRequest",
			request: &sock.Request{
				URI:      "/v1/accounts/is_valid_balance_by_customer_id",
				Method:   sock.RequestMethodPost,
				DataType: requesthandler.ContentTypeJSON,
				Data:     []byte(`{"customer_id":`),
			},

			expectRes: &sock.Response{StatusCode: 400},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockAccount := accounthandler.NewMockAccountHandler(mc)

			h := &listenHandler{
				sockHandler:    mockSock,
				accountHandler: mockAccount,
			}

			if tt.expectCall {
				mockAccount.EXPECT().IsValidBalanceByCustomerID(gomock.Any(), tt.expectCustomerID, tt.expectBillingType, tt.expectCountry, tt.expectCount).Return(tt.responseValid, tt.responseErr)
			}

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_processV1AccountsIsValidBalanceByCustomerIDPost_unmarshalError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h := &listenHandler{
		sockHandler:    sockhandler.NewMockSockHandler(mc),
		accountHandler: accounthandler.NewMockAccountHandler(mc),
	}

	res, err := h.processV1AccountsIsValidBalanceByCustomerIDPost(context.Background(), &sock.Request{
		URI:    "/v1/accounts/is_valid_balance_by_customer_id",
		Method: sock.RequestMethodPost,
		Data:   []byte(`{"customer_id":`),
	})
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil response, got: %v", res)
	}
}

func Test_processV1AccountsIsValidResourceLimitByCustomerIDPost(t *testing.T) {

	type test struct {
		name    string
		request *sock.Request

		expectCall         bool
		responseValid      bool
		responseErr        error
		expectCustomerID   uuid.UUID
		expectResourceType account.ResourceType

		expectRes *sock.Response
	}

	tests := []test{
		{
			name: "valid",
			request: &sock.Request{
				URI:      "/v1/accounts/is_valid_resource_limit_by_customer_id",
				Method:   sock.RequestMethodPost,
				DataType: requesthandler.ContentTypeJSON,
				Data:     []byte(`{"customer_id":"7c8e5a2e-1111-4a6b-9c0d-0123456789ab","resource_type":"extension"}`),
			},

			expectCall:         true,
			responseValid:      true,
			expectCustomerID:   uuid.FromStringOrNil("7c8e5a2e-1111-4a6b-9c0d-0123456789ab"),
			expectResourceType: account.ResourceTypeExtension,

			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"valid":true}`),
			},
		},
		{
			name: "limit reached",
			request: &sock.Request{
				URI:      "/v1/accounts/is_valid_resource_limit_by_customer_id",
				Method:   sock.RequestMethodPost,
				DataType: requesthandler.ContentTypeJSON,
				Data:     []byte(`{"customer_id":"7c8e5a2e-1111-4a6b-9c0d-0123456789ab","resource_type":"agent"}`),
			},

			expectCall:         true,
			responseValid:      false,
			expectCustomerID:   uuid.FromStringOrNil("7c8e5a2e-1111-4a6b-9c0d-0123456789ab"),
			expectResourceType: account.ResourceTypeAgent,

			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"valid":false}`),
			},
		},
		{
			name: "customer_id that is not a uuid is passed on as the nil uuid",
			request: &sock.Request{
				URI:      "/v1/accounts/is_valid_resource_limit_by_customer_id",
				Method:   sock.RequestMethodPost,
				DataType: requesthandler.ContentTypeJSON,
				Data:     []byte(`{"customer_id":"","resource_type":"agent"}`),
			},

			expectCall:         true,
			responseValid:      false,
			expectCustomerID:   uuid.Nil,
			expectResourceType: account.ResourceTypeAgent,

			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"valid":false}`),
			},
		},
		{
			name: "handler returns not found",
			request: &sock.Request{
				URI:      "/v1/accounts/is_valid_resource_limit_by_customer_id",
				Method:   sock.RequestMethodPost,
				DataType: requesthandler.ContentTypeJSON,
				Data:     []byte(`{"customer_id":"7c8e5a2e-1111-4a6b-9c0d-0123456789ab","resource_type":"agent"}`),
			},

			expectCall:         true,
			responseErr:        dbhandler.ErrNotFound,
			expectCustomerID:   uuid.FromStringOrNil("7c8e5a2e-1111-4a6b-9c0d-0123456789ab"),
			expectResourceType: account.ResourceTypeAgent,

			expectRes: &sock.Response{StatusCode: 404},
		},
		{
			name: "handler returns an unclassified error",
			request: &sock.Request{
				URI:      "/v1/accounts/is_valid_resource_limit_by_customer_id",
				Method:   sock.RequestMethodPost,
				DataType: requesthandler.ContentTypeJSON,
				Data:     []byte(`{"customer_id":"7c8e5a2e-1111-4a6b-9c0d-0123456789ab","resource_type":"agent"}`),
			},

			expectCall:         true,
			responseErr:        errors.New("backend failure"),
			expectCustomerID:   uuid.FromStringOrNil("7c8e5a2e-1111-4a6b-9c0d-0123456789ab"),
			expectResourceType: account.ResourceTypeAgent,

			expectRes: &sock.Response{StatusCode: 500},
		},
		{
			name: "broken body is a 400 through processRequest",
			request: &sock.Request{
				URI:      "/v1/accounts/is_valid_resource_limit_by_customer_id",
				Method:   sock.RequestMethodPost,
				DataType: requesthandler.ContentTypeJSON,
				Data:     []byte(`not json`),
			},

			expectRes: &sock.Response{StatusCode: 400},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockAccount := accounthandler.NewMockAccountHandler(mc)

			h := &listenHandler{
				sockHandler:    mockSock,
				accountHandler: mockAccount,
			}

			if tt.expectCall {
				mockAccount.EXPECT().IsValidResourceLimitByCustomerID(gomock.Any(), tt.expectCustomerID, tt.expectResourceType).Return(tt.responseValid, tt.responseErr)
			}

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_processV1AccountsIsValidResourceLimitByCustomerIDPost_unmarshalError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h := &listenHandler{
		sockHandler:    sockhandler.NewMockSockHandler(mc),
		accountHandler: accounthandler.NewMockAccountHandler(mc),
	}

	res, err := h.processV1AccountsIsValidResourceLimitByCustomerIDPost(context.Background(), &sock.Request{
		URI:    "/v1/accounts/is_valid_resource_limit_by_customer_id",
		Method: sock.RequestMethodPost,
		Data:   []byte(`not json`),
	})
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil response, got: %v", res)
	}
}
