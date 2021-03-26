package dbhandler

import (
	"context"
	"database/sql"
	"fmt"

	uuid "github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/recording"
)

const (
	// select query for recording get
	recordingSelect = `
	select
		id,
		user_id,
		type,
		reference_id,
		status,
		format,
		filename,
		webhook_uri,

		asterisk_id,
		channel_id,

		tm_start,
		tm_end,

		tm_create,
		tm_update,
		tm_delete

	from
		recordings
	`
)

// recordingGetFromRow gets the record from the row.
func (h *handler) recordingGetFromRow(row *sql.Rows) (*recording.Recording, error) {
	res := &recording.Recording{}
	if err := row.Scan(
		&res.ID,
		&res.UserID,
		&res.Type,
		&res.ReferenceID,
		&res.Status,
		&res.Format,
		&res.Filename,
		&res.WebhookURI,

		&res.AsteriskID,
		&res.ChannelID,

		&res.TMStart,
		&res.TMEnd,

		&res.TMCreate,
		&res.TMUpdate,
		&res.TMDelete,
	); err != nil {
		return nil, fmt.Errorf("could not scan the row. recordingGetFromRow. err: %v", err)
	}

	return res, nil
}

// RecordingGetFromCache returns record from the cache.
func (h *handler) RecordingGetFromCache(ctx context.Context, id uuid.UUID) (*recording.Recording, error) {

	// get from cache
	res, err := h.cache.RecordingGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// RecordingGetFromDB returns record from the DB.
func (h *handler) RecordingGetFromDB(ctx context.Context, id uuid.UUID) (*recording.Recording, error) {

	// prepare
	q := fmt.Sprintf("%s where id = ?", recordingSelect)

	row, err := h.db.Query(q, id.Bytes())
	if err != nil {
		return nil, fmt.Errorf("could not query. RecordingGetFromDB. err: %v", err)
	}
	defer row.Close()

	if row.Next() == false {
		return nil, ErrNotFound
	}

	res, err := h.recordingGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not get data. RecordingGetFromDB, err: %v", err)
	}

	return res, nil
}

// RecordingUpdateToCache gets the record from the DB and update the cache.
func (h *handler) RecordingUpdateToCache(ctx context.Context, id uuid.UUID) error {

	res, err := h.RecordingGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.RecordingSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// RecordingSetToCache sets the given record to the cache
func (h *handler) RecordingSetToCache(ctx context.Context, r *recording.Recording) error {
	if err := h.cache.RecordingSet(ctx, r); err != nil {
		return err
	}

	return nil
}

// RecordingCreate creates new record.
func (h *handler) RecordingCreate(ctx context.Context, c *recording.Recording) error {
	q := `insert into recordings(
		id,
		user_id,
		type,
		reference_id,
		status,
		format,
		filename,
		webhook_uri,

		asterisk_id,
		channel_id,

		tm_start,
		tm_end,

		tm_create,
		tm_update,
		tm_delete

	) values(
		?, ?, ?, ?, ?, ?, ?, ?,
		?, ?,
		?, ?,
		?, ?, ?
	)`

	_, err := h.db.Exec(q,
		c.ID.Bytes(),
		c.UserID,
		c.Type,
		c.ReferenceID.Bytes(),
		c.Status,
		c.Format,
		c.Filename,
		c.WebhookURI,

		c.AsteriskID,
		c.ChannelID,

		c.TMStart,
		c.TMEnd,

		getCurTime(),
		c.TMUpdate,
		c.TMDelete,
	)
	if err != nil {
		return fmt.Errorf("could not execute. RecordingCreate. err: %v", err)
	}

	// update the cache
	h.RecordingUpdateToCache(ctx, c.ID)

	return nil
}

// RecordingGet returns record.
func (h *handler) RecordingGet(ctx context.Context, id uuid.UUID) (*recording.Recording, error) {

	res, err := h.RecordingGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.RecordingGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	// set to the cache
	h.RecordingSetToCache(ctx, res)

	return res, nil
}

// RecordingGetByFilename gets the recording by the filename.
func (h *handler) RecordingGetByFilename(ctx context.Context, filename string) (*recording.Recording, error) {

	// prepare
	q := fmt.Sprintf("%s where filename = ?", recordingSelect)

	row, err := h.db.Query(q, filename)
	if err != nil {
		return nil, fmt.Errorf("could not query. RecordingGetByFilename. err: %v", err)
	}
	defer row.Close()

	if row.Next() == false {
		return nil, ErrNotFound
	}

	res, err := h.recordingGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not get data. RecordingGetByFilename, err: %v", err)
	}

	return res, nil
}

// RecordingGets returns a list of records.
func (h *handler) RecordingGets(ctx context.Context, userID uint64, size uint64, token string) ([]*recording.Recording, error) {

	// prepare
	q := fmt.Sprintf("%s where user_id = ? and tm_create < ? order by tm_create desc limit ?", recordingSelect)

	rows, err := h.db.Query(q, userID, token, size)
	if err != nil {
		return nil, fmt.Errorf("could not query. RecordingGets. err: %v", err)
	}
	defer rows.Close()

	res := []*recording.Recording{}
	for rows.Next() {
		u, err := h.recordingGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not get data. RecordingGets, err: %v", err)
		}

		res = append(res, u)
	}

	return res, nil
}

// RecordingSetStatus sets the record's status
func (h *handler) RecordingSetStatus(ctx context.Context, id uuid.UUID, status recording.Status, timestamp string) error {

	switch status {

	case recording.StatusRecording:
		return h.recordingSetStatusRecording(ctx, id, timestamp)

	case recording.StatusEnd:
		return h.recordingSetStatusEnd(ctx, id, timestamp)

	case recording.StatusInitiating:
		return h.recordingSetStatusInitiating(ctx, id)

	default:
		return fmt.Errorf("could not found correct status handler")
	}
}

// recordingSetStatusInitiating sets the record's status to initiating
func (h *handler) recordingSetStatusInitiating(ctx context.Context, id uuid.UUID) error {

	// prepare
	q := `
	update
		recordings
	set
		status = ?,
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, recording.StatusInitiating, getCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. recordingSetStatusRecording. err: %v", err)
	}

	// update the cache
	h.RecordingUpdateToCache(ctx, id)

	return nil
}

// recordingSetStatusRecording sets the record's status recording
func (h *handler) recordingSetStatusRecording(ctx context.Context, id uuid.UUID, timestamp string) error {

	// prepare
	q := `
	update
		recordings
	set
		status = ?,
		tm_start = ?,
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, recording.StatusRecording, timestamp, getCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. recordingSetStatusRecording. err: %v", err)
	}

	// update the cache
	h.RecordingUpdateToCache(ctx, id)

	return nil
}

// recordingSetStatusEnd sets the record's status to end
func (h *handler) recordingSetStatusEnd(ctx context.Context, id uuid.UUID, timestamp string) error {

	// prepare
	q := `
	update
		recordings
	set
		status = ?,
		tm_end = ?,
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, recording.StatusEnd, timestamp, getCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. recordingSetStatusEnd. err: %v", err)
	}

	// update the cache
	h.RecordingUpdateToCache(ctx, id)

	return nil
}
