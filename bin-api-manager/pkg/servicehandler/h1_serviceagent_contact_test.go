package servicehandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	cmcontact "monorepo/bin-contact-manager/models/contact"

	csaccesskey "monorepo/bin-customer-manager/models/accesskey"
)

func h1Contact(customerID string) *cmcontact.Contact {
	return &cmcontact.Contact{
		Identity: commonidentity.Identity{
			ID:         h1UUID(h1ResourceID),
			CustomerID: h1UUID(customerID),
		},
		DisplayName: "contact",
	}
}

func h1ContactWebhook(customerID string) *cmcontact.WebhookMessage {
	return &cmcontact.WebhookMessage{
		Identity: commonidentity.Identity{
			ID:         h1UUID(h1ResourceID),
			CustomerID: h1UUID(customerID),
		},
		DisplayName: "contact",
	}
}

// h1ContactSpec describes a service-agent contact operation. Every service
// agent operation is limited to agent identities, and an agent always holds
// the "all" permission inside its own customer, so the only reachable
// permission failure is a cross-tenant access.
func h1ContactSpec(
	call func(ctx context.Context, h *serviceHandler, a *h1Identity) (any, error),
	actExpect func(m *h1Mocks, ret *cmcontact.Contact, err error),
) h1ActSpec {
	id := h1UUID(h1ResourceID)
	spec := h1ActSpec{
		call:    call,
		getOK:   func(m *h1Mocks, c string) { m.req.EXPECT().ContactV1ContactGet(h1Ctx(), id).Return(h1Contact(c), nil) },
		getErr:  func(m *h1Mocks, err error) { m.req.EXPECT().ContactV1ContactGet(h1Ctx(), id).Return(nil, err) },
		okAgent: h1NoPerm(),
		// an agent without any permission still passes, so a foreign agent is the denied case
		deniedSameTenant: h1Foreign(),
		want:             h1ContactWebhook(h1CustomerID),
		directErr:        serviceerrors.ErrAuthenticationRequired,
	}
	if actExpect != nil {
		spec.actExpect = func(m *h1Mocks, c string, err error) {
			if err != nil {
				actExpect(m, nil, err)
				return
			}
			actExpect(m, h1Contact(c), nil)
		}
	}
	return spec
}

// h1NonAgentIdentities are the identities that a service-agent endpoint must refuse.
func h1NonAgentIdentities() map[string]*h1Identity {
	return map[string]*h1Identity{
		"accesskey": auth.NewAccesskeyIdentity(&csaccesskey.Accesskey{ID: h1UUID(h1AgentID), CustomerID: h1UUID(h1CustomerID)}),
		"delegate":  auth.NewDelegateIdentity(&auth.DelegateScope{CustomerID: h1UUID(h1CustomerID)}),
	}
}

func Test_ServiceAgentContactCreate_Matrix(t *testing.T) {
	cid := h1UUID(h1CustomerID)
	addrs := []cmcontact.AddressInput{{Type: "tel", Target: "+15555550100"}}
	tags := []uuid.UUID{h1UUID(h1AgentID)}
	create := func(ctx context.Context, h *serviceHandler, a *h1Identity) (any, error) {
		return h.ServiceAgentContactCreate(ctx, a, "f", "l", "d", "c", "j", "src", "ext", "notes", addrs, tags)
	}

	tests := []h1Case{}
	for name, a := range h1NonAgentIdentities() {
		tests = append(tests, h1Case{
			name:    "non agent " + name,
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return create(ctx, h, a) },
			wantErr: serviceerrors.ErrAuthenticationRequired,
		})
	}
	tests = append(tests,
		h1Case{
			name:    "direct identity",
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return create(ctx, h, h1Direct()) },
			wantErr: serviceerrors.ErrAuthenticationRequired,
		},
		h1Case{
			name: "rpc error",
			setup: func(m *h1Mocks) {
				m.req.EXPECT().ContactV1ContactCreate(h1Ctx(), cid, "f", "l", "d", "c", "j", "src", "ext", "notes", addrs, tags).Return(nil, errH1RPC)
			},
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return create(ctx, h, h1Admin()) },
			wantErr: errH1RPC,
		},
		h1Case{
			name: "success",
			setup: func(m *h1Mocks) {
				m.req.EXPECT().ContactV1ContactCreate(h1Ctx(), cid, "f", "l", "d", "c", "j", "src", "ext", "notes", addrs, tags).Return(h1Contact(h1CustomerID), nil)
			},
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return create(ctx, h, h1Admin()) },
			wantRes: h1ContactWebhook(h1CustomerID),
		},
	)

	h1Run(t, tests)
}

func Test_ServiceAgentContactGet_Matrix(t *testing.T) {
	id := h1UUID(h1ResourceID)
	h1Run(t, h1ActCases(h1ContactSpec(
		func(ctx context.Context, h *serviceHandler, a *h1Identity) (any, error) {
			return h.ServiceAgentContactGet(ctx, a, id)
		}, nil)))
}

func Test_ServiceAgentContactList_Matrix(t *testing.T) {
	filters := map[string]string{"deleted": "false"}
	typed := map[cmcontact.Field]any{cmcontact.FieldDeleted: false}

	h1Run(t, []h1Case{
		{
			name: "non agent",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.ServiceAgentContactList(ctx, h1Direct(), 10, "t", filters)
			},
			wantErr: serviceerrors.ErrAuthenticationRequired,
		},
		{
			name: "invalid filter",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.ServiceAgentContactList(ctx, h1Admin(), 10, "t", map[string]string{"id": "not-a-uuid"})
			},
			wantErrText: "invalid UUID string",
		},
		{
			name: "empty token and rpc error",
			setup: func(m *h1Mocks) {
				m.util.EXPECT().TimeGetCurTime().Return("2026-01-01T00:00:00.000000Z")
				m.req.EXPECT().ContactV1ContactList(h1Ctx(), "2026-01-01T00:00:00.000000Z", uint64(6), typed).Return(nil, errH1RPC)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.ServiceAgentContactList(ctx, h1Admin(), 6, "", filters)
			},
			wantErr: errH1RPC,
		},
		{
			name: "success",
			setup: func(m *h1Mocks) {
				m.req.EXPECT().ContactV1ContactList(h1Ctx(), "tok", uint64(6), typed).Return([]cmcontact.Contact{*h1Contact(h1CustomerID)}, nil)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.ServiceAgentContactList(ctx, h1Admin(), 6, "tok", filters)
			},
			wantRes: []*cmcontact.WebhookMessage{h1ContactWebhook(h1CustomerID)},
		},
	})
}

func Test_ServiceAgentContactUpdate_Matrix(t *testing.T) {
	id := h1UUID(h1ResourceID)
	s := func(v string) *string { return &v }
	f, l, d, c, j, e, n := s("f"), s("l"), s("d"), s("c"), s("j"), s("e"), s("n")

	h1Run(t, h1ActCases(h1ContactSpec(
		func(ctx context.Context, h *serviceHandler, a *h1Identity) (any, error) {
			return h.ServiceAgentContactUpdate(ctx, a, id, f, l, d, c, j, e, n)
		},
		func(m *h1Mocks, ret *cmcontact.Contact, err error) {
			m.req.EXPECT().ContactV1ContactUpdate(h1Ctx(), id, f, l, d, c, j, e, n).Return(ret, err)
		})))
}

func Test_ServiceAgentContactDelete_Matrix(t *testing.T) {
	id := h1UUID(h1ResourceID)
	h1Run(t, h1ActCases(h1ContactSpec(
		func(ctx context.Context, h *serviceHandler, a *h1Identity) (any, error) {
			return h.ServiceAgentContactDelete(ctx, a, id)
		},
		func(m *h1Mocks, ret *cmcontact.Contact, err error) {
			m.req.EXPECT().ContactV1ContactDelete(h1Ctx(), id).Return(ret, err)
		})))
}

func Test_ServiceAgentContactLookup_Matrix(t *testing.T) {
	cid := h1UUID(h1CustomerID)
	h1Run(t, []h1Case{
		{
			name: "non agent",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.ServiceAgentContactLookup(ctx, h1Direct(), "+15555550100", "")
			},
			wantErr: serviceerrors.ErrAuthenticationRequired,
		},
		{
			name: "rpc error",
			setup: func(m *h1Mocks) {
				m.req.EXPECT().ContactV1ContactLookup(h1Ctx(), cid, "+15555550100", "a@example.com").Return(nil, errH1RPC)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.ServiceAgentContactLookup(ctx, h1Admin(), "+15555550100", "a@example.com")
			},
			wantErr: errH1RPC,
		},
		{
			name: "success",
			setup: func(m *h1Mocks) {
				m.req.EXPECT().ContactV1ContactLookup(h1Ctx(), cid, "+15555550100", "a@example.com").Return(h1Contact(h1CustomerID), nil)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.ServiceAgentContactLookup(ctx, h1Admin(), "+15555550100", "a@example.com")
			},
			wantRes: h1ContactWebhook(h1CustomerID),
		},
	})
}

func Test_ServiceAgentContactAddressCreate_Matrix(t *testing.T) {
	id := h1UUID(h1ResourceID)
	call := func(ctx context.Context, h *serviceHandler, a *h1Identity) (any, error) {
		return h.ServiceAgentContactAddressCreate(ctx, a, id, "tel", "+15555550100", true, "n", "d")
	}
	spec := h1ContactSpec(call, func(m *h1Mocks, ret *cmcontact.Contact, err error) {
		if err != nil {
			m.req.EXPECT().ContactV1AddressCreate(h1Ctx(), id, "tel", "+15555550100", true, "n", "d").Return(nil, err)
			return
		}
		m.req.EXPECT().ContactV1AddressCreate(h1Ctx(), id, "tel", "+15555550100", true, "n", "d").Return(&cmcontact.Address{}, nil)
		m.req.EXPECT().ContactV1ContactGet(h1Ctx(), id).Return(ret, nil)
	})

	tests := h1ActCases(spec)
	tests = append(tests, h1Case{
		name: "re-read of the contact fails",
		setup: func(m *h1Mocks) {
			spec.getOK(m, h1CustomerID)
			m.req.EXPECT().ContactV1AddressCreate(h1Ctx(), id, "tel", "+15555550100", true, "n", "d").Return(&cmcontact.Address{}, nil)
			m.req.EXPECT().ContactV1ContactGet(h1Ctx(), id).Return(nil, errH1RPC)
		},
		call:    func(ctx context.Context, h *serviceHandler) (any, error) { return call(ctx, h, h1Admin()) },
		wantErr: errH1RPC,
	})

	h1Run(t, tests)
}

func Test_ServiceAgentContactAddressUpdate_Matrix(t *testing.T) {
	id := h1UUID(h1ResourceID)
	aid := h1UUID(h1AgentID)
	fields := map[string]any{"target": "+15555550101"}
	h1Run(t, h1ActCases(h1ContactSpec(
		func(ctx context.Context, h *serviceHandler, a *h1Identity) (any, error) {
			return h.ServiceAgentContactAddressUpdate(ctx, a, id, aid, fields)
		},
		func(m *h1Mocks, ret *cmcontact.Contact, err error) {
			m.req.EXPECT().ContactV1AddressUpdate(h1Ctx(), id, aid, fields).Return(ret, err)
		})))
}

func Test_ServiceAgentContactAddressDelete_Matrix(t *testing.T) {
	id := h1UUID(h1ResourceID)
	aid := h1UUID(h1AgentID)
	h1Run(t, h1ActCases(h1ContactSpec(
		func(ctx context.Context, h *serviceHandler, a *h1Identity) (any, error) {
			return h.ServiceAgentContactAddressDelete(ctx, a, id, aid)
		},
		func(m *h1Mocks, ret *cmcontact.Contact, err error) {
			m.req.EXPECT().ContactV1AddressDelete(h1Ctx(), id, aid).Return(ret, err)
		})))
}

func Test_ServiceAgentContactTagAdd_Matrix(t *testing.T) {
	id := h1UUID(h1ResourceID)
	tid := h1UUID(h1AgentID)
	h1Run(t, h1ActCases(h1ContactSpec(
		func(ctx context.Context, h *serviceHandler, a *h1Identity) (any, error) {
			return h.ServiceAgentContactTagAdd(ctx, a, id, tid)
		},
		func(m *h1Mocks, ret *cmcontact.Contact, err error) {
			m.req.EXPECT().ContactV1TagAdd(h1Ctx(), id, tid).Return(ret, err)
		})))
}

func Test_ServiceAgentContactTagRemove_Matrix(t *testing.T) {
	id := h1UUID(h1ResourceID)
	tid := h1UUID(h1AgentID)
	h1Run(t, h1ActCases(h1ContactSpec(
		func(ctx context.Context, h *serviceHandler, a *h1Identity) (any, error) {
			return h.ServiceAgentContactTagRemove(ctx, a, id, tid)
		},
		func(m *h1Mocks, ret *cmcontact.Contact, err error) {
			m.req.EXPECT().ContactV1TagRemove(h1Ctx(), id, tid).Return(ret, err)
		})))
}

func Test_ServiceAgentContact_NonAgentIdentities(t *testing.T) {
	id := h1UUID(h1ResourceID)
	for name, a := range h1NonAgentIdentities() {
		h1Run(t, []h1Case{
			{
				name:    "get " + name,
				call:    func(ctx context.Context, h *serviceHandler) (any, error) { return h.ServiceAgentContactGet(ctx, a, id) },
				wantErr: serviceerrors.ErrAuthenticationRequired,
			},
			{
				name: "delete " + name,
				call: func(ctx context.Context, h *serviceHandler) (any, error) {
					return h.ServiceAgentContactDelete(ctx, a, id)
				},
				wantErr: serviceerrors.ErrAuthenticationRequired,
			},
		})
	}
}
