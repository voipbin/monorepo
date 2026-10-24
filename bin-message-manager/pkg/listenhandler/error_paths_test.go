package listenhandler

import (
	"errors"
	"testing"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
	"monorepo/bin-message-manager/pkg/messagehandler"
)

func newErrorPathListenHandler(mc *gomock.Controller) (*listenHandler, *messagehandler.MockMessageHandler) {
	mockMessage := messagehandler.NewMockMessageHandler(mc)

	h := &listenHandler{
		sockHandler:    sockhandler.NewMockSockHandler(mc),
		messageHandler: mockMessage,
	}

	return h, mockMessage
}

func Test_processRequest_messagesGetErrors(t *testing.T) {
	tests := []struct {
		name string

		uri  string
		data []byte

		// when set, the message handler List is expected and returns this error.
		listErr error

		expectStatus int
	}{
		{
			// matches the route regex but fails url.Parse; processRequest maps the plain error to 400.
			name:         "uri that cannot be parsed",
			uri:          "/%zz/v1/messages?x",
			expectStatus: 400,
		},
		{
			name:         "filters body is not json",
			uri:          "/v1/messages?page_size=10",
			data:         []byte(`{bad`),
			expectStatus: 400,
		},
		{
			name:         "filter value has the wrong type",
			uri:          "/v1/messages?page_size=10",
			data:         []byte(`{"customer_id":"not-a-uuid"}`),
			expectStatus: 400,
		},
		{
			name:         "business handler failure maps to 500",
			uri:          "/v1/messages?page_size=10",
			data:         []byte(`{}`),
			listErr:      errors.New("list failed"),
			expectStatus: 500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			h, mockMessage := newErrorPathListenHandler(mc)

			if tt.listErr != nil {
				mockMessage.EXPECT().List(gomock.Any(), "", uint64(10), gomock.Any()).Return(nil, tt.listErr)
			}

			res, err := h.processRequest(&sock.Request{
				URI:    tt.uri,
				Method: sock.RequestMethodGet,
				Data:   tt.data,
			})
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res == nil || res.StatusCode != tt.expectStatus {
				t.Errorf("Wrong status. expect: %d, got: %v", tt.expectStatus, res)
			}
		})
	}
}

func Test_processV1MessagesGet_invalidURI(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	h, _ := newErrorPathListenHandler(mc)

	res, err := h.processV1MessagesGet(t.Context(), &sock.Request{URI: "/%zz/v1/messages?x", Method: sock.RequestMethodGet})
	if err == nil || res != nil {
		t.Errorf("Wrong match. expect: error and nil response, got: res: %v, err: %v", res, err)
	}
}

func Test_processV1MessagesPost_errors(t *testing.T) {
	tests := []struct {
		name string

		data    []byte
		sendErr error

		expectStatus int
	}{
		{
			name:         "body is not json",
			data:         []byte(`{bad`),
			expectStatus: 400,
		},
		{
			name:         "business handler failure maps to 500",
			data:         []byte(`{"customer_id":"feef3a64-4fab-46af-a61b-6a7ce31b84a9","text":"hello"}`),
			sendErr:      errors.New("send failed"),
			expectStatus: 500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			h, mockMessage := newErrorPathListenHandler(mc)

			if tt.sendErr != nil {
				mockMessage.EXPECT().Send(gomock.Any(), uuid.Nil, uuid.FromStringOrNil("feef3a64-4fab-46af-a61b-6a7ce31b84a9"), nil, nil, "hello").Return(nil, tt.sendErr)
			}

			res, err := h.processRequest(&sock.Request{
				URI:    "/v1/messages",
				Method: sock.RequestMethodPost,
				Data:   tt.data,
			})
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res == nil || res.StatusCode != tt.expectStatus {
				t.Errorf("Wrong status. expect: %d, got: %v", tt.expectStatus, res)
			}
		})
	}
}

func Test_processV1MessagesID_shortURIAndFailures(t *testing.T) {
	id := uuid.FromStringOrNil("73071e00-a29a-11ec-a43a-079fe08ce740")

	t.Run("short uri is rejected by the id handlers", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, _ := newErrorPathListenHandler(mc)

		// exactly /v1/messages splits into 3 items. The route regex never lets this through
		// processRequest, so the handlers are called directly.
		for name, call := range map[string]func() (*sock.Response, error){
			"get": func() (*sock.Response, error) {
				return h.processV1MessagesIDGet(t.Context(), &sock.Request{URI: "/v1/messages"})
			},
			"delete": func() (*sock.Response, error) {
				return h.processV1MessagesIDDelete(t.Context(), &sock.Request{URI: "/v1/messages"})
			},
		} {
			res, err := call()
			if err != nil || res == nil || res.StatusCode != 400 {
				t.Errorf("%s: Wrong match. expect: 400, got: res: %v, err: %v", name, res, err)
			}
		}
	})

	t.Run("business handler failures map to 500", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, mockMessage := newErrorPathListenHandler(mc)

		mockMessage.EXPECT().Get(gomock.Any(), id).Return(nil, errors.New("get failed"))
		mockMessage.EXPECT().Delete(gomock.Any(), id).Return(nil, errors.New("delete failed"))

		for _, method := range []sock.RequestMethod{sock.RequestMethodGet, sock.RequestMethodDelete} {
			res, err := h.processRequest(&sock.Request{URI: "/v1/messages/" + id.String(), Method: method})
			if err != nil || res == nil || res.StatusCode != 500 {
				t.Errorf("%s: Wrong match. expect: 500, got: res: %v, err: %v", method, res, err)
			}
		}
	})
}

func Test_processV1HooksPost_errors(t *testing.T) {
	t.Run("body is not json", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, _ := newErrorPathListenHandler(mc)

		res, err := h.processRequest(&sock.Request{URI: "/v1/hooks", Method: sock.RequestMethodPost, Data: []byte(`{bad`)})
		if err != nil || res == nil || res.StatusCode != 400 {
			t.Errorf("Wrong match. expect: 400, got: res: %v, err: %v", res, err)
		}
	})

	t.Run("hook failure is logged and the caller still gets 200", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, mockMessage := newErrorPathListenHandler(mc)

		// received_data is base64 of "abc"
		mockMessage.EXPECT().Hook(gomock.Any(), "hook.example.com/v1.0/messages/telnyx", []byte("abc")).Return(errors.New("hook failed"))

		res, err := h.processRequest(&sock.Request{
			URI:    "/v1/hooks",
			Method: sock.RequestMethodPost,
			Data:   []byte(`{"received_uri":"hook.example.com/v1.0/messages/telnyx","received_data":"YWJj"}`),
		})
		if err != nil || res == nil || res.StatusCode != 200 {
			t.Errorf("Wrong match. expect: 200, got: res: %v, err: %v", res, err)
		}
	})
}
