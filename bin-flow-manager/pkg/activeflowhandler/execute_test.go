package activeflowhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-flow-manager/models/action"
	"monorepo/bin-flow-manager/models/activeflow"
	"monorepo/bin-flow-manager/models/stack"
	"monorepo/bin-flow-manager/models/variable"
	"monorepo/bin-flow-manager/pkg/actionhandler"
	"monorepo/bin-flow-manager/pkg/dbhandler"
	"monorepo/bin-flow-manager/pkg/stackmaphandler"
	"monorepo/bin-flow-manager/pkg/variablehandler"
)

func Test_Execute(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseActiveflow *activeflow.Activeflow
		responseStackID    uuid.UUID
		responseAction     *action.Action
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("bef23280-a7ab-11ec-8e79-1b236556e34d"),

			responseActiveflow: &activeflow.Activeflow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("bef23280-a7ab-11ec-8e79-1b236556e34d"),
				},
				CurrentAction: action.Action{
					ID: action.IDStart,
				},
				ReferenceType: activeflow.ReferenceTypeCall,
				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:   uuid.FromStringOrNil("13c4e65e-a7ac-11ec-971e-0374e19101d3"),
								Type: action.TypeAnswer,
							},
						},
					},
				},
			},
			responseStackID: stack.IDMain,
			responseAction: &action.Action{
				ID:   uuid.FromStringOrNil("13c4e65e-a7ac-11ec-971e-0374e19101d3"),
				Type: action.TypeAnswer,
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
			mockStack := stackmaphandler.NewMockStackmapHandler(mc)
			mockVar := variablehandler.NewMockVariableHandler(mc)

			h := &activeflowHandler{
				utilHandler:     mockUtil,
				db:              mockDB,
				notifyHandler:   mockNotify,
				stackmapHandler: mockStack,
				variableHandler: mockVar,
			}

			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime().Return(utilhandler.TimeGetCurTime()).AnyTimes()

			// updateNextAction
			mockDB.EXPECT().ActiveflowGetWithLock(gomock.Any(), tt.id).Return(tt.responseActiveflow, nil)
			mockStack.EXPECT().GetNextAction(gomock.Any(), gomock.Any(), gomock.Any(), true).Return(tt.responseStackID, tt.responseAction)
			mockVar.EXPECT().Get(ctx, tt.id).Return(&variable.Variable{}, nil)
			mockVar.EXPECT().SubstituteOption(ctx, tt.responseAction.Option, &variable.Variable{})
			mockDB.EXPECT().ActiveflowReleaseLock(ctx, tt.id)

			// executeAction
			mockDB.EXPECT().ActiveflowGet(ctx, tt.id).Return(tt.responseActiveflow, nil)
			mockDB.EXPECT().ActiveflowUpdate(ctx, tt.id, gomock.Any()).Return(nil)
			mockDB.EXPECT().ActiveflowGet(ctx, tt.id).Return(tt.responseActiveflow, nil)
			mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), tt.responseActiveflow.CustomerID, activeflow.EventTypeActiveflowUpdated, tt.responseActiveflow)

			if err := h.Execute(ctx, tt.id); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_ExecuteNextAction(t *testing.T) {

	tests := []struct {
		name string

		id       uuid.UUID
		actionID uuid.UUID

		responseActiveflow *activeflow.Activeflow
		responseStackID    uuid.UUID
		responseAction     *action.Action
	}{
		{
			name: "normal",

			id:       uuid.FromStringOrNil("0d276266-0737-11eb-808f-8f2856d44e29"),
			actionID: uuid.FromStringOrNil("05e2c40a-0737-11eb-9134-5f9b578a4179"),

			responseActiveflow: &activeflow.Activeflow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("0d276266-0737-11eb-808f-8f2856d44e29"),
				},
				CurrentAction: action.Action{
					ID:   uuid.FromStringOrNil("05e2c40a-0737-11eb-9134-5f9b578a4179"),
					Type: action.TypeAnswer,
				},
				ReferenceType:   activeflow.ReferenceTypeCall,
				ForwardActionID: action.IDEmpty,
				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:   uuid.FromStringOrNil("05e2c40a-0737-11eb-9134-5f9b578a4179"),
								Type: action.TypeAnswer,
							},
							{
								ID:   uuid.FromStringOrNil("c9fffcf4-0737-11eb-a28f-2bc0bae5eeaf"),
								Type: action.TypeAnswer,
							},
						},
					},
				},
			},
			responseStackID: stack.IDMain,
			responseAction: &action.Action{
				ID:   uuid.FromStringOrNil("04bda23e-d4c7-11ec-a8a4-9ffff59826c6"),
				Type: action.TypeAnswer,
			},
		},
		{
			name: "current id start",

			id:       uuid.FromStringOrNil("950c810c-08a4-11eb-af93-93115c7f9c55"),
			actionID: action.IDStart,

			responseActiveflow: &activeflow.Activeflow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("950c810c-08a4-11eb-af93-93115c7f9c55"),
				},
				CurrentAction: action.Action{
					ID: action.IDStart,
				},
				ReferenceType:   activeflow.ReferenceTypeCall,
				ForwardActionID: action.IDEmpty,

				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:   uuid.FromStringOrNil("97f96f9c-08a4-11eb-8ea0-57d38a96eca3"),
								Type: action.TypeAnswer,
							},
							{
								ID:   uuid.FromStringOrNil("a9b365ee-08a4-11eb-87c5-e7b9e9ea9de3"),
								Type: action.TypeAnswer,
							},
						},
					},
				},
			},
			responseStackID: stack.IDMain,
			responseAction: &action.Action{
				ID:   uuid.FromStringOrNil("97f96f9c-08a4-11eb-8ea0-57d38a96eca3"),
				Type: action.TypeAnswer,
			},
		},
		{
			name: "forward action id has set",

			id:       uuid.FromStringOrNil("6ed30c30-794c-11ec-98dc-237ea83d2fcb"),
			actionID: uuid.FromStringOrNil("bf5e3b10-5733-11ec-a0c6-879d0d048e2d"),

			responseActiveflow: &activeflow.Activeflow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("6ed30c30-794c-11ec-98dc-237ea83d2fcb"),
				},
				CurrentAction: action.Action{
					ID: uuid.FromStringOrNil("bf5e3b10-5733-11ec-a0c6-879d0d048e2d"),
				},
				ReferenceType:   activeflow.ReferenceTypeCall,
				ForwardStackID:  uuid.FromStringOrNil("fbc9fe2a-d4da-11ec-8488-ffd76437ad90"),
				ForwardActionID: uuid.FromStringOrNil("ab88bd9a-5733-11ec-9fa5-df017a802cfc"),

				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:   uuid.FromStringOrNil("97f96f9c-08a4-11eb-8ea0-57d38a96eca3"),
								Type: action.TypeAnswer,
							},
							{
								ID:   uuid.FromStringOrNil("ab88bd9a-5733-11ec-9fa5-df017a802cfc"),
								Type: action.TypeAnswer,
							},
							{
								ID:   uuid.FromStringOrNil("bf5e3b10-5733-11ec-a0c6-879d0d048e2d"),
								Type: action.TypeAnswer,
							},
							{
								ID:   uuid.FromStringOrNil("bfec567a-5733-11ec-846c-efcfc0955605"),
								Type: action.TypeAnswer,
							},
						},
					},
				},
			},
			responseStackID: stack.IDMain,
			responseAction: &action.Action{
				ID:   uuid.FromStringOrNil("ab88bd9a-5733-11ec-9fa5-df017a802cfc"),
				Type: action.TypeAnswer,
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
			mockAction := actionhandler.NewMockActionHandler(mc)
			mockStack := stackmaphandler.NewMockStackmapHandler(mc)
			mockVar := variablehandler.NewMockVariableHandler(mc)

			h := &activeflowHandler{
				utilHandler:     mockUtil,
				db:              mockDB,
				notifyHandler:   mockNotify,
				actionHandler:   mockAction,
				stackmapHandler: mockStack,
				variableHandler: mockVar,
			}

			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime().Return(utilhandler.TimeGetCurTime()).AnyTimes()

			// getNextAction
			mockDB.EXPECT().ActiveflowGetWithLock(gomock.Any(), tt.id).Return(tt.responseActiveflow, nil)
			if tt.responseActiveflow.ForwardStackID != stack.IDEmpty && tt.responseActiveflow.ForwardActionID != action.IDEmpty {
				mockStack.EXPECT().GetAction(tt.responseActiveflow.StackMap, tt.responseActiveflow.ForwardStackID, tt.responseActiveflow.ForwardActionID, true).Return(tt.responseStackID, tt.responseAction, nil)
			} else {
				mockStack.EXPECT().GetNextAction(tt.responseActiveflow.StackMap, tt.responseActiveflow.CurrentStackID, &tt.responseActiveflow.CurrentAction, true).Return(tt.responseStackID, tt.responseAction)
			}

			mockVar.EXPECT().Get(ctx, tt.id).Return(&variable.Variable{}, nil)
			mockVar.EXPECT().SubstituteOption(ctx, tt.responseAction.Option, &variable.Variable{})

			// executeAction
			mockDB.EXPECT().ActiveflowGet(ctx, tt.id).Return(tt.responseActiveflow, nil)
			mockDB.EXPECT().ActiveflowUpdate(ctx, tt.id, gomock.Any()).Return(nil)
			mockDB.EXPECT().ActiveflowGet(ctx, gomock.Any()).Return(tt.responseActiveflow, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, gomock.Any(), activeflow.EventTypeActiveflowUpdated, gomock.Any())

			mockDB.EXPECT().ActiveflowReleaseLock(ctx, tt.id)

			res, err := h.ExecuteNextAction(ctx, tt.id, tt.actionID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.responseAction) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.responseAction, res)
			}
		})
	}
}

func Test_ExecuteNextActionError(t *testing.T) {

	tests := []struct {
		name string

		id       uuid.UUID
		actionID uuid.UUID

		responseActiveflow *activeflow.Activeflow
	}{
		{
			name: "stackhandler's GetNextAction returns error",

			id:       uuid.FromStringOrNil("085f48fc-08a4-11eb-8ef3-675e25cbc25c"),
			actionID: action.IDStart,

			responseActiveflow: &activeflow.Activeflow{
				CurrentAction: action.Action{
					ID: action.IDStart,
				},
				ForwardActionID: action.IDEmpty,
				StackMap:        map[uuid.UUID]*stack.Stack{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockAction := actionhandler.NewMockActionHandler(mc)
			mockStack := stackmaphandler.NewMockStackmapHandler(mc)

			h := &activeflowHandler{
				db:              mockDB,
				notifyHandler:   mockNotify,
				actionHandler:   mockAction,
				stackmapHandler: mockStack,
			}
			ctx := context.Background()

			mockDB.EXPECT().ActiveflowGetWithLock(gomock.Any(), tt.id).Return(nil, fmt.Errorf(""))

			mockDB.EXPECT().ActiveflowGet(ctx, tt.id).Return(tt.responseActiveflow, nil)
			mockDB.EXPECT().ActiveflowUpdate(ctx, tt.id, map[activeflow.Field]any{activeflow.FieldStatus: activeflow.StatusEnded}).Return(nil)
			mockDB.EXPECT().ActiveflowGet(ctx, tt.id).Return(tt.responseActiveflow, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, gomock.Any(), activeflow.EventTypeActiveflowUpdated, gomock.Any())

			_, err := h.ExecuteNextAction(ctx, tt.id, tt.actionID)
			if err == nil {
				t.Error("Wrong match. expect: err, got: ok")
			}
		})
	}
}

func Test_executeAction(t *testing.T) {

	tests := []struct {
		name string

		activeflow *activeflow.Activeflow

		expectedRes *action.Action
	}{
		{
			name: "normal",

			activeflow: &activeflow.Activeflow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("f01970ee-f49f-11ec-a545-8bd387ee59d4"),
				},
				CurrentAction: action.Action{
					ID:   uuid.FromStringOrNil("00b40040-f4a0-11ec-844f-bf9b5ac7bc7a"),
					Type: action.TypeAnswer,
				},
				ReferenceType: activeflow.ReferenceTypeCall,
			},

			expectedRes: &action.Action{
				ID:   uuid.FromStringOrNil("00b40040-f4a0-11ec-844f-bf9b5ac7bc7a"),
				Type: action.TypeAnswer,
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
			mockStack := stackmaphandler.NewMockStackmapHandler(mc)
			mockVar := variablehandler.NewMockVariableHandler(mc)

			h := &activeflowHandler{
				utilHandler:     mockUtil,
				db:              mockDB,
				notifyHandler:   mockNotify,
				stackmapHandler: mockStack,
				variableHandler: mockVar,
			}
			ctx := context.Background()

			res, err := h.executeAction(ctx, tt.activeflow)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectedRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, res)
			}
		})
	}
}
