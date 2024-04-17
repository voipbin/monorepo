package activeflowhandler

import (
	"context"
	"reflect"
	"testing"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"monorepo/bin-flow-manager/models/action"
	"monorepo/bin-flow-manager/models/flow"
	"monorepo/bin-flow-manager/pkg/actionhandler"
	"monorepo/bin-flow-manager/pkg/dbhandler"
)

func TestGenerateFlowForAgentCall(t *testing.T) {

	tests := []struct {
		name string

		customerID   uuid.UUID
		confbridgeID uuid.UUID

		expectReqActions []action.Action
		responseFlow     *flow.Flow
		expectRes        *flow.Flow
	}{
		{
			"test normal",

			uuid.FromStringOrNil("e8d81018-8ca5-11ec-99e0-6ff2cca2a2d9"),
			uuid.FromStringOrNil("e926b54c-8ca5-11ec-84bf-036e13d83721"),

			[]action.Action{
				{
					Type:   action.TypeConfbridgeJoin,
					Option: []byte(`{"confbridge_id":"e926b54c-8ca5-11ec-84bf-036e13d83721"}`),
				},
			},
			&flow.Flow{
				ID: uuid.FromStringOrNil("4abf1d80-8ca6-11ec-b130-7b0a22a773f8"),
			},
			&flow.Flow{
				ID: uuid.FromStringOrNil("4abf1d80-8ca6-11ec-b130-7b0a22a773f8"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockAction := actionhandler.NewMockActionHandler(mc)
			h := &activeflowHandler{
				db:            mockDB,
				notifyHandler: mockNotify,
				reqHandler:    mockReq,
				actionHandler: mockAction,
			}

			ctx := context.Background()

			mockReq.EXPECT().FlowV1FlowCreate(ctx, tt.customerID, flow.TypeFlow, gomock.Any(), gomock.Any(), tt.expectReqActions, false).Return(tt.responseFlow, nil)
			res, err := h.generateFlowForAgentCall(ctx, tt.customerID, tt.confbridgeID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}
