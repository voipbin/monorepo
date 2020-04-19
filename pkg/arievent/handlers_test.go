package arievent

import (
	"testing"

	gomock "github.com/golang/mock/gomock"
	channel "gitlab.com/voipbin/bin-manager/call-manager/pkg/channel"
	dbhandler "gitlab.com/voipbin/bin-manager/call-manager/pkg/db_handler"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/rabbitmq"
)

func TestEventHandlerChannelCreated(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockSock := rabbitmq.NewMockRabbit(mc)

	type test struct {
		name          string
		event         string
		expectChannel channel.Channel
	}

	tests := []test{
		{
			"normal",
			`{"type":"ChannelCreated","timestamp":"2020-04-19T14:38:00.363+0000","channel":{"id":"1587307080.49","name":"PJSIP/in-voipbin-00000030","state":"Ring","caller":{"name":"","number":"68025"},"connected":{"name":"","number":""},"accountcode":"","dialplan":{"context":"in-voipbin","exten":"011441332323027","priority":1,"app_name":"","app_data":""},"creationtime":"2020-04-19T14:38:00.363+0000","language":"en"},"asterisk_id":"42:01:0a:a4:00:05","application":"voipbin"}`,
			channel.Channel{
				AsteriskID: "42:01:0a:a4:00:05",
				ID:         "1587307080.49",
				Name:       "PJSIP/in-voipbin-00000030",
				Tech:       "pjsip",

				SourceName:        "",
				SourceNumber:      "68025",
				DestinationNumber: "011441332323027",

				State: "Ring",

				TMCreate: "2020-04-19T14:38:00.363",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewEventHandler(mockSock, mockDB)

			mockDB.EXPECT().ChannelCreate(gomock.Any(), tt.expectChannel).Return(nil)

			h.processEvent([]byte(tt.event))
		})
	}
}

func TestEventHandlerChannelDestroyed(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockSock := rabbitmq.NewMockRabbit(mc)

	type test struct {
		name  string
		event string

		expectAsteriskID string
		expectChannelID  string
		expectTimestamp  string
		expectHangup     int
	}

	tests := []test{
		{
			"normal",
			`{"type":"ChannelDestroyed","timestamp":"2020-04-19T17:02:58.651+0000","cause":42,"cause_txt":"Switching equipment congestion","channel":{"id":"1587315778.885","name":"PJSIP/in-voipbin-00000370","state":"Ring","caller":{"name":"","number":"804"},"connected":{"name":"","number":""},"accountcode":"","dialplan":{"context":"in-voipbin","exten":"00048323395006","priority":1,"app_name":"","app_data":""},"creationtime":"2020-04-19T17:02:58.651+0000","language":"en"},"asterisk_id":"42:01:0a:a4:00:03","application":"voipbin"}`,
			"42:01:0a:a4:00:03",
			"1587315778.885",
			"2020-04-19T17:02:58.651",
			42,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewEventHandler(mockSock, mockDB)

			mockDB.EXPECT().ChannelEnd(gomock.Any(), tt.expectAsteriskID, tt.expectChannelID, tt.expectTimestamp, tt.expectHangup).Return(nil)

			h.processEvent([]byte(tt.event))
		})
	}
}
