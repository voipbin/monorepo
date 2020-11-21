package dbhandler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	uuid "github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/conferencehandler/models/conference"
)

const (
	conferenceSelect = `
	select
		id,
		user_id,
		type,
		bridge_id,

		status,
		name,
		detail,
		data,

		call_ids,

		record_id,
		record_ids,

		coalesce(tm_create, '') as tm_create,
		coalesce(tm_update, '') as tm_update,
		coalesce(tm_delete, '') as tm_delete

	from
		conferences
	`
)

// conferenceGetFromRow gets the call from the row.
func (h *handler) conferenceGetFromRow(row *sql.Rows) (*conference.Conference, error) {

	var data string
	var calls string
	var RecordIDs string

	res := &conference.Conference{}
	if err := row.Scan(
		&res.ID,
		&res.UserID,
		&res.Type,
		&res.BridgeID,

		&res.Status,
		&res.Name,
		&res.Detail,
		&data,

		&calls,

		&res.RecordID,
		&RecordIDs,

		&res.TMCreate,
		&res.TMUpdate,
		&res.TMDelete,
	); err != nil {
		return nil, fmt.Errorf("could not scan the row. conferenceGetFromRow. err: %v", err)
	}

	if err := json.Unmarshal([]byte(data), &res.Data); err != nil {
		return nil, fmt.Errorf("could not unmarshal the data. conferenceGetFromRow. err: %v", err)
	}

	if err := json.Unmarshal([]byte(calls), &res.CallIDs); err != nil {
		return nil, fmt.Errorf("could not unmarshal the destination. conferenceGetFromRow. err: %v", err)
	}
	if res.CallIDs == nil {
		res.CallIDs = []uuid.UUID{}
	}

	if err := json.Unmarshal([]byte(RecordIDs), &res.RecordIDs); err != nil {
		return nil, fmt.Errorf("could not unmarshal the destination. conferenceGetFromRow. err: %v", err)
	}
	if res.RecordIDs == nil {
		res.RecordIDs = []string{}
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

	if row.Next() == false {
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
		bridge_id,

		status,
		name,
		detail,
		data,

		call_ids,

		record_id,
		record_ids,

		tm_create
	) values(
		?, ?, ?, ?,
		?, ?, ?, ?,
		?, 
		?, ?,
		?
		)
	`

	data, err := json.Marshal(cf.Data)
	if err != nil {
		return fmt.Errorf("could not marshal data. ConferenceCreate. err: %v", err)
	}

	callIDs, err := json.Marshal(cf.CallIDs)
	if err != nil {
		return fmt.Errorf("could not marshal calls. ConferenceCreate. err: %v", err)
	}

	recordIDs, err := json.Marshal(cf.RecordIDs)
	if err != nil {
		return fmt.Errorf("could not marshal record_ids. ConferenceCreate. err: %v", err)
	}

	_, err = h.db.Exec(q,
		cf.ID.Bytes(),
		cf.UserID,
		cf.Type,
		cf.BridgeID,

		cf.Status,
		cf.Name,
		cf.Detail,
		data,

		callIDs,

		cf.RecordID,
		recordIDs,

		getCurTime(),
	)
	if err != nil {
		return fmt.Errorf("could not execute. ConferenceCreate. err: %v", err)
	}

	// update the cache
	h.ConferenceUpdateToCache(ctx, cf.ID)

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
	h.ConferenceSetToCache(ctx, res)

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
	h.ConferenceUpdateToCache(ctx, id)

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
	h.ConferenceUpdateToCache(ctx, id)

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
	h.ConferenceUpdateToCache(ctx, id)

	return nil
}

// ConferenceSetBridgeID sets the bridge id
func (h *handler) ConferenceSetBridgeID(ctx context.Context, id uuid.UUID, bridgeID string) error {
	//prepare
	q := `
	update conferences set
		bridge_id = ?,
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, bridgeID, getCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. ConferenceSetBridgeID. err: %v", err)
	}

	// update the cache
	h.ConferenceUpdateToCache(ctx, id)

	return nil
}

// ConferenceSetData sets the data
func (h *handler) ConferenceSetData(ctx context.Context, id uuid.UUID, data map[string]interface{}) error {
	//prepare
	q := `
	update conference set
		data = ?,
		tm_update = ?
	where
		id = ?
	`

	tmpData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("dbhandler: Could not marshal. ConferenceSetData. err: %v", err)
	}

	_, err = h.db.Exec(q, tmpData, getCurTime(), id)
	if err != nil {
		return fmt.Errorf("could not execute. ConferenceSetData. err: %v", err)
	}

	// update the cache
	h.ConferenceUpdateToCache(ctx, id)

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
	h.ConferenceUpdateToCache(ctx, id)

	return nil
}

// ConferenceSetRecordID sets the conference's record_id.
func (h *handler) ConferenceSetRecordID(ctx context.Context, id uuid.UUID, recordID string) error {
	// prepare
	q := `
	update conferences set
		record_id = ?,
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, recordID, getCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. ConferenceSetRecordID. err: %v", err)
	}

	// update the cache
	h.ConferenceUpdateToCache(ctx, id)

	return nil
}

// ConferenceAddRecordIDs adds the record file to the bridge's record_files.
func (h *handler) ConferenceAddRecordIDs(ctx context.Context, id uuid.UUID, recordID string) error {
	// prepare
	q := `
	update conferences set
		record_ids = json_array_append(
			record_ids,
			'$',
			?
		),
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, recordID, getCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. ConferenceAddRecordIDs. err: %v", err)
	}

	// update the cache
	h.ConferenceUpdateToCache(ctx, id)

	return nil
}
