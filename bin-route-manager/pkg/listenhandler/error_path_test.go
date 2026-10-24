package listenhandler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	cerrors "monorepo/bin-common-handler/models/errors"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"go.uber.org/mock/gomock"

	"monorepo/bin-route-manager/pkg/dbhandler"
	"monorepo/bin-route-manager/pkg/providercallhandler"
	"monorepo/bin-route-manager/pkg/providerhandler"
	"monorepo/bin-route-manager/pkg/routehandler"
)

const (
	testRouteID    = "11111111-1111-4111-8111-111111111111"
	testProviderID = "22222222-2222-4222-8222-222222222222"
)

type testMocks struct {
	sock         *sockhandler.MockSockHandler
	provider     *providerhandler.MockProviderHandler
	providerCall *providercallhandler.MockProviderCallHandler
	route        *routehandler.MockRouteHandler
}

func newTestListenHandler(mc *gomock.Controller) (*listenHandler, *testMocks) {
	m := &testMocks{
		sock:         sockhandler.NewMockSockHandler(mc),
		provider:     providerhandler.NewMockProviderHandler(mc),
		providerCall: providercallhandler.NewMockProviderCallHandler(mc),
		route:        routehandler.NewMockRouteHandler(mc),
	}

	h := &listenHandler{
		sockHandler:         m.sock,
		providerHandler:     m.provider,
		providerCallHandler: m.providerCall,
		routeHandler:        m.route,
	}

	return h, m
}

type handlerFunc func(h *listenHandler, ctx context.Context, m *sock.Request) (*sock.Response, error)

func Test_handlers_invalidBody(t *testing.T) {
	tests := []struct {
		name    string
		fn      handlerFunc
		uri     string
		method  sock.RequestMethod
		body    string
		wantNil bool
	}{
		{"routes post", (*listenHandler).v1RoutesPost, "/v1/routes", sock.RequestMethodPost, `{"customer_id":`, true},
		{"routes get", (*listenHandler).v1RoutesGet, "/v1/routes?page_size=10", sock.RequestMethodGet, `{"customer_id":`, true},
		{"routes id put", (*listenHandler).v1RoutesIDPut, "/v1/routes/" + testRouteID, sock.RequestMethodPut, `{"name":`, true},
		{"providers post", (*listenHandler).v1ProvidersPost, "/v1/providers", sock.RequestMethodPost, `{"type":`, true},
		{"providers id put", (*listenHandler).v1ProvidersIDPut, "/v1/providers/" + testProviderID, sock.RequestMethodPut, `{"name":`, true},
		{"providers setup post", (*listenHandler).v1ProvidersSetupPost, "/v1/providers/setup", sock.RequestMethodPost, `{"carrier":`, true},
		{"dialroutes get", (*listenHandler).v1DialroutesGet, "/v1/dialroutes", sock.RequestMethodGet, `{"filters":`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			// no downstream expectations: any call to a business handler fails the test
			h, _ := newTestListenHandler(mc)

			res, err := tt.fn(h, context.Background(), &sock.Request{
				URI:      tt.uri,
				Method:   tt.method,
				DataType: "application/json",
				Data:     []byte(tt.body),
			})
			if err == nil {
				t.Errorf("Wrong match. expect: error, got: ok")
			}
			if tt.wantNil && res != nil {
				t.Errorf("Wrong match. expect: nil response, got: %+v", res)
			}
		})
	}
}

func Test_handlers_invalidURI(t *testing.T) {
	// "%zz" is an invalid escape, so url.Parse fails before any business handler is reached.
	badURI := "/v1/%zz"

	tests := []struct {
		name string
		fn   handlerFunc
		body string
	}{
		{"routes get", (*listenHandler).v1RoutesGet, ``},
		{"routes id get", (*listenHandler).v1RoutesIDGet, ``},
		{"routes id put", (*listenHandler).v1RoutesIDPut, `{}`},
		{"routes id delete", (*listenHandler).v1RoutesIDDelete, ``},
		{"providers get", (*listenHandler).v1ProvidersGet, ``},
		{"providers id get", (*listenHandler).v1ProvidersIDGet, ``},
		{"providers id put", (*listenHandler).v1ProvidersIDPut, `{}`},
		{"providers id delete", (*listenHandler).v1ProvidersIDDelete, ``},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, _ := newTestListenHandler(mc)

			res, err := tt.fn(h, context.Background(), &sock.Request{
				URI:    badURI,
				Method: sock.RequestMethodGet,
				Data:   []byte(tt.body),
			})
			if err == nil {
				t.Errorf("Wrong match. expect: error, got: ok")
			}
			if res != nil {
				t.Errorf("Wrong match. expect: nil response, got: %+v", res)
			}
		})
	}
}

func Test_routes_invalidID(t *testing.T) {
	// the id segment is not a UUID. processRequest never routes such a uri here, so call the handlers directly.
	tests := []struct {
		name   string
		fn     handlerFunc
		method sock.RequestMethod
		data   string
	}{
		{"get", (*listenHandler).v1RoutesIDGet, sock.RequestMethodGet, ``},
		{"put", (*listenHandler).v1RoutesIDPut, sock.RequestMethodPut, `{"name":"x"}`},
		{"delete", (*listenHandler).v1RoutesIDDelete, sock.RequestMethodDelete, ``},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, _ := newTestListenHandler(mc)

			res, err := tt.fn(h, context.Background(), &sock.Request{
				URI:    "/v1/routes/not-a-uuid",
				Method: tt.method,
				Data:   []byte(tt.data),
			})
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res == nil || res.StatusCode != http.StatusBadRequest {
				t.Errorf("Wrong match. expect: 400, got: %+v", res)
			}
		})
	}
}

func Test_routes_businessError(t *testing.T) {
	routeID := uuid.FromStringOrNil(testRouteID)
	customerID := uuid.FromStringOrNil("33333333-3333-4333-8333-333333333333")

	tests := []struct {
		name         string
		fn           handlerFunc
		uri          string
		method       sock.RequestMethod
		data         string
		setup        func(m *testMocks, err error)
		err          error
		expectStatus int // 0 means the handler returns the error itself
	}{
		{
			name: "post", fn: (*listenHandler).v1RoutesPost, uri: "/v1/routes", method: sock.RequestMethodPost,
			data: `{"customer_id":"33333333-3333-4333-8333-333333333333","name":"n","detail":"d","provider_id":"22222222-2222-4222-8222-222222222222","priority":1,"target":"82"}`,
			setup: func(m *testMocks, err error) {
				m.route.EXPECT().Create(gomock.Any(), customerID, "n", "d", uuid.FromStringOrNil(testProviderID), 1, "82").Return(nil, err)
			},
			err: errors.New("create failed"),
		},
		{
			name: "get list", fn: (*listenHandler).v1RoutesGet, uri: "/v1/routes?page_size=5&page_token=tok", method: sock.RequestMethodGet,
			data: `{"customer_id":"33333333-3333-4333-8333-333333333333"}`,
			setup: func(m *testMocks, err error) {
				m.route.EXPECT().ListByCustomerID(gomock.Any(), customerID, "tok", uint64(5)).Return(nil, err)
			},
			err: errors.New("list failed"),
		},
		{
			name: "get by id not found", fn: (*listenHandler).v1RoutesIDGet, uri: "/v1/routes/" + testRouteID, method: sock.RequestMethodGet,
			setup: func(m *testMocks, err error) {
				m.route.EXPECT().Get(gomock.Any(), routeID).Return(nil, err)
			},
			err: dbhandler.ErrNotFound, expectStatus: http.StatusNotFound,
		},
		{
			name: "put internal error", fn: (*listenHandler).v1RoutesIDPut, uri: "/v1/routes/" + testRouteID, method: sock.RequestMethodPut,
			data: `{"name":"new"}`,
			setup: func(m *testMocks, err error) {
				name := "new"
				m.route.EXPECT().Update(gomock.Any(), routeID, &name, nil, nil, nil, nil).Return(nil, err)
			},
			err: errors.New("update failed"), expectStatus: http.StatusInternalServerError,
		},
		{
			name: "delete not found", fn: (*listenHandler).v1RoutesIDDelete, uri: "/v1/routes/" + testRouteID, method: sock.RequestMethodDelete,
			setup: func(m *testMocks, err error) {
				m.route.EXPECT().Delete(gomock.Any(), routeID).Return(nil, err)
			},
			err: dbhandler.ErrNotFound, expectStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newTestListenHandler(mc)
			tt.setup(m, tt.err)

			res, err := tt.fn(h, context.Background(), &sock.Request{URI: tt.uri, Method: tt.method, Data: []byte(tt.data)})
			if tt.expectStatus == 0 {
				if err == nil || res != nil {
					t.Errorf("Wrong match. expect: error and nil response, got: %v, %+v", err, res)
				}
				return
			}

			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res == nil || res.StatusCode != tt.expectStatus {
				t.Errorf("Wrong match. expect: %d, got: %+v", tt.expectStatus, res)
			}
		})
	}
}

func Test_providers_businessError(t *testing.T) {
	providerID := uuid.FromStringOrNil(testProviderID)

	tests := []struct {
		name   string
		fn     handlerFunc
		uri    string
		method sock.RequestMethod
		data   string
		setup  func(m *testMocks, err error)
	}{
		{
			name: "post", fn: (*listenHandler).v1ProvidersPost, uri: "/v1/providers", method: sock.RequestMethodPost,
			data: `{"type":"sip","hostname":"sip.example.invalid","name":"n"}`,
			setup: func(m *testMocks, err error) {
				m.provider.EXPECT().Create(gomock.Any(), gomock.Any(), "sip.example.invalid", "", "", gomock.Any(), "n", "", "").Return(nil, err)
			},
		},
		{
			name: "get list", fn: (*listenHandler).v1ProvidersGet, uri: "/v1/providers?page_size=3&page_token=tok", method: sock.RequestMethodGet,
			setup: func(m *testMocks, err error) {
				m.provider.EXPECT().List(gomock.Any(), "tok", uint64(3)).Return(nil, err)
			},
		},
		{
			name: "get by id", fn: (*listenHandler).v1ProvidersIDGet, uri: "/v1/providers/" + testProviderID, method: sock.RequestMethodGet,
			setup: func(m *testMocks, err error) {
				m.provider.EXPECT().Get(gomock.Any(), providerID).Return(nil, err)
			},
		},
		{
			name: "put", fn: (*listenHandler).v1ProvidersIDPut, uri: "/v1/providers/" + testProviderID, method: sock.RequestMethodPut,
			data: `{"name":"new"}`,
			setup: func(m *testMocks, err error) {
				m.provider.EXPECT().Update(gomock.Any(), providerID, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, err)
			},
		},
		{
			name: "delete", fn: (*listenHandler).v1ProvidersIDDelete, uri: "/v1/providers/" + testProviderID, method: sock.RequestMethodDelete,
			setup: func(m *testMocks, err error) {
				m.provider.EXPECT().Delete(gomock.Any(), providerID).Return(nil, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newTestListenHandler(mc)
			expectErr := errors.New("business failure")
			tt.setup(m, expectErr)

			res, err := tt.fn(h, context.Background(), &sock.Request{URI: tt.uri, Method: tt.method, Data: []byte(tt.data)})
			if !errors.Is(err, expectErr) {
				t.Errorf("Wrong match. expect: %v, got: %v", expectErr, err)
			}
			if res != nil {
				t.Errorf("Wrong match. expect: nil response, got: %+v", res)
			}
		})
	}
}

func Test_dialroutes_businessError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, m := newTestListenHandler(mc)
	customerID := uuid.FromStringOrNil("33333333-3333-4333-8333-333333333333")
	expectErr := errors.New("dialroute failed")

	m.route.EXPECT().DialrouteList(gomock.Any(), customerID, "82", []uuid.UUID(nil)).Return(nil, expectErr)

	res, err := h.v1DialroutesGet(context.Background(), &sock.Request{
		URI:    "/v1/dialroutes",
		Method: sock.RequestMethodGet,
		Data:   []byte(`{"customer_id":"33333333-3333-4333-8333-333333333333","target":"82"}`),
	})
	if !errors.Is(err, expectErr) {
		t.Errorf("Wrong match. expect: %v, got: %v", expectErr, err)
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil response, got: %+v", res)
	}
}

func Test_processRequest_errorMapping(t *testing.T) {
	routeID := uuid.FromStringOrNil(testRouteID)
	typedNotFound := cerrors.NotFound("route-manager", "ROUTE_NOT_FOUND", "missing")

	tests := []struct {
		name         string
		request      *sock.Request
		setup        func(m *testMocks)
		expectStatus int
	}{
		{
			name:         "unsupported path",
			request:      &sock.Request{URI: "/v1/unknown", Method: sock.RequestMethodGet},
			setup:        func(m *testMocks) {},
			expectStatus: http.StatusNotFound,
		},
		{
			name:         "unsupported method on a known path",
			request:      &sock.Request{URI: "/v1/routes/" + testRouteID, Method: sock.RequestMethodPost},
			setup:        func(m *testMocks) {},
			expectStatus: http.StatusNotFound,
		},
		{
			name:    "plain error becomes 400",
			request: &sock.Request{URI: "/v1/providers?page_size=1", Method: sock.RequestMethodGet},
			setup: func(m *testMocks) {
				m.provider.EXPECT().List(gomock.Any(), "", uint64(1)).Return(nil, errors.New("boom"))
			},
			expectStatus: http.StatusBadRequest,
		},
		{
			name:    "legacy not found becomes 404",
			request: &sock.Request{URI: "/v1/providers/" + testProviderID, Method: sock.RequestMethodGet},
			setup: func(m *testMocks) {
				m.provider.EXPECT().Get(gomock.Any(), uuid.FromStringOrNil(testProviderID)).Return(nil, dbhandler.ErrNotFound)
			},
			expectStatus: http.StatusNotFound,
		},
		{
			name:    "typed error keeps its status",
			request: &sock.Request{URI: "/v1/providers/" + testProviderID, Method: sock.RequestMethodGet},
			setup: func(m *testMocks) {
				m.provider.EXPECT().Get(gomock.Any(), uuid.FromStringOrNil(testProviderID)).Return(nil, typedNotFound)
			},
			expectStatus: http.StatusNotFound,
		},
		{
			name:         "broken body becomes 400",
			request:      &sock.Request{URI: "/v1/routes", Method: sock.RequestMethodPost, Data: []byte(`{"name":`)},
			setup:        func(m *testMocks) {},
			expectStatus: http.StatusBadRequest,
		},
		{
			name:         "route lookup error is mapped by errorResponse",
			request:      &sock.Request{URI: "/v1/routes/" + testRouteID, Method: sock.RequestMethodGet},
			setup:        func(m *testMocks) { m.route.EXPECT().Get(gomock.Any(), routeID).Return(nil, errors.New("db down")) },
			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newTestListenHandler(mc)
			tt.setup(m)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res == nil || res.StatusCode != tt.expectStatus {
				t.Errorf("Wrong match. expect: %d, got: %+v", tt.expectStatus, res)
			}
		})
	}
}

func Test_processRequest_providersSetup(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, m := newTestListenHandler(mc)

	m.provider.EXPECT().Setup(gomock.Any(), "telnyx", "name", "detail", "synthetic-key").Return(nil, errors.New("setup failed"))

	res, err := h.processRequest(&sock.Request{
		URI:    "/v1/providers/setup",
		Method: sock.RequestMethodPost,
		Data:   []byte(`{"carrier":"telnyx","name":"name","detail":"detail","credentials":{"api_key":"synthetic-key"}}`),
	})
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res == nil || res.StatusCode != http.StatusBadRequest {
		t.Errorf("Wrong match. expect: 400, got: %+v", res)
	}
}

func Test_errorResponse_typedNilVoipbinError(t *testing.T) {
	// a typed nil pointer satisfies errors.As but cannot be converted to a response
	var ve *cerrors.VoipbinError
	var err error = ve

	res := errorResponse(err)
	if res == nil || res.StatusCode != http.StatusInternalServerError {
		t.Errorf("Wrong match. expect: 500, got: %+v", res)
	}
}

func Test_NewListenHandler_dispatches(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockProvider := providerhandler.NewMockProviderHandler(mc)
	mockProviderCall := providercallhandler.NewMockProviderCallHandler(mc)
	mockRoute := routehandler.NewMockRouteHandler(mc)

	h := NewListenHandler(mockSock, mockProvider, mockProviderCall, mockRoute)

	// the handler built by the constructor must reach the injected route handler
	routeID := uuid.FromStringOrNil(testRouteID)
	mockRoute.EXPECT().Delete(gomock.Any(), routeID).Return(nil, dbhandler.ErrNotFound)

	lh, ok := h.(*listenHandler)
	if !ok {
		t.Fatalf("Wrong match. expect: *listenHandler, got: %T", h)
	}

	res, err := lh.processRequest(&sock.Request{URI: "/v1/routes/" + testRouteID, Method: sock.RequestMethodDelete})
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("Wrong match. expect: 404, got: %d", res.StatusCode)
	}
}

func Test_Run(t *testing.T) {
	t.Run("queue creation failure is returned", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newTestListenHandler(mc)
		m.sock.EXPECT().QueueCreate("test.queue", "normal").Return(errors.New("queue failed"))

		if err := h.Run("test.queue", ""); err == nil {
			t.Errorf("Wrong match. expect: error, got: ok")
		}
	})

	t.Run("consume starts on the created queue", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newTestListenHandler(mc)

		var wg sync.WaitGroup
		wg.Add(1)
		m.sock.EXPECT().QueueCreate("test.queue", "normal").Return(nil)
		m.sock.EXPECT().ConsumeRPC(gomock.Any(), "test.queue", "route-manager", false, false, false, 10, gomock.Any()).
			DoAndReturn(func(_ context.Context, _ string, _ string, _ bool, _ bool, _ bool, _ int, _ sock.CbMsgRPC) error {
				defer wg.Done()
				return errors.New("consume failed")
			})

		if err := h.Run("test.queue", ""); err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}

		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()

		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatalf("Wrong match. expect: ConsumeRPC called, got: timeout")
		}
	})
}

func Test_processRequest_providersSetup_credentialNotLogged(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, m := newTestListenHandler(mc)

	prevLevel := logrus.GetLevel()
	logrus.SetLevel(logrus.DebugLevel)
	hook := logtest.NewGlobal()
	t.Cleanup(func() {
		logrus.SetLevel(prevLevel)
		hook.Reset()
	})

	credential := "synthetic-credential-value"
	m.provider.EXPECT().Setup(gomock.Any(), "telnyx", "name", "detail", credential).Return(nil, errors.New("setup failed"))

	if _, err := h.processRequest(&sock.Request{
		URI:    "/v1/providers/setup",
		Method: sock.RequestMethodPost,
		Data:   []byte(`{"carrier":"telnyx","name":"name","detail":"detail","credentials":{"api_key":"` + credential + `"}}`),
	}); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	redacted := false
	for _, e := range hook.AllEntries() {
		if strings.Contains(e.Message, credential) {
			t.Errorf("Wrong match. expect: credential absent from the log message, got: %s", e.Message)
		}
		for k, v := range e.Data {
			var text string
			if b, ok := v.([]byte); ok {
				text = string(b)
			} else {
				text = fmt.Sprint(v)
			}

			if strings.Contains(text, credential) {
				t.Errorf("Wrong match. expect: credential absent from log field %s, got: %s", k, text)
			}
			if k == "data" && text == "[redacted]" {
				redacted = true
			}
		}
	}
	if !redacted {
		t.Errorf("Wrong match. expect: the request body logged as [redacted], got: not found")
	}
}
