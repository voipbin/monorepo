package confbridgehandler

import (
	"context"
	"testing"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/bridge"
	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-call-manager/models/confbridge"
)

func Test_ARIChannelLeftBridge_cov2_nonConfbridge(t *testing.T) {
	h, _ := newCov2Handler(t)

	// no mock call is expected for non-confbridge channels
	if err := h.ARIChannelLeftBridge(context.Background(), &channel.Channel{Type: channel.TypeCall}, &bridge.Bridge{}); err != nil {
		t.Errorf("Wrong match. expect: nil, got: %v", err)
	}
}

func Test_ARIChannelEnteredBridge_cov2(t *testing.T) {
	ctx := context.Background()

	cn := &channel.Channel{
		ID:   "c2-channel-1",
		Type: channel.TypeConfbridge,
		StasisData: map[channel.StasisDataType]string{
			channel.StasisDataTypeConfbridgeID: cov2CBID.String(),
			channel.StasisDataTypeCallID:       cov2CallID.String(),
		},
	}

	tests := []struct {
		name string
		cn   *channel.Channel

		addErr    error
		expectErr bool
		expectHup bool
	}{
		{"non confbridge channel", &channel.Channel{Type: channel.TypeCall}, nil, false, false},
		{"confbridge channel but add channel failed", cn, errCov2, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCov2Handler(t)

			if tt.expectHup {
				m.db.EXPECT().ConfbridgeAddChannelCallID(ctx, cov2CBID, tt.cn.ID, cov2CallID).Return(tt.addErr)
				m.channel.EXPECT().HangingUp(ctx, tt.cn.ID, ari.ChannelCauseUnallocated).Return(&channel.Channel{}, nil)
			}

			err := h.ARIChannelEnteredBridge(ctx, tt.cn, &bridge.Bridge{ID: "c2-bridge-1"})
			if (err != nil) != tt.expectErr {
				t.Errorf("Wrong match. expect error: %v, got: %v", tt.expectErr, err)
			}
		})
	}
}

func Test_ARIBridgeDestroyed_cov2(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name string
		br   *bridge.Bridge

		responseConfbridge *confbridge.Confbridge
		expectGet          bool
	}{
		{
			name: "reference type is not confbridge",
			br:   &bridge.Bridge{ReferenceType: bridge.ReferenceTypeCall, ReferenceID: cov2CBID},
		},
		{
			name: "confbridge reference but not terminating",
			br:   &bridge.Bridge{ReferenceType: bridge.ReferenceTypeConfbridge, ReferenceID: cov2CBID},

			responseConfbridge: &confbridge.Confbridge{Status: confbridge.StatusProgressing},
			expectGet:          true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCov2Handler(t)

			if tt.expectGet {
				m.db.EXPECT().ConfbridgeGet(ctx, tt.br.ReferenceID).Return(tt.responseConfbridge, nil)
			}

			if err := h.ARIBridgeDestroyed(ctx, tt.br); err != nil {
				t.Errorf("Wrong match. expect: nil, got: %v", err)
			}
		})
	}
}

func Test_ARIChannelStateChange_cov2(t *testing.T) {
	ctx := context.Background()

	joinCN := &channel.Channel{
		ID:    "c2-join-1",
		Type:  channel.TypeJoin,
		State: ari.ChannelStateUp,
		StasisData: map[channel.StasisDataType]string{
			channel.StasisDataTypeBridgeID: "c2-bridge-1",
		},
	}

	tests := []struct {
		name string
		cn   *channel.Channel

		setup func(m *cov2Mocks)
	}{
		{
			name: "not a join channel",
			cn:   &channel.Channel{Type: channel.TypeCall, State: ari.ChannelStateUp},
		},
		{
			name: "join channel is not up",
			cn:   &channel.Channel{Type: channel.TypeJoin, State: ari.ChannelStateRing},
		},
		{
			name: "no bridge id anywhere",
			cn:   &channel.Channel{Type: channel.TypeJoin, State: ari.ChannelStateUp, StasisData: map[channel.StasisDataType]string{}},
		},
		{
			name: "bridge get failed",
			cn:   joinCN,
			setup: func(m *cov2Mocks) {
				m.bridge.EXPECT().Get(ctx, "c2-bridge-1").Return(nil, errCov2)
			},
		},
		{
			name: "bridge is not a call bridge",
			cn:   joinCN,
			setup: func(m *cov2Mocks) {
				m.bridge.EXPECT().Get(ctx, "c2-bridge-1").Return(&bridge.Bridge{ReferenceType: bridge.ReferenceTypeConfbridge, ChannelIDs: []string{"c2-peer-1"}}, nil)
			},
		},
		{
			name: "peer get failed, peer already up, and answer failure are tolerated",
			cn:   joinCN,
			setup: func(m *cov2Mocks) {
				m.bridge.EXPECT().Get(ctx, "c2-bridge-1").Return(&bridge.Bridge{
					ReferenceType: bridge.ReferenceTypeCall,
					ChannelIDs:    []string{"c2-join-1", "c2-peer-1", "c2-peer-2", "c2-peer-3"},
				}, nil)
				m.channel.EXPECT().Get(ctx, "c2-peer-1").Return(nil, errCov2)
				m.channel.EXPECT().Get(ctx, "c2-peer-2").Return(&channel.Channel{ID: "c2-peer-2", State: ari.ChannelStateUp}, nil)
				m.channel.EXPECT().Get(ctx, "c2-peer-3").Return(&channel.Channel{ID: "c2-peer-3", State: ari.ChannelStateRing}, nil)
				m.channel.EXPECT().Answer(ctx, "c2-peer-3").Return(errCov2)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCov2Handler(t)
			if tt.setup != nil {
				tt.setup(m)
			}

			h.ARIChannelStateChange(ctx, tt.cn)
		})
	}
}

func Test_ARIChannelEnteredBridge_cov2_joinedAddsCall(t *testing.T) {
	// Joined with unknown confbridge type returns nil after registering the channel/call pair.
	ctx := context.Background()
	h, m := newCov2Handler(t)

	cn := &channel.Channel{
		ID:   "c2-channel-2",
		Type: channel.TypeConfbridge,
		StasisData: map[channel.StasisDataType]string{
			channel.StasisDataTypeConfbridgeID: cov2CBID.String(),
			channel.StasisDataTypeCallID:       cov2CallID.String(),
		},
	}
	cb := &confbridge.Confbridge{Type: "unknown"}

	m.db.EXPECT().ConfbridgeAddChannelCallID(ctx, cov2CBID, cn.ID, cov2CallID).Return(nil)
	m.db.EXPECT().ConfbridgeGet(ctx, cov2CBID).Return(cb, nil)
	m.notify.EXPECT().PublishEvent(ctx, confbridge.EventTypeConfbridgeJoined, &confbridge.EventConfbridgeJoined{Confbridge: *cb, JoinedCallID: cov2CallID})
	m.req.EXPECT().CallV1CallUpdateConfbridgeID(ctx, cov2CallID, cov2CBID).Return(nil, nil)

	if err := h.ARIChannelEnteredBridge(ctx, cn, &bridge.Bridge{}); err != nil {
		t.Errorf("Wrong match. expect: nil, got: %v", err)
	}
}
