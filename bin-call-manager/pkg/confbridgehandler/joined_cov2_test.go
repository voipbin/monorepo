package confbridgehandler

import (
	"context"
	stderrors "errors"
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/bridge"
	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-call-manager/models/confbridge"
)

func Test_Joined_cov2_errors(t *testing.T) {
	ctx := context.Background()

	cn := &channel.Channel{
		ID: "c2-channel-1",
		StasisData: map[channel.StasisDataType]string{
			channel.StasisDataTypeConfbridgeID: cov2CBID.String(),
			channel.StasisDataTypeCallID:       cov2CallID.String(),
		},
	}

	tests := []struct {
		name  string
		setup func(m *cov2Mocks)
	}{
		{
			name: "add channel call id failed hangs up the channel",
			setup: func(m *cov2Mocks) {
				m.db.EXPECT().ConfbridgeAddChannelCallID(ctx, cov2CBID, cn.ID, cov2CallID).Return(errCov2)
				m.channel.EXPECT().HangingUp(ctx, cn.ID, ari.ChannelCauseUnallocated).Return(nil, errCov2)
			},
		},
		{
			name: "update call confbridge id failed",
			setup: func(m *cov2Mocks) {
				cb := cov2CB(func(c *confbridge.Confbridge) { c.Type = confbridge.TypeConference })
				m.db.EXPECT().ConfbridgeAddChannelCallID(ctx, cov2CBID, cn.ID, cov2CallID).Return(nil)
				m.db.EXPECT().ConfbridgeGet(ctx, cov2CBID).Return(cb, nil)
				m.notify.EXPECT().PublishEvent(ctx, confbridge.EventTypeConfbridgeJoined, &confbridge.EventConfbridgeJoined{Confbridge: *cb, JoinedCallID: cov2CallID})
				m.req.EXPECT().CallV1CallUpdateConfbridgeID(ctx, cov2CallID, cov2CBID).Return(nil, errCov2)
			},
		},
		{
			name: "conference type answer failed",
			setup: func(m *cov2Mocks) {
				cb := cov2CB(func(c *confbridge.Confbridge) { c.Type = confbridge.TypeConference })
				m.db.EXPECT().ConfbridgeAddChannelCallID(ctx, cov2CBID, cn.ID, cov2CallID).Return(nil)
				m.db.EXPECT().ConfbridgeGet(ctx, cov2CBID).Return(cb, nil)
				m.notify.EXPECT().PublishEvent(ctx, confbridge.EventTypeConfbridgeJoined, &confbridge.EventConfbridgeJoined{Confbridge: *cb, JoinedCallID: cov2CallID})
				m.req.EXPECT().CallV1CallUpdateConfbridgeID(ctx, cov2CallID, cov2CBID).Return(&call.Call{}, nil)
				m.channel.EXPECT().Answer(ctx, cn.ID).Return(errCov2)
			},
		},
		{
			name: "connect type first channel ring failed",
			setup: func(m *cov2Mocks) {
				cb := cov2CB(func(c *confbridge.Confbridge) {
					c.Type = confbridge.TypeConnect
					c.ChannelCallIDs = map[string]uuid.UUID{cn.ID: cov2CallID}
				})
				m.db.EXPECT().ConfbridgeAddChannelCallID(ctx, cov2CBID, cn.ID, cov2CallID).Return(nil)
				m.db.EXPECT().ConfbridgeGet(ctx, cov2CBID).Return(cb, nil)
				m.notify.EXPECT().PublishEvent(ctx, confbridge.EventTypeConfbridgeJoined, &confbridge.EventConfbridgeJoined{Confbridge: *cb, JoinedCallID: cov2CallID})
				m.req.EXPECT().CallV1CallUpdateConfbridgeID(ctx, cov2CallID, cov2CBID).Return(&call.Call{}, nil)
				m.channel.EXPECT().Ring(ctx, cn.ID).Return(errCov2)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCov2Handler(t)
			tt.setup(m)

			err := h.Joined(ctx, cn, &bridge.Bridge{ID: "c2-bridge-1"})
			if !stderrors.Is(err, errCov2) {
				t.Errorf("Wrong match. expect: wrapped errCov2, got: %v", err)
			}
		})
	}
}

func Test_joinedTypeConnect_cov2(t *testing.T) {
	ctx := context.Background()

	cb := cov2CB(func(c *confbridge.Confbridge) {
		c.Type = confbridge.TypeConnect
		c.ChannelCallIDs = map[string]uuid.UUID{"c2-ch-1": cov2CallID, "c2-ch-2": cov2CallID2}
	})
	c := &call.Call{}
	c.ID = cov2CallID

	tests := []struct {
		name string

		setup     func(m *cov2Mocks)
		expectErr bool
	}{
		{
			name: "call get failed",
			setup: func(m *cov2Mocks) {
				m.req.EXPECT().CallV1CallGet(ctx, gomockAnyUUID()).Return(nil, errCov2)
			},
			expectErr: true,
		},
		{
			name: "outgoing call ringing then confbridge ring failed",
			setup: func(m *cov2Mocks) {
				ringing := &call.Call{Direction: call.DirectionOutgoing, Status: call.StatusRinging}
				m.req.EXPECT().CallV1CallGet(ctx, gomockAnyUUID()).Return(ringing, nil)
				// h.Ring reads the confbridge, which fails. the error is only logged.
				m.db.EXPECT().ConfbridgeGet(ctx, cov2CBID).Return(nil, errCov2)
			},
		},
		{
			name: "no ringing call then confbridge answer failed",
			setup: func(m *cov2Mocks) {
				answered := &call.Call{Direction: call.DirectionIncoming, Status: call.StatusProgressing}
				m.req.EXPECT().CallV1CallGet(ctx, gomockAnyUUID()).Return(answered, nil).Times(2)
				m.db.EXPECT().ConfbridgeGet(ctx, cov2CBID).Return(nil, errCov2)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCov2Handler(t)
			tt.setup(m)

			err := h.joinedTypeConnect(ctx, "c2-ch-1", c, cb)
			if (err != nil) != tt.expectErr {
				t.Errorf("Wrong match. expect error: %v, got: %v", tt.expectErr, err)
			}
			if tt.expectErr && !stderrors.Is(err, errCov2) {
				t.Errorf("Wrong match. expect: wrapped errCov2, got: %v", err)
			}
		})
	}
}
