package dbhandler

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-billing-manager/models/account"
	"monorepo/bin-billing-manager/models/billing"
	"monorepo/bin-billing-manager/models/failedevent"
	"monorepo/bin-billing-manager/pkg/cachehandler"
)

var errCause = errors.New("injected database failure")

// newFailureHandler returns a handler on a fresh sqlmock. Any cache call fails the test, because
// every case below must stop before the cache is refreshed.
func newFailureHandler(t *testing.T) (*handler, sqlmock.Sqlmock, *utilhandler.MockUtilHandler, *cachehandler.MockCacheHandler) {
	t.Helper()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("could not create sqlmock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	mc := gomock.NewController(t)
	t.Cleanup(mc.Finish)

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)

	return &handler{db: db, cache: mockCache, utilHandler: mockUtil}, mock, mockUtil, mockCache
}

// Test_exec_failures verifies that a failing write is returned with its cause, names the failing
// operation, and does not refresh the cache.
func Test_exec_failures(t *testing.T) {
	id := uuid.FromStringOrNil("e1000000-0000-0000-0000-000000000001")
	now := time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)

	tests := []struct {
		name         string
		statement    string
		expectFrag   string
		call         func(h *handler) error
		callsTimeNow bool
	}{
		{
			name:         "account create",
			statement:    "INSERT INTO billing_accounts",
			expectFrag:   "AccountCreate",
			call:         func(h *handler) error { return h.AccountCreate(context.Background(), &account.Account{}) },
			callsTimeNow: true,
		},
		{
			name:       "account update",
			statement:  "UPDATE billing_accounts",
			expectFrag: "AccountUpdate",
			call: func(h *handler) error {
				return h.AccountUpdate(context.Background(), id, map[account.Field]any{account.FieldName: "n"})
			},
			callsTimeNow: true,
		},
		{
			name:         "account delete",
			statement:    "UPDATE billing_accounts",
			expectFrag:   "AccountDelete",
			call:         func(h *handler) error { return h.AccountDelete(context.Background(), id) },
			callsTimeNow: true,
		},
		{
			name:         "billing create",
			statement:    "INSERT INTO billing_billings",
			expectFrag:   "BillingCreate",
			call:         func(h *handler) error { return h.BillingCreate(context.Background(), &billing.Billing{}) },
			callsTimeNow: true,
		},
		{
			name:       "billing update",
			statement:  "UPDATE billing_billings",
			expectFrag: "BillingUpdate",
			call: func(h *handler) error {
				return h.BillingUpdate(context.Background(), id, map[billing.Field]any{billing.FieldStatus: billing.StatusEnd})
			},
			callsTimeNow: true,
		},
		{
			name:         "billing set status",
			statement:    "UPDATE billing_billings",
			expectFrag:   "BillingUpdate",
			call:         func(h *handler) error { return h.BillingSetStatus(context.Background(), id, billing.StatusEnd) },
			callsTimeNow: true,
		},
		{
			name:       "billing set status end",
			statement:  "UPDATE billing_billings",
			expectFrag: "BillingSetStatusEnd",
			call: func(h *handler) error {
				return h.BillingSetStatusEnd(context.Background(), id, 1, 2, 3, 4, 5, 6, &now)
			},
			callsTimeNow: true,
		},
		{
			name:         "billing delete",
			statement:    "UPDATE billing_billings",
			expectFrag:   "BillingDelete",
			call:         func(h *handler) error { return h.BillingDelete(context.Background(), id) },
			callsTimeNow: true,
		},
		{
			name:         "failed event create",
			statement:    "INSERT INTO billing_failed_events",
			expectFrag:   "FailedEventCreate",
			call:         func(h *handler) error { return h.FailedEventCreate(context.Background(), &failedevent.FailedEvent{}) },
			callsTimeNow: true,
		},
		{
			name:       "failed event update",
			statement:  "UPDATE billing_failed_events",
			expectFrag: "FailedEventUpdate",
			call: func(h *handler) error {
				return h.FailedEventUpdate(context.Background(), id, map[failedevent.Field]any{failedevent.FieldStatus: failedevent.StatusExhausted})
			},
			callsTimeNow: true,
		},
		{
			name:       "failed event delete",
			statement:  "DELETE FROM billing_failed_events",
			expectFrag: "FailedEventDelete",
			call:       func(h *handler) error { return h.FailedEventDelete(context.Background(), id) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock, mockUtil, _ := newFailureHandler(t)

			if tt.callsTimeNow {
				mockUtil.EXPECT().TimeNow().Return(&now).AnyTimes()
			}
			mock.ExpectExec(tt.statement).WillReturnError(errCause)

			err := tt.call(h)
			if !errors.Is(err, errCause) {
				t.Fatalf("expected the injected cause, got: %v", err)
			}
			if !strings.Contains(err.Error(), tt.expectFrag) {
				t.Errorf("expected the error to name %q, got: %v", tt.expectFrag, err)
			}
			if errMet := mock.ExpectationsWereMet(); errMet != nil {
				t.Errorf("unfulfilled expectations: %v", errMet)
			}
		})
	}
}

// Test_ledger_customer_id_parse_failure covers the failures of the two ledger transactions that depend on the
// stored customer id.
func Test_ledger_customer_id_parse_failure(t *testing.T) {
	accountID := uuid.FromStringOrNil("a3000000-0000-0000-0000-000000000001")
	now := time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)

	tests := []struct {
		name       string
		call       func(h *handler) error
		expectFrag string
	}{
		{"credit", func(h *handler) error { return h.AccountAddBalance(context.Background(), accountID, 1) }, "accountAdjustCreditWithLedger: could not parse customer_id"},
		{"token", func(h *handler) error { return h.AccountAddTokens(context.Background(), accountID, 1) }, "accountAdjustTokenWithLedger: could not parse customer_id"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock, mockUtil, _ := newFailureHandler(t)
			mockUtil.EXPECT().TimeNow().Return(&now)

			mock.ExpectBegin()
			mock.ExpectQuery("SELECT customer_id, balance_token, balance_credit FROM billing_accounts").
				WillReturnRows(sqlmock.NewRows([]string{"customer_id", "balance_token", "balance_credit"}).
					AddRow([]byte{0x01, 0x02}, int64(1), int64(1)))
			mock.ExpectExec("UPDATE billing_accounts SET").WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectRollback()

			err := tt.call(h)
			if err == nil || !strings.Contains(err.Error(), tt.expectFrag) {
				t.Errorf("expected %q, got: %v", tt.expectFrag, err)
			}
			if errMet := mock.ExpectationsWereMet(); errMet != nil {
				t.Errorf("unfulfilled expectations: %v", errMet)
			}
		})
	}
}

// Test_BillingConsumeAndRecord_failures walks the transaction and checks each failing step is
// reported with its cause and stops the later steps.
func Test_BillingConsumeAndRecord_failures(t *testing.T) {
	accountID := uuid.FromStringOrNil("a4000000-0000-0000-0000-000000000001")
	billID := uuid.FromStringOrNil("b4000000-0000-0000-0000-000000000001")
	now := time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)
	costInfo := billing.CostInfo{Mode: billing.CostModeCreditOnly, CreditPerUnit: 100}

	const selectSQL = "SELECT balance_token, balance_credit FROM billing_accounts"
	rows := func() *sqlmock.Rows {
		return sqlmock.NewRows([]string{"balance_token", "balance_credit"}).AddRow(int64(10), int64(1000))
	}

	tests := []struct {
		name       string
		setup      func(mock sqlmock.Sqlmock)
		expectFrag string
		expectIs   error
	}{
		{
			name:       "begin",
			setup:      func(mock sqlmock.Sqlmock) { mock.ExpectBegin().WillReturnError(errCause) },
			expectFrag: "could not begin transaction",
			expectIs:   errCause,
		},
		{
			name: "account not found",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(selectSQL).WillReturnError(sql.ErrNoRows)
				mock.ExpectRollback()
			},
			expectIs: ErrNotFound,
		},
		{
			name: "account read",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(selectSQL).WillReturnError(errCause)
				mock.ExpectRollback()
			},
			expectFrag: "could not read account",
			expectIs:   errCause,
		},
		{
			name: "account update",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(selectSQL).WillReturnRows(rows())
				mock.ExpectExec("UPDATE billing_accounts SET").WillReturnError(errCause)
				mock.ExpectRollback()
			},
			expectFrag: "could not update account balances",
			expectIs:   errCause,
		},
		{
			name: "billing update",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(selectSQL).WillReturnRows(rows())
				mock.ExpectExec("UPDATE billing_accounts SET").WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectExec("UPDATE billing_billings SET").WillReturnError(errCause)
				mock.ExpectRollback()
			},
			expectFrag: "could not update billing",
			expectIs:   errCause,
		},
		{
			name: "commit",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(selectSQL).WillReturnRows(rows())
				mock.ExpectExec("UPDATE billing_accounts SET").WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectExec("UPDATE billing_billings SET").WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit().WillReturnError(errCause)
			},
			expectFrag: "could not commit",
			expectIs:   errCause,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock, mockUtil, _ := newFailureHandler(t)
			mockUtil.EXPECT().TimeNow().Return(&now).AnyTimes()
			tt.setup(mock)

			res, err := h.BillingConsumeAndRecord(context.Background(), &billing.Billing{}, accountID, 3, 30, costInfo, &now)
			if res != nil {
				t.Errorf("expected no billing, got: %v", res)
			}
			if !errors.Is(err, tt.expectIs) {
				t.Fatalf("expected %v, got: %v", tt.expectIs, err)
			}
			if tt.expectFrag != "" && !strings.Contains(err.Error(), tt.expectFrag) {
				t.Errorf("expected %q in %v", tt.expectFrag, err)
			}
			if errMet := mock.ExpectationsWereMet(); errMet != nil {
				t.Errorf("unfulfilled expectations: %v", errMet)
			}
		})
	}

	// the charge is committed but the final read of the billing fails: the failure is reported
	// and names the re-read.
	t.Run("re-read after commit", func(t *testing.T) {
		h, mock, mockUtil, mockCache := newFailureHandler(t)
		mockUtil.EXPECT().TimeNow().Return(&now).AnyTimes()

		mock.ExpectBegin()
		mock.ExpectQuery(selectSQL).WillReturnRows(rows())
		mock.ExpectExec("UPDATE billing_accounts SET").WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec("UPDATE billing_billings SET").WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()
		// account refresh, billing refresh and the final read all query the database and fail.
		mock.ExpectQuery("SELECT").WillReturnError(errCause)
		mock.ExpectQuery("SELECT").WillReturnError(errCause)
		mock.ExpectQuery("SELECT").WillReturnError(errCause)
		mockCache.EXPECT().BillingGet(gomock.Any(), billID).Return(nil, errCause)

		res, err := h.BillingConsumeAndRecord(context.Background(), &billing.Billing{ID: billID}, accountID, 3, 30, costInfo, &now)
		if res != nil {
			t.Errorf("expected no billing, got: %v", res)
		}
		if !errors.Is(err, errCause) || !strings.Contains(err.Error(), "could not re-read billing") {
			t.Errorf("expected the re-read failure, got: %v", err)
		}
		if errMet := mock.ExpectationsWereMet(); errMet != nil {
			t.Errorf("unfulfilled expectations: %v", errMet)
		}
	})
}

// Test_AccountTopUpTokens_failures walks the top-up transaction failures.
func Test_AccountTopUpTokens_failures(t *testing.T) {
	accountID := uuid.FromStringOrNil("a5000000-0000-0000-0000-000000000001")
	customerID := uuid.FromStringOrNil("c5000000-0000-0000-0000-000000000001")
	billingID := uuid.FromStringOrNil("b5000000-0000-0000-0000-000000000001")
	now := time.Date(2026, 3, 17, 10, 30, 45, 0, time.UTC)

	const selectSQL = "SELECT balance_token, balance_credit FROM billing_accounts"
	rows := func() *sqlmock.Rows {
		return sqlmock.NewRows([]string{"balance_token", "balance_credit"}).AddRow(int64(5), int64(900))
	}

	tests := []struct {
		name       string
		setup      func(mock sqlmock.Sqlmock)
		expectFrag string
		expectIs   error
	}{
		{
			name:       "begin",
			setup:      func(mock sqlmock.Sqlmock) { mock.ExpectBegin().WillReturnError(errCause) },
			expectFrag: "could not begin transaction",
			expectIs:   errCause,
		},
		{
			name: "account not found",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(selectSQL).WillReturnError(sql.ErrNoRows)
				mock.ExpectRollback()
			},
			expectIs: ErrNotFound,
		},
		{
			name: "account read",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(selectSQL).WillReturnError(errCause)
				mock.ExpectRollback()
			},
			expectFrag: "could not read account",
			expectIs:   errCause,
		},
		{
			name: "account update",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(selectSQL).WillReturnRows(rows())
				mock.ExpectExec("UPDATE billing_accounts SET").WillReturnError(errCause)
				mock.ExpectRollback()
			},
			expectFrag: "could not update account",
			expectIs:   errCause,
		},
		{
			name: "rows affected",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(selectSQL).WillReturnRows(rows())
				mock.ExpectExec("UPDATE billing_accounts SET").WillReturnResult(sqlmock.NewErrorResult(errCause))
				mock.ExpectRollback()
			},
			expectFrag: "could not read rows affected",
			expectIs:   errCause,
		},
		{
			name: "ledger insert",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(selectSQL).WillReturnRows(rows())
				mock.ExpectExec("UPDATE billing_accounts SET").WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectExec("INSERT INTO billing_billings").WillReturnError(errCause)
				mock.ExpectRollback()
			},
			expectFrag: "could not insert billing record",
			expectIs:   errCause,
		},
		{
			name: "commit",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(selectSQL).WillReturnRows(rows())
				mock.ExpectExec("UPDATE billing_accounts SET").WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectExec("INSERT INTO billing_billings").WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit().WillReturnError(errCause)
			},
			expectFrag: "could not commit",
			expectIs:   errCause,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock, mockUtil, _ := newFailureHandler(t)
			mockUtil.EXPECT().TimeNow().Return(&now).AnyTimes()
			mockUtil.EXPECT().UUIDCreate().Return(billingID).AnyTimes()
			tt.setup(mock)

			applied, err := h.AccountTopUpTokens(context.Background(), accountID, customerID, 100, "basic")
			if applied {
				t.Errorf("a failed top-up must not report applied")
			}
			if !errors.Is(err, tt.expectIs) {
				t.Fatalf("expected %v, got: %v", tt.expectIs, err)
			}
			if tt.expectFrag != "" && !strings.Contains(err.Error(), tt.expectFrag) {
				t.Errorf("expected %q in %v", tt.expectFrag, err)
			}
			if errMet := mock.ExpectationsWereMet(); errMet != nil {
				t.Errorf("unfulfilled expectations: %v", errMet)
			}
		})
	}
}
