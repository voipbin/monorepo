package listenhandler

import (
	"context"
	"net/http"
	"testing"
	"time"

	cerrors "monorepo/bin-common-handler/models/errors"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-number-manager/models/number"
	"monorepo/bin-number-manager/pkg/numberhandler"
	"monorepo/bin-number-manager/pkg/providerhandler"
)

// unencodableNumber returns a number whose time is outside the range of the JSON encoder,
// so marshaling it fails.
func unencodableNumber() *number.Number {
	tm := time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC)
	return &number.Number{TMCreate: &tm}
}

func Test_NewListenHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockNumber := numberhandler.NewMockNumberHandler(mc)
	mockProvider := providerhandler.NewMockProviderHandler(mc)

	res := NewListenHandler(mockSock, mockNumber, mockProvider)

	h, ok := res.(*listenHandler)
	if !ok {
		t.Fatalf("Wrong match. expect: *listenHandler, got: %T", res)
	}
	if h.utilHandler == nil {
		t.Errorf("Wrong match. expect: a util handler, got: nil")
	}
	if h.sockHandler != mockSock || h.numberHandler != mockNumber || h.providerHandler != mockProvider {
		t.Errorf("Wrong match. the handlers were not wired as given")
	}
}

func Test_processV1AvailableNumbersGet_invalidURI(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h := &listenHandler{
		numberHandler: numberhandler.NewMockNumberHandler(mc),
	}

	// a control character in the query makes the URI unparsable; no handler call is expected.
	res, err := h.processV1AvailableNumbersGet(context.Background(), &sock.Request{
		URI:    "/v1/available_numbers?page_size=\x7f",
		Method: sock.RequestMethodGet,
	})
	if err == nil {
		t.Errorf("Wrong match. expect: an error, got: nil")
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil response, got: %v", res)
	}
}

func Test_processV1NumbersIDFlowIDsPut_messageFlowID(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockNumber := numberhandler.NewMockNumberHandler(mc)
	h := &listenHandler{
		sockHandler:   sockhandler.NewMockSockHandler(mc),
		numberHandler: mockNumber,
	}

	id := uuid.FromStringOrNil("935190b4-7c58-11eb-8b90-f777a56fe90f")
	messageFlowID := uuid.FromStringOrNil("9394929c-7c58-11eb-8af3-13d1657955b6")

	// only the message flow is sent: the update must carry exactly that field.
	mockNumber.EXPECT().Update(gomock.Any(), id, map[number.Field]any{
		number.FieldMessageFlowID: messageFlowID,
	}).Return(&number.Number{MessageFlowID: messageFlowID}, nil)

	res, err := h.processRequest(&sock.Request{
		URI:      "/v1/numbers/935190b4-7c58-11eb-8b90-f777a56fe90f/flow_ids",
		Method:   sock.RequestMethodPut,
		DataType: "application/json",
		Data:     []byte(`{"message_flow_id":"9394929c-7c58-11eb-8af3-13d1657955b6"}`),
	})
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res.StatusCode != 200 {
		t.Errorf("Wrong match. expect: 200, got: %d", res.StatusCode)
	}
}

// Test_processRequest_marshalFailure checks that a response that cannot be encoded ends in a 500
// without a body, for every route that returns a number.
func Test_processRequest_marshalFailure(t *testing.T) {
	const id = "935190b4-7c58-11eb-8b90-f777a56fe90f"
	uid := uuid.FromStringOrNil(id)

	tests := []struct {
		name string

		request *sock.Request
		setup   func(m *numberhandler.MockNumberHandler)
	}{
		{
			name:    "create",
			request: &sock.Request{URI: "/v1/numbers", Method: sock.RequestMethodPost, Data: []byte(`{"number":"+155****0001"}`)},
			setup: func(m *numberhandler.MockNumberHandler) {
				m.EXPECT().Create(gomock.Any(), uuid.Nil, "+155****0001", uuid.Nil, uuid.Nil, "", "").Return(unencodableNumber(), nil)
			},
		},
		{
			name:    "delete",
			request: &sock.Request{URI: "/v1/numbers/" + id, Method: sock.RequestMethodDelete},
			setup: func(m *numberhandler.MockNumberHandler) {
				m.EXPECT().Delete(gomock.Any(), uid).Return(unencodableNumber(), nil)
			},
		},
		{
			name:    "get",
			request: &sock.Request{URI: "/v1/numbers/" + id, Method: sock.RequestMethodGet},
			setup: func(m *numberhandler.MockNumberHandler) {
				m.EXPECT().Get(gomock.Any(), uid).Return(unencodableNumber(), nil)
			},
		},
		{
			name:    "put",
			request: &sock.Request{URI: "/v1/numbers/" + id, Method: sock.RequestMethodPut, Data: []byte(`{"name":"renamed"}`)},
			setup: func(m *numberhandler.MockNumberHandler) {
				m.EXPECT().Update(gomock.Any(), uid, map[number.Field]any{number.FieldName: "renamed"}).Return(unencodableNumber(), nil)
			},
		},
		{
			name:    "put flow ids",
			request: &sock.Request{URI: "/v1/numbers/" + id + "/flow_ids", Method: sock.RequestMethodPut, Data: []byte(`{}`)},
			setup: func(m *numberhandler.MockNumberHandler) {
				m.EXPECT().Get(gomock.Any(), uid).Return(unencodableNumber(), nil)
			},
		},
		{
			name:    "put metadata",
			request: &sock.Request{URI: "/v1/numbers/" + id + "/metadata", Method: sock.RequestMethodPut, Data: []byte(`{"metadata":{"rtp_debug":true}}`)},
			setup: func(m *numberhandler.MockNumberHandler) {
				m.EXPECT().UpdateMetadata(gomock.Any(), uid, gomock.Any()).Return(unencodableNumber(), nil)
			},
		},
		{
			name:    "list",
			request: &sock.Request{URI: "/v1/numbers?page_size=5&page_token=2030-01-01", Method: sock.RequestMethodGet, Data: []byte(`{}`)},
			setup: func(m *numberhandler.MockNumberHandler) {
				m.EXPECT().List(gomock.Any(), uint64(5), "2030-01-01", gomock.Any()).Return([]*number.Number{unencodableNumber()}, nil)
			},
		},
		{
			name:    "renew",
			request: &sock.Request{URI: "/v1/numbers/renew", Method: sock.RequestMethodPost, Data: []byte(`{"days":3}`)},
			setup: func(m *numberhandler.MockNumberHandler) {
				m.EXPECT().RenewNumbers(gomock.Any(), 3, 0, "").Return([]*number.Number{unencodableNumber()}, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockNumber := numberhandler.NewMockNumberHandler(mc)
			h := &listenHandler{
				sockHandler:   sockhandler.NewMockSockHandler(mc),
				numberHandler: mockNumber,
			}
			tt.setup(mockNumber)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res.StatusCode != http.StatusInternalServerError {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusInternalServerError, res.StatusCode)
			}
			if len(res.Data) != 0 {
				t.Errorf("Wrong match. expect: no body, got: %s", res.Data)
			}
		})
	}
}

func Test_jsonResponse_unencodable(t *testing.T) {
	res := jsonResponse(make(chan int), &sock.Request{URI: "/v1/number_providers"})
	if res.StatusCode != http.StatusInternalServerError {
		t.Errorf("Wrong match. expect: %d, got: %d", http.StatusInternalServerError, res.StatusCode)
	}
	if len(res.Data) != 0 {
		t.Errorf("Wrong match. expect: no body, got: %s", res.Data)
	}
}

func Test_errorResponse_unencodableDetails(t *testing.T) {
	ve := &cerrors.VoipbinError{
		Status:  cerrors.StatusNotFound,
		Reason:  "NUMBER_NOT_FOUND",
		Message: "x",
		Details: []map[string]any{{"k": make(chan int)}},
	}

	res := errorResponse(ve)
	if res.StatusCode != http.StatusInternalServerError {
		t.Errorf("Wrong match. expect: %d, got: %d", http.StatusInternalServerError, res.StatusCode)
	}
	if res.DataType == cerrors.DataTypeVoipbinError {
		t.Errorf("Wrong match. expect a plain 500, got the typed error body: %s", res.Data)
	}
}
