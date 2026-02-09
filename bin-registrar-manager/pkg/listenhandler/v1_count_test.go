package listenhandler

import (
	"reflect"
	"testing"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-registrar-manager/pkg/extensionhandler"
	"monorepo/bin-registrar-manager/pkg/trunkhandler"
)

func Test_processV1ExtensionsCountByCustomerGet(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		customerID    uuid.UUID
		responseCount int
		expectRes     *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/extensions/count_by_customer",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"d1e2f3a4-0001-0001-0001-000000000001"}`),
			},

			customerID:    uuid.FromStringOrNil("d1e2f3a4-0001-0001-0001-000000000001"),
			responseCount: 5,
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"count":5}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockExtension := extensionhandler.NewMockExtensionHandler(mc)

			h := &listenHandler{
				sockHandler:      mockSock,
				extensionHandler: mockExtension,
			}

			mockExtension.EXPECT().CountByCustomerID(gomock.Any(), tt.customerID).Return(tt.responseCount, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_processV1TrunksCountByCustomerGet(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		customerID    uuid.UUID
		responseCount int
		expectRes     *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/trunks/count_by_customer",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"e1f2a3b4-0001-0001-0001-000000000001"}`),
			},

			customerID:    uuid.FromStringOrNil("e1f2a3b4-0001-0001-0001-000000000001"),
			responseCount: 5,
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"count":5}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockTrunk := trunkhandler.NewMockTrunkHandler(mc)

			h := &listenHandler{
				sockHandler:  mockSock,
				trunkHandler: mockTrunk,
			}

			mockTrunk.EXPECT().CountByCustomerID(gomock.Any(), tt.customerID).Return(tt.responseCount, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
