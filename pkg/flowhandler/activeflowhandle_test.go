package flowhandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	cmaddress "gitlab.com/voipbin/bin-manager/call-manager.git/models/address"
	cmcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	cfconference "gitlab.com/voipbin/bin-manager/conference-manager.git/models/conference"
	qmqueue "gitlab.com/voipbin/bin-manager/queue-manager.git/models/queue"
	qmqueuecall "gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecall"
	"gitlab.com/voipbin/bin-manager/request-manager.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcribe"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/activeflow"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/flow"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/dbhandler"
)

func TestActiveFlowHandleActionConnect(t *testing.T) {
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
				mockReq.EXPECT().CMV1CallCreate(ctx, tt.connectFlow.UserID, tt.connectFlow.ID, tt.source, tt.destinations[i]).Return(&cmcall.Call{ID: uuid.Nil}, nil)
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

		callID        uuid.UUID
		act           *action.Action
		resActiveFlow *activeflow.ActiveFlow
		flow          *flow.Flow

		expectFlow *activeflow.ActiveFlow
		expectRes  *action.Action
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
						ID:   uuid.FromStringOrNil("f0b5605e-648e-11ec-b318-a7f267cc71fc"),
						Type: action.TypeAnswer,
					},
					{
						ID:   uuid.FromStringOrNil("ec99431a-3cbf-11ec-b530-b3c665dd8156"),
						Type: action.TypePatchFlow,
					},
					{
						ID:   uuid.FromStringOrNil("ad108d6a-648e-11ec-a226-536bc1253066"),
						Type: action.TypeTalk,
					},
				},
			},
			&flow.Flow{
				ID: uuid.FromStringOrNil("a1d247b4-3cbf-11ec-8d08-970ce7001aaa"),
				Actions: []action.Action{
					{
						ID:   uuid.FromStringOrNil("e2af181a-648e-11ec-878b-2bb6c0cebb3e"),
						Type: action.TypeAMD,
					},
				},
			},

			&activeflow.ActiveFlow{
				Actions: []action.Action{
					{
						ID:   uuid.FromStringOrNil("f0b5605e-648e-11ec-b318-a7f267cc71fc"),
						Type: action.TypeAnswer,
					},
					{
						ID:   uuid.FromStringOrNil("e2af181a-648e-11ec-878b-2bb6c0cebb3e"),
						Type: action.TypeAMD,
					},
					{
						ID:   uuid.FromStringOrNil("ad108d6a-648e-11ec-a226-536bc1253066"),
						Type: action.TypeTalk,
					},
				},
			},
			&action.Action{
				ID:   uuid.FromStringOrNil("e2af181a-648e-11ec-878b-2bb6c0cebb3e"),
				Type: action.TypeAMD,
			},
		},
		{
			"replace flow has 2 actions",
			uuid.FromStringOrNil("3639f716-648f-11ec-ba9a-3fd10dbd241b"),
			&action.Action{
				ID:     uuid.FromStringOrNil("36679982-648f-11ec-b604-63e47c25e1e7"),
				Type:   action.TypePatchFlow,
				Option: []byte(`{"flow_id": "36e14dae-648f-11ec-b947-6f91a363d29e"}`),
			},
			&activeflow.ActiveFlow{
				Actions: []action.Action{
					{
						ID:   uuid.FromStringOrNil("36900886-648f-11ec-88c7-5bc937041ab5"),
						Type: action.TypeAnswer,
					},
					{
						ID:   uuid.FromStringOrNil("36679982-648f-11ec-b604-63e47c25e1e7"),
						Type: action.TypePatchFlow,
					},
					{
						ID:   uuid.FromStringOrNil("36ba131a-648f-11ec-8a6b-830a37358fbe"),
						Type: action.TypeTalk,
					},
				},
			},
			&flow.Flow{
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

			&activeflow.ActiveFlow{
				Actions: []action.Action{
					{
						ID:   uuid.FromStringOrNil("36900886-648f-11ec-88c7-5bc937041ab5"),
						Type: action.TypeAnswer,
					},
					{
						ID:   uuid.FromStringOrNil("59b5a226-648f-11ec-a356-ff8a386afbb9"),
						Type: action.TypeAMD,
					},
					{
						ID:   uuid.FromStringOrNil("59e09512-648f-11ec-bcec-438ee13c4be1"),
						Type: action.TypeTalk,
					},
					{
						ID:   uuid.FromStringOrNil("36ba131a-648f-11ec-8a6b-830a37358fbe"),
						Type: action.TypeTalk,
					},
				},
			},
			&action.Action{
				ID:   uuid.FromStringOrNil("59b5a226-648f-11ec-a356-ff8a386afbb9"),
				Type: action.TypeAMD,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockDB.EXPECT().ActiveFlowGet(gomock.Any(), tt.callID).Return(tt.resActiveFlow, nil)
			mockDB.EXPECT().FlowGet(gomock.Any(), tt.flow.ID).Return(tt.flow, nil)
			mockDB.EXPECT().ActiveFlowSet(gomock.Any(), tt.expectFlow).Return(nil)

			res, err := h.activeFlowHandleActionPatchFlow(ctx, tt.callID, tt.act)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
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

		expectRes *action.Action
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
			&action.Action{
				ID:   uuid.FromStringOrNil("c74b311c-410c-11ec-84ac-1759f56d04b5"),
				Type: action.TypeAnswer,
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

			res, err := h.activeFlowHandleActionConferenceJoin(ctx, tt.callID, tt.act)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
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

func TestActiveFlowHandleActionQueueJoin(t *testing.T) {
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
		queueID uuid.UUID

		activeFlow   *activeflow.ActiveFlow
		queue        *qmqueue.Queue
		queueFlow    *flow.Flow
		exitActionID uuid.UUID

		expectActiveFlow *activeflow.ActiveFlow

		responseQueuecall *qmqueuecall.Queuecall

		expectRes *action.Action
	}{
		{
			"normal",
			uuid.FromStringOrNil("bee9f0c8-6590-11ec-a927-43fcfbd69db7"),
			&action.Action{
				ID:     uuid.FromStringOrNil("bf1f9cb4-6590-11ec-8502-ffcab16cf0d1"),
				Type:   action.TypeQueueJoin,
				Option: []byte(`{"queue_id": "bf45ea2c-6590-11ec-9a8c-ff92b7ef9aad"}`),
			},
			uuid.FromStringOrNil("bf45ea2c-6590-11ec-9a8c-ff92b7ef9aad"),

			&activeflow.ActiveFlow{
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
			&qmqueue.Queue{
				ID:     uuid.FromStringOrNil("bf45ea2c-6590-11ec-9a8c-ff92b7ef9aad"),
				FlowID: uuid.FromStringOrNil("0f0a4864-6591-11ec-bc0e-db27e08ddec2"),
			},
			&flow.Flow{
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
			uuid.FromStringOrNil("cdd46f0e-6591-11ec-aff5-63bb1f2f2e5f"),

			&activeflow.ActiveFlow{
				Actions: []action.Action{
					{
						ID:   uuid.FromStringOrNil("5de173bc-6592-11ec-bd97-bfe78cdda0f5"),
						Type: action.TypeAnswer,
					},
					{
						ID:   uuid.FromStringOrNil("5e0bb1c2-6592-11ec-ad88-63adb38da11e"),
						Type: action.TypeConfbridgeJoin,
					},
					{
						ID:   uuid.FromStringOrNil("cdd46f0e-6591-11ec-aff5-63bb1f2f2e5f"),
						Type: action.TypeTalk,
					},
				},
			},

			&qmqueuecall.Queuecall{
				ID: uuid.FromStringOrNil("c9002972-6592-11ec-af59-afccad96c5a4"),
			},

			&action.Action{
				ID:   uuid.FromStringOrNil("5de173bc-6592-11ec-bd97-bfe78cdda0f5"),
				Type: action.TypeAnswer,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockReq.EXPECT().QMV1QueueGet(gomock.Any(), tt.queueID).Return(tt.queue, nil)
			mockDB.EXPECT().FlowGet(gomock.Any(), tt.queue.FlowID).Return(tt.queueFlow, nil)
			mockDB.EXPECT().ActiveFlowGet(gomock.Any(), tt.callID).Return(tt.activeFlow, nil)
			mockDB.EXPECT().ActiveFlowSet(gomock.Any(), tt.expectActiveFlow).Return(nil)
			mockReq.EXPECT().QMV1QueueCreateQueuecall(gomock.Any(), tt.queue.ID, gomock.Any(), tt.callID, tt.exitActionID).Return(tt.responseQueuecall, nil)

			res, err := h.activeFlowHandleActionQueueJoin(ctx, tt.callID, tt.act)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}
