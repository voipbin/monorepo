package dbhandler

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	gomysql "github.com/go-sql-driver/mysql"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-schedule-manager/models/execution"
	"monorepo/bin-schedule-manager/models/schedule"
	"monorepo/bin-schedule-manager/pkg/cachehandler"
)

// newPrivateDB opens a sqlite database that is private to the calling test.
// It deliberately does not use the shared-cache DSN of the package-level
// dbTest, and a single connection keeps the in-memory database stable.
func newPrivateDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", "file::memory:")
	if err != nil {
		t.Fatalf("could not open the private database. err: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() {
		_ = db.Close()
	})

	return db
}

func Test_isDuplicateKeyErr(t *testing.T) {
	tests := []struct {
		name string
		err  error

		expectRes bool
	}{
		{"nil", nil, false},
		{"mysql duplicate entry", &gomysql.MySQLError{Number: 1062, Message: "Duplicate entry"}, true},
		{"mysql duplicate entry wrapped", fmt.Errorf("wrap: %w", &gomysql.MySQLError{Number: 1062}), true},
		{"mysql other errno", &gomysql.MySQLError{Number: 1048, Message: "Column cannot be null"}, false},
		{"plain text containing the number only", errors.New("error 1062 happened"), false},
		{"sqlite unique constraint text", errors.New("UNIQUE constraint failed: schedule_executions.id"), true},
		{"unrelated error", errors.New("connection refused"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isDuplicateKeyErr(tt.err); got != tt.expectRes {
				t.Errorf("isDuplicateKeyErr() = %v, expect: %v", got, tt.expectRes)
			}
		})
	}
}

// Test_ClosedDB_Errors verifies that every database-backed method reports its
// own failure marker when the connection is unusable, and that none of them
// writes to the cache after a failed database call.
func Test_ClosedDB_Errors(t *testing.T) {
	id := uuid.FromStringOrNil("5d2f1a3c-0000-4000-8000-0000000000a1")
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	s := &schedule.Schedule{
		Identity: commonidentity.Identity{ID: id},
		Name:     "closed-db",
		Type:     schedule.TypeRPC,
		Cron:     "0 1 * * *",
	}

	tests := []struct {
		name         string
		cacheMissGet bool // ScheduleGet reads the cache before the database
		call         func(h *handler) error

		expectMarker string
	}{
		{"ScheduleCreate", false, func(h *handler) error { return h.ScheduleCreate(context.Background(), s) }, "could not execute. ScheduleCreate"},
		{"ScheduleGet", true, func(h *handler) error { _, err := h.ScheduleGet(context.Background(), id); return err }, "could not query. scheduleGetFromDB"},
		{"ScheduleList", false, func(h *handler) error {
			_, err := h.ScheduleList(context.Background(), 10, "2999-01-01 00:00:00.000000", nil)
			return err
		}, "could not query. ScheduleList"},
		{"ScheduleUpdate", false, func(h *handler) error {
			return h.ScheduleUpdate(context.Background(), id, map[schedule.Field]any{schedule.FieldDetail: "x"})
		}, "could not execute. ScheduleUpdate"},
		{"ScheduleDelete", false, func(h *handler) error { return h.ScheduleDelete(context.Background(), id) }, "could not execute. ScheduleDelete"},
		{"ScheduleGetByCustomerIDName", false, func(h *handler) error {
			_, err := h.ScheduleGetByCustomerIDName(context.Background(), uuid.Nil, "n")
			return err
		}, "could not query. ScheduleGetByCustomerIDName"},
		{"ScheduleListDue", false, func(h *handler) error { _, err := h.ScheduleListDue(context.Background(), now, 5); return err }, "could not query. ScheduleListDue"},
		{"ScheduleListUninitialized", false, func(h *handler) error {
			_, err := h.ScheduleListUninitialized(context.Background(), 5)
			return err
		}, "could not query. ScheduleListUninitialized"},
		{"ScheduleInitNextRun", false, func(h *handler) error {
			ok, err := h.ScheduleInitNextRun(context.Background(), id, now)
			if ok {
				t.Errorf("ScheduleInitNextRun reported success on a failed update")
			}
			return err
		}, "could not execute. ScheduleInitNextRun"},
		{"ScheduleClaimAndCreateExecution", false, func(h *handler) error {
			res, err := h.ScheduleClaimAndCreateExecution(context.Background(), s, now, now, execution.TriggerTypeCron, now)
			if res != nil {
				t.Errorf("ScheduleClaimAndCreateExecution returned an execution on a failed begin")
			}
			return err
		}, "could not begin transaction. ScheduleClaimAndCreateExecution"},
		{"ExecutionGet", false, func(h *handler) error { _, err := h.ExecutionGet(context.Background(), id); return err }, "could not query. ExecutionGet"},
		{"ExecutionHasRunning", false, func(h *handler) error { _, err := h.ExecutionHasRunning(context.Background(), id); return err }, "could not query. ExecutionHasRunning"},
		{"ExecutionComplete", false, func(h *handler) error {
			ok, err := h.ExecutionComplete(context.Background(), id, execution.StatusSuccess, 200, "", "", 1, 10, now)
			if ok {
				t.Errorf("ExecutionComplete reported success on a failed update")
			}
			return err
		}, "could not execute. ExecutionComplete"},
		{"ExecutionReapAbandoned", false, func(h *handler) error { _, err := h.ExecutionReapAbandoned(context.Background(), now); return err }, "could not execute. ExecutionReapAbandoned"},
		{"ExecutionList", false, func(h *handler) error {
			_, err := h.ExecutionList(context.Background(), 10, "2999-01-01 00:00:00.000000", nil)
			return err
		}, "could not query. ExecutionList"},
		{"ExecutionPrune", false, func(h *handler) error { _, err := h.ExecutionPrune(context.Background(), now, 10); return err }, "could not execute. ExecutionPrune"},
		{"ExecutionCountAll", false, func(h *handler) error { _, err := h.ExecutionCountAll(context.Background()); return err }, "could not query. ExecutionCountAll"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			// no expectations: any cache write after a failed database call fails the test
			mockCache := cachehandler.NewMockCacheHandler(mc)
			if tt.cacheMissGet {
				mockCache.EXPECT().ScheduleGet(gomock.Any(), id).Return(nil, ErrNotFound)
			}

			db := newPrivateDB(t)
			if err := db.Close(); err != nil {
				t.Fatalf("could not close the private database. err: %v", err)
			}

			h := &handler{utilHandler: utilhandler.NewUtilHandler(), db: db, cache: mockCache}

			err := tt.call(h)
			if err == nil {
				t.Fatalf("expected an error from a closed database, got nil")
			}
			if !strings.Contains(err.Error(), tt.expectMarker) {
				t.Errorf("error = %q, expect it to contain %q", err.Error(), tt.expectMarker)
			}
			if !strings.Contains(err.Error(), "database is closed") {
				t.Errorf("error = %q, expect the driver cause to be preserved", err.Error())
			}
		})
	}
}

// Test_ListFilters_Unsupported verifies an unsupported filter value is
// rejected before any query reaches the database.
func Test_ListFilters_Unsupported(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h := &handler{utilHandler: utilhandler.NewUtilHandler(), db: newPrivateDB(t), cache: cachehandler.NewMockCacheHandler(mc)}

	_, err := h.ScheduleList(context.Background(), 10, "2999-01-01 00:00:00.000000", map[schedule.Field]any{schedule.FieldName: []string{"a"}})
	if err == nil || !strings.Contains(err.Error(), "could not apply filters. ScheduleList") {
		t.Errorf("ScheduleList err = %v, expect an apply-filters error", err)
	}

	_, err = h.ExecutionList(context.Background(), 10, "2999-01-01 00:00:00.000000", map[execution.Field]any{execution.FieldStatus: []string{"a"}})
	if err == nil || !strings.Contains(err.Error(), "could not apply filters. ExecutionList") {
		t.Errorf("ExecutionList err = %v, expect an apply-filters error", err)
	}
}

// Test_ScheduleClaimAndCreateExecution_NoSchema verifies the statement-level
// failures inside an open transaction: the cron path fails at the claim
// update, while the manual path skips the claim and returns the raw insert
// error, which must not be classified as a duplicate-key error.
func Test_ScheduleClaimAndCreateExecution_NoSchema(t *testing.T) {
	id := uuid.FromStringOrNil("5d2f1a3c-0000-4000-8000-0000000000b2")
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	s := &schedule.Schedule{Identity: commonidentity.Identity{ID: id}, TimeoutMS: 1000}

	tests := []struct {
		name        string
		triggerType execution.TriggerType

		expectMarker string
	}{
		{"cron fails at the claim update", execution.TriggerTypeCron, "could not execute. scheduleClaimSlot"},
		{"manual fails at the insert", execution.TriggerTypeManual, "no such table"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			// no expectations: nothing may reach the cache on a failed claim
			h := &handler{utilHandler: utilhandler.NewUtilHandler(), db: newPrivateDB(t), cache: cachehandler.NewMockCacheHandler(mc)}

			res, err := h.ScheduleClaimAndCreateExecution(context.Background(), s, now, now, tt.triggerType, now)
			if err == nil {
				t.Fatalf("expected an error, got execution %v", res)
			}
			if res != nil {
				t.Errorf("expected a nil execution, got %v", res)
			}
			if !strings.Contains(err.Error(), tt.expectMarker) {
				t.Errorf("error = %q, expect it to contain %q", err.Error(), tt.expectMarker)
			}
			if isDuplicateKeyErr(err) {
				t.Errorf("a missing-table failure must not be classified as a duplicate-key error: %q", err.Error())
			}
		})
	}
}
