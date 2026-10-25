package callhandler

import (
	"context"
	"fmt"
	"testing"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/bridge"
	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-call-manager/models/playback"
	"monorepo/bin-call-manager/pkg/dbhandler"

	"github.com/gofrs/uuid"
)

func Test_ARIStasisStart_covB(t *testing.T) {
	h, _ := newCovBHandler(t)
	// a context that Start does not know leads to an error, which must be passed through.
	cn := &channel.Channel{ID: "ch-1", StasisData: map[channel.StasisDataType]string{channel.StasisDataTypeContext: "covb-unknown"}}
	if err := h.ARIStasisStart(context.Background(), cn); err == nil {
		t.Errorf("expected error")
	}
}

func Test_ARIChannelDestroyed_covB(t *testing.T) {
	tests := []struct {
		name      string
		chType    channel.Type
		dbErr     error
		expectErr bool
	}{
		{"call type, hangup error", channel.TypeCall, fmt.Errorf("err"), true},
		{"confbridge type", channel.TypeConfbridge, nil, false},
		{"unsupported type", channel.Type("covb-unknown"), nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCovBHandler(t)
			ctx := context.Background()
			cn := &channel.Channel{ID: "ch-1", Type: tt.chType}
			if tt.chType == channel.TypeCall {
				m.db.EXPECT().CallGetByChannelID(ctx, cn.ID).Return(nil, tt.dbErr)
			}
			err := h.ARIChannelDestroyed(ctx, cn)
			if (err != nil) != tt.expectErr {
				t.Errorf("unexpected err: %v", err)
			}
		})
	}
}

func Test_ARIChannelDtmfReceived_covB(t *testing.T) {
	t.Run("non pjsip is ignored", func(t *testing.T) {
		h, _ := newCovBHandler(t)
		cn := &channel.Channel{ID: "ch-1", Tech: channel.TechSIP}
		if err := h.ARIChannelDtmfReceived(context.Background(), cn, "1", 100); err != nil {
			t.Errorf("unexpected err: %v", err)
		}
	})

	t.Run("pjsip, not a call channel is ignored", func(t *testing.T) {
		h, m := newCovBHandler(t)
		ctx := context.Background()
		cn := &channel.Channel{ID: "ch-1", Tech: channel.TechPJSIP}
		m.db.EXPECT().CallGetByChannelID(ctx, cn.ID).Return(nil, fmt.Errorf("err"))
		if err := h.ARIChannelDtmfReceived(ctx, cn, "1", 100); err != nil {
			t.Errorf("unexpected err: %v", err)
		}
	})
}

func Test_ARIPlaybackFinished_covB(t *testing.T) {
	cn := &channel.Channel{ID: "ch-1"}

	t.Run("get call error", func(t *testing.T) {
		h, m := newCovBHandler(t)
		ctx := context.Background()
		m.db.EXPECT().CallGetByChannelID(ctx, cn.ID).Return(nil, fmt.Errorf("err"))
		e := &ari.PlaybackFinished{}
		if err := h.ARIPlaybackFinished(ctx, cn, e); err == nil {
			t.Errorf("expected error")
		}
	})

	t.Run("action id mismatch", func(t *testing.T) {
		h, m := newCovBHandler(t)
		ctx := context.Background()
		c := covBCall(covBCallID, call.StatusProgressing)
		c.Action.ID = uuid.FromStringOrNil("77777777-0000-0000-0000-000000000001")
		m.db.EXPECT().CallGetByChannelID(ctx, cn.ID).Return(c, nil)
		e := &ari.PlaybackFinished{}
		e.Playback.ID = playback.IDPrefixCall + "77777777-0000-0000-0000-000000000002"
		if err := h.ARIPlaybackFinished(ctx, cn, e); err != nil {
			t.Errorf("unexpected err: %v", err)
		}
	})
}

func Test_ARIChannelStateChange_covB(t *testing.T) {
	t.Run("recovery channel not up", func(t *testing.T) {
		h, _ := newCovBHandler(t)
		cn := &channel.Channel{
			ID:         "ch-1",
			State:      ari.ChannelStateRing,
			StasisData: map[channel.StasisDataType]string{channel.StasisDataTypeContext: string(channel.ContextCallRecovery)},
		}
		if err := h.ARIChannelStateChange(context.Background(), cn); err != nil {
			t.Errorf("unexpected err: %v", err)
		}
	})

	t.Run("status is not ringing or progressing", func(t *testing.T) {
		h, _ := newCovBHandler(t)
		cn := &channel.Channel{ID: "ch-1", State: ari.ChannelStateDown}
		if err := h.ARIChannelStateChange(context.Background(), cn); err != nil {
			t.Errorf("unexpected err: %v", err)
		}
	})

	t.Run("call not found is ignored", func(t *testing.T) {
		h, m := newCovBHandler(t)
		ctx := context.Background()
		cn := &channel.Channel{ID: "ch-1", State: ari.ChannelStateUp}
		m.db.EXPECT().CallGetByChannelID(ctx, cn.ID).Return(nil, dbhandler.ErrNotFound)
		if err := h.ARIChannelStateChange(ctx, cn); err != nil {
			t.Errorf("unexpected err: %v", err)
		}
	})

	t.Run("get call error", func(t *testing.T) {
		h, m := newCovBHandler(t)
		ctx := context.Background()
		cn := &channel.Channel{ID: "ch-1", State: ari.ChannelStateUp}
		m.db.EXPECT().CallGetByChannelID(ctx, cn.ID).Return(nil, fmt.Errorf("err"))
		if err := h.ARIChannelStateChange(ctx, cn); err == nil {
			t.Errorf("expected error")
		}
	})
}

func Test_answerCallBridgePeers_covB(t *testing.T) {
	cn := &channel.Channel{ID: "ch-1", BridgeID: "br-1"}

	t.Run("bridge get error", func(t *testing.T) {
		h, m := newCovBHandler(t)
		ctx := context.Background()
		m.bridge.EXPECT().Get(ctx, cn.BridgeID).Return(nil, fmt.Errorf("err"))
		h.answerCallBridgePeers(ctx, cn)
	})

	t.Run("non call bridge", func(t *testing.T) {
		h, m := newCovBHandler(t)
		ctx := context.Background()
		m.bridge.EXPECT().Get(ctx, cn.BridgeID).Return(&bridge.Bridge{ReferenceType: bridge.ReferenceTypeConfbridge}, nil)
		h.answerCallBridgePeers(ctx, cn)
	})

	t.Run("peers: get error, already up, answer error", func(t *testing.T) {
		h, m := newCovBHandler(t)
		ctx := context.Background()
		br := &bridge.Bridge{
			ReferenceType: bridge.ReferenceTypeCall,
			ChannelIDs:    []string{"ch-1", "peer-err", "peer-up", "peer-down"},
		}
		m.bridge.EXPECT().Get(ctx, cn.BridgeID).Return(br, nil)
		m.channel.EXPECT().Get(ctx, "peer-err").Return(nil, fmt.Errorf("err"))
		m.channel.EXPECT().Get(ctx, "peer-up").Return(&channel.Channel{ID: "peer-up", State: ari.ChannelStateUp}, nil)
		m.channel.EXPECT().Get(ctx, "peer-down").Return(&channel.Channel{ID: "peer-down", State: ari.ChannelStateRing}, nil)
		m.channel.EXPECT().Answer(ctx, "peer-down").Return(fmt.Errorf("err"))
		h.answerCallBridgePeers(ctx, cn)
	})
}

func Test_ARIChannelLeftBridge_covB(t *testing.T) {
	t.Run("call channel on call bridge", func(t *testing.T) {
		h, _ := newCovBHandler(t)
		cn := &channel.Channel{ID: "ch-1", Type: channel.TypeCall}
		br := &bridge.Bridge{ReferenceType: bridge.ReferenceTypeConfbridge}
		if err := h.ARIChannelLeftBridge(context.Background(), cn, br); err != nil {
			t.Errorf("unexpected err: %v", err)
		}
	})

	t.Run("unsupported channel type", func(t *testing.T) {
		h, _ := newCovBHandler(t)
		cn := &channel.Channel{ID: "ch-1", Type: channel.Type("covb-unknown")}
		if err := h.ARIChannelLeftBridge(context.Background(), cn, &bridge.Bridge{}); err != nil {
			t.Errorf("unexpected err: %v", err)
		}
	})
}
