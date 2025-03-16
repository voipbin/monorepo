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
	"monorepo/bin-flow-manager/pkg/actionhandler"
	"monorepo/bin-flow-manager/pkg/dbhandler"
	"monorepo/bin-flow-manager/pkg/stackmaphandler"
	"monorepo/bin-flow-manager/pkg/variablehandler"
)

func Test_Create(t *testing.T) {

	tests := []struct {
		name string

		id           uuid.UUID
		refereceType activeflow.ReferenceType
		referenceID  uuid.UUID
		flowID       uuid.UUID

		responseFlow       *flow.Flow
		responseStackMap   map[uuid.UUID]*stack.Stack
		responseUUID       uuid.UUID
		responseActiveflow *activeflow.Activeflow

		expectActiveflow *activeflow.Activeflow
	}{
		{
			name: "normal",

			id:           uuid.FromStringOrNil("a58dc1e8-dc67-447b-9392-2d58531f1fb1"),
			refereceType: activeflow.ReferenceTypeCall,
			referenceID:  uuid.FromStringOrNil("03e8a480-822f-11eb-b71f-8bbc09fa1e7a"),
			flowID:       uuid.FromStringOrNil("dc8e048e-822e-11eb-8cb6-235002e45cf2"),

			responseFlow: &flow.Flow{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("dc8e048e-822e-11eb-8cb6-235002e45cf2"),
					CustomerID: uuid.FromStringOrNil("86e9108a-d699-11ec-8b3f-bf3d574b538b"),
				},
				Actions: []action.Action{
					{
						ID:   uuid.FromStringOrNil("770e7166-d692-11ec-b2e7-37e6f0fdd5ea"),
						Type: action.TypeAnswer,
					},
				},
			},
			responseStackMap: map[uuid.UUID]*stack.Stack{
				stack.IDMain: {
					ID: stack.IDMain,
					Actions: []action.Action{
						{
							ID:   uuid.FromStringOrNil("770e7166-d692-11ec-b2e7-37e6f0fdd5ea"),
							Type: action.TypeAnswer,
						},
					},
				},
			},
			responseActiveflow: &activeflow.Activeflow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a58dc1e8-dc67-447b-9392-2d58531f1fb1"),
				},
			},

			expectActiveflow: &activeflow.Activeflow{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a58dc1e8-dc67-447b-9392-2d58531f1fb1"),
					CustomerID: uuid.FromStringOrNil("86e9108a-d699-11ec-8b3f-bf3d574b538b"),
				},
				FlowID:        uuid.FromStringOrNil("dc8e048e-822e-11eb-8cb6-235002e45cf2"),
				Status:        activeflow.StatusRunning,
				ReferenceType: activeflow.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("03e8a480-822f-11eb-b71f-8bbc09fa1e7a"),

				CurrentStackID: stack.IDMain,
				CurrentAction: action.Action{
					ID: action.IDStart,
				},

				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:   uuid.FromStringOrNil("770e7166-d692-11ec-b2e7-37e6f0fdd5ea"),
								Type: action.TypeAnswer,
							},
						},
					},
				},

				ForwardStackID:  stack.IDEmpty,
				ForwardActionID: action.IDEmpty,

				ExecuteCount:    0,
				ExecutedActions: []action.Action{},
			},
		},
		{
			name: "id is empty",

			id:           uuid.Nil,
			refereceType: activeflow.ReferenceTypeCall,
			referenceID:  uuid.FromStringOrNil("d6543076-aba3-46c2-ac82-46101f294bf5"),
			flowID:       uuid.FromStringOrNil("dc8e048e-822e-11eb-8cb6-235002e45cf2"),

			responseFlow: &flow.Flow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("dc8e048e-822e-11eb-8cb6-235002e45cf2"),
				},
				Actions: []action.Action{},
			},
			responseStackMap: map[uuid.UUID]*stack.Stack{
				stack.IDMain: {
					ID:      stack.IDMain,
					Actions: []action.Action{},
				},
			},
			responseUUID: uuid.FromStringOrNil("5f0d58fe-c8cf-11ed-b23d-9b5ebf2aca94"),
			responseActiveflow: &activeflow.Activeflow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("78184d65-899f-438f-aeca-8cce4f445756"),
				},
				ReferenceType: activeflow.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("d6543076-aba3-46c2-ac82-46101f294bf5"),
				FlowID:        uuid.FromStringOrNil("dc8e048e-822e-11eb-8cb6-235002e45cf2"),
				CurrentAction: action.Action{
					ID: action.IDStart,
				},
				ExecuteCount:    0,
				ForwardActionID: action.IDEmpty,
				ExecutedActions: []action.Action{},
			},

			expectActiveflow: &activeflow.Activeflow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("5f0d58fe-c8cf-11ed-b23d-9b5ebf2aca94"),
				},
				FlowID:        uuid.FromStringOrNil("dc8e048e-822e-11eb-8cb6-235002e45cf2"),
				Status:        activeflow.StatusRunning,
				ReferenceType: activeflow.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("d6543076-aba3-46c2-ac82-46101f294bf5"),

				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID:             stack.IDMain,
						Actions:        []action.Action{},
						ReturnStackID:  stack.IDEmpty,
						ReturnActionID: action.IDEmpty,
					},
				},

				CurrentStackID: stack.IDMain,
				CurrentAction: action.Action{
					ID: action.IDStart,
				},

				ForwardStackID:  stack.IDEmpty,
				ForwardActionID: action.IDEmpty,

				ExecuteCount:    0,
				ExecutedActions: []action.Action{},
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
			mockVariableHandler := variablehandler.NewMockVariableHandler(mc)
			mockStack := stackmaphandler.NewMockStackmapHandler(mc)

			h := &activeflowHandler{
				utilHandler:     mockUtil,
				db:              mockDB,
				notifyHandler:   mockNotify,
				variableHandler: mockVariableHandler,
				stackmapHandler: mockStack,
			}

			ctx := context.Background()

			mockDB.EXPECT().FlowGet(ctx, tt.flowID).Return(tt.responseFlow, nil)
			if tt.id == uuid.Nil {
				mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			}

			mockStack.EXPECT().Create(tt.responseFlow.Actions).Return(tt.responseStackMap)
			mockDB.EXPECT().ActiveflowCreate(ctx, tt.expectActiveflow).Return(nil)
			mockVariableHandler.EXPECT().Create(ctx, tt.expectActiveflow.ID, map[string]string{}).Return(&variable.Variable{}, nil)
			mockDB.EXPECT().ActiveflowGet(ctx, tt.expectActiveflow.ID).Return(tt.responseActiveflow, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseActiveflow.CustomerID, activeflow.EventTypeActiveflowCreated, tt.responseActiveflow)

			res, err := h.Create(ctx, tt.id, tt.refereceType, tt.referenceID, tt.flowID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.responseActiveflow) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseActiveflow, res)
			}
		})
	}
}

func Test_updateCurrentAction(t *testing.T) {

	tests := []struct {
		name string

		activeflowID uuid.UUID
		stackID      uuid.UUID
		act          *action.Action

		responseActiveflow *activeflow.Activeflow
		responseDBCurTime  string

		expectActiveflowUpdate *activeflow.Activeflow
	}{
		{
			"normal",
			uuid.FromStringOrNil("f594ebd8-06ae-11eb-9bca-5757b3876041"),
			uuid.FromStringOrNil("e70a8fac-d4b4-11ec-adc8-bfe8cdd29a31"),
			&action.Action{
				ID:   uuid.FromStringOrNil("f916a6a2-06ae-11eb-a239-53802c6fbb36"),
				Type: action.TypeAnswer,
			},

			&activeflow.Activeflow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("f594ebd8-06ae-11eb-9bca-5757b3876041"),
				},
				ExecutedActions: []action.Action{},
				CurrentAction: action.Action{
					ID:   uuid.FromStringOrNil("b08981ee-d4ba-11ec-93bf-93a97d1a142f"),
					Type: action.TypeAnswer,
				},
			},
			"2022-04-18 03:22:17.995000",

			&activeflow.Activeflow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("f594ebd8-06ae-11eb-9bca-5757b3876041"),
				},
				ExecutedActions: []action.Action{
					{
						ID:   uuid.FromStringOrNil("b08981ee-d4ba-11ec-93bf-93a97d1a142f"),
						Type: action.TypeAnswer,
					},
				},
				CurrentStackID: uuid.FromStringOrNil("e70a8fac-d4b4-11ec-adc8-bfe8cdd29a31"),
				CurrentAction: action.Action{
					ID:   uuid.FromStringOrNil("f916a6a2-06ae-11eb-a239-53802c6fbb36"),
					Type: action.TypeAnswer,
				},
				ExecuteCount: 1,
				TMUpdate:     "2022-04-18 03:22:17.995000",
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

			h := &activeflowHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().ActiveflowGet(ctx, tt.activeflowID).Return(tt.responseActiveflow, nil)
			mockUtil.EXPECT().TimeGetCurTime().Return(utilhandler.TimeGetCurTime()).AnyTimes()
			mockDB.EXPECT().ActiveflowUpdate(ctx, tt.expectActiveflowUpdate).Return(nil)
			mockDB.EXPECT().ActiveflowGet(ctx, tt.activeflowID).Return(tt.responseActiveflow, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseActiveflow.CustomerID, activeflow.EventTypeActiveflowUpdated, tt.responseActiveflow)

			_, err := h.updateCurrentAction(ctx, tt.activeflowID, tt.stackID, tt.act)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_SetForwardActionID(t *testing.T) {

	tests := []struct {
		name string

		id         uuid.UUID
		actionID   uuid.UUID
		forwardNow bool

		responseStackID uuid.UUID
		responseAction  *action.Action

		af                     *activeflow.Activeflow
		expectUpdateActiveflow *activeflow.Activeflow
	}{
		{
			name: "reference type call forward now true",

			id:         uuid.FromStringOrNil("1bd514f0-af6c-11ec-bddc-db11051293e5"),
			actionID:   uuid.FromStringOrNil("1c998542-af6c-11ec-b385-c7a45742f1a1"),
			forwardNow: true,

			responseStackID: stack.IDMain,
			responseAction: &action.Action{
				ID:   uuid.FromStringOrNil("1c998542-af6c-11ec-b385-c7a45742f1a1"),
				Type: action.TypeAnswer,
			},

			af: &activeflow.Activeflow{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("1bd514f0-af6c-11ec-bddc-db11051293e5"),
					CustomerID: uuid.FromStringOrNil("fcc49e18-af6c-11ec-9857-8bc5d3558dc9"),
				},
				ReferenceType:  activeflow.ReferenceTypeCall,
				ReferenceID:    uuid.FromStringOrNil("fa923116-d67f-11ec-b2b7-83b4ced11267"),
				CurrentStackID: stack.IDMain,
				CurrentAction: action.Action{
					ID:   uuid.FromStringOrNil("1cc5a9e2-af6c-11ec-ad49-db6eee64a325"),
					Type: action.TypeAnswer,
				},

				ForwardStackID:  stack.IDEmpty,
				ForwardActionID: action.IDEmpty,

				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:   uuid.FromStringOrNil("1cc5a9e2-af6c-11ec-ad49-db6eee64a325"),
								Type: action.TypeAnswer,
							},
							{
								ID:   uuid.FromStringOrNil("1c998542-af6c-11ec-b385-c7a45742f1a1"),
								Type: action.TypeAnswer,
							},
						},
					},
				},
			},
			expectUpdateActiveflow: &activeflow.Activeflow{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("1bd514f0-af6c-11ec-bddc-db11051293e5"),
					CustomerID: uuid.FromStringOrNil("fcc49e18-af6c-11ec-9857-8bc5d3558dc9"),
				},
				ReferenceType:  activeflow.ReferenceTypeCall,
				ReferenceID:    uuid.FromStringOrNil("fa923116-d67f-11ec-b2b7-83b4ced11267"),
				CurrentStackID: stack.IDMain,
				CurrentAction: action.Action{
					ID:   uuid.FromStringOrNil("1cc5a9e2-af6c-11ec-ad49-db6eee64a325"),
					Type: action.TypeAnswer,
				},

				ForwardStackID:  stack.IDMain,
				ForwardActionID: uuid.FromStringOrNil("1c998542-af6c-11ec-b385-c7a45742f1a1"),
				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:   uuid.FromStringOrNil("1cc5a9e2-af6c-11ec-ad49-db6eee64a325"),
								Type: action.TypeAnswer,
							},
							{
								ID:   uuid.FromStringOrNil("1c998542-af6c-11ec-b385-c7a45742f1a1"),
								Type: action.TypeAnswer,
							},
						},
					},
				},
			},
		},
		{
			name: "reference type call forward now false",

			id:         uuid.FromStringOrNil("1bc62ef8-af6d-11ec-a2d2-d36eb561e845"),
			actionID:   uuid.FromStringOrNil("1c19a9f2-af6d-11ec-afe0-4bb7a2667649"),
			forwardNow: false,

			responseStackID: stack.IDMain,
			responseAction: &action.Action{
				ID:   uuid.FromStringOrNil("1c19a9f2-af6d-11ec-afe0-4bb7a2667649"),
				Type: action.TypeAnswer,
			},

			af: &activeflow.Activeflow{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("1bc62ef8-af6d-11ec-a2d2-d36eb561e845"),
					CustomerID: uuid.FromStringOrNil("fc989a84-af6c-11ec-8bb9-23ec42502bfa"),
				},
				CurrentStackID: stack.IDMain,
				CurrentAction: action.Action{
					ID:   uuid.FromStringOrNil("1bedaa5a-af6d-11ec-99f4-3b55921b1b50"),
					Type: action.TypeAnswer,
				},
				ForwardStackID:  stack.IDEmpty,
				ForwardActionID: action.IDEmpty,

				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:   uuid.FromStringOrNil("1bedaa5a-af6d-11ec-99f4-3b55921b1b50"),
								Type: action.TypeAnswer,
							},
							{
								ID:   uuid.FromStringOrNil("1c19a9f2-af6d-11ec-afe0-4bb7a2667649"),
								Type: action.TypeAnswer,
							},
						},
					},
				},
			},
			expectUpdateActiveflow: &activeflow.Activeflow{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("1bc62ef8-af6d-11ec-a2d2-d36eb561e845"),
					CustomerID: uuid.FromStringOrNil("fc989a84-af6c-11ec-8bb9-23ec42502bfa"),
				},
				CurrentStackID: stack.IDMain,
				CurrentAction: action.Action{
					ID:   uuid.FromStringOrNil("1bedaa5a-af6d-11ec-99f4-3b55921b1b50"),
					Type: action.TypeAnswer,
				},
				ForwardStackID:  stack.IDMain,
				ForwardActionID: uuid.FromStringOrNil("1c19a9f2-af6d-11ec-afe0-4bb7a2667649"),
				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:   uuid.FromStringOrNil("1bedaa5a-af6d-11ec-99f4-3b55921b1b50"),
								Type: action.TypeAnswer,
							},
							{
								ID:   uuid.FromStringOrNil("1c19a9f2-af6d-11ec-afe0-4bb7a2667649"),
								Type: action.TypeAnswer,
							},
						},
					},
				},
			},
		},
		{
			"reference type message forward now true",

			uuid.FromStringOrNil("91875644-af6d-11ec-bf11-5fa477b94be1"),
			uuid.FromStringOrNil("91d53d96-af6d-11ec-8b73-27eb3b54c06f"),
			true,

			stack.IDMain,
			&action.Action{
				ID:   uuid.FromStringOrNil("91d53d96-af6d-11ec-8b73-27eb3b54c06f"),
				Type: action.TypeAnswer,
			},

			&activeflow.Activeflow{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("91875644-af6d-11ec-bf11-5fa477b94be1"),
					CustomerID: uuid.FromStringOrNil("fc989a84-af6c-11ec-8bb9-23ec42502bfa"),
				},

				ReferenceType:  activeflow.ReferenceTypeMessage,
				CurrentStackID: stack.IDMain,
				CurrentAction: action.Action{
					ID:   uuid.FromStringOrNil("91b048a6-af6d-11ec-ba12-1f7793a35ea0"),
					Type: action.TypeAnswer,
				},
				ForwardStackID:  stack.IDEmpty,
				ForwardActionID: action.IDEmpty,

				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:   uuid.FromStringOrNil("91b048a6-af6d-11ec-ba12-1f7793a35ea0"),
								Type: action.TypeAnswer,
							},
							{
								ID:   uuid.FromStringOrNil("91d53d96-af6d-11ec-8b73-27eb3b54c06f"),
								Type: action.TypeAnswer,
							},
						},
					},
				},
			},
			&activeflow.Activeflow{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("91875644-af6d-11ec-bf11-5fa477b94be1"),
					CustomerID: uuid.FromStringOrNil("fc989a84-af6c-11ec-8bb9-23ec42502bfa"),
				},
				ReferenceType:  activeflow.ReferenceTypeMessage,
				CurrentStackID: stack.IDMain,
				CurrentAction: action.Action{
					ID:   uuid.FromStringOrNil("91b048a6-af6d-11ec-ba12-1f7793a35ea0"),
					Type: action.TypeAnswer,
				},
				ForwardStackID:  stack.IDMain,
				ForwardActionID: uuid.FromStringOrNil("91d53d96-af6d-11ec-8b73-27eb3b54c06f"),
				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:   uuid.FromStringOrNil("91b048a6-af6d-11ec-ba12-1f7793a35ea0"),
								Type: action.TypeAnswer,
							},
							{
								ID:   uuid.FromStringOrNil("91d53d96-af6d-11ec-8b73-27eb3b54c06f"),
								Type: action.TypeAnswer,
							},
						},
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
			mockAction := actionhandler.NewMockActionHandler(mc)
			mockStack := stackmaphandler.NewMockStackmapHandler(mc)

			h := &activeflowHandler{
				db:         mockDB,
				reqHandler: mockReq,

				actionHandler:   mockAction,
				stackmapHandler: mockStack,
			}

			ctx := context.Background()
			mockDB.EXPECT().ActiveflowGetWithLock(ctx, tt.id).Return(tt.af, nil)
			mockStack.EXPECT().GetAction(tt.af.StackMap, tt.af.CurrentStackID, tt.actionID, false).Return(tt.responseStackID, tt.responseAction, nil)
			mockDB.EXPECT().ActiveflowUpdate(ctx, tt.expectUpdateActiveflow).Return(nil)

			if tt.forwardNow && tt.af.ReferenceType == activeflow.ReferenceTypeCall {
				mockReq.EXPECT().CallV1CallActionNext(ctx, tt.af.ReferenceID, true).Return(nil)
			}

			mockDB.EXPECT().ActiveflowReleaseLock(ctx, tt.id).Return(nil)
			if err := h.SetForwardActionID(ctx, tt.id, tt.actionID, tt.forwardNow); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			time.Sleep(time.Microsecond * 100)
		})
	}
}

func Test_dbDelete(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseGet *activeflow.Activeflow

		expectedRes *activeflow.Activeflow
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("e25d0800-f81c-11ec-8bd9-2b2aa60686f5"),

			responseGet: &activeflow.Activeflow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("e25d0800-f81c-11ec-8bd9-2b2aa60686f5"),
				},
			},

			expectedRes: &activeflow.Activeflow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("e25d0800-f81c-11ec-8bd9-2b2aa60686f5"),
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

			h := &activeflowHandler{
				db:              mockDB,
				notifyHandler:   mockNotify,
				variableHandler: mockVariableHandler,
			}

			ctx := context.Background()

			mockDB.EXPECT().ActiveflowDelete(ctx, tt.id).Return(nil)
			mockDB.EXPECT().ActiveflowGet(ctx, tt.id).Return(tt.responseGet, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseGet.CustomerID, activeflow.EventTypeActiveflowDeleted, tt.responseGet)

			res, err := h.dbDelete(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectedRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, res)
			}
		})
	}
}

func Test_Gets(t *testing.T) {

	tests := []struct {
		name string

		token   string
		size    uint64
		filters map[string]string

		responseGet []*activeflow.Activeflow
	}{
		{
			"test normal",

			"2020-10-10T03:30:17.000000",
			10,
			map[string]string{
				"customer_id": "e3bb9832-f81d-11ec-bcd9-9f298317c9f9",
				"deleted":     "false",
			},

			[]*activeflow.Activeflow{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("7a8224b6-f81e-11ec-99b1-476bd41ee6d0"),
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
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockVariableHandler := variablehandler.NewMockVariableHandler(mc)

			h := &activeflowHandler{
				db:              mockDB,
				notifyHandler:   mockNotify,
				variableHandler: mockVariableHandler,
			}
			ctx := context.Background()

			mockDB.EXPECT().ActiveflowGets(ctx, tt.token, tt.size, tt.filters).Return(tt.responseGet, nil)

			res, err := h.Gets(ctx, tt.token, tt.size, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.responseGet) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseGet, res)
			}

		})
	}
}

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
