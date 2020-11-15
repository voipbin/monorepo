package eventhandler

import (
	"fmt"
	"testing"

	gomock "github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/callhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/eventhandler/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

func TestEventHandlerPlaybackFinished(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockSvc := callhandler.NewMockCallHandler(mc)

	h := eventHandler{
		db:          mockDB,
		rabbitSock:  mockSock,
		reqHandler:  mockReq,
		callHandler: mockSvc,
	}

	type test struct {
		name       string
		event      *rabbitmqhandler.Event
		playbackID string
		channel    *channel.Channel
	}

	tests := []test{
		{
			"normal",
			&rabbitmqhandler.Event{
				Type:     "ari_event",
				DataType: "application/json",
				Data:     []byte(`{ "application": "voipbin", "asterisk_id": "42:01:0a:a4:0f:d0", "playback": { "id": "a41baef4-04b9-403d-a9f5-8ea82c8b1749", "language": "en", "media_uri": "sound:/mnt/media/tts/00ad7c95d14643f3f6f61d18acb039e7fedf05ab", "state": "done", "target_uri": "channel:21dccba3-9792-4d57-904d-5260d57cd681" }, "timestamp": "2020-11-15T14:04:49.762+0000", "type": "PlaybackFinished"}`),
			},
			"a41baef4-04b9-403d-a9f5-8ea82c8b1749",
			&channel.Channel{
				AsteriskID: "42:01:0a:a4:0f:d0",
				ID:         "21dccba3-9792-4d57-904d-5260d57cd681",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockReq.EXPECT().AstChannelGet(tt.channel.AsteriskID, tt.channel.ID).Return(tt.channel, nil)
			mockDB.EXPECT().ChannelGet(gomock.Any(), tt.channel.ID).Return(tt.channel, nil)
			mockSvc.EXPECT().ARIPlaybackFinished(tt.channel, tt.playbackID).Return(nil)

			if err := h.processEvent(tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestEventHandlerPlaybackFinishedChannelGone(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockSvc := callhandler.NewMockCallHandler(mc)

	h := eventHandler{
		db:          mockDB,
		rabbitSock:  mockSock,
		reqHandler:  mockReq,
		callHandler: mockSvc,
	}

	type test struct {
		name    string
		event   *rabbitmqhandler.Event
		channel *channel.Channel
	}

	tests := []test{
		{
			"chanel does not exist",
			&rabbitmqhandler.Event{
				Type:     "ari_event",
				DataType: "application/json",
				Data:     []byte(`{ "application": "voipbin", "asterisk_id": "42:01:0a:a4:0f:d0", "playback": { "id": "f85eb320-2757-11eb-a00b-aff2464bde70", "language": "en", "media_uri": "sound:/mnt/media/tts/00ad7c95d14643f3f6f61d18acb039e7fedf05ab", "state": "done", "target_uri": "channel:ec552c6c-2757-11eb-b12c-9f77f7c7cb07" }, "timestamp": "2020-11-15T14:04:49.762+0000", "type": "PlaybackFinished"}`),
			},
			&channel.Channel{
				AsteriskID: "42:01:0a:a4:0f:d0",
				ID:         "ec552c6c-2757-11eb-b12c-9f77f7c7cb07",
				State:      "Down",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockReq.EXPECT().AstChannelGet(tt.channel.AsteriskID, tt.channel.ID).Return(nil, fmt.Errorf("channel does not exist"))

			if err := h.processEvent(tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
