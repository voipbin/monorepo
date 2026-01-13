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

func Test_HoldOn(t *testing.T) {

	type test struct {
		name string

		id string

		responseChannel *channel.Channel
	}

	tests := []test{
		{
			"normal",

			"e3119da2-cfcf-11ed-a0e8-afd845a8c6b4",

			&channel.Channel{
				ID:         "e3119da2-cfcf-11ed-a0e8-afd845a8c6b4",
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

			mockDB.EXPECT().ChannelGet(gomock.Any(), tt.id.Return(tt.responseChannel, nil)
			mockReq.EXPECT().AstChannelHoldOn(ctx, tt.responseChannel.AsteriskID, tt.responseChannel.ID.Return(nil)

			if err := h.HoldOn(ctx, tt.id); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_HoldOff(t *testing.T) {

	type test struct {
		name string

		id string

		responseChannel *channel.Channel
	}

	tests := []test{
		{
			"normal",

			"0b219856-cfd0-11ed-94b5-1f3372c94993",

			&channel.Channel{
				ID:         "0b219856-cfd0-11ed-94b5-1f3372c94993",
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

			mockDB.EXPECT().ChannelGet(gomock.Any(), tt.id.Return(tt.responseChannel, nil)
			mockReq.EXPECT().AstChannelHoldOff(ctx, tt.responseChannel.AsteriskID, tt.responseChannel.ID.Return(nil)

			if err := h.HoldOff(ctx, tt.id); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
