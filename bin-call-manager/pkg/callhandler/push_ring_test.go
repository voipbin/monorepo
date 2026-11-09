package callhandler

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	cucustomer "monorepo/bin-customer-manager/models/customer"
	fmaction "monorepo/bin-flow-manager/models/action"
	fmflow "monorepo/bin-flow-manager/models/flow"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-call-manager/models/pushring"
	"monorepo/bin-call-manager/pkg/pushringhandler"
)

var (
	prCustomerID  = uuid.FromStringOrNil("7a000000-0000-4000-8000-000000000001")
	prSessionID   = uuid.FromStringOrNil("7a000000-0000-4000-8000-000000000002")
	prConfbridge  = uuid.FromStringOrNil("7a000000-0000-4000-8000-000000000003")
	prExtensionID = uuid.FromStringOrNil("7a000000-0000-4000-8000-000000000004")
	prFlowID      = uuid.FromStringOrNil("7a000000-0000-4000-8000-000000000005")
	prCallID      = uuid.FromStringOrNil("7a000000-0000-4000-8000-000000000006")
	prMasterFlow  = uuid.FromStringOrNil("7a000000-0000-4000-8000-000000000007")

	// the value of a ticket as it arrives in the SIP request: the direct prefix and 32 hex characters.
	prTicket = "direct.0123456789abcdef0123456789abcdef"
)

func newPushRingHandler(t *testing.T) (*callHandler, *cov2bMocks, *pushringhandler.MockPushRingHandler) {
	h, m := newCov2bHandler(t)

	mc := gomock.NewController(t)
	t.Cleanup(mc.Finish)
	mockPush := pushringhandler.NewMockPushRingHandler(mc)
	h.pushRingHandler = mockPush

	return h, m, mockPush
}

func Test_startIncomingDomainTypeSIPDirect_pushTicket(t *testing.T) {
	ctx := context.Background()
	cn := &channel.Channel{ID: "ch-push-ticket"}
	source := &commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15550100"}
	claim := &pushring.TicketClaim{SessionID: prSessionID, DeviceID: uuid.Must(uuid.NewV4()), TicketHash: "hash"}
	session := &pushring.Session{ID: prSessionID, CustomerID: prCustomerID, ConfbridgeID: prConfbridge, ExtensionID: prExtensionID}

	expectedActions := []fmaction.Action{
		{
			Type:   fmaction.TypeConfbridgeJoin,
			Option: fmaction.ConvertOption(fmaction.OptionConfbridgeJoin{ConfbridgeID: prConfbridge}),
		},
		{Type: fmaction.TypeHangup},
	}

	t.Run("a ticket starts a call that runs a flow joining the confbridge, bound before the call exists", func(t *testing.T) {
		h, m, mockPush := newPushRingHandler(t)

		m.channel.EXPECT().AddressGetSource(cn, commonaddress.TypeTel).Return(source)
		mockPush.EXPECT().TicketClaim(ctx, prTicket).Return(claim, nil)
		mockPush.EXPECT().TicketSessionGet(ctx, claim).Return(session, nil)
		m.req.EXPECT().FlowV1FlowCreate(ctx, prCustomerID, fmflow.TypeFlow, "tmp", gomock.Any(), expectedActions, uuid.Nil, false).
			Return(&fmflow.Flow{Identity: commonidentity.Identity{ID: prFlowID}}, nil)
		mockPush.EXPECT().TicketBind(ctx, claim, prFlowID).Return(nil)

		// the call is started with the temporary flow. A rejected customer ends it at once, which shows the start was reached.
		m.util.EXPECT().UUIDCreate().Return(prCallID)
		m.req.EXPECT().CustomerV1CustomerGet(ctx, prCustomerID).Return(&cucustomer.Customer{Status: cucustomer.StatusDeleted}, nil)
		m.channel.EXPECT().HangingUp(ctx, cn.ID, ari.ChannelCauseCallRejected).Return(&channel.Channel{}, nil)

		// the direct manager is never asked about a ticket
		if err := h.startIncomingDomainTypeSIPDirect(ctx, cn, prTicket); err != nil {
			t.Errorf("Wrong match. expect: nil, got: %v", err)
		}
	})

	t.Run("a refused ticket ends the call with the same cause as an unknown value", func(t *testing.T) {
		h, m, mockPush := newPushRingHandler(t)

		m.channel.EXPECT().AddressGetSource(cn, commonaddress.TypeTel).Return(source)
		mockPush.EXPECT().TicketClaim(ctx, prTicket).Return(nil, pushring.ErrTicketRefused)
		m.channel.EXPECT().HangingUp(ctx, cn.ID, ari.ChannelCauseNoRouteDestination).Return(&channel.Channel{}, nil)

		if err := h.startIncomingDomainTypeSIPDirect(ctx, cn, prTicket); err != nil {
			t.Errorf("Wrong match. expect: nil, got: %v", err)
		}
	})

	t.Run("a value that is no ticket continues with the direct lookup", func(t *testing.T) {
		h, m, mockPush := newPushRingHandler(t)

		m.channel.EXPECT().AddressGetSource(cn, commonaddress.TypeTel).Return(source)
		mockPush.EXPECT().TicketClaim(ctx, "direct.abc").Return(nil, nil)
		m.req.EXPECT().DirectV1DirectGetByHash(ctx, "direct.abc").Return(nil, errors.New("not found"))
		m.channel.EXPECT().HangingUp(ctx, cn.ID, ari.ChannelCauseNoRouteDestination).Return(&channel.Channel{}, nil)

		if err := h.startIncomingDomainTypeSIPDirect(ctx, cn, "direct.abc"); err != nil {
			t.Errorf("Wrong match. expect: nil, got: %v", err)
		}
	})

	t.Run("the failures before the call exists end the call", func(t *testing.T) {
		tests := []struct {
			name  string
			setup func(m *cov2bMocks, mockPush *pushringhandler.MockPushRingHandler)
			cause ari.ChannelCause
		}{
			{
				name: "the session is gone",
				setup: func(m *cov2bMocks, mockPush *pushringhandler.MockPushRingHandler) {
					mockPush.EXPECT().TicketSessionGet(ctx, claim).Return(nil, pushring.ErrSessionNotFound)
				},
				cause: ari.ChannelCauseNoRouteDestination,
			},
			{
				name: "the flow could not be created",
				setup: func(m *cov2bMocks, mockPush *pushringhandler.MockPushRingHandler) {
					mockPush.EXPECT().TicketSessionGet(ctx, claim).Return(session, nil)
					m.req.EXPECT().FlowV1FlowCreate(ctx, prCustomerID, fmflow.TypeFlow, "tmp", gomock.Any(), expectedActions, uuid.Nil, false).Return(nil, errors.New("flow down"))
				},
				cause: ari.ChannelCauseNetworkOutOfOrder,
			},
			{
				name: "the ticket could not be bound",
				setup: func(m *cov2bMocks, mockPush *pushringhandler.MockPushRingHandler) {
					mockPush.EXPECT().TicketSessionGet(ctx, claim).Return(session, nil)
					m.req.EXPECT().FlowV1FlowCreate(ctx, prCustomerID, fmflow.TypeFlow, "tmp", gomock.Any(), expectedActions, uuid.Nil, false).
						Return(&fmflow.Flow{Identity: commonidentity.Identity{ID: prFlowID}}, nil)
					mockPush.EXPECT().TicketBind(ctx, claim, prFlowID).Return(errors.New("cache down"))
				},
				cause: ari.ChannelCauseNetworkOutOfOrder,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				h, m, mockPush := newPushRingHandler(t)

				m.channel.EXPECT().AddressGetSource(cn, commonaddress.TypeTel).Return(source)
				mockPush.EXPECT().TicketClaim(ctx, prTicket).Return(claim, nil)
				tt.setup(m, mockPush)
				m.channel.EXPECT().HangingUp(ctx, cn.ID, tt.cause).Return(&channel.Channel{}, nil)

				if err := h.startIncomingDomainTypeSIPDirect(ctx, cn, prTicket); err != nil {
					t.Errorf("Wrong match. expect: nil, got: %v", err)
				}
			})
		}
	})
}

// The hangup of a call reaches the session handler at the confirmed hangup point, and only there.
func Test_Hangup_pushRingHook(t *testing.T) {
	ctx := context.Background()
	id := uuid.FromStringOrNil(covBCallID)
	cn := &channel.Channel{ID: "ch-hook"}

	t.Run("the call really ended", func(t *testing.T) {
		h, m, mockPush := newPushRingHandler(t)

		c := covBCall(covBCallID, call.StatusRinging)
		c.Direction = call.DirectionIncoming
		c.FlowID = prMasterFlow
		ended := covBCall(covBCallID, call.StatusHangup)
		ended.Direction = call.DirectionIncoming
		ended.FlowID = prMasterFlow

		m.db.EXPECT().CallGetByChannelID(ctx, cn.ID).Return(c, nil)
		m.bridge.EXPECT().Destroy(ctx, c.BridgeID).Return(nil)
		m.db.EXPECT().CallSetHangup(ctx, id, gomock.Any(), gomock.Any()).Return(nil)
		m.db.EXPECT().CallGet(ctx, id).Return(ended, nil)
		m.notify.EXPECT().PublishWebhookEvent(ctx, ended.CustomerID, call.EventTypeCallHangup, ended)
		// the hook gets the call as it was written
		mockPush.EXPECT().CallHangup(ctx, ended)
		m.req.EXPECT().FlowV1ActiveflowStop(ctx, c.ActiveflowID).Return(nil, nil)

		if _, err := h.Hangup(ctx, cn); err != nil {
			t.Errorf("Wrong match. expect: nil, got: %v", err)
		}
	})

	t.Run("the call only moved to another channel: the hook does not run", func(t *testing.T) {
		h, m, _ := newPushRingHandler(t)

		c := covBCall(covBCallID, call.StatusProgressing)
		c.Direction = call.DirectionIncoming
		cur := covBCall(covBCallID, call.StatusProgressing)
		cur.ChannelID = "ch-other"

		m.db.EXPECT().CallGetByChannelID(ctx, cn.ID).Return(c, nil)
		m.bridge.EXPECT().Destroy(ctx, c.BridgeID).Return(nil)
		m.db.EXPECT().CallSetHangupIfChannel(ctx, id, cn.ID, gomock.Any(), gomock.Any()).Return(false, nil)
		m.db.EXPECT().CallGetFromDB(ctx, id).Return(cur, nil)

		if _, err := h.Hangup(ctx, cn); err != nil {
			t.Errorf("Wrong match. expect: nil, got: %v", err)
		}
	})

	t.Run("the call already hung up: the hook does not run", func(t *testing.T) {
		h, m, _ := newPushRingHandler(t)

		now := time.Now()
		c := covBCall(covBCallID, call.StatusHangup)
		c.TMHangup = &now
		m.db.EXPECT().CallGetByChannelID(ctx, cn.ID).Return(c, nil)

		if _, err := h.Hangup(ctx, cn); err != nil {
			t.Errorf("Wrong match. expect: nil, got: %v", err)
		}
	})
}

// A call handler without a session handler (older tests build one so) must not panic: the hook is a nil check.
func Test_Hangup_pushRingHook_nilHandler(t *testing.T) {
	ctx := context.Background()
	h, m := newCovBHandler(t)
	cn := &channel.Channel{ID: "ch-hook"}

	c := covBCall(covBCallID, call.StatusRinging)
	c.Direction = call.DirectionIncoming
	ended := covBCall(covBCallID, call.StatusHangup)
	ended.Direction = call.DirectionIncoming

	m.db.EXPECT().CallGetByChannelID(ctx, cn.ID).Return(c, nil)
	m.bridge.EXPECT().Destroy(ctx, c.BridgeID).Return(nil)
	m.db.EXPECT().CallSetHangup(ctx, c.ID, gomock.Any(), gomock.Any()).Return(nil)
	m.db.EXPECT().CallGet(ctx, c.ID).Return(ended, nil)
	m.notify.EXPECT().PublishWebhookEvent(ctx, ended.CustomerID, call.EventTypeCallHangup, ended)
	m.req.EXPECT().FlowV1ActiveflowStop(ctx, c.ActiveflowID).Return(nil, nil)

	if _, err := h.Hangup(ctx, cn); err != nil {
		t.Errorf("Wrong match. expect: nil, got: %v", err)
	}
}
