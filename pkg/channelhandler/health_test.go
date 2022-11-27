package channelhandler

import (
	"context"
	"testing"

	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/util"
)

func Test_HealthCheck(t *testing.T) {

	type test struct {
		name string

		id            string
		retryCount    int
		retryCountMax int
		delay         int

		responseChannel *channel.Channel
	}

	tests := []test{
		{
			"normal",

			"27959a54-6e79-11ed-aef7-df3a51cc2639",
			3,
			5,
			10000,

			&channel.Channel{
				ID:         "27959a54-6e79-11ed-aef7-df3a51cc2639",
				AsteriskID: "42:01:0a:a4:00:03",
				TMEnd:      dbhandler.DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := util.NewMockUtil(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := channelHandler{
				util:          mockUtil,
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().ChannelGet(ctx, tt.id).Return(tt.responseChannel, nil)
			mockReq.EXPECT().AstChannelGet(ctx, tt.responseChannel.AsteriskID, tt.responseChannel.ID).Return(tt.responseChannel, nil)
			mockReq.EXPECT().CallV1ChannelHealth(ctx, tt.id, tt.delay, 0, tt.retryCountMax).Return(nil)
			h.HealthCheck(ctx, tt.id, tt.retryCount, tt.retryCountMax, tt.delay)
		})
	}
}
