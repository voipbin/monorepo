package listenhandler

import (
	"context"
	"net/http"
	"testing"
	"time"

	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-webchat-manager/models/message"
	"monorepo/bin-webchat-manager/models/session"
	"monorepo/bin-webchat-manager/models/widget"
)

// The encoding of a timestamp outside the year range 0 to 9999 fails, which is
// the way to reach the response encoding error of every handler without
// touching the product code.
func Test_responseEncodingFailure(t *testing.T) {
	validTime := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	invalidTime := time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC)

	idWidget := uuid.FromStringOrNil(testIDWidget)
	idSession := uuid.FromStringOrNil(testIDSession)
	idMessage := uuid.FromStringOrNil(testIDMessage)
	idCust := uuid.FromStringOrNil(testIDCust)

	widgetAt := func(tm time.Time) *widget.Widget {
		w := testWidget(testIDWidget)
		w.TMCreate = &tm
		return w
	}
	sessionAt := func(tm time.Time) *session.Session {
		s := testSession(testIDSession)
		s.TMCreate = &tm
		return s
	}
	messageAt := func(tm time.Time) *message.Message {
		m := testMessage(testIDMessage)
		m.TMCreate = &tm
		return m
	}

	type testCase struct {
		name   string
		expect func(m *testMocks, tm time.Time)
		call   func(h *listenHandler) (*sock.Response, error)
	}

	tests := []testCase{
		{
			name: "widgets post",
			expect: func(m *testMocks, tm time.Time) {
				m.widget.EXPECT().Create(gomock.Any(), idCust, "synthetic", uuid.Nil, uuid.Nil, 0, gomock.Any()).Return(widgetAt(tm), nil)
			},
			call: func(h *listenHandler) (*sock.Response, error) {
				data := `{"customer_id":"` + testIDCust + `","name":"synthetic"}`
				return h.processV1WidgetsPost(context.Background(), newTestRequest(sock.RequestMethodPost, "/v1/widgets", data))
			},
		},
		{
			name: "widgets get",
			expect: func(m *testMocks, tm time.Time) {
				m.widget.EXPECT().List(gomock.Any(), uint64(10), "", gomock.Any()).Return([]*widget.Widget{widgetAt(tm)}, nil)
			},
			call: func(h *listenHandler) (*sock.Response, error) {
				return h.processV1WidgetsGet(context.Background(), newTestRequest(sock.RequestMethodGet, "/v1/widgets?page_size=10", `{}`))
			},
		},
		{
			name: "widgets id get",
			expect: func(m *testMocks, tm time.Time) {
				m.widget.EXPECT().Get(gomock.Any(), idWidget).Return(widgetAt(tm), nil)
			},
			call: func(h *listenHandler) (*sock.Response, error) {
				return h.processV1WidgetsIDGet(context.Background(), newTestRequest(sock.RequestMethodGet, "/v1/widgets/"+testIDWidget, ""))
			},
		},
		{
			name: "widgets id delete",
			expect: func(m *testMocks, tm time.Time) {
				m.widget.EXPECT().Delete(gomock.Any(), idWidget).Return(widgetAt(tm), nil)
			},
			call: func(h *listenHandler) (*sock.Response, error) {
				return h.processV1WidgetsIDDelete(context.Background(), newTestRequest(sock.RequestMethodDelete, "/v1/widgets/"+testIDWidget, ""))
			},
		},
		{
			name: "widgets id put",
			expect: func(m *testMocks, tm time.Time) {
				m.widget.EXPECT().UpdateBasicInfo(gomock.Any(), idWidget, "renamed", uuid.Nil, uuid.Nil, 0, gomock.Any()).Return(widgetAt(tm), nil)
			},
			call: func(h *listenHandler) (*sock.Response, error) {
				return h.processV1WidgetsIDPut(context.Background(), newTestRequest(sock.RequestMethodPut, "/v1/widgets/"+testIDWidget, `{"name":"renamed"}`))
			},
		},
		{
			name: "widgets id direct hash regenerate post",
			expect: func(m *testMocks, tm time.Time) {
				m.widget.EXPECT().DirectHashRegenerate(gomock.Any(), idWidget).Return(widgetAt(tm), nil)
			},
			call: func(h *listenHandler) (*sock.Response, error) {
				return h.processV1WidgetsIDDirectHashRegeneratePost(context.Background(), newTestRequest(sock.RequestMethodPost, "/v1/widgets/"+testIDWidget+"/direct-hash-regenerate", ""))
			},
		},
		{
			name: "sessions post",
			expect: func(m *testMocks, tm time.Time) {
				m.session.EXPECT().Create(gomock.Any(), uuid.Nil, idCust, idWidget, "", "").Return(sessionAt(tm), nil)
			},
			call: func(h *listenHandler) (*sock.Response, error) {
				data := `{"customer_id":"` + testIDCust + `","widget_id":"` + testIDWidget + `"}`
				return h.processV1SessionsPost(context.Background(), newTestRequest(sock.RequestMethodPost, "/v1/sessions", data))
			},
		},
		{
			name: "sessions get",
			expect: func(m *testMocks, tm time.Time) {
				m.session.EXPECT().List(gomock.Any(), uint64(10), "", gomock.Any()).Return([]*session.Session{sessionAt(tm)}, nil)
			},
			call: func(h *listenHandler) (*sock.Response, error) {
				return h.processV1SessionsGet(context.Background(), newTestRequest(sock.RequestMethodGet, "/v1/sessions?page_size=10", `{}`))
			},
		},
		{
			name: "sessions id get",
			expect: func(m *testMocks, tm time.Time) {
				m.session.EXPECT().Get(gomock.Any(), idSession).Return(sessionAt(tm), nil)
			},
			call: func(h *listenHandler) (*sock.Response, error) {
				return h.processV1SessionsIDGet(context.Background(), newTestRequest(sock.RequestMethodGet, "/v1/sessions/"+testIDSession, ""))
			},
		},
		{
			name: "sessions id delete",
			expect: func(m *testMocks, tm time.Time) {
				m.session.EXPECT().Delete(gomock.Any(), idSession).Return(sessionAt(tm), nil)
			},
			call: func(h *listenHandler) (*sock.Response, error) {
				return h.processV1SessionsIDDelete(context.Background(), newTestRequest(sock.RequestMethodDelete, "/v1/sessions/"+testIDSession, ""))
			},
		},
		{
			name: "sessions id end post",
			expect: func(m *testMocks, tm time.Time) {
				m.session.EXPECT().End(gomock.Any(), idSession).Return(sessionAt(tm), nil)
			},
			call: func(h *listenHandler) (*sock.Response, error) {
				return h.processV1SessionsIDEndPost(context.Background(), newTestRequest(sock.RequestMethodPost, "/v1/sessions/"+testIDSession+"/end", ""))
			},
		},
		{
			name: "messages post",
			expect: func(m *testMocks, tm time.Time) {
				m.message.EXPECT().Create(gomock.Any(), idCust, idSession, message.DirectionInbound, uuid.Nil, "synthetic text").Return(messageAt(tm), nil)
			},
			call: func(h *listenHandler) (*sock.Response, error) {
				data := `{"customer_id":"` + testIDCust + `","session_id":"` + testIDSession + `","direction":"inbound","text":"synthetic text"}`
				return h.processV1MessagesPost(context.Background(), newTestRequest(sock.RequestMethodPost, "/v1/messages", data))
			},
		},
		{
			name: "messages get",
			expect: func(m *testMocks, tm time.Time) {
				m.message.EXPECT().List(gomock.Any(), uint64(10), "", gomock.Any()).Return([]*message.Message{messageAt(tm)}, nil)
			},
			call: func(h *listenHandler) (*sock.Response, error) {
				return h.processV1MessagesGet(context.Background(), newTestRequest(sock.RequestMethodGet, "/v1/messages?page_size=10", `{}`))
			},
		},
		{
			name: "messages id get",
			expect: func(m *testMocks, tm time.Time) {
				m.message.EXPECT().Get(gomock.Any(), idMessage).Return(messageAt(tm), nil)
			},
			call: func(h *listenHandler) (*sock.Response, error) {
				return h.processV1MessagesIDGet(context.Background(), newTestRequest(sock.RequestMethodGet, "/v1/messages/"+testIDMessage, ""))
			},
		},
		{
			name: "messages id delete",
			expect: func(m *testMocks, tm time.Time) {
				m.message.EXPECT().Delete(gomock.Any(), idMessage).Return(messageAt(tm), nil)
			},
			call: func(h *listenHandler) (*sock.Response, error) {
				return h.processV1MessagesIDDelete(context.Background(), newTestRequest(sock.RequestMethodDelete, "/v1/messages/"+testIDMessage, ""))
			},
		},
	}

	for _, tt := range tests {
		// the same call with a valid time succeeds, so the 500 below comes from the encoding
		t.Run(tt.name+" valid time", func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newTestHandler(mc)
			tt.expect(m, validTime)

			res, err := tt.call(h)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res.StatusCode != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, res.StatusCode)
			}
		})

		t.Run(tt.name+" invalid time", func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newTestHandler(mc)
			tt.expect(m, invalidTime)

			res, err := tt.call(h)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res.StatusCode != http.StatusInternalServerError {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusInternalServerError, res.StatusCode)
			}
			if len(res.Data) != 0 {
				t.Errorf("Wrong match. expect: empty body, got: %s", res.Data)
			}
		})
	}
}
