package activeflowhandler

import (
	"context"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-flow-manager/models/action"
	"monorepo/bin-flow-manager/models/activeflow"
	"monorepo/bin-flow-manager/models/stack"
	"monorepo/bin-flow-manager/pkg/actionhandler"
	"monorepo/bin-flow-manager/pkg/dbhandler"
	"monorepo/bin-flow-manager/pkg/stackmaphandler"
	"monorepo/bin-flow-manager/pkg/variablehandler"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_PushStack(t *testing.T) {

	tests := []struct {
		name string

		activeflow *activeflow.Activeflow
		stackID    uuid.UUID
		actions    []action.Action

		responseStack *stack.Stack

		expectUpdateFields map[activeflow.Field]any
	}{
		{
			name: "normal",

			activeflow: &activeflow.Activeflow{
				CurrentStackID: stack.IDMain,
				CurrentAction: action.Action{
					ID:   uuid.FromStringOrNil("03ed5604-faf2-11ed-a7a9-5bebe50227a1"),
					Type: action.TypeAnswer,
				},
				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:   uuid.FromStringOrNil("03ed5604-faf2-11ed-a7a9-5bebe50227a1"),
								Type: action.TypeAnswer,
							},
						},
					},
				},
				ExecutedActions: []action.Action{},
			},
			stackID: uuid.FromStringOrNil("1c18d8a8-f9af-11ef-9d84-571979c7a171"),
			actions: []action.Action{
				{
					Type: action.TypeAnswer,
				},
			},

			responseStack: &stack.Stack{
				ID: uuid.FromStringOrNil("0484ffcc-faf2-11ed-b3af-a36c3fe16feb"),
				Actions: []action.Action{
					{
						ID: uuid.FromStringOrNil("04b85098-faf2-11ed-8a6e-23db5e3a944f"),
					},
				},
			},

			expectUpdateFields: map[activeflow.Field]any{
				activeflow.FieldCurrentStackID: stack.IDMain,
				activeflow.FieldCurrentAction: action.Action{
					ID:   uuid.FromStringOrNil("03ed5604-faf2-11ed-a7a9-5bebe50227a1"),
					Type: action.TypeAnswer,
				},

				activeflow.FieldForwardStackID:  uuid.FromStringOrNil("0484ffcc-faf2-11ed-b3af-a36c3fe16feb"),
				activeflow.FieldForwardActionID: uuid.FromStringOrNil("04b85098-faf2-11ed-8a6e-23db5e3a944f"),

				activeflow.FieldStackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:   uuid.FromStringOrNil("03ed5604-faf2-11ed-a7a9-5bebe50227a1"),
								Type: action.TypeAnswer,
							},
						},
					},
				},

				activeflow.FieldExecuteCount:    uint64(0),
				activeflow.FieldExecutedActions: []action.Action{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockVariableHandler := variablehandler.NewMockVariableHandler(mc)
			mockStack := stackmaphandler.NewMockStackmapHandler(mc)

			h := &activeflowHandler{
				db:              mockDB,
				notifyHandler:   mockNotify,
				variableHandler: mockVariableHandler,
				stackmapHandler: mockStack,
			}
			ctx := context.Background()

			mockStack.EXPECT().PushStackByActions(tt.activeflow.StackMap, tt.stackID, tt.actions, tt.activeflow.CurrentStackID, tt.activeflow.CurrentAction.ID).Return(tt.responseStack, nil)
			mockDB.EXPECT().ActiveflowUpdate(ctx, tt.activeflow.ID, tt.expectUpdateFields).Return(nil)

			if err := h.PushStack(ctx, tt.activeflow, tt.stackID, tt.actions); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_PushActions(t *testing.T) {

	tests := []struct {
		name string

		id      uuid.UUID
		actions []action.Action

		responseActiveflow *activeflow.Activeflow
		responseActions    []action.Action
		responseStack      *stack.Stack

		expectUpdateFields map[activeflow.Field]any
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("0f201196-faf3-11ed-961e-931b700f4aa9"),
			actions: []action.Action{
				{
					Type: action.TypeAnswer,
				},
			},

			responseActiveflow: &activeflow.Activeflow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("0f201196-faf3-11ed-961e-931b700f4aa9"),
				},
				CurrentStackID: stack.IDMain,
				CurrentAction: action.Action{
					ID:   uuid.FromStringOrNil("0f750bf6-faf3-11ed-b55c-7f641ca48cda"),
					Type: action.TypeAnswer,
				},
				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:   uuid.FromStringOrNil("0f750bf6-faf3-11ed-b55c-7f641ca48cda"),
								Type: action.TypeAnswer,
							},
						},
					},
				},
				ExecutedActions: []action.Action{},
			},
			responseActions: []action.Action{
				{
					ID:   uuid.FromStringOrNil("0fa853b2-faf3-11ed-a1ca-4bebadf662f8"),
					Type: action.TypeAnswer,
				},
			},
			responseStack: &stack.Stack{
				ID: uuid.FromStringOrNil("83878a82-faf3-11ed-8658-0324081290cc"),
				Actions: []action.Action{
					{
						ID: uuid.FromStringOrNil("0fa853b2-faf3-11ed-a1ca-4bebadf662f8"),
					},
				},
			},

			expectUpdateFields: map[activeflow.Field]any{
				activeflow.FieldCurrentStackID: stack.IDMain,
				activeflow.FieldCurrentAction: action.Action{
					ID:   uuid.FromStringOrNil("0f750bf6-faf3-11ed-b55c-7f641ca48cda"),
					Type: action.TypeAnswer,
				},

				activeflow.FieldForwardStackID:  uuid.FromStringOrNil("83878a82-faf3-11ed-8658-0324081290cc"),
				activeflow.FieldForwardActionID: uuid.FromStringOrNil("0fa853b2-faf3-11ed-a1ca-4bebadf662f8"),

				activeflow.FieldStackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:   uuid.FromStringOrNil("0f750bf6-faf3-11ed-b55c-7f641ca48cda"),
								Type: action.TypeAnswer,
							},
						},
					},
				},

				activeflow.FieldExecuteCount:    uint64(0),
				activeflow.FieldExecutedActions: []action.Action{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockVariableHandler := variablehandler.NewMockVariableHandler(mc)
			mockStack := stackmaphandler.NewMockStackmapHandler(mc)
			mockAction := actionhandler.NewMockActionHandler(mc)

			h := &activeflowHandler{
				db:              mockDB,
				notifyHandler:   mockNotify,
				variableHandler: mockVariableHandler,
				stackmapHandler: mockStack,
				actionHandler:   mockAction,
			}
			ctx := context.Background()

			mockDB.EXPECT().ActiveflowGet(ctx, tt.id).Return(tt.responseActiveflow, nil)
			mockAction.EXPECT().GenerateFlowActions(ctx, tt.actions).Return(tt.responseActions, nil)

			// push stack
			mockStack.EXPECT().PushStackByActions(tt.responseActiveflow.StackMap, uuid.Nil, tt.responseActions, tt.responseActiveflow.CurrentStackID, tt.responseActiveflow.CurrentAction.ID).Return(tt.responseStack, nil)
			mockDB.EXPECT().ActiveflowUpdate(ctx, tt.id, tt.expectUpdateFields).Return(nil)

			mockDB.EXPECT().ActiveflowGet(ctx, tt.id).Return(tt.responseActiveflow, nil)

			res, err := h.PushActions(ctx, tt.id, tt.actions)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseActiveflow) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.responseActiveflow, res)
			}
		})
	}
}

func Test_PopStackWithStackID(t *testing.T) {

	tests := []struct {
		name string

		af      *activeflow.Activeflow
		stackID uuid.UUID

		responseStack          *stack.Stack
		responseForwardStackID uuid.UUID
		responseForwardAction  *action.Action

		expectUpdateFields map[activeflow.Field]any
	}{
		{
			name: "normal",

			af: &activeflow.Activeflow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("fa75a570-f9e3-11ef-ade5-b321bb533392"),
				},
				CurrentStackID: uuid.FromStringOrNil("fba1274e-f9e3-11ef-8a79-afb93b277619"),
				CurrentAction:  action.Action{},
				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID: uuid.FromStringOrNil("fac2d2b4-f9e3-11ef-be40-130c380abed9"),
							},
						},
					},
					uuid.FromStringOrNil("fba1274e-f9e3-11ef-8a79-afb93b277619"): {
						ID: uuid.FromStringOrNil("fba1274e-f9e3-11ef-8a79-afb93b277619"),
						Actions: []action.Action{
							{
								ID: uuid.FromStringOrNil("62931778-f9e4-11ef-8c40-e7607cdb16ea"),
							},
						},
						ReturnStackID:  stack.IDMain,
						ReturnActionID: uuid.FromStringOrNil("fac2d2b4-f9e3-11ef-be40-130c380abed9"),
					},
				},
				ExecutedActions: []action.Action{},
			},
			stackID: uuid.FromStringOrNil("fba1274e-f9e3-11ef-8a79-afb93b277619"),

			responseStack: &stack.Stack{
				ID: uuid.FromStringOrNil("fba1274e-f9e3-11ef-8a79-afb93b277619"),
				Actions: []action.Action{
					{
						ID: uuid.FromStringOrNil("62931778-f9e4-11ef-8c40-e7607cdb16ea"),
					},
				},
				ReturnStackID:  stack.IDMain,
				ReturnActionID: uuid.FromStringOrNil("fac2d2b4-f9e3-11ef-be40-130c380abed9"),
			},
			responseForwardStackID: stack.IDMain,
			responseForwardAction:  &action.Action{ID: uuid.FromStringOrNil("fac2d2b4-f9e3-11ef-be40-130c380abed9")},

			expectUpdateFields: map[activeflow.Field]any{
				activeflow.FieldCurrentStackID: uuid.FromStringOrNil("fba1274e-f9e3-11ef-8a79-afb93b277619"),
				activeflow.FieldCurrentAction:  action.Action{},

				activeflow.FieldForwardStackID:  stack.IDMain,
				activeflow.FieldForwardActionID: uuid.FromStringOrNil("fac2d2b4-f9e3-11ef-be40-130c380abed9"),

				activeflow.FieldStackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID: uuid.FromStringOrNil("fac2d2b4-f9e3-11ef-be40-130c380abed9"),
							},
						},
					},
					uuid.FromStringOrNil("fba1274e-f9e3-11ef-8a79-afb93b277619"): {
						ID: uuid.FromStringOrNil("fba1274e-f9e3-11ef-8a79-afb93b277619"),
						Actions: []action.Action{
							{
								ID: uuid.FromStringOrNil("62931778-f9e4-11ef-8c40-e7607cdb16ea"),
							},
						},
						ReturnStackID:  stack.IDMain,
						ReturnActionID: uuid.FromStringOrNil("fac2d2b4-f9e3-11ef-be40-130c380abed9"),
					},
				},

				activeflow.FieldExecuteCount:    uint64(0),
				activeflow.FieldExecutedActions: []action.Action{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockVariableHandler := variablehandler.NewMockVariableHandler(mc)
			mockStack := stackmaphandler.NewMockStackmapHandler(mc)
			mockAction := actionhandler.NewMockActionHandler(mc)

			h := &activeflowHandler{
				db:              mockDB,
				notifyHandler:   mockNotify,
				variableHandler: mockVariableHandler,
				stackmapHandler: mockStack,
				actionHandler:   mockAction,
			}
			ctx := context.Background()

			mockStack.EXPECT().PopStack(tt.af.StackMap, tt.stackID).Return(tt.responseStack, nil)
			mockStack.EXPECT().GetNextAction(tt.af.StackMap, tt.responseStack.ReturnStackID, tt.responseStack.ReturnActionID, true).Return(tt.responseForwardStackID, tt.responseForwardAction)
			mockDB.EXPECT().ActiveflowUpdate(ctx, tt.af.ID, tt.expectUpdateFields).Return(nil)

			if err := h.PopStackWithStackID(ctx, tt.af, tt.stackID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_PopStack(t *testing.T) {

	tests := []struct {
		name string

		af *activeflow.Activeflow

		responseStack          *stack.Stack
		responseForwardStackID uuid.UUID
		responseForwardAction  *action.Action

		expectStackID      uuid.UUID
		expectUpdateFields map[activeflow.Field]any
	}{
		{
			name: "normal",

			af: &activeflow.Activeflow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a4bbdcc4-f9e5-11ef-b4f8-c30c2a699e54"),
				},
				CurrentStackID: uuid.FromStringOrNil("a555f502-f9e5-11ef-bbbe-b363e26fda0f"),
				CurrentAction:  action.Action{},
				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID: uuid.FromStringOrNil("a5887874-f9e5-11ef-a64e-3f6c58935e86"),
							},
						},
					},
					uuid.FromStringOrNil("a555f502-f9e5-11ef-bbbe-b363e26fda0f"): {
						ID: uuid.FromStringOrNil("a555f502-f9e5-11ef-bbbe-b363e26fda0f"),
						Actions: []action.Action{
							{
								ID: uuid.FromStringOrNil("62931778-f9e4-11ef-8c40-e7607cdb16ea"),
							},
						},
						ReturnStackID:  stack.IDMain,
						ReturnActionID: uuid.FromStringOrNil("a5887874-f9e5-11ef-a64e-3f6c58935e86"),
					},
				},
				ExecutedActions: []action.Action{},
			},

			responseStack: &stack.Stack{
				ID: uuid.FromStringOrNil("a555f502-f9e5-11ef-bbbe-b363e26fda0f"),
				Actions: []action.Action{
					{
						ID: uuid.FromStringOrNil("62931778-f9e4-11ef-8c40-e7607cdb16ea"),
					},
				},
				ReturnStackID:  stack.IDMain,
				ReturnActionID: uuid.FromStringOrNil("a5887874-f9e5-11ef-a64e-3f6c58935e86"),
			},
			responseForwardStackID: stack.IDMain,
			responseForwardAction: &action.Action{
				ID: uuid.FromStringOrNil("a5887874-f9e5-11ef-a64e-3f6c58935e86"),
			},

			expectStackID: uuid.FromStringOrNil("a555f502-f9e5-11ef-bbbe-b363e26fda0f"),
			expectUpdateFields: map[activeflow.Field]any{
				activeflow.FieldCurrentStackID: uuid.FromStringOrNil("a555f502-f9e5-11ef-bbbe-b363e26fda0f"),
				activeflow.FieldCurrentAction:  action.Action{},

				activeflow.FieldForwardStackID:  stack.IDMain,
				activeflow.FieldForwardActionID: uuid.FromStringOrNil("a5887874-f9e5-11ef-a64e-3f6c58935e86"),

				activeflow.FieldStackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID: uuid.FromStringOrNil("a5887874-f9e5-11ef-a64e-3f6c58935e86"),
							},
						},
					},
					uuid.FromStringOrNil("a555f502-f9e5-11ef-bbbe-b363e26fda0f"): {
						ID: uuid.FromStringOrNil("a555f502-f9e5-11ef-bbbe-b363e26fda0f"),
						Actions: []action.Action{
							{
								ID: uuid.FromStringOrNil("62931778-f9e4-11ef-8c40-e7607cdb16ea"),
							},
						},
						ReturnStackID:  stack.IDMain,
						ReturnActionID: uuid.FromStringOrNil("a5887874-f9e5-11ef-a64e-3f6c58935e86"),
					},
				},

				activeflow.FieldExecuteCount:    uint64(0),
				activeflow.FieldExecutedActions: []action.Action{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockVariableHandler := variablehandler.NewMockVariableHandler(mc)
			mockStack := stackmaphandler.NewMockStackmapHandler(mc)
			mockAction := actionhandler.NewMockActionHandler(mc)

			h := &activeflowHandler{
				db:              mockDB,
				notifyHandler:   mockNotify,
				variableHandler: mockVariableHandler,
				stackmapHandler: mockStack,
				actionHandler:   mockAction,
			}
			ctx := context.Background()

			mockStack.EXPECT().PopStack(tt.af.StackMap, tt.expectStackID).Return(tt.responseStack, nil)
			mockStack.EXPECT().GetNextAction(tt.af.StackMap, tt.responseStack.ReturnStackID, tt.responseStack.ReturnActionID, true).Return(tt.responseForwardStackID, tt.responseForwardAction)
			mockDB.EXPECT().ActiveflowUpdate(ctx, tt.af.ID, tt.expectUpdateFields).Return(nil)

			if err := h.PopStack(ctx, tt.af); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_AddActions(t *testing.T) {

	tests := []struct {
		name string

		id      uuid.UUID
		actions []action.Action

		responseActions    []action.Action
		responseActiveflow *activeflow.Activeflow

		expectedUpdateFields map[activeflow.Field]any
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("14e4a6f2-03d7-11f0-805e-4f17f94cc4d8"),
			actions: []action.Action{
				{
					Type: action.TypeAnswer,
				},
				{
					Type: action.TypeAnswer,
				},
			},

			responseActions: []action.Action{
				{
					ID:   uuid.FromStringOrNil("150e19ec-03d7-11f0-a2de-a798771ec8ef"),
					Type: action.TypeAnswer,
				},
				{
					ID:   uuid.FromStringOrNil("153775f8-03d7-11f0-a3e6-e35088338058"),
					Type: action.TypeAnswer,
				},
			},
			responseActiveflow: &activeflow.Activeflow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("14e4a6f2-03d7-11f0-805e-4f17f94cc4d8"),
				},
				CurrentStackID: uuid.FromStringOrNil("12f085aa-03ff-11f0-95d1-034a89944bc6"),
				CurrentAction: action.Action{
					ID: uuid.FromStringOrNil("150e19ec-03d7-11f0-a2de-a798771ec8ef"),
				},
				StackMap:        map[uuid.UUID]*stack.Stack{},
				ExecutedActions: []action.Action{},
			},

			expectedUpdateFields: map[activeflow.Field]any{
				activeflow.FieldCurrentStackID:  uuid.FromStringOrNil("12f085aa-03ff-11f0-95d1-034a89944bc6"),
				activeflow.FieldCurrentAction:   action.Action{ID: uuid.FromStringOrNil("150e19ec-03d7-11f0-a2de-a798771ec8ef")},
				activeflow.FieldForwardStackID:  stack.IDEmpty,
				activeflow.FieldForwardActionID: action.IDEmpty,
				activeflow.FieldStackMap:        map[uuid.UUID]*stack.Stack{},
				activeflow.FieldExecuteCount:    uint64(0),
				activeflow.FieldExecutedActions: []action.Action{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockVariableHandler := variablehandler.NewMockVariableHandler(mc)
			mockStack := stackmaphandler.NewMockStackmapHandler(mc)
			mockAction := actionhandler.NewMockActionHandler(mc)

			h := &activeflowHandler{
				db:              mockDB,
				notifyHandler:   mockNotify,
				variableHandler: mockVariableHandler,
				stackmapHandler: mockStack,
				actionHandler:   mockAction,
			}
			ctx := context.Background()

			mockAction.EXPECT().GenerateFlowActions(ctx, tt.actions).Return(tt.responseActions, nil)
			mockDB.EXPECT().ActiveflowGet(ctx, tt.id).Return(tt.responseActiveflow, nil)
			mockStack.EXPECT().AddActions(tt.responseActiveflow.StackMap, tt.responseActiveflow.CurrentStackID, tt.responseActiveflow.CurrentAction.ID, tt.responseActions).Return(nil)
			mockDB.EXPECT().ActiveflowUpdate(ctx, tt.id, tt.expectedUpdateFields).Return(nil)
			mockDB.EXPECT().ActiveflowGet(ctx, tt.id).Return(tt.responseActiveflow, nil)

			res, err := h.AddActions(ctx, tt.id, tt.actions)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseActiveflow) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.responseActiveflow, res)
			}
		})
	}
}

func Test_ServiceStop(t *testing.T) {

	tests := []struct {
		name string

		id        uuid.UUID
		serviceID uuid.UUID

		responseActiveflow *activeflow.Activeflow

		responseStack          *stack.Stack
		responseForwardStackID uuid.UUID
		responseForwardAction  *action.Action

		expectStackID      uuid.UUID
		expectUpdateFields map[activeflow.Field]any
	}{
		{
			name: "normal",

			id:        uuid.FromStringOrNil("0bfb993e-f9ea-11ef-8a00-87061c4d89e7"),
			serviceID: uuid.FromStringOrNil("0c3dbf8a-f9ea-11ef-87bb-0b801c31899c"),

			responseActiveflow: &activeflow.Activeflow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("0bfb993e-f9ea-11ef-8a00-87061c4d89e7"),
				},
				CurrentStackID: uuid.FromStringOrNil("0c3dbf8a-f9ea-11ef-87bb-0b801c31899c"),
				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID: uuid.FromStringOrNil("a5887874-f9e5-11ef-a64e-3f6c58935e86"),
							},
						},
					},
					uuid.FromStringOrNil("0c3dbf8a-f9ea-11ef-87bb-0b801c31899c"): {
						ID: uuid.FromStringOrNil("0c3dbf8a-f9ea-11ef-87bb-0b801c31899c"),
						Actions: []action.Action{
							{
								ID: uuid.FromStringOrNil("62931778-f9e4-11ef-8c40-e7607cdb16ea"),
							},
						},
						ReturnStackID:  stack.IDMain,
						ReturnActionID: uuid.FromStringOrNil("a5887874-f9e5-11ef-a64e-3f6c58935e86"),
					},
				},
				ExecutedActions: []action.Action{},
			},
			responseStack: &stack.Stack{
				ID: uuid.FromStringOrNil("0c3dbf8a-f9ea-11ef-87bb-0b801c31899c"),
				Actions: []action.Action{
					{
						ID: uuid.FromStringOrNil("62931778-f9e4-11ef-8c40-e7607cdb16ea"),
					},
				},
				ReturnStackID:  stack.IDMain,
				ReturnActionID: uuid.FromStringOrNil("a5887874-f9e5-11ef-a64e-3f6c58935e86"),
			},
			responseForwardStackID: stack.IDMain,
			responseForwardAction: &action.Action{
				ID: uuid.FromStringOrNil("a5887874-f9e5-11ef-a64e-3f6c58935e86"),
			},

			expectStackID: uuid.FromStringOrNil("0c3dbf8a-f9ea-11ef-87bb-0b801c31899c"),
			expectUpdateFields: map[activeflow.Field]any{
				activeflow.FieldCurrentStackID: uuid.FromStringOrNil("0c3dbf8a-f9ea-11ef-87bb-0b801c31899c"),
				activeflow.FieldCurrentAction:  action.Action{},

				activeflow.FieldForwardStackID:  stack.IDMain,
				activeflow.FieldForwardActionID: uuid.FromStringOrNil("a5887874-f9e5-11ef-a64e-3f6c58935e86"),

				activeflow.FieldStackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID: uuid.FromStringOrNil("a5887874-f9e5-11ef-a64e-3f6c58935e86"),
							},
						},
					},
					uuid.FromStringOrNil("0c3dbf8a-f9ea-11ef-87bb-0b801c31899c"): {
						ID: uuid.FromStringOrNil("0c3dbf8a-f9ea-11ef-87bb-0b801c31899c"),
						Actions: []action.Action{
							{
								ID: uuid.FromStringOrNil("62931778-f9e4-11ef-8c40-e7607cdb16ea"),
							},
						},
						ReturnStackID:  stack.IDMain,
						ReturnActionID: uuid.FromStringOrNil("a5887874-f9e5-11ef-a64e-3f6c58935e86"),
					},
				},

				activeflow.FieldExecuteCount:    uint64(0),
				activeflow.FieldExecutedActions: []action.Action{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockVariableHandler := variablehandler.NewMockVariableHandler(mc)
			mockStack := stackmaphandler.NewMockStackmapHandler(mc)
			mockAction := actionhandler.NewMockActionHandler(mc)

			h := &activeflowHandler{
				db:              mockDB,
				notifyHandler:   mockNotify,
				variableHandler: mockVariableHandler,
				stackmapHandler: mockStack,
				actionHandler:   mockAction,
			}
			ctx := context.Background()

			mockDB.EXPECT().ActiveflowGet(ctx, tt.id).Return(tt.responseActiveflow, nil)
			mockStack.EXPECT().PopStack(tt.responseActiveflow.StackMap, tt.expectStackID).Return(tt.responseStack, nil)
			mockStack.EXPECT().GetNextAction(tt.responseActiveflow.StackMap, tt.responseStack.ReturnStackID, tt.responseStack.ReturnActionID, true).Return(tt.responseForwardStackID, tt.responseForwardAction)
			mockDB.EXPECT().ActiveflowUpdate(ctx, tt.id, tt.expectUpdateFields).Return(nil)

			if err := h.ServiceStop(ctx, tt.id, tt.serviceID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_validateCurrentActionID(t *testing.T) {

	tests := []struct {
		name string

		af   *activeflow.Activeflow
		caID uuid.UUID

		expectedError bool
	}{
		{
			name: "success_current_action_empty",
			af: &activeflow.Activeflow{
				CurrentAction: action.Action{
					ID: action.IDEmpty,
				},
			},
			caID:          uuid.FromStringOrNil("0f789bd0-d95e-11f0-a2fa-c30934e3496e"),
			expectedError: false,
		},
		{
			name: "success_continue_block_action",
			af: &activeflow.Activeflow{
				CurrentAction: action.Action{
					ID:   uuid.FromStringOrNil("0faf3ac8-d95e-11f0-95b0-cb1e6deb9c62"),
					Type: action.TypeBlock,
				},
			},
			caID:          action.IDContinue,
			expectedError: false,
		},
		{
			name: "fail_continue_but_not_block_action",
			af: &activeflow.Activeflow{
				CurrentAction: action.Action{
					ID:   uuid.FromStringOrNil("0fdd83d8-d95e-11f0-96f5-37341a3b89a9"),
					Type: action.TypePlay, // Not Block
				},
			},
			caID:          action.IDContinue,
			expectedError: true,
		},
		{
			name: "success_id_matched",
			af: &activeflow.Activeflow{
				CurrentAction: action.Action{
					ID: uuid.FromStringOrNil("100ae53a-d95e-11f0-9a3d-cffaf4c82614"),
				},
			},
			caID:          uuid.FromStringOrNil("100ae53a-d95e-11f0-9a3d-cffaf4c82614"),
			expectedError: false,
		},
		{
			name: "fail_id_mismatched",
			af: &activeflow.Activeflow{
				CurrentAction: action.Action{
					ID: uuid.FromStringOrNil("1033f1f0-d95e-11f0-9d55-1b84cd5b06df"),
				},
			},
			caID:          uuid.FromStringOrNil("0c3dbf8a-f9ea-11ef-87bb-0b801c31899c"),
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			// Initialize Mocks (Standard procedure based on your style, though not used in this specific method logic)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockVariableHandler := variablehandler.NewMockVariableHandler(mc)
			mockStack := stackmaphandler.NewMockStackmapHandler(mc)
			mockAction := actionhandler.NewMockActionHandler(mc)

			h := &activeflowHandler{
				db:              mockDB,
				notifyHandler:   mockNotify,
				variableHandler: mockVariableHandler,
				stackmapHandler: mockStack,
				actionHandler:   mockAction,
			}

			err := h.validateCurrentActionID(tt.af, tt.caID)
			if (err != nil) != tt.expectedError {
				t.Errorf("validateCurrentActionID() error = %v, wantErr %v", err, tt.expectedError)
			}
		})
	}
}
