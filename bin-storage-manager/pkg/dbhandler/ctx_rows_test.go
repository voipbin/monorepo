package dbhandler

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"testing"

	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"github.com/mattn/go-sqlite3"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-storage-manager/models/account"
	"monorepo/bin-storage-manager/models/file"
	"monorepo/bin-storage-manager/pkg/cachehandler"
)

var errRowsBroken = errors.New("simulated failure while reading rows")

// failAfterConnector opens a connection to the same shared in-memory sqlite
// database as dbTest. Rows of every query made through it return errRowsBroken
// from Next after `after` rows were delivered.
type failAfterConnector struct {
	after int
}

func (c *failAfterConnector) Connect(_ context.Context) (driver.Conn, error) {
	conn, err := (&sqlite3.SQLiteDriver{}).Open(`file::memory:?cache=shared`)
	if err != nil {
		return nil, err
	}
	return &failAfterConn{Conn: conn, after: c.after}, nil
}

func (c *failAfterConnector) Driver() driver.Driver { return &sqlite3.SQLiteDriver{} }

type failAfterConn struct {
	driver.Conn
	after int
}

func (c *failAfterConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	rows, err := c.Conn.(driver.QueryerContext).QueryContext(ctx, query, args)
	if err != nil {
		return nil, err
	}
	return &failAfterRows{Rows: rows, after: c.after}, nil
}

type failAfterRows struct {
	driver.Rows
	after int
	n     int
}

func (r *failAfterRows) Next(dest []driver.Value) error {
	if r.n >= r.after {
		return errRowsBroken
	}
	r.n++
	return r.Rows.Next(dest)
}

func newFailAfterDB(t *testing.T, after int) *sql.DB {
	t.Helper()

	db := sql.OpenDB(&failAfterConnector{after: after})
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })

	return db
}

func insertTestAccount(t *testing.T, id uuid.UUID) {
	t.Helper()

	if _, err := dbTest.Exec(
		`insert into storage_accounts(id, total_file_count, total_file_size, tm_create) values(?, 5, 50, '2026-01-01 00:00:00.000000')`,
		id.Bytes(),
	); err != nil {
		t.Fatalf("could not insert the account: %v", err)
	}
	t.Cleanup(func() { _, _ = dbTest.Exec(`delete from storage_accounts where id = ?`, id.Bytes()) })
}

func insertTestFile(t *testing.T, id uuid.UUID) {
	t.Helper()

	if _, err := dbTest.Exec(
		`insert into storage_files(id, name, tm_create) values(?, 'before', '2026-01-01 00:00:00.000000')`,
		id.Bytes(),
	); err != nil {
		t.Fatalf("could not insert the file: %v", err)
	}
	t.Cleanup(func() { _, _ = dbTest.Exec(`delete from storage_files where id = ?`, id.Bytes()) })
}

// Test_RowsErr_NotMisreported verifies that a failure while iterating the rows
// is returned as an error carrying the cause: a single-row get must not report
// it as ErrNotFound and a list must not return a partial result as a success.
func Test_RowsErr_NotMisreported(t *testing.T) {
	id := uuid.FromStringOrNil("5c1e8a52-0f3b-4d6a-9b7e-2a4c6e8f1a03")

	tests := []struct {
		name  string
		after int
		fn    func(h *handler) error
	}{
		{"get account", 0, func(h *handler) error { _, err := h.accountGetFromDB(context.Background(), id); return err }},
		{"get file", 0, func(h *handler) error { _, err := h.fileGetFromDB(context.Background(), id); return err }},
		{"list account, fails on the first row", 0, func(h *handler) error {
			_, err := h.AccountList(context.Background(), "", 10, nil)
			return err
		}},
		{"list account, fails after one row", 1, func(h *handler) error {
			_, err := h.AccountList(context.Background(), "", 10, nil)
			return err
		}},
		{"list file, fails on the first row", 0, func(h *handler) error {
			_, err := h.FileList(context.Background(), "", 10, nil)
			return err
		}},
		{"list file, fails after one row", 1, func(h *handler) error {
			_, err := h.FileList(context.Background(), "", 10, nil)
			return err
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockUtil.EXPECT().TimeGetCurTime().Return("2999-01-01 00:00:00.000000").AnyTimes()

			// the list cases need at least one stored row to be delivered
			// before the failure.
			insertTestAccount(t, uuid.Must(uuid.NewV4()))
			insertTestFile(t, uuid.Must(uuid.NewV4()))

			h := &handler{
				util:  mockUtil,
				db:    newFailAfterDB(t, tt.after),
				cache: cachehandler.NewMockCacheHandler(mc),
			}

			err := tt.fn(h)
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

// Test_ContextCancel_WriteAborted verifies that the request context reaches
// the database call: with a cancelled context the four write functions return
// context.Canceled, leave the row unchanged and do not touch the cache.
func Test_ContextCancel_WriteAborted(t *testing.T) {
	accountID := uuid.Must(uuid.NewV4())
	fileID := uuid.Must(uuid.NewV4())
	insertTestAccount(t, accountID)
	insertTestFile(t, fileID)

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockUtil.EXPECT().TimeNow().Return(nil).AnyTimes()
	// no cache expectation: any cache call fails the test.
	h := &handler{
		util:  mockUtil,
		db:    dbTest,
		cache: cachehandler.NewMockCacheHandler(mc),
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	tests := []struct {
		name string
		fn   func() error
	}{
		{"AccountUpdate", func() error {
			return h.AccountUpdate(ctx, accountID, map[account.Field]any{account.FieldTotalFileCount: 99})
		}},
		{"AccountIncreaseFileInfo", func() error { return h.AccountIncreaseFileInfo(ctx, accountID, 1, 10) }},
		{"AccountDecreaseFileInfo", func() error { return h.AccountDecreaseFileInfo(ctx, accountID, 1, 10) }},
		{"FileUpdate", func() error {
			return h.FileUpdate(ctx, fileID, map[file.Field]any{file.FieldName: "after"})
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.fn(); !errors.Is(err, context.Canceled) {
				t.Errorf("expected context.Canceled, got: %v", err)
			}
		})
	}

	var count, size int64
	if err := dbTest.QueryRow(`select total_file_count, total_file_size from storage_accounts where id = ?`, accountID.Bytes()).Scan(&count, &size); err != nil {
		t.Fatalf("could not read the account: %v", err)
	}
	if count != 5 || size != 50 {
		t.Errorf("the account must be unchanged. count: %d, size: %d", count, size)
	}

	var name string
	if err := dbTest.QueryRow(`select name from storage_files where id = ?`, fileID.Bytes()).Scan(&name); err != nil {
		t.Fatalf("could not read the file: %v", err)
	}
	if name != "before" {
		t.Errorf("the file must be unchanged. name: %s", name)
	}
}
