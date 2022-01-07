package arieventhandler

import (
	"context"
	"fmt"
	"testing"

	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/request-manager.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/callhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
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

	tests := []struct {
		name       string
		event      *ari.PlaybackFinished
		playbackID string
		channel    *channel.Channel
	}{
		{
			"normal",
			&ari.PlaybackFinished{
				Event: ari.Event{
					Type:        ari.EventTypePlaybackFinished,
					Application: "voipbin",
					Timestamp:   "2020-11-15T14:04:49.762",
					AsteriskID:  "42:01:0a:a4:0f:d0",
				},
				Playback: ari.Playback{
					ID:        "a41baef4-04b9-403d-a9f5-8ea82c8b1749",
					Language:  "en",
					MediaURI:  "sound:/mnt/media/tts/00ad7c95d14643f3f6f61d18acb039e7fedf05ab",
					State:     ari.PlaybackStateDone,
					TargetURI: "channel:21dccba3-9792-4d57-904d-5260d57cd681",
				},
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
			ctx := context.Background()

			mockDB.EXPECT().ChannelSetPlaybackID(gomock.Any(), tt.channel.ID, "").Return(nil)
			mockReq.EXPECT().AstChannelGet(gomock.Any(), tt.channel.AsteriskID, tt.channel.ID).Return(tt.channel, nil)
			mockDB.EXPECT().ChannelGet(gomock.Any(), tt.channel.ID).Return(tt.channel, nil)
			mockSvc.EXPECT().ARIPlaybackFinished(gomock.Any(), tt.channel, tt.playbackID).Return(nil)

			if err := h.EventHandlerPlaybackFinished(ctx, tt.event); err != nil {
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
		event   *ari.PlaybackFinished
		channel *channel.Channel
	}

	tests := []test{
		{
			"chanel does not exist",
			&ari.PlaybackFinished{
				Event: ari.Event{
					Type:        ari.EventTypePlaybackFinished,
					Application: "voipbin",
					Timestamp:   "2020-11-15T14:04:49.762",
					AsteriskID:  "42:01:0a:a4:0f:d0",
				},
				Playback: ari.Playback{
					ID:        "a41baef4-04b9-403d-a9f5-8ea82c8b1749",
					Language:  "en",
					MediaURI:  "sound:/mnt/media/tts/00ad7c95d14643f3f6f61d18acb039e7fedf05ab",
					State:     ari.PlaybackStateDone,
					TargetURI: "channel:ec552c6c-2757-11eb-b12c-9f77f7c7cb07",
				},
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
			ctx := context.Background()

			mockDB.EXPECT().ChannelSetPlaybackID(gomock.Any(), tt.channel.ID, "").Return(nil)
			mockReq.EXPECT().AstChannelGet(gomock.Any(), tt.channel.AsteriskID, tt.channel.ID).Return(nil, fmt.Errorf("channel does not exist"))

			if err := h.EventHandlerPlaybackFinished(ctx, tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
