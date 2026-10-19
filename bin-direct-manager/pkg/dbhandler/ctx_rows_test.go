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
	"go.uber.org/mock/gomock"

	"monorepo/bin-direct-manager/models/direct"
)

var errRowsBroken = errors.New("simulated failure while reading rows")

// directRowValues returns the column values of one valid direct row in the
// order of the db tags, so that ScanRow can scan it.
func directRowValues(t *testing.T, id uuid.UUID) ([]string, []driver.Value) {
	t.Helper()

	cols := commondatabasehandler.GetDBFields(&direct.Direct{})
	vals := make([]driver.Value, len(cols))
	for i, col := range cols {
		switch col {
		case "id":
			vals[i] = id.Bytes()
		case "customer_id", "resource_id":
			vals[i] = uuid.FromStringOrNil("b63b9ce0-7fe1-11ec-8e99-6f2254a33c54").Bytes()
		case "resource_type":
			vals[i] = "extension"
		case "hash":
			vals[i] = "abcdef123456"
		case "tm_create", "tm_update":
			vals[i] = nil
		default:
			t.Fatalf("unexpected column: %s", col)
		}
	}

	return cols, vals
}

func newCtxTestHandler(t *testing.T) (*handler, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("could not create sqlmock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	mc := gomock.NewController(t)

	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockUtil.EXPECT().TimeNow().Return(&now).AnyTimes()
	mockUtil.EXPECT().TimeGetCurTime().Return("2026-01-01 00:00:00.000000").AnyTimes()

	return &handler{db: db, utilHandler: mockUtil}, mock
}

// Test_RowsErr_NotMisreported verifies that a failure while iterating the rows
// is returned as an error: a single-row get must not report it as ErrNotFound
// and a list must not return a partial result as a success.
func Test_RowsErr_NotMisreported(t *testing.T) {
	id := uuid.FromStringOrNil("9a0f3c4e-1b2d-11f1-8a5e-0b3f6c1d2e77")

	tests := []struct {
		name string
		fn   func(h *handler) (any, error)
	}{
		{"get", func(h *handler) (any, error) { return h.DirectGet(context.Background(), id) }},
		{"get by hash", func(h *handler) (any, error) { return h.DirectGetByHash(context.Background(), "abcdef123456") }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock := newCtxTestHandler(t)

			// the first row fails when it is read, so Next returns false and
			// the cause is only reported by rows.Err().
			cols, vals := directRowValues(t, id)
			rows := sqlmock.NewRows(cols).AddRow(vals...).RowError(0, errRowsBroken)
			mock.ExpectQuery("SELECT").WillReturnRows(rows)

			res, err := tt.fn(h)
			if err == nil {
				t.Fatalf("expected an error, got nil. res: %v", res)
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

// Test_RowsErr_DirectGets verifies that a failure after some rows were read
// returns the error and no partial list.
func Test_RowsErr_DirectGets(t *testing.T) {
	h, mock := newCtxTestHandler(t)

	// the first row is scanned successfully, the second one fails.
	cols, vals1 := directRowValues(t, uuid.FromStringOrNil("9a0f3c4e-1b2d-11f1-8a5e-0b3f6c1d2e77"))
	_, vals2 := directRowValues(t, uuid.FromStringOrNil("9a0f3c4e-1b2d-11f1-8a5e-0b3f6c1d2e78"))
	rows := sqlmock.NewRows(cols).AddRow(vals1...).AddRow(vals2...).RowError(1, errRowsBroken)
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	res, err := h.DirectGets(context.Background(), 10, "2026-01-01 00:00:00.000000", nil)
	if err == nil {
		t.Fatalf("expected an error, got nil. res: %v", res)
	}
	if !errors.Is(err, errRowsBroken) {
		t.Errorf("expected the iteration failure in the error, got: %v", err)
	}
	if res != nil {
		t.Errorf("expected no partial result, got: %v", res)
	}
}

// Test_ContextCancel_QueryAborted verifies that the request context reaches
// database/sql: with an already cancelled context the call fails before the
// driver is reached, so the registered expectation is not consumed. A call
// that ignores the context would consume it and succeed.
func Test_ContextCancel_QueryAborted(t *testing.T) {
	id := uuid.FromStringOrNil("9a0f3c4e-1b2d-11f1-8a5e-0b3f6c1d2e77")

	tests := []struct {
		name   string
		expect func(t *testing.T, mock sqlmock.Sqlmock)
		fn     func(ctx context.Context, h *handler) error
	}{
		{
			"create",
			func(t *testing.T, mock sqlmock.Sqlmock) {
				mock.ExpectExec("INSERT INTO").WillReturnResult(sqlmock.NewResult(1, 1))
			},
			func(ctx context.Context, h *handler) error {
				return h.DirectCreate(ctx, &direct.Direct{ResourceType: "extension", Hash: "abcdef123456"})
			},
		},
		{
			"update",
			func(t *testing.T, mock sqlmock.Sqlmock) {
				mock.ExpectExec("UPDATE").WillReturnResult(sqlmock.NewResult(0, 1))
			},
			func(ctx context.Context, h *handler) error {
				return h.DirectUpdate(ctx, id, map[direct.Field]any{direct.FieldHash: "newhash"})
			},
		},
		{
			"delete",
			func(t *testing.T, mock sqlmock.Sqlmock) {
				mock.ExpectExec("DELETE").WillReturnResult(sqlmock.NewResult(0, 1))
			},
			func(ctx context.Context, h *handler) error { return h.DirectDelete(ctx, id) },
		},
		{
			"get",
			func(t *testing.T, mock sqlmock.Sqlmock) {
				cols, vals := directRowValues(t, id)
				mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows(cols).AddRow(vals...))
			},
			func(ctx context.Context, h *handler) error { _, err := h.DirectGet(ctx, id); return err },
		},
		{
			"get by hash",
			func(t *testing.T, mock sqlmock.Sqlmock) {
				cols, vals := directRowValues(t, id)
				mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows(cols).AddRow(vals...))
			},
			func(ctx context.Context, h *handler) error {
				_, err := h.DirectGetByHash(ctx, "abcdef123456")
				return err
			},
		},
		{
			"gets",
			func(t *testing.T, mock sqlmock.Sqlmock) {
				cols, vals := directRowValues(t, id)
				mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows(cols).AddRow(vals...))
			},
			func(ctx context.Context, h *handler) error {
				_, err := h.DirectGets(ctx, 10, "2026-01-01 00:00:00.000000", nil)
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock := newCtxTestHandler(t)
			tt.expect(t, mock)

			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			err := tt.fn(ctx, h)
			if err == nil || !strings.Contains(err.Error(), context.Canceled.Error()) {
				t.Errorf("expected a context canceled error, got: %v", err)
			}
			if errExp := mock.ExpectationsWereMet(); errExp == nil {
				t.Errorf("the query reached the driver although the context was cancelled")
			}
		})
	}
}
