package accounthandler

import (
	"context"
	"errors"
	"strings"
	"testing"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	cmcustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-billing-manager/models/account"
	"monorepo/bin-billing-manager/models/billing"
	"monorepo/bin-billing-manager/pkg/dbhandler"
)

// Test_Paddle_unknown_plan_type verifies that a plan type outside the token map stops the flow
// before any token write.
func Test_Paddle_unknown_plan_type(t *testing.T) {
	unknown := account.PlanType("unlisted_plan")

	t.Run("subscription create", func(t *testing.T) {
		h, f := newPaddleErrFixture(t)

		f.notProcessed()
		f.customerLookup()
		f.mockDB.EXPECT().AccountUpdate(f.ctx, f.acc.ID, map[account.Field]any{
			account.FieldPaddleSubscriptionID: "sub_err_010",
			account.FieldPaddleCustomerID:     "paddle_cust_err_010",
		}).Return(nil)
		f.planUpdated(unknown)
		// no AccountPaddleTopUpTokens expectation: reaching it fails the test.

		err := h.PaddleSubscriptionCreate(f.ctx, f.acc.CustomerID, unknown, "sub_err_010", "paddle_cust_err_010", f.eventID)
		if err == nil || !strings.Contains(err.Error(), "unknown plan type") {
			t.Errorf("expected the unknown plan type error, got: %v", err)
		}
	})

	t.Run("subscription update", func(t *testing.T) {
		h, f := newPaddleErrFixture(t)

		f.notProcessed()
		f.mockDB.EXPECT().AccountGetByPaddleSubscriptionID(f.ctx, "sub_err_011").Return(f.acc, nil)
		f.planUpdated(unknown)
		f.statusReset(nil)

		err := h.PaddleSubscriptionUpdate(f.ctx, "sub_err_011", unknown, f.eventID)
		if err == nil || !strings.Contains(err.Error(), "unknown plan type") {
			t.Errorf("expected the unknown plan type error, got: %v", err)
		}
	})
}

// Test_Paddle_already_processed verifies that a replayed event is acknowledged without any write.
func Test_Paddle_already_processed(t *testing.T) {
	tests := []struct {
		name string
		call func(h *accountHandler, f *paddleErrFixture) error
	}{
		{
			name: "subscription renew",
			call: func(h *accountHandler, f *paddleErrFixture) error {
				return h.PaddleSubscriptionRenew(f.ctx, "sub_err_020", f.eventID)
			},
		},
		{
			name: "refund",
			call: func(h *accountHandler, f *paddleErrFixture) error {
				return h.PaddleRefund(f.ctx, f.acc.CustomerID, 1000, f.eventID)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, f := newPaddleErrFixture(t)

			// the event was recorded before: nothing else may be called.
			f.mockDB.EXPECT().BillingGetByIdempotencyKey(f.ctx, f.idempotencyKey).Return(&billing.Billing{}, nil)

			if err := tt.call(h, f); err != nil {
				t.Errorf("expected nil for a replayed event, got: %v", err)
			}
		})
	}
}

// Test_Delete_post_delete_lookup_not_found verifies that a not found after a successful delete is
// reported as an internal error that keeps the cause.
func Test_Delete_post_delete_lookup_not_found(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &accountHandler{db: mockDB}
	ctx := context.Background()
	id := uuid.FromStringOrNil("f1000000-0000-0000-0000-000000000001")

	mockDB.EXPECT().AccountDelete(ctx, id).Return(nil)
	mockDB.EXPECT().AccountGet(ctx, id).Return(nil, dbhandler.ErrNotFound)

	res, err := h.Delete(ctx, id)
	if res != nil {
		t.Errorf("expected no account, got: %v", res)
	}
	var ve *cerrors.VoipbinError
	if !errors.As(err, &ve) || ve.Status != cerrors.StatusInternal || ve.Reason != "POST_DELETE_LOOKUP_FAILED" {
		t.Fatalf("expected an internal POST_DELETE_LOOKUP_FAILED error, got: %v", err)
	}
	if !errors.Is(err, dbhandler.ErrNotFound) {
		t.Errorf("expected the cause to stay reachable, got: %v", err)
	}
}

// Test_IsValid_inner_failure verifies that a failure of the validation after a successful
// customer lookup is wrapped and reported as invalid.
func Test_IsValid_inner_failure(t *testing.T) {
	customerID := uuid.FromStringOrNil("f2000000-0000-0000-0000-000000000001")
	accountID := uuid.FromStringOrNil("f2000000-0000-0000-0000-000000000002")
	errInner := errors.New("inner lookup failure")

	acc := &account.Account{
		Identity: commonidentity.Identity{ID: accountID, CustomerID: customerID},
		PlanType: account.PlanTypeBasic,
		Status:   account.StatusActive,
	}
	customer := &cmcustomer.Customer{ID: customerID, BillingAccountID: accountID}

	tests := []struct {
		name       string
		expectFrag string
		call       func(h *accountHandler, ctx context.Context) (bool, error)
	}{
		{
			name:       "balance",
			expectFrag: "could not validate the account balance",
			call: func(h *accountHandler, ctx context.Context) (bool, error) {
				return h.IsValidBalanceByCustomerID(ctx, customerID, billing.ReferenceTypeCall, "", 1)
			},
		},
		{
			name:       "resource limit",
			expectFrag: "could not validate the account's resource limit",
			call: func(h *accountHandler, ctx context.Context) (bool, error) {
				return h.IsValidResourceLimitByCustomerID(ctx, customerID, account.ResourceTypeExtension)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &accountHandler{db: mockDB, reqHandler: mockReq}
			ctx := context.Background()

			// the customer lookup succeeds once, the inner account read fails with a plain error.
			mockReq.EXPECT().CustomerV1CustomerGet(ctx, customerID).Return(customer, nil)
			gomock.InOrder(
				mockDB.EXPECT().AccountGet(ctx, accountID).Return(acc, nil),
				mockDB.EXPECT().AccountGet(ctx, accountID).Return(nil, errInner),
			)

			res, err := tt.call(h, ctx)
			if res {
				t.Errorf("expected invalid, got valid")
			}
			if !errors.Is(err, errInner) || !strings.Contains(err.Error(), tt.expectFrag) {
				t.Errorf("expected %q wrapping the cause, got: %v", tt.expectFrag, err)
			}
		})
	}
}

// Test_IsValidBalance_number_with_enough_credit verifies the number purchase check passes when the
// credit covers the cost and tokens do not matter.
func Test_IsValidBalance_number_with_enough_credit(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &accountHandler{db: mockDB}
	ctx := context.Background()
	accountID := uuid.FromStringOrNil("f3000000-0000-0000-0000-000000000001")

	cost := billing.GetCostInfo(billing.CostTypeNumber).CreditPerUnit * 2
	mockDB.EXPECT().AccountGet(ctx, accountID).Return(&account.Account{
		Identity:      commonidentity.Identity{ID: accountID},
		PlanType:      account.PlanTypeBasic,
		Status:        account.StatusActive,
		BalanceCredit: cost,
	}, nil)

	res, err := h.IsValidBalance(ctx, accountID, billing.ReferenceTypeNumber, "", 2)
	if err != nil || !res {
		t.Errorf("expected valid, got: %v, err: %v", res, err)
	}

	// one unit short of the cost is rejected.
	mockDB.EXPECT().AccountGet(ctx, accountID).Return(&account.Account{
		Identity:      commonidentity.Identity{ID: accountID},
		PlanType:      account.PlanTypeBasic,
		Status:        account.StatusActive,
		BalanceCredit: cost - 1,
	}, nil)
	res, err = h.IsValidBalance(ctx, accountID, billing.ReferenceTypeNumber, "", 2)
	if err != nil || res {
		t.Errorf("expected invalid, got: %v, err: %v", res, err)
	}
}
