package dbhandler

import (
	"context"
	"database/sql"
	"fmt"

	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/callhandler/models/record"
)

const (
	// select query for record get
	recordSelect = `
	select
		id,
		user_id,
		type,
		reference_id,
		status,
		format,

		asterisk_id,
		channel_id,

		coalesce(tm_start, '') as tm_start,
		coalesce(tm_end, '') as tm_end,

		coalesce(tm_create, '') as tm_create,
		coalesce(tm_update, '') as tm_update,
		coalesce(tm_delete, '') as tm_delete

	from
		records
	`
)

// recordGetFromRow gets the record from the row.
func (h *handler) recordGetFromRow(row *sql.Rows) (*record.Record, error) {
	res := &record.Record{}
	if err := row.Scan(
		&res.ID,
		&res.UserID,
		&res.Type,
		&res.ReferenceID,
		&res.Status,
		&res.Format,

		&res.AsteriskID,
		&res.ChannelID,

		&res.TMStart,
		&res.TMEnd,

		&res.TMCreate,
		&res.TMUpdate,
		&res.TMDelete,
	); err != nil {
		return nil, fmt.Errorf("could not scan the row. recordGetFromRow. err: %v", err)
	}

	return res, nil
}

// RecordGetFromCache returns record from the cache.
func (h *handler) RecordGetFromCache(ctx context.Context, id string) (*record.Record, error) {

	// get from cache
	res, err := h.cache.RecordGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// RecordGetFromDB returns record from the DB.
func (h *handler) RecordGetFromDB(ctx context.Context, id string) (*record.Record, error) {

	// prepare
	q := fmt.Sprintf("%s where id = ?", recordSelect)

	row, err := h.db.Query(q, id)
	if err != nil {
		return nil, fmt.Errorf("could not query. RecordGetFromDB. err: %v", err)
	}
	defer row.Close()

	if row.Next() == false {
		return nil, ErrNotFound
	}

	res, err := h.recordGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not get data. RecordGetFromDB, err: %v", err)
	}

	return res, nil
}

// RecordUpdateToCache gets the record from the DB and update the cache.
func (h *handler) RecordUpdateToCache(ctx context.Context, id string) error {

	res, err := h.RecordGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.RecordSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// RecordSetToCache sets the given record to the cache
func (h *handler) RecordSetToCache(ctx context.Context, r *record.Record) error {
	if err := h.cache.RecordSet(ctx, r); err != nil {
		return err
	}

	return nil
}

// RecordCreate creates new record.
func (h *handler) RecordCreate(ctx context.Context, c *record.Record) error {
	q := `insert into records(
		id,
		user_id,
		type,
		reference_id,
		status,
		format,

		asterisk_id,
		channel_id,

		tm_create

	) values(
		?, ?, ?, ?, ?, ?,
		?, ?,
		?
	)`

	_, err := h.db.Exec(q,
		c.ID,
		c.UserID,
		c.Type,
		c.ReferenceID.Bytes(),
		c.Status,
		c.Format,

		c.AsteriskID,
		c.ChannelID,

		getCurTime(),
	)
	if err != nil {
		return fmt.Errorf("could not execute. RecordCreate. err: %v", err)
	}

	// update the cache
	h.RecordUpdateToCache(ctx, c.ID)

	return nil
}

// RecordGet returns record.
func (h *handler) RecordGet(ctx context.Context, id string) (*record.Record, error) {

	res, err := h.RecordGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.RecordGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	// set to the cache
	h.RecordSetToCache(ctx, res)

	return res, nil
}

// RecordSetStatus sets the record's status
func (h *handler) RecordSetStatus(ctx context.Context, id string, status record.Status, timestamp string) error {

	switch status {

	case record.StatusRecording:
		return h.recordSetStatusRecording(ctx, id, timestamp)

	case record.StatusEnd:
		return h.recordSetStatusEnd(ctx, id, timestamp)

	case record.StatusInitiating:
		return h.recordSetStatusInitiating(ctx, id)

	default:
		return fmt.Errorf("could not found correct status handler")
	}
}

// recordSetStatusInitiating sets the record's status to initiating
func (h *handler) recordSetStatusInitiating(ctx context.Context, id string) error {

	// prepare
	q := `
	update
		records
	set
		status = ?,
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, record.StatusInitiating, getCurTime(), id)
	if err != nil {
		return fmt.Errorf("could not execute. recordSetStatusRecording. err: %v", err)
	}

	// update the cache
	h.RecordUpdateToCache(ctx, id)

	return nil
}

// recordSetStatusRecording sets the record's status recording
func (h *handler) recordSetStatusRecording(ctx context.Context, id string, timestamp string) error {

	// prepare
	q := `
	update
		records
	set
		status = ?,
		tm_start = ?,
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, record.StatusRecording, timestamp, getCurTime(), id)
	if err != nil {
		return fmt.Errorf("could not execute. recordSetStatusRecording. err: %v", err)
	}

	// update the cache
	h.RecordUpdateToCache(ctx, id)

	return nil
}

// recordSetStatusEnd sets the record's status to end
func (h *handler) recordSetStatusEnd(ctx context.Context, id string, timestamp string) error {

	// prepare
	q := `
	update
		records
	set
		status = ?,
		tm_end = ?,
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, record.StatusRecording, timestamp, getCurTime(), id)
	if err != nil {
		return fmt.Errorf("could not execute. recordSetStatusEnd. err: %v", err)
	}

	// update the cache
	h.RecordUpdateToCache(ctx, id)

	return nil
}
