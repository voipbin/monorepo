package dbhandler

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-schedule-manager/models/execution"
	"monorepo/bin-schedule-manager/models/schedule"
	"monorepo/bin-schedule-manager/pkg/cachehandler"
)

const failureTestToken = "2999-01-01 00:00:00.000000"

// newSchemaDB returns a private sqlite database that has the test schema and
// no rows. Nothing written here is visible to the shared package database.
func newSchemaDB(t *testing.T) *sql.DB {
	t.Helper()

	db := newPrivateDB(t)
	for _, name := range []string{"table_schedule_schedules.sql", "table_schedule_executions.sql"} {
		contents, err := os.ReadFile(filepath.Join("..", "..", "scripts", "database_scripts_test", name))
		if err != nil {
			t.Fatalf("could not read the schema file. name: %s, err: %v", name, err)
		}
		if _, err := db.Exec(string(contents)); err != nil {
			t.Fatalf("could not apply the schema file. name: %s, err: %v", name, err)
		}
	}

	return db
}

// Test_ScanErrors verifies that a row whose column holds a value of the wrong
// type fails in the scan step of each read method, with that method's own
// marker. A positive control with the original value runs first, so the
// failure can only come from the corrupted column.
func Test_ScanErrors(t *testing.T) {
	ctx := context.Background()
	id := uuid.FromStringOrNil("5d2f1a3c-0000-4000-8000-0000000000c2")
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	// timeout_ms and status_code are not filter columns of any method below,
	// so the corrupted row is always selected.
	db := newSchemaDB(t)
	if _, err := db.Exec(
		`insert into schedule_schedules(id, customer_id, name, enabled, timeout_ms, tm_next_run, tm_create) values(?, ?, ?, 1, ?, ?, ?)`,
		id.Bytes(), uuid.Nil.Bytes(), "scan-target", 1000, "2020-01-01 00:00:00", "2020-01-01 00:00:00",
	); err != nil {
		t.Fatalf("could not insert the schedule row. err: %v", err)
	}
	if _, err := db.Exec(
		`insert into schedule_executions(id, schedule_id, status_code, tm_create) values(?, ?, ?, ?)`,
		id.Bytes(), id.Bytes(), 200, "2020-01-01 00:00:00",
	); err != nil {
		t.Fatalf("could not insert the execution row. err: %v", err)
	}

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockCache.EXPECT().ScheduleGet(gomock.Any(), id).Return(nil, ErrNotFound).AnyTimes()
	mockCache.EXPECT().ScheduleSet(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	h := &handler{utilHandler: utilhandler.NewUtilHandler(), db: db, cache: mockCache}

	calls := []struct {
		name string
		call func() error

		expectMarker string
	}{
		{"ScheduleGet", func() error { _, err := h.ScheduleGet(ctx, id); return err }, "could not scan the row. scheduleGetFromDB. err"},
		{"ScheduleList", func() error { _, err := h.ScheduleList(ctx, 10, failureTestToken, nil); return err }, "could not scan the row. ScheduleList. err"},
		{"ScheduleGetByCustomerIDName", func() error {
			_, err := h.ScheduleGetByCustomerIDName(ctx, uuid.Nil, "scan-target")
			return err
		}, "could not scan the row. ScheduleGetByCustomerIDName. err"},
		{"ScheduleListDue", func() error { _, err := h.ScheduleListDue(ctx, now, 10); return err }, "could not scan the row. ScheduleListDue. err"},
		{"ExecutionGet", func() error { _, err := h.ExecutionGet(ctx, id); return err }, "could not scan the row. ExecutionGet. err"},
		{"ExecutionList", func() error { _, err := h.ExecutionList(ctx, 10, failureTestToken, nil); return err }, "could not scan the row. ExecutionList. err"},
	}

	// positive control: the untouched rows are readable by every method
	for _, c := range calls {
		if err := c.call(); err != nil {
			t.Fatalf("control: %s failed on a valid row. err: %v", c.name, err)
		}
	}

	if _, err := db.Exec(`update schedule_schedules set timeout_ms = 'not-a-number'`); err != nil {
		t.Fatalf("could not corrupt the schedule row. err: %v", err)
	}
	if _, err := db.Exec(`update schedule_executions set status_code = 'not-a-number'`); err != nil {
		t.Fatalf("could not corrupt the execution row. err: %v", err)
	}

	for _, c := range calls {
		t.Run(c.name, func(t *testing.T) {
			err := c.call()
			if err == nil {
				t.Fatalf("expected a scan error, got nil")
			}
			if !strings.Contains(err.Error(), c.expectMarker) {
				t.Errorf("error = %q, expect it to contain %q", err.Error(), c.expectMarker)
			}
		})
	}

	// the uninitialized scan needs a null next run; the due scan above needed a set one
	if _, err := db.Exec(`update schedule_schedules set tm_next_run = null`); err != nil {
		t.Fatalf("could not clear the next run. err: %v", err)
	}
	_, err := h.ScheduleListUninitialized(ctx, 10)
	if err == nil || !strings.Contains(err.Error(), "could not scan the row. ScheduleListUninitialized. err") {
		t.Errorf("ScheduleListUninitialized err = %v, expect the scan marker", err)
	}
}

// fakeDriver is a minimal database/sql driver whose execution results and
// commit can be made to fail on demand.
type fakeDriver struct {
	rowsAffectedErr bool
	commitErr       bool
}

type (
	fakeConn   struct{ d *fakeDriver }
	fakeStmt   struct{ d *fakeDriver }
	fakeTx     struct{ d *fakeDriver }
	fakeResult struct{ d *fakeDriver }
)

func (d *fakeDriver) Open(string) (driver.Conn, error)         { return &fakeConn{d}, nil }
func (c *fakeConn) Prepare(string) (driver.Stmt, error)        { return &fakeStmt{c.d}, nil }
func (c *fakeConn) Close() error                               { return nil }
func (c *fakeConn) Begin() (driver.Tx, error)                  { return &fakeTx{c.d}, nil }
func (s *fakeStmt) Close() error                               { return nil }
func (s *fakeStmt) NumInput() int                              { return -1 }
func (s *fakeStmt) Exec([]driver.Value) (driver.Result, error) { return &fakeResult{s.d}, nil }
func (s *fakeStmt) Query([]driver.Value) (driver.Rows, error) {
	return nil, errors.New("fake driver: no query")
}
func (t *fakeTx) Rollback() error { return nil }
func (t *fakeTx) Commit() error {
	if t.d.commitErr {
		return errors.New("fake commit failure")
	}
	return nil
}
func (r *fakeResult) LastInsertId() (int64, error) { return 0, nil }
func (r *fakeResult) RowsAffected() (int64, error) {
	if r.d.rowsAffectedErr {
		return 0, errors.New("fake rows affected failure")
	}
	return 1, nil
}

var fakeDriverSeq atomic.Int32

func newFakeDriverDB(t *testing.T, d *fakeDriver) *sql.DB {
	t.Helper()

	name := fmt.Sprintf("schedule-fake-%d", fakeDriverSeq.Add(1))
	sql.Register(name, d)

	db, err := sql.Open(name, "")
	if err != nil {
		t.Fatalf("could not open the fake database. err: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	return db
}

// Test_FakeDriver_ResultAndCommitErrors verifies that a failing RowsAffected
// or Commit is reported with the method's own marker, with the zero value,
// and without touching the cache (the mock has no expectations).
func Test_FakeDriver_ResultAndCommitErrors(t *testing.T) {
	ctx := context.Background()
	id := uuid.FromStringOrNil("5d2f1a3c-0000-4000-8000-0000000000c1")
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	s := &schedule.Schedule{Identity: commonidentity.Identity{ID: id}, TimeoutMS: 1000}

	tests := []struct {
		name   string
		driver *fakeDriver
		call   func(h *handler) (bool, error) // bool reports a non-zero result

		expectMarker string
	}{
		{"ScheduleInitNextRun", &fakeDriver{rowsAffectedErr: true}, func(h *handler) (bool, error) {
			ok, err := h.ScheduleInitNextRun(ctx, id, now)
			return ok, err
		}, "could not get affected rows. ScheduleInitNextRun"},
		{"ExecutionComplete", &fakeDriver{rowsAffectedErr: true}, func(h *handler) (bool, error) {
			ok, err := h.ExecutionComplete(ctx, id, execution.StatusSuccess, 200, "", "", 1, 10, now)
			return ok, err
		}, "could not get affected rows. ExecutionComplete"},
		{"ExecutionReapAbandoned", &fakeDriver{rowsAffectedErr: true}, func(h *handler) (bool, error) {
			n, err := h.ExecutionReapAbandoned(ctx, now)
			return n != 0, err
		}, "could not get affected rows. ExecutionReapAbandoned"},
		{"ExecutionPrune", &fakeDriver{rowsAffectedErr: true}, func(h *handler) (bool, error) {
			n, err := h.ExecutionPrune(ctx, now, 10)
			return n != 0, err
		}, "could not get affected rows. ExecutionPrune"},
		{"scheduleClaimSlot", &fakeDriver{rowsAffectedErr: true}, func(h *handler) (bool, error) {
			res, err := h.ScheduleClaimAndCreateExecution(ctx, s, now, now, execution.TriggerTypeCron, now)
			return res != nil, err
		}, "could not get affected rows. scheduleClaimSlot"},
		{"commit", &fakeDriver{commitErr: true}, func(h *handler) (bool, error) {
			res, err := h.ScheduleClaimAndCreateExecution(ctx, s, now, now, execution.TriggerTypeCron, now)
			return res != nil, err
		}, "could not commit transaction. ScheduleClaimAndCreateExecution"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h := &handler{
				utilHandler: utilhandler.NewUtilHandler(),
				db:          newFakeDriverDB(t, tt.driver),
				cache:       cachehandler.NewMockCacheHandler(mc),
			}

			nonZero, err := tt.call(h)
			if err == nil {
				t.Fatalf("expected an error, got nil")
			}
			if !strings.Contains(err.Error(), tt.expectMarker) {
				t.Errorf("error = %q, expect it to contain %q", err.Error(), tt.expectMarker)
			}
			if nonZero {
				t.Errorf("expected the zero result on failure")
			}
		})
	}
}

func Test_ScheduleGet_CacheAndMiss(t *testing.T) {
	ctx := context.Background()
	id := uuid.FromStringOrNil("5d2f1a3c-0000-4000-8000-0000000000c3")

	t.Run("cache hit does not touch the database", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		cached := &schedule.Schedule{Identity: commonidentity.Identity{ID: id}, Name: "from-cache"}
		mockCache := cachehandler.NewMockCacheHandler(mc)
		mockCache.EXPECT().ScheduleGet(gomock.Any(), id).Return(cached, nil)

		// the schema-less database fails any query, so a database read would error
		h := &handler{utilHandler: utilhandler.NewUtilHandler(), db: newPrivateDB(t), cache: mockCache}

		res, err := h.ScheduleGet(ctx, id)
		if err != nil {
			t.Fatalf("ScheduleGet err = %v, expect nil", err)
		}
		if res != cached {
			t.Errorf("ScheduleGet = %v, expect the cached schedule", res)
		}
	})

	t.Run("cache miss and database miss returns not found without a cache write", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockCache := cachehandler.NewMockCacheHandler(mc)
		mockCache.EXPECT().ScheduleGet(gomock.Any(), id).Return(nil, ErrNotFound)

		h := &handler{utilHandler: utilhandler.NewUtilHandler(), db: newSchemaDB(t), cache: mockCache}

		res, err := h.ScheduleGet(ctx, id)
		if !errors.Is(err, ErrNotFound) || res != nil {
			t.Errorf("ScheduleGet = %v, %v, expect nil and ErrNotFound", res, err)
		}
	})
}

// Test_ScheduleUpdate_CacheRefreshFailureIsIgnored verifies that the write is
// kept and reported as success when the follow-up cache refresh cannot read
// the row or cannot store it.
func Test_ScheduleUpdate_CacheRefreshFailureIsIgnored(t *testing.T) {
	ctx := context.Background()
	id := uuid.FromStringOrNil("5d2f1a3c-0000-4000-8000-0000000000c4")

	readDetail := func(t *testing.T, db *sql.DB) string {
		t.Helper()
		var detail string
		if err := db.QueryRow(`select detail from schedule_schedules where id = ?`, id.Bytes()).Scan(&detail); err != nil {
			t.Fatalf("could not read the detail back. err: %v", err)
		}
		return detail
	}

	t.Run("refresh cannot read the row", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		// no expectations: the cache must not be written when the re-read finds nothing
		h := &handler{utilHandler: utilhandler.NewUtilHandler(), db: newSchemaDB(t), cache: cachehandler.NewMockCacheHandler(mc)}

		if err := h.ScheduleUpdate(ctx, id, map[schedule.Field]any{schedule.FieldDetail: "x"}); err != nil {
			t.Errorf("ScheduleUpdate err = %v, expect nil", err)
		}
	})

	t.Run("refresh cannot store the row", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		db := newSchemaDB(t)
		if _, err := db.Exec(`insert into schedule_schedules(id, customer_id, name, detail, tm_create) values(?, ?, ?, ?, ?)`,
			id.Bytes(), uuid.Nil.Bytes(), "n", "before", "2020-01-01 00:00:00"); err != nil {
			t.Fatalf("could not insert the row. err: %v", err)
		}

		mockCache := cachehandler.NewMockCacheHandler(mc)
		mockCache.EXPECT().ScheduleSet(gomock.Any(), gomock.Any()).Return(errors.New("cache down"))

		h := &handler{utilHandler: utilhandler.NewUtilHandler(), db: db, cache: mockCache}

		if err := h.ScheduleUpdate(ctx, id, map[schedule.Field]any{schedule.FieldDetail: "after"}); err != nil {
			t.Errorf("ScheduleUpdate err = %v, expect nil", err)
		}
		if got := readDetail(t, db); got != "after" {
			t.Errorf("detail = %q, expect the update to be kept", got)
		}
	})
}

// Test_PrepareFields_Rejections verifies that a value that cannot be encoded
// is rejected before any SQL runs, and that nothing is written.
func Test_PrepareFields_Rejections(t *testing.T) {
	ctx := context.Background()
	id := uuid.FromStringOrNil("5d2f1a3c-0000-4000-8000-0000000000c5")

	t.Run("create with invalid json target data", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		db := newSchemaDB(t)
		h := &handler{utilHandler: utilhandler.NewUtilHandler(), db: db, cache: cachehandler.NewMockCacheHandler(mc)}

		err := h.ScheduleCreate(ctx, &schedule.Schedule{
			Identity:   commonidentity.Identity{ID: id},
			TargetData: []byte("{bad"),
		})
		if err == nil || !strings.Contains(err.Error(), "could not prepare fields. ScheduleCreate") {
			t.Fatalf("ScheduleCreate err = %v, expect the prepare-fields marker", err)
		}

		var count int
		if errCount := db.QueryRow(`select count(*) from schedule_schedules`).Scan(&count); errCount != nil || count != 0 {
			t.Errorf("row count = %d, err = %v, expect no row written", count, errCount)
		}
	})

	t.Run("update with an unencodable nested value", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		db := newSchemaDB(t)
		if _, err := db.Exec(`insert into schedule_schedules(id, customer_id, name, detail, tm_create) values(?, ?, ?, ?, ?)`,
			id.Bytes(), uuid.Nil.Bytes(), "n", "unchanged", "2020-01-01 00:00:00"); err != nil {
			t.Fatalf("could not insert the row. err: %v", err)
		}

		h := &handler{utilHandler: utilhandler.NewUtilHandler(), db: db, cache: cachehandler.NewMockCacheHandler(mc)}

		err := h.ScheduleUpdate(ctx, id, map[schedule.Field]any{
			schedule.FieldDetail:     "changed",
			schedule.FieldTargetData: []any{make(chan int)},
		})
		if err == nil || !strings.Contains(err.Error(), "could not prepare fields. ScheduleUpdate") {
			t.Fatalf("ScheduleUpdate err = %v, expect the prepare-fields marker", err)
		}

		var detail string
		if errRead := db.QueryRow(`select detail from schedule_schedules where id = ?`, id.Bytes()).Scan(&detail); errRead != nil || detail != "unchanged" {
			t.Errorf("detail = %q, err = %v, expect the row to be unchanged", detail, errRead)
		}
	})
}
