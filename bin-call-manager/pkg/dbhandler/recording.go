package dbhandler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"

	uuid "github.com/gofrs/uuid"

	"monorepo/bin-call-manager/models/recording"
)

const (
	// select query for recording get
	recordingSelect = `
	select
		id,
		customer_id,
		owner_type,
		owner_id,

		reference_type,
		reference_id,
		status,
		format,

		recording_name,
		filenames,

		asterisk_id,
		channel_ids,

		tm_start,
		tm_end,

		tm_create,
		tm_update,
		tm_delete

	from
		call_recordings
	`
)

// recordingGetFromRow gets the record from the row.
func (h *handler) recordingGetFromRow(row *sql.Rows) (*recording.Recording, error) {
	var filenames sql.NullString
	var channelIDs sql.NullString

	res := &recording.Recording{}
	if err := row.Scan(
		&res.ID,
		&res.CustomerID,
		&res.OwnerType,
		&res.OwnerID,

		&res.ReferenceType,
		&res.ReferenceID,
		&res.Status,
		&res.Format,

		&res.RecordingName,
		&filenames,

		&res.AsteriskID,
		&channelIDs,

		&res.TMStart,
		&res.TMEnd,

		&res.TMCreate,
		&res.TMUpdate,
		&res.TMDelete,
	); err != nil {
		return nil, fmt.Errorf("could not scan the row. recordingGetFromRow. err: %v", err)
	}

	// Filenames
	if filenames.Valid {
		if err := json.Unmarshal([]byte(filenames.String), &res.Filenames); err != nil {
			return nil, fmt.Errorf("could not unmarshal the recording_ids. callGetFromRow. err: %v", err)
		}
	}
	if res.Filenames == nil {
		res.Filenames = []string{}
	}

	// ChannelIDs
	if channelIDs.Valid {
		if err := json.Unmarshal([]byte(channelIDs.String), &res.ChannelIDs); err != nil {
			return nil, fmt.Errorf("could not unmarshal the recording_ids. callGetFromRow. err: %v", err)
		}
	}
	if res.ChannelIDs == nil {
		res.ChannelIDs = []string{}
	}

	return res, nil
}

// RecordingCreate creates new record.
func (h *handler) RecordingCreate(ctx context.Context, c *recording.Recording) error {
	q := `insert into call_recordings(
		id,
		customer_id,
		owner_type,
		owner_id,

		reference_type,
		reference_id,
		status,
		format,

		recording_name,
        filenames,

		asterisk_id,
		channel_ids,

		tm_start,
		tm_end,

		tm_create,
		tm_update,
		tm_delete

	) values(
		?, ?, ?, ?,
		?, ?, ?, ?,
		?, ?,
		?, ?,
		?, ?,
		?, ?, ?
	)`

	tmpFilenames, err := json.Marshal(c.Filenames)
	if err != nil {
		return fmt.Errorf("could not marshal Filenames. RecordingCreate. err: %v", err)
	}

	tmpChannelIDs, err := json.Marshal(c.ChannelIDs)
	if err != nil {
		return fmt.Errorf("could not marshal ChannelIDs. RecordingCreate. err: %v", err)
	}

	_, err = h.db.Exec(q,
		c.ID.Bytes(),
		c.CustomerID.Bytes(),
		c.OwnerType,
		c.OwnerID.Bytes(),

		c.ReferenceType,
		c.ReferenceID.Bytes(),
		c.Status,
		c.Format,

		c.RecordingName,
		tmpFilenames,

		c.AsteriskID,
		tmpChannelIDs,

		c.TMStart,
		c.TMEnd,

		h.utilHandler.TimeGetCurTime(),
		DefaultTimeStamp,
		DefaultTimeStamp,
	)
	if err != nil {
		return fmt.Errorf("could not execute. RecordingCreate. err: %v", err)
	}

	// update the cache
	_ = h.recordingUpdateToCache(ctx, c.ID)

	return nil
}

// recordingGetFromCache returns record from the cache.
func (h *handler) recordingGetFromCache(ctx context.Context, id uuid.UUID) (*recording.Recording, error) {

	// get from cache
	res, err := h.cache.RecordingGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// recordingGetFromDB returns record from the DB.
func (h *handler) recordingGetFromDB(ctx context.Context, id uuid.UUID) (*recording.Recording, error) {

	// prepare
	q := fmt.Sprintf("%s where id = ?", recordingSelect)

	row, err := h.db.Query(q, id.Bytes())
	if err != nil {
		return nil, fmt.Errorf("could not query. RecordingGetFromDB. err: %v", err)
	}
	defer row.Close()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.recordingGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not get data. RecordingGetFromDB, err: %v", err)
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

	// prepare
	q := fmt.Sprintf("%s where recording_name = ?", recordingSelect)

	row, err := h.db.Query(q, recordingName)
	if err != nil {
		return nil, fmt.Errorf("could not query. RecordingGetByRecordingName. err: %v", err)
	}
	defer row.Close()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.recordingGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not get data. RecordingGetByRecordingName, err: %v", err)
	}

	return res, nil
}

// RecordingGets returns a list of records.
func (h *handler) RecordingGets(ctx context.Context, size uint64, token string, filters map[string]string) ([]*recording.Recording, error) {

	// prepare
	q := fmt.Sprintf(`%s
	where
		tm_create < ?
	`, recordingSelect)

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	values := []interface{}{
		token,
	}

	for k, v := range filters {
		switch k {
		case "customer_id", "reference_id":
			q = fmt.Sprintf("%s and %s = ?", q, k)
			tmp := uuid.FromStringOrNil(v)
			values = append(values, tmp.Bytes())

		case "deleted":
			if v == "false" {
				q = fmt.Sprintf("%s and tm_delete >= ?", q)
				values = append(values, DefaultTimeStamp)
			}

		default:
			q = fmt.Sprintf("%s and %s = ?", q, k)
			values = append(values, v)
		}
	}

	q = fmt.Sprintf("%s order by tm_create desc limit ?", q)
	values = append(values, strconv.FormatUint(size, 10))

	rows, err := h.db.Query(q, values...)
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

	// prepare
	q := `
	update
		call_recordings
	set
		status = ?,
		tm_start = ?,
		tm_update = ?
	where
		id = ?
	`

	ts := h.utilHandler.TimeGetCurTime()
	_, err := h.db.Exec(q, recording.StatusRecording, ts, ts, id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. recordingSetStatusRecording. err: %v", err)
	}

	// update the cache
	_ = h.recordingUpdateToCache(ctx, id)

	return nil
}

// recordingSetStatusEnd sets the record's status to end
func (h *handler) recordingSetStatusEnd(ctx context.Context, id uuid.UUID) error {

	// prepare
	q := `
	update
		call_recordings
	set
		status = ?,
		tm_end = ?,
		tm_update = ?
	where
		id = ?
	`

	ts := h.utilHandler.TimeGetCurTime()
	_, err := h.db.Exec(q, recording.StatusEnded, ts, ts, id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. recordingSetStatusEnd. err: %v", err)
	}

	// update the cache
	_ = h.recordingUpdateToCache(ctx, id)

	return nil
}

// recordingSetStatusStopping sets the record's status to stopping
func (h *handler) recordingSetStatusStopping(ctx context.Context, id uuid.UUID) error {

	// prepare
	q := `
	update
		call_recordings
	set
		status = ?,
		tm_update = ?
	where
		id = ?
	`

	ts := h.utilHandler.TimeGetCurTime()
	_, err := h.db.Exec(q, recording.StatusStopping, ts, id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. recordingSetStatusStopping. err: %v", err)
	}

	// update the cache
	_ = h.recordingUpdateToCache(ctx, id)

	return nil
}

// RecordingDelete deletes the recording
func (h *handler) RecordingDelete(ctx context.Context, id uuid.UUID) error {
	//prepare
	q := `
	update call_recordings set
		tm_update = ?,
		tm_delete = ?
	where
		id = ?
	`

	ts := h.utilHandler.TimeGetCurTime()
	_, err := h.db.Exec(q, ts, ts, id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. RecordingDelete. err: %v", err)
	}

	// update the cache
	_ = h.recordingUpdateToCache(ctx, id)

	return nil
}
