package servicehandler

import (
	"context"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/dbhandler"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
)

var h2RingAll amagent.RingMethod = amagent.RingMethodRingAll

var h2TargetAgentID = uuid.FromStringOrNil("a2000000-0000-0000-0000-0000000000e1")

func h2TargetAgent(customerID uuid.UUID) *amagent.Agent {
	return &amagent.Agent{
		Identity: commonidentity.Identity{ID: h2TargetAgentID, CustomerID: customerID},
		Username: "target",
	}
}

// h2AgentLookup expects AgentV1AgentGet for the target agent.
func h2AgentLookup(ctx context.Context, req *requesthandler.MockRequestHandler, customerID uuid.UUID, failure bool) {
	if failure {
		req.EXPECT().AgentV1AgentGet(ctx, h2TargetAgentID).Return(nil, errH2RPC)
		return
	}
	req.EXPECT().AgentV1AgentGet(ctx, h2TargetAgentID).Return(h2TargetAgent(customerID), nil)
}

// h2AgentRPC builds an expectation helper for an RPC that returns an agent.
func h2AgentRPC(set func(ctx context.Context, req *requesthandler.MockRequestHandler, res *amagent.Agent, err error)) h2RPCFn {
	return func(ctx context.Context, req *requesthandler.MockRequestHandler, failure bool) {
		if failure {
			set(ctx, req, nil, errH2RPC)
			return
		}
		set(ctx, req, h2TargetAgent(h2CustomerID), nil)
	}
}

func Test_AgentCreate_h2(t *testing.T) {
	addrs := []commonaddress.Address{{Type: commonaddress.TypeTel, Target: "+15555550100"}}
	tags := []uuid.UUID{uuid.FromStringOrNil("a2000000-0000-0000-0000-0000000000e2")}
	h2RunCases(t, h2StdCases(
		func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) (any, error) {
			return h.AgentCreate(ctx, a, "user", "pw", "nm", "dt", h2RingAll, amagent.PermissionNone, tags, addrs)
		},
		nil,
		h2AgentRPC(func(ctx context.Context, req *requesthandler.MockRequestHandler, res *amagent.Agent, err error) {
			req.EXPECT().AgentV1AgentCreate(ctx, 30000, h2CustomerID, "user", "pw", "nm", "dt", h2RingAll, amagent.PermissionNone, tags, addrs).Return(res, err)
		}),
		h2TargetAgent(h2CustomerID).ConvertWebhookMessage(),
	))
}

func Test_AgentGet_h2(t *testing.T) {
	cases := h2StdCases(
		func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) (any, error) {
			return h.AgentGet(ctx, a, h2TargetAgentID)
		},
		h2AgentLookup,
		func(context.Context, *requesthandler.MockRequestHandler, bool) {},
		h2TargetAgent(h2CustomerID).ConvertWebhookMessage(),
	)
	// AgentGet has no further RPC: drop the rpc error row.
	filtered := cases[:0]
	for _, c := range cases {
		if c.name != "rpc error" {
			filtered = append(filtered, c)
		}
	}
	h2RunCases(t, filtered)
}

func Test_AgentGet_h2_self(t *testing.T) {
	// an agent without any permission may read itself.
	self := h2AgentIdentity(h2CustomerID, amagent.PermissionNone)
	h2RunCases(t, []h2Case{
		{
			name: "self access without permission",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				req.EXPECT().AgentV1AgentGet(ctx, h2AgentID).Return(h2AgentResp(h2CustomerID), nil)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.AgentGet(ctx, self, h2AgentID)
			},
			wantRes: h2AgentResp(h2CustomerID).ConvertWebhookMessage(),
		},
	})
}

func Test_AgentList_h2(t *testing.T) {
	admin := h2AgentIdentity(h2CustomerID, amagent.PermissionCustomerAdmin)
	tagA := uuid.FromStringOrNil("a2000000-0000-0000-0000-0000000000f1")
	list := []amagent.Agent{*h2TargetAgent(h2CustomerID)}

	h2RunCases(t, []h2Case{
		{
			name: "direct access rejected",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.AgentList(ctx, h2DirectIdentity(h2CustomerID), 10, "tok", map[string]string{})
			},
			wantErr: serviceerrors.ErrDirectAccessNotSupported,
		},
		{
			name: "no permission",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.AgentList(ctx, h2AgentIdentity(h2CustomerID, amagent.PermissionNone), 10, "tok", map[string]string{})
			},
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name: "rpc error",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				req.EXPECT().AgentV1AgentList(ctx, "tok", uint64(10), map[amagent.Field]any{}).Return(nil, errH2RPC)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.AgentList(ctx, admin, 10, "tok", map[string]string{})
			},
			wantErr: errH2RPC,
		},
		{
			name: "success with empty token and tag filter",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, util *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				util.EXPECT().TimeGetCurTime().Return("2026-01-01 00:00:00.000000")
				req.EXPECT().AgentV1AgentList(ctx, "2026-01-01 00:00:00.000000", uint64(5), map[amagent.Field]any{
					amagent.FieldTagIDs: tagA.String(),
				}).Return(list, nil)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.AgentList(ctx, admin, 5, "", map[string]string{"tag_ids": tagA.String()})
			},
			wantRes: []*amagent.WebhookMessage{list[0].ConvertWebhookMessage()},
		},
	})
}

func Test_convertAgentFilters_h2(t *testing.T) {
	h := &serviceHandler{}
	tagA := uuid.FromStringOrNil("a2000000-0000-0000-0000-0000000000f1")
	tagB := uuid.FromStringOrNil("a2000000-0000-0000-0000-0000000000f2")

	tests := []struct {
		name    string
		in      map[string]string
		wantErr bool
		want    map[amagent.Field]any
	}{
		{name: "empty tag_ids rejected", in: map[string]string{"tag_ids": ""}, wantErr: true},
		{name: "comma only tag_ids rejected", in: map[string]string{"tag_ids": ","}, wantErr: true},
		{name: "invalid uuid in tag_ids rejected", in: map[string]string{"tag_ids": "not-a-uuid"}, wantErr: true},
		{
			name: "tag_ids normalized",
			in:   map[string]string{"tag_ids": tagA.String() + "," + tagB.String()},
			want: map[amagent.Field]any{amagent.FieldTagIDs: tagA.String() + "," + tagB.String()},
		},
		{
			name: "no filters",
			in:   map[string]string{},
			want: map[amagent.Field]any{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := h.convertAgentFilters(tt.in)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Wrong match. expect: error, got: nil (%v)", res)
				}
				return
			}
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if len(res) != len(tt.want) || res[amagent.FieldTagIDs] != tt.want[amagent.FieldTagIDs] {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.want, res)
			}
		})
	}
}

func Test_AgentDelete_h2(t *testing.T) {
	h2RunCases(t, h2StdCases(
		func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) (any, error) {
			return h.AgentDelete(ctx, a, h2TargetAgentID)
		},
		h2AgentLookup,
		h2AgentRPC(func(ctx context.Context, req *requesthandler.MockRequestHandler, res *amagent.Agent, err error) {
			req.EXPECT().AgentV1AgentDelete(ctx, h2TargetAgentID).Return(res, err)
		}),
		h2TargetAgent(h2CustomerID).ConvertWebhookMessage(),
	))
}

func Test_AgentUpdate_h2(t *testing.T) {
	name, detail := "nm", "dt"
	var rm amagent.RingMethod = amagent.RingMethodRingAll
	h2RunCases(t, h2StdCases(
		func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) (any, error) {
			return h.AgentUpdate(ctx, a, h2TargetAgentID, &name, &detail, &rm)
		},
		h2AgentLookup,
		h2AgentRPC(func(ctx context.Context, req *requesthandler.MockRequestHandler, res *amagent.Agent, err error) {
			req.EXPECT().AgentV1AgentUpdate(ctx, h2TargetAgentID, &name, &detail, &rm).Return(res, err)
		}),
		h2TargetAgent(h2CustomerID).ConvertWebhookMessage(),
	))
}

func Test_AgentUpdateAddresses_h2(t *testing.T) {
	addrs := []commonaddress.Address{{Type: commonaddress.TypeTel, Target: "+15555550101"}}
	h2RunCases(t, h2StdCases(
		func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) (any, error) {
			return h.AgentUpdateAddresses(ctx, a, h2TargetAgentID, addrs)
		},
		h2AgentLookup,
		h2AgentRPC(func(ctx context.Context, req *requesthandler.MockRequestHandler, res *amagent.Agent, err error) {
			req.EXPECT().AgentV1AgentUpdateAddresses(ctx, h2TargetAgentID, addrs).Return(res, err)
		}),
		h2TargetAgent(h2CustomerID).ConvertWebhookMessage(),
	))
}

func Test_AgentUpdateTagIDs_h2(t *testing.T) {
	tags := []uuid.UUID{uuid.FromStringOrNil("a2000000-0000-0000-0000-0000000000f1")}
	h2RunCases(t, h2StdCases(
		func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) (any, error) {
			return h.AgentUpdateTagIDs(ctx, a, h2TargetAgentID, tags)
		},
		h2AgentLookup,
		h2AgentRPC(func(ctx context.Context, req *requesthandler.MockRequestHandler, res *amagent.Agent, err error) {
			req.EXPECT().AgentV1AgentUpdateTagIDs(ctx, h2TargetAgentID, tags).Return(res, err)
		}),
		h2TargetAgent(h2CustomerID).ConvertWebhookMessage(),
	))
}

func Test_AgentUpdateStatus_h2(t *testing.T) {
	h2RunCases(t, h2StdCases(
		func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) (any, error) {
			return h.AgentUpdateStatus(ctx, a, h2TargetAgentID, amagent.StatusAvailable)
		},
		h2AgentLookup,
		h2AgentRPC(func(ctx context.Context, req *requesthandler.MockRequestHandler, res *amagent.Agent, err error) {
			req.EXPECT().AgentV1AgentUpdateStatus(ctx, h2TargetAgentID, amagent.StatusAvailable).Return(res, err)
		}),
		h2TargetAgent(h2CustomerID).ConvertWebhookMessage(),
	))
}

func Test_AgentUpdatePassword_h2(t *testing.T) {
	h2RunCases(t, h2StdCases(
		func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) (any, error) {
			return h.AgentUpdatePassword(ctx, a, h2TargetAgentID, "newpw")
		},
		h2AgentLookup,
		h2AgentRPC(func(ctx context.Context, req *requesthandler.MockRequestHandler, res *amagent.Agent, err error) {
			req.EXPECT().AgentV1AgentUpdatePassword(ctx, 30000, h2TargetAgentID, "newpw").Return(res, err)
		}),
		h2TargetAgent(h2CustomerID).ConvertWebhookMessage(),
	))
}

func Test_AgentDirectHashRegenerate_h2(t *testing.T) {
	h2RunCases(t, h2StdCases(
		func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) (any, error) {
			return h.AgentDirectHashRegenerate(ctx, a, h2TargetAgentID)
		},
		h2AgentLookup,
		h2AgentRPC(func(ctx context.Context, req *requesthandler.MockRequestHandler, res *amagent.Agent, err error) {
			req.EXPECT().AgentV1AgentDirectHashRegenerate(ctx, h2TargetAgentID).Return(res, err)
		}),
		h2TargetAgent(h2CustomerID).ConvertWebhookMessage(),
	))
}

func Test_AgentUpdatePermission_h2(t *testing.T) {
	admin := h2AgentIdentity(h2CustomerID, amagent.PermissionCustomerAdmin)
	superAdmin := h2AgentIdentity(h2CustomerID, amagent.PermissionProjectSuperAdmin)
	lookupOK := func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
		h2AgentLookup(ctx, req, h2CustomerID, false)
	}
	res := h2TargetAgent(h2CustomerID)

	h2RunCases(t, []h2Case{
		{
			name: "direct access rejected",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.AgentUpdatePermission(ctx, h2DirectIdentity(h2CustomerID), h2TargetAgentID, amagent.PermissionCustomerAdmin)
			},
			wantErr: serviceerrors.ErrDirectAccessNotSupported,
		},
		{
			name: "lookup fails",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				h2AgentLookup(ctx, req, h2CustomerID, true)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.AgentUpdatePermission(ctx, admin, h2TargetAgentID, amagent.PermissionCustomerAdmin)
			},
			wantErr: errH2RPC,
		},
		{
			name: "customer level: cross tenant",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				h2AgentLookup(ctx, req, h2OtherCustomerID, false)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.AgentUpdatePermission(ctx, admin, h2TargetAgentID, amagent.PermissionCustomerAdmin)
			},
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name:  "project level: customer admin may not set",
			setup: lookupOK,
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.AgentUpdatePermission(ctx, admin, h2TargetAgentID, amagent.PermissionProjectSuperAdmin)
			},
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name: "project level: rpc error for super admin",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				h2AgentLookup(ctx, req, h2CustomerID, false)
				req.EXPECT().AgentV1AgentUpdatePermission(ctx, h2TargetAgentID, amagent.PermissionProjectSuperAdmin).Return(nil, errH2RPC)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.AgentUpdatePermission(ctx, superAdmin, h2TargetAgentID, amagent.PermissionProjectSuperAdmin)
			},
			wantErr: errH2RPC,
		},
		{
			name: "customer level: success",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				h2AgentLookup(ctx, req, h2CustomerID, false)
				req.EXPECT().AgentV1AgentUpdatePermission(ctx, h2TargetAgentID, amagent.PermissionCustomerManager).Return(res, nil)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.AgentUpdatePermission(ctx, admin, h2TargetAgentID, amagent.PermissionCustomerManager)
			},
			wantRes: res.ConvertWebhookMessage(),
		},
	})
}
