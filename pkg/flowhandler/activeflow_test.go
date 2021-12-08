package flowhandler

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	cmaddress "gitlab.com/voipbin/bin-manager/call-manager.git/models/address"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	cmcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	cfconference "gitlab.com/voipbin/bin-manager/conference-manager.git/models/conference"
	"gitlab.com/voipbin/bin-manager/request-manager.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcribe"

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

	h := &flowHandler{
		db: mockDB,
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
		{
			"webhoook uri",
			&flow.Flow{
				ID:         uuid.FromStringOrNil("2c30890e-8233-11eb-a29c-4f8994729ffe"),
				WebhookURI: "https://test.com/webhook_uri",
				Actions:    []action.Action{},
			},
			uuid.FromStringOrNil("308ddd4e-8233-11eb-9079-2ba011592aa6"),
			&activeflow.ActiveFlow{
				CallID:     uuid.FromStringOrNil("308ddd4e-8233-11eb-9079-2ba011592aa6"),
				FlowID:     uuid.FromStringOrNil("2c30890e-8233-11eb-a29c-4f8994729ffe"),
				WebhookURI: "https://test.com/webhook_uri",
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

	h := &flowHandler{
		db: mockDB,
	}

	type test struct {
		name   string
		callID uuid.UUID
		act    *action.Action
	}

	tests := []test{
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

	h := &flowHandler{
		db: mockDB,
	}

	type test struct {
		name         string
		callID       uuid.UUID
		actionID     uuid.UUID
		af           activeflow.ActiveFlow
		expectAction action.Action
	}

	tests := []test{
		{
			"normal",
			uuid.FromStringOrNil("0d276266-0737-11eb-808f-8f2856d44e29"),
			uuid.FromStringOrNil("05e2c40a-0737-11eb-9134-5f9b578a4179"),
			activeflow.ActiveFlow{
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
			activeflow.ActiveFlow{
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
			activeflow.ActiveFlow{
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
			uuid.FromStringOrNil("950c810c-08a4-11eb-af93-93115c7f9c55"),
			uuid.FromStringOrNil("bf5e3b10-5733-11ec-a0c6-879d0d048e2d"),
			activeflow.ActiveFlow{
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
			mockDB.EXPECT().ActiveFlowGet(gomock.Any(), tt.callID).Return(&tt.af, nil).AnyTimes()

			mockDB.EXPECT().ActiveFlowSet(gomock.Any(), gomock.Any()).Return(nil)
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

func TestAppendActionsAfterID(t *testing.T) {
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
			if err := appendActionsAfterID(af, tt.targetActionID, tt.action2); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(af.Actions, tt.expectAction) != true {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectAction, tt.action1)
			}
		})
	}
}

func TestActiveFlowNextActionGetTypeConnect(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)

	h := &flowHandler{
		db:         mockDB,
		reqHandler: mockReq,
	}

	type test struct {
		name         string
		callID       uuid.UUID
		act          *action.Action
		af           activeflow.ActiveFlow
		cf           *cfconference.Conference
		connectFlow  *flow.Flow
		source       *cmaddress.Address
		destinations []*cmaddress.Address
		unchained    bool
	}

	tests := []test{
		{
			"single destination",
			uuid.FromStringOrNil("e1a258ca-0a98-11eb-8e3b-e7d2a18277fa"),
			&action.Action{
				ID:     uuid.FromStringOrNil("f4a4a87e-0a98-11eb-8f96-cba83b8b3f76"),
				Type:   action.TypeConnect,
				Option: []byte(`{"source":{"type": "tel", "target": "+123456789"}, "destinations": [{"type": "tel", "name": "", "target": "+987654321"}]}`),
			},
			activeflow.ActiveFlow{
				UserID: 1,
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("f4a4a87e-0a98-11eb-8f96-cba83b8b3f76"),
					Type:   action.TypeConnect,
					Option: []byte(`{"source":{"type": "tel", "target": "+123456789"}, "destinations": [{"type": "tel", "name": "", "target": "+987654321"}]}`),
				},
				Actions: []action.Action{
					{
						ID:     uuid.FromStringOrNil("f4a4a87e-0a98-11eb-8f96-cba83b8b3f76"),
						Type:   action.TypeConnect,
						Option: []byte(`{"source":{"type": "tel", "target": "+123456789"}, "destinations": [{"type": "tel", "name": "", "target": "+987654321"}]}`),
					},
				},
			},
			&cfconference.Conference{
				ID:     uuid.FromStringOrNil("363b4ae8-0a9b-11eb-9d08-436d6934a451"),
				UserID: 1,
			},
			&flow.Flow{
				ID:     uuid.FromStringOrNil("fa26f0ce-0a9b-11eb-8850-afda1bb6bc03"),
				UserID: 1,
			},
			&cmaddress.Address{
				Type:   cmaddress.TypeTel,
				Target: "+123456789",
			},
			[]*cmaddress.Address{
				{
					Type:   cmaddress.TypeTel,
					Target: "+987654321",
				},
			},
			false,
		},

		{
			"multiple destinations",
			uuid.FromStringOrNil("cb4accf8-2710-11eb-8e49-e73409394bef"),
			&action.Action{
				ID:     uuid.FromStringOrNil("cbe12fa4-2710-11eb-8959-87391e4bbc77"),
				Type:   action.TypeConnect,
				Option: []byte(`{"source":{"type": "tel", "target": "+123456789"}, "destinations": [{"type": "tel", "name": "", "target": "+987654321"}, {"type": "tel", "name": "", "target": "+9876543210"}]}`),
			},
			activeflow.ActiveFlow{
				UserID: 1,
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("cbe12fa4-2710-11eb-8959-87391e4bbc77"),
					Type:   action.TypeConnect,
					Option: []byte(`{"source":{"type": "tel", "target": "+123456789"}, "destinations": [{"type": "tel", "name": "", "target": "+987654321"}, {"type": "tel", "name": "", "target": "+9876543210"}]}`),
				},
				Actions: []action.Action{
					{
						ID:     uuid.FromStringOrNil("cbe12fa4-2710-11eb-8959-87391e4bbc77"),
						Type:   action.TypeConnect,
						Option: []byte(`{"source":{"type": "tel", "target": "+123456789"}, "destinations": [{"type": "tel", "name": "", "target": "+987654321"}, {"type": "tel", "name": "", "target": "+9876543210"}]}`),
					},
				},
			},
			&cfconference.Conference{
				ID:     uuid.FromStringOrNil("cc131f96-2710-11eb-b3b2-1b43dc6ffa2f"),
				UserID: 1,
			},
			&flow.Flow{
				ID:     uuid.FromStringOrNil("cc480ff8-2710-11eb-8869-0fcf3d58fd6a"),
				UserID: 1,
			},
			&cmaddress.Address{
				Type:   cmaddress.TypeTel,
				Target: "+123456789",
			},
			[]*cmaddress.Address{
				{
					Type:   cmaddress.TypeTel,
					Target: "+987654321",
				},
				{
					Type:   cmaddress.TypeTel,
					Target: "+9876543210",
				},
			},
			false,
		},

		{
			"multiple unchained destinations",
			uuid.FromStringOrNil("211a68fe-2712-11eb-ad71-97e2b1546a91"),
			&action.Action{
				ID:     uuid.FromStringOrNil("22311f94-2712-11eb-8550-0f0b066f8120"),
				Type:   action.TypeConnect,
				Option: []byte(`{"source":{"type": "tel", "target": "+123456789"}, "destinations": [{"type": "tel", "name": "", "target": "+987654321"}, {"type": "tel", "name": "", "target": "+9876543210"}], "unchained": true}`),
			},
			activeflow.ActiveFlow{
				UserID: 1,
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("22311f94-2712-11eb-8550-0f0b066f8120"),
					Type:   action.TypeConnect,
					Option: []byte(`{"source":{"type": "tel", "target": "+123456789"}, "destinations": [{"type": "tel", "name": "", "target": "+987654321"}, {"type": "tel", "name": "", "target": "+9876543210"}], "unchained": true}`),
				},
				Actions: []action.Action{
					{
						ID:     uuid.FromStringOrNil("22311f94-2712-11eb-8550-0f0b066f8120"),
						Type:   action.TypeConnect,
						Option: []byte(`{"source":{"type": "tel", "target": "+123456789"}, "destinations": [{"type": "tel", "name": "", "target": "+987654321"}, {"type": "tel", "name": "", "target": "+9876543210"}], "unchained": true}`),
					},
				},
			},
			&cfconference.Conference{
				ID:     uuid.FromStringOrNil("2266e688-2712-11eb-aab4-eb00b0a3efbe"),
				UserID: 1,
			},
			&flow.Flow{
				ID:     uuid.FromStringOrNil("229ef410-2712-11eb-9dea-a737f7b6ef2b"),
				UserID: 1,
			},
			&cmaddress.Address{
				Type:   cmaddress.TypeTel,
				Target: "+123456789",
			},
			[]*cmaddress.Address{
				{
					Type:   cmaddress.TypeTel,
					Target: "+987654321",
				},
				{
					Type:   cmaddress.TypeTel,
					Target: "+9876543210",
				},
			},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockDB.EXPECT().ActiveFlowGet(gomock.Any(), tt.callID).Return(&tt.af, nil)
			mockReq.EXPECT().CFV1ConferenceCreate(ctx, tt.af.UserID, cfconference.TypeConnect, "", "", 86400, "", nil, nil, nil).Return(tt.cf, nil)
			mockDB.EXPECT().FlowSetToCache(gomock.Any(), gomock.Any()).Return(nil)
			mockDB.EXPECT().FlowGet(gomock.Any(), gomock.Any()).Return(tt.connectFlow, nil)
			for i := range tt.destinations {
				mockReq.EXPECT().CMV1CallCreate(ctx, tt.connectFlow.UserID, tt.connectFlow.ID, tt.source, tt.destinations[i]).Return(&call.Call{ID: uuid.Nil}, nil)
				if tt.unchained == false {
					mockReq.EXPECT().CMV1CallAddChainedCall(ctx, tt.callID, uuid.Nil).Return(nil)
				}
			}
			mockDB.EXPECT().ActiveFlowSet(gomock.Any(), gomock.Any()).Return(nil)

			if err := h.activeFlowHandleActionConnect(ctx, tt.callID, tt.act); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestActiveFlowGetNextAction(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)

	h := &flowHandler{
		db:         mockDB,
		reqHandler: mockReq,
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
				UserID: 1,
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
				UserID: 1,
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
				UserID: 1,
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockDB.EXPECT().ActiveFlowGet(gomock.Any(), tt.callID).Return(&tt.af, nil)

			act, err := h.activeFlowGetNextAction(ctx, tt.callID, tt.af.CurrentAction.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(act, &tt.expectAction) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectAction, act)
			}
		})
	}
}

func TestActiveFlowNextActionGetTypeTranscribeRecording(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)

	h := &flowHandler{
		db:         mockDB,
		reqHandler: mockReq,
	}

	type test struct {
		name          string
		callID        uuid.UUID
		language      string
		webhookURI    string
		WebhookMethod string
		act           *action.Action
	}

	tests := []test{
		{
			"normal",
			uuid.FromStringOrNil("66e928da-9b42-11eb-8da0-3783064961f6"),
			"en-US",
			"http://test.com/webhook",
			"POST",
			&action.Action{
				ID:     uuid.FromStringOrNil("673ed4d8-9b42-11eb-bb79-ff02c5650f35"),
				Type:   action.TypeTranscribeRecording,
				Option: []byte(`{"language":"en-US","webhook_uri":"http://test.com/webhook","webhook_method":"POST"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockReq.EXPECT().TSV1CallRecordingCreate(ctx, tt.callID, tt.language, tt.webhookURI, tt.WebhookMethod, 120, 30).Return(nil)
			if err := h.activeFlowHandleActionTranscribeRecording(ctx, tt.callID, tt.act); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestActiveFlowNextActionGetTypeTranscribeStart(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)

	h := &flowHandler{
		db:         mockDB,
		reqHandler: mockReq,
	}

	type test struct {
		name string

		callID        uuid.UUID
		language      string
		webhookURI    string
		WebhookMethod string
		act           *action.Action

		response *transcribe.Transcribe
	}

	tests := []test{
		{
			"normal",
			uuid.FromStringOrNil("01f28ffc-0c08-11ec-8b28-0f1dd70b3428"),
			"en-US",
			"http://test.com/webhook",
			"POST",
			&action.Action{
				ID:     uuid.FromStringOrNil("0737bd5c-0c08-11ec-9ba8-3bc700c21fd4"),
				Type:   action.TypeTranscribeStart,
				Option: []byte(`{"language":"en-US","webhook_uri":"http://test.com/webhook","webhook_method":"POST"}`),
			},

			&transcribe.Transcribe{
				ID:            uuid.FromStringOrNil("e1e69720-0c08-11ec-9f5c-db1f63f63215"),
				Type:          transcribe.TypeCall,
				ReferenceID:   uuid.FromStringOrNil("01f28ffc-0c08-11ec-8b28-0f1dd70b3428"),
				HostID:        uuid.FromStringOrNil("f91b4f58-0c08-11ec-88fd-cfbbb1957a54"),
				Language:      "en-US",
				WebhookURI:    "http://test.com/webhook",
				WebhookMethod: "POST",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockReq.EXPECT().TSV1StreamingCreate(ctx, tt.callID, tt.language, tt.webhookURI, tt.WebhookMethod).Return(tt.response, nil)
			if err := h.activeFlowHandleActionTranscribeStart(ctx, tt.callID, tt.act); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestActiveFlowHandleActionPatchFlow(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)

	h := &flowHandler{
		db:         mockDB,
		reqHandler: mockReq,
	}

	tests := []struct {
		name string

		callID     uuid.UUID
		act        *action.Action
		activeFlow *activeflow.ActiveFlow
		flow       *flow.Flow
	}{
		{
			"normal",
			uuid.FromStringOrNil("01f28ffc-0c08-11ec-8b28-0f1dd70b3428"),
			&action.Action{
				ID:     uuid.FromStringOrNil("ec99431a-3cbf-11ec-b530-b3c665dd8156"),
				Type:   action.TypePatchFlow,
				Option: []byte(`{"flow_id": "a1d247b4-3cbf-11ec-8d08-970ce7001aaa"}`),
			},
			&activeflow.ActiveFlow{
				Actions: []action.Action{
					{
						ID:   uuid.FromStringOrNil("ec99431a-3cbf-11ec-b530-b3c665dd8156"),
						Type: action.TypeAnswer,
					},
				},
			},
			&flow.Flow{
				ID: uuid.FromStringOrNil("a1d247b4-3cbf-11ec-8d08-970ce7001aaa"),
				Actions: []action.Action{
					{
						Type: action.TypeAMD,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockDB.EXPECT().ActiveFlowGet(gomock.Any(), tt.callID).Return(tt.activeFlow, nil)
			mockDB.EXPECT().FlowGet(gomock.Any(), tt.flow.ID).Return(tt.flow, nil)
			mockDB.EXPECT().ActiveFlowSet(gomock.Any(), gomock.Any()).Return(nil)

			if err := h.activeFlowHandleActionPatchFlow(ctx, tt.callID, tt.act); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestActiveFlowHandleActionConferenceJoin(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)

	h := &flowHandler{
		db:         mockDB,
		reqHandler: mockReq,
	}

	tests := []struct {
		name string

		callID         uuid.UUID
		act            *action.Action
		activeFlow     *activeflow.ActiveFlow
		conference     *cfconference.Conference
		conferenceFlow *flow.Flow
	}{
		{
			"normal",
			uuid.FromStringOrNil("01f28ffc-0c08-11ec-8b28-0f1dd70b3428"),
			&action.Action{
				ID:     uuid.FromStringOrNil("7dbc6998-410d-11ec-91b8-d722b27bb799"),
				Type:   action.TypeConferenceJoin,
				Option: []byte(`{"conference_id": "b7c84d66-410b-11ec-ab21-23726c7dc3b9"}`),
			},
			&activeflow.ActiveFlow{
				Actions: []action.Action{
					{
						ID:   uuid.FromStringOrNil("7dbc6998-410d-11ec-91b8-d722b27bb799"),
						Type: action.TypeConfbridgeJoin,
					},
				},
			},
			&cfconference.Conference{
				ID:     uuid.FromStringOrNil("b7c84d66-410b-11ec-ab21-23726c7dc3b9"),
				FlowID: uuid.FromStringOrNil("b7eb3420-410b-11ec-ad87-cf5b4e34b7ed"),
				Status: cfconference.StatusProgressing,
			},
			&flow.Flow{
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
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockReq.EXPECT().CFV1ConferenceGet(ctx, tt.conference.ID).Return(tt.conference, nil)
			mockDB.EXPECT().FlowGet(gomock.Any(), tt.conference.FlowID).Return(tt.conferenceFlow, nil)
			mockDB.EXPECT().ActiveFlowGet(gomock.Any(), tt.callID).Return(tt.activeFlow, nil)
			mockDB.EXPECT().ActiveFlowSet(gomock.Any(), gomock.Any()).Return(nil)

			if err := h.activeFlowHandleActionConferenceJoin(ctx, tt.callID, tt.act); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestActiveFlowHandleActionAgentCall(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)

	h := &flowHandler{
		db:         mockDB,
		reqHandler: mockReq,
	}

	tests := []struct {
		name string

		callID  uuid.UUID
		act     *action.Action
		agentID uuid.UUID

		activeFlow     *activeflow.ActiveFlow
		conference     *cfconference.Conference
		call           *cmcall.Call
		conferenceFlow *flow.Flow
	}{
		{
			"normal",
			uuid.FromStringOrNil("71418cbe-53fc-11ec-980a-8fc233c3e802"),
			&action.Action{
				ID:     uuid.FromStringOrNil("716f309c-53fc-11ec-bff3-df8c8ffa945f"),
				Type:   action.TypeAgentCall,
				Option: []byte(`{"agent_id": "89593b12-53fc-11ec-9747-f7e71c3a8660"}`),
			},
			uuid.FromStringOrNil("89593b12-53fc-11ec-9747-f7e71c3a8660"),

			&activeflow.ActiveFlow{
				Actions: []action.Action{
					{
						ID:   uuid.FromStringOrNil("716f309c-53fc-11ec-bff3-df8c8ffa945f"),
						Type: action.TypeAgentCall,
					},
				},
			},
			&cfconference.Conference{
				ID:           uuid.FromStringOrNil("b7c84d66-410b-11ec-ab21-23726c7dc3b9"),
				FlowID:       uuid.FromStringOrNil("b7eb3420-410b-11ec-ad87-cf5b4e34b7ed"),
				ConfbridgeID: uuid.FromStringOrNil("9e60e850-53fe-11ec-a557-d7a7cce806ba"),
				Status:       cfconference.StatusProgressing,
			},
			&cmcall.Call{
				ID: uuid.FromStringOrNil("edee9f1c-53fd-11ec-a387-cb7cbdc7d345"),
				Source: cmaddress.Address{
					Type:   cmaddress.TypeTel,
					Target: "+821021656521",
				},
			},
			&flow.Flow{
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
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockDB.EXPECT().ActiveFlowGet(gomock.Any(), tt.callID).Return(tt.activeFlow, nil)
			mockReq.EXPECT().CFV1ConferenceCreate(gomock.Any(), tt.activeFlow.UserID, cfconference.TypeConnect, "", "", 86400, "", nil, nil, nil).Return(tt.conference, nil)
			mockReq.EXPECT().CMV1CallGet(gomock.Any(), tt.callID).Return(tt.call, nil)
			mockReq.EXPECT().AMV1AgentDial(gomock.Any(), tt.agentID, &tt.call.Source, tt.conference.ConfbridgeID).Return(nil)
			mockDB.EXPECT().ActiveFlowSet(gomock.Any(), gomock.Any()).Return(nil)

			if err := h.activeFlowHandleActionAgentCall(ctx, tt.callID, tt.act); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
func TestActiveFlowHandleActionGotoNoLoop(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)

	h := &flowHandler{
		db:         mockDB,
		reqHandler: mockReq,
	}

	tests := []struct {
		name string

		callID uuid.UUID
		act    *action.Action

		activeFlow *activeflow.ActiveFlow

		expectRes *action.Action
	}{
		{
			"normal",
			uuid.FromStringOrNil("01f28ffc-0c08-11ec-8b28-0f1dd70b3428"),
			&action.Action{
				ID:     uuid.FromStringOrNil("2d099c6e-55a3-11ec-85b0-db3612865f6e"),
				Type:   action.TypeGoto,
				Option: []byte(`{"target_id":"7dbc6998-410d-11ec-91b8-d722b27bb799"}`),
			},
			&activeflow.ActiveFlow{
				Actions: []action.Action{
					{
						ID:   uuid.FromStringOrNil("7dbc6998-410d-11ec-91b8-d722b27bb799"),
						Type: action.TypeAnswer,
					},
					{
						ID:     uuid.FromStringOrNil("2d099c6e-55a3-11ec-85b0-db3612865f6e"),
						Type:   action.TypeGoto,
						Option: []byte(`{"target_id":"7dbc6998-410d-11ec-91b8-d722b27bb799"}`),
					},
				},
			},
			&action.Action{
				ID:   uuid.FromStringOrNil("7dbc6998-410d-11ec-91b8-d722b27bb799"),
				Type: action.TypeAnswer,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockDB.EXPECT().ActiveFlowGet(gomock.Any(), tt.callID).Return(tt.activeFlow, nil)

			res, err := h.activeFlowHandleActionGoto(ctx, tt.callID, tt.act)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func TestActiveFlowHandleActionGotoLoopContinue(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)

	h := &flowHandler{
		db:         mockDB,
		reqHandler: mockReq,
	}

	tests := []struct {
		name string

		callID uuid.UUID
		act    *action.Action

		activeFlow       *activeflow.ActiveFlow
		updateActiveFlow *activeflow.ActiveFlow

		expectRes *action.Action
	}{
		{
			"normal",
			uuid.FromStringOrNil("01f28ffc-0c08-11ec-8b28-0f1dd70b3428"),
			&action.Action{
				ID:     uuid.FromStringOrNil("2d099c6e-55a3-11ec-85b0-db3612865f6e"),
				Type:   action.TypeGoto,
				Option: []byte(`{"target_id":"7dbc6998-410d-11ec-91b8-d722b27bb799","loop":true,"loop_count":3}`),
			},
			&activeflow.ActiveFlow{
				Actions: []action.Action{
					{
						ID:   uuid.FromStringOrNil("7dbc6998-410d-11ec-91b8-d722b27bb799"),
						Type: action.TypeAnswer,
					},
					{
						ID:     uuid.FromStringOrNil("2d099c6e-55a3-11ec-85b0-db3612865f6e"),
						Type:   action.TypeGoto,
						Option: []byte(`{"target_id":"7dbc6998-410d-11ec-91b8-d722b27bb799","loop":true,"loop_count":3}`),
					},
				},
			},
			&activeflow.ActiveFlow{
				Actions: []action.Action{
					{
						ID:   uuid.FromStringOrNil("7dbc6998-410d-11ec-91b8-d722b27bb799"),
						Type: action.TypeAnswer,
					},
					{
						ID:     uuid.FromStringOrNil("2d099c6e-55a3-11ec-85b0-db3612865f6e"),
						Type:   action.TypeGoto,
						Option: []byte(`{"target_index":0,"target_id":"7dbc6998-410d-11ec-91b8-d722b27bb799","loop":true,"loop_count":2}`),
					},
				},
			},
			&action.Action{
				ID:   uuid.FromStringOrNil("7dbc6998-410d-11ec-91b8-d722b27bb799"),
				Type: action.TypeAnswer,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockDB.EXPECT().ActiveFlowGet(gomock.Any(), tt.callID).Return(tt.activeFlow, nil)
			mockDB.EXPECT().ActiveFlowSet(gomock.Any(), tt.updateActiveFlow).Return(nil)

			res, err := h.activeFlowHandleActionGoto(ctx, tt.callID, tt.act)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func TestActiveFlowHandleActionGotoLoopStop(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)

	h := &flowHandler{
		db:         mockDB,
		reqHandler: mockReq,
	}

	tests := []struct {
		name string

		callID uuid.UUID
		act    *action.Action

		activeFlow *activeflow.ActiveFlow

		expectRes *action.Action
	}{
		{
			"normal",
			uuid.FromStringOrNil("01f28ffc-0c08-11ec-8b28-0f1dd70b3428"),
			&action.Action{
				ID:     uuid.FromStringOrNil("2d099c6e-55a3-11ec-85b0-db3612865f6e"),
				Type:   action.TypeGoto,
				Option: []byte(`{"target_id":"7dbc6998-410d-11ec-91b8-d722b27bb799","loop":true,"loop_count":0}`),
			},
			&activeflow.ActiveFlow{
				Actions: []action.Action{
					{
						ID:   uuid.FromStringOrNil("7dbc6998-410d-11ec-91b8-d722b27bb799"),
						Type: action.TypeAnswer,
					},
					{
						ID:     uuid.FromStringOrNil("2d099c6e-55a3-11ec-85b0-db3612865f6e"),
						Type:   action.TypeGoto,
						Option: []byte(`{"target_id":"7dbc6998-410d-11ec-91b8-d722b27bb799","loop":true,"loop_count":0}`),
					},
					{
						ID:   uuid.FromStringOrNil("8568520c-55f2-11ec-868f-3b955c9b9a39"),
						Type: action.TypeTalk,
					},
				},
			},
			&action.Action{
				ID:   uuid.FromStringOrNil("8568520c-55f2-11ec-868f-3b955c9b9a39"),
				Type: action.TypeTalk,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockDB.EXPECT().ActiveFlowGet(gomock.Any(), tt.callID).Return(tt.activeFlow, nil)

			res, err := h.activeFlowHandleActionGoto(ctx, tt.callID, tt.act)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}
