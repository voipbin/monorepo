package listenhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/models/sock"
	fmaction "monorepo/bin-flow-manager/models/action"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-route-manager/models/providercall"
	"monorepo/bin-route-manager/pkg/providercallhandler"
)

const (
	testProviderCallID   = "d1000000-0000-0000-0000-000000000001"
	testProviderCallJSON = `{"id":"d1000000-0000-0000-0000-000000000001","customer_id":"00000000-0000-0000-0000-000000000000","provider_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","source":null,"destinations":null,"anonymous":"","call_ids":null,"groupcall_ids":null,"tm_create":null,"tm_update":null,"tm_delete":null}`
)

func newProviderCallListenHandler(mc *gomock.Controller) (*listenHandler, *providercallhandler.MockProviderCallHandler) {
	mockPC := providercallhandler.NewMockProviderCallHandler(mc)
	return &listenHandler{providerCallHandler: mockPC}, mockPC
}

func Test_v1ProviderCallsPost(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request

		customerID   uuid.UUID
		providerID   uuid.UUID
		flowID       uuid.UUID
		actions      []fmaction.Action
		source       *commonaddress.Address
		destinations []commonaddress.Address
		anonymous    string

		responseProviderCall *providercall.ProviderCall
		expectRes            *sock.Response
	}{
		{
			"all request fields are handed to the handler",
			&sock.Request{
				URI:      "/v1/providercalls",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"d2000000-0000-0000-0000-000000000001","provider_id":"d3000000-0000-0000-0000-000000000001","flow_id":"d4000000-0000-0000-0000-000000000001","actions":[{"type":"answer"}],"source":{"type":"tel","target":"+15550000001"},"destinations":[{"type":"tel","target":"+15550000002"},{"type":"tel","target":"+15550000003"}],"anonymous":"auto"}`),
			},

			uuid.FromStringOrNil("d2000000-0000-0000-0000-000000000001"),
			uuid.FromStringOrNil("d3000000-0000-0000-0000-000000000001"),
			uuid.FromStringOrNil("d4000000-0000-0000-0000-000000000001"),
			[]fmaction.Action{{Type: fmaction.TypeAnswer}},
			&commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15550000001"},
			[]commonaddress.Address{
				{Type: commonaddress.TypeTel, Target: "+15550000002"},
				{Type: commonaddress.TypeTel, Target: "+15550000003"},
			},
			"auto",

			&providercall.ProviderCall{ID: uuid.FromStringOrNil(testProviderCallID)},
			&sock.Response{StatusCode: 200, DataType: "application/json", Data: []byte(testProviderCallJSON)},
		},
		{
			"empty body reaches the handler with zero values",
			&sock.Request{
				URI:      "/v1/providercalls",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{}`),
			},

			uuid.Nil, uuid.Nil, uuid.Nil, nil, nil, nil, "",

			&providercall.ProviderCall{ID: uuid.FromStringOrNil(testProviderCallID)},
			&sock.Response{StatusCode: 200, DataType: "application/json", Data: []byte(testProviderCallJSON)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, mockPC := newProviderCallListenHandler(mc)

			mockPC.EXPECT().Create(gomock.Any(), tt.customerID, tt.providerID, tt.flowID, tt.actions, tt.source, tt.destinations, tt.anonymous).Return(tt.responseProviderCall, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_v1ProviderCallsPost_Errors(t *testing.T) {
	t.Run("invalid json is returned as an error by the handler", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, _ := newProviderCallListenHandler(mc)

		res, err := h.v1ProviderCallsPost(context.Background(), &sock.Request{URI: "/v1/providercalls", Method: sock.RequestMethodPost, Data: []byte(`{not json`)})
		if err == nil || res != nil {
			t.Errorf("Wrong match. expect: error and nil response, got: %v, %v", res, err)
		}
	})

	t.Run("invalid json becomes 400 through processRequest and the handler is not called", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, _ := newProviderCallListenHandler(mc)

		res, err := h.processRequest(&sock.Request{URI: "/v1/providercalls", Method: sock.RequestMethodPost, Data: []byte(`{not json`)})
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		if res.StatusCode != 400 {
			t.Errorf("Wrong match. expect: 400, got: %v", res.StatusCode)
		}
	})

	t.Run("handler error is propagated", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, mockPC := newProviderCallListenHandler(mc)

		mockPC.EXPECT().Create(gomock.Any(), uuid.Nil, uuid.Nil, uuid.Nil, nil, nil, nil, "").Return(nil, fmt.Errorf("create failed"))

		res, err := h.v1ProviderCallsPost(context.Background(), &sock.Request{URI: "/v1/providercalls", Method: sock.RequestMethodPost, Data: []byte(`{}`)})
		if err == nil || res != nil {
			t.Errorf("Wrong match. expect: error and nil response, got: %v, %v", res, err)
		}
	})
}

func Test_v1ProviderCallsGet(t *testing.T) {
	customerID := uuid.FromStringOrNil("d2000000-0000-0000-0000-000000000002")
	providerID := uuid.FromStringOrNil("d3000000-0000-0000-0000-000000000002")

	tests := []struct {
		name    string
		request *sock.Request

		pageToken string
		pageSize  uint64
		filters   map[providercall.Field]any

		responseProviderCalls []*providercall.ProviderCall
		expectRes             *sock.Response
	}{
		{
			"pagination and both filters",
			&sock.Request{
				URI:    "/v1/providercalls?page_token=2020-10-10T03:30:17.000000Z&page_size=7&customer_id=d2000000-0000-0000-0000-000000000002&provider_id=d3000000-0000-0000-0000-000000000002",
				Method: sock.RequestMethodGet,
			},

			"2020-10-10T03:30:17.000000Z",
			7,
			map[providercall.Field]any{
				providercall.FieldCustomerID: customerID,
				providercall.FieldProviderID: providerID,
			},

			[]*providercall.ProviderCall{{ID: uuid.FromStringOrNil(testProviderCallID)}},
			&sock.Response{StatusCode: 200, DataType: "application/json", Data: []byte("[" + testProviderCallJSON + "]")},
		},
		{
			"no query values means zero page size, empty token and no filters",
			&sock.Request{
				URI:    "/v1/providercalls?",
				Method: sock.RequestMethodGet,
			},

			"",
			0,
			map[providercall.Field]any{},

			[]*providercall.ProviderCall{},
			&sock.Response{StatusCode: 200, DataType: "application/json", Data: []byte("[]")},
		},
		{
			"non-numeric page size falls back to zero",
			&sock.Request{
				URI:    "/v1/providercalls?page_size=abc&customer_id=d2000000-0000-0000-0000-000000000002",
				Method: sock.RequestMethodGet,
			},

			"",
			0,
			map[providercall.Field]any{providercall.FieldCustomerID: customerID},

			[]*providercall.ProviderCall{},
			&sock.Response{StatusCode: 200, DataType: "application/json", Data: []byte("[]")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, mockPC := newProviderCallListenHandler(mc)

			mockPC.EXPECT().List(gomock.Any(), tt.pageToken, tt.pageSize, tt.filters).Return(tt.responseProviderCalls, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_v1ProviderCallsGet_Errors(t *testing.T) {
	tests := []struct {
		name string
		uri  string
	}{
		{"invalid customer_id filter", "/v1/providercalls?customer_id=not-a-uuid"},
		{"invalid provider_id filter", "/v1/providercalls?provider_id=not-a-uuid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			// no List expectation: an invalid filter must not reach the handler
			h, _ := newProviderCallListenHandler(mc)

			res, err := h.v1ProviderCallsGet(context.Background(), &sock.Request{URI: tt.uri, Method: sock.RequestMethodGet})
			if err == nil || res != nil {
				t.Errorf("Wrong match. expect: error and nil response, got: %v, %v", res, err)
			}
		})
	}

	t.Run("unparsable uri", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, _ := newProviderCallListenHandler(mc)

		res, err := h.v1ProviderCallsGet(context.Background(), &sock.Request{URI: "://bad", Method: sock.RequestMethodGet})
		if err == nil || res != nil {
			t.Errorf("Wrong match. expect: error and nil response, got: %v, %v", res, err)
		}
	})

	t.Run("handler error is propagated", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, mockPC := newProviderCallListenHandler(mc)

		mockPC.EXPECT().List(gomock.Any(), "", uint64(0), map[providercall.Field]any{}).Return(nil, fmt.Errorf("list failed"))

		res, err := h.v1ProviderCallsGet(context.Background(), &sock.Request{URI: "/v1/providercalls?", Method: sock.RequestMethodGet})
		if err == nil || res != nil {
			t.Errorf("Wrong match. expect: error and nil response, got: %v, %v", res, err)
		}
	})
}

func Test_v1ProviderCallsIDGet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockPC := newProviderCallListenHandler(mc)

	id := uuid.FromStringOrNil(testProviderCallID)
	mockPC.EXPECT().Get(gomock.Any(), id).Return(&providercall.ProviderCall{ID: id}, nil)

	res, err := h.processRequest(&sock.Request{URI: "/v1/providercalls/" + testProviderCallID, Method: sock.RequestMethodGet})
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	expect := &sock.Response{StatusCode: 200, DataType: "application/json", Data: []byte(testProviderCallJSON)}
	if !reflect.DeepEqual(res, expect) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", expect, res)
	}
}

func Test_v1ProviderCallsIDGet_Errors(t *testing.T) {
	id := uuid.FromStringOrNil(testProviderCallID)

	t.Run("handler error is propagated", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, mockPC := newProviderCallListenHandler(mc)

		mockPC.EXPECT().Get(gomock.Any(), id).Return(nil, fmt.Errorf("get failed"))

		res, err := h.v1ProviderCallsIDGet(context.Background(), &sock.Request{URI: "/v1/providercalls/" + testProviderCallID, Method: sock.RequestMethodGet})
		if err == nil || res != nil {
			t.Errorf("Wrong match. expect: error and nil response, got: %v, %v", res, err)
		}
	})

	t.Run("handler error through processRequest becomes 400", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, mockPC := newProviderCallListenHandler(mc)

		mockPC.EXPECT().Get(gomock.Any(), id).Return(nil, fmt.Errorf("get failed"))

		res, err := h.processRequest(&sock.Request{URI: "/v1/providercalls/" + testProviderCallID, Method: sock.RequestMethodGet})
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		if res.StatusCode != 400 {
			t.Errorf("Wrong match. expect: 400, got: %v", res.StatusCode)
		}
	})

	t.Run("unparsable uri", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, _ := newProviderCallListenHandler(mc)

		res, err := h.v1ProviderCallsIDGet(context.Background(), &sock.Request{URI: "://bad", Method: sock.RequestMethodGet})
		if err == nil || res != nil {
			t.Errorf("Wrong match. expect: error and nil response, got: %v, %v", res, err)
		}
	})
}

func Test_v1ProviderCallsIDDelete(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockPC := newProviderCallListenHandler(mc)

	id := uuid.FromStringOrNil(testProviderCallID)
	mockPC.EXPECT().Delete(gomock.Any(), id).Return(&providercall.ProviderCall{ID: id}, nil)

	res, err := h.processRequest(&sock.Request{URI: "/v1/providercalls/" + testProviderCallID, Method: sock.RequestMethodDelete})
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	expect := &sock.Response{StatusCode: 200, DataType: "application/json", Data: []byte(testProviderCallJSON)}
	if !reflect.DeepEqual(res, expect) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", expect, res)
	}
}

func Test_v1ProviderCallsIDDelete_Errors(t *testing.T) {
	id := uuid.FromStringOrNil(testProviderCallID)

	t.Run("handler error is propagated", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, mockPC := newProviderCallListenHandler(mc)

		mockPC.EXPECT().Delete(gomock.Any(), id).Return(nil, fmt.Errorf("delete failed"))

		res, err := h.v1ProviderCallsIDDelete(context.Background(), &sock.Request{URI: "/v1/providercalls/" + testProviderCallID, Method: sock.RequestMethodDelete})
		if err == nil || res != nil {
			t.Errorf("Wrong match. expect: error and nil response, got: %v, %v", res, err)
		}
	})

	t.Run("unparsable uri", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, _ := newProviderCallListenHandler(mc)

		res, err := h.v1ProviderCallsIDDelete(context.Background(), &sock.Request{URI: "://bad", Method: sock.RequestMethodDelete})
		if err == nil || res != nil {
			t.Errorf("Wrong match. expect: error and nil response, got: %v, %v", res, err)
		}
	})
}
