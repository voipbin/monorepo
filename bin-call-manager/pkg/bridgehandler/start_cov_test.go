package bridgehandler

import (
	"context"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/bridge"
)

func Test_Start_cov(t *testing.T) {
	asteriskID := "42:01:0a:a4:00:05"
	bridgeID := "6c1d2e3f-0001-11f0-8000-000000000001"
	types := []bridge.Type{bridge.TypeMixing}

	t.Run("normal", func(t *testing.T) {
		h, m := newCovHandler(t)
		ctx := context.Background()

		m.req.EXPECT().AstBridgeCreate(ctx, asteriskID, bridgeID, "name", types).Return(nil)
		m.db.EXPECT().BridgeGet(gomock.Any(), bridgeID).Return(&bridge.Bridge{ID: bridgeID}, nil)

		res, err := h.Start(ctx, asteriskID, bridgeID, "name", types)
		if err != nil || res == nil || res.ID != bridgeID {
			t.Errorf("Wrong match. got: (%v, %v)", res, err)
		}
	})

	t.Run("create error", func(t *testing.T) {
		h, m := newCovHandler(t)
		ctx := context.Background()

		m.req.EXPECT().AstBridgeCreate(ctx, asteriskID, bridgeID, "name", types).Return(errCov)

		res, err := h.Start(ctx, asteriskID, bridgeID, "name", types)
		if res != nil || err == nil {
			t.Errorf("Wrong match. expect: (nil, error), got: (%v, %v)", res, err)
		}
	})

	t.Run("get error", func(t *testing.T) {
		h, m := newCovHandler(t)

		// the deadline is shorter than defaultExistTimeout, so the lookup gives up quickly.
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		m.req.EXPECT().AstBridgeCreate(ctx, asteriskID, bridgeID, "name", types).Return(nil)
		m.db.EXPECT().BridgeGet(gomock.Any(), bridgeID).Return(nil, errCov).AnyTimes()

		res, err := h.Start(ctx, asteriskID, bridgeID, "name", types)
		if res != nil || err == nil {
			t.Errorf("Wrong match. expect: (nil, error), got: (%v, %v)", res, err)
		}
		checkNoGetWithTimeoutGoroutine(t)
	})
}
