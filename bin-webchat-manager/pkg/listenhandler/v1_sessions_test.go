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

	"monorepo/bin-webchat-manager/models/session"
)

func testSession(id string) *session.Session {
	return &session.Session{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil(id),
			CustomerID: uuid.FromStringOrNil(testIDCust),
		},
		WidgetID: uuid.FromStringOrNil(testIDWidget),
		Status:   session.StatusActive,
		PageURL:  "https://example.com/page",
		Referrer: "https://example.com/ref",
	}
}

func assertSessionBody(t *testing.T, res *sock.Response, expect *session.Session) {
	t.Helper()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("Wrong match. expect: %d, got: %d", http.StatusOK, res.StatusCode)
	}
	if res.DataType != "application/json" {
		t.Errorf("Wrong match. expect: application/json, got: %s", res.DataType)
	}

	var got session.Session
	if err := json.Unmarshal(res.Data, &got); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !reflect.DeepEqual(&got, expect) {
		t.Errorf("Wrong match.\nexpect: %+v\ngot: %+v", expect, &got)
	}
}

func Test_processV1SessionsPost(t *testing.T) {
	t.Run("fields are forwarded", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newTestHandler(mc)
		expect := testSession(testIDSession)

		m.session.EXPECT().Create(
			gomock.Any(),
			uuid.FromStringOrNil(testIDSession),
			uuid.FromStringOrNil(testIDCust),
			uuid.FromStringOrNil(testIDWidget),
			"https://example.com/page",
			"https://example.com/ref",
		).Return(expect, nil)

		data := `{"id":"` + testIDSession + `","customer_id":"` + testIDCust + `","widget_id":"` + testIDWidget +
			`","page_url":"https://example.com/page","referrer":"https://example.com/ref"}`
		res, err := h.processV1SessionsPost(context.Background(), newTestRequest(sock.RequestMethodPost, "/v1/sessions", data))
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		assertSessionBody(t, res, expect)
	})

	t.Run("zero id means generate", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newTestHandler(mc)
		expect := testSession(testIDSession)

		m.session.EXPECT().Create(gomock.Any(), uuid.Nil, uuid.FromStringOrNil(testIDCust), uuid.FromStringOrNil(testIDWidget), "", "").Return(expect, nil)

		data := `{"customer_id":"` + testIDCust + `","widget_id":"` + testIDWidget + `"}`
		res, err := h.processV1SessionsPost(context.Background(), newTestRequest(sock.RequestMethodPost, "/v1/sessions", data))
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		assertSessionBody(t, res, expect)
	})

	badData := []struct {
		name string
		data string
	}{
		{"invalid json", `{"widget_id":`},
		{"malformed uuid", `{"widget_id":"not-a-uuid"}`},
	}
	for _, tt := range badData {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, _ := newTestHandler(mc)

			res, err := h.processV1SessionsPost(context.Background(), newTestRequest(sock.RequestMethodPost, "/v1/sessions", tt.data))
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
			m.session.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, err)
		},
		func(h *listenHandler) (*sock.Response, error) {
			return h.processV1SessionsPost(context.Background(), newTestRequest(sock.RequestMethodPost, "/v1/sessions", `{}`))
		},
	)
}

func Test_processV1SessionsGet(t *testing.T) {
	tests := []struct {
		name string

		uri  string
		data string

		expectSize   uint64
		expectToken  string
		expectFilter map[session.Field]any
	}{
		{
			name:         "pagination and filters",
			uri:          "/v1/sessions?page_size=20&page_token=tok",
			data:         `{"widget_id":"` + testIDWidget + `","status":"active"}`,
			expectSize:   20,
			expectToken:  "tok",
			expectFilter: map[session.Field]any{session.FieldWidgetID: uuid.FromStringOrNil(testIDWidget), session.FieldStatus: "active"},
		},
		{
			name:         "no body and no pagination",
			uri:          "/v1/sessions?x=1",
			data:         "",
			expectSize:   0,
			expectToken:  "",
			expectFilter: map[session.Field]any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newTestHandler(mc)
			expect := []*session.Session{testSession(testIDSession), testSession(testIDOther)}

			m.session.EXPECT().List(gomock.Any(), tt.expectSize, tt.expectToken, tt.expectFilter).Return(expect, nil)

			res, err := h.processV1SessionsGet(context.Background(), newTestRequest(sock.RequestMethodGet, tt.uri, tt.data))
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res.StatusCode != http.StatusOK {
				t.Fatalf("Wrong match. expect: %d, got: %d", http.StatusOK, res.StatusCode)
			}

			var got []*session.Session
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
		{"unparsable filter body", `{"widget_id":`},
		{"non uuid filter value", `{"widget_id":"not-a-uuid"}`},
	}
	for _, tt := range badRequests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, _ := newTestHandler(mc)

			res, err := h.processV1SessionsGet(context.Background(), newTestRequest(sock.RequestMethodGet, "/v1/sessions?page_size=1", tt.data))
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

		res, err := h.processV1SessionsGet(context.Background(), newTestRequest(sock.RequestMethodGet, "/v1/sessions?\x01", ""))
		if err == nil || res != nil {
			t.Errorf("Wrong match. expect: error and nil response, got: %v, %v", res, err)
		}
	})

	assertErrorMapping(t,
		func(m *testMocks, err error) {
			m.session.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, err)
		},
		func(h *listenHandler) (*sock.Response, error) {
			return h.processV1SessionsGet(context.Background(), newTestRequest(sock.RequestMethodGet, "/v1/sessions?page_size=1", `{}`))
		},
	)
}

func Test_processV1SessionsIDGet(t *testing.T) {
	uri := "/v1/sessions/" + testIDSession

	t.Run("success", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newTestHandler(mc)
		expect := testSession(testIDSession)

		m.session.EXPECT().Get(gomock.Any(), uuid.FromStringOrNil(testIDSession)).Return(expect, nil)

		res, err := h.processV1SessionsIDGet(context.Background(), newTestRequest(sock.RequestMethodGet, uri, ""))
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		assertSessionBody(t, res, expect)
	})

	assertIDGuards(t, (*listenHandler).processV1SessionsIDGet, sock.RequestMethodGet, "sessions", "", "")
	assertErrorMapping(t,
		func(m *testMocks, err error) {
			m.session.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil, err)
		},
		func(h *listenHandler) (*sock.Response, error) {
			return h.processV1SessionsIDGet(context.Background(), newTestRequest(sock.RequestMethodGet, uri, ""))
		},
	)
}

func Test_processV1SessionsIDDelete(t *testing.T) {
	uri := "/v1/sessions/" + testIDSession

	t.Run("success", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newTestHandler(mc)
		expect := testSession(testIDSession)

		m.session.EXPECT().Delete(gomock.Any(), uuid.FromStringOrNil(testIDSession)).Return(expect, nil)

		res, err := h.processV1SessionsIDDelete(context.Background(), newTestRequest(sock.RequestMethodDelete, uri, ""))
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		assertSessionBody(t, res, expect)
	})

	assertIDGuards(t, (*listenHandler).processV1SessionsIDDelete, sock.RequestMethodDelete, "sessions", "", "")
	assertErrorMapping(t,
		func(m *testMocks, err error) {
			m.session.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil, err)
		},
		func(h *listenHandler) (*sock.Response, error) {
			return h.processV1SessionsIDDelete(context.Background(), newTestRequest(sock.RequestMethodDelete, uri, ""))
		},
	)
}

func Test_processV1SessionsIDEndPost(t *testing.T) {
	uri := "/v1/sessions/" + testIDSession + "/end"

	t.Run("success", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newTestHandler(mc)
		expect := testSession(testIDSession)
		expect.Status = session.StatusEnded

		m.session.EXPECT().End(gomock.Any(), uuid.FromStringOrNil(testIDSession)).Return(expect, nil)

		res, err := h.processV1SessionsIDEndPost(context.Background(), newTestRequest(sock.RequestMethodPost, uri, ""))
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		assertSessionBody(t, res, expect)
	})

	assertIDGuards(t, (*listenHandler).processV1SessionsIDEndPost, sock.RequestMethodPost, "sessions", "/end", "")
	assertErrorMapping(t,
		func(m *testMocks, err error) {
			m.session.EXPECT().End(gomock.Any(), gomock.Any()).Return(nil, err)
		},
		func(h *listenHandler) (*sock.Response, error) {
			return h.processV1SessionsIDEndPost(context.Background(), newTestRequest(sock.RequestMethodPost, uri, ""))
		},
	)
}
