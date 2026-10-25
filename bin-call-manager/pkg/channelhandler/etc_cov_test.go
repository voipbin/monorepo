package channelhandler

import (
	"context"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/channel"
)

// Test_etc_cov covers the Get-then-request family in etc.go: lookup failure, already hung up channel,
// request failure and the normal path.
func Test_etc_cov(t *testing.T) {
	id := "f2a3b4c5-0012-11f0-0000-000000000012"
	ast := "ast-1"

	tests := []struct {
		name string
		call func(h *channelHandler, ctx context.Context) error
		req  func(m *covMocks, ctx context.Context, err error)
	}{
		{
			name: "Answer",
			call: func(h *channelHandler, ctx context.Context) error { return h.Answer(ctx, id) },
			req: func(m *covMocks, ctx context.Context, err error) {
				m.req.EXPECT().AstChannelAnswer(ctx, ast, id).Return(err)
			},
		},
		{
			name: "DTMFSend",
			call: func(h *channelHandler, ctx context.Context) error { return h.DTMFSend(ctx, id, "1", 100, 0, 0, 0) },
			req: func(m *covMocks, ctx context.Context, err error) {
				m.req.EXPECT().AstChannelDTMF(ctx, ast, id, "1", 100, 0, 0, 0).Return(err)
			},
		},
		{
			name: "Record",
			call: func(h *channelHandler, ctx context.Context) error {
				return h.Record(ctx, id, "file", "wav", 10, 1, true, "#", "overwrite")
			},
			req: func(m *covMocks, ctx context.Context, err error) {
				m.req.EXPECT().AstChannelRecord(ctx, ast, id, "file", "wav", 10, 1, true, "#", "overwrite").Return(err)
			},
		},
		{
			name: "Dial",
			call: func(h *channelHandler, ctx context.Context) error { return h.Dial(ctx, id, "caller", 30) },
			req: func(m *covMocks, ctx context.Context, err error) {
				m.req.EXPECT().AstChannelDial(ctx, ast, id, "caller", 30).Return(err)
			},
		},
		{
			name: "Redirect",
			call: func(h *channelHandler, ctx context.Context) error { return h.Redirect(ctx, id, "ctx", "s", "1") },
			req: func(m *covMocks, ctx context.Context, err error) {
				m.req.EXPECT().AstAMIRedirect(ctx, ast, id, "ctx", "s", "1").Return(err)
			},
		},
		{
			name: "Continue",
			call: func(h *channelHandler, ctx context.Context) error { return h.Continue(ctx, id, "ctx", "s", 1, "") },
			req: func(m *covMocks, ctx context.Context, err error) {
				m.req.EXPECT().AstChannelContinue(ctx, ast, id, "ctx", "s", 1, "").Return(err)
			},
		},
		{
			name: "Ring",
			call: func(h *channelHandler, ctx context.Context) error { return h.Ring(ctx, id) },
			req: func(m *covMocks, ctx context.Context, err error) {
				m.req.EXPECT().AstChannelRing(ctx, ast, id).Return(err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name+"/normal", func(t *testing.T) {
			h, m := newCovHandler(t)
			ctx := context.Background()

			m.db.EXPECT().ChannelGet(gomock.Any(), id).Return(&channel.Channel{ID: id, AsteriskID: ast}, nil)
			tt.req(m, ctx, nil)

			if err := tt.call(h, ctx); err != nil {
				t.Errorf("Wrong match. expect: nil, got: %v", err)
			}
		})

		t.Run(tt.name+"/request error", func(t *testing.T) {
			h, m := newCovHandler(t)
			ctx := context.Background()

			m.db.EXPECT().ChannelGet(gomock.Any(), id).Return(&channel.Channel{ID: id, AsteriskID: ast}, nil)
			tt.req(m, ctx, errCov)

			if err := tt.call(h, ctx); err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			}
		})

		t.Run(tt.name+"/already hungup", func(t *testing.T) {
			h, m := newCovHandler(t)
			ctx := context.Background()
			now := time.Now()

			m.db.EXPECT().ChannelGet(gomock.Any(), id).Return(&channel.Channel{ID: id, AsteriskID: ast, TMDelete: &now}, nil)

			if err := tt.call(h, ctx); err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			}
		})

		t.Run(tt.name+"/get error", func(t *testing.T) {
			h, m := newCovHandler(t)

			// the deadline is shorter than defaultExistTimeout, so the lookup gives up quickly.
			ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
			defer cancel()

			m.db.EXPECT().ChannelGet(gomock.Any(), id).Return(nil, errCov).AnyTimes()

			if err := tt.call(h, ctx); err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			}
			checkNoGetWithTimeoutGoroutine(t)
		})
	}
}
