package dbhandler

import (
	"context"
	"errors"
	"strings"
	"testing"

	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-outdial-manager/models/outdial"
	"monorepo/bin-outdial-manager/models/outdialtarget"
	"monorepo/bin-outdial-manager/models/outdialtargetcall"
	"monorepo/bin-outdial-manager/pkg/cachehandler"
)

var errRowsBroken = errors.New("simulated failure while reading rows")

// newMockHandler returns a handler backed by sqlmock. The cache returns a miss
// for the lookups by reference id and activeflow id so that they reach the db.
func newMockHandler(t *testing.T, mc *gomock.Controller) (*handler, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("could not create sqlmock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockUtil.EXPECT().TimeGetCurTime().Return("2026-01-01 00:00:00.000000").AnyTimes()
	mockUtil.EXPECT().TimeNow().Return(nil).AnyTimes()

	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockCache.EXPECT().OutdialTargetCallGetByReferenceID(gomock.Any(), gomock.Any()).Return(nil, errors.New("cache miss")).AnyTimes()
	mockCache.EXPECT().OutdialTargetCallGetByActiveflowID(gomock.Any(), gomock.Any()).Return(nil, errors.New("cache miss")).AnyTimes()

	return &handler{db: db, cache: mockCache, utilHandler: mockUtil}, mock
}

// Test_RowsErr_NotMisreported verifies that a failure while iterating the rows
// is returned as an error: a single-row get must not report it as ErrNotFound
// and a list must not return a partial (here empty) result as a success.
func Test_RowsErr_NotMisreported(t *testing.T) {
	id := uuid.FromStringOrNil("9a0f3c4e-1b2d-11f1-8a5e-0b3f6c1d2e77")
	ctx := context.Background()

	tests := []struct {
		name string

		// prepared is true when the query goes through a prepared statement.
		prepared bool
		// wrapped is true when the cause is wrapped with %w. The list queries
		// that already had rows.Err keep their %v and only carry the text.
		wrapped bool
		fn      func(h *handler) error
	}{
		{"get outdial", false, true, func(h *handler) error { _, err := h.outdialGetFromDB(ctx, id); return err }},
		{"get outdialtarget", false, true, func(h *handler) error { _, err := h.outdialTargetGetFromDB(ctx, id); return err }},
		{"get outdialtargetcall", false, true, func(h *handler) error { _, err := h.outdialTargetCallGetFromDB(ctx, id); return err }},
		{"get outdialtargetcall by reference id", false, true, func(h *handler) error {
			_, err := h.OutdialTargetCallGetByReferenceID(ctx, id)
			return err
		}},
		{"get outdialtargetcall by activeflow id", false, true, func(h *handler) error {
			_, err := h.OutdialTargetCallGetByActiveflowID(ctx, id)
			return err
		}},
		{"list outdial", false, false, func(h *handler) error {
			_, err := h.OutdialList(ctx, "2026-01-01 00:00:00.000000", 10, map[outdial.Field]any{})
			return err
		}},
		{"list outdialtarget", false, false, func(h *handler) error {
			_, err := h.OutdialTargetList(ctx, "2026-01-01 00:00:00.000000", 10, map[outdialtarget.Field]any{})
			return err
		}},
		{"list outdialtargetcall", false, false, func(h *handler) error {
			_, err := h.OutdialTargetCallList(ctx, "2026-01-01 00:00:00.000000", 10, map[outdialtargetcall.Field]any{})
			return err
		}},
		{"get available outdialtargets", true, true, func(h *handler) error {
			_, err := h.OutdialTargetGetAvailable(ctx, id, 1, 1, 1, 1, 1, 10)
			return err
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, mock := newMockHandler(t, mc)

			// the first row fails when it is read, so Next returns false and
			// the cause is only reported by rows.Err().
			rows := sqlmock.NewRows([]string{"id"}).AddRow(id.Bytes()).RowError(0, errRowsBroken)
			if tt.prepared {
				mock.ExpectPrepare("(?i)select").ExpectQuery().WillReturnRows(rows)
			} else {
				mock.ExpectQuery("(?i)select").WillReturnRows(rows)
			}

			err := tt.fn(h)
			if err == nil {
				t.Fatalf("expected an error, got nil")
			}
			if errors.Is(err, ErrNotFound) {
				t.Fatalf("an iteration failure must not be reported as ErrNotFound. err: %v", err)
			}
			if tt.wrapped {
				if !errors.Is(err, errRowsBroken) {
					t.Errorf("expected the iteration failure in the error chain, got: %v", err)
				}
				return
			}
			if !strings.Contains(err.Error(), errRowsBroken.Error()) {
				t.Errorf("expected the iteration failure in the error message, got: %v", err)
			}
		})
	}
}

// Test_OutdialTargetUpdateProgressing_contextCancelled verifies that the request
// context reaches the database driver: a cancelled context aborts the update
// instead of the statement running to completion.
func Test_OutdialTargetUpdateProgressing_contextCancelled(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mock := newMockHandler(t, mc)
	// the cache is not reached: the write fails first, and when the context is
	// ignored the cache refresh fails on the cancelled context and is skipped.
	h.cache = nil

	// registered so that a statement that ignores the context succeeds. It is
	// not consumed when the context is honored, so the expectations are not
	// verified.
	mock.ExpectExec("(?i)update outdial_outdialtargets").WillReturnResult(sqlmock.NewResult(0, 1))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := h.OutdialTargetUpdateProgressing(ctx, uuid.FromStringOrNil("9a0f3c4e-1b2d-11f1-8a5e-0b3f6c1d2e77"), 0)
	if err == nil || !strings.Contains(err.Error(), context.Canceled.Error()) {
		t.Errorf("expected a context canceled error, got: %v", err)
	}
}
