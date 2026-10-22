package servicehandler

import (
	"context"
	"testing"
	"time"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/dbhandler"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	cmcall "monorepo/bin-call-manager/models/call"
	cmchannel "monorepo/bin-call-manager/models/channel"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	tmsipmessage "monorepo/bin-timeline-manager/models/sipmessage"

	"github.com/gofrs/uuid"
)

var h2SIPCallID = uuid.FromStringOrNil("a2000000-0000-0000-0000-000000001001")

// h2SIPTimes returns create/hangup times used by the SIP call fixture.
func h2SIPTimes() (create, hangup, update time.Time) {
	create = time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	hangup = time.Date(2026, 1, 1, 10, 5, 0, 0, time.UTC)
	update = time.Date(2026, 1, 1, 10, 6, 0, 0, time.UTC)
	return
}

func h2SIPCall(customerID uuid.UUID, channelID string, tmCreate, tmHangup, tmUpdate *time.Time) *cmcall.Call {
	return &cmcall.Call{
		Identity:  commonidentity.Identity{ID: h2SIPCallID, CustomerID: customerID},
		ChannelID: channelID,
		TMCreate:  tmCreate,
		TMHangup:  tmHangup,
		TMUpdate:  tmUpdate,
	}
}

// h2SIPCases builds the table shared by TimelineSIPAnalysisGet and TimelineSIPPcapGet.
// rpc registers the final timeline RPC for the given (from, to) strings; failure makes it fail.
func h2SIPCases(
	call func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) (any, error),
	rpc func(ctx context.Context, req *requesthandler.MockRequestHandler, from, to string, failure bool),
	wantRes any,
) []h2Case {
	admin := h2AgentIdentity(h2CustomerID, amagent.PermissionCustomerAdmin)
	create, hangup, update := h2SIPTimes()
	channel := &cmchannel.Channel{SIPCallID: "sip-call-1"}
	setup := func(c *cmcall.Call, ch *cmchannel.Channel, chErr error, final func(ctx context.Context, req *requesthandler.MockRequestHandler)) func(context.Context, *requesthandler.MockRequestHandler, *utilhandler.MockUtilHandler, *dbhandler.MockDBHandler) {
		return func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
			req.EXPECT().CallV1CallGet(ctx, h2SIPCallID).Return(c, nil)
			if ch != nil || chErr != nil {
				req.EXPECT().CallV1ChannelGet(ctx, c.ChannelID).Return(ch, chErr)
			}
			if final != nil {
				final(ctx, req)
			}
		}
	}
	run := func(ctx context.Context, h *serviceHandler) (any, error) { return call(ctx, h, admin) }

	return []h2Case{
		{
			name: "direct access rejected",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return call(ctx, h, h2DirectIdentity(h2CustomerID))
			},
			wantErr: serviceerrors.ErrDirectAccessNotSupported,
		},
		{
			name: "call lookup fails",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				req.EXPECT().CallV1CallGet(ctx, h2SIPCallID).Return(nil, errH2RPC)
			},
			call:    run,
			wantErr: serviceerrors.ErrNotFound,
		},
		{
			name:    "cross tenant call",
			setup:   setup(h2SIPCall(h2OtherCustomerID, "chan-1", &create, &hangup, &update), nil, nil, nil),
			call:    run,
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name:  "no permission",
			setup: setup(h2SIPCall(h2CustomerID, "chan-1", &create, &hangup, &update), nil, nil, nil),
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return call(ctx, h, h2AgentIdentity(h2CustomerID, amagent.PermissionNone))
			},
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name:    "call without channel",
			setup:   setup(h2SIPCall(h2CustomerID, "", &create, &hangup, &update), nil, nil, nil),
			call:    run,
			wantErr: serviceerrors.ErrNotFound,
		},
		{
			name:    "channel lookup fails",
			setup:   setup(h2SIPCall(h2CustomerID, "chan-1", &create, &hangup, &update), nil, errH2RPC, nil),
			call:    run,
			wantErr: serviceerrors.ErrNotFound,
		},
		{
			name:    "channel without sip call id",
			setup:   setup(h2SIPCall(h2CustomerID, "chan-1", &create, &hangup, &update), &cmchannel.Channel{}, nil, nil),
			call:    run,
			wantErr: serviceerrors.ErrNotFound,
		},
		{
			name:    "call without create time",
			setup:   setup(h2SIPCall(h2CustomerID, "chan-1", nil, &hangup, &update), channel, nil, nil),
			call:    run,
			wantErr: serviceerrors.ErrNotFound,
		},
		{
			name: "upstream failure",
			setup: setup(h2SIPCall(h2CustomerID, "chan-1", &create, &hangup, &update), channel, nil, func(ctx context.Context, req *requesthandler.MockRequestHandler) {
				rpc(ctx, req, create.Format(time.RFC3339), hangup.Format(time.RFC3339), true)
			}),
			call:    run,
			wantErr: serviceerrors.ErrServiceUnavailable,
		},
		{
			name: "success uses hangup time as end",
			setup: setup(h2SIPCall(h2CustomerID, "chan-1", &create, &hangup, &update), channel, nil, func(ctx context.Context, req *requesthandler.MockRequestHandler) {
				rpc(ctx, req, create.Format(time.RFC3339), hangup.Format(time.RFC3339), false)
			}),
			call:    run,
			wantRes: wantRes,
		},
		{
			name: "success falls back to update time when not hung up",
			setup: setup(h2SIPCall(h2CustomerID, "chan-1", &create, nil, &update), channel, nil, func(ctx context.Context, req *requesthandler.MockRequestHandler) {
				rpc(ctx, req, create.Format(time.RFC3339), update.Format(time.RFC3339), false)
			}),
			call:    run,
			wantRes: wantRes,
		},
	}
}

func Test_TimelineSIPAnalysisGet_h2(t *testing.T) {
	expect := &tmsipmessage.SIPAnalysisResponse{}
	cases := h2SIPCases(
		func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) (any, error) {
			return h.TimelineSIPAnalysisGet(ctx, a, h2SIPCallID)
		},
		func(ctx context.Context, req *requesthandler.MockRequestHandler, from, to string, failure bool) {
			if failure {
				req.EXPECT().TimelineV1SIPAnalysisGet(ctx, h2SIPCallID, "sip-call-1", from, to).Return(nil, errH2RPC)
				return
			}
			req.EXPECT().TimelineV1SIPAnalysisGet(ctx, h2SIPCallID, "sip-call-1", from, to).Return(expect, nil)
		},
		expect,
	)
	h2RunCases(t, cases)
}

func Test_TimelineSIPPcapGet_h2(t *testing.T) {
	expect := []byte("pcap-bytes")
	cases := h2SIPCases(
		func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) (any, error) {
			return h.TimelineSIPPcapGet(ctx, a, h2SIPCallID)
		},
		func(ctx context.Context, req *requesthandler.MockRequestHandler, from, to string, failure bool) {
			if failure {
				req.EXPECT().TimelineV1SIPPcapGet(ctx, h2SIPCallID, "sip-call-1", from, to).Return(nil, errH2RPC)
				return
			}
			req.EXPECT().TimelineV1SIPPcapGet(ctx, h2SIPCallID, "sip-call-1", from, to).Return(expect, nil)
		},
		expect,
	)
	h2RunCases(t, cases)
}
