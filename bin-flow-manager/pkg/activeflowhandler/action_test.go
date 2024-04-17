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
	"monorepo/bin-flow-manager/pkg/stackhandler"
	"monorepo/bin-flow-manager/pkg/variablehandler"
)

func Test_getActionsFromFlow(t *testing.T) {

	tests := []struct {
		name string

		flowID     uuid.UUID
		customerID uuid.UUID

		responseFlow *flow.Flow

		expectRes []action.Action
	}{
		{
			"normal",

			uuid.FromStringOrNil("860bef82-f47d-11ec-9eed-6345e27af38c"),
			uuid.FromStringOrNil("864ec0f0-f47d-11ec-83d6-0f1b5f8a9507"),

			&flow.Flow{
				ID:         uuid.FromStringOrNil("860bef82-f47d-11ec-9eed-6345e27af38c"),
				CustomerID: uuid.FromStringOrNil("864ec0f0-f47d-11ec-83d6-0f1b5f8a9507"),
				Actions: []action.Action{
					{
						ID:   uuid.FromStringOrNil("16bb5d9c-f47e-11ec-8feb-23613c5e54da"),
						Type: action.TypeAnswer,
					},
				},
			},

			[]action.Action{
				{
					ID:   uuid.FromStringOrNil("16bb5d9c-f47e-11ec-8feb-23613c5e54da"),
					Type: action.TypeAnswer,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockAction := actionhandler.NewMockActionHandler(mc)
			mockStack := stackhandler.NewMockStackHandler(mc)
			mockVar := variablehandler.NewMockVariableHandler(mc)

			h := &activeflowHandler{
				db:              mockDB,
				reqHandler:      mockReq,
				notifyHandler:   mockNotify,
				actionHandler:   mockAction,
				stackHandler:    mockStack,
				variableHandler: mockVar,
			}

			ctx := context.Background()

			mockReq.EXPECT().FlowV1FlowGet(ctx, tt.flowID).Return(tt.responseFlow, nil)

			res, err := h.getActionsFromFlow(ctx, tt.flowID, tt.customerID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

// func Test_getNextAction(t *testing.T) {

// 	tests := []struct {
// 		name string

// 		activeflowID    uuid.UUID
// 		currentActionID uuid.UUID

// 		responseActiveflow *activeflow.Activeflow
// 		responseStackID    uuid.UUID
// 		responseAction     *action.Action
// 		responseVariable   *variable.Variable

// 		expectResStackID uuid.UUID
// 		expectResAction  *action.Action
// 	}{
// 		{
// 			"normal",

// 			uuid.FromStringOrNil("4f49a95c-f48d-11ec-8058-bbbde0a6c271"),
// 			uuid.FromStringOrNil("4f98aed0-f48d-11ec-aa16-1b034ad50819"),

// 			&activeflow.Activeflow{
// 				ID: uuid.FromStringOrNil("4f49a95c-f48d-11ec-8058-bbbde0a6c271"),

// 				CurrentAction: action.Action{
// 					ID: uuid.FromStringOrNil("4f98aed0-f48d-11ec-aa16-1b034ad50819"),
// 				},
// 			},
// 			uuid.FromStringOrNil("37dbe260-f48f-11ec-95e9-43542c39a464"),
// 			&action.Action{
// 				ID:   uuid.FromStringOrNil("9454e1c2-f48f-11ec-b9e5-8ff2c45b0cbc"),
// 				Type: action.TypeAnswer,
// 			},
// 			&variable.Variable{},

// 			uuid.FromStringOrNil("37dbe260-f48f-11ec-95e9-43542c39a464"),
// 			&action.Action{
// 				ID:   uuid.FromStringOrNil("9454e1c2-f48f-11ec-b9e5-8ff2c45b0cbc"),
// 				Type: action.TypeAnswer,
// 			},
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			mc := gomock.NewController(t)
// 			defer mc.Finish()

// 			mockDB := dbhandler.NewMockDBHandler(mc)
// 			mockReq := requesthandler.NewMockRequestHandler(mc)
// 			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
// 			mockAction := actionhandler.NewMockActionHandler(mc)
// 			mockStack := stackhandler.NewMockStackHandler(mc)
// 			mockVar := variablehandler.NewMockVariableHandler(mc)

// 			h := &activeflowHandler{
// 				db:              mockDB,
// 				reqHandler:      mockReq,
// 				notifyHandler:   mockNotify,
// 				actionHandler:   mockAction,
// 				stackHandler:    mockStack,
// 				variableHandler: mockVar,
// 			}

// 			ctx := context.Background()

// 			mockDB.EXPECT().ActiveflowGet(ctx, tt.activeflowID).Return(tt.responseActiveflow, nil)
// 			mockStack.EXPECT().GetNextAction(ctx, tt.responseActiveflow.StackMap, tt.responseActiveflow.CurrentStackID, &tt.responseActiveflow.CurrentAction, true).Return(tt.responseStackID, tt.responseAction)

// 			resStackID, resAction, err := h.getNextAction(ctx, tt.activeflowID, tt.currentActionID)
// 			if err != nil {
// 				t.Errorf("Wrong match. expect: ok, got: %v", err)
// 			}

// 			if resStackID != tt.expectResStackID {
// 				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectResStackID, resStackID)
// 			}

// 			if reflect.DeepEqual(resAction, tt.expectResAction) != true {
// 				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectResAction, resAction)
// 			}
// 		})
// 	}
// }
