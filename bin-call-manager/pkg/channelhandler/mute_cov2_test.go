package channelhandler

import (
	"context"
	"testing"

	"go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/channel"
)

func Test_Mute_cov2_getFailed(t *testing.T) {
	tests := []struct {
		name string
		run  func(h *channelHandler, ctx context.Context) error
	}{
		{"mute on", func(h *channelHandler, ctx context.Context) error {
			return h.MuteOn(ctx, cov2ChannelID, channel.MuteDirectionBoth)
		}},
		{"mute off", func(h *channelHandler, ctx context.Context) error {
			return h.MuteOff(ctx, cov2ChannelID, channel.MuteDirectionBoth)
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cov2RunGetFail(t, cov2ChannelID, tt.run, "could not get channel info")
		})
	}
}

func Test_Mute_cov2_updateDirectionFailureIsTolerated(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name string

		isOn      bool
		current   channel.MuteDirection
		direction channel.MuteDirection

		expectDirection channel.MuteDirection
	}{
		{"mute on", true, channel.MuteDirectionNone, channel.MuteDirectionIn, channel.MuteDirectionIn},
		{"mute off both", false, channel.MuteDirectionBoth, channel.MuteDirectionBoth, channel.MuteDirectionNone},
		{"mute off one side of both", false, channel.MuteDirectionBoth, channel.MuteDirectionIn, channel.MuteDirectionOut},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCovHandler(t)

			m.db.EXPECT().ChannelGet(gomock.Any(), cov2ChannelID).Return(&channel.Channel{
				ID:            cov2ChannelID,
				AsteriskID:    cov2AsteriskID,
				MuteDirection: tt.current,
			}, nil)
			if tt.isOn {
				m.req.EXPECT().AstChannelMuteOn(ctx, cov2AsteriskID, cov2ChannelID, string(tt.direction)).Return(nil)
			} else {
				m.req.EXPECT().AstChannelMuteOff(ctx, cov2AsteriskID, cov2ChannelID, string(tt.direction)).Return(nil)
			}
			m.db.EXPECT().ChannelSetMuteDirection(ctx, cov2ChannelID, tt.expectDirection).Return(errCov)

			var err error
			if tt.isOn {
				err = h.MuteOn(ctx, cov2ChannelID, tt.direction)
			} else {
				err = h.MuteOff(ctx, cov2ChannelID, tt.direction)
			}
			if err != nil {
				t.Errorf("Wrong match. expect: nil, got: %v", err)
			}
		})
	}
}
