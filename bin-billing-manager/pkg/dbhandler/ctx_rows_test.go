package dbhandler

import (
	"context"
	"database/sql/driver"
	"errors"
	"strings"
	"testing"
	"time"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-billing-manager/models/account"
	"monorepo/bin-billing-manager/models/billing"
	"monorepo/bin-billing-manager/pkg/cachehandler"
)

var errRowsBroken = errors.New("simulated failure while reading rows")

// Test_RowsErr_NotMisreported verifies that a failure while iterating the rows
// is returned as an error: a single-row get must not report it as ErrNotFound
// and a list must not return a partial (here empty) result as a success.
func Test_RowsErr_NotMisreported(t *testing.T) {
	id := uuid.FromStringOrNil("9a0f3c4e-1b2d-11f1-8a5e-0b3f6c1d2e77")

	tests := []struct {
		name string
		fn   func(h *handler) error
	}{
		{"get account", func(h *handler) error { _, err := h.accountGetFromDB(context.Background(), id); return err }},
		{"get billing", func(h *handler) error { _, err := h.billingGetFromDB(context.Background(), id); return err }},
		{"get billing by reference id", func(h *handler) error {
			_, err := h.billingGetByReferenceIDFromDB(context.Background(), id)
			return err
		}},
		{"get billing by reference type and id", func(h *handler) error {
			_, err := h.billingGetByReferenceTypeAndIDFromDB(context.Background(), billing.ReferenceTypeCall, id)
			return err
		}},
		{"get billing by idempotency key", func(h *handler) error {
			_, err := h.BillingGetByIdempotencyKey(context.Background(), id)
			return err
		}},
		{"list account", func(h *handler) error { _, err := h.AccountList(context.Background(), 10, "", nil); return err }},
		{"list billing", func(h *handler) error { _, err := h.BillingList(context.Background(), 10, "", nil); return err }},
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
			mockUtil.EXPECT().TimeGetCurTime().Return("2026-01-01 00:00:00.000000").AnyTimes()

			h := &handler{db: db, cache: cachehandler.NewMockCacheHandler(mc), utilHandler: mockUtil}

			// the first row fails when it is read, so Next returns false and
			// the cause is only reported by rows.Err().
			rows := sqlmock.NewRows([]string{"id"}).AddRow(id.Bytes()).RowError(0, errRowsBroken)
			mock.ExpectQuery("SELECT").WillReturnRows(rows)

			err = tt.fn(h)
			if err == nil {
				t.Fatalf("expected an error, got nil")
			}
			if errors.Is(err, ErrNotFound) {
				t.Fatalf("an iteration failure must not be reported as ErrNotFound. err: %v", err)
			}
			if !errors.Is(err, errRowsBroken) {
				t.Errorf("expected the iteration failure in the error, got: %v", err)
			}
		})
	}
}

// Test_ContextCancel_QueryAborted verifies that the request context reaches the
// database driver: a cancelled context aborts the query (read, write and list
// paths) instead of the query running to completion.
func Test_ContextCancel_QueryAborted(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := handler{
		utilHandler: mockUtil,
		db:          dbTest,
		cache:       mockCache,
	}
	mockUtil.EXPECT().TimeNow().Return(nil).AnyTimes()
	mockUtil.EXPECT().TimeGetCurTime().Return("2026-01-01 00:00:00.000000").AnyTimes()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	id := uuid.FromStringOrNil("9a0f3c4e-1b2d-11f1-8a5e-0b3f6c1d2e77")

	tests := []struct {
		name string
		fn   func() error
	}{
		{"read via private get (account)", func() error { _, err := h.accountGetFromDB(ctx, id); return err }},
		{"read via private get (billing)", func() error { _, err := h.billingGetFromDB(ctx, id); return err }},
		{"write (failed event delete)", func() error { return h.FailedEventDelete(ctx, id) }},
		{"list (billing)", func() error { _, err := h.BillingList(ctx, 10, "", nil); return err }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if err == nil || !strings.Contains(err.Error(), context.Canceled.Error()) {
				t.Errorf("expected a context canceled error, got: %v", err)
			}
		})
	}
}

// Test_BillingConsumeAndRecord_cancelAfterCommit verifies that the reads after
// the commit still succeed when the request context is cancelled once the
// commit is done: the committed charge must not be reported to the caller as
// a failure.
func Test_BillingConsumeAndRecord_cancelAfterCommit(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("could not create sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	mc := gomock.NewController(t)
	defer mc.Finish()
	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := &handler{db: db, cache: mockCache, utilHandler: mockUtil}

	accountID := uuid.FromStringOrNil("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	bill := &billing.Billing{}
	bill.ID = uuid.FromStringOrNil("bbbbbbbb-bbbb-cccc-dddd-eeeeeeeeeeee")

	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mockUtil.EXPECT().TimeNow().Return(&now).AnyTimes()

	// a row of all NULL values has the same shape as a real row
	nullRow := func(cols []string) *sqlmock.Rows {
		vals := make([]driver.Value, len(cols))
		return sqlmock.NewRows(cols).AddRow(vals...)
	}
	accountCols := commondatabasehandler.GetDBFields(account.Account{})
	billingCols := commondatabasehandler.GetDBFields(billing.Billing{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT balance_token, balance_credit FROM billing_accounts").
		WithArgs(accountID.Bytes()).
		WillReturnRows(sqlmock.NewRows([]string{"balance_token", "balance_credit"}).AddRow(int64(0), int64(1000000)))
	mock.ExpectExec("UPDATE billing_accounts").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE billing_billings").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	// the three reads after the commit
	mock.ExpectQuery("SELECT .* FROM billing_accounts").WillReturnRows(nullRow(accountCols))
	mock.ExpectQuery("SELECT .* FROM billing_billings").WillReturnRows(nullRow(billingCols))
	mock.ExpectQuery("SELECT .* FROM billing_billings").WillReturnRows(nullRow(billingCols))

	// the context is cancelled right after the commit, while the first cache
	// refresh is running.
	mockCache.EXPECT().AccountSet(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, _ *account.Account) error {
		cancel()
		return nil
	})
	mockCache.EXPECT().BillingSet(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	mockCache.EXPECT().BillingGet(gomock.Any(), gomock.Any()).Return(nil, errors.New("cache miss"))

	costInfo := billing.CostInfo{Mode: billing.CostModeCreditOnly, CreditPerUnit: 10}
	res, err := h.BillingConsumeAndRecord(ctx, bill, accountID, 1, 60, costInfo, &now)
	if err != nil {
		t.Fatalf("a charge committed before the cancel must be reported as a success. err: %v", err)
	}
	if res == nil {
		t.Errorf("expected the updated billing, got nil")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}
