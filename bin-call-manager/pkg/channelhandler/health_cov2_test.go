package channelhandler

import (
	"context"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/channel"
)

func Test_HealthCheck_cov2(t *testing.T) {
	ctx := context.Background()
	tmEnd := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	cn := &channel.Channel{ID: cov2ChannelID, AsteriskID: cov2AsteriskID}

	tests := []struct {
		name string

		retryCount int
		setup      func(m *covMocks)
	}{
		{
			name:       "channel is already ended",
			retryCount: 0,
			setup: func(m *covMocks) {
				m.db.EXPECT().ChannelGet(gomock.Any(), cov2ChannelID).Return(&channel.Channel{ID: cov2ChannelID, TMEnd: &tmEnd}, nil)
			},
		},
		{
			name:       "asterisk channel lookup failed increases the retry count",
			retryCount: 1,
			setup: func(m *covMocks) {
				m.db.EXPECT().ChannelGet(gomock.Any(), cov2ChannelID).Return(cn, nil)
				m.req.EXPECT().AstChannelGet(ctx, cov2AsteriskID, cov2ChannelID).Return(nil, errCov)
				m.req.EXPECT().CallV1ChannelHealth(ctx, cov2ChannelID, defaultHealthDelay, 2).Return(nil)
			},
		},
		{
			name:       "asterisk channel exists resets the retry count and health request failure is tolerated",
			retryCount: 2,
			setup: func(m *covMocks) {
				m.db.EXPECT().ChannelGet(gomock.Any(), cov2ChannelID).Return(cn, nil)
				m.req.EXPECT().AstChannelGet(ctx, cov2AsteriskID, cov2ChannelID).Return(cn, nil)
				m.req.EXPECT().CallV1ChannelHealth(ctx, cov2ChannelID, defaultHealthDelay, 0).Return(errCov)
			},
		},
		{
			name:       "exceeded retry count and publishing the fake event failed",
			retryCount: defaultHealthMaxRetryCount + 1,
			setup: func(m *covMocks) {
				m.req.EXPECT().CallPublishEvent(ctx, "ChannelDestroyed", "asterisk-proxy", "application/json", gomock.Any()).Return(errCov)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCovHandler(t)
			tt.setup(m)

			h.HealthCheck(ctx, cov2ChannelID, tt.retryCount)
		})
	}
}

func Test_HealthCheck_cov2_getFailed(t *testing.T) {
	h, m := newCovHandler(t)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// no request is expected once the channel lookup fails.
	m.db.EXPECT().ChannelGet(gomock.Any(), cov2ChannelID).Return(nil, errCov).AnyTimes()

	h.HealthCheck(ctx, cov2ChannelID, 0)

	checkNoGetWithTimeoutGoroutine(t)
}
