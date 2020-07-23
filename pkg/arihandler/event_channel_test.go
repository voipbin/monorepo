package arihandler

import (
	"testing"

	gomock "github.com/golang/mock/gomock"
	ari "gitlab.com/voipbin/bin-manager/call-manager/pkg/arihandler/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/arihandler/models/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/arihandler/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/callhandler"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/conferencehandler"
	dbhandler "gitlab.com/voipbin/bin-manager/call-manager/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/rabbitmq"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/requesthandler"
)

func TestEventHandlerChannelCreated(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockSock := rabbitmq.NewMockRabbit(mc)
	mockRequest := requesthandler.NewMockRequestHandler(mc)
	mockSvc := callhandler.NewMockCallHandler(mc)

	type test struct {
		name    string
		event   *rabbitmq.Event
		channel *channel.Channel
	}

	tests := []test{
		{
			"normal",

			&rabbitmq.Event{
				Type:     "ari_event",
				DataType: "application/json",
				Data:     `{"type":"ChannelCreated","timestamp":"2020-04-19T14:38:00.363+0000","channel":{"id":"1587307080.49","name":"PJSIP/in-voipbin-00000030","state":"Ring","caller":{"name":"","number":"68025"},"connected":{"name":"","number":""},"accountcode":"","dialplan":{"context":"in-voipbin","exten":"011441332323027","priority":1,"app_name":"","app_data":""},"creationtime":"2020-04-19T14:38:00.363+0000","language":"en"},"asterisk_id":"42:01:0a:a4:00:05","application":"voipbin"}`,
			},
			&channel.Channel{
				AsteriskID: "42:01:0a:a4:00:05",
				ID:         "1587307080.49",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// h := NewARIHandler(mockSock, mockDB, mockRequest, mockSvc)
			h := ariHandler{
				db:          mockDB,
				rabbitSock:  mockSock,
				reqHandler:  mockRequest,
				callHandler: mockSvc,
			}

			cn := &channel.Channel{}
			mockDB.EXPECT().ChannelCreate(gomock.Any(), gomock.AssignableToTypeOf(cn)).Return(nil)
			mockRequest.EXPECT().CallChannelHealth(tt.channel.AsteriskID, tt.channel.ID, gomock.Any(), gomock.Any(), gomock.Any())

			if err := h.processEvent(tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestEventHandlerChannelDestroyed(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockSock := rabbitmq.NewMockRabbit(mc)
	mockRequest := requesthandler.NewMockRequestHandler(mc)
	mockCall := callhandler.NewMockCallHandler(mc)

	type test struct {
		name  string
		event *rabbitmq.Event

		expectAsteriskID string
		expectChannelID  string
		expectTimestamp  string
		expectHangup     ari.ChannelCause
	}

	tests := []test{
		{
			"normal",
			&rabbitmq.Event{
				Type:     "ari_event",
				DataType: "application/json",
				Data:     `{"type":"ChannelDestroyed","timestamp":"2020-04-19T17:02:58.651+0000","cause":42,"cause_txt":"Switching equipment congestion","channel":{"id":"1587315778.885","name":"PJSIP/in-voipbin-00000370","state":"Ring","caller":{"name":"","number":"804"},"connected":{"name":"","number":""},"accountcode":"","dialplan":{"context":"in-voipbin","exten":"00048323395006","priority":1,"app_name":"","app_data":""},"creationtime":"2020-04-19T17:02:58.651+0000","language":"en"},"asterisk_id":"42:01:0a:a4:00:03","application":"voipbin"}`,
			},
			"42:01:0a:a4:00:03",
			"1587315778.885",
			"2020-04-19T17:02:58.651",
			ari.ChannelCauseSwitchCongestion,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// h := NewARIHandler(mockSock, mockDB, mockRequest, mockCall)
			h := ariHandler{
				db:          mockDB,
				rabbitSock:  mockSock,
				reqHandler:  mockRequest,
				callHandler: mockCall,
			}

			cn := &channel.Channel{}
			mockDB.EXPECT().ChannelEnd(gomock.Any(), tt.expectChannelID, tt.expectTimestamp, tt.expectHangup).Return(nil)
			mockDB.EXPECT().ChannelGet(gomock.Any(), tt.expectChannelID).Return(cn, nil)
			mockCall.EXPECT().ARIChannelDestroyed(cn).Return(nil)

			h.processEvent(tt.event)
		})
	}
}

func TestEventHandlerChannelStateChange(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockSock := rabbitmq.NewMockRabbit(mc)
	mockRequest := requesthandler.NewMockRequestHandler(mc)
	mockSvc := callhandler.NewMockCallHandler(mc)

	type test struct {
		name  string
		event *rabbitmq.Event

		expectAsterisID string
		expectChannelID string
		expectTmUpdate  string
		expactState     ari.ChannelState
	}

	tests := []test{
		{
			"normal",
			&rabbitmq.Event{
				Type:     "ari_event",
				DataType: "application/json",
				Data:     `{"type":"ChannelStateChange","timestamp":"2020-04-25T19:17:13.786+0000","channel":{"id":"1587842233.10218","name":"PJSIP/in-voipbin-000026ee","state":"Up","caller":{"name":"","number":"586737682"},"connected":{"name":"","number":""},"accountcode":"","dialplan":{"context":"in-voipbin","exten":"46842002310","priority":2,"app_name":"Stasis","app_data":"voipbin,CONTEXT=in-voipbin,SIP_CALLID=1491366011-850848062-1281392838,SIP_PAI=,SIP_PRIVACY=,DOMAIN=echo.voipbin.net,SOURCE=45.151.255.178"},"creationtime":"2020-04-25T19:17:13.585+0000","language":"en"},"asterisk_id":"42:01:0a:a4:00:05","application":"voipbin"}`,
			},

			"42:01:0a:a4:00:05",
			"1587842233.10218",
			"2020-04-25T19:17:13.786",
			ari.ChannelStateUp,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := ariHandler{
				db:          mockDB,
				rabbitSock:  mockSock,
				reqHandler:  mockRequest,
				callHandler: mockSvc,
			}

			mockDB.EXPECT().ChannelSetState(gomock.Any(), tt.expectChannelID, tt.expectTmUpdate, tt.expactState).Return(nil)
			mockDB.EXPECT().ChannelGet(gomock.Any(), tt.expectChannelID).Return(nil, nil)
			mockSvc.EXPECT().UpdateStatus(nil).Return(nil)

			if err := h.processEvent(tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestEventHandlerChannelEnteredBridge(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockSock := rabbitmq.NewMockRabbit(mc)
	mockRequest := requesthandler.NewMockRequestHandler(mc)
	mockCall := callhandler.NewMockCallHandler(mc)
	mockConf := conferencehandler.NewMockConferenceHandler(mc)

	h := ariHandler{
		db:          mockDB,
		rabbitSock:  mockSock,
		reqHandler:  mockRequest,
		callHandler: mockCall,
		confHandler: mockConf,
	}

	type test struct {
		name    string
		event   *rabbitmq.Event
		channel *channel.Channel
		bridge  *bridge.Bridge
	}

	tests := []test{
		{
			"normal",
			&rabbitmq.Event{
				Type:     "ari_event",
				DataType: "application/json",
				Data:     `{"type":"ChannelEnteredBridge","timestamp":"2020-05-09T10:36:04.595+0000","bridge":{"id":"a6abbe41-2a83-447b-8175-e52e5dea000f","technology":"simple_bridge","bridge_type":"mixing","bridge_class":"stasis","creator":"Stasis","name":"echo","channels":["1589020563.4752","915befe7-7fff-490e-8432-ffe063d5c46d"],"creationtime":"2020-05-09T10:36:04.360+0000","video_mode":"talker"},"channel":{"id":"1589020563.4752","name":"PJSIP/in-voipbin-000008cc","state":"Ring","caller":{"name":"tttt","number":"pchero"},"connected":{"name":"","number":""},"accountcode":"","dialplan":{"context":"in-voipbin","exten":"999999","priority":2,"app_name":"Stasis","app_data":"voipbin,CONTEXT=in-voipbin,SIP_CALLID=B0SUsFI1eo,SIP_PAI=,SIP_PRIVACY=,DOMAIN=echo.voipbin.net,SOURCE=213.127.79.161"},"creationtime":"2020-05-09T10:36:03.792+0000","language":"en"},"asterisk_id":"42:01:0a:a4:00:05","application":"voipbin"}`,
			},
			&channel.Channel{
				ID:         "1589020563.4752",
				AsteriskID: "42:01:0a:a4:00:05",
			},
			&bridge.Bridge{
				ID:         "a6abbe41-2a83-447b-8175-e52e5dea000f",
				AsteriskID: "42:01:0a:a4:00:05",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB.EXPECT().ChannelIsExist(tt.channel.ID, defaultExistTimeout).Return(true)
			mockDB.EXPECT().BridgeIsExist(tt.bridge.ID, defaultExistTimeout).Return(true)
			mockDB.EXPECT().ChannelSetBridgeID(gomock.Any(), tt.channel.ID, tt.bridge.ID)
			mockDB.EXPECT().BridgeAddChannelID(gomock.Any(), tt.bridge.ID, tt.channel.ID)
			mockDB.EXPECT().ChannelGet(gomock.Any(), tt.channel.ID).Return(tt.channel, nil)
			mockDB.EXPECT().BridgeGet(gomock.Any(), tt.bridge.ID).Return(tt.bridge, nil)
			mockConf.EXPECT().ARIChannelEnteredBridge(tt.channel, tt.bridge).Return(nil)

			if err := h.processEvent(tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestEventHandlerChannelLeftBridge(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockSock := rabbitmq.NewMockRabbit(mc)
	mockRequest := requesthandler.NewMockRequestHandler(mc)
	mockCall := callhandler.NewMockCallHandler(mc)
	mockConf := conferencehandler.NewMockConferenceHandler(mc)

	h := ariHandler{
		db:          mockDB,
		rabbitSock:  mockSock,
		reqHandler:  mockRequest,
		callHandler: mockCall,
		confHandler: mockConf,
	}

	type test struct {
		name    string
		event   *rabbitmq.Event
		channel *channel.Channel
		bridge  *bridge.Bridge
	}

	tests := []test{
		{
			"normal",
			&rabbitmq.Event{
				Type:     "ari_event",
				DataType: "application/json",
				Data:     `{"type":"ChannelLeftBridge","timestamp":"2020-05-09T10:53:39.181+0000","bridge":{"id":"a6abbe41-2a83-447b-8175-e52e5dea000f","technology":"simple_bridge","bridge_type":"mixing","bridge_class":"stasis","creator":"Stasis","name":"echo","channels":["915befe7-7fff-490e-8432-ffe063d5c46d"],"creationtime":"2020-05-09T10:36:04.360+0000","video_mode":"talker"},"channel":{"id":"1589020563.4752","name":"PJSIP/in-voipbin-000008cc","state":"Up","caller":{"name":"tttt","number":"pchero"},"connected":{"name":"","number":""},"accountcode":"","dialplan":{"context":"in-voipbin","exten":"999999","priority":2,"app_name":"Stasis","app_data":"voipbin,CONTEXT=in-voipbin,SIP_CALLID=B0SUsFI1eo,SIP_PAI=,SIP_PRIVACY=,DOMAIN=echo.voipbin.net,SOURCE=213.127.79.161"},"creationtime":"2020-05-09T10:36:03.792+0000","language":"en"},"asterisk_id":"42:01:0a:a4:00:05","application":"voipbin"}`,
			},
			&channel.Channel{
				ID:         "1589020563.4752",
				AsteriskID: "42:01:0a:a4:00:05",
			},
			&bridge.Bridge{
				ID:         "a6abbe41-2a83-447b-8175-e52e5dea000f",
				AsteriskID: "42:01:0a:a4:00:05",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB.EXPECT().ChannelIsExist(tt.channel.ID, defaultExistTimeout).Return(true)
			mockDB.EXPECT().BridgeIsExist(tt.bridge.ID, defaultExistTimeout).Return(true)
			mockDB.EXPECT().ChannelSetBridgeID(gomock.Any(), tt.channel.ID, "")
			mockDB.EXPECT().BridgeRemoveChannelID(gomock.Any(), tt.bridge.ID, tt.channel.ID).Return(nil)
			mockDB.EXPECT().ChannelGet(gomock.Any(), tt.channel.ID).Return(tt.channel, nil)
			mockDB.EXPECT().BridgeGet(gomock.Any(), tt.bridge.ID).Return(tt.bridge, nil)
			mockConf.EXPECT().ARIChannelLeftBridge(tt.channel, tt.bridge).Return(nil)

			if err := h.processEvent(tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestEventHandlerChannelDtmfReceived(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockSock := rabbitmq.NewMockRabbit(mc)
	mockRequest := requesthandler.NewMockRequestHandler(mc)
	mockCall := callhandler.NewMockCallHandler(mc)

	type test struct {
		name     string
		event    *rabbitmq.Event
		channel  *channel.Channel
		digit    string
		duration int
	}

	tests := []test{
		{
			"normal",
			&rabbitmq.Event{
				Type:     "ari_event",
				DataType: "application/json",
				Data:     `{"type":"ChannelDtmfReceived","timestamp":"2020-05-20T18:43:32.809+0000","digit":"5","duration_ms":100,"channel":{"id":"1590000197.6557","name":"PJSIP/call-in-0000067a","state":"Up","caller":{"name":"tttt","number":"pchero"},"connected":{"name":"","number":""},"accountcode":"","dialplan":{"context":"call-in","exten":"918298437394","priority":2,"app_name":"Stasis","app_data":"voipbin,CONTEXT=call-in,SIP_CALLID=Oi9M1NhtxK,SIP_PAI=,SIP_PRIVACY=,DOMAIN=echo.voipbin.net,SOURCE=213.127.79.161"},"creationtime":"2020-05-20T18:43:17.822+0000","language":"en"},"asterisk_id":"42:01:0a:a4:00:03","application":"voipbin"}`,
			},
			&channel.Channel{
				ID:         "1590000197.6557",
				AsteriskID: "42:01:0a:a4:00:03",
			},
			"5",
			100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := ariHandler{
				db:          mockDB,
				rabbitSock:  mockSock,
				reqHandler:  mockRequest,
				callHandler: mockCall,
			}

			mockDB.EXPECT().ChannelGet(gomock.Any(), tt.channel.ID).Return(tt.channel, nil)
			mockCall.EXPECT().ARIChannelDtmfReceived(tt.channel, tt.digit, tt.duration).Return(nil)

			if err := h.processEvent(tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
