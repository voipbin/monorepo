package dbhandler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"

	uuid "github.com/gofrs/uuid"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"

	"gitlab.com/voipbin/bin-manager/conference-manager.git/models/conference"
)

const (
	conferenceSelect = `
	select
		id,
		customer_id,
		type,
		flow_id,
		confbridge_id,

		status,
		name,
		detail,
		data,
		timeout,

		pre_actions,
		post_actions,

		conferencecall_ids,

		recording_id,
		recording_ids,

		transcribe_id,
		transcribe_ids,

		tm_end,

		tm_create,
		tm_update,
		tm_delete

	from
		conferences
	`
)

// conferenceGetFromRow gets the conference from the row.
func (h *handler) conferenceGetFromRow(row *sql.Rows) (*conference.Conference, error) {

	var preActions string
	var postActions string
	var data string
	var conferencecallIDs sql.NullString
	var recordingIDs sql.NullString
	var transcribeIDs sql.NullString

	res := &conference.Conference{}
	if err := row.Scan(
		&res.ID,
		&res.CustomerID,
		&res.Type,
		&res.FlowID,
		&res.ConfbridgeID,

		&res.Status,
		&res.Name,
		&res.Detail,
		&data,
		&res.Timeout,

		&preActions,
		&postActions,

		&conferencecallIDs,

		&res.RecordingID,
		&recordingIDs,

		&res.TranscribeID,
		&transcribeIDs,

		&res.TMEnd,

		&res.TMCreate,
		&res.TMUpdate,
		&res.TMDelete,
	); err != nil {
		return nil, fmt.Errorf("could not scan the row. conferenceGetFromRow. err: %v", err)
	}

	if err := json.Unmarshal([]byte(preActions), &res.PreActions); err != nil {
		return nil, fmt.Errorf("could not unmarshal the pre-actions. conferenceGetFromRow. err: %v", err)
	}
	if res.PreActions == nil {
		res.PreActions = []fmaction.Action{}
	}

	if err := json.Unmarshal([]byte(postActions), &res.PostActions); err != nil {
		return nil, fmt.Errorf("could not unmarshal the post-actions. conferenceGetFromRow. err: %v", err)
	}
	if res.PostActions == nil {
		res.PostActions = []fmaction.Action{}
	}

	if err := json.Unmarshal([]byte(data), &res.Data); err != nil {
		return nil, fmt.Errorf("could not unmarshal the data. conferenceGetFromRow. err: %v", err)
	}
	if res.Data == nil {
		res.Data = map[string]interface{}{}
	}

	if conferencecallIDs.Valid && conferencecallIDs.String != "" {
		if err := json.Unmarshal([]byte(conferencecallIDs.String), &res.ConferencecallIDs); err != nil {
			return nil, fmt.Errorf("could not unmarshal the queucall_ids. conferenceGetFromRow. err: %v", err)
		}
		if res.ConferencecallIDs == nil {
			res.ConferencecallIDs = []uuid.UUID{}
		}
	} else {
		res.ConferencecallIDs = []uuid.UUID{}
	}

	if recordingIDs.Valid {
		if errMarshal := json.Unmarshal([]byte(recordingIDs.String), &res.RecordingIDs); errMarshal != nil {
			return nil, fmt.Errorf("could not unmarshal the recording_ids. conferenceGetFromRow. err: %v", errMarshal)
		}
	}
	if res.RecordingIDs == nil {
		res.RecordingIDs = []uuid.UUID{}
	}

	if transcribeIDs.Valid {
		if errMarshal := json.Unmarshal([]byte(transcribeIDs.String), &res.TranscribeIDs); errMarshal != nil {
			return nil, fmt.Errorf("could not unmarshal the transcribe_ids. conferenceGetFromRow. err: %v", errMarshal)
		}
	}
	if res.TranscribeIDs == nil {
		res.TranscribeIDs = []uuid.UUID{}
	}

	return res, nil
}

// ConferenceCreate creates a new conference record.
func (h *handler) ConferenceCreate(ctx context.Context, cf *conference.Conference) error {
	q := `insert into conferences(
		id,
		customer_id,
		type,
		flow_id,
		confbridge_id,

		status,
		name,
		detail,
		data,
		timeout,

		pre_actions,
		post_actions,

		conferencecall_ids,

		recording_id,
		recording_ids,

		transcribe_id,
		transcribe_ids,

		tm_end,

		tm_create,
		tm_update,
		tm_delete
	) values(
		?, ?, ?, ?, ?,
		?, ?, ?, ?, ?,
		?, ?,
		?,
		?, ?,
		?, ?,
		?,
		?, ?, ?
		)
	`

	preActions, err := json.Marshal(cf.PreActions)
	if err != nil {
		return fmt.Errorf("could not marshal the preActions. ConferenceCreate. err: %v", err)
	}

	postActions, err := json.Marshal(cf.PostActions)
	if err != nil {
		return fmt.Errorf("could not marshal the postActions. ConferenceCreate. err: %v", err)
	}

	data, err := json.Marshal(cf.Data)
	if err != nil {
		return fmt.Errorf("could not marshal data. ConferenceCreate. err: %v", err)
	}

	conferencecallIDs, err := json.Marshal(cf.ConferencecallIDs)
	if err != nil {
		return fmt.Errorf("could not marshal calls. ConferenceCreate. err: %v", err)
	}

	recordingIDs, err := json.Marshal(cf.RecordingIDs)
	if err != nil {
		return fmt.Errorf("could not marshal recording_ids. ConferenceCreate. err: %v", err)
	}

	transcribeIDs, err := json.Marshal(cf.TranscribeIDs)
	if err != nil {
		return fmt.Errorf("could not marshal transcribe_ids. ConferenceCreate. err: %v", err)
	}

	_, err = h.db.Exec(q,
		cf.ID.Bytes(),
		cf.CustomerID.Bytes(),
		cf.Type,
		cf.FlowID.Bytes(),
		cf.ConfbridgeID.Bytes(),

		cf.Status,
		cf.Name,
		cf.Detail,
		data,
		cf.Timeout,

		preActions,
		postActions,

		conferencecallIDs,

		cf.RecordingID.Bytes(),
		recordingIDs,

		cf.TranscribeID.Bytes(),
		transcribeIDs,

		DefaultTimeStamp,

		h.utilHandler.TimeGetCurTime(),
		DefaultTimeStamp,
		DefaultTimeStamp,
	)
	if err != nil {
		return fmt.Errorf("could not execute. ConferenceCreate. err: %v", err)
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

	// prepare
	q := fmt.Sprintf("%s where id = ?", conferenceSelect)

	row, err := h.db.Query(q, id.Bytes())
	if err != nil {
		return nil, fmt.Errorf("could not query. conferenceGetFromDB. err: %v", err)
	}
	defer row.Close()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.conferenceGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not get call. conferenceGetFromDB, err: %v", err)
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
func (h *handler) ConferenceSetToCache(ctx context.Context, conference *conference.Conference) error {
	if err := h.cache.ConferenceSet(ctx, conference); err != nil {
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
func (h *handler) ConferenceGets(ctx context.Context, size uint64, token string, filters map[string]string) ([]*conference.Conference, error) {

	// prepare
	q := fmt.Sprintf(`
		%s
	where
		tm_create < ?
	`, conferenceSelect)

	values := []interface{}{
		token,
	}

	for k, v := range filters {
		switch k {
		case "customer_id", "confbridge_id", "flow_id", "recording_id", "transcribe_id":
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
		return nil, fmt.Errorf("could not query. ConferenceGets. err: %v", err)
	}
	defer rows.Close()

	res := []*conference.Conference{}
	for rows.Next() {
		u, err := h.conferenceGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not get data. ConferenceGets, err: %v", err)
		}

		res = append(res, u)
	}

	return res, nil
}

// ConferenceGetByConfbridgeID returns conference of the given confbridgeID
func (h *handler) ConferenceGetByConfbridgeID(ctx context.Context, confbridgeID uuid.UUID) (*conference.Conference, error) {

	// prepare
	q := fmt.Sprintf("%s where confbridge_id = ?", conferenceSelect)

	row, err := h.db.Query(q, confbridgeID.Bytes())
	if err != nil {
		return nil, fmt.Errorf("could not query. ConferenceGet. err: %v", err)
	}
	defer row.Close()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.conferenceGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not get call. ConferenceGetByConfbridgeID, err: %v", err)
	}

	return res, nil
}

// ConferenceAddConferencecallID adds the call id to the conference.
func (h *handler) ConferenceAddConferencecallID(ctx context.Context, id, conferencecallID uuid.UUID) error {
	// prepare
	q := `
	update conferences set
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
	update conferences set
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
	//prepare
	q := `
	update conferences set
		tm_update = ?,
		tm_delete = ?
	where
		id = ?
	`

	ts := h.utilHandler.TimeGetCurTime()
	_, err := h.db.Exec(q, ts, ts, id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. ConferenceDelete. err: %v", err)
	}

	// update the cache
	_ = h.conferenceUpdateToCache(ctx, id)

	return nil
}

// ConferenceSet sets the status
func (h *handler) ConferenceSet(ctx context.Context, id uuid.UUID, name, detail string, timeout int, preActions, postActions []fmaction.Action) error {
	//prepare
	q := `
	update conferences set
		name = ?,
		detail = ?,
		timeout = ?,
		pre_actions = ?,
		post_actions = ?,
		tm_update = ?
	where
		id = ?
	`

	tmpPreActions, err := json.Marshal(preActions)
	if err != nil {
		return fmt.Errorf("could not marshal the preActions. ConferenceSet. err: %v", err)
	}

	tmpPostActions, err := json.Marshal(postActions)
	if err != nil {
		return fmt.Errorf("could not marshal the postActions. ConferenceSet. err: %v", err)
	}

	_, err = h.db.Exec(q, name, detail, timeout, tmpPreActions, tmpPostActions, h.utilHandler.TimeGetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. ConferenceSet. err: %v", err)
	}

	// update the cache
	_ = h.conferenceUpdateToCache(ctx, id)

	return nil
}

// ConferenceSetStatus sets the status
func (h *handler) ConferenceSetStatus(ctx context.Context, id uuid.UUID, status conference.Status) error {
	//prepare
	q := `
	update conferences set
		status = ?,
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, status, h.utilHandler.TimeGetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. ConferenceSetStatus. err: %v", err)
	}

	// update the cache
	_ = h.conferenceUpdateToCache(ctx, id)

	return nil
}

// ConferenceSetData sets the data
func (h *handler) ConferenceSetData(ctx context.Context, id uuid.UUID, data map[string]interface{}) error {
	//prepare
	q := `
	update conferences set
		data = ?,
		tm_update = ?
	where
		id = ?
	`

	tmpData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("dbhandler: Could not marshal. ConferenceSetData. err: %v", err)
	}

	_, err = h.db.Exec(q, tmpData, h.utilHandler.TimeGetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. ConferenceSetData. err: %v", err)
	}

	// update the cache
	_ = h.conferenceUpdateToCache(ctx, id)

	return nil
}

// ConferenceEnd ends the conference
func (h *handler) ConferenceEnd(ctx context.Context, id uuid.UUID) error {
	//prepare
	q := `
	update conferences set
		status = ?,
		tm_end = ?,
		tm_update = ?
	where
		id = ?
	`

	ts := h.utilHandler.TimeGetCurTime()
	_, err := h.db.Exec(q, conference.StatusTerminated, ts, ts, id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. ConferenceEnd. err: %v", err)
	}

	// update the cache
	_ = h.conferenceUpdateToCache(ctx, id)

	return nil
}

// ConferenceSetRecordingID sets the conference's recording_id.
func (h *handler) ConferenceSetRecordingID(ctx context.Context, id uuid.UUID, recordingID uuid.UUID) error {
	// prepare
	q := `
	update conferences set
		recording_id = ?,
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, recordingID.Bytes(), h.utilHandler.TimeGetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. ConferenceSetRecordingID. err: %v", err)
	}

	// update the cache
	_ = h.conferenceUpdateToCache(ctx, id)

	return nil
}

// ConferenceAddRecordingIDs adds the recording id to the conference's recording_ids.
func (h *handler) ConferenceAddRecordingIDs(ctx context.Context, id uuid.UUID, recordingID uuid.UUID) error {
	// prepare
	q := `
	update conferences set
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

// ConferenceSetTranscribeID sets the conference's transcribe_id.
func (h *handler) ConferenceSetTranscribeID(ctx context.Context, id uuid.UUID, transcribeID uuid.UUID) error {
	// prepare
	q := `
	update conferences set
		transcribe_id = ?,
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, transcribeID.Bytes(), h.utilHandler.TimeGetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. ConferenceSetTranscribeID. err: %v", err)
	}

	// update the cache
	_ = h.conferenceUpdateToCache(ctx, id)

	return nil
}

// ConferenceAddTranscribeIDs adds the transcribe id to the conference's transcribe_ids.
func (h *handler) ConferenceAddTranscribeIDs(ctx context.Context, id uuid.UUID, transcribeID uuid.UUID) error {
	// prepare
	q := `
	update conferences set
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
