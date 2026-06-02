package listenhandler

import (
	"reflect"
	"testing"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-flow-manager/pkg/dbhandler"
	"monorepo/bin-flow-manager/pkg/flowhandler"
)

func Test_processV1FlowsCountByCustomerGet(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		customerID    uuid.UUID
		responseCount int
		responseErr   error
		expectRes     *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/flows/count_by_customer",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"a2b3c4d5-0001-0001-0001-000000000001"}`),
			},

			customerID:    uuid.FromStringOrNil("a2b3c4d5-0001-0001-0001-000000000001"),
			responseCount: 5,
			responseErr:   nil,
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"count":5}`),
			},
		},
		{
			name: "not found returns 404",
			request: &sock.Request{
				URI:      "/v1/flows/count_by_customer",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"b3c4d5e6-0002-0002-0002-000000000002"}`),
			},

			customerID:    uuid.FromStringOrNil("b3c4d5e6-0002-0002-0002-000000000002"),
			responseCount: 0,
			responseErr:   dbhandler.ErrNotFound,
			expectRes: &sock.Response{
				StatusCode: 404,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockFlow := flowhandler.NewMockFlowHandler(mc)

			h := &listenHandler{
				sockHandler: mockSock,
				flowHandler: mockFlow,
			}

			mockFlow.EXPECT().CountByCustomerID(gomock.Any(), tt.customerID).Return(tt.responseCount, tt.responseErr)
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
