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
	cmcontact "monorepo/bin-contact-manager/models/contact"

	"github.com/gofrs/uuid"
)

// h2ContactLookup expects ContactV1ContactGet for h2ContactID.
func h2ContactLookup(ctx context.Context, req *requesthandler.MockRequestHandler, customerID uuid.UUID, failure bool) {
	if failure {
		req.EXPECT().ContactV1ContactGet(ctx, h2ContactID).Return(nil, errH2RPC)
		return
	}
	req.EXPECT().ContactV1ContactGet(ctx, h2ContactID).Return(h2Contact(customerID), nil)
}

func h2ContactRPC(set func(ctx context.Context, req *requesthandler.MockRequestHandler, res *cmcontact.Contact, err error)) h2RPCFn {
	return func(ctx context.Context, req *requesthandler.MockRequestHandler, failure bool) {
		if failure {
			set(ctx, req, nil, errH2RPC)
			return
		}
		set(ctx, req, h2Contact(h2CustomerID), nil)
	}
}

func Test_ContactCreate_h2(t *testing.T) {
	addrs := []cmcontact.AddressInput{{Type: "tel", Target: "+15555550100"}}
	tags := []uuid.UUID{uuid.FromStringOrNil("a2000000-0000-0000-0000-000000000c01")}
	h2RunCases(t, h2StdCases(
		func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) (any, error) {
			return h.ContactCreate(ctx, a, "first", "last", "disp", "co", "job", "src", "ext", "notes", addrs, tags)
		},
		nil,
		h2ContactRPC(func(ctx context.Context, req *requesthandler.MockRequestHandler, res *cmcontact.Contact, err error) {
			req.EXPECT().ContactV1ContactCreate(ctx, h2CustomerID, "first", "last", "disp", "co", "job", "src", "ext", "notes", addrs, tags).Return(res, err)
		}),
		h2Contact(h2CustomerID).ConvertWebhookMessage(),
	))
}

func Test_ContactGet_h2(t *testing.T) {
	var cases []h2Case
	for _, c := range h2StdCases(
		func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) (any, error) {
			return h.ContactGet(ctx, a, h2ContactID)
		},
		h2ContactLookup,
		func(context.Context, *requesthandler.MockRequestHandler, bool) {},
		h2Contact(h2CustomerID).ConvertWebhookMessage(),
	) {
		if c.name != "rpc error" {
			cases = append(cases, c)
		}
	}
	h2RunCases(t, cases)
}

func Test_ContactUpdate_h2(t *testing.T) {
	first, last, disp, co, job, ext, notes := "f", "l", "d", "c", "j", "e", "n"
	h2RunCases(t, h2StdCases(
		func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) (any, error) {
			return h.ContactUpdate(ctx, a, h2ContactID, &first, &last, &disp, &co, &job, &ext, &notes)
		},
		h2ContactLookup,
		h2ContactRPC(func(ctx context.Context, req *requesthandler.MockRequestHandler, res *cmcontact.Contact, err error) {
			req.EXPECT().ContactV1ContactUpdate(ctx, h2ContactID, &first, &last, &disp, &co, &job, &ext, &notes).Return(res, err)
		}),
		h2Contact(h2CustomerID).ConvertWebhookMessage(),
	))
}

func Test_ContactDelete_h2(t *testing.T) {
	h2RunCases(t, h2StdCases(
		func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) (any, error) {
			return h.ContactDelete(ctx, a, h2ContactID)
		},
		h2ContactLookup,
		h2ContactRPC(func(ctx context.Context, req *requesthandler.MockRequestHandler, res *cmcontact.Contact, err error) {
			req.EXPECT().ContactV1ContactDelete(ctx, h2ContactID).Return(res, err)
		}),
		h2Contact(h2CustomerID).ConvertWebhookMessage(),
	))
}

func Test_ContactLookup_h2(t *testing.T) {
	h2RunCases(t, h2StdCases(
		func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) (any, error) {
			return h.ContactLookup(ctx, a, "+15555550100", "a@example.com")
		},
		nil,
		h2ContactRPC(func(ctx context.Context, req *requesthandler.MockRequestHandler, res *cmcontact.Contact, err error) {
			req.EXPECT().ContactV1ContactLookup(ctx, h2CustomerID, "+15555550100", "a@example.com").Return(res, err)
		}),
		h2Contact(h2CustomerID).ConvertWebhookMessage(),
	))
}

func Test_ContactAddressCreate_h2(t *testing.T) {
	// the RPC returns the created address; the handler re-reads the contact afterwards.
	h2RunCases(t, h2StdCases(
		func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) (any, error) {
			return h.ContactAddressCreate(ctx, a, h2ContactID, "tel", "+15555550100", true, "nm", "dt")
		},
		h2ContactLookup,
		func(ctx context.Context, req *requesthandler.MockRequestHandler, failure bool) {
			if failure {
				req.EXPECT().ContactV1AddressCreate(ctx, h2ContactID, "tel", "+15555550100", true, "nm", "dt").Return(nil, errH2RPC)
				return
			}
			req.EXPECT().ContactV1AddressCreate(ctx, h2ContactID, "tel", "+15555550100", true, "nm", "dt").Return(h2Addr(h2CustomerID, h2ContactID), nil)
			req.EXPECT().ContactV1ContactGet(ctx, h2ContactID).Return(h2Contact(h2CustomerID), nil)
		},
		h2Contact(h2CustomerID).ConvertWebhookMessage(),
	))
}

func Test_ContactAddressCreate_h2_refetchFails(t *testing.T) {
	admin := h2AgentIdentity(h2CustomerID, amagent.PermissionCustomerAdmin)
	h2RunCases(t, []h2Case{
		{
			name: "contact re-read fails after create",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				h2ContactLookup(ctx, req, h2CustomerID, false)
				req.EXPECT().ContactV1AddressCreate(ctx, h2ContactID, "tel", "+15555550100", true, "nm", "dt").Return(h2Addr(h2CustomerID, h2ContactID), nil)
				req.EXPECT().ContactV1ContactGet(ctx, h2ContactID).Return(nil, errH2RPC)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.ContactAddressCreate(ctx, admin, h2ContactID, "tel", "+15555550100", true, "nm", "dt")
			},
			wantErr: errH2RPC,
		},
	})
}

func Test_ContactAddressUpdate_h2(t *testing.T) {
	fields := map[string]any{"name": "new"}
	h2RunCases(t, h2StdCases(
		func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) (any, error) {
			return h.ContactAddressUpdate(ctx, a, h2ContactID, h2AddrID, fields)
		},
		h2ContactLookup,
		h2ContactRPC(func(ctx context.Context, req *requesthandler.MockRequestHandler, res *cmcontact.Contact, err error) {
			req.EXPECT().ContactV1AddressUpdate(ctx, h2ContactID, h2AddrID, fields).Return(res, err)
		}),
		h2Contact(h2CustomerID).ConvertWebhookMessage(),
	))
}

func Test_ContactAddressDelete_h2(t *testing.T) {
	h2RunCases(t, h2StdCases(
		func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) (any, error) {
			return h.ContactAddressDelete(ctx, a, h2ContactID, h2AddrID)
		},
		h2ContactLookup,
		h2ContactRPC(func(ctx context.Context, req *requesthandler.MockRequestHandler, res *cmcontact.Contact, err error) {
			req.EXPECT().ContactV1AddressDelete(ctx, h2ContactID, h2AddrID).Return(res, err)
		}),
		h2Contact(h2CustomerID).ConvertWebhookMessage(),
	))
}

func Test_ContactTagAdd_h2(t *testing.T) {
	tagID := uuid.FromStringOrNil("a2000000-0000-0000-0000-000000000c02")
	h2RunCases(t, h2StdCases(
		func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) (any, error) {
			return h.ContactTagAdd(ctx, a, h2ContactID, tagID)
		},
		h2ContactLookup,
		h2ContactRPC(func(ctx context.Context, req *requesthandler.MockRequestHandler, res *cmcontact.Contact, err error) {
			req.EXPECT().ContactV1TagAdd(ctx, h2ContactID, tagID).Return(res, err)
		}),
		h2Contact(h2CustomerID).ConvertWebhookMessage(),
	))
}

func Test_ContactTagRemove_h2(t *testing.T) {
	tagID := uuid.FromStringOrNil("a2000000-0000-0000-0000-000000000c02")
	h2RunCases(t, h2StdCases(
		func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) (any, error) {
			return h.ContactTagRemove(ctx, a, h2ContactID, tagID)
		},
		h2ContactLookup,
		h2ContactRPC(func(ctx context.Context, req *requesthandler.MockRequestHandler, res *cmcontact.Contact, err error) {
			req.EXPECT().ContactV1TagRemove(ctx, h2ContactID, tagID).Return(res, err)
		}),
		h2Contact(h2CustomerID).ConvertWebhookMessage(),
	))
}

func Test_ContactList_h2(t *testing.T) {
	admin := h2AgentIdentity(h2CustomerID, amagent.PermissionCustomerManager)
	filters := map[string]string{"customer_id": h2CustomerID.String(), "deleted": "false"}
	typed := map[cmcontact.Field]any{
		cmcontact.FieldCustomerID: h2CustomerID,
		cmcontact.FieldDeleted:    false,
	}
	list := []cmcontact.Contact{*h2Contact(h2CustomerID)}

	h2RunCases(t, []h2Case{
		{
			name: "direct access rejected",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.ContactList(ctx, h2DirectIdentity(h2CustomerID), 10, "tok", filters)
			},
			wantErr: serviceerrors.ErrDirectAccessNotSupported,
		},
		{
			name: "no permission",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.ContactList(ctx, h2AgentIdentity(h2CustomerID, amagent.PermissionNone), 10, "tok", filters)
			},
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name: "rpc error",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				req.EXPECT().ContactV1ContactList(ctx, "tok", uint64(10), typed).Return(nil, errH2RPC)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.ContactList(ctx, admin, 10, "tok", filters)
			},
			wantErr: errH2RPC,
		},
		{
			name: "success with empty token",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, util *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				util.EXPECT().TimeGetCurTime().Return("2026-01-01 00:00:00.000000")
				req.EXPECT().ContactV1ContactList(ctx, "2026-01-01 00:00:00.000000", uint64(4), typed).Return(list, nil)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.ContactList(ctx, admin, 4, "", filters)
			},
			wantRes: []*cmcontact.WebhookMessage{list[0].ConvertWebhookMessage()},
		},
	})
}

func Test_convertContactFilters_h2(t *testing.T) {
	h := &serviceHandler{}
	res, err := h.convertContactFilters(map[string]string{"customer_id": h2CustomerID.String(), "first_name": "x"})
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if len(res) != 2 || res[cmcontact.FieldCustomerID] != h2CustomerID || res[cmcontact.Field("first_name")] != "x" {
		t.Errorf("Wrong match. got: %v", res)
	}
}
