package activeflowhandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/activeflow"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/flow"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/actionhandler"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/dbhandler"
)

func TestActiveFlowCreate(t *testing.T) {

	tests := []struct {
		name string
		flow *flow.Flow

		id           uuid.UUID
		refereceType activeflow.ReferenceType
		referenceID  uuid.UUID
		expectActive *activeflow.Activeflow
		flowID       uuid.UUID
	}{
		{
			"normal",
			&flow.Flow{
				ID:      uuid.FromStringOrNil("dc8e048e-822e-11eb-8cb6-235002e45cf2"),
				Actions: []action.Action{},
			},

			uuid.FromStringOrNil("a58dc1e8-dc67-447b-9392-2d58531f1fb1"),
			activeflow.ReferenceTypeCall,
			uuid.FromStringOrNil("03e8a480-822f-11eb-b71f-8bbc09fa1e7a"),
			&activeflow.Activeflow{
				ID:            uuid.FromStringOrNil("a58dc1e8-dc67-447b-9392-2d58531f1fb1"),
				ReferenceType: activeflow.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("03e8a480-822f-11eb-b71f-8bbc09fa1e7a"),
				FlowID:        uuid.FromStringOrNil("dc8e048e-822e-11eb-8cb6-235002e45cf2"),
				CurrentAction: action.Action{
					ID: action.IDStart,
				},
				ExecuteCount:    0,
				ForwardActionID: action.IDEmpty,
				Actions:         []action.Action{},
				ExecutedActions: []action.Action{},
			},
			uuid.FromStringOrNil("dc8e048e-822e-11eb-8cb6-235002e45cf2"),
		},
		{
			"nil id",
			&flow.Flow{
				ID:      uuid.FromStringOrNil("dc8e048e-822e-11eb-8cb6-235002e45cf2"),
				Actions: []action.Action{},
			},

			uuid.Nil,
			activeflow.ReferenceTypeCall,
			uuid.FromStringOrNil("d6543076-aba3-46c2-ac82-46101f294bf5"),
			&activeflow.Activeflow{
				ID:            uuid.FromStringOrNil("78184d65-899f-438f-aeca-8cce4f445756"),
				ReferenceType: activeflow.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("d6543076-aba3-46c2-ac82-46101f294bf5"),
				FlowID:        uuid.FromStringOrNil("dc8e048e-822e-11eb-8cb6-235002e45cf2"),
				CurrentAction: action.Action{
					ID: action.IDStart,
				},
				ExecuteCount:    0,
				ForwardActionID: action.IDEmpty,
				Actions:         []action.Action{},
				ExecutedActions: []action.Action{},
			},
			uuid.FromStringOrNil("dc8e048e-822e-11eb-8cb6-235002e45cf2"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &activeflowHandler{
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().FlowGet(gomock.Any(), tt.flow.ID).Return(tt.flow, nil)
			mockDB.EXPECT().ActiveflowCreate(gomock.Any(), gomock.Any()).Return(nil)
			mockDB.EXPECT().ActiveflowGet(gomock.Any(), gomock.Any()).Return(tt.expectActive, nil)
			mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), tt.expectActive.CustomerID, activeflow.EventTypeActiveflowCreated, tt.expectActive)

			res, err := h.Create(ctx, tt.id, tt.refereceType, tt.referenceID, tt.flowID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectActive) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectActive, res)
			}
		})
	}
}

func TestActiveFlowUpdateCurrentAction(t *testing.T) {

	tests := []struct {
		name   string
		callID uuid.UUID
		act    *action.Action
	}{
		{
			"normal",
			uuid.FromStringOrNil("f594ebd8-06ae-11eb-9bca-5757b3876041"),
			&action.Action{
				ID:   uuid.FromStringOrNil("f916a6a2-06ae-11eb-a239-53802c6fbb36"),
				Type: action.TypeAnswer,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &activeflowHandler{
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()
			mockDB.EXPECT().ActiveflowGet(gomock.Any(), tt.callID).Return(&activeflow.Activeflow{}, nil)
			mockDB.EXPECT().ActiveflowUpdate(gomock.Any(), gomock.Any()).Return(nil)
			mockDB.EXPECT().ActiveflowGet(gomock.Any(), tt.callID).Return(&activeflow.Activeflow{}, nil)
			mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), gomock.Any(), activeflow.EventTypeActiveflowUpdated, gomock.Any())
			_, err := h.updateCurrentAction(ctx, tt.callID, tt.act)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_GetNextAction(t *testing.T) {

	tests := []struct {
		name         string
		id           uuid.UUID
		actionID     uuid.UUID
		af           *activeflow.Activeflow
		expectAction action.Action
	}{
		{
			"normal",
			uuid.FromStringOrNil("0d276266-0737-11eb-808f-8f2856d44e29"),
			uuid.FromStringOrNil("05e2c40a-0737-11eb-9134-5f9b578a4179"),
			&activeflow.Activeflow{
				CurrentAction: action.Action{
					ID:   uuid.FromStringOrNil("05e2c40a-0737-11eb-9134-5f9b578a4179"),
					Type: action.TypeAnswer,
				},
				ForwardActionID: action.IDEmpty,
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
			action.Action{
				ID:   uuid.FromStringOrNil("c9fffcf4-0737-11eb-a28f-2bc0bae5eeaf"),
				Type: action.TypeAnswer,
			},
		},
		{
			"current id start",
			uuid.FromStringOrNil("950c810c-08a4-11eb-af93-93115c7f9c55"),
			action.IDStart,
			&activeflow.Activeflow{
				CurrentAction: action.Action{
					ID: action.IDStart,
				},
				ForwardActionID: action.IDEmpty,
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
			action.Action{
				ID:   uuid.FromStringOrNil("97f96f9c-08a4-11eb-8ea0-57d38a96eca3"),
				Type: action.TypeAnswer,
			},
		},
		{
			"move action id has set",
			uuid.FromStringOrNil("6ed30c30-794c-11ec-98dc-237ea83d2fcb"),
			uuid.FromStringOrNil("bf5e3b10-5733-11ec-a0c6-879d0d048e2d"),
			&activeflow.Activeflow{
				CurrentAction: action.Action{
					ID: uuid.FromStringOrNil("bf5e3b10-5733-11ec-a0c6-879d0d048e2d"),
				},
				ForwardActionID: uuid.FromStringOrNil("ab88bd9a-5733-11ec-9fa5-df017a802cfc"),
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
			action.Action{
				ID:   uuid.FromStringOrNil("ab88bd9a-5733-11ec-9fa5-df017a802cfc"),
				Type: action.TypeAnswer,
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

			h := &activeflowHandler{
				db:            mockDB,
				notifyHandler: mockNotify,
				actionHandler: mockAction,
			}

			ctx := context.Background()
			mockDB.EXPECT().ActiveflowGet(gomock.Any(), tt.id).Return(tt.af, nil).AnyTimes()

			mockDB.EXPECT().ActiveflowUpdate(gomock.Any(), gomock.Any()).Return(nil)
			mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), tt.af.CustomerID, activeflow.EventTypeActiveflowUpdated, tt.af)

			act, err := h.GetNextAction(ctx, tt.id, tt.actionID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if act.ID != tt.expectAction.ID || act.Type != tt.expectAction.Type {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectAction, act)
			}
		})
	}
}

func Test_GetNextActionError(t *testing.T) {

	tests := []struct {
		name     string
		id       uuid.UUID
		actionID uuid.UUID
		af       *activeflow.Activeflow
	}{
		{
			"empty actions",
			uuid.FromStringOrNil("085f48fc-08a4-11eb-8ef3-675e25cbc25c"),
			action.IDStart,
			&activeflow.Activeflow{
				CurrentAction: action.Action{
					ID: action.IDStart,
				},
				ForwardActionID: action.IDEmpty,
				Actions:         []action.Action{},
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

			h := &activeflowHandler{
				db:            mockDB,
				notifyHandler: mockNotify,
				actionHandler: mockAction,
			}

			ctx := context.Background()
			mockDB.EXPECT().ActiveflowGet(gomock.Any(), tt.id).Return(tt.af, nil)

			if len(tt.af.Actions) == 0 {
				mockDB.EXPECT().ActiveflowDelete(ctx, tt.id).Return(nil)
				mockDB.EXPECT().ActiveflowGet(ctx, tt.id).Return(tt.af, nil)
				mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), tt.af.CustomerID, activeflow.EventTypeActiveflowDeleted, tt.af)
			}

			_, err := h.GetNextAction(ctx, tt.id, tt.actionID)
			if err == nil {
				t.Error("Wrong match. expect: err, got: ok")
			}
		})
	}
}

func Test_getNextAction(t *testing.T) {

	tests := []struct {
		name         string
		callID       uuid.UUID
		af           activeflow.Activeflow
		expectAction action.Action
	}{
		{
			"next action echo",
			uuid.FromStringOrNil("f96b5730-0c24-11eb-89ff-af22fc6e8dce"),
			activeflow.Activeflow{
				CustomerID: uuid.FromStringOrNil("a356975a-8055-11ec-9c11-37c0ba53de51"),
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("005a71ac-0c25-11eb-b9ba-ffa78e01ffc9"),
					Type:   action.TypeConnect,
					Option: []byte(`{"from":"+123456789", "destinations": [{"type": "tel", "name": "", "target": "+987654321"}]}`),
				},
				ForwardActionID: action.IDEmpty,
				Actions: []action.Action{
					{
						ID:     uuid.FromStringOrNil("005a71ac-0c25-11eb-b9ba-ffa78e01ffc9"),
						Type:   action.TypeConnect,
						Option: []byte(`{"from":"+123456789", "destinations": [{"type": "tel", "name": "", "target": "+987654321"}]}`),
					},
					{
						ID:   uuid.FromStringOrNil("686ece64-0c25-11eb-a025-ffd0ed1b73d2"),
						Type: action.TypeEcho,
					},
				},
			},
			action.Action{
				ID:   uuid.FromStringOrNil("686ece64-0c25-11eb-a025-ffd0ed1b73d2"),
				Type: action.TypeEcho,
			},
		},
		{
			"forwrad action id has set",
			uuid.FromStringOrNil("44413184-0c26-11eb-83a9-974d19b06d35"),
			activeflow.Activeflow{
				CustomerID: uuid.FromStringOrNil("a356975a-8055-11ec-9c11-37c0ba53de51"),
				CurrentAction: action.Action{
					ID: uuid.FromStringOrNil("15d7d942-574d-11ec-9e99-2fa8e28a2590"),
				},

				ForwardActionID: uuid.FromStringOrNil("055eaece-574d-11ec-a54a-8fe3a5c78c8b"),
				Actions: []action.Action{
					{
						ID:   uuid.FromStringOrNil("055eaece-574d-11ec-a54a-8fe3a5c78c8b"),
						Type: action.TypeAnswer,
					},
					{
						ID:   uuid.FromStringOrNil("15d7d942-574d-11ec-9e99-2fa8e28a2590"),
						Type: action.TypeAnswer,
					},
					{
						ID:   uuid.FromStringOrNil("15f911a2-574d-11ec-ba14-2fabebacf4bb"),
						Type: action.TypeAnswer,
					},
				},
			},
			action.Action{
				ID:   uuid.FromStringOrNil("055eaece-574d-11ec-a54a-8fe3a5c78c8b"),
				Type: action.TypeAnswer,
			},
		},
		{
			"next id has set",
			uuid.FromStringOrNil("e83a9588-9851-11ec-b987-07ce29329c80"),
			activeflow.Activeflow{
				ForwardActionID: action.IDEmpty,
				CustomerID:      uuid.FromStringOrNil("e869c452-9851-11ec-aa4c-fbddf1193904"),
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("e89463c4-9851-11ec-bc37-5ff5ed0bf091"),
					NextID: uuid.FromStringOrNil("0763d50a-9852-11ec-92d1-4b6db72a5ee8"),
				},

				Actions: []action.Action{
					{
						ID:   uuid.FromStringOrNil("e89463c4-9851-11ec-bc37-5ff5ed0bf091"),
						Type: action.TypeAnswer,
					},
					{
						ID:   uuid.FromStringOrNil("e8c79686-9851-11ec-af4a-234fea2ae8da"),
						Type: action.TypeAnswer,
					},
					{
						ID:   uuid.FromStringOrNil("0763d50a-9852-11ec-92d1-4b6db72a5ee8"),
						Type: action.TypeAnswer,
					},
				},
			},
			action.Action{
				ID:   uuid.FromStringOrNil("0763d50a-9852-11ec-92d1-4b6db72a5ee8"),
				Type: action.TypeAnswer,
			},
		}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockAction := actionhandler.NewMockActionHandler(mc)

			h := &activeflowHandler{
				db:         mockDB,
				reqHandler: mockReq,

				actionHandler: mockAction,
			}

			ctx := context.Background()
			mockDB.EXPECT().ActiveflowGet(gomock.Any(), tt.callID).Return(&tt.af, nil)

			act, err := h.getNextAction(ctx, tt.callID, tt.af.CurrentAction.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(act, &tt.expectAction) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectAction, act)
			}
		})
	}
}

func Test_getNextActionError(t *testing.T) {

	tests := []struct {
		name         string
		callID       uuid.UUID
		af           activeflow.Activeflow
		expectAction action.Action
	}{
		{
			"empty actions",
			uuid.FromStringOrNil("44413184-0c26-11eb-83a9-974d19b06d35"),
			activeflow.Activeflow{
				CustomerID: uuid.FromStringOrNil("a356975a-8055-11ec-9c11-37c0ba53de51"),
				CurrentAction: action.Action{
					ID: action.IDStart,
				},
				ForwardActionID: action.IDEmpty,
			},
			action.Action{
				ID:     action.IDFinish,
				Type:   action.TypeHangup,
				Option: []byte(`{}`),
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

			h := &activeflowHandler{
				db:         mockDB,
				reqHandler: mockReq,

				actionHandler: mockAction,
			}

			ctx := context.Background()
			mockDB.EXPECT().ActiveflowGet(gomock.Any(), tt.callID).Return(&tt.af, nil)

			_, err := h.getNextAction(ctx, tt.callID, tt.af.CurrentAction.ID)
			if err == nil {
				t.Errorf("Wrong match. expect: error, got: ok")
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

		af                     *activeflow.Activeflow
		expectUpdateActiveflow *activeflow.Activeflow
	}{
		{
			"reference type call forward now true",

			uuid.FromStringOrNil("1bd514f0-af6c-11ec-bddc-db11051293e5"),
			uuid.FromStringOrNil("1c998542-af6c-11ec-b385-c7a45742f1a1"),
			true,

			&activeflow.Activeflow{
				ID:         uuid.FromStringOrNil("1bd514f0-af6c-11ec-bddc-db11051293e5"),
				CustomerID: uuid.FromStringOrNil("fcc49e18-af6c-11ec-9857-8bc5d3558dc9"),
				CurrentAction: action.Action{
					ID:   uuid.FromStringOrNil("1cc5a9e2-af6c-11ec-ad49-db6eee64a325"),
					Type: action.TypeAnswer,
				},
				ForwardActionID: action.IDEmpty,
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
			&activeflow.Activeflow{
				ID:         uuid.FromStringOrNil("1bd514f0-af6c-11ec-bddc-db11051293e5"),
				CustomerID: uuid.FromStringOrNil("fcc49e18-af6c-11ec-9857-8bc5d3558dc9"),
				CurrentAction: action.Action{
					ID:   uuid.FromStringOrNil("1cc5a9e2-af6c-11ec-ad49-db6eee64a325"),
					Type: action.TypeAnswer,
				},
				ForwardActionID: uuid.FromStringOrNil("1c998542-af6c-11ec-b385-c7a45742f1a1"),
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
		{
			"reference type call forward now false",

			uuid.FromStringOrNil("1bc62ef8-af6d-11ec-a2d2-d36eb561e845"),
			uuid.FromStringOrNil("1c19a9f2-af6d-11ec-afe0-4bb7a2667649"),
			false,

			&activeflow.Activeflow{
				ID:         uuid.FromStringOrNil("1bc62ef8-af6d-11ec-a2d2-d36eb561e845"),
				CustomerID: uuid.FromStringOrNil("fc989a84-af6c-11ec-8bb9-23ec42502bfa"),
				CurrentAction: action.Action{
					ID:   uuid.FromStringOrNil("1bedaa5a-af6d-11ec-99f4-3b55921b1b50"),
					Type: action.TypeAnswer,
				},
				ForwardActionID: action.IDEmpty,
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
			&activeflow.Activeflow{
				ID:         uuid.FromStringOrNil("1bc62ef8-af6d-11ec-a2d2-d36eb561e845"),
				CustomerID: uuid.FromStringOrNil("fc989a84-af6c-11ec-8bb9-23ec42502bfa"),
				CurrentAction: action.Action{
					ID:   uuid.FromStringOrNil("1bedaa5a-af6d-11ec-99f4-3b55921b1b50"),
					Type: action.TypeAnswer,
				},
				ForwardActionID: uuid.FromStringOrNil("1c19a9f2-af6d-11ec-afe0-4bb7a2667649"),
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
		{
			"reference type message forward now true",

			uuid.FromStringOrNil("91875644-af6d-11ec-bf11-5fa477b94be1"),
			uuid.FromStringOrNil("91d53d96-af6d-11ec-8b73-27eb3b54c06f"),
			true,

			&activeflow.Activeflow{
				ID:         uuid.FromStringOrNil("91875644-af6d-11ec-bf11-5fa477b94be1"),
				CustomerID: uuid.FromStringOrNil("fc989a84-af6c-11ec-8bb9-23ec42502bfa"),
				CurrentAction: action.Action{
					ID:   uuid.FromStringOrNil("91b048a6-af6d-11ec-ba12-1f7793a35ea0"),
					Type: action.TypeAnswer,
				},
				ForwardActionID: action.IDEmpty,
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
			&activeflow.Activeflow{
				ID:         uuid.FromStringOrNil("91875644-af6d-11ec-bf11-5fa477b94be1"),
				CustomerID: uuid.FromStringOrNil("fc989a84-af6c-11ec-8bb9-23ec42502bfa"),
				CurrentAction: action.Action{
					ID:   uuid.FromStringOrNil("91b048a6-af6d-11ec-ba12-1f7793a35ea0"),
					Type: action.TypeAnswer,
				},
				ForwardActionID: uuid.FromStringOrNil("91d53d96-af6d-11ec-8b73-27eb3b54c06f"),
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockAction := actionhandler.NewMockActionHandler(mc)

			h := &activeflowHandler{
				db:         mockDB,
				reqHandler: mockReq,

				actionHandler: mockAction,
			}

			ctx := context.Background()
			mockDB.EXPECT().ActiveflowGet(ctx, tt.id).Return(tt.af, nil)
			mockDB.EXPECT().ActiveflowUpdate(ctx, tt.expectUpdateActiveflow).Return(nil)

			if tt.forwardNow && tt.af.ReferenceType == activeflow.ReferenceTypeCall {
				mockReq.EXPECT().CMV1CallActionNext(ctx, tt.af.ReferenceID, true).Return(nil)
			}

			if err := h.SetForwardActionID(ctx, tt.id, tt.actionID, tt.forwardNow); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
