package servicehandler

import (
	"context"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/dbhandler"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	cacampaign "monorepo/bin-campaign-manager/models/campaign"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	fmaction "monorepo/bin-flow-manager/models/action"

	"github.com/gofrs/uuid"
)

var h2CampaignID = uuid.FromStringOrNil("a2000000-0000-0000-0000-000000000e01")

func h2Campaign(customerID uuid.UUID) *cacampaign.Campaign {
	return &cacampaign.Campaign{
		Identity: commonidentity.Identity{ID: h2CampaignID, CustomerID: customerID},
		Name:     "camp",
	}
}

func h2CampaignLookup(ctx context.Context, req *requesthandler.MockRequestHandler, customerID uuid.UUID, failure bool) {
	if failure {
		req.EXPECT().CampaignV1CampaignGet(ctx, h2CampaignID).Return(nil, errH2RPC)
		return
	}
	req.EXPECT().CampaignV1CampaignGet(ctx, h2CampaignID).Return(h2Campaign(customerID), nil)
}

func h2CampaignRPC(set func(ctx context.Context, req *requesthandler.MockRequestHandler, res *cacampaign.Campaign, err error)) h2RPCFn {
	return func(ctx context.Context, req *requesthandler.MockRequestHandler, failure bool) {
		if failure {
			set(ctx, req, nil, errH2RPC)
			return
		}
		set(ctx, req, h2Campaign(h2CustomerID), nil)
	}
}

func Test_CampaignCreate_h2(t *testing.T) {
	actions := []fmaction.Action{{Type: fmaction.TypeAnswer}}
	outplan := uuid.FromStringOrNil("a2000000-0000-0000-0000-000000000e02")
	outdial := uuid.FromStringOrNil("a2000000-0000-0000-0000-000000000e03")
	queue := uuid.FromStringOrNil("a2000000-0000-0000-0000-000000000e04")
	next := uuid.FromStringOrNil("a2000000-0000-0000-0000-000000000e05")
	h2RunCases(t, h2StdCases(
		func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) (any, error) {
			return h.CampaignCreate(ctx, a, "nm", "dt", cacampaign.TypeCall, 100, cacampaign.EndHandleStop, actions, outplan, outdial, queue, next)
		},
		nil,
		h2CampaignRPC(func(ctx context.Context, req *requesthandler.MockRequestHandler, res *cacampaign.Campaign, err error) {
			req.EXPECT().CampaignV1CampaignCreate(ctx, uuid.Nil, h2CustomerID, cacampaign.TypeCall, "nm", "dt", 100, cacampaign.EndHandleStop, actions, outplan, outdial, queue, next).Return(res, err)
		}),
		h2Campaign(h2CustomerID).ConvertWebhookMessage(),
	))
}

func Test_CampaignGetsByCustomerID_h2(t *testing.T) {
	admin := h2AgentIdentity(h2CustomerID, amagent.PermissionCustomerAdmin)
	filters := map[cacampaign.Field]any{cacampaign.FieldCustomerID: h2CustomerID}
	list := []cacampaign.Campaign{*h2Campaign(h2CustomerID)}

	h2RunCases(t, []h2Case{
		{
			name: "direct access rejected",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.CampaignGetsByCustomerID(ctx, h2DirectIdentity(h2CustomerID), 10, "tok")
			},
			wantErr: serviceerrors.ErrDirectAccessNotSupported,
		},
		{
			name: "no permission",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.CampaignGetsByCustomerID(ctx, h2AgentIdentity(h2CustomerID, amagent.PermissionNone), 10, "tok")
			},
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name: "rpc error",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				req.EXPECT().CampaignV1CampaignList(ctx, "tok", uint64(10), filters).Return(nil, errH2RPC)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.CampaignGetsByCustomerID(ctx, admin, 10, "tok")
			},
			wantErr: errH2RPC,
		},
		{
			name: "success with empty token",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, util *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				util.EXPECT().TimeGetCurTime().Return("2026-01-01 00:00:00.000000")
				req.EXPECT().CampaignV1CampaignList(ctx, "2026-01-01 00:00:00.000000", uint64(7), filters).Return(list, nil)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.CampaignGetsByCustomerID(ctx, admin, 7, "")
			},
			wantRes: []*cacampaign.WebhookMessage{list[0].ConvertWebhookMessage()},
		},
	})
}

func Test_CampaignGet_h2(t *testing.T) {
	var cases []h2Case
	for _, c := range h2StdCases(
		func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) (any, error) {
			return h.CampaignGet(ctx, a, h2CampaignID)
		},
		h2CampaignLookup,
		func(context.Context, *requesthandler.MockRequestHandler, bool) {},
		h2Campaign(h2CustomerID).ConvertWebhookMessage(),
	) {
		if c.name != "rpc error" {
			cases = append(cases, c)
		}
	}
	h2RunCases(t, cases)
}

func Test_CampaignDelete_h2(t *testing.T) {
	h2RunCases(t, h2StdCases(
		func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) (any, error) {
			return h.CampaignDelete(ctx, a, h2CampaignID)
		},
		h2CampaignLookup,
		h2CampaignRPC(func(ctx context.Context, req *requesthandler.MockRequestHandler, res *cacampaign.Campaign, err error) {
			req.EXPECT().CampaignV1CampaignDelete(ctx, h2CampaignID).Return(res, err)
		}),
		h2Campaign(h2CustomerID).ConvertWebhookMessage(),
	))
}

func Test_CampaignUpdateBasicInfo_h2(t *testing.T) {
	name, detail := "nm", "dt"
	ct := cacampaign.TypeCall
	sl := 50
	eh := cacampaign.EndHandleStop
	h2RunCases(t, h2StdCases(
		func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) (any, error) {
			return h.CampaignUpdateBasicInfo(ctx, a, h2CampaignID, &name, &detail, &ct, &sl, &eh)
		},
		h2CampaignLookup,
		h2CampaignRPC(func(ctx context.Context, req *requesthandler.MockRequestHandler, res *cacampaign.Campaign, err error) {
			req.EXPECT().CampaignV1CampaignUpdateBasicInfo(ctx, h2CampaignID, &name, &detail, &ct, &sl, &eh).Return(res, err)
		}),
		h2Campaign(h2CustomerID).ConvertWebhookMessage(),
	))
}

func Test_CampaignUpdateStatus_h2(t *testing.T) {
	h2RunCases(t, h2StdCases(
		func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) (any, error) {
			return h.CampaignUpdateStatus(ctx, a, h2CampaignID, cacampaign.StatusRun)
		},
		h2CampaignLookup,
		h2CampaignRPC(func(ctx context.Context, req *requesthandler.MockRequestHandler, res *cacampaign.Campaign, err error) {
			req.EXPECT().CampaignV1CampaignUpdateStatus(ctx, h2CampaignID, cacampaign.StatusRun).Return(res, err)
		}),
		h2Campaign(h2CustomerID).ConvertWebhookMessage(),
	))
}

func Test_CampaignUpdateServiceLevel_h2(t *testing.T) {
	h2RunCases(t, h2StdCases(
		func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) (any, error) {
			return h.CampaignUpdateServiceLevel(ctx, a, h2CampaignID, 80)
		},
		h2CampaignLookup,
		h2CampaignRPC(func(ctx context.Context, req *requesthandler.MockRequestHandler, res *cacampaign.Campaign, err error) {
			req.EXPECT().CampaignV1CampaignUpdateServiceLevel(ctx, h2CampaignID, 80).Return(res, err)
		}),
		h2Campaign(h2CustomerID).ConvertWebhookMessage(),
	))
}

func Test_CampaignUpdateActions_h2(t *testing.T) {
	actions := []fmaction.Action{{Type: fmaction.TypeAnswer}}
	h2RunCases(t, h2StdCases(
		func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) (any, error) {
			return h.CampaignUpdateActions(ctx, a, h2CampaignID, actions)
		},
		h2CampaignLookup,
		h2CampaignRPC(func(ctx context.Context, req *requesthandler.MockRequestHandler, res *cacampaign.Campaign, err error) {
			req.EXPECT().CampaignV1CampaignUpdateActions(ctx, h2CampaignID, actions).Return(res, err)
		}),
		h2Campaign(h2CustomerID).ConvertWebhookMessage(),
	))
}

func Test_CampaignUpdateResourceInfo_h2(t *testing.T) {
	outplan := uuid.FromStringOrNil("a2000000-0000-0000-0000-000000000e02")
	outdial := uuid.FromStringOrNil("a2000000-0000-0000-0000-000000000e03")
	queue := uuid.FromStringOrNil("a2000000-0000-0000-0000-000000000e04")
	next := uuid.FromStringOrNil("a2000000-0000-0000-0000-000000000e05")
	h2RunCases(t, h2StdCases(
		func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) (any, error) {
			return h.CampaignUpdateResourceInfo(ctx, a, h2CampaignID, outplan, outdial, queue, next)
		},
		h2CampaignLookup,
		h2CampaignRPC(func(ctx context.Context, req *requesthandler.MockRequestHandler, res *cacampaign.Campaign, err error) {
			req.EXPECT().CampaignV1CampaignUpdateResourceInfo(ctx, h2CampaignID, outplan, outdial, queue, next).Return(res, err)
		}),
		h2Campaign(h2CustomerID).ConvertWebhookMessage(),
	))
}

func Test_CampaignUpdateNextCampaignID_h2(t *testing.T) {
	next := uuid.FromStringOrNil("a2000000-0000-0000-0000-000000000e05")
	h2RunCases(t, h2StdCases(
		func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) (any, error) {
			return h.CampaignUpdateNextCampaignID(ctx, a, h2CampaignID, next)
		},
		h2CampaignLookup,
		h2CampaignRPC(func(ctx context.Context, req *requesthandler.MockRequestHandler, res *cacampaign.Campaign, err error) {
			req.EXPECT().CampaignV1CampaignUpdateNextCampaignID(ctx, h2CampaignID, next).Return(res, err)
		}),
		h2Campaign(h2CustomerID).ConvertWebhookMessage(),
	))
}
