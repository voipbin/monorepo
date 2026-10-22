package servicehandler

import (
	"context"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/dbhandler"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	rmrag "monorepo/bin-rag-manager/models/rag"

	"github.com/gofrs/uuid"
)

func h2Rag(customerID uuid.UUID) *rmrag.Rag {
	return &rmrag.Rag{ID: h2RagID, CustomerID: customerID, Name: "rag"}
}

func h2RagLookup(ctx context.Context, req *requesthandler.MockRequestHandler, customerID uuid.UUID, failure bool) {
	if failure {
		req.EXPECT().RagV1RagGet(ctx, h2RagID).Return(nil, errH2RPC)
		return
	}
	req.EXPECT().RagV1RagGet(ctx, h2RagID).Return(h2Rag(customerID), nil)
}

func h2RagRPC(set func(ctx context.Context, req *requesthandler.MockRequestHandler, res *rmrag.Rag, err error)) h2RPCFn {
	return func(ctx context.Context, req *requesthandler.MockRequestHandler, failure bool) {
		if failure {
			set(ctx, req, nil, errH2RPC)
			return
		}
		set(ctx, req, h2Rag(h2CustomerID), nil)
	}
}

func Test_RagCreate_h2(t *testing.T) {
	files := []uuid.UUID{uuid.FromStringOrNil("a2000000-0000-0000-0000-000000000d01")}
	urls := []string{"https://example.com/doc"}
	cases := h2StdCases(
		func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) (any, error) {
			return h.RagCreate(ctx, a, "nm", "dt", files, urls)
		},
		nil,
		h2RagRPC(func(ctx context.Context, req *requesthandler.MockRequestHandler, res *rmrag.Rag, err error) {
			req.EXPECT().RagV1RagCreate(ctx, h2CustomerID, "nm", "dt", files, urls).Return(res, err)
		}),
		h2Rag(h2CustomerID).ConvertWebhookMessage(),
	)
	cases = append(cases, h2Case{
		name: "customer manager is not enough",
		call: func(ctx context.Context, h *serviceHandler) (any, error) {
			return h.RagCreate(ctx, h2AgentIdentity(h2CustomerID, amagent.PermissionCustomerManager), "nm", "dt", files, urls)
		},
		wantErr: serviceerrors.ErrPermissionDenied,
	})
	h2RunCases(t, cases)
}

func Test_RagGet_h2(t *testing.T) {
	var cases []h2Case
	for _, c := range h2StdCases(
		func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) (any, error) {
			return h.RagGet(ctx, a, h2RagID)
		},
		h2RagLookup,
		func(context.Context, *requesthandler.MockRequestHandler, bool) {},
		h2Rag(h2CustomerID).ConvertWebhookMessage(),
	) {
		if c.name != "rpc error" {
			cases = append(cases, c)
		}
	}
	h2RunCases(t, cases)
}

func Test_RagGets_h2(t *testing.T) {
	admin := h2AgentIdentity(h2CustomerID, amagent.PermissionCustomerAdmin)
	filters := map[rmrag.Field]any{rmrag.FieldCustomerID: h2CustomerID}
	list := []*rmrag.Rag{h2Rag(h2CustomerID)}

	h2RunCases(t, []h2Case{
		{
			name: "direct access rejected",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.RagGets(ctx, h2DirectIdentity(h2CustomerID), 10, "tok")
			},
			wantErr: serviceerrors.ErrDirectAccessNotSupported,
		},
		{
			name: "no permission",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.RagGets(ctx, h2AgentIdentity(h2CustomerID, amagent.PermissionNone), 10, "tok")
			},
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name: "rpc error",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				req.EXPECT().RagV1RagGets(ctx, "tok", uint64(10), filters).Return(nil, errH2RPC)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.RagGets(ctx, admin, 10, "tok")
			},
			wantErr: errH2RPC,
		},
		{
			name: "success with empty token",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, util *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				util.EXPECT().TimeGetCurTime().Return("2026-01-01 00:00:00.000000")
				req.EXPECT().RagV1RagGets(ctx, "2026-01-01 00:00:00.000000", uint64(2), filters).Return(list, nil)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.RagGets(ctx, admin, 2, "")
			},
			wantRes: []*rmrag.WebhookMessage{list[0].ConvertWebhookMessage()},
		},
	})
}

func Test_RagUpdate_h2(t *testing.T) {
	fields := map[rmrag.Field]any{rmrag.FieldName: "new"}
	h2RunCases(t, h2StdCases(
		func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) (any, error) {
			return h.RagUpdate(ctx, a, h2RagID, fields)
		},
		h2RagLookup,
		h2RagRPC(func(ctx context.Context, req *requesthandler.MockRequestHandler, res *rmrag.Rag, err error) {
			req.EXPECT().RagV1RagUpdate(ctx, h2RagID, fields).Return(res, err)
		}),
		h2Rag(h2CustomerID).ConvertWebhookMessage(),
	))
}

func Test_RagDelete_h2(t *testing.T) {
	// RagDelete returns the looked-up rag, so the RPC only reports an error.
	h2RunCases(t, h2StdCases(
		func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) (any, error) {
			return h.RagDelete(ctx, a, h2RagID)
		},
		h2RagLookup,
		func(ctx context.Context, req *requesthandler.MockRequestHandler, failure bool) {
			if failure {
				req.EXPECT().RagV1RagDelete(ctx, h2RagID).Return(errH2RPC)
				return
			}
			req.EXPECT().RagV1RagDelete(ctx, h2RagID).Return(nil)
		},
		h2Rag(h2CustomerID).ConvertWebhookMessage(),
	))
}

func Test_RagAddSources_h2(t *testing.T) {
	files := []uuid.UUID{uuid.FromStringOrNil("a2000000-0000-0000-0000-000000000d01")}
	urls := []string{"https://example.com/more"}
	h2RunCases(t, h2StdCases(
		func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) (any, error) {
			return h.RagAddSources(ctx, a, h2RagID, files, urls)
		},
		h2RagLookup,
		h2RagRPC(func(ctx context.Context, req *requesthandler.MockRequestHandler, res *rmrag.Rag, err error) {
			req.EXPECT().RagV1RagAddSources(ctx, h2RagID, files, urls).Return(res, err)
		}),
		h2Rag(h2CustomerID).ConvertWebhookMessage(),
	))
}

func Test_RagRemoveSource_h2(t *testing.T) {
	sourceID := uuid.FromStringOrNil("a2000000-0000-0000-0000-000000000d02")
	h2RunCases(t, h2StdCases(
		func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) (any, error) {
			return h.RagRemoveSource(ctx, a, h2RagID, sourceID)
		},
		h2RagLookup,
		h2RagRPC(func(ctx context.Context, req *requesthandler.MockRequestHandler, res *rmrag.Rag, err error) {
			req.EXPECT().RagV1RagRemoveSource(ctx, h2RagID, sourceID).Return(res, err)
		}),
		h2Rag(h2CustomerID).ConvertWebhookMessage(),
	))
}
