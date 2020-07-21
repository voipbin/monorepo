package dbhandler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	uuid "github.com/gofrs/uuid"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/action"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/callhandler/models/call"
)

// CallCreate creates new call record.
func (h *handler) CallCreate(ctx context.Context, call *call.Call) error {
	q := `insert into calls(
		id,
		asterisk_id,
		channel_id,
		flow_id,
		conference_id,
		type,

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
		?, ?, ?, ?, ?, ?,
		?, ?, ?, ?,
		?, ?, ?, ?, ?,?,
		?
		)`

	tmpSource, err := json.Marshal(call.Source)
	if err != nil {
		return fmt.Errorf("could not marshal source. CallCreate. err: %v", err)
	}

	tmpDestination, err := json.Marshal(call.Destination)
	if err != nil {
		return fmt.Errorf("could not marshal destination. CallCreate. err: %v", err)
	}

	tmpData, err := json.Marshal(call.Data)
	if err != nil {
		return fmt.Errorf("could not marshal data. CallCreate. err: %v", err)
	}

	tmpAction, err := json.Marshal(call.Action)
	if err != nil {
		return fmt.Errorf("could not marshal action. CallCreate. err: %v", err)
	}

	_, err = h.db.Exec(q,
		call.ID.Bytes(),
		call.AsteriskID,
		call.ChannelID,
		call.FlowID.Bytes(),
		call.ConfID.Bytes(),
		call.Type,

		tmpSource,
		call.Source.Target,
		tmpDestination,
		call.Destination.Target,

		call.Status,
		tmpData,
		tmpAction,
		call.Direction,
		call.HangupBy,
		call.HangupReason,

		call.TMCreate,
	)
	if err != nil {
		return fmt.Errorf("could not execute. CallCreate. err: %v", err)
	}

	return nil
}

// CallGet returns call.
func (h *handler) CallGet(ctx context.Context, id uuid.UUID) (*call.Call, error) {

	// prepare
	q := `
	select
		id,
		asterisk_id,
		channel_id,
		flow_id,
		conference_id,
		type,

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
	where
		id = ?
	`

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

// CallGet returns call.
func (h *handler) CallGetByChannelIDAndAsteriskID(ctx context.Context, channelID, asteriskID string) (*call.Call, error) {

	// prepare
	q := `
	select
		id,
		asterisk_id,
		channel_id,
		flow_id,
		conference_id,
		type,

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
	where
		channel_id = ? and asterisk_id = ?
	`

	row, err := h.db.Query(q, channelID, asteriskID)
	if err != nil {
		return nil, fmt.Errorf("could not query. CallGetByChannelIDAndAsteriskID. err: %v", err)
	}
	defer row.Close()

	if row.Next() == false {
		return nil, ErrNotFound
	}

	res, err := h.callGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not get call. CallGetByChannelIDAndAsteriskID, err: %v", err)
	}

	return res, nil
}

// callGetFromRow gets the call from the row.
func (h *handler) callGetFromRow(row *sql.Rows) (*call.Call, error) {
	var data string
	var source string
	var destination string
	var action string
	res := &call.Call{}
	if err := row.Scan(
		&res.ID,
		&res.AsteriskID,
		&res.ChannelID,
		&res.FlowID,
		&res.ConfID,
		&res.Type,

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

	return nil
}

// CallSetStatus sets the call status
func (h *handler) CallSetStatus(ctx context.Context, id uuid.UUID, status call.Status, tmStatus string) error {

	switch status {
	case call.StatusRinging:
		return h.callSetStatusRinging(ctx, id, tmStatus)
	case call.StatusProgressing:
		return h.callSetStatusProgressing(ctx, id, tmStatus)
	default:
		return h.callSetStatus(ctx, id, status)
	}
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

	return nil
}
