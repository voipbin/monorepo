package callhandler

import (
	"encoding/json"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/action"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/arihandler/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/callhandler/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/conferencehandler"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/conferencehandler/models/conference"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/requesthandler"
)

func TestGetService(t *testing.T) {
	type test struct {
		name       string
		channel    *channel.Channel
		expectType call.Type
	}

	tests := []test{
		{
			"normal echo",
			&channel.Channel{
				Data: map[string]interface{}{
					"CONTEXT": contextIncomingCall,
					"DOMAIN":  domainEcho,
				},
			},
			call.TypeEcho,
		},
		{
			"normal conference soft",
			&channel.Channel{
				Data: map[string]interface{}{
					"CONTEXT": contextIncomingCall,
					"DOMAIN":  domainConference,
				},
			},
			call.TypeConference,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := getType(tt.channel)
			if service != tt.expectType {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expectType, service)
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
		name    string
		channel *channel.Channel
		call    *call.Call
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
					"DOMAIN":  "echo.voipbin.net",
				},
			},
			&call.Call{
				ID:         uuid.FromStringOrNil("6611bf7e-92e4-11ea-b658-8313e9bd28f8"),
				AsteriskID: "80:fa:5b:5e:da:81",
				ChannelID:  "f82007c4-92e2-11ea-a3e2-138ed7e90501",
				Type:       call.TypeEcho,
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

			mockConf.EXPECT().Start(conference.TypeEcho, gomock.Any())
			mockReq.EXPECT().CallCallActionTimeout(gomock.Any(), option.Duration, action)

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

func TestTypeSipServiceStart(t *testing.T) {
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
				DestinationNumber: "echo",
			},
			&call.Call{
				ID:         uuid.FromStringOrNil("6611bf7e-92e4-11ea-b658-8313e9bd28f8"),
				AsteriskID: "80:fa:5b:5e:da:81",
				ChannelID:  "f82007c4-92e2-11ea-a3e2-138ed7e90501",
				Type:       call.TypeEcho,
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

			mockConf.EXPECT().Start(conference.TypeEcho, gomock.Any())
			mockReq.EXPECT().CallCallActionTimeout(gomock.Any(), option.Duration, action)

			h.Start(tt.channel)
		})
	}
}
