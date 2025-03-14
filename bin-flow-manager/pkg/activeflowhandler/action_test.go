package activeflowhandler

import (
	"context"
	"reflect"
	"testing"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-flow-manager/models/action"
	"monorepo/bin-flow-manager/models/activeflow"
	"monorepo/bin-flow-manager/models/flow"
	"monorepo/bin-flow-manager/models/stack"
	"monorepo/bin-flow-manager/models/variable"
	"monorepo/bin-flow-manager/pkg/actionhandler"
	"monorepo/bin-flow-manager/pkg/dbhandler"
	"monorepo/bin-flow-manager/pkg/stackmaphandler"
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
			name: "normal",

			flowID:     uuid.FromStringOrNil("860bef82-f47d-11ec-9eed-6345e27af38c"),
			customerID: uuid.FromStringOrNil("864ec0f0-f47d-11ec-83d6-0f1b5f8a9507"),

			responseFlow: &flow.Flow{
				ID:         uuid.FromStringOrNil("860bef82-f47d-11ec-9eed-6345e27af38c"),
				CustomerID: uuid.FromStringOrNil("864ec0f0-f47d-11ec-83d6-0f1b5f8a9507"),
				Actions: []action.Action{
					{
						ID:   uuid.FromStringOrNil("16bb5d9c-f47e-11ec-8feb-23613c5e54da"),
						Type: action.TypeAnswer,
					},
				},
			},

			expectRes: []action.Action{
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
			mockStack := stackmaphandler.NewMockStackmapHandler(mc)
			mockVar := variablehandler.NewMockVariableHandler(mc)

			h := &activeflowHandler{
				db:              mockDB,
				reqHandler:      mockReq,
				notifyHandler:   mockNotify,
				actionHandler:   mockAction,
				stackmapHandler: mockStack,
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

func Test_updateNextAction(t *testing.T) {

	tests := []struct {
		name string

		activeflowID    uuid.UUID
		currentActionID uuid.UUID

		responseActiveflow *activeflow.Activeflow
		responseStackID    uuid.UUID
		responseAction     *action.Action
		responseVariable   *variable.Variable

		expectResStackID uuid.UUID
		expectRes        *activeflow.Activeflow
	}{
		{
			name: "normal",

			activeflowID:    uuid.FromStringOrNil("bfb07efc-0081-11f0-b869-8fbf4d9c8583"),
			currentActionID: uuid.FromStringOrNil("bfcb3c4c-0081-11f0-a3b1-0b82da5f5632"),

			responseActiveflow: &activeflow.Activeflow{
				ID: uuid.FromStringOrNil("bfb07efc-0081-11f0-b869-8fbf4d9c8583"),

				CurrentAction: action.Action{
					ID: uuid.FromStringOrNil("bfcb3c4c-0081-11f0-a3b1-0b82da5f5632"),
				},
			},
			responseStackID: uuid.FromStringOrNil("e5cc26ee-0082-11f0-b97e-97f72d5c046a"),
			responseAction: &action.Action{
				ID: uuid.FromStringOrNil("c018d588-0081-11f0-80f4-5386b507569a"),
			},
			responseVariable: &variable.Variable{},

			expectResStackID: stack.IDMain,
			expectRes: &activeflow.Activeflow{
				ID:             uuid.FromStringOrNil("bfb07efc-0081-11f0-b869-8fbf4d9c8583"),
				CurrentStackID: uuid.FromStringOrNil("e5cc26ee-0082-11f0-b97e-97f72d5c046a"),
				CurrentAction: action.Action{
					ID: uuid.FromStringOrNil("c018d588-0081-11f0-80f4-5386b507569a"),
				},
				ExecuteCount: 1,
				ExecutedActions: []action.Action{
					{
						ID: uuid.FromStringOrNil("bfcb3c4c-0081-11f0-a3b1-0b82da5f5632"),
					},
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
			mockStack := stackmaphandler.NewMockStackmapHandler(mc)
			mockVar := variablehandler.NewMockVariableHandler(mc)

			h := &activeflowHandler{
				db:              mockDB,
				reqHandler:      mockReq,
				notifyHandler:   mockNotify,
				actionHandler:   mockAction,
				stackmapHandler: mockStack,
				variableHandler: mockVar,
			}
			ctx := context.Background()

			mockDB.EXPECT().ActiveflowGetWithLock(ctx, tt.activeflowID).Return(tt.responseActiveflow, nil)
			mockDB.EXPECT().ActiveflowReleaseLock(ctx, tt.activeflowID).Return(nil)

			mockStack.EXPECT().GetNextAction(tt.responseActiveflow.StackMap, tt.responseActiveflow.CurrentStackID, &tt.responseActiveflow.CurrentAction, true).Return(tt.responseStackID, tt.responseAction)

			mockVar.EXPECT().Get(ctx, tt.activeflowID).Return(tt.responseVariable, nil)
			mockVar.EXPECT().SubstituteByte(ctx, tt.responseAction.Option, tt.responseVariable).Return(tt.responseAction.Option)

			mockDB.EXPECT().ActiveflowGet(ctx, tt.activeflowID).Return(tt.responseActiveflow, nil)
			mockDB.EXPECT().ActiveflowUpdate(ctx, gomock.Any()).Return(nil)
			mockDB.EXPECT().ActiveflowGet(ctx, tt.activeflowID).Return(tt.responseActiveflow, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseActiveflow.CustomerID, activeflow.EventTypeActiveflowUpdated, tt.responseActiveflow)

			res, err := h.updateNextAction(ctx, tt.activeflowID, tt.currentActionID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
