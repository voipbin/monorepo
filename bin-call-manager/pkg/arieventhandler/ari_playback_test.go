package arieventhandler

import (
	"context"
	"testing"

	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	gomock "github.com/golang/mock/gomock"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-call-manager/pkg/callhandler"
	"monorepo/bin-call-manager/pkg/channelhandler"
	"monorepo/bin-call-manager/pkg/dbhandler"
)

func Test_EventHandlerPlaybackStarted(t *testing.T) {

	tests := []struct {
		name  string
		event *ari.PlaybackStarted

		responseChannel  *channel.Channel
		expectchannelID  string
		expectPlaybackID string
	}{
		{
			"normal",
			&ari.PlaybackStarted{
				Event: ari.Event{
					Type:        ari.EventTypePlaybackStarted,
					Application: "voipbin",
					Timestamp:   "2020-11-15T14:04:49.762",
					AsteriskID:  "42:01:0a:a4:0f:d0",
				},
				Playback: ari.Playback{
					ID:        "97bc9516-09ac-497f-b090-955c3c559c91",
					Language:  "en",
					MediaURI:  "sound:/mnt/media/tts/00ad7c95d14643f3f6f61d18acb039e7fedf05ab",
					State:     ari.PlaybackStateDone,
					TargetURI: "channel:6a5ff362-74c5-4bbb-8848-9a5d23d4f170",
				},
			},

			&channel.Channel{
				AsteriskID: "42:01:0a:a4:0f:d0",
				ID:         "6a5ff362-74c5-4bbb-8848-9a5d23d4f170",
				TMEnd:      dbhandler.DefaultTimeStamp,
			},
			"6a5ff362-74c5-4bbb-8848-9a5d23d4f170",
			"97bc9516-09ac-497f-b090-955c3c559c91",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockSock := sockhandler.NewMockSockHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockCall := callhandler.NewMockCallHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)

			h := eventHandler{
				db:             mockDB,
				sockHandler:    mockSock,
				reqHandler:     mockReq,
				callHandler:    mockCall,
				channelHandler: mockChannel,
			}

			ctx := context.Background()

			mockChannel.EXPECT().UpdatePlaybackID(ctx, tt.expectchannelID, tt.expectPlaybackID).Return(tt.responseChannel, nil)
			// mockCall.EXPECT().ARIPlaybackFinished(gomock.Any(), tt.responseChannel, tt.expectPlaybackID).Return(nil)

			if err := h.EventHandlerPlaybackStarted(ctx, tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_EventHandlerPlaybackFinished(t *testing.T) {

	tests := []struct {
		name  string
		event *ari.PlaybackFinished

		responseChannel  *channel.Channel
		expectchannelID  string
		expectPlaybackID string
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

			&channel.Channel{
				AsteriskID: "42:01:0a:a4:0f:d0",
				ID:         "21dccba3-9792-4d57-904d-5260d57cd681",
				TMEnd:      dbhandler.DefaultTimeStamp,
			},
			"21dccba3-9792-4d57-904d-5260d57cd681",
			"a41baef4-04b9-403d-a9f5-8ea82c8b1749",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockSock := sockhandler.NewMockSockHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockCall := callhandler.NewMockCallHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)

			h := eventHandler{
				db:             mockDB,
				sockHandler:    mockSock,
				reqHandler:     mockReq,
				callHandler:    mockCall,
				channelHandler: mockChannel,
			}

			ctx := context.Background()

			mockChannel.EXPECT().UpdatePlaybackID(ctx, tt.expectchannelID, "").Return(tt.responseChannel, nil)
			mockCall.EXPECT().ARIPlaybackFinished(gomock.Any(), tt.responseChannel, tt.expectPlaybackID).Return(nil)

			if err := h.EventHandlerPlaybackFinished(ctx, tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestEventHandlerPlaybackFinishedChannelGone(t *testing.T) {
	type test struct {
		name             string
		event            *ari.PlaybackFinished
		responseChannel  *channel.Channel
		expectchannelID  string
		expectPlaybackID string
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
				TMEnd:      "2023-01-18 03:22:18.995000",
			},
			"ec552c6c-2757-11eb-b12c-9f77f7c7cb07",
			"a41baef4-04b9-403d-a9f5-8ea82c8b1749",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockSock := sockhandler.NewMockSockHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockSvc := callhandler.NewMockCallHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)

			h := eventHandler{
				db:             mockDB,
				sockHandler:    mockSock,
				reqHandler:     mockReq,
				callHandler:    mockSvc,
				channelHandler: mockChannel,
			}

			ctx := context.Background()

			mockChannel.EXPECT().UpdatePlaybackID(ctx, tt.expectchannelID, "").Return(tt.responseChannel, nil)

			if err := h.EventHandlerPlaybackFinished(ctx, tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
