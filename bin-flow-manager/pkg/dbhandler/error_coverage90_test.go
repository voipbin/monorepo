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
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-flow-manager/models/activeflow"
	"monorepo/bin-flow-manager/models/flow"
	"monorepo/bin-flow-manager/pkg/cachehandler"
)

var (
	errCoverage90Boom = errors.New("simulated dependency failure")

	coverage90ID = uuid.FromStringOrNil("5b0c7d1e-1b2d-11f1-8a5e-0b3f6c1d2e01")
)

type coverage90Env struct {
	h         *handler
	sqlMock   sqlmock.Sqlmock
	mockCache *cachehandler.MockCacheHandler
	mockUtil  *utilhandler.MockUtilHandler
}

func newCoverage90Env(t *testing.T) *coverage90Env {
	t.Helper()

	db, sqlMock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("could not create sqlmock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	mc := gomock.NewController(t)
	t.Cleanup(mc.Finish)

	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)

	return &coverage90Env{
		h:         &handler{util: mockUtil, db: db, cache: mockCache},
		sqlMock:   sqlMock,
		mockCache: mockCache,
		mockUtil:  mockUtil,
	}
}

func (e *coverage90Env) assertExpectations(t *testing.T) {
	t.Helper()
	if err := e.sqlMock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sql expectations: %v", err)
	}
}

// coverage90ValidRows builds one scannable row for the given columns.
func coverage90ValidRows(columns []string) *sqlmock.Rows {
	values := make([]driver.Value, len(columns))
	for i, c := range columns {
		switch c {
		case "id", "customer_id":
			values[i] = coverage90ID.Bytes()
		case "stack_map", "current_action":
			values[i] = []byte("{}")
		case "executed_actions", "actions":
			values[i] = []byte("[]")
		}
	}
	return sqlmock.NewRows(columns).AddRow(values...)
}

// coverage90ShortRows builds a row that has fewer columns than the model, so the scan fails.
func coverage90ShortRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{"id"}).AddRow(coverage90ID.Bytes())
}

func Test_activeflow_sqlDependencyError_coverage90(t *testing.T) {
	ctx := context.Background()
	curTime := time.Date(2020, 4, 18, 3, 22, 17, 995000000, time.UTC)
	columns := commondatabasehandler.GetDBFields(&activeflow.Activeflow{})

	tests := []struct {
		name string

		setup func(e *coverage90Env)
		call  func(e *coverage90Env) error

		expectIs  error
		expectMsg string
	}{
		{
			name: "create execute error",
			setup: func(e *coverage90Env) {
				e.mockUtil.EXPECT().TimeNow().Return(&curTime)
				e.sqlMock.ExpectExec("INSERT").WillReturnError(errCoverage90Boom)
			},
			call: func(e *coverage90Env) error {
				return e.h.ActiveflowCreate(ctx, &activeflow.Activeflow{})
			},
			expectIs: errCoverage90Boom,
		},
		{
			name: "get from db query error",
			setup: func(e *coverage90Env) {
				e.sqlMock.ExpectQuery("SELECT").WillReturnError(errCoverage90Boom)
			},
			call: func(e *coverage90Env) error {
				_, err := e.h.activeflowGetFromDB(ctx, coverage90ID)
				return err
			},
			expectIs: errCoverage90Boom,
		},
		{
			name: "get from db no rows is not found",
			setup: func(e *coverage90Env) {
				e.sqlMock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows(columns))
			},
			call: func(e *coverage90Env) error {
				_, err := e.h.activeflowGetFromDB(ctx, coverage90ID)
				return err
			},
			expectIs: ErrNotFound,
		},
		{
			name: "get from db scan error",
			setup: func(e *coverage90Env) {
				e.sqlMock.ExpectQuery("SELECT").WillReturnRows(coverage90ShortRows())
			},
			call: func(e *coverage90Env) error {
				_, err := e.h.activeflowGetFromDB(ctx, coverage90ID)
				return err
			},
			expectMsg: "scan failed",
		},
		{
			name: "update to cache returns the db error",
			setup: func(e *coverage90Env) {
				e.sqlMock.ExpectQuery("SELECT").WillReturnError(errCoverage90Boom)
			},
			call: func(e *coverage90Env) error {
				return e.h.activeflowUpdateToCache(ctx, coverage90ID)
			},
			expectIs: errCoverage90Boom,
		},
		{
			name: "update to cache returns the cache error",
			setup: func(e *coverage90Env) {
				e.sqlMock.ExpectQuery("SELECT").WillReturnRows(coverage90ValidRows(columns))
				e.mockCache.EXPECT().ActiveflowSet(ctx, gomock.Any()).Return(errCoverage90Boom)
			},
			call: func(e *coverage90Env) error {
				return e.h.activeflowUpdateToCache(ctx, coverage90ID)
			},
			expectIs: errCoverage90Boom,
		},
		{
			name: "get falls back to the db and returns its error",
			setup: func(e *coverage90Env) {
				e.mockCache.EXPECT().ActiveflowGet(ctx, coverage90ID).Return(nil, errCoverage90Boom)
				e.sqlMock.ExpectQuery("SELECT").WillReturnError(errCoverage90Boom)
			},
			call: func(e *coverage90Env) error {
				res, err := e.h.ActiveflowGet(ctx, coverage90ID)
				if res != nil {
					t.Errorf("no activeflow is expected with the error. res: %v", res)
				}
				return err
			},
			expectIs: errCoverage90Boom,
		},
		{
			name: "get with lock updates the cache then fails on a db error",
			setup: func(e *coverage90Env) {
				e.mockCache.EXPECT().ActiveflowGet(ctx, coverage90ID).Return(&activeflow.Activeflow{}, nil)
				e.sqlMock.ExpectQuery("SELECT").WillReturnError(errCoverage90Boom)
			},
			call: func(e *coverage90Env) error {
				res, err := e.h.ActiveflowGetWithLock(ctx, coverage90ID)
				if res != nil {
					t.Errorf("no activeflow is expected with the error. res: %v", res)
				}
				return err
			},
			expectIs: errCoverage90Boom,
		},
		{
			name: "get with lock returns the lock error after a cache refresh",
			setup: func(e *coverage90Env) {
				e.mockCache.EXPECT().ActiveflowGet(ctx, coverage90ID).Return(&activeflow.Activeflow{}, nil)
				e.sqlMock.ExpectQuery("SELECT").WillReturnRows(coverage90ValidRows(columns))
				e.mockCache.EXPECT().ActiveflowSet(ctx, gomock.Any()).Return(nil)
				e.mockCache.EXPECT().ActiveflowGetWithLock(ctx, coverage90ID).Return(nil, errCoverage90Boom)
			},
			call: func(e *coverage90Env) error {
				_, err := e.h.ActiveflowGetWithLock(ctx, coverage90ID)
				return err
			},
			expectIs: errCoverage90Boom,
		},
		{
			name: "get with lock returns the lock error on a cache miss",
			setup: func(e *coverage90Env) {
				e.mockCache.EXPECT().ActiveflowGet(ctx, coverage90ID).Return(nil, errCoverage90Boom)
				e.mockCache.EXPECT().ActiveflowGetWithLock(ctx, coverage90ID).Return(nil, errCoverage90Boom)
			},
			call: func(e *coverage90Env) error {
				_, err := e.h.ActiveflowGetWithLock(ctx, coverage90ID)
				return err
			},
			expectIs: errCoverage90Boom,
		},
		{
			name: "list query error",
			setup: func(e *coverage90Env) {
				e.sqlMock.ExpectQuery("SELECT").WillReturnError(errCoverage90Boom)
			},
			call: func(e *coverage90Env) error {
				_, err := e.h.ActiveflowList(ctx, "2020-04-18 03:22:17.995000", 10, nil)
				return err
			},
			expectIs: errCoverage90Boom,
		},
		{
			name: "list scan error",
			setup: func(e *coverage90Env) {
				e.sqlMock.ExpectQuery("SELECT").WillReturnRows(coverage90ShortRows())
			},
			call: func(e *coverage90Env) error {
				res, err := e.h.ActiveflowList(ctx, "2020-04-18 03:22:17.995000", 10, nil)
				if res != nil {
					t.Errorf("no list is expected with the error. res: %v", res)
				}
				return err
			},
			expectMsg: "scan failed",
		},
		{
			name: "update execute error",
			setup: func(e *coverage90Env) {
				e.mockUtil.EXPECT().TimeNow().Return(&curTime)
				e.sqlMock.ExpectExec("UPDATE").WillReturnError(errCoverage90Boom)
			},
			call: func(e *coverage90Env) error {
				return e.h.ActiveflowUpdate(ctx, coverage90ID, map[activeflow.Field]any{activeflow.FieldExecuteCount: 1})
			},
			expectIs: errCoverage90Boom,
		},
		{
			name: "delete execute error",
			setup: func(e *coverage90Env) {
				e.mockUtil.EXPECT().TimeNow().Return(&curTime)
				e.sqlMock.ExpectExec("UPDATE").WillReturnError(errCoverage90Boom)
			},
			call: func(e *coverage90Env) error {
				return e.h.ActiveflowDelete(ctx, coverage90ID)
			},
			expectIs: errCoverage90Boom,
		},
		{
			name: "delete rows affected error",
			setup: func(e *coverage90Env) {
				e.mockUtil.EXPECT().TimeNow().Return(&curTime)
				e.sqlMock.ExpectExec("UPDATE").WillReturnResult(sqlmock.NewErrorResult(errCoverage90Boom))
			},
			call: func(e *coverage90Env) error {
				return e.h.ActiveflowDelete(ctx, coverage90ID)
			},
			expectIs: errCoverage90Boom,
		},
		{
			name: "delete without an affected row is not found",
			setup: func(e *coverage90Env) {
				e.mockUtil.EXPECT().TimeNow().Return(&curTime)
				e.sqlMock.ExpectExec("UPDATE").WillReturnResult(sqlmock.NewResult(0, 0))
			},
			call: func(e *coverage90Env) error {
				return e.h.ActiveflowDelete(ctx, coverage90ID)
			},
			expectIs: ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newCoverage90Env(t)
			tt.setup(e)

			err := tt.call(e)
			if err == nil {
				t.Fatalf("expected an error, got nil")
			}
			if tt.expectIs != nil && !errors.Is(err, tt.expectIs) && !errorTextContains(err, tt.expectIs.Error()) {
				t.Errorf("wrong error.\nexpect: %v\ngot: %v", tt.expectIs, err)
			}
			if tt.expectMsg != "" && !errorTextContains(err, tt.expectMsg) {
				t.Errorf("wrong error message.\nexpect to contain: %s\ngot: %v", tt.expectMsg, err)
			}
			e.assertExpectations(t)
		})
	}
}

func Test_activeflow_cacheHit_coverage90(t *testing.T) {
	ctx := context.Background()
	e := newCoverage90Env(t)

	cached := &activeflow.Activeflow{}
	cached.ID = coverage90ID
	e.mockCache.EXPECT().ActiveflowGet(ctx, coverage90ID).Return(cached, nil)

	res, err := e.h.ActiveflowGet(ctx, coverage90ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res != cached {
		t.Errorf("the cached activeflow must be returned as is. got: %v", res)
	}
	// no sql expectation was registered, so any database access fails the test.
	e.assertExpectations(t)
}

func Test_activeflow_getWithLock_success_coverage90(t *testing.T) {
	ctx := context.Background()
	columns := commondatabasehandler.GetDBFields(&activeflow.Activeflow{})
	e := newCoverage90Env(t)

	locked := &activeflow.Activeflow{}
	locked.ID = coverage90ID

	e.mockCache.EXPECT().ActiveflowGet(ctx, coverage90ID).Return(&activeflow.Activeflow{}, nil)
	e.sqlMock.ExpectQuery("SELECT").WillReturnRows(coverage90ValidRows(columns))
	e.mockCache.EXPECT().ActiveflowSet(ctx, gomock.Any()).Return(nil)
	e.mockCache.EXPECT().ActiveflowGetWithLock(ctx, coverage90ID).Return(locked, nil)

	res, err := e.h.ActiveflowGetWithLock(ctx, coverage90ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res != locked {
		t.Errorf("the locked activeflow must be returned. got: %v", res)
	}
	e.assertExpectations(t)
}

func Test_flow_sqlDependencyError_coverage90(t *testing.T) {
	ctx := context.Background()
	curTime := time.Date(2020, 4, 18, 3, 22, 17, 995000000, time.UTC)
	columns := commondatabasehandler.GetDBFields(&flow.Flow{})

	tests := []struct {
		name string

		setup func(e *coverage90Env)
		call  func(e *coverage90Env) error

		expectIs  error
		expectMsg string
	}{
		{
			name: "count query error",
			setup: func(e *coverage90Env) {
				e.sqlMock.ExpectQuery("SELECT COUNT").WillReturnError(errCoverage90Boom)
			},
			call: func(e *coverage90Env) error {
				res, err := e.h.FlowCountByCustomerID(ctx, coverage90ID)
				if res != 0 {
					t.Errorf("zero is expected with the error. res: %d", res)
				}
				return err
			},
			expectIs: errCoverage90Boom,
		},
		{
			name: "create execute error",
			setup: func(e *coverage90Env) {
				e.mockUtil.EXPECT().TimeNow().Return(&curTime)
				e.sqlMock.ExpectExec("INSERT").WillReturnError(errCoverage90Boom)
			},
			call: func(e *coverage90Env) error {
				return e.h.FlowCreate(ctx, &flow.Flow{})
			},
			expectIs: errCoverage90Boom,
		},
		{
			name: "get from db no rows is not found",
			setup: func(e *coverage90Env) {
				e.sqlMock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows(columns))
			},
			call: func(e *coverage90Env) error {
				_, err := e.h.flowGetFromDB(ctx, coverage90ID)
				return err
			},
			expectIs: ErrNotFound,
		},
		{
			name: "get from db query error",
			setup: func(e *coverage90Env) {
				e.sqlMock.ExpectQuery("SELECT").WillReturnError(errCoverage90Boom)
			},
			call: func(e *coverage90Env) error {
				_, err := e.h.flowGetFromDB(ctx, coverage90ID)
				return err
			},
			expectIs: errCoverage90Boom,
		},
		{
			name: "get from db scan error",
			setup: func(e *coverage90Env) {
				e.sqlMock.ExpectQuery("SELECT").WillReturnRows(coverage90ShortRows())
			},
			call: func(e *coverage90Env) error {
				_, err := e.h.flowGetFromDB(ctx, coverage90ID)
				return err
			},
			expectMsg: "scan failed",
		},
		{
			name: "update to cache returns the db error",
			setup: func(e *coverage90Env) {
				e.sqlMock.ExpectQuery("SELECT").WillReturnError(errCoverage90Boom)
			},
			call: func(e *coverage90Env) error {
				return e.h.flowUpdateToCache(ctx, coverage90ID)
			},
			expectIs: errCoverage90Boom,
		},
		{
			name: "update to cache returns the cache error",
			setup: func(e *coverage90Env) {
				e.sqlMock.ExpectQuery("SELECT").WillReturnRows(coverage90ValidRows(columns))
				e.mockCache.EXPECT().FlowSet(ctx, gomock.Any()).Return(errCoverage90Boom)
			},
			call: func(e *coverage90Env) error {
				return e.h.flowUpdateToCache(ctx, coverage90ID)
			},
			expectIs: errCoverage90Boom,
		},
		{
			name: "get falls back to the db and returns its error",
			setup: func(e *coverage90Env) {
				e.mockCache.EXPECT().FlowGet(ctx, coverage90ID).Return(nil, errCoverage90Boom)
				e.sqlMock.ExpectQuery("SELECT").WillReturnError(errCoverage90Boom)
			},
			call: func(e *coverage90Env) error {
				res, err := e.h.FlowGet(ctx, coverage90ID)
				if res != nil {
					t.Errorf("no flow is expected with the error. res: %v", res)
				}
				return err
			},
			expectIs: errCoverage90Boom,
		},
		{
			name: "list query error",
			setup: func(e *coverage90Env) {
				e.sqlMock.ExpectQuery("SELECT").WillReturnError(errCoverage90Boom)
			},
			call: func(e *coverage90Env) error {
				_, err := e.h.FlowList(ctx, "2020-04-18 03:22:17.995000", 10, nil)
				return err
			},
			expectIs: errCoverage90Boom,
		},
		{
			name: "list scan error",
			setup: func(e *coverage90Env) {
				e.sqlMock.ExpectQuery("SELECT").WillReturnRows(coverage90ShortRows())
			},
			call: func(e *coverage90Env) error {
				res, err := e.h.FlowList(ctx, "2020-04-18 03:22:17.995000", 10, nil)
				if res != nil {
					t.Errorf("no list is expected with the error. res: %v", res)
				}
				return err
			},
			expectMsg: "scan failed",
		},
		{
			name: "update execute error",
			setup: func(e *coverage90Env) {
				e.mockUtil.EXPECT().TimeNow().Return(&curTime)
				e.sqlMock.ExpectExec("UPDATE").WillReturnError(errCoverage90Boom)
			},
			call: func(e *coverage90Env) error {
				return e.h.FlowUpdate(ctx, coverage90ID, map[flow.Field]any{flow.FieldName: "name"})
			},
			expectIs: errCoverage90Boom,
		},
		{
			name: "delete execute error",
			setup: func(e *coverage90Env) {
				e.mockUtil.EXPECT().TimeNow().Return(&curTime)
				e.sqlMock.ExpectExec("UPDATE").WillReturnError(errCoverage90Boom)
			},
			call: func(e *coverage90Env) error {
				return e.h.FlowDelete(ctx, coverage90ID)
			},
			expectMsg: errCoverage90Boom.Error(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newCoverage90Env(t)
			tt.setup(e)

			err := tt.call(e)
			if err == nil {
				t.Fatalf("expected an error, got nil")
			}
			if tt.expectIs != nil && !errors.Is(err, tt.expectIs) && !errorTextContains(err, tt.expectIs.Error()) {
				t.Errorf("wrong error.\nexpect: %v\ngot: %v", tt.expectIs, err)
			}
			if tt.expectMsg != "" && !errorTextContains(err, tt.expectMsg) {
				t.Errorf("wrong error message.\nexpect to contain: %s\ngot: %v", tt.expectMsg, err)
			}
			e.assertExpectations(t)
		})
	}
}

func Test_flow_cacheHit_coverage90(t *testing.T) {
	ctx := context.Background()
	e := newCoverage90Env(t)

	cached := &flow.Flow{}
	cached.ID = coverage90ID
	e.mockCache.EXPECT().FlowGet(ctx, coverage90ID).Return(cached, nil)

	res, err := e.h.FlowGet(ctx, coverage90ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res != cached {
		t.Errorf("the cached flow must be returned as is. got: %v", res)
	}
	e.assertExpectations(t)
}

func Test_variable_cacheError_coverage90(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name string
		call func(e *coverage90Env) error
	}{
		{"create returns the cache error", func(e *coverage90Env) error {
			e.mockCache.EXPECT().VariableSet(ctx, gomock.Any()).Return(errCoverage90Boom)
			return e.h.VariableCreate(ctx, nil)
		}},
		{"update returns the cache error", func(e *coverage90Env) error {
			e.mockCache.EXPECT().VariableSet(ctx, gomock.Any()).Return(errCoverage90Boom)
			return e.h.VariableUpdate(ctx, nil)
		}},
		{"get returns the cache error", func(e *coverage90Env) error {
			e.mockCache.EXPECT().VariableGet(ctx, coverage90ID).Return(nil, errCoverage90Boom)
			_, err := e.h.VariableGet(ctx, coverage90ID)
			return err
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newCoverage90Env(t)
			err := tt.call(e)
			if !errors.Is(err, errCoverage90Boom) {
				t.Errorf("wrong error.\nexpect: %v\ngot: %v", errCoverage90Boom, err)
			}
		})
	}
}

func errorTextContains(err error, want string) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), want)
}
