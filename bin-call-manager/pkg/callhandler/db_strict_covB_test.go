package callhandler

import (
	"bytes"
	"context"
	stderrors "errors"
	"fmt"
	"testing"

	cerrors "monorepo/bin-common-handler/models/errors"
	fmaction "monorepo/bin-flow-manager/models/action"
	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-call-manager/pkg/dbhandler"
)

func Test_Get_errorKind_covB(t *testing.T) {
	id := uuid.FromStringOrNil(covBCallID)

	t.Run("not found is a typed not found error", func(t *testing.T) {
		h, m := newCovBHandler(t)
		ctx := context.Background()
		m.db.EXPECT().CallGet(ctx, id).Return(nil, dbhandler.ErrNotFound)

		_, err := h.Get(ctx, id)
		var ve *cerrors.VoipbinError
		if !stderrors.As(err, &ve) || ve.Status != cerrors.StatusNotFound {
			t.Errorf("Wrong match. expect: typed not found error, got: %v", err)
		}
		if !stderrors.Is(err, dbhandler.ErrNotFound) {
			t.Errorf("Wrong match. expect: the cause is kept, got: %v", err)
		}
	})

	t.Run("other error is not a typed error", func(t *testing.T) {
		h, m := newCovBHandler(t)
		ctx := context.Background()
		m.db.EXPECT().CallGet(ctx, id).Return(nil, fmt.Errorf("db failure"))

		_, err := h.Get(ctx, id)
		var ve *cerrors.VoipbinError
		if err == nil || stderrors.As(err, &ve) {
			t.Errorf("Wrong match. expect: plain error, got: %v", err)
		}
	})
}

func Test_dbDelete_success_covB(t *testing.T) {
	h, m := newCovBHandler(t)
	ctx := context.Background()
	id := uuid.FromStringOrNil(covBCallID)
	c := covBCall(covBCallID, call.StatusHangup)

	m.db.EXPECT().CallDelete(ctx, id).Return(nil)
	m.db.EXPECT().CallGet(ctx, id).Return(c, nil)
	m.notify.EXPECT().PublishWebhookEvent(ctx, c.CustomerID, call.EventTypeCallDeleted, c)

	res, err := h.dbDelete(ctx, id)
	if err != nil || res != c {
		t.Errorf("Wrong match. res: %v, err: %v", res, err)
	}
}

func Test_updateForRouteFailover_success_covB(t *testing.T) {
	h, m := newCovBHandler(t)
	ctx := context.Background()
	id := uuid.FromStringOrNil(covBCallID)
	dialrouteID := uuid.FromStringOrNil("44444444-0000-0000-0000-000000000001")
	c := covBCall(covBCallID, call.StatusDialing)

	m.db.EXPECT().CallSetForRouteFailover(ctx, id, "ch", dialrouteID).Return(nil)
	m.db.EXPECT().CallGet(ctx, id).Return(c, nil)
	m.notify.EXPECT().PublishWebhookEvent(ctx, c.CustomerID, call.EventTypeCallUpdated, c)

	res, err := h.updateForRouteFailover(ctx, id, "ch", dialrouteID)
	if err != nil || res != c {
		t.Errorf("Wrong match. res: %v, err: %v", res, err)
	}
}

func Test_updateActionAndActionNextHold_covB(t *testing.T) {
	id := uuid.FromStringOrNil(covBCallID)
	a := &fmaction.Action{ID: uuid.FromStringOrNil("44444444-0000-0000-0000-000000000002"), Type: fmaction.TypeAnswer}

	tests := []struct {
		name      string
		setErr    error
		getErr    error
		expectErr bool
	}{
		{"success", nil, nil, false},
		{"set error", fmt.Errorf("err"), nil, true},
		{"get error", nil, fmt.Errorf("err"), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCovBHandler(t)
			ctx := context.Background()
			c := covBCall(covBCallID, call.StatusProgressing)

			m.db.EXPECT().CallSetActionAndActionNextHold(ctx, id, a, false).Return(tt.setErr)
			if tt.setErr == nil {
				if tt.getErr != nil {
					m.db.EXPECT().CallGet(ctx, id).Return(nil, tt.getErr)
				} else {
					m.db.EXPECT().CallGet(ctx, id).Return(c, nil)
					m.notify.EXPECT().PublishWebhookEvent(ctx, c.CustomerID, call.EventTypeCallUpdated, c)
				}
			}

			res, err := h.updateActionAndActionNextHold(ctx, id, a)
			if (err != nil) != tt.expectErr {
				t.Fatalf("Wrong match. err: %v", err)
			}
			if !tt.expectErr && res != c {
				t.Errorf("Wrong match. res: %v", res)
			}
		})
	}
}

func Test_UpdateMuteDirection_covB(t *testing.T) {
	id := uuid.FromStringOrNil(covBCallID)

	tests := []struct {
		name      string
		setErr    error
		getErr    error
		expectErr bool
	}{
		{"success", nil, nil, false},
		{"set error", fmt.Errorf("err"), nil, true},
		{"get error", nil, fmt.Errorf("err"), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCovBHandler(t)
			ctx := context.Background()
			c := covBCall(covBCallID, call.StatusProgressing)

			m.db.EXPECT().CallSetMuteDirection(ctx, id, call.MuteDirectionBoth).Return(tt.setErr)
			if tt.setErr == nil {
				if tt.getErr != nil {
					m.db.EXPECT().CallGet(ctx, id).Return(nil, tt.getErr)
				} else {
					m.db.EXPECT().CallGet(ctx, id).Return(c, nil)
					m.notify.EXPECT().PublishWebhookEvent(ctx, c.CustomerID, call.EventTypeCallUpdated, c)
				}
			}

			res, err := h.UpdateMuteDirection(ctx, id, call.MuteDirectionBoth)
			if (err != nil) != tt.expectErr {
				t.Fatalf("Wrong match. err: %v", err)
			}
			if !tt.expectErr && res != c {
				t.Errorf("Wrong match. res: %v", res)
			}
		})
	}
}

func Test_digitsReceived_digitsReceiveVariable_covB(t *testing.T) {
	h, m := newCovBHandler(t)
	ctx := context.Background()

	// the call id and the activeflow id differ, so the variable must be set on the activeflow.
	c := covBCall(covBCallID, call.StatusProgressing)
	c.ActiveflowID = uuid.FromStringOrNil("88888888-0000-0000-0000-000000000002")
	c.Action = fmaction.Action{Type: fmaction.TypeDigitsReceive}
	cn := &channel.Channel{ID: "ch-1"}

	m.db.EXPECT().CallGetByChannelID(ctx, cn.ID).Return(c, nil)
	m.util.EXPECT().UUIDCreate().Return(uuid.Nil)
	m.util.EXPECT().TimeNow().Return(nil)
	m.notify.EXPECT().PublishEvent(ctx, gomock.Any(), gomock.Any())
	m.req.EXPECT().FlowV1VariableSetVariable(ctx, c.ActiveflowID, map[string]string{
		variableCallDigits: fmt.Sprintf("${%s}%s", variableCallDigits, "5"),
	}).Return(fmt.Errorf("err"))

	if err := h.digitsReceived(ctx, cn, "5", 100); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
}

func Test_EventFMActiveflowUpdated_hangupCause_covB(t *testing.T) {
	h, m := newCovBHandler(t)
	ctx := context.Background()
	id := uuid.FromStringOrNil(covBCallID)

	c := covBCall(covBCallID, call.StatusProgressing)
	c.Direction = call.DirectionIncoming
	c.ChannelID = "ch-1"
	upd := covBCall(covBCallID, call.StatusTerminating)
	upd.Direction = call.DirectionIncoming

	af := &fmactiveflow.Activeflow{Status: fmactiveflow.StatusEnded, ReferenceType: fmactiveflow.ReferenceTypeCall, ReferenceID: id}
	m.db.EXPECT().CallGet(ctx, id).Return(c, nil)
	m.db.EXPECT().CallSetStatus(ctx, id, call.StatusTerminating).Return(nil)
	m.db.EXPECT().CallGet(ctx, id).Return(upd, nil)
	m.notify.EXPECT().PublishWebhookEvent(ctx, upd.CustomerID, call.EventTypeCallTerminating, upd)
	m.db.EXPECT().CallGetFromDB(ctx, id).Return(c, nil)
	m.channel.EXPECT().HangingUp(ctx, "ch-1", ari.ChannelCauseNormalClearing).Return(&channel.Channel{ID: "ch-1"}, nil)

	if err := h.EventFMActiveflowUpdated(ctx, af); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
}

func Test_hangingUpWithCause_outgoingCancel_covB(t *testing.T) {
	id := uuid.FromStringOrNil(covBCallID)
	cause := ari.ChannelCauseNormalClearing

	tests := []struct {
		name   string
		status call.Status
	}{
		{"dialing", call.StatusDialing},
		{"ringing", call.StatusRinging},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCovBHandler(t)
			ctx := context.Background()

			c := covBCall(covBCallID, tt.status)
			c.Direction = call.DirectionOutgoing
			c.ChannelID = "ch-1"
			upd := covBCall(covBCallID, call.StatusCanceling)
			upd.Direction = call.DirectionOutgoing

			m.db.EXPECT().CallGet(ctx, id).Return(c, nil)
			m.db.EXPECT().CallSetStatus(ctx, id, call.StatusCanceling).Return(nil)
			m.db.EXPECT().CallGet(ctx, id).Return(upd, nil)
			m.notify.EXPECT().PublishWebhookEvent(ctx, upd.CustomerID, call.EventTypeCallCanceling, upd)
			m.db.EXPECT().CallGetFromDB(ctx, id).Return(c, nil)
			m.channel.EXPECT().HangingUp(ctx, "ch-1", cause).Return(&channel.Channel{ID: "ch-1"}, nil)

			res, err := h.hangingUpWithCause(ctx, id, cause)
			if err != nil || res != upd {
				t.Errorf("Wrong match. res: %v, err: %v", res, err)
			}
		})
	}
}

func Test_ARIChannelDestroyed_application_covB(t *testing.T) {
	var buf bytes.Buffer
	prev := logrus.StandardLogger().Out
	logrus.SetOutput(&buf)
	defer logrus.SetOutput(prev)

	h, _ := newCovBHandler(t)
	cn := &channel.Channel{ID: "ch-1", Type: channel.TypeApplication}
	if err := h.ARIChannelDestroyed(context.Background(), cn); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
	if bytes.Contains(buf.Bytes(), []byte("Unsupported channel type")) {
		t.Errorf("Wrong match. the application type must be a supported type. log: %s", buf.String())
	}
}
