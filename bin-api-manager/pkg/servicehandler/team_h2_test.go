package servicehandler

import (
	"context"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	amteam "monorepo/bin-ai-manager/models/team"
	"monorepo/bin-api-manager/pkg/dbhandler"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
)

var (
	h2TeamID     = uuid.FromStringOrNil("a2000000-0000-0000-0000-0000000000b1")
	h2TeamOther  = uuid.FromStringOrNil("a2000000-0000-0000-0000-0000000000c9")
	h2TeamMember = uuid.FromStringOrNil("a2000000-0000-0000-0000-0000000000b2")
)

func h2Team(customerID uuid.UUID) *amteam.Team {
	return &amteam.Team{
		Identity: commonidentity.Identity{ID: h2TeamID, CustomerID: customerID},
		Name:     "team",
	}
}

func h2TeamWebhook(customerID uuid.UUID) *amteam.WebhookMessage {
	return h2Team(customerID).ConvertWebhookMessage()
}

func Test_TeamCreate(t *testing.T) {
	admin := h2AgentIdentity(h2CustomerID, amagent.PermissionCustomerAdmin)
	members := []amteam.Member{{ID: h2TeamMember}}
	param := map[string]any{"k": "v"}

	tests := []h2Case{
		{
			name: "direct access rejected",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.TeamCreate(ctx, h2DirectIdentity(h2CustomerID), "n", "d", h2TeamMember, members, param)
			},
			wantErr: serviceerrors.ErrDirectAccessNotSupported,
		},
		{
			name: "no permission",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.TeamCreate(ctx, h2AgentIdentity(h2CustomerID, amagent.PermissionNone), "n", "d", h2TeamMember, members, param)
			},
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name: "rpc error",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				req.EXPECT().AIV1TeamCreate(ctx, h2CustomerID, "n", "d", h2TeamMember, members, param).Return(nil, errH2RPC)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.TeamCreate(ctx, admin, "n", "d", h2TeamMember, members, param)
			},
			wantErr: errH2RPC,
		},
		{
			name: "success",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				req.EXPECT().AIV1TeamCreate(ctx, h2CustomerID, "n", "d", h2TeamMember, members, param).Return(h2Team(h2CustomerID), nil)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.TeamCreate(ctx, admin, "n", "d", h2TeamMember, members, param)
			},
			wantRes: h2TeamWebhook(h2CustomerID),
		},
	}
	h2RunCases(t, tests)
}

func Test_TeamDirectHashRegenerate(t *testing.T) {
	admin := h2AgentIdentity(h2CustomerID, amagent.PermissionCustomerManager)
	tests := []h2Case{
		{
			name: "direct access rejected",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.TeamDirectHashRegenerate(ctx, h2DirectIdentity(h2CustomerID), h2TeamID)
			},
			wantErr: serviceerrors.ErrDirectAccessNotSupported,
		},
		{
			name: "team lookup fails",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				req.EXPECT().AIV1TeamGet(ctx, h2TeamID).Return(nil, errH2RPC)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.TeamDirectHashRegenerate(ctx, admin, h2TeamID)
			},
			wantErr: errH2RPC,
		},
		{
			name: "cross tenant",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				req.EXPECT().AIV1TeamGet(ctx, h2TeamID).Return(h2Team(h2TeamOther), nil)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.TeamDirectHashRegenerate(ctx, admin, h2TeamID)
			},
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name: "rpc error",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				req.EXPECT().AIV1TeamGet(ctx, h2TeamID).Return(h2Team(h2CustomerID), nil)
				req.EXPECT().AIV1TeamDirectHashRegenerate(ctx, h2TeamID).Return(nil, errH2RPC)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.TeamDirectHashRegenerate(ctx, admin, h2TeamID)
			},
			wantErr: errH2RPC,
		},
		{
			name: "success",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				req.EXPECT().AIV1TeamGet(ctx, h2TeamID).Return(h2Team(h2CustomerID), nil)
				req.EXPECT().AIV1TeamDirectHashRegenerate(ctx, h2TeamID).Return(h2Team(h2CustomerID), nil)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.TeamDirectHashRegenerate(ctx, admin, h2TeamID)
			},
			wantRes: h2TeamWebhook(h2CustomerID),
		},
	}
	h2RunCases(t, tests)
}

func Test_TeamGetsByCustomerID(t *testing.T) {
	admin := h2AgentIdentity(h2CustomerID, amagent.PermissionCustomerAdmin)
	filters := map[amteam.Field]any{
		amteam.FieldDeleted:    false,
		amteam.FieldCustomerID: h2CustomerID.String(),
	}
	tests := []h2Case{
		{
			name: "direct access rejected",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.TeamGetsByCustomerID(ctx, h2DirectIdentity(h2CustomerID), 10, "tok")
			},
			wantErr: serviceerrors.ErrDirectAccessNotSupported,
		},
		{
			name: "no permission",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.TeamGetsByCustomerID(ctx, h2AgentIdentity(h2CustomerID, amagent.PermissionNone), 10, "tok")
			},
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name: "rpc error",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				req.EXPECT().AIV1TeamList(ctx, "tok", uint64(10), filters).Return(nil, errH2RPC)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.TeamGetsByCustomerID(ctx, admin, 10, "tok")
			},
			wantErr: errH2RPC,
		},
		{
			name: "success with empty token uses current time",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, util *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				util.EXPECT().TimeGetCurTime().Return("2026-01-01 00:00:00.000000")
				req.EXPECT().AIV1TeamList(ctx, "2026-01-01 00:00:00.000000", uint64(5), filters).Return([]amteam.Team{*h2Team(h2CustomerID)}, nil)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.TeamGetsByCustomerID(ctx, admin, 5, "")
			},
			wantRes: []*amteam.WebhookMessage{h2TeamWebhook(h2CustomerID)},
		},
		{
			name: "success empty list",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				req.EXPECT().AIV1TeamList(ctx, "tok", uint64(10), filters).Return([]amteam.Team{}, nil)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.TeamGetsByCustomerID(ctx, admin, 10, "tok")
			},
			wantRes: []*amteam.WebhookMessage{},
		},
	}
	h2RunCases(t, tests)
}

func Test_convertTeamFilters(t *testing.T) {
	h := &serviceHandler{}

	res, err := h.convertTeamFilters(map[string]string{"customer_id": h2CustomerID.String(), "deleted": "false"})
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if len(res) != 2 || res[amteam.FieldCustomerID] != h2CustomerID.String() || res[amteam.FieldDeleted] != false {
		t.Errorf("Wrong match. got: %v", res)
	}
}

func Test_TeamGet(t *testing.T) {
	admin := h2AgentIdentity(h2CustomerID, amagent.PermissionCustomerAdmin)
	tests := []h2Case{
		{
			name: "direct access rejected",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.TeamGet(ctx, h2DirectIdentity(h2CustomerID), h2TeamID)
			},
			wantErr: serviceerrors.ErrDirectAccessNotSupported,
		},
		{
			name: "lookup fails",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				req.EXPECT().AIV1TeamGet(ctx, h2TeamID).Return(nil, errH2RPC)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.TeamGet(ctx, admin, h2TeamID)
			},
			wantErr: errH2RPC,
		},
		{
			name: "cross tenant",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				req.EXPECT().AIV1TeamGet(ctx, h2TeamID).Return(h2Team(h2TeamOther), nil)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.TeamGet(ctx, admin, h2TeamID)
			},
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name: "success",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				req.EXPECT().AIV1TeamGet(ctx, h2TeamID).Return(h2Team(h2CustomerID), nil)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.TeamGet(ctx, admin, h2TeamID)
			},
			wantRes: h2TeamWebhook(h2CustomerID),
		},
	}
	h2RunCases(t, tests)
}

func Test_TeamDelete(t *testing.T) {
	admin := h2AgentIdentity(h2CustomerID, amagent.PermissionCustomerAdmin)
	tests := []h2Case{
		{
			name: "direct access rejected",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.TeamDelete(ctx, h2DirectIdentity(h2CustomerID), h2TeamID)
			},
			wantErr: serviceerrors.ErrDirectAccessNotSupported,
		},
		{
			name: "lookup fails",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				req.EXPECT().AIV1TeamGet(ctx, h2TeamID).Return(nil, errH2RPC)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.TeamDelete(ctx, admin, h2TeamID)
			},
			wantErr: errH2RPC,
		},
		{
			name: "cross tenant",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				req.EXPECT().AIV1TeamGet(ctx, h2TeamID).Return(h2Team(h2TeamOther), nil)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.TeamDelete(ctx, admin, h2TeamID)
			},
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name: "rpc error",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				req.EXPECT().AIV1TeamGet(ctx, h2TeamID).Return(h2Team(h2CustomerID), nil)
				req.EXPECT().AIV1TeamDelete(ctx, h2TeamID).Return(nil, errH2RPC)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.TeamDelete(ctx, admin, h2TeamID)
			},
			wantErr: errH2RPC,
		},
		{
			name: "success",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				req.EXPECT().AIV1TeamGet(ctx, h2TeamID).Return(h2Team(h2CustomerID), nil)
				req.EXPECT().AIV1TeamDelete(ctx, h2TeamID).Return(h2Team(h2CustomerID), nil)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.TeamDelete(ctx, admin, h2TeamID)
			},
			wantRes: h2TeamWebhook(h2CustomerID),
		},
	}
	h2RunCases(t, tests)
}

func Test_TeamUpdate(t *testing.T) {
	admin := h2AgentIdentity(h2CustomerID, amagent.PermissionCustomerAdmin)
	name := "new"
	detail := "det"
	start := h2TeamMember
	members := []amteam.Member{{ID: h2TeamMember}}
	param := map[string]any{"k": "v"}
	tests := []h2Case{
		{
			name: "direct access rejected",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.TeamUpdate(ctx, h2DirectIdentity(h2CustomerID), h2TeamID, &name, &detail, &start, &members, &param)
			},
			wantErr: serviceerrors.ErrDirectAccessNotSupported,
		},
		{
			name: "lookup fails",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				req.EXPECT().AIV1TeamGet(ctx, h2TeamID).Return(nil, errH2RPC)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.TeamUpdate(ctx, admin, h2TeamID, &name, &detail, &start, &members, &param)
			},
			wantErr: errH2RPC,
		},
		{
			name: "cross tenant",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				req.EXPECT().AIV1TeamGet(ctx, h2TeamID).Return(h2Team(h2TeamOther), nil)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.TeamUpdate(ctx, admin, h2TeamID, &name, &detail, &start, &members, &param)
			},
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name: "rpc error",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				req.EXPECT().AIV1TeamGet(ctx, h2TeamID).Return(h2Team(h2CustomerID), nil)
				req.EXPECT().AIV1TeamUpdate(ctx, h2TeamID, &name, &detail, &start, &members, &param).Return(nil, errH2RPC)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.TeamUpdate(ctx, admin, h2TeamID, &name, &detail, &start, &members, &param)
			},
			wantErr: errH2RPC,
		},
		{
			name: "success",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				req.EXPECT().AIV1TeamGet(ctx, h2TeamID).Return(h2Team(h2CustomerID), nil)
				req.EXPECT().AIV1TeamUpdate(ctx, h2TeamID, &name, &detail, &start, &members, &param).Return(h2Team(h2CustomerID), nil)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.TeamUpdate(ctx, admin, h2TeamID, &name, &detail, &start, &members, &param)
			},
			wantRes: h2TeamWebhook(h2CustomerID),
		},
	}
	h2RunCases(t, tests)
}
