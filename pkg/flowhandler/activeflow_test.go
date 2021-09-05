package flowhandler

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/address"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/conference"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/activeflow"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/flow"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcribe"
)

func TestActiveFlowCreate(t *testing.T) {
	// we can't test this function.

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &flowHandler{
		db: mockDB,
	}

	type test struct {
		name         string
		flow         *flow.Flow
		callID       uuid.UUID
		expectActive *activeflow.ActiveFlow
	}

	tests := []test{
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
				Actions: []action.Action{},
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
				Actions: []action.Action{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockDB.EXPECT().FlowGet(gomock.Any(), tt.flow.ID).Return(tt.flow, nil)
			mockDB.EXPECT().ActiveFlowCreate(gomock.Any(), tt.expectActive).Return(nil)
			mockDB.EXPECT().ActiveFlowGet(gomock.Any(), tt.callID).Return(tt.expectActive, nil)

			// mockDB.EXPECT().ActiveFlowGet(gomock.Any(), tt.callID).Return(&activeflow.ActiveFlow{}, nil)
			// mockDB.EXPECT().ActiveFlowSet(gomock.Any(), gomock.Any()).Return(nil)
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
				Actions: []action.Action{},
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockDB.EXPECT().ActiveFlowGet(gomock.Any(), tt.callID).Return(&tt.af, nil)
			mockDB.EXPECT().ActiveFlowGet(gomock.Any(), tt.callID).Return(&tt.af, nil)

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
		cf           *conference.Conference
		connectFlow  *flow.Flow
		source       address.Address
		destinations []address.Address
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
			&conference.Conference{
				ID:     uuid.FromStringOrNil("363b4ae8-0a9b-11eb-9d08-436d6934a451"),
				UserID: 1,
			},
			&flow.Flow{
				ID:     uuid.FromStringOrNil("fa26f0ce-0a9b-11eb-8850-afda1bb6bc03"),
				UserID: 1,
			},
			address.Address{
				Type:   address.TypeTel,
				Target: "+123456789",
			},
			[]address.Address{
				{
					Type:   address.TypeTel,
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
			&conference.Conference{
				ID:     uuid.FromStringOrNil("cc131f96-2710-11eb-b3b2-1b43dc6ffa2f"),
				UserID: 1,
			},
			&flow.Flow{
				ID:     uuid.FromStringOrNil("cc480ff8-2710-11eb-8869-0fcf3d58fd6a"),
				UserID: 1,
			},
			address.Address{
				Type:   address.TypeTel,
				Target: "+123456789",
			},
			[]address.Address{
				{
					Type:   address.TypeTel,
					Target: "+987654321",
				},
				{
					Type:   address.TypeTel,
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
			&conference.Conference{
				ID:     uuid.FromStringOrNil("2266e688-2712-11eb-aab4-eb00b0a3efbe"),
				UserID: 1,
			},
			&flow.Flow{
				ID:     uuid.FromStringOrNil("229ef410-2712-11eb-9dea-a737f7b6ef2b"),
				UserID: 1,
			},
			address.Address{
				Type:   address.TypeTel,
				Target: "+123456789",
			},
			[]address.Address{
				{
					Type:   address.TypeTel,
					Target: "+987654321",
				},
				{
					Type:   address.TypeTel,
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
			mockReq.EXPECT().CMConferenceCreate(tt.af.UserID, conference.TypeConnect, "", "", 86400).Return(tt.cf, nil)
			mockDB.EXPECT().FlowSetToCache(gomock.Any(), gomock.Any()).Return(nil)
			mockDB.EXPECT().FlowGet(gomock.Any(), gomock.Any()).Return(tt.connectFlow, nil)
			for i := range tt.destinations {
				mockReq.EXPECT().CMCallCreate(tt.connectFlow.UserID, tt.connectFlow.ID, tt.source, tt.destinations[i]).Return(&call.Call{ID: uuid.Nil}, nil)
				if tt.unchained == false {
					mockReq.EXPECT().CMCallAddChainedCall(tt.callID, uuid.Nil).Return(nil)
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

	type test struct {
		name         string
		callID       uuid.UUID
		af           activeflow.ActiveFlow
		expectAction action.Action
	}

	tests := []test{
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
			ctx := context.Background()
			mockDB.EXPECT().ActiveFlowGet(gomock.Any(), tt.callID).Return(&tt.af, nil)
			mockDB.EXPECT().ActiveFlowGet(gomock.Any(), tt.callID).Return(&tt.af, nil)
			mockDB.EXPECT().ActiveFlowSet(gomock.Any(), gomock.Any()).Return(nil)

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
			mockReq.EXPECT().TMCallRecordingPost(tt.callID, tt.language, tt.webhookURI, tt.WebhookMethod, 120, 30).Return(nil)
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
			mockReq.EXPECT().TMStreamingsPost(tt.callID, tt.language, tt.webhookURI, tt.WebhookMethod).Return(tt.response, nil)
			if err := h.activeFlowHandleActionTranscribeStart(ctx, tt.callID, tt.act); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
