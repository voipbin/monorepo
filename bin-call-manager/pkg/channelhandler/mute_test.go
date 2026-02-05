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

func Test_MuteOn(t *testing.T) {

	type test struct {
		name string

		id        string
		direction channel.MuteDirection

		responseChannel *channel.Channel
	}

	tests := []test{
		{
			"normal",

			"bad221d0-d138-11ed-b162-433a46a331a4",
			channel.MuteDirectionBoth,

			&channel.Channel{
				ID:         "bad221d0-d138-11ed-b162-433a46a331a4",
				AsteriskID: "3e:50:6b:43:bb:30",
				TMDelete:   nil,
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
			mockReq.EXPECT().AstChannelMuteOn(ctx, tt.responseChannel.AsteriskID, tt.responseChannel.ID, string(tt.direction)).Return(nil)
			mockDB.EXPECT().ChannelSetMuteDirection(ctx, tt.id, tt.direction).Return(nil)
			mockDB.EXPECT().ChannelGet(gomock.Any(), tt.id).Return(tt.responseChannel, nil)

			if err := h.MuteOn(ctx, tt.id, tt.direction); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_MuteOff(t *testing.T) {

	type test struct {
		name string

		id            string
		muteDirection channel.MuteDirection

		responseChannel     *channel.Channel
		expectMuteDirection channel.MuteDirection
	}

	tests := []test{
		{
			"normal",

			"bafa3030-d138-11ed-9bdb-9f79a8a1de00",
			channel.MuteDirectionBoth,

			&channel.Channel{
				ID:         "bafa3030-d138-11ed-9bdb-9f79a8a1de00",
				AsteriskID: "3e:50:6b:43:bb:30",
				TMDelete:   nil,
			},
			channel.MuteDirectionNone,
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
			mockReq.EXPECT().AstChannelMuteOff(ctx, tt.responseChannel.AsteriskID, tt.responseChannel.ID, string(tt.muteDirection)).Return(nil)
			mockDB.EXPECT().ChannelSetMuteDirection(ctx, tt.id, tt.expectMuteDirection).Return(nil)
			mockDB.EXPECT().ChannelGet(gomock.Any(), tt.id).Return(tt.responseChannel, nil)

			if err := h.MuteOff(ctx, tt.id, tt.muteDirection); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
