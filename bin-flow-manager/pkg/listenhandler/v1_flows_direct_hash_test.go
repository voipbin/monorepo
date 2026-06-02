package listenhandler

import (
	"fmt"
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-flow-manager/models/flow"
	"monorepo/bin-flow-manager/pkg/dbhandler"
	"monorepo/bin-flow-manager/pkg/flowhandler"
)

func Test_processV1FlowsIDDirectHashRegeneratePost(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		responseFlow *flow.Flow
		responseErr  error

		expectedFlowID uuid.UUID
		expectedRes    *sock.Response
	}{
		{
			name: "success",
			request: &sock.Request{
				URI:      "/v1/flows/a6f4eae8-8a74-11ea-af75-3f1e61b9a236/direct-hash-regenerate",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
			},

			responseFlow: &flow.Flow{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a6f4eae8-8a74-11ea-af75-3f1e61b9a236"),
					CustomerID: uuid.FromStringOrNil("c1c1c1c1-1111-1111-1111-111111111111"),
				},
				DirectID:   uuid.FromStringOrNil("d1d1d1d1-1111-1111-1111-111111111111"),
				DirectHash: "newhash456",
			},
			responseErr: nil,

			expectedFlowID: uuid.FromStringOrNil("a6f4eae8-8a74-11ea-af75-3f1e61b9a236"),
			expectedRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"a6f4eae8-8a74-11ea-af75-3f1e61b9a236","customer_id":"c1c1c1c1-1111-1111-1111-111111111111","direct_id":"d1d1d1d1-1111-1111-1111-111111111111","direct_hash":"newhash456","on_complete_flow_id":"00000000-0000-0000-0000-000000000000","tm_create":null,"tm_update":null,"tm_delete":null}`),
			},
		},
		{
			name: "direct hash regenerate error",
			request: &sock.Request{
				URI:      "/v1/flows/b7f5fbe9-9b85-22fb-bf86-4f2f72c0b347/direct-hash-regenerate",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
			},

			responseFlow: nil,
			responseErr:  fmt.Errorf("could not regenerate direct hash"),

			expectedFlowID: uuid.FromStringOrNil("b7f5fbe9-9b85-22fb-bf86-4f2f72c0b347"),
			expectedRes: &sock.Response{
				StatusCode: 500,
			},
		},
		{
			name: "not found returns 404",
			request: &sock.Request{
				URI:      "/v1/flows/c8f6fcfa-ab96-33fc-cf97-5f3f83d1c458/direct-hash-regenerate",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
			},

			responseFlow: nil,
			responseErr:  dbhandler.ErrNotFound,

			expectedFlowID: uuid.FromStringOrNil("c8f6fcfa-ab96-33fc-cf97-5f3f83d1c458"),
			expectedRes: &sock.Response{
				StatusCode: 404,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockFlowHandler := flowhandler.NewMockFlowHandler(mc)

			h := &listenHandler{
				sockHandler: mockSock,
				flowHandler: mockFlowHandler,
			}

			mockFlowHandler.EXPECT().DirectHashRegenerate(gomock.Any(), tt.expectedFlowID).Return(tt.responseFlow, tt.responseErr)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectedRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, res)
			}
		})
	}
}
