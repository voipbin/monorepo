package dbhandler

// Error path and branch tests for the stale row repair that the address
// create, update and claim flows run. They drive staleRowRepairTx directly
// inside a sqlmock transaction, so no shared database is touched.

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofrs/uuid"

	commonaddress "monorepo/bin-common-handler/models/address"
)

const repairTimeLayout = "2006-01-02 15:04:05.000000"

// timeArg matches a time.Time argument by instant.
type timeArg struct{ want time.Time }

func (m timeArg) Match(v driver.Value) bool {
	got, ok := v.(time.Time)
	return ok && got.Equal(m.want)
}

func openTestTx(t *testing.T, h *handler, mock sqlmock.Sqlmock) *sql.Tx {
	t.Helper()

	mock.ExpectBegin()
	tx, err := h.db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatalf("BeginTx() error = %v", err)
	}
	t.Cleanup(func() { _ = tx.Rollback() })

	return tx
}

func Test_staleRowRepairTx(t *testing.T) {
	resetTestDB(t)

	ctx := context.Background()
	customerID := uuid.FromStringOrNil("d9000000-0000-0000-0000-000000000001")
	rowID := uuid.FromStringOrNil("d9000000-0000-0000-0000-000000000002")
	ownerID := uuid.FromStringOrNil("d9000000-0000-0000-0000-000000000003")
	target := "+15550000099"

	created := time.Date(2026, 10, 20, 9, 0, 0, 0, time.UTC)
	deleted := created.Add(2 * time.Hour)

	staleRow := func(createdAt time.Time) *sqlmock.Rows {
		return sqlmock.NewRows([]string{"id", "contact_id", "tm_create"}).
			AddRow(rowID.Bytes(), ownerID.Bytes(), createdAt.Format(repairTimeLayout))
	}
	tombstone := func(at *time.Time) *sqlmock.Rows {
		if at == nil {
			return sqlmock.NewRows([]string{"tm_delete"}).AddRow(nil)
		}
		return sqlmock.NewRows([]string{"tm_delete"}).AddRow(at.Format(repairTimeLayout))
	}
	noPeriod := func() *sqlmock.Rows { return sqlmock.NewRows([]string{"valid_to"}) }

	tests := []struct {
		name        string
		resetToNull bool
		setup       func(mock sqlmock.Sqlmock)
		wantRetry   bool
		wantErr     error // compared with == (sentinels are returned bare)
		wantText    string
		wantIs      error
	}{
		{
			name: "slot lookup failure is returned wrapped",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("FROM contact_addresses").WillReturnError(errCaseBoom)
			},
			wantText: "could not query. staleRowByTargetTx",
			wantIs:   errCaseBoom,
		},
		{
			name: "no stale row means the caller can retry at once",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("FROM contact_addresses").WillReturnRows(sqlmock.NewRows([]string{"id", "contact_id", "tm_create"}))
			},
			wantRetry: true,
		},
		{
			name: "tombstone lookup failure is returned wrapped",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("FROM contact_addresses").WillReturnRows(staleRow(created))
				mock.ExpectQuery("FROM contact_contacts").WillReturnError(errCaseBoom)
			},
			wantText: "could not query. contactTombstoneTx",
			wantIs:   errCaseBoom,
		},
		{
			name: "live owner is a genuine conflict and nothing is written",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("FROM contact_addresses").WillReturnRows(staleRow(created))
				mock.ExpectQuery("FROM contact_contacts").WillReturnRows(tombstone(nil))
			},
			wantRetry: false,
		},
		{
			name: "latest period lookup failure is returned wrapped",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("FROM contact_addresses").WillReturnRows(staleRow(created))
				mock.ExpectQuery("FROM contact_contacts").WillReturnRows(tombstone(&deleted))
				mock.ExpectQuery("FROM contact_address_ownership_periods").WillReturnError(errCaseBoom)
			},
			wantText: "could not query. latestOwnershipPeriodValidToTx",
			wantIs:   errCaseBoom,
		},
		{
			name: "the period starts at the latest closed period when it is later than the row creation",
			setup: func(mock sqlmock.Sqlmock) {
				latest := created.Add(time.Hour)
				mock.ExpectQuery("FROM contact_addresses").WillReturnRows(staleRow(created))
				mock.ExpectQuery("FROM contact_contacts").WillReturnRows(tombstone(&deleted))
				mock.ExpectQuery("FROM contact_address_ownership_periods").
					WillReturnRows(sqlmock.NewRows([]string{"valid_to"}).AddRow(latest.Format(repairTimeLayout)))
				// squirrel orders the SetMap columns alphabetically: valid_from is the
				// eighth value and valid_to the ninth.
				mock.ExpectExec("INSERT INTO contact_address_ownership_periods").
					WithArgs(
						ownerID.Bytes(), customerID.Bytes(), sqlmock.AnyArg(), target,
						sqlmock.AnyArg(), sqlmock.AnyArg(), string(commonaddress.TypeTel),
						timeArg{latest}, timeArg{deleted},
					).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectExec("DELETE FROM contact_addresses").WithArgs(rowID.Bytes()).WillReturnResult(sqlmock.NewResult(0, 1))
			},
			wantRetry: true,
		},
		{
			name: "period insert deadlock is the bare sentinel",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("FROM contact_addresses").WillReturnRows(staleRow(created))
				mock.ExpectQuery("FROM contact_contacts").WillReturnRows(tombstone(&deleted))
				mock.ExpectQuery("FROM contact_address_ownership_periods").WillReturnRows(noPeriod())
				mock.ExpectExec("INSERT INTO contact_address_ownership_periods").WillReturnError(mysqlDeadlock())
			},
			wantErr: ErrDeadlock,
		},
		{
			name: "period insert failure is returned wrapped",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("FROM contact_addresses").WillReturnRows(staleRow(created))
				mock.ExpectQuery("FROM contact_contacts").WillReturnRows(tombstone(&deleted))
				mock.ExpectQuery("FROM contact_address_ownership_periods").WillReturnRows(noPeriod())
				mock.ExpectExec("INSERT INTO contact_address_ownership_periods").WillReturnError(errCaseBoom)
			},
			wantText: "could not execute. staleRowRepairTx",
			wantIs:   errCaseBoom,
		},
		{
			name:        "reset deadlock is the bare sentinel",
			resetToNull: true,
			setup: func(mock sqlmock.Sqlmock) {
				// the row was created after the contact was deleted, so no period is inserted
				mock.ExpectQuery("FROM contact_addresses").WillReturnRows(staleRow(deleted.Add(time.Hour)))
				mock.ExpectQuery("FROM contact_contacts").WillReturnRows(tombstone(&deleted))
				mock.ExpectQuery("FROM contact_address_ownership_periods").WillReturnRows(noPeriod())
				mock.ExpectExec("UPDATE contact_addresses").WillReturnError(mysqlDeadlock())
			},
			wantErr: ErrDeadlock,
		},
		{
			name:        "reset failure is returned wrapped",
			resetToNull: true,
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("FROM contact_addresses").WillReturnRows(staleRow(deleted.Add(time.Hour)))
				mock.ExpectQuery("FROM contact_contacts").WillReturnRows(tombstone(&deleted))
				mock.ExpectQuery("FROM contact_address_ownership_periods").WillReturnRows(noPeriod())
				mock.ExpectExec("UPDATE contact_addresses").WillReturnError(errCaseBoom)
			},
			wantText: "could not execute reset. staleRowRepairTx",
			wantIs:   errCaseBoom,
		},
		{
			name:        "reset succeeds and keeps the row",
			resetToNull: true,
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("FROM contact_addresses").WillReturnRows(staleRow(deleted.Add(time.Hour)))
				mock.ExpectQuery("FROM contact_contacts").WillReturnRows(tombstone(&deleted))
				mock.ExpectQuery("FROM contact_address_ownership_periods").WillReturnRows(noPeriod())
				mock.ExpectExec("UPDATE contact_addresses").WithArgs(nil, sqlmock.AnyArg(), rowID.Bytes()).WillReturnResult(sqlmock.NewResult(0, 1))
			},
			wantRetry: true,
		},
		{
			name: "delete deadlock is the bare sentinel",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("FROM contact_addresses").WillReturnRows(staleRow(deleted.Add(time.Hour)))
				mock.ExpectQuery("FROM contact_contacts").WillReturnRows(tombstone(&deleted))
				mock.ExpectQuery("FROM contact_address_ownership_periods").WillReturnRows(noPeriod())
				mock.ExpectExec("DELETE FROM contact_addresses").WillReturnError(mysqlDeadlock())
			},
			wantErr: ErrDeadlock,
		},
		{
			name: "delete failure is returned wrapped",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("FROM contact_addresses").WillReturnRows(staleRow(deleted.Add(time.Hour)))
				mock.ExpectQuery("FROM contact_contacts").WillReturnRows(tombstone(&deleted))
				mock.ExpectQuery("FROM contact_address_ownership_periods").WillReturnRows(noPeriod())
				mock.ExpectExec("DELETE FROM contact_addresses").WillReturnError(errCaseBoom)
			},
			wantText: "could not execute delete. staleRowRepairTx",
			wantIs:   errCaseBoom,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock, _ := newCtxRowsHandler(t)
			tx := openTestTx(t, h, mock)
			tt.setup(mock)

			retry, err := h.staleRowRepairTx(ctx, tx, customerID, commonaddress.TypeTel, target, tt.resetToNull)

			switch {
			case tt.wantErr != nil:
				if err != tt.wantErr {
					t.Errorf("error = %v, want the bare %v", err, tt.wantErr)
				}
				if retry {
					t.Errorf("retry = true on error")
				}
			case tt.wantText != "":
				assertWrappedErr(t, err, tt.wantText, tt.wantIs)
				if retry {
					t.Errorf("retry = true on error")
				}
			default:
				if err != nil {
					t.Fatalf("error = %v", err)
				}
				if retry != tt.wantRetry {
					t.Errorf("retry = %v, want %v", retry, tt.wantRetry)
				}
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet sqlmock expectations: %v", err)
			}
		})
	}
}
