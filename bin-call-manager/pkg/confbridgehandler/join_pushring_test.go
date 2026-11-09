package confbridgehandler

import (
	"context"
	stderrors "errors"
	"testing"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/bridge"
	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-call-manager/models/common"
	"monorepo/bin-call-manager/models/confbridge"
	"monorepo/bin-call-manager/pkg/pushringhandler"
)

// the confbridge of a push ring session has the flag, so only its joins reach the session handler.
func Test_Join_pushRing(t *testing.T) {
	ctx := context.Background()

	cbPush := &confbridge.Confbridge{BridgeID: "c2-bridge-1", Type: confbridge.TypeConference, Flags: []confbridge.Flag{confbridge.FlagPushRing}}
	cbPush.ID = cov2CBID
	cbPlain := &confbridge.Confbridge{BridgeID: "c2-bridge-1", Type: confbridge.TypeConference}
	cbPlain.ID = cov2CBID
	c := &call.Call{ChannelID: "c2-call-channel", BridgeID: "c2-call-bridge"}
	c.ID = cov2CallID

	expectJoinChannel := func(m *cov2Mocks, result error) {
		m.bridge.EXPECT().IsExist(ctx, "c2-bridge-1").Return(true)
		m.bridge.EXPECT().Get(ctx, "c2-bridge-1").Return(&bridge.Bridge{ID: "c2-bridge-1", AsteriskID: "c2-ast"}, nil)
		m.cache.EXPECT().AsteriskAddressInternalGet(ctx, "c2-ast").Return("192.0.2.1", nil)
		m.util.EXPECT().UUIDCreate().Return(cov2ExtID)
		var res *channel.Channel
		if result == nil {
			res = &channel.Channel{ID: cov2ExtID.String()}
		}
		m.channel.EXPECT().StartChannelWithBaseChannel(
			ctx, "c2-call-channel", cov2ExtID.String(),
			"context_type=call,context=call-join,confbridge_id="+cov2CBID.String()+",bridge_id=c2-call-bridge,call_id="+cov2CallID.String(),
			"PJSIP/conf-join/sip:c2-bridge-1@192.0.2.1:5060", "", "vp8", "", map[string]string{
				"PJSIP_HEADER(add," + common.SIPHeaderCallID + ")":       cov2CallID.String(),
				"PJSIP_HEADER(add," + common.SIPHeaderConfbridgeID + ")": cov2CBID.String(),
			},
		).Return(res, result)
	}

	tests := []struct {
		name  string
		setup func(m *cov2Mocks)

		expectErr error
	}{
		{
			name: "an ordinary confbridge never reads the session",
			setup: func(m *cov2Mocks) {
				m.db.EXPECT().ConfbridgeGet(ctx, cov2CBID).Return(cbPlain, nil)
				m.req.EXPECT().CallV1CallGet(ctx, cov2CallID).Return(c, nil)
				expectJoinChannel(m, nil)
			},
		},
		{
			name: "an ordinary confbridge: a failing join does not release anything",
			setup: func(m *cov2Mocks) {
				m.db.EXPECT().ConfbridgeGet(ctx, cov2CBID).Return(cbPlain, nil)
				m.req.EXPECT().CallV1CallGet(ctx, cov2CallID).Return(c, nil)
				expectJoinChannel(m, errCov2)
			},
			expectErr: errCov2,
		},
		{
			name: "a session confbridge: the call passes the gate and joins",
			setup: func(m *cov2Mocks) {
				m.db.EXPECT().ConfbridgeGet(ctx, cov2CBID).Return(cbPush, nil)
				m.req.EXPECT().CallV1CallGet(ctx, cov2CallID).Return(c, nil)
				m.push.EXPECT().JoinGate(ctx, cbPush, cov2CallID).Return(nil)
				expectJoinChannel(m, nil)
			},
		},
		{
			name: "a session confbridge: a refused call does not join",
			setup: func(m *cov2Mocks) {
				m.db.EXPECT().ConfbridgeGet(ctx, cov2CBID).Return(cbPush, nil)
				m.req.EXPECT().CallV1CallGet(ctx, cov2CallID).Return(c, nil)
				m.push.EXPECT().JoinGate(ctx, cbPush, cov2CallID).Return(pushringhandler.ErrRefused)
			},
			expectErr: pushringhandler.ErrRefused,
		},
		{
			name: "a session confbridge: a failing join releases the lease",
			setup: func(m *cov2Mocks) {
				m.db.EXPECT().ConfbridgeGet(ctx, cov2CBID).Return(cbPush, nil)
				m.req.EXPECT().CallV1CallGet(ctx, cov2CallID).Return(c, nil)
				m.push.EXPECT().JoinGate(ctx, cbPush, cov2CallID).Return(nil)
				expectJoinChannel(m, errCov2)
				m.push.EXPECT().Release(ctx, cov2CallID)
			},
			expectErr: errCov2,
		},
		{
			name: "a session confbridge: a failure at the bridge lookup releases the lease",
			setup: func(m *cov2Mocks) {
				m.db.EXPECT().ConfbridgeGet(ctx, cov2CBID).Return(cbPush, nil)
				m.req.EXPECT().CallV1CallGet(ctx, cov2CallID).Return(c, nil)
				m.push.EXPECT().JoinGate(ctx, cbPush, cov2CallID).Return(nil)
				m.bridge.EXPECT().IsExist(ctx, "c2-bridge-1").Return(true)
				m.bridge.EXPECT().Get(ctx, "c2-bridge-1").Return(nil, errCov2)
				m.push.EXPECT().Release(ctx, cov2CallID)
			},
			expectErr: errCov2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCov2Handler(t)
			tt.setup(m)

			err := h.Join(ctx, cov2CBID, cov2CallID)
			if tt.expectErr == nil && err != nil {
				t.Errorf("Wrong match. expect: nil, got: %v", err)
			}
			if tt.expectErr != nil && !stderrors.Is(err, tt.expectErr) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectErr, err)
			}
		})
	}
}

func Test_Joined_pushRing(t *testing.T) {
	ctx := context.Background()

	cn := &channel.Channel{
		ID: "c2-channel-1",
		StasisData: map[channel.StasisDataType]string{
			channel.StasisDataTypeConfbridgeID: cov2CBID.String(),
			channel.StasisDataTypeCallID:       cov2CallID.String(),
		},
	}
	br := &bridge.Bridge{ID: "c2-bridge-1"}

	cbPush := cov2CB(func(c *confbridge.Confbridge) {
		c.Type = confbridge.TypeConference
		c.Flags = []confbridge.Flag{confbridge.FlagPushRing}
	})

	// expectUpToUpdate covers the calls of Joined before the session handler
	expectUpToUpdate := func(m *cov2Mocks, cb *confbridge.Confbridge) {
		m.db.EXPECT().ConfbridgeAddChannelCallID(ctx, cov2CBID, cn.ID, cov2CallID).Return(nil)
		m.db.EXPECT().ConfbridgeGet(ctx, cov2CBID).Return(cb, nil)
		m.notify.EXPECT().PublishEvent(ctx, confbridge.EventTypeConfbridgeJoined, &confbridge.EventConfbridgeJoined{Confbridge: *cb, JoinedCallID: cov2CallID})
		m.req.EXPECT().CallV1CallUpdateConfbridgeID(ctx, cov2CallID, cov2CBID).Return(&call.Call{}, nil)
	}

	t.Run("a session confbridge: the join is accepted and the channel is answered", func(t *testing.T) {
		h, m := newCov2Handler(t)
		expectUpToUpdate(m, cbPush)
		m.push.EXPECT().Joined(ctx, cbPush, cov2CallID).Return(nil)
		m.channel.EXPECT().Answer(ctx, cn.ID).Return(nil)

		if err := h.Joined(ctx, cn, br); err != nil {
			t.Errorf("Wrong match. expect: nil, got: %v", err)
		}
	})

	t.Run("a session confbridge: a refused join ends the channel and releases the lease before anything is answered", func(t *testing.T) {
		h, m := newCov2Handler(t)
		expectUpToUpdate(m, cbPush)
		m.push.EXPECT().Joined(ctx, cbPush, cov2CallID).Return(pushringhandler.ErrRefused)
		m.channel.EXPECT().HangingUp(ctx, cn.ID, ari.ChannelCauseUnallocated).Return(&channel.Channel{}, nil)
		m.push.EXPECT().Release(ctx, cov2CallID)

		err := h.Joined(ctx, cn, br)
		if !stderrors.Is(err, pushringhandler.ErrRefused) {
			t.Errorf("Wrong match. expect: ErrRefused, got: %v", err)
		}
	})

	t.Run("a session confbridge: the failure to add the channel releases the lease after the hangup", func(t *testing.T) {
		h, m := newCov2Handler(t)
		m.db.EXPECT().ConfbridgeAddChannelCallID(ctx, cov2CBID, cn.ID, cov2CallID).Return(errCov2)
		m.channel.EXPECT().HangingUp(ctx, cn.ID, ari.ChannelCauseUnallocated).Return(&channel.Channel{}, nil)
		m.db.EXPECT().ConfbridgeGet(ctx, cov2CBID).Return(cbPush, nil)
		m.push.EXPECT().Release(ctx, cov2CallID)

		if err := h.Joined(ctx, cn, br); !stderrors.Is(err, errCov2) {
			t.Errorf("Wrong match. expect: errCov2, got: %v", err)
		}
	})

	t.Run("a session confbridge: the failure to update the call hangs up the channel and releases the lease", func(t *testing.T) {
		h, m := newCov2Handler(t)
		m.db.EXPECT().ConfbridgeAddChannelCallID(ctx, cov2CBID, cn.ID, cov2CallID).Return(nil)
		m.db.EXPECT().ConfbridgeGet(ctx, cov2CBID).Return(cbPush, nil).Times(2)
		m.notify.EXPECT().PublishEvent(ctx, confbridge.EventTypeConfbridgeJoined, &confbridge.EventConfbridgeJoined{Confbridge: *cbPush, JoinedCallID: cov2CallID})
		m.req.EXPECT().CallV1CallUpdateConfbridgeID(ctx, cov2CallID, cov2CBID).Return(nil, errCov2)
		m.channel.EXPECT().HangingUp(ctx, cn.ID, ari.ChannelCauseUnallocated).Return(&channel.Channel{}, nil)
		m.push.EXPECT().Release(ctx, cov2CallID)

		if err := h.Joined(ctx, cn, br); !stderrors.Is(err, errCov2) {
			t.Errorf("Wrong match. expect: errCov2, got: %v", err)
		}
	})

	t.Run("a session confbridge: an answer failure after the accept hangs up the channel and releases the lease", func(t *testing.T) {
		h, m := newCov2Handler(t)
		expectUpToUpdate(m, cbPush)
		m.push.EXPECT().Joined(ctx, cbPush, cov2CallID).Return(nil)
		m.channel.EXPECT().Answer(ctx, cn.ID).Return(errCov2)
		m.channel.EXPECT().HangingUp(ctx, cn.ID, ari.ChannelCauseUnallocated).Return(&channel.Channel{}, nil)
		m.push.EXPECT().Release(ctx, cov2CallID)

		if err := h.Joined(ctx, cn, br); !stderrors.Is(err, errCov2) {
			t.Errorf("Wrong match. expect: errCov2, got: %v", err)
		}
	})

	t.Run("an ordinary confbridge: an answer failure is only returned", func(t *testing.T) {
		h, m := newCov2Handler(t)
		cbPlain := cov2CB(func(c *confbridge.Confbridge) { c.Type = confbridge.TypeConference })
		expectUpToUpdate(m, cbPlain)
		m.channel.EXPECT().Answer(ctx, cn.ID).Return(errCov2)

		if err := h.Joined(ctx, cn, br); !stderrors.Is(err, errCov2) {
			t.Errorf("Wrong match. expect: errCov2, got: %v", err)
		}
	})

	t.Run("an ordinary confbridge never reads the session", func(t *testing.T) {
		h, m := newCov2Handler(t)
		cbPlain := cov2CB(func(c *confbridge.Confbridge) { c.Type = confbridge.TypeConference })
		expectUpToUpdate(m, cbPlain)
		m.channel.EXPECT().Answer(ctx, cn.ID).Return(nil)

		if err := h.Joined(ctx, cn, br); err != nil {
			t.Errorf("Wrong match. expect: nil, got: %v", err)
		}
	})
}
