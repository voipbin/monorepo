package channelhandler

import (
	"context"
	stderrors "errors"
	"strings"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/channel"
)

func Test_StartSnoop_cov2_errors(t *testing.T) {
	ctx := context.Background()
	tmDelete := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name string

		channel *channel.Channel
		setup   func(m *covMocks)

		expectMsg string
	}{
		{
			name:      "channel hungup already",
			channel:   &channel.Channel{ID: cov2ChannelID, TMDelete: &tmDelete},
			expectMsg: "the channel has hungup already",
		},
		{
			name:    "create snoop failed",
			channel: &channel.Channel{ID: cov2ChannelID, AsteriskID: cov2AsteriskID},
			setup: func(m *covMocks) {
				m.req.EXPECT().AstChannelCreateSnoop(ctx, cov2AsteriskID, cov2ChannelID, "c2-snoop", "c2-args", channel.SnoopDirectionIn, channel.SnoopDirectionOut).Return(nil, errCov)
			},
			expectMsg: "could not create a snoop channel",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCovHandler(t)

			m.db.EXPECT().ChannelGet(gomock.Any(), cov2ChannelID).Return(tt.channel, nil)
			if tt.setup != nil {
				tt.setup(m)
			}

			res, err := h.StartSnoop(ctx, cov2ChannelID, "c2-snoop", "c2-args", channel.SnoopDirectionIn, channel.SnoopDirectionOut)
			if res != nil {
				t.Errorf("Wrong match. expect: nil result, got: %v", res)
			}
			if err == nil || !strings.Contains(err.Error(), tt.expectMsg) {
				t.Errorf("Wrong match. expect message contains: %s, got: %v", tt.expectMsg, err)
			}
		})
	}
}

func Test_StartSnoop_cov2_getFailed(t *testing.T) {
	cov2RunGetFail(t, cov2ChannelID, func(h *channelHandler, ctx context.Context) error {
		_, err := h.StartSnoop(ctx, cov2ChannelID, "c2-snoop", "c2-args", channel.SnoopDirectionIn, channel.SnoopDirectionOut)
		return err
	}, "could not get channel info")
}

func Test_StartChannelWithBaseChannel_cov2_getFailed(t *testing.T) {
	cov2RunGetFail(t, cov2ChannelID, func(h *channelHandler, ctx context.Context) error {
		_, err := h.StartChannelWithBaseChannel(ctx, cov2ChannelID, "c2-new", "c2-args", "PJSIP/test", "", "", "", nil)
		return err
	}, "could not get base channel")
}

func Test_StartChannel_cov2_requestFailures(t *testing.T) {
	ctx := context.Background()
	variables := map[string]string{"KEY": "value"}

	tests := []struct {
		name string

		setup func(m *covMocks)
		run   func(h *channelHandler) (*channel.Channel, error)

		expectMsg string
	}{
		{
			name: "external media failed",
			setup: func(m *covMocks) {
				m.req.EXPECT().AstChannelExternalMedia(ctx, cov2AsteriskID, "c2-new", "192.0.2.1:5000", "rtp", "udp", "", "client", "ulaw", "both", "c2-data", variables).Return(nil, errCov)
			},
			run: func(h *channelHandler) (*channel.Channel, error) {
				return h.StartExternalMedia(ctx, cov2AsteriskID, "c2-new", "192.0.2.1:5000", "rtp", "udp", "", "client", "ulaw", "both", "c2-data", variables)
			},
			expectMsg: "could not create an external media channel",
		},
		{
			name: "start channel failed",
			setup: func(m *covMocks) {
				m.req.EXPECT().AstChannelCreate(ctx, cov2AsteriskID, "c2-new", "c2-args", "PJSIP/test", "c2-other", "c2-originator", "ulaw", variables).Return(nil, errCov)
			},
			run: func(h *channelHandler) (*channel.Channel, error) {
				return h.StartChannel(ctx, cov2AsteriskID, "c2-new", "c2-args", "PJSIP/test", "c2-other", "c2-originator", "ulaw", variables)
			},
			expectMsg: "could not create a channel",
		},
		{
			name: "start channel with base channel failed",
			setup: func(m *covMocks) {
				m.db.EXPECT().ChannelGet(gomock.Any(), cov2ChannelID).Return(&channel.Channel{ID: cov2ChannelID, AsteriskID: cov2AsteriskID}, nil)
				m.req.EXPECT().AstChannelCreate(ctx, cov2AsteriskID, "c2-new", "c2-args", "PJSIP/test", "c2-other", "c2-originator", "ulaw", variables).Return(nil, errCov)
			},
			run: func(h *channelHandler) (*channel.Channel, error) {
				return h.StartChannelWithBaseChannel(ctx, cov2ChannelID, "c2-new", "c2-args", "PJSIP/test", "c2-other", "c2-originator", "ulaw", variables)
			},
			expectMsg: "could not create a channel",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCovHandler(t)
			tt.setup(m)

			res, err := tt.run(h)
			if res != nil {
				t.Errorf("Wrong match. expect: nil result, got: %v", res)
			}
			if err == nil || !strings.Contains(err.Error(), tt.expectMsg) {
				t.Errorf("Wrong match. expect message contains: %s, got: %v", tt.expectMsg, err)
			}
			if !stderrors.Is(err, errCov) {
				t.Errorf("Wrong match. expect: wrapped errCov, got: %v", err)
			}
		})
	}
}
