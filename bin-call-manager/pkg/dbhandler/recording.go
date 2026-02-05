package dbhandler

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Masterminds/squirrel"
	uuid "github.com/gofrs/uuid"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-call-manager/models/recording"
)

var (
	recordingTable = "call_recordings"
)

// recordingGetFromRow gets the record from the row.
func (h *handler) recordingGetFromRow(row *sql.Rows) (*recording.Recording, error) {
	res := &recording.Recording{}

	if err := commondatabasehandler.ScanRow(row, res); err != nil {
		return nil, fmt.Errorf("could not scan the row. recordingGetFromRow. err: %v", err)
	}

	// Initialize nil slices to empty
	if res.Filenames == nil {
		res.Filenames = []string{}
	}
	if res.ChannelIDs == nil {
		res.ChannelIDs = []string{}
	}

	return res, nil
}

// RecordingCreate creates new record.
func (h *handler) RecordingCreate(ctx context.Context, c *recording.Recording) error {
	now := h.utilHandler.TimeNow()

	// Set timestamps
	c.TMCreate = now
	c.TMUpdate = nil
	c.TMDelete = nil

	// Initialize nil slices
	if c.Filenames == nil {
		c.Filenames = []string{}
	}
	if c.ChannelIDs == nil {
		c.ChannelIDs = []string{}
	}

	// Use PrepareFields to get field map
	fields, err := commondatabasehandler.PrepareFields(c)
	if err != nil {
		return fmt.Errorf("could not prepare fields. RecordingCreate. err: %v", err)
	}

	// Use SetMap instead of Columns/Values
	sb := squirrel.
		Insert(recordingTable).
		SetMap(fields).
		PlaceholderFormat(squirrel.Question)

	query, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. RecordingCreate. err: %v", err)
	}

	if _, err := h.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("could not execute query. RecordingCreate. err: %v", err)
	}

	// update the cache
	_ = h.recordingUpdateToCache(ctx, c.ID)

	return nil
}

// recordingGetFromCache returns record from the cache.
func (h *handler) recordingGetFromCache(ctx context.Context, id uuid.UUID) (*recording.Recording, error) {
	res, err := h.cache.RecordingGet(ctx, id)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// recordingGetFromDB returns record from the DB.
func (h *handler) recordingGetFromDB(ctx context.Context, id uuid.UUID) (*recording.Recording, error) {
	fields := commondatabasehandler.GetDBFields(&recording.Recording{})
	query, args, err := squirrel.
		Select(fields...).
		From(recordingTable).
		Where(squirrel.Eq{string(recording.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build sql. recordingGetFromDB. err: %v", err)
	}

	row, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. recordingGetFromDB. err: %v", err)
	}
	defer func() {
		_ = row.Close()
	}()

	if !row.Next() {
		if err := row.Err(); err != nil {
			return nil, fmt.Errorf("row iteration error. recordingGetFromDB. err: %v", err)
		}
		return nil, ErrNotFound
	}

	res, err := h.recordingGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not get data. recordingGetFromDB, err: %v", err)
	}

	return res, nil
}

// recordingUpdateToCache gets the record from the DB and update the cache.
func (h *handler) recordingUpdateToCache(ctx context.Context, id uuid.UUID) error {
	res, err := h.recordingGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.recordingSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// recordingSetToCache sets the given record to the cache
func (h *handler) recordingSetToCache(ctx context.Context, r *recording.Recording) error {
	if err := h.cache.RecordingSet(ctx, r); err != nil {
		return err
	}
	return nil
}

// RecordingGet returns record.
func (h *handler) RecordingGet(ctx context.Context, id uuid.UUID) (*recording.Recording, error) {
	res, err := h.recordingGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.recordingGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	// set to the cache
	_ = h.recordingSetToCache(ctx, res)

	return res, nil
}

// RecordingGetByRecordingName gets the recording by the recording_name.
func (h *handler) RecordingGetByRecordingName(ctx context.Context, recordingName string) (*recording.Recording, error) {
	fields := commondatabasehandler.GetDBFields(&recording.Recording{})
	query, args, err := squirrel.
		Select(fields...).
		From(recordingTable).
		Where(squirrel.Eq{string(recording.FieldRecordingName): recordingName}).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build sql. RecordingGetByRecordingName. err: %v", err)
	}

	row, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. RecordingGetByRecordingName. err: %v", err)
	}
	defer func() {
		_ = row.Close()
	}()

	if !row.Next() {
		if err := row.Err(); err != nil {
			return nil, fmt.Errorf("row iteration error. RecordingGetByRecordingName. err: %v", err)
		}
		return nil, ErrNotFound
	}

	res, err := h.recordingGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not get data. RecordingGetByRecordingName, err: %v", err)
	}

	return res, nil
}

// RecordingGets returns a list of records.
func (h *handler) RecordingList(ctx context.Context, size uint64, token string, filters map[recording.Field]any) ([]*recording.Recording, error) {
	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	dbFields := commondatabasehandler.GetDBFields(&recording.Recording{})
	sb := squirrel.
		Select(dbFields...).
		From(recordingTable).
		Where(squirrel.Lt{string(recording.FieldTMCreate): token}).
		OrderBy(string(recording.FieldTMCreate) + " DESC").
		Limit(size).
		PlaceholderFormat(squirrel.Question)

	sb, err := commondatabasehandler.ApplyFields(sb, filters)
	if err != nil {
		return nil, fmt.Errorf("could not apply filters. RecordingGets. err: %v", err)
	}

	query, args, err := sb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. RecordingGets. err: %v", err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. RecordingGets. err: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	res := []*recording.Recording{}
	for rows.Next() {
		u, err := h.recordingGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not get data. RecordingGets, err: %v", err)
		}
		res = append(res, u)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error. RecordingGets. err: %v", err)
	}

	return res, nil
}

// RecordingUpdate updates recording fields using a generic typed field map
func (h *handler) RecordingUpdate(ctx context.Context, id uuid.UUID, fields map[recording.Field]any) error {
	if len(fields) == 0 {
		return nil
	}

	// Only set TMUpdate if it's not already provided
	if _, ok := fields[recording.FieldTMUpdate]; !ok {
		fields[recording.FieldTMUpdate] = h.utilHandler.TimeNow()
	}

	tmpFields, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("RecordingUpdate: prepare fields failed: %w", err)
	}

	q := squirrel.Update(recordingTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{string(recording.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question)

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("RecordingUpdate: build SQL failed: %w", err)
	}

	if _, err := h.db.ExecContext(ctx, sqlStr, args...); err != nil {
		return fmt.Errorf("RecordingUpdate: exec failed: %w", err)
	}

	_ = h.recordingUpdateToCache(ctx, id)
	return nil
}

// RecordingSetStatus sets the record's status
func (h *handler) RecordingSetStatus(ctx context.Context, id uuid.UUID, status recording.Status) error {
	switch status {
	case recording.StatusRecording:
		return h.recordingSetStatusRecording(ctx, id)
	case recording.StatusEnded:
		return h.recordingSetStatusEnd(ctx, id)
	case recording.StatusStopping:
		return h.recordingSetStatusStopping(ctx, id)
	default:
		return fmt.Errorf("could not found correct status handler")
	}
}

// recordingSetStatusRecording sets the record's status recording
func (h *handler) recordingSetStatusRecording(ctx context.Context, id uuid.UUID) error {
	ts := h.utilHandler.TimeNow()
	return h.RecordingUpdate(ctx, id, map[recording.Field]any{
		recording.FieldStatus:   recording.StatusRecording,
		recording.FieldTMStart:  ts,
		recording.FieldTMUpdate: ts,
	})
}

// recordingSetStatusEnd sets the record's status to end
func (h *handler) recordingSetStatusEnd(ctx context.Context, id uuid.UUID) error {
	ts := h.utilHandler.TimeNow()
	return h.RecordingUpdate(ctx, id, map[recording.Field]any{
		recording.FieldStatus:   recording.StatusEnded,
		recording.FieldTMEnd:    ts,
		recording.FieldTMUpdate: ts,
	})
}

// recordingSetStatusStopping sets the record's status to stopping
func (h *handler) recordingSetStatusStopping(ctx context.Context, id uuid.UUID) error {
	return h.RecordingUpdate(ctx, id, map[recording.Field]any{
		recording.FieldStatus: recording.StatusStopping,
	})
}

// RecordingDelete deletes the recording
func (h *handler) RecordingDelete(ctx context.Context, id uuid.UUID) error {
	ts := h.utilHandler.TimeNow()
	return h.RecordingUpdate(ctx, id, map[recording.Field]any{
		recording.FieldTMUpdate: ts,
		recording.FieldTMDelete: ts,
	})
}
