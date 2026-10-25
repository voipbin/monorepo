package callhandler

import (
	"context"
	"fmt"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/pkg/bridgehandler"
	"monorepo/bin-call-manager/pkg/channelhandler"
	"monorepo/bin-call-manager/pkg/dbhandler"
)

// covBMocks bundles the mocks used by the covB tests of this package.
type covBMocks struct {
	util    *utilhandler.MockUtilHandler
	req     *requesthandler.MockRequestHandler
	db      *dbhandler.MockDBHandler
	notify  *notifyhandler.MockNotifyHandler
	channel *channelhandler.MockChannelHandler
	bridge  *bridgehandler.MockBridgeHandler
}

func newCovBHandler(t *testing.T) (*callHandler, *covBMocks) {
	mc := gomock.NewController(t)
	t.Cleanup(mc.Finish)

	m := &covBMocks{
		util:    utilhandler.NewMockUtilHandler(mc),
		req:     requesthandler.NewMockRequestHandler(mc),
		db:      dbhandler.NewMockDBHandler(mc),
		notify:  notifyhandler.NewMockNotifyHandler(mc),
		channel: channelhandler.NewMockChannelHandler(mc),
		bridge:  bridgehandler.NewMockBridgeHandler(mc),
	}
	h := &callHandler{
		utilHandler:    m.util,
		reqHandler:     m.req,
		db:             m.db,
		notifyHandler:  m.notify,
		channelHandler: m.channel,
		bridgeHandler:  m.bridge,
	}
	return h, m
}

func covBCall(id string, status call.Status) *call.Call {
	return &call.Call{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil(id),
			CustomerID: uuid.FromStringOrNil("11111111-0000-0000-0000-000000000001"),
		},
		Status: status,
	}
}

const covBCallID = "22222222-0000-0000-0000-000000000001"

func Test_List_error_covB(t *testing.T) {
	h, m := newCovBHandler(t)
	ctx := context.Background()
	m.db.EXPECT().CallList(ctx, uint64(10), "", gomock.Any()).Return(nil, fmt.Errorf("err"))
	if _, err := h.List(ctx, 10, "", nil); err == nil {
		t.Errorf("expected error")
	}
}

func Test_GetByChannelID_error_covB(t *testing.T) {
	h, m := newCovBHandler(t)
	ctx := context.Background()
	m.db.EXPECT().CallGetByChannelID(ctx, "ch").Return(nil, fmt.Errorf("err"))
	if _, err := h.GetByChannelID(ctx, "ch"); err == nil {
		t.Errorf("expected error")
	}
}

func Test_Get_notFound_covB(t *testing.T) {
	h, m := newCovBHandler(t)
	ctx := context.Background()
	id := uuid.FromStringOrNil(covBCallID)
	m.db.EXPECT().CallGet(ctx, id).Return(nil, dbhandler.ErrNotFound)
	if _, err := h.Get(ctx, id); err == nil {
		t.Errorf("expected error")
	}
}

func Test_updateForRouteFailover_error_covB(t *testing.T) {
	id := uuid.FromStringOrNil(covBCallID)
	tests := []struct {
		name   string
		setErr error
		getErr error
	}{
		{"set error", fmt.Errorf("err"), nil},
		{"get error", nil, fmt.Errorf("err")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCovBHandler(t)
			ctx := context.Background()
			m.db.EXPECT().CallSetForRouteFailover(ctx, id, "ch", uuid.Nil).Return(tt.setErr)
			if tt.setErr == nil {
				m.db.EXPECT().CallGet(ctx, id).Return(nil, tt.getErr)
			}
			if _, err := h.updateForRouteFailover(ctx, id, "ch", uuid.Nil); err == nil {
				t.Errorf("expected error")
			}
		})
	}
}

func Test_UpdateStatus_covB(t *testing.T) {
	id := uuid.FromStringOrNil(covBCallID)
	tests := []struct {
		name      string
		status    call.Status
		setErr    error
		getErr    error
		expectErr bool
	}{
		{"ringing set error", call.StatusRinging, fmt.Errorf("err"), nil, true},
		{"progressing set error", call.StatusProgressing, fmt.Errorf("err"), nil, true},
		{"default set error", call.StatusTerminating, fmt.Errorf("err"), nil, true},
		{"get error", call.StatusTerminating, nil, fmt.Errorf("err"), true},
		{"hangup status has no event", call.StatusHangup, nil, nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCovBHandler(t)
			ctx := context.Background()

			switch tt.status {
			case call.StatusRinging:
				m.db.EXPECT().CallSetStatusRinging(ctx, id).Return(tt.setErr)
			case call.StatusProgressing:
				m.db.EXPECT().CallSetStatusProgressing(ctx, id).Return(tt.setErr)
			default:
				m.db.EXPECT().CallSetStatus(ctx, id, tt.status).Return(tt.setErr)
			}
			if tt.setErr == nil {
				if tt.getErr != nil {
					m.db.EXPECT().CallGet(ctx, id).Return(nil, tt.getErr)
				} else {
					m.db.EXPECT().CallGet(ctx, id).Return(covBCall(covBCallID, tt.status), nil)
				}
			}

			res, err := h.UpdateStatus(ctx, id, tt.status)
			if (err != nil) != tt.expectErr {
				t.Errorf("unexpected err: %v", err)
			}
			if !tt.expectErr && res == nil {
				t.Errorf("expected result")
			}
		})
	}
}

func Test_dbDelete_error_covB(t *testing.T) {
	id := uuid.FromStringOrNil(covBCallID)
	tests := []struct {
		name   string
		delErr error
		getErr error
	}{
		{"delete error", fmt.Errorf("err"), nil},
		{"get error", nil, fmt.Errorf("err")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCovBHandler(t)
			ctx := context.Background()
			m.db.EXPECT().CallDelete(ctx, id).Return(tt.delErr)
			if tt.delErr == nil {
				m.db.EXPECT().CallGet(ctx, id).Return(nil, tt.getErr)
			}
			if _, err := h.dbDelete(ctx, id); err == nil {
				t.Errorf("expected error")
			}
		})
	}
}

func Test_UpdateRecordingID_error_covB(t *testing.T) {
	id := uuid.FromStringOrNil(covBCallID)
	recID := uuid.FromStringOrNil("33333333-0000-0000-0000-000000000001")
	tests := []struct {
		name   string
		setErr error
		addErr error
		getErr error
	}{
		{"set error", fmt.Errorf("err"), nil, nil},
		{"add error", nil, fmt.Errorf("err"), nil},
		{"get error", nil, nil, fmt.Errorf("err")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCovBHandler(t)
			ctx := context.Background()
			m.db.EXPECT().CallSetRecordingID(ctx, id, recID).Return(tt.setErr)
			if tt.setErr == nil {
				m.db.EXPECT().CallAddRecordingIDs(ctx, id, recID).Return(tt.addErr)
				if tt.addErr == nil {
					m.db.EXPECT().CallGet(ctx, id).Return(nil, tt.getErr)
				}
			}
			if _, err := h.UpdateRecordingID(ctx, id, recID); err == nil {
				t.Errorf("expected error")
			}
		})
	}
}

func Test_ExternalMediaID_error_covB(t *testing.T) {
	id := uuid.FromStringOrNil(covBCallID)
	emID := uuid.FromStringOrNil("44444444-0000-0000-0000-000000000001")
	for _, add := range []bool{true, false} {
		for _, step := range []string{"op", "get"} {
			t.Run(fmt.Sprintf("add=%v step=%s", add, step), func(t *testing.T) {
				h, m := newCovBHandler(t)
				ctx := context.Background()

				opErr := error(nil)
				if step == "op" {
					opErr = fmt.Errorf("err")
				}
				if add {
					m.db.EXPECT().CallAddExternalMediaID(ctx, id, emID).Return(opErr)
				} else {
					m.db.EXPECT().CallRemoveExternalMediaID(ctx, id, emID).Return(opErr)
				}
				if step == "get" {
					m.db.EXPECT().CallGet(ctx, id).Return(nil, fmt.Errorf("err"))
				}

				var err error
				if add {
					_, err = h.AddExternalMediaID(ctx, id, emID)
				} else {
					_, err = h.RemoveExternalMediaID(ctx, id, emID)
				}
				if err == nil {
					t.Errorf("expected error")
				}
			})
		}
	}
}

func Test_UpdateConfbridgeID_covB(t *testing.T) {
	id := uuid.FromStringOrNil(covBCallID)
	cbID := uuid.FromStringOrNil("55555555-0000-0000-0000-000000000001")
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
			m.db.EXPECT().CallSetConfbridgeID(ctx, id, cbID).Return(tt.setErr)
			if tt.setErr == nil {
				if tt.getErr != nil {
					m.db.EXPECT().CallGet(ctx, id).Return(nil, tt.getErr)
				} else {
					m.db.EXPECT().CallGet(ctx, id).Return(c, nil)
					m.notify.EXPECT().PublishWebhookEvent(ctx, c.CustomerID, call.EventTypeCallUpdated, c)
				}
			}

			res, err := h.UpdateConfbridgeID(ctx, id, cbID)
			if (err != nil) != tt.expectErr {
				t.Fatalf("unexpected err: %v", err)
			}
			if !tt.expectErr && res != c {
				t.Errorf("wrong result")
			}
		})
	}
}

func Test_UpdateData_covB(t *testing.T) {
	id := uuid.FromStringOrNil(covBCallID)
	data := map[call.DataType]string{call.DataTypeEarlyExecution: "true"}
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
			m.db.EXPECT().CallSetData(ctx, id, data).Return(tt.setErr)
			if tt.setErr == nil {
				if tt.getErr != nil {
					m.db.EXPECT().CallGet(ctx, id).Return(nil, tt.getErr)
				} else {
					m.db.EXPECT().CallGet(ctx, id).Return(c, nil)
				}
			}

			res, err := h.UpdateData(ctx, id, data)
			if (err != nil) != tt.expectErr {
				t.Fatalf("unexpected err: %v", err)
			}
			if !tt.expectErr && res != c {
				t.Errorf("wrong result")
			}
		})
	}
}

func Test_UpdateHangupInfo_error_covB(t *testing.T) {
	id := uuid.FromStringOrNil(covBCallID)
	tests := []struct {
		name         string
		owner        string
		setErr       error
		written      bool
		getErr       error
		expectErr    bool
		expectWriten bool
	}{
		{"unconditional set error", "", fmt.Errorf("err"), false, nil, true, false},
		{"owner set error", "ch", fmt.Errorf("err"), false, nil, true, false},
		{"owner not written", "ch", nil, false, nil, false, false},
		{"get error", "", nil, true, fmt.Errorf("err"), true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCovBHandler(t)
			ctx := context.Background()

			if tt.owner == "" {
				m.db.EXPECT().CallSetHangup(ctx, id, call.HangupReasonNormal, call.HangupByLocal).Return(tt.setErr)
			} else {
				m.db.EXPECT().CallSetHangupIfChannel(ctx, id, tt.owner, call.HangupReasonNormal, call.HangupByLocal).Return(tt.written, tt.setErr)
			}
			if tt.setErr == nil && tt.written {
				m.db.EXPECT().CallGet(ctx, id).Return(nil, tt.getErr)
			}

			res, written, err := h.UpdateHangupInfo(ctx, id, tt.owner, call.HangupReasonNormal, call.HangupByLocal)
			if (err != nil) != tt.expectErr {
				t.Errorf("unexpected err: %v", err)
			}
			if written != tt.expectWriten || res != nil {
				t.Errorf("unexpected result. res: %v, written: %v", res, written)
			}
		})
	}
}
