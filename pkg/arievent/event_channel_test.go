package arievent

import (
	"testing"

	gomock "github.com/golang/mock/gomock"
	ari "gitlab.com/voipbin/bin-manager/call-manager/pkg/ari"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/channel"
	dbhandler "gitlab.com/voipbin/bin-manager/call-manager/pkg/db_handler"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/rabbitmq"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/svchandler"
)

func TestEventHandlerChannelCreated(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockSock := rabbitmq.NewMockRabbit(mc)
	mockRequest := requesthandler.NewMockRequestHandler(mc)
	mockSvc := svchandler.NewMockSVCHandler(mc)

	type test struct {
		name  string
		event *rabbitmq.Event
	}

	tests := []test{
		{
			"normal",

			&rabbitmq.Event{
				Type:     "ari_event",
				DataType: "application/json",
				Data:     `{"type":"ChannelCreated","timestamp":"2020-04-19T14:38:00.363+0000","channel":{"id":"1587307080.49","name":"PJSIP/in-voipbin-00000030","state":"Ring","caller":{"name":"","number":"68025"},"connected":{"name":"","number":""},"accountcode":"","dialplan":{"context":"in-voipbin","exten":"011441332323027","priority":1,"app_name":"","app_data":""},"creationtime":"2020-04-19T14:38:00.363+0000","language":"en"},"asterisk_id":"42:01:0a:a4:00:05","application":"voipbin"}`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewEventHandler(mockSock, mockDB, mockRequest, mockSvc)

			cn := &channel.Channel{}
			mockDB.EXPECT().ChannelCreate(gomock.Any(), gomock.AssignableToTypeOf(cn)).Return(nil)

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
	mockSvc := svchandler.NewMockSVCHandler(mc)

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
			h := NewEventHandler(mockSock, mockDB, mockRequest, mockSvc)

			cn := &channel.Channel{}
			mockDB.EXPECT().ChannelEnd(gomock.Any(), tt.expectAsteriskID, tt.expectChannelID, tt.expectTimestamp, tt.expectHangup).Return(nil)
			mockDB.EXPECT().ChannelGet(gomock.Any(), tt.expectAsteriskID, tt.expectChannelID).Return(cn, nil)
			mockSvc.EXPECT().Hangup(cn).Return(nil)

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
	mockSvc := svchandler.NewMockSVCHandler(mc)

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
			h := NewEventHandler(mockSock, mockDB, mockRequest, mockSvc)

			mockDB.EXPECT().ChannelSetState(gomock.Any(), tt.expectAsterisID, tt.expectChannelID, tt.expectTmUpdate, tt.expactState).Return(nil)
			mockDB.EXPECT().ChannelGet(gomock.Any(), tt.expectAsterisID, tt.expectChannelID).Return(nil, nil)
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
	mockSvc := svchandler.NewMockSVCHandler(mc)

	type test struct {
		name  string
		event *rabbitmq.Event

		expectAsterisID string
		expectChannelID string
		expectBridgeID  string
	}

	tests := []test{
		{
			"normal",
			&rabbitmq.Event{
				Type:     "ari_event",
				DataType: "application/json",
				Data:     `{"type":"ChannelEnteredBridge","timestamp":"2020-05-09T10:36:04.595+0000","bridge":{"id":"a6abbe41-2a83-447b-8175-e52e5dea000f","technology":"simple_bridge","bridge_type":"mixing","bridge_class":"stasis","creator":"Stasis","name":"echo","channels":["1589020563.4752","915befe7-7fff-490e-8432-ffe063d5c46d"],"creationtime":"2020-05-09T10:36:04.360+0000","video_mode":"talker"},"channel":{"id":"1589020563.4752","name":"PJSIP/in-voipbin-000008cc","state":"Ring","caller":{"name":"tttt","number":"pchero"},"connected":{"name":"","number":""},"accountcode":"","dialplan":{"context":"in-voipbin","exten":"999999","priority":2,"app_name":"Stasis","app_data":"voipbin,CONTEXT=in-voipbin,SIP_CALLID=B0SUsFI1eo,SIP_PAI=,SIP_PRIVACY=,DOMAIN=echo.voipbin.net,SOURCE=213.127.79.161"},"creationtime":"2020-05-09T10:36:03.792+0000","language":"en"},"asterisk_id":"42:01:0a:a4:00:05","application":"voipbin"}`,
			},

			"42:01:0a:a4:00:05",
			"1589020563.4752",
			"a6abbe41-2a83-447b-8175-e52e5dea000f",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewEventHandler(mockSock, mockDB, mockRequest, mockSvc)

			mockDB.EXPECT().ChannelSetBridgeID(gomock.Any(), tt.expectAsterisID, tt.expectChannelID, tt.expectBridgeID)
			mockDB.EXPECT().BridgeAddChannelID(gomock.Any(), tt.expectBridgeID, tt.expectChannelID)

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
	mockSvc := svchandler.NewMockSVCHandler(mc)

	type test struct {
		name  string
		event *rabbitmq.Event

		expectAsterisID string
		expectChannelID string
		expectBridgeID  string
	}

	tests := []test{
		{
			"normal",
			&rabbitmq.Event{
				Type:     "ari_event",
				DataType: "application/json",
				Data:     `{"type":"ChannelLeftBridge","timestamp":"2020-05-09T10:53:39.181+0000","bridge":{"id":"a6abbe41-2a83-447b-8175-e52e5dea000f","technology":"simple_bridge","bridge_type":"mixing","bridge_class":"stasis","creator":"Stasis","name":"echo","channels":["915befe7-7fff-490e-8432-ffe063d5c46d"],"creationtime":"2020-05-09T10:36:04.360+0000","video_mode":"talker"},"channel":{"id":"1589020563.4752","name":"PJSIP/in-voipbin-000008cc","state":"Up","caller":{"name":"tttt","number":"pchero"},"connected":{"name":"","number":""},"accountcode":"","dialplan":{"context":"in-voipbin","exten":"999999","priority":2,"app_name":"Stasis","app_data":"voipbin,CONTEXT=in-voipbin,SIP_CALLID=B0SUsFI1eo,SIP_PAI=,SIP_PRIVACY=,DOMAIN=echo.voipbin.net,SOURCE=213.127.79.161"},"creationtime":"2020-05-09T10:36:03.792+0000","language":"en"},"asterisk_id":"42:01:0a:a4:00:05","application":"voipbin"}`,
			},

			"42:01:0a:a4:00:05",
			"1589020563.4752",
			"a6abbe41-2a83-447b-8175-e52e5dea000f",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewEventHandler(mockSock, mockDB, mockRequest, mockSvc)

			mockDB.EXPECT().ChannelSetBridgeID(gomock.Any(), tt.expectAsterisID, tt.expectChannelID, "")
			mockDB.EXPECT().BridgeRemoveChannelID(gomock.Any(), tt.expectBridgeID, tt.expectChannelID)

			if err := h.processEvent(tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
