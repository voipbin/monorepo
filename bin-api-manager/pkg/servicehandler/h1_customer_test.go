package servicehandler

import (
	"context"
	"testing"

	cmoutboundconfig "monorepo/bin-call-manager/models/outboundconfig"
	cscustomer "monorepo/bin-customer-manager/models/customer"

	csaccesskey "monorepo/bin-customer-manager/models/accesskey"

	bmaccount "monorepo/bin-billing-manager/models/account"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"monorepo/bin-api-manager/pkg/serviceerrors"
)

func h1Cust(id string) *cscustomer.Customer {
	return &cscustomer.Customer{ID: h1UUID(id), Name: "cust"}
}

func h1CustWebhook(id string) *cscustomer.WebhookMessage {
	return &cscustomer.WebhookMessage{ID: h1UUID(id), Name: "cust"}
}

// h1CustSuperCases builds the table for customer operations restricted to
// project super admin: direct access -> permission -> [get] -> RPC.
func h1CustSuperCases(
	withGet bool,
	call func(ctx context.Context, h *serviceHandler, a *h1Identity) (any, error),
	expectAct func(m *h1Mocks, ret *cscustomer.Customer, err error),
	prepare func(m *h1Mocks),
) []h1Case {
	id := h1UUID(h1ResourceID)
	expectGet := func(m *h1Mocks) {
		if withGet {
			m.req.EXPECT().CustomerV1CustomerGet(h1Ctx(), id).Return(h1Cust(h1ResourceID), nil)
		}
		if prepare != nil {
			prepare(m)
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
		{
			name:    "no permission",
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return call(ctx, h, h1NoPerm()) },
			wantErr: serviceerrors.ErrPermissionDenied,
		},
	}
	if withGet {
		tests = append(tests, h1Case{
			name:    "get error",
			setup:   func(m *h1Mocks) { m.req.EXPECT().CustomerV1CustomerGet(h1Ctx(), id).Return(nil, errH1RPC) },
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return call(ctx, h, h1Super()) },
			wantErr: errH1RPC,
		})
	}
	tests = append(tests,
		h1Case{
			name: "action rpc error",
			setup: func(m *h1Mocks) {
				expectGet(m)
				expectAct(m, nil, errH1RPC)
			},
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return call(ctx, h, h1Super()) },
			wantErr: errH1RPC,
		},
		h1Case{
			name: "success",
			setup: func(m *h1Mocks) {
				expectGet(m)
				expectAct(m, h1Cust(h1ResourceID), nil)
			},
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return call(ctx, h, h1Super()) },
			wantRes: h1CustWebhook(h1ResourceID),
		},
	)

	return tests
}

// h1CustSelfCases builds the table for customer operations on the caller's own
// customer, restricted to customer admin.
func h1CustSelfCases(
	withGet bool,
	call func(ctx context.Context, h *serviceHandler, a *h1Identity) (any, error),
	expectAct func(m *h1Mocks, ret *cscustomer.Customer, err error),
	prepare func(m *h1Mocks),
) []h1Case {
	id := h1UUID(h1CustomerID)
	expectGet := func(m *h1Mocks) {
		if withGet {
			m.req.EXPECT().CustomerV1CustomerGet(h1Ctx(), id).Return(h1Cust(h1CustomerID), nil)
		}
		if prepare != nil {
			prepare(m)
		}
	}

	tests := []h1Case{
		{
			name:    "direct access",
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return call(ctx, h, h1Direct()) },
			wantErr: serviceerrors.ErrDirectAccessNotSupported,
		},
		{
			name:    "manager is not admin",
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return call(ctx, h, h1Manager()) },
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name:    "no permission",
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return call(ctx, h, h1NoPerm()) },
			wantErr: serviceerrors.ErrPermissionDenied,
		},
	}
	if withGet {
		tests = append(tests, h1Case{
			name:    "get error",
			setup:   func(m *h1Mocks) { m.req.EXPECT().CustomerV1CustomerGet(h1Ctx(), id).Return(nil, errH1RPC) },
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return call(ctx, h, h1Admin()) },
			wantErr: errH1RPC,
		})
	}
	tests = append(tests,
		h1Case{
			name: "action rpc error",
			setup: func(m *h1Mocks) {
				expectGet(m)
				expectAct(m, nil, errH1RPC)
			},
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return call(ctx, h, h1Admin()) },
			wantErr: errH1RPC,
		},
		h1Case{
			name: "success",
			setup: func(m *h1Mocks) {
				expectGet(m)
				expectAct(m, h1Cust(h1CustomerID), nil)
			},
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return call(ctx, h, h1Admin()) },
			wantRes: h1CustWebhook(h1CustomerID),
		},
	)

	return tests
}

func Test_CustomerCreate_Matrix(t *testing.T) {
	method := cscustomer.WebhookMethod("POST")
	call := func(ctx context.Context, h *serviceHandler, a *h1Identity) (any, error) {
		return h.CustomerCreate(ctx, a, "n", "d", "a@example.com", "+15555550100", "addr", method, "https://example.com/hook")
	}
	created := h1Cust(h1ResourceID)
	rid := h1UUID(h1ResourceID)

	tests := h1CustSuperCases(false, call, func(m *h1Mocks, ret *cscustomer.Customer, err error) {
		m.req.EXPECT().CustomerV1CustomerCreate(h1Ctx(), 30000, "n", "d", "a@example.com", "+15555550100", "addr", method, "https://example.com/hook").Return(ret, err)
		if ret != nil {
			m.req.EXPECT().CallV1OutboundConfigCreate(h1Ctx(), rid, &cmoutboundconfig.UpdateRequest{}).Return(&cmoutboundconfig.OutboundConfig{}, nil)
		}
	}, nil)

	tests = append(tests, h1Case{
		name: "outbound config failure rolls the customer back",
		setup: func(m *h1Mocks) {
			m.req.EXPECT().CustomerV1CustomerCreate(h1Ctx(), 30000, "n", "d", "a@example.com", "+15555550100", "addr", method, "https://example.com/hook").Return(created, nil)
			m.req.EXPECT().CallV1OutboundConfigCreate(h1Ctx(), rid, &cmoutboundconfig.UpdateRequest{}).Return(nil, errH1RPC).Times(3)
			m.req.EXPECT().CallV1OutboundConfigList(h1Ctx(), rid, uint64(1), "").Return(nil, errH1RPC).Times(3)
			m.req.EXPECT().CustomerV1CustomerDelete(h1Ctx(), rid).Return(nil, errH1RPC)
		},
		call:    func(ctx context.Context, h *serviceHandler) (any, error) { return call(ctx, h, h1Super()) },
		wantErr: errH1RPC,
	})

	h1Run(t, tests)
}

func Test_autoCreateOutboundConfigWithRetry_Paths(t *testing.T) {
	rid := h1UUID(h1ResourceID)
	req := &cmoutboundconfig.UpdateRequest{}

	h1Run(t, []h1Case{
		{
			name: "existing config after create failure counts as success",
			setup: func(m *h1Mocks) {
				m.req.EXPECT().CallV1OutboundConfigCreate(h1Ctx(), rid, req).Return(nil, errH1RPC)
				m.req.EXPECT().CallV1OutboundConfigList(h1Ctx(), rid, uint64(1), "").Return([]cmoutboundconfig.OutboundConfig{{}}, nil)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return nil, h.autoCreateOutboundConfigWithRetry(ctx, rid)
			},
		},
		{
			name: "cancelled context stops the retry",
			setup: func(m *h1Mocks) {
				m.req.EXPECT().CallV1OutboundConfigCreate(h1Ctx(), rid, req).Return(nil, errH1RPC)
				m.req.EXPECT().CallV1OutboundConfigList(h1Ctx(), rid, uint64(1), "").Return(nil, errH1RPC)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				cctx, cancel := context.WithCancel(ctx)
				cancel()
				return nil, h.autoCreateOutboundConfigWithRetry(cctx, rid)
			},
			wantErr: context.Canceled,
		},
	})
}

func Test_CustomerGet_Matrix(t *testing.T) {
	id := h1UUID(h1ResourceID)
	h1Run(t, h1CustSuperCases(false,
		func(ctx context.Context, h *serviceHandler, a *h1Identity) (any, error) {
			return h.CustomerGet(ctx, a, id)
		},
		func(m *h1Mocks, ret *cscustomer.Customer, err error) {
			m.req.EXPECT().CustomerV1CustomerGet(h1Ctx(), id).Return(ret, err)
		}, nil))
}

func Test_CustomerSelfGet_Matrix(t *testing.T) {
	id := h1UUID(h1CustomerID)
	tests := []h1Case{
		{
			name:    "direct access",
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return h.CustomerSelfGet(ctx, h1Direct()) },
			wantErr: serviceerrors.ErrDirectAccessNotSupported,
		},
		{
			name:    "no permission",
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return h.CustomerSelfGet(ctx, h1NoPerm()) },
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name:    "rpc error",
			setup:   func(m *h1Mocks) { m.req.EXPECT().CustomerV1CustomerGet(h1Ctx(), id).Return(nil, errH1RPC) },
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return h.CustomerSelfGet(ctx, h1Manager()) },
			wantErr: errH1RPC,
		},
		{
			name: "success includes the webhook secret",
			setup: func(m *h1Mocks) {
				c := h1Cust(h1CustomerID)
				c.WebhookSecret = "secret-value"
				m.req.EXPECT().CustomerV1CustomerGet(h1Ctx(), id).Return(c, nil)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) { return h.CustomerSelfGet(ctx, h1Manager()) },
			wantRes: &cscustomer.SelfWebhookMessage{
				WebhookMessage: *h1CustWebhook(h1CustomerID),
				WebhookSecret:  "secret-value",
			},
		},
	}
	h1Run(t, tests)
}

func Test_CustomerRawSelfGet_Matrix(t *testing.T) {
	id := h1UUID(h1CustomerID)
	h1Run(t, []h1Case{
		{
			name: "direct access",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.CustomerRawSelfGet(ctx, h1Direct())
			},
			wantErr: serviceerrors.ErrDirectAccessNotSupported,
		},
		{
			name:  "rpc error",
			setup: func(m *h1Mocks) { m.req.EXPECT().CustomerV1CustomerGet(h1Ctx(), id).Return(nil, errH1RPC) },
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.CustomerRawSelfGet(ctx, h1NoPerm())
			},
			wantErr: errH1RPC,
		},
		{
			name:  "success without permission requirement",
			setup: func(m *h1Mocks) { m.req.EXPECT().CustomerV1CustomerGet(h1Ctx(), id).Return(h1Cust(h1CustomerID), nil) },
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.CustomerRawSelfGet(ctx, h1NoPerm())
			},
			wantRes: h1CustWebhook(h1CustomerID),
		},
	})
}

func Test_CustomerList_Matrix(t *testing.T) {
	typed := map[cscustomer.Field]any{cscustomer.FieldDeleted: false}
	filters := map[string]string{"deleted": "false"}

	h1Run(t, []h1Case{
		{
			name: "direct access",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.CustomerList(ctx, h1Direct(), 10, "t", filters)
			},
			wantErr: serviceerrors.ErrDirectAccessNotSupported,
		},
		{
			name: "customer admin is not super admin",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.CustomerList(ctx, h1Admin(), 10, "t", filters)
			},
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name: "invalid filter",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.CustomerList(ctx, h1Super(), 10, "t", map[string]string{"id": "not-a-uuid"})
			},
			wantErrText: "invalid UUID string",
		},
	})

	h1Run(t, []h1Case{
		{
			name: "default size and token, rpc error",
			setup: func(m *h1Mocks) {
				m.util.EXPECT().TimeGetCurTime().Return("2026-01-01T00:00:00.000000Z")
				m.req.EXPECT().CustomerV1CustomerList(h1Ctx(), "2026-01-01T00:00:00.000000Z", uint64(10), typed).Return(nil, errH1RPC)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.CustomerList(ctx, h1Super(), 0, "", filters)
			},
			wantErr: errH1RPC,
		},
		{
			name: "success",
			setup: func(m *h1Mocks) {
				m.req.EXPECT().CustomerV1CustomerList(h1Ctx(), "tok", uint64(5), typed).Return([]cscustomer.Customer{*h1Cust(h1ResourceID), *h1Cust(h1CustomerID)}, nil)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.CustomerList(ctx, h1Super(), 5, "tok", filters)
			},
			wantRes: []*cscustomer.WebhookMessage{
				h1CustWebhook(h1ResourceID),
				h1CustWebhook(h1CustomerID),
			},
		},
	})
}

func Test_convertCustomerFilters_Invalid(t *testing.T) {
	h := &serviceHandler{}

	res, err := h.convertCustomerFilters(map[string]string{"id": "not-a-uuid"})
	if err == nil {
		t.Fatalf("Wrong match. expect: error, got: nil (res: %v)", res)
	}
	if res != nil {
		t.Errorf("Wrong result. expect: nil, got: %v", res)
	}
}

func Test_CustomerUpdate_Matrix(t *testing.T) {
	id := h1UUID(h1ResourceID)
	name, detail, email, phone, addr, uri := "n", "d", "a@example.com", "+15555550100", "addr", "https://example.com/hook"
	method := cscustomer.WebhookMethod("POST")

	h1Run(t, h1CustSuperCases(true,
		func(ctx context.Context, h *serviceHandler, a *h1Identity) (any, error) {
			return h.CustomerUpdate(ctx, a, id, &name, &detail, &email, &phone, &addr, &method, &uri)
		},
		func(m *h1Mocks, ret *cscustomer.Customer, err error) {
			m.req.EXPECT().CustomerV1CustomerUpdate(h1Ctx(), id, &name, &detail, &email, &phone, &addr, &method, &uri).Return(ret, err)
		}, nil))
}

func Test_CustomerSelfUpdate_Matrix(t *testing.T) {
	id := h1UUID(h1CustomerID)
	name, detail, email, phone, addr, uri := "n", "d", "a@example.com", "+15555550100", "addr", "https://example.com/hook"
	method := cscustomer.WebhookMethod("POST")

	h1Run(t, h1CustSelfCases(false,
		func(ctx context.Context, h *serviceHandler, a *h1Identity) (any, error) {
			return h.CustomerSelfUpdate(ctx, a, &name, &detail, &email, &phone, &addr, &method, &uri)
		},
		func(m *h1Mocks, ret *cscustomer.Customer, err error) {
			m.req.EXPECT().CustomerV1CustomerUpdate(h1Ctx(), id, &name, &detail, &email, &phone, &addr, &method, &uri).Return(ret, err)
		}, nil))
}

func Test_CustomerDelete_Matrix(t *testing.T) {
	id := h1UUID(h1ResourceID)
	h1Run(t, h1CustSuperCases(true,
		func(ctx context.Context, h *serviceHandler, a *h1Identity) (any, error) {
			return h.CustomerDelete(ctx, a, id)
		},
		func(m *h1Mocks, ret *cscustomer.Customer, err error) {
			m.req.EXPECT().CustomerV1CustomerDelete(h1Ctx(), id).Return(ret, err)
		}, nil))
}

func Test_CustomerFreeze_Matrix(t *testing.T) {
	id := h1UUID(h1ResourceID)
	h1Run(t, h1CustSuperCases(true,
		func(ctx context.Context, h *serviceHandler, a *h1Identity) (any, error) {
			return h.CustomerFreeze(ctx, a, id)
		},
		func(m *h1Mocks, ret *cscustomer.Customer, err error) {
			m.req.EXPECT().CustomerV1CustomerFreeze(h1Ctx(), id).Return(ret, err)
		}, nil))
}

func Test_CustomerRecover_Matrix(t *testing.T) {
	id := h1UUID(h1ResourceID)
	h1Run(t, h1CustSuperCases(true,
		func(ctx context.Context, h *serviceHandler, a *h1Identity) (any, error) {
			return h.CustomerRecover(ctx, a, id)
		},
		func(m *h1Mocks, ret *cscustomer.Customer, err error) {
			m.req.EXPECT().CustomerV1CustomerRecover(h1Ctx(), id).Return(ret, err)
		}, nil))
}

func Test_CustomerSelfFreeze_Matrix(t *testing.T) {
	id := h1UUID(h1CustomerID)
	h1Run(t, h1CustSelfCases(true,
		func(ctx context.Context, h *serviceHandler, a *h1Identity) (any, error) {
			return h.CustomerSelfFreeze(ctx, a)
		},
		func(m *h1Mocks, ret *cscustomer.Customer, err error) {
			m.req.EXPECT().CustomerV1CustomerFreeze(h1Ctx(), id).Return(ret, err)
		}, nil))
}

func Test_CustomerSelfFreezeAndDelete_Matrix(t *testing.T) {
	id := h1UUID(h1CustomerID)
	h1Run(t, h1CustSelfCases(true,
		func(ctx context.Context, h *serviceHandler, a *h1Identity) (any, error) {
			return h.CustomerSelfFreezeAndDelete(ctx, a)
		},
		func(m *h1Mocks, ret *cscustomer.Customer, err error) {
			m.req.EXPECT().CustomerV1CustomerFreezeAndDelete(h1Ctx(), id).Return(ret, err)
		}, nil))
}

func Test_CustomerSelfRecover_Matrix(t *testing.T) {
	id := h1UUID(h1CustomerID)
	h1Run(t, h1CustSelfCases(true,
		func(ctx context.Context, h *serviceHandler, a *h1Identity) (any, error) {
			return h.CustomerSelfRecover(ctx, a)
		},
		func(m *h1Mocks, ret *cscustomer.Customer, err error) {
			m.req.EXPECT().CustomerV1CustomerRecover(h1Ctx(), id).Return(ret, err)
		}, nil))
}

func Test_CustomerUpdateMetadata_Matrix(t *testing.T) {
	id := h1UUID(h1ResourceID)
	md := cscustomer.Metadata{RTPDebug: true}
	h1Run(t, h1CustSuperCases(true,
		func(ctx context.Context, h *serviceHandler, a *h1Identity) (any, error) {
			return h.CustomerUpdateMetadata(ctx, a, id, md)
		},
		func(m *h1Mocks, ret *cscustomer.Customer, err error) {
			m.req.EXPECT().CustomerV1CustomerUpdateMetadata(h1Ctx(), id, md).Return(ret, err)
		}, nil))
}

func Test_CustomerSelfUpdateMetadata_Matrix(t *testing.T) {
	id := h1UUID(h1CustomerID)
	md := cscustomer.Metadata{RTPDebug: true}
	h1Run(t, h1CustSelfCases(false,
		func(ctx context.Context, h *serviceHandler, a *h1Identity) (any, error) {
			return h.CustomerSelfUpdateMetadata(ctx, a, md)
		},
		func(m *h1Mocks, ret *cscustomer.Customer, err error) {
			m.req.EXPECT().CustomerV1CustomerUpdateMetadata(h1Ctx(), id, md).Return(ret, err)
		}, nil))
}

func Test_CustomerUpdateBillingAccountID_Matrix(t *testing.T) {
	id := h1UUID(h1ResourceID)
	baID := h1UUID(h1AgentID)
	ba := &bmaccount.Account{Identity: commonidentity.Identity{ID: baID, CustomerID: id}}

	tests := h1CustSuperCases(true,
		func(ctx context.Context, h *serviceHandler, a *h1Identity) (any, error) {
			return h.CustomerUpdateBillingAccountID(ctx, a, id, baID)
		},
		func(m *h1Mocks, ret *cscustomer.Customer, err error) {
			m.req.EXPECT().CustomerV1CustomerUpdateBillingAccountID(h1Ctx(), id, baID).Return(ret, err)
		},
		func(m *h1Mocks) { m.req.EXPECT().BillingV1AccountGet(h1Ctx(), baID).Return(ba, nil) })

	tests = append(tests, h1Case{
		name: "billing account lookup error",
		setup: func(m *h1Mocks) {
			m.req.EXPECT().CustomerV1CustomerGet(h1Ctx(), id).Return(h1Cust(h1ResourceID), nil)
			m.req.EXPECT().BillingV1AccountGet(h1Ctx(), baID).Return(nil, errH1RPC)
		},
		call: func(ctx context.Context, h *serviceHandler) (any, error) {
			return h.CustomerUpdateBillingAccountID(ctx, h1Super(), id, baID)
		},
		wantErr: errH1RPC,
	})

	h1Run(t, tests)
}

func Test_CustomerSelfUpdateBillingAccountID_Matrix(t *testing.T) {
	baID := h1UUID(h1AgentID)
	own := &bmaccount.Account{Identity: commonidentity.Identity{ID: baID, CustomerID: h1UUID(h1CustomerID)}}
	foreign := &bmaccount.Account{Identity: commonidentity.Identity{ID: baID, CustomerID: h1UUID(h1OtherID)}}

	tests := h1CustSelfCases(false,
		func(ctx context.Context, h *serviceHandler, a *h1Identity) (any, error) {
			return h.CustomerSelfUpdateBillingAccountID(ctx, a, baID)
		},
		func(m *h1Mocks, ret *cscustomer.Customer, err error) {
			m.req.EXPECT().CustomerV1CustomerUpdateBillingAccountID(h1Ctx(), h1UUID(h1CustomerID), baID).Return(ret, err)
		},
		func(m *h1Mocks) { m.req.EXPECT().BillingV1AccountGet(h1Ctx(), baID).Return(own, nil) })

	tests = append(tests,
		h1Case{
			name:  "billing account lookup error",
			setup: func(m *h1Mocks) { m.req.EXPECT().BillingV1AccountGet(h1Ctx(), baID).Return(nil, errH1RPC) },
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.CustomerSelfUpdateBillingAccountID(ctx, h1Admin(), baID)
			},
			wantErr: errH1RPC,
		},
		h1Case{
			name:  "billing account of another customer",
			setup: func(m *h1Mocks) { m.req.EXPECT().BillingV1AccountGet(h1Ctx(), baID).Return(foreign, nil) },
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.CustomerSelfUpdateBillingAccountID(ctx, h1Admin(), baID)
			},
			wantErr: serviceerrors.ErrPermissionDenied,
		},
	)

	h1Run(t, tests)
}

func Test_CustomerSignup_Matrix(t *testing.T) {
	method := cscustomer.WebhookMethod("POST")
	rid := h1UUID(h1ResourceID)
	signup := func(ctx context.Context, h *serviceHandler) (any, error) {
		return h.CustomerSignup(ctx, "n", "d", "a@example.com", "+15555550100", "addr", method, "https://example.com/hook", "client-ip-1")
	}
	expectSignup := func(m *h1Mocks, ret *cscustomer.SignupResult, err error) {
		m.req.EXPECT().CustomerV1CustomerSignup(h1Ctx(), "n", "d", "a@example.com", "+15555550100", "addr", method, "https://example.com/hook", "client-ip-1").Return(ret, err)
	}

	h1Run(t, []h1Case{
		{
			name:    "rpc error",
			setup:   func(m *h1Mocks) { expectSignup(m, nil, errH1RPC) },
			call:    signup,
			wantErr: errH1RPC,
		},
		{
			name: "outbound config failure rolls the customer back",
			setup: func(m *h1Mocks) {
				expectSignup(m, &cscustomer.SignupResult{Customer: h1Cust(h1ResourceID)}, nil)
				m.req.EXPECT().CallV1OutboundConfigCreate(h1Ctx(), rid, &cmoutboundconfig.UpdateRequest{}).Return(nil, errH1RPC).Times(3)
				m.req.EXPECT().CallV1OutboundConfigList(h1Ctx(), rid, uint64(1), "").Return(nil, errH1RPC).Times(3)
				m.req.EXPECT().CustomerV1CustomerDelete(h1Ctx(), rid).Return(h1Cust(h1ResourceID), nil)
			},
			call:    signup,
			wantErr: errH1RPC,
		},
		{
			name: "success with accesskey",
			setup: func(m *h1Mocks) {
				expectSignup(m, &cscustomer.SignupResult{
					Customer:  h1Cust(h1ResourceID),
					Accesskey: &csaccesskey.Accesskey{ID: h1UUID(h1AgentID), CustomerID: rid, RawToken: "raw-token"},
				}, nil)
				m.req.EXPECT().CallV1OutboundConfigCreate(h1Ctx(), rid, &cmoutboundconfig.UpdateRequest{}).Return(&cmoutboundconfig.OutboundConfig{}, nil)
			},
			call: signup,
			wantRes: &cscustomer.SignupResultWebhookMessage{
				Customer:  h1CustWebhook(h1ResourceID),
				Accesskey: &csaccesskey.WebhookMessage{ID: h1UUID(h1AgentID), CustomerID: rid, Token: "raw-token"},
			},
		},
		{
			name: "result without customer skips the outbound config",
			setup: func(m *h1Mocks) {
				expectSignup(m, &cscustomer.SignupResult{}, nil)
			},
			call:    signup,
			wantRes: &cscustomer.SignupResultWebhookMessage{},
		},
	})
}

func Test_CustomerEmailVerify_Matrix(t *testing.T) {
	h1Run(t, []h1Case{
		{
			name:    "rpc error",
			setup:   func(m *h1Mocks) { m.req.EXPECT().CustomerV1CustomerEmailVerify(h1Ctx(), "tok").Return(nil, errH1RPC) },
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return h.CustomerEmailVerify(ctx, "tok") },
			wantErr: errH1RPC,
		},
		{
			name: "success",
			setup: func(m *h1Mocks) {
				m.req.EXPECT().CustomerV1CustomerEmailVerify(h1Ctx(), "tok").Return(&cscustomer.EmailVerifyResult{Customer: h1Cust(h1ResourceID)}, nil)
			},
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return h.CustomerEmailVerify(ctx, "tok") },
			wantRes: &cscustomer.EmailVerifyResultWebhookMessage{Customer: h1CustWebhook(h1ResourceID)},
		},
	})
}

func Test_CustomerEmailVerifyResend_Matrix(t *testing.T) {
	h1Run(t, []h1Case{
		{
			name: "rpc error",
			setup: func(m *h1Mocks) {
				m.req.EXPECT().CustomerV1CustomerEmailVerifyResend(h1Ctx(), "a@example.com").Return(errH1RPC)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return nil, h.CustomerEmailVerifyResend(ctx, "a@example.com")
			},
			wantErr: errH1RPC,
		},
		{
			name: "success",
			setup: func(m *h1Mocks) {
				m.req.EXPECT().CustomerV1CustomerEmailVerifyResend(h1Ctx(), "a@example.com").Return(nil)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return nil, h.CustomerEmailVerifyResend(ctx, "a@example.com")
			},
		},
	})
}
