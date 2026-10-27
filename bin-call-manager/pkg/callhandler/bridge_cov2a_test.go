package callhandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/bridge"
	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/channel"
)

func Test_bridgeLeftJoin_cov2a(t *testing.T) {
	tests := []struct {
		name  string
		setup func(m *cov2aMocks, c *call.Call, cn *channel.Channel)

		wantErr string
	}{
		{
			name: "confbridge reset error",
			setup: func(m *cov2aMocks, c *call.Call, cn *channel.Channel) {
				m.channel.EXPECT().HangingUp(gomock.Any(), cn.ID, ari.ChannelCauseNormalClearing).Return(nil, nil)
				m.db.EXPECT().CallSetConfbridgeID(gomock.Any(), c.ID, uuid.Nil).Return(errCov2aBoom)
			},
			wantErr: "boom",
		},
		{
			name: "call lookup error",
			setup: func(m *cov2aMocks, c *call.Call, cn *channel.Channel) {
				m.channel.EXPECT().HangingUp(gomock.Any(), cn.ID, ari.ChannelCauseNormalClearing).Return(nil, nil)
				m.db.EXPECT().CallSetConfbridgeID(gomock.Any(), c.ID, uuid.Nil).Return(nil)
				m.db.EXPECT().CallGet(gomock.Any(), c.ID).Return(nil, errCov2aBoom)
			},
			wantErr: "boom",
		},
		{
			name: "terminating call does not request the next action",
			setup: func(m *cov2aMocks, c *call.Call, cn *channel.Channel) {
				c.Status = call.StatusTerminating
				m.channel.EXPECT().HangingUp(gomock.Any(), cn.ID, ari.ChannelCauseNormalClearing).Return(nil, nil)
				m.db.EXPECT().CallSetConfbridgeID(gomock.Any(), c.ID, uuid.Nil).Return(nil)
				m.db.EXPECT().CallGet(gomock.Any(), c.ID).Return(c, nil)
				m.notify.EXPECT().PublishWebhookEvent(gomock.Any(), c.CustomerID, call.EventTypeCallUpdated, c)
			},
		},
		{
			name: "next action request error",
			setup: func(m *cov2aMocks, c *call.Call, cn *channel.Channel) {
				m.channel.EXPECT().HangingUp(gomock.Any(), cn.ID, ari.ChannelCauseNormalClearing).Return(nil, nil)
				m.db.EXPECT().CallSetConfbridgeID(gomock.Any(), c.ID, uuid.Nil).Return(nil)
				m.db.EXPECT().CallGet(gomock.Any(), c.ID).Return(c, nil)
				m.notify.EXPECT().PublishWebhookEvent(gomock.Any(), c.CustomerID, call.EventTypeCallUpdated, c)
				m.req.EXPECT().CallV1CallActionNext(gomock.Any(), c.ID, false).Return(errCov2aBoom)
			},
			wantErr: "boom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := cov2aNewHandler(t)

			c := cov2aNewCall(nil)
			cn := &channel.Channel{ID: "cov2a-join-ch"}
			br := &bridge.Bridge{ID: "cov2a-br-1", ReferenceID: c.ID}
			tt.setup(m, c, cn)

			err := h.bridgeLeftJoin(context.Background(), cn, br)
			cov2aCheckErr(t, err, tt.wantErr)
		})
	}
}

func Test_removeAllChannelsInBridge_cov2a(t *testing.T) {
	h, m := cov2aNewHandler(t)
	ctx := context.Background()

	br := &bridge.Bridge{ID: "cov2a-br-1", ChannelIDs: []string{"cov2a-ch-a", "cov2a-ch-b"}}

	// a kick error is only logged and does not stop the kicking of the remaining channels.
	gomock.InOrder(
		m.bridge.EXPECT().ChannelKick(ctx, br.ID, "cov2a-ch-a").Return(errCov2aBoom),
		m.bridge.EXPECT().ChannelKick(ctx, br.ID, "cov2a-ch-b").Return(nil),
	)

	h.removeAllChannelsInBridge(ctx, br)
}
