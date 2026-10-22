package servicehandler

import (
	"context"
	"testing"
	"time"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/dbhandler"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	bmaccount "monorepo/bin-billing-manager/models/account"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	cscustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gofrs/uuid"
)

var h2BillingAccountID = uuid.FromStringOrNil("a2000000-0000-0000-0000-000000000b01")

func h2BillingAccount() *bmaccount.Account {
	return &bmaccount.Account{
		Identity: commonidentity.Identity{ID: h2BillingAccountID, CustomerID: h2CustomerID},
		Name:     "ba",
	}
}

// h2BillingSuperCases builds the guard rows for handlers reserved to the project super admin.
func h2BillingSuperCases(
	call func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) (any, error),
	rpc func(ctx context.Context, req *requesthandler.MockRequestHandler, failure bool),
	wantRes any,
) []h2Case {
	super := h2AgentIdentity(h2CustomerID, amagent.PermissionProjectSuperAdmin)
	return []h2Case{
		{
			name: "direct access rejected",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return call(ctx, h, h2DirectIdentity(h2CustomerID))
			},
			wantErr: serviceerrors.ErrDirectAccessNotSupported,
		},
		{
			name: "customer admin is not a super admin",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return call(ctx, h, h2AgentIdentity(h2CustomerID, amagent.PermissionCustomerAdmin))
			},
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name: "rpc error",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				rpc(ctx, req, true)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return call(ctx, h, super)
			},
			wantErr: errH2RPC,
		},
		{
			name: "success",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				rpc(ctx, req, false)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return call(ctx, h, super)
			},
			wantRes: wantRes,
		},
	}
}

func h2BillingRet(failure bool) (*bmaccount.Account, error) {
	if failure {
		return nil, errH2RPC
	}
	return h2BillingAccount(), nil
}

func Test_BillingAccountGet_h2(t *testing.T) {
	h2RunCases(t, h2BillingSuperCases(
		func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) (any, error) {
			return h.BillingAccountGet(ctx, a, h2BillingAccountID)
		},
		func(ctx context.Context, req *requesthandler.MockRequestHandler, failure bool) {
			req.EXPECT().BillingV1AccountGet(ctx, h2BillingAccountID).Return(h2BillingRet(failure))
		},
		h2BillingAccount(),
	))
}

func Test_BillingAccountUpdateBasicInfo_h2(t *testing.T) {
	h2RunCases(t, h2BillingSuperCases(
		func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) (any, error) {
			return h.BillingAccountUpdateBasicInfo(ctx, a, h2BillingAccountID, "nm", "dt")
		},
		func(ctx context.Context, req *requesthandler.MockRequestHandler, failure bool) {
			req.EXPECT().BillingV1AccountUpdateBasicInfo(ctx, h2BillingAccountID, "nm", "dt").Return(h2BillingRet(failure))
		},
		h2BillingAccount(),
	))
}

func Test_BillingAccountUpdatePaymentInfo_h2(t *testing.T) {
	h2RunCases(t, h2BillingSuperCases(
		func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) (any, error) {
			return h.BillingAccountUpdatePaymentInfo(ctx, a, h2BillingAccountID, bmaccount.PaymentTypePrepaid, bmaccount.PaymentMethodCreditCard)
		},
		func(ctx context.Context, req *requesthandler.MockRequestHandler, failure bool) {
			req.EXPECT().BillingV1AccountUpdatePaymentInfo(ctx, h2BillingAccountID, bmaccount.PaymentTypePrepaid, bmaccount.PaymentMethodCreditCard).Return(h2BillingRet(failure))
		},
		h2BillingAccount(),
	))
}

func Test_BillingAccountAddBalanceForce_h2(t *testing.T) {
	h2RunCases(t, h2BillingSuperCases(
		func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) (any, error) {
			return h.BillingAccountAddBalanceForce(ctx, a, h2BillingAccountID, 1500)
		},
		func(ctx context.Context, req *requesthandler.MockRequestHandler, failure bool) {
			req.EXPECT().BillingV1AccountAddBalanceForce(ctx, h2BillingAccountID, int64(1500)).Return(h2BillingRet(failure))
		},
		h2BillingAccount(),
	))
}

func Test_BillingAccountSubtractBalanceForce_h2(t *testing.T) {
	h2RunCases(t, h2BillingSuperCases(
		func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) (any, error) {
			return h.BillingAccountSubtractBalanceForce(ctx, a, h2BillingAccountID, 700)
		},
		func(ctx context.Context, req *requesthandler.MockRequestHandler, failure bool) {
			req.EXPECT().BillingV1AccountSubtractBalanceForce(ctx, h2BillingAccountID, int64(700)).Return(h2BillingRet(failure))
		},
		h2BillingAccount(),
	))
}

// h2BillingSelfCases builds the rows of the customer-admin "self" handlers.
func h2BillingSelfCases(
	call func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) (any, error),
	rpc func(ctx context.Context, req *requesthandler.MockRequestHandler, failure bool),
	wantRes any,
	customerErr error,
	rpcErr error,
) []h2Case {
	admin := h2AgentIdentity(h2CustomerID, amagent.PermissionCustomerAdmin)
	customerWith := func(billingID uuid.UUID) *cscustomer.Customer {
		return &cscustomer.Customer{ID: h2CustomerID, BillingAccountID: billingID}
	}
	return []h2Case{
		{
			name: "direct access rejected",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return call(ctx, h, h2DirectIdentity(h2CustomerID))
			},
			wantErr: serviceerrors.ErrDirectAccessNotSupported,
		},
		{
			name: "manager is not enough",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return call(ctx, h, h2AgentIdentity(h2CustomerID, amagent.PermissionCustomerManager))
			},
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name: "customer lookup fails",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				req.EXPECT().CustomerV1CustomerGet(ctx, h2CustomerID).Return(nil, errH2RPC)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return call(ctx, h, admin)
			},
			wantErr: customerErr,
		},
		{
			name: "customer without billing account",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				req.EXPECT().CustomerV1CustomerGet(ctx, h2CustomerID).Return(customerWith(uuid.Nil), nil)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return call(ctx, h, admin)
			},
			wantErr: serviceerrors.ErrStateInvalid,
		},
		{
			name: "rpc error",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				req.EXPECT().CustomerV1CustomerGet(ctx, h2CustomerID).Return(customerWith(h2BillingAccountID), nil)
				rpc(ctx, req, true)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return call(ctx, h, admin)
			},
			wantErr: rpcErr,
		},
		{
			name: "success",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				req.EXPECT().CustomerV1CustomerGet(ctx, h2CustomerID).Return(customerWith(h2BillingAccountID), nil)
				rpc(ctx, req, false)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return call(ctx, h, admin)
			},
			wantRes: wantRes,
		},
	}
}

func Test_BillingAccountSelfGet_h2(t *testing.T) {
	cases := h2BillingSelfCases(
		func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) (any, error) {
			return h.BillingAccountSelfGet(ctx, a)
		},
		func(ctx context.Context, req *requesthandler.MockRequestHandler, failure bool) {
			req.EXPECT().BillingV1AccountGet(ctx, h2BillingAccountID).Return(h2BillingRet(failure))
		},
		h2BillingAccount().ConvertWebhookMessage(),
		errH2RPC,
		errH2RPC,
	)
	h2RunCases(t, cases)
}

func Test_BillingAccountSelfGet_h2_deleted(t *testing.T) {
	admin := h2AgentIdentity(h2CustomerID, amagent.PermissionCustomerAdmin)
	h2RunCases(t, []h2Case{
		{
			name: "deleted billing account is not found",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				deleted := h2BillingAccount()
				ts := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
				deleted.TMDelete = &ts
				req.EXPECT().CustomerV1CustomerGet(ctx, h2CustomerID).Return(&cscustomer.Customer{ID: h2CustomerID, BillingAccountID: h2BillingAccountID}, nil)
				req.EXPECT().BillingV1AccountGet(ctx, h2BillingAccountID).Return(deleted, nil)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.BillingAccountSelfGet(ctx, admin)
			},
			wantErr: serviceerrors.ErrNotFound,
		},
	})
}

func Test_BillingAccountSelfUpdateBasicInfo_h2(t *testing.T) {
	h2RunCases(t, h2BillingSelfCases(
		func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) (any, error) {
			return h.BillingAccountSelfUpdateBasicInfo(ctx, a, "nm", "dt")
		},
		func(ctx context.Context, req *requesthandler.MockRequestHandler, failure bool) {
			req.EXPECT().BillingV1AccountUpdateBasicInfo(ctx, h2BillingAccountID, "nm", "dt").Return(h2BillingRet(failure))
		},
		h2BillingAccount().ConvertWebhookMessage(),
		errH2RPC,
		errH2RPC,
	))
}

func Test_BillingAccountSelfUpdatePaymentInfo_h2(t *testing.T) {
	h2RunCases(t, h2BillingSelfCases(
		func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) (any, error) {
			return h.BillingAccountSelfUpdatePaymentInfo(ctx, a, bmaccount.PaymentTypePrepaid, bmaccount.PaymentMethodCreditCard)
		},
		func(ctx context.Context, req *requesthandler.MockRequestHandler, failure bool) {
			req.EXPECT().BillingV1AccountUpdatePaymentInfo(ctx, h2BillingAccountID, bmaccount.PaymentTypePrepaid, bmaccount.PaymentMethodCreditCard).Return(h2BillingRet(failure))
		},
		h2BillingAccount().ConvertWebhookMessage(),
		errH2RPC,
		errH2RPC,
	))
}

func Test_BillingAccountSelfCreatePaddlePortalSession_h2(t *testing.T) {
	cases := h2BillingSelfCases(
		func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) (any, error) {
			return h.BillingAccountSelfCreatePaddlePortalSession(ctx, a)
		},
		func(ctx context.Context, req *requesthandler.MockRequestHandler, failure bool) {
			if failure {
				req.EXPECT().BillingV1AccountPaddlePortalSession(ctx, h2BillingAccountID).Return("", errH2RPC)
				return
			}
			req.EXPECT().BillingV1AccountPaddlePortalSession(ctx, h2BillingAccountID).Return("https://portal.example.com/s/1", nil)
		},
		"https://portal.example.com/s/1",
		serviceerrors.ErrInternal,
		serviceerrors.ErrInternal,
	)
	h2RunCases(t, cases)
}

func Test_BillingAccountList_h2(t *testing.T) {
	super := h2AgentIdentity(h2CustomerID, amagent.PermissionProjectSuperAdmin)
	filters := map[string]string{"customer_id": h2CustomerID.String(), "deleted": "false"}
	typed := map[bmaccount.Field]any{
		bmaccount.FieldCustomerID: h2CustomerID,
		bmaccount.FieldDeleted:    false,
	}
	acc := *h2BillingAccount()

	h2RunCases(t, []h2Case{
		{
			name: "direct access rejected",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.BillingAccountList(ctx, h2DirectIdentity(h2CustomerID), 10, "tok", filters)
			},
			wantErr: serviceerrors.ErrDirectAccessNotSupported,
		},
		{
			name: "customer admin is not a super admin",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.BillingAccountList(ctx, h2AgentIdentity(h2CustomerID, amagent.PermissionCustomerAdmin), 10, "tok", filters)
			},
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name: "rpc error",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				req.EXPECT().BillingV1AccountGets(ctx, "tok", uint64(10), typed).Return(nil, errH2RPC)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.BillingAccountList(ctx, super, 10, "tok", filters)
			},
			wantErr: errH2RPC,
		},
		{
			name: "success with default size and token",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, util *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				util.EXPECT().TimeGetCurTime().Return("2026-01-01 00:00:00.000000")
				req.EXPECT().BillingV1AccountGets(ctx, "2026-01-01 00:00:00.000000", uint64(10), typed).Return([]bmaccount.Account{acc}, nil)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.BillingAccountList(ctx, super, 0, "", filters)
			},
			wantRes: []*bmaccount.Account{&acc},
		},
	})
}

func Test_convertBillingAccountFilters_h2(t *testing.T) {
	h := &serviceHandler{}
	res, err := h.convertBillingAccountFilters(map[string]string{"customer_id": h2CustomerID.String()})
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if len(res) != 1 || res[bmaccount.FieldCustomerID] != h2CustomerID {
		t.Errorf("Wrong match. got: %v", res)
	}
}
