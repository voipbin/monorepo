package channelhandler

import (
	"context"
	"strings"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/channel"
)

func Test_Playback_cov2_getFailed(t *testing.T) {
	tests := []struct {
		name string
		run  func(h *channelHandler, ctx context.Context) error
	}{
		{"playback stop", func(h *channelHandler, ctx context.Context) error { return h.PlaybackStop(ctx, cov2ChannelID) }},
		{"play", func(h *channelHandler, ctx context.Context) error {
			return h.Play(ctx, cov2ChannelID, "c2-playback", []string{"sound:hello"}, "en", 0, 0)
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cov2RunGetFail(t, cov2ChannelID, tt.run, "could not get channel info")
		})
	}
}

func Test_Playback_cov2_errors(t *testing.T) {
	ctx := context.Background()
	tmDelete := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name string

		isPlay  bool
		channel *channel.Channel
		setup   func(m *covMocks)

		expectMsg string
	}{
		{
			name:      "stop: channel hungup already",
			channel:   &channel.Channel{ID: cov2ChannelID, TMDelete: &tmDelete},
			expectMsg: "the channel has hungup already",
		},
		{
			name:    "stop: playback stop failed",
			channel: &channel.Channel{ID: cov2ChannelID, AsteriskID: cov2AsteriskID, PlaybackID: "c2-playback"},
			setup: func(m *covMocks) {
				m.req.EXPECT().AstPlaybackStop(ctx, cov2AsteriskID, "c2-playback").Return(errCov)
			},
			expectMsg: "could not stop the playback",
		},
		{
			name:      "play: channel hungup already",
			isPlay:    true,
			channel:   &channel.Channel{ID: cov2ChannelID, TMDelete: &tmDelete},
			expectMsg: "the channel has hungup already",
		},
		{
			name:    "play: request failed",
			isPlay:  true,
			channel: &channel.Channel{ID: cov2ChannelID, AsteriskID: cov2AsteriskID},
			setup: func(m *covMocks) {
				m.req.EXPECT().AstChannelPlay(ctx, cov2AsteriskID, cov2ChannelID, []string{"sound:hello"}, "en", 100, 200, "c2-playback").Return(nil, errCov)
			},
			expectMsg: "could not play the media",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCovHandler(t)

			m.db.EXPECT().ChannelGet(gomock.Any(), cov2ChannelID).Return(tt.channel, nil)
			if tt.setup != nil {
				tt.setup(m)
			}

			var err error
			if tt.isPlay {
				err = h.Play(ctx, cov2ChannelID, "c2-playback", []string{"sound:hello"}, "en", 100, 200)
			} else {
				err = h.PlaybackStop(ctx, cov2ChannelID)
			}
			if err == nil || !strings.Contains(err.Error(), tt.expectMsg) {
				t.Errorf("Wrong match. expect message contains: %s, got: %v", tt.expectMsg, err)
			}
		})
	}
}
