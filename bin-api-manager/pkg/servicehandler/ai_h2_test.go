package servicehandler

import (
	"context"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	amai "monorepo/bin-ai-manager/models/ai"
	amtool "monorepo/bin-ai-manager/models/tool"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/dbhandler"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	rmrag "monorepo/bin-rag-manager/models/rag"

	"github.com/gofrs/uuid"
)

var (
	h2AIID  = uuid.FromStringOrNil("a2000000-0000-0000-0000-000000000a01")
	h2RagID = uuid.FromStringOrNil("a2000000-0000-0000-0000-000000000a02")
)

func h2AI(customerID uuid.UUID) *amai.AI {
	return &amai.AI{Identity: commonidentity.Identity{ID: h2AIID, CustomerID: customerID}, Name: "ai"}
}

func h2AILookup(ctx context.Context, req *requesthandler.MockRequestHandler, customerID uuid.UUID, failure bool) {
	if failure {
		req.EXPECT().AIV1AIGet(ctx, h2AIID).Return(nil, errH2RPC)
		return
	}
	req.EXPECT().AIV1AIGet(ctx, h2AIID).Return(h2AI(customerID), nil)
}

func h2AIRPC(set func(ctx context.Context, req *requesthandler.MockRequestHandler, res *amai.AI, err error)) h2RPCFn {
	return func(ctx context.Context, req *requesthandler.MockRequestHandler, failure bool) {
		if failure {
			set(ctx, req, nil, errH2RPC)
			return
		}
		set(ctx, req, h2AI(h2CustomerID), nil)
	}
}

func Test_AIDirectHashRegenerate_h2(t *testing.T) {
	h2RunCases(t, h2StdCases(
		func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) (any, error) {
			return h.AIDirectHashRegenerate(ctx, a, h2AIID)
		},
		h2AILookup,
		h2AIRPC(func(ctx context.Context, req *requesthandler.MockRequestHandler, res *amai.AI, err error) {
			req.EXPECT().AIV1AIDirectHashRegenerate(ctx, h2AIID).Return(res, err)
		}),
		h2AI(h2CustomerID).ConvertWebhookMessage(),
	))
}

func Test_AIActivateInsight_h2(t *testing.T) {
	h2RunCases(t, h2StdCases(
		func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) (any, error) {
			return h.AIActivateInsight(ctx, a, h2AIID)
		},
		h2AILookup,
		h2AIRPC(func(ctx context.Context, req *requesthandler.MockRequestHandler, res *amai.AI, err error) {
			req.EXPECT().AIV1AIActivateInsight(ctx, h2AIID).Return(res, err)
		}),
		h2AI(h2CustomerID).ConvertWebhookMessage(),
	))
}

func Test_AIDelete_h2(t *testing.T) {
	h2RunCases(t, h2StdCases(
		func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) (any, error) {
			return h.AIDelete(ctx, a, h2AIID)
		},
		h2AILookup,
		h2AIRPC(func(ctx context.Context, req *requesthandler.MockRequestHandler, res *amai.AI, err error) {
			req.EXPECT().AIV1AIDelete(ctx, h2AIID).Return(res, err)
		}),
		h2AI(h2CustomerID).ConvertWebhookMessage(),
	))
}

func Test_AIGet_h2(t *testing.T) {
	var cases []h2Case
	for _, c := range h2StdCases(
		func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) (any, error) {
			return h.AIGet(ctx, a, h2AIID)
		},
		h2AILookup,
		func(context.Context, *requesthandler.MockRequestHandler, bool) {},
		h2AI(h2CustomerID).ConvertWebhookMessage(),
	) {
		if c.name != "rpc error" {
			cases = append(cases, c)
		}
	}
	h2RunCases(t, cases)
}

func Test_AIGetsByCustomerID_h2(t *testing.T) {
	admin := h2AgentIdentity(h2CustomerID, amagent.PermissionCustomerAdmin)
	filters := map[amai.Field]any{
		amai.FieldDeleted:    false,
		amai.FieldCustomerID: h2CustomerID,
	}
	h2RunCases(t, []h2Case{
		{
			name: "direct access rejected",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.AIGetsByCustomerID(ctx, h2DirectIdentity(h2CustomerID), 10, "tok")
			},
			wantErr: serviceerrors.ErrDirectAccessNotSupported,
		},
		{
			name: "no permission",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.AIGetsByCustomerID(ctx, h2AgentIdentity(h2CustomerID, amagent.PermissionNone), 10, "tok")
			},
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name: "rpc error",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				req.EXPECT().AIV1AIList(ctx, "tok", uint64(10), filters).Return(nil, errH2RPC)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.AIGetsByCustomerID(ctx, admin, 10, "tok")
			},
			wantErr: errH2RPC,
		},
		{
			name: "success with empty token",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, util *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				util.EXPECT().TimeGetCurTime().Return("2026-01-01 00:00:00.000000")
				req.EXPECT().AIV1AIList(ctx, "2026-01-01 00:00:00.000000", uint64(3), filters).Return([]amai.AI{*h2AI(h2CustomerID)}, nil)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.AIGetsByCustomerID(ctx, admin, 3, "")
			},
			wantRes: []*amai.WebhookMessage{h2AI(h2CustomerID).ConvertWebhookMessage()},
		},
	})
}

func Test_convertAIFilters_h2(t *testing.T) {
	h := &serviceHandler{}
	res, err := h.convertAIFilters(map[string]string{"customer_id": h2CustomerID.String(), "deleted": "false"})
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if len(res) != 2 || res[amai.FieldCustomerID] != h2CustomerID || res[amai.FieldDeleted] != false {
		t.Errorf("Wrong match. got: %v", res)
	}
}

func Test_loggableEngineModel_h2(t *testing.T) {
	tests := []struct {
		in     amai.EngineModel
		expect string
	}{
		{"openrouter.secret/model", "openrouter.<redacted>"},
		{"  OpenRouter.Secret/Model", "openrouter.<redacted>"},
		{"customendpoint.my-model", "customendpoint.<redacted>"},
		{"CustomEndpoint.X", "customendpoint.<redacted>"},
		{"openai.gpt-4o", "openai.gpt-4o"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(string(tt.in), func(t *testing.T) {
			if got := loggableEngineModel(tt.in); got != tt.expect {
				t.Errorf("Wrong match. expect: %q, got: %q", tt.expect, got)
			}
		})
	}
}

func Test_AICreate_h2(t *testing.T) {
	tools := []amtool.ToolName{}
	call := func(ragID uuid.UUID) func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) (any, error) {
		return func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) (any, error) {
			return h.AICreate(ctx, a, "nm", "dt", amai.TypeNormal, amai.EngineModel("openai.gpt-4o"), map[string]any{"k": "v"}, "key", "", nil, ragID, "prompt", amai.TTSType("none"), "voice", amai.STTType("none"), "en-US", tools, nil, true)
		}
	}
	rpc := h2AIRPC(func(ctx context.Context, req *requesthandler.MockRequestHandler, res *amai.AI, err error) {
		req.EXPECT().AIV1AICreate(ctx, h2CustomerID, "nm", "dt", amai.TypeNormal, amai.EngineModel("openai.gpt-4o"), map[string]any{"k": "v"}, "key", "", (*string)(nil), uuid.Nil, "prompt", amai.TTSType("none"), "voice", amai.STTType("none"), "en-US", tools, (*[]uuid.UUID)(nil), true).Return(res, err)
	})
	h2RunCases(t, h2StdCases(call(uuid.Nil), nil, rpc, h2AI(h2CustomerID).ConvertWebhookMessage()))

	admin := h2AgentIdentity(h2CustomerID, amagent.PermissionCustomerAdmin)
	h2RunCases(t, []h2Case{
		{
			name: "rag lookup fails",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				req.EXPECT().RagV1RagGet(ctx, h2RagID).Return(nil, errH2RPC)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return call(h2RagID)(ctx, h, admin)
			},
			wantErr: serviceerrors.ErrInternal,
		},
		{
			name: "rag belongs to another customer",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				req.EXPECT().RagV1RagGet(ctx, h2RagID).Return(&rmrag.Rag{ID: h2RagID, CustomerID: h2OtherCustomerID}, nil)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return call(h2RagID)(ctx, h, admin)
			},
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name: "rag owned: rpc receives rag id",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				req.EXPECT().RagV1RagGet(ctx, h2RagID).Return(&rmrag.Rag{ID: h2RagID, CustomerID: h2CustomerID}, nil)
				req.EXPECT().AIV1AICreate(ctx, h2CustomerID, "nm", "dt", amai.TypeNormal, amai.EngineModel("openai.gpt-4o"), map[string]any{"k": "v"}, "key", "", (*string)(nil), h2RagID, "prompt", amai.TTSType("none"), "voice", amai.STTType("none"), "en-US", tools, (*[]uuid.UUID)(nil), true).Return(h2AI(h2CustomerID), nil)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return call(h2RagID)(ctx, h, admin)
			},
			wantRes: h2AI(h2CustomerID).ConvertWebhookMessage(),
		},
	})
}

func Test_AIUpdate_h2(t *testing.T) {
	tools := []amtool.ToolName{}
	call := func(ragID uuid.UUID) func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) (any, error) {
		return func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) (any, error) {
			return h.AIUpdate(ctx, a, h2AIID, "nm", "dt", amai.TypeNormal, amai.EngineModel("openai.gpt-4o"), map[string]any{"k": "v"}, "key", "", nil, ragID, "prompt", amai.TTSType("none"), "voice", amai.STTType("none"), "en-US", tools, nil, true)
		}
	}
	rpc := h2AIRPC(func(ctx context.Context, req *requesthandler.MockRequestHandler, res *amai.AI, err error) {
		req.EXPECT().AIV1AIUpdate(ctx, h2AIID, "nm", "dt", amai.TypeNormal, amai.EngineModel("openai.gpt-4o"), map[string]any{"k": "v"}, "key", "", (*string)(nil), uuid.Nil, "prompt", amai.TTSType("none"), "voice", amai.STTType("none"), "en-US", tools, (*[]uuid.UUID)(nil), true).Return(res, err)
	})
	h2RunCases(t, h2StdCases(call(uuid.Nil), h2AILookup, rpc, h2AI(h2CustomerID).ConvertWebhookMessage()))

	admin := h2AgentIdentity(h2CustomerID, amagent.PermissionCustomerAdmin)
	h2RunCases(t, []h2Case{
		{
			name: "rag lookup fails",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				h2AILookup(ctx, req, h2CustomerID, false)
				req.EXPECT().RagV1RagGet(ctx, h2RagID).Return(nil, errH2RPC)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return call(h2RagID)(ctx, h, admin)
			},
			wantErr: serviceerrors.ErrInternal,
		},
		{
			name: "rag belongs to another customer",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				h2AILookup(ctx, req, h2CustomerID, false)
				req.EXPECT().RagV1RagGet(ctx, h2RagID).Return(&rmrag.Rag{ID: h2RagID, CustomerID: h2OtherCustomerID}, nil)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return call(h2RagID)(ctx, h, admin)
			},
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name: "rag owned: rpc receives rag id",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				h2AILookup(ctx, req, h2CustomerID, false)
				req.EXPECT().RagV1RagGet(ctx, h2RagID).Return(&rmrag.Rag{ID: h2RagID, CustomerID: h2CustomerID}, nil)
				req.EXPECT().AIV1AIUpdate(ctx, h2AIID, "nm", "dt", amai.TypeNormal, amai.EngineModel("openai.gpt-4o"), map[string]any{"k": "v"}, "key", "", (*string)(nil), h2RagID, "prompt", amai.TTSType("none"), "voice", amai.STTType("none"), "en-US", tools, (*[]uuid.UUID)(nil), true).Return(h2AI(h2CustomerID), nil)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return call(h2RagID)(ctx, h, admin)
			},
			wantRes: h2AI(h2CustomerID).ConvertWebhookMessage(),
		},
	})
}
