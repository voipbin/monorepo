package channelhandler

import (
	"context"
	stderrors "errors"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/channel"
)

func Test_VariableSet_cov2_errors(t *testing.T) {
	ctx := context.Background()
	tmDelete := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name string

		channel *channel.Channel
		setup   func(m *covMocks)

		expectMsg string
		expectIs  error
	}{
		{
			name:      "channel hungup already",
			channel:   &channel.Channel{ID: cov2ChannelID, TMDelete: &tmDelete},
			expectMsg: "already hungup",
		},
		{
			name:    "set variable failed",
			channel: &channel.Channel{ID: cov2ChannelID, AsteriskID: cov2AsteriskID},
			setup: func(m *covMocks) {
				m.req.EXPECT().AstChannelVariableSet(ctx, cov2AsteriskID, cov2ChannelID, "KEY", "value").Return(errCov)
			},
			expectIs: errCov,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCovHandler(t)

			m.db.EXPECT().ChannelGet(gomock.Any(), cov2ChannelID).Return(tt.channel, nil)
			if tt.setup != nil {
				tt.setup(m)
			}

			err := h.VariableSet(ctx, cov2ChannelID, "KEY", "value")
			if err == nil {
				t.Fatalf("Wrong match. expect: error, got: nil")
			}
			if tt.expectMsg != "" && err.Error() != tt.expectMsg {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expectMsg, err.Error())
			}
			if tt.expectIs != nil && !stderrors.Is(err, tt.expectIs) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectIs, err)
			}
		})
	}
}

func Test_VariableSet_cov2_getFailed(t *testing.T) {
	cov2RunGetFail(t, cov2ChannelID, func(h *channelHandler, ctx context.Context) error {
		return h.VariableSet(ctx, cov2ChannelID, "KEY", "value")
	}, "not found")
}
