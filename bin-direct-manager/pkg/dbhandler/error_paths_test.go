package dbhandler

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofrs/uuid"

	"monorepo/bin-direct-manager/models/direct"
)

var (
	errExecTest = errors.New("simulated driver failure")
	testRowID   = uuid.FromStringOrNil("9a0f3c4e-1b2d-11f1-8a5e-0b3f6c1d2e77")
)

func Test_NewHandler(t *testing.T) {
	h := NewHandler(dbTest)

	got, ok := h.(*handler)
	if !ok {
		t.Fatalf("NewHandler() returned %T, want *handler", h)
	}
	if got.db != dbTest || got.utilHandler == nil {
		t.Errorf("NewHandler() did not keep the database or create the util handler")
	}
}

func Test_queryFailures(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name string
		call func(h *handler) error
	}{
		{"DirectGet", func(h *handler) error { _, err := h.DirectGet(ctx, testRowID); return err }},
		{"DirectGetByHash", func(h *handler) error { _, err := h.DirectGetByHash(ctx, "direct.abcdef123456"); return err }},
		{"DirectGets", func(h *handler) error {
			_, err := h.DirectGets(ctx, 10, "2026-01-01 00:00:00.000000", nil)
			return err
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock := newCtxTestHandler(t)
			mock.ExpectQuery("SELECT").WillReturnError(errExecTest)

			err := tt.call(h)
			if !errors.Is(err, errExecTest) {
				t.Errorf("error = %v, want the driver failure to be wrapped", err)
			}
			if errors.Is(err, ErrNotFound) {
				t.Errorf("a query failure must not be reported as ErrNotFound")
			}
			if !strings.Contains(err.Error(), tt.name) {
				t.Errorf("error = %q, want it to name %s", err.Error(), tt.name)
			}
		})
	}
}

func Test_execFailures(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name   string
		expect string
		call   func(h *handler) error
	}{
		{"DirectCreate", "INSERT", func(h *handler) error {
			return h.DirectCreate(ctx, &direct.Direct{ResourceType: "extension", Hash: "direct.abcdef123456"})
		}},
		{"DirectUpdate", "UPDATE", func(h *handler) error {
			return h.DirectUpdate(ctx, testRowID, map[direct.Field]any{direct.FieldHash: "direct.fedcba654321"})
		}},
		{"DirectDelete", "DELETE", func(h *handler) error { return h.DirectDelete(ctx, testRowID) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock := newCtxTestHandler(t)
			mock.ExpectExec(tt.expect).WillReturnError(errExecTest)

			err := tt.call(h)
			if !errors.Is(err, errExecTest) {
				t.Errorf("error = %v, want the driver failure to be wrapped", err)
			}
			if !strings.Contains(err.Error(), tt.name) {
				t.Errorf("error = %q, want it to name %s", err.Error(), tt.name)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("the statement was not executed: %v", err)
			}
		})
	}
}

func Test_notFound(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name string
		call func(h *handler) error
	}{
		{"DirectGet", func(h *handler) error { _, err := h.DirectGet(ctx, testRowID); return err }},
		{"DirectGetByHash", func(h *handler) error { _, err := h.DirectGetByHash(ctx, "direct.abcdef123456"); return err }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock := newCtxTestHandler(t)
			cols, _ := directRowValues(t, testRowID)
			mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows(cols))

			if err := tt.call(h); err != ErrNotFound {
				t.Errorf("error = %v, want ErrNotFound", err)
			}
		})
	}
}

// Test_scanFailures feeds a row that does not match the column list of the model, as a
// schema drift would, and expects every reader to return the scan error instead of a
// half filled record.
func Test_scanFailures(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name string
		call func(h *handler) (any, error)
	}{
		{"DirectGet", func(h *handler) (any, error) { return h.DirectGet(ctx, testRowID) }},
		{"DirectGetByHash", func(h *handler) (any, error) { return h.DirectGetByHash(ctx, "direct.abcdef123456") }},
		{"DirectGets", func(h *handler) (any, error) {
			res, err := h.DirectGets(ctx, 10, "2026-01-01 00:00:00.000000", nil)
			return res, err
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock := newCtxTestHandler(t)
			mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"only_one_column"}).AddRow("x"))

			res, err := tt.call(h)
			if err == nil {
				t.Fatalf("error = nil, want a scan error. res: %v", res)
			}
			if !strings.Contains(err.Error(), "could not scan the row") {
				t.Errorf("error = %q, want the scan context", err.Error())
			}
			switch v := res.(type) {
			case *direct.Direct:
				if v != nil {
					t.Errorf("res = %v, want nil", v)
				}
			case []*direct.Direct:
				if v != nil {
					t.Errorf("res = %v, want nil", v)
				}
			}
		})
	}
}

func Test_DirectGets_invalidFilter(t *testing.T) {
	h, _ := newCtxTestHandler(t)

	// no query may be sent: sqlmock fails an unexpected query
	res, err := h.DirectGets(context.Background(), 10, "2026-01-01 00:00:00.000000", map[direct.Field]any{direct.FieldCustomerID: struct{}{}})
	if err == nil {
		t.Fatalf("DirectGets() error = nil, want a filter error. res: %v", res)
	}
	if !strings.Contains(err.Error(), "could not apply filters") {
		t.Errorf("DirectGets() error = %q, want the filter context", err.Error())
	}
}
