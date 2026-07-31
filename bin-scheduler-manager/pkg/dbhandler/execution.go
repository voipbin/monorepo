package dbhandler

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"

	commonidentity "monorepo/bin-common-handler/models/identity"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-scheduler-manager/models/execution"
	"monorepo/bin-scheduler-manager/models/schedule"
)

const (
	schedulerExecutionsTable = "scheduler_executions"
)

// reap deadline budget constants (design §4.2):
// tm_deadline = tm_start + timeout_ms*(retry_max+1) + retryBackoffMS*retry_max + reapMarginMS.
const (
	reapRetryBackoffMS = 5000
	reapMarginMS       = 60000
)

// executionGetFromRow scans a single row into an Execution struct using db tags
func (h *handler) executionGetFromRow(rows *sql.Rows) (*execution.Execution, error) {
	res := &execution.Execution{}
	if err := commondatabasehandler.ScanRow(rows, res); err != nil {
		return nil, fmt.Errorf("could not scan the row. executionGetFromRow. err: %v", err)
	}

	return res, nil
}

// ScheduleClaimAndCreateExecution atomically claims the schedule's slot and
// records the execution row in ONE transaction (design §5.3 steps 3-4).
//
// For a cron trigger it performs the CAS claim:
//
//	UPDATE scheduler_schedules SET tm_next_run=next, tm_last_run=now, tm_update=now
//	 WHERE id=? AND tm_next_run=tmScheduled AND enabled=1 AND tm_delete IS NULL
//
// RowsAffected == 0 means the slot was claimed by another replica (or the
// schedule was disabled/deleted since the scan) — the transaction is rolled
// back and (nil, nil) is returned so the caller can distinguish a lost CAS
// from a real error.
//
// For a manual trigger the schedule UPDATE is skipped entirely (design §6:
// a manual run never touches tm_next_run/tm_last_run, and works on disabled
// schedules too).
//
// The execution INSERT is backstopped by the unique key
// (schedule_id, trigger_type, tm_scheduled): a duplicate-key error also rolls
// back and returns (nil, nil).
func (h *handler) ScheduleClaimAndCreateExecution(ctx context.Context, s *schedule.Schedule, next time.Time, now time.Time, triggerType execution.TriggerType, tmScheduled time.Time) (*execution.Execution, error) {
	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("could not begin transaction. ScheduleClaimAndCreateExecution. err: %v", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if triggerType == execution.TriggerTypeCron {
		claimed, err := h.scheduleClaimSlot(ctx, tx, s.ID, next, now, tmScheduled)
		if err != nil {
			return nil, err
		}
		if !claimed {
			// lost CAS — not an error
			return nil, nil
		}
	}

	res, err := h.executionInsertRunning(ctx, tx, s, triggerType, tmScheduled, now)
	if err != nil {
		if isDuplicateKeyErr(err) {
			// another claimant already recorded this slot — not an error
			return nil, nil
		}
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("could not commit transaction. ScheduleClaimAndCreateExecution. err: %v", err)
	}

	// refresh the schedule cache
	_ = h.scheduleUpdateToCache(ctx, s.ID)

	return res, nil
}

// scheduleClaimSlot performs the CAS claim UPDATE within the given transaction.
// Returns false when the CAS was lost (RowsAffected == 0).
func (h *handler) scheduleClaimSlot(ctx context.Context, tx *sql.Tx, id uuid.UUID, next time.Time, now time.Time, tmScheduled time.Time) (bool, error) {
	fields := map[schedule.Field]any{
		schedule.FieldTMNextRun: next,
		schedule.FieldTMLastRun: now,
		schedule.FieldTMUpdate:  now,
	}

	data, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return false, fmt.Errorf("could not prepare fields. scheduleClaimSlot. err: %v", err)
	}

	query, args, err := sq.Update(schedulerSchedulesTable).
		SetMap(data).
		Where(sq.Eq{string(schedule.FieldID): id.Bytes()}).
		Where(sq.Eq{string(schedule.FieldTMNextRun): tmScheduled}).
		Where(sq.Eq{string(schedule.FieldEnabled): true}).
		Where(sq.Eq{string(schedule.FieldTMDelete): nil}).
		ToSql()
	if err != nil {
		return false, fmt.Errorf("could not build query. scheduleClaimSlot. err: %v", err)
	}

	res, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return false, fmt.Errorf("could not execute. scheduleClaimSlot. err: %v", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("could not get affected rows. scheduleClaimSlot. err: %v", err)
	}

	return affected == 1, nil
}

// executionInsertRunning inserts the running execution row within the given
// transaction. tm_deadline is denormalized at INSERT time (design §4.2).
func (h *handler) executionInsertRunning(ctx context.Context, tx *sql.Tx, s *schedule.Schedule, triggerType execution.TriggerType, tmScheduled time.Time, now time.Time) (*execution.Execution, error) {
	deadlineMS := s.TimeoutMS*(s.RetryMax+1) + reapRetryBackoffMS*s.RetryMax + reapMarginMS
	deadline := now.Add(time.Duration(deadlineMS) * time.Millisecond)

	e := &execution.Execution{
		Identity: commonidentity.Identity{
			ID:         h.utilHandler.UUIDCreate(),
			CustomerID: s.CustomerID,
		},
		ScheduleID: s.ID,

		TriggerType:  triggerType,
		Status:       execution.StatusRunning,
		AttemptCount: 0,

		TMScheduled: &tmScheduled,
		TMDeadline:  &deadline,
		TMStart:     &now,

		TMCreate: &now,
		TMUpdate: &now,
	}

	fields, err := commondatabasehandler.PrepareFields(e)
	if err != nil {
		return nil, fmt.Errorf("could not prepare fields. executionInsertRunning. err: %v", err)
	}

	query, args, err := sq.Insert(schedulerExecutionsTable).SetMap(fields).ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. executionInsertRunning. err: %v", err)
	}

	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return nil, err
	}

	return e, nil
}

// ExecutionGet returns execution.
func (h *handler) ExecutionGet(ctx context.Context, id uuid.UUID) (*execution.Execution, error) {
	// get column names from db tags
	columns := commondatabasehandler.GetDBFields(&execution.Execution{})

	query, args, err := sq.Select(columns...).
		From(schedulerExecutionsTable).
		Where(sq.Eq{string(execution.FieldID): id.Bytes()}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. ExecutionGet. err: %v", err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. ExecutionGet. err: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	if !rows.Next() {
		return nil, ErrNotFound
	}

	res, err := h.executionGetFromRow(rows)
	if err != nil {
		return nil, fmt.Errorf("could not scan the row. ExecutionGet. err: %v", err)
	}

	return res, nil
}

// ExecutionHasRunning reports whether the schedule has a running execution
// (overlap guard, design §5.3 step 2).
func (h *handler) ExecutionHasRunning(ctx context.Context, scheduleID uuid.UUID) (bool, error) {
	query, args, err := sq.Select("count(*)").
		From(schedulerExecutionsTable).
		Where(sq.Eq{string(execution.FieldScheduleID): scheduleID.Bytes()}).
		Where(sq.Eq{string(execution.FieldStatus): string(execution.StatusRunning)}).
		ToSql()
	if err != nil {
		return false, fmt.Errorf("could not build query. ExecutionHasRunning. err: %v", err)
	}

	var count int64
	if err := h.db.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		return false, fmt.Errorf("could not query. ExecutionHasRunning. err: %v", err)
	}

	return count > 0, nil
}

// ExecutionComplete finalizes a running execution. The UPDATE is conditional
// on status='running' so an already-reaped (abandoned) row stays abandoned —
// one terminal state (design §5.6). Returns true when this caller performed
// the completion.
func (h *handler) ExecutionComplete(ctx context.Context, id uuid.UUID, status execution.Status, statusCode int, errStr string, result string, attemptCount int, durationMS int, now time.Time) (bool, error) {
	fields := map[execution.Field]any{
		execution.FieldStatus:       status,
		execution.FieldStatusCode:   statusCode,
		execution.FieldError:        errStr,
		execution.FieldResult:       result,
		execution.FieldAttemptCount: attemptCount,
		execution.FieldDurationMS:   durationMS,
		execution.FieldTMEnd:        now,
		execution.FieldTMUpdate:     now,
	}

	data, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return false, fmt.Errorf("could not prepare fields. ExecutionComplete. err: %v", err)
	}

	query, args, err := sq.Update(schedulerExecutionsTable).
		SetMap(data).
		Where(sq.Eq{string(execution.FieldID): id.Bytes()}).
		Where(sq.Eq{string(execution.FieldStatus): string(execution.StatusRunning)}).
		ToSql()
	if err != nil {
		return false, fmt.Errorf("could not build query. ExecutionComplete. err: %v", err)
	}

	res, err := h.db.ExecContext(ctx, query, args...)
	if err != nil {
		return false, fmt.Errorf("could not execute. ExecutionComplete. err: %v", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("could not get affected rows. ExecutionComplete. err: %v", err)
	}

	return affected == 1, nil
}

// ExecutionReapAbandoned marks running executions past their tm_deadline as
// abandoned (design §5.6). Single-table conditional UPDATE — idempotent and
// safe to run on every replica concurrently. Returns the reaped row count.
func (h *handler) ExecutionReapAbandoned(ctx context.Context, now time.Time) (int64, error) {
	fields := map[execution.Field]any{
		execution.FieldStatus:   execution.StatusAbandoned,
		execution.FieldError:    "replica died mid-dispatch",
		execution.FieldTMEnd:    now,
		execution.FieldTMUpdate: now,
	}

	data, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return 0, fmt.Errorf("could not prepare fields. ExecutionReapAbandoned. err: %v", err)
	}

	query, args, err := sq.Update(schedulerExecutionsTable).
		SetMap(data).
		Where(sq.Eq{string(execution.FieldStatus): string(execution.StatusRunning)}).
		Where(sq.Lt{string(execution.FieldTMDeadline): now}).
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("could not build query. ExecutionReapAbandoned. err: %v", err)
	}

	res, err := h.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("could not execute. ExecutionReapAbandoned. err: %v", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("could not get affected rows. ExecutionReapAbandoned. err: %v", err)
	}

	return affected, nil
}

// ExecutionList returns executions based on filters.
func (h *handler) ExecutionList(ctx context.Context, size uint64, token string, filters map[execution.Field]any) ([]*execution.Execution, error) {
	// get column names from db tags
	columns := commondatabasehandler.GetDBFields(&execution.Execution{})

	builder := sq.Select(columns...).
		From(schedulerExecutionsTable).
		Where(sq.Lt{string(execution.FieldTMCreate): token}).
		OrderBy(string(execution.FieldTMCreate) + " desc").
		Limit(size)

	// apply filters
	builder, err := commondatabasehandler.ApplyFields(builder, filters)
	if err != nil {
		return nil, fmt.Errorf("could not apply filters. ExecutionList. err: %v", err)
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. ExecutionList. err: %v", err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. ExecutionList. err: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	res := []*execution.Execution{}
	for rows.Next() {
		e, err := h.executionGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not scan the row. ExecutionList. err: %v", err)
		}

		res = append(res, e)
	}

	return res, nil
}

// ExecutionPrune hard-deletes up to batch executions older than cutoff
// (retention, design §6.1). The derived-table wrapper around the LIMIT
// subquery is required by MySQL (no LIMIT in IN-subquery, no same-table
// DELETE-from-SELECT without materialization) and is also valid sqlite —
// squirrel cannot express it, hence sq.Expr (sanctioned exception,
// conventions §7.1).
func (h *handler) ExecutionPrune(ctx context.Context, cutoff time.Time, batch uint64) (int64, error) {
	subquery := fmt.Sprintf(
		"id IN (SELECT id FROM (SELECT id FROM %s WHERE tm_create < ? LIMIT ?) tmp_prune)",
		schedulerExecutionsTable,
	)

	query, args, err := sq.Delete(schedulerExecutionsTable).
		Where(sq.Expr(subquery, cutoff, batch)).
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("could not build query. ExecutionPrune. err: %v", err)
	}

	res, err := h.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("could not execute. ExecutionPrune. err: %v", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("could not get affected rows. ExecutionPrune. err: %v", err)
	}

	return affected, nil
}

// ExecutionCountAll returns the total execution row count (execution_rows
// gauge source, design §5.7).
func (h *handler) ExecutionCountAll(ctx context.Context) (int64, error) {
	query, args, err := sq.Select("count(*)").
		From(schedulerExecutionsTable).
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("could not build query. ExecutionCountAll. err: %v", err)
	}

	var count int64
	if err := h.db.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("could not query. ExecutionCountAll. err: %v", err)
	}

	return count, nil
}
