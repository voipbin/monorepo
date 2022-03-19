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
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	h := &activeflowHandler{
		db:            mockDB,
		notifyHandler: mockNotify,
	}

	tests := []struct {
		name string
		flow *flow.Flow

		refereceType activeflow.ReferenceType
		referenceID  uuid.UUID
		expectActive *activeflow.ActiveFlow
	}{
		{
			"normal",
			&flow.Flow{
				ID:      uuid.FromStringOrNil("dc8e048e-822e-11eb-8cb6-235002e45cf2"),
				Actions: []action.Action{},
			},

			activeflow.ReferenceTypeCall,
			uuid.FromStringOrNil("03e8a480-822f-11eb-b71f-8bbc09fa1e7a"),
			&activeflow.ActiveFlow{
				ID:            uuid.FromStringOrNil("32808a7c-a7a1-11ec-8de8-2331c11da2e8"),
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
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockDB.EXPECT().FlowGet(gomock.Any(), tt.flow.ID).Return(tt.flow, nil)
			mockDB.EXPECT().ActiveFlowCreate(gomock.Any(), gomock.Any()).Return(nil)
			mockDB.EXPECT().ActiveFlowGet(gomock.Any(), gomock.Any()).Return(tt.expectActive, nil)
			mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), tt.expectActive.CustomerID, activeflow.EventTypeActiveFlowCreated, tt.expectActive)

			res, err := h.ActiveFlowCreate(ctx, tt.refereceType, tt.referenceID, tt.flow.ID)
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
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	h := &activeflowHandler{
		db:            mockDB,
		notifyHandler: mockNotify,
	}

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
			ctx := context.Background()
			mockDB.EXPECT().ActiveFlowGet(gomock.Any(), tt.callID).Return(&activeflow.ActiveFlow{}, nil)
			mockDB.EXPECT().ActiveFlowSet(gomock.Any(), gomock.Any()).Return(nil)
			mockDB.EXPECT().ActiveFlowGet(gomock.Any(), tt.callID).Return(&activeflow.ActiveFlow{}, nil)
			mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), gomock.Any(), activeflow.EventTypeActiveFlowUpdated, gomock.Any())
			_, err := h.updateCurrentAction(ctx, tt.callID, tt.act)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestActiveFlowNextActionGet(t *testing.T) {
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

	tests := []struct {
		name         string
		callID       uuid.UUID
		actionID     uuid.UUID
		af           *activeflow.ActiveFlow
		expectAction action.Action
	}{
		{
			"normal",
			uuid.FromStringOrNil("0d276266-0737-11eb-808f-8f2856d44e29"),
			uuid.FromStringOrNil("05e2c40a-0737-11eb-9134-5f9b578a4179"),
			&activeflow.ActiveFlow{
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
			"empty actions",
			uuid.FromStringOrNil("085f48fc-08a4-11eb-8ef3-675e25cbc25c"),
			action.IDStart,
			&activeflow.ActiveFlow{
				CurrentAction: action.Action{
					ID: action.IDStart,
				},
				ForwardActionID: action.IDEmpty,
				Actions:         []action.Action{},
			},
			action.Action{
				ID:   action.IDFinish,
				Type: action.TypeHangup,
			},
		},
		{
			"current id start",
			uuid.FromStringOrNil("950c810c-08a4-11eb-af93-93115c7f9c55"),
			action.IDStart,
			&activeflow.ActiveFlow{
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
			&activeflow.ActiveFlow{
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
			ctx := context.Background()
			mockDB.EXPECT().ActiveFlowGet(gomock.Any(), tt.callID).Return(tt.af, nil).AnyTimes()

			if len(tt.af.Actions) == 0 {
				mockAction.EXPECT().CreateActionHangup().Return(&action.Action{
					ID:   action.IDFinish,
					Type: action.TypeHangup,
				})
			}

			mockDB.EXPECT().ActiveFlowSet(gomock.Any(), gomock.Any()).Return(nil)
			mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), tt.af.CustomerID, activeflow.EventTypeActiveFlowUpdated, tt.af)

			act, err := h.ActiveFlowNextActionGet(ctx, tt.callID, tt.actionID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if act.ID != tt.expectAction.ID || act.Type != tt.expectAction.Type {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectAction, act)
			}
		})
	}
}

func Test_getNextAction(t *testing.T) {
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

	tests := []struct {
		name         string
		callID       uuid.UUID
		af           activeflow.ActiveFlow
		expectAction action.Action
	}{
		{
			"next action echo",
			uuid.FromStringOrNil("f96b5730-0c24-11eb-89ff-af22fc6e8dce"),
			activeflow.ActiveFlow{
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
			"empty actions",
			uuid.FromStringOrNil("44413184-0c26-11eb-83a9-974d19b06d35"),
			activeflow.ActiveFlow{
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
		{
			"forwrad action id has set",
			uuid.FromStringOrNil("44413184-0c26-11eb-83a9-974d19b06d35"),
			activeflow.ActiveFlow{
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
			activeflow.ActiveFlow{
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
			ctx := context.Background()
			mockDB.EXPECT().ActiveFlowGet(gomock.Any(), tt.callID).Return(&tt.af, nil)

			if len(tt.af.Actions) == 0 {
				mockAction.EXPECT().CreateActionHangup().Return(&action.Action{
					ID:     action.IDFinish,
					Type:   action.TypeHangup,
					Option: []byte(`{}`),
				})
			}

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
