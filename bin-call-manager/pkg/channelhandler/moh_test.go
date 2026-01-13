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

func Test_MOHOn(t *testing.T) {

	type test struct {
		name string

		id string

		responseChannel *channel.Channel
	}

	tests := []test{
		{
			"normal",

			"9c399960-d138-11ed-922d-2f430e6193fe",

			&channel.Channel{
				ID:         "9c399960-d138-11ed-922d-2f430e6193fe",
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
			mockReq.EXPECT().AstChannelMusicOnHoldOn(ctx, tt.responseChannel.AsteriskID, tt.responseChannel.ID).Return(nil)

			if err := h.MOHOn(ctx, tt.id); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_MOHOff(t *testing.T) {

	type test struct {
		name string

		id string

		responseChannel *channel.Channel
	}

	tests := []test{
		{
			"normal",

			"9c6b532e-d138-11ed-95e8-7bca000c02ad",

			&channel.Channel{
				ID:         "9c6b532e-d138-11ed-95e8-7bca000c02ad",
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
			mockReq.EXPECT().AstChannelMusicOnHoldOff(ctx, tt.responseChannel.AsteriskID, tt.responseChannel.ID).Return(nil)

			if err := h.MOHOff(ctx, tt.id); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
