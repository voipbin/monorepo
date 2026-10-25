package dbhandler

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofrs/uuid"

	commonaddress "monorepo/bin-common-handler/models/address"
)

// ownershipHelperCalls lists the single-row readers of the ownership write
// path that share the same error classification: a deadlock becomes the bare
// ErrDeadlock sentinel (callers compare with ==), any other query failure is
// wrapped with the function name and keeps the original error.
func ownershipHelperCalls(ctx context.Context) []struct {
	name     string
	call     func(tx *sql.Tx) error
	queryErr string
	scanErr  string
	columns  []string
} {
	customerID := uuid.Must(uuid.NewV4())
	contactID := uuid.Must(uuid.NewV4())

	return []struct {
		name     string
		call     func(tx *sql.Tx) error
		queryErr string
		scanErr  string
		columns  []string
	}{
		{
			name: "staleRowByTargetTx",
			call: func(tx *sql.Tx) error {
				_, err := staleRowByTargetTx(ctx, tx, customerID, commonaddress.TypeTel, "+1555080001")
				return err
			},
			queryErr: "could not query. staleRowByTargetTx",
			scanErr:  "could not scan. staleRowByTargetTx",
			columns:  []string{"id", "contact_id", "tm_create"},
		},
		{
			name: "contactTombstoneTx",
			call: func(tx *sql.Tx) error {
				_, err := contactTombstoneTx(ctx, tx, contactID)
				return err
			},
			queryErr: "could not query. contactTombstoneTx",
			scanErr:  "could not scan. contactTombstoneTx",
			columns:  []string{"tm_delete"},
		},
		{
			name: "latestOwnershipPeriodValidToTx",
			call: func(tx *sql.Tx) error {
				_, err := latestOwnershipPeriodValidToTx(ctx, tx, customerID, commonaddress.TypeTel, "+1555080001")
				return err
			},
			queryErr: "could not query. latestOwnershipPeriodValidToTx",
			scanErr:  "could not scan. latestOwnershipPeriodValidToTx",
			columns:  []string{"valid_to"},
		},
	}
}

func Test_OwnershipHelpers_QueryAndScanFailures(t *testing.T) {
	ctx := context.Background()

	for _, tt := range ownershipHelperCalls(ctx) {
		t.Run(tt.name+" deadlock", func(t *testing.T) {
			h, mock, _ := newCtxRowsHandler(t)
			mock.ExpectBegin()
			mock.ExpectQuery("SELECT").WillReturnError(mysqlDeadlock())
			tx, err := h.db.BeginTx(ctx, nil)
			if err != nil {
				t.Fatalf("BeginTx() error = %v", err)
			}
			t.Cleanup(func() { _ = tx.Rollback() })

			if err := tt.call(tx); err != ErrDeadlock {
				t.Errorf("error = %v, want the bare ErrDeadlock sentinel", err)
			}
		})

		t.Run(tt.name+" query failure", func(t *testing.T) {
			h, mock, _ := newCtxRowsHandler(t)
			mock.ExpectBegin()
			mock.ExpectQuery("SELECT").WillReturnError(errCaseBoom)
			tx, err := h.db.BeginTx(ctx, nil)
			if err != nil {
				t.Fatalf("BeginTx() error = %v", err)
			}
			t.Cleanup(func() { _ = tx.Rollback() })

			assertWrappedErr(t, tt.call(tx), tt.queryErr, errCaseBoom)
		})

		t.Run(tt.name+" scan failure", func(t *testing.T) {
			h, mock, _ := newCtxRowsHandler(t)
			mock.ExpectBegin()
			// One column more than the function scans.
			cols := append(append([]string{}, tt.columns...), "extra")
			row := make([]driver.Value, len(cols))
			for i := range row {
				row[i] = []byte("x")
			}
			mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows(cols).AddRow(row...))
			tx, err := h.db.BeginTx(ctx, nil)
			if err != nil {
				t.Fatalf("BeginTx() error = %v", err)
			}
			t.Cleanup(func() { _ = tx.Rollback() })

			assertWrappedErr(t, tt.call(tx), tt.scanErr, nil)
		})
	}
}

func Test_staleRowByTargetTx_Results(t *testing.T) {
	ctx := context.Background()
	customerID := uuid.Must(uuid.NewV4())
	rowID := uuid.Must(uuid.NewV4())
	contactID := uuid.Must(uuid.NewV4())
	created := time.Date(2026, 10, 25, 9, 0, 0, 0, time.UTC)

	cols := []string{"id", "contact_id", "tm_create"}

	tests := []struct {
		name       string
		rows       *sqlmock.Rows
		expectNil  bool
		expectText string
	}{
		{name: "no row occupies the slot", rows: sqlmock.NewRows(cols), expectNil: true},
		{
			name: "occupied slot returns id, contact and creation time",
			rows: sqlmock.NewRows(cols).AddRow(rowID.Bytes(), contactID.Bytes(), created.Format("2006-01-02 15:04:05.000000")),
		},
		{
			name: "unresolved row has no contact id",
			rows: sqlmock.NewRows(cols).AddRow(rowID.Bytes(), nil, created.Format("2006-01-02 15:04:05.000000")),
		},
		{
			name:       "malformed id is rejected",
			rows:       sqlmock.NewRows(cols).AddRow([]byte("short"), contactID.Bytes(), created.Format("2006-01-02 15:04:05.000000")),
			expectText: "could not parse id. staleRowByTargetTx",
		},
		{
			name:       "malformed contact id is rejected",
			rows:       sqlmock.NewRows(cols).AddRow(rowID.Bytes(), []byte("short"), created.Format("2006-01-02 15:04:05.000000")),
			expectText: "could not parse contact_id. staleRowByTargetTx",
		},
		{
			name:       "malformed creation time is rejected",
			rows:       sqlmock.NewRows(cols).AddRow(rowID.Bytes(), contactID.Bytes(), "not-a-time"),
			expectText: "could not parse tm_create. staleRowByTargetTx",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock, _ := newCtxRowsHandler(t)
			mock.ExpectBegin()
			mock.ExpectQuery("SELECT").WillReturnRows(tt.rows)
			tx, err := h.db.BeginTx(ctx, nil)
			if err != nil {
				t.Fatalf("BeginTx() error = %v", err)
			}
			t.Cleanup(func() { _ = tx.Rollback() })

			got, err := staleRowByTargetTx(ctx, tx, customerID, commonaddress.TypeTel, "+1555080002")
			if tt.expectText != "" {
				assertWrappedErr(t, err, tt.expectText, nil)
				if got != nil {
					t.Errorf("result = %v, want nil on error", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("error = %v", err)
			}
			if tt.expectNil {
				if got != nil {
					t.Errorf("result = %v, want nil", got)
				}
				return
			}
			if got.ID != rowID || got.TMCreate == nil || !got.TMCreate.Equal(created) {
				t.Errorf("result = %+v, want id %s created %s", got, rowID, created)
			}
			if tt.name == "unresolved row has no contact id" {
				if got.ContactID != uuid.Nil {
					t.Errorf("contact id = %s, want the nil uuid", got.ContactID)
				}
			} else if got.ContactID != contactID {
				t.Errorf("contact id = %s, want %s", got.ContactID, contactID)
			}
		})
	}
}

func Test_contactTombstoneTx_Results(t *testing.T) {
	ctx := context.Background()
	contactID := uuid.Must(uuid.NewV4())
	deleted := time.Date(2026, 10, 24, 8, 0, 0, 0, time.UTC)

	t.Run("nil contact id is live and never queries", func(t *testing.T) {
		h, mock, _ := newCtxRowsHandler(t)
		mock.ExpectBegin()
		tx, err := h.db.BeginTx(ctx, nil)
		if err != nil {
			t.Fatalf("BeginTx() error = %v", err)
		}
		t.Cleanup(func() { _ = tx.Rollback() })

		got, err := contactTombstoneTx(ctx, tx, uuid.Nil)
		if got != nil || err != nil {
			t.Errorf("contactTombstoneTx(nil) = %v, %v, want nil, nil", got, err)
		}
		// An unexpected query would fail with a sqlmock error, so a nil
		// result and nil error also prove that no query was sent.
	})

	tests := []struct {
		name      string
		rows      *sqlmock.Rows
		expectNil bool
	}{
		{name: "contact row is gone", rows: sqlmock.NewRows([]string{"tm_delete"}), expectNil: true},
		{name: "live contact", rows: sqlmock.NewRows([]string{"tm_delete"}).AddRow(nil), expectNil: true},
		{name: "deleted contact", rows: sqlmock.NewRows([]string{"tm_delete"}).AddRow(deleted.Format("2006-01-02 15:04:05.000000"))},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock, _ := newCtxRowsHandler(t)
			mock.ExpectBegin()
			mock.ExpectQuery("SELECT").WillReturnRows(tt.rows)
			tx, err := h.db.BeginTx(ctx, nil)
			if err != nil {
				t.Fatalf("BeginTx() error = %v", err)
			}
			t.Cleanup(func() { _ = tx.Rollback() })

			got, err := contactTombstoneTx(ctx, tx, contactID)
			if err != nil {
				t.Fatalf("error = %v", err)
			}
			if tt.expectNil != (got == nil) {
				t.Fatalf("result = %v, want nil = %v", got, tt.expectNil)
			}
			if got != nil && !got.Equal(deleted) {
				t.Errorf("tm_delete = %s, want %s", got, deleted)
			}
		})
	}
}

func Test_addressTypeTargetContactByID_Failures(t *testing.T) {
	ctx := context.Background()
	id := uuid.Must(uuid.NewV4())
	contactID := uuid.Must(uuid.NewV4())
	cols := []string{"type", "target", "contact_id"}

	tests := []struct {
		name       string
		setup      func(mock sqlmock.Sqlmock)
		expectText string
		expectIs   error
	}{
		{
			name:       "query failure",
			setup:      func(mock sqlmock.Sqlmock) { mock.ExpectQuery("SELECT").WillReturnError(errCaseBoom) },
			expectText: "could not query. addressTypeTargetContactByID",
			expectIs:   errCaseBoom,
		},
		{
			name: "scan failure",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"type"}).AddRow("tel"))
			},
			expectText: "could not scan. addressTypeTargetContactByID",
		},
		{
			name: "malformed contact id",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows(cols).AddRow("tel", "+1555080003", []byte("short")))
			},
			expectText: "could not parse contact_id. addressTypeTargetContactByID",
		},
		{
			name: "row iteration failure",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT").WillReturnRows(rowErrRows(errCaseBoom))
			},
			expectText: "could not iterate rows. addressTypeTargetContactByID",
			expectIs:   errCaseBoom,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock, _ := newCtxRowsHandler(t)
			tt.setup(mock)

			got, err := addressTypeTargetContactByID(ctx, h.db, id)
			if got != nil {
				t.Errorf("result = %v, want nil", got)
			}
			assertWrappedErr(t, err, tt.expectText, tt.expectIs)
		})
	}

	t.Run("resolved row", func(t *testing.T) {
		h, mock, _ := newCtxRowsHandler(t)
		mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows(cols).AddRow("tel", "+1555080003", contactID.Bytes()))

		got, err := addressTypeTargetContactByID(ctx, h.db, id)
		if err != nil || got == nil || got.Type != "tel" || got.Target != "+1555080003" || got.ContactID != contactID {
			t.Errorf("result = %+v, %v, want the row values", got, err)
		}
	})

	t.Run("no row is ErrNotFound", func(t *testing.T) {
		h, mock, _ := newCtxRowsHandler(t)
		mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows(cols))

		if _, err := addressTypeTargetContactByID(ctx, h.db, id); !errors.Is(err, ErrNotFound) {
			t.Errorf("error = %v, want ErrNotFound", err)
		}
	})
}
