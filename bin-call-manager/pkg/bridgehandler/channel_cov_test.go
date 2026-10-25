package bridgehandler

import (
	"context"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/bridge"
)

// Test_getFamily_errors_cov covers the error branches of the Get-then-request handlers
// (ChannelKick, ChannelJoin, Destroy, Play).
func Test_getFamily_errors_cov(t *testing.T) {
	id := "6c1d2e3f-0007-11f0-8000-000000000007"
	br := &bridge.Bridge{ID: id, AsteriskID: "ast-1"}

	tests := []struct {
		name string
		call func(h *bridgeHandler, ctx context.Context) error
		req  func(m *covMocks, ctx context.Context)
	}{
		{
			name: "ChannelKick",
			call: func(h *bridgeHandler, ctx context.Context) error { return h.ChannelKick(ctx, id, "ch-1") },
			req: func(m *covMocks, ctx context.Context) {
				m.req.EXPECT().AstBridgeRemoveChannel(ctx, "ast-1", id, "ch-1").Return(errCov)
			},
		},
		{
			name: "ChannelJoin",
			call: func(h *bridgeHandler, ctx context.Context) error {
				return h.ChannelJoin(ctx, id, "ch-1", "", false, false)
			},
			req: func(m *covMocks, ctx context.Context) {
				m.req.EXPECT().AstBridgeAddChannel(ctx, "ast-1", id, "ch-1", "", false, false).Return(errCov)
			},
		},
		{
			name: "Destroy",
			call: func(h *bridgeHandler, ctx context.Context) error { return h.Destroy(ctx, id) },
			req: func(m *covMocks, ctx context.Context) {
				m.req.EXPECT().AstBridgeDelete(ctx, "ast-1", id).Return(errCov)
			},
		},
		{
			name: "Play",
			call: func(h *bridgeHandler, ctx context.Context) error {
				return h.Play(ctx, id, "pb-1", []string{"sound:a"}, "en", 0, 0)
			},
			req: func(m *covMocks, ctx context.Context) {
				m.req.EXPECT().AstBridgePlay(ctx, "ast-1", id, []string{"sound:a"}, "en", 0, 0, "pb-1").Return(nil, errCov)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name+"/request error", func(t *testing.T) {
			h, m := newCovHandler(t)
			ctx := context.Background()

			m.db.EXPECT().BridgeGet(gomock.Any(), id).Return(br, nil)
			tt.req(m, ctx)

			if err := tt.call(h, ctx); err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			}
		})

		t.Run(tt.name+"/get error", func(t *testing.T) {
			h, m := newCovHandler(t)
			ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
			defer cancel()

			m.db.EXPECT().BridgeGet(gomock.Any(), id).Return(nil, errCov).AnyTimes()

			if err := tt.call(h, ctx); err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			}
			checkNoGetWithTimeoutGoroutine(t)
		})
	}
}

func Test_Play_hungup_cov(t *testing.T) {
	h, m := newCovHandler(t)
	id := "6c1d2e3f-0008-11f0-8000-000000000008"
	now := time.Now()

	m.db.EXPECT().BridgeGet(gomock.Any(), id).Return(&bridge.Bridge{ID: id, TMDelete: &now}, nil)

	if err := h.Play(context.Background(), id, "pb-1", nil, "en", 0, 0); err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
}
