package flowhandler

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/activeflow"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/flow"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/dbhandler"
)

func TestActiveFlowCreate(t *testing.T) {
	// we can't test this function.

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	h := &flowHandler{
		db:            mockDB,
		notifyHandler: mockNotify,
	}

	tests := []struct {
		name         string
		flow         *flow.Flow
		callID       uuid.UUID
		expectActive *activeflow.ActiveFlow
	}{
		{
			"normal",
			&flow.Flow{
				ID:      uuid.FromStringOrNil("dc8e048e-822e-11eb-8cb6-235002e45cf2"),
				Actions: []action.Action{},
			},
			uuid.FromStringOrNil("03e8a480-822f-11eb-b71f-8bbc09fa1e7a"),
			&activeflow.ActiveFlow{
				CallID: uuid.FromStringOrNil("03e8a480-822f-11eb-b71f-8bbc09fa1e7a"),
				FlowID: uuid.FromStringOrNil("dc8e048e-822e-11eb-8cb6-235002e45cf2"),
				CurrentAction: action.Action{
					ID: action.IDStart,
				},
				ExecuteCount:    0,
				ForwardActionID: action.IDEmpty,
				Actions:         []action.Action{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockDB.EXPECT().FlowGet(gomock.Any(), tt.flow.ID).Return(tt.flow, nil)
			mockDB.EXPECT().ActiveFlowCreate(gomock.Any(), tt.expectActive).Return(nil)
			mockDB.EXPECT().ActiveFlowGet(gomock.Any(), tt.callID).Return(tt.expectActive, nil)
			mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), tt.expectActive.CustomerID, activeflow.EventTypeActiveFlowCreated, tt.expectActive)

			res, err := h.ActiveFlowCreate(ctx, tt.callID, tt.flow.ID)
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

	h := &flowHandler{
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
			if err := h.activeFlowUpdateCurrentAction(ctx, tt.callID, tt.act); err != nil {
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

	h := &flowHandler{
		db:            mockDB,
		notifyHandler: mockNotify,
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

func TestCreateActionHangup(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &flowHandler{
		db: mockDB,
	}

	type test struct {
		name string
	}

	tests := []test{
		{
			"normal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			res := *h.CreateActionHangup()

			marString, err := json.Marshal(res)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			var act action.Action
			if err := json.Unmarshal(marString, &act); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			var opt action.OptionHangup
			if err := json.Unmarshal(act.Option, &opt); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestAppendActions(t *testing.T) {
	type test struct {
		name         string
		action1      []action.Action
		action2      []action.Action
		expectAction []action.Action

		targetActionID uuid.UUID
	}

	tests := []test{
		{
			"normal",
			[]action.Action{
				{
					ID: uuid.FromStringOrNil("c0a54954-0a96-11eb-80b2-8b6ef3a21db9"),
				},
				{
					ID: uuid.FromStringOrNil("ce32b80e-0a96-11eb-9ca3-3f423a830f93"),
				},
				{
					ID: uuid.FromStringOrNil("ce32b80e-0a96-11eb-9ca3-3f423a830f93"),
				},
			},
			[]action.Action{
				{
					ID: uuid.FromStringOrNil("e14c605c-0a96-11eb-9542-233abdd04f35"),
				},
				{
					ID: uuid.FromStringOrNil("e1858a8a-0a96-11eb-bf05-ab02488632d7"),
				},
				{
					ID: uuid.FromStringOrNil("e1b6e8d2-0a96-11eb-be8e-131d2f0bf1fe"),
				},
			},

			[]action.Action{
				{
					ID: uuid.FromStringOrNil("c0a54954-0a96-11eb-80b2-8b6ef3a21db9"),
				},
				{
					ID: uuid.FromStringOrNil("ce32b80e-0a96-11eb-9ca3-3f423a830f93"),
				},
				{
					ID: uuid.FromStringOrNil("e14c605c-0a96-11eb-9542-233abdd04f35"),
				},
				{
					ID: uuid.FromStringOrNil("e1858a8a-0a96-11eb-bf05-ab02488632d7"),
				},
				{
					ID: uuid.FromStringOrNil("e1b6e8d2-0a96-11eb-be8e-131d2f0bf1fe"),
				},
				{
					ID: uuid.FromStringOrNil("ce32b80e-0a96-11eb-9ca3-3f423a830f93"),
				},
			},
			uuid.FromStringOrNil("ce32b80e-0a96-11eb-9ca3-3f423a830f93"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			af := &activeflow.ActiveFlow{
				Actions: tt.action1,
			}
			if err := appendActions(af, tt.targetActionID, tt.action2); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(af.Actions, tt.expectAction) != true {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectAction, tt.action1)
			}
		})
	}
}

func TestReplaceActions(t *testing.T) {
	type test struct {
		name         string
		action1      []action.Action
		action2      []action.Action
		expectAction []action.Action

		targetActionID uuid.UUID
	}

	tests := []test{
		{
			"normal",
			[]action.Action{
				{
					ID: uuid.FromStringOrNil("c0a54954-0a96-11eb-80b2-8b6ef3a21db9"),
				},
				{
					ID: uuid.FromStringOrNil("ce32b80e-0a96-11eb-9ca3-3f423a830f93"),
				},
				{
					ID: uuid.FromStringOrNil("ce32b80e-0a96-11eb-9ca3-3f423a830f93"),
				},
			},
			[]action.Action{
				{
					ID: uuid.FromStringOrNil("e14c605c-0a96-11eb-9542-233abdd04f35"),
				},
				{
					ID: uuid.FromStringOrNil("e1858a8a-0a96-11eb-bf05-ab02488632d7"),
				},
				{
					ID: uuid.FromStringOrNil("e1b6e8d2-0a96-11eb-be8e-131d2f0bf1fe"),
				},
			},

			[]action.Action{
				{
					ID: uuid.FromStringOrNil("c0a54954-0a96-11eb-80b2-8b6ef3a21db9"),
				},
				{
					ID: uuid.FromStringOrNil("e14c605c-0a96-11eb-9542-233abdd04f35"),
				},
				{
					ID: uuid.FromStringOrNil("e1858a8a-0a96-11eb-bf05-ab02488632d7"),
				},
				{
					ID: uuid.FromStringOrNil("e1b6e8d2-0a96-11eb-be8e-131d2f0bf1fe"),
				},
				{
					ID: uuid.FromStringOrNil("ce32b80e-0a96-11eb-9ca3-3f423a830f93"),
				},
			},
			uuid.FromStringOrNil("ce32b80e-0a96-11eb-9ca3-3f423a830f93"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			af := &activeflow.ActiveFlow{
				Actions: tt.action1,
			}
			if err := replaceActions(af, tt.targetActionID, tt.action2); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(af.Actions, tt.expectAction) != true {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectAction, tt.action1)
			}
		})
	}
}
