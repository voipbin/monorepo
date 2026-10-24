package listenhandler

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-billing-manager/models/billing"
	"monorepo/bin-billing-manager/pkg/billinghandler"
	"monorepo/bin-billing-manager/pkg/dbhandler"
)

func Test_processV1BillingsGet_errors(t *testing.T) {

	type test struct {
		name    string
		request *sock.Request

		expectCall    bool
		responseErr   error
		expectFilters map[billing.Field]any

		expectRes *sock.Response
	}

	tests := []test{
		{
			name: "invalid filter",
			request: &sock.Request{
				URI:    "/v1/billings?page_size=5",
				Method: sock.RequestMethodGet,
				Data:   []byte(`{"customer_id":`),
			},

			expectRes: &sock.Response{StatusCode: 400},
		},
		{
			name: "account_id is not a uuid",
			request: &sock.Request{
				URI:    "/v1/billings?page_size=5",
				Method: sock.RequestMethodGet,
				Data:   []byte(`{"account_id":"not-a-uuid"}`),
			},

			expectRes: &sock.Response{StatusCode: 400},
		},
		{
			name: "malformed uri",
			request: &sock.Request{
				URI:    "/%zz/v1/billings?page_size=5",
				Method: sock.RequestMethodGet,
			},

			expectRes: &sock.Response{StatusCode: 400},
		},
		{
			name: "handler returns not found",
			request: &sock.Request{
				URI:    "/v1/billings?page_size=5",
				Method: sock.RequestMethodGet,
				Data:   []byte(`{"deleted":false}`),
			},

			expectCall:    true,
			responseErr:   dbhandler.ErrNotFound,
			expectFilters: map[billing.Field]any{billing.FieldDeleted: false},

			expectRes: &sock.Response{StatusCode: 404},
		},
		{
			name: "handler returns an unclassified error",
			request: &sock.Request{
				URI:    "/v1/billings?page_size=5",
				Method: sock.RequestMethodGet,
			},

			expectCall:    true,
			responseErr:   errors.New("backend failure"),
			expectFilters: map[billing.Field]any{},

			expectRes: &sock.Response{StatusCode: 500},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockBilling := billinghandler.NewMockBillingHandler(mc)

			h := &listenHandler{
				sockHandler:    mockSock,
				billingHandler: mockBilling,
			}

			if tt.expectCall {
				mockBilling.EXPECT().List(gomock.Any(), uint64(5), "", tt.expectFilters).Return(nil, tt.responseErr)
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

func Test_processV1BillingsGet_malformedURI(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h := &listenHandler{
		sockHandler:    sockhandler.NewMockSockHandler(mc),
		billingHandler: billinghandler.NewMockBillingHandler(mc),
	}

	res, err := h.processV1BillingsGet(context.Background(), &sock.Request{
		URI:    "/%zz/v1/billings?page_size=5",
		Method: sock.RequestMethodGet,
	})
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil response, got: %v", res)
	}
}

func Test_processV1BillingGet(t *testing.T) {

	billingID := uuid.FromStringOrNil("69cacd9e-f542-11ee-ab6d-afb3c2c93e56")

	type test struct {
		name    string
		request *sock.Request

		expectCall      bool
		expectID        uuid.UUID
		responseBilling *billing.Billing
		responseErr     error

		expectRes *sock.Response
	}

	tests := []test{
		{
			name: "normal",
			request: &sock.Request{
				URI:    "/v1/billings/69cacd9e-f542-11ee-ab6d-afb3c2c93e56",
				Method: sock.RequestMethodGet,
			},

			expectCall:      true,
			expectID:        billingID,
			responseBilling: &billing.Billing{},

			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"00000000-0000-0000-0000-000000000000","customer_id":"00000000-0000-0000-0000-000000000000","account_id":"00000000-0000-0000-0000-000000000000","transaction_type":"","status":"","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","cost_type":"","usage_duration":0,"billable_units":0,"rate_token_per_unit":0,"rate_credit_per_unit":0,"amount_token":0,"amount_credit":0,"balance_token_snapshot":0,"balance_credit_snapshot":0,"idempotency_key":"00000000-0000-0000-0000-000000000000","tm_billing_start":null,"tm_billing_end":null,"tm_create":null,"tm_update":null,"tm_delete":null}`),
			},
		},
		{
			name: "invalid id",
			request: &sock.Request{
				URI:    "/v1/billings/not-a-uuid",
				Method: sock.RequestMethodGet,
			},

			expectRes: &sock.Response{StatusCode: 404},
		},
		{
			name: "nil uuid",
			request: &sock.Request{
				URI:    "/v1/billings/00000000-0000-0000-0000-000000000000",
				Method: sock.RequestMethodGet,
			},

			expectRes: &sock.Response{StatusCode: 400},
		},
		{
			name: "path with extra segments",
			request: &sock.Request{
				URI:    "/v1/billings/69cacd9e-f542-11ee-ab6d-afb3c2c93e56/v1/billings/6a1d387c-f542-11ee-b2cb-a36ed20fc369",
				Method: sock.RequestMethodGet,
			},

			expectRes: &sock.Response{StatusCode: 400},
		},
		{
			name: "malformed uri",
			request: &sock.Request{
				URI:    "/%zz/v1/billings/69cacd9e-f542-11ee-ab6d-afb3c2c93e56",
				Method: sock.RequestMethodGet,
			},

			expectRes: &sock.Response{StatusCode: 400},
		},
		{
			name: "handler returns not found",
			request: &sock.Request{
				URI:    "/v1/billings/69cacd9e-f542-11ee-ab6d-afb3c2c93e56",
				Method: sock.RequestMethodGet,
			},

			expectCall:  true,
			expectID:    billingID,
			responseErr: dbhandler.ErrNotFound,

			expectRes: &sock.Response{StatusCode: 404},
		},
		{
			name: "handler returns an unclassified error",
			request: &sock.Request{
				URI:    "/v1/billings/69cacd9e-f542-11ee-ab6d-afb3c2c93e56",
				Method: sock.RequestMethodGet,
			},

			expectCall:  true,
			expectID:    billingID,
			responseErr: errors.New("backend failure"),

			expectRes: &sock.Response{StatusCode: 500},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockBilling := billinghandler.NewMockBillingHandler(mc)

			h := &listenHandler{
				sockHandler:    mockSock,
				billingHandler: mockBilling,
			}

			if tt.expectCall {
				mockBilling.EXPECT().Get(gomock.Any(), tt.expectID).Return(tt.responseBilling, tt.responseErr)
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
