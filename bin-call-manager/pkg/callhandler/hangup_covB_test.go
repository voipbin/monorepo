package callhandler

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-call-manager/pkg/dbhandler"
)

func Test_Hangup_error_covB(t *testing.T) {
	id := uuid.FromStringOrNil(covBCallID)
	cn := &channel.Channel{ID: "ch-1"}

	t.Run("get call error", func(t *testing.T) {
		h, m := newCovBHandler(t)
		ctx := context.Background()
		m.db.EXPECT().CallGetByChannelID(ctx, cn.ID).Return(nil, fmt.Errorf("err"))
		if _, err := h.Hangup(ctx, cn); err == nil {
			t.Errorf("expected error")
		}
	})

	t.Run("update hangup info error", func(t *testing.T) {
		h, m := newCovBHandler(t)
		ctx := context.Background()
		c := covBCall(covBCallID, call.StatusRinging)
		c.Direction = call.DirectionIncoming
		m.db.EXPECT().CallGetByChannelID(ctx, cn.ID).Return(c, nil)
		m.bridge.EXPECT().Destroy(ctx, c.BridgeID).Return(fmt.Errorf("destroy err"))
		m.db.EXPECT().CallSetHangup(ctx, id, call.HangupReasonNormal, call.HangupByRemote).Return(fmt.Errorf("err"))
		if _, err := h.Hangup(ctx, cn); err == nil {
			t.Errorf("expected error")
		}
	})

	t.Run("moved to another channel", func(t *testing.T) {
		h, m := newCovBHandler(t)
		ctx := context.Background()
		c := covBCall(covBCallID, call.StatusProgressing)
		c.Direction = call.DirectionIncoming
		cur := covBCall(covBCallID, call.StatusProgressing)
		cur.ChannelID = "ch-other"
		m.db.EXPECT().CallGetByChannelID(ctx, cn.ID).Return(c, nil)
		m.bridge.EXPECT().Destroy(ctx, c.BridgeID).Return(nil)
		m.db.EXPECT().CallSetHangupIfChannel(ctx, id, cn.ID, call.HangupReasonNormal, call.HangupByRemote).Return(false, nil)
		m.db.EXPECT().CallGetFromDB(ctx, id).Return(cur, nil)
		res, err := h.Hangup(ctx, cn)
		if err != nil || res != cur {
			t.Errorf("unexpected result. res: %v, err: %v", res, err)
		}
	})

	t.Run("moved and get from db error", func(t *testing.T) {
		h, m := newCovBHandler(t)
		ctx := context.Background()
		c := covBCall(covBCallID, call.StatusProgressing)
		c.Direction = call.DirectionIncoming
		m.db.EXPECT().CallGetByChannelID(ctx, cn.ID).Return(c, nil)
		m.bridge.EXPECT().Destroy(ctx, c.BridgeID).Return(nil)
		m.db.EXPECT().CallSetHangupIfChannel(ctx, id, cn.ID, call.HangupReasonNormal, call.HangupByRemote).Return(false, nil)
		m.db.EXPECT().CallGetFromDB(ctx, id).Return(nil, fmt.Errorf("err"))
		if _, err := h.Hangup(ctx, cn); err == nil {
			t.Errorf("expected error")
		}
	})
}

func Test_HangingUp_error_covB(t *testing.T) {
	id := uuid.FromStringOrNil(covBCallID)

	t.Run("get error", func(t *testing.T) {
		h, m := newCovBHandler(t)
		ctx := context.Background()
		m.db.EXPECT().CallGet(ctx, id).Return(nil, dbhandler.ErrNotFound)
		if _, err := h.HangingUp(ctx, id, call.HangupReasonNormal); err == nil {
			t.Errorf("expected error")
		}
	})

	t.Run("already terminating", func(t *testing.T) {
		h, m := newCovBHandler(t)
		ctx := context.Background()
		c := covBCall(covBCallID, call.StatusTerminating)
		m.db.EXPECT().CallGet(ctx, id).Return(c, nil)
		res, err := h.HangingUp(ctx, id, call.HangupReasonNormal)
		if err != nil || res != c {
			t.Errorf("unexpected result: %v, %v", res, err)
		}
	})

	t.Run("update status error", func(t *testing.T) {
		h, m := newCovBHandler(t)
		ctx := context.Background()
		c := covBCall(covBCallID, call.StatusProgressing)
		c.Direction = call.DirectionIncoming
		m.db.EXPECT().CallGet(ctx, id).Return(c, nil)
		m.db.EXPECT().CallSetStatus(ctx, id, call.StatusTerminating).Return(fmt.Errorf("err"))
		if _, err := h.HangingUp(ctx, id, call.HangupReasonNormal); err == nil {
			t.Errorf("expected error")
		}
	})
}

func Test_hangingUpWithCause_channel_covB(t *testing.T) {
	id := uuid.FromStringOrNil(covBCallID)
	cause := ari.ChannelCauseNormalClearing

	setup := func(t *testing.T, channelID string) (*callHandler, *covBMocks, *call.Call, *call.Call) {
		h, m := newCovBHandler(t)
		ctx := context.Background()
		c := covBCall(covBCallID, call.StatusProgressing)
		c.Direction = call.DirectionIncoming
		c.ChannelID = "ch-1"
		upd := covBCall(covBCallID, call.StatusTerminating)
		upd.Direction = call.DirectionIncoming
		m.db.EXPECT().CallGet(ctx, id).Return(c, nil).Times(1)
		m.db.EXPECT().CallSetStatus(ctx, id, call.StatusTerminating).Return(nil)
		m.db.EXPECT().CallGet(ctx, id).Return(upd, nil).Times(1)
		m.notify.EXPECT().PublishWebhookEvent(gomock.Any(), upd.CustomerID, gomock.Any(), upd)
		return h, m, c, upd
	}

	t.Run("channel hangup error, same channel", func(t *testing.T) {
		h, m, _, _ := setup(t, "ch-1")
		ctx := context.Background()
		m.db.EXPECT().CallGetFromDB(ctx, id).Return(nil, fmt.Errorf("err"))
		m.channel.EXPECT().HangingUp(ctx, "ch-1", cause).Return(nil, fmt.Errorf("err"))
		if _, err := h.hangingUpWithCause(ctx, id, cause); err == nil {
			t.Errorf("expected error")
		}
	})

	t.Run("channel hangup error, moved channel", func(t *testing.T) {
		h, m, _, upd := setup(t, "ch-2")
		ctx := context.Background()
		cur := covBCall(covBCallID, call.StatusTerminating)
		cur.ChannelID = "ch-2"
		m.db.EXPECT().CallGetFromDB(ctx, id).Return(cur, nil)
		m.channel.EXPECT().HangingUp(ctx, "ch-2", cause).Return(nil, fmt.Errorf("err"))
		res, err := h.hangingUpWithCause(ctx, id, cause)
		if err != nil || res != upd {
			t.Errorf("unexpected result: %v, %v", res, err)
		}
	})

	t.Run("channel already ended, hangup fails", func(t *testing.T) {
		h, m, _, upd := setup(t, "ch-1")
		ctx := context.Background()
		cn := &channel.Channel{ID: "ch-1"}
		now := time.Now()
		cn.TMEnd = &now
		m.db.EXPECT().CallGetFromDB(ctx, id).Return(nil, fmt.Errorf("err"))
		m.channel.EXPECT().HangingUp(ctx, "ch-1", cause).Return(cn, nil)
		m.db.EXPECT().CallGetByChannelID(ctx, "ch-1").Return(nil, fmt.Errorf("err"))
		res, err := h.hangingUpWithCause(ctx, id, cause)
		if err != nil || res != upd {
			t.Errorf("unexpected result: %v, %v", res, err)
		}
	})
}

func Test_hangingupWithReference_covB(t *testing.T) {
	id := uuid.FromStringOrNil(covBCallID)
	refID := uuid.FromStringOrNil("66666666-0000-0000-0000-000000000001")

	t.Run("reference get error", func(t *testing.T) {
		h, m := newCovBHandler(t)
		ctx := context.Background()
		m.db.EXPECT().CallGet(ctx, refID).Return(nil, fmt.Errorf("err"))
		if _, err := h.hangingupWithReference(ctx, covBCall(covBCallID, call.StatusProgressing), refID); err == nil {
			t.Errorf("expected error")
		}
	})

	t.Run("reference not hung up", func(t *testing.T) {
		h, m := newCovBHandler(t)
		ctx := context.Background()
		m.db.EXPECT().CallGet(ctx, refID).Return(covBCall(covBCallID, call.StatusProgressing), nil)
		if _, err := h.hangingupWithReference(ctx, covBCall(covBCallID, call.StatusProgressing), refID); err == nil {
			t.Errorf("expected error")
		}
	})

	t.Run("channel get error", func(t *testing.T) {
		h, m := newCovBHandler(t)
		ctx := context.Background()
		ref := covBCall(covBCallID, call.StatusHangup)
		ref.ChannelID = "ch-ref"
		m.db.EXPECT().CallGet(ctx, refID).Return(ref, nil)
		m.channel.EXPECT().Get(ctx, "ch-ref").Return(nil, fmt.Errorf("err"))
		if _, err := h.hangingupWithReference(ctx, covBCall(covBCallID, call.StatusProgressing), refID); err == nil {
			t.Errorf("expected error")
		}
	})

	t.Run("hangup error", func(t *testing.T) {
		h, m := newCovBHandler(t)
		ctx := context.Background()
		ref := covBCall(covBCallID, call.StatusHangup)
		ref.ChannelID = "ch-ref"
		m.db.EXPECT().CallGet(ctx, refID).Return(ref, nil)
		m.channel.EXPECT().Get(ctx, "ch-ref").Return(&channel.Channel{ID: "ch-ref", HangupCause: ari.ChannelCauseUserBusy}, nil)
		m.db.EXPECT().CallGet(ctx, id).Return(nil, fmt.Errorf("err"))
		c := covBCall(covBCallID, call.StatusProgressing)
		if _, err := h.hangingupWithReference(ctx, c, refID); err == nil {
			t.Errorf("expected error")
		}
	})
}
