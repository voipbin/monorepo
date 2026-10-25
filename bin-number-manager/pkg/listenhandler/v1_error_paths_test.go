package listenhandler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-number-manager/models/number"
	"monorepo/bin-number-manager/pkg/dbhandler"
	"monorepo/bin-number-manager/pkg/numberhandler"
)

const testNumberID = "9a6020ea-79ed-11eb-a0e7-8bcfb82a6f3f"

// Test_processRequest_handlerErrorMapping checks that an error returned by the business handler
// is mapped to the wire status for every route that goes through errorResponse.
func Test_processRequest_handlerErrorMapping(t *testing.T) {
	id := uuid.FromStringOrNil(testNumberID)

	type endpoint struct {
		name  string
		req   *sock.Request
		setup func(m *numberhandler.MockNumberHandler, err error)
	}

	endpoints := []endpoint{
		{
			"post normal number",
			&sock.Request{URI: "/v1/numbers", Method: sock.RequestMethodPost, Data: []byte(`{"number":"+155****0001"}`)},
			func(m *numberhandler.MockNumberHandler, err error) {
				m.EXPECT().Create(gomock.Any(), uuid.Nil, "+155****0001", uuid.Nil, uuid.Nil, "", "").Return(nil, err)
			},
		},
		{
			"post virtual number",
			&sock.Request{URI: "/v1/numbers", Method: sock.RequestMethodPost, Data: []byte(`{"number":"+155****0002","type":"virtual"}`)},
			func(m *numberhandler.MockNumberHandler, err error) {
				m.EXPECT().CreateVirtual(gomock.Any(), uuid.Nil, "+155****0002", uuid.Nil, uuid.Nil, "", "", false).Return(nil, err)
			},
		},
		{
			"delete",
			&sock.Request{URI: "/v1/numbers/" + testNumberID, Method: sock.RequestMethodDelete},
			func(m *numberhandler.MockNumberHandler, err error) {
				m.EXPECT().Delete(gomock.Any(), id).Return(nil, err)
			},
		},
		{
			"get",
			&sock.Request{URI: "/v1/numbers/" + testNumberID, Method: sock.RequestMethodGet},
			func(m *numberhandler.MockNumberHandler, err error) {
				m.EXPECT().Get(gomock.Any(), id).Return(nil, err)
			},
		},
		{
			"put with fields",
			&sock.Request{URI: "/v1/numbers/" + testNumberID, Method: sock.RequestMethodPut, Data: []byte(`{"name":"renamed"}`)},
			func(m *numberhandler.MockNumberHandler, err error) {
				m.EXPECT().Update(gomock.Any(), id, map[number.Field]any{number.FieldName: "renamed"}).Return(nil, err)
			},
		},
		{
			"put flow ids with fields",
			&sock.Request{URI: "/v1/numbers/" + testNumberID + "/flow_ids", Method: sock.RequestMethodPut, Data: []byte(`{"call_flow_id":"7051e796-8821-11ec-9b7d-d322b1036e7d"}`)},
			func(m *numberhandler.MockNumberHandler, err error) {
				m.EXPECT().Update(gomock.Any(), id, map[number.Field]any{
					number.FieldCallFlowID: uuid.FromStringOrNil("7051e796-8821-11ec-9b7d-d322b1036e7d"),
				}).Return(nil, err)
			},
		},
		{
			"renew",
			&sock.Request{URI: "/v1/numbers/renew", Method: sock.RequestMethodPost, Data: []byte(`{"days":3}`)},
			func(m *numberhandler.MockNumberHandler, err error) {
				m.EXPECT().RenewNumbers(gomock.Any(), 3, 0, "").Return(nil, err)
			},
		},
		{
			"list",
			&sock.Request{URI: "/v1/numbers?page_size=5&page_token=2030-01-01", Method: sock.RequestMethodGet, Data: []byte(`{}`)},
			func(m *numberhandler.MockNumberHandler, err error) {
				m.EXPECT().List(gomock.Any(), uint64(5), "2030-01-01", gomock.Any()).Return(nil, err)
			},
		},
		{
			"count virtual by customer",
			&sock.Request{URI: "/v1/numbers/count_virtual_by_customer", Method: sock.RequestMethodGet, Data: []byte(`{"customer_id":"72f3b054-7ff4-11ec-9af9-0b8c5dbee258"}`)},
			func(m *numberhandler.MockNumberHandler, err error) {
				m.EXPECT().CountVirtualByCustomerID(gomock.Any(), uuid.FromStringOrNil("72f3b054-7ff4-11ec-9af9-0b8c5dbee258")).Return(0, err)
			},
		},
		{
			"available virtual numbers",
			&sock.Request{URI: "/v1/available_numbers", Method: sock.RequestMethodGet, Data: []byte(`{"type":"virtual"}`)},
			func(m *numberhandler.MockNumberHandler, err error) {
				// the page size defaults to 10 when it is not given.
				m.EXPECT().GetAvailableVirtualNumbers(gomock.Any(), uint(10)).Return(nil, err)
			},
		},
	}

	errCases := []struct {
		name           string
		err            error
		expectStatus   int
		expectDataType string
	}{
		{
			"typed not found",
			cerrors.NotFound(commonoutline.ServiceNameNumberManager, "NUMBER_NOT_FOUND", "not found"),
			http.StatusNotFound,
			cerrors.DataTypeVoipbinError,
		},
		{
			"wrapped legacy not found",
			fmt.Errorf("wrapped: %w", dbhandler.ErrNotFound),
			http.StatusNotFound,
			"",
		},
		{
			"untyped error",
			errors.New("unexpected failure"),
			http.StatusInternalServerError,
			"",
		},
	}

	for _, ep := range endpoints {
		for _, ec := range errCases {
			t.Run(ep.name+"/"+ec.name, func(t *testing.T) {
				mc := gomock.NewController(t)
				defer mc.Finish()

				mockNumber := numberhandler.NewMockNumberHandler(mc)
				h := &listenHandler{
					sockHandler:   sockhandler.NewMockSockHandler(mc),
					numberHandler: mockNumber,
				}
				ep.setup(mockNumber, ec.err)

				res, err := h.processRequest(ep.req)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if res.StatusCode != ec.expectStatus {
					t.Errorf("wrong status. expect: %d, got: %d", ec.expectStatus, res.StatusCode)
				}
				if res.DataType != ec.expectDataType {
					t.Errorf("wrong data type. expect: %q, got: %q", ec.expectDataType, res.DataType)
				}
			})
		}
	}
}

// Test_processV1NumbersID_shortURI calls the ID handlers directly with a URI that has too few
// segments. Through processRequest such a URI is routed to another handler, so the guard is only
// reachable this way. The number handler must not be called.
func Test_processV1NumbersID_shortURI(t *testing.T) {
	tests := []struct {
		name string
		fn   func(h *listenHandler, m *sock.Request) (*sock.Response, error)
		data string
	}{
		{"delete", func(h *listenHandler, m *sock.Request) (*sock.Response, error) {
			return h.processV1NumbersIDDelete(context.Background(), m)
		}, ""},
		{"get", func(h *listenHandler, m *sock.Request) (*sock.Response, error) {
			return h.processV1NumbersIDGet(context.Background(), m)
		}, ""},
		{"put", func(h *listenHandler, m *sock.Request) (*sock.Response, error) {
			return h.processV1NumbersIDPut(context.Background(), m)
		}, `{"name":"x"}`},
		{"put flow ids", func(h *listenHandler, m *sock.Request) (*sock.Response, error) {
			return h.processV1NumbersIDFlowIDsPut(context.Background(), m)
		}, `{}`},
		{"put metadata", func(h *listenHandler, m *sock.Request) (*sock.Response, error) {
			return h.processV1NumbersIDMetadataPut(context.Background(), m)
		}, `{}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h := &listenHandler{
				sockHandler:   sockhandler.NewMockSockHandler(mc),
				numberHandler: numberhandler.NewMockNumberHandler(mc),
			}

			res, err := tt.fn(h, &sock.Request{URI: "/v1/numbers", Data: []byte(tt.data)})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res.StatusCode != http.StatusBadRequest {
				t.Errorf("wrong status. expect: 400, got: %d", res.StatusCode)
			}
		})
	}
}

// Test_processV1NumbersIDPut_noFields checks that a PUT without any updatable field returns the
// current number instead of calling Update with an empty field map.
func Test_processV1NumbersIDPut_noFields(t *testing.T) {
	id := uuid.FromStringOrNil(testNumberID)

	tests := []struct {
		name string
		uri  string
	}{
		{"put", "/v1/numbers/" + testNumberID},
		{"put flow ids", "/v1/numbers/" + testNumberID + "/flow_ids"},
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

			mockNumber.EXPECT().Get(gomock.Any(), id).Return(&number.Number{Number: "+155****0003"}, nil)

			res, err := h.processRequest(&sock.Request{URI: tt.uri, Method: sock.RequestMethodPut, Data: []byte(`{}`)})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res.StatusCode != http.StatusOK {
				t.Fatalf("wrong status. expect: 200, got: %d", res.StatusCode)
			}
			if !strings.Contains(string(res.Data), `"number":"+155****0003"`) {
				t.Errorf("response does not carry the current number. got: %s", res.Data)
			}
		})
	}
}

// Test_processRequest_rejectedInput covers requests that never reach the number handler.
func Test_processRequest_rejectedInput(t *testing.T) {
	tests := []struct {
		name         string
		req          *sock.Request
		expectStatus int
	}{
		{
			"unknown uri",
			&sock.Request{URI: "/v1/unknown", Method: sock.RequestMethodGet},
			http.StatusNotFound,
		},
		{
			"known uri with an unsupported method",
			&sock.Request{URI: "/v1/numbers/" + testNumberID, Method: sock.RequestMethodPost},
			http.StatusNotFound,
		},
		{
			"list with invalid filter json",
			&sock.Request{URI: "/v1/numbers?page_size=5", Method: sock.RequestMethodGet, Data: []byte(`{invalid`)},
			http.StatusBadRequest,
		},
		{
			"list with a malformed uuid filter",
			&sock.Request{URI: "/v1/numbers?page_size=5", Method: sock.RequestMethodGet, Data: []byte(`{"customer_id":"not-a-uuid"}`)},
			http.StatusBadRequest,
		},
		{
			"list with an unparsable uri",
			&sock.Request{URI: "/v1/numbers?page_size=5\x7f", Method: sock.RequestMethodGet},
			http.StatusBadRequest,
		},
		{
			"available numbers with invalid filter json",
			&sock.Request{URI: "/v1/available_numbers?page_size=5", Method: sock.RequestMethodGet, Data: []byte(`{invalid`)},
			http.StatusBadRequest,
		},
		{
			"count with invalid json",
			&sock.Request{URI: "/v1/numbers/count_virtual_by_customer", Method: sock.RequestMethodGet, Data: []byte(`{invalid`)},
			http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			// the strict mock fails the test on any call to the number handler.
			h := &listenHandler{
				sockHandler:   sockhandler.NewMockSockHandler(mc),
				numberHandler: numberhandler.NewMockNumberHandler(mc),
			}

			res, err := h.processRequest(tt.req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res.StatusCode != tt.expectStatus {
				t.Errorf("wrong status. expect: %d, got: %d", tt.expectStatus, res.StatusCode)
			}
		})
	}
}

func Test_Run(t *testing.T) {
	const queue = "bin-manager.number-manager.request.test"

	t.Run("queue create failure", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockSock := sockhandler.NewMockSockHandler(mc)
		cause := errors.New("queue declare failed")
		mockSock.EXPECT().QueueCreate(queue, "normal").Return(cause)

		h := &listenHandler{sockHandler: mockSock}

		err := h.Run(queue, "")
		if !errors.Is(err, cause) {
			t.Fatalf("expected the queue error to be wrapped. got: %v", err)
		}
	})

	t.Run("starts the consumer", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockSock := sockhandler.NewMockSockHandler(mc)
		started := make(chan struct{})
		mockSock.EXPECT().QueueCreate(queue, "normal").Return(nil)
		// the consumer returns an error, which Run only logs.
		mockSock.EXPECT().ConsumeRPC(gomock.Any(), queue, constCosumerName, false, false, false, 10, gomock.Any()).
			DoAndReturn(func(_ any, _ string, _ string, _ bool, _ bool, _ bool, _ int, _ any) error {
				close(started)
				return errors.New("consume stopped")
			})

		h := &listenHandler{sockHandler: mockSock}

		if err := h.Run(queue, ""); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		select {
		case <-started:
		case <-time.After(5 * time.Second):
			t.Fatal("the consumer was not started")
		}
	})
}
