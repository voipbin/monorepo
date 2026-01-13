package activeflowhandler

import (
	"context"
	"reflect"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-flow-manager/models/action"
	"monorepo/bin-flow-manager/models/activeflow"
	"monorepo/bin-flow-manager/models/flow"
	"monorepo/bin-flow-manager/models/stack"
	"monorepo/bin-flow-manager/models/variable"
	"monorepo/bin-flow-manager/pkg/dbhandler"
	"monorepo/bin-flow-manager/pkg/stackmaphandler"
	"monorepo/bin-flow-manager/pkg/variablehandler"
)

func Test_Stop(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseActiveflow *activeflow.Activeflow

		expectUpdateFields map[activeflow.Field]any
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("6d8a9464-c8d9-11ed-abfb-d3b58a5adf22"),

			responseActiveflow: &activeflow.Activeflow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("6d8a9464-c8d9-11ed-abfb-d3b58a5adf22"),
				},
				ReferenceType: activeflow.ReferenceTypeCall,
			},

			expectUpdateFields: map[activeflow.Field]any{
				activeflow.FieldStatus: activeflow.StatusEnded,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)

			h := &activeflowHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,
				reqHandler:    mockReq,
			}
			ctx := context.Background()

			mockDB.EXPECT().ActiveflowGet(ctx, tt.id.Return(tt.responseActiveflow, nil)
			mockDB.EXPECT().ActiveflowUpdate(ctx, tt.id, tt.expectUpdateFields.Return(nil)
			mockDB.EXPECT().ActiveflowGet(ctx, tt.id.Return(tt.responseActiveflow, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseActiveflow.CustomerID, activeflow.EventTypeActiveflowUpdated, tt.responseActiveflow)

			res, err := h.Stop(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.responseActiveflow, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseActiveflow, nil)
			}
		})
	}
}

func Test_Stop_has_oncompleteflowid(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseActiveflow *activeflow.Activeflow
		responseUUID       uuid.UUID
		responseFlow       *flow.Flow
		responseVariable   *variable.Variable

		expectUpdateFields map[activeflow.Field]any
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("45a93b90-cf2e-11f0-ae57-977044b1617c"),

			responseActiveflow: &activeflow.Activeflow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("45a93b90-cf2e-11f0-ae57-977044b1617c"),
				},
				ReferenceType:         activeflow.ReferenceTypeCall,
				ReferenceActiveflowID: uuid.FromStringOrNil("93c21170-cf3e-11f0-80dc-ef056d8c53c4"),
				OnCompleteFlowID:      uuid.FromStringOrNil("460e118c-cf2e-11f0-9d69-ef85539104a1"),
			},
			responseUUID: uuid.FromStringOrNil("4637245a-cf2e-11f0-a0cf-73b2b29c8931"),
			responseFlow: &flow.Flow{
				Actions: []action.Action{},
			},
			responseVariable: &variable.Variable{
				Variables: map[string]string{
					variableActiveflowCompleteCount: "2",
				},
			},

			expectUpdateFields: map[activeflow.Field]any{
				activeflow.FieldStatus: activeflow.StatusEnded,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockStack := stackmaphandler.NewMockStackmapHandler(mc)
			mockVar := variablehandler.NewMockVariableHandler(mc)

			h := &activeflowHandler{
				utilHandler:     mockUtil,
				db:              mockDB,
				notifyHandler:   mockNotify,
				reqHandler:      mockReq,
				stackmapHandler: mockStack,
				variableHandler: mockVar,
			}
			ctx := context.Background()

			mockDB.EXPECT().ActiveflowGet(ctx, tt.id.Return(tt.responseActiveflow, nil)
			mockDB.EXPECT().ActiveflowUpdate(ctx, tt.id, tt.expectUpdateFields.Return(nil)
			mockDB.EXPECT().ActiveflowGet(ctx, tt.id.Return(tt.responseActiveflow, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseActiveflow.CustomerID, activeflow.EventTypeActiveflowUpdated, tt.responseActiveflow)

			// startOnCompleteFlow
			mockUtil.EXPECT().UUIDCreate(.Return(tt.responseUUID)
			mockReq.EXPECT().FlowV1FlowGet(ctx, tt.responseActiveflow.OnCompleteFlowID.Return(tt.responseFlow, nil)
			mockStack.EXPECT().Create(gomock.Any().Return(map[uuid.UUID]*stack.Stack{})

			mockDB.EXPECT().ActiveflowCreate(ctx, gomock.Any().Return(nil)
			mockDB.EXPECT().ActiveflowGet(ctx, gomock.Any().Return(tt.responseActiveflow, nil)

			mockVar.EXPECT().Get(ctx, gomock.Any().Return(tt.responseVariable, nil)
			mockVar.EXPECT().Create(ctx, gomock.Any(), gomock.Any().Return(&variable.Variable{}, nil)

			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseActiveflow.CustomerID, activeflow.EventTypeActiveflowCreated, gomock.Any())

			mockReq.EXPECT().FlowV1ActiveflowExecute(ctx, gomock.Any().Return(nil)

			res, err := h.Stop(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.responseActiveflow, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseActiveflow, nil)
			}

			time.Sleep(time.Millisecond * 100)
		})
	}
}
