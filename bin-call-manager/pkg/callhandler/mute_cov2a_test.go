package callhandler

import (
	"context"
	"testing"

	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/channel"
)

func Test_MuteOn_cov2a(t *testing.T) {
	tests := []struct {
		name      string
		direction call.MuteDirection

		callLookupErr bool
		expectChannel call.MuteDirection

		wantErr string
	}{
		{
			name:          "call lookup error",
			direction:     call.MuteDirectionIn,
			callLookupErr: true,
			wantErr:       "Could not get call info.",
		},
		{
			name:          "none direction becomes both",
			direction:     call.MuteDirectionNone,
			expectChannel: call.MuteDirectionBoth,
		},
		{
			name:          "explicit direction is kept",
			direction:     call.MuteDirectionOut,
			expectChannel: call.MuteDirectionOut,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := cov2aNewHandler(t)
			ctx := context.Background()

			c := cov2aNewCall(nil)
			if tt.callLookupErr {
				m.db.EXPECT().CallGet(ctx, c.ID).Return(nil, errCov2aBoom)
			} else {
				updated := cov2aNewCall(nil)
				updated.MuteDirection = tt.expectChannel
				gomock.InOrder(
					m.db.EXPECT().CallGet(ctx, c.ID).Return(c, nil),
					m.channel.EXPECT().MuteOn(ctx, c.ChannelID, channel.MuteDirection(tt.expectChannel)).Return(nil),
					m.db.EXPECT().CallSetMuteDirection(ctx, c.ID, tt.expectChannel).Return(nil),
					m.db.EXPECT().CallGet(ctx, c.ID).Return(updated, nil),
					m.notify.EXPECT().PublishWebhookEvent(ctx, updated.CustomerID, call.EventTypeCallUpdated, updated),
				)
			}

			err := h.MuteOn(ctx, c.ID, tt.direction)
			cov2aCheckErr(t, err, tt.wantErr)
		})
	}
}

func Test_MuteOff_cov2a(t *testing.T) {
	tests := []struct {
		name          string
		current       call.MuteDirection
		direction     call.MuteDirection
		expectChannel call.MuteDirection
		expectNew     call.MuteDirection

		callLookupErr bool
		wantErr       string
	}{
		{
			name:          "call lookup error",
			callLookupErr: true,
			wantErr:       "Could not get call info.",
		},
		{
			name:          "none direction becomes both and clears everything",
			current:       call.MuteDirectionBoth,
			direction:     call.MuteDirectionNone,
			expectChannel: call.MuteDirectionBoth,
			expectNew:     call.MuteDirectionNone,
		},
		{
			name:          "unmute in from both leaves out",
			current:       call.MuteDirectionBoth,
			direction:     call.MuteDirectionIn,
			expectChannel: call.MuteDirectionIn,
			expectNew:     call.MuteDirectionOut,
		},
		{
			name:          "unmute out from both leaves in",
			current:       call.MuteDirectionBoth,
			direction:     call.MuteDirectionOut,
			expectChannel: call.MuteDirectionOut,
			expectNew:     call.MuteDirectionIn,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := cov2aNewHandler(t)
			ctx := context.Background()

			c := cov2aNewCall(nil)
			c.MuteDirection = tt.current
			if tt.callLookupErr {
				m.db.EXPECT().CallGet(ctx, c.ID).Return(nil, errCov2aBoom)
			} else {
				updated := cov2aNewCall(nil)
				updated.MuteDirection = tt.expectNew
				gomock.InOrder(
					m.db.EXPECT().CallGet(ctx, c.ID).Return(c, nil),
					m.channel.EXPECT().MuteOff(ctx, c.ChannelID, channel.MuteDirection(tt.expectChannel)).Return(nil),
					m.db.EXPECT().CallSetMuteDirection(ctx, c.ID, tt.expectNew).Return(nil),
					m.db.EXPECT().CallGet(ctx, c.ID).Return(updated, nil),
					m.notify.EXPECT().PublishWebhookEvent(ctx, updated.CustomerID, call.EventTypeCallUpdated, updated),
				)
			}

			err := h.MuteOff(ctx, c.ID, tt.direction)
			cov2aCheckErr(t, err, tt.wantErr)
		})
	}
}
