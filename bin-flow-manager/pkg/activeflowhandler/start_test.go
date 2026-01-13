package activeflowhandler

import (
	"context"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-flow-manager/models/action"
	"monorepo/bin-flow-manager/models/activeflow"
	"monorepo/bin-flow-manager/models/flow"
	"monorepo/bin-flow-manager/models/stack"
	"monorepo/bin-flow-manager/models/variable"
	"monorepo/bin-flow-manager/pkg/dbhandler"
	"monorepo/bin-flow-manager/pkg/stackmaphandler"
	"monorepo/bin-flow-manager/pkg/variablehandler"
	"reflect"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_startOnCompleteFlow(t *testing.T) {

	tests := []struct {
		name string

		activeflow *activeflow.Activeflow

		responseUUIDActiveflowID uuid.UUID
		responseFlow             *flow.Flow
		responseStack            map[uuid.UUID]*stack.Stack
		responseVariable         *variable.Variable
		responseVariableParent   *variable.Variable
		responseVariableNew      *variable.Variable

		expectedActiveflow *activeflow.Activeflow
		expectedVariables  map[string]string
	}{
		{
			name: "normal",

			activeflow: &activeflow.Activeflow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("1ae88584-ce64-11f0-8579-f77458ccdaa0"),
				},
				ReferenceType:    activeflow.ReferenceTypeCall,
				ReferenceID:      uuid.FromStringOrNil("1b1ad4c6-ce64-11f0-baa3-9f08199af353"),
				OnCompleteFlowID: uuid.FromStringOrNil("1b3f56c0-ce64-11f0-b334-1b77b8b401e9"),
			},

			responseUUIDActiveflowID: uuid.FromStringOrNil("1b63a214-ce64-11f0-9900-3b2ac4ac935e"),
			responseFlow: &flow.Flow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("1b3f56c0-ce64-11f0-b334-1b77b8b401e9"),
				},
				Actions: []action.Action{
					{
						Type: action.TypeAnswer,
					},
				},
				OnCompleteFlowID: uuid.FromStringOrNil("1b88f3a2-ce64-11f0-8b22-7f77afcc4652"),
			},
			responseStack: map[uuid.UUID]*stack.Stack{
				uuid.FromStringOrNil("ee6076e2-ce64-11f0-b7c8-3340c4791a16"): {
					ID: uuid.FromStringOrNil("ee6076e2-ce64-11f0-b7c8-3340c4791a16"),
				},
			},
			responseVariable: &variable.Variable{},
			responseVariableParent: &variable.Variable{
				Variables: map[string]string{
					"key1":                          "val1",
					"key2":                          "val2",
					variableActiveflowCompleteCount: "2",
				},
			},
			responseVariableNew: &variable.Variable{
				Variables: map[string]string{
					"key2": "new val2",
					"key3": "new val3",
				},
			},

			expectedActiveflow: &activeflow.Activeflow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("1b63a214-ce64-11f0-9900-3b2ac4ac935e"),
				},
				Status:                activeflow.StatusRunning,
				FlowID:                uuid.FromStringOrNil("1b3f56c0-ce64-11f0-b334-1b77b8b401e9"),
				ReferenceType:         activeflow.ReferenceTypeCall,
				ReferenceID:           uuid.FromStringOrNil("1b1ad4c6-ce64-11f0-baa3-9f08199af353"),
				ReferenceActiveflowID: uuid.FromStringOrNil("1ae88584-ce64-11f0-8579-f77458ccdaa0"),
				OnCompleteFlowID:      uuid.FromStringOrNil("1b88f3a2-ce64-11f0-8b22-7f77afcc4652"),
				StackMap: map[uuid.UUID]*stack.Stack{
					uuid.FromStringOrNil("ee6076e2-ce64-11f0-b7c8-3340c4791a16"): {
						ID: uuid.FromStringOrNil("ee6076e2-ce64-11f0-b7c8-3340c4791a16"),
					},
				},
				CurrentStackID: stack.IDMain,
				CurrentAction: action.Action{
					ID: action.IDStart,
				},
				ExecutedActions: []action.Action{},
			},
			expectedVariables: map[string]string{
				"key1":                                  "val1",
				"key2":                                  "val2",
				variableActiveflowID:                    "1b63a214-ce64-11f0-9900-3b2ac4ac935e",
				variableActiveflowReferenceType:         "call",
				variableActiveflowReferenceID:           "1b1ad4c6-ce64-11f0-baa3-9f08199af353",
				variableActiveflowReferenceActiveflowID: "1ae88584-ce64-11f0-8579-f77458ccdaa0",
				variableActiveflowFlowID:                "1b3f56c0-ce64-11f0-b334-1b77b8b401e9",
				variableActiveflowCompleteCount:         "3",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockVar := variablehandler.NewMockVariableHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockStack := stackmaphandler.NewMockStackmapHandler(mc)
			h := &activeflowHandler{
				db:              mockDB,
				reqHandler:      mockReq,
				variableHandler: mockVar,
				utilHandler:     mockUtil,
				notifyHandler:   mockNotify,
				stackmapHandler: mockStack,
			}
			ctx := context.Background()

			mockUtil.EXPECT().UUIDCreate(.Return(tt.responseUUIDActiveflowID)
			mockReq.EXPECT().FlowV1FlowGet(ctx, tt.activeflow.OnCompleteFlowID.Return(tt.responseFlow, nil)
			mockStack.EXPECT().Create(tt.responseFlow.Actions.Return(tt.responseStack)
			mockDB.EXPECT().ActiveflowCreate(ctx, tt.expectedActiveflow.Return(nil)
			mockDB.EXPECT().ActiveflowGet(ctx, tt.responseUUIDActiveflowID.Return(tt.expectedActiveflow, nil)

			mockVar.EXPECT().Get(ctx, tt.activeflow.ID.Return(tt.responseVariableParent, nil)
			mockVar.EXPECT().Create(ctx, tt.expectedActiveflow.ID, tt.expectedVariables.Return(tt.responseVariable, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.expectedActiveflow.CustomerID, activeflow.EventTypeActiveflowCreated, tt.expectedActiveflow)

			mockReq.EXPECT().FlowV1ActiveflowExecute(ctx, tt.expectedActiveflow.ID.Return(nil)

			res, err := h.startOnCompleteFlow(ctx, tt.activeflow)
			if err != nil {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", nil, err)
			}

			if reflect.DeepEqual(res, tt.expectedActiveflow) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectedActiveflow, res)
			}

			time.Sleep(time.Millisecond * 100) // wait for goroutine
		})
	}
}
