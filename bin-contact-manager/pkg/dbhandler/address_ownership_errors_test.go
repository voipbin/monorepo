package dbhandler

// Error path and branch tests for the ownership period helpers. They use
// sqlmock only, so they do not touch the shared sqlite database.

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofrs/uuid"

	commonaddress "monorepo/bin-common-handler/models/address"
)

func Test_OwnershipPeriodsLockAndResolveTx_Failures(t *testing.T) {
	ctx := context.Background()
	customerID := uuid.FromStringOrNil("da000000-0000-0000-0000-000000000001")
	contactID := uuid.FromStringOrNil("da000000-0000-0000-0000-000000000002")
	otherID := uuid.FromStringOrNil("da000000-0000-0000-0000-000000000003")
	periodID := uuid.FromStringOrNil("da000000-0000-0000-0000-000000000004")
	target := "+15550000077"
	valid := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC).Format(repairTimeLayout)
	periodCols := []string{"id", "contact_id", "valid_from", "valid_to"}

	tests := []struct {
		name     string
		setup    func(mock sqlmock.Sqlmock)
		wantErr  error // compared with ==
		wantText string
		wantIs   error
	}{
		{
			name: "query deadlock is the bare sentinel",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("FROM contact_address_ownership_periods").WillReturnError(mysqlDeadlock())
			},
			wantErr: ErrDeadlock,
		},
		{
			name: "query failure is returned wrapped",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("FROM contact_address_ownership_periods").WillReturnError(errCaseBoom)
			},
			wantText: "could not query. OwnershipPeriodsLockAndResolveTx",
			wantIs:   errCaseBoom,
		},
		{
			name: "row scan failure",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("FROM contact_address_ownership_periods").WillReturnRows(twoColumnRow())
			},
			wantText: "could not scan the row. scanOwnershipPeriodRow",
		},
		{
			name: "malformed period id",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("FROM contact_address_ownership_periods").
					WillReturnRows(sqlmock.NewRows(periodCols).AddRow([]byte("short"), contactID.Bytes(), valid, nil))
			},
			wantText: "could not parse id. scanOwnershipPeriodRow",
		},
		{
			name: "malformed contact id",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("FROM contact_address_ownership_periods").
					WillReturnRows(sqlmock.NewRows(periodCols).AddRow(periodID.Bytes(), []byte("short"), valid, nil))
			},
			wantText: "could not parse contact_id. scanOwnershipPeriodRow",
		},
		{
			name: "malformed valid_from",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("FROM contact_address_ownership_periods").
					WillReturnRows(sqlmock.NewRows(periodCols).AddRow(periodID.Bytes(), contactID.Bytes(), "not-a-time", nil))
			},
			wantText: "could not parse valid_from. scanOwnershipPeriodRow",
		},
		{
			name: "malformed valid_to",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("FROM contact_address_ownership_periods").
					WillReturnRows(sqlmock.NewRows(periodCols).AddRow(periodID.Bytes(), contactID.Bytes(), valid, "not-a-time"))
			},
			wantText: "could not parse valid_to. scanOwnershipPeriodRow",
		},
		{
			name: "row iteration failure",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("FROM contact_address_ownership_periods").
					WillReturnRows(sqlmock.NewRows(periodCols).AddRow(periodID.Bytes(), contactID.Bytes(), valid, nil).RowError(0, errCaseBoom))
			},
			wantText: "row iteration error. OwnershipPeriodsLockAndResolveTx",
			wantIs:   errCaseBoom,
		},
		{
			name: "agreement check failure while another contact holds an open period",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("FROM contact_address_ownership_periods").
					WillReturnRows(sqlmock.NewRows(periodCols).AddRow(periodID.Bytes(), otherID.Bytes(), valid, nil))
				mock.ExpectQuery("FROM contact_addresses a JOIN contact_contacts c").WillReturnError(errCaseBoom)
			},
			wantText: "could not query. ownershipAgreementHoldsTx",
			wantIs:   errCaseBoom,
		},
		{
			name: "closing the orphaned open period of another contact fails",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("FROM contact_address_ownership_periods").
					WillReturnRows(sqlmock.NewRows(periodCols).AddRow(periodID.Bytes(), otherID.Bytes(), valid, nil))
				// the address row is owned by somebody else, so the other period is an orphan
				mock.ExpectQuery("FROM contact_addresses a JOIN contact_contacts c").
					WillReturnRows(sqlmock.NewRows([]string{"contact_id", "tm_delete"}).AddRow(contactID.Bytes(), nil))
				mock.ExpectExec("UPDATE contact_address_ownership_periods").WillReturnError(errCaseBoom)
			},
			wantText: "could not execute. ownershipPeriodCloseByIDTx",
			wantIs:   errCaseBoom,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock, _ := newCtxRowsHandler(t)
			tx := openTestTx(t, h, mock)
			tt.setup(mock)

			_, rows, err := h.OwnershipPeriodsLockAndResolveTx(ctx, tx, customerID, contactID, commonaddress.TypeTel, target)
			if rows != nil {
				t.Errorf("rows = %v, want nil on error", rows)
			}
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

func Test_OwnershipPeriodsLockAndResolveTx_ForUpdateSuffix(t *testing.T) {
	ctx := context.Background()
	customerID := uuid.FromStringOrNil("da000000-0000-0000-0000-000000000011")
	contactID := uuid.FromStringOrNil("da000000-0000-0000-0000-000000000012")

	h, mock, _ := newCtxRowsHandler(t)
	h.forUpdateSuffix = "FOR UPDATE"
	tx := openTestTx(t, h, mock)
	mock.ExpectQuery("FROM contact_address_ownership_periods WHERE .* FOR UPDATE").
		WillReturnRows(sqlmock.NewRows([]string{"id", "contact_id", "valid_from", "valid_to"}))

	step, rows, err := h.OwnershipPeriodsLockAndResolveTx(ctx, tx, customerID, contactID, commonaddress.TypeTel, "+15550000078")
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if step != StepFirstRegistration || len(rows) != 0 {
		t.Errorf("step = %d rows = %v, want first registration without rows", step, rows)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// A closed period of another contact without a start time counts as an
// intervening owner, so the contact gets a new period instead of a reopen.
func Test_OwnershipPeriodsLockAndResolveTx_InterveningWithoutStart(t *testing.T) {
	ctx := context.Background()
	customerID := uuid.FromStringOrNil("da000000-0000-0000-0000-000000000021")
	contactID := uuid.FromStringOrNil("da000000-0000-0000-0000-000000000022")
	otherID := uuid.FromStringOrNil("da000000-0000-0000-0000-000000000023")
	ownID := uuid.FromStringOrNil("da000000-0000-0000-0000-000000000024")
	otherPeriodID := uuid.FromStringOrNil("da000000-0000-0000-0000-000000000025")
	closedAt := time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC).Format(repairTimeLayout)

	h, mock, _ := newCtxRowsHandler(t)
	tx := openTestTx(t, h, mock)
	mock.ExpectQuery("FROM contact_address_ownership_periods").
		WillReturnRows(sqlmock.NewRows([]string{"id", "contact_id", "valid_from", "valid_to"}).
			AddRow(ownID.Bytes(), contactID.Bytes(), nil, closedAt).
			AddRow(otherPeriodID.Bytes(), otherID.Bytes(), nil, closedAt))

	step, rows, err := h.OwnershipPeriodsLockAndResolveTx(ctx, tx, customerID, contactID, commonaddress.TypeTel, "+15550000079")
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if step != StepInsertAfterIntervening {
		t.Errorf("step = %d, want StepInsertAfterIntervening (%d)", step, StepInsertAfterIntervening)
	}
	if len(rows) != 2 {
		t.Errorf("rows = %d, want 2", len(rows))
	}
}

func Test_ownershipAgreementHoldsTx_Results(t *testing.T) {
	ctx := context.Background()
	customerID := uuid.FromStringOrNil("da000000-0000-0000-0000-000000000031")
	blockingID := uuid.FromStringOrNil("da000000-0000-0000-0000-000000000032")
	otherID := uuid.FromStringOrNil("da000000-0000-0000-0000-000000000033")
	cols := []string{"contact_id", "tm_delete"}

	tests := []struct {
		name     string
		setup    func(mock sqlmock.Sqlmock)
		want     bool
		wantErr  error // compared with ==
		wantText string
		wantIs   error
	}{
		{
			name: "address row owned by the blocking contact is live",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("FROM contact_addresses a").WillReturnRows(sqlmock.NewRows(cols).AddRow(blockingID.Bytes(), nil))
			},
			want: true,
		},
		{
			name: "address row owned by somebody else is an orphan",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("FROM contact_addresses a").WillReturnRows(sqlmock.NewRows(cols).AddRow(otherID.Bytes(), nil))
			},
			want: false,
		},
		{
			name: "query deadlock is the bare sentinel",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("FROM contact_addresses a").WillReturnError(mysqlDeadlock())
			},
			wantErr: ErrDeadlock,
		},
		{
			name: "query failure is returned wrapped",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("FROM contact_addresses a").WillReturnError(errCaseBoom)
			},
			wantText: "could not query. ownershipAgreementHoldsTx",
			wantIs:   errCaseBoom,
		},
		{
			name: "scan failure",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("FROM contact_addresses a").WillReturnRows(sqlmock.NewRows([]string{"a"}).AddRow([]byte("x")))
			},
			wantText: "could not scan. ownershipAgreementHoldsTx",
		},
		{
			name: "malformed contact id",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("FROM contact_addresses a").WillReturnRows(sqlmock.NewRows(cols).AddRow([]byte("short"), nil))
			},
			wantText: "could not parse contact_id. ownershipAgreementHoldsTx",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock, _ := newCtxRowsHandler(t)
			tx := openTestTx(t, h, mock)
			tt.setup(mock)

			got, err := h.ownershipAgreementHoldsTx(ctx, tx, customerID, blockingID, commonaddress.TypeTel, "+15550000080")
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
			if got != tt.want {
				t.Errorf("holds = %v, want %v", got, tt.want)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet sqlmock expectations: %v", err)
			}
		})
	}
}

func Test_ownershipPeriodWrites_Failures(t *testing.T) {
	ctx := context.Background()
	customerID := uuid.FromStringOrNil("da000000-0000-0000-0000-000000000041")
	contactID := uuid.FromStringOrNil("da000000-0000-0000-0000-000000000042")
	periodID := uuid.FromStringOrNil("da000000-0000-0000-0000-000000000043")
	from := time.Date(2026, 10, 3, 0, 0, 0, 0, time.UTC)

	type writeCall struct {
		name     string
		wantText string
		call     func(h *handler, tx *sql.Tx) error
	}
	writes := []writeCall{
		{
			name:     "close by id",
			wantText: "could not execute. ownershipPeriodCloseByIDTx",
			call: func(h *handler, tx *sql.Tx) error {
				return h.ownershipPeriodCloseByIDTx(ctx, tx, periodID)
			},
		},
		{
			name:     "reopen by id",
			wantText: "could not execute. ownershipPeriodReopenByIDTx",
			call: func(h *handler, tx *sql.Tx) error {
				return h.ownershipPeriodReopenByIDTx(ctx, tx, periodID)
			},
		},
		{
			name:     "insert",
			wantText: "could not execute. ownershipPeriodInsertTx",
			call: func(h *handler, tx *sql.Tx) error {
				return h.ownershipPeriodInsertTx(ctx, tx, customerID, contactID, commonaddress.TypeTel, "+15550000081", &from)
			},
		},
	}

	for _, w := range writes {
		t.Run(w.name+" deadlock", func(t *testing.T) {
			h, mock, _ := newCtxRowsHandler(t)
			tx := openTestTx(t, h, mock)
			mock.ExpectExec("ownership_periods").WillReturnError(mysqlDeadlock())

			if err := w.call(h, tx); err != ErrDeadlock {
				t.Errorf("error = %v, want the bare ErrDeadlock", err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet sqlmock expectations: %v", err)
			}
		})

		t.Run(w.name+" failure", func(t *testing.T) {
			h, mock, _ := newCtxRowsHandler(t)
			tx := openTestTx(t, h, mock)
			mock.ExpectExec("ownership_periods").WillReturnError(errCaseBoom)

			assertWrappedErr(t, w.call(h, tx), w.wantText, errCaseBoom)
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet sqlmock expectations: %v", err)
			}
		})
	}
}

func Test_ownershipPeriodLatestClosedValidTo_SkipsOpenPeriods(t *testing.T) {
	early := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	late := time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC)

	got := ownershipPeriodLatestClosedValidTo([]OwnershipPeriod{
		{ValidTo: nil},
		{ValidTo: &early},
		{ValidTo: nil},
		{ValidTo: &late},
		{ValidTo: &early},
	})
	if got == nil || !got.Equal(late) {
		t.Errorf("latest = %v, want %v", got, late)
	}

	if got := ownershipPeriodLatestClosedValidTo([]OwnershipPeriod{{ValidTo: nil}}); got != nil {
		t.Errorf("latest = %v, want nil when every period is open", got)
	}
}

func Test_applyOpenResolutionTx_Branches(t *testing.T) {
	ctx := context.Background()
	customerID := uuid.FromStringOrNil("da000000-0000-0000-0000-000000000051")
	contactID := uuid.FromStringOrNil("da000000-0000-0000-0000-000000000052")
	otherID := uuid.FromStringOrNil("da000000-0000-0000-0000-000000000053")
	closedA := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	closedB := time.Date(2026, 10, 4, 0, 0, 0, 0, time.UTC)
	newest := uuid.FromStringOrNil("da000000-0000-0000-0000-000000000054")
	older := uuid.FromStringOrNil("da000000-0000-0000-0000-000000000055")

	t.Run("an open period for the contact needs no write", func(t *testing.T) {
		h, mock, _ := newCtxRowsHandler(t)
		tx := openTestTx(t, h, mock)

		if err := h.applyOpenResolutionTx(ctx, tx, StepOpenReuse, nil, customerID, contactID, commonaddress.TypeTel, "+15550000082", nil); err != nil {
			t.Errorf("error = %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet sqlmock expectations: %v", err)
		}
	})

	t.Run("reopen picks the newest closed period of the contact and ignores others", func(t *testing.T) {
		h, mock, _ := newCtxRowsHandler(t)
		tx := openTestTx(t, h, mock)
		mock.ExpectExec("UPDATE contact_address_ownership_periods").
			WithArgs(nil, sqlmock.AnyArg(), newest.Bytes()).
			WillReturnResult(sqlmock.NewResult(0, 1))

		rows := []OwnershipPeriod{
			{ID: older, ContactID: contactID, ValidTo: &closedA},
			{ID: uuid.FromStringOrNil("da000000-0000-0000-0000-000000000056"), ContactID: otherID, ValidTo: &closedB},
			{ID: newest, ContactID: contactID, ValidTo: &closedB},
		}
		if err := h.applyOpenResolutionTx(ctx, tx, StepReopen, rows, customerID, contactID, commonaddress.TypeTel, "+15550000083", nil); err != nil {
			t.Fatalf("error = %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet sqlmock expectations: %v", err)
		}
	})

	t.Run("reopen without an own closed period is an error", func(t *testing.T) {
		h, mock, _ := newCtxRowsHandler(t)
		tx := openTestTx(t, h, mock)

		rows := []OwnershipPeriod{{ContactID: otherID, ValidTo: &closedA}, {ContactID: contactID, ValidTo: nil}}
		err := h.applyOpenResolutionTx(ctx, tx, StepReopen, rows, customerID, contactID, commonaddress.TypeTel, "+15550000084", nil)
		assertWrappedErr(t, err, "could not find own closed period to reopen", nil)
	})

	t.Run("an unknown step is an error", func(t *testing.T) {
		h, mock, _ := newCtxRowsHandler(t)
		tx := openTestTx(t, h, mock)

		err := h.applyOpenResolutionTx(ctx, tx, 999, nil, customerID, contactID, commonaddress.TypeTel, "+15550000085", nil)
		assertWrappedErr(t, err, "unknown step 999", nil)
	})
}
