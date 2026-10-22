package servicehandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"

	commonidentity "monorepo/bin-common-handler/models/identity"
	cscustomer "monorepo/bin-customer-manager/models/customer"
	nmnumber "monorepo/bin-number-manager/models/number"

	"monorepo/bin-api-manager/pkg/serviceerrors"
)

func h1Number(customerID string) *nmnumber.Number {
	return &nmnumber.Number{
		Identity: commonidentity.Identity{
			ID:         h1UUID(h1ResourceID),
			CustomerID: h1UUID(customerID),
		},
		Number: "+15555550100",
	}
}

func h1NumberWebhook(customerID string) *nmnumber.WebhookMessage {
	return &nmnumber.WebhookMessage{
		Identity: commonidentity.Identity{
			ID:         h1UUID(h1ResourceID),
			CustomerID: h1UUID(customerID),
		},
		Number: "+15555550100",
	}
}

func h1NumberSpec(
	call func(ctx context.Context, h *serviceHandler, a *h1Identity) (any, error),
	actExpect func(m *h1Mocks, ret *nmnumber.Number, err error),
) h1ActSpec {
	id := h1UUID(h1ResourceID)
	return h1ActSpec{
		call:   call,
		getOK:  func(m *h1Mocks, c string) { m.req.EXPECT().NumberV1NumberGet(h1Ctx(), id).Return(h1Number(c), nil) },
		getErr: func(m *h1Mocks, err error) { m.req.EXPECT().NumberV1NumberGet(h1Ctx(), id).Return(nil, err) },
		getDeleted: func(m *h1Mocks) {
			n := h1Number(h1CustomerID)
			now := h1Now()
			n.TMDelete = &now
			m.req.EXPECT().NumberV1NumberGet(h1Ctx(), id).Return(n, nil)
		},
		actExpect: func(m *h1Mocks, c string, err error) {
			if err != nil {
				actExpect(m, nil, err)
				return
			}
			actExpect(m, h1Number(c), nil)
		},
		okAgent:          h1Manager(),
		deniedSameTenant: h1NoPerm(),
		want:             h1NumberWebhook(h1CustomerID),
	}
}

func Test_numberGet_Paths(t *testing.T) {
	id := h1UUID(h1ResourceID)
	now := h1Now()

	h1Run(t, []h1Case{
		{
			name:    "rpc error",
			setup:   func(m *h1Mocks) { m.req.EXPECT().NumberV1NumberGet(h1Ctx(), id).Return(nil, errH1RPC) },
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return h.numberGet(ctx, id) },
			wantErr: errH1RPC,
		},
		{
			name: "deleted number",
			setup: func(m *h1Mocks) {
				n := h1Number(h1CustomerID)
				n.TMDelete = &now
				m.req.EXPECT().NumberV1NumberGet(h1Ctx(), id).Return(n, nil)
			},
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return h.numberGet(ctx, id) },
			wantErr: serviceerrors.ErrNotFound,
		},
	})
}

func Test_NumberList_Matrix(t *testing.T) {
	filters := map[nmnumber.Field]any{
		nmnumber.FieldCustomerID: h1UUID(h1CustomerID),
		nmnumber.FieldDeleted:    false,
	}

	h1Run(t, []h1Case{
		{
			name: "direct access",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.NumberList(ctx, h1Direct(), 10, "t")
			},
			wantErr: serviceerrors.ErrDirectAccessNotSupported,
		},
		{
			name: "no permission",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.NumberList(ctx, h1NoPerm(), 10, "t")
			},
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name: "empty token and rpc error",
			setup: func(m *h1Mocks) {
				m.util.EXPECT().TimeGetCurTime().Return("2026-01-01T00:00:00.000000Z")
				m.req.EXPECT().NumberV1NumberList(h1Ctx(), "2026-01-01T00:00:00.000000Z", uint64(9), filters).Return(nil, errH1RPC)
			},
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return h.NumberList(ctx, h1Admin(), 9, "") },
			wantErr: errH1RPC,
		},
		{
			name: "success",
			setup: func(m *h1Mocks) {
				m.req.EXPECT().NumberV1NumberList(h1Ctx(), "tok", uint64(9), filters).Return([]nmnumber.Number{*h1Number(h1CustomerID)}, nil)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.NumberList(ctx, h1Manager(), 9, "tok")
			},
			wantRes: []*nmnumber.WebhookMessage{h1NumberWebhook(h1CustomerID)},
		},
	})
}

func Test_NumberCreate_Matrix(t *testing.T) {
	cid := h1UUID(h1CustomerID)
	cf := h1UUID("6a1f0b10-0000-4000-8000-0000000000f1")
	mf := h1UUID("6a1f0b10-0000-4000-8000-0000000000f2")
	create := func(ctx context.Context, h *serviceHandler, a *h1Identity, num string, typ nmnumber.Type) (any, error) {
		return h.NumberCreate(ctx, a, num, typ, cf, mf, "n", "d")
	}
	customer := func(status cscustomer.IdentityVerificationStatus) *cscustomer.Customer {
		return &cscustomer.Customer{ID: cid, IdentityVerificationStatus: status}
	}

	h1Run(t, []h1Case{
		{
			name: "direct access",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return create(ctx, h, h1Direct(), "+15555550100", nmnumber.TypeVirtual)
			},
			wantErr: serviceerrors.ErrDirectAccessNotSupported,
		},
		{
			name: "empty number",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return create(ctx, h, h1Admin(), "", nmnumber.TypeVirtual)
			},
			wantErr: serviceerrors.ErrInvalidArgument,
		},
		{
			name: "no permission",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return create(ctx, h, h1NoPerm(), "+15555550100", nmnumber.TypeVirtual)
			},
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name:  "customer lookup error is reported as internal error",
			setup: func(m *h1Mocks) { m.req.EXPECT().CustomerV1CustomerGet(h1Ctx(), cid).Return(nil, errH1RPC) },
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return create(ctx, h, h1Admin(), "+15555550100", nmnumber.TypeNormal)
			},
			wantErr: serviceerrors.ErrInternal,
		},
		{
			name: "unverified identity cannot purchase a normal number",
			setup: func(m *h1Mocks) {
				m.req.EXPECT().CustomerV1CustomerGet(h1Ctx(), cid).Return(customer(cscustomer.IdentityVerificationStatusNone), nil)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return create(ctx, h, h1Admin(), "+15555550100", nmnumber.TypeNormal)
			},
			wantErr: serviceerrors.ErrIdentityVerificationRequired,
		},
		{
			name: "verified identity buys a normal number",
			setup: func(m *h1Mocks) {
				m.req.EXPECT().CustomerV1CustomerGet(h1Ctx(), cid).Return(customer(cscustomer.IdentityVerificationStatusVerified), nil)
				m.req.EXPECT().NumberV1NumberCreate(h1Ctx(), cid, "+15555550100", nmnumber.TypeNormal, cf, mf, "n", "d").Return(h1Number(h1CustomerID), nil)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return create(ctx, h, h1Admin(), "+15555550100", nmnumber.TypeNormal)
			},
			wantRes: h1NumberWebhook(h1CustomerID),
		},
		{
			name: "virtual number skips the identity check, rpc error",
			setup: func(m *h1Mocks) {
				m.req.EXPECT().NumberV1NumberCreate(h1Ctx(), cid, "+15555550100", nmnumber.TypeVirtual, cf, mf, "n", "d").Return(nil, errH1RPC)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return create(ctx, h, h1Manager(), "+15555550100", nmnumber.TypeVirtual)
			},
			wantErr: errH1RPC,
		},
	})
}

func Test_NumberGet_Matrix(t *testing.T) {
	id := h1UUID(h1ResourceID)
	spec := h1NumberSpec(
		func(ctx context.Context, h *serviceHandler, a *h1Identity) (any, error) {
			return h.NumberGet(ctx, a, id)
		},
		func(m *h1Mocks, ret *nmnumber.Number, err error) {},
	)
	// NumberGet performs no action RPC: only the lookup cases apply.
	cases := h1ActCases(spec)
	var picked []h1Case
	for _, c := range cases {
		if c.name == "direct access" || c.name == "get error" || c.name == "deleted resource is not found" || c.name == "no permission" || c.name == "cross tenant" {
			picked = append(picked, c)
		}
	}
	picked = append(picked, h1Case{
		name:    "success",
		setup:   func(m *h1Mocks) { spec.getOK(m, h1CustomerID) },
		call:    func(ctx context.Context, h *serviceHandler) (any, error) { return h.NumberGet(ctx, h1Manager(), id) },
		wantRes: h1NumberWebhook(h1CustomerID),
	})
	h1Run(t, picked)
}

func Test_NumberDelete_Matrix(t *testing.T) {
	id := h1UUID(h1ResourceID)
	h1Run(t, h1ActCases(h1NumberSpec(
		func(ctx context.Context, h *serviceHandler, a *h1Identity) (any, error) {
			return h.NumberDelete(ctx, a, id)
		},
		func(m *h1Mocks, ret *nmnumber.Number, err error) {
			m.req.EXPECT().NumberV1NumberDelete(h1Ctx(), id).Return(ret, err)
		},
	)))
}

func Test_NumberUpdateFlowIDs_Matrix(t *testing.T) {
	id := h1UUID(h1ResourceID)
	cf := h1UUID("6a1f0b10-0000-4000-8000-0000000000f1")
	mf := h1UUID("6a1f0b10-0000-4000-8000-0000000000f2")
	h1Run(t, h1ActCases(h1NumberSpec(
		func(ctx context.Context, h *serviceHandler, a *h1Identity) (any, error) {
			return h.NumberUpdateFlowIDs(ctx, a, id, &cf, &mf)
		},
		func(m *h1Mocks, ret *nmnumber.Number, err error) {
			m.req.EXPECT().NumberV1NumberUpdateFlowID(h1Ctx(), id, &cf, &mf).Return(ret, err)
		},
	)))
}

func Test_NumberUpdateMetadata_Matrix(t *testing.T) {
	id := h1UUID(h1ResourceID)
	md := nmnumber.Metadata{RTPDebug: true}
	h1Run(t, h1ActCases(h1NumberSpec(
		func(ctx context.Context, h *serviceHandler, a *h1Identity) (any, error) {
			return h.NumberUpdateMetadata(ctx, a, id, md)
		},
		func(m *h1Mocks, ret *nmnumber.Number, err error) {
			m.req.EXPECT().NumberV1NumberUpdateMetadata(h1Ctx(), id, md).Return(ret, err)
		},
	)))
}

func Test_NumberUpdate_Matrix(t *testing.T) {
	id := h1UUID(h1ResourceID)
	cf := h1UUID("6a1f0b10-0000-4000-8000-0000000000f1")
	mf := h1UUID("6a1f0b10-0000-4000-8000-0000000000f2")
	name, detail := "n", "d"

	tests := h1ActCases(h1NumberSpec(
		func(ctx context.Context, h *serviceHandler, a *h1Identity) (any, error) {
			return h.NumberUpdate(ctx, a, id, nil, nil, &name, &detail)
		},
		func(m *h1Mocks, ret *nmnumber.Number, err error) {
			m.req.EXPECT().NumberV1NumberUpdate(h1Ctx(), id, (*uuid.UUID)(nil), (*uuid.UUID)(nil), &name, &detail).Return(ret, err)
		},
	))

	getOwned := func(m *h1Mocks) {
		m.req.EXPECT().NumberV1NumberGet(h1Ctx(), id).Return(h1Number(h1CustomerID), nil)
	}
	update := func(ctx context.Context, h *serviceHandler, c, m *uuid.UUID) (any, error) {
		return h.NumberUpdate(ctx, h1Admin(), id, c, m, &name, &detail)
	}

	tests = append(tests,
		h1Case{
			name: "call flow of another customer is refused",
			setup: func(m *h1Mocks) {
				getOwned(m)
				m.req.EXPECT().FlowV1FlowGet(h1Ctx(), cf).Return(h1Flow(h1ResourceID, h1OtherID), nil)
			},
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return update(ctx, h, &cf, nil) },
			wantErr: serviceerrors.ErrInternal,
		},
		h1Case{
			name: "call flow lookup failure is refused",
			setup: func(m *h1Mocks) {
				getOwned(m)
				m.req.EXPECT().FlowV1FlowGet(h1Ctx(), cf).Return(nil, errH1RPC)
			},
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return update(ctx, h, &cf, nil) },
			wantErr: serviceerrors.ErrInternal,
		},
		h1Case{
			name: "message flow of another customer is refused",
			setup: func(m *h1Mocks) {
				getOwned(m)
				m.req.EXPECT().FlowV1FlowGet(h1Ctx(), mf).Return(h1Flow(h1AgentID, h1OtherID), nil)
			},
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return update(ctx, h, nil, &mf) },
			wantErr: serviceerrors.ErrInternal,
		},
		h1Case{
			name: "own flows are accepted",
			setup: func(m *h1Mocks) {
				getOwned(m)
				m.req.EXPECT().FlowV1FlowGet(h1Ctx(), cf).Return(h1Flow(h1ResourceID, h1CustomerID), nil)
				m.req.EXPECT().FlowV1FlowGet(h1Ctx(), mf).Return(h1Flow(h1AgentID, h1CustomerID), nil)
				m.req.EXPECT().NumberV1NumberUpdate(h1Ctx(), id, &cf, &mf, &name, &detail).Return(h1Number(h1CustomerID), nil)
			},
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return update(ctx, h, &cf, &mf) },
			wantRes: h1NumberWebhook(h1CustomerID),
		},
	)

	h1Run(t, tests)
}

func Test_NumberRenew_Matrix(t *testing.T) {
	h1Run(t, []h1Case{
		{
			name: "direct access",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.NumberRenew(ctx, h1Direct(), "2026-01-01")
			},
			wantErr: serviceerrors.ErrDirectAccessNotSupported,
		},
		{
			name: "customer admin is not super admin",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.NumberRenew(ctx, h1Admin(), "2026-01-01")
			},
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name: "rpc error",
			setup: func(m *h1Mocks) {
				m.req.EXPECT().NumberV1NumberRenewByTmRenew(h1Ctx(), "2026-01-01").Return(nil, errH1RPC)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.NumberRenew(ctx, h1Super(), "2026-01-01")
			},
			wantErr: errH1RPC,
		},
		{
			name: "success",
			setup: func(m *h1Mocks) {
				m.req.EXPECT().NumberV1NumberRenewByTmRenew(h1Ctx(), "2026-01-01").Return([]nmnumber.Number{*h1Number(h1CustomerID)}, nil)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.NumberRenew(ctx, h1Super(), "2026-01-01")
			},
			wantRes: []*nmnumber.WebhookMessage{h1NumberWebhook(h1CustomerID)},
		},
	})
}

func Test_numberVerifyFlow_Paths(t *testing.T) {
	fid := h1UUID(h1ResourceID)

	tests := []struct {
		name  string
		flow  uuid.UUID
		setup func(m *h1Mocks)
		want  bool
	}{
		{"nil flow is always valid", uuid.Nil, nil, true},
		{"lookup error", fid, func(m *h1Mocks) { m.req.EXPECT().FlowV1FlowGet(h1Ctx(), fid).Return(nil, errH1RPC) }, false},
		{"flow of another customer", fid, func(m *h1Mocks) {
			m.req.EXPECT().FlowV1FlowGet(h1Ctx(), fid).Return(h1Flow(h1ResourceID, h1OtherID), nil)
		}, false},
		{"own flow", fid, func(m *h1Mocks) {
			m.req.EXPECT().FlowV1FlowGet(h1Ctx(), fid).Return(h1Flow(h1ResourceID, h1CustomerID), nil)
		}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got bool
			h1Run(t, []h1Case{{
				name:  tt.name,
				setup: tt.setup,
				call: func(ctx context.Context, h *serviceHandler) (any, error) {
					got = h.numberVerifyFlow(ctx, h1Admin(), tt.flow)
					return nil, nil
				},
			}})
			if got != tt.want {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.want, got)
			}
		})
	}
}

func Test_convertNumberFilters_Invalid(t *testing.T) {
	h := &serviceHandler{}

	res, err := h.convertNumberFilters(map[string]string{"customer_id": "not-a-uuid"})
	if err == nil {
		t.Fatalf("Wrong match. expect: error, got: nil (res: %v)", res)
	}
	if res != nil {
		t.Errorf("Wrong result. expect: nil, got: %v", res)
	}
}
