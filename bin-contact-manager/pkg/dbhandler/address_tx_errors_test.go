package dbhandler

// Error path and branch tests for the transaction level address writers
// (AddressUpdateTx, AddressDeleteTx, AddressClaimTx, AddressResetPrimaryTx,
// AddressDeleteCompensating). They use sqlmock only.

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	mysql_driver "github.com/go-sql-driver/mysql"
	"github.com/gofrs/uuid"

	commonaddress "monorepo/bin-common-handler/models/address"

	"monorepo/bin-contact-manager/models/contact"
)

var (
	txTestAddressID  = uuid.FromStringOrNil("db000000-0000-0000-0000-000000000001")
	txTestCustomerID = uuid.FromStringOrNil("db000000-0000-0000-0000-000000000002")
	txTestContactID  = uuid.FromStringOrNil("db000000-0000-0000-0000-000000000003")
	txTestOtherID    = uuid.FromStringOrNil("db000000-0000-0000-0000-000000000004")
	txTestPeriodID   = uuid.FromStringOrNil("db000000-0000-0000-0000-000000000005")
)

const (
	txTestOldTarget = "+15550000100"
	txTestNewHigh   = "+15550000200" // sorts after the old target
	txTestNewLow    = "+15550000050" // sorts before the old target
)

var periodColumns = []string{"id", "contact_id", "valid_from", "valid_to"}

func emptyPeriods() *sqlmock.Rows { return sqlmock.NewRows(periodColumns) }

func duplicateTargetErr() error {
	return &mysql_driver.MySQLError{Number: 1062, Message: "Duplicate entry for key 'idx_contact_addresses_identifier'"}
}

func Test_AddressUpdateTx(t *testing.T) {
	ctx := context.Background()

	// expectLockEmpty queues the ownership lock query that finds no period.
	expectLockEmpty := func(mock sqlmock.Sqlmock) {
		mock.ExpectQuery("FROM contact_address_ownership_periods").WillReturnRows(emptyPeriods())
	}

	tests := []struct {
		name      string
		fields    map[string]any
		newTarget string // when set, fields["target"] is this value
		setup     func(mock sqlmock.Sqlmock)
		wantErr   error // compared with ==
		wantText  string
		wantIs    error
	}{
		{
			name:   "same target value is not a change, only the row is updated",
			fields: map[string]any{"target": txTestOldTarget},
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("UPDATE contact_addresses SET target").
					WithArgs(txTestOldTarget, sqlmock.AnyArg(), txTestAddressID.Bytes()).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
		},
		{
			name:   "detail is written",
			fields: map[string]any{"detail": "note"},
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("UPDATE contact_addresses SET detail").
					WithArgs("note", sqlmock.AnyArg(), txTestAddressID.Bytes()).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
		},
		{
			name:   "update deadlock is the bare sentinel",
			fields: map[string]any{"name": "n"},
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("UPDATE contact_addresses").WillReturnError(mysqlDeadlock())
			},
			wantErr: ErrDeadlock,
		},
		{
			name:   "update failure is returned wrapped",
			fields: map[string]any{"name": "n"},
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("UPDATE contact_addresses").WillReturnError(errCaseBoom)
			},
			wantText: "could not execute. AddressUpdateTx",
			wantIs:   errCaseBoom,
		},
		{
			name:   "a duplicate error without a target change is not a duplicate target",
			fields: map[string]any{"name": "n"},
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("UPDATE contact_addresses").WillReturnError(duplicateTargetErr())
			},
			wantText: "could not execute. AddressUpdateTx",
		},
		{
			name:   "rows affected failure",
			fields: map[string]any{"name": "n"},
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("UPDATE contact_addresses").WillReturnResult(sqlmock.NewErrorResult(errCaseBoom))
			},
			wantText: "could not read rows affected. AddressUpdateTx",
			wantIs:   errCaseBoom,
		},
		{
			name:   "no row matched means the target moved",
			fields: map[string]any{"name": "n"},
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("UPDATE contact_addresses").WillReturnResult(sqlmock.NewResult(0, 0))
			},
			wantErr: ErrStaleTarget,
		},
		{
			name:      "target change to a higher value closes the old period then opens the new one",
			newTarget: txTestNewHigh,
			setup: func(mock sqlmock.Sqlmock) {
				expectLockEmpty(mock) // old target: nothing to close
				expectLockEmpty(mock) // new target: first registration
				mock.ExpectExec("INSERT INTO contact_address_ownership_periods").WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectExec("UPDATE contact_addresses").WillReturnResult(sqlmock.NewResult(0, 1))
			},
		},
		{
			name:      "target change to a higher value stops when the old period lock fails",
			newTarget: txTestNewHigh,
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("FROM contact_address_ownership_periods").WillReturnError(errCaseBoom)
			},
			wantText: "could not query. OwnershipPeriodsLockAndResolveTx",
			wantIs:   errCaseBoom,
		},
		{
			name:      "target change to a higher value stops when the new period cannot be opened",
			newTarget: txTestNewHigh,
			setup: func(mock sqlmock.Sqlmock) {
				expectLockEmpty(mock)
				mock.ExpectQuery("FROM contact_address_ownership_periods").WillReturnError(errCaseBoom)
			},
			wantText: "could not query. OwnershipPeriodsLockAndResolveTx",
			wantIs:   errCaseBoom,
		},
		{
			name:      "target change to a lower value opens the new period first and stops on its failure",
			newTarget: txTestNewLow,
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("FROM contact_address_ownership_periods").WillReturnError(errCaseBoom)
			},
			wantText: "could not query. OwnershipPeriodsLockAndResolveTx",
			wantIs:   errCaseBoom,
		},
		{
			name:      "target change to a lower value stops when the old period cannot be closed",
			newTarget: txTestNewLow,
			setup: func(mock sqlmock.Sqlmock) {
				expectLockEmpty(mock)
				mock.ExpectExec("INSERT INTO contact_address_ownership_periods").WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectQuery("FROM contact_address_ownership_periods").WillReturnError(errCaseBoom)
			},
			wantText: "could not query. OwnershipPeriodsLockAndResolveTx",
			wantIs:   errCaseBoom,
		},
		{
			name:      "a duplicate target after a target change is reported as a duplicate",
			newTarget: txTestNewHigh,
			setup: func(mock sqlmock.Sqlmock) {
				expectLockEmpty(mock)
				expectLockEmpty(mock)
				mock.ExpectExec("INSERT INTO contact_address_ownership_periods").WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectExec("UPDATE contact_addresses").WillReturnError(duplicateTargetErr())
			},
			wantErr: ErrDuplicateTarget,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock, _ := newCtxRowsHandler(t)
			tx := openTestTx(t, h, mock)
			tt.setup(mock)

			fields := tt.fields
			if tt.newTarget != "" {
				fields = map[string]any{"target": tt.newTarget}
			}

			err := h.AddressUpdateTx(ctx, tx, txTestAddressID, txTestCustomerID, txTestContactID, commonaddress.TypeTel, txTestOldTarget, fields)
			switch {
			case tt.wantErr != nil:
				if err != tt.wantErr {
					t.Errorf("error = %v, want the bare %v", err, tt.wantErr)
				}
			case tt.wantText != "":
				assertWrappedErr(t, err, tt.wantText, tt.wantIs)
			default:
				if err != nil {
					t.Fatalf("error = %v", err)
				}
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet sqlmock expectations: %v", err)
			}
		})
	}
}

// The new target period of a moved address starts now when another contact
// already used the target, which is the reassignment case.
func Test_addressUpdateOpenNewTargetTx(t *testing.T) {
	ctx := context.Background()
	closedAt := time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC).Format(repairTimeLayout)

	t.Run("reassignment starts the period at the current time", func(t *testing.T) {
		h, mock, _ := newCtxRowsHandler(t)
		tx := openTestTx(t, h, mock)
		mock.ExpectQuery("FROM contact_address_ownership_periods").
			WillReturnRows(sqlmock.NewRows(periodColumns).AddRow(txTestPeriodID.Bytes(), txTestOtherID.Bytes(), nil, closedAt))
		now := time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC) // the time returned by newCtxRowsHandler
		// valid_from is the eighth value of the alphabetically ordered column list
		mock.ExpectExec("INSERT INTO contact_address_ownership_periods").
			WithArgs(
				txTestContactID.Bytes(), txTestCustomerID.Bytes(), sqlmock.AnyArg(), txTestNewHigh,
				sqlmock.AnyArg(), sqlmock.AnyArg(), string(commonaddress.TypeTel),
				timeArg{now}, nil,
			).
			WillReturnResult(sqlmock.NewResult(1, 1))

		if err := h.addressUpdateOpenNewTargetTx(ctx, tx, txTestCustomerID, txTestContactID, commonaddress.TypeTel, txTestNewHigh); err != nil {
			t.Fatalf("error = %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet sqlmock expectations: %v", err)
		}
	})

	t.Run("lock failure is returned", func(t *testing.T) {
		h, mock, _ := newCtxRowsHandler(t)
		tx := openTestTx(t, h, mock)
		mock.ExpectQuery("FROM contact_address_ownership_periods").WillReturnError(errCaseBoom)

		err := h.addressUpdateOpenNewTargetTx(ctx, tx, txTestCustomerID, txTestContactID, commonaddress.TypeTel, txTestNewHigh)
		assertWrappedErr(t, err, "could not query. OwnershipPeriodsLockAndResolveTx", errCaseBoom)
	})
}

func Test_AddressDeleteTx_Failures(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		setup    func(mock sqlmock.Sqlmock)
		wantErr  error // compared with ==
		wantText string
		wantIs   error
	}{
		{
			name: "lock failure stops before the delete",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("FROM contact_address_ownership_periods").WillReturnError(errCaseBoom)
			},
			wantText: "could not query. OwnershipPeriodsLockAndResolveTx",
			wantIs:   errCaseBoom,
		},
		{
			name: "delete deadlock is the bare sentinel",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("FROM contact_address_ownership_periods").WillReturnRows(emptyPeriods())
				mock.ExpectExec("DELETE FROM contact_addresses").WillReturnError(mysqlDeadlock())
			},
			wantErr: ErrDeadlock,
		},
		{
			name: "delete failure is returned wrapped",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("FROM contact_address_ownership_periods").WillReturnRows(emptyPeriods())
				mock.ExpectExec("DELETE FROM contact_addresses").WillReturnError(errCaseBoom)
			},
			wantText: "could not execute delete. AddressDeleteTx",
			wantIs:   errCaseBoom,
		},
		{
			name: "rows affected failure",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("FROM contact_address_ownership_periods").WillReturnRows(emptyPeriods())
				mock.ExpectExec("DELETE FROM contact_addresses").WillReturnResult(sqlmock.NewErrorResult(errCaseBoom))
			},
			wantText: "could not read rows affected. AddressDeleteTx",
			wantIs:   errCaseBoom,
		},
		{
			name: "no row matched means the address is already gone",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("FROM contact_address_ownership_periods").WillReturnRows(emptyPeriods())
				mock.ExpectExec("DELETE FROM contact_addresses").WillReturnResult(sqlmock.NewResult(0, 0))
			},
			wantErr: ErrStaleTarget,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock, _ := newCtxRowsHandler(t)
			tx := openTestTx(t, h, mock)
			tt.setup(mock)

			err := h.AddressDeleteTx(ctx, tx, txTestAddressID, txTestCustomerID, txTestContactID, commonaddress.TypeTel, txTestOldTarget)
			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("error = %v, want the bare %v", err, tt.wantErr)
				}
			} else {
				assertWrappedErr(t, err, tt.wantText, tt.wantIs)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet sqlmock expectations: %v", err)
			}
		})
	}
}

func Test_AddressClaimTx_Failures(t *testing.T) {
	ctx := context.Background()
	closedAt := time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC).Format(repairTimeLayout)

	tests := []struct {
		name     string
		setup    func(mock sqlmock.Sqlmock)
		wantErr  error // compared with ==
		wantText string
		wantIs   error
	}{
		{
			name: "lock failure",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("FROM contact_address_ownership_periods").WillReturnError(errCaseBoom)
			},
			wantText: "could not query. OwnershipPeriodsLockAndResolveTx",
			wantIs:   errCaseBoom,
		},
		{
			name: "reopening the previous period fails",
			setup: func(mock sqlmock.Sqlmock) {
				// the contact owned the target before and nobody intervened
				mock.ExpectQuery("FROM contact_address_ownership_periods").
					WillReturnRows(sqlmock.NewRows(periodColumns).AddRow(txTestPeriodID.Bytes(), txTestContactID.Bytes(), nil, closedAt))
				mock.ExpectExec("UPDATE contact_address_ownership_periods").WillReturnError(errCaseBoom)
			},
			wantText: "could not execute. ownershipPeriodReopenByIDTx",
			wantIs:   errCaseBoom,
		},
		{
			name: "address update deadlock is the bare sentinel",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("FROM contact_address_ownership_periods").WillReturnRows(emptyPeriods())
				mock.ExpectExec("INSERT INTO contact_address_ownership_periods").WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectExec("UPDATE contact_addresses").WillReturnError(mysqlDeadlock())
			},
			wantErr: ErrDeadlock,
		},
		{
			name: "address update failure is returned wrapped",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("FROM contact_address_ownership_periods").WillReturnRows(emptyPeriods())
				mock.ExpectExec("INSERT INTO contact_address_ownership_periods").WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectExec("UPDATE contact_addresses").WillReturnError(errCaseBoom)
			},
			wantText: "could not execute. AddressClaimTx",
			wantIs:   errCaseBoom,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock, _ := newCtxRowsHandler(t)
			tx := openTestTx(t, h, mock)
			tt.setup(mock)

			err := h.AddressClaimTx(ctx, tx, txTestCustomerID, txTestAddressID, txTestContactID, commonaddress.TypeTel, txTestOldTarget)
			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("error = %v, want the bare %v", err, tt.wantErr)
				}
			} else {
				assertWrappedErr(t, err, tt.wantText, tt.wantIs)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet sqlmock expectations: %v", err)
			}
		})
	}
}

func Test_AddressResetPrimaryTx_Failures(t *testing.T) {
	ctx := context.Background()

	t.Run("deadlock is the bare sentinel", func(t *testing.T) {
		h, mock, _ := newCtxRowsHandler(t)
		tx := openTestTx(t, h, mock)
		mock.ExpectExec("UPDATE contact_addresses SET is_primary").WithArgs(false, txTestContactID.Bytes()).WillReturnError(mysqlDeadlock())

		if err := h.AddressResetPrimaryTx(ctx, tx, txTestContactID); err != ErrDeadlock {
			t.Errorf("error = %v, want the bare ErrDeadlock", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet sqlmock expectations: %v", err)
		}
	})

	t.Run("failure is returned wrapped", func(t *testing.T) {
		h, mock, _ := newCtxRowsHandler(t)
		tx := openTestTx(t, h, mock)
		mock.ExpectExec("UPDATE contact_addresses SET is_primary").WillReturnError(errCaseBoom)

		assertWrappedErr(t, h.AddressResetPrimaryTx(ctx, tx, txTestContactID), "could not execute. AddressResetPrimaryTx", errCaseBoom)
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet sqlmock expectations: %v", err)
		}
	})
}

func Test_AddressDeleteCompensating_Failures(t *testing.T) {
	ctx := context.Background()
	openPeriod := func() *sqlmock.Rows {
		return sqlmock.NewRows(periodColumns).AddRow(txTestPeriodID.Bytes(), txTestContactID.Bytes(), nil, nil)
	}

	tests := []struct {
		name     string
		setup    func(mock sqlmock.Sqlmock)
		wantErr  error // compared with ==
		wantText string
		wantIs   error
	}{
		{
			name: "begin failure",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin().WillReturnError(errCaseBoom)
			},
			wantText: "could not begin transaction. AddressDeleteCompensating",
			wantIs:   errCaseBoom,
		},
		{
			name: "lock failure rolls back",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery("FROM contact_address_ownership_periods").WillReturnError(errCaseBoom)
				mock.ExpectRollback()
			},
			wantText: "could not query. OwnershipPeriodsLockAndResolveTx",
			wantIs:   errCaseBoom,
		},
		{
			name: "no open period of the contact is a conflict",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery("FROM contact_address_ownership_periods").WillReturnRows(emptyPeriods())
				mock.ExpectRollback()
			},
			wantErr: ErrConflict,
		},
		{
			name: "period delete failure rolls back",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery("FROM contact_address_ownership_periods").WillReturnRows(openPeriod())
				mock.ExpectExec("DELETE FROM contact_address_ownership_periods").WillReturnError(errCaseBoom)
				mock.ExpectRollback()
			},
			wantText: "could not delete period. AddressDeleteCompensating",
			wantIs:   errCaseBoom,
		},
		{
			name: "address delete failure rolls back",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery("FROM contact_address_ownership_periods").WillReturnRows(openPeriod())
				mock.ExpectExec("DELETE FROM contact_address_ownership_periods").WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectExec("DELETE FROM contact_addresses").WillReturnError(errCaseBoom)
				mock.ExpectRollback()
			},
			wantText: "could not delete address row. AddressDeleteCompensating",
			wantIs:   errCaseBoom,
		},
		{
			name: "commit failure",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery("FROM contact_address_ownership_periods").WillReturnRows(openPeriod())
				mock.ExpectExec("DELETE FROM contact_address_ownership_periods").WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectExec("DELETE FROM contact_addresses").WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit().WillReturnError(errCaseBoom)
			},
			wantText: "could not commit transaction. AddressDeleteCompensating",
			wantIs:   errCaseBoom,
		},
		{
			name: "sustained deadlock exhausts the retries",
			setup: func(mock sqlmock.Sqlmock) {
				for i := 0; i < addressMaxDeadlockRetries; i++ {
					mock.ExpectBegin()
					mock.ExpectQuery("FROM contact_address_ownership_periods").WillReturnError(mysqlDeadlock())
					mock.ExpectRollback()
				}
			},
			wantText: "exhausted retries under sustained deadlock",
			wantIs:   ErrDeadlock,
		},
		{
			name: "period delete deadlock is retried until the budget is spent",
			setup: func(mock sqlmock.Sqlmock) {
				for i := 0; i < addressMaxDeadlockRetries; i++ {
					mock.ExpectBegin()
					mock.ExpectQuery("FROM contact_address_ownership_periods").WillReturnRows(openPeriod())
					mock.ExpectExec("DELETE FROM contact_address_ownership_periods").WillReturnError(mysqlDeadlock())
					mock.ExpectRollback()
				}
			},
			wantText: "exhausted retries under sustained deadlock",
			wantIs:   ErrDeadlock,
		},
		{
			name: "address delete deadlock is retried until the budget is spent",
			setup: func(mock sqlmock.Sqlmock) {
				for i := 0; i < addressMaxDeadlockRetries; i++ {
					mock.ExpectBegin()
					mock.ExpectQuery("FROM contact_address_ownership_periods").WillReturnRows(openPeriod())
					mock.ExpectExec("DELETE FROM contact_address_ownership_periods").WillReturnResult(sqlmock.NewResult(0, 1))
					mock.ExpectExec("DELETE FROM contact_addresses").WillReturnError(mysqlDeadlock())
					mock.ExpectRollback()
				}
			},
			wantText: "exhausted retries under sustained deadlock",
			wantIs:   ErrDeadlock,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock, _ := newCtxRowsHandler(t)
			tt.setup(mock)

			err := h.AddressDeleteCompensating(ctx, txTestCustomerID, txTestContactID, commonaddress.TypeTel, txTestOldTarget)
			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("error = %v, want the bare %v", err, tt.wantErr)
				}
			} else {
				assertWrappedErr(t, err, tt.wantText, tt.wantIs)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet sqlmock expectations: %v", err)
			}
		})
	}
}

func Test_attemptStaleRowRepairNewTx(t *testing.T) {
	ctx := context.Background()
	staleCols := []string{"id", "contact_id", "tm_create"}

	tests := []struct {
		name      string
		setup     func(mock sqlmock.Sqlmock)
		wantRetry bool
		wantText  string
		wantIs    error
	}{
		{
			name:     "begin failure",
			setup:    func(mock sqlmock.Sqlmock) { mock.ExpectBegin().WillReturnError(errCaseBoom) },
			wantText: "could not begin transaction. attemptStaleRowRepairNewTx",
			wantIs:   errCaseBoom,
		},
		{
			name: "a live owner leaves the transaction to roll back and asks for no retry",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(staleRowSQL).WillReturnRows(sqlmock.NewRows(staleCols).
					AddRow(txTestAddressID.Bytes(), txTestOtherID.Bytes(), time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC).Format(repairTimeLayout)))
				mock.ExpectQuery(tombstoneSQL).WillReturnRows(sqlmock.NewRows([]string{"tm_delete"}).AddRow(nil))
				mock.ExpectRollback()
			},
			wantRetry: false,
		},
		{
			name: "commit failure",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(staleRowSQL).WillReturnRows(sqlmock.NewRows(staleCols))
				mock.ExpectCommit().WillReturnError(errCaseBoom)
			},
			wantText: "could not commit transaction. attemptStaleRowRepairNewTx",
			wantIs:   errCaseBoom,
		},
		{
			name: "an empty slot commits and asks for a retry",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(staleRowSQL).WillReturnRows(sqlmock.NewRows(staleCols))
				mock.ExpectCommit()
			},
			wantRetry: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock, _ := newCtxRowsHandler(t)
			tt.setup(mock)

			retry, err := h.attemptStaleRowRepairNewTx(ctx, txTestCustomerID, commonaddress.TypeTel, txTestOldTarget, false)
			if tt.wantText != "" {
				assertWrappedErr(t, err, tt.wantText, tt.wantIs)
			} else if err != nil {
				t.Fatalf("error = %v", err)
			}
			if retry != tt.wantRetry {
				t.Errorf("retry = %v, want %v", retry, tt.wantRetry)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet sqlmock expectations: %v", err)
			}
		})
	}
}

func Test_AddressCreateTx_LockFailure(t *testing.T) {
	ctx := context.Background()
	h, mock, _ := newCtxRowsHandler(t)
	tx := openTestTx(t, h, mock)
	mock.ExpectQuery(lockSQL).WillReturnError(errCaseBoom)

	a := &contact.Address{
		Address:    commonaddress.Address{Type: commonaddress.TypeTel, Target: txTestOldTarget},
		ID:         txTestAddressID,
		CustomerID: txTestCustomerID,
		ContactID:  txTestContactID,
	}
	err := h.AddressCreateTx(ctx, tx, a)
	assertWrappedErr(t, err, "could not query. OwnershipPeriodsLockAndResolveTx", errCaseBoom)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}
