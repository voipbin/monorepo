package transferhandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	cmcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	fmflow "gitlab.com/voipbin/bin-manager/flow-manager.git/models/flow"

	"gitlab.com/voipbin/bin-manager/transfer-manager.git/pkg/dbhandler"
)

func Test_serviceInit(t *testing.T) {

	tests := []struct {
		name string

		transfererCallID uuid.UUID

		responseCall  *cmcall.Call
		responseFlow  *fmflow.Flow
		expectActions []fmaction.Action
	}{
		{
			name: "normal",

			transfererCallID: uuid.FromStringOrNil("4aa921da-dc6c-11ed-a6af-37db475b99ac"),
			responseCall: &cmcall.Call{
				ID:           uuid.FromStringOrNil("4aa921da-dc6c-11ed-a6af-37db475b99ac"),
				CustomerID:   uuid.FromStringOrNil("4af6c836-dc6c-11ed-8ede-eb826d3723eb"),
				ConfbridgeID: uuid.FromStringOrNil("4acf5788-dc6c-11ed-b296-b72b7e4a360b"),
			},
			responseFlow: &fmflow.Flow{
				ID: uuid.FromStringOrNil("9828256e-dc6c-11ed-84fa-d322b58ab276"),
			},
			expectActions: []fmaction.Action{
				{
					Type:   fmaction.TypeConfbridgeJoin,
					Option: []byte(`{"confbridge_id":"4acf5788-dc6c-11ed-b296-b72b7e4a360b"}`),
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := transferHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockReq.EXPECT().CallV1CallGet(ctx, tt.transfererCallID).Return(tt.responseCall, nil)
			mockReq.EXPECT().FlowV1FlowCreate(ctx, tt.responseCall.CustomerID, fmflow.TypeTransfer, gomock.Any(), gomock.Any(), tt.expectActions, false).Return(tt.responseFlow, nil)

			resCall, resFlow, err := h.transferInit(ctx, tt.transfererCallID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(resCall, tt.responseCall) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseCall, resCall)
			}
			if !reflect.DeepEqual(resFlow, tt.responseFlow) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseFlow, resFlow)
			}
		})
	}
}

func Test_createFlow(t *testing.T) {

	tests := []struct {
		name string

		transfererCall *cmcall.Call

		responseFlow  *fmflow.Flow
		expectActions []fmaction.Action
	}{
		{
			name: "normal",

			transfererCall: &cmcall.Call{
				ID:           uuid.FromStringOrNil("d57a241e-dbb1-11ed-af85-873dc3222a0e"),
				CustomerID:   uuid.FromStringOrNil("09c2ed00-dbb2-11ed-8948-b79e70113428"),
				ConfbridgeID: uuid.FromStringOrNil("fa9ed858-dbb0-11ed-93b5-afd5429fc93c"),
			},

			responseFlow: &fmflow.Flow{
				ID: uuid.FromStringOrNil("4072e930-dbb3-11ed-be74-33a5b14c4d0e"),
			},
			expectActions: []fmaction.Action{
				{
					Type:   fmaction.TypeConfbridgeJoin,
					Option: []byte(`{"confbridge_id":"fa9ed858-dbb0-11ed-93b5-afd5429fc93c"}`),
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := transferHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockReq.EXPECT().FlowV1FlowCreate(ctx, tt.transfererCall.CustomerID, fmflow.TypeTransfer, gomock.Any(), gomock.Any(), tt.expectActions, false).Return(tt.responseFlow, nil)

			res, err := h.createFlow(ctx, tt.transfererCall)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseFlow) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseFlow, res)
			}
		})
	}
}
