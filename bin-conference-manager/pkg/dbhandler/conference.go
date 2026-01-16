package dbhandler

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Masterminds/squirrel"
	uuid "github.com/gofrs/uuid"
	"github.com/pkg/errors"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-conference-manager/models/conference"
)

var (
	conferenceTable = "conference_conferences"
)

// conferenceGetFromRow gets the conference from the row.
func (h *handler) conferenceGetFromRow(row *sql.Rows) (*conference.Conference, error) {
	res := &conference.Conference{}

	if err := commondatabasehandler.ScanRow(row, res); err != nil {
		return nil, fmt.Errorf("could not scan the row. conferenceGetFromRow. err: %v", err)
	}

	// Initialize nil slices and maps to empty values
	if res.Data == nil {
		res.Data = map[string]any{}
	}
	if res.ConferencecallIDs == nil {
		res.ConferencecallIDs = []uuid.UUID{}
	}
	if res.RecordingIDs == nil {
		res.RecordingIDs = []uuid.UUID{}
	}
	if res.TranscribeIDs == nil {
		res.TranscribeIDs = []uuid.UUID{}
	}

	return res, nil
}

// ConferenceCreate creates a new conference record.
func (h *handler) ConferenceCreate(ctx context.Context, cf *conference.Conference) error {
	now := h.utilHandler.TimeGetCurTime()

	// Set timestamps
	cf.TMEnd = commondatabasehandler.DefaultTimeStamp
	cf.TMCreate = now
	cf.TMUpdate = commondatabasehandler.DefaultTimeStamp
	cf.TMDelete = commondatabasehandler.DefaultTimeStamp

	// Initialize nil fields
	if cf.Data == nil {
		cf.Data = map[string]any{}
	}
	if cf.ConferencecallIDs == nil {
		cf.ConferencecallIDs = []uuid.UUID{}
	}
	if cf.RecordingIDs == nil {
		cf.RecordingIDs = []uuid.UUID{}
	}
	if cf.TranscribeIDs == nil {
		cf.TranscribeIDs = []uuid.UUID{}
	}

	// Use PrepareFields to get field map
	fields, err := commondatabasehandler.PrepareFields(cf)
	if err != nil {
		return fmt.Errorf("could not prepare fields. ConferenceCreate. err: %v", err)
	}

	// Use SetMap instead of Columns/Values
	sb := squirrel.
		Insert(conferenceTable).
		SetMap(fields).
		PlaceholderFormat(squirrel.Question)

	query, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. ConferenceCreate. err: %v", err)
	}

	if _, err := h.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("could not execute query. ConferenceCreate. err: %v", err)
	}

	// update the cache
	_ = h.conferenceUpdateToCache(ctx, cf.ID)

	return nil
}

// conferenceGetFromCache returns conference from the cache if possible.
func (h *handler) conferenceGetFromCache(ctx context.Context, id uuid.UUID) (*conference.Conference, error) {

	// get from cache
	res, err := h.cache.ConferenceGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// conferenceGetFromDB gets conference.
func (h *handler) conferenceGetFromDB(ctx context.Context, id uuid.UUID) (*conference.Conference, error) {
	fields := commondatabasehandler.GetDBFields(&conference.Conference{})
	query, args, err := squirrel.
		Select(fields...).
		From(conferenceTable).
		Where(squirrel.Eq{string(conference.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build sql. conferenceGetFromDB. err: %v", err)
	}

	row, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. conferenceGetFromDB. err: %v", err)
	}
	defer func() {
		_ = row.Close()
	}()

	if !row.Next() {
		if err := row.Err(); err != nil {
			return nil, fmt.Errorf("row iteration error. conferenceGetFromDB. err: %v", err)
		}
		return nil, ErrNotFound
	}

	res, err := h.conferenceGetFromRow(row)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get data from row. conferenceGetFromDB. id: %s", id)
	}

	return res, nil
}

// conferenceUpdateToCache gets the conference from the DB and update the cache.
func (h *handler) conferenceUpdateToCache(ctx context.Context, id uuid.UUID) error {

	res, err := h.conferenceGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.ConferenceSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// ConferenceSetToCache sets the given conference to the cache
func (h *handler) ConferenceSetToCache(ctx context.Context, conf *conference.Conference) error {
	if err := h.cache.ConferenceSet(ctx, conf); err != nil {
		return err
	}

	return nil
}

// ConferenceGet gets conference.
func (h *handler) ConferenceGet(ctx context.Context, id uuid.UUID) (*conference.Conference, error) {

	res, err := h.conferenceGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.conferenceGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	// set to the cache
	_ = h.ConferenceSetToCache(ctx, res)

	return res, nil
}

// ConferenceGets returns a list of conferences.
func (h *handler) ConferenceList(ctx context.Context, size uint64, token string, filters map[conference.Field]any) ([]*conference.Conference, error) {
	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	fields := commondatabasehandler.GetDBFields(&conference.Conference{})
	sb := squirrel.
		Select(fields...).
		From(conferenceTable).
		Where(squirrel.Lt{string(conference.FieldTMCreate): token}).
		OrderBy(string(conference.FieldTMCreate) + " DESC").
		Limit(size).
		PlaceholderFormat(squirrel.Question)

	sb, err := commondatabasehandler.ApplyFields(sb, filters)
	if err != nil {
		return nil, fmt.Errorf("could not apply filters. ConferenceGets. err: %v", err)
	}

	query, args, err := sb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. ConferenceGets. err: %v", err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. ConferenceGets. err: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	res := []*conference.Conference{}
	for rows.Next() {
		u, err := h.conferenceGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not get data. ConferenceGets, err: %v", err)
		}
		res = append(res, u)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error. ConferenceGets. err: %v", err)
	}

	return res, nil
}

// ConferenceGetByConfbridgeID returns conference of the given confbridgeID
func (h *handler) ConferenceGetByConfbridgeID(ctx context.Context, confbridgeID uuid.UUID) (*conference.Conference, error) {
	fields := commondatabasehandler.GetDBFields(&conference.Conference{})
	query, args, err := squirrel.
		Select(fields...).
		From(conferenceTable).
		Where(squirrel.Eq{string(conference.FieldConfbridgeID): confbridgeID.Bytes()}).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build sql. ConferenceGetByConfbridgeID. err: %v", err)
	}

	row, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. ConferenceGetByConfbridgeID. err: %v", err)
	}
	defer func() {
		_ = row.Close()
	}()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.conferenceGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not get call. ConferenceGetByConfbridgeID, err: %v", err)
	}

	return res, nil
}

// ConferenceUpdate updates the conference with the given fields.
func (h *handler) ConferenceUpdate(ctx context.Context, id uuid.UUID, fields map[conference.Field]any) error {
	if len(fields) == 0 {
		return nil
	}

	fields[conference.FieldTMUpdate] = h.utilHandler.TimeGetCurTime()

	tmpFields, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("ConferenceUpdate: prepare fields failed: %w", err)
	}

	q := squirrel.Update(conferenceTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{string(conference.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question)

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("ConferenceUpdate: build SQL failed: %w", err)
	}

	if _, err := h.db.ExecContext(ctx, sqlStr, args...); err != nil {
		return fmt.Errorf("ConferenceUpdate: exec failed: %w", err)
	}

	// update the cache
	_ = h.conferenceUpdateToCache(ctx, id)
	return nil
}

// ConferenceAddConferencecallID adds the call id to the conference.
func (h *handler) ConferenceAddConferencecallID(ctx context.Context, id, conferencecallID uuid.UUID) error {
	// prepare
	q := `
	update conference_conferences set
		conferencecall_ids = json_array_append(
			coalesce(conferencecall_ids, '[]'),
			'$',
			?
		),
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, conferencecallID.String(), h.utilHandler.TimeGetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. ConferenceAddConferencecallID. err: %v", err)
	}

	// update the cache
	_ = h.conferenceUpdateToCache(ctx, id)

	return nil
}

// ConferenceRemoveConferencecallID removes the call id from the conference.
func (h *handler) ConferenceRemoveConferencecallID(ctx context.Context, id, conferencecallID uuid.UUID) error {
	// prepare
	q := `
	update conference_conferences set
		conferencecall_ids = json_remove(
			conferencecall_ids, replace(
				json_search(
					conferencecall_ids,
					'one',
					?
				),
				'"',
				''
			)
		),
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, conferencecallID.String(), h.utilHandler.TimeGetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. ConferenceRemoveConferencecallID. err: %v", err)
	}

	// update the cache
	_ = h.conferenceUpdateToCache(ctx, id)

	return nil
}

// ConferenceDelete deletes the conference
func (h *handler) ConferenceDelete(ctx context.Context, id uuid.UUID) error {
	ts := h.utilHandler.TimeGetCurTime()

	fields := map[conference.Field]any{
		conference.FieldTMUpdate: ts,
		conference.FieldTMDelete: ts,
	}

	tmpFields, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("ConferenceDelete: prepare fields failed: %w", err)
	}

	sb := squirrel.Update(conferenceTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{string(conference.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question)

	sqlStr, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("ConferenceDelete: build SQL failed: %w", err)
	}

	if _, err := h.db.ExecContext(ctx, sqlStr, args...); err != nil {
		return fmt.Errorf("ConferenceDelete: exec failed: %w", err)
	}

	// update the cache
	_ = h.conferenceUpdateToCache(ctx, id)

	return nil
}

// ConferenceEnd ends the conference
func (h *handler) ConferenceEnd(ctx context.Context, id uuid.UUID) error {
	ts := h.utilHandler.TimeGetCurTime()

	fields := map[conference.Field]any{
		conference.FieldStatus:   conference.StatusTerminated,
		conference.FieldTMEnd:    ts,
		conference.FieldTMUpdate: ts,
	}

	return h.ConferenceUpdate(ctx, id, fields)
}

// ConferenceAddRecordingIDs adds the recording id to the conference's recording_ids.
func (h *handler) ConferenceAddRecordingIDs(ctx context.Context, id uuid.UUID, recordingID uuid.UUID) error {
	// prepare
	q := `
	update conference_conferences set
		recording_ids = json_array_append(
			coalesce(recording_ids, '[]'),
			'$',
			?
		),
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, recordingID.String(), h.utilHandler.TimeGetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. ConferenceAddRecordingIDs. err: %v", err)
	}

	// update the cache
	_ = h.conferenceUpdateToCache(ctx, id)

	return nil
}

// ConferenceAddTranscribeIDs adds the transcribe id to the conference's transcribe_ids.
func (h *handler) ConferenceAddTranscribeIDs(ctx context.Context, id uuid.UUID, transcribeID uuid.UUID) error {
	// prepare
	q := `
	update conference_conferences set
		transcribe_ids = json_array_append(
			coalesce(transcribe_ids, '[]'),
			'$',
			?
		),
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, transcribeID.String(), h.utilHandler.TimeGetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. ConferenceAddTranscribeIDs. err: %v", err)
	}

	// update the cache
	_ = h.conferenceUpdateToCache(ctx, id)

	return nil
}
