package dbhandler

import (
	"context"
	"encoding/json"
	"fmt"

	"gitlab.com/voipbin/bin-manager/call-manager/pkg/ari"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/channel"
)

// ChannelCreate creates new channel record and returns the created channel record.
func (h *handler) ChannelCreate(ctx context.Context, channel *channel.Channel) error {
	q := `insert into cm_channels(
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
		stasis,

		dial_result,
		hangup_cause,
		tm_create
	) values(
		?, ?, ?, ?,
		?, ?, ?, ?,
		?, ?, ?,
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
		channel.Stasis,

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
func (h *handler) ChannelGet(ctx context.Context, asteriskID, id string) (*channel.Channel, error) {

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
	stasis,

  dial_result,
  hangup_cause,

	coalesce(tm_create, '') as tm_create,
	coalesce(tm_update, '') as tm_update,

	coalesce(tm_answer, '') as tm_answer,
	coalesce(tm_ringing, '') as tm_ringing,
	coalesce(tm_end, '') as tm_end

	from
		cm_channels
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
		return nil, ErrNotFound
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
		&res.Stasis,

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
	if res.Data == nil {
		res.Data = make(map[string]interface{})
	}
	// }

	return res, nil
}

// ChannelSetData sets the data
func (h *handler) ChannelSetData(ctx context.Context, asteriskID, id string, data map[string]interface{}) error {
	//prepare
	q := `
	update cm_channels set
		data = ?,
		tm_update = ?
	where
		asterisk_id = ?
		and id = ?
	`
	stmt, err := h.db.PrepareContext(ctx, q)
	if err != nil {
		return fmt.Errorf("could not prepare. ChannelSetData. err: %v", err)
	}
	defer stmt.Close()

	tmpData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("dbhandler: Could not marshal. ChannelSetData. err: %v", err)
	}

	// execute
	_, err = stmt.ExecContext(ctx, tmpData, getCurTime(), asteriskID, id)
	if err != nil {
		return fmt.Errorf("could not execute. ChannelSetData. err: %v", err)
	}

	return nil
}

// ChannelSetStasis sets the stasis
func (h *handler) ChannelSetStasis(ctx context.Context, asteriskID, id, stasis string) error {
	//prepare
	q := `
	update cm_channels set
		stasis = ?,
		tm_update = ?
	where
		asterisk_id = ?
		and id = ?
	`
	stmt, err := h.db.PrepareContext(ctx, q)
	if err != nil {
		return fmt.Errorf("could not prepare. ChannelSetStasis. err: %v", err)
	}
	defer stmt.Close()

	// execute
	_, err = stmt.ExecContext(ctx, stasis, getCurTime(), asteriskID, id)
	if err != nil {
		return fmt.Errorf("could not execute. ChannelSetStasis. err: %v", err)
	}

	return nil
}

func (h *handler) ChannelSetState(ctx context.Context, asteriskID, id, timestamp string, state ari.ChannelState) error {

	var q string
	switch state {
	case ari.ChannelStateUp:
		q = `
		update cm_channels set
			state = ?,
			tm_update = ?,
			tm_answer = ?
		where
			asterisk_id = ?
			and id = ?
		`
	case ari.ChannelStateRing, ari.ChannelStateRinging:
		q = `
		update cm_channels set
			state = ?,
			tm_update = ?,
			tm_ringing = ?
		where
			asterisk_id = ?
			and id = ?
		`
	default:
		return fmt.Errorf("no match state. ChannelSetState. state: %s", state)
	}

	stmt, err := h.db.PrepareContext(ctx, q)
	if err != nil {
		return fmt.Errorf("could not prepare. ChannelSetState. err: %v", err)
	}
	defer stmt.Close()

	// execute
	_, err = stmt.ExecContext(ctx, string(state), timestamp, timestamp, asteriskID, id)
	if err != nil {
		return fmt.Errorf("could not execute. ChannelSetState. err: %v", err)
	}

	return nil
}

// ChannelEnd updates the channel end.
func (h *handler) ChannelEnd(ctx context.Context, asteriskID, id, timestamp string, hangup ari.ChannelCause) error {
	// prepare
	q := `
	update cm_channels set
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

// ChannelSetDataAndStasis sets the data and stasis
func (h *handler) ChannelSetDataAndStasis(ctx context.Context, asteriskID, id string, data map[string]interface{}, stasis string) error {
	//prepare
	q := `
	update cm_channels set
		data = ?,
		stasis = ?,
		tm_update = ?
	where
		asterisk_id = ?
		and id = ?
	`
	stmt, err := h.db.PrepareContext(ctx, q)
	if err != nil {
		return fmt.Errorf("could not prepare. ChannelSetDataAndStasis. err: %v", err)
	}
	defer stmt.Close()

	tmpData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("dbhandler: Could not marshal. ChannelSetDataAndStasis. err: %v", err)
	}

	// execute
	_, err = stmt.ExecContext(ctx, tmpData, stasis, getCurTime(), asteriskID, id)
	if err != nil {
		return fmt.Errorf("could not execute. ChannelSetDataAndStasis. err: %v", err)
	}

	return nil
}
