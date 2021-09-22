package callhandler

import (
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	callapplication "gitlab.com/voipbin/bin-manager/call-manager.git/models/callapplication"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/externalmedia"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/recording"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/conferencehandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
)

func TestActionExecuteConferenceJoin(t *testing.T) {
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
		name              string
		call              *call.Call
		action            *action.Action
		expectAction      *action.Action
		expectConfereneID uuid.UUID
	}

	tests := []test{
		{
			"empty option",
			&call.Call{},
			&action.Action{
				Type:   action.TypeConferenceJoin,
				Option: []byte(`{}`),
			},
			&action.Action{
				Type:   action.TypeConferenceJoin,
				ID:     uuid.Nil,
				Option: []byte(`{"conference_id":""}`),
			},
			uuid.Nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockDB.EXPECT().CallSetAction(gomock.Any(), tt.call.ID, tt.expectAction).Return(nil)
			mockConf.EXPECT().Join(tt.expectConfereneID, tt.call.ID)
			if err := h.ActionExecute(tt.call, tt.action); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestActionExecuteStreamEcho(t *testing.T) {
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
		name          string
		call          *call.Call
		action        *action.Action
		expectAction  *action.Action
		expectTimeout int
	}

	tests := []test{
		{
			"empty option",
			&call.Call{},
			&action.Action{
				Type:   action.TypeStreamEcho,
				Option: []byte(`{}`),
			},
			&action.Action{
				Type:   action.TypeStreamEcho,
				ID:     uuid.Nil,
				Option: []byte(`{"duration":180000}`),
			},
			180 * 1000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockDB.EXPECT().CallSetAction(gomock.Any(), tt.call.ID, tt.expectAction).Return(nil)
			mockReq.EXPECT().AstChannelContinue(tt.call.AsteriskID, tt.call.ChannelID, "svc-stream_echo", "s", 1, "").Return(nil)
			mockReq.EXPECT().CallCallActionTimeout(tt.call.ID, gomock.Any(), tt.expectAction).Return(nil)
			if err := h.ActionExecute(tt.call, tt.action); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestActionExecuteAnswer(t *testing.T) {
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
		name          string
		call          *call.Call
		action        *action.Action
		expectAction  *action.Action
		expectTimeout int
	}

	tests := []test{
		{
			"empty option",
			&call.Call{
				ID:         uuid.FromStringOrNil("4371b0d6-df48-11ea-9a8c-177968c165e9"),
				AsteriskID: "42:01:0a:a4:00:05",
				ChannelID:  "5b21353a-df48-11ea-8207-6fc0fa36a3fe",
			},
			&action.Action{
				Type:   action.TypeAnswer,
				Option: []byte(`{}`),
			},
			&action.Action{
				Type:   action.TypeAnswer,
				ID:     uuid.Nil,
				Option: []byte(`{}`),
			},
			180 * 1000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockDB.EXPECT().CallSetAction(gomock.Any(), tt.call.ID, tt.expectAction).Return(nil)
			mockReq.EXPECT().AstChannelAnswer(tt.call.AsteriskID, tt.call.ChannelID).Return(nil)
			mockReq.EXPECT().CallCallActionNext(tt.call.ID).Return(nil)
			if err := h.ActionExecute(tt.call, tt.action); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestActionTimeoutNext(t *testing.T) {
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
		call    *call.Call
		action  *action.Action
		channel *channel.Channel
	}

	tests := []test{
		{
			"normal",
			&call.Call{
				ID:         uuid.FromStringOrNil("66e039c6-e3fc-11ea-ae6f-53584373e7c9"),
				AsteriskID: "42:01:0a:a4:00:05",
				ChannelID:  "12a05228-e3fd-11ea-b55f-afd68e7aa755",
				Action: action.Action{
					ID:        uuid.FromStringOrNil("b44bae7a-e3fc-11ea-a908-374a03455628"),
					TMExecute: "2020-04-18T03:22:17.995000",
				},
			},
			&action.Action{
				ID:        uuid.FromStringOrNil("b44bae7a-e3fc-11ea-a908-374a03455628"),
				Type:      action.TypeAnswer,
				Option:    []byte(`{}`),
				TMExecute: "2020-04-18T03:22:17.995000",
			},
			&channel.Channel{
				ID: "12a05228-e3fd-11ea-b55f-afd68e7aa755",
				Data: map[string]interface{}{
					"CONTEXT": "conf-in",
				},
				Stasis: "call-in",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockDB.EXPECT().CallGet(gomock.Any(), tt.call.ID).Return(tt.call, nil)
			mockDB.EXPECT().ChannelGet(gomock.Any(), tt.call.ChannelID).Return(tt.channel, nil)
			mockReq.EXPECT().CallCallActionNext(tt.call.ID).Return(nil)

			if err := h.ActionTimeout(tt.call.ID, tt.action); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestActionExecuteTalk(t *testing.T) {
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
		name           string
		call           *call.Call
		action         *action.Action
		expectSSML     string
		expectGender   string
		expectLanguage string
		filename       string
		expectURI      []string
	}

	tests := []test{
		{
			"normal",
			&call.Call{
				ID:         uuid.FromStringOrNil("5e9f3946-2188-11eb-9d74-bf4bf1239da3"),
				AsteriskID: "42:01:0a:a4:00:05",
				ChannelID:  "61a1345a-2188-11eb-ba52-af82c1239d8f",
			},
			&action.Action{
				Type: action.TypeTalk,
				ID:   uuid.FromStringOrNil("5c9cd6be-2195-11eb-a9c9-bfc91ac88411"),

				Option: []byte(`{"text":"hello world","gender":"male","language":"en-US"}`),
			},
			`hello world`,
			"male",
			"en-US",
			"tts/tmp_filename.wav",
			[]string{"sound:http://localhost:8000/tts/tmp_filename.wav"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB.EXPECT().CallSetAction(gomock.Any(), tt.call.ID, tt.action).Return(nil)
			mockReq.EXPECT().TTSSpeechesPOST(tt.expectSSML, tt.expectGender, tt.expectLanguage).Return(tt.filename, nil)
			mockReq.EXPECT().AstChannelPlay(tt.call.AsteriskID, tt.call.ChannelID, tt.action.ID, tt.expectURI, "").Return(nil)

			if err := h.ActionExecute(tt.call, tt.action); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestActionExecuteRecordingStart(t *testing.T) {
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
		name   string
		call   *call.Call
		action *action.Action
	}

	tests := []test{
		{
			"default",
			&call.Call{
				ID:         uuid.FromStringOrNil("bf4ff828-2a77-11eb-a984-33588027b8c4"),
				AsteriskID: "42:01:0a:a4:00:05",
				ChannelID:  "bfd0e668-2a77-11eb-9993-e72b323b1801",
			},
			&action.Action{
				Type: action.TypeRecordingStart,
				ID:   uuid.FromStringOrNil("c06f25c6-2a77-11eb-bcc8-e3d864a76f78"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB.EXPECT().CallSetAction(gomock.Any(), tt.call.ID, tt.action).Return(nil)
			mockDB.EXPECT().RecordingCreate(gomock.Any(), gomock.Any()).Return(nil)
			mockReq.EXPECT().AstChannelCreateSnoop(tt.call.AsteriskID, tt.call.ChannelID, gomock.Any(), gomock.Any(), channel.SnoopDirectionBoth, channel.SnoopDirectionNone).Return(nil)
			mockDB.EXPECT().CallSetRecordID(gomock.Any(), tt.call.ID, gomock.Any()).Return(nil)
			mockDB.EXPECT().CallAddRecordIDs(gomock.Any(), tt.call.ID, gomock.Any()).Return(nil)
			mockReq.EXPECT().CallCallActionNext(tt.call.ID).Return(nil)
			if err := h.ActionExecute(tt.call, tt.action); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestActionExecuteRecordingStop(t *testing.T) {
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
		name   string
		call   *call.Call
		action *action.Action
		record *recording.Recording
	}

	tests := []test{
		{
			"default",
			&call.Call{
				ID:          uuid.FromStringOrNil("4dde92d0-2b9e-11eb-ad28-f732fd0afed7"),
				AsteriskID:  "42:01:0a:a4:00:05",
				ChannelID:   "5293419a-2b9e-11eb-bfa6-97a4312177f2",
				RecordingID: uuid.FromStringOrNil("b230d160-611f-11eb-9bee-2734cae1cab5"),
			},
			&action.Action{
				Type: action.TypeRecordingStop,
				ID:   uuid.FromStringOrNil("4a3925dc-2b9e-11eb-abb3-d759c4b283d0"),
			},
			&recording.Recording{
				ID:         uuid.FromStringOrNil("b230d160-611f-11eb-9bee-2734cae1cab5"),
				AsteriskID: "42:01:0a:a4:00:05",
				ChannelID:  "fe9354d8-2bb9-11eb-8ad0-9764de384853",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB.EXPECT().CallSetAction(gomock.Any(), tt.call.ID, tt.action).Return(nil)
			mockDB.EXPECT().RecordingGet(gomock.Any(), tt.call.RecordingID).Return(tt.record, nil)
			mockReq.EXPECT().AstChannelHangup(tt.record.AsteriskID, tt.record.ChannelID, ari.ChannelCauseNormalClearing).Return(nil)
			mockReq.EXPECT().CallCallActionNext(tt.call.ID).Return(nil)

			if err := h.ActionExecute(tt.call, tt.action); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestActionExecuteDTMFReceive(t *testing.T) {
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
		call        *call.Call
		storedDTMFs string
		duration    int
		action      *action.Action
	}

	tests := []test{
		{
			"normal",
			&call.Call{
				ID:         uuid.FromStringOrNil("be6ef424-6959-11eb-b70a-9bbd190cd5fd"),
				AsteriskID: "42:01:0a:a4:00:05",
				ChannelID:  "c34e2226-6959-11eb-b57a-8718398e2ffc",
			},
			"",
			1000,
			&action.Action{
				Type:   action.TypeDTMFReceive,
				ID:     uuid.FromStringOrNil("c373b8f6-6959-11eb-b768-df9f393cd216"),
				Option: []byte(`{"duration":1000, "max_number_key": 3}`),
			},
		},
		{
			"finish on key set but not qualified",
			&call.Call{
				ID:         uuid.FromStringOrNil("be6ef424-6959-11eb-b70a-9bbd190cd5fd"),
				AsteriskID: "42:01:0a:a4:00:05",
				ChannelID:  "c34e2226-6959-11eb-b57a-8718398e2ffc",
			},
			"*",
			1000,
			&action.Action{
				Type:   action.TypeDTMFReceive,
				ID:     uuid.FromStringOrNil("c373b8f6-6959-11eb-b768-df9f393cd216"),
				Option: []byte(`{"duration":1000, "max_number_key": 3, "finish_on_key": "1234567"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB.EXPECT().CallSetAction(gomock.Any(), tt.call.ID, tt.action).Return(nil)
			mockDB.EXPECT().CallDTMFGet(gomock.Any(), tt.call.ID).Return("", nil)
			mockReq.EXPECT().CallCallActionTimeout(tt.call.ID, tt.duration, tt.action).Return(nil)

			if err := h.ActionExecute(tt.call, tt.action); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestActionExecuteDTMFReceiveFinishWithStoredDTMFs(t *testing.T) {
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
		call        *call.Call
		storedDTMFs string
		action      *action.Action
	}

	tests := []test{
		{
			"max number key qualified",
			&call.Call{
				ID:         uuid.FromStringOrNil("be6ef424-6959-11eb-b70a-9bbd190cd5fd"),
				AsteriskID: "42:01:0a:a4:00:05",
				ChannelID:  "c34e2226-6959-11eb-b57a-8718398e2ffc",
			},
			"123",
			&action.Action{
				Type:   action.TypeDTMFReceive,
				ID:     uuid.FromStringOrNil("c373b8f6-6959-11eb-b768-df9f393cd216"),
				Option: []byte(`{"duration":1000, "max_number_key": 3}`),
			},
		},
		{
			"finish on key #",
			&call.Call{
				ID:         uuid.FromStringOrNil("be6ef424-6959-11eb-b70a-9bbd190cd5fd"),
				AsteriskID: "42:01:0a:a4:00:05",
				ChannelID:  "c34e2226-6959-11eb-b57a-8718398e2ffc",
			},
			"#",
			&action.Action{
				Type:   action.TypeDTMFReceive,
				ID:     uuid.FromStringOrNil("c373b8f6-6959-11eb-b768-df9f393cd216"),
				Option: []byte(`{"duration":1000, "max_number_key": 3, "finish_on_key": "#"}`),
			},
		},
		{
			"finish on key *",
			&call.Call{
				ID:         uuid.FromStringOrNil("be6ef424-6959-11eb-b70a-9bbd190cd5fd"),
				AsteriskID: "42:01:0a:a4:00:05",
				ChannelID:  "c34e2226-6959-11eb-b57a-8718398e2ffc",
			},
			"*",
			&action.Action{
				Type:   action.TypeDTMFReceive,
				ID:     uuid.FromStringOrNil("c373b8f6-6959-11eb-b768-df9f393cd216"),
				Option: []byte(`{"duration":1000, "max_number_key": 3, "finish_on_key": "1234567*"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB.EXPECT().CallSetAction(gomock.Any(), tt.call.ID, tt.action).Return(nil)
			mockDB.EXPECT().CallDTMFGet(gomock.Any(), tt.call.ID).Return(tt.storedDTMFs, nil)
			mockReq.EXPECT().CallCallActionNext(tt.call.ID).Return(nil)

			if err := h.ActionExecute(tt.call, tt.action); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestActionExecuteDTMFSend(t *testing.T) {
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
		name           string
		call           *call.Call
		action         *action.Action
		expectDtmfs    string
		expectDuration int
		expectInterval int
		expectTimeout  int
	}

	tests := []test{
		{
			"normal",
			&call.Call{
				ID:         uuid.FromStringOrNil("50270fae-69bf-11eb-a0a7-273260ea280c"),
				AsteriskID: "42:01:0a:a4:00:05",
				ChannelID:  "5daefc0e-69bf-11eb-9e3a-b7d9a5988373",
			},
			&action.Action{
				Type:   action.TypeDTMFSend,
				ID:     uuid.FromStringOrNil("508063d8-69bf-11eb-a668-abdbd47ce266"),
				Option: []byte(`{"dtmfs":"12345", "duration": 500, "interval": 500}`),
			},
			"12345",
			500,
			500,
			4500,
		},
		{
			"send 1 dtmf",
			&call.Call{
				ID:         uuid.FromStringOrNil("49a66b38-69c0-11eb-b96c-d799dd21ba8f"),
				AsteriskID: "42:01:0a:a4:00:05",
				ChannelID:  "49e625de-69c0-11eb-891d-db5407ae4982",
			},
			&action.Action{
				Type:   action.TypeDTMFSend,
				ID:     uuid.FromStringOrNil("4a24912a-69c0-11eb-a334-6f8053ede87a"),
				Option: []byte(`{"dtmfs":"1", "duration": 500, "interval": 500}`),
			},
			"1",
			500,
			500,
			500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB.EXPECT().CallSetAction(gomock.Any(), tt.call.ID, tt.action).Return(nil)
			mockReq.EXPECT().AstChannelDTMF(tt.call.AsteriskID, tt.call.ChannelID, tt.expectDtmfs, tt.expectDuration, 0, tt.expectInterval, 0)
			mockReq.EXPECT().CallCallActionTimeout(tt.call.ID, tt.expectTimeout, tt.action)

			if err := h.ActionExecute(tt.call, tt.action); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestActionExecuteExternalMediaStart(t *testing.T) {
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
		name   string
		call   *call.Call
		action *action.Action

		expectHost           string
		expectEncapsulation  string
		expectTransport      string
		expectConnectionType string
		expectFormat         string
		expectDirection      string
	}

	tests := []test{
		{
			"default",
			&call.Call{
				ID:         uuid.FromStringOrNil("3ba00ae0-02f8-11ec-863a-abd78c8246c4"),
				AsteriskID: "42:01:0a:a4:00:05",
				ChannelID:  "4455e2f4-02f8-11ec-acf9-43a391fce607",
			},
			&action.Action{
				Type:   action.TypeExternalMediaStart,
				ID:     uuid.FromStringOrNil("447f0d28-02f8-11ec-bfdb-4bb2407458ce"),
				Option: []byte(`{"external_host":"example.com","encapsulation":"rtp","transport":"udp","connection_type":"client","format":"ulaw","direction":"both","data":""}`),
			},
			"example.com",
			"rtp",
			"udp",
			"client",
			"ulaw",
			"both",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB.EXPECT().CallSetAction(gomock.Any(), tt.call.ID, tt.action).Return(nil)
			mockDB.EXPECT().ExternalMediaGet(gomock.Any(), tt.call.ID).Return(nil, nil)
			mockDB.EXPECT().CallGet(gomock.Any(), tt.call.ID).Return(tt.call, nil)
			mockReq.EXPECT().AstBridgeCreate(tt.call.AsteriskID, gomock.Any(), gomock.Any(), []bridge.Type{bridge.TypeMixing, bridge.TypeProxyMedia}).Return(nil)
			mockReq.EXPECT().AstChannelCreateSnoop(tt.call.AsteriskID, tt.call.ChannelID, gomock.Any(), gomock.Any(), channel.SnoopDirectionBoth, channel.SnoopDirectionBoth).Return(nil)
			mockReq.EXPECT().AstChannelExternalMedia(tt.call.AsteriskID, gomock.Any(), tt.expectHost, tt.expectEncapsulation, tt.expectTransport, tt.expectConnectionType, tt.expectFormat, tt.expectDirection, gomock.Any(), gomock.Any()).Return(&channel.Channel{}, nil)
			mockDB.EXPECT().ExternalMediaSet(gomock.Any(), tt.call.ID, gomock.Any()).Return(nil)
			mockReq.EXPECT().CallCallActionNext(tt.call.ID).Return(nil)
			if err := h.ActionExecute(tt.call, tt.action); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestActionExecuteExternalMediaStop(t *testing.T) {
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
		name     string
		call     *call.Call
		action   *action.Action
		extMedia *externalmedia.ExternalMedia
	}

	tests := []test{
		{
			"default",
			&call.Call{
				ID:         uuid.FromStringOrNil("50b8cb46-1aa5-11ec-9b1e-7b766955c7d1"),
				AsteriskID: "42:01:0a:a4:00:05",
				ChannelID:  "4455e2f4-02f8-11ec-acf9-43a391fce607",
			},
			&action.Action{
				Type: action.TypeExternalMediaStop,
				ID:   uuid.FromStringOrNil("50ff55d4-1aa5-11ec-8d4e-7fc834754547"),
			},
			&externalmedia.ExternalMedia{
				CallID:         uuid.FromStringOrNil("50b8cb46-1aa5-11ec-9b1e-7b766955c7d1"),
				AsteriskID:     "42:01:0a:a4:00:05",
				ChannelID:      "4455e2f4-02f8-11ec-acf9-43a391fce607",
				LocalIP:        "",
				LocalPort:      0,
				ExternalHost:   "",
				Encapsulation:  "",
				Transport:      "",
				ConnectionType: "",
				Format:         "",
				Direction:      "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB.EXPECT().CallSetAction(gomock.Any(), tt.call.ID, tt.action).Return(nil)
			mockDB.EXPECT().ExternalMediaGet(gomock.Any(), tt.call.ID).Return(tt.extMedia, nil)
			mockReq.EXPECT().AstChannelHangup(tt.extMedia.AsteriskID, tt.extMedia.ChannelID, ari.ChannelCauseNormalClearing)
			mockDB.EXPECT().ExternalMediaDelete(gomock.Any(), tt.extMedia.CallID).Return(nil)
			mockReq.EXPECT().CallCallActionNext(tt.call.ID).Return(nil)
			if err := h.ActionExecute(tt.call, tt.action); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestActionExecuteAMD(t *testing.T) {
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
		name   string
		call   *call.Call
		action *action.Action

		expectAMD *callapplication.AMD
	}

	tests := []test{
		{
			"sync false",
			&call.Call{
				ID:         uuid.FromStringOrNil("f607e1b2-19b6-11ec-8304-a33ee590d878"),
				AsteriskID: "42:01:0a:a4:00:05",
				ChannelID:  "f6593184-19b6-11ec-85ee-8bda2a70f32e",
			},
			&action.Action{
				Type:   action.TypeAMD,
				ID:     uuid.FromStringOrNil("f681c108-19b6-11ec-bc57-635de4310a4b"),
				Option: []byte(`{"machine_handle":"hangup"}`),
			},
			&callapplication.AMD{
				CallID:        uuid.FromStringOrNil("f607e1b2-19b6-11ec-8304-a33ee590d878"),
				MachineHandle: "hangup",
			},
		},
		{
			"sync true",
			&call.Call{
				ID:         uuid.FromStringOrNil("7d89362a-19b9-11ec-a1ea-a74ce01d2c9b"),
				AsteriskID: "42:01:0a:a4:00:05",
				ChannelID:  "7da1b4fc-19b9-11ec-948e-7f9ca90957a1",
			},
			&action.Action{
				Type:   action.TypeAMD,
				ID:     uuid.FromStringOrNil("7dba7df2-19b9-11ec-b426-17e356fbf5e3"),
				Option: []byte(`{"machine_handle":"hangup","sync":true}`),
			},
			&callapplication.AMD{
				CallID:        uuid.FromStringOrNil("7d89362a-19b9-11ec-a1ea-a74ce01d2c9b"),
				MachineHandle: "hangup",
				Sync:          true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB.EXPECT().CallSetAction(gomock.Any(), tt.call.ID, tt.action).Return(nil)
			mockReq.EXPECT().AstChannelCreateSnoop(tt.call.AsteriskID, tt.call.ChannelID, gomock.Any(), gomock.Any(), channel.SnoopDirectionBoth, channel.SnoopDirectionBoth).Return(nil)
			mockDB.EXPECT().CallApplicationAMDSet(gomock.Any(), gomock.Any(), tt.expectAMD).Return(nil)

			if tt.expectAMD.Sync == false {
				mockReq.EXPECT().CallCallActionNext(tt.call.ID).Return(nil)
			}
			if err := h.ActionExecute(tt.call, tt.action); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
