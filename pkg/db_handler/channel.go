package dbhandler

import (
	"context"
	"encoding/json"
	"fmt"

	"gitlab.com/voipbin/bin-manager/call-manager/pkg/channel"
)

// ChannelCreate creates new channel record and returns the created channel record.
func (h *Handler) ChannelCreate(ctx context.Context, channel channel.Channel) error {
	q := `insert into channels(
		asterisk_id,
		id,
		name,
		tech,

		src_name,
		src_number,
		dst_name,
		dst_number,

		state,
		data,

		dial_result,
		hangup_cause,
		tm_create
	) values(
		?, ?, ?, ?, 
		?, ?, ?, ?, 
		?, ?, 
		?, ?, ?
		)`
	stmt, err := h.db.PrepareContext(ctx, q)
	if err != nil {
		return fmt.Errorf("dbhandler: Could not prepare. err: %v", err)
	}
	defer stmt.Close()

	tmpData, err := json.Marshal(channel.Data)
	if err != nil {
		return fmt.Errorf("dbhandler: Could not marshal. err: %v", err)
	}

	_, err = stmt.ExecContext(ctx,
		channel.AsteriskID,
		channel.ID,
		channel.Name,
		channel.Tech,

		channel.SourceName,
		channel.SourceNumber,
		channel.DestinationName,
		channel.DestinationNumber,

		channel.State,
		tmpData,

		channel.DialResult,
		channel.HangupCause,

		channel.TMCreate,
	)
	if err != nil {
		return fmt.Errorf("dbhandler: Could not execute query. err: %v", err)
	}

	return nil
}

// ChannelGet returns channel.
func (h *Handler) ChannelGet(ctx context.Context, asteriskID, id string) (*channel.Channel, error) {

	// prepare
	q := `select 
	asterisk_id,
  id,
  name,
  tech,

  src_name,
  src_number,
  dst_name,
  dst_number,

  state,
  data,

  dial_result,
  hangup_cause,

	coalesce(tm_create, '') as tm_create,
	coalesce(tm_update, '') as tm_update,

	coalesce(tm_answer, '') as tm_answer,
	coalesce(tm_ringing, '') as tm_ringing,
	coalesce(tm_end, '') as tm_end

	from 
	channels 
	where 
	asterisk_id = ? and id = ?`
	stmt, err := h.db.PrepareContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("dbhandler: Could not prepare. err: %v", err)
	}
	defer stmt.Close()

	// query
	row, err := stmt.QueryContext(ctx, asteriskID, id)
	if err != nil {
		return nil, fmt.Errorf("dbhandler: Could not query. err: %v", err)
	}
	defer row.Close()

	if row.Next() == false {
		return nil, fmt.Errorf("dbhandler: Could not get row. err: %v", err)
	}

	var data string
	res := &channel.Channel{}
	if err := row.Scan(
		&res.AsteriskID,
		&res.ID,
		&res.Name,
		&res.Tech,

		&res.SourceName,
		&res.SourceNumber,
		&res.DestinationName,
		&res.DestinationNumber,

		&res.State,
		&data,

		&res.DialResult,
		&res.HangupCause,

		&res.TMCreate,
		&res.TMUpdate,

		&res.TMAnswer,
		&res.TMRinging,
		&res.TMEnd,
	); err != nil {
		return nil, fmt.Errorf("dbhandler: Could not scan the row. err: %v", err)
	}

	if err := json.Unmarshal([]byte(data), &res.Data); err != nil {
		return nil, fmt.Errorf("dbhandler: Could not unmarshal the result. err: %v", err)
	}

	return res, nil
}

// ChannelEnd updates the channel end.
func (h *Handler) ChannelEnd(ctx context.Context, asteriskID, id, timestamp string, hangup int) error {

	// prepare
	q := `
	update channels set
		hangup_cause = ?,
		tm_update = ?,
		tm_end = ?
	where
		asterisk_id = ?
		and id = ?
	`
	stmt, err := h.db.PrepareContext(ctx, q)
	if err != nil {
		return fmt.Errorf("dbhandler: Could not prepare. err: %v", err)
	}
	defer stmt.Close()

	// execute
	_, err = stmt.ExecContext(ctx, hangup, timestamp, timestamp, asteriskID, id)
	if err != nil {
		return fmt.Errorf("dbhandler: Could not query. err: %v", err)
	}

	return nil
}
