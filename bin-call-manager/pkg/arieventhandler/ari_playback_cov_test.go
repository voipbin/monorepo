package arieventhandler

import (
	"context"
	"errors"
	"testing"
	"time"

	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/bridge"
	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-call-manager/pkg/bridgehandler"
	"monorepo/bin-call-manager/pkg/callhandler"
	"monorepo/bin-call-manager/pkg/channelhandler"
	"monorepo/bin-call-manager/pkg/externalmediahandler"
)

func Test_EventHandlerPlaybackFinished_cov(t *testing.T) {
	now := time.Now()

	newEvent := func(target string) *ari.PlaybackFinished {
		return &ari.PlaybackFinished{Playback: ari.Playback{ID: "pb-1", TargetURI: target}}
	}

	tests := []struct {
		name  string
		event *ari.PlaybackFinished

		channelResp *channel.Channel
		bridgeResp  *bridge.Bridge
		bridgeErr   error

		callErr  error
		extErr   error
		expectCh bool
		expectBr bool
		expectCa bool
		expectEx bool

		expectErr bool
	}{
		{name: "no target info", event: newEvent("nocolon")},
		{
			name: "channel normal", event: newEvent("channel:ch-1"),
			channelResp: &channel.Channel{ID: "ch-1"}, expectCh: true, expectCa: true,
		},
		{
			name: "channel call handler error", event: newEvent("channel:ch-1"),
			channelResp: &channel.Channel{ID: "ch-1"}, expectCh: true, expectCa: true,
			callErr: errors.New("call"), expectErr: true,
		},
		{
			name: "channel already hungup", event: newEvent("channel:ch-1"),
			channelResp: &channel.Channel{ID: "ch-1", TMEnd: &now}, expectCh: true,
		},
		{
			name: "bridge normal", event: newEvent("bridge:br-1"),
			bridgeResp: &bridge.Bridge{ID: "br-1"}, expectBr: true, expectEx: true,
		},
		{
			name: "bridge get failure", event: newEvent("bridge:br-1"),
			bridgeErr: errors.New("get"), expectBr: true, expectErr: true,
		},
		{
			name: "bridge already deleted", event: newEvent("bridge:br-1"),
			bridgeResp: &bridge.Bridge{ID: "br-1", TMDelete: &now}, expectBr: true,
		},
		{
			name: "bridge external media error", event: newEvent("bridge:br-1"),
			bridgeResp: &bridge.Bridge{ID: "br-1"}, expectBr: true, expectEx: true,
			extErr: errors.New("ext"), expectErr: true,
		},
		{name: "unsupported target", event: newEvent("recording:r-1"), expectErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockChannel := channelhandler.NewMockChannelHandler(mc)
			mockBridge := bridgehandler.NewMockBridgeHandler(mc)
			mockCall := callhandler.NewMockCallHandler(mc)
			mockExt := externalmediahandler.NewMockExternalMediaHandler(mc)

			h := eventHandler{
				channelHandler:       mockChannel,
				bridgeHandler:        mockBridge,
				callHandler:          mockCall,
				externalmediaHandler: mockExt,
			}
			ctx := context.Background()

			if tt.expectCh {
				mockChannel.EXPECT().UpdatePlaybackID(ctx, "ch-1", "").Return(tt.channelResp, nil)
			}
			if tt.expectCa {
				mockCall.EXPECT().ARIPlaybackFinished(ctx, tt.channelResp, tt.event).Return(tt.callErr)
			}
			if tt.expectBr {
				mockBridge.EXPECT().Get(ctx, "br-1").Return(tt.bridgeResp, tt.bridgeErr)
			}
			if tt.expectEx {
				mockExt.EXPECT().ARIPlaybackFinished(ctx, tt.bridgeResp, tt.event).Return(tt.extErr)
			}

			err := h.EventHandlerPlaybackFinished(ctx, tt.event)
			if (err != nil) != tt.expectErr {
				t.Errorf("Wrong match. expect err: %v, got: %v", tt.expectErr, err)
			}
		})
	}
}

func Test_EventHandlerPlaybackStarted_cov(t *testing.T) {
	tests := []struct {
		name      string
		target    string
		updateErr error
		expectUpd bool
	}{
		{name: "not channel target", target: "bridge:br-1"},
		{name: "update failure is ignored", target: "channel:ch-1", updateErr: errors.New("upd"), expectUpd: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockChannel := channelhandler.NewMockChannelHandler(mc)
			h := eventHandler{channelHandler: mockChannel}
			ctx := context.Background()

			if tt.expectUpd {
				mockChannel.EXPECT().UpdatePlaybackID(ctx, "ch-1", "pb-1").Return(nil, tt.updateErr)
			}

			e := &ari.PlaybackStarted{Playback: ari.Playback{ID: "pb-1", TargetURI: tt.target}}
			if err := h.EventHandlerPlaybackStarted(ctx, e); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
