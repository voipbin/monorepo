package activeflowhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/activeflow"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/stack"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/variable"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/actionhandler"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/stackhandler"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/variablehandler"
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
			"normal",

			uuid.FromStringOrNil("bef23280-a7ab-11ec-8e79-1b236556e34d"),

			&activeflow.Activeflow{
				ID: uuid.FromStringOrNil("bef23280-a7ab-11ec-8e79-1b236556e34d"),
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
			stack.IDMain,
			&action.Action{
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
			mockStack := stackhandler.NewMockStackHandler(mc)
			mockVar := variablehandler.NewMockVariableHandler(mc)

			h := &activeflowHandler{
				utilHandler:     mockUtil,
				db:              mockDB,
				notifyHandler:   mockNotify,
				stackHandler:    mockStack,
				variableHandler: mockVar,
			}

			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime().Return(utilhandler.TimeGetCurTime()).AnyTimes()

			// getNextAction
			mockDB.EXPECT().ActiveflowGetWithLock(gomock.Any(), tt.id).Return(tt.responseActiveflow, nil)
			mockStack.EXPECT().GetNextAction(ctx, gomock.Any(), gomock.Any(), gomock.Any(), true).Return(tt.responseStackID, tt.responseAction)
			mockVar.EXPECT().Get(ctx, tt.id).Return(&variable.Variable{}, nil)
			mockVar.EXPECT().SubstituteByte(ctx, tt.responseAction.Option, &variable.Variable{}).Return(tt.responseAction.Option)
			mockDB.EXPECT().ActiveflowReleaseLock(ctx, tt.id)

			// executeAction
			mockDB.EXPECT().ActiveflowGet(ctx, tt.id).Return(tt.responseActiveflow, nil)
			mockDB.EXPECT().ActiveflowUpdate(ctx, gomock.Any()).Return(nil)
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

		id                 uuid.UUID
		actionID           uuid.UUID
		responseActiveflow *activeflow.Activeflow

		responseStackID uuid.UUID
		responseAction  *action.Action
	}{
		{
			"normal",

			uuid.FromStringOrNil("0d276266-0737-11eb-808f-8f2856d44e29"),
			uuid.FromStringOrNil("05e2c40a-0737-11eb-9134-5f9b578a4179"),
			&activeflow.Activeflow{
				ID: uuid.FromStringOrNil("0d276266-0737-11eb-808f-8f2856d44e29"),
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

			stack.IDMain,
			&action.Action{
				ID:   uuid.FromStringOrNil("04bda23e-d4c7-11ec-a8a4-9ffff59826c6"),
				Type: action.TypeAnswer,
			},
		},
		{
			"current id start",

			uuid.FromStringOrNil("950c810c-08a4-11eb-af93-93115c7f9c55"),
			action.IDStart,
			&activeflow.Activeflow{
				ID: uuid.FromStringOrNil("950c810c-08a4-11eb-af93-93115c7f9c55"),
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

			stack.IDMain,
			&action.Action{
				ID:   uuid.FromStringOrNil("97f96f9c-08a4-11eb-8ea0-57d38a96eca3"),
				Type: action.TypeAnswer,
			},
		},
		{
			"forward action id has set",

			uuid.FromStringOrNil("6ed30c30-794c-11ec-98dc-237ea83d2fcb"),
			uuid.FromStringOrNil("bf5e3b10-5733-11ec-a0c6-879d0d048e2d"),
			&activeflow.Activeflow{
				ID: uuid.FromStringOrNil("6ed30c30-794c-11ec-98dc-237ea83d2fcb"),
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

			stack.IDMain,
			&action.Action{
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
			mockStack := stackhandler.NewMockStackHandler(mc)
			mockVar := variablehandler.NewMockVariableHandler(mc)

			h := &activeflowHandler{
				utilHandler:     mockUtil,
				db:              mockDB,
				notifyHandler:   mockNotify,
				actionHandler:   mockAction,
				stackHandler:    mockStack,
				variableHandler: mockVar,
			}

			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime().Return(utilhandler.TimeGetCurTime()).AnyTimes()

			// getNextAction
			mockDB.EXPECT().ActiveflowGetWithLock(gomock.Any(), tt.id).Return(tt.responseActiveflow, nil)
			if tt.responseActiveflow.ForwardStackID != stack.IDEmpty && tt.responseActiveflow.ForwardActionID != action.IDEmpty {
				mockStack.EXPECT().GetAction(ctx, tt.responseActiveflow.StackMap, tt.responseActiveflow.ForwardStackID, tt.responseActiveflow.ForwardActionID, true).Return(tt.responseStackID, tt.responseAction, nil)
			} else {
				mockStack.EXPECT().GetNextAction(ctx, tt.responseActiveflow.StackMap, tt.responseActiveflow.CurrentStackID, &tt.responseActiveflow.CurrentAction, true).Return(tt.responseStackID, tt.responseAction)
			}

			mockVar.EXPECT().Get(ctx, tt.id).Return(&variable.Variable{}, nil)
			mockVar.EXPECT().SubstituteByte(ctx, tt.responseAction.Option, &variable.Variable{}).Return(tt.responseAction.Option)

			// executeAction
			mockDB.EXPECT().ActiveflowGet(ctx, tt.id).Return(tt.responseActiveflow, nil)
			mockDB.EXPECT().ActiveflowUpdate(ctx, gomock.Any()).Return(nil)
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
		name     string
		id       uuid.UUID
		actionID uuid.UUID

		responseActiveflow *activeflow.Activeflow
	}{
		{
			"stackhandler's GetNextAction returns error",
			uuid.FromStringOrNil("085f48fc-08a4-11eb-8ef3-675e25cbc25c"),
			action.IDStart,

			&activeflow.Activeflow{
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
			mockStack := stackhandler.NewMockStackHandler(mc)

			h := &activeflowHandler{
				db:            mockDB,
				notifyHandler: mockNotify,
				actionHandler: mockAction,
				stackHandler:  mockStack,
			}
			ctx := context.Background()

			mockDB.EXPECT().ActiveflowGetWithLock(gomock.Any(), tt.id).Return(nil, fmt.Errorf(""))

			mockDB.EXPECT().ActiveflowGet(ctx, tt.id).Return(tt.responseActiveflow, nil)
			mockDB.EXPECT().ActiveflowSetStatus(ctx, tt.id, activeflow.StatusEnded).Return(nil)
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

		expectRes *action.Action
	}{
		{
			"normal",

			&activeflow.Activeflow{
				ID: uuid.FromStringOrNil("f01970ee-f49f-11ec-a545-8bd387ee59d4"),
				CurrentAction: action.Action{
					ID:   uuid.FromStringOrNil("00b40040-f4a0-11ec-844f-bf9b5ac7bc7a"),
					Type: action.TypeAnswer,
				},
				ReferenceType: activeflow.ReferenceTypeCall,
			},

			&action.Action{
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
			mockStack := stackhandler.NewMockStackHandler(mc)
			mockVar := variablehandler.NewMockVariableHandler(mc)

			h := &activeflowHandler{
				utilHandler:     mockUtil,
				db:              mockDB,
				notifyHandler:   mockNotify,
				stackHandler:    mockStack,
				variableHandler: mockVar,
			}
			ctx := context.Background()

			res, err := h.executeAction(ctx, tt.activeflow)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
