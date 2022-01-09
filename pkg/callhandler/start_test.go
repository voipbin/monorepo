package callhandler

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	cfconference "gitlab.com/voipbin/bin-manager/conference-manager.git/models/conference"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/activeflow"
	"gitlab.com/voipbin/bin-manager/number-manager.git/models/number"
	"gitlab.com/voipbin/bin-manager/request-manager.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/address"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/confbridgehandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/notifyhandler"
)

func TestGetTypeContextIncomingCall(t *testing.T) {
	type test struct {
		name       string
		domain     string
		expectType string
	}

	tests := []test{
		{
			"normal conference",
			"conference.voipbin.net",
			domainTypeConference,
		},
		{
			"normal sip-service",
			"sip-service.voipbin.net",
			domainTypeSIPService,
		},
		{
			"pstn",
			"pstn.voipbin.net",
			domainTypePSTN,
		},
		{
			"None type",
			"randome-invalid-domain.voipbin.net",
			domainTypeNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := getDomainTypeIncomingCall(tt.domain)
			if res != tt.expectType {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expectType, res)
			}
		})
	}
}

func TestTypeSipServiceStartSvcEcho(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &callHandler{
		reqHandler:    mockReq,
		notifyHandler: mockNotify,
		db:            mockDB,
	}

	type test struct {
		name         string
		channel      *channel.Channel
		data         map[string]string
		call         *call.Call
		expectAction *action.Action
	}

	tests := []test{
		{
			"normal",
			&channel.Channel{
				ID:                "f82007c4-92e2-11ea-a3e2-138ed7e90501",
				AsteriskID:        "80:fa:5b:5e:da:81",
				Name:              "PJSIP/in-voipbin-00000948",
				DestinationNumber: string(action.TypeEcho),
			},
			map[string]string{
				"context": ContextIncomingCall,
				"domain":  "sip-service.voipbin.net",
			},
			&call.Call{
				ID:         uuid.FromStringOrNil("6611bf7e-92e4-11ea-b658-8313e9bd28f8"),
				AsteriskID: "80:fa:5b:5e:da:81",
				ChannelID:  "f82007c4-92e2-11ea-a3e2-138ed7e90501",
				Type:       call.TypeSipService,
				Direction:  call.DirectionIncoming,
			},
			&action.Action{
				ID:     action.IDStart,
				Type:   action.TypeEcho,
				Option: []byte(`{"duration":180000}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockReq.EXPECT().AstChannelVariableSet(gomock.Any(), tt.channel.AsteriskID, tt.channel.ID, "VB-TYPE", string(channel.TypeCall)).Return(nil)
			mockReq.EXPECT().AstChannelVariableSet(gomock.Any(), tt.channel.AsteriskID, tt.channel.ID, "TIMEOUT(absolute)", defaultMaxTimeoutEcho).Return(nil)
			mockReq.EXPECT().AstBridgeCreate(gomock.Any(), tt.channel.AsteriskID, gomock.Any(), gomock.Any(), []bridge.Type{bridge.TypeMixing, bridge.TypeProxyMedia})
			mockReq.EXPECT().AstBridgeAddChannel(gomock.Any(), tt.channel.AsteriskID, gomock.Any(), tt.channel.ID, "", false, false)
			mockDB.EXPECT().CallCreate(gomock.Any(), gomock.Any()).Return(nil)
			mockDB.EXPECT().CallGet(gomock.Any(), gomock.Any()).Return(tt.call, nil)
			mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), notifyhandler.EventTypeCallCreated, tt.call.WebhookURI, tt.call)
			mockDB.EXPECT().CallSetAction(gomock.Any(), gomock.Any(), tt.expectAction).Return(nil)

			mockReq.EXPECT().AstChannelContinue(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			mockReq.EXPECT().CMV1CallActionTimeout(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

			if err := h.StartCallHandle(context.Background(), tt.channel, tt.data); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestTypeConferenceStart(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &callHandler{
		reqHandler:    mockReq,
		notifyHandler: mockNotify,
		db:            mockDB,
	}

	tests := []struct {
		name       string
		channel    *channel.Channel
		data       map[string]string
		call       *call.Call
		conference *cfconference.Conference
		activeFlow *activeflow.ActiveFlow
	}{
		{
			"normal",
			&channel.Channel{
				ID:                "c08ce47e-9b59-11ea-89c6-f3435f55a6ea",
				AsteriskID:        "80:fa:5b:5e:da:81",
				Name:              "PJSIP/in-voipbin-00000999",
				DestinationNumber: "bad943d8-9b59-11ea-b409-4ba263721f17",
			},
			map[string]string{
				"context": ContextIncomingCall,
				"domain":  "conference.voipbin.net",
			},
			&call.Call{
				ID:         uuid.FromStringOrNil("c6914fcc-9b59-11ea-a5fc-4f4392f10a97"),
				AsteriskID: "80:fa:5b:5e:da:81",
				ChannelID:  "c08ce47e-9b59-11ea-89c6-f3435f55a6ea",
				Type:       call.TypeConference,
				Direction:  call.DirectionIncoming,
				FlowID:     uuid.FromStringOrNil("7d0c1efc-3fe2-11ec-b074-5b80d129f4ed"),
				Action: action.Action{
					ID: action.IDStart,
				},
			},
			&cfconference.Conference{
				ID:     uuid.FromStringOrNil("bad943d8-9b59-11ea-b409-4ba263721f17"),
				Type:   cfconference.TypeConference,
				FlowID: uuid.FromStringOrNil("7d0c1efc-3fe2-11ec-b074-5b80d129f4ed"),
			},
			&activeflow.ActiveFlow{
				CallID: uuid.FromStringOrNil("c6914fcc-9b59-11ea-a5fc-4f4392f10a97"),
				FlowID: uuid.FromStringOrNil("7d0c1efc-3fe2-11ec-b074-5b80d129f4ed"),
				CurrentAction: action.Action{
					ID: uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockReq.EXPECT().AstChannelVariableSet(gomock.Any(), tt.channel.AsteriskID, tt.channel.ID, "VB-TYPE", string(channel.TypeCall)).Return(nil)
			mockReq.EXPECT().AstChannelVariableSet(gomock.Any(), tt.channel.AsteriskID, tt.channel.ID, "TIMEOUT(absolute)", defaultMaxTimeoutConference).Return(nil)
			mockReq.EXPECT().CFV1ConferenceGet(gomock.Any(), uuid.FromStringOrNil(tt.channel.DestinationNumber)).Return(tt.conference, nil)
			mockReq.EXPECT().AstBridgeCreate(gomock.Any(), tt.channel.AsteriskID, gomock.Any(), gomock.Any(), []bridge.Type{bridge.TypeMixing, bridge.TypeProxyMedia})
			mockReq.EXPECT().AstBridgeAddChannel(gomock.Any(), tt.channel.AsteriskID, gomock.Any(), tt.channel.ID, "", false, false)
			mockReq.EXPECT().FMV1ActvieFlowCreate(gomock.Any(), gomock.Any(), tt.conference.FlowID).Return(tt.activeFlow, nil)

			mockDB.EXPECT().CallCreate(gomock.Any(), gomock.Any()).Return(nil)
			mockDB.EXPECT().CallGet(gomock.Any(), gomock.Any()).Return(tt.call, nil)
			mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), notifyhandler.EventTypeCallCreated, tt.call.WebhookURI, tt.call)

			// action next part.
			mockReq.EXPECT().FMV1ActvieFlowGetNextAction(gomock.Any(), gomock.Any(), action.IDStart).Return(&action.Action{Type: action.TypeHangup}, nil)
			mockDB.EXPECT().CallSetAction(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			mockDB.EXPECT().CallSetStatus(gomock.Any(), tt.call.ID, call.StatusTerminating, gomock.Any()).Return(nil)
			mockDB.EXPECT().CallGet(gomock.Any(), tt.call.ID).Return(tt.call, nil)
			mockReq.EXPECT().AstChannelHangup(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

			if err := h.StartCallHandle(context.Background(), tt.channel, tt.data); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestTypeSipServiceStartSvcAnswer(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &callHandler{
		reqHandler:    mockReq,
		notifyHandler: mockNotify,
		db:            mockDB,
	}

	type test struct {
		name    string
		channel *channel.Channel
		data    map[string]string
		call    *call.Call
	}

	tests := []test{
		{
			"answer service",
			&channel.Channel{
				ID:                "48a5446a-e3b1-11ea-b837-83239d9eb45f",
				AsteriskID:        "80:fa:5b:5e:da:81",
				Name:              "PJSIP/in-voipbin-00000950",
				DestinationNumber: string(action.TypeAnswer),
			},
			map[string]string{
				"context": ContextIncomingCall,
				"domain":  "sip-service.voipbin.net",
			},
			&call.Call{
				ID:         uuid.FromStringOrNil("4d609f5e-e3b1-11ea-b803-ef0912b904ff"),
				AsteriskID: "80:fa:5b:5e:da:81",
				ChannelID:  "48a5446a-e3b1-11ea-b837-83239d9eb45f",
				Type:       call.TypeSipService,
				Direction:  call.DirectionIncoming,
				Destination: address.Address{
					Target: string(action.TypeAnswer),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			option := action.OptionAnswer{}
			opt, err := json.Marshal(option)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			action := &action.Action{
				ID:     action.IDStart,
				Type:   action.TypeAnswer,
				Option: opt,
			}

			mockReq.EXPECT().AstChannelVariableSet(gomock.Any(), tt.channel.AsteriskID, tt.channel.ID, "VB-TYPE", string(channel.TypeCall)).Return(nil)
			mockReq.EXPECT().AstChannelVariableSet(gomock.Any(), tt.channel.AsteriskID, tt.channel.ID, "TIMEOUT(absolute)", defaultMaxTimeoutEcho).Return(nil)
			mockReq.EXPECT().AstBridgeCreate(gomock.Any(), tt.channel.AsteriskID, gomock.Any(), gomock.Any(), []bridge.Type{bridge.TypeMixing, bridge.TypeProxyMedia})
			mockReq.EXPECT().AstBridgeAddChannel(gomock.Any(), tt.channel.AsteriskID, gomock.Any(), tt.channel.ID, "", false, false)
			mockDB.EXPECT().CallCreate(gomock.Any(), gomock.Any()).Return(nil)
			mockDB.EXPECT().CallGet(gomock.Any(), gomock.All()).Return(tt.call, nil)
			mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), notifyhandler.EventTypeCallCreated, tt.call.WebhookURI, tt.call)

			mockDB.EXPECT().CallSetAction(gomock.Any(), gomock.Any(), action).Return(nil)
			mockReq.EXPECT().AstChannelAnswer(gomock.Any(), tt.call.AsteriskID, tt.call.ChannelID).Return(nil)
			mockReq.EXPECT().CMV1CallActionNext(gomock.Any(), tt.call.ID, false)

			if err := h.StartCallHandle(context.Background(), tt.channel, tt.data); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestTypeSipServiceStartSvcStreamEcho(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &callHandler{
		reqHandler:    mockReq,
		notifyHandler: mockNotify,
		db:            mockDB,
	}

	type test struct {
		name         string
		channel      *channel.Channel
		data         map[string]string
		call         *call.Call
		expectAction *action.Action
	}

	tests := []test{
		{
			"normal",
			&channel.Channel{
				ID:                "b1d1bf90-d2b3-11ea-8a02-035ed6a04322",
				AsteriskID:        "80:fa:5b:5e:da:81",
				Name:              "PJSIP/in-voipbin-00000948",
				DestinationNumber: string(action.TypeStreamEcho),
			},
			map[string]string{
				"context": ContextIncomingCall,
				"domain":  "sip-service.voipbin.net",
			},
			&call.Call{
				ID:         uuid.FromStringOrNil("bca4d8c6-d2b3-11ea-b5ba-1fba0632c531"),
				AsteriskID: "80:fa:5b:5e:da:81",
				ChannelID:  "b1d1bf90-d2b3-11ea-8a02-035ed6a04322",
				Type:       call.TypeSipService,
				Direction:  call.DirectionIncoming,
				Destination: address.Address{
					Target: string(action.TypeStreamEcho),
				},
			},
			&action.Action{
				ID:     action.IDStart,
				Type:   action.TypeStreamEcho,
				Option: []byte(`{"duration":180000}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockReq.EXPECT().AstChannelVariableSet(gomock.Any(), tt.channel.AsteriskID, tt.channel.ID, "VB-TYPE", string(channel.TypeCall)).Return(nil)
			mockReq.EXPECT().AstChannelVariableSet(gomock.Any(), tt.channel.AsteriskID, tt.channel.ID, "TIMEOUT(absolute)", defaultMaxTimeoutSipService).Return(nil)
			mockReq.EXPECT().AstBridgeCreate(gomock.Any(), tt.channel.AsteriskID, gomock.Any(), gomock.Any(), []bridge.Type{bridge.TypeMixing, bridge.TypeProxyMedia})
			mockReq.EXPECT().AstBridgeAddChannel(gomock.Any(), tt.channel.AsteriskID, gomock.Any(), tt.channel.ID, "", false, false)
			mockDB.EXPECT().CallCreate(gomock.Any(), gomock.Any()).Return(nil)
			mockDB.EXPECT().CallGet(gomock.Any(), gomock.Any()).Return(tt.call, nil)
			mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), notifyhandler.EventTypeCallCreated, tt.call.WebhookURI, tt.call)

			mockDB.EXPECT().CallSetAction(gomock.Any(), gomock.Any(), tt.expectAction).Return(nil)
			mockReq.EXPECT().AstChannelContinue(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			mockReq.EXPECT().CMV1CallActionTimeout(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

			if err := h.StartCallHandle(context.Background(), tt.channel, tt.data); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestTypeSipServiceStartSvcConfbridgeJoin(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockConfbridge := confbridgehandler.NewMockConfbridgeHandler(mc)

	h := &callHandler{
		reqHandler:        mockReq,
		notifyHandler:     mockNotify,
		db:                mockDB,
		confbridgeHandler: mockConfbridge,
	}

	tests := []struct {
		name         string
		channel      *channel.Channel
		data         map[string]string
		call         *call.Call
		activeFlow   *activeflow.ActiveFlow
		expectAction *action.Action
	}{
		{
			"normal",
			&channel.Channel{
				ID:                "3098c01e-dcee-11ea-b8a3-4be6fd851ab3",
				AsteriskID:        "80:fa:5b:5e:da:81",
				Name:              "PJSIP/in-voipbin-00000949",
				DestinationNumber: string(action.TypeConfbridgeJoin),
			},
			map[string]string{
				"context": ContextIncomingCall,
				"domain":  "sip-service.voipbin.net",
			},
			&call.Call{
				ID:         uuid.FromStringOrNil("38431800-dcee-11ea-b172-eb53386a16d4"),
				AsteriskID: "80:fa:5b:5e:da:81",
				ChannelID:  "3098c01e-dcee-11ea-b8a3-4be6fd851ab3",
				FlowID:     uuid.FromStringOrNil("20a32e9c-4129-11ec-b2b4-9735b724208a"),
				Type:       call.TypeSipService,
				Direction:  call.DirectionIncoming,
				Destination: address.Address{
					Target: string(action.TypeConfbridgeJoin),
				},
			},
			&activeflow.ActiveFlow{
				Actions: []action.Action{
					{
						Type: "confbridge_join",
					},
				},
			},
			&action.Action{
				ID:   action.IDStart,
				Type: action.TypeConfbridgeJoin,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockReq.EXPECT().AstChannelVariableSet(gomock.Any(), tt.channel.AsteriskID, tt.channel.ID, "VB-TYPE", string(channel.TypeCall)).Return(nil)
			mockReq.EXPECT().AstChannelVariableSet(gomock.Any(), tt.channel.AsteriskID, tt.channel.ID, "TIMEOUT(absolute)", defaultMaxTimeoutSipService).Return(nil)
			mockReq.EXPECT().AstBridgeCreate(gomock.Any(), tt.channel.AsteriskID, gomock.Any(), gomock.Any(), []bridge.Type{bridge.TypeMixing, bridge.TypeProxyMedia})
			mockReq.EXPECT().AstBridgeAddChannel(gomock.Any(), tt.channel.AsteriskID, gomock.Any(), tt.channel.ID, "", false, false)
			mockDB.EXPECT().CallCreate(gomock.Any(), gomock.Any()).Return(nil)
			mockDB.EXPECT().CallGet(gomock.Any(), gomock.Any()).Return(tt.call, nil)
			mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), notifyhandler.EventTypeCallCreated, tt.call.WebhookURI, tt.call)

			mockDB.EXPECT().CallSetAction(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			mockConfbridge.EXPECT().Join(gomock.Any(), gomock.Any(), tt.call.ID).Return(nil)

			if err := h.StartCallHandle(context.Background(), tt.channel, tt.data); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestTypeSipServiceStartSvcPlay(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &callHandler{
		reqHandler:    mockReq,
		notifyHandler: mockNotify,
		db:            mockDB,
	}

	type test struct {
		name         string
		channel      *channel.Channel
		data         map[string]string
		call         *call.Call
		expectAction *action.Action
	}

	tests := []test{
		{
			"normal",
			&channel.Channel{
				ID:                "b6721d82-e71d-11ea-a38d-5fa75c625072",
				AsteriskID:        "80:fa:5b:5e:da:81",
				Name:              "PJSIP/in-voipbin-00000949",
				DestinationNumber: string(action.TypePlay),
			},
			map[string]string{
				"context": ContextIncomingCall,
				"domain":  "sip-service.voipbin.net",
			},
			&call.Call{
				ID:         uuid.FromStringOrNil("bc143ba8-e71d-11ea-8a07-9fd9990c98e4"),
				AsteriskID: "80:fa:5b:5e:da:81",
				ChannelID:  "b6721d82-e71d-11ea-a38d-5fa75c625072",
				Type:       call.TypeSipService,
				Direction:  call.DirectionIncoming,
				Destination: address.Address{
					Target: string(action.TypePlay),
				},
			},
			&action.Action{
				ID:     action.IDStart,
				Type:   action.TypePlay,
				Option: []byte(`{"stream_urls":["https://github.com/pchero/asterisk-medias/raw/master/samples_codec/pcm_samples/example-mono_16bit_8khz_pcm.wav"]}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockReq.EXPECT().AstChannelAnswer(gomock.Any(), tt.call.AsteriskID, tt.call.ChannelID)
			mockReq.EXPECT().AstChannelVariableSet(gomock.Any(), tt.channel.AsteriskID, tt.channel.ID, "VB-TYPE", string(channel.TypeCall)).Return(nil)
			mockReq.EXPECT().AstChannelVariableSet(gomock.Any(), tt.channel.AsteriskID, tt.channel.ID, "TIMEOUT(absolute)", defaultMaxTimeoutSipService).Return(nil)
			mockReq.EXPECT().AstBridgeCreate(gomock.Any(), tt.channel.AsteriskID, gomock.Any(), gomock.Any(), []bridge.Type{bridge.TypeMixing, bridge.TypeProxyMedia})
			mockReq.EXPECT().AstBridgeAddChannel(gomock.Any(), tt.channel.AsteriskID, gomock.Any(), tt.channel.ID, "", false, false)
			mockDB.EXPECT().CallCreate(gomock.Any(), gomock.Any()).Return(nil)
			mockDB.EXPECT().CallGet(gomock.Any(), gomock.Any()).Return(tt.call, nil)
			mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), notifyhandler.EventTypeCallCreated, tt.call.WebhookURI, tt.call)
			mockDB.EXPECT().CallSetAction(gomock.Any(), gomock.Any(), tt.expectAction).Return(nil)
			mockReq.EXPECT().AstChannelPlay(gomock.Any(), tt.call.AsteriskID, tt.call.ChannelID, tt.expectAction.ID, gomock.Any(), "").Return(nil)

			if err := h.StartCallHandle(context.Background(), tt.channel, tt.data); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestTypeFlowStart(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &callHandler{
		reqHandler:    mockReq,
		notifyHandler: mockNotify,
		db:            mockDB,
	}

	type test struct {
		name    string
		channel *channel.Channel
		data    map[string]string
		numb    *number.Number
		af      *activeflow.ActiveFlow
		call    *call.Call
	}

	tests := []test{
		{
			"normal",
			&channel.Channel{
				ID:                "6e872d74-09ef-11eb-b3a6-37860f73cbd8",
				AsteriskID:        "80:fa:5b:5e:da:81",
				Name:              "PJSIP/in-voipbin-00000949",
				DestinationNumber: "+123456789",
			},
			map[string]string{
				"context": ContextIncomingCall,
				"domain":  "pstn.voipbin.net",
			},
			&number.Number{
				ID:     uuid.FromStringOrNil("bd484f7e-09ef-11eb-9347-377b97e1b9ea"),
				FlowID: uuid.FromStringOrNil("d2e558c2-09ef-11eb-bdec-e3ef3b78ac73"),
				UserID: 1,
			},
			&activeflow.ActiveFlow{
				CallID: uuid.FromStringOrNil("72a902d8-09ef-11eb-92f7-1b906bde6408"),
				FlowID: uuid.FromStringOrNil("d2e558c2-09ef-11eb-bdec-e3ef3b78ac73"),
				CurrentAction: action.Action{
					ID: uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001"),
				},
			},
			&call.Call{
				ID:         uuid.FromStringOrNil("72a902d8-09ef-11eb-92f7-1b906bde6408"),
				AsteriskID: "80:fa:5b:5e:da:81",
				ChannelID:  "6e872d74-09ef-11eb-b3a6-37860f73cbd8",
				FlowID:     uuid.FromStringOrNil("d2e558c2-09ef-11eb-bdec-e3ef3b78ac73"),
				Type:       call.TypeSipService,
				Direction:  call.DirectionIncoming,
				Destination: address.Address{
					Type:   address.TypeTel,
					Target: "+123456789",
				},
				Action: action.Action{
					ID: action.IDStart,
				},
			},
		},
		{
			"webhook",
			&channel.Channel{
				ID:                "f0426396-82e0-11eb-a230-c32d9b79e36a",
				AsteriskID:        "80:fa:5b:5e:da:81",
				Name:              "PJSIP/in-voipbin-00000949",
				DestinationNumber: "+123456789",
			},
			map[string]string{
				"context": ContextIncomingCall,
				"domain":  "pstn.voipbin.net",
			},
			&number.Number{
				ID:     uuid.FromStringOrNil("f06df84e-82e0-11eb-9ca5-7f84ada50218"),
				FlowID: uuid.FromStringOrNil("f08f0ff2-82e0-11eb-8d45-0feb42f4ca6f"),
				UserID: 1,
			},
			&activeflow.ActiveFlow{
				CallID:     uuid.FromStringOrNil("f0ae2504-82e0-11eb-8981-5752f356cf57"),
				FlowID:     uuid.FromStringOrNil("f08f0ff2-82e0-11eb-8d45-0feb42f4ca6f"),
				WebhookURI: "https://test.com/webhook",
				CurrentAction: action.Action{
					ID: uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001"),
				},
			},
			&call.Call{
				ID:         uuid.FromStringOrNil("f0ae2504-82e0-11eb-8981-5752f356cf57"),
				AsteriskID: "80:fa:5b:5e:da:81",
				ChannelID:  "f0426396-82e0-11eb-a230-c32d9b79e36a",
				FlowID:     uuid.FromStringOrNil("f08f0ff2-82e0-11eb-8d45-0feb42f4ca6f"),
				Type:       call.TypeSipService,
				Direction:  call.DirectionIncoming,
				WebhookURI: "https://test.com/webhook",
				Destination: address.Address{
					Type:   address.TypeTel,
					Target: "+123456789",
				},
				Action: action.Action{
					ID: action.IDStart,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockReq.EXPECT().AstChannelVariableSet(gomock.Any(), tt.channel.AsteriskID, tt.channel.ID, "VB-TYPE", string(channel.TypeCall)).Return(nil)
			mockReq.EXPECT().AstChannelVariableSet(gomock.Any(), tt.channel.AsteriskID, tt.channel.ID, "TIMEOUT(absolute)", defaultMaxTimeoutFlow).Return(nil)
			mockReq.EXPECT().NMV1NumberGetByNumber(gomock.Any(), tt.channel.DestinationNumber).Return(tt.numb, nil)
			mockReq.EXPECT().FMV1ActvieFlowCreate(gomock.Any(), gomock.Any(), tt.numb.FlowID).Return(tt.af, nil)
			mockReq.EXPECT().AstBridgeCreate(gomock.Any(), tt.channel.AsteriskID, gomock.Any(), gomock.Any(), []bridge.Type{bridge.TypeMixing, bridge.TypeProxyMedia})
			mockReq.EXPECT().AstBridgeAddChannel(gomock.Any(), tt.channel.AsteriskID, gomock.Any(), tt.channel.ID, "", false, false)
			mockDB.EXPECT().CallCreate(gomock.Any(), gomock.Any()).Return(nil)
			mockDB.EXPECT().CallGet(gomock.Any(), gomock.Any()).Return(tt.call, nil)
			mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), notifyhandler.EventTypeCallCreated, tt.call.WebhookURI, tt.call)

			// action next part.
			mockReq.EXPECT().FMV1ActvieFlowGetNextAction(gomock.Any(), gomock.Any(), action.IDStart).Return(&action.Action{Type: action.TypeHangup}, nil)
			mockDB.EXPECT().CallSetAction(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			mockDB.EXPECT().CallSetStatus(gomock.Any(), tt.call.ID, call.StatusTerminating, gomock.Any()).Return(nil)
			mockDB.EXPECT().CallGet(gomock.Any(), tt.call.ID).Return(tt.call, nil)
			mockReq.EXPECT().AstChannelHangup(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

			if err := h.StartCallHandle(context.Background(), tt.channel, tt.data); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestStartHandlerContextOutgoingCall(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &callHandler{
		reqHandler:    mockReq,
		notifyHandler: mockNotify,
		db:            mockDB,
	}

	type test struct {
		name    string
		channel *channel.Channel
		data    map[string]string
		call    *call.Call
	}

	tests := []test{
		{
			"normal",
			&channel.Channel{
				ID:                "08959a96-8b31-11eb-a5aa-cb0965a824f8",
				AsteriskID:        "80:fa:5b:5e:da:81",
				Name:              "PJSIP/in-voipbin-00000949",
				DestinationNumber: "+123456789",
			},
			map[string]string{
				"context": ContextOutgoingCall,
				"domain":  "pstn.voipbin.net",
				"call_id": "086c90e2-8b31-11eb-b3a0-4ba972148103",
			},
			&call.Call{
				ID:         uuid.FromStringOrNil("086c90e2-8b31-11eb-b3a0-4ba972148103"),
				AsteriskID: "80:fa:5b:5e:da:81",
				ChannelID:  "08959a96-8b31-11eb-a5aa-cb0965a824f8",
				Type:       call.TypeSipService,
				Direction:  call.DirectionIncoming,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB.EXPECT().CallSetAsteriskID(gomock.Any(), tt.call.ID, tt.channel.AsteriskID, gomock.Any()).Return(nil)
			mockReq.EXPECT().AstChannelVariableSet(gomock.Any(), tt.channel.AsteriskID, tt.channel.ID, "VB-TYPE", string(channel.TypeCall)).Return(nil)
			mockReq.EXPECT().AstBridgeCreate(gomock.Any(), tt.channel.AsteriskID, gomock.Any(), gomock.Any(), []bridge.Type{bridge.TypeMixing, bridge.TypeProxyMedia}).Return(nil)
			mockReq.EXPECT().AstBridgeAddChannel(gomock.Any(), tt.channel.AsteriskID, gomock.Any(), tt.channel.ID, "", false, false).Return(nil)
			mockDB.EXPECT().CallSetBridgeID(gomock.Any(), tt.call.ID, gomock.Any()).Return(nil)
			mockReq.EXPECT().AstChannelDial(gomock.Any(), tt.channel.AsteriskID, tt.channel.ID, tt.channel.ID, defaultDialTimeout).Return(nil)

			if err := h.StartCallHandle(context.Background(), tt.channel, tt.data); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestStartHandlerContextExternalMedia(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &callHandler{
		reqHandler:    mockReq,
		notifyHandler: mockNotify,
		db:            mockDB,
	}

	type test struct {
		name     string
		channel  *channel.Channel
		data     map[string]string
		bridgeID string
	}

	tests := []test{
		{
			"normal",
			&channel.Channel{
				ID:         "asterisk-call-5765d977d8-c4k5q-1629605410.6626",
				AsteriskID: "80:fa:5b:5e:da:81",
				Name:       "UnicastRTP/127.0.0.1:5090-0x7f6d54035300",
			},
			map[string]string{
				"context":   ContextExternalMedia,
				"bridge_id": "fab96694-0300-11ec-b4d4-c3bcab7364fd",
				"call_id":   "0648d6c0-0301-11ec-818e-53865044b15c",
			},
			"fab96694-0300-11ec-b4d4-c3bcab7364fd",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockReq.EXPECT().AstChannelVariableSet(gomock.Any(), tt.channel.AsteriskID, tt.channel.ID, "VB-TYPE", string(channel.TypeExternal)).Return(nil)
			mockReq.EXPECT().AstBridgeAddChannel(gomock.Any(), tt.channel.AsteriskID, tt.bridgeID, tt.channel.ID, "", false, false).Return(nil)
			if err := h.StartCallHandle(context.Background(), tt.channel, tt.data); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestStartHandlerContextExternalSnoop(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &callHandler{
		reqHandler:    mockReq,
		notifyHandler: mockNotify,
		db:            mockDB,
	}

	type test struct {
		name     string
		channel  *channel.Channel
		data     map[string]string
		bridgeID string
	}

	tests := []test{
		{
			"normal",
			&channel.Channel{
				ID:         "asterisk-call-5765d977d8-c4k5q-1629607067.6639",
				AsteriskID: "80:fa:5b:5e:da:81",
				Name:       "Snoop/asterisk-call-5765d977d8-c4k5q-1629250154.132-00000000",
			},
			map[string]string{
				"context":   ContextExternalSoop,
				"bridge_id": "d6aecd56-0301-11ec-aee0-77d9356147eb",
				"call_id":   "da646758-0301-11ec-b3eb-f3c05485b756",
			},
			"d6aecd56-0301-11ec-aee0-77d9356147eb",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockReq.EXPECT().AstChannelVariableSet(gomock.Any(), tt.channel.AsteriskID, tt.channel.ID, "VB-TYPE", string(channel.TypeExternal)).Return(nil)
			mockReq.EXPECT().AstBridgeAddChannel(gomock.Any(), tt.channel.AsteriskID, tt.bridgeID, tt.channel.ID, "", false, false).Return(nil)
			if err := h.StartCallHandle(context.Background(), tt.channel, tt.data); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestStartHandlerContextJoin(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &callHandler{
		reqHandler:    mockReq,
		notifyHandler: mockNotify,
		db:            mockDB,
	}

	type test struct {
		name     string
		channel  *channel.Channel
		data     map[string]string
		bridgeID string
	}

	tests := []test{
		{
			"normal",
			&channel.Channel{
				ID:         "asterisk-call-06627464-431a-11ec-bda3-2f0d6128b98f",
				AsteriskID: "80:fa:5b:5e:da:81",
				Name:       "Snoop/asterisk-call-5765d977d8-c4k5q-1629250154.139-00000000",
			},
			map[string]string{
				"context":       ContextJoinCall,
				"bridge_id":     "ed08cbf8-4319-11ec-a768-23af5da287d4",
				"call_id":       "ed4ba266-4319-11ec-80b7-9f3d3acb4aa0",
				"confbridge_id": "ed70a890-4319-11ec-a8dc-8baa3bdb6a39",
			},
			"ed08cbf8-4319-11ec-a768-23af5da287d4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockReq.EXPECT().AstChannelVariableSet(gomock.Any(), tt.channel.AsteriskID, tt.channel.ID, "VB-TYPE", string(channel.TypeJoin)).Return(nil)
			mockReq.EXPECT().AstBridgeAddChannel(gomock.Any(), tt.channel.AsteriskID, tt.bridgeID, tt.channel.ID, "", false, false).Return(nil)

			mockReq.EXPECT().AstChannelVariableSet(gomock.Any(), tt.channel.AsteriskID, tt.channel.ID, "PJSIP_HEADER(add,VB-CALL-ID)", tt.data["call_id"]).Return(nil)
			mockReq.EXPECT().AstChannelVariableSet(gomock.Any(), tt.channel.AsteriskID, tt.channel.ID, "PJSIP_HEADER(add,VB-CONFBRIDGE-ID)", tt.data["confbridge_id"]).Return(nil)

			mockReq.EXPECT().AstChannelDial(gomock.Any(), tt.channel.AsteriskID, tt.channel.ID, "", defaultDialTimeout).Return(nil)

			if err := h.StartCallHandle(context.Background(), tt.channel, tt.data); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
