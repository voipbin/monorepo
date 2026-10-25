package dbhandler

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-billing-manager/models/account"
	"monorepo/bin-billing-manager/models/billing"
	"monorepo/bin-billing-manager/pkg/cachehandler"
)

// Test_ledger_wrappers verifies that every public balance and token wrapper forwards the right
// signed amount and check flag to its ledger function, and that the credit and token variants
// touch their own column only.
func Test_ledger_wrappers(t *testing.T) {
	accountID := uuid.FromStringOrNil("a1000000-0000-0000-0000-000000000001")
	customerID := uuid.FromStringOrNil("c1000000-0000-0000-0000-000000000001")
	billingID := uuid.FromStringOrNil("b1000000-0000-0000-0000-000000000001")
	now := time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)

	var startToken int64 = 500
	var startCredit int64 = 1000000

	tests := []struct {
		name string
		call func(h *handler) error

		updateSQL       string
		expectNewValue  int64
		expectAmountCr  int64
		expectAmountTok int64
		expectSnapCr    int64
		expectSnapTok   int64
		expectRefType   billing.ReferenceType
		expectErr       error // when set, no update and no insert may happen
	}{
		{
			name:            "add balance",
			call:            func(h *handler) error { return h.AccountAddBalance(context.Background(), accountID, 300) },
			updateSQL:       "UPDATE billing_accounts SET balance_credit = ?",
			expectNewValue:  1000300,
			expectAmountCr:  300,
			expectSnapCr:    1000300,
			expectSnapTok:   startToken,
			expectRefType:   billing.ReferenceTypeCreditAdjustment,
			expectAmountTok: 0,
		},
		{
			name:            "subtract balance with check sufficient",
			call:            func(h *handler) error { return h.AccountSubtractBalanceWithCheck(context.Background(), accountID, 300) },
			updateSQL:       "UPDATE billing_accounts SET balance_credit = ?",
			expectNewValue:  999700,
			expectAmountCr:  -300,
			expectSnapCr:    999700,
			expectSnapTok:   startToken,
			expectRefType:   billing.ReferenceTypeCreditAdjustment,
			expectAmountTok: 0,
		},
		{
			name: "subtract balance with check insufficient",
			call: func(h *handler) error {
				return h.AccountSubtractBalanceWithCheck(context.Background(), accountID, startCredit+1)
			},
			expectErr: ErrInsufficientBalance,
		},
		{
			name: "subtract balance without check goes negative",
			call: func(h *handler) error {
				return h.AccountSubtractBalance(context.Background(), accountID, startCredit+1)
			},
			updateSQL:       "UPDATE billing_accounts SET balance_credit = ?",
			expectNewValue:  -1,
			expectAmountCr:  -(startCredit + 1),
			expectSnapCr:    -1,
			expectSnapTok:   startToken,
			expectRefType:   billing.ReferenceTypeCreditAdjustment,
			expectAmountTok: 0,
		},
		{
			name:            "add tokens",
			call:            func(h *handler) error { return h.AccountAddTokens(context.Background(), accountID, 70) },
			updateSQL:       "UPDATE billing_accounts SET balance_token = ?",
			expectNewValue:  570,
			expectAmountTok: 70,
			expectSnapCr:    startCredit,
			expectSnapTok:   570,
			expectRefType:   billing.ReferenceTypeTokenAdjustment,
		},
		{
			name:            "subtract tokens with check sufficient",
			call:            func(h *handler) error { return h.AccountSubtractTokensWithCheck(context.Background(), accountID, 70) },
			updateSQL:       "UPDATE billing_accounts SET balance_token = ?",
			expectNewValue:  430,
			expectAmountTok: -70,
			expectSnapCr:    startCredit,
			expectSnapTok:   430,
			expectRefType:   billing.ReferenceTypeTokenAdjustment,
		},
		{
			name: "subtract tokens with check insufficient",
			call: func(h *handler) error {
				return h.AccountSubtractTokensWithCheck(context.Background(), accountID, startToken+1)
			},
			expectErr: ErrInsufficientBalance,
		},
		{
			name:            "subtract tokens without check goes negative",
			call:            func(h *handler) error { return h.AccountSubtractTokens(context.Background(), accountID, startToken+1) },
			updateSQL:       "UPDATE billing_accounts SET balance_token = ?",
			expectNewValue:  -1,
			expectAmountTok: -(startToken + 1),
			expectSnapCr:    startCredit,
			expectSnapTok:   -1,
			expectRefType:   billing.ReferenceTypeTokenAdjustment,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("could not create sqlmock: %v", err)
			}
			defer func() { _ = db.Close() }()

			mc := gomock.NewController(t)
			defer mc.Finish()
			mockUtil := utilhandler.NewMockUtilHandler(mc)

			h := &handler{db: db, cache: cachehandler.NewMockCacheHandler(mc), utilHandler: mockUtil}

			mock.ExpectBegin()
			mock.ExpectQuery(regexp.QuoteMeta("SELECT customer_id, balance_token, balance_credit FROM billing_accounts WHERE id = ? FOR UPDATE")).
				WithArgs(accountID.Bytes()).
				WillReturnRows(sqlmock.NewRows([]string{"customer_id", "balance_token", "balance_credit"}).
					AddRow(customerID.Bytes(), startToken, startCredit))

			if tt.expectErr != nil {
				mock.ExpectRollback()
			} else {
				mockUtil.EXPECT().TimeNow().Return(&now)
				mockUtil.EXPECT().UUIDCreate().Return(billingID)

				mock.ExpectExec(regexp.QuoteMeta(tt.updateSQL)).
					WithArgs(tt.expectNewValue, now, accountID.Bytes()).
					WillReturnResult(sqlmock.NewResult(0, 1))

				// columns of the insert are in alphabetical order: account_id, amount_credit, amount_token,
				// balance_credit_snapshot, balance_token_snapshot, billable_units, cost_type, customer_id, id,
				// idempotency_key, rate_credit_per_unit, rate_token_per_unit, reference_id, reference_type,
				// status, tm_billing_end, tm_billing_start, tm_create, tm_delete, tm_update, transaction_type,
				// usage_duration
				anyArg := sqlmock.AnyArg()
				mock.ExpectExec(regexp.QuoteMeta("INSERT INTO billing_billings (account_id,amount_credit,amount_token,balance_credit_snapshot,balance_token_snapshot,")).
					WithArgs(
						accountID.Bytes(), tt.expectAmountCr, tt.expectAmountTok, tt.expectSnapCr, tt.expectSnapTok,
						anyArg, anyArg, customerID.Bytes(), billingID.Bytes(),
						anyArg, anyArg, anyArg, billingID.Bytes(), string(tt.expectRefType),
						anyArg, anyArg, anyArg, anyArg, anyArg, anyArg, string(billing.TransactionTypeAdjustment), anyArg,
					).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			}

			err = tt.call(h)
			if tt.expectErr != nil {
				if !errors.Is(err, tt.expectErr) {
					t.Errorf("expected %v, got: %v", tt.expectErr, err)
				}
			} else if err != nil {
				t.Errorf("expected no error, got: %v", err)
			}

			if errMet := mock.ExpectationsWereMet(); errMet != nil {
				t.Errorf("unfulfilled expectations: %v", errMet)
			}
		})
	}
}

// Test_AccountSetStatus_exec_error verifies that a failing update is returned with its cause and
// that the cache is not refreshed.
func Test_AccountSetStatus_exec_error(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("could not create sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	mc := gomock.NewController(t)
	defer mc.Finish()
	mockUtil := utilhandler.NewMockUtilHandler(mc)
	h := &handler{db: db, cache: cachehandler.NewMockCacheHandler(mc), utilHandler: mockUtil}

	id := uuid.FromStringOrNil("a2000000-0000-0000-0000-000000000001")
	now := time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)
	cause := errors.New("exec failed")

	mockUtil.EXPECT().TimeNow().Return(&now)
	mock.ExpectExec("UPDATE billing_accounts SET").
		WithArgs(string(account.StatusFrozen), now, id.Bytes()).
		WillReturnError(cause)

	err = h.AccountSetStatus(context.Background(), id, account.StatusFrozen)
	if !errors.Is(err, cause) {
		t.Errorf("expected the exec cause, got: %v", err)
	}
	if errMet := mock.ExpectationsWereMet(); errMet != nil {
		t.Errorf("unfulfilled expectations: %v", errMet)
	}
}
