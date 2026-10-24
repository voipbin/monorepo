package listenhandler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	cerrors "monorepo/bin-common-handler/models/errors"
	"monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-webchat-manager/pkg/dbhandler"
	"monorepo/bin-webchat-manager/pkg/messagehandler"
	"monorepo/bin-webchat-manager/pkg/sessionhandler"
	"monorepo/bin-webchat-manager/pkg/widgethandler"
)

const (
	testIDWidget  = "11111111-aaaa-bbbb-cccc-000000000001"
	testIDSession = "11111111-aaaa-bbbb-cccc-000000000002"
	testIDMessage = "11111111-aaaa-bbbb-cccc-000000000003"
	testIDCust    = "11111111-aaaa-bbbb-cccc-000000000004"
	testIDOther   = "11111111-aaaa-bbbb-cccc-000000000005"
	testIDNil     = "00000000-0000-0000-0000-000000000000"
)

type testMocks struct {
	sock    *sockhandler.MockSockHandler
	widget  *widgethandler.MockWidgetHandler
	session *sessionhandler.MockSessionHandler
	message *messagehandler.MockMessageHandler
}

func newTestHandler(mc *gomock.Controller) (*listenHandler, *testMocks) {
	m := &testMocks{
		sock:    sockhandler.NewMockSockHandler(mc),
		widget:  widgethandler.NewMockWidgetHandler(mc),
		session: sessionhandler.NewMockSessionHandler(mc),
		message: messagehandler.NewMockMessageHandler(mc),
	}
	h := &listenHandler{
		sockHandler:    m.sock,
		widgetHandler:  m.widget,
		sessionHandler: m.session,
		messageHandler: m.message,
	}
	return h, m
}

func newTestRequest(method sock.RequestMethod, uri string, data string) *sock.Request {
	r := &sock.Request{
		URI:      uri,
		Method:   method,
		DataType: "application/json",
	}
	if data != "" {
		r.Data = json.RawMessage(data)
	}
	return r
}

type handlerFunc func(h *listenHandler, ctx context.Context, m *sock.Request) (*sock.Response, error)

// assertIDGuards checks that an id route rejects a short uri and a nil uuid with 400
// before any business handler is called (the strict mocks fail on an unexpected call).
func assertIDGuards(t *testing.T, fn handlerFunc, method sock.RequestMethod, resource string, suffix string, data string) {
	t.Helper()

	tests := []struct {
		name string
		uri  string
	}{
		{"short uri", "/v1"},
		{"missing id segment", "/v1/" + resource},
		{"nil uuid", fmt.Sprintf("/v1/%s/%s%s", resource, testIDNil, suffix)},
		{"malformed uuid", fmt.Sprintf("/v1/%s/not-a-uuid%s", resource, suffix)},
	}

	for _, tt := range tests {
		t.Run("guard_"+tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, _ := newTestHandler(mc)

			res, err := fn(h, context.Background(), newTestRequest(method, tt.uri, data))
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res.StatusCode != http.StatusBadRequest {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusBadRequest, res.StatusCode)
			}
		})
	}
}

// assertErrorMapping checks how a business handler error is mapped to the response.
func assertErrorMapping(
	t *testing.T,
	setup func(m *testMocks, err error),
	invoke func(h *listenHandler) (*sock.Response, error),
) {
	t.Helper()

	notFound := cerrors.NotFound(outline.ServiceNameWebchatManager, "TEST", "not found")
	denied := cerrors.PermissionDenied(outline.ServiceNameWebchatManager, "TEST", "denied")

	tests := []struct {
		name         string
		err          error
		expectStatus int
	}{
		{"legacy not found", dbhandler.ErrNotFound, http.StatusNotFound},
		{"wrapped not found", fmt.Errorf("wrap: %w", dbhandler.ErrNotFound), http.StatusNotFound},
		{"typed not found", notFound, http.StatusNotFound},
		{"typed permission denied", denied, http.StatusForbidden},
		{"other error", errors.New("boom"), http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run("error_"+tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newTestHandler(mc)
			setup(m, tt.err)

			res, err := invoke(h)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res.StatusCode != tt.expectStatus {
				t.Errorf("Wrong match. expect: %d, got: %d", tt.expectStatus, res.StatusCode)
			}
		})
	}
}

func Test_errorResponse(t *testing.T) {
	denied := cerrors.PermissionDenied(outline.ServiceNameWebchatManager, "TEST", "denied")
	var typedNil *cerrors.VoipbinError

	tests := []struct {
		name string

		err error

		expectStatus   int
		expectDataType string
	}{
		{"nil error", nil, http.StatusInternalServerError, ""},
		{"typed error", denied, http.StatusForbidden, cerrors.DataTypeVoipbinError},
		{"wrapped typed error", fmt.Errorf("wrap: %w", denied), http.StatusForbidden, cerrors.DataTypeVoipbinError},
		{"typed nil pointer fails to encode", typedNil, http.StatusInternalServerError, ""},
		{"legacy not found", dbhandler.ErrNotFound, http.StatusNotFound, ""},
		{"wrapped legacy not found", fmt.Errorf("wrap: %w", dbhandler.ErrNotFound), http.StatusNotFound, ""},
		{"other error", errors.New("boom"), http.StatusInternalServerError, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := errorResponse(tt.err)
			if res.StatusCode != tt.expectStatus {
				t.Errorf("Wrong match. expect: %d, got: %d", tt.expectStatus, res.StatusCode)
			}
			if res.DataType != tt.expectDataType {
				t.Errorf("Wrong match. expect: %q, got: %q", tt.expectDataType, res.DataType)
			}
		})
	}
}

func Test_errorResponse_typedBody(t *testing.T) {
	e := cerrors.PermissionDenied(outline.ServiceNameWebchatManager, "TEST_REASON", "denied message")

	res := errorResponse(e)

	var got cerrors.VoipbinError
	if err := json.Unmarshal(res.Data, &got); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if got.Status != cerrors.StatusPermissionDenied || got.Reason != "TEST_REASON" || got.Message != "denied message" {
		t.Errorf("Wrong match. got: %+v", got)
	}
}

func Test_simpleResponse(t *testing.T) {
	res := simpleResponse(http.StatusTeapot)
	if res.StatusCode != http.StatusTeapot || len(res.Data) != 0 || res.DataType != "" {
		t.Errorf("Wrong match. got: %+v", res)
	}
}

func Test_NewListenHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockWidget := widgethandler.NewMockWidgetHandler(mc)
	mockSession := sessionhandler.NewMockSessionHandler(mc)
	mockMessage := messagehandler.NewMockMessageHandler(mc)

	res := NewListenHandler(mockSock, mockWidget, mockSession, mockMessage)

	h, ok := res.(*listenHandler)
	if !ok {
		t.Fatalf("Wrong type. got: %T", res)
	}
	if h.sockHandler != mockSock || h.widgetHandler != mockWidget || h.sessionHandler != mockSession || h.messageHandler != mockMessage {
		t.Errorf("handlers are not wired to the given instances")
	}
	if h.utilHanlder == nil {
		t.Errorf("utilHanlder must be initialized")
	}
}

func Test_Run(t *testing.T) {
	t.Run("queue create error", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newTestHandler(mc)

		m.sock.EXPECT().QueueCreate("test-queue", "normal").Return(errors.New("queue failure"))

		if err := h.Run("test-queue", "delay"); err == nil {
			t.Errorf("Wrong match. expect: error, got: nil")
		}
	})

	t.Run("consume started", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newTestHandler(mc)

		called := make(chan sock.CbMsgRPC, 1)
		m.sock.EXPECT().QueueCreate("test-queue", "normal").Return(nil)
		m.sock.EXPECT().ConsumeRPC(gomock.Any(), "test-queue", "webchat-manager", false, false, false, 10, gomock.Any()).
			DoAndReturn(func(_ context.Context, _ string, _ string, _ bool, _ bool, _ bool, _ int, cb sock.CbMsgRPC) error {
				called <- cb
				return nil
			})

		if err := h.Run("test-queue", "delay"); err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}

		select {
		case cb := <-called:
			if cb == nil {
				t.Errorf("the consume callback must not be nil")
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("ConsumeRPC was not called")
		}
	})

	t.Run("consume error is only logged", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newTestHandler(mc)

		called := make(chan struct{}, 1)
		m.sock.EXPECT().QueueCreate("test-queue", "normal").Return(nil)
		m.sock.EXPECT().ConsumeRPC(gomock.Any(), "test-queue", "webchat-manager", false, false, false, 10, gomock.Any()).
			DoAndReturn(func(_ context.Context, _ string, _ string, _ bool, _ bool, _ bool, _ int, _ sock.CbMsgRPC) error {
				defer close(called)
				return errors.New("consume failure")
			})

		if err := h.Run("test-queue", "delay"); err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}

		select {
		case <-called:
		case <-time.After(2 * time.Second):
			t.Fatalf("ConsumeRPC was not called")
		}
	})
}

func Test_processRequest_routing(t *testing.T) {
	widgetID := uuid.FromStringOrNil(testIDWidget)
	sessionID := uuid.FromStringOrNil(testIDSession)
	messageID := uuid.FromStringOrNil(testIDMessage)

	tests := []struct {
		name string

		method sock.RequestMethod
		uri    string
		data   string

		setup func(m *testMocks)
	}{
		{
			"widgets get", sock.RequestMethodGet, "/v1/widgets?page_size=5", "",
			func(m *testMocks) {
				m.widget.EXPECT().List(gomock.Any(), uint64(5), "", gomock.Any()).Return(nil, nil)
			},
		},
		{
			"widgets post", sock.RequestMethodPost, "/v1/widgets", `{}`,
			func(m *testMocks) {
				m.widget.EXPECT().Create(gomock.Any(), uuid.Nil, "", uuid.Nil, uuid.Nil, 0, nil).Return(nil, nil)
			},
		},
		{
			"widgets id get", sock.RequestMethodGet, "/v1/widgets/" + testIDWidget, "",
			func(m *testMocks) {
				m.widget.EXPECT().Get(gomock.Any(), widgetID).Return(nil, nil)
			},
		},
		{
			"widgets id delete", sock.RequestMethodDelete, "/v1/widgets/" + testIDWidget, "",
			func(m *testMocks) {
				m.widget.EXPECT().Delete(gomock.Any(), widgetID).Return(nil, nil)
			},
		},
		{
			"widgets id put", sock.RequestMethodPut, "/v1/widgets/" + testIDWidget, `{}`,
			func(m *testMocks) {
				m.widget.EXPECT().UpdateBasicInfo(gomock.Any(), widgetID, "", uuid.Nil, uuid.Nil, 0, nil).Return(nil, nil)
			},
		},
		{
			"widgets direct hash regenerate", sock.RequestMethodPost, "/v1/widgets/" + testIDWidget + "/direct-hash-regenerate", "",
			func(m *testMocks) {
				m.widget.EXPECT().DirectHashRegenerate(gomock.Any(), widgetID).Return(nil, nil)
			},
		},
		{
			"sessions get", sock.RequestMethodGet, "/v1/sessions?page_size=7&page_token=tok", "",
			func(m *testMocks) {
				m.session.EXPECT().List(gomock.Any(), uint64(7), "tok", gomock.Any()).Return(nil, nil)
			},
		},
		{
			"sessions post", sock.RequestMethodPost, "/v1/sessions", `{}`,
			func(m *testMocks) {
				m.session.EXPECT().Create(gomock.Any(), uuid.Nil, uuid.Nil, uuid.Nil, "", "").Return(nil, nil)
			},
		},
		{
			"sessions id get", sock.RequestMethodGet, "/v1/sessions/" + testIDSession, "",
			func(m *testMocks) {
				m.session.EXPECT().Get(gomock.Any(), sessionID).Return(nil, nil)
			},
		},
		{
			"sessions id delete", sock.RequestMethodDelete, "/v1/sessions/" + testIDSession, "",
			func(m *testMocks) {
				m.session.EXPECT().Delete(gomock.Any(), sessionID).Return(nil, nil)
			},
		},
		{
			"sessions end", sock.RequestMethodPost, "/v1/sessions/" + testIDSession + "/end", "",
			func(m *testMocks) {
				m.session.EXPECT().End(gomock.Any(), sessionID).Return(nil, nil)
			},
		},
		{
			"messages get", sock.RequestMethodGet, "/v1/messages?page_size=3", "",
			func(m *testMocks) {
				m.message.EXPECT().List(gomock.Any(), uint64(3), "", gomock.Any()).Return(nil, nil)
			},
		},
		{
			"messages post", sock.RequestMethodPost, "/v1/messages", `{}`,
			func(m *testMocks) {
				m.message.EXPECT().Create(gomock.Any(), uuid.Nil, uuid.Nil, gomock.Any(), uuid.Nil, "").Return(nil, nil)
			},
		},
		{
			"messages id get", sock.RequestMethodGet, "/v1/messages/" + testIDMessage, "",
			func(m *testMocks) {
				m.message.EXPECT().Get(gomock.Any(), messageID).Return(nil, nil)
			},
		},
		{
			"messages id delete", sock.RequestMethodDelete, "/v1/messages/" + testIDMessage, "",
			func(m *testMocks) {
				m.message.EXPECT().Delete(gomock.Any(), messageID).Return(nil, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newTestHandler(mc)
			tt.setup(m)

			res, err := h.processRequest(newTestRequest(tt.method, tt.uri, tt.data))
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res.StatusCode != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, res.StatusCode)
			}
		})
	}
}

func Test_processRequest_notFound(t *testing.T) {
	tests := []struct {
		name   string
		method sock.RequestMethod
		uri    string
	}{
		{"unknown path", sock.RequestMethodGet, "/v1/unknown"},
		{"unknown version", sock.RequestMethodGet, "/v2/widgets/" + testIDWidget + "x"},
		{"widgets collection put", sock.RequestMethodPut, "/v1/widgets"},
		{"widgets collection delete", sock.RequestMethodDelete, "/v1/widgets"},
		{"widgets id post", sock.RequestMethodPost, "/v1/widgets/" + testIDWidget},
		{"widgets regenerate get", sock.RequestMethodGet, "/v1/widgets/" + testIDWidget + "/direct-hash-regenerate"},
		{"widgets id with suffix", sock.RequestMethodGet, "/v1/widgets/" + testIDWidget + "/x"},
		{"widgets regenerate with suffix", sock.RequestMethodPost, "/v1/widgets/" + testIDWidget + "/direct-hash-regenerate/x"},
		{"widgets collection with suffix", sock.RequestMethodPost, "/v1/widgets/x"},
		{"widgets non uuid id", sock.RequestMethodGet, "/v1/widgets/not-a-uuid"},
		{"sessions end get", sock.RequestMethodGet, "/v1/sessions/" + testIDSession + "/end"},
		{"sessions end with suffix", sock.RequestMethodPost, "/v1/sessions/" + testIDSession + "/end/x"},
		{"sessions collection with suffix", sock.RequestMethodPost, "/v1/sessions/x"},
		{"sessions id put", sock.RequestMethodPut, "/v1/sessions/" + testIDSession},
		{"sessions id with suffix", sock.RequestMethodDelete, "/v1/sessions/" + testIDSession + "/x"},
		{"messages id with suffix", sock.RequestMethodGet, "/v1/messages/" + testIDMessage + "/x"},
		{"messages collection with suffix", sock.RequestMethodPost, "/v1/messages/x"},
		{"messages id put", sock.RequestMethodPut, "/v1/messages/" + testIDMessage},
		{"messages id post", sock.RequestMethodPost, "/v1/messages/" + testIDMessage},
		{"messages collection delete", sock.RequestMethodDelete, "/v1/messages"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, _ := newTestHandler(mc)

			res, err := h.processRequest(newTestRequest(tt.method, tt.uri, ""))
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res.StatusCode != http.StatusNotFound {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusNotFound, res.StatusCode)
			}
		})
	}
}

func Test_processRequest_listParseError(t *testing.T) {
	tests := []struct {
		name string
		uri  string
	}{
		{"widgets", "/v1/widgets?\x01"},
		{"sessions", "/v1/sessions?\x01"},
		{"messages", "/v1/messages?\x01"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, _ := newTestHandler(mc)

			res, err := h.processRequest(newTestRequest(sock.RequestMethodGet, tt.uri, ""))
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res.StatusCode != http.StatusBadRequest {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusBadRequest, res.StatusCode)
			}
		})
	}
}

func Test_processRequest_handlerErrorMapping(t *testing.T) {
	denied := cerrors.PermissionDenied(outline.ServiceNameWebchatManager, "TEST", "denied")
	widgetID := uuid.FromStringOrNil(testIDWidget)

	tests := []struct {
		name string

		err error

		expectStatus int
	}{
		{"typed error", denied, http.StatusForbidden},
		{"not found", dbhandler.ErrNotFound, http.StatusNotFound},
		{"other error", errors.New("boom"), http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newTestHandler(mc)

			m.widget.EXPECT().Get(gomock.Any(), widgetID).Return(nil, tt.err)

			res, err := h.processRequest(newTestRequest(sock.RequestMethodGet, "/v1/widgets/"+testIDWidget, ""))
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res.StatusCode != tt.expectStatus {
				t.Errorf("Wrong match. expect: %d, got: %d", tt.expectStatus, res.StatusCode)
			}
		})
	}
}
