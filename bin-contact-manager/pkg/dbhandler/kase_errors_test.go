package dbhandler

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	mysql_driver "github.com/go-sql-driver/mysql"
	"github.com/gofrs/uuid"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
)

var errCaseBoom = errors.New("injected database error")

// badShapeRows is a result set whose single column cannot be scanned into a
// row struct, so the scan step of a list function fails.
func badShapeRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{"c"}).AddRow([]byte("x"))
}

func assertWrappedErr(t *testing.T, err error, wantText string, wantIs error) {
	t.Helper()

	if err == nil {
		t.Fatalf("expected an error containing %q, got nil", wantText)
	}
	if !strings.Contains(err.Error(), wantText) {
		t.Errorf("error %q does not contain %q", err.Error(), wantText)
	}
	if wantIs != nil && !errors.Is(err, wantIs) {
		t.Errorf("error %v does not wrap %v", err, wantIs)
	}
}

func Test_BeginTx_Failure(t *testing.T) {
	h, mock, _ := newCtxRowsHandler(t)

	mock.ExpectBegin().WillReturnError(errCaseBoom)

	tx, err := h.BeginTx(context.Background())
	if tx != nil {
		t.Errorf("BeginTx() tx = %v, want nil", tx)
	}
	assertWrappedErr(t, err, "BeginTx", errCaseBoom)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func Test_CaseInsert_ErrorClassification(t *testing.T) {
	ctx := context.Background()
	customerID := uuid.Must(uuid.NewV4())

	tests := []struct {
		name       string
		execErr    error
		expectIs   error
		expectText string
	}{
		{
			name:     "open peer unique violation maps to ErrDuplicate",
			execErr:  &mysql_driver.MySQLError{Number: 1062, Message: "Duplicate entry for key 'uq_case_open_peer'"},
			expectIs: ErrDuplicate,
		},
		{
			name:       "other duplicate key is a plain execute error",
			execErr:    &mysql_driver.MySQLError{Number: 1062, Message: "Duplicate entry for key 'PRIMARY'"},
			expectText: "could not execute. CaseInsert",
		},
		{
			name:     "deadlock maps to ErrDeadlock",
			execErr:  mysqlDeadlock(),
			expectIs: ErrDeadlock,
		},
		{
			name:       "other failure is wrapped with the original error",
			execErr:    errCaseBoom,
			expectText: "could not execute. CaseInsert",
			expectIs:   errCaseBoom,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock, _ := newCtxRowsHandler(t)
			mock.ExpectExec("INSERT INTO contact_cases").WillReturnError(tt.execErr)

			err := h.CaseInsert(ctx, newTxTestCase(t, customerID, "+1555060001"))
			if tt.expectText != "" {
				assertWrappedErr(t, err, tt.expectText, tt.expectIs)
			} else if !errors.Is(err, tt.expectIs) {
				t.Errorf("CaseInsert() error = %v, want %v", err, tt.expectIs)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("expectations: %v", err)
			}
		})
	}
}

func Test_CaseGetOpenByPeer_ErrorClassification(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name       string
		setup      func(mock sqlmock.Sqlmock)
		expectIs   error
		expectText string
	}{
		{
			name: "query deadlock maps to ErrDeadlock",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT").WillReturnError(mysqlDeadlock())
			},
			expectIs: ErrDeadlock,
		},
		{
			name: "query failure is wrapped",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT").WillReturnError(errCaseBoom)
			},
			expectText: "could not query. CaseGetOpenByPeer",
			expectIs:   errCaseBoom,
		},
		{
			name: "row iteration deadlock maps to ErrDeadlock",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT").WillReturnRows(rowErrRows(mysqlDeadlock()))
			},
			expectIs: ErrDeadlock,
		},
		{
			name: "row iteration failure is wrapped",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT").WillReturnRows(rowErrRows(errCaseBoom))
			},
			expectText: "could not iterate rows. CaseGetOpenByPeer",
			expectIs:   errCaseBoom,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock, _ := newCtxRowsHandler(t)
			mock.ExpectBegin()
			tt.setup(mock)

			tx, err := h.db.BeginTx(ctx, nil)
			if err != nil {
				t.Fatalf("BeginTx() error = %v", err)
			}

			got, err := h.CaseGetOpenByPeer(ctx, tx, uuid.Must(uuid.NewV4()), commonaddress.TypeTel, "+1555060002", "call")
			if got != nil {
				t.Errorf("CaseGetOpenByPeer() = %v, want nil", got)
			}
			if tt.expectText != "" {
				assertWrappedErr(t, err, tt.expectText, tt.expectIs)
			} else if !errors.Is(err, tt.expectIs) {
				t.Errorf("CaseGetOpenByPeer() error = %v, want %v", err, tt.expectIs)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("expectations: %v", err)
			}
		})
	}
}

func Test_CaseUpdateStatusClosed_ErrorClassification(t *testing.T) {
	ctx := context.Background()
	closedAt := time.Date(2026, 10, 25, 11, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		setup      func(mock sqlmock.Sqlmock)
		expectIs   error
		expectText string
	}{
		{
			name: "deadlock maps to ErrDeadlock",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("UPDATE contact_cases").WillReturnError(mysqlDeadlock())
			},
			expectIs: ErrDeadlock,
		},
		{
			name: "execute failure is wrapped",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("UPDATE contact_cases").WillReturnError(errCaseBoom)
			},
			expectText: "could not execute. CaseUpdateStatusClosed",
			expectIs:   errCaseBoom,
		},
		{
			name: "rows affected failure is wrapped",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("UPDATE contact_cases").WillReturnResult(sqlmock.NewErrorResult(errCaseBoom))
			},
			expectText: "could not get rows affected. CaseUpdateStatusClosed",
			expectIs:   errCaseBoom,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock, _ := newCtxRowsHandler(t)
			tt.setup(mock)

			closed, err := h.CaseUpdateStatusClosed(ctx, uuid.Must(uuid.NewV4()), uuid.Must(uuid.NewV4()), "agent_closed", "agent", nil, &closedAt)
			if closed {
				t.Errorf("CaseUpdateStatusClosed() closed = true on error")
			}
			if tt.expectText != "" {
				assertWrappedErr(t, err, tt.expectText, tt.expectIs)
			} else if !errors.Is(err, tt.expectIs) {
				t.Errorf("CaseUpdateStatusClosed() error = %v, want %v", err, tt.expectIs)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("expectations: %v", err)
			}
		})
	}
}

func Test_CaseWrites_ExecFailure(t *testing.T) {
	ctx := context.Background()
	customerID := uuid.Must(uuid.NewV4())
	caseID := uuid.Must(uuid.NewV4())
	otherID := uuid.Must(uuid.NewV4())

	tests := []struct {
		name       string
		call       func(h *handler, tx *sql.Tx) error
		expectText string
	}{
		{
			name:       "CaseUpdateContactID",
			call:       func(h *handler, _ *sql.Tx) error { return h.CaseUpdateContactID(ctx, customerID, caseID, otherID) },
			expectText: "could not execute. CaseUpdateContactID",
		},
		{
			name: "CaseUpdateContactIDTx",
			call: func(h *handler, tx *sql.Tx) error {
				return h.CaseUpdateContactIDTx(ctx, tx, customerID, caseID, otherID)
			},
			expectText: "could not execute. CaseUpdateContactID",
		},
		{
			name: "CaseUpdateOwner",
			call: func(h *handler, _ *sql.Tx) error {
				return h.CaseUpdateOwner(ctx, customerID, caseID, commonidentity.OwnerTypeAgent, otherID)
			},
			expectText: "could not execute. CaseUpdateOwner",
		},
		{
			name: "CaseUpdateTagIDs",
			call: func(h *handler, _ *sql.Tx) error {
				return h.CaseUpdateTagIDs(ctx, customerID, caseID, []uuid.UUID{otherID})
			},
			expectText: "could not execute. CaseUpdateTagIDs",
		},
		{
			name:       "CaseClearContactID",
			call:       func(h *handler, _ *sql.Tx) error { return h.CaseClearContactID(ctx, customerID, caseID) },
			expectText: "could not execute. CaseClearContactID",
		},
		{
			name:       "CaseClearContactIDTx",
			call:       func(h *handler, tx *sql.Tx) error { return h.CaseClearContactIDTx(ctx, tx, customerID, caseID) },
			expectText: "could not execute. CaseClearContactID",
		},
		{
			name: "CaseInsertTx",
			call: func(h *handler, tx *sql.Tx) error {
				return h.CaseInsertTx(ctx, tx, newTxTestCase(t, customerID, "+1555060003"))
			},
			expectText: "could not execute. CaseInsert",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock, _ := newCtxRowsHandler(t)
			mock.ExpectBegin()
			mock.ExpectExec("(UPDATE|INSERT)").WillReturnError(errCaseBoom)

			tx, err := h.db.BeginTx(ctx, nil)
			if err != nil {
				t.Fatalf("BeginTx() error = %v", err)
			}

			assertWrappedErr(t, tt.call(h, tx), tt.expectText, errCaseBoom)
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("expectations: %v", err)
			}
		})
	}
}

func Test_CaseReads_QueryFailure(t *testing.T) {
	ctx := context.Background()
	customerID := uuid.Must(uuid.NewV4())
	caseID := uuid.Must(uuid.NewV4())

	tests := []struct {
		name       string
		call       func(h *handler, tx *sql.Tx) error
		expectText string
	}{
		{
			name: "CaseGetByID",
			call: func(h *handler, _ *sql.Tx) error {
				_, err := h.CaseGetByID(ctx, caseID)
				return err
			},
			expectText: "could not query. CaseGetByID",
		},
		{
			name: "CaseGetByIDTx",
			call: func(h *handler, tx *sql.Tx) error {
				_, err := h.CaseGetByIDTx(ctx, tx, caseID)
				return err
			},
			expectText: "could not query. CaseGetByID",
		},
		{
			name: "CaseListAll",
			call: func(h *handler, _ *sql.Tx) error {
				_, err := h.CaseListAll(ctx)
				return err
			},
			expectText: "could not query. CaseListAll",
		},
		{
			name: "CaseListUnresolved",
			call: func(h *handler, _ *sql.Tx) error {
				_, err := h.CaseListUnresolved(ctx, customerID)
				return err
			},
			expectText: "could not query. CaseListUnresolved",
		},
		{
			name: "CaseListByOwner",
			call: func(h *handler, _ *sql.Tx) error {
				_, err := h.CaseListByOwner(ctx, customerID, commonidentity.OwnerTypeAgent, caseID)
				return err
			},
			expectText: "could not query. CaseListByOwner",
		},
		{
			name: "CaseList",
			call: func(h *handler, _ *sql.Tx) error {
				_, err := h.CaseList(ctx, customerID, 10, "", "", "", uuid.Nil, uuid.Nil, "")
				return err
			},
			expectText: "could not query. CaseList",
		},
		{
			name: "AddressList",
			call: func(h *handler, _ *sql.Tx) error {
				_, err := h.AddressList(ctx, customerID, map[string]any{}, "", 10)
				return err
			},
			expectText: "could not query. AddressList",
		},
		{
			name: "AddressListAllByContactID",
			call: func(h *handler, _ *sql.Tx) error {
				_, err := h.AddressListAllByContactID(ctx, caseID)
				return err
			},
			expectText: "could not query. AddressListAllByContactID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock, _ := newCtxRowsHandler(t)
			mock.ExpectBegin()
			mock.ExpectQuery("SELECT").WillReturnError(errCaseBoom)

			tx, err := h.db.BeginTx(ctx, nil)
			if err != nil {
				t.Fatalf("BeginTx() error = %v", err)
			}

			assertWrappedErr(t, tt.call(h, tx), tt.expectText, errCaseBoom)
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("expectations: %v", err)
			}
		})
	}
}

func Test_ListReads_ScanAndRowIterationFailure(t *testing.T) {
	ctx := context.Background()
	customerID := uuid.Must(uuid.NewV4())
	ownerID := uuid.Must(uuid.NewV4())

	calls := []struct {
		name       string
		call       func(h *handler) error
		scanText   string
		rowsErrTxt string
	}{
		{
			name: "CaseListAll",
			call: func(h *handler) error {
				_, err := h.CaseListAll(ctx)
				return err
			},
			scanText:   "could not scan the row. CaseListAll",
			rowsErrTxt: "row iteration error. CaseListAll",
		},
		{
			name: "CaseListUnresolved",
			call: func(h *handler) error {
				_, err := h.CaseListUnresolved(ctx, customerID)
				return err
			},
			scanText:   "could not scan the row. CaseListUnresolved",
			rowsErrTxt: "row iteration error. CaseListUnresolved",
		},
		{
			name: "CaseListByOwner",
			call: func(h *handler) error {
				_, err := h.CaseListByOwner(ctx, customerID, commonidentity.OwnerTypeAgent, ownerID)
				return err
			},
			scanText:   "could not scan the row. CaseListByOwner",
			rowsErrTxt: "row iteration error. CaseListByOwner",
		},
		{
			name: "CaseList",
			call: func(h *handler) error {
				_, err := h.CaseList(ctx, customerID, 10, "", "", "", uuid.Nil, uuid.Nil, "")
				return err
			},
			scanText:   "could not scan the row. CaseList",
			rowsErrTxt: "row iteration error. CaseList",
		},
		{
			name: "AddressList",
			call: func(h *handler) error {
				_, err := h.AddressList(ctx, customerID, map[string]any{}, "", 10)
				return err
			},
			scanText:   "could not scan the row. AddressList",
			rowsErrTxt: "row iteration error. AddressList",
		},
		{
			name: "AddressListByContactID",
			call: func(h *handler) error {
				_, err := h.AddressListByContactID(ctx, ownerID)
				return err
			},
			scanText:   "could not scan the row. AddressListByContactID",
			rowsErrTxt: "row iteration error. AddressListByContactID",
		},
		{
			name: "AddressListAllByContactID",
			call: func(h *handler) error {
				_, err := h.AddressListAllByContactID(ctx, ownerID)
				return err
			},
			scanText:   "could not scan the row. AddressListAllByContactID",
			rowsErrTxt: "row iteration error. AddressListAllByContactID",
		},
	}

	for _, c := range calls {
		t.Run(c.name+" scan failure", func(t *testing.T) {
			h, mock, _ := newCtxRowsHandler(t)
			mock.ExpectQuery("SELECT").WillReturnRows(badShapeRows())

			assertWrappedErr(t, c.call(h), c.scanText, nil)
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("expectations: %v", err)
			}
		})

		t.Run(c.name+" row iteration failure", func(t *testing.T) {
			h, mock, _ := newCtxRowsHandler(t)
			mock.ExpectQuery("SELECT").WillReturnRows(rowErrRows(errCaseBoom))

			assertWrappedErr(t, c.call(h), c.rowsErrTxt, errCaseBoom)
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("expectations: %v", err)
			}
		})
	}
}

func Test_CaseList_InvalidPageToken(t *testing.T) {
	h, mock, _ := newCtxRowsHandler(t)

	got, err := h.CaseList(context.Background(), uuid.Must(uuid.NewV4()), 10, "not-a-timestamp", "", "", uuid.Nil, uuid.Nil, "")
	if got != nil {
		t.Errorf("CaseList() = %v, want nil", got)
	}
	assertWrappedErr(t, err, "could not parse page token. CaseList", nil)
	// No query may reach the database for a malformed token.
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}
