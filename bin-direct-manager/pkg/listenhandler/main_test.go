package listenhandler

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-direct-manager/models/direct"
	"monorepo/bin-direct-manager/pkg/dbhandler"
	"monorepo/bin-direct-manager/pkg/directhandler"
)

const testListenQueue = "bin-manager.direct-manager.request.test"

var (
	errHandlerTest = errors.New("handler failure")
	testID         = "a6b3cf48-2a4b-11ee-b574-2bad4f039ce5"
)

func Test_NewListenHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockDirect := directhandler.NewMockDirectHandler(mc)

	res := NewListenHandler(mockSock, mockDirect)
	h, ok := res.(*listenHandler)
	if !ok {
		t.Fatalf("NewListenHandler() returned %T, want *listenHandler", res)
	}

	if h.sockHandler != mockSock || h.directHandler != mockDirect {
		t.Errorf("NewListenHandler() did not store the injected dependencies in the matching fields")
	}
	if h.utilHandler == nil {
		t.Errorf("NewListenHandler() did not create the util handler")
	}
}

func Test_Run(t *testing.T) {
	t.Run("queue create fails", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockSock := sockhandler.NewMockSockHandler(mc)
		// no ConsumeRPC: the strict mock fails the test if the consumer is started anyway
		mockSock.EXPECT().QueueCreate(testListenQueue, "normal").Return(errHandlerTest)

		h := NewListenHandler(mockSock, directhandler.NewMockDirectHandler(mc))
		err := h.Run(testListenQueue, "delay")
		if !errors.Is(err, errHandlerTest) {
			t.Errorf("Run() error = %v, want %v wrapped", err, errHandlerTest)
		}
		if err != nil && !strings.Contains(err.Error(), "could not declare the queue for listenHandler") {
			t.Errorf("Run() error = %q, want the queue declaration context", err.Error())
		}
	})

	t.Run("success starts the rpc consumer", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockSock := sockhandler.NewMockSockHandler(mc)
		consuming := make(chan struct{})
		mockSock.EXPECT().QueueCreate(testListenQueue, "normal").Return(nil)
		mockSock.EXPECT().ConsumeRPC(gomock.Any(), testListenQueue, "direct-manager", false, false, false, 10, gomock.Any()).
			DoAndReturn(func(_ context.Context, _ string, _ string, _ bool, _ bool, _ bool, _ int, _ sock.CbMsgRPC) error {
				close(consuming)
				return nil
			}).Times(1)

		h := NewListenHandler(mockSock, directhandler.NewMockDirectHandler(mc))
		if err := h.Run(testListenQueue, "delay"); err != nil {
			t.Fatalf("Run() error = %v", err)
		}

		select {
		case <-consuming:
		case <-time.After(5 * time.Second):
			t.Fatalf("the rpc consumer was not started")
		}
	})
}

// Test_processRequest_errorMapping pins the status code of every handler failure as the
// dispatcher produces it today. GET by id and GET by hash hand the error to the
// dispatcher, so typed errors and dbhandler.ErrNotFound keep their own status code. The
// list, create and regenerate routes answer 500 themselves, and delete answers 400 for
// every failure, including a typed not found.
func Test_processRequest_errorMapping(t *testing.T) {
	typedNotFound := cerrors.NotFound(commonoutline.ServiceNameDirectManager, "DIRECT_NOT_FOUND", "The direct was not found.").Wrap(dbhandler.ErrNotFound)

	tests := []struct {
		name    string
		request *sock.Request

		expect       func(m *directhandler.MockDirectHandler)
		expectStatus int
	}{
		{
			name:    "get by id with a typed not found keeps its status",
			request: &sock.Request{URI: "/v1/directs/" + testID, Method: sock.RequestMethodGet},
			expect: func(m *directhandler.MockDirectHandler) {
				m.EXPECT().Get(gomock.Any(), uuid.FromStringOrNil(testID)).Return(nil, typedNotFound)
			},
			expectStatus: http.StatusNotFound,
		},
		{
			name:    "get by id with a raw not found is a 404",
			request: &sock.Request{URI: "/v1/directs/" + testID, Method: sock.RequestMethodGet},
			expect: func(m *directhandler.MockDirectHandler) {
				m.EXPECT().Get(gomock.Any(), uuid.FromStringOrNil(testID)).Return(nil, dbhandler.ErrNotFound)
			},
			expectStatus: http.StatusNotFound,
		},
		{
			name:    "get by id with another error is a 400",
			request: &sock.Request{URI: "/v1/directs/" + testID, Method: sock.RequestMethodGet},
			expect: func(m *directhandler.MockDirectHandler) {
				m.EXPECT().Get(gomock.Any(), uuid.FromStringOrNil(testID)).Return(nil, errHandlerTest)
			},
			expectStatus: http.StatusBadRequest,
		},
		{
			name:    "get by hash with a typed not found keeps its status",
			request: &sock.Request{URI: "/v1/directs/by-hash/direct.abcdef123456", Method: sock.RequestMethodGet},
			expect: func(m *directhandler.MockDirectHandler) {
				m.EXPECT().GetByHash(gomock.Any(), "direct.abcdef123456").Return(nil, typedNotFound)
			},
			expectStatus: http.StatusNotFound,
		},
		{
			name:    "get by hash with a raw not found is a 404",
			request: &sock.Request{URI: "/v1/directs/by-hash/direct.abcdef123456", Method: sock.RequestMethodGet},
			expect: func(m *directhandler.MockDirectHandler) {
				m.EXPECT().GetByHash(gomock.Any(), "direct.abcdef123456").Return(nil, dbhandler.ErrNotFound)
			},
			expectStatus: http.StatusNotFound,
		},
		{
			name:    "get by hash with another error is a 400",
			request: &sock.Request{URI: "/v1/directs/by-hash/direct.abcdef123456", Method: sock.RequestMethodGet},
			expect: func(m *directhandler.MockDirectHandler) {
				m.EXPECT().GetByHash(gomock.Any(), "direct.abcdef123456").Return(nil, errHandlerTest)
			},
			expectStatus: http.StatusBadRequest,
		},
		{
			name:    "list failure is a 500",
			request: &sock.Request{URI: "/v1/directs?page_size=10&page_token=2021-11-23T17:55:39.712000Z", Method: sock.RequestMethodGet},
			expect: func(m *directhandler.MockDirectHandler) {
				m.EXPECT().Gets(gomock.Any(), uint64(10), "2021-11-23T17:55:39.712000Z", gomock.Any()).Return(nil, errHandlerTest)
			},
			expectStatus: http.StatusInternalServerError,
		},
		{
			name:         "list with a body that is not json is a 400 and never reaches the handler",
			request:      &sock.Request{URI: "/v1/directs?page_size=10", Method: sock.RequestMethodGet, Data: []byte(`{not json`)},
			expect:       func(m *directhandler.MockDirectHandler) {},
			expectStatus: http.StatusBadRequest,
		},
		{
			name:         "list with a filter value that cannot be converted is a 400 and never reaches the handler",
			request:      &sock.Request{URI: "/v1/directs?page_size=10", Method: sock.RequestMethodGet, Data: []byte(`{"customer_id":"not-a-uuid"}`)},
			expect:       func(m *directhandler.MockDirectHandler) {},
			expectStatus: http.StatusBadRequest,
		},
		{
			name:         "list with a uri that cannot be parsed is a 400",
			request:      &sock.Request{URI: "/%zz/v1/directs?page_size=1", Method: sock.RequestMethodGet},
			expect:       func(m *directhandler.MockDirectHandler) {},
			expectStatus: http.StatusBadRequest,
		},
		{
			name: "create failure is a 500",
			request: &sock.Request{URI: "/v1/directs", Method: sock.RequestMethodPost,
				Data: []byte(`{"customer_id":"92883d56-7fe3-11ec-8931-37d08180a2b9","resource_type":"extension","resource_id":"c31676f0-4e69-11ec-afe3-77ba49fae527"}`)},
			expect: func(m *directhandler.MockDirectHandler) {
				m.EXPECT().Create(gomock.Any(), uuid.FromStringOrNil("92883d56-7fe3-11ec-8931-37d08180a2b9"), "extension", uuid.FromStringOrNil("c31676f0-4e69-11ec-afe3-77ba49fae527")).Return(nil, errHandlerTest)
			},
			expectStatus: http.StatusInternalServerError,
		},
		{
			name:         "create with a body that is not json is a 400 and never reaches the handler",
			request:      &sock.Request{URI: "/v1/directs", Method: sock.RequestMethodPost, Data: []byte(`{not json`)},
			expect:       func(m *directhandler.MockDirectHandler) {},
			expectStatus: http.StatusBadRequest,
		},
		{
			name:    "delete with another error is a 400",
			request: &sock.Request{URI: "/v1/directs/" + testID, Method: sock.RequestMethodDelete},
			expect: func(m *directhandler.MockDirectHandler) {
				m.EXPECT().Delete(gomock.Any(), uuid.FromStringOrNil(testID)).Return(nil, errHandlerTest)
			},
			expectStatus: http.StatusBadRequest,
		},
		{
			// characterization of the current behaviour: delete answers 400 even for a typed
			// not found. If delete is changed to use errorResponse, update this case to 404.
			name:    "delete with a typed not found is still a 400",
			request: &sock.Request{URI: "/v1/directs/" + testID, Method: sock.RequestMethodDelete},
			expect: func(m *directhandler.MockDirectHandler) {
				m.EXPECT().Delete(gomock.Any(), uuid.FromStringOrNil(testID)).Return(nil, typedNotFound)
			},
			expectStatus: http.StatusBadRequest,
		},
		{
			name:    "regenerate failure is a 500",
			request: &sock.Request{URI: "/v1/directs/" + testID + "/regenerate", Method: sock.RequestMethodPost},
			expect: func(m *directhandler.MockDirectHandler) {
				m.EXPECT().Regenerate(gomock.Any(), uuid.FromStringOrNil(testID)).Return(nil, errHandlerTest)
			},
			expectStatus: http.StatusInternalServerError,
		},
		{
			// characterization of the current behaviour: regenerate answers 500 even for a
			// typed not found. If regenerate is changed to use errorResponse, update to 404.
			name:    "regenerate with a typed not found is still a 500",
			request: &sock.Request{URI: "/v1/directs/" + testID + "/regenerate", Method: sock.RequestMethodPost},
			expect: func(m *directhandler.MockDirectHandler) {
				m.EXPECT().Regenerate(gomock.Any(), uuid.FromStringOrNil(testID)).Return(nil, typedNotFound)
			},
			expectStatus: http.StatusInternalServerError,
		},
		{
			name:         "unknown route is a 404",
			request:      &sock.Request{URI: "/v1/unknown", Method: sock.RequestMethodGet},
			expect:       func(m *directhandler.MockDirectHandler) {},
			expectStatus: http.StatusNotFound,
		},
		{
			name:         "known route with an unsupported method is a 404",
			request:      &sock.Request{URI: "/v1/directs/" + testID, Method: sock.RequestMethodPut},
			expect:       func(m *directhandler.MockDirectHandler) {},
			expectStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDirect := directhandler.NewMockDirectHandler(mc)
			tt.expect(mockDirect)

			h := &listenHandler{
				sockHandler:   sockhandler.NewMockSockHandler(mc),
				directHandler: mockDirect,
			}

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Fatalf("processRequest() error = %v, the dispatcher answers failures with a status code", err)
			}
			if res == nil || res.StatusCode != tt.expectStatus {
				t.Errorf("processRequest() = %+v, want status %d", res, tt.expectStatus)
			}
		})
	}
}

func Test_processRequest_typedNotFoundBody(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDirect := directhandler.NewMockDirectHandler(mc)
	mockDirect.EXPECT().Get(gomock.Any(), uuid.FromStringOrNil(testID)).
		Return(nil, cerrors.NotFound(commonoutline.ServiceNameDirectManager, "DIRECT_NOT_FOUND", "The direct was not found.").Wrap(dbhandler.ErrNotFound))

	h := &listenHandler{directHandler: mockDirect}
	res, err := h.processRequest(&sock.Request{URI: "/v1/directs/" + testID, Method: sock.RequestMethodGet})
	if err != nil {
		t.Fatalf("processRequest() error = %v", err)
	}

	if res.DataType != cerrors.DataTypeVoipbinError {
		t.Errorf("DataType = %q, want %q", res.DataType, cerrors.DataTypeVoipbinError)
	}
	if !strings.Contains(string(res.Data), "DIRECT_NOT_FOUND") {
		t.Errorf("body = %s, want the typed reason DIRECT_NOT_FOUND", res.Data)
	}
}

// Test_processV1Directs_shortURI covers the guards against uris that are too short.
// The dispatcher regexes never route such uris, so the handlers are called directly.
func Test_processV1Directs_shortURI(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	// no handler method may be called: the strict mock fails the test otherwise
	h := &listenHandler{directHandler: directhandler.NewMockDirectHandler(mc)}
	ctx := context.Background()

	tests := []struct {
		name string
		call func(m *sock.Request) (*sock.Response, error)
		uri  string
	}{
		{"get by id", func(m *sock.Request) (*sock.Response, error) { return h.processV1DirectsIDGet(ctx, m) }, "/v1"},
		{"delete", func(m *sock.Request) (*sock.Response, error) { return h.processV1DirectsIDDelete(ctx, m) }, "/v1"},
		{"get by hash", func(m *sock.Request) (*sock.Response, error) { return h.processV1DirectsByHashGet(ctx, m) }, "/v1/directs"},
		{"regenerate", func(m *sock.Request) (*sock.Response, error) { return h.processV1DirectsIDRegenerate(ctx, m) }, "/v1/directs"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := tt.call(&sock.Request{URI: tt.uri})
			if err != nil {
				t.Fatalf("error = %v", err)
			}
			if res.StatusCode != http.StatusBadRequest {
				t.Errorf("status = %d, want %d", res.StatusCode, http.StatusBadRequest)
			}
		})
	}
}

func Test_processV1DirectsGet_typedFilters(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	customerID := uuid.FromStringOrNil("92883d56-7fe3-11ec-8931-37d08180a2b9")
	mockDirect := directhandler.NewMockDirectHandler(mc)
	mockDirect.EXPECT().Gets(gomock.Any(), uint64(5), "tok", map[direct.Field]any{direct.FieldCustomerID: customerID}).Return([]*direct.Direct{}, nil)

	h := &listenHandler{directHandler: mockDirect}
	res, err := h.processRequest(&sock.Request{
		URI:    "/v1/directs?page_size=5&page_token=tok",
		Method: sock.RequestMethodGet,
		Data:   []byte(`{"customer_id":"92883d56-7fe3-11ec-8931-37d08180a2b9"}`),
	})
	if err != nil {
		t.Fatalf("processRequest() error = %v", err)
	}
	if res.StatusCode != http.StatusOK || string(res.Data) != "[]" {
		t.Errorf("processRequest() = %+v, want 200 and an empty list", res)
	}
}
