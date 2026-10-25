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

	"monorepo/bin-billing-manager/models/account"
	"monorepo/bin-billing-manager/models/billing"
	"monorepo/bin-billing-manager/pkg/accounthandler"
	"monorepo/bin-billing-manager/pkg/dbhandler"
)

func Test_processV1Accounts_inputAndErrorPaths(t *testing.T) {
	accountID := uuid.FromStringOrNil("a1000000-0000-0000-0000-000000000001")
	customerID := uuid.FromStringOrNil("c1000000-0000-0000-0000-000000000001")
	backendErr := errors.New("backend failure")

	tests := []struct {
		name string

		// call invokes the handler method under test.
		call func(h *listenHandler, req *sock.Request) (*sock.Response, error)
		req  *sock.Request

		setup func(m *accounthandler.MockAccountHandler)

		expectStatus int
		expectErr    bool
	}{
		// URI without an account id segment
		{
			name: "get - uri too short",
			call: func(h *listenHandler, req *sock.Request) (*sock.Response, error) {
				return h.processV1AccountsIDGet(context.Background(), req)
			},
			req:          &sock.Request{URI: "/v1", Method: sock.RequestMethodGet},
			setup:        func(m *accounthandler.MockAccountHandler) {},
			expectStatus: 400,
		},
		{
			name: "put - uri too short",
			call: func(h *listenHandler, req *sock.Request) (*sock.Response, error) {
				return h.processV1AccountsIDPut(context.Background(), req)
			},
			req:          &sock.Request{URI: "/v1", Method: sock.RequestMethodPut, Data: []byte(`{"name":"n"}`)},
			setup:        func(m *accounthandler.MockAccountHandler) {},
			expectStatus: 400,
		},
		{
			name: "balance add force - uri too short",
			call: func(h *listenHandler, req *sock.Request) (*sock.Response, error) {
				return h.processV1AccountsIDBalanceAddForcePost(context.Background(), req)
			},
			req:          &sock.Request{URI: "/v1", Method: sock.RequestMethodPost, Data: []byte(`{"balance":1}`)},
			setup:        func(m *accounthandler.MockAccountHandler) {},
			expectStatus: 400,
		},
		{
			name: "balance subtract force - uri too short",
			call: func(h *listenHandler, req *sock.Request) (*sock.Response, error) {
				return h.processV1AccountsIDBalanceSubtractForcePost(context.Background(), req)
			},
			req:          &sock.Request{URI: "/v1", Method: sock.RequestMethodPost, Data: []byte(`{"balance":1}`)},
			setup:        func(m *accounthandler.MockAccountHandler) {},
			expectStatus: 400,
		},
		{
			name: "is valid balance - uri too short",
			call: func(h *listenHandler, req *sock.Request) (*sock.Response, error) {
				return h.processV1AccountsIDIsValidBalancePost(context.Background(), req)
			},
			req:          &sock.Request{URI: "/v1/accounts/" + accountID.String(), Method: sock.RequestMethodPost, Data: []byte(`{}`)},
			setup:        func(m *accounthandler.MockAccountHandler) {},
			expectStatus: 400,
		},
		{
			name: "is valid resource limit - uri too short",
			call: func(h *listenHandler, req *sock.Request) (*sock.Response, error) {
				return h.processV1AccountsIDIsValidResourceLimitPost(context.Background(), req)
			},
			req:          &sock.Request{URI: "/v1/accounts/" + accountID.String(), Method: sock.RequestMethodPost, Data: []byte(`{}`)},
			setup:        func(m *accounthandler.MockAccountHandler) {},
			expectStatus: 400,
		},
		{
			name: "payment info - uri too short",
			call: func(h *listenHandler, req *sock.Request) (*sock.Response, error) {
				return h.processV1AccountsIDPaymentInfoPut(context.Background(), req)
			},
			req:          &sock.Request{URI: "/v1/accounts/" + accountID.String(), Method: sock.RequestMethodPut, Data: []byte(`{}`)},
			setup:        func(m *accounthandler.MockAccountHandler) {},
			expectStatus: 400,
		},

		// malformed request body: the handler is never reached
		{
			name: "put - malformed body",
			call: func(h *listenHandler, req *sock.Request) (*sock.Response, error) {
				return h.processV1AccountsIDPut(context.Background(), req)
			},
			req:       &sock.Request{URI: "/v1/accounts/" + accountID.String(), Method: sock.RequestMethodPut, Data: []byte(`{"name":`)},
			setup:     func(m *accounthandler.MockAccountHandler) {},
			expectErr: true,
		},
		{
			name: "balance add force - malformed body",
			call: func(h *listenHandler, req *sock.Request) (*sock.Response, error) {
				return h.processV1AccountsIDBalanceAddForcePost(context.Background(), req)
			},
			req:       &sock.Request{URI: "/v1/accounts/" + accountID.String() + "/balance_add_force", Method: sock.RequestMethodPost, Data: []byte(`{"balance":"x"}`)},
			setup:     func(m *accounthandler.MockAccountHandler) {},
			expectErr: true,
		},
		{
			name: "balance subtract force - malformed body",
			call: func(h *listenHandler, req *sock.Request) (*sock.Response, error) {
				return h.processV1AccountsIDBalanceSubtractForcePost(context.Background(), req)
			},
			req:       &sock.Request{URI: "/v1/accounts/" + accountID.String() + "/balance_subtract_force", Method: sock.RequestMethodPost, Data: []byte(`[`)},
			setup:     func(m *accounthandler.MockAccountHandler) {},
			expectErr: true,
		},
		{
			name: "is valid balance - malformed body",
			call: func(h *listenHandler, req *sock.Request) (*sock.Response, error) {
				return h.processV1AccountsIDIsValidBalancePost(context.Background(), req)
			},
			req:       &sock.Request{URI: "/v1/accounts/" + accountID.String() + "/is_valid_balance", Method: sock.RequestMethodPost, Data: []byte(`{"count":"x"}`)},
			setup:     func(m *accounthandler.MockAccountHandler) {},
			expectErr: true,
		},
		{
			name: "is valid resource limit - malformed body",
			call: func(h *listenHandler, req *sock.Request) (*sock.Response, error) {
				return h.processV1AccountsIDIsValidResourceLimitPost(context.Background(), req)
			},
			req:       &sock.Request{URI: "/v1/accounts/" + accountID.String() + "/is_valid_resource_limit", Method: sock.RequestMethodPost, Data: []byte(`{`)},
			setup:     func(m *accounthandler.MockAccountHandler) {},
			expectErr: true,
		},
		{
			name: "payment info - malformed body",
			call: func(h *listenHandler, req *sock.Request) (*sock.Response, error) {
				return h.processV1AccountsIDPaymentInfoPut(context.Background(), req)
			},
			req:       &sock.Request{URI: "/v1/accounts/" + accountID.String() + "/payment_info", Method: sock.RequestMethodPut, Data: []byte(`nope`)},
			setup:     func(m *accounthandler.MockAccountHandler) {},
			expectErr: true,
		},
		{
			name: "is valid resource limit by customer id - malformed body",
			call: func(h *listenHandler, req *sock.Request) (*sock.Response, error) {
				return h.processV1AccountsIsValidResourceLimitByCustomerIDPost(context.Background(), req)
			},
			req:       &sock.Request{URI: "/v1/accounts/is_valid_resource_limit_by_customer_id", Method: sock.RequestMethodPost, Data: []byte(`{"customer_id":`)},
			setup:     func(m *accounthandler.MockAccountHandler) {},
			expectErr: true,
		},

		// accountHandler errors are classified by errorResponse
		{
			name: "get - not found",
			call: func(h *listenHandler, req *sock.Request) (*sock.Response, error) {
				return h.processV1AccountsIDGet(context.Background(), req)
			},
			req: &sock.Request{URI: "/v1/accounts/" + accountID.String(), Method: sock.RequestMethodGet},
			setup: func(m *accounthandler.MockAccountHandler) {
				m.EXPECT().Get(gomock.Any(), accountID).Return(nil, dbhandler.ErrNotFound)
			},
			expectStatus: 404,
		},
		{
			name: "get - unclassified error",
			call: func(h *listenHandler, req *sock.Request) (*sock.Response, error) {
				return h.processV1AccountsIDGet(context.Background(), req)
			},
			req: &sock.Request{URI: "/v1/accounts/" + accountID.String(), Method: sock.RequestMethodGet},
			setup: func(m *accounthandler.MockAccountHandler) {
				m.EXPECT().Get(gomock.Any(), accountID).Return(nil, backendErr)
			},
			expectStatus: 500,
		},
		{
			name: "put - not found",
			call: func(h *listenHandler, req *sock.Request) (*sock.Response, error) {
				return h.processV1AccountsIDPut(context.Background(), req)
			},
			req: &sock.Request{URI: "/v1/accounts/" + accountID.String(), Method: sock.RequestMethodPut, Data: []byte(`{"name":"new name","detail":"new detail"}`)},
			setup: func(m *accounthandler.MockAccountHandler) {
				m.EXPECT().UpdateBasicInfo(gomock.Any(), accountID, "new name", "new detail").Return(nil, dbhandler.ErrNotFound)
			},
			expectStatus: 404,
		},
		{
			name: "balance add force - error",
			call: func(h *listenHandler, req *sock.Request) (*sock.Response, error) {
				return h.processV1AccountsIDBalanceAddForcePost(context.Background(), req)
			},
			req: &sock.Request{URI: "/v1/accounts/" + accountID.String() + "/balance_add_force", Method: sock.RequestMethodPost, Data: []byte(`{"balance":1500000}`)},
			setup: func(m *accounthandler.MockAccountHandler) {
				m.EXPECT().AddBalance(gomock.Any(), accountID, int64(1500000)).Return(nil, backendErr)
			},
			expectStatus: 500,
		},
		{
			name: "balance subtract force - error",
			call: func(h *listenHandler, req *sock.Request) (*sock.Response, error) {
				return h.processV1AccountsIDBalanceSubtractForcePost(context.Background(), req)
			},
			req: &sock.Request{URI: "/v1/accounts/" + accountID.String() + "/balance_subtract_force", Method: sock.RequestMethodPost, Data: []byte(`{"balance":250000}`)},
			setup: func(m *accounthandler.MockAccountHandler) {
				m.EXPECT().SubtractBalance(gomock.Any(), accountID, int64(250000)).Return(nil, dbhandler.ErrNotFound)
			},
			expectStatus: 404,
		},
		{
			name: "is valid balance - error",
			call: func(h *listenHandler, req *sock.Request) (*sock.Response, error) {
				return h.processV1AccountsIDIsValidBalancePost(context.Background(), req)
			},
			req: &sock.Request{URI: "/v1/accounts/" + accountID.String() + "/is_valid_balance", Method: sock.RequestMethodPost, Data: []byte(`{"billing_type":"call","country":"us","count":2}`)},
			setup: func(m *accounthandler.MockAccountHandler) {
				m.EXPECT().IsValidBalance(gomock.Any(), accountID, billing.ReferenceType("call"), "us", 2).Return(false, backendErr)
			},
			expectStatus: 500,
		},
		{
			name: "is valid resource limit - error",
			call: func(h *listenHandler, req *sock.Request) (*sock.Response, error) {
				return h.processV1AccountsIDIsValidResourceLimitPost(context.Background(), req)
			},
			req: &sock.Request{URI: "/v1/accounts/" + accountID.String() + "/is_valid_resource_limit", Method: sock.RequestMethodPost, Data: []byte(`{"resource_type":"extension"}`)},
			setup: func(m *accounthandler.MockAccountHandler) {
				m.EXPECT().IsValidResourceLimit(gomock.Any(), accountID, account.ResourceType("extension")).Return(false, dbhandler.ErrNotFound)
			},
			expectStatus: 404,
		},
		{
			name: "is valid balance by customer id - error",
			call: func(h *listenHandler, req *sock.Request) (*sock.Response, error) {
				return h.processV1AccountsIsValidBalanceByCustomerIDPost(context.Background(), req)
			},
			req: &sock.Request{URI: "/v1/accounts/is_valid_balance_by_customer_id", Method: sock.RequestMethodPost, Data: []byte(`{"customer_id":"` + customerID.String() + `","billing_type":"sms","country":"kr","count":1}`)},
			setup: func(m *accounthandler.MockAccountHandler) {
				m.EXPECT().IsValidBalanceByCustomerID(gomock.Any(), customerID, billing.ReferenceType("sms"), "kr", 1).Return(false, backendErr)
			},
			expectStatus: 500,
		},
		{
			name: "is valid resource limit by customer id - error",
			call: func(h *listenHandler, req *sock.Request) (*sock.Response, error) {
				return h.processV1AccountsIsValidResourceLimitByCustomerIDPost(context.Background(), req)
			},
			req: &sock.Request{URI: "/v1/accounts/is_valid_resource_limit_by_customer_id", Method: sock.RequestMethodPost, Data: []byte(`{"customer_id":"` + customerID.String() + `","resource_type":"agent"}`)},
			setup: func(m *accounthandler.MockAccountHandler) {
				m.EXPECT().IsValidResourceLimitByCustomerID(gomock.Any(), customerID, account.ResourceType("agent")).Return(false, dbhandler.ErrNotFound)
			},
			expectStatus: 404,
		},
		{
			name: "top up - error",
			call: func(h *listenHandler, req *sock.Request) (*sock.Response, error) {
				return h.processV1AccountsTopUpPost(context.Background(), req)
			},
			req: &sock.Request{URI: "/v1/accounts/top_up", Method: sock.RequestMethodPost},
			setup: func(m *accounthandler.MockAccountHandler) {
				m.EXPECT().TopUpDue(gomock.Any()).Return(0, 0, backendErr)
			},
			expectStatus: 500,
		},
		{
			name: "payment info - error",
			call: func(h *listenHandler, req *sock.Request) (*sock.Response, error) {
				return h.processV1AccountsIDPaymentInfoPut(context.Background(), req)
			},
			req: &sock.Request{URI: "/v1/accounts/" + accountID.String() + "/payment_info", Method: sock.RequestMethodPut, Data: []byte(`{"payment_type":"prepaid","payment_method":"credit card"}`)},
			setup: func(m *accounthandler.MockAccountHandler) {
				m.EXPECT().UpdatePaymentInfo(gomock.Any(), accountID, account.PaymentType("prepaid"), account.PaymentMethod("credit card")).Return(nil, dbhandler.ErrNotFound)
			},
			expectStatus: 404,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockAccount := accounthandler.NewMockAccountHandler(mc)
			h := &listenHandler{
				sockHandler:    sockhandler.NewMockSockHandler(mc),
				accountHandler: mockAccount,
			}
			tt.setup(mockAccount)

			res, err := tt.call(h, tt.req)

			if tt.expectErr {
				if err == nil {
					t.Errorf("Wrong match. expect: error, got: nil")
				}
				if res != nil {
					t.Errorf("Wrong match. expect: nil response, got: %v", res)
				}
				return
			}

			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if !reflect.DeepEqual(res, &sock.Response{StatusCode: tt.expectStatus}) {
				t.Errorf("Wrong match.\nexpect: status %d with no body\ngot:    %v", tt.expectStatus, res)
			}
		})
	}
}
