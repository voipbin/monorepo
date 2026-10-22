package servicehandler

import (
	"context"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/pkg/dbhandler"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	cmcontact "monorepo/bin-contact-manager/models/contact"

	"github.com/gofrs/uuid"
)

var (
	h2AddrID    = uuid.FromStringOrNil("a2000000-0000-0000-0000-0000000000d1")
	h2ContactID = uuid.FromStringOrNil("a2000000-0000-0000-0000-0000000000d2")
)

func h2Addr(customerID, contactID uuid.UUID) *cmcontact.Address {
	return &cmcontact.Address{ID: h2AddrID, CustomerID: customerID, ContactID: contactID}
}

func h2Contact(customerID uuid.UUID) *cmcontact.Contact {
	return &cmcontact.Contact{Identity: commonidentity.Identity{ID: h2ContactID, CustomerID: customerID}}
}

func Test_ContactAddressList(t *testing.T) {
	admin := h2AgentIdentity(h2CustomerID, amagent.PermissionCustomerAdmin)
	filters := func() map[string]any { return map[string]any{"type": "tel"} }
	expectFilters := map[string]any{"type": "tel", "customer_id": h2CustomerID}
	list := []cmcontact.Address{*h2Addr(h2CustomerID, h2ContactID)}

	h2RunCases(t, []h2Case{
		{
			name: "direct access rejected",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.ContactAddressList(ctx, h2DirectIdentity(h2CustomerID), filters(), "tok", 10)
			},
			wantErr: serviceerrors.ErrDirectAccessNotSupported,
		},
		{
			name: "no permission",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.ContactAddressList(ctx, h2AgentIdentity(h2CustomerID, amagent.PermissionNone), filters(), "tok", 10)
			},
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name: "rpc error",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				req.EXPECT().ContactV1ContactAddressList(ctx, h2CustomerID, expectFilters, "tok", uint64(10)).Return(nil, errH2RPC)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.ContactAddressList(ctx, admin, filters(), "tok", 10)
			},
			wantErr: errH2RPC,
		},
		{
			name: "success injects customer_id filter",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				req.EXPECT().ContactV1ContactAddressList(ctx, h2CustomerID, expectFilters, "tok", uint64(10)).Return(list, nil)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.ContactAddressList(ctx, admin, filters(), "tok", 10)
			},
			wantRes: list,
		},
	})
}

func Test_ContactAddressGet(t *testing.T) {
	admin := h2AgentIdentity(h2CustomerID, amagent.PermissionCustomerManager)

	h2RunCases(t, []h2Case{
		{
			name: "direct access rejected",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.ContactAddressGet(ctx, h2DirectIdentity(h2CustomerID), h2AddrID)
			},
			wantErr: serviceerrors.ErrDirectAccessNotSupported,
		},
		{
			name: "no permission",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.ContactAddressGet(ctx, h2AgentIdentity(h2CustomerID, amagent.PermissionNone), h2AddrID)
			},
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name: "rpc error",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				req.EXPECT().ContactV1ContactAddressGet(ctx, h2CustomerID, h2AddrID).Return(nil, errH2RPC)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.ContactAddressGet(ctx, admin, h2AddrID)
			},
			wantErr: errH2RPC,
		},
		{
			name: "cross tenant response rejected",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				req.EXPECT().ContactV1ContactAddressGet(ctx, h2CustomerID, h2AddrID).Return(h2Addr(h2OtherCustomerID, h2ContactID), nil)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.ContactAddressGet(ctx, admin, h2AddrID)
			},
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name: "success",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				req.EXPECT().ContactV1ContactAddressGet(ctx, h2CustomerID, h2AddrID).Return(h2Addr(h2CustomerID, h2ContactID), nil)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.ContactAddressGet(ctx, admin, h2AddrID)
			},
			wantRes: h2Addr(h2CustomerID, h2ContactID),
		},
	})
}

func Test_ContactAddressCreateIndependent(t *testing.T) {
	admin := h2AgentIdentity(h2CustomerID, amagent.PermissionCustomerAdmin)
	create := func(ctx context.Context, h *serviceHandler, a bool, contactID uuid.UUID) (any, error) {
		id := admin
		if !a {
			id = h2AgentIdentity(h2CustomerID, amagent.PermissionNone)
		}
		return h.ContactAddressCreateIndependent(ctx, id, contactID, "tel", "+15555550100", true, "nm", "dt")
	}

	h2RunCases(t, []h2Case{
		{
			name: "direct access rejected",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.ContactAddressCreateIndependent(ctx, h2DirectIdentity(h2CustomerID), h2ContactID, "tel", "+15555550100", true, "nm", "dt")
			},
			wantErr: serviceerrors.ErrDirectAccessNotSupported,
		},
		{
			name: "unresolved no permission",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return create(ctx, h, false, uuid.Nil)
			},
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name: "unresolved rpc error",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				req.EXPECT().ContactV1ContactAddressCreate(ctx, h2CustomerID, uuid.Nil, "tel", "+15555550100", true, "nm", "dt").Return(nil, errH2RPC)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return create(ctx, h, true, uuid.Nil)
			},
			wantErr: errH2RPC,
		},
		{
			name: "contact lookup fails",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				req.EXPECT().ContactV1ContactGet(ctx, h2ContactID).Return(nil, errH2RPC)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return create(ctx, h, true, h2ContactID)
			},
			wantErr: errH2RPC,
		},
		{
			name: "cross tenant contact",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				req.EXPECT().ContactV1ContactGet(ctx, h2ContactID).Return(h2Contact(h2OtherCustomerID), nil)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return create(ctx, h, true, h2ContactID)
			},
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name: "contact rpc error",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				req.EXPECT().ContactV1ContactGet(ctx, h2ContactID).Return(h2Contact(h2CustomerID), nil)
				req.EXPECT().ContactV1ContactAddressCreate(ctx, h2CustomerID, h2ContactID, "tel", "+15555550100", true, "nm", "dt").Return(nil, errH2RPC)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return create(ctx, h, true, h2ContactID)
			},
			wantErr: errH2RPC,
		},
		{
			name: "contact success",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				req.EXPECT().ContactV1ContactGet(ctx, h2ContactID).Return(h2Contact(h2CustomerID), nil)
				req.EXPECT().ContactV1ContactAddressCreate(ctx, h2CustomerID, h2ContactID, "tel", "+15555550100", true, "nm", "dt").Return(h2Addr(h2CustomerID, h2ContactID), nil)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return create(ctx, h, true, h2ContactID)
			},
			wantRes: h2Addr(h2CustomerID, h2ContactID),
		},
	})
}

func Test_ContactAddressClaim_Errors(t *testing.T) {
	admin := h2AgentIdentity(h2CustomerID, amagent.PermissionCustomerAdmin)
	claim := func(ctx context.Context, h *serviceHandler) (any, error) {
		return h.ContactAddressClaim(ctx, admin, h2AddrID, h2ContactID, true)
	}
	getAddr := func(ctx context.Context, req *requesthandler.MockRequestHandler, cust uuid.UUID) {
		req.EXPECT().ContactV1ContactAddressGet(ctx, h2CustomerID, h2AddrID).Return(h2Addr(cust, uuid.Nil), nil)
	}

	h2RunCases(t, []h2Case{
		{
			name: "direct access rejected",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.ContactAddressClaim(ctx, h2DirectIdentity(h2CustomerID), h2AddrID, h2ContactID, false)
			},
			wantErr: serviceerrors.ErrDirectAccessNotSupported,
		},
		{
			name: "no permission",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.ContactAddressClaim(ctx, h2AgentIdentity(h2CustomerID, amagent.PermissionNone), h2AddrID, h2ContactID, false)
			},
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name: "address lookup fails",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				req.EXPECT().ContactV1ContactAddressGet(ctx, h2CustomerID, h2AddrID).Return(nil, errH2RPC)
			},
			call:    claim,
			wantErr: errH2RPC,
		},
		{
			name: "cross tenant address is not found",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				getAddr(ctx, req, h2OtherCustomerID)
			},
			call:    claim,
			wantErr: serviceerrors.ErrNotFound,
		},
		{
			name: "contact lookup fails",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				getAddr(ctx, req, h2CustomerID)
				req.EXPECT().ContactV1ContactGet(ctx, h2ContactID).Return(nil, errH2RPC)
			},
			call:    claim,
			wantErr: errH2RPC,
		},
		{
			name: "cross tenant contact is not found",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				getAddr(ctx, req, h2CustomerID)
				req.EXPECT().ContactV1ContactGet(ctx, h2ContactID).Return(h2Contact(h2OtherCustomerID), nil)
			},
			call:    claim,
			wantErr: serviceerrors.ErrNotFound,
		},
		{
			name: "claim rpc error",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				getAddr(ctx, req, h2CustomerID)
				req.EXPECT().ContactV1ContactGet(ctx, h2ContactID).Return(h2Contact(h2CustomerID), nil)
				req.EXPECT().ContactV1ContactAddressClaim(ctx, h2CustomerID, h2AddrID, h2ContactID, true).Return(nil, errH2RPC)
			},
			call:    claim,
			wantErr: errH2RPC,
		},
	})
}

func Test_ContactAddressUpdateIndependent(t *testing.T) {
	admin := h2AgentIdentity(h2CustomerID, amagent.PermissionCustomerAdmin)
	fields := map[string]any{"name": "new"}
	update := func(ctx context.Context, h *serviceHandler) (any, error) {
		return h.ContactAddressUpdateIndependent(ctx, admin, h2AddrID, fields)
	}

	h2RunCases(t, []h2Case{
		{
			name: "direct access rejected",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.ContactAddressUpdateIndependent(ctx, h2DirectIdentity(h2CustomerID), h2AddrID, fields)
			},
			wantErr: serviceerrors.ErrDirectAccessNotSupported,
		},
		{
			name: "address lookup fails",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				req.EXPECT().ContactV1ContactAddressGet(ctx, h2CustomerID, h2AddrID).Return(nil, errH2RPC)
			},
			call:    update,
			wantErr: errH2RPC,
		},
		{
			name: "cross tenant address",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				req.EXPECT().ContactV1ContactAddressGet(ctx, h2CustomerID, h2AddrID).Return(h2Addr(h2OtherCustomerID, h2ContactID), nil)
			},
			call:    update,
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name: "rpc error",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				req.EXPECT().ContactV1ContactAddressGet(ctx, h2CustomerID, h2AddrID).Return(h2Addr(h2CustomerID, h2ContactID), nil)
				req.EXPECT().ContactV1ContactAddressUpdate(ctx, h2CustomerID, h2ContactID, h2AddrID, fields).Return(nil, errH2RPC)
			},
			call:    update,
			wantErr: errH2RPC,
		},
		{
			name: "success",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				req.EXPECT().ContactV1ContactAddressGet(ctx, h2CustomerID, h2AddrID).Return(h2Addr(h2CustomerID, h2ContactID), nil)
				req.EXPECT().ContactV1ContactAddressUpdate(ctx, h2CustomerID, h2ContactID, h2AddrID, fields).Return(h2Addr(h2CustomerID, h2ContactID), nil)
			},
			call:    update,
			wantRes: h2Addr(h2CustomerID, h2ContactID),
		},
		{
			name: "no permission on own-tenant address",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				req.EXPECT().ContactV1ContactAddressGet(ctx, h2CustomerID, h2AddrID).Return(h2Addr(h2CustomerID, h2ContactID), nil)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.ContactAddressUpdateIndependent(ctx, h2AgentIdentity(h2CustomerID, amagent.PermissionNone), h2AddrID, fields)
			},
			wantErr: serviceerrors.ErrPermissionDenied,
		},
	})
}

func Test_ContactAddressDeleteIndependent(t *testing.T) {
	admin := h2AgentIdentity(h2CustomerID, amagent.PermissionCustomerAdmin)
	del := func(ctx context.Context, h *serviceHandler) (any, error) {
		return h.ContactAddressDeleteIndependent(ctx, admin, h2AddrID)
	}

	h2RunCases(t, []h2Case{
		{
			name: "direct access rejected",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.ContactAddressDeleteIndependent(ctx, h2DirectIdentity(h2CustomerID), h2AddrID)
			},
			wantErr: serviceerrors.ErrDirectAccessNotSupported,
		},
		{
			name: "address lookup fails",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				req.EXPECT().ContactV1ContactAddressGet(ctx, h2CustomerID, h2AddrID).Return(nil, errH2RPC)
			},
			call:    del,
			wantErr: errH2RPC,
		},
		{
			name: "cross tenant address",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				req.EXPECT().ContactV1ContactAddressGet(ctx, h2CustomerID, h2AddrID).Return(h2Addr(h2OtherCustomerID, h2ContactID), nil)
			},
			call:    del,
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name: "rpc error",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				req.EXPECT().ContactV1ContactAddressGet(ctx, h2CustomerID, h2AddrID).Return(h2Addr(h2CustomerID, h2ContactID), nil)
				req.EXPECT().ContactV1ContactAddressDelete(ctx, h2CustomerID, h2ContactID, h2AddrID).Return(nil, errH2RPC)
			},
			call:    del,
			wantErr: errH2RPC,
		},
		{
			name: "success",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				req.EXPECT().ContactV1ContactAddressGet(ctx, h2CustomerID, h2AddrID).Return(h2Addr(h2CustomerID, h2ContactID), nil)
				req.EXPECT().ContactV1ContactAddressDelete(ctx, h2CustomerID, h2ContactID, h2AddrID).Return(h2Addr(h2CustomerID, h2ContactID), nil)
			},
			call:    del,
			wantRes: h2Addr(h2CustomerID, h2ContactID),
		},
	})
}

func Test_ServiceAgentContactAddressList(t *testing.T) {
	admin := h2AgentIdentity(h2CustomerID, amagent.PermissionCustomerAdmin)
	filters := map[string]any{"type": "tel"}
	list := []cmcontact.Address{*h2Addr(h2CustomerID, h2ContactID)}
	agentGet := func(ctx context.Context, req *requesthandler.MockRequestHandler) {
		req.EXPECT().AgentV1AgentGet(ctx, h2AgentID).Return(h2AgentResp(h2CustomerID), nil)
	}

	h2RunCases(t, []h2Case{
		{
			name: "agent lookup fails",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				req.EXPECT().AgentV1AgentGet(ctx, h2AgentID).Return(nil, errH2RPC)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.ServiceAgentContactAddressList(ctx, admin, filters, "tok", 10)
			},
			wantErr: errH2RPC,
		},
		{
			name: "no permission",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				agentGet(ctx, req)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.ServiceAgentContactAddressList(ctx, h2AgentIdentity(h2CustomerID, amagent.PermissionNone), filters, "tok", 10)
			},
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name: "rpc error",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				agentGet(ctx, req)
				req.EXPECT().ContactV1ContactAddressList(ctx, h2CustomerID, filters, "tok", uint64(10)).Return(nil, errH2RPC)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.ServiceAgentContactAddressList(ctx, admin, filters, "tok", 10)
			},
			wantErr: errH2RPC,
		},
		{
			name: "success",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				agentGet(ctx, req)
				req.EXPECT().ContactV1ContactAddressList(ctx, h2CustomerID, filters, "tok", uint64(10)).Return(list, nil)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.ServiceAgentContactAddressList(ctx, admin, filters, "tok", 10)
			},
			wantRes: list,
		},
	})
}

func Test_ServiceAgentContactAddressGet(t *testing.T) {
	admin := h2AgentIdentity(h2CustomerID, amagent.PermissionCustomerAdmin)
	agentGet := func(ctx context.Context, req *requesthandler.MockRequestHandler) {
		req.EXPECT().AgentV1AgentGet(ctx, h2AgentID).Return(h2AgentResp(h2CustomerID), nil)
	}
	get := func(ctx context.Context, h *serviceHandler) (any, error) {
		return h.ServiceAgentContactAddressGet(ctx, admin, h2AddrID)
	}

	h2RunCases(t, []h2Case{
		{
			name: "agent lookup fails",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				req.EXPECT().AgentV1AgentGet(ctx, h2AgentID).Return(nil, errH2RPC)
			},
			call:    get,
			wantErr: errH2RPC,
		},
		{
			name: "rpc error",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				agentGet(ctx, req)
				req.EXPECT().ContactV1ContactAddressGet(ctx, h2CustomerID, h2AddrID).Return(nil, errH2RPC)
			},
			call:    get,
			wantErr: errH2RPC,
		},
		{
			name: "cross tenant response",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				agentGet(ctx, req)
				req.EXPECT().ContactV1ContactAddressGet(ctx, h2CustomerID, h2AddrID).Return(h2Addr(h2OtherCustomerID, h2ContactID), nil)
			},
			call:    get,
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name: "success",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				agentGet(ctx, req)
				req.EXPECT().ContactV1ContactAddressGet(ctx, h2CustomerID, h2AddrID).Return(h2Addr(h2CustomerID, h2ContactID), nil)
			},
			call:    get,
			wantRes: h2Addr(h2CustomerID, h2ContactID),
		},
	})
}

func Test_ServiceAgentContactAddressCreateIndependent(t *testing.T) {
	admin := h2AgentIdentity(h2CustomerID, amagent.PermissionCustomerAdmin)
	agentGet := func(ctx context.Context, req *requesthandler.MockRequestHandler) {
		req.EXPECT().AgentV1AgentGet(ctx, h2AgentID).Return(h2AgentResp(h2CustomerID), nil)
	}
	create := func(contactID uuid.UUID) func(ctx context.Context, h *serviceHandler) (any, error) {
		return func(ctx context.Context, h *serviceHandler) (any, error) {
			return h.ServiceAgentContactAddressCreateIndependent(ctx, admin, contactID, "email", "a@example.com", false, "nm", "dt")
		}
	}

	h2RunCases(t, []h2Case{
		{
			name: "agent lookup fails",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				req.EXPECT().AgentV1AgentGet(ctx, h2AgentID).Return(nil, errH2RPC)
			},
			call:    create(h2ContactID),
			wantErr: errH2RPC,
		},
		{
			name: "unresolved rpc error",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				agentGet(ctx, req)
				req.EXPECT().ContactV1ContactAddressCreate(ctx, h2CustomerID, uuid.Nil, "email", "a@example.com", false, "nm", "dt").Return(nil, errH2RPC)
			},
			call:    create(uuid.Nil),
			wantErr: errH2RPC,
		},
		{
			name: "unresolved success",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				agentGet(ctx, req)
				req.EXPECT().ContactV1ContactAddressCreate(ctx, h2CustomerID, uuid.Nil, "email", "a@example.com", false, "nm", "dt").Return(h2Addr(h2CustomerID, uuid.Nil), nil)
			},
			call:    create(uuid.Nil),
			wantRes: h2Addr(h2CustomerID, uuid.Nil),
		},
		{
			name: "contact lookup fails",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				agentGet(ctx, req)
				req.EXPECT().ContactV1ContactGet(ctx, h2ContactID).Return(nil, errH2RPC)
			},
			call:    create(h2ContactID),
			wantErr: errH2RPC,
		},
		{
			name: "cross tenant contact",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				agentGet(ctx, req)
				req.EXPECT().ContactV1ContactGet(ctx, h2ContactID).Return(h2Contact(h2OtherCustomerID), nil)
			},
			call:    create(h2ContactID),
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name: "contact rpc error",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				agentGet(ctx, req)
				req.EXPECT().ContactV1ContactGet(ctx, h2ContactID).Return(h2Contact(h2CustomerID), nil)
				req.EXPECT().ContactV1ContactAddressCreate(ctx, h2CustomerID, h2ContactID, "email", "a@example.com", false, "nm", "dt").Return(nil, errH2RPC)
			},
			call:    create(h2ContactID),
			wantErr: errH2RPC,
		},
		{
			name: "contact success",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				agentGet(ctx, req)
				req.EXPECT().ContactV1ContactGet(ctx, h2ContactID).Return(h2Contact(h2CustomerID), nil)
				req.EXPECT().ContactV1ContactAddressCreate(ctx, h2CustomerID, h2ContactID, "email", "a@example.com", false, "nm", "dt").Return(h2Addr(h2CustomerID, h2ContactID), nil)
			},
			call:    create(h2ContactID),
			wantRes: h2Addr(h2CustomerID, h2ContactID),
		},
	})
}

func Test_ServiceAgentContactAddressClaim_Errors(t *testing.T) {
	admin := h2AgentIdentity(h2CustomerID, amagent.PermissionCustomerAdmin)
	agentGet := func(ctx context.Context, req *requesthandler.MockRequestHandler) {
		req.EXPECT().AgentV1AgentGet(ctx, h2AgentID).Return(h2AgentResp(h2CustomerID), nil)
	}
	claim := func(ctx context.Context, h *serviceHandler) (any, error) {
		return h.ServiceAgentContactAddressClaim(ctx, admin, h2AddrID, h2ContactID)
	}

	h2RunCases(t, []h2Case{
		{
			name: "agent lookup fails",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				req.EXPECT().AgentV1AgentGet(ctx, h2AgentID).Return(nil, errH2RPC)
			},
			call:    claim,
			wantErr: errH2RPC,
		},
		{
			name: "address lookup fails",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				agentGet(ctx, req)
				req.EXPECT().ContactV1ContactAddressGet(ctx, h2CustomerID, h2AddrID).Return(nil, errH2RPC)
			},
			call:    claim,
			wantErr: errH2RPC,
		},
		{
			name: "contact lookup fails",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				agentGet(ctx, req)
				req.EXPECT().ContactV1ContactAddressGet(ctx, h2CustomerID, h2AddrID).Return(h2Addr(h2CustomerID, uuid.Nil), nil)
				req.EXPECT().ContactV1ContactGet(ctx, h2ContactID).Return(nil, errH2RPC)
			},
			call:    claim,
			wantErr: errH2RPC,
		},
		{
			name: "claim rpc error",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				agentGet(ctx, req)
				req.EXPECT().ContactV1ContactAddressGet(ctx, h2CustomerID, h2AddrID).Return(h2Addr(h2CustomerID, uuid.Nil), nil)
				req.EXPECT().ContactV1ContactGet(ctx, h2ContactID).Return(h2Contact(h2CustomerID), nil)
				req.EXPECT().ContactV1ContactAddressClaim(ctx, h2CustomerID, h2AddrID, h2ContactID, false).Return(nil, errH2RPC)
			},
			call:    claim,
			wantErr: errH2RPC,
		},
	})
}

func Test_ServiceAgentContactAddressUpdateIndependent(t *testing.T) {
	admin := h2AgentIdentity(h2CustomerID, amagent.PermissionCustomerAdmin)
	fields := map[string]any{"name": "new"}
	agentGet := func(ctx context.Context, req *requesthandler.MockRequestHandler) {
		req.EXPECT().AgentV1AgentGet(ctx, h2AgentID).Return(h2AgentResp(h2CustomerID), nil)
	}
	update := func(ctx context.Context, h *serviceHandler) (any, error) {
		return h.ServiceAgentContactAddressUpdateIndependent(ctx, admin, h2AddrID, fields)
	}

	h2RunCases(t, []h2Case{
		{
			name: "agent lookup fails",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				req.EXPECT().AgentV1AgentGet(ctx, h2AgentID).Return(nil, errH2RPC)
			},
			call:    update,
			wantErr: errH2RPC,
		},
		{
			name: "address lookup fails",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				agentGet(ctx, req)
				req.EXPECT().ContactV1ContactAddressGet(ctx, h2CustomerID, h2AddrID).Return(nil, errH2RPC)
			},
			call:    update,
			wantErr: errH2RPC,
		},
		{
			name: "cross tenant address",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				agentGet(ctx, req)
				req.EXPECT().ContactV1ContactAddressGet(ctx, h2CustomerID, h2AddrID).Return(h2Addr(h2OtherCustomerID, h2ContactID), nil)
			},
			call:    update,
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name: "rpc error",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				agentGet(ctx, req)
				req.EXPECT().ContactV1ContactAddressGet(ctx, h2CustomerID, h2AddrID).Return(h2Addr(h2CustomerID, h2ContactID), nil)
				req.EXPECT().ContactV1ContactAddressUpdate(ctx, h2CustomerID, h2ContactID, h2AddrID, fields).Return(nil, errH2RPC)
			},
			call:    update,
			wantErr: errH2RPC,
		},
		{
			name: "success",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				agentGet(ctx, req)
				req.EXPECT().ContactV1ContactAddressGet(ctx, h2CustomerID, h2AddrID).Return(h2Addr(h2CustomerID, h2ContactID), nil)
				req.EXPECT().ContactV1ContactAddressUpdate(ctx, h2CustomerID, h2ContactID, h2AddrID, fields).Return(h2Addr(h2CustomerID, h2ContactID), nil)
			},
			call:    update,
			wantRes: h2Addr(h2CustomerID, h2ContactID),
		},
	})
}

func Test_ServiceAgentContactAddressDeleteIndependent(t *testing.T) {
	admin := h2AgentIdentity(h2CustomerID, amagent.PermissionCustomerAdmin)
	agentGet := func(ctx context.Context, req *requesthandler.MockRequestHandler) {
		req.EXPECT().AgentV1AgentGet(ctx, h2AgentID).Return(h2AgentResp(h2CustomerID), nil)
	}
	del := func(ctx context.Context, h *serviceHandler) (any, error) {
		return h.ServiceAgentContactAddressDeleteIndependent(ctx, admin, h2AddrID)
	}

	h2RunCases(t, []h2Case{
		{
			name: "agent lookup fails",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				req.EXPECT().AgentV1AgentGet(ctx, h2AgentID).Return(nil, errH2RPC)
			},
			call:    del,
			wantErr: errH2RPC,
		},
		{
			name: "address lookup fails",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				agentGet(ctx, req)
				req.EXPECT().ContactV1ContactAddressGet(ctx, h2CustomerID, h2AddrID).Return(nil, errH2RPC)
			},
			call:    del,
			wantErr: errH2RPC,
		},
		{
			name: "cross tenant address",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				agentGet(ctx, req)
				req.EXPECT().ContactV1ContactAddressGet(ctx, h2CustomerID, h2AddrID).Return(h2Addr(h2OtherCustomerID, h2ContactID), nil)
			},
			call:    del,
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name: "rpc error",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				agentGet(ctx, req)
				req.EXPECT().ContactV1ContactAddressGet(ctx, h2CustomerID, h2AddrID).Return(h2Addr(h2CustomerID, h2ContactID), nil)
				req.EXPECT().ContactV1ContactAddressDelete(ctx, h2CustomerID, h2ContactID, h2AddrID).Return(nil, errH2RPC)
			},
			call:    del,
			wantErr: errH2RPC,
		},
		{
			name: "success",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				agentGet(ctx, req)
				req.EXPECT().ContactV1ContactAddressGet(ctx, h2CustomerID, h2AddrID).Return(h2Addr(h2CustomerID, h2ContactID), nil)
				req.EXPECT().ContactV1ContactAddressDelete(ctx, h2CustomerID, h2ContactID, h2AddrID).Return(h2Addr(h2CustomerID, h2ContactID), nil)
			},
			call:    del,
			wantRes: h2Addr(h2CustomerID, h2ContactID),
		},
	})
}
