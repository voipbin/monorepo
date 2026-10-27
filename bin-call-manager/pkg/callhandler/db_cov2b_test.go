package callhandler

import (
	"context"
	"fmt"
	"testing"
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
	fmaction "monorepo/bin-flow-manager/models/action"

	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/pkg/bridgehandler"
	"monorepo/bin-call-manager/pkg/channelhandler"
	"monorepo/bin-call-manager/pkg/dbhandler"
	"monorepo/bin-call-manager/pkg/externalmediahandler"
	"monorepo/bin-call-manager/pkg/groupcallhandler"
	"monorepo/bin-call-manager/pkg/outboundconfighandler"
)

// cov2bMocks bundles the mocks used by the cov2b tests of this package.
type cov2bMocks struct {
	util      *utilhandler.MockUtilHandler
	req       *requesthandler.MockRequestHandler
	db        *dbhandler.MockDBHandler
	notify    *notifyhandler.MockNotifyHandler
	channel   *channelhandler.MockChannelHandler
	bridge    *bridgehandler.MockBridgeHandler
	outbound  *outboundconfighandler.MockOutboundConfigHandler
	external  *externalmediahandler.MockExternalMediaHandler
	recovery  *MockRecoveryHandler
	groupcall *groupcallhandler.MockGroupcallHandler
}

func newCov2bHandler(t *testing.T) (*callHandler, *cov2bMocks) {
	mc := gomock.NewController(t)
	t.Cleanup(mc.Finish)

	m := &cov2bMocks{
		util:      utilhandler.NewMockUtilHandler(mc),
		req:       requesthandler.NewMockRequestHandler(mc),
		db:        dbhandler.NewMockDBHandler(mc),
		notify:    notifyhandler.NewMockNotifyHandler(mc),
		channel:   channelhandler.NewMockChannelHandler(mc),
		bridge:    bridgehandler.NewMockBridgeHandler(mc),
		outbound:  outboundconfighandler.NewMockOutboundConfigHandler(mc),
		recovery:  NewMockRecoveryHandler(mc),
		external:  externalmediahandler.NewMockExternalMediaHandler(mc),
		groupcall: groupcallhandler.NewMockGroupcallHandler(mc),
	}
	h := &callHandler{
		utilHandler:           m.util,
		reqHandler:            m.req,
		db:                    m.db,
		notifyHandler:         m.notify,
		channelHandler:        m.channel,
		bridgeHandler:         m.bridge,
		outboundConfigHandler: m.outbound,
		recoveryHandler:       m.recovery,
		externalMediaHandler:  m.external,
		groupcallHandler:      m.groupcall,
	}
	return h, m
}

func Test_Create_cov2b(t *testing.T) {
	callID := uuid.FromStringOrNil("6a1b2c3d-0000-4000-8000-00000000c2b1")
	customerID := uuid.FromStringOrNil("6a1b2c3d-0000-4000-8000-00000000c2b2")

	tests := []struct {
		name string

		getErr    error
		healthErr error

		expectErr bool
	}{
		{name: "get created call error", getErr: fmt.Errorf("db failure"), expectErr: true},
		{name: "health watcher failure does not fail the create", healthErr: fmt.Errorf("rpc failure")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCov2bHandler(t)
			ctx := context.Background()

			created := &call.Call{Identity: commonidentity.Identity{ID: callID, CustomerID: customerID}}

			m.db.EXPECT().CallCreate(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, c *call.Call) error {
				if c.ID != callID || c.CustomerID != customerID {
					t.Errorf("Wrong call to create. id: %s, customer: %s", c.ID, c.CustomerID)
				}
				return nil
			})
			if tt.getErr != nil {
				m.db.EXPECT().CallGet(ctx, callID).Return(nil, tt.getErr)
			} else {
				m.db.EXPECT().CallGet(ctx, callID).Return(created, nil)
				m.notify.EXPECT().PublishWebhookEvent(ctx, customerID, call.EventTypeCallCreated, created)
				m.req.EXPECT().CallV1CallHealth(ctx, callID, defaultHealthDelay, 0).Return(tt.healthErr)
			}

			src := &commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551230001"}
			dst := &commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551230002"}
			res, err := h.Create(ctx, callID, customerID, "", uuid.Nil, "", "", uuid.Nil, uuid.Nil, uuid.Nil,
				call.TypeFlow, uuid.Nil, src, dst, call.StatusDialing, nil, fmaction.Action{}, call.DirectionOutgoing, uuid.Nil, nil, nil)
			if (err != nil) != tt.expectErr {
				t.Fatalf("Wrong error. expect error: %v, got: %v", tt.expectErr, err)
			}
			if !tt.expectErr && res != created {
				t.Errorf("Wrong match. expect: %v, got: %v", created, res)
			}
		})
	}
}

func Test_UpdateHangupInfo_cov2b_tmHangupBranches(t *testing.T) {
	callID := uuid.FromStringOrNil("6a1b2c3d-0000-4000-8000-00000000c2b3")
	created := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	hangup := created.Add(30 * time.Second)

	tests := []struct {
		name     string
		tmHangup *time.Time
	}{
		{name: "hangup time set", tmHangup: &hangup},
		{name: "hangup time missing falls back to now"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCov2bHandler(t)
			ctx := context.Background()

			res := &call.Call{
				Identity:  commonidentity.Identity{ID: callID, CustomerID: uuid.FromStringOrNil("6a1b2c3d-0000-4000-8000-00000000c2b4")},
				Direction: call.DirectionOutgoing,
				Type:      call.TypeFlow,
				TMCreate:  &created,
				TMHangup:  tt.tmHangup,
			}
			m.db.EXPECT().CallSetHangup(ctx, callID, call.HangupReasonNormal, call.HangupByLocal).Return(nil)
			m.db.EXPECT().CallGet(ctx, callID).Return(res, nil)
			m.notify.EXPECT().PublishWebhookEvent(ctx, res.CustomerID, call.EventTypeCallHangup, res)

			got, written, err := h.UpdateHangupInfo(ctx, callID, "", call.HangupReasonNormal, call.HangupByLocal)
			if err != nil {
				t.Fatalf("Wrong error. got: %v", err)
			}
			if !written || got != res {
				t.Errorf("Wrong match. written: %v, got: %v", written, got)
			}
		})
	}
}
