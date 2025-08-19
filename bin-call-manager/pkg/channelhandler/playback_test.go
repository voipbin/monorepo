package channelhandler

import (
	"context"
	"testing"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-call-manager/pkg/dbhandler"
)

func Test_PlaybackStop(t *testing.T) {

	type test struct {
		name string

		id string

		responseChannel *channel.Channel
		expectRes       *channel.Channel
	}

	tests := []test{
		{
			"has playback id",

			"56142206-a911-11ed-8d5d-c74b3f540ca7",

			&channel.Channel{
				ID:         "56142206-a911-11ed-8d5d-c74b3f540ca7",
				PlaybackID: "56f595f6-a911-11ed-b7d1-7f7aea3d1dcb",
				TMDelete:   dbhandler.DefaultTimeStamp,
			},
			&channel.Channel{
				ID: "56142206-a911-11ed-8d5d-c74b3f540ca7",
			},
		},
		{
			"has empty playback id",

			"9371421e-a911-11ed-9f6f-77336360bb04",

			&channel.Channel{
				ID:         "9371421e-a911-11ed-9f6f-77336360bb04",
				PlaybackID: "",
				TMDelete:   dbhandler.DefaultTimeStamp,
			},
			&channel.Channel{
				ID: "9371421e-a911-11ed-9f6f-77336360bb04",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := channelHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().ChannelGet(gomock.Any(), tt.id).Return(tt.responseChannel, nil)

			if tt.responseChannel.PlaybackID != "" {
				mockReq.EXPECT().AstPlaybackStop(ctx, tt.responseChannel.AsteriskID, tt.responseChannel.PlaybackID).Return(nil)
			}

			if err := h.PlaybackStop(ctx, tt.id); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_Play(t *testing.T) {

	type test struct {
		name string

		id         string
		playbackID string
		medias     []string
		language   string
		offsetms   int
		skipms     int

		responseChannel *channel.Channel
		expectRes       *channel.Channel
	}

	tests := []test{
		{
			name: "normal",

			id:         "b3bb9556-a911-11ed-82b4-5b1a0561bc34",
			playbackID: "b3ecb76c-a911-11ed-ba75-1fd6a1f4a8dc",
			medias: []string{
				"https://test.com/b41231e0-a911-11ed-826d-b783a5e07b3b.wav",
				"https://test.com/b43af49a-a911-11ed-a9b1-1f8d8b922359.wav",
			},
			language: "",
			offsetms: 0,
			skipms:   0,

			responseChannel: &channel.Channel{
				ID:       "b3bb9556-a911-11ed-82b4-5b1a0561bc34",
				TMDelete: dbhandler.DefaultTimeStamp,
			},
			expectRes: &channel.Channel{
				ID: "b3bb9556-a911-11ed-82b4-5b1a0561bc34",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := channelHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().ChannelGet(gomock.Any(), tt.id).Return(tt.responseChannel, nil)
			mockReq.EXPECT().AstChannelPlay(ctx, tt.responseChannel.AsteriskID, tt.responseChannel.ID, tt.medias, tt.language, tt.offsetms, tt.skipms, tt.playbackID).Return(nil)

			if err := h.Play(ctx, tt.id, tt.playbackID, tt.medias, tt.language, tt.offsetms, tt.skipms); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
