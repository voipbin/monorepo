package confbridgehandler

import (
	"context"
	stderrors "errors"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/bridge"
	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-call-manager/models/confbridge"
)

func Test_Leaved_cov2_errors(t *testing.T) {
	ctx := context.Background()

	cn := &channel.Channel{
		ID: "c2-channel-1",
		StasisData: map[channel.StasisDataType]string{
			channel.StasisDataTypeConfbridgeID: cov2CBID.String(),
			channel.StasisDataTypeCallID:       cov2CallID.String(),
		},
	}

	t.Run("remove channel call id failed", func(t *testing.T) {
		h, m := newCov2Handler(t)

		m.channel.EXPECT().HangingUp(ctx, cn.ID, ari.ChannelCauseNormalClearing).Return(&channel.Channel{}, nil)
		m.db.EXPECT().ConfbridgeRemoveChannelCallID(ctx, cov2CBID, cn.ID).Return(errCov2)

		err := h.Leaved(ctx, cn, &bridge.Bridge{})
		if !stderrors.Is(err, errCov2) {
			t.Errorf("Wrong match. expect: wrapped errCov2, got: %v", err)
		}
	})

	t.Run("call update failure and terminating failure", func(t *testing.T) {
		h, m := newCov2Handler(t)

		cb := cov2CB(func(c *confbridge.Confbridge) {
			c.Type = confbridge.TypeConnect
			c.ChannelCallIDs = map[string]uuid.UUID{"c2-ch-2": cov2CallID2}
		})

		m.channel.EXPECT().HangingUp(ctx, cn.ID, ari.ChannelCauseNormalClearing).Return(&channel.Channel{}, nil)
		m.db.EXPECT().ConfbridgeRemoveChannelCallID(ctx, cov2CBID, cn.ID).Return(nil)
		m.db.EXPECT().ConfbridgeGet(ctx, cov2CBID).Return(cb, nil)
		m.notify.EXPECT().PublishEvent(ctx, confbridge.EventTypeConfbridgeLeaved, &confbridge.EventConfbridgeLeaved{Confbridge: *cb, LeavedCallID: cov2CallID})

		// the call update runs in a goroutine; wait for it through the channel.
		done := make(chan struct{})
		m.req.EXPECT().CallV1CallUpdateConfbridgeID(ctx, cov2CallID, uuid.Nil).DoAndReturn(
			func(_ context.Context, _ uuid.UUID, _ uuid.UUID) (*call.Call, error) {
				defer close(done)
				return nil, errCov2
			})

		// only one channel remains and the auto leave is allowed, so Terminating is called and fails at Get.
		m.db.EXPECT().ConfbridgeGet(ctx, cov2CBID).Return(nil, errCov2)

		err := h.Leaved(ctx, cn, &bridge.Bridge{})
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatalf("Timed out waiting for the update goroutine")
		}
		if !stderrors.Is(err, errCov2) {
			t.Errorf("Wrong match. expect: wrapped errCov2, got: %v", err)
		}
	})
}
