package dbhandler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"

	"monorepo/bin-common-handler/models/address"

	fmaction "monorepo/bin-flow-manager/models/action"

	rmroute "monorepo/bin-route-manager/models/route"

	uuid "github.com/gofrs/uuid"

	"monorepo/bin-call-manager/models/call"
)

const (
	// select query for call get
	callSelect = `
	select
		id,
		customer_id,
		owner_type,
		owner_id,

		channel_id,
		bridge_id,
		flow_id,
		active_flow_id,
		confbridge_id,
		type,

		master_call_id,
		chained_call_ids,
		recording_id,
		recording_ids,
		external_media_id,
		groupcall_id,

		source,
		destination,

		status,
		data,
		action,
		action_next_hold,
		direction,
		mute_direction,

		hangup_by,
		hangup_reason,

		dialroute_id,
		dialroutes,

		tm_create,
		tm_update,
		tm_delete,

		tm_progressing,
		tm_ringing,
		tm_hangup

	from
		call_calls
	`
)

// callGetFromRow gets the call from the row.
func (h *handler) callGetFromRow(row *sql.Rows) (*call.Call, error) {
	var chainedCallIDs sql.NullString
	var recordingIDs sql.NullString
	var data sql.NullString
	var source sql.NullString
	var destination sql.NullString
	var action sql.NullString
	var dialroutes sql.NullString
	var tmDelete sql.NullString

	res := &call.Call{}
	if err := row.Scan(
		&res.ID,
		&res.CustomerID,
		&res.OwnerType,
		&res.OwnerID,

		&res.ChannelID,
		&res.BridgeID,
		&res.FlowID,
		&res.ActiveFlowID,
		&res.ConfbridgeID,
		&res.Type,

		&res.MasterCallID,
		&chainedCallIDs,
		&res.RecordingID,
		&recordingIDs,
		&res.ExternalMediaID,
		&res.GroupcallID,

		&source,
		&destination,

		&res.Status,
		&data,
		&action,
		&res.ActionNextHold,
		&res.Direction,
		&res.MuteDirection,

		&res.HangupBy,
		&res.HangupReason,

		&res.DialrouteID,
		&dialroutes,

		&res.TMCreate,
		&res.TMUpdate,
		&tmDelete,

		&res.TMProgressing,
		&res.TMRinging,
		&res.TMHangup,
	); err != nil {
		return nil, fmt.Errorf("could not scan the row. callGetFromRow. err: %v", err)
	}

	// ChainedCallIDs
	if chainedCallIDs.Valid {
		if err := json.Unmarshal([]byte(chainedCallIDs.String), &res.ChainedCallIDs); err != nil {
			return nil, fmt.Errorf("could not unmarshal the chained_call_ids. callGetFromRow. err: %v", err)
		}
	}
	if res.ChainedCallIDs == nil {
		res.ChainedCallIDs = []uuid.UUID{}
	}

	// RecordingIDs
	if recordingIDs.Valid {
		if err := json.Unmarshal([]byte(recordingIDs.String), &res.RecordingIDs); err != nil {
			return nil, fmt.Errorf("could not unmarshal the recording_ids. callGetFromRow. err: %v", err)
		}
	}
	if res.RecordingIDs == nil {
		res.RecordingIDs = []uuid.UUID{}
	}

	// Source
	if source.Valid {
		if err := json.Unmarshal([]byte(source.String), &res.Source); err != nil {
			return nil, fmt.Errorf("could not unmarshal the source. callGetFromRow. err: %v", err)
		}
	} else {
		res.Source = address.Address{}
	}

	// Destination
	if destination.Valid {
		if err := json.Unmarshal([]byte(destination.String), &res.Destination); err != nil {
			return nil, fmt.Errorf("could not unmarshal the destination. callGetFromRow. err: %v", err)
		}
	} else {
		res.Destination = address.Address{}
	}

	// Data
	if data.Valid {
		if err := json.Unmarshal([]byte(data.String), &res.Data); err != nil {
			return nil, fmt.Errorf("could not unmarshal the data. callGetFromRow. err: %v", err)
		}
	}
	if res.Data == nil {
		res.Data = map[call.DataType]string{}
	}

	// Action
	if action.Valid {
		if err := json.Unmarshal([]byte(action.String), &res.Action); err != nil {
			return nil, fmt.Errorf("could not unmarshal the action. callGetFromRow. err: %v", err)
		}
	} else {
		res.Action = fmaction.Action{}
	}

	// Dialroutes
	if dialroutes.Valid {
		if err := json.Unmarshal([]byte(dialroutes.String), &res.Dialroutes); err != nil {
			return nil, fmt.Errorf("could not unmarshal the dialroutes. callGetFromRow. err: %v", err)
		}
	}
	if res.Dialroutes == nil {
		res.Dialroutes = []rmroute.Route{}
	}

	// TMDelete
	if tmDelete.Valid {
		res.TMDelete = tmDelete.String
	} else {
		res.TMDelete = DefaultTimeStamp
	}

	return res, nil
}

// CallCreate creates new call record.
func (h *handler) CallCreate(ctx context.Context, c *call.Call) error {
	q := `insert into call_calls(
		id,
		customer_id,
		owner_type,
        owner_id,

		channel_id,
		bridge_id,

		flow_id,
		active_flow_id,
		confbridge_id,
		type,

		master_call_id,
		chained_call_ids,
		recording_id,
		recording_ids,
		external_media_id,
		groupcall_id,

		source,
		source_target,
		destination,
		destination_target,

		status,
		data,
		action,
		action_next_hold,
		direction,
		mute_direction,

		hangup_by,
		hangup_reason,

		dialroute_id,
		dialroutes,

		tm_create,
		tm_update,
		tm_delete,

		tm_progressing,
		tm_ringing,
		tm_hangup
	) values(
		?, ?, ?, ?,
		?, ?,
		?, ?, ?, ?,
		?, ?, ?, ?, ?, ?,
		?, ?, ?, ?,
		?, ?, ?, ?, ?, ?,
		?, ?,
		?, ?,
		?, ?, ?,
		?, ?, ?
		)`

	if c.ChainedCallIDs == nil {
		c.ChainedCallIDs = []uuid.UUID{}
	}
	tmpChainedCallIDs, err := json.Marshal(c.ChainedCallIDs)
	if err != nil {
		return fmt.Errorf("could not marshal calls. CallCreate. err: %v", err)
	}

	if c.RecordingIDs == nil {
		c.RecordingIDs = []uuid.UUID{}
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

	tmpDialroutes, err := json.Marshal(c.Dialroutes)
	if err != nil {
		return fmt.Errorf("could not marshal dialroutes. CallCreate. err: %v", err)
	}

	_, err = h.db.Exec(q,
		c.ID.Bytes(),
		c.CustomerID.Bytes(),
		c.OwnerType,
		c.OwnerID.Bytes(),

		c.ChannelID,
		c.BridgeID,

		c.FlowID.Bytes(),
		c.ActiveFlowID.Bytes(),
		c.ConfbridgeID.Bytes(),
		c.Type,

		c.MasterCallID.Bytes(),
		tmpChainedCallIDs,
		c.RecordingID.Bytes(),
		tmpRecordingIDs,
		c.ExternalMediaID.Bytes(),
		c.GroupcallID.Bytes(),

		tmpSource,
		c.Source.Target,
		tmpDestination,
		c.Destination.Target,

		c.Status,
		tmpData,
		tmpAction,
		c.ActionNextHold,
		c.Direction,
		c.MuteDirection,

		c.HangupBy,
		c.HangupReason,

		c.DialrouteID.Bytes(),
		tmpDialroutes,

		h.utilHandler.TimeGetCurTime(),
		DefaultTimeStamp,
		DefaultTimeStamp,

		DefaultTimeStamp,
		DefaultTimeStamp,
		DefaultTimeStamp,
	)
	if err != nil {
		return fmt.Errorf("could not execute. CallCreate. err: %v", err)
	}

	// update the cache
	_ = h.callUpdateToCache(ctx, c.ID)

	return nil
}

// CallGet returns call.
func (h *handler) CallGet(ctx context.Context, id uuid.UUID) (*call.Call, error) {

	res, err := h.callGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.callGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	// set to the cache
	_ = h.callSetToCache(ctx, res)

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

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.callGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not get call. CallGetByChannelID, err: %v", err)
	}

	return res, nil
}

// CallGets returns a list of calls.
func (h *handler) CallGets(ctx context.Context, size uint64, token string, filters map[string]string) ([]*call.Call, error) {

	// prepare
	q := fmt.Sprintf(`%s
	where
		tm_create < ?
	`, callSelect)

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	values := []interface{}{
		token,
	}

	for k, v := range filters {
		switch k {
		case "customer_id", "owner_id", "flow_id", "active_flow_id", "confbridge_id", "master_call_id", "recording_id", "external_media_id", "groupcall_id", "dialroute_id":
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
		return nil, fmt.Errorf("could not query. CallGets. err: %v", err)
	}
	defer rows.Close()

	res := []*call.Call{}
	for rows.Next() {
		u, err := h.callGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not get data. callGetFromRow, err: %v", err)
		}

		res = append(res, u)
	}

	return res, nil
}

// CallSetBridgeID sets the call bridge id
func (h *handler) CallSetBridgeID(ctx context.Context, id uuid.UUID, bridgeID string) error {
	// prepare
	q := `
	update
		call_calls
	set
		bridge_id = ?,
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, bridgeID, h.utilHandler.TimeGetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. callSetBridgeID. err: %v", err)
	}

	// update the cache
	_ = h.callUpdateToCache(ctx, id)

	return nil
}

// CallSetStatusRinging sets the call status to ringing
func (h *handler) CallSetStatusRinging(ctx context.Context, id uuid.UUID) error {
	// prepare
	q := `
	update
		call_calls
	set
		status = ?,
		tm_update = ?,
		tm_ringing = ?
	where
		id = ?
	`

	ts := h.utilHandler.TimeGetCurTime()
	_, err := h.db.Exec(q, call.StatusRinging, ts, ts, id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. CallSetStatusRinging. err: %v", err)
	}

	// update the cache
	_ = h.callUpdateToCache(ctx, id)

	return nil
}

// CallSetStatusProgressing sets the call status to progressing
func (h *handler) CallSetStatusProgressing(ctx context.Context, id uuid.UUID) error {
	// prepare
	q := `
	update
		call_calls
	set
		status = ?,
		tm_update = ?,
		tm_progressing = ?
	where
		id = ?
	`

	ts := h.utilHandler.TimeGetCurTime()
	_, err := h.db.Exec(q, call.StatusProgressing, ts, ts, id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. CallSetStatusProgressing. err: %v", err)
	}

	// update the cache
	_ = h.callUpdateToCache(ctx, id)

	return nil
}

// CallSetStatus sets the call status without update the timestamp for status
func (h *handler) CallSetStatus(ctx context.Context, id uuid.UUID, status call.Status) error {
	// prepare
	q := `
	update
		call_calls
	set
		status = ?,
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, status, h.utilHandler.TimeGetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. CallSetStatus. err: %v", err)
	}

	// update the cache
	_ = h.callUpdateToCache(ctx, id)

	return nil
}

// CallSetStatus sets the call status
func (h *handler) CallSetHangup(ctx context.Context, id uuid.UUID, reason call.HangupReason, hangupBy call.HangupBy) error {

	// prepare
	q := `
	update
		call_calls
	set
		status = ?,
		hangup_by = ?,
		hangup_reason = ?,
		tm_update = ?,
		tm_hangup = ?
	where
		id = ?
	`

	ts := h.utilHandler.TimeGetCurTime()
	_, err := h.db.Exec(q, string(call.StatusHangup), hangupBy, reason, ts, ts, id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. CallSetHangup. err: %v", err)
	}

	// update the cache
	_ = h.callUpdateToCache(ctx, id)

	return nil
}

// CallSetFlowID sets the call status
func (h *handler) CallSetFlowID(ctx context.Context, id, flowID uuid.UUID) error {

	// prepare
	q := `
	update
		call_calls
	set
		flow_id = ?,
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, flowID.Bytes(), h.utilHandler.TimeGetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. CallSetFlowID. err: %v", err)
	}

	// update the cache
	_ = h.callUpdateToCache(ctx, id)

	return nil
}

// CallSetConfbridgeID sets the call's confbridge_id
func (h *handler) CallSetConfbridgeID(ctx context.Context, id, confbridgeID uuid.UUID) error {

	// prepare
	q := `
	update
		call_calls
	set
		confbridge_id = ?,
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, confbridgeID.Bytes(), h.utilHandler.TimeGetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. CallSetConfbridgeID. err: %v", err)
	}

	// update the cache
	_ = h.callUpdateToCache(ctx, id)

	return nil
}

// CallSetAction sets the call status
func (h *handler) CallSetActionAndActionNextHold(ctx context.Context, id uuid.UUID, action *fmaction.Action, hold bool) error {

	// prepare
	q := `
	update
		call_calls
	set
		action = ?,
		action_next_hold = ?,
		tm_update = ?
	where
		id = ?
	`

	tmpAction, err := json.Marshal(action)
	if err != nil {
		return err
	}

	_, err = h.db.Exec(q, tmpAction, hold, h.utilHandler.TimeGetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. CallSetActionAndActionNextHold. err: %v", err)
	}

	// update the cache
	_ = h.callUpdateToCache(ctx, id)

	return nil
}

// callGetFromCache returns call from the cache.
func (h *handler) callGetFromCache(ctx context.Context, id uuid.UUID) (*call.Call, error) {

	// get from cache
	res, err := h.cache.CallGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// callGetFromDB returns call from the DB.
func (h *handler) callGetFromDB(ctx context.Context, id uuid.UUID) (*call.Call, error) {

	// prepare
	q := fmt.Sprintf("%s where id = ?", callSelect)

	row, err := h.db.Query(q, id.Bytes())
	if err != nil {
		return nil, fmt.Errorf("could not query. callGetFromDB. err: %v", err)
	}
	defer row.Close()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.callGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not get call. callGetFromDB, err: %v", err)
	}

	return res, nil
}

// callUpdateToCache gets the call from the DB and update the cache.
func (h *handler) callUpdateToCache(ctx context.Context, id uuid.UUID) error {

	res, err := h.callGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.callSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// callSetToCache sets the given call to the cache
func (h *handler) callSetToCache(ctx context.Context, call *call.Call) error {
	if err := h.cache.CallSet(ctx, call); err != nil {
		return err
	}

	return nil
}

// CallAddChainedCallID adds the call id to the given call's chained_call_ids.
func (h *handler) CallAddChainedCallID(ctx context.Context, id, chainedCallID uuid.UUID) error {
	// prepare
	q := `
	update call_calls set
		chained_call_ids = json_array_append(
			chained_call_ids,
			'$',
			?
		),
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, chainedCallID.String(), h.utilHandler.TimeGetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. CallAddChainedCallID. err: %v", err)
	}

	// update the cache
	_ = h.callUpdateToCache(ctx, id)

	return nil
}

// CallRemoveChainedCallID removes the call id from the given call's chained_call_ids.
func (h *handler) CallRemoveChainedCallID(ctx context.Context, id, chainedCallID uuid.UUID) error {
	// prepare
	q := `
	update call_calls set
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

	_, err := h.db.Exec(q, chainedCallID.String(), h.utilHandler.TimeGetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. CallRemoveChainedCallID. err: %v", err)
	}

	// update the cache
	_ = h.callUpdateToCache(ctx, id)

	return nil
}

// CallSetMasterCallID sets the call's master_call_id
func (h *handler) CallSetMasterCallID(ctx context.Context, id uuid.UUID, callID uuid.UUID) error {

	// prepare
	q := `
	update
		call_calls
	set
		master_call_id = ?,
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, callID.Bytes(), h.utilHandler.TimeGetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. CallSetMasterCallID. err: %v", err)
	}

	// update the cache
	_ = h.callUpdateToCache(ctx, id)

	return nil
}

// CallSetRecordingID sets the given recordID to recording_id.
func (h *handler) CallSetRecordingID(ctx context.Context, id uuid.UUID, recordID uuid.UUID) error {
	// prepare
	q := `
	update call_calls set
		recording_id = ?,
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, recordID.Bytes(), h.utilHandler.TimeGetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. CallSetRecordingID. err: %v", err)
	}

	// update the cache
	_ = h.callUpdateToCache(context.Background(), id)

	return nil
}

// CallSetExternalMediaID sets the call's external_media_id
func (h *handler) CallSetExternalMediaID(ctx context.Context, id uuid.UUID, externalMediaID uuid.UUID) error {

	// prepare
	q := `
	update
		call_calls
	set
		external_media_id = ?,
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, externalMediaID.Bytes(), h.utilHandler.TimeGetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. CallSetExternalMediaID. err: %v", err)
	}

	// update the cache
	_ = h.callUpdateToCache(ctx, id)

	return nil
}

// CallSetForRouteFailover sets the call for route failover.
func (h *handler) CallSetForRouteFailover(ctx context.Context, id uuid.UUID, channelID string, dialrouteID uuid.UUID) error {
	// prepare
	q := `
	update call_calls set
		bridge_id = '',

		channel_id = ?,
		dialroute_id = ?,

		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, channelID, dialrouteID.Bytes(), h.utilHandler.TimeGetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. CallSetForRouteFailover. err: %v", err)
	}

	// update the cache
	_ = h.callUpdateToCache(ctx, id)

	return nil
}

// CallAddRecordingIDs adds the given recording_id into the recording_ids.
func (h *handler) CallAddRecordingIDs(ctx context.Context, id uuid.UUID, recordID uuid.UUID) error {
	// prepare
	q := `
	update call_calls set
		recording_ids = json_array_append(
			recording_ids,
			'$',
			?
		),
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, recordID.String(), h.utilHandler.TimeGetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. CallAddRecordingIDs. err: %v", err)
	}

	// update the cache
	_ = h.callUpdateToCache(ctx, id)

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
		_ = tx.Rollback()
		return nil, nil, fmt.Errorf("could not query. CallTXStart. err: %v", err)
	}
	defer row.Close()

	if !row.Next() {
		_ = tx.Rollback()
		return nil, nil, ErrNotFound
	}

	res, err := h.callGetFromRow(row)
	if err != nil {
		_ = tx.Rollback()
		return nil, nil, fmt.Errorf("could not get call. CallTXStart, err: %v", err)
	}

	return tx, res, nil
}

func (h *handler) CallTXFinish(tx *sql.Tx, commit bool) {
	if commit {
		_ = tx.Commit()
	} else {
		_ = tx.Rollback()
	}
}

// CallTXAddChainedCallID adds the call id to the given call's chained_call_ids in a transaction mode.
func (h *handler) CallTXAddChainedCallID(tx *sql.Tx, id, chainedCallID uuid.UUID) error {
	// prepare
	q := `
	update call_calls set
		chained_call_ids = json_array_append(
			chained_call_ids,
			'$',
			?
		),
		tm_update = ?
	where
		id = ?
	`

	_, err := tx.Exec(q, chainedCallID.String(), h.utilHandler.TimeGetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. CallAddChainedCallID. err: %v", err)
	}

	// update the cache
	_ = h.callUpdateToCache(context.Background(), id)

	return nil
}

// CallTXRemoveChainedCallID removes the call id from the given call's chained_call_ids in a transaction mode.
func (h *handler) CallTXRemoveChainedCallID(tx *sql.Tx, id, chainedCallID uuid.UUID) error {
	// prepare
	q := `
	update call_calls set
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

	_, err := tx.Exec(q, chainedCallID.String(), h.utilHandler.TimeGetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. CallRemoveChainedCallID. err: %v", err)
	}

	// update the cache
	_ = h.callUpdateToCache(context.Background(), id)

	return nil
}

// CallSetActionNextHold sets the action_next_hold.
func (h *handler) CallSetActionNextHold(ctx context.Context, id uuid.UUID, hold bool) error {
	// prepare
	q := `
	update call_calls set
		action_next_hold = ?,
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, hold, h.utilHandler.TimeGetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. CallSetActionNextHold. err: %v", err)
	}

	// update the cache
	_ = h.callUpdateToCache(ctx, id)

	return nil
}

// CallDelete deletes the call
func (h *handler) CallDelete(ctx context.Context, id uuid.UUID) error {
	//prepare
	q := `
	update call_calls set
		tm_update = ?,
		tm_delete = ?
	where
		id = ?
	`

	ts := h.utilHandler.TimeGetCurTime()
	_, err := h.db.Exec(q, ts, ts, id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. CallDelete. err: %v", err)
	}

	// update the cache
	_ = h.callUpdateToCache(ctx, id)

	return nil
}

// CallSetData sets the call data
func (h *handler) CallSetData(ctx context.Context, id uuid.UUID, data map[call.DataType]string) error {
	// prepare
	q := `
	update
		call_calls
	set
		data = ?,
		tm_update = ?
	where
		id = ?
	`

	tmpData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("could not marshal data. CallSetData. err: %v", err)
	}

	_, err = h.db.Exec(q, tmpData, h.utilHandler.TimeGetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. CallSetData. err: %v", err)
	}

	// update the cache
	_ = h.callUpdateToCache(ctx, id)

	return nil
}

// CallSetMuteDirection sets the call mute direction
func (h *handler) CallSetMuteDirection(ctx context.Context, id uuid.UUID, muteDirection call.MuteDirection) error {
	// prepare
	q := `
	update
		call_calls
	set
		mute_direction = ?,
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, muteDirection, h.utilHandler.TimeGetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. CallSetMuteDirection. err: %v", err)
	}

	// update the cache
	_ = h.callUpdateToCache(ctx, id)

	return nil
}
