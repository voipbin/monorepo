package callhandler

import (
	"encoding/json"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/call-manager/pkg/action"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/callhandler/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/conferencehandler"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/conferencehandler/models/conference"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/eventhandler/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/requesthandler"
)

func TestGetTypeContextIncomingCall(t *testing.T) {
	type test struct {
		name       string
		channel    *channel.Channel
		expectType call.Type
	}

	tests := []test{
		{
			"normal conference",
			&channel.Channel{
				Data: map[string]interface{}{
					"DOMAIN": "conference.voipbin.net",
				},
			},
			call.TypeConference,
		},
		{
			"normal sip-service",
			&channel.Channel{
				Data: map[string]interface{}{
					"DOMAIN": "sip-service.voipbin.net",
				},
			},
			call.TypeSipService,
		},
		{
			"None type",
			&channel.Channel{
				Data: map[string]interface{}{
					"DOMAIN": "randome-invalid-domain.voipbin.net",
				},
			},
			call.TypeNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := getTypeContextIncomingCall(tt.channel)
			if res != tt.expectType {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expectType, res)
			}
		})
	}
}

func TestTypeEchoStart(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockConf := conferencehandler.NewMockConferenceHandler(mc)

	h := &callHandler{
		reqHandler:  mockReq,
		db:          mockDB,
		confHandler: mockConf,
	}

	type test struct {
		name         string
		channel      *channel.Channel
		call         *call.Call
		expectAction *action.Action
		// expectReqConf *conference.Conference
	}

	tests := []test{
		{
			"normal",
			&channel.Channel{
				ID:         "f82007c4-92e2-11ea-a3e2-138ed7e90501",
				AsteriskID: "80:fa:5b:5e:da:81",
				Name:       "PJSIP/in-voipbin-00000948",
				Data: map[string]interface{}{
					"CONTEXT": "call-in",
					"DOMAIN":  "sip-service.voipbin.net",
				},
				DestinationNumber: string(action.TypeEcho),
			},
			&call.Call{
				ID:         uuid.FromStringOrNil("6611bf7e-92e4-11ea-b658-8313e9bd28f8"),
				AsteriskID: "80:fa:5b:5e:da:81",
				ChannelID:  "f82007c4-92e2-11ea-a3e2-138ed7e90501",
				Type:       call.TypeSipService,
				Direction:  call.DirectionIncoming,
			},
			&action.Action{
				ID:     action.IDBegin,
				Type:   action.TypeEcho,
				Option: []byte(`{"duration":180000,"dtmf":true}`),
				Next:   action.IDEnd,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockReq.EXPECT().AstChannelVariableSet(tt.channel.AsteriskID, tt.channel.ID, "TIMEOUT(absolute)", defaultMaxTimeoutEcho).Return(nil)
			mockDB.EXPECT().CallCreate(gomock.Any(), gomock.Any()).Return(nil)
			mockDB.EXPECT().CallSetFlowID(gomock.Any(), gomock.Any(), uuid.Nil).Return(nil)
			mockDB.EXPECT().CallGet(gomock.Any(), gomock.Any()).Return(tt.call, nil)
			mockDB.EXPECT().CallSetAction(gomock.Any(), gomock.Any(), tt.expectAction).Return(nil)

			mockReq.EXPECT().AstChannelContinue(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			mockReq.EXPECT().CallCallActionTimeout(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

			h.Start(tt.channel)
		})
	}
}

func TestTypeConferenceStart(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockConf := conferencehandler.NewMockConferenceHandler(mc)

	h := &callHandler{
		reqHandler:  mockReq,
		db:          mockDB,
		confHandler: mockConf,
	}

	type test struct {
		name       string
		channel    *channel.Channel
		call       *call.Call
		conference *conference.Conference
	}

	tests := []test{
		{
			"normal",
			&channel.Channel{
				ID:         "c08ce47e-9b59-11ea-89c6-f3435f55a6ea",
				AsteriskID: "80:fa:5b:5e:da:81",
				Name:       "PJSIP/in-voipbin-00000999",
				Data: map[string]interface{}{
					"CONTEXT": "call-in",
					"DOMAIN":  "conference.voipbin.net",
				},
				DestinationNumber: "bad943d8-9b59-11ea-b409-4ba263721f17",
			},
			&call.Call{
				ID:         uuid.FromStringOrNil("c6914fcc-9b59-11ea-a5fc-4f4392f10a97"),
				AsteriskID: "80:fa:5b:5e:da:81",
				ChannelID:  "c08ce47e-9b59-11ea-89c6-f3435f55a6ea",
				Type:       call.TypeConference,
				Direction:  call.DirectionIncoming,
			},
			&conference.Conference{
				ID:   uuid.FromStringOrNil("bad943d8-9b59-11ea-b409-4ba263721f17"),
				Type: conference.TypeConference,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockDB.EXPECT().ConferenceGet(gomock.Any(), uuid.FromStringOrNil(tt.channel.DestinationNumber)).Return(tt.conference, nil)
			mockReq.EXPECT().AstChannelVariableSet(tt.channel.AsteriskID, tt.channel.ID, "TIMEOUT(absolute)", defaultMaxTimeoutConference).Return(nil)
			mockDB.EXPECT().CallCreate(gomock.Any(), gomock.Any()).Return(nil)
			mockDB.EXPECT().CallSetFlowID(gomock.Any(), gomock.Any(), uuid.Nil)
			mockDB.EXPECT().CallGet(gomock.Any(), gomock.Any()).Return(tt.call, nil)

			// actionExecuteConferenceJoin part.
			// just pass it.
			mockDB.EXPECT().CallSetAction(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			mockConf.EXPECT().Join(gomock.Any(), gomock.Any()).Return(nil)

			h.Start(tt.channel)
		})
	}
}

func TestTypeSipServiceStartSvcAnswer(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockConf := conferencehandler.NewMockConferenceHandler(mc)

	h := &callHandler{
		reqHandler:  mockReq,
		db:          mockDB,
		confHandler: mockConf,
	}

	type test struct {
		name    string
		channel *channel.Channel
		call    *call.Call
	}

	tests := []test{
		{
			"echo service",
			&channel.Channel{
				ID:         "48a5446a-e3b1-11ea-b837-83239d9eb45f",
				AsteriskID: "80:fa:5b:5e:da:81",
				Name:       "PJSIP/in-voipbin-00000950",
				Data: map[string]interface{}{
					"CONTEXT": "call-in",
					"DOMAIN":  "sip-service.voipbin.net",
				},
				DestinationNumber: string(action.TypeAnswer),
			},
			&call.Call{
				ID:         uuid.FromStringOrNil("4d609f5e-e3b1-11ea-b803-ef0912b904ff"),
				AsteriskID: "80:fa:5b:5e:da:81",
				ChannelID:  "48a5446a-e3b1-11ea-b837-83239d9eb45f",
				Type:       call.TypeSipService,
				Direction:  call.DirectionIncoming,
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
				ID:     action.IDBegin,
				Type:   action.TypeAnswer,
				Option: opt,
				Next:   action.IDEnd,
			}

			mockReq.EXPECT().AstChannelVariableSet(tt.channel.AsteriskID, tt.channel.ID, "TIMEOUT(absolute)", defaultMaxTimeoutEcho).Return(nil)
			mockDB.EXPECT().CallCreate(gomock.Any(), gomock.Any()).Return(nil)
			mockDB.EXPECT().CallSetFlowID(gomock.Any(), gomock.Any(), uuid.Nil).Return(nil)
			mockDB.EXPECT().CallGet(gomock.Any(), gomock.Any()).Return(tt.call, nil)

			mockDB.EXPECT().CallSetAction(gomock.Any(), gomock.Any(), action).Return(nil)
			mockReq.EXPECT().AstChannelAnswer(tt.call.AsteriskID, tt.call.ChannelID).Return(nil)
			mockReq.EXPECT().CallCallActionTimeout(tt.call.ID, 10, action).Return(nil)

			h.Start(tt.channel)
		})
	}
}

func TestTypeSipServiceStartSvcEchoLegacy(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockConf := conferencehandler.NewMockConferenceHandler(mc)

	h := &callHandler{
		reqHandler:  mockReq,
		db:          mockDB,
		confHandler: mockConf,
	}

	type test struct {
		name    string
		channel *channel.Channel
		call    *call.Call
	}

	tests := []test{
		{
			"echo service",
			&channel.Channel{
				ID:         "f82007c4-92e2-11ea-a3e2-138ed7e90501",
				AsteriskID: "80:fa:5b:5e:da:81",
				Name:       "PJSIP/in-voipbin-00000948",
				Data: map[string]interface{}{
					"CONTEXT": "call-in",
					"DOMAIN":  "sip-service.voipbin.net",
				},
				DestinationNumber: string(action.TypeEchoLegacy),
			},
			&call.Call{
				ID:         uuid.FromStringOrNil("6611bf7e-92e4-11ea-b658-8313e9bd28f8"),
				AsteriskID: "80:fa:5b:5e:da:81",
				ChannelID:  "f82007c4-92e2-11ea-a3e2-138ed7e90501",
				Type:       call.TypeSipService,
				Direction:  call.DirectionIncoming,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			option := action.OptionEcho{
				Duration: 180 * 1000,
				DTMF:     true,
			}
			opt, err := json.Marshal(option)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			action := &action.Action{
				ID:     action.IDBegin,
				Type:   action.TypeEchoLegacy,
				Option: opt,
				Next:   action.IDEnd,
			}

			mockReq.EXPECT().AstChannelVariableSet(tt.channel.AsteriskID, tt.channel.ID, "TIMEOUT(absolute)", defaultMaxTimeoutEcho).Return(nil)
			mockDB.EXPECT().CallCreate(gomock.Any(), gomock.Any()).Return(nil)
			mockDB.EXPECT().CallSetFlowID(gomock.Any(), gomock.Any(), uuid.Nil).Return(nil)
			mockDB.EXPECT().CallGet(gomock.Any(), gomock.Any()).Return(tt.call, nil)

			mockDB.EXPECT().CallSetAction(gomock.Any(), gomock.Any(), action).Return(nil)

			mockConf.EXPECT().Start(gomock.Any(), gomock.Any())
			mockReq.EXPECT().CallCallActionTimeout(gomock.Any(), option.Duration, action)

			h.Start(tt.channel)
		})
	}
}

func TestTypeSipServiceStartSvcEcho(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockConf := conferencehandler.NewMockConferenceHandler(mc)

	h := &callHandler{
		reqHandler:  mockReq,
		db:          mockDB,
		confHandler: mockConf,
	}

	type test struct {
		name    string
		channel *channel.Channel
		call    *call.Call
	}

	tests := []test{
		{
			"echo service",
			&channel.Channel{
				ID:         "f82007c4-92e2-11ea-a3e2-138ed7e90501",
				AsteriskID: "80:fa:5b:5e:da:81",
				Name:       "PJSIP/in-voipbin-00000948",
				Data: map[string]interface{}{
					"CONTEXT": "call-in",
					"DOMAIN":  "sip-service.voipbin.net",
				},
				DestinationNumber: string(action.TypeEcho),
			},
			&call.Call{
				ID:         uuid.FromStringOrNil("6611bf7e-92e4-11ea-b658-8313e9bd28f8"),
				AsteriskID: "80:fa:5b:5e:da:81",
				ChannelID:  "f82007c4-92e2-11ea-a3e2-138ed7e90501",
				Type:       call.TypeSipService,
				Direction:  call.DirectionIncoming,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			option := action.OptionEcho{
				Duration: 180 * 1000,
				DTMF:     true,
			}
			opt, err := json.Marshal(option)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			action := &action.Action{
				ID:     action.IDBegin,
				Type:   action.TypeEcho,
				Option: opt,
				Next:   action.IDEnd,
			}

			mockReq.EXPECT().AstChannelVariableSet(tt.channel.AsteriskID, tt.channel.ID, "TIMEOUT(absolute)", defaultMaxTimeoutEcho).Return(nil)
			mockDB.EXPECT().CallCreate(gomock.Any(), gomock.Any()).Return(nil)
			mockDB.EXPECT().CallSetFlowID(gomock.Any(), gomock.Any(), uuid.Nil).Return(nil)
			mockDB.EXPECT().CallGet(gomock.Any(), gomock.Any()).Return(tt.call, nil)

			mockDB.EXPECT().CallSetAction(gomock.Any(), gomock.Any(), action).Return(nil)
			mockReq.EXPECT().AstChannelContinue(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			mockReq.EXPECT().CallCallActionTimeout(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

			h.Start(tt.channel)
		})
	}
}

func TestTypeSipServiceStartSvcStreamEcho(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockConf := conferencehandler.NewMockConferenceHandler(mc)

	h := &callHandler{
		reqHandler:  mockReq,
		db:          mockDB,
		confHandler: mockConf,
	}

	type test struct {
		name         string
		channel      *channel.Channel
		call         *call.Call
		expectAction *action.Action
	}

	tests := []test{
		{
			"normal",
			&channel.Channel{
				ID:         "b1d1bf90-d2b3-11ea-8a02-035ed6a04322",
				AsteriskID: "80:fa:5b:5e:da:81",
				Name:       "PJSIP/in-voipbin-00000948",
				Data: map[string]interface{}{
					"CONTEXT": "call-in",
					"DOMAIN":  "sip-service.voipbin.net",
				},
				DestinationNumber: string(action.TypeStreamEcho),
			},
			&call.Call{
				ID:         uuid.FromStringOrNil("bca4d8c6-d2b3-11ea-b5ba-1fba0632c531"),
				AsteriskID: "80:fa:5b:5e:da:81",
				ChannelID:  "b1d1bf90-d2b3-11ea-8a02-035ed6a04322",
				Type:       call.TypeSipService,
				Direction:  call.DirectionIncoming,
			},
			&action.Action{
				ID:     action.IDBegin,
				Type:   action.TypeStreamEcho,
				Option: []byte(`{"duration":180000}`),
				Next:   action.IDEnd,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockReq.EXPECT().AstChannelVariableSet(tt.channel.AsteriskID, tt.channel.ID, "TIMEOUT(absolute)", defaultMaxTimeoutSipService).Return(nil)
			mockDB.EXPECT().CallCreate(gomock.Any(), gomock.Any()).Return(nil)
			mockDB.EXPECT().CallSetFlowID(gomock.Any(), gomock.Any(), uuid.Nil).Return(nil)
			mockDB.EXPECT().CallGet(gomock.Any(), gomock.Any()).Return(tt.call, nil)
			mockDB.EXPECT().CallSetAction(gomock.Any(), gomock.Any(), tt.expectAction).Return(nil)
			mockReq.EXPECT().AstChannelContinue(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			mockReq.EXPECT().CallCallActionTimeout(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

			h.Start(tt.channel)
		})
	}
}

func TestTypeSipServiceStartSvcConference(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockConf := conferencehandler.NewMockConferenceHandler(mc)

	h := &callHandler{
		reqHandler:  mockReq,
		db:          mockDB,
		confHandler: mockConf,
	}

	type test struct {
		name         string
		channel      *channel.Channel
		call         *call.Call
		expectAction *action.Action
	}

	tests := []test{
		{
			"normal",
			&channel.Channel{
				ID:         "3098c01e-dcee-11ea-b8a3-4be6fd851ab3",
				AsteriskID: "80:fa:5b:5e:da:81",
				Name:       "PJSIP/in-voipbin-00000949",
				Data: map[string]interface{}{
					"CONTEXT": "call-in",
					"DOMAIN":  "sip-service.voipbin.net",
				},
				DestinationNumber: string(action.TypeConferenceJoin),
			},
			&call.Call{
				ID:         uuid.FromStringOrNil("38431800-dcee-11ea-b172-eb53386a16d4"),
				AsteriskID: "80:fa:5b:5e:da:81",
				ChannelID:  "3098c01e-dcee-11ea-b8a3-4be6fd851ab3",
				Type:       call.TypeSipService,
				Direction:  call.DirectionIncoming,
			},
			&action.Action{
				ID:     action.IDBegin,
				Type:   action.TypeConferenceJoin,
				Option: []byte(`{"conference_id":"037a20b9-d11d-4b63-a135-ae230cafd495"}`),
				Next:   action.IDEnd,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockReq.EXPECT().AstChannelVariableSet(tt.channel.AsteriskID, tt.channel.ID, "TIMEOUT(absolute)", defaultMaxTimeoutSipService).Return(nil)
			mockDB.EXPECT().CallCreate(gomock.Any(), gomock.Any()).Return(nil)
			mockDB.EXPECT().CallSetFlowID(gomock.Any(), gomock.Any(), uuid.Nil).Return(nil)
			mockDB.EXPECT().CallGet(gomock.Any(), gomock.Any()).Return(tt.call, nil)
			mockDB.EXPECT().CallSetAction(gomock.Any(), gomock.Any(), tt.expectAction).Return(nil)
			mockConf.EXPECT().Join(uuid.FromStringOrNil("037a20b9-d11d-4b63-a135-ae230cafd495"), tt.call.ID)

			h.Start(tt.channel)
		})
	}
}
