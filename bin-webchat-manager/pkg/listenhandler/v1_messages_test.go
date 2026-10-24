package listenhandler

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-webchat-manager/models/message"
)

func testMessage(id string) *message.Message {
	return &message.Message{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil(id),
			CustomerID: uuid.FromStringOrNil(testIDCust),
		},
		WidgetID:  uuid.FromStringOrNil(testIDWidget),
		SessionID: uuid.FromStringOrNil(testIDSession),
		Direction: message.DirectionInbound,
		Status:    message.StatusSent,
		Text:      "synthetic text",
	}
}

func assertMessageBody(t *testing.T, res *sock.Response, expect *message.Message) {
	t.Helper()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("Wrong match. expect: %d, got: %d", http.StatusOK, res.StatusCode)
	}
	if res.DataType != "application/json" {
		t.Errorf("Wrong match. expect: application/json, got: %s", res.DataType)
	}

	var got message.Message
	if err := json.Unmarshal(res.Data, &got); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !reflect.DeepEqual(&got, expect) {
		t.Errorf("Wrong match.\nexpect: %+v\ngot: %+v", expect, &got)
	}
}

func Test_processV1MessagesPost(t *testing.T) {
	t.Run("fields are forwarded", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newTestHandler(mc)
		expect := testMessage(testIDMessage)

		m.message.EXPECT().Create(
			gomock.Any(),
			uuid.FromStringOrNil(testIDCust),
			uuid.FromStringOrNil(testIDSession),
			message.DirectionOutbound,
			uuid.FromStringOrNil(testIDOther),
			"synthetic text",
		).Return(expect, nil)

		data := `{"customer_id":"` + testIDCust + `","session_id":"` + testIDSession + `","direction":"outbound","sender_id":"` +
			testIDOther + `","text":"synthetic text"}`
		res, err := h.processV1MessagesPost(context.Background(), newTestRequest(sock.RequestMethodPost, "/v1/messages", data))
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		assertMessageBody(t, res, expect)
	})

	badData := []struct {
		name string
		data string
	}{
		{"invalid json", `{"text":`},
		{"malformed uuid", `{"session_id":"not-a-uuid"}`},
	}
	for _, tt := range badData {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, _ := newTestHandler(mc)

			res, err := h.processV1MessagesPost(context.Background(), newTestRequest(sock.RequestMethodPost, "/v1/messages", tt.data))
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res.StatusCode != http.StatusBadRequest {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusBadRequest, res.StatusCode)
			}
		})
	}

	assertErrorMapping(t,
		func(m *testMocks, err error) {
			m.message.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, err)
		},
		func(h *listenHandler) (*sock.Response, error) {
			return h.processV1MessagesPost(context.Background(), newTestRequest(sock.RequestMethodPost, "/v1/messages", `{}`))
		},
	)
}

func Test_processV1MessagesGet(t *testing.T) {
	tests := []struct {
		name string

		uri  string
		data string

		expectSize   uint64
		expectToken  string
		expectFilter map[message.Field]any
	}{
		{
			name:        "pagination and filters",
			uri:         "/v1/messages?page_size=50&page_token=tok",
			data:        `{"session_id":"` + testIDSession + `","direction":"inbound","deleted":false}`,
			expectSize:  50,
			expectToken: "tok",
			expectFilter: map[message.Field]any{
				message.FieldSessionID: uuid.FromStringOrNil(testIDSession),
				message.FieldDirection: "inbound",
				message.FieldDeleted:   false,
			},
		},
		{
			name:         "no body and no pagination",
			uri:          "/v1/messages?x=1",
			data:         "",
			expectSize:   0,
			expectToken:  "",
			expectFilter: map[message.Field]any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newTestHandler(mc)
			expect := []*message.Message{testMessage(testIDMessage), testMessage(testIDOther)}

			m.message.EXPECT().List(gomock.Any(), tt.expectSize, tt.expectToken, tt.expectFilter).Return(expect, nil)

			res, err := h.processV1MessagesGet(context.Background(), newTestRequest(sock.RequestMethodGet, tt.uri, tt.data))
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res.StatusCode != http.StatusOK {
				t.Fatalf("Wrong match. expect: %d, got: %d", http.StatusOK, res.StatusCode)
			}

			var got []*message.Message
			if err := json.Unmarshal(res.Data, &got); err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if !reflect.DeepEqual(got, expect) {
				t.Errorf("Wrong match.\nexpect: %+v\ngot: %+v", expect, got)
			}
		})
	}

	badRequests := []struct {
		name string
		data string
	}{
		{"unparsable filter body", `{"session_id":`},
		{"non uuid filter value", `{"session_id":"not-a-uuid"}`},
	}
	for _, tt := range badRequests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, _ := newTestHandler(mc)

			res, err := h.processV1MessagesGet(context.Background(), newTestRequest(sock.RequestMethodGet, "/v1/messages?page_size=1", tt.data))
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res.StatusCode != http.StatusBadRequest {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusBadRequest, res.StatusCode)
			}
		})
	}

	t.Run("uri parse error returns the error", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, _ := newTestHandler(mc)

		res, err := h.processV1MessagesGet(context.Background(), newTestRequest(sock.RequestMethodGet, "/v1/messages?\x01", ""))
		if err == nil || res != nil {
			t.Errorf("Wrong match. expect: error and nil response, got: %v, %v", res, err)
		}
	})

	assertErrorMapping(t,
		func(m *testMocks, err error) {
			m.message.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, err)
		},
		func(h *listenHandler) (*sock.Response, error) {
			return h.processV1MessagesGet(context.Background(), newTestRequest(sock.RequestMethodGet, "/v1/messages?page_size=1", `{}`))
		},
	)
}

func Test_processV1MessagesIDGet(t *testing.T) {
	uri := "/v1/messages/" + testIDMessage

	t.Run("success", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newTestHandler(mc)
		expect := testMessage(testIDMessage)

		m.message.EXPECT().Get(gomock.Any(), uuid.FromStringOrNil(testIDMessage)).Return(expect, nil)

		res, err := h.processV1MessagesIDGet(context.Background(), newTestRequest(sock.RequestMethodGet, uri, ""))
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		assertMessageBody(t, res, expect)
	})

	assertIDGuards(t, (*listenHandler).processV1MessagesIDGet, sock.RequestMethodGet, "messages", "", "")
	assertErrorMapping(t,
		func(m *testMocks, err error) {
			m.message.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil, err)
		},
		func(h *listenHandler) (*sock.Response, error) {
			return h.processV1MessagesIDGet(context.Background(), newTestRequest(sock.RequestMethodGet, uri, ""))
		},
	)
}

func Test_processV1MessagesIDDelete(t *testing.T) {
	uri := "/v1/messages/" + testIDMessage

	t.Run("success", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newTestHandler(mc)
		expect := testMessage(testIDMessage)

		m.message.EXPECT().Delete(gomock.Any(), uuid.FromStringOrNil(testIDMessage)).Return(expect, nil)

		res, err := h.processV1MessagesIDDelete(context.Background(), newTestRequest(sock.RequestMethodDelete, uri, ""))
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		assertMessageBody(t, res, expect)
	})

	assertIDGuards(t, (*listenHandler).processV1MessagesIDDelete, sock.RequestMethodDelete, "messages", "", "")
	assertErrorMapping(t,
		func(m *testMocks, err error) {
			m.message.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil, err)
		},
		func(h *listenHandler) (*sock.Response, error) {
			return h.processV1MessagesIDDelete(context.Background(), newTestRequest(sock.RequestMethodDelete, uri, ""))
		},
	)
}
