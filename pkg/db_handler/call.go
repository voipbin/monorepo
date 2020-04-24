package dbhandler

import (
	"context"
	"encoding/json"
	"fmt"

	uuid "github.com/satori/go.uuid"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/call"
)

// CallCreate creates new call record.
func (h *handler) CallCreate(ctx context.Context, call *call.Call) error {
	q := `insert into calls(
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
		direction,
		hangup_by,
		hangup_reason,

		tm_create
	) values(
		?, ?, ?, ?, ?,
		?, ?, ?, ?,
		?, ?, ?, ?, ?,
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
		return fmt.Errorf("could not marshal. CallCreate. err: %v", err)
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
		return nil, fmt.Errorf("could not get row. CallGet. err: %v", err)
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
		return nil, fmt.Errorf("could not scan the row. CallGet. err: %v", err)
	}

	if err := json.Unmarshal([]byte(data), &res.Data); err != nil {
		return nil, fmt.Errorf("could not unmarshal the data. CallGet. err: %v", err)
	}
	if err := json.Unmarshal([]byte(source), &res.Source); err != nil {
		return nil, fmt.Errorf("could not unmarshal the source. CallGet. err: %v", err)
	}
	if err := json.Unmarshal([]byte(destination), &res.Destination); err != nil {
		return nil, fmt.Errorf("could not unmarshal the destination. CallGet. err: %v", err)
	}

	return res, nil
}

// CallSetStatus sets the call status
func (h *handler) CallSetStatus(ctx context.Context, id uuid.UUID, status call.Status, tmUpdate string) error {

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
	stmt, err := h.db.PrepareContext(ctx, q)
	if err != nil {
		return fmt.Errorf("could not prepare. CallSetStatus. err: %v", err)
	}
	defer stmt.Close()

	// query
	_, err = stmt.ExecContext(ctx, status, tmUpdate, id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute query. CallSetStatus. err: %v", err)
	}

	return nil
}
