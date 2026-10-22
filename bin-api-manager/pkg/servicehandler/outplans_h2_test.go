package servicehandler

import (
	"context"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/dbhandler"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	caoutplan "monorepo/bin-campaign-manager/models/outplan"
	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
)

var h2OutplanID = uuid.FromStringOrNil("a2000000-0000-0000-0000-000000002001")

func h2Outplan(customerID uuid.UUID) *caoutplan.Outplan {
	return &caoutplan.Outplan{
		Identity: commonidentity.Identity{ID: h2OutplanID, CustomerID: customerID},
		Name:     "outplan",
	}
}

func h2OutplanLookup(ctx context.Context, req *requesthandler.MockRequestHandler, customerID uuid.UUID, failure bool) {
	if failure {
		req.EXPECT().CampaignV1OutplanGet(ctx, h2OutplanID).Return(nil, errH2RPC)
		return
	}
	req.EXPECT().CampaignV1OutplanGet(ctx, h2OutplanID).Return(h2Outplan(customerID), nil)
}

func h2OutplanRPC(set func(ctx context.Context, req *requesthandler.MockRequestHandler, res *caoutplan.Outplan, err error)) h2RPCFn {
	return func(ctx context.Context, req *requesthandler.MockRequestHandler, failure bool) {
		if failure {
			set(ctx, req, nil, errH2RPC)
			return
		}
		set(ctx, req, h2Outplan(h2CustomerID), nil)
	}
}

func Test_OutplanCreate_h2(t *testing.T) {
	src := &commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15555550100"}
	h2RunCases(t, h2StdCases(
		func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) (any, error) {
			return h.OutplanCreate(ctx, a, "nm", "dt", src, 30000, 600000, 1, 2, 3, 4, 5)
		},
		nil,
		h2OutplanRPC(func(ctx context.Context, req *requesthandler.MockRequestHandler, res *caoutplan.Outplan, err error) {
			req.EXPECT().CampaignV1OutplanCreate(ctx, h2CustomerID, "nm", "dt", src, 30000, 600000, 1, 2, 3, 4, 5).Return(res, err)
		}),
		h2Outplan(h2CustomerID).ConvertWebhookMessage(),
	))
}

func Test_OutplanDelete_h2(t *testing.T) {
	h2RunCases(t, h2StdCases(
		func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) (any, error) {
			return h.OutplanDelete(ctx, a, h2OutplanID)
		},
		h2OutplanLookup,
		h2OutplanRPC(func(ctx context.Context, req *requesthandler.MockRequestHandler, res *caoutplan.Outplan, err error) {
			req.EXPECT().CampaignV1OutplanDelete(ctx, h2OutplanID).Return(res, err)
		}),
		h2Outplan(h2CustomerID).ConvertWebhookMessage(),
	))
}

func Test_OutplanGet_h2(t *testing.T) {
	var cases []h2Case
	for _, c := range h2StdCases(
		func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) (any, error) {
			return h.OutplanGet(ctx, a, h2OutplanID)
		},
		h2OutplanLookup,
		func(context.Context, *requesthandler.MockRequestHandler, bool) {},
		h2Outplan(h2CustomerID).ConvertWebhookMessage(),
	) {
		if c.name != "rpc error" {
			cases = append(cases, c)
		}
	}
	h2RunCases(t, cases)
}

func Test_OutplanUpdateBasicInfo_h2(t *testing.T) {
	h2RunCases(t, h2StdCases(
		func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) (any, error) {
			return h.OutplanUpdateBasicInfo(ctx, a, h2OutplanID, "nm", "dt")
		},
		h2OutplanLookup,
		h2OutplanRPC(func(ctx context.Context, req *requesthandler.MockRequestHandler, res *caoutplan.Outplan, err error) {
			req.EXPECT().CampaignV1OutplanUpdateBasicInfo(ctx, h2OutplanID, "nm", "dt").Return(res, err)
		}),
		h2Outplan(h2CustomerID).ConvertWebhookMessage(),
	))
}

func Test_OutplanUpdateDialInfo_h2(t *testing.T) {
	src := &commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15555550100"}
	dialTimeout, tryInterval, c0, c1, c2, c3, c4 := 30000, 600000, 1, 2, 3, 4, 5
	h2RunCases(t, h2StdCases(
		func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) (any, error) {
			return h.OutplanUpdateDialInfo(ctx, a, h2OutplanID, src, &dialTimeout, &tryInterval, &c0, &c1, &c2, &c3, &c4)
		},
		h2OutplanLookup,
		h2OutplanRPC(func(ctx context.Context, req *requesthandler.MockRequestHandler, res *caoutplan.Outplan, err error) {
			req.EXPECT().CampaignV1OutplanUpdateDialInfo(ctx, h2OutplanID, src, &dialTimeout, &tryInterval, &c0, &c1, &c2, &c3, &c4).Return(res, err)
		}),
		h2Outplan(h2CustomerID).ConvertWebhookMessage(),
	))
}

func Test_OutplanGetsByCustomerID_h2(t *testing.T) {
	admin := h2AgentIdentity(h2CustomerID, amagent.PermissionCustomerAdmin)
	filters := map[caoutplan.Field]any{caoutplan.FieldCustomerID: h2CustomerID}
	list := []caoutplan.Outplan{*h2Outplan(h2CustomerID)}

	h2RunCases(t, []h2Case{
		{
			name: "direct access rejected",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.OutplanGetsByCustomerID(ctx, h2DirectIdentity(h2CustomerID), 10, "tok")
			},
			wantErr: serviceerrors.ErrDirectAccessNotSupported,
		},
		{
			name: "no permission",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.OutplanGetsByCustomerID(ctx, h2AgentIdentity(h2CustomerID, amagent.PermissionNone), 10, "tok")
			},
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name: "rpc error",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				req.EXPECT().CampaignV1OutplanList(ctx, "tok", uint64(10), filters).Return(nil, errH2RPC)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.OutplanGetsByCustomerID(ctx, admin, 10, "tok")
			},
			wantErr: errH2RPC,
		},
		{
			name: "success with empty token",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, util *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				util.EXPECT().TimeGetCurTime().Return("2026-01-01 00:00:00.000000")
				req.EXPECT().CampaignV1OutplanList(ctx, "2026-01-01 00:00:00.000000", uint64(6), filters).Return(list, nil)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.OutplanGetsByCustomerID(ctx, admin, 6, "")
			},
			wantRes: []*caoutplan.WebhookMessage{list[0].ConvertWebhookMessage()},
		},
	})
}
