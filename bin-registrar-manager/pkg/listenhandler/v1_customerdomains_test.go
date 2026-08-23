package listenhandler

import (
	"fmt"
	"net/http"
	"reflect"
	"testing"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-registrar-manager/models/customerdomain"
	"monorepo/bin-registrar-manager/pkg/customerdomainhandler"
	"monorepo/bin-registrar-manager/pkg/dbhandler"
)

func Test_processV1CustomerDomainsRealmGet(t *testing.T) {

	type test struct {
		name string

		request        *sock.Request
		responseDomain *customerdomain.CustomerDomain

		expectRealm string
		expectRes   *sock.Response
	}

	tests := []test{
		{
			"normal short realm",
			&sock.Request{
				URI:    "/v1/customer_domains/realm/ab12.reg.voipbin.net",
				Method: sock.RequestMethodGet,
			},
			&customerdomain.CustomerDomain{
				CustomerID:  uuid.FromStringOrNil("d5829769-dacf-420e-9260-c8931560331e"),
				DomainLabel: "ab12",
				Realm:       "ab12.reg.voipbin.net",
			},

			"ab12.reg.voipbin.net",
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"customer_id":"d5829769-dacf-420e-9260-c8931560331e","domain_label":"ab12","realm":"ab12.reg.voipbin.net","tm_create":null,"tm_update":null}`),
			},
		},
		{
			"legacy uuid realm",
			&sock.Request{
				URI:    "/v1/customer_domains/realm/d5829769-dacf-420e-9260-c8931560331e.registrar.voipbin.net",
				Method: sock.RequestMethodGet,
			},
			&customerdomain.CustomerDomain{
				CustomerID:  uuid.FromStringOrNil("d5829769-dacf-420e-9260-c8931560331e"),
				DomainLabel: "d5829769-dacf-420e-9260-c8931560331e",
				Realm:       "d5829769-dacf-420e-9260-c8931560331e.registrar.voipbin.net",
			},

			"d5829769-dacf-420e-9260-c8931560331e.registrar.voipbin.net",
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"customer_id":"d5829769-dacf-420e-9260-c8931560331e","domain_label":"d5829769-dacf-420e-9260-c8931560331e","realm":"d5829769-dacf-420e-9260-c8931560331e.registrar.voipbin.net","tm_create":null,"tm_update":null}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockCustomerDomain := customerdomainhandler.NewMockCustomerDomainHandler(mc)

			h := &listenHandler{
				sockHandler:           mockSock,
				reqHandler:            mockReq,
				customerDomainHandler: mockCustomerDomain,
			}

			mockCustomerDomain.EXPECT().GetByRealm(gomock.Any(), tt.expectRealm).Return(tt.responseDomain, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_processV1CustomerDomainsRealmGet_notFound(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockCustomerDomain := customerdomainhandler.NewMockCustomerDomainHandler(mc)

	h := &listenHandler{
		sockHandler:           mockSock,
		reqHandler:            mockReq,
		customerDomainHandler: mockCustomerDomain,
	}

	request := &sock.Request{
		URI:    "/v1/customer_domains/realm/nope.reg.voipbin.net",
		Method: sock.RequestMethodGet,
	}

	mockCustomerDomain.EXPECT().GetByRealm(gomock.Any(), "nope.reg.voipbin.net").Return(nil, fmt.Errorf("wrap: %w", dbhandler.ErrNotFound))

	res, err := h.processRequest(request)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("Wrong match. expect: %d, got: %d", http.StatusNotFound, res.StatusCode)
	}
}

func Test_processV1CustomerDomainsRealmGet_internalError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockCustomerDomain := customerdomainhandler.NewMockCustomerDomainHandler(mc)

	h := &listenHandler{
		sockHandler:           mockSock,
		reqHandler:            mockReq,
		customerDomainHandler: mockCustomerDomain,
	}

	request := &sock.Request{
		URI:    "/v1/customer_domains/realm/ab12.reg.voipbin.net",
		Method: sock.RequestMethodGet,
	}

	mockCustomerDomain.EXPECT().GetByRealm(gomock.Any(), "ab12.reg.voipbin.net").Return(nil, fmt.Errorf("db down"))

	res, err := h.processRequest(request)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res.StatusCode != http.StatusInternalServerError {
		t.Errorf("Wrong match. expect: %d, got: %d", http.StatusInternalServerError, res.StatusCode)
	}
}

func Test_processV1CustomerDomainsRealmGet_wrongMethod(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockCustomerDomain := customerdomainhandler.NewMockCustomerDomainHandler(mc)

	h := &listenHandler{
		sockHandler:           mockSock,
		reqHandler:            mockReq,
		customerDomainHandler: mockCustomerDomain,
	}

	// POST to the realm route: no handler matches -> 404, handler never called
	request := &sock.Request{
		URI:    "/v1/customer_domains/realm/ab12.reg.voipbin.net",
		Method: sock.RequestMethodPost,
	}

	res, err := h.processRequest(request)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("Wrong match. expect: %d, got: %d", http.StatusNotFound, res.StatusCode)
	}
}
