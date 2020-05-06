package dbhandler

import (
	"context"
	"encoding/json"
	"fmt"

	uuid "github.com/satori/go.uuid"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/action"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/call"
)

// CallCreate creates new call record.
func (h *handler) CallCreate(ctx context.Context, call *call.Call) error {
	q := `insert into cm_calls(
		id,
		asterisk_id,
		channel_id,
		flow_id,
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
		?, ?, ?, ?, ?,
		?, ?, ?, ?,
		?, ?, ?, ?, ?,?,
		?
		)`
	stmt, err := h.db.PrepareContext(ctx, q)
	if err != nil {
		return fmt.Errorf("could not prepare. CallCreate. err: %v", err)
	}
	defer stmt.Close()

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

	_, err = stmt.ExecContext(ctx,
		call.ID.Bytes(),
		call.AsteriskID,
		call.ChannelID,
		call.FlowID.Bytes(),
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
		return fmt.Errorf("could not execute query. CallCreate. err: %v", err)
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
		cm_calls
	where
		id = ?
	`
	stmt, err := h.db.PrepareContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("could not prepare. CallGet. err: %v", err)
	}
	defer stmt.Close()

	// query
	row, err := stmt.QueryContext(ctx, id.Bytes())
	if err != nil {
		return nil, fmt.Errorf("could not query. CallGet. err: %v", err)
	}
	defer row.Close()

	if row.Next() == false {
		return nil, ErrNotFound
	}

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
		return nil, fmt.Errorf("could not scan the row. CallGet. err: %v", err)
	}

	if err := json.Unmarshal([]byte(data), &res.Data); err != nil {
		return nil, fmt.Errorf("could not unmarshal the data. CallGet. err: %v", err)
	}
	if err := json.Unmarshal([]byte(action), &res.Action); err != nil {
		return nil, fmt.Errorf("could not unmarshal the action. CallGet. err: %v", err)
	}
	if err := json.Unmarshal([]byte(source), &res.Source); err != nil {
		return nil, fmt.Errorf("could not unmarshal the source. CallGet. err: %v", err)
	}
	if err := json.Unmarshal([]byte(destination), &res.Destination); err != nil {
		return nil, fmt.Errorf("could not unmarshal the destination. CallGet. err: %v", err)
	}

	return res, nil
}

// CallGet returns call.
func (h *handler) CallGetByChannelID(ctx context.Context, channelID string) (*call.Call, error) {

	// prepare
	q := `
	select
		id,
		asterisk_id,
		channel_id,
		flow_id,
		type,

		source,
		destination,

		status,
		data,
		direction,
		hangup_by,
		hangup_reason,

		coalesce(tm_create, '') as tm_create,
		coalesce(tm_update, '') as tm_update,

		coalesce(tm_progressing, '') as tm_progressing,
		coalesce(tm_ringing, '') as tm_ringing,
		coalesce(tm_hangup, '') as tm_hangup

	from
		cm_calls
	where
		channel_id = ?
	`
	stmt, err := h.db.PrepareContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("could not prepare. CallGetByChannelID. err: %v", err)
	}
	defer stmt.Close()

	// query
	row, err := stmt.QueryContext(ctx, channelID)
	if err != nil {
		return nil, fmt.Errorf("could not query. CallGetByChannelID. err: %v", err)
	}
	defer row.Close()

	if row.Next() == false {
		return nil, ErrNotFound
	}

	var data string
	var source string
	var destination string
	res := &call.Call{}
	if err := row.Scan(
		&res.ID,
		&res.AsteriskID,
		&res.ChannelID,
		&res.FlowID,
		&res.Type,

		&source,
		&destination,

		&res.Status,
		&data,
		&res.Direction,
		&res.HangupBy,
		&res.HangupReason,

		&res.TMCreate,
		&res.TMUpdate,

		&res.TMProgressing,
		&res.TMRinging,
		&res.TMHangup,
	); err != nil {
		return nil, fmt.Errorf("could not scan the row. CallGetByChannelID. err: %v", err)
	}

	if err := json.Unmarshal([]byte(data), &res.Data); err != nil {
		return nil, fmt.Errorf("could not unmarshal the data. CallGetByChannelID. err: %v", err)
	}
	if err := json.Unmarshal([]byte(source), &res.Source); err != nil {
		return nil, fmt.Errorf("could not unmarshal the source. CallGetByChannelID. err: %v", err)
	}
	if err := json.Unmarshal([]byte(destination), &res.Destination); err != nil {
		return nil, fmt.Errorf("could not unmarshal the destination. CallGetByChannelID. err: %v", err)
	}

	return res, nil
}

// callSetStatusRinging sets the call status to ringing
func (h *handler) callSetStatusRinging(ctx context.Context, id uuid.UUID, tmStatus string) error {
	// prepare
	q := `
	update
		cm_calls
	set
		status = ?,
		tm_update = ?,
		tm_ringing = ?
	where
		id = ?
	`
	stmt, err := h.db.PrepareContext(ctx, q)
	if err != nil {
		return fmt.Errorf("could not prepare. CallSetStatusRinging. err: %v", err)
	}
	defer stmt.Close()

	// query
	_, err = stmt.ExecContext(ctx, call.StatusRinging, getCurTime(), tmStatus, id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute query. CallSetStatusRinging. err: %v", err)
	}

	return nil
}

// callSetStatusProgressing sets the call status to progressing
func (h *handler) callSetStatusProgressing(ctx context.Context, id uuid.UUID, tmStatus string) error {
	// prepare
	q := `
	update
		cm_calls
	set
		status = ?,
		tm_update = ?,
		tm_progressing = ?
	where
		id = ?
	`
	stmt, err := h.db.PrepareContext(ctx, q)
	if err != nil {
		return fmt.Errorf("could not prepare. callSetStatusProgressing. err: %v", err)
	}
	defer stmt.Close()

	// query
	_, err = stmt.ExecContext(ctx, call.StatusProgressing, getCurTime(), tmStatus, id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute query. callSetStatusProgressing. err: %v", err)
	}

	return nil
}

// callSetStatus sets the call status without update the timestamp for status
func (h *handler) callSetStatus(ctx context.Context, id uuid.UUID, status call.Status) error {
	// prepare
	q := `
	update
		cm_calls
	set
		status = ?,
		tm_update = ?
	where
		id = ?
	`
	stmt, err := h.db.PrepareContext(ctx, q)
	if err != nil {
		return fmt.Errorf("could not prepare. callSetStatus. err: %v", err)
	}
	defer stmt.Close()

	// query
	_, err = stmt.ExecContext(ctx, status, getCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute query. callSetStatus. err: %v", err)
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
		cm_calls
	set
		status = ?,
		hangup_by = ?,
		hangup_reason = ?,
		tm_update = ?,
		tm_hangup = ?
	where
		id = ?
	`
	stmt, err := h.db.PrepareContext(ctx, q)
	if err != nil {
		return fmt.Errorf("could not prepare. CallSetStatus. err: %v", err)
	}
	defer stmt.Close()

	// query
	_, err = stmt.ExecContext(ctx, string(call.StatusHangup), hangupBy, reason, tmUpdate, tmUpdate, id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute query. CallSetStatus. err: %v", err)
	}

	return nil
}

// CallSetFlowID sets the call status
func (h *handler) CallSetFlowID(ctx context.Context, id, flowID uuid.UUID, tmUpdate string) error {

	// prepare
	q := `
	update
		cm_calls
	set
		flow_id = ?,
		tm_update = ?
	where
		id = ?
	`
	stmt, err := h.db.PrepareContext(ctx, q)
	if err != nil {
		return fmt.Errorf("could not prepare. CallSetFlowID. err: %v", err)
	}
	defer stmt.Close()

	// query
	_, err = stmt.ExecContext(ctx, flowID.Bytes(), tmUpdate, id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute query. CallSetFlowID. err: %v", err)
	}

	return nil
}

// CallSetAction sets the call status
func (h *handler) CallSetAction(ctx context.Context, id uuid.UUID, action *action.Action) error {

	// prepare
	q := `
	update
		cm_calls
	set
		action = ?,
		tm_update = ?
	where
		id = ?
	`
	stmt, err := h.db.PrepareContext(ctx, q)
	if err != nil {
		return fmt.Errorf("could not prepare. CallSetAction. err: %v", err)
	}
	defer stmt.Close()

	actStr, err := json.Marshal(action)
	if err != nil {
		return err
	}

	// query
	_, err = stmt.ExecContext(ctx, actStr, getCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute query. CallSetAction. err: %v", err)
	}

	return nil
}
