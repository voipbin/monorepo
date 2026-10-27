package channelhandler

import (
	"context"
	"testing"
)

func Test_HoldMOHSilence_cov2_getFailed(t *testing.T) {
	tests := []struct {
		name string
		run  func(h *channelHandler, ctx context.Context) error
	}{
		{"hold on", func(h *channelHandler, ctx context.Context) error { return h.HoldOn(ctx, cov2ChannelID) }},
		{"hold off", func(h *channelHandler, ctx context.Context) error { return h.HoldOff(ctx, cov2ChannelID) }},
		{"moh on", func(h *channelHandler, ctx context.Context) error { return h.MOHOn(ctx, cov2ChannelID) }},
		{"moh off", func(h *channelHandler, ctx context.Context) error { return h.MOHOff(ctx, cov2ChannelID) }},
		{"silence on", func(h *channelHandler, ctx context.Context) error { return h.SilenceOn(ctx, cov2ChannelID) }},
		{"silence off", func(h *channelHandler, ctx context.Context) error { return h.SilenceOff(ctx, cov2ChannelID) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cov2RunGetFail(t, cov2ChannelID, tt.run, "could not get channel info")
		})
	}
}
