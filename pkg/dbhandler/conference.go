package dbhandler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	uuid "github.com/gofrs/uuid"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"

	"gitlab.com/voipbin/bin-manager/conference-manager.git/models/conference"
)

const (
	conferenceSelect = `
	select
		id,
		user_id,
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

		call_ids,

		recording_id,
		recording_ids,

		webhook_uri,

		tm_create,
		tm_update,
		tm_delete

	from
		conferences
	`
)

// conferenceGetFromRow gets the call from the row.
func (h *handler) conferenceGetFromRow(row *sql.Rows) (*conference.Conference, error) {

	var preActions string
	var postActions string
	var data string
	var callIDs string
	var RecordingIDs string

	res := &conference.Conference{}
	if err := row.Scan(
		&res.ID,
		&res.UserID,
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

		&callIDs,

		&res.RecordingID,
		&RecordingIDs,

		&res.WebhookURI,

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
		res.PreActions = []action.Action{}
	}

	if err := json.Unmarshal([]byte(postActions), &res.PostActions); err != nil {
		return nil, fmt.Errorf("could not unmarshal the post-actions. conferenceGetFromRow. err: %v", err)
	}
	if res.PostActions == nil {
		res.PostActions = []action.Action{}
	}

	if err := json.Unmarshal([]byte(data), &res.Data); err != nil {
		return nil, fmt.Errorf("could not unmarshal the data. conferenceGetFromRow. err: %v", err)
	}

	if err := json.Unmarshal([]byte(callIDs), &res.CallIDs); err != nil {
		return nil, fmt.Errorf("could not unmarshal the destination. conferenceGetFromRow. err: %v", err)
	}
	if res.CallIDs == nil {
		res.CallIDs = []uuid.UUID{}
	}

	if err := json.Unmarshal([]byte(RecordingIDs), &res.RecordingIDs); err != nil {
		return nil, fmt.Errorf("could not unmarshal the destination. conferenceGetFromRow. err: %v", err)
	}
	if res.RecordingIDs == nil {
		res.RecordingIDs = []uuid.UUID{}
	}

	return res, nil
}

// ConferenceGetFromCache returns conference from the cache if possible.
func (h *handler) ConferenceGetFromCache(ctx context.Context, id uuid.UUID) (*conference.Conference, error) {

	// get from cache
	res, err := h.cache.ConferenceGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// ConferenceGet gets conference.
func (h *handler) ConferenceGetFromDB(ctx context.Context, id uuid.UUID) (*conference.Conference, error) {

	// prepare
	q := fmt.Sprintf("%s where id = ?", conferenceSelect)

	row, err := h.db.Query(q, id.Bytes())
	if err != nil {
		return nil, fmt.Errorf("could not query. ConferenceGet. err: %v", err)
	}
	defer row.Close()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.conferenceGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not get call. ConferenceGet, err: %v", err)
	}

	return res, nil
}

// ConferenceUpdateToCache gets the conference from the DB and update the cache.
func (h *handler) ConferenceUpdateToCache(ctx context.Context, id uuid.UUID) error {

	res, err := h.ConferenceGetFromDB(ctx, id)
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

// ConferenceCreate creates a new conference record.
func (h *handler) ConferenceCreate(ctx context.Context, cf *conference.Conference) error {
	q := `insert into conferences(
		id,
		user_id,
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

		call_ids,

		recording_id,
		recording_ids,

		webhook_uri,

		tm_create,
		tm_update,
		tm_delete
	) values(
		?, ?, ?, ?, ?,
		?, ?, ?, ?, ?,
		?, ?,
		?,
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

	callIDs, err := json.Marshal(cf.CallIDs)
	if err != nil {
		return fmt.Errorf("could not marshal calls. ConferenceCreate. err: %v", err)
	}

	recordingIDs, err := json.Marshal(cf.RecordingIDs)
	if err != nil {
		return fmt.Errorf("could not marshal recording_ids. ConferenceCreate. err: %v", err)
	}

	_, err = h.db.Exec(q,
		cf.ID.Bytes(),
		cf.UserID,
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

		callIDs,

		cf.RecordingID.Bytes(),
		recordingIDs,

		cf.WebhookURI,

		cf.TMCreate,
		cf.TMUpdate,
		cf.TMDelete,
	)
	if err != nil {
		return fmt.Errorf("could not execute. ConferenceCreate. err: %v", err)
	}

	// update the cache
	_ = h.ConferenceUpdateToCache(ctx, cf.ID)

	return nil
}

// ConferenceGet gets conference.
func (h *handler) ConferenceGet(ctx context.Context, id uuid.UUID) (*conference.Conference, error) {

	res, err := h.ConferenceGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.ConferenceGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	// set to the cache
	_ = h.ConferenceSetToCache(ctx, res)

	return res, nil
}

// ConferenceGets returns a list of calls.
func (h *handler) ConferenceGets(ctx context.Context, userID uint64, size uint64, token string) ([]*conference.Conference, error) {

	// prepare
	q := fmt.Sprintf(`
		%s
		where
			tm_delete >= ?
			and user_id = ?
			and tm_create < ?
		order by
			tm_create desc
		limit ?
	`, conferenceSelect)

	rows, err := h.db.Query(q, defaultTimeStamp, userID, token, size)
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

// ConferenceGetsWithType returns a list of calls.
func (h *handler) ConferenceGetsWithType(ctx context.Context, userID uint64, confType conference.Type, size uint64, token string) ([]*conference.Conference, error) {

	// prepare
	q := fmt.Sprintf(`
		%s
		where
			tm_delete >= ?
			and user_id = ?
			and type = ?
			and tm_create < ?
		order by
			tm_create desc
		limit ?
	`, conferenceSelect)

	rows, err := h.db.Query(q, defaultTimeStamp, userID, confType, token, size)
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

// ConferenceAddCallID adds the call id to the conference.
func (h *handler) ConferenceAddCallID(ctx context.Context, id, callID uuid.UUID) error {
	// prepare
	q := `
	update conferences set
		call_ids = json_array_append(
			call_ids,
			'$',
			?
		),
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, callID.String(), getCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. ConferenceAddCallID. err: %v", err)
	}

	// update the cache
	_ = h.ConferenceUpdateToCache(ctx, id)

	return nil
}

// ConferenceRemoveCallID removes the call id from the conference.
func (h *handler) ConferenceRemoveCallID(ctx context.Context, id, callID uuid.UUID) error {
	// prepare
	q := `
	update conferences set
		call_ids = json_remove(
			call_ids, replace(
				json_search(
					call_ids,
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

	_, err := h.db.Exec(q, callID.String(), getCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. ConferenceRemoveCallID. err: %v", err)
	}

	// update the cache
	_ = h.ConferenceUpdateToCache(ctx, id)

	return nil
}

// ConferenceSet sets the status
func (h *handler) ConferenceSet(ctx context.Context, id uuid.UUID, name, detail string, timeout int, webhookURI string, preActions, postActions []action.Action) error {
	//prepare
	q := `
	update conferences set
		name = ?,
		detail = ?,
		timeout = ?,
		webhook_uri = ?,
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

	_, err = h.db.Exec(q, name, detail, timeout, webhookURI, tmpPreActions, tmpPostActions, getCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. ConferenceSet. err: %v", err)
	}

	// update the cache
	_ = h.ConferenceUpdateToCache(ctx, id)

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

	_, err := h.db.Exec(q, status, getCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. ConferenceSetStatus. err: %v", err)
	}

	// update the cache
	_ = h.ConferenceUpdateToCache(ctx, id)

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

	_, err = h.db.Exec(q, tmpData, getCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. ConferenceSetData. err: %v", err)
	}

	// update the cache
	_ = h.ConferenceUpdateToCache(ctx, id)

	return nil
}

// ConferenceEnd ends the conference
func (h *handler) ConferenceEnd(ctx context.Context, id uuid.UUID) error {
	//prepare
	q := `
	update conferences set
		status = ?,
		tm_delete = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, conference.StatusTerminated, getCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. ConferenceEnd. err: %v", err)
	}

	// update the cache
	_ = h.ConferenceUpdateToCache(ctx, id)

	return nil
}

// ConferenceSetRecordID sets the conference's recording_id.
func (h *handler) ConferenceSetRecordID(ctx context.Context, id uuid.UUID, recordID uuid.UUID) error {
	// prepare
	q := `
	update conferences set
		recording_id = ?,
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, recordID.Bytes(), getCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. ConferenceSetRecordID. err: %v", err)
	}

	// update the cache
	_ = h.ConferenceUpdateToCache(ctx, id)

	return nil
}

// ConferenceAddRecordIDs adds the record file to the bridge's record_files.
func (h *handler) ConferenceAddRecordIDs(ctx context.Context, id uuid.UUID, recordID uuid.UUID) error {
	// prepare
	q := `
	update conferences set
		recording_ids = json_array_append(
			recording_ids,
			'$',
			?
		),
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, recordID.Bytes(), getCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. ConferenceAddRecordIDs. err: %v", err)
	}

	// update the cache
	_ = h.ConferenceUpdateToCache(ctx, id)

	return nil
}
