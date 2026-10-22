package servicehandler

import (
	"context"
	"testing"
	"time"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/dbhandler"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	fmaction "monorepo/bin-flow-manager/models/action"
	fmflow "monorepo/bin-flow-manager/models/flow"

	"github.com/gofrs/uuid"
)

var (
	h2FlowID         = uuid.FromStringOrNil("a2000000-0000-0000-0000-000000000f01")
	h2FlowOnComplete = uuid.FromStringOrNil("a2000000-0000-0000-0000-000000000f02")
)

func h2Flow(id, customerID uuid.UUID) *fmflow.Flow {
	return &fmflow.Flow{
		Identity: commonidentity.Identity{ID: id, CustomerID: customerID},
		Name:     "flow",
	}
}

func h2FlowLookup(ctx context.Context, req *requesthandler.MockRequestHandler, customerID uuid.UUID, failure bool) {
	if failure {
		req.EXPECT().FlowV1FlowGet(ctx, h2FlowID).Return(nil, errH2RPC)
		return
	}
	req.EXPECT().FlowV1FlowGet(ctx, h2FlowID).Return(h2Flow(h2FlowID, customerID), nil)
}

func h2FlowRPC(set func(ctx context.Context, req *requesthandler.MockRequestHandler, res *fmflow.Flow, err error)) h2RPCFn {
	return func(ctx context.Context, req *requesthandler.MockRequestHandler, failure bool) {
		if failure {
			set(ctx, req, nil, errH2RPC)
			return
		}
		set(ctx, req, h2Flow(h2FlowID, h2CustomerID), nil)
	}
}

func Test_flowGet_h2_deleted(t *testing.T) {
	admin := h2AgentIdentity(h2CustomerID, amagent.PermissionCustomerAdmin)
	deletedAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	h2RunCases(t, []h2Case{
		{
			name: "deleted flow is not found",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				f := h2Flow(h2FlowID, h2CustomerID)
				f.TMDelete = &deletedAt
				req.EXPECT().FlowV1FlowGet(ctx, h2FlowID).Return(f, nil)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.FlowGet(ctx, admin, h2FlowID)
			},
			wantErr: serviceerrors.ErrNotFound,
		},
	})
}

func Test_FlowCreate_h2(t *testing.T) {
	actions := []fmaction.Action{{Type: fmaction.TypeAnswer}}
	admin := h2AgentIdentity(h2CustomerID, amagent.PermissionCustomerAdmin)
	manager := h2AgentIdentity(h2CustomerID, amagent.PermissionCustomerManager)
	create := func(a *auth.AuthIdentity, onComplete uuid.UUID, persist bool) func(ctx context.Context, h *serviceHandler) (any, error) {
		return func(ctx context.Context, h *serviceHandler) (any, error) {
			return h.FlowCreate(ctx, a, "nm", "dt", actions, onComplete, persist)
		}
	}
	rpc := func(ctx context.Context, req *requesthandler.MockRequestHandler, onComplete uuid.UUID, persist bool, res *fmflow.Flow, err error) {
		req.EXPECT().FlowV1FlowCreate(ctx, h2CustomerID, fmflow.TypeFlow, "nm", "dt", actions, onComplete, persist).Return(res, err)
	}

	h2RunCases(t, []h2Case{
		{
			name:    "direct access rejected",
			call:    create(h2DirectIdentity(h2CustomerID), uuid.Nil, true),
			wantErr: serviceerrors.ErrDirectAccessNotSupported,
		},
		{
			name:    "persist without permission",
			call:    create(h2AgentIdentity(h2CustomerID, amagent.PermissionNone), uuid.Nil, true),
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			// a delegate token never holds PermissionAll (it excludes project level bits).
			name:    "non-persist with delegate identity is denied",
			call:    create(auth.NewDelegateIdentity(&auth.DelegateScope{CustomerID: h2CustomerID}), uuid.Nil, false),
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name: "onComplete flow lookup fails",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				req.EXPECT().FlowV1FlowGet(ctx, h2FlowOnComplete).Return(nil, errH2RPC)
			},
			call:    create(admin, h2FlowOnComplete, true),
			wantErr: errH2RPC,
		},
		{
			name: "onComplete flow of another customer",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				req.EXPECT().FlowV1FlowGet(ctx, h2FlowOnComplete).Return(h2Flow(h2FlowOnComplete, h2OtherCustomerID), nil)
			},
			call:    create(admin, h2FlowOnComplete, true),
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name: "rpc error",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				rpc(ctx, req, uuid.Nil, true, nil, errH2RPC)
			},
			call:    create(manager, uuid.Nil, true),
			wantErr: errH2RPC,
		},
		{
			name: "success with onComplete flow",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				req.EXPECT().FlowV1FlowGet(ctx, h2FlowOnComplete).Return(h2Flow(h2FlowOnComplete, h2CustomerID), nil)
				rpc(ctx, req, h2FlowOnComplete, true, h2Flow(h2FlowID, h2CustomerID), nil)
			},
			call:    create(admin, h2FlowOnComplete, true),
			wantRes: h2Flow(h2FlowID, h2CustomerID).ConvertWebhookMessage(),
		},
		{
			name: "success non-persist",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				rpc(ctx, req, uuid.Nil, false, h2Flow(h2FlowID, h2CustomerID), nil)
			},
			call:    create(h2AgentIdentity(h2CustomerID, amagent.PermissionAll), uuid.Nil, false),
			wantRes: h2Flow(h2FlowID, h2CustomerID).ConvertWebhookMessage(),
		},
	})
}

func Test_FlowDelete_h2(t *testing.T) {
	h2RunCases(t, h2StdCases(
		func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) (any, error) {
			return h.FlowDelete(ctx, a, h2FlowID)
		},
		h2FlowLookup,
		h2FlowRPC(func(ctx context.Context, req *requesthandler.MockRequestHandler, res *fmflow.Flow, err error) {
			req.EXPECT().FlowV1FlowDelete(ctx, h2FlowID).Return(res, err)
		}),
		h2Flow(h2FlowID, h2CustomerID).ConvertWebhookMessage(),
	))
}

func Test_FlowDirectHashRegenerate_h2(t *testing.T) {
	h2RunCases(t, h2StdCases(
		func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) (any, error) {
			return h.FlowDirectHashRegenerate(ctx, a, h2FlowID)
		},
		h2FlowLookup,
		h2FlowRPC(func(ctx context.Context, req *requesthandler.MockRequestHandler, res *fmflow.Flow, err error) {
			req.EXPECT().FlowV1FlowDirectHashRegenerate(ctx, h2FlowID).Return(res, err)
		}),
		h2Flow(h2FlowID, h2CustomerID).ConvertWebhookMessage(),
	))
}

func Test_FlowGet_h2(t *testing.T) {
	var cases []h2Case
	for _, c := range h2StdCases(
		func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) (any, error) {
			return h.FlowGet(ctx, a, h2FlowID)
		},
		h2FlowLookup,
		func(context.Context, *requesthandler.MockRequestHandler, bool) {},
		h2Flow(h2FlowID, h2CustomerID).ConvertWebhookMessage(),
	) {
		if c.name != "rpc error" {
			cases = append(cases, c)
		}
	}
	h2RunCases(t, cases)
}

func Test_FlowUpdateActions_h2(t *testing.T) {
	actions := []fmaction.Action{{Type: fmaction.TypeAnswer}}
	h2RunCases(t, h2StdCases(
		func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) (any, error) {
			return h.FlowUpdateActions(ctx, a, h2FlowID, actions)
		},
		h2FlowLookup,
		h2FlowRPC(func(ctx context.Context, req *requesthandler.MockRequestHandler, res *fmflow.Flow, err error) {
			req.EXPECT().FlowV1FlowUpdateActions(ctx, h2FlowID, actions).Return(res, err)
		}),
		h2Flow(h2FlowID, h2CustomerID).ConvertWebhookMessage(),
	))
}

func Test_FlowUpdate_h2(t *testing.T) {
	name, detail := "nm", "dt"
	actions := []fmaction.Action{{Type: fmaction.TypeAnswer}}
	onComplete := h2FlowOnComplete
	admin := h2AgentIdentity(h2CustomerID, amagent.PermissionCustomerAdmin)

	h2RunCases(t, append(h2StdCases(
		func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) (any, error) {
			return h.FlowUpdate(ctx, a, h2FlowID, &name, &detail, actions, nil)
		},
		h2FlowLookup,
		h2FlowRPC(func(ctx context.Context, req *requesthandler.MockRequestHandler, res *fmflow.Flow, err error) {
			req.EXPECT().FlowV1FlowUpdate(ctx, h2FlowID, &name, &detail, actions, (*uuid.UUID)(nil)).Return(res, err)
		}),
		h2Flow(h2FlowID, h2CustomerID).ConvertWebhookMessage(),
	),
		h2Case{
			name: "onComplete flow lookup fails",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				h2FlowLookup(ctx, req, h2CustomerID, false)
				req.EXPECT().FlowV1FlowGet(ctx, h2FlowOnComplete).Return(nil, errH2RPC)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.FlowUpdate(ctx, admin, h2FlowID, &name, &detail, actions, &onComplete)
			},
			wantErr: errH2RPC,
		},
		h2Case{
			name: "onComplete flow of another customer",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				h2FlowLookup(ctx, req, h2CustomerID, false)
				req.EXPECT().FlowV1FlowGet(ctx, h2FlowOnComplete).Return(h2Flow(h2FlowOnComplete, h2OtherCustomerID), nil)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.FlowUpdate(ctx, admin, h2FlowID, &name, &detail, actions, &onComplete)
			},
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		h2Case{
			name: "success with onComplete flow",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				h2FlowLookup(ctx, req, h2CustomerID, false)
				req.EXPECT().FlowV1FlowGet(ctx, h2FlowOnComplete).Return(h2Flow(h2FlowOnComplete, h2CustomerID), nil)
				req.EXPECT().FlowV1FlowUpdate(ctx, h2FlowID, &name, &detail, actions, &onComplete).Return(h2Flow(h2FlowID, h2CustomerID), nil)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.FlowUpdate(ctx, admin, h2FlowID, &name, &detail, actions, &onComplete)
			},
			wantRes: h2Flow(h2FlowID, h2CustomerID).ConvertWebhookMessage(),
		},
	))
}

func Test_FlowList_h2(t *testing.T) {
	admin := h2AgentIdentity(h2CustomerID, amagent.PermissionCustomerManager)
	filters := map[fmflow.Field]any{
		fmflow.FieldCustomerID: h2CustomerID,
		fmflow.FieldType:       fmflow.TypeFlow,
		fmflow.FieldDeleted:    false,
	}
	list := []fmflow.Flow{*h2Flow(h2FlowID, h2CustomerID)}

	h2RunCases(t, []h2Case{
		{
			name: "direct access rejected",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.FlowList(ctx, h2DirectIdentity(h2CustomerID), 10, "tok")
			},
			wantErr: serviceerrors.ErrDirectAccessNotSupported,
		},
		{
			name: "no permission",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.FlowList(ctx, h2AgentIdentity(h2CustomerID, amagent.PermissionNone), 10, "tok")
			},
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name: "rpc error",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				req.EXPECT().FlowV1FlowList(ctx, "tok", uint64(10), filters).Return(nil, errH2RPC)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.FlowList(ctx, admin, 10, "tok")
			},
			wantErr: errH2RPC,
		},
		{
			name: "success with empty token",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, util *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				util.EXPECT().TimeGetCurTime().Return("2026-01-01 00:00:00.000000")
				req.EXPECT().FlowV1FlowList(ctx, "2026-01-01 00:00:00.000000", uint64(5), filters).Return(list, nil)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.FlowList(ctx, admin, 5, "")
			},
			wantRes: []*fmflow.WebhookMessage{list[0].ConvertWebhookMessage()},
		},
	})
}
