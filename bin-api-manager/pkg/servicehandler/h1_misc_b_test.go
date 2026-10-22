package servicehandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-api-manager/pkg/serviceerrors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	fmaction "monorepo/bin-flow-manager/models/action"
	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"
	fmflow "monorepo/bin-flow-manager/models/flow"
	rmsipauth "monorepo/bin-registrar-manager/models/sipauth"
	rmtrunk "monorepo/bin-registrar-manager/models/trunk"
	rmroute "monorepo/bin-route-manager/models/route"
)

// ---- route ----

func h1Route() *rmroute.Route {
	return &rmroute.Route{
		ID:         h1UUID(h1ResourceID),
		CustomerID: h1UUID(h1CustomerID),
		Name:       "r",
	}
}

// h1RouteSuperCases builds the table for route operations that are limited to
// the project super admin; the permission is checked before any RPC.
func h1RouteSuperCases(
	withGet bool,
	call func(ctx context.Context, h *serviceHandler, a *h1Identity) (any, error),
	expectAct func(m *h1Mocks, ret *rmroute.Route, err error),
) []h1Case {
	id := h1UUID(h1ResourceID)
	getOK := func(m *h1Mocks) {
		if withGet {
			m.req.EXPECT().RouteV1RouteGet(h1Ctx(), id).Return(h1Route(), nil)
		}
	}

	tests := []h1Case{
		{
			name:    "direct access",
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return call(ctx, h, h1Direct()) },
			wantErr: serviceerrors.ErrDirectAccessNotSupported,
		},
		{
			name:    "customer admin is not super admin",
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return call(ctx, h, h1Admin()) },
			wantErr: serviceerrors.ErrPermissionDenied,
		},
	}
	if withGet {
		tests = append(tests, h1Case{
			name:    "get error",
			setup:   func(m *h1Mocks) { m.req.EXPECT().RouteV1RouteGet(h1Ctx(), id).Return(nil, errH1RPC) },
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return call(ctx, h, h1Super()) },
			wantErr: errH1RPC,
		})
	}
	tests = append(tests,
		h1Case{
			name:    "action rpc error",
			setup:   func(m *h1Mocks) { getOK(m); expectAct(m, nil, errH1RPC) },
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return call(ctx, h, h1Super()) },
			wantErr: errH1RPC,
		},
		h1Case{
			name:    "success",
			setup:   func(m *h1Mocks) { getOK(m); expectAct(m, h1Route(), nil) },
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return call(ctx, h, h1Super()) },
			wantRes: h1Route(),
		},
	)

	return tests
}

func Test_RouteGet_Matrix(t *testing.T) {
	id := h1UUID(h1ResourceID)
	h1Run(t, h1RouteSuperCases(false,
		func(ctx context.Context, h *serviceHandler, a *h1Identity) (any, error) {
			return h.RouteGet(ctx, a, id)
		},
		func(m *h1Mocks, ret *rmroute.Route, err error) {
			m.req.EXPECT().RouteV1RouteGet(h1Ctx(), id).Return(ret, err)
		}))
}

func Test_RouteDelete_Matrix(t *testing.T) {
	id := h1UUID(h1ResourceID)
	h1Run(t, h1RouteSuperCases(true,
		func(ctx context.Context, h *serviceHandler, a *h1Identity) (any, error) {
			return h.RouteDelete(ctx, a, id)
		},
		func(m *h1Mocks, ret *rmroute.Route, err error) {
			m.req.EXPECT().RouteV1RouteDelete(h1Ctx(), id).Return(ret, err)
		}))
}

func Test_RouteUpdate_Matrix(t *testing.T) {
	id := h1UUID(h1ResourceID)
	name, detail, target := "n", "d", "+1"
	provider := h1UUID(h1AgentID)
	prio := 7
	h1Run(t, h1RouteSuperCases(true,
		func(ctx context.Context, h *serviceHandler, a *h1Identity) (any, error) {
			return h.RouteUpdate(ctx, a, id, &name, &detail, &provider, &prio, &target)
		},
		func(m *h1Mocks, ret *rmroute.Route, err error) {
			m.req.EXPECT().RouteV1RouteUpdate(h1Ctx(), id, &name, &detail, &provider, &prio, &target).Return(ret, err)
		}))
}

func Test_RouteCreate_Matrix(t *testing.T) {
	cid := h1UUID(h1ResourceID)
	provider := h1UUID(h1AgentID)
	h1Run(t, h1RouteSuperCases(false,
		func(ctx context.Context, h *serviceHandler, a *h1Identity) (any, error) {
			return h.RouteCreate(ctx, a, cid, "n", "d", provider, 3, "+1")
		},
		func(m *h1Mocks, ret *rmroute.Route, err error) {
			m.req.EXPECT().RouteV1RouteCreate(h1Ctx(), cid, "n", "d", provider, 3, "+1").Return(ret, err)
		}))
}

func Test_RouteList_Matrix(t *testing.T) {
	empty := map[rmroute.Field]any{}
	byCustomer := map[rmroute.Field]any{rmroute.FieldCustomerID: h1UUID(h1OtherID)}
	cust := h1UUID(h1OtherID)

	h1Run(t, []h1Case{
		{
			name:    "list direct access",
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return h.RouteList(ctx, h1Direct(), 5, "t") },
			wantErr: serviceerrors.ErrDirectAccessNotSupported,
		},
		{
			name:    "list customer admin is not super admin",
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return h.RouteList(ctx, h1Admin(), 5, "t") },
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name: "list empty token and rpc error",
			setup: func(m *h1Mocks) {
				m.util.EXPECT().TimeGetCurTime().Return("2026-01-01T00:00:00.000000Z")
				m.req.EXPECT().RouteV1RouteList(h1Ctx(), "2026-01-01T00:00:00.000000Z", uint64(5), empty).Return(nil, errH1RPC)
			},
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return h.RouteList(ctx, h1Super(), 5, "") },
			wantErr: errH1RPC,
		},
		{
			name: "list success",
			setup: func(m *h1Mocks) {
				m.req.EXPECT().RouteV1RouteList(h1Ctx(), "tok", uint64(5), empty).Return([]rmroute.Route{*h1Route()}, nil)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.RouteList(ctx, h1Super(), 5, "tok")
			},
			wantRes: []*rmroute.Route{h1Route()},
		},
		{
			name: "by customer direct access",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.RouteGetsByCustomerID(ctx, h1Direct(), cust, 5, "t")
			},
			wantErr: serviceerrors.ErrDirectAccessNotSupported,
		},
		{
			name: "by customer admin is not super admin",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.RouteGetsByCustomerID(ctx, h1Admin(), cust, 5, "t")
			},
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name: "by customer empty token and rpc error",
			setup: func(m *h1Mocks) {
				m.util.EXPECT().TimeGetCurTime().Return("2026-01-01T00:00:00.000000Z")
				m.req.EXPECT().RouteV1RouteList(h1Ctx(), "2026-01-01T00:00:00.000000Z", uint64(5), byCustomer).Return(nil, errH1RPC)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.RouteGetsByCustomerID(ctx, h1Super(), cust, 5, "")
			},
			wantErr: errH1RPC,
		},
		{
			name: "by customer success",
			setup: func(m *h1Mocks) {
				m.req.EXPECT().RouteV1RouteList(h1Ctx(), "tok", uint64(5), byCustomer).Return([]rmroute.Route{*h1Route()}, nil)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.RouteGetsByCustomerID(ctx, h1Super(), cust, 5, "tok")
			},
			wantRes: []*rmroute.Route{h1Route()},
		},
	})
}

// ---- trunk ----

func h1Trunk(customerID string) *rmtrunk.Trunk {
	return &rmtrunk.Trunk{
		Identity: commonidentity.Identity{
			ID:         h1UUID(h1ResourceID),
			CustomerID: h1UUID(customerID),
		},
		Name: "t",
	}
}

func h1TrunkSpec(
	call func(ctx context.Context, h *serviceHandler, a *h1Identity) (any, error),
	actExpect func(m *h1Mocks, ret *rmtrunk.Trunk, err error),
) h1ActSpec {
	id := h1UUID(h1ResourceID)
	spec := h1ActSpec{
		call:    call,
		getOK:   func(m *h1Mocks, c string) { m.req.EXPECT().RegistrarV1TrunkGet(h1Ctx(), id).Return(h1Trunk(c), nil) },
		getErr:  func(m *h1Mocks, err error) { m.req.EXPECT().RegistrarV1TrunkGet(h1Ctx(), id).Return(nil, err) },
		okAgent: h1Manager(),
		want:    h1Trunk(h1CustomerID).ConvertWebhookMessage(),
	}
	if actExpect != nil {
		spec.actExpect = func(m *h1Mocks, c string, err error) {
			if err != nil {
				actExpect(m, nil, err)
				return
			}
			actExpect(m, h1Trunk(c), nil)
		}
	}
	return spec
}

func Test_TrunkDelete_Matrix(t *testing.T) {
	id := h1UUID(h1ResourceID)
	h1Run(t, h1ActCases(h1TrunkSpec(
		func(ctx context.Context, h *serviceHandler, a *h1Identity) (any, error) {
			return h.TrunkDelete(ctx, a, id)
		},
		func(m *h1Mocks, ret *rmtrunk.Trunk, err error) {
			m.req.EXPECT().RegistrarV1TrunkDelete(h1Ctx(), id).Return(ret, err)
		})))
}

func Test_TrunkUpdateBasicInfo_Matrix(t *testing.T) {
	id := h1UUID(h1ResourceID)
	name, detail, user, pass := "n", "d", "u", "p"
	auths := []rmsipauth.AuthType{rmsipauth.AuthTypeBasic}
	ips := []string{"allowed-ip-1"}
	h1Run(t, h1ActCases(h1TrunkSpec(
		func(ctx context.Context, h *serviceHandler, a *h1Identity) (any, error) {
			return h.TrunkUpdateBasicInfo(ctx, a, id, &name, &detail, &auths, &user, &pass, &ips)
		},
		func(m *h1Mocks, ret *rmtrunk.Trunk, err error) {
			m.req.EXPECT().RegistrarV1TrunkUpdateBasicInfo(h1Ctx(), id, &name, &detail, &auths, &user, &pass, &ips).Return(ret, err)
		})))
}

func Test_TrunkGet_Matrix(t *testing.T) {
	id := h1UUID(h1ResourceID)
	h1Run(t, h1ActCases(h1TrunkSpec(
		func(ctx context.Context, h *serviceHandler, a *h1Identity) (any, error) {
			return h.TrunkGet(ctx, a, id)
		}, nil)))
}

func Test_TrunkCreate_Guards(t *testing.T) {
	cid := h1UUID(h1CustomerID)
	auths := []rmsipauth.AuthType{rmsipauth.AuthTypeBasic}
	ips := []string{"allowed-ip-1"}
	create := func(ctx context.Context, h *serviceHandler, a *h1Identity) (any, error) {
		return h.TrunkCreate(ctx, a, "n", "d", "trunk.example.com", auths, "u", "p", ips)
	}

	h1Run(t, []h1Case{
		{
			name:    "direct access",
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return create(ctx, h, h1Direct()) },
			wantErr: serviceerrors.ErrDirectAccessNotSupported,
		},
		{
			name:    "no permission",
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return create(ctx, h, h1NoPerm()) },
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name: "rpc error",
			setup: func(m *h1Mocks) {
				m.req.EXPECT().RegistrarV1TrunkCreate(h1Ctx(), cid, "n", "d", "trunk.example.com", auths, "u", "p", ips).Return(nil, errH1RPC)
			},
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return create(ctx, h, h1Admin()) },
			wantErr: errH1RPC,
		},
	})
}

func Test_TrunkList_Matrix(t *testing.T) {
	filters := map[rmtrunk.Field]any{
		rmtrunk.FieldCustomerID: h1UUID(h1CustomerID),
		rmtrunk.FieldDeleted:    false,
	}
	h1Run(t, []h1Case{
		{
			name:    "direct access",
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return h.TrunkList(ctx, h1Direct(), 5, "t") },
			wantErr: serviceerrors.ErrDirectAccessNotSupported,
		},
		{
			name:    "no permission",
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return h.TrunkList(ctx, h1NoPerm(), 5, "t") },
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name: "empty token and rpc error",
			setup: func(m *h1Mocks) {
				m.util.EXPECT().TimeGetCurTime().Return("2026-01-01T00:00:00.000000Z")
				m.req.EXPECT().RegistrarV1TrunkList(h1Ctx(), "2026-01-01T00:00:00.000000Z", uint64(5), filters).Return(nil, errH1RPC)
			},
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return h.TrunkList(ctx, h1Admin(), 5, "") },
			wantErr: errH1RPC,
		},
		{
			name: "success",
			setup: func(m *h1Mocks) {
				m.req.EXPECT().RegistrarV1TrunkList(h1Ctx(), "tok", uint64(5), filters).Return([]rmtrunk.Trunk{*h1Trunk(h1CustomerID)}, nil)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.TrunkList(ctx, h1Manager(), 5, "tok")
			},
			wantRes: []*rmtrunk.WebhookMessage{h1Trunk(h1CustomerID).ConvertWebhookMessage()},
		},
	})
}

func Test_convertTrunkFilters_Invalid(t *testing.T) {
	h := &serviceHandler{}

	res, err := h.convertTrunkFilters(map[string]string{"customer_id": "not-a-uuid"})
	if err == nil {
		t.Fatalf("Wrong match. expect: error, got: nil (res: %v)", res)
	}
	if res != nil {
		t.Errorf("Wrong result. expect: nil, got: %v", res)
	}
}

// ---- activeflows ----

func h1Activeflow(customerID string) *fmactiveflow.Activeflow {
	return &fmactiveflow.Activeflow{
		Identity: commonidentity.Identity{
			ID:         h1UUID(h1ResourceID),
			CustomerID: h1UUID(customerID),
		},
	}
}

func h1ActiveflowSpec(
	call func(ctx context.Context, h *serviceHandler, a *h1Identity) (any, error),
	actExpect func(m *h1Mocks, ret *fmactiveflow.Activeflow, err error),
) h1ActSpec {
	id := h1UUID(h1ResourceID)
	now := h1Now()
	spec := h1ActSpec{
		call: call,
		getOK: func(m *h1Mocks, c string) {
			m.req.EXPECT().FlowV1ActiveflowGet(h1Ctx(), id).Return(h1Activeflow(c), nil)
		},
		getErr: func(m *h1Mocks, err error) { m.req.EXPECT().FlowV1ActiveflowGet(h1Ctx(), id).Return(nil, err) },
		getDeleted: func(m *h1Mocks) {
			af := h1Activeflow(h1CustomerID)
			af.TMDelete = &now
			m.req.EXPECT().FlowV1ActiveflowGet(h1Ctx(), id).Return(af, nil)
		},
		okAgent: h1Manager(),
		want:    h1Activeflow(h1CustomerID).ConvertWebhookMessage(),
	}
	if actExpect != nil {
		spec.actExpect = func(m *h1Mocks, c string, err error) {
			if err != nil {
				actExpect(m, nil, err)
				return
			}
			actExpect(m, h1Activeflow(c), nil)
		}
	}
	return spec
}

func Test_ActiveflowGet_Matrix(t *testing.T) {
	id := h1UUID(h1ResourceID)
	h1Run(t, h1ActCases(h1ActiveflowSpec(
		func(ctx context.Context, h *serviceHandler, a *h1Identity) (any, error) {
			return h.ActiveflowGet(ctx, a, id)
		}, nil)))
}

func Test_ActiveflowStop_Matrix(t *testing.T) {
	id := h1UUID(h1ResourceID)
	h1Run(t, h1ActCases(h1ActiveflowSpec(
		func(ctx context.Context, h *serviceHandler, a *h1Identity) (any, error) {
			return h.ActiveflowStop(ctx, a, id)
		},
		func(m *h1Mocks, ret *fmactiveflow.Activeflow, err error) {
			m.req.EXPECT().FlowV1ActiveflowStop(h1Ctx(), id).Return(ret, err)
		})))
}

func Test_ActiveflowDelete_Matrix(t *testing.T) {
	id := h1UUID(h1ResourceID)
	h1Run(t, h1ActCases(h1ActiveflowSpec(
		func(ctx context.Context, h *serviceHandler, a *h1Identity) (any, error) {
			return h.ActiveflowDelete(ctx, a, id)
		},
		func(m *h1Mocks, ret *fmactiveflow.Activeflow, err error) {
			m.req.EXPECT().FlowV1ActiveflowDelete(h1Ctx(), id).Return(ret, err)
		})))
}

func Test_ActiveflowList_Matrix(t *testing.T) {
	filters := map[fmactiveflow.Field]any{
		fmactiveflow.FieldCustomerID: h1UUID(h1CustomerID),
		fmactiveflow.FieldDeleted:    false,
	}
	h1Run(t, []h1Case{
		{
			name: "direct access",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.ActiveflowList(ctx, h1Direct(), 5, "t")
			},
			wantErr: serviceerrors.ErrDirectAccessNotSupported,
		},
		{
			name: "no permission",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.ActiveflowList(ctx, h1NoPerm(), 5, "t")
			},
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name: "empty token and rpc error",
			setup: func(m *h1Mocks) {
				m.util.EXPECT().TimeGetCurTime().Return("2026-01-01T00:00:00.000000Z")
				m.req.EXPECT().FlowV1ActiveflowList(h1Ctx(), "2026-01-01T00:00:00.000000Z", uint64(5), filters).Return(nil, errH1RPC)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.ActiveflowList(ctx, h1Admin(), 5, "")
			},
			wantErr: errH1RPC,
		},
		{
			name: "success",
			setup: func(m *h1Mocks) {
				m.req.EXPECT().FlowV1ActiveflowList(h1Ctx(), "tok", uint64(5), filters).Return([]fmactiveflow.Activeflow{*h1Activeflow(h1CustomerID)}, nil)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.ActiveflowList(ctx, h1Manager(), 5, "tok")
			},
			wantRes: []*fmactiveflow.WebhookMessage{h1Activeflow(h1CustomerID).ConvertWebhookMessage()},
		},
	})
}

func Test_ActiveflowCreate_Matrix(t *testing.T) {
	cid := h1UUID(h1CustomerID)
	afID := h1UUID(h1AgentID)
	flowID := h1UUID("6a1f0b10-0000-4000-8000-0000000000f1")
	newAFID := h1UUID("6a1f0b10-0000-4000-8000-0000000000f2")
	vars := map[string]string{"k": "v"}
	actions := []fmaction.Action{{Type: fmaction.TypeAnswer}}
	uri := "https://example.com/hook"
	method := fmactiveflow.WebhookMethod("POST")

	create := func(ctx context.Context, h *serviceHandler, a *h1Identity, af, f uuid.UUID) (any, error) {
		return h.ActiveflowCreate(ctx, a, af, f, actions, vars, uri, method)
	}
	expectFlow := func(m *h1Mocks, customer string) {
		m.req.EXPECT().FlowV1FlowGet(h1Ctx(), flowID).Return(h1Flow("6a1f0b10-0000-4000-8000-0000000000f1", customer), nil)
	}
	expectCreate := func(m *h1Mocks, af uuid.UUID, ret *fmactiveflow.Activeflow, err error) {
		m.req.EXPECT().FlowV1ActiveflowCreate(h1Ctx(), af, cid, flowID, fmactiveflow.ReferenceTypeAPI, uuid.Nil, uuid.Nil, vars, uri, method).Return(ret, err)
	}

	h1Run(t, []h1Case{
		{
			name: "direct access",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return create(ctx, h, h1Direct(), afID, flowID)
			},
			wantErr: serviceerrors.ErrDirectAccessNotSupported,
		},
		{
			name: "no permission",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return create(ctx, h, h1NoPerm(), afID, flowID)
			},
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name:  "flow lookup error",
			setup: func(m *h1Mocks) { m.req.EXPECT().FlowV1FlowGet(h1Ctx(), flowID).Return(nil, errH1RPC) },
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return create(ctx, h, h1Admin(), afID, flowID)
			},
			wantErr: errH1RPC,
		},
		{
			name:  "flow of another customer",
			setup: func(m *h1Mocks) { expectFlow(m, h1OtherID) },
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return create(ctx, h, h1Admin(), afID, flowID)
			},
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name: "create rpc error",
			setup: func(m *h1Mocks) {
				expectFlow(m, h1CustomerID)
				expectCreate(m, afID, nil, errH1RPC)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return create(ctx, h, h1Admin(), afID, flowID)
			},
			wantErr: errH1RPC,
		},
		{
			name: "execute rpc error",
			setup: func(m *h1Mocks) {
				expectFlow(m, h1CustomerID)
				expectCreate(m, afID, h1Activeflow(h1CustomerID), nil)
				m.req.EXPECT().FlowV1ActiveflowExecute(h1Ctx(), h1UUID(h1ResourceID)).Return(errH1RPC)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return create(ctx, h, h1Admin(), afID, flowID)
			},
			wantErr: errH1RPC,
		},
		{
			name: "success",
			setup: func(m *h1Mocks) {
				expectFlow(m, h1CustomerID)
				expectCreate(m, afID, h1Activeflow(h1CustomerID), nil)
				m.req.EXPECT().FlowV1ActiveflowExecute(h1Ctx(), h1UUID(h1ResourceID)).Return(nil)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return create(ctx, h, h1Manager(), afID, flowID)
			},
			wantRes: h1Activeflow(h1CustomerID).ConvertWebhookMessage(),
		},
		{
			name: "nil activeflow id is generated",
			setup: func(m *h1Mocks) {
				m.util.EXPECT().UUIDCreate().Return(newAFID)
				expectFlow(m, h1CustomerID)
				expectCreate(m, newAFID, h1Activeflow(h1CustomerID), nil)
				m.req.EXPECT().FlowV1ActiveflowExecute(h1Ctx(), h1UUID(h1ResourceID)).Return(nil)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return create(ctx, h, h1Admin(), uuid.Nil, flowID)
			},
			wantRes: h1Activeflow(h1CustomerID).ConvertWebhookMessage(),
		},
		{
			name: "nil flow id creates a temporary flow",
			setup: func(m *h1Mocks) {
				m.req.EXPECT().FlowV1FlowCreate(h1Ctx(), cid, fmflow.TypeFlow, "tmp", "tmp outbound flow", actions, uuid.Nil, false).Return(h1Flow("6a1f0b10-0000-4000-8000-0000000000f1", h1CustomerID), nil)
				expectFlow(m, h1CustomerID)
				expectCreate(m, afID, h1Activeflow(h1CustomerID), nil)
				m.req.EXPECT().FlowV1ActiveflowExecute(h1Ctx(), h1UUID(h1ResourceID)).Return(nil)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return create(ctx, h, h1Admin(), afID, uuid.Nil)
			},
			wantRes: h1Activeflow(h1CustomerID).ConvertWebhookMessage(),
		},
		{
			name: "temporary flow creation failure",
			setup: func(m *h1Mocks) {
				m.req.EXPECT().FlowV1FlowCreate(h1Ctx(), cid, fmflow.TypeFlow, "tmp", "tmp outbound flow", actions, uuid.Nil, false).Return(nil, errH1RPC)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return create(ctx, h, h1Admin(), afID, uuid.Nil)
			},
			wantErr: errH1RPC,
		},
	})
}
