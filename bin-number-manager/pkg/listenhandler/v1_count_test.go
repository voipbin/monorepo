package listenhandler

import (
	"fmt"
	"net/http"
	"reflect"
	"testing"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-number-manager/pkg/numberhandler"
)

func TestProcessV1NumbersCountVirtualByCustomerGet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockNumber := numberhandler.NewMockNumberHandler(mc)

	h := &listenHandler{
		sockHandler:   mockSock,
		numberHandler: mockNumber,
	}

	tests := []struct {
		name       string
		customerID uuid.UUID
		count      int

		request  *sock.Request
		response *sock.Response
	}{
		{
			"normal",
			uuid.FromStringOrNil("a1b2c3d4-e5f6-11ee-aaaa-000000000001"),
			5,
			&sock.Request{
				URI:      "/v1/numbers/count_virtual_by_customer",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"a1b2c3d4-e5f6-11ee-aaaa-000000000001"}`),
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"count":5}`),
			},
		},
		{
			"zero count",
			uuid.FromStringOrNil("a1b2c3d4-e5f6-11ee-aaaa-000000000002"),
			0,
			&sock.Request{
				URI:      "/v1/numbers/count_virtual_by_customer",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"a1b2c3d4-e5f6-11ee-aaaa-000000000002"}`),
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"count":0}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockNumber.EXPECT().CountVirtualByCustomerID(gomock.Any(), tt.customerID).Return(tt.count, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.response, res) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.response, res)
			}

		})
	}
}

func TestProcessV1NumbersCountVirtualByCustomerGet_errorMapping(t *testing.T) {
	customerID := uuid.FromStringOrNil("a1b2c3d4-e5f6-11ee-aaaa-000000000099")
	req := &sock.Request{
		URI:      "/v1/numbers/count_virtual_by_customer",
		Method:   sock.RequestMethodGet,
		DataType: "application/json",
		Data:     []byte(`{"customer_id":"a1b2c3d4-e5f6-11ee-aaaa-000000000099"}`),
	}

	tests := []struct {
		name           string
		handlerErr     error
		expectStatus   int
		expectDataType string
	}{
		{
			name:           "typed NotFound maps to 404",
			handlerErr:     cerrors.NotFound(commonoutline.ServiceNameNumberManager, "NUMBER_NOT_FOUND", "customer not found"),
			expectStatus:   http.StatusNotFound,
			expectDataType: cerrors.DataTypeVoipbinError,
		},
		{
			name:         "generic error maps to 500",
			handlerErr:   fmt.Errorf("db unavailable"),
			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockNumber := numberhandler.NewMockNumberHandler(mc)

			h := &listenHandler{
				sockHandler:   mockSock,
				numberHandler: mockNumber,
			}

			mockNumber.EXPECT().CountVirtualByCustomerID(gomock.Any(), customerID).Return(0, tt.handlerErr)

			res, err := h.processRequest(req)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if res.StatusCode != tt.expectStatus {
				t.Errorf("wrong status: expect %d, got %d", tt.expectStatus, res.StatusCode)
			}
			if tt.expectDataType != "" && res.DataType != tt.expectDataType {
				t.Errorf("wrong data type: expect %s, got %s", tt.expectDataType, res.DataType)
			}
		})
	}
}
