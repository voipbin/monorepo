package dbhandler

import (
	"context"
	"database/sql/driver"
	"errors"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-email-manager/models/email"
	"monorepo/bin-email-manager/pkg/cachehandler"
)

var errRowsBroken = errors.New("simulated failure while reading rows")

// newMockHandler returns a handler backed by sqlmock.
func newMockHandler(t *testing.T) (*handler, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("could not create sqlmock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	mc := gomock.NewController(t)
	t.Cleanup(mc.Finish)

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockUtil.EXPECT().TimeNow().Return(nil).AnyTimes()
	mockUtil.EXPECT().TimeGetCurTime().Return("2026-01-01 00:00:00.000000").AnyTimes()

	return &handler{util: mockUtil, db: db, cache: cachehandler.NewMockCacheHandler(mc)}, mock
}

// fullRow returns the column names and the values of a complete email row,
// in the order of the select list. The values come from PrepareFields, so they
// must stay usable as driver values when a field is added to the model.
func fullRow(t *testing.T, id uuid.UUID) ([]string, []driver.Value) {
	t.Helper()

	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	e := &email.Email{
		Identity: commonidentity.Identity{ID: id},
		TMCreate: &now,
	}
	fields, err := commondatabasehandler.PrepareFields(e)
	if err != nil {
		t.Fatalf("could not prepare fields: %v", err)
	}

	columns := commondatabasehandler.GetDBFields(&email.Email{})
	values := make([]driver.Value, len(columns))
	for i, c := range columns {
		values[i] = fields[c]
	}
	return columns, values
}

// Test_EmailGetFromDB_rowsErr verifies that a failure while iterating the rows
// is returned as an error and is not reported as ErrNotFound.
func Test_EmailGetFromDB_rowsErr(t *testing.T) {
	h, mock := newMockHandler(t)
	id := uuid.FromStringOrNil("9a0f3c4e-1b2d-11f1-8a5e-0b3f6c1d2e77")

	columns, values := fullRow(t, id)
	rows := sqlmock.NewRows(columns).AddRow(values...).RowError(0, errRowsBroken)
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	_, err := h.emailGetFromDB(context.Background(), id)
	if err == nil {
		t.Fatalf("expected an error, got nil")
	}
	if errors.Is(err, ErrNotFound) {
		t.Fatalf("an iteration failure must not be reported as ErrNotFound. err: %v", err)
	}
	if !errors.Is(err, errRowsBroken) {
		t.Errorf("expected the iteration failure in the error, got: %v", err)
	}
}

// Test_EmailGetFromDB_emptyIsNotFound verifies that a clean empty result is
// still reported as ErrNotFound.
func Test_EmailGetFromDB_emptyIsNotFound(t *testing.T) {
	h, mock := newMockHandler(t)
	id := uuid.FromStringOrNil("9a0f3c4e-1b2d-11f1-8a5e-0b3f6c1d2e77")

	columns, _ := fullRow(t, id)
	mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows(columns))

	_, err := h.emailGetFromDB(context.Background(), id)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

// Test_EmailList_rowsErr verifies that a failure after the first row has been
// read is returned as an error instead of a partial list.
func Test_EmailList_rowsErr(t *testing.T) {
	h, mock := newMockHandler(t)
	id := uuid.FromStringOrNil("9a0f3c4e-1b2d-11f1-8a5e-0b3f6c1d2e77")

	columns, values := fullRow(t, id)
	rows := sqlmock.NewRows(columns).AddRow(values...).AddRow(values...).RowError(1, errRowsBroken)
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	res, err := h.EmailList(context.Background(), "2026-01-02 00:00:00.000000", 10, nil)
	if err == nil {
		t.Fatalf("expected an error, got a list of %d", len(res))
	}
	if !errors.Is(err, errRowsBroken) {
		t.Errorf("expected the iteration failure in the error, got: %v", err)
	}
}

// Test_Context_cancel verifies that the request context reaches
// the database driver: a cancelled context aborts the read, list and write
// paths. ExpectationsWereMet is not used because the driver is never reached.
func Test_Context_cancel(t *testing.T) {
	id := uuid.FromStringOrNil("9a0f3c4e-1b2d-11f1-8a5e-0b3f6c1d2e77")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	tests := []struct {
		name string
		fn   func(h *handler) error
	}{
		{"read (emailGetFromDB)", func(h *handler) error { _, err := h.emailGetFromDB(ctx, id); return err }},
		{"list (EmailList)", func(h *handler) error {
			_, err := h.EmailList(ctx, "2026-01-02 00:00:00.000000", 10, nil)
			return err
		}},
		{"write (EmailCreate)", func(h *handler) error { return h.EmailCreate(ctx, &email.Email{}) }},
		{"write (EmailUpdate)", func(h *handler) error {
			return h.EmailUpdate(ctx, id, map[email.Field]any{email.FieldSubject: "s"})
		}},
		{"write (EmailDelete)", func(h *handler) error { return h.EmailDelete(ctx, id) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, _ := newMockHandler(t)

			err := tt.fn(h)
			if !errors.Is(err, context.Canceled) {
				t.Errorf("expected a context canceled error, got: %v", err)
			}
		})
	}
}
