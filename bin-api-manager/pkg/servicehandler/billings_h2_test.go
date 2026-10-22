package servicehandler

import (
	"context"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/dbhandler"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	bmbilling "monorepo/bin-billing-manager/models/billing"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
)

var h2BillingID = uuid.FromStringOrNil("a2000000-0000-0000-0000-000000003001")

func h2Billing(customerID uuid.UUID) *bmbilling.Billing {
	return &bmbilling.Billing{Identity: commonidentity.Identity{ID: h2BillingID, CustomerID: customerID}}
}

func Test_BillingGet_h2(t *testing.T) {
	// BillingGet requires customer admin; a manager is not enough.
	var cases []h2Case
	for _, c := range h2StdCases(
		func(ctx context.Context, h *serviceHandler, a *auth.AuthIdentity) (any, error) {
			return h.BillingGet(ctx, a, h2BillingID)
		},
		func(ctx context.Context, req *requesthandler.MockRequestHandler, customerID uuid.UUID, failure bool) {
			if failure {
				req.EXPECT().BillingV1BillingGet(ctx, h2BillingID).Return(nil, errH2RPC)
				return
			}
			req.EXPECT().BillingV1BillingGet(ctx, h2BillingID).Return(h2Billing(customerID), nil)
		},
		func(context.Context, *requesthandler.MockRequestHandler, bool) {},
		h2Billing(h2CustomerID).ConvertWebhookMessage(),
	) {
		if c.name != "rpc error" {
			cases = append(cases, c)
		}
	}
	cases = append(cases, h2Case{
		name: "customer manager is not enough",
		setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
			req.EXPECT().BillingV1BillingGet(ctx, h2BillingID).Return(h2Billing(h2CustomerID), nil)
		},
		call: func(ctx context.Context, h *serviceHandler) (any, error) {
			return h.BillingGet(ctx, h2AgentIdentity(h2CustomerID, amagent.PermissionCustomerManager), h2BillingID)
		},
		wantErr: serviceerrors.ErrPermissionDenied,
	})
	h2RunCases(t, cases)
}

func Test_BillingList_h2(t *testing.T) {
	admin := h2AgentIdentity(h2CustomerID, amagent.PermissionCustomerAdmin)
	filters := map[bmbilling.Field]any{
		bmbilling.FieldCustomerID: h2CustomerID,
		bmbilling.FieldDeleted:    false,
	}
	list := []bmbilling.Billing{*h2Billing(h2CustomerID)}

	h2RunCases(t, []h2Case{
		{
			name: "direct access rejected",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.BillingList(ctx, h2DirectIdentity(h2CustomerID), 10, "tok")
			},
			wantErr: serviceerrors.ErrDirectAccessNotSupported,
		},
		{
			name: "manager is not enough",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.BillingList(ctx, h2AgentIdentity(h2CustomerID, amagent.PermissionCustomerManager), 10, "tok")
			},
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name: "rpc error",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				req.EXPECT().BillingV1BillingList(ctx, "tok", uint64(10), filters).Return(nil, errH2RPC)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.BillingList(ctx, admin, 10, "tok")
			},
			wantErr: errH2RPC,
		},
		{
			name: "success with empty token",
			setup: func(ctx context.Context, req *requesthandler.MockRequestHandler, util *utilhandler.MockUtilHandler, _ *dbhandler.MockDBHandler) {
				util.EXPECT().TimeGetCurTime().Return("2026-01-01 00:00:00.000000")
				req.EXPECT().BillingV1BillingList(ctx, "2026-01-01 00:00:00.000000", uint64(3), filters).Return(list, nil)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.BillingList(ctx, admin, 3, "")
			},
			wantRes: []*bmbilling.WebhookMessage{list[0].ConvertWebhookMessage()},
		},
	})
}

func Test_convertBillingFilters_h2(t *testing.T) {
	h := &serviceHandler{}
	res, err := h.convertBillingFilters(map[string]string{"customer_id": h2CustomerID.String(), "deleted": "false"})
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if len(res) != 2 || res[bmbilling.FieldCustomerID] != h2CustomerID || res[bmbilling.FieldDeleted] != false {
		t.Errorf("Wrong match. got: %v", res)
	}
}
