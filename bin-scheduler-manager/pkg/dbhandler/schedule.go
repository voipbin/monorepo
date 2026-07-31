package dbhandler

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-scheduler-manager/models/schedule"
)

const (
	schedulerSchedulesTable = "scheduler_schedules"
)

// scheduleGetFromRow scans a single row into a Schedule struct using db tags
func (h *handler) scheduleGetFromRow(rows *sql.Rows) (*schedule.Schedule, error) {
	res := &schedule.Schedule{}
	if err := commondatabasehandler.ScanRow(rows, res); err != nil {
		return nil, fmt.Errorf("could not scan the row. scheduleGetFromRow. err: %v", err)
	}

	return res, nil
}

// ScheduleCreate creates new schedule record
func (h *handler) ScheduleCreate(ctx context.Context, s *schedule.Schedule) error {
	ts := h.utilHandler.TimeNow()
	s.TMCreate = ts
	s.TMUpdate = ts
	s.TMDelete = nil

	// prepare fields for insert
	fields, err := commondatabasehandler.PrepareFields(s)
	if err != nil {
		return fmt.Errorf("could not prepare fields. ScheduleCreate. err: %v", err)
	}

	query, args, err := sq.Insert(schedulerSchedulesTable).SetMap(fields).ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. ScheduleCreate. err: %v", err)
	}

	_, err = h.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("could not execute. ScheduleCreate. err: %v", err)
	}

	// update the cache
	_ = h.scheduleUpdateToCache(ctx, s.ID)

	return nil
}

// scheduleUpdateToCache gets the schedule from the DB and updates the cache.
func (h *handler) scheduleUpdateToCache(ctx context.Context, id uuid.UUID) error {
	res, err := h.scheduleGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.scheduleSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// scheduleSetToCache sets the given schedule to the cache
func (h *handler) scheduleSetToCache(ctx context.Context, s *schedule.Schedule) error {
	if err := h.cache.ScheduleSet(ctx, s); err != nil {
		return err
	}

	return nil
}

// scheduleGetFromCache returns schedule from the cache.
func (h *handler) scheduleGetFromCache(ctx context.Context, id uuid.UUID) (*schedule.Schedule, error) {
	res, err := h.cache.ScheduleGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// scheduleGetFromDB returns schedule from the DB.
func (h *handler) scheduleGetFromDB(ctx context.Context, id uuid.UUID) (*schedule.Schedule, error) {
	// get column names from db tags
	columns := commondatabasehandler.GetDBFields(&schedule.Schedule{})

	query, args, err := sq.Select(columns...).
		From(schedulerSchedulesTable).
		Where(sq.Eq{string(schedule.FieldID): id.Bytes()}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. scheduleGetFromDB. err: %v", err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. scheduleGetFromDB. err: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	if !rows.Next() {
		return nil, ErrNotFound
	}

	res, err := h.scheduleGetFromRow(rows)
	if err != nil {
		return nil, fmt.Errorf("could not scan the row. scheduleGetFromDB. err: %v", err)
	}

	return res, nil
}

// ScheduleGet returns schedule.
func (h *handler) ScheduleGet(ctx context.Context, id uuid.UUID) (*schedule.Schedule, error) {
	res, err := h.scheduleGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.scheduleGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	// set to the cache
	_ = h.scheduleSetToCache(ctx, res)

	return res, nil
}

// ScheduleList returns schedules based on filters.
func (h *handler) ScheduleList(ctx context.Context, size uint64, token string, filters map[schedule.Field]any) ([]*schedule.Schedule, error) {
	// get column names from db tags
	columns := commondatabasehandler.GetDBFields(&schedule.Schedule{})

	builder := sq.Select(columns...).
		From(schedulerSchedulesTable).
		Where(sq.Lt{string(schedule.FieldTMCreate): token}).
		OrderBy(string(schedule.FieldTMCreate) + " desc").
		Limit(size)

	// apply filters
	builder, err := commondatabasehandler.ApplyFields(builder, filters)
	if err != nil {
		return nil, fmt.Errorf("could not apply filters. ScheduleList. err: %v", err)
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. ScheduleList. err: %v", err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. ScheduleList. err: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	res := []*schedule.Schedule{}
	for rows.Next() {
		s, err := h.scheduleGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not scan the row. ScheduleList. err: %v", err)
		}

		res = append(res, s)
	}

	return res, nil
}

// ScheduleUpdate updates a schedule with the given fields.
func (h *handler) ScheduleUpdate(ctx context.Context, id uuid.UUID, fields map[schedule.Field]any) error {
	// add update timestamp
	fields[schedule.FieldTMUpdate] = h.utilHandler.TimeNow()

	// prepare fields for update
	data, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("could not prepare fields. ScheduleUpdate. err: %v", err)
	}

	query, args, err := sq.Update(schedulerSchedulesTable).
		SetMap(data).
		Where(sq.Eq{string(schedule.FieldID): id.Bytes()}).
		ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. ScheduleUpdate. err: %v", err)
	}

	_, err = h.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("could not execute. ScheduleUpdate. err: %v", err)
	}

	// update the cache
	_ = h.scheduleUpdateToCache(ctx, id)

	return nil
}

// ScheduleDelete soft-deletes the schedule info.
func (h *handler) ScheduleDelete(ctx context.Context, id uuid.UUID) error {
	ts := h.utilHandler.TimeNow()
	fields := map[schedule.Field]any{
		schedule.FieldTMUpdate: ts,
		schedule.FieldTMDelete: ts,
	}

	// prepare fields for update
	data, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("could not prepare fields. ScheduleDelete. err: %v", err)
	}

	query, args, err := sq.Update(schedulerSchedulesTable).
		SetMap(data).
		Where(sq.Eq{string(schedule.FieldID): id.Bytes()}).
		ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. ScheduleDelete. err: %v", err)
	}

	_, err = h.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("could not execute. ScheduleDelete. err: %v", err)
	}

	// delete from the cache
	_ = h.cache.ScheduleDelete(ctx, id)

	return nil
}

// ScheduleGetByCustomerIDName returns the active (tm_delete IS NULL) schedule
// with the given customer id and name.
func (h *handler) ScheduleGetByCustomerIDName(ctx context.Context, customerID uuid.UUID, name string) (*schedule.Schedule, error) {
	// get column names from db tags
	columns := commondatabasehandler.GetDBFields(&schedule.Schedule{})

	query, args, err := sq.Select(columns...).
		From(schedulerSchedulesTable).
		Where(sq.Eq{string(schedule.FieldCustomerID): customerID.Bytes()}).
		Where(sq.Eq{string(schedule.FieldName): name}).
		Where(sq.Eq{string(schedule.FieldTMDelete): nil}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. ScheduleGetByCustomerIDName. err: %v", err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. ScheduleGetByCustomerIDName. err: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	if !rows.Next() {
		return nil, ErrNotFound
	}

	res, err := h.scheduleGetFromRow(rows)
	if err != nil {
		return nil, fmt.Errorf("could not scan the row. ScheduleGetByCustomerIDName. err: %v", err)
	}

	return res, nil
}

// ScheduleListDue returns enabled, active schedules whose tm_next_run has
// arrived (tm_next_run IS NOT NULL AND tm_next_run <= now). Design §5.2.
func (h *handler) ScheduleListDue(ctx context.Context, now time.Time, limit uint64) ([]*schedule.Schedule, error) {
	// get column names from db tags
	columns := commondatabasehandler.GetDBFields(&schedule.Schedule{})

	query, args, err := sq.Select(columns...).
		From(schedulerSchedulesTable).
		Where(sq.Eq{string(schedule.FieldEnabled): true}).
		Where(sq.Eq{string(schedule.FieldTMDelete): nil}).
		Where(sq.NotEq{string(schedule.FieldTMNextRun): nil}).
		Where(sq.LtOrEq{string(schedule.FieldTMNextRun): now}).
		OrderBy(string(schedule.FieldTMNextRun) + " asc").
		Limit(limit).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. ScheduleListDue. err: %v", err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. ScheduleListDue. err: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	res := []*schedule.Schedule{}
	for rows.Next() {
		s, err := h.scheduleGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not scan the row. ScheduleListDue. err: %v", err)
		}

		res = append(res, s)
	}

	return res, nil
}

// ScheduleListUninitialized returns enabled, active schedules whose
// tm_next_run is NULL (fresh create or cron just changed). Design §5.2.
func (h *handler) ScheduleListUninitialized(ctx context.Context, limit uint64) ([]*schedule.Schedule, error) {
	// get column names from db tags
	columns := commondatabasehandler.GetDBFields(&schedule.Schedule{})

	query, args, err := sq.Select(columns...).
		From(schedulerSchedulesTable).
		Where(sq.Eq{string(schedule.FieldEnabled): true}).
		Where(sq.Eq{string(schedule.FieldTMDelete): nil}).
		Where(sq.Eq{string(schedule.FieldTMNextRun): nil}).
		Limit(limit).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. ScheduleListUninitialized. err: %v", err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. ScheduleListUninitialized. err: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	res := []*schedule.Schedule{}
	for rows.Next() {
		s, err := h.scheduleGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not scan the row. ScheduleListUninitialized. err: %v", err)
		}

		res = append(res, s)
	}

	return res, nil
}

// ScheduleInitNextRun race-safely initializes tm_next_run for a schedule whose
// tm_next_run is NULL (CAS: expected value NULL). It deliberately does NOT
// touch tm_last_run — a schedule that has never fired never claims it has
// (design §5.2). Returns true when this caller won the initialization.
func (h *handler) ScheduleInitNextRun(ctx context.Context, id uuid.UUID, next time.Time) (bool, error) {
	fields := map[schedule.Field]any{
		schedule.FieldTMNextRun: next,
		schedule.FieldTMUpdate:  h.utilHandler.TimeNow(),
	}

	data, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return false, fmt.Errorf("could not prepare fields. ScheduleInitNextRun. err: %v", err)
	}

	query, args, err := sq.Update(schedulerSchedulesTable).
		SetMap(data).
		Where(sq.Eq{string(schedule.FieldID): id.Bytes()}).
		Where(sq.Eq{string(schedule.FieldTMNextRun): nil}).
		Where(sq.Eq{string(schedule.FieldEnabled): true}).
		Where(sq.Eq{string(schedule.FieldTMDelete): nil}).
		ToSql()
	if err != nil {
		return false, fmt.Errorf("could not build query. ScheduleInitNextRun. err: %v", err)
	}

	res, err := h.db.ExecContext(ctx, query, args...)
	if err != nil {
		return false, fmt.Errorf("could not execute. ScheduleInitNextRun. err: %v", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("could not get affected rows. ScheduleInitNextRun. err: %v", err)
	}

	if affected != 1 {
		return false, nil
	}

	// update the cache
	_ = h.scheduleUpdateToCache(ctx, id)

	return true, nil
}
