package arievent

import (
	"testing"

	gomock "github.com/golang/mock/gomock"
	ari "gitlab.com/voipbin/bin-manager/call-manager/pkg/ari"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/arirequest"
	channel "gitlab.com/voipbin/bin-manager/call-manager/pkg/channel"
	dbhandler "gitlab.com/voipbin/bin-manager/call-manager/pkg/db_handler"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/rabbitmq"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/svchandler"
)

func TestEventHandlerChannelCreated(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockSock := rabbitmq.NewMockRabbit(mc)
	mockRequest := arirequest.NewMockRequestHandler(mc)
	mockSvc := svchandler.NewMockSVCHandler(mc)

	type test struct {
		name  string
		event string
	}

	tests := []test{
		{
			"normal",
			`{"type":"ChannelCreated","timestamp":"2020-04-19T14:38:00.363+0000","channel":{"id":"1587307080.49","name":"PJSIP/in-voipbin-00000030","state":"Ring","caller":{"name":"","number":"68025"},"connected":{"name":"","number":""},"accountcode":"","dialplan":{"context":"in-voipbin","exten":"011441332323027","priority":1,"app_name":"","app_data":""},"creationtime":"2020-04-19T14:38:00.363+0000","language":"en"},"asterisk_id":"42:01:0a:a4:00:05","application":"voipbin"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewEventHandler(mockSock, mockDB, mockRequest, mockSvc)

			cn := &channel.Channel{}
			mockDB.EXPECT().ChannelCreate(gomock.Any(), gomock.AssignableToTypeOf(cn)).Return(nil)

			if err := h.processEvent([]byte(tt.event)); err != nil {
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
	mockRequest := arirequest.NewMockRequestHandler(mc)
	mockSvc := svchandler.NewMockSVCHandler(mc)

	type test struct {
		name  string
		event string

		expectAsteriskID string
		expectChannelID  string
		expectTimestamp  string
		expectHangup     ari.ChannelCause
	}

	tests := []test{
		{
			"normal",
			`{"type":"ChannelDestroyed","timestamp":"2020-04-19T17:02:58.651+0000","cause":42,"cause_txt":"Switching equipment congestion","channel":{"id":"1587315778.885","name":"PJSIP/in-voipbin-00000370","state":"Ring","caller":{"name":"","number":"804"},"connected":{"name":"","number":""},"accountcode":"","dialplan":{"context":"in-voipbin","exten":"00048323395006","priority":1,"app_name":"","app_data":""},"creationtime":"2020-04-19T17:02:58.651+0000","language":"en"},"asterisk_id":"42:01:0a:a4:00:03","application":"voipbin"}`,
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

			h.processEvent([]byte(tt.event))
		})
	}
}

func TestEventHandlerStasisStart(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockSock := rabbitmq.NewMockRabbit(mc)
	mockRequest := arirequest.NewMockRequestHandler(mc)
	mockSvc := svchandler.NewMockSVCHandler(mc)

	type test struct {
		name            string
		event           string
		expectAsterisID string
		expectChannelID string
		expectTmUpdate  string
		expactData      map[string]interface{}
	}

	tests := []test{
		{
			"normal",
			`{"type":"StasisStart","timestamp":"2020-04-25T00:27:18.342+0000","args":["CONTEXT=in-voipbin","SIP_CALLID=8juJJyujlS","SIP_PAI=","SIP_PRIVACY=","DOMAIN=echo.voipbin.net","SOURCE=213.127.79.161"],"channel":{"id":"1587774438.2390","name":"PJSIP/in-voipbin-00000948","state":"Ring","caller":{"name":"tttt","number":"pchero"},"connected":{"name":"","number":""},"accountcode":"","dialplan":{"context":"in-voipbin","exten":"1234234324","priority":2,"app_name":"Stasis","app_data":"voipbin,CONTEXT=in-voipbin,SIP_CALLID=8juJJyujlS,SIP_PAI=,SIP_PRIVACY=,DOMAIN=echo.voipbin.net,SOURCE=213.127.79.161"},"creationtime":"2020-04-25T00:27:18.341+0000","language":"en"},"asterisk_id":"42:01:0a:a4:00:03","application":"voipbin"}`,
			"42:01:0a:a4:00:03",
			"1587774438.2390",
			"2020-04-25T00:27:18.342",
			map[string]interface{}{
				"CONTEXT":     "in-voipbin",
				"DOMAIN":      "echo.voipbin.net",
				"SIP_CALLID":  "8juJJyujlS",
				"SIP_PAI":     "",
				"SIP_PRIVACY": "",
				"SOURCE":      "213.127.79.161",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewEventHandler(mockSock, mockDB, mockRequest, mockSvc)

			channel := &channel.Channel{
				Data: map[string]interface{}{},
			}
			mockDB.EXPECT().ChannelGet(gomock.Any(), tt.expectAsterisID, tt.expectChannelID).Return(channel, nil)
			mockDB.EXPECT().ChannelSetData(gomock.Any(), tt.expectAsterisID, tt.expectChannelID, tt.expectTmUpdate, tt.expactData).Return(nil)
			mockSvc.EXPECT().Start(gomock.Any()).Return(nil)

			if err := h.processEvent([]byte(tt.event)); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
