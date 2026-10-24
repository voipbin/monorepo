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

	"monorepo/bin-webchat-manager/models/widget"
)

func testWidget(id string) *widget.Widget {
	return &widget.Widget{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil(id),
			CustomerID: uuid.FromStringOrNil(testIDCust),
		},
		Name:   "test widget",
		Status: widget.StatusActive,
	}
}

func assertWidgetBody(t *testing.T, res *sock.Response, expect *widget.Widget) {
	t.Helper()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("Wrong match. expect: %d, got: %d", http.StatusOK, res.StatusCode)
	}
	if res.DataType != "application/json" {
		t.Errorf("Wrong match. expect: application/json, got: %s", res.DataType)
	}

	var got widget.Widget
	if err := json.Unmarshal(res.Data, &got); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !reflect.DeepEqual(&got, expect) {
		t.Errorf("Wrong match.\nexpect: %+v\ngot: %+v", expect, &got)
	}
}

func Test_processV1WidgetsPost(t *testing.T) {
	theme := &widget.ThemeConfig{PrimaryColor: "#112233", HeaderTitle: "Hello"}
	flowSession := "22222222-aaaa-bbbb-cccc-000000000001"
	flowMessage := "22222222-aaaa-bbbb-cccc-000000000002"

	t.Run("fields are forwarded", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newTestHandler(mc)
		expect := testWidget(testIDWidget)

		m.widget.EXPECT().Create(
			gomock.Any(),
			uuid.FromStringOrNil(testIDCust),
			"test widget",
			uuid.FromStringOrNil(flowSession),
			uuid.FromStringOrNil(flowMessage),
			600,
			theme,
		).Return(expect, nil)

		data := `{"customer_id":"` + testIDCust + `","name":"test widget","session_flow_id":"` + flowSession +
			`","message_flow_id":"` + flowMessage + `","session_idle_timeout":600,"theme_config":{"primary_color":"#112233","header_title":"Hello"}}`
		res, err := h.processV1WidgetsPost(context.Background(), newTestRequest(sock.RequestMethodPost, "/v1/widgets", data))
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		assertWidgetBody(t, res, expect)
	})

	t.Run("invalid json", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, _ := newTestHandler(mc)

		res, err := h.processV1WidgetsPost(context.Background(), newTestRequest(sock.RequestMethodPost, "/v1/widgets", `{"name":`))
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		if res.StatusCode != http.StatusBadRequest {
			t.Errorf("Wrong match. expect: %d, got: %d", http.StatusBadRequest, res.StatusCode)
		}
	})

	t.Run("wrong field type", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, _ := newTestHandler(mc)

		res, err := h.processV1WidgetsPost(context.Background(), newTestRequest(sock.RequestMethodPost, "/v1/widgets", `{"session_idle_timeout":"soon"}`))
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		if res.StatusCode != http.StatusBadRequest {
			t.Errorf("Wrong match. expect: %d, got: %d", http.StatusBadRequest, res.StatusCode)
		}
	})

	assertErrorMapping(t,
		func(m *testMocks, err error) {
			m.widget.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, err)
		},
		func(h *listenHandler) (*sock.Response, error) {
			return h.processV1WidgetsPost(context.Background(), newTestRequest(sock.RequestMethodPost, "/v1/widgets", `{}`))
		},
	)
}

func Test_processV1WidgetsGet(t *testing.T) {
	tests := []struct {
		name string

		uri  string
		data string

		expectSize   uint64
		expectToken  string
		expectFilter map[widget.Field]any
	}{
		{
			name:         "pagination and filters",
			uri:          "/v1/widgets?page_size=10&page_token=2026-01-01T00:00:00.000000Z",
			data:         `{"customer_id":"` + testIDCust + `","name":"test widget","deleted":false}`,
			expectSize:   10,
			expectToken:  "2026-01-01T00:00:00.000000Z",
			expectFilter: map[widget.Field]any{widget.FieldCustomerID: uuid.FromStringOrNil(testIDCust), widget.FieldName: "test widget", widget.FieldDeleted: false},
		},
		{
			name:         "no body and missing page size",
			uri:          "/v1/widgets?page_token=tok",
			data:         "",
			expectSize:   0,
			expectToken:  "tok",
			expectFilter: map[widget.Field]any{},
		},
		{
			name:         "non numeric page size becomes zero",
			uri:          "/v1/widgets?page_size=abc",
			data:         `{}`,
			expectSize:   0,
			expectToken:  "",
			expectFilter: map[widget.Field]any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newTestHandler(mc)
			expect := []*widget.Widget{testWidget(testIDWidget), testWidget(testIDOther)}

			m.widget.EXPECT().List(gomock.Any(), tt.expectSize, tt.expectToken, tt.expectFilter).Return(expect, nil)

			res, err := h.processV1WidgetsGet(context.Background(), newTestRequest(sock.RequestMethodGet, tt.uri, tt.data))
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res.StatusCode != http.StatusOK {
				t.Fatalf("Wrong match. expect: %d, got: %d", http.StatusOK, res.StatusCode)
			}

			var got []*widget.Widget
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
		{"unparsable filter body", `{"customer_id":`},
		{"non uuid filter value", `{"customer_id":"not-a-uuid"}`},
	}
	for _, tt := range badRequests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, _ := newTestHandler(mc)

			res, err := h.processV1WidgetsGet(context.Background(), newTestRequest(sock.RequestMethodGet, "/v1/widgets?page_size=1", tt.data))
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

		res, err := h.processV1WidgetsGet(context.Background(), newTestRequest(sock.RequestMethodGet, "/v1/widgets?\x01", ""))
		if err == nil || res != nil {
			t.Errorf("Wrong match. expect: error and nil response, got: %v, %v", res, err)
		}
	})

	assertErrorMapping(t,
		func(m *testMocks, err error) {
			m.widget.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, err)
		},
		func(h *listenHandler) (*sock.Response, error) {
			return h.processV1WidgetsGet(context.Background(), newTestRequest(sock.RequestMethodGet, "/v1/widgets?page_size=1", `{}`))
		},
	)
}

func Test_processV1WidgetsIDGet(t *testing.T) {
	uri := "/v1/widgets/" + testIDWidget

	t.Run("success", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newTestHandler(mc)
		expect := testWidget(testIDWidget)

		m.widget.EXPECT().Get(gomock.Any(), uuid.FromStringOrNil(testIDWidget)).Return(expect, nil)

		res, err := h.processV1WidgetsIDGet(context.Background(), newTestRequest(sock.RequestMethodGet, uri, ""))
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		assertWidgetBody(t, res, expect)
	})

	assertIDGuards(t, (*listenHandler).processV1WidgetsIDGet, sock.RequestMethodGet, "widgets", "", "")
	assertErrorMapping(t,
		func(m *testMocks, err error) {
			m.widget.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil, err)
		},
		func(h *listenHandler) (*sock.Response, error) {
			return h.processV1WidgetsIDGet(context.Background(), newTestRequest(sock.RequestMethodGet, uri, ""))
		},
	)
}

func Test_processV1WidgetsIDDelete(t *testing.T) {
	uri := "/v1/widgets/" + testIDWidget

	t.Run("success", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newTestHandler(mc)
		expect := testWidget(testIDWidget)

		m.widget.EXPECT().Delete(gomock.Any(), uuid.FromStringOrNil(testIDWidget)).Return(expect, nil)

		res, err := h.processV1WidgetsIDDelete(context.Background(), newTestRequest(sock.RequestMethodDelete, uri, ""))
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		assertWidgetBody(t, res, expect)
	})

	assertIDGuards(t, (*listenHandler).processV1WidgetsIDDelete, sock.RequestMethodDelete, "widgets", "", "")
	assertErrorMapping(t,
		func(m *testMocks, err error) {
			m.widget.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil, err)
		},
		func(h *listenHandler) (*sock.Response, error) {
			return h.processV1WidgetsIDDelete(context.Background(), newTestRequest(sock.RequestMethodDelete, uri, ""))
		},
	)
}

func Test_processV1WidgetsIDPut(t *testing.T) {
	uri := "/v1/widgets/" + testIDWidget
	theme := &widget.ThemeConfig{SecondaryColor: "#445566"}
	flowSession := "22222222-aaaa-bbbb-cccc-000000000001"
	flowMessage := "22222222-aaaa-bbbb-cccc-000000000002"

	t.Run("fields are forwarded", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newTestHandler(mc)
		expect := testWidget(testIDWidget)

		m.widget.EXPECT().UpdateBasicInfo(
			gomock.Any(),
			uuid.FromStringOrNil(testIDWidget),
			"renamed",
			uuid.FromStringOrNil(flowSession),
			uuid.FromStringOrNil(flowMessage),
			900,
			theme,
		).Return(expect, nil)

		data := `{"name":"renamed","session_flow_id":"` + flowSession + `","message_flow_id":"` + flowMessage +
			`","session_idle_timeout":900,"theme_config":{"secondary_color":"#445566"}}`
		res, err := h.processV1WidgetsIDPut(context.Background(), newTestRequest(sock.RequestMethodPut, uri, data))
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		assertWidgetBody(t, res, expect)
	})

	t.Run("invalid json", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, _ := newTestHandler(mc)

		res, err := h.processV1WidgetsIDPut(context.Background(), newTestRequest(sock.RequestMethodPut, uri, `{"name":`))
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		if res.StatusCode != http.StatusBadRequest {
			t.Errorf("Wrong match. expect: %d, got: %d", http.StatusBadRequest, res.StatusCode)
		}
	})

	assertIDGuards(t, (*listenHandler).processV1WidgetsIDPut, sock.RequestMethodPut, "widgets", "", `{}`)
	assertErrorMapping(t,
		func(m *testMocks, err error) {
			m.widget.EXPECT().UpdateBasicInfo(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, err)
		},
		func(h *listenHandler) (*sock.Response, error) {
			return h.processV1WidgetsIDPut(context.Background(), newTestRequest(sock.RequestMethodPut, uri, `{}`))
		},
	)
}

func Test_processV1WidgetsIDDirectHashRegeneratePost(t *testing.T) {
	uri := "/v1/widgets/" + testIDWidget + "/direct-hash-regenerate"

	t.Run("success", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newTestHandler(mc)
		expect := testWidget(testIDWidget)
		expect.Hash = "synthetic-hash"

		m.widget.EXPECT().DirectHashRegenerate(gomock.Any(), uuid.FromStringOrNil(testIDWidget)).Return(expect, nil)

		res, err := h.processV1WidgetsIDDirectHashRegeneratePost(context.Background(), newTestRequest(sock.RequestMethodPost, uri, ""))
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		assertWidgetBody(t, res, expect)
	})

	assertIDGuards(t, (*listenHandler).processV1WidgetsIDDirectHashRegeneratePost, sock.RequestMethodPost, "widgets", "/direct-hash-regenerate", "")
	assertErrorMapping(t,
		func(m *testMocks, err error) {
			m.widget.EXPECT().DirectHashRegenerate(gomock.Any(), gomock.Any()).Return(nil, err)
		},
		func(h *listenHandler) (*sock.Response, error) {
			return h.processV1WidgetsIDDirectHashRegeneratePost(context.Background(), newTestRequest(sock.RequestMethodPost, uri, ""))
		},
	)
}
