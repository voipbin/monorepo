package dbhandler

import (
	"context"
	"database/sql/driver"
	"errors"
	"reflect"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/pkg/cachehandler"
)

var (
	errCov2Query = errors.New("cov2 simulated query failure")
	errCov2Row   = errors.New("cov2 simulated row failure")
	errCov2Cache = errors.New("cov2 simulated cache failure")
)

// cov2Env bundles a handler backed by sqlmock and gomock collaborators.
type cov2Env struct {
	h     *handler
	mock  sqlmock.Sqlmock
	cache *cachehandler.MockCacheHandler
	util  *utilhandler.MockUtilHandler
}

func newCov2Env(t *testing.T) cov2Env {
	t.Helper()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("could not create sqlmock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	mc := gomock.NewController(t)
	t.Cleanup(mc.Finish)

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockUtil.EXPECT().TimeNow().Return(nil).AnyTimes()

	return cov2Env{
		h:     &handler{db: db, cache: mockCache, utilHandler: mockUtil},
		mock:  mock,
		cache: mockCache,
		util:  mockUtil,
	}
}

func cov2SelectRe(table string) string {
	return "^SELECT .+ FROM " + regexp.QuoteMeta(table)
}

func cov2InsertRe(table string) string {
	return "^INSERT INTO " + regexp.QuoteMeta(table)
}

func cov2UpdateRe(table string) string {
	return "^UPDATE " + regexp.QuoteMeta(table) + " SET"
}

// cov2NullRows returns one row in which every column is NULL.
func cov2NullRows(cols []string) *sqlmock.Rows {
	vals := make([]driver.Value, len(cols))
	return sqlmock.NewRows(cols).AddRow(vals...)
}

// cov2ShortRows returns one row with fewer columns than the model has, so the scan fails.
func cov2ShortRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{"id"}).AddRow([]byte("x"))
}

// cov2RowErrRows returns a result set whose first row fails while it is read.
func cov2RowErrRows(cols []string) *sqlmock.Rows {
	vals := make([]driver.Value, len(cols))
	return sqlmock.NewRows(cols).AddRow(vals...).RowError(0, errCov2Row)
}

// runCov2SingleGet checks the failure branches of a single-row SELECT.
// prep, when set, is called first to prepare the collaborators (e.g. a cache miss).
func runCov2SingleGet(t *testing.T, table string, cols []string, prep func(e cov2Env), run func(h *handler) error) {
	t.Helper()

	tests := []struct {
		name    string
		rows    *sqlmock.Rows
		qErr    error
		checkFn func(err error) bool
	}{
		{
			name:    "query_error",
			qErr:    errCov2Query,
			checkFn: func(err error) bool { return errors.Is(err, errCov2Query) },
		},
		{
			name:    "row_iteration_error",
			rows:    cov2RowErrRows(cols),
			checkFn: func(err error) bool { return errors.Is(err, errCov2Row) && !errors.Is(err, ErrNotFound) },
		},
		{
			name:    "not_found",
			rows:    sqlmock.NewRows(cols),
			checkFn: func(err error) bool { return errors.Is(err, ErrNotFound) },
		},
		{
			name: "scan_error",
			rows: cov2ShortRows(),
			checkFn: func(err error) bool {
				return err != nil && !errors.Is(err, ErrNotFound) && !errors.Is(err, errCov2Query) && !errors.Is(err, errCov2Row)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newCov2Env(t)
			if prep != nil {
				prep(e)
			}

			exp := e.mock.ExpectQuery(cov2SelectRe(table))
			if tt.qErr != nil {
				exp.WillReturnError(tt.qErr)
			} else {
				exp.WillReturnRows(tt.rows)
			}

			err := run(e.h)
			if !tt.checkFn(err) {
				t.Errorf("Wrong match. got err: %v", err)
			}
			if err := e.mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectations: %v", err)
			}
		})
	}
}

// runCov2List checks the failure branches of a list SELECT. When wantToken is true
// the caller passes an empty page token and the current time is requested once.
func runCov2List(t *testing.T, table string, cols []string, wantToken bool, run func(h *handler) error) {
	t.Helper()

	tests := []struct {
		name    string
		rows    *sqlmock.Rows
		qErr    error
		checkFn func(err error) bool
	}{
		{
			name:    "query_error",
			qErr:    errCov2Query,
			checkFn: func(err error) bool { return errors.Is(err, errCov2Query) },
		},
		{
			name: "scan_error",
			rows: cov2ShortRows(),
			checkFn: func(err error) bool {
				return err != nil && !errors.Is(err, errCov2Query) && !errors.Is(err, errCov2Row)
			},
		},
		{
			name:    "rows_iteration_error",
			rows:    cov2RowErrRows(cols),
			checkFn: func(err error) bool { return errors.Is(err, errCov2Row) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newCov2Env(t)
			if wantToken {
				e.util.EXPECT().TimeGetCurTime().Return("2026-01-02 03:04:05.000000").Times(1)
			}

			exp := e.mock.ExpectQuery(cov2SelectRe(table))
			if tt.qErr != nil {
				exp.WillReturnError(tt.qErr)
			} else {
				exp.WillReturnRows(tt.rows)
			}

			err := run(e.h)
			if !tt.checkFn(err) {
				t.Errorf("Wrong match. got err: %v", err)
			}
			if err := e.mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectations: %v", err)
			}
		})
	}
}

// runCov2ExecFail checks that a failing write is returned (wrapped) and nothing else runs.
func runCov2ExecFail(t *testing.T, stmtRe string, run func(h *handler) error) {
	t.Helper()

	e := newCov2Env(t)
	e.mock.ExpectExec(stmtRe).WillReturnError(errCov2Query)

	err := run(e.h)
	if !errors.Is(err, errCov2Query) {
		t.Errorf("Wrong match. expect: %v, got: %v", errCov2Query, err)
	}
	if err := e.mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// runCov2UpdateToCache checks both failure branches of an <entity>UpdateToCache helper.
func runCov2UpdateToCache(t *testing.T, table string, cols []string, expectSetErr func(e cov2Env), run func(h *handler) error) {
	t.Helper()

	t.Run("read_error", func(t *testing.T) {
		e := newCov2Env(t)
		e.mock.ExpectQuery(cov2SelectRe(table)).WillReturnError(errCov2Query)

		if err := run(e.h); !errors.Is(err, errCov2Query) {
			t.Errorf("Wrong match. expect: %v, got: %v", errCov2Query, err)
		}
		if err := e.mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet expectations: %v", err)
		}
	})

	t.Run("cache_set_error", func(t *testing.T) {
		e := newCov2Env(t)
		e.mock.ExpectQuery(cov2SelectRe(table)).WillReturnRows(cov2NullRows(cols))
		expectSetErr(e)

		if err := run(e.h); !errors.Is(err, errCov2Cache) {
			t.Errorf("Wrong match. expect: %v, got: %v", errCov2Cache, err)
		}
		if err := e.mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet expectations: %v", err)
		}
	})
}

func Test_Call_cov2_singleGetFailures(t *testing.T) {
	ctx := context.Background()
	id := uuid.FromStringOrNil("c2a00001-1b2d-11f1-8a5e-0b3f6c1d2e01")
	cols := commondatabasehandler.GetDBFields(&call.Call{})

	t.Run("CallGetByChannelID", func(t *testing.T) {
		runCov2SingleGet(t, callTable, cols, nil, func(h *handler) error {
			_, err := h.CallGetByChannelID(ctx, "channel-cov2")
			return err
		})
	})

	t.Run("callGetFromDB", func(t *testing.T) {
		runCov2SingleGet(t, callTable, cols, nil, func(h *handler) error {
			_, err := h.callGetFromDB(ctx, id)
			return err
		})
	})

	t.Run("CallGet_db_error_after_cache_miss", func(t *testing.T) {
		prep := func(e cov2Env) {
			e.cache.EXPECT().CallGet(gomock.Any(), id).Return(nil, errCov2Cache).Times(1)
		}
		runCov2SingleGet(t, callTable, cols, prep, func(h *handler) error {
			res, err := h.CallGet(ctx, id)
			if err != nil && res != nil {
				t.Errorf("expected no call together with an error, got: %v", res)
			}
			return err
		})
	})

	t.Run("CallTXStart_scan_error", func(t *testing.T) {
		e := newCov2Env(t)
		e.mock.ExpectBegin()
		e.mock.ExpectQuery(cov2SelectRe(callTable) + ".* FOR UPDATE").WillReturnRows(cov2ShortRows())
		e.mock.ExpectRollback()

		tx, c, err := e.h.CallTXStart(ctx, id)
		if err == nil || errors.Is(err, ErrNotFound) {
			t.Errorf("expected a scan error, got: %v", err)
		}
		if tx != nil || c != nil {
			t.Errorf("expected no transaction and no call. tx: %v, call: %v", tx, c)
		}
		if err := e.mock.ExpectationsWereMet(); err != nil {
			t.Errorf("the transaction must be rolled back. err: %v", err)
		}
	})
}

func Test_Call_cov2_CallGet_cacheHit(t *testing.T) {
	ctx := context.Background()
	id := uuid.FromStringOrNil("c2a00002-1b2d-11f1-8a5e-0b3f6c1d2e01")
	want := &call.Call{}
	want.ID = id

	e := newCov2Env(t)
	e.cache.EXPECT().CallGet(gomock.Any(), id).Return(want, nil).Times(1)

	got, err := e.h.CallGet(ctx, id)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", want, got)
	}
	if err := e.mock.ExpectationsWereMet(); err != nil {
		t.Errorf("the DB must not be touched on a cache hit. err: %v", err)
	}
}

func Test_Call_cov2_CallGet_dbHitSetsCache(t *testing.T) {
	ctx := context.Background()
	id := uuid.FromStringOrNil("c2a00003-1b2d-11f1-8a5e-0b3f6c1d2e01")
	cols := commondatabasehandler.GetDBFields(&call.Call{})

	e := newCov2Env(t)
	e.cache.EXPECT().CallGet(gomock.Any(), id).Return(nil, errCov2Cache).Times(1)
	e.mock.ExpectQuery(cov2SelectRe(callTable)).WillReturnRows(cov2NullRows(cols))
	e.cache.EXPECT().CallSet(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	got, err := e.h.CallGet(ctx, id)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if got == nil {
		t.Fatalf("expected a call, got nil")
	}
	if got.ChainedCallIDs == nil || len(got.ChainedCallIDs) != 0 {
		t.Errorf("Wrong match. expect: empty ChainedCallIDs, got: %v", got.ChainedCallIDs)
	}
	if err := e.mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func Test_Call_cov2_CallList_failures(t *testing.T) {
	ctx := context.Background()
	cols := commondatabasehandler.GetDBFields(&call.Call{})

	runCov2List(t, callTable, cols, true, func(h *handler) error {
		res, err := h.CallList(ctx, 10, "", map[call.Field]any{})
		if err != nil && res != nil {
			t.Errorf("expected no result together with an error, got: %v", res)
		}
		return err
	})
}

func Test_Call_cov2_updateToCache(t *testing.T) {
	ctx := context.Background()
	id := uuid.FromStringOrNil("c2a00004-1b2d-11f1-8a5e-0b3f6c1d2e01")
	cols := commondatabasehandler.GetDBFields(&call.Call{})

	expectSetErr := func(e cov2Env) {
		e.cache.EXPECT().CallSet(gomock.Any(), gomock.Any()).Return(errCov2Cache).Times(1)
	}

	runCov2UpdateToCache(t, callTable, cols, expectSetErr, func(h *handler) error {
		return h.callUpdateToCache(ctx, id)
	})
}

func Test_Call_cov2_writeFailures(t *testing.T) {
	ctx := context.Background()
	id := uuid.FromStringOrNil("c2a00005-1b2d-11f1-8a5e-0b3f6c1d2e01")
	ts := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	tests := []struct {
		name   string
		stmtRe string
		run    func(h *handler) error
	}{
		{
			name:   "CallCreate",
			stmtRe: cov2InsertRe(callTable),
			run: func(h *handler) error {
				c := &call.Call{}
				c.ID = id
				return h.CallCreate(ctx, c)
			},
		},
		{
			name:   "CallUpdate",
			stmtRe: cov2UpdateRe(callTable),
			run: func(h *handler) error {
				return h.CallUpdate(ctx, id, map[call.Field]any{call.FieldBridgeID: "bridge-cov2", call.FieldTMUpdate: &ts})
			},
		},
		{
			name:   "CallDelete",
			stmtRe: cov2UpdateRe(callTable),
			run: func(h *handler) error {
				return h.CallDelete(ctx, id)
			},
		},
		{
			name:   "CallSetHangupIfChannel",
			stmtRe: cov2UpdateRe(callTable),
			run: func(h *handler) error {
				ok, err := h.CallSetHangupIfChannel(ctx, id, "channel-cov2", call.HangupReasonNormal, call.HangupByRemote)
				if ok {
					t.Errorf("expected false together with an error")
				}
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runCov2ExecFail(t, tt.stmtRe, tt.run)
		})
	}
}

func Test_Call_cov2_CallUpdate_emptyFields(t *testing.T) {
	e := newCov2Env(t)

	if err := e.h.CallUpdate(context.Background(), uuid.Nil, map[call.Field]any{}); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
	if err := e.mock.ExpectationsWereMet(); err != nil {
		t.Errorf("nothing may be executed for empty fields. err: %v", err)
	}
}

func Test_Call_cov2_callUpdateWhere_rowsAffectedError(t *testing.T) {
	ctx := context.Background()
	id := uuid.FromStringOrNil("c2a00006-1b2d-11f1-8a5e-0b3f6c1d2e01")

	e := newCov2Env(t)
	e.mock.ExpectExec(cov2UpdateRe(callTable)).WillReturnResult(sqlmock.NewErrorResult(errCov2Row))

	ok, err := e.h.CallSetHangupIfChannel(ctx, id, "channel-cov2", call.HangupReasonNormal, call.HangupByRemote)
	if ok {
		t.Errorf("Wrong match. expect: false, got: true")
	}
	if !errors.Is(err, errCov2Row) {
		t.Errorf("Wrong match. expect: %v, got: %v", errCov2Row, err)
	}
	if err := e.mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}
