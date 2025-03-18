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

		expectActiveflow *activeflow.Activeflow
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

			expectActiveflow: &activeflow.Activeflow{
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

				ForwardStackID:  uuid.FromStringOrNil("0484ffcc-faf2-11ed-b3af-a36c3fe16feb"),
				ForwardActionID: uuid.FromStringOrNil("04b85098-faf2-11ed-8a6e-23db5e3a944f"),
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
			mockDB.EXPECT().ActiveflowUpdate(ctx, tt.expectActiveflow).Return(nil)

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

		expectActiveflow *activeflow.Activeflow
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

			expectActiveflow: &activeflow.Activeflow{
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

				ForwardStackID:  uuid.FromStringOrNil("83878a82-faf3-11ed-8658-0324081290cc"),
				ForwardActionID: uuid.FromStringOrNil("0fa853b2-faf3-11ed-a1ca-4bebadf662f8"),
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
			mockDB.EXPECT().ActiveflowUpdate(ctx, tt.expectActiveflow).Return(nil)

			mockDB.EXPECT().ActiveflowGet(ctx, tt.id).Return(tt.expectActiveflow, nil)

			res, err := h.PushActions(ctx, tt.id, tt.actions)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectActiveflow) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectActiveflow, res)
			}
		})
	}
}

func Test_PopStackWithStackID(t *testing.T) {

	tests := []struct {
		name string

		af      *activeflow.Activeflow
		stackID uuid.UUID

		responseStack *stack.Stack

		expectActiveflow *activeflow.Activeflow
	}{
		{
			name: "normal",

			af: &activeflow.Activeflow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("fa75a570-f9e3-11ef-ade5-b321bb533392"),
				},
				CurrentStackID: uuid.FromStringOrNil("fba1274e-f9e3-11ef-8a79-afb93b277619"),
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
			expectActiveflow: &activeflow.Activeflow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("fa75a570-f9e3-11ef-ade5-b321bb533392"),
				},
				CurrentStackID: uuid.FromStringOrNil("fba1274e-f9e3-11ef-8a79-afb93b277619"),
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
				ForwardStackID:  stack.IDMain,
				ForwardActionID: uuid.FromStringOrNil("fac2d2b4-f9e3-11ef-be40-130c380abed9"),
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
			mockDB.EXPECT().ActiveflowUpdate(ctx, tt.expectActiveflow).Return(nil)

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

		responseStack *stack.Stack

		expectStackID    uuid.UUID
		expectActiveflow *activeflow.Activeflow
	}{
		{
			name: "normal",

			af: &activeflow.Activeflow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a4bbdcc4-f9e5-11ef-b4f8-c30c2a699e54"),
				},
				CurrentStackID: uuid.FromStringOrNil("a555f502-f9e5-11ef-bbbe-b363e26fda0f"),
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
			expectStackID: uuid.FromStringOrNil("a555f502-f9e5-11ef-bbbe-b363e26fda0f"),
			expectActiveflow: &activeflow.Activeflow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a4bbdcc4-f9e5-11ef-b4f8-c30c2a699e54"),
				},
				CurrentStackID: uuid.FromStringOrNil("a555f502-f9e5-11ef-bbbe-b363e26fda0f"),
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
				ForwardStackID:  stack.IDMain,
				ForwardActionID: uuid.FromStringOrNil("a5887874-f9e5-11ef-a64e-3f6c58935e86"),
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
			mockDB.EXPECT().ActiveflowUpdate(ctx, tt.expectActiveflow).Return(nil)

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
			mockDB.EXPECT().ActiveflowUpdate(ctx, tt.responseActiveflow).Return(nil)
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

		responseStack *stack.Stack

		expectStackID    uuid.UUID
		expectActiveflow *activeflow.Activeflow
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
			expectStackID: uuid.FromStringOrNil("0c3dbf8a-f9ea-11ef-87bb-0b801c31899c"),
			expectActiveflow: &activeflow.Activeflow{
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
				ForwardStackID:  stack.IDMain,
				ForwardActionID: uuid.FromStringOrNil("a5887874-f9e5-11ef-a64e-3f6c58935e86"),
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
			mockDB.EXPECT().ActiveflowUpdate(ctx, tt.expectActiveflow).Return(nil)

			if err := h.ServiceStop(ctx, tt.id, tt.serviceID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
