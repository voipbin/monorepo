package accounthandler

import (
	"context"
	"errors"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	cmcustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-billing-manager/models/account"
	"monorepo/bin-billing-manager/models/billing"
	"monorepo/bin-billing-manager/pkg/dbhandler"
)

// paddleErrFixture carries the shared values and mocks of one error-path case.
type paddleErrFixture struct {
	ctx      context.Context
	mockDB   *dbhandler.MockDBHandler
	mockReq  *requesthandler.MockRequestHandler
	mockNtfy *notifyhandler.MockNotifyHandler

	acc            *account.Account
	customer       *cmcustomer.Customer
	eventID        string
	idempotencyKey uuid.UUID
	errDB          error
}

func newPaddleErrFixture(t *testing.T) (*accountHandler, *paddleErrFixture) {
	t.Helper()

	mc := gomock.NewController(t)
	t.Cleanup(mc.Finish)

	f := &paddleErrFixture{
		ctx:      context.Background(),
		mockDB:   dbhandler.NewMockDBHandler(mc),
		mockReq:  requesthandler.NewMockRequestHandler(mc),
		mockNtfy: notifyhandler.NewMockNotifyHandler(mc),
		acc: &account.Account{
			Identity: commonidentity.Identity{
				ID:         uuid.FromStringOrNil("b3000000-0000-0000-0000-000000000001"),
				CustomerID: uuid.FromStringOrNil("a3000000-0000-0000-0000-000000000001"),
			},
			PlanType: account.PlanTypeBasic,
		},
		eventID: "evt_err_001",
		errDB:   errors.New("db failure"),
	}
	f.customer = &cmcustomer.Customer{
		ID:               f.acc.CustomerID,
		BillingAccountID: f.acc.ID,
	}
	f.idempotencyKey = uuid.NewV5(uuid.NamespaceDNS, f.eventID)

	h := &accountHandler{
		reqHandler:    f.mockReq,
		db:            f.mockDB,
		notifyHandler: f.mockNtfy,
	}
	return h, f
}

// notProcessed expects the idempotency lookup to report an unprocessed event.
func (f *paddleErrFixture) notProcessed() {
	f.mockDB.EXPECT().BillingGetByIdempotencyKey(f.ctx, f.idempotencyKey).Return(nil, dbhandler.ErrNotFound)
}

// customerLookup expects GetByCustomerID to succeed.
func (f *paddleErrFixture) customerLookup() {
	f.mockReq.EXPECT().CustomerV1CustomerGet(f.ctx, f.acc.CustomerID).Return(f.customer, nil)
	f.mockDB.EXPECT().AccountGet(f.ctx, f.acc.ID).Return(f.acc, nil)
}

// planUpdated expects UpdatePlanType to succeed for the given plan.
func (f *paddleErrFixture) planUpdated(plan account.PlanType) {
	f.mockDB.EXPECT().AccountUpdate(f.ctx, f.acc.ID, map[account.Field]any{account.FieldPlanType: plan}).Return(nil)
	f.mockDB.EXPECT().AccountGet(f.ctx, f.acc.ID).Return(f.acc, nil)
}

// statusReset expects the plan_status reset to active.
func (f *paddleErrFixture) statusReset(err error) {
	f.mockDB.EXPECT().AccountUpdate(f.ctx, f.acc.ID, map[account.Field]any{account.FieldPlanStatus: account.PlanStatusActive}).Return(err)
}

func Test_PaddleSubscriptionCreate_errorPaths(t *testing.T) {
	plan := account.PlanTypeProfessional
	paddleFields := func(f *paddleErrFixture) map[account.Field]any {
		return map[account.Field]any{
			account.FieldPaddleSubscriptionID: "sub_err_001",
			account.FieldPaddleCustomerID:     "paddle_cust_err_001",
		}
	}

	tests := []struct {
		name  string
		setup func(f *paddleErrFixture)
	}{
		{
			name: "idempotency lookup fails",
			setup: func(f *paddleErrFixture) {
				f.mockDB.EXPECT().BillingGetByIdempotencyKey(f.ctx, f.idempotencyKey).Return(nil, f.errDB)
			},
		},
		{
			name: "customer lookup fails",
			setup: func(f *paddleErrFixture) {
				f.notProcessed()
				f.mockReq.EXPECT().CustomerV1CustomerGet(f.ctx, f.acc.CustomerID).Return(nil, f.errDB)
			},
		},
		{
			name: "storing the paddle ids fails",
			setup: func(f *paddleErrFixture) {
				f.notProcessed()
				f.customerLookup()
				f.mockDB.EXPECT().AccountUpdate(f.ctx, f.acc.ID, paddleFields(f)).Return(f.errDB)
			},
		},
		{
			name: "plan type update fails",
			setup: func(f *paddleErrFixture) {
				f.notProcessed()
				f.customerLookup()
				f.mockDB.EXPECT().AccountUpdate(f.ctx, f.acc.ID, paddleFields(f)).Return(nil)
				f.mockDB.EXPECT().AccountUpdate(f.ctx, f.acc.ID, map[account.Field]any{account.FieldPlanType: plan}).Return(f.errDB)
			},
		},
		{
			name: "token top up fails",
			setup: func(f *paddleErrFixture) {
				f.notProcessed()
				f.customerLookup()
				f.mockDB.EXPECT().AccountUpdate(f.ctx, f.acc.ID, paddleFields(f)).Return(nil)
				f.planUpdated(plan)
				f.mockDB.EXPECT().AccountPaddleTopUpTokens(f.ctx, f.acc.ID, f.acc.CustomerID, account.PlanTokenMap[plan], string(plan), billing.TransactionTypeTopUp, f.idempotencyKey).Return(f.errDB)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, f := newPaddleErrFixture(t)
			tt.setup(f)

			err := h.PaddleSubscriptionCreate(f.ctx, f.acc.CustomerID, plan, "sub_err_001", "paddle_cust_err_001", f.eventID)
			if !errors.Is(err, f.errDB) {
				t.Errorf("Wrong match. expect: wrapped db error, got: %v", err)
			}
		})
	}
}

func Test_PaddleSubscriptionUpdate_errorPaths(t *testing.T) {
	plan := account.PlanTypeProfessional

	tests := []struct {
		name  string
		setup func(f *paddleErrFixture)
	}{
		{
			name: "idempotency lookup fails",
			setup: func(f *paddleErrFixture) {
				f.mockDB.EXPECT().BillingGetByIdempotencyKey(f.ctx, f.idempotencyKey).Return(nil, f.errDB)
			},
		},
		{
			name: "account lookup fails",
			setup: func(f *paddleErrFixture) {
				f.notProcessed()
				f.mockDB.EXPECT().AccountGetByPaddleSubscriptionID(f.ctx, "sub_err_002").Return(nil, f.errDB)
			},
		},
		{
			name: "plan type update fails",
			setup: func(f *paddleErrFixture) {
				f.notProcessed()
				f.mockDB.EXPECT().AccountGetByPaddleSubscriptionID(f.ctx, "sub_err_002").Return(f.acc, nil)
				f.mockDB.EXPECT().AccountUpdate(f.ctx, f.acc.ID, map[account.Field]any{account.FieldPlanType: plan}).Return(f.errDB)
			},
		},
		{
			name: "plan status reset fails",
			setup: func(f *paddleErrFixture) {
				f.notProcessed()
				f.mockDB.EXPECT().AccountGetByPaddleSubscriptionID(f.ctx, "sub_err_002").Return(f.acc, nil)
				f.planUpdated(plan)
				f.statusReset(f.errDB)
			},
		},
		{
			name: "token top up fails",
			setup: func(f *paddleErrFixture) {
				f.notProcessed()
				f.mockDB.EXPECT().AccountGetByPaddleSubscriptionID(f.ctx, "sub_err_002").Return(f.acc, nil)
				f.planUpdated(plan)
				f.statusReset(nil)
				f.mockDB.EXPECT().AccountPaddleTopUpTokens(f.ctx, f.acc.ID, f.acc.CustomerID, account.PlanTokenMap[plan], string(plan), billing.TransactionTypeAdjustment, f.idempotencyKey).Return(f.errDB)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, f := newPaddleErrFixture(t)
			tt.setup(f)

			err := h.PaddleSubscriptionUpdate(f.ctx, "sub_err_002", plan, f.eventID)
			if !errors.Is(err, f.errDB) {
				t.Errorf("Wrong match. expect: wrapped db error, got: %v", err)
			}
		})
	}
}

func Test_PaddleSubscriptionCancel_errorPaths(t *testing.T) {
	free := account.PlanTypeFree

	tests := []struct {
		name  string
		setup func(f *paddleErrFixture)
	}{
		{
			name: "idempotency lookup fails",
			setup: func(f *paddleErrFixture) {
				f.mockDB.EXPECT().BillingGetByIdempotencyKey(f.ctx, f.idempotencyKey).Return(nil, f.errDB)
			},
		},
		{
			name: "account lookup fails",
			setup: func(f *paddleErrFixture) {
				f.notProcessed()
				f.mockDB.EXPECT().AccountGetByPaddleSubscriptionID(f.ctx, "sub_err_003").Return(nil, f.errDB)
			},
		},
		{
			name: "downgrade fails",
			setup: func(f *paddleErrFixture) {
				f.notProcessed()
				f.mockDB.EXPECT().AccountGetByPaddleSubscriptionID(f.ctx, "sub_err_003").Return(f.acc, nil)
				f.mockDB.EXPECT().AccountUpdate(f.ctx, f.acc.ID, map[account.Field]any{account.FieldPlanType: free}).Return(f.errDB)
			},
		},
		{
			name: "plan status reset fails",
			setup: func(f *paddleErrFixture) {
				f.notProcessed()
				f.mockDB.EXPECT().AccountGetByPaddleSubscriptionID(f.ctx, "sub_err_003").Return(f.acc, nil)
				f.planUpdated(free)
				f.statusReset(f.errDB)
			},
		},
		{
			name: "token top up fails",
			setup: func(f *paddleErrFixture) {
				f.notProcessed()
				f.mockDB.EXPECT().AccountGetByPaddleSubscriptionID(f.ctx, "sub_err_003").Return(f.acc, nil)
				f.planUpdated(free)
				f.statusReset(nil)
				f.mockDB.EXPECT().AccountPaddleTopUpTokens(f.ctx, f.acc.ID, f.acc.CustomerID, account.PlanTokenMap[free], string(free), billing.TransactionTypeAdjustment, f.idempotencyKey).Return(f.errDB)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, f := newPaddleErrFixture(t)
			tt.setup(f)

			err := h.PaddleSubscriptionCancel(f.ctx, "sub_err_003", f.eventID)
			if !errors.Is(err, f.errDB) {
				t.Errorf("Wrong match. expect: wrapped db error, got: %v", err)
			}
		})
	}
}

func Test_PaddleSubscriptionRenew_errorPaths(t *testing.T) {
	tests := []struct {
		name  string
		setup func(f *paddleErrFixture)

		expectWrapped bool
	}{
		{
			name: "idempotency lookup fails",
			setup: func(f *paddleErrFixture) {
				f.mockDB.EXPECT().BillingGetByIdempotencyKey(f.ctx, f.idempotencyKey).Return(nil, f.errDB)
			},
			expectWrapped: true,
		},
		{
			name: "account lookup fails",
			setup: func(f *paddleErrFixture) {
				f.notProcessed()
				f.mockDB.EXPECT().AccountGetByPaddleSubscriptionID(f.ctx, "sub_err_004").Return(nil, f.errDB)
			},
			expectWrapped: true,
		},
		{
			name: "plan type has no token allowance",
			setup: func(f *paddleErrFixture) {
				f.notProcessed()
				f.acc.PlanType = account.PlanType("retired_plan")
				f.mockDB.EXPECT().AccountGetByPaddleSubscriptionID(f.ctx, "sub_err_004").Return(f.acc, nil)
			},
		},
		{
			name: "token top up fails",
			setup: func(f *paddleErrFixture) {
				f.notProcessed()
				f.mockDB.EXPECT().AccountGetByPaddleSubscriptionID(f.ctx, "sub_err_004").Return(f.acc, nil)
				f.mockDB.EXPECT().AccountPaddleTopUpTokens(f.ctx, f.acc.ID, f.acc.CustomerID, account.PlanTokenMap[account.PlanTypeBasic], string(account.PlanTypeBasic), billing.TransactionTypeTopUp, f.idempotencyKey).Return(f.errDB)
			},
			expectWrapped: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, f := newPaddleErrFixture(t)
			tt.setup(f)

			err := h.PaddleSubscriptionRenew(f.ctx, "sub_err_004", f.eventID)
			if err == nil {
				t.Fatalf("Wrong match. expect: error, got: nil")
			}
			if tt.expectWrapped && !errors.Is(err, f.errDB) {
				t.Errorf("Wrong match. expect: wrapped db error, got: %v", err)
			}
		})
	}
}

func Test_PaddleCreditTopUp_errorPaths(t *testing.T) {
	tests := []struct {
		name  string
		setup func(f *paddleErrFixture)
	}{
		{
			name: "idempotency lookup fails",
			setup: func(f *paddleErrFixture) {
				f.mockDB.EXPECT().BillingGetByIdempotencyKey(f.ctx, f.idempotencyKey).Return(nil, f.errDB)
			},
		},
		{
			name: "account credit update fails",
			setup: func(f *paddleErrFixture) {
				f.notProcessed()
				f.customerLookup()
				f.mockDB.EXPECT().AccountPaddleAddCredit(f.ctx, f.acc.ID, int64(4000000), f.acc.CustomerID, f.idempotencyKey).Return(f.errDB)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, f := newPaddleErrFixture(t)
			tt.setup(f)

			err := h.PaddleCreditTopUp(f.ctx, f.acc.CustomerID, 4000000, f.eventID)
			if !errors.Is(err, f.errDB) {
				t.Errorf("Wrong match. expect: wrapped db error, got: %v", err)
			}
		})
	}
}

func Test_PaddleRefund_errorPaths(t *testing.T) {
	amount := int64(2500000)

	tests := []struct {
		name  string
		setup func(f *paddleErrFixture)

		amount        int64
		expectErr     bool
		expectWrapped bool
	}{
		{
			name:      "amount is zero",
			amount:    0,
			setup:     func(f *paddleErrFixture) {},
			expectErr: true,
		},
		{
			name:   "idempotency lookup fails",
			amount: amount,
			setup: func(f *paddleErrFixture) {
				f.mockDB.EXPECT().BillingGetByIdempotencyKey(f.ctx, f.idempotencyKey).Return(nil, f.errDB)
			},
			expectErr:     true,
			expectWrapped: true,
		},
		{
			name:   "customer lookup fails",
			amount: amount,
			setup: func(f *paddleErrFixture) {
				f.notProcessed()
				f.mockReq.EXPECT().CustomerV1CustomerGet(f.ctx, f.acc.CustomerID).Return(nil, f.errDB)
			},
			expectErr:     true,
			expectWrapped: true,
		},
		{
			name:   "balance subtraction fails",
			amount: amount,
			setup: func(f *paddleErrFixture) {
				f.notProcessed()
				f.customerLookup()
				f.mockDB.EXPECT().AccountPaddleSubtractCredit(f.ctx, f.acc.ID, amount, f.acc.CustomerID, f.idempotencyKey).Return(f.errDB)
			},
			expectErr:     true,
			expectWrapped: true,
		},
		{
			name:   "reading the updated account fails",
			amount: amount,
			setup: func(f *paddleErrFixture) {
				f.notProcessed()
				f.customerLookup()
				f.mockDB.EXPECT().AccountPaddleSubtractCredit(f.ctx, f.acc.ID, amount, f.acc.CustomerID, f.idempotencyKey).Return(nil)
				f.mockDB.EXPECT().AccountGet(f.ctx, f.acc.ID).Return(nil, f.errDB)
			},
			expectErr:     true,
			expectWrapped: true,
		},
		{
			name:   "non-negative balance does not freeze the account",
			amount: amount,
			setup: func(f *paddleErrFixture) {
				f.notProcessed()
				f.customerLookup()
				f.mockDB.EXPECT().AccountPaddleSubtractCredit(f.ctx, f.acc.ID, amount, f.acc.CustomerID, f.idempotencyKey).Return(nil)
				f.mockDB.EXPECT().AccountGet(f.ctx, f.acc.ID).Return(&account.Account{Identity: f.acc.Identity, BalanceCredit: 0}, nil)
			},
		},
		{
			name:   "negative balance freezes the account",
			amount: amount,
			setup: func(f *paddleErrFixture) {
				f.notProcessed()
				f.customerLookup()
				f.mockDB.EXPECT().AccountPaddleSubtractCredit(f.ctx, f.acc.ID, amount, f.acc.CustomerID, f.idempotencyKey).Return(nil)
				negative := &account.Account{Identity: f.acc.Identity, BalanceCredit: -1}
				frozen := &account.Account{Identity: f.acc.Identity, BalanceCredit: -1, Status: account.StatusFrozen}
				f.mockDB.EXPECT().AccountGet(f.ctx, f.acc.ID).Return(negative, nil)
				f.mockDB.EXPECT().AccountSetStatus(f.ctx, f.acc.ID, account.StatusFrozen).Return(nil)
				f.mockDB.EXPECT().AccountGet(f.ctx, f.acc.ID).Return(frozen, nil)
				f.mockNtfy.EXPECT().PublishEvent(f.ctx, account.EventTypeAccountUpdated, frozen)
			},
		},
		{
			name:   "negative balance and the status update fails",
			amount: amount,
			setup: func(f *paddleErrFixture) {
				f.notProcessed()
				f.customerLookup()
				f.mockDB.EXPECT().AccountPaddleSubtractCredit(f.ctx, f.acc.ID, amount, f.acc.CustomerID, f.idempotencyKey).Return(nil)
				f.mockDB.EXPECT().AccountGet(f.ctx, f.acc.ID).Return(&account.Account{Identity: f.acc.Identity, BalanceCredit: -1}, nil)
				f.mockDB.EXPECT().AccountSetStatus(f.ctx, f.acc.ID, account.StatusFrozen).Return(f.errDB)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, f := newPaddleErrFixture(t)
			tt.setup(f)

			err := h.PaddleRefund(f.ctx, f.acc.CustomerID, tt.amount, f.eventID)
			if (err != nil) != tt.expectErr {
				t.Fatalf("Wrong match. expect error: %v, got: %v", tt.expectErr, err)
			}
			if tt.expectWrapped && !errors.Is(err, f.errDB) {
				t.Errorf("Wrong match. expect: wrapped db error, got: %v", err)
			}
		})
	}
}

func Test_PaddleSubscriptionScheduleCancel_errorPaths(t *testing.T) {
	tests := []struct {
		name  string
		setup func(f *paddleErrFixture)
	}{
		{
			name: "account lookup fails",
			setup: func(f *paddleErrFixture) {
				f.mockDB.EXPECT().AccountGetByPaddleSubscriptionID(f.ctx, "sub_err_005").Return(nil, f.errDB)
			},
		},
		{
			name: "plan status update fails",
			setup: func(f *paddleErrFixture) {
				f.mockDB.EXPECT().AccountGetByPaddleSubscriptionID(f.ctx, "sub_err_005").Return(f.acc, nil)
				f.mockDB.EXPECT().AccountUpdate(f.ctx, f.acc.ID, map[account.Field]any{account.FieldPlanStatus: account.PlanStatusCanceling}).Return(f.errDB)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, f := newPaddleErrFixture(t)
			tt.setup(f)

			err := h.PaddleSubscriptionScheduleCancel(f.ctx, "sub_err_005", f.eventID)
			if !errors.Is(err, f.errDB) {
				t.Errorf("Wrong match. expect: wrapped db error, got: %v", err)
			}
		})
	}
}
