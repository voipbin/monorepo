package listenhandler

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"go.uber.org/mock/gomock"

	"monorepo/bin-registrar-manager/models/astcontact"
	"monorepo/bin-registrar-manager/pkg/contacthandler"
	"monorepo/bin-registrar-manager/pkg/customerdomainhandler"
	"monorepo/bin-registrar-manager/pkg/extensionhandler"
	"monorepo/bin-registrar-manager/pkg/trunkhandler"
)

// Test_processV1_unparsableURI verifies that handlers which parse the request
// URI return the parse error instead of a response, and never call the
// business handlers (the mocks have no expectations).
func Test_processV1_unparsableURI(t *testing.T) {
	const bad = "/v1/x/%zz"

	tests := []struct {
		name string
		fn   func(h *listenHandler, req *sock.Request) (*sock.Response, error)
		meth sock.RequestMethod
	}{
		{"trunks get", func(h *listenHandler, r *sock.Request) (*sock.Response, error) {
			return h.processV1TrunksGet(context.Background(), r)
		}, sock.RequestMethodGet},
		{"trunks id get", func(h *listenHandler, r *sock.Request) (*sock.Response, error) {
			return h.processV1TrunksIDGet(context.Background(), r)
		}, sock.RequestMethodGet},
		{"trunks id put", func(h *listenHandler, r *sock.Request) (*sock.Response, error) {
			return h.processV1TrunksIDPut(context.Background(), r)
		}, sock.RequestMethodPut},
		{"trunks id delete", func(h *listenHandler, r *sock.Request) (*sock.Response, error) {
			return h.processV1TrunksIDDelete(context.Background(), r)
		}, sock.RequestMethodDelete},
		{"trunks domain name get", func(h *listenHandler, r *sock.Request) (*sock.Response, error) {
			return h.processV1TrunksDomainNameDomainNameGet(context.Background(), r)
		}, sock.RequestMethodGet},
		{"extensions get", func(h *listenHandler, r *sock.Request) (*sock.Response, error) {
			return h.processV1ExtensionsGet(context.Background(), r)
		}, sock.RequestMethodGet},
		{"extensions id put", func(h *listenHandler, r *sock.Request) (*sock.Response, error) {
			return h.processV1ExtensionsIDPut(context.Background(), r)
		}, sock.RequestMethodPut},
		{"extensions id delete", func(h *listenHandler, r *sock.Request) (*sock.Response, error) {
			return h.processV1ExtensionsIDDelete(context.Background(), r)
		}, sock.RequestMethodDelete},
		{"customer domains realm get", func(h *listenHandler, r *sock.Request) (*sock.Response, error) {
			return h.processV1CustomerDomainsRealmGet(context.Background(), r)
		}, sock.RequestMethodGet},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h := &listenHandler{
				trunkHandler:          trunkhandler.NewMockTrunkHandler(mc),
				extensionHandler:      extensionhandler.NewMockExtensionHandler(mc),
				customerDomainHandler: customerdomainhandler.NewMockCustomerDomainHandler(mc),
			}

			res, err := tt.fn(h, &sock.Request{URI: bad, Method: tt.meth})
			if err == nil || res != nil {
				t.Errorf("response/error = %+v/%v, want nil and an error", res, err)
			}
		})
	}
}

// Test_processV1_shortURI verifies that a request whose URI has too few path
// items is rejected with 400 before the body is read.
func Test_processV1_shortURI(t *testing.T) {
	tests := []struct {
		name string
		fn   func(h *listenHandler, req *sock.Request) (*sock.Response, error)
		uri  string
	}{
		{"trunks post", func(h *listenHandler, r *sock.Request) (*sock.Response, error) {
			return h.processV1TrunksPost(context.Background(), r)
		}, "/v1"},
		{"extensions post", func(h *listenHandler, r *sock.Request) (*sock.Response, error) {
			return h.processV1ExtensionsPost(context.Background(), r)
		}, "/v1"},
		{"extensions direct hash regenerate", func(h *listenHandler, r *sock.Request) (*sock.Response, error) {
			return h.processV1ExtensionsIDDirectHashRegenerate(context.Background(), r)
		}, "/v1/extensions"},
		{"customer domains realm get, empty realm", func(h *listenHandler, r *sock.Request) (*sock.Response, error) {
			return h.processV1CustomerDomainsRealmGet(context.Background(), r)
		}, "/v1/customer_domains/realm/"},
		{"customer domains realm get, no realm item", func(h *listenHandler, r *sock.Request) (*sock.Response, error) {
			return h.processV1CustomerDomainsRealmGet(context.Background(), r)
		}, "/v1/customer_domains"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			// the business handlers have no expectations: any call fails the test.
			h := &listenHandler{
				trunkHandler:          trunkhandler.NewMockTrunkHandler(mc),
				extensionHandler:      extensionhandler.NewMockExtensionHandler(mc),
				customerDomainHandler: customerdomainhandler.NewMockCustomerDomainHandler(mc),
			}

			res, err := tt.fn(h, &sock.Request{URI: tt.uri, Data: []byte(`{}`)})
			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			if res == nil || res.StatusCode != 400 {
				t.Errorf("expected status 400, got: %+v", res)
			}
		})
	}
}

// Test_processRequest_unhandledError verifies that an error returned by a
// handler is turned into a 400 response by processRequest.
func Test_processRequest_unhandledError(t *testing.T) {
	tests := []struct {
		name string
		req  *sock.Request
	}{
		{"unparsable trunk domain name uri", &sock.Request{URI: "/v1/trunks/domain_name/%zz", Method: sock.RequestMethodGet}},
		{"invalid filter body", &sock.Request{URI: "/v1/contacts", Method: sock.RequestMethodGet, Data: []byte(`{`)}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h := &listenHandler{
				trunkHandler:   trunkhandler.NewMockTrunkHandler(mc),
				contactHandler: contacthandler.NewMockContactHandler(mc),
			}

			res, err := h.processRequest(tt.req)
			if err != nil {
				t.Fatalf("processRequest must not return the handler error, got: %v", err)
			}
			if res == nil || res.StatusCode != 400 {
				t.Errorf("expected status 400, got: %+v", res)
			}
		})
	}
}

// Test_processV1ContactsGet_unmarshalableResponse verifies that a response
// that cannot be encoded is returned as an error.
func Test_processV1ContactsGet_unmarshalableResponse(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockContact := contacthandler.NewMockContactHandler(mc)
	h := &listenHandler{contactHandler: mockContact}

	mockContact.EXPECT().ContactGetsByExtension(gomock.Any(), gomock.Any(), gomock.Any()).
		Return([]*astcontact.AstContact{{ID: "c1", QualifyTimeout: math.NaN()}}, nil)

	res, err := h.processV1ContactsGet(context.Background(), &sock.Request{URI: "/v1/contacts", Method: sock.RequestMethodGet})
	if err == nil || res != nil {
		t.Errorf("response/error = %+v/%v, want nil and an error", res, err)
	}
}

func Test_NewListenHandler_keepsDependencies(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockTrunk := trunkhandler.NewMockTrunkHandler(mc)
	mockExtension := extensionhandler.NewMockExtensionHandler(mc)
	mockContact := contacthandler.NewMockContactHandler(mc)
	mockDomain := customerdomainhandler.NewMockCustomerDomainHandler(mc)

	res, ok := NewListenHandler(mockSock, nil, mockTrunk, mockExtension, mockContact, mockDomain).(*listenHandler)
	if !ok {
		t.Fatalf("NewListenHandler must return *listenHandler")
	}

	if res.sockHandler != mockSock || res.trunkHandler != mockTrunk || res.extensionHandler != mockExtension ||
		res.contactHandler != mockContact || res.customerDomainHandler != mockDomain {
		t.Errorf("the injected dependencies were not kept: %+v", res)
	}
	if res.utilHandler == nil {
		t.Errorf("the util handler must be created")
	}
}

func Test_Run(t *testing.T) {
	t.Run("queue creation failure is returned and nothing is consumed", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockSock := sockhandler.NewMockSockHandler(mc)
		h := &listenHandler{sockHandler: mockSock}

		errQueue := errors.New("queue failure")
		mockSock.EXPECT().QueueCreate("test.queue", "normal").Return(errQueue)

		if err := h.Run("test.queue", ""); !errors.Is(err, errQueue) {
			t.Errorf("expected the queue failure, got: %v", err)
		}
	})

	t.Run("consumer starts on the created queue", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockSock := sockhandler.NewMockSockHandler(mc)
		h := &listenHandler{sockHandler: mockSock}

		started := make(chan struct{})
		mockSock.EXPECT().QueueCreate("test.queue", "normal").Return(nil)
		mockSock.EXPECT().ConsumeRPC(gomock.Any(), "test.queue", "registrar-manager", false, false, false, 10, gomock.Any()).
			DoAndReturn(func(_ context.Context, _ string, _ string, _ bool, _ bool, _ bool, _ int, _ sock.CbMsgRPC) error {
				close(started)
				return errors.New("consumer stopped")
			})

		if err := h.Run("test.queue", ""); err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		select {
		case <-started:
		case <-time.After(5 * time.Second):
			t.Fatalf("the consumer was not started")
		}
	})
}
