package channelhandler

import (
	"context"
	"testing"
	"time"

	"github.com/pkg/errors"

	cerrors "monorepo/bin-common-handler/models/errors"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/channel"
)

func Test_HangingUp_cov(t *testing.T) {
	id := "a3b4c5d6-0013-11f0-0000-000000000013"
	cause := ari.ChannelCauseNormalClearing

	t.Run("get error", func(t *testing.T) {
		h, m := newCovHandler(t)
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		m.db.EXPECT().ChannelGet(gomock.Any(), id).Return(nil, errCov).AnyTimes()

		if _, err := h.HangingUp(ctx, id, cause); err == nil {
			t.Errorf("Wrong match. expect: error, got: nil")
		}
		checkNoGetWithTimeoutGoroutine(t)
	})

	t.Run("already hungup", func(t *testing.T) {
		h, m := newCovHandler(t)
		ctx := context.Background()
		now := time.Now()

		m.db.EXPECT().ChannelGet(gomock.Any(), id).Return(&channel.Channel{ID: id, TMDelete: &now}, nil)

		res, err := h.HangingUp(ctx, id, cause)
		if err != nil || res == nil || res.ID != id {
			t.Errorf("Wrong match. got: (%v, %v)", res, err)
		}
	})

	t.Run("hangup error", func(t *testing.T) {
		h, m := newCovHandler(t)
		ctx := context.Background()

		m.db.EXPECT().ChannelGet(gomock.Any(), id).Return(&channel.Channel{ID: id, AsteriskID: "ast-1"}, nil)
		m.req.EXPECT().AstChannelHangup(ctx, "ast-1", id, cause, 0).Return(errCov)

		if _, err := h.HangingUp(ctx, id, cause); err == nil {
			t.Errorf("Wrong match. expect: error, got: nil")
		}
	})
}

func Test_HangingUpWithAsteriskID_cov(t *testing.T) {
	id := "b4c5d6e7-0014-11f0-0000-000000000014"
	cause := ari.ChannelCauseNormalClearing

	tests := []struct {
		name      string
		reqErr    error
		expectErr bool
	}{
		{name: "normal"},
		{name: "typed not found is treated as hungup", reqErr: cerrors.NotFound("call-manager", "CHANNEL_NOT_FOUND", "not found")},
		{name: "legacy not found is treated as hungup", reqErr: errors.Wrap(requesthandler.ErrNotFound, "wrapped")},
		{name: "other error", reqErr: errCov, expectErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCovHandler(t)
			ctx := context.Background()

			m.req.EXPECT().AstChannelHangup(ctx, "ast-1", id, cause, 0).Return(tt.reqErr)

			err := h.HangingUpWithAsteriskID(ctx, "ast-1", id, cause)
			if (err != nil) != tt.expectErr {
				t.Errorf("Wrong match. expect error: %v, got: %v", tt.expectErr, err)
			}
		})
	}
}

func Test_HangingUpWithDelay_cov(t *testing.T) {
	id := "c5d6e7f8-0015-11f0-0000-000000000015"
	cause := ari.ChannelCauseNormalClearing

	t.Run("get error", func(t *testing.T) {
		h, m := newCovHandler(t)
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		m.db.EXPECT().ChannelGet(gomock.Any(), id).Return(nil, errCov).AnyTimes()

		if _, err := h.HangingUpWithDelay(ctx, id, cause, 1000); err == nil {
			t.Errorf("Wrong match. expect: error, got: nil")
		}
		checkNoGetWithTimeoutGoroutine(t)
	})

	t.Run("already hungup", func(t *testing.T) {
		h, m := newCovHandler(t)
		now := time.Now()

		m.db.EXPECT().ChannelGet(gomock.Any(), id).Return(&channel.Channel{ID: id, TMDelete: &now}, nil)

		if _, err := h.HangingUpWithDelay(context.Background(), id, cause, 1000); err == nil {
			t.Errorf("Wrong match. expect: error, got: nil")
		}
	})

	t.Run("hangup error", func(t *testing.T) {
		h, m := newCovHandler(t)
		ctx := context.Background()

		m.db.EXPECT().ChannelGet(gomock.Any(), id).Return(&channel.Channel{ID: id, AsteriskID: "ast-1"}, nil)
		m.req.EXPECT().AstChannelHangup(ctx, "ast-1", id, cause, 1000).Return(errCov)

		if _, err := h.HangingUpWithDelay(ctx, id, cause, 1000); err == nil {
			t.Errorf("Wrong match. expect: error, got: nil")
		}
	})

	t.Run("normal", func(t *testing.T) {
		h, m := newCovHandler(t)
		ctx := context.Background()

		m.db.EXPECT().ChannelGet(gomock.Any(), id).Return(&channel.Channel{ID: id, AsteriskID: "ast-1"}, nil)
		m.req.EXPECT().AstChannelHangup(ctx, "ast-1", id, cause, 1000).Return(nil)

		res, err := h.HangingUpWithDelay(ctx, id, cause, 1000)
		if err != nil || res == nil || res.ID != id {
			t.Errorf("Wrong match. got: (%v, %v)", res, err)
		}
	})
}
