package callhandler

import (
	"context"
	"reflect"
	"testing"
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"
	rmroute "monorepo/bin-route-manager/models/route"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/channel"
)

func Test_isRetryable_cov2a(t *testing.T) {
	routeA := rmroute.Route{ID: uuid.FromStringOrNil("44444444-0000-0000-0000-000000000060")}
	routeB := rmroute.Route{ID: uuid.FromStringOrNil("44444444-0000-0000-0000-000000000061")}

	tests := []struct {
		name   string
		modify func(c *call.Call)

		expect bool
	}{
		{
			name: "progressing call is not retryable",
			modify: func(c *call.Call) {
				c.Status = call.StatusProgressing
				c.Dialroutes = []rmroute.Route{routeA, routeB}
				c.DialrouteID = routeA.ID
			},
			expect: false,
		},
		{
			name: "no dialroute is left",
			modify: func(c *call.Call) {
				c.Status = call.StatusRinging
				c.Dialroutes = []rmroute.Route{routeA, routeB}
				c.DialrouteID = routeB.ID
			},
			expect: false,
		},
		{
			name: "next dialroute is available",
			modify: func(c *call.Call) {
				c.Status = call.StatusDialing
				c.Dialroutes = []rmroute.Route{routeA, routeB}
				c.DialrouteID = routeA.ID
			},
			expect: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, _ := cov2aNewHandler(t)

			c := cov2aNewCall(nil)
			c.Direction = call.DirectionOutgoing
			c.Destination = commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551230001"}
			tt.modify(c)

			// a cause which is not in the not-retryable list.
			cn := &channel.Channel{ID: c.ChannelID, HangupCause: ari.ChannelCauseUnallocated}

			if got := h.isRetryable(context.Background(), c, cn); got != tt.expect {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expect, got)
			}
		})
	}
}

func Test_Hangup_cov2a_failoverError(t *testing.T) {
	h, m := cov2aNewHandler(t)
	ctx := context.Background()

	routeA := rmroute.Route{ID: uuid.FromStringOrNil("44444444-0000-0000-0000-000000000060")}
	routeB := rmroute.Route{ID: uuid.FromStringOrNil("44444444-0000-0000-0000-000000000061")}
	channelID := uuid.FromStringOrNil("44444444-0000-0000-0000-000000000062")

	c := cov2aNewCall(nil)
	c.Status = call.StatusRinging
	c.Direction = call.DirectionOutgoing
	c.Destination = commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551230001"}
	c.Dialroutes = []rmroute.Route{routeA, routeB}
	c.DialrouteID = routeA.ID
	cn := &channel.Channel{ID: c.ChannelID, HangupCause: ari.ChannelCauseUnallocated}

	hungup := cov2aNewCall(nil)
	hungup.Status = call.StatusHangup

	m.db.EXPECT().CallGetByChannelID(ctx, cn.ID).Return(c, nil)
	m.bridge.EXPECT().Destroy(ctx, c.BridgeID).Return(nil)
	// the failover fails at the db update. the hangup process goes on.
	m.util.EXPECT().UUIDCreate().Return(channelID)
	m.db.EXPECT().CallSetForRouteFailover(ctx, c.ID, channelID.String(), routeB.ID).Return(errCov2aBoom)
	m.db.EXPECT().CallSetHangup(ctx, c.ID, gomock.Any(), gomock.Any()).Return(nil)
	m.db.EXPECT().CallGet(ctx, c.ID).Return(hungup, nil)
	m.notify.EXPECT().PublishWebhookEvent(ctx, hungup.CustomerID, call.EventTypeCallHangup, hungup)
	m.req.EXPECT().FlowV1ActiveflowStop(ctx, c.ActiveflowID).Return(nil, nil)

	got, err := h.Hangup(ctx, cn)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !reflect.DeepEqual(got, hungup) {
		t.Errorf("Wrong match. expect: %v, got: %v", hungup, got)
	}
}

func Test_Hangup_cov2a_sideEffectErrors(t *testing.T) {
	h, m := cov2aNewHandler(t)
	ctx := context.Background()

	chainedID := uuid.FromStringOrNil("44444444-0000-0000-0000-000000000070")
	groupcallID := uuid.FromStringOrNil("44444444-0000-0000-0000-000000000071")

	c := cov2aNewCall(nil)
	c.Status = call.StatusRinging
	c.Direction = call.DirectionIncoming
	cn := &channel.Channel{ID: c.ChannelID}

	hungup := cov2aNewCall(nil)
	hungup.Status = call.StatusHangup
	hungup.GroupcallID = groupcallID
	hungup.ChainedCallIDs = []uuid.UUID{chainedID}

	groupcallDone := make(chan struct{})

	m.db.EXPECT().CallGetByChannelID(ctx, cn.ID).Return(c, nil)
	m.bridge.EXPECT().Destroy(ctx, c.BridgeID).Return(errCov2aBoom)
	m.db.EXPECT().CallSetHangup(ctx, c.ID, gomock.Any(), gomock.Any()).Return(nil)
	m.db.EXPECT().CallGet(ctx, c.ID).Return(hungup, nil)
	m.notify.EXPECT().PublishWebhookEvent(ctx, hungup.CustomerID, call.EventTypeCallHangup, hungup)
	m.req.EXPECT().CallV1GroupcallHangupCall(gomock.Any(), groupcallID).DoAndReturn(
		func(_ context.Context, _ uuid.UUID) error {
			close(groupcallDone)
			return errCov2aBoom
		})
	m.req.EXPECT().FlowV1ActiveflowStop(ctx, c.ActiveflowID).Return(nil, errCov2aBoom)
	// the chained call hangup fails at the lookup. It must not fail the hangup of this call.
	m.db.EXPECT().CallGet(ctx, chainedID).Return(nil, errCov2aBoom)

	got, err := h.Hangup(ctx, cn)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !reflect.DeepEqual(got, hungup) {
		t.Errorf("Wrong match. expect: %v, got: %v", hungup, got)
	}

	select {
	case <-groupcallDone:
	case <-time.After(5 * time.Second):
		t.Errorf("The groupcall hangup request was not sent.")
	}
}

func Test_hangingUpWithCause_cov2a_channelAlreadyEnded(t *testing.T) {
	h, m := cov2aNewHandler(t)
	ctx := context.Background()

	cause := ari.ChannelCauseNormalClearing

	c := cov2aNewCall(nil)
	c.Direction = call.DirectionIncoming
	terminating := cov2aNewCall(nil)
	terminating.Status = call.StatusTerminating
	terminating.Direction = call.DirectionIncoming

	// the channel is already ended, so the call is hung up right away.
	ended := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	cn := &channel.Channel{ID: c.ChannelID, TMEnd: &ended}

	hungup := cov2aNewCall(nil)
	hungup.Status = call.StatusHangup
	hungup.Direction = call.DirectionIncoming

	gomock.InOrder(
		m.db.EXPECT().CallGet(ctx, c.ID).Return(c, nil),
		m.db.EXPECT().CallSetStatus(ctx, c.ID, call.StatusTerminating).Return(nil),
		m.db.EXPECT().CallGet(ctx, c.ID).Return(terminating, nil),
		m.notify.EXPECT().PublishWebhookEvent(ctx, terminating.CustomerID, call.EventTypeCallTerminating, terminating),
		m.db.EXPECT().CallGetFromDB(ctx, c.ID).Return(c, nil),
		m.channel.EXPECT().HangingUp(ctx, c.ChannelID, cause).Return(cn, nil),
		// Hangup() of the ended channel
		m.db.EXPECT().CallGetByChannelID(ctx, cn.ID).Return(c, nil),
		m.bridge.EXPECT().Destroy(ctx, c.BridgeID).Return(nil),
		m.db.EXPECT().CallSetHangupIfChannel(ctx, c.ID, cn.ID, gomock.Any(), gomock.Any()).Return(true, nil),
		m.db.EXPECT().CallGet(ctx, c.ID).Return(hungup, nil),
		m.notify.EXPECT().PublishWebhookEvent(ctx, hungup.CustomerID, call.EventTypeCallHangup, hungup),
		m.req.EXPECT().FlowV1ActiveflowStop(ctx, c.ActiveflowID).Return(nil, nil),
	)

	got, err := h.hangingUpWithCause(ctx, c.ID, cause)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !reflect.DeepEqual(got, hungup) {
		t.Errorf("Wrong match. expect: %v, got: %v", hungup, got)
	}
}
