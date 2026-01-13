package channelhandler

import (
	"context"
	reflect "reflect"
	"testing"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-call-manager/pkg/dbhandler"
)

func Test_HangingUp(t *testing.T) {

	type test struct {
		name string

		id    string
		cause ari.ChannelCause

		responseChannel *channel.Channel
		expectRes       *channel.Channel
	}

	tests := []test{
		{
			"normal",

			"9510666c-9a2b-11ed-9cff-67a60fb2f265",
			ari.ChannelCauseNormalClearing,

			&channel.Channel{
				ID:       "9510666c-9a2b-11ed-9cff-67a60fb2f265",
				TMDelete: dbhandler.DefaultTimeStamp,
			},
			&channel.Channel{
				ID: "764818be-6e0d-11ed-a193-33bbe6697279",
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
			mockReq.EXPECT().AstChannelHangup(ctx, tt.responseChannel.AsteriskID, tt.responseChannel.ID, tt.cause, 0).Return(nil)

			res, err := h.HangingUp(ctx, tt.id, tt.cause)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.responseChannel, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseChannel, res)
			}
		})
	}
}
