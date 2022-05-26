package activeflowhandler

import (
	"context"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	amagentdial "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agentdial"
	cmaddress "gitlab.com/voipbin/bin-manager/call-manager.git/models/address"
	cmcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	cfconference "gitlab.com/voipbin/bin-manager/conference-manager.git/models/conference"
	mmmessage "gitlab.com/voipbin/bin-manager/message-manager.git/models/message"
	qmqueue "gitlab.com/voipbin/bin-manager/queue-manager.git/models/queue"
	qmqueuecall "gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecall"
	tstranscribe "gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcribe"
	"gitlab.com/voipbin/bin-manager/webhook-manager.git/models/webhook"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/activeflow"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/flow"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/stack"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/variable"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/actionhandler"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/stackhandler"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/variablehandler"
)

func Test_actionHandleConnect(t *testing.T) {

	tests := []struct {
		name string

		af *activeflow.Activeflow

		responseConference *cfconference.Conference
		responseFlow       *flow.Flow

		expectCallSource       *cmaddress.Address
		expectCallDestinations []cmaddress.Address
		expectCallMasterCallID uuid.UUID

		responsePushStackID uuid.UUID
		responsePushAction  *action.Action

		expectUpdateActiveflow *activeflow.Activeflow

		expectFlowCreateActions []action.Action
	}{
		{
			name: "single destination",

			af: &activeflow.Activeflow{
				CustomerID:  uuid.FromStringOrNil("8220d086-7f48-11ec-a1fd-a35a08ad282c"),
				ReferenceID: uuid.FromStringOrNil("e1a258ca-0a98-11eb-8e3b-e7d2a18277fa"),
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("f4a4a87e-0a98-11eb-8f96-cba83b8b3f76"),
					Type:   action.TypeConnect,
					Option: []byte(`{"source":{"type": "tel", "target": "+123456789"}, "destinations": [{"type": "tel", "name": "", "target": "+987654321"}]}`),
				},
				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:     uuid.FromStringOrNil("f4a4a87e-0a98-11eb-8f96-cba83b8b3f76"),
								Type:   action.TypeConnect,
								Option: []byte(`{"source":{"type": "tel", "target": "+123456789"}, "destinations": [{"type": "tel", "name": "", "target": "+987654321"}]}`),
							},
						},
					},
				},
			},

			responseConference: &cfconference.Conference{
				ID:         uuid.FromStringOrNil("363b4ae8-0a9b-11eb-9d08-436d6934a451"),
				CustomerID: uuid.FromStringOrNil("8220d086-7f48-11ec-a1fd-a35a08ad282c"),
			},
			responseFlow: &flow.Flow{
				ID:         uuid.FromStringOrNil("fa26f0ce-0a9b-11eb-8850-afda1bb6bc03"),
				CustomerID: uuid.FromStringOrNil("8220d086-7f48-11ec-a1fd-a35a08ad282c"),
			},

			expectCallSource: &cmaddress.Address{
				Type:   cmaddress.TypeTel,
				Target: "+123456789",
			},
			expectCallDestinations: []cmaddress.Address{
				{
					Type:   cmaddress.TypeTel,
					Target: "+987654321",
				},
			},
			expectCallMasterCallID: uuid.FromStringOrNil("e1a258ca-0a98-11eb-8e3b-e7d2a18277fa"),

			responsePushStackID: uuid.FromStringOrNil("6ba8ba2c-d4bf-11ec-bb34-1f6a8e0bf102"),
			responsePushAction: &action.Action{
				ID:   uuid.FromStringOrNil("7b764a6e-d4bf-11ec-8f93-279c9970f53e"),
				Type: action.TypeConferenceJoin,
			},

			expectUpdateActiveflow: &activeflow.Activeflow{
				CustomerID:      uuid.FromStringOrNil("8220d086-7f48-11ec-a1fd-a35a08ad282c"),
				ReferenceID:     uuid.FromStringOrNil("e1a258ca-0a98-11eb-8e3b-e7d2a18277fa"),
				ForwardStackID:  uuid.FromStringOrNil("6ba8ba2c-d4bf-11ec-bb34-1f6a8e0bf102"),
				ForwardActionID: uuid.FromStringOrNil("7b764a6e-d4bf-11ec-8f93-279c9970f53e"),

				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("f4a4a87e-0a98-11eb-8f96-cba83b8b3f76"),
					Type:   action.TypeConnect,
					Option: []byte(`{"source":{"type": "tel", "target": "+123456789"}, "destinations": [{"type": "tel", "name": "", "target": "+987654321"}]}`),
				},
				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:     uuid.FromStringOrNil("f4a4a87e-0a98-11eb-8f96-cba83b8b3f76"),
								Type:   action.TypeConnect,
								Option: []byte(`{"source":{"type": "tel", "target": "+123456789"}, "destinations": [{"type": "tel", "name": "", "target": "+987654321"}]}`),
							},
						},
					},
				},
			},

			expectFlowCreateActions: []action.Action{
				{
					Type:   action.TypeConferenceJoin,
					Option: []byte(`{"conference_id":"363b4ae8-0a9b-11eb-9d08-436d6934a451"}`),
				},
			},
		},
		{
			name: "multiple destinations",

			af: &activeflow.Activeflow{
				CustomerID:     uuid.FromStringOrNil("a356975a-8055-11ec-9c11-37c0ba53de51"),
				ReferenceID:    uuid.FromStringOrNil("cb4accf8-2710-11eb-8e49-e73409394bef"),
				CurrentStackID: stack.IDMain,
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("cbe12fa4-2710-11eb-8959-87391e4bbc77"),
					Type:   action.TypeConnect,
					Option: []byte(`{"source":{"type": "tel", "target": "+123456789"}, "destinations": [{"type": "tel", "name": "", "target": "+987654321"}, {"type": "tel", "name": "", "target": "+9876543210"}]}`),
				},

				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:     uuid.FromStringOrNil("cbe12fa4-2710-11eb-8959-87391e4bbc77"),
								Type:   action.TypeConnect,
								Option: []byte(`{"source":{"type": "tel", "target": "+123456789"}, "destinations": [{"type": "tel", "name": "", "target": "+987654321"}, {"type": "tel", "name": "", "target": "+9876543210"}]}`),
							},
						},
					},
				},
			},

			responseConference: &cfconference.Conference{
				ID:         uuid.FromStringOrNil("cc131f96-2710-11eb-b3b2-1b43dc6ffa2f"),
				CustomerID: uuid.FromStringOrNil("a356975a-8055-11ec-9c11-37c0ba53de51"),
			},
			responseFlow: &flow.Flow{
				ID:         uuid.FromStringOrNil("cc480ff8-2710-11eb-8869-0fcf3d58fd6a"),
				CustomerID: uuid.FromStringOrNil("a356975a-8055-11ec-9c11-37c0ba53de51"),
			},

			expectCallSource: &cmaddress.Address{
				Type:   cmaddress.TypeTel,
				Target: "+123456789",
			},
			expectCallDestinations: []cmaddress.Address{
				{
					Type:   cmaddress.TypeTel,
					Target: "+987654321",
				},
				{
					Type:   cmaddress.TypeTel,
					Target: "+9876543210",
				},
			},
			expectCallMasterCallID: uuid.FromStringOrNil("cb4accf8-2710-11eb-8e49-e73409394bef"),

			responsePushStackID: uuid.FromStringOrNil("73af2dfc-d4c2-11ec-a692-9b1eafe93075"),
			responsePushAction: &action.Action{
				ID:   uuid.FromStringOrNil("845b566c-d4c2-11ec-ba4e-f739bb357410"),
				Type: action.TypeConferenceJoin,
			},

			expectUpdateActiveflow: &activeflow.Activeflow{
				CustomerID:      uuid.FromStringOrNil("a356975a-8055-11ec-9c11-37c0ba53de51"),
				ReferenceID:     uuid.FromStringOrNil("cb4accf8-2710-11eb-8e49-e73409394bef"),
				ForwardStackID:  uuid.FromStringOrNil("73af2dfc-d4c2-11ec-a692-9b1eafe93075"),
				ForwardActionID: uuid.FromStringOrNil("845b566c-d4c2-11ec-ba4e-f739bb357410"),
				CurrentStackID:  stack.IDMain,
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("cbe12fa4-2710-11eb-8959-87391e4bbc77"),
					Type:   action.TypeConnect,
					Option: []byte(`{"source":{"type": "tel", "target": "+123456789"}, "destinations": [{"type": "tel", "name": "", "target": "+987654321"}, {"type": "tel", "name": "", "target": "+9876543210"}]}`),
				},

				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:     uuid.FromStringOrNil("cbe12fa4-2710-11eb-8959-87391e4bbc77"),
								Type:   action.TypeConnect,
								Option: []byte(`{"source":{"type": "tel", "target": "+123456789"}, "destinations": [{"type": "tel", "name": "", "target": "+987654321"}, {"type": "tel", "name": "", "target": "+9876543210"}]}`),
							},
						},
					},
				},
			},

			expectFlowCreateActions: []action.Action{
				{
					Type:   action.TypeConferenceJoin,
					Option: []byte(`{"conference_id":"cc131f96-2710-11eb-b3b2-1b43dc6ffa2f"}`),
				},
			},
		},
		{
			name: "multiple unchained destinations",

			af: &activeflow.Activeflow{
				CustomerID:  uuid.FromStringOrNil("a356975a-8055-11ec-9c11-37c0ba53de51"),
				ReferenceID: uuid.FromStringOrNil("211a68fe-2712-11eb-ad71-97e2b1546a91"),
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("22311f94-2712-11eb-8550-0f0b066f8120"),
					Type:   action.TypeConnect,
					Option: []byte(`{"source":{"type": "tel", "target": "+123456789"}, "destinations": [{"type": "tel", "name": "", "target": "+987654321"}, {"type": "tel", "name": "", "target": "+9876543210"}], "unchained": true}`),
				},

				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:     uuid.FromStringOrNil("22311f94-2712-11eb-8550-0f0b066f8120"),
								Type:   action.TypeConnect,
								Option: []byte(`{"source":{"type": "tel", "target": "+123456789"}, "destinations": [{"type": "tel", "name": "", "target": "+987654321"}, {"type": "tel", "name": "", "target": "+9876543210"}], "unchained": true}`),
							},
						},
					},
				},
			},

			responseConference: &cfconference.Conference{
				ID:         uuid.FromStringOrNil("2266e688-2712-11eb-aab4-eb00b0a3efbe"),
				CustomerID: uuid.FromStringOrNil("a356975a-8055-11ec-9c11-37c0ba53de51"),
			},
			responseFlow: &flow.Flow{
				ID:         uuid.FromStringOrNil("229ef410-2712-11eb-9dea-a737f7b6ef2b"),
				CustomerID: uuid.FromStringOrNil("a356975a-8055-11ec-9c11-37c0ba53de51"),
			},

			expectCallSource: &cmaddress.Address{
				Type:   cmaddress.TypeTel,
				Target: "+123456789",
			},
			expectCallDestinations: []cmaddress.Address{
				{
					Type:   cmaddress.TypeTel,
					Target: "+987654321",
				},
				{
					Type:   cmaddress.TypeTel,
					Target: "+9876543210",
				},
			},
			expectCallMasterCallID: uuid.Nil,

			responsePushStackID: uuid.FromStringOrNil("d913dcf6-d4c2-11ec-902b-37f50ff7b4b4"),
			responsePushAction: &action.Action{
				ID:   uuid.FromStringOrNil("d96a09aa-d4c2-11ec-bcea-0bce8dd7e065"),
				Type: action.TypeConferenceJoin,
			},

			expectUpdateActiveflow: &activeflow.Activeflow{
				CustomerID:      uuid.FromStringOrNil("a356975a-8055-11ec-9c11-37c0ba53de51"),
				ReferenceID:     uuid.FromStringOrNil("211a68fe-2712-11eb-ad71-97e2b1546a91"),
				ForwardStackID:  uuid.FromStringOrNil("d913dcf6-d4c2-11ec-902b-37f50ff7b4b4"),
				ForwardActionID: uuid.FromStringOrNil("d96a09aa-d4c2-11ec-bcea-0bce8dd7e065"),
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("22311f94-2712-11eb-8550-0f0b066f8120"),
					Type:   action.TypeConnect,
					Option: []byte(`{"source":{"type": "tel", "target": "+123456789"}, "destinations": [{"type": "tel", "name": "", "target": "+987654321"}, {"type": "tel", "name": "", "target": "+9876543210"}], "unchained": true}`),
				},

				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:     uuid.FromStringOrNil("22311f94-2712-11eb-8550-0f0b066f8120"),
								Type:   action.TypeConnect,
								Option: []byte(`{"source":{"type": "tel", "target": "+123456789"}, "destinations": [{"type": "tel", "name": "", "target": "+987654321"}, {"type": "tel", "name": "", "target": "+9876543210"}], "unchained": true}`),
							},
						},
					},
				},
			},

			expectFlowCreateActions: []action.Action{
				{
					Type:   action.TypeConferenceJoin,
					Option: []byte(`{"conference_id":"2266e688-2712-11eb-aab4-eb00b0a3efbe"}`),
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
			mockStack := stackhandler.NewMockStackHandler(mc)

			h := &activeflowHandler{
				db:         mockDB,
				reqHandler: mockReq,

				actionHandler: mockAction,
				stackHandler:  mockStack,
			}

			ctx := context.Background()

			mockReq.EXPECT().CFV1ConferenceCreate(ctx, tt.af.CustomerID, cfconference.TypeConnect, "", "", 86400, nil, nil, nil).Return(tt.responseConference, nil)
			mockReq.EXPECT().FMV1FlowCreate(ctx, tt.af.CustomerID, flow.TypeFlow, "", "", tt.expectFlowCreateActions, false).Return(tt.responseFlow, nil)

			mockReq.EXPECT().CMV1CallsCreate(ctx, tt.responseFlow.CustomerID, tt.responseFlow.ID, tt.expectCallMasterCallID, tt.expectCallSource, tt.expectCallDestinations).Return([]cmcall.Call{{ID: uuid.Nil}}, nil)
			mockStack.EXPECT().Push(ctx, tt.af.StackMap, gomock.Any(), tt.af.CurrentStackID, tt.af.CurrentAction.ID).Return(tt.responsePushStackID, tt.responsePushAction, nil)
			mockDB.EXPECT().ActiveflowUpdate(gomock.Any(), tt.expectUpdateActiveflow).Return(nil)

			if err := h.actionHandleConnect(ctx, tt.af); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_getActionsFromFlow(t *testing.T) {

	tests := []struct {
		name   string
		flowID uuid.UUID
		flow   *flow.Flow
		callID uuid.UUID
	}{
		{
			"normal",
			uuid.FromStringOrNil("9091d6aa-3cbe-11ec-9a9e-7f0d954e1f7a"),
			&flow.Flow{
				ID: uuid.FromStringOrNil("9091d6aa-3cbe-11ec-9a9e-7f0d954e1f7a"),
			},
			uuid.FromStringOrNil("549d358a-fbfc-11ea-a625-43073fda56b9"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)

			h := &activeflowHandler{
				db:         mockDB,
				reqHandler: mockReq,
			}

			ctx := context.Background()

			tmpFlow := &flow.Flow{
				CustomerID: tt.flow.CustomerID,
			}
			mockReq.EXPECT().FMV1FlowGet(ctx, tt.flowID).Return(tmpFlow, nil)

			_, err := h.getActionsFromFlow(ctx, tt.flowID, tt.flow.CustomerID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_actionHandleGotoLoopContinue(t *testing.T) {

	tests := []struct {
		name string

		callID   uuid.UUID
		targetID uuid.UUID

		activeFlow       *activeflow.Activeflow
		updateActiveFlow *activeflow.Activeflow

		responseOrgStackID    uuid.UUID
		responseOrgAction     *action.Action
		responseTargetStackID uuid.UUID
		responseTargetAction  *action.Action
	}{
		{
			name:     "normal",
			callID:   uuid.FromStringOrNil("01f28ffc-0c08-11ec-8b28-0f1dd70b3428"),
			targetID: uuid.FromStringOrNil("7dbc6998-410d-11ec-91b8-d722b27bb799"),

			activeFlow: &activeflow.Activeflow{
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("2d099c6e-55a3-11ec-85b0-db3612865f6e"),
					Type:   action.TypeGoto,
					Option: []byte(`{"target_id":"7dbc6998-410d-11ec-91b8-d722b27bb799","loop_count":3}`),
				},
				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:   uuid.FromStringOrNil("7dbc6998-410d-11ec-91b8-d722b27bb799"),
								Type: action.TypeAnswer,
							},
							{
								ID:     uuid.FromStringOrNil("2d099c6e-55a3-11ec-85b0-db3612865f6e"),
								Type:   action.TypeGoto,
								Option: []byte(`{"target_id":"7dbc6998-410d-11ec-91b8-d722b27bb799","loop_count":3}`),
							},
						},
					},
				},
			},
			updateActiveFlow: &activeflow.Activeflow{
				ForwardStackID:  stack.IDMain,
				ForwardActionID: uuid.FromStringOrNil("7dbc6998-410d-11ec-91b8-d722b27bb799"),
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("2d099c6e-55a3-11ec-85b0-db3612865f6e"),
					Type:   action.TypeGoto,
					Option: []byte(`{"target_id":"7dbc6998-410d-11ec-91b8-d722b27bb799","loop_count":3}`),
				},
				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:   uuid.FromStringOrNil("7dbc6998-410d-11ec-91b8-d722b27bb799"),
								Type: action.TypeAnswer,
							},
							{
								ID:     uuid.FromStringOrNil("2d099c6e-55a3-11ec-85b0-db3612865f6e"),
								Type:   action.TypeGoto,
								Option: []byte(`{"target_id":"7dbc6998-410d-11ec-91b8-d722b27bb799","loop_count":3}`),
							},
						},
					},
				},
			},

			responseOrgStackID: stack.IDMain,
			responseOrgAction: &action.Action{
				ID:     uuid.FromStringOrNil("2d099c6e-55a3-11ec-85b0-db3612865f6e"),
				Type:   action.TypeGoto,
				Option: []byte(`{"target_id":"7dbc6998-410d-11ec-91b8-d722b27bb799","loop_count":3}`),
			},
			responseTargetStackID: stack.IDMain,
			responseTargetAction: &action.Action{
				ID:   uuid.FromStringOrNil("7dbc6998-410d-11ec-91b8-d722b27bb799"),
				Type: action.TypeAnswer,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockStack := stackhandler.NewMockStackHandler(mc)

			h := &activeflowHandler{
				db:           mockDB,
				reqHandler:   mockReq,
				stackHandler: mockStack,
			}

			ctx := context.Background()

			mockStack.EXPECT().GetAction(ctx, tt.activeFlow.StackMap, tt.activeFlow.CurrentStackID, tt.activeFlow.CurrentAction.ID, false).Return(tt.responseOrgStackID, tt.responseOrgAction, nil)
			mockStack.EXPECT().GetAction(ctx, tt.activeFlow.StackMap, tt.activeFlow.CurrentStackID, tt.targetID, false).Return(tt.responseTargetStackID, tt.responseTargetAction, nil)
			mockDB.EXPECT().ActiveflowUpdate(ctx, tt.updateActiveFlow).Return(nil)

			if err := h.actionHandleGoto(ctx, tt.activeFlow); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_actionHandleGotoLoopOver(t *testing.T) {

	tests := []struct {
		name string

		callID uuid.UUID

		activeFlow *activeflow.Activeflow

		expectActiveFlow *activeflow.Activeflow
	}{
		{
			"normal",
			uuid.FromStringOrNil("01f28ffc-0c08-11ec-8b28-0f1dd70b3428"),
			&activeflow.Activeflow{
				CurrentStackID: stack.IDMain,
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("2d099c6e-55a3-11ec-85b0-db3612865f6e"),
					Type:   action.TypeGoto,
					Option: []byte(`{"target_id":"7dbc6998-410d-11ec-91b8-d722b27bb799","loop_count":0}`),
				},

				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:   uuid.FromStringOrNil("7dbc6998-410d-11ec-91b8-d722b27bb799"),
								Type: action.TypeAnswer,
							},
							{
								ID:     uuid.FromStringOrNil("2d099c6e-55a3-11ec-85b0-db3612865f6e"),
								Type:   action.TypeGoto,
								Option: []byte(`{"target_id":"7dbc6998-410d-11ec-91b8-d722b27bb799","loop_count":0}`),
							},
							{
								ID:   uuid.FromStringOrNil("c299daf0-984c-11ec-9288-0b50517b314d"),
								Type: action.TypeAnswer,
							},
						},
					},
				},
			},

			&activeflow.Activeflow{
				CurrentStackID: stack.IDMain,
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("2d099c6e-55a3-11ec-85b0-db3612865f6e"),
					Type:   action.TypeGoto,
					Option: []byte(`{"target_id":"7dbc6998-410d-11ec-91b8-d722b27bb799"}`),
				},

				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:   uuid.FromStringOrNil("7dbc6998-410d-11ec-91b8-d722b27bb799"),
								Type: action.TypeAnswer,
							},
							{
								ID:     uuid.FromStringOrNil("2d099c6e-55a3-11ec-85b0-db3612865f6e"),
								Type:   action.TypeGoto,
								Option: []byte(`{"target_id":"7dbc6998-410d-11ec-91b8-d722b27bb799","loop_count":0}`),
							},
							{
								ID:   uuid.FromStringOrNil("c299daf0-984c-11ec-9288-0b50517b314d"),
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

			h := &activeflowHandler{
				db:         mockDB,
				reqHandler: mockReq,
			}

			ctx := context.Background()

			if err := h.actionHandleGoto(ctx, tt.activeFlow); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

		})
	}
}

func Test_actionHandleQueueJoin(t *testing.T) {
	tests := []struct {
		name string

		act *action.Action

		responseExitStackID uuid.UUID
		responseExitAction  *action.Action

		activeflow *activeflow.Activeflow
		queue      *qmqueue.Queue
		queueFlow  *flow.Flow

		expectActiveFlow  *activeflow.Activeflow
		responseQueuecall *qmqueuecall.Queuecall

		responseStackID uuid.UUID
		responseAction  *action.Action
	}{
		{
			name: "normal",

			act: &action.Action{
				ID:     uuid.FromStringOrNil("bf1f9cb4-6590-11ec-8502-ffcab16cf0d1"),
				Type:   action.TypeQueueJoin,
				Option: []byte(`{"queue_id": "bf45ea2c-6590-11ec-9a8c-ff92b7ef9aad"}`),
			},

			activeflow: &activeflow.Activeflow{
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("bf1f9cb4-6590-11ec-8502-ffcab16cf0d1"),
					Type:   action.TypeQueueJoin,
					Option: []byte(`{"queue_id": "bf45ea2c-6590-11ec-9a8c-ff92b7ef9aad"}`),
				},
				ReferenceID: uuid.FromStringOrNil("3de1fb7a-adfb-11ec-8765-9bb130635c87"),
				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:     uuid.FromStringOrNil("bf1f9cb4-6590-11ec-8502-ffcab16cf0d1"),
								Type:   action.TypeQueueJoin,
								Option: []byte(`{"queue_id": "bf45ea2c-6590-11ec-9a8c-ff92b7ef9aad"}`),
							},
							{
								ID:   uuid.FromStringOrNil("cdd46f0e-6591-11ec-aff5-63bb1f2f2e5f"),
								Type: action.TypeTalk,
							},
						},
					},
				},
			},
			queue: &qmqueue.Queue{
				ID: uuid.FromStringOrNil("bf45ea2c-6590-11ec-9a8c-ff92b7ef9aad"),
			},
			queueFlow: &flow.Flow{
				ID: uuid.FromStringOrNil("0f0a4864-6591-11ec-bc0e-db27e08ddec2"),
				Actions: []action.Action{
					{
						ID:   uuid.FromStringOrNil("5de173bc-6592-11ec-bd97-bfe78cdda0f5"),
						Type: action.TypeAnswer,
					},
					{
						ID:   uuid.FromStringOrNil("5e0bb1c2-6592-11ec-ad88-63adb38da11e"),
						Type: action.TypeConfbridgeJoin,
					},
				},
			},

			responseExitStackID: stack.IDMain,
			responseExitAction: &action.Action{
				ID:   uuid.FromStringOrNil("cdd46f0e-6591-11ec-aff5-63bb1f2f2e5f"),
				Type: action.TypeTalk,
			},

			expectActiveFlow: &activeflow.Activeflow{
				ForwardStackID:  uuid.FromStringOrNil("86f1de00-d4ea-11ec-80fc-dfbceef939e8"),
				ForwardActionID: uuid.FromStringOrNil("5de173bc-6592-11ec-bd97-bfe78cdda0f5"),
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("bf1f9cb4-6590-11ec-8502-ffcab16cf0d1"),
					Type:   action.TypeQueueJoin,
					Option: []byte(`{"queue_id": "bf45ea2c-6590-11ec-9a8c-ff92b7ef9aad"}`),
				},
				ReferenceID: uuid.FromStringOrNil("3de1fb7a-adfb-11ec-8765-9bb130635c87"),
				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:     uuid.FromStringOrNil("bf1f9cb4-6590-11ec-8502-ffcab16cf0d1"),
								Type:   action.TypeQueueJoin,
								Option: []byte(`{"queue_id": "bf45ea2c-6590-11ec-9a8c-ff92b7ef9aad"}`),
							},
							{
								ID:   uuid.FromStringOrNil("cdd46f0e-6591-11ec-aff5-63bb1f2f2e5f"),
								Type: action.TypeTalk,
							},
						},
					},
				},
			},
			responseQueuecall: &qmqueuecall.Queuecall{
				ID:     uuid.FromStringOrNil("c9002972-6592-11ec-af59-afccad96c5a4"),
				FlowID: uuid.FromStringOrNil("0f0a4864-6591-11ec-bc0e-db27e08ddec2"),
			},

			responseStackID: uuid.FromStringOrNil("86f1de00-d4ea-11ec-80fc-dfbceef939e8"),
			responseAction: &action.Action{
				ID:   uuid.FromStringOrNil("5de173bc-6592-11ec-bd97-bfe78cdda0f5"),
				Type: action.TypeAnswer,
			},
		},
		{
			name: "timeout wait",

			act: &action.Action{
				ID:     uuid.FromStringOrNil("d25ebcc6-7691-11ec-a4ed-8f4cf715eb08"),
				Type:   action.TypeQueueJoin,
				Option: []byte(`{"queue_id": "d28cb860-7691-11ec-b24f-a31daa9b0585"}`),
			},

			activeflow: &activeflow.Activeflow{
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("d25ebcc6-7691-11ec-a4ed-8f4cf715eb08"),
					Type:   action.TypeQueueJoin,
					Option: []byte(`{"queue_id": "d28cb860-7691-11ec-b24f-a31daa9b0585"}`),
				},
				ReferenceID: uuid.FromStringOrNil("9bd98a0e-adfb-11ec-8fa1-4b1e5a5964a7"),

				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:     uuid.FromStringOrNil("d25ebcc6-7691-11ec-a4ed-8f4cf715eb08"),
								Type:   action.TypeQueueJoin,
								Option: []byte(`{"queue_id": "d28cb860-7691-11ec-b24f-a31daa9b0585"}`),
							},
							{
								ID:   uuid.FromStringOrNil("d2b8883c-7691-11ec-a001-075712b96511"),
								Type: action.TypeTalk,
							},
						},
					},
				},
			},
			queue: &qmqueue.Queue{
				ID:          uuid.FromStringOrNil("d28cb860-7691-11ec-b24f-a31daa9b0585"),
				WaitTimeout: 600000,
			},
			queueFlow: &flow.Flow{
				ID: uuid.FromStringOrNil("d2e1b810-7691-11ec-b63f-a7af3ca6f888"),
				Actions: []action.Action{
					{
						ID:   uuid.FromStringOrNil("1d9b0492-7692-11ec-96dc-c3f3ba1b6fae"),
						Type: action.TypeAnswer,
					},
					{
						ID:   uuid.FromStringOrNil("5e0bb1c2-6592-11ec-ad88-63adb38da11e"),
						Type: action.TypeConfbridgeJoin,
					},
				},
			},

			responseExitStackID: stack.IDMain,
			responseExitAction: &action.Action{
				ID:   uuid.FromStringOrNil("d2b8883c-7691-11ec-a001-075712b96511"),
				Type: action.TypeTalk,
			},

			expectActiveFlow: &activeflow.Activeflow{
				ForwardStackID:  uuid.FromStringOrNil("077711e2-d4f2-11ec-8d55-f306cf094cd4"),
				ForwardActionID: uuid.FromStringOrNil("1d9b0492-7692-11ec-96dc-c3f3ba1b6fae"),
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("d25ebcc6-7691-11ec-a4ed-8f4cf715eb08"),
					Type:   action.TypeQueueJoin,
					Option: []byte(`{"queue_id": "d28cb860-7691-11ec-b24f-a31daa9b0585"}`),
				},
				ReferenceID: uuid.FromStringOrNil("9bd98a0e-adfb-11ec-8fa1-4b1e5a5964a7"),
				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:     uuid.FromStringOrNil("d25ebcc6-7691-11ec-a4ed-8f4cf715eb08"),
								Type:   action.TypeQueueJoin,
								Option: []byte(`{"queue_id": "d28cb860-7691-11ec-b24f-a31daa9b0585"}`),
							},
							{
								ID:   uuid.FromStringOrNil("d2b8883c-7691-11ec-a001-075712b96511"),
								Type: action.TypeTalk,
							},
						},
					},
				},
			},

			responseQueuecall: &qmqueuecall.Queuecall{
				ID:     uuid.FromStringOrNil("c9002972-6592-11ec-af59-afccad96c5a4"),
				FlowID: uuid.FromStringOrNil("d2e1b810-7691-11ec-b63f-a7af3ca6f888"),
			},

			responseStackID: uuid.FromStringOrNil("077711e2-d4f2-11ec-8d55-f306cf094cd4"),
			responseAction: &action.Action{
				ID:   uuid.FromStringOrNil("1d9b0492-7692-11ec-96dc-c3f3ba1b6fae"),
				Type: action.TypeAnswer,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockStack := stackhandler.NewMockStackHandler(mc)

			h := &activeflowHandler{
				db:           mockDB,
				reqHandler:   mockReq,
				stackHandler: mockStack,
			}

			ctx := context.Background()
			mockStack.EXPECT().GetNextAction(ctx, tt.activeflow.StackMap, tt.activeflow.CurrentStackID, &tt.activeflow.CurrentAction, false).Return(tt.responseExitStackID, tt.responseExitAction)

			mockReq.EXPECT().QMV1QueueCreateQueuecall(ctx, tt.queue.ID, gomock.Any(), tt.activeflow.ReferenceID, tt.activeflow.ID, tt.responseExitAction.ID).Return(tt.responseQueuecall, nil)
			mockReq.EXPECT().FMV1FlowGet(ctx, tt.responseQueuecall.FlowID).Return(tt.queueFlow, nil)

			mockStack.EXPECT().Push(ctx, tt.activeflow.StackMap, tt.queueFlow.Actions, tt.activeflow.CurrentStackID, tt.activeflow.CurrentAction.ID).Return(tt.responseStackID, tt.responseAction, nil)

			mockDB.EXPECT().ActiveflowUpdate(ctx, tt.expectActiveFlow).Return(nil)
			mockReq.EXPECT().QMV1QueuecallUpdateStatusWaiting(ctx, tt.responseQueuecall.ID).Return(tt.responseQueuecall, nil)

			if err := h.actionHandleQueueJoin(ctx, tt.activeflow); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_actionHandleFetch(t *testing.T) {

	tests := []struct {
		name string

		activeFlow *activeflow.Activeflow

		responseFetch []action.Action

		responseStackID uuid.UUID
		responseAction  *action.Action

		expectUpdateActiveflow *activeflow.Activeflow
	}{
		{
			name: "normal",

			activeFlow: &activeflow.Activeflow{
				ID:          uuid.FromStringOrNil("de10062c-d4df-11ec-bd42-a76fe4d96b2f"),
				ReferenceID: uuid.FromStringOrNil("d79ad434-d4df-11ec-8edf-0f15868a6578"),
				CurrentAction: action.Action{
					ID:   uuid.FromStringOrNil("c1c76320-d4df-11ec-acdf-8304c3ca8c1f"),
					Type: action.TypeFetch,
				},
				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:   uuid.FromStringOrNil("f0b5605e-648e-11ec-b318-a7f267cc71fc"),
								Type: action.TypeAnswer,
							},
							{
								ID:   uuid.FromStringOrNil("c1c76320-d4df-11ec-acdf-8304c3ca8c1f"),
								Type: action.TypeFetch,
							},
							{
								ID:   uuid.FromStringOrNil("ad108d6a-648e-11ec-a226-536bc1253066"),
								Type: action.TypeTalk,
							},
						},
					},
				},
			},

			responseFetch: []action.Action{
				{
					ID:   uuid.FromStringOrNil("0dc5e10c-d4e0-11ec-8dd0-a326b2d87c71"),
					Type: action.TypeAnswer,
				},
			},
			responseStackID: uuid.FromStringOrNil("5d18b072-d4e0-11ec-a4ab-1fcd7ec4f258"),
			responseAction: &action.Action{
				ID:   uuid.FromStringOrNil("0dc5e10c-d4e0-11ec-8dd0-a326b2d87c71"),
				Type: action.TypeAnswer,
			},

			expectUpdateActiveflow: &activeflow.Activeflow{
				ID:              uuid.FromStringOrNil("de10062c-d4df-11ec-bd42-a76fe4d96b2f"),
				ForwardStackID:  uuid.FromStringOrNil("5d18b072-d4e0-11ec-a4ab-1fcd7ec4f258"),
				ForwardActionID: uuid.FromStringOrNil("0dc5e10c-d4e0-11ec-8dd0-a326b2d87c71"),
				ReferenceID:     uuid.FromStringOrNil("d79ad434-d4df-11ec-8edf-0f15868a6578"),
				CurrentAction: action.Action{
					ID:   uuid.FromStringOrNil("c1c76320-d4df-11ec-acdf-8304c3ca8c1f"),
					Type: action.TypeFetch,
				},
				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:   uuid.FromStringOrNil("f0b5605e-648e-11ec-b318-a7f267cc71fc"),
								Type: action.TypeAnswer,
							},
							{
								ID:   uuid.FromStringOrNil("c1c76320-d4df-11ec-acdf-8304c3ca8c1f"),
								Type: action.TypeFetch,
							},
							{
								ID:   uuid.FromStringOrNil("ad108d6a-648e-11ec-a226-536bc1253066"),
								Type: action.TypeTalk,
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
			mockStack := stackhandler.NewMockStackHandler(mc)

			h := &activeflowHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				actionHandler: mockAction,
				stackHandler:  mockStack,
			}

			ctx := context.Background()

			mockAction.EXPECT().ActionFetchGet(&tt.activeFlow.CurrentAction, tt.activeFlow.ID, tt.activeFlow.ReferenceID).Return(tt.responseFetch, nil)
			mockStack.EXPECT().Push(ctx, tt.activeFlow.StackMap, tt.responseFetch, tt.activeFlow.CurrentStackID, tt.activeFlow.CurrentAction.ID).Return(tt.responseStackID, tt.responseAction, nil)
			mockDB.EXPECT().ActiveflowUpdate(ctx, tt.expectUpdateActiveflow).Return(nil)

			if err := h.actionHandleFetch(ctx, tt.activeFlow); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_actionHandleFetchFlow(t *testing.T) {

	tests := []struct {
		name string

		flowID     uuid.UUID
		activeflow *activeflow.Activeflow

		responseflow           *flow.Flow
		expectUpdateActiveflow *activeflow.Activeflow

		responseStackID uuid.UUID
		responseAction  *action.Action
	}{
		{
			name: "normal",

			flowID: uuid.FromStringOrNil("a1d247b4-3cbf-11ec-8d08-970ce7001aaa"),
			activeflow: &activeflow.Activeflow{
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("ec99431a-3cbf-11ec-b530-b3c665dd8156"),
					Type:   action.TypeFetchFlow,
					Option: []byte(`{"flow_id": "a1d247b4-3cbf-11ec-8d08-970ce7001aaa"}`),
				},
				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:   uuid.FromStringOrNil("f0b5605e-648e-11ec-b318-a7f267cc71fc"),
								Type: action.TypeAnswer,
							},
							{
								ID:   uuid.FromStringOrNil("ec99431a-3cbf-11ec-b530-b3c665dd8156"),
								Type: action.TypeFetchFlow,
							},
							{
								ID:   uuid.FromStringOrNil("ad108d6a-648e-11ec-a226-536bc1253066"),
								Type: action.TypeTalk,
							},
						},
					},
				},
			},

			responseflow: &flow.Flow{
				ID: uuid.FromStringOrNil("a1d247b4-3cbf-11ec-8d08-970ce7001aaa"),
				Actions: []action.Action{
					{
						ID:   uuid.FromStringOrNil("e2af181a-648e-11ec-878b-2bb6c0cebb3e"),
						Type: action.TypeAMD,
					},
				},
			},
			expectUpdateActiveflow: &activeflow.Activeflow{
				ForwardStackID:  uuid.FromStringOrNil("ede4083a-d4e1-11ec-917f-7f730832f0d0"),
				ForwardActionID: uuid.FromStringOrNil("e2af181a-648e-11ec-878b-2bb6c0cebb3e"),
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("ec99431a-3cbf-11ec-b530-b3c665dd8156"),
					Type:   action.TypeFetchFlow,
					Option: []byte(`{"flow_id": "a1d247b4-3cbf-11ec-8d08-970ce7001aaa"}`),
				},
				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:   uuid.FromStringOrNil("f0b5605e-648e-11ec-b318-a7f267cc71fc"),
								Type: action.TypeAnswer,
							},
							{
								ID:   uuid.FromStringOrNil("ec99431a-3cbf-11ec-b530-b3c665dd8156"),
								Type: action.TypeFetchFlow,
							},
							{
								ID:   uuid.FromStringOrNil("ad108d6a-648e-11ec-a226-536bc1253066"),
								Type: action.TypeTalk,
							},
						},
					},
				},
			},

			responseStackID: uuid.FromStringOrNil("ede4083a-d4e1-11ec-917f-7f730832f0d0"),
			responseAction: &action.Action{
				ID:   uuid.FromStringOrNil("e2af181a-648e-11ec-878b-2bb6c0cebb3e"),
				Type: action.TypeAMD,
			},
		},
		{
			name: "replace flow has 2 actions",

			flowID: uuid.FromStringOrNil("36e14dae-648f-11ec-b947-6f91a363d29e"),
			activeflow: &activeflow.Activeflow{
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("36679982-648f-11ec-b604-63e47c25e1e7"),
					Type:   action.TypeFetchFlow,
					Option: []byte(`{"flow_id": "36e14dae-648f-11ec-b947-6f91a363d29e"}`),
				},
				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:   uuid.FromStringOrNil("36900886-648f-11ec-88c7-5bc937041ab5"),
								Type: action.TypeAnswer,
							},
							{
								ID:   uuid.FromStringOrNil("36679982-648f-11ec-b604-63e47c25e1e7"),
								Type: action.TypeFetchFlow,
							},
							{
								ID:   uuid.FromStringOrNil("36ba131a-648f-11ec-8a6b-830a37358fbe"),
								Type: action.TypeTalk,
							},
						},
					},
				},
			},

			responseflow: &flow.Flow{
				ID: uuid.FromStringOrNil("36e14dae-648f-11ec-b947-6f91a363d29e"),
				Actions: []action.Action{
					{
						ID:   uuid.FromStringOrNil("59b5a226-648f-11ec-a356-ff8a386afbb9"),
						Type: action.TypeAMD,
					},
					{
						ID:   uuid.FromStringOrNil("59e09512-648f-11ec-bcec-438ee13c4be1"),
						Type: action.TypeTalk,
					},
				},
			},

			expectUpdateActiveflow: &activeflow.Activeflow{
				ForwardStackID:  uuid.FromStringOrNil("b0a90640-d4e2-11ec-ac01-878f8d902c0b"),
				ForwardActionID: uuid.FromStringOrNil("59b5a226-648f-11ec-a356-ff8a386afbb9"),
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("36679982-648f-11ec-b604-63e47c25e1e7"),
					Type:   action.TypeFetchFlow,
					Option: []byte(`{"flow_id": "36e14dae-648f-11ec-b947-6f91a363d29e"}`),
				},
				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:   uuid.FromStringOrNil("36900886-648f-11ec-88c7-5bc937041ab5"),
								Type: action.TypeAnswer,
							},
							{
								ID:   uuid.FromStringOrNil("36679982-648f-11ec-b604-63e47c25e1e7"),
								Type: action.TypeFetchFlow,
							},
							{
								ID:   uuid.FromStringOrNil("36ba131a-648f-11ec-8a6b-830a37358fbe"),
								Type: action.TypeTalk,
							},
						},
					},
				},
			},

			responseStackID: uuid.FromStringOrNil("b0a90640-d4e2-11ec-ac01-878f8d902c0b"),
			responseAction: &action.Action{
				ID:   uuid.FromStringOrNil("59b5a226-648f-11ec-a356-ff8a386afbb9"),
				Type: action.TypeAMD,
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
			mockStack := stackhandler.NewMockStackHandler(mc)

			h := &activeflowHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				actionHandler: mockAction,
				stackHandler:  mockStack,
			}

			ctx := context.Background()

			mockReq.EXPECT().FMV1FlowGet(ctx, tt.flowID).Return(tt.responseflow, nil)

			mockStack.EXPECT().Push(ctx, tt.activeflow.StackMap, tt.responseflow.Actions, tt.activeflow.CurrentStackID, tt.activeflow.CurrentAction.ID).Return(tt.responseStackID, tt.responseAction, nil)
			mockDB.EXPECT().ActiveflowUpdate(ctx, tt.expectUpdateActiveflow).Return(nil)

			if err := h.actionHandleFetchFlow(ctx, tt.activeflow); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_actionHandleConferenceJoin(t *testing.T) {

	tests := []struct {
		name string

		activeFlow         *activeflow.Activeflow
		responseConference *cfconference.Conference
		responseFlow       *flow.Flow

		responseStackID uuid.UUID
		responseAction  *action.Action

		expectActiveFlow *activeflow.Activeflow
	}{
		{
			name: "normal",
			activeFlow: &activeflow.Activeflow{
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("7dbc6998-410d-11ec-91b8-d722b27bb799"),
					Type:   action.TypeConferenceJoin,
					Option: []byte(`{"conference_id": "b7c84d66-410b-11ec-ab21-23726c7dc3b9"}`),
				},
				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:     uuid.FromStringOrNil("7dbc6998-410d-11ec-91b8-d722b27bb799"),
								Type:   action.TypeConferenceJoin,
								Option: []byte(`{"conference_id": "b7c84d66-410b-11ec-ab21-23726c7dc3b9"}`),
							},
						},
					},
				},
			},
			responseConference: &cfconference.Conference{
				ID:     uuid.FromStringOrNil("b7c84d66-410b-11ec-ab21-23726c7dc3b9"),
				FlowID: uuid.FromStringOrNil("b7eb3420-410b-11ec-ad87-cf5b4e34b7ed"),
				Status: cfconference.StatusProgressing,
			},
			responseFlow: &flow.Flow{
				ID: uuid.FromStringOrNil("b7eb3420-410b-11ec-ad87-cf5b4e34b7ed"),
				Actions: []action.Action{
					{
						ID:   uuid.FromStringOrNil("c74b311c-410c-11ec-84ac-1759f56d04b5"),
						Type: action.TypeAnswer,
					},
					{
						ID:   uuid.FromStringOrNil("c76c25d4-410c-11ec-9e97-e34e56d4cc4e"),
						Type: action.TypeConfbridgeJoin,
					},
					{
						ID:   uuid.FromStringOrNil("c785c6b0-410c-11ec-bd9f-5f698d905eef"),
						Type: action.TypeHangup,
					},
				},
			},

			responseStackID: uuid.FromStringOrNil("fd6d9b84-d4e3-11ec-a53b-879007c0bc0a"),
			responseAction: &action.Action{
				ID:   uuid.FromStringOrNil("c74b311c-410c-11ec-84ac-1759f56d04b5"),
				Type: action.TypeAnswer,
			},

			expectActiveFlow: &activeflow.Activeflow{
				ForwardStackID:  uuid.FromStringOrNil("fd6d9b84-d4e3-11ec-a53b-879007c0bc0a"),
				ForwardActionID: uuid.FromStringOrNil("c74b311c-410c-11ec-84ac-1759f56d04b5"),
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("7dbc6998-410d-11ec-91b8-d722b27bb799"),
					Type:   action.TypeConferenceJoin,
					Option: []byte(`{"conference_id": "b7c84d66-410b-11ec-ab21-23726c7dc3b9"}`),
				},
				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:     uuid.FromStringOrNil("7dbc6998-410d-11ec-91b8-d722b27bb799"),
								Type:   action.TypeConferenceJoin,
								Option: []byte(`{"conference_id": "b7c84d66-410b-11ec-ab21-23726c7dc3b9"}`),
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
			mockStack := stackhandler.NewMockStackHandler(mc)

			h := &activeflowHandler{
				db:           mockDB,
				reqHandler:   mockReq,
				stackHandler: mockStack,
			}

			ctx := context.Background()

			mockReq.EXPECT().CFV1ConferenceGet(ctx, tt.responseConference.ID).Return(tt.responseConference, nil)
			mockReq.EXPECT().FMV1FlowGet(ctx, tt.responseConference.FlowID).Return(tt.responseFlow, nil)
			mockStack.EXPECT().Push(ctx, tt.activeFlow.StackMap, tt.responseFlow.Actions, tt.activeFlow.CurrentStackID, tt.activeFlow.CurrentAction.ID).Return(tt.responseStackID, tt.responseAction, nil)
			mockDB.EXPECT().ActiveflowUpdate(gomock.Any(), tt.expectActiveFlow).Return(nil)

			if err := h.actionHandleConferenceJoin(ctx, tt.activeFlow); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_actionHandleAgentCall(t *testing.T) {

	tests := []struct {
		name string

		callID  uuid.UUID
		act     *action.Action
		agentID uuid.UUID

		activeFlow         *activeflow.Activeflow
		responseConference *cfconference.Conference
		call               *cmcall.Call

		expectReqActions []action.Action
		resoponseFlow    *flow.Flow

		rsponsePushStackID uuid.UUID
		responsePushAction *action.Action

		expectUpdateActiveflow *activeflow.Activeflow
	}{
		{
			name: "normal",

			callID: uuid.FromStringOrNil("71418cbe-53fc-11ec-980a-8fc233c3e802"),
			act: &action.Action{
				ID:     uuid.FromStringOrNil("716f309c-53fc-11ec-bff3-df8c8ffa945f"),
				Type:   action.TypeAgentCall,
				Option: []byte(`{"agent_id": "89593b12-53fc-11ec-9747-f7e71c3a8660"}`),
			},
			agentID: uuid.FromStringOrNil("89593b12-53fc-11ec-9747-f7e71c3a8660"),

			activeFlow: &activeflow.Activeflow{
				ReferenceID: uuid.FromStringOrNil("71418cbe-53fc-11ec-980a-8fc233c3e802"),
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("716f309c-53fc-11ec-bff3-df8c8ffa945f"),
					Type:   action.TypeAgentCall,
					Option: []byte(`{"agent_id": "89593b12-53fc-11ec-9747-f7e71c3a8660"}`),
				},

				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:   uuid.FromStringOrNil("716f309c-53fc-11ec-bff3-df8c8ffa945f"),
								Type: action.TypeAgentCall,
							},
						},
					},
				},
			},

			responseConference: &cfconference.Conference{
				ID:           uuid.FromStringOrNil("b7c84d66-410b-11ec-ab21-23726c7dc3b9"),
				FlowID:       uuid.FromStringOrNil("b7eb3420-410b-11ec-ad87-cf5b4e34b7ed"),
				ConfbridgeID: uuid.FromStringOrNil("9e60e850-53fe-11ec-a557-d7a7cce806ba"),
				Status:       cfconference.StatusProgressing,
			},
			call: &cmcall.Call{
				ID: uuid.FromStringOrNil("edee9f1c-53fd-11ec-a387-cb7cbdc7d345"),
				Source: cmaddress.Address{
					Type:   cmaddress.TypeTel,
					Target: "+821021656521",
				},
			},

			expectReqActions: []action.Action{
				{
					Type:   action.TypeConfbridgeJoin,
					Option: []byte(`{"confbridge_id":"9e60e850-53fe-11ec-a557-d7a7cce806ba"}`),
				},
			},
			resoponseFlow: &flow.Flow{
				ID: uuid.FromStringOrNil("7cff1888-8ca4-11ec-afb9-8b0839e726e5"),
			},

			rsponsePushStackID: uuid.FromStringOrNil("7ec62300-d4d4-11ec-a07a-9767aa4372af"),
			responsePushAction: &action.Action{
				ID:     uuid.FromStringOrNil("7f1d2664-d4d4-11ec-90ed-d381da69c089"),
				Type:   action.TypeConfbridgeJoin,
				Option: []byte(`{"confbridge_id":"9e60e850-53fe-11ec-a557-d7a7cce806ba"}`),
			},

			expectUpdateActiveflow: &activeflow.Activeflow{
				ReferenceID:     uuid.FromStringOrNil("71418cbe-53fc-11ec-980a-8fc233c3e802"),
				ForwardStackID:  uuid.FromStringOrNil("7ec62300-d4d4-11ec-a07a-9767aa4372af"),
				ForwardActionID: uuid.FromStringOrNil("7f1d2664-d4d4-11ec-90ed-d381da69c089"),
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("716f309c-53fc-11ec-bff3-df8c8ffa945f"),
					Type:   action.TypeAgentCall,
					Option: []byte(`{"agent_id": "89593b12-53fc-11ec-9747-f7e71c3a8660"}`),
				},
				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:   uuid.FromStringOrNil("716f309c-53fc-11ec-bff3-df8c8ffa945f"),
								Type: action.TypeAgentCall,
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
			mockStack := stackhandler.NewMockStackHandler(mc)

			h := &activeflowHandler{
				db:         mockDB,
				reqHandler: mockReq,

				actionHandler: mockAction,
				stackHandler:  mockStack,
			}

			ctx := context.Background()

			mockReq.EXPECT().CFV1ConferenceCreate(ctx, tt.activeFlow.CustomerID, cfconference.TypeConnect, "", "", 86400, nil, nil, nil).Return(tt.responseConference, nil)
			mockReq.EXPECT().CMV1CallGet(ctx, tt.callID).Return(tt.call, nil)

			mockReq.EXPECT().FMV1FlowCreate(ctx, tt.activeFlow.CustomerID, flow.TypeFlow, gomock.Any(), "", tt.expectReqActions, false).Return(tt.resoponseFlow, nil)
			mockReq.EXPECT().AMV1AgentDial(ctx, tt.agentID, &tt.call.Source, tt.resoponseFlow.ID, tt.callID).Return(&amagentdial.AgentDial{}, nil)

			mockStack.EXPECT().Push(ctx, tt.activeFlow.StackMap, gomock.Any(), tt.activeFlow.CurrentStackID, tt.activeFlow.CurrentAction.ID).Return(tt.rsponsePushStackID, tt.responsePushAction, nil)

			mockDB.EXPECT().ActiveflowUpdate(ctx, tt.expectUpdateActiveflow).Return(nil)

			if err := h.actionHandleAgentCall(ctx, tt.activeFlow); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_actionHandleBranch(t *testing.T) {

	tests := []struct {
		name string

		activeFlow     *activeflow.Activeflow
		targetActionID uuid.UUID

		responseDigits  string
		responseStackID uuid.UUID
		responseAction  *action.Action

		expectActiveFlow *activeflow.Activeflow
	}{
		{
			name: "normal",

			activeFlow: &activeflow.Activeflow{
				ReferenceID: uuid.FromStringOrNil("01f28ffc-0c08-11ec-8b28-0f1dd70b3428"),
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("4d174b14-91a3-11ec-861b-0f6aaeff6362"),
					Type:   action.TypeBranch,
					Option: []byte(`{"default_target_id":"59e4a526-91a3-11ec-83a3-7373495be152","target_ids":{"1":"623e8e48-91a4-11ec-aab0-d741c6c9423c"}}`),
				},
				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:     uuid.FromStringOrNil("4d174b14-91a3-11ec-861b-0f6aaeff6362"),
								Type:   action.TypeBranch,
								Option: []byte(`{"default_target_id":"59e4a526-91a3-11ec-83a3-7373495be152","target_ids":{"1":"623e8e48-91a4-11ec-aab0-d741c6c9423c"}}`),
							},
							{
								ID:   uuid.FromStringOrNil("59e4a526-91a3-11ec-83a3-7373495be152"),
								Type: action.TypeAnswer,
							},
							{
								ID:   uuid.FromStringOrNil("623e8e48-91a4-11ec-aab0-d741c6c9423c"),
								Type: action.TypeAnswer,
							},
						},
					},
				},
			},
			targetActionID: uuid.FromStringOrNil("623e8e48-91a4-11ec-aab0-d741c6c9423c"),

			responseDigits:  "1",
			responseStackID: stack.IDMain,
			responseAction: &action.Action{
				ID:   uuid.FromStringOrNil("623e8e48-91a4-11ec-aab0-d741c6c9423c"),
				Type: action.TypeAnswer,
			},

			expectActiveFlow: &activeflow.Activeflow{
				ReferenceID:     uuid.FromStringOrNil("01f28ffc-0c08-11ec-8b28-0f1dd70b3428"),
				ForwardStackID:  stack.IDMain,
				ForwardActionID: uuid.FromStringOrNil("623e8e48-91a4-11ec-aab0-d741c6c9423c"),
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("4d174b14-91a3-11ec-861b-0f6aaeff6362"),
					Type:   action.TypeBranch,
					Option: []byte(`{"default_target_id":"59e4a526-91a3-11ec-83a3-7373495be152","target_ids":{"1":"623e8e48-91a4-11ec-aab0-d741c6c9423c"}}`),
				},
				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:     uuid.FromStringOrNil("4d174b14-91a3-11ec-861b-0f6aaeff6362"),
								Type:   action.TypeBranch,
								Option: []byte(`{"default_target_id":"59e4a526-91a3-11ec-83a3-7373495be152","target_ids":{"1":"623e8e48-91a4-11ec-aab0-d741c6c9423c"}}`),
							},
							{
								ID:   uuid.FromStringOrNil("59e4a526-91a3-11ec-83a3-7373495be152"),
								Type: action.TypeAnswer,
							},
							{
								ID:   uuid.FromStringOrNil("623e8e48-91a4-11ec-aab0-d741c6c9423c"),
								Type: action.TypeAnswer,
							},
						},
					},
				},
			},
		},
		{
			name: "use default",

			activeFlow: &activeflow.Activeflow{
				ReferenceID: uuid.FromStringOrNil("01f28ffc-0c08-11ec-8b28-0f1dd70b3428"),
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("4d174b14-91a3-11ec-861b-0f6aaeff6362"),
					Type:   action.TypeBranch,
					Option: []byte(`{"default_target_id":"59e4a526-91a3-11ec-83a3-7373495be152","target_ids":{"1":"623e8e48-91a4-11ec-aab0-d741c6c9423c"}}`),
				},
				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:     uuid.FromStringOrNil("4d174b14-91a3-11ec-861b-0f6aaeff6362"),
								Type:   action.TypeBranch,
								Option: []byte(`{"default_target_id":"59e4a526-91a3-11ec-83a3-7373495be152","target_ids":{"1":"623e8e48-91a4-11ec-aab0-d741c6c9423c"}}`),
							},
							{
								ID:   uuid.FromStringOrNil("59e4a526-91a3-11ec-83a3-7373495be152"),
								Type: action.TypeAnswer,
							},
							{
								ID:   uuid.FromStringOrNil("623e8e48-91a4-11ec-aab0-d741c6c9423c"),
								Type: action.TypeAnswer,
							},
						},
					},
				},
			},
			targetActionID: uuid.FromStringOrNil("59e4a526-91a3-11ec-83a3-7373495be152"),

			responseDigits:  "",
			responseStackID: stack.IDMain,
			responseAction: &action.Action{
				ID:   uuid.FromStringOrNil("59e4a526-91a3-11ec-83a3-7373495be152"),
				Type: action.TypeAnswer,
			},

			expectActiveFlow: &activeflow.Activeflow{
				ReferenceID:     uuid.FromStringOrNil("01f28ffc-0c08-11ec-8b28-0f1dd70b3428"),
				ForwardStackID:  stack.IDMain,
				ForwardActionID: uuid.FromStringOrNil("59e4a526-91a3-11ec-83a3-7373495be152"),
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("4d174b14-91a3-11ec-861b-0f6aaeff6362"),
					Type:   action.TypeBranch,
					Option: []byte(`{"default_target_id":"59e4a526-91a3-11ec-83a3-7373495be152","target_ids":{"1":"623e8e48-91a4-11ec-aab0-d741c6c9423c"}}`),
				},
				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:     uuid.FromStringOrNil("4d174b14-91a3-11ec-861b-0f6aaeff6362"),
								Type:   action.TypeBranch,
								Option: []byte(`{"default_target_id":"59e4a526-91a3-11ec-83a3-7373495be152","target_ids":{"1":"623e8e48-91a4-11ec-aab0-d741c6c9423c"}}`),
							},
							{
								ID:   uuid.FromStringOrNil("59e4a526-91a3-11ec-83a3-7373495be152"),
								Type: action.TypeAnswer,
							},
							{
								ID:   uuid.FromStringOrNil("623e8e48-91a4-11ec-aab0-d741c6c9423c"),
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
			mockStack := stackhandler.NewMockStackHandler(mc)

			h := &activeflowHandler{
				db:           mockDB,
				reqHandler:   mockReq,
				stackHandler: mockStack,
			}

			ctx := context.Background()

			mockReq.EXPECT().CMV1CallGetDigits(ctx, tt.activeFlow.ReferenceID).Return(tt.responseDigits, nil)
			mockReq.EXPECT().CMV1CallSetDigits(ctx, tt.activeFlow.ReferenceID, "").Return(nil)
			mockStack.EXPECT().GetAction(ctx, tt.activeFlow.StackMap, tt.activeFlow.CurrentStackID, tt.targetActionID, false).Return(tt.responseStackID, tt.responseAction, nil)

			mockDB.EXPECT().ActiveflowUpdate(ctx, tt.expectActiveFlow).Return(nil)

			if err := h.actionHandleBranch(ctx, tt.activeFlow); err != nil {
				t.Errorf("Wrong match. exepct: ok, got: %v", err)
			}

		})
	}
}

func Test_actionHandleConditionCallDigits(t *testing.T) {

	tests := []struct {
		name string

		activeFlow *activeflow.Activeflow

		callID uuid.UUID

		responseDigits string
	}{
		{
			"length match",

			&activeflow.Activeflow{
				ReferenceID: uuid.FromStringOrNil("c2dbc228-92b4-11ec-8cc9-3358e0b8bbb5"),
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("c3434cae-92b4-11ec-aa8a-07d4fef0bef1"),
					Type:   action.TypeConditionCallDigits,
					Option: []byte(`{"length": 1}`),
				},

				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:     uuid.FromStringOrNil("c3434cae-92b4-11ec-aa8a-07d4fef0bef1"),
								Type:   action.TypeConditionCallDigits,
								Option: []byte(`{"length": 1}`),
							},
							{
								ID:   uuid.FromStringOrNil("c37492fa-92b4-11ec-94a0-1bfcaf781964"),
								Type: action.TypeAnswer,
							},
						},
					},
				},
			},

			uuid.FromStringOrNil("c2dbc228-92b4-11ec-8cc9-3358e0b8bbb5"),

			"1",
		},
		{
			"key match",

			&activeflow.Activeflow{
				ReferenceID: uuid.FromStringOrNil("6ef04e44-92b5-11ec-a70a-1b80e125f020"),
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("6f553cd2-92b5-11ec-a9cc-070ec1a9c665"),
					Type:   action.TypeConditionCallDigits,
					Option: []byte(`{"key": "3"}`),
				},

				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:     uuid.FromStringOrNil("6f553cd2-92b5-11ec-a9cc-070ec1a9c665"),
								Type:   action.TypeConditionCallDigits,
								Option: []byte(`{"key": "3"}`),
							},
							{
								ID:   uuid.FromStringOrNil("6f893f3c-92b5-11ec-9c9d-437fc938558b"),
								Type: action.TypeAnswer,
							},
						},
					},
				},
			},

			uuid.FromStringOrNil("6ef04e44-92b5-11ec-a70a-1b80e125f020"),

			"123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)

			h := &activeflowHandler{
				db:         mockDB,
				reqHandler: mockReq,
			}

			ctx := context.Background()

			mockReq.EXPECT().CMV1CallGetDigits(ctx, tt.callID).Return(tt.responseDigits, nil)

			if err := h.actionHandleConditionCallDigits(ctx, tt.activeFlow); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_actionHandleConditionCallDigitsFail(t *testing.T) {

	tests := []struct {
		name string

		activeFlow *activeflow.Activeflow

		callID               uuid.UUID
		expectTargetActionID uuid.UUID

		responseDigits      string
		expectReqActiveFlow *activeflow.Activeflow

		responseAction *action.Action
	}{
		{
			"length fail",

			&activeflow.Activeflow{
				ReferenceID: uuid.FromStringOrNil("c2dbc228-92b4-11ec-8cc9-3358e0b8bbb5"),
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("c3434cae-92b4-11ec-aa8a-07d4fef0bef1"),
					Type:   action.TypeConditionCallDigits,
					Option: []byte(`{"length": 3, "false_target_id": "c37492fa-92b4-11ec-94a0-1bfcaf781964"}`),
				},

				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:     uuid.FromStringOrNil("c3434cae-92b4-11ec-aa8a-07d4fef0bef1"),
								Type:   action.TypeConditionCallDigits,
								Option: []byte(`{"length": 3, "false_target_id": "c37492fa-92b4-11ec-94a0-1bfcaf781964"}`),
							},
							{
								ID:   uuid.FromStringOrNil("7dac84b8-d58e-11ec-8b33-b76f5cf8651a"),
								Type: action.TypeAnswer,
							},
							{
								ID:   uuid.FromStringOrNil("c37492fa-92b4-11ec-94a0-1bfcaf781964"),
								Type: action.TypeAnswer,
							},
						},
					},
				},
			},

			uuid.FromStringOrNil("c2dbc228-92b4-11ec-8cc9-3358e0b8bbb5"),
			uuid.FromStringOrNil("c37492fa-92b4-11ec-94a0-1bfcaf781964"),

			"1",
			&activeflow.Activeflow{
				ReferenceID:     uuid.FromStringOrNil("c2dbc228-92b4-11ec-8cc9-3358e0b8bbb5"),
				ForwardStackID:  stack.IDMain,
				ForwardActionID: uuid.FromStringOrNil("c37492fa-92b4-11ec-94a0-1bfcaf781964"),
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("c3434cae-92b4-11ec-aa8a-07d4fef0bef1"),
					Type:   action.TypeConditionCallDigits,
					Option: []byte(`{"length": 3, "false_target_id": "c37492fa-92b4-11ec-94a0-1bfcaf781964"}`),
				},

				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:     uuid.FromStringOrNil("c3434cae-92b4-11ec-aa8a-07d4fef0bef1"),
								Type:   action.TypeConditionCallDigits,
								Option: []byte(`{"length": 3, "false_target_id": "c37492fa-92b4-11ec-94a0-1bfcaf781964"}`),
							},
							{
								ID:   uuid.FromStringOrNil("7dac84b8-d58e-11ec-8b33-b76f5cf8651a"),
								Type: action.TypeAnswer,
							},
							{
								ID:   uuid.FromStringOrNil("c37492fa-92b4-11ec-94a0-1bfcaf781964"),
								Type: action.TypeAnswer,
							},
						},
					},
				},
			},

			&action.Action{
				ID:   uuid.FromStringOrNil("c37492fa-92b4-11ec-94a0-1bfcaf781964"),
				Type: action.TypeAnswer,
			},
		},
		{
			"key fail",

			&activeflow.Activeflow{
				ReferenceID: uuid.FromStringOrNil("6ef04e44-92b5-11ec-a70a-1b80e125f020"),
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("6f553cd2-92b5-11ec-a9cc-070ec1a9c665"),
					Type:   action.TypeConditionCallDigits,
					Option: []byte(`{"key": "5", "false_target_id": "6f893f3c-92b5-11ec-9c9d-437fc938558b"}`),
				},

				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:     uuid.FromStringOrNil("6f553cd2-92b5-11ec-a9cc-070ec1a9c665"),
								Type:   action.TypeConditionCallDigits,
								Option: []byte(`{"key": "5", "false_target_id": "6f893f3c-92b5-11ec-9c9d-437fc938558b"}`),
							},
							{
								ID:   uuid.FromStringOrNil("a128f228-d58e-11ec-9a02-c70b96d93760"),
								Type: action.TypeAnswer,
							},
							{
								ID:   uuid.FromStringOrNil("6f893f3c-92b5-11ec-9c9d-437fc938558b"),
								Type: action.TypeAnswer,
							},
						},
					},
				},
			},

			uuid.FromStringOrNil("6ef04e44-92b5-11ec-a70a-1b80e125f020"),
			uuid.FromStringOrNil("6f893f3c-92b5-11ec-9c9d-437fc938558b"),

			"123",
			&activeflow.Activeflow{
				ReferenceID:     uuid.FromStringOrNil("6ef04e44-92b5-11ec-a70a-1b80e125f020"),
				ForwardStackID:  stack.IDMain,
				ForwardActionID: uuid.FromStringOrNil("6f893f3c-92b5-11ec-9c9d-437fc938558b"),
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("6f553cd2-92b5-11ec-a9cc-070ec1a9c665"),
					Type:   action.TypeConditionCallDigits,
					Option: []byte(`{"key": "5", "false_target_id": "6f893f3c-92b5-11ec-9c9d-437fc938558b"}`),
				},

				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:     uuid.FromStringOrNil("6f553cd2-92b5-11ec-a9cc-070ec1a9c665"),
								Type:   action.TypeConditionCallDigits,
								Option: []byte(`{"key": "5", "false_target_id": "6f893f3c-92b5-11ec-9c9d-437fc938558b"}`),
							},
							{
								ID:   uuid.FromStringOrNil("a128f228-d58e-11ec-9a02-c70b96d93760"),
								Type: action.TypeAnswer,
							},
							{
								ID:   uuid.FromStringOrNil("6f893f3c-92b5-11ec-9c9d-437fc938558b"),
								Type: action.TypeAnswer,
							},
						},
					},
				},
			},
			&action.Action{
				ID:   uuid.FromStringOrNil("6f893f3c-92b5-11ec-9c9d-437fc938558b"),
				Type: action.TypeAnswer,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockStack := stackhandler.NewMockStackHandler(mc)

			h := &activeflowHandler{
				db:           mockDB,
				reqHandler:   mockReq,
				stackHandler: mockStack,
			}

			ctx := context.Background()

			mockReq.EXPECT().CMV1CallGetDigits(ctx, tt.callID).Return(tt.responseDigits, nil)
			mockStack.EXPECT().GetAction(ctx, tt.activeFlow.StackMap, tt.activeFlow.CurrentStackID, tt.expectTargetActionID, false).Return(stack.IDMain, tt.responseAction, nil)
			mockDB.EXPECT().ActiveflowUpdate(ctx, tt.expectReqActiveFlow).Return(nil)

			if err := h.actionHandleConditionCallDigits(ctx, tt.activeFlow); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_actionHandleConditionCallStatus(t *testing.T) {

	tests := []struct {
		name string

		activeFlow *activeflow.Activeflow

		responseCall *cmcall.Call
	}{
		{
			"normal",

			&activeflow.Activeflow{
				ReferenceID: uuid.FromStringOrNil("4980416e-9832-11ec-b189-5f96149e7ed8"),
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("49e01864-9832-11ec-a2de-8f0b27470613"),
					Type:   action.TypeConditionCallStatus,
					Option: []byte(`{"status": "ringing", "false_target_id": "52c1da9e-9832-11ec-9fc6-1b6bb10dc345"}`),
				},

				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:     uuid.FromStringOrNil("49e01864-9832-11ec-a2de-8f0b27470613"),
								Type:   action.TypeConditionCallDigits,
								Option: []byte(`{"status": "ringing", "false_target_id": "52c1da9e-9832-11ec-9fc6-1b6bb10dc345"}`),
							},
							{
								ID:   uuid.FromStringOrNil("5beb1950-9832-11ec-9c32-d7afb89fde90"),
								Type: action.TypeAnswer,
							},
							{
								ID:   uuid.FromStringOrNil("52c1da9e-9832-11ec-9fc6-1b6bb10dc345"),
								Type: action.TypeAnswer,
							},
						},
					},
				},
			},

			&cmcall.Call{
				ID:     uuid.FromStringOrNil("4980416e-9832-11ec-b189-5f96149e7ed8"),
				Status: cmcall.StatusRinging,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)

			h := &activeflowHandler{
				db:         mockDB,
				reqHandler: mockReq,
			}

			ctx := context.Background()

			mockReq.EXPECT().CMV1CallGet(ctx, tt.activeFlow.ReferenceID).Return(tt.responseCall, nil)

			if err := h.actionHandleConditionCallStatus(ctx, tt.activeFlow); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_actionHandleConditionCallStatusFalse(t *testing.T) {

	tests := []struct {
		name string

		activeFlow *activeflow.Activeflow

		expectTargetID uuid.UUID
		responseAction *action.Action

		responseCall        *cmcall.Call
		expectReqActiveFlow *activeflow.Activeflow
	}{
		{
			"normal",
			&activeflow.Activeflow{
				ReferenceID: uuid.FromStringOrNil("2a497210-9833-11ec-9c8c-6f81d7341b91"),
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("2a9dcc20-9833-11ec-8c07-4f5b407e5cdd"),
					Type:   action.TypeConditionCallStatus,
					Option: []byte(`{"status": "ringing", "false_target_id": "2afd89c6-9833-11ec-8e96-37a807af7aa9"}`),
				},

				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:     uuid.FromStringOrNil("2a9dcc20-9833-11ec-8c07-4f5b407e5cdd"),
								Type:   action.TypeConditionCallDigits,
								Option: []byte(`{"status": "ringing", "false_target_id": "2afd89c6-9833-11ec-8e96-37a807af7aa9"}`),
							},
							{
								ID:   uuid.FromStringOrNil("2ad1f3ce-9833-11ec-9625-13a4b9bd21a0"),
								Type: action.TypeAnswer,
							},
							{
								ID:   uuid.FromStringOrNil("2afd89c6-9833-11ec-8e96-37a807af7aa9"),
								Type: action.TypeAnswer,
							},
						},
					},
				},
			},

			uuid.FromStringOrNil("2afd89c6-9833-11ec-8e96-37a807af7aa9"),
			&action.Action{
				ID:   uuid.FromStringOrNil("2afd89c6-9833-11ec-8e96-37a807af7aa9"),
				Type: action.TypeAnswer,
			},

			&cmcall.Call{
				ID:     uuid.FromStringOrNil("2a497210-9833-11ec-9c8c-6f81d7341b91"),
				Status: cmcall.StatusProgressing,
			},
			&activeflow.Activeflow{
				ReferenceID:     uuid.FromStringOrNil("2a497210-9833-11ec-9c8c-6f81d7341b91"),
				ForwardStackID:  stack.IDMain,
				ForwardActionID: uuid.FromStringOrNil("2afd89c6-9833-11ec-8e96-37a807af7aa9"),
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("2a9dcc20-9833-11ec-8c07-4f5b407e5cdd"),
					Type:   action.TypeConditionCallStatus,
					Option: []byte(`{"status": "ringing", "false_target_id": "2afd89c6-9833-11ec-8e96-37a807af7aa9"}`),
				},

				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:     uuid.FromStringOrNil("2a9dcc20-9833-11ec-8c07-4f5b407e5cdd"),
								Type:   action.TypeConditionCallDigits,
								Option: []byte(`{"status": "ringing", "false_target_id": "2afd89c6-9833-11ec-8e96-37a807af7aa9"}`),
							},
							{
								ID:   uuid.FromStringOrNil("2ad1f3ce-9833-11ec-9625-13a4b9bd21a0"),
								Type: action.TypeAnswer,
							},
							{
								ID:   uuid.FromStringOrNil("2afd89c6-9833-11ec-8e96-37a807af7aa9"),
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
			mockStack := stackhandler.NewMockStackHandler(mc)

			h := &activeflowHandler{
				db:           mockDB,
				reqHandler:   mockReq,
				stackHandler: mockStack,
			}

			ctx := context.Background()

			mockReq.EXPECT().CMV1CallGet(ctx, tt.activeFlow.ReferenceID).Return(tt.responseCall, nil)
			mockStack.EXPECT().GetAction(ctx, tt.activeFlow.StackMap, tt.activeFlow.CurrentStackID, tt.expectTargetID, false).Return(stack.IDMain, tt.responseAction, nil)
			mockDB.EXPECT().ActiveflowUpdate(ctx, tt.expectReqActiveFlow)

			if err := h.actionHandleConditionCallStatus(ctx, tt.activeFlow); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_actionHandleMessageSend(t *testing.T) {

	tests := []struct {
		name string

		activeFlow *activeflow.Activeflow

		expectCustomerID   uuid.UUID
		expectSource       *cmaddress.Address
		expectDestinations []cmaddress.Address
		expectText         string

		responseVariable    *variable.Variable
		responseCall        *cmcall.Call
		expectReqActiveFlow *activeflow.Activeflow
	}{
		{
			name: "normal",

			activeFlow: &activeflow.Activeflow{
				ID:         uuid.FromStringOrNil("9332ed50-dc86-11ec-ba14-8b96908d546b"),
				CustomerID: uuid.FromStringOrNil("184d60ac-a2cf-11ec-a800-fb524059f338"),
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("4dcd7b64-a2ce-11ec-8711-6f247c91aa5d"),
					Type:   action.TypeMessageSend,
					Option: []byte(`{"source": {"type": "tel", "target": "+821100000001"}, "destinations": [{"type": "tel", "target": "+821100000002"},{"type": "tel", "target": "${test.destination}"}], "text": "hello world. test variable: ${test.variable}"}`),
				},

				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:     uuid.FromStringOrNil("4dcd7b64-a2ce-11ec-8711-6f247c91aa5d"),
								Type:   action.TypeMessageSend,
								Option: []byte(`{"source": {"type": "tel", "target": "+821100000001"}, "destinations": [{"type": "tel", "target": "+821100000002"},{"type": "tel", "target": "${test.destination}"}], "text": "hello world. test variable: ${test.variable}"}`),
							},
							{
								ID:   uuid.FromStringOrNil("9c06bcfa-a2ce-11ec-bcc6-5bc0b10cd014"),
								Type: action.TypeAnswer,
							},
							{
								ID:   uuid.FromStringOrNil("9c311c5c-a2ce-11ec-b1a2-d735b06f36c8"),
								Type: action.TypeAnswer,
							},
						},
					},
				},
			},

			expectCustomerID: uuid.FromStringOrNil("184d60ac-a2cf-11ec-a800-fb524059f338"),
			expectSource: &cmaddress.Address{
				Type:   cmaddress.TypeTel,
				Target: "+821100000001",
			},
			expectDestinations: []cmaddress.Address{
				{
					Type:   cmaddress.TypeTel,
					Target: "+821100000002",
				},
				{
					Type:   cmaddress.TypeTel,
					Target: "+821100000003",
				},
			},
			expectText: "hello world. test variable: ttteeesssttt",

			responseVariable: &variable.Variable{
				ID: uuid.FromStringOrNil("9332ed50-dc86-11ec-ba14-8b96908d546b"),
				Variables: map[string]string{
					"test.variable":    "ttteeesssttt",
					"test.destination": "+821100000003",
				},
			},
			responseCall: &cmcall.Call{
				ID:     uuid.FromStringOrNil("4d496946-a2ce-11ec-a96e-eb9ac0dce8e7"),
				Status: cmcall.StatusProgressing,
			},
			expectReqActiveFlow: &activeflow.Activeflow{
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("4dcd7b64-a2ce-11ec-8711-6f247c91aa5d"),
					Type:   action.TypeMessageSend,
					Option: []byte(`{"source": {"type": "tel", "target": "+821100000001"}, "destinations": [{"type": "tel", "target": "+821100000002"}], "text": "hello world"}`),
				},

				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:     uuid.FromStringOrNil("4dcd7b64-a2ce-11ec-8711-6f247c91aa5d"),
								Type:   action.TypeMessageSend,
								Option: []byte(`{"source": {"type": "tel", "target": "+821100000001"}, "destinations": [{"type": "tel", "target": "+821100000002"}], "text": "hello world"}`),
							},
							{
								ID:   uuid.FromStringOrNil("9c06bcfa-a2ce-11ec-bcc6-5bc0b10cd014"),
								Type: action.TypeAnswer,
							},
							{
								ID:   uuid.FromStringOrNil("9c311c5c-a2ce-11ec-b1a2-d735b06f36c8"),
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
			mockVar := variablehandler.NewMockVariableHandler(mc)

			h := &activeflowHandler{
				db:              mockDB,
				reqHandler:      mockReq,
				variableHandler: mockVar,
			}

			ctx := context.Background()

			mockVar.EXPECT().Get(ctx, tt.activeFlow.ID).Return(tt.responseVariable, nil)
			mockReq.EXPECT().MMV1MessageSend(ctx, tt.expectCustomerID, tt.expectSource, tt.expectDestinations, tt.expectText).Return(&mmmessage.Message{}, nil)

			if err := h.actionHandleMessageSend(ctx, tt.activeFlow); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_actionHandleTranscribeRecording(t *testing.T) {

	type test struct {
		name       string
		activeflow *activeflow.Activeflow

		callID     uuid.UUID
		customerID uuid.UUID
		language   string
	}

	tests := []test{
		{
			"normal",

			&activeflow.Activeflow{
				CustomerID:  uuid.FromStringOrNil("321089b0-8795-11ec-907f-0bae67409ef6"),
				ReferenceID: uuid.FromStringOrNil("66e928da-9b42-11eb-8da0-3783064961f6"),
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("673ed4d8-9b42-11eb-bb79-ff02c5650f35"),
					Type:   action.TypeTranscribeRecording,
					Option: []byte(`{"language":"en-US"}`),
				},
			},

			uuid.FromStringOrNil("66e928da-9b42-11eb-8da0-3783064961f6"),
			uuid.FromStringOrNil("321089b0-8795-11ec-907f-0bae67409ef6"),
			"en-US",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)

			h := &activeflowHandler{
				db:         mockDB,
				reqHandler: mockReq,
			}

			ctx := context.Background()
			mockReq.EXPECT().TSV1CallRecordingCreate(ctx, tt.customerID, tt.callID, tt.language, 120000, 30).Return([]tstranscribe.Transcribe{}, nil)
			if err := h.actionHandleTranscribeRecording(ctx, tt.activeflow); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_actionHandleTranscribeStart(t *testing.T) {

	type test struct {
		name       string
		activeFlow *activeflow.Activeflow

		customerID    uuid.UUID
		referenceID   uuid.UUID
		referenceType tstranscribe.Type
		language      string

		response *tstranscribe.Transcribe
	}

	tests := []test{
		{
			"normal",
			&activeflow.Activeflow{
				CustomerID:  uuid.FromStringOrNil("b4d3fb66-8795-11ec-997c-7f2786edbef2"),
				ReferenceID: uuid.FromStringOrNil("01f28ffc-0c08-11ec-8b28-0f1dd70b3428"),
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("0737bd5c-0c08-11ec-9ba8-3bc700c21fd4"),
					Type:   action.TypeTranscribeStart,
					Option: []byte(`{"language":"en-US","webhook_uri":"http://test.com/webhook","webhook_method":"POST"}`),
				},
			},

			uuid.FromStringOrNil("b4d3fb66-8795-11ec-997c-7f2786edbef2"),
			uuid.FromStringOrNil("01f28ffc-0c08-11ec-8b28-0f1dd70b3428"),
			tstranscribe.TypeCall,
			"en-US",

			&tstranscribe.Transcribe{
				ID:          uuid.FromStringOrNil("e1e69720-0c08-11ec-9f5c-db1f63f63215"),
				Type:        tstranscribe.TypeCall,
				ReferenceID: uuid.FromStringOrNil("01f28ffc-0c08-11ec-8b28-0f1dd70b3428"),
				HostID:      uuid.FromStringOrNil("f91b4f58-0c08-11ec-88fd-cfbbb1957a54"),
				Language:    "en-US",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)

			h := &activeflowHandler{
				db:         mockDB,
				reqHandler: mockReq,
			}

			ctx := context.Background()
			mockReq.EXPECT().TSV1StreamingCreate(ctx, tt.customerID, tt.referenceID, tt.referenceType, tt.language).Return(tt.response, nil)
			if err := h.actionHandleTranscribeStart(ctx, tt.activeFlow); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_actionHandleCall(t *testing.T) {

	tests := []struct {
		name string
		af   *activeflow.Activeflow

		source       *cmaddress.Address
		destinations []cmaddress.Address
		flowID       uuid.UUID
		actions      []action.Action
		masterCallID uuid.UUID

		responseFlow     *flow.Flow
		responseVariable *variable.Variable
		responseCall     []cmcall.Call
	}{
		{
			"single destination with flow id",
			&activeflow.Activeflow{
				ID:         uuid.FromStringOrNil("7fb0af78-a942-11ec-8c60-67384864d90a"),
				CustomerID: uuid.FromStringOrNil("4ea19a38-a941-11ec-b04d-bb69d70f3461"),
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("4edb5840-a941-11ec-b674-93c2ef347891"),
					Type:   action.TypeCall,
					Option: []byte(`{"source":{"type": "tel", "target": "+821100000001"}, "destinations": [{"type": "tel", "target": "+821100000002"}], "flow_id": "15df43ee-a941-11ec-a903-2b7266f49e4b"}`),
				},

				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:     uuid.FromStringOrNil("4edb5840-a941-11ec-b674-93c2ef347891"),
								Type:   action.TypeCall,
								Option: []byte(`{"source":{"type": "tel", "target": "+821100000001"}, "destinations": [{"type": "tel", "target": "+821100000002"}], "flow_id": "15df43ee-a941-11ec-a903-2b7266f49e4b"}`),
							},
						},
					},
				},
			},

			&cmaddress.Address{
				Type:   cmaddress.TypeTel,
				Target: "+821100000001",
			},
			[]cmaddress.Address{
				{
					Type:   cmaddress.TypeTel,
					Target: "+821100000002",
				},
			},
			uuid.FromStringOrNil("15df43ee-a941-11ec-a903-2b7266f49e4b"),
			[]action.Action{},
			uuid.Nil,

			&flow.Flow{},
			&variable.Variable{},
			[]cmcall.Call{
				{
					ID: uuid.FromStringOrNil("a49dda82-a941-11ec-b5a9-9baf180541e9"),
				},
			},
		},
		{
			"2 destinations with flow id",
			&activeflow.Activeflow{
				ID:         uuid.FromStringOrNil("fe7b3838-a941-11ec-b3d6-5337e9635d88"),
				CustomerID: uuid.FromStringOrNil("fea0d30e-a941-11ec-a38a-478b19a5dfd2"),
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("fec56ebc-a941-11ec-8e4c-9fafab93ddcc"),
					Type:   action.TypeCall,
					Option: []byte(`{"source":{"type": "tel", "target": "+821100000001"}, "destinations": [{"type": "tel", "target": "+821100000002"},{"type": "tel", "target": "+821100000003"}], "flow_id": "feedab34-a941-11ec-a6a8-1bbdada16b4d"}`),
				},

				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:     uuid.FromStringOrNil("fec56ebc-a941-11ec-8e4c-9fafab93ddcc"),
								Type:   action.TypeCall,
								Option: []byte(`{"source":{"type": "tel", "target": "+821100000001"}, "destinations": [,{"type": "tel", "target": "+821100000003"}], "flow_id": "feedab34-a941-11ec-a6a8-1bbdada16b4d"}`),
							},
						},
					},
				},
			},

			&cmaddress.Address{
				Type:   cmaddress.TypeTel,
				Target: "+821100000001",
			},
			[]cmaddress.Address{
				{
					Type:   cmaddress.TypeTel,
					Target: "+821100000002",
				},
				{
					Type:   cmaddress.TypeTel,
					Target: "+821100000003",
				},
			},
			uuid.FromStringOrNil("feedab34-a941-11ec-a6a8-1bbdada16b4d"),
			[]action.Action{},
			uuid.Nil,

			&flow.Flow{},
			&variable.Variable{},
			[]cmcall.Call{
				{
					ID: uuid.FromStringOrNil("ff16e1f2-a941-11ec-b2c1-c3aa4ce144a0"),
				},
				{
					ID: uuid.FromStringOrNil("ff477790-a941-11ec-8475-035d159c8a77"),
				},
			},
		},
		{
			"single destination with actions",
			&activeflow.Activeflow{
				ID:         uuid.FromStringOrNil("38a78fec-a943-11ec-b279-c7b264a9d36a"),
				CustomerID: uuid.FromStringOrNil("38d87102-a943-11ec-99aa-5fc910add207"),
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("39063286-a943-11ec-b54c-d3be23fdf738"),
					Type:   action.TypeCall,
					Option: []byte(`{"source":{"type": "tel", "target": "+821100000001"}, "destinations": [{"type": "tel", "target": "+821100000002"}], "actions": [{"type": "answer"}, {"type": "talk", "option": {"text": "hello world"}}]}`),
				},

				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:     uuid.FromStringOrNil("39063286-a943-11ec-b54c-d3be23fdf738"),
								Type:   action.TypeConnect,
								Option: []byte(`{"source":{"type": "tel", "target": "+821100000001"}, "destinations": [{"type": "tel", "target": "+821100000002"}], "actions": [{"type": "answer"}, {"type": "talk", "option": {"text": "hello world"}}]}`),
							},
						},
					},
				},
			},

			&cmaddress.Address{
				Type:   cmaddress.TypeTel,
				Target: "+821100000001",
			},
			[]cmaddress.Address{
				{
					Type:   cmaddress.TypeTel,
					Target: "+821100000002",
				},
			},
			uuid.Nil,
			[]action.Action{
				{
					Type: action.TypeAnswer,
				},
				{
					Type:   action.TypeTalk,
					Option: []byte(`{"text": "hello world"}`),
				},
			},
			uuid.Nil,

			&flow.Flow{
				ID: uuid.FromStringOrNil("3936013c-a943-11ec-bdf1-af72361eecf4"),
			},
			&variable.Variable{},
			[]cmcall.Call{
				{
					ID: uuid.FromStringOrNil("39691e00-a943-11ec-a69c-7f0e69becb70"),
				},
			},
		},
		{
			"2 destinations with actions",
			&activeflow.Activeflow{
				ID:         uuid.FromStringOrNil("f3ddb5ca-a943-11ec-be88-ebdd9b1c7f1b"),
				CustomerID: uuid.FromStringOrNil("f40e389e-a943-11ec-83a7-7f90e0c22e96"),
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("f43f85de-a943-11ec-9ba5-b3f019b002e7"),
					Type:   action.TypeCall,
					Option: []byte(`{"source":{"type": "tel", "target": "+821100000001"}, "destinations": [{"type": "tel", "target": "+821100000002"}], "actions": [{"type": "answer"}, {"type": "talk", "option": {"text": "hello world"}}]}`),
				},

				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:     uuid.FromStringOrNil("f43f85de-a943-11ec-9ba5-b3f019b002e7"),
								Type:   action.TypeConnect,
								Option: []byte(`{"source":{"type": "tel", "target": "+821100000001"}, "destinations": [{"type": "tel", "target": "+821100000002"}], "actions": [{"type": "answer"}, {"type": "talk", "option": {"text": "hello world"}}]}`),
							},
						},
					},
				},
			},

			&cmaddress.Address{
				Type:   cmaddress.TypeTel,
				Target: "+821100000001",
			},
			[]cmaddress.Address{
				{
					Type:   cmaddress.TypeTel,
					Target: "+821100000002",
				},
			},
			uuid.Nil,
			[]action.Action{
				{
					Type: action.TypeAnswer,
				},
				{
					Type:   action.TypeTalk,
					Option: []byte(`{"text": "hello world"}`),
				},
			},
			uuid.Nil,

			&flow.Flow{
				ID: uuid.FromStringOrNil("f46f3e96-a943-11ec-b4e1-2bfce6b84c2b"),
			},
			&variable.Variable{},
			[]cmcall.Call{
				{
					ID: uuid.FromStringOrNil("f4a90298-a943-11ec-b31a-bf1f552ced44"),
				},
				{
					ID: uuid.FromStringOrNil("f4dd1e0c-a943-11ec-9295-6f71727bd164"),
				},
			},
		},
		{
			"single destination with flow id and chained",
			&activeflow.Activeflow{
				ID:            uuid.FromStringOrNil("ec55367a-a993-11ec-9eaf-3bd79ecebfdb"),
				CustomerID:    uuid.FromStringOrNil("ec7e7e4a-a993-11ec-85a1-2f8c41cac00e"),
				ReferenceType: activeflow.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("3f0cd396-a994-11ec-95db-e73c30df842c"),
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("eca44350-a993-11ec-bb4d-cbe7cec73166"),
					Type:   action.TypeCall,
					Option: []byte(`{"source":{"type": "tel", "target": "+821100000001"}, "destinations": [{"type": "tel", "target": "+821100000002"}], "flow_id": "ecc964dc-a993-11ec-9c4c-13e5b3d40ea8", "chained": true}`),
				},

				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:     uuid.FromStringOrNil("eca44350-a993-11ec-bb4d-cbe7cec73166"),
								Type:   action.TypeCall,
								Option: []byte(`{"source":{"type": "tel", "target": "+821100000001"}, "destinations": [{"type": "tel", "target": "+821100000002"}], "flow_id": "ecc964dc-a993-11ec-9c4c-13e5b3d40ea8", "chained": true}`),
							},
						},
					},
				},
			},

			&cmaddress.Address{
				Type:   cmaddress.TypeTel,
				Target: "+821100000001",
			},
			[]cmaddress.Address{
				{
					Type:   cmaddress.TypeTel,
					Target: "+821100000002",
				},
			},
			uuid.FromStringOrNil("ecc964dc-a993-11ec-9c4c-13e5b3d40ea8"),
			[]action.Action{},
			uuid.FromStringOrNil("3f0cd396-a994-11ec-95db-e73c30df842c"),

			&flow.Flow{},
			&variable.Variable{},
			[]cmcall.Call{
				{
					ID: uuid.FromStringOrNil("2aafe7e4-a994-11ec-8bae-338c6a067225"),
				},
			},
		},
		{
			"single destination with flow id and chained but reference type is message",
			&activeflow.Activeflow{
				ID:            uuid.FromStringOrNil("87a4f032-a996-11ec-b260-4b6f3f52e1c9"),
				CustomerID:    uuid.FromStringOrNil("87ede80a-a996-11ec-9086-d77d045a5f03"),
				ReferenceType: activeflow.ReferenceTypeMessage,
				ReferenceID:   uuid.FromStringOrNil("8819cf88-a996-11ec-bd8b-f3a7053103f1"),
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("eca44350-a993-11ec-bb4d-cbe7cec73166"),
					Type:   action.TypeCall,
					Option: []byte(`{"source":{"type": "tel", "target": "+821100000001"}, "destinations": [{"type": "tel", "target": "+821100000002"}], "flow_id": "88497954-a996-11ec-b194-a71b02fcb6a8", "chained": true}`),
				},

				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:     uuid.FromStringOrNil("eca44350-a993-11ec-bb4d-cbe7cec73166"),
								Type:   action.TypeCall,
								Option: []byte(`{"source":{"type": "tel", "target": "+821100000001"}, "destinations": [{"type": "tel", "target": "+821100000002"}], "flow_id": "88497954-a996-11ec-b194-a71b02fcb6a8", "chained": true}`),
							},
						},
					},
				},
			},

			&cmaddress.Address{
				Type:   cmaddress.TypeTel,
				Target: "+821100000001",
			},
			[]cmaddress.Address{
				{
					Type:   cmaddress.TypeTel,
					Target: "+821100000002",
				},
			},
			uuid.FromStringOrNil("88497954-a996-11ec-b194-a71b02fcb6a8"),
			[]action.Action{},
			uuid.Nil,

			&flow.Flow{},
			&variable.Variable{},
			[]cmcall.Call{
				{
					ID: uuid.FromStringOrNil("8873be9e-a996-11ec-993b-438622bb78da"),
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
			mockVar := variablehandler.NewMockVariableHandler(mc)

			h := &activeflowHandler{
				db:         mockDB,
				reqHandler: mockReq,

				actionHandler:   mockAction,
				variableHandler: mockVar,
			}

			ctx := context.Background()

			flowID := tt.flowID
			if flowID == uuid.Nil {
				mockReq.EXPECT().FMV1FlowCreate(ctx, tt.af.CustomerID, flow.TypeFlow, "", "", tt.actions, false).Return(tt.responseFlow, nil)
				flowID = tt.responseFlow.ID
			}

			mockVar.EXPECT().Get(ctx, tt.af.ID).Return(tt.responseVariable, nil)
			mockReq.EXPECT().CMV1CallsCreate(ctx, tt.af.CustomerID, flowID, tt.masterCallID, tt.source, tt.destinations).Return(tt.responseCall, nil)

			if errCall := h.actionHandleCall(ctx, tt.af); errCall != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errCall)
			}
		})
	}
}

func Test_actionHandleVariableSet(t *testing.T) {

	tests := []struct {
		name         string
		activeflowID uuid.UUID
		af           *activeflow.Activeflow

		key   string
		value string
	}{
		{
			"single destination with flow id",
			uuid.FromStringOrNil("a65dd1f8-ce47-11ec-bc53-ff630bb4b69b"),
			&activeflow.Activeflow{
				ID:         uuid.FromStringOrNil("a65dd1f8-ce47-11ec-bc53-ff630bb4b69b"),
				CustomerID: uuid.FromStringOrNil("4ea19a38-a941-11ec-b04d-bb69d70f3461"),
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("a6896cc8-ce47-11ec-8fff-1f2ab0d61b07"),
					Type:   action.TypeVariableSet,
					Option: []byte(`{"key": "key 1","value":"value 1"}`),
				},

				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:     uuid.FromStringOrNil("a6896cc8-ce47-11ec-8fff-1f2ab0d61b07"),
								Type:   action.TypeVariableSet,
								Option: []byte(`{"key": "key 1","value":"value 1"}`),
							},
						},
					},
				},
			},

			"key 1",
			"value 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockAction := actionhandler.NewMockActionHandler(mc)
			mockVariable := variablehandler.NewMockVariableHandler(mc)

			h := &activeflowHandler{
				db:         mockDB,
				reqHandler: mockReq,

				actionHandler:   mockAction,
				variableHandler: mockVariable,
			}

			ctx := context.Background()

			mockVariable.EXPECT().SetVariable(ctx, tt.af.ID, tt.key, tt.value).Return(nil)

			if errCall := h.actionHandleVariableSet(ctx, tt.af); errCall != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errCall)
			}
		})
	}
}

func Test_actionHandleWebhookSend(t *testing.T) {

	tests := []struct {
		name string

		af *activeflow.Activeflow

		responseVariable *variable.Variable

		expectSync     bool
		expectURI      string
		expectMethod   webhook.MethodType
		expectDataType webhook.DataType
		expectData     []byte
	}{
		{
			"normal",

			&activeflow.Activeflow{
				ID:         uuid.FromStringOrNil("284a82d4-d9eb-11ec-aa89-3fb4df202ec8"),
				CustomerID: uuid.FromStringOrNil("4ea19a38-a941-11ec-b04d-bb69d70f3461"),
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("63574506-d9eb-11ec-a261-67821a8699b0"),
					Type:   action.TypeWebhookSend,
					Option: []byte(`{"sync":false,"uri":"test.com","method":"POST","data_type":"application/json","data":"variable test ${voipbin.test.name}."}`),
				},

				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:     uuid.FromStringOrNil("63574506-d9eb-11ec-a261-67821a8699b0"),
								Type:   action.TypeWebhookSend,
								Option: []byte(`{"sync":false,"uri":"test.com","method":"POST","data_type":"application/json","data":"variable test ${voipbin.test.name}."}`),
							},
						},
					},
				},
			},

			&variable.Variable{
				ID: uuid.FromStringOrNil("284a82d4-d9eb-11ec-aa89-3fb4df202ec8"),
				Variables: map[string]string{
					"voipbin.test.name": "test name",
				},
			},

			true,
			"test.com",
			webhook.MethodTypePOST,
			webhook.DataTypeJSON,
			[]byte(`variable test test name.`),
		},
		{
			"variable does not exist",

			&activeflow.Activeflow{
				ID:         uuid.FromStringOrNil("2751d836-da32-11ec-8ebb-df5c61605fec"),
				CustomerID: uuid.FromStringOrNil("4ea19a38-a941-11ec-b04d-bb69d70f3461"),
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("63574506-d9eb-11ec-a261-67821a8699b0"),
					Type:   action.TypeWebhookSend,
					Option: []byte(`{"sync":false,"uri":"test.com","method":"POST","data_type":"application/json","data":"variable test ${voipbin.test.name} and non-exist variable ${voipbin.test.non-exist}."}`),
				},

				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:     uuid.FromStringOrNil("63574506-d9eb-11ec-a261-67821a8699b0"),
								Type:   action.TypeWebhookSend,
								Option: []byte(`{"sync":false,"uri":"test.com","method":"POST","data_type":"application/json","data":"variable test ${voipbin.test.name} and non-exist variable ${voipbin.test.non-exist}."}`),
							},
						},
					},
				},
			},

			&variable.Variable{
				ID: uuid.FromStringOrNil("2751d836-da32-11ec-8ebb-df5c61605fec"),
				Variables: map[string]string{
					"voipbin.test.name": "test name",
				},
			},

			true,
			"test.com",
			webhook.MethodTypePOST,
			webhook.DataTypeJSON,
			[]byte(`variable test test name and non-exist variable .`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockAction := actionhandler.NewMockActionHandler(mc)
			mockVariable := variablehandler.NewMockVariableHandler(mc)

			h := &activeflowHandler{
				db:         mockDB,
				reqHandler: mockReq,

				actionHandler:   mockAction,
				variableHandler: mockVariable,
			}

			ctx := context.Background()

			mockVariable.EXPECT().Get(ctx, tt.af.ID).Return(tt.responseVariable, nil)
			mockReq.EXPECT().WMV1WebhookSendToDestination(ctx, tt.af.CustomerID, tt.expectURI, tt.expectMethod, tt.expectDataType, tt.expectData).Return(nil)

			if errCall := h.actionHandleWebhookSend(ctx, tt.af); errCall != nil {
				t.Errorf("Wrong match.\nexpect: ok\ngot: %v", errCall)
			}

			time.Sleep(500 * time.Millisecond)
		})
	}
}
