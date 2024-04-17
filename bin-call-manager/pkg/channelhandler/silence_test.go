package channelhandler

import (
	"context"
	"testing"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	gomock "github.com/golang/mock/gomock"

	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-call-manager/pkg/dbhandler"
)

func Test_SilenceOn(t *testing.T) {

	type test struct {
		name string

		id string

		responseChannel *channel.Channel
	}

	tests := []test{
		{
			"normal",

			"0166c4b6-d139-11ed-b354-7ffad3f2c2fb",

			&channel.Channel{
				ID:         "0166c4b6-d139-11ed-b354-7ffad3f2c2fb",
				AsteriskID: "3e:50:6b:43:bb:30",
				TMDelete:   dbhandler.DefaultTimeStamp,
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
			mockReq.EXPECT().AstChannelSilenceOn(ctx, tt.responseChannel.AsteriskID, tt.responseChannel.ID).Return(nil)

			if err := h.SilenceOn(ctx, tt.id); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_SilenceOff(t *testing.T) {

	type test struct {
		name string

		id string

		responseChannel *channel.Channel
	}

	tests := []test{
		{
			"normal",

			"0191d4f8-d139-11ed-8f50-833af1cb1cee",

			&channel.Channel{
				ID:         "0191d4f8-d139-11ed-8f50-833af1cb1cee",
				AsteriskID: "3e:50:6b:43:bb:30",
				TMDelete:   dbhandler.DefaultTimeStamp,
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
			mockReq.EXPECT().AstChannelSilenceOff(ctx, tt.responseChannel.AsteriskID, tt.responseChannel.ID).Return(nil)

			if err := h.SilenceOff(ctx, tt.id); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
