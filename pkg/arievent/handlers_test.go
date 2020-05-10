package arievent

import (
	"testing"

	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/callhandler"
	channel "gitlab.com/voipbin/bin-manager/call-manager/pkg/channel"
	dbhandler "gitlab.com/voipbin/bin-manager/call-manager/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/rabbitmq"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/requesthandler"
)

func TestEventHandlerStasisStart(t *testing.T) {
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
		expactData      map[string]interface{}
		expectStasis    string
	}

	tests := []test{
		{
			"normal",
			&rabbitmq.Event{
				Type:     "ari_event",
				DataType: "application/json",
				Data:     `{"type":"StasisStart","timestamp":"2020-04-25T00:27:18.342+0000","args":["CONTEXT=in-voipbin","SIP_CALLID=8juJJyujlS","SIP_PAI=","SIP_PRIVACY=","DOMAIN=echo.voipbin.net","SOURCE=213.127.79.161"],"channel":{"id":"1587774438.2390","name":"PJSIP/in-voipbin-00000948","state":"Ring","caller":{"name":"tttt","number":"pchero"},"connected":{"name":"","number":""},"accountcode":"","dialplan":{"context":"in-voipbin","exten":"1234234324","priority":2,"app_name":"Stasis","app_data":"voipbin,CONTEXT=in-voipbin,SIP_CALLID=8juJJyujlS,SIP_PAI=,SIP_PRIVACY=,DOMAIN=echo.voipbin.net,SOURCE=213.127.79.161"},"creationtime":"2020-04-25T00:27:18.341+0000","language":"en"},"asterisk_id":"42:01:0a:a4:00:03","application":"voipbin"}`,
			},

			"42:01:0a:a4:00:03",
			"1587774438.2390",
			map[string]interface{}{
				"CONTEXT":     "in-voipbin",
				"DOMAIN":      "echo.voipbin.net",
				"SIP_CALLID":  "8juJJyujlS",
				"SIP_PAI":     "",
				"SIP_PRIVACY": "",
				"SOURCE":      "213.127.79.161",
			},
			"voipbin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewEventHandler(mockSock, mockDB, mockRequest, mockSvc)

			channel := &channel.Channel{
				Data: map[string]interface{}{},
			}
			mockDB.EXPECT().ChannelIsExist(tt.expectChannelID, tt.expectAsterisID, gomock.Any()).Return(true)
			mockDB.EXPECT().ChannelSetDataAndStasis(gomock.Any(), tt.expectAsterisID, tt.expectChannelID, tt.expactData, tt.expectStasis).Return(nil)
			mockDB.EXPECT().ChannelGet(gomock.Any(), tt.expectAsterisID, tt.expectChannelID).Return(channel, nil)
			mockSvc.EXPECT().Start(gomock.Any()).Return(nil)

			if err := h.processEvent(tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestEventHandlerStasisEnd(t *testing.T) {
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
		expectStasis    string
	}

	tests := []test{
		{
			"normal",
			&rabbitmq.Event{
				Type:     "ari_event",
				DataType: "application/json",
				Data:     `{"type":"StasisEnd","timestamp":"2020-05-06T15:36:26.406+0000","channel":{"id":"1588779386.6019","name":"PJSIP/in-voipbin-00000bcb","state":"Up","caller":{"name":"","number":"287"},"connected":{"name":"","number":""},"accountcode":"","dialplan":{"context":"in-voipbin","exten":"0111049442037691478","priority":2,"app_name":"Stasis","app_data":"voipbin,CONTEXT=in-voipbin,SIP_CALLID=522514233-794783407-1635452815,SIP_PAI=,SIP_PRIVACY=,DOMAIN=echo.voipbin.net,SOURCE=103.145.12.63"},"creationtime":"2020-05-06T15:36:26.003+0000","language":"en"},"asterisk_id":"42:01:0a:a4:00:03","application":"voipbin"}`,
			},

			"42:01:0a:a4:00:03",
			"1588779386.6019",
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewEventHandler(mockSock, mockDB, mockRequest, mockSvc)

			mockDB.EXPECT().ChannelSetStasis(gomock.Any(), tt.expectAsterisID, tt.expectChannelID, tt.expectStasis).Return(nil)

			if err := h.processEvent(tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
