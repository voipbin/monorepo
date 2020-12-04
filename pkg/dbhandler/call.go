package dbhandler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	uuid "github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/callhandler/models/action"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/callhandler/models/call"
)

const (
	// select query for call get
	callSelect = `
	select
		id,
		user_id,
		asterisk_id,
		channel_id,
		flow_id,
		conference_id,
		type,

		master_call_id,
		chained_call_ids,
		recording_id,
		recording_ids,

		source,
		destination,

		status,
		data,
		action,
		direction,
		hangup_by,
		hangup_reason,


		coalesce(tm_create, '') as tm_create,
		coalesce(tm_update, '') as tm_update,

		coalesce(tm_progressing, '') as tm_progressing,
		coalesce(tm_ringing, '') as tm_ringing,
		coalesce(tm_hangup, '') as tm_hangup

	from
		calls
	`
)

// callGetFromRow gets the call from the row.
func (h *handler) callGetFromRow(row *sql.Rows) (*call.Call, error) {
	var chainedCallIDs string
	var recordingIDs string
	var data string
	var source string
	var destination string
	var action string
	res := &call.Call{}
	if err := row.Scan(
		&res.ID,
		&res.UserID,
		&res.AsteriskID,
		&res.ChannelID,
		&res.FlowID,
		&res.ConfID,
		&res.Type,

		&res.MasterCallID,
		&chainedCallIDs,
		&res.RecordingID,
		&recordingIDs,

		&source,
		&destination,

		&res.Status,
		&data,
		&action,
		&res.Direction,
		&res.HangupBy,
		&res.HangupReason,

		&res.TMCreate,
		&res.TMUpdate,

		&res.TMProgressing,
		&res.TMRinging,
		&res.TMHangup,
	); err != nil {
		return nil, fmt.Errorf("could not scan the row. callGetFromRow. err: %v", err)
	}

	if err := json.Unmarshal([]byte(chainedCallIDs), &res.ChainedCallIDs); err != nil {
		return nil, fmt.Errorf("could not unmarshal the chained_call_ids. callGetFromRow. err: %v", err)
	}
	if res.ChainedCallIDs == nil {
		res.ChainedCallIDs = []uuid.UUID{}
	}

	if err := json.Unmarshal([]byte(recordingIDs), &res.RecordingIDs); err != nil {
		return nil, fmt.Errorf("could not unmarshal the record_files. callGetFromRow. err: %v", err)
	}
	if res.RecordingIDs == nil {
		res.RecordingIDs = []string{}
	}

	if err := json.Unmarshal([]byte(data), &res.Data); err != nil {
		return nil, fmt.Errorf("could not unmarshal the data. callGetFromRow. err: %v", err)
	}
	if err := json.Unmarshal([]byte(action), &res.Action); err != nil {
		return nil, fmt.Errorf("could not unmarshal the action. callGetFromRow. err: %v", err)
	}
	if err := json.Unmarshal([]byte(source), &res.Source); err != nil {
		return nil, fmt.Errorf("could not unmarshal the source. callGetFromRow. err: %v", err)
	}
	if err := json.Unmarshal([]byte(destination), &res.Destination); err != nil {
		return nil, fmt.Errorf("could not unmarshal the destination. callGetFromRow. err: %v", err)
	}

	return res, nil
}

// CallCreate creates new call record.
func (h *handler) CallCreate(ctx context.Context, c *call.Call) error {
	q := `insert into calls(
		id,
		user_id,
		asterisk_id,
		channel_id,
		flow_id,
		conference_id,
		type,

		master_call_id,
		chained_call_ids,
		recording_id,
		recording_ids,

		source,
		source_target,
		destination,
		destination_target,

		status,
		data,
		action,
		direction,
		hangup_by,
		hangup_reason,

		tm_create
	) values(
		?, ?, ?, ?, ?, ?, ?,
		?, ?, ?, ?,
		?, ?, ?, ?,
		?, ?, ?, ?, ?,?,
		?
		)`

	if c.ChainedCallIDs == nil {
		c.ChainedCallIDs = []uuid.UUID{}
	}
	tmpChainedCallIDs, err := json.Marshal(c.ChainedCallIDs)
	if err != nil {
		return fmt.Errorf("could not marshal calls. CallCreate. err: %v", err)
	}

	if c.RecordingIDs == nil {
		c.RecordingIDs = []string{}
	}
	tmpRecordingIDs, err := json.Marshal(c.RecordingIDs)
	if err != nil {
		return fmt.Errorf("could not marshal the recording_ids. CallCreate. err: %v", err)
	}

	tmpSource, err := json.Marshal(c.Source)
	if err != nil {
		return fmt.Errorf("could not marshal source. CallCreate. err: %v", err)
	}

	tmpDestination, err := json.Marshal(c.Destination)
	if err != nil {
		return fmt.Errorf("could not marshal destination. CallCreate. err: %v", err)
	}

	tmpData, err := json.Marshal(c.Data)
	if err != nil {
		return fmt.Errorf("could not marshal data. CallCreate. err: %v", err)
	}

	tmpAction, err := json.Marshal(c.Action)
	if err != nil {
		return fmt.Errorf("could not marshal action. CallCreate. err: %v", err)
	}

	_, err = h.db.Exec(q,
		c.ID.Bytes(),
		c.UserID,
		c.AsteriskID,
		c.ChannelID,
		c.FlowID.Bytes(),
		c.ConfID.Bytes(),
		c.Type,

		c.MasterCallID.Bytes(),
		tmpChainedCallIDs,
		c.RecordingID,
		tmpRecordingIDs,

		tmpSource,
		c.Source.Target,
		tmpDestination,
		c.Destination.Target,

		c.Status,
		tmpData,
		tmpAction,
		c.Direction,
		c.HangupBy,
		c.HangupReason,

		c.TMCreate,
	)
	if err != nil {
		return fmt.Errorf("could not execute. CallCreate. err: %v", err)
	}

	// update the cache
	h.CallUpdateToCache(ctx, c.ID)

	return nil
}

// CallGet returns call.
func (h *handler) CallGet(ctx context.Context, id uuid.UUID) (*call.Call, error) {

	res, err := h.CallGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.CallGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	// set to the cache
	h.CallSetToCache(ctx, res)

	return res, nil
}

// CallGet returns call.
func (h *handler) CallGetByChannelID(ctx context.Context, channelID string) (*call.Call, error) {

	// prepare
	q := fmt.Sprintf("%s where channel_id = ?", callSelect)

	row, err := h.db.Query(q, channelID)
	if err != nil {
		return nil, fmt.Errorf("could not query. CallGetByChannelID. err: %v", err)
	}
	defer row.Close()

	if row.Next() == false {
		return nil, ErrNotFound
	}

	res, err := h.callGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not get call. CallGetByChannelID, err: %v", err)
	}

	return res, nil
}

// callSetStatusRinging sets the call status to ringing
func (h *handler) callSetStatusRinging(ctx context.Context, id uuid.UUID, tmStatus string) error {
	// prepare
	q := `
	update
		calls
	set
		status = ?,
		tm_update = ?,
		tm_ringing = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, call.StatusRinging, getCurTime(), tmStatus, id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. CallSetStatusRinging. err: %v", err)
	}

	// update the cache
	h.CallUpdateToCache(ctx, id)

	return nil
}

// callSetStatusProgressing sets the call status to progressing
func (h *handler) callSetStatusProgressing(ctx context.Context, id uuid.UUID, tmStatus string) error {
	// prepare
	q := `
	update
		calls
	set
		status = ?,
		tm_update = ?,
		tm_progressing = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, call.StatusProgressing, getCurTime(), tmStatus, id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. callSetStatusProgressing. err: %v", err)
	}

	// update the cache
	h.CallUpdateToCache(ctx, id)

	return nil
}

// callSetStatus sets the call status without update the timestamp for status
func (h *handler) callSetStatus(ctx context.Context, id uuid.UUID, status call.Status) error {
	// prepare
	q := `
	update
		calls
	set
		status = ?,
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, status, getCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. callSetStatus. err: %v", err)
	}

	// update the cache
	h.CallUpdateToCache(ctx, id)

	return nil
}

// CallSetStatus sets the call status
func (h *handler) CallSetStatus(ctx context.Context, id uuid.UUID, status call.Status, tmStatus string) error {

	// get call info
	c, err := h.CallGet(ctx, id)
	if err != nil {
		return err
	}

	// validate changable status
	if call.IsUpdatableStatus(c.Status, status) == false {
		return fmt.Errorf("The given status is not updatable. old: %s, new: %s", c.Status, status)
	}

	switch status {
	case call.StatusRinging:
		return h.callSetStatusRinging(ctx, id, tmStatus)
	case call.StatusProgressing:
		return h.callSetStatusProgressing(ctx, id, tmStatus)
	default:
		return h.callSetStatus(ctx, id, status)
	}
}

// CallSetAsteriskID sets the call aserisk_id
func (h *handler) CallSetAsteriskID(ctx context.Context, id uuid.UUID, asteriskID string, tmUpdate string) error {

	// prepare
	q := `
	update
		calls
	set
		asterisk_id = ?,
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, asteriskID, tmUpdate, id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. CallSetAsteriskID. err: %v", err)
	}

	// update the cache
	h.CallUpdateToCache(ctx, id)

	return nil
}

// CallSetStatus sets the call status
func (h *handler) CallSetHangup(ctx context.Context, id uuid.UUID, reason call.HangupReason, hangupBy call.HangupBy, tmUpdate string) error {

	// prepare
	q := `
	update
		calls
	set
		status = ?,
		hangup_by = ?,
		hangup_reason = ?,
		tm_update = ?,
		tm_hangup = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, string(call.StatusHangup), hangupBy, reason, tmUpdate, tmUpdate, id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. CallSetHangup. err: %v", err)
	}

	// update the cache
	h.CallUpdateToCache(ctx, id)

	return nil
}

// CallSetFlowID sets the call status
func (h *handler) CallSetFlowID(ctx context.Context, id, flowID uuid.UUID) error {

	// prepare
	q := `
	update
		calls
	set
		flow_id = ?,
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, flowID.Bytes(), getCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. CallSetFlowID. err: %v", err)
	}

	// update the cache
	h.CallUpdateToCache(ctx, id)

	return nil
}

// CallSetFlowID sets the call status
func (h *handler) CallSetConferenceID(ctx context.Context, id, conferenceID uuid.UUID) error {

	// prepare
	q := `
	update
		calls
	set
		conference_id = ?,
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, conferenceID.Bytes(), getCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. CallSetConferenceID. err: %v", err)
	}

	// update the cache
	h.CallUpdateToCache(ctx, id)

	return nil
}

// CallSetAction sets the call status
func (h *handler) CallSetAction(ctx context.Context, id uuid.UUID, action *action.Action) error {

	// prepare
	q := `
	update
		calls
	set
		action = ?,
		tm_update = ?
	where
		id = ?
	`

	tmpAction, err := json.Marshal(action)
	if err != nil {
		return err
	}

	_, err = h.db.Exec(q, tmpAction, getCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. CallSetAction. err: %v", err)
	}

	// update the cache
	h.CallUpdateToCache(ctx, id)

	return nil
}

// CallGetFromCache returns call from the cache.
func (h *handler) CallGetFromCache(ctx context.Context, id uuid.UUID) (*call.Call, error) {

	// get from cache
	res, err := h.cache.CallGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// CallGetFromDB returns call from the DB.
func (h *handler) CallGetFromDB(ctx context.Context, id uuid.UUID) (*call.Call, error) {

	// prepare
	q := fmt.Sprintf("%s where id = ?", callSelect)

	row, err := h.db.Query(q, id.Bytes())
	if err != nil {
		return nil, fmt.Errorf("could not query. CallGet. err: %v", err)
	}
	defer row.Close()

	if row.Next() == false {
		return nil, ErrNotFound
	}

	res, err := h.callGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not get call. CallGet, err: %v", err)
	}

	return res, nil
}

// CallUpdateToCache gets the call from the DB and update the cache.
func (h *handler) CallUpdateToCache(ctx context.Context, id uuid.UUID) error {

	res, err := h.CallGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.CallSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// CallSetToCache sets the given call to the cache
func (h *handler) CallSetToCache(ctx context.Context, call *call.Call) error {
	if err := h.cache.CallSet(ctx, call); err != nil {
		return err
	}

	return nil
}

// CallAddChainedCallID adds the call id to the given call's chained_call_ids.
func (h *handler) CallAddChainedCallID(ctx context.Context, id, chainedCallID uuid.UUID) error {
	// prepare
	q := `
	update calls set
		chained_call_ids = json_array_append(
			chained_call_ids,
			'$',
			?
		),
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, chainedCallID.String(), getCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. CallAddChainedCallID. err: %v", err)
	}

	// update the cache
	h.CallUpdateToCache(ctx, id)

	return nil
}

// CallRemoveChainedCallID removes the call id from the given call's chained_call_ids.
func (h *handler) CallRemoveChainedCallID(ctx context.Context, id, chainedCallID uuid.UUID) error {
	// prepare
	q := `
	update calls set
		chained_call_ids = json_remove(
			chained_call_ids, replace(
				json_search(
					chained_call_ids,
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

	_, err := h.db.Exec(q, chainedCallID.String(), getCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. CallRemoveChainedCallID. err: %v", err)
	}

	// update the cache
	h.CallUpdateToCache(ctx, id)

	return nil
}

// CallSetMasterCallID sets the call's master_call_id
func (h *handler) CallSetMasterCallID(ctx context.Context, id uuid.UUID, callID uuid.UUID) error {

	// prepare
	q := `
	update
		calls
	set
		master_call_id = ?,
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, callID.Bytes(), getCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. CallSetMasterCallID. err: %v", err)
	}

	// update the cache
	h.CallUpdateToCache(ctx, id)

	return nil
}

// CallSetRecordID sets the given recordID to recording_id.
func (h *handler) CallSetRecordID(ctx context.Context, id uuid.UUID, recordID string) error {
	// prepare
	q := `
	update calls set
		recording_id = ?,
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, recordID, getCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. CallSetRecordID. err: %v", err)
	}

	// update the cache
	h.CallUpdateToCache(context.Background(), id)

	return nil
}

// CallAddRecordIDs adds the given recording_id into the recording_ids.
func (h *handler) CallAddRecordIDs(ctx context.Context, id uuid.UUID, recordID string) error {
	// prepare
	q := `
	update calls set
		recording_ids = json_array_append(
			recording_ids,
			'$',
			?
		),
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, recordID, getCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. CallAddRecordIDs. err: %v", err)
	}

	// update the cache
	h.CallUpdateToCache(ctx, id)

	return nil
}

func (h *handler) CallTXStart(id uuid.UUID) (*sql.Tx, *call.Call, error) {

	tx, err := h.db.Begin()
	if err != nil {
		return nil, nil, fmt.Errorf("could not get transaction. CallTXStart. err: %v", err)
	}

	// prepare
	q := fmt.Sprintf("%s where id = ? for update", callSelect)

	row, err := tx.Query(q, id.Bytes())
	if err != nil {
		tx.Rollback()
		return nil, nil, fmt.Errorf("could not query. CallTXStart. err: %v", err)
	}
	defer row.Close()

	if row.Next() == false {
		tx.Rollback()
		return nil, nil, ErrNotFound
	}

	res, err := h.callGetFromRow(row)
	if err != nil {
		tx.Rollback()
		return nil, nil, fmt.Errorf("could not get call. CallTXStart, err: %v", err)
	}

	return tx, res, nil
}

func (h *handler) CallTXFinish(tx *sql.Tx, commit bool) {
	if commit == true {
		tx.Commit()
	} else {
		tx.Rollback()
	}
}

// CallTXAddChainedCallID adds the call id to the given call's chained_call_ids in a transaction mode.
func (h *handler) CallTXAddChainedCallID(tx *sql.Tx, id, chainedCallID uuid.UUID) error {
	// prepare
	q := `
	update calls set
		chained_call_ids = json_array_append(
			chained_call_ids,
			'$',
			?
		),
		tm_update = ?
	where
		id = ?
	`

	_, err := tx.Exec(q, chainedCallID.String(), getCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. CallAddChainedCallID. err: %v", err)
	}

	// update the cache
	h.CallUpdateToCache(context.Background(), id)

	return nil
}

// CallTXRemoveChainedCallID removes the call id from the given call's chained_call_ids in a transaction mode.
func (h *handler) CallTXRemoveChainedCallID(tx *sql.Tx, id, chainedCallID uuid.UUID) error {
	// prepare
	q := `
	update calls set
		chained_call_ids = json_remove(
			chained_call_ids, replace(
				json_search(
					chained_call_ids,
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

	_, err := tx.Exec(q, chainedCallID.String(), getCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. CallRemoveChainedCallID. err: %v", err)
	}

	// update the cache
	h.CallUpdateToCache(context.Background(), id)

	return nil
}
