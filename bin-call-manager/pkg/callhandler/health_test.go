package callhandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/channelhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"
)

func Test_HealthCheck(t *testing.T) {

	tests := []struct {
		name string

		id         uuid.UUID
		retryCount int

		responseCall    *call.Call
		responseChannel *channel.Channel

		expectRetryCount int
	}{
		{
			name: "normal",

			id:         uuid.FromStringOrNil("8b0c11f9-ad03-4bd7-98a5-102f89877e2a"),
			retryCount: 0,

			responseCall: &call.Call{
				ID:        uuid.FromStringOrNil("8b0c11f9-ad03-4bd7-98a5-102f89877e2a"),
				ChannelID: "a5416ed3-5f61-4e1e-971f-0b3ff61ce19e",
				Status:    call.StatusProgressing,
				TMHangup:  dbhandler.DefaultTimeStamp,
				TMDelete:  dbhandler.DefaultTimeStamp,
			},
			responseChannel: &channel.Channel{
				ID:       "a5416ed3-5f61-4e1e-971f-0b3ff61ce19e",
				TMEnd:    dbhandler.DefaultTimeStamp,
				TMDelete: dbhandler.DefaultTimeStamp,
			},

			expectRetryCount: 0,
		},
		{
			name: "calll channel ended",

			id:         uuid.FromStringOrNil("d0760324-75d6-443d-aa6f-d3b8703bf78a"),
			retryCount: 0,

			responseCall: &call.Call{
				ID:        uuid.FromStringOrNil("d0760324-75d6-443d-aa6f-d3b8703bf78a"),
				ChannelID: "cba7edf9-8586-40c0-992b-5885103228c1",
				Status:    call.StatusProgressing,
				TMHangup:  dbhandler.DefaultTimeStamp,
				TMDelete:  dbhandler.DefaultTimeStamp,
			},
			responseChannel: &channel.Channel{
				ID:       "cba7edf9-8586-40c0-992b-5885103228c1",
				TMEnd:    "2023-01-18 03:22:18.995000",
				TMDelete: dbhandler.DefaultTimeStamp,
			},

			expectRetryCount: 1,
		},
		{
			name: "calll channel deleted",

			id:         uuid.FromStringOrNil("d0760324-75d6-443d-aa6f-d3b8703bf78a"),
			retryCount: 0,

			responseCall: &call.Call{
				ID:        uuid.FromStringOrNil("d0760324-75d6-443d-aa6f-d3b8703bf78a"),
				ChannelID: "cba7edf9-8586-40c0-992b-5885103228c1",
				Status:    call.StatusProgressing,
				TMHangup:  dbhandler.DefaultTimeStamp,
				TMDelete:  dbhandler.DefaultTimeStamp,
			},
			responseChannel: &channel.Channel{
				ID:       "cba7edf9-8586-40c0-992b-5885103228c1",
				TMEnd:    dbhandler.DefaultTimeStamp,
				TMDelete: "2023-01-18 03:22:18.995000",
			},

			expectRetryCount: 1,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotfiy := notifyhandler.NewMockNotifyHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)

			h := &callHandler{
				utilHandler:    mockUtil,
				reqHandler:     mockReq,
				db:             mockDB,
				notifyHandler:  mockNotfiy,
				channelHandler: mockChannel,
			}
			ctx := context.Background()

			mockDB.EXPECT().CallGet(ctx, tt.id).Return(tt.responseCall, nil)
			mockChannel.EXPECT().Get(ctx, tt.responseCall.ChannelID).Return(tt.responseChannel, nil)

			mockReq.EXPECT().CallV1CallHealth(ctx, tt.id, defaultHealthDelay, tt.expectRetryCount).Return(nil)

			h.HealthCheck(ctx, tt.id, tt.retryCount)
		})
	}
}
