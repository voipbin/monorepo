package listenhandler

import (
	"fmt"
	"testing"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"go.uber.org/mock/gomock"

	"monorepo/bin-customer-manager/pkg/accesskeyhandler"
	"monorepo/bin-customer-manager/pkg/customerhandler"
)

const (
	errPathCustomerURI  = "/v1/customers/6c3e1f50-0000-4000-8000-000000000001"
	errPathAccesskeyURI = "/v1/accesskeys/6c3e1f50-0000-4000-8000-000000000002"
)

func newErrPathListenHandler(mc *gomock.Controller) (*listenHandler, *customerhandler.MockCustomerHandler, *accesskeyhandler.MockAccesskeyHandler) {
	mockCustomer := customerhandler.NewMockCustomerHandler(mc)
	mockAccesskey := accesskeyhandler.NewMockAccesskeyHandler(mc)

	h := &listenHandler{
		sockHandler:      sockhandler.NewMockSockHandler(mc),
		reqHandler:       requesthandler.NewMockRequestHandler(mc),
		customerHandler:  mockCustomer,
		accesskeyHandler: mockAccesskey,
	}
	return h, mockCustomer, mockAccesskey
}

// Test_processRequest_badInput covers the requests that are rejected before
// any business handler is called. The mocks have no expectations, so any
// handler call fails the test.
func Test_processRequest_badInput(t *testing.T) {
	tests := []struct {
		name         string
		request      *sock.Request
		expectStatus int
	}{
		{
			name: "customers list with an unparsable uri",
			request: &sock.Request{
				URI:    "/v1/customers?page_size=10\x7f",
				Method: sock.RequestMethodGet,
			},
			expectStatus: 400,
		},
		{
			name: "customers list with a malformed filter body",
			request: &sock.Request{
				URI:    "/v1/customers?page_size=10",
				Method: sock.RequestMethodGet,
				Data:   []byte(`{`),
			},
			expectStatus: 400,
		},
		{
			name: "customers list with an invalid uuid filter",
			request: &sock.Request{
				URI:    "/v1/customers?page_size=10",
				Method: sock.RequestMethodGet,
				Data:   []byte(`{"id":"not-a-uuid"}`),
			},
			expectStatus: 400,
		},
		{
			name: "customers create with a malformed body",
			request: &sock.Request{
				URI:    "/v1/customers",
				Method: sock.RequestMethodPost,
				Data:   []byte(`{`),
			},
			expectStatus: 400,
		},
		{
			name: "customer update with a malformed body",
			request: &sock.Request{
				URI:    errPathCustomerURI,
				Method: sock.RequestMethodPut,
				Data:   []byte(`{`),
			},
			expectStatus: 400,
		},
		{
			name: "customer billing account update with a malformed body",
			request: &sock.Request{
				URI:    errPathCustomerURI + "/billing_account_id",
				Method: sock.RequestMethodPut,
				Data:   []byte(`{`),
			},
			expectStatus: 400,
		},
		{
			name: "customer metadata update with a malformed body",
			request: &sock.Request{
				URI:    errPathCustomerURI + "/metadata",
				Method: sock.RequestMethodPut,
				Data:   []byte(`{`),
			},
			expectStatus: 400,
		},
		{
			name: "accesskeys list with an unparsable uri",
			request: &sock.Request{
				URI:    "/v1/accesskeys?page_size=10\x7f",
				Method: sock.RequestMethodGet,
			},
			expectStatus: 400,
		},
		{
			name: "accesskeys list with a malformed filter body",
			request: &sock.Request{
				URI:    "/v1/accesskeys?page_size=10",
				Method: sock.RequestMethodGet,
				Data:   []byte(`{`),
			},
			expectStatus: 400,
		},
		{
			name: "accesskeys list with an invalid uuid filter",
			request: &sock.Request{
				URI:    "/v1/accesskeys?page_size=10",
				Method: sock.RequestMethodGet,
				Data:   []byte(`{"id":"not-a-uuid"}`),
			},
			expectStatus: 400,
		},
		{
			name: "accesskeys create with a malformed body",
			request: &sock.Request{
				URI:    "/v1/accesskeys",
				Method: sock.RequestMethodPost,
				Data:   []byte(`{`),
			},
			expectStatus: 400,
		},
		{
			name: "accesskey update with a malformed body",
			request: &sock.Request{
				URI:    errPathAccesskeyURI,
				Method: sock.RequestMethodPut,
				Data:   []byte(`{`),
			},
			expectStatus: 400,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, _, _ := newErrPathListenHandler(mc)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res.StatusCode != tt.expectStatus {
				t.Errorf("Wrong match. expect: %d, got: %d", tt.expectStatus, res.StatusCode)
			}
		})
	}
}

// Test_processRequest_handlerError pins the status code each endpoint
// currently returns when the business handler fails.
func Test_processRequest_handlerError(t *testing.T) {
	tests := []struct {
		name         string
		request      *sock.Request
		setup        func(mockCustomer *customerhandler.MockCustomerHandler, mockAccesskey *accesskeyhandler.MockAccesskeyHandler)
		expectStatus int
	}{
		{
			name: "customer get",
			request: &sock.Request{
				URI:    errPathCustomerURI,
				Method: sock.RequestMethodGet,
			},
			setup: func(mockCustomer *customerhandler.MockCustomerHandler, _ *accesskeyhandler.MockAccesskeyHandler) {
				mockCustomer.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("handler error"))
			},
			expectStatus: 400,
		},
		{
			name: "customer delete",
			request: &sock.Request{
				URI:    errPathCustomerURI,
				Method: sock.RequestMethodDelete,
			},
			setup: func(mockCustomer *customerhandler.MockCustomerHandler, _ *accesskeyhandler.MockAccesskeyHandler) {
				mockCustomer.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("handler error"))
			},
			expectStatus: 400,
		},
		{
			name: "customer update",
			request: &sock.Request{
				URI:    errPathCustomerURI,
				Method: sock.RequestMethodPut,
				Data:   []byte(`{"name":"name"}`),
			},
			setup: func(mockCustomer *customerhandler.MockCustomerHandler, _ *accesskeyhandler.MockAccesskeyHandler) {
				mockCustomer.EXPECT().UpdateBasicInfo(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("handler error"))
			},
			expectStatus: 400,
		},
		{
			name: "customer billing account update",
			request: &sock.Request{
				URI:    errPathCustomerURI + "/billing_account_id",
				Method: sock.RequestMethodPut,
				Data:   []byte(`{"billing_account_id":"6c3e1f50-0000-4000-8000-000000000003"}`),
			},
			setup: func(mockCustomer *customerhandler.MockCustomerHandler, _ *accesskeyhandler.MockAccesskeyHandler) {
				mockCustomer.EXPECT().UpdateBillingAccountID(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("handler error"))
			},
			expectStatus: 400,
		},
		{
			name: "accesskey get",
			request: &sock.Request{
				URI:    errPathAccesskeyURI,
				Method: sock.RequestMethodGet,
			},
			setup: func(_ *customerhandler.MockCustomerHandler, mockAccesskey *accesskeyhandler.MockAccesskeyHandler) {
				mockAccesskey.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("handler error"))
			},
			expectStatus: 400,
		},
		{
			name: "accesskey delete",
			request: &sock.Request{
				URI:    errPathAccesskeyURI,
				Method: sock.RequestMethodDelete,
			},
			setup: func(_ *customerhandler.MockCustomerHandler, mockAccesskey *accesskeyhandler.MockAccesskeyHandler) {
				mockAccesskey.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("handler error"))
			},
			expectStatus: 400,
		},
		{
			name: "accesskey update",
			request: &sock.Request{
				URI:    errPathAccesskeyURI,
				Method: sock.RequestMethodPut,
				Data:   []byte(`{"name":"name","detail":"detail"}`),
			},
			setup: func(_ *customerhandler.MockCustomerHandler, mockAccesskey *accesskeyhandler.MockAccesskeyHandler) {
				mockAccesskey.EXPECT().UpdateBasicInfo(gomock.Any(), gomock.Any(), "name", "detail").Return(nil, fmt.Errorf("handler error"))
			},
			expectStatus: 400,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, mockCustomer, mockAccesskey := newErrPathListenHandler(mc)
			tt.setup(mockCustomer, mockAccesskey)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res.StatusCode != tt.expectStatus {
				t.Errorf("Wrong match. expect: %d, got: %d", tt.expectStatus, res.StatusCode)
			}
			if len(res.Data) != 0 {
				t.Errorf("Wrong match. expect: empty body on failure, got: %s", res.Data)
			}
		})
	}
}

// Test_shortURI calls the handlers directly because the routing patterns
// never hand them a uri with too few path items.
func Test_shortURI(t *testing.T) {
	tests := []struct {
		name string
		call func(h *listenHandler, m *sock.Request) (*sock.Response, error)
	}{
		{"customers get by id", func(h *listenHandler, m *sock.Request) (*sock.Response, error) {
			return h.processV1CustomersIDGet(nil, m) //nolint:staticcheck // the handler does not use the context before the uri check
		}},
		{"customers delete", func(h *listenHandler, m *sock.Request) (*sock.Response, error) {
			return h.processV1CustomersIDDelete(nil, m) //nolint:staticcheck // see above
		}},
		{"customers update", func(h *listenHandler, m *sock.Request) (*sock.Response, error) {
			return h.processV1CustomersIDPut(nil, m) //nolint:staticcheck // see above
		}},
		{"customers billing account update", func(h *listenHandler, m *sock.Request) (*sock.Response, error) {
			return h.processV1CustomersIDBillingAccountIDPut(nil, m) //nolint:staticcheck // see above
		}},
		{"customers metadata update", func(h *listenHandler, m *sock.Request) (*sock.Response, error) {
			return h.processV1CustomersIDMetadataPut(nil, m) //nolint:staticcheck // see above
		}},
		{"accesskeys get by id", func(h *listenHandler, m *sock.Request) (*sock.Response, error) {
			return h.processV1AccesskeysIDGet(nil, m) //nolint:staticcheck // see above
		}},
		{"accesskeys delete", func(h *listenHandler, m *sock.Request) (*sock.Response, error) {
			return h.processV1AccesskeysIDDelete(nil, m) //nolint:staticcheck // see above
		}},
		{"accesskeys update", func(h *listenHandler, m *sock.Request) (*sock.Response, error) {
			return h.processV1AccesskeysIDPut(nil, m) //nolint:staticcheck // see above
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, _, _ := newErrPathListenHandler(mc)

			res, err := tt.call(h, &sock.Request{URI: "/v1/x"})
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res.StatusCode != 400 {
				t.Errorf("Wrong match. expect: 400, got: %d", res.StatusCode)
			}
		})
	}
}

func Test_Run_queueCreateError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	h := &listenHandler{sockHandler: mockSock}

	mockSock.EXPECT().QueueCreate("test-queue", "normal").Return(fmt.Errorf("queue error"))

	// no ConsumeRPC expectation: a failed queue declaration must not start consuming
	err := h.Run("test-queue", "")
	if err == nil {
		t.Fatalf("Wrong match. expect: error, got: nil")
	}
	if got := err.Error(); got != "could not declare the queue for listenHandler. err: queue error" {
		t.Errorf("Wrong match. got: %s", got)
	}
}

func Test_Run_startsConsumer(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	h := &listenHandler{sockHandler: mockSock}

	started := make(chan struct{})
	mockSock.EXPECT().QueueCreate("test-queue", "normal").Return(nil)
	mockSock.EXPECT().ConsumeRPC(gomock.Any(), "test-queue", gomock.Any(), false, false, false, 10, gomock.Any()).
		DoAndReturn(func(_ any, _ string, _ string, _ bool, _ bool, _ bool, _ int, _ sock.CbMsgRPC) error {
			close(started)
			return fmt.Errorf("consume error")
		})

	if err := h.Run("test-queue", ""); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	// wait for the consumer goroutine so the mock is not called after the controller finishes
	<-started
}
