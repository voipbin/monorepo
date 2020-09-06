package callhandler

import (
	"encoding/json"
	"fmt"
	reflect "reflect"
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
		domain     string
		expectType call.Type
	}

	tests := []test{
		{
			"normal conference",
			"conference.voipbin.net",
			call.TypeConference,
		},
		{
			"normal sip-service",
			"sip-service.voipbin.net",
			call.TypeSipService,
		},
		{
			"None type",
			"randome-invalid-domain.voipbin.net",
			call.TypeNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := getTypeContextIncomingCall(tt.domain)
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
		data         map[string]interface{}
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
			map[string]interface{}{
				"context": "call-in",
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
				ID:     action.IDBegin,
				Type:   action.TypeEcho,
				Option: []byte(`{"duration":180000,"dtmf":true}`),
				Next:   action.IDEnd,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockReq.EXPECT().AstChannelVariableSet(tt.channel.AsteriskID, tt.channel.ID, "VB-TYPE", string(channel.TypeCall)).Return(nil)
			mockReq.EXPECT().AstChannelVariableSet(tt.channel.AsteriskID, tt.channel.ID, "TIMEOUT(absolute)", defaultMaxTimeoutEcho).Return(nil)
			mockDB.EXPECT().CallCreate(gomock.Any(), gomock.Any()).Return(nil)
			mockDB.EXPECT().CallSetFlowID(gomock.Any(), gomock.Any(), uuid.Nil).Return(nil)
			mockDB.EXPECT().CallGet(gomock.Any(), gomock.Any()).Return(tt.call, nil)
			mockDB.EXPECT().CallSetAction(gomock.Any(), gomock.Any(), tt.expectAction).Return(nil)

			mockReq.EXPECT().AstChannelContinue(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			mockReq.EXPECT().CallCallActionTimeout(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

			h.StartCallHandle(tt.channel, tt.data)
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
		data       map[string]interface{}
		call       *call.Call
		conference *conference.Conference
	}

	tests := []test{
		{
			"normal",
			&channel.Channel{
				ID:                "c08ce47e-9b59-11ea-89c6-f3435f55a6ea",
				AsteriskID:        "80:fa:5b:5e:da:81",
				Name:              "PJSIP/in-voipbin-00000999",
				DestinationNumber: "bad943d8-9b59-11ea-b409-4ba263721f17",
			},
			map[string]interface{}{
				"context": "call-in",
				"domain":  "conference.voipbin.net",
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

			mockReq.EXPECT().AstChannelVariableSet(tt.channel.AsteriskID, tt.channel.ID, "VB-TYPE", string(channel.TypeCall)).Return(nil)
			mockDB.EXPECT().ConferenceGet(gomock.Any(), uuid.FromStringOrNil(tt.channel.DestinationNumber)).Return(tt.conference, nil)
			mockReq.EXPECT().AstChannelVariableSet(tt.channel.AsteriskID, tt.channel.ID, "TIMEOUT(absolute)", defaultMaxTimeoutConference).Return(nil)
			mockDB.EXPECT().CallCreate(gomock.Any(), gomock.Any()).Return(nil)
			mockDB.EXPECT().CallSetFlowID(gomock.Any(), gomock.Any(), uuid.Nil)
			mockDB.EXPECT().CallGet(gomock.Any(), gomock.Any()).Return(tt.call, nil)

			// actionExecuteConferenceJoin part.
			// just pass it.
			mockDB.EXPECT().CallSetAction(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			mockConf.EXPECT().Join(gomock.Any(), gomock.Any()).Return(nil)

			h.StartCallHandle(tt.channel, tt.data)
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
		data    map[string]interface{}
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
			map[string]interface{}{
				"context": "call-in",
				"domain":  "sip-service.voipbin.net",
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

			mockReq.EXPECT().AstChannelVariableSet(tt.channel.AsteriskID, tt.channel.ID, "VB-TYPE", string(channel.TypeCall)).Return(nil)
			mockReq.EXPECT().AstChannelVariableSet(tt.channel.AsteriskID, tt.channel.ID, "TIMEOUT(absolute)", defaultMaxTimeoutEcho).Return(nil)
			mockDB.EXPECT().CallCreate(gomock.Any(), gomock.Any()).Return(nil)
			mockDB.EXPECT().CallSetFlowID(gomock.Any(), gomock.Any(), uuid.Nil).Return(nil)
			mockDB.EXPECT().CallGet(gomock.Any(), gomock.Any()).Return(tt.call, nil)

			mockDB.EXPECT().CallSetAction(gomock.Any(), gomock.Any(), action).Return(nil)
			mockReq.EXPECT().AstChannelAnswer(tt.call.AsteriskID, tt.call.ChannelID).Return(nil)
			mockReq.EXPECT().CallCallActionTimeout(tt.call.ID, 10, action).Return(nil)

			h.StartCallHandle(tt.channel, tt.data)
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
		data         map[string]interface{}
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
			map[string]interface{}{
				"context": "call-in",
				"domain":  "sip-service.voipbin.net",
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
			mockReq.EXPECT().AstChannelVariableSet(tt.channel.AsteriskID, tt.channel.ID, "VB-TYPE", string(channel.TypeCall)).Return(nil)
			mockReq.EXPECT().AstChannelVariableSet(tt.channel.AsteriskID, tt.channel.ID, "TIMEOUT(absolute)", defaultMaxTimeoutSipService).Return(nil)
			mockDB.EXPECT().CallCreate(gomock.Any(), gomock.Any()).Return(nil)
			mockDB.EXPECT().CallSetFlowID(gomock.Any(), gomock.Any(), uuid.Nil).Return(nil)
			mockDB.EXPECT().CallGet(gomock.Any(), gomock.Any()).Return(tt.call, nil)
			mockDB.EXPECT().CallSetAction(gomock.Any(), gomock.Any(), tt.expectAction).Return(nil)
			mockReq.EXPECT().AstChannelContinue(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			mockReq.EXPECT().CallCallActionTimeout(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

			h.StartCallHandle(tt.channel, tt.data)
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
		data         map[string]interface{}
		call         *call.Call
		expectAction *action.Action
	}

	tests := []test{
		{
			"normal",
			&channel.Channel{
				ID:                "3098c01e-dcee-11ea-b8a3-4be6fd851ab3",
				AsteriskID:        "80:fa:5b:5e:da:81",
				Name:              "PJSIP/in-voipbin-00000949",
				DestinationNumber: string(action.TypeConferenceJoin),
			},
			map[string]interface{}{
				"context": "call-in",
				"domain":  "sip-service.voipbin.net",
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
			mockReq.EXPECT().AstChannelVariableSet(tt.channel.AsteriskID, tt.channel.ID, "VB-TYPE", string(channel.TypeCall)).Return(nil)
			mockReq.EXPECT().AstChannelVariableSet(tt.channel.AsteriskID, tt.channel.ID, "TIMEOUT(absolute)", defaultMaxTimeoutSipService).Return(nil)
			mockDB.EXPECT().CallCreate(gomock.Any(), gomock.Any()).Return(nil)
			mockDB.EXPECT().CallSetFlowID(gomock.Any(), gomock.Any(), uuid.Nil).Return(nil)
			mockDB.EXPECT().CallGet(gomock.Any(), gomock.Any()).Return(tt.call, nil)
			mockDB.EXPECT().CallSetAction(gomock.Any(), gomock.Any(), tt.expectAction).Return(nil)
			mockConf.EXPECT().Join(uuid.FromStringOrNil("037a20b9-d11d-4b63-a135-ae230cafd495"), tt.call.ID)

			h.StartCallHandle(tt.channel, tt.data)
		})
	}
}

func TestTypeSipServiceStartSvcPlay(t *testing.T) {
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
		data         map[string]interface{}
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
			map[string]interface{}{
				"context": "call-in",
				"domain":  "sip-service.voipbin.net",
			},
			&call.Call{
				ID:         uuid.FromStringOrNil("bc143ba8-e71d-11ea-8a07-9fd9990c98e4"),
				AsteriskID: "80:fa:5b:5e:da:81",
				ChannelID:  "b6721d82-e71d-11ea-a38d-5fa75c625072",
				Type:       call.TypeSipService,
				Direction:  call.DirectionIncoming,
			},
			&action.Action{
				ID:     action.IDBegin,
				Type:   action.TypePlay,
				Option: []byte(`{"stream_url":["https://github.com/pchero/asterisk-medias/raw/master/samples_codec/pcm_samples/example-mono_16bit_8khz_pcm.wav"]}`),
				Next:   action.IDEnd,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockReq.EXPECT().AstChannelAnswer(tt.call.AsteriskID, tt.call.ChannelID)
			mockReq.EXPECT().AstChannelVariableSet(tt.channel.AsteriskID, tt.channel.ID, "VB-TYPE", string(channel.TypeCall)).Return(nil)
			mockReq.EXPECT().AstChannelVariableSet(tt.channel.AsteriskID, tt.channel.ID, "TIMEOUT(absolute)", defaultMaxTimeoutSipService).Return(nil)
			mockDB.EXPECT().CallCreate(gomock.Any(), gomock.Any()).Return(nil)
			mockDB.EXPECT().CallSetFlowID(gomock.Any(), gomock.Any(), uuid.Nil).Return(nil)
			mockDB.EXPECT().CallGet(gomock.Any(), gomock.Any()).Return(tt.call, nil)
			mockDB.EXPECT().CallSetAction(gomock.Any(), gomock.Any(), tt.expectAction).Return(nil)
			mockReq.EXPECT().AstChannelPlay(tt.call.AsteriskID, tt.call.ChannelID, tt.expectAction.ID, gomock.Any()).Return(nil)

			h.StartCallHandle(tt.channel, tt.data)
		})
	}
}

func TestCreateCallOutgoing(t *testing.T) {
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
		name        string
		id          uuid.UUID
		userID      uint64
		flowID      uuid.UUID
		source      call.Address
		destination call.Address

		expectCall *call.Call
	}

	tests := []test{
		{
			"normal",
			uuid.FromStringOrNil("f1afa9ce-ecb2-11ea-ab94-a768ab787da0"),
			1,
			uuid.FromStringOrNil("fd5b3234-ecb2-11ea-8f23-4369cba01ddb"),
			call.Address{
				Type:   call.AddressTypeSIP,
				Name:   "test",
				Target: "testincoming@test.com",
			},
			call.Address{
				Type:   call.AddressTypeSIP,
				Name:   " test target",
				Target: "testoutgoing@test.com",
			},

			&call.Call{
				ID:        uuid.FromStringOrNil("f1afa9ce-ecb2-11ea-ab94-a768ab787da0"),
				UserID:    1,
				ChannelID: call.TestChannelID,
				FlowID:    uuid.FromStringOrNil("fd5b3234-ecb2-11ea-8f23-4369cba01ddb"),
				Type:      call.TypeFlow,
				Status:    call.StatusDialing,
				Direction: call.DirectionOutgoing,
				Source: call.Address{
					Type:   call.AddressTypeSIP,
					Name:   "test",
					Target: "testincoming@test.com",
				},
				Destination: call.Address{
					Type:   call.AddressTypeSIP,
					Name:   " test target",
					Target: "testoutgoing@test.com",
				},
				Action: action.Action{
					ID:   action.IDInit,
					Next: action.IDBegin,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockDB.EXPECT().CallCreate(gomock.Any(), tt.expectCall).Return(nil)
			mockDB.EXPECT().CallGet(gomock.Any(), tt.id).Return(tt.expectCall, nil)
			mockReq.EXPECT().AstChannelCreate(requesthandler.AsteriskIDCall, gomock.Any(), fmt.Sprintf("context=%s", contextOutgoingCall), fmt.Sprintf("pjsip/call-out/sip:%s", tt.destination.Target), "", "", "").Return(nil)

			res, err := h.CreateCallOutgoing(tt.id, tt.userID, tt.flowID, tt.source, tt.destination)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectCall) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectCall, res)
			}
		})
	}
}
