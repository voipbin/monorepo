package listenhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-customer-manager/models/customer"
	"monorepo/bin-customer-manager/pkg/customerhandler"
)

func Test_processV1CustomersIDFreezeFamilyPost(t *testing.T) {
	id := uuid.FromStringOrNil("2cfbb148-7dc7-11ec-85df-47cf2c8492f0")

	tests := []struct {
		name   string
		uri    string
		expect func(m *customerhandler.MockCustomerHandler, res *customer.Customer, err error)
	}{
		{
			name: "freeze",
			uri:  "/v1/customers/2cfbb148-7dc7-11ec-85df-47cf2c8492f0/freeze",
			expect: func(m *customerhandler.MockCustomerHandler, res *customer.Customer, err error) {
				m.EXPECT().Freeze(gomock.Any(), id).Return(res, err)
			},
		},
		{
			name: "freeze_and_delete",
			uri:  "/v1/customers/2cfbb148-7dc7-11ec-85df-47cf2c8492f0/freeze_and_delete",
			expect: func(m *customerhandler.MockCustomerHandler, res *customer.Customer, err error) {
				m.EXPECT().FreezeAndDelete(gomock.Any(), id).Return(res, err)
			},
		},
		{
			name: "recover",
			uri:  "/v1/customers/2cfbb148-7dc7-11ec-85df-47cf2c8492f0/recover",
			expect: func(m *customerhandler.MockCustomerHandler, res *customer.Customer, err error) {
				m.EXPECT().Recover(gomock.Any(), id).Return(res, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name+" success", func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCustomer := customerhandler.NewMockCustomerHandler(mc)
			h := &listenHandler{
				sockHandler:     sockhandler.NewMockSockHandler(mc),
				reqHandler:      requesthandler.NewMockRequestHandler(mc),
				customerHandler: mockCustomer,
			}

			tt.expect(mockCustomer, &customer.Customer{ID: id, Status: customer.StatusFrozen}, nil)

			res, err := h.processRequest(&sock.Request{
				URI:      tt.uri,
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
			})
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			expectRes := &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"2cfbb148-7dc7-11ec-85df-47cf2c8492f0","billing_account_id":"00000000-0000-0000-0000-000000000000","email_verified":false,"status":"frozen","identity_verification_status":"","metadata":{"rtp_debug":false},"tm_deletion_scheduled":null,"tm_create":null,"tm_update":null,"tm_delete":null}`),
			}
			if !reflect.DeepEqual(res, expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", expectRes, res)
			}
		})

		t.Run(tt.name+" handler error returns 400", func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCustomer := customerhandler.NewMockCustomerHandler(mc)
			h := &listenHandler{
				sockHandler:     sockhandler.NewMockSockHandler(mc),
				reqHandler:      requesthandler.NewMockRequestHandler(mc),
				customerHandler: mockCustomer,
			}

			tt.expect(mockCustomer, nil, fmt.Errorf("handler failed"))

			res, err := h.processRequest(&sock.Request{
				URI:      tt.uri,
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
			})
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if res.StatusCode != 400 {
				t.Errorf("Wrong match. expect: 400, got: %d", res.StatusCode)
			}
		})
	}
}

// The route regexps guarantee at least five URI segments, so the short-URI
// guard is only reachable by calling the processors directly.
func Test_processV1CustomersIDFreezeFamilyPost_shortURI(t *testing.T) {
	tests := []struct {
		name string
		call func(h *listenHandler, m *sock.Request) (*sock.Response, error)
	}{
		{"freeze", func(h *listenHandler, m *sock.Request) (*sock.Response, error) {
			return h.processV1CustomersIDFreezePost(context.Background(), m)
		}},
		{"freeze_and_delete", func(h *listenHandler, m *sock.Request) (*sock.Response, error) {
			return h.processV1CustomersIDFreezeAndDeletePost(context.Background(), m)
		}},
		{"recover", func(h *listenHandler, m *sock.Request) (*sock.Response, error) {
			return h.processV1CustomersIDRecoverPost(context.Background(), m)
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			// no customerHandler call is expected: the strict mock fails on any call.
			h := &listenHandler{customerHandler: customerhandler.NewMockCustomerHandler(mc)}

			res, err := tt.call(h, &sock.Request{URI: "/v1/customers/x", Method: sock.RequestMethodPost})
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if res.StatusCode != 400 {
				t.Errorf("Wrong match. expect: 400, got: %d", res.StatusCode)
			}
		})
	}
}
